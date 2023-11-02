//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"strings"
    "io/ioutil"
	"path/filepath"
	"fmt"
	"os"
)

const defaultCK = "tmp/go/src/extbit.io/smart/testdata/configuration"

func testConfigItem(ctx *testcase, name string) (entries []Entry, d *def) {
	for _, e := range ctx.universe().configuration.entries {
		if e.name(ctx) == name { entries = append(entries, e) }
	}
	if o := ctx.Project().resolve(ctx, name); o != nil { d, _ = o.(*def) }
	return
}

func testConfigure1(ctx *testcase) {
	{
		s := filepath.Join(filepath.Dir(_tmodules), defaultCK, configuration_sm)
		if e := os.Remove(s); e == nil { ctx.Errorf("%v", s) }
	}

	var m = ctx.Project()
	var cc = closureWith(ctx, m.configure.scope)
	var outtmp *def

	if m.configure == nil {
		ctx.err("%v: nil configure", m)
	} else if w := filepath.Join(_tmodules, "configure"); m.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", m, m.configure, m.configure.absPath, w)
	} else if len(m.configure.bases) != 1 {
		ctx.err("configure: %v", m.configure.bases)
	} else if m.configure.bases[0].name != "configure.base" {
		ctx.err("configure: %v", m.configure.bases[0])
	} else if o := m.configure.resolve(ctx, "workspace"); o == nil {
		ctx.err("workspace: %v", m.configure)
	} else if workspace, y := o.(*def); !y || workspace.value == nil {
		ctx.err("workspace: %T %v", o, o)
	} else if proj := workspace.OwnerProject(); proj == nil {
		ctx.err("workspace: %T %v", o, o)
	} else if proj.name != "general" {
		ctx.err("workspace: %T %v ; %v", o, o, proj)
	} else if ws := workspace.value.String(); !strings.HasPrefix(proj.absPath, ws) {
		ctx.err("workspace: %T %v", ws, ws)
	} else if o := m.configure.resolve(ctx, "workspace.out"); o == nil {
		ctx.err("workspace.out: %v", m.configure)
	} else if workspaceOut, y := o.(*def); !y || workspaceOut.value == nil {
		ctx.err("workspace.out: %T %v", o, o)
	} else if workspaceOut.value.String() != filepath.Join(filepath.Dir(ws), "workout") {
		ctx.err("workspace.out: %T %v", workspaceOut.value, workspaceOut.value)
	} else if o := m.configure.resolve(ctx, "workspace.ext"); o == nil {
		ctx.err("workspace.out: %v", m.configure)
	} else if workspaceExt, y := o.(*def); !y || workspaceExt.value == nil {
		ctx.err("workspace.out: %T %v", o, o)
	} else if workspaceExt.value.String() != filepath.Join(ws, "external") {
		ctx.err("workspace.out: %T %v", workspaceExt.value, workspaceExt.value)
	} else if o := m.configure.resolve(ctx, "rel.chop"); o == nil {
		ctx.err("rel.chop: %v", m.configure)
	} else if relchop, y := o.(*def); !y || relchop.value == nil {
		ctx.err("rel.chop: %T %v", o, o)
	} else if relchop.value.String() != fmt.Sprintf("%%%%/.smart/modules/ %s/.smart/modules/ %s/.smart/ %s/", ws, ws, ws) {
		ctx.err("rel.chop: %T %v ; %v", relchop.value, relchop.value, ws)
	} else if o := m.resolve(ctx, "rel.chop"); o == nil {
		ctx.err("rel.chop: %v", m.configure)
	} else if relchop2, y := o.(*def); !y || relchop2.value == nil {
		ctx.err("rel.chop: %T %v", o, o)
	} else if s := filepath.Dir(filepath.Dir(filepath.Dir(ctx.WorkDir())))+"/"; relchop2.value.String() != s {
		ctx.err("rel.chop: %T %v ; %v", relchop2.value, relchop2.value, s)
	} else if o := m.configure.resolve(ctx, "rel.remnant"); o == nil {
		ctx.err("rel.remnant: %v", m.configure)
	} else if relrem, y := o.(*def); !y || relrem.value == nil {
		ctx.err("rel.remnant: %T %v", o, o)
	} else if relrem.value.String() != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("rel.remnant: %T %v ; %v", relrem.value, relrem.value, ws)
	} else if o := m.configure.resolve(ctx, "target.out"); o == nil {
		ctx.err("target.out: %v", m.configure)
	} else if targetOut, y := o.(*def); !y || targetOut.value == nil {
		ctx.err("target.out: %T %v", o, o)
	} else if targetOut.value.String() != workspaceOut.string(ctx)+"/&(target.triple)/&(variant.tag)" {
		ctx.err("target.out: %T %v", targetOut.value, targetOut.value)
	} else if o := m.configure.resolve(ctx, "target.tmp"); o == nil {
		ctx.err("target.tmp: %v", m.configure)
	} else if targetTmp, y := o.(*def); !y || targetTmp.value == nil {
		ctx.err("target.tmp: %T %v", o, o)
	} else if targetTmp.value.String() != "&(target.out)/tmp" {
		ctx.err("target.tmp: %T %v", targetTmp.value, targetTmp.value)
	} else if o := m.configure.resolve(ctx, "outtmp"); o == nil {
		ctx.err("outtmp: %v", m.configure)
	} else if outtmp, y = o.(*def); !y || outtmp.value == nil {
		ctx.err("outtmp: %T %v", o, o)
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("outtmp: %T %v", outtmp.value, outtmp.value)
	} else if o := m.configure.resolve(ctx, "configure.cc"); o == nil {
		ctx.err("configure.cc: %v", m.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.cc: %T %v", o, o)
	} else if d.value.String() != "&(cc)" {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if u, y := d.value.(unexpanded); !y {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if u.String() != "&(cc)" {
		ctx.err("configure.cc: %v", u)
	} else if s := u.string(cc); s == "" {
		ctx.err("configure.cc: %v → %s", u, s)
	}

	if e, d := testConfigItem(ctx, "FOO"); len(e) != 1 {
		ctx.err("FOO")
	} else if d == nil {
		ctx.err("FOO")
	} else if d.value != nil {
		ctx.err("%v", d)
	}

	if s := filepath.Join(outtmp.string(cc), configuration_sm); s == "" {
		ctx.err("%v", outtmp)
	} else if m.configurationFile == nil {
		ctx.err("%v: nil configuration file", m)
	} else if t := m.configurationFile.fullname(); t != s {
		ctx.err("%v %v", m.configurationFile, outtmp.value)
	}

	var f = m.configuration(cc)
	if f == nil {
		ctx.err("%v: nil configuration", m)
	} else if f.fullname() != m.configurationFile.fullname() {
		ctx.err("%v: %v %v", m, f, m.configurationFile)
	} else if e := os.Remove(f.fullname()); false && e != nil {
		ctx.err("%v: %v", m, e)
	}

	testCheckExecRecipe = func(_ctx Context, source string, recipe Value) {
		testValidateExecRecipe(ctx, _ctx, source, recipe)
	}
	testPromptConfiguration = false//true
	ctx.universe().configuration.silent = true
	ctx.universe().configure(ctx)
	testPromptConfiguration = false
	testCheckExecRecipe = nil

	if d, v := ctx.get("FOO"), ctx.get("FOO"); v == nil {
		ctx.err("%v %v", d, v)
	} else if v.String() != ".self" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "testdefaultconfigure" {
		ctx.err("%T %v -> %s", v, v, s)
	} else if _, y := v.(*self); !y {
		ctx.err("%T %v", v, v)
	}

	s := filepath.Join(filepath.Dir(_tmodules), defaultCK, configuration_sm)
	if i, e := os.Stat(s); e == nil || i != nil { ctx.Errorf("%v", e) }

	s = filepath.Join(outtmp.string(cc), configuration_sm)
	if i, e := os.Stat(s); e != nil || i == nil { ctx.Errorf("%v", e) } else
	if b, e := ioutil.ReadFile(s); e != nil { ctx.Errorf("%v", e) } else
	if !strings.Contains(string(b), "FOO = $(.self)") { ctx.Errorf("%s", b) }

	testPromptConfiguration = false//true

	runcase(ctx.T, "testdata/configuration", "testdefaultconfigure", func (c *testcase) {
		if m := c.Project(); m.configurationFile == nil /* || m.configurationSave != nil */ {
			c.err("%v", m/* , m.configurationSave */)
		} else if m.configurationFile.fullname() != s {
			c.err("%v", m.configurationFile)
		} else if m.configurationFile.stat(c) == nil {
			prompt(c, "%s:1: %v\n", m.configurationFile.fullname(), m.configurationFile)//.debug(1)
			c.err("%v", m.configurationFile)
		} else if d := m.scope.FindDef("FOO"); d == nil {
			erro(c, "FOO").debug(1)
		} else if v := d.value; v == nil {
			c.err("%v ; %T", d, v)
		} else if v.String() != "$(.self)" {
			c.err("%v ; %T %v", d, v, v)
		} else if s := v.string(c); s != "testdefaultconfigure" {
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
		} else if s := v.string(c); s != "testdefaultconfigure" {
			c.err("%v ; %T -> %s", d, v, s)
		} else if _, y := v.(*delegate); ! y {
			c.err("%v ; %T", d, v)
		}
	})

	testPromptConfiguration = false

	if f = m.configurationFile; f == nil {
		ctx.err("%v", m)
	} else if e := os.Remove(f.fullname()); e != nil {
		if false { ctx.err("%v: %v", m, e) }
	}
}

func testConfigure2(ctx *testcase) {
	defer assured(ctx, true)

	var m = ctx.Project()
	var cc = closureWith(ctx, m.configure.scope/*, m.scope */)
	var outtmp *def

	if m.configure == nil {
		ctx.err("%v: nil configure", m)
	} else if w := filepath.Join(_tmodules, "configure"); m.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", m, m.configure, m.configure.absPath, w)
	} else if o := m.configure.resolve(ctx, "configure.cc"); o == nil {
		ctx.err("configure.cc: %v", m.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.cc: %T %v", o, o)
	} else if d.value.String() != "&(cc)" {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if o := m.configure.resolve(ctx, "outtmp"); o == nil {
		ctx.err("outtmp: %v", m.configure)
	} else if outtmp, y = o.(*def); !y || outtmp.value == nil {
		ctx.err("outtmp: %T %v", o, o)
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("outtmp: %T %v", outtmp.value, outtmp.value)
	} else if u, y := d.value.(unexpanded); !y {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if u.String() != "&(cc)" {
		ctx.err("configure.cc: %v", u)
	} else if s := u.string(ctx); false && s != "" {
		ctx.err("configure.cc: %v → %s", u, s)
	}

	if e, d := testConfigItem(ctx, "FOO"); len(e) != 1 {
		ctx.err("FOO")
	} else if d == nil {
		ctx.err("FOO")
	} else if d.value != nil {
		ctx.err("%v", d)
	}

	if s := filepath.Join(outtmp.string(cc), configuration_sm); s == "" {
		ctx.err("%v", outtmp)
	} else if m.configurationFile == nil {
		ctx.err("%v: nil configuration file", m)
	} else if t := m.configurationFile.fullname(); t != s {
		ctx.err("%v (%v)", m.configurationFile, m)
	} else if s := filepath.Join(outtmp.string(ctx), configuration_sm); s == "" {
		ctx.err("%v", outtmp)
	} else if false && t == s {
		ctx.err("%v: %v != %v", m.configurationFile, t, s)
	}

	if f := m.configuration(cc); f == nil {
		ctx.err("%v: nil configuration", m)
	} else if m.configurationFile == nil {
		ctx.err("%v", m.configurationFile)
	// } else if t := m.configurationFile.fullname(); f.fullname() == t {
	// 	ctx.err("%v", f)
	} else if e := os.Remove(f.fullname()); false && e != nil {
		ctx.err("%v", e)
	}

	testCheckExecRecipe = func(_ctx Context, source string, recipe Value) {
		testValidateExecRecipe(ctx, _ctx, source, recipe)
	}
	testPromptConfiguration = false//true
	testConfigurationDiverged = true
	ctx.universe().configuration.silent = true
	ctx.universe().configure(cc)
	testConfigurationDiverged = false
	testPromptConfiguration = false
	testCheckExecRecipe = nil

	if f := m.configurationFile; f == nil {
		ctx.err("%v: nil configuration", m)
	} else if i, e := os.Stat(f.fullname()); e != nil {
		ctx.err("%v: %s: %v", f, configuration_sm, e)
	} else if i == nil {
		ctx.err("missing %s", f.fullname())
	}

	if d, v := ctx.get("FOO"), ctx.get("FOO"); v == nil {
		erro(of(ctx, d), "%v", d).debug(1)
	} else if v.String() != ".self" {
		ctx.err("%T %v", v, v)
	} else if s := v.string(ctx); s != "testdivergedconfigure" {
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
		f := m.configurationFile
		s := filepath.Join(filepath.Dir(_tmodules), defaultCK, configuration_sm)
		if i, e := os.Stat(s); e == nil || i != nil { ctx.Errorf("%v", e) }
		if f == nil {
			ctx.err("%v: nil configuration file", m)
		} else if f.fullname() == s { // NOTE: diverged
			erro(ctx, "%s", f.fullname())
			erro(ctx, "%s", s)//.debug(1)
			ctx.err("%v ; %v", outtmp, f)
		}

		/**/s = filepath.Join(outtmp.string(ctx), configuration_sm)
		if s != filepath.Join(outtmp.string(cc ), configuration_sm) {
			erro(ctx, "%s", outtmp.string(cc))
			erro(ctx, "%s", s).debug(1)
			ctx.Errorf("%v ; %v", outtmp, configuration_sm)
		} else if f == nil {
			ctx.err("%v: nil configuration file", m)
		} else if f.fullname() == s { // NOTE diverged
			erro(ctx, "%s", f.fullname())
			erro(ctx, "%s", s)//.debug(1)
			ctx.err("%v ; %v", outtmp, f)
		} else if i, e := os.Stat(/*s*/f.fullname()); e != nil || i == nil {
			erro(ctx, "%s", f.fullname())
			erro(ctx, "%s", s)//.debug(1)
			ctx.err("%v ; %v", outtmp, f)
		} else if b, e := ioutil.ReadFile(/*s*/f.fullname()); e != nil {
			ctx.Errorf("%v", e)
		} else if !strings.Contains(string(b), "FOO = $(.self)") {
			ctx.Errorf("%s", b)
		}

		testPromptConfiguration = false//true

		runcase(ctx.T, "testdata/configuration", "testdefaultconfigure", func (c *testcase) {
			if d := c.Project().scope.FindDef("FOO"); d == nil {
				erro(of(c, d), "%v", d).debug(1)
			} else if v := d.value; v != nil {
				c.err("%v ; %T %v", d, v, v)
			}
			if d := c.def("FOO"); d.value != nil {
				erro(of(c, d), "%v", d).debug(1)
			}
		})

		testPromptConfiguration = false
	}
	if f := m.configurationFile; f == nil {
		ctx.err("%v", m)
	} else if e := os.Remove(f.fullname()); e != nil {
		ctx.err("%v: %v", m, e)
	}
	if e := os.RemoveAll(outtmp.string(ctx)); e != nil {
		ctx.err("%v: %v", outtmp, e)
	}
}

func testConfigure3(ctx *testcase, spec, name string) {
	{
		s := filepath.Join(filepath.Dir(_tmodules), defaultCK, "custom", configuration_sm)
		if e := os.Remove(s); e == nil { ctx.Errorf("%v", s) }
	}
	{
		s := filepath.Join(ctx.WorkDir(), "tmp", configuration_sm)
		if e := os.Remove(s); e == nil { ctx.Errorf("%v", s) }
	}

	var m = ctx.Project()
	if m.configure == nil {
		ctx.err("%v: nil configure", m)
	} else if w := ctx.WorkDir(); m.configure.absPath != w+PathSep {
		ctx.err("%v.%v: %s != %s", m, m.configure, m.configure.absPath, w)
	} else if o := m.configure.resolve(ctx, "foo"); o == nil {
		ctx.err("configure.foo: %v %v", m.configure, m.configure.absPath)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.foo: %T %v", o, o)
	} else if d.value.String() != ".self" {
		ctx.err("configure.foo: %T %v", d.value, d.value)
	} else if self, y := d.value.(*self); !y || self == nil {
		ctx.err("configure.foo: %T %v", d.value, d.value)
	} else if self.String() != ".self" {
		ctx.err("configure.foo: %v", self)
	} else if s := self.string(ctx); s != "configure" {
		ctx.err("configure.foo: %v → %s", self, s)
	} else if self.String() != ".self" {
		ctx.err("configure.foo: %v", self)
	} else if s := self.string(ctx); s != "configure" {
		ctx.err("configure.foo: %v → %s", self, s)
	} else if s := d.value.string(ctx); s != m.configure.name {
		ctx.err("configure.foo: %T %v → %v (%v)", d.value, d.value, s, m.configure.name)
	} else if s := d.string(ctx); s != m.configure.name {
		ctx.err("configure.foo: %v → %v", d, s)
	}

	if e, d := testConfigItem(ctx, "FOO1"); len(e) != 1 {
		ctx.err("FOO1")
	} else if d == nil {
		ctx.err("FOO1")
	} else if d.value != nil {
		ctx.err("%v", d)
	}
	if e, d := testConfigItem(ctx, "FOO2"); len(e) != 1 {
		ctx.err("FOO2")
	} else if d == nil {
		ctx.err("FOO2")
	} else if d.value != nil {
		ctx.err("%v", d)
	}
	if e, d := testConfigItem(ctx, "FOO3"); len(e) != 1 {
		ctx.err("FOO3")
	} else if d == nil {
		ctx.err("FOO3")
	} else if d.value != nil {
		ctx.err("%v", d)
	}
	if e, d := testConfigItem(ctx, "FOO4"); len(e) != 1 {
		ctx.err("FOO4")
	} else if d == nil {
		ctx.err("FOO4")
	} else if d.value != nil {
		ctx.err("%v", d)
	}
	if e, d := testConfigItem(ctx, "FOO5"); len(e) != 1 {
		ctx.err("FOO5")
	} else if d == nil {
		ctx.err("FOO5")
	} else if d.value != nil {
		ctx.err("%v", d)
	}

	var f = m.configuration(ctx)
	if f == nil {
		ctx.err("%v: nil configuration", m)
	} else if e := os.Remove(f.fullname()); false && e != nil {
		ctx.err("%v: %v", m, e)
	}

	testCheckExecRecipe = func(_ctx Context, source string, recipe Value) {
		testValidateExecRecipe(ctx, _ctx, source, recipe)
	}
	ctx.universe().configuration.silent = true
	ctx.universe().configure(ctx)
	testCheckExecRecipe = nil

	if e, d := testConfigItem(ctx, "FOO1"); len(e) != 1 {
		ctx.err("FOO1")
	} else if d == nil {
		ctx.err("FOO1")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.string(ctx) != "yes" {
		ctx.err("%v", d.value)
	} else if _, y := d.value.(*answer); !y {
		ctx.err("%v", d.value)
	}
	if e, d := testConfigItem(ctx, "FOO2"); len(e) != 1 {
		ctx.err("FOO2")
	} else if d == nil {
		ctx.err("FOO2")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.string(ctx) != "yes" {
		ctx.err("%v", d.value)
	} else if _, y := d.value.(*answer); !y {
		ctx.err("%v", d.value)
	}
	if e, d := testConfigItem(ctx, "FOO3"); len(e) != 1 {
		ctx.err("FOO3")
	} else if d == nil {
		ctx.err("FOO3")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.string(ctx) != "true" {
		ctx.err("%v", d.value)
	} else if _, y := d.value.(*boolean); !y {
		ctx.err("%v", d.value)
	}
	if e, d := testConfigItem(ctx, "FOO4"); len(e) != 1 {
		ctx.err("FOO4")
	} else if d == nil {
		ctx.err("FOO4")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.string(ctx) != "true" {
		ctx.err("%v", d.value)
	} else if _, y := d.value.(*boolean); !y {
		ctx.err("%v", d.value)
	}
	if e, d := testConfigItem(ctx, "FOO5"); len(e) != 1 {
		ctx.err("FOO5")
	} else if d == nil {
		ctx.err("FOO5")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.string(ctx) != "true" {
		ctx.err("%v", d.value)
	} else if _, y := d.value.(*boolean); !y {
		ctx.err("%v", d.value)
	}

	if s := filepath.Join(filepath.Dir(_tmodules), defaultCK, "custom", configuration_sm); f == nil {
		ctx.err("%v: nil configuration", m)
	} else if f.fullname() != s {
		ctx.err("%v", f)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", configuration_sm)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else { lines := `
FOO1 = yes{}
FOO2 = yes{}
FOO3 = true{}
FOO4 = true{}
FOO5 = true{}
`
		for i, l := range strings.Split(lines, "\n") {
			if l != "" && strings.Count(s, l) != 1 { ctx.err("%d. %s", i, l) }
		}
	}

	runcase(ctx.T, spec, name, func (c *testcase) {
		if d, foo := c.def("FOO1"), c.get("FOO1"); foo == nil {
			erro(of(c, d), "%v", d).debug(1)
		} else if foo.String() != "yes{}" {
			c.err("%T %v ; %v", foo, foo, d)
		} else if s := foo.string(ctx); s != "yes" {
			c.err("%T %v -> %s", foo, foo, s)
		} else if _, y := foo.(*answer); ! y {
			c.err("%T %v ; %v", foo, foo, d)
		}
	})

	if f == nil {
		ctx.err("%v: nil configuration", m)
	} else if e := os.Remove(f.fullname()); false && e != nil {
		ctx.err("%v", e)
	}

	var (
		foo  = ctx.get("foo") // in configuration/custom/configure
		foo1 = ctx.get("FOO1")
		foo2 = ctx.get("FOO2")
		foo3 = ctx.get("FOO3")
		foo4 = ctx.get("FOO4")
		foo5 = ctx.get("FOO5")
	)
	if foo == nil { // in configuration/custom/configure, aka m.configure
		ctx.err("foo is nil")
	} else if  foo.String() != ".self" {
		ctx.err("%T %v", foo, foo)
	} else if s := foo.string(ctx); s != "configure" {
		ctx.err("%T %v -> %s", foo, foo, s)
	}
	if s := foo1.string(ctx); s != "yes" {
		ctx.err("%T %v -> %s", foo1, foo1, s)
	} else if s = foo1.String(); s != "yes{}" {
		ctx.err("%T %v -> %s", foo1, foo1, s)
	}
	if s := foo2.string(ctx); s != "yes" {
		ctx.err("%T %v -> %s", foo2, foo2, s)
	} else if s = foo2.String(); s != "yes{}" {
		ctx.err("%T %v -> %s", foo2, foo2, s)
	}
	if s := foo3.string(ctx); s != "true" {
		ctx.err("%T %v -> %s", foo3, foo3, s)
	} else if s = foo3.String(); s != "true{}" {
		ctx.err("%T %v -> %s", foo3, foo3, s)
	}
	if s := foo4.string(ctx); s != "true" {
		ctx.err("%T %v -> %s", foo4, foo4, s)
	} else if s = foo4.String(); s != "true{}" {
		ctx.err("%T %v -> %s", foo4, foo4, s)
	}
	if s := foo5.string(ctx); s != "true" {
		ctx.err("%T %v -> %s", foo5, foo5, s)
	} else if s = foo5.String(); s != "true{}" {
		ctx.err("%T %v -> %s", foo5, foo5, s)
	}
}
