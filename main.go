//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package main

import (
        "github.com/duzy/smart/build"
        _ "github.com/duzy/smart/tools/shell"
        _ "github.com/duzy/smart/tools/cc"
        _ "github.com/duzy/smart/tools/gradle"
        _ "github.com/duzy/smart/tools/android/auto"
        _ "github.com/duzy/smart/tools/android/cc"
)

func main() {
        smart.Main()
}
