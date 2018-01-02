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
        if _, err := os.Stat("many/hello"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("many/hello.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("many/hello.cpp"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("many/greeting/libgreeting.a"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("many/greeting/obj/greeting.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("many/greeting/src/greeting.c"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("many/greeting/include/greeting.h"); err == nil { t.Error("dirty test") }
}
