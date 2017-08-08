//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "path/filepath"
        "strconv"
        "strings"
        "errors"
        "fmt"
        "os"
)

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

func (prog *Program) interpret(context *types.Scope, pcd bool, i interpreter, out *types.Def, args... types.Value) (err error) {
        /* if pcd {
                workdir := filepath.Clean(prog.project.AbsPath())
                fmt.Printf("smart: Entering directory '%s'\n", workdir)
                defer fmt.Printf("smart: Leaving directory '%s'\n", workdir)
        } */
        
        var (
                value types.Value
                recipes []types.Value
        )
        for _, recipe := range prog.recipes {
                if v, e := types.Disclose(context/*prog.scope*/, recipe); e != nil {
                        return e
                } else if v != nil {
                        recipe = v
                }
                recipes = append(recipes, recipe) // types.EvalElems
        }
        
        value, err = i.evaluate(prog, args, recipes)
        if err == nil && value != nil {
                out.Assign(value)
        }
        return
}

func (prog *Program) modify(context *types.Scope, pcd bool, g *types.Group, out *types.Def) (err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name = g.Get(0).Strval()
        if name == "cd" {
                // does nothing
        } else if f, ok := modifiers[name]; ok {
                var value = out.Value
                if value, err = f(prog, value, g.Slice(1)...); err == nil && value !=  nil {
                        out.Assign(value)
                }
        } else if i, _ := interpreters[name]; i != nil {
                err = prog.interpret(context, pcd, i, out, g.Slice(1)...)
        } else {
                err = errors.New(fmt.Sprintf("no modifier or dialect '%s'", name))
        }
        return
}

