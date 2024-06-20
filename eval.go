//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

type invoker interface { invoke(Context, []Value, []Value) Value }
type executer interface { execute(Context, ...Value) []Value }

// eval evaluates smart statements
type eval struct { accumulation, eval bool }
func (p *eval) evaluate(ctx Context, args ...Value) (result Value) {
    var program = _program(ctx)
    if program == nil {
        erro(ctx, "needs program context to evaluate: %v", ctx).trace()
    }

    var list []Value
    var opts struct { generalOpts }
    args = parseOpts(final{ctx}, &opts, args...)

    for _, recipe := range program.recipes {
        var vals = merge(recipe)

        if n := len(vals); n < 1 {
            if false { list = append(list, recipe) }
            continue
        }

        var op = vals[0]
        var ov []Value // opt-vals
        if a, y := op.(*argumented); y { op, ov = a.Value, a.args }

        var ctx = at(ctx, op)

        switch t := op.(type) {
        case *returner:
            result = ease(ctx, t.vals)
            return

        case invoker:
            if v := t.invoke(ctx, ov, vals[1:]); v != nil {
                if p.accumulation {
                    list = append(list, v)
                } else {
                    list = []Value{ v }
                }
            }

        case executer:
            if a := t.execute(ctx, vals[1:]...); a != nil {
                if p.accumulation {
                    list = append(list, a...)
                } else {
                    list = a
                }
            }

        case *undetermined:
            if p.accumulation {
                list = append(list, vals...)
            } else {
                list = vals
            }

        default:
            if p.eval { vals = expand(ctx, vals...) }
            if p.accumulation {
                list = append(list, vals...)
            } else {
                list = vals
            }
        }
    }

    result = ease(ctx, list)
    return
}
