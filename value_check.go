//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"reflect"
	"strings"
	"path/filepath"
	"fmt"
)

func ex_check(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, e *bool, res, t, x *Value, a, o *[]Value) {
	if false && truly(ctx, is_test_mode{}) && truly(ctx, propExDef1) {
		if s := p.String(); true && "${.test.0 $1,$2}" == s {
			note(ctx, "1=%v", ts(auto_get(ctx, "1")))
			note(ctx, "2=%v", ts(auto_get(ctx, "2")))
			note(ctx, "p=%v", ts(p))
			note(ctx, "x=%v→%v", ts(_x), ts(*x))
			note(ctx, "o=%v→%v", ts(_o), ts(expand(ctx, _o...)))
			note(ctx, "a=%v→%v", ts(_a), ts(*a))
			note(ctx, "e=%v (expanded)", *e)
			note(ctx, "t=%v", ts(*t))
			note(ctx, "r=%v", ts(*res))
			note(ctx, "%v", ts(ctx)).debug(32)
		}
	}

	switch _x.(type) {
	case *builtin, *def, *project, self:
		if s, t := ts(_x), ts(*x); s != t {
			erro(ctx, "%v → %v → %v", s, t, *res).trace()
		}
	}

    if *res == nil {
		if a, y := _x.(*auto); y {
			if d := auto_find(ctx, a.name); d != nil {
				errostack(ctx, 10, "%v : %v → %v", ts(p), ts(_x), ts(*x)).trace()
			}
		}

		if _cl {
            // TODO: closure checkpoints ...
		} else {
			if x == nil || *x == nil {
				erro(ctx, "%v : %v → %v", ts(p), ts(_x), ts(*x)).trace()
			}
			if d, y := _x.(*def); y && false {
				if d == nil {
					erro(ctx, "%v", ts(ctx)).trace()
				} else if d.value != nil {
					erro(ctx, "%v : %v → %v", ts(p), ts(_x), ts(*x)).trace()
				}
			}
		}
    } else if false && p != *res && equal(ctx, p, *res) {
        erro(ctx, "%v: %p != %p", ts(*res), *res, p).trace()
    }

	if proj := _project(ctx); proj != nil {
		if truly(ctx, is_modifying{}) {
			switch proj.name {
			case "configure.base":
				var ent string
				if t := _entry(ctx).destiny(); t != nil {
					ent = t.string(ctx)
				}
				switch ent {
				case "-library-c":
					ex_check_configure_base_library_c(ctx, p, _x, _a, _o, _l, res, a, o)
				}
			}
		}
		switch proj.spec {
		case "testdata/value/4":
			ex_check_value_4(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/optional":
			ex_check_value_optional(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/bug_01":
			ex_check_value_bug_01(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/rule/shell/for-stdout":
			ex_check_rule_shell_forstdout(ctx, p, _x, _o, _a, res, x, a)
		}
	}
}

func ex_check_configure_base_library_c(ctx Context, p, _x Value, _a, _o []Value, _l token, res *Value, a, o *[]Value) {
	var kind string
	var t = auto_find(ctx, "TARGET")
	var d = auto_find(ctx, "FUNCTION")
	if d != nil && !isTrivial(d.value) {
		kind = "function"
	} else {
		kind = "library"
	}
	switch p.String() {
	case "$(ifdef FUNCTION,function,library)":
		if (*res).String() != kind {
			erro(ctx, "%v", *res).trace()
		}
	case "$(file .configure/$(ifdef FUNCTION,function,library)/$(TARGET).c)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", *res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v", t.value).trace()
		} else if x, y := (*res).(*file); !y {
			erro(ctx, "%v", *res).trace()
		} else if t := filepath.Join(".configure", kind, s+".c"); t != x.name {
			erro(ctx, "%s != %s", x.name, t).trace()
		}
	case "$(file $(s).x)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", *res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v %v", t.value, s).trace()
		} else {
			if len(*a) != 1 {
				erro(ctx, "%v", *a).trace()
			} else if l, y := (*a)[0].(*list); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts((*a)[0])).trace()
			} else if x, y := l.elems[0].(*compound); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(l.elems[0])).trace()
			} else if f, y := x.elems[0].(*file); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(x.elems[0])).trace()
			} else if t := filepath.Join(".configure", kind, s+".c"); t != f.name {
				erro(ctx, "%s != %s", f.name, t).trace()
			}
			if x, y := (*res).(*file); !y {
				erro(ctx, "%v %v", typeof(*res), *res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.x"); t != x.name {
				erro(ctx, "%s != %s", x.name, t).trace()
			}
		}
	case "$(file $(s).o)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", *res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v %v", t.value, s).trace()
		} else {
			if len(*a) != 1 {
				erro(ctx, "%v", *a).trace()
			} else if l, y := (*a)[0].(*list); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts((*a)[0])).trace()
			} else if x, y := l.elems[0].(*compound); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(l.elems[0])).trace()
			} else if f, y := x.elems[0].(*file); !y {
				erro(ctx, "%v %v %v", typeof(*res), *res, ts(x.elems[0])).trace()
			} else if t := filepath.Join(".configure", kind, s+".c"); t != f.name {
				erro(ctx, "%s != %s", f.name, t).trace()
			}
			if x, y := (*res).(*file); !y {
				erro(ctx, "%v %v", typeof(*res), *res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.o"); t != x.name {
				erro(ctx, "%s != %s", x.name, t).trace()
			}
		}
	}
}

