//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "extbit.io/smart/token"
  "path/filepath"
  "runtime/debug"
  "strings"
  "regexp"
  "bufio"
  "bytes"
  "sync"
  "time"
  "fmt"
  "os"
  "io"
  "io/ioutil"
)

type commandLineOpts struct {
  help            bool `h,help`
  debug           bool `d,debug;ps,print-stack`
  debugErrors     bool `de,debug-errors` // optionDebugErrors
  debugWarns      bool `dw,debug-warns`  // optionDebugWarns
  debugInfos      bool `di,debug-infos`  // optionDebugInfos
  debugPrompt     bool `dp,debug-prompt` // optionDebugInfos
  printConfig     bool `po,print-options;po,printoptions`  // optionPrintConfiguration
  printFlags      bool `pf,print-flags`                    // optionPrintFlags
  buildPlugins    bool `bp,build-plugins;bp,buildplugins`  // optionAlwaysBuildPlugins
  benchImport     bool `bi,bench-import;bi,benchimport`    // optionBenchImport
  benchBuiltins   bool `bb,bench-builtins`                 // optionBenchBuiltin
  benchSlow       bool `bs,bench-slow;bs,benchslow`        // optionBenchSlow
  verbose         bool `v,verbose`          // optionVerbose
  verboseImport   bool `vi,verbose-import`  // optionVerboseImport
  verboseChecks   bool `vc,verbose-checks`  // optionVerboseChecks
  verboseLoads    bool `vl,verbose-loading` // optionVerboseLoading
  verboseParse    bool `vp,verbose-parsing` // optionVerboseParsing
  verboseUsing    bool `vu,verbose-using`   // optionVerboseUsing
  cleanDotCache   bool `cc,clean-cache;cc,clear-cache;rc,rm-cache`
  cleanDotDeps    bool `cd,clean-deps;cd,clear-deps;rd,rm-deps`
  cleanDotGrep    bool `cg,clean-grep;cg,clear-grep;rg,rm-grep`
  cleanTmpDirs    bool `ct,clean-temp;ct,clear-temp;rt,rm-temp`
  configure       bool `g,configure`          // optionConfigure
  reconfigure     bool `r,reconfigure`        // optionReconfig
  noExec          bool `ne,no-exec;ne,no-execute`  // optionNoExec
  saveGrepSource  bool `sgs,save-grep-source`

  failOnErrors    bool `foe,fail-on-errors`

  traceLaunch     bool `tl,trace-launch`
  traceParsing    bool `tp,trace-parse`
  traceExecutor   bool `te,trace-executor`
  traceExec       bool `tx,trace-exec`
  traceEntering   bool `ti,trace-entering`
  traceConfig     bool `tc,trace-config`
  traceTraversal  bool `tt,trace-traverse`
  traceTraversalNestIndent bool `tni,trace-nest-indent`
}

type Context interface {
  // Globe returns the universe globe.
  Globe() *Globe

  // WorkDir returns the specific work directory for this context
  WorkDir() string // vs os.Getwd, aka. context.workdir

  // Pos returns the diagnostic position where this context is taking place.
  Position() Position

  // Scope returns the closure scope
  Scope() *Scope

  // String() returns a string form of the context
  String() string

  auto() *autoContext
  autoGet(string) (Value, bool)
  autoSet(string, Value) (Value, bool)
  autoArgs([]*Def, []Value) ([]string, error)

  closure() *closureContext
  closureScopes() []*Scope
  closureResolveAuto(string) (Object, bool)

  colonResolve(string) (Object, bool)

  inner() Context
  spawn() Context

  traversal() *traverseContext

  Project() *Project

  programCtx() *programContext
  program() *Program

  entry() Entry
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
  if target, ok := ctx.autoGet("@"); !ok || isNil(target) {
    if false { erro(ctx, "target '%v' is nil", target) }
  } else if vals, _, err := expandall2(ctx, expandPlainValue, target); err != nil {
    erro(ctx, "expand target '%v' failed: %v", target, err).of(target)
  } else if len(vals) == 1 { res = Scalar(vals[0]) } else {
    erro(ctx, "target '%v' expaned to many: %v", target, res).of(target)
  }
  return
}

func getTargetValueString(ctx Context) (val Value, str string) {
  var err error
  if val = getTargetValue(ctx); isNil(val) {
    if false { erro(ctx, "target '%v' is nil", val) }
  } else if str, err = fullnameOrStrval(ctx, val); err != nil {
    erro(ctx, "strval target '%v' failed: %v", val, err).of(val)
  }
  return
}

