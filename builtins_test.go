//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"sync"
	"strings"
	"path/filepath"
)

type testAssertStruct struct {
	bools []bool
	vals []Value
}
func testAssertHook(ctx Context, v Value, b bool, i any) {
	s := i.(*testAssertStruct)
	s.bools, s.vals = append(s.bools, b), append(s.vals, v)
}
func testAssert(ctx testcase1) {
	if s := "foo"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if v != d.value {
		ctx.err("%v != %v", v, d.value)
	} else if v.String() != "foo" {
		ctx.err("%v", tst{v})
	} else if v.string(ctx) != "foo" {
		ctx.err("%v", tst{v})
	}

	if s, y := ctx.i.(*testAssertStruct); !y {
		ctx.err("%T", ctx.i)
	} else if len(s.bools) != 12 {
		ctx.err("%v, %v, %v %v", s.vals, s.bools, len(s.vals), len(s.bools))
	} else if len(s.vals) != len(s.bools) {
		ctx.err("%v %v", s.vals, s.bools)
	} else if i := 0; s.vals[i].String() != "{=true}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 1; s.vals[i].String() != "{=false}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 2; s.vals[i].String() != "{=yes}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 3; s.vals[i].String() != "{=no}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 4; s.vals[i].String() != "{=none}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 5; s.vals[i].String() != "{=undef x}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 6; s.vals[i].String() != "{}" || s.bools[i] { // {=null}
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].String() != "{x}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].string(ctx) != "x" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 8; s.vals[i].String() != "foobar{}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if v, y := s.vals[i].(*compound); !y {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], v.elems)
	} else if len(v.elems) != 2 {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], v.elems)
	} else if _, y := v.elems[0].(*word); !y {
		ctx.err("%v", tst{v.elems[0]})
	} else if _, y := v.elems[1].(*null); !y {
		ctx.err("%v", tst{v.elems[1]})
	} else if i = 9;  s.vals[i].String() != "1" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 10;  s.vals[i].String() != "0" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 11; s.vals[i].String() != "$(equal $(foo),foo)" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	}
}

func testLocals(ctx *testcase) {
	s := "foo"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{=word foobar}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{=word foobar}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{=word x}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{=word foobar}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}
}

