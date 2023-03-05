//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "extbit.io/smart/token"
	"extbit.io/smart/scanner"
    "bytes"
    "io/ioutil"
    "io"
    "unicode/utf8"
    "path/filepath"
    "strings"
    "plugin"
    "time"
    "fmt"
    "os/exec"
    "os"
    "io/fs"
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

    parsingText
    parsingDir
)

const (
    dotBase = ".base"
    dotContainer = ".container"
    dotConfigure = ".configure"
)

var parseMode = DeclarationErrors //|Trace

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

func (li *loadinfo) traveUseLoop() (result bool) {
    var first bool = true
    for _, decl := range li.declares {
        if first || result {
            result = decl.project.opts.traveUseLoop
        }
        if !result { break }
        first = false
    }
    return
}

type loader struct {
    closureContext
    p *parser
    mode Mode // parsing mode
    project  *Project
    loadArgs []Value
    loadStack []*Project // load path
    useStack  []*Project // use path
    useesExecuted []*Project // all executed usees
    implicit bool // loading current project implicitly, aka. via foo.bar.Baz (implicit foo/bar loaded)
    vs string // verbose prefix
}
func (l *loader) loader() *loader { return l }
func (l *loader) parser() *parser { return l.p }
func (l *loader) inner() Context { return &l.closureContext }
func (l *loader) String() string { return fmt.Sprintf("loader{%s}", &l.closureContext) }
func (l *loader) Project() (project *Project) { return l.project }

