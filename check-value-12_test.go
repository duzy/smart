//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues12(ctx *testcase) {
	if d := ctx.def(".test.0"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.w)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	if d := ctx.def(".test.1"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.{})"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "www"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}