func testBuiltin_wildcard(ctx *testcase) {
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
	} else if s := pat1.string(ctx); s != "*.h" {
		ctx.err("%v %v %s", pat1, tst{pat1}, s)
	} else if cs := m.unmap_files(ctx, pat1, nil); len(cs) != 1 {
		ctx.err("%v %v %v %v", pat1, tst{pat1}, cs, &m.filemap)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v %v %v", g, tst{cs[0].pattern}, &m.filemap)
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if m.pattern.string(ctx) != "**.h" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if g.string(ctx) != "**.h" {
		ctx.err("%v → %v", tst{pat1}, tst{cs[0].pattern})
	}
	if g, y := pat2.(*globpat); !y || g == nil {
		ctx.err("%v %v", pat2, tst{pat2})
	} else if s := pat2.string(ctx); s != "**.h" {
		ctx.err("%v %v %s", pat2, tst{pat2}, s)
	} else if cs := m.unmap_files(ctx, pat2, nil); len(cs) != 1 {
		ctx.err("%v %v %v", pat2, tst{pat2}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v", tst{cs[0].pattern})
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if m.pattern.string(ctx) != "**.h" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if g.string(ctx) != "**.h" {
		ctx.err("%v → %v", tst{pat2}, tst{cs[0].pattern})
	}
	if p, y := pat3.(*path); !y || p == nil {
		ctx.err("%v %v", pat3, tst{pat3})
	} else if s := pat3.string(ctx); s != "foobar/config/*.def.am" {
		ctx.err("%v %v %s", pat3, tst{pat3}, s)
	} else if false {
		if t := m.unmap_files(ctx, pat3, nil); t != nil {
			ctx.err("%v %v %v", pat3, tst{pat3}, t)
		}
	} else if cs := m.unmap_files(ctx, pat3, nil); len(cs) != 1 {
		ctx.err("%v %v %v", pat3, tst{pat3}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v %v", cs[0].pattern, tst{cs[0].pattern})
	} else if g.string(ctx) != "**.def.am" {
		ctx.err("%v %v → %v", pat3, tst{pat3}, g.string(ctx))
	} else if g.String() != "**.def.am" {
		ctx.err("%v %v → %v", pat3, tst{pat3}, g)
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v %v", cs[0].pattern, tst{cs[0].filemap})
	} else if m.pattern.string(ctx) != "**.def.am" {
		ctx.err("%v %v → %v", m.pattern, tst{m.pattern}, tst{cs[0].filemap})
	} else if m.pattern.String() != "**.def.am" {
		ctx.err("%v %v → %v", m.pattern, tst{m.pattern}, tst{cs[0].filemap})
	}
	if p, y := pat4.(*path); !y || p == nil {
		ctx.err("%v %v", pat4, tst{pat4})
	} else if s := pat4.string(ctx); s != "foobar/config/*.def.in" {
		ctx.err("v %v %s", pat4, tst{pat4}, s)
	} else if cs := m.unmap_files(ctx, pat4, nil); len(cs) != 0 {
		// NOTE: because the files spec only defined "**.def.am", no "**.def.in"
		ctx.err("%v %v %v", pat4, tst{pat4}, cs)
	}

	if g, y := pat5.(*globpat); !y || g == nil {
		ctx.err("%v", tst{pat5})
	} else if s := pat5.string(ctx); s != "*.def.am" {
		ctx.err("%v %s", tst{pat5}, s)
	} else if cs := m.unmap_files(ctx, pat5, nil); len(cs) != 1 {
		ctx.err("%v %v : %v", pat5, tst{pat5}, cs)
	} else if t := cs[0].filemap; t.pattern == nil {
		ctx.err("%v", tst{t})
	} else if _, y := t.pattern.(*globpat); !y {
		ctx.err("%v", tst{t.pattern})
	} else if t.pattern.string(ctx) != "**.def.am" {
		ctx.err("%v → %v", tst{pat5}, t.pattern)
	} else if y, r, s := pat5.match(ctx, pat3); y {
		ctx.err("%v %v ; %v %v", tst{pat5}, pat3, r, s)
	} else if y, r, s := pat5.match(ctx, pat4); y {
		ctx.err("%v %v ; %v %v", tst{pat5}, pat4, r, s)
	}
	if g, y := pat6.(*globpat); !y || g == nil {
		ctx.err("%v", tst{pat6})
	} else if s := pat6.string(ctx); s != "**.def.am" {
		ctx.err("%v %s", tst{pat6}, s)
	} else if cs := m.unmap_files(ctx, pat6, nil); len(cs) != 1 {
		ctx.err("%v %v : %v", pat6, tst{pat6}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v", tst{cs[0].pattern})
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if m.pattern.string(ctx) != "**.def.am" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if y, r, s := pat6.match(ctx, pat3); !y {
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
	} else if y, r, s := pat6.match(ctx, pat4); y {
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
		var c = original{ctx, defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := builtin_wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._directory(workdirInc, pat3); len(a) != 1 {
					ctx.err("_wildcard(%v) (%d): %v", pat3, n, a)
				} else if a[0].ident(ctx) != "foobar/config/a.def.am" {
					ctx.err("_wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		var c = original{ctx, defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := builtin_wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._directory(workdirInc, pat4); len(a) != 2 {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a)
				} else if invalid(a[0].ident(ctx)) {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[0])
				} else if invalid(a[1].ident(ctx)) {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[1])
				}
			} (i)
		}
	}
	{
		var c = original{ctx, defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := builtin_wildcard{dir:workdirInc}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat3); len(a) != 1 {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
				} else if a[0].ident(ctx) != "foobar/config/a.def.am" {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		var c = original{ctx, defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := builtin_wildcard{dir:workdirInc}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat4); len(a) != 2 {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a)
				} else if invalid(a[0].ident(ctx)) {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a[0])
				} else if invalid(a[1].ident(ctx)) {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a[1])
				}
			} (i)
		}
	}
	{
		var c = original{ctx, defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := builtin_wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat3); len(a) != 1 {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
				} else if a[0].ident(ctx) != "foobar/config/a.def.am" {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		var c = original{ctx, defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := builtin_wildcard{}
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
	if s := val1.string(ctx); s == "" {
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

	if s := val2.string(ctx); s == "" {
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

	if s := val3.string(ctx); s == "" {
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

	if s := val4.string(ctx); s == "" {
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

	if s := val5.string(ctx); s == "" {
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
	if s := fix1.string(ctx); s == "" {
		ctx.err("fix1: %v", fix1)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix1: %v", fix1)
	}
	if s := fix2.string(ctx); s != "" {
		// NOTE: because the files spec defines only "**.def.am", no "**.def.in"
		ctx.err("fix2: %v", fix2)
	}
	if s := fix3.string(ctx); s == "" {
		ctx.err("fix3: %v", fix3)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix3: %v", fix3)
	}
	if s := fix4.string(ctx); s == "" {
		ctx.err("fix4: %v", fix4)
	} else if strings.Count(s, "foobar/config/a.def.in") != 1 {
		ctx.err("fix4: %v", fix4)
	} else if strings.Count(s, "foobar/config/b.def.in") != 1 {
		ctx.err("fix4: %v", fix4)
	}
}

func testBuiltin_file0(ctx *testcase) {
	if pat, str := ".test/a/**.c", ".test/a/b/c/foo.c"; false {
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err(str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val1.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if s := val.string(ctx); s != str {
		ctx.err("%v: %s != %s", tst{val}, s, str)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if   s := "val1.2" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if val.string(ctx) != str {
		ctx.err("%v", tst{val})
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	}

	if pat, str := ".test/xx/*.c", ".test/xx/foo.c"; false {
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val2.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if   s := "val2.2" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	}

	if pat, str := ".test/xx/yy/*.c", ".test/xx/yy/foo.c"; false {
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val3" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	}

	if pat, str := ".test/xx/yy/zz/*.c", ".test/xx/yy/zz/foo.c"; false {
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val4" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	}

	if s := "val5" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if _, y := val.(*null); !y {
		ctx.err("%v", tst{val})
	} else if val.String() != "{}" {
		ctx.err("%v", tst{val})
	} else if val.string(ctx) != "" {
		ctx.err("%v", tst{val})
	}

	if pat, str := "**.auto", ".test/a/b/c.auto"; false {
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("%s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if s := "p1" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if x.string(ctx) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, v); t == nil {
		ctx.err("%v %v", v, tst{v})
	} else if len(t) != 1 {
		ctx.err("%v %v %v", v, tst{v}, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%v: %v", tst{v}, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%v: %v", tst{v}, m.pattern)
	}

	if str := ".test/a/b/c.none" ; false {} else
	if t := unmap_files(ctx, str); t != nil {
		ctx.err("%v", str)
	} else if s := "p2" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v", s)
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if x.string(ctx) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, v); t != nil {
		ctx.err("%v %v", v, tst{v})
	}
}

func testBuiltin_foreach(ctx *testcase) {
	{
		var c0 = builtin_foreach{}
		var c1 = partial{&c0, nonePart}
		var c2 = builtin_foreach{}
		c0.evocation = &evocation{automatic{Context:ctx}, nil, nil, nil}
		c2.evocation = &evocation{automatic{Context:c1}, nil, nil, nil}
		if cast[*builtin_](&c0) == nil {
			ctx.err("builtin_")
		}
		if cast[partial](ctx).Context != nil {
			ctx.err("partial")
		}
		if cast[partial](&c0).Context != nil {
			ctx.err("partial")
		}
		if cast[partial](c1).Context == nil {
			ctx.err("partial")
		}
		if cast[partial](&c2).Context != nil {
			ctx.err("partial")
		}
	}

	var test_1_value Value

	if s := ".test.1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", _project(ctx), s)
	} else if test_1_value = d.value; test_1_value == nil {
		ctx.err("%v", d)
	} else if s, t := d.value.String(), "x $1 $2 $3 $4 &(.test.h){$1}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if s, t := d.value.string(ctx), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "x $1 $2 $3 $4 &(.test.h){$1}"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d.name, []string{"a", "b", "c"}); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "x a b c $2 $3 $4 &(.test.h)a? &(.test.h)b? &(.test.h)c?"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x a b c -a -b -c"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
	}

	if s := ".test.2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", _project(ctx), s)
	} else if s0, t := d.value.String(), "x xq xp x{&(.test.h){$1}}"; s0 != t {
		ctx.err("%s != %s : %v", s0, t, tst{d.value})
	} else if s1, t := d.value.string(ctx), "x xq xp"; s1 != t {
		ctx.err("%s != %s : %v", s1, t, tst{d.value})
	} else if x, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if x.len() != 2 {
		ctx.err("%d %v", x.len(), tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if s0 != v.String() {
			ctx.err("%v != %s", tst{v}, s0)
		} else if t := v.string(ctx); s1 != t {
			ctx.err("%v → %v != %s", tst{v}, t, s1)
		}
		if v := ctx.val(d.name, []string{"a", "b", "c"}); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "x xq xp x{&(.test.h)a &(.test.h)b &(.test.h)c}"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x xq xp x-a x-b x-c"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
	}

	if s := ".test.21"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x xq xp x{-{$1}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v", l.elems)
	} else if s, t := l.elems[0].String(), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[0]})
	} else if s, t := l.elems[1].String(), "xq xp x{-{$1}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t, y := l.elems[1].(*list); !y {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t.len() != 3 {
		ctx.err("%d, %v", t.len(), tst{t})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if !equal(ctx, v, d.value) {
			ctx.err("%v → %v (%v)", tst{v}, d, v.cmp(ctx, d.value))
		} else if s, t := v.String(), "x xq xp x{-{$1}}"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x xq xp"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d.name, []string{"a", "b", "c"}); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "x xq xp x-a x-b x-c"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x xq xp x-a x-b x-c"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d.name, []string{"x", "y", "z"}); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "x xq xp x-x x-y x-z"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x xq xp x-x x-y x-z"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
	}

	if s := ".test.22"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x x- x{-{$1}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "x x- x{-{$1}}"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x x-"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d.name, "a"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "x x- x-a"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x x- x-a"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
	}

	if s := ".test.23"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "xq xp x{&(.test.xx){$1}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d.name, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xq xp x{&(.test.xx)a &(.test.xx)b &(.test.xx)c}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if s := ".test.3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "x &(.test.{$1})$1{}zz y $(closure .test.{$1})$1{}99 z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s != v.String() {
			ctx.err("%s != %s : %v", v, s, tst{v})
		} else if s, t := v.string(ctx), "x y z"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d.name, []string{"foo", "bar"}); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "x {&(.test.foo) &(.test.bar)}foo bar{}zz y &(.test.foo) &(.test.bar)foo bar{}99 z"; s != t {
			for i, v := range merge(v) { note(pc(ctx,v), "%d. %32v : %v", i, v, ts(v)) }
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "x barzz y foo bar99 z"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if x, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if x.len() != 5 {
			for i, v := range x.elems { note(pc(ctx,v), "%d. %v: %v", i, typeof(v), v) }
			ctx.err("%v, %v", x.len(), v)
		} else if v, s := x.elems[0], "x"; v.String() != s {
			ctx.err("%s != %s : %s", v, s, ts(v))
		} else if v, s := x.elems[1], "{&(.test.foo) &(.test.bar)}foo bar{}zz"; v.String() != s {
			ctx.err("%s != %s : %s", v, s, ts(v))
		} else if v, s := x.elems[2], "y"; v.String() != s {
			ctx.err("%s != %s : %s", v, s, ts(v))
		} else if v, s := x.elems[3], "&(.test.foo) &(.test.bar)foo bar{}99"; v.String() != s {
			ctx.err("%s != %s : %s", v, s, ts(v))
		} else if v, s := x.elems[4], "z"; v.String() != s {
			ctx.err("%s != %s : %s", v, s, ts(v))
		} else if v, s := x.elems[1], "list"; typeof(v) != s {
			ctx.err("%s != %s : %s", v, s, ts(v))
		} else if v, s := x.elems[3], "list"; typeof(v) != s {
			ctx.err("%s != %s : %s", v, s, ts(v))
		}
	}

	if s := ".test.4"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "&(.test.{$1}.$(or $4,$3)) &(.test.{$2}.$(or $4,$3))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if s != v.String() {
			ctx.err("%s != %s : %v", v, s, tst{v})
		} else if s, t := v.string(ctx), ""; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d, "x", "y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.$(or $4,$3))? &(.test.y.$(or $4,$3))?"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), ""; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if v2 := ctx.val(v, "", "", "a", ""); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v2.String(), "&(.test.x.a)? &(.test.y.a)?"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v2})
		} else if v3 := ctx.val(v, "", "", "", "a"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v3.String(), "&(.test.x.a)? &(.test.y.a)?"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v3})
		}
		if v := ctx.val(d, "x", "y", "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.b)? &(.test.y.b)?"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "xb yb"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d, "x", "y", "a", []string{}); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.a)? &(.test.y.a)?"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "xa ya"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
		if v := ctx.val(d, "x", "y", []string{}, "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.b)? &(.test.y.b)?"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		} else if s, t := v.string(ctx), "xb yb"; s != t {
			ctx.err("%s != %s : %v", s, t, tst{v})
		}
	}

	if s := ".test.5"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if x, y := v.(cond); !y {
		ctx.err("%v : %v", v, tst{v})
	} else if _, y := x.Value.(*closure); !y {
		ctx.err("%v : %v", x.Value, tst{x.Value})
	} else if s, t := v.String(), "&(.test.x do.smart)?"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x do.smart)?"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, "xxx"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x do.smart)?"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, test_def_2{}); v == nil {
		ctx.err("%v", tst{d})
	} else if f, y := v.(*file); !y {
		ctx.err("%v : %v", v, tst{v})
	} else if f.name != "do.smart" {
		ctx.err("%v : %v", v, tst{v})
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if s := ".test.6"; false {
	} else if d := ctx.def(s); d == nil || d.value == nil {
		ctx.err("%s: %v", s, d)
	} else if v := ctx.val(d, "x", "y", "z"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x y z $9 - x y z $9"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x y z - x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if s := ".test.7"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "xa x{&(.test.z){$1}zz} xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 3 {
		ctx.err("%v", t)
	} else if v := ctx.val(d, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 3 {
		ctx.err("%v, %v", l.len(), v)
	} else if s, t := v.String(), "xa x{&(.test.z)y1zz &(.test.z)y2zz &(.test.z)y3zz} xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if s := ".test.x"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d, "do.smart"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%v != %s", tst{v}, s) // → %s
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if f, y := v.(*file); !y || f == nil {
		ctx.err("%v", tst{v})
	}
}

func testBuiltin_foreach1(ctx *testcase) {
	if s := ".test.1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x $1,B b)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s != v.String() {
		ctx.err("%v != %s", tst{v}, s)
	} else if s, t := v.string(ctx), "foo test-foo test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := `&(.test.s) $(value .test~&(.test.s)) test-a $(closure .test~&(.test.s).a)? test-B $(closure .test~&(.test.s).B)? test-b $(closure .test~&(.test.s).b)?`, v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foo test-foo test-a test-foo-a test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "xxx"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := `&(.test.s) $(value .test~&(.test.s)) test-a $(closure .test~&(.test.s).a)? test-B $(closure .test~&(.test.s).B)? test-b $(closure .test~&(.test.s).b)?`, v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foo test-foo test-a test-foo-a test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_foreach2(ctx *testcase) {
	if s := ".test.foreach.a"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $(foreach $2,a$_),b$_)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.foreach.b"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s) // "$(foreach x$1 y$1 z$1 $(foreach xx$2 yy$2,a$_),b$_)"
	}

	if s := ".test.foreach.c"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) $(foreach &(.test.foreach.x) $(foreach $1 $2,$(closure .test.foreach.x.$_)),-x$_)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); "" != t {
		ctx.err("%v → %s != ''", tst{v}, t)
	}

	if s := ".test.2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2) $(foreach &(.test.foreach.x) $(foreach $1 $2,$(closure .test.foreach.x.$_)),-x$_)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if t := v.String(); s != t {
		ctx.err("%v != %s", tst{v}, s)
	} else if s, t := v.string(ctx), "-xvw"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(.test.foreach.c $1,4)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "baxx4 bayy4 -xvw"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if x := v.expand(_final(ctx)); x == nil {
		ctx.err("%v", tst{v})
	} else if l, y := x.(*list); !y {
		ctx.err("%v", tst{x})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{x}, l.len())
	} else if l0, y := l.elems[0].(*list); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if l0.len() != 5 {
		ctx.err("%v ; %d", tst{l.elems[0]}, l0.len())
	} else if l1, y := l.elems[1].(*list); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if l1.len() != 3 {
		ctx.err("%v ; %d", tst{l.elems[1]}, l1.len())
	} else if s, t := l0.String(), "bx$1? by$1? bz$1? baxx4 bayy4"; s != t {
		for i, v := range l0.elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v → %s != %s", tst{l0}, t, s)
	} else if s, t := l1.String(), "-xvw -x{$(closure .test.foreach.x.{$1})}? -xW$1$2?"; s != t {
		for i, v := range l1.elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v → %s != %s", tst{l1}, t, s)
	} else if s, t := x.String(), "bx$1? by$1? bz$1? baxx4 bayy4 -xvw -x{$(closure .test.foreach.x.{$1})}? -xW$1$2?"; s != t {
		for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if v := ctx.val(d, "3"); v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "bx3 by3 bz3 baxx4 bayy4 -x{&(.test.foreach.x)}? -xV$1$2? -xW$1$2?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "bx3 by3 bz3 baxx4 bayy4 -xvw"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.4"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $2,$(.test.foreach.d.$_))"; s != t {
		ctx.err("%v != %s", tst{v}, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if t := v.String(); s != t {
		ctx.err("%v != %s", tst{v}, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "1", "2"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{v}, l.len())
	} else if t, y := l.elems[0].(cond); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := t.Value.(*delegate); !y {
		ctx.err("%v", tst{t.Value})
	} else if t, y := l.elems[1].(cond); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if _, y := t.Value.(*delegate); !y {
		ctx.err("%v", tst{t.Value})
	} else if s, t := v.String(), "$(foreach $1 $2,-x$_)? $(foreach $1 $2,-y$_)?"; s != t {
		ctx.err("%v != %s", tst{v}, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v2 := ctx.val(v, "1", "2"); v2 == nil {
		ctx.err("%v", tst{v})
	} else if l, y := v2.(*list); !y {
		ctx.err("%v", tst{v2})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{v2}, l.len())
	} else if l.elems = merge(l.elems...); l.len() != 4 {
		ctx.err("%v ; %d", tst{v2}, l.len())
	} else if _, y := l.elems[0].(flag); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(flag); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if _, y := l.elems[2].(flag); !y {
		ctx.err("%v", tst{l.elems[2]})
	} else if _, y := l.elems[3].(flag); !y {
		ctx.err("%v", tst{l.elems[3]})
	} else if s, t := v2.String(), "-x1 -x2 -y1 -y2"; s != t {
		ctx.err("%v != %s", tst{v2}, s)
	} else if t := v2.string(ctx); t != s {
		ctx.err("%v → %s", tst{v2}, t)
	}
}

func testBuiltin_foreach3(ctx *testcase) {
	if s := ".test.a"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,$_$3)"; s != t {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if v.String() != s {
			ctx.err("%v", tst{v})
		} else if v.string(ctx) != "" {
			ctx.err("%v → %s", tst{v}, v.string(ctx))
		}
		if v := ctx.val(d, "a"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "a$3? {$2}$3?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		} else if v2 := ctx.val(v, "", []string{"1", "2", "3"}, "x"); v == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v2.String(), "ax 1x 2x 3x"; s != t {
			ctx.err("%v → %s != %s", tst{v2}, t, s)
		} else if t := v2.string(ctx); t != s {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "a$3? b$3?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		} else if v2 := ctx.val(v, "", "", "x"); v == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v2.String(), "ax bx"; s != t {
			ctx.err("%v → %s != %s", tst{v2}, t, s)
		} else if t := v2.string(ctx); t != s {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "a", "b", "x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "ax bx"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != s {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
	}

	if s := ".test.x"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if x, y := d.value.(*delegate); !y {
		ctx.err("%v", tst{d.value})
	} else if len(x.a) != 2 {
		ctx.err("%v ; %d args", tst{d.value}, len(x.a))
	} else if s, t := x.a[0].String(), "$1 $2"; s != t {
		ctx.err("%v → %s != %s", tst{x.a[0]}, t, s)
	} else if l, y := x.a[1].(*list); !y || l.len() != 1 {
		ctx.err("%v", tst{x.a[1]})
	} else if l0, y := l.elems[0].(cond); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if l0p, y := l0.Value.(*pair); !y {
		ctx.err("%v", tst{l0.Value})
	} else if l0k, y := l0p.key.(*word); !y || l0k.String() != "std" {
		ctx.err("%v ; %v", tst{l0p.key}, l0k)
	} else if l0v, y := l0p.val.(disjunction); !y || l0v.String() != "{&(.test.$_)}" {
		ctx.err("%v ; %v", tst{l0p.val}, l0v)
	} else if l0w, y := l0v.Value.(*closure); !y || l0w.String() != "&(.test.$_)" {
		ctx.err("%v ; %v", tst{l0v.Value}, l0w)
	} else if s, t := l0w.x.String(), ".test.$_"; s != t {
		ctx.err("%v → %s != %s", tst{l0w.x}, t, s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,std={&(.test.$_)}?)"; s != t {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if t := v.String(); s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "a"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.a)? std=&(.test.{$2})?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		} else if v2 := ctx.val(v, nil, []string{"if.x", "if.y", "if.z"}); v2 == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v2.String(), "std=&(.test.a)? std=&(.test.if.x)? std=&(.test.if.y)? std=&(.test.if.z)?"; s != t {
			ctx.err("%v → %s != %s", tst{v2}, t, s)
		} else if s, t := v2.string(ctx), "std=xxx std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v2}, t, s)
		} else if v3 := ctx.val(v, nil, []string{"if.x", "if.y", "if.z"}, test_def_2{}); v3 == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v3.String(), "std=$(foreach $1 $2,$_$3)? std=xxx std=yyy std=&(.test.if.z)?"; s != t {
			ctx.err("%v → %s != %s", tst{v3}, t, s)
		} else if s, t := v3.string(ctx), "std=xxx std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v3}, t, s)
		}
		if v := ctx.val(d, "a", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if t, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := t.Value.(*pair); !y {
			ctx.err("%v", tst{t.Value})
		} else if s, t := v.String(), "std=&(.test.a)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, nil, "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if t, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := t.Value.(*pair); !y {
			ctx.err("%v", tst{t.Value})
		} else if s, t := v.String(), "std=&(.test.b)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if l, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if l.len() != 2 {
			ctx.err("%v ; %d", tst{v}, l.len())
		} else if x, y := l.elems[0].(cond); !y {
			ctx.err("%v", tst{l.elems[0]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := v.String(), "std=&(.test.a)? std=&(.test.b)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
	}

	if s := ".test.y"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if _, y := d.value.(*delegate); !y {
		ctx.err("%v", tst{d.value})
	} else if s := "$(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_)))"; s != d.value.String() {
		ctx.err("%v", tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if t := v.String(); s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if l, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if l.len() != 2 {
			ctx.err("%v ; %d", tst{v}, l.len())
		} else if x, y := l.elems[0].(cond); !y {
			ctx.err("%v", tst{l.elems[0]})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))? $(if &(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := v.String(), "$(if &(.test.if.y),std=&(.test.if.y))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if l, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if len(l.elems) != 2 {
			ctx.err("%v", l.elems)
		} else if x, y := l.elems[0].(cond); !y {
			ctx.err("%v", tst{l.elems[0]})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s := l.elems[0].string(ctx); s != "std=xxx" {
			ctx.err("%v → %s", l.elems[0], s)
		} else if s := l.elems[1].string(ctx); s != "std=yyy" {
			ctx.err("%v → %s", l.elems[1], s)
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))? $(if &(.test.if.y),std=&(.test.if.y))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "zzz", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := v.String(), "$(if &(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := v.String(), "$(if &(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
	}

	if s := ".test.z"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if _, y := d.value.(*delegate); !y {
		ctx.err("%v", tst{d.value})
	} else if s := "$(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_)))"; s != d.value.String() {
		ctx.err("%v", tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if t := v.String(); s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if l, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if l.len() != 2 {
			ctx.err("%v ; %d", tst{v}, l.len())
		} else if x, y := l.elems[0].(cond); !y {
			ctx.err("%v", tst{l.elems[0]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.String(), "std=&(.test.if.x)?"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.String(), "$(if $(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		} else if s, t := v.String(), "std=&(.test.if.x)? $(if $(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.Value.String(), "std=&(.test.if.x)"; s != t {
			ctx.err("%v → %s != %s", tst{x.Value}, t, s)
		} else if s, t := v.String(), "std=&(.test.if.x)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.Value.String(), "std=&(.test.if.y)"; s != t {
			ctx.err("%v → %s != %s", tst{x.Value}, t, s)
		} else if s, t := v.String(), "std=&(.test.if.y)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if l, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if len(l.elems) != 2 {
			ctx.err("%v", l.elems)
		} else if x, y := l.elems[0].(cond); !y {
			ctx.err("%v", tst{l.elems[0]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := l.elems[0].String(), "std=&(.test.if.x)?"; s != t {
			ctx.err("%v → %s != %s", l.elems[0], t, s)
		} else if s, t := l.elems[0].string(ctx), "std=xxx"; s != t {
			ctx.err("%v → %s != %s", l.elems[0], t, s)
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := l.elems[1].String(), "std=&(.test.if.y)?"; s != t {
			ctx.err("%v → %s != %s", l.elems[1], t, s)
		} else if s, t := l.elems[1].string(ctx), "std=yyy"; s != t {
			ctx.err("%v → %s != %s", l.elems[1], t, s)
		} else if s, t := v.String(), "std=&(.test.if.x)? std=&(.test.if.y)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "zzz", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if _, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "$(if $(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d})
		} else if _, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "$(if $(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
	}
}

func testBuiltin_foreach4(ctx *testcase) {
	if s := ".test.1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if len(l.elems) != 2 {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[0].(*compound); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(*compound); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := v.String(), "Xxa Xxb"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "Xxa Xxb"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if x, y := d.value.(*delegate); !y {
		ctx.err("%v", tst{d.value})
	} else if t, y := x.x.(*builtin); !y {
		ctx.err("%v", tst{x.x})
	} else if t.name != "foreach" {
		ctx.err("%v", tst{x.x})
	} else if s, t := d.value.String(), "$(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if len(l.elems) != 3 {
		ctx.err("%v", tst{l.elems[0]})
	} else if x, y := l.elems[0].(cond); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if z, y := x.Value.(*compound); !y {
		ctx.err("%v", tst{x.Value})
	} else if s, t := z.String(), "X{&(.test.xa)}"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if s, t := "X{~1~ $1 $2 $(foreach $1 $2,x-$_) $(foreach $(foreach $1 $2,z-$_),~1~$_)}", z.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if s, t := z.string(ctx), "X~1~"; s != t {
		ctx.err("%v : %v → %s != %s", tst{z}, z, t, s)
	} else if t := x.string(ctx); t != "" {
		ctx.err("%v → %s", tst{x}, t)
	} else if x := ctx.val(z, test_def_2{}); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X{~1~ $1 $2 $(foreach $1 $2,x-$_) $(foreach $(foreach $1 $2,z-$_),~1~$_)}"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X~1~"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if x, y := l.elems[1].(cond); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if z, y := x.Value.(*compound); !y {
		ctx.err("%v", tst{x.Value})
	} else if s, t := z.String(), "X{&(.test.xb)}"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if s, t := z.string(ctx), "X~2~"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if t := x.string(ctx); t != "" {
		ctx.err("%v → %s", tst{x}, t)
	} else if x := ctx.val(z, test_def_2{}); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X{~2~ $1 $2 $(foreach $1 $2,x-$_) $(foreach $(foreach $1 $2,z-$_),~2~$_)}"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X~2~"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if x, y := l.elems[2].(cond); !y {
		ctx.err("%v", tst{l.elems[2]})
	} else if z, y := x.Value.(*compound); !y {
		ctx.err("%v", tst{x.Value})
	} else if s, t := z.String(), "X{&(.test.xc)}"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if t := z.string(ctx); t != "X" {
		ctx.err("%v → %s", tst{z}, t)
	} else if t := x.string(ctx); t != "" {
		ctx.err("%v → %s", tst{x}, t)
	} else if x := ctx.val(z, "xx", "yy", test_def_2{}); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X{$(foreach $(foreach $(foreach $(foreach $1 $2,~$_),~$_),~$_),~$_)}"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if x = ctx.val(x, "xx", "yy", test_def_2{}); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X~~~~xx X~~~~yy"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X~~~~xx X~~~~yy"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := v.String(), "X{&(.test.xa)}? X{&(.test.xb)}? X{&(.test.xc)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d.value})
	} else if s, t := v.String(), "X{&(.test.xa)}? X{&(.test.xb)}? X{&(.test.xc)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $(foreach $(foreach $1 $2,&(.test.x$_)),X$_),Y$_)"; s != t {
		ctx.err("%v : %v → %s != %s", d.value, tst{d.value}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{v}, l.len())
	} else if s, t := v.String(), "YX{&(.test.xa)}? YX{&(.test.xb)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	}

	if s := ".test.x"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $(foreach $1 $2,x$_),X$_) $(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{v}, l.len())
	} else if s, t := v.String(), "Xxa Xxb X{&(.test.xa)}? X{&(.test.xb)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := `Xxa Xxb`, v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_foreach5(ctx *testcase) {
	if s := ".test.o"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "o"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.o.a"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.o.b"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.a"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a~ $(foreach $(foreach $1 $2,$(.test.x.o.$_)),-ao$_) ~a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s := "a~ -aox.o.a -aox.o.b ~a"; s != v.String() {
		ctx.err("%v != %s", v, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.b"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_) ~b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ -bo{&(.test.x.o.a)}? -bo{&(.test.x.o.b)}? ~b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.0"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(foreach $1 $2,&(.test.x.$_) &(.test.x.&(.test.o).$_))"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		} else if v := ctx.val(d, "a", "b", "c"); v == nil {
			ctx.err("%v", tst{d})
		} else if t := v.String(); s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		} else if x := ctx.val(v, "a", "b", "c"); x == nil {
			ctx.err("%v", tst{v})
		} else if s, t := x.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)?"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		} else if s, t := x.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		} else if x := ctx.val(v, "a", "b", "c", test_def_2{}); x == nil {
			ctx.err("%v", tst{v})
		} else if s, t := x.String(), "a~ $(foreach $(foreach $1 $2,$(.test.x.o.$_)),-ao$_)? ~a x.o.a b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_)? ~b x.o.b"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		} else if s, t := x.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		}
	}

	if s := ".test.x.1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x $1)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", tst{d})
		} else if t := v.String(); s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.{$2})? &(.test.x.&(.test.o).{$2})?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "a~ ~a x.o.a"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if x := ctx.val(v, nil, []string{"b", "c"}); x == nil {
			ctx.err("%v", tst{v})
		} else if s, t := x.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? {&(.test.x.b)}? {&(.test.x.c)}? {&(.test.x.&(.test.o).b)}? {&(.test.x.&(.test.o).c)}?"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		} else if s, t := x.string(ctx), "a~ ~a x.o.a b~ ~b ~c~ x.o.b x.o.c"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		}
	}

	if s := ".test.x.2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $2,&(.test.x.$_) &(.test.x.&(.test.o).$_))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.4"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.{$2})? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.{$2})? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.5"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", []string{"b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.6"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if x := v.expand(_final(ctx)); x == nil {
		ctx.err("%v", tst{v})
	} else if l, y := x.(*list); !y {
		ctx.err("%v", tst{x})
	} else if elems := merge(l.elems...); l.len() != 4 || len(elems) != 7 {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v ; %d, %d", tst{x}, l.len(), len(elems))
	} else if _, y := elems[2].(cond); !y {
		ctx.err("%v", tst{elems[2]})
	} else if s, t := x.String(), "a~ -aox.o.a -ao{$(.test.x.o.{$2})}? ~a x.o.a &(.test.x.{$2} a,$2)? &(.test.x.o.{$2})?"; s != t {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "a~ -aox.o.a ~a x.o.a"; s != t {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := v.String(), "&(.test.x.a a,$2)? &(.test.x.&(.test.o).a)? &(.test.x.{$2} a,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a b c,$2)? &(.test.x.&(.test.o).a)? &(.test.x.b a b c,$2)? &(.test.x.&(.test.o).b)? &(.test.x.c a b c,$2)? &(.test.x.&(.test.o).c)? &(.test.x.{$2} a b c,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b -aox.o.c ~a x.o.a b~ -box.o.a -box.o.b -box.o.c ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.7"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y ,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.8"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.10"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "&(.test.x.a a,$2)? &(.test.x.&(.test.o).a)? &(.test.x.{$2} a,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,$2)? &(.test.x.&(.test.o).a)? &(.test.x.{$2} a,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d.value); v == nil {
		ctx.err("%v", tst{d})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.11"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.x.12"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_trimprefix(ctx *testcase) {
	var pv Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if pv = p.value; pv == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = pv.string(ctx); ps == "" {
		ctx.err("%v", tst{pv})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{pv})
		return
	} else if s := "/testdata/builtins/trimprefix"; !strings.HasSuffix(ps, s) {
		ctx.err("%v → %s", tst{pv}, ps)
		return
	}

	if s := "pat0"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*globpat); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*word); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	}

	if s := "pat1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*percpat); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*word); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	}

	if s := "pat2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if t, y := p.elems[0].(*punct); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if t.token != PROOT {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if t.len() != 1 {
		ctx.err("%v ; %v", tst{t}, t.len())
	} else if _, y := t.elems[0].(*globmeta); !y {
		ctx.err("%v", tst{t.elems[0]})
	} else if _, y := p.elems[2].(*word); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if s, t := p.String(), "/**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "/**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a { // partially matched
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(x) < 0 || x[0] != "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(c) < 0 || c[0] == "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, "/"+joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	}

	if s := "pat3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if t, y := p.elems[0].(*punct); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if t.token != PROOT {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if _, y := p.elems[2].(*word); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if s, t := p.String(), "/%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "/%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(x) < 0 || x[0] != "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(c) < 0 || c[0] == "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, "/"+joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	}

	if s := "val1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val4"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val5"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val6"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_trimsuffix(ctx *testcase) {
	var pv Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if pv = p.value; pv == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = pv.string(ctx); ps == "" {
		ctx.err("%v", tst{pv})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{pv})
		return
	} else if s := "/testdata/builtins/trimsuffix"; !strings.HasSuffix(ps, s) {
		ctx.err("%v → %s", tst{pv}, ps)
		return
	}

	if s := "pat0"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "testdata/**"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "testdata/**"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if b != nil || c != nil {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	}

	if s := "pat1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "testdata/%%"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "testdata/%%"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if b != nil || c != nil {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	}

	if s := "pat2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if x, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if x.len() != 1 {
		ctx.err("%v ; %v", tst{x}, x.len())
	} else if x, y := p.elems[2].(*punct); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if x.token != PTAIL {
		ctx.err("%v ; %v", tst{x}, x.token)
	} else if x.String() != "" {
		ctx.err("%v ; %v", tst{x}, x.token)
	} else if s, t := p.String(), "testdata/**/"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "testdata/**/"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a { // partially matched
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if b != nil || c != nil {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	}

	if s := "pat3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if t, y := p.elems[2].(*punct); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if t.token != PTAIL {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if s, t := p.String(), "testdata/%%/"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "testdata/%%/"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if b != nil || c != nil {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	}

	var ds string
	if v := ctx.val("d"); v == nil {
		ctx.err("d")
	} else if x, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if x.len() < 2 {
		ctx.err("%v", tst{x})
	} else if t, y := x.elems[0].(*punct); !y {
		ctx.err("%v", tst{x.elems[0]})
	} else if PROOT != t.token {
		ctx.err("%v ; %v", tst{t}, x)
	// } else if t, y := x.elems[x.len()-1].(*punct); !y {
	// 	ctx.err("%v ; %v", tst{x.elems[x.len()-1]}, x)
	// } else if PTAIL != t.token {
	// 	ctx.err("%v", tst{t})
	} else if ds = v.string(ctx); ds == "" {
		ctx.err("%v", tst{v})
	} else if !strings.HasPrefix(ds, "/") {
		ctx.err("%v %s", tst{v}, ds)
	// } else if !strings.HasSuffix(ds, "/") {
	// 	ctx.err("%v %s", tst{v}, ds)
	}

	if s := "val1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if s, t := ds+"/", v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if s, t := ds+"/", v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if s, t := ds+"/", v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val4"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if s, t := ds, v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val5"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if s, t := ds, v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val6"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if s, t := ds, v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_addprefix(ctx *testcase) {
	if s := "val1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*pair); !y {
		ctx.err("%v", tst{v})
	} else if s := p.key.String(); s != "-std" {
		ctx.err("%v ; %v", tst{v}, tst{p.key})
	} else if s := p.val.String(); s != "foo" {
		ctx.err("%v ; %v", tst{v}, tst{p.val})
	} else if s, t := v.String(), "-std=foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t = v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l0, y := l.elems[0].(*pair); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if s, t := l0.key.String(), "-std"; s != t {
		ctx.err("%v → %s != %s", tst{l0.key}, t, s)
	} else if s, t := l0.val.String(), "foo"; s != t {
		ctx.err("%v → %s != %s", tst{l0.val}, t, s)
	} else if l1, y := l.elems[1].(*pair); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := l1.key.String(), "-std"; s != t {
		ctx.err("%v → %s != %s", tst{l1.key}, t, s)
	} else if s, t := l1.val.String(), "bar"; s != t {
		ctx.err("%v → %s != %s", tst{l1.val}, t, s)
	} else if s, t := v.String(), "-std=foo -std=bar"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t = v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{v}, l.len())
	} else if l0, y := l.elems[0].(*pair); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l0.key.(flag); !y {
		ctx.err("%v", tst{l0.key})
	} else if l1, y := l.elems[1].(cond); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if l1c, y := l1.Value.(*pair); !y {
		ctx.err("%v", tst{l1.Value})
	} else if _, y := l1c.key.(flag); !y {
		ctx.err("%v", tst{l1c.key})
	} else if l1v, y := l1c.val.(disjunction); !y {
		ctx.err("%v", tst{l1c.val})
	} else if l1vc, y := l1v.Value.(*closure); !y {
		ctx.err("%v", tst{l1v.Value})
	} else if s, t := l1vc.x.String(), "bar"; s != t {
		ctx.err("%v → %s != %s", tst{l1vc.x}, t, s)
	} else if s, t := v.String(), "-foo=bar -foo={&(bar)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if l, y := t.(*list); !y {
		ctx.err("%v", tst{t})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{t}, l.len())
	} else if _, y := l.elems[0].(*pair); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(cond); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := v.string(ctx), "-foo=bar"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val4"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix -foo={},bar &(bar))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "-foo=bar"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val5"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if len(l.elems) != 2 {
		ctx.err("%v %v", tst{v}, l.elems)
	} else if _, y := l.elems[0].(*compound); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if l1, y := l.elems[1].(cond); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if l1v, y := l1.Value.(*compound); !y {
		ctx.err("%v", tst{l1.Value})
	} else if l1v.len() != 2 {
		ctx.err("%v ; %v", tst{l1v}, l1v.len())
	} else if l1vd, y := l1v.elems[1].(disjunction); !y {
		ctx.err("%v", tst{l1v.elems[1]})
	} else if _, y := l1vd.Value.(*closure); !y {
		ctx.err("%v", tst{l1vd.Value})
	} else if s, t := v.String(), "foobar foo{&(bar)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if l, y := t.(*list); !y {
		ctx.err("%v", tst{t})
	} else if l.len() != 2 {
		ctx.err("%v %v", tst{t}, l.len())
	} else if _, y := l.elems[0].(*compound); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(cond); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := v.string(ctx), "foobar"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val6"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo,bar &(bar))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foobar"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_addsuffix(ctx *testcase) {
	if s := "val1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*pair); !y {
		ctx.err("%v", tst{v})
	} else if s, t := p.key.String(), "foo"; s != t {
		ctx.err("%v → %s != %s", tst{p.key}, t, s)
	} else if s, t := p.val.String(), "xxx"; s != t {
		ctx.err("%v → %s != %s", tst{p.val}, t, s)
	} else if s, t := v.String(), "foo=xxx"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v ; %d", tst{v}, l.len())
	} else if p, y := l.elems[0].(*pair); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if s, t := p.key.String(), "foo"; s != t {
		ctx.err("%v → %s != %s", tst{p.key}, t, s)
	} else if s, t := p.val.String(), "xxx"; s != t {
		ctx.err("%v → %s != %s", tst{p.val}, t, s)
	} else if p, y := l.elems[1].(*pair); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := p.key.String(), "bar"; s != t {
		ctx.err("%v → %s != %s", tst{p.key}, t, s)
	} else if s, t := p.val.String(), "xxx"; s != t {
		ctx.err("%v → %s != %s", tst{p.val}, t, s)
	} else if s, t := v.String(), "foo=xxx bar=xxx"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_contains(ctx *testcase) {
	if s := ".test.1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if x, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if t, y := x.x.(*builtin); !y {
		ctx.err("%v", tst{x.x})
	} else if t.name != "contains" {
		ctx.err("%v", tst{x.x})
	} else if s, t := v.String(), "$(contains a,a b c $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "$(contains a,a b c $1)", v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if x := ctx.val(d, "x"); x == nil {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "{=true}"; s != t { // $(contains a,a b c x)
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := x.string(ctx), "true"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if x, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if t, y := x.x.(*builtin); !y {
		ctx.err("%v", tst{x.x})
	} else if t.name != "contains" {
		ctx.err("%v", tst{x.x})
	} else if s, t := v.String(), "$(contains x b c,a b c $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if x := ctx.val(d, "x"); x == nil {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "{=true}"; s != t { // $(contains b c,a b c x)
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := x.string(ctx), "true"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if x, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if t, y := x.x.(*builtin); !y {
		ctx.err("%v", tst{x.x})
	} else if t.name != "contains" {
		ctx.err("%v", tst{x.x})
	} else if s, t := v.String(), "$(contains x,a b c $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	}
}

