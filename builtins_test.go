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
	"path/filepath"
)

const TER = false

func testAssert(t *testing.T) {
	var boos []bool
	var vals []Value
	runcase(t, "testdata/assert", "testassert", func (ctx *testcase) {
		if foo := ctx.get("foo"); foo == nil {
			ctx.err("foo")
		} else if foo.string(ctx) != "foo" {
			ctx.err("%T %v", foo, foo)
		}

		if len(boos) != 11 {
			ctx.err("%v, %v, %v %v", vals, boos, len(vals), len(boos))
		} else if len(vals) != len(boos) {
			ctx.err("%v %v", vals, boos)
		} else if i := 0; vals[i].String() != "true{}" || !boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 1; vals[i].String() != "false{}" || boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 2; vals[i].String() != "yes{}" || !boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 3; vals[i].String() != "no{}" || boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 4; vals[i].String() != "" || boos[i] { // none{}
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 5; vals[i].String() != "undef{}" || boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 6; vals[i].String() != "" || boos[i] { // null{}
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 7; vals[i].String() != "foobar{}" || !boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 8; vals[i].String() != "1" || !boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 9; vals[i].String() != "0" || boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		} else if i = 10; vals[i].String() != "$(equal $(foo),foo)" || !boos[i] {
			ctx.err("%v %v", vals[i], boos[i])
		}
	}, hooks{
		assert: func(ctx Context, v Value, b bool) (res bool) {
			vals, boos = append(vals, v), append(boos, b)
			return true
		},
	})
}

func testWildcard(ctx *testcase) {
	var (
		pat1 = ctx.get("pat1")
		pat2 = ctx.get("pat2")
		pat3 = ctx.get("pat3")
		pat4 = ctx.get("pat4")
		pat5 = ctx.get("pat5")
		pat6 = ctx.get("pat6")
		m = ctx.Project()
	)

	if true {
		var f = func(a []Value) { a[0], a[1], a[4], a[5] = pat1, pat2, pat5, pat6 }
		var a = []Value{ nil, nil, nil, nil, nil, nil } ; f(a)
		if a[0] != pat1 { ctx.Errorf("%v", a) }
		if a[1] != pat2 { ctx.Errorf("%v", a) }
		if a[2] != nil  { ctx.Errorf("%v", a) }
		if a[3] != nil  { ctx.Errorf("%v", a) }
		if a[4] != pat5 { ctx.Errorf("%v", a) }
		if a[5] != pat6 { ctx.Errorf("%v", a) }
	}

	if g, y := pat1.(*GlobPattern); !y || g == nil {
		ctx.err("pat1: %T %v", pat1, pat1)
	} else if s := pat1.string(ctx); s != "*.h" {
		ctx.err("pat1: %T %v %s", pat1, pat1, s)
	} else if cs := pat1.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
		ctx.err("pat1: %T %v: %v", pat1, pat1, cs)
	} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
		ctx.err("pat1: %T %v", cs[0]._key, cs[0]._key)
	} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
		ctx.err("pat1: %T %v", cs[0]._val, cs[0]._val)
	} else if m.pattern.string(ctx) != "**.h" {
		ctx.err("pat1: %T %v → %T %v", cs[0]._val, cs[0]._val, m.pattern, m.pattern)
	}
	if g, y := pat2.(*GlobPattern); !y || g == nil {
		ctx.err("pat2: %T %v", pat2, pat2)
	} else if s := pat2.string(ctx); s != "**.h" {
		ctx.err("pat2: %T %v %s", pat2, pat2, s)
	} else if cs := pat2.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
		ctx.err("pat2: %T %v: %v", pat2, pat2, cs)
	} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
		ctx.err("pat2: %T %v", cs[0]._key, cs[0]._key)
	} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
		ctx.err("pat2: %T %v", cs[0]._val, cs[0]._val)
	} else if m.pattern.string(ctx) != "**.h" {
		ctx.err("pat1: %T %v → %T %v", cs[0]._val, cs[0]._val, m.pattern, m.pattern)
	}
	if g, y := pat3.(*Path); !y || g == nil {
		ctx.err("pat3: %T %v", pat3, pat3)
	} else if s := pat3.string(ctx); s != "foobar/config/*.def.am" {
		ctx.err("pat3: %T %v %s", pat3, pat3, s)
	} else if cs0 := pat3.collect(ctx, &m.filemap, cacheZero); len(cs0) != 1 {
		ctx.err("pat3: %T %v, %v", pat3, pat3, cs0)
	} else if cs := pat3.collect(ctx, &m.filemap, cacheMatchPatts); len(cs) != 1 {
		ctx.err("pat3: %T %v, %v", pat3, pat3, cs)
	} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
		ctx.err("pat3: %T %v", cs[0]._key, cs[0]._key)
	} else if g.String() != "**.def.am" {
		ctx.err("pat3: %v → %v", pat3, g)
	} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
		ctx.err("pat3: %T %v", cs[0]._val, cs[0]._val)
	} else if m.pattern.string(ctx) != "**.def.am" {
		ctx.err("pat1: %T %v → %T %v", cs[0]._val, cs[0]._val, m.pattern, m.pattern)
	}
	if g, y := pat4.(*Path); !y || g == nil {
		ctx.err("pat4: %T %v", pat4, pat4)
	} else if s := pat4.string(ctx); s != "foobar/config/*.def.in" {
		ctx.err("pat4: %T %v %s", pat4, pat4, s)
	} else if cs := pat4.collect(ctx, &m.filemap, cacheMatchPatts); len(cs) != 0 {
		// NOTE: because the files spec only defined "**.def.am", no "**.def.in"
		ctx.err("pat4: %T %v, %v", pat4, pat4, cs)
	}

	if g, y := pat5.(*GlobPattern); !y || g == nil {
		ctx.err("pat5: %T %v", pat5, pat5)
	} else if s := pat5.string(ctx); s != "*.def.am" {
		ctx.err("pat5: %T %v %s", pat5, pat5, s)
	} else if cs := pat5.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
		ctx.err("pat5: %T %v: %v", pat5, pat5, cs)
	} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
		ctx.err("pat5: %T %v", cs[0]._key, cs[0]._key)
	} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
		ctx.err("pat5: %T %v", cs[0]._val, cs[0]._val)
	} else if y, r, s := pat5.match(ctx, pat3); y {
		ctx.err("pat5: %T %v, %v ; %v %v", pat5, pat5, pat3, r, s)
	} else if y, r, s := pat5.match(ctx, pat4); y {
		ctx.err("pat5: %T %v, %v ; %v %v", pat5, pat5, pat4, r, s)
	} else if m.pattern.string(ctx) != "**.def.am" {
		ctx.err("pat1: %T %v → %T %v", cs[0]._val, cs[0]._val, m.pattern, m.pattern)
	}
	if g, y := pat6.(*GlobPattern); !y || g == nil {
		ctx.err("pat6: %T %v", pat6, pat6)
	} else if s := pat6.string(ctx); s != "**.def.am" {
		ctx.err("pat6: %T %v %s", pat6, pat6, s)
	} else if cs := pat6.collect(ctx, &m.filemap, cacheZero); len(cs) != 1 {
		ctx.err("pat6: %T %v: %v", pat6, pat6, cs)
	} else if g, y := cs[0]._key.(*GlobPattern); !y || g == nil {
		ctx.err("pat6: %T %v", cs[0]._key, cs[0]._key)
	} else if m, y := cs[0]._val.(FileMap); !y || m.pattern == nil {
		ctx.err("pat6: %T %v", cs[0]._val, cs[0]._val)
	} else if m.pattern.string(ctx) != "**.def.am" {
		ctx.err("pat1: %T %v → %T %v", cs[0]._val, cs[0]._val, m.pattern, m.pattern)
	} else if y, r, s := pat6.match(ctx, pat3); !y {
		ctx.err("pat6: %T %v, %v ; %v %v", pat6, pat6, pat3, r, s)
	} else if r == nil {
		ctx.err("pat6: %T %v, %v ; %T %v", pat6, pat6, pat3, r, r)
	} else if a, y := r.(string); !y {
		ctx.err("pat6: %T %v, %v ; %T %v", pat6, pat6, pat3, r, r)
	} else if a != "foobar/config/*.def.am" {
		ctx.err("pat6: %T %v, %v", pat6, pat6, pat3)
	} else if s == nil || len(s) != 1 {
		ctx.err("pat6: %T %v, %v ; %v", pat6, pat6, pat3, s)
	} else if s[0] != "foobar/config/*" {
		ctx.err("pat6: %T %v, %v ; %v", pat6, pat6, pat3, s)
	} else if y, r, s := pat6.match(ctx, pat4); y {
		ctx.err("pat6: %T %v, %v ; %v %v", pat6, pat6, pat4, r, s)
	}

	const N = 1
	var wg sync.WaitGroup
	var workDirInc = ctx.WorkDir() + "/inc"
	invalid := func(name string) bool { return name == "" ||
		name != "foobar/config/a.def.in" &&
			name != "foobar/config/b.def.in" ;}
	{
		wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
			if a := (&builtin_wildcard{builtin_:builtin_{Context:of(ctx,pat3)}, dir:workDirInc})._do(pat3); len(a) != 1 {
				ctx.err("_wildcard(%v) (%d): %v", pat3, n, a)
			} else if a[0].name(ctx) != "foobar/config/a.def.am" {
				ctx.err("_wildcard(%v) (%d): %v", pat3, n, a[0])
			}
		} (i) }
	}
	{
		wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
			if a := (&builtin_wildcard{builtin_:builtin_{Context:of(ctx,pat4)}, dir:workDirInc})._do(pat4); len(a) != 2 {
				ctx.err("_wildcard(%v) (%d): %v", pat4, n, a)
			} else if invalid(a[0].name(ctx)) {
				ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[0])
			} else if invalid(a[1].name(ctx)) {
				ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[1])
			}
		} (i) }
	}
	{
		wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
			if a := (&builtin_wildcard{builtin_:builtin_{Context:of(ctx,pat3)}, dir:workDirInc}).do(pat3); len(a) != 1 {
				ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
			} else if a[0].name(ctx) != "foobar/config/a.def.am" {
				ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
			}
		} (i) }
	}
	{
		wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
			if a := (&builtin_wildcard{builtin_:builtin_{Context:of(ctx,pat4)}, dir:workDirInc}).do(pat4); len(a) != 2 {
				ctx.err("wildcard(%v) (%d): %v", pat4, n, a)
			} else if invalid(a[0].name(ctx)) {
				ctx.err("wildcard(%v) (%d): %v", pat4, n, a[0])
			} else if invalid(a[1].name(ctx)) {
				ctx.err("wildcard(%v) (%d): %v", pat4, n, a[1])
			}
		} (i) }
	}
	{
		wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
			if a := (&builtin_wildcard{builtin_:builtin_{Context:of(ctx,pat3)}}).do(pat3); len(a) != 1 {
				ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
			} else if a[0].name(ctx) != "foobar/config/a.def.am" {
				ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
			}
		} (i) }
	}
	{
		wg.Add(N) ; for i := 0; i < N; i += 1 { go func(n int) { defer wg.Done()
			if a := (&builtin_wildcard{builtin_:builtin_{Context:of(ctx,pat4)}}).do(pat4); a != nil {
				ctx.err("wildcard(%v) (%d): %v", pat4, n, a)
			}
		} (i) }
	}
	wg.Wait()

	var (
		val1 = ctx.get("val1")
		val2 = ctx.get("val2")
		val3 = ctx.get("val3")
		val4 = ctx.get("val4")
		val5 = ctx.get("val5")
	)
	if s := val1.string(ctx); s == "" {
		ctx.err("val1: %T %v", val1, val1)
	} else if strings.Count(s, "inc/bar.h") != 1 {
		ctx.err("val1: %T %v", val1, val1)
	} else if strings.Count(s, "inc/foo.h") != 1 {
		ctx.err("val1: %T %v", val1, val1)
	} else if strings.Count(s, "inc/foo/v1.h") != 1 {
		ctx.err("val1: %T %v", val1, val1)
	} else if strings.Count(s, "inc/foo/v2.h") != 1 {
		ctx.err("val1: %T %v", val1, val1)
	} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
		ctx.err("val1: %T %v", val1, val1)
	} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
		ctx.err("val1: %T %v", val1, val1)
	} else if strings.Count(s, "inc/foo/bar/zz/x.h") > 1 {
		ctx.err("val1: %T %v", val1, val1)
	}

	if s := val2.string(ctx); s == "" {
		ctx.err("val2: %T %v", val2, val2)
	} else if strings.Count(s, "inc/bar.h") != 1 {
		ctx.err("val2: %T %v", val2, val2)
	} else if strings.Count(s, "inc/foo.h") != 1 {
		ctx.err("val2: %T %v", val2, val2)
	} else if strings.Count(s, "inc/foo/v1.h") != 1 {
		ctx.err("val2: %T %v", val2, val2)
	} else if strings.Count(s, "inc/foo/v2.h") != 1 {
		ctx.err("val2: %T %v", val2, val2)
	} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
		ctx.err("val2: %T %v", val2, val2)
	} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
		ctx.err("val2: %T %v", val2, val2)
	} else if strings.Count(s, "inc/foo/bar/zz/x.h") != 1 {
		ctx.err("val2: %T %v", val2, val2)
	}

	if s := val3.string(ctx); s == "" {
		ctx.err("val3: %T %v", val3, val3)
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("val3: %T %v", val3, val3)
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("val3: %T %v", val3, val3)
	} else if strings.Count(s, "foo/v1.h") != 1 {
		ctx.err("val3: %T %v", val3, val3)
	} else if strings.Count(s, "foo/v2.h") != 1 {
		ctx.err("val3: %T %v", val3, val3)
	} else if strings.Count(s, "foo/bar/v1.h") != 1 {
		ctx.err("val3: %T %v", val3, val3)
	} else if strings.Count(s, "foo/bar/v2.h") != 1 {
		ctx.err("val3: %T %v", val3, val3)
	} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
		ctx.err("val3: %T %v", val3, val3)
	}

	if s := val4.string(ctx); s == "" {
		ctx.err("val4: %T %v", val4, val4)
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("val4: %T %v", val4, val4)
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("val4: %T %v", val4, val4)
	} else if strings.Count(s, "foo/v1.h") != 1 {
		ctx.err("val4: %T %v", val4, val4)
	} else if strings.Count(s, "foo/v2.h") != 1 {
		ctx.err("val4: %T %v", val4, val4)
	} else if strings.Count(s, "foo/bar/v1.h") != 1 {
		ctx.err("val4: %T %v", val4, val4)
	} else if strings.Count(s, "foo/bar/v2.h") != 1 {
		ctx.err("val4: %T %v", val4, val4)
	} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
		ctx.err("val4: %T %v", val4, val4)
	}

	if s := val5.string(ctx); s == "" {
		ctx.err("val5: %T %v", val5, val5)
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("val5: %T %v", val5, val5)
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("val5: %T %v", val5, val5)
	} else if strings.Count(s, "foo/v1.h") != 0 {
		ctx.err("val5: %T %v", val5, val5)
	} else if strings.Count(s, "foo/v2.h") != 0 {
		ctx.err("val5: %T %v", val5, val5)
	} else if strings.Count(s, "foo/bar/v1.h") != 0 {
		ctx.err("val5: %T %v", val5, val5)
	} else if strings.Count(s, "foo/bar/v2.h") != 0 {
		ctx.err("val5: %T %v", val5, val5)
	} else if strings.Count(s, "foo/bar/zz/x.h") != 0 {
		ctx.err("val5: %T %v", val5, val5)
	}

	var (
		fix1 = ctx.get("fix1")
		fix2 = ctx.get("fix2")
		fix3 = ctx.get("fix3")
		fix4 = ctx.get("fix4")
	)
	if s := fix1.string(ctx); s == "" {
		ctx.err("fix1: %T %v", fix1, fix1)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix1: %T %v", fix1, fix1)
	}
	if s := fix2.string(ctx); s != "" {
		// NOTE: because the files spec defines only "**.def.am", no "**.def.in"
		ctx.err("fix2: %T %v", fix2, fix2)
	}
	if s := fix3.string(ctx); s == "" {
		ctx.err("fix3: %T %v", fix3, fix3)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix3: %T %v", fix3, fix3)
	}
	if s := fix4.string(ctx); s == "" {
		ctx.err("fix4: %T %v", fix4, fix4)
	} else if strings.Count(s, "foobar/config/a.def.in") != 1 {
		ctx.err("fix4: %T %v", fix4, fix4)
	} else if strings.Count(s, "foobar/config/b.def.in") != 1 {
		ctx.err("fix4: %T %v", fix4, fix4)
	}
}

