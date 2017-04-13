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
        "path/filepath"
        //"strings"
        "errors"
        "fmt"
        "os"
)

// Program (TODO: moving program into `types` package)
type Program struct {
        context *Context
        module  *types.Module
        scope   *types.Scope
        depends []types.Value // *types.RuleEntry, *values.BarefileValue
        recipes []types.Value
        pipline []types.Value
}

func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) auto(name string, value interface{}) (auto *types.Def) {
        if sym := prog.scope.Lookup(name); sym == nil {
                auto = types.NewDef(prog.module, name, values.Make(value))
                prog.scope.Insert(auto)
        } else {
                //fmt.Printf("auto: %p %T %v\n", prog, sym, sym.Name())
                var found = false
                if auto, found = sym.(*types.Def); found {
                        auto.Set(values.Make(value))
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

func (prog *Program) modify(g *values.GroupValue, out *types.Def) (err error) {
        // TODO: using rules in a different module to implement modifiers, e.g.
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

        // TODO: using rules in a different module as prerequisites, e.g.
        //       [ c++.compiled-objects ]
        //       [ docker.instance-launched ]
dependLoop:
        for _, depend := range prog.depends {
                //fmt.Printf("Program.prepare: %T %v (%p)\n", depend, depend, depend)
        dependSwitch:
                switch d := depend.(type) {
                case *values.BarefileValue:
                        if _, s := prog.scope.LookupAt(token.NoPos, d.String()); s != nil {
                                depend = s
                                goto dependSwitch
                        } else {
                                if p, stem := prog.module.MatchPattern(d.String()); p != nil {
                                        //fmt.Printf("pattern: %v %v (%v)\n", depend, p, stem)
                                        depend = p.Entry(stem)
                                        goto dependSwitch
                                }
                                if _, err := os.Stat(d.String()); err == nil {
                                        depends.Append(d)
                                } else {
                                        Fail("no file or directory %v", d)
                                }
                        }
                case *types.RuleEntry:
                        if res, err = d.Call(); err == nil {
                                //fmt.Printf("Program.prepare: %T %v (%v)\n", depend, depend, err)
                                if res == nil {
                                        depends.Append(d)
                                } else if res != nil && res != values.None {
                                        depends.Append(res)
                                }
                        } else {
                                //fmt.Printf("Program.prepare: %T %v (%v)\n", depend, depend, err)
                                break dependLoop
                        }
                case *types.PercentPattern:
                        if stem := entry.Stem(); stem != "" {
                                dent := d.Entry(entry.Stem())
                                name := dent.String()
                                switch prog.module.EntryClass(name) {
                                case types.GeneralRuleEntry:
                                        depend = dent
                                        goto dependSwitch
                                case types.FileRuleEntry:
                                        if p, stem := prog.module.MatchPattern(name); p != nil {
                                                depend = p.Entry(stem)
                                                goto dependSwitch
                                        }
                                        if _, err := os.Stat(name); err == nil {
                                                depends.Append(dent)
                                        } else {
                                                Fail("no file or directory %v", dent)
                                        }
                                default:
                                        Fail("unknown dependency (%v)", dent)
                                }
                        } else {
                                Fail("empty stem (%s, dependency %v)", entry, d)
                        }
                default:
                        if types.IsDummyValue(d) {
                                sym, _ := d.(types.Symbol)
                                scope := sym.Parent()
                                if _, s := scope.LookupAt(token.NoPos, sym.Name()); s != nil {
                                        depend = s
                                        goto dependSwitch
                                }
                                Fail("unknown dependency %s", sym.Name())
                        } else {
                                Fail("unknown dependency (%T)", d)
                        }
                }
        }
        return
}

func (prog *Program) Execute(entry *types.RuleEntry, args []types.Value, forced bool) (result types.Value, err error) {
        defer prog.context.SetScope(prog.context.SetScope(prog.scope))

        //fmt.Printf("Program.Execute: %p %v\n", prog, prog.depends)

        var (
                top = prog.context.Getwd()
                path = prog.module.Path()
                wd, _ = os.Getwd()
                workdir string
        )
        if filepath.IsAbs(path) {
                workdir = path
        } else {
                workdir = filepath.Join(top, path)
        }
        if workdir != wd {
                fmt.Printf("smart: Entering directory '%s'\n", path)
                if err = os.Chdir(path); err == nil {
                        defer func() {
                                fmt.Printf("smart: Leaving directory '%s'\n", path)
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
        
        // TODO: define modifiers in a module, e.g.
        // 
        //      some-modifier : - :
        //              smart statments going here...
        //              
        
        if len(prog.pipline) == 0 {
                // Using the default statements interpreter.
                if i, _ := interpreters[``]; i == nil {
                        err = errors.New("no default dialect")
                        return
                } else if err = prog.interpret(i, out); err != nil {
                        // ...
                }
                return
        }

pipelineLoop:
        for _, v := range prog.pipline {
                switch op := v.(type) {
                case *values.GroupValue:
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
                case *values.BarewordValue:
                        if i, _ := interpreters[op.String()]; i == nil {
                                err = errors.New(fmt.Sprintf("no dialect '%s', required by '%s'", op, entry.Name()))
                                return
                        } else if err = prog.interpret(i, out); err != nil {
                                //fmt.Printf("interpret: %v\n", err)
                                break pipelineLoop
                        /* } else if g, _ := out.Value().(*values.GroupValue); g != nil {
                                if s, c := g.Get(0), g.Get(1); s != nil && c != nil &&
                                        s.String() == "shell" && c.Integer() != 0 {
                                        //fmt.Printf("interpret: %v\n", c)
                                        break pipelineLoop
                                } */
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

func NewProgram(context *Context, scope *types.Scope, depends []types.Value, recipes... types.Value) *Program {
        return &Program{
                context:     context,
                module:      context.CurrentModule(),
                scope:       scope,
                depends:     depends, // *types.RuleEntry, *values.BarefileValue
                recipes:     recipes,
        }
}
