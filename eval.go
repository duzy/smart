//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "fmt"
)

// evaluer evaluates smart statements
type evaluer struct { accumulation bool }
type evaluerOpts struct {
        debug bool `d,debug`
        fullname bool `f,fn,full,fullname`
}
func (p *evaluer) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var program = ctx.program()
        if program == nil {
                erro(ctx, "needs program context to evaluate: %v", ctx).debug(16)
                return
        }

        var list []Value
        var opts evaluerOpts
        args = parseOpts(ctx, &opts, expandmerge2(ctx, expandPlainValue, args...)...)

ForRecipes:
        for _, recipe := range program.recipes {
                var w = expandPlainValue | expandPathStr | expandPairVal
                if opts.fullname { w |= expandFullName }

                var (
                        ctx = positional(ctx, recipe.Position())
                        vals = expandmerge2(ctx, w, recipe)
                )
                if n := len(vals); n < 1 {
                        list = append(list, recipe)
                        continue
                } else if false && n == 1 && isTrivial(vals[0]) {
                        list = append(list, vals[0])
                        continue
                }

                var v Value
                switch tv := vals[0].(type) {
                case *undetermined:
                        // Noop, just return v to the caller.

                case *returner:
                        list = append(list, tv.Values...)
                        break ForRecipes

                case Caller:
                        v = tv.Call(positional(ctx, vals[0].Position()), vals[1:]...)

                case Executer:
                        var a, brks = tv.Execute(positional(ctx, program.Position()), vals[1:]...)
                        if a == nil {
                                // no return value
                        } else if tb := brks.not(breakCase, breakDone, breakNext); tb.has() {
                                brks = tb
                        } else if n := len(a); n == 1 { v = a[0] } else if n > 1 {
                                v = MakeList(recipe.Position(), a...)
                        }
                        for _, brk := range brks {
                                var s string
                                if brk.message != "" { s = brk.message }
                                if brk.error != nil { s += fmt.Sprintf(" (error: %s)", brk.error) }
                                erro(ctx, "eval '%v' breaked: (%s) %s", vals, brk.what, s).at(brk.pos).debug(1)
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
