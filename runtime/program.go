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
        //"strings"
        "errors"
        "fmt"
)

// Program (TODO: moving program into `types` package)
type Program struct {
        context *Context
        module  *types.Module
        scope   *types.Scope
        depends []*types.RuleEntry
        recipes []types.Value
        pipline []types.Value
}

func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) auto(name string, value interface{}) (auto *types.Def) {
        if sym := prog.scope.Lookup(name); sym == nil {
                auto = types.NewAuto(prog.module, name, values.Make(value))
                prog.scope.Insert(auto)
        } else {
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
                var value types.Value
                if value, err = f(prog, out.Value(), g.Slice(1)...); err == nil && value !=  nil {
                        out.Set(value)
                }
        } else if i, _ := interpreters[name]; i != nil {
                err = prog.interpret(i, out, g.Slice(1)...)
        } else {
                err = errors.New(fmt.Sprintf("no modifier or dialect '%s'", name))
        }
        return
}

func (prog *Program) prepare(/*entry*/ *types.RuleEntry) (err error) {
        var (
                res types.Value
                depends = values.List()
        )

        prog.auto("...", depends)

        // TODO: using rules in a different module as prerequisites, e.g.
        //       [ c++.compiled-objects ]
        //       [ docker.instance-launched ]
        for _, depend := range prog.depends {
                //fmt.Printf("Program.prepare: %T %v (%v)\n", depend, depend, depend.Pos())
                if res, err = depend.Call(); err == nil {
                        //fmt.Printf("Program.prepare: %T %v (%v)\n", depend, depend, err)
                        if res == nil {
                                //depends.Append(values.String(depend.Name()))
                                depends.Append(depend)
                        } else if res != nil && res != values.None {
                                //fmt.Printf("%s: %v\n", depend.Name(), res.Lit())
                                depends.Append(res)
                        }
                } else {
                        //fmt.Printf("Program.prepare: %T %v\n", depend, depend)
                        break
                }
        }
        return
}

func (prog *Program) Execute(entry *types.RuleEntry, args []types.Value, forced bool) (result types.Value, err error) {
        defer prog.context.SetScope(prog.context.SetScope(prog.scope))
        
        // Calculate depends and files.
        if err = prog.prepare(entry); err != nil {
                return
        }

        var (
                _   = prog.auto("@", entry)
                out = prog.auto("-", values.None)
        )
        defer func() { result = out.Value() }()
        
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
                case *values.GroupLiteral:
                        if err = prog.modify(&op.GroupValue, out); err != nil {
                                if p, ok := err.(*breaker); ok {
                                        fmt.Printf("%s, required by '%s'\n", p.message, entry.Name())
                                        if p.okay {
                                                err = nil
                                        }
                                }
                                break pipelineLoop
                        }
                case *values.BarewordLiteral:
                        if i, _ := interpreters[op.String()]; i == nil {
                                err = errors.New(fmt.Sprintf("no dialect '%s', required by '%s'", op, entry.Name()))
                                return
                        } else if err = prog.interpret(i, out); err != nil {
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

func NewProgram(context *Context, scope *types.Scope, depends []*types.RuleEntry, recipes... types.Value) *Program {
        return &Program{
                context:     context,
                module:      context.CurrentModule(),
                scope:       scope,
                depends:     depends,
                recipes:     recipes,
        }
}
