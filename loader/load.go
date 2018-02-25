//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package loader

import (
        "github.com/duzy/smart/ast"
        "github.com/duzy/smart/parser"
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "github.com/duzy/smart/runtime"
        "path/filepath"
        //"os/exec"
        "strings"
        //"bytes"
        "fmt"
        "os"
)

var parseMode = parser.DeclarationErrors //|parser.Trace

func restoreLoadingInfo(l *Loader) {
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

func saveLoadingInfo(l *Loader, specName, absDir, baseName string) *Loader {
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

func (l *Loader) parseInfo(pos token.Pos, s string, a... interface{}) {
        l.pc.ParseInfo(pos, s, a...)
}

func (l *Loader) parseWarn(pos token.Pos, s string, a... interface{}) {
        l.pc.ParseWarn(pos, s, a...)
}

func (l *Loader) searchSpecPath(linfo *loadinfo, specName string) (absPath string, isDir bool, err error) {
        var fi os.FileInfo
        if specName == "." {
                err = fmt.Errorf("Not possible to chain itself.")
        } else if abs := filepath.IsAbs(specName); abs || 
                strings.HasPrefix(specName, "../") || specName == ".." ||
                strings.HasPrefix(specName, "./") {
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
                                s = filepath.Join(l.Getwd(), base, specName)
                        }
                        if fi, err = os.Stat(s); err == nil && fi != nil {
                                isDir, absPath = fi.IsDir(), s
                                return
                        }
                }
        }
        return
}

