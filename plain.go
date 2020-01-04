//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "strings"
        "strconv"
        "fmt"
)

// Value returned by (plain) modifier.
type Plain struct {
        trivial
        Name, Value string
}
func (p *Plain) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Plain) True() bool { return strings.TrimSpace(p.Value) != "" }
func (p *Plain) String() (s string) {
        if p.Name == "" {
                s = fmt.Sprintf("((plain) %s)", p.Value)
        } else {
                s = fmt.Sprintf("((plain %s) %s)", p.Name, p.Value)
        }
        return
}
func (p *Plain) Strval() (string, error) { return p.Value, nil }
func (p *Plain) Integer() (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *Plain) Float() (float64, error) { return strconv.ParseFloat(p.Value, 64) }
func (p *Plain) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Plain); ok {
                assert(ok, "value is not Plain")
                if p.Name == a.Name && p.Value == a.Value {
                        res = cmpEqual
                }
        }
        return
}

type _plain struct {}

func (t *_plain) Evaluate(prog *Program, args []Value) (result Value, err error) {
        var str, name string
        if len(args) > 0 {
                if name, err = args[0].Strval(); err != nil { return }
        }
        if str, err = joinRecipesString(prog.recipes...); err != nil { return }
        str = strings.Replace(str, "\\\n\t", "\\\n", -1)
        result = &Plain{trivial{prog.position},name,str}
        return
}
