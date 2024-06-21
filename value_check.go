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
			note(ctx, "x=%v→%v", ts(_x), ts(x))
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
            erro(ctx, "%v: %v → %v", ts(p), ts(_x), ts(x)).trace()
        }

        var v = _x
        if a, y := v.(*auto); !y {
            // ...
        } else if d := auto_find(ctx, a.name); d != nil {
            note(ctx, "%v", ts(_x))
            note(ctx, "%v", ts(x))
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
                note(ctx, "%v", ts(_x))
                note(ctx, "%v", ts(x))
                note(ctx, "%v", ts(p))
                erro(ctx, "%v", ts(ctx)).trace()
            }
        }
    } else if false && p != *res && equal(ctx, p, *res) {
        note(ctx, "%v: %p != %p", ts(*res), *res, p)
        erro(ctx, "%v", ts(ctx)).trace()
    }
}
