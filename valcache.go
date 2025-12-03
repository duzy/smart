//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "reflect"
    "strings"
	"sort"
    "fmt"
)

const test_hit = false
const test_val = ""

type char rune

type hit_pat   struct{ *valcache }
type hit_0     struct{ *valcache ; string }
type hit_s     struct{ *valcache ; string }
type fullmatch struct{ *valcache ; any }
type valcache  struct{ a []any; o []string; v map[string]*valcache }

type cache struct{ Context }
func (c cache) do(ctx Context, op any) any {
	var ( x *valcache ; y bool )
	switch t := op.(type) {
	case property: if t&(propCache) != 0 { return true }
	case hit_s:
		if k := t.string; k == "" {
			return hit_result{t.valcache, false}
		} else {
			return c.do(ctx, hit_0{t.valcache, k})
		}
	case hit_0:
		var k = t.string
		if t.v == nil { t.v = map[string]*valcache{k:x} }
		if x, y = t.v[k]; !y || x == nil {
			x = new(valcache)
			t.o = append(t.o, k)
			t.v[k] = x
		}
		return hit_result{x, y}
	}
	return c.Context.do(ctx, op)
}

type uncache struct{ Context ; a []any }
func (u *uncache) do(ctx Context, op any) any {
	switch t := op.(type) {
	case property: if t&(propUncache) != 0 { return true }
	case hit_s:
		if k := t.string; k == "" {
			return hit_result{t.valcache, false}
		} else {
			return u.do(ctx, hit_0{t.valcache, k})
		}
	case hit_0:
		if x, y := t.v[t.string]; y && x != nil {
			return do(ctx, fullmatch{x, t.string})
		} else {
			return do(&uncache_pat{ctx, t.string, 0}, hit_pat{t.valcache})
		}
	case fullmatch:
		var x, y, s = t.valcache, false, any(nil)
		if t.any == nil {
			erro(ctx, "%v %v %v", ts(u.a), ts(t.a), x).trace()
		}
	aloop:
		for _, a := range t.a {
			switch a := a.(type) {
			case filemap:
				if y, s, _ = match(ctx, a.pattern, t.any); y {
					do(ctx, filemap_name{a, s.(string)})
					break aloop
				}
			default:
				erro(ctx, "%v", ts(a)).trace()
			}
		}
		return hit_result{x, y}
	case filemap_name, rule_name:
		u.a = append(u.a, t) // collect matched result
	}
	return u.Context.do(ctx, op)
}

type uncache_pat struct{ Context ; string ; int }
func (u *uncache_pat) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case hit_pat:
		var ( c = t.valcache )
		for ; u.int < len(u.string); u.int += 1 {
			var r = u.string[u.int]
			if x, y := c.v[string(r)]; y && x != nil { c = x ; continue }
			if truly(ctx, fullmatch{c, u.string}) { return hit_result{c, true} }
			for _, k := range []string{"*","**"} {
				if x, y := c.v[k]; y && x != nil {
					t := do(&uncache_pat{ctx, u.string, u.int+1}, hit_pat{x})
					note(ctx, "%s %s %v %v", k, u.string, c, x).debug()
					if r, y := t.(hit_result); y && r.bool { return r }
				}
			}
			return hit_result{c, false}
		}
		return do(ctx, fullmatch{c, u.string})
	}
	return u.Context.do(ctx, op)
}

type hit_ctx struct{ Context ; any }
func (p *hit_ctx) do(ctx Context, op any) any {
	switch t := op.(type) {
	case fullmatch: t.any = p.any; op = t
	}
	return p.Context.do(ctx, op)
}

type hit_rs struct{ Context ; string ; int }
func (h *hit_rs) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case fullmatch: t.any = h.string; op = t
	case hit_s:
		var ( c = t.valcache ; full bool )
		for ; h.int < len(h.string); h.int += 1 {
			var r = string(h.string[h.int])
			if x, y := do(h.Context, hit_s{c, r}).(hit_result); y {
				c, full = x.valcache, x.bool || full
			}
		}
		return hit_result{c, full}
	}
	return h.Context.do(ctx, op)
}

type hit_ss struct{ Context ; ss []string ; int }
func (u *hit_ss) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case hit_s:
		var ( c = t.valcache )
		for ; u.int < len(u.ss); u.int += 1 {
			for _, s := range seperate(u.ss[u.int], ".") {
				if x, y := do(ctx, hit_0{c, s}).(hit_result); y {
					if c = x.valcache; x.bool { return x }
				} else {
					erro(ctx, "%v %v", s, u.ss).trace()
				}
				for _, k := range []string{"*","**"} {
					if x, y := c.v[k]; y && x != nil {
						if truly(ctx, fullmatch{x, u.ss}) { return hit_result{x, true} }
						var a = do(&hit_ss{ctx, u.ss, u.int+1}, hit_s{x, ""})
						if r, y := a.(hit_result); y && r.bool { return r }
					}
				}
			}
		}
		return hit_result{c, false}
	}
	return u.Context.do(ctx, op)
}