func ex_check_value_optional(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, o, a *[]Value) {
	switch s := _x.String(); s {
	case "$_→name":
		if s, t := ts(_x), "{=selection {=delegate {=auto _}}→{=word name}}"; s != t {
			erro(ctx, "%s != %s", s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s != %s", s, t).trace() }
		} else if t := "{=def name}"; s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "$_→name?":
		if s, t := ts(_x), "{=selection {=delegate {=auto _}}→{=condval {=word name}}}"; s != t {
			erro(ctx, "%s != %s", s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s != %s", s, t).trace() }
		} else if t := "{=def name}"; s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "$_→bar?":
		if s, t := ts(_x), "{=selection {=delegate {=auto _}}→{=condval {=word bar}}}"; s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	}
	switch s := p.String() ; s {
	case "$(foo)":
		if s, t := ts(*res), "{=project foo}"; s != t {
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "$($_→name)":
		if s, v := ts(*res), auto_get(ctx, "_"); v == nil {
			if t := "{=delegate {=selection {=delegate {=auto _}}→{=word name}}}"; s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		} else if t := "{=self foo}"; s != t { // {=delegate {=def name}}
			erro(ctx, "%s != %s", s, t).trace()
		}
	case "$($_→name?)":
		if s, v := ts(*res), auto_get(ctx, "_"); v == nil {
			if t := "{=delegate {=selection {=delegate {=auto _}}→{=condval {=word name}}}}"; s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
		} else if t := "{=self foo}"; s != t { // {=delegate {=def name}}
			erro(ctx, "%s != %s", s, t).trace()
		}
	}
}

func ex_check_value_4(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, o, a *[]Value) {
	switch s := ts(_x); s {
	case "{=compound {=punct .} {=word test} {=punct .} {=word foreach}}":
		if s, t := ts(*x), "{=def .test.foreach}"; s != t {
			erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
		}
	case "{=compound {=punct .} {=word test} {=punct .} {=word v}}":
		if s, t := ts(*x), "{=def .test.v}"; s != t {
			erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
		}
	case "{=compound {=punct .} {=word test} {=punct .} {=word x}}":
		switch p.String() {
		case "&(.test.x)":
			if _project(ctx).resolveDef(ctx, ".test.x") == nil {
				if t := ts(*x); s != t {
					erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
				}
				if s, t := ts(*res), "{=closure "+ts(_x)+"}"; s != t {
					erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
				}
			} else if truly(ctx, propExClosure) {
				if s, t := ts(*x), "{=def .test.x}"; s != t {
					erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
				}
				if s, t := ts(*res), "{=compound {=punct .} {=word test} {=punct .} {=word v}}"; s != t {
					erro(ctx, "%s : %s != %s → %s", p, s, t, *res).trace()
				}
			}
		}
	}
	switch s := p.String() ; s {
	case "&(.test.x)":
		if truly(ctx, propExClosure) {
			if s, t := (*res).String(), ".test.v"; s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		} else {
			if t := (*res).String(); s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		}
	case "&(value .test.v)":
		if s, t := ts(_x), ts(*x); s != t {
			erro(ctx, "%v → %v → %v", s, t, *res).trace()
		}
	case "$(value &(.test.x))":
		if s, t := ts(_x), ts(*x); s != t {
			erro(ctx, "%v → %v → %v", s, t, *res).trace()
		}
		if truly(ctx, propExClosure) {
			if s, t := (*res).String(), "xx"; s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		} else {
			if t := (*res).String(); s != t {
				erro(ctx, "%v → %v → %v", s, t, *res).trace()
			}
		}
	}
}

func ex_check_value_bug_01(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, o, a *[]Value) {
	switch s := p.String() ; s {
	case "$1":
		if truly(ctx, propExFinal) {
			x1, x2, x3, x4 := do(ctx, evoke_x{}), do(ctx, evoke_def{}), do(ctx, evoke_def{"bug_0.2"}), do(ctx, evoke_def{"bug_0.1"})
			s1, s2, s3, s4 := ts(x1), ts(x2), ts(x3), ts(x4)
			s0 := (*res).String()

			if false && s0 == s && s1 == "{=def 1}" && x1 == x2 && x3 != nil && x4 != nil {
				erro(ctx, "%s %s %s %s", s1, s2, s3, s4).trace() // %s → %s
			}

			if false {
				if s1 == "{=def 1}" {
					note(ctx, "%s → %s : %s : %s : %s : 1=%s", s, s0, s1, s2, s3, x1.(*def).value)
				} else if xv := *x; ts(xv) == "{=def 1}" {
					note(ctx, "%s → %s : %s : %s : %s : x=%s", s, s0, s1, s2, s3, xv.(*def).value)
				} else {
					note(ctx, "%s → %s : %s : %s : %s : %s", s, s0, s1, s2, s3, ts(*x))
				}
				flush(ctx)
			}
		}

	case "&($2.$_)":
		if x, y := do(ctx, evoke_def{}).(*def); y {
			var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
			var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
			switch x.name {
			case ".flag":
				if r, t := (*res).String(), "&("+v2.String()+".$_)"; r != t {
					erro(ctx, "%s → %s != %s : %v, %v, %v", s, r, t, s1, s2, s3).trace()
				}
			case "bug_0.2":
				if x.value.String() != "$(foreach(-unique) $(foreach $1,&($2.$_)),$_)" {
					erro(ctx, "%v → %v : %v", s, *res, x).trace()
				}
				if s2 == s3 {
					if r, t := (*res).String(), "&("+v2.String()+".$_)"; r != t {
						erro(ctx, "%s → %s != %s : %v", s, r, t, s1).trace()
					}
				} else {
					if t := (*res).String(); s != t {
						erro(ctx, "%s → %s : %v, %v, %v", s, t, s1, s2, s3).trace()
					}
				}
			}
		} else if _, y := do(ctx, evoke_builtin{"foreach"}).(*builtin); y && false {
			erro(ctx, "%v → %v : %v", s, *res, do(ctx, evoke_def{})).trace()
		} else if t := (*res).String(); s != t {
			erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
		} else if false {
			erro(ctx, "%v : %v", s, do(ctx, evoke_x{})).trace()
		}

	case "$(foreach $1,$2.$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if x, y := (*res).(*list); !y || x.len() != 2 {
						erro(ctx, "%v → %v %v : %v, %v, %v", s, *res, ts(*res), s1, s2, s3).trace()
					} else if r, t := (*res).String(), s0+".{$1} "+s0+".{$2}"; s == r {
						erro(ctx, "%v → %v → %v : %v, %v, %v", s, r, t, s1, s2, s3).trace()
					} else if r != t {
						erro(ctx, "%v → %v → %v : %v, %v, %v", s, r, t, s1, s2, s3).trace()
					}
					break
				}
			}
		}
		if s0 == "" {
			if t := (*res).String(); s != t {
				erro(ctx, "%v → %v : %v → %v : %v, %v, %v", s, t, _a, *a, s1, s2, s3).trace()
			}
		}

	case "$(foreach(-unique) $(foreach $1,$2.$_),$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if x, y := (*res).(*list); !y || x.len() != 2 {
						erro(ctx, "%v → %v %v : %v, %v, %v", s, *res, ts(*res), s1, s2, s3).trace()
					} else if r, t := (*res).String(), "{"+s0+".{$1}} {"+s0+".{$2}}"; s == r {
						erro(ctx, "%v → %v → %v : %v, %v, %v", s, r, t, s1, s2, s3).trace()
					} else if r != t {
						erro(ctx, "%v → %v → %v : %v, %v, %v", s, r, t, s1, s2, s3).trace()
					}
					break
				}
			}
		}
		if s0 == "" {
			if t := (*res).String(); s != t {
				erro(ctx, "%v → %v : %v → %v : %v, %v, %v", s, t, _a, *a, s1, s2, s3).trace()
			}
		}

	case "$(foreach(-forth) x y z,$(okay.2 $1 $2,$_,$_))":
		var v1, v2 = auto_get(ctx, "1"), auto_get(ctx, "2")
		var s1, s2 = ts(v1), ts(v2)
		if x, y := do(ctx, evoke_def{"okay.1"}).(*def); y {
			if s1 == "{=list {=delegate {=auto 1}}}" && s2 == "{=list {=delegate {=auto 2}}}" {
				erro(ctx, "%v → %v : %v → %v : %v", s, *res, _a, *a, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v %v : %v → %v", s, x, s1, s2, _a, *a).trace()
			}
		} else if s, t := (*res).String(), "$(okay.2 $1 $2,x,x)? $(okay.2 $1 $2,y,y)? $(okay.2 $1 $2,z,z)?"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, *a, do(ctx, evoke_x{})).trace()
			}
		} else if s, t := ts(*res), "{=list {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word x}} {=list {=word x}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word y}} {=list {=word y}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word z}} {=list {=word z}}}}}"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, *a, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.2 $1 $2,$_,$_)":
		if v := auto_get(ctx, "_"); v == nil {
			if t := (*res).String(); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		} else {
			if s, t := (*res).String(), fmt.Sprintf("$(okay.2 $1 $2,%s,%s)", v, v); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.1 $1,$2)":
		if truly(ctx, propExFinal) {
			if s, t := (*res).String(), "{x.{$1}}? {x.{$2}}? {y.{$1}}? {y.{$2}}? {z.{$1}}? {z.{$2}}?"; s != t {
				erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, *a, do(ctx, evoke_x{})).trace()
			} else {
				var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
				if v1 != nil || v2 != nil || v3 != nil {
					erro(ctx, "%v : %v %v %v : %v", s, v1, v2, v2, do(ctx, evoke_x{})).trace()
				}
			}
		} else if t := (*res).String(); s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, *a, do(ctx, evoke_x{})).trace()
		}

		if s, t := ts(_a), "{=[Value] {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}"; s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, *a, do(ctx, evoke_x{})).trace()
		} else if s := ts(*a); s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, *a, do(ctx, evoke_x{})).trace()
		}
	}
}

