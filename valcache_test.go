//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
)

func testValueCache0(ctx *testcase) {
	v := makeNull(ctx.Position())
	m := make(map[interface{}]string)
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

	var tv Value = v
	if x, y := m[v]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if x, y := m[tv]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if _, y := m[makeNull(ctx.Position())]; y {
		ctx.err("%v", m)
	}
}

func testValueCache1(ctx *testcase) { testValueCache0(ctx)
	var u = _universe(ctx)

	if u == nil {
		ctx.err("nil universe")
	} else if c := &u.filemaps; c.a != nil {
		ctx.err("universe filecache.a : %v", c)
	} else if len(c.m) != 1 {
		ctx.err("universe filecache.m : %v", c)
	} else if x, y := c.m["foo"]; !y {
		ctx.err("universe filecache.m[foo] : %v", c.m)

	} else if len(x.m) != 2 {
		ctx.err("filecache.m : %v", x.m)

	} else if z, y := x.m[DOT]; !y {
		ctx.err("filecache.m[DOT] : %v", x)
	} else if len(z.m) != 2 {
		ctx.err("filecache.m : %v", z)
	} else if t, y := z.m["c"]; !y {
		ctx.err("filecache.m[c] : %v", z)
	} else if len(t.a) != 1 || t.m != nil {
		ctx.err("filecache.m[c] : %v", t)
	} else if t.a[0].pattern.String() != "foo.c" {
		ctx.err("filecache.m[c] : %v", t)
	} else if t, y := z.m["c++"]; !y {
		ctx.err("filecache.m[c++] : %v", z)
	} else if len(t.a) != 1 || t.m != nil {
		ctx.err("filecache.m[c++] : %v", t)
	} else if t.a[0].pattern.String() != "foo.c++" {
		ctx.err("filecache.m[c++] : %v", t)

	} else if z0, y := x.m["bar"]; !y {
		ctx.err("filecache.m[bar] : %v", x)
	} else if len(z0.m) != 1 {
		ctx.err("filecache.m : %v", z0)
	} else if z, y := z0.m[DOT]; !y {
		ctx.err("filecache.m[DOT] : %v", z0)
	} else if len(z.m) != 2 {
		ctx.err("filecache.m : %v", z.m)
	} else if t, y := z.m["c"]; !y {
		ctx.err("filecache.m[c] : %v", z)
	} else if len(t.a) != 1 || t.m != nil {
		ctx.err("filecache.m[c] : %v", t)
	} else if t.a[0].pattern.String() != "foo/bar.c" {
		ctx.err("filecache.m[c] : %v", t)
	} else if t, y := z.m["c++"]; !y {
		ctx.err("filecache.m[c++] : %v", z)
	} else if len(t.a) != 1 || t.m != nil {
		ctx.err("filecache.m[c++] : %v", t)
	} else if t.a[0].pattern.String() != "foo/bar.c++" {
		ctx.err("filecache.m[c++] : %v", t)
	}

	if s, pat := "p1", "*.c"; true {
		note(ctx, "TODO: unmapfiles %v", pat).debug(1)
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v : %v", ctx.project(), s)
	} else if s = v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", ust{v}, s, pat)
	} else if m := unmapfiles(ctx, v); m == nil {
		ctx.err("unmapfiles %v", ust{v})
	} else if len(m) != 1 {
		ctx.err("unmapfiles %v : %v", v, m)
	}

	if s, pat := "p2", "**.c"; true {
		note(ctx, "TODO: unmapfiles %v", pat).debug(1)
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v : %v", ctx.project(), s)
	} else if s = v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", ust{v}, s, pat)
	} else if m := unmapfiles(ctx, v); m == nil {
		ctx.err("unmapfiles %v", ust{v})
	} else if len(m) != 2 {
		ctx.err("unmapfiles %v : %v", v, m)
	}

	if s, pat := "p3", "**.c++"; true {
		note(ctx, "TODO: unmapfiles %v", pat).debug(1)
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v : %v", ctx.project(), s)
	} else if s = v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", ust{v}, s, pat)
	} else if m := unmapfiles(ctx, v); m == nil {
		ctx.err("unmapfiles %v", ust{v})
	} else if len(m) != 2 {
		ctx.err("unmapfiles %v : %v", v, m)
	}
}

