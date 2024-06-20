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

    mainFileName = "do.smart"
    deprFileName = "build.smart"

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
    AllErrors
)

var parseMode = AllErrors

type get_parser struct{}
type is_loading_text  struct{}
type is_implicit_load struct{}

// implicit load, aka. via foo.bar.Baz (implicitly loads foo/bar for base)
type implicit_load_context struct{ Context }

type load_dir_context  struct{ Context }
type load_file_context struct{ Context }
type load_text_context struct{ Context }

func (p implicit_load_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_implicit_load: return true
	}
	return p.Context.do(ctx, op)
}

func (p load_text_context) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_loading_text: return true
	}
	return p.Context.do(ctx, op)
}

type declare struct {
    *project   // save loader.project -- the active project being loading
    p *parser  // save loader.p
    s []*scope // save loader.terminal.s
}

func _loader(c Context) *loader { return cast[*loader](c) }

type loader struct {
    terminal      // .s -> declare.s
    p *parser        // -> declare.p
    project *project // -> declare.project -- the current project

    declares map[string]*declare

    mode Mode

    verpre string // verbose prefix
}
func (l *loader) inner() Context { return &l.terminal }
func (l *loader) cast(t reflect.Type) Context {
    if reflect.TypeOf(l) == t { return l }
    return l.terminal.cast(t)
}
func (l *loader) do(ctx Context, op any) any {
    switch op.(type) {
	case is_implicit_load: return false
    case get_parser: return l.p
    case get_project: return l.project
    case get_position:
        if l.p != nil {
            return l.p.Position()
        } else if false {
            erro(ctx, "nil parser for position").debug()
            trace(ctx)
        }
    }
	return l.terminal.do(ctx, op)
}
func (l *loader) ts(t string) string {
    if l == nil { return "{=loader}" }
    return fmt.Sprintf("{=%s %v %v}", t, l.project, ts(l.Context))
}

