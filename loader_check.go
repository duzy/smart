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
