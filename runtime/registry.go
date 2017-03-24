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
        "errors"
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

func (prog *Program) modify(g *values.GroupValue) (result types.Value) {
        name := []string{ g.Get(0).String() } // TODO: Interpreter.evalName
        value := prog.context.Fold(token.NoPos, name, g.Slice(1)...)
        result = values.String(value.String())
        return
}

func (prog *Program) pipe(value types.Value) (result types.Value, err error) {
        var out = prog.auto("-", value)
        for i, v := range prog.pipline {
                switch op := v.(type) {
                case *values.GroupLiteral:
                        //out.Reset(prog.modify(&op.GroupValue))
                        prog.modify(&op.GroupValue)
                case *values.GroupValue:
                        //out.Reset(prog.modify(op))
                        prog.modify(op)
                default:
                        fmt.Printf("todo: %d: %T %v\n", i, op, op)
                }
        }
        result = out.Value()
        return
}

func (prog *Program) execute(entry string, forced bool) (result types.Value, err error) {
        defer prog.context.SetScope(prog.context.SetScope(prog.scope))

        prog.auto("@", entry)

        var (
                res types.Value
                depends = values.List()
                updatedDepends = 0
        )
        for _, depend := range prog.depends {
                if res, err = depend.Execute(); err == nil {
                        if res == nil {
                                depends.Append(values.String(depend.Name()))
                        } else {
                                //fmt.Printf("%s: %v\n", depend.Name(), res.Lit())
                                depends.Append(res)
                        }
                } else {
                        return
                }
        }

        // Calculate depends and files.
        if depends.Len() > 0 {
                var (
                        files = values.List()
                        missing = values.List()
                )
                for _, depend := range depends.Slice(0) {
                        //fmt.Printf("depend: %T %v (from %s)\n", depend, depend, entry)
                retryDepend:
                        switch d := depend.(type) {
                        case *values.GroupValue:
                                //fmt.Printf("group: %v\n", depend)
                                switch k, _ := d.Get(0).(*values.BarewordValue); { 
                                case k == targetRegularKind, k == targetDirectoryKind:
                                        if files.Append(d.Get(1)); files.Len() == 1 {
                                                prog.auto("<", files.Get(0))
                                        }
                                case k == targetMissingKind:
                                        missing.Append(d.Get(1))
                                }
                        case *values.ListValue:
                                if depend = d.Take(0); depend != nil {
                                        goto retryDepend
                                }
                        }
                }
                fmt.Printf("%s: %v; %v\n", entry, files, missing)
                if x := missing.Len(); x > 0 {
                        var msg string
                        if x == 1 {
                                msg = fmt.Sprintf("missing file %v", missing)
                        } else {
                                msg = fmt.Sprintf("missing files %v", missing)
                        }
                        msg += fmt.Sprintf(", required by %s", entry)
                        err = errors.New(msg)
                        return
                }
                if files.Len() > 0 {
                        prog.auto("^", files)
                }
        }
        
        if updatedDepends == 0 {
                // TODO: check depends for outdated
        }

        //fmt.Printf("target: %v\n", entry)
        
        var mode = prog.interpreter.mode()
        if mode&interpretMulti != 0 {
                if res, err = prog.interpreter.evaluate(prog.recipes...); err != nil {
                        return
                } else if res != nil {
                        result, err = prog.pipe(res)
                }
        } else if mode&interpretSingle != 0 {
                var list = values.List()
                for _, recipe := range prog.recipes {
                        if res, err = prog.interpreter.evaluate(recipe); err != nil {
                                return
                        } else if res != nil {
                                if res, err = prog.pipe(res); err == nil {
                                        list.Append(res)
                                }
                        }
                }
                result = list
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
