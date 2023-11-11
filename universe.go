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

type searchlist []string
func (sl *searchlist) string() string { return fmt.Sprint(*sl) }
func (sl *searchlist) Set(value string) error {
    *sl = append(*sl, strings.Split(value, ",")...)
    return nil
}

type char rune
func (c char) string() string { return string(rune(c)) }

type filemapCache struct {
    maps []FileMap
    chars map[char  ]*filemapCache // chars; char(0) goes for patterns without bare prefixs
    vals  map[Value ]*filemapCache
    strs  map[string]*filemapCache
    pats  map[string][]*filemapCachePat
}

func (p *filemapCache) String() (s string) { // NOTE: for debug only
    s += "{"
    if m := p.maps; m != nil {
        s += "maps=["
        for k, v := range m { s += fmt.Sprintf("%v:%v,", k, v) }
        s = strings.TrimSuffix(s, ",") + "],"
    }
    if m := p.chars; m != nil {
        s += "chars=["
        for k, v := range m { s += fmt.Sprintf("%v:%v,", k, v) }
        s = strings.TrimSuffix(s, ",") + "],"
    }
    if m := p.vals; m != nil {
        s += "vals=["
        for k, v := range m { s += fmt.Sprintf("%v:%v,", k, v) }
        s = strings.TrimSuffix(s, ",") + "],"
    }
    if m := p.strs; m != nil {
        s += "strs=["
        for k, v := range m { s += fmt.Sprintf("%v:%v,", k, v) }
        s = strings.TrimSuffix(s, ",") + "],"
    }
    if m := p.pats; m != nil {
        s += "pats=["
        for k, v := range m { s += fmt.Sprintf("%v:%v,", k, v) }
        s = strings.TrimSuffix(s, ",") + "],"
    }
    return strings.TrimSuffix(s, ",") + "}"
}

type filemapCachePat struct {
    filemapCache
    key Value
    value hitched
}

type hitched struct { v interface{} } // Value, string, []string

func (h hitched) String() (s string) { // for debug
    return fmt.Sprintf("hitched{%v(%v)}", typeof(h.v), h.v)
}

func (h hitched) match(ctx Context, i interface{}) (full bool, res interface{}, stems []string) {
    if v, y := h.v.(Value); y {
        full, res, stems = v.match(ctx, i)
    } else if s, y := h.v.(string); y {
        if o, y := i.(string); y {
            if strings.HasPrefix(o, s) { res, full = s, (len(s) == len(o)) }
        }
    } else if s, y := h.v.([]string); y {
        if o, y := i.([]string); y {
            var ( n int ; f bool ; a []string )
            for ; n < len(s) && n < len(o); n++ {
                if strings.HasPrefix(o[n], s[n]) {
                    a, f = append(a, s[n]), (len(o[n]) == len(s[n]))
                }
            }
            res, full = a, (n == len(s) && n == len(o) && f)
        }
    }
    return
}

type hitch struct {
    *filemapCache
    value hitched // aka full pattern
}

func (cache hitch) String() (s string) {
    s += "hitch{"
    s += fmt.Sprintf("%v(%v)", typeof(cache.value.v), cache.value.v)
    s += ","
    s += cache.filemapCache.String()
    s += "}"
    return
}

func (cache hitch) strx(ctx Context, s string, bits int) (res *filemapCache) {
    if res = cache.str(ctx, strings.Split(s, PathSep), 0, bits); false && res == nil {
        if c := cache.char0(bits); c != nil {
            cache.filemapCache = c
            res = cache.strx(ctx, s, bits)
        }
    }
    return
}

func (cache hitch) strs(ctx Context, ss []string, bits int) (res *filemapCache) {
    return cache.str(ctx, ss, 0, bits)
}