type unilo struct{ *universe ; *loader }

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
            trace(ctx)
        } else {
            f(op, spec, val, name)
        }
    }}}
}
func (l unilo) usevars(ctx Context, user, usee *project) {
    var ddd = l.ddd == "use"
    usefor(ctx, user, func(op usevar, spec, val Value, name string) {
        var useDef *def
        if o := usee.Lookup("use."+name); o != nil {
            if d, y := o.(*def); y && d != nil { useDef = d } else {
                erro(ctx, "use.%s: nil def: %T %v", name, o, o).debug()
                trace(ctx)
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

// func (l unilo) _position() Position { return _position(l.loader) }
func (l unilo) scope() *scope { return l.loader.scope() }
func (l unilo) search(ctx Context, specName string) (absPath string, isDir bool) {
    if specName == "." {
        erro(ctx, "not possible to chain itself").debug()
        trace(ctx)
    }

    abs := filepath.IsAbs(specName)

    if abs || specName == "~" || specName == ".." || hasPrefix(specName, "~"+pathSep, "."+pathSep, ".."+pathSep) {
        var s = specName
        var sx string

        if !abs && l.project.absPath != "" {
            sx = filepath.Join(l.project.absPath, s)
            if a, e := filepath.Abs(sx); e == nil {
                s = a
            } else {
                erro(ctx, "abs: %v", e)
                trace(ctx)
            }
        }

        if fi, err := os.Stat(s); err == nil { return s, fi.IsDir() }

        sx = s + ".smart"
        if fi, err := os.Stat(sx); err == nil { return sx, fi.IsDir() }

        sx = s + ".sm"
        if fi, err := os.Stat(sx); err == nil { return sx, fi.IsDir() }
    } else {
        for _, base := range l.paths {
            var s string
            if filepath.IsAbs(base) {
                s = filepath.Join(base, specName)
            } else {
                s = filepath.Join(_workdir(ctx), base, specName)
            }
            if fi, err := os.Stat(s); err == nil && fi != nil {
                return s, fi.IsDir()
            }
        }
    }
    return
}

func (l unilo) use_spec(ctx Context, opts useOpts, specVal Value, params ...Value) (loaded *project) {
    var absPath, spec string
    var isDir, traveUseLoop bool
    if x, y := specVal.(*project); y {
        loaded = x
    } else if spec = specVal.string(ctx); spec == "" {
        erro(ctx, "empty spec: %v", ts(specVal)).debug()
        trace(ctx)
    } else if absPath, isDir = l.search(ctx, spec); absPath == "" {
        erro(ctx, "missing `%s` (in %v)", spec, l.paths).debug()
        trace(ctx)
    } else {
        loaded, y = l.globe.loaded[absPath]

        for ll := _loader(l.loader.Context); ll != nil; ll = _loader(ll.Context) {
            if ll.project.absPath == absPath {
                var s string // TODO: build load path
                // TODO: ll.opt.traveUseLoop
                erro(ctx, "%s: loop detected : %s", l.project, s).debug()
                trace(ctx)
            }
        }
    }

    defer func() {
        if loaded == nil {
            erro(ctx, "%v not loaded (%v,dir=%v)", spec, absPath, isDir).debug()
            trace(ctx)
        }

        var scope = l.project.scope
        if p, _ := scope.Lookup(loaded.name).(*project); p == nil {
            if _, alt := scope.projectname(ctx, loaded.name, loaded); alt != nil {
                if p, y := alt.(*project); !y || p == nil {
                    erro(ctx, "%s: name already taken : %s", loaded.name, ts(alt)).debug()
                    trace(ctx)
                }
            }
        }
    } ()

    if l.verboseImport {
        if /* len(l.loadStack) > 1 */ false {
            defer func(s string) { l.verpre = s } (l.verpre)
            l.verpre += "│"
        }
        if opts.reuse {
            prompt(ctx, "%s├┬→\"%s\" (reuse, %s)\n", l.verpre, spec, absPath)
        } else {
            prompt(ctx, "%s├┬→\"%s\" (%s)\n", l.verpre, spec, absPath)
        }
        defer func(t time.Time) {
            var name string
            var d = time.Now().Sub(t)//*time.Millisecond // µs, ms, s
            var ds = fmt.Sprintf("(%s)", d)
            if d>=1*time.Second { ds = fmt.Sprintf("▶%s◀",ds) }
            if loaded != nil { name = loaded.name }
            prompt(ctx, "%s├┴─\"%s\" ⇢ %s %s\n", l.verpre, spec, name, ds)
        } (time.Now())
    }

    if loaded != nil && !(/*opts.noVars || */opts.reuse) {
        if proj, res, isb := l.project.hasLoaded(ctx, loaded, traveUseLoop) ; isb {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v is already a base\n", l.project, spec)
            erro(ctx, "`%s` is already a base (proj=%s)", spec, proj)
            errostack(ctx, 10, "%v", ctx).debug()
            trace(ctx)
        } else if res {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v already imported by %v\n", l.project, spec, proj)
            erro(ctx, "'%s' already imported by '%s'", spec, proj)
            errostack(ctx, 10, "%v", ctx).debug()
            trace(ctx)
        }
    }

    var prev = l.project

    if loaded == nil {
        if isDir {
            if !l.directory(ctx, spec, absPath, nil) {
                erro(ctx, "load dir failed: %s : %s", spec, absPath).debug()
                trace(ctx)
            }
        } else {
            if !l.load(ctx, spec, absPath, nil) {
                erro(ctx, "load file failed: %s : %s", spec, absPath).debug()
                trace(ctx)
            }
        }
        if loaded, _ = l.globe.loaded[absPath]; loaded == nil {
            erro(ctx, "'%s' not loaded (%s)", spec, absPath).debug()
            trace(ctx)
        }
        if loaded == l.project {
            erro(ctx, "%v : overwrote by %v (dir=%v)", prev, loaded, isDir).debug()
            trace(ctx)
        }
    }

    if checkpoints {
        if prev != l.project {
            erro(ctx, "active project changed: %v -> %v, use %v", prev, l.project, loaded).debug()
            trace(ctx)
        }
    }

    // Check against the current load list before appending loaded.
    for _, use := range l.project.use.list {
        var up = use.project
        if loaded == up {
            if !opts.noVars && !opts.files {
                errostack(ctx, 10, "%v: using `%s` multiple times: %v", l.project, spec, l.project.use.list).debug()
                trace(ctx)
            }
            return
        }

        var proj *project
        var res, isb bool
        if false && loaded.opt.multiUseAllowed {
            // ...
        } else if proj, res, isb = loaded.hasLoaded(ctx, up, traveUseLoop); isb {
            if l.project.hasBase(up) {
                // common bases are fine
            } else {
                erro(ctx, "`%s` is already a base", spec).debug()
                trace(ctx)
            }
        } else if res && !use.opts.reuse && !up.opt.multiUseAllowed && !loaded.opt.multiUseAllowed {
            warn(ctx, "`%s` has already imported `%s` (from %s)", loaded, up, proj)
            if loaded != up { warn(at(ctx,loaded.position), "project %s", loaded) }
            if proj != up { warn(at(ctx,proj.position), "project %s", proj) }
            warn(at(ctx,up.position), "project %s", up).debug(6)
        }

        if proj, res, isb = up.hasLoaded(ctx, loaded, traveUseLoop); isb {
            warn(ctx, "`%s` is already base of `%s` (%s)", loaded, up, proj).debug()
        } else if res && !use.opts.reuse && !loaded.opt.multiUseAllowed {
            warn(ctx, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj)
            warnstack(ctx, 8, "`%s` has already been imported by `%s` (from %s)", loaded, up, proj).debug()
        }
    }

    if l.verboseImport {
        defer func(t time.Time) {
            prompt(ctx, "%s├┤ %s:import(%s) (%s)\n", l.verpre, l.project, spec, time.Now().Sub(t))
        } (time.Now()) //*time.Millisecond // µs, ms, s ┼
    }

    l.use_proj(ctx, opts, loaded, params...)
    return
}

const pluginDifferentVersionError = `plugin was built with a different version of package`
var numUpdatedPlugins = 0

func (l unilo) buildPlugin(ctx Context, s, src string) (err error) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.buildPlugin")) }

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

func (l unilo) loadPlugin(ctx Context) (err error) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.loadPlugin")) }
    if l.project == nil {
        erro(ctx, "current project is nil").debug()
        trace(ctx)
    }

    var g = stat(ctx, "smart.go", l.project)
    if g == nil { return /* smart.go was not presented */ }

    var src = g.string(ctx)
    s := strings.Replace(l.project.rel, "..", "_", -1)
    s = filepath.Join(filepath.Dir(joinTmpPath(ctx, "", "")), "plugins", s)

    var build = true

    so := stat(ctx, /*l.project.name*/"plugin", stat_dir{s}, stat_nonexist{true})
    if s = so.fullname(); s == "" {
        erro(at(ctx,so), "file '%v' has empty fullname", so)
        trace(ctx)
    } else if so.exists() && !l.buildPlugins {
        if so.info.ModTime().After(g.info.ModTime()) {
            build = false // Plugin already updated.
        }
    }
    if build { err = l.buildPlugin(ctx, s, src) }
    if err != nil { return }

    // Once plugin is opened, there's no need/way to close it.
    if l.project.ext.Plugin, err = plugin.Open(s); err == nil {
        var sym plugin.Symbol
        if sym, err = l.project.ext.Lookup("Init"); err != nil {
            erro(ctx, "nil plugin symbol Init").debug()
            trace(ctx)
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
                trace(ctx)
            }
        default:
            erro(ctx, "wrong plugin Init: %T", sym).debug()
            trace(ctx)
        }
    } else if strings.Contains(err.Error(), pluginDifferentVersionError) {
        err = l.buildPlugin(ctx, s, src)
    }
    return
}

func (l unilo) use_proj(ctx Context, opts useOpts, proj *project, params ...Value) (err error) {
    if l.verboseUsing {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            prompt(ctx, "use(%15s) %s ⇒ %v\n", d, l.project, l.project.use)
        } (time.Now())
    }

    if proj == l.project {
        erro(ctx, "%v: cannot use itself", proj).debug()
        trace(ctx)
    }

    if l.project.isUsingDirectly(proj) {
        return
    }

    // Add to the project using list, so that the use path is correct.
    if l.project.use.append(ctx, proj, params, opts); !opts.noVars {
        // aka.     XXX += $(use.XXX)
        // aka. use.XXX += $(use.XXX)
        l.usevars(ctx, l.project, proj)
        if 0 < len(opts.vars) {
            for _, v := range opts.vars {
                warn(at(ctx,v), "var: %T %v", v, v)
            }
            warn(ctx, "TODO: %d vars to import", len(opts.vars)).debug()
        }
    }
    return
}

type includeOpts struct {
    *clauseopts
    ifExists bool `if-exists,ifexists`
    isConfig bool
}
func (l unilo) include(ctx Context, specVal Value, opts includeOpts) {
    // defer flush(ctx)
    defer func(t time.Time) {
        if d := time.Now().Sub(t); d > l.slow {
            warn(ctx, "%v: slow: %v (%v)", l.project, d, l.slow).debug() //  → %s, filename
        } else if l.verbose {
            info(ctx, "included %v (%v)", specVal, d).debug()
        }
    } (time.Now())

    ctx = at(ctx, specVal)

    // Execute the rule entry to update include source.
    if x, y := specVal.(*rule); y && x != nil {
        if v, y := executeEntry(ctx, x); !y {
            erro(ctx, "%v: include entry failed : %s", x, ts(v)).debug()
            trace(ctx)
        }

        specVal = x.target
    }

    var spec, fullname string

    switch t := specVal.(type) {
    case *File:
        if !t.exists() { _ = t.stat(ctx) }
        if !t.exists() && opts.ifExists {
            return // ignore non-exists files
        }
        if fullname, spec = t.fullname(), t.ident(ctx); t.info == nil {
            prompt(ctx, "%v: no source file %v, %v\n", fullname, t, t.stat(ctx))
            erro(at(ctx,t), "%v: %v", _project(ctx), t).debug()
            trace(ctx)
        }
    default:
        if spec = specVal.string(ctx); spec == "" {
            erro(at(ctx,specVal), "include: empty string: %v", specVal).debug()
            trace(ctx)
        }

        var f = l.project.file(ctx, specVal)
        if f == nil {
            if filepath.IsAbs(spec) {
                f = stat(ctx, spec)
            } else {
                var d string
                var ll = _loader(l.loader.Context)
                if ll != nil { d = ll.project.absPath } else { d = l.project.absPath }
                f = stat(ctx, spec, stat_dir{d})
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

    if spec == "" {
        erro(ctx, "include: empty string: %v", specVal).debug()
        trace(ctx)
    }

    l.loader = &loader{terminal:terminal{parser_include_context{ctx, opts}, []*scope{}}}
    l.source(l.loader, fullname, nil)
    return
}

func (l unilo) openscope(comment string) (res []*scope) {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "loader.openscope")) }

    var pos Position
    if l.p == nil {
        pos = _position(l.loader.Context)
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

func (l unilo) closescope(scopes []*scope) {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "loader.closescope")) }

    if true {/* nooooooooooooooooooop */} else
    if scope := l.scope(); scope == nil {
        // nil scope
    } else if s := scope.comment; strings.HasPrefix(s, "dir ") {
        // Change the outer of dir scope to globe to avoid Finding symbols into the wrong context.
        l.globe.SetScopeOuter(scope)
    }

    l.s = scopes
}

// project example (base(var=value))
func (l unilo) bases(ctx Context, implicitBase string, params ...Value) (result bool) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.bases")) }

    // For &(foobar) set from loadArgs
    ctx = closure_with(ctx, l.s)

    var (
        implicitIndex int
        implicitBases []Value
    )
    if f := stat(ctx, dotBase, l.project); f != nil {
        if !f.info.IsDir() && (l.project.spec == dotBase /*|| l.project.spec == dotConfigure*/) {
            // skip the regular file '.base' to avoid self loading recursively
            // info(ctx, "%v", file).debug()
        } else {
            implicitBases = append(implicitBases, f)
        }
    }

    if ns := strings.Split(l.project.name, "."); len(ns) > 2 && ns[len(ns)-1] == "base" {
        var numBaseParams int
        for _, elem := range params {
            if x, y := elem.(*list); y && len(x.elems) == 1 { elem = x.elems[0] }
            if x, y := elem.(*argumented); y { elem = x.Value }
            if _, y := elem.(*pair); y { continue }
            numBaseParams += 1
        }
        if numBaseParams == 0 {
            var segs []Value
            for _, s := range ns[:len(ns)-2] {
                segs = append(segs, makeBareword(_position(ctx), s))
            }
            implicitBases = append(implicitBases, makePath(segs...))
            implicitBase = "" // discard the implicit base
        }
    }

    if implicitBase != "" {
        implicitIndex = len(implicitBases)
        implicitBases = append(implicitBases, _pathstr(ctx, implicitBase))
    }

ParamsLoop:
    for i, elem := range append(implicitBases, params...) {
        var ctx = at(ctx, elem)

        if x, y := elem.(*list); y && len(x.elems) == 1 {
            elem = x.elems[0]
        }
        if x, y := elem.(*argumented); y {
            elem, ctx = x.Value, &argumented_context{ctx, x.args}
        }
        if x, y := elem.(*pair); y {
            var ident = x.key
            var name string

            if name = ident.string(ctx); name == "" {
                erro(ctx, "empty base name: %v", ts(ident)).debug()
                trace(ctx)
            }
            if name[0] == '.' {
                ident = makeBarecomp(makeBareword(ident.Position(), "project"), ident)
            }

            // TODO: check the returned defs...
            _ = l.p.define_idents(ctx, ASSIGN, ident, x.val)
            continue
        }

        var (
            spec string
            specVal Value
        )
        if specVal = elem.expand(final{ctx}); specVal == nil {
            specVal = elem // okay!
        } else if true && indeterminate(ctx, specVal) {
            errostack(ctx, 5, "incomplete expand: %T %v ⇒ %T %v", elem, elem, specVal, specVal).debug()
            trace(ctx)
        } else if defs := specVal.defs(ctx); len(defs) > 0 {
            errostack(ctx, 5, "incomplete expand: %v ⇒ %v (defs=%v)", elem, specVal, defs).debug()
            trace(ctx)
        }

        if spec = specVal.string(ctx); spec == "" {
            erro(ctx, "%v: empty base name `%v` (%T)", l.project, specVal, specVal).debug()
            trace(ctx)
        } else if strings.Contains(spec, "//") {
            erro(ctx, "%v: invalid spec: %v in %T", l.project, elem, ctx)
            erro(ctx, "%v: invalid spec: %v -> %v", l.project, elem, specVal)
            erro(ctx, "%v: invalid spec: %v -> %v", l.project, elem, spec).debug()
            trace(ctx)
        } else if implicitBase != "" && spec == implicitBase {
            if i == implicitIndex {
                ctx = implicit_load_context{ctx}
            } else {
                erro(ctx, "%v: base '%v' already loaded implicitly", l.project, elem).debug()
                if false { break } else { continue }
            }
        }

        if n := flush(ctx); n > 0 {
            warn(ctx, "%v: %d errors: %v -> %v", l.project, n, elem, spec).debug()
            break
        }

        var (
            absPath string
            isDir bool
        )
        if f, y := toFile(elem); y && f.info != nil {
            absPath, isDir = f.fullname(), f.info.IsDir()
            if true { assert(filepath.IsAbs(absPath), "invalid abs path: %v", f) }
        } else if absPath, isDir = l.search(ctx, spec); absPath == "" {
            erro(ctx, "%v: search base failed: %v → %v", l.project, elem, spec).debug()
            trace(ctx)
        }

        for _, base := range l.project.bases {
            if base.absPath == absPath {
                //erro(ctx, "duplicated base: %v (in %v)", elem, l.project.bases)
                continue ParamsLoop
            }
        }

        if isDir {
            if !l.directory(ctx, spec, absPath, nil) {
                erro(ctx, "%v: %s : base dir not loaded", l.project, spec)
                trace(ctx)
            }
        } else {
            if !l.load(ctx, spec, absPath, nil) {
                erro(ctx, "%v: %s : base file not loaded", l.project, spec)
                trace(ctx)
            }
        }

        if x, y := l.globe.loaded[absPath]; y && x != nil {
            if l.project.hasBase(x) {
                continue
            }
            if l.project.bases == nil { // set .base to the first project
                l.project.projectname(ctx, ".base", x)
            }
            l.project.bases = append(l.project.bases, x)
        } else if truly(ctx, is_implicit_load{}) {
            warn(ctx, "implicit base '%s' not defined (as %s)", spec, absPath).debug()
        } else {
            erro(ctx, "project `%v`(%T: %s) not loaded (%s)", elem, elem, spec, absPath).debug()
            trace(ctx)
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

func (l unilo) loadDotContainer(ctx Context, ident Value, identStr string, file *File) (result bool) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.loadDotContainer")) }
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).debug()
        trace(ctx)
    } else if file.info.IsDir() {
        if !l.directory(ctx, dotContainer, file.fullname(), nil) {
            erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug()
            trace(ctx)
        }
    } else if !l.file(ctx, file.fullname(), nil) {
        erro(ctx, "%s: load '%v' failed (%s)", ident, dotContainer, file.fullname()).debug()
        trace(ctx)
    }

    if loaded, yes := l.globe.loaded[file.fullname()]; yes && loaded != nil {
        name, _ := l.scope().Lookup(loaded.name).(*project)
        if name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.project.name, file).debug()
            trace(ctx)
        } else {
            if false && l.verboseLoads {
                prompt(ctx, "smart: %v (%v)\n", name, file.fullname())
            }

            var opts useOpts
            // TODO: parse the useOpts
            l.use_proj(ctx, opts, loaded)

            result = true
        }
    }
    return
}

