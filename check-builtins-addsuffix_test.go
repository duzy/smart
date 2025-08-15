//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__addsuffix(ctx *testcase) {
	var s string

	s = "val1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addsuffix =xxx,foo)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(src(ctx,d)), "foo=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	s = "val2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addsuffix =xxx,foo bar)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(src(ctx,d)), "foo=xxx bar=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx bar=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	s = "val5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addsuffix =xxx,&(.test.$1))"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{&(.test.$1)}=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}
}
