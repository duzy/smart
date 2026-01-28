//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "reflect"
    "strings"
    "fmt"
)

const test_hit = false
const test_val = ""

type hit_s struct{ *valcache ; s [][]string }
type valcache struct{
	a []any
	o []string
	v map[string]*valcache
}

type cache_t struct{ Context }
func (c cache_t) do(ctx Context, op any) any {
	switch t := op.(type) {
	case property: if t&(propCache) != 0 { return true }
	case hit_s:
		var c = t.valcache
		for i := 0; i < len(t.s); i += 1 {
			for j := 0; j < len(t.s[i]); j += 1 {
				var s = t.s[i][j]
				if c.v == nil { c.v = map[string]*valcache{} }
				if x, y := c.v[s]; !y || x == nil {
					v := new(valcache)
					c.o = append(c.o, s)
					c.v[s] = v
					c = v
				} else {
					c = x
				}
			}
		}
		return []*valcache{c}
	}
	return c.Context.do(ctx, op)
}

func uncache_match(ctx Context, pat, val any) (f bool, s string) {
	var t any
	if patterned(ctx, pat) {
		if  f, t, _ = match(ctx, pat, val); !f && patterned(ctx, val) {
			f, t, _ = match(ctx, val, pat)
		}
	} else if patterned(ctx, val) {
		f, t, _ = match(ctx, val, pat)
	} else {
		f, t, _ = match(ctx, pat, val)	
	}
	if checkpoints && true { switch s := sf("%v %v → %v", pat, val, f); s {
	case "foo/bar/zz/x.h foo/bar/z?/?.h → false":
		debug(ctx, "matched incorrect: %v", t, callstack{stop:"smart.hit"})
	default:
		if strings.Contains(s, "foo/bar/z?/?.h") {
			info(ctx, "%v %v %v", pat, val, f)
		}
	}}
	if f {
		switch t := t.(type) {
		case   string: s = t
		case []string: s = strings.Join(t, pathSep)
		default: debug(ctx, "%v %v: %v", pat, val, ts(t), trace{})
		}
		if false {
			debug(ctx, "%v %v", pat, val, callstack{stop:"smart.uncache"})
		}
	}
	return
} 

