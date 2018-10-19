//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
        "extbit.io/smart/ast"
        "extbit.io/smart/token"
	"bytes"
	"io/ioutil"
	"io"
        "unicode/utf8"
	"path/filepath"
	"strings"
        "errors"
        "flag"
        "fmt"
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

        // Wants v.Strval(), expends delegates and closures,
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
}

type loadinfo struct {
        absDir string // absPath = filepath.Join(absDir, baseName)
        baseName string
        specName string
        loader *Project
        scope *Scope
        declares map[string]*declare // all project declares in the loaded dir
}

func (li *loadinfo) absPath() string {
        return filepath.Join(li.absDir, li.baseName)
}

type loader struct {
        *Context
        tracing // Tracing/debugging
        p        *parser
        fset     *token.FileSet
        paths    searchlist
        loads    []*loadinfo
        loaded   map[string]*Project // loaded projects
        project  *Project // the current project
        scope    *Scope   // the current scope
}

func (l *loader) errat(pos token.Pos, err interface{}, a... interface{}) {
        if l.p != nil {
                l.errorAt(l.p.file.Position(pos), err, a...)
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
        l.project = linfo.loader
        l.scope = linfo.scope //l.SetScope(linfo.scope)

        var names []string
        for _, declare := range linfo.declares {
                names = append(names, declare.project.Name())
        }

        /*
        if loader := linfo.loader; loader != nil {
                fmt.Printf("exit: %v from '%s' -> %v\n", names, loader.Name(), linfo.scope)
        } else {
                fmt.Printf("exit: %v -> %v\n", names, linfo.scope)
        } */
}

func saveLoadingInfo(l *loader, specName, absDir, baseName string) *loader {
        //absDir, baseName := filepath.Split(filepath.Clean(absPath))
        l.loads = append(l.loads, &loadinfo{
                absDir: absDir,
                baseName: baseName,
                specName: filepath.Clean(specName),
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

func (l *loader) loadImportSpec(spec *ast.ImportSpec) {
        var (
                linfo = l.loads[len(l.loads)-1]
                specName, s string
                params []Value
                useList []Value
                nouse bool
                err error
        )
        if 0 < len(spec.Props) {
                if ee, ok := spec.Props[0].(*ast.EvaluatedExpr); ok && ee.Data != nil {
                        if specName, err = ee.Data.(Value).Strval(); err != nil {
                                l.p.error(spec.Props[0].Pos(), "%s", err)
                                return
                        }
                } else {
                        l.p.error(spec.Props[0].Pos(), "invalid evaluated data `%T`")
                        return
                }
                for _, prop := range spec.Props[1:] {
                        ee, _ := prop.(*ast.EvaluatedExpr)
                        if ee == nil || ee.Data == nil {
                                l.p.error(prop.Pos(), "invalid evaluated data `%T`")
                                return
                        }
                        // -param
                        // -param(value)
                        // -param=value
                        switch v := ee.Data.(Value); t := v.(type) {
                        case *Flag:
                                if s, err = t.Name.Strval(); err != nil {
                                        l.p.error(prop.Pos(), "invalid flag `%v` (%v)", v, err)
                                        return
                                }
                                switch s {
                                case "nouse": nouse = true
                                default: params = append(params, v)
                                }
                        case *Pair: // -param=value
                                switch tt := t.Key.(type) {
                                case *Flag:
                                        if s, err = tt.Name.Strval(); err != nil {
                                                l.p.error(prop.Pos(), "invalid flag name `%v` (%v)", tt.Name, err)
                                                return
                                        }
                                        switch s {
                                        case "use": useList = append(useList, t.Value)
                                        default: params = append(params, v)
                                        }
                                default:
                                        l.p.error(prop.Pos(), "parameter `%v' unsupported `%T`", v, v)
                                        return
                                }
                        case *Argumented: // -param(value)
                                switch tt := t.Value.(type) {
                                case *Flag:
                                        if s, err = tt.Name.Strval(); err != nil {
                                                l.p.error(prop.Pos(), "invalid flag name `%v` (%v)", tt.Name, err)
                                                return
                                        }
                                        switch s {
                                        case "use": useList = append(useList, t.Args...)
                                        default: params = append(params, v)
                                        }
                                default:
                                        l.p.error(prop.Pos(), "parameter `%v' unsupported `%T`", v, v)
                                        return
                                }
                        default:
                                l.p.error(prop.Pos(), "parameter `%v` unsupported `%T`", v, v)
                                return
                        }
                }
        }
        if specName == "" {
                l.p.error(spec.Pos(), "invalid spec `%v`", spec)
                return
        }

        var (
                absPath string
                isDir bool
        )
        if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
                l.p.error(spec.Pos(), "no such package `%v`", specName)
                return
        } else if absPath == "" {
                l.p.error(spec.Pos(), "missing `%s` (in %v)", specName, l.paths)
                return
        }

        if isDir {
                err = l.loadDir(specName, absPath, nil)
        } else {
                err = l.load(specName, absPath, nil)
        }
        if err != nil || nouse {
                l.p.error(spec.Pos(), "import `%v` (%v): %v", specName, absPath, err)
                return
        }

        if loaded, _ := l.loaded[absPath]; loaded != nil {
                scope := l.project.Scope()
                pn, _ := scope.Lookup(loaded.Name()).(*ProjectName)
                if pn == nil {
                        l.p.error(spec.Pos(), "%v (%v,dir=%v) not in %v", specName, absPath, isDir, scope.comment)
                        return
                }
                // Add loaded project to the use list ('$(use->*)')
                if sn, _ := scope.Lookup("use").(*ScopeName); sn != nil {
                        if alt := sn.NamedScope().Insert(pn); alt != nil {
                                l.p.error(spec.Pos(), "use: '%s' already defined in %v", specName, sn.DeclScope())
                                return
                        }
                        if _, alt := sn.DeclScope().Def(l.project, "*"/*use list*/, pn); alt != nil {
                                if def, _ := alt.(*Def); def != nil {
                                        // If there's no explicit use list, we just add
                                        // the ProjectName pn, so that the default entry
                                        // will be executed. Or the specified use list
                                        // will be used.
                                        if len(useList) == 0 {
                                                def.Append(pn)
                                        } else {
                                                for _, usee := range useList {
                                                        // TODO: find usee in pn scope
                                                        if usee == nil {}
                                                }
                                        }
                                }
                        }
                }
                l.useProject(spec.Props[0].Pos(), loaded)
        } else if false {
                position := l.p.file.Position(spec.Props[0].Pos())
                fmt.Fprintf(os.Stderr, "%v: not loaded %v (%v)\n", position, specName, absPath)
                for k, v := range l.loaded {
                        fmt.Fprintf(os.Stderr, "   loaded: %v (%v)\n", v.Name(), k)
                }
        }
        return
}

func (l *loader) evaluated(x *ast.EvaluatedExpr) (v Value) {
        var ok bool
        if x.Data == nil {
                l.p.error(x.Pos(), "evaluated data is nil `%T`", x.Expr)
        } else if v, ok = x.Data.(Value); !ok {
                l.p.error(x.Pos(), "evaluated data is not value `%T`", x.Data)
        }
        return v
}

func (l *loader) argumented(x *ast.ArgumentedExpr) Value {
        av := new(Argumented)
        av.Value = l.expr(x.X)
        av.Args = l.exprs(x.Arguments)
        return av
}

func (l *loader) closuredelegate(x *ast.ClosureDelegate) (obj Object, args []Value) {
        var name = l.expr(x.Name)
        if name == nil {
                l.p.error(x.Name.Pos(), "nil name `%T`", x.Name)
                return
        }

        var tok = token.ILLEGAL
        switch x.TokLp {
        case token.LPAREN: tok = token.LPAREN
        case token.LBRACE: tok = token.LBRACE
        case token.ILLEGAL:
                if x.Tok.IsClosure() || x.Tok.IsDelegate() {
                        tok = token.LPAREN
                } else {
                        l.p.error(x.TokPos, "unregonized closure/delegate (%v).", x.Tok)
                        return
                }
        default:
                l.p.error(x.TokPos, "unregonized closure/delegate (%v, %v).", x.TokLp, x.Tok)
                return
        }

        if x.Resolved == nil {
                switch tok {
                case token.LPAREN:
                        var err error
                        if x.Resolved, err = l.resolve(name); err != nil {
                                l.p.error(x.Name.Pos(), "%s", err)
                                return
                        } else if x.Resolved == nil {
                                l.p.error(x.Name.Pos(), "object `%s` is nil", name)
                                return
                        }
                case token.LBRACE:
                        if x.Resolved = l.find(name); x.Resolved == nil {
                                l.p.error(x.Name.Pos(), "entry `%s` is nil", name)
                                return
                        }
                }
                if x.Resolved == nil {
                        l.p.error(x.Name.Pos(), "unresolved `%s`", name)
                        return
                }
        }

        switch tok {
        case token.LPAREN:
                if def, _ := x.Resolved.(Caller); def == nil {
                        l.p.error(x.Name.Pos(), "uncallable `%s` resolved `%T`", name, x.Resolved)
                        return
                } else if obj = def.(Object); obj == nil {
                        l.p.error(x.Name.Pos(), "non-object callable `%s` resolved `%T`", name, def)
                        return
                }
        case token.LBRACE:
                if resolved, _ := x.Resolved.(Executer); resolved != nil {
                        if obj = resolved.(Object); obj == nil {
                                l.p.error(x.Name.Pos(), "non-object executer `%s` resolved `%T`", name, resolved)
                                return
                        }
                } else {
                        l.p.error(x.Name.Pos(), "unexecutable `%s` resolved `%T`", name, x.Resolved)
                        return
                }
        }
        
        for i, x := range x.Args {
                if a := l.expr(x); a != nil {
                        args = append(args, a)
                } else {
                        l.p.error(x.Pos(), "nil arg #%d `%T`", i, x)
                        return
                }
        }
        return
}

func (l *loader) closure(x *ast.ClosureExpr) (v Value) {
        if obj, args := l.closuredelegate(&x.ClosureDelegate); obj != nil {
                v = MakeClosure(x.Position, x.TokLp, obj, args...)
        } else {
                l.p.error(x.Pos(), "closure nil object `%T`", x.Name)
        }
        return
}

func (l *loader) delegate(x *ast.DelegateExpr) (v Value) {
        if obj, args := l.closuredelegate(&x.ClosureDelegate); obj != nil {
                v = MakeDelegate(x.Position, x.TokLp, obj, args...)
        } else {
                l.p.error(x.Pos(), "delegate nil object `%T`", x.Name)
        }
        return
}

func (l *loader) selection(x *ast.SelectionExpr) (v Value) {
        if lhs := l.expr(x.Lhs); lhs != nil {
                if lhs.Type() != SelectionType {
                        // Resolve the first left-hand-side.
                        if o, err := l.resolve(lhs); err != nil {
                                l.p.error(x.Lhs.Pos(), "`%v`: %v", lhs, err)
                        } else if o == nil {
                                l.p.error(x.Lhs.Pos(), "`%v` is undefined", lhs)
                        } else {
                                lhs = o
                        }
                }
                if rhs := l.expr(x.Rhs); rhs != nil {
                        v = &selection{ x.Tok, lhs, rhs }
                } else {
                        l.p.error(x.Rhs.Pos(), "invalid %v `%T`", lhs, x.Rhs)
                }
        } else {
                l.p.error(x.Lhs.Pos(), "invalid `%T`", x.Lhs)
        }
        return
}

func (l *loader) barefile(x *ast.Barefile) (v Value) {
        if file, _ := x.File.(*File); file != nil {
                if x.Val != nil {
                        v = MakeBarefile(x.Val.(Value), file)
                } else {
                        v = MakeBarefile(l.expr(x.Name), file)
                }
        }
        if v == nil {
                l.p.error(x.Pos(), "invalid barefile `%s` (%T)", x.Name, x.File)
        }
        return
}

func (l *loader) pathseg(x *ast.PathSegExpr) (v Value) {
        switch x.Tok {
        case token.PCON:   v = MakePathSeg('/')
        case token.TILDE:  v = MakePathSeg('~')
        case token.PERIOD: v = MakePathSeg('.')
        case token.DOTDOT: v = MakePathSeg('^') // 
        default: l.p.error(x.Pos(), "unsupported path segment `%v`", x.Tok)
        }
        return
}

func (l *loader) recipe(x *ast.RecipeExpr) (v Value) {
        if len(x.Elems) == 0 {
                v = UniversalNone
        } else if x.Dialect == "" {
                v = MakeList(l.exprs(x.Elems)...)
        } else {
                v = MakeCompound(l.exprs(x.Elems)...)
        }
        return
}

func (l *loader) recipedefine(x *ast.RecipeDefineClause) (v Value) {
        var name = l.expr(x.Name)
        if def, _ := x.Sym.(*Def); def == nil {
                l.p.error(x.Pos(), "`%s` undefined in %v", def.Name(), def.DeclScope().comment)
        } else if str, err := name.Strval(); err != nil {
                l.p.error(x.Name.Pos(), "%s", err)
        } else if def.Name() != str {
                l.p.error(x.Name.Pos(), "`%s` differs from `%s`", str, def.Name())
        } else if o := def.DeclScope().Lookup(def.Name()); o != def {
                l.p.error(x.Pos(), "`%s` undefined in %v", def.Name(), def.DeclScope().comment)
        } else {
                v = def
        }
        return
}

func (l *loader) expr(expr ast.Expr) (v Value) {
        if expr == nil {
                //l.p.error(l.p.pos, "encountered nil expr")
                v = UniversalNone
                return
        }

        switch x := expr.(type) {
        case *ast.EvaluatedExpr:
                v = l.evaluated(x)
        case *ast.ArgumentedExpr:
                v = l.argumented(x)
        case *ast.ClosureExpr:
                v = l.closure(x)
        case *ast.DelegateExpr:
                v = l.delegate(x)
        case *ast.SelectionExpr:
                v = l.selection(x)
        case *ast.BasicLit:
                v = ParseLiteral(x.Kind, x.Value)
        case *ast.Bareword:
                v = MakeBareword(x.Value)
        case *ast.Barecomp:
                v = MakeBarecomp(l.exprs(x.Elems)...)
        case *ast.Barefile:
                v = l.barefile(x)
        case *ast.GlobExpr: // Just "*"
                v = MakeGlob(x.Tok)
        case *ast.PathExpr:
                v = MakePath(l.exprs(x.Segments)...)
        case *ast.PathSegExpr:
                v = l.pathseg(x)
        case *ast.FlagExpr:
                v = MakeFlag(l.expr(x.Name))
        case *ast.CompoundLit:
                v = MakeCompound(l.exprs(x.Elems)...)
        case *ast.GroupExpr:
                v = MakeGroup(l.exprs(x.Elems)...)
        case *ast.ListExpr:
                v = MakeList(l.exprs(x.Elems)...)
        case *ast.KeyValueExpr:
                v = MakePair(l.expr(x.Key), l.expr(x.Value))
        case *ast.PercExpr:
                v = MakeGlobPattern(l.expr(x.X), l.expr(x.Y))
        case *ast.RecipeExpr:
                v = l.recipe(x)
        case *ast.RecipeDefineClause:
                v = l.recipedefine(x)
        case *ast.BadExpr:
                l.p.error(x.Pos(), "bad expr")
                return
        }

        if v == nil {
                l.p.error(expr.Pos(), "expr value is nil `%T`", expr)
        }
        return
}

func (l *loader) exprs(exprs []ast.Expr) (values []Value) {
        for _, x := range exprs {
                values = append(values, l.expr(x))
        }
        return
}

func (l *loader) useProject(pos token.Pos, usee *Project) {
        var (
                position = l.p.file.Position(pos)
                entry *RuleEntry
                obj Object
        )
        if use := usee.Scope().Lookup("use"); use == nil {
                l.p.error(pos, "`%v` has no 'use' package.", usee.Name())
                return
        } else if sn, ok := use.(*ScopeName); !ok || sn == nil {
                l.p.error(pos, "`%v` has invalid 'use' package (%T).", usee.Name(), use)
                return
        } else if obj = sn.DeclScope().Lookup(":"); obj == nil {
                // The use entry is not defined.
                return
        } else if entry, _ = obj.(*RuleEntry); entry == nil {
                l.p.error(pos, "`%v` has invalid 'use' entry (%T).", usee.Name(), obj)
                return
        }

        results, err := entry.Execute(position)
        if err != nil {
                l.p.error(pos, "%v: %v", usee.Name(), err)
                return
        }

        var defs []*Def
        for _, result := range results {
                switch result.Type() {
                case  ListType:
                        for _, elem := range result.(*List).Elems {
                                if def, ok := elem.(*Def); ok && def != nil {
                                        defs = append(defs, def)
                                }
                        }
                }
        }

        for _, def := range defs {
                newDef, alt := l.project.Scope().Def(l.project, def.Name(), UniversalNone)
                if alt != nil {
                        if d, _ := alt.(*Def); d == nil {
                                l.p.error(pos, "name `%s` already taken in project `%s' (%T).", def.Name(), alt, l.project.Name())
                                return
                        } else {
                                newDef = d
                        }
                }
                if newDef == nil {
                        l.p.error(pos, "cannot define `%s' in project `%s'.", def.Name(), l.project.Name())
                        return
                }

                // Append the delegate if not referenced yet.
                if !Refs(newDef, def) {
                        newDef.Append(MakeDelegate(position, token.LPAREN, def))
                }
        }
        return
}

func (l *loader) useProjectName(pos token.Pos, pn *ProjectName) {
        var (
                scope = l.project.Scope()
                project = pn.OwnerProject()
                //useList []Value
        )
        if project == nil {
                l.p.error(pos, "%v is nil", pn)
                return
        }
        
        // FIXME: defined used project in represented order
        if sn, _ := scope.Lookup("use").(*ScopeName); sn != nil {
                l.p.error(pos, "`use` is not in %s", scope.comment)
        } else {
                if alt := sn.DeclScope().Insert(pn); alt != nil {
                        if alt.Type().Kind() == ProjectNameKind {
                                l.p.info(pos, "'%s' already used", pn.Name())
                        } else {
                                l.p.error(pos, "'%s' already defined in %s", pn.Name(), sn.DeclScope())
                                return
                        }
                }
                if _, alt := sn.DeclScope().Def(l.project, "*"/* use list */, pn); alt != nil {
                        if def, _ := alt.(*Def); def != nil {
                                def.Append(pn)
                        }
                }
                l.useProject(pos, project)
        }
        return
}

func (l *loader) use(spec *ast.UseSpec) {
        var (
                name Value
                params []Value
        )
        if len(spec.Props) == 0 {
                l.p.error(spec.Pos(), "empty `use` spec")
        } else if name = l.expr(spec.Props[0]); name == nil {
                l.p.error(spec.Pos(), "undefined `use` target")
        } else if name == UniversalNone {
                l.p.error(spec.Pos(), "none `use` target")
        } else {
                for _, prop := range spec.Props[1:] {
                        params = append(params, l.expr(prop))
                }

                var scope = l.project.Scope()
                switch t := name.(type) {
                case *ProjectName:
                        l.useProjectName(spec.Props[0].Pos(), t)
                case *Def:
                        if alt := scope.Insert(t); alt != nil {
                                l.p.error(spec.Pos(), "`%s` already defined in %s", t.Name(), scope.comment)
                        }
                case *RuleEntry:
                        if alt := scope.Insert(t); alt != nil {
                                l.p.error(spec.Pos(), "`%s` already defined in %s", t.Name(), scope.comment)
                        }
                default:
                        l.p.error(spec.Pos(), "`%s` is not a usee (%T)", name, name)
                }
        }
}

func (l *loader) evalspec(spec *ast.EvalSpec) (res Value) {
        if num := len(spec.Props); num > 0 {
                var id = spec.Props[0]
                var position = l.p.file.Position(id.Pos())
                switch op := l.expr(id).(type) {
                case Caller:
                        res, _ = op.Call(position, l.exprs(spec.Props[1:])...)
                default:
                        var ( str string; err error )
                        if str, err = op.Strval(); err != nil {
                                l.p.error(id.Pos(), "%s: %v", op, err)
                        } else if _, obj := l.scope.Find(str); obj == nil {
                                l.p.error(id.Pos(), "`%s` undefined", str)
                        } else if f, _ := obj.(Caller); f == nil {
                                l.p.error(id.Pos(), "`%T` is not caller (%s)", obj, str)
                        } else if res, err = f.Call(position, l.exprs(spec.Props[1:])...); err != nil {
                                l.p.error(id.Pos(), "%s: %v", str, err)
                        }
                }
        }
        return
}

func (l *loader) dock(spec *ast.DockSpec) {
        var scope = l.scope
        for scope != nil && !strings.HasPrefix(scope.comment, "file ") {
                scope = scope.Outer()
        }

        def, alt := scope.Def(l.project, DockExecVarName, UniversalNone)
        if alt != nil {
                if d, _ := alt.(*Def); d == nil {
                        l.p.error(spec.Pos(), "name `%s` already taken in %v", def.Name(), scope.comment)
                        return
                } else {
                        def = d
                }
        }
        if def != nil {
                def.Assign(l.expr(spec.Props[0]))
        } else {
                l.p.error(spec.Props[0].Pos(), "cannot define `%v` in %v", DockExecVarName, scope)
        }
        return
}

func (l *loader) rule(clause *ast.RuleClause) {
        var (
                depends []Value
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

        if p, ok := clause.Program.(*ast.ProgramExpr); ok && p != nil {
                if progScope, _ = p.Scope.(*Scope); progScope == nil {
                        l.p.error(clause.Pos(), "undefined program scope (%T).", p.Scope)
                }
                if p.Recipes != nil {
                        recipes = l.exprs(p.Recipes)
                }
                params = p.Params
        } else {
                l.p.error(clause.Program.Pos(), "unsupported program type (%T)", clause.Program)
                return
        }
        
        var modifiers []Value
        if clause.Modifier != nil {
                modifiers = l.exprs(clause.Modifier.Elems)
        }
        
        var prog = NewProgram(l.globe, clause.Position, l.project, params, progScope, depends, recipes...)
        for i, m := range modifiers {
                position := l.p.file.Position(clause.Modifier.Elems[i].Pos())
                if err := prog.AddModifier(position, m); err != nil {
                        l.p.error(clause.Program.Pos(), "%v: %v", m, err)
                        return
                }
        }

        for n, target := range l.exprs(clause.Targets) {
                if target == nil {
                        l.p.error(clause.Targets[n].Pos(), "nil target (%T)", clause.Targets[n])
                        return
                }
                var ( entry *RuleEntry; err error )
                if entry, err = l.project.SetProgram(target, prog); err != nil {
                        l.p.error(clause.Targets[n].Pos(), "%v", err)
                        return
                } else /*if entry != nil*/ {
                        entry.Position = l.p.file.Position(clause.Targets[n].Pos())
                }
        }
        return
}

func (l *loader) include(spec *ast.IncludeSpec) {
        var (
                linfo = l.loads[len(l.loads)-1]
                specVal = l.expr(spec.Props[0])
                specName string
                params []Value
                err error
        )
        if specName, err = specVal.Strval(); err != nil {
                l.p.error(spec.Props[0].Pos(), "%v: %v", specVal, err)
                return
        }
        if len(spec.Props) > 1 {
                params = l.exprs(spec.Props[1:])
        }

        var (
                jointPath = filepath.Join(linfo.absDir, specName)
                absDir, baseName = filepath.Split(jointPath)
        )

        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))
        
        if _, err = l.ParseFile(l.fset, jointPath, nil, parseMode|Flat); err != nil {
                l.p.error(spec.Pos(), "%v", err)
                return
        }

        if len(params) > 0 {
                // TODO: parsing parameters
        }
        return
}

func (l *loader) openScope(comment string) ast.Scope {
        l.scope = NewScope(l.scope, l.project, comment)
        return l.scope
}

func (l *loader) closeScope(as ast.Scope) {
        if scope, ok := as.(*Scope); ok {
                l.scope = scope.Outer()
                // Must change the outer of dir scope to globe to avoid Finding symbols
                // recursively.
                if s := scope.Comment(); strings.HasPrefix(s, "dir ") /*|| strings.HasPrefix(s, "file ")*/ {
                        l.globe.SetScopeOuter(scope)
                }
        }
        return
}

func (l *loader) loadProjectBases(linfo *loadinfo, params Value) (err error) {
        if params == nil {
                return
        }
        
        g, _ := params.(*Group)
        if g == nil {
                err = fmt.Errorf("invalid parameters (%T)", params)
                return
        }

        var (
                absPath, specName string
                isDir bool
        )
        ParamsLoop: for _, elem := range g.Elems {
                if specName, err = elem.Strval(); err != nil { return }
                absPath, isDir, err = l.searchSpecPath(linfo, specName)
                if err != nil {
                        break ParamsLoop
                }

                //fmt.Printf("base: %v %v (%T) (%v) (%v)\n", l.project.Name(), elem, elem, absPath, l.paths)

                if isDir {
                        err = l.loadDir(specName, absPath, nil)
                } else {
                        err = l.load(specName, absPath, nil)
                }
                if err != nil {
                        break ParamsLoop
                }

                loaded, _ := l.loaded[absPath]
                if loaded == nil {
                        //fmt.Printf("loadProjectBases: project %v (%s -> %s)\n", l.project.Name(), elem, absPath)
                        err = fmt.Errorf("Project `%v' not loaded. (%T, %s)", elem, elem, absPath)
                        break
                } else {
                        l.project.Chain(loaded)
                }
        }
        return
}

func (l *loader) declareProject(ident *ast.Bareword, params Value) (err error) {
        var name = ident.Value
        /*if l.project != nil && l.project.Name() == ident.Value {
                return fmt.Errorf("already in project %s", l.project.Name())
        }*/

        var (
                linfo = l.loads[len(l.loads)-1]
                dec, ok = linfo.declares[name]
        )
        //fmt.Printf("declareProject: %v (%v) %v, %v\n", ident.Value, linfo.absPath(), l.scope, l.project)
        //fmt.Printf("declareProject: %v->%v, %v\n", l.project.Name(), ident.Value, l.scope)
        if !ok {
                var (
                        outer = l.scope
                        absDir = linfo.absDir
                        relPath string
                )
                if !filepath.IsAbs(absDir) {
                        //absDir = filepath.Join(l.workdir, absDir)
                        absDir, _ = filepath.Abs(absDir)
                }
                relPath, _ = filepath.Rel(l.workdir, absDir)

                // Avoid nesting project scopes!
                for strings.HasPrefix(outer.Comment(), "project \"") {
                        outer = outer.Outer()
                }

                dec = &declare{
                        project: l.globe.NewProject(outer, absDir, relPath, linfo.specName, name),
                }
                
                l.loaded[linfo.absPath()] = dec.project
                linfo.declares[name] = dec

                var (
                        p = dec.project
                        s = p.Scope()
                        use = NewScope(s, l.project, "use")
                )
                if obj, alt := s.ScopeName(p, "use", use); alt != nil {
                        if _, ok := alt.(*ScopeName); !ok {
                                err = fmt.Errorf("Name `use' already taken (%s).", s)
                                return
                        }
                } else if obj == nil {
                        err = fmt.Errorf("Failed adding `use' scope.")
                        return
                }
        }

        //fmt.Printf("declareProject: %v (%v) (%p -> %p)\n", ident.Value, linfo.absPath(), l.project, dec.project)

        if loader := linfo.loader; loader != nil {
                //fmt.Printf("DeclareProject: %s.%s %v\n", loader.Name(), dec.project.Name(), dec.backscope)
                
                if !strings.HasPrefix(loader.Scope().Comment(), "project \"") {
                        l.p.warn(ident.Pos(), "'%s' not loaded from project scope", name)
                }

                if _, a := loader.Scope().ProjectName(loader, name, dec.project); a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                err = fmt.Errorf("Name `%s' already taken (%T).", name, a)
                                return
                        }
                }

                //fmt.Printf("DeclareProject: %s.%s %v\n", loader.Name(), name, loader.Scope())
        }

        dec.backproj = l.project
        dec.backscope = l.scope
        l.project = dec.project
        l.scope = l.project.Scope()

        err = l.loadProjectBases(linfo, params)
        return
}

