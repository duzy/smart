//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strconv"
	"strings"
	"fmt"
)

type testValueStruct struct {
	assert_bool bool
	assert_value Value
}
func testValueAssertHook(ctx Context, v Value, b bool, i any) {
	st := i.(*testValueStruct)
	st.assert_bool = b
	st.assert_value = v
}
func testValue(ctx testcase1) {
	st := ctx.i.(*testValueStruct)
	if st.assert_value == nil {
		// ctx.err("assert: nil value")
	} else if st.assert_bool {
		// ctx.err("assert")
	}

	if d := ctx.def("vals"); d == nil {
		ctx.err("vals")
	} else if d.scope != _project(ctx).scope {
		ctx.err("%v", tst{d})
	} else if d.scope.project != _project(ctx) {
		ctx.err("%v", tst{d})
	} else if val := d.value; val == nil {
		ctx.err("%v", tst{d})
	} else if l, y := val.(*list); !y {
		ctx.err("%v", tst{val})
	} else if len(l.elems) != 13 {
		ctx.err("%v : %v", len(l.elems), l)
	} else if v, y := l.elems[0].(disjunction); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if t, y := v.Value.(*list); !y {
		ctx.err("%v", tst{v.Value})
	} else if len(t.elems) != 6 {
		ctx.err("%v", tst{v.Value})
	} else if w, y := t.elems[0].(*word); !y || w.s != "word" {
		ctx.err("%v", tst{t.elems[0]})
	} else if _, y := t.elems[1].(*path); !y {
		ctx.err("%v", tst{t.elems[1]})
	} else if s, y := t.elems[2].(*strlit); !y || s.String() != "'strlit'" || s.string(ctx) != "strlit" {
		ctx.err("%v", tst{t.elems[2]})
	} else if c, y := t.elems[3].(*strcomp); !y || c.String() != `"strcomp"` || c.string(ctx) != "strcomp" {
		ctx.err("%v", tst{t.elems[3]})
	} else if s, t := `word foo/bar strlit strcomp 0 1`, v.string(ctx); s != t {
		ctx.err("%v: %s != %s", tst{v}, t, s)
	} else if s, t := `{word foo/bar 'strlit' "strcomp" 0 1}`, v.String(); s != t {
		ctx.err("%v: %s != %s", tst{v}, t, s)
	} else if a, y := l.elems[1].(*answer); !y || a.bool != true {
		ctx.err("%v", tst{l.elems[1]})
	} else if b, y := l.elems[2].(*boolean); !y || b.bool != false {
		ctx.err("%v", tst{l.elems[2]})
	} else if b, y := l.elems[3].(*boolean); !y || b.bool != true {
		ctx.err("%v", tst{l.elems[3]})
	} else if p, y := l.elems[4].(*path); !y || p.String() != "{=path foo}" || p.string(ctx) != "foo" {
		ctx.err("%v", tst{l.elems[4]})
	} else if p, y := l.elems[5].(*path); !y || p.String() != "foo/bar" || p.string(ctx) != "foo/bar" {
		ctx.err("%v", tst{l.elems[5]})
	} else if f, y := l.elems[6].(*file); !y || f.String() != "{=file foobar}" || f.string(ctx) != "foobar" {
		ctx.err("%v", tst{l.elems[6]})
	} else if g, y := l.elems[7].(*globpat); !y || g.String() != "**.c" || g.string(ctx) != "**.c" {
		ctx.err("%v", tst{l.elems[7]})
	} else if r, y := l.elems[8].(*regexpat); !y || r.String() != "{=regex xx}" || r.string(ctx) != "xx" {
		ctx.err("%v", tst{l.elems[8]})
	} else if i, y := l.elems[9].(*decimal); !y || i.String() != "1" || i.string(ctx) != "1" /* || i.int(ctx) != 1 */ {
		ctx.err("%v", tst{l.elems[9]})
	} else if f, y := l.elems[10].(*float); !y || f.String() != "0.1" {
		ctx.err("%v", tst{l.elems[10]})
	} else if n, y := l.elems[11].(*none); !y || n.String() != `` || n.string(ctx) != `` {
		ctx.err("%v", tst{l.elems[11]})
	} else if n, y := l.elems[12].(*null); !y || n.String() != `{}` || n.string(ctx) != `` {
		ctx.err("%v", tst{l.elems[12]})
	} else if s, t := `{word foo/bar 'strlit' "strcomp" 0 1} {=yes} {=false} {=true} {=path foo} foo/bar {=file foobar} **.c {=regex xx} 1 0.1 {}`, val.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{val})
	} else if s, t := `word foo/bar strlit strcomp 0 1 yes false true foo foo/bar foobar **.c xx 1 0.1`, val.string(ctx); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{val})
	}

	if d := ctx.def("disjunction0"); d == nil {
		ctx.err("disjunction0: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=disjunction {=list {=word a} {=word b} {=word c}}}"; s != t {
		ctx.err("%s != %s ; %v", s, t, v)
	} else if s, t := v.String(), "{a b c}"; s != t {
		ctx.err("%s != %s ; %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "a b c"; s != t {
		ctx.err("%s != %s ; %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction00"); d == nil {
		ctx.err("disjunction00: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=word x}"; s != t {
		ctx.err("%s != %s ; %v", s, t, v)
	} else if s, t := v.String(), "x"; s != t {
		ctx.err("%s != %s ; %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "x"; s != t {
		ctx.err("%s != %s ; %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction01"); d == nil {
		ctx.err("disjunction01: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=compound {=word x} {=disjunction {=delegate {=auto 1}}}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "x{$1}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b","c"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := ts(v), "{=list {=compound {=word x} {=word a}} {=compound {=word x} {=word b}} {=compound {=word x} {=word c}}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "xa xb xc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("disjunction1"); d == nil {
		ctx.err("disjunction1: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=compound {=word x} {=disjunction {=list {=word a} {=word b} {=word c}}}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "x{a b c}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "xa xb xc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("disjunction2"); d == nil {
		ctx.err("disjunction2: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	if d := ctx.def("disjunction3"); d == nil {
		ctx.err("disjunction3: %v", _project(ctx).defs)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	var (
		glob1 = ctx.val("glob1")
		glob2 = ctx.val("glob2")
		glob3 = ctx.val("glob3")
		regex1 = ctx.val("regex1")
		regex2 = ctx.val("regex2")
		regex3 = ctx.val("regex3")
		regex4 = ctx.val("regex4")
		regex5 = ctx.val("regex5")
		regex6 = ctx.val("regex6")
	)

	if glob1.string(ctx) != "*.c" {
		ctx.err("%v", tst{glob1})
	}

	if glob2.string(ctx) != "**.c" {
		ctx.err("%v", tst{glob2})
	}

	if g, y := glob1.(*globpat); !y {
		ctx.err("%v", tst{glob1})
	} else if g.String() != "*.c" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if g.string(ctx) != "*.c" {
		ctx.err("%v", tst{g})
	}

	if g, y := glob2.(*globpat); !y {
		ctx.err("%v", tst{glob2})
	} else if g.String() != "**.c" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if g.string(ctx) != "**.c" {
		ctx.err("%v", tst{g})
	}

	if g, y := glob3.(*globpat); !y {
		ctx.err("%v", tst{glob3})
	} else if g.String() != "{=glob x*z?.c}" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if g.string(ctx) != "x*z?.c" {
		ctx.err("%v", tst{g})
	}

	if regex1.string(ctx) != `x{1}, x{1,}, x{1,2}, x{5}?, x{2,}?, x{2,8}? \p{Greek}, \P{Greek}` {
		ctx.err("regex1 is wrong: %v", regex1)
	}

	if regex2.string(ctx) != `(re) (?P<name>re) (?:re) (?im) (?sU:re) \x{10ffff} \x1f \123 \* \. \? \$` {
		ctx.err("regex2 is wrong: %v", regex2)
	}

	if regex3.string(ctx) != `[[:xdigit:]]*, [^[:alpha:]], [^xyz] [a-z] \A \B \b \Q**??^:[]{}\E \^ \z` {
		ctx.err("regex3 is wrong: %v", regex3)
	}

	if regex4.string(ctx) != `fo{2}\.c` {
		ctx.err("regex4 is wrong: %v", regex4)
	}

	if regex5.string(ctx) != `fo{2}/bar\.c` {
		ctx.err("regex5 is wrong: %v", regex5)
	}

	if regex6.string(ctx) != `fo{2}(/o{2}){3}/bar\.c` {
		ctx.err("regex6 is wrong: %v", regex6)
	}

	if val := ctx.val("val1"); val == nil {
		ctx.err("val1")
	} else if val.string(ctx) != "foo.c" {
		ctx.err("%v", tst{val})
	} else if a, b, c := glob1.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob1, val, a, b, c)
	} else if a, b, c := glob2.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, val, a, b, c)
	} else if a, b, c := regex4.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", regex4, val, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regex4, val, a, b, c)
	} else if s != val.string(ctx) {
		ctx.err("match(%v, %v): %v %v %v", regex4, val, a, b, c)
	} else if len(c) != 0 {
		ctx.err("match(%v, %v): %v %v %v", regex4, val, a, b, c)
	}

	if val := ctx.val("val2"); val == nil {
		ctx.err("val2")
	} else if s, t := val.String(), "foo/bar.c"; s != t {
		ctx.err("%v : %s != %s", tst{val}, s, t)
	} else if s := val.string(ctx); s != t {
		ctx.err("%v : %s != %s", tst{val}, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v : %v", tst{val}, val)
	} else if len(p.elems) != 2 {
		ctx.err("%v: %v: %v", typeof(val), val, p.elems)
	} else if a, b, c := glob1.match(ctx, val); a == true {
		if false { ctx.err("match(%v, %v): %v %v %v", glob1, val, a, b, c) }
	} else if a, b, c := glob2.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, val, a, b, c)
	} else if a, b, c := regex5.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", regex5, val, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regex5, val, a, b, c)
	} else if s != val.string(ctx) {
		ctx.err("match(%v, %v): %v %v %v", regex5, val, a, b, c)
	} else if len(c) != 0 {
		ctx.err("match(%v, %v): %v %v %v", regex5, val, a, b, c)
	}

	if val := ctx.val("val3"); val == nil {
		ctx.err("val3")
	} else if val.string(ctx) != "foo/oo/oo/oo/bar.c" {
		ctx.err("%v", tst{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", tst{val})
	} else if len(p.elems) != 5 {
		ctx.err("%v: %v: %v", typeof(val), val, p.elems)
	} else if a, b, c := glob1.match(ctx, val); a == true {
		if false { ctx.err("match(%v, %v): %v %v %v", glob1, val, a, b, c) }
	} else if a, b, c := glob2.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, val, a, b, c)
	} else if a, b, c := regex6.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", regex6, val, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regex6, val, a, b, c)
	} else if s != val.string(ctx) {
		ctx.err("match(%v, %v): %v %v %v", regex6, val, a, b, c)
	} else if len(c) != 1 {
		ctx.err("match(%v, %v): %v %v %v", regex6, val, a, b, c)
	} else if c[0] != "/oo" {
		ctx.err("match(%v, %v): %v %v %v", regex6, val, a, b, c)
	}

	// TODO: test glob.stencil(...)

	if val := ctx.val("val4"); val == nil {
		ctx.err("val4")
	} else if s, t := val.String(), "a\\,b\\,c,x\\,y\\,z"; s != t {
		ctx.err("%v : %v", tst{val}, val)
	} else if s, t := val.string(ctx), "a,b,c,x,y,z"; s != t {
		ctx.err("%v : %v", tst{val}, val)
	} else if s, t := ts(val), "{=compound {=word a} {=escaped \\,} {=word b} {=escaped \\,} {=word c} {=punct ,} {=word x} {=escaped \\,} {=word y} {=escaped \\,} {=word z}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{val})
	} else if p, y := val.(*compound); !y {
		ctx.err("%v : %v", tst{val}, val)
	} else if len(p.elems) != 11 {
		ctx.err("%v: %v %v", typeof(val), val, p.elems)
	}

	if val := ctx.val("val5"); val == nil {
		ctx.err("val5")
	} else if s, t := val.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
		ctx.err("%s != %s : %v", s, t, tst{val})
	} else if s, t := val.string(ctx), `"a,b,c x,y,z"`; s != t {
		ctx.err("%s != %s : %v", s, t, tst{val})
	}

	// TODO: test regexp.stencil(...)

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := tv(v), "{url:http://extbit.io/help}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := ts(v), "{=url {=word http} {} {} {=compound {=word extbit} {=punct .} {=word io}} {} {=path {=punct root} {=word help}} {=[]} {}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := tv(v), "{url:https://extbit.com}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := ts(v), "{=url {=word https} {} {} {=compound {=word extbit} {=punct .} {=word com}} {} {} {=[]} {}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := tv(v), "{url:https://extbit.com?foo=x}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := ts(v), "{=url {=word https} {} {} {=compound {=word extbit} {=punct .} {=word com}} {} {} {=[] {=pair {=word foo}={=word x}}} {}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val9"); d == nil {
		ctx.err("val9")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := tv(v), "{url:https://extbit.com?foo=x&bar=y#foobar}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := ts(v), "{=url {=word https} {} {} {=compound {=word extbit} {=punct .} {=word com}} {} {} {=[] {=pair {=word foo}={=word x}} {=pair {=word bar}={=word y}}} {=word foobar}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := tv(v), "{url:https://ext.pub?foo=x+y+z&bar=x%20y%20z#foo+bar}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := ts(v), "{=url {=word https} {} {} {=compound {=word ext} {=punct .} {=word pub}} {} {} {=[] {=pair {=word foo}={=word x+y+z}} {=pair {=word bar}={=compound {=word x} {=punct %} {=decimal 20} {=word y} {=punct %} {=decimal 20} {=word z}}}} {=word foo+bar}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def(`configure.types."atomic.h"`); d == nil {
		ctx.err(`configure.types."atomic.h"`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "atomic_bool", v.string(ctx); s != t {
		ctx.err("%s != %s; %v", s, t, v)
	}

	if d := ctx.def(`configure.types.<atomic.h>`); d == nil {
		ctx.err(`configure.types.<atomic.h>`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "atomic_bool", v.string(ctx); s != t {
		ctx.err("%s != %s; %v", s, t, v)
	}

	if d := ctx.def(`configure.types`); d == nil {
		ctx.err(`configure.types`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if t := v.string(ctx); t == "" {
		ctx.err("%s; %v", t, v)
	} else if s := `- configure.types.<atomic.h> <atomic.h> atomic.h,`; !strings.Contains(t, s) {
		ctx.err("%s, %s; %v", s, t, v)
	} else if s := `- configure.types."atomic.h" "atomic.h" , atomic.h`; !strings.Contains(t, s) {
		ctx.err("%s, %s; %v", s, t, v)
	}

	if d := ctx.def("conf0"); d == nil {
		ctx.err("conf0")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=word conf0}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "conf0"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "conf0"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	j := _project(ctx)

	if d := ctx.def("conf1"); d == nil {
		ctx.err("conf1")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo.o foo-x.o foo-x-y.o foo-x-y-z.o foobar.o"; s != t {
		g := fmt.Sprintf(`$(grep {=regex ^.+?\.o$$},$0,%s/test.txt)`, j.absPath)
		ctx.err("%s != %s : %s : %v", s, t, g, tst{v})
	} else if s, t := v.string(ctx), "foo.o foo-x.o foo-x-y.o foo-x-y-z.o foobar.o"; s != t {
		ctx.err("%s != %s : %s : %v", s, t, v, tst{v})
	}

	s := "foo.o foo . o "
	s += "foo-x.o foo -x -x x . o "
	s += "foo-x-y.o foo -x-y -y y . o "
	s += "foo-x-y-z.o foo -x-y-z -z z . o "
	s += "foobar.o foobar . o"
	if d := ctx.def("conf2"); d == nil {
		ctx.err("conf2")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if x, y := v.(*list); !y {
		ctx.err("%v %v", typeof(v), v)
	} else if x.len() != 5*7 {
		ctx.err("%d, %v", x.len(), x)
	} else if t := x.string(ctx); s != t {
		ctx.err("%s != %s; %v", s, t, x)
	}
}

func testAuto(ctx *testcase) {
	if c := _universe(ctx); c == nil {
		ctx.err("context.cast")
	}
	{
		ac := automatic{Context:ctx, defs:make(defmap)}
		ac.args(ctx, []Value{ease(ctx, []string{"a", "b", "c"})})
		if _automatic(&ac) != &ac {
			ctx.err("%v", Context(&ac))
		} else if len(ac.defs) != 1 { // maxDigitAutoNum
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if s := d.value.String(); s != "'a' 'b' 'c'" {
			ctx.err("%v → %s", tst{d.value}, s)
		} else if s := d.value.string(ctx); s != "a b c" {
			ctx.err("%v → %s", tst{d.value}, s)
		} else if d, _ := ac.do(ctx, find_auto{"1"}).(*def); d == nil {
			ctx.err("%v", ac.defs)
		} else if d := auto_find(&ac, "1"); d == nil {
			ctx.err("%v", ac.defs)
		} else if v := auto_get(&ac, "1"); v == nil {
			ctx.err("%v", ac.defs)
		} else if s := v.string(ctx); s != "a b c" {
			ctx.err("%v → %s", tst{v}, s)
		} else if d, v := ac.amend(ctx, "1", ease(ctx, "a")); d == nil {
			ctx.err("%v", ac.defs)
		} else if v == nil {
			ctx.err("%v", ac.defs)
		} else if v.String() != "'a' 'b' 'c'" {
			ctx.err("%v %v", ac.defs, v)
		} else if d.value == nil {
			ctx.err("%v %v", ac.defs, d)
		} else if d.value.String() != "'a'" {
			ctx.err("%v %v", ac.defs, d)
		} else if false { for i := 2; i <= maxDigitAutoNum; i += 1 {
			if d, y := ac.defs[strconv.Itoa(i)]; !y {
				ctx.err("%d: %v", i, ac.defs)
			} else if d.value != nil {
				ctx.err("%v", d)
			}
		}}
	}
	{
		ac := automatic{Context:ctx, defs:make(defmap)}
		ac.args(ctx, ease(ctx, []string{"a", "b", "c"}).(*list).elems)
		if len(ac.defs) != 3 { // maxDigitAutoNum
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "a" {
			ctx.err("%v", tst{d.value})
		} else if d, y := ac.defs["2"]; !y {
			ctx.err("2: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "b" {
			ctx.err("%v", tst{d.value})
		} else if d, y := ac.defs["3"]; !y {
			ctx.err("3: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "c" {
			ctx.err("%v", tst{d.value})
		} else if false { for i := 4; i <= maxDigitAutoNum; i += 1 {
			if d, y := ac.defs[strconv.Itoa(i)]; !y {
				ctx.err("%d: %v", i, ac.defs)
			} else if d.value != nil {
				ctx.err("%v", d)
			}
		}}
	}

	if d := ctx.def("foo0"); d == nil {
		ctx.err("foo0")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v", l.elems)
	} else if s, t := d.value.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := d.value.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(ctx), ""; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, "a"); v == nil {
			ctx.err("%v", d)
		} else if s, t := ts(v), "{=list {=word a}}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.String(), "a"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "a"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, "1"); v == nil {
			ctx.err("%v", d)
		} else if s, t := ts(v), "{=list {=word 1}}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.String(), "1"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "1"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, "1", "2", "3"); v == nil {
			ctx.err("%v", d)
		} else if s, t := ts(v), "{=list {=word 1} {=word 2} {=word 3}}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.String(), "1 2 3"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(ctx), "1 2 3"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}
	}

	if d := ctx.def("foo1"); d == nil {
		ctx.err("foo1")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if _, y := d.value.(*null); !y {
		ctx.err("%v", tst{d.value})
	} else if s, t := d.value.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := d.value.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=closure {=word foobar}} {=delegate {=unresolved foobar}}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "&(foobar) $(foobar)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=closure {=word foobar}} {=delegate {=unresolved foobar}}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "&(foobar) $(foobar)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*null); !y {
		ctx.err("%v", tst{v})
	}
	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if a, y := t.x.(*builtin); !y {
		ctx.err("%v %v", v, tst{t.x})
	} else if a.scope == _scope(ctx) {
		ctx.err("%v", tst{a})
	} else if a.scope == _project(ctx).scope {
		ctx.err("%v", tst{a})
	} else if s, t := v.String(), "$(auto $(a))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if a := ctx.val(d.name, defExpand1, test_arg{"a", "x"}); a == nil {
		ctx.err("%v", tst{v})
	} else if s, y := a.(*word); !y || s.s != "x" {
		ctx.err("%v", tst{a})
	} else if s, t := a.String(), "x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := a.string(ctx), "x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if true {
		if x, y := v.(*decimal); !y {
			ctx.err("%v", tst{v})
		} else if x.int64 != 2 {
			ctx.err("%v", tst{v})
		}
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%d, %v", l.len(), tst{l})
	} else if _, y := l.elems[0].(*null); !y {
		ctx.err("%v %v", l.elems[0], tst{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v %v", l.elems[1], tst{l.elems[1]})
	} else if n.int64 != 2 {
		ctx.err("%v %v", l.elems[1], tst{l.elems[1]})
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val20"); d == nil {
		ctx.err("val20")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(auto(a=2) $(val10),$(a))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if true {
		if x, y := v.(*decimal); !y {
			ctx.err("%v", tst{v})
		} else if x.int64 != 2 {
			ctx.err("%v", tst{v})
		}
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%d, %v", l.len(), tst{v})
	} else if _, y := l.elems[0].(*null); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val30"); d == nil {
		ctx.err("val30")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(auto(a=3) $(val20))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if true {
		if x, y := v.(*decimal); !y {
			ctx.err("%v", tst{v})
		} else if x.int64 != 2 {
			ctx.err("%v", tst{v})
		}
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%d, %v", l.len(), tst{l})
	} else if _, y := l.elems[0].(*null); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if n.int64 != 2 {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := ts(v), "{=list {=null} {=decimal 2}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val40"); d == nil {
		ctx.err("val40")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if x, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if _, y := x.x.(*def); !y {
		ctx.err("%v", tst{x.x})
	} else if s, t := v.String(), "$(val30)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("bar1"); d == nil {
		ctx.err("bar1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val1" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar2"); d == nil {
		ctx.err("bar2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val2" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar3"); d == nil {
		ctx.err("bar3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val3" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar4"); d == nil {
		ctx.err("bar4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val4" {
		ctx.err("%v", v)
	}

	s := ".test.x0"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a3=3) $(a1)-$(a2)-$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "x-y-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}-{}-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "{}-{}-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.y0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a1=x a2=y a3=z) $(.test.x0)-$(a1)$(a2)$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "x-y-3-xyz"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.y1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.z0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto $(.test.x0)-$(a1)$(a2)$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "x-y-3-xy{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "x-y-3-xy"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.z1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}-{}-3-{}{}{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "{}-{}-3-{}{}{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testClosure(ctx *testcase) {
	s := "foo_pre"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := "{=closure {=def foo.pre}}", ts(v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := "&(foo.pre)", v.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := "foo", v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_pos"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := "{=closure {=compound {=word foo} {=punct .} {=word pos}}}", ts(v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := "&(foo.pos)", v.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := "foo", v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_z"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=null}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=delegate {=closure {=compound {=word foo} {=punct .} {=word tail}}}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "$(&(foo.tail))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := "{=null}", ts(v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := "{}", v.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=null}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=word foo}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
}

func testDisjunction(ctx *testcase) {
	s := "val1"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo={&(.test.a)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=word foo}={=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo={&(.test.a)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=word a}} {=pair {=word foo}={=word b}} {=pair {=word foo}={=word c}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=word bar}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "{&(.test.a)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}={=word bar}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "{&(.test.a)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=list {=pair {=word a}={=word bar}} {=pair {=word b}={=word bar}} {=pair {=word c}={=word bar}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=word foo}={=compound {=word bar} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo=bar{&(.test.a)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=word foo}={=compound {=word bar} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo=bar{&(.test.a)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=list {=pair {=word foo}={=compound {=word bar} {=word a}}} {=pair {=word foo}={=compound {=word bar} {=word b}}} {=pair {=word foo}={=compound {=word bar} {=word c}}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=compound {=word foo} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}={=word bar}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo{&(.test.a)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=pair {=compound {=word foo} {=disjunction {=closure {=compound {=punct .} {=word test} {=punct .} {=word a}}}}}={=word bar}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "foo{&(.test.a)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := ts(v), "{=list {=pair {=compound {=word foo} {=word a}}={=word bar}} {=pair {=compound {=word foo} {=word b}}={=word bar}} {=pair {=compound {=word foo} {=word c}}={=word bar}}}"; s != t {
		ctx.err("%s : %s != %s", v, s, t)
	} else if s, t := v.String(), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
}

func testValues1(ctx *testcase) {
	if v := ctx.val(".test.foo"); v == nil {
		ctx.err(".test.foo")
	} else if s := v.string(ctx); s != "-foo" {
		ctx.err("%v → %s", v, s)
	} else if s = v.String(); s != "-foo" {
		ctx.err("%v → %s", v, s)
	}
}

func testValues2(ctx *testcase) {
	s := ".test.ab"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-$1-$2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo_ab--"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.ba"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-$2-$1"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo_ba--"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=decimal 1} {=delegate {=builtin closure} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}}} {=decimal 10} {=decimal 2} {=delegate {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}} {=list {=compound {=delegate {=auto 1}} {=delegate {=auto 1}}}} {=list {=compound {=delegate {=auto 2}} {=delegate {=auto 2}}}}} {=decimal 20} {=decimal 3} {=delegate {=builtin call} {=list {=closure {=def .test.x}}} {=list {=compound {=delegate {=auto 1}} {=delegate {=auto 2}}}} {=list {=compound {=delegate {=auto 2}} {=delegate {=auto 1}}}}} {=delegate {=builtin call} {=[] {=flag {=word closure}}} {=list {=closure {=def .test.x}}} {=list {=compound {=delegate {=auto 1}} {=delegate {=auto 2}}}} {=list {=compound {=delegate {=auto 2}} {=delegate {=auto 1}}}}} {=decimal 4} {=delegate {=auto 3}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 $(closure &(.test.x)) 10 2 $(&(.test.x) $1$1,$2$2) 20 3 $(call &(.test.x),$1$2,$2$1) $(call(-closure) &(.test.x),$1$2,$2$1) 4 $3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-- 20 3 foo_ab-- foo_ab-- 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word a} {=flag {=compound {=word b} {=word b}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}}} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}}} {=decimal 4} {=word c}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba foo_ab-ab-ba 4 c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba foo_ab-ab-ba 4 c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word a} {=flag {=compound {=word b} {=word b}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}}} {=compound {=word foo_ab} {=flag {=compound {=word a} {=word b} {=flag {=compound {=word b} {=word a}}}}}} {=decimal 4} {=word c}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba foo_ab-ab-ba 4 c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba foo_ab-ab-ba 4 c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=decimal 1} {=closure {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}} {=decimal 11} {=decimal 2} {=decimal 21} {=decimal 3} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(&(.test.x)) 11 2 21 3 foo_ba-{}{}-{}{} foo_ba-{}{}-{}{} 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 11 2 21 3 foo_ba-- foo_ba-- 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 11} {=decimal 2} {=decimal 21} {=decimal 3} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 11 2 21 3 foo_ba-{}{}-{}{} foo_ba-{}{}-{}{} 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 11 2 21 3 foo_ba-- foo_ba-- 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=decimal 1} {=decimal 12} {=decimal 2} {=decimal 22} {=decimal 3} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 12 2 22 3 foo_ba-{}{}-{}{} foo_ba-{}{}-{}{} 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 12 2 22 3 foo_ba-- foo_ba-- 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=decimal 1} {=decimal 12} {=decimal 2} {=decimal 22} {=decimal 3} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 12 2 22 3 foo_ba-{}{}-{}{} foo_ba-{}{}-{}{} 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 12 2 22 3 foo_ba-- foo_ba-- 4"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.s0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 11 2 21 s0"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.s1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21} {=word s0}} {=word s1}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 11 2 21 s0 s1"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21} {=word s0}} {=word s1}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 11 2 21 s0 s1"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.s2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=strval {=list {=list {=decimal 1} {=decimal 11} {=decimal 2} {=decimal 21}} {=word s0}}}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.String(), "{=str 1 11 2 21 s0}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 11 2 21 s0"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.s3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=strval {=word a} {=word b} {=word c}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "{=str a b c}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "a b c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.t1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}} {=decimal 10} {=decimal 2} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(&(.test.x)) 10 2 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=compound {=punct .} {=word test} {=punct .} {=word ab}}} {=decimal 10} {=decimal 2} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=compound {=word foo_ab} {=flag {=compound {=null} {=flag {=null}}}}} {=decimal 10} {=decimal 2} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 foo_ab-{}- 10 2 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.t2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ba) 10 2 foo_ba-{}{}-{}{} 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ba-- 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ba) 10 2 foo_ba-{}{}-{}{} 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ba-- 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.t3"
	d = ctx.def(s)
	if d == nil || d.value == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ba) 10 2 foo_ba-{}{}-{}{} 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ba-- 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ba}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ba) 10 2 foo_ba-{}{}-{}{} 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ba-- 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=compound {=word foo_ba} {=flag {=compound {=null} {=flag {=null}}}}} {=decimal 10} {=decimal 2} {=compound {=word foo_ba} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 foo_ba-{}- 10 2 foo_ba-{}{}-{}{} 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ba-- 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.t4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=delegate {=def .test.0}} {=punct .} {=delegate {=auto 3}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "$(.test.0) . $3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-- 20 3 foo_ab-- foo_ab-- 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}} {=punct .} {=word x}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} foo_ab-{}{}-{}{} 4 . x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-- 20 3 foo_ab-- foo_ab-- 4 . x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.t5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}} {=punct .}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} foo_ab-{}{}-{}{} 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-- 20 3 foo_ab-- foo_ab-- 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}} {=punct .}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} foo_ab-{}{}-{}{} 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-- 20 3 foo_ab-- foo_ab-- 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.t6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}} {=punct .}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} foo_ab-{}{}-{}{} 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-- 20 3 foo_ab-- foo_ab-- 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {=list {=decimal 1} {=closure {=def .test.ab}} {=decimal 10} {=decimal 2} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 20} {=decimal 3} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=compound {=word foo_ab} {=flag {=compound {=null} {=null} {=flag {=compound {=null} {=null}}}}}} {=decimal 4}} {=punct .}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", v)
	} else if s, t := v.String(), "1 &(.test.ab) 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} foo_ab-{}{}-{}{} 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(ctx), "1 foo_ab-- 10 2 foo_ab-- 20 3 foo_ab-- foo_ab-- 4 ."; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
}

