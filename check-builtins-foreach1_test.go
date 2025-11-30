//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__foreach1(ctx *testcase) {
	var s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x $1,B b)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s != v.String() {
		ctx.err("%v != %s", tst{v}, s)
	} else if s, t := __string(src(ctx,nil), v), "foo test-foo test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "xxx"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), `&(.test.s) test-foo &(.test.a) &(.test~&(.test.s).a) &(.test.B) &(.test~&(.test.s).B) &(.test.b) &(.test~&(.test.s).b)`; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "foo test-foo test-a test-foo-a test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}
