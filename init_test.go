//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "path/filepath"
	"strings"
	"testing"
	"os"
)

type testcase struct { Context ; *testing.T }

const _tmodules = "/Volumes/workspace/.smart/modules"
func testHasModule(name string) (res bool) {
	if i, e := os.Stat(filepath.Join(_tmodules, name)); e == nil { res = i.IsDir() }
	return
}

func load_testcase(t *testing.T, dir, name string, ii ...interface{}) testcase {
	if !filepath.IsAbs(dir) { dir = filepath.Join(baseWorkDir, dir) }

	var ctx = init_universe(ii...) ; defer assured(ctx, false)

	ctx.workdir = dir
	ctx.globe.main = nil
	ctx.filecache = make(map[string]*filebase) // NOTE: must reset the filecache

	if false { noted(ctx, "testcase: %v %v", name, dir) }
	if tm := false; testHasModule("variant") {
		for _, s := range ctx.paths { if tm = s == _tmodules; tm { break }}
		if !tm { ctx.paths = append(ctx.paths, _tmodules) }
	}

	var s = skipint{3}
	var tc = testcase{ctx, t}

	if err := ctx.loadTopWork(); err != nil {
		erro(tc, "%v", err).debug(2)
	} else if m := ctx.globe.main; m == nil {
		erro(tc, "not loaded: %s", dir).debug(2)
	} else if name != "" && m.name != name {
		erro(tc, "main: %s <-> %s", m.name, name).debug(1, s)
	} else {
		tc.Context = closureWith(tc.Context, m.scope) // TODO: add projectContext{ctx, m}
	}

	if tc.dia().flush(); tc.dia().error() {
		tc.Errorf("%d errors in %s", tc.dia().totalErrors(), tc.Position().Filename)
	}
	return tc
}

func (tc *testcase) err(f string, i ...interface{}) {
	var ctx = tc.Context
	if i == nil { var s string
		if n := strings.Index(f, ":"); n > 0 {
			s = strings.TrimSpace(f[:n])
		} else {
			s = strings.TrimSpace(f)
		}
		if d, _ := tc.obj(s).(*def); d != nil { ctx = at(ctx, d.position) }
	} else { for _, a := range i { if d, y := a.(*def); y {
		if d != nil { ctx = at(ctx, d.position) }; break
	} else if v, y := a.(Value); y && v != nil {
		ctx = at(ctx, v.Position()); break
	}}}
	erro(ctx, f, i...).debug(1, skipint{2})
	if false { tc.Errorf(f, i...) }
}

func (tc *testcase) flush() {
	if n := tc.dia().countErrors(); n > 0 { var pos Position
		if p := tc.Project(); p != nil { pos = p.position } else { pos = tc.Position() }
		noted(at(tc.Context, pos), "%v: %v errors", tc.Project(), n).debug(1, skipint{2})
		tc.Errorf("%d errors in %s", tc.dia().flush(), pos.Filename)
	}
}

func (tc *testcase) rule(name string) (r *resolvedEntries) {
	if p := tc.Project(); p != nil { r = p.resolveEntries(tc.Context, name, false) }
	return
}

func (tc *testcase) obj(name string) (res Object) {
	if p := tc.Project(); p != nil { res = p.resolveObject(tc.Context, name) }
	return
}

func (tc *testcase) def(name string) (d *def) {
	if o := tc.obj(name); o != nil { d, _ = o.(*def) }
	return
}

func (tc *testcase) get(name string, ii ...interface{}) (res Value) {
	var d *def
	var s = skipint{2} // tRunner + testcase.get
	var a []interface{}
	for _, i := range ii { if t, y := i.(skipint); y { s.int = t.int+1 } else { a = append(a, i) } }
	if d, res = _call(tc, name, a...); d == nil {
		if false { tc.Errorf("%s: not def", name) }
		erro(tc, "%v", name).debug(1, s)
	} else if res == nil {
		if false { tc.Errorf("%s: %v", name, d.value) }
		erro(at(tc,d.position), "%v", d).debug(1, s)
		res = MakeNull(d.position)
	}
	return
}
