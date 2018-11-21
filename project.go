//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "crypto/sha256"
        "path/filepath"
        "strings"
        "bytes"
        "fmt"
        "os"
)

const PathSep = string(filepath.Separator)

type HashBytes [sha256.Size]byte

type FileMap struct {
        Pattern Value
        Paths []Value
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(filename string) (matched bool) {
        pattern, err := filemap.Pattern.Strval()
        if err != nil { return false }

        list0 := strings.Split(pattern, PathSep)
        list1 := strings.Split(filename, PathSep)
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
        tmpPath string
	spec    string
	name    string
        scope   *Scope
        bases   []*Project

        // List order is significant, duplication is acceptable.
        filemap []FileMap

        usings  *usinglist

        // Rule Registry (orderred)
        userule *RuleEntry // the 'use' rule
        concrete []*RuleEntry
        patterns []*PatternEntry

        filescopes []*Scope
}

func (p *Project) AbsPath() string { return p.absPath }
func (p *Project) RelPath() string { return p.relPath }
func (p *Project) Spec() string { return p.spec }
func (p *Project) Name() string { return p.name }
func (p *Project) Scope() *Scope { return p.scope }
func (p *Project) Bases() []*Project { return p.bases }

func (p *Project) Chain(bases... *Project) {
        for _, base := range bases {
                p.bases = append(p.bases, base)
                p.scope.chain = append(p.scope.chain, base.scope)
        }
}

func (p *Project) mapfile(pat Value, paths []Value) {
        // List order is significant, duplication is acceptable.
        p.filemap = append(p.filemap, FileMap{ pat, paths })
}

func (p *Project) filemaps() (filemaps []FileMap) {
        filemaps = append(filemaps, p.filemap...)
        for _, base := range p.bases {
                filemaps = append(filemaps, base.filemaps()...)
        }
        return
}

func (p *Project) search(file *File) bool {
        var projDir = p.AbsPath()

        ForFileMaps: for _, filemap := range p.filemaps() {
                // Match the represented file name.
                if filemap.Match(file.Name) {
                        file.Match = &filemap
                } else {
                        continue ForFileMaps
                }
                for _, v := range filemap.Paths {
                        var err error
                        var dir, path string // fullpath
                        if path, err = v.Strval(); err != nil {
                                return false
                        }
                        if filepath.IsAbs(path) {
                                dir = path
                        } else {
                                dir = filepath.Join(projDir, path)
                        }

                        // Check file in the filesystem.
                        fullname := filepath.Join(dir, file.Name)
                        fi, err := os.Stat(fullname)

                        //fmt.Printf("search: %v %v\n", fullname, err)

                        if err == nil && fi != nil {
                                file.Sub, file.Dir, file.Info = path, dir, fi
                                break ForFileMaps
                        } else if file.Dir == "" {
                                file.Sub, file.Dir = path, dir
                        }
                }
        }

        if file.Info == nil && file.Dir == "" {
                if file.Info, _ = os.Stat(filepath.Join(projDir, file.Name)); file.Info != nil {
                        file.Dir = projDir
                }
        }

        if file.Info == nil && file.Dir == "" {
                for _, base := range p.bases {
                        if base.search(file) {
                                return true
                        }
                }
        }

        return file.Info != nil
}

func (p *Project) SearchFile(name string) (file *File) {
        file = &File{ Name: name }
        if !p.search(file) {
            // It's okay if file not found.
            // It may only matched a filemap!
        }
        return
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

func (p *Project) file(s string) (file *File) {
        if p.isFile(s) { file = p.SearchFile(s) }
        return
}

func (p *Project) DefaultEntry() (entry *RuleEntry) {
        if len(p.concrete) > 0 {
                entry = p.concrete[0]
        }
        return
}

func (p *Project) resolveObject(s string) (obj Object, err error) {
        if _, obj = p.scope.Find(s); obj == nil {
                for _, base := range p.bases {
                        obj, err = base.resolveObject(s)
                        if err != nil || obj != nil {
                                break
                        }
                }
        }
        return
}

func (p *Project) resolveEntry(s string) (entry *RuleEntry, err error) {
        for _, rec := range p.concrete {
                var sv string
                if sv, err = rec.Strval(); err != nil {
                        return
                } else if sv == s {
                        entry = rec
                        return
                }
        }
        for _, base := range p.bases {
                entry, err = base.resolveEntry(s)
                if err != nil || entry != nil { break }
        }
        return
}

func (p *Project) resolvePatterns(s string) (res []*StemmedEntry, err error) {
        for _, p := range p.patterns {
                var ( stem string; found bool )
                if found, stem, err = p.Pattern.match(s); err != nil {
                        return
                } else if found && stem != "" {
                        res = append(res, &StemmedEntry{ p, stem, s, nil })
                }
        }
        for _, base := range p.bases {
                var a []*StemmedEntry
                if a, err = base.resolvePatterns(s); err == nil {
                        res = append(res, a...)
                } else {
                        return
                }
        }
        return
}

func (p *Project) updateTarget(pc *preparer, target string) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:Target: %v (project %s)\n", target, p.name)
        }

        var entry *RuleEntry
        if entry, err = p.resolveEntry(target); entry != nil {
                if trace_prepare {
                        fmt.Printf("prepare:Target: %v (found %v) (%v)\n", target, entry, p.name)
                }
                err = pc.update(entry)
                return
        }

        var pss []*StemmedEntry
        if pss, err = p.resolvePatterns(target); err == nil {
                for _, ps := range pss {
                        if trace_prepare {
                                fmt.Printf("prepare:Target: %v (stemmed %v) (%v)\n", target, ps, p.name)
                        }
                        ps.target = target // Bounds StemmedEntry with the source.
                        if err = ps.prepare(pc); err == nil {
                                return // Updated successfully!
                        } else if _, ok := err.(patternPrepareError); ok {
                                if trace_prepare {
                                        fmt.Printf("prepare:Target: %v (error: %s)\n", target, err)
                                }
                                // Discard pattern unfit errors and caller stack.
                                err = nil
                        } else {
                                break // Update failed!
                        }
                }
        }

        err = targetNotFoundError{ target }

        if trace_prepare {
                fmt.Printf("prepare: %v %+v\n", err, pc.program.depends)
        }
        return
}

