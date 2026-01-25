//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValueCache3(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{foo:{*:{bar:{0:foo/*/bar}}},&:{0:&(gen)}}`; s != t {
		note(ctx, "%s", t)
		ctx.err("%v", &p.filemap)
	}
}
