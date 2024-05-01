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

var baseWorkDir, _ = os.Getwd()
var launchTime = time.Now()
var searchPaths searchlist

type char rune
func (c char) String() string { return string(rune(c)) }

type searchlist []string
func (sl *searchlist) String() string { return fmt.Sprint(*sl) }
func (sl *searchlist) Set(value string) error {
    *sl = append(*sl, strings.Split(value, ",")...)
    return nil
}

type unmapping struct { Context }
func (c unmapping) cast(t reflect.Type) Context { return implcast(c, t) }
func (c unmapping) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propCacheUnmap)
}

type pathcache struct { Context ; p *path ; i int }
func (c *pathcache) cast(t reflect.Type) Context { return implcast(c, t) }
func (c *pathcache) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propCachePath)
}

type pstrcache struct { Context ; s string ; ss []string ; i int }
func (c *pstrcache) cast(t reflect.Type) Context { return implcast(c, t) }
func (c *pstrcache) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propCachePath)
}

type filecache_context struct { Context ; p *filecache ; k interface{} }
func (c filecache_context) cast(t reflect.Type) Context { return implcast(c, t) }

func cacheMapping(ctx Context) bool { return !cacheUnmap(ctx) }
func cacheUnmap(ctx Context) bool { t, _ := ctx.do(propCacheUnmap).(bool); return t }

var (
    closure_t = reflect.TypeOf(closure{})
    globpat_t = reflect.TypeOf(globpat{})
    percpat_t = reflect.TypeOf(percpat{})
    regepat_t = reflect.TypeOf(regexpat{})
)

type filecache struct {
    a []filemap

    // NOTE: some performance overhead when using interface{}-keys, comparing to
    // string-keys, this is easier to implement and making code clean.
    m map[interface{}]*filecache
}

