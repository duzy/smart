//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "path/filepath"
	"runtime"
	"reflect"
	"strings"
	"strconv"
	"sync"
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
	srcs map[string]struct{}
	chks map[string]struct{}
}
type testcase1  struct{ *testcase ; i any }
type test_arg   struct{ name string; val any }
type test_final struct{}

const modules_dir = "/Volumes/workspace/.smart/modules"

var test_mode bool
var testdata_s = testdata_dir()
var testdata_a = strings.Join(strings.Split(testdata_s, pathSep), " ")
var testdata_t = testdata_dir_t("1:1")

var langs_map = map[string]string{
	"asm"   : "c",
	"c"     : "c",
	"s"     : "c",
	"S"     : "c",
	"cpp"   : "c++",
	"cxx"   : "c++",
	"c++"   : "c++",
	"cc"    : "c++",
	"cu"    : "cuda",
	"cu++"  : "cuda++",
	"cuda"  : "cuda",
	"cuh"   : "cuda",
	"cuh++" : "cuda++",
	"m"     : "objc",
	"mm"    : "objc++",
	"swift" : "swift",
}

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
	if i, e := os.Stat(filepath.Join(modules_dir, name)); e == nil { res = i.IsDir() }
	return
}

type (
	trim_fmt struct{ int ; string }
	trim_prefix trim_fmt
	trim_suffix trim_fmt
)
func testdata_f(f string, _a ...any) string {
	var a []any
	var tr_pre, tr_suf []trim_fmt
	var lc = "1:1"
	for _, _t := range _a {
		switch t := _t.(type) {
		case line_column_s: lc = string(t)
		case trim_prefix: tr_pre = append(tr_pre, trim_fmt(t))
		case trim_suffix: tr_suf = append(tr_suf, trim_fmt(t))
		default: a = append(a, t)
		}
	}
	// var testdata_s = testdata_dir()
	// var testdata_a = strings.Join(strings.Split(testdata_s, pathSep), " ")
	var ss = []string{testdata_s, testdata_a, testdata_dir_t(lc)}
	for _, tr := range tr_pre {
		if i := tr.int-1; -1 < i && tr.string != "" {
			ss[i] = strings.TrimPrefix(ss[i], tr.string)
		}
	}
	for _, tr := range tr_suf {
		if i := tr.int-1; -1 < i && tr.string != "" {
			ss[i] = strings.TrimSuffix(ss[i], tr.string)
		}
	}
	return sf(f, append([]any{ss[0], ss[1], ss[2]}, a...)...)
}

func testdata_fs(_a ...any) []string {
	var f []string
	var lc = "1:1"
	for _, _t := range _a {
		switch t := _t.(type) {
		case line_column_s: lc = string(t)
		case string: f = append(f, t)
		default: panic(t)
		}
	}
	return ssf(f, testdata_s, testdata_a, testdata_dir_t(lc))
}

func testdata_dir() string {
    return filepath.Join(filepath.Dir(get_filename(1)), "testdata")
}

func testdata_dir_t(lc string) (s string) {
	var ss = strings.Split(testdata_s, pathSep)
	for i, t := range ss {
		if t == "" {
			switch i {
			case         0: s += "{"+lc+":punct ROOT}"
			case len(ss)-1: s += "{"+lc+":punct TAIL}"
			}
		} else {
			s += " {"+lc+":raw "+t+"}"
		}
	}
	return
}

type testcase_erros struct{ string; int }
func (t *testcase_erros) Error() string {
	return fmt.Sprintf("%d errors in test case `%s`", t.int, t.string)
}

