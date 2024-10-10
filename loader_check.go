//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
	"fmt"
	"bytes"
	"strings"
	"path/filepath"
)

func (l ul) bases_check_param(ctx Context, implicitBase string, i int, elem, spec Value) {
	switch l.project.name {
	case "variant.bootstrap":
		if d := l.scope().finddef("variant"); d != nil {
			errostack(ctx, 16, "non-closure: %v", d).trace()
		}
		if d := closure_finddef(ctx, "variant"); d == nil {
			errostack(ctx, 16, "undef variant").trace()
		}
		if elem.String() == "./.target/$(dir &(variant))" {
			if elem.string(ctx) == "./.target/" {
				erro(ctx, "%v : no variant set", elem).trace()
			}
		} else {
			erro(ctx, "unexpected base: %v : %v", elem, spec).trace()
		}
	case "testdefaultconfigure":
		if i != 0 {
			erro(ctx, "more than one param: %d. %v", i, elem).trace()
		}
		if ts(elem) != "{=file .base}" {
			erro(ctx, "%v %v", elem, ts(elem)).trace()
		}
		if ts(spec) != "{=file .base}" {
			erro(ctx, "%v %v", spec, ts(spec)).trace()
		}
		if elem != spec {
			erro(ctx, "%v != %v", elem, spec).trace()
		}
	case "testvarianttarget":
			if i != 0 {
			erro(ctx, "more than one param: %d. %v", i, elem).trace()
		}
		if ts(elem) != "{=path {=word variant} {=word bootstrap}}" {
			erro(ctx, "%v %v", elem, ts(elem)).trace()
		}
		if ts(spec) != "{=path {=word variant} {=word bootstrap}}" {
			erro(ctx, "%v %v", elem, ts(elem)).trace()
		}
		if elem != spec {
			erro(ctx, "%v != %v", elem, spec).trace()
		}
	}
	return
}

func (l ul) bases_check_i(ctx Context, i, implicitIndex int, implicitBase, absPath string, isDir bool, param Value) {
	switch p := l.project; p.name {
	case "lib.std":
		if s, t := ts(param), "{=path {=word app} {=compound {=punct .} {=word base}}}"; s != t {
			erro(ctx, "param: %s != %s", s, t).trace()
		}
		if b := p.bases[0]; b.name != "app.base" /* || b.spec != "app/.base" */ {
			erro(ctx, "wrong bases[0]: %v %v", b.spec, b.name).trace()
		} else if !isDir {
			erro(ctx, "not dir: %v %v, %s", b.spec, b.name, absPath).trace()
		}
	}
}

