//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testGlob(ctx *testcase) {
	testGlobMatch(ctx)

	if val := ctx.val("pat1.0"); val == nil {
		ctx.err("pat1.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v : %v", tst{val}, val)
	} else if p.len() != 2 {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/x**y"; s != t {
		ctx.err("%v: %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {4:12:punct .} {4:13:word test}} {=glob {4:18:word x} {4:19:meta **} {4:21:word y}}}"; s != t {
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
	} else if s, t := ts(val), "{=path {=compound {12:12:punct .} {12:13:word test}} {=glob {12:18:word x} {12:19:meta **} {12:21:word y}} {12:23:word z}}"; s != t {
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

	if d := ctx.def("pat3.0"); d == nil {
		ctx.err("pat3.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if val.String() != ".test/x**y/x**" {
		ctx.err("%v", tst{val})
	} else if val.string(src(ctx,d)) != ".test/x**y/x**" {
		ctx.err("%v: %s", tst{val}, val.string(src(ctx,d)))
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
	} else if s, t := ts(val), "{=path {=compound {18:12:punct .} {18:13:word test}} {=glob {18:18:word x} {18:19:meta **} {18:21:word y}} {=glob {18:23:word x} {18:24:meta **} {18:26:word y}}}"; s != t {
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
	} else if s, t := ts(val), "{=path {=compound {21:12:punct .} {21:13:word test}} {=glob {21:18:word x} {21:19:meta **}} {=glob {21:22:meta **} {21:24:word y}}}"; s != t {
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
	} else if s, t := ts(val), "{=path {=compound {24:12:punct .} {24:13:word test}} {=glob {24:18:meta **} {24:20:word y}} {=glob {24:22:meta **} {24:24:word y}}}"; s != t {
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
	} else if s, t := ts(val), "{=path {=compound {26:12:punct .} {26:13:word test}} {=glob {26:18:meta **} {26:20:word y}} {=glob {26:22:meta **} {26:24:word y}} {26:26:word z}}"; s != t {
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
	} else if s, t := ts(val), "{=path {=compound {28:12:punct .} {28:13:word test}} {=glob {28:18:meta **}} {=glob {28:21:meta **} {28:23:word z}}}"; s != t {
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
	} else if s, t := ts(val), "{=path {=compound {38:12:punct .} {38:13:word test}} {=glob {38:18:meta *}} {=glob {38:20:meta *}} {=glob {38:22:meta *} {38:23:punct .} {38:24:word h}}}"; s != t {
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

	if d := ctx.def("pat.0"); d == nil {
		ctx.err("pat.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if val.string(src(ctx,d)) != "**.auto" {
		ctx.err("%v", tst{val})
	} else if d := ctx.def("pat13.1"); d == nil {
		ctx.err("pat13.1: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, c := val.match(ctx, v); !a {
		ctx.err("%v ; %v %v %v", val, a, b, c)
	} else if d := ctx.def("pat13.2"); d == nil {
		ctx.err("pat13.2: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, c := val.match(ctx, v); a {
		ctx.err("%v ; %v %v %v", tst{val}, a, b, c)
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
