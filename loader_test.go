//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
        "testing"
        "strings"
)

func TestLoad(t *testing.T) {
        // var l = &loader{
        //	 closureContext: closureContext{&uni, nil},
        // }
        // l.path("testdata", nil)
        // TODO: loader
}

func TestBuildExample(t *testing.T) {
        // TODO: test `testdata/example-build.smart`
}

func TestLoadTopWork(t *testing.T) {
        var ctx = confine("testdata/none")

        if !strings.HasSuffix(ctx.WorkDir(), "/testdata/none") {
                t.Errorf("wrong workdir: %s", ctx.WorkDir())
        } else if err := ctx.loadTopWork(); err != nil {
                t.Errorf("%v", err)
        } else if n := ctx.countErrors(); n > 0 {
                t.Errorf("errors %v, base=%s", n, ctx.WorkDir())
	} else if m := ctx.globe.main; m == nil {
		t.Errorf("nil main")
        } else if m.name != "none" {
		t.Errorf("wrong main: %v", m)
        }

        if n := ctx.flushDiags(); n > 0 { t.Errorf("errors %d", n) }
}
