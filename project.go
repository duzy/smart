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

func (filemap *FileMap) statFile(base string, file *File) (found bool) {
        for _, sub := range filemap.Paths {
                var err error
                var dir, path string // fullpath
                if path, err = sub.Strval(); err != nil { return }

                if filepath.IsAbs(path) {
                        dir = path
                } else {
                        dir = filepath.Join(base, path)
                }

                // Check file in the filesystem.
                info, err := os.Stat(filepath.Join(dir, file.Name))
                if err == nil && info != nil {
                        file.Sub, file.Dir, file.Info = sub, dir, info
                        found = true; break
                } else {
                        if file.Dir == "" { file.Dir = dir }
                        if file.Sub == nil { file.Sub = sub }
                }
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
                                        files = append(files, &File{
                                                Name: s,
                                                Dir: filepath.Dir(s),
                                        })
                                }
                                if breakAbsRel {
                                        continue ForPats
                                } else {
                                        continue ForFilemaps
                                }
                        }

                        // Check against paths for non-abs/rel patterns.
                ForPaths:
                        for _, sub := range fm.Paths {
                                var s string
                                if s, err = sub.Strval(); err != nil { break ForPats }

                                subfile := filepath.Join(s, str)
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
                                        files = append(files, &File{
                                                Name: strings.TrimPrefix(s, prefix),
                                                Sub: sub,
                                                Dir: dir,
                                        })
                                }
                        }
                }
        }
        return
}

func (p *Project) search(file *File) (res bool) {
        for _, filemap := range p.filemaps() {
                // Match the represented file name.
                if filemap.Match(file.Name) {
                        file.Match = filemap
                        if filemap.statFile(p.absPath, file) {
                                return true
                        }
                }
        }

        if file.Info == nil && file.Dir == "" {
                if file.Info, _ = os.Stat(filepath.Join(p.absPath, file.Name)); file.Info != nil {
                        file.Dir = p.absPath
                }
        }

        if file.Info == nil && file.Dir == "" {
                for _, base := range p.bases {
                        if base.search(file) {
                                return true
                        }
                }
        }

        if res = file.Info != nil; !res {
                //file.Sub = nil
                //file.Dir = ""
        }
        return
}

func (p *Project) searchInDir(file *File, dir, name string, sys bool) (res bool) {
        var isAbs, isRel bool
        if isAbs = filepath.IsAbs(name); isAbs {
                if file.Info, _ = os.Stat(name); file.Info != nil {
                        file.Dir = filepath.Dir(name)
                        return true //continue ForScan
                }
        } else if isRel = isRelPath(name); isRel {
                var s = filepath.Join(dir, name)
                if file.Info, _ = os.Stat(s); file.Info != nil {
                        file.Dir = filepath.Dir(s)
                        return true //continue ForScan
                }
        } else if p.search(file) {
                return true //continue ForScan
        } else if !sys {
                // Check for bare non-system sub-path (e.g. foo/bar/name.xxx)
                if dir := filepath.Dir(name); /*dir != "."*/true {
                        var ( savDir = file.Dir ; savSub = file.Sub )
                        file.Dir, file.Sub = "", nil
                        file.Name = filepath.Base(name)
                        if p.search(file) && strings.HasSuffix(file.Dir, dir) {
                                file.Name = name
                                file.Dir = strings.TrimSuffix(file.Dir, dir)
                                file.Dir = strings.TrimSuffix(file.Dir, PathSep)
                                return true //continue ForScan
                        } else {
                                file.Name, file.Dir, file.Sub = name, savDir, savSub
                        }
                }
        }
        return
}

func (p *Project) searchFile(name string) (file *File, res bool) {
        file = &File{ Name: name }
        res = p.search(file)
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
                if file, okay = p.searchFile(s); enable_assertions {
                        assert(file != nil, "`%s` nil file", s)
                        assert(file.Name == s, "file name differs '%s' != '%s'", file.Name, s)
                        assert(file.Match != nil, "`%s` not matched file", s)
                        if okay {
                                assert(file.Info != nil, "`%v` found nil file info", s)
                                assert(file.Dir != "", "`%v` found empty file dir", s)
                        } else {
                                assert(file.Info == nil, "`%v` non-nil info", s)
                        }
                }
        }
        return // FIXME: return searchFile bool result
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
                        if target.Name == s { return rec, nil }
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
