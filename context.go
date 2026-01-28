//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "path/filepath"
    rt_debug "runtime/debug"
    "runtime"
    "strings"
    "reflect"
    "regexp"
    "bufio"
    "bytes"
    "sync"
    "fmt"
    "os"
    "io"
)

const (
    vertag = "dev" // dev, alpha, beta, final
)

type property uint64

const (
    propDirtyOpts property = 1<<iota
    propErros
    propReversal
    propCache
    propUncache
)

type (
    mark_dirty     struct{ a []Value }
    act_dirt       struct{ a []Value }
    act_count_dia  struct{ t []diagtype }
    act_traversed  struct{ v Value }
    act_traverse   struct{ v Value }
    init_args      struct{ *automatic }
    get_closure_scopes struct{}
    get_args       struct{}
    get_workdir    struct{}
    get_position   struct{}
    get_project    struct{}
    get_scope      struct{}
    no_position    struct{}
    on_errors      struct{ i int }
    param_name     struct{ i int }
    is_good_with   struct{ p property ; a []any }
    is_test_case   struct{}
    is_test_mode   struct{}
    is_test_univ   struct{}
)

func _workdir(ctx Context) (s string) {
	s, _ = do(ctx, get_workdir{}).(string)
	return
}

func paramName(ctx Context, n int) (s string) {
	s, _ = do(ctx, param_name{n}).(string)
    return
}

func diagCount(ctx Context, t ...diagtype) (i int) {
    i, _ = do(ctx, act_count_dia{t}).(int)
    return
}

type Context interface {
	cast(reflect.Type) Context
	do(Context, any) any
}

func do(c Context, o any) any { return c.do(c, o) }

func truly(ctx Context, ops ...any) (_ bool) {
	for _, op := range ops {
		switch t := do(ctx, op).(type) {
		case []*valcache: return len(t) > 0
		case bool: return t
		}
	}
	return
}

func try[T any](ctx Context, op any) (_ T) {
    if ctx != nil {
		if x, y := do(ctx, op).(T); y {
			return x
		}
	}
    return
}

func cast[T Context](ctx Context) (res T) {
    if ctx != nil {
        if t := ctx.cast(reflect.TypeOf(res)); t != nil {
          return t.(T)
        }
    }
    return
}

func icast(ctx Context, t reflect.Type) (res Context) {
    if v := reflect.ValueOf(ctx); v.Type() == t {
        res = ctx
    } else if i := _inner(v); i != nil {
        res = i.(Context).cast(t)
    }
    return
}

func _inner(v reflect.Value) (i Context) {
	if x, y := v.Interface().(interface{ inner() Context }); y {
		return x.inner()
	} else if t := v.Type(); t.Kind() == reflect.Struct {
		for n := 0; false && n < v.NumField(); n++ {
			if ft := t.Field(n); ft.Anonymous {
				var fv = v.FieldByIndex(ft.Index)
				if fv.CanInterface() {
					if f := fv.Interface(); ft.Name == "Context" {
						i, _ = f.(Context)
						return
					} else if i, y = f.(Context); y {
						return
					}
				}
				if fv.Type().Kind() == reflect.Struct && fv.CanAddr() {
					if fv = fv.Addr(); fv.CanInterface() {
						if i, y = fv.Interface().(Context) ; y {
							return
						}
					}
				}
			}
		}
		if x, y := t.FieldByName("Context"); y && x.Anonymous {
			if v = v.FieldByIndex(x.Index); v.IsValid() {
				if i, y = v.Interface().(Context) ; y {
					return
				}
				if false && v.Type().Kind() == reflect.Struct && v.CanAddr() {
					if i, y = v.Addr().Interface().(Context) ; y {
						return
					}
				}
			}
		} else if false {
			for n := 0; n < v.NumField(); n++ {
				if f := t.Field(n); f.Anonymous {
					var fv = v.FieldByIndex(f.Index)
					if fv.CanInterface() {
						if i, y = fv.Interface().(Context); y {
							return
						}
					}
					if fv.Type().Kind() == reflect.Struct && fv.CanAddr() {
						if fv = fv.Addr(); fv.CanInterface() {
							if i, y = fv.Interface().(Context) ; y {
								return
							}
						}
					}
				}
			}
		}
	} else if t.Kind() == reflect.Pointer {
		i = _inner(v.Elem())
	}
	return
}

