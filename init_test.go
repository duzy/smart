//
//  Copyright (C) 2012-2023, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "path/filepath"
	"reflect"
	"strings"
	"testing"
	"fmt"
	"os"
)

type testcase_f1 func (*testcase)
type testcase_f2 func (*testcase, string, string)
type testcase   struct{ Context ; *testing.T ; run func(testcase_f1) ; srcs, chks map[string]struct{} }
type testcase1  struct{ *testcase ; i any }
type test_arg   struct{ name string; val any }
type test_def_1 struct{}
type test_def_2 struct{}
type test_def_3 struct{}
type test_final struct{}

const testModulesPath = "/Volumes/workspace/.smart/modules"

var total_erros int
var total_bytes int
var test_mode bool

func init() {
	diagnostic_limit_erros = 1000
	diagnostic_limit_bytes = 2000 // est. lines
	if _, e := os.Stat("/Volumes/workspace"); e == nil {
		if _, e := os.Stat("/Volumes/workout"); e == nil {
			test_mode = true
		}
	}
}

func testHasModule(name string) (res bool) {
	if i, e := os.Stat(filepath.Join(testModulesPath, name)); e == nil { res = i.IsDir() }
	return
}

type loaderros struct{ string; int }
func (t loaderros) String() string {
	return fmt.Sprintf("%s: %d errors", t.string, t.int)
}

func loadcase(t *testing.T, dir, spec, name string, ii ...any) (res *testcase) {
	if !filepath.IsAbs(dir) { dir = filepath.Join(workBaseDir, dir) }
	if _, e := os.Stat(dir); e != nil {
		t.Errorf("%v", e)
		return
	}

	ctx := new_universe(ii...)
	ctx.erros = total_erros
	ctx.flushed = total_bytes
	ctx.panicFailureOnErrosFlushed = false
	ctx.statcache = make(map[string]*filebase) // must reset the statcache
	ctx.globe.main = nil
	ctx.workdir = dir

	defer func() {
		if ctx.diagnostic.flush(ctx) ; ctx.erros > 0 {
			var s = name
			if s == "" { s = spec }
			if s == "" { s = dir }
			panic(loaderros{s, ctx.erros})
		}
	} ()

	if true && !test_mode {
		erro(ctx, "not test mode").trace()
	}

	if testHasModule("configure") && !ctx.paths.has(testModulesPath) {
		ctx.paths = append(ctx.paths, testModulesPath)
	}

	res = &testcase{ctx, t, nil, make(map[string]struct{}), make(map[string]struct{})}

	if e := ctx.load(res); e != nil {
		erro(ctx, "%v", e).trace()
	} else if m := ctx.globe.main; m == nil {
		erro(ctx, "%s", dir).trace()
	} else if name != "" && m.name != name {
		erro(ctx, "project %v != %v", m.name, name).trace()
	} else {
		res.Context = closure_with(ctx, m) // TODO: projectContext{ctx, m}
		if false { testRemoveConfigureDir(res, _project(ctx)) }
	}
	return
}

func (tc *testcase) String() string { return ts(tc.Context) }
func (tc *testcase) ts(string) string { return "{=test "+ts(tc.Context)+"}" }
func (tc *testcase) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_test_mode: return test_mode
	case silent_configure: return true
	case source_loaded:
		tc.srcs[string(t)] = struct{}{}
		return
	case source_checked:
		tc.chks[string(t)] = struct{}{}
		return
	case get_position:
		if p := _project(ctx); p != nil {
			return p.position
		} else {
			var p Position = _position(tc.Context)
			if !p.valid() { p.Filename = _universe(tc.Context).workdir }
			return p
		}
	}
	return tc.Context.do(ctx, op)
}

