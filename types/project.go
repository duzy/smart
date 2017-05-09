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
        project *Project
        program Program
}

func (p *pattern) Type() Type        { return Invalid }
func (p *pattern) Integer() int64    { return 0 }
func (p *pattern) Float() float64    { return 0 }
func (p *pattern) Program() Program  { return p.program }
func (p *pattern) entry(name, stem string) (entry *RuleEntry) {
        var kind = PatternRuleEntry
        if p.project != nil && p.project.IsFile(name) {
                kind = PatternFileRuleEntry
        }
        entry = p.parent.NewRuleEntry(p.project, kind, name)
        entry.parent = p.parent
        entry.project = p.project
        entry.program = p.program
        entry.stem = stem
        return
}

type PercentPattern struct {
        pattern
        prefix Value
        suffix Value
}

func NewPercentPattern(m *Project, prefix, suffix Value) Pattern {
        return &PercentPattern{pattern:pattern{project:m}, prefix:prefix, suffix:suffix }
}

func (p *PercentPattern) Lit() string { return p.String() }
func (p *PercentPattern) Pos() *token.Position { return nil }
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
func (p *RegexpPattern) Pos() *token.Position { return nil }
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

type Project struct {
	absPath   string
	relPath   string
	spec      string
	name      string
        scope     *Scope
        uses      []*Use

        exts      map[string][]string
        files     []string

        // Rule Registry
        dedicated []*RuleEntry
        patterns  []Pattern
}

func (m *Project) AbsPath() string { return m.absPath }
func (m *Project) RelPath() string { return m.relPath }
func (m *Project) Spec() string { return m.spec }
func (m *Project) Name() string { return m.name }
func (m *Project) Scope() *Scope { return m.scope }
func (m *Project) Uses() []*Use { return m.uses }

func (m *Project) AddExts(exts map[string][]string) {
        if m.exts == nil {
                m.exts = make(map[string][]string)
        }
        for k, a := range exts {
                m.exts[k] = append(m.exts[k], a...)
        }
}

func (m *Project) AddFiles(a []string) {
outter:
        for _, s := range a {
                for _, f := range m.files {
                        if s == f { continue outter }
                }
                m.files = append(m.files, s)
        }
}

func (m *Project) IsFile(s string) (v bool) {
        if len(s) > 0 {
                if m.exts != nil {
                        if ext := filepath.Ext(s); ext != "" {
                                if _, v = m.exts[ext[1:]]; v {
                                        return
                                }
                        }
                }
                var ss = filepath.Base(s)
                for _, pat := range m.files {
                        if strings.ContainsAny(pat, "*?[") {
                                v, _ = filepath.Match(pat, ss)
                                //fmt.Printf("IsFile: %v, %v, %v\n", s, pat, v)
                        } else { 
                                v = s == pat
                        }
                        if v { break }
                }
        }
        return
}

func (m *Project) EntryClass(name string) (kind RuleEntryClass) {
        if kind = GeneralRuleEntry; m.IsFile(name) {
                kind = FileRuleEntry
        }
        //fmt.Printf(": %v %v %v\n", name, kind, m.files)
        return
}

func (m *Project) AddPercentPattern(p *PercentPattern, prog Program) {
        p.parent = m.scope
        p.project = m
        p.program = prog
        m.patterns = append(m.patterns, p)
}

func (m *Project) MatchPattern(s string) (res Pattern, stem string) {
        var found bool
        for _, p := range m.patterns {
                if found, stem = p.Match(s); found && stem != "" {
                        res = p
                }
        }
        return
}

func (m *Project) FindPercentPattern(s string) (res *PercentPattern) {
        for _, p := range m.patterns {
                if pp, _ := p.(*PercentPattern); pp != nil && pp.String() == s {
                        res = pp
                }
        }
        return
}

func (m *Project) Insert(name string, prog Program) (entry *RuleEntry) {
        var alt Object
        if entry, alt = m.scope.InsertNewRuleEntry(m, m.EntryClass(name), name); alt != nil {
                if entry, _ = alt.(*RuleEntry); entry == nil {
                        panic(fmt.Sprintf("name '%v' already taken\n", name))
                }
        }
        entry.program = prog
        //entry.pos = pos // overwrite position
        m.dedicated = append(m.dedicated, entry)
        return
}

func (m *Project) GetDefaultEntry() (entry *RuleEntry) {
        if len(m.dedicated) > 0 {
                entry = m.dedicated[0]
        }
        return
}
