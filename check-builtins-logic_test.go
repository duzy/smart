//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__logic(ctx *testcase) {
	s := "val1"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or &(none),a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
		// ...
		// ...
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(and $1,$2,$3)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,d), v); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "c"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "(variant/$(or $(base &(variant)),bootstrap))"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := expand(_final(src(ctx,d)),v), "(variant/bootstrap)"; s.String() != t {
		ctx.err("%s != %s | %v", s, t, tst{s})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "(variant/bootstrap)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "variant/bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or $(base &(variant)),bootstrap)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(base $(or &(variant),bootstrap))"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}
