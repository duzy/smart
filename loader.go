//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "extbit.io/smart/token"
    "bytes"
    "io/ioutil"
    "io"
    "unicode/utf8"
    "path/filepath"
    "strings"
    "plugin"
    "errors"
    "time"
    "flag"
    "fmt"
    "os/exec"
    "os"
)

const optSortErrors = false

type ResolveBits int
const (
    // If many bits are set, resolve in the listed priority.
    FromGlobe ResolveBits = 1<<iota
    FromBase
    FromProject
    FindDef
    FindRule

    FromHere

    // This is the default be
    anywhere = FromHere
    global = FromGlobe
    local = FromProject
    nonlocal = FromGlobe | FromBase | FromProject
)

type EvalBits int
const (
    KeepClosures EvalBits = 1<<iota
    KeepDelegates

    // Wants value for rule depends.
    DependValue

    // Wants v.Strval(ctx), expands delegates and closures,
    // turn off KeepClosures, KeepDelegates.
    StringValue = 0
)

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
//
type Mode uint

const (
    ModuleClauseOnly Mode = 1 << iota // stop parsing after project or module clause
    ImportsOnly               // stop parsing after import declarations
    ParseComments             // parse comments and add them to AST
    Flat                  // parsing in flat mode (donot create a new module)
    //Trace                 // print a trace of parsed productions
    DeclarationErrors         // report declaration errors
    SpuriousErrors            // same as AllErrors, for backward-compatibility
    //AllErrors = SpuriousErrors    // report all errors (not just the first 10 on different lines)
    parsingDir
)

const (
    dotContainer = ".container"
    dotConfigure = ".configure"
)

var parseMode = DeclarationErrors //|Trace

type searchlist []string

func (sl *searchlist) String() string {
    return fmt.Sprint(*sl)
}

func (sl *searchlist) Set(value string) error {
    *sl = append(*sl, strings.Split(value, ",")...)
    return nil
}

var globalPaths searchlist

func init() {
    flag.Var(&globalPaths, "search", "comma-separated list of search paths")
}

type declare struct {
    project *Project
    //backproj *Project
    backscope *Scope
    useesExecuted []*Project
}

type loadinfo struct {
    absDir string // absPath = filepath.Join(absDir, baseName)
    baseName string
    specName string
    useesExecuted []*Project
    loader *Project
    loadee *Project // the current loading project
    scopes []*Scope
    declares map[string]*declare // all projects declared in the load dir
}

func (li *loadinfo) absPath() string {
    return filepath.Join(li.absDir, li.baseName)
}

func (li *loadinfo) breakUseLoop() (result bool) {
    var first bool = true
    for _, decl := range li.declares {
        if first || result {
            result = decl.project.opts.breakUseLoop
        }
        if !result { break }
        first = false
    }
    return
}

type loader struct {
    closureContext
    *parser
    mode Mode // parsing mode
    project  *Project
    fset     *token.FileSet
    paths    searchlist
    loadArgs []Value
    loads    []*loadinfo
    loaded   map[string]*Project // loaded projects
    loadStack []*Project // load path
    useStack  []*Project // use path
    useesExecuted []*Project // all executed usees
    vs string // verbose prefix
    implicit bool // loading current project implicitly, aka. via foo.bar.Baz (implicit foo/bar loaded)
}
func (l *loader) inner() Context { return &l.closureContext }
func (l *loader) String() string { return fmt.Sprintf("loader{%s}", &l.closureContext) }
func (l *loader) Project() (project *Project) { return l.project }

func (i *loader) AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil {
            break
        }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
            i.paths = append(i.paths, s)
        } else {
            return errors.New(fmt.Sprintf("path '%s' is not dir", s))
        }
    }
    return nil
}

func restoreLoadingInfo(l *loader) {
    var (
        last = len(l.loads)-1
        linfo = l.loads[last]
    )

    l.loads = l.loads[0:last]
    l.useesExecuted = linfo.useesExecuted
    l.project = linfo.loader
    l.scopes = linfo.scopes //l.SetScope(linfo.scope)

    //assert(l.scope.project == linfo.loader, "scope/project mismatched")

    /*var names []string
    for _, declare := range linfo.declares {
        names = append(names, declare.project.Name())
    }

    if loader := linfo.loader; loader != nil {
        fmt.Fprintf(stderr, "exit: %v from '%s' → %v\n", names, loader.Name(), linfo.scope)
    } else {
        fmt.Fprintf(stderr, "exit: %v → %v\n", names, linfo.scope)
    } */
}

func saveLoadingInfo(l *loader, specName, absDir, baseName string) *loader {
    l.loads = append(l.loads, &loadinfo{
        absDir: absDir,
        baseName: baseName,
        specName: filepath.Clean(specName),
        useesExecuted: l.useesExecuted,
        loader:   l.project,
        scopes:   l.scopes,
        declares: make(map[string]*declare),
    })
    return l
}

func (l *loader) searchSpecPath(linfo *loadinfo, specName string) (absPath string, isDir bool, err error) {
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
        for _, base := range l.paths {
            var s string
            if filepath.IsAbs(base) {
                s = filepath.Join(base, specName)
            } else {
                s = filepath.Join(l.WorkDir(), base, specName)
            }
            if fi, err = os.Stat(s); err == nil && fi != nil {
                isDir, absPath = fi.IsDir(), s
                return
            }
        }
    }
    return
}

type genericoptions struct {
    keyword token.Token // e.g. use, files, eval, etc.
    dontOperate bool // e.g. -cond(false)
    options []Value // general options
    verbose bool // TODO: verbose operation
}

type useOpts struct {
	noUse    bool `nu,nouse;uu,unuse` // TODO
    noVars   bool `nv,novars;nv,no-vars`
	files    bool `f,files` // NOTE: see also '-import(xxxx)'
	filesPub bool `fp,files-pub;fp,files-public;pf,public-files`
    public   bool `p,pub;pub,public` // NOTE: work with -files flag
	reuse    bool `r,reuse;ru,reusing`
    vars   []Value `var,vars`
}

type useVarOpts struct {
	unique bool `u,uni,uniq,unique`
    args []Value
}
func (opts *useVarOpts) apply(ctx Context, def *Def, vals []Value) {
    if opts.unique {
        if def.append(ctx, vals...); len(opts.args) > 0 {
            var position = ctx.Position()
            var args = MakeList(position, opts.args...)
            def.value = builtinUnique(ctx, args, def.value)
        } else {
            def.value = builtinUnique(ctx, def.value)
        }
    }
}
func parseUseNameOpts(ctx Context, nameVal Value) (name string, opts useVarOpts) {
    if arged, ok := nameVal.(*Argumented); ok {
        var args []Value
		nameVal, args = arged.value, arged.args
        opts.args = parseOpts(ctx, &opts, expandmerge2(ctx, expandPlainValue, args...)...)
	}
	name = nameVal.Strval(ctx)
    return
}
func applyUseeVar(ctx Context, user, usee *Project, nameVal Value) {
    var name, opts = parseUseNameOpts(ctx, nameVal)
    if name == "" {
		erro(ctx, "%v: parse use name opts '%v' failed", user, nameVal)
        return
    }

	var (
        position = ctx.Position()
        none = MakeNone(position)
		def, alt = user.scope.define(ctx, DefVoid, name, none)
        isNewDef = !isNil(def) && isNil(alt)
	)
	if def == nil && !isNil(alt) { def, _ = alt.(*Def) }
	if def == nil {
		erro(ctx, `%v: "%s" is undefined`, user, name).of(nameVal).debug(1)
		return
    } else if false {
        isNewDef = /*isTrivial(def.value)*/true
    }
    if isNewDef && len(user.bases) > 0 {
        // 1: derive values from bases
        for _, base := range user.bases {
            if obj := base.resolveObject(ctx, name); isNil(obj) {
                continue
            } else if d, ok := obj.(*Def); ok && !isTrivial(d.value) {
                opts.apply(ctx, def, merge(d.value))
            }
        }
        // 2: apply using vars from bases
        for _, base := range user.bases {
            var obj Object
            if false {
                obj = base.scope.lookup("using."+name)
            } else if obj = base.resolveObject(ctx, "using."+name); isNil(obj) {
                continue
            } else if d, ok := obj.(*Def); ok && !isTrivial(d.value) {
                opts.apply(ctx, def, merge(d.value))
            }
        }
    }

    if v, e := user.using.Get(ctx, name/* NOTE: gets `using.%s` */); e != nil {
		erro(ctx, "%v: %v (using.%s)", user, e, name)
    } else if va := merge(v); len(va) > 0 {
        var c = closureWith(ctx, position, user.scope)
        for _, v := range va {
            if d, ok := v.(*Def); !ok {
                erro(ctx, "%s: not a Def: %T %v", name, v, v).debug(1)
            } else if isTrivial(d.value) {
                // does nothing
            } else {
                opts.apply(c, def, merge(d.value))
            }
        }
    }
    if false && name == "ldlibs" && strings.HasPrefix(user.name, "lld.") {
		info(ctx, "%v, %v: %v", user, usee, def)
		info(ctx, "%v, %v: %v %v", user, usee, ctx.Project(), ctx).debug(10)
	}
}
func applyUsingVar(ctx Context, user, usee *Project, nameVal Value) {
    var name, opts = parseUseNameOpts(ctx, nameVal)
    if name == "" {
		erro(ctx, "%v: parse use name opts '%v' failed", user, nameVal)
        return
    }

    var (
        position = ctx.Position()
        none = MakeNone(position)
        usingName = "using." + name
        def, alt = user.scope.define(ctx, DefVoid, usingName, none)
    )
    if def == nil && !isNil(alt) { def, _ = alt.(*Def) }
    if def == nil {
        erro(ctx, `%v: "%s" is undefined'`, user, name).of(nameVal).debug(1)
        return
    } else if isTrivial(def.value) {
        for _, base := range user.bases {
            if obj := base.resolveObject(ctx, usingName); isNil(obj) {
                continue
            } else if d, ok := obj.(*Def); ok && !isTrivial(d.value) {
                def.append(ctx, obj)
            }
        }
    }

    if o := usee.scope.Lookup(usingName); isNil(o) || isNone(o) {
        // does nothing
    } else if d, ok := o.(*Def); !ok {
        erro(ctx, "%s: not a Def: %T %v", name, o, o).debug(1)
    } else if isTrivial(d.value) {
        // does nothing
    } else {
        var c = closureWith(ctx, position, usee.scope)
        opts.apply(c, def, merge(d.value))
    }
    if def.name == "arflags" && strings.Contains(def.value.String(), "crs crs") {
        warn(ctx, "%v", def).debug(1)
    }
    if false && name == "ldlibs" && (strings.HasPrefix(user.name, "lld.") || strings.HasPrefix(usee.name, "lld.")) {
        info(ctx, "%v: %v: %v", user, usee, def)
        info(ctx, "%v: %v: %v", user, usee, ctx).debug(10)
    }
}
func applyUseeVars(ctx Context, user, usee *Project) {
    var spec Value
    if o := usee.resolveObject(ctx, "using.*"); o == nil {
        // erro(ctx, "resolve using.* failed").debug(1)
    } else if def, _ := o.(*Def); def != nil {
        spec = def.value
    }
    if !isTrivial(spec) {
        // NOTE: apply vars like 'cflags', 'cxxflags', ...
        for _, name := range merge(spec) { applyUseeVar(ctx, user, usee, name) }
    }
}
func applyUsingVars(ctx Context, user, usee *Project) {
    var spec Value
    if o := user.resolveObject(ctx, "using.*"); o == nil {
        // erro(ctx, "resolve using.* failed").at(ctx.Position()).debug(1)
    } else if def, _ := o.(*Def); def != nil {
        spec = def.value
    }
    if !isTrivial(spec) {
        // NOTE: apply vars like 'using.cflags', 'using.cxxflags', ...
        for _, value := range merge(spec) { applyUsingVar(ctx, user, usee, value) }
    }
}