func (p *Project) entry(special ruleSpecial, target Value, prog *Program) (entry *RuleEntry, err error) {
        defer func() {
                if entry != nil && err == nil {
                        entry.programs = append(entry.programs, prog)
                }
        } ()

        var closured = target.closured()
        var strval string
        if strval, err = target.Strval(); err != nil {
                return
        }

        // The 'use' rule entries.
        if special == ruleSpecialUse && !closured {
                if p.userule == nil {
                        p.userule = &RuleEntry{
                                class: UseRuleEntry,
                                target: target,
                        }
                }
                entry = p.userule
                return
        }

        var name string
        if name, err = target.Strval(); err != nil {
                return
        } else if name == "" {
                err = fmt.Errorf("name '%v' already taken as `%T'", name)
                return
        }

        // Looking for pattern rule entries.
        if glob, ok := target.(*GlobPattern); ok {
                if glob == nil { /* FIXME: error */ }
                for _, pat := range p.patterns {
                        var sv string
                        if closured && pat.RuleEntry.String() == name {
                                entry = pat.RuleEntry; break
                        } else if sv, err = pat.RuleEntry.Strval(); err != nil {
                                return
                        } else if sv == strval {
                                entry = pat.RuleEntry; break
                        }
                }
                if entry == nil {
                        entry = &RuleEntry{
                                class: GlobRuleEntry,
                                target: target,
                        }
                        p.patterns = append(p.patterns, &PatternEntry{ entry, glob })
                }
                return
        }

        // Looking for concrete rule entries.
        for _, rec := range p.concrete {
                var sv string
                if closured && rec.String() == name {
                        entry = rec; break
                } else if sv, err = rec.Strval(); err != nil {
                        return
                } else if sv == strval {
                        entry = rec; break
                }
        }
        if entry == nil {
                entry = &RuleEntry{
                        class: GeneralRuleEntry,
                        target: target,
                }
                p.concrete = append(p.concrete, entry)
        }
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
        h := fmt.Sprintf("%x", k[:2]) // HEX of the first two bytes
        return filepath.Join(p.tmpPath, ".hash", h[0:1], h[1:2], h[2:3], h[3:])
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