func (cache hitch) str(ctx Context, ss []string, i, bits int, a ...*filemapCache) (res *filemapCache) {
    var j = i+1

    defer func(v interface{}, c *filemapCache) {
        if false { if bits&cacheStore == 0 && res == nil && (
            ss[0] == ".test" ||
            false) && (
            true || // strings.HasSuffix(ss[i], ".o") ||
            false) {
            warn(ctx, "%08b: %v(%v) ⇒ %v", bits, typeof(v), v, c)
            warn(ctx, "%08b: %v[%d]: %s", bits, ss, i, ss[i])
            warn(ctx, "%08b: ⇒ %v", bits, res)
            warnstack(ctx, 3).debug(32)
        }}

        if res != nil && res.maps == nil && (bits&cacheStore == 0) && (j == len(ss)) {
            warnstack(ctx, 3, "%08b: %v[%d]: empty cache (%p, %v)\n", bits, ss, i, c, res).debug(12)
            res = nil // it doesn't make sense to fetch a 'empty' cache
        }
    } (cache.value.v, cache.filemapCache)

    a = append(a, cache.filemapCache)

    var comps []string = strings.Split(ss[i], ".")
    if c := cache.comp(ctx, comps, bits); c != nil {
        if j == len(ss) {
            if bits&cacheStore != 0 || c.maps != nil { return c }
        } else if c != cache.filemapCache {
            return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
        }
    }

    if m := cache.match(ctx, ss); m != nil {
        if c := &m.filemapCache; j == len(ss) {
            if bits&cacheStore != 0 || c.maps != nil { return c }
        } else if c != cache.filemapCache {
            return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
        } else {
            errostack(ctx, 3, "%08b: %v[%d]: %s %s\n", bits, ss, i, ss[i], ss[j]).debug(16)
            return
        }
    }

    if c := cache.charstr(ss[i], bits, true); c != nil { // prefixed-patterns: lib*.a test_*.c
        if m := c.match(ctx, ss); m != nil {
            if c := &m.filemapCache; j == len(ss) {
                if bits&cacheStore != 0 || c.maps != nil { return c }
            } else if c != cache.filemapCache {
                return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
            } else {
                errostack(ctx, 3, "%08b: %v[%d]: %s %s\n", bits, ss, i, ss[i], ss[j]).debug(16)
                return
            }
        }
    }

    if s := ss[i]; s == "" {
        return // nothing
    } else if cache.expand(ctx, s) {
        if c := cache.comp(ctx, comps, bits); c != nil {
            if j == len(ss) {
                if bits&cacheStore != 0 || c.maps != nil { return c }
            } else if c != cache.filemapCache {
                return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
            } else {
                errostack(ctx, 3, "%08b: %v[%d]: %s %s\n", bits, ss, i, ss[i], ss[j]).debug(16)
                return
            }
        } else {
            if true  { errostack(ctx, 3, "%08b: %v[%d]\n", bits, ss, i).debug(16) }
            if false { warnstack(ctx, 3, "%08b: %v[%d]\n", bits, ss, i).debug(16) }
            return
        }
    }

    if bits&cacheStore != 0 { return }

    var cs []*filemapCache
    for k := len(a)-2; 0 <= k; k-- {
        var c = a[k].char0(bits)
        if c == nil { continue }
        if p := c.match(ctx, ss); p != nil { return &p.filemapCache }
        cs = append(cs, c)
    }

    var s string = filepath.Join(ss...)
    for _, c := range cs {
        if p := c.match(ctx, s); p != nil { return &p.filemapCache }
    }
    return
}

