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
    "runtime/pprof"
    "strings"
    "reflect"
    "regexp"
    "bufio"
    "bytes"
    "sync"
    "time"
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
	get_fatpos     struct{ p Pos }
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
        str, _ = as_fullname_string(ctx, val)
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
    if u := _universe(ctx); u != nil {
        for _, t := range u.debugSyntax { if res = t == s; res { break } }
    }
    return
}

const (
    diagInfo diagtype = iota
    diagWarn
    diagError
    diagPrompt
)

type diagtype int
type diagpoint struct {
    t diagtype
    position Position
    message string
	panic any
    stack []byte // see also rt_debug.Stack()
}

type too_many_diags       struct{ int }
type too_many_erros       struct{ int }
type trace_errors         struct{ Context ; int }
type trace_evoke_loop_err struct{ Context ; Value }
type trace_evoke_loop     struct{ Context }
type trace_val            struct{ int ; val Value }
type trace_ctx            struct{ int }
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
func debug(ctx Context, f any, a ...any) {
	_debug_m.Lock(); defer _debug_m.Unlock()

	var tr = false
	var trCtx int
	var trVal int
	var cs callstack
	var dias []*diag_point
	var args []any
	for _, a := range a {
		switch t := a.(type) {
		case trace: tr = true
		case trace_ctx: trCtx = t.int
		case trace_val: trVal = t.int
		case callstack:
			if 0 < t.num     { cs.num = t.num }
			if 0 < t.skip    { cs.skip = t.skip }
			if 0 != t.frames { cs.frames = t.frames }
			if "" != t.stop  { cs.stop = t.stop }
		case []*diag_point:
			dias = append(dias, t...)
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

	if n := trCtx; n > 0 {
		var pos = _position(ctx)
		for c := inner(ctx); c != nil && 0 < n && pos.valid(); c = inner(c) {
			var _pos = _position(c)
			if !_pos.valid() || _pos.same(&pos) { continue }

			n -= 1
			pos = _pos

			proj := _project(c)
			s := pos.String() + ": " + typeof(c) + ": "
			if proj == nil {
				s += "<nil>"
			} else {
				s += proj.name
			}

			if e := _entry(c); e != nil {
				if t, _, _ := entryIndicator(c, e); t == "" {
					s += ": " + ident(ctx, e)
				} else {
					s += ": " + t
				}
			}

			if false { s += " ; " + ts(c) }

			p, _ = do(c, diag_point{diagPrompt, s+"\n", nil}).(*diagpoint)
		}
	}

	if n := trVal; n > 0 {
		p, _ = do(ctx, diag_point{diagError, "TODO: trace value\n", nil}).(*diagpoint)
	}

	for _, d := range dias {
		p, _ = do(ctx, diag_point{diagPrompt, s+d.f+"\n", d.a}).(*diagpoint)
	}

	if args = []any{}; p == nil { return }
	if cs.num > 0     { args = append(args, cs.num) }
	if cs.skip > 0    { args = append(args, skipint(cs.skip)) }
	if cs.frames != 0 { args = append(args, frames(cs.frames)) }
	if cs.stop != ""  { args = append(args, stopframe(cs.stop)) }
	if p.stack = _callstack("info:", 5, 0, args...); true { flush(ctx) }
	if tr {
		if truly(ctx, is_test_mode{}) {
			p.panic = test_fail{ctx, 0}
		} else {
			p.panic = trace_errors{ctx, diagCount(ctx, diagError)}
		}
		panic(p.panic)
	}
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
	return d.add(&diagpoint{dt, _position(ctx), fmt.Sprintf(f, args...), nil, nil})
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

		if p.panic != nil {
			msg += fmt.Sprintf(": %v", p.panic)
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
	switch x := do(ctx, get_position{}).(type) {
	case Position:
		// Fast path: Context natively provided a fat Position (e.g., from the universe or *xloc)
		if x.valid() { return x }
	case Pos:
		// Resolution path: Context provided a compact AST Pos. 
		// We dynamically resolve it into a fat Position using our bridge!
		if x.IsValid() {
			if fat, ok := do(ctx, get_fatpos{x}).(Position); ok {
				return fat
			}
		}
	}
	
	// Fallback (matches your `else if true { return }` logic)
	return 
}

func _pos(ctx Context) Pos {
	// 1. Context/Evaluation Path: Extract a compact Pos from the runtime context stack!
	// This allows posctx and evocation to inject specific AST node positions.
	switch x := do(ctx, get_position{}).(type) {
	case Pos:
		if x.IsValid() { return x }
	case positioner:
		if p := x.Pos(); p.IsValid() { return p }
	}

	// 2. Parse-Time Fallback: Extract the exact compact integer
	// offset from the active parser if no context explicitly overrides it.
	if p, ok := do(ctx, get_parser{}).(*parser); ok && p != nil {
		if p.pos.IsValid() { return p.pos }
	}

	return 0 // 0 represents NoPos
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


var searchPaths searchlist
var launchTime = time.Now()
var workBaseDir = func () string {
    if s, e := os.Getwd(); e == nil { return s } else { panic(e) }
} ()

type searchlist []string
func (sl *searchlist) String() string { return fmt.Sprint(*sl) }
func (sl *searchlist) set(s string) error {
    *sl = append(*sl, strings.Split(s, ",")...)
    return nil
}
func (sl *searchlist) has(s string) (_ bool) {
    for _, p := range *sl { if p == s { return true }}
    return
}

type hooks struct {
    assert func(Context, Value, bool) bool
    debug func(Context, string, []Value)
    error func(Context, string, []Value)
}

type packagetype uint8

const (
    packageUnknown packagetype = iota
    packageSmart  // smart package
    packageConfig // pkgconfig
)

type packageinfo struct {
    *project
    t packagetype // smart, pkgconfig, cmake, etc.
}

// ResolvePosition converts a compact AST 'Pos' back into a human-readable 'Position'
func ResolvePosition(ctx Context, v Value) Position {
	if v == nil {
		return Position{}
	}
	
	// 1. Runtime / Synthetic Position Escape Hatch
	// If the value is explicitly wrapped in an external location (*xloc), use its fat struct!
	if l, ok := v.(*xloc); ok {
		return l.pos 
	}
	
	// (Optional but robust) Check if any other node implements the fat Position interface
	if p, ok := v.(interface{ Position() Position }); ok {
		if pos := p.Position(); pos.valid() {
			return pos
		}
	}
	
	// 2. Parse-Time Position Resolution
	pos := v.Pos() // The compact integer
	if !pos.IsValid() {
		return Position{} // NoPos
	}
	
	// Safely retrieve the fset from the universe
	if u := _universe(ctx); u != nil && u.fset != nil {
		return u.fset.Position(pos)
	}
	
	return Position{}
}

func _universe(c Context) *universe { return cast[*universe](c) }

type universe struct {
    diagnostic
    commandline

    *scope
    *globe

    fset *fileset

    workdir string
    prefix  string // FIXME: prefix for distribution
    paths   searchlist
    packages  map[string]packageinfo
    statcache map[string]*filebase // file.fullname() -> File
    statmutex sync.Mutex

    hooks hooks

    expand_n int32
}
func (ctx *universe) String() string { return "universe" }
func (ctx *universe) inner() Context { return &ctx.diagnostic }
func (ctx *universe) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}
func (ctx *universe) _position() (p Position) {
    if ctx.globe != nil && ctx.globe.main != nil && ctx.fset != nil {
		p = ctx.fset.Position(ctx.globe.main.pos)
    } else {
		p.Filename, p.Line, p.Column = _workdir(ctx), 0, 0
	}
    return
}
func (ctx *universe) ts(t string) string {
    var s = ts(ctx.Context)
    if  s == "{}"  {
        s, _ = filepath.Rel(workBaseDir, ctx.workdir)
        s = bases(3, s, "testdata", true)
        if s == "." || s == "" { return "{="+t+"}" }
    }
    return "{="+t+" "+s+"}"
}
func (ctx *universe) trimSpecPath(c Context, spec string) string {
    spec = strings.ReplaceAll(spec, "../", "")
    for _, s := range ctx.paths {
        if s += pathSep; strings.HasPrefix(spec, s) {
            spec = strings.TrimPrefix(spec, s)
            break
        }
    }
    if s := ctx.workdir+pathSep; strings.HasPrefix(spec, s) {
        spec = strings.TrimPrefix(spec, s)
    }
    return spec
}
func (ctx *universe) do(_ctx Context, op any) (res any) {
    switch t := op.(type) {
    case on_errors:
        if ctx.panicFailureOnFlushedErrors && truly(_ctx, is_test_mode{}) {
            if 0 < t.i { panic(_failure(ctx, "got %d errors", t.i)) }
            res = true
        }
        return

	case get_fatpos:
		var p Position
		if ctx.fset != nil && t.p.IsValid() { p = ctx.fset.Position(t.p) } // FIX: Use the receiver 'ctx'
		return p

    case get_position:
        p := Position{}
        p.Filename = ctx.workdir
        return

    case get_workdir: return ctx.workdir
    case get_scope: if ctx.scope != nil { return ctx.scope }
    case get_project: if ctx.globe != nil { return ctx.globe.main }
    case no_exec: if ctx.noExec { return ctx.noExec }
	case is_test_mode: if ctx.testMode { return true }
    case is_test_univ: return ctx.testMode
    }
    return ctx.diagnostic.do(_ctx, op)
}

type no_exec struct{}
type commandline struct {
    help            bool `h,help`

    debug           bool `d,db,debug`
    debugErrors     bool `de,dberro,debug-errors`
    debugWarns      bool `dw,dbwarn,debug-warns`
    debugInfos      bool `di,dbinfo,debug-infos`
    debugPrompt     bool `dp,dbprom,debug-prompt`
    debugSyntax []string `ds,dbsyntax,debug-syntax`

    autoProfs       bool `ap,autoprof,auto-profiles,auto-profile`
    cpuProf         string `cpuprof,cpu-profile`
    memProf         string `memprof,memory-profile`

    printConfig     bool `opts,print-options,printoptions`
    printFlags      bool `flags,print-flags,printflags`

    buildPlugins    bool `bp,bup,build-plugins,buildplugins`

    silentOptionalArrow bool

    verbose         bool `v,verb,verbose`
    verboseBreaks   bool `vb,vbrk,verbose-breaks`
    verboseChecks   bool `vc,vchk,verbose-checks`
    verboseImport   bool `vi,vimp,verbose-import`
    verboseParse    bool `vp,vpar,verbose-parsing`
    verboseUsing    bool `vu,vuse,verbose-using`
    verboseExecFlags bool `vxf,verbose-exec-flag`

    allowClosureFilemap bool `cf,closure-filemap,closure-files`

    cleanDotCache   bool `clcac,clean-cache,clear-cache;rmc,rm-cache`
    cleanDotDeps    bool `cldep,clean-deps,clear-deps;rmd,rm-deps`
    cleanDotGrep    bool `clgrp,clean-grep,clear-grep;rmg,rm-grep`
    cleanTmpDirs    bool `cltmp,clean-temp,clear-temp;rmt,rm-temp`

    checkLoadGraph  bool `ckld,check-loads`

    reconfigure     bool `rc,reconf,reconfig,reconfigure`

    saveGrepSource  bool `savgs,save-grep-source`

    noRun           bool `nor,no-run`
    noExec          bool `nox,ne,no-exec,no-execute`  // optionNoExec
    noDeps          bool `nod,no-deps`
    noGrep          bool `nog,no-grep`
    noDepsGrep      bool `nodg,ngd,no-deps-grep,no-grep-deps`
    noImportFiles   bool `noif,no-import-files`

    parallel        bool `par,para,parallel`

    testMode        bool `test,test-mode`
    fastMode        bool `fast,fast-mode`
    errorUncache    bool `eu,error-uncache,error-no-cache`
    panicFailureOnFlushedErrors bool `foe,fail-on-errors`

    traceLaunch     bool `tl,trace-launch`
    traceParsing    bool `tp,trace-parse`
    traceExecutor   bool `te,trace-executor`
    traceExec       bool `tx,trace-exec`
    traceEntering   bool `ti,trace-entering`
    traceConfig     bool `tc,trace-config`

    slow time.Duration `slow` // time.Millisecond
}

func _commandline() commandline { return commandline{
    debugPrompt: true,
    debugErrors: true,
    debugWarns:  true,
    debugInfos:  true,

    fastMode: true,
    parallel: false, // FIXME: program.traverse not working in parallel

    panicFailureOnFlushedErrors: true,
    silentOptionalArrow: false,

    slow: 2999 * time.Millisecond,
}}

func new_universe(ii ...any) (ctx *universe) {
    ctx = &universe{}
    ctx.paths = searchPaths
    ctx.workdir = workBaseDir
    ctx.fset = _fileset()
    ctx.statcache = make(map[string]*filebase)
    ctx.scope = newscope(nil, nil, `universe`)

    var cl = true
    for _, i := range ii {
        switch t := i.(type) {
        case  commandline: ctx.commandline, cl =  t, false
        case *commandline: ctx.commandline, cl = *t, false
        case *hooks: ctx.hooks = *t
        case  hooks: ctx.hooks =  t
        }
    }
    if cl { ctx.commandline = _commandline() }

    var bin  = ease(ctx, os.Args[0])
    var args = ease(ctx, os.Args[1:])
    ctx.scope.def(ctx, defVoid, "SMART.ARGS", args)
    ctx.scope.def(ctx, defVoid, "SMART.BIN",  bin)
    ctx.scope.def(ctx, defVoid, "SMART",      bin)

    for name, f := range builtins {
        if _, alt := ctx.scope.builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
    }

    var pos Pos = 0 //ctx._position()
	// one of darwin, freebsd, linux, and so on.
	var os Value = ease(ctx, []Value{_word(pos, runtime.GOOS)})

    ctx.globe = &globe{
        scope: newscope(ctx.scope, nil, `globe`),
        flagEntries: make(map[string][]entry),
        loaded: make(map[string]*project),
        args: make(map[Value][]Value),
    }

    // FIXME: ctx.scope.scopename(ctx, ".GLOBE", ctx.globe.Scope)
    ctx.globe.os    = ctx.globe.def(ctx, defVoid, ".os",    os)
    ctx.globe.goals = ctx.globe.def(ctx, defVoid, ".goals", _none(pos))
    ctx.globe.mode  = ctx.globe.def(ctx, defVoid, ".mode",  _null(pos))
    return
}

type filestub struct {
    dir      string   // full directory where the file was or should be found
    sub      string   // matched sub path (see project.search), may be Dir (absolete path)
    name     string   // constant represented name (e.g. relative filename)
    filemap *filemap  // matched pattern (see 'files' directive)
    other   *filestub // pointed to another stub (in a different project) of the same file
}
func (p *filestub) subname() string {
    if isAbsOrRel(p.sub) {
        return p.name
    } else {
        return filepath.Join(p.sub, p.name)
    }
}

type filebase struct {
    stub filestub    // cycled-list of file stubs of different projects
    info os.FileInfo // file info if exists
    _updated bool // true if this file has been updated by a program
    _updatedDeps []Value // any updated deps
    _travin int
    _traved int
    _dirty  int
}
func (p *filebase) exists() bool { return p != nil && p.info != nil }

type stat_dir struct { string }
type stat_sub struct { string }
type stat_nonexist struct { bool }
type stat_fileinfo struct{ os.FileInfo }

func _stat(ctx Context, a0 any, aa ...any) (_ *file) {
	var name = __string(ctx, a0)
	var sub, dir string
	var nonexist bool
	var fileInfo os.FileInfo

	for _, a := range aa {
		switch t := a.(type) {
		case *project: dir = t.absPath
		case stat_dir: dir = t.string
		case stat_sub: sub = t.string
		case stat_fileinfo: fileInfo = t.FileInfo
		case stat_nonexist: nonexist = t.bool
		default: debug(ctx, "invalid stat arg: %v", ts(a), trace{})
		}
	}

	var u = _universe(ctx)
	var fullname string

	// 1. Trim slashes and clean paths
	if dir != "" { dir = filepath.Clean(dir) }
	if sub != "" { sub = filepath.Clean(sub) }

	// 2. Cleaned Path Resolution Logic (Dead code removed)
	if filepath.IsAbs(name) {
		fullname = name
		if dir != "" && strings.HasPrefix(fullname, dir+pathSep) {
			if tail := fullname[len(dir)+1:]; sub == "" {
				name = tail 
			} else if strings.HasPrefix(fullname, sub+pathSep) {
				name = tail[len(sub)+1:]
			}
		} else {
			dir = "" // Conflict resolution: Absolute name overrides directory
		}
	} else if filepath.IsAbs(sub) {
		if fullname = filepath.Join(sub, name); dir == "" {
			dir = sub
			sub = ""
		} else if sub == dir {
			sub = ""
		} else if strings.HasPrefix(sub, dir) {
			sub = strings.TrimPrefix(sub, dir)
			sub = strings.TrimPrefix(sub, pathSep)
			sub = filepath.Clean(sub)
		} else {
			debug(ctx,
				_f("conflicted sub/dir: %s", fullname),
				_f("sub=%s", sub),
				_f("dir=%s", dir),
				callstack{num:16}, trace{})
		}
	} else if filepath.IsAbs(dir) {
		fullname = filepath.Join(dir, sub, name)
	} else {
		dir = filepath.Join(_workdir(ctx), dir)
		fullname = filepath.Join(dir, sub, name)
	}

	var cleanFullname = filepath.Clean(fullname)
	
	// 3. First Pass: Fast lock to retrieve or initialize the cache entry shell
	u.statmutex.Lock()
	base, exists := u.statcache[cleanFullname]
	if !exists {
		// Initialize the shell of the filebase. We will stat it outside the lock.
		base = &filebase{filestub{dir, sub, name, nil, nil}, nil, false, nil, 0, 0, 0}
		base.stub.other = &base.stub
		u.statcache[cleanFullname] = base
	}
	u.statmutex.Unlock() // DROP THE GLOBAL LOCK BEFORE HITTING THE DISK!

	// 4. Heavy I/O: os.Stat executes concurrently without blocking the universe
	if base.info == nil && fileInfo == nil { fileInfo, _ = os.Stat(fullname) }

	// 5. Second Pass: Re-acquire lock to safely update the shared base and stubs
	u.statmutex.Lock(); defer u.statmutex.Unlock()

	// Update the file metadata if we just fetched it
	if fileInfo != nil && base.info == nil { base.info = fileInfo }

	// Bail out if the file doesn't exist and we aren't explicitly allowing non-existent files
	if base.info == nil && !nonexist { return nil }

	var head = &base.stub
	var stub *filestub

	// Sanity assertions
	if enable_assertions {
		for stub = head; stub != nil; stub = stub.other {
			if s1, s2 := fullname, filepath.Join(stub.dir, stub.sub, stub.name); s1 != s2 {
				debug(ctx,
					_f("fullname '%s' conflicted", fullname),
					_f("panic: (%s, %s, %s) %s", stub.dir, stub.sub, stub.name, s1),
					_f("panic: (%s, %s, %s) %s", dir, sub, name, s2),
					callstack{num:16}, trace{})
			}
			if stub.other == head { break }
		}
	}

	// Check for an existing stub that matches our current path parameters exactly
	for stub = head; stub != nil; stub = stub.other {
		if stub.dir == dir && stub.sub == sub && stub.name == name {
			return &file{valbase{_pos(ctx)}, base, stub}
		}
		if stub.other == head { break }
	}

	// If no matching stub was found, link a new one into the circular list
	stub = &filestub{dir, sub, name, nil, head.other}; head.other = stub
	return &file{valbase{_pos(ctx)}, base, stub}
}

func AddPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil {
            break
        }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
           searchPaths = append(searchPaths, s)
        }
    }
    return
}

