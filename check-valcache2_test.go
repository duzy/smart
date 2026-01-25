//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValueCache2(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{*:{+:{+:{c:{.:{0:*.c++}}}}},**:{c:{.:{0:**.c}}},?:{?:{?:{0:???}}},foo:{*:{+:{+:{c:{.:{0:foo/*.c++}}}},x:{x:{.:{0:foo/*.xx}}},y:{y:{.:{0:foo/*.yy}}},z:{z:{z:{0:foo/*zzz}}}},?:{?:{?:{?:{?:{.:{.:{c:{+:{+:{0:foo/??/???.c++}}},o:{0:foo/?????.o}}}}}}}},**:{z:{0:foo/**z}}}}`; s != t {
		note(ctx, "%s", t)
		ctx.err("%v", &p.filemap)
	}
}
