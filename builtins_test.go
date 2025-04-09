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
	s := "foo"
    if d := ctx.def(s); d == nil {
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
	} else if i = 4; s.vals[i].String() != "" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 5; s.vals[i].String() != "{=undef x}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 6; s.vals[i].String() != "{}" || s.bools[i] { // {=null}
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].String() != "{x}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].string(ctx) != "x" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 8; s.vals[i].String() != "foobar" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if _, y := s.vals[i].(*word); !y {
		ctx.err("%v %v", tst{s.vals[i]}, s.vals[i])
	} else {
		type rec struct{ string; bool }
		for i, r := range []rec{
			rec{"{=true}", true},
			rec{"{=false}", false},
			rec{"{=yes}", true},
			rec{"{=no}", false},
			rec{"", false},
			rec{"{=undef x}", false},
			rec{"{}", false},
			rec{"{x}", true},
			rec{"foobar", true},
			rec{"1", true},
			rec{"0", false},
			rec{"$(equal $(foo),foo)", true},
		}{
			if t := s.vals[i].String(); t != r.string {
				ctx.err("%s != %s : %s", t, r.string, tst{s.vals[i]})
			} else if s.bools[i] != r.bool {
				ctx.err("%v != %v : %v : %v", s.bools[i], r.bool, s.vals[i], tst{s.vals[i]})
			}
		}
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
		var c = original{ctx,nil,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
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
		var c = original{ctx,nil,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
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
		var c = original{ctx,nil,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{dir:workdirInc}
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
		var c = original{ctx,nil,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{dir:workdirInc}
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
		var c = original{ctx,nil,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
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
		var c = original{ctx,nil,defExpand1}
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

func test__file0(ctx *testcase) {
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

func test__foreach(ctx *testcase) {
	{
		var c0 = __foreach{}
		var c1 = partial{&c0, nonePart}
		var c2 = __foreach{}
		c0.evocation = &evocation{automatic{Context:ctx}, nil, nil, nil}
		c2.evocation = &evocation{automatic{Context:c1}, nil, nil, nil}
		if cast[*builtinbase](&c0) == nil {
			ctx.err("builtinbase")
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

	var s string
	var test_1_value Value
	var j = _project(ctx)

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", j, s)
	} else if test_1_value = d.value; test_1_value == nil {
		ctx.err("%v", d)
	} else if s, t := d.value.String(), "x $1 $2 $3 $4 $(foreach $1,&(.test.h)$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if s, t := d.value.string(ctx), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=word a} {=word b} {=word c}} {=word X} {=word Y} {=word Z} {=list {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word h}}} {=word a}} {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word h}}} {=word b}} {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word h}}} {=word c}}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "x a b c X Y Z &(.test.h)a &(.test.h)b &(.test.h)c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=word a} {=word b} {=word c}} {=word X} {=word Y} {=word Z} {=list {=flag {=word a}} {=flag {=word b}} {=flag {=word c}}}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", j, s)
	} else if s0, t := d.value.String(), "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"; s0 != t {
		ctx.err("%s != %s : %v", s0, t, tst{d.value})
	} else if s1, t := d.value.string(ctx), "x xq xp"; s1 != t {
		ctx.err("%s != %s : %v", s1, t, tst{d.value})
	} else if x, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if x.len() != 2 {
		ctx.err("%d %v", x.len(), tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=disjunction {=closure {=def .test.h}}} {=word a}} {=compound {=word x} {=disjunction {=closure {=def .test.h}}} {=word b}} {=compound {=word x} {=disjunction {=closure {=def .test.h}}} {=word c}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=flag {=word a}}} {=compound {=word x} {=flag {=word b}}} {=compound {=word x} {=flag {=word c}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.21"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v", l.elems)
	} else if s, t := l.elems[0].String(), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[0]})
	} else if s, t := l.elems[1].String(), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t, y := l.elems[1].(*list); !y {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t.len() != 2 {
		ctx.err("%d, %v", t.len(), tst{t})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if !equal(ctx, v, d.value) {
		ctx.err("%v → %v (%v)", tst{v}, d, v.cmp(ctx, d.value))
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.22"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.23"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach q p $(foreach $1,&(.test.xx)$_),x$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word a}} {=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word b}} {=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word c}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"aa", "bb", "cc"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=word aa}} {=compound {=word x} {=word bb}} {=compound {=word x} {=word cc}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp xaa xbb xcc"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x $(foreach $1,&(.test.$_)$1{}88) y $(foreach $1,$(closure .test.$_)$1{}99) z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s != v.String() {
		ctx.err("%s != %s : %v", v, s, tst{v})
	} else if s, t := v.string(ctx), "x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=word y} {=word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 3 {
		ctx.err("%d %v", x.len(), v)
	} else if v := ctx.val(d, defExpand1, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=word zz}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=word zz}}} {=word y} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=decimal 99}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=decimal 99}}} {=word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x {&(.test.foo)}foo barzz {&(.test.bar)}foo barzz y {&(.test.foo)}foo bar99 {&(.test.bar)}foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "x barzz barzz y bar99 bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d, defExpand2, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=word zz}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=word zz}}} {=word y} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=decimal 99}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=decimal 99}}} {=word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x {&(.test.foo)}foo barzz {&(.test.bar)}foo barzz y {&(.test.foo)}foo bar99 {&(.test.bar)}foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "x foo barzz foo barzz y foo bar99 foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	}

	s = ".test.4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,&(.test.$_.$(or $4,$3)))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s != v.String() {
		ctx.err("%s != %s : %v", v, s, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.$(or $4,$3)) &(.test.y.$(or $4,$3))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", ""); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "", "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", []string{}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", []string{}, "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x do.smart)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if f, y := v.(*file); !y {
		ctx.err("%v : %v", v, tst{v})
	} else if f.name != "do.smart" {
		ctx.err("%v : %v", v, tst{v})
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.6"
    if d := ctx.def(s); d == nil || d.value == nil {
		ctx.err("%s: %v", s, d)
	} else if v := ctx.val(d, defExpand1, "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "- x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.7"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach a $(foreach $1,&(.test.z)$_{}zz) b,x$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa x&(.test.z)y1zz x&(.test.z)y2zz x&(.test.z)y3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 5 {
		ctx.err("%v, %v", x.len(), v)
	} else if _, y = x.elems[1].(*list); !y && false {
		ctx.err("%v", tst{x.elems[1]})
	} else if v := ctx.val(d, defExpand2, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(stat $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "do.smart"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if f, y := v.(*file); !y || f == nil {
		ctx.err("%v", tst{v})
	}
}

func test__foreach1(ctx *testcase) {
	s := ".test.1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x $1,B b)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s != v.String() {
		ctx.err("%v != %s", tst{v}, s)
	} else if s, t := v.string(ctx), "foo test-foo test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "xxx"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), `&(.test.s) test-foo &(.test.a) &(.test~foo.a) &(.test.B) &(.test~foo.B) &(.test.b) &(.test~foo.b)`; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo test-foo test-a test-foo-a test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

func test__foreach2(ctx *testcase) {
	s := ".test.foreach.a"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $(foreach $2,a$_),b$_)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.foreach.b"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.foreach.c"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.b) $(foreach &(.test.foreach.x) $(foreach $1 $2,$(closure .test.foreach.x.$_)),-x$_)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.b)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.c)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx by bz baxx bayy -x&(.test.foreach.x)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.c $1,4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx by bz baxx bayy -x&(.test.foreach.x) -x&(.test.foreach.x.4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "3"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx by bz baxx bayy -x&(.test.foreach.x) -x&(.test.foreach.x.3) -x&(.test.foreach.x.4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw -xV -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.d)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "1", "2"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	if v := ctx.val(".test.foreach.d", defExpand1, "1", "2"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(".test.foreach.d", defExpand1, "1", "2", "a", "b"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "-xa -xb -ya -yb"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

func test__foreach3(ctx *testcase) {
	var s string

	s = ".test.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,$_$3)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{d.value})
	} else {
		if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "acc bcc"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "", []string{"1", "2", "3"}, "x"); v == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "1x 2x 3x"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "a b"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "ax bx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.x"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := v.String(), "$(foreach $1 $2,$(addprefix std={},&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.a)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", nil); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.a)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.b)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.a)? std=&(.test.b)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std=&(.test.a)? std=&(.test.if.x)? std=&(.test.if.y)? std=&(.test.if.z)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std=$(foreach $1 $2,$_$3)? std=xxx std=yyy std=&(.test.if.z)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.y"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
		if v := ctx.val(d, defExpand1, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))? $(if &(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.y),std=&(.test.if.y))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))? $(if &(.test.if.y),std=&(.test.if.y))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "zzz", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.z"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := d.value.String(), "$(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
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
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.String(), "$(if $(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.String(), "std=&(.test.if.x)? $(if $(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
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

func test__foreach4(ctx *testcase) {
	s := ".test.1"
    if d := ctx.def(s); d == nil {
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

	s = ".test.2"
    if d := ctx.def(s); d == nil {
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
	} else if x := ctx.val(z, defExpand2); x == nil {
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
	} else if x := ctx.val(z, defExpand2); x == nil {
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
	} else if x := ctx.val(z, defExpand2, "xx", "yy"); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X{$(foreach $(foreach $(foreach $(foreach $1 $2,~$_),~$_),~$_),~$_)}"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if x = ctx.val(x, defExpand2, "xx", "yy"); x == nil {
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

	s = ".test.3"
    if d := ctx.def(s); d == nil {
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

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
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

func test__foreach5(ctx *testcase) {
	var s string

	s = ".test.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "o"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.o.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.o.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
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

	s = ".test.x.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_) ~b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ -bo{&(.test.x.o.a)}? -bo{&(.test.x.o.b)}? ~b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.0"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d); v == nil {
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
	} else if x := ctx.val(v, defExpand2, "a", "b", "c"); x == nil {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "a~ $(foreach $(foreach $1 $2,$(.test.x.o.$_)),-ao$_)? ~a x.o.a b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_)? ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	}

	s = ".test.x.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x $1)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else {
		if v := ctx.val(d); v == nil {
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

	s = ".test.x.2"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.3"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.4"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.5"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.6"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.7"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.8"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.10"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "&(.test.x.a a,$2)? &(.test.x.&(.test.o).a)? &(.test.x.{$2} a,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else if v := ctx.val(d); v == nil {
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

	s = ".test.x.11"
	if d := ctx.def(s); d == nil {
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

	s = ".test.x.12"
	if d := ctx.def(s); d == nil {
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

func test__trimprefix(ctx *testcase) {
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

	s := "pat0"
    if d := ctx.def(s); d == nil {
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

	s = "pat1"
    if d := ctx.def(s); d == nil {
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

	s = "pat2"
    if d := ctx.def(s); d == nil {
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

	s = "pat3"
    if d := ctx.def(s); d == nil {
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

	s = "val1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val6"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func test__trimsuffix(ctx *testcase) {
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

	s := "pat0"
    if d := ctx.def(s); d == nil {
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

	s = "pat1"
    if d := ctx.def(s); d == nil {
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

	s = "pat2"
    if d := ctx.def(s); d == nil {
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

	s = "pat3"
    if d := ctx.def(s); d == nil {
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

	s = "val1"
    if d := ctx.def(s); d == nil {
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

	s = "val2"
    if d := ctx.def(s); d == nil {
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

	s = "val3"
    if d := ctx.def(s); d == nil {
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

	s = "val4"
    if d := ctx.def(s); d == nil {
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

	s = "val5"
    if d := ctx.def(s); d == nil {
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

	s = "val6"
    if d := ctx.def(s); d == nil {
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

func test__addprefix(ctx *testcase) {
	var s string

	s = "val1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=pair {=flag {=word std}}={=none}}} {=list {=word foo}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix -std=,foo)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "-std=foo"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=pair {=flag {=word std}}={=word foo}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "-std=foo"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=pair {=flag {=word std}}={=none}}} {=list {=word foo} {=word bar}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix -std=,foo bar)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "-std=foo -std=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=flag {=word std}}={=word foo}} {=pair {=flag {=word std}}={=word bar}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "-std=foo -std=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=pair {=flag {=word foo}}={=none}}} {=list {=word bar} {=closure {=word none}}}}"; s != t {
		ctx.err("%s: %s != %s", v, s, t)
	} else if s, t := v.String(), "$(addprefix -foo=,bar &(none))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=flag {=word foo}}={=word bar}} {=pair {=flag {=word foo}}={=disjunction {=closure {=word none}}}}}"; s != t {
		ctx.err("%s: %s != %s", v, s, t)
	} else if s, t := v.String(), "-foo=bar -foo={&(none)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=pair {=flag {=word foo}}={=word bar}}"; s != t {
		ctx.err("%s: %s != %s", v, s, t)
	} else if s, t := v.String(), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=pair {=word std}={=none}}} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix std=,&(.test.$1))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=pair {=word std}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}"; s != t {
		ctx.err("%s: %s != %s", v, s, t)
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "std=ax std=ay std=az"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word std}={=word ax}} {=pair {=word std}={=word ay}} {=pair {=word std}={=word az}}}"; s != t {
		ctx.err("%s: %s != %s", v, s, t)
	} else if s, t := v.String(), "std=ax std=ay std=az"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word std}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}} {=pair {=word std}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}}}"; s != t {
		ctx.err("%s: %s != %s", v, s, t)
	} else if s, t := v.String(), "std={&(.test.a)} std={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "std=ax std=ay std=az std=bx std=by std=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word std}={=word ax}} {=pair {=word std}={=word ay}} {=pair {=word std}={=word az}} {=pair {=word std}={=word bx}} {=pair {=word std}={=word by}} {=pair {=word std}={=word bz}}}"; s != t {
		ctx.err("%s: %s != %s", v, s, t)
	} else if s, t := v.String(), "std=ax std=ay std=az std=bx std=by std=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=word foo}} {=list {=word bar} {=closure {=word none}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix foo,bar &(none))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=compound {=word foo} {=word bar}} {=compound {=word foo} {=disjunction {=closure {=word none}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foobar foo{&(none)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=compound {=word foo} {=word bar}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=word foo} {=word bar}} {=list {=pair {=none}={=word xxx}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix foo bar,=xxx)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=xxx bar=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=word xxx}} {=pair {=word bar}={=word xxx}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foo=xxx bar=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=word foo} {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}} {=list {=pair {=none}={=word xxx}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix foo &(.test.$1),=xxx)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=word xxx}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=word xxx}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foo=xxx {&(.test.$1)}=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=word xxx}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=word xxx}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}={=word xxx}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foo=xxx {&(.test.a)}=xxx {&(.test.b)}=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=xxx ax=xxx ay=xxx az=xxx bx=xxx by=xxx bz=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=word xxx}} {=pair {=word ax}={=word xxx}} {=pair {=word ay}={=word xxx}} {=pair {=word az}={=word xxx}} {=pair {=word bx}={=word xxx}} {=pair {=word by}={=word xxx}} {=pair {=word bz}={=word xxx}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foo=xxx ax=xxx ay=xxx az=xxx bx=xxx by=xxx bz=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=word foo} {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}} {=list {=pair {=none}={=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix foo &(.test.$1),=&(.test.$1))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foo={&(.test.$1)} {&(.test.$1)}={&(.test.$1)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}} {=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}} {=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b}}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foo={&(.test.a)} foo={&(.test.b)} {&(.test.a)}={&(.test.a)} {&(.test.a)}={&(.test.b)} {&(.test.b)}={&(.test.a)} {&(.test.b)}={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=ax foo=ay foo=az foo=bx foo=by foo=bz ax=ax ax=ay ax=az ay=ax ay=ay ay=az az=ax az=ay az=az ax=bx ax=by ax=bz ay=bx ay=by ay=bz az=bx az=by az=bz bx=ax bx=ay bx=az by=ax by=ay by=az bz=ax bz=ay bz=az bx=bx bx=by bx=bz by=bx by=by by=bz bz=bx bz=by bz=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=word ax}} {=pair {=word foo}={=word ay}} {=pair {=word foo}={=word az}} {=pair {=word foo}={=word bx}} {=pair {=word foo}={=word by}} {=pair {=word foo}={=word bz}} {=pair {=word ax}={=word ax}} {=pair {=word ax}={=word ay}} {=pair {=word ax}={=word az}} {=pair {=word ax}={=word bx}} {=pair {=word ax}={=word by}} {=pair {=word ax}={=word bz}} {=pair {=word ay}={=word ax}} {=pair {=word ay}={=word ay}} {=pair {=word ay}={=word az}} {=pair {=word ay}={=word bx}} {=pair {=word ay}={=word by}} {=pair {=word ay}={=word bz}} {=pair {=word az}={=word ax}} {=pair {=word az}={=word ay}} {=pair {=word az}={=word az}} {=pair {=word az}={=word bx}} {=pair {=word az}={=word by}} {=pair {=word az}={=word bz}} {=pair {=word bx}={=word ax}} {=pair {=word bx}={=word ay}} {=pair {=word bx}={=word az}} {=pair {=word bx}={=word bx}} {=pair {=word bx}={=word by}} {=pair {=word bx}={=word bz}} {=pair {=word by}={=word ax}} {=pair {=word by}={=word ay}} {=pair {=word by}={=word az}} {=pair {=word by}={=word bx}} {=pair {=word by}={=word by}} {=pair {=word by}={=word bz}} {=pair {=word bz}={=word ax}} {=pair {=word bz}={=word ay}} {=pair {=word bz}={=word az}} {=pair {=word bz}={=word bx}} {=pair {=word bz}={=word by}} {=pair {=word bz}={=word bz}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "foo=ax foo=ay foo=az foo=bx foo=by foo=bz ax=ax ax=ay ax=az ax=bx ax=by ax=bz ay=ax ay=ay ay=az ay=bx ay=by ay=bz az=ax az=ay az=az az=bx az=by az=bz bx=ax bx=ay bx=az bx=bx bx=by bx=bz by=ax by=ay by=az by=bx by=by by=bz bz=ax bz=ay bz=az bz=bx bz=by bz=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val9"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addprefix} {=list {=compound {=word fo} {=flag {=null}}}} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "$(addprefix fo-,&(.test.$1.x.$2.y.$3.z))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}} {=punct .} {=word x} {=punct .} {=delegate {=auto 2}} {=punct .} {=word y} {=punct .} {=delegate {=auto 3}} {=punct .} {=word z}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "fo-{&(.test.$1.x.$2.y.$3.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 1} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 2} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a} {=punct .} {=word x} {=punct .} {=word 3} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b} {=punct .} {=word x} {=punct .} {=word 1} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b} {=punct .} {=word x} {=punct .} {=word 2} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word b} {=punct .} {=word x} {=punct .} {=word 3} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word c} {=punct .} {=word x} {=punct .} {=word 1} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word c} {=punct .} {=word x} {=punct .} {=word 2} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}} {=compound {=word fo} {=flag {=null}} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word c} {=punct .} {=word x} {=punct .} {=word 3} {=punct .} {=word y} {=punct .} {=word 0} {=punct .} {=word z}}}}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "fo-{&(.test.a.x.1.y.0.z)} fo-{&(.test.a.x.2.y.0.z)} fo-{&(.test.a.x.3.y.0.z)} fo-{&(.test.b.x.1.y.0.z)} fo-{&(.test.b.x.2.y.0.z)} fo-{&(.test.b.x.3.y.0.z)} fo-{&(.test.c.x.1.y.0.z)} fo-{&(.test.c.x.2.y.0.z)} fo-{&(.test.c.x.3.y.0.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "fo-ax fo-ay fo-az fo-bx fo-by fo-bz fo-cx fo-cy fo-cz fo-dx fo-dy fo-dz fo-ex fo-ey fo-ez fo-fx fo-fy fo-fz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=compound {=word fo} {=flag {=null}} {=word ax}} {=compound {=word fo} {=flag {=null}} {=word ay}} {=compound {=word fo} {=flag {=null}} {=word az}} {=compound {=word fo} {=flag {=null}} {=word bx}} {=compound {=word fo} {=flag {=null}} {=word by}} {=compound {=word fo} {=flag {=null}} {=word bz}} {=compound {=word fo} {=flag {=null}} {=word cx}} {=compound {=word fo} {=flag {=null}} {=word cy}} {=compound {=word fo} {=flag {=null}} {=word cz}} {=compound {=word fo} {=flag {=null}} {=word dx}} {=compound {=word fo} {=flag {=null}} {=word dy}} {=compound {=word fo} {=flag {=null}} {=word dz}} {=compound {=word fo} {=flag {=null}} {=word ex}} {=compound {=word fo} {=flag {=null}} {=word ey}} {=compound {=word fo} {=flag {=null}} {=word ez}} {=compound {=word fo} {=flag {=null}} {=word fx}} {=compound {=word fo} {=flag {=null}} {=word fy}} {=compound {=word fo} {=flag {=null}} {=word fz}}}"; s != t {
		ctx.err("%s != %s ; %s", s, t, v)
	} else if s, t := v.String(), "fo-ax fo-ay fo-az fo-bx fo-by fo-bz fo-cx fo-cy fo-cz fo-dx fo-dy fo-dz fo-ex fo-ey fo-ez fo-fx fo-fy fo-fz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

func test__addsuffix(ctx *testcase) {
	var s string

	s = "val1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addsuffix} {=list {=pair {=none}={=word xxx}}} {=list {=word foo}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "$(addsuffix =xxx,foo)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(ctx), "foo=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=pair {=word foo}={=word xxx}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "foo=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s := v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	s = "val2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addsuffix} {=list {=pair {=none}={=word xxx}}} {=list {=word foo} {=word bar}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "$(addsuffix =xxx,foo bar)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(ctx), "foo=xxx bar=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=word xxx}} {=pair {=word bar}={=word xxx}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "foo=xxx bar=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s := v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	s = "val5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=builtin addsuffix} {=list {=pair {=none}={=word xxx}}} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "$(addsuffix =xxx,&(.test.$1))"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=delegate {=auto 1}}}}}={=word xxx}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "{&(.test.$1)}=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}
}

func test__contains(ctx *testcase) {
	var s string

	s = ".test.1"
	if d := ctx.def(s); d == nil {
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

	s = ".test.2"
	if d := ctx.def(s); d == nil {
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

	s = ".test.3"
	if d := ctx.def(s); d == nil {
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

func test__contains2(ctx *testcase) {
	var s string

	s = "val"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=list {=word a} {=word b} {=word c} {=delegate {=auto 1}}}" {
		ctx.err("%v", tst{v})
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
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

	s = ".test.y"
    if d := ctx.def(s); d == nil {
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

	s = ".test.z"
    if d := ctx.def(s); d == nil {
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

	s = ".test"
    if d := ctx.def(s); d == nil {
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

func test__join(ctx *testcase) {
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

func test__logic(ctx *testcase) {
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

func test__if(ctx *testcase) {
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
	} else if s, t := ts(v), "{=word no}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x8"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=word yes}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
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
	} else if s, t := v.String(), "$(ifarg 1,yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x11"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x12"
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

func test__or(ctx *testcase) {
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

func test__xor(ctx *testcase) {
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

func test__file(ctx *testcase) {
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