func loadcase(t *testing.T, dir, spec, name string, ii ...any) (res *testcase) {
	if !filepath.IsAbs(dir) { dir = filepath.Join(workBaseDir, dir) }
	if _, e := os.Stat(dir); e != nil { panic(e) }

	ctx := new_universe(ii...)
	ctx.statcache = make(map[string]*filebase) // must reset the statcache
	ctx.panicFailureOnFlushedErrors = false
	ctx.globe.main = nil
	ctx.workdir = dir

	if false { defer func() {
		if ctx.flush(ctx); ctx.erros > 0 || diagCount(ctx, diagError) > 0 {
			var s = name
			if s == "" { s = spec }
			if s == "" { s = dir }
			panic(&testcase_erros{s, ctx.erros + diagCount(ctx, diagError)})
		}
	}()}

	if !test_mode {
		debug(ctx, "not test mode", trace{})
	}

	if testHasModule("configure") && !ctx.paths.has(modules_dir) {
		ctx.paths = append(ctx.paths, modules_dir)
	}

	res = &testcase{ctx, t, spec, nil, nil}
	res.srcs = make(map[string]struct{})
	res.chks = make(map[string]struct{})

	ctx.load(res)

	if m := ctx.globe.main; m == nil {
		debug(ctx, "%s", dir, trace{})
	} else if name != "" && m.name != name {
		debug(ctx, "project %v != %v", m.name, name, trace{})
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
	switch op.(type) {
	case is_test_case: return true
	case is_test_mode: return test_mode
	case silent_configure: return true
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
	debug(ctx, f, append(i, callstack{num:1, skip:1})...)
	if false { flush(ctx) }
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

func (tc *testcase) vs(a any, b ...any) (_ string) { return __string(tc, tc.val(a, b...)) }

func (tc *testcase) val(i0 any, ii ...any) (res Value) {
	var proj, skip = _project(tc), 2
	var ori origin
	var ctx Context = tc
	var pos Position
	var a, o []Value
	var x Value

	switch t := i0.(type) {
	case Position: pos = t
	case string:
		if x = proj.resolve(ctx, t) ; x == nil {
			debug(ctx, _f("%v '%s' is nil", proj, t),
				_f("%v", reflect.ValueOf(proj.scope.elems).MapKeys()),
				trace{})
		}
	case Value:
		if x = t ; t == nil {
			debug(ctx, "%v %s is nil", proj, ts(t), trace{})
		}
	default:
		debug(ctx, "%v %v", proj, ts(i0), trace{})
	}

	for _, i := range ii {
		var vb = valbase{_position(tc)}
		switch t := i.(type) {
		case test_final: ctx = _final(ctx)
		case   Position: pos = t
		case   *project: proj, ctx = t, closure_with(ctx, t.scope)
		case    skipint: skip = int(t)+1
		case     origin: ori |= t
		case       opt : o = append(o, t.Value)
		case       opts: o = append(o, t.vals...)
		case   test_arg: a = append(a, &pair{&word{vb,t.name},va(pc(tc,pos),t.val)})
		default:         a = append(a, va(pc(tc,pos), i))
		}
	}

	if false && 0 < skip { debug(ctx, "TODO: skip %d", skip) }
	if d, y := x.(*def); y {
		if pc, file, line, ok := runtime.Caller(1); ok {
			p := Position{}
			p.Filename, p.Line = file, line
			ctx = &srcctx{posctx{ctx,p}, runtime.FuncForPC(pc), d}
		}
		if 0 < len(a) {
			return evoke(original{ctx,ori}, x, o, a)
		} else if ori == 0 || ori&defExpand0 != 0 {
			return d.value
		} else if  d.value != nil && ori&(defExpand0|defExpand1|defExpand2|defExpand3|defExecute)!= 0 {
			return expand(original{ctx,ori},d.value)
		} else {
			return
		}
	} else if true {
		return evoke(ctx, x, o, a)
	} else if 0 < len(a) {
		ac := automatic{Context:ctx, defs:make(def_map)}
		ac.args(ctx, a)
		return expand(&ac,x)
	} else {
		return expand(ctx,x)
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

	if false { defer func() {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case prerequisite_evoke_loop:
				errostack(pc(ctx,e.Value), 16, "%v", e.Value, trace{})
			case trace_evoke_loop_err:
				errostack(pc(ctx,e.Value), 16, "evoke loop: %v", e.Value, trace{})
			case traverse_state:
				switch e.uint {
				case traverse_done:
				default:
					errostack(pc(ctx,e.p), 16, "%v", tv(e), trace{})
				}
			default:
				flush(ctx)
				panic(e) // continues unwind
			}
		}

		d := _diagnostic(ctx)
		d.flush(ctx)

		if d.erros > 0 || diagCount(d, diagError) > 0 {
			var s = name
			if s == "" { s = spec }
			panic(&testcase_erros{s, d.erros + diagCount(d, diagError)})
		}
	}()}

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
    case  int  :  v = _decimal(_position(ctx), int64(t))
    case  int16:  v = _decimal(_position(ctx), int64(t))
    case  int32:  v = _decimal(_position(ctx), int64(t))
    case  int64:  v = _decimal(_position(ctx), int64(t))
    case uint  :  v = _decimal(_position(ctx), int64(t))
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
        debug(ctx, "%v", ts(i), trace{})
    }
    return
}

func test_evoke(ctx Context, v Value, ii ...any) (res Value) {
	var a, o []Value
	for _, i := range ii {
		switch t := i.(type) {
		case    opt : o = append(o, t.Value)
		case    opts: o = append(o, t.vals...)
		case []Value: a = append(a, t...)
		case   Value: a = append(a, t)
		default:      a = append(a, va(ctx, i))
		}
	}
	return evoke(ctx, v, o, a)
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
	// t.Run("parser", testParseFile)
	// t.Run("parser", testParseDir)

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
	run(t, "value", "value/2/0",         "testvalue", testValues20)
	run(t, "value", "value/2/1",         "testvalue", testValues21)
	run(t, "value", "value/2/2",         "testvalue", testValues22)
	run(t, "value", "value/3",           "testvalue", testValues3)
	run(t, "value", "value/4",           "testvalue", testValues4)
	run(t, "value", "value/5",           "testvalue", testValues5)
	run(t, "value", "value/6",           "testvalue", testValues6)
	run(t, "value", "value/7",           "testvalue", testValues7)
	run(t, "value", "value/8",           "testvalue", testValues8)
	run(t, "value", "value/9",           "testvalue", testValues9)
	run(t, "value", "value/10",          "testvalue", testValues10) // empty
	run(t, "value", "value/11",          "testvalue", testValues11)
	run(t, "value", "value/12",          "testvalue", testValues12)
	run(t, "value", "value/13",          "testvalue", testValues13)

	// builtins_test.go
	run(t, "builtins", "builtins/addprefix",  "testbuiltins", test__addprefix)
	run(t, "builtins", "builtins/addsuffix",  "testbuiltins", test__addsuffix)
	run(t, "builtins", "builtins/wildcard",   "testbuiltins", test__wildcard)
	run(t, "builtins", "builtins/wildcard/1", "testbuiltins", test__wildcard1)
	run(t, "builtins", "builtins/wildcard/2", "testbuiltins", test__wildcard2)
	run(t, "builtins", "builtins/wildcard/3", "testbuiltins", test__wildcard3)
	run(t, "builtins", "builtins/if",         "testbuiltins", test__if)
	run(t, "builtins", "builtins/closure",    "testbuiltins", test__closure)
	run(t, "builtins", "builtins/delegate",   "testbuiltins", test__delegate)
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
	run(t, "valcache", "valcache",   "testvalcache", testValueCache)
	run(t, "valcache", "valcache/1", "testvalcache", testValueCache1)
	run(t, "valcache", "valcache/2", "testvalcache", testValueCache2)
	run(t, "valcache", "valcache/3", "testvalcache", testValueCache3)

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
	run(t, "modules", "modules/target/arm64-darwin", "", testVariantTarget)

	run(t, "bug", "bug/01", "testbug", testBug_01)

	if true {
		run(t, "modules", "modules/app/arm64-darwin", "", testApp)
	}
	if false {
		run(t, "modules", "modules/app/simple/arm64-darwin",  "", testApp)
	}
	if false {
		run(t, "modules", "modules/app/complex/arm64-darwin", "", testApp)
	}
	if false {
		run(t, "modules", "modules/llvm/config/arm64-darwin", "", testLLVMConfig1)
	}
	if false {
		run(t, "modules", "modules/llvm/config/arm64-darwin", "", testLLVMConfig2)
	}
	if false {
		run(t, "modules", "modules/toolchain/booting/arm64-darwin", "", testToolchainBooting)
	}
}



type testAssertStruct struct {
	bools []bool
	vals []Value
}
func testAssertHook(ctx Context, v Value, b bool, i any) {
	s := i.(*testAssertStruct)
	s.bools, s.vals = append(s.bools, b), append(s.vals, v)
}
func testAssert(ctx testcase1) {
	s := "foo"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if v != d.value {
		ctx.err("%v != %v", v, d.value)
	} else if v.String() != "foo" {
		ctx.err("%v", tst{v})
	} else if __string(ctx, v) != "foo" {
		ctx.err("%v", tst{v})
	}

	if s, y := ctx.i.(*testAssertStruct); !y {
		ctx.err("%T", ctx.i)
	} else if len(s.bools) != 12 {
		ctx.err("%v, %v, %v %v", s.vals, s.bools, len(s.vals), len(s.bools))
	} else if len(s.vals) != len(s.bools) {
		ctx.err("%v %v", s.vals, s.bools)
	} else if i := 0; s.vals[i].String() != "{=true}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 1; s.vals[i].String() != "{=false}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 2; s.vals[i].String() != "{=yes}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 3; s.vals[i].String() != "{=no}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 4; s.vals[i].String() != "" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 5; s.vals[i].String() != "{=undef x}" || s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 6; s.vals[i].String() != "{}" || s.bools[i] { // {=null}
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].String() != "x" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 7; s.vals[i].String() != "x" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else if i = 8; s.vals[i].String() != "foobar{}" || !s.bools[i] {
		ctx.err("%v %v %v", tst{s.vals[i]}, s.vals[i], s.bools[i])
	} else {
		type rec struct{ string; bool }
		for i, r := range []rec{
			rec{"{=true}", true},
			rec{"{=false}", false},
			rec{"{=yes}", true},
			rec{"{=no}", false},
			rec{"", false},
			rec{"{=undef x}", false},
			rec{"{}", false},
			rec{"x", true},
			rec{"foobar{}", true},
			rec{"1", true},
			rec{"0", false},
			rec{"{=true}", true}, // $(equal $(foo),foo)
		}{
			if t := s.vals[i].String(); t != r.string {
				ctx.err("%s != %s : %s", t, r.string, ts(s.vals[i]))
			} else if s.bools[i] != r.bool {
				ctx.err("%v != %v : %v , %v , %v", s.bools[i], r.bool, s.vals[i], ts(s.vals[i]), __true(ctx, s.vals[i]))
			}
		}
	}
}



type testValueStruct struct {
	assert_bool bool
	assert_value Value
}
func testValueAssertHook(ctx Context, v Value, b bool, i any) {
	st := i.(*testValueStruct)
	st.assert_bool = b
	st.assert_value = v
}
func testValue(ctx testcase1) {
	st := ctx.i.(*testValueStruct)
	if st.assert_value == nil {
		// ctx.err("assert: nil value")
	} else if st.assert_bool {
		// ctx.err("assert")
	}

	j := _project(ctx)

	if d := ctx.def("vals"); d == nil {
		ctx.err("vals")
	} else if d.scope != j.scope {
		ctx.err("%v", tst{d})
	} else if d.scope.project != j {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), `{word foo/bar 'strlit' "strcomp" 0 1} {=yes} {=false} {=true} {=path foo} foo/bar {=file foobar} {=glob **.c} {=regex xx} 1 0.1 {}`; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), `word foo/bar strlit strcomp 0 1 yes false true foo foo/bar foobar **.c xx 1 0.1`; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("cond0"); d == nil {
		ctx.err("cond0: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x?"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x?"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("cond01"); d == nil {
		ctx.err("cond01: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("cond02"); d == nil {
		ctx.err("cond02: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x???y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x???y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("cond03"); d == nil {
		ctx.err("cond03: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x&(something)?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("cond11"); d == nil {
		ctx.err("cond11: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("cond12"); d == nil {
		ctx.err("cond12: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x???y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x???y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("cond13"); d == nil {
		ctx.err("cond13: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x&(something)?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
		// hold line ...
	} else if s, t := __string(src(ctx,d),v), "x?y"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction0"); d == nil {
		ctx.err("disjunction0: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{a b c}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "a b c"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction00"); d == nil {
		ctx.err("disjunction00: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction01"); d == nil {
		ctx.err("disjunction01: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x{$1}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, v.Position(), []string{"a","b","c"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "xa xb xc"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction02"); d == nil {
		ctx.err("disjunction02: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x{&(something)}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction03"); d == nil {
		ctx.err("disjunction03: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x{a b c}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "xa xb xc"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("disjunction1"); d == nil {
		ctx.err("disjunction1: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := l.String(), `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s := __string(src(ctx,d), l); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	if d := ctx.def("disjunction2"); d == nil {
		ctx.err("disjunction2: %v", _project(ctx).elems)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v: %v", tst{l}, l.elems)
	} else if s, t := l.String(), `xay1z xay2z xay3z xby1z xby2z xby3z xcy1z xcy2z xcy3z`; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s := __string(src(ctx,d), l); s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	var (
		glob1 = ctx.val("glob1")
		glob2 = ctx.val("glob2")
		glob3 = ctx.val("glob3")
		regex1 = ctx.val("regex1")
		regex2 = ctx.val("regex2")
		regex3 = ctx.val("regex3")
		regex4 = ctx.val("regex4")
		regex5 = ctx.val("regex5")
		regex6 = ctx.val("regex6")
	)

	if glob1 == nil {
		ctx.err("glob1: %v", ctx.def("glob1"))
	} else if __string(ctx, glob1) != "*.c" {
		ctx.err("%v", tst{glob1})
	} else if g, y := glob1.(*globbrace); !y {
		ctx.err("%v", tst{glob1})
	} else if g.String() != "{=glob *.c}" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if __string(ctx, g) != "*.c" {
		ctx.err("%v", tst{g})
	}

	if glob2 == nil {
		ctx.err("glob2")
	} else if __string(ctx, glob2) != "**.c" {
		ctx.err("%v", tst{glob2})
	} else if g, y := glob2.(*globbrace); !y {
		ctx.err("%v", tst{glob2})
	} else if g.String() != "{=glob **.c}" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if __string(ctx, g) != "**.c" {
		ctx.err("%v", tst{g})
	}

	if glob3 == nil {
		ctx.err("glob3")
	} else if g, y := glob3.(*globbrace); !y {
		ctx.err("%v", tst{glob3})
	} else if g.String() != "{=glob x*z?.c}" {
		ctx.err("%v ; %v", tst{g}, g)
	} else if __string(ctx, g) != "x*z?.c" {
		ctx.err("%v", tst{g})
	}

	if __string(ctx,regex1) != `x{1}, x{1,}, x{1,2}, x{5}?, x{2,}?, x{2,8}? \p{Greek}, \P{Greek}` {
		ctx.err("regex1 is wrong: %v", regex1)
	}

	if __string(ctx,regex2) != `(re) (?P<name>re) (?:re) (?im) (?sU:re) \x{10ffff} \x1f \123 \* \. \? \$` {
		ctx.err("regex2 is wrong: %v", regex2)
	}

	if __string(ctx,regex3) != `[[:xdigit:]]*, [^[:alpha:]], [^xyz] [a-z] \A \B \b \Q**??^:[]{}\E \^ \z` {
		ctx.err("regex3 is wrong: %v", regex3)
	}

	if __string(ctx,regex4) != `fo{2}\.c` {
		ctx.err("regex4 is wrong: %v", regex4)
	}

	if __string(ctx,regex5) != `fo{2}/bar\.c` {
		ctx.err("regex5 is wrong: %v", regex5)
	}

	if __string(ctx,regex6) != `fo{2}(/o{2}){3}/bar\.c` {
		ctx.err("regex6 is wrong: %v", regex6)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if __string(src(ctx,d),v) != "foo.c" {
		ctx.err("%v", tst{v})
	} else if a, b, _, c := match(ctx, glob1, v); sf("%v %v → %v %v %v", glob1, v, a, b, c) != "{=glob *.c} foo.c → true foo.c [foo]" {
		ctx.err("%v %v → %v %v %v", glob1, v, a, b, c)
	} else if a, b, _, c := match(ctx, glob2, v); sf("%v %v → %v %v %v", glob2, v, a, b, c) != "{=glob **.c} foo.c → true foo.c [foo]" {
		ctx.err("%v %v → %v %v %v", glob2, v, a, b, c)
	} else if a, b, _, c := match(ctx, regex4, v); sf("%v %v → %v %v %v", regex4, v, a, b, c) != "{=regex fo{2}\\.c} foo.c → true foo.c []" {
		ctx.err("%v %v → %v %v %v", regex4, v, a, b, c)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, _, c := match(ctx, glob1, v); sf("%v %v → %v %v %v", glob1, v, a, b, c) != "{=glob *.c} foo/bar.c → false foo [foo]" {
		ctx.err("%v %v → %v %v %v", glob1, v, a, b, c)
	} else if a, b, _, c := match(ctx, glob2, v); sf("%v %v → %v %v %v", glob2, v, a, b, c) != "{=glob **.c} foo/bar.c → true foo/bar.c [foo/bar]" {
		ctx.err("%v %v → %v %v %v", glob2, v, a, b, c)
	} else if a, b, _, c := match(ctx, regex5, v); sf("%v %v → %v %v %v", regex5, v, a, b, c) != "{=regex fo{2}/bar\\.c} foo/bar.c → true foo/bar.c []" {
		ctx.err("%v %v → %v %v %v", regex5, v, a, b, c)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, _, c := match(ctx, glob1, v); sf("%v %v → %v %v %v", glob1, v, a, b, c) != "{=glob *.c} foo/oo/oo/oo/bar.c → false foo [foo]" {
		ctx.err("%v %v → %v %v %v", glob1, v, a, b, c)
	} else if a, b, _, c := match(ctx, glob2, v); sf("%v %v → %v %v %v", glob2, v, a, b, c) != "{=glob **.c} foo/oo/oo/oo/bar.c → true foo/oo/oo/oo/bar.c [foo/oo/oo/oo/bar]" {
		ctx.err("%v %v → %v %v %v", glob2, v, a, b, c)
	} else if a, b, _, c := match(ctx, regex6, v); sf("%v %v → %v %v %v", regex6, v, a, b, c) != "{=regex fo{2}(/o{2}){3}/bar\\.c} foo/oo/oo/oo/bar.c → true foo/oo/oo/oo/bar.c [/oo]" {
		ctx.err("%v %v → %v %v %v", regex6, v, a, b, c)
	}

	// TODO: test glob.stencil(...)

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "a\\,b\\,c,x\\,y\\,z"; s != t {
		ctx.err("%v : %v", tst{v}, v)
	} else if s, t := __string(src(ctx,d),v), "a,b,c,x,y,z"; s != t {
		ctx.err("%v : %v", tst{v}, v)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("v5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), `{=quote a\,b\,c x\,y\,z}`; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), `"a,b,c x,y,z"`; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	// TODO: test regexp.stencil(...)

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "http://extbit.io/help"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://extbit.com"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://extbit.com?foo=x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val9"); d == nil {
		ctx.err("val9")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://extbit.com?foo=x&bar=y#foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "https://ext.pub?foo=x+y+z&bar=x%20y%20z#foo+bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def(`configure.types."atomic.h"`); d == nil {
		ctx.err(`configure.types."atomic.h"`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := __string(src(ctx,d),v), "atomic_bool"; s != t {
		ctx.err("%s != %s; %v", s, t, v)
	}

	if d := ctx.def(`configure.types.<atomic.h>`); d == nil {
		ctx.err(`configure.types.<atomic.h>`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := __string(src(ctx,d),v), "atomic_bool"; s != t {
		ctx.err("%s != %s; %v", s, t, v)
	}

	if d := ctx.def(`configure.types`); d == nil {
		ctx.err(`configure.types`)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if t := __string(src(ctx,d),v); t == "" {
		ctx.err("%s; %v", t, v)
	} else if s := `- configure.types.<atomic.h> <atomic.h> atomic.h,`; !strings.Contains(t, s) {
		ctx.err("%s, %s; %v", s, t, v)
	} else if s := `- configure.types."atomic.h" "atomic.h" , atomic.h`; !strings.Contains(t, s) {
		ctx.err("%s, %s; %v", s, t, v)
	}

	if d := ctx.def("conf0"); d == nil {
		ctx.err("conf0")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := __string(src(ctx,d),v), "conf0"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	if d := ctx.def("conf1"); d == nil {
		ctx.err("conf1")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := __string(src(ctx,d),v), "foo.o foo-x.o foo-x-y.o foo-x-y-z.o foobar.o"; s != t {
		ctx.err("%s != %s : %s : %v", s, t, v, tst{v})
	}

	s := "foo.o foo . o "
	s += "foo-x.o foo -x -x x . o "
	s += "foo-x-y.o foo -x-y -y y . o "
	s += "foo-x-y-z.o foo -x-y-z -z z . o "
	s += "foobar.o foobar . o"
	if d := ctx.def("conf2"); d == nil {
		ctx.err("conf2")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if x, y := v.(*list); !y {
		ctx.err("%v %v", typeof(v), v)
	} else if x.len() != 5*7 {
		ctx.err("%d, %v", x.len(), x)
	} else if t := __string(src(ctx,d), x); s != t {
		ctx.err("%s != %s; %v", t, s, ts(x))
	}

	if d := ctx.def("conf3"); d == nil {
		ctx.err("conf3")
	} else if d.o != defConfig {
		ctx.err("%v %v", d.origin, d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "conf3, foo bar, foo bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "conf3, foo bar, foo bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}



func testValues1(ctx *testcase) {
	if d := ctx.def(".test.foo"); d == nil {
		ctx.err(".test.foo")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s := __string(src(ctx,d),v); s != "-foo" {
		ctx.err("%v → %s", v, s)
	} else if s = v.String(); s != "-foo" {
		ctx.err("%v → %s", v, s)
	}
}
func testValues10(ctx *testcase) {
}
func testValues11(ctx *testcase) {
	if d := ctx.def(".test.0"); d == nil {
		ctx.err(".test.0")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test$1)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1, ".s1"); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "&(.test.s1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, ".s2"); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "&(.test.s2)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{".s1",".s2"}); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "&(.test⌜.s1 .s2⌟)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	if d := ctx.def(".test.1"); d == nil {
		ctx.err(".test.1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test{$1})"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1, []string{".s1",".s2"}); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "&(.test.s1) &(.test.s2)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo bar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}
}
func testValues12(ctx *testcase) {
	if d := ctx.def(".test.0"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.w)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foobaz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	if d := ctx.def(".test.1"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.{})"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "www"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}
func testValues13(ctx *testcase) {
	if d := ctx.def("foo"); d == nil {
		ctx.err("foo")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(-g!foobar)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := "&(-g!foobar)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if s, t := "not-foobar", __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues20(ctx *testcase) {
	s := ".test.ab"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-$1-$2"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab--"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.ba"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-$2-$1"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba--"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 $1$2$3 10 2 $(&(.test.x) $1$1,$2$2) 20 3 &(&(.test.x) $1$2,$2$1) 30 4 $3 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 abc 10 2 $(&(.test.x) aa,bb) 20 3 &(&(.test.x) ab,ba) 30 4 c 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 abc 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba 30 4 c 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 $(&(.test.x) aa,bb) 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 abc 10 2 foo_ab-aa-bb 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 {} 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 abc 10 2 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 $(&(.test.x) aa,bb) 20 3 &(&(.test.x) ab,ba) 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 abc 10 2 foo_ab-aa-bb 20 3 foo_ab-ab-ba 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 abc 10 2 foo_ba-bb-aa 20 3 foo_ba-ba-ab 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	// s = ".test.s0"
	// d = ctx.def(s)
	// if d == nil {
	// 	ctx.err(s)
	// } else if v := d.value; v == nil {
	// 	ctx.err("%v", tst{d})
	// } else if s, t := v.String(), "1 xy{} 10 2 {} 20 {} s0"; s != t {
	// 	ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	// } else if s, t := __string(src(ctx,d),v), "1 xy 10 2 20 s0"; s != t {
	// 	ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	// }
	//
	// s = ".test.s1"
	// d = ctx.def(s)
	// if d == nil {
	// 	ctx.err(s)
	// } else if v := d.value; v == nil {
	// 	ctx.err("%v", tst{d})
	// } else if s, t := v.String(), "1 xy{} 10 2 {} 20 {} s0 s1"; s != t {
	// 	ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	// } else if s, t := __string(src(ctx,d),v), "1 xy 10 2 20 s0 s1"; s != t {
	// 	ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	// }

	s = ".test.t1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t3"
	d = ctx.def(s)
	if d == nil || d.value == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ba-yy-xx 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ba-yy-xx 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ba-yy-xx 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ba-yy-xx 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ba-yy-xx 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ba-yy-xx 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test x,y) . $3"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 3 &(&(.test.x) xy,yx) 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 3 &(&(.test.x) xy,yx) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 $(&(.test.x) xx,yy) 20 3 &(&(.test.x) xy,yx) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 xy{} 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 xy 10 2 foo_ab-xx-yy 20 3 foo_ab-xy-yx 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}
}
func testValues21(ctx *testcase) {
	s := ".test.ab"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-$1-$2"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab--"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.ba"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-$2-$1"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba--"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 3 foo_ba-{}{}-{}{} 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20 3 foo_ba-- 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.t1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t3"
	d = ctx.def(s)
	if d == nil || d.value == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ba-{}{}-{}{} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ba-- 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test x,y) . $3"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 $(&(.test.x) {}{},{}{}) 20 3 &(&(.test.x) {}{},{}{}) 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 foo_ab-{}{}-{}{} 20 3 foo_ab-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 foo_ab-- 20 3 foo_ab-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}
}
func testValues22(ctx *testcase) {
	s := ".test.ab"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-$1-$2"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab--"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ab-a-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.ba"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-$2-$1"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba--"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo_ba-b-a"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if v := ctx.val(d.name, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30"; s != t {
		ctx.err("%s: %s != %s %s", d.name, s, t, tst{v})
	}

	s = ".test.t1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v : %v", tst{d}, d.value)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t3"
	d = ctx.def(s)
	if d == nil || d.value == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "cc"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test x,y) . $3"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 . x"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}

	s = ".test.t6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b", "x"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 {}{}{} 10 2 {} 20 3 foo_ba-{}{}-{}{} 30 4 {} 40 . {}"; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	} else if s, t := __string(src(ctx,d),v), "1 10 2 20 3 foo_ba-- 30 4 40 ."; s != t {
		ctx.err("%s != %s | %v | %s", s, t, v, tst{v})
	}
}

func testValues3(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$1 &(.test) &(.test.5) &(.test.5 x) &(.test.5 x,y) &(.test.5 x,y,z) &(.test aa)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "v----- v-x---- v-x-y--- v-x-y-z-- v----- a v-x---- a v-x-y--- a v-x-y-z-- a aa v----- v-x---- v-x-y--- v-x-y-z--"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "a"); v == nil {
		ctx.err(".test")
	} else if s, t := v.String(), "a &(.test) &(.test.5) &(.test.5 x) &(.test.5 x,y) &(.test.5 x,y,z) &(.test aa)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "a v----- v-x---- v-x-y--- v-x-y-z-- v----- a v-x---- a v-x-y--- a v-x-y-z-- a aa v----- v-x---- v-x-y--- v-x-y-z--"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues4(ctx *testcase) {
	s := ".test.*"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "D.c(-unique) D.c++(-unique) I.c(-unique) I.c++(-unique)", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 4 {
		ctx.err("%d, %v", l.len(), tst{l})
	} else if _, y := l.elems[0].(*argumented); !y || l.elems[0].String() != "D.c(-unique)" {
		ctx.err("%v", tst{l.elems[0]})
	} else if _, y := l.elems[1].(*argumented); !y || l.elems[1].String() != "D.c++(-unique)" {
		ctx.err("%v", tst{l.elems[1]})
	} else if _, y := l.elems[2].(*argumented); !y || l.elems[2].String() != "I.c(-unique)" {
		ctx.err("%v", tst{l.elems[2]})
	} else if _, y := l.elems[3].(*argumented); !y || l.elems[3].String() != "I.c++(-unique)" {
		ctx.err("%v", tst{l.elems[3]})
	}

	s = ".test.D.c.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{=list {8:31 {=compound {6:12:word c} {8:36:punct .} {5:11:word D}}} {17:16:delegate {17:18:builtin value} {=list {17:24:closure {=compound {17:26:punct .} {17:27:word test} {17:31:punct .} {17:32:word x}}}}} {19:16:delegate {19:18:builtin value} {=list {=compound {19:26:punct .} {19:27:word test} {19:31:punct .} {19:32:word v}}}} {25:16:closure {25:18:builtin value} {=list {25:24:delegate {23:9:def .test.x}}}} {39:16:delegate {37:15:def .test.foreach} {=list {39:32:delegate {37:19:auto 1}}} {=list {39:35:closure {=compound {39:37:punct .} {39:38:word test} {39:42:punct .} {39:43:word none}}}}} {=group {39:51:delegate {37:19:auto 1}}} {41:16:delegate {41:18:builtin foreach} {=list {41:26:delegate {37:19:auto 1}}} {=list {41:29:closure {=compound {41:31:punct .} {41:32:word test} {41:36:punct .} {41:37:word x} {41:38:punct .} {41:39:delegate {41:40:auto _}}}}}} {=group {41:45:delegate {37:19:auto 1}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.D $(value &(.test.x)) $(value .test.v) &(value $(.test.x)) $(.test.foreach $1,&(.test.none)) ($1) $(foreach $1,&(.test.x.$_)) ($1)"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := __string(src(ctx,d),v), "c.D xx xx xx () () ()"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 8 {
		ctx.err("%d, %v", l.len(), tst{l})
	}

	s = ".test.D.c.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), `{=list {9:31 {=compound {6:12:word c} {9:36:punct .} {5:11:word D}}} {18:16 {18:18:null}} {20:16 {20:18:null}} {26:16:closure {26:18:builtin value} {=list {26:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}} {40:16 {=group {37:18 {40:32 {37:19:null}}}}} {=group {40:51 {37:19:null}}} {42:16 {42:18:null}} {=group {42:45 {37:19:null}}}}`; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.D {} {} &(value .test.v) ({}) ({}) {} ({})"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := __string(src(ctx,d),v), "c.D xx () () ()"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 8 {
		ctx.err("%d, %v: %v", l.len(), l, tst{l})
	}

	s = ".test.D.c++.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "c++.D"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.D.c++.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "c++.D"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.I.c.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), `{=list {8:31 {=compound {6:12:word c} {8:36:punct .} {5:13:word I}}} {28:16:closure {28:18:builtin value} {=list {28:24:closure {23:9:def .test.x}}}} {30:16:closure {30:18:builtin value} {=list {30:24:delegate {23:9:def .test.x}}}} {32:16:delegate {32:18:builtin value} {=list {32:24:closure {23:9:def .test.x}}}} {34:16:delegate {34:18:builtin value} {=list {34:24:delegate {23:9:def .test.x}}}}}`; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.I &(value &(.test.x)) &(value $(.test.x)) $(value &(.test.x)) $(value $(.test.x))"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := __string(src(ctx,d),v), "c.I xx xx xx xx"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 5 {
		ctx.err("%d, %v", l.len(), tst{l})
	}

	s = ".test.I.c.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), `{=list {9:31 {=compound {6:12:word c} {9:36:punct .} {5:13:word I}}} {29:16:closure {29:18:builtin value} {=list {29:24:closure {23:9:def .test.x}}}} {31:16:closure {31:18:builtin value} {=list {31:24 {=compound {23:12:punct .} {23:13:word test} {23:17:punct .} {23:18:word v}}}}} {33:16 {22:12:word xx}} {35:16 {22:12:word xx}}}`; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c.I &(value &(.test.x)) &(value .test.v) xx xx"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := __string(src(ctx,d),v), "c.I xx xx xx xx"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 5 {
		ctx.err("%d, %v", l.len(), tst{l})
	}


	s = ".test.I.c++.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), `{8:31 {=compound {6:14:word c++} {8:36:punct .} {5:13:word I}}}`; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c++.I"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := __string(src(ctx,d),v), "c++.I"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.I.c++.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v", tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), `{9:31 {=compound {6:14:word c++} {9:36:punct .} {5:13:word I}}}`; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := v.String(), "c++.I"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("%s", d)
	} else if s, t := __string(src(ctx,d),v), "c++.I"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.and.x.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "x1", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.x.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "x2", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.y.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "y1", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.and.y.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v", d)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
		//
		//
	} else if s, t := "y2", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues5(ctx *testcase) {
	s := ".test.0"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x0 $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "z-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-a"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-{}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "z-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "z-{}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "z-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testValues6(ctx *testcase) {
	s := ".test"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.value.String() != "z-y-x-a" {
		ctx.err("%v", d)
	} else if v := ctx.val(d.name, defExpand1, "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "z-y-x-a", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues7(ctx *testcase) {
	if d := ctx.def(".test.z"); d == nil {
		ctx.err(".test.z")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "z-$1-$2"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test.y"); d == nil {
		ctx.err(".test.y")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.z y$1,y$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test.x"); d == nil {
		ctx.err(".test.x")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.y x$1,x$2)"; s != t {
		ctx.err("%v → %s != %s", tst{v}, s, t)
	}
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x $1,$2)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := "z-yxa-yxb", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}
}

func testValues8(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test$1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, ".u"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testValues9(ctx *testcase) {
	if d := ctx.def(".test"); d == nil {
		ctx.err(".test")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(.test.x .$1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "w"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func testAuto(ctx *testcase) {
	if c := _universe(ctx); c == nil {
		ctx.err("context.cast")
	}
	{
		ac := automatic{Context:ctx, defs:make(def_map)}
		ac.args(ctx, []Value{ease(ctx, []string{"a", "b", "c"})})
		if _automatic(&ac) != &ac {
			ctx.err("%v", Context(&ac))
		} else if len(ac.defs) != 1 { // maxDigitAutoNum
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if s := d.value.String(); s != "'a' 'b' 'c'" {
			ctx.err("%v → %s", tst{d.value}, s)
		} else if s := __string(src(ctx,d),d.value); s != "a b c" {
			ctx.err("%v → %s", tst{d.value}, s)
		} else if d, _ := ac.do(ctx, find_auto{"1"}).(*def); d == nil {
			ctx.err("%v", ac.defs)
		} else if d := auto_find(&ac, "1"); d == nil {
			ctx.err("%v", ac.defs)
		} else if v := auto_get(&ac, "1"); v == nil {
			ctx.err("%v", ac.defs)
		} else if s := __string(src(ctx,d),v); s != "a b c" {
			ctx.err("%v → %s", tst{v}, s)
		} else if d, v := ac.amend(ctx, "1", ease(ctx, "a")); d == nil {
			ctx.err("%v", ac.defs)
		} else if v == nil {
			ctx.err("%v", ac.defs)
		} else if v.String() != "'a' 'b' 'c'" {
			ctx.err("%v %v", ac.defs, v)
		} else if d.value == nil {
			ctx.err("%v %v", ac.defs, d)
		} else if d.value.String() != "'a'" {
			ctx.err("%v %v", ac.defs, d)
		} else if false { for i := 2; i <= maxDigitAutoNum; i += 1 {
			if d, y := ac.defs[strconv.Itoa(i)]; !y {
				ctx.err("%d: %v", i, ac.defs)
			} else if d.value != nil {
				ctx.err("%v", d)
			}
		}}
	}
	{
		ac := automatic{Context:ctx, defs:make(def_map)}
		ac.args(ctx, ease(ctx, []string{"a", "b", "c"}).(*list).elems)
		if len(ac.defs) != 3 { // maxDigitAutoNum
			ctx.err("%v", ac.defs)
		} else if d, y := ac.defs["1"]; !y {
			ctx.err("1: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "a" {
			ctx.err("%v", tst{d.value})
		} else if d, y := ac.defs["2"]; !y {
			ctx.err("2: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "b" {
			ctx.err("%v", tst{d.value})
		} else if d, y := ac.defs["3"]; !y {
			ctx.err("3: %v", ac.defs)
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() == "c" {
			ctx.err("%v", tst{d.value})
		} else if false { for i := 4; i <= maxDigitAutoNum; i += 1 {
			if d, y := ac.defs[strconv.Itoa(i)]; !y {
				ctx.err("%d: %v", i, ac.defs)
			} else if d.value != nil {
				ctx.err("%v", d)
			}
		}}
	}

	if d := ctx.def("foo0"); d == nil {
		ctx.err("foo0")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if l, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if len(l.elems) != 9 {
		ctx.err("%v", l.elems)
	} else if s, t := d.value.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := __string(src(ctx,d),d.value), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else {
		if v := ctx.val(d.name); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,d),v), ""; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, "a"); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "a {} {} {} {} {} {} {} {}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,d),v), "a"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, 1); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "1 {} {} {} {} {} {} {} {}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,d),v), "1"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}

		if v := ctx.val(d.name, defExpand1, 1, 2, 3); v == nil {
			ctx.err("%v", d)
		} else if s, t := v.String(), "1 2 3 {} {} {} {} {} {}"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,d),v), "1 2 3"; s != t {
			ctx.err("%s != %s : %s", s, t, tst{v})
		}
	}

	if d := ctx.def("foo1"); d == nil {
		ctx.err("foo1")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if s, t := d.value.String(), "{} {} {} {} {} {} {} {} {}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	} else if s, t := __string(src(ctx,d),d.value), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{d.value})
	}

	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(foobar) {}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d.name); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(foobar) {}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val01"); d == nil {
		ctx.err("val01")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(auto $(a))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if a := ctx.val(d, defExpand1, test_arg{"a", "x"}); a == nil {
		ctx.err("%v", tst{v})
	} else if s, t := a.String(), "x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),a), "x"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val02"); d == nil {
		ctx.err("val02")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(auto(a=2) $(val01),$(a))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val03"); d == nil {
		ctx.err("val03")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a=3) $(val02))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{} 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
	if d := ctx.def("val04"); d == nil {
		ctx.err("val04")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(val03)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "2 2"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	if d := ctx.def("bar1"); d == nil {
		ctx.err("bar1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val1" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar2"); d == nil {
		ctx.err("bar2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val2" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar3"); d == nil {
		ctx.err("bar3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val3" {
		ctx.err("%v", v)
	}

	if d := ctx.def("bar4"); d == nil {
		ctx.err("bar4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val4" {
		ctx.err("%v", v)
	}

	s := ".test.x0"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a3=3) $(a1)-$(a2)-$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "x-y-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}-{}-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "{}-{}-3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "--3"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.y0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto(a1=x a2=y a3=z) $(.test.x0)-$(a1)$(a2)$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x-y-3-xyz"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "X"}, test_arg{"a2", "Y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.y1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x-y-3-xyz"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x-y-3-xyz"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.z0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(auto $(.test.x0)-$(a1)$(a2)$(a3))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "x-y-3-xy{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "x-y-3-xy"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.z1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}-{}-3-{}{}{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, test_arg{"a1", "x"}, test_arg{"a2", "y"}); v == nil {
		ctx.err("%v", d.value)
	} else if s, t := v.String(), "{}-{}-3-{}{}{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "--3-"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = ".test.0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := "x-y-3-xyz", v.String(); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if t := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	s = ".test.1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v, t := d.value, "x-y-3-xyz"; v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	}

	s = ".test.2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v, t := d.value, "a-b-3-abc"; v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%v : %s != %s", v, s, t)
	}
}

func testClosure(ctx *testcase) {
	s := "foo_pre"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "&(foo.pre)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_pos"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "&(foo.pos)"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_z"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "$(&(foo.tail))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "$(&(foo.tail))"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "foo_nest_3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
}

func testDisjunction(ctx *testcase) {
	s := "val1"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo={&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo={&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=a foo=b foo=c"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "a=bar b=bar c=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=bar{&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=bar{&(.test)}"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo=bara foo=barb foo=barc"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}

	s = "val4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "foo{&(.test)}=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%s : %v", s, d)
	} else if s, t := v.String(), "fooa=bar foob=bar fooc=bar"; s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %s", s, t, tst{v})
	}
}

func testGlob(ctx *testcase) {
	if d := ctx.def("pat1.0"); d == nil {
		ctx.err("pat1.0: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("pat1.0: %v", _project(ctx))
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if p.len() != 2 {
		ctx.err("%v", p)
	} else if s, t := p.String(), ".test/x**y"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if d1 := ctx.def("pat1.1"); d1 == nil {
		ctx.err("pat1.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat1.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y .test/xxx-yyy → true .test/xxx-yyy [xx-yy]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d2 := ctx.def("pat1.2"); d2 == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y .test/xxx-yyx → false .test/xxx-yyx [xx-yyx]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d3 := ctx.def("pat1.3"); d3 == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y .test/xxx-yyx/y → true .test/xxx-yyx/y [xx-yyx/]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d4 := ctx.def("pat1.4"); d4 == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if v := d4.value; v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y .test/xxx-yyx/z → false .test/xxx-yyx/z [xx-yyx/z]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d5 := ctx.def("pat1.5"); d5 == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if v := d5.value; v == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y .test/xxx/a/b/c/yyy → true .test/xxx/a/b/c/yyy [xx/a/b/c/yy]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d6 := ctx.def("pat1.6"); d6 == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if v := d6.value; v == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y .test/x/xx-yy/y → true .test/x/xx-yy/y [/xx-yy/]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat2.0"); d == nil {
		ctx.err("pat2.0: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("pat2.0: %v", _project(ctx))
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if s, t := p.String(), ".test/x**y/z"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := ts(v), "{=path {=compound {12:12:punct .} {12:13:word test}} {=glob {12:18:word x} {12:19:meta **} {12:21:word y}} {12:23:word z}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if d1 := ctx.def("pat2.1"); d1 == nil {
		ctx.err("pat2.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat2.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y/z .test/xxx-yyy/z → true .test/xxx-yyy/z [xx-yy]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d2 := ctx.def("pat2.2"); d2 == nil {
		ctx.err("pat2.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat2.2: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y/z .test/xxx/a/b/c/yyy/z → true .test/xxx/a/b/c/yyy/z [xx/a/b/c/yy]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat3.0"); d == nil {
		ctx.err("pat3.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if p, y := val.(*path); !y {
		ctx.err("%v : %v", tst{val}, val)
	} else if val.String() != ".test/x**y/x**" {
		ctx.err("%v", tst{val})
	} else if __string(src(ctx,d),val) != ".test/x**y/x**" {
		ctx.err("%v: %s", tst{val}, __string(src(ctx,d),val))
	} else if d1 := ctx.def("pat3.1"); d1 == nil {
		ctx.err("pat3.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat3.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y/x** .test/xaaa/bbb/ccc/y/xxx/xx → true .test/xaaa/bbb/ccc/y/xxx/xx [aaa/bbb/ccc/ xx/xx]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d2 := ctx.def("pat3.2"); d2 == nil {
		ctx.err("pat3.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat3.2: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y/x** .test/xaabbccy/xabc → true .test/xaabbccy/xabc [aabbcc abc]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat4.0"); d == nil {
		ctx.err("pat4.0: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("pat4.0: %v", _project(ctx))
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if s, t := v.String(), ".test/x**y/x**y"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := ts(v), "{=path {=compound {18:12:punct .} {18:13:word test}} {=glob {18:18:word x} {18:19:meta **} {18:21:word y}} {=glob {18:23:word x} {18:24:meta **} {18:26:word y}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if d1 := ctx.def("pat4.1"); d1 == nil {
		ctx.err("pat4.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat4.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y/x**y .test/xaa/bb/ccy/xaa/bb/ccy → true .test/xaa/bb/ccy/xaa/bb/ccy [aa/bb/cc aa/bb/cc]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d2 := ctx.def("pat4.2"); d == nil {
		ctx.err("pat4.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat4.2: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/x**y/x**y .test/xaaay/x/aaa/y → true .test/xaaay/x/aaa/y [aaa /aaa/]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat5.0"); d == nil {
		ctx.err("pat5.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat5.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/x**/**y"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {21:12:punct .} {21:13:word test}} {=glob {21:18:word x} {21:19:meta **}} {=glob {21:22:meta **} {21:24:word y}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else {
		match(ctx, p, ctx.def("pat5.1").value)
		match(ctx, p, ctx.def("pat5.2").value)
	}

	if d := ctx.def("pat6.0"); d == nil {
		ctx.err("pat6.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat6.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/**y/**y"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {24:12:punct .} {24:13:word test}} {=glob {24:18:meta **} {24:20:word y}} {=glob {24:22:meta **} {24:24:word y}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else {
		match(ctx, p, ctx.def("pat6.1").value)
	}

	if d := ctx.def("pat7.0"); d == nil {
		ctx.err("pat7.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat7.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/**y/**y/z"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {26:12:punct .} {26:13:word test}} {=glob {26:18:meta **} {26:20:word y}} {=glob {26:22:meta **} {26:24:word y}} {26:26:word z}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if d1 := ctx.def("pat7.1"); d1 == nil {
		ctx.err("pat7.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat7.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/**y/**y/z .test/a/b/cy/a/b/c/y/z → true .test/a/b/cy/a/b/c/y/z [a/b/c a/b/c/]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat8.0"); d == nil {
		ctx.err("pat8.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat8.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/**/**z"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {28:12:punct .} {28:13:word test}} {=glob {28:18:meta **}} {=glob {28:21:meta **} {28:23:word z}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if d1 := ctx.def("pat8.1"); d1 == nil {
		ctx.err("pat8.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat8.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/**/**z .test/a/b/c/xyz → true .test/a/b/c/xyz [a/b/c xy]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat10.0"); d == nil {
		ctx.err("pat10.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat10.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if val.String() != ".test/*.h" {
		ctx.err("%v", tst{val})
	} else if d1 := ctx.def("pat10.1"); d1 == nil {
		ctx.err("pat10.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat10.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*.h .test/a.h → true .test/a.h [a]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d2 := ctx.def("pat10.2"); d2 == nil {
		ctx.err("pat10.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat10.2: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*.h .test/a/b.h → false .test/a [a]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d3 := ctx.def("pat10.3"); d3 == nil {
		ctx.err("pat10.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat10.3: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*.h .test/a/b/c.h → false .test/a [a]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat11.0"); d == nil {
		ctx.err("pat11.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat11.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if val.String() != ".test/*/*.h" {
		ctx.err("%v", tst{val})
	} else if d1 := ctx.def("pat11.1"); d1 == nil {
		ctx.err("pat11.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat11.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*.h .test/a.h → false .test/a.h [a.h]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d2 := ctx.def("pat11.2"); d2 == nil {
		ctx.err("pat11.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat11.2: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*.h .test/a/b.h → true .test/a/b.h [a b]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d3 := ctx.def("pat11.3"); d3 == nil {
		ctx.err("pat11.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat11.3: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*.h .test/a/b/c.h → false .test/a/b [a b]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat12.0"); d == nil {
		ctx.err("pat12.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("pat12.0: %v", _project(ctx))
	} else if p, y := val.(*path); !y {
		ctx.err("%v", p)
	} else if s, t := val.String(), ".test/*/*/*.h"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if s, t := ts(val), "{=path {=compound {38:12:punct .} {38:13:word test}} {=glob {38:18:meta *}} {=glob {38:20:meta *}} {=glob {38:22:meta *} {38:23:punct .} {38:24:word h}}}"; s != t {
		ctx.err("%v : %s != %s", val, s, t)
	} else if d1 := ctx.def("pat12.1"); d1 == nil {
		ctx.err("pat12.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat12.1: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*/*.h .test/a.h → false .test/a.h [a.h ]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d2 := ctx.def("pat12.2"); d2 == nil {
		ctx.err("pat12.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat12.2: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*/*.h .test/a/b.h → false .test/a/b.h [a b.h]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d3 := ctx.def("pat12.3"); d3 == nil {
		ctx.err("pat12.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat12.3: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*/*.h .test/a/b/c.h → true .test/a/b/c.h [a b c]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d4 := ctx.def("pat12.4"); d4 == nil {
		ctx.err("pat12.4: %v", _project(ctx))
	} else if v := d4.value; v == nil {
		ctx.err("pat12.4: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*/*.h .test/a/b/c/d.h → false .test/a/b/c [a b c]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	} else if d5 := ctx.def("pat12.5"); d5 == nil {
		ctx.err("pat12.5: %v", _project(ctx))
	} else if v := d5.value; v == nil {
		ctx.err("pat12.5: %v", _project(ctx))
	} else if a, b, _, c := match(ctx, p, v); sf("%v %v → %v %v %v", p, v, a, b, c) != ".test/*/*/*.h .test/a/b/c/d/e.h → false .test/a/b/c [a b c]" {
		ctx.err("%v %v → %v %v %v", p, v, a, b, c)
	}

	if d := ctx.def("pat.0"); d == nil {
		ctx.err("pat.0: %v", _project(ctx))
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if __string(src(ctx,d),val) != "**.auto" {
		ctx.err("%v", tst{val})
	} else if d1 := ctx.def("pat13.1"); d1 == nil {
		ctx.err("pat13.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, _, c := match(ctx, val, v); sf("%v %v → %v %v %v", val, v, a, b, c) != "**.auto .test/a/b/c.auto → true .test/a/b/c.auto [.test/a/b/c]" {
		ctx.err("%v %v → %v %v %v", val, v, a, b, c)
	} else if d2 := ctx.def("pat13.2"); d2 == nil {
		ctx.err("pat13.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, _, c := match(ctx, val, v); sf("%v %v → %v %v %v", val, v, a, b, c) != "**.auto .test/a/b/c.test → false .test/a/b/c.test [.test/a/b/c.test]" {
		ctx.err("%v %v → %v %v %v", val, v, a, b, c)
	}
}

func testOptional(ctx *testcase) {
	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{19:10 {1:8:project foo}}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{20:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{21:10 {3:9 {1:8:self foo}}}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{22:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{23:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{24:10:null}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{25:10 {25:27 {3:9 {1:8:self foo}}}}"; s != t {
		ctx.err("%v %v → %s != %s", tst{d}, v, s, t)
	}

	if d := ctx.def("val7"); d == nil {
		ctx.err("val7")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{26:10 {26:27 {3:9 {1:8:self foo}}}}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val8"); d == nil {
		ctx.err("val8")
	} else if d.value == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(d.value), "{27:10 {27:12:null}}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val10"); d == nil {
		ctx.err("val10")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val11"); d == nil {
		ctx.err("val11")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}

	if d := ctx.def("val12"); d == nil {
		ctx.err("val12")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=yes}"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%v → %s != %s", tst{d}, s, t)
	}
}

func testPlaceholders(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$1 $2 $3 $4 $5 $6 $7 $8 $9"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "_", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 2 3 4 5 6 7 8 9"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach a b c d e f,$_)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "a b c d e f"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach $1 $2 $3 $4 $5 $6 $7 $8 $9,$_)"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	} else if v := ctx.val(d.name, defExpand1, "1", "2", "3", "4", "5", "6", "7", "8", "9", "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "1 2 3 4 5 6 7 8 9"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "a b c d e f"; s != t {
		ctx.err("%v : %s != %s", v, t, s)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s : %s", s, t, ts(v))
	}

	if d := ctx.def("foo1"); d == nil {
		ctx.err("foo1")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val1" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo2"); d == nil {
		ctx.err("foo2")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val2" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo3"); d == nil {
		ctx.err("foo3")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val3" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo4"); d == nil {
		ctx.err("foo4")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val4" {
		ctx.err("%v", v)
	}

	if d := ctx.def("foo5"); d == nil {
		ctx.err("foo5")
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if v.String() != "val5" {
		ctx.err("%v", v)
	}
}

func testLocals(ctx *testcase) {
	s := "foo"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{3:8:word foobar}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{7:9 {3:8:word foobar}}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{9:9 {8:9:word x}}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}

	s = "foo3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := ts(d.value), "{13:9 {3:8:word foobar}}"; s != t {
		ctx.err("%s != %s", tst{d.value}, t)
	}
}

type test_mod_1 struct { modifier_ }
func (ctx *test_mod_1) v(args ...Value) any {
	return append(args, _word(_position(ctx), "test_mod_1"))
}
func testValueModifierInit() {
	modifiers[`test-mod-1`] = reflect.TypeOf((*test_mod_1)(nil)).Elem()
}
func testValueModifier(ctx *testcase) {
	defer func() { delete(modifiers, `test-mod-1`) } ()

	if s := "val"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "foobar" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,d), v); s != "foobar" {
		ctx.err("%v → %s", tst{v}, s)
	}

	if s := "val1"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{(test-mod-1 $(val))}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foobar test_mod_1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if s := "val2"; false {
	} else if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val) test_mod_1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foobar test_mod_1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if s := "val3"; true {
	} else if d := ctx.def(s); d == nil { // TODO: {(plain text) text goes here...}
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*strlit); !y {
		ctx.err("%v", tst{v})
	} else if s := "this is a 'string' of plain  `text`."; s != t.s {
		ctx.err("%v", tst{v})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if s := "val4"; true {
	} else if d := ctx.def(s); d == nil { // TODO: {(plain c++) c++ code goes here...}
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if t, y := v.(*strlit); !y {
		ctx.err("%v", tst{v})
	} else if s := "int main() { return 0; }"; s != t.s {
		ctx.err("%v", tst{v})
	} else if t := v.String(); s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}
}

func testTemplate(ctx *testcase) {
	var s string

	s = "xyz"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand2 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx yyy zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var.xxx"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "-xxx" {
		ctx.err("%v", tst{x.elems[1]})
	} else if s, t := v.String(), "test-xxx"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var.yyy"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "-yyy" {
		ctx.err("%v", tst{x.elems[1]})
	} else if s, t := v.String(), "test-yyy"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var.zzz"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*compound); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 2 {
		ctx.err("%v", tst{v})
	} else if x.elems[0].String() != "test" {
		ctx.err("%v", tst{x.elems[0]})
	} else if x.elems[1].String() != "-zzz" {
		ctx.err("%v", tst{x.elems[1]})
	} else if s, t := v.String(), "test-zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "vars"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 3 {
		ctx.err("%v ; %d", tst{v}, x.len())
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx yyy zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "), "xxx yyy zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "var2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), ".self .usee var.zzz var2 vars xyz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx xxx yyy yyy zzz zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx xxx yyy yyy zzz zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := strings.Join(sortstrs(strings.Fields(v.String()))," "), "xxx xxx yyy yyy zzz zzz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := strings.Join(sortstrs(strings.Fields(__string(ctx,v)))," "); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a=x a=y a=z b=x b=y b=z c=x c=y c=z"; s != t {
		ctx.err("%s: %s != %s → %s", d.name, s, t, tst{v})
	}

	s = ".test.5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a=x a=y a=x b=y b=x b=y"; s != t {
		ctx.err("%s: %s != %s → %v", d.name, s, t, tst{v})
	}

	s = ".test.6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a=x a=y b=x b=x c=y c=x"; s != t {
		ctx.err("%s: %v → %s", d.name, tst{v}, s)
	}

	s = ".test.7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x c2=x c1=y c2=y c1=z c2=z"; s != t {
		ctx.err("%s: %s != %s → %v", d.name, s, t, tst{v})
	}

	s = ".test.8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z {}=x c2=x {}=y c2=y {}=z c2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c2=x c2=y c2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.9"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.10"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x {}=x c1=y {}=y c1=z {}=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z c1=x c1=y c1=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.11"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x a2=x a1=y a2=y a1=z a2=z b1=x b2=x b1=y b2=y b1=z b2=z"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.12"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x1 a2=x2 a1=y1 a2=y2 a1={} a2=z2 b1=x1 b2=x2 b1=y1 b2=y2 b1={} b2=z2 c1=x1 {}=x2 c1=y1 {}=y2 c1={} {}=z2"; s != t {
		ctx.err("%s: %s != %s | %v", d.name, s, t, tst{v})
	} else if s, t := __string(ctx,v), "a1=x1 a2=x2 a1=y1 a2=y2 a2=z2 b1=x1 b2=x2 b1=y1 b2=y2 b2=z2 c1=x1 c1=y1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.13"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d.o, tst{d})
	} else if v := d.value; v == nil {
		ctx.err("%v %v", d.o, tst{d})
	} else if s, t := v.String(), "a1=x1 a2=x2 a1=y1 a2=y2 b1=x1 b2=x2 b1=y1 b2=y2"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func testTemplateForeach(ctx *testcase) {
	var s string
	var proj = _project(ctx)

	s = ".test.a"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if len(t) != 1 {
		ctx.err("%s %v", s, t)
	} else if x, y := t[0].(matched_rule); !y {
		ctx.err("%v", tst{t[0]})
	} else if x.String() != s {
		ctx.err("%v", tst{x.rule})
	} else if r := ctx.rule(s); r == nil {
		ctx.err(s)
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := t.target.(*compound); !y {
		ctx.err("%v", tst{t.target})
	} else if t.target.String() != ".test.a" {
		ctx.err("%v", tst{t.target})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program[0].depends)
	} else if t.program[0].depends[0].String() != "a" {
		ctx.err("%v: %v", t.target, t.program[0].depends[0])
	} else if t.program[0].depends[1].String() != "$(foreach a d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print a $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.a" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.b"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := t.target.(*compound); !y {
		ctx.err("%v", tst{t.target})
	} else if t.target.String() != ".test.b" {
		ctx.err("%v", tst{t.target})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program[0].depends)
	} else if t.program[0].depends[0].String() != "b" {
		ctx.err("%v: %v", t.target, t.program[0].depends[0])
	} else if t.program[0].depends[1].String() != "$(foreach b d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print b $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.b" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.c"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := t.target.(*compound); !y {
		ctx.err("%v", tst{t.target})
	} else if t.target.String() != ".test.c" {
		ctx.err("%v", tst{t.target})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].depends) != 2 {
		ctx.err("%v: %v", t.target, t.program[0].depends)
	} else if t.program[0].depends[0].String() != "c" {
		ctx.err("%v: %v", t.target, t.program[0].depends[0])
	} else if t.program[0].depends[1].String() != "$(foreach c d e f,foo=$_)" {
		ctx.err("%v: %v", t.target, t.program[0].depends[1])
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print c $^", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	} else if r[0].String() != ".test.c" {
		ctx.err("%v", tst{r[0]})
	}

	s = ".test.a.aaa"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print foa.aaa foa.aaa $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = ".test.b.bbb"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print fob.bbb fob.bbb $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = ".test.c.ccc"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print foc.ccc foc.ccc $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = ".test.o.bar"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := ctx.rule(s); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if t, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if len(t.program) != 1 {
		ctx.err("%v: %v", t.target, t.program)
	} else if len(t.program[0].recipes) != 1 {
		ctx.err("%v: %v", t.target, t.program[0].recipes)
	} else if s, x := "print foo.bar foo.bar $@", t.program[0].recipes[0].String(); s != x {
		ctx.err("%v: %s != %s", t.target, x, s)
	}

	s = "v.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foa.aaa" {
		ctx.err("%v", d)
	}

	s = "v.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "fob.bbb" {
		ctx.err("%v", d)
	}

	s = "v.c"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foc.ccc" {
		ctx.err("%v", d)
	}

	s = "v.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foo.bar" {
		ctx.err("%v", d)
	}

	s = "v1.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "a" {
		ctx.err("%v", d)
	}

	s = "v1.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "b" {
		ctx.err("%v", d)
	}

	s = "v1.c"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "c" {
		ctx.err("%v", d)
	}

	s = "v1.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "o" {
		ctx.err("%v", d)
	}

	s = "v2.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "aaa" {
		ctx.err("%v", d)
	}

	s = "v2.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "bbb" {
		ctx.err("%v", d)
	}

	s = "v2.c"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "ccc" {
		ctx.err("%v", d)
	}

	s = "v2.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "bar" {
		ctx.err("%v", d)
	}

	s = "v.a.aaa"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foa.aaa" {
		ctx.err("%v", d)
	}

	s = "v.b.bbb"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "fob.bbb" {
		ctx.err("%v", d)
	}

	s = "v.c.ccc"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foc.ccc" {
		ctx.err("%v", d)
	}

	s = "v.o.bar"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if __string(ctx, d) != "foo.bar" {
		ctx.err("%v", d)
	}
}

func testValueCache(ctx *testcase) {
	p := _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{*:{g:{o:{l:{.:{0:*.log}}}}},**:{o:{.:{0:**.o}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}}`; s != t {
		note(ctx, "%s", t)
		ctx.err("%v", &p.filemap)
	}

	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if s, t := __string(src(ctx,d), d.value), "**.c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if s, t := __string(src(ctx,d), d.value), "**.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if s, t := __string(src(ctx,d), d.value), "foo.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if s, t := __string(src(ctx,d), d.value), "foo.o"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if s, t := __string(src(ctx,d), d.value), ".deps/xx/yy/zzzzzzzzzz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if s, t := __string(src(ctx,d), d.value), "foo/*.c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if s, t := __string(src(ctx,d), d.value), "foo/*.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(ctx, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}
}

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

func testValueCache3(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{foo:{*:{bar:{0:foo/*/bar}}},&:{0:&(gen)}}`; s != t {
		note(ctx, "%s", t)
		ctx.err("%v", &p.filemap)
	}
}

func test__addprefix(ctx *testcase) {
	var s string

	s = "val1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix -std=,foo)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "-std=foo"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-std=foo"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix -std=,foo bar)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "-std=foo -std=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-std=foo -std=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix -foo=,bar &(none))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v = ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-foo=bar -foo={&(none)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v = ctx.val(d, defExpand2); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "-foo=bar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := v.String(), "$(addprefix std=,&(.test.$1))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "std=test std=null"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "std=ax std=ay std=az"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.⌜a b⌟)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "std=ax std=ay std=bx std=by"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo,bar &(none))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foobar foo{&(none)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foobar"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo bar,=xxx)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=xxx bar=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx bar=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo &(.test.$1),=xxx)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=xxx test=xxx null=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx {&(.test.)}=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=xxx test=xxx null=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx {&(.test.⌜a b⌟)}=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=xxx ax=xxx ay=xxx bx=xxx by=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx ax=xxx ay=xxx bx=xxx by=xxx"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo &(.test.$1),=&(.test.$1))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=test foo=null test=test test=null null=test null=null"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo={&(.test.)} {&(.test.)}={&(.test.)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=test foo=null test=test test=null null=test null=null"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo={&(.test.⌜a b⌟)} {&(.test.⌜a b⌟)}={&(.test.⌜a b⌟)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=ax foo=ay foo=bx foo=by ax=ax ax=ay ax=bx ax=by ay=ax ay=ay ay=bx ay=by bx=ax bx=ay bx=bx bx=by by=ax by=ay by=bx by=by"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val9"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix fo-,&(.test.$1.x.$2.y.$3.z))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "fo-{&(.test.{}.x.{}.y.{}.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "fo-{&(.test.⌜a b c⌟.x.⌜1 2 3⌟.y.0.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), `fo-a1 fo-b2 fo-c3`; s != t {
		note(pc(ctx,v), "%s", ts(v))
		ctx.err("%s != %s", s, t)
	} else if v := ctx.val(d, defExpand2, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val41"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := v.String(), "$(addprefix std=,&(.test.{$1}))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "std=ax std=ay std=az"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)} std={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "std=ax std=ay std=az std=bx std=by std=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val91"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix fo-,&(.test.{$1}.x.{$2}.y.{$3}.z))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "fo-{&({})}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "fo-{&(.test.a.x.1.y.0.z)} fo-{&(.test.a.x.2.y.0.z)} fo-{&(.test.a.x.3.y.0.z)} fo-{&(.test.b.x.1.y.0.z)} fo-{&(.test.b.x.2.y.0.z)} fo-{&(.test.b.x.3.y.0.z)} fo-{&(.test.c.x.1.y.0.z)} fo-{&(.test.c.x.2.y.0.z)} fo-{&(.test.c.x.3.y.0.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "fo-ax fo-ay fo-az fo-bx fo-by fo-bz fo-cx fo-cy fo-cz fo-dx fo-dy fo-dz fo-ex fo-ey fo-ez fo-fx fo-fy fo-fz"; s != t {
		note(pc(ctx,v), "%s", ts(v))
		ctx.err("%s != %s", s, t)
	} else if v := ctx.val(d, defExpand2, []string{"a","b","c"}, []string{"1","2","3"}, "0"); v == nil {
		ctx.err("%v", tst{d})
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = "val81"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addprefix foo &(.test.{$1}),=&(.test.{$1}))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo={&({})} {&({})}={&({})}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo={&(.test.a)} foo={&(.test.b)} {&(.test.a)}={&(.test.a)} {&(.test.a)}={&(.test.b)} {&(.test.b)}={&(.test.a)} {&(.test.b)}={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "foo=ax foo=ay foo=az foo=bx foo=by foo=bz ax=ax ax=ay ax=az ay=ax ay=ay ay=az az=ax az=ay az=az ax=bx ax=by ax=bz ay=bx ay=by ay=bz az=bx az=by az=bz bx=ax bx=ay bx=az by=ax by=ay by=az bz=ax bz=ay bz=az bx=bx bx=by bx=bz by=bx by=by by=bz bz=bx bz=by bz=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a","b"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=ax foo=ay foo=az foo=bx foo=by foo=bz ax=ax ax=ay ax=az ax=bx ax=by ax=bz ay=ax ay=ay ay=az ay=bx ay=by ay=bz az=ax az=ay az=az az=bx az=by az=bz bx=ax bx=ay bx=az bx=bx bx=by bx=bz by=ax by=ay by=az by=bx by=by by=bz bz=ax bz=ay bz=az bz=bx bz=by bz=bz"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

func test__addsuffix(ctx *testcase) {
	var s string

	s = "val1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addsuffix =xxx,foo)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := __string(src(ctx,d),v), "foo=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	s = "val2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addsuffix =xxx,foo bar)"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := __string(src(ctx,d),v), "foo=xxx bar=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo=xxx bar=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}

	s = "val3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(addsuffix =xxx,&(.test.$1))"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{&(.test.{})}=xxx"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := __string(src(ctx,d),v), ""; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	}
}

func test__closure(ctx *testcase) {
}

func test__contains(ctx *testcase) {
	var s string

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,d), v); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	}
}

func test__contains2(ctx *testcase) {
	var s string

	s = "val"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "${foo}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "${foo}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.y"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.z"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(val foo)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__delegate(ctx *testcase) {
}

func test__foreach(ctx *testcase) {
	{
		var c0 = __foreach{}
		var c1 = partial{&c0, nonePart}
		var c2 = __foreach{}
		c0.evocation = &evocation{automatic{Context:ctx}, nil, nil, nil}
		c2.evocation = &evocation{automatic{Context:c1}, nil, nil, nil}
		if cast[*builtinbase](&c0) == nil {
			ctx.err("builtinbase")
		}
		if cast[partial](ctx).Context != nil {
			ctx.err("partial")
		}
		if cast[partial](&c0).Context != nil {
			ctx.err("partial")
		}
		if cast[partial](c1).Context == nil {
			ctx.err("partial")
		}
		if cast[partial](&c2).Context != nil {
			ctx.err("partial")
		}
	}

	var s string
	var test_1_value Value
	var j = _project(ctx)

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", j, s)
	} else if test_1_value = d.value; test_1_value == nil {
		ctx.err("%v", d)
	} else if s, t := d.value.String(), "x $1 $2 $3 $4 $(foreach $1,&(.test.h)$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if s, t := __string(src(ctx,nil), d.value), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x {} {} {} {} {}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {3:12:word x} {=list {3:14 {1:9:word a}} {3:14 {1:9:word b}} {3:14 {1:9:word c}}} {3:17 {1:9:word X}} {3:20 {1:9:word Y}} {3:23 {1:9:word Z}} {=list {4:12 {=compound {4:25:closure {6:9:def .test.h}} {4:35 {4:22 {1:9:word a}}}}} {4:12 {=compound {4:25:closure {6:9:def .test.h}} {4:35 {4:22 {1:9:word b}}}}} {4:12 {=compound {4:25:closure {6:9:def .test.h}} {4:35 {4:22 {1:9:word c}}}}}}}"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if s, t := v.String(), "x a b c X Y Z &(.test.h)a &(.test.h)b &(.test.h)c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {3:12:word x} {=list {3:14 {1:9:word a}} {3:14 {1:9:word b}} {3:14 {1:9:word c}}} {3:17 {1:9:word X}} {3:20 {1:9:word Y}} {3:23 {1:9:word Z}} {=list {4:12 {=compound {4:25 {=flag {6:13}}} {4:35 {4:22 {1:9:word a}}}}} {4:12 {=compound {4:25 {=flag {6:13}}} {4:35 {4:22 {1:9:word b}}}}} {4:12 {=compound {4:25 {=flag {6:13}}} {4:35 {4:22 {1:9:word c}}}}}}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "x a b c X Y Z -a -b -c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err("%v: %s", j, s)
	} else if s0, t := d.value.String(), "x $(foreach q p $(foreach $1,&(.test.h)$_),x$_)"; s0 != t {
		ctx.err("%s != %s : %v", s0, t, tst{d.value})
	} else if s1, t := __string(src(ctx,nil), d.value), "x xq xp"; s1 != t {
		ctx.err("%s != %s : %v", s1, t, tst{d.value})
	} else if x, y := d.value.(*list); !y {
		ctx.err("%v", tst{d.value})
	} else if x.len() != 2 {
		ctx.err("%d %v", x.len(), tst{d.value})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", d)
	} else if s := v.String(); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {7:12:word x} {=list {8:12 {=compound {8:53:word x} {8:54 {8:22:word q}}}} {8:12 {=compound {8:53:word x} {8:54 {8:24:word p}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39:disjunction {8:39:closure {6:9:def .test.h}}} {8:49 {8:36 {1:9:word a}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39:disjunction {8:39:closure {6:9:def .test.h}}} {8:49 {8:36 {1:9:word b}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39:disjunction {8:39:closure {6:9:def .test.h}}} {8:49 {8:36 {1:9:word c}}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x{&(.test.h)}a x{&(.test.h)}b x{&(.test.h)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"a", "b", "c"}, "X", "Y", "Z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {7:12:word x} {=list {8:12 {=compound {8:53:word x} {8:54 {8:22:word q}}}} {8:12 {=compound {8:53:word x} {8:54 {8:24:word p}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39 {=flag {6:13}}} {8:49 {8:36 {1:9:word a}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39 {=flag {6:13}}} {8:49 {8:36 {1:9:word b}}}}}}}} {8:12 {=compound {8:53:word x} {8:54 {8:26 {=compound {8:39 {=flag {6:13}}} {8:49 {8:36 {1:9:word c}}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp x-a x-b x-c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.21"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {10:13:word x} {=list {11:13 {=compound {11:54:word x} {11:55 {11:23:word q}}}} {11:13 {=compound {11:54:word x} {11:55 {11:25:word p}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if l.len() != 2 {
		ctx.err("%v", l.elems)
	} else if s, t := l.elems[0].String(), "x"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[0]})
	} else if s, t := l.elems[1].String(), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t, y := l.elems[1].(*list); !y {
		ctx.err("%s != %s : %v", s, t, tst{l.elems[1]})
	} else if t.len() != 2 {
		ctx.err("%d, %v", t.len(), tst{t})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if !equal(ctx, v, d.value) {
		ctx.err("%v → %v (%v)", tst{v}, d, cmp(ctx, v, d.value))
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"x", "y", "z"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.22"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x x-"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.23"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach q p $(foreach $1,&(.test.xx)$_),x$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {16:12 {=compound {16:54:word x} {16:55 {16:22:word q}}}} {16:12 {=compound {16:54:word x} {16:55 {16:24:word p}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:disjunction {16:39:closure {=compound {16:41:punct .} {16:42:word test} {16:46:punct .} {16:47:word xx}}}} {16:50 {16:36 {1:9:word a}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:disjunction {16:39:closure {=compound {16:41:punct .} {16:42:word test} {16:46:punct .} {16:47:word xx}}}} {16:50 {16:36 {1:9:word b}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:disjunction {16:39:closure {=compound {16:41:punct .} {16:42:word test} {16:46:punct .} {16:47:word xx}}}} {16:50 {16:36 {1:9:word c}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp x{&(.test.xx)}a x{&(.test.xx)}b x{&(.test.xx)}c"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "xq xp"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, []string{"aa", "bb", "cc"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {16:12 {=compound {16:54:word x} {16:55 {16:22:word q}}}} {16:12 {=compound {16:54:word x} {16:55 {16:24:word p}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:null} {16:50 {16:36 {1:9:word aa}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:null} {16:50 {16:36 {1:9:word bb}}}}}}}} {16:12 {=compound {16:54:word x} {16:55 {16:26 {=compound {16:39:null} {16:50 {16:36 {1:9:word cc}}}}}}}}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "xq xp x{}aa x{}bb x{}cc"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xq xp xaa xbb xcc"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "x $(foreach $1,&(.test.$_)$1{}88) y $(foreach $1,&(.test.$_)$1{}99) z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s != v.String() {
		ctx.err("%s != %s : %v", v, s, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {18:12:word x} {=list {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word foo}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}} {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word bar}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}}} {19:44:word y} {=list {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word foo}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}} {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word bar}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}}} {20:44:word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x &(.test.foo)⌜foo bar⌟{}88 &(.test.bar)⌜foo bar⌟{}88 y &(.test.foo)⌜foo bar⌟{}99 &(.test.bar)⌜foo bar⌟{}99 z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "x foo bar88 foo bar88 y foo bar99 foo bar99 z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 5 {
		ctx.err("%d %v", x.len(), v)
	} else if v := ctx.val(d, defExpand1, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {18:12:word x} {=list {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word foo}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}} {19:12 {=compound {19:25:closure {=compound {19:27:punct .} {19:28:word test} {19:32:punct .} {19:33 {19:22 {1:9:word bar}}}}} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}}} {19:44:word y} {=list {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word foo}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}} {20:12 {=compound {20:25:closure {=compound {20:27:punct .} {20:28:word test} {20:32:punct .} {20:33 {20:22 {1:9:word bar}}}}} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}}} {20:44:word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), `x &(.test.foo)⌜foo bar⌟{}88 &(.test.bar)⌜foo bar⌟{}88 y &(.test.foo)⌜foo bar⌟{}99 &(.test.bar)⌜foo bar⌟{}99 z`; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "x foo bar88 foo bar88 y foo bar99 foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if v := ctx.val(d, defExpand2, []string{"foo", "bar"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := ts(v), "{=list {18:12:word x} {=list {19:12 {=compound {19:25:null} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}} {19:12 {=compound {19:25:null} {=list {19:36 {1:9:word foo}} {19:36 {1:9:word bar}}} {19:39:null} {19:40:decimal 88}}}} {19:44:word y} {=list {20:12 {=compound {20:25:null} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}} {20:12 {=compound {20:25:null} {=list {20:36 {1:9:word foo}} {20:36 {1:9:word bar}}} {20:39:null} {20:40:decimal 99}}}} {20:44:word z}}"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := v.String(), "x {}⌜foo bar⌟{}88 {}⌜foo bar⌟{}88 y {}⌜foo bar⌟{}99 {}⌜foo bar⌟{}99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	} else if s, t := __string(src(ctx,nil), v), "x foo bar88 foo bar88 y foo bar99 foo bar99 z"; s != t {
		note(pc(ctx,v), "%s", s)
		note(pc(ctx,v), "%s", t)
		ctx.err("ineq: %v", v)
	}

	s = ".test.4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,&(.test.$_.$(or $4,$3)))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s != v.String() {
		ctx.err("%s != %s : %v", v, s, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.{}) &(.test.y.{})"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", ""); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "", "a"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", "a", []string{}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.a) &(.test.y.a)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa ya"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "x", "y", []string{}, "b"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x.b) &(.test.y.b)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xb yb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach do.smart,&(.test.x $_))"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "&(.test.x do.smart)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "xxx"); v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{28:11 {28:30 {29:11 {=file do.smart}}}}" {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.6"
    if d := ctx.def(s); d == nil || d.value == nil {
		ctx.err("%s: %v", s, d)
	} else if v := ctx.val(d, defExpand1, "x", "y", "z"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{} {} {} {} - x y z {}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "- x y z"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.7"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(foreach a $(foreach $1,&(.test.z)$_{}zz) b,x$_)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa x{&(.test.z)}y1{}zz x{&(.test.z)}y2{}zz x{&(.test.z)}y3{}zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if x.len() != 5 {
		ctx.err("%v, %v", x.len(), v)
	} else if _, y = x.elems[1].(*list); !y && false {
		ctx.err("%v", tst{x.elems[1]})
	} else if v := ctx.val(d, defExpand2, []string{"y1", "y2", "y3"}); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "xa xwy1{}zz xwy2{}zz xwy3{}zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "xa xwy1zz xwy2zz xwy3zz xb"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(stat $1)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "do.smart"); v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "{=file do.smart}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "do.smart"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if ts(v) != "{29:11 {=file do.smart}}" {
		ctx.err("%v", tst{v})
	}
}

func test__foreach1(ctx *testcase) {
	var s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x $1,B b)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s != v.String() {
		ctx.err("%v != %s", tst{v}, s)
	} else if s, t := __string(src(ctx,nil), v), "foo test-foo test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "xxx"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), `&(.test.s) test-foo &(.test.a) &(.test~&(.test.s).a) &(.test.B) &(.test~&(.test.s).B) &(.test.b) &(.test~&(.test.s).b)`; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "foo test-foo test-a test-foo-a test-B test-foo-B test-b test-foo-b"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

func test__foreach2(ctx *testcase) {
	s := ".test.foreach.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $1 $(foreach $2,a$_),b$_)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.foreach.b"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.a x$1 y$1 z$1,xx$2 yy$2)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.foreach.c"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.b) $(foreach &(.test.foreach.x) $(foreach $1 $2,&(.test.foreach.x.$_)),-x$_)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.b)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.c)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.c $1,4)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.4)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "3"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bx{} by{} bz{} baxx{} bayy{} -x{&(.test.foreach.x)} -x{&(.test.foreach.x.3)} -x{&(.test.foreach.x.4)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "bx by bz baxx bayy -xvw -xV -xW"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.foreach.d)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "1", "2"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	if v := ctx.val(".test.foreach.d", defExpand1, "1", "2"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(".test.foreach.d", defExpand1, "1", "2", "a", "b"); v == nil {
		ctx.err(".test.foreach.d")
	} else if s, t := v.String(), "-xa -xb -ya -yb"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}
}

func test__foreach3(ctx *testcase) {
	var s string

	s = ".test.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $1 $2,$_$3)"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{d.value})
	} else {
		if v := ctx.val(d, defExpand1, "a", "b", "cc"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "acc bcc"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := __string(src(ctx,nil), v); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "", []string{"1", "2", "3"}, "x"); v == nil {
			ctx.err("%v", tst{v})
		} else if s, t := v.String(), "1x 2x 3x"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := __string(src(ctx,nil), v); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "a{} b{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "a b"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "a", "b", "x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "ax bx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s := __string(src(ctx,nil), v); s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.x"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%s", d)
	} else if s, t := v.String(), "$(foreach $1 $2,$(addprefix std=,&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", nil); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "std={&(.test.a)} std={&(.test.b)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std={&(.test.if.x)} std={&(.test.if.y)} std={&(.test.if.z)}"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, nil, []string{"if.x", "if.y", "if.z"}); v == nil {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	}

	s = ".test.y"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "$(foreach $1 $2,$(if &(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
		if v := ctx.val(d, defExpand1, "if.x"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.x),std={&(.test.if.x)}) $(if &(.test.{$2}),std=&(.test.{$2}))
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", nil); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.x),std={&(.test.if.x)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.y),std={&(.test.if.y)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.if.x),std={&(.test.if.x)}) $(if &(.test.if.y),std={&(.test.if.y)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "zzz", nil); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.zzz),std={&(.test.zzz)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d}) // $(if &(.test.zzz),std={&(.test.zzz)})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}

	s = ".test.z"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if s, t := d.value.String(), "$(foreach $1 $2,$(if $(.test.$_),std=&(.test.$_)))"; s != t {
		ctx.err("%s != %s ; %s", s, t, tst{v})
	} else {
		if v := ctx.val(d, defExpand1, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.x)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.x)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.y)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=&(.test.if.x) std=&(.test.if.y)"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, "zzz", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand1, nil, "zzz"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", nil); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
		if v := ctx.val(d, defExpand2, "if.x", "if.y"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "std=xxx std=yyy"; s != t {
			ctx.err("%s != %s ; %s", s, t, tst{v})
		}
	}
}

func test__foreach4(ctx *testcase) {
	s := ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	s = ".test.2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{d.value})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "{}"; s != t {
		ctx.err("%v : %v → %s != %s", d.value, tst{d.value}, t, s)
	}

	s = ".test.x"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "$(foreach $(foreach $1 $2,x$_),X$_) $(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%v : %v → %s != %s", d.value, tst{d.value}, t, s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(foreach $(foreach $1 $2,x$_),X$_) $(foreach $(foreach $1 $2,&(.test.x$_)),X$_)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,nil), v); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "Xxa Xxb X{&(.test.xa)} X{&(.test.xb)}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), `Xxa Xxb X~1~ X~2~`; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d, defExpand1, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "YX{&(.test.xa)} YX{&(.test.xb)}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), `YX~1~ YX~2~`; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand2, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "YX~1~ YX~2~"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__foreach5(ctx *testcase) {
	var s string

	s = ".test.o"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "o"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.o.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.o.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.a"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a~ $(foreach $(foreach $1 $2,$(.test.x.o.$_)),-ao$_) ~a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ ~a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a~ -aox.o.a -aox.o.b ~a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,nil), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.b"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ $(foreach $(foreach $1 $2,&(.test.x.o.$_)),-bo$_) ~b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "b~ -bo{&(.test.x.o.a)} -bo{&(.test.x.o.b)} ~b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.0"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else if v := ctx.val(d, defExpand1); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s := "$(.test.x.x $1)"; s != d.value.String() {
		ctx.err("%v", tst{d})
	} else {
		if v := ctx.val(d); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "$(.test.x.x $1)"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		}
		if v := ctx.val(d, "a", "b"); v == nil {
			ctx.err("%v", tst{d})
		} else if s, t := v.String(), "&(.test.x.a) &(.test.x.&(.test.o).a)"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		} else if s, t := __string(src(ctx,nil), v), "a~ ~a x.o.a"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v})
		}
		if v := ctx.val(d, nil, []string{"b", "c"}); v == nil {
			ctx.err("%v → %v", tst{d}, tst{d.value})
		} else if s, t := v.String(), "{}"; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v}) // "&(.test.x.a) &(.test.x.&(.test.o).a) {&(.test.x.b)} {&(.test.x.c)} {&(.test.x.&(.test.o).b)} {&(.test.x.&(.test.o).c)}"
		} else if s, t := __string(src(ctx,nil), v), ""; s != t {
			ctx.err("%s != %s | %v", s, t, tst{v}) // "a~ ~a x.o.a b~ ~b ~c~ x.o.b x.o.c"
		}
	}

	s = ".test.x.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.x $1,$2)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a) &(.test.x.&(.test.o).a) &(.test.x.b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ ~a x.o.a b~ ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.5"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,nil), v); t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b"}, "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", []string{"b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.6"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if x := expand(_final(ctx),v); x == nil {
		ctx.err("%v", tst{v})
	} else if l, y := x.(*list); !y {
		ctx.err("%v", tst{x})
	} else if elems := merge(l.elems...); l.len() != 2 || len(elems) != 4 {
		for i, v := range elems { note(ctx, "%d. %v", i, tst{v}) }
		ctx.err("%v ; %d, %d", tst{x}, l.len(), len(elems))
	} else if s, t := x.String(), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), x), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, []string{"a", "b", "c"}); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a b c,{}) &(.test.x.&(.test.o).a) &(.test.x.b a b c,{}) &(.test.x.&(.test.o).b) &(.test.x.c a b c,{}) &(.test.x.&(.test.o).c)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b -aox.o.c ~a x.o.a b~ -box.o.a -box.o.b -box.o.c ~b x.o.b ~c~ x.o.c"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.7"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y ,$2)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.8"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(.test.x.y $1,$2)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), ""; t != "" {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b) &(.test.x.&(.test.o).a) &(.test.x.b a,b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.10"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if s, t := d.value.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{d.value})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,{}) &(.test.x.&(.test.o).a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a ~a x.o.a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.11"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.b {},b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "b~ -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.x.12"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b) &(.test.x.&(.test.o).a) &(.test.x.b a,b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if v := ctx.val(d, "a", "b"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(.test.x.a a,b) &(.test.x.&(.test.o).a) &(.test.x.b a,b) &(.test.x.&(.test.o).b)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,nil), v), "a~ -aox.o.a -aox.o.b ~a x.o.a b~ -box.o.a -box.o.b ~b x.o.b"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__if(ctx *testcase) {
	var s string

	s = "x1"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(if {=yes},yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x2"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(if {=no},yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x3"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x4"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x5"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x6"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x7"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{11:8 {11:25:word no}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x8"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{12:8 {12:25:word no}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
	s = "x81"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := ts(v), "{20:9 {20:22:word yes}}"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s := __string(src(ctx,d),v); s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x9"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x10"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(ifarg 1,yes,no)"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if v := ctx.val(d, defExpand1, "a"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "yes"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x11"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}

	s = "x12"
	if d := ctx.def(s); d == nil {
		ctx.err("%s", s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d),v), "no"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{v})
	}
}

