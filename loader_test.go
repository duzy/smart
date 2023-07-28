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
        var ctx = load_testcase(t, "testdata/none", "none")
	if ctx.Context == nil {
		t.Errorf("fail")
		return
	}

        if !strings.HasSuffix(ctx.WorkDir(), "/testdata/none") {
                ctx.err("wrong workdir: %s", ctx.WorkDir())
        }

        ctx.flush()
}
