//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package core

import (
        "extbit.io/smart/token"
        "crypto/sha256"
        "path/filepath"
        "strings"
        "errors"
        "bytes"
        //"sort"
        "fmt"
        "os"
)

const strPathSep = string(filepath.Separator)

type HashBytes [sha256.Size]byte

type FileMap struct {
        Pattern string
        Paths []string
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(filename string) (matched bool) {
        list0 := strings.Split(filemap.Pattern, strPathSep)
        list1 := strings.Split(filename, strPathSep)
        if n := len(list0); n == 0 {
                // FIXME: match any?
        } else if m := len(list1); n == m { // foo/*.o  <->  src/foo.o
                for i, pat := range list0 {
                        var s = list1[i]
                        /*if s[0]=='.' && pat[0]=='*' {
                                s = s[1:] // .foo.o|*.o  =>  foo.o|*.o
                        }*/
                        if v, _ := filepath.Match(pat, s); !v {
                                return false
                        }
                }
                matched = true
        } else if n == 1 && m > 1 { // *.o  <->  src/foo.o
                var ( pat = list0[0]; s = list1[m-1] )
                matched, _ = filepath.Match(pat, s)
        }
        return
}

type Project struct {
	absPath string
	relPath string
	spec    string
	name    string
        bases   []*Project
        scope   *Scope
        uses    []*Use

        filemap []FileMap

        // Rule Registry (orderred)
        concrete []*RuleEntry
        patterns []*PatternEntry

        filescopes []*Scope
}

func (p *Project) AbsPath() string { return p.absPath }
func (p *Project) RelPath() string { return p.relPath }
func (p *Project) Spec() string { return p.spec }
func (p *Project) Name() string { return p.name }
func (p *Project) Scope() *Scope { return p.scope }
func (p *Project) Uses() []*Use { return p.uses }
func (p *Project) Bases() []*Project { return p.bases }

func (p *Project) Chain(bases... *Project) {
        for _, base := range bases {
                p.bases = append(p.bases, base)
                p.scope.chain = append(p.scope.chain, base.scope)
        }
}

func (p *Project) MapFile(pat string, paths []string) {
        // List order is significant, duplication is acceptable.
        p.filemap = append(p.filemap, FileMap{ pat, paths })
}

func (p *Project) FileMaps() (filemaps []FileMap) {
        filemaps = append(filemaps, p.filemap...)
        for _, base := range p.bases {
                filemaps = append(filemaps, base.FileMaps()...)
        }
        return
}

func (p *Project) SearchFile(filename string) *File {
        var (
                file = &File{ Name: filename }
                projDir = p.AbsPath()
        )

        ForFiles: for _, filemap := range p.FileMaps() {
                // Match the filename (no base), '*.c' won't match 'src/x.c'.
                if filemap.Match(filename) {
                        file.Match = &filemap
                } else {
                        continue ForFiles
                }

                for _, path := range filemap.Paths {
                        var (
                                dir = path 
                                abs = filepath.IsAbs(dir)
                        )
                        if !abs {
                                dir = filepath.Join(projDir, dir)
                        }

                        // fmt.Printf("match: %v %v %v %v %v\n", dir, path, filename, filemap.Paths, p.FileMaps())

                        if fi, _ := os.Stat(filepath.Join(dir, filename)); fi != nil {
                                if file.Info, file.Dir = fi, dir; !abs {
                                        file.Sub = path 
                                }
                                break ForFiles
                        } else if file.Dir == "" {
                                if file.Dir = dir; !abs {
                                        file.Sub = path
                                }
                        }
                        if false {
                                fmt.Printf("SearchFile: %v: %v (%v)\n", p.Name(), filename, dir)
                        }
                }
        }

        if file.Info == nil && file.Dir == "" {
                if file.Info, _ = os.Stat(filepath.Join(projDir, filename)); file.Info != nil {
                        file.Dir = projDir
                }
        }

        /*SearchBases:*/ if file.Info == nil && file.Dir == "" {
                for _, base := range p.bases {
                        if v := base.SearchFile(filename); v != nil {
                                return v
                        }
                }
        }
        return file
}

func (p *Project) isFile(s string) (v bool) {
        if len(s) > 0 {
                for _, filemap := range p.filemap {
                        if false {
                                fmt.Printf("IsFile: %v %v (%v) (%s)\n", filemap, s, filemap.Match(s), p.name)
                        }
                        if filemap.Match(s) {
                                return true
                        }
                }
                for _, base := range p.bases {
                        if base == p {
                                panic(fmt.Sprintf("recursive base (%v)", p.name))
                        }
                        if v = base.isFile(s); v {
                                return
                        }
                }
        }
        return
}

func (p *Project) ToFile(s string) (file *File) {
        if p.isFile(s) { file = p.SearchFile(s) }
        return
}

func (p *Project) DefaultEntry() (entry *RuleEntry) {
        if len(p.concrete) > 0 {
                entry = p.concrete[0]
        }
        return
}

type PatternStem struct {
        Patent *PatternEntry
        Stem string
        source string // source target matched the pattern
        file *File // source file matched the pattern
}

func (ps *PatternStem) String() (s string) {
        var e error
        if s, e = ps.Patent.Strval(); e == nil {
                s = s + "(" + ps.Stem + ")"
        } else {
                s = fmt.Sprintf("PatternStem{%s}!(%s)", ps, e)
        }
        return
}

func (ps *PatternStem) MakeConcreteEntry() (*RuleEntry, error) {
        return ps.Patent.MakeConcreteEntry(ps.Stem)
}

func (ps *PatternStem) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                if ps.file != nil {
                        fmt.Printf("prepare:PatternStem: %v (%v) (file: %v) (%v -> %v)\n", ps, ps.Patent.class, ps.file, pc.entry.owner.name, pc.entry)
                } else if ps.source != "" {
                        fmt.Printf("prepare:PatternStem: %v (%v) (source: %v) (%v -> %v)\n", ps, ps.Patent.class, ps.source, pc.entry.owner.name, pc.entry)
                } else {
                        fmt.Printf("prepare:PatternStem: %v (%v) (%v -> %v)\n", ps, ps.Patent.class, pc.entry.owner.name, pc.entry)
                }
        }
        
        var (
                stems = []string{ ps.Stem }
                sources = []string{ ps.source }
                entry *RuleEntry
        )
        if ps.file != nil {
                sources = append(sources, ps.file.Name)
        }

        // Find all useful stems.
        ForSources: for _, source := range sources {
                var ( 
                        matched bool
                        stem string
                )
                if source == "" { continue }
                if matched, stem, err = ps.Patent.Pattern.Match(source); matched && stem != "" {
                        for _, s := range stems { if s == stem { continue ForSources } }
                        stems = append(stems, stem)
                }
        }

        // Try preparing target with all stems.
        ForStems: for i, stem := range stems {
                if entry, err = ps.Patent.MakeConcreteEntry(stem); err != nil {
                        return
                }

                var project = pc.program.project
                if pc.program.caller != nil && pc.program.hasCDDash() {
                        project = pc.program.caller.program.project
                }

                // Correct stemmed entry class.
                if entry.class != StemmedFileEntry && pc.entry.class == StemmedFileEntry && ps.file != nil {
                        // TODO: if project.isFile(entry.name)
                        entry.class = StemmedFileEntry     
                }
                
                if entry.class == StemmedFileEntry {
                        if ps.file == nil {
                                var file = project.SearchFile(entry.name)
                                if !file.IsKnown() {
                                        file.Dir = project.AbsPath()
                                }
                                if trace_prepare {
                                        fmt.Printf("prepare:PatternStem: %v ([%d/%d]: %v) (file: %v) (%v)\n", ps, i, len(stems), stem, file, project.name)
                                }
                                ps.file = file
                        }
                }

                if trace_prepare {
                        fmt.Printf("prepare:PatternStem: %v (%v) ([%d/%d]: %v %v) (file: %v) (%v -> %v)\n", ps, entry.class, i, len(stems), entry.Depends(), stem, ps.file, pc.entry.owner.name, pc.entry)
                }

                // Set stem for the current preparation.
                pc.stem, entry.stem, entry.file = stem, stem, ps.file
                if err = entry.prepare(pc); err == nil {
                        break ForStems // Good!
                } else if ute, ok := err.(unknownTargetError); ok {
                        fmt.Printf("prepare:PatternStem: FIXME: unknown target %v (%v)\n", ute.target, pc.entry)
                } else if ufe, ok := err.(unknownFileError); ok {
                        fmt.Printf("prepare:PatternStem: FIXME: unknown file %v (%v)\n", ufe.file, pc.entry)
                }
        }
        return
}

