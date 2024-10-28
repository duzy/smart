//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"bytes"
	"reflect"
	"strings"
	"io/ioutil"
	"path/filepath"
	"fmt"
	"os"
)

func testConfig(ctx *testcase, proj *project, s string) (d *def, res *def) {
	for _, e := range proj.configs {
		if e.ident(ctx) == s {
			if d != nil {
				ctx.err("duplicated config: %v , %v", d, e)
			}
			d = e
		}
	}
	if o := proj.resolve(ctx, s); o != nil { res, _ = o.(*def) }
	return
}

func testConfigureDefault(ctx *testcase, spec, name string) {
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var wd = _workdir(ctx)
	var triple = "arm64-apple-Darwin23.2.0-macho"
	var outtmp, outdir, confsm, ws, s string
	var workspace, workout, rel_remnant *def

	defer func() {
		if confsm != "" { os.RemoveAll(confsm) }
		if outtmp != "" { os.RemoveAll(outtmp) }
		if outdir != "" { os.RemoveAll(outdir) }
	} ()

	if t := proj.unmap_entries(ctx, "FOO", nil); t != nil {
		ctx.err("%v", &proj.entries)
	}
	if t := proj.unmap_entries(ctx, "foo", nil); t == nil {
		ctx.err("%v", &proj.entries)
	}
	if t := proj.unmap_entries(ctx, "stamp", nil); t == nil && false {
		ctx.err("%v", &proj.entries)
	}
	if t := proj.unmap_entries(ctx, "touch", nil); t == nil && false {
		ctx.err("%v", &proj.entries)
	}

	if w := joinpath(testModulesPath, "configure"); proj.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", proj, proj.configure, proj.configure.absPath, w)
	} else if len(proj.configure.bases) != 1 {
		ctx.err("%v", proj.configure.bases)
	} else if proj.configure.bases[0].name != "configure.base" {
		ctx.err("%v", proj.configure.bases[0])
	}

	if workspace = proj.configure.def(ctx, "workspace"); workspace == nil || workspace.value == nil {
		ctx.err("%v", tst{workspace})
	} else if p := workspace.owner(); p == nil {
		ctx.err("%v", tst{workspace})
	} else if p.name != "general" {
		ctx.err("%v ; %v", tst{workspace}, p)
	} else if ws = workspace.value.String(); !strings.HasPrefix(proj.absPath, ws) {
		ctx.err("%v", ws)
	}

	if workout = proj.configure.def(ctx, "workout"); workout == nil || workout.value == nil {
		ctx.err("%v", tst{workout})
	} else if workout.value.String() != joinpath(filepath.Dir(ws), "workout") {
		ctx.err("%v", tst{workout})
	}

	if workext := proj.configure.def(ctx, "workext"); workext == nil || workext.value == nil {
		ctx.err("%v", tst{workext})
	} else if workext.value.String() != joinpath(ws, "external") {
		ctx.err("%v", tst{workext})
	}

	if rel_chop := proj.configure.def(ctx, "rel.chop"); rel_chop == nil || rel_chop.value == nil {
		ctx.err("%v", tst{rel_chop})
	} else if rel_chop.value.String() != fmt.Sprintf("%%%%/.smart/modules/ %s/.smart/modules/ %s/.smart/ %s/", ws, ws, ws) {
		ctx.err("%v != %v", tst{rel_chop}, ws)
	} else if s := filepath.Dir(filepath.Dir(wd)); s != dirs(2, wd) {
		ctx.err("%s != %s", s, dirs(2, wd))
	} else if root := proj.def(ctx, "/"); root == nil || root.value == nil {
		ctx.err("%v", root)
	} else if root.value.String() != wd {
		ctx.err("%v : %s != %s", tst{root}, root.value, wd)
	} else if root := proj.bases[0].def(ctx, "/"); root == nil || root.value == nil {
		ctx.err("%v", root)
	} else if root.value.String() != filepath.Join(wd, ".base") {
		ctx.err("%v : %s != %s/.base", tst{root}, root.value, wd)
	} else if rel_chop := proj.def(ctx, "rel.chop"); rel_chop == nil || rel_chop.value == nil {
		ctx.err("%v", tst{rel_chop})
	} else if rel_chop.value.String() != s+"/" {
		ctx.err("%v : %s != %s/", tst{rel_chop}, rel_chop.value, s)
	}

	if rel_remnant = proj.configure.def(ctx, "rel.remnant"); rel_remnant == nil || rel_remnant.value == nil {
		ctx.err("%v", tst{rel_remnant})
	} else if t := rel_remnant.value.String(); t != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("%v != %v : %s", tst{rel_remnant}, ws, t)
	} else if t := rel_remnant.value.string(ctx); t != "testdata/configuration" {
		ctx.err("%v != %v : %s", tst{rel_remnant}, ws, t)
	}

	if variant := proj.def(ctx, "variant"); variant == nil || variant.value == nil {
		ctx.err("%v", tst{variant})
	} else if t := ts(variant.value); t != "{=path {=word darwin} {=word arm64} {=word bootstrap}}" {
		ctx.err("%v : %s", tst{variant}, t)
	} else if t := variant.value.string(ctx); t != "darwin/arm64/bootstrap" {
		ctx.err("%v : %s", tst{variant}, t)
	}

	if variant_tag := proj.def(ctx, "variant.tag"); variant_tag == nil || variant_tag.value == nil {
		ctx.err("%v", tst{variant_tag})
	} else if t := ts(variant_tag.value); t != "{=word bootstrap}" {
		ctx.err("%v : %s", tst{variant_tag}, t)
	} else if t := variant_tag.value.string(ctx); t != "bootstrap" {
		ctx.err("%v : %s", tst{variant_tag}, t)
	}

	if target_arch := proj.configure.def(ctx, "target.arch"); target_arch == nil || target_arch.value == nil {
		ctx.err("%v", tst{target_arch})
	} else if t := ts(target_arch.value); t != "{=word arm64}" {
		ctx.err("%v : %s", tst{target_arch}, t)
	} else if t := target_arch.value.string(ctx); t != "arm64" {
		ctx.err("%v : %s", tst{target_arch}, t)
	}

	if target_os := proj.configure.def(ctx, "target.os"); target_os == nil || target_os.value == nil {
		ctx.err("%v", tst{target_os})
	} else if t := ts(target_os.value); t != "{=word darwin}" {
		ctx.err("%v : %s", tst{target_os}, t)
	} else if t := target_os.value.string(ctx); t != "darwin" {
		ctx.err("%v : %s", tst{target_os}, t)
	}

	if target_triple := proj.configure.def(ctx, "target.triple"); target_triple == nil || target_triple.value == nil {
		ctx.err("%v", tst{target_triple})
	} else if t := ts(target_triple.value); t != "{=delegate {=builtin join} {=list {=compound {=closure {=def target.arch}} {=closure {=compound {=word target} {=punct .} {=word sub}}}} {=closure {=def target.vendor}} {=closure {=def target.sys}} {=closure {=def target.abi}}} {=list {=flag {=null}}}}" {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if v := target_triple.value.expand(final{ctx}); v == nil || v == target_triple.value {
		ctx.err("%v : %s", tst{target_triple}, v)
	} else if t := v.String(); t != "arm64&(target.sub)-apple-'Darwin'23.2.0-macho" {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if t := ts(v); t != "{=compound {=word arm64} {=closure {=compound {=word target} {=punct .} {=word sub}}} {=flag {=null}} {=word apple} {=flag {=null}} {=strlit Darwin} {=decimal 23} {=punct .} {=decimal 2} {=punct .} {=decimal 0} {=flag {=null}} {=word macho}}" {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if t := target_triple.value.string(ctx); t != triple {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if t := v.string(ctx); t != triple {
		ctx.err("%v : %s", tst{target_triple}, t)
	}

	if target_out := proj.configure.def(ctx, "target.out"); target_out == nil || target_out.value == nil {
		ctx.err("%v", tst{target_out})
	} else if t := target_out.value.String(); t != workout.string(ctx)+"/&(target.triple)/&(variant.tag)" {
		ctx.err("%v : %s", tst{target_out}, t)
	} else if t := target_out.string(ctx); t != workout.string(ctx)+"/"+triple+"/bootstrap" {
		ctx.err("%v : %s", tst{target_out}, t)
	} else {
		outdir = t
	}

	if target_tmp := proj.configure.def(ctx, "target.tmp"); target_tmp == nil || target_tmp.value == nil {
		ctx.err("%v", tst{target_tmp})
	} else if t := target_tmp.value.String(); t != "&(target.out)/tmp" {
		ctx.err("%v : %s", tst{target_tmp}, t)
	} else if t := target_tmp.value.string(ctx); t != workout.string(ctx)+"/"+triple+"/bootstrap/tmp" {
		ctx.err("%v : %s", tst{target_tmp}, t)
	}

	if d := proj.configure.def(ctx, "configure.cc"); d == nil || d.value == nil {
		ctx.err("%v", tst{d})
	} else if t := d.value.String(); t != "&(cc)" {
		ctx.err("%v → %s", tst{d}, t)
	} else if t := d.value.string(ctx); t == "" {
		ctx.err("%v → %s", tst{d}, t)
	}

	s = "outtmp"
	if x := proj.configure.def(ctx, s); x == nil {
		ctx.err("%s", s)
	} else if y := proj.def(ctx, s); y == nil {
		ctx.err("%s", s)
	} else if x != y {
		ctx.err("%v != %v", x, y)
	} else if v := x.value; v == nil {
		ctx.err("%v", tst{x})
	} else if s, t := "&(target.tmp)/&(rel.remnant)", v.String(); s != t {
		ctx.err("%v: %s != %s", tst{x}, s, t)
	} else if s := x.string(ctx); s == "" {
		ctx.err("%v: %s", tst{x}, s)
	} else if t := x.string(closure_with(ctx, proj.configure)); t == "" {
		ctx.err("%v: %s", tst{x}, t)
	} else if s == t {
		ctx.err("%v : %s == %s", x, s, t)
	} else {
		outtmp = x.string(ctx)

		if d, t := proj.tempdir(ctx); t != outtmp {
			ctx.err("tempdir: %s != %s (%v)", t, outtmp, d)
		}

		confsm = joinpath(outtmp, configuration_sm)
	}

	if _, y := ctx.srcs[confsm]; y {
		ctx.err("%v: already loaded configuration.sm, %v", proj, reflect.ValueOf(ctx.srcs).MapKeys())
	}

	if c := proj.tempfile(ctx, configuration_sm); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if c.fullname() != confsm {
			ctx.err("%s: %s != %s", proj.name, c, confsm)
		}
		if !c.exists() {
			ctx.err("%s: configuration not saved: %s", proj.name, c)
		}
		if f := proj.file(ctx, configuration_sm); f == nil {
			ctx.err("%s: %s", proj.name, configuration_sm)
		} else if t := f.fullname(); t != confsm {
			ctx.err("%s: %s != %s", proj.name, t, confsm)
		}
	}

	if c := proj.configuration_sm(ctx); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if c.fullname() != proj.configuration.fullname() {
			ctx.err("%s: %v != %v", proj.name, proj.configuration, c)
		}
		if s, t := c.fullname(), confsm; s != t {
			ctx.err("%v : %s != %s", proj.name, s, t)
		} else if !c.exists() {
			note(pc(ctx,s), "no such configuration.sm")
			ctx.err("%v : %s %v", proj.name, c, c.exists())
		} else if t, e := ioutil.ReadFile(s); e != nil {
			ctx.err("%v", e)
		} else if !bytes.Contains(t, []byte("configure FOO = {=self "+proj.name+"}\n")) {
			ctx.err("%s", t)
		}
	}

	// Checking coherence of configuration with proj.configure
	if d, t := proj.configure.tempdir(ctx); t != outtmp {
		ctx.err("tempdir: %s != %s (%v)", t, outtmp, d)
	}
	if c := proj.configure.tempfile(ctx, configuration_sm); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if f := proj.configure.file(ctx, configuration_sm); f == nil {
			ctx.err("%s: %s", proj.name, configuration_sm)
		} else if t := f.fullname(); t != confsm {
			ctx.err("%s: %s != %s", proj.name, t, confsm)
		}
	}

	if c := proj.configure.configuration_sm(ctx); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if pc := proj.configuration; pc != nil {
			if s, t := pc.fullname(), c.fullname(); s != t {
				ctx.err("%s: %s != %s", proj.name, s, t)
			} else if false && pc != c {
				ctx.err("%s: %v != %v : %s %s", proj.name, pc, c, s, t)
			}
		}
	}

	// Checking configure defs

	s = "FOO"
	if e, d := testConfig(ctx, proj, s); e == nil {
		ctx.err("%s", s)
	} else if d == nil {
		ctx.err("%s", s)
	} else if proj.configuration == nil {
		if d.value != nil {
			ctx.err("%s: already defined : %v", proj.name, d)
		}
	} else if proj.configuration.exists() {
		if d.value == nil {
			ctx.err("%s: not defined : %v", proj.name, d)
		}
	}

	if i, e := os.Stat(confsm); e != nil || i == nil {
		c := proj.configuration
		note(pc(ctx,c.fullname()), "%v %v", c, c.exists())
		ctx.err("%v", e)
	} else if b, e := ioutil.ReadFile(confsm); e != nil {
		ctx.err("%v", e)
	} else if !bytes.Contains(b, []byte("FOO = {=self testdefaultconfigure}\n")) {
		ctx.err("%s", b)
	}

	s = "FOO"
	if d := ctx.def(s); d == nil || d.value == nil {
		ctx.err("%s", s)
	} else if d.value.ident(ctx) != ".self" {
		ctx.err("%v", tst{d.value})
	} else if d.value.String() != "{=self testdefaultconfigure}" {
		ctx.err("%v", tst{d.value})
	} else if ts(d.value) != "{=self testdefaultconfigure}" {
		ctx.err("%v", tst{d.value})
	}

	return
	/* ctx.run */(func (c *testcase) {
		if _, y := c.srcs[confsm]; !y {
			ctx.err("%v: %v", proj, reflect.ValueOf(c.srcs).MapKeys())
		}

		var p = _project(c)
		if p.configuration == nil {
			ctx.err("%s: nil configuration", p)
		} else if s := p.configuration.fullname(); s != confsm {
			ctx.err("%s: %s != %s", p, s, confsm)
		}

		if f := p.configuration_sm(ctx); f == nil {
			c.err("%v", p)
		} else if f.fullname() != confsm {
			c.err("%s != %s", f.fullname(), confsm)
		} else if f.stat(c) == nil {
			prompt(c, "%s:1: %v : no such %s\n", confsm, p, f.name)
			c.err("%v", f)
		} else if b, e := ioutil.ReadFile(confsm); e != nil {
			c.err("%v", e)
		} else if !bytes.Contains(b, []byte("FOO = {=self "+p.name+"}\n")) {
			c.err("%s", b)
		}

		s := "FOO"
		if x, y := p.elems[s]; !y {
			c.err("%s", s)
		} else if d, y := x.(*def); !y {
			c.err("%v : %v", tst{x}, p)
		} else if d.o != defConfig {
			c.err("%v : %v : %v", d.o, d, p)
		} else if d := p.finddef("FOO"); d == nil || d != x {
			c.err("FOO: %v %v", x, d)
		} else if d.o != defConfig {
			c.err("%v : %v : %v", d.o, d, p)
		} else if v := d.value; v == nil {
			c.err("%v", d)
		} else if v.String() != "{=self "+p.name+"}" {
			c.err("%v : %v", d, tst{v})
		} else if v.string(c) != name {
			c.err("%v : %v → %s", d, tst{v}, v.string(c))
		} else if _, y := v.(self); !y {
			c.err("%v : %v", typeof(v), v)
		}
		if d := c.def(s); d == nil {
			c.err("%s", s)
		} else if d.o != defConfig {
			c.err("%v : %v", d.o, d)
		} else if v := d.value; v == nil {
			c.err("%v", d)
		} else if v.String() != "{=self "+p.name+"}" {
			c.err("%v : %v", d, tst{v})
		} else if v.string(c) != name {
			c.err("%v : %v → %s", d, tst{v}, v.string(c))
		} else if _, y := v.(self); ! y {
			c.err("%v : %v", typeof(v), v)
		}
	})(ctx)
}

