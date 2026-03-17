//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
    "path/filepath"
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

var modules_s = "/Volumes/workspace/.smart/modules"

var test_mode bool
var testdata_s = testdata_dir()
var testdata_t = testdata_dir_t("1:1")
var testdata_l = strings.Split(testdata_s, pathSep)
var testdata_a = strings.Join(testdata_l, " ")
var testdata_i = func() (_ int) {
	for i, s := range testdata_l { if s == "testdata" { return i } }
	return
} ()

var langs_map = map[string]string{
	"S"     : "c",
	"asm"   : "c",
	"c"     : "c",
	"c++"   : "c++",
	"cc"    : "c++",
	"cpp"   : "c++",
	"cu"    : "cuda",
	"cu++"  : "cuda++",
	"cuda"  : "cuda",
	"cuh"   : "cuda",
	"cuh++" : "cuda++",
	"cxx"   : "c++",
	"m"     : "objc",
	"mm"    : "objc++",
	"s"     : "c",
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
	if i, e := os.Stat(filepath.Join(modules_s, name)); e == nil { res = i.IsDir() }
	return
}

var (
	reLocOuter = regexp.MustCompile(`\{\d+:\d+\s+`)
	reLocInner = regexp.MustCompile(`\{\d+:\d+:`)
)

func unwrapLocStr(s string) string {
	// 1. Strip the outer injection location tags (e.g., "{15:32 " -> "")
	for {
		loc := reLocOuter.FindStringIndex(s)
		if loc == nil { break } // No more outer wrappers found
		
		start := loc[0]       // Index of the opening '{'
		innerStart := loc[1]  // Index immediately after the space
		
		depth := 1
		innerEnd := -1
		
		// Scan forward to find the matching closing brace
		for j := innerStart; j < len(s); j++ {
			if s[j] == '{' {
				depth++
			} else if s[j] == '}' {
				depth--
				if depth == 0 {
					innerEnd = j
					break
				}
			}
		}
		
		if innerEnd != -1 {
			// Rebuild the string without the outer wrapper and its closing brace
			s = s[:start] + s[innerStart:innerEnd] + s[innerEnd+1:]
		} else {
			break // Fallback protection against malformed strings
		}
	}

	// 2. Normalize the inner definition tags (e.g., "{1:1:raw" -> "{=raw")
	return reLocInner.ReplaceAllString(s, "{=")
}

