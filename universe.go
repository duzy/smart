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
    "strings"
    "sync"
    "time"
    "fmt"
    "os"
)

const maxNumVarVal = 9

var uni universe

type searchlist []string
func (sl *searchlist) String() string { return fmt.Sprint(*sl) }
func (sl *searchlist) Set(value string) error {
    *sl = append(*sl, strings.Split(value, ",")...)
    return nil
}

type char rune
func (c char) String() string { return string(rune(c)) }

type filemapCache struct {
    maps []FileMap
    chars map[char  ]*filemapCache // chars; char(0) goes for patterns without bare prefixs
    vals  map[Value ]*filemapCache
    strs  map[string]*filemapCache
    pats  map[string][]*filemapCachePat
}

type filemapCachePat struct {
    filemapCache
    key Value
    value hitched
}

func (cache hitch) strx(ctx Context, s string, bits int) (res *filemapCache) {
    return cache.str(ctx, strings.Split(s, PathSep), 0, bits)
}

func (cache hitch) strs(ctx Context, ss []string, bits int) (res *filemapCache) {
    return cache.str(ctx, ss, 0, bits)
}

func (cache hitch) str(ctx Context, ss []string, i, bits int, a ...*filemapCache) (res *filemapCache) {
    var j = i + 1
    const useAllFetched = false
    defer func(c *filemapCache) {
        if false { if (ss[0] == "llvm") && (false ||
            // ss[i] == "llvm-config.h" ||
            // strings.HasSuffix(ss[i], ".inc") ||
            // strings.HasSuffix(ss[i], ".h") ||
            // strings.HasSuffix(ss[i], ".a") ||
            // strings.HasSuffix(ss[i], ".c") ||
            false) {
            if res != nil {
                if false { warn(ctx, "%v[%d]: %s → %v\n", ss, i, ss[i], res.maps) }
                for k, m := range res.maps {
                    warn(ctx, "%08b: %v[%d]: %s → %d %v %v %v\n", bits, ss, i, ss[i], k, m.project, m.pattern, m.locs)
                }
                warnstack(ctx, 3, "%08b: %v[%d]: %s; %p %p %v", bits, ss, i, ss[i], c, res, res.maps).debug(64)
            } else {
                warnstack(ctx, 3, "%08b: %v[%d]: %s; %p %p, %v\n", bits, ss, i, ss[i], c, res, c.strs["curl"]).debug(64)
            }
        }}
        if res != nil && res.maps == nil && (bits&cacheStore == 0) && (j == len(ss)) {
            if !useAllFetched { warnstack(ctx, 3, "%08b: %v[%d]: empty cache (%p, %v)\n", bits, ss, i, c, res).debug(12) }
            res = nil // it doesn't make sense to fetch a 'empty' cache
        }
    } (cache.filemapCache)

    var aa = strings.Split(ss[i], ".")
    a = append(a, cache.filemapCache)

    if c := cache.comp(ctx, aa, ss, i, bits); c != nil {
        if j == len(ss) {
            if useAllFetched || bits&cacheStore != 0 || c.maps != nil { return c }
        } else if c != cache.filemapCache {
            return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
        } else if useAllFetched {
            errostack(ctx, 3, "%08b: %v[%d]: %s %s\n", bits, ss, i, ss[i], ss[j]).debug(16)
            return
        }
    }

    if m := cache.match(ctx, ss, i, bits); m != nil {
        if c := &m.filemapCache; j == len(ss) {
            if useAllFetched || bits&cacheStore != 0 || c.maps != nil { return c }
        } else if c != cache.filemapCache {
            return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
        } else if true {
            errostack(ctx, 3, "%08b: %v[%d]: %s %s\n", bits, ss, i, ss[i], ss[j]).debug(16)
            return
        }
    }

    if c := cache.char_str(ss[i], bits, true); c != nil { // prefixed-patterns: lib*.a test_*.c
        if false { warn(ctx, "%v[%d]: c.pats %p %v\n", ss, i, c, c.pats).debug(1) }
        if m := c.match(ctx, ss, i, bits); m != nil {
            if false { warn(ctx, "%v[%d]: %p: %v\n", ss, i, m, m.value).debug(1) }
            if c := &m.filemapCache; j == len(ss) {
                if useAllFetched || bits&cacheStore != 0 || c.maps != nil { return c }
            } else if c != cache.filemapCache {
                return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
            } else if true {
                errostack(ctx, 3, "%08b: %v[%d]: %s %s\n", bits, ss, i, ss[i], ss[j]).debug(16)
                return
            }
        }
    }

    if s := ss[i]; s == "" {
        return // nothing
    } else if cache.expand(ctx, s) {
        if c := cache.comp(ctx, aa, ss, i, bits); c != nil {
            if j == len(ss) {
                if useAllFetched || bits&cacheStore != 0 || c.maps != nil { return c }
            } else if c != cache.filemapCache {
                return hitch{c, cache.value}.str(ctx, ss, j, bits, a...)
            } else if true {
                errostack(ctx, 3, "%08b: %v[%d]: %s %s\n", bits, ss, i, ss[i], ss[j]).debug(16)
                return
            } else {
                return
            }
        } else {
            if true  { errostack(ctx, 3, "%08b: %v[%d]\n", bits, ss, i).debug(16) }
            if false { warnstack(ctx, 3, "%08b: %v[%d]\n", bits, ss, i).debug(16) }
            return
        }
    }

    if bits&cacheStore == 0 {
        for k := len(a)-2; 0 <= k; k-- {
            if c := a[k].peel(ctx, cache.value, ss, i, bits); c != nil {
                if false && strings.HasSuffix(ss[len(ss)-1], ".log") {
                    info(ctx, "%08b: %v[%d]: %v, %v\n", bits, ss, i, cache.value, c.maps).debug(16)
                }
                if useAllFetched || bits&cacheStore != 0 || c.maps != nil { return c }
            }
        }
    }

    if false && strings.HasSuffix(ss[len(ss)-1], ".log") {
        for k, a := range a {
            if c := a.chars[char(0)]; c != nil {
                if false { warn(ctx, "%v[%d]: %d. %p %v\n", ss, i, k, c, c.chars) }
                if false { warn(ctx, "%v[%d]: %d. %p %v\n", ss, i, k, c, c.pats) }
                for _, m := range c.pats {
                    for _, p := range m {
                        // if f, s, _ := p.key.match(ctx, ss[i]); f {
                        if f, s, _ := p.value.match(ctx, ss); f {
                            warn(ctx, "%v[%d]: %T %v %v ; %s %v %v\n", ss, i, p.key, p.key, p.value, s, p.chars, p.pats)
                        } else if false {
                            warn(ctx, "%v[%d]: %T %v %v\n", ss, i, p.key, p.key, p.value)
                        }
                    }
                }

                var t = c.match(ctx, ss, i, bits)
                var p interface{}
                if t != nil { p = t.chars[char(0)] }
                warn(ctx, "%v[%d]: %d. %s %p %v\n", ss, i, k, ss[i], t, p).debug(4)
            }
        }
    }
    return
}