func (cache hitch) expand(ctx Context, s string) (res bool) {
    var num int
    var p = ctx.Project()
    for v, m := range cache.vals {
        var good bool
        for _, m := range m.maps {
            if good = p == m.project || p.hasBase(m.project); good { break }
        }
        if good {
            for _, elem := range merge(v.expand(ctx, plain)) {
                if elem.expandable(ctx, plain) {
                    if false { warn(ctx, "%s: %s %v ⇒ %s %v", s, typeof(v), v, typeof(elem), elem).debug(1) }
                } else if elem.patterned(ctx) {
                    erro(ctx, "%s: unexpected: %s %v ⇒ %s %v", s, typeof(v), v, typeof(elem), elem).debug(1)
                } else if c := elem.hit(ctx, cache, cacheStore); c != nil {
                    if a, b, c := elem.match(ctx, s); a {
                        if true { warn(ctx, "%v: %s: %v: %s %v ; %v %v %v", p, s, v, typeof(elem), elem, a, b, c) }
                        num += 1
                    }
                } else {
                    if true { warn(ctx, "%s: %s %v -> %s %v", s, typeof(v), v, typeof(elem), elem).debug(1) }
                }
            }
        }
    }
    return num > 0
}

// NOTE: filemapCache.comp is the same as barecomp.hit
func (cache *filemapCache) comp(ctx Context, a []string, bits int) (res *filemapCache) {
    var (
        chars = (bits&(cacheGlob|cacheRegex)) != 0
        N = len(a)-1
    )

    for j, k := range a {
        var chars = chars && j == N
        if c := cache.charstr(k, bits, chars); c != nil {
            if cache = c; j == N { res = c }
        } else if (bits&cacheStore) != 0 {
            errostack(ctx, 3, "%08b: nil: %v[%d]: %s", bits, a, j, k).debug(64)
            break
        } else if j == 0 && k == "" {
            continue // FIXES: .configure, /foo
        } else if !chars && j == N {
            if c = cache.charstr(k, bits, true); c != nil && (bits&cacheStore != 0 || c.maps != nil) { res = c }
        } else { break }
    }
    return
}

func (cache *filemapCache) match(ctx Context, val interface{}) (res *filemapCachePat) {
    for _, m := range cache.pats { for _, p := range m {
        if len(p.maps) == 0 { continue }

        a, b, c := p.value.match(ctx, val)

        if false && !a { if k, v := p.key, p.value.v; k.string(ctx) == "**.c" {
            noted(ctx, "%v(%v) %v -> %v %v %v", typeof(v), v, val, a, b, c).debug(1)

            if false { if t, y := k.(*GlobPattern); y && len(t.components) > 1 {
                t1, t2 := t.components[0], t.components[1]
                noted(ctx, "%v(%v) %v(%v)", typeof(t1), t1, typeof(t2), t2)
            }}

            if false {
                var a, b, c = k.match(ctx, val)
                noted(ctx, "%v(%v) %v -> %v %v %v", typeof(k), k, val, a, b, c).debug(1)
            }
        }}

        if a { return p } else if false { continue }
        if t, r, s := multia(ctx, p.key); t {
            if x, y, z := p.key.match(ctx, val); x {
                erro(ctx, "wrong match: %v %v → %v %v ; %v → %v %v %v",
                    p.key, val, r, s,    p.value/*.v*/, x, y, z).debug(10)
            }
        }
    }}

    if cache.chars != nil { // for all patterns without prefixs, e.g.: *bar
        if c, _ := cache.chars[char(0)]; c != nil { res = c.match(ctx, val) }
    }
    return
}

func (cache *filemapCache) charstr(s string, bits int, chars bool) (res *filemapCache) {
    if chars {
        if s == "" {
            res = cache.char0(bits)
        } else {
            var n int
            for _, r := range s {
                if t := cache.char(char(r), bits); t != nil {
                    cache, n = t, n+1
                }
            }
            if n > 0 { res = cache }
        }
    } else if (bits&cacheStore) != 0 {
        if cache.strs == nil {
            cache.strs = make(map[string]*filemapCache)
        } else {
            res, _ = cache.strs[s]
        }

        if res == nil {
            res = &filemapCache{}
            cache.strs[s] = res
        }
    } else if cache.strs != nil {
        res, _ = cache.strs[s]
    }
    return
}

