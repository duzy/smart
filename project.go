//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "path/filepath"
        "runtime"
        "strings"
        "plugin"
        "sync"
        "time"
        "fmt"
        "os"
)

const PathSep = string(filepath.Separator)

type FileMap struct {
        Pattern Value
        Paths []Value
}

func (filemap *FileMap) String() string { return filemap.Pattern.String() }

func (filemap *FileMap) isRealPattern() (result bool) {
        switch t := filemap.Pattern.(type) {
        case Pattern: result = true
        case *Path:
                /*if t.File == nil {
                        for _, seg := range t.Elems {
                                _, result = seg.(Pattern)
                                if result { return }
                        }
                }*/
                if result = t.isPattern(); result { return }
        }
        return
}

// Match split filename into list and match each part with the pattern correspondingly.
func (filemap *FileMap) Match(filename string) (matched bool, pre string) {
        matched, pre = globMatch(filemap.Pattern, filename)
        if matched { return }
        if false { // TODO: support percent (%, %%) and regex matching
                var ( s, t string ; e error )
                if t, e = filemap.Pattern.Strval(); e != nil { return }
                for _, p := range filemap.Paths {
                        if s, e = p.Strval(); e != nil { return }
                        if filename == filepath.Join(s, t) {
                                matched = true
                        }
                }
        }
        return
}

func (filemap *FileMap) stat(base, pre, name string) (file *File) {
        var pos = filemap.Pattern.Position()
        if filemap.Paths == nil {
                // Check file in the filesystem (no paths).
                file = stat(pos, name, "", base, nil)
                return
        }
        for _, path := range filemap.Paths {
                if path == nil {
                        panic(fmt.Sprintf("`%v` nil", filemap.Paths))
                }

                var ( dir, sub string ; err error )
                if sub, err = path.Strval(); err != nil { return }

                // Clean the search path.
                sub = filepath.Clean(sub)

                // Absolute path or using the base.
                if filepath.IsAbs(sub) {
                        dir = sub
                        sub = ""
                } else {
                        dir = base //filepath.Join(base, sub)
                }

                // Check file in the filesystem.
                if file = stat(pos, name, sub, dir, nil); file != nil {
                        break
                }

                if filepath.IsAbs(sub) {
                        if pre == "" { // Fullmatch!
                                // For example of:
                                //   xxx.c  <->  (*.c => /path/to/source)
                                // Become:
                                //   /path/to/source  ""  xxx.c
                                file = stat(pos, name, "", sub, nil)
                        } else if strings.HasSuffix(sub, PathSep+pre) {
                                // For example of:
                                //   foo/bar/xxx.c  <->  (*.c => /path/to/source/foo/bar)
                                // Become:
                                //   /path/to/source  foo/bar  xxx.c
                                s := strings.TrimSuffix(sub, PathSep+pre)
                                n := strings.TrimPrefix(name, pre+PathSep)
                                file = stat(pos, n, pre, s, nil)
                        } else if false { // This is wrong, only base name matched!!
                                // For example of:
                                //   foo/bar/xxx.c  <->  (*.c => /path/to/source)
                                // Become:
                                //   /path/to/source  foo/bar  xxx.c
                                n := strings.TrimPrefix(name, pre+PathSep)
                                file = stat(pos, n, pre, sub, nil)
                        }
                } else {
                        if pre == "" { // Fullmatch!
                                // For example of:
                                //   xxx.c  <->  (*.c => source)
                                // Become:
                                //   <p.absPath>  source  xxx.c
                                file = stat(pos, name, sub, dir, nil)
                        } else if sub == pre {
                                // For example of:
                                //   foo/bar/xxx.c  <->  (*.c => foo/bar)
                                // Become:
                                //   <dir>  foo/bar  xxx.c
                                n := strings.TrimPrefix(name, pre+PathSep)
                                file = stat(pos, n, sub, dir, nil)
                        } else if strings.HasSuffix(sub, PathSep+pre) {
                                // For example of:
                                //   foo/bar/xxx.c  <->  (*.c => source/foo/bar)
                                // Become:
                                //   <dir>  source/foo/bar  xxx.c
                                s := strings.TrimSuffix(sub, PathSep+pre)
                                n := strings.TrimPrefix(name, pre+PathSep)
                                file = stat(pos, n, pre, s, nil)
                        } else if false { // This is wrong, only base name matched!!
                                // For example of:
                                //   foo/bar/xxx.c  <->  (*.c => source)
                                // Become:
                                //   <dir>  source/foo/bar  xxx.c
                                s := filepath.Join(sub, pre)
                                n := strings.TrimPrefix(name, pre+PathSep)
                                file = stat(pos, n, s, dir, nil)
                        }
                }
        }
        return
}

