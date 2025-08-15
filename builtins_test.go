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
	if pat, str := ".test/a/**.c", ".test/a/b/c/foo.c"; false {
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err(str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if   s := "val1.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if s := val.string(ctx); s != str {
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
	} else if val.string(ctx) != str {
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
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
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
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
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
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
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
	} else if val.string(ctx) != "" {
		ctx.err("%v", tst{val})
	}

	if pat, str := "**.auto", ".test/a/b/c.auto"; false {
	} else if t := unmap_files(ctx, str); t == nil {
		ctx.err("%s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%s: %v", str, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%s: %v", str, m.pattern)
	} else if s := "p1" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if x.string(ctx) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, v); t == nil {
		ctx.err("%v %v", v, tst{v})
	} else if len(t) != 1 {
		ctx.err("%v %v %v", v, tst{v}, t)
	} else if m := t[0]; m.name != str {
		ctx.err("%v: %v", tst{v}, m.name)
	} else if m.pattern.string(ctx) != pat {
		ctx.err("%v: %v", tst{v}, m.pattern)
	}

	if str := ".test/a/b/c.none" ; false {} else
	if t := unmap_files(ctx, str); t != nil {
		ctx.err("%v", str)
	} else if s := "p2" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v", s)
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if x.string(ctx) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, v); t != nil {
		ctx.err("%v %v", v, tst{v})
	}
}

func test__contains(ctx *testcase) {
	var s string

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(contains a,a b c $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
		// hold line
		// hold line
	} else if x := ctx.val(d, "x"); x == nil {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "{=true}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := x.string(src(ctx,d)), "true"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(contains x b c,a b c $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if x := ctx.val(d, "x"); x == nil {
		ctx.err("%v", tst{v})
	} else if s, t := x.String(), "{=true}"; s != t { // $(contains b c,a b c x)
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := x.string(src(ctx,d)), "true"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(contains x,a b c $1)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(src(ctx,d)); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	}
}

func test__contains2(ctx *testcase) {
	var s string

	s = "val"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "${foo}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a b c foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "${foo}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a b c foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.y"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a b c foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.z"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a b c foo"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "true"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func test__join(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(join foo bar xx yy zz,-)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(join &(target.arch) &(target.vendor) &(target.os) &(target.abi),-)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "foo-bar-a-0"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{{&(target.arch) &(XXX) &(target.vendor) &(target.os) &(target.abi)}-}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "foo-bar-a-0"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func test__logic(ctx *testcase) {
	s := "val1"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or(-final) &(none),a)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or &(none),a)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "a", v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or a,&(none))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "a"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "val6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(and $1,$2,$3)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(src(ctx,d)); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "c"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "(variant/$(or(-final) $(base &(variant)),bootstrap))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := "(variant/bootstrap)", v.expand(_final(ctx)); s != t.String() {
		ctx.err("%v : %s != %s", tst{t}, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "(variant/bootstrap)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "variant/bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or $(base &(variant)),bootstrap)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = "x5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(base $(or &(variant),bootstrap))"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "bootstrap"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func test__xor(ctx *testcase) {
	if d := ctx.def("val14.1"); d == nil {
		ctx.err("val14.1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.true(ctx) {
		ctx.err("%v", tst{v})
	} else if t := v.String(); t != "{}" {
		ctx.err("%v ⇒ %s", tst{v}, t)
	} else if t := v.string(src(ctx,d)); t != "" {
		ctx.err("%v ⇒ %s", tst{v}, t)
	}

	if d := ctx.def("val14.2"); d == nil {
		ctx.err("val14.2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if !v.true(ctx) {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%v ⇒ %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "true"; s != t {
		ctx.err("%v ⇒ %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val14.3"); d == nil {
		ctx.err("val14.3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=true}" {
		ctx.err("%v", tst{v})
	} else if s := v.string(src(ctx,d)); s != "true" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}

	if d := ctx.def("val14.4"); d == nil {
		ctx.err("val14.4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{}" {
		ctx.err("%v", tst{v})
	} else if s := v.string(src(ctx,d)); s != "" {
		ctx.err("%v ⇒ %s", tst{v}, s)
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
	} else if o.cmp(ctx, f) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if f.cmp(ctx, o) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if cmp(ctx, o, f) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if cmp(ctx, f, o) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if p := _pathstr(ctx, f.fullname()); p == nil {
		ctx.err("%v %v", tst{v}, f)
	} else if t := o.cmp(ctx, p); t != cmpEqual {
		ctx.err("%v, %v %v", t, tst{o}, tst{p})
	} else if true {
		// hold line ...
	} else if p.cmp(ctx, o) != cmpEqual {
		ctx.err("%v %v", tst{p}, tst{o})
	} else if cmp(ctx, p, o) != cmpEqual {
		ctx.err("%v %v", tst{p}, tst{o})
	} else if cmp(ctx, o, p) != cmpEqual {
		ctx.err("%v %v", tst{o}, tst{p})
	} else if s, t := v.String(), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := v.string(src(ctx,d)), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=file foo.txt}" {
		ctx.err("%v", tst{v})
	} else if v.string(src(ctx,d)) != "foo.txt" {
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
	} else if v.string(src(ctx,d)) != fullFooTxt { o, y := v.(fullname)
		ctx.err("%v ; %v %v", tst{v}, tst{o.Value}, y)
	} else if o, y := v.(fullname); !y {
		ctx.err("%v", tst{v})
	} else if f, y := o.Value.(*file); !y || f == nil {
		ctx.err("%v", tst{o.Value})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	}
}