func (l *Loader) loadImportSpec(spec *ast.ImportSpec) (err error, errArg int) {
        var (
                linfo = l.loads[len(l.loads)-1]
                specName, s string
                params []types.Value
                useList []types.Value
                nouse bool
        )
        errArg = -1 // means no wrong args
        if 0 < len(spec.Props) {
                if ee, ok := spec.Props[0].(*ast.EvaluatedExpr); ok && ee.Data != nil {
                        if specName, err = ee.Data.(types.Value).Strval(); err != nil {
                                return
                        }
                } else {
                        return ErrorIllImport, -1
                }
                for i, prop := range spec.Props[1:] {
                        if ee, ok := prop.(*ast.EvaluatedExpr); ok && ee.Data != nil {
                                // -param
                                // -param(value)
                                // -param=value
                                v := ee.Data.(types.Value)
                                switch t := v.(type) {
                                case *types.Flag:
                                        if s, err = t.Name.Strval(); err != nil { return }
                                        switch s {
                                        case "nouse": nouse = true
                                        default: params = append(params, v)
                                        }
                                case *types.Pair: // -param=value
                                        switch tt := t.Key.(type) {
                                        case *types.Flag:
                                                if s, err = tt.Name.Strval(); err != nil { return }
                                                switch s {
                                                case "use": useList = append(useList, t.Value)
                                                default: params = append(params, v)
                                                }
                                        default:
                                                return fmt.Errorf("parameter `%v' unsupported (%T)", v, v), i+1
                                        }
                                case *types.Argumented: // -param(value)
                                        switch tt := t.Value.(type) {
                                        case *types.Flag:
                                                if s, err = tt.Name.Strval(); err != nil { return }
                                                switch s {
                                                case "use": useList = append(useList, t.Args...)
                                                default: params = append(params, v)
                                                }
                                        default:
                                                return fmt.Errorf("parameter `%v' unsupported (%T)", v, v), i+1
                                        }
                                default:
                                        return fmt.Errorf("parameter `%v' unsupported (%T)", v, v), i+1
                                }
                        } else {
                                return ErrorIllImport, i+1
                        }
                }
        }

        if specName == "" {
                //fmt.Printf("%v: import %v\n", doc.Name, spec.Props)
                return ErrorIllImport, -1
        }

        var (
                absPath string
                isDir bool
        )
        if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
                return
        } else if absPath == "" {
                l.parseWarn(spec.Pos(), "missing '%s' (in %v)", specName, l.paths)
                return fmt.Errorf("'%s' not found", specName), -1
        }

        //fmt.Printf("import: %s (%s,dir=%v) (%v)\n", specName, absPath, isDir, l.project.Name())

        if isDir {
                err = l.loadDir(specName, absPath, nil)
        } else {
                err = l.load(specName, absPath, nil)
        }
        if err != nil || nouse {
                return
        }

        if loaded, _ := l.loaded[absPath]; loaded != nil {
                scope := l.project.Scope()
                pn, _ := scope.Lookup(loaded.Name()).(*types.ProjectName)
                if pn == nil {
                        l.parseWarn(spec.Pos(), "%v (%v,dir=%v) not in %v", specName, absPath, isDir, scope)
                        return fmt.Errorf("'%s' not found (%s)", specName, loaded.Name()), -1
                }
                // Add loaded project to the use list ('$(use->*)')
                if sn, _ := scope.Lookup("use").(*types.ScopeName); sn != nil {
                        if alt := sn.Scope().Insert(pn); alt != nil {
                                return fmt.Errorf("'%s' already defined in %v", specName, sn.Scope()), -1
                        }
                        if _, alt := sn.Scope().InsertDef(l.project, "*"/* use list */, pn); alt != nil {
                                if def, _ := alt.(*types.Def); def != nil {
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
                err = l.useProject(spec.Props[0].Pos(), loaded)
        } else {
                fmt.Fprintf(os.Stderr, "not loaded: %v (%v)\n", specName, absPath)
                for k, v := range l.loaded {
                        fmt.Fprintf(os.Stderr, "   loaded: %v (%v)\n", v.Name(), k)
                }
        }
        return
}

func (l *Loader) closuredelegate(x *ast.ClosureDelegate) (obj types.Object, args []types.Value, err error) {
        name, err := l.expr(x.Name)
        if err != nil {
                return nil, nil, err
        }
        if x.Resolved == nil {
                err = fmt.Errorf("Unresolved reference `%s'.", name)
                return
        }

        tok := token.ILLEGAL
        switch x.TokLp {
        case token.LPAREN: tok = token.LPAREN
        case token.LBRACE: tok = token.LBRACE
        case token.ILLEGAL:
                if x.Tok.IsClosure() || x.Tok.IsDelegate() {
                        tok = token.LPAREN
                } else {
                        err = fmt.Errorf("Unregonized closure/delegate (%v).", x.Tok)
                }
        default:
                err = fmt.Errorf("Unregonized closure/delegate (%v, %v).", x.TokLp, x.Tok)
        }

        switch tok {
        case token.LPAREN:
                def, _ := x.Resolved.(types.Caller)
                if def == nil {
                        err = fmt.Errorf("Uncallable `%s' resolved (%T).", name, x.Resolved)
                        return
                } else if obj = def.(types.Object); obj == nil {
                        err = fmt.Errorf("Non-object callable `%s' resolved (%T).", name, def)
                        return
                }
        case token.LBRACE:
                exe, _ := x.Resolved.(types.Executer)
                if exe == nil {
                        err = fmt.Errorf("Unexecutible `%s' resolved (%T).", name, x.Resolved)
                        return
                } else if obj = exe.(types.Object); obj == nil {
                        err = fmt.Errorf("Non-object executible `%s' resolved (%T).", name, exe)
                        return
                }
        default:
                if err == nil {
                        err = fmt.Errorf("Unregonized closure/delegate (%v, %v).", x.TokLp, x.Tok)
                }
                return
        }
        
        for _, x := range x.Args {
                if a,e := l.expr(x); e != nil {
                        err = e
                        return
                } else if a == nil {
                        err = fmt.Errorf("nil closure arg `%T'", e)
                        return
                } else {
                        args = append(args, a)
                }
        }
        return
}

func (l *Loader) closure(x *ast.ClosureExpr) (types.Value, error) {
        if obj, args, err := l.closuredelegate(&x.ClosureDelegate); err == nil {
                return types.Closure(x.Position, obj, args...), nil
        } else {
                return nil, err
        }
}

func (l *Loader) delegate(x *ast.DelegateExpr) (v types.Value, err error) {
        if obj, args, err := l.closuredelegate(&x.ClosureDelegate); err == nil {
                return types.Delegate(x.Position, obj, args...), nil
        } else {
                return nil, err
        }
}

func (l *Loader) recipe(x *ast.RecipeExpr) (v types.Value, err error) {
        if len(x.Elems) == 0 {
                v = values.None
        } else if x.Dialect == "" {
                var elems []types.Value
                if a, err := l.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        elems = append(elems, a...)
                        v = values.List(elems...)
                }
        } else if a, err := l.exprs(x.Elems); err != nil {
                return nil, err
        } else {
                v = values.Compound(a...)
        }
        return
}

func (l *Loader) expr(expr ast.Expr) (v types.Value, err error) {
        if expr == nil {
                //err = fmt.Errorf("nil expr")
                return
        }
        switch x := expr.(type) {
        case *ast.EvaluatedExpr:
                if x.Data != nil {
                        v = x.Data.(types.Value)
                } else {
                        err = fmt.Errorf("Expr evaluated to nil (%v).", x.Expr)
                        return
                }
        case *ast.ArgumentedExpr:
                av := new(types.Argumented)
                if av.Value, err = l.expr(x.X); err != nil {
                        return
                }
                if av.Args, err = l.exprs(x.Arguments); err != nil {
                        return
                }
                v = av
        case *ast.ClosureExpr:
                v, err = l.closure(x)
        case *ast.DelegateExpr:
                v, err = l.delegate(x)
        case *ast.RecipeExpr:
                v, err = l.recipe(x)
        case *ast.BasicLit:
                v = values.Literal(x.Kind, x.Value)
        case *ast.Bareword:
                v = values.Bareword(x.Value)
        case *ast.Barecomp:
                if a, err := l.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.Barecomp(a...)
                }
        case *ast.Barefile:
                if file, _ := x.File.(*types.File); file != nil {
                        if x.Val != nil {
                                v = values.Barefile(x.Val.(types.Value), file)
                        } else if a, err := l.expr(x.Name); err != nil {
                                return nil, err
                        } else if a != nil {
                                v = values.Barefile(a, file)
                        }
                }
                if v == nil {
                        err = fmt.Errorf("Invalid barefile '%s'", x.Name)
                }
        case *ast.GlobExpr: // Just "*"
                v = values.Glob(x.Tok)
        case *ast.PathExpr:
                if a, err := l.exprs(x.Segments); err != nil {
                        return nil, err
                } else {
                        v = values.Path(a...)
                }
        case *ast.PathSegExpr:
                switch x.Tok {
                case token.PCON:   v = values.PathSeg('/')
                case token.PERIOD: v = values.PathSeg('.')
                case token.DOTDOT: v = values.PathSeg('^') // 
                default: err := fmt.Errorf("Unsupported PathSeg `%v'.", x.Tok)
                        return nil, err
                }
        case *ast.FlagExpr:
                if a, err := l.expr(x.Name); err != nil {
                        return nil, err
                } else {
                        v = values.Flag(a)
                }
        case *ast.CompoundLit:
                if a, err := l.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.Compound(a...)
                }
        case *ast.GroupExpr:
                if a, err := l.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.Group(a...)
                }
        case *ast.ListExpr:
                if a, err := l.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.List(a...)
                }
        case *ast.KeyValueExpr:
                var a, b types.Value
                if b, err = l.expr(x.Value); err != nil {
                        return
                } else if a, err = l.expr(x.Key); err != nil {
                        return
                } else {
                        v = values.Pair(a, b)
                }
        case *ast.PercExpr:
                var a, b types.Value
                if a, err = l.expr(x.X); err != nil { // a can be nil
                        return
                } else if b, err = l.expr(x.Y); err != nil { // b can be nil
                        return
                } else {
                        v = values.GlobPattern(a, b)
                }
        case *ast.RecipeDefineClause:
                //fmt.Printf("RecipeDefineClause: %s: %T %v\n", l.project.Name(), x.Sym, x.Sym)
                var name types.Value
                if name, err = l.expr(x.Name); err != nil {
                        return nil, err
                } else if name == nil {
                        err = fmt.Errorf("Expr value is nil `%T'", expr)
                        return nil, err
                }

                def, _ := x.Sym.(*types.Def)
                if def == nil {
                        err = fmt.Errorf("Symbol `%s' undefined in %v", def.Name(), def.Parent())
                        return nil, err
                }

                var str string
                if str, err = name.Strval(); err != nil { return }
                if def.Name() != str {
                        err = fmt.Errorf("Symbol `%s' differs from `%s'", str, def.Name())
                        return nil, err
                }

                // Double-check the symbol in the scope.
                if o := def.Parent().Lookup(def.Name()); o != def {
                        err = fmt.Errorf("Symbol `%s' undefined in %v", def.Name(), def.Parent())
                        return nil, err
                }

                //fmt.Printf("expr: RecipeDefineClause: %s: %p/%p %T %v\n", l.project.Name(), x.Sym, def, x.Sym, x.Sym)
                v = def
        }
        if v == nil && err == nil {
                err = fmt.Errorf("Expr value is nil (%T)", expr)
        }
        return
}