func testValueCache2(ctx *testcase) {
	var u = _universe(ctx)
	if u == nil {
		ctx.err("nil universe")
	} else if c := &u.filemaps; c.a != nil {
		ctx.err("universe filecache.a : %v", c)
	} else if len(c.m) != 2 {
		ctx.err("universe filecache.m : %v", c)
	} else if g, y := c.m[globpat_t]; !y {
		ctx.err("universe filecache.m : %v", c.m)
	} else if len(g.m) != 3 {
		ctx.err("filecache.m : %v", g.m)
	} else if x, y := g.m["*.c++"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.a : %v", x.a)
	} else if x.a[0].pattern.String() != "*.c++" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := g.m["**.c"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.m : %v", x.a)
	} else if x.a[0].pattern.String() != "**.c" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := g.m["???"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.m : %v", x.a)
	} else if x.a[0].pattern.String() != "{=glob ???}" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x.a[0].pattern.string(ctx) != "???" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := c.m["foo"]; !y {
		ctx.err("filecache.m : %v", c.m)
	} else if len(x.m) != 1 {
		ctx.err("filecache.m : %v", x.m)
	} else if g, y := x.m[globpat_t]; !y {
		ctx.err("filecache.m : %v", x.m)
	} else if len(g.m) != 6 {
		ctx.err("filecache.m : %v", g.m)
	} else if x, y := g.m["*.c++"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.a : %v", x.a)
	} else if x.a[0].pattern.String() != "foo/*.c++" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := g.m["*.xx"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.a : %v", x.a)
	} else if x.a[0].pattern.String() != "foo/*.xx" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := g.m["*.yy"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.a : %v", x.a)
	} else if x.a[0].pattern.String() != "foo/*.yy" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := g.m["*zzz"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.a : %v", x.a)
	} else if x.a[0].pattern.String() != "foo/*zzz" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := g.m["**z"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.a : %v", x.a)
	} else if x.a[0].pattern.String() != "foo/**z" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x, y := g.m["??"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.a) != 0 {
		ctx.err("filecache.a : %v", x.a)
	} else if len(x.m) != 1 {
		ctx.err("filecache.m : %v", x.m)
	} else if g, y := x.m[globpat_t]; !y {
		ctx.err("filecache.m : %v", x.m)
	} else if len(g.a) != 0 {
		ctx.err("filecache.a : %v", g.a)
	} else if len(g.m) != 1 {
		ctx.err("filecache.m : %v", g.m)
	} else if x, y := g.m["???.c++"]; !y {
		ctx.err("filecache.m : %v", g.m)
	} else if len(x.m) != 0 {
		ctx.err("filecache.m : %v", x.m)
	} else if len(x.a) != 1 {
		ctx.err("filecache.a : %v", x.a)
	} else if x.a[0].pattern.String() != "foo/{=glob ??}/{=glob ???.c++}" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	} else if x.a[0].pattern.string(ctx) != "foo/??/???.c++" {
		ctx.err("filecache.a[0] : %v", x.a[0])
	}

	var _fm *_filemap

	if s := "a.c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("unmapfiles %s", s)
	} else if len(m) != 1 {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.String() != "*.c++" {
		ctx.err("unmapfiles %s : %v", s, m[0].pattern)
	} else if m[0].pattern.string(ctx) != "*.c++" {
		ctx.err("unmapfiles %s : %v", s, m[0].pattern)
	} else if _fm = m[0]._filemap ; len(_fm.paths) != 1 {
		ctx.err("unmapfiles %s : %v", s, _fm.paths)
	} else if _fm.paths[0].String() != "src" {
		ctx.err("unmapfiles %s : %v", s, _fm.paths[0])
	} else if len(_fm.patts) != 9 {
		ctx.err("unmapfiles %s : %v", s, _fm.patts)
	} else if _fm.patts[0].String() != "*.c++" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[0])
	} else if _fm.patts[1].String() != "**.c" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[1])
	} else if _fm.patts[2].String() != "{=glob ???}" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[2])
	} else if _fm.patts[3].String() != "foo/*.c++" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[3])
	} else if _fm.patts[4].String() != "foo/*.xx" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[4])
	} else if _fm.patts[5].String() != "foo/*.yy" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[5])
	} else if _fm.patts[6].String() != "foo/{=glob ??}/{=glob ???.c++}" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[6])
	} else if _fm.patts[7].String() != "foo/*zzz" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[7])
	} else if _fm.patts[8].String() != "foo/**z" {
		ctx.err("unmapfiles %s : %v", s, _fm.patts[8])
	}
	if s := "aaa.c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("unmapfiles %s", s)
	} else if len(m) != 1 {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.string(ctx) != "*.c++" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0]._filemap != _fm {
		ctx.err("unmapfiles %s : %v", s, m)
	}
	if s := "a/aa.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}

	if s := "foo/aaa.c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("unmapfiles %s", s)
	} else if len(m) != 1 {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.String() != "foo/*.c++" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.string(ctx) != "foo/*.c++" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0]._filemap != _fm {
		ctx.err("unmapfiles %s : %v", s, m)
	}
	if s := "foo/a/xyz.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}
	if s := "foo/a/bb.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}
	if s := "foo/aa/bb.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}
	if s := "foo/aa/bb/cc.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}
	if s := "foo/aa/bbb/ccc.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}
	if s := "foo/aa/bbb.c++/ccc"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}
	if s := "foo/ab/xyz.c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("unmapfiles %s", s)
	} else if len(m) != 1 {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.String() != "foo/{=glob ??}/{=glob ???.c++}" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.string(ctx) != "foo/??/???.c++" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0]._filemap != _fm {
		ctx.err("unmapfiles %s : %v", s, m)
	}
	if s := "foo/12/xyz.c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("unmapfiles %s", s)
	} else if len(m) != 1 {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.string(ctx) != "foo/??/???.c++" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0]._filemap != _fm {
		ctx.err("unmapfiles %s : %v", s, m)
	}

	if s := "c"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("unmapfiles %s", s)
	}
	if s := "abc"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("unmapfiles %s", s)
	} else if len(m) != 1 {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.String() != "{=glob ???}" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.string(ctx) != "???" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0]._filemap != _fm {
		ctx.err("unmapfiles %s : %v", s, m)
	}
	if s := "c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("unmapfiles %s", s)
	} else if len(m) != 1 {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.String() != "{=glob ???}" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0].pattern.string(ctx) != "???" {
		ctx.err("unmapfiles %s : %v", s, m)
	} else if m[0]._filemap != _fm {
		ctx.err("unmapfiles %s : %v", s, m)
	}

	if s := "foo/xxxzzz"; true {
		// NOTE: this test finds if the "foo/*zzz" is applied prior to "foo/**z"
		// NOTE: test for 50 times until one err, because the order of map-keys is random.
		for i := 0 ; i < 50 ; i += 1 {
			if m := unmapfiles(ctx, s); m == nil {
				ctx.err("unmapfiles %s", s) ; break
			} else if len(m) != 1 {
				ctx.err("unmapfiles %s : %v", s, m)
			} else if m[0].name != s {
				ctx.err("unmapfiles %s : %v", s, m)
			} else if m[0].pattern.String() != "foo/*zzz" {
				ctx.err("unmapfiles %s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0].pattern.string(ctx) != "foo/*zzz" {
				ctx.err("unmapfiles %s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("unmapfiles %s : %v", s, m)
			}
		}
	}

	if s := "foo/xx/yyz"; true {
		for i := 0 ; i < 50 ; i += 1 {
			if m := unmapfiles(ctx, s); m == nil {
				ctx.err("unmapfiles %s", s) ; break
			} else if len(m) != 1 {
				ctx.err("unmapfiles %s : %v", s, m)
			} else if m[0].name != s {
				ctx.err("unmapfiles %s : %v", s, m[0])
			} else if m[0].pattern.String() != "foo/**z" {
				ctx.err("unmapfiles %s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0].pattern.string(ctx) != "foo/**z" {
				ctx.err("unmapfiles %s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("unmapfiles %s : %v", s, m[0])
			}
		}
	}
	if s := "foo/xx/yy/zzzz"; true {
		for i := 0 ; i < 50 ; i += 1 {
			if m := unmapfiles(ctx, s); m == nil {
				ctx.err("unmapfiles %s", s) ; break
			} else if len(m) != 1 {
				ctx.err("unmapfiles %s : %v", s, m)
			} else if m[0].name != s {
				ctx.err("unmapfiles %s : %v", s, m[0])
			} else if m[0].pattern.String() != "foo/**z" {
				ctx.err("unmapfiles %s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0].pattern.string(ctx) != "foo/**z" {
				ctx.err("unmapfiles %s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("unmapfiles %s : %v", s, m[0])
			}
		}
	}
}

func testValueCache3(ctx *testcase) {
	var u = _universe(ctx)
	if u == nil {
		ctx.err("nil universe")
	} else if c := &u.filemaps; c.a != nil {
		ctx.err("universe filecache : %v", c)
	} else if len(c.m) != 2 {
		ctx.err("universe filecache : %v", c)
	}
}

func testValueCache(ctx *testcase) {
	var p = ctx.project()
	var u = _universe(ctx)

	if u == nil {
		ctx.err("nil universe")
	}

	if p == nil {
		ctx.err("nil project")
	} else if p.filemap._fix == nil {
		ctx.err("wrong filemap._fix")
	} else if c1, y := p.filemap._fix["*"]; !y {
		ctx.err("wrong filemap._fix: %v", p.filemap._fix)
	} else if c2, y := c1._fix[".log"]; !y {
		ctx.err("wrong filemap._fix: %v %v", c1._fix, c2)
	}

	if p.filemap.fast == nil {
		ctx.err("wrong filemap.fast: %v", p.filemap)
	} else if c1, y := p.filemap.fast[".deps"]; !y {
		ctx.err("wrong filemap.fast: %v, %v", p.filemap.fast, c1)
	}

	for i, s := range []string{
		".deps/xx/yy/zzzzzzzzz",
		"foo.log",
		"foo.o",
		"foo.c",
		"foo.c++",
	} {
		if !strings.Contains(s, pathSep) {
			if 2 < i { /* not matching patterns */ } else
			if c := p.filemap.matchPatts(ctx, s); c == nil {
				ctx.err("miss cache for %d. %s", i, s)
			}
			if c := p.filemap.str(ctx, "foo.log", cacheMatchPatts); c == nil {
				ctx.err("miss cache for %d. %s", i, s)
			}
		}
		if c := p.filemap.strx(ctx, s, cacheMatchPatts); c == nil {
			ctx.err("miss cache for %d. %s", i, s)
		}
	}

	if n := len(p.filemapx); n != 1 {
		ctx.err("wrong closure cache: %d", n)
	} else if v := p.filemapx[0]; v._key.String() != "&(gen)" {
		ctx.err("wrong closure cache: %v", us(v._key))
	} else if a, y := v._val.(filemap); !y {
		ctx.err("wrong closure cache: %v (%v)", us(v._val), a)
	}

	if v := ctx.val("val1"); v == nil {
		ctx.err("val1")
	} else if v.string(ctx) != "**.c++" {
		ctx.err("%v", v)
	} else if true {
		// skips, files() not working globs
	} else if t := files(ctx, v); len(t) != 2 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo/bar.c++" && t[0].name != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	} else if t[1].name != "foo/bar.c++" && t[1].name != "foo.c++" {
		ctx.err("%v %v", v, t[1])
	}

	if v := ctx.val("val2"); v == nil {
		ctx.err("val2")
	} else if v.string(ctx) != "foo.c++" {
		ctx.err("%v", v)
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%v %v", v, f)
	} else if f.ident(ctx) != "foo.c++" {
		ctx.err("%v %v", v, f)
	} else if t := files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	}

	if v := ctx.val("val3"); v == nil {
		ctx.err("val3")
	} else if v.string(ctx) != "foo.o" {
		ctx.err("%T %v", v, v)
	} else if t := unmapfiles(ctx, v); t == nil {
		ctx.err("%T %v", v, v)
	} else if len(t) != 1 {
		ctx.err("%T %v ; %v", v, v, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if t[0].pattern.String() != "**.o" {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if _, y := t[0].pattern.(*globpat); !y {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if t[0].project != p {
		ctx.err("%T %v ; %v %v %v", v, v, t[0], t[0].paths, t[0].project)
	} else if t[0].paths == nil {
		ctx.err("%T %v ; %v %v", v, v, t[0], t[0].paths)
	} else if t[0].paths[0].String() != "$//.tmp" {
		ctx.err("%T %v ; %v %v", v, v, t[0], t[0].paths)
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%T %v ; %v", v, v, t)
	} else if f.ident(ctx) != "foo.o" {
		ctx.err("%v %v", v, f)
	} else if t := files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%v %v", v, t[0])
	}

	if d := ctx.def("sources"); d == nil {
		ctx.err("sources is nil")
	} else if v := d.invoke(ctx, nil, nil); v == nil {
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

	if d := ctx.def("objects"); d == nil {
		ctx.err("objects is nil")
	} else if v := d.invoke(ctx, nil, nil); v == nil {
		ctx.err("objects is wrong: %v %v", d, v)
	}
}
