//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "strings"
)

func testLoad(ctx *testcase) {
    // var l = &loader{
    //	 closureContext: closureContext{&uni, nil},
    // }
    // l.path("testdata", nil)
    // TODO: loader
}

func testBuildExample(ctx *testcase) {
    // TODO: test `testdata/example-build.smart`
}

func testLoadTopWork(ctx *testcase) {
    if !strings.HasSuffix(ctx.WorkDir(), "/testdata/none") {
        ctx.err("wrong workdir: %s", ctx.WorkDir())
    }
}
