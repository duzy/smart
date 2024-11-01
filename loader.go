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
    dot_base      = ".base"
    dot_configure = ".configure"
    dot_container = ".container"

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

type is_implicit_load struct{}

// implicit load, e.g. via foo.bar.Baz (implicitly loads foo/bar for base of Baz)
type load_implicit struct{ Context }
func (p load_implicit) do(ctx Context, op any) any {
    switch op.(type) {
    case is_implicit_load: return true
    }
    return p.Context.do(ctx, op)
}

type abs_path struct{}
type abs_ctx struct{ Context ; abs string }
func (p *abs_ctx) ts(string) string {
    return "{=abs "+bases(2, p.abs, true)+" "+ts(p.Context)+"}"
}
func (p *abs_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case abs_path: return p.abs
    case get_position:
        var pos, _ = p.Context.do(ctx, op).(Position)
        if !pos.valid() { pos.Filename, pos.Line = p.abs, 1 }
        return pos
    }
    return p.Context.do(ctx, op)
}

func _abs_ctx(ctx Context, abs string) Context {
    if do(ctx, abs_path{}) == abs { return ctx }
    return &abs_ctx{ctx, abs}
}

type declare struct {
    *project  // save loader.project -- the active project being loading
    p *parser // save loader.p
    s *scope  // save loader.term.s
}

func _loader(c Context) *loader { return cast[*loader](c) }

type loader struct {
    term             // .s -> declare.s
    p *parser        // -> declare.p
    project *project // -> declare.project -- the current project
    promptEnteringDirectory bool

    declares map[string]*declare

    mode Mode

    verpre string // verbose prefix
}
func (l *loader) inner() Context { return &l.term }
func (l *loader) cast(t reflect.Type) Context {
    if reflect.TypeOf(l) == t { return l }
    return l.term.cast(t)
}
func (l *loader) ts(string) (s string) {
    s = "{=loader"
    if l.project != nil { s += " " + l.project.name }
    if l.Context != nil { s += " " + ts(l.Context) }
    s += "}"
    return
}
func (l *loader) do(ctx Context, op any) any {
    switch op.(type) {
    case is_implicit_load: return false
    case get_parser:  return l.p
    case get_project: return l.project
    case get_position: if l.p != nil { return l.p.Position() }
    case get_scope: if false { return l.project.scope }
    case get_closure_scopes:
        var t, _ = l.term.do(ctx, op).([]*scope)
        return append(t, l.project.scope)
    }
	return l.term.do(ctx, op)
}

type ul struct{ *universe ; *loader }

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
                var val = spec
                if x, y := spec.(*argumented); y {
                    val = x.Value
                    op.remainder = parse_opts(final{ctx}, &op, x.args...)
                }
                if name := val.string(ctx); name == "" {
                    if c := user.configure; c != nil {
                        note(ctx, "%v", ts(c.resolve(ctx, "use.*")))
                    }
                    erro(ctx, "%v: empty use spec: %v", user, ts(spec)).trace()
                } else {
                    f(op, spec, val, name)
                }
            }
        }
    }
}
func (l ul) usevars(ctx Context, user, usee *project) {
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

func (l ul) scope() *scope { return l.loader.scope }
func (l ul) search(ctx Context, spec string) (absPath string, isDir bool) {
    if checkpoints && l.project != nil && l.project.name == "variant.bootstrap" {
        defer func() {
            if absPath == "" {
                note(ctx, "%v → %s %v", spec, absPath, isDir).debug(2)
            }
        } ()
    }

    if spec == "." {
        erro(ctx, "self-search is not possible").trace()
    } else if filepath.IsAbs(spec) {
        var s = spec
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }

        s = spec + ".smart"
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }

        s = spec + ".sm"
        if x, y := os.Stat(s); y == nil { return s, x.IsDir() }
    } else if spec == "~" || strings.HasPrefix(spec, "~") {
        erro(ctx, "%v : wrong spec : %s (tilde not allowed)", l.project, spec).trace()
    } else if spec == ".." || has_prefix(spec, "."+pathSep, ".."+pathSep) {
        var s = spec
        var sx string

        if t := l.project.absPath; t != "" {
            if x, e := os.Stat(t); e != nil {
                erro(ctx, "%v", e).trace()
            } else if !x.IsDir() {
                t = filepath.Dir(t)
            }

            sx = filepath.Join(t, s)

            if x, e := filepath.Abs(sx); e != nil {
                erro(ctx, "%v", e).trace()
            } else {
                s = x
            }
        }

        if x, e := os.Stat(s); e == nil { return s, x.IsDir() }

        sx = s + ".smart"
        if x, e := os.Stat(sx); e == nil { return sx, x.IsDir() }

        sx = s + ".sm"
        if x, e := os.Stat(sx); e == nil { return sx, x.IsDir() }
    } else {
        for _, base := range l.paths {
            var s = filepath.Join(base, spec)
            if !filepath.IsAbs(base) { s = filepath.Join(l.workdir, s) }
            if x, e := os.Stat(s); e == nil { return s, x.IsDir() }
        }
    }
    return
}

