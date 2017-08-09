//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package interpreter

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

func restoreLoadingInfo(i *Interpreter) {
        var (
                last = len(i.loads)-1
                linfo = i.loads[last]
        )

        i.loads = i.loads[0:last]
        i.project = linfo.loader
        i.scope = linfo.scope //i.SetScope(linfo.scope)

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

func saveLoadingInfo(i *Interpreter, specName, absDir, baseName string) *Interpreter {
        //absDir, baseName := filepath.Split(filepath.Clean(absPath))
        i.loads = append(i.loads, &loadinfo{
                absDir: absDir,
                baseName: baseName,
                specName: filepath.Clean(specName),
                loader:   i.project,
                scope:    i.scope, //Scope(),
                declares: make(map[string]*declare),
        })
        return i
}

func (i *Interpreter) parseInfo(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseInfo(pos, s, a...)
}

func (i *Interpreter) parseWarn(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseWarn(pos, s, a...)
}

func (i *Interpreter) searchSpecPath(linfo *loadinfo, specName string) (absPath string, isDir bool, err error) {
        var fi os.FileInfo
        if abs := filepath.IsAbs(specName); abs || 
                strings.HasPrefix(specName, "../") ||
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
                for _, base := range i.paths {
                        s := filepath.Join(base, specName)
                        if fi, err = os.Stat(s); err == nil && fi != nil {
                                isDir, absPath = fi.IsDir(), s
                                return
                        }
                }
        }
        return
}