func (p *filecache) String() (s string) { // NOTE: for debug
    s += "{"
    if m := p.a; m != nil {
        s += "["
        for k, v := range m { s += fmt.Sprintf("%v:%v,", k, v) }
        s = strings.TrimSuffix(s, ",") + "],"
    }
    if m := p.m; m != nil {
        if strings.HasSuffix(s, "]") { s += "," }
        for k, v := range m { s += fmt.Sprintf("%v:%v,", k, v) }
        s = strings.TrimSuffix(s, ",")
    }
    return strings.TrimSuffix(s, ",") + "}"
}
func (p *filecache) hit(ctx Context, k interface{}) (res *filecache, done bool) {
    switch t := k.(type) {
    case interface{ filecache(Context, *filecache) (*filecache, bool) }:
        if res, done = t.filecache(ctx, p) ; res == nil && cacheMapping(ctx) {
            errostack(at(ctx,k), 3, "no filecache for %v : %v", us(k), p).debug(8)
        }
        return

    case Value:
        if indeterminate(ctx, t) {
            return p.indeter(ctx, t)
        }
        errostack(at(ctx,k), 5, "non-filecache-able %v : %v", us(k), p).debug(8)
        return
    }

    var y bool
    if cacheUnmap(ctx) {
        if p.m == nil { return }
        if res, y = p.m[k]; y { return }
        return p.unpats(ctx, k)
    } else {
        if p.m == nil { p.m = make(map[interface{}]*filecache) }
        if res, y = p.m[k]; !y || res == nil {
            res = &filecache{}
            p.m[k] = res
        }
        return
    }
}
func (p *filecache) indeter(ctx Context, v Value) (res *filecache, done bool) {
    if cacheUnmap(ctx) {
        note(at(ctx,v), "TODO: %v %v", us(v), reflect.TypeOf(v)).debug(1)
    } else {
        note(at(ctx,v), "TODO: %v %v", us(v), reflect.TypeOf(v)).debug(1)
    }
    return
}
func (p *filecache) unpats(ctx Context, k interface{}) (res *filecache, done bool) {
    if checkpoints { defer trace(ctx) }

    if x, y := p.m[globpat_t]; y && x.m != nil {
        if res, done = x.glob(ctx, k);  res != nil { return }
    }
    if x, y := p.m[percpat_t]; y && x.m != nil {
        if res, done = x.perc(ctx, k);  res != nil { return }
    }
    if x, y := p.m[regepat_t]; y && x.m != nil {
        if res, done = x.regex(ctx, k); res != nil { return }
    }

    if checkpoints {
        if false { note(at(ctx,k), "unmap: %v : %v", us(k), p).debug(5) }
    }

    if fc, y := ctx.(filecache_context); y && fc.p != nil {
        if checkpoints {
            if false && fc.k == k {
                erro(ctx, "%v : %v", us(k), p).debug(5)
                return
            }
            if fc.p == p {
                // erro(ctx, "%v : %v %v", us(k), us(fc.k), fc.p).debug(5)
                return
            }
            if false {
                note(ctx, "%v %v : %v %v", us(k), p, us(fc.k), fc.p).debug(5)
            }
        }
        if p != fc.p {
            return fc.p.unpats(fc.Context, fc.k)
        }
    }
    return
}
func (p *filecache) glob(ctx Context, k interface{}) (res *filecache, done bool) {
    defer trace(ctx)

    if pc := cast[*pathcache](ctx); pc != nil {
        return p.glob_path(ctx, k, pc)
    }

    if pc := cast[*pstrcache](ctx); pc != nil {
        return p.glob_pstr(ctx, k, pc)
    }

    if s, y := k.(string); y {
        return p.glob_str(ctx, s)
    } else {
        erro(ctx, "TODO: %v ; %v", us(k), p).debug(3)
        return
    }
}
func (p *filecache) glob_path(ctx Context, k interface{}, pc *pathcache) (res *filecache, done bool) {
    defer trace(ctx)
    for t, c := range p.m {
        var pat, y = t.(string)
        if !y {
            erro(ctx, "TODO: %v %v ; %v", us(k), us(t), c).debug(3)
            continue
        }

        if checkpoints {
            if strings.Contains(pat, "/") {
                erro(ctx, "%v %v %v", k, t, c).debug(1)
                return
            }
        }

        var v Value
        if pc.i == pc.p.len()-1 {
            v = pc.p.elems[pc.i]
        } else {
            v = &path{elements{pc.p.elems[pc.i:]}}
        }
        if c._glob(ctx, pat, v.string(ctx)) {
            pc.i = pc.p.len()
            return c, true
        }
    }
    return
}
func (p *filecache) glob_pstr(ctx Context, k interface{}, pc *pstrcache) (res *filecache, done bool) {
    defer trace(ctx)
    for t, c := range p.m {
        var pat, y = t.(string)
        if !y {
            erro(ctx, "TODO: %v %v ; %v", us(k), us(t), c).debug(1)
            continue
        } else if strings.Contains(pat, "**") {
            if c._glob(ctx, pat, pc.s) {
                pc.i = len(pc.ss)
                return c, true
            }
            continue
        }

        if checkpoints {
            if strings.Contains(pat, "/") {
                erro(ctx, "%v %v %v", k, t, c).debug(1)
                return
            }
        }

        var ssn = len(pc.ss)
        var str = pc.ss[pc.i]

        y = c._glob(ctx, pat, str)

        if (pat == "*zzz" || pat == "**z") && (pc.s == "foo/xxxzzz" || pc.s == "foo/xx/yy/zzzz") {
            note(ctx, "%v %v , %v %v %v , %v", pat, str, y, pc.i, ssn, c).debug(1)
        }

        if y && pc.i+1 == ssn {
            for _, a := range c.a {
                if t, y := a.pattern.(*path); y && t.len() == ssn {
                    if done, _, _ = t.match(ctx, pc.ss) ; done { res = c }
                    return
                } else if false {
                    note(at(ctx,a.pattern), "%v : %v ; %v %v ; %v , %v", pat, str, k, pc.s, c, p).debug(1)
                }
            }
        } else if y {
            if checkpoints {
                if false && pat == "??" && fmt.Sprintf("%s", k) == "ab" {
                    note(ctx, "%v %v %v", pat, str, c).debug(1)
                }
            }
            return c, done
        }
    }
    return
}
func (p *filecache) glob_str(ctx Context, s string) (res *filecache, done bool) {
    defer trace(ctx)
    for t, c := range p.m {
        if pat, y := t.(string) ; !y {
            erro(ctx, "TODO: %v : %v %v", s, us(t), c).debug(3)
        } else if c._glob(ctx, pat, s) {
            return c, /* true */done
        }
    }
    return
}
func (p *filecache) _glob(ctx Context, pat, str string) bool {
    if checkpoints { defer trace(ctx) }

    var f, s, e = globMatch(ctx, pat, str)

    if checkpoints {
        if e != nil {
            erro(ctx, "%v %v → %v %v %v", str, pat, f, s, e).debug(1)
        }
    }
    return f
}

