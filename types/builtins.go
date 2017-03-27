//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        //"github.com/duzy/smart/token"
        "strings"
        "fmt"
)

/* type Context interface {
        Globe() *Globe
        Scope() *Scope
}

type BuiltinFunc func(ctx Context, args... Value) Value */
type BuiltinFunc func(args... Value) (Value, error)

var builtins = map[string]BuiltinFunc {
        `print`:   builtinPrint,
        `printl`:  builtinPrintl,
        `println`: builtinPrintln,
}

func builtinPrint(args... Value) (Value, error) {
        var x = len(args) - 1
        for i, a := range args {
                fmt.Printf("%s", a)
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
        }
        return nil, nil
}

func builtinPrintl(args... Value) (Value, error) {
        var x = len(args) - 1
        for i, a := range args {
                s := a.String()
                fmt.Printf("%s", s)
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                if i == x && !strings.HasSuffix(s, "\n") {
                        fmt.Printf("\n")
                }
        }
        return nil, nil
}

func builtinPrintln(args... Value) (Value, error) {
        builtinPrint(args...)
        fmt.Printf("\n")
        return nil, nil
}
