//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "path/filepath"
    "reflect"
    "strings"
    "plugin"
    "sync"
    "time"
    "fmt"
    "os"
)

const configuration_sm = "configuration.sm"
const pathSepByte = filepath.Separator
const pathSep = string(pathSepByte)

type _filemap struct { project *project ; patts, paths []Value }
func (p *_filemap) String() string { return fmt.Sprintf("%s", p.patts) }

func filemap_str(t *[]filemap) (s string) {
	for i, t := range *t { if 0 < i { s += " " }; s += t.String() }
	return "["+s+"]"
}

type filemap struct { *_filemap ; pattern Value }
func (p *filemap) ts(t string) string { return fmt.Sprintf("{=%s %v}", t, ts(p.pattern)) }
func (p *filemap) String() (s string) {
    if p.pattern == nil {
        s = p._filemap.String()
    } else {
        s = p.pattern.String()
    }
    return
}

func (p *filemap) primePatterns(ctx Context) (pats []Value) {
    var patts = []Value{ p.pattern }
    if patts[0] == nil { patts = p.patts }

    for _, pattern := range patts {
        // NOTE it may preserve closure patterns after this expand
        pat := expand(ctx,pattern)
        pats = append(pats, merge(pat)...)
    }
    return
}

// match split filename into list and match each part with the pattern correspondingly.
func (p *filemap) match(ctx Context, val any) (_ bool, _ Value, _ string) {
    // TODO: escape file matching for 'String' and "strcomp" values
    for _, pat := range p.primePatterns(ctx) {
        if matched, name := p._match(ctx, pat, val); matched {
            return matched, pat, name
        }
    }
    return
}

