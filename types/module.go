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

type Module struct {
        keyword  token.Token
	path     string
	name     string
        scope    *Scope
        imports  []*Module
        uses     []*Use

        // Rule Registry
        patterns []*Pattern
        dedicated []*RuleEntry

	complete bool
}

func (m *Module) Keyword() token.Token { return m.keyword }
func (m *Module) Path() string { return m.path }
func (m *Module) Name() string { return m.name }
func (m *Module) Scope() *Scope { return m.scope }
func (m *Module) Imports() []*Module { return m.imports }
func (m *Module) Uses() []*Use { return m.uses }
func (m *Module) Complete() bool { return m.complete }

func (m *Module) Entry(name string) (entry *RuleEntry) {
        if sym := m.scope.Lookup(name); sym == nil {
                entry = &RuleEntry{symbol{nil, m, name, Invalid, 0, token.NoPos, token.NoPos}, nil}
                m.scope.Insert(entry)
        } else if entry, _ = sym.(*RuleEntry); entry == nil {
                panic(fmt.Sprintf("name '%v' already taken\n", sym.Name()))
        }
        return
}

func (m *Module) Lookup(s string) (entry *RuleEntry) {
        if sym := m.scope.Lookup(s); sym != nil {
                entry, _ = sym.(*RuleEntry)
        }
        return
}

func (m *Module) Insert(entryName string, prog Program) {
        if isPattern(entryName) {
                m.patterns = append(m.patterns, nil)
        } else {
                entry := m.Entry(entryName)
                entry.program = prog
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
