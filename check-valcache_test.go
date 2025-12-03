//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strings"
)

func testValueCache(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
		return
	}

	if p.filemap.v == nil {
		ctx.err("filemap.v")
	} else if x, y := p.filemap.v["*.log"]; !y {
		ctx.err("%v", p.filemap.v)
	} else if len(x.a) != 1 {
		ctx.err("%v", x.a)
	} else if t, y := x.a[0].(filemap); !y {
		ctx.err("%v", tst{(x.a[0])})
	} else if t.pattern.String() != "*.log" {
		ctx.err("%v", tst{(x.a[0])})
	} else if x, y := p.filemap.v["**.o"]; !y {
		ctx.err("%v", p.filemap.v)
	} else if len(x.a) != 1 {
		ctx.err("%v", x.a)
	} else if t, y := x.a[0].(filemap); !y {
		ctx.err("%v", tst{(x.a[0])})
	} else if t.pattern.String() != "**.o" {
		ctx.err("%v", tst{(x.a[0])})
	}

	if p.filemap.v == nil {
		ctx.err("filemap.v")
	} else if x, y := p.filemap.v["."]; !y {
		ctx.err("%v", p.filemap.v)
	} else if len(x.a) != 0 || x.v == nil {
		ctx.err("%v", x)
	} else if x2, y := x.v["deps"]; !y {
		ctx.err("%v", x)
	} else if len(x2.a) != 0 || x2.v != nil {
		ctx.err("%v", x2)
	} else if x3, y := x2.v["??"]; !y {
		ctx.err("%v", x2)
	} else if len(x3.a) != 0 || x3.v != nil {
		ctx.err("%v", x3)
	} else if x4, y := x3.v["??"]; !y {
		ctx.err("%v", x3)
	} else if len(x4.a) != 0 || x4.v != nil {
		ctx.err("%v", x4)
	} else if x5, y := x4.v["??????????"]; !y {
		ctx.err("%v", x4)
	} else if len(x5.a) != 1 || x5.v != nil {
		ctx.err("%v", x5)
	} else if t, y := x5.a[0].(filemap); !y {
		ctx.err("%v", tst{(x5.a[0])})
	} else if t.pattern.String() != ".deps/{=glob ??}/{=glob ??}/{=glob ??????????}" {
		ctx.err("%v", tst{(x5.a[0])})
	} else if __string(ctx,t.pattern) != ".deps/??/??/??????????" {
		ctx.err("%v", tst{(x5.a[0])})
	}

	if p.filemap.v == nil {
		ctx.err("filemap.v")
	} else if x, y := p.filemap.v["foo"]; !y {
		ctx.err("%v", p.filemap.v)
	} else if len(x.a) != 0 || x.v == nil {
		ctx.err("%v", x)
	} else if x21, y := x.v["."]; !y {
		ctx.err("%v", x)
	} else if len(x21.a) != 0 || x21.v != nil {
		ctx.err("%v", x21)
	} else if x22, y := x.v["bar"]; !y {
		ctx.err("%v", x)
	} else if len(x22.a) != 0 || x22.v == nil {
		ctx.err("%v", x22)
	}

	for i, s := range []string{
		".deps/xx/yy/zzzzzzzzzz,.deps/??/??/??????????",
		"foo.log,*.log",
		"foo.o,**.o",
		"foo.c,foo.c",
		"foo.c++,foo.c++",
		// FIXME: "foo/*.c,foo/bar.c",
		// FIXME: "foo/*.c++,foo/bar.c++",
		// FIXME: "**.c",
	}{
		var t = strings.Split(s, ",")
		if a := unmap_files(ctx, p, t[0], nil); len(a) < 1 {
			ctx.err("miss cache for %d. %s %v", i, t[0], a)
		} else if len(t) == 1 && t[0] == "**.c" {
			if len(a) != 2 {
				ctx.err("miss cache for %d. %s %v", i, t[0], tst{a})
			} else if __string(ctx,a[0].pattern) != "foo.c" {
				ctx.err("miss cache for %d. %s %v", i, t[0], tst{a[0]})
			} else if __string(ctx,a[1].pattern) != "foo/bar.c" {
				ctx.err("miss cache for %d. %s %v", i, t[0], tst{a[1]})
			}
		} else if __string(ctx,a[0].pattern) != t[1] {
			ctx.err("miss cache for %d. %s %v", i, t[0], tst{a[0]})
		}
	}

	if s := "val0" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "**.c" {
		ctx.err("%v", v)
	} else if __string(ctx,v) != "**.c" {
		ctx.err("%v", v)
	} else if a := unmap_files(ctx, p, v, nil); len(a) < 1 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, a)
	} else if len(a) != 2 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a})
	} else if __string(ctx,a[0].pattern) != "foo.c" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[0]})
	} else if __string(ctx,a[1].pattern) != "foo/bar.c" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[1]})
	} else if t := unmap_files(ctx, p, v, nil); len(t) != 2 {
		ctx.err("%v %v", v, t)
	} else if !isAny(t[0].string, "foo/bar.c", "foo.c") {
		ctx.err("%v %v", v, t[0])
	} else if !isAny(t[1].string, "foo/bar.c", "foo.c") {
		ctx.err("%v %v", v, t[1])
	}

	if s := "val1" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "**.c++" {
		ctx.err("%v", v)
	} else if __string(ctx,v) != "**.c++" {
		ctx.err("%v", v)
	} else if a := unmap_files(ctx, p, v, nil); len(a) != 2 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, a)
	} else if __string(ctx,a[0].pattern) != "foo.c++" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[0]})
	} else if __string(ctx,a[1].pattern) != "foo/bar.c++" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[1]})
	} else if t := unmap_files(ctx, p, v, nil); len(t) != 2 {
		ctx.err("%v %v", v, t)
	} else if !isAny(t[0].string, "foo/bar.c++", "foo.c++") {
		ctx.err("%v %v", v, t[0])
	} else if !isAny(t[1].string, "foo/bar.c++", "foo.c++") {
		ctx.err("%v %v", v, t[1])
	}

	if s := "val2" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if __string(ctx,v) != "foo.c++" {
		ctx.err("%v", v)
	} else if a := unmap_files(ctx, p, v, nil); len(a) != 1 {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, a)
	} else if __string(ctx,a[0].pattern) != "foo.c++" {
		ctx.err("%v: miss cache : %s %v", v, tst{v}, tst{a[0]})
	} else if t := unmap_files(ctx, p, v, nil); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].string != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	} else if __string(ctx,t[0].pattern) != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%v %v", v, tst{v})
	} else if ident(ctx,f) != "foo.c++" {
		ctx.err("%v %v", v, f)
	}

	if s := "val3" ; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if __string(ctx,v) != "foo.o" {
		ctx.err("%v %v", v, tst{v})
	} else if a := unmap_files(ctx, p, v, nil); len(a) != 1 {
		ctx.err("%v: miss cache : %v %v", v, tst{v}, a)
	} else if __string(ctx,a[0].pattern) != "**.o" {
		ctx.err("%v: miss cache : %v %v", v, tst{v}, tst{a[0].pattern})
	} else if t := unmap_files(ctx, p, v, nil); len(t) != 1 {
		ctx.err("%v %v ; %v", v, tst{v}, t)
	} else if t[0].string != "foo.o" {
		ctx.err("%v %v ; %v", v, tst{v}, t[0])
	} else if t[0].String() != "{filemap=**.o name=foo.o}" {
		ctx.err("%s : %v : %v", s, t[0], t[0].pattern)
	} else if _, y := t[0].pattern.(*globpat); !y {
		ctx.err("%v %v ; %v", v, tst{v}, t[0])
	} else if t[0].project != p {
		ctx.err("%v ; %v %v %v", tst{v}, t[0], t[0].paths, t[0].project)
	} else if t[0].paths == nil {
		ctx.err("%v %v ; %v %v", v, tst{v}, t[0], t[0].paths)
	} else if t := t[0].paths[0]; t.String() != p.absPath+"/.tmp" {//"$//.tmp"
		ctx.err("%v %v", t, tst{t})
	} else if t := unmap_files(ctx, p, v, nil); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].string != "foo.o" {
		ctx.err("%v %v", v, t[0])
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%v ; %v", tst{v}, t)
	} else if ident(ctx,f) != "foo.o" {
		ctx.err("%v %v", v, f)
	}

	if pat, s := ".deps/xx/yy/zzzzzzzzzz", "val4"; false {} else
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != pat {
		ctx.err("%v %v", v, tst{v})
	} else if __string(ctx,v) != pat {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, p, v, nil); len(t) != 1 {
		ctx.err("%v %v %v", v, tst{v}, t)
	}

	if s := "sources" ; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err("sources is nil")
	} else if v := evoke(ctx, d, nil, nil); v == nil {
		ctx.err("sources is wrong: %v %v", d, v)
	} else if s := __string(ctx,v); strings.Count(s, "foo.c") != 2 {
		ctx.err("sources is wrong: %v", v) // NOTE: "foo.c" counts foo.c foo.c++
	} else if strings.Count(s, "foo.c++") != 1 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.c") != 2 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.c++") != 1 {
		ctx.err("sources is wrong: %v", v)
	}

	if s := "objects" ; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err("objects is nil")
	} else if v := evoke(ctx, d, nil, nil); v == nil {
		ctx.err("objects is wrong: %v %v", d, v)
	}
}
