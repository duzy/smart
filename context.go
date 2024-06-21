//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
  checkpoints = vertag != "final"
  trace_recover = true
)

type property uint64

const (
  propParameters property = 1<<iota
  propFullVal
  propDirtyOpts
  propErros
  propExAuto
  propExClosure
  propExDelegate
  propExDef  //   =, :=, ::=, ...
  propExDef0 //   =
  propExDef1 //  :=
  propExDef2 // ::=
  propExDef3 // ;:= (TODO)
  propExDefValue
  propExDigital // $0, $1, ...
  propExDisjunction
  propExFullFile
  propExPairVal
  propExPathStr
  propExPlaceholder // $_
  propExCondless
  propExFinal // aka x.string(ctx)
  propReversal
  propUnmap
)

type (
  act_count_dia  struct{ t []diagType }
  act_on_erros   struct{ i int }
  act_dirty_mark struct{ a []Value }
  act_dirt       struct{ a []Value }
  act_traversed  struct{ v Value }
  act_traverse   struct{ v Value }
  // act_match_entry struct{ v Value }
  act_arguments  struct{}
  get_arguments  struct{}
  get_workdir    struct{}
  get_position   struct{}
  get_project    struct{}
  get_scope      struct{}
  get_closure_scope struct{}
  // get_object    struct{ s string }
  // get_entry     struct{ s string }
  get_param_name struct{ i int }
  is_good_with   struct{ p property ; a []any }
  is_test_mode   struct{}
)

func _position(ctx Context) (res Position) {
  if i := do(ctx, get_position{}); i == nil {
    erro(ctx, "no such operator: position, %v", ts(ctx)).trace()
  } else if x, y := i.(Position); !y {
    erro(ctx, "not position: %v", ts(i)).trace()
  } else {
    res = x
  }
  return
}

func _parameters(ctx Context) (res map[string]*auto) {
  if i := do(ctx, propParameters); i != nil {
    if t, y := i.(map[string]*auto); y { res = t }
  }
  return
}

func _paramName(ctx Context, n int) (res string) {
  if i := do(ctx, get_param_name{n}); i != nil {
    if s, y := i.(string); y { res = s }
  }
  return
}

func _workdir(ctx Context) (res string) {
  if i := do(ctx, get_workdir{}); i == nil {
    erro(ctx, "no such operator: workdir, %v", ts(ctx)).trace()
  } else if t, y := i.(string); y { res = t } else {
    erro(ctx, "not string: %v", ts(i)).trace()
  }
  return
}

func count_error(ctx Context) int { return count_diag(ctx, diagError) }
func count_diag(ctx Context, t ...diagType) (i int) {
  i, _ = do(ctx, act_count_dia{t}).(int)
  return
}

func ex_auto(ctx Context) (res bool) {
  res, _ = do(ctx, propExAuto).(bool)
  return
}

func ex_closure(ctx Context) (res bool) {
  res, _ = do(ctx, propExClosure).(bool)
  return
}

func ex_delegate(ctx Context) (res bool) {
  res, _ = do(ctx, propExDelegate).(bool)
  return
}

func ex_def(ctx Context) (res bool) {
  res, _ = do(ctx, propExDef).(bool)
  return
}

func ex_def0(ctx Context) (res bool) {
  res, _ = do(ctx, propExDef0).(bool)
  return
}

func ex_def1(ctx Context) (res bool) {
  res, _ = do(ctx, propExDef1).(bool)
  return
}

func ex_def2(ctx Context) (res bool) {
  res, _ = do(ctx, propExDef2).(bool)
  return
}

func ex_def_value(ctx Context) (res bool) {
  res, _ = do(ctx, propExDefValue).(bool)
  return
}

func ex_digital(ctx Context) (res bool) {
  res, _ = do(ctx, propExDigital).(bool)
  return
}

