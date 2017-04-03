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
                if !abs && linfo.dir != "" {
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
                } else {
                        isDir, modulePath = fi.IsDir(), s
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
                return errors.New(fmt.Sprintf("module '%s' missing", path))
        }

importModule:
        //fmt.Printf("import: %v\n", path)
        
        if isDir {
                err = i.LoadDir(modulePath, nil)
        } else {
                err = i.Load(modulePath, nil)
        }
        return
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

/* func (i *Interpreter) splitName(x *ast.Barecomp) (name []string) {
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
        return
}

func (i *Interpreter) evalName(expr ast.Expr) (name []string) {
        switch x := expr.(type) {
        case *ast.BasicLit:
                name = append(name, x.Value)
        case *ast.Bareword:
                name = append(name, x.Value)
        case *ast.Barecomp:
                name = append(name, i.splitName(x)...)
        default:
                name = append(name, "?")
        }
        return
} */

func (i *Interpreter) evalExpr(expr ast.Expr) (v types.Value) {
        switch x := expr.(type) {
        case *ast.BadExpr:
                unreachable();
        case *ast.Ident:
                //fmt.Printf("ident: %T %v\n", x, x)
                if _, v = i.Scope().LookupAt(x.Pos(), x.Name); v == nil {
                        if x.Sym != nil && x.Sym.Kind == ast.Rul {
                                //fmt.Printf("rule: %T %v\n", x, x)
                                m := i.CurrentModule()
                                v = m.Insert(x.Pos(), x.Name, nil)
                                //fmt.Printf("rule: %T %v\n", v, v)
                        } else {
                                runtime.Fail("symbol %s undefined", x.Name)
                        }
                }
                //fmt.Printf("symbol: %v %T\n", x.Name, v)
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
        case *ast.SelectorExpr:
                if mn, _ := i.evalExpr(x.X).(*types.ModuleName); mn != nil {
                        if m := mn.Imported(); m == nil {
                                runtime.Fail("module %s undefined", mn.Name())
                        //} else if _, sym := m.Scope().LookupAt(x.Pos(), x.Sel.Name); sym != nil {
                        } else if sym := m.Scope().Lookup(x.Sel.Name); sym != nil {
                                v = sym
                        } else {
                                runtime.Fail("symbol %s undefined in %s", x.Sel.Name, mn.Name())
                        }
                } else if id, _ := x.X.(*ast.Ident); id != nil {
                        runtime.Fail("module '%s' undefiend", id.Name)
                } else {
                        unreachable()
                }
                
        case *ast.CallExpr:
                var name = i.evalExpr(x.Name)
                if sym, _ := name.(types.Symbol); sym != nil {
                        //fmt.Printf("call: %T %v\n", x.Name, sym)
                        v = i.Fold(x.Pos(), sym, i.evalExprs(x.Args)...)
                } else if name != nil {
                        runtime.Fail("unsupported name '%s' (%T, %T)", name, x.Name, name)
                } else {
                        runtime.Fail("symbol %v undefined", x.Name)
                }
        case *ast.RecipeExpr:
                if x.Dialect == "" {
                        var elems []types.Value
                        switch t := x.Elems[0].(type) {
                        default: runtime.Fail("unsupported recipe %T", t)
                        case *ast.SelectorExpr, *ast.Ident:
                        }
                        elems = append(elems, i.evalExprs(x.Elems)...)
                        //fmt.Printf("recipe: %T %T\n", x.Elems[0], elems[0])
                        v = values.ListLit(x.Pos(), elems...)
                } else {
                        elems := i.evalExprs(x.Elems)
                        v = values.CompoundLit(x.Pos(), elems...)
                }
        case *ast.UnaryExpr:
                v = i.evalUnary(x)
        case nil:
                v = values.None
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

func (i *Interpreter) declare(pos token.Pos, kw token.Token, path, name string) *types.Module {
        m := i.DeclareModule(pos, kw, path, name)
        ms := m.Scope()
        if path == "" && true {
                path = "."
        }

        var workdir string
        if filepath.IsAbs(path) {
                workdir = path
        } else {
                workdir = filepath.Join(i.Getwd(), path)
        }
        if ms.Insert(types.NewDef(pos, m, "/", values.String(workdir))) != nil {
                panic(fmt.Sprintf("'$/' already defined"))
        }
        if ms.Insert(types.NewDef(pos, m, ".", values.String(path))) != nil {
                panic(fmt.Sprintf("'$.' already defined"))
        }
        return m
}

func (i *Interpreter) use(spec *ast.UseSpec) error {
        fmt.Printf("use: %v\n", spec) // TODO: use
        return nil
}

func (i *Interpreter) eval(spec *ast.EvalSpec) (res types.Value, err error) {
        if num := len(spec.Props); num > 0 {
                name := i.evalExpr(spec.Props[0])
                if _, fun := i.Scope().LookupAt(spec.EndPos, name.String()); fun != nil {
                        args := i.evalExprs(spec.Props[1:])
                        res, _ = fun.Call(args...)
                } else {
                        err = errors.New(fmt.Sprintf("undefined '%s'", name))
                        //fmt.Printf("error: `%v' is invalid\n", name)
                }
        }
        return
}

func (i *Interpreter) define(d *ast.DefineClause) (err error) {
        if m := i.CurrentModule(); m != nil {
                var (
                        scope = m.Scope()
                        name = i.evalExpr(d.Name).String()
                        v = i.evalExpr(d.Value)
                )

                if sym := scope.Insert(types.NewDef(d.TokPos, m, name, v)); sym != nil {
                        if def, ok := sym.(*types.Def); ok {
                                def.Set(v)
                        } else {
                                err = errors.New(fmt.Sprintf("name '%s' already taken", name))
                        }
                }
        } else {
                err = errors.New(fmt.Sprintf("define %v not in a module scope", d.Name))
        }
        return
}

func (i *Interpreter) rule(d *ast.RuleClause) (err error) {
        var (
                depends []*types.RuleEntry
                recipes []types.Value
                m = i.CurrentModule()
        )
        for i, depend := range i.evalExprs(d.Depends) {
                //fmt.Printf("Interpreter.rule: %T %v (%v)\n", depend, depend, depend.String())
                if entry, _ := depend.(*types.RuleEntry); entry != nil {
                        depends = append(depends, entry)
                } else if depend != nil {
                        runtime.Fail("%s is not RuleEntry (%T)", depend, depend)
                } else {
                        runtime.Fail("entry undefined (%v)", d.Depends[i])
                }
        }

        scope := types.NewScope(i.Scope(), d.TokPos, token.NoPos, "rule")
        defer i.SetScope(i.SetScope(scope))
        
        if p, ok := d.Program.(*ast.ProgramExpr); ok && p != nil {
                // mapping lexical symbols
                for name, sym := range p.Scope.Symbols {
                        //fmt.Printf("sym: %v %T\n", name, sym)
                        auto := types.NewAuto(m, name, values.None)
                        if alt := scope.Insert(auto); alt != nil {
                                runtime.Fail("%s already defined", name)
                        }
                        sym.Data = auto
                }
                
                if p.Values != nil {
                        recipes = i.evalExprs(p.Values)
                }
        } else {
                return errors.New(fmt.Sprintf("unsupported program type"))
        }
        
        var modifiers []types.Value
        if d.Modifier != nil {
                modifiers = i.evalExprs(d.Modifier.Elems)
        }
        
        var prog = runtime.NewProgram(i.Context, scope, depends, recipes...)
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }
        
        for _, target := range i.evalExprs(d.Targets) {
                m.Insert(target.Pos(), target.String(), prog)
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
                unreachable()
        }
        return nil
}

func (i *Interpreter) lexing(lexScope *ast.Scope) (err error) {
        //fmt.Printf("%p: outer = %p\n", lexScope, lexScope.Outer)
        for name, sym := range lexScope.Symbols {
                _, s := i.Scope().LookupAt(sym.Pos(), name)
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
        
        doc, err := i.pc.ParseFile(i.fset, s, nil, parseMode|parser.Flat)
        if err != nil {
                return err
        }

        if len(params) > 0 {
                // TODO: parsing parameters
        }

        i.SetExts(doc.Extensions)

        for _, d := range doc.Clauses {
                if err = i.clause(d); err != nil {
                        return err
                }
        }
        return i.lexing(doc.Scope)
}

func (i *Interpreter) file(doc *ast.File) (err error) {
        scope := types.NewScope(i.Scope(), doc.Keypos, token.NoPos, "file")
        defer i.SetScope(i.SetScope(scope))

        for _, spec := range doc.Imports {
                if err = i.loadImportSpec(doc, spec); err != nil {
                        return err
                }
        }

        i.SetExts(doc.Extensions)

        for _, i := range doc.Unresolved {
                fmt.Printf("%p: unresolved: %T %v\n", doc.Scope, i.Sym, i.Name)
        }
        
        for _, d := range doc.Clauses {
                if err = i.clause(d); err != nil {
                        return err
                }
        }
        return i.lexing(doc.Scope)
}

func (i *Interpreter) module(dir string, mod *ast.Module) (err error) {
        m := i.declare(mod.Keypos, mod.Keyword, dir, mod.Name)
        defer i.ExitModule(i.EnterModule(m))

        //fmt.Printf("module: %s %s\n", dir, mod.Name)
        
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

        doc, err := i.pc.ParseFile(i.fset, filename, source, parseMode)
        if err != nil {
                return err
        }

        //fmt.Printf("load: %v %v\n", filename, doc.Name.Name)
        
        m := i.declare(doc.Keypos, doc.Keyword, dir, doc.Name.Name)
        defer i.ExitModule(i.EnterModule(m))
        return i.file(doc)
}

func (i *Interpreter) LoadDir(path string, filter func(os.FileInfo) bool) (err error) {
        defer restoreLoadingInfo(saveLoadingInfo(i, path, ""))

        mods, err := i.pc.ParseDir(i.fset, path, filter, parseMode)
        if err == nil {
                for _, mod := range mods {
                        //fmt.Printf("LoadDir: %v (%v)\n", path, mod)
                        if err = i.module(path, mod); err != nil {
                                break
                        }
                }
        }
        return
}