func restoreLoadingInfo(l *loader) {
    var (
        last = len(universe.globe.loads)-1
        linfo = universe.globe.loads[last]
    )

    universe.globe.loads = universe.globe.loads[0:last]
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
    universe.globe.loads = append(universe.globe.loads, &loadinfo{
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
func (opts *useVarOpts) apply(ctx Context, def *def, vals []Value) {
    if opts.unique {
        if def.append(ctx, vals...); len(opts.args) > 0 {
            var position = ctx.Position()
            var args = MakeList(position, opts.args...)
            def.value = builtinUnique(ctx, plain, args, def.value)
        } else {
            def.value = builtinUnique(ctx, plain, def.value)
        }
    }
}
func parseUseNameOpts(ctx Context, nameVal Value) (name string, opts useVarOpts) {
    if arged, ok := nameVal.(*Argumented); ok {
        var args []Value
		nameVal, args = arged.value, arged.args
        opts.args = parseOpts(ctx, &opts, plain, args...)
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
		d, alt = user.scope.define(ctx, DefVoid, name, none)
        isNewDef = !isNil(d) && isNil(alt)
	)
	if d == nil && !isNil(alt) { d, _ = alt.(*def) }
	if d == nil {
		erro(of(ctx,nameVal), `%v: "%s" is undefined`, user, name).debug(1)
		return
    } else if false {
        isNewDef = /*isTrivial(def.value)*/true
    }
    if isNewDef && len(user.bases) > 0 {
        // 1: derive values from bases
        for _, base := range user.bases {
            if obj := base.resolveObject(ctx, name); isNil(obj) {
                continue
            } else if t, y := obj.(*def); y && !isTrivial(t.value) {
                opts.apply(ctx, d, merge(t.value))
            }
        }
        // 2: apply using vars from bases
        for _, base := range user.bases {
            var obj Object
            if false {
                obj = base.scope.lookup("use."+name)
            } else if obj = base.resolveObject(ctx, "use."+name); isNil(obj) {
                continue
            } else if t, y := obj.(*def); y && !isTrivial(t.value) {
                opts.apply(ctx, d, merge(t.value))
            }
        }
    }

    if v, e := user.use.Get(ctx, name/* NOTE: gets `use.%s` */); e != nil {
		erro(ctx, "%v: %v (use.%s)", user, e, name)
    } else if va := merge(v); len(va) > 0 {
        var c = closureWith(ctx, user.scope)
        for _, v := range va {
            if t, y := v.(*def); !y {
                erro(ctx, "%s: not a Def: %T %v", name, v, v).debug(1)
            } else if isTrivial(t.value) {
                // does nothing
            } else {
                opts.apply(c, d, merge(t.value))
            }
        }
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
        useName = "use." + name
        d, alt = user.scope.define(ctx, DefVoid, useName, none)
    )
    if d == nil && !isNil(alt) { d, _ = alt.(*def) }
    if d == nil {
        erro(of(ctx,nameVal), `%v: "%s" is undefined'`, user, name).debug(1)
        return
    } else if isTrivial(d.value) {
        for _, base := range user.bases {
            if obj := base.resolveObject(ctx, useName); isNil(obj) {
                continue
            } else if t, y := obj.(*def); y && !isTrivial(t.value) {
                d.append(ctx, t.value)
            }
        }
    }

    if o := usee.scope.Lookup(useName); isNil(o) || isNone(o) {
        // does nothing
    } else if t, y := o.(*def); !y {
        erro(ctx, "%s: not a Def: %T %v", name, o, o).debug(1)
    } else if isTrivial(t.value) {
        // does nothing
    } else {
        var c = closureWith(ctx, usee.scope)
        opts.apply(c, d, merge(t.value))
    }
}
func applyUseeVars(ctx Context, user, usee *Project) {
    var spec Value
    if o := usee.resolveObject(ctx, "use.*"); o == nil {
        // erro(ctx, "resolve use.* failed").debug(1)
    } else if def, _ := o.(*def); def != nil {
        spec = def.value
    }
    if !isTrivial(spec) {
        // NOTE: apply vars like 'cflags', 'cxxflags', ...
        for _, name := range merge(spec) { applyUseeVar(ctx, user, usee, name) }
    }
}
func applyUserVars(ctx Context, user, usee *Project) {
    var spec Value
    if o := user.resolveObject(ctx, "use.*"); o == nil {
        // erro(at(ctx,ctx.Position()), "resolve use.* failed").debug(1)
    } else if def, _ := o.(*def); def != nil {
        spec = def.value
    }
    if !isTrivial(spec) {
        // NOTE: apply vars like 'use.cflags', 'use.cxxflags', ...
        for _, value := range merge(spec) { applyUsingVar(ctx, user, usee, value) }
    }
}

func (l *loader) Position() (res Position) {
    if l.p != nil {
        res = l.p.Position()
    } else {
        res = l.Context.Position()
    }
    return
}

func (l *loader) loadUseSpecName(ctx Context, opts useOpts, specVal Value, arged []Value, params ...Value) (loaded *Project) {
    var (
        linfo = universe.globe.loads[len(universe.globe.loads)-1]
        absPath, specName string
        isDir, traveUseLoop bool
        err error
    )
    if n, y := specVal.(*ProjectName); y {
        if false { warnstack(ctx, 3, "use project: %v %s", n, n.spec).debug(6) }
        loaded = n.Project
    } else if specName = specVal.Strval(ctx); specName == "" {
        errostack(ctx, 3, "empty spec: %v (%T)", specVal, specVal).debug(6)
        return
    } else if absPath, isDir, err = universe.search(linfo, specName); err != nil {
        errostack(ctx, 3, "no such package `%v` (%T)", specName, specVal).debug(6)
        return
    } else if absPath == "" {
        errostack(ctx, 3, "missing `%s` (in %v)", specName, universe.paths).debug(6)
        return
    } else {
        if loaded, y = universe.globe.loaded[absPath]; !y {
            if false { warnstack(ctx, 3, "not project: %s", absPath).debug(6) }
        }
        // Checking circular loads. See also Project.loopImportPath()!
        for i, load := range universe.globe.loads {
            if load.absDir == absPath {
                var s string
                var loop, loopTravestates []*loadinfo
                for n := i; n < len(universe.globe.loads); n += 1 {
                    var load = universe.globe.loads[n]
                    loop = append(loop, load)
                    if load.traveUseLoop() {
                        loopTravestates = append(loopTravestates, load)
                        s += "<" + load.specName + "> → "
                    } else {
                        s += load.specName + " → "
                    }
                }
                if loaded != nil && loaded.opts.traveUseLoop {
                    s += "<" + specName + ">"
                } else {
                    s += specName
                }

                if traveUseLoop = (loopTravestates != nil); !traveUseLoop {
                    erro(ctx, "%s: loop detected: %s", l.project, s).debug(10)
                } else if options.verboseImport || options.verboseUsing || options.verboseLoads {
                    prompt(ctx, "%s: loop detected: %v\n", l.project, s).debug(10)
                }
            }
        }
    }

    var scope = l.Scope()
    if false && traveUseLoop {
        if loaded == nil {
            // ...
        } else if _, a := scope.ProjectName(ctx, loaded.name, loaded); a != nil {
            if val, ok := a.(*ProjectName); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, a).debug(1)
            }
        }
        return
    }

    defer func(a []*Project) {
        var scope = l.Scope()
        if loaded == nil {
            erro(ctx, "%v (%v,dir=%v) not loaded in %v", specName, absPath, isDir, scope).debug(32)
            return
        } else if name, _ := scope.Lookup(loaded.name).(*ProjectName); name == nil {
            if _, alt := scope.ProjectName(ctx, loaded.name, loaded); alt != nil {
                if val, ok := alt.(*ProjectName); !ok || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                }
            }
            if false {
                erro(ctx, "%v (%v,dir=%v) not in %v", specName, absPath, isDir, l.scopes).debug(1)
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
        if proj, res, isb, err = l.project.hasLoaded(ctx, loaded, traveUseLoop); err != nil {
            erro(ctx, "`%s`: %s", specName, err).debug(1)
            return
        } else if isb {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v is already a base\n", l.project, specName)
            erro(ctx, "`%s` is already a base (proj=%s)", specName, proj)
            errostack(ctx, 10, "%v", ctx).debug(16)
            return
        } else if res {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v already imported by %v\n", l.project, specName, proj)
            erro(ctx, "'%s' already imported by '%s'", specName, proj)
            errostack(ctx, 10, "%v", ctx).debug(16)
            return
        }
    }
    if loaded == nil {
        var okay bool
        defer l.setArgs(l.setArgs(arged))
        if isDir {
            okay = l.loadDir(ctx, specName, absPath, nil)
        } else {
            okay = l.load(ctx, specName, absPath, nil)
        }
        if !okay {
            erro(ctx, "failed loading `%v` (%v)", specName, absPath).debug(1)
            return
        }

        if loaded != nil {
            // already loaded previously
        } else if loaded, _ = universe.globe.loaded[absPath]; loaded != nil {
            // successfully loaded (first)
        } else {
            erro(ctx, "'%s' not loaded (%s)", specName, absPath).debug(1)
        }

        if loaded == nil {
            erro(ctx, "'%s' not smart project", specName).debug(1)
            return
        }
    }

    // Check against the current load list before appending loaded.
    for _, use := range l.project.use.list {
        var (
            up = use.project
            proj *Project
            res, isb bool
        )
        if loaded == up {
            if !opts.noVars && !opts.files {
                erro(ctx, "using `%s` multiple times", specName).debug(10)
            }
            return
        }

        if false && loaded.opts.multiUseAllowed {
            // ...
        } else if proj, res, isb, err = loaded.hasLoaded(ctx, up, traveUseLoop); err != nil {
            erro(ctx, "load '%s' failed: %s", specName, err).debug(1)
            return
        } else if isb {
            if l.project.hasBase(up) {
                // common bases are fine
            } else {
                erro(ctx, "`%s` is already a base", specName).debug(1)
            }
        } else if res && !use.opts.reuse && !up.opts.multiUseAllowed && !loaded.opts.multiUseAllowed {
            if true {
                warn(ctx, "`%s` has already imported `%s` (from %s)", loaded, up, proj)
                if loaded != up { warn(at(ctx,loaded.position), "project %s", loaded) }
                if proj != up { warn(at(ctx,proj.position), "project %s", proj) }
                warn(at(ctx,up.position), "project %s", up).debug(6)
            } else {
                warnstack(ctx, -1, "`%s` has already imported `%s` (from %s)", loaded, up, proj).debug(64)
            }
        }

        if proj, res, isb, err = up.hasLoaded(ctx, loaded, traveUseLoop); err != nil {
            erro(ctx, "load '%s' failed: %s", specName, err).debug(1)
            return
        } else if isb {
            warn(ctx, "`%s` is already base of `%s` (%s)", loaded, up, proj).debug(1)
        } else if res && !use.opts.reuse && !loaded.opts.multiUseAllowed {
            warn(ctx, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj)
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
            l.p.importFileMaps(ctx, opts.public || opts.filesPub, specVal)
        } else {
            l.p.importFileMaps1(ctx, opts, loaded)
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

func (l *loader) loadPlugin(ctx Context) (err error) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadPlugin")) }
    if l.project == nil {
        erro(ctx, "current project is nil").debug(32)
        return
    }

    var g = stat(ctx, "smart.go", "", l.project.absPath)
    if g == nil { return /* smart.go was not presented */ }

    var src = g.Strval(ctx)
    s := strings.Replace(l.project.relPath, "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath(ctx, "", "")), "plugins", s)

    var build = true

    so := stat(ctx, /*l.project.name*/"plugin", "", s, nil)
    if s = so.fullname(); s == "" {
        erro(of(ctx,so), "file '%v' has empty fullname", so)
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
            l.project.pluginScope, err = f(ctx.Position(), l.project)
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
            fmt.Fprintf(stderr, "use(%15s) %s ⇒ %v\n", d, l.project, l.project.use)
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
    if l.project.use.append(ctx, usee, params, opts); !opts.noVars {
        applyUseeVars(ctx, l.project, usee)  // aka. ABC += $(use.ABC)
        applyUserVars(ctx, l.project, usee) // aka. use.ABC += $(use.ABC)
        if 0 < len(opts.vars) {
            for _, v := range opts.vars {
                warn(of(ctx,v), "var: %T %v", v, v)
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
        var args = mergex(ctx, plain, t.args...)
        iterateArgumentedIdentifiers(ctx, t.value, func(ident Value, stems []Value) {
            var pos = ident.Position()
            for _, arg := range args {
                if isTrivial(arg) { continue }
                f(MakeBarecomp(pos, ident, arg), append(stems, arg))
            }
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

func (l *loader) define(ctx Context, tok token.Token, identifier, value Value) (defs []*def) {
    iterateArgumentedIdentifiers(ctx, identifier, func(ident Value, stems []Value) {
        var def = l.define1(ctx, tok, ident, value)
        defs = append(defs, def)
    })
    return
}

func (l *loader) define1(ctx Context, tok token.Token, identifier, value Value) (res *def) {
    var alt Object
    switch t := identifier.(type) {
    case *selection:
        var v = t.value(ctx, ident)
        if a, y := v.(*def); y {
            res = a
        } else {
            erro(ctx, "`%v` is not a def (%T)", t, v)
            return
        }

    case *Argumented:
        var args = mergex(ctx, plain, t.args...)
        erro(ctx, "TODO: multiple defs: %v args=%v", t.value, args)
        return

    case *Group:
        erro(ctx, "TODO: multiple defs: %v", t.Elems)
        return

    default: //case *Bareword, *Barecomp, *Qualiword, *Path, *Flag:
        var name = t.Strval(ctx)
        if _, ok := builtins[name]; ok {
            erro(ctx, "`%v` (%v) is builtin name", identifier, name)
            return
        }

        // Resolve base value to derive.
        var prev = l.project.resolveObject(ctx, name)

        if res, alt = l.def(identifier.Position(), name); alt == nil {
            if res == nil {
                erro(ctx, "`%s` is undefined, via %v (%)", name, t, t).debug(1)
                return
            } else if tok == token.ADD_ASSIGN && prev == nil {
                if false {
                    erro(ctx, "`%s` must be defined first to append", name).debug(1)
                    return
                }
                // if prev != nil { if d, y := prev.(*def); y { res.origin = d.origin } }
                // if res.origin == DefVoid { res.origin = DefDefault }
            }
        } else if tok == token.ASSIGN || tok == token.EXC_ASSIGN {
            if ad, okay := alt.(*def); !okay {
                erro(ctx, "`%v` already defined (%T) (%v,%v)", identifier, alt, alt.OwnerProject(), l.project).debug(1)
                return
            } else if ad.owner == l.project && ad.origin != DefConfRef {
                erro(ctx, "`%v` already defined (%T) (%v)", identifier, alt, l.project).debug(1)
                return
            } else {
                res = ad
            }
        } else {
            res = alt.(*def)
        }

        if prev == nil {
            // no derived value
        } else if prev.OwnerProject() == l.project {
            // not derivable def if in the same project
        } else if derived, y := prev.(*def); !y {
            // not a def
        } else if derived == nil {
            erro(ctx, "prev def '%s' is nil", name).debug(1)
        } else if derived == res || (res.value != nil && res.value.refs(ctx, derived)) {
            // same def
        } else if tok == token.ADD_ASSIGN && alt == nil {
            // NOTE: We must set the origin from Void to derived origin! If not, the
            //       Def.Call method will fail to initiate a real 'call' with arguments
            //       set correctly, (see (*def).Call for details).
            if res.origin == DefVoid { res.origin = derived.origin }
            if !isTrivial(derived.value) { res.append(ctx, derived.value) }
        }
    }

    if res == nil {
        erro(ctx, "def is nil for '%v' of %T", identifier, identifier).debug(1)
        return
    }

    res.position = identifier.Position()
    l.assign(ctx, tok, res, alt, value)
    return
}

func (l *loader) rule(clause *parsedRuleData) (entries []Entry) {
    var (
        ctx = at(l, l.Position())
        params  []*def
        depends []Value
        ordered []Value
        progScope *Scope = l.Scope()
        configure = clause.config
        recipes = clause.recipes
    )
    for _, name := range clause.params {
        def := progScope.Lookup(name).(*def)
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
        language: l.p.dialect,
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
        if isTrivial(target) {
            if true { continue } else {
                erro(ctx, "trivial target; %v", clause.targets).debug(1)
                return
            }
        }

        var entry, err = l.project.entry(ctx, clause.special, clause.options, target, prog)
        if err != nil {
            erro(of(ctx,target), "creating entry '%v' failed: %v", target, err)
            return
        } else {
            entries = append(entries, entry)
        }

        if t, okay := entry.Target().(*Flag); okay && t != nil {
            var s = t.name.Strval(ctx)
            if l.project.name != "~" { l.Globe().AddFlagEntry(s, entry) }
        } else if configure {
            if _, y := entry.(*PatternEntry); y {
                erro(ctx, "unsupported pattern configures: %v", target).debug(1)
                return
            } else {
                l.project.configs = append(l.project.configs, entry)
                configuration.entries = append(configuration.entries, entry)
            }
        }
    }
    return
}

type includeFileOpts struct {
    *genericClauseOpts
    ifExists bool `if-exists,ifexists`
    isConfiguration bool // internal
}
func (l *loader) includeFile(ctx Context, opts includeFileOpts, spec Value) {
    var (
        linfo = universe.globe.loads[len(universe.globe.loads)-1]
        specName, fullname string
        err error
    )

    defer func(t time.Time) {
        var panics, _ = checkFailure(ctx, true)
        if proj := l.project; panics > 0 {
            if err != nil { erro(ctx, "panics with error: %v", panics, err) }
            erro(ctx, "failed: got %d panics from %v (%s)", panics, proj, proj.spec).debug(128)
        } else if err != nil {
            erro(ctx, "parse file failed: %v", err).debug(128)
        }

        if d := time.Now().Sub(t); d > time.Duration(options.slow)*time.Millisecond {
            warnstack(ctx, 10, "%v: slow include (%v)", l.project, d).debug(1) //  → %s, filename
        } else if options.verbose {
            info(ctx, "included %v (%v)", spec, d).debug(1)
        }
    } (time.Now())

    ctx = at(ctx, spec.Position())

    // Execute the rule entry to update include source.
    if entry, ok := spec.(*RuleEntry); ok && entry != nil {
        var (
            result []Value
            okay bool
        )
        if result, okay = executeEntry(ctx, entry); !okay {
            erro(ctx, "include entry '%v' failed", entry).debug(1)
            return
        } else if result != nil && opts.verbose {
            info(ctx, "include %v: %v", entry, result).debug(1)
        }
        if false { warn(ctx, "include %T %v: %v", entry.target, entry.target, result).
            debug(1) }
        spec = entry.target
    }

    switch t := spec.(type) {
    case *File:
        if !t.exists() { _ = t.stat(ctx) }
        if !t.exists() && opts.ifExists {
            if opts.debug>0 {
                prompt(ctx, "%v: file not found\n", spec)
                warn(ctx, "").debug(opts.debug)
            }
            return // ignore non-exists files
        }
        if fullname, specName = t.fullname(), t.name; t.info == nil {
            prompt(ctx, "%v: no source file %v, %v\n", fullname, t, t.stat(ctx))
            erro(of(ctx,t), "%v: %v", ctx.Project(), t)
            errostack(ctx, 5, "").debug(16)
            return
        }
    default:
        if specName = spec.Strval(ctx); specName == "" {
            erro(of(ctx,spec), "include: empty string: %v", spec)
            errostack(ctx, 5, "").debug(16)
            return
        }

        var file = l.project.matchFile(ctx, spec)
        if file == nil {
            if filepath.IsAbs(specName) {
                file = stat(ctx, specName, "", "")
            } else {
                file = stat(ctx, specName, "", linfo.absDir)
            }
        } else if !file.exists() { _ = file.stat(ctx) }

        if file != nil && file.exists() {
            fullname = file.fullname()
        } else if opts.ifExists {
            if opts.debug>0 {
                prompt(ctx, "%v: file not found\n", file)
                warn(ctx, "").debug(opts.debug)
            }
            return // ignore non-exists files
        }
    }
    if specName == "" {
        erro(ctx, "include: empty string: %v", spec).debug(10)
        return
    }

    var absDir, baseName = filepath.Split(fullname)
    defer func(mode Mode) { l.mode = mode } (l.mode) // Must restore parse mode!
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))
    if _, _, err = l.parse(ctx, fullname, nil, parseMode|Flat, &opts); err != nil {
        if pe, ok := err.(*fs.PathError); ok && opts.ifExists {
            prompt(ctx, "%v: %v\n", fullname, pe.Err)
            warn(ctx, "include %v", spec)
            warnstack(ctx, 5, "").debug(16)
        } else {
            prompt(ctx, "%v: %v\n", fullname, err)
            erro(ctx, "include: %v", spec)
            errostack(ctx, 5, "").debug(16)
        }
    }

    if n := ctx.checkErrors(true); n > 0 {
        warn(ctx, "got %d errors", n).debug(1)
        if options.failOnErrors { fail(ctx.Position(), "fail by %d errors", ctx.totalErrors()) }
    }
    return
}
func (l *loader) closureScopes() (scopes []*Scope) {
    scopes = append(l.closureContext.closureScopes(), l.Scope())
    return
}

func (l *loader) openScope(comment string) (scopes []*Scope) {
    if false && options.traceLaunch { defer un(trace(t_launch, "loader.openScope")) }
    var pos Position
    if l.p != nil { pos = l.Position() } else {
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
func (l *loader) loadBases(ctx Context, linfo *loadinfo, implicitBase string, params ...Value) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadBases")) }

    // For &(foobar) set from loadArgs
    //defer setclosure(setclosure(cloctx.unshift(l.scope)))
    ctx = closureWith(ctx, l.scopes...) // at(closureWith(l, l.scopes...), position)

    var (
        implicitIndex int
        implicitBases []Value
        position = ctx.Position()
    )
    if file := stat(ctx, dotBase, "", l.project.absPath); file != nil {
        if true {
            var s = file.Strval(ctx)
            assert(s == file.name && s == dotBase, "invalid strval: %v => %v", file, s)
        }
        if !file.info.IsDir() && (l.project.spec == dotBase /*|| l.project.spec == dotConfigure*/) {
            // skip the regular file '.base' to avoid self loading recursively
            // info(ctx, "%v", file).debug(1)
        } else {
            implicitBases = append(implicitBases, file)
        }
    }

    if ns := strings.Split(l.project.name, "."); len(ns) > 2 && ns[len(ns)-1] == "base" {
        var numBaseParams int
        for _, elem := range params {
            if l, y := elem.(*List); y && len(l.Elems) == 1 { elem = l.Elems[0] }
            if a, y := elem.(*Argumented); y { elem = a.value }
            if _, y := elem.(*Pair); y { continue }
            numBaseParams += 1
        }
        if numBaseParams == 0 {
            var segs []Value
            for _, s := range ns[:len(ns)-2] {
                segs = append(segs, MakeBareword(position, s))
            }
            implicitBases = append(implicitBases, MakePath(position, segs...))
            if false { warn(ctx, "%v, %v, %v; %v, %v, %v", l.project.name, ns, segs,
                implicitBase, implicitBases, params).debug(1) }
            implicitBase = "" // discard the implicit base
        } else if false /* && numBaseParams == 1 */ {
            warn(ctx, "%v, %v, %v, %v; %v, %v, %v",
                l.project.name, ns, filepath.Join(ns[:len(ns)-2]...), numBaseParams,
                implicitBase, implicitBases, params)//.debug(6)
        }
    }

    if implicitBase != "" {
        implicitIndex = len(implicitBases)
        implicitBases = append(implicitBases, MakePathStr(position, implicitBase))
    }

    ParamsLoop: for i, elem := range append(implicitBases, params...) {
        var (
            elemPos = elem.Position()
            absPath string
            args []Value
            isDir bool
            err error
        )
        if list, y := elem.(*List); y && len(list.Elems) == 1 { elem = list.Elems[0] }
        if a, y := elem.(*Argumented); y { elem, args = a.value, a.args }
        if p, y := elem.(*Pair); y {
            var (
                identifier = p.Key
                position = identifier.Position()
                name string
            )
            if name = p.Key.Strval(ctx); len(name) > 0 && name[0] == '.' {
                identifier = MakeBarecomp(position, MakeBareword(position, "project"), p.Key)
            }

            var defs = l.define(at(ctx, position), token.ASSIGN, identifier, p.Value)
            if len(defs) == 0 {/* TODO: check defs... */}
            continue ParamsLoop
        }

        var (
            specName string
            specVal Value
            implicit bool
        )
        if specVal = elem.expand(ctx, plain); specVal == nil {
            specVal = elem // okay!
        } else if true && specVal.expandible(ctx, plain) {
            erro(at(ctx,elemPos), "incomplete expand: %T %v -> %T %v", elem, elem, specVal, specVal).debug(1)
            return
        } else if defs := specVal.defs(ctx); len(defs) > 0 {
            erro(at(ctx,elemPos), "incomplete expand: %v -> %v (defs=%v)", elem, specVal, defs).debug(1)
            return
        }

        if specName = elem.Strval(ctx); specName == "" {
            erro(at(ctx,elemPos), "%v: empty base name `%v` (%T)", l.project, elem, elem).debug(1)
            break ParamsLoop
        } else if strings.Contains(specName, "//") {
            erro(at(ctx,elemPos), "%v: invalid spec: %v in %T", l.project, elem, ctx)
            erro(at(ctx,elemPos), "%v: invalid spec: %v -> %v", l.project, elem, specVal)
            erro(at(ctx,elemPos), "%v: invalid spec: %v -> %v", l.project, elem, specName).debug(10)
            break ParamsLoop
        } else if implicitBase != "" && specName == implicitBase {
            if i == implicitIndex { implicit = true } else {
                erro(at(ctx,elemPos), "%v: base '%v' already loaded implicitly", l.project, elem).debug(1)
                if false { break ParamsLoop } else { continue }
            }
        }

        if n := ctx.checkErrors(true); n > 0 {
            warn(at(ctx,position), "%v: %d errors: %v -> %v", l.project, n, elem, specName).debug(1)
            break ParamsLoop
        } else if f, ok := toFile(elem); ok && f.info != nil {
            if absPath = f.fullname(); true { assert(filepath.IsAbs(absPath), "invalid abs path: %v", f) }
            isDir = f.info.IsDir()
        } else if absPath, isDir, err = universe.search(linfo, specName); err != nil {
            erro(at(ctx,elemPos), "%v: search base failed: %v -> %v", l.project, elem, specName)
            erro(at(ctx,elemPos), "%v: search base failed, %v", l.project, err).debug(6)
            break ParamsLoop
        } else if absPath == "" {
            erro(at(ctx,elemPos), "%v: search base failed: %v -> %v", l.project, elem, specName).debug(1)
            break ParamsLoop
        }

        for _, base := range l.project.bases {
            if base.absPath == absPath {
                //erro(at(ctx,elemPos), "duplicated base: %v (in %v)", elem, l.project.bases)
                continue ParamsLoop
            }
        }

        var (
            okay bool
            implicitSaved = l.implicit
            ctx = at(ctx, elem.Position())
        )
        defer l.setArgs(l.setArgs(args))
        if l.implicit = implicit; isDir {
            okay = l.loadDir(ctx, specName, absPath, nil)
        } else {
            okay = l.load(ctx, specName, absPath, nil)
        }
        l.implicit = implicitSaved // restore implicit flag

        if !okay {
            var pos Position
            pos.Filename, pos.Line = absPath, 1
            erro(ctx, "%v: '%s' not loaded'", l.project, specName)
            erro(at(ctx,elemPos), "%v: base '%s' not loaded, %v", l.project, specName, elem)
            erro(at(ctx,position), "%v: base '%s' not loaded, %s", l.project, specName, absPath).debug(6)
            break ParamsLoop
        } else if loaded, yes := universe.globe.loaded[absPath]; yes && loaded != nil {
            if l.project.hasBase(loaded) {
                continue ParamsLoop
            }
            // chain loaded base project, note that err might not be nil
            l.project.bases = append(l.project.bases, loaded) //l.project.Chain(loaded)
        } else if implicit {
            warn(of(ctx,elem), "implicit base '%s' not defined (as %s)", specName, absPath).debug(1)
        } else {
            erro(at(ctx,elemPos), "project `%v`(%T: %s) not loaded (%s)", elem, elem, specName, absPath).debug(1)
            break ParamsLoop
        }
    }

    if false {
        // bypass ...
    } else if o := l.project.resolveObject(ctx, "use.*"); o == nil {
        // erro(ctx, "resolve use.* failed").debug(1)
    } else if d, ok := o.(*def); ok && !isTrivial(d.value) {
        var none = MakeNone(position)
        // Derive use.xxx Defs from bases
        for _, val := range merge(d.value) {
            var name, opts = parseUseNameOpts(ctx, val)
            var us = "use." + name
            var vd, alt = l.project.scope.define(ctx, DefVoid, us, none)
            if vd == nil && !isNil(alt) { vd, _ = alt.(*def) }
            if vd == nil { continue }

            var vals []Value
            for _, base := range l.project.bases {
                var obj = base.scope.lookup(us)
                if isNil(obj) { continue }
                if d, ok := obj.(*def); !ok || isTrivial(d.value) {
                    continue
                } else {
                    vals = append(vals, merge(d.value)...)
                }
            }
            opts.apply(closureWith(ctx, l.project.scope), vd, vals)
        }
    }
    return true
}

func (l *loader) loadDotContainer(ctx Context, ident *Barecomp, identStr string, file *File) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadDotContainer")) }
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.loadDir(ctx, dotContainer, file.fullname(), nil) {
            erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug(1)
            return
        }
    } else if !l.loadFile(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug(1)
        return
    }

    if loaded, yes := universe.globe.loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.Scope().Lookup(loaded.Name()).(*ProjectName)
        if name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.project.name, file).debug(1)
        } else {
            if false && options.verboseLoads {
                prompt(l, "smart: %v (%v)\n", name, file.fullname())
            }

            var opts useOpts
            // TODO: parse the useOpts
            // l.addUsing(at(l, position), loaded, nil, opts)
            l.addUsing(ctx, loaded, nil, opts)

            result = true
        }
    }
    return
}

func (l *loader) loadDotConfigure(ctx Context, ident *Barecomp, identStr string, file *File) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadDotConfigure")) }
    var position = ident.Position()
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.loadDir(ctx, dotConfigure, file.fullname(), nil) {
            erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug(1)
            return
        }
    } else if !l.loadFile(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug(1)
        return
    }

    if loaded, yes := universe.globe.loaded[file.fullname()]; yes && loaded != nil {
        if name, _ := l.Scope().Lookup(loaded.name).(*ProjectName); name == nil {
            if _, alt := l.Scope().ProjectName(at(l, position), loaded.name, loaded); alt != nil {
                if val, ok := alt.(*ProjectName); !ok || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                }
            }
        } else {
            if conf := l.project.configure; conf != nil {
                if conf == loaded { return }
                erro(ctx, ".configure already specified").debug(1)
            }
            l.project.configure, result = loaded, true

            var ctx = at(l, position)
            var opts = useOpts{}
            if false {
                applyUseeVars(ctx, l.project, loaded)  // aka.       ABC += $(use.ABC)
                applyUserVars(ctx, l.project, loaded) // aka. use.ABC += $(use.ABC)
            } else if false {
                for _, usee := range loaded.usees(true, false, false, false) {
                    applyUseeVars(ctx, l.project, usee)  // aka.       ABC += $(use.ABC)
                    applyUserVars(ctx, l.project, usee) // aka. use.ABC += $(use.ABC)
                    //warn(l, "%v %v %v", l.project, loaded, usee).debug(1)
                }
            } else if false {
                if err := l.addUsing(ctx, loaded, nil, opts); err != nil {
                    erro(ctx, "using '%v' failed: %v", loaded, err).debug(1)
                }
                //l.importFileMaps(ctx, public, specVal)
            } else if true {
                for _, usee := range loaded.usees(true, false, false, false) {
                    if err := l.addUsing(ctx, usee, nil, opts); err != nil {
                        erro(ctx, "using '%v' failed: %v", usee, err).debug(1)
                        break
                    }
                    l.p.importFileMaps1(ctx, opts, usee)
                }
            }
        }
    }
    return
}

func (l *loader) declare(ctx Context, keyword token.Token, ident *Barecomp, identStr string, declOpts *projectDeclOpts) (result bool) {
    if identStr == "@" {
        var (
            linfo = universe.globe.loads[0]
            dec, ok = linfo.declares[identStr]
            at, _ = l.Globe().scope.Lookup(identStr).(*ProjectName)
        )
        if !ok {
            dec = &declare{ project: at.Project }
            linfo.declares[identStr] = dec
        }
        dec.backscope = l.Scope()
        l.useesExecuted = nil
        l.project = at.Project
        //l.scope = l.scope
        l.scopes[0] = at.Project.scope
        return true
    } else if _, o := l.Scope().Find(identStr); o != nil {
        if _, ok := o.(*Builtin); ok {
            erro(ctx, "project name '%s' is a builtin name", identStr)
            return
        }
    }

    var (
        name = identStr
        linfo = universe.globe.loads[len(universe.globe.loads)-1]
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
        universe.globe.loaded[linfo.absPath()] = dec.project
        linfo.declares[name] = dec
    }
    if loader := linfo.loader; loader != nil {
        if _, a := loader.scope.ProjectName(ctx, name, dec.project); a != nil {
            if v, ok := a.(*ProjectName); !ok || v == nil {
                erro(of(ctx,a), "`%s` name already taken (%T).", name, a).debug(1)
                return
            }
        }
    }

    dec.project.opts = declOpts
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
                var d, a = l.def(l.Position(), name)
                if d == nil && a != nil { d = a.(*def) }
                d.set(ctx, DefDecl, t.Value)
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
            var d, a = l.def(t.Key.Position(), name)
            if a != nil {
                var ok bool
                if d, ok = a.(*def); !ok {
                    erro(ctx, "'%v' is not a Def (%T)", a, a).debug(1)
                    return
                }
            }
            if d != nil { d.val(ctx, t.Value) }
            warn(ctx, "%v: %v", ident, t)
        }
    }

    if err := l.loadPlugin(ctx); err != nil {
        erro(ctx, "load plugin failed: %v", err).debug(1)
        return
    }
    return true
}

func isConfigureProject(proj *Project) bool {
    return proj == nil ||
        proj.name == dotConfigure ||
        proj.name == "configure" ||
        proj.name == "configure.base"
}

func (l *loader) loadAutoAfter(ctx Context, tag string) {
    if proj := l.project; isConfigureProject(proj) {
        if false && tag == "declare" { info(ctx, "%v: %v", proj, tag).debug(4) }// skip...
    } else if obj := proj.resolveObject(ctx, ".auto.after."+tag); obj == nil {
        if false && tag == "declare" { info(ctx, "%v: %v", proj, tag).debug(4) }// skip...
    } else if d, y := obj.(*def); !y {
        warnstack(ctx, 3, "%v: unsupported .auto: %T %v", proj, obj, obj).debug(1)
    } else if isTrivial(d.value) {
        if false && tag == "declare" { info(ctx, "%v: %v", proj, tag).debug(4) }// skip...
    } else if val := Scalar(d.value.expand(ctx, plain)); isTrivial(val) {
        if false && tag == "declare" { info(ctx, "%v: %v", proj, tag).debug(4) }// skip...
    } else {
        l.includeFile(ctx, includeFileOpts{}, val)
    }
}

func (l *loader) loadProjectConfiguration(ctx Context, linfo *loadinfo, ident *Barecomp, identStr string, declared bool) (result bool) {
    if false { defer un(tracef(t_traverse, "loadProjectConfiguration(%v)", ident)) }

    // ctx = at(ctx, ident.Position())

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
            info(at(ctx,cp), "%s (%s)", l.project, l.project.spec).debug(true, 1)
        } else if options.verbose {
            info(ctx, "%v for %s (%s)", file, l.project.spec, l.project).debug(16)
        }
        var isIncludingConf = l.p.isIncludingConf
        l.p.isIncludingConf = true
        l.includeFile(ctx, includeFileOpts{isConfiguration: true}, file)
        l.p.isIncludingConf = isIncludingConf
    }

    if s := l.project.name; s == dotConfigure { return }
    if l.project.opts.configure || l.project.opts.configureName != "" {
        var s = l.project.opts.configureName
        if s == "" { s = "configure" }

        var load = func(absPath string, isDir bool) bool {
            if isDir {
                return l.loadDir(ctx, s, absPath, nil)
            } else {
                return l.loadFile(ctx, absPath, nil)
            }
        }

        if absPath, isDir, err := universe.search(linfo, s); err != nil {
            erro(ctx, "%v: search configure failed: %v", l.project, s)
            erro(ctx, "%v: search configure failed: %v", l.project, err).debug(6)
            return false
        } else if !load(absPath, isDir) {
            erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, absPath).debug(1)
            return
        } else if loaded, y := universe.globe.loaded[absPath]; !y || loaded == nil {
            erro(ctx, "not loaded: %s (dir=%v)", absPath, isDir).debug(1)
            return
        } else {
            if name, _ := l.Scope().Lookup(dotConfigure).(*ProjectName); name == nil {
                if _, alt := l.Scope().ProjectName(ctx, dotConfigure, loaded); alt != nil {
                    if val, ok := alt.(*ProjectName); !ok || val == nil {
                        erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                    }
                }
            }
            if l.project.configure == loaded { return }
            if l.project.configure != nil {
                erro(ctx, ".configure already specified").debug(1)
                return
            }

            l.project.configure, result = loaded, true

            var opts = useOpts{}
            if false {
                applyUseeVars(ctx, l.project, loaded)  // aka.       ABC += $(use.ABC)
                applyUserVars(ctx, l.project, loaded) // aka. use.ABC += $(use.ABC)
            } else if false {
                for _, usee := range loaded.usees(true, false, false, false) {
                    applyUseeVars(ctx, l.project, usee)  // aka.       ABC += $(use.ABC)
                    applyUserVars(ctx, l.project, usee) // aka. use.ABC += $(use.ABC)
                    //warn(l, "%v %v %v", l.project, loaded, usee).debug(1)
                }
            } else if false {
                if err := l.addUsing(ctx, loaded, nil, opts); err != nil {
                    erro(ctx, "using '%v' failed: %v", loaded, err).debug(1)
                }
                //l.importFileMaps(ctx, public, specVal)
            } else if true {
                for _, usee := range loaded.usees(true, false, false, false) {
                    if err := l.addUsing(ctx, usee, nil, opts); err != nil {
                        erro(ctx, "using '%v' failed: %v", usee, err).debug(1)
                        break
                    }
                    l.p.importFileMaps1(ctx, opts, usee)
                }
            }
        }
    } else if file := stat(ctx, dotConfigure, "", l.project.absPath); file.exists() {
        if true {
            warn(ctx, ".configure is deprecated").debug(1)
        } else if identStr == dotConfigure {
            erro(ctx, "provided .configure for a .configure project").debug(1)
        } else if !l.loadDotConfigure(ctx, ident, identStr, file) {
            //erro(ctx, "declare %s: %s/.configure", name, l.project.absPath)
        }
    }
    return true
}

