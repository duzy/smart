//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smartest

import (
        "testing"
        "os"
)

func TestCheckCleanTest(t *testing.T) {
        if _, err := os.Stat("1.log"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("2.log"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("hello"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("hello.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("hello.cpp"); err == nil { t.Error("dirty test") }
}
