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
	"fmt"
	"os"
)

type testcase_f1 func (*testcase)
type testcase_f2 func (*testcase, string, string)
type testcase  struct { Context ; *testing.T ; run func(testcase_f1) }
type testcase1 struct { *testcase ; i any }
type test_arg struct { name string; val any }
type test_def_1 struct{}
type test_def_2 struct{}
type test_def_3 struct{}
type test_final struct{}

const testModulesPath = "/Volumes/workspace/.smart/modules"

var init_erros int
var init_lines int

func init() {
	diagnostic_limit_erros = 1000
	diagnostic_limit_lines = 2000 // est. lines
}

func testHasModule(name string) (res bool) {
	if i, e := os.Stat(filepath.Join(testModulesPath, name)); e == nil { res = i.IsDir() }
	return
}

func loadcase(t *testing.T, dir, name string, ii ...any) (res *testcase) {
	if !filepath.IsAbs(dir) { dir = filepath.Join(workBaseDir, dir) }
	if _, e := os.Stat(dir); e != nil {
		t.Errorf("%v", e)
		return
	}

	ctx := new_universe(ii...)
	res = &testcase{ ctx, t, nil }

	defer trace(ctx)

	ctx.erros = init_erros
	ctx.flued = init_lines
	ctx.panicFailureOnErrosFlushed = false
	ctx.statcache = make(map[string]*filebase) // must reset the statcache
	ctx.testMode = true
	ctx.globe.main = nil
	ctx.workdir = dir

	if testHasModule("configure") {
		for _, s := range ctx.paths { if s == testModulesPath { goto after_app_paths }}
		ctx.paths = append(ctx.paths, testModulesPath)
	after_app_paths:
	}

	if e := ctx.load(); e != nil {
		erro(ctx, "%v", e).debug()
	} else if m := ctx.globe.main; m == nil {
		erro(ctx, "%s", dir).debug()
	} else if name != "" && m.name != name {
		erro(ctx, "project %v != %v", m.name, name).debug(1, skipint{3})
	} else {
		res.Context = closure_any(ctx, m) // TODO: projectContext{ctx, m}
		testRemoveConfigureDir(res, get_project(ctx))
	}

	ctx.diagnostic.flush(ctx)

	if ctx.erros > 0 {
		res.Errorf("%d errors in %s", ctx.erros, ctx._position().Filename)
	}
	return
}

func (tc *testcase) String() string { return ts(tc.Context) }
func (tc *testcase) ts(string) string {
	return fmt.Sprintf("{=test %v}", ts(tc.Context))
}
func (tc *testcase) do(ctx Context, op any) any {
	switch op.(type) {
	case getIsTestMode: return true
	}
	return tc.Context.do(ctx, op)
}

func (tc *testcase) err(f string, i ...any) {
	var ctx = tc.Context
	if i == nil {
		var s string
		if n := strings.Index(f, ":"); n > 0 {
			s = strings.TrimSpace(f[:n])
		} else {
			s = strings.TrimSpace(f)
		}
		if d, _ := tc.obj(s).(*def); d != nil {
			ctx = at(ctx, d.position)
		}
	} else {
		for _, a := range i {
			if d, y := a.(*def); y && d != nil {
				if d.value != nil {
					ctx = at(ctx, d.value)
				} else {
					ctx = at(ctx, d.position)
				}
				break
			} else if x, y := a.(Value); y && x != nil {
				ctx = at(ctx, x)
				break
			} else if x, y := a.(tst); y && x.i != nil {
				ctx = at(ctx, x.i)
				break
			}
		}
	}
	erro(ctx, f, i...).debug(1, skipint{2})
	flush(ctx) // to avoid affecting any other defer-traces after this err
}

func (tc *testcase) flush() {
	var dia = _diagnostic(tc.Context)
	if n := dia.counterror(); n > 0 {
		var pos Position
		if p := get_project(tc); p != nil { pos = p.position } else { pos = _position(tc) }
		note(at(tc.Context, pos), "%v: %v errors", get_project(tc), n).debug(1, skipint{2})
		tc.Errorf("%d errors in %s", dia.flush(tc.Context), pos.Filename)
	}
}

