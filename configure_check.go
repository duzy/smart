//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func (cc *configurecontext) execute_check(ctx Context, e entry, p *project, s *string, _d **def) {
	switch p.name {
	case "testdefaultconfigure":
		if d := *_d; d == nil {
			erro(ctx, "%v", e).trace()
		} else {
			switch d.name {
			case "FOO":
				if d.value.String() != "{=self testdefaultconfigure}" {
					erro(ctx, "%v", d.value).trace()
				}
			}
		}
	}
}

func (ctx *modifier_configure) execute_check(target, name Value, op string, args []Value, configured *bool, result *Value) {
	switch p := _project(ctx); p.name {
	case "testdefaultconfigure":
		note(ctx, "%s %v %v %v", p.name, target, name, op).debug(2)
	}
}

func (ctx *modifier_configure) x_check(p *project, d *def, result *any) {
	switch p.name {
	case "testdefaultconfigure":
		if false { note(ctx, "%v %v %v", p.name, d, *result).debug(2) }
	}
}
