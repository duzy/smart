//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
    "bytes"
    "fmt"
    "io"
    "io/fs"
    "io/ioutil"
    "os"
    "os/exec"
    "path/filepath"
    "plugin"
    "reflect"
    "strings"
    "time"
    "unicode/utf8"
)

const (
    dotBase = ".base"
    dotContainer = ".container"
    dotConfigure = ".configure"

    entryFileName = "do.smart"

    optSortErrors = false
)

type ResolveBits int

const (
    FromBase ResolveBits = 1<<iota
    FromProject
    FromGlobe
    FromHere

    FindDef
    FindRule

    anywhere = FromHere
    global   = FromGlobe
    local    = FromProject
    nonlocal = FromGlobe | FromBase | FromProject
)

type EvalBits int

const (
    KeepClosures EvalBits = 1<<iota
    KeepDelegates

    // Wants value for rule depends.
    DependValue

    // Wants v.string(ctx), expands delegates and closures,
    // turn off KeepClosures, KeepDelegates.
    StringValue = 0
)

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
type Mode uint

const (
    ModuleClauseOnly Mode = 1<<iota // stop parsing after project or module clause
    ImportsOnly               // stop parsing after import declarations
    ParseComments             // parse comments and add them to AST
    Flat                      // parsing in flat mode (donot create a new module)
    DeclarationErrors         // report declaration errors
    SpuriousErrors            // same as AllErrors, for backward-compatibility

    parsingText
    parsingDir
)

var parseMode = DeclarationErrors

type declare struct {
    project *project
    backscope *scope
    useesExecuted []*project
}

type loadinfo struct {
    absDir string
    baseName string
    specName string
    useesExecuted []*project
    loader *project
    loadee *project // the current loading project
    scopes []*scope
    declares map[string]*declare // all projects declared in the load dir
}

func (li *loadinfo) absPath() string {
    return filepath.Join(li.absDir, li.baseName)
}

func (li *loadinfo) traveUseLoop() (result bool) {
    var first bool = true
    for _, decl := range li.declares {
        if first || result {
            result = decl.project.opt.traveUseLoop
        }
        if !result { break }
        first = false
    }
    return
}

func _loader(c Context) *loader { return cast[*loader](c) }

type loader struct {
    terminal
    mode     Mode
    p       *parser
    project *project
    loadArgs      []Value
    loadStack     []*project // load path
    useStack      []*project // use path
    useesExecuted []*project // all executed usees
    implicit bool // loading current project implicitly, aka. via foo.bar.Baz (implicit foo/bar loaded)
    verpre string // verbose prefix
}
func (l *loader) inner() Context { return &l.terminal }
func (l *loader) cast(t reflect.Type) Context {
    if reflect.TypeOf(l)   == t { return l }
    if reflect.TypeOf(l.p) == t { return l.p }
    return l.terminal.cast(t)
}
func (l *loader) Position() Position {
    if l.p != nil {
        return l.p.Position()
    } else {
        return l.Context.Position()
    }
}
func (l *loader) do(ctx Context, op any) any {
    switch op.(type) {
    case getProject: return l.project
    case getClosure:
        // if x, y := l.terminal.do(ctx, op).([]*scope); y {
        //     return append(x, ?...)
        // } else {
        //     return ?
        // }
    }
	return l.terminal.do(ctx, op)
}

func (g *globe) restoreLoadingInfo(l *loader) {
    var last  = len(g.loads)-1
    var linfo = g.loads[last]
    g.loads = g.loads[0:last]
    l.useesExecuted = linfo.useesExecuted
    l.project = linfo.loader
    l.s = linfo.scopes
}

func (g *globe) saveLoadingInfo(l *loader, specName, absDir, baseName string) *loader {
    g.loads = append(g.loads, &loadinfo{
        absDir: absDir,
        baseName: baseName,
        specName: filepath.Clean(specName),
        useesExecuted: l.useesExecuted,
        loader:   l.project,
        scopes:   l.s,
        declares: make(map[string]*declare),
    })
    return l
}

type useOpts struct {
	noUse  bool `nu,nouse,uu,unuse` // TODO
    noVars bool `nv,novars,no-vars`
	files  bool `f,files` // NOTE: see also '-import(xxxx)'
	reuse  bool `r,ru,reuse,reusing`
    vars []Value `var,vars`
}

type usevar struct {
    unique bool `uni,uniq,unique`
    remainder []Value // will be opts for unique
}
func (uo *usevar) apply(ctx Context, d *def, u ...*def) {
    var vals []Value
    for _, u := range u { for _, v := range merge(u.value) {
        if t, y := v.(*def); y && t != nil {
            vals = append(vals, merge(t.value)...)
        } else {
            vals = append(vals, v)
        }
    }}
    if len(vals) == 0 { return } else
    if d.append(ctx, vals...); uo.unique { a := merge(d.value)
        d.value = call(ctx, "unique", uo.remainder, a...)
    }
}
func usefor(ctx Context, user *project, f func(usevar, Value, Value, string)) {
    defer trace(ctx)

    var o = user.resolve(ctx, "use.*")
    if o != nil { if d, y := o.(*def); y && d != nil { for _, spec := range merge(d.value) {
        var ( val = spec ; name string ; op usevar ; ctx = at(ctx, spec) )
        if a, y := spec.(*argumented); y { val = a.Value
            op.remainder = parseOpts(final{ctx}, &op, a.args...)
        }
        if name = val.string(ctx); name == "" { c := user.configure
            if c != nil { t := c.resolve(ctx, "use.*")
                note(ctx, "%v", ts(t))
            }
            erro(at(ctx,val), "%v: empty use spec: '%v' (%T)", user, spec, spec).debug()
        } else {
            f(op, spec, val, name)
        }
    }}}
}
func usevars(ctx Context, user, usee *project) {
    var ddd = _universe(ctx).ddd == "use"
    usefor(ctx, user, func(op usevar, spec, val Value, name string) {
        var useDef *def
        if o := usee.Lookup("use."+name); o != nil {
            if d, y := o.(*def); y && d != nil { useDef = d } else {
                erro(ctx, "use.%s: nil def: %T %v", name, o, o).debug(3)
                return
            }
        }
        if useDef == nil { return }

        var dd []*def

        // 1. use.XXX += $(use.XXX)
        {
            var d, a = user.set(ctx, useDef.ident(ctx), defVoid)
            var isNewDef = d != nil && a == nil
            if d == nil && a != nil { d, _ = a.(*def) }
            if d == nil { return }
            if d.value != nil && d.value.String() == "unique" {
                note(ctx, "%v (%v, %v, %v)", d, user, isNewDef, a)
                note(ctx, "%v", useDef).debug(10)
            }
            if isNewDef || isTrivial(d.value) {
                dd = append(dd, baseNonTrivialDefs(ctx, user, useDef.ident(ctx))...)
            }
            op.apply(closure_with(ctx, usee.scope), d, append(dd, useDef)...)
        }

        if useDef.value == nil || isTrivial(useDef.value) { return }

        // 2. XXX += $(use.XXX)
        {
            var d, a = user.set(ctx, name, defVoid)
            var isNewDef = d != nil && a == nil
            if d == nil && a != nil { d, _ = a.(*def) }
            if d == nil { return }
            if isNewDef && false {
                if dd == nil { dd = append(dd, baseNonTrivialDefs(ctx, user, useDef.ident(ctx))...) }
                dd = append(dd, baseNonTrivialDefs(ctx, user, name)...)
            }
            op.apply(closure_with(ctx, user.scope), d, append(dd, useDef)...)
        }
    })
    if ddd { note(ctx, "%v ⇒ %v ; %v", user, usee, user.resolve(ctx, "use.*")).debug(5) }
}
func baseNonTrivialDefs(ctx Context, user *project, name string) (dd []*def) {
    for _, base := range user.bases { if o := base.resolve(ctx, name); o != nil {
        if t, y := o.(*def); y && !isTrivial(t.value) { dd = append(dd, t) }
    }}
    return
}

