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
  help        bool `h,help`
  debug       bool `d,debug;ps,print-stack`
  debugErrors bool `de,debug-errors` // optionDebugErrors
  debugWarns  bool `dw,debug-warns`  // optionDebugWarns
  debugInfos  bool `di,debug-infos`  // optionDebugInfos
  debugPrompt bool `dp,debug-prompt` // optionDebugInfos
  printConfig   bool `po,print-options;po,printoptions`  // optionPrintConfiguration
  printFlags    bool `pf,print-flags`                    // optionPrintFlags
  buildPlugins  bool `bp,build-plugins;bp,buildplugins`  // optionAlwaysBuildPlugins
  benchImport   bool `bi,bench-import;bi,benchimport`    // optionBenchImport
  benchBuiltins bool `bb,bench-builtins`                 // optionBenchBuiltin
  benchSlow     bool `bs,bench-slow;bs,benchslow`        // optionBenchSlow
  verbose       bool `v,verbose`          // optionVerbose
  verboseImport bool `vi,verbose-import`  // optionVerboseImport
  verboseChecks bool `vc,verbose-checks`  // optionVerboseChecks
  verboseLoads  bool `vl,verbose-loading` // optionVerboseLoading
  verboseParse  bool `vp,verbose-parsing` // optionVerboseParsing
  verboseUsing  bool `vu,verbose-using`   // optionVerboseUsing
  configure   bool `g,configure`          // optionConfigure
  reconfigure bool `r,reconfigure`        // optionReconfig
  noExec bool `ne,no-exec;ne,no-execute`  // optionNoExec
  saveGrepSource bool `sgs,save-grep-source`
}
var (
  options = commandLineOpts{
    debugPrompt: false,
    debugErrors: true,
    debugWarns : true,
    debugInfos : true,
  }

  // Tracking options
  optionTraceLaunch = false
  optionTraceParsing = false
  optionTraceTraversal = false
  optionTraceTraversalNestIndent = true
  optionTraceExecutor = false
  optionTraceExec = false
  optionTraceEntering = optionTraceTraversal && false
  optionTraceConfig = false
)

type diagType int
const (
  diagInfo diagType = iota
  diagWarn
  diagError
  diagPrompt
)