func (prog *Program) prepare(context *types.Scope, entry *types.RuleEntry) (err error) {
        var (
                res types.Value
                dependList = values.List()
                depends []types.Value
        )

        prog.auto("^", dependList)

        // Convert the original depends
        for _, depend := range prog.depends {
                if v, e := types.Disclose(context, depend); e != nil {
                        return e
                } else if v != nil {
                        depend = v
                }
                //depends = append(depends, types.Join(depend)...)
                depends = append(depends, types.EvalElems(depend)...)
        }

        // TODO: using rules in a different project as prerequisites, e.g.
        //       [ c++.compiled-objects ]
        //       [ docker.instance-launched ]
        DependsLoop: for _, depend := range depends {
                //fmt.Printf("Program.prepare: %v: %T %v\n", entry.Name(), depend, depend)
                var (
                        project = prog.project
                        isFileEntry = false
                        file string
                        args []types.Value
                )
                DependSwitch: switch d := depend.(type) {
                case *types.Argumented:
                        fmt.Printf("Program.prepare: %s: argumented: %v\n", entry.Name(), d.Args)
                        depend, args = d.Value, d.Args; goto DependSwitch
                case *types.Bareword:
                        if p, e := project.Entry(d.Strval()); e != nil {
                                return e
                        } else if p != nil {
                                depend = p; goto DependSwitch
                        }
                        return errors.New(fmt.Sprintf("No such rule `%v' (required by `%s').", d, entry.Name()))
                case *types.Barefile:
                        file = d.Strval(); goto HandleFile
                case *types.Path:
                        file = d.Strval(); goto HandleFile
                case *types.ProjectName:
                        if ent := d.Project().DefaultEntry(); ent != nil {
                                depend = ent; goto DependSwitch
                        } else {
                                continue DependsLoop
                        }
                case *types.PercentPattern:
                        if stem := entry.Stem(); stem != "" {
                                name := d.MakeString(stem)
                                MapPatent: if p, e := project.Entry(name); e != nil {
                                        return e
                                } else if p != nil {
                                        depend = p; goto DependSwitch
                                }

                                //fmt.Printf("%v\n%v\n%v\n", context.Outer(), context, prog.project.Scope())

                                // Mapping entry from the context project
                                if proj := context.FindProject(); proj != nil {
                                        if proj != project {
                                                project = proj; goto MapPatent
                                        } else if proj.IsFile(name) {
                                                //fmt.Printf("file: %s (%s)\n", name, proj.Name())
                                                project, file = proj, name
                                                goto HandleFile
                                        }
                                }
                                return errors.New(fmt.Sprintf("No such rule `%v' (required by `%s' via `%v').", d.MakeString(stem), entry.Name(), d))
                        } else {
                                return errors.New(fmt.Sprintf("empty stem (%s, dependency %v)", entry, d))
                        }
                case *types.RuleEntry:
                        //fmt.Printf("Program.prepare: %v %v\n", d, d.Class())
                        var p *Program
                        if len(d.Programs()) > 0 {
                                p = d.Programs()[0].(*Program)
                        }
                        if p == nil {
                                switch d.Class() {
                                case types.FileRuleEntry:
                                        file = d.Strval(); goto HandleFile
                                case types.PatternFileRuleEntry:
                                        // A pattern entry without program can't
                                        // help to update the file.
                                        return errors.New(fmt.Sprintf("no rule to make file '%v'", d))
                                default:
                                        return errors.New(fmt.Sprintf("%v: '%v' requies update actions (%v)\n", entry, d, d.Class()))
                                }
                                break DependSwitch
                        }
                        
                        scope := context
                        if entry.Project() != d.Project() {
                                scope = entry.Project().Scope()
                        }
                        if res, err = p.Execute(scope, d, args, false); err == nil {
                                //var fromOther = p != nil && p.project != prog.project
                                //fmt.Printf("Program.prepare: %T %v (isFileEntry: %v) (res: %v) (err: %v) (%v)\n", depend, depend, isFileEntry, res, err, fromOther)
                                dd, _ := p.scope.Lookup("@").(*types.Def).Call()
                                //fmt.Printf("Program.prepare: updated %v\n", dd)
                                if isFileEntry {
                                        dependList.Append(values.Group(targetRegularKind, dd))
                                } else {
                                        switch d.Class() {
                                        case types.FileRuleEntry, types.PatternFileRuleEntry:
                                                dependList.Append(values.Group(targetRegularKind, dd))
                                        default:
                                                if res != nil && res != values.None {
                                                        dependList.Append(res)
                                                } else {
                                                        dependList.Append(d)
                                                }
                                        }
                                }
                        } else {
                                //fmt.Printf("Program.prepare: %T %v (%v)\n", depend, depend, err)
                                var s = err.Error()
                                if strings.HasPrefix(s, "Updating ") {
                                        s = "->" + strings.TrimPrefix(s, "Updating ")
                                } else {
                                        s = ", " + s
                                }
                                err = errors.New(fmt.Sprintf("Updating %v%v", entry.Name(), s))
                                break DependsLoop
                        }
                default:
                        return errors.New(fmt.Sprintf("Unknown depend `%T' (%v) (by `%s').", d, d, entry.Name()))
                }
                
                continue // done with non-file RuleEntry
                
                HandleFile: if file == "" {
                        continue
                }

                //fmt.Printf("Program.prepare: %s (%T %v) (%v)\n", file, depend, depend, context)
                if obj := context.Find(file); obj != nil {
                        if obj != depend { // ignore the same one
                                depend, isFileEntry = obj, true
                                goto DependSwitch
                        }
                }

                if p, e := project.Entry(file); e != nil {
                        return e
                } else if p != nil {
                        depend, isFileEntry = p, true
                        goto DependSwitch
                }

                fv := project.SearchFile(context, values.File(depend, file))
                if fv.Info != nil {
                        dependList.Append(fv)
                } else {
                        return errors.New(fmt.Sprintf("No such file `%v' (required by `%v')", fv, entry.Name()))
                }
        }
        //fmt.Printf("Program.prepare: %v: %v (%v)\n", entry, dependList, prog.project.Name())
        return
}

