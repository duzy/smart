//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
)

func testTemplate(ctx *testcase) {
	if s := "xyz"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand2 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if _, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if v.String() != "xxx yyy zzz" {
		ctx.err("%v", tst{v})
	}

	if s := "var.xxx"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test-" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "xxx" {
		ctx.err("%v", tst{x.elems[1]})
	} else if v.String() != "test-xxx" {
		ctx.err("%v", tst{v})
	}

	if s := "var.yyy"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test-" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "yyy" {
		ctx.err("%v", tst{x.elems[1]})
	} else if v.String() != "test-yyy" {
		ctx.err("%v", tst{v})
	}

	if s := "var.zzz"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test-" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "zzz" {
		ctx.err("%v", tst{x.elems[1]})
	} else if v.String() != "test-zzz" {
		ctx.err("%v", tst{v})
	}

	if s := "vars"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 3 {
		ctx.err("%v ; %d", tst{v}, x.len())
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "xxx") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "yyy") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "zzz") != 1 {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "xxx") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "yyy") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "zzz") != 1 {
		ctx.err("%v", tst{v})
	}

	if s := "var2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() < 1 {
		ctx.err("%v ; %d", tst{x}, x.len())
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "var.xxx") != 0 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "var.yyy") != 0 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "var.zzz") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "xyz") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "vars") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "var2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, ".self") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, ".test.1") != 0 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, ".test.2") != 0 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, ".test.10") != 0 {
		ctx.err("%v", tst{v})
	}

	if s := ".test.1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if v.String() != "xxx yyy zzz xxx yyy zzz" {
		ctx.err("%v", tst{v})
	} else if v.string(ctx) != "xxx yyy zzz xxx yyy zzz" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%v", tst{v})
	}

	if s := ".test.3"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if strings.Count(s, "xxx") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "yyy") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "zzz") != 2 {
		ctx.err("%v", tst{v})
	}

	if s := ".test.4"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c=z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a=x a=y a=z b=x b=y b=z c=x c=y c=z" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.5"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 6 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a=x") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b=y") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b=x") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a=x a=y a=x b=y b=x b=y" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.6"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 6 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b=x") != 2 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c=y") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a=x a=y b=x b=x c=y c=x" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.7"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=x c2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=y c2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=z c2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x c2=x c1=y c2=y c1=z c2=z" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.8"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "=x c2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "=y c2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "=z c2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z {}=x c2=x {}=y c2=y {}=z c2=z" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v", tst{v})
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "=x c2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "=y c2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "=z c2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z =x c2=x =y c2=y =z c2=z" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.9"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 9+9-6 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.10"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=x {}=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=y {}=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=z {}=z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x {}=x c1=y {}=y c1=z {}=z" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v", tst{v})
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=x =x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=y =y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=z =z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x =x c1=y =y c1=z =z" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.11"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defVoid {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 9+9-6 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x a2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y a2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=z a2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x b2=x") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y b2=y") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=z b2=z") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.12"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defVoid {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x1 a2=x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y1 a2=y2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1={} a2=z2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x1 b2=x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y1 b2=y2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1={} b2=z2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=x1 {}=x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=y1 {}=y2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1={} {}=z2") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x1 a2=x2 a1=y1 a2=y2 a1={} a2=z2 b1=x1 b2=x2 b1=y1 b2=y2 b1={} b2=z2 c1=x1 {}=x2 c1=y1 {}=y2 c1={} {}=z2" {
		ctx.err("%v", tst{v})
	} else if s := v.string(ctx); s == "" {
		ctx.err("%v", tst{v})
	} else if len(strings.Fields(s)) != 9+9 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x1 a2=x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y1 a2=y2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1= a2=z2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x1 b2=x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y1 b2=y2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1= b2=z2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=x1 =x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1=y1 =y2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "c1= =z2") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x1 a2=x2 a1=y1 a2=y2 a1= a2=z2 b1=x1 b2=x2 b1=y1 b2=y2 b1= b2=z2 c1=x1 =x2 c1=y1 =y2 c1= =z2" {
		ctx.err("%v", tst{v})
	}

	if s := ".test.13"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defVoid {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s := v.String(); s == "" || s == "{}" {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	} else if len(strings.Fields(s)) != 8 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=x1 a2=x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "a1=y1 a2=y2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=x1 b2=x2") != 1 {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, "b1=y1 b2=y2") != 1 {
		ctx.err("%v", tst{v})
	} else if s != "a1=x1 a2=x2 a1=y1 a2=y2 b1=x1 b2=x2 b1=y1 b2=y2" {
		ctx.err("%v", tst{v})
	}
}

func testTemplateForeach(ctx *testcase) {
	var s string

	s = ".test.a"
	if t := unmap_entries(ctx, s); t == nil {
		ctx.err(s)
	} else if len(t) != 1 {
		ctx.err("%s %v", s, t)
	} else if x, y := t[0].(rule_name); !y {
		ctx.err("%v", tst{t[0]})
	} else if x.String() != s {
		ctx.err("%v", tst{x.rule})
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if t, y := r[0].(rule_name); !y {
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
	} else if t.program[0].depends[1].String() != "$(foreach a d e f,foo=a)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print a $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.a" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.b"
	if t := unmap_entries(ctx, s); t == nil {
		ctx.err(s)
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if t, y := r[0].(rule_name); !y {
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
	} else if t.program[0].depends[1].String() != "$(foreach b d e f,foo=b)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print b $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.b" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.c"
	if t := unmap_entries(ctx, s); t == nil {
		ctx.err(s)
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if t, y := r[0].(rule_name); !y {
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
	} else if t.program[0].depends[1].String() != "$(foreach c d e f,foo=c)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print c $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.c" {
		ctx.err("%v", tst{r[0]})
	}
}
