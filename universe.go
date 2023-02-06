//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package smart

import (
    "extbit.io/smart/token"
    "path/filepath"
    // "runtime/debug"
    "runtime/pprof"
    "runtime"
    "strconv"
    "strings"
    "time"
    "flag"
    "fmt"
    "os"
)

const maxNumVarVal = 9

var (
    universe universeContext
)

func init() {
    flag.Var(&universe.paths, "search", "comma-separated list of search paths")
    universe.init()
}

type searchlist []string
func (sl *searchlist) String() string { return fmt.Sprint(*sl) }
func (sl *searchlist) Set(value string) error {
    *sl = append(*sl, strings.Split(value, ",")...)
    return nil
}

type universeContext struct {
    diagContext

    workdir  string
    prefix   string // FIXME: prefix for distribution

    scope   *Scope
    globe   *Globe

    paths   searchlist
}
func (ctx *universeContext) arguments() []Value { return nil }
func (ctx *universeContext) argumented() *argumentedContext { return nil }
func (ctx *universeContext) argumentedSet([]Value) []Value { return nil }
func (ctx *universeContext) aquireLock() (unlock func()) { return nil }
func (ctx *universeContext) universe() *universeContext { return ctx }
func (ctx *universeContext) loader() *loader { return ctx.globe.top }
func (ctx *universeContext) inner() Context { return nil }
func (ctx *universeContext) spawn(c Context) Context { return c }
func (ctx *universeContext) auto() *autoContext { return nil }
func (ctx *universeContext) closure() *closureContext { return nil }
func (ctx *universeContext) travestates() *travestates { return nil }
func (ctx *universeContext) traversed(target Value) []Value { fail(ctx.Position(), "%v", target); return nil }
func (ctx *universeContext) entry() Entry { return nil }
func (ctx *universeContext) entryContext() *entryContext { return nil }
func (ctx *universeContext) stems() []string { return nil }
func (ctx *universeContext) stemmed() *stemmed { return nil }
func (ctx *universeContext) stemmedContext() *stemmedContext { return nil }
func (ctx *universeContext) Scope() *Scope { return ctx.globe/*.main*/.scope }
func (ctx *universeContext) Project() *Project { return ctx.globe.main }
func (ctx *universeContext) projects(_ Context, projs ...*Project) []*Project {
    if len(projs) > 0 { fail(ctx.Position(), "%v", projs) }
    return nil
}
func (ctx *universeContext) program() *Program { return nil }
func (ctx *universeContext) programContext() *programContext { return nil }
func (ctx *universeContext) positionContext() *positionContext { return nil }
func (ctx *universeContext) Position() (res Position) {
    res.Filename, res.Line = ctx.workdir, 1
    return
}
func (ctx *universeContext) wait() {}
func (ctx *universeContext) appendCallerUpdated() bool { return false }
func (ctx *universeContext) mustExists() bool { return false }
func (ctx *universeContext) WorkDir() string { return ctx.workdir }
func (ctx *universeContext) Globe() *Globe { return ctx.globe }
func (ctx *universeContext) String() (s string) {
    if fullContextStringer { s = "default" }
    return
}
func (ctx *universeContext) configuration() bool { return false }
func (ctx *universeContext) colonResolve(name string) (obj Object, found bool) {
    switch g := ctx.globe; name {
    case "os"   : obj, found = g.os.self, true
    case "goals": obj, found = g.goals,   true
    case "mode" : obj, found = g.mode,    true
    }
    return
}
func (ctx *universeContext) closureResolveAuto(name string) (obj Object, found bool) { return ctx.colonResolve(name) }
func (ctx *universeContext) autoArgs(_ []*def, _ []Value) ([]string, error) { return nil, nil }
func (ctx *universeContext) autoSet(name string, val Value) (def *def, res Value) {
    if false {
        prompt(ctx, "%v: can't set auto in default context, value=%v\n", name, val)
        errostack(ctx, 8, `(%T): %v`, ctx, name).debug(64)
    }
    return
}
func (ctx *universeContext) autoGet(name string) (res *def) {
    if obj, y := ctx.closureResolveAuto(name); y { res, y = obj.(*def) }
    return
}
func (ctx *universeContext) closureScopes() (scopes []*Scope) {
    if m := ctx.globe.main; m != nil && m.scope != nil {
        if false { scopes = append(scopes, m.scope) }
    }
    return
}
func (ctx *universeContext) dirtyOpts() *modifierSetDirtyPatsOpts { return nil }
func (ctx *universeContext) dirtyMark(vals ...Value) { return }

func (ctx *universeContext) help()       { do_helpscreen(ctx) }
func (ctx *universeContext) helpFlags()  { print_flag_trace(ctx) }
func (ctx *universeContext) helpConfig() { print_configuration(ctx) }

