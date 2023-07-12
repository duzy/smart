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

func TestValueCache(t *testing.T) {
	var ctx = load_testcase(t, "testdata/valcache", "valcache")
	// var get = func(s string, ii ...interface{}) Value { return ctx.get(s, expandZero, ii...) }
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

	if d := ctx.def("sources"); d == nil {
		ctx.err("sources is nil")
	} else if v := d.invoke(ctx, plain, nil, nil); v == nil {
		ctx.err("sources is wrong: %v %v", d, v)
	} else if false {
		info(ctx, "%v", v).debug(1)
	} else if s := v.Strval(ctx); strings.Count(s, "foo.c") != 2 {
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
	} else if s := v.Strval(ctx); strings.Count(s, "foo.o") != 2 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.o") != 2 {
		ctx.err("sources is wrong: %v", v)
	}

	ctx.flush()
}
