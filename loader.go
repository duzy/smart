//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
        "extbit.io/smart/ast"
        "extbit.io/smart/token"
        "extbit.io/smart/scanner"
	"bytes"
	"io/ioutil"
	"io"
        "unicode/utf8"
	"path/filepath"
	"strings"
        "plugin"
        "errors"
        "time"
        "flag"
        "fmt"
        "os/exec"
	"os"
)

const optSortErrors = false

type ResolveBits int
const (
        // If many bits are set, resolve in the listed priority.
        FromGlobe ResolveBits = 1<<iota
        FromBase
        FromProject
        FindDef
        FindRule

        FromHere

        // This is the default be
        anywhere = FromHere
        global = FromGlobe
        local = FromProject
        nonlocal = FromGlobe | FromBase | FromProject
)

type EvalBits int
const (
        KeepClosures EvalBits = 1<<iota
        KeepDelegates

        // Wants value for rule depends.
        DependValue

        // Wants v.Strval(), expands delegates and closures,
        // turn off KeepClosures, KeepDelegates.
        StringValue = 0
)

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
//
type Mode uint

const (
	ModuleClauseOnly Mode = 1 << iota // stop parsing after project or module clause
	ImportsOnly                       // stop parsing after import declarations
	ParseComments                     // parse comments and add them to AST
        Flat                              // parsing in flat mode (donot create a new module)
	Trace                             // print a trace of parsed productions
	DeclarationErrors                 // report declaration errors
	SpuriousErrors                    // same as AllErrors, for backward-compatibility
	AllErrors = SpuriousErrors        // report all errors (not just the first 10 on different lines)
        parsingDir
)

var parseMode = DeclarationErrors //|Trace

type searchlist []string

func (sl *searchlist) String() string {
        return fmt.Sprint(*sl)
}

func (sl *searchlist) Set(value string) error {
        *sl = append(*sl, strings.Split(value, ",")...)
        return nil
}

var globalPaths searchlist

func init() {
        flag.Var(&globalPaths, "search", "comma-separated list of search paths")
}

type declare struct {
        project *Project
        backproj *Project
        backscope *Scope
        useesExecuted []*Project
}

type loadinfo struct {
        absDir string // absPath = filepath.Join(absDir, baseName)
        baseName string
        specName string
        useesExecuted []*Project
        loader *Project
        scope *Scope
        declares map[string]*declare // all project declares in the loaded dir
}

func (li *loadinfo) absPath() string {
        return filepath.Join(li.absDir, li.baseName)
}

type loaderScope struct {
        cc closurecontext
        scope *Scope
}

type loader struct {
        *Context
        *parser
        tracing // Tracing/debugging
        fset     *token.FileSet
        paths    searchlist
        loads    []*loadinfo
        loaded   map[string]*Project // loaded projects
        usePath []*Project // use path
        importPath []*Project // import path
        importDepth int // import depth
        useesExecuted []*Project // all executed usees
        project  *Project // the current project
        scope    *Scope   // the current scope
        ruleParseFunc func(p *parser, tok token.Token, special ruleSpecial, targets []ast.Expr) *ast.RuleClause
        usefunc  func(l *loader, pos token.Pos, usee *Project, params []Value) error
        includeFunc func(l *loader, pos token.Pos, val Value)
        vs string // verbose prefix
}

func (l *loader) errat(pos token.Pos, err interface{}, a... interface{}) {
        if l.parser != nil {
                l.errorAt(l.parser.file.Position(pos), err, a...)
        } else {
                l.errorAt(token.Position{}, err, a...)
        }
}

func (i *loader) AddSearchPaths(paths... string) (err error) {
        for _, s := range paths {
                if s, err = filepath.Abs(s); err != nil {
                        break
                }
                if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
                        i.paths = append(i.paths, s)
                } else {
                        return errors.New(fmt.Sprintf("path '%s' is not dir", s))
                }
        }
        return nil
}

func restoreLoadingInfo(l *loader) {
        var (
                last = len(l.loads)-1
                linfo = l.loads[last]
        )

        l.loads = l.loads[0:last]
        l.useesExecuted = linfo.useesExecuted
        l.project = linfo.loader
        l.scope = linfo.scope //l.SetScope(linfo.scope)

        /*var names []string
        for _, declare := range linfo.declares {
                names = append(names, declare.project.Name())
        }

        if loader := linfo.loader; loader != nil {
                fmt.Fprintf(stderr, "exit: %v from '%s' -> %v\n", names, loader.Name(), linfo.scope)
        } else {
                fmt.Fprintf(stderr, "exit: %v -> %v\n", names, linfo.scope)
        } */
}

func saveLoadingInfo(l *loader, specName, absDir, baseName string) *loader {
        l.loads = append(l.loads, &loadinfo{
                absDir: absDir,
                baseName: baseName,
                specName: filepath.Clean(specName),
                useesExecuted: l.useesExecuted,
                loader:   l.project,
                scope:    l.scope, //Scope(),
                declares: make(map[string]*declare),
        })
        return l
}

func (l *loader) searchSpecPath(linfo *loadinfo, specName string) (absPath string, isDir bool, err error) {
        var fi os.FileInfo
        if specName == "." {
                err = fmt.Errorf("Not possible to chain itself.")
        } else if abs := filepath.IsAbs(specName); abs ||
                specName == "~" || specName == ".." ||
                strings.HasPrefix(specName, "~/") ||
                strings.HasPrefix(specName, "./") ||
                strings.HasPrefix(specName, "../") ||
                strings.HasPrefix(specName, "~\\") ||
                strings.HasPrefix(specName, ".\\") ||
                strings.HasPrefix(specName, "..\\") {
                var (
                        s = specName
                        sx string
                )
                if !abs && linfo.absDir != "" {
                        sx = filepath.Join(linfo.absDir, s)
                        if a, e := filepath.Abs(sx); e == nil {
                                s = a
                        } else {
                                err = e
                                return
                        }
                }
                if fi, err = os.Stat(s); err != nil {
                        sx = s + ".smart"
                        if fi, er := os.Stat(sx); fi != nil {
                                isDir, absPath, err = fi.IsDir(), sx, er
                                return
                        }
                        sx = s + ".sm"
                        if fi, er := os.Stat(sx); fi != nil {
                                isDir, absPath, err = fi.IsDir(), sx, er
                                return
                        }
                } else {
                        isDir, absPath = fi.IsDir(), s
                }
        } else {
                for _, base := range l.paths {
                        var s string
                        if filepath.IsAbs(base) {
                                s = filepath.Join(base, specName)
                        } else {
                                s = filepath.Join(l.workdir, base, specName)
                        }
                        if fi, err = os.Stat(s); err == nil && fi != nil {
                                isDir, absPath = fi.IsDir(), s
                                return
                        }
                }
        }
        return
}

type importoptions struct {
        unuse bool
        reuse bool
}

func (l *loader) parseImportProps(props []ast.Expr) (specName string, opts importoptions, params []Value, err error) {
        if specName, err = l.expr(props[0]).Strval(); err != nil {
                l.parser.error(props[0].Pos(), "%s", err)
                return
        } else if specName == "" {
                l.parser.error(props[0].Pos(), "empty import name")
                return
        }

        // Supported parameter forms:
        //      -param
        //      -param(value)
        //      -param=value
        var useList []Value // TODO: apply useList
        for _, prop := range props[1:] {
                var s string
                switch v := l.expr(prop); t := v.(type) {
                case *Flag:
                        if s, err = t.Name.Strval(); err != nil {
                                l.parser.error(prop.Pos(), "invalid flag `%v` (%v)", v, err)
                                return
                        }
                        switch s {
                        case "nouse", "unuse": opts.unuse = true
                        case "reuse": opts.reuse = true
                        default: params = append(params, v)
                        }
                case *Pair: // -param=value
                        switch tt := t.Key.(type) {
                        case *Flag:
                                if s, err = tt.Name.Strval(); err != nil {
                                        l.parser.error(prop.Pos(), "invalid flag name `%v` (%v)", tt.Name, err)
                                        return
                                }
                                switch s {
                                case "use": useList = append(useList, t.Value)
                                default: params = append(params, v)
                                }
                        default:
                                l.parser.error(prop.Pos(), "parameter `%v' unsupported `%T`", v, v)
                                return
                        }
                case *Argumented: // -param(value)
                        switch tt := t.Val.(type) {
                        case *Flag:
                                if s, err = tt.Name.Strval(); err != nil {
                                        l.parser.error(prop.Pos(), "invalid flag name `%v` (%v)", tt.Name, err)
                                        return
                                }
                                switch s {
                                case "use": useList = append(useList, t.Args...)
                                default: params = append(params, v)
                                }
                        default:
                                l.parser.error(prop.Pos(), "parameter `%v' unsupported `%T`", v, v)
                                return
                        }
                default:
                        l.parser.error(prop.Pos(), "parameter `%v` unsupported `%T`", v, v)
                        return
                }
        }
        return
}

