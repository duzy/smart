//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

func testRules0(ctx *testcase) {
	if r := ctx.rule("rule"); r == nil {
		ctx.err("rule")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "rule1 rule1" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if l, y := v.(*list); !y {
		ctx.err("%v{%v}", typeof(v), v)
	} else if len(l.Elems) != 2 {
		ctx.err("%v", l.Elems)
	}

	if r := ctx.rule("rule1"); r == nil {
		ctx.err("rule1")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() == "" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if _, y := v.(*Plain); !y {
		ctx.err("%v{%v}", typeof(v), v)
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

	if v := ctx.get(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "fxxbar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "fxxbar" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	var p = ctx.Project()

	testResolveEntries = true

	if v := makeStrlit(ctx.Position(), ".test.foo"); v == nil || v.s != ".test.foo" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p.resolveEntries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	} else if r := p.resolveEntries(ctx.Context, v.s, false); r != nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	}

	if v := ctx.get("v1"); v == nil {
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
