//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValueCache0(ctx *testcase) {
	v := _null(_position(ctx))
	m := make(map[any]string)
	m["foo"] = "foobar"
	m['f'] = "rune(f)"
	m["f"] = "string(f)"
	m[char('f')] = "char(f)"
	m[1] = "one"
	m[v] = "value"

	if x, y := m["foo"]; !y || x != "foobar" {
		ctx.err("%v", m)
	}

	s := "foobar"[:3]
	if x, y := m[s]; !y || x != "foobar" {
		ctx.err("%v", m)
	}

	if x, y := m['f']; !y || x != "rune(f)" {
		ctx.err("%v ; %v", m, x)
	}
	if x, y := m[char('f')]; !y || x != "char(f)" {
		ctx.err("%v ; %v", m, x)
	}
	if x, y := m["f"]; !y || x != "string(f)" {
		ctx.err("%v ; %v", m, x)
	}

	if x, y := m[1]; !y || x != "one" {
		ctx.err("%v", m)
	}

	var t Value = v
	if x, y := m[v]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if x, y := m[t]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if _, y := m[_null(_position(ctx))]; y {
		ctx.err("%v", m)
	}
}

func testValueCache1(ctx *testcase) {
	testValueCache0(ctx)

	var p = _project(ctx)

	if c := &p.filemap; c.a != nil {
		ctx.err("%v", c)
	} else if len(c.words) != 1 {
		ctx.err("%v ; %v", c.words, c)
	} else if x, y := c.words["foo"]; !y {
		ctx.err("%v ; %v", c.words, c)
	} else if len(x.puncs) != 1 {
		ctx.err("%v ; %v", x.puncs, x)
	} else if len(x.words) != 1 {
		ctx.err("%v ; %v", x.words, x)
	} else {
		if z, y := x.puncs[DOT]; !y {
			ctx.err("%v", x)
		} else if len(z.words) != 2 {
			ctx.err("%v", z)
		} else {
			if t, y := z.words["c"]; !y {
				ctx.err("%v", z)
			} else if len(t.a) != 1 || t.words != nil {
				ctx.err("%v", t)
			} else if slot, y := t.a[0].(filemap); !y {
				ctx.err("%v", tst{t.a[0]})
			} else if slot.String() != "foo.c" {
				ctx.err("%v", tst{t.a[0]})
			}

			if t, y := z.words["c++"]; !y {
				ctx.err("%v", z)
			} else if len(t.a) != 1 || t.words != nil {
				ctx.err("%v", t)
			} else if slot, y := t.a[0].(filemap); !y {
				ctx.err("%v", tst{t.a[0]})
			} else if slot.String() != "foo.c++" {
				ctx.err("%v", tst{t.a[0]})
			}
		}

		if z0, y := x.words["bar"]; !y {
			ctx.err("%v", x)
		} else if len(z0.puncs) != 1 {
			ctx.err("%v", z0)
		} else if z, y := z0.puncs[DOT]; !y {
			ctx.err("%v", z0)
		} else if len(z.words) != 2 {
			ctx.err("%v", z.words)
		} else if t, y := z.words["c"]; !y {
			ctx.err("%v", z)
		} else if len(t.a) != 1 || t.words != nil {
			ctx.err("%v", t)
		} else if slot, y := t.a[0].(filemap); !y {
			ctx.err("%v", tst{t.a[0]})
		} else if slot.String() != "foo/bar.c" {
			ctx.err("%v", tst{t.a[0]})
		} else if t, y := z.words["c++"]; !y {
			ctx.err("%v", z)
		} else if len(t.a) != 1 || t.words != nil {
			ctx.err("%v", t)
		} else if slot, y := t.a[0].(filemap); !y {
			ctx.err("%v", tst{t.a[0]})
		} else if slot.String() != "foo/bar.c++" {
			ctx.err("%v", tst{t.a[0]})
		}
	}

	if v, pat := ctx.val("p1"), "*.c"; v == nil {
		ctx.err("%v", p)
	} else if s := __string(ctx,v); s != pat {
		ctx.err("%v : %s != %s", tst{v}, s, pat)
	} else if m := unmap_files(ctx, p, v, nil); m == nil {
		ctx.err("unmap_files: %v", tst{v})
	} else if len(m) != 1 {
		ctx.err("%v : %v", v, m)
	} else if m[0].name != "foo.c" {
		ctx.err("%v : %v", v, m[0])
	} else if s := ts(m[0].pattern); s != "{=compound {=word foo} {=punct .} {=word c}}" {
		ctx.err("%v : %v : %v", v, m[0].pattern, s)
	}

	if v, pat := ctx.val("p2"), "**.c"; v == nil {
		ctx.err("%v", p)
	} else if s := __string(ctx,v); s != pat {
		ctx.err("%v : %s != %s", tst{v}, s, pat)
	} else if m := unmap_files(ctx, p, v, nil); m == nil {
		ctx.err("unmap_files: %v", tst{v})
	} else if len(m) != 2 {
		ctx.err("%v : %v", v, m)
	} else if i := 0; m[i].name != "foo.c" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=compound {=word foo} {=punct .} {=word c}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	} else if i := 1; m[i].name != "foo/bar.c" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=path {=word foo} {=compound {=word bar} {=punct .} {=word c}}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	}

	if v, pat := ctx.val("p3"), "**.c++"; v == nil {
		ctx.err("%v", p)
	} else if s := __string(ctx,v); s != pat {
		ctx.err("%v : %s != %s", tst{v}, s, pat)
	} else if m := unmap_files(ctx, p, v, nil); m == nil {
		ctx.err("unmap_files: %v", tst{v})
	} else if len(m) != 2 {
		ctx.err("%v : %v", v, m)
	} else if i := 0; m[i].name != "foo.c++" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=compound {=word foo} {=punct .} {=word c++}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	} else if i := 1; m[i].name != "foo/bar.c++" {
		ctx.err("%v : %v", v, m[i])
	} else if s := ts(m[i].pattern); s != "{=path {=word foo} {=compound {=word bar} {=punct .} {=word c++}}}" {
		ctx.err("%v : %v : %v", v, m[i].pattern, s)
	}
}