func (l *loader) closeCurrentProject(ident *ast.Bareword) (err error) {
        var (
                name = ident.Value
                linfo = l.loads[len(l.loads)-1]
                dec, ok = linfo.declares[name]
        )
        if dec == nil || !ok {
                return fmt.Errorf("no loaded project %s", name)
        }
        if l.project == nil {
                return fmt.Errorf("no current project")
        } else if s := l.project.Name(); s != name {
                return fmt.Errorf("current project is %s but %s", s, name)
        } else if l.project != dec.project {
                return fmt.Errorf("project conflicts (%s, %s)", l.project.Name(), dec.project.Name())
        }

        //fmt.Printf("closeCurrentProject: %v (%v) (%p -> %p)\n", ident.Value, linfo.absPath(), l.project, dec.backproj)
        
        l.scope = dec.backscope
        l.project = dec.backproj
        return
}

func (l *loader) MapFile(pat string, paths []string) {
        l.project.MapFile(pat, paths)
}

func (l *loader) File(s string) (f *File) {
        if l.project != nil {
                if f = l.project.ToFile(s); f != nil {
                        // fmt.Printf("file: %v %v\n", f, l.scope)
                }
        }
        return
}

func (l *loader) DeclareProject(ident *ast.Bareword, params Value) error {
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
                l.project = at.NamedProject()
                l.scope = l.project.Scope()
                return nil
        }
        return l.declareProject(ident, params)
}

