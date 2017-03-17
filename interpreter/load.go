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
        "errors"
        "fmt"
        "os"
)

var parseMode = parser.DeclarationErrors |parser.Trace

type lazy struct {
        i *Interpreter
        s string
        a []types.Value
}

func (p *lazy) Type() types.Type  { return nil }
func (p *lazy) String() string    { return p.call().String() }
func (p *lazy) Integer() int64    { return p.call().Integer() }
func (p *lazy) Float() float64    { return p.call().Float() }
func (p *lazy) call() types.Value {
        var (
                sym = p.i.LookupAt(p.s, token.NoPos)
                args []interface{}
        )
        for _, a := range p.a {
                args = append(args, a)
        }
        return p.i.CallSym(sym, args...)
}

func restoreLoadingInfo(i *Interpreter) {
        i.loading = i.loading[0:len(i.loading)-1]
}

func saveLoadingInfo(i *Interpreter, dir, file string) *Interpreter {
        i.loading = append(i.loading, &loadingInfo{
                dir: dir, file: file,
        })
        return i
}

func (i *Interpreter) loadImportSpec(doc *ast.File, spec *ast.ImportSpec) error {
        var path string
        if 0 < len(spec.Props) {
                switch lit := spec.Props[0].(type) {
                case *ast.BasicLit:
                        if lit.Kind == token.STRING {
                                path = lit.Value
                        }
                case *ast.CompoundLit:
                        if 0 < len(lit.Elems) {
                                if lit, ok := lit.Elems[0].(*ast.BasicLit); ok {
                                        if lit.Kind == token.STRING {
                                                path = lit.Value
                                        }
                                }
                        }
                }
        }
        if path == "" {
                // TODO: report proper errors
                fmt.Printf("%v: import %v\n", doc.Name, spec.Props)
        }
        fmt.Printf("%v: import %v\n", doc.Name, path)
        return nil
}

func (i *Interpreter) evalUnary(x *ast.UnaryExpr) (v types.Value) {
        operand := i.evalExpr(x.X)
        if t, ok := operand.Type().(*types.Basic); ok && t.IsFloat() {
                switch x.Op {
                case token.PLUS:  v = values.Float(+operand.Float())
                case token.MINUS: v = values.Float(-operand.Float())
                }
        } else {
                switch x.Op {
                case token.PLUS:  v = values.Int(+operand.Integer())
                case token.MINUS: v = values.Int(-operand.Integer())
                }
        }
        return
}

func (i *Interpreter) evalBinary(x *ast.BinaryExpr) (v types.Value) {
        operand1, operand2 := i.evalExpr(x.X), i.evalExpr(x.Y)
        switch x.Op {
        default:
                assert(operand1 != nil)
                assert(operand2 != nil)
                unreachable();
        }
        return
}

func (i *Interpreter) evalProgram(p *ast.ProgramExpr) (v types.Value) {
        return
}

func (i *Interpreter) evalExpr(expr ast.Expr) (v types.Value) {
        switch x := expr.(type) {
        case *ast.BadExpr:
                unreachable();
        case *ast.Bareword:
                v = values.BarewordLit(x.ValuePos, x.Value)
        case *ast.BasicLit:
                v = values.Literal(x.ValuePos, x.Kind, x.Value)
        case *ast.CompoundLit:
                v = values.CompoundLit(x.BegPos, i.evalExprs(x.Elems)...)
        case *ast.GroupExpr:
                v = values.GroupLit(x.Pos(), i.evalExprs(x.Elems)...)
        case *ast.ListExpr:
                v = values.ListLit(x.Pos(), i.evalExprs(x.Elems)...)
        case *ast.CallExpr:
                var name string
                if x.Name == nil {
                        assert(x.Tok != token.CALL)
                        name = x.Tok.String()
                        assert(name[0] == '$')
                        name = name[1:]
                } else {
                        name = i.evalExpr(x.Name).String()
                }
                //v = i.call(i.lookupAt(name, x.Dollar), i.evalExprs(x.Args))
                v = &lazy{i, name, i.evalExprs(x.Args)}
        case *ast.UnaryExpr:
                v = i.evalUnary(x)
        case *ast.RecipeExpr:
                v = values.CompoundLit(x.TabPos, i.evalExprs(x.Elems)...)
        case *ast.ProgramExpr:
                v = i.evalProgram(x)
        default:
                unreachable()
        }
        return
}

func (i *Interpreter) evalExprs(exprs []ast.Expr) (values []types.Value) {
        for _, x := range exprs {
                values = append(values, i.evalExpr(x))
        }
        return
}

func (i *Interpreter) use(spec *ast.UseSpec) error {
        fmt.Printf("use: %v\n", spec) // TODO: use
        return nil
}

