//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"strconv"
)

func testAuto(ctx *testcase) {
	if c := _universe(ctx); c == nil {
		ctx.err("context.cast")
	}
	{
		ac := automatic{Context:ctx, defs:make(def_map)}
		ac.args(ctx, []Value{ease(ctx, []string{"a", "b", "c"})})
		if _automatic(&ac) != &ac {
			ctx.err("%v", Context(&ac))
		} else if len(ac.defs) != 1 { // maxDigitAutoNum
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if s := d.value.String(); s != "'a' 'b' 'c'" {
			ctx.err("%v → %s", tst{d.value}, s)
		} else if s := d.value.string(src(ctx,d)); s != "a b c" {
			ctx.err("%v → %s", tst{d.value}, s)
		} else if d, _ := ac.do(ctx, find_auto{"1"}).(*def); d == nil {
			ctx.err("%v", ac.defs)
		} else if d := auto_find(&ac, "1"); d == nil {
			ctx.err("%v", ac.defs)
		} else if v := auto_get(&ac, "1"); v == nil {
			ctx.err("%v", ac.defs)
		} else if s := v.string(src(ctx,d)); s != "a b c" {
			ctx.err("%v → %s", tst{v}, s)
		} else if d, v := ac.amend(ctx, "1", ease(ctx, "a")); d == nil {
			ctx.err("%v", ac.defs)
		} else if v == nil {
			ctx.err("%v", ac.defs)
		} else if v.String() != "'a' 'b' 'c'" {
			ctx.err("%v %v", ac.defs, v)
		} else if d.value == nil {
			ctx.err("%v %v", ac.defs, d)
		} else if d.value.String() != "'a'" {
			ctx.err("%v %v", ac.defs, d)
		} else if false { for i := 2; i <= maxDigitAutoNum; i += 1 {
			if d, y := ac.defs[strconv.Itoa(i)]; !y {
				ctx.err("%d: %v", i, ac.defs)
			} else if d.value != nil {
				ctx.err("%v", d)
			}
		}}
	}
	{
		ac := automatic{Context:ctx, defs:make(def_map)}
		ac.args(ctx, ease(ctx, []string{"a", "b", "c"}).(*list).elems)
		if len(ac.defs) != 3 { // maxDigitAutoNum
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "a" {
			ctx.err("%v", tst{d.value})
		} else if d, y := ac.defs["2"]; !y {
			ctx.err("2: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "b" {
			ctx.err("%v", tst{d.value})
		} else if d, y := ac.defs["3"]; !y {
			ctx.err("3: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "c" {
			ctx.err("%v", tst{d.value})
		} else if false { for i := 4; i <= maxDigitAutoNum; i += 1 {
			if d, y := ac.defs[strconv.Itoa(i)]; !y {
				ctx.err("%d: %v", i, ac.defs)
			} else if d.value != nil {
				ctx.err("%v", d)
			}
		}}
	}

	if d := ctx.def("foo0"); d == nil {
		ctx.err("foo0")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v", l.elems)
	} else if s, t := d.value.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := d.value.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(src(ctx,d)), ""; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, "a"); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "a {} {} {} {} {} {} {} {}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(src(ctx,d)), "a"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, 1); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "1 {} {} {} {} {} {} {} {}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(src(ctx,d)), "1"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, 1, 2, 3); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "1 2 3 {} {} {} {} {} {}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := v.string(src(ctx,d)), "1 2 3"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}
	}

	if d := ctx.def("foo1"); d == nil {
		ctx.err("foo1")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if s, t := d.value.String(), "{} {} {} {} {} {} {} {} {}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := d.value.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(foobar) $(foobar)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(foobar) $(foobar)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val01"); d == nil {
		ctx.err("val01")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(auto $(a))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if a := ctx.val(d, defExpand1, test_arg{"a", "x"}); a == nil {
		ctx.err("%v", tst{v})
	} else if s, t := a.String(), "x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := a.string(src(ctx,d)), "x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val02"); d == nil {
		ctx.err("val02")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(auto(a=2) $(val01),$(a))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val03"); d == nil {
		ctx.err("val03")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a=3) $(val02))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val04"); d == nil {
		ctx.err("val04")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(val03)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("bar1"); d == nil {
		ctx.err("bar1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val1" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar2"); d == nil {
		ctx.err("bar2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val2" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar3"); d == nil {
		ctx.err("bar3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val3" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar4"); d == nil {
		ctx.err("bar4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val4" {
		ctx.err("%v", v)
	}

	s := ".test.x0"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a3=3) $(a1)-$(a2)-$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "x-y-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}-{}-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "{}-{}-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.y0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a1=x a2=y a3=z) $(.test.x0)-$(a1)$(a2)$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "x-y-3-xyz"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "X"}, test_arg{"a2", "Y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.y1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x-y-3-xyz"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x-y-3-xyz"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.z0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto $(.test.x0)-$(a1)$(a2)$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "x-y-3-xy{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "x-y-3-xy"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.z1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}-{}-3-{}{}{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "{}-{}-3-{}{}{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := v.string(src(ctx,d)), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v, t := d.value, "x-y-3-xyz"; v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v, t := d.value, "a-b-3-abc"; v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	} else if s := v.string(src(ctx,d)); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	}
}
