//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"reflect"
)

func ex_check(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, e *bool, res, t, x *Value, a *[]Value) {
	if false && truly(ctx, is_test_mode{}) && ex_def1(ctx) {
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

    if *res == nil {
        if !_cl && x == nil {
            erro(ctx, "%v: %v → %v", ts(p), ts(_x), ts(*x)).trace()
        }

        var v = _x
        if a, y := v.(*auto); !y {
            // ...
        } else if d := auto_find(ctx, a.name); d != nil {
            note(ctx, "%v", ts(_x))
            note(ctx, "%v", ts(*x))
            note(ctx, "%v", ts(p))
            erro(ctx, "%v", ts(ctx)).trace()
        }

        if _cl {
            // TODO: closure checkpoints ...
        } else if _x == nil {
            note(ctx, "%v: nil", ts(p))
            erro(ctx, "%v", ts(ctx)).trace()
        } else if d, y := _x.(*def); y && false {
			if d == nil {
				erro(ctx, "%v", ts(ctx)).trace()
			} else if d.value != nil {
				note(ctx, "x=%v", ts(_x))
				note(ctx, "x=%v", ts(*x))
				erro(ctx, "p=%v", ts(p)).trace()
			}
        }
    } else if false && p != *res && equal(ctx, p, *res) {
        note(ctx, "%v: %p != %p", ts(*res), *res, p)
        erro(ctx, "%v", ts(ctx)).trace()
    }

	switch _project(ctx).spec {
	case "testdata/rule/shell/for-stdout":
		ex_check_rule_shell_forstdout(ctx, p, _x, _a, res, x, a)
	}
}

func ex_check_rule_shell_forstdout(ctx Context, p, _x Value, _a []Value, res, x *Value, a *[]Value) {
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
		} else if ex_delegate(ctx) {
			if ts(*res) != "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}" {
				errostack(ctx, 3, "%v: %s", p, ts(*res)).trace()
			}
		} else {
			erro(ctx, "%v: %s %s ; %s", p, ts(a1), ts(a2), ts(*res)).trace()
		}
	case "${.test a,b}":
		switch o {
		case defExpand0:
			if ts(*res) != "{=delegate {=builtin debug} {=list {=bareword b} {=bareword a}}}" {
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
		case "{=[Value] {=list {=bareword b} {=bareword a}}}":
			switch o {
			case defExpand0:
				if ts(*res) != "{=delegate {=builtin debug} {=list {=bareword b} {=bareword a}}}" {
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
