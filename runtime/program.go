//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        //"path/filepath"
        "strconv"
        //"strings"
        "fmt"
        "os"
)

type dependPatternUnfit struct {
}

func (*dependPatternUnfit) Error() string { return "pattern unfit" }

const trace_workdir = false

type workinfo struct {
        project *types.Project
        print bool
}

var workstack []*workinfo

func enterWorkdir(prog *Program, print bool) (wi *workinfo) {
        var (
                project = prog.project
                l = len(workstack)
        )
        if l == 0 {
                // Push the initial work record. 
                var main = prog.globe.Main()
                workstack, l = append(workstack, &workinfo{ main, false }), 1
                if trace_workdir {
                        fmt.Printf("entering: %v (init: %v)\n", project.AbsPath(), main.AbsPath())
                }
        }
        if wd, err := os.Getwd(); err == nil {
                if s := workstack[l-1].project.AbsPath(); s != wd {
                        fmt.Fprintf(os.Stderr, "smart: diverged `%s` `%s`\n", wd, s)
                }
                if print = print && !prog.hasCDDash(); print {
                        for i := l-1; i > -1; i-- {
                                if w := workstack[i]; w.project.AbsPath() == project.AbsPath() {
                                        if w.print || i == 0 {
                                                print = false
                                                break
                                        }
                                }
                        }
                }
                if trace_workdir {
                        fmt.Printf("entering: %v (%v)\n", project.AbsPath(), print)
                }
                if print {
                        fmt.Printf("smart: Entering directory '%s'\n", project.AbsPath())
                }
                if err := os.Chdir(project.AbsPath()); err == nil {
                        prog.auto(theCurrWorkDirDef, project.AbsPath())
                        wi = &workinfo{ project, print }
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
        // Note that 0 < n, as the first record should not be removed.
        if n := len(workstack)-1; 0 < n && workstack[n] == wi {
                if wi.print {
                        fmt.Printf("smart:  Leaving directory '%s'\n", wi.project.AbsPath())
                }

                // Pop out the top record.
                workstack = workstack[0:n]

                // Go back to previous dir.
                if n--; 0 <= n && n < len(workstack) {
                        if err := os.Chdir(workstack[n].project.AbsPath()); err != nil {
                                fmt.Fprintf(os.Stderr, "smart: chdir: %s\n", err)
                        }
                } else {
                        fmt.Fprintf(os.Stderr, "smart: wrong workstack (%d, %d)\n", n, len(workstack))
                }
        }
}

// Program (TODO: moving program into `types` package)
type Program struct {
        globe   *types.Globe
        project *types.Project
        scope   *types.Scope
        disctx  *types.Scope
        params  []string // named parameters
        depends []types.Value // *types.RuleEntry, *types.Barefile
        recipes []types.Value
        pipline []types.Value
        position token.Position
}

func (prog *Program) Params() []string { return prog.params }
func (prog *Program) Project() *types.Project { return prog.project }
func (prog *Program) Position() token.Position { return prog.position }
func (prog *Program) Depends() []types.Value { return prog.depends }
func (prog *Program) Recipes() []types.Value { return prog.recipes }
func (prog *Program) Pipeline() []types.Value { return prog.pipline }
func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) SetContext(scope *types.Scope) (prev *types.Scope) {
        prev = prog.disctx
        prog.disctx = scope
        return
}

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

func (prog *Program) discloseRecipes() (recipes []types.Value, err error) {
        context := prog.disctx
        if context == nil {
                context = prog.Scope()
        }
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

func (prog *Program) interpret(i interpreter, out *types.Def, params []types.Value) (err error) {
        var (
                recipes []types.Value
                target, value types.Value
        )
        if recipes, err = prog.discloseRecipes(); err != nil {
                return
        }
        if value, err = i.evaluate(prog, params, recipes); err == nil {
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

func (prog *Program) modify(g *types.Group, out *types.Def) (dialect string, err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name = g.Get(0).Strval()
        if f, ok := modifiers[name]; ok {
                var value = out.Value
                if value, err = f(prog, value, g.Slice(1)...); err == nil && value !=  nil {
                        out.Assign(value)
                }
        } else if i, _ := interpreters[name]; i != nil {
                err = prog.interpret(i, out, g.Slice(1))
                dialect = name // return dialect name
        } else {
                err = fmt.Errorf("no modifier or dialect '%s'", name)
        }
        return
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

func (prog *Program) Execute(entry *types.RuleEntry, args []types.Value) (result types.Value, err error) {
        defer leaveWorkdir(enterWorkdir(prog, entry.Class() != types.UseRuleEntry))

        if false {
                fmt.Printf("program.Execute: %v %v (%v)\n", entry, prog.depends, prog.project.AbsPath())
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
                prog.auto("@", prog.project.SearchFile(s))
        } else {
                prog.auto("@", entry)
        }

        // Calculate and prepare depends and files.
        pc := types.NewPreparer(prog, entry)
        if err = pc.Prepare(prog.depends); err != nil {
                if false {
                        fmt.Fprintf(os.Stdout, "%s: %s\n", entry.Position, err)
                }
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
                        if lang, err = prog.modify(op, out); err != nil {
                                if p, ok := err.(*breaker); ok {
                                        if p.okay {
                                                // Discard err and change dialect to
                                                // avoid default interpreter being
                                                // called.
                                                err, dialect = nil, "--"
                                        }
                                }
                                if err != nil {
                                        err = fmt.Errorf("%v: %v", op, err)
                                }
                                break ForPipeline
                        } else if lang != "" && dialect == "" {
                                dialect = lang
                        }
                default:
                        err = fmt.Errorf("unknown modifier (%T `%v')", v, v)
                        break ForPipeline
                }
        }
        if err == nil && dialect == "" {
                // Using the default statements interpreter.
                if i, _ := interpreters[dialect]; i == nil {
                        err = fmt.Errorf("no default dialect")
                } else {
                        err = prog.interpret(i, out, nil)
                }
        }
        return
}

func (prog *Program) SetModifiers(modifiers... types.Value) (err error) {
        prog.pipline = modifiers
        return
}

func (context *Context) NewProgram(position token.Position, project *types.Project, params []string, scope *types.Scope, depends []types.Value, recipes... types.Value) *Program {
        return &Program{
                globe:    context.globe,
                project:  project,
                scope:    scope,
                params:   params,
                depends:  depends,
                recipes:  recipes,
                position: position,
        }
}

func dependEquals(a, b types.Value) bool {
        if a == b {
                return true
        }

        // TODO: more advanced checking "the same depend"
        
        return a.Strval() == b.Strval()
}