func (l *loader) loadProjectContainer(ctx Context, ident *Barecomp, identStr string) (result bool) {
    ctx = at(ctx, ident.Position())
    if l.project.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").debug(1)
            return
        }

        // Looking for project specific .container module
        if file := stat(ctx, dotContainer, "", l.project.absPath); file.exists() {
            if !l.loadDotContainer(ctx, ident, identStr, file) {
                //erro(ctx, "declare %s: %s/.container", name, l.project.absPath)
            }
            if options.verbose {
                info(ctx, "%v for %s (%s)\n", file, l.project.spec, l.project).debug(1)
            }
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(ctx, l.project.absPath, func(s string) bool {
            var file = stat(ctx, dotContainer, "", filepath.Join(s, ".smart"))
            if !file.exists() {
                // no docking enabled
            } else if !l.loadDotContainer(ctx, ident, identStr, file) {
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
        if dec, ok := universe.globe.loads[0].declares[identStr]; ok {
            l.scopes[0] = dec.backscope
            l.useesExecuted = dec.useesExecuted
            dec.backscope = nil
            dec.useesExecuted = nil
        }
        return nil
    }

    var linfo = universe.globe.loads[len(universe.globe.loads)-1]
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
    var ctx Context = l
    if outer := l.Scope(); outer == nil {
        erro(ctx, "no parent scope '%s' (%v)", name, comment).debug(8)
    } else {
        if strings.HasPrefix(outer.Comment(), "dir ") {
            outer = outer.outer // discard dir scope
        }

        scopes = l.openScope(comment)
        if scope := l.Scope(); scope != nil {
            outer.ScopeName(at(l, scope.position), name, scope)
        } else {
            erro(ctx, "open scope '%s' failed (%v)", name, comment).debug(8)
        }
    }
    return
}

func (l *loader) resolveObject(value Value) (name string, result Value) {
    var pos = value.Position()
    if !pos.IsValid() { pos = l.Position() }
    if _, ok := value.(*selection); ok {
        panic(failure{pos,"resolving a selection"})
    }

    var optional bool
    var ctx = at(l, pos)
    if name = value.Strval(ctx); name == "" {
        erro(ctx, "name '%v' is empty", name).debug(1)
        return
    } else if optional = strings.HasSuffix(name, "?"); optional {
        name = strings.TrimSuffix(name, "?")
    }

    if l.Scope() == nil {
        erro(ctx, "nil scope to resolve '%v'", name).debug(1)
        return
    } else if _, result = l.Scope().Find(name); !isNil(result) {
        // okay!
    } else if project := l.project; project == nil {
        erro(ctx, "nil project to resolve '%v'", name).debug(1)
        return
    } else if result = project.resolveObject(ctx, name); isNil(result) {
        if optional {
            result = unresolved{value, project}
            return
        }
        //erro(ctx, "%v: resolved object '%v' is nil", project, name).debug(1)
    }
    return
}

func (l *loader) resolveEntries(target Value) (entries *ResolveEntries) {
    var (
        pos = l.Position()
        ctx = at(l, pos)
        name = target.Strval(ctx)
    )
    entries = l.project.resolveEntries(ctx, name, false, false)
    return
}

func (l *loader) def(position Position, name string) (def *def, alt Object) {
    var scope = l.Scope()
    if  strings.HasPrefix(scope.comment, "file ") && l.mode&Flat != 0 {
        // use project scope if defining in flat file (aka. include)
        // to ensure that the symbol is valid in the project
        scope = l.Scope()
    }
    def, alt = scope.define(at(l, position), DefVoid, name, nil)
    if def != nil { def.position = position }
    return
}

func (l *loader) assign(ctx Context, tok token.Token, def *def, alt Object, value Value) {
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
        } else if def.value != nil && def.value.refs(ctx, value) {
            erro(ctx, "self-ref value: %v -> %v ; %v ; %v",
                def.value, value, def.value, value).debug(1)
        } else {
            def.append(ctx, value)
        }
    case token.SHI_ASSIGN: //  =+
        if isTrivial(value) {
            // NOOP
        } else if def.value != nil && def.value.refs(ctx, value) {
            erro(ctx, "self-ref value: %v -> %v ; %v ; %v",
                def.value, value, def.value, value).debug(1)
        } else {
            var tail = def.value
            def.val(ctx, value)
            def.append(ctx, merge(tail)...)
            warn(ctx, "%v; %v; %v", value, tail, def).debug(1)
        }
    case token.SUB_ASSIGN: // -=
        if isTrivial(def.value) {
            // NOOP
        } else {
            var (
                vals []Value
                sub = merge(value)
            )
        ForOldVals:
            for _, val := range merge(def.value) {
                for _, v := range sub {
                    if val.cmp(ctx, v) == cmpEqual { continue ForOldVals }
                }
                vals = append(vals, val)
            }
            def.value = MakeList(def.position, vals...)
        }
    case token.SAD_ASSIGN, token.SSH_ASSIGN: // -+=, -=+
        var (
            vals []Value
            newVals = merge(value)
        )
        if isTrivial(def.value) {
            // NOOP
        } else {
        ForOldVals1:
            for _, val := range merge(def.value) {
                for _, v := range newVals {
                    if val.cmp(ctx, v) == cmpEqual { continue ForOldVals1 }
                }
                vals = append(vals, val)
            }
        }
        if token.SAD_ASSIGN == tok {
            vals = append(vals, newVals...) // -+=
        } else {
            vals = append(newVals, vals...) // -=+
        }
        def.value = MakeList(def.position, vals...)
    default:
        erro(ctx, "unknown origin: %v %v %v",
            def.origin, def.name, tok).debug(1)
    }
    return
}

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
//
func readSource(filename string, source... interface{}) ([]byte, error) {
    if len(source) > 0 {
        var ( buf bytes.Buffer ; n int )
        for _, src := range source {
            if src == nil { continue } else { n += 1 }
            switch s := src.(type) {
            case string: buf.Write([]byte(s))
            case []byte: buf.Write(s)
            case *bytes.Buffer: if s != nil { buf.Write(s.Bytes()) }
            case io.Reader: if _, e := io.Copy(&buf, s); e != nil { return nil, e }
            default: return nil, fmt.Errorf("invalid source (%T)", src)
            }
        }
        if n > 0 { return buf.Bytes(), nil }
    }
    return ioutil.ReadFile(filename)
}

