//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package smart

import (
    "path/filepath"
    "runtime/pprof"
    "runtime"
    "reflect"
    "strings"
    "sync"
    "time"
    "fmt"
    "os"
)

const test_path = "foo/xx/yy/zzzz"

var baseWorkDir, _ = os.Getwd()
var launchTime = time.Now()
var searchPaths searchlist

type char rune
func (c char) String() string { return string(rune(c)) }

type searchlist []string
func (sl *searchlist) String() string { return fmt.Sprint(*sl) }
func (sl *searchlist) Set(value string) error {
    *sl = append(*sl, strings.Split(value, ",")...)
    return nil
}

type cache struct { Context } // versus `unmap`
func (c cache) cast(t reflect.Type) Context { return implcast(c, t) }
func (c cache) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actHitValue:
        return _valcache_bool(c.value(ctx, t.valcache, t.Value))
    case actHitPunc:
        return _valcache_bool(c.punc(ctx, t.valcache, t.token))
    case actHitWord:
        return _valcache_bool(c.word(ctx, t.valcache, t.string))
    case actHitGlob:
        return _valcache_bool(c.hit_glob(ctx, t.valcache, t.string))
    case actHitPerc:
        return _valcache_bool(c.hit_perc(ctx, t.valcache, t.string))
    case actHitRege:
        return _valcache_bool(c.hit_rege(ctx, t.valcache, t.string))
    default:
        return c.Context.do(ctx, op)
    }
}

type unmap struct { Context ; a []matched_filemap } // versus `cache`
func (u *unmap) cast(t reflect.Type) Context { return implcast(u, t) }
func (u *unmap) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actHitPunc:
        return _valcache_bool(u.punc(ctx, t.valcache, t.token))
    case actHitWord:
        return _valcache_bool(u.word(ctx, t.valcache, t.string))
    case actHitGlob:
        return _valcache_bool(u.hit_glob(ctx, t.valcache, t.string))
    case actHitPerc:
        return _valcache_bool(u.hit_perc(ctx, t.valcache, t.string))
    case actHitRege:
        return _valcache_bool(u.hit_rege(ctx, t.valcache, t.string))
    case actUnglob:
        return _valcache_bool(u.glob(ctx, t.valcache, t.string))
    case actUnperc:
        // return _valcache_bool(nil, false)
    case actUnrege:
        // return _valcache_bool(nil, false)
    case actUnpat:
        return _valcache_bool(u.pat(ctx, t.valcache, t.string))
    case matched_filemap:
        u.a = append(u.a, t)
    case property:
        if t&(propUnmap) != 0 { return true }
    }
    return u.Context.do(ctx, op)
}

type unmap_bare struct { Context ; *valcache ; s string ; solo bool ; i int }
func (p *unmap_bare) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *unmap_bare) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnglob:
        x := _valcache_bool(p.glob(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    case property:
        if p.solo && t&(propPath) != 0 { return p.s }
    }
    return p.Context.do(ctx, op)
}

type unmap_path struct { Context ; p *path ; i int }
func (p *unmap_path) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *unmap_path) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnglob:
        x := _valcache_bool(p.glob(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    case property:
        if t&(propPath) != 0 { return p.p }
    }
    return p.Context.do(ctx, op)
}

type unmap_pstr struct { Context ; s string ; ss []string ; i int }
func (p *unmap_pstr) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *unmap_pstr) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnglob:
        x := _valcache_bool(p.glob(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    case property:
        if t&(propPath) != 0 { return p.s }
    }
    return p.Context.do(ctx, op)
}

func cacheMapping(ctx Context) bool { return !cacheUnmap(ctx) }
func cacheUnmap(ctx Context) bool { t, _ := do(ctx, propUnmap).(bool); return t }

func _valcache_bool(x *valcache, y bool) valcache_bool { return valcache_bool{x,y} }

type matched_filemap struct { filemap ; name string }
type filemap_slot struct { *_filemap ; Value }
func (s filemap_slot) String() string { return s.Value.String() }
func (s filemap_slot) filemap() filemap { return filemap{s._filemap,s.Value} }
func (s filemap_slot) matched_filemap(a string) (t matched_filemap) {
    t._filemap, t.pattern, t.name = s._filemap, s.Value, a
    return
}

type anycache interface { *valcache | *_DEPRECATED_vcache }

type valcacheable interface { match(Context, any) (bool, any, []string) }
type valcache_bool struct { *valcache ; bool }
type valcache_val struct { *valcache ; Value }
type valcache struct {
    a []valcacheable
    v []valcache_val
    o [][]string // priority of globs/percs/reges

    // NOTE: using `map[interface{}]*valcache` is easier but lose some performance
    puncs map[token ]*valcache
    words map[string]*valcache
    globs map[string]*valcache
    percs map[string]*valcache
    reges map[string]*valcache
}

func (p *valcache) String() (s string) { // NOTE: for debug
    for k, v := range p.a     { s += fmt.Sprintf("%v:%v,", k, v) }
    for k, v := range p.puncs { s += fmt.Sprintf("%v:%v,", k, v) }
    for k, v := range p.words { s += fmt.Sprintf("%v:%v,", k, v) }
    for i, m := range []map[string]*valcache{ p.globs, p.percs, p.reges } {
        if m != nil { for _, k := range p.o[i] { s += fmt.Sprintf("%v:%v,", k, m[k]) } }
    }
    if s != "" { s = s[:len(s)-1] } // aka strings.TrimSuffix(s, ",")
    return "{" + s + "}"
}

func (p *valcache) _o(s string, i int) {
    if len(p.o) < i+1 {
        p.o = append(p.o, []string{s})
    } else {
        p.o[i] = append(p.o[i], s)
    }
}

func (p *valcache) glob_o(s string) { p._o(s, 0) }
func (p *valcache) perc_o(s string) { p._o(s, 1) }
func (p *valcache) rege_o(s string) { p._o(s, 2) }

