//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package smart

import (
    "path/filepath"
    "runtime/pprof"
    "runtime"
    "reflect"
    "strings"
    "sync"
    "time"
    "fmt"
    "os"
)

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

func _universe(c Context) *universe { return cast[*universe](c) }

type universe struct {
    diagnostic
    commandline

    *scope

    // benchmark_builtin func(*builtin, *evocation, time.Time, reflect.Value)

    globe *globe
    fset  *FileSet

    workdir string
    prefix  string // FIXME: prefix for distribution
    paths   searchlist
    packages  map[string]packageinfo
    statcache map[string]*filebase // file.fullname() -> File
    statmutex sync.Mutex

    hooks hooks

    expand_n int32

    ddd string // debug parsing via `eval -ddd=example`, also project.dd
}
func (ctx *universe) String() string { return "universe" }
func (ctx *universe) inner() Context { return &ctx.diagnostic }
func (ctx *universe) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}
func (ctx *universe) _position() (p Position) {
    if ctx.globe != nil && ctx.globe.main != nil {
        return ctx.globe.main.position
    }

    p.Filename, p.Line, p.Column = _workdir(ctx), 0, 0
    return
}
func (ctx *universe) ts(t string) string {
    var s = ts(ctx.Context)
    if  s == "{}"  {
        s, _ = filepath.Rel(workBaseDir, ctx.workdir)
        s = bases(3, s, true)
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
    case act_on_erros:
        if ctx.panicFailureOnErrosFlushed {
            if 0 < t.i { panic(_failure(ctx, "got %d errors", t.i)) }
            res = true
        }
        return

    case get_position:
        var p Position
        p.Filename = ctx.workdir
        return

    case get_workdir: return ctx.workdir
    case get_project: if ctx.globe != nil { return ctx.globe.main }
    case get_scope: if ctx.scope != nil { return ctx.scope }
    // case get_closure_scopes:
    //     if m := ctx.globe.main; m != nil && m.scope != nil && false {
    //         return []*scope{ m.scope }
    //     }
    //     return
    }
    return ctx.diagnostic.do(_ctx, op)
}

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

  parallel        bool `p,par,para,parallel`

  fastMode        bool `f,fm,fast,fast-mode`
  panicFailureOnErrosFlushed bool `foe,fail-on-errors`
  errorUncache    bool `eu,error-uncache,error-no-cache`

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

    panicFailureOnErrosFlushed: true,
    silentOptionalSelection: false,

    slow: 2999 * time.Millisecond,
}}

