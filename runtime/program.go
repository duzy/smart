//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "path/filepath"
        "strconv"
        //"strings"
        "fmt"
        "os"
)

type dependPatternUnfit struct {
}

func (*dependPatternUnfit) Error() string { return "pattern unfit" }

type workinfo struct {
        dir, backdir string
        print bool
}

var workstack []*workinfo

func enterWorkdir(dir string, print bool) (wi *workinfo) {
        if wd, err := os.Getwd(); err == nil {
                if n := len(workstack); 0 < n && workstack[n-1].dir == dir {
                        print = false
                }
                if print {
                        fmt.Printf("smart: Entering directory '%s'\n", dir)
                }
                if err := os.Chdir(dir); err == nil {
                        wi = &workinfo{
                                dir: dir,
                                backdir: filepath.Clean(wd),
                                print: print,
                        }
                        workstack = append(workstack, wi)
                } else {
                        fmt.Fprintf(os.Stderr, "smart: chdir: %s\n", err)
                }
        } else {
                fmt.Fprintf(os.Stderr, "smart: %s\n", err)
        }
        return
}

func leaveWorkdir(wi *workinfo) {
        if n := len(workstack); 0 < n && workstack[n-1] == wi {
                if wi.print {
                        fmt.Printf("smart:  Leaving directory '%s'\n", wi.dir)
                }
                if err := os.Chdir(wi.backdir); err != nil {
                        fmt.Fprintf(os.Stderr, "smart: chdir: %s\n", err)
                }
                workstack = workstack[0:n-1]
        }
}

// Program (TODO: moving program into `types` package)
type Program struct {
        context *Context
        project  *types.Project
        scope   *types.Scope
        params  []string // named parameters
        depends []types.Value // *types.RuleEntry, *types.Barefile
        recipes []types.Value
        pipline []types.Value
}

func (prog *Program) Params() []string { return prog.params }
func (prog *Program) Project() *types.Project { return prog.project }
func (prog *Program) Depends() []types.Value { return prog.depends }
func (prog *Program) Recipes() []types.Value { return prog.recipes }
func (prog *Program) Pipeline() []types.Value { return prog.pipline }
func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) auto(name string, value interface{}) (auto *types.Def) {
        var alt types.Object
        if auto, alt = prog.scope.InsertDef(prog.project, name, values.Make(value)); alt != nil {
                //fmt.Printf("auto: %p %T %v\n", prog, sym, sym.Name())
                var found = false
                if auto, found = alt.(*types.Def); found {
                        auto.Assign(values.Make(value))
                } else {
                        Fail("Name '%v' already taken, not auto (%T)", name, alt)
                }
        }
        return
}

func (prog *Program) discloseRecipes(context *types.Scope) (recipes []types.Value, err error) {
        for _, recipe := range prog.recipes {
                if v, e := types.Disclose(context, recipe); e != nil {
                        return nil, e
                } else if v != nil {
                        recipe = v
                }
                recipes = append(recipes, recipe) // types.EvalElems
        }
        return
}

func (prog *Program) interpret(context *types.Scope, i interpreter, out *types.Def, params []types.Value) (err error) {
        var (
                recipes []types.Value
                target, value types.Value
        )
        if recipes, err = prog.discloseRecipes(context); err != nil {
                return
        }
        if value, err = i.evaluate(prog, context, params, recipes); err == nil {
                if value != nil {
                        out.Assign(value)
                }
                def := prog.scope.Lookup("@").(*types.Def)
                if target, err = def.Call(); err == nil {
                        _, _, err = prog.project.UpdateCmdHash(target, recipes)
                }
        }
        return
}

func (prog *Program) modify(context *types.Scope, g *types.Group, out *types.Def) (dialect string, err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name = g.Get(0).Strval()
        /*if name == "cd" {
                // does nothing
        } else*/ if f, ok := modifiers[name]; ok {
                var value = out.Value
                if value, err = f(prog, context, value, g.Slice(1)...); err == nil && value !=  nil {
                        out.Assign(value)
                }
        } else if i, _ := interpreters[name]; i != nil {
                err = prog.interpret(context, i, out, g.Slice(1))
                dialect = name // return dialect name
        } else {
                err = fmt.Errorf("no modifier or dialect '%s'", name)
        }
        return
}