func (cache *filemapCache) char0(bits int) (res *filemapCache) { return cache.char(char(0), bits) }
func (cache *filemapCache) char(c char, bits int) (res *filemapCache) {
    if (bits&cacheStore) != 0 {
        if cache.chars == nil { cache.chars = make(map[char]*filemapCache) }
        if res, _ = cache.chars[c]; res == nil {
            res = &filemapCache{}
            cache.chars[c] = res
        }
    } else if cache.chars != nil {
        res, _ = cache.chars[c]
    }
    return
}

func (cache *filemapCache) val(ctx Context, key Value, bits int) (res *filemapCache) {
    if (bits&cacheStore) != 0 {
        if cache.vals == nil { cache.vals = make(map[Value]*filemapCache) }
        if res, _ = cache.vals[key]; res == nil {
            res = &filemapCache{}
            cache.vals[key] = res
        }
    } else if cache.vals != nil {
        res, _ = cache.vals[key]
    }

    if res == nil && cast[*universe](ctx).errorUncache {
        errostack(ctx, 10, "%08b: %s(%v) → %s", bits, typeof(key), key, key.string(ctx)).debug(32)
    }
    return
}

func (cache hitch) pat(ctx Context, key Value, bits int) (res *filemapCache) {
    var a []*filemapCachePat
    if s := key.string(ctx); (bits&cacheStore) != 0 {
        if cache.pats != nil { a, _ = cache.pats[s] } else {
            cache.pats = make(map[string][]*filemapCachePat)
        }

        t := &filemapCachePat{filemapCache{}, key, cache.value}
        a = append(a, t)
        cache.pats[s] = a
        return &t.filemapCache
    } else if cache.pats != nil {
        a, _ = cache.pats[s]

        var p = ctx.Project()
        for i, t := range a {
            for j, m := range t.maps {
                if m.project == p || p.hasBase(m.project) {
                    return &t.filemapCache
                } else if false {
                    warn(of(ctx, t.key), "duplications[%d]: %v (%v, %v, %d)", i, t.value, t.key, m.project, j)
                }
            }
        }
    }
    return
}

type hooks struct {
    assert func(Context, Value, bool) bool
}

type packagetype uint8

const (
    packageUnknown packagetype = iota
    packageSmart  // smart package
    packageConfig // pkgconfig
)

type packageinfo struct {
    *Project
    t packagetype // smart, pkgconfig, cmake, etc.
}

func _universe(c Context) *universe { return cast[*universe](c) }

type universe struct {
    diaContext
    commandline
    hooks

    workdir string
    prefix  string // FIXME: prefix for distribution

    scope   *Scope
    globe   *globe

    fset    *FileSet
    paths   searchlist

    packages map[string]packageinfo

    statmutex sync.Mutex
    filemaps filemapCache // value -> dirs
    filecache map[string]*filebase // File.fullname() -> File

    facet_expand_n int64

    ddd string // debug parsing via `eval -ddd=example`, also project.dd
}
func (ctx *universe) String() (s string) { return /*"universe"*/ }
func (ctx *universe) dirtyMark(vals ...Value) { return }
func (ctx *universe) ref(_ Context, _ Value) bool { return false }
func (ctx *universe) isConfigure() bool { return false }
func (ctx *universe) stems() []string { return nil }
func (ctx *universe) entry() Entry { return nil }
func (ctx *universe) loader() *loader { return ctx.globe.top }
func (ctx *universe) Globe() *globe { return ctx.globe }
func (ctx *universe) Scope() *Scope { return ctx.scope }
func (ctx *universe) string() (s string) { if fullContextStringer { s = "{}" }; return }
func (ctx *universe) workDir() (s string) { if s = ctx.workdir; s == "" { s = baseWorkDir }; return }
func (ctx *universe) Project() *Project { return ctx.globe.main }
func (ctx *universe) Position() (p Position) {
    if ctx.globe == nil || ctx.globe.main == nil {
        p.Filename, p.Line = ctx.workDir(), 1
    } else {
        p = ctx.globe.main.position
    }
    return
}
func (ctx *universe) cast(t reflect.Type) (c Context) {
    if false {
        if c = implCast(ctx, t); c == nil { c = ctx.diaContext.cast(t) }
        return
    } else {
        if reflect.TypeOf(ctx) == t { return ctx }
        return ctx.diaContext.cast(t)
    }
}
func (ctx *universe) projects(_ Context, projs ...*Project) []*Project {
    if len(projs) > 0 { panic(failure{"%v",ia(ctx.Position(), projs)}) }
    return nil
}
func (ctx *universe) closure() (scopes []*Scope) {
    if m := ctx.globe.main; m != nil && m.scope != nil {
        if false { scopes = append(scopes, m.scope) }
    }
    return
}

