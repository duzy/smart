//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testOptional(ctx *testcase) {
	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{19:10 {1:8:project foo}}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{20:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{21:10 {3:9 {1:8:self foo}}}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{22:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{23:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{24:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{25:10 {25:27 {3:9 {1:8:self foo}}}}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{26:10 {26:27 {3:9 {1:8:self foo}}}}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if d.value == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(d.value), "{27:10 {27:12:null}}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := v.string(src(ctx,d)), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val11"); d == nil {
		ctx.err("val11")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := v.string(src(ctx,d)), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val12"); d == nil {
		ctx.err("val12")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := v.string(src(ctx,d)), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}
}
