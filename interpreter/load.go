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
        "strings"
        "errors"
        "fmt"
        "os"
)

var parseMode = parser.DeclarationErrors //|parser.Trace

func restoreLoadingInfo(i *Interpreter) {
        i.loads = i.loads[0:len(i.loads)-1]
}

func saveLoadingInfo(i *Interpreter, dir, file string) *Interpreter {
        i.loads = append(i.loads, &loadInfo{
                dir: dir, file: file,
        })
        return i
}

func (i *Interpreter) loadImportSpec(doc *ast.File, spec *ast.ImportSpec) (err error) {
        var (
                linfo = i.loads[len(i.loads)-1]
                path string
        )
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
                //fmt.Printf("%v: import %v\n", doc.Name, spec.Props)
                return ErrorIllImport
        }

        var (
                modulePath string
                isDir bool
        )
        if abs := filepath.IsAbs(path); abs || strings.HasPrefix(path, "./") {
                var s = path
                if !abs {
                        s = filepath.Join(linfo.dir, s)
                        if a, e := filepath.Abs(s); e == nil {
                                s = a
                        } else {
                                return e
                        }
                }
                if fi, err := os.Stat(s); err != nil {
                        var sx = s + ".smart"
                        if fi, err = os.Stat(sx); fi != nil {
                                isDir, modulePath = fi.IsDir(), sx
                                goto importModule
                        }
                        sx = s + ".sm"
                        if fi, err = os.Stat(sx); fi != nil {
                                isDir, modulePath = fi.IsDir(), sx
                                goto importModule
                        }
                }
        } else {
                for _, base := range i.paths {
                        s := filepath.Join(base, path)
                        if fi, err := os.Stat(s); err == nil && fi != nil {
                                isDir, modulePath = fi.IsDir(), s
                                goto importModule
                        }
                }
        }
        if modulePath == "" {
                return ErrorNoModule
        }

importModule:
        if isDir {
                err = i.LoadDir(modulePath, nil)
        } else {
                err = i.Load(modulePath, nil)
        }
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

func (i *Interpreter) evalName(expr ast.Expr) (name []string) {
        switch x := expr.(type) {
        case *ast.BasicLit:
                name = append(name, x.Value)
        case *ast.Bareword:
                name = append(name, x.Value)
        case *ast.Barecomp:
                var part string
                for _, elem := range x.Elems {
                        s := i.evalExpr(elem).String()
                        if s == "." {
                                name = append(name, part)
                                part = ""
                        } else {
                                part += s
                        }
                }
                if part != "" {
                        name = append(name, part)
                }
        default:
                name = append(name, "?")
        }
        return
}

func (i *Interpreter) evalExpr(expr ast.Expr) (v types.Value) {
        switch x := expr.(type) {
        case *ast.BadExpr:
                unreachable();
        case *ast.BasicLit:
                v = values.Literal(x.Pos(), x.Kind, x.Value)
        case *ast.Bareword:
                v = values.BarewordLit(x.Pos(), x.Value)
        case *ast.Barecomp:
                v = values.BarecompLit(x.Pos(), i.evalExprs(x.Elems)...)
        case *ast.CompoundLit:
                v = values.CompoundLit(x.Pos(), i.evalExprs(x.Elems)...)
        case *ast.GroupExpr:
                v = values.GroupLit(x.Pos(), i.evalExprs(x.Elems)...)
        case *ast.ListExpr:
                v = values.ListLit(x.Pos(), i.evalExprs(x.Elems)...)
        case *ast.CallExpr:
                var name []string
                if x.Name == nil {
                        assert(x.Tok != token.CALL)
                        s := x.Tok.String()
                        assert(s[0] == '$')
                        name = append(name, s[1:])
                } else {
                        name = i.evalName(x.Name)
                }
                v = i.Fold(x.Pos(), name, i.evalExprs(x.Args)...)
        case *ast.UnaryExpr:
                v = i.evalUnary(x)
        case *ast.RecipeExpr:
                if elems := i.evalExprs(x.Elems); x.Dialect == "" {
                        v = values.ListLit(x.Pos(), elems...)
                } else {
                        v = values.CompoundLit(x.Pos(), elems...)
                }
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
                if _, fun := i.Scope().LookupAt(name.String(), spec.EndPos); fun != nil {
                        args := i.evalExprs(spec.Props[1:])
                        res = fun.Call(/*i.Context,*/ args...)
                } else {
                        err = errors.New(fmt.Sprintf("undefined '%s'", name))
                        //fmt.Printf("error: `%v' is invalid\n", name)
                }
        }
        return
}

func (i *Interpreter) define(d *ast.DefineClause) (err error) {
        if m := i.CurrentModule(); m != nil {
                n, v := i.evalExpr(d.Name), i.evalExpr(d.Value)
                m.Scope().Insert(types.NewDef(d.TokPos, m, n.String(), v))
        } else {
                err = ErrorNotModuleScope
        }
        return
}

func (i *Interpreter) rule(d *ast.RuleClause) (err error) {
        var (
                depends []*types.RuleEntry
                recipes []types.Value
                m = i.CurrentModule()
        )
        for _, depend := range i.evalExprs(d.Depends) {
                entry := m.Entry(depend.String())
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
        
        var (
                scope = types.NewScope(i.Scope(), d.TokPos, token.NoPos, "rule")
                prog = runtime.NewProgram(i.Context, scope, depends, recipes...)
        )
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }
        
        for _, target := range i.evalExprs(d.Targets) {
                m.Insert(target.String(), prog)
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
                linfo = i.loads[len(i.loads)-1]
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

func (i *Interpreter) file(doc *ast.File) (err error) {
        scope := types.NewScope(i.Scope(), doc.Keypos, token.NoPos, "file")
        defer i.SetScope(i.SetScope(scope))

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
        //fmt.Printf("end file: %v\n", scope.Names())
        return
}

func (i *Interpreter) module(mod *ast.Module) (err error) {
        m := i.DeclareModule(mod.Keypos, mod.Keyword, mod.Name, mod.Name)
        defer i.ExitModule(i.EnterModule(m))
        for _, f := range mod.Files {
                if err = i.file(f); err != nil {
                        break
                }
        }
        return
}

// Interpreter.Load loads script from a file or source code (string, []byte).
func (i *Interpreter) Load(filename string, source interface{}) error {
        dir, file := filepath.Split(filename)
        defer restoreLoadingInfo(saveLoadingInfo(i, dir, file))

        doc, err := parser.ParseFile(i.fset, filename, source, parseMode)
        if err != nil {
                return err
        }

        m := i.DeclareModule(doc.Keypos, doc.Keyword, filename, doc.Name.Value)
        defer i.ExitModule(i.EnterModule(m))
        return i.file(doc)
}

func (i *Interpreter) LoadDir(path string, filter func(os.FileInfo) bool) (err error) {
        defer restoreLoadingInfo(saveLoadingInfo(i, path, ""))

        mods, err := parser.ParseDir(i.fset, path, filter, parseMode)
        if err == nil {
                for _, mod := range mods {
                        if err = i.module(mod); err != nil {
                                break
                        }
                }
        }
        return
}
