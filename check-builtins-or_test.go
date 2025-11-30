//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__or(ctx *testcase) {
	if v := ctx.val("val11.0"); v == nil {
		ctx.err("val11.0")
	} else if v.String() != "-no -yes -false -true" {
		ctx.err("%v", tst{v})
	} else if s, t := __string(src(ctx,nil),v), "-no -yes -false -true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else {
		for _, t := range l.elems {
			if f, y := t.(flag); !y {
				ctx.err("%v", tst{t})
			} else if _, y := f.Value.(*word); !y {
				ctx.err("%v", tst{f.Value})
			} else if !__true(ctx,f) {
				ctx.err("%v", tst{t})
			} else if !__true(ctx,f.Value) {
				ctx.err("%v", tst{f.Value})
			}
		}
	}

	if v := ctx.val("val11"); v == nil {
		ctx.err("val11")
	} else if v.String() != "-no" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,nil),v); s != "-no" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}

	if v := ctx.val("val12"); v == nil {
		ctx.err("val12")
	} else if v.String() != "-yes" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,nil),v); s != "-yes" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}

	if v := ctx.val("val13"); v == nil {
		ctx.err("val13")
	} else if v.String() != "-false" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,nil),v); s != "-false" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}
}
