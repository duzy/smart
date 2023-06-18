//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"sync"
	"strings"
	"testing"
)

func TestWildcard(t *testing.T) {
	var ctx = confine("testdata/wildcard")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testwildcard" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string) (res Value) {
			if o := m.resolveObject(ctx, name); o == nil {
				t.Errorf("%s is nil", name)
			} else if d, y := o.(*def); !y {
				t.Errorf("%s is not def: %T", name, o)
			} else if res = d.Call(ctx, nil); res == nil {
				t.Errorf("nil: %v", d)

				res = MakeNone(o.Position())
			}
			return
		}

		var (
			pat1 = get("pat1")
			pat2 = get("pat2")
			pat3 = get("pat3")
		)
		if s := pat1.Strval(ctx); s != "*.h" {
			t.Errorf("pat1 is wrong: %T %v %s", pat1, pat1, s)
		} else if cs := pat1.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
			t.Errorf("pat1 is wrong: %T %v: %v", pat1, pat1, cs)
		} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
			t.Errorf("pat1 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat1 is wrong: %T %v", cs[0]._val, cs[0]._val)
		}
		if s := pat2.Strval(ctx); s != "**.h" {
			t.Errorf("pat2 is wrong: %T %v %s", pat2, pat2, s)
		} else if cs := pat2.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
			t.Errorf("pat2 is wrong: %T %v: %v", pat2, pat2, cs)
		} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
			t.Errorf("pat2 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat2 is wrong: %T %v", cs[0]._val, cs[0]._val)
		}
		if s := pat3.Strval(ctx); s != "foobar/config/*.def.in" {
			t.Errorf("pat3 is wrong: %T %v %s", pat3, pat3, s)
		} else if cs := pat3.collect(ctx, &m.filemap, cacheMatchPatts); len(cs) != 1 {
			t.Errorf("pat3 is wrong: %T %v, %v", pat3, pat3, cs)
		} else if g, y := cs[0]._key.(*Path); !y || g == nil {
			t.Errorf("pat3 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat3 is wrong: %T %v", cs[0]._val, cs[0]._val)
		}

		{
			var N = 100
			var wg sync.WaitGroup
			wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
				var opts = wildcardOpts{ dir: ctx.WorkDir() }
				if a := _wildcard(ctx, &opts, pat3); a == nil {
					t.Errorf("pat3 _wildcard no files: %v (%d)", pat3, n)
				}
			} (i) }
			wg.Wait()
		}

		var (
			val1 = get("val1")
			val2 = get("val2")
			val3 = get("val3")
			val4 = get("val4")
			val5 = get("val5")
		)
		if s := val1.Strval(ctx); s == "" {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		} else if strings.Count(s, "inc/bar.h") != 1 {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		} else if strings.Count(s, "inc/foo.h") != 1 {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		} else if strings.Count(s, "inc/foo/v1.h") != 1 {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		} else if strings.Count(s, "inc/foo/v2.h") != 1 {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		} else if strings.Count(s, "inc/foo/bar/zz/x.h") > 1 {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		}

		if s := val2.Strval(ctx); s == "" {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		} else if strings.Count(s, "inc/bar.h") != 1 {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		} else if strings.Count(s, "inc/foo.h") != 1 {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		} else if strings.Count(s, "inc/foo/v1.h") != 1 {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		} else if strings.Count(s, "inc/foo/v2.h") != 1 {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		} else if strings.Count(s, "inc/foo/bar/zz/x.h") != 1 {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		}

		if s := val3.Strval(ctx); s == "" {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		} else if strings.Count(s, "bar.h") != 1 {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		} else if strings.Count(s, "foo.h") != 1 {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		} else if strings.Count(s, "foo/v1.h") != 1 {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		} else if strings.Count(s, "foo/v2.h") != 1 {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		} else if strings.Count(s, "foo/bar/v1.h") != 1 {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		} else if strings.Count(s, "foo/bar/v2.h") != 1 {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
			t.Errorf("val3 is wrong: %T %v", val3, val3)
		}

		if s := val4.Strval(ctx); s == "" {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		} else if strings.Count(s, "bar.h") != 1 {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		} else if strings.Count(s, "foo.h") != 1 {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		} else if strings.Count(s, "foo/v1.h") != 1 {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		} else if strings.Count(s, "foo/v2.h") != 1 {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		} else if strings.Count(s, "foo/bar/v1.h") != 1 {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		} else if strings.Count(s, "foo/bar/v2.h") != 1 {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
			t.Errorf("val4 is wrong: %T %v", val4, val4)
		}

		if s := val5.Strval(ctx); s == "" {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		} else if strings.Count(s, "bar.h") != 1 {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		} else if strings.Count(s, "foo.h") != 1 {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		} else if strings.Count(s, "foo/v1.h") != 0 {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		} else if strings.Count(s, "foo/v2.h") != 0 {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		} else if strings.Count(s, "foo/bar/v1.h") != 0 {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		} else if strings.Count(s, "foo/bar/v2.h") != 0 {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		} else if strings.Count(s, "foo/bar/zz/x.h") != 0 {
			t.Errorf("val5 is wrong: %T %v", val5, val5)
		}

		var (
			fix1 = get("fix1")
			fix2 = get("fix2")
		)
		if s := fix1.Strval(ctx); s == "" {
			t.Errorf("fix1 is wrong: %T %v", fix1, fix1)
		} else if strings.Count(s, "foobar/config/a.def.in") != 1 {
			t.Errorf("fix1 is wrong: %T %v", fix1, fix1)
		} else if strings.Count(s, "foobar/config/b.def.in") != 1 {
			t.Errorf("fix1 is wrong: %T %v", fix1, fix1)
		}
		if s := fix2.Strval(ctx); s == "" {
			t.Errorf("fix2 is wrong: %T %v", fix2, fix2)
		} else if strings.Count(s, "foobar/config/a.def.in") != 1 {
			t.Errorf("fix2 is wrong: %T %v", fix2, fix2)
		} else if strings.Count(s, "foobar/config/b.def.in") != 1 {
			t.Errorf("fix2 is wrong: %T %v", fix2, fix2)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}
