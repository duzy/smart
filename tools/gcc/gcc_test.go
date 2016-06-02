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

func testToolsetGcc(t *testing.T) {
        testCleanFiles(t)

        ctx := Build(make(map[string]string))
        modules := ctx.GetModules()

        var m *Module
        var ok bool
        if m, ok = modules["foo_gcc_exe"]; !ok { t.Errorf("expecting module foo_gcc_exe") }
        if s, x := m.GetName(ctx), "foo_gcc_exe"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.GetDir(ctx), "exe"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "name"), "foo_gcc_exe"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "dir"), "exe"; s != x { t.Errorf("%v != %v", s, x) }

        if m, ok = modules["foo_shared"]; !ok { t.Errorf("expecting module foo_shared") }
        if s, x := m.GetName(ctx), "foo_shared"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.GetDir(ctx), "shared"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "name"), "foo_shared"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "dir"), "shared"; s != x { t.Errorf("%v != %v", s, x) }

        if m, ok = modules["foo_static"]; !ok { t.Errorf("expecting module foo_static") }
        if s, x := m.GetName(ctx), "foo_static"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.GetDir(ctx), "static"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "name"), "foo_static"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "dir"), "static"; s != x { t.Errorf("%v != %v", s, x) }

        if m, ok = modules["foo_gcc_exe_use_shared"]; !ok { t.Errorf("expecting module foo_gcc_exe_use_shared") }
        if s, x := m.GetName(ctx), "foo_gcc_exe_use_shared"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.GetDir(ctx), "exe_use_shared"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "name"), "foo_gcc_exe_use_shared"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "dir"), "exe_use_shared"; s != x { t.Errorf("%v != %v", s, x) }

        if m, ok = modules["foo_gcc_exe_use_static"]; !ok { t.Errorf("expecting module foo_gcc_exe_static") }
        if s, x := m.GetName(ctx), "foo_gcc_exe_use_static"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.GetDir(ctx), "exe_use_static"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "name"), "foo_gcc_exe_use_static"; s != x { t.Errorf("%v != %v", s, x) }
        if s, x := m.Get(ctx, "dir"), "exe_use_static"; s != x { t.Errorf("%v != %v", s, x) }

        if fi, e := os.Stat("shared/libfoo_shared.so"); fi == nil || e != nil { t.Errorf("%v", e) }
        if fi, e := os.Stat("static/libfoo_static.a"); fi == nil || e != nil { t.Errorf("%v", e) }
        if fi, e := os.Stat("exe/foo_gcc_exe"); fi == nil || e != nil { t.Errorf("%v", e) }
        if fi, e := os.Stat("exe_use_static/foo_gcc_exe_use_static"); fi == nil || e != nil { t.Errorf("%v", e) }
        if fi, e := os.Stat("exe_use_shared/foo_gcc_exe_use_shared"); fi == nil || e != nil { t.Errorf("%v", e) }

        if s, e := Runcmd("exe/foo_gcc_exe"); e != nil { t.Errorf("%v", e) } else {
                if s != "hello: exe/foo_gcc_exe\n" { t.Errorf("unexpected output: '%v'", s) }
        }
        if s, e := Runcmd("exe_use_static/foo_gcc_exe_use_static"); e != nil { t.Errorf("%v", e) } else {
                if s != "hello: exe_use_static/foo_gcc_exe_use_static (static: 100)\n" { t.Errorf("unexpected output: '%v'", s) }
        }
        if s, e := Runcmd("exe_use_shared/foo_gcc_exe_use_shared"); e != nil { t.Errorf("%v", e) } else {
                if s != "hello: exe_use_shared/foo_gcc_exe_use_shared (shared: 100)\n" { t.Errorf("unexpected output: '%v'", s) }
        }

        testCleanFiles(t)
}

func TestToolsetGCC(t *testing.T) {
        RunToolsetTestCase(t, "../..", "gcc", testToolsetGcc)
}
