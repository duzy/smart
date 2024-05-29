//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package smart

import (
    "reflect"
    "strings"
    "fmt"
)

const test_hit = false
const test_val = ".deps/xx/yy/zzzzzzzzzz"

type char rune
func (c char) String() string { return string(rune(c)) }

type cache struct { Context } // versus `unmap`
func (c cache) cast(t reflect.Type) Context { return implcast(c, t) }
func (c cache) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actPuncHit:
        return _valcache_bool(c.punc(ctx, t.valcache, t.token))
    case actWordHit:
        return _valcache_bool(c.word(ctx, t.valcache, t.string))
    case actValuHit:
        return _valcache_bool(c.value(ctx, t.valcache, t.Value))
    case actGlobHit:
        return _valcache_bool(c.glob_hit(ctx, t.valcache, t.string))
    case actPercHit:
        return _valcache_bool(c.perc_hit(ctx, t.valcache, t.string))
    case actRegeHit:
        return _valcache_bool(c.rege_hit(ctx, t.valcache, t.string))
    default:
        return c.Context.do(ctx, op)
    }
}

type unmap struct { Context ; a []any } // versus `cache`
func (u *unmap) cast(t reflect.Type) Context { return implcast(u, t) }
func (u *unmap) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actPuncHit:
        return _valcache_bool(u.punc(ctx, t.valcache, t.token))
    case actWordHit:
        return _valcache_bool(u.word(ctx, t.valcache, t.string))
    case actUnmap:
        return _valcache_bool(u.unmap(ctx, t.valcache, t.string))
    case actGlobHit:
		return do(ctx, actUnmap{t.valcache, t.string})
    case actPercHit:
		return do(ctx, actUnmap{t.valcache, t.string})
    case actRegeHit:
		return do(ctx, actUnmap{t.valcache, t.string})
    case filemap_name:
        u.a = append(u.a, t)
    case property:
        if t&(propUnmap) != 0 { return true }
    }
    return u.Context.do(ctx, op)
}

type bare_hit struct { Context ; *valcache ; s string ; i int ; solo bool }
func (p *bare_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *bare_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    case property:
        if p.solo && t&(propFullVal) != 0 { return p.s }
    }
    return p.Context.do(ctx, op)
}

type path_hit struct { Context ; s string ; ss []string ; i int }
func (p *path_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *path_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    case property:
        if t&(propFullVal) != 0 { return p.s }
    }
    return p.Context.do(ctx, op)
}