type (
	unwrap_loc_part string
	trim_unloc struct{ int }
	trim_fmt struct{ int ; string }
	trim_prefix trim_fmt
	trim_suffix trim_fmt
)
func testdata_f2(f string, a ...any) unwrap_loc_part { return unwrap_loc_part(testdata_f(f, a...)) }
func testdata_f(f string, _a ...any) string {
	var a []any
	var ss = []string{testdata_s, testdata_a, testdata_dir_t("1:1")}
	for _, _t := range _a { if t, ok := _t.(line_column_s); ok { ss[2] = testdata_dir_t(string(t)) } }
	for _, _t := range _a {
		switch t := _t.(type) {
		case trim_unloc:
			if i := t.int-1; -1 < i && i < len(ss) {
				ss[i] = unwrapLocStr(ss[i])
			}
		case trim_prefix:
			if i := t.int-1; -1 < i && i < len(ss) && t.string != "" {
				ss[i] = strings.TrimPrefix(ss[i], t.string)
			}
		case trim_suffix:
			if i := t.int-1; -1 < i && i < len(ss) && t.string != "" {
				ss[i] = strings.TrimSuffix(ss[i], t.string)
			}
		default: a = append(a, t)
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

func modules_f(f string, _a ...any) string {
	var a []any
	var ss = []string{modules_s, strings.Join(strings.Split(modules_s, pathSep), " ")}
	// for _, _t := range _a { if t, ok := _t.(line_column_s); ok { ss[2] = testdata_dir_t(string(t)) } }
	for _, _t := range _a {
		switch t := _t.(type) {
		case trim_unloc:
			if i := t.int-1; -1 < i && i < len(ss) {
				ss[i] = unwrapLocStr(ss[i])
			}
		case trim_prefix:
			if i := t.int-1; -1 < i && i < len(ss) && t.string != "" {
				ss[i] = strings.TrimPrefix(ss[i], t.string)
			}
		case trim_suffix:
			if i := t.int-1; -1 < i && i < len(ss) && t.string != "" {
				ss[i] = strings.TrimSuffix(ss[i], t.string)
			}
		default: a = append(a, t)
		}
	}
	return sf(f, append([]any{ss[0], ss[1], /* ss[2] */}, a...)...)
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

	if testHasModule("configure") && !ctx.paths.has(modules_s) {
		ctx.paths = append(ctx.paths, modules_s)
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
	} else if evoker(x) {
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
				debug(pc(ctx,e.Value), "%v", e.Value, trace{})
			case trace_evoke_loop_err:
				debug(pc(ctx,e.Value), "evoke loop: %v", e.Value, trace{})
			case traverse_state:
				switch e.uint {
				case traverse_done:
				default:
					debug(pc(ctx,e.p), "%v", tv(e), trace{})
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
	if t, ok := v.(matched_rule); ok { v = t.rule }
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
	run(t, "valcache", "valcache/3/a", "testvalcache.a", testValueCache3a)
	run(t, "valcache", "valcache/4", "testvalcache", testValueCache4)

	// builtins_test.go
	run(t, "builtins", "builtins/file",       "testbuiltins", test__file)
	run(t, "builtins", "builtins/file/0",     "testbuiltins", test__file0)

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


type fooctx  struct { Context }
type foo1ctx struct {  fooctx }
type foo2ctx struct { *fooctx }

func (p *foo1ctx) inner() Context { return p.fooctx }
func (p *foo2ctx) inner() Context { return p.fooctx }

func testInner(t *testing.T) {
	if i := inner(&fooctx{ &fooctx{} }); i == nil {
		t.Fatalf("inner(fooctx{fooctx})")
	} else if i = inner(&fooctx{}); i != nil {
		t.Fatalf("inner(fooctx{}): %v", i)
	}
	if i := inner(&foo1ctx{}); i == nil {
		t.Fatalf("inner(foo1ctx{fooctx{}})")
	} else if _, y := i.(fooctx); !y {
		t.Fatalf("inner(foo1ctx{fooctx{}}): %T", i)
	} else if i = inner(i); i != nil {
		t.Fatalf("inner(foo1ctx{fooctx{}}): %v", i)
	}
	if i := inner(&foo2ctx{ &fooctx{} }); i == nil {
		t.Fatalf("inner(foo2ctx{fooctx{}})")
	} else if _, y := i.(*fooctx); !y {
		t.Fatalf("inner(foo2ctx{fooctx{}}): %T", i)
	} else if i = inner(i); i != nil {
		t.Fatalf("inner(foo2ctx{fooctx{}}): %v", i)
	}
}


func testLoader(ctx *testcase) {
    if s := _workdir(ctx); s == "" {
        ctx.err("empty workdir")
    } else if !strings.HasSuffix(s, "/testdata/empty") {
        ctx.err("incorrect workdir: %s", s)
    } else if d := ctx.def("d"); d == nil {
        ctx.err("d")
    } else if s, t := d.value.String(), fmt.Sprintf("%s/do.smart:5:12:xxx", s); s != t {
        ctx.err("%s != %s", s, t)
    }

    d := (*diagpoint)(nil)
    u := _universe(ctx)
    r := regexp.MustCompile(`^.*?/testdata/empty/do.smart:[0-9]+:[0-9]+: *here…`)
    for _, p := range u.points {
        if r.MatchString(p.message) { d = p ; break }
    }
    if false && d == nil {
        ctx.err("incorrect diagpoints (%d)", len(u.points))
    }
}


func testPositionExample(t *testing.T) {
    src := []byte(`
project foo
include modules/*.smart
`)
    filename := filepath.Join("test", "TestPositionExample")
    fs := _fileset()
    f := fs.AddFile(filename, fs.Base(), len(src))
    f.SetLinesForContent(src)
    if x := f.LineCount(); x < 2 {
        t.Errorf("LineCount: %v < 2", x)
    } else {

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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y .test/xxx-yyy → true .test/xxx-yyy <nil> [xx-yy]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat1.2"); d2 == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y .test/xxx-yyx → false .test/xxx-yyx <nil> [xx-yyx]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d3 := ctx.def("pat1.3"); d3 == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y .test/xxx-yyx/y → true .test/xxx-yyx/y <nil> [xx-yyx/]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d4 := ctx.def("pat1.4"); d4 == nil {
		ctx.err("pat1.4: %v", _project(ctx))
	} else if v := d4.value; v == nil {
		ctx.err("pat1.4: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y .test/xxx-yyx/z → false .test/xxx-yyx/z <nil> [xx-yyx/z]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d5 := ctx.def("pat1.5"); d5 == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if v := d5.value; v == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y .test/xxx/a/b/c/yyy → true .test/xxx/a/b/c/yyy <nil> [xx/a/b/c/yy]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d6 := ctx.def("pat1.6"); d6 == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if v := d6.value; v == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y .test/x/xx-yy/y → true .test/x/xx-yy/y <nil> [/xx-yy/]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	}

	if d := ctx.def("pat1.01"); d == nil {
		ctx.err("pat1.01: %v", _project(ctx))
	} else if v := d.value; v == nil {
		ctx.err("pat1.01: %v", _project(ctx))
	} else if p, y := v.(*path); !y {
		ctx.err("%v : %v", tst{v}, v)
	} else if p.len() != 2 {
		ctx.err("%v", p)
	} else if s, t := p.String(), ".test/x*?y"; s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if d1 := ctx.def("pat1.1"); d1 == nil {
		ctx.err("pat1.1: %v", _project(ctx))
	} else if v := d1.value; v == nil {
		ctx.err("pat1.1: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x*?y .test/xxx-yyy → true .test/xxx-y yy [xx-]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat1.2"); d2 == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat1.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x*?y .test/xxx-yyx → true .test/xxx-y yx [xx-]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d3 := ctx.def("pat1.3"); d3 == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat1.3: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x*?y .test/xxx-yyx/y → true .test/xxx-y yx/y [xx-]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d4 := ctx.def("pat1.4"); d4 == nil {
		ctx.err("pat1.4: %v", _project(ctx))
	} else if v := d4.value; v == nil {
		ctx.err("pat1.4: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x*?y .test/xxx-yyx/z → true .test/xxx-y yx/z [xx-]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d5 := ctx.def("pat1.5"); d5 == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if v := d5.value; v == nil {
		ctx.err("pat1.5: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x*?y .test/xxx/a/b/c/yyy → true .test/xxx/a/b/c/y yy [xx/a/b/c/]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d6 := ctx.def("pat1.6"); d6 == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if v := d6.value; v == nil {
		ctx.err("pat1.6: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x*?y .test/x/xx-yy/y → true .test/x/xx-y y/y [/xx-]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y/z .test/xxx-yyy/z → true .test/xxx-yyy/z <nil> [xx-yy]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat2.2"); d2 == nil {
		ctx.err("pat2.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat2.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y/z .test/xxx/a/b/c/yyy/z → true .test/xxx/a/b/c/yyy/z <nil> [xx/a/b/c/yy]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y/x** .test/xaaa/bbb/ccc/y/xxx/xx → true .test/xaaa/bbb/ccc/y/xxx/xx <nil> [aaa/bbb/ccc/ xx/xx]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat3.2"); d2 == nil {
		ctx.err("pat3.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat3.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y/x** .test/xaabbccy/xabc → true .test/xaabbccy/xabc <nil> [aabbcc abc]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y/x**y .test/xaa/bb/ccy/xaa/bb/ccy → true .test/xaa/bb/ccy/xaa/bb/ccy <nil> [aa/bb/cc aa/bb/cc]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat4.2"); d == nil {
		ctx.err("pat4.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat4.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/x**y/x**y .test/xaaay/x/aaa/y → true .test/xaaay/x/aaa/y <nil> [aaa /aaa/]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/**y/**y/z .test/a/b/cy/a/b/c/y/z → true .test/a/b/cy/a/b/c/y/z <nil> [a/b/c a/b/c/]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/**/**z .test/a/b/c/xyz → true .test/a/b/c/xyz <nil> [a/b/c xy]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*.h .test/a.h → true .test/a.h <nil> [a]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat10.2"); d2 == nil {
		ctx.err("pat10.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat10.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*.h .test/a/b.h → false .test/a /b.h [a]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d3 := ctx.def("pat10.3"); d3 == nil {
		ctx.err("pat10.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat10.3: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*.h .test/a/b/c.h → false .test/a /b/c.h [a]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*.h .test/a.h → false .test/a.h <nil> [a.h]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat11.2"); d2 == nil {
		ctx.err("pat11.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat11.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*.h .test/a/b.h → true .test/a/b.h <nil> [a b]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d3 := ctx.def("pat11.3"); d3 == nil {
		ctx.err("pat11.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat11.3: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*.h .test/a/b/c.h → false .test/a/b /c.h [a b]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*/*.h .test/a.h → false .test/a.h <nil> [a.h]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d2 := ctx.def("pat12.2"); d2 == nil {
		ctx.err("pat12.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("pat12.2: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*/*.h .test/a/b.h → false .test/a/b.h <nil> [a b.h]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d3 := ctx.def("pat12.3"); d3 == nil {
		ctx.err("pat12.3: %v", _project(ctx))
	} else if v := d3.value; v == nil {
		ctx.err("pat12.3: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*/*.h .test/a/b/c.h → true .test/a/b/c.h <nil> [a b c]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d4 := ctx.def("pat12.4"); d4 == nil {
		ctx.err("pat12.4: %v", _project(ctx))
	} else if v := d4.value; v == nil {
		ctx.err("pat12.4: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*/*.h .test/a/b/c/d.h → false .test/a/b/c /d.h [a b c]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
	} else if d5 := ctx.def("pat12.5"); d5 == nil {
		ctx.err("pat12.5: %v", _project(ctx))
	} else if v := d5.value; v == nil {
		ctx.err("pat12.5: %v", _project(ctx))
	} else if a, b, c, d := match(ctx, p, v); sf("%v %v → %v %v %v %v", p, v, a, b, c, d) != ".test/*/*/*.h .test/a/b/c/d/e.h → false .test/a/b/c /d/e.h [a b c]" {
		ctx.err("%v %v → %v %v %v %v", p, v, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, val, v); sf("%v %v → %v %v %v %v", val, v, a, b, c, d) != "**.auto .test/a/b/c.auto → true .test/a/b/c.auto <nil> [.test/a/b/c]" {
		ctx.err("%v %v → %v %v %v %v", val, v, a, b, c, d)
	} else if d2 := ctx.def("pat13.2"); d2 == nil {
		ctx.err("pat13.2: %v", _project(ctx))
	} else if v := d2.value; v == nil {
		ctx.err("%v", d)
	} else if a, b, c, d := match(ctx, val, v); sf("%v %v → %v %v %v %v", val, v, a, b, c, d) != "**.auto .test/a/b/c.test → false .test/a/b/c.test <nil> [.test/a/b/c.test]" {
		ctx.err("%v %v → %v %v %v %v", val, v, a, b, c, d)
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
		ctx.err("%s %v", s, &proj.entries)
	} else if len(t) != 1 {
		ctx.err("%s %v", s, t)
	} else if x, y := t[0].(matched_rule); !y {
		ctx.err("%v", tst{t[0]})
	} else if x.String() != "{=matched_rule .test.a}" {
		ctx.err("%v != {=matched_rule .test.a} ; %v", x, tst{x.rule})
	} else if r := proj._entries(ctx, s, false); r == nil {
		ctx.err("%v: %v", x.target, &proj.entries)
	} else if x, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := x.target.(*compound); !y {
		ctx.err("%v", tst{x.target})
	} else if s, t := x.target.String(), ".test.a"; s != t {
		ctx.err("%s != %s ; %v", s, t, tst{x.target})
	} else if len(x.program) != 1 {
		ctx.err("%v: %v", x.target, x.program)
	} else if len(x.program[0].depends) != 2 {
		ctx.err("%v: %v", x.target, x.program[0].depends)
	} else if x.program[0].depends[0].String() != "a" {
		ctx.err("%v: %v", x.target, x.program[0].depends[0])
	} else if x.program[0].depends[1].String() != "$(foreach a d e f,foo=$_)" {
		ctx.err("%v: %v", x.target, x.program[0].depends[1])
	} else if len(x.program[0].recipes) != 1 {
		ctx.err("%v: %v", x.target, x.program[0].recipes)
	} else if s, c := "print a $^", x.program[0].recipes[0].String(); s != c {
		ctx.err("%v: %s != %s", x.target, c, s)
	} else if r[0].String() != "{=matched_rule .test.a}" {
		ctx.err("%v: %v %v", x.target, r[0], tst{r[0]})
	}

	s = ".test.b"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := proj._entries(ctx, s, false); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if x, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := x.target.(*compound); !y {
		ctx.err("%v", tst{x.target})
	} else if x.target.String() != ".test.b" {
		ctx.err("%v", tst{x.target})
	} else if len(x.program) != 1 {
		ctx.err("%v: %v", x.target, x.program)
	} else if len(x.program[0].depends) != 2 {
		ctx.err("%v: %v", x.target, x.program[0].depends)
	} else if x.program[0].depends[0].String() != "b" {
		ctx.err("%v: %v", x.target, x.program[0].depends[0])
	} else if x.program[0].depends[1].String() != "$(foreach b d e f,foo=$_)" {
		ctx.err("%v: %v", x.target, x.program[0].depends[1])
	} else if len(x.program[0].recipes) != 1 {
		ctx.err("%v: %v", x.target, x.program[0].recipes)
	} else if x.program[0].recipes[0].String() != "print b $^" {
		ctx.err("%v: %s != `print b $^`", x.target, x)
	} else if r[0].String() != "{=matched_rule .test.b}" {
		ctx.err("%v: %v %v", x.target, r[0], tst{r[0]})
	}

	s = ".test.c"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := proj._entries(ctx, s, false); r == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if x, y := r[0].(matched_rule); !y {
		ctx.err("%v", tst{r[0]})
	} else if _, y := x.target.(*compound); !y {
		ctx.err("%v", tst{x.target})
	} else if x.target.String() != ".test.c" {
		ctx.err("%v", tst{x.target})
	} else if len(x.program) != 1 {
		ctx.err("%v: %v", x.target, x.program)
	} else if len(x.program[0].depends) != 2 {
		ctx.err("%v: %v", x.target, x.program[0].depends)
	} else if x.program[0].depends[0].String() != "c" {
		ctx.err("%v: %v", x.target, x.program[0].depends[0])
	} else if x.program[0].depends[1].String() != "$(foreach c d e f,foo=$_)" {
		ctx.err("%v: %v", x.target, x.program[0].depends[1])
	} else if len(x.program[0].recipes) != 1 {
		ctx.err("%v: %v", x.target, x.program[0].recipes)
	} else if x.program[0].recipes[0].String() != "print c $^" {
		ctx.err("%v: %s != `print c $^`", x.target, x)
	} else if r[0].String() != "{=matched_rule .test.c}" {
		ctx.err("%v: %v %v", x.target, r[0], tst{r[0]})
	}

	s = ".test.a.aaa"
	if t := unmap_entries(ctx, proj, s, nil); t == nil {
		ctx.err("%s %v", s, proj.entries.ks())
	} else if r := proj._entries(ctx, s, false); r == nil {
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
	} else if r := proj._entries(ctx, s, false); r == nil {
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
	} else if r := proj._entries(ctx, s, false); r == nil {
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
	} else if r := proj._entries(ctx, s, false); r == nil {
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

	if s, t := p.filemap.String(), `{*:{.:{log:{0:*.log}}},**:{*:{.:{o:{0:**.o}}}},.:{deps:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{?:{0:.deps/??/??/??????????}}}}}}}}}}}}}}}},&:{0:&(gen)},foo:{bar:{.:{c:{0:foo/bar.c},c++:{0:foo/bar.c++}}},.:{c:{0:foo.c},c++:{0:foo.c++}}}}`; s != t {
		ctx.err("%v", &p.filemap)
	}

	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if s, t := __string(src(ctx,d), d.value), "**.c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(&uncache_t{ctx,nil}, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if s, t := __string(src(ctx,d), d.value), "**.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(&uncache_t{ctx,nil}, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if s, t := __string(src(ctx,d), d.value), "foo.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(&uncache_t{ctx,nil}, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if s, t := __string(src(ctx,d), d.value), "foo.o"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(&uncache_t{ctx,nil}, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val4"); d == nil {
		ctx.err("val4")
	} else if s, t := __string(src(ctx,d), d.value), ".deps/xx/yy/zzzzzzzzzz"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(&uncache_t{ctx,nil}, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val5"); d == nil {
		ctx.err("val5")
	} else if s, t := __string(src(ctx,d), d.value), "foo/*.c"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(&uncache_t{ctx,nil}, &p.filemap, d.value); r == nil {
		ctx.err("%v %v", d.value, r)
	}

	if d := ctx.def("val6"); d == nil {
		ctx.err("val6")
	} else if s, t := __string(src(ctx,d), d.value), "foo/*.c++"; s != t {
		ctx.err("%s != %s : %v", s, t, tst{d.value})
	} else if r := _hit(&uncache_t{ctx,nil}, &p.filemap, d.value); r == nil {
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
		ctx.err("%v", &p.filemap)
	}
}

func testValueCache2(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{*:{.:{c++:{0:*.c++}}},**:{*:{.:{c:{0:**.c}}}},?:{?:{?:{0:???}}},foo:{*:{.:{c++:{0:foo/*.c++},xx:{0:foo/*.xx},yy:{0:foo/*.yy}},zzz:{0:foo/*zzz}},?:{?:{?:{?:{?:{.:{c++:{0:foo/??/???.c++},o:{0:foo/?????.o}}}}}}},**:{z:{0:foo/**z}}}}`; s != t {
		ctx.err("%v", &p.filemap)
	}
}

func testValueCache3(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{*:{a:{.:{c:{0:*/a.c}}}},&:{0:&(gen)}}`; s != t {
		ctx.err("%v", &p.filemap)
	}
}

func testValueCache3a(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{}`; s != t {
		ctx.err("%v", &p.filemap)
	}
}

func testValueCache4(ctx *testcase) {
	var p = _project(ctx)

	if p == nil {
		ctx.err("nil project")
	}

	if s, t := p.filemap.String(), `{foo:{0:foo,&:{0:foo&(va2)bar,1:foo/&(va3),2:foo/&(va1)/xx&(va2)yy/&(va3)zz}},&:{0:&(va1)}}`; s != t {
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
	} else if s, t := v.String(), "a b c $1"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	} else if s, t := __string(src(ctx,d), v), "a b c"; s != t {
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
	} else if s, t := v.String(), "a b c foo"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = ".test.z"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a b c foo"; s != t {
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
	}

	s = "val2"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "a"; s != t {
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
	}

	s = "x1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "(variant/bootstrap)"; s != t {
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
	}

	if v := ctx.val("val12"); v == nil {
		ctx.err("val12")
	} else if v.String() != "-yes" {
		ctx.err("%v", tst{v})
	}

	if v := ctx.val("val13"); v == nil {
		ctx.err("val13")
	} else if v.String() != "-false" {
		ctx.err("%v", tst{v})
	}
}

func test__trimprefix(ctx *testcase) {
	var root Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if root = p.value; root == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = __string(ctx,root); ps == "" {
		ctx.err("%v", tst{root})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{root})
		return
	} else if s := "/testdata/builtins/trimprefix"; !strings.HasSuffix(ps, s) {
		ctx.err("%v → %s", tst{root}, ps)
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
	} else if a, b, c, d := match(ctx, p, root); sf("%v %v %v %v", a, b, c, d) != "true /Volumes/workspace/go/src/extbit.io/smart/testdata /builtins/trimprefix [/Volumes/workspace/go/src/extbit.io/smart]" {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
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
	} else if true {
		// skip...
	} else if a, b, c, d := match(ctx, p, root); sf("%v %v %v %v", a, b, c, d) != "false /Volumes/workspace/go/src/extbit.io/smart/testdata builtins/trimprefix [/Volumes/workspace/go/src/extbit.io/smart]" {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
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
	} else if a, b, c, d := match(ctx, p, root); sf("%v %v %v %v", a, b, c, d) != "true /Volumes/workspace/go/src/extbit.io/smart/testdata /builtins/trimprefix [Volumes/workspace/go/src/extbit.io/smart]" {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
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
	} else if true {
		// skip...
	} else if a, b, c, d := match(ctx, p, root); sf("%v %v %v %v", a, b, c, d) != "false /Volumes/workspace/go/src/extbit.io/smart/testdata builtins/trimprefix [Volumes/workspace/go/src/extbit.io/smart]" {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
	}

	s = "val1"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val6"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val7"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), "builtins/trimprefix"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__trimsuffix(ctx *testcase) {
	var root Value
	var ps string
	var pa []string
	if p := ctx.def("p"); p == nil {
		ctx.err("p")
		return
	} else if root = p.value; root == nil {
		ctx.err("%v", tst{p})
		return
	} else if ps = __string(ctx,root); ps == "" {
		ctx.err("%v", tst{root})
		return
	} else if pa = strings.Split(ps, pathSep); len(pa) < 3 {
		ctx.err("%v", tst{root})
		return
	} else if s := "/testdata/builtins/trimsuffix"; !strings.HasSuffix(ps, s) {
		ctx.err("%v → %s", tst{root}, ps)
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
	} else if a, b, c, d := match(reversal{ctx}, p, root); sf("%v %v %v %v", a, b, c, d) != "true testdata/builtins/trimsuffix /Volumes/workspace/go/src/extbit.io/smart/ [builtins/trimsuffix]" {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
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
	} else if true {
		// skip...
	} else if a, b, c, d := match(reversal{ctx}, p, root); sf("%v %v %v %v", a, b, c, d) != "false testdata/builtins/trimprefix [testdata/builtins/trimprefix]" {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
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
	} else if a, b, c, d := match(reversal{ctx}, p, root); sf("%v %v %v %v", a, b, c, d) != sf("false <nil> %s/builtins/trimsuffix []", testdata_s) {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
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
	} else if true {
		// skip...
	} else if a, b, c, d := match(reversal{ctx}, p, root); sf("%v %v %v %v", a, b, c, d) != "false testdata/builtins/trimprefix [testdata/builtins/trimprefix]" {
		ctx.err("%v %v: %v %v %v %v", p, root, a, b, c, d)
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
	}

	s = "val2"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val3"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds+"/"; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val4"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val5"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}

	s = "val6"
    if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if s, t := v.String(), ds; s != t {
		ctx.err("%s != %s | %v", s, t, tst{v})
	}
}

func test__wildcard(ctx *testcase) {
	var m = _project(ctx)
	if x, y := m.filemap.get("**"); !y {
		ctx.err("%v", &m.filemap)
	} else if x2, y := x.get("*"); !y {
		ctx.err("%v", x)
	} else if _, y := x2.get("."); !y {
		ctx.err("%v", x2)
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
	if s, t := p.filemap.String(), `{foo:{bar:{zz:{x:{.:{h:{0:foo/bar/zz/x.h}}}},v:{?:{.:{h:{0:foo/bar/v?.h}}}}},*:{.:{h:{0:foo/*.h}}},**:{*:{.:{hh:{0:foo/**.hh}}}},?:{?:{?:{x:{.:{h:{0:foo???/x.h}}}}}}}}`; s != t {
		ctx.err("%s != %s", s, t)
	}
}

func test__wildcard2(ctx *testcase) {
	var p = _project(ctx)
	if s, t := p.filemap.String(), `{foo:{b:{*:{v:{*:{.:{h:{0:foo/b*/v*.h}}}}}},x:{*:{y:{.:{h:{0:foo/x*y.h}}}}}},f:{*?:{x:{.:{h:{0:f*?/x.h}}}}}}`; s != t {
		ctx.err("%s != %s", s, t)
	}
}

func test__wildcard3(ctx *testcase) {
	var p = _project(ctx)
	if s, t := p.filemap.String(), `{foo:{ba:{*:{v:{?:{.:{h:{0:foo/ba*/v?.h}}}}},?:{xyz:{*:{.:{txt:{0:foo/ba?/xyz*.txt}}}}}}}}`; s != t {
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
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", pat, str) {
		ctx.err("%v %v != %v %v", pat, str, m.pattern, m.value)
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err(str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", pat, str) {
		ctx.err("%v %v != %v %v", pat, str, m.pattern, m.value)
	} else if   s := "val1.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err(s)
	} else if val = unloc(val); false { // UNBOX AST WRAPPER
	} else if strVal := __string(ctx, val); strVal != str {
		ctx.err("%v: %s != %s", tst{val}, strVal, str)
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
	} else if val = unloc(val); false { // UNBOX AST WRAPPER
	} else if strVal := __string(ctx, val); strVal != str {
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
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", pat, str) {
		ctx.err("%v %v != %v %v", pat, str, m.pattern, m.value)
	} else if   s := "val2.1" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val = unloc(val); false { // UNBOX AST WRAPPER
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
	} else if val = unloc(val); false { // UNBOX AST WRAPPER
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
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", pat, str) {
		ctx.err("%v %v != %v %v", pat, str, m.pattern, m.value)
	} else if   s := "val3" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val = unloc(val); false { // UNBOX AST WRAPPER
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
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", pat, str) {
		ctx.err("%v %v != %v %v", pat, str, m.pattern, m.value)
	} else if   s := "val4" ; false {
	} else if val := ctx.val(s); val == nil {
		ctx.err("%s: %v", s, _project(ctx))
	} else if val = unloc(val); false { // UNBOX AST WRAPPER
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
	} else if val = unloc(val); false { // UNBOX AST WRAPPER
	} else if _, y := val.(*null); !y {
		ctx.err("%v", tst{val})
	} else if __string(ctx, val) != "" {
		ctx.err("%v", tst{val})
	}

	if pat, str := "**.auto", ".test/a/b/c.auto"; false {
	} else if t := unmap_files(ctx, proj, str, nil); t == nil {
		ctx.err("%s", str)
	} else if len(t) != 1 {
		ctx.err("%s: %v", str, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", pat, str) {
		ctx.err("%v %v != %v %v", pat, str, m.pattern, m.value)
	} else if s := "p1" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%s : %v", s, _project(ctx))
	} else if v = unloc(v); false { // UNBOX AST WRAPPER
	} else if x, y := v.(*path); !y {
		ctx.err("%v %v", v, tst{v})
	} else if __string(ctx, x) != str {
		ctx.err("%v %v", v, tst{v})
	} else if t := unmap_files(ctx, proj, v, nil); t == nil {
		ctx.err("%v %v", v, tst{v})
	} else if len(t) != 1 {
		ctx.err("%v %v %v", v, tst{v}, t)
	} else if m := t[0]; sf("%v %v", m.pattern, m.value) != sf("%v %v", pat, str) {
		ctx.err("%v %v != %v %v", pat, str, m.pattern, m.value)
	}

	if str := ".test/a/b/c.none" ; false {} else
	if t := unmap_files(ctx, proj, str, nil); t != nil {
		ctx.err("%v", str)
	} else if s := "p2" ; false {
	} else if v := ctx.val(s); v == nil {
		ctx.err("%v", s)
	} else if v = unloc(v); false { // UNBOX AST WRAPPER
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
	} else if f, y := unloc(v).(*file); !y || f == nil {
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

func testRules0(ctx *testcase) {
	var p = _project(ctx)

	if d := ctx.def("items"); d == nil {
		ctx.err("items")
	} else if s, t := ts(d.value), `{=list {3:10:word a} {3:12:word b} {3:14:word c}}`; s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	} else if s, t := d.value.String(), `a b c`; s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	}

	if d := ctx.def("lines"); d == nil {
		ctx.err("lines")
	} else if s, t := ts(d.value), `{=list {4:10 {=plainline {4:39:raw line-} {4:45 {4:20:word foo}}}} {4:10 {=plainline {4:39:raw line-} {4:45 {4:24:word bar}}}}}`; s != t {
		ctx.err("%v: %s != %s ; %s", d.value, s, t, ts(d.value))
	} else if s, t := d.value.String(), "{=plainline line-foo} {=plainline line-bar}"; s != t {
		ctx.err("%v: %s != %s ; %s", d.value, s, t, ts(d.value))
	} else if s, t := __string(ctx,d.value), "line-foo\nline-bar\n"; s != t {
		ctx.err("%v: %s != %s ; %s", d.value, s, t, ts(d.value))
	}

	if d := ctx.def("line"); d == nil {
		ctx.err("line")
	} else if s, t := ts(d.value), `{=list {=plainline {5:19:raw foo } {5:24:delegate {5:25:auto 1}} {5:26:raw 	}} {5:29:word bar}}`; s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	} else if s, t := d.value.String(), `{=plainline foo $1	} bar`; s != t {
		ctx.err("%v: %s != %s", d.value, s, t)
	} else if s, t := __string(ctx,d.value), "foo 	\nbar"; s != t {
		t = strings.ReplaceAll(t, "\n", `\n`)
		s = strings.ReplaceAll(s, "\n", `\n`)
		ctx.err("%v: %s != %s", d.value, t, s)
	}

	if p.entries.String() != `{-:{0:-},rule0:{0:rule0},rule1:{0:rule1},rule-x:{0:rule-x},rule-y:{0:rule-y},rule-z:{0:rule-z},rule-xx:{0:rule-xx},rule-yy:{0:rule-yy},rule-zz:{0:rule-zz},rule-xxx:{0:rule-xxx},rule-yyy:{0:rule-yyy},rule-zzz:{0:rule-zzz}}` {
		ctx.err("%v", &p.entries)
	}

	s := "rule0"
	r := p._entries(ctx, s, false)
	if r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if mr, ok := r[0].(matched_rule); !ok {
		ctx.err("%v: %v", r[0], tst{r[0]})
	} else if v := test_evoke(ctx, mr.rule, "x", "y", "z"); v == nil {
		ctx.err("%v: %v", mr.rule, tst{r[0]})
	} else if v.String() != "rule1 rule1 - xyz" {
		ctx.err("%v %v", v, tst{v})
	} else if ts(v) != `{=list {10:2 {9:24:word rule1}} {10:5 {9:24:word rule1}} {=flag {10:9}} {=compound {10:10 {1:9:word x}} {10:17 {1:9:word y}} {10:24 {1:9:word z}}}}` {
		ctx.err("%v %v", v, tst{v})
	} else if x, y := v.(*list); !y {
		ctx.err("%v", tst{v})
	} else if i := len(x.elems); i != 4 {
		ctx.err("%v", tst{x})
	} else {
		i = 0
		if z, y := unloc(x.elems[i]).(*word); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if z.s != "rule1" {
			ctx.err("%v", tst{z})
		} else if ts(x.elems[i]) != "{10:2 {9:24:word rule1}}" {
			ctx.err("%v", tst{x.elems[i]})
		}

		i = 1
		if z, y := unloc(x.elems[i]).(*word); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if z.s != "rule1" {
			ctx.err("%v", tst{z})
		} else if ts(x.elems[i]) != "{10:5 {9:24:word rule1}}" {
			ctx.err("%v", tst{x.elems[i]})
		}

		i = 2
		if z, y := unloc(x.elems[i]).(flag); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if z.Value == nil {
			ctx.err("%v", tst{z})
		} else if _, ok := z.Value.(*valbase); !ok {
			ctx.err("%v", tst{z.Value})
		} else if ts(x.elems[i]) != "{=flag {10:9}}" {
			ctx.err("%v", tst{x.elems[i]})
		}

		i = 3
		if _, y := unloc(x.elems[i]).(*compound); !y {
			ctx.err("%v", tst{x.elems[i]})
		} else if ts(x.elems[i]) != "{=compound {10:10 {1:9:word x}} {10:17 {1:9:word y}} {10:24 {1:9:word z}}}" {
			ctx.err("%v", tst{x.elems[i]})
		}
	}

	s = "rule1"
	r = p._entries(ctx, s, false)
	if r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", r)
	} else if mr, ok := r[0].(matched_rule); !ok {
		ctx.err("%v: %v", r[0], tst{r[0]})
	} else if v := test_evoke(ctx, mr.rule, bare("xxYzz")); v == nil {
		ctx.err("%v: %v", mr.rule, tst{r[0]})
	} else if s, t := v.String(), "{=plain(text) {=plainline xxYzz}}"; s != t {
		ctx.err("%v : %s != %s", v, s, t)
	} else if s, t := ts(v), "{=plain(text) {=plainline {13:2 {1:9:word xxYzz}}}}"; s != t {
		ctx.err("%v : %s != %s", v, s, t)
	}

	for _, tag := range []string{"x", "y", "z"} {
		s = "rule-"+tag
		r = p._entries(ctx, s, false)
		if r == nil {
			ctx.err("no such %s; %v", s, p.entries)
		} else if len(r) != 1 {
			ctx.err("%v", r)
		} else if recipes := r[0].programs()[0].recipes; len(recipes) != 1 {
			ctx.err("%v", recipes)
		} else if s, t := "", __string(ctx,recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline $(foreach $(ARGS),arg-$_)}", recipes[0].String(); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline {17:2:delegate {17:4:builtin foreach} {=list {17:12:delegate {1:1:auto ARGS}}} {=list {=compound {17:20:word arg} {=flag {17:24:delegate {4:46:auto _}}}}}}}", ts(recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if v := test_evoke(ctx, r[0].(matched_rule).rule, []string{"aa","bb","cc"}); v == nil {
			ctx.err("%v", r[0].(matched_rule).rule)
		} else if s, t := "{=plain(text) {=plainline arg-aa arg-bb arg-cc}}", v.String(); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline {=list {17:2 {=compound {17:20:word arg} {=flag {17:24 {17:12 {1:9:word aa}}}}}} {17:2 {=compound {17:20:word arg} {=flag {17:24 {17:12 {1:9:word bb}}}}}} {17:2 {=compound {17:20:word arg} {=flag {17:24 {17:12 {1:9:word cc}}}}}}}}}", ts(v); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "arg-aa arg-bb arg-cc\n", __string(ctx,v); s != t {
			// t = strings.ReplaceAll(t, "\n", `\n`)
			// s = strings.ReplaceAll(s, "\n", `\n`)
			ctx.err("%v : %s != %s", v, t, s)
		}
	}

	for _, tag := range []string{"xx", "yy", "zz"} {
		s = "rule-"+tag
		r = p._entries(ctx, s, false)
		if r == nil {
			ctx.err("no such %s; %v", s, p.entries)
		} else if len(r) != 1 {
			ctx.err("%v", r)
		} else if recipes := r[0].programs()[0].recipes; len(recipes) != 1 {
			ctx.err("%v", recipes)
		} else if s, t := "{=plainline $(foreach $(ARGS),{=plainline arg-$_})}", recipes[0].String(); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline {22:2:delegate {22:4:builtin foreach} {=list {22:12:delegate {1:1:auto ARGS}}} {=list {=plainline {22:31:raw arg-} {22:36:delegate {4:46:auto _}}}}}}", ts(recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if v := test_evoke(ctx, r[0].(matched_rule).rule, []string{"aa","bb","cc"}); v == nil {
			ctx.err("%v", r[0].(matched_rule).rule)
		} else if s, t := "{=plain(text) {=plainline {=plainline arg-aa} {=plainline arg-bb} {=plainline arg-cc}}}", v.String(); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline {=list {22:2 {=plainline {22:31:raw arg-} {22:36 {22:12 {1:9:word aa}}}}} {22:2 {=plainline {22:31:raw arg-} {22:36 {22:12 {1:9:word bb}}}}} {22:2 {=plainline {22:31:raw arg-} {22:36 {22:12 {1:9:word cc}}}}}}}}", ts(v); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "arg-aa\narg-bb\narg-cc\n\n", __string(ctx,v); s != t {
			t = strings.ReplaceAll(t, "\n", `\n`)
			s = strings.ReplaceAll(s, "\n", `\n`)
			ctx.err("%v : %s != %s", v, t, s)
		}
	}

	for _, tag := range []string{"xxx", "yyy", "zzz"} {
		s = "rule-"+tag
		r = p._entries(ctx, s, false)
		if r == nil {
			ctx.err("no such %s; %v", s, p.entries)
		} else if len(r) != 1 {
			ctx.err("%v", r)
		} else if recipes := r[0].programs()[0].recipes; len(recipes) != 3 {
			ctx.err("%v", recipes)
		} else if s, t := "{=plainline {=compound {27:21:word arg} {=flag {27:25 {27:12 {3:10:word a}}}}}}", ts(recipes[0]); s != t {
			ctx.err("%v : %s != %s", recipes[0], t, s)
		} else if s, t := "{=plainline {=compound {27:21:word arg} {=flag {27:25 {27:12 {3:12:word b}}}}}}", ts(recipes[1]); s != t {
			ctx.err("%v : %s != %s", recipes[1], t, s)
		} else if s, t := "{=plainline {=compound {27:21:word arg} {=flag {27:25 {27:12 {3:14:word c}}}}}}", ts(recipes[2]); s != t {
			ctx.err("%v : %s != %s", recipes[2], t, s)
		} else if v := test_evoke(ctx, r[0], bare("aa"), bare("bb"), bare("cc")); v == nil {
			ctx.err("%v", r[0])
		} else if s, t := "arg-a\narg-b\narg-c\n", __string(ctx,v); s != t {
			t = strings.ReplaceAll(t, "\n", `\n`)
			s = strings.ReplaceAll(s, "\n", `\n`)
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline arg-a} {=plainline arg-b} {=plainline arg-c}}", v.String(); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		} else if s, t := "{=plain(text) {=plainline {=compound {27:21:word arg} {=flag {27:25 {27:12 {3:10:word a}}}}}} {=plainline {=compound {27:21:word arg} {=flag {27:25 {27:12 {3:12:word b}}}}}} {=plainline {=compound {27:21:word arg} {=flag {27:25 {27:12 {3:14:word c}}}}}}}", ts(v); s != t {
			ctx.err("%v : %s != %s", v, t, s)
		}
	}
}

func testRules1(ctx *testcase) {
	var p = _project(ctx)

	if p.entries.String() != `{-:{0:-},.:{test:{.:{foobax:{0:.test.foobax},foobay:{0:.test.foobay},foobaz:{0:.test.foobaz},fxx:{0:.test.fxx}}}},':{.:{test:{.:{foo':{0:'.test.foo'}}}}}}` {
		ctx.err("%v", &p.entries)
	}

	var s string

	s = ".test.foobax"
	if r := p._entries(ctx, s, false); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{5:27 {15:7 {5:33:word fxxbar}}}" {
		ctx.err("%v", tst{v})
	}

	s = ".test.foobay"
	if r := p._entries(ctx, s, false); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{6:27 {15:7 {6:33 {=compound {6:15:punct .} {6:16:word test} {6:20:punct .} {6:21:word fxx}}}}}" {
		ctx.err("%v", tst{v})
	}

	s = ".test.foobaz"
	if r := p._entries(ctx, s, false); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{7:27 {15:7 {7:33 {=compound {7:15:punct .} {7:16:word test} {7:20:punct .} {7:21:word fxx}}}}}" {
		ctx.err("%v", tst{v})
	}

	s = ".test.1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if ts(d.value) != "{17:12 {5:27 {15:7 {5:33:word fxxbar}}}}" {
		ctx.err("%v", tst{d.value})
	} else if s := __string(ctx,d.value); s != "fxxbar" {
		ctx.err("%v -> %s", tst{d.value}, s)
	}

	s = ".test.2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if ts(d.value) != "{18:12 {6:27 {15:7 {6:33 {=compound {6:15:punct .} {6:16:word test} {6:20:punct .} {6:21:word fxx}}}}}}" {
		ctx.err("%v", tst{d.value})
	} else if s := __string(ctx,d.value); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{d.value}, s)
	}

	s = ".test.3"
	if d := ctx.def(s); d.value == nil {
		ctx.err(s)
	} else if ts(d.value) != "{19:12 {7:27 {15:7 {7:33 {=compound {7:15:punct .} {7:16:word test} {7:20:punct .} {7:21:word fxx}}}}}}" {
		ctx.err("%v", tst{d.value})
	} else if s := __string(ctx,d.value); s != ".test.fxx" {
		ctx.err("%v -> %s", tst{d.value}, s)
	}

	s = ".test.foo"
	if v := _strlit(_position(ctx), s); v == nil || v.s != s {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p._entries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	} else if r := p._entries(ctx.Context, v.s, false); r != nil {
		ctx.err("%v{%v}, %v", typeof(v), v, &p.entries)
	}

	s = "v1"
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "'.test.foo'" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if s := __string(ctx,v); s != ".test.foo" {
		ctx.err("%v{%v} -> %s", typeof(v), v, s)
	} else if _, y := v.(*strlit); !y {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p._entries(ctx.Context, v, false); r == nil {
		ctx.err("%v{%v}", typeof(v), v)
	} else if r := p._entries(ctx.Context, __string(ctx,v), false); r != nil {
		ctx.err("%v{%v}", typeof(v), v)
	}
}

type testShellForStdoutDebugStruct struct { v, s string }
func testShellForStdoutDebugHook(ctx Context, s string, v []Value, i any) {
	t := i.(*testShellForStdoutDebugStruct)
	for _, v := range v { if t.v != "" { t.v += " " }; t.v += ts(expand(ctx, v)) }
	t.s += s
}
func testShellForStdout(ctx testcase1) {
	if u := _universe(ctx); u.hooks.debug == nil {
		ctx.err("hooks.debug, %v", u.hooks)
	}

	var t = ctx.i.(*testShellForStdoutDebugStruct)

	if false && t.s != "b ab ab a" {
		ctx.err("%v", t.s)
	}
	if false && ts(t.v) != "{=[Value] {=list {=word b} {=word a}} {=list {=word b} {=word a}} {=list {=word b} {=word a}}}" {
		ctx.err("%d %v", len(t.v), tst{t.v})
	}

	t.v, t.s = "", ""

	if s := ".test.01"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{8:13 {6:2:null}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if len(t.v) != 0 {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test.02"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{9:13:delegate {=compound {9:15:punct .} {9:16:word test} {9:20:punct .} {9:21:decimal 0}} {=list {9:23:word a}} {=list {9:25:word b}}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if t.s != "b a" {
		ctx.err("%v", t.s)
	}
	if t.v != "{=list {6:10 {9:25:word b}} {6:18 {9:23:word a}}}" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test.v1"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if ts(v) != "{14:13 {12:2:null}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if t.v != "" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	p := _project(ctx)

	if s := ".test"; false {} else
	if r := p._entries(ctx, s, false); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0], bare("a"), bare("b")); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{12:2:null}" {
		ctx.err("%v", tst{v})
	} else if v := test_evoke(_final(ctx), r[0], "a", "b"); v == nil {
		ctx.err("%v", r)
	} else if ts(v) != "{12:2:null}" {
		ctx.err("%v", tst{v})
	}
	if t.s != "b ab a" {
		ctx.err("%v", t.s)
	}
	if t.v != "{=list {12:10 {1:9:word b}} {12:18 {1:9:word a}}} {=list {12:10 {1:9:word b}} {12:18 {1:9:word a}}}" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test.v2"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{15:13:delegate {=compound {15:15:punct .} {15:16:word test}} {=list {15:21:word a}} {=list {15:23:word b}}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	}
	if s := "b a"; t.s != s {
		ctx.err("%s != %s", t.s, s)
	}
	if s := "{=list {12:10 {15:23:word b}} {12:18 {15:21:word a}}}"; t.v != s {
		ctx.err("%s != %s", t.v, s)
	}

	t.v, t.s = "", ""

	if s := ".test.v3"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{16:13:delegate {=compound {16:15:punct .} {16:16:word test}} {=list {16:21:delegate {16:22:auto 1}}} {=list {16:24:delegate {16:25:auto 2}}}}" {
		ctx.err("%v", tst{v})
	} else if __string(ctx,v) != "" {
		ctx.err("%v", tst{v})
	} else if t := ctx.val(v, bare("a"), bare("b")); t == nil {
		ctx.err("%v", tst{v})
	} else if ts(t) != "{16:13 {12:2:null}}" {
		ctx.err("%v", tst{t})
	}
	if s := "b a"; t.s != s {
		ctx.err("%s != %s", t.s, s)
	}
	if s := "{=list {12:10 {16:24 {16:25:null}}} {12:18 {16:21 {16:22:null}}}} {=list {12:10 {16:24 {1:9:word b}}} {12:18 {16:21 {1:9:word a}}}}"; t.v != s {
		ctx.err("%s != %s", t.v, s)
	}

	t.v, t.s = "", ""

	if s := ".test.v4"; false {} else
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{17:13 {16:13 {12:2:null}}}" {
		ctx.err("%v", tst{v})
	}
	if t.s != "" {
		ctx.err("%v", t.s)
	}
	if t.v != "" {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test1"; false {} else
	if r := p._entries(ctx, s, false); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", tst{r[0]})
	} else if v.String() != "{=exec {=status 0}}" {
		ctx.err("%v", v)
	} else if __string(ctx,v) != "0" {
		ctx.err("%v", v)
	} else if t, y := v.(*exec_result); !y {
		ctx.err("%v", ts(v))
	} else if t.Status != 0 {
		ctx.err("%v", v)
	}
	if t.s != "1 test one\n2 test two\n" {
		ctx.err("%v", t.s)
	}
	if t.v != `{=list {12:10 {19:39 {0:0:decimal 1}}} {12:18 {19:36 {0:0:strlit 'test one\n'}}}} {=list {12:10 {19:39 {0:0:decimal 2}}} {12:18 {19:36 {0:0:strlit 'test two\n'}}}}` {
		ctx.err("%v", t.v)
	}

	t.v, t.s = "", ""

	if s := ".test2"; false {} else
	if r := p._entries(ctx, s, false); r == nil {
		ctx.err(s)
	} else if len(r) != 1 {
		ctx.err("%v", tst{r})
	} else if v := test_evoke(ctx, r[0]); v == nil {
		ctx.err("%v", r)
	} else if v.String() != "{=exec {=status 0}}" {
		ctx.err("%v", v)
	} else if __string(ctx,v) != "0" {
		ctx.err("%v", v)
	} else if t, y := v.(*exec_result); !y {
		ctx.err("%v", ts(v))
	} else if t.Status != 0 {
		ctx.err("%v", v)
	}
	if t.s != "1 test one\n2 test two\n3 test thr\n" {
		ctx.err("%v", t.s)
	}
	if t.v != `{=list {12:10 {23:39 {0:0:decimal 1}}} {12:18 {23:36 {0:0:strlit 'test one\n'}}}} {=list {12:10 {23:39 {0:0:decimal 2}}} {12:18 {23:36 {0:0:strlit 'test two\n'}}}} {=list {12:10 {23:39 {0:0:decimal 3}}} {12:18 {23:36 {0:0:strlit 'test thr\n'}}}}` {
		ctx.err("%v", t.v)
	}
}


func testConfig(ctx *testcase, proj *project, s string) (d *def, res *def) {
	for _, e := range proj.configs {
		if e.ident(ctx) == s {
			if d != nil {
				ctx.err("duplicated config: %v , %v", d, e)
			}
			d = e
		}
	}
	if o := proj.resolve(ctx, s); o != nil { res, _ = o.(*def) }
	return
}

func testConfigureDefault(ctx *testcase, spec, name string) {
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var wd = _workdir(ctx)
	var triple = "arm64-apple-Darwin23.2.0-macho"
	var outtmp, outdir, confsm, ws, s string
	var workspace, workout, rel_remnant *def

	defer func() {
		if confsm != "" { os.RemoveAll(confsm) }
		if outtmp != "" { os.RemoveAll(outtmp) }
		if outdir != "" { os.RemoveAll(outdir) }
	} ()

	if t := unmap_entries(ctx, proj, "FOO", nil); t != nil {
		ctx.err("%v", &proj.entries)
	}
	if t := unmap_entries(ctx, proj, "foo", nil); t == nil {
		ctx.err("%v", &proj.entries)
	}
	if t := unmap_entries(ctx, proj, "stamp", nil); t == nil && false {
		ctx.err("%v", &proj.entries)
	}
	if t := unmap_entries(ctx, proj, "touch", nil); t == nil && false {
		ctx.err("%v", &proj.entries)
	}

	if w := joinpath(modules_s, "configure"); proj.configure.absPath != w {
		ctx.err("%v.%v: %s != %s", proj, proj.configure, proj.configure.absPath, w)
	} else if len(proj.configure.bases) != 1 {
		ctx.err("%v", proj.configure.bases)
	} else if proj.configure.bases[0].name != "configure.base" {
		ctx.err("%v", proj.configure.bases[0])
	}

	if workspace = proj.configure.resolveDef(ctx, "workspace"); workspace == nil || workspace.value == nil {
		ctx.err("%v", tst{workspace})
	} else if p := workspace.owner(); p == nil {
		ctx.err("%v", tst{workspace})
	} else if p.name != "general" {
		ctx.err("%v ; %v", tst{workspace}, p)
	} else if ws = workspace.value.String(); !strings.HasPrefix(proj.absPath, ws) {
		ctx.err("%v", ws)
	}

	if workout = proj.configure.resolveDef(ctx, "workout"); workout == nil || workout.value == nil {
		ctx.err("%v", tst{workout})
	} else if workout.value.String() != joinpath(filepath.Dir(ws), "workout") {
		ctx.err("%v", tst{workout})
	}

	if workext := proj.configure.resolveDef(ctx, "workext"); workext == nil || workext.value == nil {
		ctx.err("%v", tst{workext})
	} else if workext.value.String() != joinpath(ws, "external") {
		ctx.err("%v", tst{workext})
	}

	if rel_chop := proj.configure.resolveDef(ctx, "rel.chop"); rel_chop == nil || rel_chop.value == nil {
		ctx.err("%v", tst{rel_chop})
	} else if rel_chop.value.String() != fmt.Sprintf("%%%%/.smart/modules/ %s/.smart/modules/ %s/.smart/ %s/", ws, ws, ws) {
		ctx.err("%v != %v", tst{rel_chop}, ws)
	} else if s := filepath.Dir(filepath.Dir(wd)); s != dirs(2, wd) {
		ctx.err("%s != %s", s, dirs(2, wd))
	} else if root := proj.resolveDef(ctx, "/"); root == nil || root.value == nil {
		ctx.err("%v", root)
	} else if root.value.String() != wd {
		ctx.err("%v : %s != %s", tst{root}, root.value, wd)
	} else if root := proj.bases[0].resolveDef(ctx, "/"); root == nil || root.value == nil {
		ctx.err("%v", root)
	} else if root.value.String() != filepath.Join(wd, ".base") {
		ctx.err("%v : %s != %s/.base", tst{root}, root.value, wd)
	} else if rel_chop := proj.resolveDef(ctx, "rel.chop"); rel_chop == nil || rel_chop.value == nil {
		ctx.err("%v", tst{rel_chop})
	} else if rel_chop.value.String() != s+"/" {
		ctx.err("%v : %s != %s/", tst{rel_chop}, rel_chop.value, s)
	}

	if rel_remnant = proj.configure.resolveDef(ctx, "rel.remnant"); rel_remnant == nil || rel_remnant.value == nil {
		ctx.err("%v", tst{rel_remnant})
	} else if t := rel_remnant.value.String(); t != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("%v != %v : %s", tst{rel_remnant}, ws, t)
	} else if t := __string(ctx, rel_remnant.value); t != "testdata/configuration" {
		ctx.err("%v != %v : %s", tst{rel_remnant}, ws, t)
	}

	if variant := proj.resolveDef(ctx, "variant"); variant == nil || variant.value == nil {
		ctx.err("%v", tst{variant})
	} else if t := ts(variant.value); t != "{=path {=word darwin} {=word arm64} {=word bootstrap}}" {
		ctx.err("%v : %s", tst{variant}, t)
	} else if t := __string(ctx, variant.value); t != "darwin/arm64/bootstrap" {
		ctx.err("%v : %s", tst{variant}, t)
	}

	if variant_tag := proj.resolveDef(ctx, "variant.tag"); variant_tag == nil || variant_tag.value == nil {
		ctx.err("%v", tst{variant_tag})
	} else if t := ts(variant_tag.value); t != "{=word bootstrap}" {
		ctx.err("%v : %s", tst{variant_tag}, t)
	} else if t := __string(ctx, variant_tag.value); t != "bootstrap" {
		ctx.err("%v : %s", tst{variant_tag}, t)
	}

	if target_arch := proj.configure.resolveDef(ctx, "target.arch"); target_arch == nil || target_arch.value == nil {
		ctx.err("%v", tst{target_arch})
	} else if t := ts(target_arch.value); t != "{=word arm64}" {
		ctx.err("%v : %s", tst{target_arch}, t)
	} else if t := __string(ctx, target_arch.value); t != "arm64" {
		ctx.err("%v : %s", tst{target_arch}, t)
	}

	if target_os := proj.configure.resolveDef(ctx, "target.os"); target_os == nil || target_os.value == nil {
		ctx.err("%v", tst{target_os})
	} else if t := ts(target_os.value); t != "{=word darwin}" {
		ctx.err("%v : %s", tst{target_os}, t)
	} else if t := __string(ctx, target_os.value); t != "darwin" {
		ctx.err("%v : %s", tst{target_os}, t)
	}

	if target_triple := proj.configure.resolveDef(ctx, "target.triple"); target_triple == nil || target_triple.value == nil {
		ctx.err("%v", tst{target_triple})
	} else if t := ts(target_triple.value); t != "{=delegate {=builtin join} {=list {=compound {=closure {=def target.arch}} {=closure {=compound {=word target} {=punct .} {=word sub}}}} {=closure {=def target.vendor}} {=closure {=def target.sys}} {=closure {=def target.abi}}} {=list {=flag {=null}}}}" {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if v := expand(_final(ctx),target_triple.value); v == nil || v == target_triple.value {
		ctx.err("%v : %s", tst{target_triple}, v)
	} else if t := v.String(); t != "arm64&(target.sub)-apple-Darwin23.2.0-macho" {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if t := ts(v); t != "{=compound {=word arm64} {=closure {=compound {=word target} {=punct .} {=word sub}}} {=flag {=null}} {=word apple} {=flag {=null}} {=raw Darwin} {=decimal 23} {=punct .} {=decimal 2} {=punct .} {=decimal 0} {=flag {=null}} {=word macho}}" {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if t := __string(ctx, target_triple.value); t != triple {
		ctx.err("%v : %s", tst{target_triple}, t)
	} else if t := __string(ctx, v); t != triple {
		ctx.err("%v : %s", tst{target_triple}, t)
	}

	if target_out := proj.configure.resolveDef(ctx, "target.out"); target_out == nil || target_out.value == nil {
		ctx.err("%v", tst{target_out})
	} else if t := target_out.value.String(); t != __string(ctx, workout)+"/&(target.triple)/&(variant.tag)" {
		ctx.err("%v : %s", tst{target_out}, t)
	} else if t := __string(ctx, target_out); t != __string(ctx, workout)+"/"+triple+"/bootstrap" {
		ctx.err("%v : %s", tst{target_out}, t)
	} else {
		outdir = t
	}

	if target_tmp := proj.configure.resolveDef(ctx, "target.tmp"); target_tmp == nil || target_tmp.value == nil {
		ctx.err("%v", tst{target_tmp})
	} else if t := target_tmp.value.String(); t != "&(target.out)/tmp" {
		ctx.err("%v : %s", tst{target_tmp}, t)
	} else if t := __string(ctx, target_tmp.value); t != __string(ctx, workout)+"/"+triple+"/bootstrap/tmp" {
		ctx.err("%v : %s", tst{target_tmp}, t)
	}

	if d := proj.configure.resolveDef(ctx, "configure.cc"); d == nil || d.value == nil {
		ctx.err("%v", tst{d})
	} else if t := d.value.String(); t != "&(cc)" {
		ctx.err("%v → %s", tst{d}, t)
	} else if t := __string(ctx, d.value); t == "" {
		ctx.err("%v → %s", tst{d}, t)
	}

	s = "outtmp"
	if x := proj.configure.resolveDef(ctx, s); x == nil {
		ctx.err("%s", s)
	} else if y := proj.resolveDef(ctx, s); y == nil {
		ctx.err("%s", s)
	} else if x != y {
		ctx.err("%v != %v", x, y)
	} else if v := x.value; v == nil {
		ctx.err("%v", tst{x})
	} else if s, t := "&(target.tmp)/&(rel.remnant)", v.String(); s != t {
		ctx.err("%v: %s != %s", tst{x}, s, t)
	} else if s := __string(ctx, x); s == "" {
		ctx.err("%v: %s", tst{x}, s)
	} else if t := __string(closure_with(ctx, proj.configure), x); t == "" {
		ctx.err("%v: %s", tst{x}, t)
	} else if s == t {
		ctx.err("%v : %s == %s", x, s, t)
	} else {
		outtmp = __string(ctx, x)

		if d, t := proj.tempdir(ctx); t != outtmp {
			ctx.err("tempdir: %s != %s (%v)", t, outtmp, d)
		}

		confsm = joinpath(outtmp, configuration_sm)
	}

	if _, y := ctx.srcs[confsm]; y {
		ctx.err("%v: already loaded configuration.sm, %v", proj, reflect.ValueOf(ctx.srcs).MapKeys())
	}

	if c := proj.tempfile(ctx, configuration_sm); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if c.fullname() != confsm {
			ctx.err("%s: %s != %s", proj.name, c, confsm)
		}
		if !c.exists() {
			ctx.err("%s: configuration not saved: %s", proj.name, c)
		}
		if f := proj.file(ctx, configuration_sm); f == nil {
			ctx.err("%s: %s", proj.name, configuration_sm)
		} else if t := f.fullname(); t != confsm {
			ctx.err("%s: %s != %s", proj.name, t, confsm)
		}
	}

	if c := proj.configuration_sm(ctx); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if c.fullname() != proj.configuration.fullname() {
			ctx.err("%s: %v != %v", proj.name, proj.configuration, c)
		}
		if s, t := c.fullname(), confsm; s != t {
			ctx.err("%v : %s != %s", proj.name, s, t)
		} else if !c.exists() {
			note(pc(ctx,s), "no such configuration.sm")
			ctx.err("%v : %s %v", proj.name, c, c.exists())
		} else if t, e := ioutil.ReadFile(s); e != nil {
			ctx.err("%v", e)
		} else if !bytes.Contains(t, []byte("configure FOO = {=self "+proj.name+"}\n")) {
			ctx.err("%s", t)
		}
	}

	// Checking coherence of configuration with proj.configure
	if d, t := proj.configure.tempdir(ctx); t != outtmp {
		ctx.err("tempdir: %s != %s (%v)", t, outtmp, d)
	}
	if c := proj.configure.tempfile(ctx, configuration_sm); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if f := proj.configure.file(ctx, configuration_sm); f == nil {
			ctx.err("%s: %s", proj.name, configuration_sm)
		} else if t := f.fullname(); t != confsm {
			ctx.err("%s: %s != %s", proj.name, t, confsm)
		}
	}

	if c := proj.configure.configuration_sm(ctx); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if pc := proj.configuration; pc != nil {
			if s, t := pc.fullname(), c.fullname(); s != t {
				ctx.err("%s: %s != %s", proj.name, s, t)
			} else if false && pc != c {
				ctx.err("%s: %v != %v : %s %s", proj.name, pc, c, s, t)
			}
		}
	}

	// Checking configure defs

	s = "FOO"
	if e, d := testConfig(ctx, proj, s); e == nil {
		ctx.err("%s", s)
	} else if d == nil {
		ctx.err("%s", s)
	} else if proj.configuration == nil {
		if d.value != nil {
			ctx.err("%s: already defined : %v", proj.name, d)
		}
	} else if proj.configuration.exists() {
		if d.value == nil {
			ctx.err("%s: not defined : %v", proj.name, d)
		}
	}

	if i, e := os.Stat(confsm); e != nil || i == nil {
		c := proj.configuration
		note(pc(ctx,c.fullname()), "%v %v", c, c.exists())
		ctx.err("%v", e)
	} else if b, e := ioutil.ReadFile(confsm); e != nil {
		ctx.err("%v", e)
	} else if !bytes.Contains(b, []byte("FOO = {=self testdefaultconfigure}\n")) {
		ctx.err("%s", b)
	}

	s = "FOO"
	if d := ctx.def(s); d == nil || d.value == nil {
		ctx.err("%s", s)
	} else if ident(ctx, d.value) != ".self" {
		ctx.err("%v", tst{d.value})
	} else if d.value.String() != "{=self testdefaultconfigure}" {
		ctx.err("%v", tst{d.value})
	} else if ts(d.value) != "{=self testdefaultconfigure}" {
		ctx.err("%v", tst{d.value})
	}

	return
	/* ctx.run */(func (c *testcase) {
		if _, y := c.srcs[confsm]; !y {
			ctx.err("%v: %v", proj, reflect.ValueOf(c.srcs).MapKeys())
		}

		var p = _project(c)
		if p.configuration == nil {
			ctx.err("%s: nil configuration", p)
		} else if s := p.configuration.fullname(); s != confsm {
			ctx.err("%s: %s != %s", p, s, confsm)
		}

		if f := p.configuration_sm(ctx); f == nil {
			c.err("%v", p)
		} else if f.fullname() != confsm {
			c.err("%s != %s", f.fullname(), confsm)
		} else if f.stat(c) == nil {
			prompt(c, "%s:1: %v : no such %s\n", confsm, p, f.name)
			c.err("%v", f)
		} else if b, e := ioutil.ReadFile(confsm); e != nil {
			c.err("%v", e)
		} else if !bytes.Contains(b, []byte("FOO = {=self "+p.name+"}\n")) {
			c.err("%s", b)
		}

		s := "FOO"
		if x, y := p.elems[s]; !y {
			c.err("%s", s)
		} else if d, y := x.(*def); !y {
			c.err("%v : %v", tst{x}, p)
		} else if d.o != defConfig {
			c.err("%v : %v : %v", d.o, d, p)
		} else if d := p.finddef("FOO"); d == nil || d != x {
			c.err("FOO: %v %v", x, d)
		} else if d.o != defConfig {
			c.err("%v : %v : %v", d.o, d, p)
		} else if v := d.value; v == nil {
			c.err("%v", d)
		} else if v.String() != "{=self "+p.name+"}" {
			c.err("%v : %v", d, tst{v})
		} else if __string(c, v) != name {
			c.err("%v : %v → %s", d, tst{v}, __string(c, v))
		} else if _, y := v.(self); !y {
			c.err("%v : %v", typeof(v), v)
		}
		if d := c.def(s); d == nil {
			c.err("%s", s)
		} else if d.o != defConfig {
			c.err("%v : %v", d.o, d)
		} else if v := d.value; v == nil {
			c.err("%v", d)
		} else if v.String() != "{=self "+p.name+"}" {
			c.err("%v : %v", d, tst{v})
		} else if __string(c, v) != name {
			c.err("%v : %v → %s", d, tst{v}, __string(c, v))
		} else if _, y := v.(self); ! y {
			c.err("%v : %v", typeof(v), v)
		}
	})(ctx)
}