func (l *loader) usespec(ctx Context, opts useOpts, specVal Value, arged []Value, params ...Value) (loaded *project) {
	if true { defer trace(ctx) }

    var (
        u = _universe(l.Context)
        globe = _universe(l.Context).globe
        linfo = globe.loads[len(globe.loads)-1]
        absPath, specName string
        isDir, traveUseLoop bool
        err error
    )
    if t, y := specVal.(*project); y {
        loaded = t
    } else if specName = specVal.string(ctx); specName == "" {
        errostack(ctx, 3, "empty spec: %v", ts(specVal)).debug(6)
        return
    } else if absPath, isDir = u.search(ctx, linfo, specName); absPath == "" {
        errostack(ctx, 3, "missing `%s` (in %v)", specName, u.paths).debug(6)
        return
    } else {
        if loaded, y = globe.loaded[absPath]; !y {
            if false { warnstack(ctx, 3, "not project: %s", absPath).debug(6) }
        }
        // Checking circular loads. See also project.loopImportPath()!
        for i, load := range globe.loads {
            if load.absDir == absPath {
                var s string
                var loop, loopTravestates []*loadinfo
                for n := i; n < len(globe.loads); n += 1 {
                    var load = globe.loads[n]
                    loop = append(loop, load)
                    if load.traveUseLoop() {
                        loopTravestates = append(loopTravestates, load)
                        s += "<" + load.specName + "> → "
                    } else {
                        s += load.specName + " → "
                    }
                }
                if loaded != nil && loaded.opt.traveUseLoop {
                    s += "<" + specName + ">"
                } else {
                    s += specName
                }

                if traveUseLoop = (loopTravestates != nil); !traveUseLoop {
                    erro(ctx, "%s: loop detected: %s", l.project, s).debug(10)
                } else if u.verboseImport || u.verboseUsing || u.verboseLoads {
                    prompt(ctx, "%s: loop detected: %v\n", l.project, s).debug(10)
                }
            }
        }
    }

    var scope = l.scope()
    if false && traveUseLoop {
        if loaded == nil {
            // ...
        } else if _, a := scope.projectname(ctx, loaded.name, loaded); a != nil {
            if val, ok := a.(*project); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, a).debug()
            }
        }
        return
    }

    defer func(a []*project) { if l.loadStack = a; loaded == nil {
        erro(ctx, "%v not loaded (%v,dir=%v)", specName, absPath, isDir).debug()
        return
    } else if name, _ := scope.Lookup(loaded.name).(*project); name == nil {
        if _, alt := scope.projectname(ctx, loaded.name, loaded); alt != nil {
            if val, ok := alt.(*project); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug()
            }
        }
        if false {
            erro(ctx, "%v (%v,dir=%v)", specName, absPath, isDir).debug()
            return
        }
    }} (l.loadStack)

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
    if u.verboseImport {
        if len(l.loadStack) > 1 {
            defer func(s string) { l.verpre = s } (l.verpre)
            l.verpre += "│"
        }
        if opts.reuse {
            prompt(ctx, "%s├┬→\"%s\" (reuse, %s)\n", l.verpre, specName, absPath)
        } else {
            prompt(ctx, "%s├┬→\"%s\" (%s)\n", l.verpre, specName, absPath)
        }
        defer func(t time.Time) {
            var name string
            var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s
            var ds = fmt.Sprintf("(%s)", d)
            if d>=1*time.Second { ds = fmt.Sprintf("▶%s◀",ds) }
            if loaded != nil { name = loaded.name }
            prompt(ctx, "%s├┴─\"%s\" ⇢ %s %s\n", l.verpre, specName, name, ds)
        } (time.Now())
    }

    if loaded != nil && !(/*opts.noVars || */opts.reuse) {
        var ( proj *project ; res, isb bool )
        if proj, res, isb, err = l.project.hasLoaded(ctx, loaded, traveUseLoop); err != nil {
            erro(ctx, "`%s`: %s", specName, err).debug()
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
            okay = l.directory(ctx, specName, absPath, nil)
        } else {
            okay = l.load(ctx, specName, absPath, nil)
        }
        if !okay {
            erro(ctx, "failed loading `%v` (%v)", specName, absPath).debug()
            return
        }

        if loaded != nil {
            // already loaded previously
        } else if loaded, _ = globe.loaded[absPath]; loaded != nil {
            // successfully loaded (first)
        } else {
            erro(ctx, "'%s' not loaded (%s)", specName, absPath).debug()
        }

        if loaded == nil {
            erro(ctx, "'%s' not smart project", specName).debug()
            return
        }
    }

    // Check against the current load list before appending loaded.
    for _, use := range l.project.use.list {
        var (
            up = use.project
            proj *project
            res, isb bool
        )
        if loaded == up {
            if !opts.noVars && !opts.files {
                errostack(ctx, 10, "%v: using `%s` multiple times: %v", l.project, specName, l.project.use.list).debug(10)
            }
            return
        }

        if false && loaded.opt.multiUseAllowed {
            // ...
        } else if proj, res, isb, err = loaded.hasLoaded(ctx, up, traveUseLoop); err != nil {
            erro(ctx, "load '%s' failed: %s", specName, err).debug()
            return
        } else if isb {
            if l.project.hasBase(up) {
                // common bases are fine
            } else {
                erro(ctx, "`%s` is already a base", specName).debug()
            }
        } else if res && !use.opts.reuse && !up.opt.multiUseAllowed && !loaded.opt.multiUseAllowed {
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
            erro(ctx, "load '%s' failed: %s", specName, err).debug()
            return
        } else if isb {
            warn(ctx, "`%s` is already base of `%s` (%s)", loaded, up, proj).debug()
        } else if res && !use.opts.reuse && !loaded.opt.multiUseAllowed {
            warn(ctx, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj)
            warnstack(ctx, 8, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj).debug()
        }
    }

    if u.verboseImport { defer func(t time.Time) {
        prompt(ctx, "%s├┤ %s:import(%s) (%s)\n", l.verpre, l.project, specName, time.Now().Sub(t))
    } (time.Now()) } //*time.Millisecond // µs, ms, s ┼

    if err = l.use(ctx, loaded, params, opts); err != nil { // see usevars
        erro(ctx, "using '%v' failed: %v", loaded, err).debug()
        return
    }
    return
}