func ex_check_rule_shell_forstdout(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, a *[]Value) {
	o := try[origin](ctx, get_origin{})

	switch p.String() {
	case "${.test $1,$2}":
		if a1, a2 := auto_get(ctx, "1"), auto_get(ctx, "2") ; a1 != nil && a2 != nil {
			if ts(*a) != sfmt("{=[Value] {=list %s} {=list %s}}", ts(a1), ts(a2)) {
				errostack(ctx, 5, "%s %s: %s, %s ; %v, %v", typeof(_x), typeof(*x), ts(a1), ts(a2), ts(_a), ts(*a)).trace()
			}
			switch o {
			case defExpand0:
				if ts(*res) != sfmt("{=delegate {=builtin debug} {=list %s %s}}", ts(a1), ts(a2)) {
					errostack(ctx, 3, "%s %s, %s", ts(a1), ts(a2), ts(*res)).trace()
				}
			case defExpand1:
				if ts(*res) != "{=null}" {
					errostack(ctx, 3, "%s, %s", ts(*a), ts(*res)).trace()
				}
			}
		} else if truly(ctx, propExDelegate) {
			if ts(*res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
				errostack(ctx, 3, "%v: %s", p, ts(*res)).trace()
			}
		} else {
			erro(ctx, "%v: %s %s ; %s", p, ts(a1), ts(a2), ts(*res)).trace()
		}
	case "${.test a,b}":
		switch o {
		case defExpand0:
			if ts(*res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
				errostack(ctx, 3, "%v", ts(*res)).trace()
			}
		case defExpand1:
			if ts(*res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(*res)).trace()
			}
		}
	case "$(.test.v3 a,b)":
		switch o {
		case defExpand1:
			if ts(*res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(*res)).trace()
			}
		}
	case "$(debug $(line) $(str))":
		if ts(_a) != "{=[Value] {=list {=delegate {=auto line}} {=delegate {=auto str}}}}" {
			erro(ctx, "%v", ts(_a)).trace()
		}

		if a := _automatic(ctx); a == nil {
			errostack(ctx, 5, "%v", ts(*res)).trace()
		} else {
			keys := reflect.ValueOf(a.defs).MapKeys()

			if x1, y := a.defs["1"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x2, y := a.defs["str"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(*res)).trace()
			}

			if x1, y := a.defs["2"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x2, y := a.defs["line"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(*res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(*res)).trace()
			}
		}
		if a, b := auto_get(ctx, "1"), auto_get(ctx, "str"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(*res)).trace()
		}
		if a, b := auto_get(ctx, "2"), auto_get(ctx, "line"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(*res)).trace()
		}

		switch ts(*a) {
		case "{=[Value] {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}":
			switch o {
			case defExpand0, defExpand1:
				if ts(*res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			}
		case "{=[Value] {=list {=word b} {=word a}}}":
			switch o {
			case defExpand0:
				if ts(*res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			case defExpand1:
				if ts(*res) != "{}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			}
		case "{=[Value]}":
			var t = []Value{auto_get(ctx, "1"), auto_get(ctx, "2")}
			if ts(t) != "{=[Value] {} {}}" {
				errostack(ctx, 5, "%v %v", ts(*x), ts(t)).trace()
			}
			if ts(*res) != "{}" {
				errostack(ctx, 5, "%v", ts(*res)).trace()
			}
		case
			`{=[Value] {=list {=decimal 1} {=strlit test one\n}}}`,
			`{=[Value] {=list {=decimal 2} {=strlit test two\n}}}`,
			`{=[Value] {=list {=decimal 3} {=strlit test thr\n}}}`:
			if ts(*res) != "{}" {
				errostack(ctx, 5, "%v", ts(*res)).trace()
			}
		default:
			errostack(ctx, 5, "untested: %v, %s, %s", o, ts(*a), ts(*res)).trace()
		}
	}

	switch o {
	case 0, defExpand0, defExpand1:
	default:
		errostack(ctx, 5, "untested: %v %s %s", o, ts(_a), ts(*res)).trace()
	}
}

func expand_check_elem(ctx Context, e, v Value) {
	if e == nil || v == nil {
		erro(ctx, "nil : %v %v", tv(e), tv(v)).trace()
	} else {
		var a = e.cmp(ctx, v)
		var b = v.cmp(ctx, e)
		if a != b {
			erro(ctx, "%v → %v : cmp→(%v,%v)", tv(e), tv(v), a, b).trace()
		}
	}
}

func (p *selection) expand_check(ctx Context, _o, _s, res *Value) {
	if equal(ctx, p, *res) {
		col := truly(ctx, propExCondless)
		if col {
			if _, y := (*res).(condval); y {
				erro(ctx, "%v → %v", p, *res).trace()
			}

			var po = p.o; if x, y := po.(condval); y { po = x.Value }
			var ps = p.s; if x, y := ps.(condval); y { ps = x.Value }
			if s, t := ts(*_o), ts(po); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
			if s, t := ts(*_s), ts(ps); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}

			if !cond(p.o) && !cond(p.s) {
				if s, t := (*res).String(), p.String(); s != t {
					erro(ctx, "%v → %v : %s != %s", p, *res, s, t).trace()
				}
				if s, t := ts(*res), ts(p); s != t {
					erro(ctx, "%v → %v : %s != %s", p, *res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(*_o), ts(p.o); s != t {
				erro(ctx, "%s != %s : %v %v", s, t, cond(*_o), cond(p.o)).trace()
			}
			if s, t := ts(*_s), ts(p.s); s != t {
				erro(ctx, "%s != %s : %v %v", s, t, cond(*_s), cond(p.s)).trace()
			}
			if s, t := (*res).String(), p.String(); s != t {
				erro(ctx, "%v → %v : %s != %s", p, *res, s, t).trace()
			}
			if s, t := ts(*res), ts(p); s != t {
				erro(ctx, "%v → %v : %s != %s", p, *res, s, t).trace()
			}
		}
	} else {
		if s, t := (*res).String(), p.String(); s == t {
			erro(ctx, "%v → %v : %s == %s", p, *res, s, t).trace()
		}
		if s, t := ts(*res), ts(p); s == t {
			erro(ctx, "%v → %v : %s == %s", p, *res, s, t).trace()
		}
		if false && *_s != nil {
			if *_o != nil {
				if x, y := (*_o).(condval); y {
					erro(ctx, "%v", ts(x)).trace()
				}
			}
			if *_s != nil {
				if x, y := (*_s).(condval); y {
					erro(ctx, "%v", ts(x)).trace()
				}
			}
		}
	}
	switch proj := _project(ctx) ; proj.spec {
	case "testdata/value/optional":
		p.expand_check_value_optional(ctx, proj, _o, _s, res)
	}
}

func (p *selection) expand_check_value_optional(ctx Context, proj *project, _o, _s, res *Value) {
	switch s := p.String() ; s {
	case "$_→name":
		if v := auto_get(ctx, "_"); v == nil {
			if *_o != p.o {
				erro(ctx, "%v != %v", *_o, p.o).trace()
			}
		} else if s, t := ts(*_o), ts(v); s != t {
			erro(ctx, "%v != %v", s, t).trace()
		}
	case "$_→name?":
	case "$_→bar?":
	case "foo→name?":
	case "foo→bar":
	case "foo→bar?":
	case "fo?→bar":
	case "fo?→bar?":
	}
}

func (p *compound) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        note(ctx, "%v, %v ; %v == %v", res, p==v, p, v)
        note(ctx, "%v", ts(p))
        note(ctx, "%v", ts(v))
        erro(ctx, "%v", ts(ctx)).trace()
    }
}
func (p *compound) expand_check(ctx Context, res *Value) {
	if *res == nil {
		if x, y := recover().(trace_err_evoke_loop); y {
			erro(ctx, "%v : %v", p, x.string).trace()
		} else {
			erro(ctx, "%v : %v", p, ts(p)).trace()
		}
	} else if p.expandable(ctx) && equal(ctx, p, *res) {
		if s := p.String(); strings.Contains(s, "$_") {
			if r := (*res).String(); (*res) == p || r == s || strings.Contains(r, "$_") {
				if d := auto_find(ctx, "_"); d != nil {
					note(ctx, "%v", ts(d))
					note(ctx, "%v → %v", ts(p), ts(*res))
					erro(ctx, "%v", ts(ctx)).trace()
				}
			}
		}
	}
}

func (p condval) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}
func (p condval) expand_check(ctx Context, v, res Value) {
    if cond(v) {
        note(ctx, "%v → %v → %v", p.Value, v, res)
        note(ctx, "%-20v : %v", p.Value, ts(p.Value))
        note(ctx, "%-20v : %v", v,       ts(v))
        note(ctx, "%-20v : %v", res,     ts(res))
        erro(ctx, "%v", ts(ctx)).trace()
    }
    if truly(ctx, propExFinal) {
        // ...
    } else {
        if v == nil {
            note(ctx, "%v → %v", p.Value, res)
            note(ctx, "%v : %v", p.Value, ts(p.Value))
            erro(ctx, "%v", ts(ctx)).trace()
            return
        }
        if !equal(ctx, v, p.Value) && p.Value.String() == v.String() {
            note(ctx, "%v → %v → %v", p.Value, v, res)
            note(ctx, "%-20v : %v", p.Value, ts(p.Value))
            note(ctx, "%-20v : %v", v,       ts(v))
            note(ctx, "%-20v : %v", res,     ts(res))
            erro(ctx, "%v", ts(ctx)).trace()
        }
    }
}