func (l ul) use_spec(ctx Context, opts useopts, specVal Value, params ...Value) (loaded *project) {
    var absPath, spec string
    var isDir, traveUseLoop bool
    if x, y := specVal.(*project); y {
        loaded = x
    } else if spec = specVal.string(ctx); spec == "" {
        erro(pc(ctx,specVal), "empty spec: %v", ts(specVal)).trace()
    } else if absPath, isDir = l.search(ctx, spec); absPath == "" {
        erro(pc(ctx,specVal), "missing `%s` (in %v)", spec, l.paths).trace()
    } else {
        loaded, y = l.globe.loaded[absPath]

        for ll := _loader(l.loader.Context); ll != nil; ll = _loader(ll.Context) {
            if ll.project.absPath == absPath {
                erro(pc(ctx,specVal), "%s: loop detected", l.project).trace()
            }
        }
    }

    defer func() {
        if loaded == nil {
            if false {
                erro(ctx, "%v not loaded (%v,dir=%v)", spec, absPath, isDir).trace()
            }
            return
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
        if cc := pc(ctx, specVal); isDir {
            l.directory(cc, spec, absPath, nil)
        } else {
            l.file(cc, spec, absPath, nil)
        }
        if loaded, _ = l.globe.loaded[absPath]; loaded == nil {
            erro(ctx, "%s not loaded (%s)", spec, absPath).trace()
        }
        if loaded == l.project {
            erro(ctx, "%v : overwrote by %v (dir=%v)", prev, loaded, isDir).trace()
        }
    }

    if checkpoints && prev != l.project {
        erro(ctx, "active project changed: %v → %v, use %v", prev, l.project, loaded).trace()
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
            if loaded != up { warn(ctx, "project %s", loaded) }
            if proj != up { warn(ctx, "project %s", proj) }
            warn(ctx, "project %s", up).debug(6)
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

func (l ul) buildPlugin(ctx Context, s, src string) (err error) {
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

func (l ul) loadPlugin(ctx Context) (err error) {
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
        erro(ctx, "file '%v' has empty fullname", so)
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

func (l ul) use_proj(ctx Context, opts useopts, proj *project, params ...Value) (err error) {
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
                warn(ctx, "var: %T %v", v, v)
            }
            warn(ctx, "TODO: %d vars to import", len(opts.vars)).debug()
        }
    }
    return
}

func (l ul) spec_file(ctx Context, specVal Value) (res *file, spec, fullname string) {
    switch t := specVal.(type) {
    case *file:
        if !t.exists() { _ = t.stat(ctx) }
        return t, t.ident(ctx), t.fullname()
    default:
        if spec = specVal.string(ctx) ; spec == "" {
            erro(ctx, "empty string: %v", tv(specVal)).trace()
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

        if f != nil {
            res, fullname = f, f.fullname()
        }
        return
    }
}

type get_include_opts struct{}
type get_include_spec struct{}
type is_flat_mode struct{}

type include_opts struct {
    *clauseopts
    ifExists bool `if-exists,ifexists`
}

type include_ctx struct {
    Context
    o include_opts
    p Position
    spec string
}
func (i include_ctx) ts(t string) string {
	return "{="+t+" "+i.spec+" "+ts(i.Context)+"}"
}
func (i include_ctx) do(ctx Context, op any) (_ any) {
	switch op.(type) {
    case get_position: if i.p.valid() { return i.p }
	case get_include_opts: return &i.o
    case get_include_spec: return i.spec
	case is_flat_mode    : return true
	}
	return i.Context.do(ctx, op)
}

func (l ul) include(ctx Context, doc *commentGroup, g *clauseopts, _ int) {
	if l_traverse.enabled { defer un(l_trace(l_traverse, "include")) }

	var opts = include_opts{ clauseopts: g }
	if va := parse_opts(ctx, &opts, g.remainder...); len(va) > 0 {
		erro(ctx, "unknown opts: %v", va).trace()
	}

	if len(g.spec) < 1 {
		erro(ctx, "expect include file: %v", g.spec).trace()
	}

	var val = g.spec[0].expand(final{ctx})

	if l.p.spaces(ctx); l.p.tok == COLON {
		switch val.(type) {
		case *file, *strlit, *strcomp: // escape from file searching
		default: if f := l.project.file(ctx, val); f != nil { val = f }
		}
		val = l.rule(ctx, nil, []Value{val}) // this should return a Rule
	}

    if g.skip { return }

    ctx = pc(ctx, g.spec[0])

    // Execute the rule entry to update include source.
    if x, y := val.(*rule); y && x != nil {
        if z, y := execute_entry(ctx, x); !y {
            erro(ctx, "%v: include entry failed : %s", x, tv(z)).trace()
        }

        val = x.target
    }

    var f, spec, fullname = l.spec_file(ctx, val)
    if (f == nil || !f.exists()) && opts.ifExists {
        return // ignore non-exists files
    }

    if spec == "" || fullname == "" {
        erro(ctx, "empty string: %v", tv(val)).trace()
    } else {
        var p, s = val.Position(), l.trimSpecPath(ctx, spec)
        l.source(include_ctx{ctx, opts, p, s}, fullname, nil)
    }
    return
}

func (l ul) openscope(comment string) *scope {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "openscope")) }

    var pos Position
    if l.p == nil {
        pos = _position(l.loader.Context)
    } else {
        pos = l.p.Position()
    }

    var t = &term{} ; *t = l.term
    l.term = term{t, newscope(pos, l.scope(), l.project, comment)}
    return t.scope
}

