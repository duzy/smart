//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

func testValueCache0(ctx *testcase) {
	v := _null(_position(ctx))
	m := make(map[any]string)
	m["foo"] = "foobar"
	m['f'] = "rune(f)"
	m["f"] = "string(f)"
	// m[char('f')] = "char(f)"
	m[1] = "one"
	m[v] = "value"

	if x, y := m["foo"]; !y || x != "foobar" {
		ctx.err("%v", m)
	}

	s := "foobar"[:3]
	if x, y := m[s]; !y || x != "foobar" {
		ctx.err("%v", m)
	}

	if x, y := m['f']; !y || x != "rune(f)" {
		ctx.err("%v ; %v", m, x)
	}
	// if x, y := m[char('f')]; !y || x != "char(f)" {
	// 	ctx.err("%v ; %v", m, x)
	// }
	if x, y := m["f"]; !y || x != "string(f)" {
		ctx.err("%v ; %v", m, x)
	}

	if x, y := m[1]; !y || x != "one" {
		ctx.err("%v", m)
	}

	var t Value = v
	if x, y := m[v]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if x, y := m[t]; !y || x != "value" {
		ctx.err("%v", m)
	}
	if _, y := m[_null(_position(ctx))]; y {
		ctx.err("%v", m)
	}
}

func testValueCache1(ctx *testcase) {
	testValueCache0(ctx)

	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{foo:{.:{c:{0:foo.c},c++:{0:foo.c++}},bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}}}}`; s != t {
		note(ctx, "%s", t)
		ctx.err("%v", &p.filemap)
	}
}
