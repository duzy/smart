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
)

type testcase struct { Context ; *testing.T }

func load_testcase(t *testing.T, dir, name string) testcase {
	if !filepath.IsAbs(dir) { dir = filepath.Join(baseWorkDir, dir) }

	var ctx = &uni
	ctx.workdir = dir
	ctx.globe.main = nil

	var s = skip{3}
	var tc = testcase{ctx, t}

	if err := ctx.loadTopWork(); err != nil {
		erro(tc, "%v", err).debug(2)
	} else if m := ctx.globe.main; m == nil {
		erro(tc, "not loaded: %s", dir).debug(2)
	} else if m.name != name {
		erro(tc, "main: %s <-> %s", m.name, name).debug(1, s)
	} else {
		tc.Context = &closureContext{tc.Context, []*Scope{m.scope}} // TODO: add projectContext{ctx, m}
	}

	if tc.dia().countErrors() > 0 {
		tc.Errorf("%d errors in %s", tc.dia().flush(), tc.Position().Filename)
	}
	return tc
}

func (tc testcase) err(f string, i ...interface{}) {
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
	erro(ctx, f, i...).debug(1, skip{2})
	if false { tc.Errorf(f, i...) }
}

func (tc testcase) flush() {
	if n := tc.dia().countErrors(); n > 0 { var pos Position
		if p := tc.Project(); p != nil { pos = p.position } else { pos = tc.Position() }
		noted(at(tc.Context, pos), "%v: %v errors", tc.Project(), n).debug(1, skip{2})
		tc.Errorf("%d errors in %s", tc.dia().flush(), pos.Filename)
	}
}

func (tc testcase) obj(name string) (res Object) {
	if p := tc.Project(); p != nil { res = p.resolveObject(tc.Context, name) }
	return
}

func (tc testcase) def(name string) (d *def) {
	if o := tc.obj(name); o != nil { d, _ = o.(*def) }
	return
}

// var get = func(name string) (res Value) { return (testcase{ctx,t}.get(name, expandZero)) }
func (tc testcase) get(name string, ii ...interface{}) (res Value) {
	var d *def
	var s = skip{3}
	var a []interface{}
	for _, i := range ii { if t, y := i.(skip); y { s = t } else { a = append(a, i) } }
	if d, res = vc(tc, name, a...); d == nil {
		if false { tc.Errorf("%s: not def", name) }
		erro(tc, "%v", name).debug(1, s)
	} else if res == nil {
		if false { tc.Errorf("%s: %v", name, d.value) }
		erro(at(tc,d.position), "%v", d).debug(1, s)
		res = MakeNull(d.position)
	}
	return
}
