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
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo_ab-a-b"; s != t {
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
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 $1$2$3 10 2 $(&(.test.x) $1$1,$2$2) 20 3 &(&(.test.x) $1$2,$2$1) 30 4 $3 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 10 2 20 3 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 abc 10 2 $(&(.test.x) aa,bb) 20 3 &(&(.test.x) ab,ba) 30 4 c 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 10 2 20 3 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 abc 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba 30 4 c 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 10 2 20 3 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 $(&(.test.x) aa,bb) 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 abc 10 2 foo_ab-aa-bb 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 {} 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 abc 10 2 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 $(&(.test.x) aa,bb) 20 3 &(&(.test.x) ab,ba) 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 abc 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 foo_ba-bb-aa 20 3 foo_ba-ba-ab 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 abc 10 2 foo_ba-bb-aa 20 3 foo_ba-ba-ab 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.s0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 11 2 {} 21 {} s0"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 11 2 21 s0"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.s1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 11 2 {} 21 {} s0 s1"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 11 2 21 s0 s1"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t3"
	d = ctx.def(s)
	if d == nil || d.value == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ba-yy-xx 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ba-yy-xx 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ba-yy-xx 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ba-yy-xx 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ba-yy-xx 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ba-yy-xx 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.0 x,y) . $3"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 3 &(&(.test.x) xy,yx) 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 3 &(&(.test.x) xy,yx) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 3 &(&(.test.x) xy,yx) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := v.string(src(ctx,d)), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}
}
