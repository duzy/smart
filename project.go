//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
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
func (filemap *FileMap) Match(filename string) bool {
        return globMatch(filemap.Pattern, filename)
}

func (filemap *FileMap) stat(base, name string) (file *File) {
        for _, path := range filemap.Paths {
                var ( dir, sub string ; err error )
                if sub, err = path.Strval(); err != nil { return }
                if filepath.IsAbs(sub) {
                        dir = sub
                        sub = ""
                } else {
                        dir = base //filepath.Join(base, sub)
                }

                // Check file in the filesystem.
                if file = stat(name, sub, dir); file != nil { break }
        }
        return
}

func globMatch(patval Value, filename string) (matched bool) {
        pattern, err := patval.Strval()
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

type enterec struct {
        wd, dir string
        print, silent bool
        num int
}

func (rec *enterec) String() string { return rec.dir }

var cd = &struct{
        stack []*enterec // entered directories
        enters map[string]*enterec // enters
}{
        enters: make(map[string]*enterec),
}

type Project struct {
        keyword  token.Token // project, package, module
        
	absPath string
	relPath string
        tmpPath string
	spec    string
	name    string
        scope   *Scope
        bases   []*Project

        // List order is significant, duplication is acceptable.
        filemap []*FileMap

        usings  *usinglist

        // Rule Registry (orderred)
        userule *RuleEntry // the 'use' rule
        concrete []*RuleEntry
        patterns []*PatternEntry

        filescopes []*Scope

        // TODO: printEntering() ...
        // TODO: printLeaving() ...
}

func (p *Project) String() string { return fmt.Sprintf("<project %s>", p.name) }

func (p *Project) AbsPath() string { return p.absPath }
func (p *Project) RelPath() string { return p.relPath }
func (p *Project) Spec() string { return p.spec }
func (p *Project) Name() string { return p.name }
func (p *Project) Scope() *Scope { return p.scope }
func (p *Project) Bases() []*Project { return p.bases }

func (p *Project) isa(proj *Project) (res bool) {
        for _, base := range p.bases {
                if base == proj { res = true; break }
        }
        return
}

func (p *Project) Chain(bases ...*Project) {
        for _, base := range bases {
                p.bases = append(p.bases, base)
                p.scope.chain = append(p.scope.chain, base.scope)
        }
}

func (p *Project) mapfile(pat Value, paths []Value) {
        // List order is significant, duplication is acceptable.
        p.filemap = append(p.filemap, &FileMap{ pat, paths })
}

func (p *Project) filemaps() (filemaps []*FileMap) {
        filemaps = append(filemaps, p.filemap...)
        for _, base := range p.bases {
                filemaps = append(filemaps, base.filemaps()...)
        }
        return
}

func (p *Project) wildcard(patterns ...Value) (files []*File, err error) {
        var filemaps = p.filemaps()
ForPats:
        for _, pat := range patterns {
                var ( str string; matched, breakAbsRel bool )
                if str, err = pat.Strval(); err != nil { break ForPats }
                // The 'str' value could be GlobPattern or just
                // regular file/path names. PercPattern is not
                // supported yet.
        ForFilemaps:
                for _, fm := range filemaps {
                        if matched = globMatch(fm.Pattern, str); !matched {
                                // Flip glob matching order.
                                if _, yes := pat.(*GlobPattern); !yes {
                                        continue ForFilemaps
                                } else if str, err = fm.Pattern.Strval(); err != nil {
                                        break ForPats
                                } else if matched = globMatch(pat, str); !matched {
                                        continue ForFilemaps
                                } else {
                                        // using the arg glob
                                        breakAbsRel = true
                                }
                        }

                        /*if p.name == "..." {
                                fmt.Printf("wildcard: (%v %T %v) -> %v\n", pat, fm.Pattern, fm.Pattern, str)
                        }*/

                        var names []string

                        // Absolute or relative files are not related to the
                        // paths.
                        if filepath.IsAbs(str) || strings.HasPrefix(str, "./") || strings.HasPrefix(str, "../") {
                                if names, err = filepath.Glob(str); err != nil { break ForPats }
                                for _, s := range names {
                                        file := stat(filepath.Base(s), "", filepath.Dir(s))
                                        assert(file != nil, "`%s` missing", s)
                                        files = append(files, file)
                                }
                                if breakAbsRel {
                                        continue ForPats
                                } else {
                                        continue ForFilemaps
                                }
                        }

                        // Check against paths for non-abs/rel patterns.
                ForPaths:
                        for _, path := range fm.Paths {
                                var sub string
                                if sub, err = path.Strval(); err != nil { break ForPats }

                                subfile := filepath.Join(sub, str)
                                if names, err = filepath.Glob(subfile); err != nil { break ForPats }
                                if len(names) == 0 { continue ForPaths }

                                dir := filepath.Dir(subfile)
                                if !isAbsOrRel(dir) {
                                        // FIXME: using Getwd()?
                                        dir = filepath.Join(p.absPath, dir)
                                }

                                // Chop off path 'sub' prefix to have shorter names
                                // Aka. trim prefix 'file.Sub+PathSep'
                                prefix := strings.TrimSuffix(subfile, str)
                                for _, s := range names {
                                        name := strings.TrimPrefix(s, prefix)
                                        file := stat(name, sub, prefix)
                                        assert(file != nil, "`%s` missing (%s)", s, name)
                                        files = append(files, file)
                                }
                        }
                }
        }
        return
}

func (p *Project) search(name string) (file *File) {
        for _, filemap := range p.filemaps() {
                // Match the represented file name.
                if filemap.Match(name) {
                        if file = filemap.stat(p.absPath, name); file != nil {
                                file.match = filemap
                                if enable_assertions {
                                        assert(file.exists(), "`%s` file not existed", file)
                                }
                                return
                        }
                }
        }

        if file = stat(name, "", p.absPath); file == nil {
                for _, base := range p.bases {
                        file = base.search(name)
                        if file != nil && file.exists() {
                                return
                        }
                }
        }
        return
}

func (p *Project) searchInDir(dir, name string, ignoreMissing bool) (file *File) {
        var isAbs, isRel bool
        if isAbs = filepath.IsAbs(name); isAbs {
                file = stat(name, "", "")
        } else if isRel = isRelPath(name); isRel {
                file = stat(name, "", dir)
        } else if file = p.search(name); file != nil {
                // return
        } else if ignoreMissing {
                // ignore missing
        } else if s := filepath.Dir(name); dir != "." {
                // Check for bare non-system sub-path (e.g. foo/bar/name.xxx)
                file = p.search(filepath.Base(name))
                if file != nil && !strings.HasSuffix(file.dir, s) {
                        file = nil // 
                }
        }
        return
}

func (p *Project) isFileName(s string) (res bool) {
        if len(s) > 0 {
                for _, filemap := range p.filemap {
                        if filemap.Match(s) { return true }
                }
                for _, base := range p.bases {
                        if res = base.isFileName(s); res { break }
                }
        }
        return
}

func (p *Project) file(s string) (file *File) {
        if okay := p.isFileName(s); okay {
                file = p.search(s)
                if file != nil && enable_assertions {
                        assert(file.exists(), "`%s` file not existed", file)
                        assert(file.name == s, "file name differs '%s' != '%s'", file.name, s)
                        //assert(file.match != nil, "`%s` not matched file", s)
                        assert(file.info != nil, "`%v` found nil file info", s)
                        assert(file.dir != "", "`%v` found empty file dir", s)
                }
        }
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
                switch target := rec.target.(type) {
                case *File:
                        if target.name == s { return rec, nil }
                default:
                        var sv string
                        if sv, err = target.Strval(); err != nil { return }
                        if sv == s { return rec, nil }
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
        var entry *RuleEntry
        if entry, err = p.resolveEntry(target); err != nil {
                //if trace_prepare { pc.tracef("%s", err) }
                return
        } else if entry != nil {
                err = pc.update(entry)
                return
        }

        var pss []*StemmedEntry
        if pss, err = p.resolvePatterns(target); err == nil {
                for _, ps := range pss {
                        ps.target = target // Bounds StemmedEntry with the source.
                        if err = ps.prepare(pc); err == nil {
                                return // Updated successfully!
                        } else if _, ok := err.(patternPrepareError); ok {
                                // Discard pattern unfit errors and caller stack.
                                err = nil
                        } else {
                                break // Update failed!
                        }
                }
        }

        if file := p.file(target); file != nil {
                if trace_prepare { pc.tracef("file: %s", file) }
                pc.addTarget(file)
                return
        }

        err = targetNotFoundError{ p, target }
        //if trace_prepare { pc.tracef("%s", err) }
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
        if pat, ok := target.(*PercPattern); ok {
                if pat == nil { /* FIXME: error */ }
                /*for _, pat := range p.patterns {
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
                }*/
                entry = &RuleEntry{
                        class: GlobRuleEntry,
                        target: target,
                }
                p.patterns = append(p.patterns, &PatternEntry{ entry, pat })
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

func enter(prog *Program, dir string) (err error) {
        if trace_entering {
                fmt.Printf("entering: %v (%v)\n", dir, prog.project.name)
        }

        var wd string
        if wd, err = os.Getwd(); err != nil { return }
        if err = os.Chdir(dir); err == nil {
                prog.auto("CWD", &String{dir})
        }

        var ( enter *enterec ; ok bool )
        if enter, ok = cd.enters[dir]; !ok {
                enter = &enterec{ wd:wd, dir:dir }
                cd.enters[dir] = enter
        }
        enter.num += 1
        cd.stack = append([]*enterec{enter}, cd.stack...)
        return
}

func leave(prog *Program, top int) (err error) {
        if trace_entering {
                fmt.Printf("leaving: %v (%v)\n", top, prog.project.name)
        }

        if enable_assertions {
                assert(len(cd.stack) > top, "wrong cd stack")
        }

        var stop = len(cd.stack) - top
        for i, enter := range cd.stack {
                /* if enter.print {
                        fmt.Printf("smart:  Leaving directory '%s'\n", enter.dir)
                        enter.print = false
                } */
                enter.num -= 1
                if i == stop {
                        err = os.Chdir(enter.wd)
                        cd.stack = cd.stack[i:]
                        break
                }
        }
        return
}

func printEnteringDirectory() {
        if l := len(cd.stack); l > 0 {
                var enter = cd.stack[0]
                if enter.silent { return }
                for _, p := range cd.stack {
                        if p.print && p != enter {
                                fmt.Printf("smart:  Leaving directory '%s'\n", p.dir)
                                p.print = false
                        }
                }
                if !enter.print {
                        fmt.Printf("smart: Entering directory '%s'\n", enter.dir)
                        //fmt.Printf("smart: %d %v\n", l, cd.stack)
                        enter.print = true
                }
        }
}

func printLeavingDirectory() {
        if l := len(cd.stack); l > 0 {
                for _, p := range cd.stack {
                        if p.print {
                                fmt.Printf("smart:  Leaving directory '%s'\n", p.dir)
                                p.print = false
                        }
                }
        }
}
