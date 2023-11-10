//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package smart

import (
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

const (
    dotBase = ".base"
    dotContainer = ".container"
    dotConfigure = ".configure"

    entryFileName = "do.smart"

    optSortErrors = false
)

func isEntryFileName(s string) bool { return filepath.Base(s) == entryFileName }

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

    // Wants v.string(ctx), expands delegates and closures,
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
    closurecontext
    p *parser
    mode Mode // parsing mode
    project *Project
    loadArgs []Value
    loadStack []*Project // load path
    useStack  []*Project // use path
    useesExecuted []*Project // all executed usees
    implicit bool // loading current project implicitly, aka. via foo.bar.Baz (implicit foo/bar loaded)
    verpre string // verbose prefix
}
func (l *loader) String() string {
    if fullContextStringer {
        return fmt.Sprintf("loader{%s}", &l.closurecontext)
    } else {
        return l.closurecontext.String()
    }
}
func (l *loader) loader() *loader { return l }
func (l *loader) parser() *parser { return l.p }
func (l *loader) inner() Context { return &l.closurecontext }
func (l *loader) Project() (project *Project) { return l.project }

func restoreLoadingInfo(l *loader) {
    var (
        globe = l.Globe()
        last = len(globe.loads)-1
        linfo = globe.loads[last]
    )
    globe.loads = globe.loads[0:last]
    l.useesExecuted = linfo.useesExecuted
    l.project = linfo.loader
    l.scopes = linfo.scopes //l.SetScope(linfo.scope)

    //assert(l.scope.project == linfo.loader, "scope/project mismatched")

    /*var names []string
    for _, declare := range linfo.declares {
        names = append(names, declare.project.Name())
    }

    if loader := linfo.loader; loader != nil {
        prompt(ctx, "exit: %v from '%s' → %v\n", names, loader.Name(), linfo.scope)
    } else {
        prompt(ctx, "exit: %v → %v\n", names, linfo.scope)
    } */
}

