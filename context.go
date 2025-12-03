//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "path/filepath"
    "runtime/debug"
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
    get_args       struct{}
    get_workdir    struct{}
    get_position   struct{}
    get_project    struct{}
    get_scope      struct{}
    on_erros       struct{ i int }
    param_name     struct{ i int }
    get_closure_scopes struct{}
    is_good_with struct{ p property ; a []any }
    is_test_case struct{}
    is_test_mode struct{}
    is_test_univ struct{}
    no_position  struct{}
    invalid_position struct{}
)

func _param_name(ctx Context, n int) (_ string) {
    if x, y := do(ctx, param_name{n}).(string); y { return x }
    return
}

func _workdir(ctx Context) (_ string) {
    if x, y := do(ctx, get_workdir{}).(string); y {
        return x
    } else {
        erro(ctx, "no workdir").trace()
        return
    }
}

func count_diag(ctx Context, t ...diagtype) (i int) {
    i, _ = do(ctx, act_count_dia{t}).(int)
    return
}

type Context interface { caster ; doer }
type caster interface { cast(reflect.Type) Context }
type doer interface { do(Context, any) any }

func do(c Context, o any) any { return c.do(c, o) }
func truly(ctx Context, ops ...any) (_ bool) {
	for _, op := range ops {
		switch t := do(ctx, op).(type) {
		case hit_result: return t.bool
		case bool: return t
		}
	}
	return
}