var options = commandLineOpts{
  debugPrompt: true,
  debugErrors: true,
  debugWarns : true,
  debugInfos : true,

  traceTraversalNestIndent: true,
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
func (d *diagPoint) at(position Position) *diagPoint {
  d.position = position
  return d
}
func (d *diagPoint) of(value Value) *diagPoint {
  d.position = value.Position()
  return d
}
func (d *diagPoint) debug(args ...interface{}) *diagPoint {
  const skips = 5 // skips the standard stack lines, which is not very useful
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
    for j += j % 2; i+1 < len(v) && 0 < j; i += 2 {
      if false { fmt.Fprintf(stderr, "%s\n%s\n", v[i+0], v[i+1]) }
      var (
        sm1 = goStackLine1.FindAllSubmatch(v[i+0], 1)
        sm2 = goStackLine2.FindAllSubmatch(v[i+1], 1)
        isPanic = len(sm1) > 0 && len(sm1[0]) > 1 && bytes.Equal(sm1[0][1], []byte("panic"))
        se string
      )
      if gotPanic { se = "		<---- panic" }
      if sm1 != nil && sm2 != nil && !isPanic {
        d.stack = append(d.stack, sm2[0][1]...)
        d.stack = append(d.stack, []byte(":"+s+" ")...)
        d.stack = append(d.stack, sm1[0][1]...)
        d.stack = append(d.stack, sm1[0][2]...)
        d.stack = append(d.stack, []byte(se+"\n")...)
      }
      gotPanic = isPanic
      j -= 2
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
func (diag *diagContext) String() string { return fmt.Sprintf("diag{%s}", diag.Context) }
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
  if p = ctx.diag(dt, f, a...); p != nil { p.at(ctx.Position()) }
  return
}
func info(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagInfo, f, a...) }
func warn(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagWarn, f, a...) }
func erro(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagError, f, a...) }
func prompt(ctx Context, f string, a ...interface{}) *diagPoint { return diag(ctx, diagPrompt, f, a...) }

type spawnPositionalContext struct { positionalContext }
func (pc *spawnPositionalContext) String() string { return fmt.Sprintf("spawn-%s", pc.positionalContext.String()) }

type positionalContext struct { Context; position Position }
func (pc *positionalContext) inner() Context { return pc.Context }
func (pc *positionalContext) Position() Position { return pc.position }
func (pc *positionalContext) String() string { return fmt.Sprintf("positional{%s}", pc.Context) }
func (pc *positionalContext) spawn() Context {
  var ctx = pc.Context
  switch t := ctx.(type) {
  case *programContext, *traverseContext, *closureContext: ctx = t.spawn()
  default: erro(pc, "needs to spawn positional context: %v", ctx).debug(1)
  }
  return &spawnPositionalContext{positionalContext{ ctx, pc.position }}
}
func positional(ctx Context, pos Position) Context {
  if pc, ok := ctx.(*positionalContext); ok && pos.Equals(&pc.position) {
     return ctx;
  }
  return &positionalContext{ ctx, pos }
}

type argumentedContext struct {
  Context
  args []Value
}
func (ac *argumentedContext) inner() Context { return ac.Context }
func (ac *argumentedContext) String() string { return fmt.Sprintf(`argumented{%s}`, ac.Context) }
func (ac *argumentedContext) arguments() []Value { return ac.args }
func (ac *argumentedContext) argumented() *argumentedContext { return ac }
func (ac *argumentedContext) argumentedSet(args []Value) (prev []Value) {
  prev, ac.args = ac.args, args
  return
}

func executeEntry(ctx Context, entry *RuleEntry, args ...Value) (result []Value, okay bool) {
  var brks breakers
  if result, brks = entry.Execute(positional(ctx, entry.position), args...); !brks.has() {
    okay = true; return
  }
  if tb := brks.of(breakNext, breakCase, breakDone); tb.has() {
    brks, okay = brks.not(breakNext, breakCase, breakDone), true
  }
  if tb := brks.of(breakFail, breakErro); tb.has() {
    brks, okay = brks.not(breakFail, breakErro), false
    for _, brk := range tb {
      switch brk.what {
      case breakErro: erro(ctx, "broken execution '%v' error: %v", entry, brk.error).at(brk.pos).debug(1)
      case breakFail: erro(ctx, "broken execution '%v' failed: %v", entry, brk.message).at(brk.pos).debug(1)
      }
    }
  }
  if brks.has() {
    for _, brk := range brks {
      erro(ctx, "broken execution for '%v' (%v)", entry, brk.what).at(brk.pos).debug(1)
    }
    okay = false
  }
  return
}