func (l *loader) CloseCurrentProject(ident *ast.Bareword) error {
        if ident.Value == "@" {
                var (
                        linfo = l.loads[0]
                        dec, ok = linfo.declares[ident.Value]
                )
                if !ok {
                        panic("no @ declaraction")
                }
                l.project = dec.backproj
                l.scope = dec.backscope
                dec.backproj = nil
                dec.backscope = nil
                return nil
        }
        return l.closeCurrentProject(ident)
}

func (l *loader) OpenNamedScope(name, comment string) (ast.Scope, error) {
        if l.scope == nil {
                return nil, fmt.Errorf("no parent scope (%v)", comment)
        }
        
        var (
                outer = l.scope
                scope = NewScope(outer, l.project, comment)
        )
        if strings.HasPrefix(outer.Comment(), "dir ") {
                outer = outer.Outer() // discard dir scope
        }

        //fmt.Printf("OpenNamedScope: %v %v %v\n", name, l.scope, l.scope.Outer())

        outer.ScopeName(l.project, name, scope)
        l.scope = scope
        return l.scope, nil
}

func (l *loader) eval(x ast.Expr, ec EvalBits) (res Value, err error) {
	/*defer func() {
		if e := recover(); e != nil {
                        if fault := GoFault(e); fault != nil {
                                err = fault
                        } else if err, _ = e.(error); err == nil {
                                err = fmt.Errorf("%v", e)
                        }
		}
        }()*/

        if res = l.expr(x); res == nil {
                l.p.error(x.Pos(), "eval invalid expr `%T`", x)
                return
        }
        if ec&KeepClosures == 0 {
                if res, err = Disclose(res); err != nil {
                        l.p.error(x.Pos(), "%v", err)
                        return
                } else if res == nil {
                        return
                }
        }
        if ec&KeepDelegates == 0 {
                if res, err = Reveal(res); err != nil {
                        l.p.error(x.Pos(), "%v", err)
                        return
                } else if res == nil {
                        return
                }
        }
        if ec&DependValue == 0 {
                return
        }
        if def, ok := res.(*Def); ok && def != nil {
                res = def.Value
        }
        return
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

        const bits = anywhere
        if bits&FromGlobe != 0 {
                // If resolving @ in a rule (program) scope selection context,
                // e.g. '$(@.FOO)', Resolve have to ensure @ is pointing to the global
                // @ package.
                if _, obj = l.globe.scope.Find(name); obj != nil {
                        return
                }
        }
        if bits&FromBase != 0 && l.project != nil {
                for _, base := range l.project.Bases() {
                        if _, obj = base.Scope().Find(name); obj != nil {
                                return
                        }
                }
        }
        if bits&FromProject != 0 && l.project != nil {
                if _, obj = l.project.Scope().Find(name); obj != nil {
                        return
                }
        }
        if bits&FromHere != 0 && l.scope != nil {
                if _, obj = l.scope.Find(name); obj != nil {
                        return
                }
                if obj == nil && name != "use" {
                        // TODO: add this search path into Scope.Find
                        for _, base := range l.project.Bases() {
                                if _, obj = base.Scope().Find(name); obj != nil {
                                        return
                                }
                        }
                }
                if obj != nil && name == "use" {
                        if sn, ok := obj.(*ScopeName); ok && sn != nil {
                                // TODO: FindDef
                                // TODO: FindRule
                        }
                }
                if obj == nil /*&& (name == "use")*/ {
                        //fmt.Printf("Resolve: `%v' not in %v\n", name, l.scope)
                        //for _, base := range l.project.Bases() {
                        //        fmt.Printf("Resolve: %v %v\n", base.Name(), base.Scope())
                        //}
                        //for _, base := range l.scope.Chain() {
                        //        fmt.Printf("Resolve: %v\n", base)
                        //}
                }
        }
        //obj = MakeUnknownObject(name)
        return
}

