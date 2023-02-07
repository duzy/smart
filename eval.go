//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

// evaluer evaluates smart statements
type evaluer struct { accumulation bool }
type evaluerOpts struct {
    generalOpts
}
func (p *evaluer) Evaluate(ctx Context, args ...Value) (result Value, err error) {
    var program = ctx.program()
    if program == nil {
        erro(ctx, "needs program context to evaluate: %v", ctx).debug(16)
        return
    }

    var list []Value
    var opts evaluerOpts
    args = parseOpts(ctx, &opts, plain, args...)

ForRecipes:
    for _, recipe := range program.recipes {
        var w = plain | expandPathStr | expandPairVal
        if opts.fullname { w |= expandFullName }

        var (
            ctx = at(ctx, recipe.Position())
            vals = mergex(ctx, w, recipe)
        )
        if n := len(vals); n < 1 {
            list = append(list, recipe)
            continue
        } else if false && n == 1 && isTrivial(vals[0]) {
            list = append(list, vals[0])
            continue
        }

        ctx = at(ctx, vals[0].Position())

        var v Value
        switch tv := vals[0].(type) {
        case *undetermined:
            // Noop, just return v to the caller.

        case *returner:
            list = append(list, tv.Values...)
            break ForRecipes

        case Caller:
            v = tv.Call(ctx, vals[1:]...)

        case Executer:
            var a, traves = tv.Execute(ctx, vals[1:]...)
            if a == nil {
                // no return value
            } else if t := traves.not(traveCase, traveDone, traveNext); t.has() {
                traves = t
            } else if n := len(a); n == 1 {
                v = a[0]
            } else if n > 1 {
                v = MakeList(recipe.Position(), a...)
            }
            for _, brk := range traves {
                erro(at(ctx,brk.pos), "eval '%v': %v", vals, brk).debug(1)
            }

        default:
            list = append(list, vals...)
            continue
        }

        if /*isNil*/isTrivial(v) { continue }

        list = append(list, v)
        if g, ok := v.(*Group); ok && g != nil && g.Len() > 0 {
            if s, c := g.Get(0), g.Get(1); s != nil && c != nil {
                var str = s.Strval(ctx)
                if num, e := c.Integer(ctx); e != nil {
                    erro(ctx, "%v: %v", c, e).debug(1)
                } else if str == "shell" && num != 0 {
                    //fmt.Fprintf(stderr, "evaluate: %v\n", v)
                    break ForRecipes
                }
            }
        }
    }
    result = MakeListOrScalar(program.position, list)
    return
}