func (tc *testcase) err(f string, i ...any) {
	var ctx Context = tc
	if i == nil {
		var s string
		if n := strings.Index(f, ":"); n > 0 {
			s = strings.TrimSpace(f[:n])
		} else {
			s = strings.TrimSpace(f)
		}
		if o := tc.obj(s); o != nil {
			ctx = pc(ctx, o)
		}
	} else {
		for _, a := range i {
			if x, y := a.(positioner); y {
				ctx = pc(ctx,x)
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
		if p := _project(tc); p != nil { pos = p.position } else { pos = _position(tc) }
		note(tc.Context, "%v: %v errors", _project(tc), n).debug(1, skipint{2})
		tc.Errorf("%d errors in %s", dia.flush(tc.Context), pos.Filename)
	}
}

func (tc *testcase) rule(name string) (r []entry) {
	if p := _project(tc); p != nil { r = p._entries(tc.Context, name, false) }
	return
}

func (tc *testcase) obj(name string) (res object) {
	if p := _project(tc); p != nil { res = p.resolve(tc.Context, name) }
	return
}

func (tc *testcase) def(name string) (d *def) {
	if o := tc.obj(name); o != nil { d, _ = o.(*def) }
	return
}

func (tc *testcase) str(a any, b ...any) (_ string) {
	if v := tc.val(a, b...); v != nil { return v.string(tc) }
	return
}

func (tc *testcase) val(i0 any, ii ...any) (res Value) {
	var x Value
	var a, o []Value
	var s = skipint{2}
	var ctx Context = tc
	var origin = defExpand1
	var proj = _project(tc)

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
		case   test_arg: a = append(a, &pair{&word{vb,t.name},va(tc,t.val)})
		default:         a = append(a, va(tc, i))
		}
	}

	switch t := i0.(type) {
	case string:
		if x = proj.resolve(ctx, t) ; x == nil {
			note(ctx, "%v: %v", proj, reflect.ValueOf(proj.scope.elems).MapKeys())
			erro(ctx, "%v: '%s' is nil", proj, t).trace()
		}
	case  Value:
		if x = t ; t == nil {
			erro(ctx, "%v: %s is nil", proj, ts(t)).trace()
		}
	default:
		erro(ctx, "%v: %v", proj, ts(i0)).trace()
	}

	var c = original{ctx, origin}

	if d, y := x.(*def) ; y && 0 < len(a) {
		res, _ = evoke(c, x, o, a)
		return
	} else if y {
		if d.value != nil {
			return d.value.expand(c)
		} else {
			return
		}
	} else {
		if 0 < len(a) {
			ac := automatic{Context:c.Context, defs:make(defs_map)}
			ac.args(ctx, a)
			c.Context = &ac
		}
		return x.expand(c)
	}
}

func testRemoveConfigureDir(ctx *testcase, p *project) {
	if f := p.configuration_sm(ctx); f == nil {
		// skip
	} else if s := f.fullname(); s == "" {
		ctx.err("%v", f)
	} else if !strings.HasSuffix(s, pathSep+configuration_sm) {
		ctx.err("%v %v", f, s)
	} else if s = filepath.Dir(s); s == "" {
		ctx.err("%v %v", f, s)
	} else if e := os.RemoveAll(s); e != nil {
		ctx.err("%v", e)
	}
	for _, base := range p.bases { testRemoveConfigureDir(ctx, base) }
}

func runcase(t *testing.T, name, spec string, f testcase_f1, ii ...any) {
	ctx := loadcase(t, joinpath("testdata", spec), spec, name, ii...)
	ctx.run = func(f testcase_f1) { runcase(t, name, spec, f) }

	defer func() {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case prerequisite_evoke_loop:
				errostack(pc(ctx,e.Value), 16, "%v", e.Value).trace()
			case trace_err_evoke_loop:
				errostack(pc(ctx,e.Value), 16, "evoke loop: %v", e.Value).trace()
			case traverse_state:
				switch e.uint {
				case traverse_done:
				default:
					errostack(pc(ctx,e.p), 16, "%v", tv(e)).trace()
				}
			default:
				errostack(ctx, 16, "%v", tv(e))//.trace()
				panic(e) // continues unwind
			}
		}

		d := _diagnostic(ctx)
		d.flush(ctx)

		if true { return }

		if d.erros == 0 {
			total_erros  = 0
		} else {
			total_erros += d.erros
		}
		total_bytes += d.flushed
	} ()

	f(ctx)
}