func (l *loader) loadImportSpec(spec *ast.ImportSpec) {
        var (
                linfo = l.loads[len(l.loads)-1]
                specName string
                params []Value
                opts importoptions
                err error
        )
        if 0 < len(spec.Props) {
                specName, opts, params, err = l.parseImportProps(spec.Props)
                if err != nil { return }
        }
        if specName == "" {
                l.parser.error(spec.Pos(), "invalid spec `%v`", spec)
                return
        }

        var ( absPath string; isDir bool )
        if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
                l.parser.error(spec.Pos(), "no such package `%v`", specName)
                return
        } else if absPath == "" {
                l.parser.error(spec.Pos(), "missing `%s` (in %v)", specName, l.paths)
                return
        }

        if false /* UNUSED */ {
                defer func(a []*Project) { l.importPath = a } (l.importPath)
                l.importPath = append(l.importPath, l.project) // build the import path
        }

        var loaded, yes = l.loaded[absPath]

        defer func(n int) { l.importDepth = n } (l.importDepth)
        l.importDepth += 1 // increase depth for verbose imports

        // https://unicode-table.com/en/sets/arrows-symbols/
        // ┌────────────────────────────────┐
        // ├────────────────────────────────┼───┬──⇢·
        // ├──────────────────────┬────→┬←──┤   │    ⇡
        // ├┬─→───────────────────┼─────┴───┘   ├────┼⇢
        // │├┬───→             ↑  └──┬──┐       │    ⇣
        // ││└──→    ·         │     │  ├─⇥     ↓
        // │└──→───⇥─┴─⇤────┬──┴──┬──┘  │
        // └──→           ⇠─┘     ↓     └─→ ⇒ …
        if optionVerboseImport {
                //for i := 0; i < l.importDepth; i += 1 {
                //        if i == 0 { vs = "" } else { vs += "│" }
                //}
                if l.importDepth > 1 {
                        defer func(s string) { l.vs = s } (l.vs)
                        l.vs += "│"
                }
                if opts.reuse {
                        fmt.Fprintf(stderr, "%s├┬→\"%s\" (reuse, %s)\n", l.vs, specName, absPath)
                } else {
                        fmt.Fprintf(stderr, "%s├┬→\"%s\" (%s)\n", l.vs, specName, absPath)
                }
                defer func(t time.Time) {
                        var name string
                        if loaded != nil { name = loaded.name }
                        var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s
                        var ds = fmt.Sprintf("(%s)", d)
                        if d>=1*time.Second { ds = fmt.Sprintf("▶%s◀",ds) }
                        fmt.Fprintf(stderr, "%s├┘·\"%s\" ⇢ %s %s\n", l.vs, specName, name, ds)
                } (time.Now())
        }

        if yes && !opts.reuse {
                var ( proj *Project ; res, isb bool )
                if proj, res, isb, err = l.project.hasImported(loaded); err != nil {
                        l.parser.error(spec.Pos(), "`%s`: %s", specName, err)
                        return
                } else if isb {
                        l.parser.error(spec.Pos(), "`%s` is a base (%s)", specName, proj.name)
                        return
                } else if res {
                        l.parser.error(spec.Pos(), "'%s' already imported by '%s'", specName, proj.name)
                        return
                }
        }

        if isDir {
                err = l.loadDir(specName, absPath, nil)
        } else {
                err = l.load(specName, absPath, nil)
        }
        if err != nil {
                if _, ok := err.(scanner.Errors); ok {
                        l.parser.error(spec.Pos(), "import `%v` failed (%v)", specName, absPath)
                } else {
                        l.parser.error(spec.Pos(), "import `%v` (%v): %v", specName, absPath, err)
                }
                return
        }

        if loaded != nil {
                // already loaded previously
        } else if loaded, yes = l.loaded[absPath]; yes {
                // successfully loaded (first)
        } else {
                l.parser.error(spec.Pos(), "'%s' not loaded (%s)", specName, absPath)
        }

        if enable_assertions {
                assert(loaded != nil, "'%s' loaded nil", specName)
        }

        // The project import list is different from using list.
        l.project.imports = append(l.project.imports, loaded)

        for _, u := range l.project.usings.list {
                var ( proj *Project ; res, isb bool )
                if proj, res, isb, err = loaded.hasImported(u.project); err != nil {
                        l.parser.error(spec.Pos(), "`%s`: %s", specName, err)
                        return
                } else if isb {
                        if l.project.hasBase(u.project) {
                                // common bases are fine
                        } else {
                                l.parser.warn(spec.Pos(), "`%s` has base `%s` (%s)", loaded, u.project, proj)
                        }
                } else if res && !u.project.allowMultiImported {
                        l.parser.warn(spec.Pos(), "`%s` has already imported `%s` (from %s)", loaded, u.project, proj)
                }
        }

        if !opts.unuse {
                name, _ := l.project.scope.Lookup(loaded.name).(*ProjectName)
                if name == nil {
                        l.parser.error(spec.Pos(), "%v (%v,dir=%v) not in %v", specName, absPath, isDir, l.project.scope.comment)
                } else {
                        err = l.useProject(spec.Props[0].Pos(), loaded, params)
                }
        }       
        return
}

func (l *loader) loadPlugin() (err error) {
        g := stat("smart.go", "", l.project.absPath)
        if g == nil { return /* smart.go was not presented */ }

        var src string
        if src, err = g.Strval(); err != nil { return }

        s := strings.Replace(l.project.relPath, "..", "_", -1)
        s = filepath.Join(filepath.Dir(joinTmpPath("", "")), "plugins", s)

        var build = true

        so := stat(/*l.project.name*/"plugin", "", s, nil)
        if s, err = so.Strval(); err != nil { return }
        if so.exists() && !optionAlwaysBuildPlugins {
                if so.info.ModTime().After(g.info.ModTime()) {
                        build = false // Plugin already updated.
                }
        }
        if build {
                fmt.Fprintf(stderr, "smart: Build %v …", src)
                o := &bytes.Buffer{}
                c := exec.Command("go", "build", "-buildmode=plugin", "-o", s, src)
                c.Stdout, c.Stderr = o, o
                if err = c.Run(); err == nil {
                        fmt.Fprintf(stderr, "… ok\n")
                } else {
                        fmt.Fprintf(stderr, "… error\n")
                        fmt.Fprintf(stderr, "%s", o)
                }
        }

        if err != nil { return }

        // Once plugin is opened, there's no need/way to close it.
        if l.project.plugin, err = plugin.Open(s); err == nil {
                var p plugin.Symbol
                if p, err = l.project.plugin.Lookup("Init"); err != nil {
                        return
                } else if p == nil {
                        // no initialization (optional)
                } else if f, ok := p.(func(*Project) (*Scope, error)); ok {
                        l.project.pluginScope, err = f(l.project)
                }
        } else {
                // FIXME: err == "plugin was built with a different version"
        }
        return
}

func (l *loader) exprEvaluated(x *ast.EvaluatedExpr) (v Value) {
        var ok bool
        if x.Data == nil {
                l.parser.error(x.Pos(), "evaluated data is nil `%T`", x.Expr)
        } else if v, ok = x.Data.(Value); !ok {
                l.parser.error(x.Pos(), "evaluated data is not value `%T`", x.Data)
        }
        return v
}

func (l *loader) exprArgumented(x *ast.ArgumentedExpr) Value {
        return &Argumented{
                Val: l.expr(x.X),
                Args: l.exprs(x.Arguments),
        }
}