func (l *Loader) exprs(exprs []ast.Expr) (values []types.Value, err error) {
        for _, x := range exprs {
                if v, err := l.expr(x); err != nil {
                        return nil, err
                } else {
                        values = append(values, v)
                }
        }
        return
}

func (l *Loader) useProject(pos token.Pos, usee *types.Project) error {
        var (
                position = l.pc.Position(pos)
                entry *types.RuleEntry
                obj types.Object
        )
        if use := usee.Scope().Lookup("use"); use == nil {
                return fmt.Errorf("Project `%v' has no 'use' package.", usee.Name())
        } else if sn, ok := use.(*types.ScopeName); !ok || sn == nil {
                return fmt.Errorf("Project `%v' has invalid 'use' package (%T).", usee.Name(), use)
        } else if obj = sn.Scope().Lookup(":"); obj == nil {
                return nil // The use entry is not defined.
        }

        //fmt.Printf("useProject: %T %v\n", obj, obj.Strval())
        if entry, _ = obj.(*types.RuleEntry); entry == nil {
                return fmt.Errorf("Project `%v' has invalid 'use' entry (%T).", usee.Name(), obj)
        }

        results, err := entry.Execute(position)
        if err != nil {
                return err
        }

        var defs []*types.Def
        for _, result := range results {
                switch result.Type() {
                case  types.ListType:
                        for _, elem := range result.(*types.List).Elems {
                                if def, ok := elem.(*types.Def); ok && def != nil {
                                        defs = append(defs, def)
                                }
                        }
                }
        }
        for _, def := range defs {
                //fmt.Printf("%v: %v\n", l.project.Name(), elem)
                //fmt.Printf("useProject: %v: %v\n", l.project.Name(), def)

                newDef, alt := l.project.Scope().InsertDef(l.project, def.Name(), values.None)
                if alt != nil {
                        if d, _ := alt.(*types.Def); d == nil {
                                return fmt.Errorf("Name `%s' already taken in project `%s' (%T).", def.Name(), alt, l.project.Name())
                        } else {
                                newDef = d
                        }
                }
                if newDef == nil {
                        return fmt.Errorf("Cannot define `%s' in project `%s'.", def.Name(), l.project.Name())
                }

                // Append the delegate if not referenced yet.
                if !types.Refs(newDef, def) {
                        newDef.Append(types.Delegate(position, def))
                }

                //fmt.Printf("useProject: %s: %s: %v (%s)\n", l.project.Name(), usee.Name(), newDef, def.Strval())
        }
        return nil
}

