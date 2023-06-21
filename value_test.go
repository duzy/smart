//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	// "strings"
	"testing"
)

func TestValues1(t *testing.T) {
	var ctx = confine("testdata/value/1")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testvalues" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name, ii...); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		var (
			test_foo = get(".test.foo")
		)
		if s := test_foo.Strval(ctx); s != "-foo" {
			t.Errorf("%T %v, %s", test_foo, test_foo, s)
		} else if s = test_foo.String(); s != "-foo" {
			t.Errorf("%T %v, %s", test_foo, test_foo, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}

func TestValues2(t *testing.T) {
	var ctx = confine("testdata/value/2")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testvalues" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name, ii...); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		if v := get(".test.ab"); v == nil {
			t.Errorf(".test.ab")
		} else if s := v.String(); s != "foobar $1-$2" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar -" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.ba"); v == nil {
			t.Errorf(".test.ba")
		} else if s := v.String(); s != "foobaz $2-$1" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobaz -" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.ab", "a", "b"); v == nil {
			t.Errorf(".test.ab")
		} else if s := v.String(); s != "foobar a-b" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar a-b" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.ba", "a", "b"); v == nil {
			t.Errorf(".test.ba")
		} else if s := v.String(); s != "foobaz b-a" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobaz b-a" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test", "a", "b", "c"); v == nil {
			t.Errorf(".test")
		} else if s := v.String(); s != "$(value(-c) &(.test.x)) $(&(.test.x) aa,bb) foobar aa-bb" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar aa-bb foobar aa-bb" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.1", "a", "b", "c"); v == nil {
			t.Errorf(".test.1")
		} else if s := v.String(); s != "foobaz $2-$1 $(.test.ba $1$1,$2$2) $3" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobaz -" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.2", "a", "b", "c"); v == nil {
			t.Errorf(".test.2")
		} else if s := v.String(); s != "foobar $1-$2 foobar aa-bb c" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar - foobar aa-bb c" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.3", "a", "b", "c"); v == nil {
			t.Errorf(".test.3")
		} else if s := v.String(); s != "foobar $1-$2 $(.test.ab $1$1,$2$2) $(call(-c) .test.ab,$1$1,$2$2) $3" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar -" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.4", "a", "b", "c"); v == nil {
			t.Errorf(".test.4")
		} else if s := v.String(); s != "foobar $1-$2 foobar aa-bb foobar aa-bb c" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar - foobar aa-bb foobar aa-bb c" {
			t.Errorf("%T %v -> %s", v, v, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}

func TestValues3(t *testing.T) {
	var ctx = confine("testdata/value/3")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testvalues" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name, ii...); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		if v := get(".test.x"); v == nil {
			t.Errorf(".test.x")
		} else if s := v.String(); s != "$(a1)-$(a2)-3" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "--3" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test", "a", "b", "c"); v == nil {
			t.Errorf(".test")
		} else if s := v.String(); s != "x-y-3-xyz" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x-y-3-xyz" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.1", "a", "b", "c"); v == nil {
			t.Errorf(".test.1")
		} else if s := v.String(); s != "x-y-3-xyz" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x-y-3-xyz" {
			t.Errorf("%T %v, %s", v, v, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}

