//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "reflect"
	"testing"
)

type test_mod_1 struct { modifier_ }
func (ctx *test_mod_1) v(args ...Value) (result interface{}) {
	return append(args, MakeBareword(ctx.Position(), "test_mod_1"))
}

func TestValueModifier(t *testing.T) {
	modifiers[`test-mod-1`] = reflect.TypeOf((*test_mod_1)(nil)).Elem()

	defer func(o commandLineOpts) { options = o
		delete(modifiers, `test-mod-1`)
	} (options)
	options.failOnErrors = false

	var ctx = load_testcase(t, "testdata/modifier", "testmodifier")

	if v := ctx.get("val"); v == nil {
		ctx.err("val")
	} else if v.String() != "foobar" {
		ctx.err("%T %v", v, v)
	} else if s := v.strval(ctx); s != "foobar" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get("foo"); v == nil {
		ctx.err("foo")
	} else if v.String() != "$(val) test_mod_1" {
		ctx.err("%T %v", v, v)
	} else if l, y := v.(*List); !y {
		ctx.err("%T %v", v, v)
	} else if len(l.Elems) != 2 {
		ctx.err("%T %v -> %v", v, v, l.Elems)
	} else if s := v.strval(ctx); s != "foobar test_mod_1" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	ctx.flush()
}
