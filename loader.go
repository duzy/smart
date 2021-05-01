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
        "runtime/debug"
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

const dotContainer = ".container"

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

func (li *loadinfo) breakUseLoop() (result bool) {
        var first bool = true
        for _, decl := range li.declares {
                if first || result {
                        result = decl.project.breakUseLoop
                }
                if !result { break }
                first = false
        }
        return
}

type loaderScope struct {
        cc closurecontext
        scope *Scope
}

type loader struct {
        *Context
        *parser
        tracing // tracing/debugging
        fset     *token.FileSet
        paths    searchlist
        loadArgs []Value
        loads    []*loadinfo
        loaded   map[string]*Project // loaded projects
        loadStack []*Project // load path
        useStack []*Project // use path
        useesExecuted []*Project // all executed usees
        project  *Project // the current project
        scope    *Scope   // the current scope
        isIncludingConf bool // including configuration
        vs string // verbose prefix
}

func (l *loader) error(pos token.Pos, f string, a... interface{}) {
        var pp Position
        if l.parser != nil {
                pp = Position(l.parser.file.Position(pos))
        }
        diag.errorAt(pp, f, a...)
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
                fmt.Fprintf(stderr, "exit: %v from '%s' → %v\n", names, loader.Name(), linfo.scope)
        } else {
                fmt.Fprintf(stderr, "exit: %v → %v\n", names, linfo.scope)
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
                strings.HasPrefix(specName, "~\\") ||
                strings.HasPrefix(specName, "~/") ||
                strings.HasPrefix(specName, "./") ||
                strings.HasPrefix(specName, "../") ||
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

type genericoptions struct {
        keyword token.Token
        verbose bool // TODO: verbose operation
        dontOperate bool
        options []Value
}

type useoptions struct {
        allowReuse bool
}

type importoptions struct {
        useoptions
}

type importspecoptions struct {
        unuse bool
        reuse bool
}

func (l *loader) parseUseProps(props []ast.Expr) (opts importspecoptions, params []Value, err error) {
        // Supported parameter forms:
        //      -param
        //      -param(value)
        //      -param=value
        var useList []Value // TODO: apply useList
        for _, prop := range props {
                var s string
                switch v := l.expr(prop); t := v.(type) {
                case *Flag:
                        if s, err = t.name.Strval(); err != nil {
                                diag.errorOf(v, "invalid flag `%v` (%v)", v, err)
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
                                if s, err = tt.name.Strval(); err != nil {
                                        diag.errorOf(t.Key, "invalid flag name `%v` (%v)", tt.name, err)
                                        return
                                }
                                switch s {
                                case "use": useList = append(useList, t.Value)
                                default: params = append(params, v)
                                }
                        default:
                                diag.errorOf(t.Key, "parameter `%v' unsupported `%T`", v, v)
                                return
                        }
                case *Argumented: // -param(value)
                        switch tt := t.value.(type) {
                        case *Flag:
                                if s, err = tt.name.Strval(); err != nil {
                                        diag.errorOf(t.value, "invalid flag name `%v` (%v)", tt.name, err)
                                        return
                                }
                                switch s {
                                case "use": useList = append(useList, t.args...)
                                default: params = append(params, v)
                                }
                        default:
                                diag.errorOf(t.value, "parameter `%v' unsupported `%T`", v, v)
                                return
                        }
                default:
                        diag.errorOf(v, "parameter `%v` unsupported `%T`", v, v)
                        return
                }
        }
        return
}

func (l *loader) loadUseSpec(opts importoptions, spec *ast.UseSpec) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadUseSpec")) }
        var (
                specOpts importspecoptions
                specNames []string
                specVal Value
                params []Value
                err error
        )
        if len(spec.Props) < 1 {
                l.error(spec.Pos(), "insufficient props")
                return
        }

        specVal = l.expr(spec.Props[0])
        if specOpts, params, err = l.parseUseProps(spec.Props[1:]); err != nil {
                return
        }

        switch v := specVal.(type) {
        case *Pair:
                var s string
                if f, ok := v.Key.(*Flag); !ok {
                        l.error(spec.Props[0].Pos(), "'%v' invalid use spec", v.Key)
                        return
                } else if s, err = f.name.Strval(); err != nil {
                        l.error(spec.Props[0].Pos(), "%s", err)
                        return
                } else if s != "list" {
                        l.error(spec.Props[0].Pos(), "'%v' invalid use spec, do you mean -list?", v.Key)
                        return
                }

                var list []Value
                if list, err = mergeresult(ExpandAll(v.Value)); err != nil {
                        l.error(spec.Props[0].Pos(), "%s", err)
                        return
                }
                for _, val := range list {
                        if s, err = val.Strval(); err != nil {
                                l.error(spec.Props[0].Pos(), "%s", err)
                                return
                        } else if s == "" { continue }
                        specNames = append(specNames, s)
                }
        default:
                var specName string
                if specName, err = specVal.Strval(); err != nil {
                        l.error(spec.Props[0].Pos(), "%s", err)
                        return
                } else if specName == "" { break }
                specNames = append(specNames, specName)
        }

        if len(specNames) == 0 {
                l.error(spec.Props[0].Pos(), "empty use spec (%v)", spec.Props[0])
                return
        }
        for _, specName := range specNames {
                l.loadUseSpecName(opts, spec, specName, &specOpts, params)
        }

        if err == nil && false {
                var using Object
                if using, err = l.project.resolveObject("using.*"); err == nil && !isNil(using) {
                        if def, ok := using.(*Def); ok {
                                l.info(spec.Pos(), "TODO: using: %T %v", def, def.value)
                        }
                }
                err = nil
        }
        return
}