func (l *loader) exprClosureDelegate(x *ast.ClosureDelegate) (name Value, obj Object, args []Value) {
        if name = l.expr(x.Name); name == nil {
                l.parser.error(x.Name.Pos(), "invalid name `%T`", x.Name)
                return
        }

        var tok = token.ILLEGAL
        switch x.TokLp {
        case token.LPAREN, token.LBRACE, token.COLON:
                tok = x.TokLp
        case token.STRING, token.COMPOUND:
                if x.Tok == token.DELEGATE {
                        l.parser.error(x.TokPos, "unsupported delegate (%v).", x.TokLp)
                        return
                } else {
                        tok = x.TokLp
                }
        case token.ILLEGAL:
                if x.Tok.IsClosure() || x.Tok.IsDelegate() {
                        tok = token.LPAREN
                } else {
                        l.parser.error(x.TokPos, "unregonized closure/delegate (%v).", x.Tok)
                        return
                }
        default:
                if x.Tok == token.DELEGATE {
                        l.parser.error(x.TokPos, "unregonized delegate (%v).", x.TokLp)
                } else {
                        l.parser.error(x.TokPos, "unregonized closure (%v).", x.TokLp)
                }
                return
        }

        var ( resolved Object; err error )
        if x.Resolved != nil {
                resolved = x.Resolved.(Object)
        }

        s, err := name.Strval()
        if err != nil {
                l.parser.error(x.Name.Pos(), "%v", err)
                return
        }

        switch tok {
        case token.COLON:
                switch s {
                case "use": obj = l.project.usings;
                default:
                        l.parser.error(x.Name.Pos(), "unsupported special delegation")
                        return
                }
        case token.LPAREN:
                if resolved == nil { // if not resolved at parse time
                        if resolved, err = l.resolve(name); err != nil {
                                l.parser.error(x.Name.Pos(), "%s", err)
                                return
                        }
                }
                if resolved != nil {
                        if def, _ := resolved.(Caller); def == nil {
                                l.parser.error(x.Name.Pos(), "uncallable `%s` resolved `%T`", name, resolved)
                                return
                        } else if obj = def.(Object); obj == nil {
                                l.parser.error(x.Name.Pos(), "non-object callable `%s` resolved `%T`", name, def)
                                return
                        }
                }
        case token.LBRACE:
                if resolved == nil { // if not resolved at parse time
                        if resolved, err = l.find(name); err != nil {
                                l.parser.error(x.Name.Pos(), "%s", err)
                                return
                        } else if resolved == nil {
                                //l.parser.error(x.Name.Pos(), "entry `%s` is nil", name)
                                return
                        } 
                }
                if exe, _ := resolved.(Executer); exe != nil {
                        if obj = exe.(Object); obj == nil {
                                l.parser.error(x.Name.Pos(), "non-object executer `%s` resolved `%T`", name, resolved)
                                return
                        }
                } else {
                        l.parser.error(x.Name.Pos(), "unexecutable `%s` resolved `%T`", name, resolved)
                        return
                }
        case token.STRING, token.COMPOUND:
                if resolved == nil { // if not resolved at parse time
                        if resolved, err = l.find(name); err != nil {
                                l.parser.error(x.Name.Pos(), "%s", err)
                                return
                        } else if resolved == nil {
                                //resolved = unresolved(l.project, name)
                        }
                }
                obj = resolved
        }
        if obj == nil && l.project.plugin != nil {
                if l.project.pluginScope != nil {
                        obj = l.project.pluginScope.Lookup(s)
                }
        }
        for i, x := range x.Args {
                if a := l.expr(x); a != nil {
                        args = append(args, a)
                } else {
                        l.parser.error(x.Pos(), "nil arg #%d `%T`", i, x)
                        return
                }
        }
        return
}

func (l *loader) exprClosure(x *ast.ClosureExpr) (v Value) {
        if name, obj, args := l.exprClosureDelegate(&x.ClosureDelegate); name == nil {
                l.parser.error(x.Name.Pos(), "invalid closure name `%T`", x.Name)
        } else if obj != nil {
                v = MakeClosure(Position(x.Position), x.TokLp, obj, args...)
        } else if true {
                obj = unresolved(l.project, name)
                v = MakeClosure(Position(x.Position), x.TokLp, obj, args...)
        } else {
                l.parser.error(x.Pos(), "closure nil object (name `%v`, `%v`)", name, l.scope.comment)
        }
        return
}

func (l *loader) exprDelegate(x *ast.DelegateExpr) (v Value) {
        if name, obj, args := l.exprClosureDelegate(&x.ClosureDelegate); name == nil {
                l.parser.error(x.Name.Pos(), "invalid delegate name `%T`", x.Name)
        } else if obj != nil {
                v = MakeDelegate(Position(x.Position), x.TokLp, obj, args...)
        } else if sel, ok := name.(*selection); ok {
                if o, err := sel.object(); err == nil && o.DeclScope().comment == usecomment {
                        obj = unresolved(l.project, name)
                        v = MakeDelegate(Position(x.Position), x.TokLp, obj, args...)
                } else if err != nil {
                        l.parser.error(x.Name.Pos(), "delegate selection `%v`: %v", name, err)
                } else if o == nil {
                        l.parser.error(x.Name.Pos(), "delegate selection `%v`: nil object", name)
                } else if v, err = sel.value(); err != nil {
                        l.parser.error(x.Name.Pos(), "delegate selection `%v`: %v", name, err)
                } else if v == nil {
                        l.parser.error(x.Name.Pos(), "delegate selection `%v`: nil value", name)
                }
        } else {
                l.parser.error(x.Name.Pos(), "delegate expression `%v`: nil object (in %v)", name, l.scope.comment)
        }
        return
}

func (l *loader) exprSelection(x *ast.SelectionExpr) (v Value) {
        if lhs := l.expr(x.Lhs); lhs != nil {
                if lhs.Type() != SelectionType {
                        // Resolve the first left-hand-side.
                        if o, err := l.resolve(lhs); err != nil {
                                l.parser.error(x.Lhs.Pos(), "selection expression `%v`: %v", lhs, err)
                        } else if o == nil {
                                l.parser.error(x.Lhs.Pos(), "`%v` is undefined", lhs)
                        } else {
                                lhs = o
                        }
                }
                if rhs := l.expr(x.Rhs); rhs != nil {
                        v = &selection{ x.Tok, lhs, rhs }
                } else {
                        l.parser.error(x.Rhs.Pos(), "invalid %v `%T`", lhs, x.Rhs)
                }
        } else {
                l.parser.error(x.Lhs.Pos(), "invalid `%T`", x.Lhs)
        }
        return
}

func (l *loader) exprBasicLit(x *ast.BasicLit) (v Value) {
        switch x.Kind {
        case token.BAR:      v = modifierbar
        case token.BIN:      v = ParseBin(x.Value)
        case token.OCT:      v = ParseOct(x.Value)
        case token.INT:      v = ParseInt(x.Value)
        case token.HEX:      v = ParseHex(x.Value)
        case token.FLOAT:    v = ParseFloat(x.Value)
        case token.DATETIME: v = ParseDateTime(x.Value)
        case token.DATE:     v = ParseDate(x.Value)
        case token.TIME:     v = ParseTime(x.Value)
        case token.URI:      v = ParseURL(x.Value)
        case token.BAREWORD: v = &Bareword{x.Value}
        case token.STRING:   v = &String{x.Value}
        case token.ESCAPE:   v = &String{EscapeChar(x.Value)}
        case token.RAW:      v = &Raw{x.Value}
        default: unreachable()
        }
        return
}

func (l *loader) exprBareword(x *ast.Bareword) (res Value) {
        res = &Bareword{x.Value}
        return
}

func (l *loader) exprConstant(x *ast.Constant) (res Value) {
        switch x.Tok {
        case token.TRUE:  res = &boolean{ true }
        case token.FALSE: res = &boolean{ false }
        case token.YES:   res = &answer{ true }
        case token.NO:    res = &answer{ false }
        }
        return
}

func (l *loader) exprBarecomp(x *ast.Barecomp) (res Value) {
        res = MakeBarecomp(l.exprs(x.Elems)...)
        return
}

func (l *loader) exprBarefile(x *ast.Barefile) (v Value) {
        if file, _ := x.File.(*File); file != nil {
                if x.Val != nil {
                        v = &Barefile{x.Val.(Value), file}
                } else {
                        v = &Barefile{l.expr(x.Name), file}
                }
        }
        if v == nil {
                l.parser.error(x.Pos(), "invalid barefile `%s` (%T)", x.Name, x.File)
        }
        return
}

func (l *loader) exprPath(x *ast.PathExpr) (res Value) {
        res = MakePath(l.exprs(x.Segments)...)
        return
}

func (l *loader) exprPathSeg(x *ast.PathSegExpr) (v Value) {
        switch x.Tok {
        case token.PCON:   v = MakePathSeg('/')
        case token.TILDE:  v = MakePathSeg('~')
        case token.PERIOD: v = MakePathSeg('.')
        case token.DOTDOT: v = MakePathSeg('^') // 
        default: l.parser.error(x.Pos(), "unsupported path segment `%v`", x.Tok)
        }
        return
}

func (l *loader) exprFlag(x *ast.FlagExpr) (v Value) {
        if x.Name == nil {
                v = &Flag{ universalnone }
        } else {
                v = &Flag{ l.expr(x.Name) }
        }
        return
}

func (l *loader) exprNeg(x *ast.NegExpr) (v Value) {
        v = Negative(l.expr(x.Val))
        return
}

func (l *loader) exprCompoundLit(x *ast.CompoundLit) (v Value) {
        v = MakeCompound(l.exprs(x.Elems)...)
        return
}

func (l *loader) exprGroup(x *ast.GroupExpr) (v Value) {
        v = MakeGroup(l.exprs(x.Elems)...)
        return
}

func (l *loader) exprList(x *ast.ListExpr) (v Value) {
        v = MakeList(l.exprs(x.Elems)...)
        return
}

func (l *loader) exprKeyValue(x *ast.KeyValueExpr) (res Value) {
        if k := l.expr(x.Key); l.parser.bits&specialKeyValue != 0 {
                res = &Pair{k, l.expr(x.Value)}
        } else if k.Type().Bits()&IsKeyName != 0 {
                res = MakePair(k, l.expr(x.Value))
        } else {
                l.parser.error(x.Key.Pos(), "not valid key `%T`", k)
        }
        return
}

