//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues9(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x .$1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}
