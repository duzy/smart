//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        //"encoding/yaml"
        //"strings"
        //"io"
)

func DecodeYAML(source string, ws bool) (result types.Value, err error) {
        result = values.None
        return
}

type dialectYaml struct {
        polyInterpreter
        whitespace bool
}

func (t *dialectYaml) dialect() string { return "yaml" }
func (t *dialectYaml) evaluate(prog *Program, args []types.Value, recipes []types.Value) (result types.Value, err error) {
        var source = joinRecipesString(recipes...)
        if result, err = DecodeYAML(source, t.whitespace); err == nil {
                result = &types.YAML{ result }
        } else {
                result = &types.YAML{ values.None }
        }
        return
}
