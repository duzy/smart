//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"reflect"
	"testing"
	"fmt"
)

func testAutoContext(ctx *testcase) {
	if c := ctx.cast(reflect.TypeOf((*universe)(nil))); c == nil {
		ctx.err("Context.cast")
	} else if c := cast[*universe](ctx); c == nil {
		ctx.err("Context.cast")
	}

	{
		ac := autoContext{ Context:ctx, defs:make(autoDefMap) }
		ac.args(ctx, nil, []Value{ease(ctx, []string{"a", "b", "c"})})
		if Context(&ac).ac() != &ac {
			ctx.err("%v", Context(&ac))
		} else if len(ac.defs) != 1 {
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if s := d.value.String(); s != "'a' 'b' 'c'" {
			ctx.err("%T %v -> %s", d.value, d.value, s)
		} else if s := d.value.string(ctx); s != "a b c" {
			ctx.err("%T %v -> %s", d.value, d.value, s)
		} else if d := ac.get(ctx, "1"); d == nil {
			ctx.err("%v", ac.defs)
		} else if d := autoDef(&ac, "1"); d == nil {
			ctx.err("%v", ac.defs)
		} else if v := autoVal(&ac, "1"); v == nil {
			ctx.err("%v", ac.defs)
		} else if s := v.string(ctx); s != "a b c" {
			ctx.err("%T %v -> %s", v, v, s)
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
		}
	}
	{
		ac := autoContext{ Context:ctx, defs:make(autoDefMap) }
		ac.args(ctx, nil, ease(ctx, []string{"a", "b", "c"}).(*List).Elems)
		if len(ac.defs) != 3 {
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "a" {
			ctx.err("%T %v", d.value, d.value)
		}
	}

	if foo := ctx.def("foo"); foo == nil {
		ctx.err("foo")
	} else if foo.value == nil {
		ctx.err("%v", foo)
	} else if s := foo.value.String(); s != "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v ; %s", foo.value, foo.value, s)
	} else if s := foo.value.string(ctx); s != "" {
		ctx.err("%T %v ; %s", foo.value, foo.value, s)
	} else if l, y := foo.value.(*List); !y {
		ctx.err("%T %v", foo.value, foo.value)
	} else if len(l.Elems) != 10 {
		ctx.err("%v", l.Elems)
	} else if !l.Elems[0].expandable(ctx, expandAuto|expandDigits) {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if !l.Elems[0].expandable(ctx, expandAuto|expandDigits|expandDelegate) {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if l.Elems[0].expandable(ctx, expandAuto) {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if l.Elems[0].expandable(ctx, expandDigits) {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if l.Elems[0].expandable(ctx, expandDelegate) {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if l.Elems[0].expandable(ctx, expandPlaceholder) {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	}

	if foo := ctx.get("foo"); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if foo := ctx.get("foo", "1"); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "1" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if foo := ctx.get("foo", "1", "2", "3"); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 1 2 3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "1 2 3" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if foo := ctx.get("foo", expandDelegate, "1"); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "1" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if foo := ctx.get("foo", expandAutoKept|expandDelegate, "1"); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if foo := ctx.get("foo", "1", "2", "3", expandAutoKept|expandDelegate); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if foo := ctx.get("foo", expandDelegate, "1"); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "1" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if foo := ctx.get("foo", expandDelegate, "1", "2", "3"); foo == nil {
		ctx.err("foo")
	} else if s := foo.String(); s != "$0 1 2 3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", foo, foo, s)
	} else if s := foo.string(ctx); s != "1 2 3" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	if d := ctx.def("val"); d == nil {
		ctx.err("val")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "&(foobar)" {
		ctx.err("%T %v", d.value, d.value)
	} else if s := d.value.string(ctx); s != "" {
		ctx.err("%T %v -> %s", d.value, d.value, s)
	}
	if v := ctx.get("val"); v == nil {
		ctx.err("val")
	} else if v.String() != "&(foobar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v1, v2 := ctx.get("val1"), ctx.get("val2"); v1 == nil || v2 == nil {
		ctx.err("val1")
		ctx.err("val2")
	} else if v1.String() != "$(a)" {
		ctx.err("%T %v", v1, v1)
	} else if v2.String() != "$(a)" {
		ctx.err("%T %v", v2, v2)
	} else if u1, y := v1.(unexpanded); !y {
		ctx.err("%T %v", v1, v1)
	} else if u2, y := v2.(unexpanded); !y {
		ctx.err("%T %v", v2, v2)
	} else if l1, y := u1.Value.(*List); !y || l1.Len() != 1 {
		ctx.err("%T %v , %v", u1.Value, u1.Value, l1)
	} else if l2, y := u2.Value.(*List); !y || l2.Len() != 1 {
		ctx.err("%T %v , %v", u2.Value, u2.Value, l2)
	} else if u1, y := l1.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l1.Elems[0], l1.Elems[0])
	} else if u2, y := l2.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l2.Elems[0], l2.Elems[0])
	} else if d1, y := u1.Value.(*delegate); !y {
		ctx.err("%T %v", u1.Value, u1.Value)
	} else if d2, y := u2.Value.(*delegate); !y {
		ctx.err("%T %v", u2.Value, u2.Value)
	} else if u1, y := d1.x.(unexpanded); !y {
		ctx.err("%T %v", d1.x, d1.x)
	} else if u2, y := d2.x.(unexpanded); !y {
		ctx.err("%T %v", d2.x, d2.x)
	} else if a1, y := u1.Value.(*auto); !y {
		ctx.err("%T %v", u1.Value, u1.Value)
	} else if a2, y := u2.Value.(*auto); !y {
		ctx.err("%T %v", u2.Value, u2.Value)
	} else if t := a1.cmp(ctx, a2); t != cmpEqual {
		ctx.err("%v, %v ; %v", a1, a2, t)
	} else if t := a2.cmp(ctx, a1); t != cmpEqual {
		ctx.err("%v, %v ; %v", a1, a2, t)
	} else if t := v1.cmp(ctx, v2); t != cmpEqual {
		ctx.err("%T %v, %T %v ; %v", v2, v2, v1, v1, t)
	} else if t := v2.cmp(ctx, v1); t != cmpEqual {
		ctx.err("%T %v, %T %v ; %v", v2, v2, v1, v1, t)
	}
}

