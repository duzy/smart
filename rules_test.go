//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

func testRules0(ctx *testcase) {
	if s := "rule0"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", tst{r})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if len(x.elems) != 6 {
		ctx.err("%v", tst{x})
	} else {
		i := 0
		if z, y := x.elems[i].(*delegate); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=delegate {=auto <}}" {
			ctx.err("%v", tst{x.elems[i]})
		} else if a, y := z.x.(*auto); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if a.name != "<" {
			ctx.err("%v", tst{z.x})
		}

		i = 1
		if z, y := x.elems[i].(*delegate); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=delegate {=auto >}}" {
			ctx.err("%v", tst{x.elems[i]})
		} else if a, y := z.x.(*auto); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if a.name != ">" {
			ctx.err("%v", tst{z.x})
		}

		i = 2
		if z, y := x.elems[i].(flag); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=flag {=null}}" {
			ctx.err("%v", tst{x.elems[i]})
		} else if z.Value == nil {
			ctx.err("%v", tst{z})
		} else if !isNull(z.Value) {
			ctx.err("%v", tst{z})
		}

		i = 3
		if z, y := x.elems[i].(*delegate); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=delegate {=auto ARG1}}" {
			ctx.err("%v", tst{x.elems[i]})
		} else if a, y := z.x.(*auto); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if a.name != "ARG1" {
			ctx.err("%v", tst{z.x})
		}

		i = 4
		if z, y := x.elems[i].(*delegate); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=delegate {=auto ARG2}}" {
			ctx.err("%v", tst{x.elems[i]})
		} else if a, y := z.x.(*auto); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if a.name != "ARG2" {
			ctx.err("%v", tst{z.x})
		}

		i = 5
		if z, y := x.elems[i].(*delegate); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=delegate {=auto ARG3}}" {
			ctx.err("%v", tst{x.elems[i]})
		} else if a, y := z.x.(*auto); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if a.name != "ARG3" {
			ctx.err("%v", tst{z.x})
		}

		if ts(v) != "{=list {=delegate {=auto <}} {=delegate {=auto >}} {=flag {=null}} {=delegate {=auto ARG1}} {=delegate {=auto ARG2}} {=delegate {=auto ARG3}}}" {
			ctx.err("%v %v", v, tst{v})
		}

		if v.String() != "$< $> - $(ARG1) $(ARG2) $(ARG3)" {
			ctx.err("%v %v", v, tst{v})
		}

		if s := v.string(ctx); s != "rule1 rule1 -" {
			ctx.err("%v : %s", tst{v}, s)
		}
	}

	if s := "rule1"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", tst{r})
	} else if v.String() == "$(ARG1)" {
		ctx.err("%v", tst{v})
	} else if x, y := v.(*Plain); !y {
		ctx.err("%v", tst{x})
	} else if s := v.string(ctx); s != "" {
		ctx.err("%v", tst{v})
	}
}

func testRules1(ctx *testcase) {
	if s := ".test.foobar"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "fxxbar" {
		ctx.err("%v", tst{v})
	} else if _, y := v.(*bareword); !y {
		ctx.err("%v", tst{v})
	}

	if s := ".test.foobaz"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if v.String() != ".test.fxx" {
		ctx.err("%v", tst{v})
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%v", tst{v})
	}

	if s := ".test.foobay"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if v.String() != ".test.fxx" {
		ctx.err("%v", tst{v})
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%v", tst{v})
	}

	if s := ".test.1"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "fxxbar" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "fxxbar" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	if s := ".test.2"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != ".test.fxx" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	if s := ".test.3"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != ".test.fxx" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	var p = _project(ctx)

	testResolveEntries = true

	if s := ".test.foo"; false {} else
	if v := makeStrlit(_position(ctx), s); v == nil || v.s != s {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p.resolveEntries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	} else if r := p.resolveEntries(ctx.Context, v.s, false); r != nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	}

	if s := "v1"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "'.test.foo'" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if s := v.string(ctx); s != ".test.foo" {
		ctx.err("%v{%v} -> %s", typeof(v), v, s)
	} else if _, y := v.(*strlit); !y {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p.resolveEntries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p.resolveEntries(ctx.Context, v.string(ctx), false); r != nil {
		ctx.err("%v{%v}", typeof(v), v)
	}

	testResolveEntries = false
}

