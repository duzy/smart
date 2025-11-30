//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
)

func testRules0(ctx *testcase) {
	var p = _project(ctx)

	if d := ctx.def("items"); d == nil {
		ctx.err("items")
	} else if s, t := "{=list {=word a} {=word b} {=word c}}", ts(d.value); s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	} else if s, t := "a b c", d.value.String(); s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	}

	if d := ctx.def("line"); d == nil {
		ctx.err("line")
	} else if s, t := "{=list {=plainline {=raw foo } {=delegate {=auto 1}} {=raw 	}} {=word bar}}", ts(d.value); s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	} else if s, t := "{=plainline foo $1	} bar", d.value.String(); s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	} else if s, t := "foo 	\nbar", __string(ctx,d.value); s != t {
		t = strings.ReplaceAll(t, "\n", `\n`)
		s = strings.ReplaceAll(s, "\n", `\n`)
		ctx.err("%v: %s != %s", d.value, t, s)
	}

	if d := ctx.def("lines"); d == nil {
		ctx.err("lines")
	} else if s, t := "{=list {=plainline {=raw line-} {=word foo}} {=plainline {=raw line-} {=word bar}}}", ts(d.value); s != t {
		ctx.err("%v: %s != %s ; %s", d.value, s, t, ts(d.value))
	} else if s, t := "{=plainline line-foo} {=plainline line-bar}", d.value.String(); s != t {
		ctx.err("%v: %s != %s ; %s", d.value, s, t, ts(d.value))
	} else if s, t := "line-foo\nline-bar\n", __string(ctx,d.value); s != t {
		ctx.err("%v: %s != %s ; %s", d.value, s, t, ts(d.value))
	}

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
		for _, tag := range []string{"x","y","z","xx","yy","zz","xxx","yyy","zzz"} {
			if x, y := p.entries.words["rule-"+tag]; !y {
				ctx.err("%v", p.entries.words)
			} else if len(x.a) != 1 {
				ctx.err("%v", ts(x))
			} else if z, y := x.a[0].(*rule); !y {
				ctx.err("%v", tst{x.a[0]})
			} else if len(z.program) != 1 {
				ctx.err("%v", ts(z.program))
			}
		}
	}

	s := "rule0"
	r := ctx.rule(s)
	if r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0], "x", "y", "z"); v == nil {
		ctx.err("%v", tst{r})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if i := len(x.elems); i != 4 {
		ctx.err("%v", tst{x})
	} else {
		i = 0
		if z, y := x.elems[i].(*word); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=word rule1}" {
			ctx.err("%v", tst{z})
		} else if z.s != "rule1" {
			ctx.err("%v", tst{z})
		}

		i = 1
		if z, y := x.elems[i].(*word); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=word rule1}" {
			ctx.err("%v", tst{z})
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
		if z, y := x.elems[i].(*compound); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(z) != "{=compound {=word x} {=word y} {=word z}}" {
			ctx.err("%v", tst{x.elems[i]})
		}

		if ts(v) != "{=list {=word rule1} {=word rule1} {=flag {=null}} {=compound {=word x} {=word y} {=word z}}}" {
			ctx.err("%v %v", v, tst{v})
		}

		if v.String() != "rule1 rule1 - xyz" {
			ctx.err("%v %v", v, tst{v})
		}

		if s := __string(ctx,v); s != "rule1 rule1 - xyz" {
			ctx.err("%v : %s", tst{v}, s)
		}
	}

	s = "rule1"
	r = ctx.rule(s)
	if r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", r)
	} else if v := test_evoke(ctx, r[0], bare("xxyzz")); v == nil {
		ctx.err("%v", r[0])
	} else if s, t := "{=plain(text) {=plainline xxyzz}}", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "{=plain(text) {=plainline {=word xxyzz}}}", ts(v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	for _, tag := range []string{"x", "y", "z"} {
		s = "rule-"+tag
		r = ctx.rule(s)
		if r == nil {
			ctx.err("no such %s; %v", s, p.entries)
		} else if len(r) != 1 {
			ctx.err("%v", r)
		} else if recipes := r[0].programs()[0].recipes; len(recipes) != 1 {
			ctx.err("%v", recipes)
		} else if s, t := "", __string(ctx,recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline $(foreach $(ARGS),arg-$_)}", recipes[0].String(); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline {=delegate {=builtin foreach} {=list {=delegate {=auto ARGS}}} {=list {=compound {=word arg-} {=delegate {=auto _}}}}}}", ts(recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if v := test_evoke(ctx, r[0], []string{"aa","bb","cc"}); v == nil {
			ctx.err("%v", r[0])
		} else if s, t := "{=plain(text) {=plainline arg-aa arg-bb arg-cc}}", v.String(); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline {=list {=compound {=word arg-} {=word aa}} {=compound {=word arg-} {=word bb}} {=compound {=word arg-} {=word cc}}}}}", ts(v); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "arg-aa arg-bb arg-cc\n", __string(ctx,v); s != t {
			// t = strings.ReplaceAll(t, "\n", `\n`)
			// s = strings.ReplaceAll(s, "\n", `\n`)
			ctx.err("%v : %s != %s", v, t, s)
		}
	}

	for _, tag := range []string{"xx", "yy", "zz"} {
		s = "rule-"+tag
		r = ctx.rule(s)
		if r == nil {
			ctx.err("no such %s; %v", s, p.entries)
		} else if len(r) != 1 {
			ctx.err("%v", r)
		} else if recipes := r[0].programs()[0].recipes; len(recipes) != 1 {
			ctx.err("%v", recipes)
		} else if s, t := "{=plainline $(foreach $(ARGS),{=plainline arg-$_})}", recipes[0].String(); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline {=delegate {=builtin foreach} {=list {=delegate {=auto ARGS}}} {=list {=plainline {=raw arg-} {=delegate {=auto _}}}}}}", ts(recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if v := test_evoke(ctx, r[0], []string{"aa","bb","cc"}); v == nil {
			ctx.err("%v", r[0])
		} else if s, t := "{=plain(text) {=plainline {=plainline arg-aa} {=plainline arg-bb} {=plainline arg-cc}}}", v.String(); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline {=list {=plainline {=raw arg-} {=word aa}} {=plainline {=raw arg-} {=word bb}} {=plainline {=raw arg-} {=word cc}}}}}", ts(v); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "arg-aa\narg-bb\narg-cc\n", __string(ctx,v); s != t {
			t = strings.ReplaceAll(t, "\n", `\n`)
			s = strings.ReplaceAll(s, "\n", `\n`)
			ctx.err("%v : %s != %s", v, t, s)
		}
	}

	for _, tag := range []string{"xxx", "yyy", "zzz"} {
		s = "rule-"+tag
		r = ctx.rule(s)
		if r == nil {
			ctx.err("no such %s; %v", s, p.entries)
		} else if len(r) != 1 {
			ctx.err("%v", r)
		} else if recipes := r[0].programs()[0].recipes; len(recipes) != 3 {
			ctx.err("%v", recipes)
		} else if s, t := "{=plainline {=compound {=word arg-} {=word a}}}", ts(recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline {=compound {=word arg-} {=word b}}}", ts(recipes[1]); s != t {
			ctx.err("%v : %s != %s", recipes[1], t, s)
		} else if s, t := "{=plainline {=compound {=word arg-} {=word c}}}", ts(recipes[2]); s != t {
			ctx.err("%v : %s != %s", recipes[2], t, s)
		} else if v := test_evoke(ctx, r[0], bare("aa"), bare("bb"), bare("cc")); v == nil {
			ctx.err("%v", r[0])
		} else if s, t := "arg-a\narg-b\narg-c\n", __string(ctx,v); s != t {
			t = strings.ReplaceAll(t, "\n", `\n`)
			s = strings.ReplaceAll(s, "\n", `\n`)
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline arg-a} {=plainline arg-b} {=plainline arg-c}}", v.String(); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline {=compound {=word arg-} {=word a}}} {=plainline {=compound {=word arg-} {=word b}}} {=plainline {=compound {=word arg-} {=word c}}}}", ts(v); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		}
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
			if _, y := x.words["foobax"]; !y {
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

	var s string

	s = ".test.foobax"
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=closure {=word foo} {=list {=word fxxbar}}}" {
		ctx.err("%v", tst{v})
	}

	s = ".test.foobay"
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=closure {=word foo} {=list {=compound {=punct .} {=word test} {=punct .} {=word fxx}}}}" {
		ctx.err("%v", tst{v})
	}

	s = ".test.foobaz"
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=closure {=word foo} {=list {=compound {=punct .} {=word test} {=punct .} {=word fxx}}}}" {
		ctx.err("%v", tst{v})
	}

	s = ".test.1"
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=closure {=word foo} {=list {=word fxxbar}}}" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx,v); s != "fxxbar" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	s = ".test.2"
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=closure {=word foo} {=list {=compound {=punct .} {=word test} {=punct .} {=word fxx}}}}" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx,v); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	s = ".test.3"
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=closure {=word foo} {=list {=compound {=punct .} {=word test} {=punct .} {=word fxx}}}}" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx,v); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{v}, s)
	}

	s = ".test.foo"
	if v := _strlit(_position(ctx), s); v == nil || v.s != s {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p._entries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	} else if r := p._entries(ctx.Context, v.s, false); r != nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	}

	s = "v1"
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "'.test.foo'" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if s := __string(ctx,v); s != ".test.foo" {
		ctx.err("%v{%v} -> %s", typeof(v), v, s)
	} else if _, y := v.(*strlit); !y {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p._entries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p._entries(ctx.Context, __string(ctx,v), false); r != nil {
		ctx.err("%v{%v}", typeof(v), v)
	}
}

