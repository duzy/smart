//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"path/filepath"
)

func test__file0(ctx *testcase) {
	var proj = _project(ctx)
	if pat, str := ".test/a/**.c", ".test/a/b/c/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.string != str {
		ctx.err("%s: %v", str, m.string)
	} else if __string(ctx, m.pattern) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err(str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.string != str {
		ctx.err("%s: %v", str, m.string)
	} else if __string(ctx, m.pattern) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val1.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if s := __string(ctx, val); s != str {
		ctx.err("%v: %s != %s", tst{val}, s, str)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if   s := "val1.2" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if __string(ctx, val) != str {
		ctx.err("%v", tst{val})
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	}

	if pat, str := ".test/xx/*.c", ".test/xx/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.string != str {
		ctx.err("%s: %v", str, m.string)
	} else if __string(ctx, m.pattern) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val2.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if   s := "val2.2" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	}

	if pat, str := ".test/xx/yy/*.c", ".test/xx/yy/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.string != str {
		ctx.err("%s: %v", str, m.string)
	} else if __string(ctx, m.pattern) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val3" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	}

	if pat, str := ".test/xx/yy/zz/*.c", ".test/xx/yy/zz/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.string != str {
		ctx.err("%s: %v", str, m.string)
	} else if __string(ctx, m.pattern) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val4" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	}

	if s := "val5" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if _, y := val.(*null); !y {
		ctx.err("%v", tst{val})
	} else if val.String() != "{}" {
		ctx.err("%v", tst{val})
	} else if __string(ctx, val) != "" {
		ctx.err("%v", tst{val})
	}

	if pat, str := "**.auto", ".test/a/b/c.auto"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("%s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.string != str {
		ctx.err("%s: %v", str, m.string)
	} else if __string(ctx, m.pattern) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if s := "p1" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if __string(ctx, x) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, proj, v, nil); t == nil {
		ctx.err("%v %v", v, tst{v})
	} else if len(t) != 1 {
		ctx.err("%v %v %v", v, tst{v}, t)
	} else if m := t[0]; m.string != str {
		ctx.err("%v: %v", tst{v}, m.string)
	} else if __string(ctx, m.pattern) != pat {
		ctx.err("%v: %v", tst{v}, m.pattern)
	}

	if str := ".test/a/b/c.none" ; false {} else
	if t := unmap_files(ctx, proj, str, nil); t != nil {
		ctx.err("%v", str)
	} else if s := "p2" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v", s)
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if __string(ctx, x) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, proj, v, nil); t != nil {
		ctx.err("%v %v", v, tst{v})
	}
}

func test__file(ctx *testcase) {
	var fullFooTxt = filepath.Join(_workdir(ctx), "foo.txt")

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if f := (as{v}.file(ctx)); f == nil {
		ctx.err("%v", tst{v})
	} else if o := (as{v}.fullname(ctx)); o.Value == nil {
		ctx.err("%v ; %v", tst{v}, tst{o.Value})
	} else if f, y := o.Value.(*file); !y || f == nil {
		ctx.err("%v", tst{o.Value})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	} else if cmp(ctx, o, f) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if cmp(ctx, f, o) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if p := _pathStr(ctx, f.fullname()); p == nil {
		ctx.err("%v %v", tst{v}, f)
	} else if true {
		// hold line ...
	} else if cmp(ctx, p, o) != cmpEqual {
		ctx.err("%v %v", tst{p}, tst{o})
	} else if cmp(ctx, o, p) != cmpEqual {
		ctx.err("%v %v", tst{o}, tst{p})
	} else if s, t := v.String(), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := __string(src(ctx,d), v), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=file foo.txt}" {
		ctx.err("%v", tst{v})
	} else if __string(src(ctx,d), v) != "foo.txt" {
		ctx.err("%v", tst{v})
	} else if f, y := v.(*file); !y || f == nil {
		ctx.err("%v", tst{v})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=file foo.txt}" {
		ctx.err("%v", tst{v})
	} else if __string(src(ctx,d), v) != fullFooTxt { o, y := v.(fullname)
		ctx.err("%v ; %v %v", tst{v}, tst{o.Value}, y)
	} else if o, y := v.(fullname); !y {
		ctx.err("%v", tst{v})
	} else if f, y := o.Value.(*file); !y || f == nil {
		ctx.err("%v", tst{o.Value})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	}
}
