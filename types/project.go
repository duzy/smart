//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        //"github.com/duzy/smart/token"
        "path/filepath"
        "strings"
        "errors"
        "fmt"
        "os"
)

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
        files     map[string][]string

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

func (m *Project) AddFiles(files map[string][]string) {
        if m.files == nil {
                m.files = make(map[string][]string)
        }
        for k, a := range files {
                m.files[k] = append(m.files[k], a...)
        }
}

func (m *Project) SearchFile(s string) (full string) {
        if len(s) > 0 {
                /*
                if m.exts != nil {
                        if ext := filepath.Ext(s); ext != "" {
                                if _, v = m.exts[ext[1:]]; v {
                                        return
                                }
                        }
                } */
                var ss = filepath.Base(s)
                for pat, paths := range m.files {
                        matched := false
                        if strings.ContainsAny(pat, "*?[") {
                                matched, _ = filepath.Match(pat, ss)
                        } else { 
                                matched = s == pat
                        }
                        fmt.Printf("search: %v (%v)\n", s, pat)
                        if matched {
                                if filepath.IsAbs(s) {
                                        full = s; return
                                }
                                for _, p := range paths {
                                        p = filepath.Join(p, s)
                                        if fi, err := os.Stat(p); err == nil && fi != nil {
                                                full = p; return
                                        }
                                }
                        }
                }
        }
        return
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
                for pat, _ := range m.files {
                        if strings.ContainsAny(pat, "*?[") {
                                v, _ = filepath.Match(pat, ss)
                        } else { 
                                v = s == pat
                        }
                        if v { return }
                }
        }
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

func (m *Project) Insert(name string, prog Program, class RuleEntryClass) (entry *RuleEntry, err error) {
        var alt Object
        if entry, alt = m.scope.InsertNewRuleEntry(m, class, name); alt != nil {
                if entry, _ = alt.(*RuleEntry); entry == nil {
                        err = errors.New(fmt.Sprintf("name '%v' already taken (%T)\n", name, alt))
                }
        }
        if entry != nil && err == nil {
                entry.program = prog
                //entry.pos = pos // overwrite position
                m.dedicated = append(m.dedicated, entry)
        }
        return
}

func (m *Project) GetDefaultEntry() (entry *RuleEntry) {
        if len(m.dedicated) > 0 {
                entry = m.dedicated[0]
        }
        return
}
