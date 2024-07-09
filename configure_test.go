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
	var proj = _project(ctx)
	for _, e := range proj.configs {
		if e.ident(ctx) == s { res = append(res, e) }
	}
	if o := proj.resolve(ctx, s); o != nil { d, _ = o.(*def) }
	return
}

func testConfigureFoo(ctx *testcase, spec, name string) {
	configuration := joinPath(filepath.Dir(testModulesPath), defaultCK, configuration_sm)
	if e := os.Remove(configuration); e == nil { ctx.err("%v", configuration) }

	var ws string
	var outtmp, workout, workspace *def
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var wd = _workdir(ctx)
	var cc = closure_with(ctx, proj.configure)

	if w := joinPath(testModulesPath, "configure"); proj.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", proj, proj.configure, proj.configure.absPath, w)
	} else if len(proj.configure.bases) != 1 {
		ctx.err("%v", proj.configure.bases)
	} else if proj.configure.bases[0].name != "configure.base" {
		ctx.err("%v", proj.configure.bases[0])
	}

	if o, y := proj.configure.resolve(ctx, "workspace"), false; o == nil {
		ctx.err("%v", proj.configure)
	} else if workspace, y = o.(*def); !y || workspace.value == nil {
		ctx.err("%v", tst{o})
	} else if p := workspace.owner(); p == nil {
		ctx.err("%v", tst{o})
	} else if p.name != "general" {
		ctx.err("%v ; %v", tst{o}, p)
	} else if ws = workspace.value.String(); !strings.HasPrefix(proj.absPath, ws) {
		ctx.err("%v", ws)
	}

	if o, y := proj.configure.resolve(ctx, "workout"), false; o == nil {
		ctx.err("%v", proj.configure)
	} else if workout, y = o.(*def); !y || workout.value == nil {
		ctx.err("%v", tst{o})
	} else if workout.value.String() != joinPath(filepath.Dir(ws), "workout") {
		ctx.err("%v", tst{workout})
	}

	if o := proj.configure.resolve(ctx, "workext"); o == nil {
		ctx.err("%v", proj.configure)
	} else if workext, y := o.(*def); !y || workext.value == nil {
		ctx.err("%v", tst{o})
	} else if workext.value.String() != joinPath(ws, "external") {
		ctx.err("%v", tst{workext})
	}

	if o := proj.configure.resolve(ctx, "rel.chop"); o == nil {
		ctx.err("%v", proj.configure)
	} else if rel_chop, y := o.(*def); !y || rel_chop.value == nil {
		ctx.err("%v", tst{o})
	} else if rel_chop.value.String() != fmt.Sprintf("%%%%/.smart/modules/ %s/.smart/modules/ %s/.smart/ %s/", ws, ws, ws) {
		ctx.err("%v != %v", tst{rel_chop}, ws)
	}
	if s := filepath.Dir(filepath.Dir(wd)); s != dirs(2, wd) {
		ctx.err("%s != %s", s, dirs(2, wd))
	} else if root := proj.resolveDef(ctx, "/"); root == nil || root.value == nil {
		ctx.err("%v", root)
	} else if root.value.String() != wd {
		ctx.err("%v : %s != %s", tst{root}, root.value, wd)
	} else if root := proj.bases[0].resolveDef(ctx, "/"); root == nil || root.value == nil {
		ctx.err("%v", root)
	} else if root.value.String() != filepath.Join(wd, ".base") {
		ctx.err("%v : %s != %s/.base", tst{root}, root.value, wd)
	} else if o := proj.resolve(ctx, "rel.chop"); o == nil {
		ctx.err("%v, %v", proj, proj.configure)
	} else if rel_chop, y := o.(*def); !y || rel_chop.value == nil {
		ctx.err("%v", tst{o})
	} else if rel_chop.value.String() != s+"/" {
		ctx.err("%v : %s != %s/", tst{rel_chop}, rel_chop.value, s)
	}

	if o := proj.configure.resolve(ctx, "rel.remnant"); o == nil {
		ctx.err("%v", proj.configure)
	} else if rel_remnant, y := o.(*def); !y || rel_remnant.value == nil {
		ctx.err("%v", tst{o})
	} else if rel_remnant.value.String() != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("%v != %v", tst{rel_remnant}, ws)
	}

	if o := proj.configure.resolve(ctx, "target.out"); o == nil {
		ctx.err("%v", proj.configure)
	} else if target_out, y := o.(*def); !y || target_out.value == nil {
		ctx.err("%v", tst{o})
	} else if target_out.value.String() != workout.string(ctx)+"/&(target.triple)/&(variant.tag)" {
		ctx.err("%v", tst{target_out})
	}

	if o := proj.configure.resolve(ctx, "target.tmp"); o == nil {
		ctx.err("%v", proj.configure)
	} else if target_tmp, y := o.(*def); !y || target_tmp.value == nil {
		ctx.err("%v", tst{o})
	} else if target_tmp.value.String() != "&(target.out)/tmp" {
		ctx.err("%v", tst{target_tmp})
	}

	if o, y := proj.configure.resolve(ctx, "outtmp"), false; o == nil {
		ctx.err("%v", proj.configure)
	} else if outtmp, y = o.(*def); !y || outtmp.value == nil {
		ctx.err("%v", tst{o})
	} else if outtmp.value.string(ctx) == outtmp.value.string(cc) {
		ctx.err("%v", tst{outtmp})
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("%v", tst{outtmp})
	}

	if o := proj.configure.resolve(ctx, "configure.cc"); o == nil {
		ctx.err("%v", proj.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("%v", tst{o})
	} else if d.value.String() != "&(cc)" {
		ctx.err("%v", tst{d})
	} else if s := d.value.string(cc); s == "" {
		ctx.err("%v → %s", tst{d}, s)
	}

	if e, d := testConfigItem(ctx, "FOO"); len(e) != 1 {
		ctx.err("FOO")
	} else if d == nil {
		ctx.err("FOO")
	} else if d.value != nil {
		ctx.err("%v", d)
	}

	if outtmp == nil {
		ctx.err("outtmp")
	} else if outtmp.value == nil {
		ctx.err("%v", outtmp)
	} else if s := joinPath(outtmp.value.string(ctx), configuration_sm); s == "" {
		ctx.err("%v", outtmp)
	} else if proj.configuration == nil {
		ctx.err("%v: nil configuration file", proj)
	} else if proj.configuration.fullname() != s {
		ctx.err("%v : %s != %v", outtmp.value, s, proj.configuration.fullname())
	}

	if f := proj.configuration_sm(cc); f == nil {
		ctx.err("%v: nil configuration", proj)
	} else if f.fullname() != proj.configuration.fullname() {
		ctx.err("%v: %v %v", proj, f, proj.configuration)
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
		ctx.err("%v", tst{v})
	} else if v.ident(ctx) != ".self" {
		ctx.err("%v → %s", tst{v}, v.string(ctx))
	} else if v.string(ctx) != name {
		ctx.err("%v → %s", tst{v}, v.string(ctx))
	} else if _, y := v.(*self); !y {
		ctx.err("%v", tst{v})
	}

	if i, e := os.Stat(configuration); e == nil || i != nil { ctx.Errorf("%v", e) }

	configuration = joinPath(outtmp/*.value*/.string(ctx), configuration_sm)
	if i, e := os.Stat(configuration); e != nil || i == nil {
		ctx.err("%v", e)
	} else if b, e := ioutil.ReadFile(configuration); e != nil {
		ctx.err("%v", e)
	} else if !strings.Contains(string(b), "FOO = $(.self)\n") {
		ctx.err("%s", b)
	}

	testPromptConfiguration = false//true

	ctx.run(func (c *testcase) {
		if p := _project(c); p.configuration == nil /* || p.configurationSave != nil */ {
			c.err("%v", p/*, p.configurationSave */)
		} else if p.configuration.fullname() != configuration {
			c.err("%v", p.configuration)
		} else if p.configuration.stat(c) == nil {
			prompt(c, "%s:1: %v: no configuration file\n", configuration, p)
			c.err("%v", p.configuration)
		} else if d := p.finddef("FOO"); d == nil {
			erro(c, "FOO").trace()
		} else if v := d.value; v == nil {
			c.err("%v ; %v", d, typeof(v))
		} else if v.String() != "$(.self)" {
			c.err("%v ; %v", d, tst{v})
		} else if v.string(c) != name {
			c.err("%v ; %v → %s", d, tst{v}, v.string(c))
		} else if _, y := v.(*delegate); ! y {
			c.err("%v ; %v", d, typeof(v))
		}
		if d := c.def("FOO"); d == nil {
			erro(c, "FOO").trace()
		} else if v := c.val("FOO"); v == nil {
			c.err("%v", d)
		} else if v.String() != "$(.self)" {
			c.err("%v ; %v", d, tst{v})
		} else if v.string(c) != name {
			c.err("%v ; %v → %s", d, typeof(v), v.string(c))
		} else if _, y := v.(*delegate); ! y {
			c.err("%v ; %v", d, typeof(v))
		}
	})

	testPromptConfiguration = false

	if proj.configuration == nil {
		ctx.err("%v", proj)
	} else if e := os.Remove(proj.configuration.fullname()); e != nil {
		if false { ctx.err("%v: %v", proj, e) }
	}
}

