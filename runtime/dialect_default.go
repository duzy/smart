//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/extbit/smart/types"
        "github.com/extbit/smart/values"
        "errors"
        "fmt"
        "os"
)

// dialectDefault evaluates smart statements
type dialectDefault struct {
}

func (t *dialectDefault) Evaluate(prog *types.Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var list = values.List()
LoopRecipes:
        for _, recipe := range recipes {
                switch stmt := recipe.(type) {
                case *types.None:
                case *types.List:
                        if stmt.Len() == 0 { continue }
                        var (
                                v = stmt.Get(0)
                                e error
                        )
                        switch t := v.(type) {
                        case *types.Def:
                                // Noop, just return v to the caller.
                                //fmt.Printf("dialectDefault: def: %s: %p %v (%s)\n", prog.project.Name(), t, t, t.Strval())

                        case types.Caller:
                                v, e = t.Call(prog.Position(), stmt.Slice(1)...)

                        case types.Executer:
                                var a []types.Value
                                if a, e = t.Execute(prog.Position(), stmt.Slice(1)...); e == nil {
                                        if n := len(a); n == 1 {
                                                v = a[0]
                                        } else if n > 1 {
                                                v = values.List(a...)
                                        }
                                }

                        default:
                                err = errors.New(fmt.Sprintf("Unknown recipe command `%v' (%T)", t, t))
                                break LoopRecipes
                        }

                        if e == nil && v != nil {
                                list.Append(v)
                                if g, _ := v.(*types.Group); g != nil {
                                        if s, c := g.Get(0), g.Get(1); s != nil && c != nil {
                                                var (
                                                        str string
                                                        num int64
                                                )
                                                if str, err = s.Strval(); err != nil { return }
                                                if num, err = c.Integer(); err != nil { return }
                                                if str == "shell" && num != 0 {
                                                        //fmt.Printf("evaluate: %v\n", v)
                                                        break LoopRecipes
                                                }
                                        }
                                }
                        } else if p, _ := e.(*types.Returner); p != nil {
                                if p.Value != nil {
                                        list.Append(p.Value)
                                }
                                break LoopRecipes
                        } else if e != nil {
                                fmt.Fprintf(os.Stderr, "%v\n", e)
                                err = e; break LoopRecipes
                        }

                default:
                        fmt.Fprintf(os.Stderr, "fatal: unsupported recipe: %v (%T)\n", recipe, recipe)
                        panic("unreachable")
                }
        }
        //fmt.Printf("statement: %v\n", list)
        return list, err
}

func init() {
        types.RegisterDialect("", new(dialectDefault))
}
