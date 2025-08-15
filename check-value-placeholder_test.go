//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testPlaceholders(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "_", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 2 3 4 5 6 7 8 9"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach a b c d e f,$_)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "a b c d e f"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 2 3 4 5 6 7 8 9"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "a b c d e f"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, ts(v))
	}

	if d := ctx.def("foo1"); d == nil {
		ctx.err("foo1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val1" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo2"); d == nil {
		ctx.err("foo2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val2" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo3"); d == nil {
		ctx.err("foo3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val3" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo4"); d == nil {
		ctx.err("foo4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val4" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo5"); d == nil {
		ctx.err("foo5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val5" {
		ctx.err("%v", v)
	}
}