func uncache(ctx Context, _c *valcache, ss [][]string) (r []*valcache) {
	var f0 func(*valcache, [][]string, int, int, int) bool
	var f2 func(*valcache, []string, int, ...string) bool // **
	var f3 func(*valcache, []string, int, ...string) bool // **
	var ff func(*valcache)
	var left, right func(any) (bool, string)
	if a := do(ctx, ctxany{}); a != nil {
		left  = func(b any) (bool, string) { return uncache_match(ctx, b, a) }
		right = func(b any) (bool, string) { return uncache_match(ctx, a, b) }
	} else {
		debug(ctx, "no full value: %v %v", ss, _c, trace{})
	}

	fullmatch := func(c *valcache, match func(any) (bool, string)) (ok bool) {
		for _, a := range c.a {
			switch a := a.(type) {
			case filemap:
				var _a = filemap{a._filemap, expand(_final(ctx), a.pattern)}
				for _, v := range merge(_a.pattern) {
					if f, s := match(v); f {
						if 0 < do(ctx, matched_filemap{_a, s}).(int) {
							r, ok = append(r, c), true
						} else {
							debug(ctx, "%v %v", ts(a), s, trace{})
						}
					}
				}
			case *rule:
				var _a = &rule{expand(_final(ctx), a.target), a.arged, a.program}
				for _, v := range merge(_a.target) {
					if f, s := match(v); f {
						if 0 < do(ctx, matched_rule{_a, s}).(int) {
							r, ok = append(r, c), true
						} else {
							debug(ctx, "%v %v", ts(a), s, trace{})
						}
					}
				}
			default:
				debug(ctx, "%v", ts(a), trace{})
			}
		}
		return
	}

	f0 = func(c *valcache, ss [][]string, i, j, k int) (_ bool) {
		if c == nil {
			return
		} else if i == len(ss) {
			if j == len(ss[i-1]) {
				if k == len(ss[i-1][j-1]) {
					return fullmatch(c, left)
				}
			}
			return
		} else if j == len(ss[i]) {
			return f0(c, ss, i+1, 0, 0)
		} else if k == len(ss[i][j]) {
			return f0(c, ss, i, j+1, 0)
		}

		var nc = c
		var s = ss[i][j]
		switch s {
		case "*":
			if t := i+1; t < len(ss) {
				if f2(c, ss[t], 0) { return true }
				if false { debug(ctx, "%v %v %v", ss, ss[t], c) }
				return f0(c, ss, t+1, 0, 0)
			} else {
				ff(c)
				return true
			}
		case "**":
			if t := len(ss)-1; i < t {
				if f2(c, ss[t], 0) { return true }
				if false { debug(ctx, "%v %v %v", ss, ss[t], c) }
				return f0(c, ss, t+1, 0, 0)
			} else {
				ff(c)
				return true
			}
		}

		if x, y := c.v[s[:k+1]]; y {
			if false { debug(ctx, "%v %v", s[:k+1], x) }
			if f0(x, ss, i, j, k+1) { return true }
		}

		if x, y := c.v["**"]; y {
			if f3(x, ss[len(ss)-1], 0) {
				if false { debug(ctx, "%v %v → %v |%v", ss[len(ss)-1], x, r[len(r)-1], s[:k+1]) }
				return true
			}
		}

		if x, y := c.v["*?"]; y {
			if f3(x, ss[i], 0) {
				if false { debug(ctx, "%v %v → %v |%v", ss[i], x, r[len(r)-1], s[:k+1]) }
				return true
			}
		}

		if checkpoints && (c.a != nil || c.o != nil || c.v != nil) &&
			strings.Contains(c.String(),sf("%v",do(ctx,ctxany{}))) {
			var cs = callstack{}
			if line_column(ctx) == "1:1" { cs.frames = -1 }
			cs.debug(ctx, _f("%v (%d.%d.%d) %s %v", ss, i, j, k, s[:k+1], c))
		}
		return f0(nc, ss, i, j, k+1)
	}

	f2 = func(c *valcache, s []string, j int, k ...string) (res bool) {
		if l := len(s); j == l {
			return fullmatch(c, left)
		} else if x, y := c.v[s[j]]; y {
			if f2(x, s, j+1, append(k,s[j])...) { return true }
		} else if j == 0 && k != nil && s[0] == k[len(k)-1] {
			var j = 1 // reversed-counting tail, example: [h .] [foo bar zz x . h]
			for n := len(k)-1; j < l && j <= n && s[j] == k[n-j] ; j += 1 {}
			if j == l { return fullmatch(c, left) }
		}
		for _, o := range c.o {
			if f2(c.v[o], s, j, append(k,o)...) { res = true; if false { break }}
		}
		return
	}

	f3 = func(c *valcache, s []string, j int, k ...string) (res bool) {
		if l := len(s); j == l {
			return fullmatch(c, left)
		} else if x, y := c.v[s[j]]; y {
			return f3(x, s, j+1, append(k,s[j])...)
		} else if j == 0 && k != nil && s[0] == k[len(k)-1] {
			var j = 1 // reversed-counting tail, example: [h .] [foo bar zz x . h]
			for n := len(k)-1; j < l && j <= n && s[j] == k[n-j] ; j += 1 {}
			if j == l { return fullmatch(c, left) }
		}
		return
	}

	ff = func(c *valcache) {
		for _, k := range c.o { _c := c.v[k]
			fullmatch(_c, right)
			ff(_c)
		}
	}

	if f0(_c, ss, 0, 0, 0) {
		if checkpoints {}
	} else {
		if checkpoints {}
	}
	return
}

type uncache_t struct{ Context ; a []any }
func (u *uncache_t) do(ctx Context, op any) (res any) {
	switch t := op.(type) {
	case property: if t&(propUncache) != 0 { return true }
	case hit_s: return uncache(ctx, t.valcache, t.s)
	case matched_filemap, matched_rule:
		u.a = append(u.a, t)
		return len(u.a)
	}
	return u.Context.do(ctx, op)
}

type ctxany struct{}
type hit_ctx struct{ Context ; any }
func (p *hit_ctx) do(ctx Context, op any) any {
	switch op.(type) {
	case ctxany: return p.any
	}
	return p.Context.do(ctx, op)
}

func seperate(s, sep string) (ss []string) {
	for i, s := range strings.Split(s, ".") {
		if 0 == i && s == "" { continue }
		if 0 < i { ss = append(ss, ".") }
		ss = append(ss, s)
	}
	return
}

func vcs(a any) string {
	switch t := a.(type) {
	case filemap: return t.String()
	}
	return fmt.Sprintf("%v", a)
}

func c_in(k string, s ...string) bool {
	for _, s := range s { if strings.Contains(s,k) { return true } }
	return false
}

type matched_filemap struct{ filemap ; string }
type matched_rule struct{ *rule ; string }
func (p matched_filemap) String() string { return "{"+p.filemap.String()+" name="+p.string+"}" }
func (p matched_rule) String() string { return "{"+p.rule.String()+" name="+p.string+"}" }