func (ctx *universe) AddPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil { break }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
            ctx.paths = append(ctx.paths, s)
        } else {
            return fmt.Errorf("path '%s' is not dir", s)
        }
    }
    return nil
}

func cpu_profile(ctx Context, name string, heap ...bool) (stop func()) {
    var fn string
    if filepath.IsAbs(name) { fn = name } else
    if m := _universe(ctx).globe.main; m == nil {} else
    if f := m.tempfile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        debug(ctx, "%T: %v", e, e, trace{})
    } else if e = pprof.StartCPUProfile(f); e != nil {
        debug(ctx, "%T: %v", e, e, trace{})
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        if heap != nil && heap[0] { runtime.GC() // update memory statistics
            if e = pprof.WriteHeapProfile(f); e != nil {
                debug(ctx, "WriteHeapProfile: %v", e, trace{})
            }
        }
        f.Close()
    }}
}

func heap_profile(ctx Context, name string) (stop func()) {
    var fn string
    if filepath.IsAbs(name) { fn = name } else
    if m := _universe(ctx).globe.main; m == nil {} else
    if f := m.tempfile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        debug(ctx, "%T: %v", e, e, trace{})
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        runtime.GC() // update memory statistics
        if e = pprof.WriteHeapProfile(f); e != nil {
            debug(ctx, "WriteHeapProfile: %v", e, trace{})
        }
        f.Close()
    }}
}