func (p *path) cmp_check(ctx Context, v Value, res *cmpres) {
    if *res != cmpEqual && p.String() == v.String() {
        note(ctx, "%v, %v, %v", p, p==v, *res)
        note(ctx, "%v", ts(p))
        note(ctx, "%v", ts(v))
        erro(ctx, "%v", ts(ctx)).trace()
    }
}
func (p *path) match2_check(ctx Context, srcs []string, full *bool, res, stems *[]string) {
	var s, t = joinpath(srcs...), p.string(ctx)

	if strings.HasPrefix(s, t) && *res == nil {
		note(ctx, "%v →", p)
		note(ctx, "%v →", s)
		note(ctx, "→ %v %v %v", full, res, stems)
		erro(ctx, "%v", ts(ctx)).trace()
	}

	switch t {
	case "%%/.smart/modules/":
		if strings.Contains(s, "/.smart/modules/") {
			if a := *res; len(a) < 4 || a[len(a)-1] != "" {
				erro(ctx, "%v : %v %v %v", s, *full, *res, *stems).trace()
			}
		}
	}
	return
}

func (p *strcomp) cmp_check(ctx Context, v Value, res *cmpres) {
	if *res != cmpEqual && p.String() == v.String() {
		erro(ctx, "%v, %v ⇔ %v, %v ⇔ %v", *res, p, v, ts(p), ts(v)).trace()
	}
}

