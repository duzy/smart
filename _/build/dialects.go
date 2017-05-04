//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"strings"
        //"fmt"
        //"os"
)

type dialect func(ctx *Context, script Item) Items

var (
        dialects = map[string]dialect {
                "text":     dialectText,
        }
)

func dialectText(ctx *Context, script Item) (is Items) {
        is = append(is, stringitem(script.Expand(ctx)))
        return
}