func testFiles(ctx *testcase) {
	if t := unmap(ctx, ".test/a/b/c/foo.c"); t == nil {
		ctx.err(".test/a/b/c/foo.c")
	} else if len(t) != 1 {
		ctx.err(".test/a/b/c/foo.c: %v", t)
	} else if m := t[0]; m.name != ".test/a/b/c/foo.c" {
		ctx.err(".test/a/b/c/foo.c: %v", m.name)
	} else if m.pattern.string(ctx) != ".test/a/**.c" {
		ctx.err(".test/a/b/c/foo.c: %v", m.pattern)
	} else if false {
		noted(ctx, ".test/a/b/c/foo.c: %v %v %v", m.pattern, m.name, m.FileMap).debug(1)
	}

	if t := files(ctx, ".test/a/b/c/foo.c"); t == nil {
		ctx.err(".test/a/b/c/foo.c")
	} else if len(t) != 1 {
		ctx.err(".test/a/b/c/foo.c: %v", t)
	} else if m := t[0]; m.name != ".test/a/b/c/foo.c" {
		ctx.err(".test/a/b/c/foo.c: %v", m.name)
	} else if m.pattern.string(ctx) != ".test/a/**.c" {
		ctx.err(".test/a/b/c/foo.c: %v", m.pattern)
	} else if false {
		noted(ctx, ".test/a/b/c/foo.c: %v %v %v", m.pattern, m.name, m.FileMap).debug(1)
	}

	if val := ctx.get("val1.1"); val == nil {
		ctx.err("val1.1: %v", ctx.Project())
	} else if val.String() != ".test/a/b/c/foo.c" {
		ctx.err("%T %v", val, val)
	} else if f, y := val.(*File); !y {
		ctx.err("%T %v", val, val)
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != ".test/a/**.c" {
		ctx.err("%v", f)
	}

	if val := ctx.get("val1.2"); val == nil {
		ctx.err("val1.2: %v", ctx.Project())
	} else if val.String() != ".test/a/b/c/foo.c" {
		ctx.err("%T %v", val, val)
	} else if f, y := val.(*File); !y {
		ctx.err("%T %v", val, val)
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != ".test/a/**.c" {
		ctx.err("%v", f)
	}

	if val := ctx.get("val2.1"); val == nil {
		ctx.err("val2.1: %v", ctx.Project())
	} else if val.String() != ".test/xx/foo.c" {
		ctx.err("%T %v", val, val)
	} else if f, y := val.(*File); !y {
		ctx.err("%T %v", val, val)
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != ".test/xx/*.c" {
		ctx.err("%v", f)
	}

	if val := ctx.get("val2.2"); val == nil {
		ctx.err("val2.2: %v", ctx.Project())
	} else if val.String() != ".test/xx/foo.c" {
		ctx.err("%T %v", val, val)
	} else if f, y := val.(*File); !y {
		ctx.err("%T %v", val, val)
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != ".test/xx/*.c" {
		ctx.err("%v", f)
	}

	if val := ctx.get("val3"); val == nil {
		ctx.err("val3: %v", ctx.Project())
	} else if val.String() != ".test/xx/yy/foo.c" {
		ctx.err("%T %v", val, val)
	} else if f, y := val.(*File); !y {
		ctx.err("%T %v", val, val)
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != ".test/xx/yy/*.c" {
		ctx.err("%v", f)
	}

	if val := ctx.get("val4"); val == nil {
		ctx.err("val4: %v", ctx.Project())
	} else if val.String() != ".test/xx/yy/zz/foo.c" {
		ctx.err("%T %v", val, val)
	} else if f, y := val.(*File); !y {
		ctx.err("%T %v", val, val)
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != ".test/xx/yy/zz/*.c" {
		ctx.err("%v", f)
	}

	if val := ctx.get("val5"); val == nil {
		ctx.err("val5: %v", ctx.Project())
	} else if val.String() != "" {
		ctx.err("%T %v", val, val)
	} else if _, y := val.(*none); !y {
		ctx.err("%T %v", val, val)
	}

	if val := ctx.get("pat1.0"); val == nil {
		ctx.err("pat1.0: %v", ctx.Project())
	} else if val.String() != ".test/x**y" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat1.1"); v == nil {
		ctx.err("pat1.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xxx-yyy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "xx-yy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat1.2"); v == nil {
		ctx.err("pat1.2: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xxx-yyx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "xx-yyx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat1.3"); v == nil {
		ctx.err("pat1.3: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xxx-yyx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "y" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "xx-yyx/" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat1.4"); v == nil {
		ctx.err("pat1.3: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xxx-yyx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "z" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "xx-yyx/z" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat1.5"); v == nil {
		ctx.err("pat1.5: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 6 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xxx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[5] != "yyy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "xx/a/b/c/yy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat1.6"); v == nil {
		ctx.err("pat1.6: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "x" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "xx-yy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "/xx-y" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat2.0"); val == nil {
		ctx.err("pat2.0: %v", ctx.Project())
	} else if val.String() != ".test/x**y/z" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat2.1"); v == nil {
		ctx.err("pat2.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xxx-yyy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "z" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "xx-yy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat2.2"); v == nil {
		ctx.err("pat2.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 7 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xxx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[5] != "yyy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[6] != "z" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "xx/a/b/c/yy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat3.0"); val == nil {
		ctx.err("pat3.0: %v", ctx.Project())
	} else if val.string(ctx) != ".test/x**y/x**" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat3.1"); v == nil {
		ctx.err("pat3.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 7 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xaaa" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "bbb" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "ccc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "y" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[5] != "xxx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[6] != "xx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "aaa/bbb/ccc/" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "xx/xx" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat3.2"); v == nil {
		ctx.err("pat3.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xaabbccy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "xabc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "aabbcc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "abc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat4.0"); val == nil {
		ctx.err("pat4.0: %v", ctx.Project())
	} else if val.String() != ".test/x**y/x**y" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat4.1"); v == nil {
		ctx.err("pat4.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 7 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xaa" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "bb" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "ccy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "xaa" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[5] != "bb" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[6] != "ccy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "aa/bb/cc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "aa/bb/cc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat4.2"); v == nil {
		ctx.err("pat4.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 5 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xaaay" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "x" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "aaa" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "y" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "aaa" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "/aaa/" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat5.0"); val == nil {
		ctx.err("pat5.0: %v", ctx.Project())
	} else if val.String() != ".test/x**/**y" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat5.1"); v == nil {
		ctx.err("pat5.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xabc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "abcy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "abc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "abc" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat5.2"); v == nil {
		ctx.err("pat5.2: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 5 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "xa" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "dy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "d" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat6.0"); val == nil {
		ctx.err("pat6.0: %v", ctx.Project())
	} else if val.String() != ".test/**y/**y" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat6.1"); v == nil {
		ctx.err("pat6.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 8 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "cy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[5] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[6] != "c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[7] != "y" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "a/b/c/" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat7.0"); val == nil {
		ctx.err("pat7.0: %v", ctx.Project())
	} else if val.String() != ".test/**y/**y/z" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat7.1"); v == nil {
		ctx.err("pat7.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 9 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "cy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[5] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[6] != "c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[7] != "y" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[8] != "z" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "a/b/c/" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat8.0"); val == nil {
		ctx.err("pat8.0: %v", ctx.Project())
	} else if val.String() != ".test/**/**z" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat8.1"); v == nil {
		ctx.err("pat8.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 5 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[4] != "xyz" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "xy" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat10.0"); val == nil {
		ctx.err("pat10.0: %v", ctx.Project())
	} else if val.String() != ".test/*.h" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat10.1"); v == nil {
		ctx.err("pat10.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat10.2"); v == nil {
		ctx.err("pat10.2: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.(string); !y {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 0 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat10.3"); v == nil {
		ctx.err("pat10.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.(string); !y {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 0 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat11.0"); val == nil {
		ctx.err("pat11.0: %v", ctx.Project())
	} else if val.String() != ".test/*/*.h" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat11.1"); v == nil {
		ctx.err("pat11.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat11.2"); v == nil {
		ctx.err("pat11.2: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "b.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat11.3"); v == nil {
		ctx.err("pat11.3: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat12.0"); val == nil {
		ctx.err("pat12.0: %v", ctx.Project())
	} else if val.String() != ".test/*/*/*.h" {
		ctx.err("%T %v", val, val)
	} else if p, y := val.(*Path); !y {
		ctx.err("%v", p)
	} else if v := ctx.get("pat12.1"); v == nil {
		ctx.err("pat12.1: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat12.2"); v == nil {
		ctx.err("pat12.2: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "b.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "b.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat12.3"); v == nil {
		ctx.err("pat12.3: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 4 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if b[3] != "c.h" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[2] != "c" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat12.4"); v == nil {
		ctx.err("pat12.4: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if v := ctx.get("pat12.5"); v == nil {
		ctx.err("pat12.5: %v", ctx.Project())
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", v, p, a, b, c)
	}

	if val := ctx.get("pat13.0"); val == nil {
		ctx.err("pat13.0: %v", ctx.Project())
	} else if val.string(ctx) != "**.auto" {
		ctx.err("%T %v", val, val)
	} else if v := ctx.get("pat13.1"); v == nil {
		ctx.err("pat13.1: %v", ctx.Project())
	} else if a, b, c := val.match(ctx, v); !a {
		ctx.err("%T %v ; %v %v %v", val, val, a, b, c)
	} else if t := unmap(ctx, v); t == nil {
		ctx.err("%v", v)
	} else if len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if m := t[0]; m.name != ".test/a/b/c.auto" {
		ctx.err("%v: %v", v, m.name)
	} else if m.pattern.string(ctx) != "**.auto" {
		ctx.err("%v: %v", v, m.pattern)
	} else if v := ctx.get("pat13.2"); v == nil {
		ctx.err("pat13.2: %v", ctx.Project())
	} else if a, b, c := val.match(ctx, v); a {
		ctx.err("%T %v ; %v %v %v", val, val, a, b, c)
	} else if t := unmap(ctx, v); t != nil {
		ctx.err("%v", v)
	}
}

func testForeach(ctx *testcase) {
	var to_list func(v Value) (*list, bool)
	var to_list_direct = func(v Value) (l *list, y bool) { l, y = v.(*list) ; return }
	var to_list_unexpanded = func(v Value) (l *list, y bool) {
		if a, b := v.(unexpanded); !b { l, y = to_list_direct(a.Value) }
		return
	}
	var test_1_value Value
	var test_1_str string

	if d := ctx.def(".test.1"); d == nil {
		ctx.err(".test.1: %v", ctx.Project())
	} else if test_1_value = d.value; test_1_value == nil {
		ctx.err("%v", d)
	} else if test_1_str = d.value.String(); test_1_str != "x $(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v", d.value, d.value)
	} else if s := d.value.string(ctx); s != "x -" {
		ctx.err("%T %v → %s", d.value, d.value, s)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%T %v", d.value, d.value)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if _, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if s := l.Elems[1].String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v ; %v", l.Elems[1], l.Elems[1], d)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", expandZero); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x $(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -" {
		ctx.err("%T %v → %s", v, v, s)
		// } else if u, y := v.(unexpanded); !y {
		// 	ctx.err("%T %v", v, v)
		// } else if l, y := u.Value.(*list); !y {
		// 	ctx.err("%T %v", u.Value, u.Value)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if s := l.Elems[0].String(); s != "x" {
		ctx.err("%T %v → %s", l.Elems[0], l.Elems[0], s)
	} else if s := l.Elems[1].String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v → %s", l.Elems[1], l.Elems[1], s)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%T %v", w, w)
	} else if u, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := d.String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%v", d)
	} else if s := d.x.String(); s != "foreach" {
		ctx.err("%T %v", d.x, d.x)
	} else if len(d.a) != 2 {
		ctx.err("%v", d.a)
	} else if s := d.a[0].String(); s != "$1" {
		ctx.err("%v", d.a)
	} else if s := d.a[1].String(); s != "&(.test.h)$_" {
		ctx.err("%v", d.a)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x $(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if l, y := u.Value.(*list); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if s := l.Elems[0].String(); s != "x" {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if s := l.Elems[1].String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%T %v", w, w)
	} else if u, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := d.String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%v", d)
	} else if s := d.x.String(); s != "foreach" {
		ctx.err("%T %v", d.x, d.x)
	} else if len(d.a) != 2 {
		ctx.err("%v", d.a)
	} else if s := d.a[0].String(); s != "$1" {
		ctx.err("%v", d.a)
	} else if s := d.a[1].String(); s != "&(.test.h)$_" {
		ctx.err("%v", d.a)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x &(.test.h)a &(.test.h)b &(.test.h)c" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.1"))
	} else if s := v.string(ctx); s != "x -a -b -c" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.1"))
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.1"))
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%v", w.s)
	} else if l1, y := l.Elems[1].(*list); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if len(l1.Elems) != 3 {
		ctx.err("%v", l.Elems)
	} else if s := l1.Elems[0].String(); s != "&(.test.h)a" {
		ctx.err("%T %v → %s", l1.Elems[0], l1.Elems[0], s)
	} else if s := l1.Elems[1].String(); s != "&(.test.h)b" {
		ctx.err("%T %v → %s", l1.Elems[1], l1.Elems[1], s)
	} else if s := l1.Elems[2].String(); s != "&(.test.h)c" {
		ctx.err("%T %v → %s", l1.Elems[2], l1.Elems[2], s)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", expandDelegate|expandDigitsKept, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x $(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if l, y := u.Value.(*list); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%v", w.s)
	} else if u, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := d.x.String(); s != "foreach" {
		ctx.err("%T %v", d.x, d.x)
	} else if len(d.a) != 2 {
		ctx.err("%v", d.a)
	} else if s := d.a[0].String(); s != "$1" {
		ctx.err("%v", d.a)
	} else if s := d.a[1].String(); s != "&(.test.h)$_" {
		ctx.err("%v", d.a)
	} else if s := d.String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%v", d)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", expandDelegate, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x &(.test.h)x &(.test.h)y &(.test.h)z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.1"))
	} else if s := v.string(ctx); s != "x -x -y -z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.1"))
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%v ; %v", w.s, v)
	} else if l1, y := l.Elems[1].(*list); !y {
		ctx.err("%T %v ; %v", l.Elems[1], l.Elems[1], v)
	} else if len(l1.Elems) != 3 {
		ctx.err("%v ; %v", l1.Elems, v)
	} else if u, y := l1.Elems[0].(unexpanded); !y { // ---- 1
		ctx.err("%T %v", l1.Elems[0], l1.Elems[0])
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)x" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v %v", l2, l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if c.x.String() != ".test.h" {
		ctx.err("%T %v", c.x, c.x)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	} else if u, y := l1.Elems[1].(unexpanded); !y { // ---- 2
		ctx.err("%T %v", l1.Elems[1], l1.Elems[1])
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)y" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v %v", l2, l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if c.x.String() != ".test.h" {
		ctx.err("%T %v", c.x, c.x)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	} else if u, y := l1.Elems[2].(unexpanded); !y { // ---- 3
		ctx.err("%T %v", l1.Elems[2], l1.Elems[2])
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)z" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v %v", l2, l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if c.x.String() != ".test.h" {
		ctx.err("%T %v", c.x, c.x)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", expandDelegate|expandDigits, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x &(.test.h)x &(.test.h)y &(.test.h)z" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -x -y -z" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%v", w.s)
	} else if l1, y := l.Elems[1].(*list); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if len(l1.Elems) != 3 {
		ctx.err("%v", l1.Elems)
	} else if l1, y := l.Elems[1].(*list); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if len(l1.Elems) != 3 {
		ctx.err("%v", l1.Elems)
	} else if u, y := l1.Elems[0].(unexpanded); !y { // 1
		ctx.err("%T %v", l1.Elems[0], l1.Elems[0])
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)x" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v", l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	} else if u, y := l1.Elems[1].(unexpanded); !y { // 2
		ctx.err("%T %v", l1.Elems[1], l1.Elems[1])
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)y" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v", l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	} else if u, y := l1.Elems[2].(unexpanded); !y { // 3
		ctx.err("%T %v", l1.Elems[2], l1.Elems[2])
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)z" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v", l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", expandClosure|expandDelegate, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x -x -y -z" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -x -y -z" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%v", w.s)
	} else if d, y := l.Elems[1].(*list); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if s := d.String(); s != "-x -y -z" {
		ctx.err("%T %v → %s", d.Elems, d.Elems, s)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if true { to_list = to_list_direct } else { to_list = to_list_unexpanded }
	if v := ctx.get(".test.1", expandClosure, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x -x -y -z" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -x -y -z" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get(".test.1", expandClosure|expandDefOriginOff, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x $(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if l, y := u.Value.(*list); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%v", w.s)
	} else if s := l.Elems[1].String(); s != "$(foreach $1,&(.test.h)$_)" { // NOTE: &(.test.h) kept
		ctx.err("%T %v → %s", l.Elems[1], l.Elems[1], s)
	} else if u, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := d.String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%v", d)
	} else if b, y := d.x.(*builtin); !y {
		ctx.err("%T %v", d.x, d.x)
	} else if b.name_ != "foreach" {
		ctx.err("%v", b)
	} else if d.o != nil {
		ctx.err("%v", d)
	} else if len(d.a) != 2 {
		ctx.err("%v", d)
	} else if s := d.a[0].String(); s != "$1" {
		ctx.err("%T %v - %s", d.a[0], d.a[0], s)
	} else if s := d.a[1].String(); s != "&(.test.h)$_" {
		ctx.err("%T %v - %s", d.a[1], d.a[1], s)
	} else if l, y := to_list(d.a[0]); !y {
		ctx.err("%T %v", d.a[0], d.a[0])
	} else if len(l.Elems) != 1 {
		ctx.err("%v", l.Elems)
	} else if l, y := d.a[1].(*list); !y {
		ctx.err("%T %v", d.a[1], d.a[1])
	} else if len(l.Elems) != 1 {
		ctx.err("%v", l.Elems)
	} else if c, y := l.Elems[0].(*barecomp); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if len(c.Elems) != 2 {
		ctx.err("%v", c.Elems)
	} else if c.Elems[0].String() != "&(.test.h)" {
		ctx.err("%T %v", c.Elems[0], c.Elems[0])
	} else if c.Elems[1].String() != "$_" {
		ctx.err("%T %v", c.Elems[1], c.Elems[1])
	} else if _, y := c.Elems[0].(*closure); !y { // TODO: unexpanded ??
		ctx.err("%T %v", c.Elems[0], c.Elems[0])
	} else if h, y := c.Elems[1].(placeholder); !y {
		ctx.err("%T %v", c.Elems[1], c.Elems[1])
	} else if hd, y := h.Value.(*delegate); !y {
		ctx.err("%T %v", h.Value, h.Value)
	} else if s := hd.String(); s != "$_" {
		ctx.err("%v", hd)
	} else if x := d.expand(ctx, plain|expandDelegate|expandPlaceholder); v == nil {
		ctx.err("%v", d)
	} else if s := x.string(ctx); s != "-" {
		ctx.err("%T %v → %s", x, x, s)
	} else if s := d.string(ctx); s != "-" {
		ctx.err("%v → %s", d, s)
	}
	if test_1_value == nil {
		ctx.err("%v %p", ctx.def(".test.1"), ctx.ic())
	} else if test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v - %v %p", test_1_value, test_1_value, test_1_str, ctx.def(".test.1"), ctx.ic())
	}
	if v := ctx.get(".test.1", expandDelegate, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "x &(.test.h)x &(.test.h)y &(.test.h)z" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x -x -y -z" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if w, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if w.s != "x" {
		ctx.err("%v", w.s)
	} else if s := l.Elems[1].String(); s != "&(.test.h)x &(.test.h)y &(.test.h)z" {
		ctx.err("%T %v → %s", l.Elems[1], l.Elems[1], s)
	} else if l1, y := l.Elems[1].(*list); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if len(l1.Elems) != 3 {
		ctx.err("%v", l1.Elems)
	} else if u, y := l1.Elems[0].(unexpanded); !y { // 1
		ctx.err("%T %v", l1.Elems[0], l1.Elems[0])
	} else if s := u.Value.string(ctx); s != "-x" {
		ctx.err("%T %v → %s", u.Value, u.Value, s)
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)x" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v", l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	} else if u, y := l1.Elems[1].(unexpanded); !y { // 2
		ctx.err("%T %v", l1.Elems[1], l1.Elems[1])
	} else if s := u.Value.string(ctx); s != "-y" {
		ctx.err("%T %v → %s", u.Value, u.Value, s)
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)y" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v", l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	} else if u, y := l1.Elems[2].(unexpanded); !y { // 3
		ctx.err("%T %v", l1.Elems[2], l1.Elems[2])
	} else if s := u.Value.string(ctx); s != "-z" {
		ctx.err("%T %v → %s", u.Value, u.Value, s)
	} else if l2, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := l2.String(); s != "&(.test.h)z" {
		ctx.err("%v → %v", l2, s)
	} else if len(l2.Elems) != 2 {
		ctx.err("%v", l2.Elems)
	} else if u, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if s := c.String(); s != "&(.test.h)" {
		ctx.err("%v → %v", c, s)
	}
	// NOTE: check def again to assure it's not affected
	if d := ctx.def(".test.1"); d == nil || d.value == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if s := d.value.String(); s != "x $(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v", d.value, d.value)
	} else if s := d.value.string(ctx); s != "x -" {
		ctx.err("%T %v → %s", d.value, d.value, s)
	} else if d.value != test_1_value {
		ctx.err("%T %v - %v", d.value, d.value, test_1_value)
	} else if d.value.String() != test_1_str {
		ctx.err("%T %v - %v", d.value, d.value, test_1_str)
	} else if test_1_value == nil || test_1_value.String() != test_1_str {
		ctx.err("%T %v - %v", test_1_value, test_1_value, test_1_str)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%T %v", d.value, d.value)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if _, y := l.Elems[0].(*bareword); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if s := l.Elems[1].String(); s != "$(foreach $1,&(.test.h)$_)" {
		ctx.err("%T %v - %v", l.Elems[1], l.Elems[1], test_1_value)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x xq xp x&(.test.h)a x&(.test.h)b x&(.test.h)c" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"a", "b", "c"}, expandDefOriginOff); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x xq xp x&(.test.h)x x&(.test.h)y x&(.test.h)z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"x", "y", "z"}, expandDefOriginOff); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"a", "b", "c"}, expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x xq xp x&(.test.h)a x&(.test.h)b x&(.test.h)c" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"x", "y", "z"}, expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x xq xp x&(.test.h)x x&(.test.h)y x&(.test.h)z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"a", "b", "c"}, expandClosure|expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}
	if v := ctx.get(".test.2", []string{"x", "y", "z"}, expandClosure|expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.2"))
	}

	if d := ctx.def(".test.21"); d == nil {
		ctx.err("%v", ctx.def(".test.21"))
	} else if d.value == nil {
		ctx.err("%v", d) // "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"
	} else if s := d.value.String(); s != "x $(foreach q p $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v → %s ; %v", d.value, d.value, s, d)
	} else if s := d.value.string(ctx); s != "x xq xp x-" {
		ctx.err("%T %v → %s ; %v", d.value, d.value, s, d)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%T %v ; %v", d.value, d.value, d)
	} else if len(l.Elems) != 2 {
		ctx.err("%v ; %v", l.Elems, d) // "$(foreach q p $(foreach $1,&(.test.h)$_),x$_)"
	} else if s := l.Elems[1].String(); s != "$(foreach q p $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v ; %v", l.Elems[1], l.Elems[1], d)
	} else if u, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v ; %v", l.Elems[1], l.Elems[1], d)
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v ; %v", u.Value, u.Value, d)
	} else if len(d.a) != 2 {
		ctx.err("%v ; %v", d, d.a)
	} else if u, y := d.a[0].(unexpanded); !y {
		ctx.err("%v ; %T %v", d, d.a[0], d.a[0])
	} else if l, y := u.Value.(*list); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if len(l.Elems) != 3 {
		ctx.err("%v : %v", d, l.Elems)
	} else if l.Elems[0].String() != "q" {
		ctx.err("%v : %v", d, l.Elems[0])
	} else if l.Elems[1].String() != "p" {
		ctx.err("%v : %v", d, l.Elems[1]) // "$(foreach $1,&(.test.h)$_)"
	} else if l.Elems[2].String() != "$(foreach $1,-$_)" {
		ctx.err("%v : %v", d, l.Elems[2])
	} else if _, y := l.Elems[2].(unexpanded); !y {
		ctx.err("%v : %T %v", d, l.Elems[2], l.Elems[2])
	}
	if v := ctx.get(".test.21"); v == nil {
		ctx.err("%v", ctx.def(".test.21"))
	} else if v != ctx.def(".test.21").value {
		ctx.err("%T %v → %v", v, v, ctx.def(".test.21")) // "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"
	} else if v.String() != "x $(foreach q p $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	} else if t := v.expand(ctx, /* expandUnexpandedForth */expandZero); t == nil {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21")) // "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"
	} else if s := t.String(); s != "x $(foreach q p $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	} else if t := v.expand(ctx, expandUnexpandedForth); t == nil {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	} else if s := t.String(); s != "x xq xp x-$1" { // "x xq xp x&(.test.h)$1"
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	}
	if v := ctx.get(".test.21", expandUnexpandedForth); v == nil {
		ctx.err("%v", ctx.def(".test.21"))
	} else if v.String() != "x xq xp x-$1" { // "x xq xp x&(.test.h)$1"
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	} else if t := v.expand(ctx, expandUnexpandedForth); t == nil {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	} else if t.String() != "x xq xp x-$1" { // "x xq xp x&(.test.h)$1"
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	}
	if v := ctx.get(".test.21"); v == nil {
		ctx.err("%v", ctx.def(".test.21")) // "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"
	} else if v.String() != "x $(foreach q p $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	} else if t := v.expand(ctx, expandDelegate|expandUnexpandedForth); t == nil {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	} else if t.String() != "x xq xp x-$1" { // "x xq xp x&(.test.h)$1"
		ctx.err("%v → %T %v ; %v", v, t, t, ctx.def(".test.21"))
	} else if t.string(ctx) != "x xq xp x-" {
		ctx.err("%v → %T %v ; %v", v, t, t, ctx.def(".test.21"))
	}
	if v := ctx.get(".test.21", expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.21")) // "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"
	} else if v.String() != "x $(foreach q p $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%v → %s ; %v", v, s, ctx.def(".test.21"))
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if l, y := u.Value.(*list); !y {
		ctx.err("%T %v ; %v", u.Value, u.Value, ctx.def(".test.21"))
	} else if len(l.Elems) != 2 {
		ctx.err("%v ; %v", l.Elems, ctx.def(".test.21")) // "$(foreach q p $(foreach $1,&(.test.h)$_),x$_)"
	} else if l.Elems[1].String() != "$(foreach q p $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v ; %v", l.Elems[1], l.Elems[1], ctx.def(".test.21"))
	} else if u, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v ; %v", l.Elems[1], l.Elems[1], ctx.def(".test.21"))
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v ; %v", u.Value, u.Value, ctx.def(".test.21"))
	} else if len(d.a) != 2 {
		ctx.err("%v ; %v", d, d.a)
	} else if u, y := d.a[0].(unexpanded); !y {
		ctx.err("%v ; %T %v", d, d.a[0], d.a[0])
	} else if l, y := u.Value.(*list); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if len(l.Elems) != 3 {
		ctx.err("%v : %v", d, l.Elems)
	} else if l.Elems[0].String() != "q" {
		ctx.err("%v : %v", d, l.Elems[0])
	} else if l.Elems[1].String() != "p" {
		ctx.err("%v : %v", d, l.Elems[1]) // "$(foreach $1,&(.test.h)$_)"
	} else if l.Elems[2].String() != "$(foreach $1,-$_)" {
		ctx.err("%v : %v", d, l.Elems[2])
	} else if _, y := l.Elems[2].(unexpanded); !y {
		ctx.err("%v : %T %v", d, l.Elems[2], l.Elems[2])
	}
	if v := ctx.get(".test.21", expandDelegate|expandUnexpandedForth); v == nil {
		ctx.err("%v", ctx.def(".test.21")) // "x xq xp x&(.test.h)$1"
	} else if v.String() != "x xq xp x-$1" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-" {
		ctx.err("%v → %s ; %v", v, s, ctx.def(".test.21"))
	}
	if v := ctx.get(".test.21", []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", ctx.def(".test.21"))
	} else if v.String() != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	}
	if v := ctx.get(".test.21", []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", ctx.def(".test.21"))
	} else if v.String() != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	}
	if v := ctx.get(".test.21", []string{"a", "b", "c"}, expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.21"))
	} else if v.String() != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-a x-b x-c" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	}
	if v := ctx.get(".test.21", []string{"x", "y", "z"}, expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.21"))
	} else if v.String() != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.21"))
	} else if s := v.string(ctx); s != "x xq xp x-x x-y x-z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.21"))
	}

	if v := ctx.get(".test.22"); v == nil {
		ctx.err("%v", ctx.def(".test.22")) // "x $(foreach - $(foreach $1,&(.test.h)$_),x$_)"
	} else if v.String() != "x $(foreach - $(foreach $1,-$_),x$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.22"))
	} else if s := v.string(ctx); s != "x x- x-" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.22"))
	}
	if v := ctx.get(".test.22", "a"); v == nil {
		ctx.err("%v", ctx.def(".test.22"))
	} else if v.String() != "x x- x-a" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.22"))
	} else if s := v.string(ctx); s != "x x- x-a" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.22"))
	}

	if v := ctx.get(".test.3"/* , expandDebug */); v == nil {
		ctx.err("%v", ctx.def(".test.3"))
	} else if v.String() != "x $(foreach $1,&(.test.$_)$1) y $(foreach $1,$(value(-c) .test.$_)$1) z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.3"))
	} else if s := v.string(ctx); s != "x y z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.3"))
	}
	if v := ctx.get(".test.3", []string{"foo", "bar"}/* , expandDebug */); v == nil {
		ctx.err("%v", ctx.def(".test.3"))
	} else if v.String() != "x &(.test.foo)foo bar &(.test.bar)foo bar y &(.test.foo)foo bar &(.test.bar)foo bar z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.3"))
	} else if s := v.string(ctx); s != "x foo bar foo bar y foo bar foo bar z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.3"))
	}
	if v := ctx.get(".test.3", []string{"foo", "bar"}, expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.3"))
	} else if v.String() != "x &(.test.foo)foo bar &(.test.bar)foo bar y &(.test.foo)foo bar &(.test.bar)foo bar z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.3"))
	} else if s := v.string(ctx); s != "x foo bar foo bar y foo bar foo bar z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.3"))
	}
	if v := ctx.get(".test.3", []string{"foo", "bar"}, expandClosure); v == nil {
		ctx.err("%v", ctx.def(".test.3"))
	} else if v.String() != "x &(.test.foo)foo bar &(.test.bar)foo bar y &(.test.foo)foo bar &(.test.bar)foo bar z" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.3"))
	} else if s := v.string(ctx); s != "x foo bar foo bar y foo bar foo bar z" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.3"))
	}
	if false { // TODO: add expandClosureKept and expandDelegateKept
		if v := ctx.get(".test.3", []string{"foo", "bar"}, expandClosure|expandDebug); v == nil {
			ctx.err("%v", ctx.def(".test.3"))
		} else if v.String() != "x foo bar foo bar y foo bar foo bar z" {
			ctx.err("%T %v ; %v", v, v, ctx.def(".test.3"))
		} else if s := v.string(ctx); s != "x foo bar foo bar y foo bar foo bar z" {
			ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.3"))
		}
		if v := ctx.get(".test.3", []string{"foo", "bar"}, expandClosure|expandDelegate|expandDebug); v == nil {
			ctx.err("%v", ctx.def(".test.3"))
		} else if v.String() != "x foo bar foo bar y foo bar foo bar z" {
			ctx.err("%T %v ; %v", v, v, ctx.def(".test.3"))
		} else if s := v.string(ctx); s != "x foo bar foo bar y foo bar foo bar z" {
			ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.3"))
		}
	}

	if v := ctx.get(".test.4"); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "$(foreach $1 $2,&(.test.$_.$(or $4,$3)))" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y"); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "&(.test.x.$(or $4,$3)) &(.test.y.$(or $4,$3))" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", "a", "b"); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "&(.test.x.b) &(.test.y.b)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xb yb" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", "a", []string{}); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "&(.test.x.a) &(.test.y.a)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xa ya" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", []string{}, "b"); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "&(.test.x.b) &(.test.y.b)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xb yb" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", "a", "b", expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "&(.test.x.b) &(.test.y.b)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xb yb" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", []Value{}, "b", expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "&(.test.x.b) &(.test.y.b)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xb yb" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", "a", []Value{}, expandClosure|expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "xa ya" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xa ya" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", "a", "b", expandClosure|expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "xb yb" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xb yb" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", []Value{}, "b", expandClosure|expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "xb yb" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xb yb" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}
	if v := ctx.get(".test.4", "x", "y", "a", []Value{}, expandClosure|expandDelegate); v == nil {
		ctx.err("%v", ctx.def(".test.4"))
	} else if v.String() != "xa ya" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.4"))
	} else if s := v.string(ctx); s != "xa ya" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.4"))
	}

	if d := ctx.def(".test.5"); d == nil || d.value == nil {
		ctx.err("%v", ctx.def(".test.5"))
	} else if s := d.value.String(); s != "&(.test.x do.smart)" {
		ctx.err("%T %v ; %v", d.value, d.value, ctx.def(".test.5"))
	} else if s := d.value.string(ctx); s != "true" {
		ctx.err("%T %v → %s ; %v", d.value, d.value, s, ctx.def(".test.5"))
	} else if u, y := d.value.(unexpanded); !y {
		ctx.err("%T %v", d.value, d.value)
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if c.x.String() != ".test.x" {
		ctx.err("%T %v", c.x, c.x)
	} else if s := c.string(ctx); s != "true" {
		ctx.err("%v → %s", c, s)
	}
	if v := ctx.get(".test.5"); v == nil {
		ctx.err("%v", ctx.def(".test.5"))
	} else if v.String() != "&(.test.x do.smart)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.5"))
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.5"))
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if c.x.String() != ".test.x" {
		ctx.err("%T %v", c.x, c.x)
	} else if s := c.string(ctx); s != "true" {
		ctx.err("%v → %s", c, s)
	}
	if v := ctx.get(".test.5", "xxx"); v == nil {
		ctx.err("%v", ctx.def(".test.5"))
	} else if v.String() != "&(.test.x do.smart)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.5"))
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.5"))
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if c, y := u.Value.(*closure); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if c.x.String() != ".test.x" {
		ctx.err("%T %v", c.x, c.x)
	} else if s := c.string(ctx); s != "true" {
		ctx.err("%v → %s", c, s)
	}
	if v := ctx.get(".test.5", "xxx", expandClosure); v == nil {
		ctx.err("%v", ctx.def(".test.5"))
	} else if v.String() != "true{}" { // do.smart
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.5"))
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.5"))
	} else if b, y := v.(*boolean); !y {
		ctx.err("%T %v", v, v)
	} else if !b.bool {
		ctx.err("%v", b)
	}

	if v := ctx.get(".test.x", "do.smart"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "true{}" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.x"))
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.x"))
	} else if b, y := v.(*boolean); !y {
		ctx.err("%T %v", v, v)
	} else if !b.bool {
		ctx.err("%v", b)
	}
}