// copy of filepath.hasMeta
func hasGlobMeta(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
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
        if len(list0) == 0 {
                // FIXME: match any?
        } else if len(list0) == len(list1) { // foo/*.o  <->  src/foo.o
                // Matching all components
                for i, pat := range list0 {
                        if true /*hasGlobMeta(pat)*/ {
                                matched, _ = filepath.Match(pat, list1[i])
                                if !matched { return }
                        } else {
                                matched = (pat == list1[i])
                        }
                }
        } else if len(list0) == 1 && len(list1) > 1 { // *.o|foo.o  <->  src/foo.o
                // Matching the last component of filename and returns
                // the prefix if matched.
                list1_tail := list1[len(list1)-1]
                if true /*hasGlobMeta(list0[0])*/ {
                        matched, _ = filepath.Match(list0[0], list1_tail)
                } else {
                        matched = (list0[0] == list1_tail)
                }
                if matched {
                        pre = filepath.Join(list1[:len(list1)-1]...)
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
        position Position
        keyword  token.Token // project, package, module

        self *ProjectName // $:self:

        changedWD string
	absPath string
	relPath string
        tmpPath string
	spec    string
	name    string
        scope   *Scope
        bases []*Project
        loads []*Project
        using   *usinglist

        // List order is significant, duplication is acceptable.
        filemap []*FileMap

        // Rule Registry (orderred)
        userules []*useRuleEntry // the 'use' rule
        concrete []*RuleEntry
        patterns []*PatternEntry

        filescopes []*Scope

        // TODO: printEntering() ...
        // TODO: printLeaving() ...

        plugin *plugin.Plugin
        pluginScope *Scope

        multiUseAllowed bool // this project is used multiple times
        breakUseLoop bool // don't recursively use this project
}

func (p *Project) String() string {
        //return fmt.Sprintf("<project %s>", p.name)
        return p.name
}

func (p *Project) NewScope(pos Position, comment string) *Scope {
        return NewScope(pos, p.scope, p, comment)
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

func (p *Project) filemaps() (filemaps []*FileMap) {
        var appendUnique = func(a *FileMap) {
                for _, m := range filemaps {
                        if a == m { return }
                }
                filemaps = append(filemaps, a)
        }
        for _, m := range p.filemap {
                appendUnique(m)
        }
        for _, base := range p.bases {
                for _, m := range base.filemaps() {
                        appendUnique(m)
                }
        }
        /*
        if false {
                for _, u := range p.using.list {
                        app(u.project.filemaps(loads))
                }
        } else {
                for _, proj := range p.loads {
                        app(proj.filemaps(loads))
                }
        }
        */
        return
}

func (p *Project) wildcard(pos Position, wo wildcardOpts, patterns ...Value) (files []*File, err error) {
        var filemaps = p.filemaps()
ForPats:
        for _, pat := range patterns {
                var ( patStr string; matched, breakAbsRel bool )
                if patStr, err = pat.Strval(); err != nil { break ForPats }
                // The 'patStr' could be GlobPattern or just
                // regular file/path names. PercPattern is not
                // supported yet.
        ForFilemaps:
                for _, fm := range filemaps {
                        var pre string // <pre>/*.xxx
                        var str = patStr
                        if matched, pre = globMatch(fm.Pattern, patStr); !matched {
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
                                        file := stat(pos, filepath.Base(s), "", filepath.Dir(s))
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
                        for _, path := range fm.Paths {
                                var sub string
                                if sub, err = path.Strval(); err != nil {
                                        break ForPats
                                }

                                subfile := filepath.Join(sub, str)
                                if names, err = filepath.Glob(subfile); err != nil {
                                        break ForPats
                                }
                                // Chop off path 'sub' prefix to have shorter names
                                // Aka. trim prefix 'file.Sub+PathSep'
                                prefix := strings.TrimSuffix(subfile, str)
                                if len(names) > 0 {
                                        for _, s := range names {
                                                name := strings.TrimPrefix(s, prefix)
                                                file := stat(pos, name, sub, prefix)
                                                files = append(files, file)
                                                if enable_assertions {
                                                        assert(file != nil, "`%s` missing (%s)", s, name)
                                                }
                                        }
                                } else if ok := fm.isRealPattern(); !ok && wo.optIncludeMissing {
                                        // If the filemap is not a pattern (e.g. foobar.cpp),
                                        // we include it in the returning files.
                                        var name string
                                        name, err = fm.Pattern.Strval()
                                        if err != nil { break ForPats }

                                        // Append this non-existed/missing file.
                                        file := stat(pos, name, sub, prefix, nil)
                                        files = append(files, file)

                                        if false { fmt.Fprintf(stderr, "%s: %s -> %s\n", pos, pat, file) }
                                } else if ok {
                                        // Just report that the pattern matches no files in the
                                        // file system.
                                        fmt.Fprintf(stderr, "%s: wildcard '%s' in %s: files like '%v' not found in %v\n", pos, pat, p.name, fm, sub)
                                } else if optionWildcardMissingError {
                                        err = fmt.Errorf("files like '%v' not found", fm)
                                        break ForPats
                                }
                        }
                }
        }
        return
}

// TODO: searchFile deprecated, use only matchFile instead
func (p *Project) searchFile(name string) (file *File) {
        for _, filemap := range p.filemaps() {
                // Match the represented file name.
                matched, pre := filemap.Match(name)
                if !matched { continue }
                if p.changedWD != "" {
                        file = filemap.stat(p.changedWD, pre, name)
                }
                if file == nil {
                        file = filemap.stat(p.absPath, pre, name)
                }
                if file != nil {
                        if file.match == nil { file.match = filemap }
                        if pre != "" { /* FIXME: file.change(...pre) */ }
                        if enable_assertions {
                                assert(exists(file), "`%s` file not existed", file)
                        }
                        break
                }
        }
        if file != nil && enable_assertions {
                assert(exists(file), "`%s` file not existed", file)
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
                                assert(file.fullname() == file.name, "conflicted name: file{%s %s %s}", file.dir, file.sub, file.name)
                        }
                        assert(file.fullname() == name, "conflicted name: file{%s %s %s}", file.dir, file.sub, file.name)
                } else {
                        assert(file.dir != "", "`%v` found empty file dir", name)
                        assert(filepath.IsAbs(file.dir), "not abs file{%s %s %s}", file.dir, file.sub, file.name)
                }
        }
        return
}

func (p *Project) matchFile(name string) (file *File) {
        //[optional]: defer setclosure(setclosure(cloctx.unshift(p.scope)))

        var isNameMatched bool
        var first *File
ForFilemaps:
        for _, filemap := range p.filemaps() {
                // Match the represented file name.
                var matched, pre = filemap.Match(name)
                if !matched { continue ForFilemaps } else {
                        isNameMatched = true
                }
                if p.changedWD != "" {
                        file = filemap.stat(p.changedWD, pre, name)
                }
                if file == nil {
                        file = filemap.stat(p.absPath, pre, name)
                }
                if file != nil {
                        if file.match == nil { file.match = filemap }
                        if pre != "" { /* FIXME: file.change(...pre) */ }
                        if exists(file) { break ForFilemaps }
                        if first == nil { first = file }
                }
                // If the filemap entry is defined by the project itself,
                // we have to break the matching loop. So that the current
                // project have a chance to define it's own file. This is
                // usefull when the bases (or imported projects) have also
                // matched files. The current project have the highest
                // priority to match.
                for _, fm := range p.filemap {
                        if filemap == fm {
                                break ForFilemaps
                        }
                }
        }
        if first != file && !exists(file) {
                file = first
        }
        if isNameMatched && enable_assertions {
                assert(file != nil, "`%s` is a file", name)
        }
        return
}

func (p *Project) matchTempFile(pos Position, name string) (file *File) {
        if file = p.matchFile(name); file != nil {
                // good
        } else if ctd := p.scope.FindDef("CTD"); ctd == nil {
                unreachable()
        } else if s, err := ctd.Strval(); err == nil {
                // stat temp file (maybe not existed)
                file = stat(pos, filepath.Join(s, name), "", "", nil)
        } else {
                fmt.Fprintf(stderr, "%v: %v\n", p, err)
        }
        return
}

func (p *Project) isFileName(s string) (res bool) {
        if len(s) > 0 {
                for _, filemap := range p.filemaps() {
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
        if err == nil && entry == nil {
                /*for _, using := range p.using.list {
                        entry, err = using.project.resolveEntry(s)
                        if err != nil || entry != nil { break }
                }*/
        }
        return
}

func (p *Project) resolvePatterns(i interface{}) (res []*StemmedEntry, err error) {
        for _, pat := range p.patterns {
                var ( s string ; stems []string )
                if s, stems, err = pat.Pattern.match(i); err != nil {
                        return
                } else if s != "" && stems != nil {
                        res = append(res, &StemmedEntry{
                                pat, stems, s, nil,
                        })
                }
        }
        for _, base := range p.bases {
                var ses []*StemmedEntry
                ses, err = base.resolvePatterns(i)
                if err != nil { return }
                res = append(res, ses...)
        }
        for _, using := range p.using.list {
                var ses []*StemmedEntry
                ses, err = using.project.resolvePatterns(i)
                if err != nil { return }
                res = append(res, ses...)
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
                if _, err = parseFlags(options, []string{
                        "p,post",
                }, func(ru rune, v Value) {
                        switch ru {
                        case 'p': optPostExecute = trueVal(v, false)
                        }
                }); err != nil { return }
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
        switch t := target.(type) {
        case *PercPattern:
                assert(t != nil, "nil PercPattern")
                entry = &RuleEntry{
                        class: PercRuleEntry,
                        target: target,
                }
                p.patterns = append(p.patterns, &PatternEntry{ t, entry })
                return
        case *GlobPattern:
                assert(t != nil, "nil GlobPattern")
                entry = &RuleEntry{
                        class: GlobRuleEntry,
                        target: target,
                }
                panic("TODO: GlobPattern target")
        case *RegexpPattern:
                assert(t != nil, "nil RegexpRuleEntry")
                entry = &RuleEntry{
                        class: RegexpRuleEntry,
                        target: target,
                }
                panic("TODO: RegexpPattern target")
        case *Path:
                var isPathPattern bool
        ForPathElements:
                for _, elem := range t.Elems {
                        switch elem.(type) {
                        case *PercPattern:
                                isPathPattern = true
                                break ForPathElements
                        case *GlobPattern:
                                panic("TODO: GlobPattern path target")
                        case *RegexpPattern:
                                panic("TODO: RegexpPattern path target")
                        }
                }
                if isPathPattern {
                        entry = &RuleEntry{
                                class: PathPattRuleEntry,
                                target: target,
                        }
                        p.patterns = append(p.patterns, &PatternEntry{ t, entry })
                        return
                }
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

func (p *Project) hasLoaded(proj *Project, breakUseLoop bool) (rp *Project, res, isb bool, err error) {
        return p.hasLoadedRecur(p, proj, breakUseLoop)
}

func (p *Project) hasLoadedRecur(top, proj *Project, breakUseLoop bool) (rp *Project, res, isb bool, err error) {
        for _, base := range p.bases {
                if isb = base == proj; isb { return }
                if rp, res, isb, err = base.hasLoadedRecur(top, proj, breakUseLoop); err != nil {
                        return
                } else if res || isb { rp = base ; return }
        }
        for _, imp := range p.loads {
                if imp == top && !breakUseLoop {
                        s := top.loopLoadPath()
                        err = fmt.Errorf("loop `%v`", s)
                        return
                }
                if res = imp == proj; res { rp = imp; return }
                if rp, res, res, err = imp.hasLoadedRecur(top, proj, breakUseLoop); err != nil {
                        return
                } else if res { rp = imp; return }
        }
        rp = p
        return
}

func (p *Project) loopLoadPath() (s string) { return p.loopLoadRecur(p) }
func (p *Project) loopLoadRecur(top *Project) (s string) {
        for _, imp := range p.loads {
                if imp == top {
                        if p != top { s = "⇢" }
                        s += fmt.Sprintf("(%s)⇢(%s)", p.spec, imp.spec)
                        break
                }
                if t := imp.loopLoadRecur(top); t != "" {
                        if p != top { s = "⇢" }
                        s += fmt.Sprintf("(%s)%s", p.spec, t)
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
        if p.breakUseLoop { return }
        for _, u := range p.using.list {
                if !post { res = append(res, u.project) }
                for _, u := range u.project.usees(post) {
                        if !p.isUsingDirectly(u) { res = append(res, u) }
                }
                if post { res = append(res, u.project) }
        }
        return
}

//var cdUnlocked = make(chan bool, 1)
// Note: this is okay not using an atomic value, because
// cdUnlockMutex can serve to protect the whole timeframe.
//var cdUnlockTime atomic.Value
var cdUnlockMutex = new(sync.Mutex)

func lockCD(dir string, lockDura time.Duration) error {
        // Protect the work directory, `cdUnlockMutex` ensures that
        // there's only one timer being counting to avoid work
        // directory being changed before the deadline.
        cdUnlockMutex.Lock()
        /*
        defer cdUnlockMutex.Unlock()
        if v := cdUnlockTime.Load(); v == nil {
                // no deadline was set
        } else if t, ok := v.(time.Time); ok && t.After(time.Now()) {
                //for t.After(time.Now())
                select {
                //case <-cdUnlocked: //cdLocker.Wait():
                case <-time.After(time.Until(t)): //(t.Sub(time.Now())):
                }
        }
        if lockDura > 0 {
                cdUnlockTime.Store(time.Now().Add(lockDura))
        } */
        if lockDura > 0 {
                //fmt.Printf("cd: %s (lock %v)\n", dir, lockDura)
                go func() {
                        time.Sleep(lockDura)
                        cdUnlockMutex.Unlock()
                        //fmt.Printf("cd: %s (unlocked)\n", dir)
                } ()
        } else {
                //fmt.Printf("cd: %s\n", dir)
                defer cdUnlockMutex.Unlock()
        }
        return os.Chdir(dir)
}

func enter(prog *Program, dir string) (err error) {
        if optionTraceEntering {
                fmt.Fprintf(stderr, "entering: %v (%v)\n", dir, prog.project.name)
        }

        var wd string
        if wd, err = os.Getwd(); err != nil { return }
        if err = lockCD(dir, 0); err != nil { return }
        if !filepath.IsAbs(dir) { dir = filepath.Join(wd, dir) }
        prog.auto("CWD", &String{trivial{prog.position},dir})

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
                        err = lockCD(enter.wd, 0)
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