func isConfigureproject(proj *project) bool {
    return proj == nil ||
        proj.name == dotConfigure ||
        proj.name == "configure" ||
        proj.name == "configure.base"
}

func (l unilo) autoload(ctx Context, tag string) {
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
        const ( o = true ; t = false ; s = "autoload" )
        if t && l.ddd == s { prompt(ctx, "%s: autoload - %s,\n", l.p.Position(), tag) }
        if o { l.include(ctx, val, includeOpts{}) }
        if t && l.ddd == s { prompt(ctx, "%s: autoload - %s.\n", l.p.Position(), tag) }
    }
}

func (l unilo) configure(ctx Context, ident Value, identStr string, declared bool) (result bool) {
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

    var loaded *project
    var load = func(absPath string, isDir bool) (res bool) {
        if isDir {
            if !l.directory(ctx, configure, absPath, nil) { return }
        } else {
            if !l.file(ctx, absPath, nil) { return }
        }

        if loaded, res = l.globe.loaded[absPath]; loaded == nil { res = false }
        if !res {
            erro(ctx, "not loaded: %s (%s, dir=%v)", configure, absPath, isDir).debug()
            trace(ctx)
        }
        return
    }

    var isDir bool
    var absPath string
    if filepath.IsAbs(configure) {
        if f := stat(ctx, configure); f.exists() {
            absPath, isDir = f.fullname(), f.info.IsDir()
        }
    } else if f := stat(ctx, configure, l.project); f.exists() {
        absPath, isDir = f.fullname(), f.info.IsDir()
    }

    if absPath == "" && v != nil {
        if !local {
            absPath, isDir = l.search(ctx, configure)
        }
        if absPath == "" {
            erro(ctx, "%v: no such project: %s", l.project, configure).debug()
            trace(ctx)
        }
    }

    if absPath == "" {
        return
    }
    if !load(absPath, isDir) {
        erro(ctx, "%v: configure not loaded: %s", l.project, configure).debug()
        trace(ctx)
    }

    if name, _ := l.scope().Lookup(dotConfigure).(*project); name == nil {
        if _, alt := l.scope().projectname(ctx, dotConfigure, loaded); alt != nil {
            if val, y := alt.(*project); !y || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug()
                trace(ctx)
            }
        }
    }
    if l.project.configure == loaded { return }
    if l.project.configure != nil {
        erro(ctx, ".configure already specified").debug()
        trace(ctx)
    }

    var opts = useOpts{}
    for _, proj := range loaded.usees(true, false, false, false) {
        if err := l.use_proj(ctx, opts, proj); err != nil { // see usevars
            erro(ctx, "using '%v' failed: %v", proj, err).debug()
            trace(ctx)
        }
    }

    // Load configuration.sm after .configure was loaded.
    l.project.configure = loaded // must set .configure first to get the correct configuration file
    l.project.configuration = l.project._configuration(ctx)
    if f := l.project.configuration; f == nil {
        erro(ctx, "%v: nil configuration file", ident).debug()
        trace(ctx)
    } else if declared || l.commandline.configure {
        // l.configuration.clean[f] = struct{}{}
    } else if f.exists() || f.stat(ctx) != nil {
        l.include(ctx, f, includeOpts{isConfig:true})
    }
    return true
}