func inner(c Context) Context { return _inner(reflect.ValueOf(c)) }

func _scope(ctx Context) (s *scope) {
    s, _ = do(ctx, get_scope{}).(*scope)
    return
}

func _project(ctx Context) (p *project) {
    p, _ = do(ctx, get_project{}).(*project)
    return
}

func auto_target_value(ctx Context) (res Value) {
    if val := auto_get(ctx, "@"); val == nil {
        if false { erro(ctx, "target is nil") }
    } else if v := expand(ctx, val); v == nil {
        erro(ctx, "multiple targets: %v → %v", val, v)
    } else {
        res = scalarize(v)
    }
    return
}

func auto_target_valstr(ctx Context) (val Value, str string) {
    if val = auto_target_value(ctx); val == nil {
        if false { erro(ctx, "target is nil") }
    } else {
        str, _ = as{val}.fullname_string(ctx)
    }
    return
}

type stopframe string
type skipint int
type frames int
type callstack struct{
	num, frames, skip int
	stop string
}

var (
    callstackLine1 = regexp.MustCompile(`^(?:extbit\.io/)?((?:smart\.\(.+?\)\.)?.+?)(\(.*\))$`)
    callstackLine2 = regexp.MustCompile(`^	(.*?:\d+)(?: \+.*)?$`)
    callstackPanic = regexp.MustCompile(`^panic(\(.+\))$`)
    callstackSkips = regexp.MustCompile(`^(?:(?:testing\.tRunner`+
		`|created by testing\.(\*T)\.Run in goroutine [0-9]+`+ // skips: |erro|recovered
		`|(?:extbit\.io/)?(?:.+?)smart\.(?:do(?:_hit)?|tr(?:ace|uly|y)|(?:\*diagnostic|diagtracer)\.trace)`+
		`|runtime\.Goexit)\(.+\)|exit status [0-9]+)$`)
)
func _callstack(s string, i, j int, args ...any) (res []byte) {
    var nums []int
	var stop string
    var v = bytes.Split(rt_debug.Stack(), []byte{'\n'})

    i += 2 // skips this func
	for _, a := range args {
		switch t := a.(type) {
		case bool: if !t { return /* d */ }
		case int: nums = append(nums, t)
		case stopframe: stop, j = string(t), len(v) / 2
		case skipint: i += int(t) * 2
		case frames: if 0 < t { j += int(t) } else { j += len(v) / 2 }
		}
	}

	switch len(nums) {
	case 0: j += 1
	case 1: j += nums[0]
	case 2: j += nums[1]; i += nums[0]*2
	default: panic("too many stack nums")
	}

    var wasPanic bool
    for 0 < j && i+1 < len(v) {
        if callstackSkips.Match(v[i]) { i += 2; continue }

		sm1 := callstackLine1.FindSubmatch(v[i+0]) //extbit.io/smart.recovered(...)
		sm2 := callstackLine2.FindSubmatch(v[i+1]) //	/.../src/context.go:123 +0x456

		if sm1 != nil && sm2 != nil { n := i
			switch string(sm1[1]) {
			case stop: i, j = len(v), 0
			case "panic":
				if false { fmt.Printf("%s: %s `%s`\n", sm2[1], v[i+0], v[i+1]) }
				i, wasPanic = i+2, true
				continue
			}

			var e string
			if i, j = i+1, j-1; 0 < j && i < len(v) {
				if wasPanic { wasPanic, e = false, "	<---- panic" }
			} else {
				e = fmt.Sprintf("  (%d more)", (len(v)-n)/2)
			}

            res = append(res, sm2[1]...)
            res = append(res, []byte(":"+s+" ")...)
            res = append(res, sm1[1]...)
            res = append(res, sm1[2]...)
            res = append(res, []byte(e+"\n")...)
        } else {
			i += 1
		}
    }
    return
}