func (p *valcache) fullmatch(ctx Context, k any) (_y bool) {
    for _, a := range p.a {
        if _y, _, _ = a.match(ctx, k); _y {
            switch t := a.(type) {
            case filemap_slot: do(ctx, t.matched_filemap(_path(ctx, k)))
            }
            return
        }
    }
    return
}

func (p *valcache) hit(ctx Context, k interface{}) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v %v", k, p)
        note(ctx, "%5v %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    ctx = at(ctx, k)

    defer trace(ctx)

    switch t := k.(type) {
    case string:
        if x, y := do(ctx, actHitWord{p, t}).(valcache_bool); y {
            return x.valcache, x.bool
        } else {
            erro(ctx, "unhit: %v %v", us(k), us(ctx)).debug()
            return
        }
    case token:
        if x, y := do(ctx, actHitPunc{p, t}).(valcache_bool); y {
            return x.valcache, x.bool
        } else {
            erro(ctx, "unhit: %v %v", us(k), us(ctx)).debug()
            return
        }
    case interface{ hit(Context, *valcache) (*valcache, bool) }:
        if res, donePat = t.hit(ctx, p) ; res == nil && cacheMapping(ctx) {
            erro(ctx, "no valcache for %v : %v", us(k), p).debug()
            return
        } else {
            return
        }
    case Value:
        if indeterminate(ctx, t) {
            if x, y := do(ctx, actHitValue{p, t}).(valcache_bool); y {
                return x.valcache, x.bool
            } else {
                erro(ctx, "unhit: %v %v", us(k), us(ctx)).debug()
                return
            }
        } else {
            erro(ctx, "non-valcacheable value %v : %v", us(k), p).debug()
            return
        }
    default:
        erro(ctx, "non-valcacheable %v : %v", us(k), p).debug()
        return
    }
}