func updateGoal(ctx Context, goal Value, args []Value) (result []Value) {
    switch g := goal.(type) {
    case *rule:
        var y bool
        if result, y = execute_entry(ctx, g, args...); !y {
            debug(ctx, "update '%v' failed", g, trace{})
        }
    default:
        debug(ctx, "not an entry: %v", ts(goal), trace{})
    }
    return
}

func (l ul) parse_args(base string, a ...string) {
    var args []Value

	if s := strings.Join(a, " "); s != "" {
		if v := l.text(l.universe, base, s); v != nil {
			args = parseOpts(l.universe, &l.commandline, merge(v)...)
		}
	}

    if v := l.fastMode; v { // Turn off many things for fast mode:
        //l.noImportFiles = v
        l.noDepsGrep = v
        l.noDeps = v
        l.noGrep = v
    }

    var mode = new(word)

    for _, target := range args {
        switch t := target.(type) {
        case *pair: l.globe.pairs = append(l.globe.pairs, t)
        case  flag: l.globe.flags = append(l.globe.flags, t)
            if s := __string(l.universe, t.Value); s == "clean" {
                mode.pos, mode.s = t.Pos(), "clean"
            }
        case *argumented:
            l.globe.args[t.Value] = t.args
            if f, y := t.Value.(flag); y {
                l.globe.flags = append(l.globe.flags, f)
            } else {
                l.globe.goals.append(l.universe, t/*.Value*/)
            }
        default:
            l.globe.goals.append(l.universe, t)
        }
    }

    if mode.s == "" {
        mode.s = "goals"
    }

    l.globe.mode.value = mode
}

