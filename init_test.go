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

func loadcase(t *testing.T, dir, name string, ii ...interface{}) testcase {
	if !filepath.IsAbs(dir) { dir = filepath.Join(baseWorkDir, dir) }
	if _, e := os.Stat(dir); e != nil {
		t.Errorf("%v", e)
		return testcase{}
	}

	var ctx = init_universe(ii...)

	defer assured(ctx, false)

	ctx.workdir = dir
	ctx.globe.main = nil
	ctx.filecache = make(map[string]*filebase) // NOTE: must reset the filecache

	if testHasModule("variant") { var tm bool
		for _, s := range ctx.paths { if tm = s == _tmodules; tm { break }}
		if !tm { ctx.paths = append(ctx.paths, _tmodules) }
	}

	var tc = testcase{ctx, t}

	if e := ctx.loadTopWork(); e != nil {
		erro(ctx, "%v", e).debug(2)
	} else if m := ctx.globe.main; m == nil {
		erro(ctx, "%s", dir).debug(2)
	} else if name != "" && m.name != name {
		erro(ctx, "%v <-> %v", m.name, name).debug(1, skipint{3})
	} else {
		tc.Context = closureWith(tc.Context, m.scope) // TODO: add projectContext{ctx, m}
		testRemoveConfigureDir(tc, tc.Project())
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
	if p := tc.Project(); p != nil { res = p.resolve(tc.Context, name) }
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

	if o := tc.Project().resolve(tc, name); o == nil {
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

func testRemoveConfigureDir(ctx testcase, p *Project) {
	if f := p.configurationFile; f == nil {
		// skip
	} else if s := f.fullname(); s == "" {
		ctx.err("%v", f)
	} else if !strings.HasSuffix(s, PathSep+configuration_sm) {
		ctx.err("%v %v", f, s)
	} else if s = filepath.Dir(s); s == "" {
		ctx.err("%v %v", f, s)
	} else if e := os.RemoveAll(s); e != nil {
		ctx.err("%v", e)
	} else if false {
		noted(ctx, "%v", s).debug(10)
	}
	for _, base := range p.bases { testRemoveConfigureDir(ctx, base) }
}

type testcase_f1 func (*testcase)
type testcase_f2 func (*testcase, string, string)

func runcase(t *testing.T, spec, name string, f testcase_f1, ii ...interface{}) {
	var ctx = loadcase(t, spec, name, ii...)
	if ctx.Context == nil {
		t.Errorf(spec)
		return
	}

	defer ctx.flush()
	defer assured(ctx, true)

	f(&ctx)
}

func Test(t *testing.T) {
	run := func (str, spec, name string, i interface{}, b ...bool) {
		t.Run(str, func (t *testing.T) {
			var a []interface{}
			if len(b) > 0 && b[0] {
				c := init_commandline()
				c.configure = true
				a = append(a, c)
			}

			spec = "testdata/" + spec

			var f testcase_f1
			switch v := i.(type) {
			case func(*testcase): f = v // testcase_f1
			case func(*testcase, string, string): // testcase_f2
				f = func(ctx *testcase) { v(ctx, spec, name) }
			default: t.Errorf("%T", i) ; return
			}

			runcase(t, spec, name, f, a...)
		})
	}

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
	t.Run("assert", testAssert)
	run("wildcard", "wildcard", "testwildcard", testWildcard)
	run("files", "files", "testfiles", testFiles)
	run("auto", "value/0", "testvalues0", testAutoContext) // value_test.go
	run("foreach", "foreach",   "testforeach",  testForeach)
	run("foreach", "foreach/1", "testforeach1", testForeach1)
	run("foreach", "foreach/2", "testforeach2", testForeach2)
	run("foreach", "foreach/3", "testforeach3", testForeach3)
	run("foreach", "foreach/4", "testforeach4", testForeach4)
	run("foreach", "foreach/5", "testforeach5", testForeach5)
	run("addprefix", "addprefix", "testaddprefix", testAddPrefix)
	run("pushcontext", "pushcontext", "pushcontext", testPushContext)
	run("contains", "contains", "testcontains", testContains)
	run("logic", "logic", "testlogic", testLogic)
	run("trim-prefix", "trimprefix", "testtrimprefix", testTrimPrefix)
	run("builtins", "builtins", "testbuiltins", testBuiltins)

	// template_test.go
	run("template", "template",         "testtemplate", testTemplate)
	run("template", "template/foreach", "testtemplate", testTemplateForeach)

	// modifiers_test.go
	{   testValueModifierInit()
		run("modifier", "modifier", "testmodifier", testValueModifier)
	}

	// value_test.go
	// run("auto", "value/0", "testvalues0", testAutoContext) // value_test.go
	run("value", "value/1",  "testvalues1",  testValues1)
	run("value", "value/2",  "testvalues2",  testValues2)
	run("value", "value/3",  "testvalues3",  testValues3)
	run("value", "value/4",  "testvalues4",  testValues4)
	run("value", "value/5",  "testvalues5",  testValues5)
	run("value", "value/6",  "testvalues6",  testValues6)
	run("value", "value/7",  "testvalues7",  testValues7)
	run("value", "value/8",  "testvalues8",  testValues8)
	run("value", "value/9",  "testvalues9",  testValues9)
	run("value", "value/10", "testvalues10", testValues10)
	run("value", "value/11", "testvalues11", testValues11)
	run("value", "value/12", "testvalues12", testValues12)
	run("value", "value/13", "testvalues13", testValues13)
	run("value", "value/placeholder", "testplaceholder", testPlaceholders)
	t.Run("value",       testOptional)
	t.Run("value",       testGlobMatch)
	t.Run("value",       testValueGeneral)

	// defs_test.go
	run("defs", "defs", "testdefs", testDefs0)

	// valcache_test.go
	run("valcache", "valcache", "valcache", testValueCache)

	// rules_test.go
	run("rules", "rule/0", "testrules0", testRules0)
	run("rules", "rule/1", "testrules1", testRules1)

	// loader_test.go
	// run("load", , , testLoad)
	// run("build", , , testBuildExample)
	run("load", "none", "none", testLoadTopWork)

	// configure_test.go
	run("configure", "configuration",          "testdefaultconfigure",  testConfigure1)
	run("configure", "configuration/diverged", "testdivergedconfigure", testConfigure2)
	run("configure", "configuration/custom",   "testcustomconfigure",   testConfigure3)

	// modules_test.go
	run("variant", "modules/target", "testtarget", testVariantTarget)
	run("app", "modules/app", "testapp", testApp)
	run("llvm/Config", "modules/llvm/config", "testllvmconfig", testLLVMConfig1, true)
	run("llvm/Config", "modules/llvm/config", "testllvmconfig", testLLVMConfig2)
	run("toolchain/booting", "modules/toolchain/booting", "testtoolchainbooting",  testToolchainBooting)
}