func debugSyntax(ctx Context, s string) (res bool) {
    if u := _universe(ctx); u != nil && u.ddd == s {
        for _, t := range u.debugSyntax { if res = t == s; res { break } }
    }
    return
}

func db(ctx Context, ss ...string) (res bool) {
    for _, d := range strings.Fields(_universe(ctx).ddd) {
        for _, s := range ss { if d == s { return true } }
    }
    return
}

type diagtype int
const (
    diagInfo diagtype = iota
    diagWarn
    diagError
    diagPrompt
)

type diagpoint struct {
    t diagtype
    position Position
    message string
    stack []byte // see also rt_debug.Stack()
}

type too_many_diags       struct{ int }
type too_many_erros       struct{ int }
type trace_errors         struct{ Context ; int }
type trace_evoke_loop_err struct{ Context ; Value }
type trace_evoke_loop     struct{ Context }
type trace                struct{}
type evoke_loop_null      struct{}
type evoke_loop_panic     struct{}

func (x trace_evoke_loop) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case evoke_loop_panic: return true
    }
    return x.Context.do(ctx, op)
}

func (t trace_evoke_loop_err) String() string {
    return "evoke loop: " + ts(t.Value)
}

func (t trace_errors) String() string {
    return fmt.Sprintf("trace %d errors, %v", t.int, ts(t.Context))
}

func (t too_many_diags) String() string { return fmt.Sprintf("too many diagnostics (%d)", t.int) }
func (t too_many_erros) String() string { return fmt.Sprintf("too many errors (%d)", t.int) }

// NOTE: never recover test_fail in recovered, it will break the test runner
type test_fail struct{ Context; i int }
type test_failed struct{}

func (t test_fail) Error() string { return typeof(t.Context)+": test fail" }
func (_ test_failed) Error() string { return "test failed" }

func recovered(ctx Context) {
    var ( n int ; te trace_errors )

	for e := recover(); e != nil; e = recover() {
		switch n += 1 ; t := e.(type) {
		case              bailout:
		case              failure: erro(t.Context, t.Error())
		case                Value: erro(ctx, "trace: %s", ts(t))
		case               string: erro(ctx, "trace: %s", t)
		case        runtime.Error: erro(ctx, "trace: %s", t.Error())
		case       too_many_diags: erro(ctx, "too many diagnostics (%v)", t.int)
		case       too_many_erros: erro(ctx, "too many errors (%v)", t.int)
		case trace_evoke_loop_err: erro(pc(ctx,t.Value), "evoke loop (%s)", ts(t.Value))
		case trace_errors: te = t
		case test_fail:
			if t.i += 1; t.i == 1 {
				debug(ctx, "%s: failed (%d panics)", typeof(t.Context), n, callstack{frames:-1})
			}
			if flush(ctx); true { runtime.Goexit() } else if false { panic(test_failed{}) } else { return }
		default:
			if true { panic(e) } else { note(ctx, "%s: %v", typeof(e), e) }
		}
	}

    if 0 < n {
        debug(ctx, "%s (%d panics)", typeof(te.Context), n, callstack{frames:-1})
        flush(ctx)
    }

    if false && truly(ctx, is_test_mode{}) {
        if te.Context != nil && 0 < te.int {
            panic(test_fail{te.Context, 0}) // rethrow to break the test runner
        }
    }
}

