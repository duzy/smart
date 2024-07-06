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
type implicit_load struct{ Context }

type load_dir  struct{ Context }
type load_file struct{ Context }
type load_text struct{ Context }

func (p implicit_load) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case is_implicit_load: return true
	}
	return p.Context.do(ctx, op)
}

func (p load_text) do(ctx Context, op any) (_ any) {
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
    case get_parser:  return l.p
    case get_project: return l.project
    case get_position:
        if l.p != nil {
            return l.p.Position()
        } else if false {
            erro(ctx, "nil parser for position").trace()
        }
    }
	return l.terminal.do(ctx, op)
}
func (l *loader) ts(t string) string {
    return fmt.Sprintf("{=%s %v %v}", t, l.project, ts(l.Context))
}

type unilo struct{ *universe ; *loader }

type useopts struct {
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
    for _, u := range u {
        for _, v := range merge(u.value) {
            if t, y := v.(*def); y && t != nil {
                vals = append(vals, merge(t.value)...)
            } else {
                vals = append(vals, v)
            }
        }
    }
    if len(vals) == 0 {
        return
    }
    if d.append(ctx, vals...); uo.unique {
        d.value = call(ctx, "unique", uo.remainder, merge(d.value)...)
    }
}
func usefor(ctx Context, user *project, f func(usevar, Value, Value, string)) {
    if o := user.resolve(ctx, "use.*"); o != nil {
        if d, y := o.(*def); y && d != nil {
            for _, spec := range merge(d.value) {
                var op usevar
                var ctx, val = at(ctx, spec), spec
                if x, y := spec.(*argumented); y {
                    val = x.Value
                    op.remainder = parseOpts(final{ctx}, &op, x.args...)
                }
                if name := val.string(ctx); name == "" {
                    if c := user.configure; c != nil {
                        note(ctx, "%v", ts(c.resolve(ctx, "use.*")))
                    }
                    erro(at(ctx,val), "%v: empty use spec: %v", user, ts(spec)).trace()
                } else {
                    f(op, spec, val, name)
                }
            }
        }
    }
}
func (l unilo) usevars(ctx Context, user, usee *project) {
    var ddd = l.ddd == "use"
    usefor(ctx, user, func(op usevar, spec, val Value, name string) {
        var useDef *def
        if o := usee.Lookup("use."+name); o != nil {
            if d, y := o.(*def); y && d != nil { useDef = d } else {
                erro(ctx, "use.%s: nil def: %T %v", name, o, o).trace()
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
                dd = append(dd, nonTrivialDefsFromBase(ctx, user, useDef.ident(ctx))...)
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
                if dd == nil { dd = append(dd, nonTrivialDefsFromBase(ctx, user, useDef.ident(ctx))...) }
                dd = append(dd, nonTrivialDefsFromBase(ctx, user, name)...)
            }
            op.apply(closure_with(ctx, user.scope), d, append(dd, useDef)...)
        }
    })
    if ddd { note(ctx, "%v ⇒ %v ; %v", user, usee, user.resolve(ctx, "use.*")).debug(5) }
}

func nonTrivialDefsFromBase(ctx Context, p *project, name string) (dd []*def) {
    for _, base := range p.bases {
        d, y := base.resolve(ctx, name).(*def)
        if y && d != nil && !isTrivial(d.value) {
            dd = append(dd, d)
        }
    }
    return
}

func (l unilo) scope() *scope { return l.loader.scope() }
func (l unilo) search(ctx Context, specName string) (absPath string, isDir bool) {
    if specName == "." {
        erro(ctx, "not possible to chain itself").trace()
    }

    var abs = filepath.IsAbs(specName)

    if abs || specName == "~" || specName == ".." || has_prefix(specName, "~"+pathSep, "."+pathSep, ".."+pathSep) {
        var s = specName
        var sx string

        if !abs && l.project.absPath != "" {
            sx = filepath.Join(l.project.absPath, s)
            if x, e := filepath.Abs(sx); e == nil {
                s = x
            } else {
                erro(ctx, "abs: %v", e).trace()
            }
        }

        if x, e := os.Stat(s); e == nil { return s, x.IsDir() }

        sx = s + ".smart"
        if x, e := os.Stat(sx); e == nil { return sx, x.IsDir() }

        sx = s + ".sm"
        if x, e := os.Stat(sx); e == nil { return sx, x.IsDir() }
    } else {
        var wd = _workdir(ctx)
        for _, base := range l.paths {
            var s = filepath.Join(base, specName)
            if !filepath.IsAbs(base) { s = filepath.Join(wd, s) }
            if x, e := os.Stat(s); e == nil { return s, x.IsDir() }
        }
    }
    return
}