func (l *loader) Position() Position { return l.parser.Position() }
func (l *loader) loadUseSpecName(opts useOpts, specVal Value, specName string, arged []Value, params ...Value) (loaded *Project) {
    var (
        linfo = l.loads[len(l.loads)-1]
        position = specVal.Position()
        ctx = positional(l, position)
        absPath string
        isDir bool
        err error
    )
    if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
        errostack(ctx, 3, "no such package `%v` (%T)", specName, specVal).of(specVal).debug(6)
        return
    } else if absPath == "" {
        errostack(ctx, 3, "missing `%s` (in %v)", specName, l.paths).of(specVal).debug(6)
        return
    }

    loaded, _ = l.loaded[absPath]

    // Checking circular loads. See also Project.loopImportPath()!
    var breakUseLoop bool
    for i, load := range l.loads {
        if load.absDir == absPath {
            var s string
            var loop, loopBreakers []*loadinfo
            for n := i; n < len(l.loads); n += 1 {
                var load = l.loads[n]
                loop = append(loop, load)
                if load.breakUseLoop() {
                    loopBreakers = append(loopBreakers, load)
                    s += "<" + load.specName + "> → "
                } else {
                    s += load.specName + " → "
                }
            }
            if loaded != nil && loaded.opts.breakUseLoop {
                s += "<" + specName + ">"
            } else {
                s += specName
            }

            if breakUseLoop = (loopBreakers != nil); !breakUseLoop {
                erro(ctx, "%s: loop detected: %s", l.project, s).of(specVal).debug(10)
            } else if options.verboseImport || options.verboseUsing || options.verboseLoads {
                prompt(ctx, "%s: loop detected: %v\n", l.project, s).debug(10)
            }
        }
    }

    var scope = l.Scope()
    if false && breakUseLoop {
        if loaded == nil {
            // ...
        } else if _, a := scope.ProjectName(ctx, loaded.name, loaded); a != nil {
            if val, ok := a.(*ProjectName); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, a).at(position).debug(1)
            }
        }
        return
    }

    defer func(a []*Project) {
        var scope = l.Scope()
        if loaded == nil {
            erro(ctx, "%v (%v,dir=%v) not loaded in %v", specName, absPath, isDir, scope).of(specVal).debug(32)
            return
        } else if name, _ := scope.Lookup(loaded.name).(*ProjectName); name == nil {
            if _, alt := scope.ProjectName(ctx, loaded.name, loaded); alt != nil {
                if val, ok := alt.(*ProjectName); !ok || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).at(position).debug(1)
                }
            }
            if false {
                erro(ctx, "%v (%v,dir=%v) not in %v", specName, absPath, isDir, l.scopes).of(specVal).debug(1)
                return
            }
        }
        l.loadStack = a
    } (l.loadStack)
    l.loadStack = append(l.loadStack, l.project) // build the load path

    // https://unicode-table.com/en/sets/arrows-symbols/
    // ┌────────────────────────────────┐
    // ├────────────────────────────────┼───┬──⇢·
    // ├──────────────────────┬────→┬←──┤   │    ⇡
    // ├┬─→───────────────────┼─────┴───┘   ├────┼⇢
    // │├┬───→             ↑  └──┬──┐       │    ⇣
    // ││└──→    ·         │     │  ├─⇥     ↓
    // │└──→───⇥─┴─⇤────┬──┴──┬──┘  │
    // └──→           ⇠─┘     ↓     └─→ ⇒ …
    if options.verboseImport {
        if len(l.loadStack) > 1 {
            defer func(s string) { l.vs = s } (l.vs)
            l.vs += "│"
        }
        if opts.reuse {
            fmt.Fprintf(stderr, "%s├┬→\"%s\" (reuse, %s)\n", l.vs, specName, absPath)
        } else {
            fmt.Fprintf(stderr, "%s├┬→\"%s\" (%s)\n", l.vs, specName, absPath)
        }
        defer func(t time.Time) {
            var name string
            var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s
            var ds = fmt.Sprintf("(%s)", d)
            if d>=1*time.Second { ds = fmt.Sprintf("▶%s◀",ds) }
            if loaded != nil { name = loaded.name }
            fmt.Fprintf(stderr, "%s├┴─\"%s\" ⇢ %s %s\n", l.vs, specName, name, ds)
        } (time.Now())
    }

    if loaded != nil && !(/*opts.noVars || */opts.reuse) {
        var ( proj *Project ; res, isb bool )
        if proj, res, isb, err = l.project.hasLoaded(ctx, loaded, breakUseLoop); err != nil {
            erro(ctx, "`%s`: %s", specName, err).of(specVal).debug(1)
            return
        } else if isb {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v is already a base\n", l.project, specName)
            erro(ctx, "`%s` is already a base (proj=%s)", specName, proj).of(specVal)
            errostack(ctx, 10, "%v", ctx).debug(16)
            return
        } else if res {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v already imported by %v\n", l.project, specName, proj)
            erro(ctx, "'%s' already imported by '%s'", specName, proj).of(specVal)
            errostack(ctx, 10, "%v", ctx).debug(16)
            return
        }
    }
    if loaded == nil {
        var okay bool
        if isDir {
            okay = l.loadDirWithArgs(position, specName, absPath, arged, nil)
        } else {
            okay = l.loadWithArgs(position, specName, absPath, arged, nil)
        }
        if !okay {
            erro(ctx, "failed loading `%v` (%v)", specName, absPath).of(specVal).debug(1)
            return
        }

        if loaded != nil {
            // already loaded previously
        } else if loaded, _ = l.loaded[absPath]; loaded != nil {
            // successfully loaded (first)
        } else {
            erro(ctx, "'%s' not loaded (%s)", specName, absPath).of(specVal).debug(1)
        }

        if loaded == nil {
            erro(ctx, "'%s' not smart project", specName).of(specVal).debug(1)
            return
        }
        if false && l.project.name == "llvm.Analysis" {
            warn(ctx, "%v, %v, %v", l.project, loaded, opts).debug(16)
        }
    }

    // Check against the current load list before appending loaded.
    for _, using := range l.project.using.list {
        var (
            up = using.project
            proj *Project
            res, isb bool
        )
        if loaded == up {
            if !opts.noVars && !opts.files {
                erro(ctx, "using `%s` multiple times", specName).of(specVal).debug(10)
            }
            return
        }

        if false && loaded.opts.multiUseAllowed {
            // ...
        } else if proj, res, isb, err = loaded.hasLoaded(ctx, up, breakUseLoop); err != nil {
            erro(ctx, "load '%s' failed: %s", specName, err).of(specVal).debug(1)
            return
        } else if isb {
            if l.project.hasBase(up) {
                // common bases are fine
            } else {
                erro(ctx, "`%s` is already a base", specName).of(specVal).debug(1)
            }
        } else if res && !using.opts.reuse && !up.opts.multiUseAllowed && !loaded.opts.multiUseAllowed {
            if true {
                warn(ctx, "`%s` has already imported `%s` (from %s)", loaded, up, proj).at(position)
                if loaded != up { warn(ctx, "project %s", loaded).at(loaded.position) }
                if proj != up { warn(ctx, "project %s", proj).at(proj.position) }
                warn(ctx, "project %s", up).at(up.position).debug(6)
            } else {
                warnstack(ctx, -1, "`%s` has already imported `%s` (from %s)", loaded, up, proj).at(position).debug(64)
            }
        }

        if proj, res, isb, err = up.hasLoaded(ctx, loaded, breakUseLoop); err != nil {
            erro(ctx, "load '%s' failed: %s", specName, err).of(specVal).debug(1)
            return
        } else if isb {
            warn(ctx, "`%s` is already base of `%s` (%s)", loaded, up, proj).at(position).debug(1)
        } else if res && !using.opts.reuse && !loaded.opts.multiUseAllowed {
            warn(ctx, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj).at(position)
            warnstack(ctx, 8, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj).debug(1)
        }
    }

    if options.verboseImport {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s ┼
            fmt.Fprintf(stderr, "%s├┤ %s:import(%s) (%s)\n", l.vs, l.project, specName, d)
        } (time.Now())
    }
    if err = l.addUsing(ctx, loaded, params, opts); err != nil {
        erro(ctx, "using '%v' failed: %v", loaded, err).debug(1)
        return
    }
    if opts.files || opts.filesPub {
        if false {
            l.importFileMaps(ctx, opts.public || opts.filesPub, specVal)
        } else {
            l.importFileMaps1(ctx, opts, loaded)
        }
    }
    return
}

