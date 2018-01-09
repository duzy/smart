//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "github.com/duzy/smart/token"
        //"github.com/duzy/smart/values"
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
        project *Project
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
                        prog.auto(TheCurrWorkDirDef, &String{project.AbsPath()})
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
        globe   *Globe
        project *Project
        scope   *Scope
        disctx  *Scope
        caller  *Preparer
        params  []string // named parameters
        depends []Value // *RuleEntry, *Barefile
        recipes []Value
        pipline []Value
        position token.Position
}

func (prog *Program) Scope() *Scope { return prog.scope }

func (prog *Program) setCallerContext(pc *Preparer, ctx *Scope) (pc0 *Preparer, ctx0 *Scope) {
        pc0, ctx0 = prog.caller, prog.disctx
        prog.caller, prog.disctx = pc, ctx
        return
}

func (prog *Program) auto(name string, value Value) (auto *Def) {
        var alt Object
        if auto, alt = prog.scope.InsertDef(prog.project, name, value); alt != nil {
                var found = false
                if auto, found = alt.(*Def); found {
                        auto.Assign(value)
                } else {
                        Fail("Name '%v' already taken, not auto (%T)", name, alt)
                }
        }
        return
}

func (prog *Program) discloseRecipes() (recipes []Value, err error) {
        context := prog.disctx
        if context == nil {
                context = prog.Scope()
        }
        for _, recipe := range prog.recipes {
                if v, e := Disclose(context, recipe); e != nil {
                        return nil, e
                } else if v != nil {
                        recipe = v
                }
                recipes = append(recipes, recipe) // EvalElems
        }
        return
}

func (prog *Program) interpret(i Interpreter, out *Def, params []Value) (err error) {
        var (
                recipes []Value
                target, value Value
        )
        if recipes, err = prog.discloseRecipes(); err != nil {
                return
        }
        if value, err = i.Evaluate(prog, params, recipes); err == nil {
                if value != nil {
                        out.Assign(value)
                }
                def := prog.scope.Lookup("@").(*Def)
                if target, err = def.Call(); err == nil {
                        _, _, err = prog.project.UpdateCmdHash(target, recipes)
                }
        }
        return
}

func (prog *Program) modify(g *Group, out *Def) (dialect string, err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name = g.Get(0).Strval()
        if f, ok := modifiers[name]; ok {
                var value = out.Value
                if value, err = f(prog, value, g.Slice(1)...); err == nil && value !=  nil {
                        out.Assign(value)
                }
        } else if i, _ := dialects[name]; i != nil {
                err = prog.interpret(i, out, g.Slice(1))
                dialect = name // return dialect name
        } else {
                err = fmt.Errorf("no modifier or dialect '%s'", name)
        }
        return
}

func (prog *Program) hasCDDash() (res bool) {
        for _, p := range prog.pipline {
                if g, _ := p.(*Group); g != nil {
                        if a := g.Elems; len(a) > 1 && a[0].Strval() == "cd" && a[1].Strval() == "-" {
                                res = true
                        }
                }
        }
        return
}

func (prog *Program) Execute(entry *RuleEntry, args []Value) (result Value, err error) {
        defer leaveWorkdir(enterWorkdir(prog, entry.Class() != UseRuleEntry))

        if trace_prepare {
                fmt.Printf("program.Execute: %v (%v) (%v) (%v) (%v)\n", entry.name, entry.file, entry, prog.depends, prog.project.AbsPath())
        }

        var argn = 0
        for _, a := range args {
                switch t := a.(type) {
                case *Pair:
                        prog.auto(t.Key.Strval(), t.Value)
                default:
                        prog.auto(strconv.Itoa(argn+1), a)
                        if argn < len(prog.params) {
                                prog.auto(prog.params[argn], a)
                        }
                        argn += 1
                }
        }

        switch entry.class {
        case ExplicitFileEntry, StemmedFileEntry:
                if entry.file != nil {
                        prog.auto("@", entry.file)
                } else {
                        // prog.auto("@", prog.project.SearchFile(entry.name))
                        if trace_prepare {
                                fmt.Printf("program.Execute: %v (unknown) (%v)\n", entry.name, entry.class)
                        }
                        err = fmt.Errorf("unknown file '%v'", entry.name)
                        fmt.Fprintf(os.Stdout, "%s: %s\n", entry.Position, err)
                        return
                }
        default:
                prog.auto("@", entry)
        }

        // Calculate and prepare depends and files.
        pc := NewPreparer(prog, entry)
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

        var out = prog.auto("-", UniversalNone)
        defer func() { result = out.Value }()

        // Chdir again to ensure workdir was not changed by preparation.
        if false {
                if err = os.Chdir(prog.project.AbsPath()); err != nil {
                        fmt.Printf("smart: Chdir '%s'\n", prog.project.AbsPath())
                        return
                }
        }

        // TODO: define modifiers in a project, e.g.
        // 
        //      some-modifier : - :
        //              smart statments going here...
        //              
        var dialect string
        ForPipeline: for _, v := range prog.pipline {
                switch op := v.(type) {
                case *Group:
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
                if i, _ := dialects[dialect]; i == nil {
                        err = fmt.Errorf("no default dialect")
                } else {
                        err = prog.interpret(i, out, nil)
                }
        }
        return
}

func (prog *Program) SetModifiers(modifiers... Value) (err error) {
        prog.pipline = modifiers
        return
}

func NewProgram(globe *Globe, position token.Position, project *Project, params []string, scope *Scope, depends []Value, recipes... Value) *Program {
        return &Program{
                globe:    globe,
                project:  project,
                scope:    scope,
                params:   params,
                depends:  depends,
                recipes:  recipes,
                position: position,
        }
}

func dependEquals(a, b Value) bool {
        if a == b {
                return true
        }

        // TODO: more advanced checking "the same depend"
        
        return a.Strval() == b.Strval()
}