func (cache) value(ctx Context, p *valcache, v Value) (res *valcache, donePat bool) {
    for _, t := range p.v {
        if equal(ctx, v, t.Value) {
            res = t.valcache
            return
        }
    }

    res = &valcache{}
    p.v = append(p.v, valcache_val{res, v})
    return
}
func (cache) punc(ctx Context, p *valcache, t token) (res *valcache, donePat bool) {
    if  p.puncs == nil {   res = &valcache{}
        p.puncs = make(map[token]*valcache)
        p.puncs[t] = res
        return
    } else if x, y := p.puncs[t]; !y || x == nil {
        res = &valcache{}
        p.puncs[t] = res
        return
    } else {
        res = x
        return
    }
}
func (cache) word(ctx Context, p *valcache, s string) (res *valcache, donePat bool) {
    if  p.words == nil {    res = &valcache{}
        p.words = make(map[string]*valcache)
        p.words[s] = res
        return
    } else if x, y := p.words[s]; !y || x == nil {
        res = &valcache{}
        p.words[s] = res
        return
    } else {
        res = x
        return
    }
}
func (cache) hit_glob(ctx Context, c *valcache, s string) (res *valcache, donePat bool) {
    if  c.globs == nil {    res = &valcache{}
        c.globs = make(map[string]*valcache)
        c.globs[s] = res
        c.glob_o(s)
        return
    } else if x, y := c.globs[s]; !y || res == nil {
        res = &valcache{}
        c.globs[s] = res
        c.glob_o(s)
        return
    } else {
        res = x
        return
    }
}
func (cache) hit_perc(ctx Context, c *valcache, s string) (res *valcache, donePat bool) {
    if  c.percs == nil {    res = &valcache{}
        c.percs = make(map[string]*valcache)
        c.percs[s] = res
        c.perc_o(s)
        return
    } else if x, y := c.percs[s]; !y || res == nil {
        res = &valcache{}
        c.percs[s] = res
        c.perc_o(s)
        return
    } else {
        res = x
        return
    }
}
func (cache) hit_rege(ctx Context, c *valcache, s string) (res *valcache, donePat bool) {
    if  c.reges == nil {    res = &valcache{}
        c.reges = make(map[string]*valcache)
        c.reges[s] = res
        c.rege_o(s)
        return
    } else if x, y := c.reges[s]; !y || res == nil {
        res = &valcache{}
        c.reges[s] = res
        c.rege_o(s)
        return
    } else {
        res = x
        return
    }
}
func (u unmap) value(ctx Context, p *valcache, v Value) (res *valcache, donePat bool) {
    for _, t := range p.v {
        if equal(ctx, v, t.Value) {
            res = t.valcache
            return
        }
    }
    return
}
func (u unmap) punc(ctx Context, c *valcache, t token) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v : %v", t, c)
        note(ctx, "%5v : %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if nil != c.puncs {
        if x, y := c.puncs[t]; y {
            return x, false
        }
    }
    return u.pat(ctx, c, t.String())
}
func (u unmap) word(ctx Context, c *valcache, s string) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v : %v", s, c)
        note(ctx, "%5v : %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if nil != c.words {
        if x, y := c.words[s]; y {
            return x, false
        }
    }
    return u.pat(ctx, c, s)
}
func (u unmap) hit_glob(ctx Context, c *valcache, s string) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v : %v", s, c)
        note(ctx, "%5v : %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if nil != c.globs {
        if x, y := c.globs[s]; y {
            return x, false
        }
    }
    return //u.pat(ctx, c, s)
}
func (u unmap) hit_perc(ctx Context, c *valcache, s string) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v : %v", s, c)
        note(ctx, "%5v : %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if nil != c.percs {
        if x, y := c.percs[s]; y {
            return x, false
        }
    }
    return //u.pat(ctx, c, s)
}
func (u unmap) hit_rege(ctx Context, c *valcache, s string) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v : %v", s, c)
        note(ctx, "%5v : %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if nil != c.reges {
        if x, y := c.reges[s]; y {
            return x, false
        }
    }
    return //u.pat(ctx, c, s)
}
func (unmap) pat(ctx Context, c *valcache, k string) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v %v", k, c)
        note(ctx, "%5v %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if x, y := do(ctx, actUnglob{c, k}).(valcache_bool); y {
        if res, donePat = x.valcache, x.bool ; x.bool { return }
    }
    if x, y := do(ctx, actUnperc{c, k}).(valcache_bool); y || res == nil {
        if res, donePat = x.valcache, x.bool ; x.bool { return }
    }
    if x, y := do(ctx, actUnrege{c, k}).(valcache_bool); y || res == nil {
        if res, donePat = x.valcache, x.bool ; x.bool { return }
    }
    return
}
func (unmap) glob(ctx Context, _c *valcache, s string) (res *valcache, donePat bool) {
    if false && s == test_path { defer func() {
        note(ctx, "%5v %v", s, _c)
        note(ctx, "%5v %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    if len(_c.o) == 0 { return }
    if strings.Contains(s, pathSep) { return }

    defer trace(ctx)

    for _, pat := range _c.o[0] {
        var c, _ = _c.globs[pat]
        if c == nil {
            erro(ctx, "%v %v - nil glob", s, pat).debug()
            continue
        }
        if f, _, _ := globMatch(ctx, pat, s); f {
            return c, c.fullmatch(ctx, s)
        }
    }
    return
}
func (p *unmap_bare) glob(ctx Context, _c *valcache, s string) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        var pat string
        if 0 < len(p.o) && p.i < len(p.o[0]) {
            pat = fmt.Sprintf("%d. %v", p.i, p.o[0][p.i])
        }
        note(ctx, "%5v %v %v", p.s, p.valcache, pat)
        note(ctx, "%5v %v %v", s, _c, do(ctx, propPath)); t := ""; if donePat { t = "F" }
        note(ctx, "%5v %v %v %v", t, res, _c == p.valcache, cast[*unmap](ctx).a)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    defer trace(ctx)

    if false && 0 < len(_c.o) && _c != p.valcache {
        for _, pat := range _c.o[0] {
            var c, _ = _c.globs[pat]
            if c == nil {
                erro(ctx, "%v %v - nil glob", s, pat).debug()
                return
            }
            if f, _, _ := globMatch(ctx, pat, s); f {
                if c.fullmatch(ctx, s) {
                    return c, true
                } else {
                    res = c
                    break
                }
            }
        }
    }

    if 0 < len(p.o) {
        var t = do(ctx, propPath)

        for _, pat := range p.o[0][p.i:] {
            var c, _ = p.globs[pat]
            if c == nil {
                erro(ctx, "%v %v - nil glob", s, pat).debug()
                return
            }

            p.i += 1

            if f, _, _ := globMatch(ctx, pat, p.s); f {
                if t == nil {
                    return c, c.fullmatch(ctx, p.s)
                } else {
                    return c, c.fullmatch(ctx, t)
                }
            }

            if p.solo || p.s == s { continue }

            if f, _, _ := globMatch(ctx, pat, s) ; f {
                if t == nil {
                    return c, c.fullmatch(ctx, s)
                } else {
                    return c, c.fullmatch(ctx, t)
                }
            }

            if t != nil {
                if f := c.fullmatch(ctx, t); f {
                    return c, true
                }
            }
        }
    }
    return
}
func (pc *unmap_path) glob(ctx Context, _c *valcache, k string) (res *valcache, donePat bool) {
    if false && pc.p.string(ctx) == test_path { defer func() {
        note(ctx, "%5v %v", k, _c)
        note(ctx, "%5v %v", donePat, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    if len(_c.o) < 1 { return }

    defer trace(ctx)

    for _, pat := range _c.o[0] {
        var c = _c.globs[pat]

        if checkpoints {
            if strings.Contains(pat, "/") {
                erro(ctx, "%v %v %v", k, pat, c).debug()
                return
            }
        }

        var s string
        var d = strings.Contains(pat, "**")
        if false && d {
            s = pc.p.string(ctx)
        } else if pc.i + 1 == pc.p.len() {
            s = pc.p.elems[pc.i].string(ctx)
        } else {
            s = (&path{elements{pc.p.elems[pc.i:]}}).string(ctx)
        }

        if y, _, _ := globMatch(ctx, pat, s); y {
            return c, true
        }
    }
    return
}
func (pc *unmap_pstr) glob(ctx Context, _c *valcache, k string) (res *valcache, donePat bool) {
    if false && do(ctx, propPath) == test_path { defer func() {
        note(ctx, "%5v %v", k, _c)
        note(ctx, "%5v %v ; %d/%d %v", donePat, res, pc.i, len(pc.ss), pc.ss[pc.i])
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    if len(_c.o) < 1 { return }

    defer trace(ctx)

    var num = len(pc.ss)

    for _, pat := range _c.o[0] {
        var c = _c.globs[pat]

        if checkpoints {
            if strings.Contains(pat, "/") {
                erro(ctx, "%v %v %v", k, pat, c).debug()
            }
        }

        var s string
        if strings.Contains(pat, "**") {
            s = pc.s
        } else {
            s = pc.ss[pc.i] // aka k
        }

        if y, _, _ := globMatch(ctx, pat, s) ; y {
            if c.fullmatch(ctx, pc.ss) {
                return c, true
            }
            if pc.i + 1 < num {
                return c, false
            }
        }
    }
    return
}

func (unmap) perc(ctx Context, _c *valcache, s string) (res *valcache, donePat bool) {
    if len(_c.o) < 2 { return }
    for _, pat := range _c.o[1] {
        note(ctx, "TODO: %v %v", us(s), pat).debug(3)
    }
    return
}
func (unmap) regx(ctx Context, _c *valcache, s string) (res *valcache, donePat bool) {
    if len(_c.o) < 3 { return }
    for _, pat := range _c.o[2] {
        note(ctx, "TODO: %v %v", us(s), pat).debug(3)
    }
    return
}

func (u *universe) filemap(ctx Context, p *project, patts, paths []Value) (res []filemap) {
    defer trace(ctx)

    var base = &_filemap{p, patts, paths}

    for _, patt := range patts {
        if patt == nil {
            errostack(ctx, 5, "nil pattern ; paths=%v", paths).debug()
        } else if c, _ := u.filemaps.hit(cache{ctx}, patt); c != nil {
            c.a = append(c.a, filemap_slot{base, patt})
            res = append(res, filemap{base, patt})
        }
    }
    return
}

func (u *universe) unmap_bare(ctx Context, c *valcache, s string, solo bool) (Context, *valcache, bool) {
    var y bool
    var cc Context = &unmap_bare{ctx, c, s, solo, 0}
    if ss := strings.Split(s, DOT.String()) ; len(ss) == 1 {
        if c, y = c.hit(cc, s) ; true && y && cast[*unmap](ctx).a == nil {
            note(ctx, "%v %v", s, c).debug(1)
        }
    } else {
        for j, t := range ss {
            if c != nil && 0 < j {
                if c, y = c.hit(cc, DOT) ; y || c == nil { break }
            }
            if c != nil && t != "" {
                if c, y = c.hit(cc, t) ; y || c == nil { break }
            }
        }
    }
    return cc, c, y
}

func (u *universe) unmap_pstr(ctx Context, c *valcache, s string, ss []string) *valcache {
    var x, y = unmap_pstr{ctx, s, ss, 0}, false
    for cc := Context(&x) ; c != nil && x.i < len(ss) ; x.i += 1 {
        if cc, c, y = u.unmap_bare(cc, c, ss[x.i], false); y {
            if false && cast[*unmap](ctx).a == nil {
                note(ctx, "%v %v %v", s, ss[x.i], c).debug(1)
            }
            break
        }
    }
    return c
}

func (u *universe) unmap(ctx Context, key interface{}) (_ []matched_filemap) {
    defer trace(ctx)

    var c = &u.filemaps

    if x, y := key.(Value); y && x.patterned(ctx) {
        erro(at(ctx,x), "TODO: %v : %v", x, c).debug()
        return
    }

    var um = &unmap{ctx, nil}

    if s, y := key.(string); y {
        if ss := strings.Split(s, pathSep); len(ss) == 1 {
            if _, c, y = u.unmap_bare(um, c, s, true); y && um.a == nil {
                note(ctx, "%v %v", us(key), c).debug(1)
            }
        } else if y {
            c = u.unmap_pstr(um, c, s, ss)
        }
    } else {
        if c, y = c.hit(um, key); y && um.a == nil {
            note(ctx, "%v %v", us(key), c).debug(1)
        }
    }

    return um.a
}

func unmap_files(ctx Context, key interface{}) []matched_filemap {
    return _universe(ctx).unmap(ctx, key)
}

type hooks struct {
    assert func(Context, Value, bool) bool
    debug func(Context, string, []Value)
}

type benchmark struct {
    benchmark_builtin_expand func(*builtin, Context, time.Time, reflect.Value)
}

type packagetype uint8

const (
    packageUnknown packagetype = iota
    packageSmart  // smart package
    packageConfig // pkgconfig
)

type packageinfo struct {
    *project
    t packagetype // smart, pkgconfig, cmake, etc.
}

func _universe(c Context) *universe { return cast[*universe](c) }

type universe struct {
    diagnostic
    commandline

    benchmark

    hooks hooks

    workdir string
    prefix  string // FIXME: prefix for distribution

    scope   *Scope
    globe   *globe

    fset    *FileSet
    paths   searchlist

    packages map[string]packageinfo

    statmutex sync.Mutex
    statcache map[string]*filebase // File.fullname() -> File
    filemaps valcache // value -> dirs

    expand_n int32

    ddd string // debug parsing via `eval -ddd=example`, also project.dd
}
func (ctx *universe) loader() *loader { return ctx.globe.top }
func (ctx *universe) Globe() *globe { return ctx.globe }
func (ctx *universe) Scope() *Scope { return ctx.scope }
func (ctx *universe) String() string { return "universe" }
func (ctx *universe) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}
func (ctx *universe) Position() (p Position) {
    if ctx.globe == nil || ctx.globe.main == nil {
        p.Filename, p.Line, p.Column = _workdir(ctx), 0, 0
    } else {
        p = ctx.globe.main.position
    }
    return
}
func (ctx *universe) project() (p *project) {
    if ctx != nil && ctx.globe != nil { p = ctx.globe.main }
    return
}
func (ctx *universe) projects(_ Context, projs ...*project) []*project {
    if len(projs) > 0 { panic(_failure(ctx, "%v", projs)) }
    return nil
}
func (ctx *universe) closure() (scopes []*Scope) {
    if m := ctx.globe.main; m != nil && m.scope != nil {
        if false { scopes = append(scopes, m.scope) }
    }
    return
}
func (ctx *universe) file(filename string, src []byte) *TokFile {
    return ctx.fset.AddFile(filename, -1, len(src))
}
func (ctx *universe) do(_ctx Context, op any) (res any) {
    switch t := op.(type) {
    case actOnErros:
        if ctx.panicFailureOnErrosFlushed {
            if 0 < t.i { panic(_failure(ctx, "got %d errors", t.i)) }
            res = true
        }
        return

    case property:
        if t&propPosition != 0 { return ctx.Position() }
        if t&propWorkDir != 0 {
            if s := ctx.workdir ; s == "" {
                return baseWorkDir
            } else {
                return s
            }
        }
    }
    return ctx.diagnostic.do(_ctx, op)
}

type commandline struct {
  help            bool `h,help`

  debug           bool `d,db,debug`
  debugErrors     bool `de,dberro,debug-errors`
  debugWarns      bool `dw,dbwarn,debug-warns`
  debugInfos      bool `di,dbinfo,debug-infos`
  debugPrompt     bool `dp,dbprom,debug-prompt`
  debugSyntax []string `ds,dbsyntax,debug-syntax`

  autoProfs       bool `ap,autoprof,auto-profiles,auto-profile`
  cpuProf         string `cpuprof,cpu-profile`
  memProf         string `memprof,memory-profile`

  printConfig     bool `opts,print-options,printoptions`
  printFlags      bool `flags,print-flags,printflags`

  buildPlugins    bool `bp,bup,build-plugins,buildplugins`

  silentOptionalSelection bool

  verbose         bool `v,verb,verbose`
  verboseBreaks   bool `vb,vbrk,verbose-breaks`
  verboseChecks   bool `vc,vchk,verbose-checks`
  verboseImport   bool `vi,vimp,verbose-import`
  verboseLoads    bool `vl,vloa,verbose-loading`
  verboseParse    bool `vp,vpar,verbose-parsing`
  verboseUsing    bool `vu,vuse,verbose-using`
  verboseExecFlags bool `vxf,verbose-exec-flag`

  allowClosureFilemap bool `cf,closure-filemap,closure-files`

  cleanDotCache   bool `clcac,clean-cache,clear-cache;rmc,rm-cache`
  cleanDotDeps    bool `cldep,clean-deps,clear-deps;rmd,rm-deps`
  cleanDotGrep    bool `clgrp,clean-grep,clear-grep;rmg,rm-grep`
  cleanTmpDirs    bool `cltmp,clean-temp,clear-temp;rmt,rm-temp`

  checkLoadGraph  bool `ckld,check-loads`

  configure       bool `c,con,conf,configure`               // optionConfigure
  reconfigure     bool `rc,rec,reconf,reconfig,reconfigure` // optionReconfig

  saveGrepSource  bool `savgs,save-grep-source`

  noRun           bool `nor,no-run`
  noExec          bool `nox,ne,no-exec,no-execute`  // optionNoExec
  noDeps          bool `nod,no-deps`
  noGrep          bool `nog,no-grep`
  noDepsGrep      bool `nodg,ngd,no-deps-grep,no-grep-deps`
  noImportFiles   bool `noif,no-import-files`

  parallel        bool `p,par,para,parallel`

  fastMode        bool `f,fm,fast,fast-mode`
  panicFailureOnErrosFlushed    bool `fe,foe,fail-on-errors`
  errorUncache    bool `eu,error-uncache,error-no-cache`

  traceLaunch     bool `tl,trace-launch`
  traceParsing    bool `tp,trace-parse`
  traceExecutor   bool `te,trace-executor`
  traceExec       bool `tx,trace-exec`
  traceEntering   bool `ti,trace-entering`
  traceConfig     bool `tc,trace-config`

  slow time.Duration `sl,slow` // time.Millisecond
}

func _commandline() commandline { return commandline{
    debugPrompt: true,
    debugErrors: true,
    debugWarns:  true,
    debugInfos:  true,

    fastMode: true,
    parallel: false, // FIXME: program.traverse not working in parallel

    panicFailureOnErrosFlushed: true,
    silentOptionalSelection: false,

    slow: 9990 * time.Millisecond,
}}

func new_universe(ii ...interface{}) (ctx *universe) {
    var p positional
    ctx = &universe{}
    ctx.Context = &p

    if s, e := os.Getwd(); e != nil {
        erro(ctx, "%v", e).debug(6)
        return
    } else {
        p.position.Filename = s
        ctx.paths = searchPaths
        ctx.workdir = s
        ctx.fset = NewFileSet()
        ctx.statcache = make(map[string]*filebase)
        ctx.scope = newScope(ctx.Position(), nil, nil, `universe`)
    }

    var cl = true
    for _, i := range ii {
        switch t := i.(type) {
        case  commandline: ctx.commandline, cl =  t, false
        case *commandline: ctx.commandline, cl = *t, false
        case *hooks: ctx.hooks = *t
        case  hooks: ctx.hooks =  t
        }
    }
    if cl { ctx.commandline = _commandline() }
    if true { ctx.benchmark_builtin_expand = (*builtin).benchmark_expand }

    var bin  = ease(ctx, os.Args[0])
    var args = ease(ctx, os.Args[1:])
    _, _ = ctx.scope.define(ctx, defVoid, "SMART.ARGS", args)
    _, _ = ctx.scope.define(ctx, defVoid, "SMART.BIN", bin)
    _, _ = ctx.scope.define(ctx, defVoid, "SMART", bin)

    for name, f := range builtins {
        if _, alt := ctx.scope.builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
    }

    var os Value // one of darwin, freebsd, linux, and so on.
    var pos = p.position
    {
        var vs []Value
        for _, s := range strings.Fields(runtime.GOOS) {
            vs = append(vs, makeBareword(pos, s))
        }
        os = ease(ctx, vs)
    }

    ctx.globe = &globe{
        Scope: newScope(pos, ctx.scope, nil, `globe "smart"`),
        flagEntries: make(map[string][]entry),
        loaded: make(map[string]*project),
        args: make(map[Value][]Value),
    }

    // FIXME: ctx.scope.scopename(ctx, ".GLOBE", ctx.globe.Scope)
    ctx.globe.os,    _ = ctx.globe.define(ctx, defVoid, ".os", os)
    ctx.globe.goals, _ = ctx.globe.define(ctx, defVoid, ".goals", makeNone(pos))
    ctx.globe.mode,  _ = ctx.globe.define(ctx, defVoid, ".mode",  makeNull(pos))
    return
}

type filestub struct {
    dir      string   // full directory where the file was or should be found
    sub      string   // matched sub path (see project.search), may be Dir (absolete path)
    name     string   // constant represented name (e.g. relative filename)
    filemap *filemap  // matched pattern (see 'files' directive)
    other   *filestub // pointed to another stub (in a different project) of the same file
}
func (p *filestub) subname() string {
    if isAbsOrRel(p.sub) {
        return p.name
    } else {
        return filepath.Join(p.sub, p.name)
    }
}

type filebase struct {
    stub filestub    // cycled-list of file stubs of different projects
    info os.FileInfo // file info if exists
    _updated bool // true if this file has been updated by a program
    _updatedDeps []Value // any updated deps
    _travin int
    _traved int
    _dirty  int
}
func (p *filebase) exists() bool { return p.info != nil }

type stat_dir struct { string }
type stat_sub struct { string }
type stat_nonexist struct { bool }
type stat_fileinfo struct{ os.FileInfo }

func stat(ctx Context, name string, ii ...interface{}) (file *File) {
    var sub, dir string
    var nonexist bool
    var fileInfo os.FileInfo
    for _, i := range ii {
        switch t := i.(type) {
        case *project: dir = t.absPath
        case stat_dir: dir = t.string
        case stat_sub: sub = t.string
        case stat_fileinfo: fileInfo = t.FileInfo
        case stat_nonexist: nonexist = t.bool
        default:
            erro(ctx, "invalid stat arg: %v", us(i)).debug(2)
            return
        }
    }

    var (
        base *filebase
        stub *filestub
        fullname string
        u = _universe(ctx)
    )

    u.statmutex.Lock(); defer u.statmutex.Unlock()

    // Trims / suffix
    if dir != "" { dir = filepath.Clean(dir) }
    if sub != "" { sub = filepath.Clean(sub) }
    if false {
        var t = strings.HasPrefix(name, "./")
        if name!= "" { name = filepath.Clean(name) }
        if t         { name = "./" + name }
    }

    if filepath.IsAbs(name) {
        if fullname = name; dir == "" {
            //dir, sub = filepath.Dir(fullname), ""
            //name = filepath.Base(fullname)
        } else if strings.HasPrefix(fullname, dir+pathSep) {
            tail := fullname[len(dir)+1:]
            //sub  = filepath.Dir(tail)
            //name = filepath.Base(tail)
            if sub == "" { name = tail } else
            if strings.HasPrefix(fullname, sub+pathSep) {
                name = tail[len(sub)+1:]
            }
        } else if dir != "" {
            if true { dir = "" } else if false {
                erro(ctx, "dir name conflicts: %s <-> %s (sub=%v)", dir, name, sub).debug(16)
                unreachable("path error")
            } else {
                return
            }
        }
    } else if filepath.IsAbs(sub) {
        if fullname = filepath.Join(sub, name); dir == "" {
            dir = sub // trims / suffix
            sub = "" // .
        } else if sub == dir {
            sub = "" // .
        } else if strings.HasPrefix(sub, dir) {
            sub = strings.TrimPrefix(sub, dir)
            sub = strings.TrimPrefix(sub, pathSep)
            sub = filepath.Clean(sub)
        } else if false {
            dir = sub
            sub = ""
        } else {
            unreachable("conflicted sub/dir: ", sub, " ", dir) //return
        }
    } else if filepath.IsAbs(dir) {
        fullname = filepath.Join(dir, sub, name)
    } else {
        dir = filepath.Join(_workdir(ctx), dir)
        fullname = filepath.Join(dir, sub, name)
    }

    // NOTE: filepath.Join can have the same efffect as filepath.Clean
    var cleanFullname = filepath.Clean(fullname) // clean paths like /path/to/foo/../bar -> /path/to/bar
    if base, _ = u.statcache[cleanFullname]; base != nil {
        if base.info == nil {
            if fileInfo == nil { fileInfo, _ = os.Stat(fullname) }
            if fileInfo == nil && !nonexist { return nil }
            base.info = fileInfo
        }

        var head = &base.stub
        if enable_assertions {
            for stub = head; stub != nil ; stub = stub.other {
                s1, s2 := filepath.Join(stub.dir, stub.sub, stub.name), filepath.Join(fullname)
                assert(s1 == s2, "fullname '%s' conflicted:\n" +
                    "panic: (%s, %s, %s) %s\n" +
                    "panic: (%s, %s, %s) %s\n",
                    fullname,
                    stub.dir, stub.sub, stub.name, s1,
                    dir, sub, name, s2)
                if stub.other == head { break }
            }
        }

        for stub = head; stub != nil; stub = stub.other {
            if stub.dir == dir && stub.sub == sub && stub.name == name {
                goto GotFile
            }
            if stub.other == head { break }
        }

        stub = &filestub{ dir, sub, name, nil, head.other }
        head.other = stub
    } else {
        if fileInfo == nil { fileInfo, _ = os.Stat(fullname)
            if fileInfo == nil && !nonexist { return nil }
        }

        base = &filebase{ filestub{ dir, sub, name, nil, nil }, fileInfo, false, nil, 0, 0, 0 }
        base.stub.other = &base.stub
        stub = &base.stub
        u.statcache[cleanFullname] = base
    }

GotFile:
    file = &File{valbase{ctx.Position()},base,stub}
    return
}

func AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil {
            break
        }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
           searchPaths = append(searchPaths, s)
        }
    }
    return
}

func (ctx *universe) AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil { break }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
            ctx.paths = append(ctx.paths, s)
        } else {
            return fmt.Errorf("path '%s' is not dir", s)
        }
    }
    return nil
}