func (p *valcache) km() (ss map[string]struct{}) {
	ss = make(map[string]struct{})
	for _, v := range reflect.ValueOf(p.v).MapKeys() {
		ss[fmt.Sprintf("%s", v.Interface())] = struct{}{}
	}
	return
}
func (p *valcache) keys() []string { return p.o }
func (p *valcache) ks() string { return "["+strings.Join(p.o," ")+"]" }
func (p *valcache) String() (s string) { // NOTE: for debug
	// switch len(p.a) {
	// case 1 : BUG: s += fmt.Sprintf("%v", vcs(p.a[0]))
	// default: for i, v := range p.a { s += fmt.Sprintf("%v:%v,", i, vcs(v)) }
	// }
	for i, v := range p.a { s += fmt.Sprintf("%v:%v,", i, vcs(v)) }
	for _, k := range p.o { s += fmt.Sprintf("%v:%v,", k, p.v[k]) }
	if s != "" { s = s[:len(s)-1] } // strings.TrimSuffix(s, ",")
	return "{"+s+"}"
}
func (p *valcache) k(s []string, j int) (_ string, _ bool) {
	for _, o := range p.o {
		var i, l = 0, len(o)
		for n := j; n < len(s); n += 1 {
			if t := s[n]; t == "?" {
				i += 1
			} else if i < l && strings.HasPrefix(o[i:], t) {
				i += len(t)
			} else {
				break
			}
		}
		if i == l { return o, true }
	}
	return
}
func (p *valcache) u(s []string, j int) (_ *valcache, _ bool) {
	if x, y := p.v[s[j]]; y && x != nil { return x, y }
	if k, y := p.k(s,j); y { x, y := p.v[k]; return x, y }
	return
}

func do_hit(c Context, a any) (r []*valcache) { r, _ = do(c,a).([]*valcache); return }
func hit(ctx Context, c *valcache, k any) []*valcache { return _hit(ctx, c, k) }
func hits(c *valcache, _s ...string) hit_s {
	var ss [][]string
	for _, s := range _s { ss = append(ss, seperate(s, ".")) }
	return hit_s{c, ss}
}
func hitg(ctx Context, c *valcache, g *globpat) hit_s {
	var ( s []string ; ss [][]string )
elems_loop:
	for i, e := range g.elems {
		switch t := e.(type) {
		case *globrange:
			debug(pc(ctx,e), "TODO: %v", t, trace{})
		case *globmeta:
			switch t.token {
			case ASTQ: // *?
				for n := i+1; n < len(g.elems); n += 1 {
					var t = __string(ctx, g.elems[n])
					for m := 0; m < len(t); m += 1 { s = append(s, string(t[m])) }
				}
				ss = append(ss, []string{"*?"}, s)
				break elems_loop
			case DAST: // **
				for n := len(g.elems)-1; i < n; n -= 1 {
					var t = __string(ctx, g.elems[n])
					for m := len(t)-1; 0 <= m; m -= 1 { s = append(s, string(t[m])) }
				}
				ss = append(ss, []string{"**"}, s)
				break elems_loop
			case SAST: // *
				for n := len(g.elems)-1; i < n; n -= 1 {
					var t = __string(ctx, g.elems[n])
					for m := len(t)-1; 0 <= m; m -= 1 { s = append(s, string(t[m])) }
				}
				ss = append(ss, []string{"*"}, s)
				break elems_loop
			case QUE: // ?    s = append(s, "?")
				if i := len(ss)-1; i == -1 {
					ss = append(ss, []string{"?"})
				} else {
					ss[i] = append(ss[i], "?")
				}
			default:
				debug(pc(ctx,e), "TODO: %v %v", t.token, c, trace{})
			}
		default:
			if t := __string(ctx, t); i == 0 {
				ss = append(ss, append(s, t))
			} else {
				for _, r := range t { s = append(s, string(r)) }
				ss = append(ss, s)
			}
			s = nil
		}
	}
	return hit_s{c, ss}
}
func hitp(ctx Context, c *valcache, p *path) hit_s {
	var ss [][]string
	for _, e := range p.elems {
		switch t := unbox(e).(type) {
		case *globpat: ss = append(ss, hitg(ctx, c, t).s...)
		default: ss = append(ss, hits(c, __string(ctx, t)).s...)
		}
	}
	return hit_s{c, ss}
}
func _hit(ctx Context, c *valcache, k any) (r []*valcache) {
	if checkpoints { defer func(s string) {
		switch {
		case truly(ctx, propCache): check_cache(ctx, k, s, c, r)
		case truly(ctx, propUncache): check_uncache(ctx, k, s, c, r)
		default: debug(ctx, "%v %v", k, c, trace{})
		}
	}(c.String())}
	switch t := k.(type) {
	case  nil : return
	case *loc : return _hit(ctx, c, t.Value)
	case *file: return _hit(ctx, c, t.name)
	case *rule: return _hit(ctx, c, t.target)
	case *argumented: return _hit(ctx, c, t.Value)
	case   string : return do_hit(&hit_ctx{ctx,t}, hits(c, strings.Split(t, pathSep)...))
	case []string : return do_hit(&hit_ctx{ctx,t}, hits(c, t...))
	case *closure : return do_hit(&hit_ctx{ctx,t}, hits(c, "&"))
	case *percpat : return do_hit(&hit_ctx{ctx,t}, hits(c))
	case *regexpat: return do_hit(&hit_ctx{ctx,t}, hits(c))
	case *globpat : return do_hit(&hit_ctx{ctx,t}, hitg(ctx,c,t))
	case *path    : return do_hit(&hit_ctx{ctx,t}, hitp(ctx,c,t))
	case *strval  : return do_hit(ctx, hits(c, `{`+__string(ctx,t)+`}`))
	case *strlit  : return do_hit(ctx, hits(c, `'`+__string(ctx,t.s)+`'`))
	case *strcomp : return do_hit(ctx, hits(c, `"`+__string(ctx,t)+`"`))
	}
	if true {
		return _hit(ctx, c, __string(ctx, k))
	} else {
		var t = __string(ctx, k)
		return do_hit(&hit_ctx{ctx,t}, hits(c, strings.Split(t, pathSep)...))
	}
}

