//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"testing"
)

func testRules0(t *testing.T) {
	var ctx = load_testcase(t, "testdata/rule/0", "testrules0")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	if true {} else
	if foo := ctx.get("foo"); foo == nil {
		ctx.err("foo")
	} else if foo.String() != "$0 $1 $2 $3 $4 $5 $6 $7 $8 $9" {
		ctx.err("%T %v", foo, foo)
	} else if s := foo.string(ctx); s != "" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}

	ctx.flush()
}

func testRules1(t *testing.T) {
	var ctx = load_testcase(t, "testdata/rule/1", "testrules1")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

	if r := ctx.rule(".test.foobar"); r == nil {
		ctx.err(".test.foobar")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "fxxbar" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*bareword); !y {
		ctx.err("%T %v", v, v)
	}

	if r := ctx.rule(".test.foobaz"); r == nil {
		ctx.err(".test.foobaz")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%T %v", v, v)
	}

	if r := ctx.rule(".test.foobay"); r == nil {
		ctx.err(".test.foobay")
	} else if v := inv(ctx, r); v == nil {
		ctx.err("%v", r)
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if _, y := v.(*barecomp); !y {
		ctx.err("%T %v", v, v)
	}

	if v := ctx.get(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if v.String() != "fxxbar" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "fxxbar" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	if v := ctx.get(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if v.String() != ".test.fxx" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != ".test.fxx" {
		ctx.err("%T %v -> %s", v, v, s)
	}

	ctx.flush()
}