func (ctx *universe) db(ss ...string) (res bool) {
    for _, d := range strings.Fields(ctx.ddd) {
        for _, s := range ss { if d == s { return true }}
    }
    return
}

func (ctx *universe) doHelp()       { do_helpscreen(ctx) }
func (ctx *universe) doHelpFlags()  { print_flag_trace(ctx) }
func (ctx *universe) doHelpConfig() { print_configuration(ctx) }

func init_commandline() commandline { return commandline{
    debugPrompt: true,
    debugErrors: true,
    debugWarns:  true,
    debugInfos:  true,

    silentOptionalSelection: false,

    failOnErrors: true,
    fastMode: true,

    parallel: false, // FIXME: program.traverse not working in parallel

    slow: 9990 * time.Millisecond,
}}

func init_universe(ii ...interface{}) (ctx *universe) {
    ctx = &universe{}

    if s, e := os.Getwd(); e != nil {
        erro(ctx, "%v", e).debug(6)
        return
    } else {
        ctx.workdir = s
        ctx.paths = searchPaths
        ctx.fset = NewFileSet()
        ctx.filecache = make(map[string]*filebase)
        ctx.scope = NewScope(ctx.Position(), nil, nil, `universe`)
    }

    var cl = true
    for _, i := range ii {
        switch t := i.(type) {
        case  commandline: ctx.commandline, cl =  t, false
        case *commandline: ctx.commandline, cl = *t, false
        case hooks: if t.assert != nil { ctx.hooks.assert = t.assert }
        }
    }
    if cl { ctx.commandline = init_commandline() }

    var bin  = ease(ctx, os.Args[0])
    var args = ease(ctx, os.Args[1:])
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART.ARGS", args)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART.BIN", bin)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART", bin)

    for name, f := range builtins {
        if _, alt := ctx.scope.builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
    }

    var pos = ctx.Position()
    var dotos Value
    if strings.Contains(runtime.GOOS, " ") {
        dotos = makeStrlit(pos, runtime.GOOS)
    } else {
        dotos = makeBareword(pos, runtime.GOOS)
    }

    ctx.globe = &globe{
        Scope: NewScope(pos, ctx.scope, nil, `globe "smart"`),
        flagEntries: make(map[string][]Entry),
        loaded: make(map[string]*Project),
        args: make(map[Value][]Value),
    }
    ctx.scope.scopename(ctx, ".GLOBE", ctx.globe.Scope)
    ctx.globe.os,    _ = ctx.globe.define(ctx, DefVoid, ".os", dotos)
    ctx.globe.goals, _ = ctx.globe.define(ctx, DefVoid, ".goals", makeNone(pos))
    ctx.globe.mode,  _ = ctx.globe.define(ctx, DefVoid, ".mode",  makeNull(pos))
    ctx.Context = &positionContext{ctx.Context/* = nil */, pos}
    return
}

func (u *universe) file(filename string, src []byte) *TokFile {
    return u.fset.AddFile(filename, -1, len(src))
}

