//
//  Copyright (C) 2012-2024, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

func (l unilo) bases_check_param(ctx Context, implicitBase string, i int, elem, spec Value) {
	switch l.project.name {
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
	case "variant.bootstrap":
		if d := l.scope().finddef("variant"); d != nil {
			errostack(ctx, 16, "non-closure: %v", d).trace()
		}
		if d := closure_finddef(ctx, "variant"); d == nil {
			errostack(ctx, 16, "undef variant").trace()
		}
		switch elem.String() {
		case "./.target/$(dir &(variant))":
			if elem.string(ctx) != "./.target/" {

			}
		}
	}
	return
}

func (l unilo) bases_check(ctx Context, implicitIndex int, implicitBase, absPath string, isDir bool, param Value) {
	switch p := l.project; p.name {
	case "testdefaultconfigure":
		if d := p.resolveDef(ctx, "variant"); d == nil || d.value == nil {
			erro(ctx, "nil variant").trace()
		} else if s, t := ts(d.value), "{=path {=bareword darwin} {=bareword arm64} {=bareword bootstrap}}"; s != t {
			erro(ctx, "variant: %s != %s", s, t).trace()
		}
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if p.bases[0].name != ".base" {
			erro(ctx, "wrong bases[0]: %v", p.bases[0]).trace()
		}
		if false && implicitBase != ".base" {
			erro(ctx, "wrong implicit base: %v %v", implicitIndex, implicitBase).trace()
		}
		if false && !truly(ctx, is_implicit_load{}) {
			erro(ctx, "not implicit: %v %v", ts(param), p.bases).trace()
		}
	case "lib.std":
		if s, t := ts(param), "{=path {=bareword app} {=barecomp {=punctuation .} {=bareword base}}}"; s != t {
			erro(ctx, "param: %s != %s", s, t).trace()
		}
		if len(p.bases) != 1 {
			erro(ctx, "wrong bases: %v", p.bases).trace()
		}
		if b := p.bases[0]; b.name != "app.base" /* || b.spec != "app/.base" */ {
			erro(ctx, "wrong bases[0]: %v %v", b.spec, b.name).trace()
		} else if !isDir { // app/.base is dir
			erro(ctx, "not dir: %v %v, %s", b.spec, b.name, absPath).trace()
		}
		if implicitBase != "" {
			erro(ctx, "wrong implicit base: %v %v", implicitIndex, implicitBase).trace()
		}
	}
}

func (l unilo) directory_check(ctx Context, spec, absDir string) {
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

func (l unilo) configure_check(ctx Context, ident Value, absPath, configure *string) {
    switch l.project.name {
	case "testdefaultconfigure":
		if s, t := ts(l.project.opt.configure), "{=boolean true}"; s != t {
			erro(ctx, "-configure incorrect: %s != %s : %v", s, t, l.project.configure).trace()
		}
		if *configure != "configure" {
			erro(ctx, "incorrect configure name: %s", *configure).trace()
		}
    }
}