func (p *filemap) _match(ctx Context, pat Value, val any) (matched bool, name string) {
    // TODO: escape file matching for 'String' and "strcomp" values
    var res any
    matched, res, _ = match(ctx, pat, val)

    if false && !matched && !(isNone(pat) || isNull(pat)) {
        var str string // NOOP
        if n := strings.Index(str, pathSep); n < 0 { return }

        // NOTE: Dealing with these files:
        //     files (
        //         (foo.c) ⇒ $(srcdir)/sub/dir
        //         (sub/dir/foo.c) ⇒ $(srcdir)
        //     )
        for _, p := range p.paths { // FIXME: performance, operate on p.(*path) instead
            if _, ok := p.(*path); !ok { continue } // NOTE: only work with paths to improve performance
            var ps = __string(ctx, p)
            for i := strings.LastIndex(ps, pathSep); -1 <= i; {
                var ( prefix = ps[i+1:]; l = len(prefix) ) // NOTE: -1 <= i < len(ps)
                if has := strings.HasPrefix(str, prefix) && str[l] == '/'; has {
                    if matched, _, _ = match(ctx, pat, str[len(prefix)+1:]); matched { break }
                }
                if 0 < i { i = strings.LastIndex(ps[:i], pathSep) } else { break }
            }
        }
    }

    if res == nil {
        // okay
    } else if s, y := res.(string); y {
        name = s
    } else if a, y := res.([]string); y {
        name = joinpath(a...)
    } else {
        erro(ctx, "unexpected result: %v", ts(res)).trace()
    }
    return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (p *filemap) stat(ctx Context, name string) (res *file) {
    var patts = p.patts
    if len(patts) == 0 {
        erro(ctx, "no map patterns: %v", p).trace()
    }

    if len(p.paths) == 0 {
        for _, pat := range p.patts {
            if f, y := pat.(*file); y && ident(ctx,f) == name { return f }
        }
        for i, pat := range p.patts {
            if f, y := pat.(*file); y {
                info(ctx, "pattern %d. %v %s (exists=%v)", i, f, f.fullname(), f.exists())
            } else {
                info(ctx, "pattern %d. %v", i, ts(pat))
            }
        }
        errostack(ctx, 5, "%s → %v", name, p.patts).trace()
    }

    for _, path := range p.paths {
        if isNull(path) {
            erro(ctx, "nil path: name=%s",  name)
            erro(ctx, "nil path: %v", p).trace()
        } else if isNone(path) {
            warn(ctx, "nil path: name=%s",  name)
            warn(ctx, "nil path: %v", p).debug(32)
            continue
        }

        var dir, sub string

        if sub = __string(ctx, path); sub == "" {
            if false {
                erro(ctx, "empty filemap path: %v, patterns=%v", path, patts).trace()
            }
            return
        }

        if s := filepath.Clean(sub); sub != s { sub = s }

        if filepath.IsAbs(sub) {      // 'sub' is abs
            if filepath.IsAbs(name) { // 'name' is abs too
                if s := sub+pathSep; strings.HasPrefix(name, s) { // 'name' should have 'sub' prefix
                    name = strings.TrimPrefix(name, s)
                } else {
                    continue
                }
            }
        } else if !filepath.IsAbs(name) {
            // dir = filepath.Clean(base)
        }

        if res = _stat(ctx, name, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true}); res != nil { break }

        var pre string // Not used!
        if filepath.IsAbs(sub) {
            if pre == "" { // Fullmatch!
                // For example of:
                //   xxx.c  <->  (*.c => /path/to/source)
                // Become:
                //   /path/to/source  ""  xxx.c
                res = _stat(ctx, name, stat_dir{sub}, stat_nonexist{true})
            } else if strings.HasSuffix(sub, pathSep+pre) {
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => /path/to/source/foo/bar)
                // Become:
                //   /path/to/source  foo/bar  xxx.c
                s := strings.TrimSuffix(sub, pathSep+pre)
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{pre}, stat_dir{s}, stat_nonexist{true})
            } else if false { // This is wrong, only base name matched!!
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => /path/to/source)
                // Become:
                //   /path/to/source  foo/bar  xxx.c
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{pre}, stat_dir{sub}, stat_nonexist{true})
            }
        } else {
            if pre == "" { // Fullmatch!
                // For example of:
                //   xxx.c  <->  (*.c => source)
                // Become:
                //   <p.absPath>  source  xxx.c
                res = _stat(ctx, name, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true})
            } else if sub == pre {
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => foo/bar)
                // Become:
                //   <dir>  foo/bar  xxx.c
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true})
            } else if strings.HasSuffix(sub, pathSep+pre) {
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => source/foo/bar)
                // Become:
                //   <dir>  source/foo/bar  xxx.c
                s := strings.TrimSuffix(sub, pathSep+pre)
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{pre}, stat_dir{s}, stat_nonexist{true})
            } else if false { // This is wrong, only base name matched!!
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => source)
                // Become:
                //   <dir>  source/foo/bar  xxx.c
                s := filepath.Join(sub, pre)
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{s}, stat_dir{dir}, stat_nonexist{true})
            }
        }
    }
    return
}

type debug_y struct{}
type debug_ctx struct{ Context }
func (c debug_ctx) cast(t reflect.Type) Context { return icast(c,t) }
func (c debug_ctx) inner() Context { return c.Context }
func (c debug_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case debug_y: return true
    }
    return c.Context.do(ctx, op)
}

type unmap_uncheck_y struct{}
type unmap_uncheck_ctx struct{ Context }
func (c unmap_uncheck_ctx) cast(t reflect.Type) Context { return icast(c,t) }
func (c unmap_uncheck_ctx) inner() Context { return c.Context }
func (c unmap_uncheck_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case unmap_uncheck_y: return true
    }
    return c.Context.do(ctx, op)
}

type project_ctx struct{ Context ; p *project }
func (c project_ctx) cast(t reflect.Type) Context { return icast(c,t) }
func (c project_ctx) inner() Context { return c.Context }
func (c project_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case get_project: return c.p
    }
    return c.Context.do(ctx, op)
}