var _debug_m sync.Mutex
func debug(ctx Context, f any, a ...any) { (callstack{skip:1}).debug(ctx, f, a...) }
func (cs callstack) debug(ctx Context, f any, a ...any) {
	_debug_m.Lock(); defer _debug_m.Unlock()

	var tr = false
	var dias []*diag_point
	var args []any
	for _, a := range a {
		switch t := a.(type) {
		case trace: tr = true
		case callstack:
			if 0 < t.num     { cs.num = t.num }
			if 0 < t.skip    { cs.skip = t.skip }
			if 0 != t.frames { cs.frames = t.frames }
			if "" != t.stop  { cs.stop = t.stop }
		case *diag_point:
			dias = append(dias, t)
		default:
			args = append(args, t)
		}
	}

	var p *diagpoint
	var s = _position(ctx).String()+": "
	switch t := f.(type) {
	case *diag_point:
		if t.f != "" {
			p, _ = do(ctx, diag_point{diagPrompt, s+t.f+"\n", t.a}).(*diagpoint)
		}
	case []*diag_point:
		for _, t := range t {
			p, _ = do(ctx, diag_point{diagPrompt, s+t.f+"\n", t.a}).(*diagpoint)
		}
	case string:
		for _, t := range strings.Split(t, "\n") { if t == "" { continue }
			p, _ = do(ctx, diag_point{diagPrompt, s+t+"\n", args}).(*diagpoint)
		}
	default:
		p, _ = do(ctx, diag_point{diagPrompt, s+typeof(t)+": %v\n", args}).(*diagpoint)
	}
	for _, d := range dias {
		p, _ = do(ctx, diag_point{diagPrompt, s+d.f+"\n", d.a}).(*diagpoint)
	}

	if args = []any{}; p == nil { return }
	if 0 < cs.num     { args = append(args, cs.num) }
	if 0 < cs.skip    { args = append(args, skipint(cs.skip)) }
	if 0 != cs.frames { args = append(args, frames(cs.frames)) }
	if "" != cs.stop  { args = append(args, stopframe(cs.stop)) }
	if p.stack = _callstack("info:", 5, 0, args...); true { flush(ctx) }
	if tr {
		if truly(ctx, is_test_mode{}) {
			panic(test_fail{ctx, 0})
		} else {
			panic(trace_errors{ctx, diagCount(ctx, diagError)})
		}
	}
}

var _trace_m sync.Mutex
func trace_err(ctx Context, a ...any) {
	_trace_m.Lock()
	defer _trace_m.Unlock()
	defer recovered(ctx)
	if a = append(a, frames(-1)); truly(ctx, is_test_mode{}) {
		note(ctx, "%s: failed", typeof(ctx))
	} else {
		note(ctx, "%d errors", diagCount(ctx, diagError))
	}
	_callstack("", 5, 0, a...)
	flush(ctx)
	runtime.Goexit()
}

const diagnostic_limit = 10_000
var   diagnostic_limit_erros = 520
var   diagnostic_limit_bytes = 1_000_000

func _diagnostic(c Context) *diagnostic { return cast[*diagnostic](c) }
func _f(f string, a ...any) *diag_point { return &diag_point{0, f, a} }

