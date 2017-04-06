//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        //"errors"
        //"fmt"
)

type interpretMode int

const (
        interpretSingle interpretMode = 1<<iota
        interpretMulti
)

type interpreter interface {
        dialect() string
        mode() interpretMode
        evaluate(prog *Program, args []types.Value, recipes []types.Value) (types.Value, error)
}

type monoInterpreter struct {
}

type polyInterpreter struct {
}

func (*monoInterpreter) mode() interpretMode { return interpretSingle }
func (*polyInterpreter) mode() interpretMode { return interpretMulti }

func joinRecipesString(recipes... types.Value) string {
        var (
                x = len(recipes)-1
                s string
        )
        for n, recipe := range recipes {
                if s += recipe.String(); n < x {
                        s += "\n"
                }
        }
        return s
}

// dialectDefault evaluates smart statements
type dialectDefault struct {
        polyInterpreter
}

func (t *dialectDefault) dialect() string { return "default" }
func (t *dialectDefault) evaluate(prog *Program, args []types.Value, recipes []types.Value) (types.Value, error) {
        var (
                list = values.List()
                err error
        )
evaluationLoop:
        for _, recipe := range recipes {
                switch stmt := recipe.(type) {
                case *values.ListValue:
                        if stmt.Len() == 0 {
                                continue
                        }
                        var (
                                v = stmt.Get(0)
                                e error
                        )
                        switch ident := v.(type) {
                        case *types.Builtin:   v, e = ident.Call(stmt.Slice(1)...)
                        case *types.RuleEntry: v, e = ident.Call(stmt.Slice(1)...)
                         default:
                                if stmt.Len() == 1 {
                                        list.Append(v)
                                } else {
                                        list.Append(recipe)
                                }
                        }
                        if e == nil && v != nil {
                                list.Append(v)
                                if g, _ := v.(*values.GroupValue); g != nil {
                                        if s, c := g.Get(0), g.Get(1); s != nil && c != nil &&
                                                s.String() == "shell" && c.Integer() != 0 {
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
                                err = e; break evaluationLoop
                        }
                default:
                        panic("unreachable")
                }
        }
        //fmt.Printf("statement: %v\n", list)
        return list, err
}