func testValues1(ctx *testcase) {
	if v := ctx.get(".test.foo"); v == nil {
		ctx.err(".test.foo")
	} else if s := v.string(ctx); s != "-foo" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if s = v.String(); s != "-foo" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues2(ctx *testcase) {
	if v := ctx.get(".test.ab"); v == nil {
		ctx.err(".test.ab")
	} else if s := v.String(); s != "foobar $1-$2" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if s := v.string(ctx); s != "foobar -" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.ab", "a", "b"); v == nil {
		ctx.err(".test.ab")
	} else if v.String() != "foobar a-b" {
		ctx.err("%T %v", v, v) // "foobar $1-$2"
	} else if s := v.string(ctx); s != "foobar a-b" {
		ctx.err("%T %v -> %s", v, v, s) // "foobar -"
	}
	if v := ctx.get(".test.ab", "a", "b", expandDelegate); v == nil {
		ctx.err(".test.ab")
	} else if v.String() != "foobar a-b" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar a-b" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.ba"); v == nil {
		ctx.err(".test.ba")
	} else if v.String() != "foobaz $2-$1" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobaz -" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.ba", "a", "b"); v == nil {
		ctx.err(".test.ba")
	} else if v.String() != "foobaz b-a" {
		ctx.err("%T %v", v, v) // "foobaz $2-$1"
	} else if s := v.string(ctx); s != "foobaz b-a" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.ba", "a", "b", expandDelegate); v == nil {
		ctx.err(".test.ba")
	} else if s := v.String(); s != "foobaz b-a" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobaz b-a" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test", "a", "b", "c"); v == nil {
		ctx.err(".test")
	} else if s := v.String(); s != "$(value(-c) &(.test.x)) b $(&(.test.x) aa,bb) c $(call(-c) &(.test.x),aa,bb)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c foobar aa-bb" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test", "a", "b", "c", expandClosure); v == nil {
		ctx.err(".test")
	} else if s := v.String(); s != "foobar $1-$2 b foobar aa-bb c foobar aa-bb" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c foobar aa-bb" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if true {
		// TODO: test the rest part
	} else if t := xa(ctx, v, "a", "b", "c"); t == nil {
		ctx.err("%T %v", v, v)
	} else if t.String() != "foobar a-b b foobar aa-bb c foobar aa-bb" {
		ctx.err("%T %v -> %T %v", v, v, t, t)
	} else if s := t.string(ctx); s != "foobar a-b b foobar aa-bb c foobar aa-bb" {
		ctx.err("%T %v -> %s", t, t, s)
	}

	if v := ctx.get(".test.0", "a", "b", "c"); v == nil {
		ctx.err(".test.0")
	} else if v.String() != "foobar $1-$2 b foobar aa-bb c" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.s1", "a", "b", "c"); v == nil {
		ctx.err(".test.s1")
	} else if v.String() != "'foobar $1-$2 b foobar aa-bb c'" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.s1"))
	} else if s := v.string(ctx); s != "foobar $1-$2 b foobar aa-bb c" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.s2", "a", "b", "c"); v == nil {
		ctx.err(".test.s2")
	} else if v.String() != "'foobar a-b b foobar aa-bb c'" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.s2"))
	} else if s := v.string(ctx); s != "foobar a-b b foobar aa-bb c" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.1", "a", "b", "c"); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "$(value(-c) &(.test.x)) b $(&(.test.x) aa,bb) c" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.1", "a", "b", "c", expandClosure); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "foobar $1-$2 b foobar aa-bb c" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if d := ctx.def(".test.2"); d == nil || d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "foobaz $2-$1 b foobaz $2$2-$1$1 $3" {
		ctx.err("%v", d)
	}
	if v := ctx.get(".test.2", "a", "b", "cc"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != "foobaz b-a b foobaz bb-aa cc" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "foobaz b-a b foobaz bb-aa cc" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.2", "a", "b", "cc"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != "foobaz b-a b foobaz bb-aa cc" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobaz b-a b foobaz bb-aa cc" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if d := ctx.def(".test.3"); d == nil || d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "$(value(-c) &(.test.x)) b $(&(.test.x) $1$1,$2$2) $3" {
		ctx.err("%v", d) // "foobaz $2-$1 b foobaz $22-$11 $3"
	}
	if v := ctx.get(".test.3", "a", "b", "c"); v == nil {
		ctx.err(".test.3")
	} else if v.String() != "$(value(-c) &(.test.x)) b $(&(.test.x) aa,bb) c" {
		ctx.err("%T %v", v, v) // foobar $1-$2 $(.test.ab $1$1,$2$2) $3
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.3", "a", "b", "c", expandClosure); v == nil {
		ctx.err(".test.3")
	} else if v.String() != "foobar $1-$2 b foobar aa-bb c" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.4", "a", "b", "x"); v == nil {
		ctx.err(".test.4")
	} else if v.String() != "foobar a-b b foobar aa-bb c foobar aa-bb x" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar a-b b foobar aa-bb c foobar aa-bb x" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.5", "a", "b", "x"); v == nil {
		ctx.err(".test.5")
	} else if v.String() != "$(value(-c) &(.test.x)) b $(&(.test.x) aa,bb) c $(call(-c) &(.test.x),aa,bb) x" {
		ctx.err("%T %v", v, v) // foobar $1-$2 b foobar aa-bb c foobar aa-bb x
	} else if s := v.string(ctx); s != "foobar - b foobar aa-bb c foobar aa-bb x" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues3(ctx *testcase) {
	if d := ctx.def(".test.x"); d == nil {
		ctx.err(".test.x")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "$(a1)-$(a2)-3" {
		ctx.err("%T %v", d.value, d.value)
	}
	if v := ctx.get(".test.x"); v == nil {
		ctx.err(".test.x")
	} else if v.String() != "$(a1)-$(a2)-3" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.x"))
	} else if s := v.string(ctx); s != "--3" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if d := ctx.def(".test.y"); d == nil {
		ctx.err(".test.y")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "x-y-3-xyz" {
		ctx.err("%T %v", d.value, d.value)
	}
	if v := ctx.get(".test.y"); v == nil {
		ctx.err(".test.y")
	} else if v.String() != "x-y-3-xyz" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.y"))
	} else if s := v.string(ctx); s != "x-y-3-xyz" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if d := ctx.def(".test.z"); d == nil {
		ctx.err(".test.z")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "$(a1)-$(a2)-3-$(a1)$(a2)$(a3)" {
		ctx.err("%T %v", d.value, d.value)
	}
	if v := ctx.get(".test.z"); v == nil {
		ctx.err(".test.z")
	} else if v.String() != "$(a1)-$(a2)-3-$(a1)$(a2)$(a3)" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.y"))
	} else if s := v.string(ctx); s != "--3-" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "x-y-3-xyz" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.1"))
	} else if s := v.string(ctx); s != "x-y-3-xyz" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != "x-y-3-xyz" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.2"))
	} else if s := v.string(ctx); s != "x-y-3-xyz" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if v.String() != "x-y-3-xyz" {
		ctx.err("%T %v ; %v", v, v, ctx.def(".test.3"))
	} else if s := v.string(ctx); s != "x-y-3-xyz" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues4(ctx *testcase) {
	if d := ctx.def(".test.*"); d == nil {
		ctx.err(".test.*")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "D.c(-unique) D.c++(-unique) I.c(-unique) I.c++(-unique)" {
		ctx.err("%v", d)
	} else if s := d.value.string(ctx); s != "D.c(-unique) D.c++(-unique) I.c(-unique) I.c++(-unique)" {
		ctx.err("%v -> %v", d, s)
	}

	if v := ctx.get(".test.D.c"); v == nil {
		ctx.err(".test.D.c")
	} else if v.String() != "D c $(value &(.test.x)) &(value .test.v) ($1) ($1) $(foreach $1,&(.test.x.$_)) ($1)" {
		ctx.err("%T %v", v, v) // "D c $(value &(.test.x)) xx $(.test.foreach $1,&(.test.none)) ($1) $(foreach $1,&(.test.x.$_)) ($1)"
	} else if s := v.string(ctx); s != "D c xx xx () () ()" {
		ctx.err("%T %v -> %s", v, v, s)
	}
	if v := ctx.get(".test.D.c", "a"); v == nil {
		ctx.err(".test.D.c")
	} else if v.String() != "D c $(value &(.test.x)) &(value .test.v) (a) (a) &(.test.x.a) (a)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "D c xx xx (a) (a) x (a)" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.D.c++"); v == nil {
		ctx.err(".test.D.c++")
	} else if s := v.String(); s != "D c++" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if s := v.string(ctx); s != "D c++" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.I.c"); v == nil {
		ctx.err(".test.I.c")
	} else if v.String() != "I c &(value &(.test.x)) &(value .test.v) $(value &(.test.x)) xx" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "I c xx xx xx xx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.I.c++"); v == nil {
		ctx.err(".test.I.c++")
	} else if s := v.String(); s != "I c++" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if s := v.string(ctx); s != "I c++" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues5(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "z-$1" {
		ctx.err("%v", d)
	}

	if v := ctx.get(".test", "a"); v == nil {
		ctx.err(".test")
	} else if v.String() != "z-a" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "z-a" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues6(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "z-y-x-a" {
		ctx.err("%v", d)
	}

	if v := ctx.get(".test", "b"); v == nil {
		ctx.err(".test")
	} else if v.String() != "z-y-x-a" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "z-y-x-a" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues7(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "z-yx$1-yx$2" {
		ctx.err("%v", d)
	}

	if v := ctx.get(".test", "a", "b"); v == nil {
		ctx.err(".test")
	} else if v.String() != "z-yxa-yxb" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "z-yxa-yxb" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues8(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "$1" {
		ctx.err("%v", d)
	}

	if v := ctx.get(".test", "a"); v == nil {
		ctx.err(".test")
	} else if v.String() != "a" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a" {
		ctx.err("%T %v -> %s", v, v, s)
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

	if v := ctx.get(".test", ".u"); v == nil {
		ctx.err(".test")
	} else if v.String() != "foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues10(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "$(.test.$1)" {
		ctx.err("%v", d)
	}

	if v := ctx.get(".test", "w"); v == nil {
		ctx.err(".test")
	} else if v.String() != "foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v -> %s", v, v, s)
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

	if v := ctx.get(".test", ".v2"); v == nil {
		ctx.err(".test")
	} else if v.String() != "&(.test.v2)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues12(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "&(.test.$1)" {
		ctx.err("%v", d)
	}

	if v := ctx.get(".test", "w2"); v == nil {
		ctx.err(".test")
	} else if v.String() != "&(.test.w2)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "foobaz" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testValues13(ctx *testcase) {
	if d := ctx.def("foo"); d == nil {
		ctx.err("foo")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "&(-g!foobar)" {
		ctx.err("%v", d)
	}

	if v := ctx.get("foo"); v == nil {
		ctx.err(".test")
	} else if v.String() != "&(-g!foobar)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "not-foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	}
}

