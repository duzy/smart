//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"reflect"
	"strings"
	"strconv"
	"path/filepath"
	"time"
	"fmt"
)

func ex_check(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, res, t, x *Value, a, o *[]Value) {
	if false && truly(ctx, propExDef1) {
		if s := p.String(); "$(name?)" == s {
			note(pc(ctx,_x), "p=%v, x=%v→%v", ts(p), ts(_x), ts(*x))
			notestack(pc(ctx,_x), 1, "r=%v", ts(*res)).debug(16)
		}
	}

	switch _x.(type) {
	case *builtin, *def, *project, self:
		if s, t := ts(_x), ts(*x); s != t {
			errostack(pc(ctx,p), 3, "%v → %v → %v", s, t, *res).trace()
		}
	}

	if *res == nil {
		if a, y := _x.(*auto); y {
			if d := auto_find(ctx, a.name); d != nil {
				errostack(pc(ctx,p), 10, "%v : %v → %v", ts(p), ts(_x), ts(*x)).trace()
			}
		}
		if !_cl && (x == nil || *x == nil) {
			errostack(pc(ctx,p), 3, "%v : %v → %v", ts(p), ts(_x), ts(*x)).trace()
		}
	}

	if j := _project(ctx); j != nil {
		switch j.name {
		case "configure.base":
			ex_check_configure_base(ctx, p, _x, _a, _o, _l, _cl, res, t, x, a, o)
		}
		switch j.spec {
		case "testdata/value":
			ex_check_value(ctx, p, _x, *x, *res, *o, *a)
		case "testdata/value/2":
			ex_check_value_2(ctx, p, _x, *x, *o, *a, *res)
		case "testdata/value/4":
			ex_check_value_4(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/closure":
			ex_check_value_closure(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/optional":
			ex_check_value_optional(ctx, p, _x, _o, _a, res, x, o, a)
		case "testdata/value/bug_01":
			ex_check_value_bug_01(ctx, p, _x, _o, _a, *res, *x, *o, *a)
		case "testdata/builtins/foreach":
			ex_check_builtins_foreach(ctx, p, _x, _o, _a, *res, *x, *o, *a)
		case "testdata/rule/shell/for-stdout":
			ex_check_rule_shell_forstdout(ctx, p, _x, _o, _a, res, x, a)
		}
	}
}

func ex_check_configure_base(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, res, t, x *Value, a, o *[]Value) {
	if at := ts(auto_get(ctx, "@")); strings.HasPrefix(at, "{=file .configure/library/HAVE_LIB") {
		switch s := p.String(); s {
		case `$(foreach $(INCLUDE),"#include $_\n")`:
			if v := (*res).String(); strings.HasPrefix(v, `$(foreach {},"#include`) {
				note(ctx, "%v → %v ; %v %v %v", p, *res, _cl, truly(ctx, ex_closure{}), truly(ctx, ex_delegate{}))
				erro(pc(ctx,p), "%s : %s → %s ; %v", at, s, v, *a).trace()
			}
		}
	}
	if ent := _entry(ctx); ent != nil {
		switch ent.destiny().string(ctx) {
		case "-compiles-c", "-library-c", "-symbol-c":
			if truly(ctx, is_exec{}) {
				switch p.String() {
				case "$(file $(name).c)", "$(file $(name).c++)", "$(file $(name).log)":
					if _, y := (*res).(*file); !y {
						errostack(ctx, 8, "not a file: %v: %v → %v", p, ts(p), ts(*res)).trace()
					}
				case "$<", "$>", "$(file $(s).x)", "$(file $(s).o)":
					if _, y := (*res).(fullfile); !y {
						errostack(ctx, 8, "not a fullfile: %v: %v → %v", p, ts(p), ts(*res)).trace()
					}
				}
			}
			if truly(ctx, is_modify{}) {
				ex_check_configure_base_library_c(ctx, p, _x, _a, _o, _l, res, a, o)
			}
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

func ex_check_value(ctx Context, p, _x, x, res Value, o, a []Value) {
	switch p.String() {
	case `$(quote a\,b\,c,x\,y\,z)`:
		if s, t := res.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
			erro(ctx, "%s != %s : %s", s, t, ts(res))
			note(ctx, "%v", ts(p))
			note(ctx, "_x=%v", ts(_x))
			note(ctx, " x=%v", ts(x)).trace()
		}
	}
}

func ex_check_value_optional(ctx Context, p, _x Value, _o, _a []Value, _res, x *Value, o, a *[]Value) {
	var res = *_res

	if truly(ctx, propExDef1) {
		switch p.String() {
		case "$(name?)":
			if "{=null}" != ts(res) {
				errostack(pc(ctx,_x), 1, "p=%v, x=%v→%v, r=%v %s", ts(p), ts(_x), ts(*x), ts(res), res).trace()
			}
		}
	}

	switch ps := p.String(); ps {
	case "$(foo)":
		if s, t := ts(res), "{=project foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$(name?)":
		if s, t := ts(res), "{=null}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}→name?)":
		if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}→baz?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$(fo?→bar)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$(fo?→bar?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$({=project foo}→name→xxxx?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$({=project foo}→name→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$({=project foo}?→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
		}
	case "$($_→name)":
		if false {
			if s, v := ts(res), auto_get(ctx, "_"); v == nil {
				if t := "{=delegate {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
				}
			} else if t := "{=self foo}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	case "$($_→name?)":
		if false {
			if s, v := ts(res), auto_get(ctx, "_"); v == nil {
				if t := "{=delegate {=cond {=arrow {=delegate {=auto _}}→{=word name}}}}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
				}
			} else if t := "{=self foo}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s; %v → %v", ps, s, t, _x, *x).trace()
			}
		}
	}

	switch s := _x.String(); s {
	case "$_→name":
		if s, t := ts(_x), "{=arrow {=delegate {=auto _}}→{=word name}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		} else if t := "{=arrow {=project foo}→{=word name}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s; %s", p, s, t, res).trace()
		} else if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s; %s", p, s, t, res).trace()
		}
	case "$_→name?":
		if s, t := ts(_x), "{=cond {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		} else if t := "{=arrow {=project foo}→{=word name}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		}
	case "$_→bar?":
		if s, t := ts(_x), "{=cond {=arrow {=delegate {=auto _}}→{=word bar}}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		} else if s, v := ts(*x), auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		} else if t := "{=arrow {=project foo}→{=word bar}}"; s != t {
			erro(pc(ctx,_x), "%s: %v != %s", p, s, t).trace()
		}
	}
}

func ex_check_value_closure(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, o, a *[]Value) {
	if truly(ctx, ex_closure{}) {
		switch v := *res ; p.String() {
		case "&(foo.pre)":
			if s := do(ctx, get_scope{}); s == nil {
				erro(pc(ctx,p), "%v → %v : nil scope", p, *res).trace()
			}
			if cs := do(ctx, get_closure_scopes{}); cs == nil {
				erro(pc(ctx,p), "%v → %v : nil closure scopes ; %v", p, *res, do(ctx, get_scope{})).trace()
			}
			if cs := closure_scopes(ctx); cs == nil {
				erro(pc(ctx,p), "%v → %v : nil closure scopes ; %v", p, *res, do(ctx, get_closure_scopes{})).trace()
			}
			if cp := closure_projects(ctx); cp == nil {
				erro(pc(ctx,p), "%v → %v : nil closure projects", p, *res).trace()
			}
			if o := closure_resolve(ctx, "foo.pre"); o == nil {
				erro(pc(ctx,p), "%v → %v", p, *res).trace()
			}
			if ts(v) != "{=word foo}" {
				erro(pc(ctx,v), "%v → %v", p, *res).trace()
			}
		case "&(foo.pos)":
			if o := closure_resolve(ctx, "foo.pos"); o == nil {
				erro(pc(ctx,p), "%v → %v", p, *res).trace()
			}
			if ts(v) != "{=word foo}" {
				erro(pc(ctx,v), "%v → %v", p, *res).trace()
			}
		case "&(&(foo.tail))":
			if ts(v) != "{=word foo}" {
				erro(pc(ctx,v), "%v → %v", p, *res).trace()
			}
		}
	}
}

func ex_check_value_2(ctx Context, p, _x, x Value, o, a []Value, res Value) {
	if false && p.String() == "$(string x $(closure &(.test.x)) y $(&(.test.x) aa,bb) c)" {
		note(pc(ctx,_x), "p=%v, x=%v→%v", ts(p), ts(_x), ts(x))
		notestack(pc(ctx,_x), 1, "r=%v", ts(res)).debug(16)
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
			if _project(ctx).def(ctx, ".test.x") == nil {
				if t := ts(*x); s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
				if s, t := ts(*res), "{=closure "+ts(_x)+"}"; s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
			} else if truly(ctx, ex_closure{}) {
				if s, t := ts(*x), "{=def .test.x}"; s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
				if s, t := ts(*res), "{=compound {=punct .} {=word test} {=punct .} {=word v}}"; s != t {
					erro(pc(ctx,p), "%s : %s != %s → %s", p, s, t, *res).trace()
				}
			}
		}
	}
	switch s := p.String() ; s {
	case "&(.test.x)":
		if truly(ctx, ex_closure{}) {
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
		if truly(ctx, ex_closure{}) {
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

func ex_check_builtins_foreach(ctx Context, p, _x Value, _o, _a []Value, res, x Value, o, a []Value) {
	switch p.String() {
	case "&(.test.h)":
		note(ctx, "%v %v %v", p, tv(x), res).debug()
	}
}

func ex_check_value_bug_01(ctx Context, p, _x Value, _o, _a []Value, res, x Value, o, a []Value) {
	switch s := p.String() ; s {
	case "$1":
		if false && truly(ctx, ex_closure{}) {
			x1, x2, x3, x4 := do(ctx, evoke_x{}), do(ctx, evoke_def{}), do(ctx, evoke_def{"bug_0.1"}), do(ctx, evoke_def{"bug_0.2"})
			s1, s2, s3, s4 := ts(x1), ts(x2), ts(x3), ts(x4)
			s0 := res.String()

			if s0 == s && s1 == "{=def 1}" && x1 == x2 && x3 != nil && x4 != nil {
				erro(ctx, "%s %s %s %s", s1, s2, s3, s4).trace() // %s → %s
			}

			if s1 == "{=def 1}" {
				note(ctx, "%s → %s : %s : %s : %s : 1=%s", s, s0, s1, s2, s3, x1.(*def).value).debug()
			} else if xv := x; ts(xv) == "{=def 1}" {
				note(ctx, "%s → %s : %s : %s : %s : x=%s", s, s0, s1, s2, s3, xv.(*def).value).debug()
			} else {
				note(ctx, "%s → %s : %s : %s : %s : %s", s, s0, s1, s2, s3, ts(x)).debug()
			}
		}

	case "&($2.$_)":
		if x, y := do(ctx, evoke_def{}).(*def); y {
			var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
			var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
			switch x.name {
			case ".flag":
				if r, t := res.String(), "&("+v2.String()+".$_)"; r != t {
					erro(ctx, "%s → %s != %s : %v, %v, %v", s, r, t, s1, s2, s3).trace()
				}
			case "bug_0.2":
				if x.value.String() != "$(foreach(-unique) $(foreach $1,&($2.$_)),$_)" {
					erro(ctx, "%v → %v : %v", s, res, x).trace()
				}
				if s2 == s3 {
					if r, t := res.String(), "&("+v2.String()+".$_)"; r != t {
						erro(ctx, "%s → %s != %s : %v", s, r, t, s1).trace()
					}
				} else {
					if t := res.String(); s != t {
						erro(ctx, "%s → %s : %v, %v, %v", s, t, s1, s2, s3).trace()
					}
				}
			}
		} else if _, y := do(ctx, evoke_builtin{"foreach"}).(*builtin); y && false {
			erro(ctx, "%v → %v : %v : %v", s, res, ts(auto_get(ctx, "_")), do(ctx, evoke_def{})).trace()
		} else if s, t := "&($2.{$1})", res.String(); s != t {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if v := auto_get(ctx, "_"); v == nil {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if s, t := "{=disjunction {=delegate {=auto 1}}}", ts(v); s != t {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if false {
			erro(ctx, "%v : %v : %v", s, auto_get(ctx, "_"), ts(do(ctx, evoke_x{}))).trace()
		}

	case "$(foreach $1,$2.$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), s0+".{$1} "+s0+".{$2}"; s == r {
						erro(ctx, "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					} else if r != t {
						erro(ctx, "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					}
					break
				}
			}
		}
		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "$2.{$1}", res.String(); s != t {
				note(ctx, "1: %v → %v", v1, s1)
				note(ctx, "2: %v → %v", v2, s2)
				note(ctx, "3: %v → %v", v3, s3)
				note(ctx, "%v → %v", _a, a)
				errostack(ctx, 8, "%s != %s; %v", s, t, v0).trace()
			}
		}

	case "$(foreach(-unique) $(foreach $1,$2.$_),$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), "{"+s0+".{$1}} {"+s0+".{$2}}"; s == r {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					} else if r != t {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					}
					break
				}
			}
		}
		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "{$2.{$1}}", res.String(); s != t {
				erro(ctx, "%v → %v : %v → %v : %v, %v, %v", s, t, _a, a, s1, s2, s3).trace()
			}
		}

	case "$(foreach(-final) x y z,$(okay.2 $1 $2,$_,$_))":
		var v1, v2 = auto_get(ctx, "1"), auto_get(ctx, "2")
		var s1, s2 = ts(v1), ts(v2)
		if x, y := do(ctx, evoke_def{"okay.1"}).(*def); y {
			if s1 == "{=list {=delegate {=auto 1}}}" && s2 == "{=list {=delegate {=auto 2}}}" {
				erro(ctx, "%v → %v : %v → %v : %v", s, res, _a, a, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v %v : %v → %v", s, x, s1, s2, _a, a).trace()
			}
		} else if s, t := res.String(), "$(okay.2 $1 $2,x,x)? $(okay.2 $1 $2,y,y)? $(okay.2 $1 $2,z,z)?"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
			}
		} else if s, t := ts(res), "{=list {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word x}} {=list {=word x}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word y}} {=list {=word y}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word z}} {=list {=word z}}}}}"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.2 $1 $2,$_,$_)":
		if v := auto_get(ctx, "_"); v == nil {
			if t := res.String(); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		} else {
			if s, t := res.String(), fmt.Sprintf("$(okay.2 $1 $2,%s,%s)", v, v); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.1 $1,$2)":
		if truly(ctx, ex_delegate{}) {
			if s, t := res.String(), "{x.{$1 $2}} {y.{$1 $2}} {z.{$1 $2}}"; s != t {
				erro(ctx, "%v != %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
			} else {
				var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
				if v1 != nil || v2 != nil || v3 != nil {
					erro(ctx, "%v : %v %v %v : %v", s, v1, v2, v2, do(ctx, evoke_x{})).trace()
				}
			}
		} else if t := res.String(); s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
		}
		if s, t := ts(_a), "{=[]Value {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}"; s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
		} else if s := ts(a); s != t {
			erro(ctx, "%v → %v : %v → %v : %v", s, t, _a, a, do(ctx, evoke_x{})).trace()
		}
	}
}

