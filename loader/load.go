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
        "errors"
        //"bytes"
        "fmt"
        "os"
)

const (
        ignoreRuleName = "ignore"
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
                err = errors.New("Not possible to chain itself.")
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

func (l *Loader) loadImportSpec(spec *ast.ImportSpec) (err error) {
        var (
                linfo = l.loads[len(l.loads)-1]
                specName string
                params []types.Value
                nouse bool
        )
        if 0 < len(spec.Props) {
                if ee, ok := spec.Props[0].(*ast.EvaluatedExpr); ok && ee.Data != nil {
                        specName = ee.Data.(types.Value).Strval()
                } else {
                        return ErrorIllImport
                }
                for _, prop := range spec.Props[1:] {
                        if ee, ok := prop.(*ast.EvaluatedExpr); ok && ee.Data != nil {
                                v := ee.Data.(types.Value)
                                switch s := v.Strval(); s {
                                case "nouse": nouse = true
                                default: params = append(params, v)
                                }
                        } else {
                                return ErrorIllImport
                        }
                }
        }

        if specName == "" {
                //fmt.Printf("%v: import %v\n", doc.Name, spec.Props)
                return ErrorIllImport
        }

        var (
                absPath string
                isDir bool
        )
        if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
                return
        } else if absPath == "" {
                l.parseWarn(spec.Pos(), "missing '%s' (in %v)", specName, l.paths)
                return errors.New(fmt.Sprintf("'%s' not found", specName))
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
                        return errors.New(fmt.Sprintf("'%s' not found (%s)", specName, loaded.Name()))
                }
                if sn, _ := scope.Lookup("use").(*types.ScopeName); sn != nil {
                        if alt := sn.Scope().Insert(pn); alt != nil {
                                return errors.New(fmt.Sprintf("'%s' already defined in %v", specName, sn.Scope()))
                        }
                        if _, alt := sn.Scope().InsertDef(l.project, "*"/* use list */, pn); alt != nil {
                                if def, _ := alt.(*types.Def); def != nil {
                                        def.Append(pn)
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
                err = errors.New(fmt.Sprintf("Unresolved reference `%s'.", name))
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
                        err = errors.New(fmt.Sprintf("Unregonized closure/delegate (%v).", x.Tok))
                }
        default:
                err = errors.New(fmt.Sprintf("Unregonized closure/delegate (%v, %v).", x.TokLp, x.Tok))
        }

        switch tok {
        case token.LPAREN:
                def, _ := x.Resolved.(types.Caller)
                if def == nil {
                        err = errors.New(fmt.Sprintf("Uncallable `%s' resolved (%T).", name, x.Resolved))
                        return
                } else if obj = def.(types.Object); obj == nil {
                        err = errors.New(fmt.Sprintf("Non-object callable `%s' resolved (%T).", name, def))
                        return
                }
        case token.LBRACE:
                exe, _ := x.Resolved.(types.Executer)
                if exe == nil {
                        err = errors.New(fmt.Sprintf("Unexecutible `%s' resolved (%T).", name, x.Resolved))
                        return
                } else if obj = exe.(types.Object); obj == nil {
                        err = errors.New(fmt.Sprintf("Non-object executible `%s' resolved (%T).", name, exe))
                        return
                }
        default:
                if err == nil {
                        err = errors.New(fmt.Sprintf("Unregonized closure/delegate (%v, %v).", x.TokLp, x.Tok))
                }
                return
        }
        
        for _, x := range x.Args {
                if a,e := l.expr(x); e != nil {
                        err = e
                        return
                } else if a == nil {
                        err = errors.New(fmt.Sprintf("nil closure arg `%T'", e))
                        return
                } else {
                        args = append(args, a)
                }
        }
        return
}

func (l *Loader) closure(x *ast.ClosureExpr) (types.Value, error) {
        if obj, args, err := l.closuredelegate(&x.ClosureDelegate); err == nil {
                return types.Closure(obj, args...), nil
        } else {
                return nil, err
        }
}

func (l *Loader) delegate(x *ast.DelegateExpr) (v types.Value, err error) {
        if obj, args, err := l.closuredelegate(&x.ClosureDelegate); err == nil {
                return types.Delegate(obj, args...), nil
        } else {
                return nil, err
        }
}

func (l *Loader) recipe(x *ast.RecipeExpr) (v types.Value, err error) {
        if len(x.Elems) == 0 {
                v = values.None
        } else if x.Dialect == "" {
                var elems []types.Value
                /*switch t := x.Elems[0].(type) {
                case *ast.Bareword:
                case *ast.UseDefineClause:
                default: 
                        s := fmt.Sprintf("Unsupported recipe command (%T)", t)
                        return nil, errors.New(s)
                }*/
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
                //err = errors.New("nil expr")
                return
        }
        switch x := expr.(type) {
        case *ast.EvaluatedExpr:
                if x.Data != nil {
                        v = x.Data.(types.Value)
                } else {
                        err = errors.New(fmt.Sprintf("Expr `%T' evaluated to nil.", x.Expr))
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
                if a, err := l.expr(x.Name); err != nil {
                        return nil, err
                } else {
                        v = values.Barefile(a, x.Ext)
                }
        case *ast.Globfile:
                v = values.Globfile(x.Glob.Tok, x.Ext)
        case *ast.GlobExpr: // Just "*"
                v = values.Globfile(x.Tok, "")
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
                default: err := errors.New(fmt.Sprintf("Unsupported PathSeg `%v'.", x.Tok))
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
                        v = values.PercentPattern(a, b)
                }
        case *ast.RecipeDefineClause:
                //fmt.Printf("RecipeDefineClause: %s: %T %v\n", l.project.Name(), x.Sym, x.Sym)
                if name, err := l.expr(x.Name); err != nil {
                        return nil, err
                } else if name != nil {
                        def, _ := x.Sym.(*types.Def)
                        if def == nil {
                                err = errors.New(fmt.Sprintf("Symbol `%s' undefined in %v", def.Name(), def.Parent()))
                                return nil, err
                        }
                        
                        // TODO: check l.scope.Lookup(name.Strval()).(*Def)
                        if def.Name() != name.Strval() {
                                err = errors.New(fmt.Sprintf("Symbol `%s' differs from `%s'", name.Strval(), def.Name()))
                                return nil, err
                        }
                        
                        if o := def.Parent().Lookup(def.Name()); o != def {
                                err = errors.New(fmt.Sprintf("Symbol `%s' undefined in %v", def.Name(), def.Parent()))
                                return nil, err
                        }

                        v = def
                }
        }
        if v == nil {
                err = errors.New(fmt.Sprintf("Expr value is nil `%T'", expr))
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

func (l *Loader) useProject(pos token.Pos, project *types.Project) error {
        var entry *types.RuleEntry
        use := project.Scope().Lookup("use")
        if use == nil {
                return errors.New(fmt.Sprintf("Project `%v' has no 'use' package.", project.Name()))
        }
        if sn, ok := use.(*types.ScopeName); ok && sn != nil {
                // Get the 'use' rule entry in the 'use' scope. 
                if obj := sn.Scope().Lookup(":"); obj != nil {
                        //fmt.Printf("useProject: %T\n", obj)
                        if entry, _ = obj.(*types.RuleEntry); entry == nil {
                                return errors.New(fmt.Sprintf("Project `%v' has invalid 'use' entry (%T).", project.Name(), obj))
                        } else {
                                results, err := entry.Execute(l.scope)
                                if err != nil {
                                        return err
                                }
                                for _, result := range results {
                                        if result.Type() != types.ListType {
                                                continue
                                        }
                                        for _, elem := range result.(*types.List).Elems {
                                                def, ok := elem.(*types.Def)
                                                if !ok || def == nil {
                                                        continue
                                                }

                                                //fmt.Printf("%v: %v\n", l.project.Name(), elem)
                                                newd, alt := l.project.Scope().InsertDef(l.project, def.Name(), values.None)
                                                if alt != nil {
                                                        if d, _ := alt.(*types.Def); d == nil {
                                                                return errors.New(fmt.Sprintf("Name `%s' already taken in project `%s' (%T).", def.Name(), alt, l.project.Name()))
                                                        } else {
                                                                newd = d
                                                        }
                                                }
                                                if newd != nil {
                                                        // Append the delegate.
                                                        newd.Append(types.Delegate(def))
                                                }
                                        }
                                }
                                return nil
                        }
                } else {
                        //fmt.Printf("useProject: %v\n", sn.Scope())
                        // The 'use' rule entry is not defined.
                        return nil
                }
        }
        return errors.New(fmt.Sprintf("Project `%v' has invalid 'use' package (%T).", project.Name(), use))
}

func (l *Loader) useProjectName(pos token.Pos, pn *types.ProjectName) error {
        var (
                scope = l.project.Scope()
                project = pn.Project()
        )
        if project == nil {
                return errors.New(fmt.Sprintf("%v is nil", pn))
        }
        
        // FIXME: defined used project in represented order
        if sn, _ := scope.Lookup("use").(*types.ScopeName); sn != nil {
                if alt := sn.Scope().Insert(pn); alt != nil {
                        if alt.Type().Kind() == types.ProjectNameKind {
                                l.parseInfo(pos, "'%s' already used", pn.Name())
                        } else {
                                return errors.New(fmt.Sprintf("'%s' already defined in %s", pn.Name(), sn.Scope()))
                        }
                }
                if _, alt := sn.Scope().InsertDef(l.project, "*"/* use list */, pn); alt != nil {
                        if def, _ := alt.(*types.Def); def != nil {
                                def.Append(pn)
                        }
                }
        } else {
                return nil //errors.New(fmt.Sprintf("'use' scope is not in %s", scope))
        }
        return l.useProject(pos, project)
}

func (l *Loader) use(spec *ast.UseSpec) (err error) {
        var (
                name types.Value
                params []types.Value
        )
        if len(spec.Props) == 0 {
                return errors.New("Empty use spec.")
        } else if name, err = l.expr(spec.Props[0]); err != nil {
                return
        } else if name == nil {
                return errors.New("Undefined `use' target.")
        } else if  name == values.None {
                return errors.New("None `use' target.")
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
                        return errors.New(fmt.Sprintf("'%s' already defined in %s", t.Name(), scope))
                } else {
                        return nil // okay
                }
        case *types.RuleEntry:
                if alt := scope.Insert(t); alt != nil {
                        return errors.New(fmt.Sprintf("'%s' already defined in %s", t.Name(), scope))
                } else {
                        return nil // okay
                }
        }

        return errors.New(fmt.Sprintf("'%s' is not a usee (%T)", name, name))
}

func (l *Loader) eval(spec *ast.EvalSpec) (res types.Value, err error) {
        if num := len(spec.Props); num > 0 {
                var v types.Value
                if v, err = l.expr(spec.Props[0]); err != nil {
                        return
                }
                switch op := v.(type) {
                case types.Caller:
                        if a, err := l.exprs(spec.Props[1:]); err != nil {
                                return nil, err
                        } else {
                                res, _ = op.Call(a...)
                        }
                default:
                        if _, obj := l.scope.Find(op.Strval()); obj != nil {
                                if f, _ := obj.(types.Caller); f != nil {
                                        if a, err := l.exprs(spec.Props[1:]); err != nil {
                                                return nil, err
                                        } else {
                                                res, err = f.Call(a...)
                                        }
                                }
                        } else {
                                err = errors.New(fmt.Sprintf("Eval undefined `%s'", op.Strval()))
                        }
                }
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
                        err = errors.New(fmt.Sprintf("Invalid depend type `%T'.", depend))
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
                        err = errors.New(fmt.Sprintf("Undefined program scope (%T).", p.Scope))
                        return
                }
                if p.Recipes != nil {
                        if recipes, err = l.exprs(p.Recipes); err != nil {
                                return
                        }
                }
                params = p.Params
        } else {
                err = errors.New(fmt.Sprintf("unsupported program type (%T)", clause.Program))
                return
        }
        
        var modifiers []types.Value
        if clause.Modifier != nil {
                modifiers, err = l.exprs(clause.Modifier.Elems)
                if err != nil {
                        return
                }
        }
        
        var prog = l.NewProgram(l.project, params, progScope, depends, recipes...)
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }

        if targets, err = l.exprs(clause.Targets); err != nil {
                return
        }
        
        var name string
        for n, target := range targets {
                if target == nil {
                        err = errors.New(fmt.Sprintf("nil target (%T)", clause.Targets[n]))
                        return
                }

                class := types.GeneralRuleEntry
                if name = target.Strval(); name == "use" {
                        if n == 0 && len(clause.Targets) == 1 {
                                class = types.UseRuleEntry
                        } else {
                                l.parseWarn(clause.Targets[n].Pos(), "'use' rule mixed with other targets")
                                err = errors.New(fmt.Sprintf("mixes 'use' and normal targets"))
                                return
                        }
                } else if l.project.IsFile(name) {
                        class = types.FileRuleEntry
                }
                
                switch namv := target.(type) {
                case *types.PercentPattern:
                        l.project.SetPercentPatternProgram(namv, class, prog)
                default:
                        if _, err = l.project.SetProgram(name, class, prog); err != nil {
                                return
                        }
                }
        }
        return
}