func (l *loader) parse(ctx Context, filename string, src interface{}, mode Mode, opts *includeFileOpts) (f *parsedFile, res []Value, err error) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.ParseFile")) }
    if options.verbose {
        if ctx.Position().Filename == filename {
            info(ctx, "loading ...")
        } else {
            prompt(ctx, "%s:1:info: loading ...\n", filename)
            info(ctx, "loading %v", filename)
        }
    }

    assert(ctx.loader() == l, "require the same loader context")

    defer func(t time.Time, saved *parser, m Mode) {
        if true { ctx = l.p.posit() }

        var panics, _ = checkFailure(ctx, true)
        if proj := l.project; panics > 0 {
            if err != nil { erro(ctx, "panics with error: %v", panics, err) }
            erro(ctx, "failed: got %d panics from %v (%s)", panics, proj, proj.spec).debug(128)
        } else if err != nil {
            erro(ctx, "parse file failed: %v", err).debug(128)
        }

        if d := time.Now().Sub(t); d > time.Duration(options.slow)*time.Millisecond {
            warnstack(ctx, 10, "%v: slow loading (%v)", l.project, d).debug(1) //  → %s, filename
        } else if options.verbose {
            info(ctx, "loaded %v (%v)", filename, d).debug(1)
        }

        l.p, l.mode = saved, m
    } (time.Now(), l.p, l.mode)

    l.mode = mode

    var text []byte
    if text, err = readSource(filename, src); err != nil {
        if _, ok := err.(*fs.PathError); ok && opts.ifExists {
            if opts.debug>0 {
                prompt(ctx, "%v: source file not found\n", filename)
                warnstack(ctx, 5, "#>", opts.all[0]).debug(opts.debug)
            }
        } else {
            prompt(ctx, "%v: %v\n", filename, err)
            erro(ctx, "read source failed: %v (%T)", err, err)
            errostack(ctx, 5, "").debug(32)
        }
        return
    }

    if l.p = (&parser{ Context: ctx }); opts != nil {
        l.p.isIncludingConf = opts.isConfiguration
    }

	var scanMode scanner.Mode
	if l.mode&ParseComments != 0 {
		//scanMode = scanner.ScanComments
	}
    var file = universe.globe.file(filename, text)
	l.p.scanner.Init(file, text, scanMode, func(p token.Position, s string) {
        errostack(at(ctx,Position(p)), 3, "%s", s).debug(128)
    })
	l.p.next(true)

    if ctx = l.p.posit(); l.mode&parsingText != 0 {
        res = l.p.parseText(ctx)
    } else if f = l.p.parseFile(ctx); f == nil {
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

    var def *def
    var ctx = at(l, l.p.Position())
ListLoop:
    for _, d := range list {
        var name = d.Name()
        if  strings.HasPrefix(name, "~") ||
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
            def.set(ctx, DefConfDir, MakeString(ctx.Position(), s))
        } else if s != nil {
            erro(ctx, "Name `%s' already taken, not def (%T).", name, s)
            break ListLoop
        }
    }
    return
}