func (l *loader) exprPerc(x *ast.PercExpr) (v Value) {
        v = MakePercPattern(l.expr(x.X), l.expr(x.Y))
        return
}

func (l *loader) exprGlob(x *ast.GlobExpr) (v Value) {
        v = MakeGlobPattern(l.exprs(x.Components)...)
        return
}

func (l *loader) exprGlobMeta(x *ast.GlobMeta) (v Value) {
        v = MakeGlobMeta(x.Tok)
        return
}

func (l *loader) exprGlobRange(x *ast.GlobRange) (v Value) {
        v = MakeGlobRange(l.expr(x.Chars))
        return
}

func (l *loader) exprRecipe(x *ast.RecipeExpr) (v Value) {
        if len(x.Elems) == 0 {
                v = universalnone
        } else if x.Dialect == "" || x.Dialect == "eval" {
                v = MakeList(l.exprs(x.Elems)...)
        } else {
                v = MakeCompound(l.exprs(x.Elems)...)
        }
        return
}

func (l *loader) exprRecipeDefineClause(x *ast.RecipeDefineClause) (v Value) {
        return &undetermined{ x.Tok, l.expr(x.Name), l.expr(x.Value) }
}

func (l *loader) exprIncludeRuleClause(x *ast.IncludeRuleClause) (v Value) {
        entries := l.rule(x.RuleClause, ruleSpecialNor)
        if n := len(entries); n == 1 {
                v = entries[0]
        } else if n > 1 {
                l.parser.error(x.Pos(), "including multiple target `%v`", x)
        } else {
                l.parser.error(x.Pos(), "invalid rule `%v`", x)
        }
        return
}

func (l *loader) expr(expr ast.Expr) (v Value) {
        if expr == nil {
                //l.parser.error(l.parser.pos, "encountered nil expr")
                v = universalnone
                return
        }

        switch x := expr.(type) {
        case *ast.EvaluatedExpr:
                v = l.exprEvaluated(x)
        case *ast.ArgumentedExpr:
                v = l.exprArgumented(x)
        case *ast.ClosureExpr:
                v = l.exprClosure(x)
        case *ast.DelegateExpr:
                v = l.exprDelegate(x)
        case *ast.SelectionExpr:
                v = l.exprSelection(x)
        case *ast.BasicLit:
                v = l.exprBasicLit(x)
        case *ast.Bareword:
                v = l.exprBareword(x)
        case *ast.Constant:
                v = l.exprConstant(x)
        case *ast.Barecomp:
                v = l.exprBarecomp(x)
        case *ast.Barefile:
                v = l.exprBarefile(x)
        case *ast.PathExpr:
                v = l.exprPath(x)
        case *ast.PathSegExpr:
                v = l.exprPathSeg(x)
        case *ast.FlagExpr:
                v = l.exprFlag(x)
        case *ast.NegExpr:
                v = l.exprNeg(x)
        case *ast.CompoundLit:
                v = l.exprCompoundLit(x)
        case *ast.GroupExpr:
                v = l.exprGroup(x)
        case *ast.ListExpr:
                v = l.exprList(x)
        case *ast.KeyValueExpr:
                v = l.exprKeyValue(x)
        case *ast.PercExpr:
                v = l.exprPerc(x)
        case *ast.GlobExpr:
                v = l.exprGlob(x)
        case *ast.GlobMeta: // "*", "?"
                v = l.exprGlobMeta(x)
        case *ast.GlobRange: // "[a-z]", "[abc]", `[a\-b]`, `[a\]b]`
                v = l.exprGlobRange(x)
        case *ast.RecipeExpr:
                v = l.exprRecipe(x)
        case *ast.RecipeDefineClause:
                v = l.exprRecipeDefineClause(x)
        case *ast.IncludeRuleClause:
                v = l.exprIncludeRuleClause(x)
        case *ast.BadExpr:
                l.parser.error(x.Pos(), "bad expr")
                return
        }

        if v == nil {
                l.parser.error(expr.Pos(), "`%v` nil expression (%T)", expr, expr)
                v = new(Nil)
        }
        return
}

func (l *loader) exprs(exprs []ast.Expr) (values []Value) {
        for _, x := range exprs {
                values = append(values, l.expr(x))
        }
        return
}

func (l *loader) useProject(pos token.Pos, usee *Project, params []Value) (err error) {
        if l.usefunc == nil {
                l.parser.error(pos, "`%v` use clause forbiden", usee.name)
        } else if err = l.usefunc(l, pos, usee, params); err != nil {
                if p, ok := err.(*scanner.Error); ok {
                        l.parser.error(pos, "%v", p.Err)
                } else {
                        l.parser.error(pos, "%v", err)
                }
        }
        return
}

func (l *loader) isExecutedUsee(usee *Project) (res bool) {
        for _, p := range l.useesExecuted {
                if res = usee == p; res { break }
        }
        return
}

func (l *loader) executeUseRule(pos token.Pos, userule *RuleEntry, params []Value) (err error) {
        position := l.parser.file.Position(pos)
        for _, prog := range userule.programs {
                defer prog.setUser(prog.setUser(l.project))
        }
        results, err := mergeresult(userule.Execute(Position(position), params...))
        if err != nil { return }

        for _, elem := range results {
                switch t := elem.(type) {
                case *None: // does nothing
                case *undetermined:
                        /*if def := l.determine(pos, t.tok, t.identifier, t.value); def != nil {
                                // ...
                        }*/
                        if sel, ok := t.identifier.(*selection); ok && sel != nil {
                                var ( val Value; def *Def )
                                if val, err = sel.value(); err != nil { return }
                                if def, ok = val.(*Def); !ok && def == nil {
                                        err = scanner.Errorf(position, "`%v` not a def", t.identifier)
                                        return
                                }

                                var alt Object
                                
                                // FIXES: 'user->xxx' is selecting 'xxx'
                                // from the base if the current project has
                                // no 'xxx' defined. So we have to fix it by
                                // having it defined in the current project.
                                if def.OwnerProject() != l.project {
                                        if def, alt = l.def(def.name); alt != nil && t.tok != token.QUE_ASSIGN {
                                                err = scanner.Errorf(position, "`%s` already defined", alt.Name())
                                                return
                                        }
                                }

                                if err = l.assign(pos, t.tok, def, alt, t.value); err != nil {
                                        return
                                }
                        } else {
                                err = scanner.Errorf(position, "unsupported using def `%v` (%T)", t.identifier, t.identifier)
                                return
                        }
                default:
                        err = scanner.Errorf(position, "todo: using def `%T`", elem)
                        return
                }
        }
        return
}

func (l *loader) executeUseRulesRecursively(pos token.Pos, usee *Project, params []Value) (err error) {
        // Monitor the execution time of the :use: rule.
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        if optionBenchImport && d > 1*time.Millisecond {
                                var s = l.usePathStr()
                                fmt.Fprintf(stderr, "%s││▶%s:use(%s).execute() … (%s) (%s)◀\n", l.vs, l.project.name, usee.name, d, s)
                        }
                } else if optionBenchSlow && d > 500*time.Millisecond { // ⌚ ⌛
                        fmt.Fprintf(stderr, "smart: %s: slow ▶use(%s).execute()◀ … (%s)\n", l.project.name, usee.name, d)
                }
        } (time.Now())

        // Execute use rules recursively in the second defer.
        defer func() {
                if true {
                        for _, u := range usee.usings.list {
                                err = l.executeUseRulesRecursively(pos, u.project, params)
                                if err != nil { break }
                        }
                } else {
                        for _, u := range usee.usees() {
                                err = l.executeUseRulesRecursively(pos, u, params)
                                if err != nil { break }
                        }
                }
        } ()

        if usee.userule != nil {
                err = l.executeUseRule(pos, usee.userule, params)
        }
        return
}

func (l *loader) executeUseRuleDirectly(pos token.Pos, usee *Project, params []Value) (err error) {
        // Return immediately if this usee is executed. If the use rule
        // accumulates values (e.g. += or =+), it may take very long time.
        if !optionExecuteUseRuleMultiTimes && l.isExecutedUsee(usee) {
                return // execute use rule only once
        }

        // Monitor the execution time of the :use: rule.
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        if optionBenchImport && d > 1*time.Millisecond {
                                var s = l.usePathStr()
                                fmt.Fprintf(stderr, "%s││▶%s:use(%s).execute() … (%s) (%s)◀\n", l.vs, l.project.name, usee.name, d, s)
                        }
                } else if optionBenchSlow && d > 500*time.Millisecond { // ⌚ ⌛
                        fmt.Fprintf(stderr, "smart: %s: slow ▶use(%s).execute()◀ … (%s)\n", l.project.name, usee.name, d)
                }
        } (time.Now())

        if usee.userule != nil {
                err = l.executeUseRule(pos, usee.userule, params)
                l.useesExecuted = append(l.useesExecuted, usee)
        }
        return
}