func (l ul) closescope(s *scope) {
    if false && l.traceLaunch { defer un(l_trace(l_launch, "closescope")) }
    if x, y := l.term.Context.(*term); y {
        var ctx Context = l.loader
        if l.p != nil {
            ctx = pc(l.loader, l.p.Position())
        }
        if x == &l.term {
            erro(ctx, "conflict term: %s", x.comment).trace()
        }
        if x.scope != s {
            erro(ctx, "conflict scope: %s != %s", x.comment, s.comment).trace()
        }
        l.term = *x
    }
}

// project example (base(var=value))
func (l ul) bases(ctx Context, implicitBase string, params ...Value) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.bases")) }

    // For &(foobar) set from command line args
    if true { ctx = closure_with(ctx, l.scope) }

    var implicitBases []Value

    if f := stat(ctx, dot_base, l.project) ; f != nil {
        if !f.info.IsDir() && (l.project.spec == dot_base /*|| l.project.spec == dot_configure*/) {
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
                segs = append(segs, _word(_position(ctx), s))
            }
            implicitBases = append(implicitBases, makePath(segs...))
            implicitBase = "" // discard the implicit base
        }
    }

    var implicitIndex int
    if  implicitBase != ""  {
        implicitIndex = len(implicitBases)
        implicitBases = append(implicitBases, _pathstr(ctx, implicitBase))
    }