func (u *universe) load(ctx Context) {
    if u.traceLaunch { defer un(l_trace(l_launch, "universe.load")) }

    if false { loadGrepCache(ctx) }

    if s := filepath.Join(u.workdir, ".smart", "modules"); s != "" {
        if _, e := os.Stat(s); e == nil { u.AddPaths(s) }
    }
    if s := filepath.Join(u.workdir, mainFileName); s != "" {
        if _, e := os.Stat(s); e != nil {
            s = filepath.Join(u.workdir, deprFileName)
            if _, e := os.Stat(s); e != nil { s = "" }
        }
    }

    u.globe.top = &loader{term:term{ctx, u.globe.scope}}

    l := ul{u, u.globe.top}
    l.parse_args(u.workdir, os.Args[1:]...)

    if u.autoProfs {
        if f, e := os.Create(filepath.Join(workBaseDir, "load.cpu.auto.prof")); e != nil {
            debug(ctx, "%v", e, trace{})
        } else {
            defer f.Close()
            if e := pprof.StartCPUProfile(f); e != nil {
                debug(ctx, "could not start CPU profile: %v", e, trace{})
            }
            defer pprof.StopCPUProfile()
        }
        defer func() {
            var prof string //= u.memProf
            if prof == "" { prof = filepath.Join(workBaseDir, "load.mem.auto.prof") }
            if f, e := os.Create(prof); e != nil {
                debug(ctx, "%v", e, trace{})
            } else {
                defer f.Close()
                runtime.GC() // update memory statistics
                if e := pprof.WriteHeapProfile(f); e != nil {
                    debug(ctx, "could not start CPU profile: %v", e, trace{})
                }
            }
        } ()
    }

    if u.verboseImport { prompt(ctx, "┌→%s\n", u.workdir) }

    defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseImport {
            var name string
            if p := _project(u.globe.top); p != nil { name = p.name }
            prompt(ctx, "└·%s … (%s)\n", name, d)
        } else if false && u.slow < d {
            debug(pc(ctx, u.workdir), "slow loading (%v)!!\n", d)
        }
    } (time.Now())

    spec, _ := filepath.Rel(workBaseDir, u.workdir)
    l.directory(l.loader, spec, u.workdir, nil)

    if l.globe.main == nil {
        debug(ctx, "nothing loaded", trace{})
    }
    return
}

