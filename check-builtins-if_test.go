//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__if(ctx *testcase) {
	var s string

	s = "x1"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(if {=yes},yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x2"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(if {=no},yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x3"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x4"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x5"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x6"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x7"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{11:8 {11:25:word no}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x8"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{12:8 {12:25:word no}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	s = "x81"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{20:9 {20:22:word yes}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x9"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x10"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(ifarg 1,yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x11"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x12"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}
