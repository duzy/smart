//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

func testRules0(ctx *testcase) {
	var p = _project(ctx)

	if p.entries.puncs == nil {
		ctx.err("%v", ts(&p.entries))
	} else {
		if x, y := p.entries.puncs[MINUS]; !y {
			ctx.err("%v", p.entries.puncs)
		} else if len(x.a) != 1 {
			ctx.err("%v", ts(x))
		} else if z, y := x.a[0].(*rule); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if len(z.program) != 1 {
			ctx.err("%v", ts(z.program))
		}
	}

	if p.entries.words == nil {
		ctx.err("%v", ts(&p.entries))
	} else {
		if x, y := p.entries.words["rule0"]; !y {
			ctx.err("%v", p.entries.words)
		} else if len(x.a) != 1 {
			ctx.err("%v", ts(x))
		} else if z, y := x.a[0].(*rule); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if len(z.program) != 1 {
			ctx.err("%v", ts(z.program))
		}
		if x, y := p.entries.words["rule1"]; !y {
			ctx.err("%v", p.entries.words)
		} else if len(x.a) != 1 {
			ctx.err("%v", ts(x))
		} else if z, y := x.a[0].(*rule); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if len(z.program) != 1 {
			ctx.err("%v", ts(z.program))
		}
	}

	if s := "rule0"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0], "x", "y", "z"); v == nil {
		ctx.err("%v", tst{r})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if len(x.elems) != 4 {
		ctx.err("%v", tst{x})
	} else {
		i := 0
		if z, y := x.elems[i].(*bareword); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=bareword rule1}" {
		} else if z.s != "rule1" {
			ctx.err("%v", tst{z})
		}

		i = 1
		if z, y := x.elems[i].(*bareword); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=bareword rule1}" {
		} else if z.s != "rule1" {
			ctx.err("%v", tst{z})
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
		if z, y := x.elems[i].(*barecomp); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=barecomp {=bareword x} {=bareword y} {=bareword z}}" {
			ctx.err("%v", tst{x.elems[i]})
		}

		if ts(v) != "{=list {=bareword rule1} {=bareword rule1} {=flag {=null}} {=barecomp {=bareword x} {=bareword y} {=bareword z}}}" {
			ctx.err("%v %v", v, tst{v})
		}

		if v.String() != "rule1 rule1 - xyz" {
			ctx.err("%v %v", v, tst{v})
		}

		if s := v.string(ctx); s != "rule1 rule1 - xyz" {
			ctx.err("%v : %s", tst{v}, s)
		}
	}

	if s := "rule1"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0], bare("xyz")); v == nil {
		ctx.err("%v", tst{r})
	} else if ts(v) != "{=plain {=bareword xyz}}" {
		ctx.err("%v", tst{v})
	}
}

