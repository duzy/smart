//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValues1(ctx *testcase) {
	if d := ctx.def(".test.foo"); d == nil {
		ctx.err(".test.foo")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s := v.string(src(ctx,d)); s != "-foo" {
		ctx.err("%v → %s", v, s)
	} else if s = v.String(); s != "-foo" {
		ctx.err("%v → %s", v, s)
	}
}
