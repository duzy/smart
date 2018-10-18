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
        "fmt"
)

type YAML struct {
        Value Value
}
func (p *YAML) closured() bool { return p.Value.closured() }
func (p *YAML) disclose() (Value, error) { return p.Value.disclose() }
func (p *YAML) reveal() (Value, error) { return p.Value.reveal() }
func (p *YAML) refs(_ Object) bool { return false }
func (p *YAML) Type() Type { return YAMLType }
func (p *YAML) String() string { return "(json " + p.Value.String() + ")" }
func (p *YAML) Strval() (string, error) { return p.Value.Strval() }
func (p *YAML) Integer() (int64, error) { return 0, nil }
func (p *YAML) Float() (float64, error) { return 0, nil }

func DecodeYAML(source string, ws bool) (result Value, err error) {
        err = fmt.Errorf("DecodeYAML not implemented yet")
        return 
}

type _yaml struct {
        whitespace bool
}

func (t *_yaml) Evaluate(prog *Program, args []Value, recipes []Value) (result Value, err error) {
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
        RegisterDialect("yaml", &_yaml{})
}
