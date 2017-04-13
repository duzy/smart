//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "path/filepath"
        "strings"
        "fmt"
)

// Pattern
type Pattern interface {
        Value
        Entry(stem string) *RuleEntry
        Match(s string) (matched bool, stem string)
}

type pattern struct {
        parent *Scope
        module *Module
        program Program
}

func (p *pattern) Type() Type        { return Invalid }
func (p *pattern) Integer() int64    { return 0 }
func (p *pattern) Float() float64    { return 0 }
func (p *pattern) Program() Program  { return p.program }
func (p *pattern) entry(name, stem string) (entry *RuleEntry) {
        var kind = PatternRuleEntry
        if p.module != nil && p.module.IsFile(name) {
                kind = PatternFileRuleEntry
        }
        entry = NewRuleEntry(kind, name)
        entry.parent = p.parent
        entry.module = p.module
        entry.program = p.program
        entry.stem = stem
        return
}

type PercentPattern struct {
        pattern
        prefix Value
        suffix Value
}

func NewPercentPattern(m *Module, prefix, suffix Value) Pattern {
        return &PercentPattern{pattern:pattern{module:m}, prefix:prefix, suffix:suffix }
}

func (p *PercentPattern) Lit() string { return p.String() }
func (p *PercentPattern) String() (s string) {
        if p.prefix != nil {
                s = p.prefix.String()
        }
        s += "%"
        if p.suffix != nil {
                s += p.suffix.String()
        }
        return
}
func (p *PercentPattern) Match(s string) (matched bool, stem string) {
        /*
        if pp, _ := p.prefix.(*PercentPattern); pp != nil {
        }
        if pp, _ := p.suffix.(*PercentPattern); pp != nil {
        } */
        if prefix := p.prefix.String(); prefix == "" || strings.HasPrefix(s, prefix) {
                if suffix := p.suffix.String(); suffix == "" || strings.HasSuffix(s, suffix) {
                        if a, b := len(prefix), len(s)-len(suffix); a < b {
                                matched, stem = true, s[a:b]
                        }
                }
        }
        return
}

func (p *PercentPattern) Entry(stem string) (entry *RuleEntry) {
        name := p.prefix.String() + stem + p.suffix.String()
        entry = p.entry(name, stem)
        return
}

type RegexpPattern struct {
        pattern
}

func NewRegexpPattern() Pattern {
        return &RegexpPattern{}
}

func (p *RegexpPattern) Lit() string { return p.String() }
func (p *RegexpPattern) String() (s string) { return "" }
func (p *RegexpPattern) Match(s string) (matched bool, stem string) {
        return
}
func (p *RegexpPattern) Entry(stem string) (entry *RuleEntry) {
        return
}

type Program interface {
        Scope() *Scope
        Execute(entry *RuleEntry, args []Value, forced bool) (result Value, err error)
}

type Module struct {
        keyword   token.Token
	path      string
	name      string
        scope     *Scope
        imports   []*Module
        uses      []*Use

        exts      map[string][]string
        files     []string

        // Rule Registry
        dedicated []*RuleEntry
        patterns  []Pattern
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

func (m *Module) AddExts(exts map[string][]string) {
        if m.exts == nil {
                m.exts = make(map[string][]string)
        }
        for k, a := range exts {
                m.exts[k] = append(m.exts[k], a...)
        }
}

func (m *Module) AddFiles(a []string) {
outter:
        for _, s := range a {
                for _, f := range m.files {
                        if s == f { continue outter }
                }
                m.files = append(m.files, s)
        }
}

func (m *Module) IsFile(s string) (v bool) {
        if len(s) > 0 {
                if m.exts != nil {
                        if ext := filepath.Ext(s); ext != "" {
                                if _, v = m.exts[ext[1:]]; v {
                                        return
                                }
                        }
                }
                for _, pat := range m.files {
                        if strings.ContainsAny(pat, "*?[") {
                                v, _ = filepath.Match(pat, s)
                        } else { 
                                v = s == pat
                        }
                        if v { break }
                }
        }
        return
}

func (m *Module) EntryClass(name string) (kind RuleEntryClass) {
        if kind = GeneralRuleEntry; m.IsFile(name) {
                kind = FileRuleEntry
        }
        //fmt.Printf(": %v %v %v\n", name, kind, m.files)
        return
}

func (m *Module) Lookup(s string) (entry *RuleEntry) {
        if sym := m.scope.Lookup(s); sym != nil {
                entry, _ = sym.(*RuleEntry)
        }
        return
}

func (m *Module) AddPercentPattern(p *PercentPattern, prog Program) {
        p.parent = m.scope
        p.module = m
        p.program = prog
        m.patterns = append(m.patterns, p)
}

func (m *Module) MatchPattern(s string) (res Pattern, stem string) {
        var found bool
        for _, p := range m.patterns {
                if found, stem = p.Match(s); found && stem != "" {
                        res = p
                }
        }
        return
}

func (m *Module) FindPercentPattern(s string) (res *PercentPattern) {
        for _, p := range m.patterns {
                if pp, _ := p.(*PercentPattern); pp != nil && pp.String() == s {
                        res = pp
                }
        }
        return
}

func (m *Module) Insert(name string, prog Program) (entry *RuleEntry) {
        if sym := m.scope.Lookup(name); sym == nil {
                entry = NewRuleEntry(m.EntryClass(name), name)
                entry.parent = m.scope
                entry.module = m
                m.scope.Insert(entry)
        } else if entry, _ = sym.(*RuleEntry); entry == nil {
                panic(fmt.Sprintf("name '%v' already taken\n", sym.Name()))
        }
        entry.program = prog
        //entry.pos = pos // overwrite position
        m.dedicated = append(m.dedicated, entry)
        return
}

func (m *Module) GetDefaultEntry() (entry *RuleEntry) {
        if len(m.dedicated) > 0 {
                entry = m.dedicated[0]
        }
        return
}