func (l *loader) loadUseSpecName(opts importoptions, spec *ast.UseSpec, specName string, specOpts *importspecoptions, params []Value) {
        var (
                linfo = l.loads[len(l.loads)-1]
                err error
        )

        var ( absPath string; isDir bool )
        if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
                l.error(spec.Pos(), "no such package `%v`", specName)
                return
        } else if absPath == "" {
                l.error(spec.Pos(), "missing `%s` (in %v)", specName, l.paths)
                return
        }

        var loaded, loadedValid = l.loaded[absPath]

        // Checking circular loads. See also Project.loopImportPath()!
        var breakUseLoop bool
        for i, load := range l.loads {
                if load.absDir == absPath {
                        var s string
                        var loop, loopBreakers []*loadinfo
                        for n := i; n < len(l.loads); n += 1 {
                                var load = l.loads[n]
                                loop = append(loop, load)
                                if load.breakUseLoop() {
                                        loopBreakers = append(loopBreakers, load)
                                        s += "<" + load.specName + "> → "
                                } else {
                                        s += load.specName + " → "
                                }
                        }
                        if loadedValid && loaded.breakUseLoop {
                                s += "<" + specName + ">"
                        } else {
                                s += specName
                        }

                        breakUseLoop = (loopBreakers != nil)
                        if !breakUseLoop {
                                l.error(spec.Pos(), "loop detected: %s", s)
                        } else if optionVerboseImport || optionVerboseUsing || optionVerboseLoading {
                                fmt.Fprintf(stderr, "%s: loop detected: %v\n", l.project, s)
                        }
                }
        }

        /*if breakUseLoop {
                if loadedValid {
                        _, a := l.project.scope.ProjectName(l.project, loaded.Name(), loaded)
                        if a != nil {
                                if v, ok := a.(*ProjectName); !ok || v == nil {
                                        err = fmt.Errorf("`%s' name already taken (%T).", loaded.Name(), a)
                                }
                        }
                }
                return
        }*/

        defer func(a []*Project) { l.loadStack = a } (l.loadStack)
        l.loadStack = append(l.loadStack, l.project) // build the load path

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
                if len(l.loadStack) > 1 {
                        defer func(s string) { l.vs = s } (l.vs)
                        l.vs += "│"
                }
                if specOpts.reuse {
                        fmt.Fprintf(stderr, "%s├┬→\"%s\" (reuse, %s)\n", l.vs, specName, absPath)
                } else {
                        fmt.Fprintf(stderr, "%s├┬→\"%s\" (%s)\n", l.vs, specName, absPath)
                }
                defer func(t time.Time) {
                        var name string
                        var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s
                        var ds = fmt.Sprintf("(%s)", d)
                        if d>=1*time.Second { ds = fmt.Sprintf("▶%s◀",ds) }
                        if loaded != nil { name = loaded.name }
                        fmt.Fprintf(stderr, "%s├┴─\"%s\" ⇢ %s %s\n", l.vs, specName, name, ds)
                } (time.Now())
        }

        if loadedValid && !specOpts.reuse {
                var ( proj *Project ; res, isb bool )
                if proj, res, isb, err = l.project.hasLoaded(loaded, breakUseLoop); err != nil {
                        l.error(spec.Pos(), "`%s`: %s", specName, err)
                        return
                } else if isb {
                        l.error(spec.Pos(), "`%s` is a base (%s)", specName, proj.name)
                        return
                } else if res {
                        l.error(spec.Pos(), "'%s' already imported by '%s'", specName, proj.name)
                        return
                }
        }

        if /*!loadedValid || loaded == nil*/true {
                if isDir {
                        err = l.loadDir(specName, absPath, nil)
                } else {
                        err = l.load(specName, absPath, nil)
                }
                if err != nil {
                        switch e := err.(type) {
                        case *scanner.Error:
                                // Report errors immediately, so that they could be
                                // discoverred asap.
                                fmt.Fprintf(stderr, "%v\n", e)
                                l.error(spec.Pos(), "import `%v` failed (%v)", specName, absPath)
                        default:
                                l.error(spec.Pos(), "import `%v` (%v): %v", specName, absPath, err)
                        }
                        return
                }

                if loaded != nil {
                        // already loaded previously
                } else if loaded, loadedValid = l.loaded[absPath]; loadedValid {
                        // successfully loaded (first)
                } else {
                        l.error(spec.Pos(), "'%s' not loaded (%s)", specName, absPath)
                }

                if loaded == nil {
                        l.error(spec.Pos(), "'%s' not smart project", specName)
                        return
                }
        }
        if breakUseLoop { /*return*/ }

        // Check against the current load list before appending loaded.
        for _, lp := range l.project.loads {
                var ( proj *Project ; res, isb bool )

                if loaded == lp {
                        l.error(spec.Pos(), "using `%s` multiple times", specName)
                        return
                }

                if proj, res, isb, err = loaded.hasLoaded(lp, breakUseLoop); err != nil {
                        l.error(spec.Pos(), "%s: %s", specName, err)
                        return
                } else if isb {
                        if l.project.hasBase(lp) {
                                // common bases are fine
                        } else {
                                l.error(spec.Pos(), "`%s` is already a base", specName)
                        }
                } else if res && !lp.multiUseAllowed {
                        l.parser.warn(spec.Pos(), "`%s` has already imported `%s` (from %s)", loaded, lp, proj)
                }

                if proj, res, isb, err = lp.hasLoaded(loaded, breakUseLoop); err != nil {
                        l.error(spec.Pos(), "%s: %s", specName, err)
                        return
                } else if isb {
                        l.parser.warn(spec.Pos(), "`%s` is already base of `%s` (%s)", loaded, lp, proj)
                } else if res && !loaded.multiUseAllowed {
                        l.parser.warn(spec.Pos(), "`%s` has already been imported by `%s` (from %s)", loaded, lp, proj)
                }
        }

        if specOpts.unuse || breakUseLoop { return } else {
                // The project load list is different from using list.
                l.project.loads = append(l.project.loads, loaded)
        }

        name, _ := l.project.scope.Lookup(loaded.name).(*ProjectName)
        if name == nil {
                l.error(spec.Pos(), "%v (%v,dir=%v) not in %v", specName, absPath, isDir, l.project.scope.comment)
                return
        }
        
        var useopts = opts.useoptions
        if specOpts.reuse {
                // override reuse option
                useopts.allowReuse = true
        }

        if optionVerboseImport {
                defer func(t time.Time) {
                        var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s ┼
                        fmt.Fprintf(stderr, "%s├┤ %s:import(%s) (%s)\n", l.vs, l.project, specName, d)
                } (time.Now())
        }

        var pos = spec.Props[0].Pos()
        if err = l.useProject(pos, loaded, params, useopts); err == nil && !specOpts.unuse {
                var ( using Value; names []string )
                if o, e := l.project.resolveObject("using.*"); e == nil && !isNil(o) {
                        if def, ok := o.(*Def); ok && !isNil(def) { using = def.value }
                }
                if !(isNil(using) || isNone(using)) {
                        if s, e := using.Strval(); e == nil {
                                names = strings.Fields(s)
                        }
                }
		var position = Position(l.file.Position(pos))
                for _, nameprops := range names {
                        var parts = strings.Split(nameprops, ".")
                        var name = parts[0]
                        var usingVarName = fmt.Sprintf("using.%s", name)
			var def, alt = l.project.scope.define(l.project, name, &None{trivial{position}})
                        if def != nil && alt == nil { // if it's new Def (first time being defined)
                                if false { l.info(pos, "1: %v", def) }
                                for _, base := range l.project.bases {
                                        if obj, err := base.resolveObject(name); err == nil && !(isNil(obj) || isNone(obj)) {
                                                def.append(obj) // append all base Defs (act like '+=')
                                        }
                                }
                                if false { l.info(pos, "2: %v", def) }
                        }
			if def == nil && alt != nil { def, _ = alt.(*Def) }
                        if o := loaded.scope.Lookup(usingVarName); !isNil(o) {
                                if false { l.info(pos, "3: %v; %v", def, o) }
                                def.append(o)
                        }
                }
        }
        return
}

const pluginDifferentVersionError = `plugin was built with a different version of package`
var numUpdatedPlugins = 0

func buildPlugin(s, src string) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.buildPlugin")) }

        fmt.Fprintf(stderr, "smart: Build %v …", src)
        dir, _ := filepath.Split(src)
        o := &bytes.Buffer{}
        c := exec.Command("go", "build", "-buildmode=plugin", "-o", s)
        c.Stdout, c.Stderr, c.Dir = o, o, dir
        if err = c.Run(); err == nil {
                numUpdatedPlugins += 1
                fmt.Fprintf(stderr, "… ok\n")
                fmt.Fprintf(stderr, "smart: Plugin updated, please relaunch.\n")
                os.Exit(0)
        } else {
                fmt.Fprintf(stderr, "… error\n")
                fmt.Fprintf(stderr, "%s", o)
        }
        return
}

func (l *loader) loadPlugin(pos Position) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadPlugin")) }

        g := stat(pos, "smart.go", "", l.project.absPath)
        if g == nil { return /* smart.go was not presented */ }

        var src string
        if src, err = g.Strval(); err != nil { return }

        s := strings.Replace(l.project.relPath, "..", "_", -1)
        s = filepath.Join(filepath.Dir(joinTmpPath("", "")), "plugins", s)

        var build = true

        so := stat(pos, /*l.project.name*/"plugin", "", s, nil)
        if s, err = so.Strval(); err != nil { return }
        if exists(so) && !optionAlwaysBuildPlugins {
                if so.info.ModTime().After(g.info.ModTime()) {
                        build = false // Plugin already updated.
                }
        }
        if build { err = buildPlugin(s, src) }
        if err != nil { return }

        // Once plugin is opened, there's no need/way to close it.
        if l.project.plugin, err = plugin.Open(s); err == nil {
                var p plugin.Symbol
                if p, err = l.project.plugin.Lookup("Init"); err != nil {
                        return
                } else if p == nil {
                        // no initialization (optional)
                } else if f, ok := p.(func(Position, *Project) (*Scope, error)); ok {
                  l.project.pluginScope, err = f(pos, l.project)
                } else if f, ok := p.(func(*Project) (*Scope, error)); ok {
                  l.project.pluginScope, err = f(l.project)
                }
        } else if es := err.Error(); strings.Contains(es, pluginDifferentVersionError) {
                err = buildPlugin(s, src)
        }
        return
}

func (l *loader) exprEvaluated(x *ast.EvaluatedExpr) (v Value) {
        var ok bool
        if x.Data == nil {
                l.error(x.Pos(), "evaluated data is nil `%T`", x.Expr)
        } else if v, ok = x.Data.(Value); !ok {
                l.error(x.Pos(), "evaluated data is not value `%T`", x.Data)
        }
        return v
}

func (l *loader) exprArgumented(x *ast.ArgumentedExpr) Value {
        return &Argumented{
                value: l.expr(x.X),
                args: l.exprs(x.Arguments),
        }
}