func (l unilo) container(ctx Context, ident Value, identStr string) (result bool) {
    ctx = at(ctx, ident)

    if l.project.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").debug()
            trace(ctx)
        }

        // Looking for project specific .container module
        if f := stat(ctx, dotContainer, l.project); f.exists() {
            if !l.loadDotContainer(ctx, ident, identStr, f) {
                //erro(ctx, "declare %s: %s/.container", name, l.project.absPath)
            }
            if l.verbose {
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

func (l unilo) closecurrent(ctx Context, name string) {
    var x, y = l.declares[name]
    if x == nil || !y {
        erro(ctx, "undeclared project: %v", name).debug()
        trace(ctx)
    }

    if l.project == nil {
        erro(ctx, "current project unset").debug()
        trace(ctx)
    }

    if l.project.name != name {
        erro(ctx, "current project is %s, not %s", l.project, name).debug()
        trace(ctx)
    }

    if l.project != x.project {
        erro(ctx, "project conflicts (%s, %s)", l.project.name, x.project.name).debug()
        trace(ctx)
    }

    l.p, l.s = x.p, x.s
}

// If src != nil, source_bytes converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, source_bytes returns
// the result of reading the file specified by filename.
func source_bytes(ctx Context, filename string, source ...any) (_ []byte, _ bool) {
    if 0 < len(source) {
        var n int
        var buf bytes.Buffer
        for _, src := range source {
            if src == nil { continue } else { n += 1 }

            var e error
            switch s := src.(type) {
            case        string: _, e = buf.Write([]byte(s))
            case        []byte: _, e = buf.Write(s)
            case *bytes.Buffer: _, e = buf.Write(s.Bytes())
            case     io.Reader: _, e = io.Copy(&buf, s)
            default:
                erro(ctx, "invalid source : %v", ts(src)).debug()
                trace(ctx)
            }

            if e != nil {
                erro(ctx, "copy bytes (%s) failed : %v", typeof(src), e).debug()
                trace(ctx)
            }
        }
        if 0 < n { return buf.Bytes(), false }
    }
    if t, e := ioutil.ReadFile(filename); e == nil {
        return t, false
    } else if _, y := e.(*fs.PathError); y {
        return nil, true
    } else {
        erro(ctx, "%v", e).debug()
        trace(ctx)
        return
    }
}

func (l unilo) source(ctx Context, filename string, src any) (res Value) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.source")) }
    if l.verbose {
        if _position(ctx).Filename == filename {
            info(ctx, "loading ...")
        } else {
            prompt(ctx, "%s:1:info: loading ...\n", filename)
            info(ctx, "loading %v", filename)
        }
    }

    defer func(t time.Time, p *parser) {
        if l.p == nil {
            erro(ctx, "nil parser ; %v", p).debug()
            trace(ctx)
        }

        if true && l.p != nil { ctx = at(ctx, l.p) }

        if d := time.Now().Sub(t); d > l.slow {
            warnstack(ctx, 10, "%v: slow: %v (%v)", l.project, d, l.slow).debug(2) // → %s, filename
        } else if l.verbose {
            info(ctx, "loaded %v (%v)", filename, d).debug(2)
        }

        l.p = p
    } (time.Now(), l.p)

    var opts, _ = do(ctx, getParseIncOpts{}).(*includeOpts)
    var text, patherror = source_bytes(ctx, filename, src)
    if patherror && (opts != nil && !opts.ifExists) {
        prompt(ctx, "%v: no such source file\n", filename)
        errostack(ctx, 3).debug()
        trace(ctx)
    }

    if text == nil { return }

    l.p = &parser{}

	var smod scanmode
	if l.mode&ParseComments != 0 {
		// smod = scanner.ScanComments
	}

    l.p.scanner.init(l.fset.AddFile(filename, -1, len(text)), text, smod,
        func(p Position, s string, a ...any) {
            if a == nil { a = append(a, 4, 4) }
            note(at(ctx,p), "%s", s)
            erro(at(ctx,p), "scan=%v", l.p.scanner.scanstate).debug(a...)
            trace(ctx)
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

    if truly(ctx, is_loading_text{}) {
        return ease(ctx, l.values(ctx))
    } else if l.parse_file(ctx) {
        return l.project
    }

    erro(ctx, "%v", filename).debug()
    trace(ctx)
    return
}

func (l unilo) _source(ctx Context, filename string) (_ bool) {
    if false {
        var p Position
        p.Filename, p.Line = filename, 1
        ctx = at(ctx, p)
    }

    var res = l.source(ctx, filename, nil)

    if i := flush(ctx); 0 < i {
        s := filepath.Base(filename)
        erro(ctx, "got %d errors in file '%s'", i, s).debug()
        trace(ctx)
    }

    if res == nil {
        erro(ctx, "parse failed").debug()
        trace(ctx)
    }

    if x, y := res.(*project); !y {
        erro(ctx, "non-project: %v", ts(res)).debug()
        trace(ctx)
    } else if _, y = l.declares[x.name] ; !y {
        erro(ctx, "%v: declared incorrectly", x).debug()
        trace(ctx)
    }

    return true
}

// ParseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (l unilo) config_dir(ctx Context, pathname, linked string) (err error) {
    var fd *os.File // Directory of the destination.
    if fd, err = os.Open(linked); err != nil { return }
    defer fd.Close()

    var fs []os.FileInfo
    if fs, err = fd.Readdir(-1); err != nil || len(fs) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        erro(ctx, "invalid package name %s", ident).debug()
        trace(ctx)
    }

	var sof, _ = filepath.Rel(workBaseDir, pathname)
    defer l.closescope(l.openscope("config "+sof))

    var scope = l.scope()

    ctx = at(ctx, l.p)

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
            if err = l.config_dir(ctx, filepath.Join(pathname, name), fullname); err != nil {
                erro(ctx, "parse config failed: %v", err).debug()
                trace(ctx)
            }
            if 0 < flush(ctx) { return } else { continue }
        }

        d, a := scope.set(ctx, name, defConfDir)

        if a != nil && a != d {
            erro(ctx, "declare project: %v", name).debug()
            trace(ctx)
        } else if d == nil {
            erro(ctx, "%v", name).debug()
            trace(ctx)
        }

        var v []byte
        if v, err = ioutil.ReadFile(fullname); err != nil {
            erro(ctx, "%v", err).debug()
            trace(ctx)
        }

        var s = string(v)
        if !utf8.ValidString(s) {
            erro(ctx, "%s: invalid UTF8 content", fullname)
            trace(ctx)
        }

        d.set(ctx, defConfDir, makeStrlit(_position(ctx), s))
    }
    return
}

