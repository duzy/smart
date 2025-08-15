//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues7(ctx *testcase) {
	if d := ctx.def(".test.z"); d == nil {
		ctx.err(".test.z")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "z-$1-$2"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test.y"); d == nil {
		ctx.err(".test.y")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.z y$1,y$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test.x"); d == nil {
		ctx.err(".test.x")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.y x$1,x$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x $1,$2)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "z-yxa-yxb", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}