func (l *loader) exprClosureDelegate(x *ast.ClosureDelegate) (name Value, obj Value) {
        if name = l.expr(x.Name); name == nil {
                l.error(x.Name.Pos(), "invalid name `%T`", x.Name)
                return
        }

        var tok = token.ILLEGAL
        switch x.TokLp {
        case token.LPAREN, token.LBRACE, token.LCOLON:
                tok = x.TokLp
        case token.STRING, token.COMPOUND:
                if x.Tok == token.DELEGATE {
                        l.error(x.TokPos, "unsupported delegate (%v).", x.TokLp)
                        return
                } else {
                        tok = x.TokLp
                }
        case token.ILLEGAL:
                if x.Tok.IsClosure() || x.Tok.IsDelegate() {
                        tok = token.LPAREN
                } else {
                        l.error(x.TokPos, "unregonized closure/delegate (%v).", x.Tok)
                        return
                }
        default:
                if x.Tok == token.DELEGATE {
                        l.error(x.TokPos, "unregonized delegate (%v).", x.TokLp)
                } else {
                        l.error(x.TokPos, "unregonized closure (%v).", x.TokLp)
                }
                return
        }

        var ( resolved Value; err error )
        if x.Resolved != nil { resolved = x.Resolved.(Value) }

        switch tok {
        case token.LCOLON:
                s, err := name.Strval()
                if err != nil { l.error(x.Name.Pos(), "%v", err); return }
                switch s {
                //case "arch": obj = context.globe.arch
                case "os": obj = context.globe.os.self
                case "goals": obj = context.goals
                case "mode": obj = context.mode
                case "self": obj = l.project.self
                case "usee": obj = l.project.using
                default:
                        l.error(x.Name.Pos(), "unsupported special delegation")
                        return
                }
        case token.LPAREN:
                if resolved == nil { // if not resolved at parse time
                        if resolved, err = l.resolve(name); err != nil {
                                l.error(x.Name.Pos(), "%s", err)
                                return
                        }
                }
                /*if s := typeof(name); s == "selection" {
                        var pos = Position(l.parser.file.Position(x.Pos()))
                        fmt.Fprintf(stderr, "%s: %s %v %v\n", pos, s, name, resolved)
                }*/
                if resolved != nil {
                        if sel, ok := resolved.(*selection); ok {
                                obj = sel
                        } else if def, _ := resolved.(Caller); def == nil {
                                l.error(x.Name.Pos(), "uncallable `%s` (%s) resolved", name, typeof(resolved))
                                return
                        } else if obj = def.(Object); obj == nil {
                                l.error(x.Name.Pos(), "non-object callable `%s` resolved `%T`", name, def)
                                return
                        }
                } else if l.isIncludingConf {
                        s, err := name.Strval()
                        if err != nil { l.error(x.Name.Pos(), "%v", err); return }

                        // Create an empty Def if it's referred in
                        // configuration.sm.
                        def, _ := l.def(x.Name.Pos(), s)
                        def.origin = DefConfRef
                        obj = def
                }
        case token.LBRACE:
                if resolved == nil { // if not resolved at parse time
                        if resolved, err = l.find(name); err != nil {
                                l.error(x.Name.Pos(), "%s", err)
                                return
                        } else if resolved == nil {
                                //l.error(x.Name.Pos(), "entry `%s` is nil", name)
                                return
                        }
                }
                if exe, _ := resolved.(Executer); exe != nil {
                        if obj = exe.(Object); obj == nil {
                                l.error(x.Name.Pos(), "resolved Executer `%s` of `%T` is not Object", name, resolved)
                                return
                        }
                } else {
                        l.error(x.Name.Pos(), "resolved `%s` of `%T` is not Executer", name, resolved)
                        return
                }
        case token.STRING, token.COMPOUND:
                if resolved == nil { // if not resolved at parse time
                        if resolved, err = l.find(name); err != nil {
                                l.error(x.Name.Pos(), "%s", err)
                                return
                        } else if resolved == nil {
                                //resolved = unresolved(l.project, name)
                        }
                }
                obj = resolved//.(Object)
        }
        if obj == nil && l.project.plugin != nil {
                if l.project.pluginScope != nil {
                        s, err := name.Strval()
                        if err != nil { l.error(x.Name.Pos(), "%v", err); return }
                        obj = l.project.pluginScope.Lookup(s)
                }
        }
        return
}

func (l *loader) exprClosure(x *ast.ClosureExpr) (v Value) {
        if name, obj := l.exprClosureDelegate(&x.ClosureDelegate); name == nil {
                l.error(x.Name.Pos(), "invalid closure name `%T`", x.Name)
        } else if obj != nil {
                v = MakeClosure(Position(x.Position), x.TokLp, obj, l.exprs(x.Args)...)
        } else if true {
                obj = unresolved(l.project, name)
                v = MakeClosure(Position(x.Position), x.TokLp, obj, l.exprs(x.Args)...)
        } else {
                l.error(x.Pos(), "closure nil object `%v` (from %v)", name, l.scope.comment)
        }
        return
}

func (l *loader) exprDelegate(x *ast.DelegateExpr) (v Value) {
        if name, obj := l.exprClosureDelegate(&x.ClosureDelegate); name == nil {
                l.error(x.Name.Pos(), "`%T` is invalid delegation name", x.Name)
        } else if obj != nil {
                v = MakeDelegate(Position(x.Position), x.TokLp, obj, l.exprs(x.Args)...)
        } else if sel, ok := name.(*selection); ok {
                if o, err := sel.object(); err == nil && o.DeclScope().comment == usecomment {
                        obj = unresolved(l.project, name)
                        v = MakeDelegate(Position(x.Position), x.TokLp, obj, l.exprs(x.Args)...)
                } else if err != nil {
                        l.error(x.Name.Pos(), "`%v` invalid delegate selection", name)
                } else if o == nil {
                        l.error(x.Name.Pos(), "`%v` nil selection object", name)
                } else if v, err = sel.value(); err != nil {
                        l.error(x.Name.Pos(), "`%v` invalid delegate selection", name)
                } else if v == nil {
                        if !l.isIncludingConf {
                                l.error(x.Name.Pos(), "`%v` not found in %v", sel.s, o)
                        } else {
                                unreachable("`%v` nil delegation", name)
                        }
                } else if obj, ok := v.(Object); ok {
                        v = MakeDelegate(Position(x.Position), x.TokLp, obj, l.exprs(x.Args)...)
                } else {
                        // just use the selected value
                }
        } else if name.refdef(defany) || name.closured() { // recursive delegation or closure
          obj = unresolved(l.project, name)
          v = MakeDelegate(Position(x.Position), x.TokLp, obj, l.exprs(x.Args)...)
        } else {
          l.error(x.Name.Pos(), "delegate nil object `%v` (%T, %v, from %v)", name, name, v, l.scope.comment)
        }
        return
}

func (l *loader) exprSelection(x *ast.SelectionExpr) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        var obj = l.expr(x.Lhs)
        if obj == nil {
                l.error(x.Lhs.Pos(), "`%s` invalid object expression (%T)", x, x.Lhs)
                return
        }
        switch t := obj.(type) {
        case *selection:
                // TODO: ...
        case *Bareword:
                switch t.string {
                case "usee": obj = l.project.using
                case "self": obj = l.project.self
                case "os": obj = context.globe.os.self
                case "goals": obj = context.goals
                default:
                        if o, err := l.resolve(obj); err != nil {
                                l.error(x.Lhs.Pos(), "selection expression `%v`: %v", obj, err)
                                return
                        } else if o == nil {
                                if x.Tok == token.SELECT_PROG2 {
                                        v = &Nil{trivial{pos}} // ignore
                                } else {
                                        l.error(x.Lhs.Pos(), "`%v` is undefined", obj)
                                }
                                return
                        } else {
                                obj = o
                        }
                }
        case *Barecomp: // for cases like '.foo'
                if o, err := l.resolve(t); err != nil {
                        l.error(x.Lhs.Pos(), "selection expression `%v`: %v", obj, err)
                        return
                } else if o == nil {
                        if x.Tok == token.SELECT_PROG2 {
                                v = &Nil{trivial{pos}} // ignore
                        } else {
                                l.error(x.Lhs.Pos(), "`%v` is undefined", obj)
                        }
                        return
                } else {
                        obj = o
                }
        }

        if prop := l.expr(x.Rhs); prop == nil {
                if x.Tok == token.SELECT_PROG2 {
                        v = &Nil{trivial{pos}} // ignore
                } else {
                        l.error(x.Rhs.Pos(), "`%s` invalid property expression (%T)", x, x.Rhs)
                }
        } else {
                v = &selection{trivial{pos}, x.Tok, obj, prop}
        }
        return
}