func (_tx *universe) search(ctx Context, linfo *loadinfo, specName string) (absPath string, isDir bool) {
    if specName == "." {
        erro(ctx, "not possible to chain itself").debug()
    } else if abs := filepath.IsAbs(specName); abs || specName == "~" || specName == ".." ||
        hasPrefix(specName, "~"+pathSep, "."+pathSep, ".."+pathSep) {
        var (
            s = specName
            sx string
        )
        if !abs && linfo.absDir != "" {
            sx = filepath.Join(linfo.absDir, s)
            if a, e := filepath.Abs(sx); e == nil {
                s = a
            } else {
                erro(ctx, "abs: %v", e)
                return
            }
        }
        if fi, err := os.Stat(s); err == nil { return s, fi.IsDir() }

        sx = s + ".smart"
        if fi, err := os.Stat(sx); err == nil { return sx, fi.IsDir() }

        sx = s + ".sm"
        if fi, err := os.Stat(sx); err == nil { return sx, fi.IsDir() }
    } else {
        for _, base := range _tx.paths {
            var s string
            if filepath.IsAbs(base) {
                s = filepath.Join(base, specName)
            } else {
                s = filepath.Join(_workdir(ctx), base, specName)
            }
            if fi, err := os.Stat(s); err == nil && fi != nil {
                return s, fi.IsDir()
            }
        }
    }
    return
}

