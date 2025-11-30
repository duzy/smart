//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__foreach2(ctx *testcase) {
	s := ".test.foreach.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $(foreach $2,a$_),b$_)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.foreach.b"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.foreach.c"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.b) $(foreach &(.test.foreach.x) $(foreach $1 $2,&(.test.foreach.x.$_)),-x$_)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.b)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.c)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.c $1,4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.4)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "3"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.3)} -x{&(.test.foreach.x.4)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw -xV -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.d)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "1", "2"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	if v := ctx.val(".test.foreach.d", defExpand1, "1", "2"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(".test.foreach.d", defExpand1, "1", "2", "a", "b"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "-xa -xb -ya -yb"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}