type defaultContext struct {
  diagContext
  workdir  string
  prefix   string // FIXME: prefix for distribution
  globe   *Globe
  loader  *loader
}
func (ctx *defaultContext) arguments() []Value { return nil }
func (ctx *defaultContext) argumented() *argumentedContext { return nil }
func (ctx *defaultContext) argumentedSet([]Value) []Value { return nil }
func (ctx *defaultContext) inner() Context { return nil }
func (ctx *defaultContext) spawn() Context { return nil }
func (ctx *defaultContext) auto() *autoContext { return nil }
func (ctx *defaultContext) closure() *closureContext { return nil }
func (ctx *defaultContext) traversal() *traverseContext { return nil }
func (ctx *defaultContext) entry() Entry { return nil }
func (ctx *defaultContext) stems() []string { return nil }
func (ctx *defaultContext) Scope() *Scope { return ctx.globe/*.main*/.scope }
func (ctx *defaultContext) Project() *Project { return ctx.globe.main }
func (ctx *defaultContext) program() *Program { return nil }
func (ctx *defaultContext) programCtx() *programContext { return nil }
func (ctx *defaultContext) Position() (res Position) { res.Filename, res.Line = ctx.workdir, 1; return }
func (ctx *defaultContext) appendCallerUpdated() bool { return false }
func (ctx *defaultContext) mustExists() bool { return false }
func (ctx *defaultContext) WorkDir() string { return ctx.workdir }
func (ctx *defaultContext) Globe() *Globe { return ctx.globe }
func (ctx *defaultContext) String() string { return "default" }
func (ctx *defaultContext) configuration() bool { return false }
func (ctx *defaultContext) colonResolve(name string) (obj Object, found bool) {
  switch g := ctx.globe; name {
  case "os"   : obj, found = g.os.self, true
  case "goals": obj, found = g.goals,   true
  case "mode" : obj, found = g.mode,    true
  }
  return
}
func (ctx *defaultContext) closureResolveAuto(name string) (obj Object, found bool) { return ctx.colonResolve(name) }
func (ctx *defaultContext) autoArgs(_ []*Def, _ []Value) ([]string, error) { return nil, nil }
func (ctx *defaultContext) autoSet(name string, val Value) (res Value, ok bool) {
  erro(ctx, `setting auto "%v" in base context, value=%v`, val).debug(64)
  return
}
func (ctx *defaultContext) autoGet(name string) (res Value, found bool) {
  var obj Object
  if obj, found = ctx.closureResolveAuto(name); found {
    if def, ok := obj.(*Def); ok {
      res = def.value
    } else {
      res = obj // FIXME: should not return obj directly
    }
  }
  return
}
func (ctx *defaultContext) closureScopes() (scopes []*Scope) {
  if m := ctx.globe.main; m != nil && m.scope != nil {
    if false { scopes = append(scopes, m.scope) }
  }
  return
}

func (ctx *defaultContext) help()       { do_helpscreen(ctx) }
func (ctx *defaultContext) helpFlags()  { print_flag_trace(ctx) }
func (ctx *defaultContext) helpConfig() { print_configuration(ctx) }

