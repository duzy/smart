//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "errors"
        "fmt"
        "os"
)

// dialectDefault evaluates smart statements
type dialectDefault struct {
        polyInterpreter
}

func (t *dialectDefault) dialect() string { return "default" }
func (t *dialectDefault) evaluate(prog *Program, context *types.Scope, args []types.Value, recipes []types.Value) (types.Value, error) {
        var (
                list = values.List()
                err error
        )
evaluationLoop:
        for _, recipe := range recipes {
                switch stmt := recipe.(type) {
                case *types.None:
                case *types.List:
                        if stmt.Len() == 0 {
                                continue
                        }
                        var (
                                v = stmt.Get(0)
                                e error
                        )
                        switch t := v.(type) {
                        case *types.Def:
                                // Noop, just return to the caller.

                        case types.Caller:
                                v, e = t.Call(stmt.Slice(1)...)

                        case types.Executer:
                                var a []types.Value
                                if a, e = t.Execute(context, stmt.Slice(1)...); e == nil {
                                        if n := len(a); n == 1 {
                                                v = a[0]
                                        } else if n > 1 {
                                                v = values.List(a...)
                                        }
                                }

                        /*case yield:
                                if stmt.Len() == 1 {
                                        list.Append(v)
                                } else {
                                        list.Append(recipe)
                                }*/

                        default:
                                err = errors.New(fmt.Sprintf("Unsupported recipe command `%v' (%T)", t, t))
                                break evaluationLoop
                        }
                        if e == nil && v != nil {
                                list.Append(v)
                                if g, _ := v.(*types.Group); g != nil {
                                        if s, c := g.Get(0), g.Get(1); s != nil && c != nil &&
                                                s.Strval() == "shell" && c.Integer() != 0 {
                                                //fmt.Printf("evaluate: %v\n", v)
                                                break evaluationLoop
                                        }
                                }
                        } else if p, _ := e.(*returner); p != nil {
                                if p.value != nil {
                                        list.Append(p.value)
                                }
                                break evaluationLoop
                        } else if e != nil {
                                fmt.Fprintf(os.Stderr, "%v\n", e)
                                err = e; break evaluationLoop
                        }
                default:
                        fmt.Printf("recipe: %v (%T)\n", recipe, recipe)
                        panic("unreachable")
                }
        }
        //fmt.Printf("statement: %v\n", list)
        return list, err
}