func ex_disjunction(ctx Context) (res bool) {
  res, _ = do(ctx, propExDisjunction).(bool)
  return
}

func ex_fullfile(ctx Context) (res bool) {
  res, _ = do(ctx, propExFullFile).(bool)
  return
}

func ex_pair_value(ctx Context) (res bool) {
  res, _ = do(ctx, propExPairVal).(bool)
  return
}

func ex_path_str(ctx Context) (res bool) {
  res, _ = do(ctx, propExPathStr).(bool)
  return
}

func ex_placeholder(ctx Context) (res bool) {
  res, _ = do(ctx, propExPlaceholder).(bool)
  return
}

func ex_condless(ctx Context) (res bool) {
  res, _ = do(ctx, propExCondless).(bool)
  return
}

func ex_final(ctx Context) (res bool) {
  res, _ = do(ctx, propExFinal).(bool)
  return
}

func goodwith(ctx Context, p property, a ...any) (res bool) {
  res, _ = do(ctx, is_good_with{p, a}).(bool)
  return
}

type Context interface { caster ; doer }
type caster interface { cast(reflect.Type) Context }
type doer interface { do(Context, any) any }

func do(ctx Context, op any) any { return ctx.do(ctx, op) }
func try[T any](ctx Context, op any) (_ T) {
  if x, y := do(ctx, op).(T); y { return x }
  return
}
func truly(ctx Context, op any) (_ bool) {
  if x, y := do(ctx, op).(bool); x && y { return x }
  return
}

func cast[T Context](ctx Context) (res T) {
  if ctx != nil {
    var t = ctx.cast(reflect.TypeOf(res))
    if t != nil { return t.(T) }
  }
  return
}

func implcast(ctx Context, t reflect.Type) (c Context) {
  if v := reflect.ValueOf(ctx); v.Type() == t {
    c = ctx
  } else if i := _inner(v); i != nil {
    c = i.(Context).cast(t)
  }
  return
}

func _inner(v reflect.Value) (i any) {
  if x, y := v.Interface().(interface{ inner() Context }); y && x != nil {
    i = x.inner()
  } else if t := v.Type(); t.Kind() == reflect.Struct {
    if x, y := t.FieldByName("Context"); y && x.Anonymous {
      if v = v.FieldByIndex(x.Index); v.IsValid() {
        i = v.Interface()
      }
    }
  } else if t.Kind() == reflect.Pointer {
    i = _inner(v.Elem())
  }
  return
}

func inner(ctx Context) (c Context) {
  if i := _inner(reflect.ValueOf(ctx)); i != nil { c = i.(Context) }
  return
}

func _scope(ctx Context) (s *scope) {
  s, _ = do(ctx, get_scope{}).(*scope)
  return
}

func _project(ctx Context) (p *project) {
  p, _ = do(ctx, get_project{}).(*project)
  return
}

func getTargetValue(ctx Context) (res Value) {
  if val := auto_get(ctx, "@"); val == nil {
    if false { erro(ctx, "target is nil") }
  } else if vals := expand(ctx, val); len(vals) == 1 {
    res = scalarize(vals[0])
  } else {
    erro(at(ctx,val), "multiple targets: %v → %v", val, vals)
  }
  return
}

func getTargetValueString(ctx Context) (val Value, str string) {
  if val = getTargetValue(ctx); isNull(val) {
    if false { erro(ctx, "target '%v' is nil", val) }
  } else {
    str, _ = as{val}.fullnameOrFinal(ctx)
  }
  return
}

type callstack []byte
type frames  struct{ int }
type skipint struct{ int }

var (
  callstackSkips = regexp.MustCompile(`^(?:extbit\.io/)?(?:.+?)smart\.(?:do(?:_bits)?|tr(?:ace|uly|y)|erro|(?:diagtracer|\(\*diagnostic\))\.trace)\(.+\)$`)
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
  diagInfo diagType = iota
  diagWarn
  diagError
  diagPrompt
  diagPromptLine
)