const pluginDifferentVersionError = `plugin was built with a different version of package`
var numUpdatedPlugins = 0

func buildPlugin(ctx Context, s, src string) (err error) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.buildPlugin")) }

    prompt(ctx, "smart: Build %v …", src)
    dir, _ := filepath.Split(src)
    o := &bytes.Buffer{}
    c := exec.Command("go", "build", "-buildmode=plugin", "-o", s)
    c.Stdout, c.Stderr, c.Dir = o, o, dir
    if err = c.Run(); err == nil {
        numUpdatedPlugins += 1
        prompt(ctx, "… ok\n")
        prompt(ctx, "smart: Plugin updated, please relaunch.\n")
        os.Exit(0)
    } else {
        prompt(ctx, "… error\n")
        prompt(ctx, "%s", o)
    }
    return
}

func (l *loader) loadPlugin(ctx Context) (err error) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.loadPlugin")) }
    if l.project == nil {
        erro(ctx, "current project is nil").debug(32)
        return
    }

    var g = stat(ctx, "smart.go", l.project)
    if g == nil { return /* smart.go was not presented */ }

    var src = g.string(ctx)
    s := strings.Replace(l.project.relPath, "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath(ctx, "", "")), "plugins", s)

    var build = true

    so := stat(ctx, /*l.project.name*/"plugin", stat_dir{s}, stat_nonexist{true})
    if s = so.fullname(); s == "" {
        erro(at(ctx,so), "file '%v' has empty fullname", so)
        return
    } else if so.exists() && !u.buildPlugins {
        if so.info.ModTime().After(g.info.ModTime()) {
            build = false // Plugin already updated.
        }
    }
    if build { err = buildPlugin(ctx, s, src) }
    if err != nil { return }

    // Once plugin is opened, there's no need/way to close it.
    if l.project.ext.Plugin, err = plugin.Open(s); err == nil {
        var sym plugin.Symbol
        if sym, err = l.project.ext.Lookup("Init"); err != nil {
            erro(ctx, "nil plugin symbol Init").debug()
            return
        }
        if sym == nil {
            return // no initialization (optional)
        }
        switch init := sym.(type) {
        case func(Context) (error):
            if err = init(ctx); err == nil {
                return
            } else {
                erro(ctx, "plugin Init: %v", err).debug()
                return
            }
        default:
            erro(ctx, "wrong plugin Init: %T", sym).debug()
            return
        }
    } else if es := err.Error(); strings.Contains(es, pluginDifferentVersionError) {
        err = buildPlugin(ctx, s, src)
    }
    return
}