func (l *loader) find(target Value) (entry *RuleEntry) {
        // ...
        return
}

func (l *loader) def(name string) (obj, alt Object) {
        obj, alt = l.scope.Def(l.project, name, UniversalNone)
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
func (l *loader) ParseFile(fset *token.FileSet, filename string, src interface{}, mode Mode) (f *ast.File, err error) {
	// get source
        var text []byte
	if text, err = readSource(filename, src); err != nil {
		return
	}

        // Save the current parser to restore later.
        saved := l.p

	l.mode = mode //| Trace
	l.tracing.enabled = l.mode&Trace != 0 // for convenience (l.trace is used frequently)

        // set the current parser
        l.p = new(parser)
	l.p.init(l, fset, filename, text)

	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

                if optSortErrors {
                        l.errors.Sort()
                }

		err = l.errors.Err()

                // decouple
                l.p.loader = nil
                l.p = saved
	} ()

        // set result values
        if f = l.p.parseFile(); f == nil {
                // source is not a valid source file - satisfy
                // ParseFile API and return a valid (but) empty
                // *ast.File
                f = &ast.File{
                        Name:  new(ast.Bareword),
                        Scope: l.openScope(fmt.Sprintf("file %s", filename)),
                }
                l.closeScope(f.Scope)
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
	if list, err = fd.Readdir(-1); err != nil || len(list) == 0 {
                return 
        }

        var (
                sym Object
                wd = l.workdir
                rel , _ = filepath.Rel(wd, pathname)
                ident = filepath.Base(pathname)
        )
        if ident == "_" {
                return fmt.Errorf("invalid package name %s", ident)
        }

        scope, err := l.OpenNamedScope(ident, fmt.Sprintf("config %s", pathname))
        if err != nil {
                return
        }

        defer l.closeScope(scope)

        sym, _ = l.def("/")
        sym.(*Def).Assign(MakeString(pathname))

        sym, _ = l.def(".")
        sym.(*Def).Assign(MakeString(rel))

	ListLoop: for _, d := range list {
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
                        if err = l.ParseConfigDir(filepath.Join(pathname, name), fullname); err != nil {
                                break ListLoop
                        }
                } else if s, a := l.def(name); a != nil {
                        err = fmt.Errorf("declare project: %v", err)
                        break ListLoop
                } else if def, _ := s.(*Def); def != nil {
                        var ( v []byte; s string )
                        if v, err = ioutil.ReadFile(fullname); err != nil { break ListLoop }
                        if s = string(v); !utf8.ValidString(s) {
                                err = fmt.Errorf("%s: invalid UTF8 content", fullname)
                                break ListLoop
                        }
                        def.SetOrigin(ImmediateDef)
                        def.Assign(MakeString(s))
                        //fmt.Printf("%s: %v = %v\n", ident, name, s)
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
func (l *loader) ParseDir(fset *token.FileSet, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*ast.Project, first error) {
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

        scope := l.openScope(fmt.Sprintf("dir %s", path))
        defer l.closeScope(scope)

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

                if strings.HasPrefix(d.Name(), ".#") ||
                        (!strings.HasSuffix(d.Name(), ".smart") &&
                        !strings.HasSuffix(d.Name(), ".sm")) {
                        continue
                } else if s := d.Name(); (s == "configure.smart" || s == "configure.sm") && (len(linked) > 0 || mo.IsDir()) {
                        if err := l.ParseConfigDir(filepath.Dir(filename), linked); err != nil {
                                if first == nil {
                                        first = err
                                }
                                return
                        }
                        continue ListLoop
                } else if s == "config.smart" || s == "config.sm" {
                        err = fmt.Errorf("use configure.sm[art] instead of config.sm[art]")
                        break
                }

		if mo.IsRegular() && (filter == nil || filter(d)) {
			if src, err := l.ParseFile(fset, filename, nil, mode|parsingDir); err == nil {
                                if src.Name == nil {
                                        first = fmt.Errorf("module '%v' has no name", filename)
                                        return
                                }

				name := src.Name.Value
				mod, found := mods[name]
				if !found {
					mod = &ast.Project{
                                                Name:    name,
                                                Scope:   scope,
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
func (l *loader) load(specName, absPath string, source interface{}) error {
        //fmt.Printf("load: %v (%v)\n", specName, absPath)

        if absPath == "" {
                return fmt.Errorf("No such module `%s' (in paths %v).", specName, l.paths)
        }

        if !filepath.IsAbs(absPath) {
                //panic(fmt.Sprintf("Invalid abs name `%s' (%s).", absPath, specName))
                return fmt.Errorf("Invalid abs name `%s' (%s).", absPath, specName)
        }
        
        // Check already project.
        if loaded, ok := l.loaded[absPath]; ok {
                var (
                        s = l.project.Scope()
                        name = loaded.Name()
                )
                //fmt.Printf("loaded: %v (%v)\n", loaded.Name(), absPath)
                if _, a := s.ProjectName(l.project, name, loaded); a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                return fmt.Errorf("Name `%s' already taken (%T).", name, a)
                        }
                }
                return nil
        }
        
        var absDir, baseName = filepath.Split(absPath)
        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

        doc, err := l.ParseFile(l.fset, absPath, source, parseMode)
        if err != nil {
                return err
        }
        if doc == nil {
                // FIXME: ...
        }

        //fmt.Printf("Load: %v %v\n", absPath, doc.Name.Name)
        return nil
}

func (l *loader) loadDir(specName, absDir string, filter func(os.FileInfo) bool) (err error) {
        //fmt.Printf("loadDir: %v: %v (%v)\n", l.project, specName, absDir)

        if !filepath.IsAbs(absDir) {
                panic(fmt.Sprintf("Invalid abs name `%s' (%s).", absDir, specName))
                err = fmt.Errorf("Invalid abs name `%s' (%s).", absDir, specName)
                return
        }

        // Check already project.
        if loaded, ok := l.loaded[absDir]; ok {
                var (
                        s = l.project.Scope()
                        name = loaded.Name()
                )
                //fmt.Printf("loaded: %v: %v (%v)\n", l.project.Name(), name, absDir)
                if _, a := s.ProjectName(l.project, name, loaded); a != nil {
                        if v, ok := a.(*ProjectName); !ok || v == nil {
                                err = fmt.Errorf("Name `%s' already taken (%T).", name, a)
                        }
                }
                return nil
        }

        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

        mods, err := l.ParseDir(l.fset, absDir, filter, parseMode)
        if err == nil && mods != nil {
        }

        //fmt.Printf("loadDir: %v %v\n", absDir, mods)
        return
}

func (l *loader) Load(filename string, source interface{}) error {
        s, _ := filepath.Split(filename)
        s, _  = filepath.Rel(l.workdir, s)
        return l.load(s, filename, source)
}

func (l *loader) LoadDir(path string, filter func(os.FileInfo) bool) (err error) {
        s, _ := filepath.Rel(l.workdir, path)
        return l.loadDir(s, path, filter)
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