func testBuiltin_contains2(ctx *testcase) {
	if s := "val"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=list {=word a} {=word b} {=word c} {=delegate {=auto 1}}}" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.x"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=delegate {=rule {=word foo}}}" {
		ctx.err("%v", tst{v})
	} else if x, y := v.(*delegate); !y {
		ctx.err("%v", tst{x})
	} else if a, y := x.x.(entry); !y {
		ctx.err("%v", tst{x.x})
	} else if a == nil {
		ctx.err("%v", tst{x.x})
	} else if s, t := v.String(), "${foo}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := x.string(ctx), "a b c foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.y"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=list {=word a} {=word b} {=word c} {=word foo}}" {
		ctx.err("%v", tst{v})
	} else if true {
		// ...
	} else if x, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if o, y := x.x.(*def); !y {
		ctx.err("%v", tst{x.x})
	} else if s, t := o.value.String(), "a b c $1"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "a b c foo", v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if s, t := x.string(ctx), "a b c foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test.z"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=list {=word a} {=word b} {=word c} {=word foo}}" {
		ctx.err("%v", tst{v})
	} else if true {
		// ...
	} else if x, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if o, y := x.x.(*def); !y {
		ctx.err("%v", tst{x.x})
	} else if s, t := o.value.String(), "a b c $1"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := x.string(ctx), "a b c foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := ".test"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if x, y := v.(*boolean); !y {
		ctx.err("%v", tst{v})
	} else if !x.bool {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "true"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_join(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=compound {=word foo} {=flag {=null}} {=word bar} {=flag {=null}} {=word xx} {=flag {=null}} {=word yy} {=flag {=null}} {=word zz}}" {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(join foo bar xx yy zz,-)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(join &(target.arch) &(target.vendor) &(target.os) &(target.abi),-)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foo-bar-a-0"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if c, y := v.(conjunction); !y {
		ctx.err("%v", tst{v})
	} else if c.len() != 5 {
		ctx.err("%v", tst{c.list})
	} else if c.sep == nil {
		ctx.err("%v", tst{c})
	} else if s, t := c.sep.String(), "-"; s != t {
		ctx.err("%v → %s != %s", tst{c.sep}, t, s)
	} else if s, t := v.String(), "{{&(target.arch) &(XXX) &(target.vendor) &(target.os) &(target.abi)}-}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foo-bar-a-0"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_logic(ctx *testcase) {
	s := "val1"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if x, y := v.(*word); !y {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := x.string(ctx), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*word); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(or(-final) &(none),a)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(or &(none),a)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "a", v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(or a,&(none))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(and $1,$2,$3)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*word); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*group); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "(variant/$(or(-final) $(base &(variant)),bootstrap))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "(variant/bootstrap)", v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if g, y := t.(*group); !y || g.len() != 1 {
		ctx.err("%v, %v", tst{t}, g)
	} else if _, y := g.elems[0].(*path); !y {
		ctx.err("%v", tst{g.elems[0]})
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*group); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "(variant/bootstrap)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "variant/bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*word); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or $(base &(variant)),bootstrap)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(base $(or &(variant),bootstrap))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testBuiltin_if(ctx *testcase) {
	var s string

	s = "x1"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(if {=yes},yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x2"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(if {=no},yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x3"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x4"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x5"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x6"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x7"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(if &(none),yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x8"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x9"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x10"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testBuiltin_or(ctx *testcase) {
	if v := ctx.val("val11.0"); v == nil {
		ctx.err("val11.0")
	} else if v.String() != "-no -yes -false -true" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "-no -yes -false -true" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else { for _, t := range l.elems {
		if f, y := t.(flag); !y {
			ctx.err("%v", tst{t})
		} else if _, y := f.Value.(*word); !y {
			ctx.err("%v", tst{f.Value})
		} else if !f.true(ctx) {
			ctx.err("%v", tst{t})
		} else if !f.Value.true(ctx) {
			ctx.err("%v", tst{f.Value})
		}
	}}

	if v := ctx.val("val11"); v == nil {
		ctx.err("val11")
	} else if v.String() != "-no" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "-no" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	} else if f, y := v.(flag); !y {
		ctx.err("%v", tst{v})
	} else if _, y := f.Value.(*word); !y {
		ctx.err("%v", tst{f.Value})
	}

	if v := ctx.val("val12"); v == nil {
		ctx.err("val12")
	} else if v.String() != "-yes" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "-yes" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	} else if f, y := v.(flag); !y {
		ctx.err("%v", tst{v})
	} else if _, y := f.Value.(*word); !y {
		ctx.err("%v", tst{f.Value})
	}

	if v := ctx.val("val13"); v == nil {
		ctx.err("val13")
	} else if v.String() != "-false" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "-false" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	} else if f, y := v.(flag); !y {
		ctx.err("%v", tst{v})
	} else if _, y := f.Value.(*word); !y {
		ctx.err("%v", tst{f.Value})
	}
}

