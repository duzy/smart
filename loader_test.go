//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "strings"
)

func testLoader(ctx *testcase) {
    if s := _workdir(ctx); s == "" {
        ctx.err("%s", us(ctx))
    } else if !strings.HasSuffix(s, "/testdata/empty") {
        ctx.err("%s", s)
    }
}