func (l *loader) use(ctx Context, usee *project, params []Value, opts useOpts) (err error) {
    var u = _universe(ctx)
    if u.verboseUsing { defer func(t time.Time) {
        var d = time.Now().Sub(t)
        prompt(ctx, "use(%15s) %s ⇒ %v\n", d, l.project, l.project.use)
    } (time.Now())}

    if usee == l.project {
        erro(l, "'%v' use loop (%s)", usee.name, l.usePath())
        return
    } else if l.project.isUsingDirectly(usee) {
        return
    }

    defer func(a []*project) { l.useStack = a } (l.useStack)
    l.useStack = append(l.useStack, usee) // build the use path

    // Add to the project using list, so that the use path is correct.
    if l.project.use.append(ctx, usee, params, opts); !opts.noVars {
        // aka.     XXX += $(use.XXX)
        // aka. use.XXX += $(use.XXX)
        usevars(ctx, l.project, usee)
        if 0 < len(opts.vars) {
            for _, v := range opts.vars {
                warn(at(ctx,v), "var: %T %v", v, v)
            }
            warn(ctx, "TODO: %d vars to import", len(opts.vars)).debug()
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

type includeOpts struct {
    *clauseopts
    ifExists bool `if-exists,ifexists`
    isConfig bool
}
func (l *loader) include(ctx Context, spec Value, opts includeOpts) {
    defer trace(ctx)
    defer flush(ctx)

    var u = _universe(ctx)
    var linfo = u.globe.loads[len(u.globe.loads)-1]

    defer func(t time.Time) {
        if d := time.Now().Sub(t); d > u.slow {
            warn(ctx, "%v: slow: %v (%v)", l.project, d, u.slow).debug() //  → %s, filename
        } else if u.verbose {
            info(ctx, "included %v (%v)", spec, d).debug()
        }
    } (time.Now())

    ctx = at(ctx, spec)

    // Execute the rule entry to update include source.
    if entry, y := spec.(*rule); y && entry != nil {
        if x, y := executeEntry(ctx, entry); !y {
            erro(ctx, "include entry '%v' failed: %s", entry, ts(x)).debug()
            return
        }

        spec = entry.target
    }

    var specName, fullname string

    switch t := spec.(type) {
    case *File:
        if !t.exists() { _ = t.stat(ctx) }
        if !t.exists() && opts.ifExists {
            return // ignore non-exists files
        }
        if fullname, specName = t.fullname(), t.ident(ctx); t.info == nil {
            prompt(ctx, "%v: no source file %v, %v\n", fullname, t, t.stat(ctx))
            erro(at(ctx,t), "%v: %v", get_project(ctx), t).debug()
            return
        }
    default:
        if specName = spec.string(ctx); specName == "" {
            erro(at(ctx,spec), "include: empty string: %v", spec).debug()
            return
        }

        var f = l.project.file(ctx, spec)
        if f == nil {
            if filepath.IsAbs(specName) {
                f = stat(ctx, specName)
            } else {
                f = stat(ctx, specName, stat_dir{linfo.absDir})
            }
        } else if !f.exists() {
            _ = f.stat(ctx)
        }

        if f != nil && f.exists() {
            fullname = f.fullname()
        } else if opts.ifExists {
            return // ignore non-exists files
        }
    }

    if specName == "" {
        erro(ctx, "include: empty string: %v", spec).debug()
        return
    }

    var absDir, baseName = filepath.Split(fullname)
    defer func(m Mode) { l.mode = m } (l.mode) // must restore parse mode!
    defer u.globe.restoreLoadingInfo(u.globe.saveLoadingInfo(l, specName, absDir, baseName))

    ctx = parser_include_context{ctx, opts}

    if _, _, err := l.source(ctx, fullname, nil, parseMode|Flat); err != nil {
        if x, y := err.(*fs.PathError); y && opts.ifExists {
            prompt(ctx, "%v: %v\n", fullname, x.Err)
            warn(ctx, "include %v", spec)
            warnstack(ctx, 5).debug(16)
        } else {
            prompt(ctx, "%v: %v\n", fullname, err)
            erro(ctx, "include %v", spec)
            errostack(ctx, 5).debug(16)
        }
    }
    return
}

func (l *loader) openscope(comment string) (res []*scope) {
    if false && _universe(l.Context).traceLaunch { defer un(l_trace(l_launch, "loader.openscope")) }

    var pos Position
    if l.p == nil {
        pos = l.Position()
    } else {
        pos = l.p.Position()
    }

    s := newscope(pos, l.scope(), l.project, comment)

    if true {
        res, l.s = l.s, append([]*scope{s}, l.s...)
    } else {
        res = l.s
        l.s = append([]*scope{s}, res...)
    }
    return
}

func (l *loader) closescope(scopes []*scope) {
    if false && _universe(l.Context).traceLaunch { defer un(l_trace(l_launch, "loader.closescope")) }

    if true {/* nooooooooooooooooooop */} else
    if scope := l.scope(); scope == nil {
        // nil scope
    } else if s := scope.comment; strings.HasPrefix(s, "dir ") {
        // Change the outer of dir scope to globe to avoid Finding symbols into the wrong context.
        _universe(l.Context).globe.SetScopeOuter(scope)
    }

    l.s = scopes
}

func (l *loader) setArgs(args []Value) (oldArgs []Value) {
    oldArgs = l.loadArgs
    l.loadArgs = args
    return
}

// project example (base(var=value))
func (l *loader) bases(ctx Context, linfo *loadinfo, implicitBase string, params ...Value) (result bool) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.bases")) }

    // For &(foobar) set from loadArgs
    ctx = closure_with(ctx, l.s...)

    var (
        implicitIndex int
        implicitBases []Value
        position = ctx.Position()
    )
    if file := stat(ctx, dotBase, l.project); file != nil {
        if true { var s = file.string(ctx)
            assert(s == file.ident(ctx) && s == dotBase, "invalid finalization: %v => %v", file, s)
        }
        if !file.info.IsDir() && (l.project.spec == dotBase /*|| l.project.spec == dotConfigure*/) {
            // skip the regular file '.base' to avoid self loading recursively
            // info(ctx, "%v", file).debug()
        } else {
            implicitBases = append(implicitBases, file)
        }
    }

    if ns := strings.Split(l.project.name, "."); len(ns) > 2 && ns[len(ns)-1] == "base" {
        var numBaseParams int
        for _, elem := range params {
            if l, y := elem.(*list); y && len(l.elems) == 1 { elem = l.elems[0] }
            if a, y := elem.(*argumented); y { elem = a.Value }
            if _, y := elem.(*pair); y { continue }
            numBaseParams += 1
        }
        if numBaseParams == 0 {
            var segs []Value
            for _, s := range ns[:len(ns)-2] {
                segs = append(segs, makeBareword(position, s))
            }
            implicitBases = append(implicitBases, makePath(segs...))
            if false { warn(ctx, "%v, %v, %v; %v, %v, %v", l.project.name, ns, segs,
                implicitBase, implicitBases, params).debug() }
            implicitBase = "" // discard the implicit base
        } else if false /* && numBaseParams == 1 */ {
            warn(ctx, "%v, %v, %v, %v; %v, %v, %v",
                l.project.name, ns, filepath.Join(ns[:len(ns)-2]...), numBaseParams,
                implicitBase, implicitBases, params)//.debug(6)
        }
    }

    if implicitBase != "" {
        implicitIndex = len(implicitBases)
        implicitBases = append(implicitBases, _pathstr(ctx, implicitBase))
    }

ParamsLoop:
    for i, elem := range append(implicitBases, params...) {
        var (
            elemPos = elem.Position()
            absPath string
            args []Value
            isDir bool
        )
        if list, y := elem.(*list); y && len(list.elems) == 1 { elem = list.elems[0] }
        if a, y := elem.(*argumented); y { elem, args = a.Value, a.args }
        if p, y := elem.(*pair); y {
            var (
                ident = p.key
                position = ident.Position()
                name string
            )
            if name = p.key.string(ctx); len(name) > 0 && name[0] == '.' {
                ident = makeBarecomp(makeBareword(position, "project"), p.key)
            }

            _ = l.p.define_idents(at(ctx, position), ASSIGN, ident, p.val)
            // TODO: check the returning defs...
            continue ParamsLoop
        }

        var (
            specName string
            specVal Value
            implicit bool
        )
        if specVal = elem.expand(final{ctx}); specVal == nil {
            specVal = elem // okay!
        } else if true && indeterminate(ctx, specVal) {
            errostack(at(ctx,elemPos), 5, "incomplete expand: %T %v ⇒ %T %v", elem, elem, specVal, specVal).debug(16)
            return
        } else if defs := specVal.defs(ctx); len(defs) > 0 {
            errostack(at(ctx,elemPos), 5, "incomplete expand: %v ⇒ %v (defs=%v)", elem, specVal, defs).debug(16)
            return
        }

        if specName = specVal.string(ctx); specName == "" {
            erro(at(ctx,elemPos), "%v: empty base name `%v` (%T)", l.project, specVal, specVal).debug()
            break ParamsLoop
        } else if strings.Contains(specName, "//") {
            erro(at(ctx,elemPos), "%v: invalid spec: %v in %T", l.project, elem, ctx)
            erro(at(ctx,elemPos), "%v: invalid spec: %v -> %v", l.project, elem, specVal)
            erro(at(ctx,elemPos), "%v: invalid spec: %v -> %v", l.project, elem, specName).debug(10)
            break ParamsLoop
        } else if implicitBase != "" && specName == implicitBase {
            if i == implicitIndex { implicit = true } else {
                erro(at(ctx,elemPos), "%v: base '%v' already loaded implicitly", l.project, elem).debug()
                if false { break ParamsLoop } else { continue }
            }
        }

        if n := flush(ctx); n > 0 {
            warn(at(ctx,position), "%v: %d errors: %v -> %v", l.project, n, elem, specName).debug()
            break ParamsLoop
        } else if f, y := toFile(elem); y && f.info != nil {
            absPath, isDir = f.fullname(), f.info.IsDir()
            if true { assert(filepath.IsAbs(absPath), "invalid abs path: %v", f) }
        } else if absPath, isDir = u.search(at(ctx,position), linfo, specName); absPath == "" {
            erro(at(ctx,elemPos), "%v: search base failed: %v → %v", l.project, elem, specName).debug()
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
            okay = l.directory(ctx, specName, absPath, nil)
        } else {
            okay = l.load(ctx, specName, absPath, nil)
        }
        l.implicit = implicitSaved // restore implicit flag

        var globe = _universe(l.Context).globe
        if !okay {
            var pos Position
            pos.Filename, pos.Line = absPath, 1
            erro(ctx, "%v: '%s' not loaded'", l.project, specName)
            erro(at(ctx,elemPos), "%v: base '%s' not loaded, %v", l.project, specName, elem)
            erro(at(ctx,position), "%v: base '%s' not loaded, %s", l.project, specName, absPath).debug(6)
            break ParamsLoop
        } else if loaded, y := globe.loaded[absPath]; y && loaded != nil {
            if l.project.hasBase(loaded) { continue ParamsLoop }
            if l.project.bases == nil { // set .base to the first project name
                l.project.projectname(ctx, ".base", loaded)
            }
            l.project.bases = append(l.project.bases, loaded)
        } else if implicit {
            warn(at(ctx,elem), "implicit base '%s' not defined (as %s)", specName, absPath).debug()
        } else {
            erro(at(ctx,elemPos), "project `%v`(%T: %s) not loaded (%s)", elem, elem, specName, absPath).debug()
            break ParamsLoop
        }
    }

    usefor(ctx, l.project, func(op usevar, _, _ Value, name string) {
        var us = "use." + name
        var d, a = l.project.set(ctx, us, defVoid)
        if d == nil && a != nil { d, _ = a.(*def) }
        if d == nil { return }
        op.apply(closure_with(ctx, l.project.scope), d, baseNonTrivialDefs(ctx, l.project, us)...)
    })
    return true
}

func (l *loader) loadDotContainer(ctx Context, ident *barecomp, identStr string, file *File) (result bool) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.loadDotContainer")) }
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug()
        return
    } else if file.info.IsDir() {
        if !l.directory(ctx, dotContainer, file.fullname(), nil) {
            erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug()
            return
        }
    } else if !l.file(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug()
        return
    }

    if loaded, yes := _universe(l.Context).globe.loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.scope().Lookup(loaded.name).(*project)
        if name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.project.name, file).debug()
        } else {
            if false && u.verboseLoads {
                prompt(l, "smart: %v (%v)\n", name, file.fullname())
            }

            var opts useOpts
            // TODO: parse the useOpts
            l.use(ctx, loaded, nil, opts) // see usevars

            result = true
        }
    }
    return
}

