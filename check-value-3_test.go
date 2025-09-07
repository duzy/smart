//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues3(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$1 &(.test) &(.test.5) &(.test.5 x) &(.test.5 x,y) &(.test.5 x,y,z) &(.test aa)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "v----- v-x---- v-x-y--- v-x-y-z-- v----- a v-x---- a v-x-y--- a v-x-y-z-- a aa v----- v-x---- v-x-y--- v-x-y-z--"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "a"); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "a &(.test) &(.test.5) &(.test.5 x) &(.test.5 x,y) &(.test.5 x,y,z) &(.test aa)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "a v----- v-x---- v-x-y--- v-x-y-z-- v----- a v-x---- a v-x-y--- a v-x-y-z-- a aa v----- v-x---- v-x-y--- v-x-y-z--"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}
