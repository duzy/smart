//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__foreach4(ctx *testcase) {
	s := ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{d.value})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "{}"; s != t {
		ctx.err("%v : %v → %s != %s", d.value, tst{d.value}, t, s)
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $(foreach $1 $2,x$_),X$_) $(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%v : %v → %s != %s", d.value, tst{d.value}, t, s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $(foreach $1 $2,x$_),X$_) $(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,nil), v); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "Xxa Xxb X{&(.test.xa)} X{&(.test.xb)}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), `Xxa Xxb X~1~ X~2~`; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "YX{&(.test.xa)} YX{&(.test.xb)}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), `YX~1~ YX~2~`; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "YX~1~ YX~2~"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}
