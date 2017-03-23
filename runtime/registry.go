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
        pipline []types.Value
}

func (prog *Program) Scope() *types.Scope { return prog.scope }

func (prog *Program) defineAuto(name string, value interface{}) (auto *types.Auto) {
        auto = types.NewAuto(prog.module, name, values.Make(value))
        prog.scope.Insert(auto)
        return
}

func (prog *Program) modify(g *values.GroupValue) (result types.Value) {
        name := []string{ g.Get(0).String() } // TODO: Interpreter.evalName
        value := prog.context.Fold(token.NoPos, name, g.Slice(1)...)
        result = values.String(value.String())
        return
}

func (prog *Program) pipe(value types.Value) (result types.Value, err error) {
        for i, v := range prog.pipline {
                switch op := v.(type) {
                case *values.GroupLiteral:
                        result = prog.modify(&op.GroupValue)
                case *values.GroupValue:
                        result = prog.modify(op)
                default:
                        fmt.Printf("todo: %d: %T %v\n", i, op, op)
                }
        }
        return
}

func (prog *Program) execute(entry string, forced bool) (result types.Value, err error) {
        defer prog.context.SetScope(prog.context.SetScope(prog.scope))

        prog.defineAuto("@", entry)

        var (
                res types.Value
                depends = values.List()
        )
        for _, depend := range prog.depends {
                if res, err = depend.Execute(); err == nil {
                        if res == nil {
                                depends.Append(values.String(depend.Name()))
                        } else {
                                depends.Append(res)
                        }
                } else {
                        return
                }
        }

        if depends.Len() > 0 {
                prog.defineAuto("<", depends.Get(0))
                prog.defineAuto("^", depends)
        }
        
        // TODO: execute depends and check outdated

        var (
                mode = prog.interpreter.mode()
                pipe = len(prog.pipline) > 0
        )
        if mode&interpretMulti != 0 {
                if result, err = prog.interpreter.evaluate(prog.recipes...); err != nil {
                        return
                } else if result != nil {
                        if pipe {
                                result, err = prog.pipe(result)
                        } else {
                                fmt.Printf("%v\n", result)
                        }
                }
        } else if mode&interpretSingle != 0 {
                var results = values.List()
                for _, recipe := range prog.recipes {
                        if result, err = prog.interpreter.evaluate(recipe); err != nil {
                                return
                        } else if result != nil {
                                if len(prog.pipline) > 0 {
                                        results.Append(result)
                                } else {
                                        fmt.Printf("%v", result)
                                }
                        }
                }
                if pipe {
                        result, err = prog.pipe(results)
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
