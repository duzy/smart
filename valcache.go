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
type fullmatch *valcache
type pattmatch *valcache
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

func uncache(ctx Context, _c *valcache, ss [][]string) (r []*valcache) {
	var f0     func(*valcache, [][]string, int) (*valcache, bool)
	var f1     func(*valcache,   []string, int) (*valcache, bool)
	var f2, F2 func(*valcache,   []string, int) (*valcache, bool) // patterns: ** *?
	var f3, F3 func(*valcache,     string, int) (*valcache, bool)
	var ff     func(*valcache)
	var dd = false && sf("%v %v", _c, do(ctx, ctxany{})) == "{foo:{b:{*:{v:{*:{h:{.:{0:foo/b*/v*.h}}}}}},x:{*:{h:{.:{y:{0:foo/x*y.h}}}}}},f:{*?:{x:{.:{h:{0:f*?/x.h}}}}}} foo/bar/v1.h"
	if false && dd { debug(ctx, "%v %v", ss, _c) }

	f0 = func(c *valcache, ss [][]string, i int) (_ *valcache, _ bool) {
		if dd { defer debug(ctx, "%v %v %v", ss, i, c, callstack{stop:"smart._hit"}) }
		if i == len(ss) {
			if c != nil && truly(ctx, fullmatch(c)) { r = append(r, c)
				return c, true
			} else {
				return c, false
			}
		}
		var nc = c
		if x, y := f1(c, ss[i], 0); x != nil { nc = x; if y { return x, y }}
		if x, y := c.v["**"]; y && x != nil {
			for c, n := x, len(ss)-1; i <= n; n -= 1 {
				if x, y = f2(c, ss[n], 0); x != nil { c = x; if y { return x, y }}
			}
		}
		if x, y := c.v["*?"]; y && x != nil {
			for c, n := x, i; n < len(ss); n += 1 {
				if x, y = f2(c, ss[n], 0); x != nil { c = x; if y { return x, y }}
			}
		}
		if x, y := c.v["&"]; y && x != nil {
			if truly(ctx, fullmatch(x)) { r = append(r, x); return x, true }
		}
		return f0(nc, ss, i+1)
	}

	f1 = func(c *valcache, s []string, j int) (_ *valcache, _ bool) {
		if dd { defer debug(ctx, "%v %v %v", s, j, c, callstack{stop:"smart._hit"}) }
		if j == len(s) {
			if c != nil && truly(ctx, fullmatch(c)) { r = append(r, c)
				return c, true
			} else {
				return c, false
			}
		}
		var t = s[j]
		switch t {
		case "*", "**":
			ff(c)
			return c, true
		}
		if x, y := c.v[t]; y {
			if x, y = f1(x, s, j+1); y && x != nil {
				return x, true
			} else if false {
				return x, false
			}
		}
		for _, o := range c.o {
			var i, lo = 0, len(o)
			for n := j; n < len(s); n += 1 {
				if t := s[n]; t == "?" {
					i += 1
				} else if i < lo && strings.HasPrefix(o[i:], t) {
					i += len(t)
				} else if lo < len(t) && strings.HasPrefix(t, o) {
					if x, y := c.v[o]; y {
						if x, y = F3(x, t, lo); x != nil {
							if o == "b" && sf("%v",ss) == `[[foo] [bar] [v1 . h]]` {
								debug(ctx, "%v %v", s, x, callstack{frames:-1})
							}
							if !y { x, y = f1(x, s, n+1) }
							if y && x != nil { return x, true }
							if false && x != nil { return x, false }
						}
					}
					break
				} else {
					break
				}
			}
			if i == lo {
				if x, y := c.v[o]; y {
					if x, y = f1(x, s, j+1); y && x != nil {
						return x, true
					} else if true && x != nil {
						return x, false
					}
				}
			}
		}
		if x, y := c.v["*"] ; y && x != nil {
			return f1(x, s, j)
		} else {
			return c, false
		}
	}

	f2 = func(c *valcache, s []string, j int) (_ *valcache, _ bool) {
		if dd { defer debug(ctx, "%v %v %v", s, j, c, callstack{stop:"smart._hit"}) }
		if j == len(s) { return F2(c, s, j-2) }
		if x, y := f3(c, s[j], len(s[j])-1); y {
			return x, true
		} else if x != nil {
			return f2(x, s, j+1)
		} else {
			return c, false
		}
	}
	F2 = func(c *valcache, s []string, j int) (_ *valcache, _ bool) {
		if dd { defer debug(ctx, "%v %v %v", s, j, c, callstack{stop:"smart._hit"}) }
		if j < 0 {
			if c != nil && truly(ctx, fullmatch(c)) { r = append(r, c)
				return c, true
			} else {
				return c, false
			}
		}
		if x, y := f3(c, s[j], len(s[j])-1); y {
			return x, true
		} else if x != nil {
			return F2(x, s, j-1)
		} else {
			return c, false
		}
	}
	f3 = func(c *valcache, s string, k int) (_ *valcache, _ bool) {
		if dd { defer debug(ctx, "%v %v %v", s, k, c, callstack{stop:"smart._hit"}) }
		if k == -1 { return F3(c, s, 1) }
		var x, y = c.v[string(s[k])]
		if !y || x == nil { x, y = c.v["?"] }
		// if !y || x == nil { x, y = c.v["*"]
		// 	if y && x != nil { return f3(x, s, -1) }
		// }
		if y && x != nil { return f3(x, s, k-1) }
		return c, false
	}
	F3 = func(c *valcache, s string, k int) (_ *valcache, _ bool) {
		if dd { defer debug(ctx, "%v %v %v", s, k, c, callstack{stop:"smart._hit"}) }
		if k == len(s) {
			if c != nil && truly(ctx, fullmatch(c)) { r = append(r, c)
				return c, true
			} else {
				return c, false
			}
		}
		var x, y = c.v[string(s[k])]
		if !y || x == nil { x, y = c.v["?"] }
		if !y || x == nil { x, y = c.v["*"] ;
			if false && y && x != nil { return F3(x, s, len(s)) } else { k = len(s)-1 }
		}
		if y && x != nil { return F3(x, s, k+1) }
		return c, false
	}

	ff = func(c *valcache) {
		for _, k := range c.o { v := c.v[k]
			if truly(ctx, pattmatch(v)) { r = append(r, v) }
			ff(v)
		}
	}

	x, y := f0(_c, ss, 0)
	if y {
		if false && checkpoints {
			if r == nil || x == nil {
				debug(ctx, "%v %v → %v", ss, _c, x) // FIXME
			}
		}
		if false { debug(ctx, "%v %v %v", ss, x, r, callstack{stop:"smart._hit"}) }
		if false && checkpoints {
			switch len(r) {
			case 0: debug(ctx, "%v", x, trace{})
			case 1: if x != r[0] { debug(ctx, "%v != %v", x, r[0], trace{}) }
			default:
				if true { var t bool
					for _, r := range r { if t = x == r; t { break }}
					if !t { debug(ctx, "%v != %v", x, r, trace{}) }
				}
			}
		}
	}
	if checkpoints {
		switch sf("%v %v %v", ss, x, y) {
		case "[[foo] [bar] [v1 . h]]":
			debug(ctx, "%v → %v %v → %v", ss, x, y, r, callstack{frames:-1})
		}
	}
	return
}