func (cache *filemapCache) peel(ctx Context, value hitched, ss []string, i, bits int) (res *filemapCache) {
outermost:
    for ; i < len(ss); i++ {
        if c := cache.chars[char(0)]; c != nil {
            for _, m := range c.pats {
                for _, p := range m {
                    if f, s, m := p.value.match(ctx, ss); f {
                        if false {
                            var v = true // p.value.String() == ".configure/*/*/*.log"
                            if v && strings.HasSuffix(ss[len(ss)-1], ".log") {
                                info(ctx, "%v[%d]: %v %v %v → %v %v\n", ss, i, p.value, p.key, p.maps, s, m).debug(1)
                            }
                        }
                        if cache = &p.filemapCache; len(p.maps) > 0 { return cache } else {
                            i--; continue
                        }
                    }
                }
            }
        } else if true { break outermost } else { erro(ctx, "%v[%d]", ss, i).debug(16) }
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
                if elem.expandible(ctx, plain) {
                    if false { warn(ctx, "%s: %s %v -> %s %v", s, typeof(v), v, typeof(elem), elem).debug(1) }
                } else if elem.patterned(ctx) {
                    erro(ctx, "%s: unexpected: %s %v -> %s %v", s, typeof(v), v, typeof(elem), elem).debug(1)
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

// NOTE: filemapCache.comp is identical to barecomp.hit
func (cache *filemapCache) comp(ctx Context, a, ss []string, i, bits int) (res *filemapCache) {
    var ( chars = (bits&(cacheGlob|cacheRegex)) != 0 ; N = len(a)-1 )
    for j, k := range a {
        var chars = chars && j == N
        if c := cache.char_str(k, bits, chars); c != nil {
            if cache = c; j == N { res = c }
        } else if (bits&cacheStore) != 0 {
            errostack(ctx, 3, "%08b: nil: %v[%d]: %v[%d]: %s", bits, ss, i, a, j, k).debug(64)
            break
        } else if j == 0 && k == "" {
            continue // FIXES: .configure, /foo
        } else if !chars && j == N {
            if c = cache.char_str(k, bits, true); c != nil && (bits&cacheStore != 0 || c.maps != nil) { res = c }
        } else { break }
    }
    return
}

func (cache *filemapCache) match(ctx Context, ss []string, i, bits int) (res *filemapCachePat) {
    if cache.pats != nil {
        for _, m := range cache.pats {
            for _, p := range m {
                if f, _, _ := p.value.match(ctx, ss); f { return p }
            }
        }
    }
    if cache.chars != nil { // for all patterns without prefixs, e.g.: *bar
        if c, _ := cache.chars[char(0)]; c != nil { res = c.match(ctx, ss, i, bits) }
    }
    return
}

func (cache *filemapCache) char_str(s string, bits int, chars bool) (res *filemapCache) {
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

    if res == nil && options.errorUncache {
        errostack(ctx, 10, "%08b: %s(%v) → %s", bits, typeof(key), key, key.Strval(ctx)).debug(32)
    }
    return
}

func (cache hitch) pat(ctx Context, key Value, bits int) (res *filemapCache) {
    var a []*filemapCachePat
    if s := key.Strval(ctx); (bits&cacheStore) != 0 {
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

type universe struct {
    diagContext

    t time.Time

    workdir  string
    prefix   string // FIXME: prefix for distribution

    scope   *Scope
    globe   *Globe

    paths   searchlist
    fset    *FileSet

    statmutex sync.Mutex
    filemaps filemapCache // value -> dirs
    filecache map[string]*filebase // File.fullname() -> File
}
func (ctx *universe) gap(a ...interface{}) (d time.Duration) {
    var t = time.Now()
    if d = t.Sub(ctx.t); a != nil {
        if x, y := a[0].(bool); x && y { ctx.t = t }
    }
    return
}
func (ctx *universe) arguments() []Value { return nil }
func (ctx *universe) argumented() *argumentedContext { return nil }
func (ctx *universe) argumentedSet([]Value) []Value { return nil }
func (ctx *universe) aquireLock() (unlock func()) { return nil }
func (ctx *universe) universe() *universe { return ctx }
func (ctx *universe) loader() *loader { return ctx.globe.top }
func (ctx *universe) parser() *parser { return nil }
func (ctx *universe) inner() Context { return nil }
func (ctx *universe) spawn(c Context) Context { return c }
func (ctx *universe) auto() *autoContext { return nil }
func (ctx *universe) closure() *closureContext { return nil }
func (ctx *universe) travestates(...*travestate) *travestates { return nil }
func (ctx *universe) traversed(target Value) []Value { fail(ctx.Position(), "%v", target); return nil }
func (ctx *universe) entry() Entry { return nil }
func (ctx *universe) entryContext() *entryContext { return nil }
func (ctx *universe) stems() []string { return nil }
func (ctx *universe) stemmed() *stemmed { return nil }
func (ctx *universe) stemmedContext() *stemmedContext { return nil }
func (ctx *universe) Scope() *Scope { return ctx.globe.Scope }
func (ctx *universe) Project() *Project { return ctx.globe.main }
func (ctx *universe) projects(_ Context, projs ...*Project) []*Project {
    if len(projs) > 0 { fail(ctx.Position(), "%v", projs) }
    return nil
}
func (ctx *universe) program() *Program { return nil }
func (ctx *universe) programContext() *programContext { return nil }
func (ctx *universe) positionContext() *positionContext { return nil }
func (ctx *universe) Position() (res Position) {
    res.Filename, res.Line = ctx.workdir, 1
    return
}
func (ctx *universe) wait() {}
func (ctx *universe) dirty(_ Context, args ...Value) (res bool, reason string) { return }
func (ctx *universe) appendCallerUpdated() bool { return false }
func (ctx *universe) mustExists() bool { return false }
func (ctx *universe) WorkDir() string { return ctx.workdir }
func (ctx *universe) Globe() *Globe { return ctx.globe }
func (ctx *universe) String() (s string) {
    if fullContextStringer { s = "default" }
    return
}
func (ctx *universe) isConfiguration() bool { return false }
func (ctx *universe) closureResolveAuto(name string) (obj Object, found bool) { return }
func (ctx *universe) autoArgs(_ []*def, _ []Value) ([]string, error) { return nil, nil }
func (ctx *universe) autoSet(name string, val Value) (def *def, res Value) {
    if false {
        prompt(ctx, "%v: can't set auto in default context, value=%v\n", name, val)
        errostack(ctx, 8, `(%T): %v`, ctx, name).debug(64)
    }
    return
}
func (ctx *universe) autoGet(name string) (res *def) {
    if obj, y := ctx.closureResolveAuto(name); y { res, y = obj.(*def) }
    return
}
func (ctx *universe) closureScopes() (scopes []*Scope) {
    if m := ctx.globe.main; m != nil && m.scope != nil {
        if false { scopes = append(scopes, m.scope) }
    }
    return
}
func (ctx *universe) dirtyOpts() *modifierSetDirtyPatsOpts { return nil }
func (ctx *universe) dirtyMark(vals ...Value) { return }

func (ctx *universe) help()       { do_helpscreen(ctx) }
func (ctx *universe) helpFlags()  { print_flag_trace(ctx) }
func (ctx *universe) helpConfig() { print_configuration(ctx) }

func init() {
    var ctx = &uni
    if s, e := os.Getwd(); e != nil {
        erro(ctx, "%v", e).debug(6)
        return
    } else { ctx.workdir = s }
    ctx.Context = ctx // self context for diagnostic
    ctx.fset = NewFileSet()
    ctx.filecache = make(map[string]*filebase)

    var (
        pos Position = ctx.Position()
        bin = MakeString(pos, os.Args[0])
        args = MakeList(pos)
    )
    for _, a := range os.Args[1:] {
        args.Elems = append(args.Elems, MakeString(pos, a))
    }

    ctx.scope = NewScope(pos, nil, nil, `universe`)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART.ARGS", args)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART.BIN", bin)
    _, _ = ctx.scope.define(ctx, DefVoid, "SMART", bin)

    for name, f := range builtins {
        if _, alt := ctx.scope.Builtin(ctx, name, f); alt != nil {
            panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
    }
    for name, f := range commands {
        if _, alt := ctx.scope.Builtin(ctx, name, f); alt == nil {
            // good
        } else if o, y := alt.(*Builtin); y {
            if f.f != nil { panic(fmt.Sprintf("duplicated command '%s' cannot has a func (%s)", name, typeof(alt))) }
            o.s.b |= f.b // combine the bits only
        } else {
            panic(fmt.Sprintf("builtin '%s' already defined (%s)", name, typeof(alt)))
        }
    }

    ctx.globe = &Globe{
        Scope: NewScope(ctx.Position(), ctx.scope, nil, `globe "smart"`),
        loaded: make(map[string]*Project),
        args: make(map[Value][]Value),
        flagEntries: make(map[string][]Entry),
        //_timestamps: make(map[string]time.Time),
        //_timestampx: new(sync.Mutex),
    }
    _, _ = ctx.scope.ScopeName(ctx, ".GLOBE", ctx.globe.Scope)

    ctx.globe.os,    _ = ctx.globe.define(ctx, DefVoid, ".os",    MakeString(pos, runtime.GOOS))
    ctx.globe.goals, _ = ctx.globe.define(ctx, DefVoid, ".goals", MakeNone(pos))
    ctx.globe.mode,  _ = ctx.globe.define(ctx, DefVoid, ".mode",  MakeNil(pos))
    ctx.t = time.Now()
}

func (uc *universe) file(filename string, src []byte) *TokFile {
    return uc.fset.AddFile(filename, -1, len(src))
}

func (uc *universe) cache(ctx Context, p *Project, patts, paths []Value) (res []FileMap) {
    var base = &filemap{ p, patts, paths }
    for _, patt := range patts {
        if c := patt.hit(of(ctx, patt), hitch{&uc.filemaps, hitched{patt}}, cacheStore); c != nil {
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

func (m matchedFileMap) String() string {
    return fmt.Sprintf("{%v, %v, %v}", m.name, m.pattern, m.project)
}

func (uc *universe) unmap(ctx Context, name interface{}) (maps []matchedFileMap) {
    var cache *filemapCache
    var db bool
    if false { if s, y := name.(string); y && s == "curl/curlver.h" {
        defer func() { info(ctx, "%T %v %p", name, name, cache).debug(16) } ()
        db = true
    } }

    defer func (t time.Time) {
        if d := time.Now().Sub(t); d > time.Duration(1)*time.Second {
            warn(ctx, "%T %v %v", name, name, d).debug(1)
        }
    } (time.Now())

    var h = hitch{&uc.filemaps, hitched{name}}
    if v, y := name.(Value); y {
        cache = v.hit(ctx, h, cacheZero)
        if db { warn(ctx, "%p", cache).debug(1) }
    } else if s, y := name.(string); y {
        cache = h.strx(ctx, s, cacheZero) // checks PathSep
        if db { warn(ctx, "%v %p %p, %v", strings.Split(s, PathSep), h.filemapCache, cache,
            h.filemapCache.strs["curl"]).debug(1) }
    } else if a, y := name.([]string); y {
        cache = h.strs(ctx, a, cacheZero)
        if db { warn(ctx, "%p", cache).debug(1) }
    } else {
        if uncachedGetError { errostack(ctx, 3, "uncached: %T %v", name, name).debug(16) }
        return
    }

    if cache == nil { return }

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

func (dc *universe) search(linfo *loadinfo, specName string) (absPath string, isDir bool, err error) {
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
        for _, base := range uni.paths {
            var s string
            if filepath.IsAbs(base) {
                s = filepath.Join(base, specName)
            } else {
                s = filepath.Join(dc.workdir, base, specName)
            }
            if fi, err = os.Stat(s); err == nil && fi != nil {
                isDir, absPath = fi.IsDir(), s
                return
            }
        }
    }
    return
}

func (dc *universe) run() (result []Value, travestates []*travestate) {
    if options.noRun { return }

    var main = dc.globe.main
    if main == nil {
        erro(dc, "no targets to update `%v`", dc.globe.goals).debug(1)
        return
    }

    var ctx Context = &closureContext{dc, []*Scope{main.scope}}
    if options.verbose { info(ctx, "goal: %v", main).debug(1) }

    removeTempDirs(ctx)
    if false && ddd { info(ctx, "%v", main).debug(1) ; ctx.checkErrors(true) }

    if options.cpuProf != "" || options.autoProfs {
        var prof = options.cpuProf
        if prof == "" {
            var s = "run.cpu.auto.prof"
            if file := main.tempFile(ctx, s); file == nil {
                prof = filepath.Join(baseWorkDir, s)
            } else {
                prof = file.fullname()
            }
        }

        if f, e := os.Create(prof); e != nil {
            erro(dc, "%T: %v", e, e).debug(1)
            return
        } else {
            defer f.Close()
            if e := pprof.StartCPUProfile(f); e != nil {
                erro(dc, "could not start CPU profile: %v", e).debug(1)
                return
            }
            defer pprof.StopCPUProfile()
        }
    }

    if options.memProf != "" || options.autoProfs {
        var prof = options.memProf
        if prof == "" {
            var s = "run.mem.auto.prof"
            if file := main.tempFile(ctx, s); file == nil {
                prof = filepath.Join(baseWorkDir, s)
            } else {
                prof = file.fullname()
            }
        }
        defer func() {
            if f, e := os.Create(prof); e != nil {
                erro(dc, "%v", e).debug(1)
                return
            } else {
                defer f.Close()
                runtime.GC() // update memory statistics
                if e := pprof.WriteHeapProfile(f); e != nil {
                    erro(dc, "could not start CPU profile: %v", e).debug(1)
                    return
                }
            }
        } ()
    }

    var done bool
    for _, flag := range dc.globe.flags {
        if options.verboseExecFlags { info(of(ctx, flag), "%v", flag) }

        var s = flag.name.Strval(ctx)
        var args, _ = dc.globe.args[flag]
        var entries, _ = dc.globe.flagEntries[s]
        for _, entry := range entries {
            var ctx = at(ctx, entry.Position())
            if options.verboseExecFlags {
                info(ctx, "%v", entry)
                ctx.checkErrors(true)
            }

            var ( res []Value; traves []*travestate )
            if res, traves = entry.Execute(ctx, args...); len(traves) > 0 {
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
            case *Nil, *None: // just ignore
            case *bareword:
                if entries := proj.resolveEntries(ctx, t.string, false, true); entries == nil {
                    erro(ctx, "no such entry `%s`", t.string).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case *delegate:
                var s = t.Strval(ctx)
                if entries := proj.resolveEntries(ctx, s, false, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case *Flag:
                var s = t.Strval(ctx)
                if entries := proj.resolveEntries(ctx, s, false, true); entries == nil {
                    erro(ctx, "no such entry `%s` (via `%v`)", s, t).debug(1)
                    return false
                } else {
                    for _, entry := range entries.all {
                        goals = append(goals, entry)
                    }
                }
            case *Argumented:
                {
                    // For examples:
                    //     project-name(-clean)
                    //     project/spec(-clean)
                    //     xxxx()
                    var (
                        s = t.value.Strval(ctx)
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
func (dc *universe) loadTopWork() (err error) {
    if options.traceLaunch { defer un(trace(t_launch, "universe.load")) }
    defer func(l *loader) { dc.globe.top = l } (dc.globe.top)

    var (
        ctx Context = dc
        base = baseWorkDir
        args []Value
    )
    if false && ddd { defer func() { info(dc, "%v", base).debug(10) } () }
    if s := filepath.Join(base, ".smart", "modules"); /* s != "" */true {
        if _, e := os.Stat(s); e == nil { dc.AddSearchPaths(s) }
    }
    if f := filepath.Join(base, entryFileName); f != "" {
        if _, e := os.Stat(f); e != nil {
            f = filepath.Join(base, "build.smart")
            if _, e := os.Stat(f); e != nil { f = "" }
        }
        if f != "" {
            var pos Position
            pos.Filename = f
            pos.Line = 1
            // pos.Column = 1
            ctx = at(ctx, pos)
        }
    }

    dc.globe.top = &loader{
        closureContext: closureContext{ctx, []*Scope{dc.globe.Scope}},
    }

    if text := strings.Join(os.Args[1:], " "); text == "" {
        // Relax!
    } else if args = dc.globe.top.text("@", text); len(args) == 0 {
        // ohh...
    } else {
        args = parseOpts(ctx, &options, 0, args...)
    }

    if v := options.reconfigure; v { options.configure = v }
    if v := options.fastMode; v {
        // Turn off many things for fast mode:
        //options.noImportFiles = v
        options.noDepsGrep = v
        options.noDeps = v
        options.noGrep = v
    }

    if options.verbose {
        defer func(t time.Time) {
            var d = time.Now().Sub(t)
            prompt(ctx, "Goals %v (%s)\n", dc.globe.goals, d)
        } (time.Now())
    }

    assert(dc.globe.args != nil, "globe args is nil")

    if options.autoProfs {
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
    }
    if options.autoProfs {
        defer func() {
            var prof string //= options.memProf
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
        case *Pair: dc.globe.pairs = append(dc.globe.pairs, t)
        case *Flag: dc.globe.flags = append(dc.globe.flags, t)
            if s := t.name.Strval(ctx); s == "clean" {
                mode.position, mode.string = t.position, "clean"
            }
        case *Argumented:
            dc.globe.args[t.value] = t.args
            if f, ok := t.value.(*Flag); ok {
                dc.globe.flags = append(dc.globe.flags, f)
            } else {
                dc.globe.goals.append(ctx, t/*.value*/)
            }
        default:
            dc.globe.goals.append(ctx, t)
        }
    }
    if mode.string == "" {
        if options.configure {
            mode.string = "configure"
        } else {
            mode.string = "goals"
        }
    }
    dc.globe.mode.value = mode

    defer func(t time.Time) {
        var d = time.Now().Sub(t)
        if options.verboseImport {
            var name string
            if p := dc.globe.top.Project(); p != nil { name = p.name }
            fmt.Fprintf(stderr, "└·%s … (%s)\n", name, d)
        } else if d > time.Duration(options.slow)*time.Millisecond*10 {
            if m := dc.globe.main; m != nil {
                warn(at(ctx, m.position), "slow loading (%v)!!\n", d).debug(6)
            } else {
                prompt(ctx, "%s:1:warning: slow loading (%v)!!\n", base, d).debug(6)
            }
        }
    } (time.Now())
    if options.verboseImport { fmt.Fprintf(stderr, "┌→%s\n", base) }

    if false && ddd {
        info(dc, "%v", base).debug(128) ; dc.checkErrors(true)
        defer func() { info(dc, "%v", base).debug(6) ; dc.checkErrors(true) } ()
    }

    if !dc.globe.top.path(base, nil) { return }
    if dc.globe.main == nil { fmt.Fprintf(stderr, "nothing loaded\n") }
    return
}

// A Globe represents a global execution context. 
type Globe struct {
    *Scope

    loads []*loadinfo
    top     *loader

    main   *Project
    loaded map[string]*Project // loaded projects

    stack  []map[string]*def

    args    map[Value][]Value
    flagEntries map[string][]Entry
    flags []*Flag
    pairs []*Pair

    os    *def
    goals *def
    mode  *def
}

// Main returns the main project.
func (g *Globe) Main() *Project { return g.main }

func (g *Globe) SetScopeOuter(scope *Scope) { scope.outer = g.Scope }

func (g *Globe) AddFlagEntry(name string, entry Entry) {
    flags, _ := g.flagEntries[name]
    flags     = append(flags, entry)
    g.flagEntries[name] = flags
    return
}

// project returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *Globe) project(ctx Context, outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *Project) {
    if outer == nil { outer = g.Scope }

    m = &Project{
        position: ctx.Position(),
        absPath: absPath,
        relPath: relPath,
        tmpPath: tmpPath,
        use:  new(uselist), // TODO: use ScopeName instead?
        spec: spec,
        name: name,
    }
    m.scope = NewScope(m.position, outer, m, fmt.Sprintf("project %q", name))
    m.scope.mutex.Lock()
    m.scope.elems[".self"] = &ProjectName{ m, m.scope }
    m.scope.elems[".usee"] = m.use
    m.scope.mutex.Unlock()
    m.use.name = "usee"
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
           uni.paths = append(uni.paths, s)
        }
    }
    return
}
