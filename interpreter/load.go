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
        useScopeName = "~usee~"
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

/*
func defSet(op token.Token, def *types.Def, value types.Value) (err error) {
        //fmt.Printf("defSet: %v %T %v %v\n", def.Name(), def.Value(), op, value)
        switch op {
        case token.QUE_ASSIGN: // ?=
                if def.Value == values.None {
                        def.Set(value)
                } else {
                        // noop, only set if absent (not defined)
                }
        case token.ADD_ASSIGN: // +=
                var (
                        l []types.Value
                        v = def.Value
                )
                if a, ok := v.(*types.List); ok {
                        l = append(l, a.Slice(0)...)
                } else {
                        l = append(l, v)
                }
                if a, ok := value.(*types.List); ok {
                        l = append(l, a.Slice(0)...)
                } else {
                        l = append(l, value)
                }
                if len(l) == 1 {
                        def.Set(l[0])
                } else {
                        def.Set(values.List(l...))
                }
        case token.EXC_ASSIGN: // !=
                var (
                        source = value.Strval()
                        sh = exec.Command("sh", "-c", source)
                        stdout bytes.Buffer
                        stderr bytes.Buffer
                )
                sh.Stdout, sh.Stderr = &stdout, &stderr
                if err = sh.Run(); err == nil {
                        def.Set(values.String(strings.TrimSpace(stdout.String())))
                } else {
                        def.Set(values.None)
                        //fmt.Printf("%v\n", err)
                        //err = nil // ignore the error
                }
        case token.SCO_ASSIGN, token.DCO_ASSIGN:
                // TODO: 'expand' all calls?
                def.Set(value)
        case token.ASSIGN: // =
                def.Set(value)
        default:
                runtime.Fail("unknown set operation %v\n", op)
        }
        return
}

func set(p *types.Project, op token.Token, name string, value types.Value) (def *types.Def, err error) {
        // See https://www.gnu.org/software/make/manual/html_node/Setting.html
        var (
                scope = p.Scope()
                obj = scope.Lookup(name) // Only lookup the project's scope!
        )
        if obj == nil {
                var alt types.Object
                if obj, alt = scope.InsertDef(p, name, values.None); alt != nil {
                        unreachable()
                }
        }
        if def, _ = obj.(*types.Def); def == nil {
                err = errors.New(fmt.Sprintf("name '%s' already taken in '%s'", name, p.Name()))
                return
        }

        if err = defSet(op, def, value); err != nil {
                //i.parseWarn()
        }
        return
} */

func (i *Interpreter) parseInfo(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseInfo(pos, s, a...)
}

func (i *Interpreter) parseWarn(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseWarn(pos, s, a...)
}