type project_ext struct{ *plugin.Plugin }
type project struct {
    *scope

    position Position

    bases []*project

    use *uselist

    configure *project // .configure project
    configuration *file // configuration.sm if saved or loaded

    absPath string
    tmpPath string
    name    string
    rel     string // path segment relative to the workBaseDir
    spec    string // relative to search-paths as a specification

    filemap valcache
    entries valcache

    patterns []*rule // order is important
    configs  []*def // configure entries
    main entry

    ext project_ext
    opt project_opts
}
func (_ *project) kind() Kind { return KindObject|KindKnownObject|KindProject }
func (p *project) stencil(_ Context, stems []string) (Value, []string) { return p, stems }
func (p *project) Position() Position { return p.position }
func (p *project) String() string { return "{=project "+p.name+"}" }
func (p *project) owner() *project { return p.scope.project }
func (p *project) ts(ctx Context, t string) string {
    return "{" + lp(ctx,p.position,t) + " " + p.name + "}"
}

type self struct { *project }
func (p self) kind() Kind { return p.project.kind()|KindSelf }
func (p self) String() string { return "{=self "+p.name+"}" }

func findfile(ctx Context, s string, ps ...*project) (_ *file) {
    if len(ps) == 0 { ps = append(ps, _project(ctx)) }
    for _, p := range ps { if f := p.file(ctx, s); f != nil { return f } }
    return
}

func select_file_1(ctx Context, m filemap_name) (res *file) {
    defer func() { if res != nil { res.filemap = &m.filemap } } ()

    if m.paths == nil {
        if res, _ = m.pattern.(*file); res != nil {
            return
        } else {
            var s = _project(ctx).absPath
            return _stat(ctx, m.string, stat_dir{s}, stat_nonexist{true})
        }
    }

    var fs []*file

    for _, v := range m.paths {
        if t := expand(_final(ctx),v); t != nil {
            if s := __string(ctx, t); s != "" {
                if f := _stat(ctx, m.string, stat_dir{s}, stat_nonexist{true}); f != nil {
                    fs = append(fs, f)
                } else {
                    erro(ctx, "%s ⇒ %v → %v → ''", m.string, v, t).trace()
                }
            } else if false {
                erro(ctx, "%s ⇒ %v → %v → ''", m.string, v, t).trace()
            }
        } else {
            erro(ctx, "%s ⇒ %v", m.string, v).trace()
        }
    }

    for _, f := range fs { if f.exists() { return f } }
    if 0 < len(fs) { res = fs[0] }
    return
}

func select_files(ctx Context, m []filemap_name) (res []*file) {
    for _, m := range m {
        if f := select_file_1(ctx, m); f != nil {
            res = append(res, f)
        }
    }
    return
}

func select_file(ctx Context, m []filemap_name) (res *file) {
    if a := select_files(ctx, m); 0 < len(a) {
        if res = a[0] ; !res.exists() {
            for _, f := range a { if f.exists() { return f } }
        }
    }
    return
}

func (p *project) file(ctx Context, a any) *file {
    return select_file(ctx, unmap_files(ctx, p, a, nil))
}

func (p *project) tempdir(ctx Context) (d *def, s string) {
    for _, t := range []string{"outtmp", ".tmp", "CTD"} {
        if d = p.resolveDef(ctx, t); d != nil { break }
    }

    if d == nil {
        erro(ctx, "%v: tmp is not defined", p).trace()
    }

    s = filepath.Clean(__string(/*closure_with(ctx,p)*/ctx, d))

    if checkpoints { tempdir_check(ctx, p, d, s) }
    return
}

func (p *project) tempfile(ctx Context, name string) (f *file) {
    var t, d = p.tempdir(ctx)
    switch d {
    case "", "/":
        erro(ctx, "%v: %s: tempdir is illegal: %v → '%s', %s", p.name, name, t, __string(ctx, t), d)
        note(ctx, "%v", p.resolveDef(ctx, "outtmp"))
        note(ctx, "%v", p.resolveDef(ctx, "target.tmp"))
        note(ctx, "%v", p.resolveDef(ctx, "target.out"))
        note(ctx, "%v", p.resolveDef(ctx, "target.triple"))
        note(ctx, "%v", p.resolveDef(ctx, "rel.remnant"))
        note(ctx, "%v", p.resolveDef(ctx, "rel.chop"))
        note(ctx, "%v", p.resolveDef(ctx, "variant.tag")).trace()
    }

    if f = _stat(ctx, name, stat_dir{d}, stat_nonexist{true}); f == nil {
        erro(ctx, "%v: not a file: %v : %v", p, name, d).trace()
    }

    if checkpoints { tempfile_check(ctx, p, name, d, f) }
    return
}