func new_universe(ii ...any) (ctx *universe) {
    ctx = &universe{}
    ctx.paths = searchPaths
    ctx.workdir = workBaseDir
    ctx.fset = NewFileSet()
    ctx.statcache = make(map[string]*filebase)
    ctx.scope = newscope(ctx._position(), nil, nil, `universe`)

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
    ctx.scope.set(ctx, "SMART.ARGS", defVoid, args)
    ctx.scope.set(ctx, "SMART.BIN",  defVoid, bin)
    ctx.scope.set(ctx, "SMART",      defVoid, bin)

    for name, f := range builtins {
        if _, alt := ctx.scope.builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
    }

    var os Value // one of darwin, freebsd, linux, and so on.
    var pos = ctx._position()
    {
        var vs []Value
        for _, s := range strings.Fields(runtime.GOOS) {
            vs = append(vs, makeWord(pos, s))
        }
        os = ease(ctx, vs)
    }

    ctx.globe = &globe{
        scope: newscope(pos, ctx.scope, nil, `globe`),
        flagEntries: make(map[string][]entry),
        loaded: make(map[string]*project),
        args: make(map[Value][]Value),
    }

    // FIXME: ctx.scope.scopename(ctx, ".GLOBE", ctx.globe.Scope)
    ctx.globe.os,    _ = ctx.globe.set(ctx, ".os",    defVoid, os)
    ctx.globe.goals, _ = ctx.globe.set(ctx, ".goals", defVoid, makeNone(pos))
    ctx.globe.mode,  _ = ctx.globe.set(ctx, ".mode",  defVoid, makeNull(pos))
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
func (p *filebase) exists() bool { return p.info != nil }

type stat_dir struct { string }
type stat_sub struct { string }
type stat_nonexist struct { bool }
type stat_fileinfo struct{ os.FileInfo }

func stat(ctx Context, name string, aa ...any) (_ *file) {
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
        default: erro(ctx, "invalid stat arg: %v", ts(a)).trace()
        }
    }

    var base *filebase
    var stub *filestub
    var fullname string
    var u = _universe(ctx)

    u.statmutex.Lock(); defer u.statmutex.Unlock()

    // Trims / suffix
    if dir != "" { dir = filepath.Clean(dir) }
    if sub != "" { sub = filepath.Clean(sub) }
    if false {
        var t = strings.HasPrefix(name, "./")
        if name!= "" { name = filepath.Clean(name) }
        if t         { name = "./" + name }
    }

    if filepath.IsAbs(name) {
        if fullname = name; dir == "" {
            //dir, sub = filepath.Dir(fullname), ""
            //name = filepath.Base(fullname)
        } else if strings.HasPrefix(fullname, dir+pathSep) {
            tail := fullname[len(dir)+1:]
            //sub  = filepath.Dir(tail)
            //name = filepath.Base(tail)
            if sub == "" { name = tail } else
            if strings.HasPrefix(fullname, sub+pathSep) {
                name = tail[len(sub)+1:]
            }
        } else if dir != "" {
            if true { dir = "" } else if false {
                erro(ctx, "dir name conflicts: %s <-> %s (sub=%v)", dir, name, sub).trace()
                unreachable("path error")
            } else {
                return
            }
        }
    } else if filepath.IsAbs(sub) {
        if fullname = filepath.Join(sub, name); dir == "" {
            dir = sub // trims / suffix
            sub = "" // .
        } else if sub == dir {
            sub = "" // .
        } else if strings.HasPrefix(sub, dir) {
            sub = strings.TrimPrefix(sub, dir)
            sub = strings.TrimPrefix(sub, pathSep)
            sub = filepath.Clean(sub)
        } else if false {
            dir = sub
            sub = ""
        } else {
            unreachable("conflicted sub/dir: ", sub, " ", dir) //return
        }
    } else if filepath.IsAbs(dir) {
        fullname = filepath.Join(dir, sub, name)
    } else {
        dir = filepath.Join(_workdir(ctx), dir)
        fullname = filepath.Join(dir, sub, name)
    }

    // NOTE: filepath.Join can have the same efffect as filepath.Clean
    var cleanFullname = filepath.Clean(fullname) // clean paths like /path/to/foo/../bar -> /path/to/bar
    if base, _ = u.statcache[cleanFullname]; base != nil {
        if base.info == nil {
            if fileInfo == nil { fileInfo, _ = os.Stat(fullname) }
            if fileInfo == nil && !nonexist { return nil }
            base.info = fileInfo
        }

        var head = &base.stub
        if enable_assertions {
            for stub = head; stub != nil ; stub = stub.other {
                s1, s2 := filepath.Join(stub.dir, stub.sub, stub.name), filepath.Join(fullname)
                assert(s1 == s2, "fullname '%s' conflicted:\n" +
                    "panic: (%s, %s, %s) %s\n" +
                    "panic: (%s, %s, %s) %s\n",
                    fullname,
                    stub.dir, stub.sub, stub.name, s1,
                    dir, sub, name, s2)
                if stub.other == head { break }
            }
        }

        for stub = head; stub != nil; stub = stub.other {
            if stub.dir == dir && stub.sub == sub && stub.name == name {
                goto GotFile
            }
            if stub.other == head { break }
        }

        stub = &filestub{ dir, sub, name, nil, head.other }
        head.other = stub
    } else {
        if fileInfo == nil {
            fileInfo, _ = os.Stat(fullname)
            if fileInfo == nil && !nonexist { return nil }
        }

        base = &filebase{filestub{ dir, sub, name, nil, nil }, fileInfo, false, nil, 0, 0, 0}
        base.stub.other = &base.stub
        stub = &base.stub
        u.statcache[cleanFullname] = base
    }