func ex_check_rule_shell_forstdout(ctx Context, p, _x Value, _o, _a []Value, res, x *Value, a *[]Value) {
	var o = try[origin](ctx, get_origin{})

	switch p.String() {
	case "${.test $1,$2}":
		if a1, a2 := auto_get(ctx, "1"), auto_get(ctx, "2") ; a1 != nil && a2 != nil {
			if ts(*a) != sfmt("{=[]Value {=list %s} {=list %s}}", ts(a1), ts(a2)) {
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
		} else if truly(ctx, ex_delegate{}) {
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
		if ts(_a) != "{=[]Value {=list {=delegate {=auto line}} {=delegate {=auto str}}}}" {
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
		case "{=[]Value {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}":
			switch o {
			case defExpand0, defExpand1:
				if ts(*res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			}
		case "{=[]Value {=list {=word b} {=word a}}}":
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
		case "{=[]Value}":
			var t = []Value{auto_get(ctx, "1"), auto_get(ctx, "2")}
			if ts(t) != "{=[]Value {} {}}" {
				errostack(ctx, 5, "%v %v", ts(*x), ts(t)).trace()
			}
			if ts(*res) != "{}" {
				errostack(ctx, 5, "%v", ts(*res)).trace()
			}
		case
			`{=[]Value {=list {=decimal 1} {=strlit test one\n}}}`,
			`{=[]Value {=list {=decimal 2} {=strlit test two\n}}}`,
			`{=[]Value {=list {=decimal 3} {=strlit test thr\n}}}`:
			if ts(*res) != "{}" {
				errostack(ctx, 5, "%v", ts(*res)).trace()
			}
		default:
			errostack(ctx, 5, "untested: %v, %s, %s", o, ts(*a), ts(*res)).trace()
		}
	}

	switch o {
	case defExpand0, defExpand1, 0:
	default:
		errostack(ctx, 5, "untested: %v %s %s", o, ts(_a), ts(*res)).trace()
	}
}

func expand_check_elem(ctx Context, v, w Value) {
	if v == nil {
		erro(ctx, "nil a").trace()
	}
	if w == nil {
		erro(ctx, "nil b").trace()
	}

	a := v.cmp(ctx, w)
	b := w.cmp(ctx, v)
	if a != b {
		note(ctx, "cmp.a: %v", v)
		note(ctx, "cmp.b: %v", w)
		erro(pc(ctx,v), "cmp(%s, %s) → (%v, %v)", typeof(v), typeof(w), a, b).trace()
	}
}

func equal_check(x Context, a, b Value, _res *bool) {
	switch j := _project(x); j.spec {
	case "testdata/value/auto":
		equal_check_value_auto(x, a, b, *_res)
	}
}
func equal_check_value_auto(x Context, a, b Value, res bool) {
	if res {
		if ts(a) != ts(b) {
			erro(pc(x,a), "%v != %v : %v != %v", a, b, ts(a), ts(b)).trace()
		}
	} else {
		if a == nil || b == nil {
			erro(pc(x,a), "%v ⇔ %v", a, b).trace()
		}
		if ts(a) == ts(b) {
			erro(pc(x,a), "%v != %v : %v != %v", a, b, ts(a), ts(b)).trace()
		}
	}
}

func (p *none) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *null) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p negative) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *escaped) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *boolean) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *float) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *integer) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *binary) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *datetime) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *url) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *word) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *arrow) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && v != nil {
		if v == nil {
			errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
		}
		if p.String() == v.String() {
			if x, y := v.(*arrow); y {
				note(ctx, "%v %v, %v", ts(p.o), ts(x.o), p.o.cmp(ctx, x.o))
				note(ctx, "%v %v, %v", ts(p.s), ts(x.s), p.s.cmp(ctx, x.s))
			}
			errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
		}
	}
}

func (p cond) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p disjunction) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && ts(p) == ts(v) {
		errostack(pc(ctx,p), 3, "%v, %s == %s, %s ⇔ %s", res, p, v, ts(p), ts(v)).trace()
	} else if x, y := v.(disjunction); y {
		if p.Value.cmp(ctx, x.Value) != res {
			errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p.Value), ts(x.Value)).trace()
		}
	}
}
func (p disjunction) expand_check(ctx Context, _res *Value) {
	var res = *_res
	if j := _project(ctx); j != nil {
		switch j.spec {
		case "testdata/builtins/foreach":
			switch s0 := p.String(); s0 {
			case "&(.test.h)$_":
				if u := auto_get(ctx, "_"); u == nil {
					erro(pc(ctx,p), "%s : %v", s0, ts(res)).trace()
				} else {
					if s, t := res.String(), "&(.test.h){"+u.String()+"}"; s != t {
						erro(pc(ctx,p), "%s != %s : %v, %v", s, t, ts(u), ts(res)).trace()
					}
				}
			case "{&(.test.h){$1}}":
				if a := auto_get(ctx, "1"); a == nil {
					if s := res.String(); s != s0 {
						erro(pc(ctx,p), "%s != %s : %v", s, s0, ts(res)).trace()
					}
				} else {
					var t = "{"
					for i, v := range merge(a) {
						if 0 < i { t += " " }
						t += "&(.test.h)" + v.String()
					}
					t += "}"
					if s := res.String(); s != t {
						erro(pc(ctx,p), "%s != %s : %v, %v", s, t, a, ts(res)).trace()
					}
				}
			default:
				// note(ctx, "%v %v %v", p, indeterminate(ctx, p.Value), res).debug()
			}
		}
	}
}