func testValues3(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$1"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a"); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues4(ctx *testcase) {
	s := ".test.*"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "D.c(-unique) D.c++(-unique) I.c(-unique) I.c++(-unique)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 4 {
		ctx.err("%d, %v", l.len(), tst{l})
	} else if _, y := l.elems[0].(*argumented); !y || l.elems[0].String() != "D.c(-unique)" {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(*argumented); !y || l.elems[1].String() != "D.c++(-unique)" {
		ctx.err("%v", tst{l.elems[1]})
	} else if _, y := l.elems[2].(*argumented); !y || l.elems[2].String() != "I.c(-unique)" {
		ctx.err("%v", tst{l.elems[2]})
	} else if _, y := l.elems[3].(*argumented); !y || l.elems[3].String() != "I.c++(-unique)" {
		ctx.err("%v", tst{l.elems[3]})
	}

	s = ".test.D.c.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=compound {=word c} {=punct .} {=word D}} {=delegate {=builtin value} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x}}}}} {=delegate {=builtin value} {=list {=compound {=punct .} {=word test} {=punct .} {=word v}}}} {=closure {=builtin value} {=list {=delegate {=def .test.x}}}} {=delegate {=def .test.foreach} {=list {=delegate {=auto 1}}} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word none}}}}} {=group {=delegate {=auto 1}}} {=delegate {=builtin foreach} {=list {=delegate {=auto 1}}} {=list {=closure {=compound {=punct .} {=word test} {=punct .} {=word x} {=punct .} {=delegate {=auto _}}}}}} {=group {=delegate {=auto 1}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.D $(value &(.test.x)) $(value .test.v) &(value $(.test.x)) $(.test.foreach $1,&(.test.none)) ($1) $(foreach $1,&(.test.x.$_)) ($1)"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(ctx), "c.D xx xx xx () () ()"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 8 {
		ctx.err("%d, %v", l.len(), tst{l})
	}

	s = ".test.D.c.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=compound {=word c} {=punct .} {=word D}} {=closure {=builtin value} {=list {=compound {=punct .} {=word test} {=punct .} {=word v}}}} {=group {=null}} {=group {=null}} {=group {=null}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.D &(value .test.v) ({}) ({}) ({})"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(ctx), "c.D xx () () ()"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 5 {
		ctx.err("%d, %v", l.len(), tst{l})
	}

	s = ".test.D.c++.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "c++.D"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.D.c++.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "c++.D"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.I.c.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=compound {=word c} {=punct .} {=word I}} {=closure {=builtin value} {=list {=closure {=def .test.x}}}} {=closure {=builtin value} {=list {=delegate {=def .test.x}}}} {=delegate {=builtin value} {=list {=closure {=def .test.x}}}} {=delegate {=builtin value} {=list {=delegate {=def .test.x}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.I &(value &(.test.x)) &(value $(.test.x)) $(value &(.test.x)) $(value $(.test.x))"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(ctx), "c.I xx xx xx xx"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 5 {
		ctx.err("%d, %v", l.len(), tst{l})
	}

	s = ".test.I.c.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {=compound {=word c} {=punct .} {=word I}} {=closure {=builtin value} {=list {=closure {=def .test.x}}}} {=closure {=builtin value} {=list {=compound {=punct .} {=word test} {=punct .} {=word v}}}} {=word xx} {=word xx}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.I &(value &(.test.x)) &(value .test.v) xx xx"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(ctx), "c.I xx xx xx xx"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 5 {
		ctx.err("%d, %v", l.len(), tst{l})
	}


	s = ".test.I.c++.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=compound {=word c++} {=punct .} {=word I}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c++.I"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(ctx), "c++.I"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.I.c++.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=compound {=word c++} {=punct .} {=word I}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c++.I"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.string(ctx), "c++.I"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.and.x.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if _, y := v.(*compound); !y {
		ctx.err("%v", v)
	} else if s, t := "x1", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.x.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if _, y := v.(*compound); !y {
		ctx.err("%v", v)
	} else if s, t := "x2", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.y.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if _, y := v.(*compound); !y {
		ctx.err("%v", v)
	} else if s, t := "y1", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.y.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if _, y := v.(*compound); !y {
		ctx.err("%v", v)
	} else if s, t := "y2", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues5(ctx *testcase) {
	s := ".test.0"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x0 $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "z-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-a"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testValues6(ctx *testcase) {
	s := ".test"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "z-y-x-a" {
		ctx.err("%v", d)
	} else if v := ctx.val(d.name, defExpand1, "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "z-y-x-a", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues7(ctx *testcase) {
	if d := ctx.def(".test.z"); d == nil {
		ctx.err(".test.z")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "z-$1-$2"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test.y"); d == nil {
		ctx.err(".test.y")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.z y$1,y$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test.x"); d == nil {
		ctx.err(".test.x")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.y x$1,x$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x $1,$2)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "z-yxa-yxb", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues8(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test$1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, ".u"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testValues9(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x .$1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := v.string(ctx); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testValues10(ctx *testcase) {
}

func testValues11(ctx *testcase) {
	if d := ctx.def(".test.0"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test$1)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1, []string{".v1",".v2"}); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "&(.test.v1) &(.test.v2)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "foo foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test{})"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}
}

func testValues12(ctx *testcase) {
	if d := ctx.def(".test.0"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.w)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	if d := ctx.def(".test.1"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.{})"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.string(ctx), "www"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testValues13(ctx *testcase) {
	if d := ctx.def("foo"); d == nil {
		ctx.err("foo")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(-g!foobar)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := "&(-g!foobar)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "not-foobar", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testPlaceholders(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if x, y := v.(*list); !y {
		ctx.err("%v", v)
	} else if x.len() != 9 {
		ctx.err("%v", x)
	} else if s, t := "$1 $2 $3 $4 $5 $6 $7 $8 $9", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "_", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "1 2 3 4 5 6 7 8 9", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "1 2 3 4 5 6 7 8 9", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", v)
	} else if s, t := "$(foreach a b c d e f,$_)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "a b c d e f", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", v)
	} else if s, t := "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "1 2 3 4 5 6 7 8 9", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", v)
	} else if len(l.elems) != 6 {
		ctx.err("%v", l)
	} else if s, t := "a b c d e f", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, ts(v))
	} else if s, t := v.string(ctx), ""; s != t {
		ctx.err("%s != %s : %s", s, t, ts(v))
	}

	if d := ctx.def("foo1"); d == nil {
		ctx.err("foo1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val1" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo2"); d == nil {
		ctx.err("foo2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val2" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo3"); d == nil {
		ctx.err("foo3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val3" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo4"); d == nil {
		ctx.err("foo4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val4" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo5"); d == nil {
		ctx.err("foo5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val5" {
		ctx.err("%v", v)
	}
}

func testOptional(ctx *testcase) {
	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=project foo}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=self foo}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=self foo}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=self foo}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if d.value == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(d.value), "{=null}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := v.string(ctx), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val11"); d == nil {
		ctx.err("val11")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := v.string(ctx), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val12"); d == nil {
		ctx.err("val12")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := v.string(ctx), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}
}

