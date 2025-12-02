//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValueCache3(ctx *testcase) {
	if p := _project(ctx); p == nil {
		ctx.err("nil universe")
	} else if c := &p.filemap; c.a != nil {
		ctx.err("universe valcache : %v", c)
	} else if len(c.words) != 1 {
		ctx.err("universe valcache : %v", c)
	}
}
