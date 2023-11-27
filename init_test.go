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

type testcase_f1 func (*testcase)
type testcase_f2 func (*testcase, string, string)
type testcase  struct { Context ; *testing.T ; run func(testcase_f1) }
type testcase1 struct { *testcase ; i interface{} }

const testModulesPath = "/Volumes/workspace/.smart/modules"

func testHasModule(name string) (res bool) {
	if i, e := os.Stat(filepath.Join(testModulesPath, name)); e == nil { res = i.IsDir() }
	return
}

func loadcase(t *testing.T, dir, name string, ii ...interface{}) (tc testcase) {
	if !filepath.IsAbs(dir) { dir = filepath.Join(baseWorkDir, dir) }
	if _, e := os.Stat(dir); e != nil {
		t.Errorf("%v", e)
		return
	}

	tc.T = t

	var ctx = init_universe(ii...)

	defer assured(ctx, false)

	ctx.filecache = make(map[string]*filebase) // must reset the filecache
	ctx.globe.main = nil
	ctx.workdir = dir

	if testHasModule("configure") {
		for _, s := range ctx.paths { if s == testModulesPath { goto after_app_paths }}
		ctx.paths = append(ctx.paths, testModulesPath)
	after_app_paths:
	}

	if e := ctx.load(); e != nil {
		erro(ctx, "%v", e).debug(2)
	} else if m := ctx.globe.main; m == nil {
		erro(ctx, "%s", dir).debug(2)
	} else if name != "" && m.name != name {
		erro(ctx, "project %v != %v", m.name, name).debug(1, skipint{3})
	} else {
		tc.Context = _closureWith(ctx, m) // TODO: add projectContext{ctx, m}
		testRemoveConfigureDir(tc, tc.Project())
	}

	var dia = _diaContext(tc.Context)
	if dia.flush(); dia.error() {
		tc.Errorf("%d errors in %s", dia.totalErrors(), tc.Position().Filename)
	}
	return
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
	var dia = _diaContext(tc.Context)
	if n := dia.countErrors(); n > 0 { var pos Position
		if p := tc.Project(); p != nil { pos = p.position } else { pos = tc.Position() }
		noted(at(tc.Context, pos), "%v: %v errors", tc.Project(), n).debug(1, skipint{2})
		tc.Errorf("%d errors in %s", dia.flush(), pos.Filename)
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
	var w facet
	var a, o []Value
	var s = skipint{2} // tRunner + testcase.get
	var ctx Context = tc
	var proj = tc.Project()
	for _, i := range ii {
		switch t := i.(type) {
		case *Project: proj, ctx = t, closureWith(ctx, t.scope)
		case  skipint: s.int = t.int+1
		case    facet: w |= t
		case     opt : o = append(o, t.Value)
		case     opts: o = append(o, t.vals...)
		default:       a = append(a, va(tc, i))
		}
	}

	if t := proj.resolve(ctx, name); t == nil {
		erro(ctx, "%v: %s is nil", proj, name)
	} else if d, y = t.(*def); !y {
		erro(ctx, "%v: %s is not def: %v", proj, name, typeof(t))
	} else if len(a) > 0 {
		res, _, _ = evoke(ctx, d, w, o, a)
	} else if d.value == nil {
		// nil
	} else if w != 0 {
		res = d.value.expand(ctx, w)
	} else {
		res = d.value
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

func runcase(t *testing.T, spec, name string, f testcase_f1, ii ...interface{}) {
	var ctx = loadcase(t, joinPath("testdata", spec), name, ii...)
	if ctx.Context == nil {
		t.Errorf("testdata/%s", spec)
		return
	}

	ctx.run = func(f testcase_f1) { runcase(t, spec, name, f) }

	defer ctx.flush()
	defer assured(ctx, true)

	f(&ctx)
}

type (
	test_casedata    struct { i interface{} }
	test_caseinit    struct { f func() }
	test_configure   struct { bool }
	test_hook_assert struct { f func(Context, Value, bool, interface{}) }
	test_variant     struct { string }
	test_silentOptionalSelection struct { bool }
)

func Test(t *testing.T) {
	run := func (str, spec, name string, ii ...interface{}) {
		t.Run(str, func (t *testing.T) {
			var a []interface{}
			var d   interface{}
			var f testcase_f1
			var _hooks *hooks
			var c = init_commandline()
			for _, i := range ii {
				switch v := i.(type) {
				case func(*testcase): f = v // testcase_f1
				case func(*testcase, string, string): // testcase_f2
					f = func(ctx *testcase) { v(ctx, spec, name) }
				case func(testcase):
					f = func(ctx *testcase) { v(*ctx) }
				case func(testcase1):
					f = func(ctx *testcase) { v(testcase1{ctx, d}) }
				case test_hook_assert:
					if _hooks == nil { _hooks = &hooks{} }
					_hooks.assert = func(c Context, a Value, b bool) bool {
						v.f(c, a, b, d)
						return true
					}
				case test_caseinit:
					v.f()
				case test_casedata:
					d = v.i
				case test_configure:
					c.configure = v.bool
				case test_silentOptionalSelection:
					c.silentOptionalSelection = v.bool
				case test_variant:
					t.Errorf("TODO: variant=%s", v.string)
				default:
					a = append(a, v)
				}
			}
			if f == nil {
				t.Errorf("%v: %v", str, ii)
			} else {
				if _hooks != nil { a = append(a, *_hooks) }
				runcase(t, spec, name, f, append(a,c)...)
			}
		})
	}

	loader_sources_bench = false

	// scanner_test.go
	t.Run("scanner", testInit)
	t.Run("scanner", testStrings)
	// t.Run("scanner", testIntegers)
	// t.Run("scanner", testFloats)
	// t.Run("scanner", testDatetime)
	// t.Run("scanner", testArrays)
	// t.Run("scanner", testMaps)
	// t.Run("scanner", testCalls)
	// t.Run("scanner", testRules)
	// t.Run("scanner", testProgConstructs)
	// parser_test.go
	t.Run("parser",  testParseFile)
	t.Run("parser",  testParseDir)
	// position_test.go
	t.Run("position", testPositionExample)

	// builtins_test.go
	run("assert", "assert", "testassert", testAssert,
		test_casedata{&testAssertStruct{}},
		test_hook_assert{testAssertHook})
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
	run("modifier", "modifier", "testmodifier", testValueModifier,
		test_caseinit{testValueModifierInit})

	// value_test.go
	// run("auto", "value/0", "testvalues0", testAutoContext) // value_test.go
	run("value", "value/1",           "testvalues1",     testValues1)
	run("value", "value/2",           "testvalues2",     testValues2)
	run("value", "value/3",           "testvalues3",     testValues3)
	run("value", "value/4",           "testvalues4",     testValues4)
	run("value", "value/5",           "testvalues5",     testValues5)
	run("value", "value/6",           "testvalues6",     testValues6)
	run("value", "value/7",           "testvalues7",     testValues7)
	run("value", "value/8",           "testvalues8",     testValues8)
	run("value", "value/9",           "testvalues9",     testValues9)
	run("value", "value/10",          "testvalues10",    testValues10)
	run("value", "value/11",          "testvalues11",    testValues11)
	run("value", "value/12",          "testvalues12",    testValues12)
	run("value", "value/13",          "testvalues13",    testValues13)
	run("value", "value/placeholder", "testplaceholder", testPlaceholders)
	run("value", "value/glob",        "testglobmatch",   testGlobMatch)
	run("value", "value/optional",    "testoptional",    testOptional,
		test_silentOptionalSelection{true})
	run("value", "value",             "testvalues",      testValueGeneral,
		test_casedata{&testValueGeneralStruct{}},
		test_hook_assert{testValueGeneralAssertHook})

	// defs_test.go
	run("defs", "defs", "testdefs", testDefs0)

	// valcache_test.go
	run("valcache", "valcache", "valcache", testValueCache)

	// rules_test.go
	run("rules", "rule/0", "testrules0", testRules0)
	run("rules", "rule/1", "testrules1", testRules1)

	// loader_test.go
	run("load", "none", "none", testLoader)

	// configure_test.go
	run("configure", "configuration",          "testdefaultconfigure",  testConfigureFoo)
	run("configure", "configuration/diverged", "testdivergedconfigure", testConfigureDivergedOuttmp)
	run("configure", "configuration/custom",   "testcustomconfigure",   testConfigureCustom)

	// modules_test.go
	run("variant",           "modules/target/arm64-darwin",            "", testVariantTarget_arm64_darwin)
	run("app",               "modules/app/arm64-darwin",               "", testApp_arm64_darwin)
	run("llvm/Config",       "modules/llvm/config/arm64-darwin",       "", testLLVMConfig1_arm64_darwin, test_configure{true})
	run("llvm/Config",       "modules/llvm/config/arm64-darwin",       "", testLLVMConfig2_arm64_darwin)
	run("toolchain/booting", "modules/toolchain/booting/arm64-darwin", "", testToolchainBooting_arm64_darwin)
}