func (dc *defaultContext) run() (result []Value, breakers []*breaker) {
  if options.traceLaunch { defer un(trace(t_launch, "defaultContext.run")) }

  var main = dc.globe.main
  if main == nil {
    erro(dc, "no targets to update `%v`", dc.globe.goals)
    return
  }

  var ctx Context = &closureContext{dc, []*Scope{main.scope}}
  removeTempDirs(ctx)

  var done bool
  for _, flag := range dc.globe.flags {
    var ( s string; err error )
    if s, err = flag.name.Strval(ctx); err != nil { return }
    var args, _ = dc.globe.args[flag]
    var entries, _ = dc.globe.flagEntries[s]
    for _, entry := range entries {
      var ( res []Value; brks []*breaker )
      if res, brks = entry.Execute(positional(ctx, entry.Position()), args...); len(brks) > 0 {
        for _, brk := range brks {
          if brk.what == breakErro {
            erro(ctx, "execute '%v' failed: %v", entry, brk.error).at(entry.Position()).debug(1)
          }
        }
      }
      result = append(result, res...)
      done = true
    }
  }
  if done { return }

  var updated int
  var goals []Value
  var collect func(proj *Project, vals []Value) bool
  collect = func(proj *Project, vals []Value) bool {
    if len(vals) == 0 {
      if entry := proj.DefaultEntry(); entry != nil {
        goals = append(goals, entry)
      } else {
        // NOTE: ignored project
      }
      return true
    }
    for _, goal := range vals {
      switch t := goal.(type) {
      case *None: // just ignore
      case *Bareword:
        if entry, err := proj.resolveEntry(ctx, t.string, false); err != nil {
          erro(ctx, "resolve '%s': %s", t.string, err).debug(1)
          return false
        } else if entry == nil {
          erro(ctx, "no such entry `%s`", t.string).debug(1)
          return false
        } else {
          goals = append(goals, entry)
        }
      case *delegate:
        if s, err := t.Strval(ctx); err != nil {
          erro(ctx, "strval '%v' failed: %s", t, err).debug(1)
          return false
        } else if entry, err := proj.resolveEntry(ctx, s, false); err != nil {
          erro(ctx, "resolve entry '%s' failed: %s", s, err).debug(1)
          return false
        } else if entry == nil {
          erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
          return false
        } else {
          goals = append(goals, entry)
        }
      case *Flag:
        if s, err := t.Strval(ctx); err != nil {
          erro(ctx, "strval '%v' failed: %s", t, err).debug(1)
          return false
        } else if entry, err := proj.resolveEntry(ctx, s, false); err != nil {
          erro(ctx, "resolve entry '%s' failed: %s", s, err).debug(1)
          return false
        } else if entry == nil {
          erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
          return false
        } else {
          goals = append(goals, entry)
        }
      case *Argumented:
        if s, err := t.value.Strval(ctx); err != nil {
          erro(ctx, "strval '%v' failed: %s", t.value, err).debug(1)
          return false
        } else {
          // For examples:
          //     project-name(-clean)
          //     project/spec(-clean)
          //     xxxx()
          var ( args = merge(t.args...); found int )
          for _, p := range dc.globe.projects {
            if p.name == s || p.spec == s { found += 1
              if !collect(p, args) { return false }
            }
          }
          if found == 0 {
            erro(ctx, `"%s" not loaded: %v`, s, args).debug(1)
            return false
          }
        }
      default:
        erro(ctx, "%v: unknown target: %v (%s)", proj, goal, typeof(goal)).debug(1)
        return false
      }
    }
    return true
  }

  if collect(main, merge(dc.globe.goals.value)) {
    if len(goals) == 0 {
      if entry := main.DefaultEntry(); entry != nil {
        goals = append(goals, entry)
      }
    }
    for _, goal := range goals {
      var args, _ = dc.globe.args[goal]
      result = append(result, dc.update(goal, args)...)
      updated += 1
    }
  }
  return
}

