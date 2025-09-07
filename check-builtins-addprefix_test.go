//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__addprefix(ctx *testcase) {
	var s string

	s = "val1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix -std=,foo)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "-std=foo"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-std=foo"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix -std=,foo bar)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "-std=foo -std=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-std=foo -std=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix -foo=,bar &(none))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v = ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-foo=bar -foo={&(none)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v = ctx.val(d, defExpand2); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := v.String(), "$(addprefix std=,&(.test.$1))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "std=test std=null"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "std=ax std=ay std=az"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=ax std=ay std=az"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)} std={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "std=ax std=ay std=az std=bx std=by std=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std=ax std=ay std=az std=bx std=by std=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo,bar &(none))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foobar foo{&(none)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo bar,{}=xxx)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo=xxx bar=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx bar=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo &(.test.$1),{}=xxx)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo=xxx test=xxx null=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx {&(.test.)}=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo=xxx test=xxx null=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx {&(.test.a)}=xxx {&(.test.b)}=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo=xxx ax=xxx ay=xxx az=xxx bx=xxx by=xxx bz=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx ax=xxx ay=xxx az=xxx bx=xxx by=xxx bz=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo &(.test.$1),{}=&(.test.$1))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo=test foo=null test=test test=null null=test null=null"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo={&(.test.)} {&(.test.)}={&(.test.)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo=test foo=null test=test test=null null=test null=null"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo={&(.test.a)} foo={&(.test.b)} {&(.test.a)}={&(.test.a)} {&(.test.a)}={&(.test.b)} {&(.test.b)}={&(.test.a)} {&(.test.b)}={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "foo=ax foo=ay foo=az foo=bx foo=by foo=bz ax=ax ax=ay ax=az ay=ax ay=ay ay=az az=ax az=ay az=az ax=bx ax=by ax=bz ay=bx ay=by ay=bz az=bx az=by az=bz bx=ax bx=ay bx=az by=ax by=ay by=az bz=ax bz=ay bz=az bx=bx bx=by bx=bz by=bx by=by by=bz bz=bx bz=by bz=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=ax foo=ay foo=az foo=bx foo=by foo=bz ax=ax ax=ay ax=az ax=bx ax=by ax=bz ay=ax ay=ay ay=az ay=bx ay=by ay=bz az=ax az=ay az=az az=bx az=by az=bz bx=ax bx=ay bx=az bx=bx bx=by bx=bz by=ax by=ay by=az by=bx by=by by=bz bz=ax bz=ay bz=az bz=bx bz=by bz=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val9"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix fo-{},&(.test.$1.x.$2.y.$3.z))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "fo-{&(.test.{}.x.{}.y.{}.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "fo-{&(.test.a.x.1.y.0.z)} fo-{&(.test.a.x.2.y.0.z)} fo-{&(.test.a.x.3.y.0.z)} fo-{&(.test.b.x.1.y.0.z)} fo-{&(.test.b.x.2.y.0.z)} fo-{&(.test.b.x.3.y.0.z)} fo-{&(.test.c.x.1.y.0.z)} fo-{&(.test.c.x.2.y.0.z)} fo-{&(.test.c.x.3.y.0.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "fo-ax fo-ay fo-az fo-bx fo-by fo-bz fo-cx fo-cy fo-cz fo-dx fo-dy fo-dz fo-ex fo-ey fo-ez fo-fx fo-fy fo-fz"; s != t {
		note(pc(ctx,v), "%s", ts(v))
		ctx.err("%s != %s", s, t)
	} else if v := ctx.val(d, defExpand2, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "fo-ax fo-ay fo-az fo-bx fo-by fo-bz fo-cx fo-cy fo-cz fo-dx fo-dy fo-dz fo-ex fo-ey fo-ez fo-fx fo-fy fo-fz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}