func (p *filecache) perc(ctx Context, k interface{}) (res *filecache, done bool) {
    erro(ctx, "TODO: %v %v", us(k), p).debug(3)
    return
}

func (p *filecache) regex(ctx Context, k interface{}) (res *filecache, done bool) {
    erro(ctx, "TODO: %v %v", us(k), p).debug(3)
    return
}

type hooks struct {
    assert func(Context, Value, bool) bool
    debug func(Context, string, []Value)
}

type benchmark struct {
    benchmark_builtin_expand func(*builtin, Context, time.Time, reflect.Value)
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

    benchmark

    hooks hooks

    workdir string
    prefix  string // FIXME: prefix for distribution

    scope   *Scope
    globe   *globe

    fset    *FileSet
    paths   searchlist

    packages map[string]packageinfo

    statmutex sync.Mutex
    statcache map[string]*filebase // File.fullname() -> File
    filemaps filecache // value -> dirs

    expand_n int32

    ddd string // debug parsing via `eval -ddd=example`, also project.dd
}
func (ctx *universe) dirtyMark(vals ...Value) { return }
func (ctx *universe) ref(_ Context, _ Value) bool { return false }
func (ctx *universe) loader() *loader { return ctx.globe.top }
func (ctx *universe) Globe() *globe { return ctx.globe }
func (ctx *universe) Scope() *Scope { return ctx.scope }
func (ctx *universe) String() (s string) { return /*"universe"*/ }
func (ctx *universe) do(prop property, a ...interface{}) (res interface{}) {
    switch prop {
    case actOnErros:
        if ctx.panicFailureOnErrosFlushed {
            var errs int
            for _, i := range a { errs += i.(int) }
            if 0 < errs { panic(_failure(ctx, "got %d errors", errs)) }
            res = true
        }
        return

    case propPosition: return ctx.Position()
    case propWorkDir: if ctx.workdir == "" { return baseWorkDir }
        return ctx.workdir
    }
    return ctx.diagnostic.do(prop, a...)
}
func (ctx *universe) project() (p *project) {
    if ctx != nil && ctx.globe != nil { p = ctx.globe.main }
    return
}
func (ctx *universe) Position() (p Position) {
    if ctx.globe == nil || ctx.globe.main == nil {
        p.Filename = _workdir(ctx)
        p.Line, p.Column = 0, 0
    } else {
        p = ctx.globe.main.position
    }
    return
}
func (ctx *universe) cast(t reflect.Type) (c Context) {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}
func (ctx *universe) projects(_ Context, projs ...*project) []*project {
    if len(projs) > 0 { panic(_failure(ctx, "%v", projs)) }
    return nil
}
func (ctx *universe) closure() (scopes []*Scope) {
    if m := ctx.globe.main; m != nil && m.scope != nil {
        if false { scopes = append(scopes, m.scope) }
    }
    return
}

func (ctx *universe) doHelp()       { do_helpscreen(ctx) }
func (ctx *universe) doHelpFlags()  { print_flag_trace(ctx) }
func (ctx *universe) doHelpConfig() { print_configuration(ctx) }

func _commandline() commandline { return commandline{
    debugPrompt: true,
    debugErrors: true,
    debugWarns:  true,
    debugInfos:  true,

    fastMode: true,
    parallel: false, // FIXME: program.traverse not working in parallel

    panicFailureOnErrosFlushed: true,
    silentOptionalSelection: false,

    slow: 9990 * time.Millisecond,
}}