func (i *Interpreter) eval(spec *ast.EvalSpec) (res types.Value, err error) {
        if num := len(spec.Props); num > 0 {
                name := i.evalExpr(spec.Props[0])
                //fmt.Printf("eval: %v\n", name)
                if fun := i.LookupAt(name.String(), spec.EndPos); fun != nil {
                        args := i.evalExprs(spec.Props[1:])
                        res = fun.Call(args...)
                        //fmt.Printf("eval: %v\n", res)
                } else {
                        err = errors.New(fmt.Sprintf("undefined '%s'", name))
                        //fmt.Printf("error: `%v' is invalid\n", name)
                }
        }
        return
}

func (i *Interpreter) define(d *ast.DefineClause) error {
        m := i.CurrentModule()
        n, v := i.evalExpr(d.Name), i.evalExpr(d.Value)
        m.Scope().Insert(types.NewDef(d.TokPos, m, n.String(), v))
        return nil
}

func (i *Interpreter) rule(d *ast.RuleClause) (err error) {
        var (
                depends []*runtime.RuleEntry
                recipes []types.Value
        )
        for _, depend := range i.evalExprs(d.Depends) {
                entry := i.Registry().Entry(depend.String())
                depends = append(depends, entry)
        }
        if p, ok := d.Program.(*ast.ProgramExpr); ok && p != nil {
                if p.Values != nil {
                        recipes = i.evalExprs(p.Values)
                }
        }
        
        var modifiers []types.Value
        if d.Modifier != nil {
                modifiers = i.evalExprs(d.Modifier.Elems)
        }
        
        var prog = runtime.NewProgram(&i.Context, depends, recipes...)
        if len(modifiers) > 0 {
                dialect := modifiers[0].String()
                if err = prog.InitDialect(dialect, modifiers[1:]...); err != nil {
                        return
                }
        }
        
        for _, target := range i.evalExprs(d.Targets) {
                i.Registry().Insert(target.String(), prog)
        }
        return
}

func (i *Interpreter) clause(clause ast.Clause) error {
        switch d := clause.(type) {
        case *ast.GenericClause:
                for _, spec := range d.Specs {
                        var err error
                        switch s := spec.(type) {
                        case *ast.IncludeSpec:  err = i.include(s)
                        case *ast.UseSpec:      err = i.use(s)
                        case *ast.EvalSpec:  _, err = i.eval(s)
                        }
                        if err != nil {
                                return err
                        }
                }
        case *ast.DefineClause:
                return i.define(d)
        case *ast.RuleClause:
                return i.rule(d)
        default:
                //fmt.Printf("clause: %v\n", clause)
                unreachable()
        }
        return nil
}

func (i *Interpreter) include(spec *ast.IncludeSpec) error {
        var (
                linfo = i.loading[len(i.loading)-1]
                filename = i.evalExpr(spec.Props[0])
                params []types.Value
        )

        if len(spec.Props) > 1 {
                params = i.evalExprs(spec.Props[1:])
        }

        var (
                s = filepath.Join(linfo.dir, filename.String())
                dir, file = filepath.Split(s)
        )
        defer restoreLoadingInfo(saveLoadingInfo(i, dir, file))
        
        doc, err := parser.ParseFile(i.fset, s, nil, parseMode|parser.Flat)
        if err != nil {
                return err
        }

        if len(params) > 0 {
                // TODO: parsing parameters
        }

        for _, d := range doc.Clauses {
                if err = i.clause(d); err != nil {
                        return err
                }
        }
        return nil
}

// Interpreter.Load loads script from a file or source code (string, []byte).
func (i *Interpreter) Load(filename string, source interface{}) error {
        dir, file := filepath.Split(filename)
        defer restoreLoadingInfo(saveLoadingInfo(i, dir, file))
        
        doc, err := parser.ParseFile(i.fset, filename, source, parseMode)
        if err != nil {
                return err
        }

        m := i.NewModule(doc.Keypos, doc.Keyword, filename, doc.Name.Value)
        defer i.ExitCurrentScope()

        if m == nil { }

        for _, spec := range doc.Imports {
                if err = i.loadImportSpec(doc, spec); err != nil {
                        return err
                }
        }

        for _, d := range doc.Clauses {
                if err = i.clause(d); err != nil {
                        return err
                }
        }
        return nil
}

func (i *Interpreter) LoadDir(path string, filter func(os.FileInfo) bool) error {
        defer restoreLoadingInfo(saveLoadingInfo(i, path, ""))

        mods, err := parser.ParseDir(i.fset, path, filter, parseMode)
        if err != nil {
                return err
        }

        //m := i.newModule(doc.Keyword, filename, doc.Name.value)
        //defer i.upperScope()
        
        var keyword token.Token
        for name, mod := range mods {
                if keyword == token.ILLEGAL {
                        keyword = mod.Keyword
                }
                fmt.Printf("module: %v: %v\n", keyword, name)
        }
        return nil
}