func (l *loader) exprBasicLit(x *ast.BasicLit) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        switch x.Kind {
        case token.BAR: l.error(x.Pos(), "`|` is deprecated, changed the modifiers!")
        case token.BIN:      v = ParseBin(pos,x.Value)
        case token.OCT:      v = ParseOct(pos,x.Value)
        case token.INT:      v = ParseInt(pos,x.Value)
        case token.HEX:      v = ParseHex(pos,x.Value)
        case token.FLOAT:    v = ParseFloat(pos,x.Value)
        case token.DATETIME: v = ParseDateTime(pos,x.Value)
        case token.DATE:     v = ParseDate(pos,x.Value)
        case token.TIME:     v = ParseTime(pos,x.Value)
        case token.URI:      v = ParseURL(pos,x.Value)
        case token.BAREWORD: v = &Bareword{trivial{pos},x.Value}
        case token.STRING:   v = &String{trivial{pos},x.Value}
        case token.ESCAPE:   v = &String{trivial{pos},EscapeChar(x.Value)}
        case token.RAW:      v = &Raw{trivial{pos},x.Value}
        default: unreachable()
        }
        return
}

func (l *loader) exprBareword(x *ast.Bareword) (res Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        res = &Bareword{trivial{pos},x.Value}
        return
}

func (l *loader) exprQualiword(x *ast.Qualiword) (res Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        res = &Qualiword{trivial{pos},x.Words}
        return
}

func (l *loader) exprConstant(x *ast.Constant) (res Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        switch x.Tok {
        case token.TRUE:  res = &boolean{trivial{pos},true}
        case token.FALSE: res = &boolean{trivial{pos},false}
        case token.YES:   res = &answer{trivial{pos},true}
        case token.NO:    res = &answer{trivial{pos},false}
        }
        return
}

func (l *loader) exprBarecomp(x *ast.Barecomp) (res Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        res = MakeBarecomp(pos, l.exprs(x.Elems)...)
        return
}

func (l *loader) exprBarefile(x *ast.Barefile) (v Value) {
        if file, _ := x.File.(*File); file != nil {
                var pos = Position(l.parser.file.Position(x.Pos()))
                if x.Val != nil {
                        v = &Barefile{trivial{pos},x.Val.(Value),file}
                } else {
                        v = &Barefile{trivial{pos},l.expr(x.Name),file}
                }
        }
        if v == nil {
                l.error(x.Pos(), "invalid barefile `%s` (%T)", x.Name, x.File)
        }
        return
}

func (l *loader) convertBarefiles(targets []ast.Expr) []ast.Expr {
        for i, target := range targets {
                var pos = Position(l.parser.file.Position(target.Pos()))
                switch t := target.(type) {
                case *ast.Bareword:
                        if file := l.project.matchFile(t.Value); file != nil {
                                targets[i] = &ast.Barefile{Name:target,File:file}
                                file.position = pos
                        }
                case *ast.Barecomp:
                        var value = l.exprBarecomp(t)
                        if value.closured() || value.refdef(DefArg) { break }
                        target = &ast.EvaluatedExpr{target, value}
                        if s, err := value.Strval(); err != nil {
                                l.error(t.Pos(), "%v: %v", value, err)
                        } else if file := l.project.matchFile(s); file != nil {
                                targets[i] = &ast.Barefile{Name:target,File:file}
                                file.position = pos
                        }
                case *ast.ArgumentedExpr:
                        vals := l.convertBarefiles(append([]ast.Expr{t.X},t.Arguments...))
                        t.X, t.Arguments = vals[0], vals[1:]
                }
        }
        return targets
}

func (l *loader) exprURL(x *ast.URLExpr) (res Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        var url = &URL{ Scheme:l.expr(x.Scheme) }
        if x.Username != nil { url.Username = l.expr(x.Username) }
        if x.Password != nil { url.Password = l.expr(x.Password) } else if x.Colon2 != token.NoPos {
                url.Password = &None{trivial{pos}}
        }
        if x.Host != nil { url.Host = l.expr(x.Host) }
        if x.Port != nil { url.Port = l.expr(x.Port) } else if x.Colon3 != token.NoPos {
                url.Port = &None{trivial{pos}}
        }
        if x.Path != nil { url.Path = l.expr(x.Path) }
        if x.Query != nil { url.Query = l.expr(x.Query) } else if x.Que != token.NoPos {
                url.Query = &None{trivial{pos}}
        }
        if x.Fragment != nil { url.Fragment = l.expr(x.Fragment) } else if x.NumSign != token.NoPos {
                url.Fragment = &None{trivial{pos}}
        }
        return url
}

func (l *loader) exprPath(x *ast.PathExpr) (res Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        res = MakePath(pos, l.exprs(x.Segments)...)
        return
}

func (l *loader) exprPathSeg(x *ast.PathSegExpr) (v Value) {
        var pos = Position(l.parser.file.Position(l.pos))
        switch x.Tok {
        case token.PCON:   v = MakePathSeg(pos,'/') // TODO: should be NONE
        case token.TILDE:  v = MakePathSeg(pos,'~')
        case token.DOT:    v = MakePathSeg(pos,'.')
        case token.DOTDOT: v = MakePathSeg(pos,'^') // 
        case 0: v = MakePathSeg(pos,0) // the tailing empty segment after '/', e.g. /foo/bar/
        default: l.error(x.Pos(), "unsupported path segment `%v`", x.Tok)
        }
        return
}

func (l *loader) exprFlag(x *ast.FlagExpr) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        if x.Name == nil {
                v = &Flag{trivial{pos},&None{trivial{pos}}}
        } else {
                v = &Flag{trivial{pos},l.expr(x.Name)}
        }
        return
}

func (l *loader) exprNeg(x *ast.NegExpr) (v Value) {
        v = Negative(l.expr(x.Val))
        return
}

func (l *loader) exprCompoundLit(x *ast.CompoundLit) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        v = MakeCompound(pos, l.exprs(x.Elems)...)
        return
}

func (l *loader) exprGroup(x *ast.GroupExpr) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        v = MakeGroup(pos, l.exprs(x.Elems)...)
        return
}

func (l *loader) exprList(x *ast.ListExpr) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        v = MakeList(pos, l.exprs(x.Elems)...)
        return
}

func (l *loader) exprKeyValue(x *ast.KeyValueExpr) (res Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        if k := l.expr(x.Key); l.parser.bits&parsingFilesSpec != 0 {
                res = &Pair{trivial{pos},k,l.expr(x.Value)}
        } else {
                res = MakePair(pos,k,l.expr(x.Value))
        }
        return
}

func (l *loader) exprPerc(x *ast.PercExpr) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        v = MakePercPattern(pos, l.expr(x.X), l.expr(x.Y))
        return
}

func (l *loader) exprGlob(x *ast.GlobExpr) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        v = MakeGlobPattern(pos, l.exprs(x.Components)...)
        return
}

func (l *loader) exprGlobMeta(x *ast.GlobMeta) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        v = MakeGlobMeta(pos, x.Tok)
        return
}

func (l *loader) exprGlobRange(x *ast.GlobRange) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        v = MakeGlobRange(pos, l.expr(x.Chars))
        return
}

func (l *loader) exprModifierGroup(x *ast.ModifiersExpr) Value {
        var modifiers []*modifier
        for i, elem := range l.exprs(x.Elems) {
                switch t := elem.(type) {
                case *Group:
                        // Just ignore empty modifier
                        if len(t.Elems) == 0 { continue }
                        var m = &modifier{
                                trivial: trivial{t.Position()},
                                name: t.Elems[0],
                        }
                        if len(t.Elems) > 1 {
                                m.args = t.Elems[1:]
                        }
                        modifiers = append(modifiers, m)
                default:
                        l.error(x.Elems[i].Pos(), "invalid modifier `%v` (%T)", elem, elem)
                }
        }
        return &modifiergroup{
                trivial: trivial{Position(l.parser.file.Position(x.Pos()))},
                modifiers: modifiers,
        }
}

func (l *loader) exprRecipe(x *ast.RecipeExpr) (v Value) {
        var pos = Position(l.parser.file.Position(x.Pos()))
        if len(x.Elems) == 0 {
                v = &None{trivial{pos}}
        } else if x.Dialect == "" || x.Dialect == "eval" {
                v = MakeList(pos, l.exprs(x.Elems)...)
        } else {
                v = MakeCompound(pos, l.exprs(x.Elems)...)
        }
        return
}

func (l *loader) exprRecipeDefineClause(x *ast.RecipeDefineClause) (v Value) {
        return &undetermined{ x.Tok, l.expr(x.Name), l.expr(x.Value) }
}

