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
  "time"
  "fmt"
  "os"
  "io"
)

const (
  productVerTag = "dev" // dev, alpha, beta, final
  checkpoints = productVerTag != "final"
  trace_recover = true
)

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

  silentOptionalSelection bool

  verbose         bool `v,verb,verbose`
  verboseBreaks   bool `vb,vbrk,verbose-breaks`
  verboseChecks   bool `vc,vchk,verbose-checks`
  verboseImport   bool `vi,vimp,verbose-import`
  verboseLoads    bool `vl,vloa,verbose-loading`
  verboseParse    bool `vp,vpar,verbose-parsing`
  verboseUsing    bool `vu,vuse,verbose-using`
  verboseExecFlags bool `vxf,verbose-exec-flag`

  allowClosureFilemap bool `cf,closure-filemap,closure-files`

  cleanDotCache   bool `clcac,clean-cache,clear-cache;rmc,rm-cache`
  cleanDotDeps    bool `cldep,clean-deps,clear-deps;rmd,rm-deps`
  cleanDotGrep    bool `clgrp,clean-grep,clear-grep;rmg,rm-grep`
  cleanTmpDirs    bool `cltmp,clean-temp,clear-temp;rmt,rm-temp`

  checkLoadGraph  bool `ckld,check-loads`

  configure       bool `c,con,conf,configure`               // optionConfigure
  reconfigure     bool `rc,rec,reconf,reconfig,reconfigure` // optionReconfig

  saveGrepSource  bool `savgs,save-grep-source`

  noRun           bool `nor,no-run`
  noExec          bool `nox,ne,no-exec,no-execute`  // optionNoExec
  noDeps          bool `nod,no-deps`
  noGrep          bool `nog,no-grep`
  noDepsGrep      bool `nodg,ngd,no-deps-grep,no-grep-deps`
  noImportFiles   bool `noif,no-import-files`

  slow time.Duration `sl,slow` // time.Millisecond

  parallel        bool `p,par,para,parallel`

  fastMode        bool `f,fm,fast,fast-mode`
  failOnErrors    bool `fe,foe,fail-on-errors`
  errorUncache    bool `eu,error-uncache,error-no-cache`

  traceLaunch     bool `tl,trace-launch`
  traceParsing    bool `tp,trace-parse`
  traceExecutor   bool `te,trace-executor`
  traceExec       bool `tx,trace-exec`
  traceEntering   bool `ti,trace-entering`
  traceConfig     bool `tc,trace-config`
}

func debugSyntax(ctx Context, s string) (res bool) {
  if u := _universe(ctx); u != nil && u.ddd == s {
    for _, t := range u.debugSyntax { if res = t == s; res { break } }
  }
  return
}

func db(ctx Context, ss ...string) (res bool) {
    for _, d := range strings.Fields(_universe(ctx).ddd) {
        for _, s := range ss { if d == s { return true }}
    }
    return
}

func cl(ctx Context) (res *commandline) {
  if u := _universe(ctx); u != nil { res = &u.commandline }
  return
}

const fullContextStringer = false

type property uint

const (
  propPosition property = 1<<iota
  propProgram
  propParameters
  propParamName
  propWorkDir
  propExAuto
  propExClosure
  propExDelegate
  propExDef // =, :=, ::=, ...
  propExDef0      //   =
  propExDef1      //  :=
  propExDef2      // ::=
  propExDef3      // ;:= (TODO)
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
  propCacheUnmap
  propCachePath
)

func _position(ctx Context) (res Position) {
  if i := ctx.do(propPosition); i == nil {
    erro(ctx, "no such property: position, %v", us(ctx)).debug(24)
  } else if t, y := i.(Position); y { res = t } else {
    erro(ctx, "not position: %v", us(i)).debug(2)
  }
  return
}