func (p *qualword) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *raw) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v; %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
	}
}

func (p *strlit) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *strval) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *globmeta) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *globrange) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *argumented) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *recipe) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *barefile) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(pc(ctx,p), 3, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
	}
}

func (p *arrow) evoke_check(ctx Context, _o, _s, _res *Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/value/optional":
		p.evoke_check_value_optional(ctx, j, *_o, *_s, *_res)
	}
}
func (p *arrow) evoke_check_value_optional(ctx Context, j *project, o, s, res Value) {
	switch ps := p.String(); ps {
	case "foo→name":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word name}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo→name→item":
		if ts(o) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=answer yes}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo→baz":
		if ts(o) != "{=word foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word baz}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=null}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "foo→bar":
		if ts(o) != "{=word foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "fo→bar":
		if ts(o) != "{=word fo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=null}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "fo?→bar":
		if ts(o) != "{=cond {=word fo}}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=cond {=word fo}}→{=word bar}}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "$_→name":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			} else if false {
				note(ctx, "%v %v %v %v", p, o, s, res).debug()
			}
		} else if s, t := "{=project foo}", ts(v); s != t {
			erro(ctx, "%v != %v", s, t).trace()
		} else {
			note(ctx, "%v %v %v %v", p, o, s, res).debug()
		}
	case "$_→bar":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			} else if false {
				note(ctx, "%v %v %v %v", p, o, s, res).debug()
			}
		} else if s, t := "{=project foo}", ts(v); s != t {
			erro(ctx, "%v != %v", s, t).trace()
		} else {
			note(ctx, "%v %v %v %v", p, o, s, res).debug()
		}
	case "{=project foo}→name":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word name}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		} else if false {
			note(pc(ctx,p), "%v %v→%v %v", p, o, s, res).debug(6)
		}
	case "{=project foo}→bar":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "{=project foo}→baz":
		if ts(o) != "{=project foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word baz}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=null}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "{=project foo}→name→xxxx":
		if ts(o) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word xxxx}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if res != nil {
			if ts(res) != "{=null}" {
				erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "{=project foo}→name→item":
		if ts(o) != "{=self foo}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=answer yes}" {
			erro(ctx, "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
	default:
		erro(ctx, "%s: %s; %s→%s; %s", ps, ts(p), ts(o), ts(s), ts(res)).trace()
	}
}

func (p *arrow) expand_check(ctx Context, _o, _s, _res *Value) {
	var res = *_res
	if res == nil { return }

	switch j := _project(ctx); j.spec {
	case "testdata/value/optional":
		p.expand_check_value_optional(ctx, j, *_o, *_s, res)
	}

	if equal(ctx, p, res) {
		if truly(ctx, ex_condless{}) {
			if _, y := res.(cond); y {
				erro(ctx, "%v → %v", p, res).trace()
			}

			var po = p.o; if x, y := po.(cond); y { po = x.Value }
			var ps = p.s; if x, y := ps.(cond); y { ps = x.Value }
			if s, t := ts(*_o), ts(po); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}
			if s, t := ts(*_s), ts(ps); s != t {
				erro(ctx, "%s != %s", s, t).trace()
			}

			if !_cond(p.o) && !_cond(p.s) {
				if s, t := res.String(), p.String(); s != t {
					erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
				}
				if s, t := ts(res), ts(p); s != t {
					erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
				}
			}
		} else {
			if s, t := ts(*_o), ts(p.o); s != t {
				erro(ctx, "%s != %s : %v %v", s, t, _cond(*_o), _cond(p.o)).trace()
			}
			if s, t := ts(*_s), ts(p.s); s != t {
				erro(ctx, "%s != %s : %v %v", s, t, _cond(*_s), _cond(p.s)).trace()
			}
			if s, t := res.String(), p.String(); s != t {
				erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
			}
			if s, t := ts(res), ts(p); s != t {
				erro(ctx, "%v → %v : %s != %s", p, res, s, t).trace()
			}
		}
	} else {
		if s, t := res.String(), p.String(); s == t {
			erro(ctx, "%v → %v : %s == %s", p, res, s, t).trace()
		}
		if s, t := ts(res), ts(p); s == t {
			erro(ctx, "%v → %v : %s == %s", p, res, s, t).trace()
		}
		if false && *_s != nil {
			if *_o != nil {
				if x, y := (*_o).(cond); y {
					erro(ctx, "%v", ts(x)).trace()
				}
			}
			if *_s != nil {
				if x, y := (*_s).(cond); y {
					erro(ctx, "%v", ts(x)).trace()
				}
			}
		}
	}
}
func (p *arrow) expand_check_value_optional(ctx Context, proj *project, o, s, res Value) {
	switch ps := p.String(); ps {
	case "foo→name":
		if ts(o) != "{=word foo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word name}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=word foo}→{=word name}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo→name→item":
		if ts(o) != "{=arrow {=word foo}→{=word name}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=arrow {=word foo}→{=word name}}→{=word item}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo?→name":
		if ts(o) != "{=word foo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word name}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=word foo}→{=word name}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo?→name?→item":
		if ts(o) != "{=arrow {=word foo}→{=word name}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=arrow {=word foo}→{=word name}}→{=word item}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo→baz":
		if ts(o) != "{=word foo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word baz}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=word foo}→{=word baz}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "foo→bar":
		if ts(o) != "{=word foo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=word foo}→{=word bar}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "fo→bar", "fo?→bar":
		if ts(o) != "{=word fo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=word fo}→{=word bar}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "$_→name":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			}
		} else if s0, t := "{=project foo}", ts(v); s0 != t {
			erro(ctx, "%v != %v", s0, t).trace()
		} else {
			if ts(o) != "{=project foo}" {
				erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
			if ts(s) != "{=word name}" {
				erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
			if ts(res) != "{=arrow {=project foo}→{=word name}}" {
				erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "$_→bar":
		if v := auto_get(ctx, "_"); v == nil {
			if o != p.o {
				erro(ctx, "%v != %v", o, p.o).trace()
			}
		} else if s0, t := "{=project foo}", ts(v); s0 != t {
			erro(ctx, "%v != %v", s0, t).trace()
		} else {
			if ts(o) != "{=project foo}" {
				erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
			if ts(s) != "{=word bar}" {
				erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
			if ts(res) != "{=arrow {=project foo}→{=word bar}}" {
				erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
			}
		}
	case "{=project foo}→name":
		if ts(o) != "{=project foo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word name}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=project foo}→{=word name}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "{=project foo}→bar":
		if ts(o) != "{=project foo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word bar}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=project foo}→{=word bar}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "{=project foo}→baz":
		if ts(o) != "{=project foo}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word baz}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=project foo}→{=word baz}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "{=project foo}→name→xxxx":
		if ts(o) != "{=arrow {=project foo}→{=word name}}" {
			erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word xxxx}" {
			erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=arrow {=project foo}→{=word name}}→{=word xxxx}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "{=project foo}→name→item":
		if ts(o) != "{=arrow {=project foo}→{=word name}}" {
			erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=arrow {=project foo}→{=word name}}→{=word item}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	case "{=project foo}→name?→item":
		if ts(o) != "{=arrow {=project foo}→{=word name}}" {
			erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(s) != "{=word item}" {
			erro(pc(ctx,p), "%s; %s→%s; %s", ts(p), ts(o), ts(s), ts(res)).trace()
		}
		if ts(res) != "{=arrow {=arrow {=project foo}→{=word name}}→{=word item}}" {
			erro(pc(ctx,p), "%s: %s; %s→%s; %s", p, ts(p), ts(o), ts(s), ts(res)).trace()
		}
	default:
		erro(ctx, "%v %v %v %v", p, o, s, res).debug()
	}
}

func (p *compound) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && ts(p) == ts(v) {
        note(ctx, "%s, %s", p, ts(p))
        note(ctx, "%s, %s", v, ts(v))
        errostack(pc(ctx,v), 3, "%v, %v ; %v == %v", res, p==v, p, v).trace()
    }
}
func (p *compound) expand_check(ctx Context, _res *Value) {
	var res = *_res
	if res == nil {
		if false { errostack(pc(ctx,p), 3, "%v : %v", p, ts(p)).trace() }
		return
	}

	if j := _project(ctx); j == nil {
		if false { note(pc(ctx,p), "%v %v", p, ts(res)).debug(16) }
	} else {
		switch j.name {
		case "configure.base":
			switch p.String() {
			case "-fautolink$(or $4)":
				if "-fautolinkxar" == res.String() {
					errostack(pc(ctx,p), 3, "%v: %v: %v", j.name, p, ts(res)).trace()
				}
			}
		}
		switch j.spec {
		case "testdata/builtins/foreach":
			if u := auto_get(ctx, "_"); u != nil {
				switch p.String() {
				case "x$_":
					if /* indeterminate(ctx, u) */true {
						if x, y := u.(disjunction); !y {
							erro(pc(ctx,p), "%v %s : %s", p, ts(u), ts(res)).trace()
						} else if s, t := res.String(), "x"+x.String(); s != t {
							erro(pc(ctx,p), "%v → %s != %s : %s", p, s, t, ts(res)).trace()
						}
					}
					switch u.String() {
					case "{&(.test.h){$1}}":
						if s, t := res.String(), "x{&(.test.h){$1}}"; s != t {
							erro(pc(ctx,p), "%v → %s != %s : %s", p, s, t, ts(res)).trace()
						}
					}
				}
			}
		}
	}

	if /* false && p.expandable(ctx) && */ equal(ctx, p, res) {
		if s := p.String(); strings.Contains(s, "$_") {
			if r := res.String(); res == p || r == s || strings.Contains(r, "$_") {
				if d := auto_find(ctx, "_"); d != nil {
					note(ctx, "%v", ts(d))
					note(ctx, "%v → %v", ts(p), ts(res))
					erro(ctx, "%v", ts(ctx)).trace()
				}
			}
		}
	}
}

func (p cond) expand_check(ctx Context, v Value, _res *Value) {
	res := *_res
	j := _project(ctx)
	if j.spec == "testdata/value/optional" {
		if a, y := p.Value.(*arrow); y && false {
			if j, y := a.o.(*project); y && j.name == "foo" && a.s.String() == "name" {
				if d, y := v.(*def); !y {
					erro(ctx, "%v → %v → %v", p, ts(v), ts(res)).trace()
				} else if ts(d.value) != "{=self foo}" {
					erro(ctx, "%v → %v", p, ts(d.value)).trace()
				}
			}
		}
		if s1 := p.String(); s1 == "name?" {
			if s2 := v.String(); s1 == s2 {
				erro(pc(ctx,p), "%s != %s; res=%s; %s", s2, s1, res, ts(p.Value)).trace()
			} else if s2 != "name" {
				erro(pc(ctx,p), "%s != %s; v=%s; %s", s2, s1, v, ts(p.Value)).trace()
			}
			if s2 := res.String(); s1 == s2 {
				erro(pc(ctx,p), "%s != %s; v=%s; %s", s2, s1, v, ts(p.Value)).trace()
			} else if s2 != "name" {
				erro(pc(ctx,p), "%s != %s; res=%s; %s", s2, s1, res, ts(p.Value)).trace()
			}
		}
		if s1 := p.String(); s1 == "foo→name?" {
			if s2 := v.String(); s1 != s2 {
				erro(pc(ctx,p), "%s != %s; res=%s; %s", s2, s1, res, ts(p.Value)).trace()
			}
			if s2 := res.String(); s1 != s2 {
				erro(pc(ctx,p), "%s != %s; v=%s; %s", s2, s1, v, ts(p.Value)).trace()
			}
		}
		if s1 := p.String(); s1 == "fo?→bar" {
			if s2 := v.String(); s1 != s2 {
				erro(pc(ctx,p), "%s != %s; res=%s; %s", s2, s1, res, ts(p.Value)).trace()
			}
			if s2 := res.String(); s1 != s2 {
				erro(pc(ctx,p), "%s != %s; v=%s; %s", s2, s1, v, ts(p.Value)).trace()
			}
		}
		if s1 := p.String(); s1 == "fo?→bar?" {
			if s1, s2 := "fo→bar", v.String(); s1 != s2 {
				erro(pc(ctx,p), "%s != %s; res=%s; %s", s2, s1, res, ts(p.Value)).trace()
			}
			if s1, s2 := "fo→bar", res.String(); s1 != s2 {
				erro(pc(ctx,p), "%s != %s; v=%s; %s", s2, s1, v, ts(p.Value)).trace()
			}
		}
		if s1 := ts(p); s1 == fmt.Sprintf("{=cond {=word %s}}", v) {
			if s2 := ts(v); s2 != fmt.Sprintf("{=word %s}", v) {
				erro(pc(ctx,p), "%s != %s ; %s", s1, s2, ts(res)).trace()
			}
			if s2 := ts(res); false && s1 != s2 {
				erro(pc(ctx,p), "%s != %s ; %s", s1, s2, ts(res)).trace()
			}
		}
		if s, t1 := "{=arrow {=project foo}→{=word name}}", ts(p.Value); s == t1 && false {
			if s, t2 := "{=def name}", ts(res); s != t2 {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			} else if d, y := res.(*def); !y {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			} else if s, t2 := "{=self foo}", ts(d.value); s != t2 {
				erro(pc(ctx,p), "%v: %v → %v", p, t1, t2).trace()
			}
		}
	}
	if v == nil {
		if res == nil {
			erro(ctx, "%v", p).trace()
			return
		}
		if !isNull(res) {
			erro(pc(ctx,res), "%v → %v", p, ts(res)).trace()
			return
		}
	} else if !truly(ctx, ex_closure{}) {
		if !equal(ctx, v, p.Value) && p.Value.String() == v.String() {
			note(ctx, "%v → %v → %v", p.Value, v, res)
			note(ctx, "%-20v : %v", p.Value, ts(p.Value))
			note(ctx, "%-20v : %v", v,       ts(v))
			note(ctx, "%-20v : %v", res,     ts(res))
			errostack(pc(ctx,res), 3, "%v", p).trace()
		}
	}
	if _cond(v) {
		note(ctx, "%v → %v → %v", p.Value, v, res)
		note(ctx, "%-20v : %v", p.Value, ts(p.Value))
		note(ctx, "%-20v : %v", v,       ts(v))
		note(ctx, "%-20v : %v", res,     ts(res))
		errostack(pc(ctx,res), 3, "%v", p).trace()
	}
}

func (p *path) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
        note(ctx, "%v, %v, %v", p, p==v, res)
        note(ctx, "%v", ts(p))
        note(ctx, "%v", ts(v))
        errostack(ctx, 3, "%v", ts(ctx)).trace()
    }
}
func (p *path) match2_check(ctx Context, srcs []string, full *bool, res, stems *[]string) {
	var s, t = joinpath(srcs...), p.string(ctx)

	if strings.HasPrefix(s, t) && *res == nil {
		note(ctx, "%v →", p)
		note(ctx, "%v →", s)
		note(ctx, "→ %v %v %v", full, res, stems)
		errostack(ctx, 3, "%v", ts(ctx)).trace()
	}

	switch t {
	case "%%/.smart/modules/":
		if strings.Contains(s, "/.smart/modules/") {
			if a := *res; len(a) < 4 || a[len(a)-1] != "" {
				errostack(ctx, 3, "%v : %v %v %v", s, *full, *res, *stems).trace()
			}
		}
	}
	return
}

func (p *strcomp) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(ctx, 3, "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
	}
}

func (p *punct) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(ctx, 3, "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
    }
}

func (p fullname) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
		errostack(ctx, 3, "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
    }
}

