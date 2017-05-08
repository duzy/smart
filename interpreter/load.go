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

type usedefine struct {
        name string
        value types.Value
        pos *token.Position
}

func (p *usedefine) Type() types.Type     { return p.value.Type() }
func (p *usedefine) Pos() *token.Position { return p.pos }
func (p *usedefine) Lit() string          { return p.name + " = " + p.value.Lit() }
func (p *usedefine) String() string       { return p.name + " = " + p.value.String() }
func (p *usedefine) Integer() int64       { return 0 }
func (p *usedefine) Float() float64       { return 0 }
func (p *usedefine) Define(project *types.Project) (result types.Value, err error) {
        var (
                scope = project.Scope()
                def = scope.Lookup(p.name)
        )
        if def == nil {
                scope.Insert(types.NewDef(project, p.name, p.value))
        } else if t, _ := def.(*types.Def); t != nil {
                t.Set(p.value)
        } else {
                err = errors.New(fmt.Sprintf("name '%s' taken in '%s'", p.name, project.Name()))
        }
        return
}

func (i *Interpreter) loadImportSpec(spec *ast.ImportSpec) (err error) {
        var (
                scope = i.Scope()
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
                        if v := i.expr(scope, prop); v.String() == "nouse" {
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
                loaded := i.loaded[absPath]
                use := loaded.Scope().Lookup("use")
                if rule, _ := use.(*types.RuleEntry); rule != nil {
                        result, err := rule.Call(values.Any(i.project))
                        if err != nil {
                                //...
                        } else if result == nil {
                        }
                        //fmt.Printf("use: %v, %v (%v)\n", i.project.Name(), loaded.Name(), result)
                }
        }
        return
}

func (i *Interpreter) unary(scope *types.Scope, x *ast.UnaryExpr) (v types.Value) {
        operand := i.expr(scope, x.X)
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

func (i *Interpreter) binary(scope *types.Scope, x *ast.BinaryExpr) (v types.Value) {
        operand1, operand2 := i.expr(scope, x.X), i.expr(scope, x.Y)
        switch x.Op {
        default:
                assert(operand1 != nil)
                assert(operand2 != nil)
                unreachable();
        }
        return
}

func (i *Interpreter) ident(scope *types.Scope, x *ast.Ident) (v types.Value) {
        if _, v = scope.LookupAt(x.Pos(), x.Name); v == nil {
                p := i.project
                if x.Sym != nil && x.Sym.Kind == ast.Rul {
                        v = p.Insert(x.Name, nil)
                } else {
                        v = types.NewDummy(p, i.Scope(), x.Name)
                }
        }
        return
}

func (i *Interpreter) selector(scope *types.Scope, p *types.Project, x *ast.SelectorExpr) (v types.Value) {
        var base types.Value
        switch t := x.X.(type) {
        case *ast.Ident:
                if base = p.Scope().Lookup(t.Name); base == nil {
                        runtime.Fail("'%s' undefined in '%s'", t.Name, p.Name())
                }
        default:
                if name := i.expr(scope, t).String(); name == "" {
                        if c, ok := t.(*ast.CallExpr); ok {
                                runtime.Fail("'%v' is empty", c.Name)
                        } else {
                                runtime.Fail("'%T' is empty", t)
                        }
                } else {
                        if base = p.Scope().Lookup(name); base == nil {
                                runtime.Fail("'%s' undefined in '%s'", name, p.Name())
                        }
                }
        }

        if base == nil {
                runtime.Fail("'%T' undefined in '%s'", x.X, p.Name())
        }

        if pn, _ := base.(*types.ProjectName); pn != nil {
                sub := pn.Imported()
                if sub == nil {
                        runtime.Fail("importee of %s is nil", pn.Name())
                }

                switch s := x.S.(type) {
                case *ast.Ident:
                        if obj := sub.Scope().Lookup(s.Name); obj == nil {
                                runtime.Fail("'%s' undefined in %s", s.Name, pn.Name())
                        } else {
                                v = obj
                        }
                case *ast.SelectorExpr:
                        v = i.selector(scope, sub, s)
                default:
                        if name := i.expr(scope, s).String(); name == "" {
                                if c, ok := s.(*ast.CallExpr); ok {
                                        runtime.Fail("'%v' is empty", c.Name)
                                } else {
                                        runtime.Fail("'%T' is empty", s)
                                }
                        } else if obj := sub.Scope().Lookup(name); obj == nil {
                                runtime.Fail("'%s' undefined in %s", name, pn.Name())
                        } else {
                                v = obj
                        }
                }
        } else {
                runtime.Fail("bad selection upon %T %v", base, base)
        }
        return
}

func (i *Interpreter) call(scope *types.Scope, x *ast.CallExpr) (v types.Value) {
        var name = i.expr(scope, x.Name)
        if obj, _ := name.(types.Object); obj != nil {
                v = i.Fold(x.Pos(), obj, i.exprs(scope, x.Args)...)
        } else if name != nil {
                runtime.Fail("unsupported name '%s' (%T, %T)", name, x.Name, name)
        } else {
                runtime.Fail("calling undefined object %v", x.Name)
        }
        return
}

func (i *Interpreter) recipe(scope *types.Scope, x *ast.RecipeExpr) (v types.Value) {
        if x.Dialect == "" {
                var elems []types.Value
                switch t := x.Elems[0].(type) {
                default: runtime.Fail("unimplemented recipe (%T)", t)
                case *ast.SelectorExpr, *ast.Ident:
                case *ast.UseDefineClause:
                }
                elems = append(elems, i.exprs(scope, x.Elems)...)
                //fmt.Printf("recipe: %T %T\n", x.Elems[0], elems[0])
                v = values.List(elems...)
        } else {
                elems := i.exprs(scope, x.Elems)
                v = values.Compound(elems...)
        }
        return
}

func (i *Interpreter) expr(scope *types.Scope, expr ast.Expr) (v types.Value) {
        switch x := expr.(type) {
        case *ast.Ident:
                v = i.ident(scope, x)
        case *ast.SelectorExpr:
                v = i.selector(scope, i.project, x)
        case *ast.CallExpr:
                v = i.call(scope, x)
        case *ast.RecipeExpr:
                v = i.recipe(scope, x)
        case *ast.BasicLit:
                v = values.Literal(x.Kind, x.Value)
        case *ast.Bareword:
                v = values.Bareword(x.Value)
        case *ast.Barecomp:
                v = values.Barecomp(i.exprs(scope, x.Elems)...)
        case *ast.Barefile:
                v = values.Barefile(i.expr(scope, x.Name), x.Ext)
        case *ast.PathExpr:
                v = values.Path(i.exprs(scope, x.Segments)...)
        case *ast.FlagExpr:
                v = values.Flag(i.expr(scope, x.Name))
        case *ast.CompoundLit:
                v = values.Compound(i.exprs(scope, x.Elems)...)
        case *ast.GroupExpr:
                v = values.Group(i.exprs(scope, x.Elems)...)
        case *ast.ListExpr:
                v = values.List(i.exprs(scope, x.Elems)...)
        case *ast.KeyValueExpr:
                v = values.Pair(i.expr(scope, x.Key), i.expr(scope, x.Value))
        case *ast.PercExpr:
                v = types.NewPercentPattern(i.project, i.expr(scope, x.X), i.expr(scope, x.Y))
        case *ast.UnaryExpr:
                v = i.unary(scope, x)
        case nil:
                v = values.None
        case *ast.UseDefineClause:
                v = &usedefine{
                        name: i.expr(scope, x.Name).String(),
                        value: i.expr(scope, x.Value),
                        pos: nil,
                }
        default:
                runtime.Fail("unimplemented expression (%T)", x)
        }
        return
}

func (i *Interpreter) exprs(scope *types.Scope, exprs []ast.Expr) (values []types.Value) {
        for _, x := range exprs {
                values = append(values, i.expr(scope, x))
        }
        return
}

func (i *Interpreter) use(spec *ast.UseSpec) error {
        runtime.Fail("unimplemented: %T\n", spec) // TODO: use
        return nil
}

func (i *Interpreter) eval(spec *ast.EvalSpec) (res types.Value, err error) {
        if num := len(spec.Props); num > 0 {
                scope := i.Scope()
                name := i.expr(scope, spec.Props[0])
                if _, fun := i.Scope().LookupAt(spec.EndPos, name.String()); fun != nil {
                        args := i.exprs(scope, spec.Props[1:])
                        res, _ = fun.(types.Caller).Call(args...)
                } else {
                        err = errors.New(fmt.Sprintf("undefined '%s'", name))
                        //fmt.Printf("error: `%v' is invalid\n", name)
                }
        }
        return
}

func (i *Interpreter) define(scope *types.Scope, d *ast.DefineClause) (obj types.Object, err error) {
        if i.project == nil {
                err = errors.New(fmt.Sprintf("define %v not in a project scope", d.Name))
                return
        }
        
        var (
                name = i.expr(scope, d.Name).String()
                v = i.expr(scope, d.Value)
        )

        if t, _ := v.(*types.Def); t != nil {
                v = t.Value()
        }
        
        
        if obj = i.project.Scope().Insert(types.NewDef(i.project, name, v)); obj != nil {
                if def, ok := obj.(*types.Def); ok {
                        def.Set(v)
                } else {
                        err = errors.New(fmt.Sprintf("name '%s' already taken", name))
                }
        }
        return
}

func (i *Interpreter) rule(scope *types.Scope, d *ast.RuleClause) (err error) {
        var (
                depends []types.Value
                recipes []types.Value
        )
        for i, depend := range i.exprs(scope, d.Depends) {
                //fmt.Printf("Interpreter.rule: %T %v (%v)\n", depend, depend, depend.String())
                switch entry := depend.(type) {
                case *types.RuleEntry, *values.BarefileValue, *values.PathValue, *types.PercentPattern:
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

        if p, ok := d.Program.(*ast.ProgramExpr); ok && p != nil {
                // mapping lexical objects
                for name, sym := range p.Scope.Symbols {
                        //fmt.Printf("sym: %v %T\n", name, sym)
                        auto := types.NewDef(i.project, name, values.None)
                        if alt := scope.Insert(auto); alt != nil {
                                runtime.Fail("%s already defined", name)
                        }
                        sym.Data = auto
                }
                
                if p.Values != nil {
                        recipes = i.exprs(scope, p.Values)
                }
        } else {
                return errors.New(fmt.Sprintf("unsupported program type"))
        }
        
        var modifiers []types.Value
        if d.Modifier != nil {
                modifiers = i.exprs(scope, d.Modifier.Elems)
        }
        
        var prog = runtime.NewProgram(i.Context, i.project, scope, depends, recipes...)
        if len(modifiers) > 0 {
                if err = prog.SetModifiers(modifiers...); err != nil {
                        return
                }
        }
        
        for _, target := range i.exprs(scope, d.Targets) {
                //fmt.Printf("Interpreter.rule: %T %v (%v)\n", target, target, target.String())
                switch entry := target.(type) {
                case *types.PercentPattern:
                        i.project.AddPercentPattern(entry, prog)
                default:
                        i.project.Insert(target.String(), prog)
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
                scope = i.Scope()
                linfo = i.loads[len(i.loads)-1]
                specPath = i.expr(scope, spec.Props[0]).String()
                params []types.Value
        )

        if len(spec.Props) > 1 {
                params = i.exprs(scope, spec.Props[1:])
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

                //fmt.Printf("declare: %s, %s, %s\n", name, relPath, absPath)
                
                dec = &declare{
                        project: i.Globe().NewProject(absPath, relPath, linfo.specPath, name),
                }
                linfo.declares[name] = dec

                var ms = dec.project.Scope()
                if ms.Insert(types.NewDef(dec.project, "/", values.String(absPath))) != nil {
                        panic(fmt.Sprintf("'$/' already defined"))
                }
                if ms.Insert(types.NewDef(dec.project, ".", values.String(relPath))) != nil {
                        panic(fmt.Sprintf("'$.' already defined"))
                }
                if ms.Insert(types.NewDef(dec.project, "..", values.String(relPathParent))) != nil {
                        panic(fmt.Sprintf("'$..' already defined"))
                }
        }

        if loader := linfo.loader; loader != nil {
                //fmt.Printf("DeclareProject: %s -> %s, %v\n", loader.Name(), dec.project.Name(), dec.s)

                if obj := loader.Scope().Lookup(name); obj == nil {
                        pn := types.NewProjectName(loader, name, dec.project)
                        loader.Scope().Insert(pn)
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
                pn := types.NewProjectName(i.project, loaded.Name(), loaded)
                i.project.Scope().Insert(pn)
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
                pn := types.NewProjectName(i.project, loaded.Name(), loaded)
                i.project.Scope().Insert(pn)
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
        return pc.define(pc.Scope(), clause)
}

func (pc *parseContext) DeclareRule(clause *ast.RuleClause) (parser.RuntimeSym, error) {
        //scope := types.NewScope(i.Scope(), d.TokPos, token.NoPos, "rule")
        //defer i.SetScope(i.SetScope(scope))
        return nil, pc.rule(pc.Scope(), clause)
}

func (pc *parseContext) EvalExpr(x ast.Expr) (s fmt.Stringer, err error) {
	defer func() {
		if e := recover(); e != nil {
                        err = errors.New(fmt.Sprintf("%v", e))
		}
        }()

        s = pc.expr(pc.Scope(), x)
        //fmt.Printf("EvalExpr: %T '%s'\n", x, s)
        return
}
