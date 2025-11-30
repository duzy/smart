//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues21(ctx *testcase) {
	s := ".test.ab"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-$1-$2"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab--"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.ba"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-$2-$1"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba--"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 3 foo_ba-{}{}-{}{} 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20 3 foo_ba-- 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.t1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t3"
	d = ctx.def(s)
	if d == nil || d.value == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test x,y) . $3"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}
}