func (l *Loader) useProjectName(pos token.Pos, pn *types.ProjectName) error {
        var (
                scope = l.project.Scope()
                project = pn.Project()
                //useList []types.Value
        )
        if project == nil {
                return fmt.Errorf("%v is nil", pn)
        }
        
        // FIXME: defined used project in represented order
        if sn, _ := scope.Lookup("use").(*types.ScopeName); sn != nil {
                if alt := sn.Scope().Insert(pn); alt != nil {
                        if alt.Type().Kind() == types.ProjectNameKind {
                                l.parseInfo(pos, "'%s' already used", pn.Name())
                        } else {
                                return fmt.Errorf("'%s' already defined in %s", pn.Name(), sn.Scope())
                        }
                }
                if _, alt := sn.Scope().InsertDef(l.project, "*"/* use list */, pn); alt != nil {
                        if def, _ := alt.(*types.Def); def != nil {
                                def.Append(pn)
                        }
                }
        } else {
                return nil //fmt.Errorf("'use' scope is not in %s", scope)
        }

        return l.useProject(pos, project)
}

func (l *Loader) use(spec *ast.UseSpec) (err error) {
        var (
                name types.Value
                params []types.Value
        )
        if len(spec.Props) == 0 {
                return fmt.Errorf("Empty use spec.")
        } else if name, err = l.expr(spec.Props[0]); err != nil {
                return
        } else if name == nil {
                return fmt.Errorf("Undefined `use' target.")
        } else if  name == values.None {
                return fmt.Errorf("None `use' target.")
        }
        for _, prop := range spec.Props[1:] {
                if v, err := l.expr(prop); err != nil {
                        return err
                } else {
                        params = append(params, v)
                }
        }

        var scope = l.project.Scope()
        switch t := name.(type) {
        case *types.ProjectName:
                return l.useProjectName(spec.Props[0].Pos(), t)
        case *types.Def:
                if alt := scope.Insert(t); alt != nil {
                        return fmt.Errorf("'%s' already defined in %s", t.Name(), scope)
                } else {
                        return nil // okay
                }
        case *types.RuleEntry:
                if alt := scope.Insert(t); alt != nil {
                        return fmt.Errorf("'%s' already defined in %s", t.Name(), scope)
                } else {
                        return nil // okay
                }
        }

        return fmt.Errorf("'%s' is not a usee (%T)", name, name)
}