func testConfigureDefault2(ctx *testcase, spec, name string) {
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var outdir, outtmp, confsm string

	defer func() {
		if confsm != "" { os.RemoveAll(confsm) }
		if outtmp != "" { os.RemoveAll(outtmp) }
		if outdir != "" { os.RemoveAll(outdir) }
	} ()

	if x := proj.resolveDef(ctx, "outtmp"); x == nil { // $//tmp
		ctx.err("%v", proj)
	} else if s, t := __string(ctx, x.value), joinpath(proj.absPath, "tmp"); s != t { // $//tmp
		ctx.err("%v : {=%v %v} : %s != %s", proj, typeof(x.value), x.value, s, t)
	} else if t := joinpath(_workdir(ctx), "tmp"); s != t { // $//tmp
		ctx.err("%v : {=%v %v} : %s != %s", proj, typeof(x.value), x.value, s, t)
	} else if p, y := x.value.(*path); !y {
		ctx.err("%v : {=%v %v}", proj, typeof(x.value), x.value)
	} else if !strings.HasSuffix(__string(ctx, p), joinpath("", spec, "tmp")) { // $//tmp
		ctx.err("%v : %v (%s)", proj, p, joinpath("", spec, "tmp"))
	} else if __string(ctx, x.value) != __string(closure_with(ctx, proj.configure), x.value) {
		ctx.err("%v : %v", proj, x.value)
	} else if o := proj.configure.resolveDef(ctx, "outtmp"); o == nil || o.value == nil { // &(target.tmp)/&(rel.remnant)
		ctx.err("%v : %v", proj, proj.configure)
	} else if __string(ctx, o) == __string(ctx, x) { // diverged (different outtmp)
		ctx.err("%v: %v == %v", proj, o, x)
	} else {
		outtmp = __string(ctx, x) // //tmp
		confsm = joinpath(outtmp, configuration_sm)
	}

	if _, y := ctx.srcs[confsm]; y {
		ctx.err("%v: already loaded configuration.sm, %v", proj, reflect.ValueOf(ctx.srcs).MapKeys())
	}

	if c := proj.configuration_sm(ctx); c == nil {
		ctx.err("%s is nil", configuration_sm)
	} else {
		if c.name != configuration_sm {
			ctx.err("%s: %s", proj.name, c.name)
		}
		if c.dir != outtmp {
			ctx.err("%s: %s != %s", proj.name, c.dir, outtmp)
		}
		if c.sub != "" {
			ctx.err("%s: %s", proj.name, c.sub)
		}
		if c.fullname() != proj.configuration.fullname() {
			ctx.err("%s: %v != %v", proj.name, proj.configuration, c)
		}
		if s, t := c.fullname(), confsm; s != t {
			ctx.err("%v : %s != %s", proj.name, s, t)
		} else if !c.exists() {
			note(pc(ctx,s), "no such configuration.sm")
			ctx.err("%v : %s %v", proj.name, c, c.exists())
		} else if t, e := ioutil.ReadFile(s); e != nil {
			ctx.err("%v", e)
		} else if !bytes.Contains(t, []byte("configure FOO = {=self "+proj.name+"}\n")) {
			ctx.err("%s", t)
		}
	}

	if joinpath(modules_s, "configure") != proj.configure.absPath {
		ctx.err("%v", proj)
	} else if o := proj.configure.resolve(ctx, "configure.cc"); o == nil {
		ctx.err("%v", proj.configure)
	} else if d, y := o.(*def); !y || d.value == nil {
		ctx.err("%v", tst{o})
	} else if d.value.String() != "&(cc)" {
		ctx.err("%v", tst{d.value})
	} else if __string(ctx, d.value) == "" {
		ctx.err("%v → %s", d.value, __string(ctx, d.value))
	} else if d := closure_finddef(ctx, "/"); d == nil {
		ctx.err("%v: &/", proj)
	} else if d.value == nil {
		ctx.err("%v: %v", proj, d)
	} else if __string(ctx, d.value) != proj.absPath {
		ctx.err("%v: %v", proj, d.value)
	} else if x := proj.resolveDef(ctx, "rel.chop"); x == nil {
		ctx.err("%v", proj)
	} else if x.value.String() == "" { // "%%/.smart/modules/ $(dir $/)/ $(dir2 $/)/ $(dir3 $/)/"
		ctx.err("%v: %v", proj, ts(x.value))
	} else if x := proj.resolveDef(ctx, "rel.remnant"); x == nil {
		ctx.err("%v", proj)
	} else if x.value.String() != "$(trim-prefix &(rel.chop),&/)" {
		ctx.err("%v: %v : %s", proj, x, ts(x.value))
	} else if t := __string(ctx, x); t == "" {
		ctx.err("%v: %v → %s", proj, x, t)
	} else if filepath.IsAbs(t) {
		ctx.err("%v: %v → %s", proj, x, t)
	} else if strings.HasPrefix(t, pathSep) {
		ctx.err("%v: %v → %s", proj, x, t)
	} else if strings.HasSuffix(t, pathSep) {
		ctx.err("%v: %v → %s", proj, x, t)
	}

	if e, d := testConfig(ctx, proj, "FOO"); e == nil {
		ctx.err("FOO")
	} else if d == nil {
		ctx.err("FOO")
	} else {
		if proj.configuration == nil {
			if d.value != nil {
				ctx.err("%s : already defined : %v", proj.name, d)
			}
		} else if proj.configuration.exists() {
			if d.value == nil {
				ctx.err("%s : not defined : %v", proj.name, d)
			}
		}
	}

	if d := ctx.def("FOO"); d == nil || d.value == nil {
		debug(ctx, "%v", d, trace{})
	} else if d.value.String() != "{=self "+proj.name+"}" {
		ctx.err("%v", d)
	} else if __string(ctx, d) != proj.name {
		ctx.err("%v ⇒ %s", d, __string(ctx, d))
	} else if ts(d.value) != "{=self "+proj.name+"}" {
		ctx.err("%v", tst{d.value})
	}
}

