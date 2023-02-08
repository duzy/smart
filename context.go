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
  "regexp"
  "bufio"
  "bytes"
  "sync"
  "fmt"
  "os"
  "io"
)

const productVerTag = "dev" // dev, alpha, beta, release

var dd bool

type commandLineOpts struct {
  help            bool `h,help`

  debug           bool `d,db,debug`
  debugErrors     bool `de,dberro,debug-errors` // optionDebugErrors
  debugWarns      bool `dw,dbwarn,debug-warns`  // optionDebugWarns
  debugInfos      bool `di,dbinfo,debug-infos`  // optionDebugInfos
  debugPrompt     bool `dp,dbprom,debug-prompt` // optionDebugInfos

  autoProfs       bool `ap,autoprof,auto-profiles,auto-profile`
  cpuProf         string `cpuprof,cpu-profile`
  memProf         string `memprof,memory-profile`

  printConfig     bool `opts,print-options,printoptions`    // optionPrintConfiguration
  printFlags      bool `flags,print-flags,printflags`       // optionPrintFlags

  buildPlugins    bool `bp,bup,build-plugins,buildplugins`  // optionAlwaysBuildPlugins

  verbose         bool `v,verb,verbose`
  verboseBreaks   bool `vb,vbrk,verbose-breaks`
  verboseChecks   bool `vc,vchk,verbose-checks`
  verboseImport   bool `vi,vimp,verbose-import`
  verboseLoads    bool `vl,vloa,verbose-loading`
  verboseParse    bool `vp,vpar,verbose-parsing`
  verboseUsing    bool `vu,vuse,verbose-using`

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

  slow int64 `sl,slow`

  parallel        bool `p,par,para,parallel`

  fastMode        bool `f,fm,fast,fast-mode`
  failOnErrors    bool `fe,foe,fail-on-errors`

  traceLaunch     bool `tl,trace-launch`
  traceParsing    bool `tp,trace-parse`
  traceExecutor   bool `te,trace-executor`
  traceExec       bool `tx,trace-exec`
  traceEntering   bool `ti,trace-entering`
  traceConfig     bool `tc,trace-config`
}

const fullContextStringer bool = false

type Context interface {
  // Globe returns the universe globe.
  Globe() *Globe

  // WorkDir returns the specific work directory for this context
  WorkDir() string // vs os.Getwd, aka. context.workdir

  // Pos returns the diagnostic position where this context is taking place.
  Position() Position

  // Scope returns the closure scope
  Scope() *Scope

  // String() returns a string representation of the context
  String() string

  aquireLock() (unlock func())
  wait()

  universe() *universeContext

  loader() *loader // only in load stage
  parser() *parser // only in parse stage

  auto() *autoContext
  autoGet(string) *def
  autoSet(string, Value) (*def, Value)
  autoArgs([]*def, []Value) ([]string, error)

  closure() *closureContext
  closureScopes() []*Scope
  closureResolveAuto(string) (Object, bool)

  colonResolve(string) (Object, bool)

  inner() Context
  // at(Context) Context
  spawn(Context) Context

  positionContext() *positionContext

  travestates() *travestates
  traversed(target Value) []Value

  Project() *Project
  projects(Context, ...*Project) []*Project
  // resolveObject(s string) (obj Object)
  // resolveEntries(s string, matchingFullSuffix, alwaysResolveBases bool) (entries *ResolveEntries)
  // resolvePatterns(v Value, s string) (res []*stemmed)

  programContext() *programContext
  program() *Program

  dirtyOpts() *modifierSetDirtyPatsOpts
  dirtyMark(...Value)

  entry() Entry
  entryContext() *entryContext

  stemmedContext() *stemmedContext
  stemmed() *stemmed
  stems() []string

  argumented() *argumentedContext
  argumentedSet([]Value) []Value
  arguments() []Value

  configuration() bool

  diagnostic() *diagContext
  diag(diagType, string, ...interface{}) *diagPoint
  checkErrors(bool) int
  countErrors() int
  totalErrors() int

  appendCallerUpdated() bool
  mustExists() bool
}

func getTargetValue(ctx Context) (res Value) {
  if val := autoGet(ctx, "@"); val == nil {
    if false { erro(ctx, "target is nil") }
  } else if vals, u, n := plain.expand(ctx, val); len(vals) == 1 {
    res = Scalar(vals[0])
  } else {
    erro(of(ctx,val), "target '%v' expaned to many: %v (%d,%d)", val, res, u, n)
  }
  return
}

func getTargetValueString(ctx Context) (val Value, str string) {
  if val = getTargetValue(ctx); isNil(val) {
    if false { erro(ctx, "target '%v' is nil", val) }
  } else {
    str = fullnameOrStrval(ctx, val)
  }
  return
}