func (l *loader) exprIncludeRuleClause(x *ast.IncludeRuleClause) (v Value) {
        entries := l.rule(x.RuleClause, specialRuleNor, nil)
        if n := len(entries); n == 1 {
                v = entries[0]
        } else if n > 1 {
                l.error(x.Pos(), "including multiple target `%v`", x)
        } else {
                l.error(x.Pos(), "invalid rule `%v`", x)
        }
        return
}

func (l *loader) expr(expr ast.Expr) (v Value) {
        if expr == nil {
                v = &None{trivial{Position(l.parser.file.Position(l.pos))}}
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
        case *ast.Qualiword:
                v = l.exprQualiword(x)
        case *ast.Constant:
                v = l.exprConstant(x)
        case *ast.Barecomp:
                v = l.exprBarecomp(x)
        case *ast.Barefile:
                v = l.exprBarefile(x)
        case *ast.URLExpr:
                v = l.exprURL(x)
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
                v = l.exprGlobRange(x) // FIXME: `[...]` is used for modifier expressions
        case *ast.ModifiersExpr: // [(foo a b c) (bar a b c)]
                v = l.exprModifierGroup(x)
        case *ast.RecipeExpr:
                v = l.exprRecipe(x)
        case *ast.RecipeDefineClause:
                v = l.exprRecipeDefineClause(x)
        case *ast.IncludeRuleClause:
                v = l.exprIncludeRuleClause(x)
        case *ast.BadExpr:
                l.error(x.Pos(), "bad expr")
                return
        }

        if v == nil {
                if l.isIncludingConf {
                        v = &Nil{trivial{Position(l.parser.file.Position(expr.Pos()))}}
                } else {
                        l.error(expr.Pos(), "nil expr: `%v` (%s)", expr, typeof(expr))
                }
        }
        return
}

func (l *loader) exprs(exprs []ast.Expr) (values []Value) {
        for _, x := range exprs { values = append(values, l.expr(x)) }
        return
}

func (l *loader) useProject(pos token.Pos, usee *Project, params []Value, opts useoptions) (err error) {
        if optionVerboseUsing && optionVerboseImport && optionBenchImport {
                defer func(t time.Time) {
                        var d = time.Now().Sub(t)
                        fmt.Fprintf(stderr, "%s││ using(%8s) %s ⇒ %v\n", l.vs, d, l.project, l.project.using)
                } (time.Now())
        } else if optionVerboseUsing {
                defer func(t time.Time) {
                        var d = time.Now().Sub(t)
                        fmt.Fprintf(stderr, "using(%8s) %s ⇒ %v\n", d, l.project, l.project.using)
                } (time.Now())
        }
        if err = l.useProject2(pos, usee, params, opts); err != nil {
                if p, ok := err.(*scanner.Error); ok {
                        l.error(pos, "%v", p.Brief())
                } else {
                        l.error(pos, "%v", err)
                }
        }
        return
}

func (l *loader) usePath() (s string) {
        for i, u := range l.useStack {
                if i > 0 { s += "," }
                s += u.name
        }
        return
}

func (l *loader) useProject2(pos token.Pos, usee *Project, params []Value, opts useoptions) (err error) {
        position := l.parser.file.Position(pos)
        if usee == l.project {
                err = scanner.Errorf(position, "'%v' use loop (%s)", usee.name, l.usePath())
                return
        } else if false {
                for _, using := range l.project.using.list {
                        if using.project == usee { return }
                }
        } else if l.project.isUsingDirectly(usee) {
                return
        }

        // clocks:🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseImport {
                        if optionBenchImport /*&& d > 1*time.Millisecond*/ {
                                var s = l.usePath()
                                fmt.Fprintf(stderr, "%s││ %s:use(%s) … (%s) (%s)\n", l.vs, l.project.name, usee.name, d, s)
                        }
                } else if optionBenchSlow && d > 500*time.Millisecond { // ⌚ ⌛
                        fmt.Fprintf(stderr, "smart: %s: slow ▶use(%s)◀ … (%s)\n", l.project.name, usee.name, d)
                }
        } (time.Now())

        defer func(a []*Project) { l.useStack = a } (l.useStack)
        l.useStack = append(l.useStack, usee) // build the use path

        // Add to the project using list, so that the use path is correct.
        l.project.using.append(Position(position), usee, params, opts)

        return // :user: rules are deprecated!
}

func (l *loader) determine(pos token.Pos, tok token.Token, identifier, value Value) (def *Def) {
        var alt Object
        switch t := identifier.(type) {
        case *selection:
                var v, err = t.value()
                if err != nil {
                        l.error(pos, "determine `%v`: %v", t, err)
                        return
                } else if d, ok := v.(*Def); ok {
                        def = d
                } else {
                        l.error(pos, "`%v` is not a def (%T)", t, v)
                        return
                }

        case *Bareword, *Barecomp, *Qualiword:
                var name, err = t.Strval()
                if err != nil {
                        l.error(pos, "determine `%v`: %v", t, err)
                        return
                } else if _, ok := builtins[name]; ok {
                        l.error(pos, "`%v` (%v) is builtin name", identifier, name)
                        return
                }

                // Resolve base value to derive.
                var prev Object
                prev, err = l.project.resolveObject(name)
                if err != nil { l.error(pos, "%v", err) }
                if def, alt = l.def(pos, name); alt == nil {
                        // does nothing...
                } else if alt != nil && (tok == token.ASSIGN || tok == token.EXC_ASSIGN) {
                        var ( okay bool; ad *Def )
                        if ad, okay = alt.(*Def); !okay {
                                l.error(pos, "`%v` already defined (%T) (%v,%v)", identifier, alt, alt.OwnerProject(), l.project)
                                if optionPrintStack {
                                        fmt.Fprintf(stderr, "%s: `%v` already defined here\n", alt.Position(), alt)
                                        debug.PrintStack()
                                }
                                return
                        } else if ad.owner == l.project && ad.origin != DefConfRef {
                                l.error(pos, "`%v` already defined (%T) (%v)", identifier, alt, l.project)
                                if optionPrintStack {
                                        fmt.Fprintf(stderr, "%s: `%v` already defined here\n", alt.Position(), alt)
                                        debug.PrintStack()
                                }
                                return
                        } else {
                                def = ad
                        }
                } else if alt != nil { def = alt.(*Def) }

                if prev == nil {
                        // no derived value
                } else if prev.OwnerProject() == l.project {
                        // don't derive in the same project
                } else if derived, okay := prev.(*Def); !okay {
                        // not a def
                } else if derived == nil {
                        assert(false, "encounterred nil def `%s`", name)
                } else if derived == def || def.value.refs(derived) {
                        // same def
                } else if tok == token.ADD_ASSIGN {
                        // Unshift the delegation to derive value.
                        position := Position(l.parser.file.Position(pos))
                        err := def.append(MakeDelegate(position, token.LPAREN, derived))
                        if err != nil {
                                l.error(pos, "%v", err)
                        }
                }
        }

        if def == nil {
                l.error(pos, "identifier `%v' is nil", identifier)
                return
        }

        // Ensures that all immediate assignments are in the current
        // project context.
        defer setclosure(setclosure(cloctx.unshift(l.scope)))

        if err := l.assign(pos, tok, def, alt, value); err != nil {
                l.error(pos, "%v", err)
        }
        return
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
                case Caller: res = op.Call(position, l.exprs(spec.Props[1:])...)
                default:
                        var ( str string; err error )
                        if str, err = op.Strval(); err != nil {
                                l.error(id.Pos(), "%s: %v", op, err)
                        } else if _, obj := l.scope.Find(str); obj == nil {
                                l.error(id.Pos(), "`%s` undefined", str)
                        } else if f, _ := obj.(Caller); f == nil {
                                l.error(id.Pos(), "`%T` is not caller (%s)", obj, str)
                        } else {
                                res = f.Call(position, l.exprs(spec.Props[1:])...)
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

func (l *loader) rule(clause *ast.RuleClause, special specialRule, options []ast.Expr) (entries []*RuleEntry) {
        defer setclosure(setclosure(cloctx.unshift(l.project.scope)))

        var (
                depends []Value
                ordered []Value
                recipes []Value
                params  []*Def
                progScope *Scope
        )
        for _, depend := range clause.Depends {
                switch dep := l.expr(depend).(type) {
                case *List: depends = append(depends, dep.Elems...)
                default:    depends = append(depends, dep)
                }
        }
        for _, depend := range clause.Ordered {
                switch dep := l.expr(depend).(type) {
                case *List: ordered = append(ordered, dep.Elems...)
                default:    ordered = append(ordered, dep)
                }
        }

        var configure bool
        if p, ok := clause.Program.(*ast.ProgramExpr); ok && p != nil {
                if progScope, _ = p.Scope.(*Scope); progScope == nil {
                        l.error(clause.Pos(), "undefined program scope (%T).", p.Scope)
                }
                if p.Recipes != nil { recipes = l.exprs(p.Recipes) }
                for _, name := range p.Params {
                        def := progScope.Lookup(name).(*Def)
                        params = append(params, def)
                }
                configure = p.Configure
        } else {
                l.error(clause.Program.Pos(), "unsupported program type (%T)", clause.Program)
                return
        }
        
        var prog = &Program{
                project:  l.project,
                scope:    progScope,
                params:   params,
                depends:  depends,
                ordered:  ordered,
                recipes:  recipes,
                configure: configure,
                position: Position(clause.Position),
        }

        var optionVals = l.exprs(options)
        for n, target := range l.exprs(clause.Targets) {
                if target == nil {
                        l.error(clause.Targets[n].Pos(), "nil target (%T)", clause.Targets[n])
                        return
                }
                var ( name string ; entry *RuleEntry ; err error )
                if name, err = target.Strval(); err != nil {
                        l.error(clause.Targets[n].Pos(), "%v", err)
                }
                if true {// it should work too if not checking against files
                        switch target.(type) {
                        case *File, *Path, Pattern:
                        default:
                                var file = l.project.matchFile(name)
                                if file != nil {
                                        file.position = target.Position()
                                        target = file
                                }
                        }
                }
                entry, err = l.project.entry(special, optionVals, target, prog)
                if err != nil {
                        l.error(clause.Targets[n].Pos(), "%v", err)
                        return
                } else /*if entry != nil*/ {
                        entry.position = Position(l.parser.file.Position(clause.Targets[n].Pos()))
                        entries = append(entries, entry)
                }
                if t, okay := entry.target.(*Flag); okay && t != nil {
                        var s string
                        if s, err = t.name.Strval(); err != nil {
                                l.error(clause.Targets[n].Pos(), "%v", err)
                        } else if l.project.name != "~" {
                                flags, _ := context.flagEntries[s]
                                flags = append(flags, entry)
                                context.flagEntries[s] = flags
                        }
                        //if s == "configure" { configuration.configs = append(configuration.configs, entry) }
                } else if configure {
                        configuration.entries = append(configuration.entries, entry)
                }
        }
        return
}

func (l *loader) include(spec *ast.IncludeSpec) {
        if len(spec.Props) > 0 {
                var prop = spec.Props[0]
                l.includeFile(prop.Pos(), l.expr(prop))
        }
}

func (l *loader) includeFile(_pos token.Pos, spec Value) {
        var (
                pos = Position(l.file.Position(_pos))
                linfo = l.loads[len(l.loads)-1]
                specName, fullname string
                err error
        )

        // Execute the rule entry to update include source.
        if entry, ok := spec.(*RuleEntry); ok && entry != nil {
                var ( result []Value; breakers []*breaker )
                if result, breakers = entry.Execute(entry.position); len(breakers) > 0 {
                        diag.errorAt(pos, "include error occurred (entry %v)", entry)
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
                        diag.errorAt(pos, "`%v` no source file", t)
                        return
                }
                fullname = t.fullname() //filepath.Join(t.dir, t.Name)
                specName = t.name
        default:
                if specName, err = spec.Strval(); err != nil {
                        diag.errorAt(pos, "include error occurred (spec %v)", spec)
                        return
                }
                if filepath.IsAbs(specName) {
                        fullname = specName
                } else {
                        fullname = filepath.Join(linfo.absDir, specName)
                }
        }

        if specName == "" {
                diag.errorAt(pos, "`%v` is empty string", spec)
                return
        }

        var mode = l.tracemode
        var absDir, baseName = filepath.Split(fullname)
        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))
        if _, err = l.ParseFile(fullname, nil, parseMode|Flat); err != nil {
                diag.errorAt(pos, "include error occurred (from %v)", fullname)
        } else {
                // The parse mode could still be 'Flat' here as ParseFile
                // changed it, so we have to restore the previous parse mode.
                l.tracemode = mode
        }
        return
}