func (l *loader) DEPRECATED_loadDotConfigure(ctx Context, ident *barecomp, identStr string, file *File) (result bool) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.loadDotConfigure")) }

    var position = ident.Position()
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug()
        return
    } else if file.info.IsDir() {
        if !l.directory(ctx, dotConfigure, file.fullname(), nil) {
            erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug()
            return
        }
    } else if !l.file(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug()
        return
    }

    if loaded, y := _universe(l.Context).globe.loaded[file.fullname()]; y && loaded != nil {
        if name, _ := l.scope().Lookup(loaded.name).(*project); name == nil {
            if _, alt := l.scope().projectname(at(l, position), loaded.name, loaded); alt != nil {
                if val, ok := alt.(*project); !ok || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug()
                }
            }
        } else {
            if conf := l.project.configure; conf != nil {
                if conf == loaded { return }
                erro(ctx, ".configure already specified").debug()
            }

            l.project.configure, result = loaded, true

            var opts = useOpts{}
            var ctx = at(l, position)
            for _, usee := range loaded.usees(true, false, false, false) {
                if err := l.use(ctx, usee, nil, opts); err != nil { // see usevars
                    erro(ctx, "using '%v' failed: %v", usee, err).debug()
                    break
                }
            }
        }
    }
    return
}

func (l *loader) declare(ctx Context, keyword token, ident *barecomp, identStr string, declOpts *project_opt) (result bool) {
    defer trace(ctx)

    var globe = _universe(l.Context).globe

    if identStr == "@" {
        var linfo = globe.loads[0]
        var x, y = linfo.declares[identStr]
        var at, _ = globe.Lookup(identStr).(*project)
        if !y {
            x = &declare{ project: at }
            linfo.declares[identStr] = x
        }
        x.backscope = l.scope()
        l.useesExecuted = nil
        l.project = at
        l.s[0] = at.scope
        return true
    }

    if _, o := l.scope().find(identStr); o != nil {
        if _, y := o.(*builtin); y {
            erro(ctx, "project name '%s' is a builtin name", identStr)
            return
        }
    }

    var name = identStr
    var linfo = globe.loads[len(globe.loads)-1]
    var dec, declared = linfo.declares[name]
    if !declared {
        var wd = _workdir(l)
        var outer = l.scope()
        var absDir = linfo.absDir
        var relPath, tmpPath string
        if !filepath.IsAbs(absDir) {
            if false {
                absDir = filepath.Join(wd, absDir)
            } else {
                absDir, _ = filepath.Abs(absDir)
            }
        }
        relPath, _ = filepath.Rel(wd, absDir)
        tmpPath = joinTmpPath(ctx, wd, relPath)

        // Avoid nesting project scopes!
        for strings.HasPrefix(outer.comment, "project \"") {
            outer = outer.outer
        }

        dec = &declare{project:globe.project(ctx, outer, absDir, relPath, tmpPath, linfo.specName, name)}
        globe.loaded[linfo.absPath()] = dec.project
        linfo.declares[name] = dec
    }
    if loader := linfo.loader; loader != nil {
        if _, a := loader.projectname(ctx, name, dec.project); a != nil {
            if v, y := a.(*project); !y || v == nil {
                erro(at(ctx,a), "`%s` name already taken (%v).", name, typeof(a)).debug()
                return
            }
        }
    }

    dec.project.opt = *declOpts
    dec.backscope = l.scope()
    l.useesExecuted = nil
    l.project = dec.project
    l.s[0] = dec.project.scope

    if globe.main != nil && globe.main == l.project && l.project.name != "~" {
        for _, t := range globe.pairs {
            switch k := t.key.(type) {
            case *bareword, *barecomp:
                l.scope().set(at(ctx, t), k, defDecl, t.val)
            case flag:
                if false { warn(ctx, "unknown flag : %v", t).debug() }
            default:
                warn(ctx, "unknown target : %v", ts(t)).debug()
            }
        }
    }

    for _, arg := range merge(l.loadArgs...) {
        switch t := arg.(type) {
        case *pair:
            switch k := t.key.(type) {
            case *bareword, *barecomp:
                l.scope().set(at(ctx, t), k, defDecl, t.val)
            case flag:
                if false { warn(ctx, "unknown flag : %v", t).debug() }
            default:
                warn(ctx, "unknown target : %v", ts(t)).debug()
            }
        }
    }

    if err := l.loadPlugin(ctx); err != nil {
        erro(ctx, "load plugin failed: %v", err).debug()
        return
    }
    return true
}