func test__join(ctx *testcase) {
	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(join foo bar xx yy zz,-)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foo-bar-xx-yy-zz"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "&(target.arch)-&(target.vendor)-&(target.os)-&(target.abi)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foo-bar--0"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "{{&(target.arch) &(XXX) &(target.vendor) &(target.os) &(target.abi)}-}"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "foo-bar-0"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__logic(ctx *testcase) {
	s := "val1"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or &(none),a)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
		// ...
		// ...
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val6"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(and $1,$2,$3)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,d), v); t != "" {
		ctx.err("%v → %s", tst{v}, t)
	} else if v := ctx.val(d, "a", "b", "c"); v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "c"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if t := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "(variant/$(or $(base &(variant)),bootstrap))"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := expand(_final(src(ctx,d)),v), "(variant/bootstrap)"; s.String() != t {
		ctx.err("%s != %s | %v", s, t, tst{s})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "(variant/bootstrap)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "variant/bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(src(ctx,d), v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x3"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x4"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(or $(base &(variant)),bootstrap)"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "x5"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "$(base $(or &(variant),bootstrap))"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "bootstrap"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__or(ctx *testcase) {
	if v := ctx.val("val11.0"); v == nil {
		ctx.err("val11.0")
	} else if v.String() != "-no -yes -false -true" {
		ctx.err("%v", tst{v})
	} else if s, t := __string(src(ctx,nil),v), "-no -yes -false -true"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if l, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else {
		for _, t := range l.elems {
			if f, y := t.(flag); !y {
				ctx.err("%v", tst{t})
			} else if _, y := f.Value.(*word); !y {
				ctx.err("%v", tst{f.Value})
			} else if !__true(ctx,f) {
				ctx.err("%v", tst{t})
			} else if !__true(ctx,f.Value) {
				ctx.err("%v", tst{f.Value})
			}
		}
	}

	if v := ctx.val("val11"); v == nil {
		ctx.err("val11")
	} else if v.String() != "-no" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,nil),v); s != "-no" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}

	if v := ctx.val("val12"); v == nil {
		ctx.err("val12")
	} else if v.String() != "-yes" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,nil),v); s != "-yes" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}

	if v := ctx.val("val13"); v == nil {
		ctx.err("val13")
	} else if v.String() != "-false" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,nil),v); s != "-false" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}
}