type filestub struct {
    dir  string      // full directory where the file was or should be found
    sub  string      // matched sub path (see Project.search), may be Dir (absolete path)
    name string      // constant represented name (e.g. relative filename)
    filemap *FileMap // matched pattern (see 'files' directive)
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
        case *Project: dir = t.absPath
        case stat_dir: dir = t.string
        case stat_sub: sub = t.string
        case stat_fileinfo: fileInfo = t.FileInfo
        case stat_nonexist: nonexist = t.bool
        default:
            erro(ctx, "stat: invalid arg: %v(%v)", typeof(i), i).debug(2)
            return
        }
    }

    var (
        base *filebase
        stub *filestub
        fullname string
        u = cast[*universe](ctx)
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
        } else if strings.HasPrefix(fullname, dir+PathSep) {
            tail := fullname[len(dir)+1:]
            //sub  = filepath.Dir(tail)
            //name = filepath.Base(tail)
            if sub == "" { name = tail } else
            if strings.HasPrefix(fullname, sub+PathSep) {
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
            sub = strings.TrimPrefix(sub, PathSep)
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
        dir = filepath.Join(ctx.workDir(), dir)
        fullname = filepath.Join(dir, sub, name)
    }

    // NOTE: filepath.Join can have the same efffect as filepath.Clean
    var cleanFullname = filepath.Clean(fullname) // clean paths like /path/to/foo/../bar -> /path/to/bar
    if base, _ = u.filecache[cleanFullname]; base != nil {
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
        u.filecache[cleanFullname] = base
    }

GotFile:
    file = &File{valbase{ctx.Position()},base,stub}
    return
}

func (u *universe) cache(ctx Context, p *Project, patts, paths []Value) (res []FileMap) {
    var base = &filemap{ p, patts, paths }
    for _, patt := range patts {
        if c := patt.hit(of(ctx, patt), hitch{&u.filemaps, hitched{patt}}, cacheStore); c != nil {
            var m = FileMap{ base, patt }
            c.maps, res = append(c.maps, m), append(res, m)
        } else {
            errostack(of(ctx, patt), 5, "%v: uncached (%T)", patt, patt).debug(16)
        }
    }
    return
}

const uncachedGetError = false

type matchedFileMap struct {
    FileMap
    pattern Value
    name string
}

func (m matchedFileMap) string() string {
    return fmt.Sprintf("{%v, %v, %v}", m.name, m.pattern, m.project)
}