func (ctx *universeContext) init() {
    if s, e := os.Getwd(); e != nil {
        erro(ctx, "%v", e).debug(6)
        return
    } else {
        ctx.workdir = s
    }
    ctx.Context = ctx // self context for diagnostic

    var (
        pos Position = ctx.Position()
        bin = MakeString(pos, os.Args[0])
        args = MakeList(pos)
    )
    for _, a := range os.Args[1:] {
        args.Elems = append(args.Elems, MakeString(pos, a))
    }

    ctx.scope = NewScope(pos, nil, nil, `universe`)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART.ARGS", args)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART.BIN", bin)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART", bin)

    for name, f := range builtins {
        if _, alt := ctx.scope.Builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
    }
    for name, f := range commands {
        if _, alt := ctx.scope.Builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined (command)", name))
        }
    }

    ctx.globe = &Globe{
        scope: NewScope(ctx.Position(), ctx.scope, nil, `globe "smart"`),
        fset: token.NewFileSet(), // the global fileset
        loaded: make(map[string]*Project),
        args: make(map[Value][]Value),
        flagEntries: make(map[string][]Entry),
        //_timestamps: make(map[string]time.Time),
        //_timestampx: new(sync.Mutex),
    }

    var absPath, relPath, tmpPath, spec string
    // TODO: determines absPath, relPath, tmpPath, spec
    ctx.globe.os = ctx.globe.project(ctx, nil, absPath, relPath, tmpPath, spec, runtime.GOOS)
    //ctx.globe.os.scope.define(g.os, "name", &None{})
}

func (dc *universeContext) AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil { break }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
            dc.paths = append(dc.paths, s)
        } else {
            return fmt.Errorf("path '%s' is not dir", s)
        }
    }
    return nil
}

func (dc *universeContext) search(linfo *loadinfo, specName string) (absPath string, isDir bool, err error) {
    var fi os.FileInfo
    if specName == "." {
        err = fmt.Errorf("Not possible to chain itself.")
    } else if abs := filepath.IsAbs(specName); abs ||
        specName == "~" || specName == ".." ||
        strings.HasPrefix(specName, "~\\") ||
        strings.HasPrefix(specName, "~/") ||
        strings.HasPrefix(specName, "./") ||
        strings.HasPrefix(specName, "../") ||
        strings.HasPrefix(specName, ".\\") ||
        strings.HasPrefix(specName, "..\\") {
        var (
            s = specName
            sx string
        )
        if !abs && linfo.absDir != "" {
            sx = filepath.Join(linfo.absDir, s)
            if a, e := filepath.Abs(sx); e == nil {
                s = a
            } else {
                err = e
                return
            }
        }
        if fi, err = os.Stat(s); err != nil {
            sx = s + ".smart"
            if fi, er := os.Stat(sx); fi != nil {
                isDir, absPath, err = fi.IsDir(), sx, er
                return
            }
            sx = s + ".sm"
            if fi, er := os.Stat(sx); fi != nil {
                isDir, absPath, err = fi.IsDir(), sx, er
                return
            }
        } else {
            isDir, absPath = fi.IsDir(), s
        }
    } else {
        for _, base := range universe.paths {
            var s string
            if filepath.IsAbs(base) {
                s = filepath.Join(base, specName)
            } else {
                s = filepath.Join(dc.workdir, base, specName)
            }
            if fi, err = os.Stat(s); err == nil && fi != nil {
                isDir, absPath = fi.IsDir(), s
                return
            }
        }
    }
    return
}