func (p *project) configuration_sm(ctx Context) (f *file) {
    if f = p.tempfile(ctx, configuration_sm); f == nil {
        erro(ctx, "%v: no file %s", p, configuration_sm).trace()
    }
    if checkpoints {
        p.configuration_sm_check(ctx, f)
    }
    return
}

func project_entry(c Context, s any, a ...bool) entry { return _project(c).entry(c, s, a...) }
func project_resolve(c Context, s string) object { return _project(c).resolve(c, s) }

func (p *project) resolveDef(ctx Context, name string) (res *def) {
    if o := p.resolve(ctx, name); o != nil { res, _ = o.(*def) }
    return
}

func (p *project) resolve(ctx Context, name string) (obj object) {
    if _, obj = p.find(name); obj != nil { return }

    if p.ext.Plugin != nil {
        if sym, e := p.ext.Lookup(name); e == nil && sym != nil {
            erro(ctx, "TODO: convert ext symbol: %v: %s", name, typeof(sym)).trace()
        }
    }

    for _, base := range p.bases {
        if base.has_base(p) {
            erro(ctx, "recursive derivation: %v ⇔ %v", ts(p), ts(base)).trace()
        }
        if obj = base.resolve(ctx, name); obj != nil { return }
    }

    if p.configure != nil && p.configure != p {
        return p.configure.resolve(ctx, name)
    }
    return
}

func (p *project) _entries(ctx Context, name any, _b ...bool) (entries []entry) {
    entries = unmap_entries(ctx, p, name, nil)

    if false && p.configure != nil && is_configurecontext(ctx) {
        entries = append(entries, p.configure._entries(ctx, name, true)...)
    }

    var alwaysResolveBases bool
    if n := len(_b); n > 0 { alwaysResolveBases = _b[n-1] }

    if alwaysResolveBases || entries == nil {
        for _, base := range p.bases {
            if t := base._entries(ctx, name, alwaysResolveBases); t != nil {
                entries = append(entries, t...)
                break
            }
        }
    }

    if false && entries == nil { // NOTE: this would be SLOW
        for _, u := range p.use.list {
            t := u.project._entries(ctx, name, alwaysResolveBases)
            if t != nil { entries = append(entries, t...); break }
        }
    }
    return
}

func (p *project) entry(c Context, name any, a ...bool) (_ entry) {
    var entries = p._entries(c, name, a...)
    if n := len(entries); 0 < n {
        if 1 < n { erro(c, "%v : %d entries", name, n).trace() }
        return entries[0]
    }
    return
}

func (p *project) resolvePatterns(ctx Context, v Value, s string) (res []*stemmed_rule) {
    var t1, t2 time.Time

    defer func(t0 time.Time) {
        var t = time.Now()
        if d := t.Sub(t0); d > 1*time.Second {
            var ( d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) ; d3 = t.Sub(t2) ; n int )
            var a = auto_get(ctx, "@")
            for sc := _stemmed(ctx); sc != nil; n += 1 {
                if c := inner(sc); c != nil { sc = _stemmed(c) } else { break }
            }

            var pos = _position(ctx)
            prompt(ctx, "%v: slow: %v: %v, %v %v %v", pos, p, d, d1, d2, d3)
            prompt(ctx, "%v: slow: %v: %v: %v %v, %d nests", pos, p, a, v, p.patterns, n).debug(4)

            for _, pat := range p.patterns {
                var pt = pat.target
                var pa = pat.arged
                var full, r, stems = match(ctx, pt, s)
                var m = joinp(ctx, r)
                prompt(ctx, "%v: slow: %v%v: %v: %v %v %v, %v ; %v", pos, pt, pa, s, full, r, stems, m)
            }
            warnstack(ctx, 3).debug(6)
        }
    } (time.Now())

    if res, t1, t2 = p.resolvePatterns123(ctx, v, s); false && len(res) > 0 {
        for _, t := range res {
            if f, _ := to_file(t.target); f != nil {
                f.position = t.Position()
            } else if f = p.file(ctx, s); f != nil {
                f.position = t.Position()
                t.target = f
            }
        }
    }
    return
}