func (p *punct) cmp_check(ctx Context, v Value, res *cmpres) {
    if *res != cmpEqual && p.String() == v.String() {
		erro(ctx, "%v, %v ⇔ %v, %v ⇔ %v", *res, p, v, ts(p), ts(v)).trace()
    }
}

func (o fullname) cmp_check(ctx Context, v Value, res *cmpres) {
    if *res != cmpEqual && o.String() == v.String() {
		erro(ctx, "%v, %v ⇔ %v, %v ⇔ %v", *res, o, v, ts(o), ts(v)).trace()
    }
}

func (p *list) cmp_check(ctx Context, v Value, res *cmpres) {
	if *res != cmpEqual {
		if s1, s2 := ts(p), ts(v) ; s1 == s2 {
			erro(ctx, "%v, %v ⇔ %v", *res, s1, s2).trace()
		} else if false && p.String() == v.String() {
			erro(ctx, "%v, %v ⇔ %v", *res, s1, s2).trace()
		}
	}
	return
}
func (p *list) expand_check(ctx Context, a []Value, d bool, res Value) {
	if s1, s2 := ts(p.elems), ts(a) ; (d && s1 == s2) || (!d && s1 != s2) {
		for i, v := range a {
			if p.len() <= i { erro(ctx, "%d. %v", i, ts(v)).trace() }

			var t = p.elems[i]

			if x, y := equal(ctx, t, v), equal(ctx, v, t) ; x != y {
				erro(ctx, "%d. %v → %v → equal→(%v,%v)", i, ts(t), ts(v), x, y).trace()
			}
			if x, y := t.cmp(ctx, v), v.cmp(ctx, t) ; x != y {
				erro(ctx, "%d. %v → %v → cmp→(%v,%v)", i, ts(t), ts(v), x, y).trace()
			}
		}
		if false {
			note(ctx, "%s", p.elems)
			note(ctx, "%s", a)
			erro(ctx, "diff=%v", d).trace()
		}
	}
	return
}