func testPlaceholders(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1 is nil")
	} else if s := d.value.String(); s != "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v -> %s", d.value, d.value, s)
	} else if l, y := d.value.(*List); !y {
		ctx.err("%T %v", d.value, d.value)
	} else if len(l.Elems) != 10 {
		ctx.err("%v", l)
	} else { for i, v := range l.Elems { if d, y := v.(digital); !y {
		ctx.err("%d - %T %v", i, v, v)
	} else if d, y := d.Value.(*delegate); !y {
		ctx.err("%d - %T %v", i, v, v)
	} else if d.l != Token(int(DELEGATE_0)+i) {
		ctx.err("%d - %T %v - %v", i, v, v, d.l)
	}}}

	if v := ctx.get("val1", "1", "2", "3", "4", "5", "6", "7", "8", "9", "_"); v == nil {
		ctx.err("val1")
	} else if v.String() != "$0 1 2 3 4 5 6 7 8 9" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "1 2 3 4 5 6 7 8 9" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if l, y := u.Value.(*List); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if u, y := l.Elems[0].(unexpanded); !y {
		ctx.err("%T %v", l.Elems[0], l.Elems[0])
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if u, y := d.x.(unexpanded); !y {
		ctx.err("%T %v", d.x, d.x)
	} else if a, y := u.Value.(*auto); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if a.name(ctx) != "0" {
		ctx.err("%v", a)
	} else { for i, v := range l.Elems[1:] { if w, y := v.(*bareword); !y {
		ctx.err("%T %v", v, v)
	} else if w.s != fmt.Sprintf("%d", i+1) {
		ctx.err("%T %v , %s", v, v, w.s)
	}}}

	if v := ctx.get("val2", "1", "2", "3", "4", "5", "6", "7", "8", "9"); v == nil {
		ctx.err("val2")
	} else if v.String() != "$(foreach a b c d e f,$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "a b c d e f" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if b, y := d.x.(*builtin); !y { // optional
		ctx.err("%T %v ; %T %v", v, v, d.x, d.x)
	} else if b.name_ != "foreach" {
		ctx.err("%T %v ; %v", v, v, b)
	} else if false {
		info(of(ctx,v), "%T %v, %T", v, v, u.Value).debug(1)
	}

	if v := ctx.get("val3", "1", "2", "3", "4", "5", "6", "7", "8", "9"); v == nil {
		ctx.err("val3")
	} else if v.String() != "$(foreach 1 2 3 4 5 6 7 8 9,$_)" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "1 2 3 4 5 6 7 8 9" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if u, y := v.(unexpanded); !y {
		ctx.err("%T %v", v, v)
	} else if d, y := u.Value.(*delegate); !y {
		ctx.err("%T %v", u.Value, u.Value)
	} else if b, y := d.x.(*builtin); !y { // optional
		ctx.err("%T %v ; %T %v", v, v, d.x, d.x)
	} else if b.name_ != "foreach" {
		ctx.err("%T %v ; %v", v, v, b)
	}

	if v := ctx.get("val4", "1", "2", "3", "4", "5", "6", "7", "8", "9"); v == nil {
		ctx.err("val4")
	} else if s := v.String(); s != "a b c d e f" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if s := v.string(ctx); s != "a b c d e f" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if _, y := v.(*List); !y {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get("val5", "1", "2", "3", "4", "5", "6", "7", "8", "9"); v == nil {
		ctx.err("val5")
	} else if s := v.String(); s != "1 2 3 4 5 6 7 8 9" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if s := v.string(ctx); s != "1 2 3 4 5 6 7 8 9" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if _, y := v.(*List); !y {
		ctx.err("%T %v", v, v)
	}
}

