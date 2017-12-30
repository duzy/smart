//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "crypto/sha256"
        "path/filepath"
        "strings"
        "errors"
        "bytes"
        "fmt"
        "os"
)

type HashBytes [sha256.Size]byte

type Program interface {
        Execute(context *Scope, entry *RuleEntry, args []Value) (result Value, err error)
        Params() []string // parameter names
        Project() *Project
        Position() token.Position
        Depends() []Value
        Recipes() []Value
        Pipeline() []Value
        Scope() *Scope
}

type Project struct {
	absPath string
	relPath string
	spec    string
	name    string
        bases   []*Project
        scope   *Scope
        uses    []*Use

        filemap   map[string][]string

        // Rule Registry (orderred)
        concrete []*RuleEntry
        patterns []*PatternEntry

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

func (m *Project) MapFiles(files map[string][]string) {
        if m.filemap == nil {
                m.filemap = make(map[string][]string)
        }
        for k, a := range files {
                m.filemap[k] = append(m.filemap[k], a...)
        }
}

func (m *Project) combineFileMap(filemap map[string][]string) map[string][]string {
        for _, base := range m.bases {
                base.combineFileMap(filemap)
        }
        for pat, paths := range m.filemap {
                filemap[pat] = paths
        }
        return filemap
}

func (m *Project) FileMap() map[string][]string {
        return m.combineFileMap(make(map[string][]string))
}

func (m *Project) SearchFile(filename string) *File {
        var (
                firstMatched []string
                info os.FileInfo
                dir string
        )

        if filepath.IsAbs(filename) {
                info, _ = os.Stat(filename)
                goto SearchBases
        }

        ForFiles: for pat, paths := range m.FileMap() {
                // Match the filename (not base). Note that '*.c' won't match 'src/x.c'.
                if v, _ := filepath.Match(pat, filepath.Base(filename)); !v && pat != filename {
                        continue
                }

                if firstMatched == nil {
                        firstMatched = paths
                }

                for _, path := range paths {
                        var ( p = path )
                        if !filepath.IsAbs(p) {
                                p = filepath.Join(m.AbsPath(), p)
                        }

                        if fi, _ := os.Stat(filepath.Join(p, filename)); fi != nil {
                                info, dir = fi, p
                                break ForFiles
                        } else if false {
                                fmt.Printf("SearchFile: %v: %v\n", m.Name(), filename)
                        }
                }
        }

        SearchBases: var file = &File{
                Name: filename,
                Info: info,
                Dir: dir,
        }
        if file.Info == nil {
                if len(firstMatched) > 0 {
                        file.Dir = firstMatched[0]
                } else {
                        for _, base := range m.bases {
                                if v := base.SearchFile(filename); v != nil {
                                        return v
                                }
                        }
                }
        }
        return file
}

func (m *Project) IsFile(s string) (v bool) {
        if len(s) > 0 {
                var ss = filepath.Base(s)
                for pat, _ := range m.filemap {
                        if false {
                                fmt.Printf("IsFile: %s %v\n", s, pat)
                        }
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

func (ps *PatternStem) String() string {
        return "<" + ps.Patent.Strval() + "~" + ps.Stem + ">"
}

func (ps *PatternStem) MakeConcreteEntry() (*RuleEntry, error) {
        return ps.Patent.MakeConcreteEntry(ps.Stem)
}

func (ps *PatternStem) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:PatternStem: %v\n", ps)
        }
        if entry, e := ps.MakeConcreteEntry(); e == nil {
                err = pc.Prepare(entry)
        } else {
                err = e
        }
        return
}

func (m *Project) FindPatterns(s string) (res []*PatternStem) {
        for _, p := range m.patterns {
                if found, stem := p.Pattern.Match(s); found && stem != "" {
                        res = append(res, &PatternStem{ p, stem })
                }
        }
        for _, base := range m.bases {
                res = append(res, base.FindPatterns(s)...)
        }
        //fmt.Printf("%s: %s: %v\n", s, m.Name(), res)
        return
}

// Find rule entry by name or create new one.
func (m *Project) Entry(name string) (entry *RuleEntry, err error) {
        _, obj := m.scope.Find(name)
        if obj != nil {
                if entry, _ = obj.(*RuleEntry); entry != nil {
                        return
                }
        }
        
        // TODO: Improves patter searching on base chain. 
        //fmt.Printf("Project.Entry: %v: %v %v\n", name, m.patterns, m.scope)
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
        case ExplicitFileEntry:
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
        //fmt.Printf("SetPercentPatternProgram: %v %v -> %v\n", p, class, prog.Depends())
        
        switch class {
        case GeneralRuleEntry: class = PatternRuleEntry
        case ExplicitFileEntry: class = StemmedFileEntry
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

func (m *Project) CmdHash(target Value, recipes []Value) (k, v HashBytes) {
        var (
                key = sha256.New()
                val = sha256.New()
        )
        fmt.Fprintf(key, "%s", m.AbsPath())
        fmt.Fprintf(key, "%s", target.Strval())
        //fmt.Fprintf(key, "%s", depend.Strval())
        for _, recipe := range recipes {
                fmt.Fprintf(val, "%v", Reveal(recipe).Strval())
        }
        copy(k[:], key.Sum(nil))
        copy(v[:], val.Sum(nil))
        return
}

func (m *Project) hashDir(k []byte) string {
        s := fmt.Sprintf("%x", k[:2])
        return filepath.Join(m.AbsPath(), ".smart", "hash", 
                s[0:1], s[1:2], s[2:3], s[3:])
}

func (m *Project) CheckCmdHash(target Value, recipes []Value) (same bool, err error) {
        var (
                k, v = m.CmdHash(target, recipes)
                dir = m.hashDir(k[:])
        )
        if f, e := os.Open(filepath.Join(dir, fmt.Sprintf("%x", k))); e == nil {
                var h []byte
                if n, e := fmt.Fscanf(f, "%x", &h); e != nil {
                        err = e; return
                } else if n == 1 {
                        same = bytes.Equal(v[:], h)
                        //fmt.Printf("CheckCmdHash: %x -> %x (%x)\n", k, v, h)
                }
                err = f.Close()
        } else {
                err = e
        }
        return
}

func (m *Project) UpdateCmdHash(target Value, recipes []Value) (k, v HashBytes, err error) {
        k, v = m.CmdHash(target, recipes)
        dir := m.hashDir(k[:])
        if err = os.MkdirAll(dir, 0700); err != nil {
                return
        }
        if f, e := os.Create(filepath.Join(dir, fmt.Sprintf("%x", k))); e == nil {
                //fmt.Printf("UpdateCmdHash: %x -> %x (%s)\n", k, v, target.Strval())
                fmt.Fprintf(f, "%x", v)
                err = f.Close()
        } else {
                err = e
        }
        return
}
