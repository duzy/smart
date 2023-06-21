//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "path/filepath"
	"testing"
	"os"
)

func TestConfigure(t *testing.T) {
	var ctx = confine("testdata/configuration")
	var wd = ctx.WorkDir()

	if err := ctx.loadTopWork(); err != nil {
		t.Errorf("%v", err)
	} else if n := ctx.countErrors(); n > 0 {
		t.Errorf("errors %v, base=%s", n, wd)
		ctx.flushDiags()
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
	} else if m.name != "testconfigure" {
		t.Errorf("wrong main: %v", m)
	} else if m.configure == nil {
		t.Errorf("nil configure: %v", m)
	} else if o := m.configure.resolveObject(ctx, "foo"); o == nil {
		t.Errorf("configure.foo: %v", m.configure)
	} else if o.Strval(ctx) != m.configure.name {
		t.Errorf("configure.foo: %v", o)
	} else {
		configuration.silent = true

		var config = func(name string) (entries []Entry) {
			for _, e := range configuration.entries {
				if e.Name(ctx) == name { entries = append(entries, e) }
			}
			return
		}
		if FOO1 := config("FOO1"); FOO1 == nil {
			t.Errorf("FOO1")
		}
		if FOO2 := config("FOO2"); FOO2 == nil {
			t.Errorf("FOO2")
		}
		if FOO3 := config("FOO3"); FOO3 == nil {
			t.Errorf("FOO3")
		}
		if FOO4 := config("FOO4"); FOO4 == nil {
			t.Errorf("FOO4")
		}
		if FOO5 := config("FOO5"); FOO5 == nil {
			t.Errorf("FOO5")
		}

		var theConfigurationSM = filepath.Join(wd, configuration_sm)
		os.Remove(theConfigurationSM)
		ctx.configure()

		if fi, e := os.Stat(theConfigurationSM); e != nil {
			t.Errorf("%s: %v", configuration_sm, e)
		} else if fi == nil {
			t.Errorf("missing %s", configuration_sm)
		} else {
			os.Remove(theConfigurationSM)
		}

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
			foo1 = get("FOO1")
			foo2 = get("FOO2")
			foo3 = get("FOO3")
			foo4 = get("FOO4")
			foo5 = get("FOO5")
		)
		if s := foo1.Strval(ctx); s != "yes" {
			t.Errorf("%T %v -> %s", foo1, foo1, s)
		} else if s = foo1.String(); s != "yes{}" {
			t.Errorf("%T %v -> %s", foo1, foo1, s)
		}
		if s := foo2.Strval(ctx); s != "yes" {
			t.Errorf("%T %v -> %s", foo2, foo2, s)
		} else if s = foo2.String(); s != "yes{}" {
			t.Errorf("%T %v -> %s", foo2, foo2, s)
		}
		if s := foo3.Strval(ctx); s != "true" {
			t.Errorf("%T %v -> %s", foo3, foo3, s)
		} else if s = foo3.String(); s != "true{}" {
			t.Errorf("%T %v -> %s", foo3, foo3, s)
		}
		if s := foo4.Strval(ctx); s != "true" {
			t.Errorf("%T %v -> %s", foo4, foo4, s)
		} else if s = foo4.String(); s != "true{}" {
			t.Errorf("%T %v -> %s", foo4, foo4, s)
		}
		if s := foo5.Strval(ctx); s != "true" {
			t.Errorf("%T %v -> %s", foo5, foo5, s)
		} else if s = foo5.String(); s != "true{}" {
			t.Errorf("%T %v -> %s", foo5, foo5, s)
		}
	}

	if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}
