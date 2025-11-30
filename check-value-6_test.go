//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues6(ctx *testcase) {
	s := ".test"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "z-y-x-a" {
		ctx.err("%v", d)
	} else if v := ctx.val(d.name, defExpand1, "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "z-y-x-a", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}