type diag_struct struct{ t diagtype; f string; a []any }
type diag_trace diag_struct
type diag_point diag_struct
type diag_flush struct{}
type diagnostic struct{
    Context
    sync.Mutex
    points []*diagpoint
    erros int // number of flushed erros
    flushed int // in bytes
}
func (d *diagnostic) aquire() func() { d.Lock(); return d.Unlock }
func (d *diagnostic) cast(t reflect.Type) Context { return icast(d, t) }
func (d *diagnostic) inner() Context { return d.Context }
func (d *diagnostic) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case property: if t&propErros != 0 { return d.erros }
    case diag_flush   : return d.flush(ctx)
	case diag_point   : return d.point(ctx, t.t, t.f, t.a...)
    case act_count_dia: return d.count(t.t...)
    }
    if d.Context == nil { return }
    return d.Context.do(ctx, op)
}
func (d *diagnostic) add(p *diagpoint) *diagpoint {
    defer d.aquire()()
    if i := len(d.points); diagnostic_limit < i {
        panic(too_many_diags{i})
    }
	d.points = append(d.points, p)
	return p
}
func (d *diagnostic) point(ctx Context, dt diagtype, f string, args ...any) *diagpoint {
	if dt != diagPrompt { f = strings.TrimSpace(f) }
	return d.add(&diagpoint{dt, _position(ctx), fmt.Sprintf(f, args...), nil})
}
func (d *diagnostic) count(dt ...diagtype) (errs int) {
	defer d.aquire()()
	for _, d := range d.points {
		for _, t := range dt {
			if d.t == t { errs += 1 ; break }
		}
	}
	return
}
func (d *diagnostic) flush(ctx Context) (errs int) {
    const count_bytes = false

	defer func() { if d.erros += errs ; errs > 0 { do(ctx, on_errors{errs}) }} ()

	print := func(p *diagpoint, pend bool) (_ bool) {
		defer func() {
			if x, y := diagnostic_limit_erros, d.erros; 0 < x && x < y {
				if false { d.erros = 0 } // reset to avoid causing next panics
				panic(too_many_erros{y})
			}
			if x, y := diagnostic_limit_bytes, d.flushed; 0 < x && x < y && false {
				if false { d.flushed = 0 } // reset to avoid causing next panics
				panic(too_many_diags{y})
			}
		} ()

        pos, msg := p.position.String(), p.message

        if count_bytes {
            d.flushed += len(pos) + len(msg)
        } else {
            d.flushed += 1
        }

        switch p.t {
        case diagInfo: fmt.Fprintf(stderr, "%v:info: %s\n", pos, msg)
        case diagWarn: fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
        case diagPrompt:
            if msg != "" { fmt.Fprintf(stderr, "%s", msg) }
            if pend && !strings.HasSuffix(msg, "\n") { return true }
        case diagError:
            if errs += 1 ; p.stack == nil {
                fmt.Fprintf(stderr, "%v:error: %s\n", pos, msg)
            } else {
                fmt.Fprintf(stderr, "%v: %s\n", pos, msg)
            }
        }

        if p.stack != nil {
            fmt.Fprintf(stderr, "%s\n", bytes.TrimSpace(p.stack))
            if count_bytes {
                d.flushed += len(p.stack)
            } else {
                d.flushed += 1 + bytes.Count(p.stack, []byte("\n"))
            }
        }
        return
    }

	d.Lock(); defer d.Unlock()
	for 0 < len(d.points) {
		var point = d.points[0]
		d.points = d.points[1:]
		if print(point, true); 16 < errs {
			fmt.Fprintf(stderr, "%v: too many errors (%d)\n", _position(ctx), errs)
		}
	}
    return
}

func flush(ctx Context) (i int) { i, _ = do(ctx, diag_flush{}).(int); return }

func diag(ctx Context, dt diagtype, f string, a ...any) (res *diagpoint) {
    res, _ = do(ctx, diag_point{dt, f, a}).(*diagpoint)
    return
}

func prompt(c Context, f string, a ...any) *diagpoint { return diag(c,   diagPrompt, f, a...) }
func info(ctx Context, f string, a ...any) *diagpoint { return diag(ctx, diagInfo,   f, a...) }
func warn(ctx Context, f string, a ...any) *diagpoint { return diag(ctx, diagWarn,   f, a...) }
func erro(ctx Context, f string, a ...any) *diagpoint { return diag(ctx, diagError,  f, a...) }
func note(ctx Context, f string, a ...any) *diagpoint {
    if !strings.HasSuffix(f, "\n") { f += "\n" }
    return prompt(ctx, _position(ctx).String()+": "+f, a...)
}

