//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "testing"
        "fmt"
        "os"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testCleanFiles(t *testing.T) {
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }
        if objs, e := FindFiles(".", `\.o$`); e == nil && 0 < len(objs) {
                fmt.Printf("test: remove %v\n", objs)
                for _, s := range objs {
                        if e := os.Remove(s); e != nil {
                                t.Errorf("failed remove `%v'", s)
                        }
                }
        }
}

func testToolsetAndroidCC(t *testing.T) {
        testCleanFiles(t)

        ctx := Build(make(map[string]string))
        modules := ctx.GetModules()

        var m *Module
        var ok bool
        if m, ok = modules["na"]; !ok { t.Errorf("expecting module foo_cc_exe") }
        if s, x := m.GetName(ctx), "na"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.GetDir(ctx), "."; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "name"), "na"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "dir"), "."; s != x { t.Errorf("%v != %v", s, x) }

        if fi, e := os.Stat("na.o"); fi == nil || e != nil { t.Errorf("%v", e) }
        if fi, e := os.Stat("libna.so"); fi == nil || e != nil { t.Errorf("%v", e) }

        if s, e := Runcmd("file libna.so"); e != nil { t.Errorf("%v", e) } else {
                if s != "libna.so: ELF..." { t.Errorf("unexpected output: '%v'", s) }
        }
        
        testCleanFiles(t)
}

func TestToolsetAndroidCC(t *testing.T) {
        RunToolsetTestCase(t, "../../..", "android-cc", testToolsetAndroidCC)
}
