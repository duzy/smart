//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"sync"
	"strings"
	"path/filepath"
)

func test__wildcard(ctx *testcase) {
	var m = _project(ctx)
	if len(m.filemap.globs) != 2 {
		ctx.err("%v", &m.filemap)
	} else if x, y := m.filemap.globs["**.h"]; !y {
		ctx.err("%v", &m.filemap)
	} else if len(x.a) != 1 {
		ctx.err("%v", x)
	} else if x.o != nil {
		ctx.err("%v", x)
	} else if x.puncs != nil {
		ctx.err("%v", x)
	} else if x.words != nil {
		ctx.err("%v", x)
	} else if x.globs != nil {
		ctx.err("%v", x)
	} else if x.percs != nil {
		ctx.err("%v", x)
	} else if x.reges != nil {
		ctx.err("%v", x)
	} else if x.value != nil {
		ctx.err("%v", x)
	} else if x, y := m.filemap.globs["**.def.am"]; !y {
		ctx.err("%v", &m.filemap)
	} else if len(x.a) != 1 {
		ctx.err("%v", x)
	} else if x.o != nil {
		ctx.err("%v", x)
	} else if x.puncs != nil {
		ctx.err("%v", x)
	} else if x.words != nil {
		ctx.err("%v", x)
	} else if x.globs != nil {
		ctx.err("%v", x)
	} else if x.percs != nil {
		ctx.err("%v", x)
	} else if x.reges != nil {
		ctx.err("%v", x)
	} else if x.value != nil {
		ctx.err("%v", x)
	}

	var (
		pat1 = ctx.val("pat1")
		pat2 = ctx.val("pat2")
		pat3 = ctx.val("pat3")
		pat4 = ctx.val("pat4")
		pat5 = ctx.val("pat5")
		pat6 = ctx.val("pat6")
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

	if g, y := pat1.(*globpat); !y || g == nil {
		ctx.err("%v %v", pat1, tst{pat1})
	} else if s := __string(ctx,pat1); s != "*.h" {
		ctx.err("%v %v %s", pat1, tst{pat1}, s)
	} else if cs := m.unmap_files(ctx, pat1, nil); len(cs) != 1 {
		ctx.err("%v %v %v %v", pat1, tst{pat1}, cs, &m.filemap)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v %v %v", g, tst{cs[0].pattern}, &m.filemap)
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "**.h" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if __string(ctx, g) != "**.h" {
		ctx.err("%v → %v", tst{pat1}, tst{cs[0].pattern})
	}
	if g, y := pat2.(*globpat); !y || g == nil {
		ctx.err("%v %v", pat2, tst{pat2})
	} else if s := __string(ctx,pat2); s != "**.h" {
		ctx.err("%v %v %s", pat2, tst{pat2}, s)
	} else if cs := m.unmap_files(ctx, pat2, nil); len(cs) != 1 {
		ctx.err("%v %v %v", pat2, tst{pat2}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v", tst{cs[0].pattern})
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "**.h" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if __string(ctx,g) != "**.h" {
		ctx.err("%v → %v", tst{pat2}, tst{cs[0].pattern})
	}
	if p, y := pat3.(*path); !y || p == nil {
		ctx.err("%v %v", pat3, tst{pat3})
	} else if s := __string(ctx,pat3); s != "foobar/config/*.def.am" {
		ctx.err("%v %v %s", pat3, tst{pat3}, s)
	} else if false {
		if t := m.unmap_files(ctx, pat3, nil); t != nil {
			ctx.err("%v %v %v", pat3, tst{pat3}, t)
		}
	} else if cs := m.unmap_files(ctx, pat3, nil); len(cs) != 1 {
		ctx.err("%v %v %v", pat3, tst{pat3}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v %v", cs[0].pattern, tst{cs[0].pattern})
	} else if __string(ctx,g) != "**.def.am" {
		ctx.err("%v %v → %v", pat3, tst{pat3}, __string(ctx,g))
	} else if g.String() != "**.def.am" {
		ctx.err("%v %v → %v", pat3, tst{pat3}, g)
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v %v", cs[0].pattern, tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "**.def.am" {
		ctx.err("%v %v → %v", m.pattern, tst{m.pattern}, tst{cs[0].filemap})
	} else if m.pattern.String() != "**.def.am" {
		ctx.err("%v %v → %v", m.pattern, tst{m.pattern}, tst{cs[0].filemap})
	}
	if p, y := pat4.(*path); !y || p == nil {
		ctx.err("%v %v", pat4, tst{pat4})
	} else if s := __string(ctx,pat4); s != "foobar/config/*.def.in" {
		ctx.err("v %v %s", pat4, tst{pat4}, s)
	} else if cs := m.unmap_files(ctx, pat4, nil); len(cs) != 0 {
		// NOTE: because the files spec only defined "**.def.am", no "**.def.in"
		ctx.err("%v %v %v", pat4, tst{pat4}, cs)
	}

	if g, y := pat5.(*globpat); !y || g == nil {
		ctx.err("%v", tst{pat5})
	} else if s := __string(ctx,pat5); s != "*.def.am" {
		ctx.err("%v %s", tst{pat5}, s)
	} else if cs := m.unmap_files(ctx, pat5, nil); len(cs) != 1 {
		ctx.err("%v %v : %v", pat5, tst{pat5}, cs)
	} else if t := cs[0].filemap; t.pattern == nil {
		ctx.err("%v", tst{t})
	} else if _, y := t.pattern.(*globpat); !y {
		ctx.err("%v", tst{t.pattern})
	} else if __string(ctx,t.pattern) != "**.def.am" {
		ctx.err("%v → %v", tst{pat5}, t.pattern)
	} else if y, r, s := match(ctx, pat5, pat3); y {
		ctx.err("%v %v ; %v %v", tst{pat5}, pat3, r, s)
	} else if y, r, s := match(ctx, pat5, pat4); y {
		ctx.err("%v %v ; %v %v", tst{pat5}, pat4, r, s)
	}
	if g, y := pat6.(*globpat); !y || g == nil {
		ctx.err("%v", tst{pat6})
	} else if s := __string(ctx,pat6); s != "**.def.am" {
		ctx.err("%v %s", tst{pat6}, s)
	} else if cs := m.unmap_files(ctx, pat6, nil); len(cs) != 1 {
		ctx.err("%v %v : %v", pat6, tst{pat6}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v", tst{cs[0].pattern})
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "**.def.am" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if y, r, s := match(ctx, pat6, pat3); !y {
		ctx.err("%v, %v ; %v %v", tst{pat6}, pat3, r, s)
	} else if r == nil {
		ctx.err("%v, %v ; %v", tst{pat6}, pat3, tst{r})
	} else if a, y := r.(string); !y {
		ctx.err("%v, %v ; %v", tst{pat6}, pat3, tst{r})
	} else if a != "foobar/config/*.def.am" {
		ctx.err("%v, %v", tst{pat6}, pat3)
	} else if s == nil || len(s) != 1 {
		ctx.err("%v, %v ; %v", tst{pat6}, pat3, s)
	} else if s[0] != "foobar/config/*" {
		ctx.err("%v, %v ; %v", tst{pat6}, pat3, s)
	} else if y, r, s := match(ctx, pat6, pat4); y {
		ctx.err("%v, %v ; %v %v", tst{pat6}, pat4, r, s)
	}

	if s := _workdir(ctx); s == "" {
		ctx.err("workdir")
	} else if !filepath.IsAbs(s) {
		ctx.err("workdir: %v", s)
	}
	if v := ctx.val("top"); v == nil {
		ctx.err("top")
	} else if v.String() != _workdir(ctx) {
		ctx.err("%v", v)
	}
	if v := ctx.val("inc"); v == nil {
		ctx.err("inc")
	} else if v.String() != _workdir(ctx)+"/inc" {
		ctx.err("%v", v)
	}

	const N = 1
	var wg sync.WaitGroup
	var workdirInc = _workdir(ctx) + "/inc"
	invalid := func(name string) bool { return name == "" ||
		name != "foobar/config/a.def.in" &&
		name != "foobar/config/b.def.in" ;}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._directory(workdirInc, pat3); len(a) != 1 {
					ctx.err("_wildcard(%v) (%d): %v", pat3, n, a)
				} else if ident(ctx,a[0]) != "foobar/config/a.def.am" {
					ctx.err("_wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._directory(workdirInc, pat4); len(a) != 2 {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a)
				} else if invalid(ident(ctx,a[0])) {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[0])
				} else if invalid(ident(ctx,a[1])) {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[1])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{dir:workdirInc}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat3); len(a) != 1 {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
				} else if ident(ctx,a[0]) != "foobar/config/a.def.am" {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{dir:workdirInc}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat4); len(a) != 2 {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a)
				} else if invalid(ident(ctx,a[0])) {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a[0])
				} else if invalid(ident(ctx,a[1])) {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a[1])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat3); len(a) != 1 {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
				} else if ident(ctx,a[0]) != "foobar/config/a.def.am" {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat4); a != nil {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a)
				}
			} (i)
		}
	}
	wg.Wait()

	var (
		val1 = ctx.val("val1")
		val2 = ctx.val("val2")
		val3 = ctx.val("val3")
		val4 = ctx.val("val4")
		val5 = ctx.val("val5")
	)
	if s := __string(ctx,val1); s == "" {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/bar.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/v1.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/v2.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/bar/zz/x.h") > 1 {
		ctx.err("%v %v", val1, tst{val1})
	}

	if s := __string(ctx,val2); s == "" {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/bar.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/v1.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/v2.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/bar/zz/x.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	}

	if s := __string(ctx,val3); s == "" {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/v1.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/v2.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	}

	if s := __string(ctx,val4); s == "" {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/v1.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/v2.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	}

	if s := __string(ctx,val5); s == "" {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/v1.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/v2.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/bar/v1.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/bar/v2.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/bar/zz/x.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	}

	var (
		fix1 = ctx.val("fix1")
		fix2 = ctx.val("fix2")
		fix3 = ctx.val("fix3")
		fix4 = ctx.val("fix4")
	)
	if s := __string(ctx,fix1); s == "" {
		ctx.err("fix1: %v", fix1)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix1: %v", fix1)
	}
	if s := __string(ctx,fix2); s != "" {
		// NOTE: because the files spec defines only "**.def.am", no "**.def.in"
		ctx.err("fix2: %v", fix2)
	}
	if s := __string(ctx,fix3); s == "" {
		ctx.err("fix3: %v", fix3)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix3: %v", fix3)
	}
	if s := __string(ctx,fix4); s == "" {
		ctx.err("fix4: %v", fix4)
	} else if strings.Count(s, "foobar/config/a.def.in") != 1 {
		ctx.err("fix4: %v", fix4)
	} else if strings.Count(s, "foobar/config/b.def.in") != 1 {
		ctx.err("fix4: %v", fix4)
	}
}
