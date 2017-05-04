//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "os"
        "fmt"
        "bytes"
        "testing"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testCleanFiles(t *testing.T) {
        if e := os.RemoveAll("foo"); e != nil { t.Errorf("failed remove `foo'") }
        if e := os.RemoveAll("foobar"); e != nil { t.Errorf("failed remove `foobar'") }
        if e := os.RemoveAll("o/o/o/fooo"); e != nil { t.Errorf("failed remove `o/o/o/fooo'") }
}

func testToolsetShell(t *testing.T) {
        testCleanFiles(t)

        info := new(bytes.Buffer)
        f := SetBuiltinInfoFunc(func(ctx *Context, args Items) {
                fmt.Fprintf(info, "test: %v\n", args.Expand(ctx))
        })
        defer func(){ SetBuiltinInfoFunc(f) }()
        
        ctx := Build(make(map[string]string))
        modules := ctx.GetModules()

        var (
                m *Module
                ok bool
        )
        if m, ok = modules["touch-foo"]; !ok { t.Errorf("expecting module touch-foo") }
        if m.GetName(ctx) != "touch-foo" { t.Errorf("expecting touch-foo but %v", m.GetName(ctx)) }
        if fi, e := os.Stat("foo"); fi == nil || e != nil { t.Errorf("failed: %v", e) }

        if m, ok = modules["touch-foobar"]; !ok { t.Errorf("expecting module touch-foobar") }
        if m.GetName(ctx) != "touch-foobar" { t.Errorf("expecting touch-foobar, but %v", m.GetName(ctx)) }
        if fi, e := os.Stat("foobar"); fi == nil || e != nil { t.Errorf("failed: %v", e) }

        if m, ok = modules["touch-o-o-o-foo"]; !ok { t.Errorf("expecting module touch-o-o-o-foo") }
        if m.GetName(ctx) != "touch-o-o-o-foo" { t.Errorf("expecting touch-o-o-o-foo but %v", m.GetName(ctx)) }
        if fi, e := os.Stat("o/o/o/fooo"); fi == nil || e != nil { t.Errorf("failed: %v", e) }

        if s, x := info.String(), `test: me.name: touch-foobar

test: touch-foobar: using: 
test: touch-foobar: using: touch-foo
test: touch-foobar: args: touch: start 
test: touch-foo: args: touch: start 
`; s != x { t.Errorf("%v != %v", s, x) }
        
        testCleanFiles(t)
}

func TestToolsetShell(t *testing.T) {
        RunToolsetTestCase(t, "../..", "shell", testToolsetShell)
}