func unmap(ctx Context, name interface{}) (maps []matchedFileMap) {
    var db bool
    var cache *filemapCache
    var t1 time.Time
    defer func (t0 time.Time) {
        var ( t2 = time.Now() ; d = t2.Sub(t0) )
        if db {
            prompt(ctx, "%v: %T %v %p ; %v\n", ctx.Position(), name, name, cache, d).debug(16)
        } else if d > 1*time.Second {
            var ( pos = ctx.Position() ; d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) )
            prompt(ctx, "%v: slow: %T %v\n", pos, name, name)
            prompt(ctx, "%v: slow:→%v\n", pos, maps)
            prompt(ctx, "%v: slow: %v⇒%v\n", pos, d, d1, d2).debug(6)
        }
    } (time.Now())

    const S = ""

    var u = cast[*universe](ctx)
    var h = hitch{&u.filemaps, hitched{name}}
    if v, y := name.(Value); y {
        if cache = v.hit(ctx, h, cacheZero); cache == nil {
            var s = v.string(ctx) ; if S != "" { db = S == s }
            if c := h.char0(cacheZero); c != nil { // FIXES: *.o <-> sub/foo.o
                h.filemapCache = c
                cache = h.strx(ctx, s, cacheZero)
            }
        } else if S != "" { db = S == v.string(ctx) }
    } else if s, y := name.(string); y {
        if cache = h.strx(ctx, s, cacheZero); S != "" { db = S == s }
    } else if a, y := name.([]string); y {
        if cache = h.strs(ctx, a, cacheZero); S != "" { db = S == filepath.Join(a...) }
    } else {
        if uncachedGetError { errostack(ctx, 3, "uncached: %T %v", name, name).debug(16) }
        return
    }

    if t1 = time.Now(); cache == nil { return }

    for _, m := range cache.maps {
        var matched, pattern, s = m.Match(ctx, name)
        if matched { maps = append(maps, matchedFileMap{m, pattern, s}) }
    }

    if len(maps) == 0 && len(cache.maps) > 0 {
        for i, m := range cache.maps { warn(ctx, "%v: %d. %v %v", name, i, m.pattern, m.locs) }
        warn(ctx, "%v (%T)", name, name).debug(1)
    }
    return
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
        hasPrefix(specName, "~"+PathSep, "."+PathSep, ".."+PathSep) {
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
                s = filepath.Join(dc.workDir(), base, specName)
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
        fn = filepath.Join(ctx.workDir(), name)
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
        fn = filepath.Join(ctx.workDir(), name)
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
        if dc.verboseExecFlags { info(of(ctx, flag), "%v", flag) }

        var s = flag.Value.string(ctx)
        var args, _ = dc.globe.args[flag]
        var entries, _ = dc.globe.flagEntries[s]
        for _, entry := range entries {
            var ctx = at(ctx, entry.Position())
            if dc.verboseExecFlags {
                info(ctx, "%v", entry)
                _diaContext(ctx).flush()
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
    var collect func(proj *Project, vals []Value) bool
    collect = func(proj *Project, vals []Value) bool {
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
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case *delegate:
                var s = t.string(ctx)
                if entries := proj.resolveEntries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case flag:
                var s = t.string(ctx)
                if entries := proj.resolveEntries(ctx, s, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
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
    if uni.traceLaunch { defer un(trace(t_launch, "universe.load")) }

    var (
        args []Value
        base = uni.workDir()
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

    uni.globe.top = &loader{ closurecontext: closurecontext{
        ctx, []*Scope{uni.globe.Scope},
    }}

    if text := strings.Join(os.Args[1:], " "); text == "" {
        // Relax!
    } else if args = uni.globe.top.text(ctx, "@", text); len(args) == 0 {
        // ohh...
    } else {
        args = parseOpts(ctx, &uni.commandline, 0, args...)
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
        if p := uni.globe.top.Project(); p != nil { name = p.name }
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

    main   *Project
    loaded map[string]*Project // loaded projects

    stack  []map[string]*def

    args    map[Value][]Value
    flagEntries map[string][]Entry
    flags []flag
    pairs []*pair

    os    *def
    goals *def
    mode  *def
}

// Main returns the main project.
func (g *globe) Main() *Project { return g.main }

func (g *globe) SetScopeOuter(scope *Scope) { scope.outer = g.Scope }

func (g *globe) AddFlagEntry(name string, entry Entry) {
    flags, _ := g.flagEntries[name]
    flags     = append(flags, entry)
    g.flagEntries[name] = flags
    return
}

// project returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *globe) project(ctx Context, outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *Project) {
    if outer == nil { outer = g.Scope }

    m = &Project{
        position: ctx.Position(),
        absPath: absPath,
        relPath: relPath,
        tmpPath: tmpPath,
        use:  new(uselist), // TODO: use scopename instead?
        spec: spec,
        name: name,
    }
    m.scope = NewScope(m.position, outer, m, fmt.Sprintf("project %q", name))
    m.scope.mutex.Lock()
    // outer.mutex.Lock()
    // if s, y := outer.elems[".self"]; y { m.scope.elems[".base"] = s } else {
    //     if true { warn(ctx, "%v: no base", name).debug(12) }
    // }
    // outer.mutex.Unlock()
    m.scope.elems[".self"] = &self{projectname{m,m.scope}}
    m.scope.elems[".usee"] = m.use
    m.scope.mutex.Unlock()
    m.use.name_ = "usee"
    m.use.scope = m.scope
    m.use.owner = m

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
