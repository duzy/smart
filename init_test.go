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
	pkg_flag "flag"
	pkg_time "time"
)

type testcase_f1 func (*testcase)
type testcase_f2 func (*testcase, string, string)
type testcase struct{
	Context
	*testing.T
	spec string
	// run func(testcase_f1)
	srcs map[string]struct{}
	chks map[string]struct{}
}
type testcase1  struct{ *testcase ; i any }
type test_arg   struct{ name string; val any }
type test_final struct{}

const testModulesPath = "/Volumes/workspace/.smart/modules"

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
	if _, e := os.Stat(dir); e != nil { panic(e) }

	ctx := new_universe(ii...)
	ctx.statcache = make(map[string]*filebase) // must reset the statcache
	ctx.panicFailureOnFlushedErrors = false
	ctx.globe.main = nil
	ctx.workdir = dir

	defer func() {
		if ctx.flush(ctx); ctx.erros > 0 || count_diag(ctx, diagError) > 0 {
			var s = name
			if s == "" { s = spec }
			if s == "" { s = dir }
			panic(loaderros{s, ctx.erros + count_diag(ctx, diagError)})
		}
	} ()

	if !test_mode {
		erro(ctx, "not test mode").trace()
	}

	if testHasModule("configure") && !ctx.paths.has(testModulesPath) {
		ctx.paths = append(ctx.paths, testModulesPath)
	}

	res = &testcase{ctx, t, spec, nil, nil}
	res.srcs = make(map[string]struct{})
	res.chks = make(map[string]struct{})

	ctx.load(res)

	if m := ctx.globe.main; m == nil {
		erro(ctx, "%s", dir).trace()
	} else if name != "" && m.name != name {
		erro(ctx, "project %v != %v", m.name, name).trace()
	} else {
		res.Context = closure_with(ctx, m)
	}
	return
}

func (tc *testcase) inner() Context { return tc.Context }
func (tc *testcase) cast(t reflect.Type) Context { return icast(tc,t) }
func (tc *testcase) ts(string) string { return "{=test "+tc.spec+" "+ts(tc.Context)+"}" }
func (tc *testcase) String() string { return ts(tc.Context) }
func (tc *testcase) do(ctx Context, op any) (_ any) {
	switch t := op.(type) {
	case is_test_case: return true
	case is_test_mode: return test_mode
	case silent_configure: return true
	case loading_source:
		tc.srcs[string(t)] = struct{}{}
		return
	case checked_source:
		tc.chks[string(t)] = struct{}{}
		return
	case is_loading_source:
		_, y := tc.srcs[string(t)]
		return y
	case is_checked_source:
		_, y := tc.chks[string(t)]
		return y
	case get_position:
		if p := _project(ctx); p != nil { return p.position }
		var p = _position(tc.Context)
		if !p.valid() { p.Filename = _workdir(tc.Context) }
		return p
	}
	return tc.Context.do(ctx, op)
}

func (tc *testcase) err(f string, i ...any) {
	var ctx Context = tc
argsloop:
	for _, a := range i {
		if x, y := a.(tst); y {
			a = x.i
		}
		switch t := a.(type) {
		case positioner:
			ctx = pc(ctx, t.Position())
			break argsloop
		}
	}
	erro(ctx, f, i...).debug(1, skipint{2})
	if false { flush(ctx) }
}