func (p *list) cmp_check(ctx Context, v Value, _res *cmpres) {
	if res := *_res; res != cmpEqual {
		if s1, s2 := ts(p), ts(v) ; s1 == s2 {
			errostack(ctx, 3, "%v, %v ⇔ %v", res, s1, s2).trace()
		} else if false && p.String() == v.String() {
			errostack(ctx, 3, "%v, %v ⇔ %v", res, s1, s2).trace()
		}
	}
	return
}
func (p *list) expand_check(ctx Context, d bool, elems []Value, _res *Value) {
	if j := _project(ctx); j != nil {
		switch j.spec {
		case "testdata/value/auto":
			p.expand_check_value_auto(ctx, d, elems, *_res)
		case "testdata/value/placeholder":
			p.expand_check_value_placeholder(ctx, d, elems, *_res)
		}
	}

	var res = *_res
	var s1 = ts(p.elems)
	var s2 = ts(elems)
	if d {
		if s1 == s2 {
			errostack(pc(ctx,p), 3, "%s == %s", s1, s2).trace()
		}
		if p == res || p.String() == res.String() {
			errostack(pc(ctx,p), 3, "%v == %v; %v", p, res, p==res).trace()
		}
	} else {
		if s1 != s2 {
			errostack(pc(ctx,p), 3, "%s != %s", s1, s2).trace()
		}
		if p != res || p.String() != res.String() {
			errostack(pc(ctx,p), 3, "%v != %v; %v", p, res, p!=res).trace()
		}
	}
	return
}
func (p *list) expand_check_value_auto(ctx Context, d bool, elems []Value, res Value) {
	switch p.String() {
	case "$1 $2 $3 $4 $5 $6 $7 $8 $9":
		var t string
		for i := 1; i <= 9; i += 1 {
			if 1 < i { t += " " }
			if a := auto_get(ctx, strconv.Itoa(i)); a != nil {
				t += a.String()
			} else {
				t += fmt.Sprintf("$%d", i)
			}
		}
		if s := res.String(); s != t {
			errostack(pc(ctx,p), 3, "%s != %s : %s", s, t, ts(res)).trace()
		}
	}
}
func (p *list) expand_check_value_placeholder(ctx Context, d bool, elems []Value, res Value) {
	switch p.String() {
	case "$_":
		if a := auto_get(ctx, "_"); a == nil {
			erro(ctx, "%v %v", do(ctx, evoke_x{}), res).debug()
		} else if s, t := res.String(), a.String(); s != t {
			erro(ctx, "%s != %s : %s", s, t, ts(res)).debug()
		}
	}
}