var baseWorkDir, _ = os.Getwd()
var options = commandLineOpts{
  debugPrompt: true,
  debugErrors: true,
  debugWarns:  true,
  debugInfos:  true,

  failOnErrors: true,
  fastMode: true,

  parallel: false, // FIXME: Program.traverse not working in parallel

  slow: 1999 * 80,
}

type diagType int
const (
  diagInfo diagType = iota
  diagWarn
  diagError
  diagPrompt
)

var (
  goStackLine1 = regexp.MustCompile(`^(?:extbit\.io/)?(.+)(\(.*\))$`)
  goStackLine2 = regexp.MustCompile(`^	(.*?:\d+)(?: \+.*)?$`)
)
type diagPoint struct {
  dt diagType
  position Position
  message string
  stack []byte // see also debug.Stack()
}
func (d *diagPoint) debug(args ...interface{}) *diagPoint {
  const skips = 5 // skips the standard stack lines, which is not very useful
  switch productVerTag {
  case "dev", "debug": // only print debug diags for dev and debug versions
  default: return d
  }
  switch d.dt {
  case diagPrompt: if !options.debugPrompt { return d }
  case diagInfo:   if !options.debugInfos  { return d }
  case diagWarn:   if !options.debugWarns  { return d }
  case diagError:  if !options.debugErrors { return d }
  }
  if n := len(args); n  > 1 {
    if enabled, ok := args[0].(bool); ok {
      if enabled { args = args[1:] }  else { return d }
    }
  }

  var (
    ln = []byte{ '\n' }
    v = bytes.Split(debug.Stack(), ln)
    i, j int
  )
  if skips > 0 && len(v) > skips { i = skips }
  if n := len(args); n == 1 {
    if t, ok := args[0].(int); ok { j = t }
  } else if n == 2 {
    if t, ok := args[0].(int); ok { i += t }
    if t, ok := args[1].(int); ok { j = t }
  } else if n > 2 {
    panic("too many debug args")
  } else {
    panic("needs debug args")
  }

  var s string
  switch d.dt {
  case diagPrompt: s = "note:"
  case diagInfo:   s = "info:"
  case diagWarn:   s = "warning:"
  }

 if false {
    var (
      sm1 = goStackLine1.FindAllSubmatch(v[i+0], 1)
      sm2 = goStackLine2.FindAllSubmatch(v[i+1], 1)
    )
    if j == 1 && sm1 != nil && sm2 != nil {
      d.stack = append(sm2[0][1], []byte(":"+s+" ")...)
      d.stack = append(d.stack, sm1[0][1]...)
      d.stack = append(d.stack, []byte("\n")...)
    } else if 0 < j && i+j <= len(v) {
      if j % 2 != 0 { j += 1 }
      ending := []byte(" (and more frames…)\n") //[]byte("\n…more frames not displayed ……\n")
      d.stack = append(bytes.Join(v[i:i+j], ln), ending...)
    }
  } else if true {
    var gotPanic bool
    for j += j % 2; 0 < j && i+1 < len(v); i, j = i+2, j-2 {
      var (
        sm1 = goStackLine1.FindAllSubmatch(v[i+0], 1)
        sm2 = goStackLine2.FindAllSubmatch(v[i+1], 1)
        isPanic = len(sm1) > 0 && len(sm1[0]) > 1 && bytes.Equal(sm1[0][1], []byte("panic"))
        se string
      )
      if gotPanic { se = "		<---- panic" }
      if sm1 != nil && sm2 != nil && !isPanic {
        var e string
        if 0 < j-2 && i+3 < len(v) { e = se+"\n" } else { e = " ...\n" }
        d.stack = append(d.stack, sm2[0][1]...)
        d.stack = append(d.stack, []byte(":"+s+" ")...)
        d.stack = append(d.stack, sm1[0][1]...)
        d.stack = append(d.stack, sm1[0][2]...)
        d.stack = append(d.stack, []byte(e)...)
      }
      gotPanic = isPanic
    }
  } else {
    d.stack = bytes.Join(v[i:], ln)
  }
  return d
}

type diagContext struct {
  Context
  sync.Mutex
  points []*diagPoint
  nested [][]*diagPoint
  errs int
}
func (diag *diagContext) inner() Context { return diag.Context }
func (diag *diagContext) spawn(ctx Context) Context {
  return &diagContext{ Context: diag.Context.spawn(ctx) }
}
func (diag *diagContext) aquireLock() (unlock func()) {
    diag.Lock() ; return func() { diag.Unlock() }
}
func (diag *diagContext) String() string {
  if fullContextStringer {
    return fmt.Sprintf("diag{%s}", diag.Context)
  } else {
    return diag.Context.String()
  }
}
func (diag *diagContext) diagnostic() *diagContext { return diag }
func (diag *diagContext) reset() {
  diag.Lock(); defer diag.Unlock()
  diag.points = []*diagPoint{}
}