func new_universe(ii ...interface{}) (ctx *universe) {
    var p positional
    ctx = &universe{}
    ctx.Context = &p

    if s, e := os.Getwd(); e != nil {
        erro(ctx, "%v", e).debug(6)
        return
    } else {
        p.position.Filename = s
        ctx.workdir = s
        ctx.paths = searchPaths
        ctx.fset = NewFileSet()
        ctx.statcache = make(map[string]*filebase)
        ctx.scope = newScope(ctx.Position(), nil, nil, `universe`)
    }

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
    if true { ctx.benchmark_builtin_expand = (*builtin).benchmark_expand }

    var bin  = ease(ctx, os.Args[0])
    var args = ease(ctx, os.Args[1:])
    _, _ = ctx.scope.define(ctx, defVoid, "SMART.ARGS", args)
    _, _ = ctx.scope.define(ctx, defVoid, "SMART.BIN", bin)
    _, _ = ctx.scope.define(ctx, defVoid, "SMART", bin)

    for name, f := range builtins {
        if _, alt := ctx.scope.builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
    }

    var os Value // one of darwin, freebsd, linux, and so on.
    var pos = p.position
    {
        var vs []Value
        for _, s := range strings.Fields(runtime.GOOS) {
            vs = append(vs, makeBareword(pos, s))
        }
        os = ease(ctx, vs)
    }

    ctx.globe = &globe{
        Scope: newScope(pos, ctx.scope, nil, `globe "smart"`),
        flagEntries: make(map[string][]entry),
        loaded: make(map[string]*project),
        args: make(map[Value][]Value),
    }

    // FIXME: ctx.scope.scopename(ctx, ".GLOBE", ctx.globe.Scope)
    ctx.globe.os,    _ = ctx.globe.define(ctx, defVoid, ".os", os)
    ctx.globe.goals, _ = ctx.globe.define(ctx, defVoid, ".goals", makeNone(pos))
    ctx.globe.mode,  _ = ctx.globe.define(ctx, defVoid, ".mode",  makeNull(pos))
    return
}

func (u *universe) file(filename string, src []byte) *TokFile {
    return u.fset.AddFile(filename, -1, len(src))
}

type filestub struct {
    dir  string      // full directory where the file was or should be found
    sub  string      // matched sub path (see project.search), may be Dir (absolete path)
    name string      // constant represented name (e.g. relative filename)
    filemap *filemap // matched pattern (see 'files' directive)
    other *filestub  // pointed to another stub (in a different project) of the same file
}
func (p *filestub) subname() (s string) {
    if isAbsOrRel(p.sub) {
        s = p.name
    } else {
        s = filepath.Join(p.sub, p.name)
    }
    return
}

type filebase struct {
    stub filestub    // cycled-list of file stubs of different projects
    info os.FileInfo // file info if exists
    _updated bool // true if this file has been updated by a program
    _updatedDeps []Value // any updated deps
    _travin int
    _traved int
    _dirty int
}
func (p *filebase) exists() bool { return p.info != nil }

type stat_dir struct { string }
type stat_sub struct { string }
type stat_nonexist struct { bool }
type stat_fileinfo struct{ os.FileInfo }

func stat(ctx Context, name string, ii ...interface{}) (file *File) {
    var sub, dir string
    var nonexist bool
    var fileInfo os.FileInfo
    for _, i := range ii {
        switch t := i.(type) {
        case *project: dir = t.absPath
        case stat_dir: dir = t.string
        case stat_sub: sub = t.string
        case stat_fileinfo: fileInfo = t.FileInfo
        case stat_nonexist: nonexist = t.bool
        default:
            erro(ctx, "invalid stat arg: %v", us(i)).debug(2)
            return
        }
    }

    var (
        base *filebase
        stub *filestub
        fullname string
        u = _universe(ctx)
    )

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
                erro(ctx, "dir name conflicts: %s <-> %s (sub=%v)", dir, name, sub).debug(16)
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
        if fileInfo == nil { fileInfo, _ = os.Stat(fullname)
            if fileInfo == nil && !nonexist { return nil }
        }

        base = &filebase{ filestub{ dir, sub, name, nil, nil }, fileInfo, false, nil, 0, 0, 0 }
        base.stub.other = &base.stub
        stub = &base.stub
        u.statcache[cleanFullname] = base
    }

GotFile:
    file = &File{valbase{ctx.Position()},base,stub}
    return
}

func (u *universe) filemap(ctx Context, p *project, patts, paths []Value) (res []filemap) {
    defer trace(ctx)

    var base = &_filemap{p, patts, paths}

    for _, patt := range patts {
        if patt == nil {
            errostack(ctx, 5, "nil pattern ; paths=%v", paths).debug(1)
        } else if c, _ := u.filemaps.hit(ctx, patt); c != nil {
            m := filemap{base, patt}
            c.a = append(c.a, m)
            res = append(res, m)
        }
    }
    return
}