func (p *delegate) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && /* p.String() == v.String() */ts(p) == ts(v) {
        erro(ctx, "%v, %v ⇔ %v, %v ⇔ %v", res, p, v, ts(p), ts(v)).trace()
    }
}
func (p *delegate) expand_check(ctx *uvc, x Value, a *[]Value, v, _res *Value) {
	if j := _project(ctx); j != nil {
		switch j.name {
		case "configure.base":
			p.expand_check_configure_base(ctx, *_res)
		}
		switch j.spec {
		case "testdata/value":
			p.expand_check_value(ctx, j, *a, *v, *_res)
		case "testdata/value/auto":
			p.expand_check_value_auto(ctx, j, *a, *v, *_res)
		case "testdata/value/placeholder":
			p.expand_check_value_placeholder(ctx, j, *a, *v, *_res)
		case "testdata/value/2":
			p.expand_check_value_2(ctx, *_res)
		case "testdata/value/4":
			p.expand_check_value_4(ctx, *_res)
		case "testdata/value/closure":
			p.expand_check_value_closure(ctx, *_res)
		case "testdata/value/optional":
			p.expand_check_value_optional(ctx, *_res)
		case "testdata/value/bug_01":
			p.expand_check_value_bug_01(ctx, *_res)
		case "testdata/builtins/foreach":
			p.expand_check_builtins_foreach(ctx, *_res)
		case "testdata/rule/shell/for-stdout":
			p.expand_check_rule_shell_forstdout(ctx, *_res)
		}
	}
}
func (p *delegate) expand_check_configure_base(ctx Context, res Value) {
	if at := ts(auto_get(ctx, "@")); strings.HasPrefix(at, "{=file .configure/library/HAVE_LIB") {
		switch s := p.String(); s {
		case `$(foreach $(INCLUDE),"#include $_\n")`:
			if v := res.String(); strings.HasPrefix(v, `$(foreach {},"#include`) {
				note(ctx, "%v → %v ; %v %v", p, res, truly(ctx, ex_closure{}), truly(ctx, ex_delegate{}))
				erro(pc(ctx,p), "%s : %s → %s", at, s, v).trace()
			}
		}
	}
	if ent := _entry(ctx); ent != nil {
		switch ent.destiny().string(ctx) {
		case "-compiles-c", "-library-c", "-symbol-c":
			if truly(ctx, is_exec{}) {
				switch p.String() {
				case "$(file $(name).c)", "$(file $(name).c++)", "$(file $(name).log)":
					if _, y := res.(*file); !y {
						errostack(ctx, 8, "not a file: %v: %v → %v", p, ts(p), ts(res)).trace()
					}
				case "$<", "$>", "$(file $(s).x)", "$(file $(s).o)":
					if _, y := res.(fullfile); !y {
						errostack(ctx, 8, "not a fullfile: %v: %v → %v", p, ts(p), ts(res)).trace()
					}
				}
			}
			if truly(ctx, is_modify{}) {
				p.expand_check_configure_base_library_c(ctx, res)
			}
		}
	}
}
func (p *delegate) expand_check_configure_base_library_c(ctx Context, res Value) {
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
		if (res).String() != kind {
			erro(ctx, "%v", res).trace()
		}
	case "$(file .configure/$(ifdef FUNCTION,function,library)/$(TARGET).c)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v", t.value).trace()
		} else if x, y := (res).(*file); !y {
			erro(ctx, "%v", res).trace()
		} else if t := filepath.Join(".configure", kind, s+".c"); t != x.name {
			erro(ctx, "%s != %s", x.name, t).trace()
		}
	case "$(file $(s).x)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v %v", t.value, s).trace()
		} else {
			if x, y := (res).(*file); !y {
				erro(ctx, "%v %v", typeof(res), res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.x"); t != x.name {
				erro(ctx, "%s != %s", x.name, t).trace()
			}
		}
	case "$(file $(s).o)":
		if t == nil || isTrivial(t.value) {
			erro(ctx, "%v", res).trace()
		} else if s := t.string(ctx); s == "" {
			erro(ctx, "%v %v", t.value, s).trace()
		} else {
			if x, y := (res).(*file); !y {
				erro(ctx, "%v %v", typeof(res), res).trace()
			} else if t := filepath.Join(".configure", kind, s+".c.o"); t != x.name {
				erro(ctx, "%s != %s", x.name, t).trace()
			}
		}
	}
}
func (p *delegate) expand_check_value(ctx *uvc, j *project, a []Value, v, res Value) {
	switch p.String() {
	case `$(quote a\,b\,c,x\,y\,z)`:
		if s, t := res.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
			erro(ctx, "%s != %s : %s", s, t, ts(res))
			note(ctx, "%v", ts(p)).trace()
		}
	case fmt.Sprintf(`$(grep {=regex ^.+?\.o$$},$0,%s/test.txt)`, j.absPath):
		if ctx.uv != nil {
			erro(pc(ctx,p.x), "%v → %v : %v", p, v, tv(ctx.uv)).trace()
		}
		note(ctx, "%v", p)
		note(ctx, "%v", res).debug(3)
	}
}
func (p *delegate) expand_check_value_auto(ctx *uvc, j *project, a []Value, v, res Value) {
	var ps = p.String()
	switch ps {
	case "$(closure foobar)":
		if s, t := res.String(), "&(foobar)"; s != t {
			erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
		}
		if s, t := ts(res), "{=closure {=word foobar}}"; s != t {
			erro(pc(ctx,p.x), "%s != %s", s, t).trace()
		}
	case "$(auto,$(a))":
		if o := do(ctx, evoke_def{"val1"}); o == nil {
			if s, t := res.String(), "$(a)"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=delegate {=auto a}}"; s != t {
				erro(pc(ctx,p.x), "%s != %s", s, t).trace()
			}
		} else {
			if s, t := res.String(), "2"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "2"; s != t {
				erro(pc(ctx,p.x), "%s != %s", s, t).trace()
			}
		}
	case "$(auto a=1,$(val1),$(a))":
		if o := do(ctx, evoke_def{"val2"}); o == nil {
			if s, t := res.String(), "2 2"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "2 2"; s != t {
				erro(pc(ctx,p.x), "%s != %s", s, t).trace()
			}
		} else {
			if s, t := res.String(), "3 3"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "3 3"; s != t {
				erro(pc(ctx,p.x), "%s != %s", s, t).trace()
			}
		}
	case "$(auto a=1,$(val2))":
		if o := do(ctx, evoke_def{"val3"}); true {
			if s, t := res.String(), "3 3"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s ; %v", s, t, ts(res), o).trace()
			}
			if s, t := ts(res), "3 3"; s != t {
				erro(pc(ctx,p.x), "%s != %s ; %v", s, t, o).trace()
			}
		}
	case "$(val3)":
		if o := do(ctx, evoke_def{"val4"}); o == nil {
			if s, t := res.String(), "2 2"; s != t {
				erro(pc(ctx,p.x), "%s != %s : %s", s, t, ts(res)).trace()
			}
			if s, t := ts(res), "{=list {=decimal 2} {=decimal 2}}"; s != t {
				erro(pc(ctx,p.x), "%s != %s", s, t).trace()
			}
		} else {
			erro(ctx, "%v → %v; %v", p, res, o).debug()
		}
	default:
		for i := 1; i <= 9; i += 1 {
			if s := strconv.Itoa(i); ps == "$"+s {
				if t := auto_get(ctx, s); t != nil {
					if res.cmp(ctx, t) != cmpEqual {
						erro(pc(ctx,p.x), "%d, %s != %s", i, res, t).trace()
					}
				}
			}
		}
	}
}
func (p *delegate) expand_check_value_placeholder(ctx *uvc, j *project, a []Value, v, res Value) {
	var ps = p.String()
	switch ps {
	case "$(foreach a b c d e f,$_)":
		if s, t := res.String(), "a b c d e f"; s != t {
			erro(ctx, "%s != %s : %s", s, t, ts(res)).trace()
		}
	case "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)":
		if s, t := res.String(), "{$1} {$2} {$3} {$4} {$5} {$6} {$7} {$8} {$9}"; s != t {
			erro(ctx, "%s != %s : %s", s, t, ts(res)).trace()
		}
	}
}
func (p *delegate) expand_check_value_2(ctx Context, res Value) {
}
func (p *delegate) expand_check_value_4(ctx Context, res Value) {
}
func (p *delegate) expand_check_value_closure(ctx Context, res Value) {
}
func (p *delegate) expand_check_value_optional(ctx Context, res Value) {
	if truly(ctx, propExDef1) {
		switch p.String() {
		case "$(name?)":
			if "{=null}" != ts(res) {
				errostack(pc(ctx,p.x), 1, "p=%v, r=%v %s", ts(p), ts(res), res).trace()
			}
		}
	}

	switch ps := p.String(); ps {
	case "$(foo)":
		if s, t := ts(res), "{=project foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$(name?)":
		if s, t := ts(res), "{=null}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}→name?)":
		if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}→baz?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$(fo?→bar)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$(fo?→bar?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$({=project foo}→name→xxxx?)":
		if res != nil {
			if s, t := ts(res), "{=null}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$({=project foo}→name→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$({=project foo}?→name?→item?)":
		if s, t := ts(res), "{=answer yes}"; s != t {
			erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
		}
	case "$($_→name)":
		if false {
			if s, v := ts(res), auto_get(ctx, "_"); v == nil {
				if t := "{=delegate {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
				}
			} else if t := "{=self foo}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	case "$($_→name?)":
		if false {
			if s, v := ts(res), auto_get(ctx, "_"); v == nil {
				if t := "{=delegate {=cond {=arrow {=delegate {=auto _}}→{=word name}}}}"; s != t {
					erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
				}
			} else if t := "{=self foo}"; s != t {
				erro(pc(ctx,res), "%s: %s != %s", ps, s, t).trace()
			}
		}
	}

	switch s := p.x.String(); s {
	case "$_→name":
		if s, t := ts(p.x), "{=arrow {=delegate {=auto _}}→{=word name}}"; s != t {
			erro(pc(ctx,p.x), "%s: %v != %s", p, s, t).trace()
		} else if s, t := ts(res), "{=self foo}"; s != t {
			erro(pc(ctx,p.x), "%s: %v != %s; %s", p, s, t, res).trace()
		}
	case "$_→name?":
		if s, t := ts(p.x), "{=cond {=arrow {=delegate {=auto _}}→{=word name}}}"; s != t {
			erro(pc(ctx,p.x), "%s: %v != %s", p, s, t).trace()
		} else if v := auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		}
	case "$_→bar?":
		if s, t := ts(p.x), "{=cond {=arrow {=delegate {=auto _}}→{=word bar}}}"; s != t {
			erro(pc(ctx,p.x), "%s: %v != %s", p, s, t).trace()
		} else if v := auto_get(ctx, "_"); v == nil {
			if s != t { erro(ctx, "%s: %s != %s", p, s, t).trace() }
		}
	}
}
func (p *delegate) expand_check_value_bug_01(ctx Context, res Value) {
	switch s := p.String() ; s {
	case "$1":
	case "$(foreach $1,$2.$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), s0+".{$1} "+s0+".{$2}"; s == r {
						erro(ctx, "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					} else if r != t {
						erro(ctx, "%v → %v != %v; %v, %v, %v", s, r, t, v1, v2, v3).trace()
					}
					break
				}
			}
		}
		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "$2.{$1}", res.String(); s != t {
				note(ctx, "1: %v → %v", v1, s1)
				note(ctx, "2: %v → %v", v2, s2)
				note(ctx, "3: %v → %v", v3, s3)
				errostack(ctx, 8, "%s != %s; %v", s, t, v0).trace()
			}
		}

	case "$(foreach(-unique) $(foreach $1,$2.$_),$_)":
		var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
		var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
		var s0 string
		if s1 == "{=list {=delegate {=auto 1}} {=delegate {=auto 2}}}" {
			for _, s0 = range []string{"x", "y", "z"} {
				if s2 == "{=word "+s0+"}" && s3 == "{=word "+s0+"}" {
					if r, t := res.String(), "{"+s0+".{$1}} {"+s0+".{$2}}"; s == r {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					} else if r != t {
						errostack(pc(ctx,p), 3, "%v → %v != %v", s, r, t).trace()
					}
					break
				}
			}
		}
		if v0 := auto_get(ctx, "_"); v0 == nil && s0 == "" {
			if s, t := "{$2.{$1}}", res.String(); s != t {
				erro(ctx, "%v != %v : %v, %v, %v", s, t, s1, s2, s3).trace()
			}
		}

	case "$(foreach(-final) x y z,$(okay.2 $1 $2,$_,$_))":
		var v1, v2 = auto_get(ctx, "1"), auto_get(ctx, "2")
		var s1, s2 = ts(v1), ts(v2)
		if x, y := do(ctx, evoke_def{"okay.1"}).(*def); y {
			if s1 == "{=list {=delegate {=auto 1}}}" && s2 == "{=list {=delegate {=auto 2}}}" {
				erro(ctx, "%v → %v : %v", s, res, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v %v", s, x, s1, s2).trace()
			}
		} else if s, t := res.String(), "$(okay.2 $1 $2,x,x)? $(okay.2 $1 $2,y,y)? $(okay.2 $1 $2,z,z)?"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		} else if s, t := ts(res), "{=list {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word x}} {=list {=word x}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word y}} {=list {=word y}}}} {=condval {=delegate {=def okay.2} {=list {=delegate {=auto 1}} {=delegate {=auto 2}}} {=list {=word z}} {=list {=word z}}}}}"; s != t {
			if s1 != "{}" || s2 != "{}" {
				erro(ctx, "%v → %v : %v %v : %v", s, t, s1, s2, do(ctx, evoke_x{})).trace()
			} else {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.2 $1 $2,$_,$_)":
		if v := auto_get(ctx, "_"); v == nil {
			if t := res.String(); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		} else {
			if s, t := res.String(), fmt.Sprintf("$(okay.2 $1 $2,%s,%s)", v, v); s != t {
				erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
			}
		}

	case "$(okay.1 $1,$2)":
		if truly(ctx, ex_delegate{}) {
			if s, t := res.String(), "{x.{$1 $2}} {y.{$1 $2}} {z.{$1 $2}}"; s != t {
				erro(ctx, "%v != %v : %v", s, t, do(ctx, evoke_x{})).trace()
			} else {
				var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
				if v1 != nil || v2 != nil || v3 != nil {
					erro(ctx, "%v : %v %v %v : %v", s, v1, v2, v2, do(ctx, evoke_x{})).trace()
				}
			}
		} else if t := res.String(); s != t {
			erro(ctx, "%v → %v : %v", s, t, do(ctx, evoke_x{})).trace()
		}
	}
}
func (p *delegate) expand_check_builtins_foreach(ctx Context, res Value) {
}
func (p *delegate) expand_check_rule_shell_forstdout(ctx Context, res Value) {
	var o = try[origin](ctx, get_origin{})

	switch p.String() {
	case "${.test $1,$2}":
		if a1, a2 := auto_get(ctx, "1"), auto_get(ctx, "2") ; a1 != nil && a2 != nil {
			switch o {
			case defExpand0:
				if ts(res) != sfmt("{=delegate {=builtin debug} {=list %s %s}}", ts(a1), ts(a2)) {
					errostack(ctx, 3, "%s %s, %s", ts(a1), ts(a2), ts(res)).trace()
				}
			case defExpand1:
				if ts(res) != "{=null}" {
					errostack(ctx, 3, "%s", ts(res)).trace()
				}
			}
		} else if truly(ctx, ex_delegate{}) {
			if ts(res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
				errostack(ctx, 3, "%v: %s", p, ts(res)).trace()
			}
		} else {
			erro(ctx, "%v: %s %s ; %s", p, ts(a1), ts(a2), ts(res)).trace()
		}
	case "${.test a,b}":
		switch o {
		case defExpand0:
			if ts(res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
				errostack(ctx, 3, "%v", ts(res)).trace()
			}
		case defExpand1:
			if ts(res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(res)).trace()
			}
		}
	case "$(.test.v3 a,b)":
		switch o {
		case defExpand1:
			if ts(res) != "{=null}" {
				errostack(ctx, 3, "%s", ts(res)).trace()
			}
		}
	case "$(debug $(line) $(str))":
		if ts(p.a) != "{=[]Value {=list {=delegate {=auto line}} {=delegate {=auto str}}}}" {
			erro(ctx, "%v", ts(p.a)).trace()
		}

		if a := _automatic(ctx); a == nil {
			errostack(ctx, 5, "%v", ts(res)).trace()
		} else {
			keys := reflect.ValueOf(a.defs).MapKeys()

			if x1, y := a.defs["1"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x2, y := a.defs["str"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(res)).trace()
			}

			if x1, y := a.defs["2"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x2, y := a.defs["line"]; !y {
				errostack(ctx, 5, "%v; %v", keys, ts(res)).trace()
			} else if x1 != x2 {
				errostack(ctx, 5, "%v != %v; %v", x1, x2, ts(res)).trace()
			}
		}
		if a, b := auto_get(ctx, "1"), auto_get(ctx, "str"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(res)).trace()
		}
		if a, b := auto_get(ctx, "2"), auto_get(ctx, "line"); a != b {
			errostack(ctx, 5, "%v != %v; %v", a, b, ts(res)).trace()
		}

		switch ts(p.a) {
		case "{=[]Value {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}":
			switch o {
			case defExpand0, defExpand1:
				if ts(res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
					errostack(ctx, 5, "%v", ts(res)).trace()
				}
			}
		case "{=[]Value {=list {=word b} {=word a}}}":
			switch o {
			case defExpand0:
				if ts(res) != "{=delegate {=builtin debug} {=list {=word b} {=word a}}}" {
					errostack(ctx, 5, "%v", ts(res)).trace()
				}
			case defExpand1:
				if ts(res) != "{}" {
					errostack(ctx, 5, "%v", ts(res)).trace()
				}
			}
		case "{=[]Value}":
			var t = []Value{auto_get(ctx, "1"), auto_get(ctx, "2")}
			if ts(t) != "{=[]Value {} {}}" {
				errostack(ctx, 5, "%v %v", ts(p.x), ts(t)).trace()
			}
			if ts(res) != "{}" {
				errostack(ctx, 5, "%v", ts(res)).trace()
			}
		case
			`{=[]Value {=list {=decimal 1} {=strlit test one\n}}}`,
			`{=[]Value {=list {=decimal 2} {=strlit test two\n}}}`,
			`{=[]Value {=list {=decimal 3} {=strlit test thr\n}}}`:
			if ts(res) != "{}" {
				errostack(ctx, 5, "%v", ts(res)).trace()
			}
		default:
			errostack(ctx, 5, "untested: %v, %s", o, ts(res)).trace()
		}
	}

	switch o {
	case defExpand0, defExpand1, 0:
	default:
		errostack(ctx, 5, "untested: %v %s", o, ts(res)).trace()
	}
}