func testConfigureDefault2(ctx *testcase, spec, name string) {
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var outdir, outtmp, confsm string

	defer func() {
		if confsm != "" { os.RemoveAll(confsm) }
		if outtmp != "" { os.RemoveAll(outtmp) }
		if outdir != "" { os.RemoveAll(outdir) }
	} ()

	if x := proj.def(ctx, "outtmp"); x == nil { // $//tmp
		ctx.err("%v", proj)
	} else if s, t := x.value.string(ctx), joinpath(proj.absPath, "tmp"); s != t { // $//tmp
		ctx.err("%v : {=%v %v} : %s != %s", proj, typeof(x.value), x.value, s, t)
	} else if t := joinpath(_workdir(ctx), "tmp"); s != t { // $//tmp
		ctx.err("%v : {=%v %v} : %s != %s", proj, typeof(x.value), x.value, s, t)
	} else if p, y := x.value.(*path); !y {
		ctx.err("%v : {=%v %v}", proj, typeof(x.value), x.value)
	} else if !strings.HasSuffix(p.string(ctx), joinpath("", spec, "tmp")) { // $//tmp
		ctx.err("%v : %v (%s)", proj, p, joinpath("", spec, "tmp"))
	} else if x.value.string(ctx) != x.value.string(closure_with(ctx, proj.configure)) {
		ctx.err("%v : %v", proj, x.value)
	} else if o := proj.configure.def(ctx, "outtmp"); o == nil || o.value == nil { // &(target.tmp)/&(rel.remnant)
		ctx.err("%v : %v", proj, proj.configure)
	} else if o.string(ctx) == x.string(ctx) { // diverged (different outtmp)
		ctx.err("%v: %v == %v", proj, o, x)
	} else {
		outtmp = x.string(ctx) // //tmp
		confsm = joinpath(outtmp, configuration_sm)
	}

	if _, y := ctx.srcs[confsm]; y {
		ctx.err("%v: already loaded configuration.sm, %v", proj, reflect.ValueOf(ctx.srcs).MapKeys())
	}

	if c := proj.configuration_sm(ctx); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if c.fullname() != proj.configuration.fullname() {
			ctx.err("%s: %v != %v", proj.name, proj.configuration, c)
		}
		if s, t := c.fullname(), confsm; s != t {
			ctx.err("%v : %s != %s", proj.name, s, t)
		} else if !c.exists() {
			note(pc(ctx,s), "no such configuration.sm")
			ctx.err("%v : %s %v", proj.name, c, c.exists())
		} else if t, e := ioutil.ReadFile(s); e != nil {
			ctx.err("%v", e)
		} else if !bytes.Contains(t, []byte("configure FOO = {=self "+proj.name+"}\n")) {
			ctx.err("%s", t)
		}
	}

	if joinpath(testModulesPath, "configure") != proj.configure.absPath {
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
	} else if x := proj.def(ctx, "rel.chop"); x == nil {
		ctx.err("%v", proj)
	} else if x.value.String() == "" { // "%%/.smart/modules/ $(dir $/)/ $(dir2 $/)/ $(dir3 $/)/"
		ctx.err("%v: %v", proj, ts(x.value))
	} else if x := proj.def(ctx, "rel.remnant"); x == nil {
		ctx.err("%v", proj)
	} else if x.value.String() != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("%v: %v : %s", proj, x, ts(x.value))
	} else if t := x.string(ctx); t == "" {
		ctx.err("%v: %v → %s", proj, x, t)
	} else if filepath.IsAbs(t) {
		ctx.err("%v: %v → %s", proj, x, t)
	} else if strings.HasPrefix(t, pathSep) {
		ctx.err("%v: %v → %s", proj, x, t)
	} else if strings.HasSuffix(t, pathSep) {
		ctx.err("%v: %v → %s", proj, x, t)
	}

	if e, d := testConfig(ctx, proj, "FOO"); e == nil {
		ctx.err("FOO")
	} else if d == nil {
		ctx.err("FOO")
	} else {
		if proj.configuration == nil {
			if d.value != nil {
				ctx.err("%s : already defined : %v", proj.name, d)
			}
		} else if proj.configuration.exists() {
			if d.value == nil {
				ctx.err("%s : not defined : %v", proj.name, d)
			}
		}
	}

	if d := ctx.def("FOO"); d == nil || d.value == nil {
		erro(ctx, "%v", d).trace()
	} else if d.value.String() != "{=self "+proj.name+"}" {
		ctx.err("%v", d)
	} else if d.string(ctx) != proj.name {
		ctx.err("%v ⇒ %s", d, d.string(ctx))
	} else if ts(d.value) != "{=self "+proj.name+"}" {
		ctx.err("%v", tst{d.value})
	}
}