func testGlobMatch(ctx *testcase) {
	if a, b, c := globMatch(ctx, "*.c", "foo.c"); !a || c != nil {
		ctx.err("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if b[0] != "foo" {
		ctx.err("glob(*.c, foo.c): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**.c", "foo.c"); !a || c != nil {
		ctx.err("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if b[0] != "foo" {
		ctx.err("glob(*.c, foo.c): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "*.c", "foo/bar.c"); a == true || c != nil {
		ctx.err("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if len(b) != 0 {
		ctx.err("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**.c", "foo/bar.c"); !a || c != nil {
		ctx.err("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		ctx.err("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "*", "foobar"); !a || c != nil {
		ctx.err("glob(*, foobar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(*, foobar): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**", "foobar"); !a || c != nil {
		ctx.err("glob(**, foobar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(**, foobar): %v %v %v", a, b, c)
	} else if b[0] != "foobar" {
		ctx.err("glob(**, foobar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "*", "foobar/"); a == true || c != nil {
		ctx.err("glob(*, foobar/): %v %v %v", a, b, c)
	} else if len(b) != 0 {
		ctx.err("glob(*, foobar/): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**", "foobar/"); !a || c != nil {
		ctx.err("glob(**, foobar/): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(**, foobar/): %v %v %v", a, b, c)
	} else if b[0] != "foobar/" {
		ctx.err("glob(**, foobar/): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**", "foo/bar/"); !a || c != nil {
		ctx.err("glob(**, foo/bar/): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(**, foo/bar/): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar/" {
		ctx.err("glob(**, foo/bar/): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**xx**", "foo/bar/xx/foo/bar"); !a || c != nil {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 2 {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar/" {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "/foo/bar" {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**/xx/**", "foo/bar/xx/foo/bar"); !a || c != nil {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 2 {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "foo/bar" {
		ctx.err("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**/??/**", "foo/bar/xx/foo/bar"); !a || c != nil {
		ctx.err("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 4 {
		ctx.err("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		ctx.err("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "x" {
		ctx.err("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[2] != "x" {
		ctx.err("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[3] != "foo/bar" {
		ctx.err("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**/[xyz]/**", "foo/bar/z/foo/bar"); !a || c != nil {
		ctx.err("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 3 {
		ctx.err("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		ctx.err("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "z" {
		ctx.err("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[2] != "foo/bar" {
		ctx.err("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "foo/???/bar", "foo/xyz/bar"); !a || c != nil {
		ctx.err("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if len(b) != 3 {
		ctx.err("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[0] != "x" {
		ctx.err("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[1] != "y" {
		ctx.err("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[2] != "z" {
		ctx.err("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "foo/[xyz]/bar", "foo/z/bar"); !a || c != nil {
		ctx.err("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		ctx.err("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	} else if b[0] != "z" {
		ctx.err("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	}
}

func testGlob(ctx *testcase) { testGlobMatch(ctx)
	if val := ctx.val("pat1.0"); val == nil {
		ctx.err("pat1.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v : %v", tst{val}, val)
	} else if p.len() != 2 {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/x**y"; s != t {
		ctx.err("%v: %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=word x} {=globmeta **} {=word y}}}"; s != t {
		ctx.err("%v: %s != %s", val, s, t)
	} else if v := ctx.val("pat1.1"); v == nil {
		ctx.err("pat1.1: %v", _project(ctx))
	} else if a, b0, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx-yyy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx-yy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat1.2"); v == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if a, b0, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b, y := b0.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", v, p, a, b0, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx-yyx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx-yyx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat1.3"); v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx-yyx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "y" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx-yyx/" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat1.4"); v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx-yyx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "z" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx-yyx/z" { // NOTE: not full-match
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat1.5"); v == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 6 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[5] != "yyy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx/a/b/c/yy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat1.6"); v == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "x" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "xx-yy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "/xx-y" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat2.0"); val == nil {
		ctx.err("pat2.0: %v", _project(ctx))
	} else if s, t := val.String(), ".test/x**y/z"; s != t {
		ctx.err("%v: %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=word x} {=globmeta **} {=word y}} {=word z}}"; s != t {
		ctx.err("%v: %s != %s", val, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat2.1"); v == nil {
		ctx.err("pat2.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v : %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v : %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v : %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx-yyy" {
		ctx.err("%v %v : %v %v %v", p, v, a, b, c)
	} else if b[2] != "z" {
		ctx.err("%v %v : %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v : %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx-yy" {
		ctx.err("%v %v : %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat2.2"); v == nil {
		ctx.err("pat2.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v : %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 7 {
		ctx.err("%v %v : %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v : %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[5] != "yyy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[6] != "z" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx/a/b/c/yy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat3.0"); val == nil {
		ctx.err("pat3.0: %v", _project(ctx))
	} else if val.String() != ".test/x**y/x**" {
		ctx.err("%v", tst{val})
	} else if val.string(ctx) != ".test/x**y/x**" {
		ctx.err("%v: %s", tst{val}, val.string(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat3.1"); v == nil {
		ctx.err("pat3.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 7 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xaaa" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "bbb" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "ccc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "y" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[5] != "xxx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[6] != "xx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "aaa/bbb/ccc/" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "xx/xx" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat3.2"); v == nil {
		ctx.err("pat3.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xaabbccy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "xabc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "aabbcc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "abc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat4.0"); val == nil {
		ctx.err("pat4.0: %v", _project(ctx))
	} else if s, t := val.String(), ".test/x**y/x**y"; s != t {
		ctx.err("%v: %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=word x} {=globmeta **} {=word y}} {=globpat {=word x} {=globmeta **} {=word y}}}"; s != t {
		ctx.err("%v: %s != %s", val, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat4.1"); v == nil {
		ctx.err("pat4.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 7 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xaa" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "bb" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "ccy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "xaa" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[5] != "bb" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[6] != "ccy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "aa/bb/cc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "aa/bb/cc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat4.2"); v == nil {
		ctx.err("pat4.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 5 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xaaay" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "x" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "aaa" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "y" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "aaa" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "/aaa/" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat5.0"); val == nil {
		ctx.err("pat5.0: %v", _project(ctx))
	} else if s, t := val.String(), ".test/x**/**y"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=word x} {=globmeta **}} {=globpat {=globmeta **} {=word y}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat5.1"); v == nil {
		ctx.err("pat5.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xabc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "abcy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "abc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "abc" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat5.2"); v == nil {
		ctx.err("pat5.2: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 5 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xa" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "dy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "d" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat6.0"); val == nil {
		ctx.err("pat6.0: %v", _project(ctx))
	} else if s, t := val.String(), ".test/**y/**y"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=globmeta **} {=word y}} {=globpat {=globmeta **} {=word y}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat6.1"); v == nil {
		ctx.err("pat6.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 8 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "cy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[5] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[6] != "c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[7] != "y" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "a/b/c/" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat7.0"); val == nil {
		ctx.err("pat7.0: %v", _project(ctx))
	} else if s, t := val.String(), ".test/**y/**y/z"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=globmeta **} {=word y}} {=globpat {=globmeta **} {=word y}} {=word z}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat7.1"); v == nil {
		ctx.err("pat7.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 9 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "cy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[5] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[6] != "c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[7] != "y" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[8] != "z" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "a/b/c/" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat8.0"); val == nil {
		ctx.err("pat8.0: %v", _project(ctx))
	} else if s, t := val.String(), ".test/**/**z"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=globmeta **}} {=globpat {=globmeta **} {=word z}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat8.1"); v == nil {
		ctx.err("pat8.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 5 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[4] != "xyz" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a/b/c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "xy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat10.0"); val == nil {
		ctx.err("pat10.0: %v", _project(ctx))
	} else if val.String() != ".test/*.h" {
		ctx.err("%v", tst{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat10.1"); v == nil {
		ctx.err("pat10.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat10.2"); v == nil {
		ctx.err("pat10.2: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.(string); !y {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 0 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat10.3"); v == nil {
		ctx.err("pat10.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.(string); !y {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 0 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat11.0"); val == nil {
		ctx.err("pat11.0: %v", _project(ctx))
	} else if val.String() != ".test/*/*.h" {
		ctx.err("%v", tst{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat11.1"); v == nil {
		ctx.err("pat11.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat11.2"); v == nil {
		ctx.err("pat11.2: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "b.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat11.3"); v == nil {
		ctx.err("pat11.3: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat12.0"); val == nil {
		ctx.err("pat12.0: %v", _project(ctx))
	} else if s, t := val.String(), ".test/*/*/*.h"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {=punct .} {=word test}} {=globpat {=globmeta *}} {=globpat {=globmeta *}} {=globpat {=globmeta *} {=punct .} {=word h}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat12.1"); v == nil {
		ctx.err("pat12.1: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat12.2"); v == nil {
		ctx.err("pat12.2: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "b.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "b.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat12.3"); v == nil {
		ctx.err("pat12.3: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 4 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[3] != "c.h" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[2] != "c" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat12.4"); v == nil {
		ctx.err("pat12.4: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat12.5"); v == nil {
		ctx.err("pat12.5: %v", _project(ctx))
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 2 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "a" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[1] != "b" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat.0"); val == nil {
		ctx.err("pat.0: %v", _project(ctx))
	} else if val.string(ctx) != "**.auto" {
		ctx.err("%v", tst{val})
	} else if v := ctx.val("pat13.1"); v == nil {
		ctx.err("pat13.1: %v", _project(ctx))
	} else if a, b, c := val.match(ctx, v); !a {
		ctx.err("%v ; %v %v %v", val, a, b, c)
	} else if v := ctx.val("pat13.2"); v == nil {
		ctx.err("pat13.2: %v", _project(ctx))
	} else if a, b, c := val.match(ctx, v); a {
		ctx.err("%v ; %v %v %v", tst{val}, a, b, c)
	}
}