type diagType int
type diagpoint struct {
  dt diagType
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

type diagtracer struct { *diagpoint ; ctx Context }
func (d diagtracer) trace() {
  if false {
    defer trace(d.ctx)
    d.stack = _callstack(d.tag(), 5, 0)
  } else {
    trace(d.ctx)
  }
}

type act_traced struct{}
type too_many_diagnostics struct{ i int }
type too_many_errors struct{ i int }
type trace_errors struct { Context ; e int }

func (t trace_errors) String() string {
  return fmt.Sprintf("%v %d, %v", typeof(t), t.e, ts(t.Context))
}

func (t too_many_diagnostics) String() string { return fmt.Sprintf("too many diagnostics (%d)", t.i) }
func (t too_many_errors) String() string { return fmt.Sprintf("too many errors (%d)", t.i) }

func trace(ctx Context, _ ...any) {
  if trace_recover {
    var x Context
    var recovered int
    for e := recover() ; e != nil ; e = recover() {
      switch recovered += 1 ; t := e.(type) {
      case       bailout:
      case       trace_errors: x = t.Context
      case       failure: erro(t.Context, t.Error())
      case         Value: erro(at(ctx,t), "trace: %s", ts(t))
      case        string: erro(   ctx   , "trace: %s", t)
      case runtime.Error: erro(   ctx   , "trace: %s", t.Error())
      case too_many_diagnostics: erro(ctx, "too many diagnostics (%v)", t.i)
      case too_many_errors     : erro(ctx, "too many errors (%v)", t.i)
      default: erro(ctx, "trace: %s", ts(e))
      }
    }
    if recovered > 0 {
      erro(ctx, "%s (%d panics)", ts(x), recovered).debug(64)
      if true { flush(ctx) }
    }
  }
  if x, y := do(ctx, act_traced{}).(int); y && x == 1 {
    panic(trace_errors{ctx, x})
  }
  return
}

type act_flush_diags struct{}

const diagnostic_limit = 10_000
var   diagnostic_limit_erros = 520
var   diagnostic_limit_bytes = 1_000_000

func _diagnostic(c Context) *diagnostic { return cast[*diagnostic](c) }

type diagnostic struct {
  Context
  sync.Mutex
  newlines []*diagpoint
  points   []*diagpoint
  nested [][]*diagpoint // TODO: this shall perish
  erros int // number of flushed erros
  flushed int // in bytes
  traced int
}
func (d *diagnostic) aquire() (unlock func()) { d.Lock(); return func(){ d.Unlock() }}
func (d *diagnostic) cast(t reflect.Type) Context { return implcast(d,t) }
func (d *diagnostic) do(ctx Context, op any) any {
  switch t := op.(type) {
  case act_count_dia: return d.count(t.t...)
  case act_flush_diags: return d.flush(ctx)
  case act_traced: if i := d.counterror(); i > 0 { d.traced += 1 ; return i }
  case property: if t&propErros != 0 { return d.erros }
  }
  if d.Context == nil { return nil }
  return d.Context.do(ctx, op)
}
func (d *diagnostic) reset() { defer d.aquire()(); d.points = []*diagpoint{} }
func (d *diagnostic) add(point *diagpoint) *diagpoint {
  defer d.aquire()()

  if true {
    if i := len(d.points)+len(d.newlines); diagnostic_limit < i {
      panic(too_many_diagnostics{i})
    }
  }

  if false && 0 < diagnostic_limit_bytes {
    var x = d.flushed
    for _, t := range append(d.points, d.newlines...) {
      x += 1 + bytes.Count(t.stack, []byte("\n"))
    }
    if diagnostic_limit_bytes < x {
      d.flushed = 0 // reset to avoid causing next panics
      panic(too_many_diagnostics{x})
    }
  }

  if point.dt == diagPromptLine {
    d.newlines = append(d.newlines, point)
    return point
  } else if strings.HasSuffix(point.message, "\n") {
    if d.points = append(d.points, point); d.newlines != nil {
      d.points = append(d.points, d.newlines...)
      d.newlines = nil
    }
    return point
  } else {
    d.points = append(d.points, point)
    return point
  }
}
func (d *diagnostic) nest(points []*diagpoint) {
  defer d.aquire()()
  d.nested = append(d.nested, points)
}
func (d *diagnostic) point(ctx Context, dt diagType, f string, args ...any) *diagpoint {
  if dt != diagPrompt { f = strings.TrimSpace(f) }
  return d.add(&diagpoint{ dt, _position(ctx), fmt.Sprintf(f, args...), nil })
}
func (d *diagnostic) error() bool { return d.counterror() > 0 }
func (d *diagnostic) counterror() int { return d.count(diagError) }
func (d *diagnostic) count(dt ...diagType) (errs int) {
  defer d.aquire()()
  for _, d := range d.points {
    for _, t := range dt {
      if d.dt == t { errs += 1 ; break }
    }
  }
  return
}
func (d *diagnostic) flush(ctx Context) (errs int) {
  defer func() { if d.erros += errs ; errs > 0 { do(ctx, act_on_erros{errs}) }} ()

  var flush = func(p *diagpoint, pend bool) bool {
    defer func() {
      if x, y := diagnostic_limit_erros, d.erros; 0 < x && x < y {
        if false { d.erros = 0 } // reset to avoid causing next panics
        panic(too_many_errors{y})
      }
      if x, y := diagnostic_limit_bytes, d.flushed; 0 < x && x < y {
        if false { d.flushed = 0 } // reset to avoid causing next panics
        panic(too_many_diagnostics{y})
      }
    } ()

    pos, msg := p.position.String(), p.message

    d.flushed += len(pos) + len(msg)

    switch p.dt {
    case diagError: fmt.Fprintf(stderr, "%v:error: %s\n",   pos, msg); errs += 1
    case diagInfo : fmt.Fprintf(stderr, "%v:info: %s\n",    pos, msg)
    case diagWarn : fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
    case diagPromptLine: if msg != "" { fmt.Fprintf(stderr, "%s\n", msg) }
    case diagPrompt    : if msg != "" { fmt.Fprintf(stderr, "%s"  , msg) }
      if pend && !strings.HasSuffix(msg, "\n") { return true }
    }

    if p.stack != nil {
      fmt.Fprintf(stderr, "%s\n", bytes.TrimSpace(p.stack))
      d.flushed += len(p.stack) //(1 + bytes.Count(p.stack, []byte("\n")))
    }

    return false
  }

  for {
    var point *diagpoint

    d.Lock()
    if len(d.points) > 0 {
      point = d.points[0]
      d.points = d.points[1:]
    }
    d.Unlock()

    if point == nil || flush(point, true) { break }
    if errs > 16 {
      fmt.Fprintf(stderr, "%v: too many errors (%d)\n", _position(ctx), errs)
      break
    }
  }

  d.Lock()
  for i := 0; len(d.nested) > 0; d.nested = d.nested[1:] {
    i += 1
    fmt.Fprintf(stderr, "\n#%d:\n", i)
    for _, d := range d.nested[0] { flush(d, false) }
    fmt.Fprintf(stderr, "#%d;\n\n", i)
  }
  d.Unlock()
  return
}

// func flush(ctx Context) int { return _diagnostic(ctx).flush(ctx) }
func flush(ctx Context) (i int) { i, _ = do(ctx, act_flush_diags{}).(int); return }

func diag(ctx Context, dt diagType, f string, a ...any) (_ diagtracer) {
  if d := _diagnostic(ctx) ; d != nil {
    return diagtracer{d.point(ctx, dt, f, a...),ctx}
  }
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
    if len(a) > 0 { if s, y := a[0].(string); y {
      f, a = s, a[1:]
    }}

    a = append([]any{"%v: "+f+"\n", _position(ctx)}, a...)
  }
  return diagstack(ctx, n, diagPrompt, a...)
}
func diagstack(ctx Context, n int, dt diagType, a ...any) (point diagtracer) {
  var s string
  if 0 < len(a) {
    if x, y := a[0].(string); y {
      s, a = x, a[1:] // separate the format string and args
      if !strings.HasSuffix(s, "\n") { s += "\n" }
    }
  }

  point = diag(ctx, dt, s, a...)

  if _p := _positional(ctx); _p != nil {
    point.position = _p.position
    dt = diagInfo

    for last := &_p.position; 0 < n && _p != nil && _p.Context != nil; {
      var pos = &_p.position
      if pos == last || last.Same(pos) {
        _p = _positional(_p.Context)
        continue
      }

      n -= 1
      last = pos

      if true {
        s = _project(ctx).name + " ; " + ts(_p)
      } else {
        s = ts(_p)
      }

      if _e := _entry(_p); _e != nil {
        if t, _, _ := entryIndicator(_p, _e); t == "" {
          s += " : " + _e.ident(ctx)
        } else {
          s += " : " + t
        }
      }

      var p = _p
      if _p = _positional(_p.Context) ; 1 == n {
        var c int
        for t := _p; t != nil; t = _positional(t.Context) { c += 1 }
        if 0 < c { s += fmt.Sprintf(" ... (%d more)", c) }
      }

      point = diag(p, dt, s)
    }
  }
  return
}

