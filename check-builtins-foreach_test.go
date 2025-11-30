//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__foreach(ctx *testcase) {
	{
		var c0 = __foreach{}
		var c1 = partial{&c0, nonePart}
		var c2 = __foreach{}
		c0.evocation = &evocation{automatic{Context:ctx}, nil, nil, nil}
		c2.evocation = &evocation{automatic{Context:c1}, nil, nil, nil}
		if cast[*builtinbase](&c0) == nil {
			ctx.err("builtinbase")
		}
		if cast[partial](ctx).Context != nil {
			ctx.err("partial")
		}
		if cast[partial](&c0).Context != nil {
			ctx.err("partial")
		}
		if cast[partial](c1).Context == nil {
			ctx.err("partial")
		}
		if cast[partial](&c2).Context != nil {
			ctx.err("partial")
		}
	}

	var s string
	var test_1_value Value
	var j = _project(ctx)

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", j, s)
	} else if test_1_value = d.value; test_1_value == nil {
		ctx.err("%v", d)
	} else if s, t := d.value.String(), "x $1 $2 $3 $4 $(foreach $1,&(.test.h)$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if s, t := __string(src(ctx,nil), d.value), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x {} {} {} {} {}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {3:12:word x} {=list {3:14 {1:9:word a}} {3:14 {1:9:word b}} {3:14 {1:9:word c}}} {3:17 {1:9:word X}} {3:20 {1:9:word Y}} {3:23 {1:9:word Z}} {=list {4:12 {=compound {4:25:closure {6:9:def .test.h}} {4:35 {4:22 {1:9:word a}}}}} {4:12 {=compound {4:25:closure {6:9:def .test.h}} {4:35 {4:22 {1:9:word b}}}}} {4:12 {=compound {4:25:closure {6:9:def .test.h}} {4:35 {4:22 {1:9:word c}}}}}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "x a b c X Y Z &(.test.h)a &(.test.h)b &(.test.h)c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {3:12:word x} {=list {3:14 {1:9:word a}} {3:14 {1:9:word b}} {3:14 {1:9:word c}}} {3:17 {1:9:word X}} {3:20 {1:9:word Y}} {3:23 {1:9:word Z}} {=list {4:12 {=compound {4:25 {=flag {6:13}}} {4:35 {4:22 {1:9:word a}}}}} {4:12 {=compound {4:25 {=flag {6:13}}} {4:35 {4:22 {1:9:word b}}}}} {4:12 {=compound {4:25 {=flag {6:13}}} {4:35 {4:22 {1:9:word c}}}}}}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", j, s)
	} else if s0, t := d.value.String(), "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"; s0 != t {
		ctx.err("%s != %s : %v", s0, t, tst{d.value})
	} else if s1, t := __string(src(ctx,nil), d.value), "x xq xp"; s1 != t {
		ctx.err("%s != %s : %v", s1, t, tst{d.value})
	} else if x, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if x.len() != 2 {
		ctx.err("%d %v", x.len(), tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {7:12:word x} {=list {8:12 {=compound {8:53:word x} {8:54 {8:22:word q}}}} {8:12 {=compound {8:53:word x} {8:54 {8:24:word p}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39:disjunction {8:39:closure {6:9:def .test.h}}} {8:49 {8:36 {1:9:word a}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39:disjunction {8:39:closure {6:9:def .test.h}}} {8:49 {8:36 {1:9:word b}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39:disjunction {8:39:closure {6:9:def .test.h}}} {8:49 {8:36 {1:9:word c}}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {7:12:word x} {=list {8:12 {=compound {8:53:word x} {8:54 {8:22:word q}}}} {8:12 {=compound {8:53:word x} {8:54 {8:24:word p}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39 {=flag {6:13}}} {8:49 {8:36 {1:9:word a}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39 {=flag {6:13}}} {8:49 {8:36 {1:9:word b}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39 {=flag {6:13}}} {8:49 {8:36 {1:9:word c}}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.21"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {10:13:word x} {=list {11:13 {=compound {11:54:word x} {11:55 {11:23:word q}}}} {11:13 {=compound {11:54:word x} {11:55 {11:25:word p}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v", l.elems)
	} else if s, t := l.elems[0].String(), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[0]})
	} else if s, t := l.elems[1].String(), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t, y := l.elems[1].(*list); !y {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t.len() != 2 {
		ctx.err("%d, %v", t.len(), tst{t})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if !equal(ctx, v, d.value) {
		ctx.err("%v → %v (%v)", tst{v}, d, cmp(ctx, v, d.value))
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.22"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.23"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach q p $(foreach $1,&(.test.xx)$_),x$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {16:12 {=compound {16:54:word x} {16:55 {16:22:word q}}}} {16:12 {=compound {16:54:word x} {16:55 {16:24:word p}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:disjunction {16:39:closure {=compound {16:41:punct .} {16:42:word test} {16:46:punct .} {16:47:word xx}}}} {16:50 {16:36 {1:9:word a}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:disjunction {16:39:closure {=compound {16:41:punct .} {16:42:word test} {16:46:punct .} {16:47:word xx}}}} {16:50 {16:36 {1:9:word b}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:disjunction {16:39:closure {=compound {16:41:punct .} {16:42:word test} {16:46:punct .} {16:47:word xx}}}} {16:50 {16:36 {1:9:word c}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"aa", "bb", "cc"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {16:12 {=compound {16:54:word x} {16:55 {16:22:word q}}}} {16:12 {=compound {16:54:word x} {16:55 {16:24:word p}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:null} {16:50 {16:36 {1:9:word aa}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:null} {16:50 {16:36 {1:9:word bb}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:null} {16:50 {16:36 {1:9:word cc}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp x{}aa x{}bb x{}cc"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xq xp xaa xbb xcc"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x $(foreach $1,&(.test.$_)$1{}88) y $(foreach $1,&(.test.$_)$1{}99) z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s != v.String() {
		ctx.err("%s != %s : %v", v, s, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {18:12:word x} {=list {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word foo}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}} {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word bar}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}}} {19:44:word y} {=list {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word foo}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}} {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word bar}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}}} {20:44:word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x &(.test.foo)⌜foo bar⌟{}88 &(.test.bar)⌜foo bar⌟{}88 y &(.test.foo)⌜foo bar⌟{}99 &(.test.bar)⌜foo bar⌟{}99 z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x foo bar88 foo bar88 y foo bar99 foo bar99 z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 5 {
		ctx.err("%d %v", x.len(), v)
	} else if v := ctx.val(d, defExpand1, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {18:12:word x} {=list {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word foo}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}} {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word bar}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}}} {19:44:word y} {=list {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word foo}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}} {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word bar}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}}} {20:44:word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), `x &(.test.foo)⌜foo bar⌟{}88 &(.test.bar)⌜foo bar⌟{}88 y &(.test.foo)⌜foo bar⌟{}99 &(.test.bar)⌜foo bar⌟{}99 z`; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "x foo bar88 foo bar88 y foo bar99 foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d, defExpand2, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {18:12:word x} {=list {19:12 {=compound {19:25:null} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}} {19:12 {=compound {19:25:null} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}}} {19:44:word y} {=list {20:12 {=compound {20:25:null} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}} {20:12 {=compound {20:25:null} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}}} {20:44:word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x {}⌜foo bar⌟{}88 {}⌜foo bar⌟{}88 y {}⌜foo bar⌟{}99 {}⌜foo bar⌟{}99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "x foo bar88 foo bar88 y foo bar99 foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	}

	s = ".test.4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,&(.test.$_.$(or $4,$3)))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s != v.String() {
		ctx.err("%s != %s : %v", v, s, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.{}) &(.test.y.{})"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", ""); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "", "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", []string{}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", []string{}, "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x do.smart)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{28:11 {28:30 {29:11 {=file do.smart}}}}" {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.6"
    if d := ctx.def(s); d == nil || d.value == nil {
		ctx.err("%s: %v", s, d)
	} else if v := ctx.val(d, defExpand1, "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{} {} {} {} - x y z {}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "- x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.7"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach a $(foreach $1,&(.test.z)$_{}zz) b,x$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa x{&(.test.z)}y1{}zz x{&(.test.z)}y2{}zz x{&(.test.z)}y3{}zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 5 {
		ctx.err("%v, %v", x.len(), v)
	} else if _, y = x.elems[1].(*list); !y && false {
		ctx.err("%v", tst{x.elems[1]})
	} else if v := ctx.val(d, defExpand2, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa xwy1{}zz xwy2{}zz xwy3{}zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(stat $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "do.smart"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if ts(v) != "{29:11 {=file do.smart}}" {
		ctx.err("%v", tst{v})
	}
}
