//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "strings"
        "fmt"
)

// Pattern
type Pattern interface {
}

func isPattern(s string) bool {
        if strings.Contains(s, "%") {
                return true
        }
        return false
}

type Program interface {
        Scope() *Scope
        Execute(entry *RuleEntry, args []Value, forced bool) (result Value, err error)
}

type Module struct {
        keyword  token.Token
	path     string
	name     string
        scope    *Scope
        imports  []*Module
        uses     []*Use

        // Rule Registry
        dedicated []*RuleEntry
        patterns []*Pattern
}

func (m *Module) Keyword() token.Token { return m.keyword }
func (m *Module) Path() string { return m.path }
func (m *Module) Name() string { return m.name }
func (m *Module) Scope() *Scope { return m.scope }
func (m *Module) Imports() []*Module { return m.imports }
func (m *Module) Uses() []*Use { return m.uses }

func (m *Module) AddImport(o *Module) {
        m.imports = append(m.imports, o)
}

func (m *Module) FindImport(name string) (res *Module) {
        for _, m := range m.imports {
                if m.Name() == name {
                        res = m; break
                }
        }
        return
}

func (m *Module) Lookup(s string) (entry *RuleEntry) {
        if sym := m.scope.Lookup(s); sym != nil {
                entry, _ = sym.(*RuleEntry)
        }
        return
}

func (m *Module) Insert(kind RuleEntryClass, name string, prog Program) (entry *RuleEntry) {
        if isPattern(name) {
                m.patterns = append(m.patterns, nil)
        } else {
                if sym := m.scope.Lookup(name); sym == nil {
                        //entry = &RuleEntry{symbol{m.scope, m, name, RuleEntryType, 0, pos, token.NoPos}, nil}
                        entry = NewRuleEntry(kind, name)
                        entry.parent = m.scope
                        entry.module = m
                        m.scope.Insert(entry)
                } else if entry, _ = sym.(*RuleEntry); entry == nil {
                        panic(fmt.Sprintf("name '%v' already taken\n", sym.Name()))
                }
                entry.program = prog
                //entry.pos = pos // overwrite position
                m.dedicated = append(m.dedicated, entry)
        }
        return
}

func (m *Module) GetDefaultEntry() (entry *RuleEntry) {
        if len(m.dedicated) > 0 {
                entry = m.dedicated[0]
        }
        return
}