func (l unilo) use_spec(ctx Context, opts useopts, specVal Value, params ...Value) (loaded *project) {
    var absPath, spec string
    var isDir, traveUseLoop bool
    if x, y := specVal.(*project); y {
        loaded = x
    } else if spec = specVal.string(ctx); spec == "" {
        erro(ctx, "empty spec: %v", ts(specVal)).trace()
    } else if absPath, isDir = l.search(ctx, spec); absPath == "" {
        erro(ctx, "missing `%s` (in %v)", spec, l.paths).trace()
    } else {
        loaded, y = l.globe.loaded[absPath]

        for ll := _loader(l.loader.Context); ll != nil; ll = _loader(ll.Context) {
            if ll.project.absPath == absPath {
                var s string // TODO: build load path
                // TODO: ll.opt.traveUseLoop
                erro(ctx, "%s: loop detected : %s", l.project, s).trace()
            }
        }
    }

    defer func() {
        if loaded == nil {
            erro(ctx, "%v not loaded (%v,dir=%v)", spec, absPath, isDir).trace()
        }

        var scope = l.project.scope
        if p, _ := scope.Lookup(loaded.name).(*project); p == nil {
            if _, alt := scope.projectname(ctx, loaded.name, loaded); alt != nil {
                if p, y := alt.(*project); !y || p == nil {
                    erro(ctx, "%s: name already taken : %s", loaded.name, ts(alt)).trace()
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
        if proj, res, isb := l.project.has_loaded(ctx, loaded, traveUseLoop) ; isb {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v is already a base\n", l.project, spec)
            erro(ctx, "`%s` is already a base (proj=%s)", spec, proj)
            errostack(ctx, 10, "%v", ctx).trace()
        } else if res {
            // NOTE: proj could be nil
            prompt(ctx, "%v: %v already imported by %v\n", l.project, spec, proj)
            erro(ctx, "'%s' already imported by '%s'", spec, proj)
            errostack(ctx, 10, "%v", ctx).trace()
        }
    }

    var prev = l.project

    if loaded == nil {
        if isDir {
            l.directory(ctx, spec, absPath, nil)
        } else {
            l.load(ctx, spec, absPath, nil)
        }
        if loaded, _ = l.globe.loaded[absPath]; loaded == nil {
            erro(ctx, "'%s' not loaded (%s)", spec, absPath).trace()
        }
        if loaded == l.project {
            erro(ctx, "%v : overwrote by %v (dir=%v)", prev, loaded, isDir).trace()
        }
    }

    if checkpoints {
        if prev != l.project {
            erro(ctx, "active project changed: %v -> %v, use %v", prev, l.project, loaded).trace()
        }
    }

    // Check against the current load list before appending loaded.
    for _, use := range l.project.use.list {
        var up = use.project
        if loaded == up {
            if !opts.noVars && !opts.files {
                errostack(ctx, 10, "%v: using `%s` multiple times: %v", l.project, spec, l.project.use.list).trace()
            }
            return
        }

        var proj *project
        var res, isb bool
        if false && loaded.opt.multiUseAllowed {
            // ...
        } else if proj, res, isb = loaded.has_loaded(ctx, up, traveUseLoop); isb {
            if l.project.has_base(up) {
                // common bases are fine
            } else {
                erro(ctx, "`%s` is already a base", spec).trace()
            }
        } else if res && !use.opts.reuse && !up.opt.multiUseAllowed && !loaded.opt.multiUseAllowed {
            warn(ctx, "`%s` has already imported `%s` (from %s)", loaded, up, proj)
            if loaded != up { warn(at(ctx,loaded.position), "project %s", loaded) }
            if proj != up { warn(at(ctx,proj.position), "project %s", proj) }
            warn(at(ctx,up.position), "project %s", up).debug(6)
        }

        if proj, res, isb = up.has_loaded(ctx, loaded, traveUseLoop); isb {
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
        erro(ctx, "current project is nil").trace()
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
            erro(ctx, "nil plugin symbol Init").trace()
        }
        if sym == nil {
            return // no initialization (optional)
        }
        switch init := sym.(type) {
        case func(Context) (error):
            if err = init(ctx); err == nil {
                return
            } else {
                erro(ctx, "plugin Init: %v", err).trace()
            }
        default:
            erro(ctx, "wrong plugin Init: %T", sym).trace()
        }
    } else if strings.Contains(err.Error(), pluginDifferentVersionError) {
        err = l.buildPlugin(ctx, s, src)
    }
    return
}

func (l unilo) use_proj(ctx Context, opts useopts, proj *project, params ...Value) (err error) {
    if l.verboseUsing {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            prompt(ctx, "use(%15s) %s ⇒ %v\n", d, l.project, l.project.use)
        } (time.Now())
    }

    if proj == l.project {
        erro(ctx, "%v: cannot use itself", proj).trace()
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
        if z, y := executeEntry(ctx, x); !y {
            erro(ctx, "%v: include entry failed : %s", x, ts(z)).trace()
        }

        specVal = x.target
    }

    var spec, fullname string

    switch t := specVal.(type) {
    case *File:
        if !t.exists() { _ = t.stat(ctx) }
        if !t.exists() && opts.ifExists { return } // ignore non-exists files
        if fullname, spec = t.fullname(), t.ident(ctx); t.info == nil {
            prompt(ctx, "%v: no source file %v, %v\n", fullname, t, t.stat(ctx))
            erro(at(ctx,t), "%v: %v", _project(ctx), t).trace()
        }
    default:
        if spec = specVal.string(ctx) ; spec == "" {
            erro(at(ctx,specVal), "include: empty string: %v", specVal).trace()
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
        erro(ctx, "include: empty string: %v", specVal).trace()
    }

    l.loader = &loader{
        terminal: terminal{parse_include{ctx, opts}, []*scope{l.scope()}},
        project: l.project,
    }
    l.source(l.loader, fullname, nil)
    return
}

func (l unilo) openscope(comment string) (res []*scope) {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "openscope")) }

    var pos Position
    if l.p == nil {
        pos = _position(l.loader.Context)
    } else {
        pos = l.p.Position()
    }

    s := newscope(pos, l.scope(), l.project, comment)
    res, l.s = l.s, append([]*scope{s}, l.s...)
    return
}