type testShellForStdoutDebugStruct struct {
	s string
	v []Value
}
func testShellForStdoutDebugHook(ctx Context, s string, v []Value, i interface{}) {
	t := i.(*testShellForStdoutDebugStruct)
	t.v = append(t.v, v...)
	t.s += s
}
func testShellForStdout(ctx testcase1) {
	if _universe(ctx).hooks.debug == nil {
		ctx.err("hooks.debug, %v", _universe(ctx).hooks)
	}

	var t = ctx.i.(*testShellForStdoutDebugStruct)

	var v1 = ctx.val(".test.v1")
	if v1 == nil {
		ctx.err(".test.v1")
	} else if v1.String() != "" {
		ctx.err("%v{%v}", typeof(v1), v1)
	} else if v1.string(ctx) != "" {
		ctx.err("%v{%v}", typeof(v1), v1)
	}
	if len(t.v) != 1 {
		ctx.err("%v", t.v)
	} else if t.v[0].String() != "b a" {
		ctx.err("%v", t.v[0])
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}

	t.v, t.s = nil, ""

	if r := ctx.rule(".test.for"); r == nil {
		ctx.err(".test.for")
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0], "abc", "1"); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "" {
		ctx.err("%v", v)
	} else if v.string(ctx) != "" {
		ctx.err("%v", v)
	}
	if len(t.v) != 1 {
		ctx.err("%v", t.v)
	} else if t.v[0].String() != "1 abc" {
		ctx.err("%v", t.v[0])
	}
	if t.s != "1 abc" {
		ctx.err("%v", t.s)
	}

	t.v, t.s = nil, ""

	var v2 = ctx.val(".test.v2")
	if v2 == nil {
		ctx.err(".test.v2")
	} else if v2.String() != "${.test.for a,b}" {
		ctx.err("%v{%v}", typeof(v2), v2)
	}
	if len(t.v) != 0 {
		ctx.err("%v", t.v)
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if v2.string(ctx) != "" {
		ctx.err("%v{%v}", typeof(v2), v2)
	}
	if len(t.v) != 1 {
		ctx.err("%v", t.v)
	} else if t.v[0].String() != "b a" {
		ctx.err("%v", t.v[0])
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}

	t.v, t.s = nil, ""

	var v3 = ctx.val(".test.v3", "a", "b")
	if v3 == nil {
		ctx.err(".test.v3")
	} else if v3.String() != "${.test.for a,b}" {
		ctx.err("%v{%v}", typeof(v3), v3)
	}
	if len(t.v) != 0 {
		ctx.err("%v", t.v)
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if v3.string(ctx) != "" {
		ctx.err("%v{%v}", typeof(v3), v3)
	}
	if len(t.v) != 1 {
		ctx.err("%v", t.v)
	} else if t.v[0].String() != "b a" {
		ctx.err("%v", t.v[0])
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}

	t.v, t.s = nil, ""

	if r := ctx.rule(".test1"); r == nil {
		ctx.err(".test1")
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "exec{status=0}" {
		ctx.err("%v", v)
	} else if v.string(ctx) != "0" {
		ctx.err("%v", v)
	} else if t, y := v.(*execResult); !y {
		ctx.err("%v{%v}", typeof(v), v)
	} else if t.Status != 0 {
		ctx.err("%v", v)
	}

	if len(t.v) != 2 {
		ctx.err("%v", t.v)
	}
	if t.s == "1 test one\n2 test two" {
		ctx.err("%v", t.s)
	}

	t.v, t.s = nil, ""

	if r := ctx.rule(".test2"); r == nil {
		ctx.err(".test2")
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "exec{status=0}" {
		ctx.err("%v", v)
	} else if v.string(ctx) != "0" {
		ctx.err("%v", v)
	} else if t, y := v.(*execResult); !y {
		ctx.err("%v{%v}", typeof(v), v)
	} else if t.Status != 0 {
		ctx.err("%v", v)
	}

	if len(t.v) != 2 {
		ctx.err("%v", t.v)
	}
	if t.s == "1 test one\n2 test two" {
		ctx.err("%v", t.s)
	}
}