func (u *universe) unmap(ctx Context, key interface{}) (res []matchedfilemap) {
    defer trace(ctx)

    var c = &u.filemaps

    if x, y := key.(Value); y && x.patterned(ctx) {
        erro(at(ctx,x), "TODO: %v : %v", x, c).debug(1)
        return
    }

    if s, y := key.(string); y && strings.ContainsAny(s, pathSep) {
        var ss = strings.Split(s, pathSep)
        var x = pstrcache{unmapping{ctx}, s, ss, 0}
        var cc Context = &x
    out:
        for c != nil {
            var _c = c
            for j, t := range strings.Split(ss[x.i], DOT.String()) {
                if c != nil &&  0 < j  { c, y = c.hit(cc, DOT)
                    if y/* done */ { if checkpoints{break out}; goto fullhit }}
                if c != nil && t != "" { c, y = c.hit(cc, t)
                    if false && y { note(ctx, "TODO: %v → %-4s → %v", s, t, c) }
                    if y/* done */ { if checkpoints{break out}; goto fullhit }}
            }
            if c == nil || len(ss) <= x.i+1 { break }
            cc = filecache_context{cc, _c, ss[x.i]}
            x.i += 1
        }
        if checkpoints {
            if false && y && len(ss) <= x.i { // full-matched
                note(ctx, "TODO: %v → %v", ss, c).debug(3)
            } else if true && c == nil && strings.HasPrefix(s, ".test/") {
                // for i := len(cs)-1; 0 <= i; i -= 1 { note(ctx, "%v. %v", i, cs[i]) }
                note(ctx, "TODO: %v → %v", ss, c).debug(3)
            }
        }
    } else {
        c, y = c.hit(unmapping{ctx}, key)
    }

    if c == nil { return }

fullhit:
    for _, m := range c.a {
        var matched, pattern, s = m.match(ctx, key)
        if  matched && pattern != nil  {
            if checkpoints {
                if !equal(ctx, m.pattern, pattern) {
                    note(ctx, "%v != %v", us(m.pattern), pattern).debug(3)
                }
            }
            res = append(res, matchedfilemap{m, s})
        } else {
            erro(ctx, "%v %v ; %v %v", tv(key), m.pattern, pattern, c).debug(1)
        }
    }
    return
}

type matchedfilemap struct {
    filemap // ; pattern Value
    name string
}

func (m matchedfilemap) string() string {
    return fmt.Sprintf("{%v, %v, %v}", m.name, m.pattern, m.project)
}

func unmapfiles(ctx Context, key interface{}) (maps []matchedfilemap) {
    return _universe(ctx).unmap(ctx, key)
}

func (dc *universe) AddSearchPaths(paths... string) (err error) {
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

func (dc *universe) search(ctx Context, linfo *loadinfo, specName string) (absPath string, isDir bool) {
    if specName == "." {
        erro(ctx, "not possible to chain itself").debug(1)
    } else if abs := filepath.IsAbs(specName); abs || specName == "~" || specName == ".." ||
        hasPrefix(specName, "~"+pathSep, "."+pathSep, ".."+pathSep) {
        var (
            s = specName
            sx string
        )
        if !abs && linfo.absDir != "" {
            sx = filepath.Join(linfo.absDir, s)
            if a, e := filepath.Abs(sx); e == nil {
                s = a
            } else {
                erro(ctx, "abs: %v", e)
                return
            }
        }
        if fi, err := os.Stat(s); err == nil { return s, fi.IsDir() }

        sx = s + ".smart"
        if fi, err := os.Stat(sx); err == nil { return sx, fi.IsDir() }

        sx = s + ".sm"
        if fi, err := os.Stat(sx); err == nil { return sx, fi.IsDir() }
    } else {
        for _, base := range dc.paths {
            var s string
            if filepath.IsAbs(base) {
                s = filepath.Join(base, specName)
            } else {
                s = filepath.Join(_workdir(dc), base, specName)
            }
            if fi, err := os.Stat(s); err == nil && fi != nil {
                return s, fi.IsDir()
            }
        }
    }
    return
}

func startCPUProfile(ctx Context, name string, heap ...bool) (stop func()) {
    var fn string
    if filepath.IsAbs(name) { fn = name } else
    if m := ctx.Globe().main; m == nil {} else
    if f := m.tempFile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        erro(ctx, "%T: %v", e, e).debug(1)
    } else if e = pprof.StartCPUProfile(f); e != nil {
        erro(ctx, "%T: %v", e, e).debug(1)
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        if heap != nil && heap[0] { runtime.GC() // update memory statistics
            if e = pprof.WriteHeapProfile(f); e != nil {
                erro(ctx, "WriteHeapProfile: %v", e).debug(1)
            }
        }
        f.Close()
    }}
}

