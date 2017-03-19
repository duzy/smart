//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
)

type interpretMode int

const (
        interpretSingle interpretMode = 1<<iota
        interpretMulti
)

type interpreter interface {
        dialect() string
        mode() interpretMode
        evaluate(recipes... types.Value) (types.Value, error)
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

type dialectTrivial struct {
        polyInterpreter
}

func (t *dialectTrivial) dialect() string { return "trivial" }
func (t *dialectTrivial) evaluate(recipes... types.Value) (types.Value, error) {
        return nil, nil
}

var trivialDialect = new(dialectTrivial)
