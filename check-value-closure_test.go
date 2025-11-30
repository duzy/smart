//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testClosure(ctx *testcase) {
	s := "foo_pre"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "&(foo.pre)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_pos"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "&(foo.pos)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_z"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "$(&(foo.tail))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "$(&(foo.tail))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
}
