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
        //"strings"
        "errors"
        "fmt"
        "os"
)

// Program (TODO: moving program into `types` package)
type Program struct {
        context *Context
        project  *types.Project
        scope   *types.Scope
        depends []types.Value // *types.RuleEntry, *values.BarefileValue
        recipes []types.Value
        pipline []types.Value
}

func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) auto(name string, value interface{}) (auto *types.Def) {
        var alt types.Object
        if auto, alt = prog.scope.InsertNewDef(prog.project, name, values.Make(value)); alt != nil {
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

func (prog *Program) interpret(i interpreter, out *types.Def, args... types.Value) (err error) {
        var value types.Value
        value, err = i.evaluate(prog, args, prog.recipes)
        if err == nil && value != nil {
                out.Set(value)
        }
        return
}

func (prog *Program) modify(g *types.GroupValue, out *types.Def) (err error) {
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
                err = prog.interpret(i, out, g.Slice(1)...)
        } else {
                err = errors.New(fmt.Sprintf("no modifier or dialect '%s'", name))
        }
        return
}

func (prog *Program) prepare(entry *types.RuleEntry) (err error) {
        var (
                res types.Value
                depends = values.List()
        )

        prog.auto("...", depends)

        // TODO: using rules in a different project as prerequisites, e.g.
        //       [ c++.compiled-objects ]
        //       [ docker.instance-launched ]
        dependLoop: for _, depend := range prog.depends {
                //fmt.Printf("Program.prepare: %T %v (%p)\n", depend, depend, depend)
                var (
                        isFileEntry = false
                        file string
                )
                dependSwitch: switch d := depend.(type) {
                case *types.RuleEntry:
                        if res, err = d.Call(); err == nil {
                                //fmt.Printf("Program.prepare: %T %v (%v) (isFileEntry:%v)\n", depend, depend, res, isFileEntry)
                                if isFileEntry {
                                        depends.Append(values.Group(targetRegularKind, d))
                                } else if res == values.None {
                                        //fmt.Printf("Program.prepare: %T %v (%v)\n", depend, d, d.Kind())
                                        switch d.Class() {
                                        case types.FileRuleEntry, types.PatternFileRuleEntry:
                                                depends.Append(values.Group(targetRegularKind, d))
                                        }
                                } else if res != nil {
                                        depends.Append(res)
                                } else {
                                        depends.Append(d)
                                }
                        } else {
                                //fmt.Printf("Program.prepare: %T %v (%v)\n", depend, depend, err)
                                Fail("failed to update '%v' (%v)", entry, err); break dependLoop
                        }
                case *types.BarefileValue:
                        file = d.String(); goto handleFileEntry
                case *types.PathValue:
                        file = d.String(); goto handleFileEntry
                case *types.PercentPattern:
                        if stem := entry.Stem(); stem != "" {
                                var (
                                        dent = d.Entry(entry.Stem())
                                        name = dent.String()
                                )
                                switch prog.project.EntryClass(name) {
                                case types.GeneralRuleEntry:
                                        //fmt.Printf("%v: %v -> %v (general)\n", entry, depend, dent)
                                        depend = dent; goto dependSwitch
                                case types.FileRuleEntry:
                                        //fmt.Printf("%v: %v -> %v (file)\n", entry, depend, dent)
                                        depend, file = dent, name; goto handleFileEntry
                                default:
                                        Fail("unknown dependency (%v)", dent)
                                }
                        } else {
                                Fail("empty stem (%s, dependency %v)", entry, d)
                        }
                default:
                        if types.IsDummy(d) {
                                sym, _ := d.(types.Object)
                                scope := sym.Parent()
                                if _, s := scope.LookupAt(token.NoPos, sym.Name()); s != nil {
                                        depend = s; goto dependSwitch
                                }
                                Fail("unknown dependency %s", sym.Name())
                        } else {
                                Fail("unknown dependency (%T)", d)
                        }
                }
                
                continue // done with RuleEntry
                
                handleFileEntry: if file != "" {
                        //fmt.Printf("convert: %T %v\n", depend, depend)
                        if _, s := prog.scope.LookupAt(token.NoPos, file); s != nil {
                                depend, isFileEntry = s, true
                                goto dependSwitch
                        }
                        if p, stem := prog.project.MatchPattern(file); p != nil {
                                entry := p.Entry(stem)
                                //fmt.Printf("pattern: %T %T %v (%v, %v)\n", depend, p, p, stem, entry)
                                depend, isFileEntry = entry, true
                                goto dependSwitch
                        }
                        
                        if _, err := os.Stat(file); err == nil {
                                depends.Append(depend)
                        } else {
                                Fail("no rule to make file '%v'", depend)
                        }
                }
        }
        //fmt.Printf("Program.prepare: %v: %v\n", entry, depends)
        return
}

func (prog *Program) Execute(entry *types.RuleEntry, args []types.Value, forced bool) (result types.Value, err error) {
        //fmt.Printf("Program.Execute: %v %v %v\n", entry, args, prog.depends)
        var (
                p = prog.project
                workdir = p.AbsPath()
                wd, _ = os.Getwd()
                //top = prog.context.Getwd()
        )
        //fmt.Printf("%s: %s, %s; %s\n", p.Name(), p.RelPath(), p.AbsPath(), wd)
        if workdir != wd {
                fmt.Printf("smart: Entering directory '%s'\n", workdir)
                if err = os.Chdir(workdir); err == nil {
                        defer func() {
                                fmt.Printf("smart: Leaving directory '%s'\n", workdir)
                                os.Chdir(wd)
                        }()
                } else {
                        Fail("%v", err)
                }
        }
        
        // Calculate depends and files.
        if err = prog.prepare(entry); err != nil {
                return
        }

        var (
                _   = prog.auto("@", entry)
                out = prog.auto("-", values.None)
        )
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
                } else if err = prog.interpret(i, out, args...); err != nil {
                        // ...
                }
                return
        }

pipelineLoop:
        for _, v := range prog.pipline {
                switch op := v.(type) {
                case *types.GroupValue:
                        if err = prog.modify(op, out); err != nil {
                                if p, ok := err.(*breaker); ok {
                                        if p.okay {
                                                err = nil
                                        } else {
                                                fmt.Printf("%s, required by '%s'\n", p.message, entry.Name())
                                        }
                                }
                                break pipelineLoop
                        }
                case *types.BarewordValue:
                        if i, _ := interpreters[op.String()]; i == nil {
                                err = errors.New(fmt.Sprintf("no dialect '%s', required by '%s'", op, entry.Name()))
                                return
                        } else if err = prog.interpret(i, out, args...); err != nil {
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

func (context *Context) NewProgram(project *types.Project, scope *types.Scope, depends []types.Value, recipes... types.Value) *Program {
        return &Program{
                context:     context,
                project:     project,
                scope:       scope,
                depends:     depends, // *types.RuleEntry, *types.BarefileValue
                recipes:     recipes,
        }
}
