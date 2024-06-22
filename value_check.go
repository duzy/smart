//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func ex_check(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, e *bool, res, t, x *Value, a *[]Value) {
	if true && truly(ctx, is_test_mode{}) && ex_def1(ctx) {
		if s := p.String(); true && "${.test.for $1,$2}" == s {
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
        } else if d, y := _x.(*def); y {
            if d == nil {
                erro(ctx, "%v", ts(ctx)).trace()
            } else if d.value != nil {
                note(ctx, "x=%v", ts(_x))
                note(ctx, "x=%v", ts(*x))
                note(ctx, "p=%v", ts(p))
                erro(ctx, "%v", ts(ctx)).trace()
            }
        }
    } else if false && p != *res && equal(ctx, p, *res) {
        note(ctx, "%v: %p != %p", ts(*res), *res, p)
        erro(ctx, "%v", ts(ctx)).trace()
    }

	switch _project(ctx).spec {
	case "testdata/rule/shell/for-stdout":
		ex_check_rule_shell_forstdout(ctx, p, _x, _a, res, a)
	}
}

func ex_check_rule_shell_forstdout(ctx Context, p, _x Value, _a []Value, res *Value, a *[]Value) {
	switch p.String() {
	case "$(debug $(line) $(str))":
		if ts(_a) == "{=[Value] {=list {=delegate {=auto line}} {=delegate {=auto str}}}}" {
			var s string
			switch ts(*a) {
			case "{=[Value] {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}": // okay
				s = "{=delegate {=builtin debug} {=list {=delegate {=auto 2}} {=delegate {=auto 1}}}}"
			case "{=[Value] {=list {=bareword b} {=bareword a}}}": // okay
				s = "{}"
			default:
				if false { errostack(ctx, 5, "%v %v", ts(*a), ts(*res)).trace() }
			}
			switch try[origin](ctx, get_origin{}) {
			case defExpand0:
				if ts(*res) != "{=delegate {=builtin debug} {=list {=bareword b} {=bareword a}}}" {
					errostack(ctx, 5, "%v", ts(*res)).trace()
				}
			case defExpand1, defExpand2:
				if ts(*res) != s {
					errostack(ctx, 5, "%v != %s", ts(*res), s).trace()
				}
			}
		}
	}
}
