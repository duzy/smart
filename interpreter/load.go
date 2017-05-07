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
        var (
                last = len(i.loads)-1
                linfo = i.loads[last]
        )

        i.loads = i.loads[0:last]
        i.project = linfo.loader
        i.SetScope(linfo.scope)

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
                scope:    i.Scope(),
                declares: make(map[string]*declare),
        })
        return i
}

func (i *Interpreter) loadImportSpec(spec *ast.ImportSpec) (err error) {
        var (
                linfo = i.loads[len(i.loads)-1]
                specPath string
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
        return
}

func (i *Interpreter) evalUnary(x *ast.UnaryExpr) (v types.Value) {
        operand := i.evalExpr(x.X)
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

func (i *Interpreter) evalExpr(expr ast.Expr) (v types.Value) {
        switch x := expr.(type) {
        case *ast.BadExpr:
                unreachable();
        case *ast.Ident:
                if _, v = i.Scope().LookupAt(x.Pos(), x.Name); v == nil {
                        m := i.project
                        //fmt.Printf("eval:ident: '%v' (%T %v) in %v, %v, %v\n", x.Name, x.Sym, x.Sym,
                        //        i.Scope(), m.Scope(), i.project.Scope())
                        if x.Sym != nil && x.Sym.Kind == ast.Rul {
                                //fmt.Printf("rule: %T %v\n", x, x)
                                v = m.Insert(x.Name, nil)
                                //fmt.Printf("rule: %T %v\n", v, v)
                        } else {
                                //runtime.Fail("symbol %s undefined", x.Name)
                                v = types.NewDummy(m, i.Scope(), x.Name)
                        }
                }
        case *ast.BasicLit:
                v = values.Literal(x.Kind, x.Value)
        case *ast.Bareword:
                v = values.Bareword(x.Value)
        case *ast.Barecomp:
                v = values.Barecomp(i.evalExprs(x.Elems)...)
        case *ast.Barefile:
                v = values.Barefile(i.evalExpr(x.Name), x.Ext)
        case *ast.PathExpr:
                v = values.Path(i.evalExprs(x.Segments)...)
        case *ast.FlagExpr:
                v = values.Flag(i.evalExpr(x.Name))
        case *ast.CompoundLit:
                v = values.Compound(i.evalExprs(x.Elems)...)
        case *ast.GroupExpr:
                v = values.Group(i.evalExprs(x.Elems)...)
        case *ast.ListExpr:
                v = values.List(i.evalExprs(x.Elems)...)
        case *ast.KeyValueExpr:
                v = values.Pair(i.evalExpr(x.Key), i.evalExpr(x.Value))
        case *ast.SelectorExpr:
                if mn, _ := i.evalExpr(x.X).(*types.ProjectName); mn != nil {
                        if m := mn.Imported(); m == nil {
                                runtime.Fail("importee of %s is nil", mn.Name())
                        } else if sym := m.Scope().Lookup(x.Sel.Name); sym != nil {
                                v = sym
                        } else {
                                runtime.Fail("symbol %s undefined in %s", x.Sel.Name, mn.Name())
                        }
                } else if id, _ := x.X.(*ast.Ident); id != nil {
                        //fmt.Printf("eval:selector: '%v'.%v in %v, %v\n", id.Name, x.Sel.Name, i.Scope(), i.project.Scope())
                        //runtime.Fail("project %s is not imported (%s.%v)", id.Name, id.Name, x.Sel.Name)
                        runtime.Fail("project %s is not imported (%s.%v) in %v, %v", 
                                id.Name, id.Name, x.Sel.Name,
                                i.Scope()/*.Parent().Parent().Parent()*/,
                                i.project.Scope())
                } else {
                        unreachable()
                }
        case *ast.CallExpr:
                var name = i.evalExpr(x.Name)
                if sym, _ := name.(types.Object); sym != nil {
                        //fmt.Printf("call: %T %T %v\n", x.Name, name, sym)
                        v = i.Fold(x.Pos(), sym, i.evalExprs(x.Args)...)
                } else if name != nil {
                        runtime.Fail("unsupported name '%s' (%T, %T)", name, x.Name, name)
                } else {
                        runtime.Fail("calling undefined object %v", x.Name)
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
                        v = values.List(elems...)
                } else {
                        elems := i.evalExprs(x.Elems)
                        v = values.Compound(elems...)
                }
        case *ast.PercExpr:
                v = types.NewPercentPattern(i.project, i.evalExpr(x.X), i.evalExpr(x.Y))
        case *ast.UnaryExpr:
                v = i.evalUnary(x)
        case nil:
                v = values.None
        default:
                //fmt.Printf("%T %v\n", x, x)
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
        runtime.Fail("unimplemented: use %v\n", spec) // TODO: use
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

func (i *Interpreter) define(d *ast.DefineClause) (sym types.Object, err error) {
        if p := i.project; p != nil {
                var (
                        scope = p.Scope()
                        name = i.evalExpr(d.Name).String()
                        v = i.evalExpr(d.Value)
                )

                if t, _ := v.(*types.Def); t != nil {
                        v = t.Value()
                }
                
                if sym = scope.Insert(types.NewDef(p, name, v)); sym != nil {
                        if def, ok := sym.(*types.Def); ok {
                                def.Set(v)
                        } else {
                                err = errors.New(fmt.Sprintf("name '%s' already taken", name))
                        }
                }
        } else {
                err = errors.New(fmt.Sprintf("define %v not in a project scope", d.Name))
        }
        return
}

func (i *Interpreter) rule(d *ast.RuleClause) (err error) {
        var (
                depends []types.Value // *types.RuleEntry, *values.BarefileValue
                recipes []types.Value
                m = i.project//CurrentProject()
        )
        for i, depend := range i.evalExprs(d.Depends) {
                //fmt.Printf("Interpreter.rule: %T %v (%v)\n", depend, depend, depend.String())
                switch entry := depend.(type) {
                case *types.RuleEntry, *values.BarefileValue, *values.PathValue,
                        *types.PercentPattern:
                        depends = append(depends, entry)
                case nil:
                        runtime.Fail("entry undefined (%T %v)", d.Depends[i], d.Depends[i])
                default:
                        if types.IsDummyValue(depend) {
                                depends = append(depends, entry)
                        } else {
                                runtime.Fail("%T is not valid RuleEntry (%s)", depend, depend)
                        }
                }
        }

        //scope := types.NewScope(i.Scope(), d.TokPos, token.NoPos, "rule")
        //defer i.SetScope(i.SetScope(scope))
        scope := i.Scope()
        
        if p, ok := d.Program.(*ast.ProgramExpr); ok && p != nil {
                // mapping lexical objects
                for name, sym := range p.Scope.Symbols {
                        //fmt.Printf("sym: %v %T\n", name, sym)
                        auto := types.NewDef(m, name, values.None)
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
        
        var prog = runtime.NewProgram(i.Context, i.project, scope, depends, recipes...)
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }
        
        for _, target := range i.evalExprs(d.Targets) {
                //fmt.Printf("Interpreter.rule: %T %v (%v)\n", target, target, target.String())
                switch entry := target.(type) {
                case *types.PercentPattern:
                        m.AddPercentPattern(entry, prog)
                default:
                        m.Insert(target.String(), prog)
                }
        }
        return
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
                specPath = i.evalExpr(spec.Props[0]).String()
                params []types.Value
        )

        if len(spec.Props) > 1 {
                params = i.evalExprs(spec.Props[1:])
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
        scope := types.NewScope(i.Scope(), pos, token.NoPos, comment)
        as.Runtime = i.SetScope(scope)
        //fmt.Printf("OpenScope: %s in %s\n", i.Scope(), as.Runtime)
        return
}

func (i *Interpreter) closeScope(as *ast.Scope) (err error) {
        if scope, ok := as.Runtime.(*types.Scope); ok {
                //fmt.Printf("CloseScope: %s -> %s\n", i.Scope(), scope)
                i.SetScope(scope)
        } else {
                err = errors.New(fmt.Sprintf("bad runtime scope (%T)", as.Runtime))
        }
        return
}

func (i *Interpreter) declareProject(name string) (err error) {
        if i.project != nil && i.project.Name() == name {
                return nil
        }
        
        linfo := i.loads[len(i.loads)-1]
        dec, ok := linfo.declares[name]
        if !ok {
                var projectPath string
                for n := len(i.loads)-1; n >= 0; n -= 1 {
                        s := i.loads[n].specPath
                        if strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") {
                                projectPath = filepath.Join(s, projectPath)
                        } else if strings.HasPrefix(s, "/") {
                                projectPath = filepath.Join(s, projectPath)
                                break
                        } else {
                                projectPath = filepath.Join(i.loads[n].absPath, projectPath)
                                break
                        }
                }
                if projectPath == "" {
                        projectPath = "."
                }
                
                dec = &declare{
                        project: i.Globe().NewProject(projectPath, name),
                }
                linfo.declares[name] = dec

                var (
                        absPath = linfo.absPath
                        specPath = linfo.specPath
                        specPathParent = filepath.Dir(specPath)
                        ms = dec.project.Scope()
                )
                if !filepath.IsAbs(absPath) {
                        //absPath = filepath.Join(i.Getwd(), absPath)
                        absPath, _ = filepath.Abs(absPath)
                }
                if specPath == "." && specPathParent == "." {
                        specPathParent = ".."
                }
                if ms.Insert(types.NewDef(dec.project, "/", values.String(absPath))) != nil {
                        panic(fmt.Sprintf("'$/' already defined"))
                }
                if ms.Insert(types.NewDef(dec.project, ".", values.String(specPath))) != nil {
                        panic(fmt.Sprintf("'$.' already defined"))
                }
                if ms.Insert(types.NewDef(dec.project, "..", values.String(specPathParent))) != nil {
                        panic(fmt.Sprintf("'$..' already defined"))
                }
        }

        if loader := linfo.loader; loader != nil {
                //fmt.Printf("DeclareProject: %s -> %s, %v\n", loader.Name(), dec.project.Name(), dec.s)

                if obj := loader.Scope().Lookup(name); obj == nil {
                        mn := types.NewProjectName(loader, name, dec.project)
                        loader.Scope().Insert(mn)
                } else if v, ok := obj.(*types.ProjectName); !ok || v == nil {
                        err = errors.New(fmt.Sprintf("name '%s' already taken (%T)", name, obj))
                }

                //fmt.Printf("DeclareProject: %v from %v\n", name, loader.Scope())
        }

        if i.project != nil {
                //ee := i.declared[name]
                //i.Context.ExitProject(dec.s)
        }

        i.project = dec.project
        dec.backscope = i.SetScope(dec.project.Scope())

        /*
        if loader, backscope := linfo.loader, dec.backscope; loader != nil {
                fmt.Printf("DeclareProject: %v in %v; %v -> %v\n", dec.project.Name(), loader.Name(), backscope, i.Scope())
        } else {
                fmt.Printf("DeclareProject: %v in %v; %v -> %v\n", dec.project.Name(), loader, backscope, i.Scope())
        } */
        return
}

// Interpreter.Load loads script from a file or source code (string, []byte).
func (i *Interpreter) load(specPath, absPath string, source interface{}) error {
        if loaded, ok := i.loaded[absPath]; ok {
                mn := types.NewProjectName(i.project, loaded.Name(), loaded)
                i.project.Scope().Insert(mn)
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
                mn := types.NewProjectName(i.project, loaded.Name(), loaded)
                i.project.Scope().Insert(mn)
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

func (pc *parseContext) DeclareProject(name string) error {
        return pc.declareProject(name)
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

        s = pc.evalExpr(x)
        //fmt.Printf("EvalExpr: %T '%s'\n", x, s)
        return
}