func testBuiltin_xor(ctx *testcase) {
	if d := ctx.def("val14.1"); d == nil {
		ctx.err("val14.1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.true(ctx) {
		ctx.err("%v", tst{v})
	} else if _, y := v.(*null); !y {
		ctx.err("%v", tst{v})
	} else if t := v.String(); t != "{}" {
		ctx.err("%v ⇒ %s", tst{v}, t)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v ⇒ %s", tst{v}, t)
	}

	if d := ctx.def("val14.2"); d == nil {
		ctx.err("val14.2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if !v.true(ctx) {
		ctx.err("%v", tst{v})
	} else if t, y := v.(*boolean); !y {
		ctx.err("%v", tst{v})
	} else if !t.bool {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%v ⇒ %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "true"; s != t {
		ctx.err("%v ⇒ %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val14.3"); d == nil {
		ctx.err("val14.3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if !v.true(ctx) {
		ctx.err("%v", tst{v})
	} else if t, y := v.(*boolean); !y {
		ctx.err("%v", tst{v})
	} else if !t.bool {
		ctx.err("%v", tst{v})
	} else if v.String() != "{=true}" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "true" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}

	if d := ctx.def("val14.4"); d == nil {
		ctx.err("val14.4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.true(ctx) {
		ctx.err("%v", tst{v})
	} else if _, y := v.(*null); !y {
		ctx.err("%v", tst{v})
	} else if v.String() != "{}" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}
}

func testBuiltin_file(ctx *testcase) {
	var fullFooTxt = filepath.Join(_workdir(ctx), "foo.txt")

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if f := (as{v}.file(ctx)); f == nil {
		ctx.err("%v", tst{v})
	} else if o := (as{v}.fullname(ctx)); o.Value == nil {
		ctx.err("%v ; %v %v", tst{v}, tst{o.Value}, y)
	} else if f, y := o.Value.(*file); !y || f == nil {
		ctx.err("%v", tst{o.Value})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	} else if o.cmp(ctx, f) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if f.cmp(ctx, o) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if cmp(ctx, o, f) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if cmp(ctx, f, o) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if p := _pathstr(ctx, f.fullname()); p == nil {
		ctx.err("%v %v", tst{v}, f)
	} else if t := o.cmp(ctx, p); t != cmpEqual {
		ctx.err("%v, %v %v", t, tst{o}, tst{p})
	} else if true {
		// ...
	} else if p.cmp(ctx, o) != cmpEqual {
		ctx.err("%v %v", tst{p}, tst{o})
	} else if cmp(ctx, p, o) != cmpEqual {
		ctx.err("%v %v", tst{p}, tst{o})
	} else if cmp(ctx, o, p) != cmpEqual {
		ctx.err("%v %v", tst{o}, tst{p})
	} else if s, t := v.String(), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=file foo.txt}" {
		ctx.err("%v", tst{v})
	} else if v.string(ctx) != "foo.txt" {
		ctx.err("%v", tst{v})
	} else if f, y := v.(*file); !y || f == nil {
		ctx.err("%v", tst{v})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=file foo.txt}" {
		ctx.err("%v", tst{v})
	} else if v.string(ctx) != fullFooTxt { o, y := v.(fullname)
		ctx.err("%v ; %v %v", tst{v}, tst{o.Value}, y)
	} else if o, y := v.(fullname); !y {
		ctx.err("%v", tst{v})
	} else if f, y := o.Value.(*file); !y || f == nil {
		ctx.err("%v", tst{o.Value})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	}
}
