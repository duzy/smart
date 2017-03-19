//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package main

import (
        "github.com/duzy/smart/interpreter"
        "fmt"
)

func main() {
        i := interpreter.New()
        if err := i.Load("build.smart", nil); err != nil {
                fmt.Printf("%v\n", err)
                return
        } else if err = i.Run(); err != nil {
                fmt.Printf("%v\n", err)
                return
        }
}
