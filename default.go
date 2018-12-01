//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "errors"
        "fmt"
        "os"
)

// dialectDefault evaluates smart statements
type dialectDefault struct {
}

func (t *dialectDefault) Evaluate(prog *Program, args []Value) (result Value, err error) {
        var list []Value
        ForRecipes: for _, recipe := range prog.recipes {
                switch stmt := recipe.(type) {
                case *None:
                case *List:
                        if stmt.Len() == 0 { continue }

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
                                                v = &List{Elements{a}}
                                        }
                                }

                        default:
                                err = errors.New(fmt.Sprintf("unknown command `%v` (%T)", t, t))
                                break ForRecipes
                        }

                        if err == nil && v != nil {
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
                        } else if p, _ := err.(*Returner); p != nil {
                                if p.Value != nil {
                                        list = append(list, p.Value)
                                }
                                err = nil
                                break ForRecipes
                        } else {
                                fmt.Fprintf(os.Stderr, "%v\n", err)
                                break ForRecipes
                        }

                default:
                        fmt.Fprintf(os.Stderr, "fatal: unsupported recipe: %v (%T)\n", recipe, recipe)
                        panic("unreachable")
                }
        }
        result = MakeListOrScalar(list)
        return
}

func init() {
        var p = new(dialectDefault)
        RegisterDialect("eval", p)
        RegisterDialect("", p)
}