func (p *Project) FindPatterns(s string) (res []*PatternStem, err error) {
        for _, p := range p.patterns {
                var (
                        found bool
                        stem string
                )
                if found, stem, err = p.Pattern.Match(s); err != nil {
                        return
                } else if found && stem != "" {
                        res = append(res, &PatternStem{ p, stem, "", nil })
                }
        }
        for _, base := range p.bases {
                var a []*PatternStem
                if a, err = base.FindPatterns(s); err == nil {
                        res = append(res, a...)
                } else {
                        return
                }
        }
        return
}

// Find rule entry by name or create new one.
func (p *Project) Entry(name string) (entry *RuleEntry, err error) {
        _, obj := p.scope.Find(name)
        if obj != nil {
                if entry, _ = obj.(*RuleEntry); entry != nil {
                        return
                }
        }
        
        // TODO: Improves patter searching on base chain. 
        //fmt.Printf("Project.Entry: %v: %v %v\n", name, p.patterns, p.scope)
        var pss []*PatternStem
        if pss, err = p.FindPatterns(name); err != nil {
                return
        }

        for _, ps := range pss {
                if ps.Patent.programs == nil {
                        continue // FIXME: ???
                }
                //fmt.Printf("%s: %v has %v programs\n", name, ps.Patent.Name(), len(ps.Patent.programs))
                if entry, err = ps.MakeConcreteEntry(); entry != nil || err != nil {
                        return
                }
        }
        return
}