func testForeach1(ctx *testcase) {
	if v := ctx.get(".test.4"); v == nil {
		ctx.err(".test.4")
	} else if v.String() != "&(.test.s) $(value .test~&(.test.s)) $(foreach $1 B b,$(value(-c) .test.$_) $(value(-c) .test~&(.test.s).$_))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foo test-foo test-B test-foo-B test-b test-foo-b" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get(".test.4", "a"); v == nil {
		ctx.err(".test.4")
	} else if v.String() != "&(.test.s) $(value .test~&(.test.s)) test-a $(value(-c) .test~&(.test.s).a) test-B $(value(-c) .test~&(.test.s).B) test-b $(value(-c) .test~&(.test.s).b)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foo test-foo test-a test-foo-a test-B test-foo-B test-b test-foo-b" {
		ctx.err("%T %v → %s", v, v, s)
	}
}

func testForeach2(ctx *testcase) {
	if v := ctx.get(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "$(foreach x$1 y$1 z$1 $(foreach xx$2 yy$2,a$_),b$_)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.1"))
	} else if s := v.string(ctx); s != "bx by bz baxx bayy" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != "$(foreach x$1 y$1 z$1 $(foreach xx$2 yy$2,a$_),b$_) $(foreach &(.test.foreach.x) $(foreach $1 $2,$(value(-c) .test.foreach.x.$_)),-x$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "bx by bz baxx bayy -xVW -x -x" { // FIXME: exclude "-x -x" (xEmpty)
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if v.String() != "$(foreach x$1 y$1 z$1 axx4 ayy4,b$_) $(foreach &(.test.foreach.x) $(foreach $1 4,$(value(-c) .test.foreach.x.$_)),-x$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "bx by bz baxx4 bayy4 -xVW -xW" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get(".test.4"); v == nil {
		ctx.err(".test.4")
	} else if v.String() != "$(foreach $1 $2,$(.test.foreach.d.$_))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
	}
}