func test__trimprefix(ctx *testcase) {
	var pv Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if pv = p.value; pv == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = __string(ctx,pv); ps == "" {
		ctx.err("%v", tst{pv})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{pv})
		return
	} else if s := "/testdata/builtins/trimprefix"; !strings.HasSuffix(ps, s) {
		ctx.err("%v → %s", tst{pv}, ps)
		return
	}

	s := "pat0"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*globpat); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*word); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := __string(ctx, p), "**/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, _, c := match(ctx, v, pv); sf("%v %v %v", a, b, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	s = "pat1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*percpat); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*word); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if s, t := __string(ctx, p), "%%/testdata"; s != t {
		ctx.err("%v → %s != %s", tst{p}, t, s)
	} else if a, b, _, c := match(ctx, v, pv); sf("%v %v %v", a, b, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	s = "pat2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if t, y := p.elems[0].(*punct); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if t.token != PROOT {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if t.len() != 1 {
		ctx.err("%v ; %v", tst{t}, t.len())
	} else if _, y := t.elems[0].(*globmeta); !y {
		ctx.err("%v", tst{t.elems[0]})
	} else if _, y := p.elems[2].(*word); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if s, t := p.String(), "/**/testdata"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "/**/testdata"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, _, c := match(ctx, v, pv); sf("%v %v %v", a, b, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	s = "pat3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if t, y := p.elems[0].(*punct); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if t.token != PROOT {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if _, y := p.elems[2].(*word); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if s, t := p.String(), "/%%/testdata"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "/%%/testdata"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, _, c := match(ctx, v, pv); sf("%v %v %v", a, b, c) != "false <nil> []" {
		ctx.err("%v %v: %v %v %v", p, v, a, b, c)
	}

	s = "val1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val6"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val7"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(ctx,v), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__trimsuffix(ctx *testcase) {
	var pv Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if pv = p.value; pv == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = __string(ctx,pv); ps == "" {
		ctx.err("%v", tst{pv})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{pv})
		return
	} else if s := "/testdata/builtins/trimsuffix"; !strings.HasSuffix(ps, s) {
		ctx.err("%v → %s", tst{pv}, ps)
		return
	}

	s := "pat0"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "testdata/**"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/**"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, _, c := match(reversal{ctx}, v, pv); a {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	} else if s, t := sf("%v %v", b, c), "[testdata builtins trimsuffix] [builtins/trimsuffix]"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	}

	s = "pat1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 2 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if s, t := p.String(), "testdata/%%"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/%%"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, _, c := match(reversal{ctx}, v, pv); a {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	} else if s, t := sf("%v %v", b, c), "[testdata builtins trimsuffix] [builtins/trimsuffix]"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	}

	s = "pat2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if x, y := p.elems[1].(*globpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if x.len() != 1 {
		ctx.err("%v ; %v", tst{x}, x.len())
	} else if x, y := p.elems[2].(*punct); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if x.token != PTAIL {
		ctx.err("%v ; %v", tst{x}, x.token)
	} else if x.String() != "" {
		ctx.err("%v ; %v", tst{x}, x.token)
	} else if s, t := p.String(), "testdata/**/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/**/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, _, c := match(reversal{ctx}, v, pv); a { // partially matched
		ctx.err("%v %v → %v %v", tst{v}, v, b, c)
	} else if b != nil || c != nil {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	}

	s = "pat3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if p, y := v.(*path); !y {
		ctx.err("%v", tst{v})
	} else if p.len() != 3 {
		ctx.err("%v ; %v", tst{p}, p.len())
	} else if _, y := p.elems[0].(*word); !y {
		ctx.err("%v", tst{p.elems[0]})
	} else if _, y := p.elems[1].(*percpat); !y {
		ctx.err("%v", tst{p.elems[1]})
	} else if t, y := p.elems[2].(*punct); !y {
		ctx.err("%v", tst{p.elems[2]})
	} else if t.token != PTAIL {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if t.String() != "" {
		ctx.err("%v ; %v", tst{t}, t.token)
	} else if s, t := p.String(), "testdata/%%/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if s, t := __string(ctx, p), "testdata/%%/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{p})
	} else if a, b, _, c := match(reversal{ctx}, v, pv); a {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	} else if b != nil || c != nil {
		ctx.err("%v → %v %v | %v", pv, b, c, tst{v})
	}

	var ds string
	if v := ctx.val("d"); v == nil {
		ctx.err("d")
	} else if x, y := unbox(v).(*path); !y {
		ctx.err("%v", tst{v})
	} else if x.len() < 2 {
		ctx.err("%v", tst{x})
	} else if t, y := x.elems[0].(*punct); !y {
		ctx.err("%v", tst{x.elems[0]})
	} else if PROOT != t.token {
		ctx.err("%v ; %v", tst{t}, x)
	} else if ds = __string(ctx,v); ds == "" {
		ctx.err("%v", tst{v})
	} else if !strings.HasPrefix(ds, "/") {
		ctx.err("%v %s", tst{v}, ds)
	}

	s = "val1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val6"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s := __string(ctx,v); s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__wildcard(ctx *testcase) {
	var m = _project(ctx)
	if x, y := m.filemap.get("**"); !y {
		ctx.err("%v", &m.filemap)
	} else if _, y := x.get("."); !y {
		ctx.err("%v", x)
	}

	var (
		pat1 = ctx.val("pat1")
		pat2 = ctx.val("pat2")
		pat3 = ctx.val("pat3")
		pat4 = ctx.val("pat4")
		pat5 = ctx.val("pat5")
		pat6 = ctx.val("pat6")
	)

	if true {
		var f = func(a []Value) { a[0], a[1], a[4], a[5] = pat1, pat2, pat5, pat6 }
		var a = []Value{ nil, nil, nil, nil, nil, nil } ; f(a)
		if a[0] != pat1 { ctx.Errorf("%v", a) }
		if a[1] != pat2 { ctx.Errorf("%v", a) }
		if a[2] != nil  { ctx.Errorf("%v", a) }
		if a[3] != nil  { ctx.Errorf("%v", a) }
		if a[4] != pat5 { ctx.Errorf("%v", a) }
		if a[5] != pat6 { ctx.Errorf("%v", a) }
	}

	if g, y := pat1.(*globpat); !y || g == nil {
		ctx.err("%v %v", pat1, tst{pat1})
	} else if s := __string(ctx,pat1); s != "*.h" {
		ctx.err("%v %v %s", pat1, tst{pat1}, s)
	} else if cs := unmap_files(ctx, m, pat1, nil); len(cs) != 1 {
		ctx.err("%v %v %v %v", pat1, tst{pat1}, cs, &m.filemap)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v %v %v", g, tst{cs[0].pattern}, &m.filemap)
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "**.h" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if __string(ctx, g) != "**.h" {
		ctx.err("%v → %v", tst{pat1}, tst{cs[0].pattern})
	}
	if g, y := pat2.(*globpat); !y || g == nil {
		ctx.err("%v %v", pat2, tst{pat2})
	} else if s := __string(ctx,pat2); s != "**.h" {
		ctx.err("%v %v %s", pat2, tst{pat2}, s)
	} else if cs := unmap_files(ctx, m, pat2, nil); len(cs) != 2 {
		ctx.err("%v %v %v", pat2, tst{pat2}, cs)
	} else if g, y := cs[0].pattern.(*path); !y || g == nil {
		ctx.err("%v", tst{cs[0].pattern})
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "foo/bar/zz/x.h" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if __string(ctx,g) != "foo/bar/zz/x.h" {
		ctx.err("%v → %v", tst{pat2}, tst{cs[0].pattern})
	} else if g, y := cs[1].pattern.(*globpat); !y || g == nil {
		ctx.err("%v", tst{cs[1].pattern})
	} else if m := cs[1].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[1].filemap})
	} else if __string(ctx,m.pattern) != "**.h" {
		ctx.err("%v → %v", tst{cs[1].filemap}, tst{m.pattern})
	} else if __string(ctx,g) != "**.h" {
		ctx.err("%v → %v", tst{pat2}, tst{cs[1].pattern})
	}
	if p, y := pat3.(*path); !y || p == nil {
		ctx.err("%v %v", pat3, tst{pat3})
	} else if s := __string(ctx,pat3); s != "foobar/config/*.def.am" {
		ctx.err("%v %v %s", pat3, tst{pat3}, s)
	} else if false {
		if t := unmap_files(ctx, m, pat3, nil); t != nil {
			ctx.err("%v %v %v", pat3, tst{pat3}, t)
		}
	} else if cs := unmap_files(ctx, m, pat3, nil); len(cs) != 1 {
		ctx.err("%v %v %v", pat3, tst{pat3}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v %v", cs[0].pattern, tst{cs[0].pattern})
	} else if __string(ctx,g) != "**.def.am" {
		ctx.err("%v %v → %v", pat3, tst{pat3}, __string(ctx,g))
	} else if g.String() != "**.def.am" {
		ctx.err("%v %v → %v", pat3, tst{pat3}, g)
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v %v", cs[0].pattern, tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "**.def.am" {
		ctx.err("%v %v → %v", m.pattern, tst{m.pattern}, tst{cs[0].filemap})
	} else if m.pattern.String() != "**.def.am" {
		ctx.err("%v %v → %v", m.pattern, tst{m.pattern}, tst{cs[0].filemap})
	}
	if p, y := pat4.(*path); !y || p == nil {
		ctx.err("%v %v", pat4, tst{pat4})
	} else if s := __string(ctx,pat4); s != "foobar/config/*.def.in" {
		ctx.err("v %v %s", pat4, tst{pat4}, s)
	} else if cs := unmap_files(ctx, m, pat4, nil); len(cs) != 0 {
		// NOTE: because the files spec only defined "**.def.am", no "**.def.in"
		ctx.err("%v %v %v", pat4, tst{pat4}, cs)
	}

	if g, y := pat5.(*globpat); !y || g == nil {
		ctx.err("%v", tst{pat5})
	} else if s := __string(ctx,pat5); s != "*.def.am" {
		ctx.err("%v %s", tst{pat5}, s)
	} else if cs := unmap_files(ctx, m, pat5, nil); len(cs) != 1 {
		ctx.err("%v %v : %v", pat5, tst{pat5}, cs)
	} else if t := cs[0].filemap; t.pattern == nil {
		ctx.err("%v", tst{t})
	} else if _, y := t.pattern.(*globpat); !y {
		ctx.err("%v", tst{t.pattern})
	} else if __string(ctx,t.pattern) != "**.def.am" {
		ctx.err("%v → %v", tst{pat5}, t.pattern)
	} else if a, b, _, c := match(ctx, pat5, pat3); sf("%v %v %v %v %v", pat5, pat3, a, b, c) != "*.def.am foobar/config/*.def.am false foobar [foobar]" {
		ctx.err("%v %v %v %v %v", pat5, pat3, a, b, c)
	} else if a, b, _, c := match(ctx, pat5, pat4); sf("%v %v %v %v %v", pat5, pat4, a, b, c) != "*.def.am foobar/config/*.def.in false foobar [foobar]" {
		ctx.err("%v %v %v %v %v", pat5, pat4, a, b, c)
	}
	if g, y := pat6.(*globpat); !y || g == nil {
		ctx.err("%v", tst{pat6})
	} else if s := __string(ctx,pat6); s != "**.def.am" {
		ctx.err("%v %s", tst{pat6}, s)
	} else if cs := unmap_files(ctx, m, pat6, nil); len(cs) != 1 {
		ctx.err("%v %v : %v", pat6, tst{pat6}, cs)
	} else if g, y := cs[0].pattern.(*globpat); !y || g == nil {
		ctx.err("%v", tst{cs[0].pattern})
	} else if m := cs[0].filemap; m.pattern == nil {
		ctx.err("%v", tst{cs[0].filemap})
	} else if __string(ctx,m.pattern) != "**.def.am" {
		ctx.err("%v → %v", tst{cs[0].filemap}, tst{m.pattern})
	} else if a, b, _, c := match(ctx, pat6, pat3); sf("%v %v %v %v %v", pat6, pat3, a, b, c) != "**.def.am foobar/config/*.def.am true foobar/config/*.def.am [foobar/config/*]" {
		ctx.err("%v %v %v %v %v", pat6, pat3, a, b, c)
	} else if a, b, _, c := match(ctx, pat6, pat4); sf("%v %v %v %v %v", pat6, pat4, a, b, c) != "**.def.am foobar/config/*.def.in false foobar/config/*.def.in [foobar/config/*.def.in]" {
		ctx.err("%v %v %v %v %v", pat6, pat4, a, b, c)
	}

	if s := _workdir(ctx); s == "" {
		ctx.err("workdir")
	} else if !filepath.IsAbs(s) {
		ctx.err("workdir: %v", s)
	}
	if v := ctx.val("top"); v == nil {
		ctx.err("top")
	} else if v.String() != _workdir(ctx) {
		ctx.err("%v", v)
	}
	if v := ctx.val("inc"); v == nil {
		ctx.err("inc")
	} else if v.String() != _workdir(ctx)+"/inc" {
		ctx.err("%v", v)
	}

	const N = 1
	var wg sync.WaitGroup
	var workdirInc = _workdir(ctx) + "/inc"
	invalid := func(name string) bool { return name == "" ||
		name != "foobar/config/a.def.in" &&
		name != "foobar/config/b.def.in" ;}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				b.directory(workdirInc, pat3)
				if a := b.files; len(a) != 1 {
					ctx.err("_wildcard(%v) (%d): %v", pat3, n, a)
				} else if ident(ctx,a[0]) != "foobar/config/a.def.am" {
					ctx.err("_wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				b.directory(workdirInc, pat4)
				if a := b.files; len(a) != 2 {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a)
				} else if invalid(ident(ctx,a[0])) {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[0])
				} else if invalid(ident(ctx,a[1])) {
					ctx.err("_wildcard(%v) (%d): %v", pat4, n, a[1])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{dir:workdirInc}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat3); len(a) != 1 {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
				} else if ident(ctx,a[0]) != "foobar/config/a.def.am" {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{dir:workdirInc}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat4); len(a) != 2 {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a)
				} else if invalid(ident(ctx,a[0])) {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a[0])
				} else if invalid(ident(ctx,a[1])) {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a[1])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat3); len(a) != 1 {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a)
				} else if ident(ctx,a[0]) != "foobar/config/a.def.am" {
					ctx.err("wildcard(%v) (%d): %v", pat3, n, a[0])
				}
			} (i)
		}
	}
	{
		c := original{ctx,defExpand1}
		wg.Add(N)
		for i := 0; i < N; i += 1 {
			go func(n int) {
				defer wg.Done()
				b := __wildcard{}
				b.evocation = &evocation{automatic{Context:c}, nil, nil, nil}
				if a := b._do(pat4); a != nil {
					ctx.err("wildcard(%v) (%d): %v", pat4, n, a)
				}
			} (i)
		}
	}
	wg.Wait()

	var (
		val1 = ctx.val("val1")
		val2 = ctx.val("val2")
		val3 = ctx.val("val3")
		val4 = ctx.val("val4")
		val5 = ctx.val("val5")
	)
	if s := __string(ctx,val1); s == "" {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/bar.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/v1.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/v2.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val1, tst{val1})
	} else if strings.Count(s, "inc/foo/bar/zz/x.h") > 1 {
		ctx.err("%v %v", val1, tst{val1})
	}

	if s := __string(ctx,val2); s == "" {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/bar.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/v1.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/v2.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	} else if strings.Count(s, "inc/foo/bar/zz/x.h") != 1 {
		ctx.err("%v %v", val2, tst{val2})
	}

	if s := __string(ctx,val3); s == "" {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/v1.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/v2.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
		ctx.err("%v %v", val3, tst{val3})
	}

	if s := __string(ctx,val4); s == "" {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/v1.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/v2.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/bar/v1.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/bar/v2.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	} else if strings.Count(s, "foo/bar/zz/x.h") != 1 {
		ctx.err("%v %v", val4, tst{val4})
	}

	if s := __string(ctx,val5); s == "" {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "bar.h") != 1 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo.h") != 1 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/v1.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/v2.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/bar/v1.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/bar/v2.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	} else if strings.Count(s, "foo/bar/zz/x.h") != 0 {
		ctx.err("%v %v", val5, tst{val5})
	}

	var (
		fix1 = ctx.val("fix1")
		fix2 = ctx.val("fix2")
		fix3 = ctx.val("fix3")
		fix4 = ctx.val("fix4")
	)
	if s := __string(ctx,fix1); s == "" {
		ctx.err("fix1: %v", fix1)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix1: %v", fix1)
	}
	if s := __string(ctx,fix2); s != "" {
		// NOTE: because the files spec defines only "**.def.am", no "**.def.in"
		ctx.err("fix2: %v", fix2)
	}
	if s := __string(ctx,fix3); s == "" {
		ctx.err("fix3: %v", fix3)
	} else if strings.Count(s, "foobar/config/a.def.am") != 1 {
		ctx.err("fix3: %v", fix3)
	}
	if s := __string(ctx,fix4); s == "" {
		ctx.err("fix4: %v", fix4)
	} else if strings.Count(s, "foobar/config/a.def.in") != 1 {
		ctx.err("fix4: %v", fix4)
	} else if strings.Count(s, "foobar/config/b.def.in") != 1 {
		ctx.err("fix4: %v", fix4)
	}
}