func (p *delegate) cmp_check(ctx Context, v Value, res *cmpres) {
    if *res != cmpEqual && /* p.String() == v.String() */ts(p) == ts(v) {
        erro(ctx, "%v, %v ⇔ %v, %v ⇔ %v", *res, p, v, ts(p), ts(v)).trace()
    }
}
func (p *delegate) final_val_check(ctx Context, v Value, nr bool) {
    if false && p.String() == "$/" {
        note(ctx, "%v → %v", ts(p), ts(v))
        erro(ctx, "%v", ts(ctx)).trace()
    }
    if nr {
        var u = v.expandable(ctx)
        if v == p || (false && v.refs(ctx, p.x)) {
            note(ctx, "%v → %v (%v)", ts(p), ts(v), (v==p))
            erro(ctx, "%v", ts(ctx)).trace()
        }
        if u && v == p {
            note(ctx, "%v → %v", ts(p), ts(v))
            erro(ctx, "%v", ts(ctx)).trace()
        }
        if p.String() == v.String() {
            if u {
                note(ctx, "%v → %v , %v", ts(p), ts(v), p.cmp(ctx, v))
                erro(ctx, "%v", ts(ctx)).trace()
            } else {
                note(ctx, "%v → %v , %v", ts(p), ts(v), v.cmp(ctx, p))
                erro(ctx, "%v", ts(ctx)).trace()
            }
            return
        }
    }
}