func (l *loader) usePathStr() (s string) {
        for i, u := range l.usePath {
                if i > 0 { s += "," }
                s += u.name
        }
        return
}

func useProject(l *loader, pos token.Pos, usee *Project, params []Value) (err error) {
        if usee == l.project {
                position := l.parser.file.Position(pos)
                err = scanner.Errorf(position, "'%v' use loop (%s)", usee.name, l.usePathStr())
                return
        } else if false {
                for _, using := range l.project.usings.list {
                        if using.project == usee { return }
                }
        } else if optionRecursiveUsing && l.project.isUsingProject(usee) {
                return
        } else if l.project.isUsingDirectly(usee) {
                return
        }

        // clocks:🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        if optionBenchImport && d > 1*time.Millisecond {
                                var s = l.usePathStr()
                                fmt.Fprintf(stderr, "%s││▶%s:use(%s) … (%s) (%s)◀\n", l.vs, l.project.name, usee.name, d, s)
                        }
                } else if optionBenchSlow && d > 500*time.Millisecond { // ⌚ ⌛
                        fmt.Fprintf(stderr, "smart: %s: slow ▶use(%s)◀ … (%s)\n", l.project.name, usee.name, d)
                }
        } (time.Now())

        defer func(a []*Project) { l.usePath = a } (l.usePath)
        l.usePath = append(l.usePath, usee) // build the use path

        // Also use the bases and usee's using list, so that all
        // dependencies are included.
        for _, base := range usee.bases {
                //if isUsingProject(l.project, base) { continue }
                if err = useProject(l, pos, base, params); err != nil { return }
        }
        if optionRecursiveUsing {
                // Recursive using projects takes very much longer time
                // (especially to big projects), so be careful of enabling
                // it.
                for _, using := range usee.usings.list {
                        //if isUsingProject(l.project, using.project) { continue }
                        if err = useProject(l, pos, using.project, params); err != nil { return }
                }
        }

        // Add to the project using list, so that the use path is correct.
        l.project.usings.append(usee, params)

        // Execute the :use: rule if presented to apply the conditions
        // of using the project.
        if optionRecursiveUsing {
                // No need to recursively execute use rules if useProject
                // is already called recursively.
                err = l.executeUseRuleDirectly(pos, usee, params)
        } else if optionExecuteUseRulesRecursively {
                err = l.executeUseRulesRecursively(pos, usee, params)
        } else {
                // Get usees 'recursively' and use each directly.
                for _, u := range usee.usees() {
                        err = l.executeUseRuleDirectly(pos, u, params)
                        if err != nil { break }
                }
                // Execute the usee rule at the end. 
                if err == nil {
                        err = l.executeUseRuleDirectly(pos, usee, params)
                }
        }
        return
}

func (l *loader) determine(pos token.Pos, tok token.Token, identifier, value Value) (def *Def) {
        var alt, derived Object
        switch t := identifier.(type) {
        case *selection:
                if v, err := t.value(); err != nil {
                        l.parser.error(pos, "determine `%v`: %v", t, err)
                        return
                } else if d, ok := v.(*Def); ok {
                        def = d
                } else {
                        l.parser.error(pos, "`%v` is not a def (%T)", t, v)
                        return
                }
        case *Bareword, *Barecomp:
                if name, err := t.Strval(); err != nil {
                        l.parser.error(pos, "determine `%v`: %v", t, err)
                        return
                } else if _, ok := builtins[name]; ok {
                        l.parser.error(pos, "`%v` (%v) is builtin name", identifier, name)
                        return
                } else {
                        _, derived = l.scope.Find(name) // value to derive
                        if def, alt = l.def(name); alt != nil {
                                def = alt.(*Def)
                        }
                }
        }

        if alt != nil && (tok == token.ASSIGN || tok == token.EXC_ASSIGN) {
                if alt.OwnerProject() == l.project {
                        l.parser.error(pos, "`%v` already defined (%T)", identifier, alt)
                        return
                } else if def, alt = l.def(alt.Name()); alt != nil {
                        l.parser.error(pos, "`%v` already defined (%T)", identifier, alt)
                        return
                }
        }

        if derived != nil && derived != def && tok == token.ADD_ASSIGN {
                // If it's the first time this name was determined.
                if d, _ := derived.(*Def); d != nil && !def.Value.refs(d) {
                        // Unshift the delegation to derive value.
                        position := Position(l.parser.file.Position(pos))
                        if err := def.append(MakeDelegate(position, token.LPAREN, d)); err != nil {
                                l.parser.error(pos, "%v", err)
                                return
                        }
                }
        }

        if err := l.assign(pos, tok, def, alt, value); err != nil {
                l.parser.error(pos, "%v", err)
        }
        return
}

func (l *loader) use(spec *ast.UseSpec) {
        if l.project.keyword == token.PACKAGE {
                l.parser.error(spec.Pos(), "forbiden package `use`")
        } else if len(spec.Props) == 0 {
                l.parser.error(spec.Pos(), "empty `use` spec")
        } else if name := l.expr(spec.Props[0]); name == nil {
                l.parser.error(spec.Pos(), "undefined `use` target")
        } else if name == universalnone {
                l.parser.error(spec.Pos(), "none `use` target")
        } else {
                var usee Object
                switch t := name.(type) {
                case *Bareword, *Barecomp, *String, *Compound:
                        if str, err := name.Strval(); err != nil {
                                l.parser.error(spec.Props[0].Pos(), "%s", err)
                                return
                        } else {
                                _, usee = l.project.scope.Find(str)
                        }
                case *Flag:
                        l.useOptional(spec.Props[0].Pos(), t.Name)
                        return
                default:
                        l.parser.error(spec.Pos(), "`%v` invalid usee (%T)", name, name)
                        return
                }
                if usee == nil {
                        l.parser.error(spec.Pos(), "nil usee `%v`", name)
                        return
                }

                switch t := usee.(type) {
                case *ProjectName:
                        l.useProject(spec.Props[0].Pos(), t.project, l.exprs(spec.Props[1:]))
                case *Def:
                        if alt := l.project.scope.Insert(t); alt != nil {
                                l.parser.error(spec.Pos(), "`%s` already defined in %s", t.Name(), l.project.scope.comment)
                        }
                case *RuleEntry:
                        if alt := l.project.scope.Insert(t); alt != nil {
                                l.parser.error(spec.Pos(), "`%s` already defined in %s", t.Name(), l.project.scope.comment)
                        }
                default:
                        l.parser.error(spec.Pos(), "unknown usee `%v` (%T)", t, t)
                }
        }
}

func (l *loader) useOptional(pos token.Pos, opt Value) {
        s, err := opt.Strval()
        if err != nil {
                l.parser.error(pos, "`%v` invalid value (%s)", opt, err)
                return
        }
        switch s {
        case "recursively": l.useRecursively(pos)
        default:
                l.parser.error(pos, "`%s` unknown use option (%v)", s, opt)
                return
        }
}


func (l *loader) useRecursively(pos token.Pos) {
        if optionNoDeprecatedFeatures {
                l.parser.error(pos, "'use -recursively' is deprecated")
                return
        }

        if l.usefunc == nil {
                l.parser.error(pos, "`%v` use is forbiden", l.project.name)
                return
        }

        var usees = l.project.usees()
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        if optionBenchImport && d > 0*time.Millisecond {
                                fmt.Fprintf(stderr, "%s││▶%s:use(-recursively) … (%s)◀\n", l.vs, l.project.name, d)
                        }
                } else if optionBenchSlow && d > 500*time.Millisecond { // ⌚ ⌛
                        fmt.Fprintf(stderr, "smart: %s: slow ▶use(-recursively)◀ … (%s)\n", l.project.name, d)
                }
        } (time.Now())

        // Recursive using projects may take very long time.
        unique := make(map[*Project]int)
        for _, usee := range usees {
                if _, ok := unique[usee]; ok { continue }
                //if l.project.isUsingDirectly(usee) { continue }
                unique[usee] += 1 // ensure it's used only once
                if err := l.usefunc(l, pos, usee, nil); err != nil {
                        break
                }
        }
        unique = nil
}

func (l *loader) configuration(spec *ast.ConfigurationSpec) (res Value) {
        // Parses name and value in current scope.
        var name = l.expr(spec.Name)
        var value = l.expr(spec.Value)
        defer func(scope *Scope) { l.scope = scope } (l.scope)
        l.scope = configuration.project.scope
        res = l.determine(spec.Pos(), spec.Tok, name, value)
        return
}