func testOptional(t *testing.T) {
	cl := init_commandline()
	cl.silentOptionalSelection = true
	runcase(t, "testdata/value/optional", "testoptional", func (ctx *testcase) {
		if v := ctx.get("val1"); v == nil {
			ctx.err("val1")
		} else if v.String() != "$(name)" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if u, y := v.(unexpanded); !y {
			ctx.err("%T %v", v, v)
		} else if d, y := u.Value.(*delegate); !y {
			ctx.err("%T %v", u.Value, u.Value)
		} else if u, y := d.x.(unresolved); !y { // optional
			ctx.err("%T %v ; %T %v", v, v, d.x, d.x)
		} else if w, y := u.Value.(*bareword); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		} else if w.s != "name" {
			ctx.err("%T %v , %v", v, v, w)
		} else if false {
			info(of(ctx,v), "%T %v, %T", v, v, u.Value).debug(1)
		}

		if v := ctx.get("val2"); v == nil {
			ctx.err("val2")
		} else if v.String() != ".self" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "foo" {
			ctx.err("%T %v -> %s", v, v, s)
			// } else if x, y := v.(expanded); !y {
			// 	ctx.err("%T %v", v, v)
			// } else if p, y := x.Value.(*self); !y {
			// 	ctx.err("%T %v", x.Value, x.Value)
		} else if p, y := v.(*self); !y {
			ctx.err("%T %v", v, v)
		} else if p.Project.name != "foo" {
			ctx.err("%T %v -> %v", v, v, p)
		} else if false {
			info(of(ctx,v), "%T %v", v, v).debug(1)
		}

		if v := ctx.get("val3"); v == nil {
			ctx.err("val3")
		} else if v.String() != "$(foo→baz?)" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if u, y := v.(unexpanded); !y {
			ctx.err("%T %v", v, v)
		} else if d, y := u.Value.(*delegate); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		} else if u, y := d.x.(unresolved); !y {
			ctx.err("%T %v ; %T %v", v, v, d.x, d.x)
		} else if _, y := u.Value.(*selection); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		}

		if v := ctx.get("val4"); v == nil {
			ctx.err("val4")
		} else if v.String() != "$(fo?→bar)" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if u, y := v.(unexpanded); !y {
			ctx.err("%T %v", v, v)
		} else if d, y := u.Value.(*delegate); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		} else if u, y := d.x.(unresolved); !y {
			ctx.err("%T %v ; %T %v", v, v, d.x, d.x)
		} else if _, y := u.Value.(*selection); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		}

		if v := ctx.get("val5"); v == nil {
			ctx.err("val5")
		} else if v.String() != "$(fo?→bar?)" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if u, y := v.(unexpanded); !y {
			ctx.err("%T %v", v, v)
		} else if d, y := u.Value.(*delegate); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		} else if u, y := d.x.(unresolved); !y {
			ctx.err("%T %v ; %T %v", v, v, d.x, d.x)
		} else if _, y := u.Value.(*selection); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		}

		if d := ctx.def("val6"); d == nil {
			ctx.err("val6")
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() != "foo" {
			ctx.err("%T %v", d.value, d.value)
		}
		if v := ctx.get("val6"); v == nil {
			ctx.err("val6")
		} else if v.String() != "foo" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "foo" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if p, y := v.(*projectname); !y {
			ctx.err("%T %v", v, v)
		} else if p.Project.name != "foo" {
			ctx.err("%T %v ; %v", v, v, p)
		}

		if v := ctx.get("val7"); v == nil {
			ctx.err("val7")
		} else if v.String() != "foo" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "foo" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if p, y := v.(*projectname); !y {
			ctx.err("%T %v", v, v)
		} else if p.Project.name != "foo" {
			ctx.err("%T %v ; %v", v, v, p)
		}

		if d := ctx.def("val8"); d == nil {
			ctx.err("val8")
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() != "$(foo→bar?)" {
			ctx.err("%T %v", d.value, d.value)
		}
		if v := ctx.get("val8"); v == nil {
			ctx.err("val8")
		} else if v.String() != "$(foo→bar?)" {
			ctx.err("%T %v", v, v)
		} else if s := v.string(ctx); s != "" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if u, y := v.(unexpanded); !y {
			ctx.err("%T %v", v, v)
		} else if d, y := u.Value.(*delegate); !y {
			ctx.err("%T %v", u.Value, u.Value)
		} else if u, y := d.x.(unresolved); !y { // optional
			ctx.err("%T %v ; %T %v", v, v, d.x, d.x)
		} else if _, y := u.Value.(*selection); !y {
			ctx.err("%T %v ; %T %v", v, v, u.Value, u.Value)
		}
	}, cl)
}