func startCPUProfile(ctx Context, name string, heap ...bool) (stop func()) {
    var fn string
    if filepath.IsAbs(name) { fn = name } else
    if m := ctx.Globe().main; m == nil {} else
    if f := m.tempFile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        erro(ctx, "%T: %v", e, e).debug()
    } else if e = pprof.StartCPUProfile(f); e != nil {
        erro(ctx, "%T: %v", e, e).debug()
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        if heap != nil && heap[0] { runtime.GC() // update memory statistics
            if e = pprof.WriteHeapProfile(f); e != nil {
                erro(ctx, "WriteHeapProfile: %v", e).debug()
            }
        }
        f.Close()
    }}
}

func startHeapProfile(ctx Context, name string) (stop func()) {
    var fn string
    if filepath.IsAbs(name) { fn = name } else
    if m := ctx.Globe().main; m == nil {} else
    if f := m.tempFile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        erro(ctx, "%T: %v", e, e).debug()
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        runtime.GC() // update memory statistics
        if e = pprof.WriteHeapProfile(f); e != nil {
            erro(ctx, "WriteHeapProfile: %v", e).debug()
        }
        f.Close()
    }}
}

func (_tx *universe) run() (result []Value, travestates []*travestate) {
    if _tx.noRun { return }

    var main = _tx.globe.main
    if main == nil {
        erro(_tx, "no targets to update `%v`", _tx.globe.goals).debug()
        return
    }

    var ctx Context = closureWith(_tx, main.scope)
    if _tx.verbose { info(ctx, "goal: %v", main).debug() }

    removeTempDirs(ctx)

    if _tx.cpuProf != "" || _tx.autoProfs {
        var name = _tx.cpuProf
        if name == "" { name = "run.cpu.auto.prof" }
        defer startCPUProfile(ctx, name, true)()
    } else if _tx.memProf != "" || _tx.autoProfs {
        var name = _tx.memProf
        if name == "" { name = "run.mem.auto.prof" }
        defer startHeapProfile(ctx, name)()
    }

    var done bool
    for _, flag := range _tx.globe.flags {
        if _tx.verboseExecFlags { info(at(ctx, flag), "%v", flag) }

        var s = flag.Value.string(ctx)
        var args, _ = _tx.globe.args[flag]
        var entries, _ = _tx.globe.flagEntries[s]
        for _, entry := range entries {
            var ctx = at(ctx, entry.Position())
            if _tx.verboseExecFlags {
                info(ctx, "%v", entry)
                flush(ctx)
            }

            var ( res []Value; traves []*travestate )
            if res, traves = entry.execute(ctx, args...); len(traves) > 0 {
                for _, brk := range traves {
                    if brk.what == traveFail {
                        erro(at(ctx,brk.pos), "execute '%v': %v", entry, brk).debug()
                    }
                }
            }
            result = append(result, res...)
            done = true
        }
    }
    if done { return }

    var updated int
    var goals []Value
    var collect func(proj *project, vals []Value) bool
    collect = func(proj *project, vals []Value) bool {
        if len(vals) == 0 {
            if entry := proj.defaultEntry; entry != nil {
                goals = append(goals, entry)
            } else {
                // NOTE: ignored project
            }
            return true
        }
        for _, goal := range vals {
            switch t := goal.(type) {
            case *null, *none: // just ignore
            case *bareword:
                if entries := proj.resolveEntries(ctx, t.s, true); entries == nil {
                    erro(ctx, "no such entry `%s`", t.s).debug()
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case *delegate:
                var s = t.string(ctx)
                if entries := proj.resolveEntries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug()
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case flag:
                var s = t.string(ctx)
                if entries := proj.resolveEntries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug()
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case *argumented:
                {
                    // For examples:
                    //     project-name(-clean)
                    //     project/spec(-clean)
                    //     xxxx()
                    var (
                        s = t.Value.string(ctx)
                        args = merge(t.args...)
                        found int
                    )
                    for _, p := range _tx.globe.loaded {
                        if p.name == s || p.spec == s { found += 1
                            if !collect(p, args) { return false }
                        }
                    }
                    if found == 0 {
                        erro(ctx, `"%s" not loaded: %v`, s, args).debug()
                        return false
                    }
                }
            default:
                errostack(ctx, 3, "%v: unknown target: %v (%s)", proj, goal, typeof(goal)).debug(16)
                return false
            }
        }
        return true
    }

    if collect(main, merge(_tx.globe.goals.value)) {
        if len(goals) == 0 {
            if entry := main.defaultEntry; entry != nil {
                goals = append(goals, entry)
            }
        }
        for _, goal := range goals {
            var args, _ = _tx.globe.args[goal]
            result = append(result, updateGoal(ctx, goal, args)...)
            updated += 1
        }
    }
    return
}

// load loads smart files, making it as individual func to avoid being abused by loaders.
func (_tx *universe) load() (err error) {
    if _tx.traceLaunch { defer un(l_trace(l_launch, "universe.load")) }

    var args []Value
    var base = _workdir(_tx)
    var ctx Context = _tx
    if s := filepath.Join(base, ".smart", "modules"); s != "" {
        if _, e := os.Stat(s); e == nil { _tx.AddSearchPaths(s) }
    }
    if s := filepath.Join(base, entryFileName); s != "" {
        if _, e := os.Stat(s); e != nil { s = filepath.Join(base, "build.smart")
            if _, e := os.Stat(s); e != nil { s = "" }
        }
        if s != "" {
            var pos Position
            pos.Filename = s
            pos.Line = 1
            ctx = at(ctx, pos)
        }
    }

    defer func(l *loader) { _tx.globe.top = l } (_tx.globe.top)

    _tx.globe.top = &loader{terminal:terminal{ctx, []*Scope{_tx.globe.Scope}}}

    if s := strings.Join(os.Args[1:], " "); s != "" {
        if v := _tx.globe.top.text(ctx, base, s); 0 < len(v) {
            args = parseOpts(ctx, &_tx.commandline, v...)
        }
    }

    if v := _tx.reconfigure; v { _tx.commandline.configure = v }
    if v := _tx.fastMode; v { // Turn off many things for fast mode:
        //_tx.noImportFiles = v
        _tx.noDepsGrep = v
        _tx.noDeps = v
        _tx.noGrep = v
    }

    if _tx.verbose { defer func(t time.Time) {
        prompt(ctx, "Goals %v (%s)\n", _tx.globe.goals, time.Now().Sub(t))
    } (time.Now()) }

    assert(_tx.globe.args != nil, "globe args is nil")

    if _tx.autoProfs {
        if f, e := os.Create(filepath.Join(baseWorkDir, "load.cpu.auto.prof")); e != nil {
            erro(ctx, "%v", e).debug()
            return
        } else {
            defer f.Close()
            if e := pprof.StartCPUProfile(f); e != nil {
                erro(ctx, "could not start CPU profile: %v", e).debug()
                return
            }
            defer pprof.StopCPUProfile()
        }
        defer func() {
            var prof string //= _tx.memProf
            if prof == "" { prof = filepath.Join(baseWorkDir, "load.mem.auto.prof") }
            if f, e := os.Create(prof); e != nil {
                erro(ctx, "%v", e).debug()
                return
            } else {
                defer f.Close()
                runtime.GC() // update memory statistics
                if e := pprof.WriteHeapProfile(f); e != nil {
                    erro(ctx, "could not start CPU profile: %v", e).debug()
                    return
                }
            }
        } ()
    }

    var mode = new(bareword)
    for _, target := range args {
        switch t := target.(type) {
        case *pair: _tx.globe.pairs = append(_tx.globe.pairs, t)
        case flag: _tx.globe.flags = append(_tx.globe.flags, t)
            if s := t.Value.string(ctx); s == "clean" {
                mode.position, mode.s = t.Position(), "clean"
            }
        case *argumented:
            _tx.globe.args[t.Value] = t.args
            if f, ok := t.Value.(flag); ok {
                _tx.globe.flags = append(_tx.globe.flags, f)
            } else {
                _tx.globe.goals.append(ctx, t/*.value*/)
            }
        default:
            _tx.globe.goals.append(ctx, t)
        }
    }
    if mode.s == "" { if _tx.commandline.configure {
        mode.s = "configure"
    } else {
        mode.s = "goals"
    }}
    _tx.globe.mode.value = mode

    defer func(t time.Time) { if d := time.Now().Sub(t); _tx.verboseImport {
        var name string
        if p := _tx.globe.top.project(); p != nil { name = p.name }
        prompt(ctx, "└·%s … (%s)\n", name, d)
    } else if d > _tx.slow {
        if m := _tx.globe.main; m != nil {
            warn(at(ctx, m.position), "slow loading (%v)!!\n", d).debug(6)
        } else {
            prompt(ctx, "%s:1:warning: slow loading (%v)!!\n", base, d).debug(6)
        }
    }} (time.Now())

    if _tx.verboseImport { prompt(ctx, "┌→%s\n", base) }
    if!_tx.globe.top.path(ctx, base, nil) { return }
    if _tx.globe.main == nil { erro(ctx, "nothing loaded\n").debug() }
    return
}

// A globe represents a global execution context.
type globe struct {
    *Scope

    loads  []*loadinfo
    top      *loader

    main     *project
    loaded   map[string]*project // loaded projects

    stack  []map[string]*def

    args map[Value][]Value
    flagEntries map[string][]entry
    flags []flag
    pairs []*pair

    os    *def
    goals *def
    mode  *def
}

func (g *globe) SetScopeOuter(scope *Scope) { scope.outer = g.Scope }
func (g *globe) AddFlagEntry(name string, entry entry) {
    flags, _ := g.flagEntries[name]
    flags     = append(flags, entry)
    g.flagEntries[name] = flags
    return
}

// project returns a new project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *globe) project(ctx Context, outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *project) {
    if outer == nil { outer = g.Scope }

    m = &project{
        position: ctx.Position(),
        absPath: absPath,
        relPath: relPath,
        tmpPath: tmpPath,
        use:  new(uselist), // TODO: use scopename instead?
        spec: spec,
        name: name,
    }
    m.scope = newScope(m.position, outer, m, fmt.Sprintf("project %q", name))
    m.scope.mutex.Lock()
    // outer.mutex.Lock()
    // if s, y := outer.elems[".self"]; y { m.scope.elems[".base"] = s } else {
    //     if true { warn(ctx, "%v: no base", name).debug(12) }
    // }
    // outer.mutex.Unlock()
    m.scope.elems[".self"] = self{m}
    m.scope.elems[".usee"] = m.use
    m.scope.mutex.Unlock()
    m.use.name = "usee"
    m.use.owner_ = m
    m.use.scope = m.scope

    if g.main == nil && spec != "" && name != "@" && name != "~" {
        for outer != nil && outer != g.Scope {
            if p := outer.project; p != nil && p.name == "@" {
                return
            }
            outer = outer.outer
        }
        g.main = m
    }
    return
}