func test__wildcard1(ctx *testcase) {
	var p = _project(ctx)
	if x, y := p.filemap.get("**"); y {
		ctx.err("%v %v", &p.filemap, x)
	} else if s, t := p.filemap.String(), `{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{.:{h:{0:foo/*.h}}},**:{.:{hh:{0:foo/**.hh}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}}`; s != t {
		ctx.err("%s != %s", s, t)
	}
}

func test__wildcard2(ctx *testcase) {
	var p = _project(ctx)
	if x, y := p.filemap.get("**"); y {
		ctx.err("%v %v", &p.filemap, x)
	} else if s, t := p.filemap.String(), `{foo:{b:{*:{v:{*:{.:{h:{0:foo/b*/v*.h}}}}}},x:{*:{y:{.:{h:{0:foo/x*y.h}}}}}},f:{*?:{x:{.:{h:{0:f*?/x.h}}}}}}`; s != t {
		ctx.err("%s != %s", s, t)
	}
}

func test__wildcard3(ctx *testcase) {
	var p = _project(ctx)
	if x, y := p.filemap.get("**"); y {
		ctx.err("%v %v", &p.filemap, x)
	} else if s, t := p.filemap.String(), `{foo:{ba:{*:{v:{?:{.:{h:{0:foo/ba*/v?.h}}}}},?:{xyz:{*:{.:{txt:{0:foo/ba?/xyz*.txt}}}}}}}}`; s != t {
		ctx.err("%s != %s", s, t)
	}
}

