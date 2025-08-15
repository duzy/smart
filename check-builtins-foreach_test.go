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
	} else if s, t := d.value.string(ctx), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=word a} {=word b} {=word c}} {=word X} {=word Y} {=word Z} {=list {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word h}}} {=word a}} {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word h}}} {=word b}} {=compound {=closure {=compound {=punct .} {=word test} {=punct .} {=word h}}} {=word c}}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "x a b c X Y Z &(.test.h)a &(.test.h)b &(.test.h)c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=word a} {=word b} {=word c}} {=word X} {=word Y} {=word Z} {=list {=flag {=word a}} {=flag {=word b}} {=flag {=word c}}}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", j, s)
	} else if s0, t := d.value.String(), "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"; s0 != t {
		ctx.err("%s != %s : %v", s0, t, tst{d.value})
	} else if s1, t := d.value.string(ctx), "x xq xp"; s1 != t {
		ctx.err("%s != %s : %v", s1, t, tst{d.value})
	} else if x, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if x.len() != 2 {
		ctx.err("%d %v", x.len(), tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=disjunction {=closure {=def .test.h}}} {=word a}} {=compound {=word x} {=disjunction {=closure {=def .test.h}}} {=word b}} {=compound {=word x} {=disjunction {=closure {=def .test.h}}} {=word c}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=flag {=word a}}} {=compound {=word x} {=flag {=word b}}} {=compound {=word x} {=flag {=word c}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.21"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
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
		ctx.err("%v → %v (%v)", tst{v}, d, v.cmp(ctx, d.value))
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.22"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x x-"; s != t {
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
	} else if s, t := ts(v), "{=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word a}} {=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word b}} {=compound {=word x} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word xx}}}} {=word c}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"aa", "bb", "cc"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=compound {=word x} {=word q}} {=compound {=word x} {=word p}} {=compound {=word x} {=word aa}} {=compound {=word x} {=word bb}} {=compound {=word x} {=word cc}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp xaa xbb xcc"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x $(foreach $1,&(.test.$_)$1{}88) y $(foreach $1,$(closure .test.$_)$1{}99) z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s != v.String() {
		ctx.err("%s != %s : %v", v, s, tst{v})
	} else if s, t := v.string(ctx), "x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=word y} {=word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 3 {
		ctx.err("%d %v", x.len(), v)
	} else if v := ctx.val(d, defExpand1, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=word zz}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=word zz}}} {=word y} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=decimal 99}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=decimal 99}}} {=word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x {&(.test.foo)}foo barzz {&(.test.bar)}foo barzz y {&(.test.foo)}foo bar99 {&(.test.bar)}foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "x barzz barzz y bar99 bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d, defExpand2, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=word x} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=word zz}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=word zz}}} {=word y} {=list {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word foo}}}} {=word foo}} {=compound {=word bar} {=decimal 99}} {=compound {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word bar}}}} {=word foo}} {=compound {=word bar} {=decimal 99}}} {=word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x {&(.test.foo)}foo barzz {&(.test.bar)}foo barzz y {&(.test.foo)}foo bar99 {&(.test.bar)}foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.string(ctx), "x foo barzz foo barzz y foo bar99 foo bar99 z"; s != t {
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
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.$(or $4,$3)) &(.test.y.$(or $4,$3))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", ""); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "", "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", []string{}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", []string{}, "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x do.smart)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if f, y := v.(*file); !y {
		ctx.err("%v : %v", v, tst{v})
	} else if f.name != "do.smart" {
		ctx.err("%v : %v", v, tst{v})
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.6"
    if d := ctx.def(s); d == nil || d.value == nil {
		ctx.err("%s: %v", s, d)
	} else if v := ctx.val(d, defExpand1, "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "- x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.7"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach a $(foreach $1,&(.test.z)$_{}zz) b,x$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa x&(.test.z)y1zz x&(.test.z)y2zz x&(.test.z)y3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 5 {
		ctx.err("%v, %v", x.len(), v)
	} else if _, y = x.elems[1].(*list); !y && false {
		ctx.err("%v", tst{x.elems[1]})
	} else if v := ctx.val(d, defExpand2, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
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
	} else if s, t := v.string(ctx), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if f, y := v.(*file); !y || f == nil {
		ctx.err("%v", tst{v})
	}
}