func (u *universe) run() (result []Value) {
    if u.noRun { return }

    var main = u.globe.main
    if main == nil {
        debug(u, "no targets to update `%v`", u.globe.goals, trace{})
    }

    var ctx Context = closure_with(u, main.scope)
    if u.verbose { debug(ctx, "goal: %v", main) }

    removeTempDirs(ctx)

    if u.cpuProf != "" || u.autoProfs {
        var name = u.cpuProf
        if name == "" { name = "run.cpu.auto.prof" }
        defer cpu_profile(ctx, name, true)()
    } else if u.memProf != "" || u.autoProfs {
        var name = u.memProf
        if name == "" { name = "run.mem.auto.prof" }
        defer heap_profile(ctx, name)()
    }

    var done bool
    for _, flag := range u.globe.flags {
        if u.verboseExecFlags { info(ctx, "%v", flag) }

        var s = __string(ctx, flag.Value)
        var args, _ = u.globe.args[flag]
        var entries, _ = u.globe.flagEntries[s]
        for _, entry := range entries {
            if u.verboseExecFlags {
                info(ctx, "%v", entry)
                flush(ctx)
            }

            var res = entry.execute(ctx, args...)
            result = append(result, res...)
            done = true
        }
    }
    if done { return }

    var updated int
    var goals []Value
    var collect func(proj *project, vals []Value) bool
    collect = func(proj *project, vals []Value) bool {
        if len(vals) == 0 {
            if entry := proj.main; entry != nil {
                goals = append(goals, entry)
            } else {
                // NOTE: ignored project
            }
            return true
        }
        for _, goal := range vals {
            switch t := goal.(type) {
            case *null, *none: // just ignore
            case *word:
                if entries := proj._entries(ctx, t.s, true); entries == nil {
                    debug(ctx, "no such entry `%s`", t.s, trace{})
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case *delegate:
                var s = __string(ctx, t)
                if entries := proj._entries(ctx, s, true); entries == nil {
                    debug(ctx, "no such entry `%s` (via `%v`)", s, t, trace{})
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case flag:
                var s = __string(ctx, t)
                if entries := proj._entries(ctx, s, true); entries == nil {
                    debug(ctx, "no such entry `%s` (via `%v`)", s, t, trace{})
                    return false
                } else {
                    for _, entry := range entries { goals = append(goals, entry) }
                }
            case *argumented:
                {
                    // For examples:
                    //     project-name(-clean)
                    //     project/spec(-clean)
                    //     xxxx()
                    var (
                        s = __string(ctx, t.Value)
                        args = merge(t.args...)
                        found int
                    )
                    for _, p := range u.globe.loaded {
                        if p.name == s || p.spec == s { found += 1
                            if !collect(p, args) { return false }
                        }
                    }
                    if found == 0 {
                        debug(ctx, `"%s" not loaded: %v`, s, args, trace{})
                        return false
                    }
                }
            default:
                debug(ctx, "%v: unknown target: %v (%s)", proj, goal, typeof(goal), trace{})
                return false
            }
        }
        return true
    }

    if collect(main, merge(u.globe.goals.value)) {
        if len(goals) == 0 {
            if entry := main.main; entry != nil {
                goals = append(goals, entry)
            }
        }
        for _, goal := range goals {
            var args, _ = u.globe.args[goal]
            result = append(result, updateGoal(ctx, goal, args)...)
            updated += 1
        }
    }
    return
}

// A globe represents a global execution context.
type globe struct {
    *scope

    top    *loader
    main   *project
    loaded map[string]*project // loaded projects

    args map[Value][]Value
    flagEntries map[string][]entry
    flags []flag
    pairs []*pair

    os    *def
    goals *def
    mode  *def
}

func (g *globe) SetScopeOuter(scope *scope) { scope.outer = g.scope }
func (g *globe) AddFlagEntry(name string, entry entry) {
    flags, _ := g.flagEntries[name]
    flags     = append(flags, entry)
    g.flagEntries[name] = flags
    return
}
