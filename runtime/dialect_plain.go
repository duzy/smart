//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        //"encoding/json"
        //"io"
)

type dialectPlain struct {
        polyInterpreter
}

func (t *dialectPlain) dialect() string { return "plain" }
func (t *dialectPlain) evaluate(prog *Program, recipes... types.Value) (result types.Value, err error) {
        result = values.Group(targetPlainKind, 
                values.String(joinRecipesString(recipes...)))
        return
}