func (p *closure) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}

func (p *globpat) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}

func (p *regexpat) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}

func (p *project) unmap_files_check(ctx Context, _k any, res *[]filemap_name) {
	switch p.name {
	case "configure.base":
		if x, y := _k.(Value); y && *res == nil && truly(ctx, is_modifying{}) {
			var s = x.string(ctx)
			if strings.HasPrefix(s, ".configure/library/") && strings.HasSuffix(s, ".x") {
				erro(ctx, "%s %v %s", typeof(_k), _k, s).trace()
			}
		}

	case "testllvmconfig":
		var k string
		switch x := _k.(type) {
		case    Value: k = x.String()
		case   string: k = x
		case []string: k = filepath.Join(x...)
		default: erro(ctx, "%v", ts(_k)).trace()
		}
		switch k {
		case "llvm/Config/llvm-config.h.cmake":
			var srcinc string
			if d := p.resolveDef(ctx, "srcinc"); d == nil {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else {
				srcinc = d.string(ctx)
			}
			if n := len(*res); n == 0 {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else if t := (*res)[0]; t.name != k {
				erro(ctx, "%v %v != %v", typeof(k), k, t.name).trace()
			} else if x, y := t.pattern.(*file); !y {
				erro(ctx, "%v %v != %v", typeof(k), k, t.pattern).trace()
			} else if x.dir != srcinc {
				erro(ctx, "%s != %s", x.dir, srcinc).trace()
			}
		case "llvm/Config/llvm-config.h":
			var outinc string
			if d := p.resolveDef(ctx, "outinc"); d == nil {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else {
				outinc = d.string(ctx)
			}
			if n := len(*res); n == 0 {
				erro(ctx, "%v %v %v", p.name, typeof(k), k).trace()
			} else if t := (*res)[0]; t.name != k {
				erro(ctx, "%v %v != %v", typeof(k), k, t.name).trace()
			} else if x, y := t.pattern.(*file); !y {
				erro(ctx, "%v %v != %v", typeof(k), k, t.pattern).trace()
			} else if false && x.dir != outinc {
				erro(ctx, "%s != %s", x.dir, outinc).trace()
			}
		}
	}
}

func select_file_1_check(ctx Context, m filemap_name, _res **file) {
	switch f, p := *_res, _project(ctx); f.name {
	case "llvm/Config/llvm-config.h.cmake":
		s := p.resolveDef(ctx, "srcinc").string(ctx)
		if x, y := m.pattern.(*file); !y {
			erro(ctx, "%v", ts(m.pattern)).trace()
		} else if f.name != x.name {
			erro(ctx, "%s != %s", f.name, x.name).trace()
		} else if f.dir != x.dir {
			erro(ctx, "%s != %s", f.dir, x.dir).trace()
		} else if x.dir != s {
			erro(ctx, "%s != %s", x.dir, s).trace()
		} else if f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		}
	case "llvm/Config/llvm-config.h":
		s := p.resolveDef(ctx, "outinc").string(ctx)
		if x, y := m.pattern.(*file); !y {
			erro(ctx, "%v", ts(m.pattern)).trace()
		} else if f.name != x.name {
			erro(ctx, "%s != %s", f.name, x.name).trace()
		} else if f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		} else if false && x.dir != s {
			erro(ctx, "%s != %s", x.dir, s).trace()
		}
	}
}

func select_files_check(ctx Context, m []filemap_name, res *[]*file) {
	if *res == nil { return }

	var p = _project(ctx)

	switch f := (*res)[0]; f.name {
	case "llvm/Config/llvm-config.h.cmake":
		if s := p.resolveDef(ctx, "srcinc").string(ctx); f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		}
	case "llvm/Config/llvm-config.h":
		if s := p.resolveDef(ctx, "outinc").string(ctx); f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		}
	}
}