func test__xor(ctx *testcase) {
	if d := ctx.def("val14.1"); d == nil {
		ctx.err("val14.1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if __true(ctx, v) {
		ctx.err("%v", tst{v})
	} else if t := v.String(); t != "{}" {
		ctx.err("%v ⇒ %s", tst{v}, t)
	} else if t := __string(src(ctx,d), v); t != "" {
		ctx.err("%v ⇒ %s", tst{v}, t)
	}

	if d := ctx.def("val14.2"); d == nil {
		ctx.err("val14.2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if !__true(ctx, v) {
		ctx.err("%v", tst{v})
	} else if s, t := v.String(), "{=true}"; s != t {
		ctx.err("%v ⇒ %s != %s", tst{v}, t, s)
	} else if s, t := __string(src(ctx,d), v), "true"; s != t {
		ctx.err("%v ⇒ %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val14.3"); d == nil {
		ctx.err("val14.3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=true}" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,d), v); s != "true" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}

	if d := ctx.def("val14.4"); d == nil {
		ctx.err("val14.4")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{}" {
		ctx.err("%v", tst{v})
	} else if s := __string(src(ctx,d), v); s != "" {
		ctx.err("%v ⇒ %s", tst{v}, s)
	}
}

func test__file0(ctx *testcase) {
	var proj = _project(ctx)
	if pat, str := ".test/a/**.c", ".test/a/b/c/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", str, pat) {
		ctx.err("%v %v != %v", str, pat, m.pattern, m.value)
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err(str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", str, pat) {
		ctx.err("%v %v != %v", str, pat, m.pattern, m.value)
	} else if   s := "val1.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if s := __string(ctx, val); s != str {
		ctx.err("%v: %s != %s", tst{val}, s, str)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if   s := "val1.2" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if __string(ctx, val) != str {
		ctx.err("%v", tst{val})
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	}

	if pat, str := ".test/xx/*.c", ".test/xx/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", str, pat) {
		ctx.err("%v %v != %v", str, pat, m.pattern, m.value)
	} else if   s := "val2.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if   s := "val2.2" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	}

	if pat, str := ".test/xx/yy/*.c", ".test/xx/yy/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", str, pat) {
		ctx.err("%v %v != %v", str, pat, m.pattern, m.value)
	} else if   s := "val3" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	}

	if pat, str := ".test/xx/yy/zz/*.c", ".test/xx/yy/zz/foo.c"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("unmap_files %s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", str, pat) {
		ctx.err("%v %v != %v", str, pat, m.pattern, m.value)
	} else if   s := "val4" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if f, y := val.(*file); !y {
		ctx.err("%v", tst{val})
	} else if f.filemap == nil {
		ctx.err("%v", f)
	} else if f.filemap.String() != pat {
		ctx.err("%v: %v", f, f.filemap)
	} else if val.String() != "{=file "+str+"}" {
		ctx.err("%v", tst{val})
	}

	if s := "val5" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if _, y := val.(*null); !y {
		ctx.err("%v", tst{val})
	} else if val.String() != "{}" {
		ctx.err("%v", tst{val})
	} else if __string(ctx, val) != "" {
		ctx.err("%v", tst{val})
	}

	if pat, str := "**.auto", ".test/a/b/c.auto"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("%s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", str, pat) {
		ctx.err("%v %v != %v", str, pat, m.pattern, m.value)
	} else if s := "p1" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if __string(ctx, x) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, proj, v, nil); t == nil {
		ctx.err("%v %v", v, tst{v})
	} else if len(t) != 1 {
		ctx.err("%v %v %v", v, tst{v}, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", str, pat) {
		ctx.err("%v %v != %v", str, pat, m.pattern, m.value)
	}

	if str := ".test/a/b/c.none" ; false {} else
	if t := unmap_files(ctx, proj, str, nil); t != nil {
		ctx.err("%v", str)
	} else if s := "p2" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v", s)
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if __string(ctx, x) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, proj, v, nil); t != nil {
		ctx.err("%v %v", v, tst{v})
	}
}