func (l unilo) closescope(scopes []*scope) {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "closescope")) }

    if true {
        // nooooooooooooooooooop
    } else if scope := l.scope(); scope == nil {
        // nil scope
    } else if s := scope.comment; strings.HasPrefix(s, "dir ") {
        // Change the outer of dir scope to globe to avoid Finding symbols into the wrong context.
        l.globe.SetScopeOuter(scope)
    }

    l.s = scopes
}

// project example (base(var=value))
func (l unilo) bases(ctx Context, implicitBase_ string, params ...Value) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.bases")) }

    // For &(foobar) set from command line args
    if true { ctx = closure_with(ctx, l.s) }

    var implicitBases []Value

    if f := stat(ctx, dotBase, l.project) ; f != nil {
        if !f.info.IsDir() && (l.project.spec == dotBase /*|| l.project.spec == dotConfigure*/) {
            // skip the regular file .base to avoid self loading recursively
        } else {
            implicitBases = append(implicitBases, f)
        }
    }

    if ss := strings.Split(l.project.name, ".") ; len(ss) > 2 && ss[len(ss)-1] == "base" {
        var numBaseParams int
        for _, elem := range params {
            if x, y := elem.(*list); y && len(x.elems) == 1 { elem = x.elems[0] }
            if x, y := elem.(*argumented); y { elem = x.Value }
            if _, y := elem.(*pair); y { continue }
            numBaseParams += 1
        }
        if numBaseParams == 0 {
            var segs []Value
            for _, s := range ss[:len(ss)-2] {
                segs = append(segs, makeBareword(_position(ctx), s))
            }
            implicitBases = append(implicitBases, makePath(segs...))
            implicitBase_ = "" // discard the implicit base
        }
    }

    var implicitIndex int
    if  implicitBase_ != ""  {
        implicitIndex = len(implicitBases)
        implicitBases = append(implicitBases, _pathstr(ctx, implicitBase_))
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
            if s := x.key.string(ctx) ; s == "" {
                erro(ctx, "empty base name: %v", ts(x.key)).trace()
            } else if s[0] == '.' {
                x.key = makeBarecomp(makeBareword(x.key.Position(), "project"), x.key)
            }

            // TODO: check the returned defs...

            _ = l.p.define_idents(ctx, ASSIGN, x.key, x.val)
            continue
        }

        var spec string
        var specVal Value
        if specVal = elem.expand(final{ctx}); specVal == nil { specVal = elem }
        if checkpoints && truly(ctx, is_test_mode{}) {
            l.bases_check_param(ctx, implicitBase_, i, elem, specVal)
        }

        if indeterminate(ctx, specVal) {
            errostack(ctx, 10, "incomplete spec: %v ⇒ %v", elem, specVal).trace()
        }

        if spec = specVal.string(ctx) ; spec == "" {
            erro(ctx, "%v: empty base name: %v", l.project, ts(specVal)).trace()
        } else if strings.Contains(spec, "//") {
            note(ctx, "%v: invalid spec: %v → %v", l.project, elem, specVal)
            note(ctx, "%v: invalid spec: %v → %v", l.project, elem, spec)
            errostack(ctx, 5).trace()
        } else if implicitBase_ != "" && spec == implicitBase_ {
            if i == implicitIndex {
                ctx = implicit_load{ctx}
            } else {
                erro(ctx, "%v: implicit base '%v' already loaded", l.project, elem).trace()
            }
        }

        var absPath string
        var isDir bool

        if x, y := toFile(elem); y && x.info != nil {
            absPath, isDir = x.fullname(), x.info.IsDir()
            if !filepath.IsAbs(absPath) {
                erro(ctx, "%v: not abs path: %v → %v", l.project, elem, spec).trace()
            }
        } else if absPath, isDir = l.search(ctx, spec); absPath == "" {
            erro(ctx, "%v: search base failed: %v → %v", l.project, elem, spec).trace()
        }

        for _, base := range l.project.bases {
            if base.absPath == absPath {
                if true {
                    erro(ctx, "duplicated base: %v (in %v)", elem, l.project.bases).trace()
                }
                continue ParamsLoop
            }
        }

        if isDir {
            l.directory(ctx, spec, absPath, nil)
        } else {
            l.load(ctx, spec, absPath, nil)
        }

        if checkpoints { l.bases_check(ctx, implicitIndex, implicitBase_, absPath, isDir, elem) }
    }

    usefor(ctx, l.project, func(op usevar, _, _ Value, name string) {
        var us = "use." + name
        var d, a = l.project.set(ctx, us, defVoid)
        if d == nil && a != nil { d, _ = a.(*def) }
        if d == nil { return }
        op.apply(closure_with(ctx, l.project.scope), d, nonTrivialDefsFromBase(ctx, l.project, us)...)
    })
    return
}