func isConfigureproject(proj *project) bool {
    return proj == nil ||
        proj.name == dotConfigure ||
        proj.name == "configure" ||
        proj.name == "configure.base"
}

func (l *loader) autoload(ctx Context, tag string) {
    if proj := l.project; isConfigureproject(proj) {
        // skip...
    } else if obj := proj.resolve(ctx, ".autoload."+tag); obj == nil {
        // skip...
    } else if d, y := obj.(*def); !y {
        warnstack(ctx, 3, "%v: unsupported .auto: %T %v", proj, obj, obj).debug()
    } else if isTrivial(d.value) {
        // skip...
    } else if val := scalarize(d.value.expand(final{ctx})); isTrivial(val) {
        // skip...
    } else {
        var u = _universe(ctx)
        const ( o = true ; t = false ; s = "autoload" )
        if t && u.ddd == s { prompt(ctx, "%s: autoload - %s,\n", l.Position(), tag) }
        if o { l.include(ctx, val, includeOpts{}) }
        if t && u.ddd == s { prompt(ctx, "%s: autoload - %s.\n", l.Position(), tag) }
    }
}

func (l *loader) configure(ctx Context, linfo *loadinfo, ident *barecomp, identStr string, declared bool) (result bool) {
    if false { defer un(l_tracef(l_traverse, "configuration(%v)", ident)) }
    if s := l.project.name; s == dotConfigure { return }

    var local bool
    var configure string
    var v = l.project.opt.configure
    if v != nil {
        if t, y := v.(*boolean); y { if !t.bool { return } } else
        if !is(v, KindNumber) { configure = v.string(ctx) }
    }
    if local = configure == "."; local || configure == "" { configure = "configure" }

    defer trace(ctx)

    var loaded *project
    var load = func(absPath string, isDir bool) (res bool) {
        if isDir {
            if !l.directory(ctx, configure, absPath, nil) { return }
        } else {
            if !l.file(ctx, absPath, nil) { return }
        }

        var globe = _universe(l.Context).globe
        if loaded, res = globe.loaded[absPath]; loaded == nil { res = false }
        if !res { erro(ctx, "not loaded: %s (%s, dir=%v)", configure, absPath, isDir).debug(16) }
        return
    }

    var isDir bool
    var absPath string
    if filepath.IsAbs(configure) { if file := stat(ctx, configure); file.exists() {
        absPath, isDir = file.fullname(), file.info.IsDir()
    }} else if file := stat(ctx, configure, l.project); file.exists() {
        absPath, isDir = file.fullname(), file.info.IsDir()
    }
    if absPath == "" && v != nil {
        if !local { absPath, isDir = _universe(ctx).search(ctx, linfo, configure) }
        if absPath == "" {
            erro(ctx, "%v: no such project: %s", l.project, configure).debug()
        }
    }
    if absPath == "" { return } else
    if !load(absPath, isDir) {
        erro(ctx, "%v: configure not loaded: %s", l.project, configure).debug()
        return
    }

    if name, _ := l.scope().Lookup(dotConfigure).(*project); name == nil {
        if _, alt := l.scope().projectname(ctx, dotConfigure, loaded); alt != nil {
            if val, y := alt.(*project); !y || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug()
            }
        }
    }
    if l.project.configure == loaded { return }
    if l.project.configure != nil {
        erro(ctx, ".configure already specified").debug()
        return
    }

    var opts = useOpts{}
    for _, usee := range loaded.usees(true, false, false, false) {
        if err := l.use(ctx, usee, nil, opts); err != nil { // see usevars
            erro(ctx, "using '%v' failed: %v", usee, err).debug()
            break
        }
    }

    var u = _universe(ctx)

    // Load configuration.sm after .configure was loaded.
    l.project.configure = loaded // must set .configure first to get the correct configuration file
    l.project.configuration = l.project._configuration(ctx)
    if f := l.project.configuration; f == nil {
        erro(ctx, "%v: nil configuration file", ident).debug()
        return
    } else if declared || u.commandline.configure {
        // u.configuration.clean[f] = struct{}{}
    } else if f.exists() || f.stat(ctx) != nil {
        l.include(ctx, f, includeOpts{isConfig:true})
    }
    return true
}

func (l *loader) container(ctx Context, ident *barecomp, identStr string) (result bool) {
    ctx = at(ctx, ident.Position())
    if l.project.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").debug()
            return
        }

        var u = _universe(ctx)

        // Looking for project specific .container module
        if f := stat(ctx, dotContainer, l.project); f.exists() {
            if !l.loadDotContainer(ctx, ident, identStr, f) {
                //erro(ctx, "declare %s: %s/.container", name, l.project.absPath)
            }
            if u.verbose {
                info(ctx, "%v for %s (%s)\n", f, l.project.spec, l.project).debug()
            }
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(ctx, l.project.absPath, func(s string) bool {
            var f = stat(ctx, dotContainer, stat_dir{filepath.Join(s, ".smart")})
            if !f.exists() {
                // no docking enabled
            } else if !l.loadDotContainer(ctx, ident, identStr, f) {
                //erro(ctx, "%v", err)
            }
            return false
        })

        result = true
    }
    return
}

