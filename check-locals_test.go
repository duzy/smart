//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testLocals(ctx *testcase) {
	s := "foo"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{3:8:word foobar}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{7:9 {3:8:word foobar}}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{9:9 {8:9:word x}}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{13:9 {3:8:word foobar}}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}
}
