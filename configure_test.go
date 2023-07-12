//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"testing"
	"os"
)

func TestConfigure(t *testing.T) {
	var ctx = load_testcase(t, "testdata/configuration", "testconfigure")
	var get = func(s string, ii ...interface{}) Value { return ctx.get(s, ii...) }

	if m := ctx.Project(); m.configure == nil {
		ctx.err("%v: nil configure", m)
	} else if o := m.configure.resolveObject(ctx, "foo"); o == nil {
		ctx.err("configure.foo: %v", m.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.foo: %T %v", o, o)
	} else if d.value.String() != ".self" {
		ctx.err("configure.foo: %T %v", d.value, d.value)
	} else if t, y := d.value.(*self); !y || t == nil {
		ctx.err("configure.foo: %T %v", d.value, d.value)
	} else if t.String() != ".self" {
		ctx.err("configure.foo: %v", t)
	} else if s := t.Strval(ctx); s != "configure" {
		ctx.err("configure.foo: %v → %s", t, s)
	} else if s := d.value.Strval(ctx); s != m.configure.name {
		ctx.err("configure.foo: %T %v → %v (%v)", d.value, d.value, s, m.configure.name)
	} else if s := d.Strval(ctx); s != m.configure.name {
		ctx.err("configure.foo: %v → %v", d, s)
	} else {
		configuration.silent = true

		var config = func(name string) (entries []Entry) {
			for _, e := range configuration.entries {
				if e.Name(ctx) == name { entries = append(entries, e) }
			}
			return
		}
		if FOO1 := config("FOO1"); FOO1 == nil {
			ctx.err("FOO1")
		}
		if FOO2 := config("FOO2"); FOO2 == nil {
			ctx.err("FOO2")
		}
		if FOO3 := config("FOO3"); FOO3 == nil {
			ctx.err("FOO3")
		}
		if FOO4 := config("FOO4"); FOO4 == nil {
			ctx.err("FOO4")
		}
		if FOO5 := config("FOO5"); FOO5 == nil {
			ctx.err("FOO5")
		}

		var f = m.configuration(ctx)
		if f == nil {
			ctx.err("%v: nil configuration", m)
		} else {
			os.Remove(f.fullname())
		}

		ctx.universe().configure()

		var (
			// foo  = get("foo")
			foo1 = get("FOO1")
			foo2 = get("FOO2")
			foo3 = get("FOO3")
			foo4 = get("FOO4")
			foo5 = get("FOO5")
		)
		// if foo == nil {
		// 	ctx.err("foo")
		// } else if  foo.String() != ".self" {
		// 	ctx.err("%T %v", foo, foo)
		// } else if s := foo.Strval(ctx); s != "configure" {
		// 	ctx.err("%T %v -> %s", foo, foo, s)
		// }
		if s := foo1.Strval(ctx); s != "yes" {
			ctx.err("%T %v -> %s", foo1, foo1, s)
		} else if s = foo1.String(); s != "yes{}" {
			ctx.err("%T %v -> %s", foo1, foo1, s)
		}
		if s := foo2.Strval(ctx); s != "yes" {
			ctx.err("%T %v -> %s", foo2, foo2, s)
		} else if s = foo2.String(); s != "yes{}" {
			ctx.err("%T %v -> %s", foo2, foo2, s)
		}
		if s := foo3.Strval(ctx); s != "true" {
			ctx.err("%T %v -> %s", foo3, foo3, s)
		} else if s = foo3.String(); s != "true{}" {
			ctx.err("%T %v -> %s", foo3, foo3, s)
		}
		if s := foo4.Strval(ctx); s != "true" {
			ctx.err("%T %v -> %s", foo4, foo4, s)
		} else if s = foo4.String(); s != "true{}" {
			ctx.err("%T %v -> %s", foo4, foo4, s)
		}
		if s := foo5.Strval(ctx); s != "true" {
			ctx.err("%T %v -> %s", foo5, foo5, s)
		} else if s = foo5.String(); s != "true{}" {
			ctx.err("%T %v -> %s", foo5, foo5, s)
		}

		if f == nil {
			ctx.err("%v: nil configuration", m)
		} else if fi, e := os.Stat(f.fullname()); e != nil {
			ctx.err("%s: %v", configuration_sm, e)
		} else if fi == nil {
			ctx.err("missing %s", configuration_sm)
		} else {
			os.Remove(f.fullname())
		}
	}

	ctx.flush()
}
