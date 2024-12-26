//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strconv"
	"strings"
)

type testValueGeneralStruct struct {
	assert_bool bool
	assert_value Value
}
func testValueGeneralAssertHook(ctx Context, v Value, b bool, i any) {
	st := i.(*testValueGeneralStruct)
	st.assert_bool = b
	st.assert_value = v
}
func testValueGeneral(ctx testcase1) {
	st := ctx.i.(*testValueGeneralStruct)
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
	} else if n, y := l.elems[11].(*none); !y || n.String() != `{=none anything goes}` || n.string(ctx) != `` {
		ctx.err("%v", tst{l.elems[11]})
	} else if t, y := n.x.(*list); !y {
		ctx.err("%v", tst{n.x})
	} else if len(t.elems) != 2 {
		ctx.err("%v", tst{n.x})
	} else if t.elems[0].String() != "anything" {
		ctx.err("%v", tst{t.elems[0]})
	} else if t.elems[1].String() != "goes" {
		ctx.err("%v", tst{t.elems[1]})
	} else if n, y := l.elems[12].(*null); !y || n.String() != `{}` || n.string(ctx) != `` {
		ctx.err("%v", tst{l.elems[12]})
	} else if s, t := `{word foo/bar 'strlit' "strcomp" 0 1} {=yes} {=false} {=true} {=path foo} foo/bar {=file foobar} **.c {=regex xx} 1 0.1 {=none anything goes} {}`, val.String(); s != t {
		ctx.err("%v != %s", tst{val}, s)
	} else if s, t := `word foo/bar strlit strcomp 0 1 yes false true foo foo/bar foobar **.c xx 1 0.1`, val.string(ctx); s != t {
		ctx.err("%v: %s != %s", tst{val}, t, s)
	}

	if d := ctx.def("disjunction0"); d == nil {
		ctx.err("disjunction0")
	} else if d.value == nil {
		ctx.err("%v", tst{d})
	} else if t, y := d.value.(*compound); !y {
		ctx.err("%v", tst{d.value})
	} else if len(t.elems) != 2 {
		ctx.err("%v", tst{d.value})
	} else if v, y := t.elems[0].(*word); !y {
		ctx.err("%v %v", tst{v}, v.s)
	} else if v, y := t.elems[1].(disjunction); !y {
		ctx.err("%v", tst{v.Value})
	} else if s, t := v.String(), "{a b c}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := v.Value.String(), "a b c"; s != t {
		ctx.err("%v → %s != %s", tst{v.Value}, t, s)
	} else if s, t := d.value.String(), "x{a b c}"; t != s {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	} else if s, t := d.value.string(ctx), "xa xb xc"; t != s {
		ctx.err("%v → %s != %s", tst{d.value}, t, s)
	}

	if d := ctx.def("disjunction1"); d == nil {
		ctx.err("disjunction1")
	} else if d.value == nil {
		ctx.err("%v", tst{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.String(); s != t {
		ctx.err("%v: %s != %s", tst{l}, t, s)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.string(ctx); s != t {
		ctx.err("%v: %s != %s", tst{l}, t, s)
	}

	if d := ctx.def("disjunction2"); d == nil {
		ctx.err("disjunction2")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.String(); s != t {
		ctx.err("%v: %s != %s", tst{l}, t, s)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.string(ctx); s != t {
		ctx.err("%v: %s != %s", tst{l}, t, s)
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
		ctx.err("%v : %v", tst{val}, val)
	} else if p, y := val.(*compound); !y {
		ctx.err("%v : %v", tst{val}, val)
	} else if len(p.elems) != 11 {
		ctx.err("%v: %v %v", typeof(val), val, p.elems)
	}

	if val := ctx.val("val5"); val == nil {
		ctx.err("val5")
	} else if val.String() != `'"a,b,c x,y,z"'` {
		ctx.err("%v", tst{val})
	} else if val.string(ctx) != `"a,b,c x,y,z"` {
		ctx.err("%v", tst{val})
	} else if _, y := val.(*strlit); !y {
		ctx.err("%v", tst{val})
	}

	// TODO: test regexp.stencil(...)

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if tv(v) != "{=url http://extbit.io/help}" {
		ctx.err("%v %v", tst{d}, tv(v))
	} else if ts(v) != "{=url {=word http} {} {} {=compound {=word extbit} {=punct .} {=word io}} {} {=path {=punct root} {=word help}} {=[]Value} {}}" {
		ctx.err("%v %v", tst{d}, ts(v))
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if tv(v) != "{=url https://extbit.com}" {
		ctx.err("%v %v", tst{d}, tv(v))
	} else if ts(v) != "{=url {=word https} {} {} {=compound {=word extbit} {=punct .} {=word com}} {} {} {=[]Value} {}}" {
		ctx.err("%v %v", tst{d}, ts(v))
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if tv(v) != "{=url https://extbit.com?foo=x}" {
		ctx.err("%v %v", tst{d}, tv(v))
	} else if ts(v) != "{=url {=word https} {} {} {=compound {=word extbit} {=punct .} {=word com}} {} {} {=[]Value {=pair {=word foo}={=word x}}} {}}" {
		ctx.err("%v %v", tst{d}, ts(v))
	}

	if d := ctx.def("val9"); d == nil {
		ctx.err("val9")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if tv(v) != "{=url https://extbit.com?foo=x&bar=y#foobar}" {
		ctx.err("%v %v", tst{d}, tv(v))
	} else if ts(v) != "{=url {=word https} {} {} {=compound {=word extbit} {=punct .} {=word com}} {} {} {=[]Value {=pair {=word foo}={=word x}} {=pair {=word bar}={=word y}}} {=word foobar}}" {
		ctx.err("%v %v", tst{d}, ts(v))
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if tv(v) != "{=url https://ext.pub?foo=x+y+z&bar=x%20y%20z#foo+bar}" {
		ctx.err("%v, %v", tst{d}, tv(v))
	} else if ts(v) != "{=url {=word https} {} {} {=compound {=word ext} {=punct .} {=word pub}} {} {} {=[]Value {=pair {=word foo}={=word x+y+z}} {=pair {=word bar}={=compound {=word x} {=punct %} {=decimal 20} {=word y} {=punct %} {=decimal 20} {=word z}}}} {=word foo+bar}}" {
		ctx.err("%v, %v", tst{d}, ts(v))
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
	} else if s, t := "foo.o foo-x.o foo-x-y.o foo-x-y-z.o foobar.o", v.string(ctx); s != t {
		ctx.err("%s != %s; %v", s, t, v)
	}

	s := "foo.o foo . o "
	s += "foo-x.o foo -x -x x . o "
	s += "foo-x-y.o foo -x-y -y y . o "
	s += "foo-x-y-z.o foo -x-y-z -z z . o "
	s += "foobar.o foobar . o"
	if d := ctx.def("conf1"); d == nil {
		ctx.err("conf1")
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
	} else if false {
		note(pc(ctx,v), "%v", v).debug()
	}
}

func testAutomatic(ctx *testcase) {
	if c := _universe(ctx); c == nil {
		ctx.err("context.cast")
	}
	{
		ac := automatic{Context:ctx, defs:make(defs_map)}
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
		ac := automatic{Context:ctx, defs:make(defs_map)}
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

	if d := ctx.def("foo"); d == nil {
		ctx.err("foo")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v", l.elems)
	} else if s := "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != d.value.String() {
		ctx.err("%v != %s", tst{d.value}, s)
	} else if s := d.value.string(ctx); s != "" {
		ctx.err("%v ; %s", tst{d.value}, s)
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s, t := "$1 $2 $3 $4 $5 $6 $7 $8 $9", v.String(); s != t {
			ctx.err("%v: %s != %s", v, s, t)
		} else if t := v.string(ctx); t != "" {
			ctx.err("%v: %s", v, t)
		}

		if v := ctx.val(d.name, "a"); v == nil {
			ctx.err("%v", d)
		} else if s, t := "a $2 $3 $4 $5 $6 $7 $8 $9", v.String(); s != t {
			ctx.err("%v: %s != %s", v, t, s)
		} else if s, t := "a", v.string(ctx); s != t {
			ctx.err("%v: %s != %s", v, t, s)
		}

		if v := ctx.val(d.name, "1"); v == nil {
			ctx.err("%v", d)
		} else if s, t := "1 $2 $3 $4 $5 $6 $7 $8 $9", v.String(); s != t {
			ctx.err("%v: %s != %s", v, t, s)
		} else if s, t := "1", v.string(ctx); s != t {
			ctx.err("%v: %s != %s", v, t, s)
		}

		if v := ctx.val(d.name, "1", "2", "3"); v == nil {
			ctx.err("%v", d)
		} else if s, t := "1 2 3 $4 $5 $6 $7 $8 $9", v.String(); s != t {
			ctx.err("%v: %s != %s", v, t, s)
		} else if s, t := "1 2 3", v.string(ctx); s != t {
			ctx.err("%v: %s != %s", v, t, s)
		}
	}

	if d := ctx.def("val"); d == nil {
		ctx.err("val")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "&(foobar)", v.String(); s != t {
		ctx.err("%v → %s != %s", v, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", v, s, t)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := "&(foobar)", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if a, y := t.x.(*auto); !y {
		ctx.err("%v %v", tst{t.x}, a)
	} else if a.scope.outer != _scope(ctx) {
		ctx.err("%v", tst{a})
	} else if a.scope.outer != _project(ctx).scope {
		ctx.err("%v", tst{a})
	} else if a.scope == _scope(ctx) {
		ctx.err("%v", tst{a})
	} else if a.scope == _project(ctx).scope {
		ctx.err("%v", tst{a})
	} else if s, t := "$(a)", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if a := ctx.val(d.name, test_arg{"a", "x"}); a == nil {
		ctx.err("%v", tst{v})
	} else if s, y := a.(*word); !y || s.s != "x" {
		ctx.err("%v", tst{a})
	} else if s, t := "x", a.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if t := a.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v", tst{v})
	} else if n, y := l.elems[0].(*decimal); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if n.int64 != 1 {
		ctx.err("%v", tst{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if n.int64 != 1 {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := "1 1", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v", tst{v})
	} else if n, y := l.elems[0].(*decimal); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if n.int64 != 1 {
		ctx.err("%v", tst{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if n.int64 != 1 {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := "1 1", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v", tst{v})
	} else if n, y := l.elems[0].(*decimal); !y {
		ctx.err("%v", tst{l.elems[0]})
	} else if n.int64 != 1 {
		ctx.err("%v", tst{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v", tst{l.elems[1]})
	} else if n.int64 != 1 {
		ctx.err("%v", tst{l.elems[1]})
	} else if s, t := "1 1", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
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
}

func testClosure(ctx *testcase) {
	s := "foo_pre"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := "{=closure {=def foo.pre}}", ts(v); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "&(foo.pre)", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "foo", v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}

	s = "foo_pos"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := "{=closure {=compound {=word foo} {=punct .} {=word pos}}}", ts(v); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "&(foo.pos)", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "foo", v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}

	s = "foo_nest_1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := "{=closure {=closure {=compound {=word foo} {=punct .} {=word tail}}}}", ts(v); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "&(&(foo.tail))", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "foo", v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}

	s = "foo_nest_2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := "{=closure {=closure {=def foo.tail}}}", ts(v); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "&(&(foo.tail))", v.String(); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	} else if s, t := "foo", v.string(ctx); s != t {
		ctx.err("%v: %s != %s", v, t, s)
	}
}

func testValues_bug_01(ctx *testcase) {
	s := "okay"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else {
		var v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					erro(ctx, "%s", x.string).trace()
				}
			} ()
			v = d.value.expand(_final(ctx))
		} ()
		if v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "{x.{$1}}? {x.{$2}}? {y.{$1}}? {y.{$2}}? {z.{$1}}? {z.{$2}}?"; s != t {
			ctx.err("%v : %s != %s", d, s, t)
		} else if s, t := ts(v), "{=list {=list {=cond {=disjunction {=compound {=word x} {=punct .} {=disjunction {=delegate {=auto 1}}}}}} {=cond {=disjunction {=compound {=word x} {=punct .} {=disjunction {=delegate {=auto 2}}}}}}} {=list {=cond {=disjunction {=compound {=word y} {=punct .} {=disjunction {=delegate {=auto 1}}}}}} {=cond {=disjunction {=compound {=word y} {=punct .} {=disjunction {=delegate {=auto 2}}}}}}} {=list {=cond {=disjunction {=compound {=word z} {=punct .} {=disjunction {=delegate {=auto 1}}}}}} {=cond {=disjunction {=compound {=word z} {=punct .} {=disjunction {=delegate {=auto 2}}}}}}}}"; s != t {
			ctx.err("%v : %s != %s", d, s, t)
		}
	}

	s = "bug_0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(bug_0.1 $1,$2)"; s != t {
		ctx.err("%v : %s != %s", d, s, t)
	} else {
		var v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					erro(ctx, "%s", x.string).trace()
				}
			} ()
			v = d.value.expand(trace_evoke_loop{_final(ctx)})
		} ()
		if v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "{&(x.{$1})}? {&(x.{$2})}? {&(y.{$1})}? {&(y.{$2})}? {&(z.{$1})}? {&(z.{$2})}?"; s != t {
			ctx.err("%v : %s != %s", d, s, t)
		} else if s, t := ts(v), "{=list {=list {=cond {=disjunction {=closure {=compound {=word x} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}} {=cond {=disjunction {=closure {=compound {=word x} {=punct .} {=disjunction {=delegate {=auto 2}}}}}}}} {=list {=cond {=disjunction {=closure {=compound {=word y} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}} {=cond {=disjunction {=closure {=compound {=word y} {=punct .} {=disjunction {=delegate {=auto 2}}}}}}}} {=list {=cond {=disjunction {=closure {=compound {=word z} {=punct .} {=disjunction {=delegate {=auto 1}}}}}}} {=cond {=disjunction {=closure {=compound {=word z} {=punct .} {=disjunction {=delegate {=auto 2}}}}}}}}}"; s != t {
			ctx.err("%v : %s != %s", d, s, t)
		}
	}

	s = "bug_1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(bug_1.1 $1,$2)"; s != t {
		ctx.err("%v : %s != %s", d, s, t)
	} else {
		var e, v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			v = d.value.expand(trace_evoke_loop{_final(ctx)})
		} ()
		if s, t := ts(e), "{=def bug_1.1}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}
	}

	s = "flags"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s := ts(v); s != "{=delegate {=def .flags} {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}" {
		ctx.err("%v : %v", d, s)
	} else {
		var e, v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			v = d.value.expand(trace_evoke_loop{_final(ctx)})
		} ()
		if s, t := ts(e), "{=def .flags}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}
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
	} else if s, t := "foobar $1-$2", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "foobar -", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "foobar a-b", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.ba"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "foobaz $2-$1", v.String(); s != t {
		ctx.err("%v", v)
	} else if s, t := "foobaz -", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "foobaz b-a", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 8 {
		ctx.err("%v", tst{v})
	} else if x, y := l.elems[7].(*delegate); !y {
		ctx.err("%v", tst{l.elems[7]})
	} else if z, y := x.x.(*auto); !y {
		ctx.err("%v", tst{x.x})
	} else if z.name != "3" {
		ctx.err("%v", tst{z})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) z $(call(-closure) &(.test.x),$1$2,$2$1) Z $3", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar - z foobar - Z", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) z $(call(-closure) &(.test.x),ab,ba) Z c", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z c", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c", test_def_1{}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) z $(call(-closure) &(.test.x),ab,ba) Z c", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z c", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c", test_def_2{}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x foobar $1-$2 y foobar aa-bb z foobar ab-ba Z c", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z c", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.s1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*strval); !y {
		ctx.err("%v", tst{v})
	} else if len(t.v) != 5 {
		ctx.err("%v %v", tst{v}, tst{t.v})
	} else if s, t := "$(string x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "$(string x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.s2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if x, y := v.(*list); !y {
		ctx.err("%v : %v", v, tst{v})
	} else if x.len() != 5 {
		ctx.err("%v : %v", v, tst{v})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.s3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*strlit); !y {
		ctx.err("%v", tst{v})
	} else if s, t := l.s, "a b c"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := `'a b c'`, v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := `a b c`, v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.3"
	d = ctx.def(s)
	if d == nil || d.value == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x foobaz $2-$1 y $(.test.ba $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "x foobaz b-a y foobaz bb-aa cc", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v := ctx.val(d, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) z $(call(-closure) &(.test.x),ab,ba) Z x . x", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z x . x", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v := ctx.val(d, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "x foobar a-b y foobar aa-bb z foobar ab-ba Z x . x", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues3(ctx *testcase) {
	s := ".test.x"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "$(a1)-$(a2)-3", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := "x-y-3", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.y"
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

	s = ".test.z"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "$(a1)-$(a2)-3-$(a1)$(a2)$(a3)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := "$(a1)-$(a2)-3-$(a1)$(a2)$(a3)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "--3-", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
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

func testValues4(ctx *testcase) {
	s := ".test.*"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", v)
	} else if l.len() != 4 {
		ctx.err("%v ; (%d)", tst{l.elems}, l.len())
	} else if _, y := l.elems[0].(*argumented); !y || l.elems[0].String() != "D.c(-unique)" {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(*argumented); !y || l.elems[1].String() != "D.c++(-unique)" {
		ctx.err("%v", tst{l.elems[1]})
	} else if _, y := l.elems[2].(*argumented); !y || l.elems[2].String() != "I.c(-unique)" {
		ctx.err("%v", tst{l.elems[2]})
	} else if _, y := l.elems[3].(*argumented); !y || l.elems[3].String() != "I.c++(-unique)" {
		ctx.err("%v", tst{l.elems[3]})
	} else if s, t := "D.c(-unique) D.c++(-unique) I.c(-unique) I.c++(-unique)", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.D.c"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 9 {
		ctx.err("%v ; (%d)", tst{v}, l.len())
	} else if _, y := l.elems[0].(*word); !y || l.elems[0].String() != "D" {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(*word); !y || l.elems[1].String() != "c" {
		ctx.err("%v", tst{l.elems[1]})
	} else if _, y := l.elems[2].(*delegate); !y || l.elems[2].String() != "$(value &(.test.x))" {
		ctx.err("%v", tst{l.elems[2]})
	} else if t, y := l.elems[3].(*delegate); !y || l.elems[3].String() != "$(.test.v)" {
		ctx.err("%v", tst{l.elems[3]})
	} else if _, y := t.x.(*compound); !y || t.x.String() != ".test.v" {
		ctx.err("%v", tst{t.x})
	} else if t.string(ctx) != "xx" {
		ctx.err("%v", tst{t})
	} else if _, y := l.elems[4].(*closure);  !y || l.elems[4].String() != "&(value .test.v)" {
		ctx.err("%v", tst{l.elems[4]})
	} else if _, y := l.elems[5].(*delegate); !y || l.elems[5].String() != "$(.test.foreach $1,&(.test.none))" {
		ctx.err("%v", tst{l.elems[5]})
	} else if _, y := l.elems[6].(*group);    !y || l.elems[6].String() != "($1)" {
		ctx.err("%v", tst{l.elems[6]})
	} else if _, y := l.elems[7].(*delegate); !y || l.elems[7].String() != "$(foreach $1,&(.test.x.$_))" {
		ctx.err("%v", tst{l.elems[7]})
	} else if _, y := l.elems[8].(*group);    !y || l.elems[8].String() != "($1)" {
		ctx.err("%v", tst{l.elems[8]})
	} else if s, t := v.String(), "D c $(value &(.test.x)) $(.test.v) &(value .test.v) $(.test.foreach $1,&(.test.none)) ($1) $(foreach $1,&(.test.x.$_)) ($1)"; s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %-20v %v", i, tst{v}, v) } }
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := v.string(ctx), "D c xx xx xx () () ()"; s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %-20v %v", i, tst{v}, v) } }
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 9 {
		ctx.err("%v ; (%d)", tst{v}, l.len())
	} else if s, t := v.String(), "D c $(value &(.test.x)) xx &(value .test.v) (a) (a) &(.test.x.a)? (a)"; s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %-20v %v", i, v, tst{v}) } }
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := v.string(ctx), "D c xx xx xx (a) (a) x (a)"; s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %-10v %v", i, v, tst{v}) } }
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d.name, "a", test_def_2{}); v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 9 {
		ctx.err("%v ; (%d)", tst{v}, l.len())
	} else if s, t := "D c xx xx xx $(.test.foreach a,&(.test.none)) (a) x (a)", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v : %v", i, v, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "D c xx xx xx (a) (a) x (a)", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.D.c++"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "D c++", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.I.c"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", v)
	} else if l.len() != 6 {
		ctx.err("%v ; (%d)", v, l.len())
	} else if _, y := l.elems[0].(*word); !y || l.elems[0].String() != "I" {
		ctx.err("%-22v : %v", l.elems[0], tst{l.elems[0]})
	} else if _, y := l.elems[1].(*word); !y || l.elems[1].String() != "c" {
		ctx.err("%-22v : %v", l.elems[1], tst{l.elems[1]})
	} else if _, y := l.elems[2].(*closure);  !y || l.elems[2].String() != "&(value &(.test.x))" {
		ctx.err("%-22v : %v", l.elems[2], tst{l.elems[2]})
	} else if _, y := l.elems[3].(*closure);  !y || l.elems[3].String() != "&(value .test.v)" {
		ctx.err("%-22v : %v", l.elems[3], tst{l.elems[3]})
	} else if _, y := l.elems[4].(*delegate); !y || l.elems[4].String() != "$(value &(.test.x))" {
		ctx.err("%-22v : %v", l.elems[4], tst{l.elems[4]})
	} else if _, y := l.elems[5].(*word); !y || l.elems[5].String() != "xx" {
		ctx.err("%-22v : %v", l.elems[5], tst{l.elems[5]})
	} else if s, t := "I c &(value &(.test.x)) &(value .test.v) $(value &(.test.x)) xx", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "I c xx xx xx xx", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, tst{v}) } }
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.I.c++"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "I c++", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
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
	s := ".test"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "z-$1" {
		ctx.err("%v", d)
	} else if v := ctx.val(d.name, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "z-a", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
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
	} else if v := ctx.val(d.name, "b"); v == nil {
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
	} else if v := ctx.val(d.name, "a", "b"); v == nil {
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
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "$1" {
		ctx.err("%v", d)
	} else if v := ctx.val(d.name, "a"); v == nil {
		ctx.err(".test")
	} else if s, t := "a", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues9(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "$(.test$1)" {
		ctx.err("%v", d)
	}

	if v := ctx.val(".test", ".u"); v == nil {
		ctx.err(".test")
	} else if v.String() != "foobar" {
		ctx.err("%v", v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%v → %s", v, s)
	}
}

func testValues10(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "$(.test.y .$1)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "foobar", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "foobar", v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues11(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "&(.test$1)" {
		ctx.err("%v", d)
	}

	if v := ctx.val(".test", ".v2"); v == nil {
		ctx.err(".test")
	} else if v.String() != "&(.test.v2)" {
		ctx.err("%v", v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%v → %s", v, s)
	}
}

func testValues12(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.y .$1)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "w2"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.w2)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := v.string(ctx), "foobaz"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
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
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "_", "x", "y", "z"); v == nil {
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
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
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
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
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
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", v)
	} else if s, t := "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "1 2 3 4 5 6 7 8 9", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v : %s != %s", v, t, s)
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
	} else if o, y := v.(*project); !y {
		ctx.err("%v", tst{v})
	} else if s, t := ts(v), "{=project foo}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := o.name, "foo"; s != t {
		ctx.err("%v", tst{v})
	} else if t := v.ident(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=cond {=word name}}}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if o, y := t.x.(cond); !y {
		ctx.err("%v", tst{t.x})
	} else if _, y := o.Value.(*word); !y {
		ctx.err("%v", tst{o.Value})
	} else if s, t := v.String(), "$(name?)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(name?)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*project); !y {
		ctx.err("%v", tst{v})
	} else if s, t := p.String(), "{=project "+p.name+"}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if o := v.expand(ctx); o == nil {
		ctx.err("%v → nil", tst{v})
	} else if t := o.String(); s != t {
		ctx.err("%v → %s != %s", tst{o}, t, s)
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := p.name, "foo"; s != t {
		ctx.err("%v", tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if t := v.ident(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if t := v.ident(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=delegate {=cond {=arrow {=project foo}→{=word baz}}}}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if c, y := t.x.(cond); !y {
		ctx.err("%v", tst{t.x})
	} else if _, y := c.Value.(*arrow); !y {
		ctx.err("%v", tst{c.Value})
	} else if s, t := v.String(), "$({=project foo}→baz?)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if _, y := t.x.(*arrow); !y {
		ctx.err("%v", tst{t.x})
	} else if s, t := v.String(), "$(fo?→bar)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(fo?→bar)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if c, y := t.x.(cond); !y {
		ctx.err("%v", tst{t.x})
	} else if _, y := c.Value.(*arrow); !y {
		ctx.err("%v", tst{c.Value})
	} else if s, t := v.String(), "$(fo?→bar?)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(fo?→bar?)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(self); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=self "+p.name+"}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := p.name, "foo"; s != t {
		ctx.err("%v", tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(self); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=self "+p.name+"}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := p.name, "foo"; s != t {
		ctx.err("%v", tst{v})
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$({=project foo}→bar)?"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=project foo}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := v.string(ctx), "foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}

	if d := ctx.def("val11"); d == nil {
		ctx.err("val11")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=project foo}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	} else if s, t := v.string(ctx), "foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
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