func testForeach3(ctx *testcase) {
	if v := ctx.get(".test.x"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "$(foreach $1 $2,std=&(.test.$_))" {
		ctx.err("%T %v", v, v) // "$(foreach $1 $2,$(addprefix std=,&(.test.$_)))"
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
		if l, y := v.(*list); y { for i, elem := range l.Elems {
			ctx.err("%v %v %v → %s", i, typeof(elem), elem, elem.string(ctx))
		}}
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if d.x.String() != "foreach" {
		ctx.err("%T %v", d.x, d.x)
	}
	if v := ctx.get(".test.x", expandUnexpandedForth); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "std=&(.test.$1) std=&(.test.$2)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.x"))
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if p1, y := l.Elems[0].(paircomp); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if p2, y := l.Elems[1].(paircomp); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if s := p1.string(ctx); s != "" {
		ctx.err("%v → %s", p1, s)
	} else if s := p2.string(ctx); s != "" {
		ctx.err("%v → %s", p2, s)
	} else if s := p1.pair.string(ctx); s != "std=" {
		ctx.err("%v → %s", p1, s)
	} else if s := p2.pair.string(ctx); s != "std=" {
		ctx.err("%v → %s", p2, s)
	}
	if v := ctx.get(".test.x", "a"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "$(foreach a $2,std=&(.test.$_))" {
		ctx.err("%T %v", v, v) // "$(foreach a $2,$(addprefix std=,&(.test.$_)))"
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
		if l, y := v.(*list); y { for i, elem := range l.Elems {
			ctx.err("%v %v %v → %s", i, typeof(elem), elem, elem.string(ctx))
		}}
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	}
	if v := ctx.get(".test.x", "a", expandUnexpandedForth); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "std=&(.test.a)" { // FIXME: consider std=&(.test.$2)
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
	} else if false {
		if l, y := v.(*list); !y {
			ctx.err("%T %v", v, v)
		} else if len(l.Elems) != 2 {
			ctx.err("%v", l.Elems)
		} else if p1, y := l.Elems[0].(paircomp); !y {
			ctx.err("%T %v", l.Elems[0], l.Elems[0])
		} else if p2, y := l.Elems[1].(paircomp); !y {
			ctx.err("%T %v", l.Elems[1], l.Elems[1])
		} else if s := p1.string(ctx); s != "" {
			ctx.err("%v → %s", p1, s)
		} else if s := p2.string(ctx); s != "" {
			ctx.err("%v → %s", p2, s)
		} else if s := p1.pair.string(ctx); s != "std=" {
			ctx.err("%v → %s", p1, s)
		} else if s := p2.pair.string(ctx); s != "std=" {
			ctx.err("%v → %s", p2, s)
		}
	} else if p1, y := v.(paircomp); !y {
		ctx.err("%T %v", v, v)
	} else if s := p1.string(ctx); s != "" {
		ctx.err("%v → %s", p1, s)
	} else if s := p1.pair.string(ctx); s != "std=" {
		ctx.err("%v → %s", p1, s)
	}
	if v := ctx.get(".test.x", "a", nil); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "std=&(.test.a)" { // std=&(.test.)
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
		if l, y := v.(*list); y { for i, elem := range l.Elems {
			ctx.err("%v %v %v → %s", i, typeof(elem), elem, elem.string(ctx))
		}}
	} else if false {
		if l, y := v.(*list); !y {
			ctx.err("%T %v", v, v)
		} else if len(l.Elems) != 2 {
			ctx.err("%v", l.Elems)
		} else if p1, y := l.Elems[0].(paircomp); !y {
			ctx.err("%T %v", l.Elems[0], l.Elems[0])
		} else if p2, y := l.Elems[1].(paircomp); !y {
			ctx.err("%T %v", l.Elems[1], l.Elems[1])
		} else if s := p1.string(ctx); s != "" {
			ctx.err("%v → %s", p1, s)
		} else if s := p2.string(ctx); s != "" {
			ctx.err("%v → %s", p2, s)
		} else if s := p1.pair.string(ctx); s != "std=" {
			ctx.err("%v → %s", p1, s)
		} else if s := p2.pair.string(ctx); s != "std=" {
			ctx.err("%v → %s", p2, s)
		}
	} else if p, y := v.(paircomp); !y {
		ctx.err("%T %v", v, v)
	} else if s := p.string(ctx); s != "" {
		ctx.err("%v → %s", p, s)
	} else if s := p.pair.string(ctx); s != "std=" {
		ctx.err("%v → %s", p, s)
	}
	if v := ctx.get(".test.x", nil, "b"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "std=&(.test.b)" { // std=&(.test.)
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
		if l, y := v.(*list); y { for i, elem := range l.Elems {
			ctx.err("%v %v %v → %s", i, typeof(elem), elem, elem.string(ctx))
		}}
	} else if false {
		if l, y := v.(*list); !y {
			ctx.err("%T %v", v, v)
		} else if len(l.Elems) != 2 {
			ctx.err("%v", l.Elems)
		} else if p1, y := l.Elems[0].(paircomp); !y {
			ctx.err("%T %v", l.Elems[0], l.Elems[0])
		} else if p2, y := l.Elems[1].(paircomp); !y {
			ctx.err("%T %v", l.Elems[1], l.Elems[1])
		} else if s := p1.string(ctx); s != "" {
			ctx.err("%v → %s", p1, s)
		} else if s := p2.string(ctx); s != "" {
			ctx.err("%v → %s", p2, s)
		} else if s := p1.pair.string(ctx); s != "std=" {
			ctx.err("%v → %s", p1, s)
		} else if s := p2.pair.string(ctx); s != "std=" {
			ctx.err("%v → %s", p2, s)
		}
	} else if p, y := v.(paircomp); !y {
		ctx.err("%T %v", v, v)
	} else if s := p.string(ctx); s != "" {
		ctx.err("%v → %s", p, s)
	} else if s := p.pair.string(ctx); s != "std=" {
		ctx.err("%v → %s", p, s)
	}
	if v := ctx.get(".test.x", "a", "b"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "std=&(.test.a) std=&(.test.b)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
		if l, y := v.(*list); y { for i, elem := range l.Elems {
			ctx.err("%v %v %v → %s", i, typeof(elem), elem, elem.string(ctx))
		}}
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if p1, y := l.Elems[0].(paircomp); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if p2, y := l.Elems[1].(paircomp); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if s := p1.string(ctx); s != "" {
		ctx.err("%v → %s", p1, s)
	} else if s := p2.string(ctx); s != "" {
		ctx.err("%v → %s", p2, s)
	} else if s := p1.pair.string(ctx); s != "std=" {
		ctx.err("%v → %s", p1, s)
	} else if s := p2.pair.string(ctx); s != "std=" {
		ctx.err("%v → %s", p2, s)
	}

	if v := ctx.get(".test.y"); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "$(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_)))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	}
	if true { ctx.universe().ddd = "test.if" }
	if v := ctx.get(".test.y", expandUnexpandedForth); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "$(if &(.test.$1),std=&(.test.$1)) $(if &(.test.$2),std=&(.test.$2))" {
		ctx.err("%T %v  ;  %v", v, v, ctx.def(".test.y"))
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s ; %v", v, v, s, ctx.def(".test.y"))
	} else if _, y := v.(*none); y {
		ctx.Logf("%T %v", v, v) // FIXME: ...
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if u1, y := l.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if u2, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if s := u1.string(ctx); s != "" {
		ctx.err("%v → %s", u1.Value, s)
	} else if s := u2.string(ctx); s != "" {
		ctx.err("%v → %s", u1.Value, s)
	} else if p1, y := u1.Value.(*list); !y { // delegate
		ctx.err("%T %v", u1.Value, u1.Value)
	} else if p2, y := u2.Value.(*list); !y { // delegate
		ctx.err("%T %v", u2.Value, u2.Value)
	} else if s := p1.string(ctx); s != "" {
		ctx.err("%v → %s", p1, s)
	} else if s := p2.string(ctx); s != "" {
		ctx.err("%v → %s", p2, s)
	}
	if v := ctx.get(".test.y", "if.x"); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "$(foreach if.x $2,$(if &(.test.$_),std=&(.test.$_)))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "std=xxx" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	}
	if v := ctx.get(".test.y", "if.x", expandUnexpandedForth); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "$(if &(.test.if.x),std=&(.test.if.x))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "std=xxx" {
		ctx.err("%T %v → %s", v, v, s)
	} else if false {
		if l, y := v.(*list); !y {
			ctx.err("%T %v", v, v)
		} else if len(l.Elems) != 2 {
			ctx.err("%v", l.Elems)
		} else if l.Elems[0].String() != "$(if &(.test.if.x),std=&(.test.if.x))" {
			ctx.err("%T %v", l.Elems[0], l.Elems[0])
		} else if l.Elems[1].String() != "$(if &(.test.$2),std=&(.test.$2))" {
			ctx.err("%T %v", l.Elems[1], l.Elems[1])
		} else if s := l.Elems[0].string(ctx); s != "std=xxx" {
			ctx.err("%T %v → %s", l.Elems[0], l.Elems[0], s)
		} else if s := l.Elems[1].string(ctx); s != "" {
			ctx.err("%T %v → %s", l.Elems[1], l.Elems[1], s)
		} else if u1, y := l.Elems[0].(unexpanded); !y {
			ctx.err("%T %v", l.Elems[0], l.Elems[0])
		} else if u2, y := l.Elems[1].(unexpanded); !y {
			ctx.err("%T %v", l.Elems[1], l.Elems[1])
		} else if _, y := u1.Value.(*list); !y { // delegate
			ctx.err("%T %v", u1.Value, u1.Value)
		} else if _, y := u2.Value.(*list); !y { // delegate
			ctx.err("%T %v", u2.Value, u2.Value)
		}
	} else if true {
		if u1, y := v.(unexpanded); !y {
			ctx.err("%T %v", v, v)
		} else if _, y := u1.Value.(*list); !y { // delegate
			ctx.err("%T %v", u1.Value, u1.Value)
		}
	} else if _, y := v.(*pair); !y {
		ctx.err("%T %v", v, v)
	}
	if v := ctx.get(".test.y", "if.x", nil); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "$(if &(.test.if.x),std=&(.test.if.x))" { // $(if &(.test.),std=&(.test.))
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "std=xxx" {
		ctx.err("%T %v → %s", v, v, s)
	} else if false {
		if l, y := v.(*list); !y {
			ctx.err("%T %v", v, v)
		} else if len(l.Elems) != 2 {
			ctx.err("%v", l.Elems)
		} else if s := l.Elems[0].string(ctx); s != "std=xxx" {
			ctx.err("%T %v → %s", l.Elems[0], l.Elems[0], s)
		} else if s := l.Elems[1].string(ctx); s != "" {
			ctx.err("%T %v → %s", l.Elems[1], l.Elems[1], s)
		} else if u1, y := l.Elems[0].(unexpanded); !y {
			ctx.err("%T %v", l.Elems[0], l.Elems[0])
		} else if u2, y := l.Elems[1].(unexpanded); !y {
			ctx.err("%T %v", l.Elems[1], l.Elems[1])
		} else if _, y := u1.Value.(*list); !y { // delegate
			ctx.err("%T %v", u1.Value, u1.Value)
		} else if _, y := u2.Value.(*list); !y { // delegate
			ctx.err("%T %v", u2.Value, u2.Value)
		}
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := u.Value.(*list); !y {
		ctx.err("%T %v", u.Value, u.Value)
	}
	if v := ctx.get(".test.y", nil, "if.y"); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "$(if &(.test.if.y),std=&(.test.if.y))" { // $(if &(.test.),std=&(.test.))
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "std=yyy" {
		ctx.err("%T %v → %s", v, v, s)
	} else if false {
		if l, y := v.(*list); !y {
			ctx.err("%T %v", v, v)
		} else if len(l.Elems) != 2 {
			ctx.err("%v", l.Elems)
		} else if s := l.Elems[0].string(ctx); s != "" {
			ctx.err("%T %v → %s", l.Elems[0], l.Elems[0], s)
		} else if s := l.Elems[1].string(ctx); s != "std=yyy" {
			ctx.err("%T %v → %s", l.Elems[1], l.Elems[1], s)
		} else if u1, y := l.Elems[0].(unexpanded); !y {
			ctx.err("%T %v", l.Elems[0], l.Elems[0])
		} else if u2, y := l.Elems[1].(unexpanded); !y {
			ctx.err("%T %v", l.Elems[1], l.Elems[1])
		} else if _, y := u1.Value.(*list); !y { // delegate
			ctx.err("%T %v", u1.Value, u1.Value)
		} else if _, y := u2.Value.(*list); !y { // delegate
			ctx.err("%T %v", u2.Value, u2.Value)
		}
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := u.Value.(*list); !y {
		ctx.err("%T %v", u.Value, u.Value)
	}
	if v := ctx.get(".test.y", "if.x", "if.y"); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "$(if &(.test.if.x),std=&(.test.if.x)) $(if &(.test.if.y),std=&(.test.if.y))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "std=xxx std=yyy" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if s := l.Elems[0].string(ctx); s != "std=xxx" {
		ctx.err("%v → %s", l.Elems[0], s)
	} else if s := l.Elems[1].string(ctx); s != "std=yyy" {
		ctx.err("%v → %s", l.Elems[1], s)
	} else if u1, y := l.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if u2, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if _, y := u1.Value.(*list); !y { // delegate
		ctx.err("%T %v", u1.Value, u1.Value)
	} else if _, y := u2.Value.(*list); !y { // delegate
		ctx.err("%T %v", u2.Value, u2.Value)
	}
	if v := ctx.get(".test.y", "if.x", "if.y", expandClosure); v == nil {
		ctx.err("%v", ctx.def(".test.y"))
	} else if v.String() != "std=xxx std=yyy" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "std=xxx std=yyy" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	} else if s := l.Elems[0].string(ctx); s != "std=xxx" {
		ctx.err("%v → %s", l.Elems[0], s)
	} else if s := l.Elems[1].string(ctx); s != "std=yyy" {
		ctx.err("%v → %s", l.Elems[1], s)
	} else if _, y := l.Elems[0].(*pair); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if _, y := l.Elems[1].(*pair); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	}
}

func testForeach4(ctx *testcase) {
	if v := ctx.get(".test.1", "a", "b"); v == nil {
		ctx.err("%v", ctx.def(".test.1"))
	} else if v.String() != "Xxa Xxb" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "Xxa Xxb" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get(".test.2", "a", "b"); v == nil {
		ctx.err("%v", ctx.def(".test.2"))
	} else if v.String() != "X&(.test.xa) X&(.test.xb)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "X~1~ x- x- ~1~x- ~1~x- X~2~ x- x- ~2~x- ~2~x- ~~~~ ~~~~" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); y {
		if l, y := u.Value.(*list); !y {
			ctx.err("%T %v", u.Value, u.Value)
		} else if len(l.Elems) != 2 {
			ctx.err("%T %v", v, l.Elems)
		}
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v", v, l.Elems)
	}
	if v := ctx.get(".test.3", "a", "b"); v == nil {
		ctx.err("%v", ctx.def(".test.3"))
	} else if v.String() != "YX&(.test.xa) YX&(.test.xb)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "YX~1~ x- x- ~1~x- ~1~x- YX~2~ x- x- ~2~x- ~2~x- ~~~~ ~~~~" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); y {
		if l, y := u.Value.(*list); !y {
			ctx.err("%T %v", u.Value, u.Value)
		} else if len(l.Elems) != 2 {
			ctx.err("%T %v", v, l.Elems)
		}
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v", v, l.Elems)
	}

	if v := ctx.get(".test.x"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "$(foreach $(foreach $1 $2,x$_),X$_) $(foreach $(foreach $1 $2,&(.test.x$_)),X$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "Xx Xx" { // X X
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get(".test.x", "a", "b"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "Xxa Xxb X&(.test.xa) X&(.test.xb)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "Xxa Xxb X~1~ x- x- ~1~x- ~1~x- X~2~ x- x- ~2~x- ~2~x- ~~~~ ~~~~" {
		ctx.err("%T %v → %s", v, v, s)
	} else if u, y := v.(unexpanded); y {
		if l, y := u.Value.(*list); !y {
			ctx.err("%T %v", u.Value, u.Value)
		} else if len(l.Elems) != 2 {
			ctx.err("%T %v", v, l.Elems)
		}
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v %v", v, l.Elems, len(l.Elems))
	}
}

func testForeach5(ctx *testcase) {
	if v := ctx.get(".test.x"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "$(foreach $1 $2,&(.test.x.$_) &(.test.x.&(.test.o).$_))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get(".test.x", "a", "b"); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "&(.test.x.a) &(.test.x.&(.test.o).a) &(.test.x.b) &(.test.x.&(.test.o).b)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a~ ~a x.o.a b~ ~b x.o.b" {
		ctx.err("%T %v → %s", v, v, s)
		// } else if t := xa(ctx, v, "a", "b", expandDelegate); t == nil {
		// 	ctx.err("%T %v", v, v)
		// } else if t.String() != "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -bo&(.test.x.o.a) -bo&(.test.x.o.b) ~b x.o.b" {
		// 	ctx.err("%T %v", t, t) // &(.test.x.a) &(.test.x.&(.test.o).a) &(.test.x.b) &(.test.x.&(.test.o).b)
		// } else if s := t.string(ctx); s != "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b" {
		// 	ctx.err("%T %v → %s", t, t, s)
	} else if t := xa(ctx, v, "a", "b", expandClosure|expandDelegate); t == nil {
		ctx.err("%T %v", v, v)
	} else if t.String() != "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b" {
		ctx.err("%T %v", t, t)
	} else if s := t.string(ctx); s != "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b" {
		ctx.err("%T %v → %s", t, t, s)
	}

	if v := ctx.get(".test.x", "a", "b", expandClosure); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if l.Len() != 10 {
		ctx.err("%v %v", l.Elems, l.Len())
	} else if l.Elems[1].String() != "-aox.o.a" {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if l.Elems[2].String() != "-aox.o.b" {
		ctx.err("%T %v", l.Elems[2], l.Elems[2])
	} else if f, y := l.Elems[1].(flag); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if l, y := f.Value.(*barecomp); !y {
		ctx.err("%T %v", f.Value, f.Value)
	} else if l.Len() != 2 {
		ctx.err("%v %v", l.Elems, l.Len())
	} else if l.Elems[0].String() != "ao" {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if l.Elems[1].String() != "x.o.a" {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if _, y := l.Elems[1].(*barecomp); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	}

	if true {} else
	if v := ctx.get(".test.x", "a", "b", expandClosure); v == nil {
		ctx.err("%v", ctx.def(".test.x"))
	} else if v.String() != "a~ -ao$(.test.x.o.a) -ao$(.test.x.o.b) ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a~ -ao -ao ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if l.Len() != 10 {
		ctx.err("%v %v", l.Elems, l.Len())
	} else if l.Elems[1].String() != "-ao$(.test.x.o.a)" {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if l.Elems[2].String() != "-ao$(.test.x.o.b)" {
		ctx.err("%T %v", l.Elems[2], l.Elems[2])
	} else if f, y := l.Elems[1].(flag); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if u, y := f.Value.(unexpanded); !y {
		ctx.err("%T %v", f.Value, f.Value)
	} else if l, y := u.Value.(*barecomp); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if l.Len() != 2 {
		ctx.err("%v %v", l.Elems, l.Len())
	} else if l.Elems[0].String() != "ao" {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if l.Elems[1].String() != "$(.test.x.o.a)" {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if u, y := l.Elems[1].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if _, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	}
}

func testAddPrefix(ctx *testcase) {
	if v := ctx.get("val1"); v == nil {
		ctx.err("val1")
	} else if v.String() != /* "$(addprefix -std=,foo)" */"-std=foo" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-std=foo" {
		ctx.err("%T %v → %s", v, v, s)
	} else if p, y := v.(*pair); !y {
		ctx.err("%T %v", v, v)
	} else if s := p.Key.String(); s != "-std" {
		ctx.err("%T %v ; %T %v", v, v, p.Key, p.Key)
	} else if s := p.Value.String(); s != "foo" {
		ctx.err("%T %v ; %T %v", v, v, p.Value, p.Value)
	}

	if v := ctx.get("val2"); v == nil {
		ctx.err("val2")
	} else if v.String() != /* "$(addprefix -std=,foo bar foobar)" */"-std=foo -std=bar -std=foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-std=foo -std=bar -std=foobar" {
		ctx.err("%T %v → %s", v, v, s)
	}
}

func testPushContext(ctx *testcase) {
	if v := ctx.get("foo"); v == nil {
		ctx.err("foo")
	} else if v.String() != "foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("foo1"); v == nil {
		ctx.err("foo1")
	} else if v.String() != "foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("foo2"); v == nil {
		ctx.err("foo2")
	} else if v.String() != "x" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "x" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("foo3"); v == nil {
		ctx.err("foo3")
	} else if v.String() != "foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v → %s", v, v, s)
	}
}

func testContains(ctx *testcase) {
	if v := ctx.get(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "true{}" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != "true{}" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if v.String() != "false{}" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "false" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get(".test.4"); v == nil {
		ctx.err(".test.4")
	} else if v.String() != "true{}" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%T %v → %s", v, v, s)
	}
}

func testLogic(ctx *testcase) {
	if v := ctx.get("val1"); v == nil {
		ctx.err("val1")
	} else if v.String() != "a" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val2"); v == nil {
		ctx.err("val2")
	} else if v.String() != "a" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val3"); v == nil {
		ctx.err("val3")
	} else if v.String() != "$(or(-forth) &(none),a)" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val4"); v == nil {
		ctx.err("val4")
	} else if v.String() != "$(or &(none),a)" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val5"); v == nil {
		ctx.err("val5")
	} else if v.String() != "$(or a,&(none))" {
		ctx.err("%T %v", v, v)
	} else if true {
		if _, y := v.(unexpanded); !y {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "a" {
			ctx.err("%T %v → %s", v, v, s)
		}
	} else if _, y := v.(*bareword); !y {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get("val6"); v == nil {
		ctx.err("val6")
	} else if v.String() != "$(and $1,$2,$3)" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get("val6", "a", "b", "c"); v == nil {
		ctx.err("val6")
	} else if v.String() != "c" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*bareword); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "c" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("x0"); v == nil {
		ctx.err("x0")
	} else if v.String() != "(variant/$(or(-forth) $(base &(variant)),bootstrap))" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*group); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "(variant/.)" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("x1"); v == nil {
		ctx.err("x1")
	} else if v.String() != "(variant/bootstrap)" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*group); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "(variant/bootstrap)" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("x2"); v == nil {
		ctx.err("x2")
	} else if v.String() != "variant/bootstrap" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*Path); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "variant/bootstrap" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("x3"); v == nil {
		ctx.err("x3")
	} else if v.String() != "bootstrap" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*bareword); !y {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get("x4"); v == nil {
		ctx.err("x4")
	} else if v.String() != "$(or $(base &(variant)),bootstrap)" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "." {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("x5"); v == nil {
		ctx.err("x5")
	} else if v.String() != "$(base $(or &(variant),bootstrap))" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "bootstrap" {
		ctx.err("%T %v → %s", v, v, s)
	}
}

func testTrimPrefix(ctx *testcase) {
	if v0 := ctx.get("pat0.0"); v0 == nil {
		ctx.err("pat0.0")
	} else if s := v0.string(ctx); s != "**/testdata" {
		ctx.err("%T %v → %s", v0, v0, s)
	} else if v1 := ctx.get("pat0.1"); v1 == nil {
		ctx.err("pat0.1")
	} else if !strings.HasSuffix(v1.string(ctx), "/testdata/trimprefix") {
		ctx.err("%T %v → %s", v1, v1, v1.string(ctx))
	} else if a, b0, c := v0.match(ctx, v1); a {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if b1, y := b0.([]string); !y {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if p := strings.Split(v1.string(ctx), PathSep); len(b1) > 0 && len(b1) != len(p)-1 {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if p[0] != b1[0] {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if strings.Join(b1, PathSep) != strings.Join(p[:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, b1, p)
	} else if len(c) != 1 {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else if c[0] != strings.Join(p[:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else {
		for i, s := range b1 { if s != p[i] {
			ctx.err("%v: %v %v %v", v1, i, s, p)
		}}
		if false { for i, s := range c { if s != p[i] {
			ctx.err("%v: %v %v %v", v1, i, s, p)
		}}}
	}

	if v0 := ctx.get("pat1.0"); v0 == nil {
		ctx.err("pat1.0")
	} else if s := v0.string(ctx); s != "%%/testdata" {
		ctx.err("%T %v → %s", v0, v0, s)
	} else if v1 := ctx.get("pat1.1"); v1 == nil {
		ctx.err("pat1.1")
	} else if !strings.HasSuffix(v1.string(ctx), "/testdata/trimprefix") {
		ctx.err("%T %v → %s", v1, v1, v1.string(ctx))
	} else if a, b0, c := v0.match(ctx, v1); a {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if b1, y := b0.([]string); !y {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if p := strings.Split(v1.string(ctx), PathSep); len(b1) > 0 && len(b1) != len(p)-1 {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if p[0] != b1[0] {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if strings.Join(b1, PathSep) != strings.Join(p[:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, b1, p)
	} else if len(c) != 1 {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else if c[0] != strings.Join(p[:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else {
		for i, s := range b1 { if s != p[i] {
			ctx.err("%v: %v %v %v", v1, i, s, p)
		}}
	}

	if v0 := ctx.get("pat2.0"); v0 == nil {
		ctx.err("pat2.0")
	} else if s := v0.string(ctx); s != "/**/testdata" {
		ctx.err("%T %v → %s", v0, v0, s)
	} else if v1 := ctx.get("pat2.1"); v1 == nil {
		ctx.err("pat2.1")
	} else if !strings.HasSuffix(v1.string(ctx), "/testdata/trimprefix") {
		ctx.err("%T %v → %s", v1, v1, v1.string(ctx))
	} else if a, b0, c := v0.match(ctx, v1); a {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if b1, y := b0.([]string); !y {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if p := strings.Split(v1.string(ctx), PathSep); len(b1) > 0 && len(b1) != len(p)-1 {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if p[0] != b1[0] {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if strings.Join(b1, PathSep) != strings.Join(p[:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, b1, p)
	} else if len(c) != 1 {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else if c[0] != strings.Join(p[1:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else {
		for i, s := range b1 { if s != p[i] {
			ctx.err("%v: %v %v %v", v1, i, s, p)
		}}
	}

	if v0 := ctx.get("pat3.0"); v0 == nil {
		ctx.err("pat3.0")
	} else if s := v0.string(ctx); s != "/%%/testdata" {
		ctx.err("%T %v → %s", v0, v0, s)
	} else if v1 := ctx.get("pat3.1"); v1 == nil {
		ctx.err("pat3.1")
	} else if !strings.HasSuffix(v1.string(ctx), "/testdata/trimprefix") {
		ctx.err("%T %v → %s", v1, v1, v1.string(ctx))
	} else if a, b0, c := v0.match(ctx, v1); a {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if b1, y := b0.([]string); !y {
		ctx.err("%T %v %v ; %v %v %v", v1, v1, v0, a, b0, c)
	} else if p := strings.Split(v1.string(ctx), PathSep); len(b1) > 0 && len(b1) != len(p)-1 {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if p[0] != b1[0] {
		ctx.err("%v %v ; %v %v", v1, v0, strings.Join(b1, PathSep), v1)
	} else if strings.Join(b1, PathSep) != strings.Join(p[:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, b1, p)
	} else if len(c) != 1 {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else if c[0] != strings.Join(p[1:len(p)-1], PathSep) {
		ctx.err("%v %v ; %v %v", v1, v0, c, v1)
	} else {
		for i, s := range b1 { if s != p[i] {
			ctx.err("%v: %v %v %v", v1, i, s, p)
		}}
	}

	if v := ctx.get("val1"); v == nil {
		ctx.err("val1")
	} else if s := v.string(ctx); s != "/trimprefix" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val2"); v == nil {
		ctx.err("val2")
	} else if s := v.string(ctx); s != "trimprefix" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val3"); v == nil {
		ctx.err("val3")
	} else if s := v.string(ctx); s != "trimprefix" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val4"); v == nil {
		ctx.err("val4")
	} else if s := v.string(ctx); s != "trimprefix" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if v := ctx.get("val5"); v == nil {
		ctx.err("val5")
	} else if s := v.string(ctx); s != "trimprefix" {
		ctx.err("%T %v → %s", v, v, s)
	}
}

func testBuiltins(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "-foo=bar -foo=&(bar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-foo=bar" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v %v", v, v, l.Elems)
	} else if _, y := l.Elems[0].(*pair); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if _, y := l.Elems[1].(paircomp); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	}
	if v := ctx.get("val1"); v == nil {
		ctx.err("val1")
	} else if v.String() != "-foo=bar -foo=&(bar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-foo=bar" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v %v", v, v, l.Elems)
	} else if _, y := l.Elems[0].(*pair); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if _, y := l.Elems[1].(paircomp); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if false {
		noted(ctx, "%T %v", l.Elems[0], l.Elems[0])
		noted(ctx, "%T %v", l.Elems[1], l.Elems[1])
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "$(addprefix -foo=,bar &(bar))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-foo=bar" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get("val2"); v == nil {
		ctx.err("val2")
	} else if v.String() != "$(addprefix -foo=,bar &(bar))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-foo=bar" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "foobar foo&(bar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v %v", v, v, l.Elems)
	} else if _, y := l.Elems[0].(*barecomp); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if _, y := l.Elems[1].(precomp); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	}
	if v := ctx.get("val3"); v == nil {
		ctx.err("val3")
	} else if v.String() != "foobar foo&(bar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v → %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v %v", v, v, l.Elems)
	} else if _, y := l.Elems[0].(*barecomp); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if _, y := l.Elems[1].(precomp); !y {
		ctx.err("%T %v", l.Elems[1], l.Elems[1])
	} else if false {
		noted(ctx, "%T %v", l.Elems[0], l.Elems[0])
		noted(ctx, "%T %v", l.Elems[1], l.Elems[1])
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "$(addprefix foo,bar &(bar))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get("val4"); v == nil {
		ctx.err("val4")
	} else if v.String() != "$(addprefix foo,bar &(bar))" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "'foo-bar-xx-yy-zz'" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foo-bar-xx-yy-zz" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get("val6"); v == nil {
		ctx.err("val6")
	} else if v.String() != "$(join foo bar xx yy zz,-)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foo-bar-xx-yy-zz" {
		ctx.err("%T %v → %s", v, v, s)
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "$(join &(target.arch) &(target.vendor) &(target.sys) &(target.abi),-)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foo-bar-a-0" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if v := ctx.get("val7"); v == nil {
		ctx.err("val7")
	} else if v.String() != "$(join &(target.arch) &(target.vendor) &(target.sys) &(target.abi),-)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foo-bar-a-0" {
		ctx.err("%T %v → %s", v, v, s)
	}
	if d := ctx.def("val7.1"); d == nil {
		ctx.err("val7.1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "&(target.arch)-&(XXX)-&(target.vendor)-&(target.sys)-&(target.abi)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foo-bar-a-0" {
		ctx.err("%T %v → %s", v, v, s)
	}

	fullFooTxt := filepath.Join(ctx.WorkDir(), "foo.txt")
	if v := ctx.get("val8"); v == nil {
		ctx.err("val8")
	} else if v.String() != "foo.txt" {
		ctx.err("%T %v", v, v)
	} else if v.string(ctx) != "foo.txt" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%T %v", v, v)
	} else if f := (as{v}.file(ctx)); f == nil {
		ctx.err("%T %v", v, v)
	} else if o, y := (as{v}.fullnameOpt(ctx)); !y || o.Value == nil {
		ctx.err("%T %v ; %T %v %v", v, v, o.Value, o.Value, y)
	} else if f, y := o.Value.(*File); !y || f == nil {
		ctx.err("%T %v", o.Value, o.Value)
	} else if f.fullname() != fullFooTxt {
		ctx.err("%T %v %v", v, v, f.fullname())
	} else if o.cmp(ctx, f) != cmpEqual {
		ctx.err("%T %v %v", v, o, f)
	} else if f.cmp(ctx, o) != cmpEqual {
		ctx.err("%T %v %v", v, o, f)
	} else if cmp(ctx, o, f) != cmpEqual {
		ctx.err("%T %v %v", v, o, f)
	} else if cmp(ctx, f, o) != cmpEqual {
		ctx.err("%T %v %v", v, o, f)
	} else if p := pathStr(ctx, f.position, f.fullname()); o.cmp(ctx, p) != cmpEqual {
		ctx.err("%T %v %v", v, o, p)
	} else if p.cmp(ctx, o) != cmpEqual {
		ctx.err("%T %v %v", v, o, p)
	} else if cmp(ctx, p, o) != cmpEqual {
		ctx.err("%T %v %v", v, o, p)
	} else if cmp(ctx, o, p) != cmpEqual {
		ctx.err("%T %v %v", v, o, p)
	}
	if v := ctx.get("val9"); v == nil {
		ctx.err("val9")
	} else if v.String() != "foo.txt" {
		ctx.err("%T %v", v, v)
	} else if v.string(ctx) != "foo.txt" {
		ctx.err("%T %v", v, v)
	} else if f, y := v.(*File); !y || f == nil {
		ctx.err("%T %v", v, v)
	} else if f.fullname() != fullFooTxt {
		ctx.err("%T %v %v", v, v, f.fullname())
	}
	if v := ctx.get("val10"); v == nil {
		ctx.err("val10")
	} else if v.String() != "foo.txt" {
		ctx.err("%T %v", v, v)
	} else if v.string(ctx) != fullFooTxt { o, y := v.(fullnameOpt)
		ctx.err("%T %v ; %T %v %v", v, v, o.Value, o.Value, y)
	} else if o, y := v.(fullnameOpt); !y {
		ctx.err("%T %v", v, v)
	} else if f, y := o.Value.(*File); !y || f == nil {
		ctx.err("%T %v", o.Value, o.Value)
	} else if f.fullname() != fullFooTxt {
		ctx.err("%T %v %v", v, v, f.fullname())
	}

	if v := ctx.get("val11.0"); v == nil {
		ctx.err("val11.0")
	} else if v.String() != "-no -yes -false -true" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-no -yes -false -true" {
		ctx.err("%T %v ⇒ %s", v, v, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%T %v", v, v)
	} else { for _, t := range l.Elems {
		if f, y := t.(flag); !y {
			ctx.err("%T %v", t, t)
		} else if _, y := f.Value.(*bareword); !y {
			ctx.err("%T %v", f.Value, f.Value)
		} else if !f.true(ctx) {
			ctx.err("%T %v", t, t)
		} else if !f.Value.true(ctx) {
			ctx.err("%T %v", f.Value, f.Value)
		}
	}}
	if v := ctx.get("val11"); v == nil {
		ctx.err("val11")
	} else if v.String() != "-no" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-no" {
		ctx.err("%T %v ⇒ %s", v, v, s)
	} else if f, y := v.(flag); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := f.Value.(*bareword); !y {
		ctx.err("%T %v", f.Value, f.Value)
	}
	if v := ctx.get("val12"); v == nil {
		ctx.err("val12")
	} else if v.String() != "-yes" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-yes" {
		ctx.err("%T %v ⇒ %s", v, v, s)
	} else if f, y := v.(flag); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := f.Value.(*bareword); !y {
		ctx.err("%T %v", f.Value, f.Value)
	}
	if v := ctx.get("val13"); v == nil {
		ctx.err("val13")
	} else if v.String() != "-false" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "-false" {
		ctx.err("%T %v ⇒ %s", v, v, s)
	} else if f, y := v.(flag); !y {
		ctx.err("%T %v", v, v)
	} else if _, y := f.Value.(*bareword); !y {
		ctx.err("%T %v", f.Value, f.Value)
	}
}
