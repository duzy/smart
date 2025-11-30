//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testDisjunction(ctx *testcase) {
	s := "val1"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo={&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo={&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=bar{&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=bar{&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
}
