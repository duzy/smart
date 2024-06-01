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
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", ust{r})
	} else if v.String() != "$< $> - $(ARG1) $(ARG2) $(ARG3)" {
		ctx.err("%v", ust{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if len(x.elems) != 6 {
		ctx.err("%v", ust{x})
	} else if s := v.string(ctx); s != "rule1 rule1 -" {
		ctx.err("%v : %s", ust{v}, s)
	}

	if s := "rule1"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", ust{r})
	} else if v.String() == "$(ARG1)" {
		ctx.err("%v", ust{v})
	} else if x, y := v.(*Plain); !y {
		ctx.err("%v", ust{x})
	} else if s := v.string(ctx); s != "" {
		ctx.err("%v", ust{v})
	}
}

func testRules1(ctx *testcase) {
	if r := ctx.rule(".test.foobar"); r == nil {
		ctx.err(".test.foobar")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "fxxbar" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*bareword); !y {
		ctx.err("%T %v", v, v)
	}

	if r := ctx.rule(".test.foobaz"); r == nil {
		ctx.err(".test.foobaz")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%T %v", v, v)
	}

	if r := ctx.rule(".test.foobay"); r == nil {
		ctx.err(".test.foobay")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.val(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "fxxbar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "fxxbar" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.val(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.val(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	var p = ctx.project()

	testResolveEntries = true

	if v := makeStrlit(ctx.Position(), ".test.foo"); v == nil || v.s != ".test.foo" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p.resolveEntries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	} else if r := p.resolveEntries(ctx.Context, v.s, false); r != nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	}

	if v := ctx.val("v1"); v == nil {
		ctx.err("v1")
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
	} else if v := inv(ctx, r, "abc", "1"); v == nil {
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
	} else if v := inv(ctx, r); v == nil {
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
	} else if v := inv(ctx, r); v == nil {
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
