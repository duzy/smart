//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
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
        fullname bool `f,fn,full,fullname`
}
func (p *evaluer) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var (
                program = ctx.program()
                opts evaluerOpts
                list []Value
        )
        if program == nil {
                erro(ctx, "needs program context to evaluate: %v", ctx).debug(16)
                return
        } else if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse opts failed: %v", err)
                errostack(ctx, 5, "%v", ctx).debug(1)
                return
        }

ForRecipes:
        for _, recipe := range program.recipes {
                if p.accumulation {
                        var v Value
                        // Expand both closures and delegates to ensure that
                        // the right recipe value is returned.
                        if v, err = recipe.expand(ctx, expandPlainValue|expandPairVal); err != nil {
                                erro(ctx, "expand recipe failed: %v", err).debug(1)
                                return
                        } else if isNil(v) {
                                v = recipe
                        }
                        list = append(list, v)
                        continue ForRecipes
                }

                var (
                        ctx = positional(ctx, recipe.Position())
                        w = expandPlainValue
                        vals []Value
                )
                if opts.fullname { w |= expandFullName }
                if vals, err = expandmerge2(ctx, w, recipe); err != nil {
                        erro(ctx, "merge recipes failed: %v", err).debug(1)
                        return
                }
                if len(vals) < 1 {
                        list = append(list, recipe)
                        continue
                } else if isTrivial(vals[0]) {
                        erro(ctx, "trivial recipe op: %v -> %v", recipe, vals).debug(1)
                        return
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
                        if n := len(vals); n == 1 {
                                v = tv
                        } else if n > 1 {
                                v = MakeList(tv.Position(), vals...) // normal list
                        }
                }

                if isNil(v) { continue }

                list = append(list, v)
                if g, ok := v.(*Group); ok && g != nil && g.Len() > 0 {
                        if s, c := g.Get(0), g.Get(1); s != nil && c != nil {
                                var ( str string; num int64 )
                                if str, err = s.Strval(ctx); err != nil {
                                        erro(ctx, "strval '%v' failed: %v", s, err).debug(1)
                                        return
                                }
                                if num, err = c.Integer(ctx); err != nil {
                                        erro(ctx, "integify '%v' failed: %v", c, err).debug(1)
                                        return
                                }
                                if str == "shell" && num != 0 {
                                        //fmt.Fprintf(stderr, "evaluate: %v\n", v)
                                        break ForRecipes
                                }
                        }
                }
        }
        result = MakeListOrScalar(program.position, list)
        return
}
