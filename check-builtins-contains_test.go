//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__contains(ctx *testcase) {
	var s string

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,d), v); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	}
}

func test__contains2(ctx *testcase) {
	var s string

	s = "val"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "${foo}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "${foo}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.y"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.z"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}