paramsloop:
    for i, elem := range append(implicitBases, params...) {
        if x, y := elem.(*list); y && len(x.elems) == 1 {
            elem = x.elems[0]
        }
        if x, y := elem.(*argumented); y {
            elem, ctx = x.Value, &argumented_ctx{ctx, x}
        }
        if x, y := elem.(*pair); y {
            erro(pc(ctx,x), "use -set(%v) instead", x).trace()
        }

        var spec string
        var specVal Value
        if specVal = elem.expand(final{ctx}); specVal == nil { specVal = elem }
        if checkpoints && truly(ctx, is_test_mode{}) {
            l.bases_check_param(ctx, implicitBase, i, elem, specVal)
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
        } else if implicitBase != "" && spec == implicitBase {
            if i == implicitIndex {
                ctx = load_implicit{ctx}
            } else {
                erro(ctx, "%v: implicit base '%v' already loaded", l.project, elem).trace()
            }
        }

        var abs string
        var isDir bool

        if x, y := to_file(elem); y && x.info != nil {
            abs, isDir = x.fullname(), x.info.IsDir()
        } else {
            abs, isDir = l.search(ctx, spec)
        }

        for _, base := range l.project.bases {
            if base.absPath == abs {
                erro(ctx, "duplicated base: %v : %v → %v (in %v)", base, elem, spec).trace()
                continue paramsloop
            }
        }

        if cc := _abs_ctx(ctx, abs); isDir {
            l.directory(cc, spec, abs, nil)
        } else {
            l.file(cc, spec, abs, nil)
        }

        if checkpoints && truly(ctx, is_test_mode{}) {
            l.bases_check_i(ctx, i, implicitIndex, implicitBase, abs, isDir, elem)
        }
    }

    if checkpoints && truly(ctx, is_test_mode{}) {
        l.bases_check(ctx, implicitIndex, implicitBase)
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

func filespec(workdir, filename string) (spec string) {
    switch dir, base := filepath.Split(filename); base {
    case dot_base, dot_configure: spec = base
    default: spec, _ = filepath.Rel(workdir, dir)
    }
    return
}

func (l ul) dot_container(ctx Context, ident Value, identStr string, f *file) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.dot_container")) }

    if s := f.fullname(); f.info == nil {
        erro(ctx, "%s: file not exists: %s", ident, s).trace()
    } else if cc := pc(ctx, ident); f.info.IsDir() {
        l.directory(cc, dot_container, s, nil)
    } else {
        l.file(cc, filespec(l.workdir, s), s, nil)
    }

    if x, y := l.globe.loaded[f.fullname()]; y && x != nil {
        if name, _ := l.scope().Lookup(x.name).(*project) ; name == nil {
            erro(ctx, "%v: %v: `dock` is not a project", l.project.name, f).trace()
        }

        var opts useopts
        // TODO: parse the useopts
        l.use_proj(ctx, opts, x)
    }
    return
}

func is_configure_project(proj *project) bool {
    return proj == nil ||
        proj.name == dot_configure ||
        proj.name == "configure" ||
        proj.name == "configure.base"
}

type is_autoload struct{}

type autoload_ctx struct {
    Context
    p Position
    v Value
}
func (a autoload_ctx) ts(t string) string {
	return "{="+t+" "+bases(2, a.v.String(), true)+" "+ts(a.Context)+"}"
}
func (a autoload_ctx) do(ctx Context, op any) (_ any) {
	switch op.(type) {
    case get_position: if a.p.valid() { return a.p }
    case is_autoload: return true
	case is_flat_mode: return true
	}
	return a.Context.do(ctx, op)
}