func (l unilo) loadDotContainer(ctx Context, ident Value, identStr string, file *File) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.loadDotContainer")) }
    if file.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, file.fullname()).trace()
    } else if file.info.IsDir() {
        l.directory(ctx, dotContainer, file.fullname(), nil)
    } else {
        l.file(ctx, file.fullname(), nil)
    }

    if x, y := l.globe.loaded[file.fullname()]; y && x != nil {
        if name, _ := l.scope().Lookup(x.name).(*project) ; name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.project.name, file).trace()
        }

        var opts useopts
        // TODO: parse the useopts
        l.use_proj(ctx, opts, x)
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
    if p := l.project ; !isConfigureproject(p) {
        if o := p.resolve(ctx, ".autoload."+tag); o != nil {
            if x, y := o.(*def); !y {
                warnstack(ctx, 3, "%v: unsupported .auto: %v", p, ts(o)).debug()
            } else {
                var v = scalarize(x.value.expand(final{ctx}))
                if !isTrivial(v) { l.include(ctx, v, includeOpts{}) }
            }
        }
    }
}

func (l unilo) configure(ctx Context, ident Value, identStr string, declared bool) {
    if false { defer un(l_tracef(l_traverse, "configuration(%v)", ident)) }
    if l.project.name == dotConfigure { return }

    var local, isDir bool
    var absPath, configure string

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer l.configure_check(ctx, ident, &absPath, &configure)
    }

    if v := l.project.opt.configure; v != nil {
        if x, y := v.(*boolean); y {
            if !x.bool { return }
            configure = "configure"
        } else if configure = v.string(ctx); configure == "" {
            erro(at(ctx,v), "empty configure spec: %v", ts(v)).trace()
        }
    }

    if configure == "." {
        configure, local = "configure", true
    } else if configure == "" {
        if false { erro(ctx, "empty configure spec").trace() }
        return
    }

    if filepath.IsAbs(configure) {
        if f := stat(ctx, configure); f.exists() {
            absPath, isDir = f.fullname(), f.info.IsDir()
        }
    } else if f := stat(ctx, configure, l.project); f.exists() {
        absPath, isDir = f.fullname(), f.info.IsDir()
    }

    if absPath == "" && l.project.opt.configure != nil {
        if !local {
            absPath, isDir = l.search(ctx, configure)
        }
        if absPath == "" {
            erro(ctx, "%v: no such project: %s", l.project, configure).trace()
        }
    }

    if absPath == "" {
        if l.project.opt.configure != nil {
            erro(ctx, "%v: missing the default .configure", l.project).trace()
        }
        return
    } else if isDir {
        l.directory(ctx, configure, absPath, nil)
    } else {
        l.file(ctx, absPath, nil)
    }

    var loaded *project
    if loaded, _ = l.globe.loaded[absPath]; loaded == nil {
        erro(ctx, "not loaded: %s (%s, dir=%v)", configure, absPath, isDir).trace()
    }

    if name, _ := l.scope().Lookup(dotConfigure).(*project); name == nil {
        if _, alt := l.scope().projectname(ctx, dotConfigure, loaded); alt != nil {
            if val, y := alt.(*project); !y || val == nil {
                erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).trace()
            }
        }
    }
    if l.project.configure == loaded { return }
    if l.project.configure != nil {
        erro(ctx, ".configure already specified").trace()
    }

    var opts = useopts{}
    for _, proj := range loaded.usees(true, false, false, false) {
        if err := l.use_proj(ctx, opts, proj); err != nil { // see usevars
            erro(ctx, "using '%v' failed: %v", proj, err).trace()
        }
    }

    // Load configuration.sm after .configure was loaded.
    l.project.configure = loaded // must set .configure first to get the correct configuration file
    l.project.configuration = l.project._configuration(ctx)
    if f := l.project.configuration; f == nil {
        erro(ctx, "%v: nil configuration file", ident).trace()
    } else if declared || l.commandline.configure {
        // l.configuration.clean[f] = struct{}{}
    } else if f.exists() || f.stat(ctx) != nil {
        l.include(ctx, f, includeOpts{isConfig:true})
    }
}

