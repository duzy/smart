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

func (p *evaluer) Evaluate(pos Position, t *traversal, args ...Value) (result Value, err error) {
        var list []Value
ForRecipes:
        for _, recipe := range t.program.recipes {
                if p.accumulation {
                        var v Value
                        // Expand both closures and delegates to ensure that
                        // the right recipe value is returned.
                        if v, err = recipe.expand(expandAll|expandPairVal); err != nil { return } else {
                                list = append(list, v)
                        }
                        continue ForRecipes
                }

                switch stmt := recipe.(type) {
                case *None:
                case *List:
                        if stmt.Len() == 0 { continue ForRecipes }

                        var v = stmt.Get(0)
                        switch tv := v.(type) {
                        case *undetermined:
                                // Noop, just return v to the caller.

                        case Caller:
                                v = tv.Call(t.program.position, stmt.Slice(1)...)

                        case Executer:
                                var ( a []Value; brks []*breaker )
                                if a, brks = tv.Execute(t.program.Position(), stmt.Slice(1)...); len(brks) == 0 {
                                        if n := len(a); n == 1 {
                                                v = a[0]
                                        } else if n > 1 {
                                                v = &List{elements{a}}
                                        }
                                } else {
                                        for _, brk := range brks {
                                                var s string
                                                if brk.message != "" { s = brk.message }
                                                if brk.error != nil { s += fmt.Sprintf(" (error: %s)", brk.error) }
                                                diag.errorAt(brk.pos, "%s: %s", brk.what, s)
                                        }
                                }

                        default:
                                v, err = tv.expand(expandClosure)
                        }
                        if v != nil {
                                if ret, okay := v.(*returner); okay {
                                        list = append(list, ret.Values...)
                                        break ForRecipes
                                }
                        }

                        if err != nil {
                                break ForRecipes
                        }

                        if v != nil {
                                list = append(list, v)
                                if g, _ := v.(*Group); g != nil {
                                        if s, c := g.Get(0), g.Get(1); s != nil && c != nil {
                                                var (str string; num int64)
                                                if str, err = s.Strval(); err != nil { return }
                                                if num, err = c.Integer(); err != nil { return }
                                                if str == "shell" && num != 0 {
                                                        //fmt.Fprintf(stderr, "evaluate: %v\n", v)
                                                        break ForRecipes
                                                }
                                        }
                                }
                        }

                default:
                        diag.errorOf(recipe, "unsupported recipe: %T", recipe)
                        return
                }
        }
        result = MakeListOrScalar(t.program.position, list)
        return
}