func (l *Loader) eval(spec *ast.EvalSpec) (res types.Value, err error) {
        if num := len(spec.Props); num > 0 {
                var v types.Value
                if v, err = l.expr(spec.Props[0]); err != nil {
                        return
                }
                var position = l.pc.Position(spec.Props[0].Pos())
                switch op := v.(type) {
                case types.Caller:
                        if a, err := l.exprs(spec.Props[1:]); err != nil {
                                return nil, err
                        } else {
                                res, _ = op.Call(position, a...)
                        }
                default:
                        var str string
                        if str, err = op.Strval(); err != nil { return }
                        if _, obj := l.scope.Find(str); obj != nil {
                                if f, _ := obj.(types.Caller); f != nil {
                                        if a, err := l.exprs(spec.Props[1:]); err != nil {
                                                return nil, err
                                        } else {
                                                res, err = f.Call(position, a...)
                                        }
                                }
                        } else {
                                err = fmt.Errorf("Eval undefined `%s'", str)
                        }
                }
        }
        return
}

func (l *Loader) dock(spec *ast.DockSpec) (err error) {
        var scope = l.scope
        for scope != nil && !strings.HasPrefix(scope.Comment(), "file ") {
                scope = scope.Outer()
        }
        
        def, alt := scope.InsertDef(l.project, runtime.DockExecVarName, values.None)
        if alt != nil {
                if d, _ := alt.(*types.Def); d == nil {
                        return fmt.Errorf("Name `%s' already taken in %v.", def.Name(), scope)
                } else {
                        def = d
                }
        }
        if def != nil {
                var val types.Value
                if val, err = l.expr(spec.Props[0]); err == nil {
                        def.Assign(val)
                }
        } else {
                err = fmt.Errorf("Cannot define `%v' in %v", runtime.DockExecVarName, scope)
        }
        return
}

func (l *Loader) rule(clause *ast.RuleClause) (err error) {
        var (
                targets []types.Value
                depends []types.Value
                recipes []types.Value
                depval types.Value
                progScope *types.Scope
                params []string
        )
        for _, depend := range clause.Depends {
                if depval, err = l.expr(depend); err != nil {
                        return
                } else if depval == nil {
                        err = fmt.Errorf("Invalid depend type `%T'.", depend)
                        return
                }
                switch dep := depval.(type) {
                case *types.List:
                        depends = append(depends, dep.Elems...)
                default:
                        depends = append(depends, depval)
                }
        }

        if p, ok := clause.Program.(*ast.ProgramExpr); ok && p != nil {
                if progScope, _ = p.Scope.(*types.Scope); progScope == nil {
                        err = fmt.Errorf("Undefined program scope (%T).", p.Scope)
                        return
                }
                if p.Recipes != nil {
                        if recipes, err = l.exprs(p.Recipes); err != nil {
                                return
                        }
                }
                params = p.Params
        } else {
                err = fmt.Errorf("unsupported program type (%T)", clause.Program)
                return
        }
        
        var modifiers []types.Value
        if clause.Modifier != nil {
                modifiers, err = l.exprs(clause.Modifier.Elems)
                if err != nil {
                        return
                }
        }
        
        var prog = l.NewProgram(clause.Position, l.project, params, progScope, depends, recipes...)
        for i, m := range modifiers {
                position := l.pc.Position(clause.Modifier.Elems[i].Pos())
                if err = prog.AddModifier(position, m); err != nil {
                        return
                }
        }

        if targets, err = l.exprs(clause.Targets); err != nil {
                return
        }
        
        var name string
        for n, target := range targets {
                if target == nil {
                        err = fmt.Errorf("nil target (%T)", clause.Targets[n])
                        return
                }

                class := types.GeneralRuleEntry
                if name, err = target.Strval(); err != nil {
                        return
                } else if name == "use" {
                        if n == 0 && len(clause.Targets) == 1 {
                                class = types.UseRuleEntry
                        } else {
                                l.parseWarn(clause.Targets[n].Pos(), "'use' rule mixed with other targets")
                                err = fmt.Errorf("mixes 'use' and normal targets")
                                return
                        }
                }/* else if l.project.IsFile(name) {
                        class = types.ExplicitFileEntry
                }*/
                
                switch namv := target.(type) {
                case *types.GlobPattern:
                        l.project.SetGlobPatternProgram(namv, class, prog)
                default:
                        if _, err = l.project.SetProgram(name, class, prog); err != nil {
                                return
                        }
                }
        }
        return
}