func testRules1(ctx *testcase) {
	var p = _project(ctx)

	if p.entries.puncs == nil {
		ctx.err("%v", ts(&p.entries))
	} else {
		if x, y := p.entries.puncs[STRING]; !y {
			ctx.err("%v", p.entries.puncs)
		} else if z, y := x.words[".test.foo"]; !y {
			ctx.err("%v", x)
		} else if len(z.a) != 1 {
			ctx.err("%v", ts(z))
		} else if x, y := z.a[0].(*rule); !y {
			ctx.err("%v", tst{z.a[0]})
		} else if len(x.program) != 1 {
			ctx.err("%v", ts(x.program))
		}

		if x, y := p.entries.puncs[DOT]; !y {
			ctx.err("%v", p.entries.puncs)
		} else if z, y := x.words["test"]; !y {
			ctx.err("%v", ts(x))
		} else if x, y := z.puncs[DOT]; !y {
			ctx.err("%v", z.puncs)
		} else {
			if _, y := x.words["foobar"]; !y {
				ctx.err("%v", ts(x))
			}
			if _, y := x.words["foobaz"]; !y {
				ctx.err("%v", ts(x))
			}
			if _, y := x.words["foobay"]; !y {
				ctx.err("%v", ts(x))
			}
			if _, y := x.words["fxx"]; !y {
				ctx.err("%v", ts(x))
			}
		}
	}

	if p.entries.words != nil {
		ctx.err("%v", ts(&p.entries))
	}

	if s := ".test.foobar"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=closure {=bareword foo} {=list {=bareword fxxbar}}}" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.foobaz"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=closure {=bareword foo} {=list {=barecomp {=punctuation .} {=bareword test} {=punctuation .} {=bareword fxx}}}}" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.foobay"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=closure {=bareword foo} {=list {=barecomp {=punctuation .} {=bareword test} {=punctuation .} {=bareword fxx}}}}" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.1"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=closure {=bareword foo} {=list {=bareword fxxbar}}}" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != "fxxbar" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	if s := ".test.2"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=closure {=bareword foo} {=list {=barecomp {=punctuation .} {=bareword test} {=punctuation .} {=bareword fxx}}}}" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	if s := ".test.3"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=closure {=bareword foo} {=list {=barecomp {=punctuation .} {=bareword test} {=punctuation .} {=bareword fxx}}}}" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{v}, s)
	}

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

	var v0, v1, v2, v3 Value
	var t = ctx.i.(*testShellForStdoutDebugStruct)

	if t.s != "b a b a b a" {
		ctx.err("%v", t.s)
	}
	if len(t.v) != 3 {
		ctx.err("%v", t.v)
	} else if ts(t.v) != "{=[Value] {=list {=bareword b} {=bareword a}} {=list {=bareword b} {=bareword a}} {=list {=bareword b} {=bareword a}}}" {
		ctx.err("%v", tst{t.v})
	}

	t.v, t.s = nil, ""

	if s := ".test.01"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=null}" {
		ctx.err("%v", tst{v})
	} else {
		v0 = v
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if len(t.v) != 0 {
		ctx.err("%v", tst{t.v})
	}
	if v0.string(ctx) != "" {
		ctx.err("%v", v0)
	}

	t.v, t.s = nil, ""

	if s := ".test.02"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
		ctx.err("%v", tst{v})
	} else {
		v0 = v
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}
	if len(t.v) != 1 {
		ctx.err("%v", t.v)
	} else if ts(t.v) != "{=list {=bareword b} {=bareword a}}" {
		ctx.err("%v", tst{t.v})
	}
	if v0.string(ctx) != "" {
		ctx.err("%v", v0)
	}

	t.v, t.s = nil, ""

	if s := ".test.v1"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=null}" {
		ctx.err("%v", tst{v})
	} else {
		v1 = v
	}
	if len(t.v) != 1 {
		ctx.err("%v", t.v)
	} else if ts(t.v[0]) != "{=list {=bareword b} {=bareword a}}" {
		ctx.err("%v", tst{t.v[0]})
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}
	if v1.string(ctx) != "" {
		ctx.err("%v", v1)
	}
	// if len(t.v) != 2 {
	// 	ctx.err("%v", t.v)
	// } else if ts(t.v[0]) != "{=list {=bareword b} {=bareword a}}" {
	// 	ctx.err("%v", tst{t.v[0]})
	// } else if ts(t.v[1]) != "{=list {=bareword b} {=bareword a}}" {
	// 	ctx.err("%v", tst{t.v[1]})
	// }

	t.v, t.s = nil, ""

	if s := ".test"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := _evoke_(ctx, r[0], "abc", "1"); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=delegate {=builtin debug} {=list {=bareword 1} {=bareword abc}}}" {
		ctx.err("%v", tst{v})
	} else if v := _evoke_(final{ctx}, r[0], "abc", "1"); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=null}" {
		ctx.err("%v", tst{v})
	}
	if len(t.v) != 1 {
		ctx.err("%v", t.v)
	} else if ts(t.v[0]) != "{=list {=bareword 1} {=bareword abc}}" {
		ctx.err("%v", tst{t.v[0]})
	}
	if t.s != "1 abc" {
		ctx.err("%v", t.s)
	}

	t.v, t.s = nil, ""

	if s := ".test.v2"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=rule_name .test} {=list {=bareword a}} {=list {=bareword b}}}" {
		ctx.err("%v", tst{v})
	} else {
		v2 = v
	}
	if len(t.v) != 0 {
		ctx.err("%v", t.v)
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if v2.string(ctx) != "" {
		ctx.err("%v", tst{v2})
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

	if s := ".test.v3"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v3 = d.value; v3 == nil {
		ctx.err("%v", d)
	} else if ts(v3) != "{=delegate {=rule_name .test} {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}" {
		ctx.err("%v", tst{v3})
	}
	if len(t.v) != 0 {
		ctx.err("%v", t.v)
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if v := ctx.val(v3, "a", "b"); v == nil {
		ctx.err("%v", tst{v3})
	} else if ts(v) != "{=delegate {=builtin debug} {=list {=strlit b} {=strlit a}}}" {
		ctx.err("%v", tst{v})
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

	if s := ".test.v4"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=builtin debug} {=list {=strlit b} {=strlit a}}}" {
		ctx.err("%v", tst{v})
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
