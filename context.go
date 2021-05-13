//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
  "extbit.io/smart/token"
  "path/filepath"
  "strings"
  "bufio"
  "time"
  "fmt"
  "os"
  "io"
  "io/ioutil"
)

var (
  optionHelp = false
  optionClean = false
  optionConfigure = false
  optionReconfig = false
  optionAlwaysBuildPlugins = false
  optionVerbose = false
  optionVerboseUsing = false
  optionVerboseParsing = false
  optionVerboseLoading = false
  optionVerboseImport = false
  optionVerboseChecks = false
  optionBenchImport = false
  optionBenchSlow = false
  optionBenchBuiltin = false
  optionPrintConfiguration = false
  optionPrintFlags = false
  optionPrintStack = false
  optionNoExec = false

  // Tracking options
  optionTraceLaunch = false
  optionTraceParsing = false
  optionTraceTraversal = false
  optionTraceTraversalNestIndent = true
  optionTraceExecutor = false
  optionTraceExec = false
  optionTraceEntering = optionTraceTraversal && false
  optionTraceConfig = false

  // Return error if wildcard files not found.
  optionWildcardMissingError = false

  optionSaveGrepSourceName = false
)

type diagType int
const (
  diagInfo diagType = iota
  diagWarn
  diagError
)

type diagnostic struct {
  dt diagType
  position Position
  value Value
  message string
}
func (d *diagnostic) getPosition() Position {
  if isNil(d.value) {
    return d.position
  } else {
    return d.value.Position()
  }
}

type Diagnostic struct {
  points []*diagnostic
}
func (diag *Diagnostic) infoOf(value Value, f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagInfo, Position{}, value, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) warnOf(value Value, f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagWarn, Position{}, value, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) errorOf(value Value, f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagError, Position{}, value, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) infoAt(pos Position, f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagInfo, pos, nil, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) warnAt(pos Position, f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagWarn, pos, nil, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) errorAt(pos Position, f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagError, pos, nil, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) info(f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagInfo, Position{}, nil, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) warn(f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagWarn, Position{}, nil, fmt.Sprintf(f, args...) })
}
func (diag *Diagnostic) error(f string, args... interface{}) {
  diag.points = append(diag.points, &diagnostic{ diagError, Position{}, nil, fmt.Sprintf(f, args...) })
}


