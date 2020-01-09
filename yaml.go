//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        //yaml_enc "encoding/yaml"
        //"strings"
        //"io"
        "fmt"
)

type YAML struct { Value }
func (p *YAML) String() string { return "(json " + p.Value.String() + ")" }
func (p *YAML) cmp(v Value) (res cmpres) {
        if a, ok := v.(*YAML); ok {
                assert(ok, "value is not YAML")
                res = p.Value.cmp(a.Value)
        }
        return
}

func DecodeYAML(source string, ws bool) (result Value, err error) {
        err = fmt.Errorf("DecodeYAML not implemented yet")
        return 
}

type yaml struct {
        whitespace bool
}

func (t *yaml) Evaluate(pc *traversal, args []Value) (result Value, err error) {
        var source string
        if source, err = joinRecipesString(pc.program.recipes...); err != nil { return }
        if result, err = DecodeYAML(source, t.whitespace); err == nil {
                result = &YAML{ result }
        } else {
                result = &YAML{ &None{trivial{pc.program.position}} }
        }
        return
}
