//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package core

import (
        "strconv"
        //"fmt"
)

// Value returned by (plain) modifier.
type Plain struct {
        Value string
        Name string
}
func (p *Plain) closured() bool { return false }
func (p *Plain) disclose() (Value, error) { return nil, nil }
func (p *Plain) reveal() (Value, error) { return nil, nil }
func (p *Plain) refs(_ Object) bool { return false }
func (p *Plain) Type() Type  { return PlainType }
func (p *Plain) String() string {
        s := "(plain"
        if p.Name != "" {
                s += "(" + p.Name + ")"
        } 
        s += " " + p.Value + ")"
        return s
}
func (p *Plain) Strval() (string, error) { return p.Value, nil }
func (p *Plain) Integer() (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *Plain) Float() (float64, error) { return strconv.ParseFloat(p.Value, 64) }

type _plain struct {}

func (t *_plain) Evaluate(prog *Program, args []Value, recipes []Value) (result Value, err error) {
        var str, name string
        if len(args) > 0 {
                if name, err = args[0].Strval(); err != nil { return }
        }
        if str, err = joinRecipesString(recipes...); err != nil { return }
        //fmt.Printf("plain: %s\n", str)
        result = &Plain{ str, name, }
        return
}

func init() {
        RegisterDialect("plain", new(_plain))
}
