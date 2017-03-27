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

func EscapedString(v Value) (s string) {
        if v.Type() == String {
                s = strings.Replace(v.String(), "\\'", "'", -1)
        } else {
                s = v.String()
        }
        return
}

func builtinPrint(args... Value) (Value, error) {
        var x = len(args)
        for i, a := range args {
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                fmt.Printf("%s", EscapedString(a))
        }
        return nil, nil
}

func builtinPrintl(args... Value) (Value, error) {
        var x = len(args)
        for i, a := range args {
                if 0 < i && i < x {
                        fmt.Printf(" ")
                }
                s := EscapedString(a)
                fmt.Printf("%s", s)
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
