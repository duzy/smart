//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
)

type interpretMode int

const (
        interpretSingle interpretMode = 1<<iota
        interpretMulti
)

type interpreter interface {
        dialect() string
        mode() interpretMode
        evaluate(prog *Program, recipes... types.Value) (types.Value, error)
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
func (t *dialectDefault) evaluate(prog *Program, recipes... types.Value) (types.Value, error) {
        var list = values.List()
        for _, recipe := range recipes {
                switch stmt := recipe.(type) {
                case *values.ListLiteral:
                        if stmt.Len() == 0 {
                                continue
                        }
                        var v = stmt.Get(0)
                        if bw, _ := v.(*values.BarewordLiteral); bw != nil {
                                var rest = stmt.Slice(1)
                                list.Append(prog.context.Call(bw.String(), rest...))
                        } else if bc, _ := v.(*values.BarecompLiteral); bc != nil {
                                panic("todo: BarecompLiteral")
                        } else if stmt.Len() == 1 {
                                list.Append(v)
                        } else {
                                list.Append(recipe)
                        }
                default:
                        panic("unreachable")
                }
        }
        return list, nil
}