func (p *closure) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}
func (p *closure) expand_check(ctx *uvc, x Value, _res *Value) {
	if j := _project(ctx); j != nil {
		switch j.name {
		case "configure.base":
			p.expand_check_configure_base(ctx, x, *_res)
		}
		switch j.spec {
		case "testdata/value":
			p.expand_check_value(ctx, x, *_res)
		case "testdata/value/2":
			p.expand_check_value_2(ctx, x, *_res)
		case "testdata/value/4":
			p.expand_check_value_4(ctx, x, *_res)
		case "testdata/value/closure":
			p.expand_check_value_closure(ctx, x, *_res)
		case "testdata/value/optional":
			p.expand_check_value_optional(ctx, x, *_res)
		case "testdata/value/bug_01":
			p.expand_check_value_bug_01(ctx, x, *_res)
		case "testdata/builtins/foreach":
			p.expand_check_builtins_foreach(ctx, x, *_res)
		case "testdata/rule/shell/for-stdout":
			p.expand_check_rule_shell_forstdout(ctx, x, *_res)
		}
	}
}
func (p *closure) expand_check_configure_base(ctx Context, x, res Value) {
}
func (p *closure) expand_check_value(ctx Context, x, res Value) {
}
func (p *closure) expand_check_value_2(ctx Context, x, res Value) {
}
func (p *closure) expand_check_value_4(ctx Context, x, res Value) {
}
func (p *closure) expand_check_value_closure(ctx Context, x, res Value) {
	switch p.String() {
	case "&(&(foo.tail))":
		if _, y := x.(*closure); y {
			if c := cast[*uvc](ctx); c != nil && !c.indeterminate() {
				erro(pc(ctx,x), "not indeterminate : %s", ts(x)).trace()
			}
		}
	}
}
func (p *closure) expand_check_value_optional(ctx Context, x, res Value) {
}
func (p *closure) expand_check_value_bug_01(ctx Context, x, res Value) {
	switch s := p.String() ; s {
	case "&($2.$_)":
		if x, y := do(ctx, evoke_def{}).(*def); y {
			var v1, v2, v3 = auto_get(ctx, "1"), auto_get(ctx, "2"), auto_get(ctx, "3")
			var s1, s2, s3 = ts(v1), ts(v2), ts(v3)
			switch x.name {
			case ".flag":
				if r, t := res.String(), "&("+v2.String()+".$_)"; r != t {
					erro(ctx, "%s → %s != %s : %v, %v, %v", s, r, t, s1, s2, s3).trace()
				}
			case "bug_0.2":
				if x.value.String() != "$(foreach(-unique) $(foreach $1,&($2.$_)),$_)" {
					erro(ctx, "%v → %v : %v", s, res, x).trace()
				}
				if s2 == s3 {
					if r, t := res.String(), "&("+v2.String()+".$_)"; r != t {
						erro(ctx, "%s → %s != %s : %v", s, r, t, s1).trace()
					}
				} else {
					if t := res.String(); s != t {
						erro(ctx, "%s → %s : %v, %v, %v", s, t, s1, s2, s3).trace()
					}
				}
			}
		} else if _, y := do(ctx, evoke_builtin{"foreach"}).(*builtin); y && false {
			erro(ctx, "%v → %v : %v : %v", s, res, ts(auto_get(ctx, "_")), do(ctx, evoke_def{})).trace()
		} else if s, t := "&($2.{$1})", res.String(); s != t {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if v := auto_get(ctx, "_"); v == nil {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if s, t := "{=disjunction {=delegate {=auto 1}}}", ts(v); s != t {
			erro(ctx, "%v → %v : %v : %v", s, t, ts(auto_get(ctx, "_")), ts(do(ctx, evoke_x{}))).trace()
		} else if false {
			erro(ctx, "%v : %v : %v", s, auto_get(ctx, "_"), ts(do(ctx, evoke_x{}))).trace()
		}
	}
}
func (p *closure) expand_check_builtins_foreach(ctx Context, x, res Value) {
	switch p.String() {
	case "&(.test.h)":
		note(ctx, "%v %v %v", p, tv(p.x), res).debug()
	}
}
func (p *closure) expand_check_rule_shell_forstdout(ctx Context, x, res Value) {
	var o = try[origin](ctx, get_origin{})
	switch o {
	case defExpand0, defExpand1, 0:
	default:
		errostack(ctx, 5, "untested: %v %s", o, ts(res)).trace()
	}
}

func (p *globpat) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}

func (p *regexpat) cmp_check(ctx Context, v Value, _res *cmpres) {
    if res := *_res; res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
    }
}