func startHeapProfile(ctx Context, name string) (stop func()) {
    var fn string
    if filepath.IsAbs(name) { fn = name } else
    if m := ctx.Globe().main; m == nil {} else
    if f := m.tempFile(ctx, name); f == nil {
        fn = filepath.Join(_workdir(ctx), name)
    } else {
        fn = f.fullname()
    }

    f, e := os.Create(fn)
    if e != nil {
        erro(ctx, "%T: %v", e, e).debug(1)
    }
    return func() { if f != nil {
        if e != nil { pprof.StopCPUProfile() }
        runtime.GC() // update memory statistics
        if e = pprof.WriteHeapProfile(f); e != nil {
            erro(ctx, "WriteHeapProfile: %v", e).debug(1)
        }
        f.Close()
    }}
}

func (dc *universe) run() (result []Value, travestates []*travestate) {
    if dc.noRun { return }

    var main = dc.globe.main
    if main == nil {
        erro(dc, "no targets to update `%v`", dc.globe.goals).debug(1)
        return
    }

    var ctx Context = closureWith(dc, main.scope)
    if dc.verbose { info(ctx, "goal: %v", main).debug(1) }

    removeTempDirs(ctx)

    if dc.cpuProf != "" || dc.autoProfs {
        var name = dc.cpuProf
        if name == "" { name = "run.cpu.auto.prof" }
        defer startCPUProfile(ctx, name, true)()
    } else if dc.memProf != "" || dc.autoProfs {
        var name = dc.memProf
        if name == "" { name = "run.mem.auto.prof" }
        defer startHeapProfile(ctx, name)()
    }

    var done bool
    for _, flag := range dc.globe.flags {
        if dc.verboseExecFlags { info(at(ctx, flag), "%v", flag) }

        var s = flag.Value.string(ctx)
        var args, _ = dc.globe.args[flag]
        var entries, _ = dc.globe.flagEntries[s]
        for _, entry := range entries {
            var ctx = at(ctx, entry.Position())
            if dc.verboseExecFlags {
                info(ctx, "%v", entry)
                flush(ctx)
            }

            var ( res []Value; traves []*travestate )
            if res, traves = entry.execute(ctx, args...); len(traves) > 0 {
                for _, brk := range traves {
                    if brk.what == traveFail {
                        erro(at(ctx,brk.pos), "execute '%v': %v", entry, brk).debug(1)
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
            case *bareword:
                if entries := proj.resolveEntries(ctx, t.s, true); entries == nil {
                    erro(ctx, "no such entry `%s`", t.s).debug(1)
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case *delegate:
                var s = t.string(ctx)
                if entries := proj.resolveEntries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
                    return false
                } else {
                    for _, entry := range entries {
                        goals = append(goals, entry)
                    }
                }
            case flag:
                var s = t.string(ctx)
                if entries := proj.resolveEntries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
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
                errostack(ctx, 3, "%v: unknown target: %v (%s)", proj, goal, typeof(goal)).debug(16)
                return false
            }
        }
        return true
    }

    if collect(main, merge(dc.globe.goals.value)) {
        if len(goals) == 0 {
            if entry := main.defaultEntry; entry != nil {
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
func (uni *universe) load() (err error) {
    if uni.traceLaunch { defer un(l_trace(l_launch, "universe.load")) }

    var (
        args []Value
        base = _workdir(uni)
        ctx Context = uni
    )
    if s := filepath.Join(base, ".smart", "modules"); s != "" {
        if _, e := os.Stat(s); e == nil { uni.AddSearchPaths(s) }
    }
    if s := filepath.Join(base, entryFileName); s != "" {
        if _, e := os.Stat(s); e != nil { s = filepath.Join(base, "build.smart")
            if _, e := os.Stat(s); e != nil { s = "" }
        }
        if s != "" {
            var pos Position
            pos.Filename = s
            pos.Line = 1
            ctx = at(ctx, pos)
        }
    }

    defer func(l *loader) { uni.globe.top = l } (uni.globe.top)

    uni.globe.top = &loader{ terminal: terminal{
        ctx, []*Scope{uni.globe.Scope},
    }}

    if s := strings.Join(os.Args[1:], " "); s != "" {
        if v := uni.globe.top.text(ctx, base, s); 0 < len(v) {
            args = parseOpts(ctx, &uni.commandline, v...)
        }
    }

    if v := uni.reconfigure; v { uni.commandline.configure = v }
    if v := uni.fastMode; v { // Turn off many things for fast mode:
        //uni.noImportFiles = v
        uni.noDepsGrep = v
        uni.noDeps = v
        uni.noGrep = v
    }

    if uni.verbose { defer func(t time.Time) {
        prompt(ctx, "Goals %v (%s)\n", uni.globe.goals, time.Now().Sub(t))
    } (time.Now()) }

    assert(uni.globe.args != nil, "globe args is nil")

    if uni.autoProfs {
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
        defer func() {
            var prof string //= uni.memProf
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

    var mode = new(bareword)
    for _, target := range args {
        switch t := target.(type) {
        case *pair: uni.globe.pairs = append(uni.globe.pairs, t)
        case flag: uni.globe.flags = append(uni.globe.flags, t)
            if s := t.Value.string(ctx); s == "clean" {
                mode.position, mode.s = t.Position(), "clean"
            }
        case *argumented:
            uni.globe.args[t.Value] = t.args
            if f, ok := t.Value.(flag); ok {
                uni.globe.flags = append(uni.globe.flags, f)
            } else {
                uni.globe.goals.append(ctx, t/*.value*/)
            }
        default:
            uni.globe.goals.append(ctx, t)
        }
    }
    if mode.s == "" { if uni.commandline.configure {
        mode.s = "configure"
    } else {
        mode.s = "goals"
    }}
    uni.globe.mode.value = mode

    defer func(t time.Time) { if d := time.Now().Sub(t); uni.verboseImport {
        var name string
        if p := uni.globe.top.project(); p != nil { name = p.name }
        prompt(ctx, "└·%s … (%s)\n", name, d)
    } else if d > uni.slow {
        if m := uni.globe.main; m != nil {
            warn(at(ctx, m.position), "slow loading (%v)!!\n", d).debug(6)
        } else {
            prompt(ctx, "%s:1:warning: slow loading (%v)!!\n", base, d).debug(6)
        }
    }} (time.Now())

    if uni.verboseImport { prompt(ctx, "┌→%s\n", base) }
    if!uni.globe.top.path(ctx, base, nil) { return }
    if uni.globe.main == nil { erro(ctx, "nothing loaded\n").debug(1) }
    return
}

// A globe represents a global execution context.
type globe struct {
    *Scope

    loads []*loadinfo
    top     *loader

    main   *project
    loaded map[string]*project // loaded projects

    stack  []map[string]*def

    args map[Value][]Value
    flagEntries map[string][]entry
    flags []flag
    pairs []*pair

    os    *def
    goals *def
    mode  *def
}

// Main returns the main project.
func (g *globe) Main() *project { return g.main }

func (g *globe) SetScopeOuter(scope *Scope) { scope.outer = g.Scope }

func (g *globe) AddFlagEntry(name string, entry entry) {
    flags, _ := g.flagEntries[name]
    flags     = append(flags, entry)
    g.flagEntries[name] = flags
    return
}

// project returns a new project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *globe) project(ctx Context, outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *project) {
    if outer == nil { outer = g.Scope }

    m = &project{
        position: ctx.Position(),
        absPath: absPath,
        relPath: relPath,
        tmpPath: tmpPath,
        use:  new(uselist), // TODO: use scopename instead?
        spec: spec,
        name: name,
    }
    m.scope = newScope(m.position, outer, m, fmt.Sprintf("project %q", name))
    m.scope.mutex.Lock()
    // outer.mutex.Lock()
    // if s, y := outer.elems[".self"]; y { m.scope.elems[".base"] = s } else {
    //     if true { warn(ctx, "%v: no base", name).debug(12) }
    // }
    // outer.mutex.Unlock()
    m.scope.elems[".self"] = self{m}
    m.scope.elems[".usee"] = m.use
    m.scope.mutex.Unlock()
    m.use.name = "usee"
    m.use.owner_ = m
    m.use.scope = m.scope

    if g.main == nil && spec != "" && name != "@" && name != "~" {
        for outer != nil && outer != g.Scope {
            if p := outer.project; p != nil && p.name == "@" {
                return
            }
            outer = outer.outer
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
           searchPaths = append(searchPaths, s)
        }
    }
    return
}