type globpat_hit struct { Context ; x *globpat }
func (p *globpat_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *globpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

type percpat_hit struct { Context ; x *percpat }
func (p *percpat_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *percpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

type regexpat_hit struct { Context ; x *regexpat }
func (p *regexpat_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *regexpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case actUnmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

func cacheMapping(ctx Context) bool { return !cacheUnmap(ctx) }
func cacheUnmap(ctx Context) bool { t, _ := do(ctx, propUnmap).(bool); return t }

type filemap_name struct { filemap ; name string }
type filemap_slot struct { filemap }
func (p filemap_slot) String() string { return p.filemap.String() }
func (p filemap_slot) match(ctx Context, a any) (bool, any, []string) {
    x, y, z := p.filemap.match(ctx, a)
    return x, y, []string{z}
}

func _valcache_bool(x *valcache, y bool) valcache_bool { return valcache_bool{x,y} }

type valcacheable interface { match(Context, any) (bool, any, []string) }
type valcache_hit interface { hit(Context, *valcache) (*valcache, bool) }
type valcache_bool   struct { *valcache ; bool }
type valcache_value  struct { *valcache ; Value }
type valcache struct {
    a []valcacheable
    o [][]string // priority of globs/percs/reges

    // NOTE: using `map[any]*valcache` is easier but lose some performance
    puncs map[token ]*valcache
    words map[string]*valcache
    globs map[string]*valcache
    percs map[string]*valcache
    reges map[string]*valcache

    value []valcache_value
}

// func (v valcache_value) String() string { return fmt.Sprintf("{%s %s}", v.valcache, v.Value) }
func (v valcache_value) String() string { return v.valcache.String() }

func (p *valcache) String() (s string) { // NOTE: for debug
    for k, v := range p.a     { s += fmt.Sprintf("%v:%s(%v),", k, typeof(v), v) }
    for k, v := range p.puncs { s += fmt.Sprintf("%v:%v,", k, v) }
    for k, v := range p.words { s += fmt.Sprintf("%v:%v,", k, v) }
    for i, m := range []map[string]*valcache{ p.globs, p.percs, p.reges } {
        if m != nil { for _, k := range p.o[i] { s += fmt.Sprintf("%v:%v,", k, m[k]) } }
    }
    for k, v := range p.value { s += fmt.Sprintf("%v:%s(%v),", k, typeof(v), v) }
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

func (p *valcache) do_filemap_name(ctx Context, f filemap, a any) {
	do(ctx, filemap_name{f, _path(ctx, a)})
	return
}

func (p *valcache) fullmatch(ctx Context, k any) (yes bool) {
    for _, a := range p.a {
        if yes, _, _ = a.match(ctx, k); yes {
			switch t := a.(type) {
			case filemap_slot:
				p.do_filemap_name(ctx, t.filemap, k)
			}
            return
        }
    }
    return
}

func (p *valcache) hit(ctx Context, k any) (res *valcache, fullmatch bool) {
    if test_hit && do(ctx, propFullVal) == test_val { defer func() {
        note(ctx, "%5v %v", k, p)
        note(ctx, "%5v %v", fullmatch, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    ctx = at(ctx, k)

    defer trace(ctx)

    switch t := k.(type) {
    case string:
        if x, y := do(ctx, actWordHit{p, t}).(valcache_bool); y {
            return x.valcache, x.bool
        } else {
            erro(ctx, "unhit: %v %v", us(k), us(ctx)).debug()
            return
        }
    case token:
        if x, y := do(ctx, actPuncHit{p, t}).(valcache_bool); y {
            return x.valcache, x.bool
        } else {
            erro(ctx, "unhit: %v %v", us(k), us(ctx)).debug()
            return
        }
    case valcache_hit:
        if res, fullmatch = t.hit(ctx, p) ; res == nil && cacheMapping(ctx) {
            erro(ctx, "no valcache for %v : %v", us(k), p).debug()
            return
        } else {
            return
        }
    case Value:
        if indeterminate(ctx, t) {
            if x, y := do(ctx, actValuHit{p, t}).(valcache_bool); y {
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

func (cache) value(ctx Context, p *valcache, v Value) (res *valcache, fullmatch bool) {
    for _, t := range p.value {
        if equal(ctx, v, t.Value) {
            res = t.valcache
            return
        }
    }

    res = &valcache{}
    p.value = append(p.value, valcache_value{res, v})
    return
}

func (cache) punc(ctx Context, p *valcache, t token) (res *valcache, fullmatch bool) {
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
func (cache) word(ctx Context, p *valcache, s string) (res *valcache, fullmatch bool) {
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
func (cache) glob_hit(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
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
func (cache) perc_hit(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
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
func (cache) rege_hit(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
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
func (u *unmap) value(ctx Context, p *valcache, v Value) (res *valcache, fullmatch bool) {
    for _, t := range p.value {
        if equal(ctx, v, t.Value) {
            res = t.valcache
            return
        }
    }
    return
}
func (u *unmap) punc(ctx Context, c *valcache, t token) (res *valcache, fullmatch bool) {
    if test_hit && do(ctx, propFullVal) == test_val { defer func() {
        note(ctx, "%5v : %v , %v", t, c, us(do(ctx, propFullVal)))
        note(ctx, "%5v : %v", fullmatch, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if nil != c.puncs {
        if x, y := c.puncs[t]; y {
			if a := do(ctx, propFullVal); a != nil {
				return x, x.fullmatch(ctx, a)
			} else {
				return x, x.fullmatch(ctx, t)
			}
        }
    }
    if x, y := do(ctx, actUnmap{c, t.String()}).(valcache_bool); y {
        if res, fullmatch = x.valcache, x.bool ; x.bool { return }
    }
    return
}
func (u *unmap) word(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
    if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) { defer func() {
        note(ctx, "%5v : %v , %v", s, c, us(do(ctx, propFullVal)))
        note(ctx, "%5v : %v", fullmatch, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}
    if nil != c.words {
        if x, y := c.words[s]; y {
			if a := do(ctx, propFullVal); a != nil {
				return x, x.fullmatch(ctx, a)
			} else {
				return x, x.fullmatch(ctx, s)
			}
        }
    }
    if x, y := do(ctx, actUnmap{c, s}).(valcache_bool); y {
        if res, fullmatch = x.valcache, x.bool ; x.bool { return }
    }
    return
}
func (*unmap) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) { defer func() {
        note(ctx, "%5v %v", s, _c)
        note(ctx, "%5v %v", fullmatch, res)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    if strings.Contains(s, pathSep) { return }

    defer trace(ctx)

	if 0 < len(_c.o) {
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
	}
	if 1 < len(_c.o) {
		for _, pat := range _c.o[1] {
			var c, _ = _c.percs[pat]
			if c == nil {
				erro(ctx, "%v %v - nil glob", s, pat).debug()
				continue
			}
			erro(ctx, "TODO: ", pat).debug()
		}
	}
	if 2 < len(_c.o) {
		for _, pat := range _c.o[2] {
			var c, _ = _c.reges[pat]
			if c == nil {
				erro(ctx, "%v %v - nil glob", s, pat).debug()
				continue
			}
			erro(ctx, "TODO: ", pat).debug()
		}
	}
    return
}
func (p *bare_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if test_hit && do(ctx, propFullVal) == test_val { defer func() {
        var pat string
        if 0 < len(p.o) && p.i < len(p.o[0]) {
            pat = fmt.Sprintf("%d. %v", p.i, p.o[0][p.i])
        }
        note(ctx, "%5v %v %v", p.s, p.valcache, pat)
        note(ctx, "%5v %v %v", s, _c, do(ctx, propFullVal)); t := ""; if fullmatch { t = "F" }
        note(ctx, "%5v %v %v %v", t, res, _c == p.valcache, cast[*unmap](ctx).a)
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    defer trace(ctx)

    if 0 < len(p.o) && p.i < len(p.o[0]) {
        var fullval = do(ctx, propFullVal)
        for _, pat := range p.o[0][p.i:] {
            var c, _ = p.globs[pat]
            if c == nil {
                erro(ctx, "%v %v - nil glob", s, pat).debug()
                return
            }

            p.i += 1

			if f, _, _ := globMatch(ctx, pat, s) ; f {
				if fullval == nil {
					return c, c.fullmatch(ctx, s)
				} else {
					return c, c.fullmatch(ctx, fullval)
				}
			}

			if true && p.s != s {
				if f, _, _ := globMatch(ctx, pat, p.s); f {
					if fullval == nil {
						return c, c.fullmatch(ctx, p.s)
					} else {
						return c, c.fullmatch(ctx, fullval)
					}
				}
			}

            if true && fullval != nil {
                if f := c.fullmatch(ctx, fullval); f {
                    return c, true
                }
            }
        }
    }
    return
}
func (p *path_hit) unmap(ctx Context, _c *valcache, k string) (res *valcache, fullmatch bool) {
    if test_hit && do(ctx, propFullVal) == test_val { defer func() {
        note(ctx, "%5v %v", k, _c)
        note(ctx, "%5v %v ; %d/%d %v", fullmatch, res, p.i, len(p.ss), p.ss[p.i])
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    if len(_c.o) < 1 { return }

    defer trace(ctx)

    var num = len(p.ss)

    for _, pat := range _c.o[0] {
        var c = _c.globs[pat]

        if checkpoints {
            if strings.Contains(pat, "/") {
                erro(ctx, "%v %v %v", k, pat, c).debug()
            }
        }

        var s string
        if strings.Contains(pat, "**") {
            s = p.s
        } else {
            s = p.ss[p.i] // aka k
        }

        if y, _, _ := globMatch(ctx, pat, s) ; y {
            if c.fullmatch(ctx, p.ss) {
                return c, true
            }
            if p.i + 1 < num {
                return c, false
            }
        }
    }
    return
}
func (gc *globpat_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) { defer func() {
        note(ctx, "%5v : %v", s, _c)
        note(ctx, "%5v : %v , fullval=%v", fullmatch, res, us(do(ctx, propFullVal)))
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

	var g func(*valcache)

	g = func(c *valcache) {
		for _, a := range c.a {
			if x, y := a.(filemap_slot) ; y {
				var p = x.pattern.string(ctx)
				if f, _, _ := globMatch(ctx, s, p) ; f {
					c.do_filemap_name(ctx, x.filemap, p)
					res, fullmatch = c, true
				}
			}
		}
		for _, c := range c.puncs { g(c) }
		for _, c := range c.words { g(c) }
		for _, c := range c.globs { g(c) }
	}

	g(_c)
    return
}
func (p *percpat_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) { defer func() {
        note(ctx, "%5v : %v", s, _c)
        note(ctx, "%5v : %v , %v", fullmatch, res, us(do(ctx, propFullVal)))
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    panic("TODO: "+s)
}
func (p *regexpat_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) { defer func() {
        note(ctx, "%5v : %v", s, _c)
        note(ctx, "%5v : %v , %v", fullmatch, res, us(do(ctx, propFullVal)))
        note(ctx, "%v", us(ctx)).debug(30)
    }()}

    panic("TODO: "+s)
}

func (*unmap) perc(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if len(_c.o) < 2 { return }
    for _, pat := range _c.o[1] {
        erro(ctx, "TODO: %v %v", us(s), pat).debug(3)
    }
    return
}
func (*unmap) regx(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if len(_c.o) < 3 { return }
    for _, pat := range _c.o[2] {
        erro(ctx, "TODO: %v %v", us(s), pat).debug(3)
    }
    return
}

func (p *project) map_files(ctx Context, patts, paths []Value) (res []filemap) {
    defer trace(ctx)

    if p == nil {
        erro(ctx, "nil project : %v %v", patts, paths).debug()
        return
    }

    var base = &_filemap{p, patts, paths}

    for _, patt := range patts {
        if patt == nil {
            erro(ctx, "nil pattern : paths=%v", paths).debug()
        } else if c, _ := p.filemap.hit(cache{ctx}, patt); c == nil {
            erro(ctx, "cache failed : %v", us(patt)).debug()
        } else {
            t  := filemap{base, patt}
            c.a = append(c.a, filemap_slot{t})
            res = append(res, t)
        }
    }
    return
}

func map_files(ctx Context, patts, paths []Value) []filemap {
    return ctx.project().map_files(ctx, patts, paths)
}

func (p *project) unmap_bare(ctx Context, c *valcache, s string, solo bool) (_ Context, _ *valcache, y bool) {
    var cc = &bare_hit{ctx, c, s, 0, solo}
	for i, t := range strings.Split(s, DOT.String()) {
		if c != nil && (0 < i) {
			if c, y = c.hit(cc, DOT) ; y || c == nil { break }
		}
		if c != nil && (0 < i || t != "") {
			if c, y = c.hit(cc, t) ; y || c == nil { break }
		}
	}
	return cc, c, y
}

func (p *project) unmap_ps(ctx Context, c *valcache, s string, ss []string) (_ *valcache, y bool) {
    var x = path_hit{ctx, s, ss, 0}
    for cc := Context(&x) ; c != nil && x.i < len(ss) ; x.i += 1 {
        if cc, c, y = p.unmap_bare(cc, c, ss[x.i], false); y {
            if false && cast[*unmap](ctx).a == nil {
                note(ctx, "%v %v %v", s, ss[x.i], c).debug(1)
            }
            return c, y
        }
    }
    return
}

func (p *project) unmap(ctx *unmap, c *valcache, key any) (res *valcache, fullmatch bool) {
    defer trace(ctx)
    defer flush(ctx)

	if checkpoints {
		defer func() {
			if fullmatch && res == nil {
				erro(at(ctx, key), "nil full unmap: %v", c).debug(10)
				return
			}
			if fullmatch && ctx.a == nil {
				note(at(ctx, key), "{=%s %v} %v", typeof(key), key, c)
				erro(at(ctx, key), "uncollected full unmap : %v", res).debug(10)
				return
			}
		} ()
	}

	if x, y := key.(string) ; y {
		if x == "" {
			erro(ctx, "empty key : %v", c).debug()
		} else if s := strings.Split(x, pathSep) ; len(s) > 1 {
			return p.unmap_ps(ctx, c, x, s)
		} else {
			_, res, fullmatch = p.unmap_bare(ctx, c, x, true)
			return
		}
	} else if x, y := key.(valcache_hit); y {
		return x.hit(ctx, c)
	} else {
		return c.hit(ctx, key)
	}

	return
}

func unmap_t[T any](ctx Context, p *project, c *valcache, key any) (res []T) {
    var u = unmap{ctx, nil}
    p.unmap(&u, c, key)

    defer trace(ctx)

    for _, a := range u.a {
        if x, y := a.(T); y {
            res = append(res, x)
        } else {
			erro(ctx, "%v : %v", us(key), us(a)).debug(1)
		}
    }
    return
}

func (p *project) unmap_entries(ctx Context, key any) []entry {
    return unmap_t[entry](ctx, p, &p.entries, key)
}

func (p *project) unmap_files(ctx Context, key any) []filemap_name {
    return unmap_t[filemap_name](ctx, p, &p.filemap, key)
}

func unmap_entries(ctx Context, key any) entryArray {
    return ctx.project().unmap_entries(ctx, key)
}

func unmap_files(ctx Context, key any) []filemap_name {
    return ctx.project().unmap_files(ctx, key)
}