const pluginDifferentVersionError = `plugin was built with a different version of package`
var numUpdatedPlugins = 0

func buildPlugin(s, src string) (err error) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.buildPlugin")) }

    fmt.Fprintf(stderr, "smart: Build %v …", src)
    dir, _ := filepath.Split(src)
    o := &bytes.Buffer{}
    c := exec.Command("go", "build", "-buildmode=plugin", "-o", s)
    c.Stdout, c.Stderr, c.Dir = o, o, dir
    if err = c.Run(); err == nil {
        numUpdatedPlugins += 1
        fmt.Fprintf(stderr, "… ok\n")
        fmt.Fprintf(stderr, "smart: Plugin updated, please relaunch.\n")
        os.Exit(0)
    } else {
        fmt.Fprintf(stderr, "… error\n")
        fmt.Fprintf(stderr, "%s", o)
    }
    return
}

func (l *loader) loadPlugin(pos Position) (err error) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadPlugin")) }
    if l.project == nil {
        erro(l, "current project is nil").at(pos).debug(32)
        return
    }

    var ctx Context = l
    var g = stat(ctx, "smart.go", "", l.project.absPath)
    if g == nil { return /* smart.go was not presented */ }

    var src = g.Strval(ctx)
    s := strings.Replace(l.project.relPath, "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath(ctx, "", "")), "plugins", s)

    var build = true

    so := stat(ctx, /*l.project.name*/"plugin", "", s, nil)
    if s = so.fullname(); s == "" {
        erro(ctx, "file '%v' has empty fullname", so).of(so)
        return
    } else if so.exists() && !options.buildPlugins {
        if so.info.ModTime().After(g.info.ModTime()) {
            build = false // Plugin already updated.
        }
    }
    if build { err = buildPlugin(s, src) }
    if err != nil { return }

    // Once plugin is opened, there's no need/way to close it.
    if l.project.plugin, err = plugin.Open(s); err == nil {
        var p plugin.Symbol
        if p, err = l.project.plugin.Lookup("Init"); err != nil {
            return
        } else if p == nil {
            // no initialization (optional)
        } else if f, ok := p.(func(Position, *Project) (*Scope, error)); ok {
            l.project.pluginScope, err = f(pos, l.project)
        } else if f, ok := p.(func(*Project) (*Scope, error)); ok {
            l.project.pluginScope, err = f(l.project)
        }
    } else if es := err.Error(); strings.Contains(es, pluginDifferentVersionError) {
        err = buildPlugin(s, src)
    }
    return
}

func (l *loader) addUsing(ctx Context, usee *Project, params []Value, opts useOpts) (err error) {
    // clocks:🕐🕑🕒🕓🕔🕕🕖🕗🕘🕙🕚🕛🕜🕝🕞🕟🕠🕡🕢🕣🕤🕥🕦🕧
    if options.verboseUsing {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            fmt.Fprintf(stderr, "using(%15s) %s ⇒ %v\n", d, l.project, l.project.using)
        } (time.Now())
    }

    if usee == l.project {
        erro(l, "'%v' use loop (%s)", usee.name, l.usePath())
        return
    } else if l.project.isUsingDirectly(usee) {
        return
    }

    defer func(a []*Project) { l.useStack = a } (l.useStack)
    l.useStack = append(l.useStack, usee) // build the use path

    // Add to the project using list, so that the use path is correct.
    if l.project.using.append(ctx, usee, params, opts); !opts.noVars {
        applyUseeVars(ctx, l.project, usee)  // aka. ABC += $(using.ABC)
        applyUsingVars(ctx, l.project, usee) // aka. using.ABC += $(using.ABC)
        if 0 < len(opts.vars) {
            for _, v := range opts.vars {
                warn(ctx, "var: %T %v", v, v).of(v)
            }
            warn(ctx, "TODO: %d vars to import", len(opts.vars)).debug(1)
        }
    }
    return
}

func (l *loader) usePath() (s string) {
    for i, u := range l.useStack {
        if i > 0 { s += "," }
        s += u.name
    }
    return
}

func iterateArgumentedIdentElems(ctx Context, elems, stems []Value, f func(elems, stems []Value)) {
    for i, elem := range elems {
        if a, ok := elem.(*Argumented); ok {
            var prefix, suffix = elems[:i], elems[i+1:]
            iterateArgumentedIdentifiers(ctx, a, func(ident Value, stems2 []Value) {
                var head   = append(prefix, ident)
                var stems3 = append(stems , stems2...)
                iterateArgumentedIdentElems(ctx, suffix, stems3, func(elems, stems []Value) {
                    f(append(head, elems...), stems)
                })
            })
            return
        }
    }
    f(elems, stems)
}

func iterateArgumentedIdentifiers(ctx Context, identifier Value, f func(ident Value, stem []Value)) {
    switch t := identifier.(type) {
    case *Argumented:
        var args = expandmerge2(ctx, expandPlainValue, t.args...)
        iterateArgumentedIdentifiers(ctx, t.value, func(ident Value, stems []Value) {
            var pos = ident.Position()
            for _, arg := range args { f(MakeBarecomp(pos, ident, arg), append(stems, arg)) }
        })
    case *Barecomp:
        iterateArgumentedIdentElems(ctx, t.Elems, nil, func(elems, stems []Value) {
            if len(stems) == 0 { f(t, stems) } else {
                f(MakeBarecomp(t.Position(), elems...), stems)
            }
        })
    default:
        f(t, nil)
    }
}

func (l *loader) determine(position Position, tok token.Token, identifier, value Value) (defs []*Def) {
    var ctx Context = positional(l, position)
    iterateArgumentedIdentifiers(ctx, identifier, func(ident Value, stems []Value) {
        var def = l.determine1(ctx, tok, ident, value)
        if false && l.project.name == "lld.tools.lld" && strings.HasPrefix(ident.String(), "ldlibs") {
            warn(ctx, "%v -> %v, %v -> %v", identifier, ident, stems, def).of(ident).debug(1)
        }
        defs = append(defs, def)
    })
    return
}

func (l *loader) determine1(ctx Context, tok token.Token, identifier, value Value) (def *Def) {
    var alt Object
    switch t := identifier.(type) {
    case *selection:
        var v = t.value(ctx)
        if d, ok := v.(*Def); ok {
            def = d
        } else {
            erro(ctx, "`%v` is not a def (%T)", t, v)
            return
        }

    case *Bareword, *Barecomp, *Qualiword, *Path, *Flag:
        var name = t.Strval(ctx)
        if _, ok := builtins[name]; ok {
            erro(ctx, "`%v` (%v) is builtin name", identifier, name)
            return
        }

        // Resolve base value to derive.
        var prev = l.project.resolveObject(ctx, name)

        if def, alt = l.def(identifier.Position(), name); alt == nil {
            // does nothing...
        } else if alt != nil && (tok == token.ASSIGN || tok == token.EXC_ASSIGN) {
            var ( okay bool; ad *Def )
            if ad, okay = alt.(*Def); !okay {
                erro(ctx, "`%v` already defined (%T) (%v,%v)", identifier, alt, alt.OwnerProject(), l.project).debug(1)
                return
            } else if ad.owner == l.project && ad.origin != DefConfRef {
                erro(ctx, "`%v` already defined (%T) (%v)", identifier, alt, l.project).debug(1)
                return
            } else {
                def = ad
            }
        } else if alt != nil {
            def = alt.(*Def)
        }

        if prev == nil {
            // no derived value
        } else if prev.OwnerProject() == l.project {
            // not derivable def if in the same project
        } else if derived, okay := prev.(*Def); !okay {
            // not a def
        } else if derived == nil {
            erro(ctx, "prev def '%s' is nil", name).debug(1)
        } else if derived == def || def.value.refs(ctx, derived) {
            // same def
        } else if tok == token.ADD_ASSIGN {
            if false {
                // Unshift the delegation to derive value.
                var delegate = MakeDelegate(ctx.Position(), token.LPAREN, derived)
                def.append(ctx, delegate)
            } else {
                def.append(ctx, derived.value)
            }
        }

    case *Argumented:
        var args = expandmerge2(ctx, expandPlainValue, t.args...)
        erro(ctx, "TODO: multiple defs: %v %v", t.value, args)
        return
    }
    if def == nil {
        erro(ctx, "def is nil for '%v' of %T", identifier, identifier).debug(1)
        return
    }

    // Ensures that all immediate assignments are in the current
    // project context.
    //defer setclosure(setclosure(cloctx.unshift(l.scope)))

    if !def.position.IsValid() {
        def.position = identifier.Position()
    }
    l.assign(tok, def, alt, value)
    return
}