func uncache_match(ctx Context, pat, val any) (ok bool, s string) {
	var ( f bool ; t any )
	if !patterned(ctx, pat) && patterned(ctx, val) {
		f, t, _ = match(ctx, val, pat)
	} else {
		f, t, _ = match(ctx, pat, val)
	}
	if f {
		switch t := t.(type) {
		case   string: s = t
		case []string: s = strings.Join(t, pathSep)
		default: debug(ctx, "%v %v: %v", pat, val, ts(t), trace{})
		}
		ok = true
	}
	return
} 

type uncache_t struct{ Context ; a []any }
func (u *uncache_t) do(ctx Context, op any) (res any) {
	switch t := op.(type) {
	case property: if t&(propUncache) != 0 { return true }
	case hit_s: return uncache(ctx, t.valcache, t.s)
	case fullmatch:
		var ( ca = do(ctx, ctxany{}); ok bool )
		for _, a := range t.a {
			switch a := a.(type) {
			case filemap:
				var _a = filemap{a._filemap, expand(_final(ctx), a.pattern)}
				for _, v := range merge(_a.pattern) {
					if f, s := uncache_match(ctx, v, ca); f {
						u.a, ok = append(u.a, matched_filemap{_a, s}), true
					}
				}
			case *rule:
				var _a = &rule{expand(_final(ctx), a.target), a.arged, a.program}
				for _, v := range merge(_a.target) {
					if f, s := uncache_match(ctx, v, ca); f {
						u.a, ok = append(u.a, matched_rule{_a, s}), true
					}
				}
			default:
				debug(ctx, "%v", ts(a), trace{})
			}
			if ok { break }
		}
		return ok
	case pattmatch:
		var ( ca = do(ctx, ctxany{}); ok bool )
		for _, a := range t.a {
			switch a := a.(type) {
			case filemap:
				var _a = filemap{a._filemap, expand(_final(ctx), a.pattern)}
				for _, v := range merge(_a.pattern) {
					if f, s := uncache_match(ctx, ca, v); f {
						u.a, ok = append(u.a, matched_filemap{_a, s}), true
					}
				}
			case *rule:
				var _a = &rule{expand(_final(ctx), a.target), a.arged, a.program}
				for _, v := range merge(_a.target) {
					if f, s := uncache_match(ctx, ca, v); f {
						u.a, ok = append(u.a, matched_rule{_a, s}), true
					}
				}
			default:
				debug(ctx, "%v", ts(a), trace{})
			}
			if ok { break }
		}
		return ok
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
		default: debug(ctx, "%v: %v", ts(key), ts(a), trace{})
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