func (diag *Diagnostic) checkErrors(reset bool) (num int) {
  for _, d := range diag.points {
    switch d.dt {
    case diagInfo:  fmt.Fprintf(stderr, "%v:info: %s\n",    d.getPosition(), d.message)
    case diagWarn:  fmt.Fprintf(stderr, "%v:warning: %s\n", d.getPosition(), d.message)
    case diagError: fmt.Fprintf(stderr, "%v: %s\n",         d.getPosition(), d.message)
      num += 1
    }
    if num > 22 {
      fmt.Fprintf(stderr, "%v: too many errors\n", d.getPosition())
      break
    }
  }
  if reset {
    diag.points = []*diagnostic{}
  }
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
      res, brks = entry.Execute(entry.position, args...)
      if len(brks) == 0 {
        result = append(result, res...)
        done = true
      } else {
        return
      }
    }
  }
  if done { return }

  var goals []Value
  for _, goal := range merge(ctx.goals.value) {
    switch t := goal.(type) {
    case *None: // just ignore
    case *Bareword:
      if entry, err := main.resolveEntry(t.string); err != nil {
        fmt.Fprintf(stderr, "%s\n", err)
      } else if entry == nil {
        fmt.Fprintf(stderr, "no such entry `%s`\n", t)
      } else {
        goals = append(goals, entry)
      }
    case *delegate:
      if s, err := t.Strval(); err != nil {
        fmt.Fprintf(stderr, "%s\n", err)
      } else if entry, err := main.resolveEntry(s); err != nil {
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
    var res []Value
    var brks []*breaker
    var args, _ = ctx.args[goal]
    if res, brks = updateGoal(goal, args); len(brks) > 0 {
      break
    } else {
      result = append(result, res...)
      updated += 1
    }
  }
  return
}

func updateGoal(goal Value, args []Value) (result []Value, breakers []*breaker) {
  if !isNil(goal) {
    switch g := goal.(type) {
    case *RuleEntry: result, breakers = g.Execute(g.position, args...)
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
    knownobject{trivialobject{scope:ctx.globe.scope}, "goals"},
    DefDefault, &None{trivial{pos}},
  }
  ctx.mode = &Def{
    knownobject{trivialobject{scope:ctx.globe.scope}, "mode"},
    DefDefault, &None{trivial{pos}},
  }

  if _, e := os.Stat(sp); e == nil { ctx.loader.AddSearchPaths(sp) }
  if optionVerbose || optionBenchImport {
    defer func(t time.Time) {
      var d = time.Now().Sub(t)
      fmt.Fprintf(stderr, "smart: Goals %v (%s)\n", ctx.goals, d)
    } (time.Now())
  }

  if text := strings.Join(os.Args[1:], " "); text == "" {
    // Relax!
  } else if args = ctx.loader.loadText("@", text); err != nil {
    // ...
  } else if args, err = tryParseFlags(args, []string{
    /* TODO: using struct and field tags: */
    /* type struct { */
    /*   optionHelp string `h,help` */
    /* } */
    "h,help",
    "d,debug",
    "d,print-stack",
    "o,print-options",
    "f,print-flags",
    "b,build-plugins",
    "n,bench-import",
    "e,bench-builtins",
    "v,verbose",
    "i,verbose-import",
    "c,verbose-checks",
    "l,verbose-loading",
    "p,verbose-parsing",
    "u,verbose-using",
    "r,reconfigure",
    "g,configure",
    "m,no-exec", // optionNoExec
  }, func(ru rune, v Value) {
    switch ru {
    case 'h': if optionHelp              , err = trueVal(v, true); err != nil { return }
    case 'b': if optionAlwaysBuildPlugins, err = trueVal(v, true); err != nil { return }
    case 'o': if optionPrintConfiguration, err = trueVal(v, true); err != nil { return }
    case 'f': if optionPrintFlags        , err = trueVal(v, true); err != nil { return }
    case 'd': if optionPrintStack        , err = trueVal(v, true); err != nil { return }
    case 'n': if optionBenchImport       , err = trueVal(v, true); err != nil { return }
    case 'e': if optionBenchBuiltin      , err = trueVal(v, true); err != nil { return }
    case 'v': if optionVerbose           , err = trueVal(v, true); err != nil { return }
    case 'i': if optionVerboseImport     , err = trueVal(v, true); err != nil { return }
    case 'c': if optionVerboseChecks     , err = trueVal(v, true); err != nil { return }
    case 'p': if optionVerboseParsing    , err = trueVal(v, true); err != nil { return }
    case 'l': if optionVerboseLoading    , err = trueVal(v, true); err != nil { return }
    case 'u': if optionVerboseUsing      , err = trueVal(v, true); err != nil { return }
    case 'g': if optionConfigure         , err = trueVal(v, true); err != nil { return }
    case 'm': if optionNoExec            , err = trueVal(v, true); err != nil { return }
    case 'r': if optionReconfig          , err = trueVal(v, true); err != nil { return }
      optionConfigure = optionReconfig
    }
  }); err != nil { return }

  ctx.args = make(map[Value][]Value)
  for _, target := range args {
    switch t := target.(type) {
    case *Flag: ctx.flags = append(ctx.flags, t)
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

  var mode string
  if optionConfigure { mode = "configure" } else { mode = "goals" }
  context.mode.value = &Bareword{string:mode}

  defer func(t time.Time) {
    var d = time.Now().Sub(t)
    if optionVerboseImport {
      var name string
      if p := ctx.loader.project; p != nil { name = p.name }
      fmt.Fprintf(stderr, "└·%s … (%s)\n", name, d)
    } else if d > 5000*time.Millisecond {
      fmt.Fprintf(stderr, "smart: Long load time: %s !\n", d)
    }
  } (time.Now())
  if optionVerboseImport { fmt.Fprintf(stderr, "┌→%s\n", base) }

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
  } else if optionHelp {
    do_helpscreen()
  } else if optionPrintFlags {
    print_flag_trace()
  } else if optionPrintConfiguration {
    print_configuration()
  } else if numUpdatedPlugins > 0 { // see buildPlugin
    fmt.Fprintf(stderr, "smart: Plugin updated, please relaunch.\n")
  } else if optionConfigure {
    do_configuration()
    if diag.checkErrors(true) > 0 { return }
  } else if result, err := context.run(); err != nil {
    /*defer printLeavingDirectory()

    var brks, errs = extractBreakers(err)
    for _, e := range brks {
      switch e.what {
    default: report(e)
  case breakDone, breakCase:
        // just relax
      }
    }*/

    printLeavingDirectory()
  } else if result != nil {
    for _, v := range result {
      var ( s string; err error )
      if s, err = v.Strval(); err != nil {
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
