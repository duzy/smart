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
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}}

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
			"foo.log",
			".deps/xx/yy/zzzzzzzzz",
		} {
			if !strings.Contains(s, PathSep) {
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
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}
