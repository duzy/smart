//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strconv"
)

type testValueGeneralStruct struct {
	assert_bool bool
	assert_value Value
}
func testValueGeneralAssertHook(ctx Context, v Value, b bool, i interface{}) {
	st := i.(*testValueGeneralStruct)
	st.assert_bool = b
	st.assert_value = v
}
func testValueGeneral(ctx testcase1) {
	st := ctx.i.(*testValueGeneralStruct)
	if st.assert_value == nil {
		ctx.err("assert: %v", st.assert_value)
	} else if st.assert_bool {
		ctx.err("assert")
	}

	if d := ctx.def("vals"); d == nil {
		ctx.err("vals")
	} else if val := d.value; val == nil {
		ctx.err("%v", ust{d})
	} else if l, y := val.(*list); !y {
		ctx.err("%v", ust{val})
	} else if len(l.elems) != 13 {
		ctx.err("%v : {%v}", len(l.elems), l)
	} else if v, y := l.elems[0].(disjunction); !y {
		ctx.err("%v", ust{l.elems[0]})
	} else if t, y := v.Value.(*list); !y {
		ctx.err("%v", ust{v.Value})
	} else if len(t.elems) != 6 {
		ctx.err("%v", ust{v.Value})
	} else if w, y := t.elems[0].(*bareword); !y || w.s != "word" {
		ctx.err("%v", ust{t.elems[0]})
	} else if _, y := t.elems[1].(*path); !y {
		ctx.err("%v", ust{t.elems[1]})
	} else if s, y := t.elems[2].(*strlit); !y || s.String() != "'strlit'" || s.string(ctx) != "strlit" {
		ctx.err("%v", ust{t.elems[2]})
	} else if c, y := t.elems[3].(*compound); !y || c.String() != `"compound"` || c.string(ctx) != "compound" {
		ctx.err("%v", ust{t.elems[3]})
	} else if s, t := `word foo/bar strlit compound 0 1`, v.string(ctx); s != t {
		ctx.err("%v: %s != %s", ust{v}, t, s)
	} else if s, t := `{word foo/bar 'strlit' "compound" 0 1}`, v.String(); s != t {
		ctx.err("%v: %s != %s", ust{v}, t, s)
	} else if a, y := l.elems[1].(*answer); !y || a.bool != true {
		ctx.err("%v", ust{l.elems[1]})
	} else if b, y := l.elems[2].(*boolean); !y || b.bool != false {
		ctx.err("%v", ust{l.elems[2]})
	} else if b, y := l.elems[3].(*boolean); !y || b.bool != true {
		ctx.err("%v", ust{l.elems[3]})
	} else if p, y := l.elems[4].(*path); !y || p.String() != "{=path foo}" || p.string(ctx) != "foo" {
		ctx.err("%v", ust{l.elems[4]})
	} else if p, y := l.elems[5].(*path); !y || p.String() != "foo/bar" || p.string(ctx) != "foo/bar" {
		ctx.err("%v", ust{l.elems[5]})
	} else if f, y := l.elems[6].(*File); !y || f.String() != "{=file foobar}" || f.string(ctx) != "foobar" {
		ctx.err("%v", ust{l.elems[6]})
	} else if g, y := l.elems[7].(*globpat); !y || g.String() != "**.c" || g.string(ctx) != "**.c" {
		ctx.err("%v", ust{l.elems[7]})
	} else if r, y := l.elems[8].(*regexpat); !y || r.String() != "{=regex xx}" || r.string(ctx) != "xx" {
		ctx.err("%v", ust{l.elems[8]})
	} else if i, y := l.elems[9].(*decimal); !y || i.String() != "1" || i.string(ctx) != "1" /* || i.int(ctx) != 1 */ {
		ctx.err("%v", ust{l.elems[9]})
	} else if f, y := l.elems[10].(*Float); !y || f.String() != "0.1" {
		ctx.err("%v", ust{l.elems[10]})
	} else if n, y := l.elems[11].(*none); !y || n.String() != `{=none anything goes}` || n.string(ctx) != `` {
		ctx.err("%v", ust{l.elems[11]})
	} else if t, y := n.x.(*list); !y {
		ctx.err("%v", ust{n.x})
	} else if len(t.elems) != 2 {
		ctx.err("%v", ust{n.x})
	} else if t.elems[0].String() != "anything" {
		ctx.err("%v", ust{t.elems[0]})
	} else if t.elems[1].String() != "goes" {
		ctx.err("%v", ust{t.elems[1]})
	} else if n, y := l.elems[12].(*null); !y || n.String() != `{}` || n.string(ctx) != `` {
		ctx.err("%v", ust{l.elems[12]})
	} else if s, t := `{word foo/bar 'strlit' "compound" 0 1} {=yes} {=false} {=true} {=path foo} foo/bar {=file foobar} **.c {=regex xx} 1 0.1 {=none anything goes} {}`, val.String(); s != t {
		ctx.err("%v != %s", ust{val}, s)
	} else if s, t := `word foo/bar strlit compound 0 1 yes false true foo foo/bar foobar **.c xx 1 0.1`, val.string(ctx); s != t {
		ctx.err("%v: %s != %s", ust{val}, t, s)
	}

	if d := ctx.def("disjunction0"); d == nil {
		ctx.err("disjunction0")
	} else if d.value == nil {
		ctx.err("%v", ust{d})
	} else if t, y := d.value.(*barecomp); !y {
		ctx.err("%v", ust{d.value})
	} else if len(t.elems) != 2 {
		ctx.err("%v", ust{d.value})
	} else if v, y := t.elems[0].(*bareword); !y {
		ctx.err("%v %v", ust{v}, v.s)
	} else if v, y := t.elems[1].(disjunction); !y {
		ctx.err("%v", ust{v.Value})
	} else if s, t := "{a b c}", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "a b c", v.Value.String(); s != t {
		ctx.err("%v → %s != %s", ust{v.Value}, t, s)
	} else if s, t := "x{a b c}", d.value.String(); t != s {
		ctx.err("%v → %s != %s", ust{d.value}, t, s)
	} else if s, t := "xa xb xc", d.value.string(ctx); t != s {
		ctx.err("%v → %s != %s", ust{d.value}, t, s)
	}

	if d := ctx.def("disjunction1"); d == nil {
		ctx.err("disjunction1")
	} else if d.value == nil {
		ctx.err("%v", ust{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", ust{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", ust{l}, l.elems)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.String(); s != t {
		ctx.err("%v: %s != %s", ust{l}, t, s)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.string(ctx); s != t {
		ctx.err("%v: %s != %s", ust{l}, t, s)
	}

	if d := ctx.def("disjunction2"); d == nil {
		ctx.err("disjunction2")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", ust{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", ust{l}, l.elems)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.String(); s != t {
		ctx.err("%v: %s != %s", ust{l}, t, s)
	} else if s, t := `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`, l.string(ctx); s != t {
		ctx.err("%v: %s != %s", ust{l}, t, s)
	}

	var (
		glob1 = ctx.val("glob1")
		glob2 = ctx.val("glob2")
		glob3 = ctx.val("glob3")
		regexp1 = ctx.val("regexp1")
		regexp2 = ctx.val("regexp2")
		regexp3 = ctx.val("regexp3")
		regexp4 = ctx.val("regexp4")
		regexp5 = ctx.val("regexp5")
		regexp6 = ctx.val("regexp6")
	)

	if glob1.string(ctx) != "*.c" {
		ctx.err("%v", ust{glob1})
	}

	if glob2.string(ctx) != "**.c" {
		ctx.err("%v", ust{glob2})
	}

	if g, y := glob1.(*globpat); !y {
		ctx.err("%v", ust{glob1})
	} else if g.String() != "*.c" {
		ctx.err("%v ; %v", ust{g}, g)
	} else if g.string(ctx) != "*.c" {
		ctx.err("%v", ust{g})
	}

	if g, y := glob2.(*globpat); !y {
		ctx.err("%v", ust{glob2})
	} else if g.String() != "**.c" {
		ctx.err("%v ; %v", ust{g}, g)
	} else if g.string(ctx) != "**.c" {
		ctx.err("%v", ust{g})
	}

	if g, y := glob3.(*globpat); !y {
		ctx.err("%v", ust{glob3})
	} else if g.String() != "{=glob x*z?.c}" {
		ctx.err("%v ; %v", ust{g}, g)
	} else if g.string(ctx) != "x*z?.c" {
		ctx.err("%v", ust{g})
	}

	if regexp1.string(ctx) != `x{1}, x{1,}, x{1,2}, x{5}?, x{2,}?, x{2,8}? \p{Greek}, \P{Greek}` {
		ctx.err("regexp1 is wrong: %T %v", regexp1, regexp1)
	}

	if regexp2.string(ctx) != `(re) (?P<name>re) (?:re) (?im) (?sU:re) \x{10ffff} \x1f \123 \* \. \? \$` {
		ctx.err("regexp2 is wrong: %T %v", regexp2, regexp2)
	}

	if regexp3.string(ctx) != `[[:xdigit:]]*, [[:^alpha:]], [^xyz] [a-z] \A \B \b \Q**??^:[]{}\E \^ \z` {
		ctx.err("regexp3 is wrong: %T %v", regexp3, regexp3)
	}

	if regexp4.string(ctx) != `fo{2}\.c` {
		ctx.err("regexp4 is wrong: %T %v", regexp4, regexp4)
	}

	if regexp5.string(ctx) != `fo{2}/bar\.c` {
		ctx.err("regexp5 is wrong: %T %v", regexp5, regexp5)
	}

	if regexp6.string(ctx) != `fo{2}(/o{2}){3}/bar\.c` {
		ctx.err("regexp6 is wrong: %T %v", regexp6, regexp6)
	}

	if val := ctx.val("val1"); val == nil {
		ctx.err("val1")
	} else if val.string(ctx) != "foo.c" {
		ctx.err("%v", ust{val})
	} else if a, b, c := glob1.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob1, val, a, b, c)
	} else if a, b, c := glob2.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, val, a, b, c)
	} else if a, b, c := regexp4.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", regexp4, val, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regexp4, val, a, b, c)
	} else if s != val.string(ctx) {
		ctx.err("match(%v, %v): %v %v %v", regexp4, val, a, b, c)
	} else if len(c) != 0 {
		ctx.err("match(%v, %v): %v %v %v", regexp4, val, a, b, c)
	}

	if val := ctx.val("val2"); val == nil {
		ctx.err("val2")
	} else if val.string(ctx) != "foo/bar.c" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", ust{val})
	} else if len(p.elems) != 2 {
		ctx.err("%v: %v: %v", typeof(val), val, p.elems)
	} else if a, b, c := glob1.match(ctx, val); a == true {
		if false { ctx.err("match(%v, %v): %v %v %v", glob1, val, a, b, c) }
	} else if a, b, c := glob2.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, val, a, b, c)
	} else if a, b, c := regexp5.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", regexp5, val, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regexp5, val, a, b, c)
	} else if s != val.string(ctx) {
		ctx.err("match(%v, %v): %v %v %v", regexp5, val, a, b, c)
	} else if len(c) != 0 {
		ctx.err("match(%v, %v): %v %v %v", regexp5, val, a, b, c)
	}

	if val := ctx.val("val3"); val == nil {
		ctx.err("val3")
	} else if val.string(ctx) != "foo/oo/oo/oo/bar.c" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", ust{val})
	} else if len(p.elems) != 5 {
		ctx.err("%v: %v: %v", typeof(val), val, p.elems)
	} else if a, b, c := glob1.match(ctx, val); a == true {
		if false { ctx.err("match(%v, %v): %v %v %v", glob1, val, a, b, c) }
	} else if a, b, c := glob2.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", glob2, val, a, b, c)
	} else if a, b, c := regexp6.match(ctx, val); !a {
		ctx.err("match(%v, %v): %v %v %v", regexp6, val, a, b, c)
	} else if s, y := b.(string); !y {
		ctx.err("match(%v, %v): %v %v %v", regexp6, val, a, b, c)
	} else if s != val.string(ctx) {
		ctx.err("match(%v, %v): %v %v %v", regexp6, val, a, b, c)
	} else if len(c) != 1 {
		ctx.err("match(%v, %v): %v %v %v", regexp6, val, a, b, c)
	} else if c[0] != "/oo" {
		ctx.err("match(%v, %v): %v %v %v", regexp6, val, a, b, c)
	}

	// TODO: test glob.stencil(...)

	if val := ctx.val("val4"); val == nil {
		ctx.err("val4")
	} else if val.String() != "a\\,b\\,c,x\\,y\\,z" {
		ctx.err("%v", ust{val})
	} else if val.string(ctx) != "a,b,c,x,y,z" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*barecomp); !y {
		ctx.err("%v", ust{val})
	} else if len(p.elems) != 11 {
		ctx.err("%v: %v %v", typeof(val), val, p.elems)
	}

	if val := ctx.val("val5"); val == nil {
		ctx.err("val5")
	} else if val.String() != `'"a,b,c x,y,z"'` {
		ctx.err("%v", ust{val})
	} else if val.string(ctx) != `"a,b,c x,y,z"` {
		ctx.err("%v", ust{val})
	} else if _, y := val.(*strlit); !y {
		ctx.err("%v", ust{val})
	}

	// TODO: test regexp.stencil(...)
}