func (p *project) unmap_files_check(ctx Context, _k any, res *[]filemap_name) {
	switch p.name {
	case "configure.base":
		if x, y := _k.(Value); y && *res == nil && truly(ctx, is_modify{}) {
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
			if d := p.def(ctx, "srcinc"); d == nil {
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
			if d := p.def(ctx, "outinc"); d == nil {
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
	if f := *_res; f != nil {
		switch p := _project(ctx); f.name {
		case "llvm/Config/llvm-config.h.cmake":
			s := p.def(ctx, "srcinc").string(ctx)
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
			s := p.def(ctx, "outinc").string(ctx)
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
}

func select_files_check(ctx Context, m []filemap_name, res *[]*file) {
	if *res == nil { return }

	var p = _project(ctx)

	switch f := (*res)[0]; f.name {
	case "llvm/Config/llvm-config.h.cmake":
		if s := p.def(ctx, "srcinc").string(ctx); f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		}
	case "llvm/Config/llvm-config.h":
		if s := p.def(ctx, "outinc").string(ctx); f.dir != s {
			erro(ctx, "%s != %s", f.dir, s).trace()
		}
	}
}

func (a as) file_check(ctx Context, projs []*project, v Value, f **file) {
	if len(projs) == 0 { projs = []*project{ _project(ctx) } }

	var p = projs[0]

	if *f == nil {
		var s = v.string(ctx)
		if s == "" {
			return // note(ctx, "as.file %v", ts(v)).debug()
		}
		if f := findfile(ctx, s, projs...); f != nil {
			for _, m := range unmap_files(ctx, f) {
				erro(ctx, "FIXME: %v (%s) ⇒ %v", v, s, m)
			}
			erro(ctx, "FIXME: %v (%s) ⇒ %v (%s)", v, s, f, f.fullname())
			errostack(ctx, 5).trace()
		}
	} else if (*f).name == "llvm/Config/llvm-config.h.cmake" {
		if s := p.def(ctx, "srcinc").string(ctx); (*f).dir != s {
			erro(ctx, "%s != %s", (*f).dir, s).trace()
		}
	} else if (*f).name == "llvm/Config/llvm-config.h" {
		if s := p.def(ctx, "outinc").string(ctx); (*f).dir != s {
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
				if s := p.def(ctx, "srcinc").string(ctx); x.dir != s {
					erro(ctx, "%s != %s", x.dir, s).trace()
				}
			} else if x.name == "llvm/Config/llvm-config.h" {
				if s := p.def(ctx, "outinc").string(ctx); x.dir != s {
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
				erro(ctx, "%v: %v != %v", p.name, (res), v).trace()
			}
		} else if cacheMapping(ctx) {
			erro(ctx, "%v: %v", p.name, v).trace()
		}
	}
}

func (p *builtin) evoke_check(ctx *evocation, res *Value) {
	var j = _project(ctx)

	switch p.name {
	case "print":
	case "grep":
	}

	if j.name == "configure.base" && p.name == "file" {
		switch (*res).(type) {
		case fullfile:
			if len(ctx.a) == 1 && truly(ctx, is_compound{}) {
				errostack(ctx, 8, "unexpected fullfile: %v → %v", ts(ctx.a[0]), ts(*res)).trace()
			}
		case *file:
			if len(ctx.a) == 1 {
				if truly(ctx, is_compound{}) {
					// note(ctx, "%v → %v", ts(ctx.a[0]), ts(*res)).debug()
				} else if truly(ctx, is_exec{}) {
					errostack(ctx, 8, "expected fullfile: %v → %v", ts(ctx.a[0]), ts(*res)).trace()
				}
			}
		default:
			for _, a := range ctx.a {
				if x, y := a.(*list); !y {
					errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
				} else if x.len() != 1 {
					errostack(pc(ctx,a), 8, "%v ; %v", ts(x.elems), ts(*res)).trace()
				} else {
					a = x.elems[0]
				}

				var s = a.string(ctx)

				if strings.HasPrefix(ts(a), "{=compound {=fullfile .configure/") {
					if filepath.IsAbs(s) {
						errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
					}
				}

				if x, y := a.(*compound); y {
					if strings.Contains(ts(a), "{=fullfile .configure/") {
						if strings.Contains(s, "/Volumes/workout/") {
							errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
						}
					}
				} else if x != nil && x.len() == 0 {
					errostack(pc(ctx,a), 8, "%v ; %v", ts(a), ts(*res)).trace()
				}

				if f := j.file(ctx, s); f == nil {
					if strings.Contains(s, ".configure/") {
						if strings.HasSuffix(s, ".x") || strings.HasSuffix(s, ".o") {
							errostack(pc(ctx,a), 8, "not a file: %v ; %v", ts(a), ts(*res)).trace()
						}
					}
				}
			}
		}
	}

	if truly(ctx, is_exec{}) {
		switch j.name {
		case "configure.base":
			var e = _entry(ctx)
			if e == nil {
				errostack(ctx, 8, "%v %v → %v", ctx.x, ctx.a, *res).trace()
			}
			switch e.destiny().string(ctx) {
			case "-compiles-c", "-library-c", "-symbol-c":
				switch p.name {
				case "file":
					if len(ctx.a) == 1 {
						var s = ctx.a[0].String()
						if strings.HasPrefix(s, "{=file .configure/") {
							if strings.HasSuffix(s, ".c}.x") {
								if _, y := (*res).(fullfile); !y {
									errostack(ctx, 8, "not a fullfile: %v %v → %v", ctx.x, ctx.a[0], *res).trace()
								}
							}
						}
					}
				}
			}
		}
	}
}

func (d *def) set_check(ctx Context, o origin, val Value, app []Value) {
	switch j := _project(ctx); j.spec {
	case "testdata/value":
		switch d.name {
		case "val4":
			if s, t := val.String(), `a\,b\,c,x\,y\,z`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(val)).trace()
			}
			if s, t := d.value.String(), `a\,b\,c,x\,y\,z`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(d.value)).trace()
			}
		case "val5":
			if s, t := val.String(), `$(quote a\,b\,c,x\,y\,z)`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(val)).trace()
			}
			if s, t := d.value.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
				erro(ctx, "%v: %s != %s : %v", o, s, t, ts(d.value)).trace()
			}
		}
	case "testdata/value/optional":
		if d.name == "val2" && val != nil {
			if s, t1 := "{=delegate {=cond {=arrow {=project foo}→{=word name}}}}", ts(val); s == t1 {
				if s, t2 := "{=self foo}", ts(d.value); s != t2 {
					erro(ctx, "%v: %v: %v → %v != %s", o, val, t1, t2, s).trace()
				}
			}
		}
	}
	if isNull(d.value) {
		if truly(ctx, propExDef) && (app != nil || val != nil) {
			erro(ctx, "%v ; %v %v", d, val, app).trace()
		}
		if o == defExpand0 && (app != nil || !isNull(val)) {
			erro(ctx, "%v ; %v %v", d, val, app).trace()
		}
	}
	if !d.position.valid() && d.name != ".goals" {
		erro(ctx, "%v ; %v %v", d, val, app).trace()
	}
}

