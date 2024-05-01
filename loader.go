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

func isEntryFileName(s string) bool { return filepath.Base(s) == entryFileName }

type ResolveBits int
const (
    // If many bits are set, resolve in the listed priority.
    FromGlobe ResolveBits = 1<<iota
    FromBase
    Fromproject
    FindDef
    FindRule

    FromHere

    // This is the default be
    anywhere = FromHere
    global = FromGlobe
    local = Fromproject
    nonlocal = FromGlobe | FromBase | Fromproject
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
    project *project
    //backproj *project
    backscope *Scope
    useesExecuted []*project
}

type loadinfo struct {
    absDir string // absPath = filepath.Join(absDir, baseName)
    baseName string
    specName string
    useesExecuted []*project
    loader *project
    loadee *project // the current loading project
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

func _loader(c Context) *loader { return cast[*loader](c) }

type loader struct {
    terminal
    p *parser
    proj *project
    mode Mode // parsing mode
    loadArgs []Value
    loadStack []*project // load path
    useStack  []*project // use path
    useesExecuted []*project // all executed usees
    implicit bool // loading current project implicitly, aka. via foo.bar.Baz (implicit foo/bar loaded)
    verpre string // verbose prefix
}
func (l *loader) project() *project { return l.proj }
func (l *loader) inner() Context { return &l.terminal }
func (l *loader) cast(t reflect.Type) Context {
    if reflect.TypeOf(l)   == t { return l }
    if reflect.TypeOf(l.p) == t { return l.p }
    return l.terminal.cast(t)
}
func (l *loader) String() string {
    if fullContextStringer {
        return fmt.Sprintf("loader{%s}", &l.terminal)
    } else {
        return l.terminal.String()
    }
}

func restoreLoadingInfo(l *loader) {
    var globe = l.Globe()
    var last = len(globe.loads)-1
    var linfo = globe.loads[last]
    globe.loads = globe.loads[0:last]
    l.useesExecuted = linfo.useesExecuted
    l.proj = linfo.loader
    l.scopes = linfo.scopes
}

func saveLoadingInfo(l *loader, specName, absDir, baseName string) *loader {
    var globe = l.Globe()
    globe.loads = append(globe.loads, &loadinfo{
        absDir: absDir,
        baseName: baseName,
        specName: filepath.Clean(specName),
        useesExecuted: l.useesExecuted,
        loader:   l.proj,
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
                note(ctx, "%v", us(t))
            }
            erro(at(ctx,val), "%v: empty use spec: '%v' (%T)", user, spec, spec).debug(1)
        } else {
            f(op, spec, val, name)
        }
    }}}
}
func usevars(ctx Context, user, usee *project) {
    var ddd = _universe(ctx).ddd == "use"
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
            var d, a = user.scope.define(ctx, defVoid, useDef.ident(ctx), nil)
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
            op.apply(closureWith(ctx, usee.scope), d, append(dd, useDef)...)
        }

        if useDef.value == nil || isTrivial(useDef.value) { return }

        // 2. XXX += $(use.XXX)
        {
            var d, a = user.scope.define(ctx, defVoid, name, nil)
            var isNewDef = d != nil && a == nil
            if d == nil && a != nil { d, _ = a.(*def) }
            if d == nil { return }
            if isNewDef && false {
                if dd == nil { dd = append(dd, baseNonTrivialDefs(ctx, user, useDef.ident(ctx))...) }
                dd = append(dd, baseNonTrivialDefs(ctx, user, name)...)
            }
            op.apply(closureWith(ctx, user.scope), d, append(dd, useDef)...)
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

func (l *loader) Position() (res Position) {
    if l.p != nil { return l.p.Position() }
    return l.Context.Position()
}

func (l *loader) usespec(ctx Context, opts useOpts, specVal Value, arged []Value, params ...Value) (loaded *project) {
	if true { defer trace(ctx) }

    var (
        u = _universe(l.Context)
        globe = l.Globe()
        linfo = globe.loads[len(globe.loads)-1]
        absPath, specName string
        isDir, traveUseLoop bool
        err error
    )
    if t, y := specVal.(*project); y {
        loaded = t
    } else if specName = specVal.string(ctx); specName == "" {
        errostack(ctx, 3, "empty spec: %v", us(specVal)).debug(6)
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
            if val, ok := a.(*project); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, a).debug(1)
            }
        }
        return
    }

    defer func(a []*project) { if l.loadStack = a; loaded == nil {
        erro(ctx, "%v not loaded (%v,dir=%v)", specName, absPath, isDir).debug(1)
        return
    } else if name, _ := scope.Lookup(loaded.name).(*project); name == nil {
        if _, alt := scope.projectname(ctx, loaded.name, loaded); alt != nil {
            if val, ok := alt.(*project); !ok || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
            }
        }
        if false {
            erro(ctx, "%v (%v,dir=%v)", specName, absPath, isDir).debug(1)
            return
        }
    }} (l.loadStack)

    l.loadStack = append(l.loadStack, l.proj) // build the load path

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
        if proj, res, isb, err = l.proj.hasLoaded(ctx, loaded, traveUseLoop); err != nil {
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
            okay = l.directory(ctx, specName, absPath, nil)
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
    for _, use := range l.proj.use.list {
        var (
            up = use.project
            proj *project
            res, isb bool
        )
        if loaded == up {
            if !opts.noVars && !opts.files {
                errostack(ctx, 10, "%v: using `%s` multiple times: %v", l.project, specName, l.proj.use.list).debug(10)
            }
            return
        }

        if false && loaded.opts.multiUseAllowed {
            // ...
        } else if proj, res, isb, err = loaded.hasLoaded(ctx, up, traveUseLoop); err != nil {
            erro(ctx, "load '%s' failed: %s", specName, err).debug(1)
            return
        } else if isb {
            if l.proj.hasBase(up) {
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
    if l.proj == nil {
        erro(ctx, "current project is nil").debug(32)
        return
    }

    var g = stat(ctx, "smart.go", l.proj)
    if g == nil { return /* smart.go was not presented */ }

    var src = g.string(ctx)
    s := strings.Replace(l.proj.relPath, "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath(ctx, "", "")), "plugins", s)

    var build = true

    so := stat(ctx, /*l.proj.name*/"plugin", stat_dir{s}, stat_nonexist{true})
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
    if l.proj.plugin, err = plugin.Open(s); err == nil {
        var p plugin.Symbol
        if p, err = l.proj.plugin.Lookup("Init"); err != nil {
            return
        } else if p == nil {
            // no initialization (optional)
        } else if f, ok := p.(func(Position, *project) (*Scope, error)); ok {
            l.proj.pluginScope, err = f(ctx.Position(), l.proj)
        } else if f, ok := p.(func(*project) (*Scope, error)); ok {
            l.proj.pluginScope, err = f(l.proj)
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
        prompt(ctx, "use(%15s) %s ⇒ %v\n", d, l.proj, l.proj.use)
    } (time.Now())}

    if usee == l.proj {
        erro(l, "'%v' use loop (%s)", usee.name, l.usePath())
        return
    } else if l.proj.isUsingDirectly(usee) {
        return
    }

    defer func(a []*project) { l.useStack = a } (l.useStack)
    l.useStack = append(l.useStack, usee) // build the use path

    // Add to the project using list, so that the use path is correct.
    if l.proj.use.append(ctx, usee, params, opts); !opts.noVars {
        // aka.     XXX += $(use.XXX)
        // aka. use.XXX += $(use.XXX)
        usevars(ctx, l.proj, usee)
        if 0 < len(opts.vars) {
            for _, v := range opts.vars {
                warn(at(ctx,v), "var: %T %v", v, v)
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

func for_ident_elems(ctx Context, elems, stems []Value, f func(elems, stems []Value)) {
    for i, elem := range elems { if a, y := elem.(*argumented); y {
        var prefix, suffix = elems[:i], elems[i+1:]
        for_idents(ctx, a, func(ident Value, stems2 []Value) {
            var head   = append(prefix, ident)
            var stems3 = append(stems , stems2...)
            for_ident_elems(ctx, suffix, stems3, func(elems, stems []Value) {
                f(append(head, elems...), stems)
            })
        })
        return
    }}
    f(elems, stems)
}

func for_idents(ctx Context, ident Value, f func(ident Value, stem []Value)) {
    switch t := ident.(type) {
    case *argumented:
        var args = xmerge(ctx, t.args...)
        for_idents(ctx, t.Value, func(ident Value, stems []Value) {
            for _, arg := range args {
                if isTrivial(arg) { continue }
                f(makeBarecomp(arg), append(stems, arg))
            }
        })
    case *barecomp:
        for_ident_elems(ctx, t.elems, nil, func(elems, stems []Value) {
            if len(stems) == 0 { f(t, stems) } else {
                f(makeBarecomp(elems...), stems)
            }
        })
    default:
        f(t, nil)
    }
}

func (l *loader) define(ctx Context, tok token, ident, value Value) (defs []*def) {
    for_idents(ctx, ident, func(ident Value, stems []Value) {
        if d := l.define1(ctx, tok, ident, value); d != nil { defs = append(defs, d) }
    })
    return
}

func (l *loader) define1(ctx Context, tok token, ident, value Value) (d *def) {
    defer trace(ctx)

	if checkpoints { defer func() {
		if d == nil {
			erro(ctx, "%v %v %v", ident, tok, us(value)).debug(5)
		} else if d.value == nil && value != nil {
			erro(ctx, "%v %v %v", ident, tok, us(value)).debug(5)
		}
    }()}

    var alt Object

    switch t := ident.(type) {
    case *argumented:
        erro(ctx, "TODO: multiple defs: %v, args=%v", t.Value, t.args).debug(1)
        return

    case *group:
        erro(ctx, "TODO: multiple defs: %v", t.elems).debug(1)
        return

    case *selection:
        if v := t.expand(final{ctx}); v == nil {
            erro(ctx, "%v is nil", us(t)).debug(1)
            return
        } else if x, y := v.(*def); !y {
            erro(ctx, "%v is not a def: %v", us(t), us(v)).debug(1)
            return
        } else {
            d = x
        }

    default: // *bareword, *barecomp, *qualiword, *path, flag:
        var name = t.string(ctx)
        if _, y := builtins[name]; y {
            erro(ctx, "`%v` is a builtin name (%v)", ident, name).debug(1)
            return
        }

        // Resolve base value to derive.
        var prev = l.proj.resolve(ctx, name)

        if d, alt = l.def(t.Position(), name); alt == nil {
            if d == nil {
                erro(ctx, "`%s` is undefined (%v)", name, us(t)).debug(1)
                return
            }
        } else if tok == ASSIGN || tok == ASSIGN_EXC {
            if a, y := alt.(*def); !y {
                erro(ctx, "`%v` already defined (%T) (%v,%v)", ident, alt, alt.owner(), l.proj).debug(1)
                return
            } else if a.owner() == l.proj && a.origin != defConfRef {
                erro(ctx, "`%v` already defined (%T) (%v)", ident, alt, l.proj).debug(1)
                return
            } else {
                d = a
            }
        } else {
            d = alt.(*def)
        }

        if prev == nil {
            // no derived value
        } else if prev.owner() == l.proj {
            // not derivable def if they are from the same project
        } else if derived, y := prev.(*def); !y {
            // not a def
        } else if derived == nil {
            erro(ctx, "prev def '%s' is nil", name).debug(1)
        } else if derived == d || (d.value != nil && d.value.refs(ctx, derived)) {
            // same def
        } else if d != nil && (tok == ASSIGN_ADD || tok == ASSIGN_SHI) && alt == nil {
            if d.origin == defVoid { d.origin = derived.origin }
            if !isTrivial(derived.value) { d.append(ctx, derived.value) }
        }
    }

    if d == nil {
        erro(ctx, "def is nil: %v", us(ident)).debug(1)
        return
    }

    d.position = ident.Position()

    switch tok {
    case ASSIGN    : d.set(ctx, defExpand0, value) //   =
    case ASSIGN_CO1: d.set(ctx, defExpand1, value) //  :=
    case ASSIGN_CO2: d.set(ctx, defExpand2, value) // ::=
    case ASSIGN_EXC: d.set(ctx, defExecute, value) //  !=
    case ASSIGN_QUE: if alt == nil { d.set(ctx, d.origin, value) } // ?=
    case ASSIGN_ADD: if!isTrivial(value) { d.set(ctx, d.origin, nil, merge(value)...) } // +=
    case ASSIGN_SHI: if!isTrivial(value) { d.set(ctx, d.origin, value, merge(d.value)...) } // =+
    case ASSIGN_SUB: if nil != d.value { if dv := merge(d.value); len(dv) > 0 { // -=
        var vals []Value
        var sub = merge(value)
    outer1:
        for _, v := range dv {
            for _, sv := range sub { if v.cmp(ctx, sv) == cmpEqual { continue outer1 }}
            vals = append(vals, v)
        }
        d.value = ease(ctx, vals)
    }}
    case ASSIGN_SAD, ASSIGN_SSH: // -+=, -=+
        var vals []Value
        var sub = merge(value)
        if d.value != nil { if dv := merge(d.value); len(dv) > 0 {
        outer2:
            for _, v := range dv {
                for _, sv := range sub { if v.cmp(ctx, sv) == cmpEqual { continue outer2 }}
                vals = append(vals, v)
            }
        }}
        if ASSIGN_SAD == tok {
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

func (l *loader) rule(clause *parsedRuleData) (entries []entry) {
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
        case *list: depends = append(depends, dep.elems...)
        default:    depends = append(depends, dep)
        }
    }
    for _, depend := range clause.ordered {
        switch dep := depend.(type) {
        case *list: ordered = append(ordered, dep.elems...)
        default:    ordered = append(ordered, dep)
        }
    }

    var prog = &program{
        language: l.p.dialect,
        project:  l.proj,
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
        var entry, err = l.proj.entry(ctx, clause.special, clause.options, target, prog)
        if err != nil {
            erro(at(ctx,target), "creating entry '%v' failed: %v", target, err)
            return
        } else {
            entries = append(entries, entry)
        }

        if t, okay := entry.Target().(flag); okay && t.Value != nil {
            var s = t.Value.string(ctx)
            if l.proj.name != "~" { l.Globe().AddFlagEntry(s, entry) }
        } else if configure {
            if entry.Class() == PatternRule {
                erro(ctx, "unsupported pattern configures: %v", target).debug(1)
                return
            } else {
                l.proj.configs = append(l.proj.configs, entry)
            }
        }
    }
    return
}

type includeOpts struct {
    *clauseopts
    ifExists bool `if-exists,ifexists`
    isConfigure bool // internal
}
func (l *loader) include(ctx Context, opts includeOpts, spec Value) {
    defer trace(ctx)

    var (
        u = _universe(ctx)
        globe = l.Globe()
        linfo = globe.loads[len(globe.loads)-1]
        specName, fullname string
        err error
    )

    defer func(t time.Time) {
        if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 10, "%v: slow: %v (%v)", l.proj, d, u.slow).debug(1) //  → %s, filename
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
        if fullname, specName = t.fullname(), t.ident(ctx); t.info == nil {
            prompt(ctx, "%v: no source file %v, %v\n", fullname, t, t.stat(ctx))
            erro(at(ctx,t), "%v: %v", ctx.project(), t)
            errostack(ctx, 5, "").debug(16)
            return
        }
    default:
        if specName = spec.string(ctx); specName == "" {
            erro(at(ctx,spec), "include: empty string: %v", spec)
            errostack(ctx, 5, "").debug(16)
            return
        }

        var file = l.proj.file(ctx, spec)
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

    flush(ctx)
    return
}
func (l *loader) closure() (scopes []*Scope) {
    scopes = append(l.terminal.closure(), l.Scope())
    return
}

func (l *loader) openScope(comment string) (scopes []*Scope) {
    var u = _universe(l.Context)
    if false && u.traceLaunch { defer un(l_trace(l_launch, "loader.openScope")) }

    var pos Position
    if l.p != nil { pos = l.Position() } else {
        // TODO: pos.Filename = l.path(), 1
    }

    var scope = newScope(pos, l.Scope(), l.proj, comment)
    scopes = l.scopes
    l.scopes = append([]*Scope{scope}, scopes...)
    return
}

func (l *loader) closeScope(scopes []*Scope) {
    var u = _universe(l.Context)
    if false && u.traceLaunch { defer un(l_trace(l_launch, "loader.closeScope")) }
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
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.bases")) }

    // For &(foobar) set from loadArgs
    ctx = closureWith(ctx, l.scopes...)

    var (
        implicitIndex int
        implicitBases []Value
        position = ctx.Position()
    )
    if file := stat(ctx, dotBase, l.proj); file != nil {
        if true { var s = file.string(ctx)
            assert(s == file.ident(ctx) && s == dotBase, "invalid finalization: %v => %v", file, s)
        }
        if !file.info.IsDir() && (l.proj.spec == dotBase /*|| l.proj.spec == dotConfigure*/) {
            // skip the regular file '.base' to avoid self loading recursively
            // info(ctx, "%v", file).debug(1)
        } else {
            implicitBases = append(implicitBases, file)
        }
    }

    if ns := strings.Split(l.proj.name, "."); len(ns) > 2 && ns[len(ns)-1] == "base" {
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
            if false { warn(ctx, "%v, %v, %v; %v, %v, %v", l.proj.name, ns, segs,
                implicitBase, implicitBases, params).debug(1) }
            implicitBase = "" // discard the implicit base
        } else if false /* && numBaseParams == 1 */ {
            warn(ctx, "%v, %v, %v, %v; %v, %v, %v",
                l.proj.name, ns, filepath.Join(ns[:len(ns)-2]...), numBaseParams,
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

            var defs = l.define(at(ctx, position), ASSIGN, ident, p.val)
            if len(defs) == 0 {/* TODO: check defs... */}
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
            erro(at(ctx,elemPos), "%v: empty base name `%v` (%T)", l.proj, specVal, specVal).debug(1)
            break ParamsLoop
        } else if strings.Contains(specName, "//") {
            erro(at(ctx,elemPos), "%v: invalid spec: %v in %T", l.proj, elem, ctx)
            erro(at(ctx,elemPos), "%v: invalid spec: %v -> %v", l.proj, elem, specVal)
            erro(at(ctx,elemPos), "%v: invalid spec: %v -> %v", l.proj, elem, specName).debug(10)
            break ParamsLoop
        } else if implicitBase != "" && specName == implicitBase {
            if i == implicitIndex { implicit = true } else {
                erro(at(ctx,elemPos), "%v: base '%v' already loaded implicitly", l.proj, elem).debug(1)
                if false { break ParamsLoop } else { continue }
            }
        }

        if n := flush(ctx); n > 0 {
            warn(at(ctx,position), "%v: %d errors: %v -> %v", l.proj, n, elem, specName).debug(1)
            break ParamsLoop
        } else if f, y := toFile(elem); y && f.info != nil {
            absPath, isDir = f.fullname(), f.info.IsDir()
            if true { assert(filepath.IsAbs(absPath), "invalid abs path: %v", f) }
        } else if absPath, isDir = u.search(at(ctx,position), linfo, specName); absPath == "" {
            erro(at(ctx,elemPos), "%v: search base failed: %v → %v", l.proj, elem, specName).debug(1)
            break ParamsLoop
        }

        for _, base := range l.proj.bases {
            if base.absPath == absPath {
                //erro(at(ctx,elemPos), "duplicated base: %v (in %v)", elem, l.proj.bases)
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

        var globe = l.Globe()
        if !okay {
            var pos Position
            pos.Filename, pos.Line = absPath, 1
            erro(ctx, "%v: '%s' not loaded'", l.proj, specName)
            erro(at(ctx,elemPos), "%v: base '%s' not loaded, %v", l.proj, specName, elem)
            erro(at(ctx,position), "%v: base '%s' not loaded, %s", l.proj, specName, absPath).debug(6)
            break ParamsLoop
        } else if loaded, y := globe.loaded[absPath]; y && loaded != nil {
            if l.proj.hasBase(loaded) { continue ParamsLoop }
            if l.proj.bases == nil { // set .base to the first project name
                l.proj.scope.projectname(ctx, ".base", loaded)
            }
            l.proj.bases = append(l.proj.bases, loaded)
        } else if implicit {
            warn(at(ctx,elem), "implicit base '%s' not defined (as %s)", specName, absPath).debug(1)
        } else {
            erro(at(ctx,elemPos), "project `%v`(%T: %s) not loaded (%s)", elem, elem, specName, absPath).debug(1)
            break ParamsLoop
        }
    }

    usefor(ctx, l.proj, func(op usevar, _, _ Value, name string) {
        var us = "use." + name
        var d, a = l.proj.scope.define(ctx, defVoid, us, nil)
        if d == nil && a != nil { d, _ = a.(*def) }
        if d == nil { return }
        op.apply(closureWith(ctx, l.proj.scope), d, baseNonTrivialDefs(ctx, l.proj, us)...)
    })
    return true
}

func (l *loader) loadDotContainer(ctx Context, ident *barecomp, identStr string, file *File) (result bool) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.loadDotContainer")) }
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.directory(ctx, dotContainer, file.fullname(), nil) {
            erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug(1)
            return
        }
    } else if !l.file(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug(1)
        return
    }

    if loaded, yes := l.Globe().loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.Scope().Lookup(loaded.name).(*project)
        if name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.proj.name, file).debug(1)
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
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug(1)
        return
    } else if file.info.IsDir() {
        if !l.directory(ctx, dotConfigure, file.fullname(), nil) {
            erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug(1)
            return
        }
    } else if !l.file(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load %v failed  (%s)", ident, dotConfigure, file.fullname()).debug(1)
        return
    }

    if loaded, y := l.Globe().loaded[file.fullname()]; y && loaded != nil {
        if name, _ := l.Scope().Lookup(loaded.name).(*project); name == nil {
            if _, alt := l.Scope().projectname(at(l, position), loaded.name, loaded); alt != nil {
                if val, ok := alt.(*project); !ok || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
                }
            }
        } else {
            if conf := l.proj.configure; conf != nil {
                if conf == loaded { return }
                erro(ctx, ".configure already specified").debug(1)
            }

            l.proj.configure, result = loaded, true

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

func (l *loader) declare(ctx Context, keyword token, ident *barecomp, identStr string, declOpts *projectDeclOpts) (result bool) {
    defer trace(ctx)

    var globe = l.Globe()

    if identStr == "@" {
        var (
            linfo = globe.loads[0]
            dec, y = linfo.declares[identStr]
            at, _ = globe.Lookup(identStr).(*project)
        )
        if !y {
            dec = &declare{ project: at }
            linfo.declares[identStr] = dec
        }
        dec.backscope = l.Scope()
        l.useesExecuted = nil
        l.proj = at
        //l.scope = l.scope
        l.scopes[0] = at.scope
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
            wd = _workdir(l)
            outer = l.Scope()
            absDir = linfo.absDir
            relPath, tmpPath string
        )
        if !filepath.IsAbs(absDir) {
            //absDir = filepath.Join(_workdir(l), absDir)
            absDir, _ = filepath.Abs(absDir)
        }
        relPath, _ = filepath.Rel(wd, absDir)
        tmpPath = joinTmpPath(ctx, wd, relPath)

        // Avoid nesting project scopes!
        for strings.HasPrefix(outer.comment, "project \"") {
            outer = outer.outer
        }

        dec = &declare{ project: globe.project(ctx, outer, absDir, relPath, tmpPath, linfo.specName, name) }
        globe.loaded[linfo.absPath()] = dec.project
        linfo.declares[name] = dec
    }
    if loader := linfo.loader; loader != nil {
        if _, a := loader.scope.projectname(ctx, name, dec.project); a != nil {
            if v, y := a.(*project); !y || v == nil {
                erro(at(ctx,a), "`%s` name already taken (%T).", name, a).debug(1)
                return
            }
        }
    }

    dec.project.opts = declOpts
    dec.backscope = l.Scope()
    l.useesExecuted = nil
    l.proj = dec.project
    l.scopes[0] = dec.project.scope
    if globe.main != nil && globe.main == l.proj && l.proj.name != "~" {
        for _, t := range globe.pairs {
            switch k := t.key.(type) {
            case *bareword, *barecomp:
                var name = k.string(ctx);
                //if name[0] == '.' { name = "project" + name }
                var d, a = l.def(l.Position(), name)
                if d == nil && a != nil { d = a.(*def) }
                d.set(ctx, defDecl, t.val)
            case flag:
                if false { warn(ctx, "%v: unknown flag: %v", l.proj, t).debug(1) }
            default:
                warn(ctx, "%v: unknown target (%s): %v", l.proj, typeof(t), t).debug(1)
            }
        }
    }

    for _, arg := range merge(l.loadArgs...) {
        switch t := arg.(type) {
        case *pair:
            var name = t.key.string(ctx)
            var d, a = l.def(t.key.Position(), name)
            if a != nil {
                var y bool
                if d, y = a.(*def); !y {
                    erro(ctx, "'%v' is not a Def (%T)", a, a).debug(1)
                    return
                }
            }
            if d != nil { d.val(ctx, t.val) }
            warn(ctx, "%v: %v", ident, t)
        }
    }

    if err := l.loadPlugin(ctx); err != nil {
        erro(ctx, "load plugin failed: %v", err).debug(1)
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
    if proj := l.proj; isConfigureproject(proj) {
        // skip...
    } else if obj := proj.resolve(ctx, ".autoload."+tag); obj == nil {
        // skip...
    } else if d, y := obj.(*def); !y {
        warnstack(ctx, 3, "%v: unsupported .auto: %T %v", proj, obj, obj).debug(1)
    } else if isTrivial(d.value) {
        // skip...
    } else if val := scalarize(d.value.expand(final{ctx})); isTrivial(val) {
        // skip...
    } else {
        var u = _universe(ctx)
        const ( o = true ; t = false ; s = "autoload" )
        if t && u.ddd == s { prompt(ctx, "%s: autoload - %s,\n", l.Position(), tag) }
        if o { l.include(ctx, includeOpts{}, val) }
        if t && u.ddd == s { prompt(ctx, "%s: autoload - %s.\n", l.Position(), tag) }
    }
}

func (l *loader) configure(ctx Context, linfo *loadinfo, ident *barecomp, identStr string, declared bool) (result bool) {
    if false { defer un(l_tracef(l_traverse, "configuration(%v)", ident)) }
    if s := l.proj.name; s == dotConfigure { return }

    var local bool
    var configure string
    var v = l.proj.opts.configure
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

        var globe = l.Globe()
        if loaded, res = globe.loaded[absPath]; loaded == nil { res = false }
        if !res { erro(ctx, "not loaded: %s (%s, dir=%v)", configure, absPath, isDir).debug(16) }
        return
    }

    var isDir bool
    var absPath string
    if filepath.IsAbs(configure) { if file := stat(ctx, configure); file.exists() {
        absPath, isDir = file.fullname(), file.info.IsDir()
    }} else if file := stat(ctx, configure, l.proj); file.exists() {
        absPath, isDir = file.fullname(), file.info.IsDir()
    }
    if absPath == "" && v != nil {
        if !local { absPath, isDir = _universe(ctx).search(ctx, linfo, configure) }
        if absPath == "" {
            erro(ctx, "%v: no such project: %s", l.proj, configure).debug(1)
        }
    }
    if absPath == "" { return } else
    if !load(absPath, isDir) {
        erro(ctx, "%v: configure not loaded: %s", l.proj, configure).debug(1)
        return
    }

    if name, _ := l.Scope().Lookup(dotConfigure).(*project); name == nil {
        if _, alt := l.Scope().projectname(ctx, dotConfigure, loaded); alt != nil {
            if val, y := alt.(*project); !y || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(1)
            }
        }
    }
    if l.proj.configure == loaded { return }
    if l.proj.configure != nil {
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

    var u = _universe(ctx)

    // Load configuration.sm after .configure was loaded.
    l.proj.configure = loaded // must set .configure first to get the correct configuration file
    l.proj.configurationFile = l.proj.configuration(ctx)
    if f := l.proj.configurationFile; f == nil {
        erro(ctx, "%v: nil configuration file", ident).debug(1)
        return
    } else if declared || u.commandline.configure {
        // u.configuration.clean[f] = struct{}{}
    } else if f.exists() || f.stat(ctx) != nil {
        defer func(t parseBits) { l.p.bits = t } (l.p.bits)
        l.p.bits |= parseIncludingConf
        l.include(ctx, includeOpts{ isConfigure:true }, f)
    }
    return true
}

func (l *loader) container(ctx Context, ident *barecomp, identStr string) (result bool) {
    ctx = at(ctx, ident.Position())
    if l.proj.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").debug(1)
            return
        }

        var u = _universe(ctx)

        // Looking for project specific .container module
        if f := stat(ctx, dotContainer, l.proj); f.exists() {
            if !l.loadDotContainer(ctx, ident, identStr, f) {
                //erro(ctx, "declare %s: %s/.container", name, l.proj.absPath)
            }
            if u.verbose {
                info(ctx, "%v for %s (%s)\n", f, l.proj.spec, l.proj).debug(1)
            }
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(ctx, l.proj.absPath, func(s string) bool {
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
    if l.proj == nil {
        return fmt.Errorf("no current project")
    } else if s := l.proj.name; s != identStr {
        return fmt.Errorf("current project is %s but %s", s, identStr)
    } else if l.proj != dec.project {
        return fmt.Errorf("project conflicts (%s, %s)", l.proj.name, dec.project.name)
    }

    l.scopes[0] = dec.backscope
    l.useesExecuted = dec.useesExecuted
    return
}

func (l *loader) resolve(ctx Context, value Value) (name string, result Value) {
    var pos = value.Position()
    if !pos.IsValid() { pos = l.Position() }
    if name = value.string(ctx); name != "" {
        if d := autoDef(ctx, name); d == nil {
            result = l.proj.resolve(ctx, name)
        } else {
            result = d
        }
    }
    return
}

func (l *loader) def(position Position, name string) (def *def, alt Object) {
    var scope = l.Scope()
    def, alt = scope.define(at(l, position), defVoid, name, nil)
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
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.source")) }
    if u.verbose { if ctx.Position().Filename == filename {
        info(ctx, "loading ...")
    } else {
        prompt(ctx, "%s:1:info: loading ...\n", filename)
        info(ctx, "loading %v", filename)
    }}

    defer trace(ctx)
    defer func(t time.Time, p *parser, m Mode) { if true { ctx = l.p.ctx(ctx) }
        if d := time.Now().Sub(t); d > u.slow {
            warnstack(ctx, 10, "%v: slow: %v (%v)", l.proj, d, u.slow).debug(2) //  → %s, filename
        } else if u.verbose {
            info(ctx, "loaded %v (%v)", filename, d).debug(2)
        }

        l.p, l.mode = p, m

        if true {
            // ...
        } else if err != nil {
            errostack(ctx, 3, "source error: %v", err).debug(1)
        } else if f == nil && res == nil {
            erro(ctx, "source not loaded: %s", filepath.Base(filename)).debug(1)
        }
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
        if opts.isConfigure {
            l.p.bits |= parseIncludingConf
        } else {
            l.p.bits &= ^parseIncludingConf
        }
    }

	var smod scanmode
	if l.mode&ParseComments != 0 {
		//smod = scanner.ScanComments
	}

    l.p.scanner.init(u.file(filename, text), text, smod,
        func(p Position, s string, a ...interface{}) {
            if a == nil { a = append(a, 4, 4) }
            note(at(ctx,p), "%s", s)
            erro(at(ctx,p), "scan=%v", l.p.scanner.scanstate).debug(a...)
        },
        func(p Position, s string, a ...interface{}) {
            if a == nil { a = append(a, 4, 1) }
            warn(at(ctx,p), "%s", s)
            warn(at(ctx,p), "scan=%v", l.p.scanner.scanstate).debug(a...)
        },
        func(p Position, s string, a ...interface{}) {
            if a == nil { a = append(a, 4, 1) }
            info(at(ctx,p), "%s", s)
            info(at(ctx,p), "scan=%v", l.p.scanner.scanstate).debug(a...)
        })

	l.p.next(true) // starts scanning

    if ctx = l.p.ctx(ctx); l.mode&parsingText != 0 {
        res = l.p.values(ctx)
    } else if f = l.p.file(ctx); f == nil {
        // Source is not a valid source file, returnning a valid but empty parsedFile
        defer l.closeScope(l.openScope(fmt.Sprintf("file %s", filename)))
        f = &parsedFile{ scope:l.Scope() }
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

    var list []os.FileInfo
    if list, err = fd.Readdir(-1); err != nil || len(list) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        err = fmt.Errorf("invalid package name %s", ident)
        return
    }

    defer l.closeScope(l.openScope("config "+pathname))

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
            if flush(ctx) > 0 { return }
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
            def.set(ctx, defConfDir, makeStrlit(ctx.Position(), s))
        } else if s != nil {
            erro(ctx, "Name `%s' already taken, not def (%T).", name, s)
            break ListLoop
        }
    }
    return
}

var loader_sources_bench = true

func (l *loader) sources(ctx Context, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*project) {
    var u = _universe(l.Context)

    defer trace(ctx)

    if loader_sources_bench { defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseParse || d > time.Second {
            note(ctx, "slow: %s (%v)", l.proj, d).debug(1)
        } else if debugSyntax(ctx, "sources") {
			note(ctx, "sources: %s (%v)", l.proj, time.Now().Sub(t)).debug(6)
		}
    }(time.Now())}

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

            var src, _, err = l.source(ctx, filename, nil, mode|parsingDir, nil)
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
                mod = &project{ name: name, scope: l.Scope() }
                mods[name] = mod
            }
        }
    }
    return
}

// loader.Load loads script from a file or source code (string, []byte).
func (l *loader) load(ctx Context, specName, absPath string, source interface{}) (result bool) {
    var u = _universe(ctx)
    if u.traceLaunch { defer un(l_trace(l_launch, "loader.load")) }

    var globe = l.Globe()
    defer func(t time.Time) {
        if d := time.Now().Sub(t); u.verboseLoads && d>1*time.Second {
            loaded, _ := globe.loaded[absPath]
            if l.proj == nil {
                prompt(ctx, "load (%15s) ⇒ %s (%s)\n", d, loaded, specName)
            } else {
                prompt(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, l.proj.name, loaded, specName)
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
            if val, ok := a.(*project); !ok || val == nil {
                erro(ctx, "`%v` name already taken (%T).", loaded, a)
            }
        }
        result = true
        return
    }

    var absDir, baseName = filepath.Split(absPath)
    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, baseName))

    var doc, _, err = l.source(ctx, absPath, source, parseMode, nil)
    if err != nil {
        erro(ctx, "load: %v", err).debug(1)
    } else if doc == nil {
        erro(ctx, "load: nil: %s", absPath).debug(1)
    } else {
        result = true
    }

    flush(l.Context)
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
    var globe = ctx.Globe()
    defer func(t time.Time, ver bool) {
        if specName == "." { specName = absDir }

        if d := time.Now().Sub(t); ver && d>1*time.Second { if l.proj == nil {
            note(ctx, "load (%15s) ⇒ %s (%s)\n", d, loadedProj, specName).debug(1)
        } else {
            note(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, l.proj.name, loadedProj, specName).debug(1)
        }}

        if loadedProj == nil { return }
        if globe.main == nil { globe.main = loadedProj }

        if proj := l.Scope().project; proj == nil {
            if false { erro(ctx, "%v: no owner project for %s", loadedProj.name, l.Scope()).debug(2) }
        } else if name, _ := proj.scope.Lookup(loadedProj.name).(*project); name == nil {
            if _, alt := proj.scope.projectname(at(ctx,pos), loadedProj.name, loadedProj); alt != nil {
                if val, y := alt.(*project); !y || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loadedProj.name, alt).debug(2)
                }
            }
        }
    } (time.Now(), u.verboseLoads)

    // Check loaded project.
    if loadedProj, okay = globe.loaded[absDir]; okay { return }

    defer restoreLoadingInfo(saveLoadingInfo(l, specName, absDir, ""))

    var mods map[string]*project
    if mods = l.sources(at(l, pos), absDir, filter, parseMode); mods == nil {
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
            for s, m := range globe.loaded { erro(ctx, "%v: %v", s, m) }
            errostack(ctx, 3, "%s not loaded (as %s)", specName, absDir).debug(10)
        }
    } else if loadedProj, okay = globe.loaded[absDir]; okay && loadedProj != nil {
        // Good!
    } else if filepath.Base(specName) != "@" {
        erro(ctx, "%s not loaded (as %s, implicit=%v)", specName, absDir, l.implicit).debug(1)
    }
    return
}

func (l *loader) file(ctx Context, filename string, source interface{}) (res bool) {
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

    var spec, _ = filepath.Rel(_workdir(l), path)

    var position Position
    position.Filename = spec
    return l.directory(at(ctx, position), spec, path, filter)
}

func (l *loader) text(ctx Context, filename string, text string) (res []Value) {
    if _universe(ctx).traceLaunch { defer un(l_trace(l_launch, "loader.text")) }

    defer func(saved *parser) { l.p = saved } (l.p)

    if g := l.Globe(); g.main == nil {
        l.scopes[0] = g.os.scope_
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