func (i *Interpreter) loadImportSpec(spec *ast.ImportSpec) (err error) {
        var (
                //scope = i.Scope()
                linfo = i.loads[len(i.loads)-1]
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
        if absPath, isDir, err = i.searchSpecPath(linfo, specName); err != nil {
                return
        } else if absPath == "" {
                i.parseWarn(spec.Pos(), "missing '%s' (in %v)", specName, i.paths)
                return errors.New(fmt.Sprintf("'%s' not found", specName))
        }

        //fmt.Printf("import: %s (%s,dir=%v) (%v)\n", specName, absPath, isDir, i.project.Name())

        if isDir {
                err = i.loadDir(specName, absPath, nil)
        } else {
                err = i.load(specName, absPath, nil)
        }
        if err != nil || nouse {
                return
        }
        
        if loaded, _ := i.loaded[absPath]; loaded != nil {
                scope := i.project.Scope()
                pn, _ := scope.Lookup(loaded.Name()).(*types.ProjectName)
                if pn == nil {
                        i.parseWarn(spec.Pos(), "%v (%v,dir=%v) not in %v", specName, absPath, isDir, scope)
                        return errors.New(fmt.Sprintf("'%s' not found (%s)", specName, loaded.Name()))
                }
                if sn, _ := scope.Lookup("use").(*types.ScopeName); sn != nil {
                        if alt := sn.Scope().Insert(pn); alt != nil {
                                return errors.New(fmt.Sprintf("'%s' already defined in %v", specName, sn.Scope()))
                        }
                        if _, alt := sn.Scope().InsertDef(i.project, "*"/* use list */, pn); alt != nil {
                                if def, _ := alt.(*types.Def); def != nil {
                                        def.Append(pn)
                                }
                        }
                } else {
                        return errors.New(fmt.Sprintf("'use' scope is not in %s", scope))
                }
                err = i.useProject(spec.Props[0].Pos(), loaded)
        } else {
                fmt.Printf("%v (%v)\n", specName, absPath)
                for k, v := range i.loaded {
                        fmt.Printf("  loaded: %v (%v)\n", v.Name(), k)
                }
                unreachable()
        }
        return
}

func (i *Interpreter) closuredelegate(x *ast.ClosureDelegate) (obj types.Object, args []types.Value, err error) {
        name, err := i.expr(x.Name)
        if err != nil {
                return nil, nil, err
        }
        if x.Resolved == nil {
                //i.parseWarn(x.Name.Pos(), fmt.Sprintf("Unresolved reference `%s'.", name))
                err = errors.New(fmt.Sprintf("Unresolved reference `%s'.", name))
                return
        }

        def, _ := x.Resolved.(types.Caller)
        if def == nil {
                //i.parseWarn(x.Pos(), fmt.Sprintf("uncallable resolved %s (%T)", name, x.Resolved))
                err = errors.New(fmt.Sprintf("Uncallable `%s' resolved (%T).", name, x.Resolved))
                return
        } else if obj = def.(types.Object); obj == nil {
                err = errors.New(fmt.Sprintf("Non-object callable `%s' resolved (%T).", name, def))
                return
        }

        for _, x := range x.Args {
                if a,e := i.expr(x); e != nil {
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

func (i *Interpreter) closure(x *ast.ClosureExpr) (types.Value, error) {
        if obj, args, err := i.closuredelegate(&x.ClosureDelegate); err == nil {
                return types.Closure(obj, args...), nil
        } else {
                return nil, err
        }
}

func (i *Interpreter) delegate(x *ast.DelegateExpr) (v types.Value, err error) {
        if obj, args, err := i.closuredelegate(&x.ClosureDelegate); err == nil {
                return types.Delegate(obj, args...), nil
        } else {
                return nil, err
        }
}

func (i *Interpreter) recipe(x *ast.RecipeExpr) (v types.Value, err error) {
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
                if a, err := i.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        elems = append(elems, a...)
                        v = values.List(elems...)
                }
        } else if a, err := i.exprs(x.Elems); err != nil {
                return nil, err
        } else {
                v = values.Compound(a...)
        }
        return
}

func (i *Interpreter) expr(expr ast.Expr) (v types.Value, err error) {
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
                if av.Value, err = i.expr(x.X); err != nil {
                        return
                }
                if av.Args, err = i.exprs(x.Arguments); err != nil {
                        return
                }
                v = av
        case *ast.ClosureExpr:
                v, err = i.closure(x)
        case *ast.DelegateExpr:
                v, err = i.delegate(x)
        case *ast.RecipeExpr:
                v, err = i.recipe(x)
        case *ast.BasicLit:
                v = values.Literal(x.Kind, x.Value)
        case *ast.Bareword:
                v = values.Bareword(x.Value)
        case *ast.Barecomp:
                if a, err := i.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.Barecomp(a...)
                }
        case *ast.Barefile:
                if a, err := i.expr(x.Name); err != nil {
                        return nil, err
                } else {
                        v = values.Barefile(a, x.Ext)
                }
        case *ast.PathExpr:
                if a, err := i.exprs(x.Segments); err != nil {
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
                if a, err := i.expr(x.Name); err != nil {
                        return nil, err
                } else {
                        v = values.Flag(a)
                }
        case *ast.CompoundLit:
                if a, err := i.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.Compound(a...)
                }
        case *ast.GroupExpr:
                if a, err := i.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.Group(a...)
                }
        case *ast.ListExpr:
                if a, err := i.exprs(x.Elems); err != nil {
                        return nil, err
                } else {
                        v = values.List(a...)
                }
        case *ast.KeyValueExpr:
                var a, b types.Value
                if b, err = i.expr(x.Value); err != nil {
                        return
                } else if a, err = i.expr(x.Key); err != nil {
                        return
                } else {
                        v = values.Pair(a, b)
                }
        case *ast.PercExpr:
                var a, b types.Value
                if a, err = i.expr(x.X); err != nil {
                        return
                } else if b, err = i.expr(x.Y); err != nil {
                        return
                } else {
                        v = values.PercentPattern(a, b)
                }
        case *ast.RecipeDefineClause:
                //fmt.Printf("RecipeDefineClause: %s: %T %v\n", i.project.Name(), x.Sym, x.Sym)
                if name, err := i.expr(x.Name); err != nil {
                        return nil, err
                } else if name != nil {
                        def, _ := x.Sym.(*types.Def)
                        if def == nil {
                                err = errors.New(fmt.Sprintf("Symbol `%s' undefined in %v", def.Name(), def.Parent()))
                                return nil, err
                        }
                        
                        // TODO: check i.scope.Lookup(name.Strval()).(*Def)
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
        return
}

func (i *Interpreter) exprs(exprs []ast.Expr) (values []types.Value, err error) {
        for _, x := range exprs {
                if v, err := i.expr(x); err != nil {
                        return nil, err
                } else {
                        values = append(values, v)
                }
        }
        return
}

func (i *Interpreter) useProject(pos token.Pos, project *types.Project) error {
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
                                results, err := entry.ExecutePrograms(/*values.Any(i.project)*/)
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

                                                //fmt.Printf("%v: %v\n", i.project.Name(), elem)
                                                newd, alt := i.project.Scope().InsertDef(i.project, def.Name(), values.None)
                                                if alt != nil {
                                                        if d, _ := alt.(*types.Def); d == nil {
                                                                return errors.New(fmt.Sprintf("Name `%s' already taken in project `%s' (%T).", def.Name(), alt, i.project.Name()))
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

func (i *Interpreter) useProjectName(pos token.Pos, pn *types.ProjectName) error {
        var (
                scope = i.project.Scope()
                project = pn.Project()
        )
        if project == nil {
                return errors.New(fmt.Sprintf("%v is nil", pn))
        }
        
        // FIXME: defined used project in represented order
        if sn, _ := scope.Lookup("use").(*types.ScopeName); sn != nil {
                if alt := sn.Scope().Insert(pn); alt != nil {
                        if alt.Type().Kind() == types.ProjectNameKind {
                                i.parseInfo(pos, "'%s' already used", pn.Name())
                        } else {
                                return errors.New(fmt.Sprintf("'%s' already defined in %s", pn.Name(), sn.Scope()))
                        }
                }
                if _, alt := sn.Scope().InsertDef(i.project, "*"/* use list */, pn); alt != nil {
                        if def, _ := alt.(*types.Def); def != nil {
                                def.Append(pn)
                        }
                }
        } else {
                return errors.New(fmt.Sprintf("'use' scope is not in %s", scope))
        }
        return i.useProject(pos, project)
}

func (i *Interpreter) use(spec *ast.UseSpec) (err error) {
        var (
                name types.Value
                params []types.Value
        )
        if len(spec.Props) == 0 {
                return errors.New("Empty use spec.")
        } else if name, err = i.expr(spec.Props[0]); err != nil {
                return
        } else if name == nil {
                return errors.New("Undefined `use' target.")
        } else if  name == values.None {
                return errors.New("None `use' target.")
        }
        for _, prop := range spec.Props[1:] {
                if v, err := i.expr(prop); err != nil {
                        return err
                } else {
                        params = append(params, v)
                }
        }

        var scope = i.project.Scope()
        switch t := name.(type) {
        case *types.ProjectName:
                return i.useProjectName(spec.Props[0].Pos(), t)
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

func (i *Interpreter) eval(spec *ast.EvalSpec) (res types.Value, err error) {
        if num := len(spec.Props); num > 0 {
                var v types.Value
                if v, err = i.expr(spec.Props[0]); err != nil {
                        return
                }
                switch op := v.(type) {
                case types.Caller:
                        if a, err := i.exprs(spec.Props[1:]); err != nil {
                                return nil, err
                        } else {
                                res, _ = op.Call(a...)
                        }
                default:
                        if _, obj := i.scope.FindAt(spec.EndPos, op.Strval()); obj != nil {
                                if f, _ := obj.(types.Caller); f != nil {
                                        if a, err := i.exprs(spec.Props[1:]); err != nil {
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

func (i *Interpreter) rule(clause *ast.RuleClause) (err error) {
        var (
                targets []types.Value
                depends []types.Value
                recipes []types.Value
                depval types.Value
                progScope *types.Scope
                params []string
        )
        for _, depend := range clause.Depends {
                if depval, err = i.expr(depend); err != nil {
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
                        if recipes, err = i.exprs(p.Recipes); err != nil {
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
                modifiers, err = i.exprs(clause.Modifier.Elems)
                if err != nil {
                        return
                }
        }
        
        var prog = i.NewProgram(i.project, params, progScope, depends, recipes...)
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }

        if targets, err = i.exprs(clause.Targets); err != nil {
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
                                i.parseWarn(clause.Targets[n].Pos(), "'use' rule mixed with other targets")
                                err = errors.New(fmt.Sprintf("mixes 'use' and normal targets"))
                                return
                        }
                } else if i.project.IsFile(name) {
                        class = types.FileRuleEntry
                }
                
                switch namv := target.(type) {
                case *types.PercentPattern:
                        i.project.SetPercentPatternProgram(namv, class, prog)
                default:
                        if _, err = i.project.SetProgram(name, class, prog); err != nil {
                                return
                        }
                }
        }
        return
}

func (i *Interpreter) include(spec *ast.IncludeSpec) error {
        var (
                linfo = i.loads[len(i.loads)-1]
                specVal, err = i.expr(spec.Props[0])
                params []types.Value
        )
        if err != nil {
                return err
        }

        if len(spec.Props) > 1 {
                params, err = i.exprs(spec.Props[1:])
                if err != nil {
                        return err
                }
        }

        var (
                specName = specVal.Strval()
                jointPath = filepath.Join(linfo.absDir, specName)
                absDir, baseName = filepath.Split(jointPath)
        )
        defer restoreLoadingInfo(saveLoadingInfo(i, specName, absDir, baseName))
        
        _, err = i.pc.ParseFile(i.fset, jointPath, nil, parseMode|parser.Flat)
        if err != nil {
                return err
        }

        if len(params) > 0 {
                // TODO: parsing parameters
        }
        return nil
}

func (i *Interpreter) openScope(pos token.Pos, comment string) ast.Scope {
        i.scope = types.NewScope(i.scope, pos, token.NoPos, comment)
        return i.scope
}

func (i *Interpreter) closeScope(as ast.Scope) (err error) {
        if scope, ok := as.(*types.Scope); ok {
                i.scope = scope.Outer()
        } else {
                err = errors.New(fmt.Sprintf("bad runtime scope (%T)", as))
        }
        return
}

func (i *Interpreter) loadProjectBases(linfo *loadinfo, params types.Value) (err error) {
        if params == nil {
                return
        }
        
        g, _ := params.(*types.Group)
        if g == nil {
                err = errors.New(fmt.Sprintf("invalid parameters (%T)", params))
                return
        }

        var (
                //args []types.Value
                absPath, specName string
                isDir bool
        )
        ParamsLoop: for _, elem := range g.Elems {
                specName = elem.Strval()
                absPath, isDir, err = i.searchSpecPath(linfo, specName)
                if err != nil {
                        break ParamsLoop
                }
                
                if isDir {
                        err = i.loadDir(specName, absPath, nil)
                } else {
                        err = i.load(specName, absPath, nil)
                }
                if err != nil {
                        break ParamsLoop
                }

                loaded, _ := i.loaded[absPath]
                if loaded == nil {
                        err = errors.New(fmt.Sprintf("Project `%v' not loaded.", specName))
                } else {
                        i.project.Chain(loaded)
                }

                //fmt.Printf("base: %s (%s %v)\n", specName, loaded.Name(), absPath)
        }
        return
}

func (i *Interpreter) declareProject(ident *ast.Bareword, params types.Value) (err error) {
        var name = ident.Value
        if i.project != nil && i.project.Name() == ident.Value {
                return errors.New(fmt.Sprintf("already in project %s", i.project.Name()))
        }

        var (
                linfo = i.loads[len(i.loads)-1]
                dec, ok = linfo.declares[name]
        )
        //fmt.Printf("declareProject: %v (%v) %v, %v\n", ident.Value, linfo.absPath(), i.scope, i.project)
        if !ok {
                var (
                        absDir = linfo.absDir
                        relPath string
                )
                if !filepath.IsAbs(absDir) {
                        //absDir = filepath.Join(i.Getwd(), absDir)
                        absDir, _ = filepath.Abs(absDir)
                }
                relPath, _ = filepath.Rel(i.Getwd(), absDir)

                dec = &declare{
                        project: i.Globe().NewProject(i.scope, absDir, relPath, linfo.specName, name),
                }
                
                i.loaded[linfo.absPath()] = dec.project
                linfo.declares[name] = dec

                var (
                        p = dec.project
                        s = p.Scope()
                        use = types.NewScope(s, token.NoPos, token.NoPos, "use")
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

        //fmt.Printf("declareProject: %v (%v) (%p -> %p)\n", ident.Value, linfo.absPath(), i.project, dec.project)

        if loader := linfo.loader; loader != nil {
                //fmt.Printf("DeclareProject: %s.%s %v\n", loader.Name(), dec.project.Name(), dec.backscope)
                
                if !strings.HasPrefix(loader.Scope().Comment(), "project \"") {
                        i.parseWarn(ident.Pos(), "'%s' not loaded from project scope", name)
                }

                if _, a := loader.Scope().InsertProjectName(loader, name, dec.project); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                err = errors.New(fmt.Sprintf("Name `%s' already taken (%T).", name, a))
                                return
                        }
                }

                //fmt.Printf("DeclareProject: %s.%s %v\n", loader.Name(), name, loader.Scope())
        }

        dec.backproj = i.project
        dec.backscope = i.scope
        i.project = dec.project
        i.scope = i.project.Scope()

        err = i.loadProjectBases(linfo, params)
        return
}

func (i *Interpreter) closeCurrentProject(ident *ast.Bareword) (err error) {
        var (
                name = ident.Value
                linfo = i.loads[len(i.loads)-1]
                dec, ok = linfo.declares[name]
        )
        if dec == nil || !ok {
                return errors.New(fmt.Sprintf("no loaded project %s", name))
        }
        if i.project == nil {
                return errors.New("no current project")
        } else if s := i.project.Name(); s != name {
                return errors.New(fmt.Sprintf("current project is %s but %s", s, name))
        } else if i.project != dec.project {
                return errors.New(fmt.Sprintf("project conflicts (%s, %s)", i.project.Name(), dec.project.Name()))
        }

        //fmt.Printf("closeCurrentProject: %v (%v) (%p -> %p)\n", ident.Value, linfo.absPath(), i.project, dec.backproj)
        
        i.scope = dec.backscope
        i.project = dec.backproj
        return
}

// Interpreter.Load loads script from a file or source code (string, []byte).
func (i *Interpreter) load(specName, absPath string, source interface{}) error {
        //fmt.Printf("load: %v (%v)\n", specName, absPath)
        
        // Check already project.
        if loaded, ok := i.loaded[absPath]; ok {
                var (
                        s = i.project.Scope()
                        name = loaded.Name()
                )
                if _, a := s.InsertProjectName(i.project, name, loaded); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                return errors.New(fmt.Sprintf("Name `%s' already taken (%T).", name, a))
                        }
                }
                return nil
        }
        
        var absDir, baseName = filepath.Split(absPath)
        defer restoreLoadingInfo(saveLoadingInfo(i, specName, absDir, baseName))

        doc, err := i.pc.ParseFile(i.fset, absPath, source, parseMode)
        if err != nil {
                return err
        }
        if doc == nil {
                // FIXME: ...
        }

        //fmt.Printf("Load: %v %v\n", absPath, doc.Name.Name)
        return nil
}

func (i *Interpreter) loadDir(specName, absDir string, filter func(os.FileInfo) bool) (err error) {
        //fmt.Printf("loaddir: %v (%v)\n", specName, absDir)

        // Check already project.
        if loaded, ok := i.loaded[absDir]; ok {
                var (
                        s = i.project.Scope()
                        name = loaded.Name()
                )
                if _, a := s.InsertProjectName(i.project, name, loaded); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                err = errors.New(fmt.Sprintf("Name `%s' already taken (%T).", name, a))
                        }
                }
                return nil
        }

        defer restoreLoadingInfo(saveLoadingInfo(i, specName, absDir, ""))

        mods, err := i.pc.ParseDir(i.fset, absDir, filter, parseMode)
        if err == nil && mods != nil {
        }

        //fmt.Printf("LoadDir: %v %v\n", absDir, mods)
        return
}

func (i *Interpreter) Load(filename string, source interface{}) error {
        dir, _ := filepath.Split(filename)
        if dir == "" { dir = "." }
        return i.load(dir, filename, source)
}

func (i *Interpreter) LoadDir(path string, filter func(os.FileInfo) bool) (err error) {
        return i.loadDir(path, path, filter)
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

func (pc *parseContext) OpenScope(pos token.Pos, comment string) ast.Scope {
        return pc.openScope(pos, comment)
}

func (pc *parseContext) CloseScope(as ast.Scope) error {
        return pc.closeScope(as)
}

func (pc *parseContext) WithScope(scope ast.Scope, f func() error) error {
        restore := pc.scope
        pc.scope = scope.(*types.Scope)
        defer func() {
                pc.scope = restore
        }()
        return f()
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
                if res = types.Eval(res); res == nil {
                        return
                }
        }
        if ec&parser.CastDepends == 0 {
                return
        }

        //fmt.Printf("Eval: depend: %T %v (%v)\n", res, res, res.Strval())
        
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
                obj := pc.Globe().Scope().Find(name)
                if obj != nil {
                        return obj.(parser.RuntimeObj)
                }
        }
        if bits&parser.FromBase != 0 && pc.project != nil {
                for _, base := range pc.project.Bases() {
                        if obj := base.Scope().Find(name); obj != nil {
                                return obj.(parser.RuntimeObj)
                        }
                }
        }
        if bits&parser.FromProject != 0 && pc.project != nil {
                obj := pc.project.Scope().Find(name)
                if obj != nil {
                        return obj.(parser.RuntimeObj)
                }
        }
        if bits&parser.FromHere != 0 && pc.scope != nil {
                obj := pc.scope.Find(name)
                if obj != nil {
                        return obj.(parser.RuntimeObj)
                }
                if obj == nil && name != "use" {
                        // TODO: add this search path into Scope.Find
                        for _, base := range pc.project.Bases() {
                                if obj = base.Scope().Find(name); obj != nil {
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
        }
        return
}
