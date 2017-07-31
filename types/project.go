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
        bases     []*Project
        scope     *Scope
        uses      []*Use

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

func (m *Project) Chain(bases... *Project) {
        m.bases = append(m.bases, bases...)
        for _, base := range bases {
                m.scope.chain = append(m.scope.chain, base.scope)
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

func (m *Project) SearchFile(fv *File) *File {
        var (
                ss = filepath.Base(fv.Name)
                firstMatched []string
        )
        files_loop: for pat, paths := range m.files {
                matched := false
                if strings.ContainsAny(pat, "*?[") {
                        matched, _ = filepath.Match(pat, ss)
                } else { 
                        matched = fv.Name == pat
                }

                if !matched { continue }
                if firstMatched == nil {
                        firstMatched = paths
                }
                
                if filepath.IsAbs(fv.Name) {
                        fi, _ := os.Stat(fv.Name)
                        fv.Info, fv.Dir = fi, ""
                        break files_loop
                }

                for _, p := range paths {
                        full := filepath.Join(p, fv.Name)
                        if fi, er := os.Stat(full); fi != nil && er == nil {
                                fv.Info, fv.Dir = fi, p
                                break files_loop
                        }
                }
        }
        if fv.Info == nil && len(firstMatched) > 0 {
                fv.Dir = firstMatched[0]
        } else {
                for _, base := range m.bases {
                        base.SearchFile(fv)
                        if fv.Info != nil || fv.Dir != "" {
                                return fv
                        }
                }
        }
        return fv
}

func (m *Project) IsFile(s string) (v bool) {
        if len(s) > 0 {
                var ss = filepath.Base(s)
                for pat, _ := range m.files {
                        if strings.ContainsAny(pat, "*?[") {
                                v, _ = filepath.Match(pat, ss)
                        } else { 
                                v = s == pat
                        }
                        if v { return }
                }
                for _, base := range m.bases {
                        if v = base.IsFile(s); v {
                                return
                        }
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
                        res = p; return
                }
        }
        for _, base := range m.bases {
                res, stem = base.MatchPattern(s)
                if res != nil && stem != "" {
                        return
                }
        }
        return
}

func (m *Project) FindPercentPattern(s string) (res *PercentPattern) {
        for _, p := range m.patterns {
                if pp, _ := p.(*PercentPattern); pp != nil && pp.String() == s {
                        res = pp; return
                }
        }
        for _, base := range m.bases {
                if res = base.FindPercentPattern(s); res != nil {
                        return
                }
        }
        return
}

func (m *Project) SetProgram(name string, prog Program, class RuleEntryClass) (entry *RuleEntry, err error) {
        var alt Object
        if entry, alt = m.scope.InsertEntry(m, class, name); alt != nil {
                if entry, _ = alt.(*RuleEntry); entry == nil {
                        err = errors.New(fmt.Sprintf("name '%v' already taken (%T)\n", name, alt))
                }
        }
        if entry != nil && err == nil {
                entry.program = prog
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
