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

func testConfigItem(ctx *testcase, s string) (res []entry, d *def) {
	var proj = ctx.project()
	for _, e := range proj.configs { if e.ident(ctx) == s { res = append(res, e) } }
	if o := proj.resolve(ctx, s); o != nil { d, _ = o.(*def) }
	return
}

func testConfigureFoo(ctx *testcase, spec, name string) {
	{
		s := joinPath(filepath.Dir(testModulesPath), defaultCK, configuration_sm)
		if e := os.Remove(s); e == nil { ctx.Errorf("%v", s) }
	}

	var outtmp *def
	var proj = ctx.project()
	var cc = _closureWith(ctx, proj.configure)

	if proj.configure == nil {
		ctx.err("%v: nil configure", proj)
	} else if w := joinPath(testModulesPath, "configure"); proj.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", proj, proj.configure, proj.configure.absPath, w)
	} else if len(proj.configure.bases) != 1 {
		ctx.err("configure: %v", proj.configure.bases)
	} else if proj.configure.bases[0].name != "configure.base" {
		ctx.err("configure: %v", proj.configure.bases[0])
	} else if o := proj.configure.resolve(ctx, "workspace"); o == nil {
		ctx.err("workspace: %v", proj.configure)
	} else if workspace, y := o.(*def); !y || workspace.value == nil {
		ctx.err("workspace: %T %v", o, o)
	} else if p := workspace.owner(); p == nil {
		ctx.err("workspace: %T %v", o, o)
	} else if p.name != "general" {
		ctx.err("workspace: %T %v ; %v", o, o, p)
	} else if ws := workspace.value.String(); !strings.HasPrefix(proj.absPath, ws) {
		ctx.err("workspace: %T %v", ws, ws)
	} else if o := proj.configure.resolve(ctx, "workout"); o == nil {
		ctx.err("workout: %v", proj.configure)
	} else if workspaceOut, y := o.(*def); !y || workspaceOut.value == nil {
		ctx.err("workout: %T %v", o, o)
	} else if workspaceOut.value.String() != joinPath(filepath.Dir(ws), "workout") {
		ctx.err("workout: %T %v", workspaceOut.value, workspaceOut.value)
	} else if o := proj.configure.resolve(ctx, "workext"); o == nil {
		ctx.err("workext: %v", proj.configure)
	} else if workspaceExt, y := o.(*def); !y || workspaceExt.value == nil {
		ctx.err("workext: %T %v", o, o)
	} else if workspaceExt.value.String() != joinPath(ws, "external") {
		ctx.err("workext: %T %v", workspaceExt.value, workspaceExt.value)
	} else if o := proj.configure.resolve(ctx, "rel.chop"); o == nil {
		ctx.err("rel.chop: %v", proj.configure)
	} else if relchop, y := o.(*def); !y || relchop.value == nil {
		ctx.err("rel.chop: %T %v", o, o)
	} else if relchop.value.String() != fmt.Sprintf("%%%%/.smart/modules/ %s/.smart/modules/ %s/.smart/ %s/", ws, ws, ws) {
		ctx.err("rel.chop: %T %v ; %v", relchop.value, relchop.value, ws)
	} else if o := proj.resolve(ctx, "rel.chop"); o == nil {
		ctx.err("rel.chop: %v", proj.configure)
	} else if relchop2, y := o.(*def); !y || relchop2.value == nil {
		ctx.err("rel.chop: %T %v", o, o)
	} else if s := filepath.Dir(filepath.Dir(filepath.Dir(_workdir(ctx))))+"/"; relchop2.value.String() != s {
		ctx.err("rel.chop: %T %v ; %v", relchop2.value, relchop2.value, s)
	} else if o := proj.configure.resolve(ctx, "rel.remnant"); o == nil {
		ctx.err("rel.remnant: %v", proj.configure)
	} else if relrem, y := o.(*def); !y || relrem.value == nil {
		ctx.err("rel.remnant: %T %v", o, o)
	} else if relrem.value.String() != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("rel.remnant: %T %v ; %v", relrem.value, relrem.value, ws)
	} else if o := proj.configure.resolve(ctx, "target.out"); o == nil {
		ctx.err("target.out: %v", proj.configure)
	} else if targetOut, y := o.(*def); !y || targetOut.value == nil {
		ctx.err("target.out: %T %v", o, o)
	} else if targetOut.value.String() != workspaceOut.string(ctx)+"/&(target.triple)/&(variant.tag)" {
		ctx.err("target.out: %T %v", targetOut.value, targetOut.value)
	} else if o := proj.configure.resolve(ctx, "target.tmp"); o == nil {
		ctx.err("target.tmp: %v", proj.configure)
	} else if targetTmp, y := o.(*def); !y || targetTmp.value == nil {
		ctx.err("target.tmp: %T %v", o, o)
	} else if targetTmp.value.String() != "&(target.out)/tmp" {
		ctx.err("target.tmp: %T %v", targetTmp.value, targetTmp.value)
	} else if o := proj.configure.resolve(ctx, "outtmp"); o == nil {
		ctx.err("outtmp: %v", proj.configure)
	} else if outtmp, y = o.(*def); !y || outtmp.value == nil {
		ctx.err("outtmp: %T %v", o, o)
	} else if outtmp.value.string(ctx) == outtmp.value.string(cc) {
		ctx.err("outtmp: %v", outtmp.value)
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("outtmp: %T %v", outtmp.value, outtmp.value)
	} else if o := proj.configure.resolve(ctx, "configure.cc"); o == nil {
		ctx.err("configure.cc: %v", proj.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.cc: %T %v", o, o)
	} else if d.value.String() != "&(cc)" {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if s := d.value.string(cc); s == "" {
		ctx.err("configure.cc: %v → %s", d.value, s)
	}

	if e, d := testConfigItem(ctx, "FOO"); len(e) != 1 {
		ctx.err("FOO")
	} else if d == nil {
		ctx.err("FOO")
	} else if d.value != nil {
		ctx.err("%v", d)
	}

	if s := joinPath(outtmp.value.string(ctx), configuration_sm); s == "" {
		ctx.err("%v", outtmp)
	} else if proj.configurationFile == nil {
		ctx.err("%v: nil configuration file", proj)
	} else if t := proj.configurationFile.fullname(); t != s {
		noted(ctx, "%v: %v", proj, t)
		noted(ctx, "%v: %v", proj, s)
		ctx.err("%v != %v", outtmp.value, t)
	}

	if f := proj.configuration(cc); f == nil {
		ctx.err("%v: nil configuration", proj)
	} else if f.fullname() != proj.configurationFile.fullname() {
		ctx.err("%v: %v %v", proj, f, proj.configurationFile)
	} else if e := os.Remove(f.fullname()); false && e != nil {
		ctx.err("%v: %v", proj, e)
	}

	testCheckExecRecipe = func(_ctx Context, source string, recipe Value) {
		testValidateExecRecipe(ctx, _ctx, source, recipe)
	}
	testPromptConfiguration = false//true
	configure(ctx, configure_silent{})
	testPromptConfiguration = false
	testCheckExecRecipe = nil

	if d := ctx.def("FOO"); d == nil {
		ctx.err("FOO")
	} else if v := ctx.val("FOO"); v == nil {
		ctx.err("FOO")
	} else if v.String() != "$(.self)" {
		ctx.err("%v", ust{v})
	} else if v.ident(ctx) != ".self" {
		ctx.err("%v → %s", ust{v}, v.string(ctx))
	} else if v.string(ctx) != name {
		ctx.err("%v → %s", ust{v}, v.string(ctx))
	// } else if x, y := v.(expanded); !y {
	// 	ctx.err("%v{%v}", ust{v})
	} else if _, y := v.(*self); !y {
		ctx.err("%v", ust{v})
	}

	configurationFile := joinPath(filepath.Dir(testModulesPath), defaultCK, configuration_sm)
	if i, e := os.Stat(configurationFile); e == nil || i != nil { ctx.Errorf("%v", e) }

	configurationFile = joinPath(outtmp/*.value*/.string(ctx), configuration_sm)
	if i, e := os.Stat(configurationFile); e != nil || i == nil {
		ctx.err("%v", e)
	} else if b, e := ioutil.ReadFile(configurationFile); e != nil {
		ctx.err("%v", e)
	} else if !strings.Contains(string(b), "FOO = $(.self)\n") {
		ctx.err("%s", b)
	}

	testPromptConfiguration = false//true

	ctx.run(func (c *testcase) {
		if p := c.project(); p.configurationFile == nil /* || p.configurationSave != nil */ {
			c.err("%v", p/*, p.configurationSave */)
		} else if p.configurationFile.fullname() != configurationFile {
			c.err("%v", p.configurationFile)
		} else if p.configurationFile.stat(c) == nil {
			prompt(c, "%s:1: %v: no configuration file\n", configurationFile, p)
			c.err("%v", p.configurationFile)
		} else if d := p.scope.FindDef("FOO"); d == nil {
			erro(c, "FOO").debug(1)
		} else if v := d.value; v == nil {
			c.err("%v ; %v", d, typeof(v))
		} else if v.String() != "$(.self)" {
			c.err("%v ; %v", d, ust{v})
		} else if v.string(c) != name {
			c.err("%v ; %v → %s", d, ust{v}, v.string(c))
		} else if _, y := v.(*delegate); ! y {
			c.err("%v ; %v", d, typeof(v))
		}
		if d := c.def("FOO"); d == nil {
			erro(c, "FOO").debug(1)
		} else if v := c.val("FOO"); v == nil {
			c.err("%v", d)
		} else if v.String() != "$(.self)" {
			c.err("%v ; %v", d, ust{v})
		} else if v.string(c) != name {
			c.err("%v ; %v → %s", d, typeof(v), v.string(c))
		} else if _, y := v.(*delegate); ! y {
			c.err("%v ; %v", d, typeof(v))
		}
	})

	testPromptConfiguration = false

	if proj.configurationFile == nil {
		ctx.err("%v", proj)
	} else if e := os.Remove(proj.configurationFile.fullname()); e != nil {
		if false { ctx.err("%v: %v", proj, e) }
	}
}

func testConfigureDivergedOuttmp(ctx *testcase, spec, name string) {
	defer assured(ctx, true)

	var outtmp Value
	var proj = ctx.project()
	var cc = _closureWith(ctx, proj.configure/*, proj*/)

	if proj.configure == nil {
		ctx.err("%v: nil configure", proj)
	} else if joinPath(testModulesPath, "configure") != proj.configure.absPath {
		noted(ctx, "%v: %v", proj, joinPath(testModulesPath, "configure"))
		noted(ctx, "%v: %v", proj, proj.configure.absPath)
		ctx.err("%v", proj)
	} else if o := proj.configure.resolve(ctx, "configure.cc"); o == nil {
		ctx.err("configure.cc: %v", proj.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.cc: %T %v", o, o)
	} else if d.value.String() != "&(cc)" {
		ctx.err("configure.cc: %T %v", d.value, d.value)
	} else if d.value.string(ctx) == "" {
		ctx.err("configure.cc: %v → %s", d.value, d.value.string(ctx))
	} else if d := closureGet(ctx, "/"); d == nil {
		ctx.err("%v: &/", proj)
	} else if d.value == nil {
		ctx.err("%v: %v", proj, d)
	} else if d.value.string(ctx) != proj.absPath {
		ctx.err("%v: %v", proj, d.value)
	} else if x := proj.resolveDef(ctx, "rel.chop"); x == nil {
		ctx.err("%v: rel.chop", proj)
	} else if x.value.String() == "" { // "%%/.smart/modules/ $(dir $/)/ $(dir2 $/)/ $(dir3 $/)/"
		ctx.err("%v: rel.chop: %v(%v)", proj, typeof(x.value), x.value)
	} else if x := proj.resolveDef(ctx, "rel.remnant"); x == nil {
		ctx.err("%v: rel.remnant", proj)
	} else if x.value.String() != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("%v: rel.remnant: %v(%v)", proj, typeof(x.value), x.value)
	} else if x.value.string(ctx) == "" && false {
		ctx.err("%v: rel.remnant: %v(%v): '%s'", proj, typeof(x.value), x.value, x.value.string(ctx))
	} else if filepath.IsAbs(x.value.string(ctx)) {
		ctx.err("%v: rel.remnant: %v(%v): '%s'", proj, typeof(x.value), x.value, x.value.string(ctx))
	} else if strings.HasSuffix(x.value.string(ctx), pathSep) {
		ctx.err("%v: rel.remnant: %v(%v): '%s'", proj, typeof(x.value), x.value, x.value.string(ctx))
	} else if x := proj.resolveDef(ctx, "outtmp"); x == nil { // $//tmp
		ctx.err("%v: outtmp", proj)
	} else if x.value.String() != joinPath(proj.absPath, "tmp") { // $//tmp
		ctx.err("%v: outtmp: %v(%v)", proj, typeof(x.value), x.value)
	} else if x.value.String() != joinPath(_workdir(ctx), "tmp") { // $//tmp
		ctx.err("%v: outtmp: %v(%v)", proj, typeof(x.value), x.value)
	} else if p, y := x.value.(*path); !y {
		ctx.err("%v: outtmp: %v(%v)", proj, typeof(x.value), x.value)
	} else if !strings.HasSuffix(p.string(ctx), joinPath("", spec, "tmp")) { // $//tmp
		ctx.err("%v: outtmp: %v (%s)", proj, p, joinPath("", spec, "tmp"))
	} else if x.value.string(ctx) != x.value.string(cc) {
		ctx.err("%v: %v", proj, x.value)
	} else if o := proj.configure.resolveDef(ctx, "outtmp"); o == nil || o.value == nil { // &(target.tmp)/&(rel.remnant)
		ctx.err("%v: %v: outtmp", proj, proj.configure)
	} else if o.value.string(ctx) == x.value.string(ctx) { // diverged (different outtmp)
		noted(of(ctx,o.value), "%v: %v", proj, o.value.string(ctx))
		noted(of(ctx,o.value), "%v: %v", proj, x.value.string(ctx))
		ctx.err("%v: %v == %v", proj, o.value, x.value)
	} else {
		outtmp = x.value // := $//tmp
	}

	if e, d := testConfigItem(ctx, "FOO"); len(e) != 1 {
		ctx.err("FOO")
	} else if d == nil {
		ctx.err("FOO")
	} else if d.value != nil {
		ctx.err("%v", d)
	}

	configurationFile := joinPath(outtmp.string(ctx), configuration_sm)

	// NOTE: checking diverged configuration file (due to different outtmp)
	if configurationFile == "" {
		ctx.err("%v: %v", proj, outtmp)
	} else if proj.configurationFile == nil {
		ctx.err("%v: nil configuration file", proj)
	} else if    configurationFile     ==     proj.configurationFile.fullname() { // diverged (different outtmp)
		noted(of(ctx,outtmp), "%v: %v", proj, proj.configurationFile.fullname())
		noted(of(ctx,outtmp), "%v: %v", proj, configurationFile)
		ctx.err("%v: %v", proj, proj.configurationFile)
	}

	if f := proj.configuration(cc); f == nil { // NOTE: this is the real configuration file
		ctx.err("%v: nil configuration", proj)
	} else if proj.configurationFile == nil {
		ctx.err("%v: configurationFile", proj)
	} else if proj.configurationFile == f { // diverged configuration file
		ctx.err("%v: %v == %v", proj, proj.configurationFile, f)
	} else if proj.configurationFile.fullname() == f.fullname() {
		noted(ctx, "%v: %v", proj, proj.configurationFile.fullname())
		noted(ctx, "%v: %v", proj, f.fullname())
		ctx.err("%v: %v == %v", proj, proj.configurationFile, f)
	} else if f.stat(ctx) == nil {
		// noop
	} else if e := os.Remove(f.fullname()); e != nil {
		ctx.err("%v: %v", proj, e)
	}

	testCheckExecRecipe = func(_ctx Context, source string, recipe Value) {
		testValidateExecRecipe(ctx, _ctx, source, recipe)
	}
	testPromptConfiguration = false//true
	testConfigurationDiverged = true
	configure(cc, configure_silent{})
	testConfigurationDiverged = false
	testPromptConfiguration = false
	testCheckExecRecipe = nil

	if f := proj.configurationFile; f == nil {
		ctx.err("%v: nil configuration", proj)
	} else if i, e := os.Stat(f.fullname()); e != nil {
		ctx.err("%v: %s: %v", f, configuration_sm, e)
	} else if i == nil {
		ctx.err("missing %s", f.fullname())
	}

	if d, v := ctx.val("FOO"), ctx.val("FOO"); v == nil {
		erro(of(ctx, d), "%v", d).debug(1)
	} else if v.String() != "$(.self)" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if v.string(ctx) != proj.name {
		ctx.err("%v{%v} ⇒ %s", typeof(v), v, v.string(ctx))
	// } else if x, y := v.(expanded); !y {
	// 	ctx.err("%v{%v}", typeof(v), v)
	// } else if _, y := x.Value.(*self); !y {
	// 	ctx.err("%v{%v}", typeof(x.Value), x.Value)
	} else if _, y := v.(*self); !y {
		ctx.err("%v{%v}", typeof(v), v)
	}

	{
		f := proj.configurationFile
		s := joinPath(filepath.Dir(testModulesPath), defaultCK, configuration_sm)
		if i, e := os.Stat(s); e == nil || i != nil { ctx.Errorf("%v", e) }
		if f == nil {
			ctx.err("%v: nil configuration file", proj)
		} else if f.fullname() == s { // NOTE: diverged
			erro(ctx, "%s", f.fullname())
			erro(ctx, "%s", s)//.debug(1)
			ctx.err("%v ; %v", outtmp, f)
		}

		/**/s = joinPath(outtmp.string(ctx), configuration_sm)
		if s != joinPath(outtmp.string(cc ), configuration_sm) {
			erro(ctx, "%s", outtmp.string(cc))
			erro(ctx, "%s", s).debug(1)
			ctx.Errorf("%v ; %v", outtmp, configuration_sm)
		} else if f == nil {
			ctx.err("%v: nil configuration file", proj)
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

		ctx.run(func (c *testcase) {
			if d := c.def("FOO"); d == nil {
				c.err("%v: FOO", c.project())
			} else if d.value == nil {
				c.err("%v: %v", c.project(), d)
			} else if d.value.String() != "$(.self)" {
				c.err("%v: %v", c.project(), d.value)
			} else if d.value.string(c) != c.project().name {
				c.err("%v: %v", c.project(), d.value)
			} else if d != c.project().scope.FindDef("FOO") {
				c.err("%v: %v != %v", c.project(), d, c.project().scope.FindDef("FOO"))
			}
		})

		testPromptConfiguration = false
	}

	if proj.configurationFile == nil {
		ctx.err("%v", proj)
	} else if proj.configurationFile.stat(ctx) != nil {
		ctx.err("%v: %v", proj, proj.configurationFile)
	} else if f := proj.configuration(ctx); f == nil { // NOTE: this is the real configuration file
		ctx.err("%v", proj)
	} else if !strings.HasPrefix(f.fullname(), joinPath(outtmp.string(ctx), "")) {
		ctx.err("%v: %v", proj, f)
	} else if f.stat(ctx) == nil {
		// FIXME: configuration file should exist
	} else if e := os.Remove(f.fullname()); e != nil {
		ctx.err("%v: %v: %v", proj, f, e)
	}

	if e := os.RemoveAll(outtmp.string(ctx)); e != nil {
		ctx.err("%v: %v: %v", proj, outtmp, e)
	}
}

func testConfigureCustom(ctx *testcase) {
	{
		s := joinPath(filepath.Dir(testModulesPath), defaultCK, "custom", configuration_sm)
		if e := os.Remove(s); e == nil { ctx.err("%v", s) }
	}
	{
		s := joinPath(_workdir(ctx), "tmp", configuration_sm)
		if e := os.Remove(s); e == nil { ctx.err("%v", s) }
	}

	var proj = ctx.project()
	if proj.configure == nil {
		ctx.err("%v: nil configure", proj)
	} else if w := _workdir(ctx); proj.configure.absPath != w+pathSep {
		ctx.err("%v.%v: %s != %s", proj, proj.configure, proj.configure.absPath, w)
	} else if o := proj.configure.resolve(ctx, "foo"); o == nil {
		ctx.err("configure.foo: %v %v", proj.configure, proj.configure.absPath)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("configure.foo: %T %v", o, o)
	} else if d.value.String() != "$(.self)" {
		ctx.err("configure.foo: %v{%v}", typeof(d.value), d.value)
	} else if self, y := d.value.(*self); !y || self == nil {
		ctx.err("configure.foo: %v{%v}", typeof(d.value), d.value)
	} else if self.String() != "$(.self)" {
		ctx.err("configure.foo: %v", self)
	} else if s := self.string(ctx); s != "configure" {
		ctx.err("configure.foo: %v → %s", self, s)
	} else if self.String() != "$(.self)" {
		ctx.err("configure.foo: %v", self)
	} else if s := self.string(ctx); s != "configure" {
		ctx.err("configure.foo: %v → %s", self, s)
	} else if s := d.value.string(ctx); s != proj.configure.name {
		ctx.err("configure.foo: %v{%v} → %v (%v)", typeof(d.value), d.value, s, proj.configure.name)
	} else if s := d.string(ctx); s != proj.configure.name {
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

	var f = proj.configuration(ctx)
	if f == nil {
		ctx.err("%v: nil configuration", proj)
	} else if e := os.Remove(f.fullname()); false && e != nil {
		ctx.err("%v: %v", proj, e)
	}

	testCheckExecRecipe = func(_ctx Context, source string, recipe Value) {
		testValidateExecRecipe(ctx, _ctx, source, recipe)
	}
	configure(ctx, configure_silent{})
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

	if s := joinPath(filepath.Dir(testModulesPath), defaultCK, "custom", configuration_sm); f == nil {
		ctx.err("%v: nil configuration", proj)
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

	ctx.run(func (c *testcase) {
		if d, foo := c.def("FOO1"), c.val("FOO1"); foo == nil {
			erro(of(c, d), "%v", d).debug(1)
		} else if foo.String() != "yes{}" {
			c.err("%T %v ; %v", foo, foo, d)
		} else if s := foo.string(ctx); s != "yes" {
			c.err("%T %v → %s", foo, foo, s)
		} else if _, y := foo.(*answer); ! y {
			c.err("%T %v ; %v", foo, foo, d)
		}
	})

	if f == nil {
		ctx.err("%v: nil configuration", proj)
	} else if e := os.Remove(f.fullname()); false && e != nil {
		ctx.err("%v", e)
	}

	var (
		foo  = ctx.val("foo") // in configuration/custom/configure
		foo1 = ctx.val("FOO1")
		foo2 = ctx.val("FOO2")
		foo3 = ctx.val("FOO3")
		foo4 = ctx.val("FOO4")
		foo5 = ctx.val("FOO5")
	)
	if foo == nil { // in configuration/custom/configure, aka proj.configure
		ctx.err("foo is nil")
	} else if  foo.String() != "$(.self)" {
		ctx.err("%v{%v}", typeof(foo), foo)
	} else if s := foo.string(ctx); s != "configure" {
		ctx.err("%v{%v} → %s", typeof(foo), foo, s)
	}
	if s := foo1.string(ctx); s != "yes" {
		ctx.err("%T %v → %s", foo1, foo1, s)
	} else if s = foo1.String(); s != "yes{}" {
		ctx.err("%T %v → %s", foo1, foo1, s)
	}
	if s := foo2.string(ctx); s != "yes" {
		ctx.err("%T %v → %s", foo2, foo2, s)
	} else if s = foo2.String(); s != "yes{}" {
		ctx.err("%T %v → %s", foo2, foo2, s)
	}
	if s := foo3.string(ctx); s != "true" {
		ctx.err("%T %v → %s", foo3, foo3, s)
	} else if s = foo3.String(); s != "true{}" {
		ctx.err("%T %v → %s", foo3, foo3, s)
	}
	if s := foo4.string(ctx); s != "true" {
		ctx.err("%T %v → %s", foo4, foo4, s)
	} else if s = foo4.String(); s != "true{}" {
		ctx.err("%T %v → %s", foo4, foo4, s)
	}
	if s := foo5.string(ctx); s != "true" {
		ctx.err("%T %v → %s", foo5, foo5, s)
	} else if s = foo5.String(); s != "true{}" {
		ctx.err("%T %v → %s", foo5, foo5, s)
	}
}