func (l ul) bases_check(ctx Context, implicitIndex int, implicitBase string) {
	switch p := l.project; p.name {
	case "general":
		if len(p.bases) != 0 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
	case "variant":
		if len(p.bases) != 0 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
	case "variant.target.base":
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if b := p.bases[0]; b.name != "general" {
			erro(ctx, "wrong bases[0]: %v %v", b.spec, b.name).trace()
		}
	case "variant.target":
		if len(p.bases) != 2 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if b := p.bases[0]; b.name != "variant.target.base" {
			erro(ctx, "wrong bases[0]: %v %v", b.spec, b.name).trace()
		} else if !strings.HasSuffix(b.spec, "/variant/.target/.base") {
			erro(ctx, "wrong bases[0]: %v %v", b.spec, b.name).trace()
		}
		if b := p.bases[1]; b.name != "variant" {
			erro(ctx, "wrong bases[1]: %v %v", b.spec, b.name).trace()
		} else if !strings.HasSuffix(b.spec, "/variant") {
			erro(ctx, "wrong bases[1]: %v %v", b.spec, b.name).trace()
		}
	case "variant.bootstrap":
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if b := p.bases[0]; !strings.HasPrefix(b.name, "variant.target.") { // variant.target.darwin.arm64
			erro(ctx, "wrong bases[0]: %v", b).trace()
		} else if b := b.bases[0]; !strings.HasPrefix(b.name, "variant.target.") { // variant.target.darwin
			erro(ctx, "wrong bases[0]: %v", b).trace()
		} else if b := b.bases[0]; b.name != "variant.target" {
			erro(ctx, "wrong bases[0]: %v", b).trace()
		}
	case "app.base":
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if false && p.bases[0].name != "bootstrap" {
			erro(ctx, "wrong bases[0]: %v", p.bases[0]).trace()
		}
	case "configure.base":
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if p.bases[0].name != "app.base" {
			erro(ctx, "wrong bases[0]: %v", p.bases[0]).trace()
		}
	case "lib.std":
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if b := p.bases[0]; b.name != "app.base" {
			erro(ctx, "wrong bases[0]: %v %v", b.spec, b.name).trace()
		} else if !strings.HasSuffix(b.spec, "/app/.base") {
			erro(ctx, "wrong bases[0]: %v %v", b.spec, b.name).trace()
		}
		if implicitBase != "" {
			erro(ctx, "wrong implicit base: %v %v", implicitIndex, implicitBase).trace()
		}
	case "testdefaultconfigure":
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if p.bases[0].name != ".base" {
			erro(ctx, "wrong bases[0]: %v", p.bases[0]).trace()
		}
		if false && implicitBase != ".base" {
			erro(ctx, "wrong implicit base: %v %v", implicitIndex, implicitBase).trace()
		}
		if d := p.def(ctx, "variant"); d == nil || d.value == nil {
			erro(ctx, "nil variant").trace()
		} else if s, t := ts(d.value), "{=path {=word darwin} {=word arm64} {=word bootstrap}}"; s != t {
			erro(ctx, "variant: %s != %s", s, t).trace()
		}
	case "testdeftwoconfigure":
		if len(p.bases) != 0 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
	case "testcustomconfigure":
		if len(p.bases) != 0 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
	case "testvarianttarget":
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if b := p.bases[0]; b.name != "variant.bootstrap" {
			erro(ctx, "wrong bases[0]: %v", b).trace()
		}
	}
}

func (l ul) directory_check(ctx Context, spec, absDir string) {
	if l.project != nil {
		switch l.project.name {
		case "configure.base", "lib.std":
			if len(l.project.bases) != 1 {
				erro(ctx, "%v: %v, %s %s", l.project, l.project.bases, spec, absDir).trace()
			}
			if l.project.bases[0].name != "app.base" {
				erro(ctx, "%v: %v", l.project, l.project.bases[0].name).trace()
			}
			if false && l.project.bases[0].spec != "app/.base" {
				erro(ctx, "%v: %v", l.project, l.project.bases[0].spec).trace()
			}
		}
	}
}

func (l ul) configure_check(ctx *configure_ctx, ident Value) {
	if ctx.configure != "" {
		if x, y := l.globe.loaded[ctx.abs]; !y || x == nil {
			prompt(ctx, "%v: %v\n", ctx.abs, ctx.configure)
			erro(ctx, "configure not loaded (dir=%v)", ctx.isDir).trace()
		} else if x != ctx.p {
			erro(ctx, "differs : %s : %v != %v", ctx.configure, x, ctx.p).trace()
		}
	}

	switch l.project.name {
	case "lib.c++.inc":
		// note(ctx, "%v %v %v %v", l.project.spec, ctx.configure, l.project.configuration, l.project.opt.configure).debug()
		if ctx.configure != "configure/.base" {
			erro(ctx, "%s : %v %v", l.project.spec, l.project.configuration, l.project.opt.configure).trace()
		}
	case "testdefaultconfigure", "testdeftwoconfigure":
		if s, t := ts(l.project.opt.configure), "{=boolean true}"; s != t {
			erro(ctx, "incorrect -configure: %s != %s : %v", s, t, l.project.configure).trace()
		}
		if ctx.configure != "configure" {
			erro(ctx, "incorrect configure name: %s", ctx.configure).trace()
		}
	}
}