var loader_sources_bench = true

func (l unilo) sources(ctx Context, path string, filter func(os.FileInfo) bool) (_ bool) {
    if loader_sources_bench {
        defer func(t time.Time) {
            if d := time.Now().Sub(t); l.verboseParse || d > time.Second {
                note(ctx, "slow: %s (%v)", l.project, d).debug()
            } else if debugSyntax(ctx, "sources") {
                note(ctx, "sources: %s (%v)", l.project, time.Now().Sub(t)).debug(6)
            }
        } (time.Now())
    }

    fd, err := os.Open(path)
    if err != nil {
        erro(ctx, "%v", err).debug()
        trace(ctx)
    }

    defer fd.Close()

    fis, err := fd.Readdir(-1)
    if err != nil {
        erro(ctx, "readdir: %v", err).debug()
        trace(ctx)
    }
    if len(fis) == 0 {
        erro(ctx, "no files underneath: %s", path).debug()
        trace(ctx)
    }

    var first = fis[0]
    for i := 1; i < len(fis); i += 1 {
        var s = fis[i].Name()
        if s == mainFileName || (s == deprFileName && first.Name() != mainFileName) {
            fis[0], fis[i] = fis[i], first
        }
    }

	var sof, _ = filepath.Rel(workBaseDir, path)
    defer l.closescope(l.openscope("dir "+sof))

    // FIXES: use globe scope as outer to avoid chaining with the other unrelated projects.
    // It's important to name resolving for not interfering with other projects.
    l.scope().outer = l.globe.scope

    for _, d := range fis {
        var name, mo = d.Name(), d.Mode()
        if name == "" || name == configuration_sm || strings.HasPrefix(name, ".#") ||
            !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm")) { continue }

        if !mo.IsRegular() || filter != nil && filter(d) { continue }

        filename := filepath.Join(path, name)
        linked, _ := _readlink(ctx, filename, d)

        if false && (name == "configure.smart" || name == "configure.sm") && (linked != "" || mo.IsDir()) {
            // hasConfDir = true // TODO: remove ConfigDir feature
            if l.config_dir(ctx, filepath.Dir(filename), linked) != nil { return }
            continue
        }

        if !l._source(load_dir_context{ctx}, filename) { return }
    }
    return true
}