func _programProp(ctx Context) (res *program) {
  if i := ctx.do(propProgram); i == nil {
    erro(ctx, "no such property: program, %v", us(ctx)).debug(24)
  } else if t, y := i.(*program); y { res = t } else {
    erro(ctx, "not program: %v", us(i)).debug(2)
  }
  return
}

func _parameters(ctx Context) (res map[string]*auto) {
  if i := ctx.do(propParameters); i != nil {
    if t, y := i.(map[string]*auto); y { res = t }
  }
  return
}

func _paramName(ctx Context, n int) (res string) {
  if i := ctx.do(propParamName, n); i != nil {
    if s, y := i.(string); y { res = s }
  }
  return
}

func _workdir(ctx Context) (res string) {
  if i := ctx.do(propWorkDir); i == nil {
    erro(ctx, "no such property: workdir, %v", us(ctx)).debug(24)
  } else if t, y := i.(string); y { res = t } else {
    erro(ctx, "not string: %v", us(i)).debug(2)
  }
  return
}

func _exAuto(ctx Context) (res bool) {
  res, _ = ctx.do(propExAuto).(bool)
  return
}

func _exClosure(ctx Context, x Value) (res bool) {
  res, _ = ctx.do(propExClosure, x).(bool)
  return
}

func _exDelegate(ctx Context, x Value) (res bool) {
  res, _ = ctx.do(propExDelegate, x).(bool)
  return
}

func _exDef(ctx Context) (res bool) {
  res, _ = ctx.do(propExDef).(bool)
  return
}

func _exDef0(ctx Context) (res bool) {
  res, _ = ctx.do(propExDef0).(bool)
  return
}

func _exDef1(ctx Context) (res bool) {
  res, _ = ctx.do(propExDef1).(bool)
  return
}

func _exDef2(ctx Context) (res bool) {
  res, _ = ctx.do(propExDef2).(bool)
  return
}

func _exDefValue(ctx Context) (res bool) {
  res, _ = ctx.do(propExDefValue).(bool)
  return
}

func _exDigital(ctx Context) (res bool) {
  res, _ = ctx.do(propExDigital).(bool)
  return
}

func _exDisjunction(ctx Context) (res bool) {
  res, _ = ctx.do(propExDisjunction).(bool)
  return
}

func _exFullFile(ctx Context) (res bool) {
  res, _ = ctx.do(propExFullFile).(bool)
  return
}

func _exEvaluation(ctx Context) (res bool) {
  res, _ = ctx.do(propExEvaluation).(bool)
  return
}

func _exPairVal(ctx Context) (res bool) {
  res, _ = ctx.do(propExPairVal).(bool)
  return
}

func _exPathStr(ctx Context) (res bool) {
  res, _ = ctx.do(propExPathStr).(bool)
  return
}

func _exPlaceholder(ctx Context) (res bool) {
  res, _ = ctx.do(propExPlaceholder).(bool)
  return
}

func _exCondless(ctx Context) (res bool) {
  res, _ = ctx.do(propExCondless).(bool)
  return
}

func _exFinal(ctx Context) (res bool) {
  res, _ = ctx.do(propExFinal).(bool)
  return
}

type Context interface {
  Position() Position

  String() string

  Globe() *globe

  cast(reflect.Type) Context
  closure() []*Scope

  Scope() *Scope
  project() *project
  projects(Context,...*project) []*project

  dirtyMark(...Value)
  dirtyOpts() *dirtyOpts
  dirty(Context,...Value) bool

  traversed(Context, Value) []Value
  traverse(Context, Value) travestates

  ref(Context, Value) bool

  do(property,...interface{}) interface{}
}

