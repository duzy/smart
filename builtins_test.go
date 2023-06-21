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
			var d *def
			if d, res = call(ctx, name); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		var (
			pat1 = get("pat1")
			pat2 = get("pat2")
			pat3 = get("pat3")
			pat4 = get("pat4")
			pat5 = get("pat5")
			pat6 = get("pat6")
		)
		if g, y := pat1.(*GlobPattern); !y || g == nil {
			t.Errorf("pat1 is wrong: %T %v", pat1, pat1)
		} else if s := pat1.Strval(ctx); s != "*.h" {
			t.Errorf("pat1 is wrong: %T %v %s", pat1, pat1, s)
		} else if cs := pat1.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
			t.Errorf("pat1 is wrong: %T %v: %v", pat1, pat1, cs)
		} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
			t.Errorf("pat1 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat1 is wrong: %T %v", cs[0]._val, cs[0]._val)
		}
		if g, y := pat2.(*GlobPattern); !y || g == nil {
			t.Errorf("pat2 is wrong: %T %v", pat2, pat2)
		} else if s := pat2.Strval(ctx); s != "**.h" {
			t.Errorf("pat2 is wrong: %T %v %s", pat2, pat2, s)
		} else if cs := pat2.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
			t.Errorf("pat2 is wrong: %T %v: %v", pat2, pat2, cs)
		} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
			t.Errorf("pat2 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat2 is wrong: %T %v", cs[0]._val, cs[0]._val)
		}
		if g, y := pat3.(*Path); !y || g == nil {
			t.Errorf("pat3 is wrong: %T %v", pat3, pat3)
		} else if s := pat3.Strval(ctx); s != "foobar/config/*.def.am" {
			t.Errorf("pat3 is wrong: %T %v %s", pat3, pat3, s)
		} else if cs0 := pat3.collect(ctx, &m.filemap, cacheZero); len(cs0) != 1 {
			t.Errorf("pat3 is wrong: %T %v, %v", pat3, pat3, cs0)
		} else if cs := pat3.collect(ctx, &m.filemap, cacheMatchPatts); len(cs) != 1 {
			t.Errorf("pat3 is wrong: %T %v, %v", pat3, pat3, cs)
		} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
			t.Errorf("pat3 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if g.String() != "**.def.am" {
			t.Errorf("pat3 is wrong: %v -> %v", pat3, g)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat3 is wrong: %T %v", cs[0]._val, cs[0]._val)
		}
		if g, y := pat4.(*Path); !y || g == nil {
			t.Errorf("pat4 is wrong: %T %v", pat4, pat4)
		} else if s := pat4.Strval(ctx); s != "foobar/config/*.def.in" {
			t.Errorf("pat4 is wrong: %T %v %s", pat4, pat4, s)
		} else if cs := pat4.collect(ctx, &m.filemap, cacheMatchPatts); len(cs) != 0 {
			// NOTE: because the files spec only defined "**.def.am", no "**.def.in"
			t.Errorf("pat4 is wrong: %T %v, %v", pat4, pat4, cs)
		}

		if g, y := pat5.(*GlobPattern); !y || g == nil {
			t.Errorf("pat5 is wrong: %T %v", pat5, pat5)
		} else if s := pat5.Strval(ctx); s != "*.def.am" {
			t.Errorf("pat5 is wrong: %T %v %s", pat5, pat5, s)
		} else if cs := pat5.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
			t.Errorf("pat5 is wrong: %T %v: %v", pat5, pat5, cs)
		} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
			t.Errorf("pat5 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat5 is wrong: %T %v", cs[0]._val, cs[0]._val)
		} else if y, r, s := pat5.match(ctx, pat3); y {
			t.Errorf("pat5 is wrong: %T %v, %v ; %v %v", pat5, pat5, pat3, r, s)
		} else if y, r, s := pat5.match(ctx, pat4); y {
			t.Errorf("pat5 is wrong: %T %v, %v ; %v %v", pat5, pat5, pat4, r, s)
		}
		if g, y := pat6.(*GlobPattern); !y || g == nil {
			t.Errorf("pat6 is wrong: %T %v", pat6, pat6)
		} else if s := pat6.Strval(ctx); s != "**.def.am" {
			t.Errorf("pat6 is wrong: %T %v %s", pat6, pat6, s)
		} else if cs := pat6.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
			t.Errorf("pat6 is wrong: %T %v: %v", pat6, pat6, cs)
		} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
			t.Errorf("pat6 is wrong: %T %v", cs[0]._key, cs[0]._key)
		} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
			t.Errorf("pat6 is wrong: %T %v", cs[0]._val, cs[0]._val)
		} else if y, r, s := pat6.match(ctx, pat3); !y {
			t.Errorf("pat6 is wrong: %T %v, %v ; %v %v", pat6, pat6, pat3, r, s)
		} else if r == nil {
			t.Errorf("pat6 is wrong: %T %v, %v ; %T %v", pat6, pat6, pat3, r, r)
		} else if a, y := r.(string); !y {
			t.Errorf("pat6 is wrong: %T %v, %v ; %T %v", pat6, pat6, pat3, r, r)
		} else if a != "foobar/config/*.def.am" {
			t.Errorf("pat6 is wrong: %T %v, %v ; %v", pat6, pat6, pat3, t)
		} else if s == nil || len(s) != 1 {
			t.Errorf("pat6 is wrong: %T %v, %v ; %v", pat6, pat6, pat3, s)
		} else if s[0] != "foobar/config/*" {
			t.Errorf("pat6 is wrong: %T %v, %v ; %v", pat6, pat6, pat3, s)
		} else if y, r, s := pat6.match(ctx, pat4); y {
			t.Errorf("pat6 is wrong: %T %v, %v ; %v %v", pat6, pat6, pat4, r, s)
		}

		const N = 10
		var wg sync.WaitGroup
		var optsNoDir   = wildcardOpts{ dir: "" }
		var optsWorkDir = wildcardOpts{ dir: ctx.WorkDir() + "/inc" }
		invalid := func(name string) bool { return name == "" ||
			name != "foobar/config/a.def.in" &&
			name != "foobar/config/b.def.in" ;
		}
		{
			wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
				if a := _wildcard(of(ctx,pat3), &optsWorkDir, pat3); len(a) != 1 {
					t.Errorf("_wildcard(%v) is wrong (%d): %v", pat3, n, a)
				} else if a[0].name != "foobar/config/a.def.am" {
					t.Errorf("_wildcard(%v) is wrong (%d): %v", pat3, n, a[0])
				}
			} (i) }
		}
		{
			wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
				if a := _wildcard(of(ctx,pat4), &optsWorkDir, pat4); len(a) != 2 {
					t.Errorf("_wildcard(%v) is wrong (%d): %v", pat4, n, a)
				} else if invalid(a[0].name) {
					t.Errorf("_wildcard(%v) is wrong (%d): %v", pat4, n, a[0])
				} else if invalid(a[1].name) {
					t.Errorf("_wildcard(%v) is wrong (%d): %v", pat4, n, a[1])
				}
			} (i) }
		}
		{
			wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
				if a := wildcard(of(ctx,pat3), &optsWorkDir, pat3); len(a) != 1 {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat3, n, a)
				} else if a[0].name != "foobar/config/a.def.am" {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat3, n, a[0])
				}
			} (i) }
		}
		{
			wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
				if a := wildcard(of(ctx,pat4), &optsWorkDir, pat4); len(a) != 2 {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat4, n, a)
				} else if invalid(a[0].name) {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat4, n, a[0])
				} else if invalid(a[1].name) {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat4, n, a[1])
				}
			} (i) }
		}
		{
			wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
				if a := wildcard(ctx, &optsNoDir, pat3); len(a) != 1 {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat3, n, a)
				} else if a[0].name != "foobar/config/a.def.am" {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat3, n, a[0])
				}
			} (i) }
		}
		{
			wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
				if a := wildcard(ctx, &optsNoDir, pat4); a != nil {
					t.Errorf("wildcard(%v) is wrong (%d): %v", pat4, n, a)
				}
			} (i) }
		}
		wg.Wait()

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
			fix3 = get("fix3")
			fix4 = get("fix4")
		)
		if s := fix1.Strval(ctx); s == "" {
			t.Errorf("fix1 is wrong: %T %v", fix1, fix1)
		} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
			t.Errorf("fix1 is wrong: %T %v", fix1, fix1)
		}
		if s := fix2.Strval(ctx); s != "" {
			// NOTE: because the files spec defines only "**.def.am", no "**.def.in"
			t.Errorf("fix2 is wrong: %T %v", fix2, fix2)
		}
		if s := fix3.Strval(ctx); s == "" {
			t.Errorf("fix3 is wrong: %T %v", fix3, fix3)
		} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
			t.Errorf("fix3 is wrong: %T %v", fix3, fix3)
		}
		if s := fix4.Strval(ctx); s == "" {
			t.Errorf("fix4 is wrong: %T %v", fix4, fix4)
		} else if strings.Count(s, "foobar/config/a.def.in") != 1 {
			t.Errorf("fix4 is wrong: %T %v", fix4, fix4)
		} else if strings.Count(s, "foobar/config/b.def.in") != 1 {
			t.Errorf("fix4 is wrong: %T %v", fix4, fix4)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}