func map_files(ctx Context, p *project, patts, paths []Value) (res []filemap) {
	var base = &_filemap{p, patts, paths}
	for _, patt := range patts {
		switch patt.(type) {
		case *valbase, *null, *none:
			continue
		}
		if c := hit(cache_t{ctx}, &p.filemap, patt); c != nil {
			switch len(c) {
			case 1:
				c0, f := c[0], filemap{base, patt}
				c0.a, res = append(c0.a, f), append(res, f)
			default:
				debug(pc(ctx,patt), "too many cached: %v %v", ts(patt), c, trace{})
			}
		} else {
			debug(pc(ctx,patt), "cache failed: %v", ts(patt), trace{})
		}
    }
    return
}

func map_entry(ctx Context, p *project, target Value, prog *program) (entry entry) {
	var patterned = patterned(ctx, target)
	if !patterned {
		switch target.(type) {
		case *barefile, *file, *path, *percpat, *globpat, *regexpat:
			goto skip_file
		}
		if t := p.file(unmap_uncheck_ctx{ctx}, target); t != nil {
			target, t.position = t, target.Position()
		}
	skip_file:
	}

    var args []Value // e.g. for pattern filtering
	switch t := target.(type) {
	case *argumented: target, args = t.Value, merge(t.args...)
	case *group: debug(ctx, "not supported target: %v", t, trace{})
	}

	if c := hit(cache_t{ctx}, &p.entries, target); c == nil {
		debug(ctx, "uncachable for: %v | %s", target, ts(target), trace{})
	} else {
		switch len(c) {
		case 1:
			r := &rule{target:target, arged:args, program:[]*program{prog}}
			if patterned { p.patterns = append(p.patterns, r) }
			if p.main == nil { p.main = r }
			c[0].a = append(c[0].a, r)
			entry = r
		default:
			debug(pc(ctx,target), "too many cached: %v %v", ts(target), c, trace{})
		}
	}
	return
}

func unmap[T any](ctx Context, c *valcache, key any) (res []T) {
	var u = &uncache_t{ctx, nil}
	var x = hit(u, c, key)
	for _, a := range u.a {
		switch t := a.(type) {
		case T : res = append(res, t)
		default: debug(ctx, "%v %v", ts(key), ts(a), trace{})
		}
	}
	if checkpoints { check_unmap(u, key, c, x) }
    return
}

func unmap_entries(ctx Context, p *project, key any, m map[*project]struct{}) (res []entry) {
	if false && checkpoints { defer check_unmap_entries(ctx, p, key, &res) }
	if m == nil { m = map[*project]struct{}{} } else if _, y := m[p]; y { return }
	if m != nil { m[p] = struct{}{} }
	if res = unmap[entry](ctx, &p.entries, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_entries(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; false && c != nil {
		if res = unmap_entries(ctx, c, key, m); res != nil { return }
	}
	return
}

func unmap_files(ctx Context, p *project, key any, m map[*project]struct{}) (res []matched_filemap) {
	if false && checkpoints { defer check_unmap_files(ctx, p, key, &res) }
	if m == nil { m = map[*project]struct{}{} } else if _, y := m[p]; y { return }
	if m != nil { m[p] = struct{}{} }
	if res = unmap[matched_filemap](ctx, &p.filemap, key); res != nil { return }
	for _, b := range p.bases {
		if res = unmap_files(ctx, b, key, m); res != nil { return }
	}
	if c := p.configure; c != nil {
		if res = unmap_files(ctx, c, key, m); res != nil { return }
	}
	return
}
