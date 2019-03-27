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
        "plugin"
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

func (filemap *FileMap) String() string { return filemap.Pattern.String() }

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(filename string) (bool, string) {
        return globMatch(filemap.Pattern, filename)
}

func (filemap *FileMap) stat(base, name string) (file *File) {
        for _, path := range filemap.Paths {
                if path == nil {
                        msg := fmt.Sprintf("`%v` nil", filemap.Paths)
                        panic(msg)
                }

                var ( dir, sub string ; err error )
                if sub, err = path.Strval(); err != nil { return }
                if filepath.IsAbs(sub) {
                        dir = sub
                        sub = ""
                } else {
                        dir = base //filepath.Join(base, sub)
                }

                // Check file in the filesystem.
                if file = stat(name, sub, dir); file != nil {
                        //if file.match == nil { file.match = filemap }
                        break
                }
        }
        return
}

// globMatch - Glob matching each component of the filename against the
// glob value. It checks in two different ways. If the filename and the
// glob pattern has the some number of components (splitted by PathSep),
// all components are compared. If the pattern has only one component,
// the last filename component is compared with the pattern, and the prefix
// components are returned in 'pre'.
func globMatch(patval Value, filename string) (matched bool, pre string) {
        pattern, err := patval.Strval()
        if err != nil { return false, "" }

        list0 := strings.Split(filepath.Clean(pattern), PathSep)
        list1 := strings.Split(filepath.Clean(filename), PathSep)
        if n := len(list0); n == 0 {
                // FIXME: match any?
        } else if m := len(list1); n == m { // foo/*.o  <->  src/foo.o
                // Matching all components
                for i, pat := range list0 {
                        matched, _ = filepath.Match(pat, list1[i])
                        if !matched { return }
                }
        } else if n == 1 && m > 1 { // *.o|foo.o  <->  src/foo.o
                // Matching the last component of filename and returns
                // the prefix if matched.
                if matched, _ = filepath.Match(list0[0], list1[m-1]); matched {
                        pre = filepath.Join(list1[:m-1]...)
                }
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

type useRuleEntry struct {
        RuleEntry
        post bool
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
        imports []*Project

        self *ProjectName

        // List order is significant, duplication is acceptable.
        filemap []*FileMap

        using  *usinglist

        // Rule Registry (orderred)
        userules []*useRuleEntry // the 'use' rule
        concrete []*RuleEntry
        patterns []*PatternEntry

        filescopes []*Scope

        // TODO: printEntering() ...
        // TODO: printLeaving() ...

        plugin *plugin.Plugin
        pluginScope *Scope

        allowMultiImported bool // allow being imported multiple times
        breakRecursiveUsing bool // don't recursively using this project
}

func (p *Project) String() string {
        //return fmt.Sprintf("<project %s>", p.name)
        return p.name
}

func (p *Project) NewScope(comment string) *Scope {
        return NewScope(p.scope, p, comment)
}

func (p *Project) AbsPath() string { return p.absPath }
func (p *Project) RelPath() string { return p.relPath }
func (p *Project) Spec() string { return p.spec }
func (p *Project) Name() string { return p.name }
func (p *Project) Scope() *Scope { return p.scope }
func (p *Project) Bases() []*Project { return p.bases }

func (p *Project) Chain(bases ...*Project) {
        for _, base := range bases {
                p.bases = append(p.bases, base)
        }
}

func (p *Project) mapfile(pat Value, paths []Value) {
        // List order is significant, duplication is acceptable.
        p.filemap = append(p.filemap, &FileMap{ pat, paths })
}

func (p *Project) filemaps(using bool) (filemaps []*FileMap) {
        var unique = make(map[*FileMap]int)
        var app = func(a []*FileMap) {
                for _, m := range a {
                        if _, ok := unique[m]; ok { continue }
                        filemaps = append(filemaps, m)
                        unique[m] += 1
                }
        }
        app(p.filemap)
        for _, base := range p.bases {
                app(base.filemaps(using))
        }
        if using {
                for _, u := range p.using.list {
                        app(u.project.filemaps(using))
                }
        }
        unique = nil
        return
}

func (p *Project) wildcard(patterns ...Value) (files []*File, err error) {
        var filemaps = p.filemaps(false)
ForPats:
        for _, pat := range patterns {
                var ( pre, str string; matched, breakAbsRel bool )
                if str, err = pat.Strval(); err != nil { break ForPats }
                // The 'str' value could be GlobPattern or just
                // regular file/path names. PercPattern is not
                // supported yet.
        ForFilemaps:
                for _, fm := range filemaps {
                        if matched, pre = globMatch(fm.Pattern, str); !matched {
                                // Flip glob matching order.
                                if _, yes := pat.(*GlobPattern); !yes {
                                        continue ForFilemaps
                                } else if str, err = fm.Pattern.Strval(); err != nil {
                                        break ForPats
                                } else if matched, pre = globMatch(pat, str); !matched {
                                        continue ForFilemaps
                                } else {
                                        // using the arg glob
                                        breakAbsRel = true
                                }
                        }

                        if pre != "" { /* FIXME: ... */ }

                        var names []string

                        // Absolute or relative files are not related to the
                        // paths.
                        if filepath.IsAbs(str) || strings.HasPrefix(str, "./") || strings.HasPrefix(str, "../") {
                                if names, err = filepath.Glob(str); err != nil { break ForPats }
                                for _, s := range names {
                                        file := stat(filepath.Base(s), "", filepath.Dir(s))
                                        files = append(files, file)
                                        if enable_assertions {
                                                assert(file != nil, "`%s` missing", s)
                                        }
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
                                        files = append(files, file)
                                        if enable_assertions {
                                                assert(file != nil, "`%s` missing (%s)", s, name)
                                        }
                                }
                        }
                }
        }
        return
}

func (p *Project) searchFile(name string) (file *File) {
        for _, filemap := range p.filemaps(true) {
                // Match the represented file name.
                matched, pre := filemap.Match(name)
                if !matched { continue }
                if file = filemap.stat(p.absPath, name); file != nil {
                        if file.match == nil { file.match = filemap }
                        if pre != "" { /* FIXME: file.change(...pre) */ }
                        if enable_assertions {
                                assert(file.exists(), "`%s` file not existed", file)
                        }
                        break
                }
        }
        if file != nil && enable_assertions {
                assert(file.exists(), "`%s` file not existed", file)
                assert(file.match != nil, "`%s` not matched file", name)
                assert(file.info != nil, "`%v` found nil file info", name)
                if filepath.IsAbs(name) {
                        if strings.HasPrefix(name, file.dir+PathSep) {
                                //assert(file.name == filepath.Base(name), "conflicted name: file{%s %s %s} != %s", file.dir, file.sub, file.name, filepath.Base(name))
                                //assert(file.name == name, "conflicted name: file{%s %s %s} != %s", file.dir, file.sub, file.name, name)
                                //assert(file.dir != "", "invalid file{%s %s %s}", file.dir, file.sub, file.name)
                        } else {
                                assert(file.name == name, "conflicted name: file{%s %s %s} != %s", file.dir, file.sub, file.name, name)
                                assert(file.dir == "", "invalid file{%s %s %s}", file.dir, file.sub, file.name)
                                assert(file.FullName() == file.name, "conflicted name: file{%s %s %s}", file.dir, file.sub, file.name)
                        }
                        assert(file.FullName() == name, "conflicted name: file{%s %s %s}", file.dir, file.sub, file.name)
                } else {
                        assert(file.dir != "", "`%v` found empty file dir", name)
                        assert(filepath.IsAbs(file.dir), "not abs file{%s %s %s}", file.dir, file.sub, file.name)
                }
        }
        return
}

func (p *Project) matchFile(name string) (file *File) {
        var first *File
ForFilemaps:
        for _, filemap := range p.filemaps(true) {
                // Match the represented file name.
                matched, pre := filemap.Match(name)
                if !matched { continue }
                if file = filemap.stat(p.absPath, name); file != nil {
                        if file.match == nil { file.match = filemap }
                        if pre != "" { /* FIXME: file.change(...pre) */ }
                        if enable_assertions {
                                assert(file.exists(), "`%s` file not existed", file)
                        }
                } else if len(filemap.Paths) > 0 {
                        var sub, err = filemap.Paths[0].Strval()
                        sub = filepath.Clean(sub) // clean path
                        if filepath.IsAbs(sub) {
                                if pre == "" {
                                        // For example of:
                                        //   xxx.c  <->  (*.c => /path/to/source)
                                        // Became:
                                        //   /path/to/source  ""  xxx.c
                                        file = stat(name, "", sub, nil)
                                } else if strings.HasSuffix(sub, PathSep+pre) {
                                        // For example of:
                                        //   foo/bar/xxx.c  <->  (*.c => /path/to/source/foo/bar)
                                        // Became:
                                        //   /path/to/source  foo/bar  xxx.c
                                        s := strings.TrimSuffix(sub, PathSep+pre)
                                        n := strings.TrimPrefix(name, pre+PathSep)
                                        file = stat(n, pre, s, nil)
                                } else {
                                        // For example of:
                                        //   foo/bar/xxx.c  <->  (*.c => /path/to/source)
                                        // Became:
                                        //   /path/to/source  foo/bar  xxx.c
                                        n := strings.TrimPrefix(name, pre+PathSep)
                                        file = stat(n, pre, sub, nil)
                                }
                        } else {
                                if pre == "" {
                                        // For example of:
                                        //   xxx.c  <->  (*.c => source)
                                        // Became:
                                        //   <p.absPath>  source  xxx.c
                                        file = stat(name, sub, p.absPath, nil)
                                } else if sub == pre {
                                        // For example of:
                                        //   foo/bar/xxx.c  <->  (*.c => foo/bar)
                                        // Became:
                                        //   <p.absPath>  foo/bar  xxx.c
                                        n := strings.TrimPrefix(name, pre+PathSep)
                                        file = stat(n, sub, p.absPath, nil)
                                } else if strings.HasSuffix(sub, PathSep+pre) {
                                        // For example of:
                                        //   foo/bar/xxx.c  <->  (*.c => source/foo/bar)
                                        // Became:
                                        //   <p.absPath>  source/foo/bar  xxx.c
                                        s := strings.TrimSuffix(sub, PathSep+pre)
                                        n := strings.TrimPrefix(name, pre+PathSep)
                                        file = stat(n, pre, s, nil)
                                } else {
                                        // For example of:
                                        //   foo/bar/xxx.c  <->  (*.c => source)
                                        // Became:
                                        //   <p.absPath>  source/foo/bar  xxx.c
                                        s := filepath.Join(sub, pre)
                                        n := strings.TrimPrefix(name, pre+PathSep)
                                        file = stat(n, s, p.absPath, nil)
                                }
                        }
                        if file.match == nil { file.match = filemap }
                        if enable_assertions {
                                assert(err == nil, "%v: %v", p, err)
                        }
                }
                if file.exists() { break }
                if first == nil { first = file }
                // If the filemap entry is defined by the project itself,
                // we have to break the matching loop. So that the current
                // project have a chance to define it's own file. This is
                // usefull when the bases or imported projects have also
                // matched filemaps. The current project have the highest
                // priority to match.
                for _, fm := range p.filemap {
                        if filemap == fm {
                                break ForFilemaps
                        }
                }
        }
        if first != file && (file == nil || !file.exists()) {
                file = first
        }
        return
}

func (p *Project) isFileName(s string) (res bool) {
        if len(s) > 0 {
                for _, filemap := range p.filemaps(true) {
                        if res, _ = filemap.Match(s); res { break }
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
                if p.pluginScope != nil {
                        if obj = p.pluginScope.Lookup(s); obj != nil {
                                return
                        }
                }
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
        if file := p.matchFile(target); file != nil {
                if enable_assertions {
                        assert(file.match != nil, "`%s` nil match", target)
                }

                // Invoke file rules no matter if it existed or not.
                var okay bool // true if doing good
                if okay, err = p.updateFile(pc, file); err != nil || okay {
                        if optionTracePrepare {
                                if okay {
                                        pc.tracef("%s: updateTarget(file{%s}) (okay)", p.name, file)
                                } else {
                                        pc.tracef("%s: updateTarget(file{%s}): %v", p.name, file, err)
                                }
                        }
                        return
                } else if enable_assertions {
                        assert(err == nil, "got error: %v", err)
                        assert(!file.exists(), "`%s` file exists", file)
                }

                err = fileNotFoundError{p, file}
                if optionTracePrepare {
                        pc.tracef("%s: `updateTarget(file{%s,%s,%s})` not found", p.name, file.dir, file.sub, file.name)
                }
                return
        }

        var entry *RuleEntry
        if entry, err = p.resolveEntry(target); err != nil {
                //if optionTracePrepare { pc.tracef("%s", err) }
                return
        } else if entry != nil {
                err = pc.traverse(entry)
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
       
        err = targetNotFoundError{ p, target }
        if optionTracePrepare {
                pc.tracef("%s: `updateTarget(%s)` not found", p.name, target)
        }
        return
}

func (p *Project) updateFile(pc *preparer, file *File) (okay bool, err error) {
        var names = make(map[string]int)
        for stub := file.filestub; true; stub = stub.other {
                if strings.Index(stub.name, "IntrinsicEnums") > 0 {
                        fmt.Fprintf(stderr, "update: %v : file{%v %v %v}\n", pc.entry.target, stub.dir, stub.sub, stub.name)
                }
                names[stub.name] += 1 // mark to avoid retrying later
                okay, err = p.updateFileStub(pc, stub)
                if err != nil || okay { file.filestub = stub; return }
                if stub.other == file.filestub { break }
        }

        // Try other shorter names
        var name string
        for s, i := file.name, strings.LastIndex(file.name, PathSep); s != "" && i > 0; {
                name = filepath.Join(s[i+1:], name)
                s = s[:i] // slice out the prefix 
                if _, tried := names[name]; !tried {
                        names[name] += 1 // mark to avoid duplication

                        var sub = filepath.Join(file.sub, s)
                        var stub = &filestub{ file.dir, sub, name, file.match, file.filestub.other }
                        file.filestub.other = stub

                        okay, err = p.updateFileStub(pc, stub)
                        if err != nil || okay { file.filestub = stub; return }
                }
                i = strings.LastIndex(s, PathSep)
        }

        names = nil // clean names cache

        if file.exists() {
                pc.addNotExistedTarget1(file)
                okay = true
                return
        }

        if file.match != nil {
                if file.searchInMatchedPaths(p) {
                        pc.addNotExistedTarget1(file)
                        okay = true
                        return
                }
        } else if alt := p.searchFile(file.name); alt != nil {
                pc.addNotExistedTarget1(alt)
                okay = true
                return
        }

        /*
        for _, other := range pc.related {
                if other == p { continue }
                if alt := other.searchFile(file.name); alt != nil {
                        pc.addNotExistedTarget1(alt)
                        okay = true
                        return
                }
        }
        */

        err = fileNotFoundError{p, file}
        if optionTracePrepare {
                //pc.tracef("execstack: %s", execstack)
                pc.tracef("%s: `updateFile({%s,%s,%s})` not found", p.name, file.dir, file.sub, file.name)
        }
        return
}

func (p *Project) updateFileStub(pc *preparer, stub *filestub) (okay bool, err error) {
        var entry *RuleEntry
        if entry, err = p.resolveEntry(stub.name); err != nil {
                return
        } else if entry != nil {
                err, okay = entry.prepare(pc), true
                return
        }

        var pss []*StemmedEntry
        if pss, err = p.resolvePatterns(stub.name); err != nil {
                return
        } else if len(pss) == 0 {
                return //goto SearchFile
        }

ForPatterns:
        for _, ps := range pss {
                for _, prog := range ps.programs {
                        for _, dep := range prog.depends {
                                if g, ok := dep.(*PercPattern); ok && g != nil {
                                        ok, err = checkPatternFileDepend(pc, p, ps, prog, g)
                                        if err != nil { return }
                                        if !ok { continue ForPatterns }
                                }
                        }
                }
                ps.stub = stub // Bounds StemmedEntry with the File.
                if err = ps.prepare(pc); err == nil {
                        okay = true
                        return // Updated successfully!
                } else if e, ok := err.(patternPrepareError); ok {
                        if _, ok = e.error.(*breaker); ok {
                                okay = true
                                return // Breaked!
                        }
                } else {
                        okay = true
                        return // Update failed!
                }
        }
        return
}

func (p *Project) entry(special specialRule, options []Value, target Value, prog *Program) (entry *RuleEntry, err error) {
        defer func() {
                if entry != nil && err == nil {
                        entry.programs = append(entry.programs, prog)
                }
        } ()

        var strval string
        if strval, err = target.Strval(); err != nil {
                return
        }

        // The 'use' rule entries.
        var closured = target.closured()
        if special == specialRuleUse && !closured {
                var optPostExecute bool
                for _, v := range options {
                        var opt bool
                        switch t := v.(type) {
                        case *Flag:
                                if opt, err = t.is(0, "post"); err != nil { return }
                                if opt { optPostExecute = true }
                        case *Pair:
                                if opt, err = t.isFlag(0, "post"); err != nil { return }
                                if opt { optPostExecute = t.Value.True() }
                        default:
                                err = fmt.Errorf("`%v` invalid package option (%T)", v, v)
                                return
                        }
                }

                var userule = &useRuleEntry{
                        RuleEntry{ class:UseRuleEntry, target:target },
                        optPostExecute, // post-execute use rule?
                }
                p.userules = append(p.userules, userule)
                entry = &userule.RuleEntry
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
                assert(pat != nil, "nil pattern")
                entry = &RuleEntry{
                        class: GlobRuleEntry,
                        target: target,
                }
                p.patterns = append(p.patterns, &PatternEntry{ pat, entry })
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
        if k, v, err = p.CmdHash(target, recipes); err != nil { return }
        if f, e := os.Open(filepath.Join(dir, fmt.Sprintf("%x", k))); e == nil {
                var h []byte
                if n, e := fmt.Fscanf(f, "%x", &h); e != nil {
                        err = e; return
                } else if n == 1 {
                        same = bytes.Equal(v[:], h)
                        //fmt.Fprintf(stderr, "CheckCmdHash: %x -> %x (%x)\n", k, v, h)
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
                //fmt.Fprintf(stderr, "UpdateCmdHash: %x -> %x (%s)\n", k, v, target.Strval())
                fmt.Fprintf(f, "%x", v)
                err = f.Close()
        } else {
                err = e
        }
        return
}

func (p *Project) isa(proj *Project) (res bool) {
        for _, base := range p.bases {
                if base == proj { res = true; break }
        }
        return
}

func (p *Project) hasBase(proj *Project) (res bool) {
        for _, base := range p.bases {
                if res = base == proj; res { break }
                if res = base.hasBase(proj); res { break }
        }
        return
}

func (p *Project) hasImported(proj *Project) (rp *Project, res, isb bool, err error) {
        return p.hasImportedRecur(p, proj)
}

func (p *Project) hasImportedRecur(top, proj *Project) (rp *Project, res, isb bool, err error) {
        for _, base := range p.bases {
                if isb = base == proj; isb { return }
                if rp, res, isb, err = base.hasImportedRecur(top, proj); err != nil {
                        return
                } else if res || isb { rp = base ; return }
        }
        for _, imp := range p.imports {
                if imp == top {
                        s := top.loopImportPath()
                        err = fmt.Errorf("loop `%v`", s)
                        return
                }
                if res = imp == proj; res { rp = imp; return }
                if rp, res, res, err = imp.hasImportedRecur(top, proj); err != nil {
                        return
                } else if res { rp = imp; return }
        }
        rp = p
        return
}

func (p *Project) loopImportPath() (s string) { return p.loopImportRecur(p) }
func (p *Project) loopImportRecur(top *Project) (s string) {
        for _, imp := range p.imports {
                if imp == top {
                        if p != top { s = "⇢" }
                        s += p.name + "⇢" + imp.name
                        break
                }
                if t := imp.loopImportRecur(top); t != "" {
                        if p != top { s = "⇢" }
                        s += p.name + t
                        break
                }
        }
        return
}

func (p *Project) isUsingProject(usee *Project) (res bool) {
        for _, using := range p.using.list {
                if res = using.project == usee; res { break  }
                if res = using.project.isUsingProject(usee); res { break }
        }
        return
}

func (p *Project) isUsingDirectly(proj *Project) (res bool) {
        for _, u := range p.using.list {
                if res = u.project == proj; res { break }
        }
        return
}

func (p *Project) usees(post bool) (res []*Project) {
        if p.breakRecursiveUsing { return }
        for _, u := range p.using.list {
                if !post { res = append(res, u.project) }
                for _, u := range u.project.usees(post) {
                        if !p.isUsingDirectly(u) { res = append(res, u) }
                }
                if post { res = append(res, u.project) }
        }
        return
}

func enter(prog *Program, dir string) (err error) {
        if optionTraceEntering {
                fmt.Fprintf(stderr, "entering: %v (%v)\n", dir, prog.project.name)
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

func leave(prog *Program, stop *enterec) (err error) {
        var size = len(cd.stack)
        if optionTraceEntering {
                fmt.Fprintf(stderr, "leaving: %v (%v %v %v)\n", stop.dir, prog.project.name, stop.num, size)
        }

        for _, enter := range cd.stack {
                if enter.num == 0 { continue } else {
                        enter.num -= 1
                }

                if enter == stop {
                        if enter.print && false {
                                enter.print = false
                                fmt.Fprintf(stderr, "smart:  Leaving directory '%s'\n", enter.dir)
                        }
                        err = os.Chdir(enter.wd)
                        break
                }
        }

        // Erase 'zero' and unprint records, the first record is always kept.
        // So that the right entering/leaving pairs are printed.
        if size > 1 {
                var stack = []*enterec{ cd.stack[0] }
                for i := 1; i < size; i += 1 {
                        var rec = cd.stack[i]
                        if rec.num > 0 || rec.print {
                                stack = append(stack, rec)
                        }
                }
                cd.stack = stack
        }
        return
}

func printEnteringDirectory() {
        if size := len(cd.stack); size > 0 {
                var enter = cd.stack[0]
                if enter.silent { return }
                for _, p := range cd.stack {
                        if p.print && p != enter {
                                p.print = false
                                fmt.Fprintf(stderr, "smart:  Leaving directory '%s'\n", p.dir)
                        }
                }
                if !enter.print {
                        enter.print = true
                        fmt.Fprintf(stderr, "smart: Entering directory '%s'\n", enter.dir)
                }
        }
}

func printLeavingDirectory() {
        if size := len(cd.stack); size > 0 {
                for _, enter := range cd.stack {
                        if enter.print {
                                enter.print = false
                                fmt.Fprintf(stderr, "smart:  Leaving directory '%s'\n", enter.dir)
                        }
                }
        }
}