func (l *Loader) include(spec *ast.IncludeSpec) error {
        var (
                linfo = l.loads[len(l.loads)-1]
                specVal, err = l.expr(spec.Props[0])
                params []types.Value
        )
        if err != nil {
                return err
        }

        if len(spec.Props) > 1 {
                params, err = l.exprs(spec.Props[1:])
                if err != nil {
                        return err
                }
        }

        var (
                specName = specVal.Strval()
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
        l.scope = types.NewScope(l.scope, comment)
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
                err = errors.New(fmt.Sprintf("bad runtime scope (%T)", as))
        }
        return
}

func (l *Loader) loadProjectBases(linfo *loadinfo, params types.Value) (err error) {
        if params == nil {
                return
        }
        
        g, _ := params.(*types.Group)
        if g == nil {
                err = errors.New(fmt.Sprintf("invalid parameters (%T)", params))
                return
        }

        var (
                absPath, specName string
                isDir bool
        )
        ParamsLoop: for _, elem := range g.Elems {
                specName = elem.Strval()
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
                        err = errors.New(fmt.Sprintf("Project `%v' not loaded. (%T, %s)", elem, elem, absPath))
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
                return errors.New(fmt.Sprintf("already in project %s", l.project.Name()))
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
                        use = types.NewScope(s, "use")
                )
                if obj, alt := s.InsertScopeName(p, "use", use); alt != nil {
                        if _, ok := alt.(*types.ScopeName); !ok {
                                err = errors.New(fmt.Sprintf("Name `use' already taken (%s).", s))
                                return
                        }
                } else if obj == nil {
                        err = errors.New("Failed adding `use' scope.")
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
                                err = errors.New(fmt.Sprintf("Name `%s' already taken (%T).", name, a))
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
                return errors.New(fmt.Sprintf("no loaded project %s", name))
        }
        if l.project == nil {
                return errors.New("no current project")
        } else if s := l.project.Name(); s != name {
                return errors.New(fmt.Sprintf("current project is %s but %s", s, name))
        } else if l.project != dec.project {
                return errors.New(fmt.Sprintf("project conflicts (%s, %s)", l.project.Name(), dec.project.Name()))
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
                return errors.New(fmt.Sprintf("Invalid abs name `%s' (%s).", absPath, specName))
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
                                return errors.New(fmt.Sprintf("Name `%s' already taken (%T).", name, a))
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
                err = errors.New(fmt.Sprintf("Invalid abs name `%s' (%s).", absDir, specName))
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
                                err = errors.New(fmt.Sprintf("Name `%s' already taken (%T).", name, a))
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

func (pc *parseContext) Files(m map[string][]string) {
        pc.project.AddFiles(m)
}

func (pc *parseContext) IsFileName(s string) bool {
        if pc.project != nil {
                //fmt.Printf("IsFileName: %s %v\n", s, pc.project.IsFile(s))
                return pc.project.IsFile(s)
        }
        return false
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
                scope = types.NewScope(outer, comment)
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

func (pc *parseContext) ClauseImport(spec *ast.ImportSpec) error {
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
        
func (pc *parseContext) Rule(clause *ast.RuleClause) (parser.RuntimeObj, error) {
        return nil, pc.rule(clause)
}

func (pc *parseContext) Eval(x ast.Expr, ec parser.EvalBits) (res types.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
                        if fault := runtime.GoFault(e); fault != nil {
                                err = fault
                        } else if err, _ = e.(error); err == nil {
                                err = errors.New(fmt.Sprintf("%v", e))
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
        if ec&parser.CastDepends == 0 {
                return
        }

        //fmt.Printf("Reveal: depend: %T %v (%v)\n", res, res, res.Strval())
        
        // Cast depends so that it's could be easily used.

        if def, ok := res.(*types.Def); ok && def != nil {
                res = def.Value
        }

        /*switch res.Type() {
        case types.BarewordType:
        case types.BarefileType:
        case types.PathType:
        case types.PatternType:
        case types.ClosureType:
        case types.ListType:
                // TODO: check list elements type?
        case types.RuleEntryType: // e.g. other.entry
        case types.ProjectNameType: // e.g. use.*
        default:
                res, err = nil, errors.New(fmt.Sprintf("Unsupported depend type '%v' (%T) (%v).", res.Type(), res, res))
        }*/
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