func infostack(ctx Context, n int, a ...any) *diagpoint { return diagstack(ctx, n, diagInfo  , a...) }
func warnstack(ctx Context, n int, a ...any) *diagpoint { return diagstack(ctx, n, diagWarn  , a...) }
func errostack(ctx Context, n int, a ...any) *diagpoint { return diagstack(ctx, n, diagError , a...) }
func notestack(ctx Context, n int, a ...any) *diagpoint {
    if true {
        var f string
        if len(a) > 0 {
            if x, y := a[0].(string); y {
                f, a = x, a[1:]
            }
        }
        a = append([]any{"%v: "+f+"\n", _position(ctx)}, a...)
    }
    return diagstack(ctx, n, diagPrompt, a...)
}
func diagstack(ctx Context, n int, dt diagtype, a ...any) (point *diagpoint) {
    var s string

    if 0 < len(a) {
        if x, y := a[0].(string); y {
            s, a = x, a[1:] // separate the format string and args
            if !strings.HasSuffix(s, "\n") { s += "\n" }
        }
    }

    point = diag(ctx, dt, s, a...)
    dt = diagInfo

    var proj = _project(ctx)
    var p = _position(ctx)
    for c := inner(ctx); c != nil && 0 < n && p.valid(); c = inner(c) {
        var pos = _position(c)
        if !pos.valid() || pos.same(&p) { continue }

        n -= 1
        p = pos

        if proj == nil {
            s = "<nil>"
        } else {
            s = proj.name
        }

        proj = _project(c)

        if e := _entry(c); e != nil {
            if t, _, _ := entryIndicator(c, e); t == "" {
                s += ": " + ident(ctx, e)
            } else {
                s += ": " + t
            }
        }

        if true { s += " ; " + ts(c) }

        point = diag(c, dt, s)
    }
    return
}

func _position(ctx Context) (_ Position) {
    if x, y := do(ctx, get_position{}).(Position); y /* && x.Filename != "" */ {
        return x
    } else if true {
        return
    } else {
        panic(no_position{})
    }
}

func walkSmartBaseDirs(ctx Context, cwd string, vis func(string) bool) (s string) {
    for s = cwd ; s != "" ; {
        var f = _stat(ctx, ".smart", stat_dir{s})
        if f != nil && f.info.IsDir() && !vis(s) { break }
        if up := filepath.Dir(s); up == s { break } else { s = up }
    }
    if s == "" { s = cwd }
    return
}

// baseTmpPath is the base tmp path initialized only once.
var baseTmpPath string

func joinTmpPath(ctx Context, base, rel string) string {
    if baseTmpPath == "" {
        var s = walkSmartBaseDirs(ctx, base, func(d string) bool {
            return false // return the first found
        })
        if s == "" {
            // FIXME: Windows system temporary path.
            s = filepath.Join("/", "tmp")
        }
        baseTmpPath = s
    }
    if s := filepath.Dir(rel); s != "" {
        if strings.HasSuffix(base, s) {
            // In case like '/foo/bar/a/b/c/x'+'a/b/c/x', we set
            // rel to 'x' to produce 'foo/bar/.smart/tmp/a/b/c/x'.
            rel = filepath.Base(rel)
        } else if t, _ := filepath.Rel(baseTmpPath, base); strings.HasPrefix(t, ".smart"+pathSep) {
            // In case like '/foo/bar/.smart/a/b/x'+'a/e/f/x', we set
            // base to '/foo/bar/.smart' to produce 'foo/bar/.smart/tmp/a/e/f/x'.
            v1 := strings.Split(t, pathSep)
            v2 := strings.Split(s, pathSep)
            for i := len(v1)-1; i >= 0; i -= 1 {
                if v1[i] == v2[0] {
                    base = filepath.Join(v1[i-1:]...)
                    break
                }
            }
        }
    }
    if s, err := filepath.Rel(baseTmpPath, filepath.Join(base, rel)); err == nil {
        rel = s
    }
    if s := ".smart"+pathSep; strings.HasPrefix(rel, s) { // .smart/
        rel = strings.TrimPrefix(rel, s)
        if s = "modules"+pathSep; strings.HasPrefix(rel, s) { // modules/
            rel = strings.TrimPrefix(rel, s)
        }
    }
    rel = strings.Replace(rel, "..", "_", -1)
    if strings.HasPrefix(rel, "tmp"+pathSep) {
        return filepath.Join(baseTmpPath, ".smart", rel)
    }
    return filepath.Join(baseTmpPath, ".smart", "tmp", rel)
}

