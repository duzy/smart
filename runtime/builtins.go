//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        //"fmt"
)

type builtin func(ctx *Context, args... types.Value) (types.Value, error)

var (
        builtins = map[string]builtin {
                `return`:       builtinReturn,
                `lit`:          builtinLit,
                //`run`:          builtinRun,
        }
)

func GetBuiltinNames() (a []string) {
        for s, _ := range builtins {
                a = append(a, s)
        }
        return
}

type returner struct {
        value types.Value
}

func (p *returner) Error() string {
        return "evaluation return"
}

func builtinReturn(ctx *Context, args... types.Value) (result types.Value, err error) {
        var value types.Value
        if x := len(args); x == 0 {
                value = args[x]
        } else {
                value = values.List(args...)
        }
        return nil, &returner{ value }
}

func builtinLit(ctx *Context, args... types.Value) (result types.Value, err error) {
        var s string
        for _, a := range args {
                s += a.Lit()
        }
        return values.String(s), nil
}

/*
func builtinRun(ctx *Context, args... types.Value) (result types.Value, err error) {
        if len(args) > 0 {
                var (
                        err error
                        name = args[0]
                        //rest = args[1:]
                        m = ctx.CurrentModule()
                        entry = m.Lookup(name.String())
                )
                if entry != nil {
                        if result, err = entry.Call(args...); err != nil {
                                //...
                        }
                }
        }
        return
} */
