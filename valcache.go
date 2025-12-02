//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
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

const test_hit = false
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
	act_unmap struct{ *valcache ; string }
)

type char rune
func (c char) String() string { return string(rune(c)) }

type cache struct{ Context }
func (c cache) inner() Context { return c.Context }
func (c cache) cast(t reflect.Type) Context { return icast(c, t) }
func (c cache) do(ctx Context, op any) any {
    switch t := op.(type) {
    case hit_punc:
        return _hit_result(c.punc(ctx, t.valcache, t.token))
    case hit_word:
        return _hit_result(c.word(ctx, t.valcache, t.string))
    case hit_value:
        return _hit_result(c.value(ctx, t.valcache, t.Value))
	case hit_bare:
        return _hit_result(c.hit_bare(ctx, t.valcache, t.compound))
	case hit_path:
        return _hit_result(c.hit_path(ctx, t.valcache, t.path))
    case hit_glob:
        return _hit_result(c.hit_glob(ctx, t.valcache, t.string))
    case hit_perc:
        return _hit_result(c.hit_perc(ctx, t.valcache, t.string))
    case hit_regex:
        return _hit_result(c.hit_regex(ctx, t.valcache, t.string))
    default:
        return c.Context.do(ctx, op)
    }
}

type uncache struct{ Context ; a []any }
func (u *uncache) inner() Context { return u.Context }
func (u *uncache) cast(t reflect.Type) Context { return icast(u, t) }
func (u *uncache) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        return _hit_result(u.unmap(ctx, t.valcache, t.string))
    case hit_punc:
        return _hit_result(u.punc(ctx, t.valcache, t.token))
    case hit_word:
        return _hit_result(u.word(ctx, t.valcache, t.string))
	case hit_bare:
        return _hit_result(u.hit_bare(ctx, t.valcache, t.compound))
	case hit_path:
        return _hit_result(u.hit_path(ctx, t.valcache, t.path))
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

type full_kval struct{}

type bare_hit struct{ Context ; *valcache ; s string ; i int ; solo bool }
func (p *bare_hit) inner() Context { return p.Context }
func (p *bare_hit) cast(t reflect.Type) Context { return icast(p, t) }
func (p *bare_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _hit_result(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
	case full_kval:
		if p.solo { return p.s }
    }
    return p.Context.do(ctx, op)
}

type flag_hit struct{ Context ; flag }
func (p *flag_hit) inner() Context { return p.Context }
func (p *flag_hit) cast(t reflect.Type) Context { return icast(p, t) }
func (p *flag_hit) do(ctx Context, op any) any {
	switch op.(type) {
	case full_kval: return p.flag
	}
	return p.Context.do(ctx, op)
}

type path_hit struct{ Context ; s string ; ss []string ; i int }
func (p *path_hit) inner() Context { return p.Context }
func (p *path_hit) cast(t reflect.Type) Context { return icast(p, t) }
func (p *path_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _hit_result(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
	case full_kval: return p.s
    }
    return p.Context.do(ctx, op)
}