GotFile:
    return &file{valbase{_position(ctx)},base,stub}
}

func AddSearchPaths(paths... string) (err error) {
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

func (ctx *universe) AddSearchPaths(paths... string) (err error) {
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

func startCPUProfile(ctx Context, name string, heap ...bool) (stop func()) {
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
        erro(ctx, "%T: %v", e, e).trace()
    } else if e = pprof.StartCPUProfile(f); e != nil {
        erro(ctx, "%T: %v", e, e).trace()
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        if heap != nil && heap[0] { runtime.GC() // update memory statistics
            if e = pprof.WriteHeapProfile(f); e != nil {
                erro(ctx, "WriteHeapProfile: %v", e).trace()
            }
        }
        f.Close()
    }}
}

func startHeapProfile(ctx Context, name string) (stop func()) {
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
        erro(ctx, "%T: %v", e, e).trace()
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        runtime.GC() // update memory statistics
        if e = pprof.WriteHeapProfile(f); e != nil {
            erro(ctx, "WriteHeapProfile: %v", e).trace()
        }
        f.Close()
    }}
}

func updateGoal(ctx Context, goal Value, args []Value) (result []Value) {
    switch g := goal.(type) {
    case *rule:
        var y bool
        if result, y = execute_entry(ctx, g, args...); !y {
            erro(ctx, "update '%v' failed", g).trace()
        }
    default:
        erro(ctx, "not an entry: %v", ts(goal)).trace()
    }
    return
}

func (_tx *universe) run() (result []Value) {
    if _tx.noRun { return }

    var main = _tx.globe.main
    if main == nil {
        erro(_tx, "no targets to update `%v`", _tx.globe.goals).trace()
    }

    var ctx Context = closure_with(_tx, main.scope)
    if _tx.verbose { info(ctx, "goal: %v", main).debug() }

    removeTempDirs(ctx)

    if _tx.cpuProf != "" || _tx.autoProfs {
        var name = _tx.cpuProf
        if name == "" { name = "run.cpu.auto.prof" }
        defer startCPUProfile(ctx, name, true)()
    } else if _tx.memProf != "" || _tx.autoProfs {
        var name = _tx.memProf
        if name == "" { name = "run.mem.auto.prof" }
        defer startHeapProfile(ctx, name)()
    }

    var done bool
    for _, flag := range _tx.globe.flags {
        if _tx.verboseExecFlags { info(ctx, "%v", flag) }

        var s = flag.Value.string(ctx)
        var args, _ = _tx.globe.args[flag]
        var entries, _ = _tx.globe.flagEntries[s]
        for _, entry := range entries {
            if _tx.verboseExecFlags {
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
            if entry := proj.defaultEntry; entry != nil {
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
                    erro(ctx, "no such entry `%s`", t.s).trace()
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case *delegate:
                var s = t.string(ctx)
                if entries := proj._entries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).trace()
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case flag:
                var s = t.string(ctx)
                if entries := proj._entries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).trace()
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case *argumented:
                {
                    // For examples:
                    //     project-name(-clean)
                    //     project/spec(-clean)
                    //     xxxx()
                    var (
                        s = t.Value.string(ctx)
                        args = merge(t.args...)
                        found int
                    )
                    for _, p := range _tx.globe.loaded {
                        if p.name == s || p.spec == s { found += 1
                            if !collect(p, args) { return false }
                        }
                    }
                    if found == 0 {
                        erro(ctx, `"%s" not loaded: %v`, s, args).trace()
                        return false
                    }
                }
            default:
                errostack(ctx, 3, "%v: unknown target: %v (%s)", proj, goal, typeof(goal)).trace()
                return false
            }
        }
        return true
    }

    if collect(main, merge(_tx.globe.goals.value)) {
        if len(goals) == 0 {
            if entry := main.defaultEntry; entry != nil {
                goals = append(goals, entry)
            }
        }
        for _, goal := range goals {
            var args, _ = _tx.globe.args[goal]
            result = append(result, updateGoal(ctx, goal, args)...)
            updated += 1
        }
    }
    return
}

