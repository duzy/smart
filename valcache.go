//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package smart

import (
    "reflect"
    "strings"
	"sort"
    "fmt"
)

const test_hit = false && checkpoints
const test_val = ""

type (
	hit_bare  struct{ *valcache ; *compound }
	hit_path  struct{ *valcache ; *path  }
	hit_punc  struct{ *valcache ; token  }
	hit_value struct{ *valcache ; Value  }
	hit_word  struct{ *valcache ; string }
	hit_glob  struct{ *valcache ; string }
	hit_perc  struct{ *valcache ; string }
	hit_regex struct{ *valcache ; string }
	act_cache struct{ *valcache ; string }
	act_unmap struct{ *valcache ; string }
)

type char rune
func (c char) String() string { return string(rune(c)) }

type cache struct { Context } // versus `unmap`
func (c cache) cast(t reflect.Type) Context { return implcast(c, t) }
func (c cache) do(ctx Context, op any) any {
    switch t := op.(type) {
    case hit_punc:
        return _valcache_bool(c.punc(ctx, t.valcache, t.token))
    case hit_word:
        return _valcache_bool(c.word(ctx, t.valcache, t.string))
    case hit_value:
        return _valcache_bool(c.value(ctx, t.valcache, t.Value))
	case hit_bare:
        return _valcache_bool(c.bare_hit(ctx, t.valcache, t.compound))
	case hit_path:
        return _valcache_bool(c.path_hit(ctx, t.valcache, t.path))
    case hit_glob:
        return _valcache_bool(c.glob_hit(ctx, t.valcache, t.string))
    case hit_perc:
        return _valcache_bool(c.perc_hit(ctx, t.valcache, t.string))
    case hit_regex:
        return _valcache_bool(c.rege_hit(ctx, t.valcache, t.string))
    default:
        return c.Context.do(ctx, op)
    }
}

type unmap struct { Context ; a []any } // versus `cache`
func (u *unmap) cast(t reflect.Type) Context { return implcast(u, t) }
func (u *unmap) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        return _valcache_bool(u.unmap(ctx, t.valcache, t.string))
    case hit_punc:
        return _valcache_bool(u.punc(ctx, t.valcache, t.token))
    case hit_word:
        return _valcache_bool(u.word(ctx, t.valcache, t.string))
	case hit_bare:
        return _valcache_bool(u.bare_hit(ctx, t.valcache, t.compound))
	case hit_path:
        return _valcache_bool(u.path_hit(ctx, t.valcache, t.path))
    case hit_glob:
		return do(ctx, act_unmap{t.valcache, t.string})
    case hit_perc:
		return do(ctx, act_unmap{t.valcache, t.string})
    case hit_regex:
		return do(ctx, act_unmap{t.valcache, t.string})
    case filemap_name, rule_name:
        u.a = append(u.a, t)
    case property:
        if t&(propUnmap) != 0 { return true }
    }
    return u.Context.do(ctx, op)
}

type bare_hit struct { Context ; *valcache ; s string ; i int ; solo bool }
func (p *bare_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *bare_hit) ts(t string) string {
	return "{="+t+" "+p.s+" "+ts(p.Context)+"}"
}
func (p *bare_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    case property:
        if p.solo && t&(propFullVal) != 0 { return p.s }
    }
    return p.Context.do(ctx, op)
}

type flag_hit struct { Context ; flag }
func (p *flag_hit) do(ctx Context, op any) any {
	switch t := op.(type) {
	case property:
		if t&(propFullVal) != 0 { return p.flag }
	}
	return p.Context.do(ctx, op)
}

type path_hit struct { Context ; s string ; ss []string ; i int }
func (p *path_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *path_hit) ts(t string) string {
	return "{="+t+" "+p.s+" "+ts(p.Context)+"}"
}
func (p *path_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    case property:
        if t&(propFullVal) != 0 { return p.s }
    }
    return p.Context.do(ctx, op)
}