func test__file(ctx *testcase) {
	var fullFooTxt = filepath.Join(_workdir(ctx), "foo.txt")

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if f := (as{v}.file(ctx)); f == nil {
		ctx.err("%v", tst{v})
	} else if o := (as{v}.fullname(ctx)); o.Value == nil {
		ctx.err("%v ; %v", tst{v}, tst{o.Value})
	} else if f, y := o.Value.(*file); !y || f == nil {
		ctx.err("%v", tst{o.Value})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	} else if cmp(ctx, o, f) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if cmp(ctx, f, o) != cmpEqual {
		ctx.err("%v %v", tst{v}, f)
	} else if p := _pathStr(ctx, f.fullname()); p == nil {
		ctx.err("%v %v", tst{v}, f)
	} else if true {
		// hold line ...
	} else if cmp(ctx, p, o) != cmpEqual {
		ctx.err("%v %v", tst{p}, tst{o})
	} else if cmp(ctx, o, p) != cmpEqual {
		ctx.err("%v %v", tst{o}, tst{p})
	} else if s, t := v.String(), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	} else if s, t := __string(src(ctx,d), v), "foo.txt"; s != t {
		ctx.err("%v → %s != %s", tst{v}, t, s)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=file foo.txt}" {
		ctx.err("%v", tst{v})
	} else if __string(src(ctx,d), v) != "foo.txt" {
		ctx.err("%v", tst{v})
	} else if f, y := v.(*file); !y || f == nil {
		ctx.err("%v", tst{v})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if v.String() != "{=file foo.txt}" {
		ctx.err("%v", tst{v})
	} else if __string(src(ctx,d), v) != fullFooTxt { o, y := v.(fullname)
		ctx.err("%v ; %v %v", tst{v}, tst{o.Value}, y)
	} else if o, y := v.(fullname); !y {
		ctx.err("%v", tst{v})
	} else if f, y := o.Value.(*file); !y || f == nil {
		ctx.err("%v", tst{o.Value})
	} else if f.fullname() != fullFooTxt {
		ctx.err("%v %v", tst{v}, f.fullname())
	}
}