// parseDir calls ParseFile for all files with names ending in ".go" in the
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
func (l *loader) parseDir(pos Position, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*Project) {
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseParse /*&& d > 50*time.Millisecond*/ {
            fmt.Fprintf(stderr, "parse(%15s) %s ⇒ %s\n", d, l.project, path)
        }
    } (time.Now())

    var ctx = at(l, pos)
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
        if i > 0 {
            if first, s := list[0], a.Name(); s == "do.smart" ||
                (s == "build.smart" && first.Name() != "do.smart") {
                list[0] = a
                list[i] = first
            }
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
            var src, _, err = l.parse(ctx, filename, nil, mode|parsingDir, nil)
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
                if l.p != nil && l.p.scanner.File() != nil {
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
func (l *loader) load(ctx Context, specName, absPath string, source interface{}) (result bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.load")) }
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseLoads /*&& d > 50*time.Millisecond*/ {
            loaded, _ := universe.globe.loaded[absPath]
            if l.project == nil {
                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
            }
        }
    } (time.Now())

    if absPath == "" {
        erro(ctx, "no such module `%s' (in paths %v)", specName, universe.paths)
        return
    } else if !filepath.IsAbs(absPath) {
        erro(ctx, "invalid abs name `%s' (%s)", absPath, specName)
        return
    }

    // Check loaded project.
    if loaded, yes := universe.globe.loaded[absPath]; yes {
        if _, a := l.Scope().ProjectName(at(l, l.Position()), loaded.Name(), loaded); a != nil {
            if val, ok := a.(*ProjectName); !ok || val == nil {
                erro(ctx, "`%v` name already taken (%T).", loaded, a)
            }
        }
        result = true
        return
    }

    var absDir, baseName = filepath.Split(absPath)
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

    var doc, _, err = l.parse(ctx, absPath, source, parseMode, nil)
    if n := l.checkErrors(true); n > 0 {
        warn(ctx, "load '%s' got %d errors", specName, n).debug(1)
        if options.failOnErrors { fail(l.Position(), "fail by %d errors", l.totalErrors()) }
        return
    } else if err != nil {
        erro(ctx, "load: %v", err).debug(1)
    } else if doc == nil {
        erro(ctx, "load: doc is nil (%s)", absPath).debug(1)
    } else {
        result = true
    }
    return
}

func (l *loader) loadDir(ctx Context, specName, absDir string, filter func(os.FileInfo) bool) (loadedOkay bool) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadDir")) }
    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseLoads /*&& d > 50*time.Millisecond*/ {
            loaded, _ := universe.globe.loaded[absDir]
            if l.project == nil {
                fmt.Fprintf(stderr, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                fmt.Fprintf(stderr, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
            }
        }
    } (time.Now())

    var pos Position = ctx.Position()
    if !pos.IsValid() { pos = positionForDir(absDir) }
    if !filepath.IsAbs(absDir) {
        erro(ctx, "needs absolute dir `%s' (%s)", absDir, specName).debug(1)
        return
    }

    var loaded *Project
    defer func() {
        if loaded == nil {
            erro(ctx, "%v (%v) not loaded in %v", specName, filepath.Base(absDir), l.Scope())
            errostack(l, 16, "%v", specName).debug(512)
            return
        }
        if proj := l.Scope().project; proj == nil {
            if false { erro(ctx, "%v: no owner project for %s", loaded.name, l.Scope()).debug(1) }
        } else if name, _ := proj.scope.Lookup(loaded.name).(*ProjectName); name == nil {
            if _, alt := proj.scope.ProjectName(at(l, pos), loaded.name, loaded); alt != nil {
                if val, ok := alt.(*ProjectName); !ok || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                }
            }
        }
    } ()

    // Check loaded project.
    if loaded, loadedOkay = universe.globe.loaded[absDir]; loadedOkay {
        /*if _, a := l.Scope().ProjectName(at(l, pos), loaded.name, loaded); a != nil {
            if val, ok := a.(*ProjectName); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, a).debug(1)
            }
        }*/
        return
    }

    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

    var mods = l.parseDir(pos, absDir, filter, parseMode)
    if n := l.checkErrors(true); n > 0 {
        erro(ctx, "%d diagnostic errors parsing module '%s'", n, specName).debug(12)
        if options.failOnErrors { fail(l.Position(), "fail by %d errors", l.totalErrors()) }
        return
    }

    // FIXME: loading failed if different 'project' found in
    // the same dir, for example:
    //      project Foo # file do.smart
    //      project # file config.smart
    if len(mods) == 0 && filepath.Base(specName) != "@" {
        if l.implicit {
            warn(l, "%s not loaded (as %s, implicitly)", specName, absDir).debug(8)
            loadedOkay = true // okay for implicit loading
        } else {
            erro(ctx, "%s not loaded (as %s)", specName, absDir).debug(8)
        }
    } else if loaded, loadedOkay = universe.globe.loaded[absDir]; loadedOkay && loaded != nil {
        // Good!
    } else if filepath.Base(specName) != "@" {
        erro(ctx, "%s not loaded (as %s, implicit=%v)", specName, absDir, l.implicit).debug(1)
    }
    return
}