func TestForeach(t *testing.T) {
	var ctx = confine("testdata/foreach")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testforeach" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name, ii...); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		if v := get(".test.1"); v == nil {
			t.Errorf(".test.1")
		} else if s := v.String(); s != "x $(foreach $1,&(.test.foo)$_)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.2"); v == nil {
			t.Errorf(".test.2")
		} else if s := v.String(); s != "x $(foreach q p $(foreach $1,-$_),x$_)" {
		// } else if s := v.String(); s != "x $(foreach q p $(foreach $1,&(.test.foo)$_),x$_)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x xq xp" {
			t.Errorf("%T %v -> %s", v, v, s)
		}
		if v := get(".test.2", []string{"a", "b", "c"}); v == nil {
			t.Errorf(".test.2")
		} else if s := v.String(); s != "x $(foreach q p $(foreach a b c,-$_),x$_)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x xq xp x-a x-b x-c" {
			t.Errorf("%T %v -> %s", v, v, s)
		}
		if v := get(".test.2", []string{"x", "y", "z"}); v == nil {
			t.Errorf(".test.2")
		} else if s := v.String(); s != "x $(foreach q p $(foreach x y z,-$_),x$_)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x xq xp x-x x-y x-z" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.3"); v == nil {
			t.Errorf(".test.3")
		} else if s := v.String(); s != "x $(foreach $1,&(.test.$_)$1) $(foreach $1,$(value(-c) .test.$_)$1)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.4"); v == nil {
			t.Errorf(".test.4")
		} else if s := v.String(); s != "$(foreach $1 $2,&(.test.$_.$(or $4,$3)))" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "" {
			t.Errorf("%T %v -> %s", v, v, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}

func TestForeach1(t *testing.T) {
	var ctx = confine("testdata/foreach/1")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testforeach" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		if v := get(".test.4"); v == nil {
			t.Errorf(".test.4")
		} else if s := v.String(); s != "$(.test.x $1,B b)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foo test-foo test-B test-foo-B test-b test-foo-b" {
			t.Errorf("%T %v -> %s", v, v, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}

func TestAddPrefix(t *testing.T) {
	var ctx = confine("testdata/addprefix")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testaddprefix" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name, ii...); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		if v := get("val1"); v == nil {
			t.Errorf("val1")
		} else if s := v.String(); s != /* "$(addprefix -std=,foo)" */"-std=foo" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "-std=foo" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if p, y := v.(*Pair); !y {
			t.Errorf("%T %v", v, v)
		} else if s := p.Key.String(); s != "-std" {
			t.Errorf("%T %v ; %T %v", v, v, p.Key, p.Key)
		} else if s := p.Value.String(); s != "foo" {
			t.Errorf("%T %v ; %T %v", v, v, p.Value, p.Value)
		}

		if v := get("val2"); v == nil {
			t.Errorf("val2")
		} else if s := v.String(); s != /* "$(addprefix -std=,foo bar foobar)" */"-std=foo -std=bar -std=foobar" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "-std=foo -std=bar -std=foobar" {
			t.Errorf("%T %v -> %s", v, v, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}