func (i *Interpreter) parseFail(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseWarn(pos, s, a...)
        runtime.Fail("parse failed")
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
                if sn, _ := scope.Lookup(useScopeName).(*types.ScopeName); sn != nil {
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
                i.parseWarn(x.Name.Pos(), fmt.Sprintf("unresolved %s", name))
                err = errors.New(fmt.Sprintf("unresolved %s", name))
                return
        }

        def, _ := x.Resolved.(*types.Def)
        if def == nil {
                i.parseWarn(x.Pos(), fmt.Sprintf("uncallable resolved %s (%T)", name, x.Resolved))
                err = errors.New(fmt.Sprintf("uncallable resolved %s", name))
                return
        } else {
                obj = def
        }

        for _, x := range x.Args {
                if a,e := i.expr(x); e != nil {
                        i.parseWarn(x.Pos(), fmt.Sprintf("invalid closure arg %T (%v)", a, e))
                        return nil, nil, e
                } else if a == nil {
                        i.parseWarn(x.Pos(), fmt.Sprintf("nil closure arg (%T)", x, e))
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
                switch t := x.Elems[0].(type) {
                case *ast.UseDefineClause:
                default: 
                        s := fmt.Sprintf("unsupported recipe command (%T)", t)
                        return nil, errors.New(s)
                }
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
                        err = errors.New("evaluated expr has nil data")
                        return
                }
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
                        v = values.PercentPattern(i.project, a, b)
                }
        case *ast.UseDefineClause:
                //fmt.Printf("UseDefineClause: %T %v\n", x.Sym, x.Sym)
                if d, _ := x.Sym.(*types.Def); d != nil {
                        // !strings.HasPrefix(s, "rule use")
                        if s := d.Parent().Comment(); s != "rule use" {
                                err = errors.New(fmt.Sprintf("not a 'use' scope (%s)", s))
                                return
                        } else if o := d.Parent().Lookup(d.Name()); o == nil {
                                err = errors.New(fmt.Sprintf("'%s' undefined in %v", d.Name(), d.Parent()))
                                return
                        }
                }
                
                if name, err := i.expr(x.Name); err != nil {
                        return nil, err
                } else if val, err := i.expr(x.Value); err != nil {
                        return nil, err
                } else if name != nil && val != nil {
                        // TODO: check i.scope.Lookup(name.Strval()).(*Def)
                        v = types.MakeDefiner(x.Tok, name.Strval())
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
        use := project.Scope().Lookup("use")
        if rule, _ := use.(*types.RuleEntry); rule != nil {
                result, err := rule.Call(values.Any(i.project))
                //i.parseInfo(pos, "use: %v: %v (%v)\n", i.project.Name(), project.Name(), result)
                if err != nil {
                        return err
                } else if result == nil {
                        // ...
                }
        } else if false {
                i.parseInfo(pos, "nil use rule of '%s' (%T %v)\n", project.Name(), use, use)
        }
        return nil
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
        if sn, _ := scope.Lookup(useScopeName).(*types.ScopeName); sn != nil {
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
                return errors.New("empty use spec")
        } else if name, err = i.expr(spec.Props[0]); err != nil {
                return
        } else if name == nil {
                return errors.New("undefined use target")
        } else if  name == values.None {
                return errors.New("none use target")
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
                                err = errors.New(fmt.Sprintf("undefined '%s'", op.Strval()))
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
                        i.parseWarn(depend.Pos(), "invalid depend (%T %v -> %T %v)", depend, depend, depval, depval)
                        err = errors.New(fmt.Sprintf("invalid depend"))
                        return
                } else if l, _ := depval.(*types.List); l != nil {
                        depends = append(depends, l.Elems...)
                } else {
                        depends = append(depends, depval)
                }
        }

        if p, ok := clause.Program.(*ast.ProgramExpr); ok && p != nil {
                if progScope, _ = p.Scope.(*types.Scope); progScope == nil {
                        err = errors.New(fmt.Sprintf("undefined program scope (%T)", p.Scope))
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
                /* if k := elem.Type().Kind(); k != types.InvalidKind &&
                        k <= types.ListKind {
                        args = append(args, elem)
                        continue ParamsLoop
                } */
                
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
                i.project.Chain(loaded)

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
                        use = types.NewScope(s, token.NoPos, token.NoPos, useScopeName)
                )
                if _, alt := s.InsertScopeName(p, useScopeName, use); alt != nil {
                        i.parseFail(ident.Pos(), "name '%s' already taken in %s", useScopeName, s)
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
                                i.parseFail(ident.Pos(), "name '%s' already taken (%T)", name, a)
                                err = errors.New(fmt.Sprintf("name '%s' already taken (%T)", name, a))
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
                                return errors.New(fmt.Sprintf("name '%s' already taken (%T)", name, a))
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
                                err = errors.New(fmt.Sprintf("name '%s' already taken (%T)", name, a))
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
                        if err, _ = e.(error); err == nil {
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
        switch res.Type() {
        //case types.RuleEntryType:
        //case types.BarewordType:
        case types.BarefileType:
        case types.PathType:
        case types.PatternType:
        case types.ClosureType:
        default:
                res, err = nil, errors.New(fmt.Sprintf("unsupported depend type '%v' (%T %v)", res.Type(), res, res))
        }
        return
}

func (pc *parseContext) Resolve(name string) parser.RuntimeObj {
        if pc.scope != nil {
                obj := pc.scope.Find(name)
                if obj != nil {
                        return obj.(parser.RuntimeObj)
                }
                if obj == nil && name == "use" {
                        obj = pc.scope.Find(useScopeName)
                }
                if obj == nil {
                        // TODO: add this search path into Scope.Find
                        /*if obj = pc.project.Scope().Find(name); obj != nil {
                             return
                        }*/
                        for _, base := range pc.project.Bases() {
                                if obj = base.Scope().Find(name); obj != nil {
                                        return obj.(parser.RuntimeObj)
                                }
                        }
                }
                if obj == nil && (name == "^") {
                        fmt.Printf("Resolve: '%v' not in %v\n", name, pc.scope)
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
        case types.DefType/*, types.DefinerType*/:
                scope := pc.scope // always in the current scope
                obj, alt = scope.InsertDef(pc.project, name, values.None)
        case types.RuleEntryType:
                scope := pc.project.Scope() // always in the project
                obj, alt = scope.InsertEntry(pc.project, types.GeneralRuleEntry, name)
        }
        return
}