func (l *loader) closeCurrent(ident *barecomp, identStr string) (err error) {
    var globe = _universe(l.Context).globe
    if identStr == "@" {
        if dec, y := globe.loads[0].declares[identStr]; y {
            l.s[0] = dec.backscope
            l.useesExecuted = dec.useesExecuted
            dec.backscope = nil
            dec.useesExecuted = nil
        }
        return nil
    }

    var linfo = globe.loads[len(globe.loads)-1]
    var dec, y = linfo.declares[identStr]
    if dec == nil || !y {
        return fmt.Errorf("no loaded project %s", identStr)
    }
    if l.project == nil {
        return fmt.Errorf("no current project")
    } else if s := l.project.name; s != identStr {
        return fmt.Errorf("current project is %s but %s", s, identStr)
    } else if l.project != dec.project {
        return fmt.Errorf("project conflicts (%s, %s)", l.project.name, dec.project.name)
    }

    l.s[0] = dec.backscope
    l.useesExecuted = dec.useesExecuted
    return
}

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
//
func readSource(filename string, source ...any) ([]byte, error) {
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

func (l *loader) source(ctx Context, filename string, src any, mode Mode) (f *parsedFile, res []Value, err error) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.source")) }
    if u.verbose { if ctx.Position().Filename == filename {
        info(ctx, "loading ...")
    } else {
        prompt(ctx, "%s:1:info: loading ...\n", filename)
        info(ctx, "loading %v", filename)
    }}

    defer trace(ctx)
    defer func(t time.Time, p *parser, m Mode) { if true { ctx = at(ctx, l.p) }
        if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 10, "%v: slow: %v (%v)", l.project, d, u.slow).debug(2) //  → %s, filename
        } else if u.verbose {
            info(ctx, "loaded %v (%v)", filename, d).debug(2)
        }

        l.p, l.mode = p, m

        if true {
            // ...
        } else if err != nil {
            errostack(ctx, 3, "source error: %v", err).debug()
        } else if f == nil && res == nil {
            erro(ctx, "source not loaded: %s", filepath.Base(filename)).debug()
        }
    } (time.Now(), l.p, l.mode)

    l.mode = mode

    var opts, _ = do(ctx, getParseIncOpts{}).(*includeOpts)
    var text []byte
    if text, err = readSource(filename, src); err != nil {
        if _, y := err.(*fs.PathError); y && opts.ifExists {
        } else {
            prompt(ctx, "%v: %v\n", filename, err)
            erro(ctx, "read source failed: %v (%T)", err, err)
            errostack(ctx, 5, "").debug(32)
        }
        return
    }

    l.p = &parser{Context:ctx}
    ctx = l.p // switch context

	var smod scanmode
	if l.mode&ParseComments != 0 {
		// smod = scanner.ScanComments
	}

    l.p.scanner.init(u.file(filename, text), text, smod,
        func(p Position, s string, a ...any) {
            if a == nil { a = append(a, 4, 4) }
            note(at(ctx,p), "%s", s)
            erro(at(ctx,p), "scan=%v", l.p.scanner.scanstate).debug(a...)
        },
        func(p Position, s string, a ...any) {
            if a == nil { a = append(a, 4, 1) }
            warn(at(ctx,p), "%s", s)
            warn(at(ctx,p), "scan=%v", l.p.scanner.scanstate).debug(a...)
        },
        func(p Position, s string, a ...any) {
            if a == nil { a = append(a, 4, 1) }
            info(at(ctx,p), "%s", s)
            info(at(ctx,p), "scan=%v", l.p.scanner.scanstate).debug(a...)
        })

	l.p.next(ctx, true) // starts scanning
    ctx = at(ctx, l.p)

    if l.mode&parsingText != 0 {
        res = l.p.values(ctx)
    } else if f = l.p.file(ctx); f == nil {
        // Source is not a valid source file, returnning a valid but empty parsedFile
        defer l.closescope(l.openscope(fmt.Sprintf("file %s", filename)))
        f = &parsedFile{ scope:l.scope() }
        f.position.Filename = filename
        // TODO: validate basename as a valid identifier
        f.name = makeBarecomp(makeBareword(f.position, filepath.Base(filepath.Dir(filename))))
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

    var fs []os.FileInfo
    if fs, err = fd.Readdir(-1); err != nil || len(fs) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        err = fmt.Errorf("invalid package name %s", ident)
        return
    }

    defer l.closescope(l.openscope("config "+pathname))

    var ctx = at(l, l.p)
    var scope = l.scope()

    for _, f := range fs {
        var name = f.Name()
        if hasPrefix(name, "~") || hasSuffix(name, ".#", ".smart", ".sm") {
            continue
        }

        var fullname = filepath.Join(linked, name)
        if f.Mode()&os.ModeSymlink != 0 {
            var ( l string; t os.FileInfo )
            if l, err = os.Readlink(fullname); err != nil { continue }
            if !filepath.IsAbs(l) { l = filepath.Join(linked, l) }
            if t, err = os.Stat(l); err != nil { continue }
            if t.IsDir() { continue }
        }

        if f.IsDir() {
            if err = l.ParseConfigDir(filepath.Join(pathname, name), fullname); err != nil {
                erro(ctx, "parse config failed: %v", err).debug()
                break
            }
            if 0 < flush(ctx) { return } else { continue }
        }

        d, a := scope.set(ctx, name, defConfDir)

        if a != nil && a != d {
            erro(ctx, "declare project: %v", name).debug()
            break
        } else if d == nil {
            erro(ctx, "%v", name).debug()
            return
        }

        var v []byte
        if v, err = ioutil.ReadFile(fullname); err != nil {
            erro(ctx, "%v", err).debug()
            break
        }

        var s = string(v)
        if !utf8.ValidString(s) {
            erro(ctx, "%s: invalid UTF8 content", fullname)
            break
        }

        d.set(ctx, defConfDir, makeStrlit(ctx.Position(), s))
    }
    return
}

var loader_sources_bench = true

