//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "fmt"
        "os"
)

// evaluer evaluates smart statements
type evaluer struct {
        accumulation bool
}

func (t *evaluer) Evaluate(prog *Program, args []Value) (result Value, err error) {
        var list []Value
ForRecipes:
        for _, recipe := range prog.recipes {
                if t.accumulation {
                        var v Value
                        // Expand both closures and delegates to ensure that
                        // the right recipe value is returned.
                        if v, err = recipe.expand(expandAll); err != nil { return } else {
                                list = append(list, v)
                        }
                        continue ForRecipes
                }
                
                switch stmt := recipe.(type) {
                case *None:
                case *List:
                        if stmt.Len() == 0 { continue ForRecipes }

                        var v = stmt.Get(0)
                        switch t := v.(type) {
                        case *undetermined:
                                // Noop, just return v to the caller.

                        case Caller:
                                v, err = t.Call(prog.Position(), stmt.Slice(1)...)

                        case Executer:
                                var a []Value
                                if a, err = t.Execute(prog.Position(), stmt.Slice(1)...); err == nil {
                                        if n := len(a); n == 1 {
                                                v = a[0]
                                        } else if n > 1 {
                                                v = &List{elements{a}}
                                        }
                                }

                        default:
                                v, err = t.expand(expandClosure)
                        }

                        if err != nil {
                                if p, _ := err.(*Returner); p != nil {
                                        if p.Value != nil {
                                                list = append(list, p.Value)
                                        }
                                        err = nil
                                        break ForRecipes
                                } else {
                                        fmt.Fprintf(os.Stderr, "eval: %v\n", err)
                                        break ForRecipes
                                }
                        }

                        if v != nil {
                                list = append(list, v)
                                if g, _ := v.(*Group); g != nil {
                                        if s, c := g.Get(0), g.Get(1); s != nil && c != nil {
                                                var (str string; num int64)
                                                if str, err = s.Strval(); err != nil { return }
                                                if num, err = c.Integer(); err != nil { return }
                                                if str == "shell" && num != 0 {
                                                        //fmt.Printf("evaluate: %v\n", v)
                                                        break ForRecipes
                                                }
                                        }
                                }
                        }

                default:
                        fmt.Fprintf(os.Stderr, "fatal: unsupported recipe: %v (%T)\n", recipe, recipe)
                        unreachable()
                }
        }
        result = MakeListOrScalar(list)
        return
}
