//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
	"testing"
    "io/ioutil"
	"path/filepath"
	"fmt"
	"os"
)

const defaultCK = "tmp/go/src/extbit.io/smart/testdata/configuration"

func TestConfigureDefault(t *testing.T) {
	if s := "configure"; !testHasModule(s) {
		t.Logf("skip %s", s)
		return
	}
	{
		s := filepath.Join(filepath.Dir(_tmodules), defaultCK, configuration_sm)
		if e := os.Remove(s); e == nil { t.Errorf("%v", s) }
	}

	var ctx = load_testcase(t, "testdata/configuration", "testdefaultconfigure")
	var m = ctx.Project()
	var cc = closureWith(ctx, m.configure.scope)

	if m.configure == nil {
		ctx.err("%v: nil configure", m)
	} else if w := filepath.Join(_tmodules, "configure"); m.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", m, m.configure, m.configure.absPath, w)
	} else if o := m.configure.resolveObject(ctx, "workspace"); o == nil {
		ctx.err("workspace: %v", m.configure)
	} else if workspace, y := o.(*def); !y || workspace.value == nil {
		ctx.err("workspace: %T %v", o, o)
	} else if ws := workspace.value; ws.String() != "/Volumes/workspace" {
		ctx.err("workspace: %T %v", ws, ws)
	} else if o := m.configure.resolveObject(ctx, "workspace.out"); o == nil {
		ctx.err("workspace.out: %v", m.configure)
	} else if workspaceOut, y := o.(*def); !y || workspaceOut.value == nil {
		ctx.err("workspace.out: %T %v", o, o)
	} else if workspaceOut.value.String() != "/Volumes/workout" {
		ctx.err("workspace.out: %T %v", workspaceOut.value, workspaceOut.value)
	} else if o := m.configure.resolveObject(ctx, "rel.chop"); o == nil {
		ctx.err("rel.chop: %v", m.configure)
	} else if relchop, y := o.(*def); !y || relchop.value == nil {
		ctx.err("rel.chop: %T %v", o, o)
	} else if relchop.value.String() != fmt.Sprintf("%%%%/.smart/modules/ %s/.smart/modules/ %s/.smart/ %s/", ws, ws, ws) {
		ctx.err("rel.chop: %T %v ; %v", relchop.value, relchop.value, ws)
	} else if o := m.resolveObject(ctx, "rel.chop"); o == nil {
		ctx.err("rel.chop: %v", m.configure)
	} else if relchop2, y := o.(*def); !y || relchop2.value == nil {
		ctx.err("rel.chop: %T %v", o, o)
	} else if s := filepath.Dir(filepath.Dir(filepath.Dir(ctx.WorkDir())))+"/"; relchop2.value.String() != s {
		ctx.err("rel.chop: %T %v ; %v", relchop2.value, relchop2.value, s)
	} else if o := m.configure.resolveObject(ctx, "rel.remnant"); o == nil {
		ctx.err("rel.remnant: %v", m.configure)
	} else if relrem, y := o.(*def); !y || relrem.value == nil {
		ctx.err("rel.remnant: %T %v", o, o)
	} else if relrem.value.String() != "&(trim-prefix &(rel.chop),&/)" {
		ctx.err("rel.remnant: %T %v ; %v", relrem.value, relrem.value, ws)
	} else if o := m.configure.resolveObject(ctx, "target.out"); o == nil {
		ctx.err("target.out: %v", m.configure)
	} else if targetOut, y := o.(*def); !y || targetOut.value == nil {
		ctx.err("target.out: %T %v", o, o)
	} else if targetOut.value.String() != workspaceOut.Strval(ctx)+"/&(target.triple)/&(variant.tag)" {
		ctx.err("target.out: %T %v", targetOut.value, targetOut.value)
	} else if o := m.configure.resolveObject(ctx, "target.tmp"); o == nil {
		ctx.err("target.tmp: %v", m.configure)
	} else if targetTmp, y := o.(*def); !y || targetTmp.value == nil {
		ctx.err("target.tmp: %T %v", o, o)
	} else if targetTmp.value.String() != "&(target.out)/tmp" {
		ctx.err("target.tmp: %T %v", targetTmp.value, targetTmp.value)
	} else if o := m.configure.resolveObject(ctx, "outtmp"); o == nil {
		ctx.err("outtmp: %v", m.configure)
	} else if outtmp, y := o.(*def); !y || outtmp.value == nil {
		ctx.err("outtmp: %T %v", o, o)
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("outtmp: %T %v", outtmp.value, outtmp.value)
	} else if o := m.configure.resolveObject(ctx, "configure.cc"); o == nil {
		ctx.err("configure.cc: %v", m.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.cc: %T %v", o, o)
	} else if d.value.String() != "&(cc)" {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if t, y := d.value.(unexpanded); !y {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if t.String() != "&(cc)" {
		ctx.err("configure.cc: %v", t)
	} else if s := t.Strval(cc); s == "" {
		ctx.err("configure.cc: %v → %s", t, s)
	} else { ctx.universe().configuration.silent = true
		var config = func(name string) (entries []Entry) {
			for _, e := range ctx.universe().configuration.entries {
				if e.Name(ctx) == name { entries = append(entries, e) }
			}
			return
		}
		if FOO := config("FOO"); FOO == nil {
			ctx.err("FOO")
		}

		if s := filepath.Join(outtmp.Strval(cc), configuration_sm); s == "" {
			ctx.err("%v", outtmp)
		} else if m.configurationLoad == nil {
			ctx.err("%v: nil configuration file", m)
		} else if t := m.configurationLoad.fullname(); t != s { v := outtmp.value
			prompt(ctx, "%v:1: %v ⇒ %v\n", s, v, v.expand(ctx, strval))
			prompt(ctx, "%v:1: %v\n", t, m.configurationLoad)
			ctx.err("%v (%v)", m.configurationLoad, m)
		}

		var f = m.configuration(cc)
		if f == nil {
			ctx.err("%v: nil configuration", m)
		} else if f.fullname() != m.configurationLoad.fullname() {
			ctx.err("%v: %v %v", m, f, m.configurationLoad)
		} else if e := os.Remove(f.fullname()); false && e != nil {
			ctx.err("%v: %v", m, e)
		}

		testPromptConfiguration = false//true
		ctx.universe().configure()
		testPromptConfiguration = false

		if f = m.configurationSave; f == nil {
			ctx.err("%v: nil configuration", m)
		} else if f.fullname() != m.configurationLoad.fullname() {
			ctx.err("%v: %v", f, m.configurationLoad)
		} else if i, e := os.Stat(f.fullname()); e != nil {
			ctx.err("%s: %v", configuration_sm, e)
		} else if i == nil {
			ctx.err("missing %s", f.fullname())
		}

		if d, v := ctx.get("FOO"), ctx.get("FOO"); v == nil {
			erro(of(ctx, d), "%v", d).debug(1)
		} else if v.String() != ".self" {
			ctx.err("%T %v", v, v)
		} else if s := v.Strval(ctx); s != "testdefaultconfigure" {
			ctx.err("%T %v -> %s", v, v, s)
		} else if _, y := v.(*self); ! y {
			ctx.err("%T %v", v, v)
		}
		{
			s := filepath.Join(filepath.Dir(_tmodules), defaultCK, configuration_sm)
			if i, e := os.Stat(s); e == nil || i != nil { ctx.Errorf("%v", e) }

			s = filepath.Join(outtmp.Strval(cc), configuration_sm)
			if i, e := os.Stat(s); e != nil || i == nil { ctx.Errorf("%v", e) } else
			if b, e := ioutil.ReadFile(s); e != nil { ctx.Errorf("%v", e) } else
			if !strings.Contains(string(b), "FOO = $(.self)") { ctx.Errorf("%s", b) }

			testPromptConfiguration = false//true
			c := load_testcase(ctx.T, "testdata/configuration", "testdefaultconfigure")
			testPromptConfiguration = false

			if m := c.Project(); m.configurationLoad == nil || m.configurationSave != nil {
				c.err("%v %v", m, m.configurationSave)
			} else if m.configurationLoad.fullname() != s {
				c.err("%v", m.configurationLoad)
			} else if m.configurationLoad.stat(c) == nil {
				prompt(c, "%s:1: %v\n", m.configurationLoad.fullname(), m.configurationLoad)//.debug(1)
				c.err("%v", m.configurationLoad)
			} else if d := m.scope.FindDef("FOO"); d == nil {
				erro(c, "FOO").debug(1)
			} else if v := d.value; v == nil {
				c.err("%v ; %T", d, v)
			} else if v.String() != "$(.self)" {
				c.err("%v ; %T %v", d, v, v)
			} else if s := v.Strval(c); s != "testdefaultconfigure" {
				c.err("%v ; %T -> %s", d, v, s)
			} else if _, y := v.(*delegate); ! y {
				c.err("%v ; %T", d, v)
			}
			if d := c.def("FOO"); d == nil {
				erro(c, "FOO").debug(1)
			} else if v := c.get("FOO"); v == nil {
				c.err("%v", d)
			} else if v.String() != "$(.self)" {
				c.err("%v ; %T %v", d, v, v)
			} else if s := v.Strval(c); s != "testdefaultconfigure" {
				c.err("%v ; %T -> %s", d, v, s)
			} else if _, y := v.(*delegate); ! y {
				c.err("%v ; %T", d, v)
			}
			c.flush()
		}
		if f = m.configurationSave; f == nil {
			ctx.err("%v", m)
		} else if e := os.Remove(f.fullname()); e != nil {
			ctx.err("%v: %v", m, e)
		}
	}

	ctx.flush()
}

func TestConfigureDiverged(t *testing.T) {
	if s := "configure"; !testHasModule(s) {
		t.Logf("skip %s", s)
		return
	}
	{
		s := filepath.Join(filepath.Dir(_tmodules), defaultCK, configuration_sm)
		if e := os.Remove(s); e == nil { t.Errorf("%v", s) }
	}
	var ctx = load_testcase(t, "testdata/configuration/diverged", "testdivergedconfigure")
	var m = ctx.Project()
	var cc = closureWith(ctx, m.configure.scope)

	if m := ctx.Project(); m.configure == nil {
		ctx.err("%v: nil configure", m)
	} else if w := filepath.Join(_tmodules, "configure"); m.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", m, m.configure, m.configure.absPath, w)
	} else if o := m.configure.resolveObject(ctx, "outtmp"); o == nil {
		ctx.err("outtmp: %v", m.configure)
	} else if outtmp, y := o.(*def); !y || outtmp.value == nil {
		ctx.err("outtmp: %T %v", o, o)
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("outtmp: %T %v", outtmp.value, outtmp.value)
	} else if o := m.configure.resolveObject(ctx, "configure.cc"); o == nil {
		ctx.err("configure.cc: %v", m.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.cc: %T %v", o, o)
	} else if d.value.String() != "&(cc)" {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if t, y := d.value.(unexpanded); !y {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if t.String() != "&(cc)" {
		ctx.err("configure.cc: %v", t)
	} else if s := t.Strval(ctx); s != "" {
		ctx.err("configure.cc: %v → %s", t, s)
	} else { ctx.universe().configuration.silent = true
		var config = func(name string) (entries []Entry) {
			for _, e := range ctx.universe().configuration.entries {
				if e.Name(ctx) == name { entries = append(entries, e) }
			}
			return
		}
		if FOO := config("FOO"); FOO == nil {
			ctx.err("FOO")
		}

		if s := filepath.Join(outtmp.Strval(cc), configuration_sm); s == "" {
			ctx.err("%v", outtmp)
		} else if m.configurationLoad == nil {
			ctx.err("%v: nil configuration file", m)
		} else if t := m.configurationLoad.fullname(); t != s { v := outtmp.value
			prompt(ctx, "%v:1: %v ⇒ %v\n", s, v, v.expand(ctx, strval))
			prompt(ctx, "%v:1: %v\n", t, m.configurationLoad)
			ctx.err("%v (%v)", m.configurationLoad, m)
		} else if s := filepath.Join(outtmp.Strval(ctx), configuration_sm); s == "" {
			ctx.err("%v", outtmp)
		} else if t == s { v := outtmp.value
			prompt(ctx, "%v:1: %v ⇒ %v\n", s, v, v.expand(ctx, strval))
			prompt(ctx, "%v:1: %v\n", t, m.configurationLoad)
			ctx.err("%v (%v)", m.configurationLoad, m)
		}

		if f := m.configuration(cc); f == nil {
			ctx.err("%v: nil configuration", m)
		} else if t := m.configurationLoad.fullname(); f.fullname() == t {
			prompt(ctx, "%v:1: %v\n", f.fullname(), f)
			prompt(ctx, "%v:1: %v\n", t, m.configurationLoad)
			ctx.err("%v: %v %v", m, f, m.configurationLoad)
		} else if e := os.Remove(f.fullname()); false && e != nil {
			ctx.err("%v: %v", m, e)
		}

		testPromptConfiguration = false//true
		testConfigurationDiverged = true
		ctx.universe().configure()
		testConfigurationDiverged = false
		testPromptConfiguration = false

		if f := m.configurationSave; f == nil {
			ctx.err("%v: nil configuration", m)
		} else if t := m.configurationLoad.fullname(); f.fullname() == t {
			prompt(ctx, "%v:1: %v\n", f.fullname(), f)
			prompt(ctx, "%v:1: %v\n", t, m.configurationLoad)
			ctx.err("%v: %v", f, m.configurationLoad)
		} else if i, e := os.Stat(f.fullname()); e != nil {
			ctx.err("%v: %s: %v", f, configuration_sm, e)
		} else if i == nil {
			ctx.err("missing %s", f.fullname())
		}

		if d, v := ctx.get("FOO"), ctx.get("FOO"); v == nil {
			erro(of(ctx, d), "%v", d).debug(1)
		} else if v.String() != ".self" {
			ctx.err("%T %v", v, v)
		} else if s := v.Strval(ctx); s != "testdivergedconfigure" {
			ctx.err("%T %v ⇒ %s", v, v, s)
		} else if _, y := v.(*self); !y {
			ctx.err("%T %v", v, v)
		}

		if outtmp = ctx.def("outtmp"); outtmp == nil {
			ctx.err("outtmp")
		} else if outtmp.value == nil {
			ctx.err("%v", outtmp)
		} else if outtmp.value.String() != filepath.Join(ctx.WorkDir(), "tmp") {
			ctx.err("%v", outtmp)
		}
		{
			s := filepath.Join(filepath.Dir(_tmodules), defaultCK, configuration_sm)
			if i, e := os.Stat(s); e == nil || i != nil { ctx.Errorf("%v", e) }

			s = filepath.Join(outtmp.Strval(ctx), configuration_sm)
			if s != filepath.Join(outtmp.Strval(cc), configuration_sm) { ctx.Errorf("%v", s) } else
			if i, e := os.Stat(s); e != nil || i == nil { ctx.Errorf("%v", e) } else
			if b, e := ioutil.ReadFile(s); e != nil { ctx.Errorf("%v", e) } else
			if !strings.Contains(string(b), "FOO = $(.self)") { ctx.Errorf("%s", b) }

			testPromptConfiguration = false//true
			c := load_testcase(ctx.T, "testdata/configuration", "testdefaultconfigure")
			testPromptConfiguration = false

			if d := c.Project().scope.FindDef("FOO"); d == nil {
				erro(of(c, d), "%v", d).debug(1)
			} else if v := d.value; v != nil {
				c.err("%v ; %T %v", d, v, v)
			}
			if d := c.def("FOO"); d.value != nil {
				erro(of(c, d), "%v", d).debug(1)
			}
			c.flush()
		}
		if f := m.configurationSave; f == nil {
			ctx.err("%v", m)
		} else if e := os.Remove(f.fullname()); e != nil {
			ctx.err("%v: %v", m, e)
		}
		if e := os.RemoveAll(outtmp.Strval(ctx)); e != nil {
			ctx.err("%v: %v", outtmp, e)
		}
	}

	ctx.flush()
}

func TestConfigureCustom(t *testing.T) {
	{
		s := filepath.Join(filepath.Dir(_tmodules), defaultCK, "custom", configuration_sm)
		if e := os.Remove(s); e == nil { t.Errorf("%v", s) }
	}
	var ctx = load_testcase(t, "testdata/configuration/custom", "testcustomconfigure")
	{
		s := filepath.Join(ctx.WorkDir(), "tmp", configuration_sm)
		if e := os.Remove(s); e == nil { t.Errorf("%v", s) }
	}

	if m := ctx.Project(); m.configure == nil {
		ctx.err("%v: nil configure", m)
	} else if w := ctx.WorkDir(); m.configure.absPath != w+PathSep {
		ctx.err("%v.%v: %s != %s", m, m.configure, m.configure.absPath, w)
	} else if o := m.configure.resolveObject(ctx, "foo"); o == nil {
		ctx.err("configure.foo: %v %v", m.configure, m.configure.absPath)
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
	} else if t.String() != ".self" {
		ctx.err("configure.foo: %v", t)
	} else if s := t.Strval(ctx); s != "configure" {
		ctx.err("configure.foo: %v → %s", t, s)
	} else if s := d.value.Strval(ctx); s != m.configure.name {
		ctx.err("configure.foo: %T %v → %v (%v)", d.value, d.value, s, m.configure.name)
	} else if s := d.Strval(ctx); s != m.configure.name {
		ctx.err("configure.foo: %v → %v", d, s)
	} else {
		ctx.universe().configuration.silent = true

		var config = func(name string) (entries []Entry) {
			for _, e := range ctx.universe().configuration.entries {
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
		} else if e := os.Remove(f.fullname()); false && e != nil {
			ctx.err("%v: %v", m, e)
		}

		ctx.universe().configure()
		{
			s := filepath.Join(filepath.Dir(_tmodules), defaultCK, "custom", configuration_sm)
			if i, e := os.Stat(s); e != nil || i == nil { ctx.Errorf("%v", e) }
			if b, e := ioutil.ReadFile(s); e != nil { ctx.Errorf("%v", e) } else
			if !strings.Contains(string(b), "FOO1 = yes{}") { ctx.Errorf("%s", b) } else
			if !strings.Contains(string(b), "FOO2 = yes{}") { ctx.Errorf("%s", b) } else
			if !strings.Contains(string(b), "FOO3 = true{}") { ctx.Errorf("%s", b) } else
			if !strings.Contains(string(b), "FOO4 = true{}") { ctx.Errorf("%s", b) } else
			if !strings.Contains(string(b), "FOO5 = true{}") { ctx.Errorf("%s", b) }

			c := load_testcase(ctx.T, "testdata/configuration/custom", "testcustomconfigure")
			if d, foo := c.def("FOO1"), c.get("FOO1"); foo == nil {
				erro(of(c, d), "%v", d).debug(1)
			} else if foo.String() != "yes{}" {
				c.err("%T %v ; %v", foo, foo, d)
			} else if s := foo.Strval(ctx); s != "yes" {
				c.err("%T %v -> %s", foo, foo, s)
			} else if _, y := foo.(*answer); ! y {
				c.err("%T %v ; %v", foo, foo, d)
			}
			c.flush()
		}

		if f == nil {
			ctx.err("%v: nil configuration", m)
		} else if i, e := os.Stat(f.fullname()); e != nil {
			ctx.err("%s: %v", configuration_sm, e)
		} else if i == nil {
			ctx.err("missing %s", configuration_sm)
		} else if e := os.Remove(f.fullname()); e != nil {
			ctx.err("%v: %v", m, e)
		}

		var (
			// foo  = ctx.get("foo")
			foo1 = ctx.get("FOO1")
			foo2 = ctx.get("FOO2")
			foo3 = ctx.get("FOO3")
			foo4 = ctx.get("FOO4")
			foo5 = ctx.get("FOO5")
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
	}

	ctx.flush()
}
