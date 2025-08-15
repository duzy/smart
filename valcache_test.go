//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
)

func testValueCache0(ctx *testcase) {
	v := _null(_position(ctx))
	m := make(map[any]string)
	m["foo"] = "foobar"
	m['f'] = "rune(f)"
	m["f"] = "string(f)"
	m[char('f')] = "char(f)"
	m[1] = "one"
	m[v] = "value"

	if x, y := m["foo"]; !y || x != "foobar" {
		ctx.err("%v", m)
	}

	s := "foobar"[:3]
	if x, y := m[s]; !y || x != "foobar" {
		ctx.err("%v", m)
	}

	if x, y := m['f']; !y || x != "rune(f)" {
		ctx.err("%v ; %v", m, x)
	}
	if x, y := m[char('f')]; !y || x != "char(f)" {
		ctx.err("%v ; %v", m, x)
	}
	if x, y := m["f"]; !y || x != "string(f)" {
		ctx.err("%v ; %v", m, x)
	}

	if x, y := m[1]; !y || x != "one" {
		ctx.err("%v", m)
	}

	var t Value = v
	if x, y := m[v]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if x, y := m[t]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if _, y := m[_null(_position(ctx))]; y {
		ctx.err("%v", m)
	}
}

func testValueCache1(ctx *testcase) {
	testValueCache0(ctx)

	var p = _project(ctx)

	if c := &p.filemap; c.a != nil {
		ctx.err("%v", c)
	} else if len(c.words) != 1 {
		ctx.err("%v ; %v", c.words, c)
	} else if x, y := c.words["foo"]; !y {
		ctx.err("%v ; %v", c.words, c)
	} else if len(x.puncs) != 1 {
		ctx.err("%v ; %v", x.puncs, x)
	} else if len(x.words) != 1 {
		ctx.err("%v ; %v", x.words, x)
	} else {
		if z, y := x.puncs[DOT]; !y {
			ctx.err("%v", x)
		} else if len(z.words) != 2 {
			ctx.err("%v", z)
		} else {
			if t, y := z.words["c"]; !y {
				ctx.err("%v", z)
			} else if len(t.a) != 1 || t.words != nil {
				ctx.err("%v", t)
			} else if slot, y := t.a[0].(filemap_slot); !y {
				ctx.err("%v", tst{t.a[0]})
			} else if slot.String() != "foo.c" {
				ctx.err("%v", tst{t.a[0]})
			}

			if t, y := z.words["c++"]; !y {
				ctx.err("%v", z)
			} else if len(t.a) != 1 || t.words != nil {
				ctx.err("%v", t)
			} else if slot, y := t.a[0].(filemap_slot); !y {
				ctx.err("%v", tst{t.a[0]})
			} else if slot.String() != "foo.c++" {
				ctx.err("%v", tst{t.a[0]})
			}
		}

		if z0, y := x.words["bar"]; !y {
			ctx.err("%v", x)
		} else if len(z0.puncs) != 1 {
			ctx.err("%v", z0)
		} else if z, y := z0.puncs[DOT]; !y {
			ctx.err("%v", z0)
		} else if len(z.words) != 2 {
			ctx.err("%v", z.words)
		} else if t, y := z.words["c"]; !y {
			ctx.err("%v", z)
		} else if len(t.a) != 1 || t.words != nil {
			ctx.err("%v", t)
		} else if slot, y := t.a[0].(filemap_slot); !y {
			ctx.err("%v", tst{t.a[0]})
		} else if slot.String() != "foo/bar.c" {
			ctx.err("%v", tst{t.a[0]})
		} else if t, y := z.words["c++"]; !y {
			ctx.err("%v", z)
		} else if len(t.a) != 1 || t.words != nil {
			ctx.err("%v", t)
		} else if slot, y := t.a[0].(filemap_slot); !y {
			ctx.err("%v", tst{t.a[0]})
		} else if slot.String() != "foo/bar.c++" {
			ctx.err("%v", tst{t.a[0]})
		}
	}

	if v, pat := ctx.val("p1"), "*.c"; v == nil {
		ctx.err("%v", _project(ctx))
	} else if s := v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", tst{v}, s, pat)
	} else if m := unmap_files(ctx, v); m == nil {
		ctx.err("unmap_files: %v", tst{v})
	} else if len(m) != 1 {
		ctx.err("%v : %v", v, m)
	} else if m[0].name != "foo.c" {
		ctx.err("%v : %v", v, m[0])
	} else if s := ts(m[0].pattern); s != "{=compound {=word foo} {=punct .} {=word c}}" {
		ctx.err("%v : %v : %v", v, m[0].pattern, s)
	}

	if v, pat := ctx.val("p2"), "**.c"; v == nil {
		ctx.err("%v", _project(ctx))
	} else if s := v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", tst{v}, s, pat)
	} else if m := unmap_files(ctx, v); m == nil {
		ctx.err("unmap_files: %v", tst{v})
	} else if len(m) != 2 {
		ctx.err("%v : %v", v, m)
	} else if i := 0; m[i].name != "foo.c" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=compound {=word foo} {=punct .} {=word c}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	} else if i := 1; m[i].name != "foo/bar.c" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=path {=word foo} {=compound {=word bar} {=punct .} {=word c}}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	}

	if v, pat := ctx.val("p3"), "**.c++"; v == nil {
		ctx.err("%v", _project(ctx))
	} else if s := v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", tst{v}, s, pat)
	} else if m := unmap_files(ctx, v); m == nil {
		ctx.err("unmap_files: %v", tst{v})
	} else if len(m) != 2 {
		ctx.err("%v : %v", v, m)
	} else if i := 0; m[i].name != "foo.c++" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=compound {=word foo} {=punct .} {=word c++}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	} else if i := 1; m[i].name != "foo/bar.c++" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=path {=word foo} {=compound {=word bar} {=punct .} {=word c++}}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	}
}

