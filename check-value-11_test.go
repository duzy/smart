//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues11(ctx *testcase) {
	if d := ctx.def(".test.0"); d == nil {
		ctx.err(".test.0")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test$1)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1, []string{".v1",".v2"}); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "&(.test.v1) &(.test.v2)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test{})"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}
}