func (l *Loader) include(spec *ast.IncludeSpec) (err error) {
        var (
                linfo = l.loads[len(l.loads)-1]
                specVal types.Value
                specName string
                params []types.Value
        )
        if specVal, err = l.expr(spec.Props[0]); err != nil { return }
        if specName, err = specVal.Strval(); err != nil { return }
        if len(spec.Props) > 1 {
                if params, err = l.exprs(spec.Props[1:]); err != nil { return err }
        }

        var (
                jointPath = filepath.Join(linfo.absDir, specName)
                absDir, baseName = filepath.Split(jointPath)
        )
        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))
        
        _, err = l.pc.ParseFile(l.fset, jointPath, nil, parseMode|parser.Flat)
        if err != nil {
                return err
        }

        if len(params) > 0 {
                // TODO: parsing parameters
        }
        return nil
}

func (l *Loader) openScope(comment string) ast.Scope {
        l.scope = types.NewScope(l.scope, l.project, comment)
        return l.scope
}

func (l *Loader) closeScope(as ast.Scope) (err error) {
        if scope, ok := as.(*types.Scope); ok {
                l.scope = scope.Outer()
                // Must change the outer of dir scope to globe to avoid Finding symbols
                // recursively.
                if s := scope.Comment(); strings.HasPrefix(s, "dir ") /*|| strings.HasPrefix(s, "file ")*/ {
                        l.Globe().SetScopeOuter(scope)
                }
        } else {
                err = fmt.Errorf("bad runtime scope (%T)", as)
        }
        return
}

func (l *Loader) loadProjectBases(linfo *loadinfo, params types.Value) (err error) {
        if params == nil {
                return
        }
        
        g, _ := params.(*types.Group)
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

                //fmt.Printf("base: %v %v (%T) %v\n", l.project.Name(), elem, elem, absPath)

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

func (l *Loader) declareProject(ident *ast.Bareword, params types.Value) (err error) {
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
                        //absDir = filepath.Join(l.Getwd(), absDir)
                        absDir, _ = filepath.Abs(absDir)
                }
                relPath, _ = filepath.Rel(l.Getwd(), absDir)

                // Avoid nesting project scopes!
                for strings.HasPrefix(outer.Comment(), "project \"") {
                        outer = outer.Outer()
                }

                dec = &declare{
                        project: l.Globe().NewProject(outer, absDir, relPath, linfo.specName, name),
                }
                
                l.loaded[linfo.absPath()] = dec.project
                linfo.declares[name] = dec

                var (
                        p = dec.project
                        s = p.Scope()
                        use = types.NewScope(s, l.project, "use")
                )
                if obj, alt := s.InsertScopeName(p, "use", use); alt != nil {
                        if _, ok := alt.(*types.ScopeName); !ok {
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
                        l.parseWarn(ident.Pos(), "'%s' not loaded from project scope", name)
                }

                if _, a := loader.Scope().InsertProjectName(loader, name, dec.project); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
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

func (l *Loader) closeCurrentProject(ident *ast.Bareword) (err error) {
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

// Loader.Load loads script from a file or source code (string, []byte).
func (l *Loader) load(specName, absPath string, source interface{}) error {
        //fmt.Printf("load: %v (%v)\n", specName, absPath)

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
                if _, a := s.InsertProjectName(l.project, name, loaded); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                return fmt.Errorf("Name `%s' already taken (%T).", name, a)
                        }
                }
                return nil
        }
        
        var absDir, baseName = filepath.Split(absPath)
        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

        doc, err := l.pc.ParseFile(l.fset, absPath, source, parseMode)
        if err != nil {
                return err
        }
        if doc == nil {
                // FIXME: ...
        }

        //fmt.Printf("Load: %v %v\n", absPath, doc.Name.Name)
        return nil
}

