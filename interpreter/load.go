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
        "os/exec"
        "strings"
        "errors"
        "bytes"
        "fmt"
        "os"
)

const (
        useScopeName = "~usee~"
        useRuleName = "~use~"
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

func defSet(op token.Token, def *types.Def, value types.Value) (err error) {
        //fmt.Printf("defSet: %v %T %v %v\n", def.Name(), def.Value(), op, value)
        switch op {
        case token.QUE_ASSIGN: // ?=
                if def.Value() == values.None {
                        def.Set(value)
                } else {
                        // noop, only set if absent (not defined)
                }
        case token.ADD_ASSIGN: // +=
                var (
                        l []types.Value
                        v = def.Value()
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
                        source = value.String()
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
}

// TODO: move it into 'runtime' package
type usedefiner struct {
        op token.Token
        name string
        value types.Value
        pos *token.Position
        types.None
}
func (p *usedefiner) Pos() *token.Position { return p.pos }
func (p *usedefiner) Type() types.Type     { return p.value.Type() }
func (p *usedefiner) Lit() string          { return p.name + " = " + p.value.Lit() }
func (p *usedefiner) String() string       { return p.name + " = " + p.value.String() }
func (p *usedefiner) Integer() int64       { return 0 }
func (p *usedefiner) Float() float64       { return 0 }
func (p *usedefiner) Define(project *types.Project) (result types.Value, err error) {
        var value types.Value
        // FIXME: use caller scope
        if value, err = types.Disclosure(project.Scope(), p.value); err != nil {
                return
        } else if value == nil {
                value = p.value
        }
        return set(project, p.op, p.name, value)
}

func (i *Interpreter) parseInfo(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseInfo(pos, s, a...)
}

func (i *Interpreter) parseWarn(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseWarn(pos, s, a...)
}

func (i *Interpreter) parseFail(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseWarn(pos, s, a...)
        runtime.Fail("fail: "+s, a...)
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
                        specName = ee.Data.(types.Value).String()
                } else {
                        return ErrorIllImport
                }
                for _, prop := range spec.Props[1:] {
                        if ee, ok := prop.(*ast.EvaluatedExpr); ok && ee.Data != nil {
                                v := ee.Data.(types.Value)
                                switch s := v.String(); s {
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
                                        defSet(token.ADD_ASSIGN, def, pn)
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

func (i *Interpreter) unary(x *ast.UnaryExpr) (v types.Value) {
        operand := i.expr(x.X)
        if t, ok := operand.Type().(*types.Basic); ok && t.IsFloat() {
                switch x.Op {
                case token.PLUS:  v = values.Float(+operand.Float())
                //case token.MINUS: v = values.Float(-operand.Float())
                }
        } else {
                switch x.Op {
                case token.PLUS:  v = values.Int(+operand.Integer())
                //case token.MINUS: v = values.Int(-operand.Integer())
                }
        }
        return
}

func (i *Interpreter) binary(x *ast.BinaryExpr) (v types.Value) {
        operand1, operand2 := i.expr(x.X), i.expr(x.Y)
        switch x.Op {
        default:
                assert(operand1 != nil)
                assert(operand2 != nil)
                unreachable();
        }
        return
}

func (i *Interpreter) ident(x *ast.Ident) (v types.Value) {
        var (
                scope = i.scope
                p = i.project
        )

        //fmt.Printf("ident: %T (%v) (%v)\n", x.Sym, x.Sym, x.Value)
        //fmt.Printf("ident: %T (%v) (%v)\n", x.Sym, x.Sym, scope)
        
        if !parser.IsUnresolved(x) {
                if v = x.Sym.(types.Value); v != nil {
                        return
                }
        }

        if _, v = scope.FindAt(x.Pos(), x.Value); v == nil && scope != p.Scope() {
                v = p.Scope().Find(x.Value)
        }

        if v == nil {
                v = values.None //scope.NewDummy(p, x.Value)
        }
        return
}

func (i *Interpreter) selector(first types.NameScoper, x *ast.SelectorExpr) (v types.Value) {
        var (
                scope = first.Scope()
                base types.Value
                next types.NameScoper
                name string
        )
        switch t := x.X.(type) {
        case *ast.Ident: 
                if name = t.Value; name == "use" {
                        name = useScopeName // demangle use scope name
                }
        case *ast.CallExpr:
                if name = i.expr(t).String(); name == "" {
                        i.parseInfo(t.Pos(), "selection on (%T)\n", t)
                        i.parseFail(t.Pos(), "'%v' is empty", t.Name)
                }
                //i.parseInfo(t.Pos(), "selection on '%s' (%T)\n", name, t)
        default:
                if name = i.expr(t).String(); name == "" {
                        i.parseFail(t.Pos(), "'%T' is empty", t)
                }
                //i.parseInfo(t.Pos(), "selection on '%s' (%T)\n", name, t)
        }

        if _, base = scope.FindAt(x.X.Pos(), name); base == nil {
                i.parseFail(x.X.Pos(), "'%s' is nil in %s", name, scope)
        }

        switch t := base.(type) {
        case *types.ProjectName:
                if sub := t.Project(); sub == nil {
                        i.parseFail(x.Pos(), "importee of %s is nil", t.Name())
                } else {
                        next = sub
                }
        case *types.ScopeName:
                // i.parseInfo(x.Pos(), "selector: %s", t.Scope())
                if scope = t.Scope(); scope == nil {
                        i.parseFail(x.Pos(), "importee of %s (scope) is nil", t.Name())
                } else {
                        next = types.NameScope(t.Name(), scope)
                }
        case nil:
                i.parseFail(x.Pos(), "'%T' undefined in '%s'", x.X, first.Name())
        default:
                i.parseFail(x.X.Pos(), "bad selection on %T %v", base, base)
                return
        }

        switch scope = next.Scope(); s := x.S.(type) {
        case *ast.Ident:
                if obj := scope.Lookup(s.Value); obj == nil {
                        i.parseFail(s.Pos(), "'%s' undefined in %s (%s)", s.Value, scope, i.project.Name())
                } else {
                        v = obj
                }
        case *ast.SelectorExpr:
                v = i.selector(next, s)
        case *ast.GlobExpr:
                switch s.Tok {
                case token.STAR:
                        //i.parseInfo(s.Pos(), "%v %v", scope.Names(), scope)
                        var list = values.List()
                        if name == useScopeName {
                                obj := scope.Lookup("*"/* use list */)
                                def, _ := obj.(*types.Def)
                                if def == nil {
                                        i.parseFail(s.Pos(), "bad use list (%T)", obj)
                                }
                                
                                v, elems := def.Value(), []types.Value{}
                                switch t := v.(type) {
                                case *types.List: elems = t.Elems
                                case *types.ProjectName: elems = append(elems, t)
                                default:
                                        i.parseFail(s.Pos(), "bad use list (%T %v)", v, v)
                                }

                                for _, elem := range elems {
                                        if pn, _ := elem.(*types.ProjectName); pn != nil {
                                                if entry := pn.Project().GetDefaultEntry(); entry != nil {
                                                        if s := entry.Name(); s != useRuleName && s != ignoreRuleName {
                                                                //i.parseInfo(s.Pos(), "%v %v", pn.Name(), entry)
                                                                list.Append(entry)
                                                        }
                                                }
                                        } else {
                                                i.parseFail(s.Pos(), "'%s' is not project in %s", elem)
                                        }
                                }
                        } else {
                                for _, name := range scope.Names() {
                                        if pn, _ := scope.Lookup(name).(*types.ProjectName); pn != nil {
                                                if entry := pn.Project().GetDefaultEntry(); entry != nil {
                                                        if s := entry.Name(); s != useRuleName && s != ignoreRuleName {
                                                                list.Append(entry)
                                                        }
                                                }
                                        } else {
                                                i.parseFail(s.Pos(), "'%s' is not project in %s", name, scope)
                                        }
                                }
                        }
                        v = list
                default:
                        i.parseFail(s.Pos(), "unimplemented glob (%s)", s.Tok)
                }
        default:
                if name := i.expr(s).String(); name == "" {
                        if c, ok := s.(*ast.CallExpr); ok {
                                i.parseFail(s.Pos(), "'%v' is empty", c.Name)
                        } else {
                                i.parseFail(s.Pos(), "'%T' is empty", s)
                        }
                } else if obj := scope.Lookup(name); obj == nil {
                        i.parseFail(s.Pos(), "'%s' undefined in %s (%s)", name, scope, i.project.Name())
                } else {
                        v = obj
                }
        }
        return
}

func (i *Interpreter) call(x *ast.CallExpr) (v types.Value) {
        var name = i.expr(x.Name)
        switch t := name.(type) {
        case types.Object:
                v = types.Delegate(t, i.exprs(x.Args)...)
        case *types.None, nil:
                if ident, _ := x.Name.(*ast.Ident); ident != nil {
                        i.parseFail(ident.Pos(), "'%s' undefined (%T %v)", ident.Value, x.Name, x.Name)
                } else {
                        i.parseFail(x.Name.Pos(), "undefined callable (%T %v)", x.Name, x.Name)
                }
        default:
                i.parseFail(x.Name.Pos(), "uncallable '%s' (%T -> %T)", x.Name, x.Name, name)
        }
        return
}

func (i *Interpreter) recipe(x *ast.RecipeExpr) (v types.Value) {
        if len(x.Elems) == 0 {
                v = values.None
        } else if x.Dialect == "" {
                var elems []types.Value
                switch t := x.Elems[0].(type) {
                default: runtime.Fail("unimplemented recipe (%T)", t)
                case *ast.SelectorExpr, *ast.Ident:
                case *ast.UseDefineClause:
                }
                elems = append(elems, i.exprs(x.Elems)...)
                //fmt.Printf("recipe: %T %T\n", x.Elems[0], elems[0])
                v = values.List(elems...)
        } else {
                v = values.Compound(i.exprs(x.Elems)...)
        }
        return
}

func (i *Interpreter) expr(expr ast.Expr) (v types.Value) {
        switch x := expr.(type) {
        case *ast.EvaluatedExpr:
                v = x.Data.(types.Value)
        case *ast.Ident:
                v = i.ident(x)
        case *ast.SelectorExpr:
                v = i.selector(i.project, x)
        case *ast.CallExpr:
                v = i.call(x)
        case *ast.RecipeExpr:
                v = i.recipe(x)
        case *ast.BasicLit:
                v = values.Literal(x.Kind, x.Value)
        case *ast.Bareword:
                v = values.Bareword(x.Value)
        case *ast.Barecomp:
                v = values.Barecomp(i.exprs(x.Elems)...)
        case *ast.Barefile:
                v = values.Barefile(i.expr(x.Name), x.Ext)
        case *ast.PathExpr:
                v = values.Path(i.exprs(x.Segments)...)
        case *ast.FlagExpr:
                v = values.Flag(i.expr(x.Name))
        case *ast.CompoundLit:
                v = values.Compound(i.exprs(x.Elems)...)
        case *ast.GroupExpr:
                v = values.Group(i.exprs(x.Elems)...)
        case *ast.ListExpr:
                v = values.List(i.exprs(x.Elems)...)
        case *ast.KeyValueExpr:
                v = values.Pair(i.expr(x.Key), i.expr(x.Value))
        case *ast.PercExpr:
                v = types.NewPercentPattern(i.project, i.expr(x.X), i.expr(x.Y))
        case *ast.UnaryExpr:
                v = i.unary(x)
        case nil:
                v = values.None
        case *ast.UseDefineClause:
                v = &usedefiner{
                        op: x.Tok,
                        name: i.expr(x.Name).String(),
                        value: i.expr(x.Value),
                        pos: nil,
                }
        case *ast.ClosureExpr:
                ee, ok := x.X.(*ast.EvaluatedExpr)
                if !ok || ee == nil {
                        i.parseFail(x.X.Pos(), "invalid ref operan (%T)", x.X)
                        break
                }
                name := ee.Data.(types.Value)
                if obj := i.scope.Find(name.String()); obj != nil {
                        v = values.Closure(obj, name)
                } else {
                        i.parseFail(x.X.Pos(), "'%v' undefined", name)
                }
        default:
                i.parseFail(x.Pos(), "unimplemented expression (%T %v)", x, x)
        }
        return
}

func (i *Interpreter) exprs(exprs []ast.Expr) (values []types.Value) {
        for _, x := range exprs {
                values = append(values, i.expr(x))
        }
        return
}

func (i *Interpreter) useProject(pos token.Pos, project *types.Project) error {
        use := project.Scope().Lookup(useRuleName)
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
                                defSet(token.ADD_ASSIGN, def, pn)
                        }
                }
        } else {
                return errors.New(fmt.Sprintf("'use' scope is not in %s", scope))
        }
        return i.useProject(pos, project)
}

func (i *Interpreter) use(spec *ast.UseSpec) error {
        var (
                name types.Value
                params []types.Value
        )
        if len(spec.Props) == 0 {
                //i.parseFail(spec.Pos(), "empty use spec")
                return errors.New("empty use spec")
        } else if name = i.expr(spec.Props[0]); name == nil {
                //i.parseFail(spec.Props[0].Pos(), "undefined use spec")
                return errors.New("undefined use target")
        } else if  name == values.None {
                //i.parseFail(spec.Props[0].Pos(), "none use spec")
                return errors.New("none use target")
        }
        for _, prop := range spec.Props[1:] {
                params = append(params, i.expr(prop))
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
                switch op := i.expr(spec.Props[0]).(type) {
                case types.Caller:
                        res, _ = op.Call(i.exprs(spec.Props[1:])...)
                default:
                        if _, obj := i.scope.FindAt(spec.EndPos, op.String()); obj != nil {
                                if f, _ := obj.(types.Caller); f != nil {
                                        res, err = f.Call(i.exprs(spec.Props[1:])...)
                                }
                        } else {
                                err = errors.New(fmt.Sprintf("undefined '%s'", op.String()))
                        }
                }
        }
        return
}

func (i *Interpreter) define(d *ast.DefineClause) (obj types.Object, err error) {
        //fmt.Printf("Interpreter.define: %v\n", d.Name)
        if i.project == nil {
                err = errors.New(fmt.Sprintf("not in a project"))
                return
        }
        var (
                name = i.expr(d.Name).String()
                v = i.expr(d.Value)
        )
        if obj, err = set(i.project, d.Tok, name, v); err != nil {
                i.parseWarn(d.Value.Pos(), "%v", err)
                err = nil // ignore errors
        }
        return
}

func (i *Interpreter) depend(depend types.Value) (result types.Value) {
        //fmt.Printf("rule: %T %v (%v)\n", depend, depend, depend.String())       
        switch entry := depend.(type) {
        case *types.RuleEntry, *types.ArgumentedEntry, *types.Closure, *types.Barefile, *types.Path, *types.PercentPattern:
                result = depend
        case *types.List:
                var list []types.Value
                for _, elem := range entry.Elems {
                        if v := i.depend(elem); v == nil {
                                return
                        } else if l, _ := v.(*types.List); l != nil {
                                list = append(list, l.Elems...)
                        } else {
                                list = append(list, v)
                        }
                }
                result = values.List(list...)
        case nil:
        default:
                if types.IsDummy(depend) {
                        result = depend
                }
        }
        return
}

func (i *Interpreter) rule(clause *ast.RuleClause) (err error) {
        var (
                depends []types.Value
                recipes []types.Value
                progScope *types.Scope
                params []string
        )
        for _, depend := range clause.Depends {
                depval := i.expr(depend)
                //fmt.Printf("depend: %T %v\n", depval, depval)
                if v := i.depend(depval); v == nil {
                        i.parseWarn(depend.Pos(), "invalid depend (%T %v -> %T %v)", depend, depend, depval, depval)
                        return errors.New(fmt.Sprintf("invalid depend"))
                } else if l, _ := v.(*types.List); l != nil {
                        depends = append(depends, l.Elems...)
                } else {
                        depends = append(depends, v)
                }
        }

        if p, ok := clause.Program.(*ast.ProgramExpr); ok && p != nil {
                if progScope, _ = p.Scope.(*types.Scope); progScope == nil {
                        return errors.New(fmt.Sprintf("undefined program scope (%T)", p.Scope))
                }
                if p.Values != nil {
                        recipes = i.exprs(p.Values)
                }
                params = p.Params
        } else {
                return errors.New(fmt.Sprintf("unsupported program type (%T)", clause.Program))
        }
        
        var modifiers []types.Value
        if clause.Modifier != nil {
                modifiers = i.exprs(clause.Modifier.Elems)
        }
        
        var prog = i.NewProgram(i.project, params, progScope, depends, recipes...)
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }
        
        var name string
        for n, target := range i.exprs(clause.Targets) {
                switch entry := target.(type) {
                case *types.PercentPattern:
                        i.project.AddPercentPattern(entry, prog)
                default:
                        class := types.GeneralRuleEntry
                        if name = target.String(); name == "use" {
                                if n == 0 && len(clause.Targets) == 1 {
                                        class = types.UseRuleEntry
                                        name = useRuleName
                                } else {
                                        i.parseFail(clause.Targets[n].Pos(), "'use' rule mixed with other targets")
                                }
                        } else if i.project.IsFile(name) {
                                class = types.FileRuleEntry
                        }

                        _, err = i.project.SetProgram(name, prog, class)
                        if err != nil {
                                break
                        }
                }
        }
        return
}

func (i *Interpreter) include(spec *ast.IncludeSpec) error {
        var (
                linfo = i.loads[len(i.loads)-1]
                specName = i.expr(spec.Props[0]).String()
                params []types.Value
        )

        if len(spec.Props) > 1 {
                params = i.exprs(spec.Props[1:])
        }

        var (
                jointPath = filepath.Join(linfo.absDir, specName)
                absDir, baseName = filepath.Split(jointPath)
        )
        defer restoreLoadingInfo(saveLoadingInfo(i, specName, absDir, baseName))
        
        doc, err := i.pc.ParseFile(i.fset, jointPath, nil, parseMode|parser.Flat)
        if err != nil {
                return err
        }
        if doc == nil {
                // ...
        }

        if len(params) > 0 {
                // TODO: parsing parameters
        }
        return nil //i.lexing(doc.Scope)
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
                
                specName = elem.String()
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

func (i *Interpreter) declareProject(ident *ast.Ident, params types.Value) (err error) {
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

func (i *Interpreter) closeCurrentProject(ident *ast.Ident) (err error) {
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
                return pc.project.IsFile(s)
        }
        return false
}

func (pc *parseContext) DeclareProject(ident *ast.Ident, params types.Value) error {
        if ident.Value == "@" {
                at := pc.Globe().Scope().Lookup(ident.Value)
                pc.project = at.Project()
                pc.scope = pc.project.Scope()
                return nil
        }
        return pc.declareProject(ident, params)
}

func (pc *parseContext) CloseCurrentProject(ident *ast.Ident) error {
        if ident.Value == "@" {
                //at := pc.Globe().Scope().Lookup(ident.Value)
                //pc.project = at.Project()
                //pc.scope = pc.project.Scope()
                //return nil
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
        
func (pc *parseContext) Define(clause *ast.DefineClause) (parser.RuntimeObj, error) {
        return pc.define(clause)
}

func (pc *parseContext) DeclareRule(clause *ast.RuleClause) (parser.RuntimeObj, error) {
        return nil, pc.rule(clause)
}

func (pc *parseContext) Eval(x ast.Expr) (res types.Value, err error) {
	defer func() {
		if e := recover(); e != nil {
                        err = errors.New(fmt.Sprintf("%v", e))
		}
        }()
        res = pc.expr(x)
        return
}

func (pc *parseContext) Resolve(name string) (obj parser.RuntimeObj) {
        if pc.scope != nil {
                obj = pc.scope.Find(name)
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
                                        return
                                }
                        }
                }
                if obj == nil && (name == "src") {
                        //fmt.Printf("Resolve: '%v' not in %v\n", name, pc.scope.Outer())
                        //for _, base := range pc.project.Bases() {
                        //        fmt.Printf("Resolve: %v %v\n", base.Name(), base.Scope())
                        //}
                        //for _, base := range pc.scope.Chain() {
                        //        fmt.Printf("Resolve: %v\n", base)
                        //}
                }
        }
        return
}

func (pc *parseContext) Symbol(name string, t types.Type) (obj, alt parser.RuntimeObj) {
        switch t {
        case types.DefineType:
                scope := pc.scope // always in the current scope
                obj, alt = scope.InsertDef(pc.project, name, values.None)
        case types.RuleEntryType:
                scope := pc.project.Scope() // always in the project
                obj, alt = scope.InsertEntry(pc.project, types.GeneralRuleEntry, name)
        }
        return
}
