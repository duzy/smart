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
        depends []types.Value // *types.RuleEntry, *values.BarefileValue
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
                        auto.Set(values.Make(value))
                } else {
                        Fail("name '%v' already taken", name)
                }
        }
        return
}

func (prog *Program) interpret(pcd bool, i interpreter, out *types.Def, args... types.Value) (err error) {
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
                if v, e := types.Disclose(prog.scope, recipe); e != nil {
                        return e
                } else if v != nil {
                        //fmt.Printf("recipe: %v -> %v\n", recipe, v)
                        recipe = v
                }
                recipes = append(recipes, recipe)
        }
        
        value, err = i.evaluate(prog, args, recipes)
        if err == nil && value != nil {
                out.Set(value)
        }
        return
}

func (prog *Program) modify(pcd bool, g *types.Group, out *types.Def) (err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name = g.Get(0).Strval()
        if name == "cd" {
                // does nothing
        } else if f, ok := modifiers[name]; ok {
                var value = out.Value
                if value, err = f(prog, value, g.Slice(1)...); err == nil && value !=  nil {
                        out.Set(value)
                }
        } else if i, _ := interpreters[name]; i != nil {
                err = prog.interpret(pcd, i, out, g.Slice(1)...)
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
                if v, e := types.Disclose(prog.scope, depend); e != nil {
                        return e
                } else if v != nil {
                        depend = v
                }
                //if vr, ok := depend.(types.Valuer); ok {
                //        depend = vr.Value
                //}
                depends = append(depends, types.EvalElems(depend)...)
        }

        // TODO: using rules in a different project as prerequisites, e.g.
        //       [ c++.compiled-objects ]
        //       [ docker.instance-launched ]
        DependsLoop: for _, depend := range depends {
                //fmt.Printf("Program.prepare: %T %v\n", depend, depend)               
                var (
                        isFileEntry = false
                        file string
                        args []types.Value
                )
                DependSwitch: switch d := depend.(type) {
                case *types.ArgumentedEntry:
                        depend, args = d.RuleEntry, d.Args
                        goto DependSwitch
                case *types.RuleEntry:
                        //fmt.Printf("Program.prepare: %v %v\n", d, d.Class())
                        var p, _ = d.Program().(*Program)
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
                                dd, _ := p.scope.Lookup("@").(*types.Def)
                                if isFileEntry {
                                        dependList.Append(values.Group(targetRegularKind, dd.Value))
                                } else {
                                        switch d.Class() {
                                        case types.FileRuleEntry, types.PatternFileRuleEntry:
                                                dependList.Append(values.Group(targetRegularKind, dd.Value))
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
                                //fmt.Printf("%s\n", s)
                                
                                if strings.HasPrefix(s, "updating ") {
                                        s = "->" + strings.TrimPrefix(s, "updating ")
                                } else {
                                        s = ", " + s
                                }
                                err = errors.New(fmt.Sprintf("updating %v%v", entry, s))
                                break DependsLoop
                        }
                case *types.Barefile:
                        file = d.Strval(); goto HandleFile
                case *types.Path:
                        file = d.Strval(); goto HandleFile
                case *types.PercentPattern:
                        if stem := entry.Stem(); stem != "" {
                                var (
                                        dent types.Value
                                        name = dent.Strval()
                                )
                                if dent, err = d.MakeConcreteEntry(nil, entry.Stem()); err != nil {
                                        return err
                                }
                                if prog.project.IsFile(name) {
                                        //fmt.Printf("%v: %v -> %v (file)\n", entry, depend, dent)
                                        depend, file = dent, name; goto HandleFile
                                } else {
                                        //fmt.Printf("%v: %v -> %v (general)\n", entry, depend, dent)
                                        depend = dent; goto DependSwitch
                                }
                        } else {
                                return errors.New(fmt.Sprintf("empty stem (%s, dependency %v)", entry, d))
                        }
                default:
                        return errors.New(fmt.Sprintf("unknown dependency (%T %v)", d, d))
                }
                
                continue // done with non-file RuleEntry
                
                HandleFile: if file != "" {
                        //fmt.Printf("Program.prepare: %s (%T %v) (%v)\n", file, depend, depend, context)
                        if obj := context.Find(file); obj != nil {
                                if obj == depend { // ignore the same one
                                        goto FindPatterns
                                } else {
                                        depend, isFileEntry = obj, true
                                        goto DependSwitch
                                }
                        }
                }
                FindPatterns: if file != "" {
                        // TODO: Improves patter searching on base chain. 
                        //fmt.Printf("Program.prepare: FindPatterns: %v: %v\n", prog.project.Name(), file)
                        if pss := prog.project.FindPatterns(file); pss != nil {
                                //fmt.Printf("Program.prepare: FoundPatterns: %v %v\n", file, len(pss))
                                for _, ps := range pss {
                                        p := ps.Patent.Program().(*Program)
                                        if p == nil {
                                                // goto SearchFile
                                        }

                                        //fmt.Printf("Program.prepare: %v -> %v (%v)\n", ps.Pattern, p.depends, ps.Stem)
                                        if entry, err := ps.MakeConcreteEntry(); err == nil {
                                                //fmt.Printf("Program.prepare: pattern: %T %T %v (%v, %v)\n", depend, ps.Pattern, ps.Pattern, ps.Stem, entry)
                                                depend, isFileEntry = entry, true
                                                goto DependSwitch
                                        } else {
                                                return err
                                        }
                                }
                        }
                }
                /*SearchFile:*/ if file != "" {
                        fv := prog.project.SearchFile(values.File(depend, file))
                        if fv.Info != nil {
                                dependList.Append(fv)
                        } else {
                                return errors.New(fmt.Sprintf("no such file '%v' (required by %v)", fv, entry))
                        }
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
        //fmt.Printf("Program.Execute: %v %v %v\n", entry, args, prog.depends)
        //fmt.Printf("Program.Execute: %v %v %v\n", entry, args, prog.pipline)
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
                file := prog.project.SearchFile(values.File(entry, s))
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
                } else if err = prog.interpret(pcd, i, out, args...); err != nil {
                        // ...
                }
                return
        }

pipelineLoop:
        for _, v := range prog.pipline {
                switch op := v.(type) {
                case *types.Group:
                        if err = prog.modify(pcd, op, out); err != nil {
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
                        } else if err = prog.interpret(pcd, i, out, args...); err != nil {
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