func (l *loader) rule(clause *parsedRuleData) (entries []Entry) {
    var (
        ctx = positional(l, l.Position())
        params  []*Def
        depends []Value
        ordered []Value
        progScope *Scope = l.Scope()
        configure = clause.config
        recipes = clause.recipes
    )
    for _, name := range clause.params {
        def := progScope.Lookup(name).(*Def)
        params = append(params, def)
    }
    for _, depend := range clause.depends {
        switch dep := depend.(type) {
        case *List: depends = append(depends, dep.Elems...)
        default:    depends = append(depends, dep)
        }
    }
    for _, depend := range clause.ordered {
        switch dep := depend.(type) {
        case *List: ordered = append(ordered, dep.Elems...)
        default:    ordered = append(ordered, dep)
        }
    }

    var prog = &Program{
        project:  l.project,
        scope:    progScope,
        params:   params,
        depends:  depends,
        ordered:  ordered,
        recipes:  recipes,
        configure: configure,
        position: clause.position,
    }

    for _, target := range clause.targets {
        var (
            entry Entry
            name string
            err error
        )
        if isTrivial(target) {
            erro(ctx, "trivial target").debug(1)
            return
        }

        var args []Value // e.g. for pattern filtering
        switch t := target.(type) {
        case *Group:
            erro(ctx, "group target not supported: %v", t).debug(1)
            return
        case *Argumented:
            target, args = t.value, merge(t.args...)
        }

        name = target.Strval(ctx)

        var patterned = target.patterned(ctx)
        if true && !patterned {
            // NOTE: it should work too if not checking against files
            switch target.(type) {
            case *File, *Path, *Barefile, *PercPattern, *GlobPattern, *RegexpPattern:
            default:
                if file := l.project.FindFile(ctx, name); file != nil {
                    file.position = target.Position()
                    target = file
                }
            }
        }

        entry, err = l.project.entry(ctx, clause.special, clause.options, patterned, target, args, prog)
        if err != nil {
            erro(ctx, "creating entry '%v' failed: %v", target, err).of(target)
            return
        } else /*if entry != nil*/ {
            entries = append(entries, entry)
        }
        if t, okay := entry.Target().(*Flag); okay && t != nil {
            var s = t.name.Strval(ctx)
            if l.project.name != "~" {
                l.Globe().AddFlagEntry(s, entry)
            }
        } else if configure {
            if patterned {
                erro(ctx, "unsupported pattern configures: %v", target).debug(1)
                return
            }
            l.project.configs = append(l.project.configs, entry)
            configuration.entries = append(configuration.entries, entry)
        }
    }
    return
}

type includeFileOpts struct {
    verbose bool `v,verb,verbose`
}
func (l *loader) includeFile(pos Position, opts includeFileOpts, spec Value) {
    var (
        ctx = positional(l, pos)
        linfo = l.loads[len(l.loads)-1]
        specName, fullname string
        err error
    )

    // Execute the rule entry to update include source.
    if false { warn(ctx, "include %T %v", spec, spec).debug(1) }
    if entry, ok := spec.(*RuleEntry); ok && entry != nil {
        var ( result []Value; okay bool )
        if false { warn(ctx, "include %v, %v", entry.programs, entry.programs[0].recipes).debug(1) }
        if result, okay = executeEntry(ctx, entry); !okay {
            erro(ctx, "include entry '%v' failed", entry).debug(1)
            return
        } else if result != nil && opts.verbose {
            info(ctx, "include %v: %v", entry, result).debug(1)
        }
        if false {
            warn(ctx, "include %T %v: %v", entry.target, entry.target, result).
                at(pos).debug(1)
        }
        spec = entry.target
    }

    switch t := spec.(type) {
    case *File:
        if fullname, specName = t.fullname(), t.name; t.info == nil {
            prompt(ctx, "%v: no source file %v, %v\n", fullname, t, t.stat(ctx))
            erro(ctx, "%v: %v", ctx.Project(), t).of(t)
            errostack(ctx, 5, "").debug(16)
            return
        }
    default:
        if specName = spec.Strval(ctx); filepath.IsAbs(specName) {
            fullname = specName
        } else {
            fullname = filepath.Join(linfo.absDir, specName)
        }
    }

    if specName == "" {
        erro(ctx, "`%v` is empty string", spec).debug(1)
        return
    }

    var absDir, baseName = filepath.Split(fullname)
    defer func(mode Mode) { l.mode = mode } (l.mode) // Must restore parse mode!
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))
    if _, err = l.ParseFile(fullname, nil, parseMode|Flat); err != nil {
        prompt(ctx, "%v: %v\n", fullname, specName)
        erro(ctx, "include error occurred (from %v)", fullname)
        errostack(ctx, 5, "%v", ctx).debug(16)
        if true { fail(pos, "parse file failed: %s", fullname) }
    }

    if n := ctx.checkErrors(true); n > 0 {
        warn(ctx, "got %d errors", n).debug(1)
        if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
    }
    return
}
/*
func (l *loader) closureGet(name string) (val Value) {
    if def := l.Scope().FindDef(name); def != nil {
        val = def.Call(l)
    }
    return
}

func (l *loader) closureSet(name string, val Value) (prev Value, okay bool) {
    if def := l.Scope().FindDef(name); def != nil {
        var prev = def.value
        def.val(l, val)
        return prev, true
    }
    return
}
*/
func (l *loader) closureScopes() (scopes []*Scope) {
    scopes = append(l.closureContext.closureScopes(), l.Scope())
    return
}
func (l *loader) openScope(comment string) (scopes []*Scope) {
    if false && options.traceLaunch { defer un(trace(t_launch, "loader.openScope")) }
    var pos Position
    if l.parser != nil { pos = l.Position() } else {
        // TODO: pos.Filename = l.path(), 1
    }

    var scope = NewScope(pos, l.Scope(), l.project, comment)
    scopes = l.scopes
    l.scopes = append([]*Scope{scope}, scopes...)
    return
}

func (l *loader) closeScope(scopes []*Scope) {
    if false && options.traceLaunch { defer un(trace(t_launch, "loader.closeScope")) }
    if scope := l.Scope(); scope == nil {
        // nil scope
    } else if s := scope.comment; strings.HasPrefix(s, "dir ") {
        // Must change the outer of dir scope to globe to avoid Finding symbols
        // into the wrong context.
        l.Globe().SetScopeOuter(scope)
    }
    l.scopes = scopes
}

func (l *loader) setArgs(args []Value) (oldArgs []Value) {
    oldArgs = l.loadArgs
    l.loadArgs = args
    return
}

// project example (base(var=value))
func (l *loader) loadBases(position Position, linfo *loadinfo, implicitBase string, params ...Value) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadBases")) }

    // For &(foobar) set from loadArgs
    //defer setclosure(setclosure(cloctx.unshift(l.scope)))

    var (
        ctx = positional(closureWith(l, position, l.scopes...), position)
        implicitIndex int
        implicitBases []Value
    )
    if file := stat(ctx, "./.base", "", l.project.absPath); file != nil /*&& file.info.IsDir()*/ {
        if true {
            var s = file.Strval(ctx)
            assert(s == file.name && s == "./.base", "invalid strval: %v => %v", file, s)
        }
        if !file.info.IsDir() && (l.project.spec == ".base" /*|| l.project.spec == ".configure"*/) {
            // skip the regular file '.base' to avoid self loading recursively
            // info(ctx, "%v", file).debug(1)
        } else {
            implicitBases = append(implicitBases, file)
        }
    }
    if implicitBase != "" {
        implicitIndex = len(implicitBases)
        implicitBases = append(implicitBases, MakePathStr(position, implicitBase))
    }
    params = append(implicitBases, params...)

    if false { defer func() { warn(ctx, "%v: bases = %v", l.project, params).debug(1) }() }

