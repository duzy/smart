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
        "strings"
        "errors"
        "fmt"
)

// Program (TODO: moving program into `types` package)
type Program struct {
        context *Context
        module  *types.Module
        scope   *types.Scope
        depends []*RuleEntry
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
                        auto.Reset(values.Make(value))
                }
        }
        return
}

func (prog *Program) interpret(i interpreter, out *types.Def) (err error) {
        var value types.Value
        value, err = i.evaluate(prog, prog.recipes...)
        if err == nil && value != nil {
                out.Reset(value)
        }
        return
}

func (prog *Program) modify(g *values.GroupValue, out *types.Def) (err error) {
        // TODO: using rules in a different module to implement modifiers, e.g.
        //       [ foo.check-preprequisites ]
        //       [ foo.baaaar ]
        if f, ok := modifiers[g.Get(0).String()]; ok {
                var value types.Value
                if value, err = f(prog, out.Value(), g.Slice(1)...); err == nil && value !=  nil {
                        out.Reset(value)
                }
        }
        return
}

func (prog *Program) prepare(entry string) (err error) {
        var (
                res types.Value
                depends = values.List()
        )

        prog.auto("...", depends)

        // TODO: using rules in a different module as prerequisites, e.g.
        //       [ c++.compiled-objects ]
        //       [ docker.instance-launched ]
        for _, depend := range prog.depends {
                if res, err = depend.Execute(); err == nil {
                        if res == nil {
                                depends.Append(values.String(depend.Name()))
                        } else if res != nil && res != values.None {
                                //fmt.Printf("%s: %v\n", depend.Name(), res.Lit())
                                depends.Append(res)
                        }
                } else {
                        break
                }
        }
        return
}

func (prog *Program) execute(entry string, forced bool) (result types.Value, err error) {
        defer prog.context.SetScope(prog.context.SetScope(prog.scope))

        // Calculate depends and files.
        if err = prog.prepare(entry); err != nil {
                return
        }

        var (
                _   = prog.auto("@", entry)
                out = prog.auto("-", values.None)
        )
        
        // TODO: define modifiers in a module, e.g.
        // 
        //      some-modifier : - :
        //              smart statments going here...
        //              
        
        if len(prog.pipline) == 0 {
                // Using the default statements interpreter.
                if i, _ := interpreters[``]; i == nil {
                        err = ErrorNoDialect
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
                                        if false {
                                                fmt.Printf("%s\n", p.message)
                                        }
                                        err = nil
                                }
                                break pipelineLoop
                        }
                case *values.BarewordLiteral:
                        if i, _ := interpreters[op.String()]; i == nil {
                                err = ErrorNoDialect
                                return
                        } else if err = prog.interpret(i, out); err != nil {
                                break pipelineLoop
                        }
                default:
                        err = errors.New(fmt.Sprintf("unsupported modifier: %s", v))
                        break pipelineLoop
                }
        }
        result = out.Value()
        return
}

func (prog *Program) SetModifiers(modifiers... types.Value) (err error) {
        prog.pipline = modifiers
        return
}

func NewProgram(context *Context, scope *types.Scope, depends []*RuleEntry, recipes... types.Value) *Program {
        return &Program{
                context:     context,
                module:      context.CurrentModule(),
                scope:       scope,
                depends:     depends,
                recipes:     recipes,
        }
}

// RuleEntry represents a declared rule.
type RuleEntry struct {
        name    string
        program *Program
}

func (entry *RuleEntry) Name() string { return entry.name }
func (entry *RuleEntry) Program() *Program { return entry.program }

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
func (entry *RuleEntry) Execute() (result types.Value, err error) {
        if entry.program == nil {
                return nil, ErrorNilExec
        }
        return entry.program.execute(entry.name, false)
}

// Pattern
type Pattern interface {
}

func isPattern(s string) bool {
        if strings.Contains(s, "%") {
                return true
        }
        return false
}

// Registry 
type Registry struct {
        patterns []*Pattern
        dedicated []*RuleEntry
        m map[string]*RuleEntry
}

func (reg *Registry) Entry(name string) (entry *RuleEntry) {
        if entry, _ = reg.m[name]; entry == nil {
                entry = &RuleEntry{ name: name }
                reg.m[name] = entry
        }
        return
}

func (reg *Registry) Lookup(s string) (entry *RuleEntry) {
        entry, _ = reg.m[s]
        return
}

func (reg *Registry) Insert(entryName string, prog *Program) {
        if isPattern(entryName) {
                reg.patterns = append(reg.patterns, nil)
        } else {
                entry := reg.Entry(entryName)
                entry.program = prog
                reg.dedicated = append(reg.dedicated, entry)
        }
        return
}

func (reg *Registry) GetDefaultEntry() (entry *RuleEntry) {
        if len(reg.dedicated) > 0 {
                entry = reg.dedicated[0]
        }
        return
}

func NewRegistry() *Registry {
        return &Registry{
                m: make(map[string]*RuleEntry),
        }
}
