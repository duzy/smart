//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        //"github.com/duzy/smart/token"
        "fmt"
)

/* type Context interface {
        Globe() *Globe
        Scope() *Scope
}

type BuiltinFunc func(ctx Context, args... Value) Value */
type BuiltinFunc func(args... Value) Value

var builtins = map[string]BuiltinFunc {
        `print`:   builtinPrint,
        `println`: builtinPrintln,
}

func builtinPrint(/*ctx Context,*/ args... Value) Value {
        for _, a := range args {
                fmt.Printf("%s", a)
        }
        return nil
}

func builtinPrintln(/*ctx Context,*/ args... Value) Value {
        for _, a := range args {
                fmt.Printf("%s", a)
        }
        fmt.Printf("\n")
        return nil
}
