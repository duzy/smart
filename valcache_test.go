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
	var ctx = confine("testdata/valcache")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "valcache" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}

		if m.filemap._fix == nil {
			t.Errorf("wrong filemap._fix")
		} else if c1, y := m.filemap._fix["*"]; !y {
			t.Errorf("wrong filemap._fix: %v", m.filemap._fix)
		} else if c2, y := c1._fix[".log"]; !y {
			t.Errorf("wrong filemap._fix: %v %v", c1._fix, c2)
		} else if false {
			info(ctx, "%v: %v", m, m.filemap)
			info(ctx, "%v: %v", m, c1)
			info(ctx, "%v: %v", m, c2)
		}

		if m.filemap.fast == nil {
			t.Errorf("wrong filemap.fast: %v", m.filemap)
		} else if c1, y := m.filemap.fast[".deps"]; !y {
			t.Errorf("wrong filemap.fast: %v", m.filemap.fast)
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
					t.Errorf("miss cache for %d. %s", i, s)
				}
				if c := m.filemap.str(ctx, "foo.log", cacheMatchPatts); c == nil {
					t.Errorf("miss cache for %d. %s", i, s)
				}
			}
			if c := m.filemap.strx(ctx, s, cacheMatchPatts); c == nil {
				t.Errorf("miss cache for %d. %s", i, s)
			}
		}

		if n := len(m.filemapx); n != 1 {
			t.Errorf("wrong closure cache: %d", n)
		} else if v := m.filemapx[0]; v._key.String() != "&(gen)" {
			t.Errorf("wrong closure cache: %T %v", v._key, v._key)
		} else if a, y := v._val.(FileMap); !y {
			t.Errorf("wrong closure cache: %T %v", v._val, v._val)
		} else if false {
			info(ctx, "%T %v %v", v._key, v._key, a)
		}

		var d1 = m.resolveObject(ctx, "sources")
		if d1 == nil {
			t.Errorf("sources is nil")
		} else if d, y := d1.(*def); !y {
			t.Errorf("sources is not def: %T", d1)
		} else if v := d.Call(ctx, nil); v == nil {
			t.Errorf("sources is wrong: %v %v", d, v)
		} else if false {
			info(ctx, "%v", v).debug(1)
		} else if s := v.Strval(ctx); strings.Count(s, "foo.c") != 2 {
			t.Errorf("sources is wrong: %v", v) // NOTE: "foo.c" counts foo.c foo.c++
		} else if strings.Count(s, "foo.c++") != 1 {
			t.Errorf("sources is wrong: %v", v)
		} else if strings.Count(s, "foo/bar.c") != 2 {
			t.Errorf("sources is wrong: %v", v)
		} else if strings.Count(s, "foo/bar.c++") != 1 {
			t.Errorf("sources is wrong: %v", v)
		}

		var d2 = m.resolveObject(ctx, "objects")
		if d2 == nil {
			t.Errorf("objects is nil")
		} else if d, y := d2.(*def); !y {
			t.Errorf("objects is not def: %T", d2)
		} else if v := d.Call(ctx, nil); v == nil {
			t.Errorf("objects is wrong: %v %v", d, v)
		} else if false {
			info(ctx, "%v", v).debug(1)
		} else if s := v.Strval(ctx); strings.Count(s, "foo.o") != 2 {
			t.Errorf("sources is wrong: %v", v)
		} else if strings.Count(s, "foo/bar.o") != 2 {
			t.Errorf("sources is wrong: %v", v)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}
