//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
)

func testTemplate(ctx *testcase) {
	var s string

	s = "xyz"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand2 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx yyy zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var.xxx"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "-xxx" {
		ctx.err("%v", tst{x.elems[1]})
	} else if s, t := v.String(), "test-xxx"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var.yyy"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "-yyy" {
		ctx.err("%v", tst{x.elems[1]})
	} else if s, t := v.String(), "test-yyy"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var.zzz"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "-zzz" {
		ctx.err("%v", tst{x.elems[1]})
	} else if s, t := v.String(), "test-zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "vars"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 3 {
		ctx.err("%v ; %d", tst{v}, x.len())
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx yyy zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "), "xxx yyy zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), ".self .usee var.zzz var2 vars xyz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx xxx yyy yyy zzz zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx xxx yyy yyy zzz zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx xxx yyy yyy zzz zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a=x a=y a=z b=x b=y b=z c=x c=y c=z"; s != t {
		ctx.err("%s: %s != %s → %s", d.name, s, t, tst{v})
	}

	s = ".test.5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a=x a=y a=x b=y b=x b=y"; s != t {
		ctx.err("%s: %s != %s → %v", d.name, s, t, tst{v})
	}

	s = ".test.6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a=x a=y b=x b=x c=y c=x"; s != t {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	}

	s = ".test.7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x c2=x c1=y c2=y c1=z c2=z"; s != t {
		ctx.err("%s: %s != %s → %v", d.name, s, t, tst{v})
	}

	s = ".test.8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z {}=x c2=x {}=y c2=y {}=z c2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c2=x c2=y c2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.9"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.10"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x {}=x c1=y {}=y c1=z {}=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x c1=y c1=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.11"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.12"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x1 a2=x2 a1=y1 a2=y2 a1={} a2=z2 b1=x1 b2=x2 b1=y1 b2=y2 b1={} b2=z2 c1=x1 {}=x2 c1=y1 {}=y2 c1={} {}=z2"; s != t {
		ctx.err("%s: %s != %s | %v", d.name, s, t, tst{v})
	} else if s, t := __string(ctx,v), "a1=x1 a2=x2 a1=y1 a2=y2 a2=z2 b1=x1 b2=x2 b1=y1 b2=y2 b2=z2 c1=x1 c1=y1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.13"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x1 a2=x2 a1=y1 a2=y2 b1=x1 b2=x2 b1=y1 b2=y2"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func testTemplateForeach(ctx *testcase) {
	var s string
	var proj = _project(ctx)

	s = ".test.a"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if len(t) != 1 {
		ctx.err("%s %v", s, t)
	} else if x, y := t[0].(matched_rule); !y {
		ctx.err("%v", tst{t[0]})
	} else if x.String() != s {
		ctx.err("%v", tst{x.rule})
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := t.target.(*compound); !y {
		ctx.err("%v", tst{t.target})
	} else if t.target.String() != ".test.a" {
		ctx.err("%v", tst{t.target})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program[0].depends)
	} else if t.program[0].depends[0].String() != "a" {
		ctx.err("%v: %v", t.target, t.program[0].depends[0])
	} else if t.program[0].depends[1].String() != "$(foreach a d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print a $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.a" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.b"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := t.target.(*compound); !y {
		ctx.err("%v", tst{t.target})
	} else if t.target.String() != ".test.b" {
		ctx.err("%v", tst{t.target})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program[0].depends)
	} else if t.program[0].depends[0].String() != "b" {
		ctx.err("%v: %v", t.target, t.program[0].depends[0])
	} else if t.program[0].depends[1].String() != "$(foreach b d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print b $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.b" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.c"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := t.target.(*compound); !y {
		ctx.err("%v", tst{t.target})
	} else if t.target.String() != ".test.c" {
		ctx.err("%v", tst{t.target})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program[0].depends)
	} else if t.program[0].depends[0].String() != "c" {
		ctx.err("%v: %v", t.target, t.program[0].depends[0])
	} else if t.program[0].depends[1].String() != "$(foreach c d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print c $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.c" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.a.aaa"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print foa.aaa foa.aaa $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = ".test.b.bbb"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print fob.bbb fob.bbb $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = ".test.c.ccc"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print foc.ccc foc.ccc $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = ".test.o.bar"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print foo.bar foo.bar $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = "v.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foa.aaa" {
		ctx.err("%v", d)
	}

	s = "v.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "fob.bbb" {
		ctx.err("%v", d)
	}

	s = "v.c"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foc.ccc" {
		ctx.err("%v", d)
	}

	s = "v.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foo.bar" {
		ctx.err("%v", d)
	}

	s = "v1.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "a" {
		ctx.err("%v", d)
	}

	s = "v1.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "b" {
		ctx.err("%v", d)
	}

	s = "v1.c"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "c" {
		ctx.err("%v", d)
	}

	s = "v1.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "o" {
		ctx.err("%v", d)
	}

	s = "v2.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "aaa" {
		ctx.err("%v", d)
	}

	s = "v2.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "bbb" {
		ctx.err("%v", d)
	}

	s = "v2.c"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "ccc" {
		ctx.err("%v", d)
	}

	s = "v2.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "bar" {
		ctx.err("%v", d)
	}

	s = "v.a.aaa"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foa.aaa" {
		ctx.err("%v", d)
	}

	s = "v.b.bbb"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "fob.bbb" {
		ctx.err("%v", d)
	}

	s = "v.c.ccc"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foc.ccc" {
		ctx.err("%v", d)
	}

	s = "v.o.bar"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foo.bar" {
		ctx.err("%v", d)
	}
}