type hit_glob struct{ Context ; *globpat ; int }
func (g *hit_glob) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case fullmatch: t.any = g.globpat; op = t
	case hit_s:
		var ( c = t.valcache ; full bool )
		for ; g.int < len(g.elems); g.int += 1 {
			switch k := g.elems[g.int].(type) {
			case *globmeta, *globrange:
				if x, y := do(g.Context, hit_s{c, __string(ctx, k)}).(hit_result); y {
					c, full = x.valcache, x.bool || full
				}
			default:
				if x, y := do(&hit_rs{g.Context, __string(ctx, k), 0}, hit_s{c,""}).(hit_result); y {
					c, full = x.valcache, x.bool || full
				}
			}
		}
		return hit_result{c, full}
	}
	return g.Context.do(ctx, op)
}

type hit_perc struct{ Context ; *percpat ; int }
func (p *hit_perc) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case fullmatch: t.any = p.percpat; op = t
	case hit_s:
		erro(pc(ctx,p.percpat), "%v %s %v", p.percpat, t.string, t.valcache).trace()
	}
	return p.Context.do(ctx, op)
}

type hit_regex struct{ Context ; *regexpat ; int }
func (r *hit_regex) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case fullmatch: t.any = r.regexpat; op = t
	case hit_s:
		erro(pc(ctx,r.regexpat), "%v %s %v", r.regexpat, t.string, t.valcache).trace()
	}
	return r.Context.do(ctx, op)
}

func seperate(s, sep string) (ss []string) {
	if true {
		for i, s := range strings.Split(s, ".") {
			if 0 < i { ss = append(ss, ".") }; ss = append(ss, s)
		}
	} else {
		ss = []string{s}
	}
	return
}

func any_str(a any) (s string) {
	switch t := a.(type) {
	case filemap: return t.String()
	}
	return fmt.Sprintf("%v", a)
}

type rule_name struct{ *rule ; string }
func (p rule_name) String() string { return "{"+p.rule.String()+" name="+p.string+"}" }

type filemap_name struct{ filemap ; string }
func (p filemap_name) String() string { return "{"+p.filemap.String()+" name="+p.string+"}" }

type valcache_value struct{ *valcache ; Value }
func (v valcache_value) String() string { return v.valcache.String() }
func (v valcache_value) ts(t string) string { return "{="+t+" "+v.valcache.String()+" "+ts(v.Value)+"}" }
func (p *valcache) _keys() (ss map[string]struct{}) {
	ss = make(map[string]struct{})
	for _, v := range reflect.ValueOf(p.v).MapKeys() {
		ss[fmt.Sprintf("%s", v.Interface())] = struct{}{}
	}
	return
}
func (p *valcache) keys(b ...bool) (ss []string) {
	for _, v := range reflect.ValueOf(p.v).MapKeys() {
		ss = append(ss, fmt.Sprintf("%s", v.Interface()))
	}
	if __t(b...) { sort.Strings(ss) }
	return
}
func (p *valcache) ks(b ...bool) (s string) {
	var ss []string
	for _, v := range reflect.ValueOf(p.v).MapKeys() {
		ss = append(ss, fmt.Sprintf("%s", v.Interface()))
	}
	if __t(b...) { sort.Strings(ss) }
	return "["+strings.Join(ss, " ")+"]"
}
func (p *valcache) String() (s string) { // NOTE: for debug
    for k, v := range p.a { s += fmt.Sprintf("%v:%v,", k, any_str(v)) }
	for _, k := range p.o { s += fmt.Sprintf("%v:%v,", k, p.v[k]) }
    if s != "" { s = s[:len(s)-1] } // aka strings.TrimSuffix(s, ",")
    return "{"+s+"}"
}
func (p *valcache) fullmatch(ctx Context, k any) (res bool) {
	for _, a := range p.a {
		if res, _, _ = match(ctx, a, k); res {
			switch t := a.(type) {
			case filemap: do(ctx, filemap_name{t, joinp(ctx, k)})
			case *rule: do(ctx, rule_name{t, joinp(ctx, k)})
			}
			return
		}
	}
	return
}

type hit_result struct{ *valcache ; bool }
func (r hit_result) String() string { return fmt.Sprintf("{%v,%v}", r.valcache, r.bool) }