func (tc *testcase) rule(name string) (r []entry) {
	if p := get_project(tc); p != nil { r = p.resolveEntries(tc.Context, name, false) }
	return
}

func (tc *testcase) obj(name string) (res Object) {
	if p := get_project(tc); p != nil { res = p.resolve(tc.Context, name) }
	return
}

func (tc *testcase) def(name string) (d *def) {
	if o := tc.obj(name); o != nil { d, _ = o.(*def) }
	return
}

func (tc *testcase) val(i0 any, ii ...any) (res Value) {
	var x Value
	var a, o []Value
	var s = skipint{2}
	var ctx Context = tc
	var origin = defExpand1
	var proj = get_project(tc)

	for _, i := range ii {
		var vb = valbase{_position(tc)}
		switch t := i.(type) {
		case test_def_1: origin = defExpand1
		case test_def_2: origin = defExpand2
		case test_def_3: origin = defExpand3
		case test_final: ctx = final{ctx}
		case   *project: proj, ctx = t, closure_with(ctx, t.scope)
		case    skipint: s.int = t.int+1
		case       opt : o = append(o, t.Value)
		case       opts: o = append(o, t.vals...)
		case   test_arg: a = append(a, &pair{&bareword{vb,t.name},va(tc,t.val)})
		default:         a = append(a, va(tc, i))
		}
	}

	switch t := i0.(type) {
	case string:
		if x = proj.resolve(ctx, t) ; x == nil {
			erro(ctx, "%v: '%s' is nil", proj, t).debug()
			return
		}
	case  Value:
		if t == nil {
			erro(ctx, "%v: %s is nil", proj, ts(t)).debug()
			return
		}
		if x = t ; 0 < len(a) {
			ac := automatic{Context:ctx, defs:make(auto_defs)}
			ac.args(ctx, a)
			ctx = &ac
		}
	default:
		erro(ctx, "%v: %v", proj, ts(i0)).debug()
		return
	}

	c := evaluation{ctx, origin}

	if 0 < len(a) && _automatic(ctx) == nil {
		res, _ = evoke(c, x, o, a)
		return
	} else if d, y := x.(*def); y {
		if d.value != nil {
			return d.value.expand(c)
		} else {
			return
		}
	} else {
		return x.expand(c)
	}
}

func testRemoveConfigureDir(ctx *testcase, p *project) {
	if f := p.configuration; f == nil {
		// skip
	} else if s := f.fullname(); s == "" {
		ctx.err("%v", f)
	} else if !strings.HasSuffix(s, pathSep+configuration_sm) {
		ctx.err("%v %v", f, s)
	} else if s = filepath.Dir(s); s == "" {
		ctx.err("%v %v", f, s)
	} else if e := os.RemoveAll(s); e != nil {
		ctx.err("%v", e)
	} else if false {
		note(ctx, "%v", s).debug(10)
	}
	for _, base := range p.bases { testRemoveConfigureDir(ctx, base) }
}

func runcase(t *testing.T, name, spec string, f testcase_f1, ii ...any) {
	ctx := loadcase(t, joinPath("testdata", spec), name, ii...)
	ctx.run = func(f testcase_f1) { runcase(t, name, spec, f) }

	defer trace(ctx)
	defer func() {
		u := _universe(ctx)
		if u.flush(ctx) == 0 && u.erros == 0 {
			init_erros = 0
			init_lines = 0
		} else {
			init_erros += u.erros
			init_lines += u.flued
		}
	} ()

	f(ctx)
}