func (tc *testcase) flush() {
	if n := count_diag(tc.Context, diagError); n > 0 {
		var pos Position
		if p := _project(tc); p != nil { pos = p.position } else { pos = _position(tc) }
		note(tc.Context, "%v: %v errors", _project(tc), n).debug(1, skipint{2})
		tc.Errorf("%d errors in %s", flush(tc.Context), pos.Filename)
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
	var ori origin
	var s = skipint{2}
	var j = _project(tc)
	var ctx Context = tc

	for _, i := range ii {
		var vb = valbase{_position(tc)}
		switch t := i.(type) {
		case test_final: ctx = _final(ctx)
		case     origin: ori = t
		case   *project: j, ctx = t, closure_with(ctx, t.scope)
		case    skipint: s.int = t.int+1
		case       opt : o = append(o, t.Value)
		case       opts: o = append(o, t.vals...)
		case   test_arg: a = append(a, &pair{&word{vb,t.name},va(tc,t.val)})
		default:         a = append(a, va(tc, i))
		}
	}

	switch t := i0.(type) {
	case string:
		if x = j.resolve(ctx, t) ; x == nil {
			erro(ctx, "%v '%s' is nil", j, t)
			note(ctx, "%v", reflect.ValueOf(j.scope.elems).MapKeys()).trace()
		}
	case Value:
		if x = t ; t == nil {
			erro(ctx, "%v %s is nil", j, ts(t)).trace()
		}
	default:
		erro(ctx, "%v %v", j, ts(i0)).trace()
	}

	if ori != 0 { ctx = original{ctx,nil,ori} }

	if d, y := x.(*def); y {
		if 0 < len(a) {
			res, _, _ = evoke(ctx, x, o, a)
			return
		} else if ori == 0 || ori == defExpand0 {
			return d.value
		} else  if d.value != nil && defExpand0 < ori && ori < defExecute {
			return d.value.expand(ctx)
		} else {
			return
		}
	} else if true {
		res, _, _ = evoke(ctx, x, o, a)
		return
	} else if 0 < len(a) {
		ac := automatic{Context:ctx, defs:make(defmap)}
		ac.args(ctx, a)
		return x.expand(&ac)
	} else {
		return x.expand(ctx)
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
	ctx := loadcase(t, "testdata/"+spec, spec, name, ii...)
	// ctx.run = func(f2 testcase_f1) { runcase(t, name, spec, f2) }

	defer func() {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case prerequisite_evoke_loop:
				errostack(pc(ctx,e.Value), 16, "%v", e.Value).trace()
			case trace_evoke_loop_err:
				errostack(pc(ctx,e.Value), 16, "evoke loop: %v", e.Value).trace()
			case traverse_state:
				switch e.uint {
				case traverse_done:
				default:
					errostack(pc(ctx,e.p), 16, "%v", tv(e)).trace()
				}
			default:
				flush(ctx)
				panic(e) // continues unwind
			}
		}

		d := _diagnostic(ctx)
		d.flush(ctx)

		if d.erros > 0 || count_diag(d, diagError) > 0 {
			var s = name
			if s == "" { s = spec }
			panic(loaderros{s, d.erros + count_diag(d, diagError)})
		}
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
			case test_silentOptionalArrow:
				c.silentOptionalArrow = v.bool
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
    case []Value: v = _list(t...)
    case  int:    v = _decimal(_position(ctx), int64(t))
    case  int16:  v = _decimal(_position(ctx), int64(t))
    case  int32:  v = _decimal(_position(ctx), int64(t))
    case  int64:  v = _decimal(_position(ctx), int64(t))
    case uint:    v = _decimal(_position(ctx), int64(t))
    case uint16:  v = _decimal(_position(ctx), int64(t))
    case uint32:  v = _decimal(_position(ctx), int64(t))
    case uint64:  v = _decimal(_position(ctx), int64(t))
	case   bare:  v =    _word(_position(ctx), string(t))
    case string:
        if t == "" {
            v = _none(_position(ctx))
        } else {
            v = _word(_position(ctx), t)
        }
    case []string:
        var elems []Value
        for _, s := range t {
            if s == "" {
                v = _none(_position(ctx))
            } else {
                v = _word(_position(ctx), s)
            }
            elems = append(elems, v)
        }
        v = _list(elems...)
    case []any:
        var elems []Value
        for _, i := range t { elems = append(elems, va(ctx, i)) }
        v = _list(elems...)
    case nil:
        v = _null(_position(ctx))
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
	res, _, _ = evoke(ctx, v, o, a)
    return
}

type (
	test_caseinit    struct { f func() }
	test_hook_assert struct { f func(Context, Value, bool, any); i any }
	test_hook_debug  struct { f func(Context, string, []Value, any); i any }
	test_variant     struct { string }
	test_silentOptionalArrow struct { bool }
)

func Test(t *testing.T) {
	if false {
		pkg_flag.Set("test.timeout", fmt.Sprintf("%v", 3600*pkg_time.Second))
	}

	// context_test.go
	t.Run("context", testInner)

	// position_test.go
	t.Run("position", testPositionExample)

	// scanner_test.go
	// t.Run("scanner", testInit)
	// t.Run("scanner", testStrings)
	// t.Run("scanner", testIntegers)
	// t.Run("scanner", testFloats)
	// t.Run("scanner", testDatetime)
	// t.Run("scanner", testArrays)
	// t.Run("scanner", testMaps)
	// t.Run("scanner", testCalls)
	// t.Run("scanner", testRules)
	// t.Run("scanner", testProgConstructs)

	// parser_test.go
	t.Run("parser", testParseFile)
	t.Run("parser", testParseDir)

	// loader_test.go
	run(t, "loader", "empty", "testloader", testLoader)

	// value_test.go
	run(t, "value", "value", "testvalue", testValue, test_hook_assert{testValueAssertHook, &testValueStruct{}})

	// builtins_test.go
	run(t, "builtins", "assert",         "testassert", testAssert, test_hook_assert{testAssertHook, &testAssertStruct{}})
	run(t, "builtins", "locals",         "testlocals", testLocals)

	// value_test.go
	run(t, "value", "value/auto",        "testvalue", testAuto)
	run(t, "value", "value/closure",     "testvalue", testClosure)
	run(t, "value", "value/disjunction", "testvalue", testDisjunction)
	run(t, "value", "value/placeholder", "testvalue", testPlaceholders)
	run(t, "value", "value/optional",    "testvalue", testOptional, test_silentOptionalArrow{true})
	run(t, "value", "value/glob",        "testvalue", testGlob)
	run(t, "value", "value/1",           "testvalue", testValues1)
	run(t, "value", "value/2",           "testvalue", testValues2)
	run(t, "value", "value/3",           "testvalue", testValues3)
	run(t, "value", "value/4",           "testvalue", testValues4)
	run(t, "value", "value/5",           "testvalue", testValues5)
	run(t, "value", "value/6",           "testvalue", testValues6)
	run(t, "value", "value/7",           "testvalue", testValues7)
	run(t, "value", "value/8",           "testvalue", testValues8)
	run(t, "value", "value/9",           "testvalue", testValues9)
	run(t, "value", "value/10",          "testvalue", testValues10) // NOOP
	run(t, "value", "value/11",          "testvalue", testValues11)
	run(t, "value", "value/12",          "testvalue", testValues12)
	run(t, "value", "value/13",          "testvalue", testValues13)

	// builtins_test.go
	run(t, "builtins", "builtins/addprefix",  "testbuiltins", test__addprefix)
	run(t, "builtins", "builtins/addsuffix",  "testbuiltins", test__addsuffix)
	run(t, "builtins", "builtins/wildcard",   "testbuiltins", test__wildcard)
	run(t, "builtins", "builtins/if",         "testbuiltins", test__if)
	run(t, "builtins", "builtins/foreach",    "testbuiltins", test__foreach)
	run(t, "builtins", "builtins/foreach/1",  "testbuiltins", test__foreach1)
	run(t, "builtins", "builtins/foreach/2",  "testbuiltins", test__foreach2)
	run(t, "builtins", "builtins/foreach/3",  "testbuiltins", test__foreach3)
	run(t, "builtins", "builtins/foreach/4",  "testbuiltins", test__foreach4)
	run(t, "builtins", "builtins/foreach/5",  "testbuiltins", test__foreach5)
	run(t, "builtins", "builtins/logic",      "testbuiltins", test__logic)
	run(t, "builtins", "builtins/contains",   "testbuiltins", test__contains)
	run(t, "builtins", "builtins/join",       "testbuiltins", test__join)
	run(t, "builtins", "builtins/or",         "testbuiltins", test__or)
	run(t, "builtins", "builtins/xor",        "testbuiltins", test__xor)
	run(t, "builtins", "builtins/trimprefix", "testbuiltins", test__trimprefix)
	run(t, "builtins", "builtins/trimsuffix", "testbuiltins", test__trimsuffix)

	// template_test.go
	run(t, "template", "template", "testtemplate", testTemplate)

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
	run(t, "builtins", "builtins/file/0",     "testbuiltins", test__file0)
	run(t, "builtins", "builtins/file",       "testbuiltins", test__file)

	// template_test.go
	run(t, "template", "template/foreach", "testtemplate", testTemplateForeach)

	// rules_test.go
	run(t, "rules", "rule/0",                "testrules", testRules0)
	run(t, "rules", "rule/1",                "testrules", testRules1)
	run(t, "rules", "rule/contains",         "testrules", test__contains2)
	run(t, "rules", "rule/shell/for-stdout", "testrules", testShellForStdout, test_hook_debug{testShellForStdoutDebugHook, &testShellForStdoutDebugStruct{}})

	// configure_test.go
	run(t, "configure", "configuration",        "testdefaultconfigure", testConfigureDefault)
	run(t, "configure", "configuration/two",    "testdeftwoconfigure",  testConfigureDefault2)
	run(t, "configure", "configuration/custom", "testcustomconfigure",  testConfigureCustom)

	// modules_test.go
	run(t, "modules", "modules/target/arm64-darwin",                "", testVariantTarget)

	run(t, "bug", "bug/01", "testbug", testBug_01)

	if true {
		run(t, "modules", "modules/app/arm64-darwin",               "", testApp)
	} else if false {
		run(t, "modules", "modules/app/simple/arm64-darwin",        "", testApp)
	} else if false {
		run(t, "modules", "modules/app/complex/arm64-darwin",       "", testApp)
	} else if false {
		run(t, "modules", "modules/app/arm64-darwin",               "", testApp)
		run(t, "modules", "modules/app/simple/arm64-darwin",        "", testApp)
		run(t, "modules", "modules/app/complex/arm64-darwin",       "", testApp)
	} else if false {
		run(t, "modules", "modules/app/arm64-darwin",               "", testApp)
		run(t, "modules", "modules/llvm/config/arm64-darwin",       "", testLLVMConfig1)
		run(t, "modules", "modules/llvm/config/arm64-darwin",       "", testLLVMConfig2)
		run(t, "modules", "modules/toolchain/booting/arm64-darwin", "", testToolchainBooting)
	} else if false {
		run(t, "modules", "modules/app/complex/arm64-darwin",       "", testApp)
		run(t, "modules", "modules/llvm/config/arm64-darwin",       "", testLLVMConfig1)
		run(t, "modules", "modules/llvm/config/arm64-darwin",       "", testLLVMConfig2)
		run(t, "modules", "modules/toolchain/booting/arm64-darwin", "", testToolchainBooting)
	}
}