func (d *def) evoke_check(ctx *evocation, _res *Value, t time.Time) {
	var j, res = _project(ctx), *_res
	if u := time.Since(t); u > 2*time.Second {
		notestack(pc(ctx,d.value), 1, "%v: %v; %v", j.name, u, res).debug(32)
	}
	switch j.name {
	case "configure.base":
		d.evoke_check_configure_base(ctx, j, res)
	case "testvalue":
		switch j.spec {
		case "testdata/value":
			d.evoke_check_testvalue(ctx, j, res)
		}
	}
}
func (d *def) evoke_check_configure_base(ctx *evocation, j *project, res Value) {
	switch dest := _entry(ctx).destiny().string(ctx); dest {
	case "-cc", "-cxx", "-compiles-c", "-compiles-c++", "-library-c", "-library-c++", "-symbol-c", "-symbol-c++", "-function-c", "-function-c++", "-type-c", "-type-c++", "-variable-c", "-variable-c++", "-struct-member-c", "-struct-member-c++", "-headers-c", "-headers-c++":
		switch d.name {
		case "name":
			if t := auto_find(ctx, "TARGET"); t == nil {
				erro(ctx, "TARGET is nil: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
			}
			if _, y := res.(*path); !y {
				errostack(ctx, 8, "not a path: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
			}
		case "x", "o", "s":
			if !truly(ctx, is_compound{}) && truly(ctx, is_exec{}) {
				if _, y := res.(fullfile); !y {
					errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
				}
			} else {
				if _, y := res.(*file); !y {
					errostack(ctx, 8, "not a file: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
				}
			}
			if s := d.string(ctx); s == "" {
				errostack(ctx, 8, "empty: %s, %v → %v (%T)", dest, d, s, res).trace()
			}
		case "@":
			if _, y := res.(fullfile); !y {
				errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
			}
			if s := d.string(ctx); s == "" {
				errostack(ctx, 8, "empty: %s, %v → %v (%T)", dest, d, s, res).trace()
			}
		}
	case "-feature-c", "-feature-c++", "-sizeof-c", "-sizeof-c++", "-alignof-c", "-alignof-c++":
		// if _, y := res.(fullfile); !y {
		// 	note(ctx, "%v", auto_get(ctx, "@"))
		// 	note(ctx, "%v", auto_get(ctx, "<"))
		// 	note(ctx, "%v", auto_get(ctx, ">"))
		// 	errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
		// }
	case "-program-stdout", "-program-stderr", "-program-status":
		// if _, y := res.(fullfile); !y {
		// 	note(ctx, "%v", auto_get(ctx, "@"))
		// 	note(ctx, "%v", auto_get(ctx, "<"))
		// 	note(ctx, "%v", auto_get(ctx, ">"))
		// 	errostack(ctx, 8, "not a fullfile: %s, %v → %v (%T)", dest, d, ts(res), res).trace()
		// }
	}
}
func (d *def) evoke_check_testvalue(ctx *evocation, j *project, res Value) {
	switch d.name {
	case "disjunction0":
	case "disjunction00":
	case "disjunction01":
		if a := auto_get(ctx, "1"); a == nil {
			erro(pc(ctx,d.value), "$1 is nil : %s", ts(res)).trace()
		} else {
			switch v := a.(type) {
			case *list:
				var t string
				for i, e := range v.elems {
					if 0 < i { t += " " }
					t += "x"+e.String()
				}
				if s := res.String(); s != t {
					erro(pc(ctx,d.value), "%s != %s : %s, %s", s, t, ts(a), ts(res)).trace()
				}
			default:
				erro(pc(ctx,d.value), "%s, %s : %s, %s", a, res, ts(a), ts(res)).trace()
			}
		}
	case "disjunction02":
	case "disjunction1":
	case "disjunction2":
	case "disjunction3":
	}
}

func auto_find_check(ctx Context, name string, d *def) {
	if p := _project(ctx); p != nil {
		switch p.name {
		case "configure.base":
			if false && d != nil && d.name == "TYPE" && d.value != nil {
				switch d.value.String() {
				case "_Bool", "char", "int", "long", "long long":
				default:
					errostack(pc(ctx,d), 8, "%v %v", d.o, d.value).trace()
				}
			}
		}
		switch p.spec {
		case "testdata/value/auto":
			if t := do(ctx, find_auto{name}); d != nil && t == nil {
				var a = _automatic(ctx)
				var m, _ = a.defs[name]
				note(ctx, "%v", ts(ctx))
				note(ctx, "%v", ts(a))
				errostack(ctx, 8, "%v %v %v", name, d, m).trace()
			}
			if false {
				if ed, _ := do(ctx, evoke_def{"foo"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
			}
		case "testdata/value/placeholder":
			if false {
				if ed, _ := do(ctx, evoke_def{"val1"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val2"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val3"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val4"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
				if ed, _ := do(ctx, evoke_def{"val5"}).(*def); ed != nil {
					note(ctx, "%v %s", d, ts(ctx)).debug(32)
				}
			}
		}
	}
	return
}

func (ac *automatic) set_check(ctx Context, o origin, name string, val Value, _out **def, _old *Value) {
	if _project(ctx).name == "configure.base" {
		switch name {
		case "@", "<", ">":
			if val.patterned(ctx) {
				errostack(ctx, 3, "%v %v %v %s", o, name, val, ts(val)).trace()
			} else if s := val.String(); strings.Contains(s, "%") {
				errostack(ctx, 3, "%v %v %v %s", o, name, val, ts(val)).trace()
			}
		case "TYPE":
			v := val.String()

			if strings.Contains(v, "$1") {
				errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
			}

			a := auto_get(ctx, "@").string(ctx)
			s := strings.ToUpper(val.string(ctx))
			s  = strings.Replace(s, " ", "_",  -1)
			s  = strings.Replace(s, "*", "_P", -1)

			switch {
			case a == "-alignof-c":
				s = "ALIGNOF_" + s
				if x, y := ac.defs["TARGET"]; !y || x.value == nil {
					errostack(ctx, 8, "%v %v %v %s", o, name, val, ac.defs).trace()
				} else if s != x.value.String() {
					errostack(ctx, 8, "%v %v %v : %s != %s", o, name, val, x.value, s).trace()
				} else if strings.Contains(v, "$1") {
					errostack(ctx, 8, "%v %v %v : %s %s", o, name, val, x.value, s).trace()
				}
			case a == "-sizeof-c":
				s = "SIZEOF_" + s
				if x, y := ac.defs["TARGET"]; !y || x.value == nil {
					errostack(ctx, 8, "%v %v %v %s", o, name, val, ac.defs).trace()
				} else if s != x.value.String() {
					errostack(ctx, 8, "%v %v %v : %s != %s", o, name, val, x.value, s).trace()
				} else if strings.Contains(v, "$1") {
					errostack(ctx, 8, "%v %v %v : %s %s", o, name, val, x.value, s).trace()
				}
			case !strings.Contains(a, s):
				// @⇒{=file .configure/type/size/SIZEOF__BOOL.c.x}
				// @⇒{=file .configure/type/size/SIZEOF_CHAR.c.x}
				errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
			}

			switch {
			case strings.Contains(a, ".configure/type/align/ALIGNOF_"):
				// *⇒'align' 'ALIGNOF_CHAR'
				t := auto_get(ctx, "*").string(ctx)
				if !strings.Contains(t, "ALIGNOF_"+s) {
					errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
				}
			case strings.Contains(a, ".configure/type/size/SIZEOF_"):
				// *⇒'size' 'SIZEOF_CHAR'
				t := auto_get(ctx, "*").string(ctx)
				if !strings.Contains(t, "SIZEOF_"+s) {
					errostack(ctx, 8, "%v %v %v %s %v", o, name, val, ac.defs, *_old).trace()
				}
			}
		}
	}
}

func (ac *automatic) find_auto_check(ctx Context, d *def, name string) {
	if _project(ctx).name == "configure.base" {
		if name == "TYPE" && d.value != nil {
			if s := d.value.string(ctx); s == "_Bool" {
				if x, y := ac.defs["@"]; y {
					s = strings.ToUpper(s)
					s = strings.Replace(s, " ", "_",  -1)
					s = strings.Replace(s, "*", "_P", -1)
					a := x.String()
					switch {
					case strings.Contains(a, "SIZEOF_"+s),
						strings.Contains(a, "ALIGNOF_"+s),
						strings.Contains(a, "HAVE_TYPE_"+s): // okay
					default:
						erro(ctx, "TYPE is incorrect: %s %v", s, x).trace()
					}
				}
			}
		}
	}
}

func (ac *argumented_ctx) init_args_check(ctx Context, args []Value) {
	if false && 0 < len(args) && args[0].String() == "_Bool" {
		notestack(ctx, 3, "%v %v", args, auto_get(ctx, "1")).debug()
	}
	return
}

func (p *argumented) expand_check(ctx Context, res, val Value, args []Value) {
	if j := _project(ctx); j.name == "llvm.Config" {
		if a := auto_get(ctx, "_"); a != nil && len(p.args) == 1 && len(args) == 1 {
			if x1, y := p.Value.(*delegate); y {
				if a1, y := x1.x.(*auto); y && IsDigits(a1.name) {
					if x2, y := val.(*delegate); y {
						if a2, y := x2.x.(*auto); y && a1.name == a2.name {
							if s, t := a.String(), args[0].String(); s != t {
								errostack(ctx, 5, "%v: %v, %v, %s != %s", a, p, val, s, t).trace()
							} else if false {
								notestack(ctx, 5, "%v: %v, %v, %v, %s", a, p, res, val, t).debug(16)
							}
						}
					}
				}
			}
		}
	}
}

func (p *argumented) traverse_check(ctx Context, str string, args []Value) {
	if x, y := ctx.(*execution); y {
		if _project(ctx).name == "configure.base" {
			if p.Value.String() == "$(name).c.x" && len(p.args) == 3 {
				var a = auto_get(&x.automatic, "@")
				var t = auto_get(&x.automatic, "TYPE")
				if p.args[0].String() != "$(TYPE)" {
					errostack(ctx, 8, "%v %v %v %v", t, a, str, args).trace()
				}
				if p.args[1].String() != "$(INCLUDE)" {
					errostack(ctx, 8, "%v %v %v %v", t, a, str, args).trace()
				}
				if p.args[2].String() != "$(LIB)" {
					errostack(ctx, 8, "%v %v %v %v", t, a, str, args).trace()
				}
				if t != nil && t.string(ctx) != p.args[0].string(ctx) {
					if v, y := x.prerequisite.(*argumented); y && v != nil {
						errostack(ctx, 8, "%v %v %v %v %v %v", t, a, (p==v), v.Value, v.args, args).trace()
					}
				}
			}
		}
	}
}
