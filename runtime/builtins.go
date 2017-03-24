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
        //"strings"
        //"fmt"
        //"os"
)

type builtin func(ctx *Context, args... types.Value) types.Value

var (
        builtins = map[string]builtin {
                `lit`:          builtinLit,
        }
)

func builtinLit(ctx *Context, args... types.Value) types.Value {
        var s string
        for _, a := range args {
                s += a.Lit()
        }
        return values.String(s)
}
