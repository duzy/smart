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

func (p *evaluer) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var (
                program = ctx.program()
                list []Value
        )
        if program == nil {
                erro(ctx, "needs program context to evaluate: %v", ctx).debug(16)
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

                var ctx = positional(ctx, recipe.Position())
                switch stmt := recipe.(type) {
                case *Nil, *None, *unresolvedobject:
                case *List:
                        if stmt.Len() == 0 { continue ForRecipes }

                        var v = stmt.Get(0)
                        switch tv := v.(type) {
                        case *undetermined:
                                // Noop, just return v to the caller.

                        case Caller:
                                v = tv.Call(positional(ctx, v.Position()), stmt.Slice(1)...)

                        case Executer:
                                var ( a []Value; brks []*breaker )
                                if a, brks = tv.Execute(positional(ctx, program.Position()), stmt.Slice(1)...); len(brks) == 0 {
                                        if n := len(a); n == 1 {
                                                v = a[0]
                                        } else if n > 1 {
                                                v = MakeList(recipe.Position(), a...)
                                        }
                                } else {
                                        for _, brk := range brks {
                                                var s string
                                                if brk.message != "" { s = brk.message }
                                                if brk.error != nil { s += fmt.Sprintf(" (error: %s)", brk.error) }
                                                erro(ctx, "eval '%v' breaked: (%s) %s", stmt, brk.what, s).at(brk.pos).debug(1)
                                        }
                                }

                        default:
                                if v, err = tv.expand(ctx, expandPlainValue); err != nil {
                                        erro(ctx, "expand recipe value failed: %v", err).debug(1)
                                        return
                                } else if isNil(v) { v = tv }
                        }
                        if v != nil {
                                if ret, okay := v.(*returner); okay {
                                        list = append(list, ret.Values...)
                                        break ForRecipes
                                }
                        }

                        if err != nil {
                                erro(ctx, "evaluation failed: %v", err).debug(1)
                                break ForRecipes
                        }

                        if v != nil {
                                list = append(list, v)
                                if g, _ := v.(*Group); g != nil {
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

                default:
                        var tv, _ = ctx.autoGet("@")
                        erro(ctx, "unsupported recipe: %T (target=%v)", recipe, tv).of(recipe).debug(16)
                        return
                }
        }
        result = MakeListOrScalar(program.position, list)
        return
}
