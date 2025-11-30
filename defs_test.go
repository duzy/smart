//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testDefs0(ctx *testcase) {
	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "a b c" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "x a b" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if d.o != defExpand2 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "x a b" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if d.o != defExpand2 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "a b c" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}
}
