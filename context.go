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
  propPosition property = 1<<iota
  propParameters
  propFullVal
  propWorkDir
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
  propExEvaluation
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
  actCountDiag  struct{ t []diagType }
  actOnErros    struct{ i int }
  actDirtyMark  struct{ a []Value }
  actDirty      struct{ a []Value }
  actTraversed  struct{ v Value }
  actTraverse   struct{ v Value }
  actMatchEntry struct{ v Value }
  actArguments  struct{}
  getArguments  struct{}
  getIsTestMode struct{}
  getScope      struct{}
  getProject    struct{}
  getClosure    struct{}
  getObject     struct{ s string }
  getEntry      struct{ s string }
  propGoodWith  struct{ p property ; a []any }
  propParamName struct{ i int }
)

func _position(ctx Context) (res Position) {
  if i := do(ctx, propPosition); i == nil {
    erro(ctx, "no such operator: position, %v", ts(ctx)).debug()
    trace(ctx)
  } else if x, y := i.(Position); !y {
    erro(ctx, "not position: %v", ts(i)).debug()
    trace(ctx)
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
  if i := do(ctx, propParamName{n}); i != nil {
    if s, y := i.(string); y { res = s }
  }
  return
}

func _workdir(ctx Context) (res string) {
  if i := do(ctx, propWorkDir); i == nil {
    erro(ctx, "no such operator: workdir, %v", ts(ctx)).debug(24)
  } else if t, y := i.(string); y { res = t } else {
    erro(ctx, "not string: %v", ts(i)).debug(2)
  }
  return
}

func count_error(ctx Context) int { return count_diag(ctx, diagError) }
func count_diag(ctx Context, t ...diagType) (i int) {
  i, _ = do(ctx, actCountDiag{t}).(int)
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

func ex_evaluation(ctx Context) (res bool) {
  res, _ = do(ctx, propExEvaluation).(bool)
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
  res, _ = do(ctx, propGoodWith{p, a}).(bool)
  return
}

func is_test_mode(ctx Context) (res bool) {
  res, _ = do(ctx, getIsTestMode{}).(bool)
  return
}

type caster interface { cast(reflect.Type) Context }
type doer interface { do(Context, any) any }
type Context interface {
  positioner
  caster
  doer
}

func do(ctx Context, op any) any { return ctx.do(ctx, op) }

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

func get_scope(ctx Context) (s *scope) {
  s, _ = do(ctx, getScope{}).(*scope)
  return
}

func get_project(ctx Context) (p *project) {
  p, _ = do(ctx, getProject{}).(*project)
  return
}

func closure_scopes(ctx Context) (s []*scope) {
  s, _ = do(ctx, getClosure{}).([]*scope)
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
  callstackLine1 = regexp.MustCompile(`^(?:extbit\.io/)?(.+)(\(.*\))$`)
  callstackLine2 = regexp.MustCompile(`^	(.*?:\d+)(?: \+.*)?$`)
  callstackSkips = regexp.MustCompile(`^(?:extbit\.io/)?(?:.+?)smart\.(?:do|\(\*diagnostic\)\.trace)\(.+\)$`)
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
type diagPoint struct {
  dt diagType
  position Position
  message string
  stack []byte // see also debug.Stack()
}
func (d *diagPoint) debug(args ...any) *diagPoint {
  if d == nil { panic("nil diag point") }

  switch vertag {
  case "dev", "debug": // only print debug diags for dev and debug versions
  default: return d
  }

  var s string
  switch d.dt {
  case diagPrompt: if /* !options.debugPrompt */false { return d } else { s = "note:" }
  case diagInfo  : if /* !options.debugInfos  */false { return d } else { s = "info:" }
  case diagWarn  : if /* !options.debugWarns  */false { return d } else { s = "info:" }
  case diagError : if /* !options.debugErrors */false { return d } else { s = "info:" }
  }

  // skips the standard stack lines, which are not informative
  // number of frames to dump
  d.stack = _callstack(s, 5, 0, args...)
  return d
}

type tracend struct { Context }
func (t tracend) String() string { return "trace "+ts(t.Context) }

func trace(ctx Context, a ...any) {
  if trace_recover {
    var x Context
    var recovered int
    for e := recover() ; e != nil ; e = recover() {
      switch recovered += 1 ; t := e.(type) {
      case       bailout:
      case       tracend:  x = t.Context
      case       failure: erro(t.Context, t.Error())
      case         Value: erro(at(ctx,t), "trace: %s", ts(t))
      case        string: erro(   ctx   , "trace: %s", t)
      case runtime.Error: erro(   ctx   , "trace: %s", t.Error())
      default:            erro(   ctx   , "trace: %s", ts(e))
      }
    }
    if 0 < recovered {
      erro(ctx, "%s (%d panics)", ts(x), recovered).debug(64)
    }
  }
  if d := _diagnostic(ctx) ; d.countError() > 0 {
    if d.traced += 1 ; d.traced == 1 { panic(tracend{ctx}) }
  }
  return
}

func _diagnostic(c Context) *diagnostic { return cast[*diagnostic](c) }

const diagnostic_limit = 10_000
var   diagnostic_limit_erros = 520
var   diagnostic_limit_lines = 1_000_000

type diagnostic struct {
  Context
  sync.Mutex
  newlines []*diagPoint
  points   []*diagPoint
  nested [][]*diagPoint // TODO: this shall perish
  erros int // number of flushed erros
  flued int
  traced int
}
func (diag *diagnostic) aquire() (unlock func()) { diag.Lock(); return func(){ diag.Unlock() }}
func (diag *diagnostic) cast(t reflect.Type) Context { return implcast(diag,t) }
func (diag *diagnostic) do(ctx Context, op any) any {
  switch t := op.(type) {
  case actCountDiag: return diag.count(t.t...)
  case property: if t&propErros != 0 { return diag.erros }
  }
  if diag.Context == nil { return nil }
  return diag.Context.do(ctx, op)
}
func (diag *diagnostic) reset() { defer diag.aquire()(); diag.points = []*diagPoint{} }
func (diag *diagnostic) add(point *diagPoint) *diagPoint {
  defer diag.aquire()()

  if diagnostic_limit < len(diag.points)+len(diag.newlines) {
    panic("too many diagnostics")
  } else if 0 < diagnostic_limit_lines {
    var x = diag.flued
    for _, t := range append(diag.points, diag.newlines...) {
      x += 1 + bytes.Count(t.stack, []byte("\n"))
    }
    if diagnostic_limit_lines < x {
      panic(fmt.Sprintf("too many diagnostics (%d)", x))
    }
  }

  if point.dt == diagPromptLine {
    diag.newlines = append(diag.newlines, point)
    return point
  } else if strings.HasSuffix(point.message, "\n") {
    if diag.points = append(diag.points, point); diag.newlines != nil {
      diag.points = append(diag.points, diag.newlines...)
      diag.newlines = nil
    }
    return point
  } else {
    diag.points = append(diag.points, point)
    return point
  }
}
func (diag *diagnostic) nest(points []*diagPoint) {
  defer diag.aquire()()
  diag.nested = append(diag.nested, points)
}
func (diag *diagnostic) point(ctx Context, dt diagType, f string, args ...any) *diagPoint {
  if dt != diagPrompt { f = strings.TrimSpace(f) }
  return diag.add(&diagPoint{ dt, ctx.Position(), fmt.Sprintf(f, args...), nil })
}
func (diag *diagnostic) error() bool { return diag.countError() > 0 }
func (diag *diagnostic) countError() int { return diag.count(diagError) }
func (diag *diagnostic) count(dt ...diagType) (errs int) {
  defer diag.aquire()()
  for _, d := range diag.points {
    for _, t := range dt {
      if d.dt == t { errs += 1 ; break }
    }
  }
  return
}
func (diag *diagnostic) flush(ctx Context) (errs int) {
  var flush = func(d *diagPoint, pend bool) bool {
    defer func() {
      if x := diagnostic_limit_erros ; 0 < x && x < diag.erros {
        panic(fmt.Sprintf("too many errors (%d)", diag.erros))
      }
      if x := diagnostic_limit_lines ; 0 < x && x < diag.flued {
        panic(fmt.Sprintf("too many diagnostics (%d)", x))
      }
    } ()

    pos, msg := d.position.String(), d.message

    diag.flued += 1

    switch d.dt {
    case diagError: fmt.Fprintf(stderr, "%v:error: %s\n",   pos, msg); errs += 1
    case diagInfo : fmt.Fprintf(stderr, "%v:info: %s\n",    pos, msg)
    case diagWarn : fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
    case diagPromptLine: if msg != "" { fmt.Fprintf(stderr, "%s\n", msg) }
    case diagPrompt    : if msg != "" { fmt.Fprintf(stderr, "%s"  , msg) }
      if pend && !strings.HasSuffix(msg, "\n") { return true }
    }

    if d.stack != nil {
      fmt.Fprintf(stderr, "%s\n", bytes.TrimSpace(d.stack))
      diag.flued += 1 + bytes.Count(d.stack, []byte("\n"))
    }
    return false
  }

  defer func() {
    diag.erros += errs
    do(ctx, actOnErros{errs})
  } ()

  for {
    var point *diagPoint

    diag.Lock()
    if len(diag.points) > 0 {
      point = diag.points[0]
      diag.points = diag.points[1:]
    }
    diag.Unlock()

    if point == nil || flush(point, true) { break }
    if errs > 49 {
      fmt.Fprintf(stderr, "%v: too many errors (%d)\n", diag.Position(), errs)
      break
    }
  }

  diag.Lock()
  for i := 0; len(diag.nested) > 0; diag.nested = diag.nested[1:] {
    i += 1
    fmt.Fprintf(stderr, "\n#%d:\n", i)
    for _, d := range diag.nested[0] { flush(d, false) }
    fmt.Fprintf(stderr, "#%d;\n\n", i)
  }
  diag.Unlock()
  return
}

func flush(ctx Context) int { return _diagnostic(ctx).flush(ctx) }

func diag(ctx Context, dt diagType, f string, a ...any) (_ *diagPoint) {
  if _diag := _diagnostic(ctx) ; _diag != nil {
    return _diag.point(ctx, dt, f, a...)
  }
  return
}
func info(ctx Context, f string, a ...any) *diagPoint { return diag(ctx, diagInfo, f, a...) }
func warn(ctx Context, f string, a ...any) *diagPoint { return diag(ctx, diagWarn, f, a...) }
func erro(ctx Context, f string, a ...any) *diagPoint { return diag(ctx, diagError, f, a...) }
func prompt(ctx Context, f string, a ...any) *diagPoint { return diag(ctx, diagPrompt, f, a...) }

func note(ctx Context, f string, a ...any) *diagPoint {
  if !strings.HasSuffix(f, "\n") { f += "\n" }
  if false {
    return prompt(ctx, "%v: "+f, append([]any{ctx.Position()}, a...)...)
  } else {
    return prompt(ctx, ctx.Position().String()+": "+f, a...)
  }
}

func infostack(ctx Context, n int, a ...any) *diagPoint { return diagstack(ctx, n, diagInfo  , a...) }
func warnstack(ctx Context, n int, a ...any) *diagPoint { return diagstack(ctx, n, diagWarn  , a...) }
func errostack(ctx Context, n int, a ...any) *diagPoint { return diagstack(ctx, n, diagError , a...) }
func promstack(ctx Context, n int, a ...any) *diagPoint { return diagstack(ctx, n, diagPrompt, a...) }
func notestack(ctx Context, n int, a ...any) *diagPoint {
  if true {
    var f string
    if len(a) > 0 { if s, y := a[0].(string); y {
      f, a = s, a[1:]
    }}

    a = append([]any{"%v: "+f+"\n", ctx.Position()}, a...)
  }
  return diagstack(ctx, n, diagPrompt, a...)
}
func diagstack(ctx Context, n int, dt diagType, a ...any) (point *diagPoint) {
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
        s = get_project(ctx).name + " ; " + ts(_p)
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
func (p *positional) caller() *positional { return _positional(p.Context) }
func (p *positional) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *positional) ts(string) string { return ts(p.Context) }
func (p *positional) do(ctx Context, op any) any {
  switch t := op.(type) {
  case property:
    if t&propPosition != 0 { return p.position }
  }
  if p.Context == nil { return nil }
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
    if x, y := t.do(ctx, propPosition).(Position); y&&x.valid() { pos = x }
  default:
    if false { erro(ctx, "non-position arg: %v", ts(a)).debug(3) }
    return ctx
  }

  if p := ctx.Position(); p.valid() && pos.valid() && !p.Same(&pos) {
    for c, i, n := ctx, 0, 0; c != nil; c, i = inner(c), i+1 {
      if _, y := c.(*positional); y {
        if n += 1; n > /* 999 */100 {
          if false { prompt(ctx, "%v: too many positions: %T\n", p, c) }
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
  if strings.HasSuffix(dir, entryFileName) || strings.HasSuffix(dir, "build.smart") {
    pos.Filename = dir
  } else if _, e := os.Stat(filepath.Join(dir, entryFileName)); e == nil {
    pos.Filename = filepath.Join(dir, entryFileName)
  } else if _, e := os.Stat(filepath.Join(dir, "build.smart")); e == nil {
    pos.Filename = filepath.Join(dir, "build.smart")
  } else {
    pos.Filename = dir
  }
  pos.Line = 1
  return
}

func assured(ctx Context, dontCheckErrors ...bool) (recovered, errs int) {
  var te tracend
  var f *failure
  for e := recover(); e != nil; e = recover() {
    switch recovered += 1; t := e.(type) {
    case bailout: continue
    case tracend: t.Context = nil ; continue
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
    promstack(ctx, 5, "%v: %s (%d panics)", ctx.Position(), ts(te.Context), recovered).debug(128)
  }

  te.Context = nil

  var dia = _diagnostic(ctx) ; dia.flush(ctx)
  if len(dontCheckErrors) > 0 && dontCheckErrors[0] { return }

  if errs = dia.countError(); 0 < errs && recovered == 0 {
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
  if dia.countError() > 0 { return }

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
  } else if result, err := context.run(); err != nil {
    erro(context, "run work failed: %v", err)
  } else if dia.flush(context) > 0 {
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
