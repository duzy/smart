//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
        Name string
        Value *String
}
func (p *Plain) String() (s string) {
        var value = strings.Replace(p.Value.string, "'", "\\'", -1)
        if p.Name == "" {
                s = fmt.Sprintf("(plain '%s')", value)
        } else {
                s = fmt.Sprintf("((plain %s) '%s')", p.Name, value)
        }
        return
}
func (p *Plain) Strval(_ Context) string { return p.Value.string }
func (p *Plain) True(_ Context) bool { return strings.TrimSpace(p.Value.string) != "" }
func (p *Plain) Integer(_ Context) (n int64, e error) { return strconv.ParseInt(p.Value.string, 10, 64) }
func (p *Plain) Float(_ Context) (n float64, e error) { return strconv.ParseFloat(p.Value.string, 64) }
func (p *Plain) expand(_ Context, _ expandfacet) (val Value) {
        val = p.Value //MakeString(p.position, p.Value)
        return
}
func (p *Plain) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*Plain); ok {
                if p.Name == a.Name && (p.Value == a.Value || p.Value.string == a.Value.string) {
                        res = cmpEqual
                }
        } else if v.Strval(ctx) == p.Value.string {
                res = cmpEqual
        }
        return
}

type (
        plain struct {}
        plainOpts struct {
                generalOpts
        }
)
func (_ *plain) Evaluate(ctx Context, args ...Value) (result Value, err error) {
        var (
                program = ctx.program()
                pos = ctx.Position()
                str, name string
                opts plainOpts
        )
        if args = parseOpts(ctx, &opts, expandPlainValue, args...); len(args) > 0 {
                name = args[0].Strval(ctx)
                program.language = name
        }
        if str, err = multiline(ctx, program.recipes...); err != nil {
                erro(ctx, "%v", err).of(args[0]).debug(1)
                return
        } else if len(program.recipes) > 0 {
                pos = program.recipes[0].Position()
        }
        str = strings.Replace(str, "\\\n\t", "\\\n", -1)
        result = &Plain{valbase{pos}, name, MakeString(ctx.Position(), str)}
        if opts.debug>0 { warn(ctx, "%v", str).debug(opts.debug) }
        return
}

func multiline(ctx Context, recipes... Value) (res string, err error) {
        var (
                x = len(recipes)-1
                w = new(bytes.Buffer)
        )
        for n, recipe := range recipes {
                if fmt.Fprint(w, recipe.Strval(ctx)); n < x { fmt.Fprint(w, "\n") }
        }
        res = w.String()
        return
}