// load loads smart files, making it as individual func to avoid being abused by loaders.
func (u *universe) load(ctx Context) {
    if u.traceLaunch { defer un(l_trace(l_launch, "universe.load")) }

    var base = u.workdir
    if s := filepath.Join(base, ".smart", "modules"); s != "" {
        if _, e := os.Stat(s); e == nil { u.AddSearchPaths(s) }
    }
    if s := filepath.Join(base, mainFileName); s != "" {
        if _, e := os.Stat(s); e != nil {
            s = filepath.Join(base, deprFileName)
            if _, e := os.Stat(s); e != nil { s = "" }
        }
    }

    defer func(l *loader) { u.globe.top = l } (u.globe.top)
    u.globe.top = &loader{term:term{ctx, u.globe.scope}}

    l := ul{u, u.globe.top}
    l.parse_args(base, os.Args[1:]...)

    if u.verbose {
        defer func(t time.Time) {
            prompt(ctx, "Goals %v (%s)\n", u.globe.goals, time.Now().Sub(t))
        } (time.Now())
    }

    if u.autoProfs {
        if f, e := os.Create(filepath.Join(workBaseDir, "load.cpu.auto.prof")); e != nil {
            erro(ctx, "%v", e).trace()
        } else {
            defer f.Close()
            if e := pprof.StartCPUProfile(f); e != nil {
                erro(ctx, "could not start CPU profile: %v", e).trace()
            }
            defer pprof.StopCPUProfile()
        }
        defer func() {
            var prof string //= u.memProf
            if prof == "" { prof = filepath.Join(workBaseDir, "load.mem.auto.prof") }
            if f, e := os.Create(prof); e != nil {
                erro(ctx, "%v", e).trace()
            } else {
                defer f.Close()
                runtime.GC() // update memory statistics
                if e := pprof.WriteHeapProfile(f); e != nil {
                    erro(ctx, "could not start CPU profile: %v", e).trace()
                }
            }
        } ()
    }

    if u.verboseImport { prompt(ctx, "┌→%s\n", base) }

    defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseImport {
            var name string
            if p := _project(u.globe.top); p != nil { name = p.name }
            prompt(ctx, "└·%s … (%s)\n", name, d)
        } else if false && u.slow < d {
            if m := u.globe.main; m != nil {
                warn(ctx, "slow loading (%v)!!\n", d).debug(6)
            } else {
                prompt(ctx, "%s:1:warning: slow loading (%v)!!\n", base, d).debug(6)
            }
        }
    } (time.Now())

    spec, _ := filepath.Rel(workBaseDir, base)
    l.directory(l.loader, spec, base, nil)

    if l.globe.main == nil {
        erro(ctx, "nothing loaded").trace()
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

    if v := l.reconfigure; v { l.universe.configure = v }
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
        case flag: l.globe.flags = append(l.globe.flags, t)
            if s := t.Value.string(l.universe); s == "clean" {
                mode.position, mode.s = t.Position(), "clean"
            }
        case *argumented:
            l.globe.args[t.Value] = t.args
            if f, ok := t.Value.(flag); ok {
                l.globe.flags = append(l.globe.flags, f)
            } else {
                l.globe.goals.append(l.universe, t/*.Value*/)
            }
        default:
            l.globe.goals.append(l.universe, t)
        }
    }

    if mode.s == "" {
        if l.universe.configure {
            mode.s = "configure"
        } else {
            mode.s = "goals"
        }
    }

    l.globe.mode.value = mode
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