ParamsLoop:
    for i, elem := range params {
        if false { warn(ctx, "%v %v; %d %v; %s, %v", l.project, l.project.spec, i, elem, implicitBase, params).debug(16) }
        var (
            elemPos = elem.Position()
            absPath string
            args []Value
            isDir bool
            err error
        )
        if list, ok := elem.(*List); ok && len(list.Elems) == 1 { elem = list.Elems[0] }
        if a, ok := elem.(*Argumented); ok { elem, args = a.value, a.args }
        if p, ok := elem.(*Pair); ok {
            var (
                identifier = p.Key
                position = identifier.Position()
                name string
            )
            if name = p.Key.Strval(ctx); len(name) > 0 && name[0] == '.' {
                identifier = MakeBarecomp(position, MakeBareword(position, "project"), p.Key)
            }

            var defs = l.determine(position, token.ASSIGN, identifier, p.Value)
            if len(defs) == 0 {/* TODO: check defs... */}
            continue ParamsLoop
        }

        var (
            specName string
            specVal Value
            implicit bool
        )
        if specVal = elem.expand(ctx, expandPlainValue); specVal == nil {
            specVal = elem // okay!
        } else if specVal.expandible(ctx, expandPlainValue) {
            erro(ctx, "incomplete expand: %v -> %v", elem, specVal).at(elemPos).debug(1)
            return
        } else if defs := specVal.defs(ctx); len(defs) > 0 {
            erro(ctx, "incomplete expand: %v -> %v (defs=%v)", elem, specVal, defs).at(elemPos).debug(1)
            return
        }

        if specName = elem.Strval(ctx); specName == "" {
            erro(ctx, "%v: empty base name `%v` (%T)", l.project, elem, elem).at(elemPos).debug(1)
            break ParamsLoop
        } else if strings.Contains(specName, "//") {
            erro(ctx, "%v: invalid spec: %v in %T", l.project, elem, ctx).at(elemPos)
            erro(ctx, "%v: invalid spec: %v -> %v", l.project, elem, specVal).at(elemPos)
            erro(ctx, "%v: invalid spec: %v -> %v", l.project, elem, specName).at(elemPos).debug(10)
            break ParamsLoop
        } else if implicitBase != "" && specName == implicitBase {
            if i == implicitIndex { implicit = true } else {
                erro(ctx, "%v: base '%v' already loaded implicitly", l.project, elem).at(elemPos).debug(1)
                if false { break ParamsLoop } else { continue }
            }
        }

        if false { info(ctx, "%v: %v -> %v", l.project, elem, specName).at(position).debug(1) }
        if n := ctx.checkErrors(true); n > 0 {
            warn(ctx, "%v: %d errors: %v -> %v", l.project, n, elem, specName).at(position).debug(1)
            break ParamsLoop
        }

        if f, ok := elem.(*File); ok && f.info != nil {
            if absPath = f.fullname(); true { assert(filepath.IsAbs(absPath), "invalid abs path: %v", f) }
            isDir = f.info.IsDir()
        } else if absPath, isDir, err = l.searchSpecPath(linfo, specName); err != nil {
            erro(ctx, "%v: search base failed: %v -> %v", l.project, elem, specName).at(elemPos)
            erro(ctx, "%v: search base failed, %v", l.project, err).at(elemPos).debug(6)
            break ParamsLoop
        } else if absPath == "" {
            erro(ctx, "%v: search base failed: %v -> %v", l.project, elem, specName).at(elemPos).debug(1)
            break ParamsLoop
        }

        for _, base := range l.project.bases {
            if base.absPath == absPath {
                //erro(ctx, "duplicated base: %v (in %v)", elem, l.project.bases).at(elemPos)
                continue ParamsLoop
            }
        }

        var (
            okay bool
            implicitSaved = l.implicit
        )
        if l.implicit = implicit; isDir {
            okay = l.loadDirWithArgs(position, specName, absPath, args, nil)
        } else {
            okay = l.loadWithArgs(position, specName, absPath, args, nil)
        }
        l.implicit = implicitSaved // restore implicit flag

        if !okay {
            var pos Position
            pos.Filename, pos.Line = absPath, 1
            erro(ctx, "%v: '%s' not loaded'", l.project, specName)
            erro(ctx, "%v: base '%s' not loaded, %v", l.project, specName, elem).at(elemPos)
            erro(ctx, "%v: base '%s' not loaded, %s", l.project, specName, absPath).at(position).debug(6)
            break ParamsLoop
        } else if loaded, yes := l.loaded[absPath]; yes && loaded != nil {
            if l.project.hasBase(loaded) {
                if false { warn(ctx, "%v %v; %v", l.project, l.project.bases, loaded).debug(1) }
                continue ParamsLoop
            }
            // chain loaded base project, note that err might not be nil
            l.project.bases = append(l.project.bases, loaded) //l.project.Chain(loaded)
        } else if implicit {
            warn(ctx, "implicit base '%s' not defined (as %s)", specName, absPath).of(elem).debug(1)
        } else {
            erro(ctx, "project `%v`(%T: %s) not loaded (%s)", elem, elem, specName, absPath).at(elemPos).debug(1)
            break ParamsLoop
        }
    }
    if false {
        // bypass ...
    } else if o := l.project.resolveObject(ctx, "using.*"); o == nil {
        // erro(ctx, "resolve using.* failed").debug(1)
    } else if d, ok := o.(*Def); ok && !isTrivial(d.value) {
        var none = MakeNone(position)
        // Derive using.xxx Defs from bases
        for _, val := range merge(d.value) {
            var name, opts = parseUseNameOpts(ctx, val)
            var us = "using." + name
            var def, alt = l.project.scope.define(ctx, DefVoid, us, none)
            if def == nil && !isNil(alt) { def, _ = alt.(*Def) }
            if def == nil { continue }

            var vals []Value
            for _, base := range l.project.bases {
                var obj = base.scope.lookup(us)
                if isNil(obj) { continue }
                if d, ok := obj.(*Def); !ok || isTrivial(d.value) {
                    continue
                } else {
                    vals = append(vals, merge(d.value)...)
                }
                if false { warn(ctx, "%v: %v: %v", l.project, base, def).debug(1) }
            }
            opts.apply(closureWith(ctx, position, l.project.scope), def, vals)
            if false && l.project.name == "extbit.main.macOS" {
                warn(ctx, "%v: %v, %v, %v", def.owner, def.origin, name, opts)
                warnstack(ctx, 5, "%v", def).debug(16)
            }
        }
    }
    return true
}

func (l *loader) loadDotContainer(ident *Barecomp, identStr string, file *File) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadDotContainer")) }
    var position = ident.Position()
    if file.info == nil {
        erro(l, "%s: file not exists: %s", ident, file.fullname()).of(ident).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.loadDir(position, dotContainer, file.fullname(), nil) {
            erro(l, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).of(ident).debug(1)
            return
        }
    } else if !l.loadFile(file.fullname(), nil) {
        erro(l, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).of(ident).debug(1)
        return
    }

    if loaded, yes := l.loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.Scope().Lookup(loaded.Name()).(*ProjectName)
        if name == nil {
            erro(l, "%v: %v: `dock` is not a project", l.project.name, file).of(ident).debug(1)
        } else {
            if false && options.verboseLoads {
                prompt(l, "smart: %v (%v)\n", name, file.fullname())
            }

            var opts useOpts
            // TODO: parse the useOpts
            l.addUsing(positional(l, position), loaded, nil, opts)

            result = true
        }
    }
    return
}

func (l *loader) loadDotConfigure(ident *Barecomp, identStr string, file *File) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadDotConfigure")) }
    var position = ident.Position()
    if file.info == nil {
        erro(l, "%s: file not exists: %s", ident, file.fullname()).at(position).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.loadDir(position, dotConfigure, file.fullname(), nil) {
            erro(l, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).at(position).debug(1)
            return
        }
    } else if !l.loadFile(file.fullname(), nil) {
        erro(l, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).at(position).debug(1)
        return
    }

    if loaded, yes := l.loaded[file.fullname()]; yes && loaded != nil {
        if name, _ := l.Scope().Lookup(loaded.name).(*ProjectName); name == nil {
            if _, alt := l.Scope().ProjectName(positional(l, position), loaded.name, loaded); alt != nil {
                if val, ok := alt.(*ProjectName); !ok || val == nil {
                    erro(l, "name `%s' already taken (%T).", loaded.name, alt).at(position).debug(1)
                }
            }
            if false {
                erro(l, "%v: %v: `.configure` is not a project", l.project.name, file).at(position).debug(1)
            }
        } else {
            if false && options.verboseLoads {
                prompt(l, "smart: %v (%v)\n", name, file.fullname())
            }
            if conf := l.project.configure; conf != nil {
                if conf == loaded { return }
                erro(l, ".configure already specified").at(position).debug(1)
            }
            l.project.configure, result = loaded, true

            var ctx = positional(l, position)
            var opts = useOpts{}
            if false {
                applyUseeVars(ctx, l.project, loaded)  // aka.       ABC += $(using.ABC)
                applyUsingVars(ctx, l.project, loaded) // aka. using.ABC += $(using.ABC)
            } else if false {
                for _, usee := range loaded.usees(true, false, false, false) {
                    applyUseeVars(ctx, l.project, usee)  // aka.       ABC += $(using.ABC)
                    applyUsingVars(ctx, l.project, usee) // aka. using.ABC += $(using.ABC)
                    //warn(l, "%v %v %v", l.project, loaded, usee).debug(1)
                }
            } else if false {
                if err := l.addUsing(ctx, loaded, nil, opts); err != nil {
                    erro(ctx, "using '%v' failed: %v", loaded, err).debug(1)
                }
                //l.importFileMaps(ctx, public, specVal)
            } else if true {
                for _, usee := range loaded.usees(true, false, false, false) {
                    if false { warn(ctx, "%v: %v: %v", l.project, loaded, usee).at(usee.position).debug(1) }
                    if err := l.addUsing(ctx, usee, nil, opts); err != nil {
                        erro(ctx, "using '%v' failed: %v", usee, err).debug(1)
                        break
                    }
                    l.importFileMaps1(ctx, opts, usee)
                }
            }
        }
    }
    return
}