func (p *project) resolvePatterns123(ctx Context, v Value, s string) (res []*stemmed_rule, t1, t2 time.Time) {
    if true  { res = append(res, p.resolvePatterns1(ctx, v, s)...) } ; t1 = time.Now()
    if true  { res = append(res, p.resolvePatterns2(ctx, v, s)...) } ; t2 = time.Now()
    if false { res = append(res, p.resolvePatterns3(ctx, v, s)...)/* heavy work, VERY SLOW! */ }
    return
}

func (p *project) resolvePatterns1(ctx Context, val Value, s string) (res []*stemmed_rule) {
    defer func(t0 time.Time) {
        if d := time.Now().Sub(t0); d > 1*time.Second {
            prompt(ctx, "%v: slow: %v %v", _position(ctx), val, d).debug()
        }
    } (time.Now())

ForPatterns:
    for _, pat := range p.patterns {
        if full, r, stems := match(ctx, pat.target, s); full {
            var m = joinp(ctx, r)

            if true {
                for sc := _stemmed(ctx); sc != nil; { // pattern loop detection
                    if s := __string(ctx, sc.target); s == m { continue ForPatterns }
                    if c := inner(sc); c != nil { sc = _stemmed(c) } else { break }
                }
            }

            if pa := pat.arged; len(pa) > 0 {
                var y bool
                var t1 = time.Now()
                var av = xmerge(ctx, pa...)
                var t2 = time.Now()
                for _, a := range av { if y, _, _ = match(ctx, a, s); y { break } }

                var t3 = time.Now()
                if d := t3.Sub(t1); d > 1*time.Second {
                    var ( d2 = t2.Sub(t1) ; d3 = t3.Sub(t2) )
                    var ( p = _position(ctx) ; pt = pat.target )
                    prompt(ctx, "%v: slow: %v, %v→%d; %v⇒%v+%v", p, pt, pa, len(av), d, d2, d3).debug()
                }

                if !y { continue ForPatterns }
            }

            res = append(res, &stemmed_rule{pat, val, stems})
        }
    }
    return
}

func (p *project) resolvePatterns2(ctx Context, val Value, s string) (res []*stemmed_rule) {
    for _, base := range p.bases {
        var a, _, _ = base.resolvePatterns123(ctx, val, s)
        res = append(res, a...)
    }
    if p.configure != nil && is_configurecontext(ctx) {
        var a, _, _ = p.configure.resolvePatterns123(ctx, val, s)
        res = append(res, a...)
    }
    return
}

func (p *project) resolvePatterns3(ctx Context, val Value, s string) (res []*stemmed_rule) {
    for _, use := range p.use.list {
        var a, _, _ = use.project.resolvePatterns123(ctx, val, s)
        res = append(res, a...)
    }
    return
}

func (p *project) family() (res []*project) {
    res = append(res, p)
    for _, base := range p.bases {
        res = append(res, base.family()...)
    }
    return
}

func (p *project) _isa(s string) (_ bool) {
    for _, base := range p.bases {
        if base.name == s || base._isa(s) { return true }
    }
    return
}

func (p *project) isa(proj *project) (_ bool) {
    for _, base := range p.bases {
        if base == proj || base.isa(proj) { return true }
    }
    return
}

func (p *project) has_base(proj *project) (_ bool) {
    for _, base := range p.bases {
        if base == proj || base.has_base(proj) { return true }
    }
    return
}

func (p *project) has_loaded(ctx Context, proj *project, traveUseLoop bool) (rp *project, res, isb bool) {
    if u := _universe(ctx) ; u.checkLoadGraph || !u.fastMode {
        rp, res, isb, _ = p.has_loaded_recur(ctx, p, proj, 1, traveUseLoop)
    }
    return
}