func (ctx *defaultContext) update(goal Value, args []Value) (result []Value) {
  if isNil(goal) {
    // TODO: report nil goal
  } else {
    var okay bool
    switch g := goal.(type) {
    case *RuleEntry:
      if result, okay = executeEntry(positional(ctx, g.position), g, args...); !okay {
        erro(ctx, "update '%v' failed", g).at(ctx.Position()).debug(1)
      }
    default:
      erro(ctx, "'%v' is not an entry (%T)", goal, goal).of(goal).debug(1)
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

// load loads smart files, making it as individual func to avoid being abused by loaders.
func (ctx *defaultContext) load() (err error) {
  if options.traceLaunch { defer un(trace(t_launch, "defaultContext.load")) }
  defer func(prevLoader *loader) {
    ctx.globe.projects = ctx.loader.loaded
    ctx.loader = prevLoader
  } (ctx.loader)

  var (
    base, _ = os.Getwd()
    sp = filepath.Join(base, ".smart", "modules")
    pos Position // FIXME: find a useful position
    args []Value
  )
  pos.Filename = base
  ctx.loader = &loader{
    closureContext: closureContext{ctx, []*Scope{ctx.globe.scope}},
    fset:     token.NewFileSet(), 
    paths:    []string(globalPaths),
    loaded:   make(map[string]*Project),
  }
  ctx.globe.goals = &Def{
    knownobject: knownobject{objbase{scope:ctx.globe.scope}, "goals"},
    origin: DefDefault, value: MakeNone(pos),
  }
  ctx.globe.mode = &Def{
    knownobject: knownobject{objbase{scope:ctx.globe.scope}, "mode"},
    origin: DefDefault, value: MakeNone(pos),
  }

  if _, e := os.Stat(sp); e == nil { ctx.loader.AddSearchPaths(sp) }

  if text := strings.Join(os.Args[1:], " "); text == "" {
    // Relax!
  } else if args = ctx.loader.loadText("@", text); len(args) == 0 {
    // ohh...
  } else if args, err = parseOpts(ctx, &options, args...); err != nil {
    erro(ctx, "parse opts failed: %v", err).at(pos).debug(1)
    return
  }

  if v := options.reconfigure; v { options.configure = v }

  if options.verbose || options.benchImport {
    defer func(t time.Time) {
      var d = time.Now().Sub(t)
      prompt(ctx, "Goals %v (%s)\n", ctx.globe.goals, d)
    } (time.Now())
  }

  assert(ctx.globe.args != nil, "globe args is nil")

  var mode = new(Bareword)
  for _, target := range args {
    switch t := target.(type) {
    case *Pair: ctx.globe.pairs = append(ctx.globe.pairs, t)
    case *Flag: ctx.globe.flags = append(ctx.globe.flags, t)
      if s, err := t.name.Strval(ctx); err == nil && s == "clean" {
        mode.position, mode.string = t.position, "clean"
      }
    case *Argumented:
      ctx.globe.args[t.value] = t.args
      if f, ok := t.value.(*Flag); ok {
        ctx.globe.flags = append(ctx.globe.flags, f)
      } else {
        ctx.globe.goals.append(ctx, t/*.value*/)
      }
    default:
      ctx.globe.goals.append(ctx, t)
    }
  }
  if mode.string == "" {
    if options.configure {
      mode.string = "configure"
    } else {
      mode.string = "goals"
    }
  }
  ctx.globe.mode.value = mode

  defer func(t time.Time) {
    var d = time.Now().Sub(t)
    if options.verboseImport {
      var name string
      if p := ctx.loader.Project(); p != nil { name = p.name }
      fmt.Fprintf(stderr, "└·%s … (%s)\n", name, d)
    } else if d > 4999*time.Millisecond {
      prompt(ctx, "%s: long load time: %s !!\n", base, d).debug(1)
    }
  } (time.Now())
  if options.verboseImport { fmt.Fprintf(stderr, "┌→%s\n", base) }

  if !ctx.loader.loadPath(base, nil) { return }
  if ctx.globe.main == nil { fmt.Fprintf(stderr, "nothing loaded\n") }
  return
}

func checkPanicsErrors(ctx Context, dontCheckErrors ...bool) (panics, errs int) {
  for e := recover(); e != nil; e = recover() {
    switch t := e.(type) {
    case bailout: continue
    case failure: erro(ctx, "panic: %v", t.metainfo).at(t.position)
    default     : erro(ctx, "panic: %v", e)
    }
    panics += 1
  }
  if panics > 0 {
    erro(ctx, "failed: got %d panics (%s)", panics, ctx).debug(128)
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
  var context = &_context
  defer checkPanicsErrors(context)

  if options.traceLaunch { defer un(trace(t_launch, "CommandLine")) }
  if optionEnableBenchmarks {
    var w *bufio.Writer
    var d = filepath.Join(context.workdir, "benchmarks")
    if err := os.MkdirAll(d, os.FileMode(0777)); err != nil {
      erro(context, "%v", err).at(context.Position()).debug(1)
      return
    } else if f, err := ioutil.TempFile(d, "*.log"); err != nil {
      erro(context, "%v", err).at(context.Position()).debug(1)
      return
    } else {
      w = bufio.NewWriter(f)
      benchmark.start = time.Now()
      benchmark.spot = benchmark.start
      defer func(t time.Time) {
        benchspot_report(w)
        w.WriteString("--------\n")
        benchmark.spent = time.Now().Sub(t)
        benchmark.summary(w)
        benchmark.report(w, 0, nil)
        w.Flush()
        f.Close()
      } (benchmark.spot)
    }
  }

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
  globalPaths = append(modulesPaths, globalPaths...)
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
        globalPaths = append(globalPaths, line)
      }
    }
    if err != nil { fmt.Fprintf(stderr, "%v: %v", file, err); return }
  }

  if context.countErrors() > 0 { return }

  //loadGrepCache()

  defer func(globe *Globe) {
    saveGrepCache(context)
    context.globe = globe
  } (context.globe)
  context.globe = NewGlobe(context, "smart")

  if err := context.load(); err != nil {
    erro(context, "loading work failed: %v", err).at(context.Position())
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
    erro(context, "run work failed: %v", err).at(context.Position())
  } else if context.checkErrors(true) > 0 {
    prompt(context, "run work got %d errors\n", context.totalErrors())
  } else if result != nil {
    var ( s string; err error )
    for i, v := range result {
      if isNil(v) { continue } else if i > 0 && s != "" {
        fmt.Fprintf(stderr, ", ")
      }
      if s, err = v.Strval(context); err != nil {
        fmt.Fprintf(stderr, "%s [%s]", v, err)
      } else {
        fmt.Fprintf(stderr, "%s", s)
      }
    }
    fmt.Fprintf(stderr, "\n")
  }

  printLeavingDirectory(context)
}