func (l ul) autoload(ctx Context, tag string) {
    if !is_configure_project(l.project) {
        if d := l.project.def(ctx, ".autoload."+tag); d != nil && d.value != nil {
            for _, v := range merge(d.value.expand(final{ctx})) {
                if isTrivial(v) {
                    continue
                } else if f, s, t := l.spec_file(ctx, v); f == nil || !f.exists() {
                    continue//erro(ctx, "no such source file: %v → %v", tv(d.value), tv(v)).trace()
                } else if s == "" || t == "" {
                    continue//erro(ctx, "empty string: %v → %v", tv(d.value), tv(v)).trace()
                } else {
                    l.source(autoload_ctx{ctx,l.p.Position(),v}, t, nil)
                }
            }
        }
    }
}

type is_config_mode struct{}

type configure_ctx struct {
    Context
    abs, configure string
    local, isDir bool
    configuration *file
    p *project
}
func (cc *configure_ctx) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case declared_project:
        if t.absPath == cc.abs {
            cc.p = t.project
            return
        }
    case abs_path: return cc.abs
	case is_config_mode: if cc.configuration != nil { return true }
	case is_flat_mode: if cc.configuration != nil { return true }
    }
    return cc.Context.do(ctx, op)
}

func (l ul) configuration(ctx Context, ident Value, identStr string) {
    if false { defer un(l_tracef(l_traverse, "configuration(%v)", ident)) }
    if l.project.name == dot_configure { return }

    var cc = configure_ctx{Context:ctx}

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer l.configuration_check(&cc, ident)
    }

    const cs = "configure"

    var f *file

    if v := l.project.opt.configure; v != nil {
        if x, y := v.(*boolean); y {
            if !x.bool { return }
            cc.configure = cs // use the default 'configure' module
        } else {
            cc.configure = v.string(ctx)
            if cc.configure == "" {
                erro(ctx, "empty configure spec: %v", ts(v)).trace()
            } else if cc.configure == "." {
                cc.configure, cc.local = cs, true
            }
        }
    } else if f = stat(ctx, cs, l.project); f != nil {
        cc.configure, cc.local = cs, true
    }

    if f == nil && cc.configure != "" {
        if filepath.IsAbs(cc.configure) {
            f = stat(ctx, cc.configure)
        } else {
            f = stat(ctx, cc.configure, l.project)
        }
    }

    if f != nil && f.exists() {
        cc.abs, cc.isDir = f.fullname(), f.info.IsDir()
    }

    if cc.abs == "" && l.project.opt.configure != nil {
        if !cc.local {
            cc.abs, cc.isDir = l.search(ctx, cc.configure)
        }
        if cc.abs == "" {
            erro(ctx, "%v: no such project: %s", l.project, cc.configure).trace()
        }
    }

    if cc.abs == "" {
        if l.project.opt.configure != nil {
            erro(ctx, "%v: missing the default .configure", l.project).trace()
        }
        return
    }

    if cc.Context = pc(cc.Context, ident); cc.isDir {
        l.directory(&cc, cc.configure, cc.abs, nil)
    } else {
        l.file(&cc, filespec(l.workdir, cc.abs), cc.abs, nil)
    }

    if cc.p == nil {
        erro(ctx, "%s not loaded", cc.configure).trace()
    }

    if x, y := l.project.Lookup(dot_configure).(*project); !y || x == nil {
        if _, alt := l.project.projectname(ctx, dot_configure, cc.p); alt != nil {
            if p, y := alt.(*project); !y || p == nil {
                erro(ctx, "name `%s' already taken (%s).", cc.p.name, typeof(alt)).trace()
            }
        }
    }

    if l.project.configure == cc.p { return }
    if l.project.configure != nil {
        erro(ctx, "%s already specified", dot_configure).trace()
    }

    // Set .configure first to ensure the configuration.sm is correct
    l.project.configure = cc.p

    for _, proj := range cc.p.usees(true, false, false, false) {
        if e := l.use_proj(ctx, useopts{}, proj); e != nil { // see usevars
            erro(ctx, "failed to use %v : %v", proj, e).trace()
        }
    }

    // Load configuration.sm after .configure was loaded.

    var c = l.project.configuration
    if c != nil {
        erro(ctx, "%v: already loaded %v", l.project, c).trace()
    }
    if c = l.project.configuration_sm(ctx); c == nil {
        erro(ctx, "%v: nil configuration file", ident).trace()
    }
    if !c.exists() || c.stat(ctx) == nil {
        return // not configured yet
    }

    cc.configuration = c
    l.source(&cc, c.fullname(), nil)
    l.project.configuration = c // loaded configuration.sm
}

