//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func test__foreach5(ctx *testcase) {
	var s string

	s = ".test.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "o"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.o.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.o.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a~ $(foreach $(foreach $1 $2,$(.test.x.o.$_)),-ao$_) ~a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ ~a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a~ -aox.o.a -aox.o.b ~a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_) ~b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ -bo{&(.test.x.o.a)} -bo{&(.test.x.o.b)} ~b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.0"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x $1)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else {
		if v := ctx.val(d); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(.test.x.x $1)"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		}
		if v := ctx.val(d, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.a) &(.test.x.&(.test.o).a)"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "a~ ~a x.o.a"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		}
		if v := ctx.val(d, nil, []string{"b", "c"}); v == nil {
			ctx.err("%v → %v", tst{d}, tst{d.value})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v}) // "&(.test.x.a) &(.test.x.&(.test.o).a) {&(.test.x.b)} {&(.test.x.c)} {&(.test.x.&(.test.o).b)} {&(.test.x.&(.test.o).c)}"
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v}) // "a~ ~a x.o.a b~ ~b ~c~ x.o.b x.o.c"
		}
	}

	s = ".test.x.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1,$2)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a) &(.test.x.&(.test.o).a) &(.test.x.b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,nil), v); t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", []string{"b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if x := expand(_final(ctx),v); x == nil {
		ctx.err("%v", tst{v})
	} else if l, y := x.(*list); !y {
		ctx.err("%v", tst{x})
	} else if elems := merge(l.elems...); l.len() != 2 || len(elems) != 4 {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v ; %d, %d", tst{x}, l.len(), len(elems))
	} else if s, t := x.String(), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), x), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a b c,{}) &(.test.x.&(.test.o).a) &(.test.x.b a b c,{}) &(.test.x.&(.test.o).b) &(.test.x.c a b c,{}) &(.test.x.&(.test.o).c)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b -aox.o.c ~a x.o.a b~ -box.o.a -box.o.b -box.o.c ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y ,$2)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1,$2)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b) &(.test.x.&(.test.o).a) &(.test.x.b a,b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.10"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{d.value})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.11"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.12"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b) &(.test.x.&(.test.o).a) &(.test.x.b a,b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b) &(.test.x.&(.test.o).a) &(.test.x.b a,b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}
