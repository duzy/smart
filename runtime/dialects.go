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
        var list = values.List()
evaluationLoop:
        for _, recipe := range recipes {
                switch stmt := recipe.(type) {
                case *values.ListLiteral:
                        if stmt.Len() == 0 {
                                continue
                        }
                        var v = stmt.Get(0)
                        switch ident := v.(type) {
                        case *types.Builtin:
                                if v, e := ident.Call(stmt.Slice(1)...); e == nil && v != nil {
                                        list.Append(v)
                                } else if p, _ := e.(*returner); p != nil {
                                        if p.value != nil {
                                                list.Append(p.value)
                                        }
                                        break evaluationLoop
                                }
                        case *types.RuleEntry:
                                if v, e := ident.Call(stmt.Slice(1)...); e == nil && v != nil {
                                        list.Append(v)
                                } else if p, _ := e.(*returner); p != nil {
                                        if p.value != nil {
                                                list.Append(p.value)
                                        }
                                        break evaluationLoop
                                }
                                /*
                        case *values.IdentValue:
                                var _, sym = prog.context.lookupAt(token.NoPos, ident.Names, false)
                                if sym == nil {
                                        s := fmt.Sprintf("undefined statement %s", ident)
                                        return nil, errors.New(s)
                                } else {
                                        //fmt.Printf("statement: %v %T\n", ident.Names, sym)
                                        if v, _ = sym.Call(stmt.Slice(1)...); v != nil {
                                                list.Append(v)
                                        }
                                        //fmt.Printf("statement: %v %v\n", ident.Names, v)
                                } */
                        default:
                                if stmt.Len() == 1 {
                                        list.Append(v)
                                } else {
                                        list.Append(recipe)
                                }
                        }
                default:
                        panic("unreachable")
                }
        }
        //fmt.Printf("statement: %v\n", list)
        return list, nil
}
