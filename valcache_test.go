//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
)

func testValueCache0(ctx *testcase) {
	v := makeNull(ctx.Position())
	m := make(map[interface{}]string)
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

	var tv Value = v
	if x, y := m[v]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if x, y := m[tv]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if _, y := m[makeNull(ctx.Position())]; y {
		ctx.err("%v", m)
	}
}

func testValueCache1(ctx *testcase) { testValueCache0(ctx)
	var u = _universe(ctx)

	if u == nil {
		ctx.err("nil universe")
	}

	if c := &u.filemaps; c.a != nil {
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
			} else if slot, y := t.a[0].(filemap_slot); !y {
				ctx.err("%v", ust{t.a[0]})
			} else if slot.String() != "foo.c" {
				ctx.err("%v", ust{t.a[0]})
			}

			if t, y := z.words["c++"]; !y {
				ctx.err("%v", z)
			} else if len(t.a) != 1 || t.words != nil {
				ctx.err("%v", t)
			} else if slot, y := t.a[0].(filemap_slot); !y {
				ctx.err("%v", ust{t.a[0]})
			} else if slot.String() != "foo.c++" {
				ctx.err("%v", ust{t.a[0]})
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
		} else if slot, y := t.a[0].(filemap_slot); !y {
			ctx.err("%v", ust{t.a[0]})
		} else if slot.String() != "foo/bar.c" {
			ctx.err("%v", ust{t.a[0]})
		} else if t, y := z.words["c++"]; !y {
			ctx.err("%v", z)
		} else if len(t.a) != 1 || t.words != nil {
			ctx.err("%v", t)
		} else if slot, y := t.a[0].(filemap_slot); !y {
			ctx.err("%v", ust{t.a[0]})
		} else if slot.String() != "foo/bar.c++" {
			ctx.err("%v", ust{t.a[0]})
		}
	}

	if s, pat := "p1", "*.c"; true {
		note(ctx, "TODO: unmapfiles %v", pat).debug(1)
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v : %v", ctx.project(), s)
	} else if s = v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", ust{v}, s, pat)
	} else {
		if m := unmapfiles(ctx, v); m == nil {
			ctx.err("unmapfiles %v", ust{v})
		} else if len(m) != 1 {
			ctx.err("unmapfiles %v : %v", v, m)
		}
	}

	if s, pat := "p2", "**.c"; true {
		note(ctx, "TODO: unmapfiles %v", pat).debug(1)
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v : %v", ctx.project(), s)
	} else if s = v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", ust{v}, s, pat)
	} else {
		if m := unmapfiles(ctx, v); m == nil {
			ctx.err("unmapfiles %v", ust{v})
		} else if len(m) != 2 {
			ctx.err("unmapfiles %v : %v", v, m)
		}
	}

	if s, pat := "p3", "**.c++"; true {
		note(ctx, "TODO: unmapfiles %v", pat).debug(1)
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v : %v", ctx.project(), s)
	} else if s = v.string(ctx); s != pat {
		ctx.err("%v : %s != %s", ust{v}, s, pat)
	} else {
		if m := unmapfiles(ctx, v); m == nil {
			ctx.err("unmapfiles %v", ust{v})
		} else if len(m) != 2 {
			ctx.err("unmapfiles %v : %v", v, m)
		}
	}
}