func (a as) file_check(ctx Context, projs []*project, v Value, f **file) {
	if len(projs) == 0 { projs = []*project{ _project(ctx) } }

	var p = projs[0]

	if *f == nil {
		var s = v.string(ctx)
		if f := findfile(ctx, s, projs...); f != nil {
			for _, m := range unmap_files(ctx, f) {
				erro(ctx, "FIXME: %v (%s) ⇒ %v", v, s, m)
			}
			erro(ctx, "FIXME: %v (%s) ⇒ %v (%s)", v, s, f, f.fullname())
			errostack(ctx, 5).trace()
		}
	} else if (*f).name == "llvm/Config/llvm-config.h.cmake" {
		if s := p.resolveDef(ctx, "srcinc").string(ctx); (*f).dir != s {
			erro(ctx, "%s != %s", (*f).dir, s).trace()
		}
	} else if (*f).name == "llvm/Config/llvm-config.h" {
		if s := p.resolveDef(ctx, "outinc").string(ctx); (*f).dir != s {
			erro(ctx, "%s != %s", (*f).dir, s).trace()
		}
	}
}

func (a as) fullname_check(ctx Context, projs []*project, t Value, res fullname) {
	if len(projs) == 0 { projs = []*project{ _project(ctx) } }

	var p = projs[0]
	var s = t.string(ctx)

	if res.Value == nil {
		var v = a.Value
		var u = p.unmap_files(ctx, s, nil)
		if 0 < len(u) {
			if t := select_files(ctx, u); t == nil {
				erro(ctx, "%s {=%s %s} %v %v", p.name, typeof(v), v, u, t).trace()
			} else {
				erro(ctx, "%s {=%s %s} %v", p.name, typeof(v), v, u).trace()
			}
		}
	} else {
		switch p.name {
		case "testllvmconfig":
			if x, y := res.Value.(*file); !y {
				erro(ctx, "%v", res.Value).trace()
			} else if x.name == "llvm/Config/llvm-config.h.cmake" {
				if s := p.resolveDef(ctx, "srcinc").string(ctx); x.dir != s {
					erro(ctx, "%s != %s", x.dir, s).trace()
				}
			} else if x.name == "llvm/Config/llvm-config.h" {
				if s := p.resolveDef(ctx, "outinc").string(ctx); x.dir != s {
					erro(ctx, "%s != %s", x.dir, s).trace()
				}
			}
		}
	}
}

func (f flag) hit_check(ctx Context, c *valcache, _res **valcache, fullmatch *bool) {
	switch p, res := _project(ctx), *_res; p.name {
	case "configure.base":
		if cacheMapping(ctx) && res == nil {
			erro(ctx, "%v %v", res, c.ks(true)).trace()
		}

		var v = f.Value
		var k = v.String()
		var cc, y = p.entries.puncs[MINUS]
		if !y {
			if !cacheMapping(ctx) { break }
			erro(ctx, "%v: %v", p.name, v).trace()
		}

		var ss = cc.keys()
		if len(ss) == 0 {
			erro(ctx, "%v: %v", p.name, v).trace()
		}

		if _, y := ss[k]; y {
			if false && res.String() != k {
				erro(ctx, "%v: %v != %v", p.name, (res), v).debug()
			}
		} else if cacheMapping(ctx) {
			erro(ctx, "%v: %v", p.name, v).trace()
		}
	}
}

func (p *builtin) evoke_check(ctx *evocation, res *Value) {
	switch proj := _project(ctx); proj.name {
	case "configure.base":
		switch p.name {
		case "file":
		}
	}
}
