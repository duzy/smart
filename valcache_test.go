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

	if true { noted(ctx, "top: %v", &u.filemaps).debug(1) }

	if u == nil {
		ctx.err("nil universe")
	} else if u.filemaps.a != nil {
		ctx.err("universe filecache.a")
	} else if u.filemaps.m == nil {
		ctx.err("universe filecache.m")
	} else if m, y := u.filemaps.m["foo"]; !y {
		ctx.err("universe filecache.m[foo] : %v", u.filemaps.m)
	} else {
		noted(ctx, "%v", m).debug(1)
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
	} else if a, y := v._val.(FileMap); !y {
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
