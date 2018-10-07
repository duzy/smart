//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package core

import (
        "errors"
        "fmt"
        "os"
)

// dialectDefault evaluates smart statements
type dialectDefault struct {
}

func (t *dialectDefault) Evaluate(prog *Program, args []Value, recipes []Value) (result Value, err error) {
        var list = &List{}
LoopRecipes:
        for _, recipe := range recipes {
                switch stmt := recipe.(type) {
                case *None:
                case *List:
                        if stmt.Len() == 0 { continue }
                        var (
                                v = stmt.Get(0)
                                e error
                        )
                        switch t := v.(type) {
                        case *Def:
                                // Noop, just return v to the caller.
                                //fmt.Printf("dialectDefault: def: %s: %p %v (%s)\n", prog.project.Name(), t, t, t.Strval())

                        case Caller:
                                v, e = t.Call(prog.Position(), stmt.Slice(1)...)

                        case Executer:
                                var a []Value
                                if a, e = t.Execute(prog.Position(), stmt.Slice(1)...); e == nil {
                                        if n := len(a); n == 1 {
                                                v = a[0]
                                        } else if n > 1 {
                                                v = &List{Elements{a}}
                                        }
                                }

                        default:
                                err = errors.New(fmt.Sprintf("Unknown recipe command `%v' (%T)", t, t))
                                break LoopRecipes
                        }

                        if e == nil && v != nil {
                                list.Append(v)
                                if g, _ := v.(*Group); g != nil {
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
                        } else if p, _ := e.(*Returner); p != nil {
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
        RegisterDialect("", new(dialectDefault))
}