func positionForDir(dir string) (pos Position) {
    if strings.HasSuffix(dir, mainFileName) || strings.HasSuffix(dir, deprFileName) {
        pos.Filename = dir
    } else if _, e := os.Stat(filepath.Join(dir, mainFileName)); e == nil {
        pos.Filename = filepath.Join(dir, mainFileName)
    } else if _, e := os.Stat(filepath.Join(dir, deprFileName)); e == nil {
        pos.Filename = filepath.Join(dir, deprFileName)
    } else {
        pos.Filename = dir
    }
    pos.Line = 1
    return
}

func loadSeachPaths(ctx *universe, s string) (paths []string) {
    var f, err = os.Open(filepath.Join(s, ".search"))
    if err != nil { return }

    defer f.Close()

    for r := bufio.NewReader(f); err == nil; {
        var fi os.FileInfo
        var line string
        if line, err = r.ReadString('\n'); err != nil {
            if err != io.EOF {
                fmt.Fprintf(stderr, "%v", err)
            } else {
                err = nil
                if line == "" { break }
            }
        } else {
            line = strings.TrimSpace(line)
        }

        if strings.HasPrefix(line, "#") { continue }

        if filepath.IsAbs(line) {
            line = filepath.Clean(line)
        } else {
            line = filepath.Clean(filepath.Join(s, line))
        }

        if fi, err = os.Stat(line); err == nil && fi.IsDir() {
            ctx.paths = append(ctx.paths, line)
        }
    }

    if err != nil {
        fmt.Fprintf(stderr, "%v: %v", f, err)
    }
    return
}

func Main() {
    var ctx = new_universe()
    var modulesPaths, packagePaths searchlist

    walkSmartBaseDirs(ctx, ctx.workdir, func(s string) bool {
        if baseTmpPath == "" { baseTmpPath = s }
        packagePaths = append(packagePaths, filepath.Join(s, ".smart", "packages"))
        modulesPaths = append(modulesPaths, filepath.Join(s, ".smart", "modules"))
        return true
    })

    userLib := filepath.Join(ctx.prefix, "user", "lib", "smart")
    packagePaths = append(packagePaths, filepath.Join(userLib, "packages"))
    modulesPaths = append(modulesPaths, filepath.Join(userLib, "modules"))

    // make sure that .smart dirs have higher priority.
    ctx.paths = append(modulesPaths, ctx.paths...)

    for _, s := range modulesPaths { loadSeachPaths(ctx, s) }

    ctx.load(ctx)

    if ctx.flush(ctx) > 0 {
        prompt(ctx, "loading work got %d errors\n", ctx.erros)
    } else if ctx.help {
        do_helpscreen(ctx)
    } else if ctx.printFlags {
        print_flag_trace(ctx)
    } else if ctx.printConfig {
        print_configuration(ctx)
    } else if numUpdatedPlugins > 0 { // see buildPlugin
        prompt(ctx, "plugins updated, please relaunch.\n")
    } else if result := ctx.run(); ctx.flush(ctx) > 0 {
        prompt(ctx, "run work got %d errors\n", ctx.erros)
    } else if result != nil {
        for i, v := range result {
            if s := ""; v == nil {
                s = "<nil>"
            } else if s = strings.TrimSpace(__string(ctx, v)); s == "" {
                continue
            } else if i == 0 {
                fmt.Fprintf(stderr, "%s", s)
            } else {
                fmt.Fprintf(stderr, ", %s", s)
            }
        }
        fmt.Fprintf(stderr, "\n")
    }
}