func va(ctx Context, i any) (v Value) {
    switch t := i.(type) {
    case   Value: v = t
    case []Value: v = makeList(t...)
    case  int:    v = makeDecimal(_position(ctx), int64(t))
    case  int16:  v = makeDecimal(_position(ctx), int64(t))
    case  int32:  v = makeDecimal(_position(ctx), int64(t))
    case  int64:  v = makeDecimal(_position(ctx), int64(t))
    case uint:    v = makeDecimal(_position(ctx), int64(t))
    case uint16:  v = makeDecimal(_position(ctx), int64(t))
    case uint32:  v = makeDecimal(_position(ctx), int64(t))
    case uint64:  v = makeDecimal(_position(ctx), int64(t))
    case string:
        if t == "" {
            v = makeNone(_position(ctx))
        } else {
            v = makeBareword(_position(ctx), t)
        }
    case []string: {
        var l = makeList()
        for _, s := range t {
            if s == "" {
                v = makeNone(_position(ctx))
            } else {
                v = makeBareword(_position(ctx), s)
            }
            l.elems = append(l.elems, v)
        }
        v = l
    }
    case []any:
        var l = makeList()
        for _, i := range t { l.elems = append(l.elems, va(ctx, i)) }
        v = l
    case nil:
        v = makeNull(_position(ctx))
    default:
        erro(ctx, "%v", ts(i)).debug(2)
    }
    return
}

func _evoke_(ctx Context, v Value, ii ...any) (res Value) {
	var a, o []Value
    for _, i := range ii {
        switch t := i.(type) {
        case    opt : o = append(o, t.Value)
        case    opts: o = append(o, t.vals...)
        case []Value: a = append(a, t...)
        case   Value: a = append(a, t)
        default     : a = append(a, va(ctx, i))
        }
    }
	res, _ = evoke(ctx, v, o, a)
    return
}

type (
	test_configure   struct { bool }
	test_caseinit    struct { f func() }
	test_hook_assert struct { f func(Context, Value, bool, any); i any }
	test_hook_debug  struct { f func(Context, string, []Value, any); i any }
	test_variant     struct { string }
	test_silentOptionalSelection struct { bool }
)

