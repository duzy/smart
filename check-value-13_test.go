//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues13(ctx *testcase) {
	if d := ctx.def("foo"); d == nil {
		ctx.err("foo")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(-g!foobar)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := "&(-g!foobar)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "not-foobar", v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}