func testConfigureCustom(ctx *testcase) {
	var proj = _project(ctx)
	if proj.configure == nil {
		ctx.err("%v : nil configure", proj)
		return
	}

	var confsm, outtmp, outdir string

	defer func() {
		if confsm != "" { os.RemoveAll(confsm) }
		if outtmp != "" { os.RemoveAll(outtmp) }
		if outdir != "" { os.RemoveAll(outdir) }
	} ()

	if e, d := testConfig(ctx, proj, "FOO0"); e == nil {
		ctx.err("FOO0")
	} else if d == nil {
		ctx.err("FOO0")
	} else if s, t := "{=decimal 123}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO1"); e == nil {
		ctx.err("FOO1")
	} else if d == nil {
		ctx.err("FOO1")
	} else if s, t := "{=answer yes}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO2"); e == nil {
		ctx.err("FOO2")
	} else if d == nil {
		ctx.err("FOO2")
	} else if s, t := "{=boolean true}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO3"); e == nil {
		ctx.err("FOO3")
	} else if d == nil {
		ctx.err("FOO3")
	} else if s, t := "{=boolean true}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}
	if e, d := testConfig(ctx, proj, "FOO4"); e == nil {
		ctx.err("FOO4")
	} else if d == nil {
		ctx.err("FOO4")
	} else if s, t := "{=boolean true}", ts(d.value); s != t {
		ctx.err("%v : %s != %s", d, s, t)
	}

	confsm = proj.configuration.fullname()

	if s := confsm; !filepath.IsAbs(s) {
		ctx.err("%v", proj.configuration)
	} else if i, e := os.Stat(s); e != nil {
		note(pc(ctx,s), "no such configuration.sm")
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", configuration_sm)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else {
		lines := []byte(`
configure FOO0 = 123
configure FOO1 = {=yes}
configure FOO2 = {=true}
configure FOO3 = {=true}
configure FOO4 = {=true}
`)
		for i, l := range bytes.Split(lines, []byte("\n")) {
			if len(l) != 0 && bytes.Count(b, l) != 1 { ctx.err("%d. %s", i, l) }
		}
	}
}


func testDefs0(ctx *testcase) {
	if d := ctx.def("val0"); d == nil {
		ctx.err("val0")
	} else if d.o != defExpand0 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "a b c" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}

	if d := ctx.def("val1"); d == nil {
		ctx.err("val1")
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "x a b" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}

	if d := ctx.def("val2"); d == nil {
		ctx.err("val2")
	} else if d.o != defExpand2 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "x a b" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}

	if d := ctx.def("val3"); d == nil {
		ctx.err("val3")
	} else if d.o != defExpand2 {
		ctx.err("%v %v", d, d.o)
	} else if val := d.value; val == nil {
		ctx.err("%v", d)
	} else if s := __string(ctx, val); s != "a b c" {
		ctx.err("%v", val)
	} else if _, y := val.(*list); !y {
		ctx.err("%v", val)
	}
}


func testBug_01(ctx *testcase) {
	s := "okay"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else {
		var v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = expand(_final(ctx), d.value)
		} ()
		if v == nil {
			ctx.err("%v", d)
		}

		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = ctx.val(d, defExpand1, "a", "b", "c", "d")
		} ()
		if v == nil {
			ctx.err("%v", d)
		} else if s, t := ts(v), "{=list {=compound {=word x} {=punct .} {=word a}} {=compound {=word x} {=punct .} {=word b}} {=compound {=word y} {=punct .} {=word a}} {=compound {=word y} {=punct .} {=word b}} {=compound {=word z} {=punct .} {=word a}} {=compound {=word z} {=punct .} {=word b}}}"; s != t {
			note(ctx, "%s", s)
			note(ctx, "%s", t)
			ctx.err("%s", v)
		}
	}

	s = "bug_0"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(bug_0.1 $1,$2)"; s != t {
		ctx.err("%v : %s != %s", d, s, t)
	} else {
		var v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = expand(trace_evoke_loop{_final(ctx)},d.value)
		} ()
		if v == nil {
			ctx.err("%v", d)
		}

		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y {
					debug(ctx, "%s", __string(ctx, x.Value), trace{})
				}
			} ()
			v = ctx.val(d, defExpand1, "a", "b", "c", "d")
		} ()
		if v == nil {
			ctx.err("%v", d)
		} else if s, t := ts(v), "{=list {=closure {=compound {=word x} {=punct .} {=word a}}} {=closure {=compound {=word x} {=punct .} {=word b}}} {=closure {=compound {=word y} {=punct .} {=word a}}} {=closure {=compound {=word y} {=punct .} {=word b}}} {=closure {=compound {=word z} {=punct .} {=word a}}} {=closure {=compound {=word z} {=punct .} {=word b}}}}"; s != t {
			note(ctx, "%s", s)
			note(ctx, "%s", t)
			ctx.err("%s", v)
		}
	}

	s = "bug_1"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s, t := v.String(), "$(bug_1.1 $1,$2)"; s != t {
		ctx.err("%v : %s != %s", d, s, t)
	} else {
		var e, v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			v = expand(trace_evoke_loop{_final(ctx)},d.value)
		} ()
		if s, t := ts(e), "{=def bug_1.1}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}
	}

	s = "flags"
	d = ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if s := ts(v); s != "{=delegate {=def .flags} {=list {=delegate {=auto 1}}} {=list {=delegate {=auto 2}}}}" {
		ctx.err("%v : %v", d, s)
	} else {
		var e, v Value
		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			v = expand(trace_evoke_loop{_final(ctx)},d.value)
		} ()
		if s, t := ts(e), "{=def .flags}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}

		func () {
			defer func () {
				if x, y := recover().(trace_evoke_loop_err); y { e = x.Value }
			} ()
			a, _ := va(ctx, []string{"a", "b", "c", "d"}).(*list)
			v = evoke(trace_evoke_loop{_final(ctx)}, d, nil, a.elems)
		} ()
		if s, t := ts(e), "{=def .flags}"; s != t {
			ctx.err("expecting evocation loop: %s != %s", s, t)
		}
		if v != nil {
			ctx.err("%v → %v", d, v)
		}
	}
}


var testValidFlag = []*regexp.Regexp{
	regexp.MustCompile(`^-(?:-target|triple)=[[:alnum:]\.\-_]+$`),
	regexp.MustCompile(`^-(?:shared|static|ObjC(?:\+\+)?)$`),
	regexp.MustCompile(`^-(?:std|Werror)=[[:alnum:]_\-]+$`),
	regexp.MustCompile(`^-(?:(?:(?:cxx|stdlib\+\+)-)?isystem(?:-after)?)=?[[:alnum:]\.\-/_]+$`),
	regexp.MustCompile(`^-Wl,-platform_version,(?:MacOSX|iPhone(?:Simulator)?|AppleTV|Watch(?:Simulator)?|DriverKit),[[:digit:].]+,[[:digit:].]+$`),
	regexp.MustCompile(`^-Wl,(?:-v|-demangle|-rpath,"[^"]+")$`),
	regexp.MustCompile(`^-[IL]=?[[:alnum:]\.\-/_]+$`),
	regexp.MustCompile(`^-[DWfl][[:alnum:]\.\-_]+$`),
	regexp.MustCompile(`^-m(?:arch|linker-version=[[:digit:]]+)$`),
	regexp.MustCompile(`^-no[[:alnum:]\.\-\+_]+$`),
	// regexp.MustCompile(`^-X(?:clang)$`),
	regexp.MustCompile(`^-x(?:c(?:\+\+)?)$`),
	regexp.MustCompile(`^-O[0-6]$`),
	regexp.MustCompile(`^-l[^/]+$`),
	regexp.MustCompile(`^-[cvg]$`),
	regexp.MustCompile(`^-(?:-|i(?:(?:framework)?with)?)sysroot=[[:alnum:]\.\-/_]+$`),
	regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+\.(?:[coh])$`), // /a/b/c/foo.c
}

var testValidFlagVal = map[*regexp.Regexp][]*regexp.Regexp{
	regexp.MustCompile(`^-(?:std|Werror)$`): []*regexp.Regexp{
		regexp.MustCompile(`^[[:alnum:]\.\-_]+$`),
	},
	regexp.MustCompile(`^-(?:-target|triple)$`): []*regexp.Regexp{
		regexp.MustCompile(`^[[:alnum:]\.\-_]+$`),
	},
	regexp.MustCompile(`^-D$`): []*regexp.Regexp{
		regexp.MustCompile(`^[[:alnum:]\.\-_]+$`),
	},
	regexp.MustCompile(`^-([I]|include)$`): []*regexp.Regexp{
		regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+\.(?:[ch](?:xx|pp)|inc)$`), // /a/b/c/foo.c
	},
	regexp.MustCompile(`^-o$`): []*regexp.Regexp{
		regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+\.(?:[ox]|out|exe)$`), // /a/b/c/foo.c
	},
	regexp.MustCompile(`^-framework$`): []*regexp.Regexp{
		regexp.MustCompile(`^[^-].+$`),
	},
	regexp.MustCompile(`^-(L|(?:-|i(?:(?:framework)?with)?)sysroot)$`): []*regexp.Regexp{
		regexp.MustCompile(`^(?:/(?:[^/]*/)+)?.+$`),
	},
	regexp.MustCompile(`^-l$`): []*regexp.Regexp{
		regexp.MustCompile(`^[^/]+$`),
	},
	regexp.MustCompile(`^-X(?:clang)$`): []*regexp.Regexp{
		regexp.MustCompile(`^-triple=[^/]+$`),
	},
}

var testInvalidArg = []*regexp.Regexp{
	regexp.MustCompile(`<[^>]*>`),
}

var testWrongExecOutput = []*regexp.Regexp{
	regexp.MustCompile(`^#error (Unsupported architecture|architecture not supported)`),
	regexp.MustCompile(`^clang: error: unknown argument '[^']+'; did you mean '[^']+'\?`),
	regexp.MustCompile(`^ld: (missing OS version in target triple '([^']+)') in '([^']+)'`),
	regexp.MustCompile(`^ld: (more than three dashes in target triple '([^']+)') in '([^']+)'`),
	regexp.MustCompile(`^ld: (unknown OS in target triple '([^']+)') in '([^']+)'`),
	regexp.MustCompile(`^ld: (unknown file type) in '([^']+)'`),
	regexp.MustCompile(`^ld: (unknown options: (.+))`),
	regexp.MustCompile(`^ld: (Missing -platform_version option)`),
	regexp.MustCompile(`^ld: (-platform_version unknown platform: (.+))`),
	regexp.MustCompile(`^(ld: library '[^=]+=.+?' not found)`), // ld: library 'NAME=bsd' not found

	// Errors caused by wrong #include, example: #include TARGET=HAVE_LIBIEEE
	regexp.MustCompile(`^.+:[[:digit:]]+:[[:digit:]]+: (error: expected "FILENAME" or <FILENAME>)`),
	regexp.MustCompile(`^(#include [^=]+=.+)`),
}
type testSuspicious struct {
	rx *regexp.Regexp
	ignore map[string]struct{}
	i, k int // info, key
}
var testSuspiciousExecOutput = []*testSuspicious{
	&testSuspicious{
		regexp.MustCompile(`^clang:(?: warning:)? (argument unused during compilation: '[^']+' \[[^\]]+\])`),
		map[string]struct{}{}, 1, 0,
	},
	&testSuspicious{
		regexp.MustCompile(`^ld:(?: warning:)? (search path '([^']+)' not found)`),
		map[string]struct{}{}, 1, 2,
	},
	&testSuspicious{
		regexp.MustCompile(`TODO: ^ld:(?: warning:)? (ignoring duplicate libraries: '([^']+)')`),
		map[string]struct{}{}, 1, 2,
	},
	&testSuspicious{
		regexp.MustCompile(`^(ignoring nonexistent directory "([^"]+)")`),
		map[string]struct{}{
			`/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/Library/Frameworks`: struct{}{},
			`/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/usr/local/include`: struct{}{},
			`/usr/local/include`: struct{}{},
			`/usr/include`: struct{}{},
		}, 1, 2,
	},
}

func validFlag(s string) (res bool, vxs []*regexp.Regexp) {
	for _, x := range testValidFlag { if res = x.MatchString(s); res { return }}
	for x, v := range testValidFlagVal { if x.MatchString(s) { return true, v }}
	return
}

func validFlags(ctx *testcase, v Value, s string) (res bool) {
	var rxs []*regexp.Regexp
	var fields = strings.Fields(s)
	for i := 0; i < len(fields); i += 1 {
		flag := fields[i]

		if res, rxs = validFlag(flag); !res {
			ctx.err("invalid flag: %s ; %v{%v}", flag, typeof(v), v) ; break
		} else if len(rxs) == 0 {
			continue
		}

		if i += 1; i == len(fields) {
			ctx.err("wrong flag: %s ; %v{%v}", flag, typeof(v), v)
			return
		}

		val := fields[i]

		for _, rx := range rxs {
			var ( s = val ; n = 0 )
			for ; n < strings.Count(rx.String(), "[ ]"); n += 1 {
				if i+n+1 < len(fields) { s += " " + fields[i+n+1] } else { break }
			}

			if !rx.MatchString(s) {
				ctx.err("wrong flag: %s %s, %v ; %v{%v}", flag, s, rx, typeof(v), v)
			} else {
				if false { debug(ctx, "%v: %v ; %v", flag, s, rx) }
				i += n
				break
			}
		}
	}
	return
}

var testValidateClang = regexp.MustCompile(`^@?(?:/(?:[^/]*/)+)?(clang(?:\+{2})?)[[:space:]]+`)
var testValidateOther = regexp.MustCompile(`^@?(?:/(?:[^/]*/)+)?(echo)[[:space:]]+`)
var testValidateOutFilename = regexp.MustCompile(`\.configure/[^/=]+?/[^/=]+$`)

func testValidateExecRecipe(tc *testcase, ctx Context, source string, recipe Value) {
	if source == "" || recipe == nil { return }

	if m := testValidateClang.FindStringSubmatch(source); m != nil {
		if !validFlags(tc, recipe, source[len(m[0]):]) {
			debug(ctx, "validate: %v; %v", m, source)
		}
	} else if m := testValidateOther.FindStringSubmatch(source); m != nil {
		// okay
	} else {
		debug(ctx, "TODO: validate: %v", source)
	}
}

