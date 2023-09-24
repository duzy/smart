//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
	"testing"
)

func testValueCache(t *testing.T) {
	var ctx = load_testcase(t, "testdata/valcache", "valcache")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	var m = ctx.Project()

	if m.filemap._fix == nil {
		ctx.err("wrong filemap._fix")
	} else if c1, y := m.filemap._fix["*"]; !y {
		ctx.err("wrong filemap._fix: %v", m.filemap._fix)
	} else if c2, y := c1._fix[".log"]; !y {
		ctx.err("wrong filemap._fix: %v %v", c1._fix, c2)
	} else if false {
		info(ctx, "%v: %v", m, m.filemap)
		info(ctx, "%v: %v", m, c1)
		info(ctx, "%v: %v", m, c2)
	}

	if m.filemap.fast == nil {
		ctx.err("wrong filemap.fast: %v", m.filemap)
	} else if c1, y := m.filemap.fast[".deps"]; !y {
		ctx.err("wrong filemap.fast: %v", m.filemap.fast)
	} else if false {
		info(ctx, "%v: %v", m, m.filemap)
		info(ctx, "%v: %v", m, c1)
	}

	for i, s := range []string{
		".deps/xx/yy/zzzzzzzzz",
		"foo.log",
		"foo.o",
		"foo.c",
		"foo.c++",
	} {
		if !strings.Contains(s, PathSep) {
			if 2 < i { /* not matching patterns */ } else
			if c := m.filemap.matchPatts(ctx, s); c == nil {
				ctx.err("miss cache for %d. %s", i, s)
			}
			if c := m.filemap.str(ctx, "foo.log", cacheMatchPatts); c == nil {
				ctx.err("miss cache for %d. %s", i, s)
			}
		}
		if c := m.filemap.strx(ctx, s, cacheMatchPatts); c == nil {
			ctx.err("miss cache for %d. %s", i, s)
		}
	}

	if n := len(m.filemapx); n != 1 {
		ctx.err("wrong closure cache: %d", n)
	} else if v := m.filemapx[0]; v._key.String() != "&(gen)" {
		ctx.err("wrong closure cache: %T %v", v._key, v._key)
	} else if a, y := v._val.(FileMap); !y {
		ctx.err("wrong closure cache: %T %v", v._val, v._val)
	} else if false {
		info(ctx, "%T %v %v", v._key, v._key, a)
	}

	if v := ctx.get("val1"); v == nil {
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

	if v := ctx.get("val2"); v == nil {
		ctx.err("val2")
	} else if v.string(ctx) != "foo.c++" {
		ctx.err("%v", v)
	} else if f := m.file(ctx, v); f == nil {
		ctx.err("%v %v", v, f)
	} else if f.name(ctx) != "foo.c++" {
		ctx.err("%v %v", v, f)
	} else if t := files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	}

	if v := ctx.get("val3"); v == nil {
		ctx.err("val3")
	} else if v.string(ctx) != "foo.o" {
		ctx.err("%T %v", v, v)
	} else if t := unmap(ctx, v); t == nil {
		ctx.err("%T %v", v, v)
	} else if len(t) != 1 {
		ctx.err("%T %v ; %v", v, v, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if t[0].pattern.String() != "**.o" {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if _, y := t[0].pattern.(*GlobPattern); !y {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if t[0].project != m {
		ctx.err("%T %v ; %v %v %v", v, v, t[0], t[0].locs, t[0].project)
	} else if t[0].locs == nil {
		ctx.err("%T %v ; %v %v", v, v, t[0], t[0].locs)
	} else if t[0].locs[0].String() != "$//.tmp" {
		ctx.err("%T %v ; %v %v", v, v, t[0], t[0].locs)
	} else if f := m.file(ctx, v); f == nil {
		ctx.err("%T %v ; %v", v, v, t)
	} else if f.name(ctx) != "foo.o" {
		ctx.err("%v %v", v, f)
	} else if t := files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%v %v", v, t[0])
	}

	if d := ctx.def("sources"); d == nil {
		ctx.err("sources is nil")
	} else if v := d.invoke(ctx, plain, nil, nil); v == nil {
		ctx.err("sources is wrong: %v %v", d, v)
	} else if false {
		info(ctx, "%v", v).debug(1)
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
	} else if v := d.invoke(ctx, plain, nil, nil); v == nil {
		ctx.err("objects is wrong: %v %v", d, v)
	} else if false {
		info(ctx, "%v", v).debug(1)
	} else if s := v.string(ctx); strings.Count(s, "foo.o") != 2 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.o") != 2 {
		ctx.err("sources is wrong: %v", v)
	}

	ctx.flush()
}