func (prog *Program) Getwd(context *types.Scope) string {
        /*for _, m := range prog.pipline {
                if g, ok := m.(*types.Group); ok && g != nil {
                        if n := len(g.Elems); n > 0 && g.Elems[0].Strval() == "cd" {
                                var s string
                                if n > 1 {
                                        if v, e := types.Disclose(context, g.Elems[1]); e != nil {
                                                // TODO: error...
                                        } else if v != nil {
                                                s = filepath.Clean(v.Strval())
                                                if s == "-" { s = "" }
                                        }
                                }
                                return s
                        }
                }
        }*/
        return filepath.Clean(prog.project.AbsPath())
}

func (prog *Program) hasCDDash() (res bool) {
        for _, p := range prog.pipline {
                if g, _ := p.(*types.Group); g != nil {
                        if a := g.Elems; len(a) > 1 && a[0].Strval() == "cd" && a[1].Strval() == "-" {
                                res = true
                        }
                }
        }
        return
}

func (prog *Program) Execute(context *types.Scope, entry *types.RuleEntry, args []types.Value) (result types.Value, err error) {
        if workdir := prog.Getwd(context); workdir != "" {
                var printCD = entry.Class() != types.UseRuleEntry && !prog.hasCDDash()
                defer leaveWorkdir(enterWorkdir(workdir, printCD))
                prog.auto(theCurrWorkDirDef, workdir)
        }

        if false {
                fmt.Printf("execute: %v\n", entry.Name())
        }

        var argn = 0
        for _, a := range args {
                switch t := a.(type) {
                case *types.Pair:
                        prog.auto(t.Key.Strval(), t.Value)
                default:
                        prog.auto(strconv.Itoa(argn+1), a)
                        if argn < len(prog.params) {
                                prog.auto(prog.params[argn], a)
                        }
                        argn += 1
                }
        }

        if s := entry.Name(); prog.project.IsFile(s) {
                file := prog.project.SearchFile(context, values.File(entry, s))
                prog.auto("@", file)
        } else {
                prog.auto("@", entry)
        }

        // Calculate and prepare depends and files.
        pc := types.NewPrepareContext(prog, context, entry, nil, values.List())
        if err = pc.Prepare(prog.depends); err != nil {
                fmt.Fprintf(os.Stdout, "%s: %s\n", entry.Position, err)
                return
        } else if pc.Targets().Len() > 0 {
                var elems = pc.Targets().Elems[:]
                for i := 0; i < len(elems); i += 1 {
                        for j := i + 1; j < len(elems); j += 1 {
                                if dependEquals(elems[i], elems[j]) {
                                        elems = append(elems[:j], elems[j+1:]...)
                                        j -= 1
                                }
                        }
                }
                pc.Targets().Elems = elems
                prog.auto("<", pc.Targets().Elems[0])
                prog.auto("^", pc.Targets())
        }

        var out = prog.auto("-", values.None)
        defer func() { result = out.Value }()

        // TODO: define modifiers in a project, e.g.
        // 
        //      some-modifier : - :
        //              smart statments going here...
        //              
        var dialect string
        ForPipeline: for _, v := range prog.pipline {
                switch op := v.(type) {
                case *types.Group:
                        var lang string
                        if lang, err = prog.modify(context, op, out); err != nil {
                                if p, ok := err.(*breaker); ok {
                                        if p.okay {
                                                // Discard err and change dialect to
                                                // avoid default interpreter being
                                                // called.
                                                err, dialect = nil, "--"
                                        } else {
                                                fmt.Printf("%s, required by '%s' (from %v)\n", p.message, entry.Name(), prog.project.RelPath())
                                        }
                                }
                                break ForPipeline
                        } else if lang != "" && dialect == "" {
                                dialect = lang
                        }
                default:
                        err = fmt.Errorf("unsupported modifier `%s' (%T)", v, v)
                        break ForPipeline
                }
        }
        if err == nil && dialect == "" {
                // Using the default statements interpreter.
                if i, _ := interpreters[dialect]; i == nil {
                        err = fmt.Errorf("no default dialect")
                } else {
                        err = prog.interpret(context, i, out, nil)
                }
        }
        return
}

func (prog *Program) SetModifiers(modifiers... types.Value) (err error) {
        prog.pipline = modifiers
        return
}

func (context *Context) NewProgram(project *types.Project, params []string, scope *types.Scope, depends []types.Value, recipes... types.Value) *Program {
        return &Program{
                context:     context,
                project:     project,
                scope:       scope,
                params:      params,
                depends:     depends, // *types.RuleEntry, *types.Barefile
                recipes:     recipes,
        }
}

func dependEquals(a, b types.Value) bool {
        if a == b {
                return true
        }

        // TODO: more advanced checking "the same depend"
        
        return a.Strval() == b.Strval()
}