func (l *loader) loadFile(ctx Context, filename string, source interface{}) bool {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadFile")) }
    var spec string
    switch dir, base := filepath.Split(filename); base {
    case dotBase, dotConfigure: spec = base
    default: spec, _  = filepath.Rel(l.WorkDir(), dir)
    }

    var position Position
    position.Filename = filename
    return l.load(at(ctx, position), spec, filename, source)
}

func (l *loader) loadPath(path string, filter func(os.FileInfo) bool) bool {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadPath")) }
    var position Position
    var s, _ = filepath.Rel(l.WorkDir(), path)
    position.Filename = s
    return l.loadDir(at(l, position), s, path, filter)
}

func (l *loader) loadText(filename string, text string) (res []Value) {
    if options.traceLaunch { defer un(trace(t_launch, "loader.loadText")) }

    defer func(saved *parser) { l.p = saved } (l.p)

    if g := l.Globe(); g.main == nil {
        l.scopes[0] = g.os.scope
    } else {
        l.scopes[0] = g.main.scope
    }
    l.useesExecuted = nil

    var err error
    var opts includeFileOpts
    var position Position
    position.Filename = filename

    var ctx Context = at(l, position)
    if _, res, err = l.parse(ctx, filename, text, parsingText, &opts); err != nil {
        prompt(ctx, "%v: %v\n", filename, err)
        erro(ctx, "load text failed: %v", err)
        errostack(ctx, 5, "").debug(32)
    }
    return
}