func test__foreach1(ctx *testcase) {
	s := ".test.1"
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
	} else if s, t := v.string(ctx), "foo test-foo test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "xxx"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), `&(.test.s) test-foo &(.test.a) &(.test~foo.a) &(.test.B) &(.test~foo.B) &(.test.b) &(.test~foo.b)`; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo test-foo test-a test-foo-a test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

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
	} else if s, t := v.String(), "$(.test.foreach.b) $(foreach &(.test.foreach.x) $(foreach $1 $2,$(closure .test.foreach.x.$_)),-x$_)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.b)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
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
	} else if s, t := v.String(), "bx by bz baxx bayy -x&(.test.foreach.x)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.c $1,4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx by bz baxx bayy -x&(.test.foreach.x) -x&(.test.foreach.x.4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "3"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx by bz baxx bayy -x&(.test.foreach.x) -x&(.test.foreach.x.3) -x&(.test.foreach.x.4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "bx by bz baxx bayy -xvw -xV -xW"; s != t {
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
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "1", "2"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	if v := ctx.val(".test.foreach.d", defExpand1, "1", "2"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(".test.foreach.d", defExpand1, "1", "2", "a", "b"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "-xa -xb -ya -yb"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

func test__foreach3(ctx *testcase) {
	var s string

	s = ".test.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,$_$3)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{d.value})
	} else {
		if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "acc bcc"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "", []string{"1", "2", "3"}, "x"); v == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "1x 2x 3x"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "a b"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "ax bx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := v.string(ctx); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.x"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := v.String(), "$(foreach $1 $2,$(addprefix std={},&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.a)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", nil); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.a)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.b)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=&(.test.a)? std=&(.test.b)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std=&(.test.a)? std=&(.test.if.x)? std=&(.test.if.y)? std=&(.test.if.z)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std=$(foreach $1 $2,$_$3)? std=xxx std=yyy std=&(.test.if.z)?"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.y"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
		if v := ctx.val(d, defExpand1, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))? $(if &(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.y),std=&(.test.if.y))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.if.x),std=&(.test.if.x))? $(if &(.test.if.y),std=&(.test.if.y))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "zzz", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(if &(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.z"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := d.value.String(), "$(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
		if v := ctx.val(d, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if l, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if l.len() != 2 {
			ctx.err("%v ; %d", tst{v}, l.len())
		} else if x, y := l.elems[0].(cond); !y {
			ctx.err("%v", tst{l.elems[0]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.String(), "std=&(.test.if.x)?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*delegate); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.String(), "$(if $(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.String(), "std=&(.test.if.x)? $(if $(.test.{$2}),std=&(.test.{$2}))?"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.Value.String(), "std=&(.test.if.x)"; s != t {
			ctx.err("%v → %s != %s", tst{x.Value}, t, s)
		} else if s, t := v.String(), "std=&(.test.if.x)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if x, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := x.Value.String(), "std=&(.test.if.y)"; s != t {
			ctx.err("%v → %s != %s", tst{x.Value}, t, s)
		} else if s, t := v.String(), "std=&(.test.if.y)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if l, y := v.(*list); !y {
			ctx.err("%v", tst{v})
		} else if len(l.elems) != 2 {
			ctx.err("%v", l.elems)
		} else if x, y := l.elems[0].(cond); !y {
			ctx.err("%v", tst{l.elems[0]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := l.elems[0].String(), "std=&(.test.if.x)?"; s != t {
			ctx.err("%v → %s != %s", l.elems[0], t, s)
		} else if s, t := l.elems[0].string(ctx), "std=xxx"; s != t {
			ctx.err("%v → %s != %s", l.elems[0], t, s)
		} else if x, y := l.elems[1].(cond); !y {
			ctx.err("%v", tst{l.elems[1]})
		} else if _, y := x.Value.(*pair); !y {
			ctx.err("%v", tst{x.Value})
		} else if s, t := l.elems[1].String(), "std=&(.test.if.y)?"; s != t {
			ctx.err("%v → %s != %s", l.elems[1], t, s)
		} else if s, t := l.elems[1].string(ctx), "std=yyy"; s != t {
			ctx.err("%v → %s != %s", l.elems[1], t, s)
		} else if s, t := v.String(), "std=&(.test.if.x)? std=&(.test.if.y)?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "std=xxx std=yyy"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		}
		if v := ctx.val(d, "zzz", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if _, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "$(if $(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d})
		} else if _, y := v.(cond); !y {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "$(if $(.test.zzz),std=&(.test.zzz))?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
	}
}

func test__foreach4(ctx *testcase) {
	s := ".test.1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if len(l.elems) != 2 {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[0].(*compound); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(*compound); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := v.String(), "Xxa Xxb"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "Xxa Xxb"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if x, y := d.value.(*delegate); !y {
		ctx.err("%v", tst{d.value})
	} else if t, y := x.x.(*builtin); !y {
		ctx.err("%v", tst{x.x})
	} else if t.name != "foreach" {
		ctx.err("%v", tst{x.x})
	} else if s, t := d.value.String(), "$(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if len(l.elems) != 3 {
		ctx.err("%v", tst{l.elems[0]})
	} else if x, y := l.elems[0].(cond); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if z, y := x.Value.(*compound); !y {
		ctx.err("%v", tst{x.Value})
	} else if s, t := z.String(), "X{&(.test.xa)}"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if s, t := "X{~1~ $1 $2 $(foreach $1 $2,x-$_) $(foreach $(foreach $1 $2,z-$_),~1~$_)}", z.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if s, t := z.string(ctx), "X~1~"; s != t {
		ctx.err("%v : %v → %s != %s", tst{z}, z, t, s)
	} else if t := x.string(ctx); t != "" {
		ctx.err("%v → %s", tst{x}, t)
	} else if x := ctx.val(z, defExpand2); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X{~1~ $1 $2 $(foreach $1 $2,x-$_) $(foreach $(foreach $1 $2,z-$_),~1~$_)}"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X~1~"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := z.String(), "X{&(.test.xb)}"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if s, t := z.string(ctx), "X~2~"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if t := x.string(ctx); t != "" {
		ctx.err("%v → %s", tst{x}, t)
	} else if x := ctx.val(z, defExpand2); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X{~2~ $1 $2 $(foreach $1 $2,x-$_) $(foreach $(foreach $1 $2,z-$_),~2~$_)}"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X~2~"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
		// hold line
		// hold line
		// hold line
		// hold line
	} else if s, t := z.String(), "X{&(.test.xc)}"; s != t {
		ctx.err("%v → %s != %s", tst{z}, t, s)
	} else if t := z.string(ctx); t != "X" {
		ctx.err("%v → %s", tst{z}, t)
	} else if t := x.string(ctx); t != "" {
		ctx.err("%v → %s", tst{x}, t)
	} else if x := ctx.val(z, defExpand2, "xx", "yy"); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X{$(foreach $(foreach $(foreach $(foreach $1 $2,~$_),~$_),~$_),~$_)}"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if x = ctx.val(x, defExpand2, "xx", "yy"); x == nil {
		ctx.err("%v", tst{z})
	} else if s, t := x.String(), "X~~~~xx X~~~~yy"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "X~~~~xx X~~~~yy"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := v.String(), "X{&(.test.xa)}? X{&(.test.xb)}? X{&(.test.xc)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d.value})
	} else if s, t := v.String(), "X{&(.test.xa)}? X{&(.test.xb)}? X{&(.test.xc)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $(foreach $(foreach $1 $2,&(.test.x$_)),X$_),Y$_)"; s != t {
		ctx.err("%v : %v → %s != %s", d.value, tst{d.value}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
		// hold line
		// hold line
		// hold line
		// hold line
	} else if s, t := v.String(), "YX{&(.test.xa)}? YX{&(.test.xb)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $(foreach $1 $2,x$_),X$_) $(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
		// hold line
		// hold line
		// hold line
		// hold line
	} else if s, t := v.String(), "Xxa Xxb X{&(.test.xa)}? X{&(.test.xb)}?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), `Xxa Xxb`; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func test__foreach5(ctx *testcase) {
	var s string

	s = ".test.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "o"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.o.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.o.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a~ $(foreach $(foreach $1 $2,$(.test.x.o.$_)),-ao$_) ~a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s := "a~ -aox.o.a -aox.o.b ~a"; s != v.String() {
		ctx.err("%v != %s", v, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_) ~b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ -bo{&(.test.x.o.a)}? -bo{&(.test.x.o.b)}? ~b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.0"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $2,&(.test.x.$_) &(.test.x.&(.test.o).$_))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if x := ctx.val(v, "a", "b", "c"); x == nil {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if x := ctx.val(v, defExpand2, "a", "b", "c"); x == nil {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "a~ $(foreach $(foreach $1 $2,$(.test.x.o.$_)),-ao$_)? ~a x.o.a b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_)? ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{x}, t, s)
	}

	s = ".test.x.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x $1)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else {
		if v := ctx.val(d); v == nil {
			ctx.err("%v", tst{d})
		} else if t := v.String(); s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v → %s", tst{v}, t)
		}
		if v := ctx.val(d, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.{$2})? &(.test.x.&(.test.o).{$2})?"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if s, t := v.string(ctx), "a~ ~a x.o.a"; s != t {
			ctx.err("%v → %s != %s", tst{v}, t, s)
		} else if x := ctx.val(v, nil, []string{"b", "c"}); x == nil {
			ctx.err("%v", tst{v})
		} else if s, t := x.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? {&(.test.x.b)}? {&(.test.x.c)}? {&(.test.x.&(.test.o).b)}? {&(.test.x.&(.test.o).c)}?"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		} else if s, t := x.string(ctx), "a~ ~a x.o.a b~ ~b ~c~ x.o.b x.o.c"; s != t {
			ctx.err("%v → %s != %s", tst{x}, t, s)
		}
	}

	s = ".test.x.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $2,&(.test.x.$_) &(.test.x.&(.test.o).$_))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.{$2})? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.{$2})? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", []string{"b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a)? &(.test.x.&(.test.o).a)? &(.test.x.b)? &(.test.x.&(.test.o).b)? &(.test.x.c)? &(.test.x.&(.test.o).c)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ ~a x.o.a b~ ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if x := v.expand(_final(ctx)); x == nil {
		ctx.err("%v", tst{v})
	} else if l, y := x.(*list); !y {
		ctx.err("%v", tst{x})
	} else if elems := merge(l.elems...); l.len() != 4 || len(elems) != 7 {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v ; %d, %d", tst{x}, l.len(), len(elems))
	} else if _, y := elems[2].(cond); !y {
		ctx.err("%v", tst{elems[2]})
	} else if s, t := x.String(), "a~ -aox.o.a -ao{$(.test.x.o.{$2})}? ~a x.o.a &(.test.x.{$2} a,$2)? &(.test.x.o.{$2})?"; s != t {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := x.string(ctx), "a~ -aox.o.a ~a x.o.a"; s != t {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v → %s != %s", tst{x}, t, s)
	} else if s, t := v.String(), "&(.test.x.a a,$2)? &(.test.x.&(.test.o).a)? &(.test.x.{$2} a,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a b c,$2)? &(.test.x.&(.test.o).a)? &(.test.x.b a b c,$2)? &(.test.x.&(.test.o).b)? &(.test.x.c a b c,$2)? &(.test.x.&(.test.o).c)? &(.test.x.{$2} a b c,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b -aox.o.c ~a x.o.a b~ -box.o.a -box.o.b -box.o.c ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y ,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1,$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(ctx); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.10"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "&(.test.x.a a,$2)? &(.test.x.&(.test.o).a)? &(.test.x.{$2} a,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,$2)? &(.test.x.&(.test.o).a)? &(.test.x.{$2} a,$2)? &(.test.x.&(.test.o).{$2})?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d.value); v == nil {
		ctx.err("%v", tst{d})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.11"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x.12"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b)? &(.test.x.&(.test.o).a)? &(.test.x.b a,b)? &(.test.x.&(.test.o).b)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}
