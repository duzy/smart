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
        "github.com/duzy/smart/literal"
        "path/filepath"
        //"errors"
        "fmt"
        "os"
)

var parseMode = parser.DeclarationErrors |parser.Trace

func popLoadingInfo(i *Interpreter) {
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

func (i *Interpreter) evalExpr(expr ast.Expr) types.Value {
        switch x := expr.(type) {
        case *ast.BadExpr:
                panic("bad expression")
        case *ast.Bareword:
                return literal.NewBareword(x.ValuePos, x.Value)
        case *ast.BasicLit:
                return literal.NewValue(x.ValuePos, x.Kind, x.Value)
        case *ast.CompoundLit:
                return types.NewCompound(x.BegPos, x.EndPos, i.evalExprs(x.Elems)...)
        case *ast.ListExpr:
                return types.NewList(x.Pos(), x.End(), i.evalExprs(x.Elems)...)
        case *ast.CallExpr:
                sym := i.lookupAt(i.evalExpr(x.Name).String(), x.Dollar)
                return i.call(sym, i.evalExprs(x.Args))
        case *ast.GroupExpr:
                // TODO: ...
        case *ast.UnaryExpr:
                // TODO: ...
        case *ast.RecipeExpr:
                // TODO: ...
        case *ast.ProgramExpr:
                // TODO: ...
        }
        return nil
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

func (i *Interpreter) eval(spec *ast.EvalSpec) error {
        fmt.Printf("eval: %v\n", spec) // TODO: eval
        return nil
}

func (i *Interpreter) declDefine(d *ast.DefineClause) error {
        fmt.Printf("define: %v\n", d.Name)
        return nil
}

func (i *Interpreter) declRule(d *ast.RuleClause) error {
        fmt.Printf("rule: %v, %v\n", d.Targets, d.Depends)
        return nil
}

func (i *Interpreter) clause(clause ast.Clause) error {
        switch d := clause.(type) {
        case *ast.GenericClause:
                for _, spec := range d.Specs {
                        var err error
                        switch s := spec.(type) {
                        case *ast.IncludeSpec:  err = i.include(s)
                        case *ast.UseSpec:      err = i.use(s)
                        case *ast.EvalSpec:     err = i.eval(s)
                        }
                        if err != nil {
                                return err
                        }
                }
        case *ast.DefineClause:
                return i.declDefine(d)
        case *ast.RuleClause:
                return i.declRule(d)
        default:
                fmt.Printf("clause: %v\n", clause)
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

        var s = filepath.Join(linfo.dir, filename.String())
        doc, err := parser.ParseFile(i.fset, s, nil, parseMode|parser.Flat)
        if err != nil {
                return err
        }

        dir, file := filepath.Split(s)
        defer popLoadingInfo(saveLoadingInfo(i, dir, file))

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
        doc, err := parser.ParseFile(i.fset, filename, source, parseMode)
        if err != nil {
                return err
        }

        dir, file := filepath.Split(filename)
        defer popLoadingInfo(saveLoadingInfo(i, dir, file))
        
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
        mods, err := parser.ParseDir(i.fset, path, filter, parseMode)
        if err != nil {
                return err
        }

        defer popLoadingInfo(saveLoadingInfo(i, path, ""))
        
        var keyword token.Token
        for name, mod := range mods {
                if keyword == token.ILLEGAL {
                        keyword = mod.Keyword
                }
                fmt.Printf("%v: %v\n", keyword, name)
        }
        return nil
}