func (l unilo) container(ctx Context, ident Value, identStr string) {
    ctx = at(ctx, ident)

    if l.project.name != dotContainer {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").trace()
        }

        // Looking for project specific .container module
        if f := stat(ctx, dotContainer, l.project); f.exists() {
            l.loadDotContainer(ctx, ident, identStr, f)
            if l.verbose {
                info(ctx, "%v for %s (%s)\n", f, l.project.spec, l.project).debug()
            }
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(ctx, l.project.absPath, func(s string) bool {
            var f = stat(ctx, dotContainer, stat_dir{filepath.Join(s, ".smart")})
            if f.exists() {
                l.loadDotContainer(ctx, ident, identStr, f)
            }
            return false
        })
    }
    return
}

// If src != nil, load_source_bytes converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, load_source_bytes returns
// the result of reading the file specified by filename.
func load_source_bytes(ctx Context, filename string, source ...any) (_ []byte, _ bool) {
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
                erro(ctx, "invalid source : %v", ts(src)).trace()
            }

            if e != nil {
                erro(ctx, "copy bytes (%s) failed : %v", typeof(src), e).trace()
            }
        }
        if 0 < n { return buf.Bytes(), false }
    }
    if t, e := ioutil.ReadFile(filename); e == nil {
        return t, false
    } else if _, y := e.(*fs.PathError); y {
        return nil, true
    } else {
        erro(ctx, "%v", e).trace()
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
            erro(ctx, "nil parser ; %v", p).trace()
        }

        if true && l.p != nil { ctx = at(ctx, l.p) }

        if d := time.Now().Sub(t); d > l.slow {
            warnstack(ctx, 10, "%v: slow: %v (%v)", l.project, d, l.slow).debug(2) // → %s, filename
        } else if l.verbose {
            info(ctx, "loaded %v (%v)", filename, d).debug(2)
        }

        l.p = p
    } (time.Now(), l.p)

    var opts = try[*includeOpts](ctx, parse_inc_opts{})
    var text, patherror = load_source_bytes(ctx, filename, src)
    if patherror && (opts != nil && !opts.ifExists) {
        prompt(ctx, "%v: no such source file\n", filename)
        errostack(ctx, 3).trace()
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
    }

    if l.parse_file(ctx) {
        return l.project
    }

    if true { return }

    if filepath.IsAbs(filename) {
        prompt(ctx, "%s: no project parsed\n", filename)
        errostack(ctx, 5).trace()
    } else {
        errostack(ctx, 5, "%v", filename).trace()
    }
    return
}

// ParseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (l unilo) config_dir(ctx Context, pathname, linked string) (err error) {
    var fd *os.File // Directory of the destination.
    if fd, err = os.Open(linked); err != nil {
        erro(ctx, "%v", err).trace()
        return
    }

    defer fd.Close()

    var fs []os.FileInfo
    if fs, err = fd.Readdir(-1); err != nil || len(fs) == 0 { return }

    var ident = filepath.Base(pathname)
    if ident == "_" {
        erro(ctx, "invalid package name %s", ident).trace()
    }

	var sof, _ = filepath.Rel(workBaseDir, pathname)
    defer l.closescope(l.openscope("config "+sof))

    var scope = l.scope()

    ctx = at(ctx, l.p)

    for _, f := range fs {
        var name = f.Name()
        if has_prefix(name, "~") || hasSuffix(name, ".#", ".smart", ".sm") {
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
                erro(ctx, "parse config failed: %v", err).trace()
            }
            if 0 < flush(ctx) { return } else { continue }
        }

        d, a := scope.set(ctx, name, defConfDir)

        if a != nil && a != d {
            erro(ctx, "declare project: %v", name).trace()
        } else if d == nil {
            erro(ctx, "%v", name).trace()
        }

        var v []byte
        if v, err = ioutil.ReadFile(fullname); err != nil {
            erro(ctx, "%v", err).trace()
        }

        var s = string(v)
        if !utf8.ValidString(s) {
            erro(ctx, "%s: invalid UTF8 content", fullname)
        }

        d.set(ctx, defConfDir, makeStrlit(_position(ctx), s))
    }
    return
}

func nonsource(name string, mo os.FileMode) (_ bool) {
    if  !mo.IsRegular() || name == "" || name == configuration_sm || strings.HasPrefix(name, ".#") ||
        !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm")) { return true }
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
        erro(ctx, "%v", err).trace()
    }

    defer fd.Close()

    fis, err := fd.Readdir(-1)
    if err != nil {
        erro(ctx, "readdir: %v", err).trace()
    }
    if len(fis) == 0 {
        erro(ctx, "no files underneath: %s", path).trace()
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
        if nonsource(name, mo) || filter != nil && filter(d) { continue }

        var filename = filepath.Join(path, name)
        var linked,_ = _readlink(ctx, filename, d)

        if false && (name == "configure.smart" || name == "configure.sm") && (linked != "" || mo.IsDir()) {
            // hasConfDir = true // TODO: remove ConfigDir feature
            if l.config_dir(ctx, filepath.Dir(filename), linked) != nil { return }
            continue
        }

        if false {
            var p Position
            p.Filename, p.Line = filename, 1
            ctx = at(ctx, p)
        }

        var res = l.source(load_dir{ctx}, filename, nil)

        // if i := flush(ctx); 0 < i {
        //     s := filepath.Base(filename)
        //     erro(ctx, "got %d errors in file '%s'", i, s).trace()
        // }

        if res == nil {
            erro(ctx, "parse failed").trace()
        }

        if x, y := res.(*project); !y {
            erro(ctx, "non-project: %v", ts(res)).trace()
        } else if _, y = l.declares[x.name] ; !y {
            erro(ctx, "%v: declared incorrectly", x).trace()
        }
    }
    return true
}