type globpat_hit struct { Context ; x *globpat }
func (p *globpat_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *globpat_hit) ts(t string) string {
	return "{="+t+" "+p.x.String()+" "+ts(p.Context)+"}"
}
func (p *globpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

type percpat_hit struct { Context ; x *percpat }
func (p *percpat_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *percpat_hit) ts(t string) string {
	return "{="+t+" "+p.x.String()+" "+ts(p.Context)+"}"
}
func (p *percpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

type regexpat_hit struct { Context ; x *regexpat }
func (p *regexpat_hit) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *regexpat_hit) ts(t string) string {
	return "{="+t+" "+p.x.String()+" "+ts(p.Context)+"}"
}
func (p *regexpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _valcache_bool(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

func cacheMapping(ctx Context) bool { return !cacheUnmap(ctx) }
func cacheUnmap(ctx Context) bool { t, _ := do(ctx, propUnmap).(bool); return t }

type rule_name struct { *rule ; name string }
// func (p rule_name) ts(string) string { return "{=rule "+p.name+"}" }

type filemap_name struct { filemap ; name string }
func (p filemap_name) String() string {
	return "{"+p.filemap.String()+" "+p.name+"}"
}

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

func (v valcache_value) String() string { return v.valcache.String() }
func (v valcache_value) ts(t string) string {
	return "{="+t+" "+v.valcache.String()+" "+ts(v.Value)+"}"
}
func (p *valcache) keys() (ss map[string]struct{}) {
	ss = make(map[string]struct{})
	for _, v := range reflect.ValueOf(p.puncs).MapKeys() {
		ss[fmt.Sprintf("%s", v.Interface())] = struct{}{}
	}
	for _, v := range reflect.ValueOf(p.words).MapKeys() {
		ss[v.String()] = struct{}{}
	}
	for _, v := range reflect.ValueOf(p.globs).MapKeys() {
		ss[v.String()] = struct{}{}
	}
	for _, v := range reflect.ValueOf(p.percs).MapKeys() {
		ss[v.String()] = struct{}{}
	}
	for _, v := range reflect.ValueOf(p.reges).MapKeys() {
		ss[v.String()] = struct{}{}
	}
	return
}
func (p *valcache) ks(b ...bool) (s string) {
	var ss []string
	for _, v := range reflect.ValueOf(p.puncs).MapKeys() {
		ss = append(ss, fmt.Sprintf("%s", v.Interface()))
	}
	for _, v := range reflect.ValueOf(p.words).MapKeys() {
		ss = append(ss, v.String())
	}
	for _, v := range reflect.ValueOf(p.globs).MapKeys() {
		ss = append(ss, v.String())
	}
	for _, v := range reflect.ValueOf(p.percs).MapKeys() {
		ss = append(ss, v.String())
	}
	for _, v := range reflect.ValueOf(p.reges).MapKeys() {
		ss = append(ss, v.String())
	}
	if b != nil && b[0] { sort.Strings(ss) }
	return "["+strings.Join(ss, " ")+"]"
}
func (p *valcache) _ks(b ...bool) (s string) {
	for i, v := range reflect.ValueOf(p.puncs).MapKeys() {
		if 0 < i { s += " " }; s += fmt.Sprintf("%s", v.Interface())
		if t := p.puncs[v.Interface().(token)].ks(); t != "[]" { s += ":"+t }
	}
	for _, v := range reflect.ValueOf(p.words).MapKeys() {
		if s != "" { s += " " }; s += v.String()
		if t := p.words[v.String()].ks(); t != "[]" { s += ":"+t }
	}
	for _, v := range reflect.ValueOf(p.globs).MapKeys() {
		if s != "" { s += " " }; s += v.String()
		if t := p.globs[v.String()].ks(); t != "[]" { s += ":"+t }
	}
	for _, v := range reflect.ValueOf(p.percs).MapKeys() {
		if s != "" { s += " " }; s += v.String()
		if t := p.percs[v.String()].ks(); t != "[]" { s += ":"+t }
	}
	for _, v := range reflect.ValueOf(p.reges).MapKeys() {
		if s != "" { s += " " }; s += v.String()
		if t := p.reges[v.String()].ks(); t != "[]" { s += ":"+t }
	}
	return "[" + s + "]"
}
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

func (p *valcache) do_rule_name(ctx Context, r *rule, a any) {
	do(ctx, rule_name{r, _path(ctx, a)})
	return
}

func (p *valcache) fullmatch(ctx Context, k any) (res bool) {
    for _, a := range p.a {
        if res, _, _ = a.match(ctx, k); res {
			switch t := a.(type) {
			case filemap_slot:
				p.do_filemap_name(ctx, t.filemap, k)
			case *rule:
				p.do_rule_name(ctx, t, k)
			}
            return
        }
    }
    return
}

func (p *valcache) hit(ctx Context, k any) (res *valcache, fullmatch bool) {
	if test_hit && do(ctx, propFullVal) == test_val {
		defer func() {
			note(pc(ctx,k), "%5v %v", k, p)
			note(pc(ctx,k), "%5v %v", fullmatch, res)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

	if len(p.a) > 1 { panic(fmt.Sprintf("valcache many: %s %d", typeof(k), len(p.a))) }

	defer func() {
		if res == nil && cacheMapping(ctx) {
			erro(pc(ctx,k), "no valcache for %v : %v", ts(k), p).trace()
		}
	} ()

	switch t := k.(type) {
	case token:
		if x, y := do(ctx, hit_punc{p, t}).(valcache_bool); y { return x.valcache, x.bool }
	case string:
		if x, y := do(ctx, hit_word{p, t}).(valcache_bool); y { return x.valcache, x.bool }
	case valcache_hit:
		return t.hit(ctx, p)
	case []string:
		if true {
			return p.hit_words(ctx, t...)
		} else {
			return unmap_path(ctx, p, strings.Join(t, pathSep), t)
		}
	case Value:
		if indeterminate(ctx, t) {
			if x, y := do(ctx, hit_value{p, t}).(valcache_bool); y { return x.valcache, x.bool }
		} else {
			if x, y := k.(*rule); y {
				a, ay := do(ctx, hit_value{p, x.target}).(valcache_bool)
				b, by := do(ctx, hit_word{p, x.target.string(ctx)}).(valcache_bool)
				note(ctx, "%v : %v %v, %v %v", x.target, a.bool, ay, b.bool, by).debug()
			}
			erro(pc(ctx,k), "non-valcacheable %v : %v", ts(k), p).trace()
		}
	default:
		erro(pc(ctx,k), "non-valcacheable %v : %v", ts(k), p).trace()
	}
	return
}

func (p *valcache) hit_words(ctx Context, words ...string) (res *valcache, full bool) {
	defer func() {
		if res != nil && 0 < len(res.a) && !cacheMapping(ctx) {
			full = res.fullmatch(ctx, strings.Join(words, pathSep))
		}
	} ()
	for _, s := range words {
		for i, s := range strings.Split(s, DOT.String()) {
			if p != nil && (0 < i) {
				if p, _ = p.hit(ctx, DOT); p == nil { return }
			}
			if p != nil && (0 < i || s != "") {
				if p, full = p.hit(ctx, s); p == nil { return }
				if full { return p, full }
			}
		}
	}
	return p, full
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
func (cache) bare_hit(ctx Context, c *valcache, p *compound) (res *valcache, fullmatch bool) {
	for _, elem := range p.elems {
		if c, fullmatch = c.hit(ctx, elem); c == nil {
			erro(ctx, "no valcache for %v : %v : %v", p, ts(elem), c).trace()
		} else if res = c ; fullmatch {
			return
		}
	}
	return
}
func (cache) path_hit(ctx Context, c *valcache, p *path) (res *valcache, fullmatch bool) {
    var x = path_hit{ctx, "", nil, 0}
    var cc Context = &x
    for ; c != nil && x.i < len(p.elems) ; x.i += 1 {
        var elem = p.elems[x.i]

        if c, fullmatch = c.hit(cc, elem) ; c == nil {
			erro(ctx, "no valcache for %v : %v", ts(elem), c).trace()
        }

        if fullmatch || p.len() <= x.i+1 {
            return c, true
        }
    }
	return
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
	if test_hit && do(ctx, propFullVal) == test_val {
		defer func() {
			note(ctx, "%5v : %v , %v", t, c, ts(do(ctx, propFullVal)))
			note(ctx, "%5v : %v", fullmatch, res)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}
    if nil != c.puncs {
        if x, y := c.puncs[t]; y {
			if a := do(ctx, propFullVal); a != nil {
				return x, x.fullmatch(ctx, a)
			} else {
				return x, x.fullmatch(ctx, t)
			}
        }
    }
    if x, y := do(ctx, act_unmap{c, t.String()}).(valcache_bool); y {
        if res, fullmatch = x.valcache, x.bool ; x.bool { return }
    }
    return
}
func (u *unmap) word(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
	if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v , %v", s, c, ts(do(ctx, propFullVal)))
			note(ctx, "%5v : %v", fullmatch, res)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}
	if c.words != nil {
		if x, y := c.words[s]; y {
			if a := do(ctx, propFullVal); a != nil {
				return x, x.fullmatch(ctx, a)
			} else {
				return x, x.fullmatch(ctx, s)
			}
		}
	}
	if x, y := do(ctx, act_unmap{c, s}).(valcache_bool); y {
		if res, fullmatch = x.valcache, x.bool ; x.bool { return }
	}
    return
}
func (*unmap) bare_hit(ctx Context, c *valcache, p *compound) (res *valcache, fullmatch bool) {
	if true {
		if s := p.string(ctx); strings.Contains(s, pathSep) {
			return unmap_path(ctx, c, s, strings.Split(s, pathSep))
		} else {
			_, res, fullmatch = unmap_word(ctx, c, s, cast[*path_hit](ctx) == nil)
		}
	} else {
		for _, elem := range p.elems {
			if c, fullmatch = c.hit(ctx, elem); c == nil {
				erro(ctx, "no valcache for %v : %v : %v", p, ts(elem), c).trace()
			} else if res = c ; fullmatch {
				return
			}
		}
	}
	return
}
func (*unmap) path_hit(ctx Context, c *valcache, p *path) (_ *valcache, _ bool) {
	s := p.string(ctx); return unmap_path(ctx, c, s, strings.Split(s, pathSep))
}
func (*unmap) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
	if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v %v", s, _c)
			note(ctx, "%5v %v", fullmatch, res)
			note(ctx, "%v", ts(ctx)).debug(30)
		}()
	}

	if strings.Contains(s, pathSep) {
		if false { erro(ctx, "%v %v", s, _c).trace() }
		return
	}

	if 0 < len(_c.o) {
		for _, pat := range _c.o[0] {
			var c, y = _c.globs[pat]
			if !y || c == nil {
				if false {erro(ctx, "%v %v - nil glob", s, pat).trace()}
				continue
			}
			if f, _, _ := globMatch(ctx, pat, s); f {
				return c, c.fullmatch(ctx, s)
			}
		}
	}
	if 1 < len(_c.o) {
		for _, pat := range _c.o[1] {
			var c, y = _c.percs[pat]
			if !y || c == nil {
				if false {erro(ctx, "%v %v - nil glob", s, pat).trace()}
				continue
			}
			for _, a := range c.a {
				if x, y, z := a.match(ctx, s); x {
					return c, c.fullmatch(ctx, s)
				} else if false {
					erro(ctx, "TODO: %v %v, %v %v %v, %v", s, pat, x, y, z, ts(a.(*rule).target)).trace()
				}
			}
		}
	}
	if 2 < len(_c.o) {
		for _, pat := range _c.o[2] {
			var c, y = _c.reges[pat]
			if !y || c == nil {
				if false {erro(ctx, "%v %v - nil glob", s, pat).trace()}
				continue
			}
			for _, a := range c.a {
				if x, y, z := a.match(ctx, s); x {
					return c, c.fullmatch(ctx, s)
				} else if false {
					erro(ctx, "TODO: %v %v, %v %v %v, %v", s, pat, x, y, z, ts(a.(*rule).target)).trace()
				}
			}
		}
	}
    return
}

func (p *bare_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if test_hit && do(ctx, propFullVal) == test_val {
		defer func() {
			var pat string
			if 0 < len(p.o) && p.i < len(p.o[0]) {
				pat = fmt.Sprintf("%d. %v", p.i, p.o[0][p.i])
			}
			note(ctx, "%5v %v %v", p.s, p.valcache, pat)
			note(ctx, "%5v %v %v", s, _c, do(ctx, propFullVal)); t := ""; if fullmatch { t = "F" }
			note(ctx, "%5v %v %v %v", t, res, _c == p.valcache, cast[*unmap](ctx).a)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

    if 0 < len(p.o) && p.i < len(p.o[0]) {
        var fullval = do(ctx, propFullVal)
        for _, pat := range p.o[0][p.i:] {
            var c, _ = p.globs[pat]
			if c == nil {
				erro(ctx, "%v: nil cache : %v %v (i=%d)", pat, s, tv(fullval), p.i)
				note(ctx, "%v", p).trace()
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

            if true && fullval != nil && c != nil {
                if f := c.fullmatch(ctx, fullval); f {
                    return c, true
                }
            }
        }
    }
    return
}
func (p *path_hit) unmap(ctx Context, _c *valcache, k string) (res *valcache, fullmatch bool) {
    if test_hit && do(ctx, propFullVal) == test_val {
		defer func() {
			note(ctx, "%5v %v", k, _c)
			note(ctx, "%5v %v ; %d/%d %v", fullmatch, res, p.i, len(p.ss), p.ss[p.i])
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}
	if 0 < len(_c.o) {
		for _, pat := range _c.o[0] {
			var c = _c.globs[pat]

			if checkpoints && truly(ctx, is_test_mode{}) && strings.Contains(pat, "/") {
				erro(ctx, "%v %v %v", k, pat, c).trace()
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
			}
		}
	}
    return
}
func (gc *globpat_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
	if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v", s, _c)
			note(ctx, "%5v : %v , fullval=%v", fullmatch, res, ts(do(ctx, propFullVal)))
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

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
	if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v", s, _c)
			note(ctx, "%5v : %v , %v", fullmatch, res, ts(do(ctx, propFullVal)))
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

    panic("TODO: "+s)
}
func (p *regexpat_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
	if test_hit && (do(ctx, propFullVal) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v", s, _c)
			note(ctx, "%5v : %v , %v", fullmatch, res, ts(do(ctx, propFullVal)))
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

    panic("TODO: "+s)
}

func (*unmap) perc(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if len(_c.o) < 2 { return }
    for _, pat := range _c.o[1] {
        erro(ctx, "TODO: %v %v", ts(s), pat).trace()
    }
    return
}
func (*unmap) _regex(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if len(_c.o) < 3 { return }
    for _, pat := range _c.o[2] {
        erro(ctx, "TODO: %v %v", ts(s), pat).trace()
    }
    return
}

func (p *project) map_files(ctx Context, patts, paths []Value) (res []filemap) {
    if p == nil {
        erro(ctx, "nil project : %v %v", patts, paths).trace()
    }

    var base = &_filemap{p, patts, paths}

    for _, patt := range patts {
        if patt == nil {
            erro(ctx, "nil pattern : paths=%v", paths).trace()
        } else if c, _ := p.filemap.hit(cache{ctx}, patt); c == nil {
            erro(ctx, "cache failed : %v", ts(patt)).trace()
        } else {
            t  := filemap{base, patt}
            c.a = append(c.a, filemap_slot{t})
            res = append(res, t)
        }
    }
    return
}

func (p *project) unmap_entries(ctx Context, key any, m *map[*project]struct{}) (res []entry) {
	if checkpoints && truly(ctx, is_test_mode{}) { defer p.unmap_entries_check(ctx, key, &res) }
	if m == nil { m = &map[*project]struct{}{} } else if _, y := (*m)[p]; y { return }
	if m != nil { (*m)[p] = struct{}{} }
	if truly(ctx, debug_y{}) { ctx = project_ctx{ctx, p} }
	if res = unmap_any[entry](ctx, &p.entries, key); res != nil { return }
	for _, b := range p.bases {
		if res = b.unmap_entries(ctx, key, m); res != nil { return }
	}
	if c := p.configure; false && c != nil {
		if res = c.unmap_entries(ctx, key, m); res != nil { return }
	}
	return
}

func (p *project) unmap_files(ctx Context, key any, m *map[*project]struct{}) (res []filemap_name) {
	if checkpoints && truly(ctx, is_test_mode{}) { defer p.unmap_files_check(ctx, key, &res) }
	if m == nil { m = &map[*project]struct{}{} } else if _, y := (*m)[p]; y { return }
	if m != nil { (*m)[p] = struct{}{} }
	if truly(ctx, debug_y{}) { ctx = project_ctx{ctx, p} }
	if res = unmap_any[filemap_name](ctx, &p.filemap, key); res != nil { return }
	for _, b := range p.bases {
		if res = b.unmap_files(ctx, key, m); res != nil { return }
	}
	if c := p.configure; c != nil {
		if res = c.unmap_files(ctx, key, m); res != nil { return }
	}
	return
}

func unmap__(ctx *unmap, c *valcache, key any) {
	if checkpoints && truly(ctx, is_test_mode{}) {
		defer unmap_check(ctx, c, key)
	}

	var x string
	switch t := key.(type) {
	case string: x = t
	case Value: x = t.string(ctx)
	}
	if x == "" {
		erro(ctx, "empty key: %v", c).trace()
	}

	if s := strings.Split(x, pathSep); 1 < len(s) {
		unmap_path(ctx, c, x, s)
		return
	} else {
		unmap_word(ctx, c, x, true)
		return
	}
}

func unmap_void(ctx *unmap, c *valcache, key any) {
	if checkpoints && truly(ctx, is_test_mode{}) {
		defer unmap_check(ctx, c, key)
	}

	switch x := key.(type) {
	case string:
		if x == "" {
			erro(ctx, "empty key: %v", c).trace()
		} else if s := strings.Split(x, pathSep); 1 < len(s) {
			unmap_path(ctx, c, x, s); return
		} else {
			unmap_word(ctx, c, x, true); return
		}
	case valcache_hit:
		x.hit(ctx, c); return
	default:
		c.hit(ctx, x); return
	}
}

func unmap_any[T any](ctx Context, c *valcache, key any) (res []T) {
	var u = unmap{ctx, nil}

	unmap_void(&u, c, key)

	for _, a := range u.a {
		if x, y := a.(T); y {
			res = append(res, x)
		} else {
			erro(ctx, "%v : %v", ts(key), ts(a)).trace()
		}
	}
    return
}

func unmap_word(ctx Context, c *valcache, s string, solo bool) (_ Context, _ *valcache, y bool) {
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

func unmap_path(ctx Context, c *valcache, s string, ss []string) (_ *valcache, y bool) {
    var x = path_hit{ctx, s, ss, 0}
    for cc := Context(&x) ; c != nil && x.i < len(ss) ; x.i += 1 {
        if cc, c, y = unmap_word(cc, c, ss[x.i], false); y { return c, y }
    }
    return
}

func unmap_entries(ctx Context, key any) []entry {
    return _project(ctx).unmap_entries(ctx, key, nil)
}

func unmap_files(ctx Context, key any) []filemap_name {
    return _project(ctx).unmap_files(ctx, key, nil)
}

func map_files(ctx Context, patts, paths []Value) []filemap {
    return _project(ctx).map_files(ctx, patts, paths)
}