func _positional(c Context) *positional { return cast[*positional](c) }

type positional struct { Context ; position Position }
func (p *positional) Position() Position { return p.position }
func (p *positional) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *positional) ts(string) string { return ts(p.Context) }
func (p *positional) do(ctx Context, op any) (_ any) {
  switch op.(type) {
  case get_position: return p.position
  }
  if p.Context == nil { return }
  return p.Context.do(ctx, op)
}

func _at(ctx Context, p Position) Context { return &positional{ctx, p} }
func at(ctx Context, a any) Context {
  if ctx == nil { panic("nil context") }

  var pos Position
  switch t := a.(type) {
  case Position  : pos = t
  case positioner: pos = t.Position()
  case doer:
    if x, y := do(ctx, get_position{}).(Position); y && x.valid() { pos = x }
  default:
    if false { erro(ctx, "non-position arg: %v", ts(a)).trace() }
    return ctx
  }

  if p := _position(ctx) ; p.valid() && pos.valid() && !p.Same(&pos) {
    for c, i, n := ctx, 0, 0; c != nil; c, i = inner(c), i+1 {
      if _, y := c.(*positional); y {
        if n += 1; n > /* 999 */100 {
          warnstack(ctx, 3, "too many positions: %v/%v; %v", n, i, ctx).debug(16)
          if true { flush(ctx) }
          return ctx
        }
      }
    }

    ctx = _at(ctx, pos)
  }
  return ctx
}