func (diag *diagContext) add(point *diagPoint) *diagPoint {
  diag.Lock(); defer diag.Unlock()
  diag.points = append(diag.points, point)
  return point
}
func (diag *diagContext) nest(points []*diagPoint) {
  diag.Lock(); defer diag.Unlock()
  diag.nested = append(diag.nested, points)
}

func (diag *diagContext) diag(dt diagType, f string, args ...interface{}) *diagPoint {
  if dt != diagPrompt { f = strings.TrimSpace(f) }
  return diag.add(&diagPoint{ dt, diag.Position(), fmt.Sprintf(f, args...), nil })
}

func (diag *diagContext) countErrors() (num int) {
  diag.Lock(); defer diag.Unlock()
  for _, d := range diag.points {
    if d.dt == diagError { num += 1 }
  }
  return
}
func (diag *diagContext) totalErrors() (num int) { return diag.errs }
func (diag *diagContext) checkErrors(reset bool) (num int) {
  diag.Lock(); defer func() { diag.errs += num; diag.Unlock() } ()
  for i, points := range append([][]*diagPoint{diag.points}, diag.nested...) {
    var nested = i > 0 && len(points) > 0 && len(diag.nested) > 0
    if nested { fmt.Fprintf(stderr, "\n#%d:\n", i) }
    var lastPromptLn = -1
    for j, d := range points {
      var (
        msg = d.message
        pos = d.position.String()
      )
      if d.dt == diagPrompt {
        if msg == "" {
          // nothing needed to be done
        } else if fmt.Fprintf(stderr, "%s", msg); strings.HasSuffix(msg, "\n") {
          lastPromptLn = 1
        } else {
          lastPromptLn = 0
        }
      } else {
        if false && lastPromptLn == 0 && j > 0 { fmt.Fprintf(stderr, "\n") }
        switch lastPromptLn = -1; d.dt {
        case diagError: fmt.Fprintf(stderr, "%v: %s\n",         pos, msg); num += 1
        case diagInfo : fmt.Fprintf(stderr, "%v:info: %s\n",    pos, msg)
        case diagWarn : fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
        }
      }
      if len(d.stack) > 0 {
        if lastPromptLn == 0 { fmt.Fprintf(stderr, "\n") }
        fmt.Fprintf(stderr, "%s\n", bytes.TrimSpace(d.stack))
      }
      if num > 49 { fmt.Fprintf(stderr, "%v: too many errors (%d)\n", pos, num); break }
    }
    if nested { fmt.Fprintf(stderr, "#%d;\n\n", i) }
  }
  if reset {
    diag.points =   []*diagPoint{}
    diag.nested = [][]*diagPoint{}
  }
  return
}

func diagnostic(ctx Context) Context { return &diagContext{ Context: ctx } }
func diag(ctx Context, dt diagType, f string, a ...interface{}) (p *diagPoint) {
  if p = ctx.diag(dt, f, a...); p != nil { p.position = ctx.Position() }
  return
}
func info(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagInfo, f, a...) }
func warn(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagWarn, f, a...) }
func erro(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagError, f, a...) }
func prompt(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagPrompt, f, a...) }

type positionContext struct { Context; position Position }
func (pc *positionContext) positionContext() *positionContext { return pc }
func (pc *positionContext) caller() *positionContext { return pc.Context.positionContext() }
func (pc *positionContext) inner() Context { return pc.Context }
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
  if ctx == nil { panic("nil inner context") } else
  if p := ctx.Position(); p.IsValid() && pos.IsValid() && !p.Same(&pos) {
    var ( wrap bool = true ; num int )
    for c, i, y := ctx, 0, true; c != &universe; c, i = c.inner(), i+1 {
      if _, y = c.(*positionContext); y && i > 9999 {
        if wrap { wrap, num = false, i }
        if true {
          prompt(ctx, "%v: too many positions: %T\n", p, c)
          warn(ctx, "too many positions: %v, %v, %v", i, num, ctx).debug(1)
          ctx.checkErrors(true)
        }
      }
    }
    if wrap { ctx = &positionContext{ ctx, pos } } else {
      warn(ctx, "too many positions: %v, %v", num, ctx).debug(128)
      ctx.checkErrors(true)
    }
  } else if _, y := ctx.(*parser); false && y && p.Same(&pos) {
    ctx = &positionContext{ ctx, pos }
  }
  return ctx
}

