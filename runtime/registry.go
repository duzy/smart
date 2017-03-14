//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "strings"
        "fmt"
)

// Program
type Program struct {
        depends []*RuleEntry
        recipes []types.Value
}

func (prog *Program) execute(entry string, forced bool) (err error) {
        // TODO: execute depends and check outdated
        fmt.Printf("TODO: %v: %v\n", entry, prog.recipes)
        return
}

func NewProgram(depends []*RuleEntry, recipes... types.Value) *Program {
        return &Program{
                depends: depends,
                recipes: recipes,
        }
}

// RuleEntry represents a declared rule.
type RuleEntry struct {
        name    string
        program *Program
}

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
func (entry *RuleEntry) Execute() (err error) {
        if entry.program == nil {
                return ErrorNilExec
        }

        // TODO: checking prerequisites befor executing the program
        
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

func (reg *Registry) Entry(s string) (entry *RuleEntry) {
        if entry, _ = reg.m[s]; entry == nil {
                entry = &RuleEntry{ name: s }
                reg.m[s] = entry
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
