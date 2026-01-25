//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValueCache(ctx *testcase) {
	p := _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}}`; s != t {
		note(ctx, "%s", t)
		ctx.err("%v", &p.filemap)
	}

	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if s, t := __string(src(ctx,d), d.value), "**.c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if s, t := __string(src(ctx,d), d.value), "**.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if s, t := __string(src(ctx,d), d.value), "foo.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if s, t := __string(src(ctx,d), d.value), "foo.o"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if s, t := __string(src(ctx,d), d.value), ".deps/xx/yy/zzzzzzzzzz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if s, t := __string(src(ctx,d), d.value), "foo/*.c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if s, t := __string(src(ctx,d), d.value), "foo/*.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}
}
