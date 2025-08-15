//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
)

func test__trimprefix(ctx *testcase) {
	var pv Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if pv = p.value; pv == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = pv.string(ctx); ps == "" {
		ctx.err("%v", tst{pv})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{pv})
		return
	} else if s := "/testdata/builtins/trimprefix"; !strings.HasSuffix(ps, s) {
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
	} else if _, y := p.elems[0].(*globpat); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*word); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
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
	} else if _, y := p.elems[0].(*percpat); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*word); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
	} else if !strings.HasPrefix(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, pv, b, c)
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
	} else if t, y := p.elems[0].(*punct); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if t.token != PROOT {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if t.len() != 1 {
		ctx.err("%v ; %v", tst{t}, t.len())
	} else if _, y := t.elems[0].(*globmeta); !y {
		ctx.err("%v", tst{t.elems[0]})
	} else if _, y := p.elems[2].(*word); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if s, t := p.String(), "/**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "/**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a { // partially matched
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(x) < 0 || x[0] != "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(c) < 0 || c[0] == "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, "/"+joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
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
	} else if t, y := p.elems[0].(*punct); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if t.token != PROOT {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if _, y := p.elems[2].(*word); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if s, t := p.String(), "/%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := p.string(ctx), "/%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, c := v.match(ctx, pv); a {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if x, y := b.([]string); !y {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(x) < 0 || x[0] != "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if len(c) < 0 || c[0] == "" {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.Contains(ps, joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, joinpath(x...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if !strings.HasPrefix(ps, "/"+joinpath(c...)) {
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	}

	s = "val1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val6"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(ctx), "builtins/trimprefix"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}