func saveLoadingInfo(l *loader, specName, absDir, baseName string) *loader {
    var globe = l.Globe()
    globe.loads = append(globe.loads, &loadinfo{
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
    for _, u := range u { for _, v := range umerge(true, u.value) {
        if t, y := v.(*def); y && t != nil {
            vals = append(vals, umerge(true, t.value)...)
        } else {
            vals = append(vals, v)
        }
    }}
    if len(vals) == 0 { return } else
    if d.append(ctx, vals...); uo.unique { a := umerge(true, d.value)
        d.value = call(ctx, "unique", plain, uo.remainder, a...)
    }
}
func usefor(ctx Context, user *Project, f func(usevar, Value, Value, string)) {
    defer dtrace(ctx, "use")

    var o = user.resolve(ctx, "use.*")
    if o != nil { if d, y := o.(*def); y && d != nil { for _, spec := range umerge(true, d.value) {
        var ( val = spec ; name string ; op usevar ; ctx = of(ctx, spec) )
        if a, y := spec.(*argumented); y { val = a.Value
            op.remainder = parseOpts(ctx, &op, strval, a.args...)
        }
        if name = val.string(ctx); name == "" { c := user.configure
            if c != nil { t := c.resolve(ctx, "use.*")
                noted(ctx, "%T %v", t, t)
            }
            erro(of(ctx,val), "%v: empty use spec: '%v' (%T)", user, spec, spec).debug(1)
        } else {
            f(op, spec, val, name)
        }
    }}}
}
func usevars(ctx Context, user, usee *Project) {
    var ddd = cast[*universe](ctx).ddd == "use"
    usefor(ctx, user, func(op usevar, spec, val Value, name string) {
        var useDef *def
        if o := usee.scope.Lookup("use."+name); o != nil {
            if d, y := o.(*def); y && d != nil { useDef = d } else {
                erro(ctx, "use.%s: nil def: %T %v", name, o, o).debug(3)
                return
            }
        }
        if useDef == nil { return }

        var dd []*def

        // 1. use.XXX += $(use.XXX)
        {
            var d, a = user.scope.define(ctx, DefVoid, useDef.name(ctx), nil)
            var isNewDef = d != nil && a == nil
            if d == nil && a != nil { d, _ = a.(*def) }
            if d == nil { return }
            if d.value != nil && d.value.String() == "unique" {
                noted(ctx, "%v (%v, %v, %v)", d, user, isNewDef, a)
                noted(ctx, "%v", useDef).debug(10)
            }
            if isNewDef || isTrivial(d.value) {
                dd = append(dd, baseNonTrivialDefs(ctx, user, useDef.name(ctx))...)
            }
            op.apply(closureWith(ctx, usee.scope), d, append(dd, useDef)...)
        }

        if useDef.value == nil || isTrivial(useDef.value) { return }

        // 2. XXX += $(use.XXX)
        {
            var d, a = user.scope.define(ctx, DefVoid, name, nil)
            var isNewDef = d != nil && a == nil
            if d == nil && a != nil { d, _ = a.(*def) }
            if d == nil { return }
            if isNewDef && false {
                if dd == nil { dd = append(dd, baseNonTrivialDefs(ctx, user, useDef.name(ctx))...) }
                dd = append(dd, baseNonTrivialDefs(ctx, user, name)...)
            }
            op.apply(closureWith(ctx, user.scope), d, append(dd, useDef)...)
        }
    })
    if ddd { noted(ctx, "%v ⇒ %v ; %v", user, usee, user.resolve(ctx, "use.*")).debug(5) }
}
func baseNonTrivialDefs(ctx Context, user *Project, name string) (dd []*def) {
    for _, base := range user.bases { if o := base.resolve(ctx, name); o != nil {
        if t, y := o.(*def); y && !isTrivial(t.value) { dd = append(dd, t) }
    }}
    return
}

func (l *loader) Position() (res Position) {
    if l.p != nil { return l.p.Position() }
    return l.Context.Position()
}

func (l *loader) usespec(ctx Context, opts useOpts, specVal Value, arged []Value, params ...Value) (loaded *Project) {
	if true { defer dtrace(ctx, "usespec") }

    var (
        u = cast[*universe](l.Context)
        globe = l.Globe()
        linfo = globe.loads[len(globe.loads)-1]
        absPath, specName string
        isDir, traveUseLoop bool
        err error
    )
    if n, y := specVal.(*projectname); y {
        if false { warnstack(ctx, 3, "use project: %v %s", n, n.spec).debug(6) }
        loaded = n.Project
    } else if specName = specVal.string(ctx); specName == "" {
        errostack(ctx, 3, "empty spec: %v (%T)", specVal, specVal).debug(6)
        return
    } else if absPath, isDir = u.search(ctx, linfo, specName); absPath == "" {
        errostack(ctx, 3, "missing `%s` (in %v)", specName, u.paths).debug(6)
        return
    } else {
        if loaded, y = globe.loaded[absPath]; !y {
            if false { warnstack(ctx, 3, "not project: %s", absPath).debug(6) }
        }
        // Checking circular loads. See also Project.loopImportPath()!
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
                if loaded != nil && loaded.opts.traveUseLoop {
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

    var scope = l.Scope()
    if false && traveUseLoop {
        if loaded == nil {
            // ...
        } else if _, a := scope.projectname(ctx, loaded.name, loaded); a != nil {
            if val, ok := a.(*projectname); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, a).debug(1)
            }
        }
        return
    }

    defer func(a []*Project) { if l.loadStack = a; loaded == nil {
        erro(ctx, "%v not loaded (%v,dir=%v)", specName, absPath, isDir).debug(1)
        return
    } else if name, _ := scope.Lookup(loaded.name).(*projectname); name == nil {
        if _, alt := scope.projectname(ctx, loaded.name, loaded); alt != nil {
            if val, ok := alt.(*projectname); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
            }
        }
        if false {
            erro(ctx, "%v (%v,dir=%v)", specName, absPath, isDir).debug(1)
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
            okay = l.dir(ctx, specName, absPath, nil)
        } else {
            okay = l.load(ctx, specName, absPath, nil)
        }
        if !okay {
            erro(ctx, "failed loading `%v` (%v)", specName, absPath).debug(1)
            return
        }

        if loaded != nil {
            // already loaded previously
        } else if loaded, _ = globe.loaded[absPath]; loaded != nil {
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
                errostack(ctx, 10, "%v: using `%s` multiple times: %v", l.project, specName, l.project.use.list).debug(10)
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

    if u.verboseImport { defer func(t time.Time) {
        prompt(ctx, "%s├┤ %s:import(%s) (%s)\n", l.verpre, l.project, specName, time.Now().Sub(t))
    } (time.Now()) } //*time.Millisecond // µs, ms, s ┼

    if err = l.use(ctx, loaded, params, opts); err != nil { // see usevars
        erro(ctx, "using '%v' failed: %v", loaded, err).debug(1)
        return
    }
    return
}

const pluginDifferentVersionError = `plugin was built with a different version of package`
var numUpdatedPlugins = 0

func buildPlugin(ctx Context, s, src string) (err error) {
    var u = cast[*universe](ctx)
    if u.traceLaunch { defer un(trace(t_launch, "loader.buildPlugin")) }

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
    var u = cast[*universe](ctx)
    if u.traceLaunch { defer un(trace(t_launch, "loader.loadPlugin")) }
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
        erro(of(ctx,so), "file '%v' has empty fullname", so)
        return
    } else if so.exists() && !u.buildPlugins {
        if so.info.ModTime().After(g.info.ModTime()) {
            build = false // Plugin already updated.
        }
    }
    if build { err = buildPlugin(ctx, s, src) }
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
        err = buildPlugin(ctx, s, src)
    }
    return
}

func (l *loader) use(ctx Context, usee *Project, params []Value, opts useOpts) (err error) {
    var u = cast[*universe](ctx)
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

    defer func(a []*Project) { l.useStack = a } (l.useStack)
    l.useStack = append(l.useStack, usee) // build the use path

    // Add to the project using list, so that the use path is correct.
    if l.project.use.append(ctx, usee, params, opts); !opts.noVars {
        // aka.     XXX += $(use.XXX)
        // aka. use.XXX += $(use.XXX)
        usevars(ctx, l.project, usee)
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
    for i, elem := range elems { if a, y := elem.(*argumented); y {
        var prefix, suffix = elems[:i], elems[i+1:]
        iterateArgumentedIdentifiers(ctx, a, func(ident Value, stems2 []Value) {
            var head   = append(prefix, ident)
            var stems3 = append(stems , stems2...)
            iterateArgumentedIdentElems(ctx, suffix, stems3, func(elems, stems []Value) {
                f(append(head, elems...), stems)
            })
        })
        return
    }}
    f(elems, stems)
}

func iterateArgumentedIdentifiers(ctx Context, identifier Value, f func(ident Value, stem []Value)) {
    switch t := identifier.(type) {
    case *argumented:
        var args = xmerge(ctx, plain, t.args...)
        iterateArgumentedIdentifiers(ctx, t.Value, func(ident Value, stems []Value) {
            var pos = ident.Position()
            for _, arg := range args {
                if isTrivial(arg) { continue }
                f(makeBarecomp(pos, ident, arg), append(stems, arg))
            }
        })
    case *barecomp:
        iterateArgumentedIdentElems(ctx, t.Elems, nil, func(elems, stems []Value) {
            if len(stems) == 0 { f(t, stems) } else {
                f(makeBarecomp(t.Position(), elems...), stems)
            }
        })
    default:
        f(t, nil)
    }
}

func (l *loader) define(ctx Context, tok Token, identifier, value Value) (defs []*def) {
    iterateArgumentedIdentifiers(ctx, identifier, func(ident Value, stems []Value) {
        if d := l.define1(ctx, tok, ident, value); d != nil { defs = append(defs, d) }
    })
    return
}

func (l *loader) define1(ctx Context, tok Token, identifier, value Value) (d *def) {
    var alt Object
    switch t := identifier.(type) {
    case *argumented:
        var args = xmerge(ctx, plain, t.args...)
        erro(ctx, "TODO: multiple defs: %v args=%v", t.Value, args)
        return

    case *group:
        erro(ctx, "TODO: multiple defs: %v", t.Elems)
        return

    case *selection:
        if v := t.value(ctx, ident); v == nil {
            erro(ctx, "nil selection: %v", t).debug(1)
            return
        } else if a, y := v.(*def); y { d = a } else {
            erro(ctx, "`%v` is not a def (%T)", t, v).debug(1)
            return
        }

    default: //case *bareword, *barecomp, *Qualiword, *Path, flag:
        var name = t.string(ctx)
        if _, y := builtins[name]; y {
            erro(ctx, "`%v` (%v) is builtin name", identifier, name)
            return
        }

        // Resolve base value to derive.
        var prev = l.project.resolve(ctx, name)

        if d, alt = l.def(t.Position(), name); alt == nil {
            if d == nil {
                erro(ctx, "`%s` is undefined (%v %v)", name, typeof(t), t).debug(1)
                return
            }
        } else if tok == ASSIGN || tok == EXC_ASSIGN {
            if a, y := alt.(*def); !y {
                erro(ctx, "`%v` already defined (%T) (%v,%v)", identifier, alt, alt.OwnerProject(), l.project).debug(1)
                return
            } else if a.owner == l.project && a.origin != DefConfRef {
                erro(ctx, "`%v` already defined (%T) (%v)", identifier, alt, l.project).debug(1)
                return
            } else {
                d = a
            }
        } else {
            d = alt.(*def)
        }

        if prev == nil {
            // no derived value
        } else if prev.OwnerProject() == l.project {
            // not derivable def if they are from the same project
        } else if derived, y := prev.(*def); !y {
            // not a def
        } else if derived == nil {
            erro(ctx, "prev def '%s' is nil", name).debug(1)
        } else if derived == d || (d.value != nil && d.value.refs(ctx, derived)) {
            // same def
        } else if d != nil && (tok == ADD_ASSIGN || tok == SHI_ASSIGN) && alt == nil {
            if d.origin == DefVoid { d.origin = derived.origin }
            if !isTrivial(derived.value) { d.append(ctx, derived.value) }
        }
    }

    if d == nil {
        erro(ctx, "def is nil: %v %v", typeof(identifier), identifier).debug(1)
        return
    }

    switch d.position = identifier.Position(); tok {
    case     ASSIGN: d.set(ctx, DefDefault, value) //   =
    case CO1_ASSIGN: d.set(ctx, DefExpand1, value, expandDefAssign) //  :=
    case CO2_ASSIGN: d.set(ctx, DefExpand2, value, expandDefAssign) // ::=
    case EXC_ASSIGN: d.set(ctx, DefExecute, value, expandDefAssign) //  !=
    case QUE_ASSIGN: if alt == nil { d.set(ctx, DefDefault, value, expandDefAssign) } // ?=
    case ADD_ASSIGN: if !isTrivial(value) { // +=
        var ii []interface{}
        if !isTrivial(d.value) { ii = vi(umerge(true, d.value)...) }
        if !isTrivial(  value) { ii = vi(umerge(true,   value)...) }
        d.set(ctx, d.origin, nil, append(ii, expandDefAssign)...)
    }
    case SHI_ASSIGN: if !isTrivial(value) { // =+
        var ii []interface{}
        if !isTrivial(d.value) { ii = vi(umerge(true, d.value)...) }
        d.set(ctx, d.origin, value, append(ii, expandDefAssign)...)
    }
    case SUB_ASSIGN: if d.value != nil { if dv := merge(d.value); len(dv) > 0 { // -=
        var vals []Value
        var sub = merge(value)
    outer1:
        for _, v := range dv {
            for _, sv := range sub { if v.cmp(ctx, sv) == cmpEqual { continue outer1 }}
            vals = append(vals, v)
        }
        d.value = ease(ctx, vals)
    }}
    case SAD_ASSIGN, SSH_ASSIGN: // -+=, -=+
        var vals []Value
        var sub = merge(value)
        if d.value != nil { if dv := merge(d.value); len(dv) > 0 {
        outer2:
            for _, v := range dv {
                for _, sv := range sub { if v.cmp(ctx, sv) == cmpEqual { continue outer2 }}
                vals = append(vals, v)
            }
        }}
        if SAD_ASSIGN == tok {
            vals = append(vals, sub...) // -+=
        } else {
            vals = append(sub, vals...) // -=+
        }
        d.value = ease(ctx, vals)
    default:
        erro(ctx, "unknown origin: %v %v %v", d.origin, d.name, tok).debug(1)
    }
    return
}

func (l *loader) rule(clause *parsedRuleData) (entries []Entry) {
    var (
        ctx = at(l, l.Position())
        params []*auto
        depends []Value
        ordered []Value
        progScope *Scope = l.Scope()
        configure = clause.config
        recipes = clause.recipes
    )
    for _, name := range clause.params { o := progScope.Lookup(name)
        if a, y := o.(*auto); y { params = append(params, a) } else {
            errostack(ctx, 3, "invalid param: %T %v", o, o).debug(5)
        }
    }
    for _, depend := range clause.depends {
        switch dep := depend.(type) {
        case *list: depends = append(depends, dep.Elems...)
        default:    depends = append(depends, dep)
        }
    }
    for _, depend := range clause.ordered {
        switch dep := depend.(type) {
        case *list: ordered = append(ordered, dep.Elems...)
        default:    ordered = append(ordered, dep)
        }
    }

    var prog = &program{
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

        var ctx = at(ctx, target.Position())
        var entry, err = l.project.entry(ctx, clause.special, clause.options, target, prog)
        if err != nil {
            erro(of(ctx,target), "creating entry '%v' failed: %v", target, err)
            return
        } else {
            entries = append(entries, entry)
        }

        if t, okay := entry.Target().(flag); okay && t.Value != nil {
            var s = t.Value.string(ctx)
            if l.project.name != "~" { l.Globe().AddFlagEntry(s, entry) }
        } else if configure {
            if entry.Class() == PatternRule {
                erro(ctx, "unsupported pattern configures: %v", target).debug(1)
                return
            } else {
                l.project.configs = append(l.project.configs, entry)
            }
        }
    }
    return
}

type includeOpts struct {
    *clauseOpts
    ifExists bool `if-exists,ifexists`
    isConfigure bool // internal
}
func (l *loader) include(ctx Context, opts includeOpts, spec Value) {
    defer dtrace(ctx, "include")

    var (
        u = cast[*universe](ctx)
        globe = l.Globe()
        linfo = globe.loads[len(globe.loads)-1]
        specName, fullname string
        err error
    )

    defer func(t time.Time) {
        if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 10, "%v: slow: %v (%v)", l.project, d, u.slow).debug(1) //  → %s, filename
        } else if u.verbose {
            info(ctx, "included %v (%v)", spec, d).debug(1)
        }
        if err != nil { erro(ctx, "parse file failed: %v", err).debug(2) }
    } (time.Now())

    ctx = at(ctx, spec.Position())

    // Execute the rule entry to update include source.
    if entry, ok := spec.(*rule); ok && entry != nil {
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
        if fullname, specName = t.fullname(), t.name(ctx); t.info == nil {
            prompt(ctx, "%v: no source file %v, %v\n", fullname, t, t.stat(ctx))
            erro(of(ctx,t), "%v: %v", ctx.Project(), t)
            errostack(ctx, 5, "").debug(16)
            return
        }
    default:
        if specName = spec.string(ctx); specName == "" {
            erro(of(ctx,spec), "include: empty string: %v", spec)
            errostack(ctx, 5, "").debug(16)
            return
        }

        var file = l.project.file(ctx, spec)
        if file == nil {
            if filepath.IsAbs(specName) {
                file = stat(ctx, specName)
            } else {
                file = stat(ctx, specName, stat_dir{linfo.absDir})
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
    if _, _, err = l.source(ctx, fullname, nil, parseMode|Flat, &opts); err != nil {
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

    if n := ctx.dia().flush(); n > 0 { warn(ctx, "got %d errors", n).debug(1)
        if u.failOnErrors { total := ctx.dia().totalErrors()
            panic(failure{"fail by %d errors",ia(ctx.Position(), total)})
        }
    }
    return
}
func (l *loader) closureScopes() (scopes []*Scope) {
    scopes = append(l.closurecontext.closureScopes(), l.Scope())
    return
}

func (l *loader) openScope(comment string) (scopes []*Scope) {
    var u = cast[*universe](l.Context)
    if false && u.traceLaunch { defer un(trace(t_launch, "loader.openScope")) }
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
    var u = cast[*universe](l.Context)
    if false && u.traceLaunch { defer un(trace(t_launch, "loader.closeScope")) }
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
func (l *loader) bases(ctx Context, linfo *loadinfo, implicitBase string, params ...Value) (result bool) {
    var u = cast[*universe](ctx)
    if u.traceLaunch { defer un(trace(t_launch, "loader.bases")) }

    // For &(foobar) set from loadArgs
    ctx = closureWith(ctx, l.scopes...)

    var (
        implicitIndex int
        implicitBases []Value
        position = ctx.Position()
    )
    if file := stat(ctx, dotBase, l.project); file != nil {
        if true { var s = file.string(ctx)
            assert(s == file.name(ctx) && s == dotBase, "invalid strval: %v => %v", file, s)
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
            if l, y := elem.(*list); y && len(l.Elems) == 1 { elem = l.Elems[0] }
            if a, y := elem.(*argumented); y { elem = a.Value }
            if _, y := elem.(*pair); y { continue }
            numBaseParams += 1
        }
        if numBaseParams == 0 {
            var segs []Value
            for _, s := range ns[:len(ns)-2] {
                segs = append(segs, makeBareword(position, s))
            }
            implicitBases = append(implicitBases, makePath(position, segs...))
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
        implicitBases = append(implicitBases, pathStr(ctx, position, implicitBase))
    }

ParamsLoop:
    for i, elem := range append(implicitBases, params...) {
        var (
            elemPos = elem.Position()
            absPath string
            args []Value
            isDir bool
        )
        if list, y := elem.(*list); y && len(list.Elems) == 1 { elem = list.Elems[0] }
        if a, y := elem.(*argumented); y { elem, args = a.Value, a.args }
        if p, y := elem.(*pair); y {
            var (
                identifier = p.Key
                position = identifier.Position()
                name string
            )
            if name = p.Key.string(ctx); len(name) > 0 && name[0] == '.' {
                identifier = makeBarecomp(position, makeBareword(position, "project"), p.Key)
            }

            var defs = l.define(at(ctx, position), ASSIGN, identifier, p.Value)
            if len(defs) == 0 {/* TODO: check defs... */}
            continue ParamsLoop
        }

        var (
            specName string
            specVal Value
            implicit bool
        )
        if specVal = elem.expand(ctx, strval); specVal == nil {
            specVal = elem // okay!
        } else if true && specVal.expandable(ctx, strval) {
            errostack(at(ctx,elemPos), 5, "incomplete expand: %T %v ⇒ %T %v", elem, elem, specVal, specVal).debug(16)
            return
        } else if defs := specVal.defs(ctx); len(defs) > 0 {
            errostack(at(ctx,elemPos), 5, "incomplete expand: %v ⇒ %v (defs=%v)", elem, specVal, defs).debug(16)
            return
        }

        if specName = specVal.string(ctx); specName == "" {
            erro(at(ctx,elemPos), "%v: empty base name `%v` (%T)", l.project, specVal, specVal).debug(1)
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

        if n := ctx.dia().flush(); n > 0 {
            warn(at(ctx,position), "%v: %d errors: %v -> %v", l.project, n, elem, specName).debug(1)
            break ParamsLoop
        } else if f, y := toFile(elem); y && f.info != nil {
            absPath, isDir = f.fullname(), f.info.IsDir()
            if true { assert(filepath.IsAbs(absPath), "invalid abs path: %v", f) }
        } else if absPath, isDir = u.search(at(ctx,position), linfo, specName); absPath == "" {
            erro(at(ctx,elemPos), "%v: search base failed: %v → %v", l.project, elem, specName).debug(1)
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
            okay = l.dir(ctx, specName, absPath, nil)
        } else {
            okay = l.load(ctx, specName, absPath, nil)
        }
        l.implicit = implicitSaved // restore implicit flag

        var globe = l.Globe()
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
                l.project.scope.projectname(ctx, ".base", loaded)
            }
            l.project.bases = append(l.project.bases, loaded)
        } else if implicit {
            warn(of(ctx,elem), "implicit base '%s' not defined (as %s)", specName, absPath).debug(1)
        } else {
            erro(at(ctx,elemPos), "project `%v`(%T: %s) not loaded (%s)", elem, elem, specName, absPath).debug(1)
            break ParamsLoop
        }
    }

    usefor(ctx, l.project, func(op usevar, _, _ Value, name string) {
        var us = "use." + name
        var d, a = l.project.scope.define(ctx, DefVoid, us, nil)
        if d == nil && a != nil { d, _ = a.(*def) }
        if d == nil { return }
        op.apply(closureWith(ctx, l.project.scope), d, baseNonTrivialDefs(ctx, l.project, us)...)
    })
    return true
}

func (l *loader) loadDotContainer(ctx Context, ident *barecomp, identStr string, file *File) (result bool) {
    var u = cast[*universe](ctx)
    if u.traceLaunch { defer un(trace(t_launch, "loader.loadDotContainer")) }
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.dir(ctx, dotContainer, file.fullname(), nil) {
            erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug(1)
            return
        }
    } else if !l.file(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug(1)
        return
    }

    if loaded, yes := l.Globe().loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.Scope().Lookup(loaded.name).(*projectname)
        if name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.project.name, file).debug(1)
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
    var u = cast[*universe](ctx)
    if u.traceLaunch { defer un(trace(t_launch, "loader.loadDotConfigure")) }

    var position = ident.Position()
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.dir(ctx, dotConfigure, file.fullname(), nil) {
            erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug(1)
            return
        }
    } else if !l.file(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug(1)
        return
    }

    if loaded, y := l.Globe().loaded[file.fullname()]; y && loaded != nil {
        if name, _ := l.Scope().Lookup(loaded.name).(*projectname); name == nil {
            if _, alt := l.Scope().projectname(at(l, position), loaded.name, loaded); alt != nil {
                if val, ok := alt.(*projectname); !ok || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                }
            }
        } else {
            if conf := l.project.configure; conf != nil {
                if conf == loaded { return }
                erro(ctx, ".configure already specified").debug(1)
            }

            l.project.configure, result = loaded, true

            var opts = useOpts{}
            var ctx = at(l, position)
            for _, usee := range loaded.usees(true, false, false, false) {
                if err := l.use(ctx, usee, nil, opts); err != nil { // see usevars
                    erro(ctx, "using '%v' failed: %v", usee, err).debug(1)
                    break
                }
            }
        }
    }
    return
}

func (l *loader) declare(ctx Context, keyword Token, ident *barecomp, identStr string, declOpts *projectDeclOpts) (result bool) {
    var globe = l.Globe()

    if identStr == "@" {
        var (
            linfo = globe.loads[0]
            dec, y = linfo.declares[identStr]
            at, _ = globe.Lookup(identStr).(*projectname)
        )
        if !y {
            dec = &declare{ project: at.Project }
            linfo.declares[identStr] = dec
        }
        dec.backscope = l.Scope()
        l.useesExecuted = nil
        l.project = at.Project
        //l.scope = l.scope
        l.scopes[0] = at.Project.scope
        return true
    } else if _, o := l.Scope().find(identStr); o != nil {
        if _, y := o.(*builtin); y {
            erro(ctx, "project name '%s' is a builtin name", identStr)
            return
        }
    }

    var (
        name = identStr
        linfo = globe.loads[len(globe.loads)-1]
        dec, declared = linfo.declares[name]
    )
    if !declared {
        var (
            wd = l.workDir()
            outer = l.Scope()
            absDir = linfo.absDir
            relPath, tmpPath string
        )
        if !filepath.IsAbs(absDir) {
            //absDir = filepath.Join(l.workDir(), absDir)
            absDir, _ = filepath.Abs(absDir)
        }
        relPath, _ = filepath.Rel(wd, absDir)
        tmpPath = joinTmpPath(ctx, wd, relPath)

        // Avoid nesting project scopes!
        for strings.HasPrefix(outer.Comment(), "project \"") {
            outer = outer.outer
        }

        dec = &declare{ project: globe.project(ctx, outer, absDir, relPath, tmpPath, linfo.specName, name) }
        globe.loaded[linfo.absPath()] = dec.project
        linfo.declares[name] = dec
    }
    if loader := linfo.loader; loader != nil {
        if _, a := loader.scope.projectname(ctx, name, dec.project); a != nil {
            if v, y := a.(*projectname); !y || v == nil {
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
    if globe.main != nil && globe.main == l.project && l.project.name != "~" {
        for _, t := range globe.pairs {
            switch k := t.Key.(type) {
            case *bareword, *barecomp:
                var name = k.string(ctx);
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
        case *pair:
            var name = t.Key.string(ctx)
            var d, a = l.def(t.Key.Position(), name)
            if a != nil {
                var y bool
                if d, y = a.(*def); !y {
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

func (l *loader) autoload(ctx Context, tag string) {
    if proj := l.project; isConfigureProject(proj) {
        // skip...
    } else if obj := proj.resolve(ctx, ".autoload."+tag); obj == nil {
        // skip...
    } else if d, y := obj.(*def); !y {
        warnstack(ctx, 3, "%v: unsupported .auto: %T %v", proj, obj, obj).debug(1)
    } else if isTrivial(d.value) {
        // skip...
    } else if val := scalarize(d.value.expand(ctx, strval)); isTrivial(val) {
        // skip...
    } else {
        var u = cast[*universe](ctx)
        const ( o = true ; t = false ; s = "autoload" )
        if t && u.ddd == s { prompt(ctx, "%s: autoload - %s,\n", l.Position(), tag) }
        if o { l.include(ctx, includeOpts{}, val) }
        if t && u.ddd == s { prompt(ctx, "%s: autoload - %s.\n", l.Position(), tag) }
    }
}

func (l *loader) configure(ctx Context, linfo *loadinfo, ident *barecomp, identStr string, declared bool) (result bool) {
    if false { defer un(tracef(t_traverse, "configuration(%v)", ident)) }
    if s := l.project.name; s == dotConfigure { return }

    var local bool
    var configure string
    var v = l.project.opts.configure
    if v != nil {
        if t, y := v.(*boolean); y { if !t.bool { return } } else
        if !Is(v, KindNumber) { configure = v.string(ctx) }
    }
    if local = configure == "."; local || configure == "" { configure = "configure" }

    defer dtrace(ctx, "configuration: %v", configure)

    var loaded *Project
    var load = func(absPath string, isDir bool) (res bool) {
        if isDir {
            if !l.dir(ctx, configure, absPath, nil) { return }
        } else {
            if !l.file(ctx, absPath, nil) { return }
        }

        var globe = l.Globe()
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
        if !local { absPath, isDir = cast[*universe](ctx).search(ctx, linfo, configure) }
        if absPath == "" {
            erro(ctx, "%v: no such project: %s", l.project, configure).debug(1)
        }
    }
    if absPath == "" { return } else
    if !load(absPath, isDir) {
        erro(ctx, "%v: configure not loaded: %s", l.project, configure).debug(1)
        return
    }

    if name, _ := l.Scope().Lookup(dotConfigure).(*projectname); name == nil {
        if _, alt := l.Scope().projectname(ctx, dotConfigure, loaded); alt != nil {
            if val, y := alt.(*projectname); !y || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
            }
        }
    }
    if l.project.configure == loaded { return }
    if l.project.configure != nil {
        erro(ctx, ".configure already specified").debug(1)
        return
    }

    var opts = useOpts{}
    for _, usee := range loaded.usees(true, false, false, false) {
        if err := l.use(ctx, usee, nil, opts); err != nil { // see usevars
            erro(ctx, "using '%v' failed: %v", usee, err).debug(1)
            break
        }
    }

    var u = cast[*universe](ctx)

    // Load configuration.sm after .configure was loaded.
    l.project.configure = loaded // must set .configure first to get the correct configuration file
    l.project.configurationFile = l.project.configuration(ctx)
    if f := l.project.configurationFile; f == nil {
        erro(ctx, "%v: nil configuration file", ident).debug(1)
        return
    } else if declared || u.commandline.configure {
        // u.configuration.clean[f] = struct{}{}
    } else if f.exists() || f.stat(ctx) != nil {
        if false && (u.verboseImport || u.verboseLoads) {
            var cp Position; cp.Filename, cp.Line = f.fullname(), 1
            info(at(ctx,cp), "%s (%s)", l.project, l.project.spec).debug(true, 1)
        } else if u.verbose {
            info(ctx, "%v for %s (%s)", f, l.project.spec, l.project).debug(16)
        }

        var isIncludingConf = l.p.isIncludingConf
        l.p.isIncludingConf = true
        l.include(ctx, includeOpts{isConfigure:true}, f)
        l.p.isIncludingConf = isIncludingConf
    }
    return true
}

func (l *loader) container(ctx Context, ident *barecomp, identStr string) (result bool) {
    ctx = at(ctx, ident.Position())
    if l.project.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").debug(1)
            return
        }

        var u = cast[*universe](ctx)

        // Looking for project specific .container module
        if f := stat(ctx, dotContainer, l.project); f.exists() {
            if !l.loadDotContainer(ctx, ident, identStr, f) {
                //erro(ctx, "declare %s: %s/.container", name, l.project.absPath)
            }
            if u.verbose {
                info(ctx, "%v for %s (%s)\n", f, l.project.spec, l.project).debug(1)
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
    var globe = l.Globe()
    if identStr == "@" {
        if dec, y := globe.loads[0].declares[identStr]; y {
            l.scopes[0] = dec.backscope
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
            outer.scopename(at(l, scope.position), name, scope)
        } else {
            erro(ctx, "open scope '%s' failed (%v)", name, comment).debug(8)
        }
    }
    return
}

func (l *loader) resolve(value Value) (name string, result Value) {
    var pos = value.Position()
    if !pos.IsValid() { pos = l.Position() }
    if _, y := value.(*selection); y {
        panic(failure{"resolving a selection",ia(pos)})
    }

    var ctx = at(l, pos)
    if name = value.string(ctx); name == "" {
        erro(ctx, "name '%v' is empty", name).debug(1)
        return
    }

    if l.Scope() == nil {
        erro(ctx, "nil scope to resolve '%v'", name).debug(1)
        return
    } else if _, result = l.Scope().find(name); !isNull(result) {
        // okay!
    } else if project := l.project; project == nil {
        erro(ctx, "nil project to resolve '%v'", name).debug(1)
        return
    } else if result = project.resolve(ctx, name); isNull(result) {
        if o, y := value.(optional); y {
            result = unresolved{o.Value, project}
            return
        }
        //erro(ctx, "%v: resolved object '%v' is nil", project, name).debug(1)
    }
    return
}

// func (l *loader) auto(ctx Context, name string) (a *auto, alt Object) {
//     var scope = l.Scope()
//     if  strings.HasPrefix(scope.comment, "file ") && l.mode&Flat != 0 {
//         // use project scope if defining in flat file (aka. include)
//         // to ensure that the symbol is valid in the project
//         scope = l.Scope()
//     }
//     if a, alt = scope.auto(l, name); a != nil { a.position = ctx.Position() }
//     return
// }

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

func (l *loader) source(ctx Context, filename string, src interface{}, mode Mode, opts *includeOpts) (f *parsedFile, res []Value, err error) {
    var u = cast[*universe](ctx)
    if u.traceLaunch { defer un(trace(t_launch, "loader.ParseFile")) }
    if u.verbose { if ctx.Position().Filename == filename {
        info(ctx, "loading ...")
    } else {
        prompt(ctx, "%s:1:info: loading ...\n", filename)
        info(ctx, "loading %v", filename)
    }}

    assert(ctx.loader() == l, "require the same loader context")

    defer dtrace(ctx, "source")

    defer func(t time.Time, p *parser, m Mode) { if true { ctx = l.p.ctx(ctx) }
        if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 10, "%v: slow: %v (%v)", l.project, d, u.slow).debug(2) //  → %s, filename
        } else if u.verbose {
            info(ctx, "loaded %v (%v)", filename, d).debug(2)
        }

        l.p, l.mode = p, m

        if err != nil { errostack(ctx, 3, "source error: %v", err).debug(1) } else
        if f == nil && res == nil { erro(ctx, "source not loaded: %s", filepath.Base(filename)).debug(1) }
    } (time.Now(), l.p, l.mode)

    l.mode = mode

    var text []byte
    if text, err = readSource(filename, src); err != nil {
        if _, y := err.(*fs.PathError); y && opts.ifExists {
            if opts.debug>0 {
                prompt(ctx, "%v: source file not found\n", filename)
                warnstack(ctx, 5, "#>", opts.values[0]).debug(opts.debug)
            }
        } else {
            prompt(ctx, "%v: %v\n", filename, err)
            erro(ctx, "read source failed: %v (%T)", err, err)
            errostack(ctx, 5, "").debug(32)
        }
        return
    }

    if l.p = (&parser{Context:ctx}); opts != nil {
        l.p.isIncludingConf = opts.isConfigure
    }

	var scanMode ScanMode
	if l.mode&ParseComments != 0 {
		//scanMode = scanner.ScanComments
	}

    var file = u.file(filename, text)
    l.p.scanner.Init(file, text, scanMode,
        func(p Position, s string) {
            var pos = Position(p)
            errostack(at(ctx,pos), 3, "%s, scan=%v", s, l.p.scanner.scanState).debug(128)
            panic(failure{"syntax error",ia(pos)})
        },
        func(p Position, s string) {
            // warnstack(at(ctx,Position(p)), 3, "%s, scan=%v", s, l.p.scanner.scanState).debug(1)
            warn(at(ctx,Position(p)), "%s, scan=%v", s, l.p.scanner.scanState).debug(6)
        })
	l.p.next(true)

    if ctx = l.p.ctx(ctx); l.mode&parsingText != 0 {
        res = l.p.text(ctx)
    } else if f = l.p.file(ctx); f == nil {
        // Source is not a valid source file, returnning a valid but empty parsedFile
        defer l.closeScope(l.openScope(fmt.Sprintf("file %s", filename)))
        f = &parsedFile{ scope:l.Scope() }
        f.position.Filename = filename
        // TODO: validate basename as a valid identifier
        f.name = makeBarecomp(f.position, makeBareword(f.position, filepath.Base(filepath.Dir(filename))))
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

    defer l.closeScope(l.OpenNamedScope(ident, "config "+pathname))

    var def *def
    var ctx = at(l, l.p.Position())
ListLoop:
    for _, d := range list {
        var name = d.Name()
        if hasPrefix(name, "~") || hasSuffix(name, ".#", ".smart", ".sm") {
            continue ListLoop
        }

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
            if ctx.dia().flush() > 0 { return }
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
            def.set(ctx, DefConfDir, makeStrlit(ctx.Position(), s))
        } else if s != nil {
            erro(ctx, "Name `%s' already taken, not def (%T).", name, s)
            break ListLoop
        }
    }
    return
}

func (l *loader) sources(ctx Context, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*Project) {
    var u = cast[*universe](l.Context)

    defer dtrace(ctx, "sources")

    defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseParse || d > 3*time.Second {
            noted(ctx, "slow: %s (%v)", l.project, d).debug(1)
        } else if u.debugParsing(ctx, "sources") {
			noted(ctx, "sources: %s (%v)", l.project, time.Now().Sub(t)).debug(6)
		}
    } (time.Now())

    var fd, err = os.Open(path)
    if err != nil {
        erro(ctx, "%v", err).debug(1)
        return
    }
    defer fd.Close()

    fis, err := fd.Readdir(-1)
    if err != nil {
        erro(ctx, "readdir: %v", err).debug(1)
        return
    } else if len(fis) == 0 {
        erro(ctx, "no files underneath: %s", path).debug(1)
        return
    }

    first := fis[0]
    for i, a := range fis { if i > 0 { s := a.Name()
        if s == entryFileName || (s == "build.smart" && first.Name() != entryFileName) {
            fis[0] = a
            fis[i] = first
        }
    }}

    defer l.closeScope(l.openScope("dir "+path))

    // FIXES: use 'globe' scope as outer to avoid chaining scopes to other unrelated
    // projects which are in consequence load order. Setting dir scope outer to such
    // project scopes will cause resolving objects to the wrong ones.
    l.Scope().outer = ctx.Globe().Scope

    if  l.Scope().position.Filename == "" {
        l.Scope().position.Filename = path
        l.Scope().position.Line = 1
    }

    mods = make(map[string]*Project)

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

        if mo.IsRegular() && (filter == nil || filter(d)) { var pos Position
            pos.Filename, pos.Line = filename, 1

            var src, _, err = l.source(ctx, filename, nil, mode|parsingDir, nil)

            var d *diagPoint
            if n := ctx.dia().flush(); n > 0 { total := ctx.dia().totalErrors()
                if err != nil { erro(ctx, "parse failed: %v", err) }

                s := filepath.Base(filename)
                d = erro(ctx, "got %d errors in file '%s'", n, s)
                if u.failOnErrors {
                    panic(failure{"got %d errors, %s",ia(ctx.Position(),total,s)})
                }
            } else if err != nil {
                d = erro(ctx, "parse file failed: %v", err)
            } else if src == nil {
                d = erro(ctx, "parsed nil module from")
            } else if isNull(src.name) {
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

            var name = src.name.string(ctx)
            if mod, found := mods[name]; !found {
                mod = &Project{ name: name, scope: l.Scope() }
                mods[name] = mod
            }
        }
    }
    return
}

// loader.Load loads script from a file or source code (string, []byte).
func (l *loader) load(ctx Context, specName, absPath string, source interface{}) (result bool) {
    var u = cast[*universe](ctx)
    if u.traceLaunch { defer un(trace(t_launch, "loader.load")) }

    var globe = l.Globe()
    defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseLoads && d>1*time.Second {
            loaded, _ := globe.loaded[absPath]
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
    if loaded, yes := globe.loaded[absPath]; yes {
        if _, a := l.Scope().projectname(at(l, l.Position()), loaded.name, loaded); a != nil {
            if val, ok := a.(*projectname); !ok || val == nil {
                erro(ctx, "`%v` name already taken (%T).", loaded, a)
            }
        }
        result = true
        return
    }

    var absDir, baseName = filepath.Split(absPath)
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

    var doc, _, err = l.source(ctx, absPath, source, parseMode, nil)
    if n := l.dia().flush(); n > 0 {
        warn(ctx, "load '%s' got %d errors", specName, n).debug(1)
        if u.failOnErrors {
            panic(failure{"fail by %d errors",ia(l.Position(), l.dia().totalErrors())})
        }
        return
    } else if err != nil {
        erro(ctx, "load: %v", err).debug(1)
    } else if doc == nil {
        erro(ctx, "load: nil: %s", absPath).debug(1)
    } else {
        result = true
    }
    return
}

func (l *loader) dir(ctx Context, specName, absDir string, filter func(os.FileInfo) bool) (loadedOkay bool) {
    if cast[*universe](ctx).traceLaunch { defer un(trace(t_launch, "loader.dir")) }

	defer dtrace(ctx, "dir (%s)", specName)

    if !filepath.IsAbs(absDir) {
        errostack(ctx, 3, "needs absolute dir `%s' (%s)", absDir, specName).debug(10)
        return
    }

    var pos Position = ctx.Position()
    if !pos.IsValid() { pos = positionForDir(absDir) }

    var loaded *Project
    var globe = ctx.Globe()
    defer func(t time.Time, ver bool) {
        if specName == "." { specName = absDir }

        if d := time.Now().Sub(t); ver && d>1*time.Second { if l.project == nil {
            noted(ctx, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName).debug(1)
        } else {
            noted(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, specName).debug(1)
        }}

        if loaded == nil {
            erro(ctx, "dir not loaded: %v", specName).debug(1)
            return
        }

        if globe.main == nil { globe.main = loaded }

        if proj := l.Scope().project; proj == nil {
            if false { erro(ctx, "%v: no owner project for %s", loaded.name, l.Scope()).debug(1) }
        } else if name, _ := proj.scope.Lookup(loaded.name).(*projectname); name == nil {
            if _, alt := proj.scope.projectname(at(ctx,pos), loaded.name, loaded); alt != nil {
                if val, y := alt.(*projectname); !y || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                }
            }
        }
    } (time.Now(), cast[*universe](ctx).verboseLoads)

    // Check loaded project.
    if loaded, loadedOkay = globe.loaded[absDir]; loadedOkay { return }

    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

    var mods map[string]*Project
    if mods = l.sources(at(l, pos), absDir, filter, parseMode); mods == nil {
        errostack(ctx, 3, "failed parsing module: %s", specName).debug(12)
        if cast[*universe](ctx).failOnErrors {
            panic(failure{"fail by %d errors",ia(l.Position(), l.dia().totalErrors())})
        }
        return
    }

    // FIXME: loading failed if different 'project' found in
    // the same dir, for example:
    //      project Foo # file do.smart
    //      project # file config.smart
    if len(mods) == 0 && filepath.Base(specName) != "@" {
        if l.implicit {
            loadedOkay = true // okay for implicit loading
            warn(l, "%s not loaded (as %s, implicitly)", specName, absDir).debug(10)
        } else {
            for s, m := range globe.loaded { erro(ctx, "%v: %v", s, m) }
            errostack(ctx, 3, "%s not loaded (as %s)", specName, absDir).debug(10)
        }
    } else if loaded, loadedOkay = globe.loaded[absDir]; loadedOkay && loaded != nil {
        // Good!
    } else if filepath.Base(specName) != "@" {
        erro(ctx, "%s not loaded (as %s, implicit=%v)", specName, absDir, l.implicit).debug(1)
    }
    return
}

func (l *loader) file(ctx Context, filename string, source interface{}) (res bool) {
    if cast[*universe](ctx).traceLaunch { defer un(trace(t_launch, "loader.file")) }

    var spec string
    switch dir, base := filepath.Split(filename); base {
    case dotBase, dotConfigure: spec = base
    default: spec, _  = filepath.Rel(l.workDir(), dir)
    }

    var position Position
    position.Filename = filename
    return l.load(at(ctx, position), spec, filename, source)
}

func (l *loader) path(ctx Context, path string, filter func(os.FileInfo) bool) bool {
    if cast[*universe](ctx).traceLaunch { defer un(trace(t_launch, "loader.path")) }

    var spec, _ = filepath.Rel(l.workDir(), path)

    var position Position
    position.Filename = spec
    return l.dir(at(ctx, position), spec, path, filter)
}

func (l *loader) text(ctx Context, filename string, text string) (res []Value) {
    if cast[*universe](ctx).traceLaunch { defer un(trace(t_launch, "loader.text")) }

    defer func(saved *parser) { l.p = saved } (l.p)

    if g := l.Globe(); g.main == nil {
        l.scopes[0] = g.os.scope
    } else {
        l.scopes[0] = g.main.scope
    }
    l.useesExecuted = nil

    var err error
    var opts includeOpts
    var position Position
    position.Filename = filename
    ctx = at(ctx, position)

    if _, res, err = l.source(ctx, filename, text, parsingText, &opts); err != nil {
        prompt(ctx, "%v: %v\n", filename, err)
        erro(ctx, "load text failed: %v", err)
        errostack(ctx, 5, "").debug(32)
    }
    return
}