func Test(t *testing.T) {
	run := func (str, spec, name string, ii ...any) {
		t.Run(str, func (t *testing.T) {
			var c = _commandline()
			var a []any
			var d   any
			var f testcase_f1
			var _hooks *hooks
			for _, i := range ii {
				switch v := i.(type) {
				case func(*testcase): f = v // testcase_f1
				case func(*testcase, string, string): // testcase_f2
					f = func(ctx *testcase) { /* defer trace(ctx); */ v(ctx, spec, name) }
				case func(testcase):
					f = func(ctx *testcase) { /* defer trace(ctx); */ v(*ctx) }
				case func(testcase1):
					f = func(ctx *testcase) { /* defer trace(ctx); */ v(testcase1{ctx, d}) }
				case test_hook_assert:
					if _hooks == nil { _hooks = &hooks{} }
					d, _hooks.assert = v.i, func(c Context, a Value, b bool) bool {
						v.f(c, a, b, d)
						return true
					}
				case test_hook_debug:
					if _hooks == nil { _hooks = &hooks{} }
					d, _hooks.debug = v.i, func(c Context, s string, a []Value) {
						v.f(c, s, a, d)
					}
				case test_caseinit:
					v.f()
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
				if _hooks != nil { a = append(a, _hooks) }
				runcase(t, name, spec, f, append(a, c)...)
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

	// position_test.go
	t.Run("position", testPositionExample)

	// parser_test.go
	t.Run("parser", testParseFile)
	t.Run("parser", testParseDir)

	// loader_test.go
	run("loader", "empty", "testloader", testLoader)

	// value_test.go
	run("value", "value", "testvalue", testValueGeneral,
		test_hook_assert{testValueGeneralAssertHook, &testValueGeneralStruct{}})

	// builtins_test.go
	run("builtins", "assert", "testbuiltins", testAssert,
		test_hook_assert{testAssertHook, &testAssertStruct{}})
	run("builtins", "pushcontext", "testbuiltins", testPushContext)

	// value_test.go
	run("value", "value/1",           "testvalue", testValues1)
	run("value", "value/2",           "testvalue", testValues2)
	run("value", "value/auto",        "testvalue", testAutomatic)
	run("value", "value/3",           "testvalue", testValues3)
	run("value", "value/4",           "testvalue", testValues4)
	run("value", "value/5",           "testvalue", testValues5)
	run("value", "value/6",           "testvalue", testValues6)
	run("value", "value/7",           "testvalue", testValues7)
	run("value", "value/8",           "testvalue", testValues8)
	run("value", "value/9",           "testvalue", testValues9)
	run("value", "value/10",          "testvalue", testValues10)
	run("value", "value/11",          "testvalue", testValues11)
	run("value", "value/12",          "testvalue", testValues12)
	run("value", "value/13",          "testvalue", testValues13)
	run("value", "value/placeholder", "testvalue", testPlaceholders)
	run("value", "value/glob",        "testvalue", testGlob)
	run("value", "value/optional",    "testvalue", testOptional,
		test_silentOptionalSelection{true})

	// builtins_test.go
	run("builtins", "builtins/wildcard",   "testbuiltins", testBuiltin_wildcard)
	run("builtins", "builtins/foreach",    "testbuiltins", testBuiltin_foreach)
	run("builtins", "builtins/foreach/1",  "testbuiltins", testBuiltin_foreach1)
	run("builtins", "builtins/foreach/2",  "testbuiltins", testBuiltin_foreach2)
	run("builtins", "builtins/foreach/3",  "testbuiltins", testBuiltin_foreach3)
	run("builtins", "builtins/foreach/4",  "testbuiltins", testBuiltin_foreach4)
	run("builtins", "builtins/foreach/5",  "testbuiltins", testBuiltin_foreach5)
	run("builtins", "builtins/logic",      "testbuiltins", testBuiltin_logic)
	run("builtins", "builtins/addprefix",  "testbuiltins", testBuiltin_addprefix)
	run("builtins", "builtins/addsuffix",  "testbuiltins", testBuiltin_addsuffix)
	run("builtins", "builtins/contains",   "testbuiltins", testBuiltin_contains)
	run("builtins", "builtins/join",       "testbuiltins", testBuiltin_join)
	run("builtins", "builtins/or",         "testbuiltins", testBuiltin_or)
	run("builtins", "builtins/xor",        "testbuiltins", testBuiltin_xor)
	run("builtins", "builtins/trimprefix", "testbuiltins", testBuiltin_trimprefix)
	run("builtins", "builtins/trimsuffix", "testbuiltins", testBuiltin_trimsuffix)

	// template_test.go
	run("template", "template",         "testtemplate", testTemplate)

	// modifiers_test.go
	run("modifiers", "modifier", "testmodifier", testValueModifier,
		test_caseinit{testValueModifierInit})

	// defs_test.go
	run("defs", "defs", "testdefs", testDefs0)

	// valcache_test.go
	run("valcache", "valcache/1", "testvalcache", testValueCache1)
	run("valcache", "valcache/2", "testvalcache", testValueCache2)
	run("valcache", "valcache/3", "testvalcache", testValueCache3)
	run("valcache", "valcache",   "testvalcache", testValueCache)

	// builtins_test.go
	run("builtins", "builtins/file/0",     "testbuiltins", testBuiltin_file0)
	run("builtins", "builtins/file",       "testbuiltins", testBuiltin_file)

	// template_test.go
	run("template", "template/foreach", "testtemplate", testTemplateForeach)

	// rules_test.go
	run("rules", "rule/0",                "testrules", testRules0)
	run("rules", "rule/1",                "testrules", testRules1)
	run("rules", "rule/contains",         "testrules", testBuiltin_contains2)
	run("rules", "rule/shell/for-stdout", "testrules", testShellForStdout,
		test_hook_debug{testShellForStdoutDebugHook, &testShellForStdoutDebugStruct{}})

	// configure_test.go
	run("configure", "configuration",          "testdefaultconfigure",  testConfigureFoo)
	run("configure", "configuration/diverged", "testdivergedconfigure", testConfigureDivergedOuttmp)
	run("configure", "configuration/custom",   "testcustomconfigure",   testConfigureCustom)

	// modules_test.go
	run("modules", "modules/target/arm64-darwin",            "", testVariantTarget_arm64_darwin)
	run("modules", "modules/app/arm64-darwin",               "", testApp_arm64_darwin)
	run("modules", "modules/llvm/config/arm64-darwin",       "", testLLVMConfig1_arm64_darwin, test_configure{true})
	run("modules", "modules/llvm/config/arm64-darwin",       "", testLLVMConfig2_arm64_darwin)
	run("modules", "modules/toolchain/booting/arm64-darwin", "", testToolchainBooting_arm64_darwin)
}
