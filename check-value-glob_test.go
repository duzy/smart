//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testGlob(ctx *testcase) {
	if d := ctx.def("pat1.0"); d == nil {
		ctx.err("pat1.0: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("pat1.0: %v", _project(ctx))
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if p.len() != 2 {
		ctx.err("%v", p)
	} else if s, t := p.String(), ".test/x**y"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := ts(v), "{=path {=compound {4:12:punct .} {4:13:word test}} {=glob {4:18:word x} {4:19:meta **} {4:21:word y}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if d1 := ctx.def("pat1.1"); d1 == nil {
		ctx.err("pat1.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat1.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xxx-yyy [xx-yy]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d2 := ctx.def("pat1.2"); d2 == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d3 := ctx.def("pat1.3"); d3 == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xxx-yyx/y [xx-yyx/]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d4 := ctx.def("pat1.4"); d4 == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if v := d4.value; v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d5 := ctx.def("pat1.5"); d5 == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if v := d5.value; v == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xxx/a/b/c/yyy [xx/a/b/c/yy]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d6 := ctx.def("pat1.6"); d6 == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if v := d6.value; v == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/x/xx-yy/y [/xx-yy/]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat2.0"); d == nil {
		ctx.err("pat2.0: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("pat2.0: %v", _project(ctx))
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if s, t := p.String(), ".test/x**y/z"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := ts(v), "{=path {=compound {12:12:punct .} {12:13:word test}} {=glob {12:18:word x} {12:19:meta **} {12:21:word y}} {12:23:word z}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if d1 := ctx.def("pat2.1"); d1 == nil {
		ctx.err("pat2.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat2.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xxx-yyy/z [xx-yy]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d2 := ctx.def("pat2.2"); d2 == nil {
		ctx.err("pat2.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat2.2: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xxx/a/b/c/yyy/z [xx/a/b/c/yy]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat3.0"); d == nil {
		ctx.err("pat3.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if p, y := val.(*path); !y {
		ctx.err("%v : %v", tst{val}, val)
	} else if val.String() != ".test/x**y/x**" {
		ctx.err("%v", tst{val})
	} else if __string(src(ctx,d),val) != ".test/x**y/x**" {
		ctx.err("%v: %s", tst{val}, __string(src(ctx,d),val))
	} else if d1 := ctx.def("pat3.1"); d1 == nil {
		ctx.err("pat3.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat3.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xaaa/bbb/ccc/y/xxx/xx [aaa/bbb/ccc/ xx/xx]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d2 := ctx.def("pat3.2"); d2 == nil {
		ctx.err("pat3.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat3.2: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xaabbccy/xabc [aabbcc abc]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat4.0"); d == nil {
		ctx.err("pat4.0: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("pat4.0: %v", _project(ctx))
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if s, t := v.String(), ".test/x**y/x**y"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := ts(v), "{=path {=compound {18:12:punct .} {18:13:word test}} {=glob {18:18:word x} {18:19:meta **} {18:21:word y}} {=glob {18:23:word x} {18:24:meta **} {18:26:word y}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if d1 := ctx.def("pat4.1"); d1 == nil {
		ctx.err("pat4.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat4.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xaa/bb/ccy/xaa/bb/ccy [aa/bb/cc aa/bb/cc]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d2 := ctx.def("pat4.2"); d == nil {
		ctx.err("pat4.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat4.2: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/xaaay/x/aaa/y [aaa /aaa/]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat5.0"); d == nil {
		ctx.err("pat5.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat5.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/x**/**y"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {21:12:punct .} {21:13:word test}} {=glob {21:18:word x} {21:19:meta **}} {=glob {21:22:meta **} {21:24:word y}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else {
		match(ctx, p, ctx.def("pat5.1").value)
		match(ctx, p, ctx.def("pat5.2").value)
	}

	if d := ctx.def("pat6.0"); d == nil {
		ctx.err("pat6.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat6.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/**y/**y"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {24:12:punct .} {24:13:word test}} {=glob {24:18:meta **} {24:20:word y}} {=glob {24:22:meta **} {24:24:word y}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else {
		match(ctx, p, ctx.def("pat6.1").value)
	}

	if d := ctx.def("pat7.0"); d == nil {
		ctx.err("pat7.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat7.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/**y/**y/z"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {26:12:punct .} {26:13:word test}} {=glob {26:18:meta **} {26:20:word y}} {=glob {26:22:meta **} {26:24:word y}} {26:26:word z}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if d1 := ctx.def("pat7.1"); d1 == nil {
		ctx.err("pat7.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat7.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/a/b/cy/a/b/c/y/z [a/b/c a/b/c/]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat8.0"); d == nil {
		ctx.err("pat8.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat8.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/**/**z"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {28:12:punct .} {28:13:word test}} {=glob {28:18:meta **}} {=glob {28:21:meta **} {28:23:word z}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if d1 := ctx.def("pat8.1"); d1 == nil {
		ctx.err("pat8.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat8.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/a/b/c/xyz [a/b/c xy]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat10.0"); d == nil {
		ctx.err("pat10.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat10.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if val.String() != ".test/*.h" {
		ctx.err("%v", tst{val})
	} else if d1 := ctx.def("pat10.1"); d1 == nil {
		ctx.err("pat10.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat10.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/a.h [a]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d2 := ctx.def("pat10.2"); d2 == nil {
		ctx.err("pat10.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat10.2: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d3 := ctx.def("pat10.3"); d3 == nil {
		ctx.err("pat10.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat10.3: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat11.0"); d == nil {
		ctx.err("pat11.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat11.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if val.String() != ".test/*/*.h" {
		ctx.err("%v", tst{val})
	} else if d1 := ctx.def("pat11.1"); d1 == nil {
		ctx.err("pat11.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat11.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d2 := ctx.def("pat11.2"); d2 == nil {
		ctx.err("pat11.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat11.2: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/a/b.h [a b]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d3 := ctx.def("pat11.3"); d3 == nil {
		ctx.err("pat11.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat11.3: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat12.0"); d == nil {
		ctx.err("pat12.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat12.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/*/*/*.h"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {38:12:punct .} {38:13:word test}} {=glob {38:18:meta *}} {=glob {38:20:meta *}} {=glob {38:22:meta *} {38:23:punct .} {38:24:word h}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if d1 := ctx.def("pat12.1"); d1 == nil {
		ctx.err("pat12.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat12.1: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d2 := ctx.def("pat12.2"); d2 == nil {
		ctx.err("pat12.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat12.2: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d3 := ctx.def("pat12.3"); d3 == nil {
		ctx.err("pat12.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat12.3: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "true .test/a/b/c.h [a b c]" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d4 := ctx.def("pat12.4"); d4 == nil {
		ctx.err("pat12.4: %v", _project(ctx))
	} else if v := d4.value; v == nil {
		ctx.err("pat12.4: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	} else if d5 := ctx.def("pat12.5"); d5 == nil {
		ctx.err("pat12.5: %v", _project(ctx))
	} else if v := d5.value; v == nil {
		ctx.err("pat12.5: %v", _project(ctx))
	} else if a, b0, c := match(ctx, p, v); sf("%v %v %v", a, b0, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b0, c)
	}

	if d := ctx.def("pat.0"); d == nil {
		ctx.err("pat.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if __string(src(ctx,d),val) != "**.auto" {
		ctx.err("%v", tst{val})
	} else if d1 := ctx.def("pat13.1"); d1 == nil {
		ctx.err("pat13.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, c := match(ctx, val, v); sf("%v %v %v", a, b, c) != "true .test/a/b/c.auto [.test/a/b/c]" {
		ctx.err("%v %v: %v %v %v", val, v, a, b, c)
	} else if d2 := ctx.def("pat13.2"); d2 == nil {
		ctx.err("pat13.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, c := match(ctx, val, v); sf("%v %v %v", a, b, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", val, v, a, b, c)
	}
}