var langs_map = map[string]string{
	"asm"   : "c",
	"c"     : "c",
	"s"     : "c",
	"S"     : "c",
	"cpp"   : "c++",
	"cxx"   : "c++",
	"c++"   : "c++",
	"cc"    : "c++",
	"cu"    : "cuda",
	"cu++"  : "cuda++",
	"cuda"  : "cuda",
	"cuh"   : "cuda",
	"cuh++" : "cuda++",
	"m"     : "objc",
	"mm"    : "objc++",
	"swift" : "swift",
}

type  loading_source string
type _loading_source struct{ string }
type  checked_source string
type _checked_source struct{ string }

func (l ul) pre_source_check(ctx Context, filename string, src any) {
	do(ctx, loading_source(filename))

	var mode string

	if l.project != nil {
		if d := l.project.def(ctx, ".mode"); d == nil {
			erro(ctx, "%v", l.project).trace()
		} else {
			switch mode = d.string(ctx); mode {
			case "clean", "configure", "goals":
			default:
				erro(ctx, "%v : wrong mode : %s", l.project, mode).trace()
			}
		}
	}

	if strings.HasSuffix(filename, "/llvm/Config/do.smart") {
		if false {
			note(ctx, "%s %v %v", bases(3, filename, true), l.project, truly(ctx, _loading_source{filename})).debug()
		}
		if !truly(ctx, _loading_source{filename}) {
			erro(ctx, "%v : %s", l.project, bases(5, filename, true)).trace()
		} else {
			var d = filepath.Dir(filename)
			if s := filepath.Join(d,".configure.declared"); truly(ctx, _loading_source{s}) {
				erro(ctx, "%v : %s", l.project, bases(5, s, true)).trace()
			}
			if s := filepath.Join(d,".configure.appendix"); truly(ctx, _loading_source{s}) {
				erro(ctx, "%v : %s", l.project, bases(5, s, true)).trace()
			}
		}
	}
	if strings.HasSuffix(filename, "/llvm/Config/.configure.declared") {
		erro(ctx, "%v %s", l.project, bases(3, filename, true)).trace()
	}
	if strings.HasSuffix(filename, "/llvm/Config/.configure.appendix") {
		if false {
			note(ctx, "%s %v %v", bases(3, filename, true), l.project, truly(ctx, _loading_source{filename})).debug()
		}
		if !truly(ctx, _loading_source{filename}) {
			erro(ctx, "%v : %s", l.project, bases(5, filename, true)).trace()
		} else {
			var d = filepath.Dir(filename)
			if s := filepath.Join(d,".configure.declared"); truly(ctx, _loading_source{s}) {
				erro(ctx, "%v : %s", l.project, bases(5, s, true)).trace()
			}
			if s := filepath.Join(d,"do.smart"); !truly(ctx, _loading_source{s}) {
				erro(ctx, "%v : %s", l.project, bases(5, s, true)).trace()
			}
			if l.project.name != "llvm.Config" {
				erro(ctx, "wrong project: %v", l.project.name).trace()
			}
		}
		var d = l.project.def(ctx, "LLVM_TARGETS_TO_BUILD")
		if l.project.configuration == nil {
			// note(ctx, "%s : %v", mode, d).debug()
		} else if d == nil {
			erro(pc(ctx,filename,1), "LLVM_TARGETS_TO_BUILD not def")
			erro(pc(ctx,l.p.Position()), "%s", mode)
			erro(ctx, "%s : %v", mode, l.project.configuration).trace()
		} else {
			switch mode {
			case "configure":
				if d.string(ctx) != "" {
					erro(ctx, "%v", d).trace()
				}
			case "clean", "goals":
				if d.string(ctx) == "" {
					erro(ctx, "%v", d).trace()
				}
			}
		}
	}
}