func testValidateExecOutput(tc *testcase, ctx Context, line string, l int) {
	if s := _position(ctx).Filename; !testValidateOutFilename.MatchString(s) {
		debug(ctx, "bad out-file: %v", s, trace{})
	}
	for _, rx := range testWrongExecOutput {
		if m := rx.FindStringSubmatch(line); len(m) > 0 {
			if len(m) < 2 {
				debug(ctx, "%v", m[0], trace{})
			} else {
				debug(ctx, "%v", m[1], trace{})
			}
		}
	}
	for _, t := range testSuspiciousExecOutput {
		if m := t.rx.FindStringSubmatch(line); len(m) > 0 {
			if _, y := t.ignore[m[t.k]]; !y {
				debug(ctx, "%v", m[t.i], trace{})
			}
		}
	}
}

func testVariantTargetVars(ctx *testcase) {
	testVariantTargetVars1(ctx)
	if v := ctx.val("target.os"); v == nil {
		ctx.err("%v: target.os is nil", _project(ctx))
	}
}
func testVariantTargetVars1(ctx *testcase) {
	var p = _project(ctx)
	var workspace, modules string

	if d := p.resolveDef(ctx, "workspace"); d == nil {
		ctx.err("%v: workspace is nil", p)
	} else if workspace = __string(ctx, d); workspace == "" {
		ctx.err("%v: %v", p, d)
	} else if !filepath.IsAbs(workspace) {
		ctx.err("%v: %v", p, workspace)
	} else {
		modules = filepath.Join(workspace, ".smart", "modules")
	}

	for _, s := range []string {
		"variant/.target/.base/do.smart",
		"variant/.target/do.smart",
		"variant/.target/darwin/do.smart",
		"variant/.target/darwin/arm64/do.smart",
		"variant/bootstrap",
		"variant/do.smart",
	}{
		var t = filepath.Join(modules, s)
		if _, y := ctx.chks[t]; !y {
			ctx.err("%v: %v", p, s)
		}
	}

	if v := p.resolve(ctx, "variant"); v == nil {
		ctx.err("%v: variant is nil", p)
	} else if d, y := v.(*def); !y {
		ctx.err("%v: variant is not def: %v (%v)", p, v, typeof(v))
	} else if d.value == nil {
		ctx.err("%v: variant value is nil: %v", p, d)
	} else if s := __string(ctx, d.value); s != "darwin/arm64/bootstrap" {
		ctx.err("%v: variant: %v", p, d.value)
	}

	if v := ctx.val("target.arch"); v == nil {
		ctx.err("%v: target.arch is nil", p)
	} else if __string(ctx, v) != "arm64" {
		ctx.err("%v: target.arch: %v", p, v)
	}

	if v := ctx.val("target.abi"); v == nil {
		ctx.err("%v: target.abi is nil", p)
	} else if __string(ctx, v) != "macho" {
		ctx.err("%v: target.abi: %v", p, v)
	}

	if v := ctx.val("target.vendor"); v == nil {
		ctx.err("%v: target.vendor is nil", p)
	} else if __string(ctx, v) != "apple" {
		ctx.err("%v: target.vendor: %v", p, v)
	}

	if v := ctx.val("target.release"); v == nil {
		ctx.err("%v: target.release is nil", p)
	} else if !regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`).MatchString(__string(ctx, v)) {
		ctx.err("%v: target.release: %v", p, v)
	}

	if v := ctx.val("target.sys"); v == nil {
		ctx.err("%v: target.sys is nil", p)
	} else if !regexp.MustCompile(`Darwin[0-9]+\.[0-9]+\.[0-9]+`).MatchString(__string(ctx, v)) {
		ctx.err("%v: target.sys: %v", p, v)
	}

	if v := ctx.val("target.triple"); v == nil {
		ctx.err("%v: target.triple is nil", p)
	} else if !regexp.MustCompile(`arm64-apple-Darwin[0-9]+\.[0-9]+\.[0-9]+-macho`).MatchString(__string(ctx, v)) {
		ctx.err("%v: target.triple: %v", p, v)
	}
}

func testVariantTarget(ctx *testcase) {
	testVariantTargetVars1(ctx)

	var p = _project(ctx)

	for k, v := range langs_map {
		var s = "lang."+k
		if d := ctx.def(s); d == nil {
			ctx.err("%v : %v", p, s)//, p.names()
		} else if d.value == nil {
			ctx.err("%v", d)
		} else if d.value.String() != v {
			ctx.err("%v", d)
		}
	}

	if d := ctx.def("host.triple"); d == nil {
		ctx.err("host.triple")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, d)
	} else if s, t := d.value.String(), "$(join &(host.arch)&(host.sub) &(host.vendor) &(host.sys) &(host.abi),-)"; s != t {
		ctx.err("%v : %s != %s", tst{d}, s, t)
	} else if s := __string(ctx, d.value); strings.Count(s, "-") > 3 {
		ctx.err("more than three dashes: %v: %v", d.value, s)
	}

	if d := ctx.def("target.os"); d == nil {
		ctx.err("%v: target.os is nil", p)
	} else if __string(ctx, d) != "foo" {
		ctx.err("%v: target.os: %v", p, d)
	}

	if d := ctx.def("target.triple"); d == nil {
		ctx.err("target.triple")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, d)
	} else if s, t := d.value.String(), "$(join &(target.arch)&(target.sub) &(target.vendor) &(target.sys) &(target.abi),-)"; s != t {
		ctx.err("%v : %s != %s", tst{d}, s, t)
	} else if s := __string(ctx, d.value); strings.Count(s, "-") > 3 {
		ctx.err("more than three dashes: %v: %v", d.value, s)
	}

	var usev, uses string
	if d := ctx.def("use.*"); d == nil {
		ctx.err("use.*")
	} else if d.o != defExpand1 {
		ctx.err("%v %v", d.o, d)
	}
	if v := ctx.val("use.*"); v == nil {
		ctx.err("use.*")
	} else if usev = v.String(); usev == "" {
		ctx.err("%v{%v}", typeof(v), v)
	} else if uses = __string(ctx, v); uses == "" {
		ctx.err("%v{%v} → %s", typeof(v), v, uses)
	}

	usev = " "+usev
	uses = " "+uses

	for _, flag := range []string{
		"-D", "-I", "-L", "-O", "-W", "-Wl", "-Werror", "-Wno-error",
		"-f", "-f.ld", "-m", "-g", "-v", "-no", "-no.ld",
		"--sysroot", "-isysroot", "-iwithsysroot", "-iframeworkwithsysroot", "-iframework",
		"-isystem", "-isystem-after", "-cxx-isystem", "-stdlib", "-stdlib++-isystem",
		"-diagnostics", "-platform_version",
	}{
		s1 := fmt.Sprintf("%s(-unique)", flag)
		s2 := fmt.Sprintf("&(target.os)~%s(-unique)", flag)
		s3 := fmt.Sprintf("foo~%s(-unique)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v ; %v", s1, usev) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v ; %v", s2, usev) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v ; %v", s3, uses) }
		if d := ctx.def(flag); d == nil {
			ctx.err("%s", flag)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}

		for _, lang := range langs_map {
			s0 := fmt.Sprintf("%s.%s", flag, lang)
			s1 := fmt.Sprintf("%s.%s(-unique)", flag, lang)
			s2 := fmt.Sprintf("&(target.os)~%s.%s(-unique)", flag, lang)
			s3 := fmt.Sprintf("foo~%s.%s(-unique)", flag, lang)
			if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
			if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
			if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
			if d := ctx.def(s0); d == nil {
				ctx.err("%s", s0)
			} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
				ctx.err("%v %v", d.o, d)
			}
		}
	}
	for _, flag := range []string{"-l","-framework"} {
		s1 := fmt.Sprintf("%s(-unique -reverse)", flag)
		s2 := fmt.Sprintf("&(target.os)~%s(-unique -reverse)", flag)
		s3 := fmt.Sprintf("foo~%s(-unique -reverse)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(flag); d == nil {
			ctx.err("%s", flag)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}
	}
	for _, flag := range []string{"ar","asm","c","cpp","cxx","oc","ocxx","cl","cuda","cudaxx","ld"} {
		s0 := fmt.Sprintf("%sflags", flag)
		s1 := fmt.Sprintf("%sflags(-unique -auto)", flag)
		s2 := fmt.Sprintf("&(target.os)~%sflags(-unique -auto)", flag)
		s3 := fmt.Sprintf("foo~%sflags(-unique -auto)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(s0); d == nil {
			ctx.err("%s", s0)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}
	}
	for _, flag := range []string{"ld"} {
		for _, suffix := range strings.Fields("shared program") {
			s0 := fmt.Sprintf("%sflags.%s", flag, suffix)
			s1 := fmt.Sprintf("%sflags.%s(-unique -auto)", flag, suffix)
			s2 := fmt.Sprintf("&(target.os)~%sflags.%s(-unique -auto)", flag, suffix)
			s3 := fmt.Sprintf("foo~%sflags.%s(-unique -auto)", flag, suffix)
			if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
			if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
			if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
			if d := ctx.def(s0); d == nil {
				ctx.err("%s", s0)
			} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
				ctx.err("%v %v", d.o, d)
			}
		}
	}
	for _, flag := range []string{"ld.framework","ldlibs","loadlibs","loadlibes"} {
		s0 := fmt.Sprintf("%s", flag)
		s1 := fmt.Sprintf("%s(-unique -auto -reverse)", flag)
		s2 := fmt.Sprintf("&(target.os)~%s(-unique -auto -reverse)", flag)
		s3 := fmt.Sprintf("foo~%s(-unique -auto -reverse)", flag)
		if strings.Count(usev, " "+s1) != 1 { ctx.err("%v", s1) }
		if strings.Count(usev, " "+s2) != 1 { ctx.err("%v", s2) }
		if strings.Count(uses, " "+s3) != 1 { ctx.err("%v", s3) }
		if d := ctx.def(s0); d == nil {
			ctx.err("%s", s0)
		} else if d.o != defVoid && d.o != defExpand1 && d.o != defExpand2 {
			ctx.err("%v %v", d.o, d)
		}
	}

	s := "neg1"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=negative {=word foobar}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "!foobar" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s != "!foobar" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else if x, y := v.(negative); !y {
		ctx.err("%v", tst{v})
	} else if t1, t2 := __true(ctx, x), __true(ctx, x.Value); t1 != !t2 {
		ctx.err("%v : %v != !%v", tst{v}, t1, t2)
	}

	s = "neg2"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=compound {=word a} {=negative {=word foobar}}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "a!foobar" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s != "a!foobar" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}

	s = "neg3"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=closure {=compound {=word a} {=negative {=word foobar}}}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s); v == nil {
		ctx.err(s)
	} else if v.String() != "&(a!foobar)" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s != "xxx" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}

	s = "neg4"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if ts(d.value) != "{=delegate {=builtin foreach} {=list {=delegate {=auto 1}}} {=list {=closure {=compound {=word a} {=negative {=delegate {=auto _}}}}}}}" {
		ctx.err("%v", tst{d.value})
	}
	if v := ctx.val(s, "xxx"); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=cond {=closure {=compound {=word a} {=negative {=word xxx}}}}}" {
		ctx.err("%v", tst{v})
	} else if s := v.String(); s != "&(a!xxx)?" {
		ctx.err("%v : %s", tst{v}, s)
	} else if s := __string(ctx, v); s != "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}
	if v := ctx.val(s, "foobar"); v == nil {
		ctx.err(s)
	} else if ts(v) != "{=cond {=closure {=compound {=word a} {=negative {=word foobar}}}}}" {
		ctx.err("%v", tst{v})
	} else if s := v.String(); s != "&(a!foobar)?" {
		ctx.err("%v : %s", tst{v}, s)
	} else if s := __string(ctx, v); s != "xxx" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}

	s = "cflags"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=def .flags} {=list {=delegate {=auto 1}}} {=list {=word c}}}" {
		ctx.err("%v", tst{v})
	} else {
		var t = expand(_final(ctx),v)

		var str1 = t.String()
		for _, s := range []string{
			"&(-v.{$1})? "+ctx.vs("-v.c"),
			"-std=&(-std.{$1})? -std=&(std.{$1})? -std=&(-std.c)? -std="+ctx.vs("std.c"),
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(t)
		for _, s := range []string{
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}
	}
	if v := ctx.val(s, "z"); v == nil {
		ctx.err(s)
	} else {
		var str1 = v.String()
		for _, s := range []string{
			"&(-v.z)? &(-v.c)?",
			"-std=&(-std.z)? -std=&(std.z)? -std=&(-std.c)? -std=&(std.c)?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c}}}}}",
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}

		var str3 = expand(_final(ctx),v).String()
		for _, s := range []string{
			"&(-v.z)? "+ctx.vs("-v.c"),
			"-std=&(-std.z)? -std=&(std.z)? -std=&(-std.c)? -std="+ctx.vs("std.c"),
		}{
			if strings.Count(str3, s) != 1 { ctx.err("%s : %s", s, str3) }
		}

		var str4 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c}}}}}",
		}{
			if strings.Count(str4, s) != 1 { ctx.err("%s : %s", s, str4) }
		}
	}

	s = "cxxflags"
	if d := ctx.def(s); d == nil {
		ctx.err(s)
	} else if v := d.value; v == nil {
		ctx.err("%v", d)
	} else if ts(v) != "{=delegate {=def .flags+} {=list {=delegate {=auto 1}}} {=list {=word c++}} {=list {=word cxx}}}" {
		ctx.err("%v", tst{v})
	} else {
		var t = expand(_final(ctx),v)

		var str1 = t.String()
		for _, s := range []string{
			"&(-v.{$1})? "+ctx.vs("-v.c++"),
			"-std=&(-std.{$1})? -std=&(std.{$1})? -std=&(-std.c++)? -std="+ctx.vs("std.c++"),
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(t)
		for _, s := range []string{
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}
	}
	if v := ctx.val(s, "z++"); v == nil {
		ctx.err(s)
	} else {
		var str1 = v.String()
		for _, s := range []string{
			"&(-v.z++)? &(-v.c++)?",
			"-std=&(-std.z++)? -std=&(std.z++)? -std=&(-std.c++)? -std=&(std.c++)?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z++}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c++}}}}}",
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}

		var str3 = expand(_final(ctx),v).String()
		for _, s := range []string{
			"&(-v.z++)? "+ctx.vs("-v.c++"),
			"-std=&(-std.z++)? -std=&(std.z++)? -std=&(-std.c++)? -std="+ctx.vs("std.c++"),
		}{
			if strings.Count(str3, s) != 1 { ctx.err("%s : %s", s, str3) }
		}

		var str4 = ts(v)
		for _, s := range []string{
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word z++}}}}}",
			"{=cond {=closure {=flag {=compound {=word v} {=punct .} {=word c++}}}}}",
		}{
			if strings.Count(str4, s) != 1 { ctx.err("%s : %s", s, str4) }
		}
	}
}

func testApp(ctx *testcase) {
	testVariantTargetVars(ctx)

	var p = _project(ctx)

	if p.configure != nil {
		ctx.err("%v: nil configure", p)
	}

	// cc, cancel := context.WithTimeout(context.Background(), 2400*time.Second)
	// defer cancel()

	s := ".flag"
	d := ctx.def(s)
	if d == nil {
		ctx.err(s)
	} else if v := d.value ; v == nil {
		ctx.err("%v", d)
	} else {
		var t = expand(_final(ctx),v)

		var str1 = t.String()
		for _, s := range []string {
			"{$(filter-out $(foreach $1,&($2!$_) &(darwin~$2!$_)),&($2) &(darwin~$2))}?",
			"{$(foreach $1,&($2.$_) &(darwin~$2.$_))}?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(t)
		for _, s := range []string {
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}
	}
	if v := ctx.val(s, []string{"a", "b", "c"}, "-x"); v == nil {
		ctx.err(s)
	} else {
		var str1 = v.String()
		for _, s := range []string{
			"$(or $3,-x){$(filter-out &(-x!a)? &(&(target.os)~-x!a)? &(-x!b)? &(&(target.os)~-x!b)? &(-x!c)? &(&(target.os)~-x!c)?,&(-x) &(&(target.os)~-x))}$(or $4)?",
			"$(or $3,-x){&(-x.a)}$(or $4)?",
			"$(or $3,-x){&(&(target.os)~-x.a)}$(or $4)?",
			"$(or $3,-x){&(-x.b)}$(or $4)?",
			"$(or $3,-x){&(&(target.os)~-x.b)}$(or $4)?",
			"$(or $3,-x){&(-x.c)}$(or $4)?",
			"$(or $3,-x){&(&(target.os)~-x.c)}$(or $4)?",
		}{
			if strings.Count(str1, s) != 1 { ctx.err("%s : %s", s, str1) }
		}

		var str2 = ts(v)
		for _, s := range []string{
		}{
			if strings.Count(str2, s) != 1 { ctx.err("%s : %s", s, str2) }
		}

		var str3 = expand(_final(ctx),v).String()
		for _, s := range []string{
		}{
			if strings.Count(str3, s) != 1 { ctx.err("%s : %s", s, str3) }
		}

		var str4 = ts(v)
		for _, s := range []string{
		}{
			if strings.Count(str4, s) != 1 { ctx.err("%s : %s", s, str4) }
		}
	}

	// select {
	// case <-cc.Done():
	// }
}

func _testApp(ctx *testcase) {
	if _project(ctx).configure != nil {
		ctx.err("%v: nil configure", _project(ctx))
	}

	ss := func(s string) string { os := "darwin"
		return strings.Replace(s, "<OS>", os, -1)
	}

	flag1 := func(a ...any) string { // $(.flag $1)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out $(foreach $1,&(%[1]s!$_) &(&(target.os)~%[1]s!$_)),&(%[1]s) &(&(target.os)~%[1]s)) $(foreach $1,&(%[1]s.$_) &(&(target.os)~%[1]s.$_)),$(or $3,%[1]s)$_$(or $4))", a...)
	}
	flag2 := func(a ...any) string { if len(a) > 1 { a[0], a[1] = a[1], a[0] } // $(.flag $1 yyy,xxx)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out $(foreach $1 %[1]s,&(%[2]s!$_) &(&(target.os)~%[2]s!$_)),&(%[2]s) &(&(target.os)~%[2]s)) $(foreach $1 %[1]s,&(%[2]s.$_) &(&(target.os)~%[2]s.$_)),%[2]s$_$(or $4))", a...)
	}
	flag3 := func(a ...any) string { // $(.flag $1,xxx,yy)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out &(%[1]s!%[2]s) &(&(target.os)~%[1]s!%[2]s),&(%[1]s) &(&(target.os)~%[1]s)) &(%[1]s.%[2]s) &(&(target.os)~%[1]s.%[2]s),$(or $3,%[1]s)$_$(or $4))", a...)
	}
	flag4 := func(a ...any) string { // $(.flag $1,xxx,y,y)
		return fmt.Sprintf("$(foreach(-unique) $(filter-out &(%[1]s!%[2]s) &(&(target.os)~%[1]s!%[2]s) &(%[1]s!c) &(&(target.os)~%[1]s!c),&(%[1]s) &(&(target.os)~%[1]s)) &(%[1]s.%[2]s) &(&(target.os)~%[1]s.%[2]s) &(%[1]s.%[3]s) &(&(target.os)~%[1]s.%[3]s),%[1]s$_$(or $4))", a...)
	}

	var foo1 = strings.Fields(ss(`cppflags-foo cppflags~foo~<OS>
-std=foostd -ffooF -IfooI -DfooD
-isystemfooisystem -isystem-afterfooisystem-after`))
	var foo2 = strings.Fields(ss(`cxxflags-foo cxxflags~foo~<OS>
-std=foostd -ffooF -IfooI -DfooD -gfooG -OfooO
-isystemfooisystem -isystem-afterfooisystem-after
-cxx-isystemfoocxxisystem -stdlib++-isystemfoostdlib++isystem`))
	var foo3 = strings.Fields(ss(`ldflags-foo ldflags~foo~<OS> -ffooF -OfooO -LfooL
-Wl,fooWl -Wl,-rpath,"foorpath"`))
	var foo4 = strings.Fields(ss(`ldlibs-foo ldlibs~foo~<OS> -lfool`))
	var foo5 = strings.Fields(ss(`loadlibes-foo loadlibes~foo~<OS>`))
	var foo6 = strings.Fields(ss(`loadlibs-foo loadlibs~foo~<OS>`))

	if d := ctx.def(".flag"); d == nil {
		ctx.err(".flag")
	} else if d.value == nil {
		ctx.err("%v", d)
	} else if s := d.value.String(); s == "" {
		ctx.err("%v", d)
	} else if strings.Count(s, "$(foreach(-unique) ") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, "$(filter-out $(foreach $1,&($2!$_) &(&(target.os)~$2!$_)),&($2) &(&(target.os)~$2))") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, "$(foreach $1,&($2.$_) &(&(target.os)~$2.$_)),") != 1 {
		ctx.err("%v", d)
	} else if strings.Count(s, ",$(or $3,$2)$_$(or $4))") != 1 {
		ctx.err("%v", d)
	} else if s != flag1("$2") {
		ctx.err("%v ; %v", s, d)
	}
	if v := ctx.val(".flag", []string{"a", "b", "c"}, "-x"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!a) &(&(target.os)~-x!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!b) &(&(target.os)~-x!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!c) &(&(target.os)~-x!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-x) &(&(target.os)~-x)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.a) &(&(target.os)~-x.a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.b) &(&(target.os)~-x.b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.c) &(&(target.os)~-x.c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-x!a) &(&(target.os)~-x!a) &(-x!b) &(&(target.os)~-x!b) &(-x!c) &(&(target.os)~-x!c),&(-x) &(&(target.os)~-x))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-x.a) &(&(target.os)~-x.a) &(-x.b) &(&(target.os)~-x.b) &(-x.c) &(&(target.os)~-x.c)"; strings.Count(s, s2) != 1 {
		ctx.err("%v", v)
	} else if s3 := "$(or $3,-x)$_$(or $4))"; strings.Count(s, s3) != 1 {
		ctx.err("%v", v)
	} else if s != "$(foreach(-unique) "+s1+" "+s2+","+s3 {
		ctx.err("%v", v)
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xyy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xzz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xsa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxb") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-xxc") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if s != "-xxx -xzz -xsx -xsz -xxa -xsa -xxb -xxc" {
		ctx.err("%v ; %v", s, v)
	}
	if v := ctx.val(".flag", []string{"a", "b", "c"}, "-z"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!a) &(&(target.os)~-z!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!b) &(&(target.os)~-z!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z!c) &(&(target.os)~-z!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-z) &(&(target.os)~-z)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.a) &(&(target.os)~-z.a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.b) &(&(target.os)~-z.b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-z.c) &(&(target.os)~-z.c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-z!a) &(&(target.os)~-z!a) &(-z!b) &(&(target.os)~-z!b) &(-z!c) &(&(target.os)~-z!c),&(-z) &(&(target.os)~-z))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-z.a) &(&(target.os)~-z.a) &(-z.b) &(&(target.os)~-z.b) &(-z.c) &(&(target.os)~-z.c)"; strings.Count(s, s2) != 1 {
		ctx.err("%v", v)
	} else if s3 := "$(or $3,-z)$_$(or $4))"; strings.Count(s, s3) != 1 {
		ctx.err("%v", v)
	} else if s != "$(foreach(-unique) "+s1+" "+s2+","+s3 {
		ctx.err("%v", v)
	} else if s := __string(ctx, v); s != "" {
		ctx.err("%v ; %v", s, v)
	}
	if v := ctx.val(".flag", []string{"a", "b", "c"}, "-x", "-y"); v == nil {
		ctx.err(".flag")
	} else if s := v.String(); s == "" {
		ctx.err("%v", v)
	} else if strings.Count(s, "$(foreach(-unique) $(filter-out &") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!a) &(&(target.os)~-x!a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!b) &(&(target.os)~-x!b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x!c) &(&(target.os)~-x!c),") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, ",&(-x) &(&(target.os)~-x)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.a) &(&(target.os)~-x.a)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.b) &(&(target.os)~-x.b)") != 1 {
		ctx.err("%v", v)
	} else if strings.Count(s, "&(-x.c) &(&(target.os)~-x.c)") != 1 {
		ctx.err("%v", v)
	} else if s1 := "$(filter-out &(-x!a) &(&(target.os)~-x!a) &(-x!b) &(&(target.os)~-x!b) &(-x!c) &(&(target.os)~-x!c),&(-x) &(&(target.os)~-x))"; strings.Count(s, s1) != 1 {
		ctx.err("%v", v)
	} else if s2 := "&(-x.a) &(&(target.os)~-x.a) &(-x.b) &(&(target.os)~-x.b) &(-x.c) &(&(target.os)~-x.c)"; strings.Count(s, s2) != 1 {
		ctx.err("%v", v)
	} else if s3 := "-y$_$(or $4))"; strings.Count(s, s3) != 1 {
		ctx.err("%v", v)
	} else if s != "$(foreach(-unique) "+s1+" "+s2+","+s3 {
		ctx.err("%v", v)
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yyy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yzz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysx") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysy") != 0 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysz") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-ysa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxa") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxb") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if strings.Count(s, "-yxc") != 1 {
		ctx.err("%v ; %v", s, v)
	} else if s != "-yxx -yzz -ysx -ysz -yxa -ysa -yxb -yxc" {
		ctx.err("%v ; %v", s, v)
	}

	if d1, d2 := ctx.def("cppflags"), ctx.def("fooflags"); d1 == nil || d2 == nil {
		ctx.err("cppflags")
		ctx.err("fooflags")
	} else if d1.value == nil {
		ctx.err("%v", d1)
	} else if d2.value == nil {
		ctx.err("%v", d2)
	} else if s := d1.value.String(); true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", d1.value)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,&(-v.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-D", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-f", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-I", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem-after", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag1("cppflags")) != 1 {
		ctx.err("%v", d1.value)
	} else if s := d2.value.String(); true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", d2.value)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", d2.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,&(-v.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, "$(foreach(-unique) $1 $2,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-D", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-f", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-I", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag2("-isystem-after", "$2")) != 1 {
		ctx.err("%v", d1.value)
	} else if strings.Count(s, flag1("cppflags")) != 1 {
		ctx.err("%v", d1.value)
	} else if s1, s2 := __string(ctx, d1.value), __string(ctx, d2.value); s1 != s2 {
		ctx.err("%v", s1)
		ctx.err("%v", s2)
		ctx.err("%v", d1.value)
		ctx.err("%v", d2.value)
	}

	if v1, v2 := ctx.val("cflags"), ctx.val("xflags"); v1 == nil || v2 == nil {
		ctx.err("cflags")
		ctx.err("xflags")
	} else if s := v1.String(); s == "" {
		ctx.err("%T %v", v1, v1)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v1)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&(-v.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v1)
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) $(foreach $1 c,&(-g.$_) &(&(target.os)~-g.$_))),-g)") != 1 {
		ctx.err("%v", v1)
	} else if t := flag2("-g", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-O", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-D", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-f", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-m", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-W", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-I", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-no", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-isystem", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag2("-isystem-after", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag1("cflags"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v2)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,&(-v.$_))") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(foreach(-unique) $1 c,-std=&(-std.$_) -std=&(std.$_))") != 1 {
		ctx.err("%v", v2)
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) $(foreach $1 c,&(-g.$_) &(&(target.os)~-g.$_))),-g)") != 1 {
		ctx.err("%v", v2)
	} else if t := flag2("-g", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-O", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-D", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-f", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-m", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-W", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-I", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-no", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-isystem", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag2("-isystem-after", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag1("cflags"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if s1, s2 := __string(ctx, v1), __string(ctx, v2); s1 != s2 {
		ctx.err("%v → %s ; %v → %s", v1, s1, v2, s2)
	}

	var crossbuild bool
	if v := ctx.val("cross.build"); v == nil {
		ctx.err("cross.build")
	} else if ct := ctx.val("cross.target"); ct == nil {
		ctx.err("cross.target")
	} else if crossbuild = __true(ctx, v); crossbuild {
		if strings.Count(__string(ctx, ct), "-") <= 0 {
			ctx.err("cross.target: %v → %v", ct, __string(ctx, ct))
		}
	}

	if v := ctx.val("std.fxxbxx"); v == nil {
		ctx.err("std.fxxbxx")
	} else if s := __string(ctx, v); s != "stdfxxbxx1" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}
	if v := ctx.val("-std.fxxbxx"); v == nil {
		ctx.err("-std.fxxbxx")
	} else if s := __string(ctx, v); s != "stdfxxbxx2" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	}
	if v1, v2 := ctx.val("cflags", "fxxbxx"), ctx.val("xflags", "fxxbxx"); v1 == nil || v2 == nil {
		ctx.err("cflags")
		ctx.err("xflags")
	} else if s := v1.String(); s == "" {
		ctx.err("%v", v1)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v1)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) &(-g.fxxbxx) &(&(target.os)~-g.fxxbxx) &(-g.c) &(&(target.os)~-g.c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if t := flag4("-g", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-O", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-D", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-f", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-m", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-W", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-I", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-no", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-isystem", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag4("-isystem-after", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if t := flag3("cflags", "fxxbxx"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v1)
	} else if s := v2.String(); s == "" {
		ctx.err("%T %v", v2, v2)
	} else if true && strings.Count(s, "&(cross.target)") != 1 {
		ctx.err("%v", v2)
	} else if false && strings.Count(s, "$(foreach(-unique) &(target.triple),--target=$_ -Xclang -triple=$_)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "&(-v.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.fxxbxx)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(-std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "-std=&(std.c)") != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, "$(if $(or &(-g) &(&(target.os)~-g) &(-g.fxxbxx) &(&(target.os)~-g.fxxbxx) &(-g.c) &(&(target.os)~-g.c)),-g)") != 1 {
		ctx.err("%v", s)
	} else if t := flag4("-g", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-O", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-D", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-f", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-m", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-W", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-I", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-no", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-isystem", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag4("-isystem-after", "fxxbxx", "c"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if t := flag3("cflags", "fxxbxx"); strings.Count(s, t) != 1 {
		note(ctx, "%v", t) ; ctx.err("%v", v2)
	} else if s1 := __string(ctx, v1); s1 == "" {
		ctx.err("%v → %s", ts(v2), s1)
	} else if s2 := __string(ctx, v2); s2 == "" {
		ctx.err("%v → %s", ts(v2), s2)
	} else if !validFlags(ctx, v1, s1) {
		ctx.err("%s", s1)
	} else if !validFlags(ctx, v2, s2) {
		ctx.err("%s", s2)
	} else if s := s1+s2; s1 != s1 {
		ctx.err("%s", s1)
		ctx.err("%s", s1)
		ctx.err("%v", v1)
		ctx.err("%v", v2)
	} else if strings.Count(s, "std=stdfxxbxx1") != 2 {
		ctx.err("%s", s1)
		ctx.err("%s", s2)
		ctx.err("%v", v1)
		ctx.err("%v", v2)
	} else if strings.Count(s, "-std=stdfxxbxx2") != 2 {
		ctx.err("%s", s1)
		ctx.err("%s", s2)
		ctx.err("%v", v1)
		ctx.err("%v", v2)
	}

	if v := ctx.val(".test.1"); v == nil {
		ctx.err(".test.1")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%v ⇒ %s", ts(v), s)
	} else if false {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v : %s ; %v", t, s, ts(v))
			}
		}
	} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v ⇒ %s", ts(v), s)
	}

	if v := ctx.val(".test.2"); v == nil {
		ctx.err(".test.2")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else if false {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v : %s ; %v", t, s, ts(v))
			}
		}
	} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v", v)
	}

	if v := ctx.val(".test.3"); v == nil {
		ctx.err(".test.3")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else if false {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v : %s ; %v", t, s, ts(v))
			}
		}
	} else if strings.Count(s, "-std=fxxbar") != 1 {
		ctx.err("%v", v)
	}

	if v := ctx.val(".test.4"); v == nil {
		ctx.err(".test.4")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo1 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.5"); v == nil {
		ctx.err(".test.5")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo2 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, v)
			}
		}
	}

	if v := ctx.val(".test.6"); v == nil {
		ctx.err(".test.6")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo3 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.7"); v == nil {
		ctx.err(".test.7")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, flag3("ldlibs", "foo")) != 1 {
		ctx.err("%v", s)
	} else if strings.Count(s, flag3("-l", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo4 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.8"); v == nil {
		ctx.err(".test.8")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, flag3("loadlibes", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo5 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}

	if v := ctx.val(".test.9"); v == nil {
		ctx.err(".test.9")
	} else if s := v.String(); s == "" {
		ctx.err("%v", tst{v})
	} else if strings.Count(s, flag3("loadlibs", "foo")) != 1 {
		ctx.err("%v", s)
	} else if s := __string(ctx, v); s == "" {
		ctx.err("%s : %v → %s", typeof(v), v, s)
	} else {
		for _, t := range foo6 {
			if n := strings.Count(s, t); n != 1 {
				ctx.err("%v (%d) : %s ; %v", t, n, s, ts(v))
			}
		}
	}
}

const testCxxincConfigLines = `
_LIBCPP_ABI_VERSION = '2'
_LIBCPP_ABI_NAMESPACE = '_extbit'
_LIBCPP_ABI_DEFINES = ((plain c) '//TODO: #define ...')
_LIBCPP_EXTRA_SITE_DEFINES = ((plain c) '//TODO: #define ...')
_LIBCPP_HAS_MUSL_LIBC = {=no}
_LIBCPP_HAS_PARALLEL_ALGORITHMS = {=no}
_LIBCPP_PSTL_CPU_BACKEND_SERIAL = {=no}
_LIBCPP_PSTL_CPU_BACKEND_THREAD = {=yes}
_LIBCPP_TYPEINFO_COMPARISON_IMPLEMENTATION = 1
LIBCXX_ENABLE_FILESYSTEM = {=yes}
LIBCXX_ENABLE_FSTREAM = {=yes}
LIBCXX_ENABLE_LOCALIZATION = {=yes}
LIBCXX_ENABLE_THREADS = {=yes}
LIBCXX_ENABLE_WIDE_CHARACTERS = {=yes}
requires_LIBCXX_ENABLE_WIDE_CHARACTERS =
requires_LIBCXX_ENABLE_FILESYSTEM =
requires_LIBCXX_ENABLE_THREADS =
requires_LIBCXX_ENABLE_LOCALIZATION =
requires_LIBCXX_ENABLE_FSTREAM =
`

const testCxxabiConfigLines = `
LIBCXXABI_ENABLE_NEW_DELETE_DEFINITIONS = {=yes}
LIBCXXABI_ENABLE_EXCEPTIONS = {=yes}
LIBCXXABI_ENABLE_THREADS = {=yes}
`

const testAppConfigLines = `
VERSION = 0.0.1
PACKAGE = extbit.app
PACKAGE_NAME = 'app'
PACKAGE_VERSION = 0.0.1
PACKAGE_VENDOR = 'ExtBit LLC'
PACKAGE_TARNAME = 'app'-0.0.1
PACKAGE_STRING = "app-0.0.1"
PACKAGE_URL = "https://extbit.dev/package/extbit.app/0.0.1"
PACKAGE_BUGREPORT = "https://extbit.dev/package/extbit.app/0.0.1/bugs"
HAVE_ALLOCA_H =
HAVE_ARPA_INET_H =
HAVE_ARPA_NAMESER_H =
HAVE_ARPA_TFTP_H =
HAVE_ASM_TYPES_H =
HAVE_ASSERT_H =
HAVE_ATOMIC_H =
HAVE_BLUETOOTH_BLUETOOTH_H =
HAVE_BLUETOOTH_H =
HAVE_BSD_STDLIB_H =
HAVE_BSD_STRING_H =
HAVE_BSD_UNISTD_H =
HAVE_COMPLEX_H =
HAVE_CONIO_H =
HAVE_CRASHREPORTERCLIENT_H =
HAVE_CRYPT_H =
HAVE_CTYPE_H =
HAVE_CURSES_H =
HAVE_DB_H =
HAVE_DIRECT_H =
HAVE_DIRENT_H =
HAVE_DLFCN_H =
HAVE_DL_H =
HAVE_EDITLINE_READLINE_H =
HAVE_ENDIAN_H =
HAVE_ERRNO_H =
HAVE_EXECINFO_H =
HAVE_FCNTL_H =
HAVE_FENV_H =
HAVE_FFI_FFI_H =
HAVE_FFI_H =
HAVE_FLOAT_H =
HAVE_FP_CLASS_H =
HAVE_GDBM-NDBM_H =
HAVE_GDBMERRNO_H =
HAVE_GDBM_H =
HAVE_GDBM_NDBM_H =
HAVE_GRP_H =
HAVE_IEEEFP_H =
HAVE_IFADDRS_H =
HAVE_INTRIN_H =
HAVE_INTTYPES_H =
HAVE_IO_H =
HAVE_JEMALLOC_JEMALLOC_H =
HAVE_LANGINFO_H =
HAVE_LIBBSD =
HAVE_LIBCRYPT =
HAVE_LIBCURSES =
HAVE_LIBDBM =
HAVE_LIBDL =
HAVE_LIBDLD =
HAVE_LIBEDIT =
HAVE_LIBGEN =
HAVE_LIBGEN_H =
HAVE_LIBHISTORY =
HAVE_LIBIEEE =
HAVE_LIBINTL =
HAVE_LIBINTL_H =
HAVE_LIBJEMALLOC =
HAVE_LIBLZMA =
HAVE_LIBNCURSES =
HAVE_LIBNCURSESW =
HAVE_LIBPFM =
HAVE_LIBPSAPI =
HAVE_LIBPTHREAD =
HAVE_LIBREADLINE =
HAVE_LIBRESOLV =
HAVE_LIBRT =
HAVE_LIBSENDFILE =
HAVE_LIBTERMINFO =
HAVE_LIBTINFO =
HAVE_LIBUNWIND =
HAVE_LIBUTIL =
HAVE_LIBUUID =
HAVE_LIBXAR =
HAVE_LIBZ =
HAVE_LIMITS_H =
HAVE_LINK_H =
HAVE_LINUX_CAN_BCM_H =
HAVE_LINUX_CAN_H =
HAVE_LINUX_CAN_J1939_H =
HAVE_LINUX_CAN_RAW_H =
HAVE_LINUX_CLOSE_RANGE_H =
HAVE_LINUX_IF_ALG_H =
HAVE_LINUX_MEMFD_H =
HAVE_LINUX_NETLINK_H =
HAVE_LINUX_QRTR_H =
HAVE_LINUX_RANDOM_H =
HAVE_LINUX_SOUNDCARD_H =
HAVE_LINUX_TCP_H =
HAVE_LINUX_TIPC_H =
HAVE_LINUX_VM_SOCKETS_H =
HAVE_LINUX_WAIT_H =
HAVE_LOCALE_H =
HAVE_LZMA_H =
HAVE_MACH-O_DYLD_H =
HAVE_MACH_MACH_H =
HAVE_MACH_MACH_TIME_H =
HAVE_MALLOC_H =
HAVE_MALLOC_MALLOC_H =
HAVE_MALLOC_NP_H =
HAVE_MATH_H =
HAVE_MBARRIER_H =
HAVE_MEMORY_H =
HAVE_MKDEV_H =
HAVE_NCURSES_H =
HAVE_NDBM_H =
HAVE_NDIR_H =
HAVE_NETDB_H =
HAVE_NETINET_IN_H =
HAVE_NETINET_TCP_H =
HAVE_NETPACKET_PACKET_H =
HAVE_NET_IF_H =
HAVE_PERFMON_PERF_EVENT_H =
HAVE_PERFMON_PFMLIB_H =
HAVE_PERFMON_PFMLIB_PERF_EVENT_H =
HAVE_POLL_H =
HAVE_PROCESS_H =
HAVE_PTHREAD_H =
HAVE_PTY_H =
HAVE_PWD_H =
HAVE_READLINE_HISTORY_H =
HAVE_READLINE_READLINE_H =
HAVE_RESOLV_H =
HAVE_SCHED_H =
HAVE_SEMAPHORE_H =
HAVE_SETJMP_H =
HAVE_SHADOW_H =
HAVE_SIGNAL_H =
HAVE_SPAWN_H =
HAVE_STDARG_H =
HAVE_STDATOMIC_H =
HAVE_STDBOOL_H =
HAVE_STDDEF_H =
HAVE_STDINT_H =
HAVE_STDIO_H =
HAVE_STDLIB_H =
HAVE_STRINGS_H =
HAVE_STRING_H =
HAVE_STROPTS_H =
HAVE_STRUCT_ADDRINFO =
HAVE_STRUCT_PASSWD =
HAVE_STRUCT_PASSWD_PW_GECOS =
HAVE_STRUCT_PASSWD_PW_PASSWD =
HAVE_STRUCT_SOCKADDR =
HAVE_STRUCT_SOCKADDR_SA_LEN =
HAVE_STRUCT_SOCKADDR_STORAGE =
HAVE_STRUCT_SOCKADDR_STORAGE_SS_FAMILY =
HAVE_STRUCT_SOCKADDR_STORAGE___SS_FAMILY =
HAVE_STRUCT_STAT =
HAVE_STRUCT_STATFS =
HAVE_STRUCT_STATFS_F_FLAGS =
HAVE_STRUCT_STATFS_F_FSTYPENAME =
HAVE_STRUCT_STATVFS =
HAVE_STRUCT_STATVFS_F_FLAGS =
HAVE_STRUCT_STATVFS_F_FSTYPENAME =
HAVE_STRUCT_STAT_ST_ATIMESPEC_TV_NSEC =
HAVE_STRUCT_STAT_ST_ATIM_NSEC =
HAVE_STRUCT_STAT_ST_ATIM_TV_NSEC =
HAVE_STRUCT_STAT_ST_BIRTHTIME =
HAVE_STRUCT_STAT_ST_BLKSIZE =
HAVE_STRUCT_STAT_ST_BLOCKS =
HAVE_STRUCT_STAT_ST_CTIMESPEC_TV_NSEC =
HAVE_STRUCT_STAT_ST_CTIM_NSEC =
HAVE_STRUCT_STAT_ST_CTIM_TV_NSEC =
HAVE_STRUCT_STAT_ST_FLAGS =
HAVE_STRUCT_STAT_ST_GEN =
HAVE_STRUCT_STAT_ST_MTIMESPEC_TV_NSEC =
HAVE_STRUCT_STAT_ST_MTIM_NSEC =
HAVE_STRUCT_STAT_ST_MTIM_TV_NSEC =
HAVE_STRUCT_STAT_ST_RDEV =
HAVE_STRUCT_TIMEVAL =
HAVE_STRUCT_TIMEVAL_TV_SEC =
HAVE_STRUCT_TIMEVAL_TV_USEC =
HAVE_STRUCT_TM =
HAVE_STRUCT_TM_TM_ZONE =
HAVE_SYSEXITS_H =
HAVE_SYSMACROS_H =
HAVE_SYS_AUDIOIO_H =
HAVE_SYS_AUDIO_H =
HAVE_SYS_BSDTTY_H =
HAVE_SYS_DEVPOLL_H =
HAVE_SYS_DIR_H =
HAVE_SYS_ENDIAN_H =
HAVE_SYS_EPOLL_H =
HAVE_SYS_EVENTFD_H =
HAVE_SYS_EVENT_H =
HAVE_SYS_FILE_H =
HAVE_SYS_FILIO_H =
HAVE_SYS_IOCTL_H =
HAVE_SYS_KERN_CONTROL_H =
HAVE_SYS_LOADAVG_H =
HAVE_SYS_LOCK_H =
HAVE_SYS_MEMFD_H =
HAVE_SYS_MKDEV_H =
HAVE_SYS_MMAN_H =
HAVE_SYS_MODEM_H =
HAVE_SYS_MOUNT_H =
HAVE_SYS_NDIR_H =
HAVE_SYS_PARAM_H =
HAVE_SYS_POLLSET_H =
HAVE_SYS_POLL_H =
HAVE_SYS_RANDOM_H =
HAVE_SYS_RESOURCE_H =
HAVE_SYS_SELECT_H =
HAVE_SYS_SENDFILE_H =
HAVE_SYS_SOCKET_H =
HAVE_SYS_SOCKIO_H =
HAVE_SYS_SOUNDCARD_H =
HAVE_SYS_STATFS_H =
HAVE_SYS_STATVFS_H =
HAVE_SYS_STAT_H =
HAVE_SYS_SYSCALL_H =
HAVE_SYS_SYSMACROS_H =
HAVE_SYS_SYS_DOMAIN_H =
HAVE_SYS_TERMIO_H =
HAVE_SYS_TIMEB_H =
HAVE_SYS_TIMES_H =
HAVE_SYS_TIME_H =
HAVE_SYS_TYPES_H =
HAVE_SYS_UIO_H =
HAVE_SYS_UN_H =
HAVE_SYS_UTIME_H =
HAVE_SYS_UTSNAME_H =
HAVE_SYS_VFS_H =
HAVE_SYS_WAIT_H =
HAVE_SYS_XATTR_H =
HAVE_TERMIOS_H =
HAVE_TERMIO_H =
HAVE_TERM_H =
HAVE_TIME_H =
HAVE_TYPE_ATOMIC_INT =
HAVE_TYPE_ATOMIC_UINTPTR_T =
HAVE_TYPE_BLKCNT_T =
HAVE_TYPE_BLKSIZE_T =
HAVE_TYPE_BOOL =
HAVE_TYPE_CHAR =
HAVE_TYPE_CLOCKID_T =
HAVE_TYPE_CLOCK_T =
HAVE_TYPE_CONST_CHAR =
HAVE_TYPE_DEV_T =
HAVE_TYPE_DOUBLE =
HAVE_TYPE_FLOAT =
HAVE_TYPE_FPOS_T =
HAVE_TYPE_FSBLKCNT_T =
HAVE_TYPE_FSFILCNT_T =
HAVE_TYPE_GID_T =
HAVE_TYPE_ID_T =
HAVE_TYPE_INO_T =
HAVE_TYPE_INT =
HAVE_TYPE_KEY_T =
HAVE_TYPE_LONG =
HAVE_TYPE_LONG_DOUBLE =
HAVE_TYPE_LONG_LONG =
HAVE_TYPE_MODE_T =
HAVE_TYPE_NLINK_T =
HAVE_TYPE_OFF_T =
HAVE_TYPE_PID_T =
HAVE_TYPE_PTHREAD_ATTR_T =
HAVE_TYPE_PTHREAD_CONDATTR_T =
HAVE_TYPE_PTHREAD_COND_T =
HAVE_TYPE_PTHREAD_KEY_T =
HAVE_TYPE_PTHREAD_MUTEXATTR_T =
HAVE_TYPE_PTHREAD_MUTEX_T =
HAVE_TYPE_PTHREAD_ONCE_T =
HAVE_TYPE_PTHREAD_RWLOCKATTR_T =
HAVE_TYPE_PTHREAD_RWLOCK_T =
HAVE_TYPE_PTHREAD_T =
HAVE_TYPE_PTRDIFF_T =
HAVE_TYPE_SA_FAMILY_T =
HAVE_TYPE_SHORT =
HAVE_TYPE_SIGINFO_T =
HAVE_TYPE_SIGNED_CHAR =
HAVE_TYPE_SIZE_T =
HAVE_TYPE_SOCKLEN_T =
HAVE_TYPE_SSIZE_T =
HAVE_TYPE_SUSECONDS_T =
HAVE_TYPE_TIMER_T =
HAVE_TYPE_TIME_T =
HAVE_TYPE_UID_T =
HAVE_TYPE_UINT32_T =
HAVE_TYPE_UINT64_T =
HAVE_TYPE_UINTPTR_T =
HAVE_TYPE_USECONDS_T =
HAVE_TYPE_VOID_P =
HAVE_TYPE_WCHAR_T =
HAVE_TYPE__BOOL =
HAVE_TYPE___INT64 =
HAVE_TYPE___INT64_T =
HAVE_UNISTD_H =
HAVE_UNWIND_H =
HAVE_UTIL_H =
HAVE_UTIME_H =
HAVE_UTMP_H =
HAVE_UUID_H =
HAVE_UUID_UUID_H =
HAVE_VALGRIND_VALGRIND_H =
HAVE_WCHAR_H =
HAVE_ZLIB_H =
ALIGNOF_ATOMIC_INT =
ALIGNOF_ATOMIC_INT_CODE =
ALIGNOF_ATOMIC_UINTPTR_T =
ALIGNOF_ATOMIC_UINTPTR_T_CODE =
ALIGNOF_BLKCNT_T =
ALIGNOF_BLKCNT_T_CODE =
ALIGNOF_BLKSIZE_T =
ALIGNOF_BLKSIZE_T_CODE =
ALIGNOF_BOOL =
ALIGNOF_BOOL_CODE =
ALIGNOF_CHAR =
ALIGNOF_CHAR_CODE =
ALIGNOF_CLOCKID_T =
ALIGNOF_CLOCKID_T_CODE =
ALIGNOF_CLOCK_T =
ALIGNOF_CLOCK_T_CODE =
ALIGNOF_CONST_CHAR =
ALIGNOF_CONST_CHAR_CODE =
ALIGNOF_DEV_T =
ALIGNOF_DEV_T_CODE =
ALIGNOF_DOUBLE =
ALIGNOF_DOUBLE_CODE =
ALIGNOF_FLOAT =
ALIGNOF_FLOAT_CODE =
ALIGNOF_FPOS_T =
ALIGNOF_FPOS_T_CODE =
ALIGNOF_FSBLKCNT_T =
ALIGNOF_FSBLKCNT_T_CODE =
ALIGNOF_FSFILCNT_T =
ALIGNOF_FSFILCNT_T_CODE =
ALIGNOF_GID_T =
ALIGNOF_GID_T_CODE =
ALIGNOF_ID_T =
ALIGNOF_ID_T_CODE =
ALIGNOF_INO_T =
ALIGNOF_INO_T_CODE =
ALIGNOF_INT =
ALIGNOF_INT_CODE =
ALIGNOF_KEY_T =
ALIGNOF_KEY_T_CODE =
ALIGNOF_LONG =
ALIGNOF_LONG_CODE =
ALIGNOF_LONG_DOUBLE =
ALIGNOF_LONG_DOUBLE_CODE =
ALIGNOF_LONG_LONG =
ALIGNOF_LONG_LONG_CODE =
ALIGNOF_MODE_T =
ALIGNOF_MODE_T_CODE =
ALIGNOF_NLINK_T =
ALIGNOF_NLINK_T_CODE =
ALIGNOF_OFF_T =
ALIGNOF_OFF_T_CODE =
ALIGNOF_PID_T =
ALIGNOF_PID_T_CODE =
ALIGNOF_PTHREAD_ATTR_T =
ALIGNOF_PTHREAD_ATTR_T_CODE =
ALIGNOF_PTHREAD_CONDATTR_T =
ALIGNOF_PTHREAD_CONDATTR_T_CODE =
ALIGNOF_PTHREAD_COND_T =
ALIGNOF_PTHREAD_COND_T_CODE =
ALIGNOF_PTHREAD_KEY_T =
ALIGNOF_PTHREAD_KEY_T_CODE =
ALIGNOF_PTHREAD_MUTEXATTR_T =
ALIGNOF_PTHREAD_MUTEXATTR_T_CODE =
ALIGNOF_PTHREAD_MUTEX_T =
ALIGNOF_PTHREAD_MUTEX_T_CODE =
ALIGNOF_PTHREAD_ONCE_T =
ALIGNOF_PTHREAD_ONCE_T_CODE =
ALIGNOF_PTHREAD_RWLOCKATTR_T =
ALIGNOF_PTHREAD_RWLOCKATTR_T_CODE =
ALIGNOF_PTHREAD_RWLOCK_T =
ALIGNOF_PTHREAD_RWLOCK_T_CODE =
ALIGNOF_PTHREAD_T =
ALIGNOF_PTHREAD_T_CODE =
ALIGNOF_PTRDIFF_T =
ALIGNOF_PTRDIFF_T_CODE =
ALIGNOF_SA_FAMILY_T =
ALIGNOF_SA_FAMILY_T_CODE =
ALIGNOF_SHORT =
ALIGNOF_SHORT_CODE =
ALIGNOF_SIGINFO_T =
ALIGNOF_SIGINFO_T_CODE =
ALIGNOF_SIGNED_CHAR =
ALIGNOF_SIGNED_CHAR_CODE =
ALIGNOF_SIZE_T =
ALIGNOF_SIZE_T_CODE =
ALIGNOF_SOCKLEN_T =
ALIGNOF_SOCKLEN_T_CODE =
ALIGNOF_SSIZE_T =
ALIGNOF_SSIZE_T_CODE =
ALIGNOF_SUSECONDS_T =
ALIGNOF_SUSECONDS_T_CODE =
ALIGNOF_TIMER_T =
ALIGNOF_TIMER_T_CODE =
ALIGNOF_TIME_T =
ALIGNOF_TIME_T_CODE =
ALIGNOF_UID_T =
ALIGNOF_UID_T_CODE =
ALIGNOF_UINT32_T =
ALIGNOF_UINT32_T_CODE =
ALIGNOF_UINT64_T =
ALIGNOF_UINT64_T_CODE =
ALIGNOF_UINTPTR_T =
ALIGNOF_UINTPTR_T_CODE =
ALIGNOF_USECONDS_T =
ALIGNOF_USECONDS_T_CODE =
ALIGNOF_VOID_P =
ALIGNOF_VOID_P_CODE =
ALIGNOF_WCHAR_T =
ALIGNOF_WCHAR_T_CODE =
ALIGNOF__BOOL =
ALIGNOF__BOOL_CODE =
ALIGNOF___INT64 =
ALIGNOF___INT64_CODE =
ALIGNOF___INT64_T =
ALIGNOF___INT64_T_CODE =
SIZEOF_ATOMIC_INT =
SIZEOF_ATOMIC_INT_CODE =
SIZEOF_ATOMIC_UINTPTR_T =
SIZEOF_ATOMIC_UINTPTR_T_CODE =
SIZEOF_BLKCNT_T =
SIZEOF_BLKCNT_T_CODE =
SIZEOF_BLKSIZE_T =
SIZEOF_BLKSIZE_T_CODE =
SIZEOF_BOOL =
SIZEOF_BOOL_CODE =
SIZEOF_CHAR =
SIZEOF_CHAR_CODE =
SIZEOF_CLOCKID_T =
SIZEOF_CLOCKID_T_CODE =
SIZEOF_CLOCK_T =
SIZEOF_CLOCK_T_CODE =
SIZEOF_CONST_CHAR =
SIZEOF_CONST_CHAR_CODE =
SIZEOF_DEV_T =
SIZEOF_DEV_T_CODE =
SIZEOF_DOUBLE =
SIZEOF_DOUBLE_CODE =
SIZEOF_FLOAT =
SIZEOF_FLOAT_CODE =
SIZEOF_FPOS_T =
SIZEOF_FPOS_T_CODE =
SIZEOF_FSBLKCNT_T =
SIZEOF_FSBLKCNT_T_CODE =
SIZEOF_FSFILCNT_T =
SIZEOF_FSFILCNT_T_CODE =
SIZEOF_GID_T =
SIZEOF_GID_T_CODE =
SIZEOF_ID_T =
SIZEOF_ID_T_CODE =
SIZEOF_INO_T =
SIZEOF_INO_T_CODE =
SIZEOF_INT =
SIZEOF_INT_CODE =
SIZEOF_KEY_T =
SIZEOF_KEY_T_CODE =
SIZEOF_LONG =
SIZEOF_LONG_CODE =
SIZEOF_LONG_DOUBLE =
SIZEOF_LONG_DOUBLE_CODE =
SIZEOF_LONG_LONG =
SIZEOF_LONG_LONG_CODE =
SIZEOF_MODE_T =
SIZEOF_MODE_T_CODE =
SIZEOF_NLINK_T =
SIZEOF_NLINK_T_CODE =
SIZEOF_OFF_T =
SIZEOF_OFF_T_CODE =
SIZEOF_PID_T =
SIZEOF_PID_T_CODE =
SIZEOF_PTHREAD_ATTR_T =
SIZEOF_PTHREAD_ATTR_T_CODE =
SIZEOF_PTHREAD_CONDATTR_T =
SIZEOF_PTHREAD_CONDATTR_T_CODE =
SIZEOF_PTHREAD_COND_T =
SIZEOF_PTHREAD_COND_T_CODE =
SIZEOF_PTHREAD_KEY_T =
SIZEOF_PTHREAD_KEY_T_CODE =
SIZEOF_PTHREAD_MUTEXATTR_T =
SIZEOF_PTHREAD_MUTEXATTR_T_CODE =
SIZEOF_PTHREAD_MUTEX_T =
SIZEOF_PTHREAD_MUTEX_T_CODE =
SIZEOF_PTHREAD_ONCE_T =
SIZEOF_PTHREAD_ONCE_T_CODE =
SIZEOF_PTHREAD_RWLOCKATTR_T =
SIZEOF_PTHREAD_RWLOCKATTR_T_CODE =
SIZEOF_PTHREAD_RWLOCK_T =
SIZEOF_PTHREAD_RWLOCK_T_CODE =
SIZEOF_PTHREAD_T =
SIZEOF_PTHREAD_T_CODE =
SIZEOF_PTRDIFF_T =
SIZEOF_PTRDIFF_T_CODE =
SIZEOF_SA_FAMILY_T =
SIZEOF_SA_FAMILY_T_CODE =
SIZEOF_SHORT =
SIZEOF_SHORT_CODE =
SIZEOF_SIGINFO_T =
SIZEOF_SIGINFO_T_CODE =
SIZEOF_SIGNED_CHAR =
SIZEOF_SIGNED_CHAR_CODE =
SIZEOF_SIZE_T =
SIZEOF_SIZE_T_CODE =
SIZEOF_SOCKLEN_T =
SIZEOF_SOCKLEN_T_CODE =
SIZEOF_SSIZE_T =
SIZEOF_SSIZE_T_CODE =
SIZEOF_SUSECONDS_T =
SIZEOF_SUSECONDS_T_CODE =
SIZEOF_TIMER_T =
SIZEOF_TIMER_T_CODE =
SIZEOF_TIME_T =
SIZEOF_TIME_T_CODE =
SIZEOF_UID_T =
SIZEOF_UID_T_CODE =
SIZEOF_UINT32_T =
SIZEOF_UINT32_T_CODE =
SIZEOF_UINT64_T =
SIZEOF_UINT64_T_CODE =
SIZEOF_UINTPTR_T =
SIZEOF_UINTPTR_T_CODE =
SIZEOF_USECONDS_T =
SIZEOF_USECONDS_T_CODE =
SIZEOF_VOID_P =
SIZEOF_VOID_P_CODE =
SIZEOF_WCHAR_T =
SIZEOF_WCHAR_T_CODE =
SIZEOF__BOOL =
SIZEOF__BOOL_CODE =
SIZEOF___INT64 =
SIZEOF___INT64_CODE =
SIZEOF___INT64_T =
SIZEOF___INT64_T_CODE =
`
func testLLVMConfig1(ctx *testcase) {
	testVariantTargetVars(ctx)

	var p = _project(ctx)
	var names = []string{
		".configure", "configuration.sm", "stamp", "foo.log",
		".deps/11/22/333333333333333333333333333333333333333333333333333333333333",
		".grep/11/22/333333333333333333333333333333333333333333333333333333333333",
		".cache/11/22/333333333333333333333333333333333333333333333333333333333333",
		".configure/type/align/test.c",
		".configure/type/size/test.c",
		".configure/type/test.c",
		".configure/type/xxx/test.c",
		".configure/type/xxx/yyy/test.c",
		".configure/xxx/yyy/zzz/test.c",
		".configure/xxx/yyy/zzz/test.c++",
		".configure/xxx/yyy/zzz/test.o",
		".configure/xxx/yyy/zzz/test.log",
		".configure/test_xxx.c",
		".configure/test_xxx.c++",
		".configure/test_xxx.log",
		".configure/xxx.x",
		".configure/xxx.o",
		".configure/xxx.c",
		".configure/xxx.c++",
		".configure/xxx.log",
		".configure/std/x.stdc.headers.o",
		".configure/std/x.stdc.headers.c",
		".configure/std/x.stdc.headers.c++",
		".configure/std/x.words.bigendian",
		".configure/std/x.words.bigendian.o",
		".configure/std/x.words.bigendian.c",
		".configure/std/x.words.bigendian.c++",
		".configure/std/x.float.words.bigendian.o",
		".configure/std/x.float.words.bigendian.c",
		".configure/std/x.float.words.bigendian.c++",
	}

	for _, name := range names {
		if f := unmap_files(ctx, p, name, nil); f == nil {
			ctx.err("unmap %s", name)
		}
	}

	var tail = "/arm64-darwin"
	var name, ver1, ver2, ver3 string
	var ver1_val, ver2_val, ver3_val Value
	if !strings.HasSuffix(p.absPath, tail) {
		ctx.err("%v: %v", p, p.absPath)
	}

	name = "LLVM_VERSION_MAJOR"
	if v := ctx.val(name); v == nil {
		ctx.err("%s", name)
	} else if v.String() != "$(grep {=regex ^ *set\\(LLVM_VERSION_MAJOR +([0-9]+) *\\)},$1,{=file LLVMVersion.cmake})" { // '18'
		ctx.err("%s: %v : %s", name, v, tst{v})
	} else if t := expand(_final(ctx),v); ts(t) != "{=strlit 20}" {
		ctx.err("%s: %v : %s", name, v, tst{t})
	} else if ver1_val, ver1 = v, __string(ctx, v); ver1 != "20" {
		ctx.err("%s: %v : %v", name, ver1, v)
	}

	name = "LLVM_VERSION_MINOR"
	if v := ctx.val(name); v == nil {
		ctx.err("%s", name)
	} else if v.String() != "$(grep {=regex ^ *set\\(LLVM_VERSION_MINOR +([0-9]+) *\\)},$1,{=file LLVMVersion.cmake})" { // '0'
		ctx.err("%s: %v : %s", name, v, tst{v})
	} else if t := expand(_final(ctx),v); ts(t) != "{=strlit 0}" {
		ctx.err("%s: %v : %s", name, v, tst{t})
	} else if ver2_val, ver2 = v, __string(ctx, v); ver2 != "0" {
		ctx.err("%s: %v : %v", name, ver2, v)
	}

	name = "LLVM_VERSION_PATCH"
	if v := ctx.val(name); v == nil {
		ctx.err("%s", name)
	} else if v.String() != "$(grep {=regex ^ *set\\(LLVM_VERSION_PATCH +([0-9]+) *\\)},$1,{=file LLVMVersion.cmake})" { // '0'
		ctx.err("%s: %v : %s", name, v, tst{v})
	} else if t := expand(_final(ctx),v); ts(t) != "{=strlit 0}" {
		ctx.err("%s: %v : %s", name, v, tst{t})
	} else if ver3_val, ver3 = v, __string(ctx, v); ver3 != "0" {
		ctx.err("%s: %v : %v", name, ver3, v)
	}

	var general *project
	if o := ctx.obj("general"); o == nil {
		ctx.err("general")
	} else if p, y := o.(*project); !y {
		ctx.err("%v", tst{o})
	} else {
		general = p
	}

	var proj, base *project
	if proj = _project(ctx); proj == nil {
		ctx.err("configure fail")
	} else if !strings.HasSuffix(proj.absPath, tail) {
		ctx.err("%v: %v %v", proj, proj.absPath, tail)
	} else if proj.configure != nil {
		ctx.err("%v: configure", proj)
	} else if proj.resolve(ctx, "general") != ctx.obj("general") {
		ctx.err("%v: %v != %v", proj, proj.resolve(ctx, "general"), ctx.obj("general"))
	} else if len(proj.bases) != 1 {
		ctx.err("bases: %v", proj.bases)
	} else if base = proj.bases[0]; base.name != "testllvmconfig" {
		ctx.err("base: %v", base)
	} else if proj = base; len(proj.bases) != 1 {
		ctx.err("bases: %v", base.bases)
	} else if proj.configure != nil {
		ctx.err("proj.configure")
	} else if proj.resolve(ctx, "general") != ctx.obj("general") {
		ctx.err("%v: %v != %v", proj, proj.resolve(ctx, "general"), ctx.obj("general"))
	} else if base = proj.bases[0]; base.name != "llvm.Config" {
		ctx.err("base: %v", base)
	} else if base.configure == nil {
		ctx.err("base.configure")
	} else {
		for _, name := range names {
			if f := findfile(ctx, name, base.configure); f == nil {
				ctx.err("file %s", name)
			}
		}
	}

	var ver Value
	s := "configure.version"
	if o := proj.resolve(ctx, s); o == nil {
		ctx.err("%v: %s", proj, s)
	} else if false {
		if o2 := proj.configure.resolve(ctx, s); o != o2 {
			ctx.err("%v: %v != %v", proj, o, o2)
		}
	} else if d, y := o.(*def); !y {
		ctx.err("%v", o)
	} else if ver = d.value; ver == nil {
		ctx.err("%v", o)
	} else if __string(ctx, ver) != fmt.Sprintf("%v.%v.%v", ver1, ver2, ver3) {
		ctx.err("%v: %v (%v.%v.%v)", typeof(ver), __string(ctx, ver), ver1, ver2, ver3)
	} else if d, y = ver.(*def); !y {
		ctx.err("%v", o)
	} else if ver = d.value; ver == nil {
		ctx.err("%v", o)
	} else if ver.String() != fmt.Sprintf("%v.%v.%v", ver1_val, ver2_val, ver3_val) {
		ctx.err("%v", ts(ver))
	}

	s = "configure.package"
	if o := proj.resolve(ctx, s); o == nil {
		ctx.err(s)
	} else if false {
		if o2 := proj.configure.resolve(ctx, s); o != o2 {
			ctx.err("%v: %v != %v", proj, o, o2)
		}
	} else if d, y := o.(*def); !y {
		ctx.err("%v", o)
	} else if d.value == nil {
		ctx.err("%v", o)
	} else if pkg := "extbit.llvm"; d.value.String() != pkg { // "&(name)"
		ctx.err("%v", ts(d.value))
	} else if r := proj.entry(ctx, _strlit(_position(ctx), "VERSION"), false); r == nil {
		ctx.err("VERSION")
	} else if len(r.programs()) != 1 {
		ctx.err("VERSION: %v", r)
	} else if recipes := r.programs()[0].recipes; len(recipes) != 1 {
		ctx.err("VERSION: %v", r)
	} else {
		recipe := recipes[0]//.expand(_final(ctx))
		verval := expand(_final(ctx),ver)
		if x, y := recipe.(*list); !y {
			ctx.err("%v", tst{recipe})
		} else if x.len() == 0 { // != 1
			ctx.err("%v %v", x.len(), ts(x.elems))
		} else if elem := x.elems[0]; elem == nil {
			ctx.err("%v", tst{elem})
		} else if x, y := elem.(*def); !y {
			ctx.err("%v", tst{elem})
		} else if v := expand(_final(ctx),x.value); v == nil {
			ctx.err("%v", tst{x.value})
		} else if false && v.String() != verval.String() {
			ctx.err("%v: %v != %v", typeof(v), v, verval)
		}

		s = "PACKAGE"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_NAME"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_VERSION"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_VENDOR"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_TARNAME"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_STRING"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("s: %v", s, r)
		}

		s = "PACKAGE_URL"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}

		s = "PACKAGE_BUGREPORT"
		if r = proj.entry(ctx, _strlit(_position(ctx), s), false); r == nil {
			ctx.err(s)
		} else if len(r.programs()) != 1 {
			ctx.err("%s: %v", s, r)
		}
	}

	if v := ctx.val("/", proj); v == nil {
		ctx.err("/")
	} else if _, y := v.(*path); !y {
		ctx.err("%v", ts(v))
	} else if v.String() != proj.absPath {
		ctx.err("%v != %v", ts(v), proj.absPath)
	} else if __string(ctx, v) != proj.absPath {
		ctx.err("%v != %v", ts(v), proj.absPath)
	}

	var outtmp string
	var outtmp_val = ctx.val("outtmp", proj)
	if v := outtmp_val; v == nil {
		ctx.err("%v", ts(v))
	} else if outtmp = __string(/*closure_with(ctx, proj)*/ctx, v); outtmp == "" {
		ctx.err("%v", ts(v))
	} else if strings.HasSuffix(outtmp, tail) {
		outtmp = strings.TrimSuffix(outtmp, tail)
	}

	if !filepath.IsAbs(outtmp) {
		ctx.err("%v: %v → %v", proj, outtmp_val, outtmp)
	} else if i := strings.Index(outtmp, "/testdata/"); i <= 0 {
		ctx.err("%v: %v → %v", proj, outtmp_val, outtmp)
	} else if !strings.HasSuffix(proj.absPath, outtmp[i:]) {
		note(ctx, "%v: %v (%v, %v)", proj, proj.absPath, proj.spec, proj.rel)
		ctx.err("%v: %v → %v", proj, outtmp_val, outtmp)
	}

	s = "root1"
	if v1 := ctx.val(s); v1 == nil {
		ctx.err(s)
	} else if _, y := v1.(*path); !y {
		ctx.err("%v", ts(v1))
	} else if v1.String() != proj.absPath {
		ctx.err("%v", ts(v1))
	} else if __string(ctx,v1) != proj.absPath {
		ctx.err("%v: %v", _project(ctx), tv(v1))
	} else if v2 := ctx.val("root2"); v2 == nil {
		ctx.err("root2")
	} else if _, y := v2.(*path); !y {
		ctx.err("%v", ts(v2))
	} else if v2.String() != proj.absPath {
		ctx.err("%v", ts(v2))
	} else if __string(ctx,v2) != proj.absPath {
		ctx.err("%v", ts(v2))
	} else if __string(ctx,v2) != __string(ctx,v1) {
		ctx.err("%v: %v != %v", _project(ctx), v2, v1)
	} else if v3 := ctx.val("root3"); v3 == nil {
		ctx.err("root3")
	} else if c, y := v3.(*closure); !y {
		ctx.err("%v", tst{v3})
	} else if t := expand(_final(ctx),c); t == nil {
		ctx.err("%v", c)
	} else if p, y := t.(*path); !y || len(p.elems) == 0 {
		ctx.err("%v: %v: %v", c, typeof(t), t)
	} else if p.elems[len(p.elems)-1].String() != filepath.Base(tail) {
		ctx.err("%v: %v → %v ; %v", proj, c, p, tail)
	} else if cc := closure_with(ctx.Context, proj); _project(cc) == nil {
		ctx.err("%v: %v != %v", c, _project(cc), proj)
	} else if t := expand(_final(cc),c); t == nil {
		ctx.err("%v", c)
	} else if p, y := t.(*path); !y {
		ctx.err("%v: %v", c, tv(t))
	} else if len(p.elems) == 0 {
		ctx.err("%v: %v", c, p.elems)
	} else if p.elems[len(p.elems)-1].String() == filepath.Base(tail) {
		ctx.err("%v: %v → %v ; %v", proj, c, p, tail)
	} else if v3.String() != "&/" {
		ctx.err("%v", tv(v3))
	} else if __string(cc,v3) != __string(ctx,v1) {
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v1))
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v3))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if __string(cc,v3) != __string(ctx,v2) {
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v2))
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v3))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if __string(ctx,v3) == __string(ctx,v1) {
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v1))
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v3))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if __string(ctx,v3) == __string(ctx,v2) {
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v2))
		note(ctx, "%v: %v", _project(ctx), __string(ctx,v3))
		ctx.err("%v: %v ; %v{%v}",  _project(ctx), v3, typeof(ctx.Context), typeof(inner(ctx.Context)))
	} else if !strings.HasSuffix(__string(ctx,v3), tail) {
		ctx.err("%v: %v", tv(v3), tail)
	} else if strings.HasSuffix(__string(cc,v3), tail) {
		ctx.err("%v: %v", tv(v3), tail)
	}

	var chop0, chop1 string
	var chop3 = fmt.Sprintf("%%%%/.smart/modules/ %s/ %s/ %s/",
		filepath.Dir(general.absPath),
		filepath.Dir(filepath.Dir(general.absPath)),
		filepath.Dir(filepath.Dir(filepath.Dir(general.absPath))))

	if v := ctx.val("chop0"); v == nil {
		ctx.err("chop0")
	} else if chop0 = __string(ctx, v); chop0 == "" {
		ctx.err("%v", tv(v))
	}

	if v := ctx.val("chop1"); v == nil {
		ctx.err("chop1")
	} else if chop1 = __string(ctx, v); chop1 == "" {
		ctx.err("%v", tv(v))
	} else if strings.HasSuffix(chop1, tail) {
		ctx.err("%v %s", tv(v), chop1)
	}

	if v := ctx.val("rel.chop"); v == nil { // from general
		ctx.err("rel.chop")
	} else if !strings.HasPrefix(v.String(), chop1) {
		ctx.err("%v", tv(v))
	} else if !strings.HasPrefix(__string(ctx, v), chop1) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(v.String(), chop0) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(__string(ctx, v), chop0) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(v.String(), chop3) {
		ctx.err("%v", tv(v))
	} else if !strings.HasSuffix(__string(ctx, v), chop3) {
		ctx.err("%v", tv(v))
	}

	var cc1 = closure_with(ctx.Context, base.configure, base)
	var cc2 = closure_with(ctx.Context, base, base.configure)
	if s, t := ts(cc1), "{=term llvm.Config {=term configure {=term arm64-darwin {=universe …/llvm/config/arm64-darwin}}}}"; s != t {
		ctx.err("%s != %s", s, t)
	}
	if s, t := ts(cc2), "{=term configure {=term llvm.Config {=term arm64-darwin {=universe …/llvm/config/arm64-darwin}}}}"; s != t {
		ctx.err("%s != %s", s, t)
	}

	var remnant string
	var remnant_val = ctx.val("rel.remnant")
	if v := remnant_val; v == nil {
		ctx.err("%v: rel.remnant", proj)
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if v.String() != "$(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%v", tst{v})
	} else if s0 := __string(ctx,v); s0 == "" {
		ctx.err("%v", tst{v})
	} else if s1 := __string(cc1,v); s1 == "" {
		ctx.err("%v", tst{v})
	} else if s2 := __string(cc2,v); s2 == "" {
		ctx.err("%v", tst{v})
	} else if v0 := ctx.val("remnant0"); v0 == nil {
		ctx.err("%v: remnant0", proj)
	} else if __string(ctx,v0) == __string(ctx, v) {
		ctx.err("%v: %v", proj,	tv(v))
	} else if __string(ctx,v0) != __string(closure_with(ctx, proj),v) {
		note(ctx, "%v → %v", v, __string(ctx,v0))
		note(ctx, "%v → %v", v,  __string(closure_with(ctx, proj),v))
		note(ctx, "%v → %v", v,  __string(closure_with(ctx.Context, base),v))
		note(ctx, "%v → %v", v,  __string(closure_with(ctx.Context, base.configure),v))
		ctx.err("%v: %v", proj, tv(v))
	} else if v1 := ctx.val("remnant1"); v0 == nil {
		ctx.err("%v: remnant1", proj)
	} else if __string(ctx,v1) == __string(ctx, v) {
		ctx.err("%v: %v", proj, tv(v))
	} else if __string(ctx,v1) != __string(closure_with(ctx, proj),v) {
		note(ctx, "%v → %v", v, __string(ctx,v1))
		note(ctx, "%v → %v", v,  __string(closure_with(ctx, proj),v))
		note(ctx, "%v → %v", v,  __string(closure_with(ctx.Context, base),v))
		note(ctx, "%v → %v", v,  __string(closure_with(ctx.Context, base.configure),v))
		ctx.err("%v: %v", proj, tv(v))
	} else if strings.HasSuffix(s1, tail) {
		note(ctx, "%v → %v", v, s0)
		note(ctx, "%v → %v", v, s1)
		note(ctx, "%v → %v", v, s2)
		ctx.err("%v: %v", proj, tv(v))
	} else {
		remnant = s1
	}

	if remnant == "" {
		ctx.err("%v: %v", proj, tv(remnant_val))
	}

	s = "rel.remnant"
	if d := proj.resolveDef(cc1, s); d == nil {
		ctx.err("%v: %s", proj, s)
	} else if c := base.resolveDef(cc1, s); c != d {
		ctx.err("%v : %v", tst{d}, tst{c})
	} else if v := d.value; v == nil {
		ctx.err("%v", tst{d})
	} else if _, y := v.(*delegate); !y {
		ctx.err("%v", tst{v})
	} else if v.String() != "$(trim-prefix &(rel.chop),&/)" { // from general
		ctx.err("%v", tst{v})
	} else if s1 := __string(cc1,v); s1 == "" {
		ctx.err("%v", tst{v})
	} else if s2 := __string(cc2,v); s2 == "" {
		ctx.err("%v", tst{v})
	}

	s = "val1"
	if v := ctx.val(s, proj); v == nil {
		ctx.err("%s", s)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v", base)
	} else if s, t := ident(ctx, v), ident(ctx, c); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if __string(ctx, v) == c.fullname() {
		ctx.err("%v: %v", proj, base)
	} else if __string(ctx, v) != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", v, __string(ctx, v))
		note(ctx, "%v: %v/%v", v, outtmp, configuration_sm)
		ctx.err("%v: different (%v)", v, proj)
	}

	s = "val2"
	if v := ctx.val(s, proj); v == nil {
		ctx.err("%s", s)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v", base)
	} else if s, t := ident(ctx, v), ident(ctx, c); s != t {
		ctx.err("%v: %s != %s", v, s, t)
	} else if f, y := v.(*file); !y {
		ctx.err("%v", tst{v})
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v", proj, base)
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", v, f.fullname())
		note(ctx, "%v: %v/%v", v, outtmp, configuration_sm)
		ctx.err("%v: different (%v)", v, proj)
	}

	var srcinc string
	if v := ctx.val("srcinc"); v == nil {
		ctx.err("srcinc")
	} else if srcinc = __string(ctx, v); srcinc == "" {
		ctx.err("%v", tst{v})
	}

	var outinc string
	if v := ctx.val("outinc"); v == nil {
		ctx.err("outinc")
	} else if outinc = __string(ctx, v); outinc == "" {
		ctx.err("%v", tst{v})
	}

	s = "val3"
	if d := ctx.def(s+".a"); d == nil {
		ctx.err(s+".a")
	} else if v1 := d.value; v1 == nil {
		ctx.err("%v", d)
	} else if x, y := v1.(fullname); !y {
		ctx.err("%v: %v", proj.name, tst{v1})
	} else if z, y := x.Value.(*file); !y {
		ctx.err("%v: %v", proj.name, tst{x.Value})
	} else if z.dir != srcinc {
		ctx.err("%s: %s != %s", z.name, z.dir, srcinc)
	} else if d := ctx.def(s+".b"); d == nil {
		ctx.err(s+".b")
	} else if v2 := d.value; v2 == nil {
		ctx.err("%v", d)
	} else if _, y := v2.(*path); !y {
		ctx.err("%v: %v", proj.name, tst{v2})
	} else if s1, s2 := __string(ctx,v1), __string(ctx,v2); s1 != s2 {
		note(ctx, "%v: %s", proj.name, s1)
		note(ctx, "%v: %s", proj.name, s2)
		ctx.err("%v: %v != %v", proj.name, tst{v1}, tst{v2})
	}

	s = "val4"
	if d := ctx.def(s+".a"); d == nil {
		ctx.err(s+".a")
	} else if v1 := d.value; v1 == nil {
		ctx.err("%v", d)
	} else if x, y := v1.(fullname); !y {
		ctx.err("%v: %v", proj.name, tst{v1})
	} else if z, y := x.Value.(*file); !y {
		ctx.err("%v: %v", proj.name, tst{x.Value})
	} else if z.dir != outinc {
		ctx.err("%s: %s != %s", z.name, z.dir, srcinc)
	} else if d := ctx.def(s+".b"); d == nil {
		ctx.err(s+".b")
	} else if v2 := d.value; v2 == nil {
		ctx.err("%v", d)
	} else if _, y := v2.(*path); false && !y {
		ctx.err("%v: %v", proj.name, tst{v2})
	} else if s1, s2 := __string(ctx,v1), __string(ctx,v2); s1 != s2 {
		note(ctx, "%v: %s", proj.name, s1)
		note(ctx, "%v: %s", proj.name, s2)
		ctx.err("%v: %v != %v", proj.name, tst{v1}, tst{v2})
	}

	if f := proj.file(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := proj.file(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v", x, x.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := base.file(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := base.file(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", base, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := proj.tempfile(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := proj.tempfile(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := base.tempfile(closure_with(ctx.Context, proj), configuration_sm); f == nil {
		ctx.err("%v: nil %s", proj, configuration_sm)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f, base)
	} else if f.fullname() == c.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), c.fullname())
	} else if x := base.tempfile(ctx, configuration_sm); x == nil {
		ctx.err("%v: nil %s", base, configuration_sm)
	} else if f.fullname() == x.fullname() {
		ctx.err("%v: %v %v", f, f.fullname(), x.fullname())
	} else if f.fullname() != joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f, f.fullname())
		note(ctx, "%v: %v/%v", f, outtmp, configuration_sm)
		ctx.err("%v: different (%s)", f, proj.absPath)
	}

	if f := base.configuration_sm(closure_with(ctx.Context, base.configure)); f == nil {
		ctx.err("%v: %v: nil configuration", proj, base)
	} else if ident(ctx, f) != configuration_sm {
		ctx.err("%v: %v", f.name, base)
	} else if !filepath.IsAbs(f.fullname()) {
		ctx.err("%v: %v", f.name, base)
	} else if c := base.configuration_sm(ctx); c == nil {
		ctx.err("%v: %v", f.name, base)
	} else if !filepath.IsAbs(c.fullname()) {
		ctx.err("%v: %v", f.name, base)
	} else if false && f.fullname() != c.fullname() {
		note(ctx, "%v: %v", f.name, f.fullname())
		note(ctx, "%v: %v", c.name, c.fullname())
		ctx.err("%v: %v", f.name, proj)
	} else if false && f.fullname() == joinpath(outtmp, configuration_sm) {
		note(ctx, "%v: %v", f.name, f.fullname())
		note(ctx, "%v: %v/%v", f.name, outtmp, configuration_sm)
		ctx.err("%v: different", f.name)
	}

	// configure(&exec_check{unmap_uncheck_ctx{ctx},
	// 	func(_ctx Context, source string, recipe Value) {
	// 		testValidateExecRecipe(ctx, _ctx, source, recipe)
	// 	},
	// 	func(_ctx Context, line string, l int) {
	// 		testValidateExecOutput(ctx, _ctx, line, l)
	// 	},
	// })

	if s := filepath.Join(outtmp, "lib", "c++inc"); s == "" {
		ctx.err("%s", s)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else {
		lines := testCxxincConfigLines // TODO: specific lines for different OS
		for i, l := range strings.Split(lines, "\n") {
			if l != "" && strings.Count(s, l) != 1 { ctx.err("%d. %s", i, l) }
			if n := strings.Index(l, " "); n <= 0 { ctx.err("%d. %s", i, l)	} else {
				if name := strings.TrimSpace(l[:n]); name == "" {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "ALIGNOF_") && !strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "ALIGNOF_") &&  strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "SIZEOF_") && !strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "SIZEOF_") &&  strings.HasSuffix(name, "_CODE") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_LIB") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_STRUCT_") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_TYPE_") {
					ctx.err("%d. %s", i, l)
				} else if strings.HasPrefix(name, "HAVE_") && strings.HasSuffix(name, "_H") {
					ctx.err("%d. %s", i, l)
				} else {
					ctx.err("%d. %s", i, l)
				}
			}
		}
	}

	if s := filepath.Join(outtmp, "lib", "c++abi"); s == "" {
		ctx.err("%s", s)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else {
		lines := testCxxabiConfigLines // TODO: specific lines for different OS
		for i, l := range strings.Split(lines, "\n") {
			if l != "" && strings.Count(s, l) != 1 { ctx.err("%d. %s", i, l) }
		}
	}

	if s := filepath.Join(outtmp, "app"); s == "" {
		ctx.err("%s", s)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%v", e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else {
		lines := testAppConfigLines // TODO: app configs for different OS
		for i, l := range strings.Split(lines, "\n") {
			if l != "" && strings.Count(s, l) != 1 { ctx.err("%d. %s", i, l) }
		}
	}

	if f := base.configuration_sm(ctx); f == nil {
		ctx.err("%v: nil configuration", base)
	} else if s := f.fullname(); s == "" {
		ctx.err("%v", f)
	} else if i, e := os.Stat(s); e != nil {
		ctx.err("%s: %v", configuration_sm, e)
	} else if i == nil {
		ctx.err("missing %s", s)
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%s", s)
	} else if !strings.Contains(s, "FOO1 = {=yes}")  {
		ctx.err("%s", b)
	} else if true {
		debug(ctx, "%v\n%s", f.fullname(), b)
	}

	if o := base.configure.resolve(ctx, "outtmp"); o == nil {
		ctx.err("%v", base.configure)
	} else if outtmp, y := o.(*def); !y || outtmp.value == nil {
		ctx.err("%v", ts(o))
	} else if outtmp.value.String() != "&(target.tmp)/&(rel.remnant)" {
		ctx.err("%v", ts(outtmp.value))
	} else if f := base.configuration_sm(ctx); f == nil {
		ctx.err("configuration.sm")
	} else if s := filepath.Join(__string(cc1, outtmp), configuration_sm); s != f.fullname() {
		ctx.err("%v != %v", s, f.fullname())
	} else if b, e := ioutil.ReadFile(s); e != nil {
		ctx.err("%v", e)
	} else if s := string(b); s == "" {
		ctx.err("%v", outtmp.value)
	} else if strings.Count(s, "FOO = $(.self)") != 1 {
		ctx.err("%v %s", outtmp.value, s)
	}

	if false {
		if v := ctx.val("enum1", "*AsmPrinter.cpp", "LLVM_ASM_PRINTER"); v == nil {
			ctx.err("enum1")
		} else if s := v.String(); s == "" {
			ctx.err("%v", v)
		} else if strings.Count(s, "AsmPrinter.cpp") != 1 {
			ctx.err("%v", v)
		} else if strings.Count(s, "LLVM_ASM_PRINTER") != 1 {
			ctx.err("%v", v)
		} else if true {
			debug(pc(ctx,v), "%v", v)
		}
	}
}

func testLLVMConfig2(ctx *testcase) {
	testVariantTargetVars(ctx)

	if false {
		if v := ctx.val("enum1", "*AsmPrinter.cpp", "LLVM_ASM_PRINTER"); v == nil {
			ctx.err("enum1")
		} else if s := v.String(); s == "" {
			ctx.err("%v", v)
		} else if strings.Count(s, "AsmPrinter.cpp") != 1 {
			ctx.err("%v", v)
		} else if strings.Count(s, "LLVM_ASM_PRINTER") != 1 {
			ctx.err("%v", v)
		} else if true {
			debug(pc(ctx,v), "%v", v)
		}
	}
}

func testToolchainBooting(ctx *testcase) {
	testVariantTargetVars(ctx)

	proj := _project(ctx)

	// configure(&exec_check{ctx,
	// 	func(_ctx Context, source string, recipe Value) {
	// 		testValidateExecRecipe(ctx, _ctx, source, recipe)
	// 	},
	// 	func(_ctx Context, line string, l int) {
	// 		testValidateExecOutput(ctx, _ctx, line, l)
	// 	},
	// })

	if r := proj._entries(ctx, "stamp", false); r == nil {
		ctx.err("stamp")
	} else if v := _universe(ctx).run(); v != nil {
		ctx.err("%v: %v", r, v)
	}
}
