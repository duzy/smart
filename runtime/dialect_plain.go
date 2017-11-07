//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
)

type dialectPlain struct {
        polyInterpreter
}

func (t *dialectPlain) dialect() string { return "plain" }
func (t *dialectPlain) evaluate(prog *Program, context *types.Scope, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var name string
        if len(args) > 0 {
                name = args[0].Strval()
        }
        result = &types.Plain{
                joinRecipesString(recipes...),
                name,
        }
        return
}