var (
  //goStackLine1 = regexp.MustCompile(`^(?:extbit\.io/smart\.)?(.+)(\(.*\))$`)
  goStackLine1 = regexp.MustCompile(`^(?:extbit\.io/)?(.+)(\(.*\))$`)
  goStackLine2 = regexp.MustCompile(`^	(.*?:\d+) \+.*$`)
)
type diagnostic struct {
  dt diagType
  position Position
  value Value
  message string
  stack []byte // see debug.Stack()
}
func (d *diagnostic) getPosition() Position {
  if d.value == nil { return d.position }
  return d.value.Position()
}
func (d *diagnostic) debug(args ...interface{}) {
  const skips = 5 // skips the standard stack lines, which is not very useful
  switch d.dt {
  case diagPrompt: if !options.debugPrompt { return }
  case diagInfo:   if !options.debugInfos  { return }
  case diagWarn:   if !options.debugWarns  { return }
  case diagError:  if !options.debugErrors { return }
  }
  if n := len(args); n  > 1 {
    if enabled, ok := args[0].(bool); ok {
      if enabled { args = args[1:] }  else { return }
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
    for j += j % 2; i+1 < len(v) && 0 < j; i += 2 {
      var (
        sm1 = goStackLine1.FindAllSubmatch(v[i+0], 1)
        sm2 = goStackLine2.FindAllSubmatch(v[i+1], 1)
      )
      if sm1 != nil && sm2 != nil {
        d.stack = append(d.stack, sm2[0][1]...)
        d.stack = append(d.stack, []byte(":"+s+" ")...)
        d.stack = append(d.stack, sm1[0][1]...)
        d.stack = append(d.stack, sm1[0][2]...)
        d.stack = append(d.stack, []byte("\n")...)
      }
      j -= 2
    }
  } else {
    d.stack = bytes.Join(v[i:], ln)
  }
}

type Diagnostic struct {
  points []*diagnostic
  m sync.Mutex
}
func (diag *Diagnostic) reset() {
  diag.m.Lock(); defer diag.m.Unlock()
  diag.points = []*diagnostic{}
}
func (diag *Diagnostic) add(point *diagnostic) *diagnostic {
  diag.m.Lock(); defer diag.m.Unlock()
  diag.points = append(diag.points, point)
  return point
}
func (diag *Diagnostic) infoOf(value Value, f string, args... interface{}) *diagnostic {
  var pos Position
  if value != nil { pos = value.Position() }
  return diag.add(&diagnostic{ diagInfo, pos, value, fmt.Sprintf(strings.TrimSpace(f), args...), nil })
}
func (diag *Diagnostic) warnOf(value Value, f string, args... interface{}) *diagnostic {
  var pos Position
  if value != nil { pos = value.Position() }
  return diag.add(&diagnostic{ diagWarn, pos, value, fmt.Sprintf(strings.TrimSpace(f), args...), nil })
}
func (diag *Diagnostic) errorOf(value Value, f string, args... interface{}) *diagnostic {
  var pos Position
  var s = fmt.Sprintf(strings.TrimSpace(f), args...)
  if value != nil { pos = value.Position() }
  return diag.add(&diagnostic{ diagError, pos, value, s, nil })
}
func (diag *Diagnostic) infoAt(pos Position, f string, args... interface{}) *diagnostic {
  return diag.add(&diagnostic{ diagInfo, pos, nil, fmt.Sprintf(strings.TrimSpace(f), args...), nil })
}
func (diag *Diagnostic) warnAt(pos Position, f string, args... interface{}) *diagnostic {
  return diag.add(&diagnostic{ diagWarn, pos, nil, fmt.Sprintf(strings.TrimSpace(f), args...), nil })
}
func (diag *Diagnostic) errorAt(pos Position, f string, args... interface{}) *diagnostic {
  var s = fmt.Sprintf(strings.TrimSpace(f), args...)
  return diag.add(&diagnostic{ diagError, pos, nil, s, nil })
}
func (diag *Diagnostic) info(f string, args... interface{}) *diagnostic {
  return diag.add(&diagnostic{ diagInfo, Position{}, nil, fmt.Sprintf(strings.TrimSpace(f), args...), nil })
}
func (diag *Diagnostic) warn(f string, args... interface{}) *diagnostic {
  return diag.add(&diagnostic{ diagWarn, Position{}, nil, fmt.Sprintf(strings.TrimSpace(f), args...), nil })
}
func (diag *Diagnostic) error(f string, args... interface{}) *diagnostic {
  var s = fmt.Sprintf(strings.TrimSpace(f), args...)
  return diag.add(&diagnostic{ diagError, Position{}, nil, s, nil })
}
func (diag *Diagnostic) prompt(f string, args... interface{}) *diagnostic {
  var s = fmt.Sprintf(f, args...)
  return diag.add(&diagnostic{ diagPrompt, Position{}, nil, s, nil })
}

func (diag *Diagnostic) numErrors() (num int) {
  diag.m.Lock(); defer diag.m.Unlock()
  for _, d := range diag.points {
    if d.dt == diagError { num += 1 }
  }
  return
}
func (diag *Diagnostic) checkErrors(reset bool) (num int) {
  diag.m.Lock(); defer diag.m.Unlock()
  for _, d := range diag.points {
    var (
      msg = d.message
      pos = d.getPosition().String()
    )
    switch ; d.dt {
    case diagPrompt: if msg != "" { fmt.Fprintf(stderr, "%s",    msg) }
    case diagInfo:  fmt.Fprintf(stderr, "%v:info: %s\n",    pos, msg)
    case diagWarn:  fmt.Fprintf(stderr, "%v:warning: %s\n", pos, msg)
    case diagError: fmt.Fprintf(stderr, "%v: %s\n",         pos, msg)
      num += 1
    }
    if len(d.stack) > 0 {
      fmt.Fprintf(stderr, "%s\n", bytes.TrimSpace(d.stack))
    }
    if num > 22 {
      fmt.Fprintf(stderr, "%v: too many errors (%d)\n", pos, num)
      break
    }
  }
  if reset { diag.points = []*diagnostic{} }
  return
}

type Context struct {
  workdir string
  prefix  string // FIXME: prefix for distribution
  globe   *Globe
  goals   *Def
  mode    *Def
  pairs []*Pair
  loader  *loader
  flagEntries map[string][]*RuleEntry
  flags []*Flag
  args map[Value][]Value
}

var (
  context Context
  diag    Diagnostic
)

func current() (proj *Project) {
  if len(cloctx) > 0 && cloctx[0].project != nil {
    proj = cloctx[0].project
  } else if context.loader != nil { // for load time
    proj = context.loader.project
  }
  return
}

func (ctx *Context) run() (result []Value, breakers []*breaker) {
  if optionTraceLaunch { defer un(trace(t_launch, "Context.run")) }

  var main = ctx.globe.main
  if main == nil {
    fmt.Fprintf(stderr, "no targets to update `%v`", ctx.goals)
    return
  }

  defer setclosure(setclosure(cloctx.unshift(main.scope)))

  var done bool
  for _, flag := range ctx.flags {
    var ( s string; err error )
    if s, err = flag.name.Strval(); err != nil { return }
    var args, _ = ctx.args[flag]
    var entries, _ = ctx.flagEntries[s]
    for _, entry := range entries {
      var ( res []Value; brks []*breaker )
      if res, brks = entry.Execute(entry.position, args...); len(brks) > 0 {
        for _, brk := range brks {
          if brk.what == breakErro {
            diag.errorAt(entry.position, "execute '%v' failed: %v", entry, brk.error)
          }
        }
      }
      result = append(result, res...)
      done = true
    }
  }
  if done { return }

  var goals []Value
  for _, goal := range merge(ctx.goals.value) {
    switch t := goal.(type) {
    case *None: // just ignore
    case *Bareword:
      if entry, err := main.resolveEntry(t.string, false); err != nil {
        fmt.Fprintf(stderr, "%s\n", err)
      } else if entry == nil {
        fmt.Fprintf(stderr, "no such entry `%s`\n", t)
      } else {
        goals = append(goals, entry)
      }
    case *delegate:
      if s, err := t.Strval(); err != nil {
        fmt.Fprintf(stderr, "%s\n", err)
      } else if entry, err := main.resolveEntry(s, false); err != nil {
        fmt.Fprintf(stderr, "%s\n", err)
      } else if entry == nil {
        fmt.Fprintf(stderr, "no such entry `%s` (via `%v`)\n", s, t)
      } else {
        goals = append(goals, entry)
      }
    default:
      fmt.Fprintf(stderr, "unknown target (%s): %v\n", typeof(goal), goal)
    }
  }

  var updated int
  if len(goals) == 0 {
    if entry := main.DefaultEntry(); entry != nil {
      goals = append(goals, main.DefaultEntry())
    }
  }
  for _, goal := range goals {
    var args, _ = ctx.args[goal]
    result = append(result, updateGoal(goal, args)...)
    updated += 1
  }
  return
}

func updateGoal(goal Value, args []Value) (result []Value) {
  if !isNil(goal) {
    switch g := goal.(type) {
    case *RuleEntry:
      var brks []*breaker
      if result, brks = g.Execute(g.position, args...); len(brks) > 0 {
        for _, brk := range brks {
          if brk.what == breakErro {
            diag.errorAt(g.position, "execute '%v' failed: %v", g, brk.error)
          }
        }
      }
    default: diag.errorOf(goal, "'%v' is not an entry (%T)", goal, goal)
    }
  }
  return
}

func walkSmartBaseDirs(cwd string, vis func(string)bool) (s string) {
  s = cwd
  for s != "" {
    file := stat(Position{}, ".smart", "", s)
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

func joinTmpPath(base, rel string) string {
  if baseTmpPath == "" {
    var s = walkSmartBaseDirs(base, func(d string) bool {
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

// loadwork loads smart files, making it as individual func to avoid being
// abused by loaders.
func (ctx *Context) loadwork() (err error) {
  if optionTraceLaunch { defer un(trace(t_launch, "Context.loadwork")) }
  defer func(l *loader) { ctx.loader = l } (ctx.loader)

  var (
    base, _ = os.Getwd()
    sp = filepath.Join(base, ".smart", "modules")
    pos Position // FIXME: find a useful position
    args []Value
  )
  ctx.loader = &loader{
    Context:  ctx,
    fset:     token.NewFileSet(), 
    paths:    []string(globalPaths),
    loaded:   make(map[string]*Project),
    scope:    ctx.globe.scope,
  }
  ctx.goals = &Def{
    knownobject{objbase{scope:ctx.globe.scope}, "goals"},
    DefDefault, MakeNone(pos),
  }
  ctx.mode = &Def{
    knownobject{objbase{scope:ctx.globe.scope}, "mode"},
    DefDefault, MakeNone(pos),
  }

  if _, e := os.Stat(sp); e == nil { ctx.loader.AddSearchPaths(sp) }

  if text := strings.Join(os.Args[1:], " "); text == "" {
    // Relax!
  } else if args = ctx.loader.loadText("@", text); err != nil {
    // ...
  } else if args, err = parseOpts(Position{}, &options, args...); err != nil {
    return
  }

  if v := options.reconfigure; v { options.configure = v }

  if options.verbose || options.benchImport {
    defer func(t time.Time) {
      var d = time.Now().Sub(t)
      diag.prompt("Goals %v (%s)\n", ctx.goals, d)
    } (time.Now())
  }

  ctx.args = make(map[Value][]Value)

  var mode = new(Bareword)
  for _, target := range args {
    switch t := target.(type) {
    case *Flag: ctx.flags = append(ctx.flags, t)
      if s, err := t.name.Strval(); err == nil && s == "clean" {
        mode.position, mode.string = t.position, "clean"
      }
    case *Pair: ctx.pairs = append(ctx.pairs, t)
    case *Argumented:
      ctx.args[t.value] = t.args
      if f, ok := t.value.(*Flag); ok {
        ctx.flags = append(ctx.flags, f)
      } else {
        ctx.goals.append(t.value)
      }
    default: ctx.goals.append(t)
    }
  }
  if mode.string == "" {
    if options.configure {
      mode.string = "configure"
    } else {
      mode.string = "goals"
    }
  }
  context.mode.value = mode

  defer func(t time.Time) {
    var d = time.Now().Sub(t)
    if options.verboseImport {
      var name string
      if p := ctx.loader.project; p != nil { name = p.name }
      fmt.Fprintf(stderr, "└·%s … (%s)\n", name, d)
    } else if d > 4999*time.Millisecond {
      diag.prompt("warning: long load time: %s !\n", d).debug(options.debug, 1)
    }
  } (time.Now())
  if options.verboseImport { fmt.Fprintf(stderr, "┌→%s\n", base) }

  if !ctx.loader.loadPath(base, nil) { return }
  if ctx.loader.globe.main == nil { fmt.Fprintf(stderr, "nothing loaded\n") }
  return
}

func CommandLine() {
  if s, err := os.Getwd(); err != nil { return } else {
    context.workdir = s
  }

  if optionTraceLaunch { defer un(trace(t_launch, "CommandLine")) }
  if optionEnableBenchmarks {
    var w *bufio.Writer
    var d = filepath.Join(context.workdir, "benchmarks")
    if err := os.MkdirAll(d, os.FileMode(0777)); err != nil {
      fmt.Fprintf(stderr, "MkdirAll: %s\n", err)
      return
    } else if f, err := ioutil.TempFile(d, "*.log"); err != nil {
      fmt.Fprintf(stderr, "TempFile: %s\n", err)
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
  walkSmartBaseDirs(context.workdir, func(s string) bool {
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
    if fi, _ := os.Stat(searchFile); fi == nil {
      continue
    }
    file, err := os.Open(searchFile)
    if err != nil { fmt.Fprintf(stderr, "%v", err); return }
    defer file.Close()
    r := bufio.NewReader(file)
    for err == nil {
      var line string
      if line, err = r.ReadString('\n'); err != nil {
        if err != io.EOF { fmt.Fprintf(stderr, "%v", err) }
        break
      } else {
        line = strings.TrimSpace(line)
      }
      if strings.HasPrefix(line, "#") {
        continue
      }
      line = filepath.Clean(filepath.Join(s, line))
      if fi, err := os.Stat(line); err == nil && fi.IsDir() {
        globalPaths = append(globalPaths, line)
      }
    }
  }

  if diag.checkErrors(true) > 0 { return }

  //loadGrepCache()

  defer func(globe *Globe) {
    saveGrepCache()
    context.globe = globe
  } (context.globe)
  context.globe = NewGlobe("smart")
  context.flagEntries = make(map[string][]*RuleEntry)

  if err := context.loadwork(); err != nil {
    if diag.checkErrors(true) > 0 { return } //report(err)
  } else if options.help {
    do_helpscreen()
  } else if options.printFlags {
    print_flag_trace()
  } else if options.printConfig {
    print_configuration()
  } else if numUpdatedPlugins > 0 { // see buildPlugin
    fmt.Fprintf(stderr, "smart: Plugin updated, please relaunch.\n")
  } else if options.configure {
    do_configuration()
    if diag.checkErrors(true) > 0 { return }
  } else if result, err := context.run(); err != nil {
    printLeavingDirectory()
  } else if result != nil {
    for _, v := range result {
      var ( s string; err error )
      if isNil(v) {
        // does nothing
      } else if s, err = v.Strval(); err != nil {
        fmt.Fprintf(stderr, "%s [%s]", v, err)
      } else {
        fmt.Fprintf(stderr, "%s", s)
      }
    }
    fmt.Fprintf(stderr, "\n")

    printLeavingDirectory()
  }

  if diag.checkErrors(true) > 0 { return }
}