func testConfigureDivergedOuttmp(ctx *testcase, spec, name string) {
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var outtmp Value
	var cc = closure_with(ctx, proj.configure)

	if joinPath(testModulesPath, "configure") != proj.configure.absPath {
		ctx.err("%v", proj)
	} else if o := proj.configure.resolve(ctx, "configure.cc"); o == nil {
		ctx.err("%v", proj.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("%v", tst{o})
	} else if d.value.String() != "&(cc)" {
		ctx.err("%v", tst{d.value})
	} else if d.value.string(ctx) == "" {
		ctx.err("%v → %s", d.value, d.value.string(ctx))
	} else if d := closure_finddef(ctx, "/"); d == nil {
		ctx.err("%v: &/", proj)
	} else if d.value == nil {
		ctx.err("%v: %v", proj, d)
	} else if d.value.string(ctx) != proj.absPath {
		ctx.err("%v: %v", proj, d.value)
	} else if x := proj.resolveDef(ctx, "rel.chop"); x == nil {
		ctx.err("%v", proj)
	} else if x.value.String() == "" { // "%%/.smart/modules/ $(dir $/)/ $(dir2 $/)/ $(dir3 $/)/"
		ctx.err("%v: %v", proj, ts(x.value))
	} else if x := proj.resolveDef(ctx, "rel.remnant"); x == nil {
		ctx.err("%v", proj)
	} else if x.value.String() != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("%v: %v", proj, ts(x.value))
	} else if x.value.string(ctx) == "" && false {
		ctx.err("%v: %v: '%s'", proj, ts(x.value), x.value.string(ctx))
	} else if filepath.IsAbs(x.value.string(ctx)) {
		ctx.err("%v: %v: '%s'", proj, ts(x.value), x.value.string(ctx))
	} else if strings.HasSuffix(x.value.string(ctx), pathSep) {
		ctx.err("%v: %v: '%s'", proj, ts(x.value), x.value.string(ctx))
	}

	if x := proj.resolveDef(ctx, "outtmp"); x == nil { // $//tmp
		ctx.err("%v", proj)
	} else if x.value.String() != joinPath(proj.absPath, "tmp") { // $//tmp
		ctx.err("%v: %v(%v)", proj, typeof(x.value), x.value)
	} else if x.value.String() != joinPath(_workdir(ctx), "tmp") { // $//tmp
		ctx.err("%v: %v(%v)", proj, typeof(x.value), x.value)
	} else if p, y := x.value.(*path); !y {
		ctx.err("%v: %v(%v)", proj, typeof(x.value), x.value)
	} else if !strings.HasSuffix(p.string(ctx), joinPath("", spec, "tmp")) { // $//tmp
		ctx.err("%v: %v (%s)", proj, p, joinPath("", spec, "tmp"))
	} else if x.value.string(ctx) != x.value.string(cc) {
		ctx.err("%v: %v", proj, x.value)
	} else if o := proj.configure.resolveDef(ctx, "outtmp"); o == nil || o.value == nil { // &(target.tmp)/&(rel.remnant)
		ctx.err("%v: %v", proj, proj.configure)
	} else if o.value.string(ctx) == x.value.string(ctx) { // diverged (different outtmp)
		// note(at(ctx,o.value), "%v: %v", proj, o.value.string(ctx))
		// note(at(ctx,o.value), "%v: %v", proj, x.value.string(ctx))
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

	configuration := joinPath(outtmp.string(ctx), configuration_sm)

	// NOTE: checking diverged configuration file (due to different outtmp)
	if configuration == "" {
		ctx.err("%v: %v", proj, outtmp)
	} else if proj.configuration == nil {
		ctx.err("%v: nil configuration file", proj)
	} else if    configuration     ==     proj.configuration.fullname() { // diverged (different outtmp)
		note(at(ctx,outtmp), "%v: %v", proj, proj.configuration.fullname())
		note(at(ctx,outtmp), "%v: %v", proj, configuration)
		ctx.err("%v: %v", proj, proj.configuration)
	}

	if f := proj.configuration_sm(cc); f == nil { // NOTE: this is the real configuration file
		ctx.err("%v: nil configuration", proj)
	} else if proj.configuration == nil {
		ctx.err("%v: configuration", proj)
	} else if proj.configuration == f { // diverged configuration file
		ctx.err("%v: %v == %v", proj, proj.configuration, f)
	} else if proj.configuration.fullname() == f.fullname() {
		note(ctx, "%v: %v", proj, proj.configuration.fullname())
		note(ctx, "%v: %v", proj, f.fullname())
		ctx.err("%v: %v == %v", proj, proj.configuration, f)
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

	if f := proj.configuration; f == nil {
		ctx.err("%v: nil configuration", proj)
	} else if i, e := os.Stat(f.fullname()); e != nil {
		ctx.err("%v: %s: %v", f, configuration_sm, e)
	} else if i == nil {
		ctx.err("missing %s", f.fullname())
	}

	if d, v := ctx.val("FOO"), ctx.val("FOO"); v == nil {
		erro(at(ctx, d), "%v", d).trace()
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
		f := proj.configuration
		s := joinPath(filepath.Dir(testModulesPath), defaultCK, configuration_sm)
		if i, e := os.Stat(s); e == nil || i != nil { ctx.Errorf("%v", e) }
		if f == nil {
			ctx.err("%v: nil configuration file", proj)
		} else if f.fullname() == s { // NOTE: diverged
			erro(ctx, "%s", f.fullname())
			erro(ctx, "%s", s)//.debug()
			ctx.err("%v ; %v", outtmp, f)
		}

		/**/s = joinPath(outtmp.string(ctx), configuration_sm)
		if s != joinPath(outtmp.string(cc ), configuration_sm) {
			erro(ctx, "%s", outtmp.string(cc))
			erro(ctx, "%s", s).trace()
			ctx.Errorf("%v ; %v", outtmp, configuration_sm)
		} else if f == nil {
			ctx.err("%v: nil configuration file", proj)
		} else if f.fullname() == s { // NOTE diverged
			erro(ctx, "%s", f.fullname())
			erro(ctx, "%s", s)//.debug()
			ctx.err("%v ; %v", outtmp, f)
		} else if i, e := os.Stat(/*s*/f.fullname()); e != nil || i == nil {
			erro(ctx, "%s", f.fullname())
			erro(ctx, "%s", s)//.debug()
			ctx.err("%v ; %v", outtmp, f)
		} else if b, e := ioutil.ReadFile(/*s*/f.fullname()); e != nil {
			ctx.Errorf("%v", e)
		} else if !strings.Contains(string(b), "FOO = $(.self)") {
			ctx.Errorf("%s", b)
		}

		testPromptConfiguration = false//true

		ctx.run(func (c *testcase) {
			if d := c.def("FOO"); d == nil {
				c.err("%v: FOO", _project(c))
			} else if d.value == nil {
				c.err("%v: %v", _project(c), d)
			} else if d.value.String() != "$(.self)" {
				c.err("%v: %v", _project(c), d.value)
			} else if d.value.string(c) != _project(c).name {
				c.err("%v: %v", _project(c), d.value)
			} else if d != _project(c).finddef("FOO") {
				c.err("%v: %v != %v", _project(c), d, _project(c).finddef("FOO"))
			}
		})

		testPromptConfiguration = false
	}

	if proj.configuration == nil {
		ctx.err("%v", proj)
	} else if proj.configuration.stat(ctx) != nil {
		ctx.err("%v: %v", proj, proj.configuration)
	} else if f := proj.configuration_sm(ctx); f == nil { // NOTE: this is the real configuration file
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

	var proj = _project(ctx)
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

	var f = proj.configuration_sm(ctx)
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
			erro(c, "%v", d).trace()
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

	if foo := ctx.val("foo") ; foo == nil { // in configuration/custom/configure, aka proj.configure
		ctx.err("foo is nil")
	} else if  foo.String() != "$(.self)" {
		ctx.err("%v{%v}", typeof(foo), foo)
	} else if s := foo.string(ctx); s != "configure" {
		ctx.err("%v{%v} → %s", typeof(foo), foo, s)
	}

	if foo1 := ctx.val("FOO1"); foo1 == nil {
		ctx.err("FOO1 is nil")
	} else if s := foo1.string(ctx); s != "yes" {
		ctx.err("%T %v → %s", foo1, foo1, s)
	} else if s = foo1.String(); s != "yes{}" {
		ctx.err("%T %v → %s", foo1, foo1, s)
	}

	if foo2 := ctx.val("FOO2"); foo2 == nil {
		ctx.err("FOO2 is nil")
	} else if s := foo2.string(ctx); s != "yes" {
		ctx.err("%T %v → %s", foo2, foo2, s)
	} else if s = foo2.String(); s != "yes{}" {
		ctx.err("%T %v → %s", foo2, foo2, s)
	}

	if foo3 := ctx.val("FOO3"); foo3 == nil {
		ctx.err("FOO3 is nil")
	} else if s := foo3.string(ctx); s != "true" {
		ctx.err("%T %v → %s", foo3, foo3, s)
	} else if s = foo3.String(); s != "true{}" {
		ctx.err("%T %v → %s", foo3, foo3, s)
	}

	if foo4 := ctx.val("FOO4"); foo4 == nil {
		ctx.err("FOO4 is nil")
	} else if s := foo4.string(ctx); s != "true" {
		ctx.err("%T %v → %s", foo4, foo4, s)
	} else if s = foo4.String(); s != "true{}" {
		ctx.err("%T %v → %s", foo4, foo4, s)
	}

	if foo5 := ctx.val("FOO5"); foo5 == nil {
		ctx.err("FOO5 is nil")
	} else if s := foo5.string(ctx); s != "true" {
		ctx.err("%T %v → %s", foo5, foo5, s)
	} else if s = foo5.String(); s != "true{}" {
		ctx.err("%T %v → %s", foo5, foo5, s)
	}
}