func (prog *Program) Getwd() string {
        for _, m := range prog.pipline {
                if g, ok := m.(*types.Group); ok && g != nil {
                        if n := len(g.Elems); n > 0 && g.Elems[0].Strval() == "cd" {
                                var s string
                                if n > 1 {
                                        s = filepath.Clean(g.Elems[1].Strval())
                                        if s == "-" { s = "" }
                                }
                                return s
                        }
                }
        }
        return filepath.Clean(prog.project.AbsPath())
}

func (prog *Program) Execute(context *types.Scope, entry *types.RuleEntry, args []types.Value, forced bool) (result types.Value, err error) {
        /*if entry.Name() == "lib.a" {
                //fmt.Printf("Program.Execute: %v: %v %v\n", entry, args, prog.depends)
                //fmt.Printf("Program.Execute: %v: %v %v\n", entry, args, prog.pipline)
                //fmt.Printf("Program.Execute: %v: %v\n", entry.Name(), context)
                fmt.Printf("Program.Execute: %v: %v\n", entry.Name(), args)
        }*/
        var pcd = entry.Class() != types.UseRuleEntry
        if workdir := prog.Getwd(); workdir != "" {
                if wd, _ := os.Getwd(); workdir != filepath.Clean(wd) {
                        // print-change-directory
                        if pcd {
                                fmt.Printf("smart: Entering directory '%s'\n", workdir)
                        }
                        if err = os.Chdir(workdir); err == nil {
                                if pcd {
                                        defer fmt.Printf("smart: Leaving directory '%s'\n", workdir)
                                }
                                defer os.Chdir(wd)
                        } else {
                                return
                        }
                }
        }

        for i, a := range args {
                // TODO: handle with Pair, map 'key => value' into
                // parameters.
                prog.auto(strconv.Itoa(i+1), a)
                if i < len(prog.params) {
                        name := prog.params[i]
                        prog.auto(name, a)
                }
        }

        // Calculate and prepare depends and files.
        if err = prog.prepare(context, entry); err != nil {
                return
        }

        /* if pcd {
                fmt.Printf("smart: Entering directory '%s'\n", workdir)
                defer fmt.Printf("smart: Leaving directory '%s'\n", workdir)
        } */
        
        if s := entry.Name(); prog.project.IsFile(s) {
                file := prog.project.SearchFile(context, values.File(entry, s))
                prog.auto("@", file)
        } else {
                prog.auto("@", entry)
        }

        var out = prog.auto("-", values.None)
        defer func() { result = out.Value }()
        
        // TODO: define modifiers in a project, e.g.
        // 
        //      some-modifier : - :
        //              smart statments going here...
        //              
        
        if len(prog.pipline) == 0 {
                // Using the default statements interpreter.
                if i, _ := interpreters[``]; i == nil {
                        err = errors.New("no default dialect")
                        return
                } else if err = prog.interpret(context, pcd, i, out, args...); err != nil {
                        // ...
                }
                return
        }

pipelineLoop:
        for _, v := range prog.pipline {
                switch op := v.(type) {
                case *types.Group:
                        if err = prog.modify(context, pcd, op, out); err != nil {
                                if p, ok := err.(*breaker); ok {
                                        if p.okay {
                                                err = nil
                                        } else {
                                                fmt.Printf("%s, required by '%s' (from %v)\n", p.message, entry.Name(), prog.project.RelPath())
                                        }
                                }
                                break pipelineLoop
                        }
                case *types.Bareword:
                        if i, _ := interpreters[op.Strval()]; i == nil {
                                err = errors.New(fmt.Sprintf("no dialect '%s', required by '%s'", op, entry.Name()))
                                return
                        } else if err = prog.interpret(context, pcd, i, out, args...); err != nil {
                                //fmt.Printf("interpret: %v\n", err)
                                break pipelineLoop
                        }
                default:
                        err = errors.New(fmt.Sprintf("unsupported modifier '%s'", v))
                        break pipelineLoop
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