func run(t *testing.T, str, spec, name string, ii ...any) {
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
				f = func(ctx *testcase) { v(ctx, spec, name) }
			case func(testcase):
				f = func(ctx *testcase) { v(*ctx) }
			case func(testcase1):
				f = func(ctx *testcase) { v(testcase1{ctx, d}) }
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
	case   bare:  v = makeWord(_position(ctx), string(t))
    case string:
        if t == "" {
            v = makeNone(_position(ctx))
        } else {
            v = makeWord(_position(ctx), t)
        }
    case []string:
        var l = makeList()
        for _, s := range t {
            if s == "" {
                v = makeNone(_position(ctx))
            } else {
                v = makeWord(_position(ctx), s)
            }
            l.elems = append(l.elems, v)
        }
        v = l
    case []any:
        var l = makeList()
        for _, i := range t { l.elems = append(l.elems, va(ctx, i)) }
        v = l
    case nil:
        v = makeNull(_position(ctx))
    default:
        erro(ctx, "%v", ts(i)).trace()
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
	run(t, "loader", "empty", "testloader", testLoader)

	// value_test.go
	run(t, "value", "value", "testvalue", testValueGeneral, test_hook_assert{testValueGeneralAssertHook, &testValueGeneralStruct{}})

	// builtins_test.go
	run(t, "builtins", "assert", "testbuiltins", testAssert, test_hook_assert{testAssertHook, &testAssertStruct{}})
	run(t, "builtins", "pushcontext", "testbuiltins", testPushContext)

	// value_test.go
	run(t, "value", "value/closure",     "testvalue", testClosure)
	run(t, "value", "value/1",           "testvalue", testValues1)
	run(t, "value", "value/2",           "testvalue", testValues2)
	run(t, "value", "value/auto",        "testvalue", testAutomatic)
	run(t, "value", "value/3",           "testvalue", testValues3)
	run(t, "value", "value/4",           "testvalue", testValues4)
	run(t, "value", "value/5",           "testvalue", testValues5)
	run(t, "value", "value/6",           "testvalue", testValues6)
	run(t, "value", "value/7",           "testvalue", testValues7)
	run(t, "value", "value/8",           "testvalue", testValues8)
	run(t, "value", "value/9",           "testvalue", testValues9)
	run(t, "value", "value/10",          "testvalue", testValues10)
	run(t, "value", "value/11",          "testvalue", testValues11)
	run(t, "value", "value/12",          "testvalue", testValues12)
	run(t, "value", "value/13",          "testvalue", testValues13)
	run(t, "value", "value/placeholder", "testvalue", testPlaceholders)
	run(t, "value", "value/glob",        "testvalue", testGlob)
	run(t, "value", "value/optional",    "testvalue", testOptional, test_silentOptionalSelection{true})

	// builtins_test.go
	run(t, "builtins", "builtins/wildcard",   "testbuiltins", testBuiltin_wildcard)
	run(t, "builtins", "builtins/foreach",    "testbuiltins", testBuiltin_foreach)
	run(t, "builtins", "builtins/foreach/1",  "testbuiltins", testBuiltin_foreach1)
	run(t, "builtins", "builtins/foreach/2",  "testbuiltins", testBuiltin_foreach2)
	run(t, "builtins", "builtins/foreach/3",  "testbuiltins", testBuiltin_foreach3)
	run(t, "builtins", "builtins/foreach/4",  "testbuiltins", testBuiltin_foreach4)
	run(t, "builtins", "builtins/foreach/5",  "testbuiltins", testBuiltin_foreach5)
	run(t, "builtins", "builtins/logic",      "testbuiltins", testBuiltin_logic)
	run(t, "builtins", "builtins/addprefix",  "testbuiltins", testBuiltin_addprefix)
	run(t, "builtins", "builtins/addsuffix",  "testbuiltins", testBuiltin_addsuffix)
	run(t, "builtins", "builtins/contains",   "testbuiltins", testBuiltin_contains)
	run(t, "builtins", "builtins/join",       "testbuiltins", testBuiltin_join)
	run(t, "builtins", "builtins/or",         "testbuiltins", testBuiltin_or)
	run(t, "builtins", "builtins/xor",        "testbuiltins", testBuiltin_xor)
	run(t, "builtins", "builtins/trimprefix", "testbuiltins", testBuiltin_trimprefix)
	run(t, "builtins", "builtins/trimsuffix", "testbuiltins", testBuiltin_trimsuffix)

	// template_test.go
	run(t, "template", "template",         "testtemplate", testTemplate)

	// modifiers_test.go
	run(t, "modifiers", "modifier", "testmodifier", testValueModifier, test_caseinit{testValueModifierInit})

	// defs_test.go
	run(t, "defs", "defs", "testdefs", testDefs0)

	// valcache_test.go
	run(t, "valcache", "valcache/1", "testvalcache", testValueCache1)
	run(t, "valcache", "valcache/2", "testvalcache", testValueCache2)
	run(t, "valcache", "valcache/3", "testvalcache", testValueCache3)
	run(t, "valcache", "valcache",   "testvalcache", testValueCache)

	// builtins_test.go
	run(t, "builtins", "builtins/file/0",     "testbuiltins", testBuiltin_file0)
	run(t, "builtins", "builtins/file",       "testbuiltins", testBuiltin_file)

	// template_test.go
	run(t, "template", "template/foreach", "testtemplate", testTemplateForeach)

	// rules_test.go
	run(t, "rules", "rule/0",                "testrules", testRules0)
	run(t, "rules", "rule/1",                "testrules", testRules1)
	run(t, "rules", "rule/contains",         "testrules", testBuiltin_contains2)
	run(t, "rules", "rule/shell/for-stdout", "testrules", testShellForStdout, test_hook_debug{testShellForStdoutDebugHook, &testShellForStdoutDebugStruct{}})

	// configure_test.go
	run(t, "configure", "configuration",        "testdefaultconfigure", testConfigureDefault)
	run(t, "configure", "configuration/two",    "testdeftwoconfigure",  testConfigureDefault2)
	run(t, "configure", "configuration/custom", "testcustomconfigure",  testConfigureCustom)

	// value_test.go
	run(t, "value", "value/bug_01", "testvalue", testValues_bug_01)

	// modules_test.go
	run(t, "modules", "modules/target/arm64-darwin", "", testVariantTarget_arm64_darwin) //testvarianttarget
	run(t, "modules", "modules/app/arm64-darwin", "", testApp_arm64_darwin) //testapp
	run(t, "modules", "modules/llvm/config/arm64-darwin", "", testLLVMConfig1_arm64_darwin, test_configure{true}) //testllvmconfig
	run(t, "modules", "modules/llvm/config/arm64-darwin", "", testLLVMConfig2_arm64_darwin) //testllvmconfig
	run(t, "modules", "modules/toolchain/booting/arm64-darwin", "", testToolchainBooting_arm64_darwin) //testtoolchainbooting
}
