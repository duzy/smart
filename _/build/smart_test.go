//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "testing"
)

func TestSplitVarArgs(t *testing.T) {
        vars, rest := splitVarArgs([]string{
                "a", "FOO=foo", "b", "BAR=bar", "c", " FOOBAR = foobar ",
        })

        if vars == nil || rest == nil { t.Errorf("vars and rest is invalid"); return }
        if s, ok := vars["FOO"]; !ok || s != "foo" { t.Errorf("FOO is incorrect: %v", s); return }
        if s, ok := vars["FOOBAR"]; !ok || s != "foobar" { t.Errorf("FOOBAR is incorrect: %v", s); return }
        if s, ok := vars["BAR"]; !ok || s != "bar" { t.Errorf("BAR is incorrect: %v", s); return }
}
