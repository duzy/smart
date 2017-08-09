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
        Execute(context *Scope, entry *RuleEntry, args []Value, forced bool) (result Value, err error)
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

        // Rule Registry (orderred)
        concrete []*RuleEntry
        patterns  []*PatternEntry

        filescopes []*Scope
}

func (m *Project) AbsPath() string { return m.absPath }
func (m *Project) RelPath() string { return m.relPath }
func (m *Project) Spec() string { return m.spec }
func (m *Project) Name() string { return m.name }
func (m *Project) Scope() *Scope { return m.scope }
func (m *Project) Uses() []*Use { return m.uses }
func (m *Project) Bases() []*Project { return m.bases }

func (m *Project) Chain(bases... *Project) {
        for _, base := range bases {
                m.bases = append(m.bases, base)
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

func (m *Project) SearchFile(context *Scope, fv *File) *File {
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
                        base.SearchFile(context, fv)
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
                                //fmt.Printf("IsFileName: %s %v\n", s, pat)
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

func (m *Project) DefaultEntry() (entry *RuleEntry) {
        if len(m.concrete) > 0 {
                entry = m.concrete[0]
        }
        return
}

type PatternStem struct {
        Patent *PatternEntry
        Stem string
}

func (ps *PatternStem) MakeConcreteEntry() (*RuleEntry, error) {
        return ps.Patent.MakeConcreteEntry(ps.Stem)
}

func (m *Project) FindPatterns(s string) (res []*PatternStem) {
        //fmt.Printf("FindPatterns: %v (%v %v %v)\n", s, m.Name(), m.patterns, m.bases)
        for _, p := range m.patterns {
                if found, stem := p.Pattern.Match(s); found && stem != "" {
                        res = append(res, &PatternStem{ p, stem })
                }
        }
        for _, base := range m.bases {
                res = append(res, base.FindPatterns(s)...)
        }
        return
}

// Find rule entry by name or create new one.
func (m *Project) Entry(name string) (entry *RuleEntry, err error) {
        obj := m.scope.Find(name)
        if obj != nil {
                if entry, _ = obj.(*RuleEntry); entry != nil {
                        return
                }
        }
        
        // TODO: Improves patter searching on base chain. 
        if pss := m.FindPatterns(name); pss != nil {
                for _, ps := range pss {
                        if ps.Patent.programs == nil {
                                continue // FIXME: ???
                        }
                        //fmt.Printf("%s: %v has %v programs\n", name, ps.Patent.Name(), len(ps.Patent.programs))
                        if entry, err = ps.MakeConcreteEntry(); entry != nil || err != nil {
                                return
                        }
                }
        }
        return
}

func (m *Project) SetProgram(name string, class RuleEntryClass, prog Program) (entry *RuleEntry, err error) {
        switch class {
        case GeneralRuleEntry:
        case FileRuleEntry:
        case UseRuleEntry:
        default:
                err = errors.New(fmt.Sprintf("Invalid entry class `%v' (%v).\n", class, name))
                return
        }
        
        var alt Object
        if entry, alt = m.scope.InsertEntry(m, class, name); alt != nil {
                if entry, _ = alt.(*RuleEntry); entry == nil {
                        err = errors.New(fmt.Sprintf("Name '%v' already taken as `%T', failed mapping entry.", name, alt))
                }
        }
        if entry != nil && err == nil {
                entry.programs = append(entry.programs, prog)
                m.concrete = append(m.concrete, entry)
        }
        return
}

func (m *Project) SetPercentPatternProgram(p *PercentPattern, class RuleEntryClass, prog Program) (patent *PatternEntry, err error) {
        switch class {
        case GeneralRuleEntry: class = PatternRuleEntry
        case FileRuleEntry: class = PatternFileRuleEntry
        default:
                err = errors.New(fmt.Sprintf("Invalid pattern class `%v' (%v).\n", class, p))
                return
        }
        
        var (
                entry *RuleEntry
                alt Object
        )
        if entry, alt = m.scope.InsertEntry(m, class, p.Strval()); alt != nil {
                if entry, _ = alt.(*RuleEntry); entry == nil {
                        err = errors.New(fmt.Sprintf("Pattern '%v' already taken as `%T', failed mapping entry.", p, alt))
                }
        }
        if entry != nil && err == nil {
                entry.class = class
                entry.programs = append(entry.programs, prog)
                patent = &PatternEntry{ entry, p }
                m.patterns = append(m.patterns, patent)
        }
        return
}