func (l *Loader) loadDir(specName, absDir string, filter func(os.FileInfo) bool) (err error) {
        //fmt.Printf("loadDir: %v: %v (%v)\n", l.project.Name(), specName, absDir)

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
                if _, a := s.InsertProjectName(l.project, name, loaded); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                err = fmt.Errorf("Name `%s' already taken (%T).", name, a)
                        }
                }
                return nil
        }

        defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

        mods, err := l.pc.ParseDir(l.fset, absDir, filter, parseMode)
        if err == nil && mods != nil {
        }

        //fmt.Printf("LoadDir: %v %v\n", absDir, mods)
        return
}

func (l *Loader) Load(filename string, source interface{}) error {
        s, _ := filepath.Split(filename)
        s, _  = filepath.Rel(l.Getwd(), s)
        //fmt.Printf("Load: %v (%v)\n", s, filename)
        return l.load(s, filename, source)
}

func (l *Loader) LoadDir(path string, filter func(os.FileInfo) bool) (err error) {
        s, _ := filepath.Rel(l.Getwd(), path)
        //fmt.Printf("LoadDir: %v (%v)\n", s, path)
        return l.loadDir(s, path, filter)
}

func (pc *parseContext) MapFile(pat string, paths []string) {
        pc.project.MapFile(pat, paths)
}

func (pc *parseContext) File(s string) (f *types.File) {
        if pc.project != nil {
                if f = pc.project.ToFile(s); f != nil {
                        // fmt.Printf("file: %v %v\n", f, pc.scope)
                }
        }
        return
}

func (pc *parseContext) DeclareProject(ident *ast.Bareword, params types.Value) error {
        if ident.Value == "@" {
                var (
                        linfo = pc.loads[0]
                        dec, ok = linfo.declares[ident.Value]
                        at = pc.Globe().Scope().Lookup(ident.Value)
                )
                if !ok {
                        dec = &declare{ project: at.Project() }
                        linfo.declares[ident.Value] = dec
                }
                dec.backproj = pc.project
                dec.backscope = pc.scope
                pc.project = at.Project()
                pc.scope = pc.project.Scope()
                return nil
        }
        return pc.declareProject(ident, params)
}

func (pc *parseContext) CloseCurrentProject(ident *ast.Bareword) error {
        if ident.Value == "@" {
                var (
                        linfo = pc.loads[0]
                        dec, ok = linfo.declares[ident.Value]
                )
                if !ok {
                        panic("no @ declaraction")
                }
                pc.project = dec.backproj
                pc.scope = dec.backscope
                dec.backproj = nil
                dec.backscope = nil
                return nil
        }
        return pc.closeCurrentProject(ident)
}

func (pc *parseContext) OpenNamedScope(name, comment string) (ast.Scope, error) {
        if pc.scope == nil {
                return nil, fmt.Errorf("no parent scope (%v)", comment)
        }
        
        var (
                outer = pc.scope
                scope = types.NewScope(outer, pc.project, comment)
        )
        if strings.HasPrefix(outer.Comment(), "dir ") {
                outer = outer.Outer() // discard dir scope
        }

        //fmt.Printf("OpenNamedScope: %v %v %v\n", name, pc.scope, pc.scope.Outer())

        outer.InsertScopeName(pc.project, name, scope)
        pc.scope = scope
        return pc.scope, nil
}

func (pc *parseContext) OpenScope(comment string) ast.Scope {
        return pc.openScope(comment)
}

func (pc *parseContext) CloseScope(as ast.Scope) error {
        return pc.closeScope(as)
}

func (pc *parseContext) ClauseImport(spec *ast.ImportSpec) (error, int) {
        return pc.loadImportSpec(spec)
}

