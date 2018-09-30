//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package main

import (
        "extbit.io/smart/loader"
        "fmt"
        "os"
)

func main() {
        var a = os.Args[1:]

        if err := loader.AddSearchPaths(a...); err != nil {
                fmt.Fprintf(os.Stderr, "error: %s\n", err)
                return
        }

        w, _ := os.Getwd()
        l, a := loader.LoadWork()

        fmt.Printf("smart: Entering directory '%v'\n", w)

        if err := l.Run(a...); err != nil {
                fmt.Fprintf(os.Stderr, "error: %s\n", err)
        }

        fmt.Printf("smart:  Leaving directory '%v'\n", w)
}
