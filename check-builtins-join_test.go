//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__join(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(join foo bar xx yy zz,-)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(target.arch)-&(target.vendor)-&(target.os)-&(target.abi)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foo-bar--0"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{{&(target.arch) &(XXX) &(target.vendor) &(target.os) &(target.abi)}-}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foo-bar-0"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}