func testValueCache2(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil universe")
	} else if c := &p.filemap; c.a != nil {
		ctx.err("%v", c)
	} else if len(c.globs) != 3 {
		ctx.err("%v", c.globs)
	} else {
		if x, y := c.globs["*.c++"]; !y {
			ctx.err("%v", c.globs)
		} else if x.words != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap_slot); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if slot.String() != "*.c++" {
			ctx.err("%v", tst{x.a[0]})
		}

		if x, y := c.globs["**.c"]; !y {
			ctx.err("%v", c.globs)
		} else if x.words != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap_slot); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if slot.String() != "**.c" {
			ctx.err("%v", tst{x.a[0]})
		}

		if x, y := c.globs["???"]; !y {
			ctx.err("%v", c.globs)
		} else if x.words != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap_slot); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if slot.String() != "{=glob ???}" {
			ctx.err("%v", tst{x.a[0]})
		}

		if x, y := c.words["foo"]; !y {
			ctx.err("%v", c.words)
		} else if x.words != nil {
			ctx.err("%v", x.words)
		} else if len(x.globs) != 6 {
			ctx.err("%v", x.globs)
		} else {
			if c, y := x.globs["*.c++"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*.c++" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.globs["*.xx"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*.xx" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.globs["*.yy"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*.yy" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.globs["*zzz"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*zzz" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.globs["**z"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil || c.globs != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/**z" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.globs["??"]; !y {
				ctx.err("%v", x.globs)
			} else if c.a != nil {
				ctx.err("%v", c.a)
			} else if len(c.globs) != 1 {
				ctx.err("%v", c.globs)
			} else {
				if x, y := c.globs["???.c++"]; !y {
					ctx.err("%v", x.globs)
				} else if x.words != nil {
					ctx.err("%v", x)
				} else if len(x.a) != 1 {
					ctx.err("%v", c.a)
				} else if slot, y := x.a[0].(filemap_slot); !y {
					ctx.err("%v", tst{x.a[0]})
				} else if slot.String() != "foo/{=glob ??}/{=glob ???.c++}" {
					ctx.err("%v", tst{x.a[0]})
				}
			}
		}
	}

	var _fm *_filemap

	if s := "foo"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s %v ; %v", s, m, &p.filemap)
	}

	if s := "a.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m == nil {
		ctx.err("%s ; %v", s, x)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m)
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m)
	} else if m[0].String() != "{filemap=*.c++ name=a.c++}" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else if m[0].pattern.string(ctx) != "*.c++" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else {
		_fm = m[0]._filemap

		if len(_fm.paths) != 1 {
			ctx.err("%s : %v", s, _fm.paths)
		} else if _fm.paths[0].String() != "src" {
			ctx.err("%s : %v", s, _fm.paths[0])
		}

		if len(_fm.patts) != 9 {
			ctx.err("%s : %v", s, _fm.patts)
		} else if _fm.patts[0].String() != "*.c++" {
			ctx.err("%s : %v", s, _fm.patts[0])
		} else if _fm.patts[1].String() != "**.c" {
			ctx.err("%s : %v", s, _fm.patts[1])
		} else if _fm.patts[2].String() != "{=glob ???}" {
			ctx.err("%s : %v", s, _fm.patts[2])
		} else if _fm.patts[3].String() != "foo/*.c++" {
			ctx.err("%s : %v", s, _fm.patts[3])
		} else if _fm.patts[4].String() != "foo/*.xx" {
			ctx.err("%s : %v", s, _fm.patts[4])
		} else if _fm.patts[5].String() != "foo/*.yy" {
			ctx.err("%s : %v", s, _fm.patts[5])
		} else if _fm.patts[6].String() != "foo/{=glob ??}/{=glob ???.c++}" {
			ctx.err("%s : %v", s, _fm.patts[6])
		} else if _fm.patts[7].String() != "foo/*zzz" {
			ctx.err("%s : %v", s, _fm.patts[7])
		} else if _fm.patts[8].String() != "foo/**z" {
			ctx.err("%s : %v", s, _fm.patts[8])
		}
	}

	if s := "aaa.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m == nil {
		ctx.err("%s ; %v", s, x)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=*.c++ name=aaa.c++}" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else if m[0].pattern.string(ctx) != "*.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}

	if s := "a/aa.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s", s)
	}

	if s := "foo/aaa.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=foo/*.c++ name=foo/aaa.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "foo/*.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}
	if s := "foo/a/xyz.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/a/bb.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bb.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bb/cc.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bbb/ccc.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bbb.c++/ccc"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s %v", s, m)
	}
	if s := "foo/ab/xyz.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=foo/{=glob ??}/{=glob ???.c++} name=foo/ab/xyz.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "foo/??/???.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}
	if s := "foo/12/xyz.c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=foo/{=glob ??}/{=glob ???.c++} name=foo/12/xyz.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "foo/??/???.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}

	if s := "c"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m != nil {
		ctx.err("%s", s)
	}

	if s := "abc"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap={=glob ???} name=abc}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "???" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v ; %v %v", s, m[0], m[0]._filemap, _fm)
	}
	if s := "c++"; false {
	} else if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap={=glob ???} name=c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "???" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v ; %v %v", s, m[0], m[0]._filemap, _fm)
	}

	if s, pat := "foo/xxxzzz", "foo/*zzz"; true {
		// NOTE: this test finds if the "foo/*zzz" is applied prior to "foo/**z"
		// NOTE: test for 50 times until one err, because the order of map-keys is random.
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
				ctx.err("%s %v", s, &p.filemap) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &p.filemap) ; break
			} else if m := unmap_files(ctx, s); m == nil {
				ctx.err("%s ; %v", s, ts(ctx)) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m) ; break
			} else if m[0].name != s {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].String() != "{filemap="+pat+" name=foo/xxxzzz}" {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0].pattern.string(ctx) != pat {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0]) ; break
			}
		}
	}
	if s, pat := "foo/xx/yyz", "foo/**z"; true {
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
				ctx.err("%s %v", s, &p.filemap) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &p.filemap) ; break
			} else if m := unmap_files(ctx, s); m == nil {
				ctx.err("%s ; %v", s, ts(ctx)) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m) ; break
			} else if m[0].name != s {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].String() != "{filemap="+pat+" name=foo/xx/yyz}" {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0].pattern.string(ctx) != pat {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0]) ; break
			}
		}
	}
	if s, pat := "foo/xx/yy/zzzz", "foo/**z"; true {
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := p.filemap.hit(&unmap{ctx,nil}, s); x != nil {
				ctx.err("%s %v", s, &p.filemap) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &p.filemap) ; break
			} else if m := unmap_files(ctx, s); m == nil {
				ctx.err("%s ; %v", s, ts(ctx)) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m) ; break
			} else if m[0].name != s {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].String() != "{filemap="+pat+" name=foo/xx/yy/zzzz}" {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0].pattern.string(ctx) != pat {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0]) ; break
			}
		}
	}
}

func testValueCache3(ctx *testcase) {
	if p := _project(ctx); p == nil {
		ctx.err("nil universe")
	} else if c := &p.filemap; c.a != nil {
		ctx.err("universe valcache : %v", c)
	} else if len(c.words) != 1 {
		ctx.err("universe valcache : %v", c)
	}
}

func testValueCache(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
		return
	}

	if p.filemap.globs == nil {
		ctx.err("filemap.globs")
	} else if x, y := p.filemap.globs["*.log"]; !y {
		ctx.err("%v", p.filemap.globs)
	} else if len(x.a) != 1 {
		ctx.err("%v", x.a)
	} else if t, y := x.a[0].(filemap_slot); !y {
		ctx.err("%v", tst{(x.a[0])})
	} else if t.pattern.String() != "*.log" {
		ctx.err("%v", tst{(x.a[0])})
	} else if x, y := p.filemap.globs["**.o"]; !y {
		ctx.err("%v", p.filemap.globs)
	} else if len(x.a) != 1 {
		ctx.err("%v", x.a)
	} else if t, y := x.a[0].(filemap_slot); !y {
		ctx.err("%v", tst{(x.a[0])})
	} else if t.pattern.String() != "**.o" {
		ctx.err("%v", tst{(x.a[0])})
	}

	if p.filemap.puncs == nil {
		ctx.err("filemap.puncs")
	} else if x, y := p.filemap.puncs[DOT]; !y {
		ctx.err("%v", p.filemap.puncs)
	} else if len(x.a) != 0 || x.words == nil {
		ctx.err("%v", x)
	} else if x2, y := x.words["deps"]; !y {
		ctx.err("%v", x)
	} else if len(x2.a) != 0 || x2.words != nil || x2.globs == nil {
		ctx.err("%v", x2)
	} else if x3, y := x2.globs["??"]; !y {
		ctx.err("%v", x2)
	} else if len(x3.a) != 0 || x3.words != nil || x3.globs == nil {
		ctx.err("%v", x3)
	} else if x4, y := x3.globs["??"]; !y {
		ctx.err("%v", x3)
	} else if len(x4.a) != 0 || x4.words != nil || x4.globs == nil {
		ctx.err("%v", x4)
	} else if x5, y := x4.globs["??????????"]; !y {
		ctx.err("%v", x4)
	} else if len(x5.a) != 1 || x5.words != nil || x5.globs != nil {
		ctx.err("%v", x5)
	} else if t, y := x5.a[0].(filemap_slot); !y {
		ctx.err("%v", tst{(x5.a[0])})
	} else if t.pattern.String() != ".deps/{=glob ??}/{=glob ??}/{=glob ??????????}" {
		ctx.err("%v", tst{(x5.a[0])})
	} else if t.pattern.string(ctx) != ".deps/??/??/??????????" {
		ctx.err("%v", tst{(x5.a[0])})
	}

	if len(p.filemap.value) != 1 {
		ctx.err("%v", tst{&p.filemap})
	} else if p.filemap.value[0].Value.String() != "&(gen)" {
		ctx.err("%v", tst{p.filemap.value[0].Value})
	} else if p.filemap.value[0].valcache.String() != "{0:filemap_slot(&(gen))}" {
		ctx.err("%v", tst{p.filemap.value[0].valcache})
	}

	if p.filemap.words == nil {
		ctx.err("filemap.words")
	} else if x, y := p.filemap.words["foo"]; !y {
		ctx.err("%v", p.filemap.words)
	} else if len(x.a) != 0 || x.puncs == nil || x.words == nil {
		ctx.err("%v", x)
	} else if x21, y := x.puncs[DOT]; !y {
		ctx.err("%v", x)
	} else if len(x21.a) != 0 || x21.puncs != nil || x21.words == nil {
		ctx.err("%v", x21)
	} else if x22, y := x.words["bar"]; !y {
		ctx.err("%v", x)
	} else if len(x22.a) != 0 || x22.puncs == nil || x22.words != nil || x22.globs != nil {
		ctx.err("%v", x22)
	}

	for i, s := range []string{
		".deps/xx/yy/zzzzzzzzzz,.deps/??/??/??????????",
		"foo.log,*.log",
		"foo.o,**.o",
		"foo.c,foo.c",
		"foo.c++,foo.c++",
		// FIXME: "foo/*.c,foo/bar.c",
		// FIXME: "foo/*.c++,foo/bar.c++",
		// FIXME: "**.c",
	}{
		var t = strings.Split(s, ",")
		if a := p.unmap_files(ctx, t[0], nil); len(a) < 1 {
			ctx.err("miss cache for %d. %s %v", i, t[0], a)
		} else if len(t) == 1 && t[0] == "**.c" {
			if len(a) != 2 {
				ctx.err("miss cache for %d. %s %v", i, t[0], tst{a})
			} else if a[0].pattern.string(ctx) != "foo.c" {
				ctx.err("miss cache for %d. %s %v", i, t[0], tst{a[0]})
			} else if a[1].pattern.string(ctx) != "foo/bar.c" {
				ctx.err("miss cache for %d. %s %v", i, t[0], tst{a[1]})
			}
		} else if a[0].pattern.string(ctx) != t[1] {
			ctx.err("miss cache for %d. %s %v", i, t[0], tst{a[0]})
		}
	}

	if s := "val0" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "**.c" {
		ctx.err("%v", v)
	} else if v.string(ctx) != "**.c" {
		ctx.err("%v", v)
	} else if a := p.unmap_files(ctx, v, nil); len(a) < 1 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, a)
	} else if len(a) != 2 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a})
	} else if a[0].pattern.string(ctx) != "foo.c" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[0]})
	} else if a[1].pattern.string(ctx) != "foo/bar.c" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[1]})
	} else if t := unmap_files(ctx, v); len(t) != 2 {
		ctx.err("%v %v", v, t)
	} else if !isAny(t[0].name, "foo/bar.c", "foo.c") {
		ctx.err("%v %v", v, t[0])
	} else if !isAny(t[1].name, "foo/bar.c", "foo.c") {
		ctx.err("%v %v", v, t[1])
	}

	if s := "val1" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "**.c++" {
		ctx.err("%v", v)
	} else if v.string(ctx) != "**.c++" {
		ctx.err("%v", v)
	} else if a := p.unmap_files(ctx, v, nil); len(a) != 2 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, a)
	} else if a[0].pattern.string(ctx) != "foo.c++" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[0]})
	} else if a[1].pattern.string(ctx) != "foo/bar.c++" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[1]})
	} else if t := unmap_files(ctx, v); len(t) != 2 {
		ctx.err("%v %v", v, t)
	} else if !isAny(t[0].name, "foo/bar.c++", "foo.c++") {
		ctx.err("%v %v", v, t[0])
	} else if !isAny(t[1].name, "foo/bar.c++", "foo.c++") {
		ctx.err("%v %v", v, t[1])
	}

	if s := "val2" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.string(ctx) != "foo.c++" {
		ctx.err("%v", v)
	} else if a := p.unmap_files(ctx, v, nil); len(a) != 1 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, a)
	} else if a[0].pattern.string(ctx) != "foo.c++" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[0]})
	} else if t := unmap_files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	} else if t[0].pattern.string(ctx) != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%v %v", v, tst{v})
	} else if f.ident(ctx) != "foo.c++" {
		ctx.err("%v %v", v, f)
	}

	if s := "val3" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.string(ctx) != "foo.o" {
		ctx.err("%v %v", v, tst{v})
	} else if a := p.unmap_files(ctx, v, nil); len(a) != 1 {
		ctx.err("%v: miss cache : %v %v", v, tst{v}, a)
	} else if a[0].pattern.string(ctx) != "**.o" {
		ctx.err("%v: miss cache : %v %v", v, tst{v}, tst{a[0].pattern})
	} else if t := unmap_files(ctx, v); len(t) != 1 {
		ctx.err("%v %v ; %v", v, tst{v}, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%v %v ; %v", v, tst{v}, t[0])
	} else if t[0].String() != "{filemap=**.o name=foo.o}" {
		ctx.err("%s : %v : %v", s, t[0], t[0].pattern)
	} else if _, y := t[0].pattern.(*globpat); !y {
		ctx.err("%v %v ; %v", v, tst{v}, t[0])
	} else if t[0].project != p {
		ctx.err("%v ; %v %v %v", tst{v}, t[0], t[0].paths, t[0].project)
	} else if t[0].paths == nil {
		ctx.err("%v %v ; %v %v", v, tst{v}, t[0], t[0].paths)
	} else if t := t[0].paths[0]; t.String() != p.absPath+"/.tmp" {//"$//.tmp"
		ctx.err("%v %v", t, tst{t})
	} else if t := unmap_files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%v %v", v, t[0])
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%v ; %v", tst{v}, t)
	} else if f.ident(ctx) != "foo.o" {
		ctx.err("%v %v", v, f)
	}

	if s, p := "val4", ".deps/xx/yy/zzzzzzzzzz" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != p {
		ctx.err("%v %v", v, tst{v})
	} else if v.string(ctx) != p {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, v); len(t) != 1 {
		ctx.err("%v %v %v", v, tst{v}, t)
	}

	if s := "sources" ; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err("sources is nil")
	} else if v := evoke(ctx, d, nil, nil); v == nil {
		ctx.err("sources is wrong: %v %v", d, v)
	} else if s := v.string(ctx); strings.Count(s, "foo.c") != 2 {
		ctx.err("sources is wrong: %v", v) // NOTE: "foo.c" counts foo.c foo.c++
	} else if strings.Count(s, "foo.c++") != 1 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.c") != 2 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.c++") != 1 {
		ctx.err("sources is wrong: %v", v)
	}

	if s := "objects" ; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err("objects is nil")
	} else if v := evoke(ctx, d, nil, nil); v == nil {
		ctx.err("objects is wrong: %v %v", d, v)
	}
}
