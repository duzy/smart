//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "path/filepath"
  "runtime/debug"
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
  clocks = "🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧"
  productVerTag = "dev" // dev, alpha, beta, stable
)

type commandline struct {
  help            bool `h,help`

  debug           bool `d,db,debug`
  debugErrors     bool `de,dberro,debug-errors`
  debugWarns      bool `dw,dbwarn,debug-warns`
  debugInfos      bool `di,dbinfo,debug-infos`
  debugPrompt     bool `dp,dbprom,debug-prompt`
  debugFileEntry  bool `debug-file-entry`
  debugFiles  []string `df,dbfile,debug-file`
  debugSyn    []string `ds,dbsyntax,debug-syntax`

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

  cleanConf       bool `cc,clean-conf,clean-configure`
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

func (o *commandline) debugParsing(ctx Context, syntax string) (res bool) {
  if ctx.universe().ddd == syntax { for _, s := range o.debugSyn {
    if res = s == syntax; res { break }
  }}
  return
}

const fullContextStringer bool = false

type Context interface {
  Position() Position

  String() string // for debug
  WorkDir() string

  Scope() *Scope
  Globe() *Globe

  aquireLock() (unlock func())

  universe() *universe
  // unmap(Context, interface{}) []matchedFileMap
  // cache(Context, []Value, []Value)

  loader() *loader // only in load stage
  parser() *parser // only in parse stage

  inner() Context
  cast(reflect.Type) Context

  argumented() *argumentedContext
  dia() *diaContext

  ac() *autoContext
  rc() *refContext
  ic() *invocation

  closure() *closureContext
  closureScopes() []*Scope

  Project() *Project
  projects(Context, ...*Project) []*Project

  poco() *positionContext
  pc() *programContext
  program() *program

  dirtyMark(...Value)
  dirtyOpts() *dirtyOpts
  dirty(ctx Context, args ...Value) bool

  traversed(ctx Context, target Value) []Value
  traverse(ctx Context, prereqValue Value) (traves travestates)

  ruleContext() *ruleContext
  entry() Entry

  sc() *stemmedContext
  stems() []string

  // TODO: call(Context, string, facet, []Value, ...Value) Value

  ref(Context, Value) bool

  appendCallerUpdated() bool
  isConfigure() bool
  mustExists() bool
}

func cast[C Context](ctx Context) (res C) {
  if t := ctx.cast(reflect.TypeOf(res)); t != nil {
    if c, y := t.(C); y { res = c } else { errostack(ctx, 3, "%T", t).debug(10) }
  }
  return
}

func getTargetValue(ctx Context) (res Value) {
  if val := autoVal(ctx, "@"); val == nil {
    if false { erro(ctx, "target is nil") }
  } else if vals, u, n := plain.expand(ctx, val); len(vals) == 1 {
    res = scalarize(vals[0])
  } else {
    erro(of(ctx,val), "multiple targets: %v → %v (%d,%d)", val, vals, u, n)
  }
  return
}

func getTargetValueString(ctx Context) (val Value, str string) {
  if val = getTargetValue(ctx); isNull(val) {
    if false { erro(ctx, "target '%v' is nil", val) }
  } else {
    str, _ = as{val}.fullnameOrStrval(ctx)
  }
  return
}

type diagType int

const (
  diagInfo diagType = iota
  diagWarn
  diagError
  diagPrompt
  diagPromptNL
)

var (
  goStackLine1 = regexp.MustCompile(`^(?:extbit\.io/)?(.+)(\(.*\))$`)
  goStackLine2 = regexp.MustCompile(`^	(.*?:\d+)(?: \+.*)?$`)
  goStackSmartTrace = regexp.MustCompile(`^(?:extbit\.io/)?(?:.+?)smart\.\(\*diaContext\)\.trace\(.+\)$`)
)
type skipint struct{ int }
type frames struct{ int }
type diagPoint struct {
  dt diagType
  position Position
  message string
  stack []byte // see also debug.Stack()
}
func (d *diagPoint) debug(args ...interface{}) *diagPoint {
  switch productVerTag {
  case "dev", "debug": // only print debug diags for dev and debug versions
  default: return d
  }

  var s string
  switch d.dt {
  case diagPrompt: if /* !options.debugPrompt */false { return d } else { s = "note:" }
  case diagInfo:   if /* !options.debugInfos */false  { return d } else { s = "info:" }
  case diagWarn:   if /* !options.debugWarns */false  { return d } else { s = "info:" }
  case diagError:  if /* !options.debugErrors */false { return d } else { s = "info:" }
  }

  var (
    nums []int
    i = 5 // skips the standard stack lines, which is not useful
    j = 0 // number of frames to dump
  )
  for _, a := range args { if t, y := a.(bool); y {
    if !t { return d }
  } else if t, y := a.(int); y {
    nums = append(nums, t)
  } else if t, y := a.(skipint); y {
    i += t.int
  } else if t, y := a.(frames); y {
    j += t.int
  }}
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

  var ln = []byte{ '\n' }
  var v = bytes.Split(debug.Stack(), ln)
  if true {
    var gotPanic bool
    for ; 0 < j && i+1 < len(v); i = i+1 {
      // skip diaContext.trace lines
      if goStackSmartTrace.Match(v[i]) { continue }

      var (
        sm1 = goStackLine1.FindAllSubmatch(v[i+0], 1)
        sm2 = goStackLine2.FindAllSubmatch(v[i+1], 1)
        isPanic = len(sm1) > 0 && len(sm1[0]) > 1 && bytes.Equal(sm1[0][1], []byte("panic"))
        se string
      )
      if gotPanic { se = "		<---- panic" }
      if sm1 != nil && sm2 != nil && !isPanic {
        var e string
        if 0 < j-1 && i+3 < len(v) { e = se+"\n" } else { e = " ...\n" }
        d.stack = append(d.stack, sm2[0][1]...)
        d.stack = append(d.stack, []byte(":"+s+" ")...)
        d.stack = append(d.stack, sm1[0][1]...)
        d.stack = append(d.stack, sm1[0][2]...)
        d.stack = append(d.stack, []byte(e)...)
        j -= 1
      }
      gotPanic = isPanic
    }
  } else {
    d.stack = bytes.Join(v[i:], ln)
  }
  return d
}

type diaContext struct {
  Context
  sync.Mutex
  points   []*diagPoint
  nested [][]*diagPoint
  errs, traced int
}
func (diag *diaContext) inner() Context { return diag.Context }
func (diag *diaContext) dia() *diaContext { return diag }
func (diag *diaContext) aquireLock() (unlock func()) { diag.Lock(); return func(){ diag.Unlock() }}
func (diag *diaContext) String() string {
  if fullContextStringer {
    return fmt.Sprintf("diag{%s}", diag.Context)
  } else {
    return diag.Context.String()
  }
}
func (diag *diaContext) reset() { defer diag.aquireLock()(); diag.points = []*diagPoint{}}
func (diag *diaContext) add(point *diagPoint) *diagPoint { defer diag.aquireLock()()
  diag.points = append(diag.points, point)
  return point
}
func (diag *diaContext) nest(points []*diagPoint) { defer diag.aquireLock()()
  diag.nested = append(diag.nested, points)
}

func (diag *diaContext) point(ctx Context, dt diagType, f string, args ...interface{}) *diagPoint {
  if dt != diagPrompt { f = strings.TrimSpace(f) }
  return diag.add(&diagPoint{ dt, ctx.Position(), fmt.Sprintf(f, args...), nil })
}

func dtrace(ctx Context, fmt string, a ...interface{}) {
  if diag := ctx.dia(); diag.error() {
    if diag.traced += 1; diag.traced > 1 { return }
    if false { erro(ctx, fmt, a...).debug(3) }
    if len(a) == 0 { a = append(a, ctx.Position()) } else
    if _, y := a[0].(Position); !y { a = append([]interface{}{ctx.Position()}, a...) }
    panic(failure{fmt, a})
  }
  return
}

func (diag *diaContext) error() bool { return diag.errs > 0 || diag.countErrors() > 0 }
func (diag *diaContext) totalErrors() (errs int) { return diag.errs }
func (diag *diaContext) countErrors() (errs int) { return diag.count(diagError) }
func (diag *diaContext) count(dt ...diagType) (errs int) { defer diag.aquireLock()()
  for _, d := range diag.points {
    for _, t := range dt {
      if d.dt == t { errs += 1 ; break }
    }
  }
  return
}
func (diag *diaContext) flush() (errs int) {
  var flush = func(d *diagPoint, pend bool) (pended bool) {
    var (
      pos = d.position.String()
      msg = d.message
    )

    switch d.dt {
    case diagError: fmt.Fprintf(stderr, "%v: %s\n",         pos, msg); errs += 1
    case diagInfo : fmt.Fprintf(stderr, "%v:info: %s\n",    pos, msg)
    case diagWarn : fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
    case diagPromptNL: if msg != "" { fmt.Fprintf(stderr, "%s\n", msg) }
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

func diagnostic(ctx Context) Context { return &diaContext{ Context: ctx } }
func diag(ctx Context, dt diagType, f string, a ...interface{}) (p *diagPoint) {
  if p = ctx.dia().point(ctx, dt, f, a...); false && p != nil { p.position = ctx.Position() }
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
func notestack(ctx Context, n int, a ...interface{}) *diagPoint { var f string
  if len(a) > 0 { if s, y := a[0].(string); y { f, a = s, a[1:] } }
  a = append([]interface{}{ "%v: "+f+"\n", ctx.Position() }, a...)
  return diagstack(ctx, n, diagPrompt, a...)
}
func diagstack(ctx Context, n int, dt diagType, a ...interface{}) (point *diagPoint) {
    var (
        proj = ctx.Project()
        entry = ctx.entry()
        s, str string
    )
    if len(a) > 0 { if t, y := a[0].(string); y { s, a = t, a[1:] }}
    if s != "" && s != "<#" && s != "#>" {
      if len(a) > 0 { if p, y := a[0].(Position); y && p.IsValid() && strings.HasPrefix(s, "%") {
        if strings.HasSuffix(s, "\n") {
          s = strings.TrimSuffix(s, "\n") + "#%d\n"
          a = append([]interface{}{p, n}, a[1:]...)
        }
      } else {
        s = "#%d: " + s
        a = append([]interface{}{n}, a...)
      }}
      point = diag(ctx, dt, s, a...)
    } else if entry == nil {
        if point = diag(ctx, dt, "#%d: in project %v:", n, proj); proj != nil {
            point.position = proj.position
        }
        for last, i := ctx.Position(), ctx.inner(); i != nil; i = i.inner() {
            if pos := i.Position(); !pos.Same(&last) {
                point = diag(ctx, dt, "%v: from here", proj)
                point.position = pos
                last = pos
            }
        }
        return
    } else {
        if proj == nil { proj = entry.OwnerProject() }
        str, _, _ = entryIndicator(ctx, entry)
    }

    if s == "<#" && len(a) > 0 { for _, t := range a {
        if v, ok := t.(Value); ok {
            if str == "" {
                point = diag(ctx, dt, "%v: %v (%T)", proj, v, v)
            } else {
                point = diag(ctx, dt, "%v: %v: %v (%T)", proj, str, v, v)
            }
            point.position = v.Position()
        }
    }}

    if point == nil {
        point = diag(ctx, dt, "%v", proj)
    } else if true {
        // ...
    } else if str == "" {
        point = diag(ctx, dt, "%v: %v", proj, ctx)
    } else {
        point = diag(ctx, dt, "%v: %v: %v", proj, str, ctx)
    }

    if pc := ctx.poco(); pc != nil {
        point.position = pc.position

        if s == "#>" && len(a) > 0 { for _, t := range a {
            if v, ok := t.(Value); ok {
                if str == "" {
                    point = diag(ctx, dt, "%v: %v (%T)", proj, v, v)
                } else {
                    point = diag(ctx, dt, "%v: %v: %v (%T)", proj, str, v, v)
                }
                point.position = v.Position()
            }
        }}

        for last := &pc.position; pc != nil && n > 0; pc = pc.Context.poco() {
            var pos = &pc.position
            if last == pos || last.Same(pos) { continue }  else { n -= 1 }

            var suf string
            if n == 0 && pc.caller() != nil { suf = " ..." }

            if entry := pc.entry(); entry == nil {
                point = diag(ctx, /* dt */diagInfo, "#%d: %v%s", n, proj, suf)
            } else if str, _, _ := entryIndicator(pc, entry); str != "" {
                point = diag(ctx, /* dt */diagInfo, "#%d: %v: %v%s", n, proj, str, suf)
            } else {
                point = diag(ctx, /* dt */diagInfo, "#%d: %v: %v%s", n, proj, entry, suf)
            }

            point.position = *pos
            last = pos
        }
    }
    return
}

type positionContext struct { Context; position Position }
func (pc *positionContext) poco() *positionContext { return pc }
func (pc *positionContext) inner() Context { return pc.Context }
func (pc *positionContext) caller() *positionContext { return pc.Context.poco() }
func (pc *positionContext) Position() Position { return pc.position }
func (pc *positionContext) String() string {
  if fullContextStringer {
    return fmt.Sprintf("positional{%s}", pc.Context)
  } else {
    return pc.Context.String()
  }
}

func of(ctx Context, val Value) Context { return at(ctx, val.Position()) }
func at(ctx Context, pos Position) Context {
  if ctx == nil { panic("nil context") } else
  if p := ctx.Position(); p._valid() && pos._valid() && !p.Same(&pos) {
    for c, i, n := ctx, 0, 0; c != /* ctx.universe() */nil; c, i = c.inner(), i+1 {
      if _, y := c.(*positionContext); y { n += 1 ; if n > /* 999 */100 {
        if false { prompt(ctx, "%v: too many positions: %T\n", p, c) }
        warnstack(ctx, 3, "too many positions: %v/%v; %v", n, i, ctx).debug(16)
        ctx.dia().flush()
        return ctx
      }}
    }
    ctx = _at(ctx, pos)
  }
  return ctx
}
func _at(ctx Context, pos Position) Context { return &positionContext{ ctx, pos } }

type argumentedContext struct { Context ; args []Value }
func (ac *argumentedContext) inner() Context { return ac.Context }
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
      erro(of(ctx,goal), "'%v' is not an entry (%T)", goal, goal).debug(1)
    }
  }
  return
}

func walkSmartBaseDirs(ctx Context, cwd string, vis func(string)bool) (s string) {
  s = cwd
  for s != "" {
    file := stat(ctx, ".smart", "", s)
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
    } else if t, _ := filepath.Rel(baseTmpPath, base); strings.HasPrefix(t, ".smart"+PathSep) {
      // In case like '/foo/bar/.smart/a/b/x'+'a/e/f/x', we set
      // base to '/foo/bar/.smart' to produce 'foo/bar/.smart/tmp/a/e/f/x'.
      v1 := strings.Split(t, PathSep)
      v2 := strings.Split(s, PathSep)
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
  if s := ".smart"+PathSep; strings.HasPrefix(rel, s) { // .smart/
    rel = strings.TrimPrefix(rel, s)
    if s = "modules"+PathSep; strings.HasPrefix(rel, s) { // modules/
      rel = strings.TrimPrefix(rel, s)
    }
  }
  rel = strings.Replace(rel, "..", "_", -1)
  if strings.HasPrefix(rel, "tmp"+PathSep) {
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
  var f *failure
  for e := recover(); e != nil; e = recover() {
    switch recovered += 1; t := e.(type) {
    case bailout: continue
    case Value: erro(at(ctx,t.Position()), "%v %v", t, t).debug(1)
    case failure: erro(t.at(ctx), "[failure] "+t.fmt, t.ia()...).debug(1); if f == nil { f = &t }
    default: erro(ctx, "[assured: %T]: %v", e, e).debug(1)
    }
  }

  if recovered > 0 { var pos = ctx.Position()
    if !strings.HasSuffix(pos.Filename, entryFileName) {
      var s = filepath.Join(pos.Filename, entryFileName)
      if _, e := os.Stat(s); e == nil { pos.Filename = s }
    } else if !strings.HasSuffix(pos.Filename, "build.smart") {
      var s = filepath.Join(pos.Filename, "build.smart")
      if _, e := os.Stat(s); e == nil { pos.Filename = s }
    }
    // if defer assured from top stack, this will dump the full stack of panics
    errostack(ctx, 5, "failed, %d recovered", recovered).debug(/*1,*/128)
  }

  var dia = ctx.dia() ; dia.flush()
  if len(dontCheckErrors) > 0 && dontCheckErrors[0] {
    return
  }

  if errs = dia.countErrors(); errs > 0 && recovered == 0 { t := dia.totalErrors()
    noted(ctx, "got %d errors (total %d, recovered %d)", errs, t, recovered).debug(10)
    if f != nil && (len(dontCheckErrors) == 0 || !dontCheckErrors[0]) {
      panic(failure{"fail [assured]",ia(ctx.Position())})
    } else {
      dia.flush()
    }
  }
  return
}

func CommandLine() { var context = init_universe() ; defer assured(context, false)
  if context.traceLaunch { defer un(trace(t_launch, "CommandLine")) }

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

  if context.dia().countErrors() > 0 { return }

  if false { loadGrepCache(context) }

  if err := context.loadTopWork(); err != nil {
    erro(context, "loading work failed: %v", err)
  } else if context.dia().flush() > 0 {
    prompt(context, "loading work got %d errors\n", context.dia().totalErrors())
  } else if context.help {
    context.doHelp()
  } else if context.printFlags {
    context.doHelpFlags()
  } else if context.printConfig {
    context.doHelpConfig()
  } else if numUpdatedPlugins > 0 { // see buildPlugin
    prompt(context, "plugins updated, please relaunch.\n")
  } else if context.commandline.configure {
    context.configure(context)
  } else if result, err := context.run(); err != nil {
    erro(context, "run work failed: %v", err)
  } else if context.dia().flush() > 0 {
    prompt(context, "run work got %d errors\n", context.dia().totalErrors())
  } else if result != nil {
    for i, v := range result {
      if s := ""; isNull(v) {
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

  printLeavingDirectory(context)
}
