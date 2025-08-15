//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues5(ctx *testcase) {
	s := ".test.0"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x0 $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-a"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-{}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "z-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-{}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "z-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}