func try[T any](ctx Context, op any) (_ T) {
    if ctx != nil {
        if x, y := do(ctx, op).(T); y { return x }
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

type callstack []byte
type frames  struct{ int }
type skipint struct{ int }

var (
    callstackSkips = regexp.MustCompile(`^(?:extbit\.io/)?(?:.+?)smart\.(?:do(?:_bits)?|(?:recover_)?tr(?:ace|uly|y)|erro|(?:diagtracer|\(\*diagnostic\))\.trace)\(.+\)$`)
    callstackLine1 = regexp.MustCompile(`^(?:extbit\.io/)?(.+)(\(.*\))$`)
    callstackLine2 = regexp.MustCompile(`^	(.*?:\d+)(?: \+.*)?$`)
)
func cstack(i, j int, a ...any) callstack { return _callstack("", i+1, j, a...) }
func _callstack(s string, i, j int, args ...any) (res callstack) {
    i += 1 // skips this func

    var nums []int
    for _, a := range args {
        if t, y := a.(bool); y {
            if !t { return /* d */ }
        } else if t, y := a.(int); y {
            nums = append(nums, t)
        } else if t, y := a.(skipint); y {
            i += t.int
        } else if t, y := a.(frames); y {
            j += t.int
        }
    }
    if n := len(nums); n == 0 {
        j += 1
    } else if n == 1 {
        j += nums[0]
    } else if n == 2 {
        i += nums[0]
        j += nums[1]
    } else {
        panic("too many stack nums")
    }

    var gotPanic bool
    var v = bytes.Split(debug.Stack(), []byte{'\n'})
    for ; 0 < j && i+1 < len(v); i = i+1 {
        // skip diagnostic.trace lines
        if callstackSkips.Match(v[i]) { continue }

        var (
            sm1 = callstackLine1.FindSubmatch(v[i+0]) // versus FindAllSubmatch(v[i+0], 1)
            sm2 = callstackLine2.FindSubmatch(v[i+1]) // versus FindAllSubmatch(v[i+1], 1)
            isPanic = len(sm1) > 1 && bytes.Equal(sm1[1], []byte("panic"))
        )
        if sm1 != nil && sm2 != nil && !isPanic {
            var e string
            if 0 < j-1 && i < len(v) {
                if gotPanic { e = "		<---- panic" }
            } else {
                e = fmt.Sprintf("  (%d more)", len(v)-i)
            }

            res = append(res, sm2[1]...)
            res = append(res, []byte(":"+s+" ")...)
            res = append(res, sm1[1]...)
            res = append(res, sm1[2]...)

            if false {
                // collapse duplicated lines
                var dups int
                for n := i+2; n+1 < len(v); n += 1 {
                    if bytes.Equal(v[i+0], v[n+0]) /* && bytes.Equal(v[i+1], v[n+1]) */ {
                        dups += 1
                    } else {
                        break
                    }
                }
                if 0 < dups {
                    res = append(res, []byte(fmt.Sprintf("  (%d)", dups))...)
                }
            }

            res = append(res, []byte(e+"\n")...)
            j -= 1
        }

        gotPanic = isPanic
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

const (
    diagInfo diagtype = iota
    diagWarn
    diagError
    diagPrompt
    diagPromptLine
)

type diagtype int
type diagpoint struct {
    dt diagtype
    position Position
    message string
    stack []byte // see also debug.Stack()
}
func (d *diagpoint) tag() (s string) {
    switch d.dt {
    case diagPrompt: if /* !options.debugPrompt */false { return } else { s = "note:" }
    case diagInfo  : if /* !options.debugInfos  */false { return } else { s = "info:" }
    case diagWarn  : if /* !options.debugWarns  */false { return } else { s = "info:" }
    case diagError : if /* !options.debugErrors */false { return } else { s = "info:" }
    }
    return
}
func (d *diagpoint) debug(args ...any) *diagpoint {
    switch vertag {
    case "dev", "debug": // only print debug diags for dev and debug versions
    default: return d
    }

    // skips the standard stack lines, which are not informative
    // number of frames to dump
    d.stack = _callstack(d.tag(), 5, 0, args...)
    return d
}

type diagtracer struct { *diagpoint ; c Context }
func (d diagtracer) trace(a ...any) { trace(d.c, a...) }
func (d diagtracer) flush() { flush(d.c) }

type act_traced           struct{}
type too_many_diagnostics struct{ int }
type too_many_errors      struct{ int }
type trace_errors         struct{ Context ; int }
type trace_evoke_loop_err struct{ Context ; Value }
type trace_evoke_loop     struct{ Context }
type evoke_loop_null      struct{}
type evoke_loop_panic     struct{}

func (x trace_evoke_loop) inner() Context { return x.Context }
func (x trace_evoke_loop) cast(t reflect.Type) Context { return icast(x,t) }
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

func (t too_many_diagnostics) String() string { return fmt.Sprintf("too many diagnostics (%d)", t.int) }
func (t too_many_errors) String() string { return fmt.Sprintf("too many errors (%d)", t.int) }

// NOTE: never recover test_fail in recover_trace, it will break the test runner
type test_fail struct{ Context; int; i int }
func (t test_fail) String() (s string) {
    if t.int == 1 {
        s = fmt.Sprintf("test fail, %v", ts(t.Context))
    } else {
        s = fmt.Sprintf("test fail, %d errors, %v", t.int, ts(t.Context))
    }
    return
}

func recover_trace(ctx Context) {
    var te trace_errors
    var recovered int

    for e := recover(); e != nil; e = recover() {
        switch recovered += 1 ; t := e.(type) {
        case              bailout:
        case         trace_errors: te = t
        case              failure: erro(t.Context, t.Error())
        case                Value: erro(ctx, "trace: %s", ts(t))
        case               string: erro(ctx, "trace: %s", t)
        case        runtime.Error: erro(ctx, "trace: %s", t.Error())
        case too_many_diagnostics: erro(ctx, "too many diagnostics (%v)", t.int)
        case too_many_errors     : erro(ctx, "too many errors (%v)", t.int)
        case trace_evoke_loop_err: erro(pc(ctx,t.Value), "evocation loop (%s)", t.Value)
        case test_fail:
            if t.i += 1; t.i == 1 {
                note(ctx, "%s (%d panics)", t, recovered).debug(1024)
            }
            flush(ctx)
            panic(t)
        default:
            panic(e) //erro(ctx, "trace: %s", ts(e))
        }
    }

    if recovered > 0 {
        note(ctx, "%s (%d panics)", ts(te.Context), recovered).debug(512)
        if true { flush(ctx) }
    }

    if false && truly(ctx, is_test_mode{}) {
        if te.Context != nil && 0 < te.int {
            panic(test_fail{te.Context, te.int, 0}) // rethrow to break the test runner
        }
    }
}

type no_recover struct{}

func trace(ctx Context, args ...any) {
    var loop trace_evoke_loop_err
    var recov = true
    for _, a := range args {
        switch t := a.(type) {
        case no_recover: recov = false
        case trace_evoke_loop_err: loop = t
        }
    }
    if recov { defer recover_trace(ctx) }
    if x, y := do(ctx, act_traced{}).(int); y && x > 0 {
        if truly(ctx, is_test_mode{}) {
            panic(test_fail{ctx, x, 0})
        } else if loop.Value == nil {
            panic(trace_errors{ctx, x})
        } else {
            panic(loop)
        }
    }
    return
}

type flush_diags struct{}

const diagnostic_limit = 10_000
var   diagnostic_limit_erros = 520
var   diagnostic_limit_bytes = 1_000_000

func _diagnostic(c Context) *diagnostic { return cast[*diagnostic](c) }

type add_diag struct { dt diagtype; fmt string; a []any }
type diagnostic struct {
    Context
    sync.Mutex
    newlines []*diagpoint
    points   []*diagpoint
    nested [][]*diagpoint // TODO: this shall perish
    erros   int // number of flushed erros
    flushed int // in bytes
    traced  int
}
func (d *diagnostic) aquire() (unlock func()) { d.Lock(); return func(){ d.Unlock() }}
func (d *diagnostic) cast(t reflect.Type) Context { return icast(d,t) }
func (d *diagnostic) inner() Context { return d.Context }
func (d *diagnostic) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case property: if t&propErros != 0 { return d.erros }
    case flush_diags:  return d.flush(ctx)
    case act_count_dia: return d.count(t.t...)
    case act_traced : if i := d.count(diagError); i > 0 { d.traced += 1 ; return i }
    case add_diag: return diagtracer{ d.point(ctx, t.dt, t.fmt, t.a...), ctx }
    }
    if d.Context == nil { return }
    return d.Context.do(ctx, op)
}
func (d *diagnostic) reset() { defer d.aquire()(); d.points = []*diagpoint{} }
func (d *diagnostic) add(p *diagpoint) *diagpoint {
    defer d.aquire()()

    if i := len(d.points)+len(d.newlines); diagnostic_limit < i {
        panic(too_many_diagnostics{i})
    }

    if p.dt == diagPromptLine {
        d.newlines = append(d.newlines, p)
        return p
    } else if strings.HasSuffix(p.message, "\n") {
        if  d.points = append(d.points, p) ; d.newlines != nil {
            d.points = append(d.points, d.newlines...)
            d.newlines = nil
        }
        return p
    } else {
        d.points = append(d.points, p)
        return p
    }
}
func (d *diagnostic) nest(points []*diagpoint) {
    defer d.aquire()()
    d.nested = append(d.nested, points)
}
func (d *diagnostic) point(ctx Context, dt diagtype, f string, args ...any) *diagpoint {
    if dt != diagPrompt { f = strings.TrimSpace(f) }
    return d.add(&diagpoint{dt, _position(ctx), fmt.Sprintf(f, args...), nil})
}
func (d *diagnostic) count(dt ...diagtype) (errs int) {
    defer d.aquire()()
    for _, d := range d.points {
        for _, t := range dt {
            if d.dt == t { errs += 1 ; break }
        }
    }
    return
}
func (d *diagnostic) flush(ctx Context) (errs int) {
    defer func() { if d.erros += errs ; errs > 0 { do(ctx, on_erros{errs}) }} ()

    var restrict_diagnostics = func() {
        if x, y := diagnostic_limit_erros, d.erros; 0 < x && x < y {
            if false { d.erros = 0 } // reset to avoid causing next panics
            panic(too_many_errors{y})
        }
        if x, y := diagnostic_limit_bytes, d.flushed; false && 0 < x && x < y {
            if false { d.flushed = 0 } // reset to avoid causing next panics
            panic(too_many_diagnostics{y})
        }
    }

    const count_bytes = false

    var flush_point = func(p *diagpoint, pend bool) (_ bool) {
        defer restrict_diagnostics()

        pos, msg := p.position.String(), p.message

        if count_bytes {
            d.flushed += len(pos) + len(msg)
        } else {
            d.flushed += 1
        }

        switch p.dt {
        case diagInfo: fmt.Fprintf(stderr, "%v:info: %s\n", pos, msg)
        case diagWarn: fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
        case diagPromptLine:
            if msg != "" { fmt.Fprintf(stderr, "%s\n", msg) }
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

    for {
        var point *diagpoint

        d.Lock()
        if len(d.points) > 0 {
            point = d.points[0]
            d.points = d.points[1:]
        }
        d.Unlock()

        if point == nil || flush_point(point, true) { break }
        if errs > 16 {
            fmt.Fprintf(stderr, "%v: too many errors (%d)\n", _position(ctx), errs)
            break
        }
    }

    d.Lock()
    for i := 0; len(d.nested) > 0; d.nested = d.nested[1:] {
        i += 1
        fmt.Fprintf(stderr, "\n#%d:\n", i)
        for _, d := range d.nested[0] { flush_point(d, false) }
        fmt.Fprintf(stderr, "#%d;\n\n", i)
    }
    d.Unlock()
    return
}

func flush(ctx Context) (i int) { i, _ = do(ctx, flush_diags{}).(int); return }

func diag(ctx Context, dt diagtype, f string, a ...any) (res diagtracer) {
    res, _ = do(ctx, add_diag{dt, f, a}).(diagtracer)
    return
}

func info(ctx Context, f string, a ...any) diagtracer { return diag(ctx, diagInfo,   f, a...) }
func warn(ctx Context, f string, a ...any) diagtracer { return diag(ctx, diagWarn,   f, a...) }
func erro(ctx Context, f string, a ...any) diagtracer { return diag(ctx, diagError,  f, a...) }
func prompt(c Context, f string, a ...any) diagtracer { return diag(c,   diagPrompt, f, a...) }
func note(ctx Context, f string, a ...any) diagtracer {
    if !strings.HasSuffix(f, "\n") { f += "\n" }
    return prompt(ctx, _position(ctx).String()+": "+f, a...)
}

func infostack(ctx Context, n int, a ...any) diagtracer { return diagstack(ctx, n, diagInfo  , a...) }
func warnstack(ctx Context, n int, a ...any) diagtracer { return diagstack(ctx, n, diagWarn  , a...) }
func errostack(ctx Context, n int, a ...any) diagtracer { return diagstack(ctx, n, diagError , a...) }
func promstack(ctx Context, n int, a ...any) diagtracer { return diagstack(ctx, n, diagPrompt, a...) }
func notestack(ctx Context, n int, a ...any) diagtracer {
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
func diagstack(ctx Context, n int, dt diagtype, a ...any) (point diagtracer) {
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

func assured(ctx Context, dontCheckErrors ...bool) (recovered, errs int) {
    var te trace_errors
    var f *failure
    for e := recover(); e != nil; e = recover() {
        switch recovered += 1; t := e.(type) {
        case bailout: continue
        case trace_errors: t.Context = nil ; continue
        case failure:
            erro(t.Context, t.Error()) ; t.Context = nil
            if f == nil { f = &t }
        case         Value: erro(ctx, "assured: %s", ts(t))
        case        string: erro(ctx, "assured: %s", t)
        case runtime.Error: erro(ctx, "assured: %s", t.Error())
        default:            erro(ctx, "assured: %s", ts(e))
        }
    }

    if 0 < recovered {
        // if defer assured from top stack, this will dump the full stack of panics
        promstack(ctx, 5, "%v: %s (%d panics)", _position(ctx), ts(te.Context), recovered).debug(128)
    }

    te.Context = nil

    var dia = _diagnostic(ctx) ; dia.flush(ctx)
    if len(dontCheckErrors) > 0 && dontCheckErrors[0] { return }

    if errs = dia.count(diagError); 0 < errs && recovered == 0 {
        note(ctx, "got %d errors (flushed %d, recovered %d)", errs, dia.erros, recovered).debug(10)
        if f != nil && (len(dontCheckErrors) == 0 || !dontCheckErrors[0]) {
            panic(_failure(ctx, "fail [assured]"))
        } else {
            dia.flush(ctx)
        }
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
