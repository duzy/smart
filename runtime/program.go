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
        //"strings"
        "strconv"
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
        
        var value types.Value
        value, err = i.evaluate(prog, args, prog.recipes)
        if err == nil && value != nil {
                out.Set(value)
        }
        return
}

func (prog *Program) modify(pcd bool, g *types.Group, out *types.Def) (err error) {
        // TODO: using rules in a different project to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        var name = g.Get(0).String()
        if f, ok := modifiers[name]; ok {
                var value, _ = out.Call()
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

func (prog *Program) prepare(entry *types.RuleEntry) (err error) {
        var (
                res types.Value
                dependList = values.List()
                depends []types.Value
        )

        prog.auto("^", dependList)

        for _, depend := range prog.depends {
                if closure, ok := depend.(*types.Closure); ok {
                        a, e := closure.Call(prog.scope)
                        if e != nil {
                                err = e
                                return
                        }
                        depends = append(depends, types.EvalElems(a)...)
                } else {
                        depends = append(depends, depend)
                }
        }

        // TODO: using rules in a different project as prerequisites, e.g.
        //       [ c++.compiled-objects ]
        //       [ docker.instance-launched ]
        dependsLoop: for _, depend := range depends {
                //fmt.Printf("Program.prepare: %T %v\n", depend, depend)
                var (
                        isFileEntry = false
                        file string
                        args []types.Value
                )
                dependSwitch: switch d := depend.(type) {
                case *types.ArgumentedEntry:
                        depend, args = d.RuleEntry, d.Args
                        goto dependSwitch
                case *types.RuleEntry:
                        //fmt.Printf("Program.prepare: %v %v\n", d, d.Class())
                        if res, err = d.Call(args...); err == nil {
                                var p, _ = d.Program().(*Program)
                                if p == nil {
                                        switch d.Class() {
                                        case types.FileRuleEntry, types.PatternFileRuleEntry:
                                                dependList.Append(values.Group(targetRegularKind, d))
                                        default:
                                                Fail("%v: '%v' requies update actions (%v)\n", entry, d, d.Class())
                                        }
                                        break dependSwitch
                                }
                                
                                //var fromOther = p != nil && p.project != prog.project
                                //fmt.Printf("Program.prepare: %T %v (isFileEntry: %v) (res: %v) (err: %v) (%v)\n", depend, depend, isFileEntry, res, err, fromOther)
                                dd, _ := p.scope.Lookup("@").(*types.Def)
                                dt, _ := dd.Call()
                                if isFileEntry {
                                        dependList.Append(values.Group(targetRegularKind, dt))
                                } else {
                                        switch d.Class() {
                                        case types.FileRuleEntry, types.PatternFileRuleEntry:
                                                dependList.Append(values.Group(targetRegularKind, dt))
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
                                //Fail("failed to update '%v' (%v)", entry, err)
                                break dependsLoop
                        }
                case *types.Barefile:
                        file = d.String(); goto handleFileEntry
                case *types.Path:
                        file = d.String(); goto handleFileEntry
                case *types.PercentPattern:
                        if stem := entry.Stem(); stem != "" {
                                var (
                                        dent = d.Entry(entry.Stem())
                                        name = dent.String()
                                )
                                if prog.project.IsFile(name) {
                                        //fmt.Printf("%v: %v -> %v (file)\n", entry, depend, dent)
                                        depend, file = dent, name; goto handleFileEntry
                                } else {
                                        //fmt.Printf("%v: %v -> %v (general)\n", entry, depend, dent)
                                        depend = dent; goto dependSwitch
                                }
                        } else {
                                Fail("empty stem (%s, dependency %v)", entry, d)
                        }
                default:
                        if types.IsDummy(d) {
                                sym, _ := d.(types.Object)
                                scope := sym.Parent()
                                if s := scope.Find(sym.Name()); s != nil {
                                        depend = s; goto dependSwitch
                                }
                                Fail("unknown dependency %s", sym.Name())
                        } else {
                                Fail("unknown dependency (%T %v)", d, d)
                        }
                }
                
                continue // done with non-file RuleEntry
                
                handleFileEntry: if file != "" {
                        //fmt.Printf("convert: %T %v\n", depend, depend)
                        if s := prog.scope.Find(file); s != nil {
                                depend, isFileEntry = s, true
                                goto dependSwitch
                        }
                        if p, stem := prog.project.MatchPattern(file); p != nil {
                                entry := p.Entry(stem)
                                //fmt.Printf("pattern: %T %T %v (%v, %v)\n", depend, p, p, stem, entry)
                                depend, isFileEntry = entry, true
                                goto dependSwitch
                        }

                        fv := prog.project.SearchFile(values.File(depend, file))
                        if fv.Info != nil {
                                dependList.Append(fv)
                        } else {
                                Fail("no rule to make file '%v'", fv)
                        }
                }
        }
        //fmt.Printf("Program.prepare: %v: %v (%v)\n", entry, dependList, prog.project.Name())
        return
}

func (prog *Program) Execute(entry *types.RuleEntry, args []types.Value, forced bool) (result types.Value, err error) {
        //fmt.Printf("Program.Execute: %v %v %v\n", entry, args, prog.depends)
        var (
                p = prog.project
                workdir = filepath.Clean(p.AbsPath())
                wd, _ = os.Getwd() //prog.context.Getwd()
                // print-change-directory
                pcd = entry.Class() != types.UseRuleEntry
        )
        if workdir != filepath.Clean(wd) {
                if pcd {
                        fmt.Printf("smart: Entering directory '%s'\n", workdir)
                }
                if err = os.Chdir(workdir); err == nil {
                        if pcd {
                                defer fmt.Printf("smart: Leaving directory '%s'\n", workdir)
                        }
                        defer os.Chdir(wd)
                } else {
                        Fail("%v", err)
                }
        }  else {
                //pcd = false
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
        if err = prog.prepare(entry); err != nil {
                //Fail("failed to update '%v' (%v)", entry, err)
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
        defer func() { result, _ = out.Call() }()
        
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
                        if i, _ := interpreters[op.String()]; i == nil {
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