type testShellForStdoutDebugStruct struct { v, s string }
func testShellForStdoutDebugHook(ctx Context, s string, v []Value, i any) {
	t := i.(*testShellForStdoutDebugStruct)
	for _, v := range v { t.v += ts(v) }
	t.s += s
}
func testShellForStdout(ctx testcase1) {
	if u := _universe(ctx); u.hooks.debug == nil {
		ctx.err("hooks.debug, %v", u.hooks)
	}

	var t = ctx.i.(*testShellForStdoutDebugStruct)

	if false && t.s != "b ab ab a" {
		ctx.err("%v", t.s)
	}
	if false && ts(t.v) != "{=[Value] {=list {=word b} {=word a}} {=list {=word b} {=word a}} {=list {=word b} {=word a}}}" {
		ctx.err("%d %v", len(t.v), tst{t.v})
	}

	t.v, t.s = "", ""

	if s := ".test.01"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=null}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if len(t.v) != 0 {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test.02"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=delegate {=rule {=compound {=punct .} {=word test} {=punct .} {=decimal 0}}} {=list {=word a}} {=list {=word b}}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}
	if t.v != "{=list {=word b} {=word a}}" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test.v1"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{=null}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if t.v != "" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0], bare("a"), bare("b")); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
		ctx.err("%v", tst{v})
	} else if v := test_evoke(_final(ctx), r[0], "a", "b"); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{=null}" {
		ctx.err("%v", tst{v})
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}
	if t.v != "{=list {=word b} {=word a}}" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test.v2"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=rule {=compound {=punct .} {=word test}}} {=list {=word a}} {=list {=word b}}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if s := "b a"; t.s != s {
		ctx.err("%s != %s", t.s, s)
	}
	if s := "{=list {=word b} {=word a}}"; t.v != s {
		ctx.err("%s != %s", t.v, s)
	}

	t.v, t.s = "", ""

	if s := ".test.v3"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=rule {=compound {=punct .} {=word test}}} {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	} else if t := ctx.val(v, bare("a"), bare("b")); t == nil {
		ctx.err("%v", tst{v})
	} else if ts(t) != "{=null}" {
		ctx.err("%v", tst{t})
	}
	if s := "b a"; t.s != s {
		ctx.err("%s != %s", t.s, s)
	}
	if s := "{=list {=word b} {=word a}}"; t.v != s {
		ctx.err("%s != %s", t.v, s)
	}

	t.v, t.s = "", ""

	if s := ".test.v4"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=null}" {
		ctx.err("%v", tst{v})
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if t.v != "" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test1"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", tst{r[0]})
	} else if v.String() != "{=exec {=status 0}}" {
		ctx.err("%v", v)
	} else if __string(ctx,v) != "0" {
		ctx.err("%v", v)
	} else if t, y := v.(*exec_result); !y {
		ctx.err("%v", ts(v))
	} else if t.Status != 0 {
		ctx.err("%v", v)
	}
	if t.s != "1 test one\n2 test two\n" {
		ctx.err("%v", t.s)
	}
	if t.v != `{=list {=decimal 1} {=strlit test one\n}}{=list {=decimal 2} {=strlit test two\n}}` {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test2"; false {} else
	if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "{=exec {=status 0}}" {
		ctx.err("%v", v)
	} else if __string(ctx,v) != "0" {
		ctx.err("%v", v)
	} else if t, y := v.(*exec_result); !y {
		ctx.err("%v", ts(v))
	} else if t.Status != 0 {
		ctx.err("%v", v)
	}
	if t.s != "1 test one\n2 test two\n3 test thr\n" {
		ctx.err("%v", t.s)
	}
	if t.v != `{=list {=decimal 1} {=strlit test one\n}}{=list {=decimal 2} {=strlit test two\n}}{=list {=decimal 3} {=strlit test thr\n}}` {
		ctx.err("%v", t.v)
	}
}
