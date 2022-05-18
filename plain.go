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
func (p *Plain) expand(_ Context, _ expandwhat) (val Value, err error) {
        val = MakeString(p.position, p.Value)
        return
}
func (p *Plain) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*Plain); ok {
                if p.Name == a.Name && p.Value == a.Value {
                        res = cmpEqual
                }
        } else if s, err := v.Strval(ctx); err != nil {
                erro(ctx, `strval "%v" failed: %v`, v, err).debug(1)
        } else if s == p.Value {
                res = cmpEqual
        }
        return
}

type plain struct {}

func (_ *plain) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var (
                program = ctx.program()
                pos = ctx.Position()
                str, name string
        )
        if len(args) > 0 {
                if name, err = args[0].Strval(ctx); err != nil {
                        erro(ctx, "%v", err).of(args[0]).debug(1)
                        return
                }
                program.language = name
        }
        if str, err = multiline(ctx, program.recipes...); err != nil {
                erro(ctx, "%v", err).of(args[0]).debug(1)
                return
        } else if len(program.recipes) > 0 {
                pos = program.recipes[0].Position()
        }
        str = strings.Replace(str, "\\\n\t", "\\\n", -1)
        result = &Plain{valbase{pos},name,str}
        if false && ctx.Project().name == "c++" { warn(ctx, "%v", str).debug(1) }
        return
}

func multiline(ctx Context, recipes... Value) (res string, err error) {
        var (
                x = len(recipes)-1
                w = new(bytes.Buffer)
                s string
        )
        for n, recipe := range recipes {
                if s, err = recipe.Strval(ctx); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        return
                }
                if fmt.Fprint(w, s); n < x { fmt.Fprint(w, "\n") }
        }
        res = w.String()
        return
}
