//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "reflect"
)

type test_mod_1 struct { modifier_ }
func (ctx *test_mod_1) v(args ...Value) any {
	return append(args, _word(_position(ctx), "test_mod_1"))
}

func testValueModifierInit() {
	modifiers[`test-mod-1`] = reflect.TypeOf((*test_mod_1)(nil)).Elem()
}

func testValueModifier(ctx *testcase) {
	defer func() { delete(modifiers, `test-mod-1`) } ()

	if s := "val"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "foobar" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,d), v); s != "foobar" {
		ctx.err("%v → %s", tst{v}, s)
	}

	if s := "val1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{(test-mod-1 $(val))}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foobar test_mod_1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if s := "val2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val) test_mod_1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foobar test_mod_1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if s := "val3"; true {
	} else if d := ctx.def(s); d == nil { // TODO: {(plain text) text goes here...}
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*strlit); !y {
		ctx.err("%v", tst{v})
	} else if s := "this is a 'string' of plain  `text`."; s != t.s {
		ctx.err("%v", tst{v})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val4"; true {
	} else if d := ctx.def(s); d == nil { // TODO: {(plain c++) c++ code goes here...}
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*strlit); !y {
		ctx.err("%v", tst{v})
	} else if s := "int main() { return 0; }"; s != t.s {
		ctx.err("%v", tst{v})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}