// loader.Load loads script from a file or source code (string, []byte).
func (l unilo) load(ctx Context, spec, absPath string, source any) (result bool) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.load")) }

    defer flush(ctx)

    defer func(t time.Time) {
        if d := time.Now().Sub(t); l.verboseLoads && d>1*time.Second {
            loaded, _ := l.globe.loaded[absPath]
            if l.project == nil {
                prompt(ctx, "load (%15s) ⇒ %s (%s)\n", d, loaded, spec)
            } else {
                prompt(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, loaded, spec)
            }
        }
    } (time.Now())

    if absPath == "" {
        erro(ctx, "no such module `%s' (in paths %v)", spec, l.paths)
        trace(ctx)
    } else if !filepath.IsAbs(absPath) {
        erro(ctx, "invalid abs name `%s' (%s)", absPath, spec)
        trace(ctx)
    }

    // Check loaded project.
    if loaded, yes := l.globe.loaded[absPath]; yes {
        if _, a := l.scope().projectname(ctx, loaded.name, loaded); a != nil {
            if val, ok := a.(*project); !ok || val == nil {
                erro(ctx, "`%v` name already taken (%T).", loaded, a)
                trace(ctx)
            }
        }
        result = true
        return
    }

    // var absDir, baseName = filepath.Split(absPath)
    if l.project != nil /* && _loader(l.loader.Context) != nil */ {
        l.loader = &loader{terminal:terminal{ctx, []*scope{}}}
        ctx = l.loader
    }

    var res = l.source(ctx, absPath, source)
    if res == nil {
        erro(ctx, "load: nil: %s", absPath).debug()
        trace(ctx)
    }

    _, result = res.(*project)
    return
}