func (l *loader) openScope(comment string) loaderScope {
        if false && optionTraceLaunch { defer un(trace(t_launch, "loader.openScope")) }
        var pos Position
        if l.parser != nil {
                pos = Position(l.file.Position(l.pos))
        }
        l.scope = NewScope(pos, l.scope, l.project, comment)
        cc := setclosure(cloctx.unshift(l.scope))
        return loaderScope{ cc, l.scope }
}

func (l *loader) closeScope(ls loaderScope) {
        if false && optionTraceLaunch { defer un(trace(t_launch, "loader.closeScope")) }
        if ls.scope != nil {
                l.scope = ls.scope.outer
                if ls.cc != nil { setclosure(ls.cc) }

                // Must change the outer of dir scope to globe to avoid Finding symbols
                // recursively.
                if s := ls.scope.Comment(); strings.HasPrefix(s, "dir ") /*|| strings.HasPrefix(s, "file ")*/ {
                        l.globe.SetScopeOuter(ls.scope)
                }
        }
        return
}

func (l *loader) setArgs(args []Value) (oldArgs []Value) {
        oldArgs = l.loadArgs
        l.loadArgs = args
        return
}

// project example (base(var=value))
func (l *loader) loadBases(linfo *loadinfo, params []Value) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadBases")) }

        var isDir bool
        var absPath, specName string

        absPath = filepath.Join(l.project.absPath, ".base")
        if fi, e := os.Stat(absPath); e == nil && fi.IsDir() {
                base := MakePathStr(l.project.position, "./.base")
                params = append([]Value{base}, params...)
                absPath = ""
        }

        // For &(foobar) set from loadArgs
        defer setclosure(setclosure(cloctx.unshift(l.project.scope)))

        ParamsLoop: for _, elem := range params {
                var args []Value
                if list, ok := elem.(*List); ok && len(list.Elems) == 1 { elem = list.Elems[0] }
                if a, ok := elem.(*Argumented); ok { elem, args = a.value, a.args }
                if p, ok := elem.(*Pair); ok {
                        var identifier = p.Key
                        var pos = identifier.Position()
                        var name string
                        if name, err = p.Key.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if len(name) > 0 && name[0] == '.' { identifier = MakeBarecomp(pos, &Bareword{trivial{pos},"project"}, p.Key) }
                        def := l.determine(l.pos, token.ASSIGN, identifier, p.Value)
                        if def == nil {/* FIXME: ... */}
                        continue ParamsLoop
                }

                if specName, err = elem.Strval(); err != nil { return }
                if specName == "" {
                        diag.errorOf(elem, "%v: empty base name `%v` (%T)", l.project, elem, elem)
                        break ParamsLoop
                }
                if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
                        break ParamsLoop
                }

                for _, base := range l.project.bases {
                        if base.absPath == absPath {
                                //diag.errorOf(elem, "duplicated base: %v (in %v)", elem, l.project.bases)
                                continue ParamsLoop
                        }
                }

                if isDir {
                        err = l.loadDirWithArgs(specName, absPath, args, nil)
                } else {
                        err = l.loadWithArgs(specName, absPath, args, nil)
                }

                if err != nil {
                        diag.errorAt(elem.Position(), "%v", err)
                        break ParamsLoop
                }

                // chain loaded base project, note that err might not be nil
                if loaded, yes := l.loaded[absPath]; yes && loaded != nil {
                        l.project.Chain(loaded)
                } else {
                        diag.errorOf(elem, "project `%v`(%T: %s) not loaded (%s)", elem, elem, specName, absPath)
                        break ParamsLoop
                }
        }
        return
}

