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

func (t *dialectPlain) Dialect() string { return "plain" }
func (t *dialectPlain) Evaluate(prog *types.Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
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

func init() {
        types.RegisterInterpreter("plain", new(dialectPlain))
}