func hit(ctx Context, c *valcache, k any) (*valcache, bool) {
	var t = _hit(ctx, c, k)
	return t.valcache, t.bool
}
func _hit(ctx Context, c *valcache, k any) (r hit_result) {
	switch t := k.(type) {
	case *loc : return _hit(ctx, c, t.Value)
	case *file: return _hit(ctx, c, t.name)
	case *rule: return _hit(ctx, c, t.target)
	case *argumented: return _hit(ctx, c, t.Value)
	case []string:
		return do(&hit_ss{ctx, t, 0}, hit_s{c,""}).(hit_result)
	case string:
		return do(&hit_ss{ctx, strings.Split(t, pathSep), 0}, hit_s{c,""}).(hit_result)
	case *path:
		return do(&hit_ss{ctx, strings.Split(__string(ctx, t), pathSep), 0}, hit_s{c,""}).(hit_result)
	case *globpat:
		return do(&hit_glob{ctx,t,0}, hit_s{c,""}).(hit_result)
	case *percpat:
		return do(&hit_perc{ctx,t,0}, hit_s{c,""}).(hit_result)
	case *regexpat:
		return do(&hit_regex{ctx,t,0}, hit_s{c,""}).(hit_result)
	case *closure:
		return do(&hit_ctx{ctx,t}, hit_s{c,"&"}).(hit_result)
	case *strval:
		return do(ctx, hit_s{c,`{`+__string(ctx,t)+`}`}).(hit_result)
	case *strcomp:
		return do(ctx, hit_s{c,`"`+__string(ctx,t)+`"`}).(hit_result)
	case *strlit:
		return do(ctx, hit_s{c,`'`+__string(ctx,t.s)+`'`}).(hit_result)
	default:
		return do(ctx, hit_s{c,__string(ctx,t)}).(hit_result)
	}
}

func map_files(ctx Context, p *project, patts, paths []Value) (res []filemap) {
	if checkpoints { defer check_map_files(ctx, p, patts, paths, &res) }
	var base = &_filemap{p, patts, paths}
	for _, patt := range patts {
		switch patt.(type) {
		case *valbase, *null, *none:
			continue
		}

		if c, _ := hit(cache{ctx}, &p.filemap, patt); c == nil {
			erro(pc(ctx,patt), "cache failed: %v", ts(patt)).trace()
		} else {
			f := filemap{base, patt}
			c.a = append(c.a, f)
			res = append(res, f)
		}
    }
    return
}

func map_entry(ctx Context, p *project, target Value, prog *program) (entry entry) {
	var patterned = patterned(ctx, target)
	if !patterned {
		switch target.(type) {
		case *barefile, *file, *path, *percpat, *globpat, *regexpat: goto skip_file
		}
		if t := p.file(unmap_uncheck_ctx{ctx}, target); t != nil {
			target, t.position = t, target.Position()
		}
	skip_file:
	}

    var args []Value // e.g. for pattern filtering
	switch t := target.(type) {
	case *argumented: target, args = t.Value, merge(t.args...)
	case *group:
		erro(ctx, "not supported target: %v", t).trace()
	}

	if c, _ := hit(cache{ctx}, &p.entries, target); c == nil {
		erro(ctx, "uncachable for: %v | %s", target, ts(target)).trace()
	} else {
		r := &rule{target:target, arged:args, program:[]*program{prog}}
		if patterned { p.patterns = append(p.patterns, r) }
		if p.main == nil { p.main = r }
		c.a = append(c.a, r)
		entry = r
	}
	return
}

func unmap[T any](ctx Context, c *valcache, key any) (res []T) {
	var u = uncache{ctx, nil}

	if checkpoints { defer unmap_check(&u, c, key) }

	hit(&u, c, key)

	for _, a := range u.a {
		if x, y := a.(T); y {
			res = append(res, x)
		} else {
			erro(ctx, "%v : %v", ts(key), ts(a)).trace()
		}
	}
    return
}

func unmap_entries(ctx Context, p *project, key any, m *map[*project]struct{}) (res []entry) {
	if checkpoints { defer check_unmap_entries(ctx, p, key, &res) }
	if m == nil { m = &map[*project]struct{}{} } else if _, y := (*m)[p]; y { return }
	if m != nil { (*m)[p] = struct{}{} }
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
	if checkpoints { defer check_unmap_files(ctx, p, key, &res) }
	if m == nil { m = &map[*project]struct{}{} } else if _, y := (*m)[p]; y { return }
	if m != nil { (*m)[p] = struct{}{} }
	if res = unmap[filemap_name](ctx, &p.filemap, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_files(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; c != nil {
		if res = unmap_files(ctx, c, key, m); res != nil { return }
	}
	return
}
