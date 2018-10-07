//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package core

import (
        //"encoding/yaml"
        //"strings"
        //"io"
)

func DecodeYAML(source string, ws bool) (result Value, err error) {
        result = UniversalNone
        return 
}

type dialectYaml struct {
        whitespace bool
}

func (t *dialectYaml) Evaluate(prog *Program, args []Value, recipes []Value) (result Value, err error) {
        var source string
        if source, err = joinRecipesString(recipes...); err != nil { return }
        if result, err = DecodeYAML(source, t.whitespace); err == nil {
                result = &YAML{ result }
        } else {
                result = &YAML{ UniversalNone }
        }
        return
}

func init() {
        RegisterDialect("yaml", &dialectYaml{})
}
