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
        if _, err := os.Stat("alt/foo.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("alt/other.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/1.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/2.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/3.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/foo.1.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/fo.o.2.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/f.o.o.3.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/f.o.o.4.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/foo.5.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/foo.6.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("obj/main.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("src/baz1.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("src/baz2.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("src/baz3.o"); err == nil { t.Error("dirty test") }
        if _, err := os.Stat("src/baz4.o"); err == nil { t.Error("dirty test") }
}