func (l *loader) declare(keyword token.Token, ident *Barecomp, identStr string, declOpts []Value) (result bool) {
    var (
        pos = ident.Position()
        ctx = positional(l, pos)
    )
    if identStr == "@" {
        var (
            linfo = l.loads[0]
            dec, ok = linfo.declares[identStr]
            at, _ = l.Globe().scope.Lookup(identStr).(*ProjectName)
        )
        if !ok {
            dec = &declare{ project: at.NamedProject() }
            linfo.declares[identStr] = dec
        }
        dec.backscope = l.Scope()
        l.useesExecuted = nil
        l.project = at.NamedProject()
        //l.scope = l.scope
        l.scopes[0] = at.NamedProject().scope
        return true
    } else if _, o := l.Scope().Find(identStr); o != nil {
        if _, ok := o.(*Builtin); ok {
            erro(ctx, "project name '%s' is a builtin name", identStr)
            return
        }
    }

    var (
        name = identStr
        linfo = l.loads[len(l.loads)-1]
        dec, declared = linfo.declares[name]
    )
    if !declared {
        var (
            wd = l.WorkDir()
            outer = l.Scope()
            absDir = linfo.absDir
            relPath, tmpPath string
        )
        if !filepath.IsAbs(absDir) {
            //absDir = filepath.Join(l.WorkDir(), absDir)
            absDir, _ = filepath.Abs(absDir)
        }
        relPath, _ = filepath.Rel(wd, absDir)
        tmpPath = joinTmpPath(ctx, wd, relPath)

        // Avoid nesting project scopes!
        for strings.HasPrefix(outer.Comment(), "project \"") {
            outer = outer.outer
        }

        dec = &declare{ project: l.Globe().project(ctx, outer, absDir, relPath, tmpPath, linfo.specName, name) }
        l.loaded[linfo.absPath()] = dec.project
        linfo.declares[name] = dec
    }
    if loader := linfo.loader; loader != nil {
        if _, a := loader.scope.ProjectName(ctx, name, dec.project); a != nil {
            if v, ok := a.(*ProjectName); !ok || v == nil {
                erro(ctx, "`%s` name already taken (%T).", name, a).of(a).debug(1)
                return
            }
        }
    }

    declOpts = parseOpts(ctx, &dec.project.opts, declOpts...)
    dec.backscope = l.Scope()
    l.useesExecuted = nil
    l.project = dec.project
    l.scopes[0] = dec.project.scope
    if l.Globe().main != nil && l.Globe().main == l.project && l.project.name != "~" {
        for _, t := range l.Globe().pairs {
            switch k := t.Key.(type) {
            case *Bareword, *Barecomp:
                var name = k.Strval(ctx);
                //if name[0] == '.' { name = "project" + name }
                var def, alt = l.def(l.Position(), name)
                if def == nil && alt != nil { def = alt.(*Def) }
                def.set(ctx, DefDecl, t.Value)
            default:
                erro(ctx, "`%v` unknown target from command line (%v)", t, l.project)
                return
            }
        }
    }

    for _, arg := range merge(l.loadArgs...) {
        switch t := arg.(type) {
        case *Pair:
            var name = t.Key.Strval(ctx)
            var def, alt = l.def(t.Key.Position(), name)
            if alt != nil {
                var ok bool
                if def, ok = alt.(*Def); !ok {
                    erro(ctx, "'%v' is not a Def (%T)", alt, alt).debug(1)
                    return
                }
            }
            if def != nil { def.val(ctx, t.Value) }
            warn(ctx, "%v: %v", ident, t)
        }
    }

    if err := l.loadPlugin(pos); err != nil {
        erro(ctx, "load plugin failed: %v", err).debug(1)
        return
    }
    return true
}

func (l *loader) loadProjectConfiguration(ident *Barecomp, identStr string, declared bool) (result bool) {
    // FIXES: set cloctx immediately to ensure the right configuration is matched!
    //defer setclosure(setclosure(cloctx.unshift(l.scope)))
    if false { defer un(tracef(t_traverse, "loadProjectConfiguration(%v)", ident)) }

    var (
        pos = ident.Position()
        ctx = positional(l, pos)
    )
    // Get configuration file name for the project and include it in flat mode.
    if file := l.project.configuration(ctx); file == nil {
        erro(ctx, "%v: nil configuration file", ident).debug(1)
        return
    } else if declared || options.configure {
        var ( s string = file.fullname(); exists bool )
        for _, v := range configuration.clean { if s == v { exists = true; break }}
        if !exists { configuration.clean = append(configuration.clean, s) }
    } else if file.exists() {
        if false && (options.verboseImport || options.verboseLoads) {
            var cp Position; cp.Filename, cp.Line = file.fullname(), 1
            info(ctx, "%s (%s)", l.project, l.project.spec).at(cp).debug(true, 1)
        } else if options.verbose {
            prompt(ctx, "Configuration for %s (%s)\n", l.project, l.project.spec).debug(1)
        }
        var opts includeFileOpts
        l.isIncludingConf = true
        l.includeFile(pos, opts, file)
        l.isIncludingConf = false
    }

    if l.project.name != dotConfigure {
        // Looking for project specific .configure module
        if file := stat(ctx, dotConfigure, "", l.project.absPath); file.exists() {
            if false { warn(ctx, "%v %v; %v", l.project.name, l.project.spec, file).debug(1) }
            if identStr == dotConfigure {
                erro(ctx, "provided .configure for a .configure project").debug(1)
            } else if !l.loadDotConfigure(ident, identStr, file) {
                //erro(ctx, "declare %s: %s/.configure", name, l.project.absPath)
            }
        }
    }
    return true
}

func (l *loader) loadProjectContainer(ident *Barecomp, identStr string) (result bool) {
    // FIXES: set cloctx immediately to ensure the right configuration is matched!
    //defer setclosure(setclosure(cloctx.unshift(l.scope)))

    var ( pos = ident.Position(); ctx = positional(l, pos) )
    if l.project.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").debug(1)
            return
        }

        // Looking for project specific .container module
        if file := stat(ctx, dotContainer, "", l.project.absPath); file.exists() {
            if !l.loadDotContainer(ident, identStr, file) {
                //erro(ctx, "declare %s: %s/.container", name, l.project.absPath)
            }
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(ctx, l.project.absPath, func(s string) bool {
            var file = stat(ctx, dotContainer, "", filepath.Join(s, ".smart"))
            if !file.exists() {
                // no docking enabled
            } else if !l.loadDotContainer(ident, identStr, file) {
                //erro(ctx, "%v", err)
            }
            return false
        })

        result = true
    }
    return
}

func (l *loader) closeCurrent(ident *Barecomp, identStr string) (err error) {
    if identStr == "@" {
        if dec, ok := l.loads[0].declares[identStr]; ok {
            l.scopes[0] = dec.backscope
            l.useesExecuted = dec.useesExecuted
            dec.backscope = nil
            dec.useesExecuted = nil
        }
        return nil
    }

    var linfo = l.loads[len(l.loads)-1]
    var dec, ok = linfo.declares[identStr]
    if dec == nil || !ok {
        return fmt.Errorf("no loaded project %s", identStr)
    }
    if l.project == nil {
        return fmt.Errorf("no current project")
    } else if s := l.project.Name(); s != identStr {
        return fmt.Errorf("current project is %s but %s", s, identStr)
    } else if l.project != dec.project {
        return fmt.Errorf("project conflicts (%s, %s)", l.project.Name(), dec.project.Name())
    }

    l.scopes[0] = dec.backscope
    l.useesExecuted = dec.useesExecuted
    return
}

func (l *loader) OpenNamedScope(name, comment string) (scopes []*Scope) {
    if outer := l.Scope(); outer == nil {
        erro(l, "no parent scope '%s' (%v)", name, comment).at(l.Position()).debug(8)
    } else {
        if strings.HasPrefix(outer.Comment(), "dir ") {
            outer = outer.outer // discard dir scope
        }

        scopes = l.openScope(comment)
        if scope := l.Scope(); scope != nil {
            outer.ScopeName(positional(l, scope.position), name, scope)
        } else {
            erro(l, "open scope '%s' failed (%v)", name, comment).at(l.Position()).debug(8)
        }
    }
    return
}

func (l *loader) resolveObject(value Value) (name string, result Value, err error) {
    var pos = value.Position()
    if !pos.IsValid() { pos = l.Position() }
    if _, ok := value.(*selection); ok {
        panic(failure{pos,"resolving a selection"})
    }

    var ctx = positional(l, pos)
    if name = value.Strval(ctx); name == "" {
        erro(ctx, "name '%v' is empty", name).debug(1)
    } else if l.Scope() == nil {
        erro(ctx, "nil scope to resolve '%v'", name).debug(1)
    } else if _, result = l.Scope().Find(name); !isNil(result) {
        // okay!
    } else if project := l.project; project == nil {
        erro(ctx, "nil project to resolve '%v'", name).debug(1)
    } else if result = project.resolveObject(ctx, name); isNil(result) {
        if false { erro(ctx, "%v: resolved object '%v' is nil", project, name).debug(1) }
        //erro(ctx, "%v: resolved object '%v' is nil", project, name).debug(1)
    }
    return
}

func (l *loader) resolveEntries(target Value) (entries *ResolveEntries) {
    var (
        pos = l.Position()
        ctx = positional(l, pos)
        name = target.Strval(ctx)
    )
    entries = l.project.resolveEntries(ctx, name, false, false)
    return
}

func (l *loader) def(position Position, name string) (def *Def, alt Object) {
    var scope = l.Scope()
    if  strings.HasPrefix(scope.comment, "file ") && l.mode&Flat != 0 {
        // use project scope if defining in flat file (aka. include)
        // to ensure that the symbol is valid in the project
        scope = l.Scope()
    }
    def, alt = scope.define(positional(l, position), DefVoid, name, MakeNone(position))
    if def != nil { def.position = position }
    return
}

