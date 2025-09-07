//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

type testAssertStruct struct {
	bools []bool
	vals []Value
}
func testAssertHook(ctx Context, v Value, b bool, i any) {
	s := i.(*testAssertStruct)
	s.bools, s.vals = append(s.bools, b), append(s.vals, v)
}
func testAssert(ctx testcase1) {
	s := "foo"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if v != d.value {
		ctx.err("%v != %v", v, d.value)
	} else if v.String() != "foo" {
		ctx.err("%v", tst{v})
	} else if v.string(ctx) != "foo" {
		ctx.err("%v", tst{v})
	}

	if s, y := ctx.i.(*testAssertStruct); !y {
		ctx.err("%T", ctx.i)
	} else if len(s.bools) != 12 {
		ctx.err("%v, %v, %v %v", s.vals, s.bools, len(s.vals), len(s.bools))
	} else if len(s.vals) != len(s.bools) {
		ctx.err("%v %v", s.vals, s.bools)
	} else if i := 0; s.vals[i].String() != "{=true}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 1; s.vals[i].String() != "{=false}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 2; s.vals[i].String() != "{=yes}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 3; s.vals[i].String() != "{=no}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 4; s.vals[i].String() != "" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 5; s.vals[i].String() != "{=undef x}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 6; s.vals[i].String() != "{}" || s.bools[i] { // {=null}
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].String() != "x" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].string(ctx) != "x" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 8; s.vals[i].String() != "foobar" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if _, y := s.vals[i].(*word); !y {
		ctx.err("%v %v", tst{s.vals[i]}, s.vals[i])
	} else {
		type rec struct{ string; bool }
		for i, r := range []rec{
			rec{"{=true}", true},
			rec{"{=false}", false},
			rec{"{=yes}", true},
			rec{"{=no}", false},
			rec{"", false},
			rec{"{=undef x}", false},
			rec{"{}", false},
			rec{"x", true},
			rec{"foobar", true},
			rec{"1", true},
			rec{"0", false},
			rec{"{=true}", true}, // $(equal $(foo),foo)
		}{
			if t := s.vals[i].String(); t != r.string {
				ctx.err("%s != %s : %s", t, r.string, tst{s.vals[i]})
			} else if s.bools[i] != r.bool {
				ctx.err("%v != %v : %v : %v", s.bools[i], r.bool, s.vals[i], tst{s.vals[i]})
			}
		}
	}
}
