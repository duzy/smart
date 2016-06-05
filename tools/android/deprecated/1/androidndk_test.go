//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "path/filepath"
        "testing"
        //"fmt"
        "os"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testToolsetAndroidNDK(t *testing.T) {
        modules, moduleOrderList, moduleBuildList := GetModules(), GetModuleOrderList(), GetModuleBuildList()
        if l := len(modules); l != 0 { t.Errorf("expecting len(modules) for 0, but %v", l); return }
        if l := len(moduleOrderList); l != 0 { t.Errorf("expecting len(moduleOrderList) for 0, but %v", l); return }
        if l := len(moduleBuildList); l != 0 { t.Errorf("expecting len(moduleBuildList) for 0, but %v", l); return }
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }

        Build(ComputeTestRunParams())

        var m *Module
        var ok bool
        if m, ok = modules["foo_androidndk_so"]; !ok { t.Errorf("expecting module foo_androidndk_so"); return }
        if m.Name != "foo_androidndk_so" { t.Errorf("expecting module foo_androidndk_so, but %v", m.Name); return }
        if m.Dir != "shared" { t.Errorf("expecting dir `shared' for `%v', but %v", m.Name, m.Dir); return }
        if m.Kind != "shared" { t.Errorf("expecting shared for `foo_androidndk_so', but %v", m.Name, m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.Action.Targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.Action.Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.Action.Prequisites[0].Targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Targets[0]; s != filepath.Join("shared", "na.c.o") { t.Errorf("expect shared/na.c.o, but %v (%v)", s, m.Action.Prequisites[0].Targets); return }
        if l := len(m.Action.Prequisites[0].Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Prequisites[0].Targets[0]; s != filepath.Join("shared", "na.c") { t.Errorf("expect shared/na.c, but %v (%v)", s, m.Action.Prequisites[0].Prequisites[0].Targets); return }

        if m, ok = modules["native_app_glue"]; !ok { t.Errorf("expecting module native_app_glue"); return }
        if m.Name != "native_app_glue" { t.Errorf("expecting module native_app_glue, but %v", m.Name); return }
        if m.Dir != "native_app_glue" { t.Errorf("expecting dir `%v', but %v", m.Name, m.Dir); return }
        if m.Kind != "static" { t.Errorf("expecting static for `%v', but %v", m.Name, m.Kind); return }
        if m.Action == nil { t.Errorf("no action for the module"); return }
        if l := len(m.Action.Targets); l != 1 { t.Errorf("expection 1 targets, but %v", l); return }
        if l := len(m.Action.Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if l := len(m.Action.Prequisites[0].Targets); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Targets[0]; s != filepath.Join("native_app_glue", "android_native_app_glue.c.o") { t.Errorf("expect native_app_glue/android_native_app_glue.c.o, but %v (%v)", s, m.Action.Prequisites[0].Targets); return }
        if l := len(m.Action.Prequisites[0].Prequisites); l != 1 { t.Errorf("expecting 1 prequisite, but %v", l); return }
        if s := m.Action.Prequisites[0].Prequisites[0].Targets[0]; s != filepath.Join("native_app_glue", "android_native_app_glue.c") { t.Errorf("expect native_app_glue/android_native_app_glue.c, but %v (%v)", s, m.Action.Prequisites[0].Prequisites[0].Targets); return }

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/native_app_glue"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/native_app_glue/libnative_app_glue.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidndk_so"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidndk_so/foo_androidndk_so.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if s, ex := Runcmd("file", "out/native_app_glue/libnative_app_glue.a"), "out/native_app_glue/libnative_app_glue.a: current ar archive\n"; s != ex { t.Errorf("expectiong '%v', but: '%v'", ex, s); return }
        if s, ex := Runcmd("file", "out/foo_androidndk_so/foo_androidndk_so.so"), "out/foo_androidndk_so/foo_androidndk_so.so: ELF 32-bit LSB shared object, ARM, version 1 (SYSV), dynamically linked, not stripped\n"; s != ex { t.Errorf("expectiong '%v', but: '%v'", ex, s); return }

        os.RemoveAll("out")
}

func TestToolsetAndroidNDK(t *testing.T) {
        //runToolsetTestCase(t, "android-ndk", testToolsetAndroidNDK)
}