func (l *loader) evalspec(spec *ast.EvalSpec) (res Value) {
        if num := len(spec.Props); num > 0 {
                // At the point of `eval` was represented, the closure context
                // might be empty. So we start closure with the current scope.
                defer setclosure(setclosure(cloctx.unshift(l.scope)))

                var id = spec.Props[0]
                var position = Position(l.parser.file.Position(id.Pos()))
                switch op := l.expr(id).(type) {
                case Caller:
                        res, _ = op.Call(position, l.exprs(spec.Props[1:])...)
                default:
                        var ( str string; err error )
                        if str, err = op.Strval(); err != nil {
                                l.parser.error(id.Pos(), "%s: %v", op, err)
                        } else if _, obj := l.scope.Find(str); obj == nil {
                                l.parser.error(id.Pos(), "`%s` undefined", str)
                        } else if f, _ := obj.(Caller); f == nil {
                                l.parser.error(id.Pos(), "`%T` is not caller (%s)", obj, str)
                        } else if res, err = f.Call(position, l.exprs(spec.Props[1:])...); err != nil {
                                l.parser.error(id.Pos(), "%s: %v", str, err)
                        }
                }
        }
        return
}

func (l *loader) define(clause *ast.DefineClause) {
        var identifier = l.expr(clause.Name)
        var value = l.expr(clause.Value)
        l.determine(clause.Pos(), clause.Tok, identifier, value)
}

func (l *loader) rule(clause *ast.RuleClause, special ruleSpecial) (entries []*RuleEntry) {
        defer setclosure(setclosure(cloctx.unshift(l.project.scope)))

        var (
                depends []Value
                ordered []Value
                recipes []Value
                progScope *Scope
                params []string
        )
        for _, depend := range clause.Depends {
                switch dep := l.expr(depend).(type) {
                case *List:
                        depends = append(depends, dep.Elems...)
                default:
                        depends = append(depends, dep)
                }
        }
        for _, depend := range clause.Ordered {
                switch dep := l.expr(depend).(type) {
                case *List:
                        ordered = append(ordered, dep.Elems...)
                default:
                        ordered = append(ordered, dep)
                }
        }

        if p, ok := clause.Program.(*ast.ProgramExpr); ok && p != nil {
                if progScope, _ = p.Scope.(*Scope); progScope == nil {
                        l.parser.error(clause.Pos(), "undefined program scope (%T).", p.Scope)
                }
                if p.Recipes != nil {
                        recipes = l.exprs(p.Recipes)
                }
                params = p.Params
        } else {
                l.parser.error(clause.Program.Pos(), "unsupported program type (%T)", clause.Program)
                return
        }
        
        var modifiers []Value
        if clause.Modifier != nil {
                modifiers = l.exprs(clause.Modifier.Elems)
                /*if n := len(modifiers); n == 0 {
                        //l.parser.error(clause.Modifier.Pos(), "empty modifier")
                } else if m := modifiers[0]; m != nil {
                        //...
                }*/
        }

        var configure = false
        var prog = &Program{
                globe:    l.globe,
                project:  l.project,
                scope:    progScope,
                params:   params,
                depends:  depends,
                ordered:  ordered,
                recipes:  recipes,
                position: Position(clause.Position),
        }
        for i, m := range modifiers {
                position := l.parser.file.Position(clause.Modifier.Elems[i].Pos())
                if p, err := prog.pipe(Position(position), m); err != nil {
                        l.parser.error(clause.Program.Pos(), "modifier `%v`: %v", m, err)
                        return
                } else if !configure {
                        if s, err := p.name.Strval(); err != nil {
                                l.parser.error(clause.Program.Pos(), "modifier `%v`: %v", m, err)
                                return
                        } else if s == "configure" {
                                configure = true
                        }
                }
        }
        for n, target := range l.exprs(clause.Targets) {
                if target == nil {
                        l.parser.error(clause.Targets[n].Pos(), "nil target (%T)", clause.Targets[n])
                        return
                }
                if true {// it should work too if not checking against files
                        switch target.(type) {
                        default:
                                if s, err := target.Strval(); err != nil {
                                        l.parser.error(clause.Targets[n].Pos(), "%v", err)
                                } else if file := l.project.matchFile(s); file != nil {
                                        target = file
                                }
                        case *File, *Path:
                        case *PercPattern:
                        }
                }
                var entry, err = l.project.entry(special, target, prog)
                if err != nil {
                        l.parser.error(clause.Targets[n].Pos(), "%v", err)
                        return
                } else /*if entry != nil*/ {
                        entry.Position = Position(l.parser.file.Position(clause.Targets[n].Pos()))
                        entries = append(entries, entry)
                }
                if configure {
                        configuration.entires = append(configuration.entires, entry)
                }
        }
        return
}

func (l *loader) include(spec *ast.IncludeSpec) {
        if l.includeFunc == nil {
                l.parser.error(spec.Pos(), "`include` is forbiden")
        } else if len(spec.Props) > 0 {
                prop := spec.Props[0]
                l.includeFunc(l, prop.Pos(), l.expr(prop))
        }
}

func includespec(l *loader, pos token.Pos, spec Value) {
        var linfo = l.loads[len(l.loads)-1]
        var specName, fullname string
        var err error

        // Execute the rule entry to update include source.
        if entry, ok := spec.(*RuleEntry); ok && entry != nil {
                var result []Value
                if result, err = entry.Execute(entry.Position); err != nil {
                        l.parser.error(pos, "include `%v`: %v", spec, err)
                        return
                } else if result != nil {
                        // result ignored
                }
                spec = entry.target
        }

        switch t := spec.(type) {
        /*case *Path:
                panic(fmt.Sprintf("include not implemented (%T)", t))*/
        case *File:
                if t.info == nil {
                        l.parser.error(pos, "`%v` no source file", t)
                        return
                }
                fullname = t.FullName() //filepath.Join(t.dir, t.Name)
                specName = t.name
        default:
                if specName, err = spec.Strval(); err != nil {
                        l.parser.error(pos, "include `%v`: %v", spec, err)
                        return
                }
                if filepath.IsAbs(specName) {
                        fullname = specName
                } else {
                        fullname = filepath.Join(linfo.absDir, specName)
                }
        }

        if specName == "" {
                l.parser.error(pos, "`%v` is empty string", spec)
                return
        }

        var mode = l.mode
        var absDir, baseName = filepath.Split(fullname)
        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))
        if _, err = l.ParseFile(fullname, nil, parseMode|Flat); err != nil {
                l.parser.error(pos, "include: %v", err)
        } else {
                // The parse mode could still be 'Flat' here as ParseFile
                // changed it, so we have to restore the previous parse mode.
                l.mode = mode
        }
        return
}

func (l *loader) openScope(comment string) loaderScope {
        l.scope = NewScope(l.scope, l.project, comment)
        cc := setclosure(cloctx.unshift(l.scope))
        return loaderScope{ cc, l.scope }
}

func (l *loader) closeScope(ls loaderScope) {
        if ls.scope != nil {
                l.scope = ls.scope.Outer()
                if ls.cc != nil { setclosure(ls.cc) }

                // Must change the outer of dir scope to globe to avoid Finding symbols
                // recursively.
                if s := ls.scope.Comment(); strings.HasPrefix(s, "dir ") /*|| strings.HasPrefix(s, "file ")*/ {
                        l.globe.SetScopeOuter(ls.scope)
                }
        }
        return
}

func (l *loader) loadProjectBases(linfo *loadinfo, params []Value) (err error) {
        var isDir bool
        var absPath, specName string
        ParamsLoop: for _, elem := range params {
                if specName, err = elem.Strval(); err != nil { return }
                if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
                        break ParamsLoop
                }

                if isDir {
                        err = l.loadDir(specName, absPath, nil)
                } else {
                        err = l.load(specName, absPath, nil)
                }

                // chain loaded base project, note that err might not be nil
                if loaded, _ := l.loaded[absPath]; loaded != nil {
                        l.project.Chain(loaded)
                } else {
                        err = fmt.Errorf("project `%v` not loaded. (%T, %s)", elem, elem, absPath)
                        break ParamsLoop
                }

                // check err after chainning
                if err != nil {
                        if _, ok := err.(scanner.Errors); ok {
                                //fmt.Fprintf(stderr, "%v\n", err)
                        }
                        break ParamsLoop
                }
        }
        return
}