func (l *loader) sources(ctx Context, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*project) {
    var u = _universe(l.Context)

    defer trace(ctx)

    if loader_sources_bench {
        defer func(t time.Time) {
            if d := time.Now().Sub(t); u.verboseParse || d > time.Second {
                note(ctx, "slow: %s (%v)", l.project, d).debug()
            } else if debugSyntax(ctx, "sources") {
                note(ctx, "sources: %s (%v)", l.project, time.Now().Sub(t)).debug(6)
            }
        }(time.Now())
    }

    var fd, err = os.Open(path)
    if err != nil {
        erro(ctx, "%v", err).debug()
        return
    }
    defer fd.Close()

    fis, err := fd.Readdir(-1)
    if err != nil {
        erro(ctx, "readdir: %v", err).debug()
        return
    } else if len(fis) == 0 {
        erro(ctx, "no files underneath: %s", path).debug()
        return
    }

    first := fis[0]
    for i, a := range fis { if i > 0 { s := a.Name()
        if s == entryFileName || (s == "build.smart" && first.Name() != entryFileName) {
            fis[0] = a
            fis[i] = first
        }
    }}

    defer l.closescope(l.openscope("dir "+path))

    // FIXES: use 'globe' scope as outer to avoid chaining scopes to other unrelated
    // projects which are in consequence load order. Setting dir scope outer to such
    // project scopes will cause resolving objects to the wrong ones.
    l.scope().outer = u.globe.scope

    // if  l.scope().position.Filename == "" {
    //     l.scope().position.Filename = path
    //     l.scope().position.Line = 1
    // }

    mods = make(map[string]*project)

ListLoop:
    for _, d := range fis {
        var (
            name, mo = d.Name(), d.Mode()
            linked, linkPath = "", path
            filename = filepath.Join(path, name)
        )
        if name == "" || name == configuration_sm || strings.HasPrefix(name, ".#") ||
            !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm")) { continue ListLoop }

        for fn := filename; mo&os.ModeSymlink != 0; { if s, e := os.Readlink(fn); e != nil {
            prompt(ctx, "%s: readlink failed\n", fn)
            warn(ctx, "%v", e)
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
        }}

        if false { if (name == "configure.smart" || name == "configure.sm") && (linked != "" || mo.IsDir()) {
            // hasConfDir = true // TODO: remove ConfigDir feature
            if l.ParseConfigDir(filepath.Dir(filename), linked) != nil { return }
            continue ListLoop
        }}

        if linked != "" { }

        if mo.IsRegular() && (filter == nil || filter(d)) {
            var pos Position
            pos.Filename, pos.Line = filename, 1

            var src, _, err = l.source(ctx, filename, nil, mode|parsingDir)
            if err != nil { erro(ctx, "parse failed: %v", err) }

            var d *diagPoint
            if flush(ctx) > 0 {
                e := _diagnostic(ctx).countError()
                s := filepath.Base(filename)
                d = erro(ctx, "got %d errors in file '%s'", e, s)
            } else if err == nil {
                if src == nil {
                    d = erro(ctx, "parsed nil module from")
                } else if isNull(src.name) {
                    d = erro(ctx, "parsed module name is <nil>")
                } else if isNone(src.name) {
                    d = erro(ctx, "parsed module name is <none>")
                }
            }
            if d != nil {
                if l.p != nil && l.p.scanner.file != nil {
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

            var name = src.name.string(ctx)
            if mod, found := mods[name]; !found {
                mod = &project{ name:name, scope:l.scope() }
                mods[name] = mod
            }
        }
    }
    return
}

// loader.Load loads script from a file or source code (string, []byte).
func (l *loader) load(ctx Context, specName, absPath string, source any) (result bool) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.load")) }

    defer trace(ctx)
    defer flush(ctx)

    defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseLoads && d>1*time.Second {
            loaded, _ := u.globe.loaded[absPath]
            if l.project == nil {
                prompt(ctx, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                prompt(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName)
            }
        }
    } (time.Now())

    if absPath == "" {
        erro(ctx, "no such module `%s' (in paths %v)", specName, u.paths)
        return
    } else if !filepath.IsAbs(absPath) {
        erro(ctx, "invalid abs name `%s' (%s)", absPath, specName)
        return
    }

    // Check loaded project.
    if loaded, yes := u.globe.loaded[absPath]; yes {
        if _, a := l.scope().projectname(at(l, l.Position()), loaded.name, loaded); a != nil {
            if val, ok := a.(*project); !ok || val == nil {
                erro(ctx, "`%v` name already taken (%T).", loaded, a)
            }
        }
        result = true
        return
    }

    var absDir, baseName = filepath.Split(absPath)
    defer u.globe.restoreLoadingInfo(u.globe.saveLoadingInfo(l, specName, absDir, baseName))

    var doc, _, err = l.source(ctx, absPath, source, parseMode)
    if err != nil {
        erro(ctx, "load: %v", err).debug()
    } else if doc == nil {
        erro(ctx, "load: nil: %s", absPath).debug()
    } else {
        result = true
    }
    return
}

func (l *loader) directory(ctx Context, specName, absDir string, filter func(os.FileInfo) bool) (okay bool) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.directory")) }

	defer trace(ctx)

    if !filepath.IsAbs(absDir) {
        errostack(ctx, 3, "needs absolute dir `%s' (%s)", absDir, specName).debug(10)
        return
    }

    var pos Position = ctx.Position()
    if !pos.IsValid() { pos = positionForDir(absDir) }

    var loadedProj *project
    defer func(t time.Time, ver bool) {
        if specName == "." { specName = absDir }

        if d := time.Now().Sub(t); ver && d>1*time.Second { if l.project == nil {
            note(ctx, "load (%15s) ⇒ %s (%s)\n", d, loadedProj, specName).debug()
        } else {
            note(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loadedProj, specName).debug()
        }}

        if loadedProj == nil { return }
        if u.globe.main == nil { u.globe.main = loadedProj }

        if proj := l.scope().project; proj == nil {
            if false { erro(ctx, "%v: no owner project for %s", loadedProj.name, l.scope()).debug(2) }
        } else if name, _ := proj.Lookup(loadedProj.name).(*project); name == nil {
            if _, alt := proj.projectname(at(ctx,pos), loadedProj.name, loadedProj); alt != nil {
                if val, y := alt.(*project); !y || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loadedProj.name, alt).debug(2)
                }
            }
        }
    } (time.Now(), u.verboseLoads)

    // Check loaded project.
    if loadedProj, okay = u.globe.loaded[absDir]; okay { return }

    defer u.globe.restoreLoadingInfo(u.globe.saveLoadingInfo(l, specName, absDir, ""))

    var mods map[string]*project
    if mods = l.sources(at(ctx, pos), absDir, filter, parseMode); mods == nil {
        errostack(ctx, 3, "failed parsing module: %s", specName).debug(12)
        return
    }

    // FIXME: loading failed if different 'project' found in the same dir, for example:
    //      project Foo # file do.smart
    //      project # file config.smart
    if len(mods) == 0 && filepath.Base(specName) != "@" {
        if l.implicit {
            warn(l, "%s not loaded (as %s, implicitly)", specName, absDir).debug(10)
            okay = true // okay for implicit loading
        } else {
            for s, m := range u.globe.loaded { erro(ctx, "%v: %v", s, m) }
            errostack(ctx, 3, "%s not loaded (as %s)", specName, absDir).debug(10)
        }
    } else if loadedProj, okay = u.globe.loaded[absDir]; okay && loadedProj != nil {
        // Good!
    } else if filepath.Base(specName) != "@" {
        erro(ctx, "%s not loaded (as %s, implicit=%v)", specName, absDir, l.implicit).debug()
    }
    return
}

func (l *loader) file(ctx Context, filename string, source any) (res bool) {
    if _universe(ctx).traceLaunch { defer un(l_trace(l_launch, "loader.file")) }

    var spec string
    switch dir, base := filepath.Split(filename); base {
    case dotBase, dotConfigure: spec = base
    default: spec, _  = filepath.Rel(_workdir(l), dir)
    }

    var position Position
    position.Filename = filename
    return l.load(at(ctx, position), spec, filename, source)
}

func (l *loader) path(ctx Context, path string, filter func(os.FileInfo) bool) bool {
    if _universe(ctx).traceLaunch { defer un(l_trace(l_launch, "loader.path")) }

    var spec, _ = filepath.Rel(_workdir(ctx), path)

    var position Position
    position.Filename = spec
    return l.directory(at(ctx, position), spec, path, filter)
}

func (l *loader) text(ctx Context, filename string, text string) (res []Value) {
    if _universe(ctx).traceLaunch { defer un(l_trace(l_launch, "loader.text")) }

    defer func(saved *parser) { l.p = saved } (l.p)

    if g := _universe(l.Context).globe; g.main == nil {
        l.s[0] = g.os.scope
    } else {
        l.s[0] = g.main.scope
    }
    l.useesExecuted = nil

    var err error
    var position Position
    position.Filename = filename

    ctx = at(ctx, position)

    if _, res, err = l.source(ctx, filename, text, parsingText); err != nil {
        prompt(ctx, "%v: %v\n", filename, err)
        erro(ctx, "load text failed: %v", err)
        errostack(ctx, 5, "").debug(32)
    }
    return
}