func testValueCache2(ctx *testcase) {
	var u = _universe(ctx)
	if u == nil {
		ctx.err("nil universe")
	} else if c := &u.filemaps; c.a != nil {
		ctx.err("%v", c)
	} else if len(c.globs) != 3 {
		ctx.err("%v", c.globs)
	} else {
		if x, y := c.globs["*.c++"]; !y {
			ctx.err("%v", c.globs)
		} else if x.words != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap_slot); !y {
			ctx.err("%v", ust{x.a[0]})
		} else if slot.String() != "*.c++" {
			ctx.err("%v", ust{x.a[0]})
		}

		if x, y := c.globs["**.c"]; !y {
			ctx.err("%v", c.globs)
		} else if x.words != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap_slot); !y {
			ctx.err("%v", ust{x.a[0]})
		} else if slot.String() != "**.c" {
			ctx.err("%v", ust{x.a[0]})
		}

		if x, y := c.globs["???"]; !y {
			ctx.err("%v", c.globs)
		} else if x.words != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap_slot); !y {
			ctx.err("%v", ust{x.a[0]})
		} else if slot.String() != "{=glob ???}" {
			ctx.err("%v", ust{x.a[0]})
		} else if slot.string(ctx) != "???" {
			ctx.err("%v", x.a[0])
		}

		if x, y := c.words["foo"]; !y {
			ctx.err("%v", c.words)
		} else if x.words != nil {
			ctx.err("%v", x.words)
		} else if len(x.globs) != 6 {
			ctx.err("%v", x.globs)
		} else {
			if c, y := x.globs["*.c++"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", ust{c.a[0]})
			} else if slot.String() != "foo/*.c++" {
				ctx.err("%v", ust{c.a[0]})
			}

			if c, y := x.globs["*.xx"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", ust{c.a[0]})
			} else if slot.String() != "foo/*.xx" {
				ctx.err("%v", ust{c.a[0]})
			}

			if c, y := x.globs["*.yy"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", ust{c.a[0]})
			} else if slot.String() != "foo/*.yy" {
				ctx.err("%v", ust{c.a[0]})
			}

			if c, y := x.globs["*zzz"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", ust{c.a[0]})
			} else if slot.String() != "foo/*zzz" {
				ctx.err("%v", ust{c.a[0]})
			}

			if c, y := x.globs["**z"]; !y {
				ctx.err("%v", x.globs)
			} else if c.words != nil || c.globs != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap_slot); !y {
				ctx.err("%v", ust{c.a[0]})
			} else if slot.String() != "foo/**z" {
				ctx.err("%v", ust{c.a[0]})
			}

			if c, y := x.globs["??"]; !y {
				ctx.err("%v", x.globs)
			} else if c.a != nil {
				ctx.err("%v", c.a)
			} else if len(c.globs) != 1 {
				ctx.err("%v", c.globs)
			} else {
				if x, y := c.globs["???.c++"]; !y {
					ctx.err("%v", x.globs)
				} else if x.words != nil {
					ctx.err("%v", x)
				} else if len(x.a) != 1 {
					ctx.err("%v", c.a)
				} else if slot, y := x.a[0].(filemap_slot); !y {
					ctx.err("%v", ust{x.a[0]})
				} else if slot.String() != "foo/{=glob ??}/{=glob ???.c++}" {
					ctx.err("%v", ust{x.a[0]})
				} else if slot.string(ctx) != "foo/??/???.c++" {
					ctx.err("%v", ust{x.a[0]})
				}
			}
		}
	}

	var _fm *_filemap

	if s := "foo"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x == nil {
		ctx.err("%s %v", s, &u.filemaps)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s %v ; %v", s, m, &u.filemaps)
	}

	if s := "a.c++"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x == nil {
		ctx.err("%s %v", s, &u.filemaps)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m)
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m)
	} else if m[0].String() != "*.c++" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else if m[0].pattern.string(ctx) != "*.c++" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else {
		_fm = m[0]._filemap

		if len(_fm.paths) != 1 {
			ctx.err("%s : %v", s, _fm.paths)
		} else if _fm.paths[0].String() != "src" {
			ctx.err("%s : %v", s, _fm.paths[0])
		}

		if len(_fm.patts) != 9 {
			ctx.err("%s : %v", s, _fm.patts)
		} else if _fm.patts[0].String() != "*.c++" {
			ctx.err("%s : %v", s, _fm.patts[0])
		} else if _fm.patts[1].String() != "**.c" {
			ctx.err("%s : %v", s, _fm.patts[1])
		} else if _fm.patts[2].String() != "{=glob ???}" {
			ctx.err("%s : %v", s, _fm.patts[2])
		} else if _fm.patts[3].String() != "foo/*.c++" {
			ctx.err("%s : %v", s, _fm.patts[3])
		} else if _fm.patts[4].String() != "foo/*.xx" {
			ctx.err("%s : %v", s, _fm.patts[4])
		} else if _fm.patts[5].String() != "foo/*.yy" {
			ctx.err("%s : %v", s, _fm.patts[5])
		} else if _fm.patts[6].String() != "foo/{=glob ??}/{=glob ???.c++}" {
			ctx.err("%s : %v", s, _fm.patts[6])
		} else if _fm.patts[7].String() != "foo/*zzz" {
			ctx.err("%s : %v", s, _fm.patts[7])
		} else if _fm.patts[8].String() != "foo/**z" {
			ctx.err("%s : %v", s, _fm.patts[8])
		}
	}

	if s := "aaa.c++"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x == nil {
		ctx.err("%s %v", s, &u.filemaps)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "*.c++" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else if m[0].pattern.string(ctx) != "*.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}

	if s := "a/aa.c++"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x != nil {
		ctx.err("%s %v", s, &u.filemaps)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s", s)
	}

	if s := "foo/aaa.c++"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x != nil {
		ctx.err("%s %v", s, &u.filemaps)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "foo/*.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "foo/*.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}
	if s := "foo/a/xyz.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/a/bb.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bb.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bb/cc.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bbb/ccc.c++"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bbb.c++/ccc"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x != nil {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s %v", s, m)
	}
	if s := "foo/ab/xyz.c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "foo/{=glob ??}/{=glob ???.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "foo/??/???.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}
	if s := "foo/12/xyz.c++"; false {
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "foo/{=glob ??}/{=glob ???.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "foo/??/???.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}

	if s := "c"; false {
	} else if m := unmapfiles(ctx, s); m != nil {
		ctx.err("%s", s)
	}

	if s := "abc"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x == nil {
		ctx.err("%s %v", s, &u.filemaps)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{=glob ???}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "???" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v ; %v %v", s, m[0], m[0]._filemap, _fm)
	}
	if s := "c++"; false {
	} else if x, y := u.filemaps.hit(unmap{ctx}, s); x == nil {
		ctx.err("%s %v", s, &u.filemaps)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &u.filemaps)
	} else if m := unmapfiles(ctx, s); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].name != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{=glob ???}" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern.string(ctx) != "???" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v ; %v %v", s, m[0], m[0]._filemap, _fm)
	}

	if s := "foo/xxxzzz"; true {
		// NOTE: this test finds if the "foo/*zzz" is applied prior to "foo/**z"
		// NOTE: test for 50 times until one err, because the order of map-keys is random.
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := u.filemaps.hit(unmap{ctx}, s); x != nil {
				ctx.err("%s %v", s, &u.filemaps) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &u.filemaps) ; break
			} else if m := unmapfiles(ctx, s); m == nil {
				ctx.err("%s", s) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m)
			} else if m[0].name != s {
				ctx.err("%s : %v", s, m[0])
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0])
			} else if m[0].String() != "foo/*zzz" {
				ctx.err("%s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0].pattern.string(ctx) != "foo/*zzz" {
				ctx.err("%s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0])
			}
		}
	}
	if s := "foo/xx/yyz"; true {
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := u.filemaps.hit(unmap{ctx}, s); x != nil {
				ctx.err("%s %v", s, &u.filemaps) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &u.filemaps) ; break
			} else if m := unmapfiles(ctx, s); m == nil {
				ctx.err("%s", s) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m)
			} else if m[0].name != s {
				ctx.err("%s : %v", s, m[0])
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0])
			} else if m[0].String() != "foo/**z" {
				ctx.err("%s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0].pattern.string(ctx) != "foo/**z" {
				ctx.err("%s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0])
			}
		}
	}
	if s := "foo/xx/yy/zzzz"; true {
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := u.filemaps.hit(unmap{ctx}, s); x != nil {
				ctx.err("%s %v", s, &u.filemaps) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &u.filemaps) ; break
			} else if m := unmapfiles(ctx, s); m == nil {
				ctx.err("%s", s) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m)
			} else if m[0].name != s {
				ctx.err("%s : %v", s, m[0])
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0])
			} else if m[0].String() != "foo/**z" {
				ctx.err("%s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0].pattern.string(ctx) != "foo/**z" {
				ctx.err("%s : %v (%d)", s, m[0].pattern, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0])
			}
		}
	}
}

func testValueCache3(ctx *testcase) {
	var u = _universe(ctx)
	if u == nil {
		ctx.err("nil universe")
	} else if c := &u.filemaps; c.a != nil {
		ctx.err("universe valcache : %v", c)
	} else if len(c.words) != 1 {
		ctx.err("universe valcache : %v", c)
	}
}

func testValueCache(ctx *testcase) {
	var p = ctx.project()
	var u = _universe(ctx)

	if u == nil {
		ctx.err("nil universe")
	}

	if p == nil {
		ctx.err("nil project")
	} else if p.filemap._fix == nil {
		ctx.err("wrong filemap._fix")
	} else if c1, y := p.filemap._fix["*"]; !y {
		ctx.err("wrong filemap._fix: %v", p.filemap._fix)
	} else if c2, y := c1._fix[".log"]; !y {
		ctx.err("wrong filemap._fix: %v %v", c1._fix, c2)
	}

	if p.filemap.fast == nil {
		ctx.err("wrong filemap.fast: %v", p.filemap)
	} else if c1, y := p.filemap.fast[".deps"]; !y {
		ctx.err("wrong filemap.fast: %v, %v", p.filemap.fast, c1)
	}

	for i, s := range []string{
		".deps/xx/yy/zzzzzzzzz",
		"foo.log",
		"foo.o",
		"foo.c",
		"foo.c++",
	} {
		if !strings.Contains(s, pathSep) {
			if 2 < i { /* not matching patterns */ } else
			if c := p.filemap.matchPatts(ctx, s); c == nil {
				ctx.err("miss cache for %d. %s", i, s)
			}
			if c := p.filemap.str(ctx, "foo.log", cacheMatchPatts); c == nil {
				ctx.err("miss cache for %d. %s", i, s)
			}
		}
		if c := p.filemap.strx(ctx, s, cacheMatchPatts); c == nil {
			ctx.err("miss cache for %d. %s", i, s)
		}
	}

	if n := len(p.filemapx); n != 1 {
		ctx.err("wrong closure cache: %d", n)
	} else if v := p.filemapx[0]; v._key.String() != "&(gen)" {
		ctx.err("wrong closure cache: %v", us(v._key))
	} else if a, y := v._val.(filemap); !y {
		ctx.err("wrong closure cache: %v (%v)", us(v._val), a)
	}

	if v := ctx.val("val1"); v == nil {
		ctx.err("val1")
	} else if v.string(ctx) != "**.c++" {
		ctx.err("%v", v)
	} else if true {
		// skips, files() not working globs
	} else if t := files(ctx, v); len(t) != 2 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo/bar.c++" && t[0].name != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	} else if t[1].name != "foo/bar.c++" && t[1].name != "foo.c++" {
		ctx.err("%v %v", v, t[1])
	}

	if v := ctx.val("val2"); v == nil {
		ctx.err("val2")
	} else if v.string(ctx) != "foo.c++" {
		ctx.err("%v", v)
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%v %v", v, f)
	} else if f.ident(ctx) != "foo.c++" {
		ctx.err("%v %v", v, f)
	} else if t := files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.c++" {
		ctx.err("%v %v", v, t[0])
	}

	if v := ctx.val("val3"); v == nil {
		ctx.err("val3")
	} else if v.string(ctx) != "foo.o" {
		ctx.err("%T %v", v, v)
	} else if t := unmapfiles(ctx, v); t == nil {
		ctx.err("%T %v", v, v)
	} else if len(t) != 1 {
		ctx.err("%T %v ; %v", v, v, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if t[0].String() != "**.o" {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if _, y := t[0].pattern.(*globpat); !y {
		ctx.err("%T %v ; %v", v, v, t[0])
	} else if t[0].project != p {
		ctx.err("%T %v ; %v %v %v", v, v, t[0], t[0].paths, t[0].project)
	} else if t[0].paths == nil {
		ctx.err("%T %v ; %v %v", v, v, t[0], t[0].paths)
	} else if t[0].paths[0].String() != "$//.tmp" {
		ctx.err("%T %v ; %v %v", v, v, t[0], t[0].paths)
	} else if f := p.file(ctx, v); f == nil {
		ctx.err("%T %v ; %v", v, v, t)
	} else if f.ident(ctx) != "foo.o" {
		ctx.err("%v %v", v, f)
	} else if t := files(ctx, v); len(t) != 1 {
		ctx.err("%v %v", v, t)
	} else if t[0].name != "foo.o" {
		ctx.err("%v %v", v, t[0])
	}

	if d := ctx.def("sources"); d == nil {
		ctx.err("sources is nil")
	} else if v := d.invoke(ctx, nil, nil); v == nil {
		ctx.err("sources is wrong: %v %v", d, v)
	} else if s := v.string(ctx); strings.Count(s, "foo.c") != 2 {
		ctx.err("sources is wrong: %v", v) // NOTE: "foo.c" counts foo.c foo.c++
	} else if strings.Count(s, "foo.c++") != 1 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.c") != 2 {
		ctx.err("sources is wrong: %v", v)
	} else if strings.Count(s, "foo/bar.c++") != 1 {
		ctx.err("sources is wrong: %v", v)
	}

	if d := ctx.def("objects"); d == nil {
		ctx.err("objects is nil")
	} else if v := d.invoke(ctx, nil, nil); v == nil {
		ctx.err("objects is wrong: %v %v", d, v)
	}
}
