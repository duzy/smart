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
        useScopeName = "~use~scope~"
        useRuleName = "~use~rule~"
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

func saveLoadingInfo(i *Interpreter, specPath, absPath, baseName string) *Interpreter {
        i.loads = append(i.loads, &loadinfo{
                specPath: specPath,
                absPath:  absPath,
                loader:   i.project,
                scope:    i.scope, //Scope(),
                declares: make(map[string]*declare),
        })
        return i
}

func set(p *types.Project, op token.Token, name string, value types.Value) (def *types.Def, err error) {
        // See https://www.gnu.org/software/make/manual/html_node/Setting.html
        var (
                scope = p.Scope()
                obj = scope.Lookup(name)
        )
        if obj == nil {
                var alt types.Object
                if obj, alt = scope.InsertNewDef(p, name, value); alt != nil {
                        // ...
                }
                return
        }
        if def, _ = obj.(*types.Def); def == nil {
                err = errors.New(fmt.Sprintf("name '%s' already taken in '%s'", name, p.Name()))
                return
        }

        switch op {
        case token.QUE_ASSIGN: // ?=
                // noop, only set if absent (not defined)
        case token.ADD_ASSIGN: // +=
                var (
                        l []types.Value
                        v = def.Value()
                )
                if a, ok := v.(*types.ListValue); ok {
                        l = append(l, a.Slice(0)...)
                } else {
                        l = append(l, v)
                }
                if a, ok := value.(*types.ListValue); ok {
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
                        def.Set(values.String(stdout.String()))
                } else {
                        def.Set(values.None)
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

type usedefine struct {
        op token.Token
        name string
        value types.Value
        pos *token.Position
}
func (p *usedefine) Pos() *token.Position { return p.pos }
func (p *usedefine) Type() types.Type     { return p.value.Type() }
func (p *usedefine) Lit() string          { return p.name + " = " + p.value.Lit() }
func (p *usedefine) String() string       { return p.name + " = " + p.value.String() }
func (p *usedefine) Integer() int64       { return 0 }
func (p *usedefine) Float() float64       { return 0 }
func (p *usedefine) Define(project *types.Project) (result types.Value, err error) {
        return set(project, p.op, p.name, p.unref(project, p.value))
}

func (p *usedefine) unref(project *types.Project, value types.Value) types.Value {
        var elements []types.Value
        switch v := value.(type) {
        case *types.AnyValue:
                if a, ok := v.V.(types.Value); ok {
                        v.V = p.unref(project, a)
                }
        case *types.BarecompValue:
                elements = v.Elems; goto unrefElems
        case *types.BarefileValue:
                v.Name = p.unref(project, v.Name)
        case *types.PathValue:
                elements = v.Segments; goto unrefElems
        case *types.FlagValue:
                v.Name = p.unref(project, v.Name)
        case *types.CompoundValue:
                elements = v.Elems; goto unrefElems
        case *types.ListValue:
                elements = v.Elems; goto unrefElems
        case *types.GroupValue:
                elements = v.Elems; goto unrefElems
        /* case *types.MapValue:
                for k, v := range v.Elems {
                        v.Elems[k] = p.unref(project, v)
                } */
        case *types.PairValue:
                v.K = p.unref(project, v.K)
                v.V = p.unref(project, v.V)
        case *useref:
                var args []types.Value
                name := p.unref(project, v.name).String()
                for _, a := range v.args {
                        args = append(args, p.unref(project, a))
                }
                value = v.unref(project, name)
        }
        goto done
        unrefElems: for i := 0; i < len(elements); i += 1 {
                elements[i] = p.unref(project, elements[i])
        }
        done: return value
}

type useref struct {
        name types.Value
        args []types.Value
        pos *token.Position
}
func (p *useref) Pos() *token.Position { return p.pos }
func (p *useref) Type() types.Type     { return types.None }
func (p *useref) Lit() string          { return "&" + p.name.Lit() }
func (p *useref) String() string       { return p.Lit() }
func (p *useref) Integer() int64       { return 0 }
func (p *useref) Float() float64       { return 0 }
func (p *useref) unref(project *types.Project, s string, a... types.Value) types.Value {
        var (
                obj = project.Scope().Lookup(s)
        )
        if c, ok := obj.(types.Caller); ok {
                if v, err := c.Call(a...); err == nil {
                        return v
                }
        }
        return obj
}

func (i *Interpreter) parseInfo(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseInfo(pos, s, a...)
}

func (i *Interpreter) parseFail(pos token.Pos, s string, a... interface{}) {
        i.pc.ParseWarn(pos, s, a...)
        runtime.Fail("fail: "+s, a...)
}

func (i *Interpreter) loadImportSpec(spec *ast.ImportSpec) (err error) {
        var (
                //scope = i.Scope()
                linfo = i.loads[len(i.loads)-1]
                specPath string
                params []types.Value
                nouse bool
        )
        if 0 < len(spec.Props) {
                switch lit := spec.Props[0].(type) {
                case *ast.BasicLit:
                        if lit.Kind == token.STRING {
                                specPath = lit.Value
                        }
                case *ast.CompoundLit:
                        if 0 < len(lit.Elems) {
                                if lit, ok := lit.Elems[0].(*ast.BasicLit); ok {
                                        if lit.Kind == token.STRING {
                                                specPath = lit.Value
                                        }
                                }
                        }
                }

                for _, prop := range spec.Props[1:] {
                        if v := i.expr(prop); v.String() == "nouse" {
                                nouse = true
                        } else {
                                params = append(params, v)
                        }
                }
        }

        if specPath == "" {
                //fmt.Printf("%v: import %v\n", doc.Name, spec.Props)
                return ErrorIllImport
        }

        var (
                absPath string
                isDir bool
        )
        if abs := filepath.IsAbs(specPath); abs || 
                strings.HasPrefix(specPath, "../") ||
                strings.HasPrefix(specPath, "./") {
                var s = specPath
                if !abs && linfo.absPath != "" {
                        s = filepath.Join(linfo.absPath, s)
                        if a, e := filepath.Abs(s); e == nil {
                                s = a
                        } else {
                                return e
                        }
                }
                if fi, err := os.Stat(s); err != nil {
                        var sx = s + ".smart"
                        if fi, err = os.Stat(sx); fi != nil {
                                isDir, absPath = fi.IsDir(), sx
                                goto importProject
                        }
                        sx = s + ".sm"
                        if fi, err = os.Stat(sx); fi != nil {
                                isDir, absPath = fi.IsDir(), sx
                                goto importProject
                        }
                } else {
                        isDir, absPath = fi.IsDir(), s
                }
        } else {
                for _, base := range i.paths {
                        s := filepath.Join(base, specPath)
                        if fi, err := os.Stat(s); err == nil && fi != nil {
                                isDir, absPath = fi.IsDir(), s
                                goto importProject
                        }
                }
        }
        
        if absPath == "" {
                return errors.New(fmt.Sprintf("import: '%s' not found", specPath))
        }

importProject:
        //fmt.Printf("import: '%s' (%s)\n", specPath, absPath)
        
        if isDir {
                err = i.loadDir(specPath, absPath, nil)
        } else {
                err = i.load(specPath, absPath, nil)
        }

        if err == nil && !nouse {
                if loaded, _ := i.loaded[absPath]; loaded != nil {
                        scope := i.project.Scope()
                        pn, _ := scope.Lookup(loaded.Name()).(*types.ProjectName)
                        if pn == nil {
                                return errors.New(fmt.Sprintf("no project name for '%s'", loaded.Name()))
                        }
                        // FIXME: defined used project in represented order
                        if sn, _ := scope.Lookup(useScopeName).(*types.ScopeName); sn != nil {
                                if alt := sn.Scope().Insert(pn); alt != nil {
                                        return errors.New(fmt.Sprintf("'%s' already defined in use scope", pn.Name()))
                                }
                        } else {
                                return errors.New(fmt.Sprintf("'use' scope is not in %s", scope))
                        }
                        err = i.useProject(loaded)
                } else {
                        unreachable()
                }
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
                err error
        )
        if _, v = scope.LookupAt(x.Pos(), x.Name); v == nil {
                p := i.project
                if x.Sym != nil && x.Sym.Kind == ast.Rul {
                        if v, err = p.Insert(x.Name, nil); err != nil {
                                i.parseFail(x.Pos(), err.Error())
                        }
                } else {
                        v = scope.NewDummy(p, x.Name)
                }
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
                if name = t.Name; name == "use" {
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

        if base = scope.Lookup(name); base == nil {
                i.parseFail(x.X.Pos(), "selection '%s' (nil) in %s", name, first.Scope())
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
                if obj := scope.Lookup(s.Name); obj == nil {
                        i.parseFail(s.Pos(), "'%s' undefined in %s (%s)", s.Name, scope, i.project.Name())
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
                        for _, name := range scope.Names() {
                                if pn, _ := scope.Lookup(name).(*types.ProjectName); pn != nil {
                                        if entry := pn.Project().GetDefaultEntry(); entry != nil {
                                                if entry.Name() != useRuleName /*"use"*/ {
                                                        list.Append(entry)
                                                }
                                        }
                                } else {
                                        i.parseFail(s.Pos(), "'%s' is not project in %s", name, scope)
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
        if obj, _ := name.(types.Object); obj != nil {
                v = i.Fold(x.Pos(), obj, i.exprs(x.Args)...)
        } else if name != nil {
                i.parseFail(x.Pos(), "bad call '%s' (%T, %T)", name, name, x.Name)
        } else {
                i.parseFail(x.Pos(), "calling undefined object %v", x.Name)
        }
        return
}

func (i *Interpreter) recipe(x *ast.RecipeExpr) (v types.Value) {
        if x.Dialect == "" {
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
                v = &usedefine{
                        op: x.Tok,
                        name: i.expr(x.Name).String(),
                        value: i.expr(x.Value),
                        pos: nil,
                }
        case *ast.RefExpr:
                if c, ok := x.X.(*ast.CallExpr); ok {
                        var name types.Value
                        if ident, ok := c.Name.(*ast.Ident); ok {
                                name = values.Bareword(ident.Name)
                        } else {
                                name = i.expr(c.Name)
                        }
                        v = &useref{
                                name: name,
                                args: i.exprs(c.Args),
                                pos: nil, //x.Pos(),
                        }
                } else {
                        i.parseFail(x.Pos(), "bad ref (%T)", x.X)
                }
        default:
                i.parseFail(x.Pos(), "unimplemented expression (%T)", x)
        }
        return
}

func (i *Interpreter) exprs(exprs []ast.Expr) (values []types.Value) {
        for _, x := range exprs {
                values = append(values, i.expr(x))
        }
        return
}

func (i *Interpreter) useProject(project *types.Project) error {
        use := project.Scope().Lookup(useRuleName)
        if rule, _ := use.(*types.RuleEntry); rule != nil {
                result, err := rule.Call(values.Any(i.project))
                //fmt.Printf("use: %v, %v (%v)\n", i.project.Name(), loaded.Name(), result)
                if err != nil {
                        return err
                } else if result == nil {
                        // ...
                }
        }
        return nil
}

func (i *Interpreter) use(spec *ast.UseSpec) error {
        var (
                scope = i.project.Scope()
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

        var (
                project *types.Project
                pn *types.ProjectName
                obj types.Object
                ok bool
        )
        if pn, ok = name.(*types.ProjectName); ok {
                if project = pn.Project(); project == nil {
                        return errors.New(fmt.Sprintf("%v is nil", pn))
                } else {
                        goto useProject
                }
        } else if obj = scope.Lookup(name.String()); obj == nil {
                return errors.New(fmt.Sprintf("'%s' is undefined in %v", name, scope))
        } else if pn, ok = obj.(*types.ProjectName); ok {
                if project = pn.Project(); project == nil {
                        return errors.New(fmt.Sprintf("project '%s' is nil", name))
                }
        } else {
                return errors.New(fmt.Sprintf("'%s' is not a project (%T)", name, obj))
        }

        useProject: if scope = i.project.Scope(); pn != nil {
                // FIXME: defined used project in represented order
                if sn, _ := scope.Lookup(useScopeName).(*types.ScopeName); sn != nil {
                        if alt := sn.Scope().Insert(pn); alt != nil {
                                if alt.Type().Kind() == types.ProjectNameKind {
                                        i.parseInfo(spec.Props[0].Pos(), "'%s' already used", pn.Name())
                                } else {
                                        return errors.New(fmt.Sprintf("'%s' already defined in %s", pn.Name(), sn.Scope()))
                                }
                        }
                } else {
                        return errors.New(fmt.Sprintf("'use' scope is not in %s", scope))
                }
        }
        return i.useProject(project)
}

func (i *Interpreter) eval(spec *ast.EvalSpec) (res types.Value, err error) {
        if num := len(spec.Props); num > 0 {
                switch op := i.expr(spec.Props[0]).(type) {
                case types.Caller:
                        res, _ = op.Call(i.exprs(spec.Props[1:])...)
                default:
                        if _, obj := i.scope.LookupAt(spec.EndPos, op.String()); obj != nil {
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
        if i.project == nil {
                err = errors.New(fmt.Sprintf("define %v not in a project scope", d.Name))
                return
        }
        var (
                name = i.expr(d.Name).String()
                v = i.expr(d.Value)
        )
        return set(i.project, d.Tok, name, v)
}

func (i *Interpreter) ruleDepend(depend types.Value) (result types.Value) {
        //fmt.Printf("rule: %T %v (%v)\n", depend, depend, depend.String())
        switch entry := depend.(type) {
        case *types.RuleEntry, *types.BarefileValue, *types.PathValue, *types.PercentPattern:
                result = depend
        case *types.ListValue:
                var list []types.Value
                for _, elem := range entry.Elems {
                        if v := i.ruleDepend(elem); v == nil {
                                return
                        } else if l, _ := v.(*types.ListValue); l != nil {
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

func (i *Interpreter) rule(d *ast.RuleClause) (err error) {
        var (
                depends []types.Value
                recipes []types.Value
        )
        for n, depend := range i.exprs(d.Depends) {
                if v := i.ruleDepend(depend); v == nil {
                        i.parseFail(d.Depends[n].Pos(), "invalid depend (%T %v)", d.Depends[n], depend)
                } else if l, _ := v.(*types.ListValue); l != nil {
                        depends = append(depends, l.Elems...)
                } else {
                        depends = append(depends, v)
                }
        }

        if p, ok := d.Program.(*ast.ProgramExpr); ok && p != nil {
                // mapping lexical objects
                for name, sym := range p.Scope.Symbols {
                        if auto, alt := i.scope.InsertNewDef(i.project, name, values.None); alt != nil {
                                i.parseFail(d.Pos(), "%s already defined", name)
                        } else {
                                sym.Data = auto
                        }
                }
                
                if p.Values != nil {
                        recipes = i.exprs(p.Values)
                }
        } else {
                return errors.New(fmt.Sprintf("unsupported program type"))
        }
        
        var modifiers []types.Value
        if d.Modifier != nil {
                modifiers = i.exprs(d.Modifier.Elems)
        }

        var prog = i.NewProgram(i.project, i.scope, depends, recipes...)
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }
        
        var name string
        for n, target := range i.exprs(d.Targets) {
                switch entry := target.(type) {
                case *types.PercentPattern:
                        i.project.AddPercentPattern(entry, prog)
                default:
                        if name = target.String(); name == "use" {
                                if n == 0 && len(d.Targets) == 1 {
                                        name = useRuleName
                                } else {
                                        i.parseFail(d.Targets[n].Pos(), "'use' rule mixed with other targets")
                                }
                        }
                        i.project.Insert(name, prog)
                }
        }
        return
}

func (i *Interpreter) lexing(lexScope *ast.Scope) (err error) {
        //fmt.Printf("%p: outer = %p\n", lexScope, lexScope.Outer)
        for name, sym := range lexScope.Symbols {
                _, s := i.scope.LookupAt(sym.Pos(), name)
                //fmt.Printf("lexing: %T %v (%v)\n", s, s, sym.Data)
                if sym.Data == nil {
                        sym.Data = s
                } else if sym.Data != s {
                        // FIXME: complain errors
                }
        }
        return
}

func (i *Interpreter) include(spec *ast.IncludeSpec) error {
        var (
                //scope = i.Scope()
                linfo = i.loads[len(i.loads)-1]
                specPath = i.expr(spec.Props[0]).String()
                params []types.Value
        )

        if len(spec.Props) > 1 {
                params = i.exprs(spec.Props[1:])
        }

        var (
                jointPath = filepath.Join(linfo.absPath, specPath)
                dir, base = filepath.Split(jointPath)
        )
        defer restoreLoadingInfo(saveLoadingInfo(i, specPath, dir, base))
        
        doc, err := i.pc.ParseFile(i.fset, jointPath, nil, parseMode|parser.Flat)
        if err != nil {
                return err
        }

        if len(params) > 0 {
                // TODO: parsing parameters
        }

        p := i.project
        p.AddFiles(doc.Files)
        p.AddExts(doc.Extensions)
        return i.lexing(doc.Scope)
}

func (i *Interpreter) openScope(as *ast.Scope, pos token.Pos, comment string) (err error) {
        //scope := types.NewScope(i.Scope(), doc.Keypos, token.NoPos, "file")
        //defer i.SetScope(i.SetScope(scope))
        scope := types.NewScope(i.scope, pos, token.NoPos, comment)
        as.Runtime = i.scope
        i.scope = scope
        //fmt.Printf("OpenScope: %s in %s\n", i.Scope(), as.Runtime)
        return
}

func (i *Interpreter) closeScope(as *ast.Scope) (err error) {
        if scope, ok := as.Runtime.(*types.Scope); ok {
                //fmt.Printf("CloseScope: %s -> %s\n", i.Scope(), scope)
                i.scope = scope
        } else {
                err = errors.New(fmt.Sprintf("bad runtime scope (%T)", as.Runtime))
        }
        return
}

func (i *Interpreter) declareProject(ident *ast.Ident) (err error) {
        var name = ident.Name
        if i.project != nil && i.project.Name() == name {
                return nil
        }
        
        linfo := i.loads[len(i.loads)-1]
        dec, ok := linfo.declares[name]
        if !ok {
                var (
                        absPath = linfo.absPath
                        relPath, relPathParent string
                )
                if !filepath.IsAbs(absPath) {
                        //absPath = filepath.Join(i.Getwd(), absPath)
                        absPath, _ = filepath.Abs(absPath)
                }

                relPath, _ = filepath.Rel(i.Getwd(), absPath)
                relPathParent = filepath.Dir(relPath)
                if relPath == "." && relPathParent == "." {
                        relPathParent = ".."
                }

                dec = &declare{
                        project: i.Globe().NewProject(absPath, relPath, linfo.specPath, name),
                }
                
                linfo.declares[name] = dec

                var (
                        p = dec.project
                        s = p.Scope()
                        use = types.NewScope(s, token.NoPos, token.NoPos, useScopeName)
                )
                if _, alt := s.InsertNewScopeName(p, useScopeName, use); alt != nil {
                        i.parseFail(ident.Pos(), "name '%s' already taken in %s", useScopeName, s)
                }
                if _, alt := s.InsertNewDef(p, "/", values.String(absPath)); alt != nil {
                        i.parseFail(ident.Pos(), "'$/' already defined in %s", s)
                }
                if _, alt := s.InsertNewDef(p, ".", values.String(relPath)); alt != nil {
                        i.parseFail(ident.Pos(), "'$.' already defined in %s", s)
                }
                if _, alt := s.InsertNewDef(p, "..", values.String(relPathParent)); alt != nil {
                        i.parseFail(ident.Pos(), "'$..' already defined in %s", s)
                }
        }

        if loader := linfo.loader; loader != nil {
                //fmt.Printf("DeclareProject: %s -> %s, %v\n", loader.Name(), dec.project.Name(), dec.s)

                var s = loader.Scope()
                if _, a := s.InsertNewProjectName(loader, name, dec.project); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                i.parseFail(ident.Pos(), "name '%s' already taken (%T)", name, a)
                                err = errors.New(fmt.Sprintf("name '%s' already taken (%T)", name, a))
                        }
                }

                //fmt.Printf("DeclareProject: %v from %v\n", name, loader.Scope())
        }

        i.project = dec.project
        dec.backscope = i.scope
        i.scope = dec.project.Scope()
        return
}

// Interpreter.Load loads script from a file or source code (string, []byte).
func (i *Interpreter) load(specPath, absPath string, source interface{}) error {
        if loaded, ok := i.loaded[absPath]; ok {
                var (
                        s = i.project.Scope()
                        name = loaded.Name()
                )
                if _, a := s.InsertNewProjectName(i.project, name, loaded); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                return errors.New(fmt.Sprintf("name '%s' already taken (%T)", name, a))
                        }
                }
                //fmt.Printf("Load: already loaded '%v'\n", specPath)
                //fmt.Printf("Load: %v\n", i.project.Scope())
                //fmt.Printf("Load: %v\n", loaded.Scope())
                return nil
        }
        
        dir, file := filepath.Split(absPath)
        defer restoreLoadingInfo(saveLoadingInfo(i, specPath, dir, file))

        doc, err := i.pc.ParseFile(i.fset, absPath, source, parseMode)
        if err != nil {
                return err
        }

        i.loaded[absPath] = i.project

        //fmt.Printf("Load: %v %v\n", absPath, doc.Name.Name)
        return i.lexing(doc.Scope)
}

func (i *Interpreter) loadDir(specPath, absPath string, filter func(os.FileInfo) bool) (err error) {
        if loaded, ok := i.loaded[absPath]; ok {
                var (
                        s = i.project.Scope()
                        name = loaded.Name()
                )
                if _, a := s.InsertNewProjectName(i.project, name, loaded); a != nil {
                        if v, ok := a.(*types.ProjectName); !ok || v == nil {
                                err = errors.New(fmt.Sprintf("name '%s' already taken (%T)", name, a))
                        }
                }
                //fmt.Printf("LoadDir: already loaded '%v'\n", specPath)
                //fmt.Printf("LoadDir: %v\n", i.project.Scope())
                //fmt.Printf("LoadDir: %v\n", loaded.Scope())
                return
        }

        defer restoreLoadingInfo(saveLoadingInfo(i, specPath, absPath, ""))

        mods, err := i.pc.ParseDir(i.fset, absPath, filter, parseMode)
        if err == nil && mods != nil {
                i.loaded[absPath] = i.project
                for _, mod := range mods {
                        //fmt.Printf("LoadDir: %v (%v)\n", absPath, mod)
                        if err = i.lexing(mod.Scope); err != nil {
                                return
                        }
                }
        }

        //fmt.Printf("LoadDir: %v %v\n", absPath, mods)
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

func (pc *parseContext) Extensions(exts map[string][]string) {
        pc.project.AddExts(exts)
}

func (pc *parseContext) Files(a []string) {
        pc.project.AddFiles(a)
}

func (pc *parseContext) DeclareProject(ident *ast.Ident) error {
        return pc.declareProject(ident)
}

func (pc *parseContext) OpenScope(as *ast.Scope, pos token.Pos, comment string) error {
        return pc.openScope(as, pos, comment)
}

func (pc *parseContext) CloseScope(as *ast.Scope) error {
        return pc.closeScope(as)
}

func (pc *parseContext) Import(spec *ast.ImportSpec) error {
        return pc.loadImportSpec(spec)
}

func (pc *parseContext) Include(spec *ast.IncludeSpec) error {
        return pc.include(spec)
}

func (pc *parseContext) Use(spec *ast.UseSpec) error {
        return pc.use(spec)
}

func (pc *parseContext) Eval(spec *ast.EvalSpec) error {
        _, err := pc.eval(spec)
        return err
}
        
func (pc *parseContext) Define(clause *ast.DefineClause) (parser.RuntimeSym, error) {
        return pc.define(clause)
}

func (pc *parseContext) DeclareRule(clause *ast.RuleClause) (parser.RuntimeSym, error) {
        return nil, pc.rule(clause)
}

func (pc *parseContext) EvalExpr(x ast.Expr) (s fmt.Stringer, err error) {
	defer func() {
		if e := recover(); e != nil {
                        err = errors.New(fmt.Sprintf("%v", e))
		}
        }()

        s = pc.expr(x)
        //fmt.Printf("EvalExpr: %T '%s'\n", x, s)
        return
}