func testAutomatic(ctx *testcase) {
	if c := _universe(ctx); c == nil {
		ctx.err("Context.cast")
	}
	{
		ac := automatic{ Context:ctx, defs:make(autodefs) }
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
			ctx.err("%v → %s", ust{d.value}, s)
		} else if s := d.value.string(ctx); s != "a b c" {
			ctx.err("%v → %s", ust{d.value}, s)
		} else if d := ac.search(ctx, "1"); d == nil {
			ctx.err("%v", ac.defs)
		} else if d := autoDef(&ac, "1"); d == nil {
			ctx.err("%v", ac.defs)
		} else if v := autoVal(&ac, "1"); v == nil {
			ctx.err("%v", ac.defs)
		} else if s := v.string(ctx); s != "a b c" {
			ctx.err("%v → %s", ust{v}, s)
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
		ac := automatic{ Context:ctx, defs:make(autodefs) }
		ac.args(ctx, ease(ctx, []string{"a", "b", "c"}).(*list).elems)
		if len(ac.defs) != 3 { // maxDigitAutoNum
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "a" {
			ctx.err("%v", ust{d.value})
		} else if d, y := ac.defs["2"]; !y {
			ctx.err("2: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "b" {
			ctx.err("%v", ust{d.value})
		} else if d, y := ac.defs["3"]; !y {
			ctx.err("3: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "c" {
			ctx.err("%v", ust{d.value})
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
		ctx.err("%v", ust{d.value})
	} else if len(l.elems) != 10 {
		ctx.err("%v", l.elems)
	} else if s := "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9"; s != d.value.String() {
		ctx.err("%v != %s", ust{d.value}, s)
	} else if s := d.value.string(ctx); s != "" {
		ctx.err("%v ; %s", ust{d.value}, s)
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s := "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9"; s != v.String() {
			ctx.err("%v != %s", ust{v}, s)
		} else if s := v.string(ctx); s != "" {
			ctx.err("%v → %s", ust{v}, s)
		}

		if v := ctx.val(d.name, "a"); v == nil {
			ctx.err("%v", d)
		} else if s := "$0 a $2 $3 $4 $5 $6 $7 $8 $9"; s != v.String() {
			ctx.err("%v != %s", ust{v}, s)
		} else if s := v.string(ctx); s != "a" {
			ctx.err("%v → %s", ust{v}, s)
		}

		if v := ctx.val(d.name, "1"); v == nil {
			ctx.err("%v", d)
		} else if s := "$0 1 $2 $3 $4 $5 $6 $7 $8 $9"; s != v.String() {
			ctx.err("%v != %s", ust{v}, s)
		} else if s := v.string(ctx); s != "1" {
			ctx.err("%v → %s", ust{v}, s)
		}

		if v := ctx.val(d.name, "1", "2", "3"); v == nil {
			ctx.err("%v", d)
		} else if s := "$0 1 2 3 $4 $5 $6 $7 $8 $9"; s != v.String() {
			ctx.err("%v != %s", ust{v}, s)
		} else if s := v.string(ctx); s != "1 2 3" {
			ctx.err("%v → %s", ust{v}, s)
		}
	}

	if d := ctx.def("val"); d == nil {
		ctx.err("val")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "&(foobar)", v.String(); s != t {
		ctx.err("%v", ust{d})
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s", ust{d}, s)
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s, t := "&(foobar)", v.String(); s != t {
			ctx.err("%v → %s != %s", ust{v}, t, s)
		} else if s, t := "", v.string(ctx); s != t {
			ctx.err("%v → %s != %s", ust{v}, t, s)
		}
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if a, y := t.x.(*auto); !y {
		ctx.err("%v %v", ust{t.x}, a)
	} else if s, t := "$(a)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if a := ctx.val(d.name, test_arg{"a", "x"}); a == nil {
		ctx.err("%v", ust{v})
	} else if s, y := a.(*bareword); !y || s.s != "x" {
		ctx.err("%v", ust{a})
	} else if s, t := "x", a.String(); s != t {
		ctx.err("%v → %s != %s", ust{a}, t, s)
	} else if s, t := "x", a.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{a}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 2 {
		ctx.err("%v", ust{v})
	} else if n, y := l.elems[0].(*decimal); !y {
		ctx.err("%v", ust{l.elems[0]})
	} else if n.int64 != 1 {
		ctx.err("%v", ust{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v", ust{l.elems[1]})
	} else if n.int64 != 1 {
		ctx.err("%v", ust{l.elems[1]})
	} else if s, t := "1 1", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "1 1", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 2 {
		ctx.err("%v", ust{v})
	} else if n, y := l.elems[0].(*decimal); !y {
		ctx.err("%v", ust{l.elems[0]})
	} else if n.int64 != 1 {
		ctx.err("%v", ust{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v", ust{l.elems[1]})
	} else if n.int64 != 1 {
		ctx.err("%v", ust{l.elems[1]})
	} else if s, t := "1 1", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "1 1", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 2 {
		ctx.err("%v", ust{v})
	} else if n, y := l.elems[0].(*decimal); !y {
		ctx.err("%v", ust{l.elems[0]})
	} else if n.int64 != 1 {
		ctx.err("%v", ust{l.elems[0]})
	} else if n, y := l.elems[1].(*decimal); !y {
		ctx.err("%v", ust{l.elems[1]})
	} else if n.int64 != 1 {
		ctx.err("%v", ust{l.elems[1]})
	} else if s, t := "1 1", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "1 1", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues1(ctx *testcase) {
	if v := ctx.val(".test.foo"); v == nil {
		ctx.err(".test.foo")
	} else if s := v.string(ctx); s != "-foo" {
		ctx.err("%v → %s", ust{v}, s)
	} else if s = v.String(); s != "-foo" {
		ctx.err("%v → %s", ust{v}, s)
	}
}

func testValues2(ctx *testcase) {
	if d := ctx.def(".test.ab"); d == nil {
		ctx.err(".test.ab")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "foobar $1-$2", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "foobar -", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "foobar a-b", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.ba"); d == nil {
		ctx.err(".test.ba")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "foobaz $2-$1", v.String(); s != t {
		ctx.err("%v", ust{v})
	} else if s, t := "foobaz -", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "foobaz b-a", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) z $(call(-closure) &(.test.x),$1$2,$2$1) Z $3", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar - z foobar - Z", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) z $(call(-closure) &(.test.x),ab,ba) Z c", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z c", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c", test_def_1{}); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) z $(call(-closure) &(.test.x),ab,ba) Z c", v.String(); s != t {
		ctx.err("%v != %s", ust{v}, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z c", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c", test_def_2{}); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x foobar $1-$2 y foobar aa-bb z foobar ab-ba Z c", v.String(); s != t {
		ctx.err("%v != %s", ust{v}, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z c", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.0"); d == nil {
		ctx.err(".test.0")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v != %s", ust{v}, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v != %s", ust{v}, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.s1"); d == nil {
		ctx.err(".test.s1")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if t, y := v.(*strval); !y {
		ctx.err("%v", ust{v})
	} else if len(t.v) != 5 {
		ctx.err("%v %v", ust{v}, ust{t.v})
	} else if s, t := "$(string x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(string x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.s2"); d == nil {
		ctx.err(".test.s2")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 5 {
		ctx.err("%v", ust{l})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b", "c"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.s3"); d == nil {
		ctx.err(".test.s3")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*strlit); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "a b c", l.s; s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := `'a b c'`, v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := `a b c`, v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.1"); d == nil {
		ctx.err(".test.1")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.value, "a", "b", "c"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.2"); d == nil || d.value == nil {
		ctx.err(".test.2")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x foobaz $2-$1 y $(.test.ba $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v", ust{d})
	} else if v := ctx.val(d.value, "a", "b", "cc"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x foobaz b-a y foobaz bb-aa cc", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobaz b-a y foobaz bb-aa cc", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.3"); d == nil {
		ctx.err(".test.3")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) $1$1,$2$2) $3", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar -", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.value, "a", "b", "c"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar aa-bb c", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.4"); d == nil {
		ctx.err(".test.4")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if v := ctx.val(d.value, "a", "b", "x"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x foobar a-b y foobar aa-bb z foobar ab-ba Z x . x", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.5"); d == nil {
		ctx.err(".test.5")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if v := ctx.val(d.value, "a", "b", "x"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x $(closure &(.test.x)) y $(&(.test.x) aa,bb) z $(call(-closure) &(.test.x),ab,ba) Z x . x", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x foobar - y foobar aa-bb z foobar ab-ba Z x . x", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues3(ctx *testcase) {
	if d := ctx.def(".test.x"); d == nil {
		ctx.err(".test.x")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(a1)-$(a2)-3", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", ust{d.value})
	} else if s, t := "x-y-3", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x-y-3", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.y"); d == nil {
		ctx.err(".test.y")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x-y-3-xyz", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.z"); d == nil {
		ctx.err(".test.z")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "$(a1)-$(a2)-3-$(a1)$(a2)$(a3)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d.value})
	} else if s, t := "$(a1)-$(a2)-3-$(a1)$(a2)$(a3)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "--3-", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.1"); d == nil {
		ctx.err(".test.1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x-y-3-xyz", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.2"); d == nil {
		ctx.err(".test.2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x-y-3-xyz", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.3"); d == nil {
		ctx.err(".test.3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "x-y-3-xyz", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues4(ctx *testcase) {
	if d := ctx.def(".test.*"); d == nil {
		ctx.err(".test.*")
	} else if d.origin != defExpand1 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 4 {
		ctx.err("%v ; (%d)", ust{l.elems}, l.len())
	} else if _, y := l.elems[0].(*argumented); !y || l.elems[0].String() != "D.c(-unique)" {
		ctx.err("%v", ust{l.elems[0]})
	} else if _, y := l.elems[1].(*argumented); !y || l.elems[1].String() != "D.c++(-unique)" {
		ctx.err("%v", ust{l.elems[1]})
	} else if _, y := l.elems[2].(*argumented); !y || l.elems[2].String() != "I.c(-unique)" {
		ctx.err("%v", ust{l.elems[2]})
	} else if _, y := l.elems[3].(*argumented); !y || l.elems[3].String() != "I.c++(-unique)" {
		ctx.err("%v", ust{l.elems[3]})
	} else if s, t := "D.c(-unique) D.c++(-unique) I.c(-unique) I.c++(-unique)", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.D.c"); d == nil {
		ctx.err(".test.D.c")
	} else if d.origin != defExpand1 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 9 {
		ctx.err("%v ; (%d)", ust{v}, l.len())
	} else if _, y := l.elems[0].(*bareword); !y || l.elems[0].String() != "D" {
		ctx.err("%v", ust{l.elems[0]})
	} else if _, y := l.elems[1].(*bareword); !y || l.elems[1].String() != "c" {
		ctx.err("%v", ust{l.elems[1]})
	} else if _, y := l.elems[2].(*delegate); !y || l.elems[2].String() != "$(value &(.test.x))" {
		ctx.err("%v", ust{l.elems[2]})
	} else if t, y := l.elems[3].(*delegate); !y || l.elems[3].String() != "$(.test.v)" {
		ctx.err("%v", ust{l.elems[3]})
	} else if _, y := t.x.(*barecomp); !y || t.x.String() != ".test.v" {
		ctx.err("%v", ust{t.x})
	} else if t.string(ctx) != "xx" {
		ctx.err("%v", ust{t})
	} else if _, y := l.elems[4].(*closure);  !y || l.elems[4].String() != "&(value .test.v)" {
		ctx.err("%v", ust{l.elems[4]})
	} else if _, y := l.elems[5].(*delegate); !y || l.elems[5].String() != "$(.test.foreach $1,&(.test.none))" {
		ctx.err("%v", ust{l.elems[5]})
	} else if _, y := l.elems[6].(*group);    !y || l.elems[6].String() != "($1)" {
		ctx.err("%v", ust{l.elems[6]})
	} else if _, y := l.elems[7].(*delegate); !y || l.elems[7].String() != "$(foreach $1,&(.test.x.$_))" {
		ctx.err("%v", ust{l.elems[7]})
	} else if _, y := l.elems[8].(*group);    !y || l.elems[8].String() != "($1)" {
		ctx.err("%v", ust{l.elems[8]})
	} else if s, t := "D c $(value &(.test.x)) $(.test.v) &(value .test.v) $(.test.foreach $1,&(.test.none)) ($1) $(foreach $1,&(.test.x.$_)) ($1)", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "D c xx xx xx () () ()", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a"); v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 9 {
		ctx.err("%v ; (%d)", ust{v}, l.len())
	} else if s, t := "D c $(value &(.test.x)) xx &(value .test.v) (a) (a) &(.test.x.a)? (a)", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "D c xx xx xx (a) (a) x (a)", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", test_def_2{}); v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 9 {
		ctx.err("%v ; (%d)", ust{v}, l.len())
	} else if s, t := "D c xx xx xx $(.test.foreach a,&(.test.none)) (a) x (a)", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "D c xx xx xx (a) (a) x (a)", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.D.c++"); d == nil {
		ctx.err(".test.D.c++")
	} else if d.origin != defExpand1 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "D c++", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "D c++", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.I.c"); d == nil {
		ctx.err(".test.I.c")
	} else if d.origin != defExpand1 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if l.len() != 6 {
		ctx.err("%v ; (%d)", ust{v}, l.len())
	} else if _, y := l.elems[0].(*bareword); !y || l.elems[0].String() != "I" {
		ctx.err("%v", ust{l.elems[0]})
	} else if _, y := l.elems[1].(*bareword); !y || l.elems[1].String() != "c" {
		ctx.err("%v", ust{l.elems[1]})
	} else if _, y := l.elems[2].(*closure);  !y || l.elems[2].String() != "&(value &(.test.x))" {
		ctx.err("%v", ust{l.elems[2]})
	} else if _, y := l.elems[3].(*closure);  !y || l.elems[3].String() != "&(value .test.v)" {
		ctx.err("%v", ust{l.elems[3]})
	} else if _, y := l.elems[4].(*delegate); !y || l.elems[4].String() != "$(value &(.test.x))" {
		ctx.err("%v", ust{l.elems[4]})
	} else if _, y := l.elems[5].(*bareword); !y || l.elems[5].String() != "xx" {
		ctx.err("%v", ust{l.elems[5]})
	} else if s, t := "I c &(value &(.test.x)) &(value .test.v) $(value &(.test.x)) xx", v.String(); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "I c xx xx xx xx", v.string(ctx); s != t {
		if true { for i, v := range l.elems { note(ctx, "%d. %v", i, ust{v}) } }
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.I.c++"); d == nil {
		ctx.err(".test.I.c++")
	} else if d.origin != defExpand1 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "I c++", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "I c++", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.and.x.1"); d == nil {
		ctx.err(".test.and.x.1")
	} else if d.origin != defExpand0 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "x1", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.and.x.2"); d == nil {
		ctx.err(".test.and.x.2")
	} else if d.origin != defExpand0 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "x2", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.and.y.1"); d == nil {
		ctx.err(".test.and.y.1")
	} else if d.origin != defExpand0 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "y1", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def(".test.and.y.2"); d == nil {
		ctx.err(".test.and.y.2")
	} else if d.origin != defExpand0 {
		ctx.err("%v", ust{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "y2", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues5(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", ust{d})
	} else if d.value.String() != "z-$1" {
		ctx.err("%v", ust{d})
	} else if v := ctx.val(d.name, "a"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "z-a", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "z-a", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues6(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", ust{d})
	} else if d.value.String() != "z-y-x-a" {
		ctx.err("%v", ust{d})
	} else if v := ctx.val(d.name, "b"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "z-y-x-a", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "z-y-x-a", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues7(ctx *testcase) {
	if d := ctx.def(".test.z"); d == nil {
		ctx.err(".test.z")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "z-$1-$2", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
	if d := ctx.def(".test.y"); d == nil {
		ctx.err(".test.y")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(.test.z y$1,y$2)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
	if d := ctx.def(".test.x"); d == nil {
		ctx.err(".test.x")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(.test.y x$1,x$2)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(.test.x $1,$2)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "a", "b"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "z-yxa-yxb", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "z-yxa-yxb", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues8(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", ust{d})
	} else if d.value.String() != "$1" {
		ctx.err("%v", ust{d})
	} else if v := ctx.val(d.name, "a"); v == nil {
		ctx.err(".test")
	} else if s, t := "a", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "a", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
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
		ctx.err("%v", ust{v})
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%v → %s", ust{v}, s)
	}
}

func testValues10(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(.test.y .$1)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "w"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "foobar", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "foobar", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
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
		ctx.err("%v", ust{v})
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%v → %s", ust{v}, s)
	}
}

func testValues12(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(.test.y .$1)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "w2"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "&(.test.w2)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "foobaz", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testValues13(ctx *testcase) {
	if d := ctx.def("foo"); d == nil {
		ctx.err("foo")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "&(-g!foobar)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "&(-g!foobar)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "not-foobar", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testPlaceholders(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if len(l.elems) != 10 {
		ctx.err("%v", l)
	} else if s, t := "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "_", "x", "y", "z"); v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if d, y := l.elems[0].(*delegate); !y {
		ctx.err("%v", ust{l.elems[0]})
	} else if a, y := d.x.(*auto); !y {
		ctx.err("%v", ust{d.x})
	} else if a.ident(ctx) != "0" {
		ctx.err("%v", a)
	} else if s, t := "$0 1 2 3 4 5 6 7 8 9", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "1 2 3 4 5 6 7 8 9", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "$(foreach a b c d e f,$_)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "a b c d e f", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "a b c d e f", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "1 2 3 4 5 6 7 8 9", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", ust{v})
	} else if len(l.elems) != 6 {
		ctx.err("%v", l)
	} else if s, t := "a b c d e f", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "1 2 3 4 5 6 7 8 9", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}
}

func testOptional(ctx *testcase) {
	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if o, y := v.(project_box); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "foo", o.name; s != t {
		ctx.err("%v", ust{v})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.ident(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if o, y := t.x.(condval); !y {
		ctx.err("%v", ust{t.x})
	} else if _, y := o.Value.(*bareword); !y {
		ctx.err("%v", ust{o.Value})
	} else if s, t := "$(name?)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(name?)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if o, y := v.(project_box); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "foo", o.name; s != t {
		ctx.err("%v", ust{v})
	} else if o := v.expand(ctx); o == nil {
		ctx.err("%v → nil", ust{t})
	} else if t := o.String(); s != t {
		ctx.err("%v → %s != %s", ust{o}, t, s)
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.ident(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.ident(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if _, y := t.x.(*selection); !y {
		ctx.err("%v", ust{t.x})
	} else if s, t := "$(foo→baz?)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(foo→baz?)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if _, y := t.x.(*selection); !y {
		ctx.err("%v", ust{t.x})
	} else if s, t := "$(fo?→bar)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(fo?→bar)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if t, y := v.(*delegate); !y {
		ctx.err("%v", ust{v})
	} else if _, y := t.x.(*selection); !y {
		ctx.err("%v", ust{t.x})
	} else if s, t := "$(fo?→bar?)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(fo?→bar?)", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if o, y := v.(project_box); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "foo", o.name; s != t {
		ctx.err("%v", ust{v})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if o, y := v.(project_box); !y {
		ctx.err("%v", ust{v})
	} else if s, t := "foo", o.name; s != t {
		ctx.err("%v", ust{v})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if t := v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if v := d.value; v == nil {
		ctx.err("%v", ust{d})
	} else if s, t := "$(foo→bar?)?", v.String(); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
	} else if s, t := "", v.string(ctx); s != t {
		ctx.err("%v → %s != %s", ust{v}, t, s)
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
		ctx.err("pat1.0: %v", ctx.project())
	} else if p, y := val.(*path); !y {
		ctx.err("%v", ust{val})
	} else if p.len() != 2 {
		ctx.err("%v", p)
	} else if val.String() != ".test/x**y" {
		ctx.err("%v", ust{val})
	} else if v := ctx.val("pat1.1"); v == nil {
		ctx.err("pat1.1: %v", ctx.project())
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
		ctx.err("pat1.2: %v", ctx.project())
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
		ctx.err("pat1.3: %v", ctx.project())
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
		ctx.err("pat1.3: %v", ctx.project())
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
		ctx.err("pat1.5: %v", ctx.project())
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
		ctx.err("pat1.6: %v", ctx.project())
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
		ctx.err("pat2.0: %v", ctx.project())
	} else if val.String() != ".test/x**y/z" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat2.1"); v == nil {
		ctx.err("pat2.1: %v", ctx.project())
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 3 {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b[0] != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[1] != "xxx-yyy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if b[2] != "z" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx-yy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat2.2"); v == nil {
		ctx.err("pat2.1: %v", ctx.project())
	} else if a, ib, c := p.match(ctx, v); !a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.([]string); !y || len(b) != 7 {
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
	} else if b[6] != "z" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 1 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if c[0] != "xx/a/b/c/yy" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	if val := ctx.val("pat3.0"); val == nil {
		ctx.err("pat3.0: %v", ctx.project())
	} else if val.String() != ".test/x**y/x**" {
		ctx.err("%v", ust{val})
	} else if val.string(ctx) != ".test/x**y/x**" {
		ctx.err("%v: %s", ust{val}, val.string(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat3.1"); v == nil {
		ctx.err("pat3.1: %v", ctx.project())
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
		ctx.err("pat3.1: %v", ctx.project())
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
		ctx.err("pat4.0: %v", ctx.project())
	} else if val.String() != ".test/x**y/x**y" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat4.1"); v == nil {
		ctx.err("pat4.1: %v", ctx.project())
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
		ctx.err("pat4.1: %v", ctx.project())
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
		ctx.err("pat5.0: %v", ctx.project())
	} else if val.String() != ".test/x**/**y" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat5.1"); v == nil {
		ctx.err("pat5.1: %v", ctx.project())
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
		ctx.err("pat5.2: %v", ctx.project())
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
		ctx.err("pat6.0: %v", ctx.project())
	} else if val.String() != ".test/**y/**y" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat6.1"); v == nil {
		ctx.err("pat6.1: %v", ctx.project())
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
		ctx.err("pat7.0: %v", ctx.project())
	} else if val.String() != ".test/**y/**y/z" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat7.1"); v == nil {
		ctx.err("pat7.1: %v", ctx.project())
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
		ctx.err("pat8.0: %v", ctx.project())
	} else if val.String() != ".test/**/**z" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat8.1"); v == nil {
		ctx.err("pat8.1: %v", ctx.project())
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
		ctx.err("pat10.0: %v", ctx.project())
	} else if val.String() != ".test/*.h" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat10.1"); v == nil {
		ctx.err("pat10.1: %v", ctx.project())
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
		ctx.err("pat10.2: %v", ctx.project())
	} else if a, ib, c := p.match(ctx, v); a {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b, y := ib.(string); !y {
		ctx.err("%v %v: %v %v %v", p, v, a, ib, c)
	} else if b != ".test" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if len(c) != 0 {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	} else if v := ctx.val("pat10.3"); v == nil {
		ctx.err("pat10.1: %v", ctx.project())
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
		ctx.err("pat11.0: %v", ctx.project())
	} else if val.String() != ".test/*/*.h" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat11.1"); v == nil {
		ctx.err("pat11.1: %v", ctx.project())
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
		ctx.err("pat11.2: %v", ctx.project())
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
		ctx.err("pat11.3: %v", ctx.project())
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
		ctx.err("pat12.0: %v", ctx.project())
	} else if val.String() != ".test/*/*/*.h" {
		ctx.err("%v", ust{val})
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if v := ctx.val("pat12.1"); v == nil {
		ctx.err("pat12.1: %v", ctx.project())
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
		ctx.err("pat12.2: %v", ctx.project())
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
		ctx.err("pat12.3: %v", ctx.project())
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
		ctx.err("pat12.4: %v", ctx.project())
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
		ctx.err("pat12.5: %v", ctx.project())
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

	if val := ctx.val("pat13.0"); val == nil {
		ctx.err("pat13.0: %v", ctx.project())
	} else if val.string(ctx) != "**.auto" {
		ctx.err("%v", ust{val})
	} else if v := ctx.val("pat13.1"); v == nil {
		ctx.err("pat13.1: %v", ctx.project())
	} else if a, b, c := val.match(ctx, v); !a {
		ctx.err("%T %v ; %v %v %v", val, val, a, b, c)
	} else if v := ctx.val("pat13.2"); v == nil {
		ctx.err("pat13.2: %v", ctx.project())
	} else if a, b, c := val.match(ctx, v); a {
		ctx.err("%v ; %v %v %v", ust{val}, a, b, c)
	}
}
