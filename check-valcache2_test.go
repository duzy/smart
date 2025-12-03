//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValueCache2(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil universe")
	} else if c := &p.filemap; c.a != nil {
		ctx.err("%v", c)
	} else if len(c.v) != 3 {
		ctx.err("%v", c.v)
	} else {
		if x, y := c.v["*.c++"]; !y {
			ctx.err("%v", c.v)
		} else if x.v != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if slot.String() != "*.c++" {
			ctx.err("%v", tst{x.a[0]})
		}

		if x, y := c.v["**.c"]; !y {
			ctx.err("%v", c.v)
		} else if x.v != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if slot.String() != "**.c" {
			ctx.err("%v", tst{x.a[0]})
		}

		if x, y := c.v["???"]; !y {
			ctx.err("%v", c.v)
		} else if x.v != nil {
			ctx.err("%v", x)
		} else if len(x.a) != 1 {
			ctx.err("%v", x.a)
		} else if slot, y := x.a[0].(filemap); !y {
			ctx.err("%v", tst{x.a[0]})
		} else if slot.String() != "{=glob ???}" {
			ctx.err("%v", tst{x.a[0]})
		}

		if x, y := c.v["foo"]; !y {
			ctx.err("%v", c.v)
		} else if x.v != nil {
			ctx.err("%v", x.v)
		} else if len(x.v) != 6 {
			ctx.err("%v", x.v)
		} else {
			if c, y := x.v["*.c++"]; !y {
				ctx.err("%v", x.v)
			} else if c.v != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*.c++" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.v["*.xx"]; !y {
				ctx.err("%v", x.v)
			} else if c.v != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*.xx" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.v["*.yy"]; !y {
				ctx.err("%v", x.v)
			} else if c.v != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*.yy" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.v["*zzz"]; !y {
				ctx.err("%v", x.v)
			} else if c.v != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/*zzz" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.v["**z"]; !y {
				ctx.err("%v", x.v)
			} else if c.v != nil {
				ctx.err("%v", c)
			} else if len(c.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := c.a[0].(filemap); !y {
				ctx.err("%v", tst{c.a[0]})
			} else if slot.String() != "foo/**z" {
				ctx.err("%v", tst{c.a[0]})
			}

			if c, y := x.v["??"]; !y {
				ctx.err("%v", x.v)
			} else if c.a != nil {
				ctx.err("%v", c.a)
			} else if len(c.v) != 1 {
				ctx.err("%v", c.v)
			} else if x, y := c.v["???.c++"]; !y {
					ctx.err("%v", x.v)
			} else if x.v != nil {
				ctx.err("%v", x)
			} else if len(x.a) != 1 {
				ctx.err("%v", c.a)
			} else if slot, y := x.a[0].(filemap); !y {
				ctx.err("%v", tst{x.a[0]})
			} else if slot.String() != "foo/{=glob ??}/{=glob ???.c++}" {
				ctx.err("%v", tst{x.a[0]})
			}
		}
	}

	var _fm *_filemap

	if s := "foo"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s %v ; %v", s, m, &p.filemap)
	}

	if s := "a.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m == nil {
		ctx.err("%s ; %v", s, x)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].string != s {
		ctx.err("%s : %v", s, m)
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m)
	} else if m[0].String() != "{filemap=*.c++ name=a.c++}" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else if __string(ctx,m[0].pattern) != "*.c++" {
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
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m == nil {
		ctx.err("%s ; %v", s, x)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].string != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=*.c++ name=aaa.c++}" {
		ctx.err("%s : %v", s, m[0].pattern)
	} else if __string(ctx,m[0].pattern) != "*.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}

	if s := "a/aa.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s", s)
	}

	if s := "foo/aaa.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].string != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=foo/*.c++ name=foo/aaa.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if __string(ctx,m[0].pattern) != "foo/*.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}
	if s := "foo/a/xyz.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/a/bb.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bb.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bb/cc.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bbb/ccc.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s", s)
	}
	if s := "foo/aa/bbb.c++/ccc"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s %v", s, m)
	}
	if s := "foo/ab/xyz.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].string != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=foo/{=glob ??}/{=glob ???.c++} name=foo/ab/xyz.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if __string(ctx,m[0].pattern) != "foo/??/???.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}
	if s := "foo/12/xyz.c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].string != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap=foo/{=glob ??}/{=glob ???.c++} name=foo/12/xyz.c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if __string(ctx,m[0].pattern) != "foo/??/???.c++" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v", s, m[0])
	}

	if s := "c"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m != nil {
		ctx.err("%s", s)
	}

	if s := "abc"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].string != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap={=glob ???} name=abc}" {
		ctx.err("%s : %v", s, m[0])
	} else if __string(ctx,m[0].pattern) != "???" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v ; %v %v", s, m[0], m[0]._filemap, _fm)
	}
	if s := "c++"; false {
	} else if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x == nil {
		ctx.err("%s %v", s, &p.filemap)
	} else if !y {
		ctx.err("%s %v ; %v", s, x, &p.filemap)
	} else if m := unmap_files(ctx, p, s, nil); m == nil {
		ctx.err("%s", s)
	} else if len(m) != 1 {
		ctx.err("%s : %v", s, m)
	} else if m[0].string != s {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].pattern == nil {
		ctx.err("%s : %v", s, m[0])
	} else if m[0].String() != "{filemap={=glob ???} name=c++}" {
		ctx.err("%s : %v", s, m[0])
	} else if __string(ctx,m[0].pattern) != "???" {
		ctx.err("%s : %v", s, m[0])
	} else if m[0]._filemap != _fm {
		ctx.err("%s : %v ; %v %v", s, m[0], m[0]._filemap, _fm)
	}

	if s, pat := "foo/xxxzzz", "foo/*zzz"; true {
		// NOTE: this test finds if the "foo/*zzz" is applied prior to "foo/**z"
		// NOTE: test for 50 times until one err, because the order of map-keys is random.
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
				ctx.err("%s %v", s, &p.filemap) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &p.filemap) ; break
			} else if m := unmap_files(ctx, p, s, nil); m == nil {
				ctx.err("%s ; %v", s, ts(ctx)) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m) ; break
			} else if m[0].string != s {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].String() != "{filemap="+pat+" name=foo/xxxzzz}" {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if __string(ctx,m[0].pattern) != pat {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0]) ; break
			}
		}
	}
	if s, pat := "foo/xx/yyz", "foo/**z"; true {
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
				ctx.err("%s %v", s, &p.filemap) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &p.filemap) ; break
			} else if m := unmap_files(ctx, p, s, nil); m == nil {
				ctx.err("%s ; %v", s, ts(ctx)) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m) ; break
			} else if m[0].string != s {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].String() != "{filemap="+pat+" name=foo/xx/yyz}" {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if __string(ctx,m[0].pattern) != pat {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0]) ; break
			}
		}
	}
	if s, pat := "foo/xx/yy/zzzz", "foo/**z"; true {
		for i := 0 ; i < 100 ; i += 1 {
			if x, y := hit(&uncache{ctx,nil}, &p.filemap, s); x != nil {
				ctx.err("%s %v", s, &p.filemap) ; break
			} else if y {
				ctx.err("%s %v ; %v", s, x, &p.filemap) ; break
			} else if m := unmap_files(ctx, p, s, nil); m == nil {
				ctx.err("%s ; %v", s, ts(ctx)) ; break
			} else if len(m) != 1 {
				ctx.err("%s : %v", s, m) ; break
			} else if m[0].string != s {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].pattern == nil {
				ctx.err("%s : %v", s, m[0]) ; break
			} else if m[0].String() != "{filemap="+pat+" name=foo/xx/yy/zzzz}" {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if __string(ctx,m[0].pattern) != pat {
				ctx.err("%s : %v : %v : %v (%d)", s, m[0], m[0].pattern, pat, i) ; break
			} else if m[0]._filemap != _fm {
				ctx.err("%s : %v", s, m[0]) ; break
			}
		}
	}
}