func (l ul) container(ctx Context, ident Value, identStr string) {
    if l.project.name != dot_container {
        if _, e := os.Stat(".dock"); e == nil {
            erro(ctx, "Must rename .dock into .container !").trace()
        }

        // Looking for project specific .container module
        if f := stat(ctx, dot_container, l.project); f.exists() {
            l.dot_container(ctx, ident, identStr, f)
            if l.verbose {
                info(ctx, "%v for %s (%s)\n", f, l.project.spec, l.project).debug()
            }
            return
        }

        // Looking for .smart/.container
        walkSmartBaseDirs(ctx, l.project.absPath, func(s string) bool {
            var f = stat(ctx, dot_container, stat_dir{filepath.Join(s, ".smart")})
            if f.exists() {
                l.dot_container(ctx, ident, identStr, f)
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

func (l ul) source(ctx Context, filename string, src any) (res Value) {
    if l.traceLaunch { defer un(l_trace(l_launch, "loader.source")) }

    var t = time.Now()
    var text []byte

    if checkpoints && truly(ctx, is_test_mode{}) {
        l.pre_source_check(ctx, filename, src)
        defer l.source_check(ctx, filename, src, &text, &res)
    }

    defer func(p *parser) {
        if l.p == nil {
            erro(ctx, "nil parser ; %v", p).trace()
        } else if d := time.Now().Sub(t); l.slow < d {
            if p != nil { ctx = pc(ctx,p.Position()) }

            var t = l.p.Position()
            if s := filename; s != t.Filename { warn(pc(ctx,s), "%s", d).debug() }
            warnstack(pc(ctx,t), 8, "%v %v", d, l.project).debug(128)
        }
        l.p = p
    } (l.p)

    var opts, _ = do(ctx, get_include_opts{}).(*include_opts)
    var path_err bool

    text, path_err = load_source_bytes(ctx, filename, src)
    if path_err && (opts != nil && !opts.ifExists) {
        var d = time.Now().Sub(t)
        errostack(pc(ctx,filename), 3, "%v : no such source file", d).trace()
    }

    if text == nil { return }

	var smod scanmode
	if l.mode&ParseComments != 0 {
		// smod = scanner.ScanComments
	}

    f := l.fset.AddFile(filename, -1, len(text))
    l.p = &parser{}
    l.p.scanner.init(ctx, f, text, smod)
	l.p.next(ctx, true) // starts scanning

    if truly(ctx, parse_is_text{}) {
        return ease(ctx, l.values(ctx))
    }

    l.parse(ctx, filename)

    if truly(ctx, is_flat_mode{}) {
        return nil
    } else {
        return l.project
    }
}

// ParseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (l ul) config_dir(ctx Context, pathname, linked string) (err error) {
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
    defer l.closescope(l.openscope(bases(2, sof, true)))

    var scope = l.scope()

    for _, f := range fs {
        var name = f.Name()
        if has_prefix(name, "~") || has_suffix(name, ".#", ".smart", ".sm") {
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

        d.set(ctx, defConfDir, _strlit(_position(ctx), s))
    }
    return
}

func nonsource(name string, mo os.FileMode) (_ bool) {
    if  !mo.IsRegular() || name == "" || name == configuration_sm || strings.HasPrefix(name, ".#") ||
        !(strings.HasSuffix(name, ".smart") || strings.HasSuffix(name, ".sm")) { return true }
    return
}

func (l ul) sources(ctx Context, path string, filter func(os.FileInfo) bool) (sources []string) {
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

        sources = append(sources, filename)
    }
    return
}

// ul.load loads script from a file or source code
func (l ul) file(ctx Context, spec, absPath string, source any) {
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
        erro(ctx, "%v: no such base: %v", l.project, spec).trace()
    } else if !filepath.IsAbs(absPath) {
        erro(ctx, "%v: not absolute path: %v", l.project, spec).trace()
    }

    // Check loaded project.
    if p, y := l.globe.loaded[absPath]; y {
        if _, a := l.scope().projectname(ctx, p.name, p); a != nil {
            if x, y := a.(*project); !y || x == nil {
                erro(ctx, "name already taken: %v (%s).", p, typeof(a)).trace()
            }
        }
        do(ctx, declared_project{p})
        return
    }

    var lo = l
    if l.project != nil {
        lo.loader = &loader{term:term{ctx,l.scope()}}
        ctx = lo.loader
    }

    defer lo.configure_save(ctx)
    lo.source(ctx, absPath, source)
    return
}

func (l ul) directory(ctx Context, spec, absDir string, filter func(os.FileInfo) bool) {
    if absDir == "" {
        erro(ctx, "%v: no such base: %v", l.project, spec).trace()
    } else if !filepath.IsAbs(absDir) {
        erro(ctx, "%v: not absolute path: %v", l.project, spec).trace()
    }

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
            if _, alt := proj.projectname(ctx, loaded.name, loaded); alt != nil {
                if val, y := alt.(*project); !y || val == nil {
                    erro(ctx, "name `%s' already taken (%T).", loaded.name, alt).trace()
                }
            }
        }
    } (time.Now(), l.verboseLoads)

	if checkpoints && truly(ctx, is_test_mode{}) {
        defer l.directory_check(ctx, spec, absDir)
    }

    // Check previously loaded project.
    if loaded, okay = l.globe.loaded[absDir]; okay && loaded != nil {
        do(ctx, declared_project{loaded})
        return
    }

    var lo = l
    if l.project != nil {
        lo.loader = &loader{term:term{ctx,nil}}
        ctx = lo.loader
    }

    var cc = _abs_ctx(ctx, absDir)
	var sof,_ = filepath.Rel(workBaseDir, absDir)
    defer lo.closescope(lo.openscope(bases(2, sof, true)))
    defer lo.configure_save(ctx)

    // Use globe outer scope to avoid conflicting with other unrelated projects.
    lo.scope().outer = lo.globe.scope
    for _, s := range lo.sources(ctx, absDir, filter) { lo.source(cc, s, nil) }

    if len(lo.declares) == 0 && filepath.Base(spec) != "@" {
        if truly(ctx, is_implicit_load{}) {
            warn(ctx, "%s not loaded (as %s, implicitly)", spec, absDir).debug()
            return // okay for implicit loading
        }

        for s, m := range l.globe.loaded { erro(ctx, "%v: %v", s, m) }
        errostack(ctx, 3, "%s not loaded (as %s)", spec, absDir).trace()
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

type parse_is_text struct{}
type loadtext_ctx struct{ Context }
func (p loadtext_ctx) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case parse_is_text: return true
	}
	return p.Context.do(ctx, op)
}

func (l ul) text(ctx Context, filename string, text string) Value {
    if l.globe.main == nil {
        l.loader.scope = l.globe.os.scope
    } else {
        l.loader.scope = l.globe.main.scope
    }
    return l.source(loadtext_ctx{ctx}, filename, text)
}
