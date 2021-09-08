//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "strings"
        "strconv"
        "bytes"
        "fmt"
)

// Value returned by (plain) modifier.
type Plain struct {
        valbase
        Name, Value string
}
func (p *Plain) String() (s string) {
        var value = strings.Replace(p.Value, "'", "\\'", -1)
        if p.Name == "" {
                s = fmt.Sprintf("(plain '%s')", value)
        } else {
                s = fmt.Sprintf("((plain %s) '%s')", p.Name, value)
        }
        return
}
func (p *Plain) Strval(_ Context) (string, error) { return p.Value, nil }
func (p *Plain) True(_ Context) (bool, error) { return strings.TrimSpace(p.Value) != "", nil }
func (p *Plain) Integer(_ Context) (int64, error) { return strconv.ParseInt(p.Value, 10, 64) }
func (p *Plain) Float(_ Context) (float64, error) { return strconv.ParseFloat(p.Value, 64) }
func (p *Plain) expand(_ Context, _ expandwhat) (Value, error) { return p, nil }
func (p *Plain) cmp(_ Context, v Value) (res cmpres) {
        if a, ok := v.(*Plain); ok {
                assert(ok, "value is not Plain")
                if p.Name == a.Name && p.Value == a.Value {
                        res = cmpEqual
                }
        }
        return
}

type plain struct {}

func (_ *plain) Evaluate(t *traversal, args ...Value) (result Value, err error) {
        var (
                ctx = t.Context
                pos = ctx.Position()
                str, name string
        )
        if len(args) > 0 {
                if name, err = args[0].Strval(ctx); err != nil {
                        diag.errorOf(args[0], "%v", err).debug(1)
                        return
                }
                t.program.language = name
        }
        if str, err = multiline(ctx, t.program.recipes...); err != nil {
                diag.errorOf(args[0], "%v", err).debug(1)
                return
        }
        str = strings.Replace(str, "\\\n\t", "\\\n", -1)
        result = &Plain{valbase{pos},name,str}
        return
}

func multiline(ctx Context, recipes... Value) (res string, err error) {
        var (
                pos = ctx.Position()
                x = len(recipes)-1
                w = new(bytes.Buffer)
                s string
        )
        for n, recipe := range recipes {
                if s, err = recipe.Strval(ctx); err != nil {
                        diag.errorAt(pos, "%v", err).debug(1)
                        return
                }
                if fmt.Fprint(w, s); n < x { fmt.Fprint(w, "\n") }
        }
        res = w.String()
        return
}