func (l unilo) directory(ctx Context, spec, absDir string, filter func(os.FileInfo) bool) (okay bool) {
    if !filepath.IsAbs(absDir) {
        erro(ctx, "needs absolute dir `%s' (%s)", absDir, spec).debug()
        trace(ctx)
    }

    var pos = positionForDir(absDir)

    var loaded *project
    defer func(t time.Time, ver bool) {
        if spec == "." { spec = absDir }

        if d := time.Now().Sub(t) ; ver && 1*time.Second < d {
            if p := l.project ; p != nil {
                note(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, p.name, loaded, spec).debug()
            } else {
                note(ctx, "load (%15s) ⇒ %s (%s)\n", d, loaded, spec).debug()
            }
        }

        if loaded == nil { return }

        if l.globe.main == nil { l.globe.main = loaded }

        if proj := l.scope().project; proj == nil {
            if false { erro(ctx, "%v: no owner project for %s", loaded.name, l.scope()).debug(2) }
        } else if name, _ := proj.Lookup(loaded.name).(*project); name == nil {
            if _, alt := proj.projectname(at(ctx,pos), loaded.name, loaded); alt != nil {
                if val, y := alt.(*project); !y || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).debug(2)
                    trace(ctx)
                }
            }
        }
    } (time.Now(), l.verboseLoads)

    // Check previously loaded project.
    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        return
    }

    ctx = at(ctx, pos)

    var lo = l
    if l.project != nil /* && _loader(l.loader.Context) != nil */ {
        lo.loader = &loader{terminal:terminal{ctx, []*scope{}}}
        ctx = lo.loader
    }
    if !lo.sources(ctx, absDir, filter) {
        erro(ctx, "failed parsing module: %s", spec).debug()
        trace(ctx)
    }

    if len(lo.declares) == 0 && filepath.Base(spec) != "@" {
        if truly(ctx, is_implicit_load{}) {
            warn(ctx, "%s not loaded (as %s, implicitly)", spec, absDir).debug()
            return true // okay for implicit loading
        } else {
            for s, m := range l.globe.loaded { erro(ctx, "%v: %v", s, m) }
            errostack(ctx, 3, "%s not loaded (as %s)", spec, absDir).debug()
            trace(ctx)
        }
    }

    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        return // Good!
    }

    if filepath.Base(spec) == "@" {
        return // Okay!
    }

    erro(ctx, "%s not loaded", spec).debug()
    trace(ctx)
    return
}

func (l unilo) file(ctx Context, filename string, source any) (res bool) {
    var dir, base = filepath.Split(filename)
    var spec string

    switch base {
    case dotBase, dotConfigure: spec = base
    default: spec, _  = filepath.Rel(l.workdir, dir)
    }

    var pos Position
    pos.Filename = filename
    return l.load(load_file_context{at(ctx, pos)}, spec, filename, source)
}

func (l unilo) text(ctx Context, filename string, text string) (res Value) {
    if l.globe.main == nil {
        l.s[0] = l.globe.os.scope
    } else {
        l.s[0] = l.globe.main.scope
    }

    var pos Position
    pos.Filename = filename
    return l.source(load_text_context{at(ctx, pos)}, filename, text)
}
