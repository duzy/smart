//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
        "testing"
        "os"
)

func testToolsetAndroidToolchain(t *testing.T) {
        ctx := Build(make(map[string]string))
        modules := ctx.GetModules()

        var m *Module
        var ok bool
        if m, ok = modules["toolchain"]; !ok { t.Errorf("expecting module toolchain") }
        if s, x := m.GetName(ctx), "toolchain"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.GetDir(ctx), "."; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "name"), "toolchain"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "dir"), "."; s != x { t.Errorf("%v != %v", s, x) }

        if fi, e := os.Stat("bin/arm-linux-androideabi-gcc"); fi == nil || e != nil { t.Errorf("%v", e) }
}

func TestToolsetAndroidTOOLCHAIN(t *testing.T) {
        RunToolsetTestCase(t, "../../..", "android-toolchain", testToolsetAndroidToolchain)
}