func testGlobMatch(t *testing.T) {
	var ctx Context = init_universe()//&uni

	if a, b, c := globMatch(ctx, "*.c", "foo.c"); !a || c != nil {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if b[0] != "foo" {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**.c", "foo.c"); !a || c != nil {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if b[0] != "foo" {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "*.c", "foo/bar.c"); a == true || c != nil {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if len(b) != 0 {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**.c", "foo/bar.c"); !a || c != nil {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "*", "foobar"); !a || c != nil {
		t.Errorf("glob(*, foobar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(*, foobar): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**", "foobar"); !a || c != nil {
		t.Errorf("glob(**, foobar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**, foobar): %v %v %v", a, b, c)
	} else if b[0] != "foobar" {
		t.Errorf("glob(**, foobar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "*", "foobar/"); a == true || c != nil {
		t.Errorf("glob(*, foobar/): %v %v %v", a, b, c)
	} else if len(b) != 0 {
		t.Errorf("glob(*, foobar/): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch(ctx, "**", "foobar/"); !a || c != nil {
		t.Errorf("glob(**, foobar/): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**, foobar/): %v %v %v", a, b, c)
	} else if b[0] != "foobar/" {
		t.Errorf("glob(**, foobar/): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**", "foo/bar/"); !a || c != nil {
		t.Errorf("glob(**, foo/bar/): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**, foo/bar/): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar/" {
		t.Errorf("glob(**, foo/bar/): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**xx**", "foo/bar/xx/foo/bar"); !a || c != nil {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 2 {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar/" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "/foo/bar" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**/xx/**", "foo/bar/xx/foo/bar"); !a || c != nil {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 2 {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "foo/bar" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**/??/**", "foo/bar/xx/foo/bar"); !a || c != nil {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 4 {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "x" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[2] != "x" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[3] != "foo/bar" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "**/[xyz]/**", "foo/bar/z/foo/bar"); !a || c != nil {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 3 {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "z" {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[2] != "foo/bar" {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "foo/???/bar", "foo/xyz/bar"); !a || c != nil {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if len(b) != 3 {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[0] != "x" {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[1] != "y" {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[2] != "z" {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch(ctx, "foo/[xyz]/bar", "foo/z/bar"); !a || c != nil {
		t.Errorf("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	} else if b[0] != "z" {
		t.Errorf("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	}
}

func testValueGeneral(t *testing.T) {
	var assert_bool bool
	var assert_value Value
	runcase(t, "testdata/value", "testvalues", func (ctx *testcase) {
		if assert_value == nil {
			t.Errorf("assert: %v", assert_value)
		} else if assert_bool {
			t.Errorf("assert")
		}

		// Globs

		var (
			glob1 = ctx.get("glob1")
			glob2 = ctx.get("glob2")
			val1 = ctx.get("val1")
			val2 = ctx.get("val2")
			val3 = ctx.get("val3")
		)

		if glob1.string(ctx) != "*.c" {
			ctx.err("glob1 is wrong: %T %v", glob1, glob1)
		}

		if glob2.string(ctx) != "**.c" {
			ctx.err("glob2 is wrong: %T %v", glob2, glob2)
		}

		if val1.string(ctx) != "foo.c" {
			ctx.err("val1 is wrong: %T %v", val1, val1)
		}

		if val2.string(ctx) != "foo/bar.c" {
			ctx.err("val2 is wrong: %T %v", val2, val2)
		}

		if val3.string(ctx) != "foo/bar.c" {
			if false { ctx.err("val3 is wrong: %T %v", val3, val3) }
		}

		if a, b, c := glob1.match(ctx, val1); !a {
			ctx.err("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}
		if a, b, c := glob2.match(ctx, val1); !a {
			ctx.err("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}

		if a, b, c := glob1.match(ctx, val2); a == true {
			if false { ctx.err("match(%v, %v): %v %v %v", glob1, val1, a, b, c) }
		}
		if a, b, c := glob2.match(ctx, val2); !a {
			ctx.err("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}

		if a, b, c := glob1.match(ctx, val3); a == true {
			if false { ctx.err("match(%v, %v): %v %v %v", glob1, val1, a, b, c) }
		}
		if a, b, c := glob2.match(ctx, val3); !a {
			ctx.err("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}

		// TODO: test glob.stencil(...)

		// Regexps

		var (
			regexp1 = ctx.get("regexp1")
			regexp2 = ctx.get("regexp2")
			regexp3 = ctx.get("regexp3")
			regexp4 = ctx.get("regexp4")
			regexp5 = ctx.get("regexp5")
			regexp6 = ctx.get("regexp6")
		)

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
		} else if a, b, c := regexp4.match(ctx, val1); !a {
			ctx.err("match(%v, %v): %v %v %v", regexp4, val1, a, b, c)
		} else if s, y := b.(string); !y {
			ctx.err("match(%v, %v): %v %v %v", regexp4, val1, a, b, c)
		} else if s != val1.string(ctx) {
			ctx.err("match(%v, %v): %v %v %v", regexp4, val1, a, b, c)
		} else if len(c) != 0 {
			ctx.err("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		}

		if regexp5.string(ctx) != `fo{2}/bar\.c` {
			ctx.err("regexp5 is wrong: %T %v", regexp5, regexp5)
		} else if a, b, c := regexp5.match(ctx, val2); !a {
			ctx.err("match(%v, %v): %v %v %v", regexp5, val2, a, b, c)
		} else if s, y := b.(string); !y {
			ctx.err("match(%v, %v): %v %v %v", regexp5, val2, a, b, c)
		} else if s != val2.string(ctx) {
			ctx.err("match(%v, %v): %v %v %v", regexp5, val2, a, b, c)
		} else if len(c) != 0 {
			ctx.err("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		}

		if regexp6.string(ctx) != `fo{2}(/o{2}){3}/bar\.c` {
			ctx.err("regexp6 is wrong: %T %v", regexp6, regexp6)
		} else if a, b, c := regexp6.match(ctx, val3); !a {
			ctx.err("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if s, y := b.(string); !y {
			ctx.err("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if s != val3.string(ctx) {
			ctx.err("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if len(c) != 1 {
			ctx.err("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if c[0] != "/oo" {
			ctx.err("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		}

		// TODO: test regexp.stencil(...)

	}, hooks{
		assert: func(ctx Context, v Value, b bool) (res bool) {
			assert_bool, assert_value = b, v
			return true
		},
	})
}