func walkSmartBaseDirs(ctx Context, cwd string, vis func(string)bool) (s string) {
  s = cwd
  for s != "" {
    file := stat(ctx, ".smart", stat_dir{s})
    if file != nil && file.info.IsDir() && !vis(s) { break }
    if up := filepath.Dir(s); up == s {
      break
    } else {
      s = up
    }
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
    case         Value: erro(at(ctx,t), "assured: %s", ts(t))
    case        string: erro(   ctx   , "assured: %s", t)
    case runtime.Error: erro(   ctx   , "assured: %s", t.Error())
    default:            erro(   ctx   , "assured: %s", ts(e))
    }
  }

  if 0 < recovered {
    // if defer assured from top stack, this will dump the full stack of panics
    promstack(ctx, 5, "%v: %s (%d panics)", _position(ctx), ts(te.Context), recovered).debug(128)
  }

  te.Context = nil

  var dia = _diagnostic(ctx) ; dia.flush(ctx)
  if len(dontCheckErrors) > 0 && dontCheckErrors[0] { return }

  if errs = dia.counterror(); 0 < errs && recovered == 0 {
    note(ctx, "got %d errors (flushed %d, recovered %d)", errs, dia.erros, recovered).debug(10)
    if f != nil && (len(dontCheckErrors) == 0 || !dontCheckErrors[0]) {
      panic(_failure(ctx, "fail [assured]"))
    } else {
      dia.flush(ctx)
    }
  }
  return
}