func (l *loader) loadDotContainer(ident *ast.Bareword, file *File) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadDotContainer")) }
        if err = l.loadDir(dotContainer, file.fullname(), nil); err != nil {
                l.error(ident.Pos(), "%s: errors occured while loading: %v", file.fullname(), dotContainer)
        } else if loaded, yes := l.loaded[file.fullname()]; yes && loaded != nil {
                name, _ := l.project.scope.Lookup(loaded.Name()).(*ProjectName)
                if name == nil {
                        l.error(ident.Pos(), "%v: %v: `dock` is not a project", l.project.name, file)
                } else {
                        if optionVerboseLoading { fmt.Fprintf(stderr, "smart: %v (%v)\n", name, file.fullname()) }

                        var opts useoptions
                        // TODO: parse the useoptions
                        l.useProject(ident.Pos(), loaded, nil, opts)
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
                        if s, err = t.name.Strval(); err != nil { return }
                        switch s {
                        case "final": optFinal = true;
                        case "nodock": optNoDock = true;
                        } 
                default:
                        bases = append(bases, t)
                }
        }

        var pos = Position(l.parser.file.Position(ident.Pos()))
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
        } else if _, o := l.scope.Find(ident.Value); o != nil {
                if _, ok := o.(*Builtin); ok {
                        diag.errorAt(pos, "project name '%s' is a builtin name", ident.Value)
                        return
                }
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
                        outer = outer.outer
                }

                dec = new(declare)
                dec.project = l.globe.project(pos, outer, absDir, relPath, tmpPath, linfo.specName, name)
                l.loaded[linfo.absPath()] = dec.project
                linfo.declares[name] = dec
        }
        if loader := linfo.loader; loader != nil {
                if !strings.HasPrefix(loader.scope.comment, "project \"") {
                        l.parser.warn(ident.Pos(), "'%s' not loaded from project scope", name)
                }
                n, a := loader.scope.ProjectName(loader, name, dec.project)
                if a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                err = fmt.Errorf("`%s` name already taken (%T).", name, a)
                                return
                        } else {
                                n = v
                        }
                }
                if n != nil {
                }
        }

        if _, err = parseFlags(options, []string{
                "b,break",
                "l,loop",
                "m,multi",
        }, func(ru rune, v Value) {
                switch ru {
                case 'b', 'l':  if dec.project.breakUseLoop     , err = trueVal(v, false); err != nil { return }
                case 'm':       if dec.project.multiUseAllowed  , err = trueVal(v, false); err != nil { return }
                }
        }); err != nil { return }

        dec.backproj = l.project
        dec.backscope = l.scope
        l.useesExecuted = nil
        l.project = dec.project
        l.scope = l.project.scope
        if l.globe.main != nil && l.globe.main == l.project && l.project.name != "~" {
                for _, t := range context.pairs {
                        switch k := t.Key.(type) {
                        case *Bareword, *Barecomp:
                                var name string
                                if name, err = k.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                                //if name[0] == '.' { name = "project" + name }
                                var def, alt = l.def(l.pos, name)
                                if def == nil && alt != nil { def = alt.(*Def) }
                                def.set(DefDecl, t.Value)
                        default:
                                diag.errorAt(pos, "`%v` unknown target from command line (%v)\n", t, l.project)
                                return
                        }
                }
        }

        for _, arg := range merge(l.loadArgs...) {
                switch t := arg.(type) {
                case *Pair:
                        var name string
                        name, err = t.Key.Strval()
                        if err != nil { return }

                        var def, alt = l.def(l.pos, name)
                        if alt != nil {
                                var ok bool
                                def, ok = alt.(*Def)
                                if !ok {
                                        diag.errorAt(pos, "'%v' is not a Def (%T)", alt, alt)
                                        return
                                }
                        }
                        if def != nil { def.setval(t.Value) }
                }
        }

        if err = l.loadPlugin(pos); err != nil { return }

        // no bases or docking for external packages
        if keyword == token.PACKAGE { return }
        if !optFinal {
                if err = l.loadBases(linfo, bases); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                }
        }

        // FIXES: set cloctx immediately to ensure the right configuration
        //        is matched!
        defer setclosure(setclosure(cloctx.unshift(l.scope)))

        if s, err := configurationFileName(l.project); err != nil { return err } else
        if declared || optionConfigure {
                var exists bool
                for _, v := range configuration.clean { if s == v { exists = true; break }}
                if !exists { configuration.clean = append(configuration.clean, s) }
        } else if file := stat(pos, filepath.Base(s), "", filepath.Dir(s)); file != nil {
                if optionVerboseImport || optionVerboseLoading {
                        full, _ := file.Strval()
                        fmt.Fprintf(stderr, "smart: Configuration for %s (%s) ⇒ %s\n", l.project, l.project.spec, full)
                } else if optionVerbose || true {
                        fmt.Fprintf(stderr, "smart: Configuration for %s (%s)\n", l.project, l.project.spec)
                }
                l.isIncludingConf = true
                l.includeFile(ident.Pos(), file)
                l.isIncludingConf = false
        }

        if optNoDock || l.project.name == dotContainer { return }
        if _, e := os.Stat(".dock"); e == nil {
                diag.errorAt(pos, "Must rename .dock into .container !")
                return
        }

        // Looking for project specific dotContainer module
        file := stat(pos, dotContainer, "", l.project.absPath)
        if exists(file) { err = l.loadDotContainer(ident, file); return }

        // Looking for .smart/.container
        walkSmartBaseDirs(l.project.absPath, func(s string) bool {
                file := stat(pos, dotContainer, "", filepath.Join(s, ".smart"))
                if !exists(file) {
                        // no docking enabled
                } else if err = l.loadDotContainer(ident, file); err != nil {
                        // FIXME: error...
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
        var pos = Position(l.parser.file.Position(l.parser.pos))
        var scope = NewScope(pos, outer, l.project, comment)
        if strings.HasPrefix(outer.Comment(), "dir ") {
                outer = outer.outer // discard dir scope
        }

        outer.ScopeName(l.project, name, scope)

        ls := loaderScope{ setclosure(cloctx.unshift(scope)), scope }
        l.scope = ls.scope
        return ls, nil
}

func (l *loader) resolve(value Value) (obj Value, err error) {
  if sel, ok := value.(*selection); ok {
    obj = sel
    return
  }

  var name string
  if name, err = value.Strval(); err != nil { return }


  if l.scope != nil { _, obj = l.scope.Find(name) }
  if obj == nil && l.project != nil {
    obj, err = l.project.resolveObject(name)
  }

  /*if obj == nil {
    diag.errorOf(value, "`%s` is nil (%T)", name, value)
  }*/
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

func (l *loader) def(pos token.Pos, name string) (def *Def, alt Object) {
        var scope = l.scope
        if strings.HasPrefix(scope.comment, "file ") && l.tracemode&Flat != 0 {
                // use project scope if defining in flat file (aka. include)
                // to ensure that the symbol is valid in the project
                scope = l.project.scope
        }
        var position = Position(l.parser.file.Position(pos))
        def, alt = scope.define(l.project, name, &None{trivial{position}})
        if def != nil { def.position = position }
        return
}

func (l *loader) assign(pos token.Pos, tok token.Token, def *Def, alt Object, value Value) (err error) {
        def.position = Position(l.parser.file.Position(pos))
        switch tok {
        case token.ASSIGN: // =
                err = def.set(DefDefault, value)
        case token.EXC_ASSIGN: // !=
                err = def.set(DefExecute, value)
        case token.ADD_ASSIGN: // +=
                if value ==  nil {
                        // NOOP
                } else if def.value == nil || !def.value.refs(value) {
                        err = def.append(value)
                }
        case token.SHI_ASSIGN: // =+
                if !def.value.refs(value) {
                        var tail = def.value
                        if err = def.set(DefDefault, value); err == nil {
                                err = def.append(tail)
                        }
                }
        case token.SUB_ASSIGN: // -=
                if def.value == nil {
                        // ...
                } else if _, ok := def.value.(*None); ok {
                        var vals []Value
                        for _, val := range merge(def.value) {
                                if val.cmp(value) != cmpEqual {
                                        vals = append(vals, val)
                                }
                        }
                        def.value = &List{elements{vals}}
                }
        case token.SAD_ASSIGN: // -+=
                var vals []Value
                if def.value == nil {
                        // ...
                } else if _, ok := def.value.(*None); ok {
                        for _, val := range merge(def.value) {
                                if val.cmp(value) != cmpEqual || true {
                                        vals = append(vals, val)
                                }
                        }
                }
                vals = append(vals, value)
                def.value = &List{elements{vals}}
        case token.SSH_ASSIGN: // -=+
                var vals = []Value{ value }
                if def.value == nil {
                        // ...
                } else if _, ok := def.value.(*None); ok {
                        for _, val := range merge(def.value) {
                                if val.cmp(value) != cmpEqual {
                                        vals = append(vals, val)
                                }
                        }
                }
                def.value = &List{elements{vals}}
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
        if optionTraceLaunch { defer un(trace(t_launch, "loader.ParseFile")) }

        var text []byte
	if text, err = readSource(filename, src); err != nil { return }

	l.tracemode = mode
        if optionTraceParsing {
                l.tracemode |= Trace
        }

	l.tracing.enabled = l.tracemode&Trace != 0 // for convenience (l.trace is used frequently)
	defer func(saved *parser) {
		for e := recover(); e != nil; e = recover() {
                        if _, ok := e.(bailout); ok { continue }

                        var pos Position
                        if l.parser != nil && l.parser.file != nil {
                                pos = Position(l.parser.file.Position(l.pos))
                        } else {
                                pos.Filename = filename
                        }

                        var er, ok = e.(error)
                        if !ok { er = fmt.Errorf("%v", e) }
                        diag.errorAt(pos, "%v", er)
		}
                if err != nil && optionPrintStack {
                        fmt.Fprintf(stderr, "%v\n", err)
                        debug.PrintStack()
                }

                if len(l.errors) > 0 {
                        // var e = new(scanner.Error)
                        // e.Pos = l.parser.file.Position(l.pos)
                        // for _, p := range l.errors { e.Merge(p) }
                        // err = e
                        var p = Position(l.parser.file.Position(l.pos))
                        for _, e := range l.errors { diag.errorAt(p, "%v", e) }
                }

                l.parser.loader = nil
                l.parser = saved
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
                f = &ast.File{ Name:new(ast.Bareword), Scope:ls.scope }
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
                if strings.HasPrefix(name, "~") || 
                   strings.HasSuffix(name, ".#") || 
                   strings.HasSuffix(name, ".smart") ||
                   strings.HasSuffix(name, ".sm") { continue ListLoop }

                var fullname = filepath.Join(linked, name)
                if d.Mode()&os.ModeSymlink != 0 {
                        var ( l string; t os.FileInfo )
                        if l, err = os.Readlink(fullname); err != nil { continue ListLoop }
                        if !filepath.IsAbs(l) { l = filepath.Join(linked, l) }
                        if t, err = os.Stat(l); err != nil { continue ListLoop }
                        if t.IsDir() { continue ListLoop }
                }

                var pos = Position(l.parser.file.Position(l.pos))
                if d.IsDir() {
                        if err = l.ParseConfigDir(filepath.Join(pathname, name), fullname); err != nil { break ListLoop }
                } else if s, a := l.def(l.pos, name); a != nil {
                        diag.errorAt(pos, "declare project: %v", err)
                        break ListLoop
                } else if def = s; def != nil {
                        var ( v []byte; s string )
                        if v, err = ioutil.ReadFile(fullname); err != nil { break ListLoop }
                        if s = string(v); !utf8.ValidString(s) {
                                diag.errorAt(pos, "%s: invalid UTF8 content", fullname)
                                break ListLoop
                        }
                        def.set(DefConfDir, &String{trivial{pos},s})
                } else if s != nil {
                        diag.errorAt(pos, "Name `%s' already taken, not def (%T).", name, s)
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
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseParsing /*&& d > 50*time.Millisecond*/ {
                        fmt.Fprintf(stderr, "parse(%15s) %s ⇒ %s\n", d, l.project, path)
                } else if optionBenchSlow && l.project == nil && d>5000*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: slow ▶parse(%s)◀ … (%s)\n", path, d)
                } else if optionBenchSlow && l.project != nil && d>2500*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: %s: slow ▶parse(%s)◀ … (%s)\n", l.project, path, d)
                }
        } (time.Now())

	fd, err := os.Open(path)
	if err != nil { return nil, err }
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil { return nil, err }

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

                var name = d.Name()
                if name != "" {
                        var skip = strings.HasPrefix(name, ".#")
                        skip = skip || !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm"))
                        if skip { continue ListLoop }
                }
                /*if (name == "configure.smart" || name == "configure.sm") && (linked != "" || mo.IsDir()) {
                        //hasConfDir = true // TODO: remove ConfigDir feature
                        if err := l.ParseConfigDir(filepath.Dir(filename), linked); err != nil {
                                if first == nil {
                                        first = err
                                }
                                return
                        }
                        continue ListLoop
                }*/
                if linked != "" { }

		if mo.IsRegular() && (filter == nil || filter(d)) {
			if src, err := l.ParseFile(filename, nil, mode|parsingDir); err == nil {
                                var pos Position
                                if l.parser != nil && l.parser.file != nil {
                                        pos = Position(l.parser.file.Position(l.pos))
                                } else {
                                        pos.Filename = filename
                                }
                                if src == nil {
                                        diag.errorAt(pos, "'%v' not loaded", filename)
                                        return
                                } else if src.Name == nil {
                                        diag.errorAt(pos, "module '%v' has no name", filename)
                                        return
                                }

				name := src.Name.Value
				mod, found := mods[name]
				if !found {
					mod = &ast.Project{
                                                Name:  name,
                                                Scope: ls.scope,
                                                Files: make(map[string]*ast.File),
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
        if optionTraceLaunch { defer un(trace(t_launch, "loader.load")) }
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseLoading /*&& d > 50*time.Millisecond*/ {
                        loaded, _ := l.loaded[absPath]
                        if l.project == nil {
                                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
                        } else {
                                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
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
        if loaded, yes := l.loaded[absPath]; yes {
                _, a := l.project.scope.ProjectName(l.project, loaded.Name(), loaded)
                if a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                err =  fmt.Errorf("`%v` name already taken (%T).", loaded, a)
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
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadDir")) }
        defer func(t time.Time) {
                var d = time.Now().Sub(t)
                if optionVerboseLoading /*&& d > 50*time.Millisecond*/ {
                        loaded, _ := l.loaded[absDir]
                        if l.project == nil {
                                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
                        } else {
                                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
                        }
                } else if optionBenchSlow && l.project == nil && d>5000*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: slow ▶load(%s)◀ … (%s)\n", specName, d)
                } else if optionBenchSlow && l.project != nil && d>2500*time.Millisecond {
                        fmt.Fprintf(stderr, "smart: %s: slow ▶load(%s)◀ … (%s)\n", l.project.name, specName, d)
                }
        } (time.Now())

        if !filepath.IsAbs(absDir) {
                err = fmt.Errorf("Invalid abs name `%s' (%s).", absDir, specName)
                return
        }

        // Check already loaded project.
        if loaded, valid := l.loaded[absDir]; valid {
                //fmt.Fprintf(stderr, "%s: %v %s\n", l.project, loaded, absDir)
                _, a := l.project.scope.ProjectName(l.project, loaded.Name(), loaded)
                if a != nil { if v, ok := a.(*ProjectName); !ok || v == nil {
                        err = fmt.Errorf("`%s' name already taken (%T).", loaded.Name(), a)
                }}
                return
        }

        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

        var mods map[string]*ast.Project
        // FIXME: loading failed if different 'project' found in
        // the same dir, for example:
        //      project Foo # file build.smart
        //      project # file config.smart
        mods, err = l.ParseDir(absDir, filter, parseMode)
        if err == nil && mods == nil && filepath.Base(specName) != "@" {
                err = fmt.Errorf("`%s` invalid project", specName)
        }
        if loaded, valid := l.loaded[absDir]; valid && loaded != nil {
                // Good!
                //a := l.project.scope.Lookup(loaded.Name())
                //fmt.Fprintf(stderr, "%s: %v %s\n", l.project, loaded, a)
        } else if filepath.Base(specName) != "@" {
                err = fmt.Errorf("`%s` not loaded (%s)", specName, absDir)
        }
        return
}

func (l *loader) loadWithArgs(specName, absPath string, args []Value, source interface{}) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadWithArgs")) }
        defer l.setArgs(l.setArgs(args))
        return l.load(specName, absPath, source)
}

func (l *loader) loadDirWithArgs(specName, absPath string, args []Value, filter func(os.FileInfo) bool) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadDirWithArgs")) }
        defer l.setArgs(l.setArgs(args))
        return l.loadDir(specName, absPath, filter)
}

func (l *loader) loadFile(filename string, source interface{}) error {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadFile")) }
        s, _ := filepath.Split(filename)
        s, _  = filepath.Rel(l.workdir, s)
        return l.load(s, filename, source)
}

func (l *loader) loadPath(path string, filter func(os.FileInfo) bool) (err error) {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadPath")) }
        s, _ := filepath.Rel(l.workdir, path)
        err = l.loadDir(s, path, filter)
        return err
}

func (l *loader) loadText(filename string, text string) []Value {
        if optionTraceLaunch { defer un(trace(t_launch, "loader.loadText")) }

	defer func(saved *parser) {
                l.parser.loader = nil
                l.parser = saved
	} (l.parser)

        if l.globe.main == nil {
                l.project = l.globe.os
        } else {
                l.project = l.globe.main
        }
        l.scope = l.project.scope
        l.useesExecuted = nil

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
