//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
)

func test__trimsuffix(ctx *testcase) {
	var pv Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if pv = p.value; pv == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = __string(ctx,pv); ps == "" {
		ctx.err("%v", tst{pv})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{pv})
		return
	} else if s := "/testdata/builtins/trimsuffix"; !strings.HasSuffix(ps, s) {
		ctx.err("%v → %s", tst{pv}, ps)
		return
	}

	s := "pat0"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "testdata/**"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/**"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, c := match(reversal{ctx}, v, pv); a {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	} else if s, t := sfmt("%v %v", b, c), "[testdata builtins trimsuffix] [builtins/trimsuffix]"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	}

	s = "pat1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "testdata/%%"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/%%"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, c := match(reversal{ctx}, v, pv); a {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	} else if s, t := sfmt("%v %v", b, c), "[testdata builtins trimsuffix] [builtins/trimsuffix]"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	}

	s = "pat2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if x, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if x.len() != 1 {
		ctx.err("%v ; %v", tst{x}, x.len())
	} else if x, y := p.elems[2].(*punct); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if x.token != PTAIL {
		ctx.err("%v ; %v", tst{x}, x.token)
	} else if x.String() != "" {
		ctx.err("%v ; %v", tst{x}, x.token)
	} else if s, t := p.String(), "testdata/**/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/**/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, c := match(reversal{ctx}, v, pv); a { // partially matched
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if b != nil || c != nil {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	}

	s = "pat3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if t, y := p.elems[2].(*punct); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if t.token != PTAIL {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if s, t := p.String(), "testdata/%%/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/%%/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, c := match(reversal{ctx}, v, pv); a {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	} else if b != nil || c != nil {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	}

	var ds string
	if v := ctx.val("d"); v == nil {
		ctx.err("d")
	} else if x, y := unbox(v).(*path); !y {
		ctx.err("%v", tst{v})
	} else if x.len() < 2 {
		ctx.err("%v", tst{x})
	} else if t, y := x.elems[0].(*punct); !y {
		ctx.err("%v", tst{x.elems[0]})
	} else if PROOT != t.token {
		ctx.err("%v ; %v", tst{t}, x)
	} else if ds = __string(ctx,v); ds == "" {
		ctx.err("%v", tst{v})
	} else if !strings.HasPrefix(ds, "/") {
		ctx.err("%v %s", tst{v}, ds)
	}

	s = "val1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val6"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}