func (p *project) has_loaded_recur(ctx Context, top, proj *project, depth int, traveUseLoop bool) (rp *project, res, isb bool, err error) {
    if depth > 1 && top == p && true {
        err = fmt.Errorf("loop '%v' (depth=%d)", p.loop_load_path(), depth)
        erro(ctx, "%v: %v", p, err).trace()
    } else if depth > 128 {
        err = fmt.Errorf("exceeds maximum base depth (%d) (start=%v, target=%v)", depth, top, proj)
        erro(ctx, "%v: %v", p, err)
        erro(ctx, "start: %v", top)
        erro(ctx, "target: %v", proj).trace()
    }
    for _, base := range p.bases {
        if isb = base == proj; isb { return }
        if rp, res, isb, err = base.has_loaded_recur(ctx, top, proj, depth+1, traveUseLoop); err != nil {
            return
        } else if res || isb { rp = base ; return }
    }
    for _, use := range /*p.loads*/p.use.list {
        var imp = use.project
        if imp == top && !traveUseLoop {
            s := top.loop_load_path()
            err = fmt.Errorf("loop `%v`", s)
            erro(ctx, "start: %v", top)
            erro(ctx, "stop: %v", proj)
            erro(ctx, "%v: %v", p, err).trace()
        }
        if res = imp == proj; res { rp = imp; return }
        if rp, res, res, err = imp.has_loaded_recur(ctx, top, proj, depth+1, traveUseLoop); err != nil {
            return
        } else if res { rp = imp; return }
    }
    rp = p
    return
}

func (p *project) loop_base_path(ctx Context, _p *project, s string) (_ string) {
    if s == "" { s = p.name }
    for _, base := range p.bases {
        if t := s + " → " + base.name; base == _p {
            return t
        } else if t = base.loop_base_path(ctx, _p, t); t != "" {
            return t
        }
    }
    return
}

func (p *project) loop_load_path() (s string) { return p.loop_load_recur(p) }
func (p *project) loop_load_recur(top *project) (s string) {
    for _, use := range /*p.loads*/p.use.list {
        var imp = use.project
        if imp == top {
            if p != top { s = "⇢" }
            s += fmt.Sprintf("(%s)⇢(%s)", p.spec, imp.spec)
            break
        }
        if t := imp.loop_load_recur(top); t != "" {
            if p != top { s = "⇢" }
            s += fmt.Sprintf("(%s)%s", p.spec, t)
            break
        }
    }
    return
}

func (p *project) isUsing(usee *project) (res bool) {
    for _, use := range p.use.list {
        if res = use.project == usee; res { break  }
        if res = use.project.isUsing(usee); res { break }
    }
    return
}

func (p *project) isUsingDirectly(proj *project) (res bool) {
    for _, u := range p.use.list {
        if res = u.project == proj; res { break }
    }
    return
}

func (p *project) usees(bases, basesRecur, useeRecur, pre bool) (res []*project) {
    if p.opt.traveUseLoop { return }
    if bases {
        for _, base := range p.bases {
            res = append(res, base.usees(basesRecur, basesRecur, useeRecur, pre)...)
        }
    }
    for _, u := range p.use.list {
        if pre { res = append(res, u.project) }
        if useeRecur {
            for _, u := range u.project.usees(bases && basesRecur, basesRecur, true, pre) {
                if !p.isUsingDirectly(u) { res = append(res, u) }
            }
        }
        if !pre { res = append(res, u.project) }
    }
    return
}

// Note: this is okay not using an atomic value, because
// chdirMutex can serve to protect the whole timeframe.
var chdirMutex = new(sync.Mutex)

func lockCD(dir string, dura time.Duration) error {
    // Protect the work directory, `chdirMutex` ensures that
    // there's only one timer being counting to avoid work
    // directory being changed before the deadline.
    chdirMutex.Lock()
    go func() {
        if dura > 0 { time.Sleep(dura) }
        chdirMutex.Unlock()
    } ()
    return os.Chdir(dir)
}