func CommandLine() {
  var context = new_universe() ; defer assured(context, false)
  if  context.traceLaunch { defer un(l_trace(l_launch, "CommandLine")) }

  var modulesPaths, packagePaths searchlist
  walkSmartBaseDirs(context, context.workdir, func(s string) bool {
    if baseTmpPath == "" { baseTmpPath = s }
    packagePaths = append(packagePaths, filepath.Join(s, ".smart", "packages"))
    modulesPaths = append(modulesPaths, filepath.Join(s, ".smart", "modules"))
    return true
  })
  packagePaths = append(packagePaths, filepath.Join(context.prefix, "user", "lib", "smart", "packages"))
  modulesPaths = append(modulesPaths, filepath.Join(context.prefix, "user", "lib", "smart", "modules"))

  // make sure that .smart dirs have higher priority.
  context.paths = append(modulesPaths, context.paths...)
  for _, s := range modulesPaths {
    searchFile := filepath.Join(s, ".search")
    if fi, _ := os.Stat(searchFile); fi == nil { continue }
    var file, err = os.Open(searchFile)
    if err != nil { fmt.Fprintf(stderr, "%v", err); return } else { defer file.Close() }
    for r := bufio.NewReader(file); err == nil; {
      var ( fi os.FileInfo; line string )
      if line, err = r.ReadString('\n'); err != nil {
        if err != io.EOF { fmt.Fprintf(stderr, "%v", err) } else { err = nil
          if line == "" { break } }
      } else {
        line = strings.TrimSpace(line)
      }
      if strings.HasPrefix(line, "#") {
        continue
      } else if filepath.IsAbs(line) {
        line = filepath.Clean(line)
      } else {
        line = filepath.Clean(filepath.Join(s, line))
      }
      if fi, err = os.Stat(line); err == nil && fi.IsDir() {
        context.paths = append(context.paths, line)
      }
    }
    if err != nil { fmt.Fprintf(stderr, "%v: %v", file, err); return }
  }

  var dia = _diagnostic(context)
  if dia.counterror() > 0 { return }

  if false { loadGrepCache(context) }

  if err := context.load(); err != nil {
    erro(context, "loading work failed: %v", err)
  } else if dia.flush(context) > 0 {
    prompt(context, "loading work got %d errors\n", dia.erros)
  } else if context.help {
    do_helpscreen(context)
  } else if context.printFlags {
    print_flag_trace(context)
  } else if context.printConfig {
    print_configuration(context)
  } else if numUpdatedPlugins > 0 { // see buildPlugin
    prompt(context, "plugins updated, please relaunch.\n")
  } else if context.commandline.configure {
    configure(context)
  } else if result := context.run(); dia.flush(context) > 0 {
    prompt(context, "run work got %d errors\n", dia.erros)
  } else if result != nil {
    for i, v := range result {
      if s := ""; v == nil {
        s = "<nil>"
      } else if s = strings.TrimSpace(v.string(context)); s == "" {
        continue
      } else if i == 0 {
        fmt.Fprintf(stderr, "%s", s)
      } else {
        fmt.Fprintf(stderr, ", %s", s)
      }
    }
    fmt.Fprintf(stderr, "\n")
  }

  // if false { promptLeavingDirectory(context) }
}