func (dc *universeContext) run() (result []Value, travestates []*travestate) {
    if options.noRun { return }

    var main = dc.globe.main
    if main == nil {
        erro(dc, "no targets to update `%v`", dc.globe.goals).debug(1)
        return
    }

    var ctx Context = &closureContext{dc, []*Scope{main.scope}}
    if options.verbose { info(ctx, "goal: %v", main).debug(1) }

    removeTempDirs(ctx)

    if options.cpuProf != "" || options.autoProfs {
        var prof = options.cpuProf
        if prof == "" {
            var s = "run.cpu.auto.prof"
            if file := main.matchTempFile(ctx, s); file == nil {
                prof = filepath.Join(baseWorkDir, s)
            } else {
                prof = file.fullname()
            }
        }

        if f, e := os.Create(prof); e != nil {
            erro(dc, "%T: %v", e, e).debug(1)
            return
        } else {
            defer f.Close()
            if e := pprof.StartCPUProfile(f); e != nil {
                erro(dc, "could not start CPU profile: %v", e).debug(1)
                return
            }
            defer pprof.StopCPUProfile()
        }
    }

    if options.memProf != "" || options.autoProfs {
        var prof = options.memProf
        if prof == "" {
            var s = "run.mem.auto.prof"
            if file := main.matchTempFile(ctx, s); file == nil {
                prof = filepath.Join(baseWorkDir, s)
            } else {
                prof = file.fullname()
            }
        }
        defer func() {
            if f, e := os.Create(prof); e != nil {
                erro(dc, "%v", e).debug(1)
                return
            } else {
                defer f.Close()
                runtime.GC() // update memory statistics
                if e := pprof.WriteHeapProfile(f); e != nil {
                    erro(dc, "could not start CPU profile: %v", e).debug(1)
                    return
                }
            }
        } ()
    }

    var done bool
    for _, flag := range dc.globe.flags {
        var s = flag.name.Strval(ctx)
        var args, _ = dc.globe.args[flag]
        var entries, _ = dc.globe.flagEntries[s]
        for _, entry := range entries {
            var ( res []Value; traves []*travestate )
            if res, traves = entry.Execute(at(ctx, entry.Position()), args...); len(traves) > 0 {
                for _, brk := range traves {
                    if brk.what == traveFail {
                        erro(ctx, "execute '%v': %v", entry, brk).at(brk.pos).debug(1)
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
                if entries := proj.resolveEntries(ctx, t.string, false, true); entries == nil {
                    erro(ctx, "no such entry `%s`", t.string).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case *delegate:
                var s = t.Strval(ctx)
                if entries := proj.resolveEntries(ctx, s, false, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case *Flag:
                var s = t.Strval(ctx)
                if entries := proj.resolveEntries(ctx, s, false, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case *Argumented:
                {
                    // For examples:
                    //     project-name(-clean)
                    //     project/spec(-clean)
                    //     xxxx()
                    var (
                        s = t.value.Strval(ctx)
                        args = merge(t.args...)
                        found int
                    )
                    for _, p := range dc.globe.loaded {
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
            result = append(result, updateGoal(ctx, goal, args)...)
            updated += 1
        }
    }
    return
}

// load loads smart files, making it as individual func to avoid being abused by loaders.
func (dc *universeContext) loadTopWork() (err error) {
    if options.traceLaunch { defer un(trace(t_launch, "universeContext.load")) }
    defer func(l *loader) { dc.globe.top = l } (dc.globe.top)

    var (
        ctx Context = dc
        base = baseWorkDir
        pos = positionForDir(base) // FIXME: find a useful position
        args []Value
    )
    if s := filepath.Join(base, ".smart", "modules"); /* s != "" */true {
        if _, e := os.Stat(s); e == nil { dc.AddSearchPaths(s) }
    }
    if f := filepath.Join(base, "do.smart"); f != "" {
        if _, e := os.Stat(f); e != nil {
            f = filepath.Join(base, "build.smart")
            if _, e := os.Stat(f); e != nil { f = "" }
        }
        if f != "" {
            var pos Position
            pos.Filename = f
            pos.Line = 1
            // pos.Column = 1
            ctx = at(ctx, pos)
        }
    }

    dc.globe.top = &loader{
        closureContext: closureContext{ctx, []*Scope{dc.globe.scope}},
    }
    dc.globe.goals = &def{
        knownobject: knownobject{objbase{scope:dc.globe.scope}, "goals"},
        origin: DefDefault, value: MakeNone(pos),
    }
    dc.globe.mode = &def{
        knownobject: knownobject{objbase{scope:dc.globe.scope}, "mode"},
        origin: DefDefault, value: MakeNone(pos),
    }

    if text := strings.Join(os.Args[1:], " "); text == "" {
        // Relax!
    } else if args = dc.globe.top.loadText("@", text); len(args) == 0 {
        // ohh...
    } else {
        args = parseOpts(ctx, &options, 0, args...)
    }

    if v := options.reconfigure; v { options.configure = v }
    if v := options.fastMode; v {
        // Turn off many things for fast mode:
        //options.noImportFiles = v
        options.noDepsGrep = v
        options.noDeps = v
        options.noGrep = v
    }

    if options.verbose {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            prompt(ctx, "Goals %v (%s)\n", dc.globe.goals, d)
        } (time.Now())
    }

    assert(dc.globe.args != nil, "globe args is nil")

    if options.autoProfs {
        if f, e := os.Create(filepath.Join(baseWorkDir, "load.cpu.auto.prof")); e != nil {
            erro(ctx, "%v", e).debug(1)
            return
        } else {
            defer f.Close()
            if e := pprof.StartCPUProfile(f); e != nil {
                erro(ctx, "could not start CPU profile: %v", e).debug(1)
                return
            }
            defer pprof.StopCPUProfile()
        }
    }
    if options.autoProfs {
        defer func() {
            var prof string //= options.memProf
            if prof == "" { prof = filepath.Join(baseWorkDir, "load.mem.auto.prof") }
            if f, e := os.Create(prof); e != nil {
                erro(ctx, "%v", e).debug(1)
                return
            } else {
                defer f.Close()
                runtime.GC() // update memory statistics
                if e := pprof.WriteHeapProfile(f); e != nil {
                    erro(ctx, "could not start CPU profile: %v", e).debug(1)
                    return
                }
            }
        } ()
    }

    var mode = new(Bareword)
    for _, target := range args {
        switch t := target.(type) {
        case *Pair: dc.globe.pairs = append(dc.globe.pairs, t)
        case *Flag: dc.globe.flags = append(dc.globe.flags, t)
            if s := t.name.Strval(ctx); s == "clean" {
                mode.position, mode.string = t.position, "clean"
            }
        case *Argumented:
            dc.globe.args[t.value] = t.args
            if f, ok := t.value.(*Flag); ok {
                dc.globe.flags = append(dc.globe.flags, f)
            } else {
                dc.globe.goals.append(ctx, t/*.value*/)
            }
        default:
            dc.globe.goals.append(ctx, t)
        }
    }
    if mode.string == "" {
        if options.configure {
            mode.string = "configure"
        } else {
            mode.string = "goals"
        }
    }
    dc.globe.mode.value = mode

    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseImport {
            var name string
            if p := dc.globe.top.Project(); p != nil { name = p.name }
            fmt.Fprintf(stderr, "└·%s … (%s)\n", name, d)
        } else if d > 2999*time.Millisecond {
            if m := dc.globe.main; m != nil {
                prompt(ctx, "%v:warning: long loading: %s !!\n", m.position, d).debug(6)
            } else {
                prompt(ctx, "%s:1:warning: long loading: %s !!\n", base, d).debug(6)
            }
        }
    } (time.Now())
    if options.verboseImport { fmt.Fprintf(stderr, "┌→%s\n", base) }

    if !dc.globe.top.loadPath(base, nil) { return }
    if dc.globe.main == nil { fmt.Fprintf(stderr, "nothing loaded\n") }
    return
}

// A Globe represents a global execution context. 
type Globe struct {
    scope  *Scope

    fset    *token.FileSet
    loads []*loadinfo
    top     *loader

    os     *Project
    main   *Project
    loaded map[string]*Project // loaded projects

    stack  []map[string]*def

    args    map[Value][]Value
    flagEntries map[string][]Entry
    flags []*Flag
    pairs []*Pair
    goals   *def
    mode    *def
}

// Scope returns the globe scope.
func (g *Globe) Scope() *Scope { return g.scope }

// Main returns the main project.
func (g *Globe) Main() *Project { return g.main }

func (g *Globe) SetScopeOuter(scope *Scope) { scope.outer = g.scope }

func (g *Globe) AddFlagEntry(name string, entry Entry) {
    flags, _ := g.flagEntries[name]
    flags     = append(flags, entry)
    g.flagEntries[name] = flags
    return
}

func (g *Globe) file(filename string, src []byte) *token.File {
    return g.fset.AddFile(filename, -1, len(src))
}

// project returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *Globe) project(ctx Context, outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *Project) {
    if outer == nil { outer = g.scope }

    m = &Project{
        position: ctx.Position(),
        absPath: absPath,
        relPath: relPath,
        tmpPath: tmpPath,
        use:  new(uselist),
        self: new(ProjectName),
        spec: spec,
        name: name,
    }
    m.scope = NewScope(m.position, outer, m, fmt.Sprintf("project %q", name))
    m.self.name = name
    m.self.scope = m.scope
    m.self.owner = m
    m.self.project = m
    m.use.name = "usee"
    m.use.scope = m.scope
    m.use.owner = m

    if g.main == nil && spec != "" && name != "@" && name != "~" {
        for outer != nil && outer != g.scope {
            if p := outer.project; p != nil && p.name == "@" {
                return
            }
            outer = outer.outer
        }
        if false {
            var def, _ = g.scope.define(ctx, DefAuto, "_", /*none*/nil)
            if enable_assertions { assert(def != nil, "'$_' is nil") }
            for i := 0; i <= maxNumVarVal; i += 1 {
                var def, _ = g.scope.define(ctx, DefAuto, strconv.Itoa(i), nil)
                if enable_assertions { assert(def != nil, "'$%d' is nil", i) }
            }
        }
        g.main = m
    }
    return
}

func AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil {
            break
        }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
           universe.paths = append(universe.paths, s)
        }
    }
    return
}
