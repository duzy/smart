//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "strings"
        "fmt"
)

// Program
type Program struct {
        context *Context
        module  *types.Module
        scope   *types.Scope
        depends []*RuleEntry
        recipes []types.Value
        interpreter interpreter // TODO: global (sharing states) or instance interpreter
}

func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) defineAuto(name string, value interface{}) (auto *types.Auto) {
        auto = types.NewAuto(prog.module, name, values.Make(value))
        prog.scope.Insert(auto)
        return
}

func (prog *Program) execute(entry string, forced bool) (err error) {
        defer prog.context.SetScope(prog.context.SetScope(prog.scope))

        prog.defineAuto("@", entry)
        
        for _, depend := range prog.depends {
                if err = depend.Execute(); err != nil {
                        return
                }
        }
        
        // TODO: execute depends and check outdated

        var (
                mode = prog.interpreter.mode()
                result types.Value
        )
        if mode&interpretMulti != 0 {
                if result, err = prog.interpreter.evaluate(prog.recipes...); err != nil {
                        return
                } else if result != nil {
                        fmt.Printf("%v\n", result)
                }
        } else if mode&interpretSingle != 0 {
                for _, recipe := range prog.recipes {
                        if result, err = prog.interpreter.evaluate(recipe); err != nil {
                                return
                        } else if result != nil {
                                fmt.Printf("%v", result)
                        }
                }
        }
        return
}

func (prog *Program) InitDialect(name string, modifiers... types.Value) (err error) {
        switch name {
        case "plain": 
                prog.interpreter = &dialectPlain{
                }
        case "shell":
                prog.interpreter = &dialectShell{
                        interpreter: defaultShellInterpreter, // "sh"
                        xopt: "-c",
                }
        case "python":
                prog.interpreter = &dialectShell{
                        interpreter: "python",
                        xopt: "-c",
                }
        case "perl":
                prog.interpreter = &dialectShell{
                        interpreter: "perl",
                        xopt: "-e",
                }
        case "xml":
                prog.interpreter = &dialectXml{
                        whitespace: false,
                }
        case "json":
                prog.interpreter = &dialectJson{
                }
        default:
                err = ErrorNoDialect
        }
        return
}

func NewProgram(context *Context, scope *types.Scope, depends []*RuleEntry, recipes... types.Value) *Program {
        return &Program{
                context:     context,
                module:      context.CurrentModule(),
                scope:       scope,
                depends:     depends,
                recipes:     recipes,
                interpreter: trivialDialect,
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
func (entry *RuleEntry) Execute() (err error) {
        if entry.program == nil {
                return ErrorNilExec
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