func (l *loader) assign(tok token.Token, def *Def, alt Object, value Value) {
    var ( pos = l.Position(); ctx = positional(l, pos) )
    switch tok {
    case token.ASSIGN:     //   =
        def.set(ctx, DefDefault, value)
    case token.SCO_ASSIGN: //  :=
        def.set(ctx, DefExpand1, value)
    case token.DCO_ASSIGN: // ::=
        def.set(ctx, DefExpand2, value)
    case token.EXC_ASSIGN: //  !=
        def.set(ctx, DefExecute, value)
    case token.QUE_ASSIGN: //  ?=
        if isNil(alt) { def.set(ctx, DefDefault, value) }
    case token.ADD_ASSIGN: //  +=
        if isTrivial(value) {
            // NOOP
        } else if isTrivial(def.value) || !def.value.refs(ctx, value) {
            def.append(ctx, value)
        } else {
            erro(ctx, "can't append value '%v' to: %v", value, def).debug(1)
        }
    case token.SHI_ASSIGN: //  =+
        if !def.value.refs(ctx, value) {
            var tail = def.value
            def.val(ctx, value)
            def.append(ctx, merge(tail)...)
            warn(ctx, "%v; %v; %v", value, tail, def).debug(1)
        }
    case token.SUB_ASSIGN: // -=
        if !isTrivial(def.value) {
            var (
                vals []Value
                sub = merge(value)
            )
            for _, val := range merge(def.value) {
                var b bool
                for _, v := range sub {
                    if b = val.cmp(ctx, v) == cmpEqual; b { break }
                }
                if !b { vals = append(vals, val) }
            }
            def.value = MakeList(def.position, vals...)
        }
    case token.SAD_ASSIGN: // -+=
        var vals []Value
        if !isTrivial(def.value) {
            var sub = merge(value)
            for _, val := range merge(def.value) {
                var b bool
                for _, v := range sub {
                    if b = val.cmp(ctx, v) == cmpEqual; b { break }
                }
                if !b { vals = append(vals, val) }
            }
            vals = append(vals, sub...)
        }
        def.value = MakeList(def.position, vals...)
    case token.SSH_ASSIGN: // -=+
        var vals []Value
        if !isTrivial(def.value) {
            var sub = merge(value)
            for _, val := range merge(def.value) {
                var b bool
                for _, v := range sub {
                    if b = val.cmp(ctx, v) == cmpEqual; b { break }
                }
                if !b { vals = append(vals, val) }
            }
            vals = append(sub, vals...)
        }
        def.value = MakeList(def.position, vals...)
    }
    return
}

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
//
func readSource(filename string, src interface{}) ([]byte, error) {
    if src != nil {
        switch s := src.(type) {
        case string:
            return []byte(s), nil
        case []byte:
            return s, nil
        case *bytes.Buffer:
            // is io.Reader, but src is already available in []byte form
            if s != nil {
                return s.Bytes(), nil
            }
        case io.Reader:
            var buf bytes.Buffer
            if _, err := io.Copy(&buf, s); err != nil {
                return nil, err
            }
            return buf.Bytes(), nil
        }
        return nil, fmt.Errorf("invalid source")
    }
    return ioutil.ReadFile(filename)
}

// ParseFile parses the source code of a single source file and returns
// the corresponding ast.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
func (l *loader) ParseFile(filename string, src interface{}, mode Mode) (f *parsedFile, err error) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.ParseFile")) }

    var text []byte
    if text, err = readSource(filename, src); err != nil {
        prompt(l, "%v: %v\n", filename, err)
        erro(l, "reading source failed: %v", err)
        errostack(l, 5, "%v", l).debug(32)
        return
    }

    defer func(saved *parser, m Mode) {
        var pos Position
        if l.parser != nil && l.parser.file != nil {
            pos = l.Position()
        } else {
            pos.Filename = filename
        }
        var panics, _ = checkPanicsErrors(positional(l, pos), true)
        if proj := l.project; panics > 0 {
            if err != nil { erro(l, "panics with error: %v", panics, err) }
            if proj.position.Equals(&pos) {
                erro(l, "failed: got %d panics from %v (%s)", panics, proj, proj.spec).debug(128)
            } else {
                erro(l, "failed: got %d panics from %v (%s)", panics, proj, proj.spec)
                erro(l, "%s: got %d panics (%s)", proj, panics, proj.spec).at(proj.position).debug(128)
            }
        } else if err != nil {
            erro(l, "parse file failed: %v", err).debug(128)
        }
        l.parser.loader, l.parser, l.mode = nil, saved, m
    } (l.parser, l.mode)

    // set the current parser
    l.parser = new(parser)
    l.parser.init(l, filename, text)
    l.mode = mode

    // set result values
    if f = l.parser.parseFile(); f == nil {
        // Source is not a valid source file, returnning a valid but empty parsedFile
        defer l.closeScope(l.openScope(fmt.Sprintf("file %s", filename)))
        f = &parsedFile{ scope:l.Scope() }
        f.position.Filename = filename
        // TODO: validate basename as a valid identifier
        f.name = MakeBarecomp(f.position, MakeBareword(f.position, filepath.Base(filepath.Dir(filename))))

    }
    return
}

// ParseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (l *loader) ParseConfigDir(pathname, linked string) (err error) {
    var fd *os.File // Directory of the destination.
    if fd, err = os.Open(linked); err != nil { return }
    defer fd.Close()

    var list []os.FileInfo
    if list, err = fd.Readdir(-1); err != nil || len(list) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        err = fmt.Errorf("invalid package name %s", ident)
        return
    }

    defer l.closeScope(l.OpenNamedScope(ident, fmt.Sprintf("config %s", pathname)))

    var ctx = positional(l, l.Position())
    var def *Def
ListLoop:
    for _, d := range list {
        var name = d.Name()
        if strings.HasPrefix(name, "~") ||
            strings.HasSuffix(name, ".#") ||
            strings.HasSuffix(name, ".smart") ||
            strings.HasSuffix(name, ".sm") { continue ListLoop }

        var fullname = filepath.Join(linked, name)
        if d.Mode()&os.ModeSymlink != 0 {
            var ( l string; t os.FileInfo )
            if l, err = os.Readlink(fullname); err != nil { continue ListLoop }
            if !filepath.IsAbs(l) { l = filepath.Join(linked, l) }
            if t, err = os.Stat(l); err != nil { continue ListLoop }
            if t.IsDir() { continue ListLoop }
        }

        var pos = Position(l.parser.file.Position(l.pos))
        var ctx = positional(ctx, pos)
        if d.IsDir() {
            if err = l.ParseConfigDir(filepath.Join(pathname, name), fullname); err != nil {
                erro(ctx, "parse config failed: %v", err).debug(1)
                break ListLoop
            }
            if ctx.checkErrors(true) > 0 { return }
        } else if s, a := l.def(l.Position(), name); a != nil {
            erro(ctx, "declare project: %v", err).debug(1)
            break ListLoop
        } else if def = s; def != nil {
            var ( v []byte; s string )
            if v, err = ioutil.ReadFile(fullname); err != nil { break ListLoop }
            if s = string(v); !utf8.ValidString(s) {
                erro(ctx, "%s: invalid UTF8 content", fullname)
                break ListLoop
            }
            def.set(ctx, DefConfDir, MakeString(pos, s))
        } else if s != nil {
            erro(ctx, "Name `%s' already taken, not def (%T).", name, s)
            break ListLoop
        }
    }
    return
}

// ParseDir calls ParseFile for all files with names ending in ".go" in the
// directory specified by path and returns a map of package name -> package
// AST with all the packages found.
//
// If filter != nil, only the files with os.FileInfo entries passing through
// the filter (and ending in ".go") are considered. The mode bits are passed
// to ParseFile unchanged. Position information is recorded in fset.
//
// If the directory couldn't be read, a nil map and the respective error are
// returned. If a parse error occurred, a non-nil but incomplete map and the
// first error encountered are returned.
//
func (l *loader) ParseDir(pos Position, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*Project) {
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseParse /*&& d > 50*time.Millisecond*/ {
            fmt.Fprintf(stderr, "parse(%15s) %s ⇒ %s\n", d, l.project, path)
        }
    } (time.Now())

    var ctx = positional(l, pos)
    var fd, err = os.Open(path)
    if err != nil {
        erro(ctx, "open(%s): %v", path, err).debug(1)
        return
    }
    defer fd.Close()

    list, err := fd.Readdir(-1)
    if err != nil {
        erro(ctx, "readdir: %v", err).debug(1)
        return
    } else if len(list) == 0 {
        erro(ctx, "no files underneath: %s", path).debug(1)
        return
    }
    for i, a := range list {
        if i > 0 && a.Name() == "build.smart" {
            first := list[0]
            list[0] = a
            list[i] = first
        }
    }

    defer l.closeScope(l.openScope(fmt.Sprintf("dir %s", path)))
    if l.Scope().position.Filename == "" {
       l.Scope().position.Filename = path
       l.Scope().position.Line = 1
    }

    // FIXES: use 'globe' scope as outer to avoid chaining scopes to other unrelated
    // projects which are in consequence load order. Setting dir scope outer to such
    // project scopes will cause resolving objects to the wrong ones.
    l.Scope().outer = ctx.Globe().scope

    mods = make(map[string]*Project)
