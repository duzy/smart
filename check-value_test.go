//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
)

type testValueStruct struct {
	assert_bool bool
	assert_value Value
}
func testValueAssertHook(ctx Context, v Value, b bool, i any) {
	st := i.(*testValueStruct)
	st.assert_bool = b
	st.assert_value = v
}
func testValue(ctx testcase1) {
	st := ctx.i.(*testValueStruct)
	if st.assert_value == nil {
		// ctx.err("assert: nil value")
	} else if st.assert_bool {
		// ctx.err("assert")
	}

	j := _project(ctx)

	if d := ctx.def("vals"); d == nil {
		ctx.err("vals")
	} else if d.scope != j.scope {
		ctx.err("%v", tst{d})
	} else if d.scope.project != j {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), `{word foo/bar 'strlit' "strcomp" 0 1} {=yes} {=false} {=true} {=path foo} foo/bar {=file foobar} **.c {=regex xx} 1 0.1 {}`; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), `word foo/bar strlit strcomp 0 1 yes false true foo foo/bar foobar **.c xx 1 0.1`; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("cond0"); d == nil {
		ctx.err("cond0: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x?"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "x"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("cond01"); d == nil {
		ctx.err("cond01: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x?y"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "xy"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("cond02"); d == nil {
		ctx.err("cond02: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x???y"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "xy"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("cond03"); d == nil {
		ctx.err("cond03: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x&(something)?y"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}


	if d := ctx.def("cond11"); d == nil {
		ctx.err("cond11: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "xy"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "xy"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("cond12"); d == nil {
		ctx.err("cond12: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "xy"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "xy"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("cond13"); d == nil {
		ctx.err("cond13: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x&(something)y?"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("disjunction0"); d == nil {
		ctx.err("disjunction0: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{a b c}"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "a b c"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("disjunction00"); d == nil {
		ctx.err("disjunction00: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "x"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("disjunction01"); d == nil {
		ctx.err("disjunction01: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x{$1}"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d, defExpand1, v.Position(), []string{"a","b","c"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "xa xb xc"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("disjunction02"); d == nil {
		ctx.err("disjunction02: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x{&(something)}"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("disjunction03"); d == nil {
		ctx.err("disjunction03: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x{a b c}"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	} else if s, t := v.string(src(ctx,d)), "xa xb xc"; s != t {
		ctx.err("%v: %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("disjunction1"); d == nil {
		ctx.err("disjunction1: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := l.String(), `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s := l.string(src(ctx,d)); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	if d := ctx.def("disjunction2"); d == nil {
		ctx.err("disjunction2: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := l.String(), `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s := l.string(src(ctx,d)); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	var (
		glob1 = ctx.val("glob1")
		glob2 = ctx.val("glob2")
		glob3 = ctx.val("glob3")
		regex1 = ctx.val("regex1")
		regex2 = ctx.val("regex2")
		regex3 = ctx.val("regex3")
		regex4 = ctx.val("regex4")
		regex5 = ctx.val("regex5")
		regex6 = ctx.val("regex6")
	)

	if glob1 == nil {
		ctx.err("glob1: %v", ctx.def("glob1"))
	} else if glob1.string(ctx) != "*.c" {
		ctx.err("%v", tst{glob1})
	} else if g, y := glob1.(*globpat); !y {
		ctx.err("%v", tst{glob1})
	} else if g.String() != "*.c" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if g.string(ctx) != "*.c" {
		ctx.err("%v", tst{g})
	}

	if glob2 == nil {
		ctx.err("glob2")
	} else if glob2.string(ctx) != "**.c" {
		ctx.err("%v", tst{glob2})
	} else if g, y := glob2.(*globpat); !y {
		ctx.err("%v", tst{glob2})
	} else if g.String() != "**.c" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if g.string(ctx) != "**.c" {
		ctx.err("%v", tst{g})
	}

	if glob3 == nil {
		ctx.err("glob3")
	} else if g, y := glob3.(*globpat); !y {
		ctx.err("%v", tst{glob3})
	} else if g.String() != "{=glob x*z?.c}" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if g.string(ctx) != "x*z?.c" {
		ctx.err("%v", tst{g})
	}

	if regex1.string(ctx) != `x{1}, x{1,}, x{1,2}, x{5}?, x{2,}?, x{2,8}? \p{Greek}, \P{Greek}` {
		ctx.err("regex1 is wrong: %v", regex1)
	}

	if regex2.string(ctx) != `(re) (?P<name>re) (?:re) (?im) (?sU:re) \x{10ffff} \x1f \123 \* \. \? \$` {
		ctx.err("regex2 is wrong: %v", regex2)
	}

	if regex3.string(ctx) != `[[:xdigit:]]*, [^[:alpha:]], [^xyz] [a-z] \A \B \b \Q**??^:[]{}\E \^ \z` {
		ctx.err("regex3 is wrong: %v", regex3)
	}

	if regex4.string(ctx) != `fo{2}\.c` {
		ctx.err("regex4 is wrong: %v", regex4)
	}

	if regex5.string(ctx) != `fo{2}/bar\.c` {
		ctx.err("regex5 is wrong: %v", regex5)
	}

	if regex6.string(ctx) != `fo{2}(/o{2}){3}/bar\.c` {
		ctx.err("regex6 is wrong: %v", regex6)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.string(src(ctx,d)) != "foo.c" {
		ctx.err("%v", tst{v})
	} else if a, b, c := glob1.match(ctx, v); !a {
		ctx.err("match(%v, %v): %v %v %v", glob1, v, a, b, c)
	} else if a, b, c := glob2.match(ctx, v); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, v, a, b, c)
	} else if a, b, c := regex4.match(ctx, v); !a {
		ctx.err("match(%v, %v): %v %v %v", regex4, v, a, b, c)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo/bar.c"; s != t {
		ctx.err("%v : %s != %s", tst{v}, s, t)
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", tst{v}, s, t)
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if len(p.elems) != 2 {
		ctx.err("%v: %v: %v", typeof(v), v, p.elems)
	} else if a, b, c := glob1.match(ctx, v); a == true {
		if false { ctx.err("match(%v, %v): %v %v %v", glob1, v, a, b, c) }
	} else if a, b, c := glob2.match(ctx, v); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, v, a, b, c)
	} else if a, b, c := regex5.match(ctx, v); !a {
		ctx.err("match(%v, %v): %v %v %v", regex5, v, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regex5, v, a, b, c)
	} else if s != v.string(src(ctx,d)) {
		ctx.err("match(%v, %v): %v %v %v", regex5, v, a, b, c)
	} else if len(c) != 0 {
		ctx.err("match(%v, %v): %v %v %v", regex5, v, a, b, c)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.string(src(ctx,d)) != "foo/oo/oo/oo/bar.c" {
		ctx.err("%v", tst{v})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if len(p.elems) != 5 {
		ctx.err("%v: %v: %v", typeof(v), v, p.elems)
	} else if a, b, c := glob1.match(ctx, v); a == true {
		if false { ctx.err("match(%v, %v): %v %v %v", glob1, v, a, b, c) }
	} else if a, b, c := glob2.match(ctx, v); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, v, a, b, c)
	} else if a, b, c := regex6.match(ctx, v); !a {
		ctx.err("match(%v, %v): %v %v %v", regex6, v, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regex6, v, a, b, c)
	} else if s != v.string(src(ctx,d)) {
		ctx.err("match(%v, %v): %v %v %v", regex6, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("match(%v, %v): %v %v %v", regex6, v, a, b, c)
	} else if c[0] != "/oo" {
		ctx.err("match(%v, %v): %v %v %v", regex6, v, a, b, c)
	}

	// TODO: test glob.stencil(...)

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "a\\,b\\,c,x\\,y\\,z"; s != t {
		ctx.err("%v : %v", tst{v}, v)
	} else if s, t := v.string(src(ctx,d)), "a,b,c,x,y,z"; s != t {
		ctx.err("%v : %v", tst{v}, v)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("v5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), `"a,b,c x,y,z"`; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	// TODO: test regexp.stencil(...)

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "http://extbit.io/help"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://extbit.com"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://extbit.com?foo=x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val9"); d == nil {
		ctx.err("val9")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://extbit.com?foo=x&bar=y#foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://ext.pub?foo=x+y+z&bar=x%20y%20z#foo+bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def(`configure.types."atomic.h"`); d == nil {
		ctx.err(`configure.types."atomic.h"`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.string(src(ctx,d)), "atomic_bool"; s != t {
		ctx.err("%s != %s; %v", s, t, v)
	}

	if d := ctx.def(`configure.types.<atomic.h>`); d == nil {
		ctx.err(`configure.types.<atomic.h>`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.string(src(ctx,d)), "atomic_bool"; s != t {
		ctx.err("%s != %s; %v", s, t, v)
	}

	if d := ctx.def(`configure.types`); d == nil {
		ctx.err(`configure.types`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if t := v.string(src(ctx,d)); t == "" {
		ctx.err("%s; %v", t, v)
	} else if s := `- configure.types.<atomic.h> <atomic.h> atomic.h,`; !strings.Contains(t, s) {
		ctx.err("%s, %s; %v", s, t, v)
	} else if s := `- configure.types."atomic.h" "atomic.h" , atomic.h`; !strings.Contains(t, s) {
		ctx.err("%s, %s; %v", s, t, v)
	}

	if d := ctx.def("conf0"); d == nil {
		ctx.err("conf0")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.string(src(ctx,d)), "conf0"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("conf1"); d == nil {
		ctx.err("conf1")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.string(src(ctx,d)), "foo.o foo-x.o foo-x-y.o foo-x-y-z.o foobar.o"; s != t {
		ctx.err("%s != %s : %s : %v", s, t, v, tst{v})
	}

	s := "foo.o foo . o "
	s += "foo-x.o foo -x -x x . o "
	s += "foo-x-y.o foo -x-y -y y . o "
	s += "foo-x-y-z.o foo -x-y-z -z z . o "
	s += "foobar.o foobar . o"
	if d := ctx.def("conf2"); d == nil {
		ctx.err("conf2")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if x, y := v.(*list); !y {
		ctx.err("%v %v", typeof(v), v)
	} else if x.len() != 5*7 {
		ctx.err("%d, %v", x.len(), x)
	} else if t := x.string(src(ctx,d)); s != t {
		ctx.err("%s != %s; %v", s, t, x)
	}

	if d := ctx.def("conf3"); d == nil {
		ctx.err("conf3")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "conf3, foo bar, foo bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "conf3, foo bar, foo bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}
