//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "extbit.io/smart/types"
        //"fmt"
)

type dialectPlain struct {
}

func (t *dialectPlain) Evaluate(prog *types.Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var str, name string
        if len(args) > 0 {
                if name, err = args[0].Strval(); err != nil { return }
        }
        if str, err = joinRecipesString(recipes...); err != nil { return }
        //fmt.Printf("plain: %s\n", str)
        result = &types.Plain{ str, name, }
        return
}

func init() {
        types.RegisterDialect("plain", new(dialectPlain))
}