ListLoop:
    for _, d := range list {
        var (
            name, mo = d.Name(), d.Mode()
            filename = filepath.Join(path, name)
            linked, linkPath = "", path
            skip = (name == "")
        )
        if!skip { skip =   strings.HasPrefix(name, ".#") }
        if!skip { skip = !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm")) }
        if skip { continue ListLoop }
        for fn := filename; mo&os.ModeSymlink != 0; {
            if s, err := os.Readlink(fn); err != nil {
                prompt(ctx, "%s: readlink failed\n", fn)
                warn(ctx, "%v", err)
                warnstack(ctx, 6, "%v", ctx).debug(6)
                continue ListLoop
            } else {
                var linkName = s
                var rel = !filepath.IsAbs(linkName)
                if rel { linkName = filepath.Join(linkPath, linkName) }
                if fi, err := os.Lstat(linkName); err != nil {
                    prompt(ctx, "%s: lstat %s\n", fn, s)
                    warn(ctx, "%v", err)
                    warnstack(ctx, 6, "%v", ctx).debug(6)
                    continue ListLoop
                } else {
                    if rel { linkPath = filepath.Dir(s) }
                    mo, fn = fi.Mode(), s
                    linked = fn
                }
            }
        }
        /*if (name == "configure.smart" || name == "configure.sm") && (linked != "" || mo.IsDir()) {
            //hasConfDir = true // TODO: remove ConfigDir feature
            if err := l.ParseConfigDir(filepath.Dir(filename), linked); err != nil {
                if first == nil {
                    first = err
                }
                return
            }
            continue ListLoop
        }*/
        if linked != "" { }

        if mo.IsRegular() && (filter == nil || filter(d)) {
            var pos Position
            pos.Filename, pos.Line = filename, 1

            var d *diagPoint
            var src, err = l.ParseFile(filename, nil, mode|parsingDir)
            if n := ctx.checkErrors(true); n > 0 {
                if s, n := filepath.Base(filename), n; err == nil {
                    d = erro(ctx, "%d diagnostic errors parsing file '%s'", n, s)
                } else {
                    d = erro(ctx, "%d diagnostic errors parsing file '%s'", n, s)
                    d = erro(ctx, "parsing failed: %v", err)
                }
                warn(ctx, "parse dir '%s' got %d errors", filepath.Base(path), ctx.totalErrors()).debug(10)
                if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
            } else if err != nil {
                d = erro(ctx, "parse file failed: %v", err)
            } else if src == nil {
                d = erro(ctx, "parsed nil module from")
            } else if isNil(src.name) {
                d = erro(ctx, "parsed module name is <nil>")
            } else if isNone(src.name) {
                d = erro(ctx, "parsed module name is <none>")
            }
            if d != nil {
                if l.parser != nil && l.parser.file != nil {
                    var s = "… parser stopped here here"
                    if d.dt == diagWarn {
                        d = warn(ctx, s)
                    } else {
                        d = erro(ctx, s)
                    }
                }
                d.debug(16)
                return
            }

            var name = src.name.Strval(ctx)
            if mod, found := mods[name]; !found {
                mod = &Project{ name: name, scope: l.Scope() }
                mods[name] = mod
            }
            //mod.Files[filename] = src
        }
    }
    return
}

// loader.Load loads script from a file or source code (string, []byte).
func (l *loader) load(position Position, specName, absPath string, source interface{}) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.load")) }
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseLoads /*&& d > 50*time.Millisecond*/ {
            loaded, _ := l.loaded[absPath]
            if l.project == nil {
                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
            }
        }
    } (time.Now())

    if false && options.verboseLoads { fmt.Fprintf(stderr, "load %s …\n", specName) }

    if absPath == "" {
        erro(l, "no such module `%s' (in paths %v)", specName, l.paths)
        return
    } else if !filepath.IsAbs(absPath) {
        erro(l, "invalid abs name `%s' (%s)", absPath, specName)
        return
    }

    // Check loaded project.
    if loaded, yes := l.loaded[absPath]; yes {
        if _, a := l.Scope().ProjectName(positional(l, l.Position()), loaded.Name(), loaded); a != nil {
            if val, ok := a.(*ProjectName); !ok || val == nil {
                erro(l, "`%v` name already taken (%T).", loaded, a)
            }
        }
        result = true
        return
    }

    var absDir, baseName = filepath.Split(absPath)
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

    var doc, err = l.ParseFile(absPath, source, parseMode)
    if n := l.checkErrors(true); n > 0 {
        warn(l, "load '%s' got %d errors", specName, n).debug(1)
        if options.failOnErrors { fail(l.Position(), "fail by %d errors", l.totalErrors()) }
        return
    } else if err != nil {
        erro(l, "load: %v", err).debug(1)
    } else if doc == nil {
        erro(l, "load: doc is nil (%s)", absPath).debug(1)
    } else {
        result = true
    }
    return
}

func (l *loader) loadDir(pos Position, specName, absDir string, filter func(os.FileInfo) bool) (loadedOkay bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadDir")) }
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseLoads /*&& d > 50*time.Millisecond*/ {
            loaded, _ := l.loaded[absDir]
            if l.project == nil {
                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
            }
        }
    } (time.Now())

    if false && options.verboseLoads { fmt.Fprintf(stderr, "load %s …\n", specName) }

    if !pos.IsValid() {
        pos = positionForDir(absDir)
    }
    if !filepath.IsAbs(absDir) {
        erro(l, "needs absolute dir `%s' (%s)", absDir, specName).debug(1)
        return
    }

    var loaded *Project
    defer func() {
        if loaded == nil {
            erro(l, "%v (%v) not loaded in %v", specName, filepath.Base(absDir), l.Scope())
            errostack(l, 16, "%v", specName).debug(512)
            return
        }
        if proj := l.Scope().project; proj == nil {
            if false { erro(l, "%v: no owner project for %s", loaded.name, l.Scope()).debug(1) }
        } else if name, _ := proj.scope.Lookup(loaded.name).(*ProjectName); name == nil {
            if _, alt := proj.scope.ProjectName(positional(l, pos), loaded.name, loaded); alt != nil {
                if val, ok := alt.(*ProjectName); !ok || val == nil {
                    erro(l, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                }
            }
        }
    } ()

    // Check loaded project.
    if loaded, loadedOkay = l.loaded[absDir]; loadedOkay {
        /*if _, a := l.Scope().ProjectName(positional(l, pos), loaded.name, loaded); a != nil {
            if val, ok := a.(*ProjectName); !ok || val == nil {
                erro(l, "name `%s' already taken (%T).", loaded.name, a).debug(1)
            }
        }*/
        return
    }

    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

    var mods = l.ParseDir(pos, absDir, filter, parseMode)
    if n := l.checkErrors(true); n > 0 {
        erro(l, "%d diagnostic errors parsing module '%s'", n, specName).debug(12)
        if options.failOnErrors { fail(l.Position(), "fail by %d errors", l.totalErrors()) }
        return
    }

    // FIXME: loading failed if different 'project' found in
    // the same dir, for example:
    //      project Foo # file build.smart
    //      project # file config.smart
    if len(mods) == 0 && filepath.Base(specName) != "@" {
        if l.implicit {
            warn(l, "%s not loaded (as %s, implicitly)", specName, absDir).debug(8)
            loadedOkay = true // okay for implicit loading
        } else {
            erro(l, "%s not loaded (as %s)", specName, absDir).debug(8)
        }
    } else if loaded, loadedOkay = l.loaded[absDir]; loadedOkay && loaded != nil {
        // Good!
    } else if filepath.Base(specName) != "@" {
        erro(l, "%s not loaded (as %s, implicit=%v)", specName, absDir, l.implicit).debug(1)
    }
    return
}

func (l *loader) loadWithArgs(position Position, specName, absPath string, args []Value, source interface{}) bool {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadWithArgs")) }
    defer l.setArgs(l.setArgs(args))
    return l.load(position, specName, absPath, source)
}

func (l *loader) loadDirWithArgs(position Position, specName, absPath string, args []Value, filter func(os.FileInfo) bool) bool {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadDirWithArgs")) }
    defer l.setArgs(l.setArgs(args))
    if false { info(l, "%v, %v, %v, %v", l.project, specName, absPath, args).at(position).debug(1) }
    return l.loadDir(position, specName, absPath, filter)
}

func (l *loader) loadFile(filename string, source interface{}) bool {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadFile")) }
    var spec string
    switch dir, base := filepath.Split(filename); base {
    case ".base", ".configure": spec = base
    default: spec, _  = filepath.Rel(l.WorkDir(), dir)
    }
    var position Position
    position.Filename = filename
    return l.load(position, spec, filename, source)
}

func (l *loader) loadPath(path string, filter func(os.FileInfo) bool) bool {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadPath")) }
    var (
        position Position
        s, _ = filepath.Rel(l.WorkDir(), path)
    )
    position.Filename = s
    return l.loadDir(position, s, path, filter)
}

func (l *loader) loadText(filename string, text string) []Value {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadText")) }

    defer func(saved *parser) {
        l.parser.loader = nil
        l.parser = saved
    } (l.parser)

    if g := l.Globe(); g.main == nil {
        l.scopes[0] = g.os.scope
    } else {
        l.scopes[0] = g.main.scope
    }
    l.useesExecuted = nil
    l.parser = new(parser)
    l.parser.init(l, filename, []byte(text))
    return l.parser.parseText()
}

func AddSearchPaths(paths... string) (err error) {
    for _, s := range paths {
        if s, err = filepath.Abs(s); err != nil {
            break
        }
        if fi, _ := os.Stat(s); fi != nil && fi.IsDir() {
            globalPaths = append(globalPaths, s)
        }
    }
    return
}