type globpat_hit struct{ Context ; x *globpat }
func (p *globpat_hit) inner() Context { return p.Context }
func (p *globpat_hit) cast(t reflect.Type) Context { return icast(p, t) }
func (p *globpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _hit_result(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

type percpat_hit struct{ Context ; x *percpat }
func (p *percpat_hit) inner() Context { return p.Context }
func (p *percpat_hit) cast(t reflect.Type) Context { return icast(p, t) }
func (p *percpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _hit_result(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

type regexpat_hit struct{ Context ; x *regexpat }
func (p *regexpat_hit) inner() Context { return p.Context }
func (p *regexpat_hit) cast(t reflect.Type) Context { return icast(p, t) }
func (p *regexpat_hit) do(ctx Context, op any) any {
    switch t := op.(type) {
    case act_unmap:
        x := _hit_result(p.unmap(ctx, t.valcache, t.string))
        if x.valcache != nil { return x }
    }
    return p.Context.do(ctx, op)
}

func cacheMapping(ctx Context) bool { return !cacheUnmap(ctx) }
func cacheUnmap(ctx Context) bool { t, _ := do(ctx, propUnmap).(bool); return t }

type rule_name struct{ *rule ; name string }
type filemap_name struct{ filemap ; name string }
func (p filemap_name) String() string { return "{"+p.filemap.String()+" name="+p.name+"}" }

func any_str(a any) (s string) {
	switch t := a.(type) {
	case filemap: return t.String()
	}
	return fmt.Sprintf("%v", a)
}

func _hit_result(x *valcache, y bool) hit_result { return hit_result{x,y} }

type hit_result struct{ *valcache ; bool }
type valcache_value struct{ *valcache ; Value }
type valcache struct{
	a []any
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
func (v valcache_value) ts(t string) string { return "{="+t+" "+v.valcache.String()+" "+ts(v.Value)+"}" }
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
    for k, v := range p.a     { s += fmt.Sprintf("%v:%s(%v),", k, typeof(v), any_str(v)) }
    for k, v := range p.puncs { s += fmt.Sprintf("%v:%v,", k, v) }
    for k, v := range p.words { s += fmt.Sprintf("%v:%v,", k, v) }
    for i, m := range []map[string]*valcache{ p.globs, p.percs, p.reges } {
        if m != nil { for _, k := range p.o[i] { s += fmt.Sprintf("%v:%v,", k, m[k]) } }
    }
    for k, v := range p.value { s += fmt.Sprintf("%v:%s(%v),", k, typeof(v), any_str(v)) }
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
	s, _ := _joinpath(ctx, a)
	do(ctx, filemap_name{f, s})
	return
}

func (p *valcache) do_rule_name(ctx Context, r *rule, a any) {
	s, _ := _joinpath(ctx, a)
	do(ctx, rule_name{r, s})
	return
}

func (p *valcache) fullmatch(ctx Context, k any) (res bool) {
	for _, a := range p.a {
		if res, _, _ = match(ctx, a, k); res {
			switch t := a.(type) {
			case filemap:
				p.do_filemap_name(ctx, t, k)
			case *rule:
				p.do_rule_name(ctx, t, k)
			}
			return
		}
	}
	return
}

func hit(ctx Context, c *valcache, k any) (res *valcache, full bool) {
	if test_hit && do(ctx, full_kval{}) == test_val {
		defer func() {
			note(pc(ctx,k), "%5v %v", k, c)
			note(pc(ctx,k), "%5v %v", full, res)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}
	if 1 < len(c.a) {
		panic(fmt.Sprintf("valcache many: %s %d", typeof(k), len(c.a)))
	}
	
	switch t := k.(type) {
	case *argumented: return hit(ctx, c, t.Value)
	case *loc: return hit(ctx, c, t.Value)
	case *file: return hit(ctx, c, strings.Split(t.name, pathSep))
	case *rule: return hit(ctx, c, t.target)
	case *word: return hit(ctx, c, t.s)
	case *punct: return hit(ctx, c, t.token)
	case *globmeta: return hit(ctx, c, t.token)
	case *binary, *octal, *decimal, *hexadecimal:
		return hit(ctx, c, __string(ctx, t))
	case *strval:
		if c, full = hit(ctx, c, STRING|LBRACE/* TODO: */); c == nil { return }
		return hit(ctx, c, __string(ctx, t))
	case *strlit:
		if c, full = hit(ctx, c, STRING); c == nil { return }
		return hit(ctx, c, t.s)
	case flag:
		if c, full = hit(ctx, c, MINUS); c == nil { return }
		if t.Value.kind()&(KindNull|KindNone) != 0 { return c, full }
		return hit(&flag_hit{ctx,t}, c, t.Value)
	case token:
		if x, y := do(ctx, hit_punc{c,t}).(hit_result); y { return x.valcache, x.bool }
	case string:
		if x, y := do(ctx, hit_word{c,t}).(hit_result); y { return x.valcache, x.bool }
	case *compound:
		if x, y := do(ctx, hit_bare{c,t}).(hit_result); y { return x.valcache, x.bool }
	case *path:
		if x, y := do(ctx, hit_path{c,t}).(hit_result); y { return x.valcache, x.bool }
	case *globpat:
		if x, y := do(&globpat_hit{ctx,t}, hit_glob{c, __string(ctx,t)}).(hit_result); y { return x.valcache, x.bool }
	case *percpat:
		if x, y := do(&percpat_hit{ctx,t}, hit_perc{c, __string(ctx,t)}).(hit_result); y { return x.valcache, x.bool }
	case *regexpat:
		if x, y := do(&regexpat_hit{ctx,t}, hit_regex{c, __string(ctx,t)}).(hit_result); y { return x.valcache, x.bool }
	case []string:
		for _, s := range t {
			for i, s := range strings.Split(s, DOT.String()) {
				if c != nil && (0 < i) { if c, _ = hit(ctx, c, DOT); c == nil { break } }
				if c != nil && (0 < i || s != "") {
					if c, full = hit(ctx, c, s); c == nil { break }
					if full { return c, full }
				}
			}
		}
		if c != nil && 0 < len(c.a) && !cacheMapping(ctx) {
			full = c.fullmatch(ctx, strings.Join(t, pathSep))
		}
		return c, full
	default:
		if x, y := k.(*rule); y {
			a, ay := do(ctx, hit_value{c, x.target}).(hit_result)
			b, by := do(ctx, hit_word{c, __string(ctx, x.target)}).(hit_result)
			note(ctx, "%v : %v %v, %v %v", x.target, a.bool, ay, b.bool, by).debug()
		}
		erro(pc(ctx,k), "uncachable %v | %v", ts(k), c).trace()
	}

	erro(ctx, "hit: %s | %s", ts(k), ts(c)).trace()
	return
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
    } else if x, y := p.puncs[t]; !y || x == nil {
        res = &valcache{}
        p.puncs[t] = res
    } else {
        res = x
    }
	return
}
func (cache) word(ctx Context, p *valcache, s string) (res *valcache, fullmatch bool) {
    if  p.words == nil {    res = &valcache{}
        p.words = make(map[string]*valcache)
        p.words[s] = res
    } else if x, y := p.words[s]; !y || x == nil {
        res = &valcache{}
        p.words[s] = res
    } else {
        res = x
    }
	return
}
func (cache) hit_bare(ctx Context, c *valcache, p *compound) (res *valcache, fullmatch bool) {
	for _, elem := range p.elems {
		if c, fullmatch = hit(ctx, c, elem); c == nil {
			erro(ctx, "no valcache for %v : %v : %v", p, ts(elem), c).trace()
		} else if res = c ; fullmatch {
			return
		}
	}
	return
}
func (cache) hit_path(ctx Context, c *valcache, p *path) (res *valcache, fullmatch bool) {
    var x = path_hit{ctx, "", nil, 0}
    var cc Context = &x
    for ; c != nil && x.i < len(p.elems) ; x.i += 1 {
        var elem = p.elems[x.i]

        if c, fullmatch = hit(cc, c, elem) ; c == nil {
			erro(ctx, "no valcache for %v : %v", ts(elem), c).trace()
        }

        if fullmatch || p.len() <= x.i+1 {
            return c, true
        }
    }
	return
}
func (cache) hit_glob(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
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
func (cache) hit_perc(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
    if  c.percs == nil {    res = &valcache{}
        c.percs = make(map[string]*valcache)
        c.percs[s] = res
        c.perc_o(s)
    } else if x, y := c.percs[s]; !y || res == nil {
        res = &valcache{}
        c.percs[s] = res
        c.perc_o(s)
    } else {
        res = x
    }
	return
}
func (cache) hit_regex(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
    if  c.reges == nil {    res = &valcache{}
        c.reges = make(map[string]*valcache)
        c.reges[s] = res
        c.rege_o(s)
    } else if x, y := c.reges[s]; !y || res == nil {
        res = &valcache{}
        c.reges[s] = res
        c.rege_o(s)
    } else {
        res = x
    }
	return
}

func (u *uncache) value(ctx Context, p *valcache, v Value) (res *valcache, fullmatch bool) {
    for _, t := range p.value {
        if equal(ctx, v, t.Value) {
            res = t.valcache
            return
        }
    }
    return
}
func (u *uncache) punc(ctx Context, c *valcache, t token) (res *valcache, fullmatch bool) {
	if test_hit && do(ctx, full_kval{}) == test_val {
		defer func() {
			note(ctx, "%5v : %v , %v", t, c, ts(do(ctx, full_kval{})))
			note(ctx, "%5v : %v", fullmatch, res)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}
    if nil != c.puncs {
        if x, y := c.puncs[t]; y {
			if a := do(ctx, full_kval{}); a != nil {
				return x, x.fullmatch(ctx, a)
			} else {
				return x, x.fullmatch(ctx, t)
			}
        }
    }
    if x, y := do(ctx, act_unmap{c, t.String()}).(hit_result); y {
        if res, fullmatch = x.valcache, x.bool ; x.bool { return }
    }
    return
}
func (u *uncache) word(ctx Context, c *valcache, s string) (res *valcache, fullmatch bool) {
	if test_hit && (do(ctx, full_kval{}) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v , %v", s, c, ts(do(ctx, full_kval{})))
			note(ctx, "%5v : %v", fullmatch, res)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}
	if c.words != nil {
		if x, y := c.words[s]; y {
			if a := do(ctx, full_kval{}); a != nil {
				return x, x.fullmatch(ctx, a)
			} else {
				return x, x.fullmatch(ctx, s)
			}
		}
	}
	if x, y := do(ctx, act_unmap{c, s}).(hit_result); y {
		if res, fullmatch = x.valcache, x.bool ; x.bool { return }
	}
    return
}
func (*uncache) hit_bare(ctx Context, c *valcache, p *compound) (res *valcache, fullmatch bool) {
	if true {
		if s := __string(ctx, p); strings.Contains(s, pathSep) {
			return unmap_path(ctx, c, s, strings.Split(s, pathSep))
		} else {
			_, res, fullmatch = unmap_word(ctx, c, s, cast[*path_hit](ctx) == nil)
		}
	} else {
		for _, elem := range p.elems {
			if c, fullmatch = hit(ctx, c, elem); c == nil {
				erro(ctx, "no valcache for %v : %v : %v", p, ts(elem), c).trace()
			} else if res = c ; fullmatch {
				return
			}
		}
	}
	return
}
func (*uncache) hit_path(ctx Context, c *valcache, p *path) (_ *valcache, _ bool) {
	s := __string(ctx, p); return unmap_path(ctx, c, s, strings.Split(s, pathSep))
}
func (*uncache) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
	if test_hit && (do(ctx, full_kval{}) == test_val || s == test_val) {
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
				if x, y, z := match(ctx, a, s); x {
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
				if x, y, z := match(ctx, a, s); x {
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
    if test_hit && do(ctx, full_kval{}) == test_val {
		defer func() {
			var pat string
			if 0 < len(p.o) && p.i < len(p.o[0]) {
				pat = fmt.Sprintf("%d. %v", p.i, p.o[0][p.i])
			}
			note(ctx, "%5v %v %v", p.s, p.valcache, pat)
			note(ctx, "%5v %v %v", s, _c, do(ctx, full_kval{})); t := ""; if fullmatch { t = "F" }
			note(ctx, "%5v %v %v %v", t, res, _c == p.valcache, cast[*uncache](ctx).a)
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

    if 0 < len(p.o) && p.i < len(p.o[0]) {
        var fullval = do(ctx, full_kval{})
        for _, pat := range p.o[0][p.i:] {
            var c, _ = p.globs[pat]
			if c == nil {
				erro(ctx, "nil cache : %v %v %v (i=%d)", tv(pat), tv(fullval), s, p.i)
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
    if test_hit && do(ctx, full_kval{}) == test_val {
		defer func() {
			note(ctx, "%5v %v", k, _c)
			note(ctx, "%5v %v ; %d/%d %v", fullmatch, res, p.i, len(p.ss), p.ss[p.i])
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}
	if 0 < len(_c.o) {
		for _, pat := range _c.o[0] {
			var c = _c.globs[pat]

			if checkpoints && strings.Contains(pat, "/") {
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
	if test_hit && (do(ctx, full_kval{}) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v", s, _c)
			note(ctx, "%5v : %v , fullval=%v", fullmatch, res, ts(do(ctx, full_kval{})))
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

	var g func(*valcache)

	g = func(c *valcache) {
		for _, a := range c.a {
			if x, y := a.(filemap) ; y {
				var p = __string(ctx, x.pattern)
				if f, _, _ := globMatch(ctx, s, p) ; f {
					c.do_filemap_name(ctx, x, p)
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
	if test_hit && (do(ctx, full_kval{}) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v", s, _c)
			note(ctx, "%5v : %v , %v", fullmatch, res, ts(do(ctx, full_kval{})))
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

    panic("TODO: "+s)
}
func (p *regexpat_hit) unmap(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
	if test_hit && (do(ctx, full_kval{}) == test_val || s == test_val) {
		defer func() {
			note(ctx, "%5v : %v", s, _c)
			note(ctx, "%5v : %v , %v", fullmatch, res, ts(do(ctx, full_kval{})))
			note(ctx, "%v", ts(ctx)).debug(30)
		} ()
	}

    panic("TODO: "+s)
}

func (*uncache) perc(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if len(_c.o) < 2 { return }
    for _, pat := range _c.o[1] {
        erro(ctx, "TODO: %v %v", ts(s), pat).trace()
    }
    return
}
func (*uncache) _regex(ctx Context, _c *valcache, s string) (res *valcache, fullmatch bool) {
    if len(_c.o) < 3 { return }
    for _, pat := range _c.o[2] {
        erro(ctx, "TODO: %v %v", ts(s), pat).trace()
    }
    return
}

func map_files(ctx Context, p *project, patts, paths []Value) (res []filemap) {
    var base = &_filemap{p, patts, paths}
    for _, patt := range patts {
		switch patt.(type) {
		case *valbase, *null, *none:
			continue
		}

		if c, _ := hit(cache{ctx}, &p.filemap, patt); c != nil {
			c.a = append(c.a, filemap{base, patt})
			res = append(res, filemap{base, patt})
			continue
		}

		erro(pc(ctx,patt), "cache failed: %v", ts(patt)).trace()
    }
    return
}

func map_entry(ctx Context, p *project, target Value, prog *program) (entry entry) {
    var patterned = patterned(ctx,target)
    if !patterned {
        // NOTE: it should work too if not checking against files
        switch target.(type) {
        case *file, *path, *barefile, *percpat, *globpat, *regexpat:
        default:
            if f := p.file(unmap_uncheck_ctx{ctx}, target); f != nil {
                f.position = target.Position()
                target = f
            }
        }
    }

    defer func() {
        if entry != nil {
            entry.programs(append(entry.programs(), prog)...)
        }
    } ()

    var arged []Value // e.g. for pattern filtering
    switch t := target.(type) {
    case *group:
        erro(ctx, "group target not supported: %v", t).trace()
    case *argumented:
        target, arged = t.Value, merge(t.args...)
    }

    var c, _ = hit(cache{ctx}, &p.entries, target)
    if c == nil {
        erro(ctx, "no cache for target: %v", target).trace()
    }

    if len(c.a) == 0 {
        var rule = &rule{ target:target, arged:arged }

        if patterned {
            p.patterns = append(p.patterns, rule)
        }

        entry = rule
        c.a = append(c.a, rule)
    } else if p, y := c.a[0].(*rule); y {
        entry = p
    } else {
        errostack(ctx, 3, "wrong cache: %v", c).trace()
    }

    if entry != nil && p.main == nil { p.main = entry }
    return
}

func unmap_entries(ctx Context, p *project, key any, m *map[*project]struct{}) (res []entry) {
	if checkpoints { defer p.unmap_entries_check(ctx, key, &res) }
	if m == nil { m = &map[*project]struct{}{} } else if _, y := (*m)[p]; y { return }
	if m != nil { (*m)[p] = struct{}{} }
	if truly(ctx, debug_y{}) { ctx = project_ctx{ctx, p} }
	if res = unmap[entry](ctx, &p.entries, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_entries(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; false && c != nil {
		if res = unmap_entries(ctx, c, key, m); res != nil { return }
	}
	return
}

func unmap_files(ctx Context, p *project, key any, m *map[*project]struct{}) (res []filemap_name) {
	if m == nil { m = &map[*project]struct{}{} } else if _, y := (*m)[p]; y { return }
	if m != nil { (*m)[p] = struct{}{} }
	if truly(ctx, debug_y{}) { ctx = project_ctx{ctx, p} }
	if res = unmap[filemap_name](ctx, &p.filemap, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_files(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; c != nil {
		if res = unmap_files(ctx, c, key, m); res != nil { return }
	}
	return
}

func unmap_void(ctx *uncache, c *valcache, key any) {
	if checkpoints { defer unmap_check(ctx, c, key) }

	switch x := key.(type) {
	case string:
		if x == "" {
			erro(ctx, "empty key: %v", c).trace()
		} else if s := strings.Split(x, pathSep); 1 < len(s) {
			unmap_path(ctx, c, x, s); return
		} else {
			unmap_word(ctx, c, x, true); return
		}
	default:
		hit(ctx, c, x); return
	}
}

func unmap[T any](ctx Context, c *valcache, key any) (res []T) {
	var u = uncache{ctx, nil}

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
			if c, y = hit(cc, c, DOT) ; y || c == nil { break }
		}
		if c != nil && (0 < i || t != "") {
			if c, y = hit(cc, c, t) ; y || c == nil { break }
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
