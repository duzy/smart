//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__foreach3(ctx *testcase) {
	var s string

	s = ".test.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,$_$3)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{d.value})
	} else {
		if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "acc bcc"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := __string(src(ctx,nil), v); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "", []string{"1", "2", "3"}, "x"); v == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "1x 2x 3x"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := __string(src(ctx,nil), v); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "a{} b{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "a b"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "ax bx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := __string(src(ctx,nil), v); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.x"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := v.String(), "$(foreach $1 $2,$(addprefix std=,&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", nil); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)} std={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std={&(.test.if.x)} std={&(.test.if.y)} std={&(.test.if.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.y"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
		if v := ctx.val(d, defExpand1, "if.x"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.x),std={&(.test.if.x)}) $(if &(.test.{$2}),std=&(.test.{$2}))
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", nil); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.x),std={&(.test.if.x)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.y),std={&(.test.if.y)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.x),std={&(.test.if.x)}) $(if &(.test.if.y),std={&(.test.if.y)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "zzz", nil); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.zzz),std={&(.test.zzz)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.zzz),std={&(.test.zzz)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.z"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := d.value.String(), "$(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
		if v := ctx.val(d, defExpand1, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.x)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.x)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.y)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.x) std=&(.test.if.y)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "zzz", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}
}
