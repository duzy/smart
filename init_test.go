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
    var y bool
	var s = skipint{2} // tRunner + testcase.get
	var w facet
	var a []Value
	for _, i := range ii {
		if t, y := i.(skipint); y {
			s.int = t.int+1
		}else if t, y := i.(facet); y {
			w |= t
		} else {
			a = append(a, va(tc, i))
		}
	}

	if o := tc.Project().resolveObject(tc, name); o == nil {
		erro(tc, "%v: %s is nil", tc.Project(), name)
	} else if d, y = o.(*def); !y {
		erro(tc, "%v: %s is not def: %T", tc.Project(), name, o)
	} else if len(a) > 0 {
		if t := invoke(tc, d, w, nil, a); t != o { res = t }
	} else if d.value != nil { if res = d.value; w != 0 {
		res = res.expand(tc, w)
	}}

	if res == nil && d != nil { res = makeNull(d.position)
		if false { tc.Errorf("%s: %v", name, d.value) }
		erro(at(tc,d.position), "%v", d).debug(1, s)
	}
	return
}

func Test(t *testing.T) {
	// scanner_test.go
	t.Run("init",        testInit)
	t.Run("strings",     testStrings)
	// t.Run("integers",    testIntegers)
	// t.Run("floats",      testFloats)
	// t.Run("datetime",    testDatetime)
	// t.Run("arrays",      testArrays)
	// t.Run("maps",        testMaps)
	// t.Run("calls",       testCalls)
	// t.Run("rules",       testRules)
	// t.Run("prog",        testProgConstructs)

	// parser_test.go
	t.Run("file",        testParseFile)
	t.Run("dir",         testParseDir)

	// position_test.go
	t.Run("position",    testPositionExample)

	// builtins_test.go
	t.Run("assert",      testAssert)
	t.Run("wildcard",    testWildcard)
	t.Run("files",       testFiles)
	t.Run("auto",        testAutoContext) // value_test.go
	t.Run("foreach",     testForeach)
	t.Run("foreach",     testForeach1)
	t.Run("foreach",     testForeach2)
	t.Run("foreach",     testForeach3)
	t.Run("foreach",     testForeach4)
	t.Run("foreach",     testForeach5)
	t.Run("addprefix",   testAddPrefix)
	t.Run("pushcontext", testPushContext)
	t.Run("contains",    testContains)
	t.Run("logic",       testLogic)
	t.Run("trim-prefix", testTrimPrefix)
	t.Run("builtins",    testBuiltins)

	// template_test.go
	t.Run("template",    testTemplate)
	t.Run("template",    testTemplateForeach)

	// modifiers_test.go
	t.Run("modifier",    testValueModifier)

	// value_test.go
	// t.Run("auto",       testAutoContext)
	t.Run("value",       testValues1)
	t.Run("value",       testValues2)
	t.Run("value",       testValues3)
	t.Run("value",       testValues4)
	t.Run("value",       testValues5)
	t.Run("value",       testValues6)
	t.Run("value",       testValues7)
	t.Run("value",       testValues8)
	t.Run("value",       testValues9)
	t.Run("value",       testValues10)
	t.Run("value",       testValues11)
	t.Run("value",       testValues12)
	t.Run("value",       testValues13)
	t.Run("value",       testPlaceholders)
	t.Run("value",       testOptional)
	t.Run("value",       testGlobMatch)
	t.Run("value",       testValueGeneral)

	// defs_test.go
	t.Run("defs",        testDefs0)

	// valcache_test.go
	t.Run("valcache",    testValueCache)

	// rules_test.go
	t.Run("rules",       testRules0)
	t.Run("rules",       testRules1)

	// loader_test.go
	t.Run("load",        testLoad)
	t.Run("build",       testBuildExample)
	t.Run("load.top",    testLoadTopWork)

	// modules_test.go
	t.Run("variant",            testVariantTarget)
	t.Run("app",                testApp)
	t.Run("llvm/Config",        testLLVMConfigConfigure)
	t.Run("llvm/Config",        testLLVMConfig)
	t.Run("toolchain/booting",  testToolchainBootingConfigure)
	t.Run("toolchain/booting",  testToolchainBooting)

	// configure_test.go
	t.Run("configure.default",  testConfigureDefault)
	t.Run("configure.diverged", testConfigureDiverged)
	t.Run("configure.custom",   testConfigureCustom)
}
