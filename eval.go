//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

// evaluer evaluates smart statements
type evaluer struct { accumulation bool }
type evaluerOpts struct { generalOpts }
func (p *evaluer) evaluate(ctx Context, args ...Value) (result Value, err error) {
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
        if !p.accumulation { list = nil }

        var (
            ctx = at(ctx, recipe.Position())
            vals = xmerge(ctx, w, recipe)
            n = len(vals)
        )
        if n < 1 {
            list = append(list, recipe)
            continue
        }

        var (
            name = vals[0]
            ov []Value
        )
        if a, y := name.(*argumented); y { name, ov = a.Value, a.args }

        ctx = at(ctx, name.Position())

        var v Value
        switch tv := name.(type) {
        case *undetermined:
            // Noop, just return v to the caller.

        case *returner:
            list = append(list, tv.Values...)
            break ForRecipes

        case invoker:
            v = tv.invoke(ctx, plain, ov, vals[1:])

        case executer:
            var a, traves = tv.execute(ctx, vals[1:]...)
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
        if g, ok := v.(*group); ok && g != nil && g.Len() > 0 {
            if s, c := g.Get(0), g.Get(1); s != nil && c != nil {
                var str = s.string(ctx)
                if num, e := c.int(ctx); e != nil {
                    erro(ctx, "%v: %v", c, e).debug(1)
                } else if str == "shell" && num != 0 {
                    //prompt(ctx, "evaluate: %v\n", v)
                    break ForRecipes
                }
            }
        }
    }
    result = ease(ctx, list)
    return
}