func testConfigureCustom(ctx *testcase) {
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var confsm, outtmp, outdir string

	defer func() {
		if confsm != "" { os.RemoveAll(confsm) }
		if outtmp != "" { os.RemoveAll(outtmp) }
		if outdir != "" { os.RemoveAll(outdir) }
	} ()

	if e, d := testConfig(ctx, proj, "FOO0"); e == nil {
		ctx.err("FOO0")
	} else if d == nil {
		ctx.err("FOO0")
	} else if s, t := "{=decimal 123}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO1"); e == nil {
		ctx.err("FOO1")
	} else if d == nil {
		ctx.err("FOO1")
	} else if s, t := "{=answer yes}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO2"); e == nil {
		ctx.err("FOO2")
	} else if d == nil {
		ctx.err("FOO2")
	} else if s, t := "{=boolean true}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO3"); e == nil {
		ctx.err("FOO3")
	} else if d == nil {
		ctx.err("FOO3")
	} else if s, t := "{=boolean true}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO4"); e == nil {
		ctx.err("FOO4")
	} else if d == nil {
		ctx.err("FOO4")
	} else if s, t := "{=boolean true}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}

	confsm = proj.configuration.fullname()

	if s := confsm; !filepath.IsAbs(s) {
		ctx.err("%v", proj.configuration)
	} else if i, e := os.Stat(s); e != nil {
		note(pc(ctx,s), "no such configuration.sm")
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", configuration_sm)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else {
		lines := []byte(`
configure FOO0 = 123
configure FOO1 = {=yes}
configure FOO2 = {=true}
configure FOO3 = {=true}
configure FOO4 = {=true}
`)
		for i, l := range bytes.Split(lines, []byte("\n")) {
			if len(l) != 0 && bytes.Count(b, l) != 1 { ctx.err("%d. %s", i, l) }
		}
	}
}