type argumentedContext struct {
  Context
  args []Value
}
func (ac *argumentedContext) inner() Context { return ac.Context }
func (ac *argumentedContext) String() string {
  if fullContextStringer {
    return fmt.Sprintf(`argumented{%s}`, ac.Context)
  } else {
    return ac.Context.String()
  }
}
func (ac *argumentedContext) arguments() []Value { return ac.args }
func (ac *argumentedContext) argumented() *argumentedContext { return ac }
func (ac *argumentedContext) argumentedSet(args []Value) (prev []Value) {
  prev, ac.args = ac.args, args
  return
}

func executeEntry(ctx Context, entry *RuleEntry, args ...Value) (result []Value, okay bool) {
  var traves travestates
  if result, traves = entry.Execute(at(ctx, entry.position), args...); !traves.has() {
    okay = true; return
  }

  if t := traves.of(traveFail); t.has() {
    traves, okay = traves.not(traveFail), false
    for _, brk := range t { erro(at(ctx,brk.pos), "%v: %v", entry, brk).debug(1) }
    return
  }

  if t := traves.of(traveCase, traveDone, traveNext, traveRule, traveFile); t.has() {
    traves, okay = traves.not(traveCase, traveDone, traveNext, traveRule, traveFile), true
  }

  if traves.has() {
    for _, brk := range traves { erro(at(ctx,brk.pos), "%v: %v", entry, brk).debug(1) }
    okay = false
  }
  return
}

func updateGoal(ctx Context, goal Value, args []Value) (result []Value) {
  if isNil(goal) {
    // TODO: report nil goal
  } else {
    var okay bool
    switch g := goal.(type) {
    case *RuleEntry:
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
  if s == "" {
    s = cwd
  }
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
  if strings.HasSuffix(dir, "do.smart") || strings.HasSuffix(dir, "build.smart") {
    pos.Filename = dir
  } else if _, e := os.Stat(filepath.Join(dir, "do.smart")); e == nil {
    pos.Filename = filepath.Join(dir, "do.smart")
  } else if _, e := os.Stat(filepath.Join(dir, "build.smart")); e == nil {
    pos.Filename = filepath.Join(dir, "build.smart")
  } else {
    pos.Filename = dir
  }
  pos.Line = 1
  return
}

func checkFailure(ctx Context, dontCheckErrors ...bool) (panics, errs int) {
  for e := recover(); e != nil; e = recover() {
    switch t := e.(type) {
    case bailout: continue
    case failure: erro(at(ctx,t.position), "panic: %v", t.metainfo)
    default     : erro(ctx, "panic: %v", e)
    }
    panics += 1
  }
  if panics > 0 {
    var pos = ctx.Position()
    if !strings.HasSuffix(pos.Filename, "do.smart") {
      var s = filepath.Join(pos.Filename, "do.smart")
      if _, e := os.Stat(s); e == nil { pos.Filename = s }
    } else if !strings.HasSuffix(pos.Filename, "build.smart") {
      var s = filepath.Join(pos.Filename, "build.smart")
      if _, e := os.Stat(s); e == nil { pos.Filename = s }
    }
    erro(at(ctx,pos), "failed: got %d panics", panics).debug(128)
  }
  if len(dontCheckErrors) > 0 && dontCheckErrors[0] {
    // okay
  } else if errs = ctx.checkErrors(true); errs > 0 && panics == 0 {
    warn(ctx, "got %d errors (%s)", ctx.totalErrors(), ctx).debug(16)
    if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
  }
  return
}

func CommandLine() {
  var context = &universe
  defer checkFailure(context)

  if options.traceLaunch { defer un(trace(t_launch, "CommandLine")) }

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
  universe.paths = append(modulesPaths, universe.paths...)
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
        universe.paths = append(universe.paths, line)
      }
    }
    if err != nil { fmt.Fprintf(stderr, "%v: %v", file, err); return }
  }

  if context.countErrors() > 0 { return }

  //loadGrepCache()

  if err := context.loadTopWork(); err != nil {
    erro(context, "loading work failed: %v", err)
  } else if context.checkErrors(true) > 0 {
    prompt(context, "loading work got %d errors\n", context.totalErrors())
  } else if options.help {
    context.help()
  } else if options.printFlags {
    context.helpFlags()
  } else if options.printConfig {
    context.helpConfig()
  } else if numUpdatedPlugins > 0 { // see buildPlugin
    prompt(context, "plugins updated, please relaunch.\n")
  } else if options.configure {
    context.configure()
  } else if result, err := context.run(); err != nil {
    erro(context, "run work failed: %v", err)
  } else if context.checkErrors(true) > 0 {
    prompt(context, "run work got %d errors\n", context.totalErrors())
  } else if result != nil {
    for i, v := range result {
      if s := ""; isNil(v) {
        s = "<nil>"
      } else if s = strings.TrimSpace(v.Strval(context)); s == "" {
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