func cast[Ctx Context](ctx Context) (c Ctx) {
  if ctx != nil {
    var t = ctx.cast(reflect.TypeOf(c))
    if t != nil { c = t.(Ctx) }
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

func _inner(v reflect.Value) (i interface{}) {
  if t := v.Type(); t.Kind() == reflect.Struct {
    if f, y := t.FieldByName("Context"); y && f.Anonymous {
      if v = v.FieldByIndex(f.Index); v.IsValid() { i = v.Interface() }
    } else if f, y := v.Interface().(interface{ inner() Context }); y && f != nil {
      i = f.inner()
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

func getTargetValue(ctx Context) (res Value) {
  if val := autoVal(ctx, "@"); val == nil {
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
  callstackSmartTrace = regexp.MustCompile(`^(?:extbit\.io/)?(?:.+?)smart\.\(\*diagnostic\)\.trace\(.+\)$`)
)
func _callstack(s string, i, j int, args ...interface{}) (res callstack) {
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
    if callstackSmartTrace.Match(v[i]) { continue }

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

func cstack(i, j int, a ...interface{}) callstack { return _callstack("", i+1, j, a...) }

type diagType int

const (
  diagInfo diagType = iota
  diagWarn
  diagError
  diagPrompt
  diagPromptNewline
)

type diagPoint struct {
  dt diagType
  position Position
  message string
  stack []byte // see also debug.Stack()
}
func (d *diagPoint) debug(args ...interface{}) *diagPoint {
  if d == nil {
    panic("nil diag point")
  }

  switch productVerTag {
  case "dev", "debug": // only print debug diags for dev and debug versions
  default: return d
  }

  var s string
  switch d.dt {
  case diagPrompt: if /* !options.debugPrompt */false { return d } else { s = "note:" }
  case diagInfo:   if /* !options.debugInfos  */false { return d } else { s = "info:" }
  case diagWarn:   if /* !options.debugWarns  */false { return d } else { s = "info:" }
  case diagError:  if /* !options.debugErrors */false { return d } else { s = "info:" }
  }

  var i = 5 // skips the standard stack lines, which are not informative
  var j = 0 // number of frames to dump
  d.stack = _callstack(s, i, j, args...)
  return d
}

type tracend struct { Context }
func trace(ctx Context, a ...interface{}) {
  if trace_recover {
    var te tracend
    var recovered int
    for e := recover(); e != nil; e = recover() {
      switch recovered += 1; t := e.(type) {
      case       bailout:
      case       tracend: te = t
      case       failure: erro(t.Context, t.Error()) ; t.Context = nil
      case         Value: erro(at(ctx,t), "trace: %s", us(t))
      case        string: erro(   ctx   , "trace: %s", t)
      case runtime.Error: erro(   ctx   , "trace: %s", t.Error())
      default:            erro(   ctx   , "trace: %s", us(e))
      }
    }
    if 0 < recovered {
      erro(ctx, "%s (%d panics)", us(te.Context), recovered).debug(32)
    }
    te.Context = nil
  }

  if d := _diagnostic(ctx); d.error() {
    if d.traced += 1 ; 1 == d.traced {
      panic(tracend{ctx}) // break out of the call stack
    }
  }
  return
}

func _diagnostic(c Context) *diagnostic { return cast[*diagnostic](c) }

type diagnostic struct {
  Context
  sync.Mutex
  newlines []*diagPoint
  points   []*diagPoint
  nested [][]*diagPoint
  errs, traced int
}
func (diag *diagnostic) aquire() (unlock func()) { diag.Lock(); return func(){ diag.Unlock() }}
func (diag *diagnostic) cast(t reflect.Type) Context { return implcast(diag,t) }
func (diag *diagnostic) String() string {
  if fullContextStringer {
    return fmt.Sprintf("diag{%s}", diag.Context)
  } else {
    return diag.Context.String()
  }
}
func (diag *diagnostic) reset() { defer diag.aquire()(); diag.points = []*diagPoint{} }
func (diag *diagnostic) add(point *diagPoint) *diagPoint {
  defer diag.aquire()()
  if point.dt == diagPromptNewline {
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
func (diag *diagnostic) point(ctx Context, dt diagType, f string, args ...interface{}) *diagPoint {
  if dt != diagPrompt { f = strings.TrimSpace(f) }
  return diag.add(&diagPoint{ dt, ctx.Position(), fmt.Sprintf(f, args...), nil })
}
func (diag *diagnostic) error() bool { return diag.errs > 0 || diag.countError() > 0 }
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
func (diag *diagnostic) flush() (errs int) {
  var flush = func(d *diagPoint, pend bool) (pended bool) {
    var (
      pos = d.position.String()
      msg = d.message
    )

    switch d.dt {
    case diagError: fmt.Fprintf(stderr, "%v: %s\n",         pos, msg); errs += 1
    case diagInfo : fmt.Fprintf(stderr, "%v:info: %s\n",    pos, msg)
    case diagWarn : fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
    case diagPromptNewline: if msg != "" { fmt.Fprintf(stderr, "%s\n", msg) }
    case diagPrompt  : if msg != "" { fmt.Fprintf(stderr, "%s", msg) }
      if pend && !strings.HasSuffix(msg, "\n") { return true }
    }

    if len(d.stack) > 0 { fmt.Fprintf(stderr, "%s\n", bytes.TrimSpace(d.stack)) }
    return
  }

  defer func() { diag.errs += errs } ()

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

func flush(ctx Context) int { return _diagnostic(ctx).flush() }

func diag(ctx Context, dt diagType, f string, a ...interface{}) (_ *diagPoint) {
  if _diag := _diagnostic(ctx) ; _diag != nil {
    return _diag.point(ctx, dt, f, a...)
  }
  return
}
func info(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagInfo, f, a...) }
func warn(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagWarn, f, a...) }
func erro(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagError, f, a...) }
func prompt(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagPrompt, f, a...) }

func noted(ctx Context, f string, a ...interface{}) *diagPoint {
  if !strings.HasSuffix(f, "\n") { f += "\n" }
  if false {
    return prompt(ctx, "%v: "+f, append([]interface{}{ctx.Position()}, a...)...)
  } else {
    return prompt(ctx, ctx.Position().String()+": "+f, a...)
  }
}

func infostack(ctx Context, n int, a ...interface{}) *diagPoint { return diagstack(ctx, n, diagInfo  , a...) }
func warnstack(ctx Context, n int, a ...interface{}) *diagPoint { return diagstack(ctx, n, diagWarn  , a...) }
func errostack(ctx Context, n int, a ...interface{}) *diagPoint { return diagstack(ctx, n, diagError , a...) }
func promstack(ctx Context, n int, a ...interface{}) *diagPoint { return diagstack(ctx, n, diagPrompt, a...) }
func notestack(ctx Context, n int, a ...interface{}) *diagPoint {
  if true {
    var f string
    if len(a) > 0 { if s, y := a[0].(string); y {
      f, a = s, a[1:]
    }}

    a = append([]interface{}{"%v: "+f+"\n", ctx.Position()}, a...)
  }
  return diagstack(ctx, n, diagPrompt, a...)
}
func diagstack(ctx Context, n int, dt diagType, a ...interface{}) (point *diagPoint) {
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
        s = _p.project().name + " ; " + us(_p)
      } else {
        s = us(_p)
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

type positional struct { Context; position Position }
func (pc *positional) caller() *positional { return _positional(pc.Context) }
func (pc *positional) cast(t reflect.Type) Context { return implcast(pc, t) }
func (pc *positional) Position() Position { return pc.position }
func (pc *positional) String() string {
  if fullContextStringer {
    return fmt.Sprintf("positional{%s}", pc.Context)
  } else {
    return pc.Context.String()
  }
}
func (pc *positional) do(prop property, a ...interface{}) interface{} {
  if prop == propPosition { return pc.position }
  if pc.Context != nil { return pc.Context.do(prop, a...) }
  return nil
}

func _at(ctx Context, p Position) Context { return &positional{ctx, p} }
func at(ctx Context, a interface{}) Context {
  if ctx == nil { panic("nil context") }

  var pos Position

  switch t := a.(type) {
  case positioner: pos = t.Position()
  case Position  : pos = t
  default:
    if false { erro(ctx, "non-position arg: %v", us(a)).debug(3) }
    return ctx
  }

  if p := ctx.Position(); p._valid() && pos._valid() && !p.Same(&pos) {
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

func _argumentedContext(c Context) *argumentedContext { return cast[*argumentedContext](c) }

type argumentedContext struct { Context ; args []Value }
func (ac *argumentedContext) cast(t reflect.Type) Context { return implcast(ac,t) }
func (ac *argumentedContext) argumented() *argumentedContext { return ac }
func (ac *argumentedContext) String() string {
  if fullContextStringer {
    return fmt.Sprintf(`argumented{%s}`, ac.Context)
  } else {
    return ac.Context.String()
  }
}

func executeEntry(ctx Context, entry *rule, args ...Value) (result []Value, okay bool) {
  var traves travestates
  if result, traves = entry.execute(at(ctx, entry.position), args...); !traves.has() {
    return result, true
  }

  if t := traves.of(traveFail); t.has() {
    for _, brk := range t { erro(at(ctx,brk.pos), "%v: %v", entry, brk).debug(1) }
    traves = traves.not(traveFail)
    return result, false
  }

  if t := traves.of(traveCase, traveDone, traveNext, traveRule, traveFile); t.has() {
    traves = traves.not(traveCase, traveDone, traveNext, traveRule, traveFile)
  }

  if traves.has() {
    for _, brk := range traves { erro(at(ctx,brk.pos), "%v: %v", entry, brk).debug(1) }
  } else {
    okay = true
  }
  return
}

func updateGoal(ctx Context, goal Value, args []Value) (result []Value) {
  if isNull(goal) {
    // TODO: report nil goal
  } else {
    var okay bool
    switch g := goal.(type) {
    case *rule:
      if result, okay = executeEntry(at(ctx, g.position), g, args...); !okay {
        erro(at(ctx,ctx.Position()), "update '%v' failed", g).debug(1)
      }
    default:
      erro(at(ctx,goal), "'%v' is not an entry (%T)", goal, goal).debug(1)
    }
  }
  return
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

// usage: defer assured(ctx, ...)
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
    case         Value: erro(at(ctx,t), "assured: %s", us(t))
    case        string: erro(   ctx   , "assured: %s", t)
    case runtime.Error: erro(   ctx   , "assured: %s", t.Error())
    default:            erro(   ctx   , "assured: %s", us(e))
    }
  }
  if 0 < recovered {
    // if defer assured from top stack, this will dump the full stack of panics
    promstack(ctx, 5, "%v: %s (%d panics)", ctx.Position(), us(te.Context), recovered).debug(128)
  }
  te.Context = nil

  var dia = _diagnostic(ctx) ; dia.flush()
  if len(dontCheckErrors) > 0 && dontCheckErrors[0] {
    return
  }

  if errs = dia.countError(); errs > 0 && recovered == 0 {
    noted(ctx, "got %d errors (total %d, recovered %d)", errs, dia.errs, recovered).debug(10)
    if f != nil && (len(dontCheckErrors) == 0 || !dontCheckErrors[0]) {
      panic(_failure(ctx, "fail [assured]"))
    } else {
      dia.flush()
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
  } else if dia.flush() > 0 {
    prompt(context, "loading work got %d errors\n", dia.errs)
  } else if context.help {
    context.doHelp()
  } else if context.printFlags {
    context.doHelpFlags()
  } else if context.printConfig {
    context.doHelpConfig()
  } else if numUpdatedPlugins > 0 { // see buildPlugin
    prompt(context, "plugins updated, please relaunch.\n")
  } else if context.commandline.configure {
    configure(context)
  } else if result, err := context.run(); err != nil {
    erro(context, "run work failed: %v", err)
  } else if dia.flush() > 0 {
    prompt(context, "run work got %d errors\n", dia.errs)
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