// unilo.load loads script from a file or source code
func (l unilo) load(ctx Context, spec, absPath string, source any) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.load")) }
    if false { defer flush(ctx) }

    defer func(t time.Time) {
        if d := time.Now().Sub(t) ; l.verboseLoads && d>1*time.Second {
            if x, _ := l.globe.loaded[absPath] ; l.project == nil {
                prompt(ctx, "load (%15s) ⇒ %s (%s)\n", d, x, spec)
            } else {
                prompt(ctx, "load (%15s) %s ⇒ %s (%s)\n", d, l.project.name, x, spec)
            }
        }
    } (time.Now())

    if absPath == "" {
        erro(ctx, "no such module `%s' (in paths %v)", spec, l.paths).trace()
    } else if !filepath.IsAbs(absPath) {
        erro(ctx, "invalid abs name `%s' (%s)", absPath, spec).trace()
    }

    // Check loaded project.
    if p, y := l.globe.loaded[absPath]; y {
        if _, a := l.scope().projectname(ctx, p.name, p); a != nil {
            if x, y := a.(*project); !y || x == nil {
                erro(ctx, "name already taken: %v (%s).", p, typeof(a)).trace()
            }
        }
        do(ctx, set_base{p})
        return
    }

    if l.project != nil /* && _loader(l.loader.Context) != nil */ {
        l.loader = &loader{terminal:terminal{ctx, []*scope{l.scope()}}}
        ctx = l.loader
    }

    var res = l.source(ctx, absPath, source)
    if false && res == nil {
        prompt(ctx, "%s: nil project parsed for %s", absPath, spec)
        errostack(ctx, 3).trace()
    }
    return
}

func (l unilo) directory(ctx Context, spec, absDir string, filter func(os.FileInfo) bool) {
    if !filepath.IsAbs(absDir) {
        erro(ctx, "needs absolute dir `%s' (%s)", absDir, spec).trace()
    }

    var pos = positionForDir(absDir)

    var okay bool
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
            if false { erro(ctx, "%v: no owner project for %s", loaded.name, l.scope()).trace() }
        } else if name, _ := proj.Lookup(loaded.name).(*project); name == nil {
            if _, alt := proj.projectname(at(ctx,pos), loaded.name, loaded); alt != nil {
                if val, y := alt.(*project); !y || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).trace()
                }
            }
        }
    } (time.Now(), l.verboseLoads)

    defer l.directory_check(ctx, spec, absDir)

    // Check previously loaded project.
    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        do(ctx, set_base{loaded})
        return
    }

    ctx = at(ctx, pos)

    var lo = l
    if l.project != nil /* && _loader(l.loader.Context) != nil */ {
        lo.loader = &loader{terminal:terminal{ctx, []*scope{}}}
        ctx = lo.loader
    }
    if !lo.sources(ctx, absDir, filter) {
        erro(ctx, "failed parsing module: %s", spec).trace()
    }

    if len(lo.declares) == 0 && filepath.Base(spec) != "@" {
        if truly(ctx, is_implicit_load{}) {
            warn(ctx, "%s not loaded (as %s, implicitly)", spec, absDir).debug()
            return // okay for implicit loading
        } else {
            for s, m := range l.globe.loaded { erro(ctx, "%v: %v", s, m) }
            errostack(ctx, 3, "%s not loaded (as %s)", spec, absDir).trace()
        }
    }

    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        return // Good!
    }

    if filepath.Base(spec) == "@" {
        return // Okay!
    }

    erro(ctx, "%s not loaded", spec).trace()
    return
}

func (l unilo) file(ctx Context, filename string, source any) {
    var dir, base = filepath.Split(filename)
    var spec string

    switch base {
    case dotBase, dotConfigure: spec = base
    default: spec, _  = filepath.Rel(l.workdir, dir)
    }

    var pos Position
    pos.Filename = filename
    l.load(load_file{at(ctx, pos)}, spec, filename, source)
}

func (l unilo) text(ctx Context, filename string, text string) Value {
    if l.globe.main == nil {
        l.s[0] = l.globe.os.scope
    } else {
        l.s[0] = l.globe.main.scope
    }

    var pos Position
    pos.Filename = filename
    return l.source(load_text{at(ctx, pos)}, filename, text)
}