func (p *Project) SetProgram(name string, class RuleEntryClass, prog *Program) (entry *RuleEntry, err error) {
        switch class {
        case GeneralRuleEntry:
        case ExplicitFileEntry:
        case UseRuleEntry:
        default:
                err = errors.New(fmt.Sprintf("Invalid entry class `%v' (%v).\n", class, name))
                return
        }
        
        var alt Object
        if entry, alt = p.scope.InsertEntry(p, class, name); alt != nil {
                if entry, _ = alt.(*RuleEntry); entry == nil {
                        err = errors.New(fmt.Sprintf("Name '%v' already taken as `%T', failed mapping entry.", name, alt))
                }
        }
        if entry != nil && err == nil {
                entry.programs = append(entry.programs, prog)
                p.concrete = append(p.concrete, entry)
        }
        return
}

func (p *Project) SetGlobPatternProgram(pp *GlobPattern, class RuleEntryClass, prog *Program) (patent *PatternEntry, err error) {
        switch class {
        case GeneralRuleEntry: class = GlobRuleEntry
        case ExplicitFileEntry: class = StemmedFileEntry
        default:
                err = errors.New(fmt.Sprintf("Invalid pattern class `%v' (%v).\n", class, p))
                return
        }
        
        // Patterns don't calls p.scope.InsertEntry(...)
        var name string
        if name, err = pp.Strval(); err != nil {
                return
        }
        
        var entry = &RuleEntry{
                object{
                        scope: p.scope,
                        owner: p,
                        name:  name,
                        typ:   RuleEntryType,
                },
                class, // class
                nil,   // file
                nil,   // path
                "",    // stem
                nil,   // caller
                nil,   // closure
                []*Program{ prog },
                token.Position{},
        }
        patent = &PatternEntry{ entry, pp }
        p.patterns = append(p.patterns, patent)
        return
}

func (p *Project) CmdHash(target Value, recipes []string) (k, v HashBytes, err error) {
        var (
                key = sha256.New()
                val = sha256.New()
                str string
        )
        fmt.Fprintf(key, "%s", p.AbsPath())
        if str, err = target.Strval(); err == nil {
                fmt.Fprintf(key, "%s", str)
        } else {
                return
        }
        /* if str, err = depend.Strval(); err == nil {
                fmt.Fprintf(key, "%s", str)
        } else {
                return
        } */
        for _, recipe := range recipes {
                /* if recipe, err = Reveal(recipe); err != nil { return }
                if str, err = recipe.Strval(); err != nil { return }
                fmt.Fprintf(val, "%v", str) */
                fmt.Fprintf(val, "%v", recipe)
        }
        copy(k[:], key.Sum(nil))
        copy(v[:], val.Sum(nil))
        return
}

func (p *Project) hashDir(k []byte) string {
        s := fmt.Sprintf("%x", k[:2])
        return filepath.Join(p.AbsPath(), ".smart", "hash", 
                s[0:1], s[1:2], s[2:3], s[3:])
}

func (p *Project) CheckCmdHash(target Value, recipes []string) (same bool, err error) {
        var (
                k, v HashBytes
                dir = p.hashDir(k[:])
        )
        if k, v, err = p.CmdHash(target, recipes); err != nil {
                return
        }
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

func (p *Project) UpdateCmdHash(target Value, recipes []string) (k, v HashBytes, err error) {
        if k, v, err = p.CmdHash(target, recipes); err != nil {
                return
        }
        dir := p.hashDir(k[:])
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
