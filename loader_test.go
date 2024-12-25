//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "fmt"
    "regexp"
    "strings"
)

func testLoader(ctx *testcase) {
    s := _workdir(ctx)
    if s == "" {
        ctx.err("empty workdir")
    } else if !strings.HasSuffix(s, "/testdata/empty") {
        ctx.err("incorrect workdir: %s", s)
    }

    if d := ctx.def("d"); d == nil {
        ctx.err("d")
    } else if s, t := fmt.Sprintf("{=compound {=raw %s/do.smart} {=punct :} {=decimal 5} {=punct :} {=decimal 12} {=punct :}}", s), ts(d.value); s != t {
        ctx.err("%s != %s", s, t)
    }

    d := (*diagpoint)(nil)
    u := _universe(ctx)
    r := regexp.MustCompile(`^.*?/testdata/empty/do.smart:[0-9]+:[0-9]+: *here…`)
    for _, p := range u.points {
        if r.MatchString(p.message) { d = p ; break }
    }
    if false && d == nil {
        ctx.err("incorrect diagpoints (%d)", len(u.points))
    }
}