func (l ul) source_check(ctx Context, filename string, src any, text *[]byte, res *Value) {
	if e := recover(); e != nil {
		switch e := e.(type) {
		case trace_err_evoke_loop:
			erro(ctx, "%s : %v %v", l.project.name, e, ts(e.ctx)).trace()
		default:
			var pos positioner = l.p
			if pos == nil { pos = l.project }
			switch typeof(e) {
			case "errorString":
				note(pc(ctx,pos), "%s", e)
			}
			erro(ctx, "%s %s", l.project, bases(3, filename, true)).trace()
		}
	}

	do(ctx, checked_source(filename))

	flat_mode := truly(ctx, is_flat_mode{})
	text_mode := truly(ctx, parse_is_text{})

	if flat_mode {
		if *res != nil {
			erro(ctx, "non-nil result in flat mode: %v", *res).trace()
		}
	}
	if text_mode {
		if *res == nil {
			erro(ctx, "nil result in text mode: %v", *res).trace()
		}
	}

	if !(flat_mode || text_mode) && *res != l.project {
		erro(ctx, "%v : wrong result : %v", l.project.name, *res).trace()
	}

	var mode string

	if d := l.project.def(ctx, ".mode"); d == nil {
		erro(ctx, "%v", l.project).trace()
	} else {
		switch mode = d.string(ctx); mode {
		case "clean", "configure", "goals":
		default:
			erro(ctx, "%v : wrong mode : %v : %s", l.project, d, mode).trace()
		}
	}

	if strings.HasSuffix(filename, pathSep+configuration_sm) {
		if !flat_mode {
			erro(ctx, "not flat mode in %v : res=%v", configuration_sm, *res).trace()
		}
		switch l.project.name {
		case "testdefaultconfigure", "testdeftwoconfigure":
			if !bytes.Contains(*text, []byte("FOO = {=self "+l.project.name+"}\n")) {
				erro(ctx, "wrong text: %s", *text).trace()
			}
			if x, y := l.project.elems["FOO"]; !y {
				prompt(ctx, "%v:\n%s", filename, *text)
				erro(ctx, "FOO not defined : %v", l.project.names()).trace()
			} else if t := l.project.def(ctx, "FOO"); t != x {
				erro(ctx, "%v", tv(t)).trace()
			} else if d, y := x.(*def); !y {
				erro(ctx, "%v : %v", l.project.names(), ts(x)).trace()
			} else if d.o != defConfig {
				erro(ctx, "%v %v", d.o, x).trace()
			} else if d.value.String() != "{=self "+l.project.name+"}" {
				erro(ctx, "%v %v", d.o, x).trace()
			} else if ts(d.value) != "{=self "+l.project.name+"}" {
				erro(ctx, "%v : %v", d.o, ts(d.value)).trace()
			} else if false {
				note(ctx, "%v : %v : %s", l.project, x, filename)
			}
		}
	}

	if strings.Contains(filename, "testdata/configuration/") && strings.HasSuffix(filename, "/do.smart") {
		switch l.project.name {
		case "testdefaultconfigure", "testdeftwoconfigure":
			if !bytes.Contains(*text, []byte("FOO : {(configure)} ; $(.self)\n")) {
				erro(ctx, "wrong text: %s", *text).trace()
			}
			if x, y := l.project.elems["FOO"]; !y {
				prompt(ctx, "%v:\n%s", filename, *text)
				erro(ctx, "FOO not defined : %v", l.project.names()).trace()
			} else if t := l.project.def(ctx, "FOO"); t != x {
				erro(ctx, "%v", ts(t)).trace()
			} else if d, y := x.(*def); !y {
				erro(ctx, "%v : %v", l.project.names(), ts(x)).trace()
			} else if d.o != defConfig {
				erro(ctx, "%v %v", d.o, x).trace()
			} else if false {
				if d.value == nil {
					erro(ctx, "%v %v", d.o, x).trace()
				} else if d.value.String() != "$(.self)" {
					erro(ctx, "%v %v", d.o, x).trace()
				}
			} else if false {
				note(ctx, "%v : %v : %s", l.project, x, filename)
			}
		}
	}

	if strings.HasSuffix(filename, "/variant/do.smart") {
		if l.project.name != "variant" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		}
		if len(l.project.bases) != 0 {
			erro(ctx, "wrong bases: %v", l.project.bases).trace()
		}
		if d := l.project.def(ctx, "variant.debug"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if s := d.value.String(); s != "{=no}" && s != "{=yes}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		} else if s := ts(d.value); s != "{=answer no}" && s != "{=answer yes}" {
			erro(ctx, "%v : %v", l.project, ts(d.value)).trace()
		}
		if d := l.project.def(ctx, "variant.tag"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if ts(d.value) != "{=word unknown}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "variant.name"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if ts(d.value) != "{=closure {=def variant.tag}}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "variant.targets"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if x, y := d.value.(*list); !y {
			erro(ctx, "%v : %v", l.project, d).trace()
		} else if x.len() < 1 {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
	}

	if strings.HasSuffix(filename, "/variant/.target/.base/do.smart") {
		if l.project.name != "variant.target.base" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		}
		if !l.project._isa("general") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}
		if l.project._isa("variant.target") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}
		if l.project._isa("variant") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}

		if d := l.project.def(ctx, "variant.debug"); d != nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		}
		if d := l.project.def(ctx, "variant.tag"); d != nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		}
		if d := l.project.def(ctx, "variant.name"); d != nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		}
		if d := l.project.def(ctx, "variant.targets"); d != nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		}

		var workout string
		if d := l.project.def(ctx, "workout"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if workout = d.string(ctx); workout == "" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "host.triple"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "$(join &(host.arch)&(host.sub) &(host.vendor) &(host.sys) &(host.abi),-)" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "host.out"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != filepath.Join(workout, "&(host.triple)", "&(variant.tag)") {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "host.tmp"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != filepath.Join("&(host.out)", "tmp") {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "target.triple"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "$(join &(target.arch)&(target.sub) &(target.vendor) &(target.sys) &(target.abi),-)" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "target.out"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != filepath.Join(workout, "&(target.triple)", "&(variant.tag)") {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "target.tmp"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != filepath.Join("&(target.out)", "tmp") {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "bootstrap.variant"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if ts(d.value) != "{=word bootstrap}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "bootstrap"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != filepath.Join(workout, "&(host.triple)", "&(bootstrap.variant)") {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "clang_rt"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "clang_rt.$(join $(sure $1) &(clang_rt.tail),-)" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "outtmp"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "&(target.tmp)/&(rel.remnant)" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "outpre"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "&(target.out)/" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "outinc"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "&(target.out)/include" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "outbin"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "&(target.out)/bin" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "outlib"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "&(target.out)/lib" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "outobj"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "&(outtmp)" { //"&(target.out)/obj"
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "prefix"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if d.value.String() != "&(outpre)" { //"&(target.out)/"
			erro(ctx, "%v : %v", l.project, d).trace()
		}
	}

	if strings.HasSuffix(filename, "/variant/.target/do.smart") {
		if l.project.name != "variant.target" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		}
		if !l.project._isa("variant.target.base") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}
		if !l.project._isa("variant") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}
		if !l.project._isa("general") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}

		if d := l.project.def(ctx, "variant.debug"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if s := d.value.String(); s != "{=no}" && s != "{=yes}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		} else if s := ts(d.value); s != "{=answer no}" && s != "{=answer yes}" {
			erro(ctx, "%v : %v", l.project, ts(d.value)).trace()
		}
		if d := l.project.def(ctx, "variant.tag"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if ts(d.value) != "{=word unknown}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "variant.name"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if ts(d.value) != "{=closure {=def variant.tag}}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "variant.targets"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if x, y := d.value.(*list); !y {
			erro(ctx, "%v : %v", l.project, d).trace()
		} else if x.len() < 1 {
			erro(ctx, "%v : %v", l.project, d).trace()
		}

		if d := l.project.def(ctx, "use.*"); d == nil {
			erro(ctx, "%v : %v", l.project).trace()
		} else if ts(d.value) == "{}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		} else if x, y := d.value.(*list); !y {
			erro(ctx, "%v : %v", l.project, d).trace()
		} else if x.len() == 0 {
			erro(ctx, "%v : %v", l.project, d).trace()
		}

		for k, v := range langs_map {
			var s = "lang."+k
			if d := l.project.def(ctx, s); d == nil {
				erro(ctx, "%v : undefined %s", l.project, s).trace()
			} else if ts(d.value) != "{=word "+v+"}" {
				erro(ctx, "%v : %v", l.project, d).trace()
			}
		}
	}

	if strings.HasSuffix(filename, "/variant/bootstrap") {
		if l.project.name != "variant.bootstrap" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		}
		if !l.project._isa("variant.target.base") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}
		if !l.project._isa("variant.target") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}
		if !l.project._isa("variant") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}
		if !l.project._isa("general") {
			erro(ctx, "%v : %v", l.project, l.project.bases).trace()
		}

		if d := l.project.def(ctx, "variant.debug"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if s := d.value.String(); s != "{=no}" && s != "{=yes}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		} else if s := ts(d.value); s != "{=answer no}" && s != "{=answer yes}" {
			erro(ctx, "%v : %v", l.project, ts(d.value)).trace()
		}
		if d := l.project.def(ctx, "variant.tag"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if ts(d.value) != "{=word bootstrap}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
		if d := l.project.def(ctx, "variant.name"); d == nil {
			erro(ctx, "%v : %v", l.project, l.project.names()).trace()
		} else if ts(d.value) != "{=closure {=def variant.tag}}" {
			erro(ctx, "%v : %v", l.project, d).trace()
		}
	}

	if strings.HasSuffix(filename, "/app/.base/do.smart") {
		if l.project.name != "app.base" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		}

		w := l.project.absPath//"/Volumes/workspace/.smart/modules/app/.base"
		t := fmt.Sprintf("$(if $(equal &/,%s),,%s/.autoload", w, w)
		if d := l.project.def(ctx, ".autoload.declared"); d == nil {
			erro(ctx, ".autoload.declared").trace()
		} else if d.value.String() != t+".declared)" {
			erro(ctx, "%v != %s.declared)", d, t).trace()
		}
		if d := l.project.def(ctx, ".autoload.appendix"); d == nil {
			erro(ctx, ".autoload.appendix").trace()
		} else if d.value.String() != t+".appendix)" {
			erro(ctx, "%v != %s.appendix)", d, t).trace()
		}
	}

	var srcdir, srcinc string
	var ws = l.project.def(ctx, "workspace")

	if strings.HasSuffix(filename, "/external/do.smart") {
		if l.project.name != "external" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		} else if ws == nil {
			erro(ctx, "%v: workspace", l.project.name).trace()
		}
		if d := l.project.def(ctx, "srcdir"); d == nil {
			erro(ctx, "srcdir").trace()
		} else if srcdir = d.string(ctx); srcdir == "" {
			erro(ctx, "%v", d).trace()
		} else if srcdir != ws.string(ctx)+"/external" {
			erro(ctx, "%v %s", d, l.project.absPath).trace()
		}
	}

	if strings.HasSuffix(filename, "/external/llvm-project/do.smart") {
		if l.project.name != "external.llvm-project" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		} else if ws == nil {
			erro(ctx, "%v: workspace", l.project.name).trace()
		}
		if d := l.project.def(ctx, "srcdir"); d == nil {
			erro(ctx, "srcdir").trace()
		} else if srcdir = d.string(ctx); srcdir == "" {
			erro(ctx, "%v", d).trace()
		} else if srcdir != ws.string(ctx)+"/external/llvm-project" {
			erro(ctx, "%v %s", d, l.project.absPath).trace()
		}
	}

	if l.project.name == "llvm.Config" {
		if strings.HasSuffix(filename, "/llvm/Config/.configure.appendix") {
			d := l.project.def(ctx, "LLVM_TARGETS_TO_BUILD")
			switch mode {
			case "configure":
				if l.project.configuration != nil {
					errostack(ctx, 5, "not loaded configuration.sm").trace()
				}
				if d == nil {
					erro(ctx, "%s: LLVM_TARGETS_TO_BUILD is undef", mode).trace()
				} else if d.value != nil {
					erro(ctx, "%s: %v", mode, d).trace()
				} else if d.string(ctx) != "" {
					erro(ctx, "%s: %v", mode, d).trace()
				}
			case "clean", "goals":
				if l.project.configuration == nil {
					errostack(ctx, 5, "not loaded configuration.sm").trace()
				}
				if d == nil {
					erro(ctx, "%s: LLVM_TARGETS_TO_BUILD is undef", mode).trace()
				} else if d.value == nil {
					erro(ctx, "%s: %v", mode, d).trace()
				} else if d.string(ctx) == "" {
					erro(ctx, "%s: %v", mode, d).trace()
				}
			}
		}

		if strings.HasSuffix(filename, "/llvm/Config/do.smart") {
			if ws == nil {
				erro(ctx, "%v: workspace", l.project.name).trace()
			}
			if d := l.project.def(ctx, "srcinc"); d == nil {
				erro(ctx, "srcinc").trace()
			} else if srcinc = d.string(ctx); srcinc == "" {
				erro(ctx, "%v", d).trace()
			} else if srcinc != ws.string(ctx)+"/external/llvm-project/llvm/include" {
				erro(ctx, "%v %s", d, l.project.absPath).trace()
			}
			if d := l.project.def(ctx, "src.def.in"); d == nil {
				erro(ctx, "src.def.in").trace()
			} else {
				var m = map[string]struct{}{
					"llvm/Config/AsmParsers.def.in"     :struct{}{},
					"llvm/Config/AsmPrinters.def.in"    :struct{}{},
					"llvm/Config/Disassemblers.def.in"  :struct{}{},
					"llvm/Config/TargetExegesis.def.in" :struct{}{},
					"llvm/Config/TargetMCAs.def.in"     :struct{}{},
					"llvm/Config/Targets.def.in"        :struct{}{},
				}
				if x, y := d.value.(*list); !y {
					erro(ctx, "%v", d.value).trace()
				} else if x.len() != len(m) {
					erro(ctx, "%d, %d; %v", x.len(), len(m), d.value).trace()
				} else {
					for _, e := range x.elems {
						if x, y := e.(*file); !y {
							erro(ctx, "%v", ts(e)).trace()
						} else if x.dir != srcinc {
							erro(ctx, "%s != %s", x.dir, srcinc).trace()
						} else if _, y = m[x.name]; !y {
							erro(ctx, "%v", x.name).trace()
						}
					}
				}
			}
			if d := l.project.def(ctx, "src.h.cmake"); d == nil {
				erro(ctx, "src.h.cmake").trace()
			} else {
				var m = map[string]struct{}{
					"llvm/Config/abi-breaking.h.cmake" :struct{}{},
					"llvm/Config/llvm-config.h.cmake"  :struct{}{},
					"llvm/Config/config.h.cmake"       :struct{}{},
				}
				if x, y := d.value.(*list); !y {
					erro(ctx, "%v", d.value).trace()
				} else if x.len() != len(m) {
					erro(ctx, "%d, %d; %v", x.len(), len(m), d.value).trace()
				} else {
					for _, e := range x.elems {
						if x, y := e.(*file); !y {
							erro(ctx, "%v", ts(e)).trace()
						} else if x.dir != srcinc {
							erro(ctx, "%s != %s", x.dir, srcinc).trace()
						} else if _, y = m[x.name]; !y {
							erro(ctx, "%v", x.name).trace()
						}
					}
				}
			}
			if d := l.project.def(ctx, "headers"); d == nil {
				erro(ctx, "headers").trace()
			} else {
				var m = map[string]struct{}{
					"llvm/Config/AsmPrinters.def"    :struct{}{},
					"llvm/Config/AsmParsers.def"     :struct{}{},
					"llvm/Config/Disassemblers.def"  :struct{}{},
					"llvm/Config/Targets.def"        :struct{}{},
					"llvm/Config/TargetMCAs.def"     :struct{}{},
					"llvm/Config/TargetExegesis.def" :struct{}{},
					"llvm/Config/abi-breaking.h"     :struct{}{},
					"llvm/Config/llvm-config.h"      :struct{}{},
					"llvm/Config/config.h"           :struct{}{},
				}
				if x, y := d.value.(*list); !y {
					erro(ctx, "%v", d.value).trace()
				} else if x.len() != 2 {
					erro(ctx, "%d; %v", x.len(), d.value).trace()
				} else if x1,  y := x.elems[0].(*list); !y {
					erro(ctx, "%v", x.elems[0]).trace()
				} else if x2,  y := x.elems[1].(*list); !y {
					erro(ctx, "%v", x.elems[1]).trace()
				} else if x1.len()+x2.len() != len(m) {
					erro(ctx, "%d, %d, %d; %v", x1.len(), x2.len(), len(m), d.value).trace()
				} else {
					for _, e := range append(x1.elems, x2.elems...) {
						if x, y := e.(*file); !y {
							erro(ctx, "%v", ts(e)).trace()
						} else if x.dir != srcinc {
							erro(ctx, "%s != %s", x.dir, srcinc).trace()
						} else if _, y = m[x.name]; !y {
							erro(ctx, "%v", x.name).trace()
						}
					}
				}
			}
			for _, c := range l.project.filemap.value {
				switch c.Value.String() {
				case "$(headers)", "$(src.def.in)", "$(src.h.cmake)":
					erro(ctx, "unexpanded: %v : %v", c.Value, c.Value.expand(original{ctx,defExpand1})).trace()
				default:
					note(ctx, "%v", c.Value).debug()
				}
			}
			if t := l.project.unmap_files(ctx, strings.Split("llvm/Config/llvm-config.h.cmake", pathSep), nil); t == nil {
				erro(ctx, "%v: %v", l.project.name, &l.project.filemap).trace()
			} else if x, y := t[0].pattern.(*file); !y {
				erro(ctx, "%v", t[0].pattern).trace()
			} else if x.dir != srcinc {
				erro(ctx, "%s != %s", x.dir, srcinc).trace()
			}
			if t := l.project.unmap_files(ctx, strings.Split("llvm/Config/llvm-config.h", pathSep), nil); t == nil {
				erro(ctx, "%v: %v", l.project.name, &l.project.filemap).trace()
			} else if x, y := t[0].pattern.(*file); !y {
				erro(ctx, "%v", t[0].pattern).trace()
			} else if x.dir != srcinc {
				erro(ctx, "%s != %s", x.dir, srcinc).trace()
			}
		}
	}

	if strings.HasSuffix(filename, "/testdata/modules/llvm/config/do.smart") {
		if l.project.name != "testllvmconfig" {
			erro(ctx, "wrong project: %v", l.project.name).trace()
		} else if ws == nil {
			erro(ctx, "%v: workspace", l.project.name).trace()
		}
		if d := l.project.def(ctx, "srcinc"); d == nil {
			erro(ctx, "srcinc").trace()
		} else if srcinc = d.string(ctx); srcinc == "" {
			erro(ctx, "%v", d).trace()
		}
		if t := l.project.unmap_files(ctx, strings.Split("llvm/Config/llvm-config.h.cmake", pathSep), nil); t == nil {
			erro(ctx, "%v: %v", l.project.name, &l.project.filemap).trace()
		} else if x, y := t[0].pattern.(*file); !y {
			erro(ctx, "%v", t[0].pattern).trace()
		} else if x.dir != srcinc {
			erro(ctx, "%s != %s", x.dir, srcinc).trace()
		} else if false {
			note(ctx, "%v", x.dir).debug()
		}
		if t := l.project.unmap_files(ctx, strings.Split("llvm/Config/llvm-config.h", pathSep), nil); t == nil {
			erro(ctx, "%v: %v", l.project.name, &l.project.filemap).trace()
		} else if x, y := t[0].pattern.(*file); !y {
			erro(ctx, "%v", t[0].pattern).trace()
		} else if x.dir != srcinc {
			erro(ctx, "%s != %s", x.dir, srcinc).trace()
		} else if false {
			note(ctx, "%v", x.dir).debug()
		}
	}
}