func TestValues4(t *testing.T) {
	var ctx = confine("testdata/value/4")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testvalues" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name, ii...); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		if v := get(".test.D.c"); v == nil {
			t.Errorf(".test.D.c")
		} else if s := v.String(); s != "D c $(value &(.test.x)) xx $(.test.foreach $1,&(.test.none)) ($1) $(foreach $1,&(.test.x.$_)) ($1)" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "D c xx xx () ()" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.D.c++"); v == nil {
			t.Errorf(".test.D.c++")
		} else if s := v.String(); s != "D c++" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "D c++" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.I.c"); v == nil {
			t.Errorf(".test.I.c")
		} else if s := v.String(); s != "I c xx xx xx xx" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "I c xx xx xx xx" {
			t.Errorf("%T %v, %s", v, v, s)
		}

		if v := get(".test.I.c++"); v == nil {
			t.Errorf(".test.I.c++")
		} else if s := v.String(); s != "I c++" {
			t.Errorf("%T %v, %s", v, v, s)
		} else if s := v.Strval(ctx); s != "I c++" {
			t.Errorf("%T %v, %s", v, v, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}

func TestGlobMatch(t *testing.T) {
	if a, b, c := globMatch("*.c", "foo.c"); !a || c != nil {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if b[0] != "foo" {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch("**.c", "foo.c"); !a || c != nil {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	} else if b[0] != "foo" {
		t.Errorf("glob(*.c, foo.c): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("*.c", "foo/bar.c"); a == true || c != nil {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if len(b) != 0 {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch("**.c", "foo/bar.c"); !a || c != nil {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**.c, foo/bar.c): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("*", "foobar"); !a || c != nil {
		t.Errorf("glob(*, foobar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(*, foobar): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch("**", "foobar"); !a || c != nil {
		t.Errorf("glob(**, foobar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**, foobar): %v %v %v", a, b, c)
	} else if b[0] != "foobar" {
		t.Errorf("glob(**, foobar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("*", "foobar/"); a == true || c != nil {
		t.Errorf("glob(*, foobar/): %v %v %v", a, b, c)
	} else if len(b) != 0 {
		t.Errorf("glob(*, foobar/): %v %v %v", a, b, c)
	}
	if a, b, c := globMatch("**", "foobar/"); !a || c != nil {
		t.Errorf("glob(**, foobar/): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**, foobar/): %v %v %v", a, b, c)
	} else if b[0] != "foobar/" {
		t.Errorf("glob(**, foobar/): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("**", "foo/bar/"); !a || c != nil {
		t.Errorf("glob(**, foo/bar/): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(**, foo/bar/): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar/" {
		t.Errorf("glob(**, foo/bar/): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("**xx**", "foo/bar/xx/foo/bar"); !a || c != nil {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 2 {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar/" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "/foo/bar" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("**/xx/**", "foo/bar/xx/foo/bar"); !a || c != nil {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 2 {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "foo/bar" {
		t.Errorf("glob(**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("**/??/**", "foo/bar/xx/foo/bar"); !a || c != nil {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 4 {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "x" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[2] != "x" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	} else if b[3] != "foo/bar" {
		t.Errorf("glob(**/??/**, foo/bar/xx/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("**/[xyz]/**", "foo/bar/z/foo/bar"); !a || c != nil {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if len(b) != 3 {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[0] != "foo/bar" {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[1] != "z" {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	} else if b[2] != "foo/bar" {
		t.Errorf("glob(**/[xyz]/**, foo/bar/z/foo/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("foo/???/bar", "foo/xyz/bar"); !a || c != nil {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if len(b) != 3 {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[0] != "x" {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[1] != "y" {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	} else if b[2] != "z" {
		t.Errorf("glob(foo/???/bar, foo/xyz/bar): %v %v %v", a, b, c)
	}

	if a, b, c := globMatch("foo/[xyz]/bar", "foo/z/bar"); !a || c != nil {
		t.Errorf("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	} else if len(b) != 1 {
		t.Errorf("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	} else if b[0] != "z" {
		t.Errorf("glob(foo/[xyz]/bar, foo/z/bar): %v %v %v", a, b, c)
	}
}

func TestPatterns(t *testing.T) {
	var ctx = confine("testdata/value")

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testvalues" {
		t.Errorf("wrong main: %v", m)
	} else {
		var ctx Context = &closureContext{ctx, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
		var get = func(name string, ii ...interface{}) (res Value) {
			var d *def
			if d, res = call(ctx, name, ii...); d == nil {
				t.Errorf("%s is not def", name)
			} else if res == nil {
				t.Errorf("%s is nil", name)
				res = MakeNone(d.position)
			}
			return
		}

		// Globs

		var (
			glob1 = get("glob1")
			glob2 = get("glob2")
			val1 = get("val1")
			val2 = get("val2")
			val3 = get("val3")
		)

		if glob1.Strval(ctx) != "*.c" {
			t.Errorf("glob1 is wrong: %T %v", glob1, glob1)
		}

		if glob2.Strval(ctx) != "**.c" {
			t.Errorf("glob2 is wrong: %T %v", glob2, glob2)
		}

		if val1.Strval(ctx) != "foo.c" {
			t.Errorf("val1 is wrong: %T %v", val1, val1)
		}

		if val2.Strval(ctx) != "foo/bar.c" {
			t.Errorf("val2 is wrong: %T %v", val2, val2)
		}

		if val3.Strval(ctx) != "foo/bar.c" {
			if false { t.Errorf("val3 is wrong: %T %v", val3, val3) }
		}

		if a, b, c := glob1.match(ctx, val1); !a {
			t.Errorf("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}
		if a, b, c := glob2.match(ctx, val1); !a {
			t.Errorf("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}

		if a, b, c := glob1.match(ctx, val2); a == true {
			if false { t.Errorf("match(%v, %v): %v %v %v", glob1, val1, a, b, c) }
		}
		if a, b, c := glob2.match(ctx, val2); !a {
			t.Errorf("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}

		if a, b, c := glob1.match(ctx, val3); a == true {
			if false { t.Errorf("match(%v, %v): %v %v %v", glob1, val1, a, b, c) }
		}
		if a, b, c := glob2.match(ctx, val3); !a {
			t.Errorf("match(%v, %v): %v %v %v", glob1, val1, a, b, c)
		}

		// TODO: test glob.stencil(...)

		// Regexps

		var (
			regexp1 = get("regexp1")
			regexp2 = get("regexp2")
			regexp3 = get("regexp3")
			regexp4 = get("regexp4")
			regexp5 = get("regexp5")
			regexp6 = get("regexp6")
		)

		if regexp1.Strval(ctx) != `x{1}, x{1,}, x{1,2}, x{5}?, x{2,}?, x{2,8}? \p{Greek}, \P{Greek}` {
			t.Errorf("regexp1 is wrong: %T %v", regexp1, regexp1)
		}

		if regexp2.Strval(ctx) != `(re) (?P<name>re) (?:re) (?im) (?sU:re) \x{10ffff} \x1f \123 \* \. \? \$` {
			t.Errorf("regexp2 is wrong: %T %v", regexp2, regexp2)
		}

		if regexp3.Strval(ctx) != `[[:xdigit:]]*, [[:^alpha:]], [^xyz] [a-z] \A \B \b \Q**??^:[]{}\E \^ \z` {
			t.Errorf("regexp3 is wrong: %T %v", regexp3, regexp3)
		}

		if regexp4.Strval(ctx) != `fo{2}\.c` {
			t.Errorf("regexp4 is wrong: %T %v", regexp4, regexp4)
		} else if a, b, c := regexp4.match(ctx, val1); !a {
			t.Errorf("match(%v, %v): %v %v %v", regexp4, val1, a, b, c)
		} else if s, y := b.(string); !y {
			t.Errorf("match(%v, %v): %v %v %v", regexp4, val1, a, b, c)
		} else if s != val1.Strval(ctx) {
			t.Errorf("match(%v, %v): %v %v %v", regexp4, val1, a, b, c)
		} else if len(c) != 0 {
			t.Errorf("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		}

		if regexp5.Strval(ctx) != `fo{2}/bar\.c` {
			t.Errorf("regexp5 is wrong: %T %v", regexp5, regexp5)
		} else if a, b, c := regexp5.match(ctx, val2); !a {
			t.Errorf("match(%v, %v): %v %v %v", regexp5, val2, a, b, c)
		} else if s, y := b.(string); !y {
			t.Errorf("match(%v, %v): %v %v %v", regexp5, val2, a, b, c)
		} else if s != val2.Strval(ctx) {
			t.Errorf("match(%v, %v): %v %v %v", regexp5, val2, a, b, c)
		} else if len(c) != 0 {
			t.Errorf("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		}

		if regexp6.Strval(ctx) != `fo{2}(/o{2}){3}/bar\.c` {
			t.Errorf("regexp6 is wrong: %T %v", regexp6, regexp6)
		} else if a, b, c := regexp6.match(ctx, val3); !a {
			t.Errorf("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if s, y := b.(string); !y {
			t.Errorf("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if s != val3.Strval(ctx) {
			t.Errorf("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if len(c) != 1 {
			t.Errorf("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		} else if c[0] != "/oo" {
			t.Errorf("match(%v, %v): %v %v %v", regexp6, val3, a, b, c)
		}

		// TODO: test regexp.stencil(...)

		if v := get(".test.ab"); v == nil {
			t.Errorf(".test.ab")
		} else if s := v.String(); s != "foobar $1-$2" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar -" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.ba"); v == nil {
			t.Errorf(".test.ba")
		} else if s := v.String(); s != "foobaz $2-$1" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobaz -" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.ab", "a", "b"); v == nil {
			t.Errorf(".test.ab")
		} else if s := v.String(); s != "foobar a-b" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar a-b" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.ba", "a", "b"); v == nil {
			t.Errorf(".test.ba")
		} else if s := v.String(); s != "foobaz b-a" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobaz b-a" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.1", "a", "b", "c"); v == nil {
			t.Errorf(".test.1")
		} else if s := v.String(); s != "foobaz $2-$1 $(.test.ba $1$1,$2$2)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobaz -" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.2", "a", "b", "c"); v == nil {
			t.Errorf(".test.2")
		} else if s := v.String(); s != "foobar $1-$2 $(.test.ab $1$1,$2$2) $(call(-c) .test.ab,$1$1,$2$2)" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobar -" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.3", "a", "b", "c"); v == nil {
			t.Errorf(".test.3")
		} else if s := v.String(); s != "foobaz $2-$1" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "foobaz -" {
			t.Errorf("%T %v -> %s", v, v, s)
		}

		if v := get(".test.4", "a", "b", "c"); v == nil {
			t.Errorf(".test.4")
		} else if s := v.String(); s != "x-y-3-xyz" {
			t.Errorf("%T %v -> %s", v, v, s)
		} else if s := v.Strval(ctx); s != "x-y-3-xyz" {
			t.Errorf("%T %v -> %s", v, v, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}