func (l *loader) declare(keyword token.Token, ident *ast.Bareword, options, params []Value) (err error) {
        var optFinal, optNoDock bool
        var bases []Value
        for _, param := range params {
                switch t := param.(type) {
                case *Flag:
                        var s string
                        if s, err = t.Name.Strval(); err != nil { return }
                        switch s {
                        case "final": optFinal = true;
                        case "nodock": optNoDock = true;
                        } 
                default:
                        bases = append(bases, t)
                }
        }

        if ident.Value == "@" {
                var (
                        linfo = l.loads[0]
                        dec, ok = linfo.declares[ident.Value]
                        at, _ = l.globe.scope.Lookup(ident.Value).(*ProjectName)
                )
                if !ok {
                        dec = &declare{ project: at.NamedProject() }
                        linfo.declares[ident.Value] = dec
                }
                dec.backproj = l.project
                dec.backscope = l.scope
                l.useesExecuted = nil
                l.project = at.NamedProject()
                l.scope = l.project.scope
                return nil
        }

        var (
                name = ident.Value
                linfo = l.loads[len(l.loads)-1]
                dec, declared = linfo.declares[name]
        )
        if !declared {
                var (
                        outer = l.scope
                        absDir = linfo.absDir
                        relPath, tmpPath string
                )
                if !filepath.IsAbs(absDir) {
                        //absDir = filepath.Join(l.workdir, absDir)
                        absDir, _ = filepath.Abs(absDir)
                }
                relPath, _ = filepath.Rel(l.workdir, absDir)
                tmpPath = joinTmpPath(l.workdir, relPath)

                // Avoid nesting project scopes!
                for strings.HasPrefix(outer.Comment(), "project \"") {
                        outer = outer.Outer()
                }

                dec = &declare{
                        project: l.globe.project(outer, absDir, relPath, tmpPath, linfo.specName, name),
                }
                
                l.loaded[linfo.absPath()] = dec.project
                linfo.declares[name] = dec
        }

        if loader := linfo.loader; loader != nil {
                if !strings.HasPrefix(loader.scope.comment, "project \"") {
                        l.parser.warn(ident.Pos(), "'%s' not loaded from project scope", name)
                }
                if _, a := loader.scope.ProjectName(loader, name, dec.project); a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                err = fmt.Errorf("Name `%s' already taken (%T).", name, a)
                                return
                        }
                }
        }

        for _, v := range options {
                var bv bool
                switch opt := v.(type) {
                case *Flag:
                        if bv, err = opt.is(0, "multi"); err != nil {
                                return
                        } else if bv {
                                dec.project.allowMultiImported = true
                        }
                case *Pair:
                        if bv, err = opt.isFlag(0, "multi"); err != nil {
                                return
                        } else if bv {
                                dec.project.allowMultiImported = opt.Value.True()
                        }
                default:
                        err = fmt.Errorf("`%v` invalid package option (%T)", v, v)
                        return
                }
        }

        dec.backproj = l.project
        dec.backscope = l.scope
        l.useesExecuted = nil
        l.project = dec.project
        l.scope = l.project.scope

        if err = l.loadPlugin(); err != nil { return }

        // no bases or docking for external packages
        if keyword == token.PACKAGE { return }
        if !optFinal {
                err = l.loadProjectBases(linfo, bases)
                if err != nil { return }
        }

        if declared || l.includeFunc == nil || optionConfigure {
                // Does nothing!
        } else if ctd := l.project.scope.FindDef("CTD"); ctd == nil {
                unreachable()
        } else if s, err := ctd.Strval(); err != nil {
                return err
        } else if file := stat("configuration.sm", "", s); file != nil {
                l.includeFunc(l, ident.Pos(), file)
        }

        if optNoDock || l.project.name == "dock" { return }
        walkSmartBaseDirs(l.project.absPath, func(s string) bool {
                if file := stat("dock", "", filepath.Join(s, ".smart")); !file.exists() {
                        // no docking enabled
                } else if err = l.loadDir("dock", file.FullName(), nil); err != nil {
                        l.parser.error(ident.Pos(), "dock: %v", err)
                } else if loaded, _ := l.loaded[file.FullName()]; loaded != nil {
                        name, _ := l.project.scope.Lookup(loaded.Name()).(*ProjectName)
                        if name == nil {
                                l.parser.error(ident.Pos(), "%v: %v: `dock` is not a project", l.project.name, file)
                        } else {
                                l.useProject(ident.Pos(), loaded, nil)
                        }
                }
                return false
        })
        return
}

func (l *loader) closeCurrent(ident *ast.Bareword) (err error) {
        if ident.Value == "@" {
                if dec, ok := l.loads[0].declares[ident.Value]; ok {
                        l.scope = dec.backscope
                        l.project = dec.backproj
                        l.useesExecuted = dec.useesExecuted
                        dec.backproj = nil
                        dec.backscope = nil
                        dec.useesExecuted = nil
                }
                return nil
        }

        var linfo = l.loads[len(l.loads)-1]
        var dec, ok = linfo.declares[ident.Value]
        if dec == nil || !ok {
                return fmt.Errorf("no loaded project %s", ident.Value)
        }
        if l.project == nil {
                return fmt.Errorf("no current project")
        } else if s := l.project.Name(); s != ident.Value {
                return fmt.Errorf("current project is %s but %s", s, ident.Value)
        } else if l.project != dec.project {
                return fmt.Errorf("project conflicts (%s, %s)", l.project.Name(), dec.project.Name())
        }

        l.scope = dec.backscope
        l.project = dec.backproj
        l.useesExecuted = dec.useesExecuted
        return
}

func (l *loader) OpenNamedScope(name, comment string) (loaderScope, error) {
        if l.scope == nil {
                return loaderScope{}, fmt.Errorf("no parent scope (%v)", comment)
        }
        
        var outer = l.scope
        var scope = NewScope(outer, l.project, comment)
        if strings.HasPrefix(outer.Comment(), "dir ") {
                outer = outer.Outer() // discard dir scope
        }

        outer.ScopeName(l.project, name, scope)

        ls := loaderScope{ setclosure(cloctx.unshift(scope)), scope }
        l.scope = ls.scope
        return ls, nil
}

func (l *loader) resolve(value Value) (obj Object, err error) {
        if sel, ok := value.(*selection); ok {
                if value, err = sel.value(); err == nil {
                        obj, ok = value.(Object)
                }
                return
        }

        var name string
        if name, err = value.Strval(); err != nil {
                return
        }

        if l.scope != nil {
                _, obj = l.scope.Find(name)
        }

        if obj == nil && l.project != nil {
                if obj, err = l.project.resolveObject(name); err != nil {
                        return
                }
        }
        return
}

func (l *loader) find(target Value) (obj Object, err error) {
        var name string
        if name, err = target.Strval(); err != nil { return }
        
        var entry *RuleEntry
        if entry, err = l.project.resolveEntry(name); err != nil {
                return
        } else if entry != nil {
                obj = entry
        }
        return
}

func (l *loader) def(name string) (def *Def, alt Object) {
        var scope = l.scope
        if strings.HasPrefix(scope.comment, "file ") && l.mode&Flat != 0 {
                // use project scope if defining in flat file (aka. include)
                // to ensure that the symbol is valid in the project
                scope = l.project.scope
        }
        return scope.Def(l.project, name, universalnone)
}

func (l *loader) assign(pos token.Pos, tok token.Token, def *Def, alt Object, value Value) (err error) {
        switch tok {
        case token.ASSIGN: // =
                err = def.set(DefDefault, value)
        case token.EXC_ASSIGN: // !=
                err = def.set(DefExecute, value)
        case token.ADD_ASSIGN: // +=
                if !def.Value.refs(value) { err = def.append(value) }
        case token.SHI_ASSIGN: // =+
                if !def.Value.refs(value) {
                        var tail = def.Value
                        if err = def.set(DefDefault, value); err == nil {
                                err = def.append(tail)
                        }
                }
        case token.QUE_ASSIGN: // ?=
                if alt == nil { err = def.set(DefDefault, value) }
        case token.SCO_ASSIGN: // :=
                err = def.set(DefSimple, value)
        case token.DCO_ASSIGN: // ::=
                err = def.set(DefExpand, value)
        }
        return
}

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
//
func readSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch s := src.(type) {
		case string:
			return []byte(s), nil
		case []byte:
			return s, nil
		case *bytes.Buffer:
			// is io.Reader, but src is already available in []byte form
			if s != nil {
				return s.Bytes(), nil
			}
		case io.Reader:
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, s); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
		return nil, fmt.Errorf("invalid source")
	}
	return ioutil.ReadFile(filename)
}

// ParseFile parses the source code of a single source file and returns
// the corresponding ast.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
func (l *loader) ParseFile(filename string, src interface{}, mode Mode) (f *ast.File, err error) {
	// get source
        var text []byte
	if text, err = readSource(filename, src); err != nil { return }

	l.mode = mode //| Trace
	l.tracing.enabled = l.mode&Trace != 0 // for convenience (l.trace is used frequently)
	defer func(saved *parser) {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

                // decouple
                l.parser.loader = nil
                l.parser = saved

                if optSortErrors { l.errors.Sort() }
		err = l.errors.Err()
	} (l.parser)

        // set the current parser
        l.parser = new(parser)
	l.parser.init(l, filename, text)

        // set result values
        if f = l.parser.parseFile(); f == nil {
                // source is not a valid source file - satisfy
                // ParseFile API and return a valid (but) empty
                // *ast.File
                ls := l.openScope(fmt.Sprintf("file %s", filename))
                f = &ast.File{ Name: new(ast.Bareword), Scope: ls.scope }
                l.closeScope(ls)
        }
	return
}

// ParseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (l *loader) ParseConfigDir(pathname, linked string) (err error) {
        var fd *os.File // Directory of the destination.
	if fd, err = os.Open(linked); err != nil { return }
	defer fd.Close()

        var list []os.FileInfo
	if list, err = fd.Readdir(-1); err != nil || len(list) == 0 { return }

        var ident = filepath.Base(pathname)
        if ident == "_" {
                err = fmt.Errorf("invalid package name %s", ident)
                return
        }

        ls, err := l.OpenNamedScope(ident, fmt.Sprintf("config %s", pathname))
        if err != nil { return }
        defer l.closeScope(ls)

        var def *Def
ListLoop:
        for _, d := range list {
                var name = d.Name()
                if strings.HasPrefix(name, ".#") || 
                   strings.HasSuffix(name, "~") || 
                   strings.HasSuffix(name, ".smart") ||
                   strings.HasSuffix(name, ".sm") {
                        continue ListLoop
                }

                var fullname = filepath.Join(linked, name)
                if d.Mode()&os.ModeSymlink != 0 {
                        var ( l string; t os.FileInfo )
                        if l, err = os.Readlink(fullname); err != nil { continue ListLoop }
                        if !filepath.IsAbs(l) { l = filepath.Join(linked, l) }
                        if t, err = os.Stat(l); err != nil { continue ListLoop }
                        if t.IsDir() { continue ListLoop }
                }

                if d.IsDir() {
                        if err = l.ParseConfigDir(filepath.Join(pathname, name), fullname); err != nil { break ListLoop }
                } else if s, a := l.def(name); a != nil {
                        err = fmt.Errorf("declare project: %v", err)
                        break ListLoop
                } else if def = s; def != nil {
                        var ( v []byte; s string )
                        if v, err = ioutil.ReadFile(fullname); err != nil { break ListLoop }
                        if s = string(v); !utf8.ValidString(s) {
                                err = fmt.Errorf("%s: invalid UTF8 content", fullname)
                                break ListLoop
                        }
                        def.set(DefExpand, &String{s})
                } else if s != nil {
                        err =  fmt.Errorf("Name `%s' already taken, not def (%T).", name, s)
                        break ListLoop
                }
        }
        return
}

// ParseDir calls ParseFile for all files with names ending in ".go" in the
// directory specified by path and returns a map of package name -> package
// AST with all the packages found.
//
// If filter != nil, only the files with os.FileInfo entries passing through
// the filter (and ending in ".go") are considered. The mode bits are passed
// to ParseFile unchanged. Position information is recorded in fset.
//
// If the directory couldn't be read, a nil map and the respective error are
// returned. If a parse error occurred, a non-nil but incomplete map and the
// first error encountered are returned.
//
func (l *loader) ParseDir(path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*ast.Project, first error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		return nil, err
	}

        for i, a := range list {
                if i > 0 && a.Name() == "build.smart" {
                        first := list[0]
                        list[0] = a
                        list[i] = first
                }
        }

        ls := l.openScope(fmt.Sprintf("dir %s", path))
        defer l.closeScope(ls)

	mods = make(map[string]*ast.Project)
	ListLoop: for _, d := range list {
                var (
                        filename, mo = filepath.Join(path, d.Name()), d.Mode()
                        linked, linkPath = "", path
                )
                for fn := filename; mo&os.ModeSymlink != 0; {
                        if s, err := os.Readlink(fn); err != nil {
                                continue ListLoop
                        } else {
                                rel := !filepath.IsAbs(s)
                                if rel { s = filepath.Join(linkPath, s) }
                                if fi, err := os.Lstat(s); err != nil {
                                        continue ListLoop
                                } else {
                                        if rel { linkPath = filepath.Dir(s) }
                                        mo, fn = fi.Mode(), s
                                        linked = fn
                                }
                        }
                }

                if s := d.Name(); s != "" {
                        var skip = strings.HasPrefix(s, ".#")
                        skip = skip || !(strings.HasSuffix(s, ".smart") || strings.HasSuffix(s, ".sm"))
                        if skip { continue ListLoop }
                } else if (s == "configure.smart" || s == "configure.sm") && (len(linked) > 0 || mo.IsDir()) {
                        if err := l.ParseConfigDir(filepath.Dir(filename), linked); err != nil {
                                if first == nil {
                                        first = err
                                }
                                return
                        }
                        continue ListLoop
                }

		if mo.IsRegular() && (filter == nil || filter(d)) {
			if src, err := l.ParseFile(filename, nil, mode|parsingDir); err == nil {
                                if src.Name == nil {
                                        first = fmt.Errorf("module '%v' has no name", filename)
                                        return
                                }

				name := src.Name.Value
				mod, found := mods[name]
				if !found {
					mod = &ast.Project{
                                                Name:    name,
                                                Scope:   ls.scope,
                                                Files:   make(map[string]*ast.File),
                                        }
					mods[name] = mod
				}
                                mod.Files[filename] = src
			} else if first == nil {
				first = err
			}
		}
	}
	return
}

// loader.Load loads script from a file or source code (string, []byte).
func (l *loader) load(specName, absPath string, source interface{}) (err error) {
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        if optionBenchImport && d > 50*time.Millisecond {
                                fmt.Fprintf(stderr, "%s││▶%s:load(%s) … (%s)◀\n", l.vs, l.project.name, specName, d)
                        }
                } else if optionBenchSlow && d > 100*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: %s: slow ▶load(%s) … (%s)◀\n", l.project.name, specName, d)
                }
        } (time.Now())

        if absPath == "" {
                err =  fmt.Errorf("no such module `%s' (in paths %v)", specName, l.paths)
                return
        } else if !filepath.IsAbs(absPath) {
                err =  fmt.Errorf("invalid abs name `%s' (%s)", absPath, specName)
                return
        }
        
        // Check already project.
        if loaded, ok := l.loaded[absPath]; ok {
                var (
                        s = l.project.scope
                        name = loaded.Name()
                )
                if _, a := s.ProjectName(l.project, name, loaded); a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                err =  fmt.Errorf("Name `%s' already taken (%T).", name, a)
                        }
                }
                return
        }
        
        var absDir, baseName = filepath.Split(absPath)
        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

        doc, err := l.ParseFile(absPath, source, parseMode)
        if err == nil && doc != nil {
                // TODO: parse documentation
        }
        return
}

func (l *loader) loadDir(specName, absDir string, filter func(os.FileInfo) bool) (err error) {
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        if optionBenchImport && d > 50*time.Millisecond {
                                if l.project == nil {
                                        fmt.Fprintf(stderr, "%s│▶load(%s) … (%s)◀\n", l.vs, specName, d)
                                } else {
                                        fmt.Fprintf(stderr, "%s││▶%s:load(%s) … (%s)◀\n", l.vs, l.project.name, specName, d)
                                }
                        }
                } else if optionBenchSlow && l.project == nil && d>5000*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: slow ▶load(%s)◀ … (%s)\n", specName, d)
                } else if optionBenchSlow && l.project != nil && d>2500*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: %s: slow ▶load(%s)◀ … (%s)\n", l.project.name, specName, d)
                }
        } (time.Now())

        if !filepath.IsAbs(absDir) {
                panic(fmt.Sprintf("Invalid abs name `%s' (%s).", absDir, specName))
                err = fmt.Errorf("Invalid abs name `%s' (%s).", absDir, specName)
                return
        }

        // Check already project.
        if loaded, ok := l.loaded[absDir]; ok {
                var (
                        s = l.project.scope
                        name = loaded.Name()
                )
                if _, a := s.ProjectName(l.project, name, loaded); a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                err = fmt.Errorf("Name `%s' already taken (%T).", name, a)
                        }
                }
                return
        }

        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

        mods, err := l.ParseDir(absDir, filter, parseMode)
        if err == nil && mods != nil {
                // FIXME: no modules parsed
        }
        return
}

func (l *loader) loadFile(filename string, source interface{}) error {
        s, _ := filepath.Split(filename)
        s, _  = filepath.Rel(l.workdir, s)
        return l.load(s, filename, source)
}

func (l *loader) loadPath(path string, filter func(os.FileInfo) bool) (err error) {
        s, _ := filepath.Rel(l.workdir, path)
        return l.loadDir(s, path, filter)
}

func (l *loader) loadText(filename string, text string) []Value {
	defer func(saved *parser) {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

                // decouple
                l.parser.loader = nil
                l.parser = saved

                /*if optSortErrors {
                        l.errors.Sort()
                }
		err = l.errors.Err()*/
	} (l.parser)

        if l.globe.main == nil {
                return
        }

        l.useesExecuted = nil
        l.project = l.globe.main
        l.scope = l.project.scope

        l.parser = new(parser)
        l.parser.init(l, filename, []byte(text))
        return l.exprs(l.parser.parseText())
}

func AddSearchPaths(paths... string) (err error) {
        for _, s := range paths {
                if s, err = filepath.Abs(s); err != nil {
                        break
                }
                if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
                        globalPaths = append(globalPaths, s)
                }
        }
        return
}