func (pc *parseContext) ClauseInclude(spec *ast.IncludeSpec) error {
        return pc.include(spec)
}

func (pc *parseContext) ClauseUse(spec *ast.UseSpec) error {
        return pc.use(spec)
}

func (pc *parseContext) ClauseEval(spec *ast.EvalSpec) error {
        _, err := pc.eval(spec)
        return err
}

func (pc *parseContext) ClauseDock(spec *ast.DockSpec) error {
        return pc.dock(spec)
}

func (pc *parseContext) Rule(clause *ast.RuleClause) (parser.RuntimeObj, error) {
        return nil, pc.rule(clause)
}

func (pc *parseContext) Eval(x ast.Expr, ec parser.EvalBits) (res types.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
                        if fault := types.GoFault(e); fault != nil {
                                err = fault
                        } else if err, _ = e.(error); err == nil {
                                err = fmt.Errorf("%v", e)
                        }
		}
        }()

        if res, err = pc.expr(x); err != nil || res == nil {
                return
        }
        if ec&parser.KeepClosures == 0 {
                if res, err = types.Disclose(pc.scope, res); err != nil || res == nil {
                        return
                }
        }
        if ec&parser.KeepDelegates == 0 {
                if res = types.Reveal(res); res == nil {
                        return
                }
        }
        if ec&parser.DependValue == 0 {
                return
        }

        // Cast depends so that it's could be easily used.

        if def, ok := res.(*types.Def); ok && def != nil {
                res = def.Value
        }
        return
}

func (pc *parseContext) Resolve(name string, bits parser.ResolveBits) parser.RuntimeObj {
        if bits&parser.FromGlobe != 0 {
                // If resolving @ in a rule (program) scope selection context,
                // e.g. '$(@.FOO)', Resolve have to ensure @ is pointing to the global
                // @ package.
                _, obj := pc.Globe().Scope().Find(name)
                if obj != nil {
                        return obj.(parser.RuntimeObj)
                }
        }
        if bits&parser.FromBase != 0 && pc.project != nil {
                for _, base := range pc.project.Bases() {
                        if _, obj := base.Scope().Find(name); obj != nil {
                                return obj.(parser.RuntimeObj)
                        }
                }
        }
        if bits&parser.FromProject != 0 && pc.project != nil {
                _, obj := pc.project.Scope().Find(name)
                if obj != nil {
                        return obj.(parser.RuntimeObj)
                }
        }
        if bits&parser.FromHere != 0 && pc.scope != nil {
                _, obj := pc.scope.Find(name)
                if obj != nil {
                        return obj.(parser.RuntimeObj)
                }
                if obj == nil && name != "use" {
                        // TODO: add this search path into Scope.Find
                        for _, base := range pc.project.Bases() {
                                if _, obj = base.Scope().Find(name); obj != nil {
                                        return obj.(parser.RuntimeObj)
                                }
                        }
                }
                if obj != nil && name == "use" {
                        if sn, ok := obj.(*types.ScopeName); ok && sn != nil {
                                // TODO: parser.FindDef
                                // TODO: parser.FindRule
                        }
                }
                if obj == nil /*&& (name == "use")*/ {
                        //fmt.Printf("Resolve: `%v' not in %v\n", name, pc.scope)
                        //for _, base := range pc.project.Bases() {
                        //        fmt.Printf("Resolve: %v %v\n", base.Name(), base.Scope())
                        //}
                        //for _, base := range pc.scope.Chain() {
                        //        fmt.Printf("Resolve: %v\n", base)
                        //}
                }
        }
        return nil
}

func (pc *parseContext) Symbol(name string, t types.Type) (obj, alt parser.RuntimeObj) {
        switch t {
        case types.DefType:
                var (
                        scope = pc.scope // always in the current scope
                        def *types.Def
                )
                def, alt = scope.InsertDef(pc.project, name, values.None)
                if def != nil {
                        obj = def
                }
        case types.RuleEntryType:
                var (
                        scope = pc.project.Scope() // always in the project
                        entry *types.RuleEntry
                )
                entry, alt = scope.InsertEntry(pc.project, types.GeneralRuleEntry, name)
                if entry != nil {
                        obj = entry
                }
                //fmt.Printf("entry: %v (%v) (%v)\n", name, pc.project.Name(), alt)
        }
        return
}
