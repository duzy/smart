//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "bufio"
    "bytes"
    "crypto/sha256"
    "errors"
    "fmt"
    "hash/crc64"
    "io"
    "io/ioutil"
    "os"
    "os/exec"
    "path/filepath"
    "reflect"
    "regexp"
    "strings"
    "sync"
    "syscall"
    "time"
)

// NOTE: all single character opt names/shortcuts should be preserved for general purposes.
type general_opts struct {
    debug    int  `db,dbg,debug` // NOTE: compatible with 'bool'
    stack    int  `stack,stack-number`
    fail     bool `fail` // fail on errors
    fullname bool `full,fullname,fullfile`
    silent   bool `silent` // force silent, contrast 'verbose'
    timing   bool `time,timing`
    verbose  bool `verb,verbose` // prompts more information
    warning  bool `warn,warning` // prompts more warnings
}

type modifier_ struct { Context ; general_opts }
type modifier_v interface{ v(...Value) any }
type modifier_x interface{ x(...Value) any }
type modifier_y interface{ x(*execution, ...Value) any }

var modifier_v_t = reflect.TypeOf((*modifier_v)(nil)).Elem()
var modifier_x_t = reflect.TypeOf((*modifier_x)(nil)).Elem()
var modifier_y_t = reflect.TypeOf((*modifier_y)(nil)).Elem()
var (
    modifiers = map[string]reflect.Type{
        `debug`:        reflect.TypeOf((*modifier_debug)(nil)).Elem(),
        `print`:        reflect.TypeOf((*modifier_print)(nil)).Elem(),
        `prompt`:       reflect.TypeOf((*modifier_prompt)(nil)).Elem(),

        `preserve`:     reflect.TypeOf((*modifier_preserve)(nil)).Elem(),
        `expand`:       reflect.TypeOf((*modifier_expand)(nil)).Elem(),
        `plain`:        reflect.TypeOf((*modifier_plain)(nil)).Elem(),
        `stringify`:    reflect.TypeOf((*modifier_stringify)(nil)).Elem(),
        `reveal`:       reflect.TypeOf((*modifier_reveal)(nil)).Elem(),
        `disclose`:     reflect.TypeOf((*modifier_disclose)(nil)).Elem(),

        `select`:       reflect.TypeOf((*modifier_select)(nil)).Elem(),

        `env`:          reflect.TypeOf((*modifier_env)(nil)).Elem(), // interpreter environments
        `var`:          reflect.TypeOf((*modifier_var)(nil)).Elem(),
        `set`:          reflect.TypeOf((*modifier_set)(nil)).Elem(),
        `defer`:        reflect.TypeOf((*modifier_defer)(nil)).Elem(),

        `closure`:      reflect.TypeOf((*modifier_closure)(nil)).Elem(),

        `cd`:           reflect.TypeOf((*modifier_cd)(nil)).Elem(),
        `mkdir`:        reflect.TypeOf((*modifier_mkdir)(nil)).Elem(),

        `sudo`:         reflect.TypeOf((*modifier_sudo)(nil)).Elem(),

        `touch`:        reflect.TypeOf((*modifier_touch)(nil)).Elem(),
        `grep`:         reflect.TypeOf((*modifier_grep)(nil)).Elem(),
        `deps`:         reflect.TypeOf((*modifier_deps)(nil)).Elem(),

        `copy-file`:       reflect.TypeOf((*modifier_copyfile)(nil)).Elem(),
        `write-file`:      reflect.TypeOf((*modifier_writefile)(nil)).Elem(),
        `read-file`:       reflect.TypeOf((*modifier_readfile)(nil)).Elem(),
        `update-file`:     reflect.TypeOf((*modifier_updatefile)(nil)).Elem(),
        `configure-input`: reflect.TypeOf((*modifier_configureinput)(nil)).Elem(),
        `configure-file`:  reflect.TypeOf((*modifier_configurefile)(nil)).Elem(),
        // `configure`:       reflect.TypeOf((*modifier_configure)(nil)).Elem(),

        `wait`:         reflect.TypeOf((*modifier_wait)(nil)).Elem(),
        `stamp`:        reflect.TypeOf((*modifier_stamp)(nil)).Elem(),

        `check`:        reflect.TypeOf((*modifier_check)(nil)).Elem(),
        `assert`:       reflect.TypeOf((*modifier_assert)(nil)).Elem(),
        `case`:         reflect.TypeOf((*modifier_case)(nil)).Elem(),
        `cond`:         reflect.TypeOf((*modifier_cond)(nil)).Elem(),
        `if`:           reflect.TypeOf((*modifier_cond)(nil)).Elem(),
        `where`:        reflect.TypeOf((*modifier_cond)(nil)).Elem(),

        `once`:         reflect.TypeOf((*modifier_once)(nil)).Elem(),

        `fork`:         reflect.TypeOf((*modifier_fork)(nil)).Elem(),

        `git-ahead`:    reflect.TypeOf((*modifier_gitahead)(nil)).Elem(),
        `git-modified`: reflect.TypeOf((*modifier_gitmodified)(nil)).Elem(),

        `by`:           reflect.TypeOf((*modifier_setDirtyPats)(nil)).Elem(),
        `dirty`:        reflect.TypeOf((*modifier_predictDirty)(nil)).Elem(),
    }

    crc64Table = crc64.MakeTable(crc64.ECMA)//crc64.ISO
)

type is_modify struct{}
type    modify_ctx struct{ Context }
func (c modify_ctx) inner() Context { return c.Context }
func (c modify_ctx) cast(t reflect.Type) Context { return icast(c,t) }
func (c modify_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case is_modify: return true
    }
    return c.Context.do(ctx, op)
}

func modify(ctx Context, g *group, hyphen bool) (res Value) {
    var name = __string(ctx, g.elems[0])
    var args = g.elems[1:]

    if t, y := modifiers[name]; !y {
        var _, e, _ = entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s failed for %s\n", e, name, _project(ctx))
        debug(ctx, "unknown modifier: %s (args=%v)", name, args, trace{})
    } else {
        var exe = _execution(ctx)
        var mv = reflect.New(t)
        var mi = mv.Interface()

        var fv modifier_v
        var fx modifier_x
        var fy modifier_y
        if !hyphen {
            if fv, _ = mi.(modifier_v); fv == nil {
                debug(ctx, "%v: no method: (*%s).v(...)", name, typeof(mi), trace{})
            }
        } else if fx, _ = mi.(modifier_x); fx == nil {
            if fy, _ = mi.(modifier_y); fy == nil {
                debug(ctx, "%v: no method: (*%s).x(...)", name, typeof(mi), trace{})
            } else if exe == nil {
                debug(ctx, "%v: nil execution: (*%s).x(...)", name, typeof(mi), trace{})
            }
        }

        if c := mv.Elem().FieldByName("Context"); c.IsValid() {
            c.Set(reflect.ValueOf(modify_ctx{pc(ctx, g)})) // c.Type().String() == "smart.Context"
        } else {
            debug(ctx, "%v: no field: %s.Context", name, typeof(mi), trace{})
        }

        args = _opts(ctx, mv, args)

        if fv != nil {
            res = ease(ctx, fv.v(args...))
        } else if fx != nil {
            res = ease(ctx, fx.x(args...))
        } else if fy != nil {
            res = ease(ctx, fy.x(exe, args...))
        }
    }

    if !hyphen {
        // $- remains
    } else if res == nil {
        res = _null(g.position) // $- remains too
    } else if name == "defer" || name == "set" || name == "var" {
        debug(ctx, "invalid result: (set ...) ⇒ %v", res, trace{})
    } else if a := _automatic(ctx); a != nil {
        a.amend(ctx, "-", res)
    }
    return
}

type modifier struct { group }
func (m *modifier) kind() Kind { return m.group.kind()|KindModifier }
func (m *modifier) _cmp(ctx Context, v Value) (_ cmpres) {
    if x, y := v.(*modifier); y { return cmp(ctx, &m.group, &x.group) }
    return
}

type modification struct { valbase ; list []*modifier }
func (_ *modification) kind() Kind { return KindModification }
func (g *modification) _cmp(ctx Context, v Value) (res cmpres) {
    if o, y := v.(*modification); y && len(g.list) == len(o.list) {
        for i, m := range g.list {
            if t := cmp(ctx, m, o.list[i]); t != cmpEqual { return t }
        }
        res = cmpEqual
    }
    return
}
func (g *modification) String() (s string) {
    s = "{"
    for i, m := range g.list {
        if i > 0 { s += " " }
        if m != nil { s += m.String() }
    }
    s += "}"
    return
}

func getGroupElem(value Value, n int, v Value) Value {
    if g, y := value.(*group); y {
        if elem := g.at(n); elem != nil {
            v = elem
        }
    }
    return v
}

func promptShellResult(ctx Context, value Value, n int) {
    if g, y := value.(*group); y && g != nil {
        if elem := g.at(0); elem != nil {
            if str := __string(ctx, elem); str == "shell" {
                if elem = g.at(n); elem != nil {
                    if str = __string(ctx, elem); strings.HasSuffix(str, "\n") {
                        prompt(ctx, "%s", str)
                    } else if str != "" {
                        prompt(ctx, "%s\n", str)
                    }
                }
            }
        }
    }
    return
}

type modifier_debug struct { modifier_
    cond   Value `if,cond,where,when`
    info []Value `info`
    warn []Value `warn`
    erro []Value `err,erro,error`
    checkOutdated bool `dirty,checkdirty,check-dirty,check-outdated`
    trave int `tr,trave,traverse`
    s int `stack,stack-number`
    n int `count,num,call-number`
}
func (ctx *modifier_debug) x(args ...Value) (result any) {
    if ctx.cond != nil && !__true(ctx, ctx.cond) { return }
    if ctx.s == 0 && ctx.stack > 0 { ctx.s = ctx.stack }
    if ctx.n == 0 && ctx.debug > 0 { ctx.n = ctx.debug }
    for _, v := range ctx.info { info(ctx, "%s", __string(ctx, v)) }
    for _, v := range ctx.warn { warn(ctx, "%s", __string(ctx, v)) }
    for _, v := range ctx.erro { debug(ctx, "%s", __string(ctx, v), trace{}) }

    var (
        target  = auto_get(ctx, "@")
        depends = auto_get(ctx, "^")
    )
    if ctx.checkOutdated && target != nil {
        var (
            ordered = auto_get(ctx, "|")
            grepped = auto_get(ctx, "~")
            tt = statFile(ctx, target).mod()
        )
        if tt.IsZero() {
            debug(ctx, "target not exists: %v", target)
            return
        }
        for _, dep := range merge(depends, ordered, grepped) {
            if dt := statFile(ctx, dep).mod(); dt.After(tt) {
                debug(ctx, "%v: outdated by %v (%v)", target, dep, dt.Sub(tt))
            }
        }
    }
    // if len(ctx.info) == 0 && len(ctx.warn) == 0 && len(ctx.erro) == 0 {
    //     var ( p = _position(ctx) ; s = _stems(ctx) ; m diagtracer )
    //     if len(args) == 0 {
    //         m = prompt(ctx, "%v: target=%v stems=%v depends=%v\n", p, target, s, depends)
    //     } else if ctx.verbose {
    //         m = prompt(ctx, "%v: target=%v stems=%v depends=%v ; %v\n", p, target, s, depends, args)
    //     } else if len(args) == 1 {
    //         m = prompt(ctx, "%v: %v (%T)\n", p, args[0], args[0])
    //     } else {
    //         m = prompt(ctx, "%v: %v\n", p, args)
    //     }
    //     if ctx.s > 0 { m = infostack(ctx, ctx.s) }
    //     if ctx.n > 0 { m.debug(ctx.n) }
    // }
    return
}

type modifier_print struct { modifier_
    stdout bool `o,stdout`
    stderr bool `e,stderr` // TODO: = true
    reset  bool `r,reset`
}
func (ctx *modifier_print) x(args ...Value) (result any) {
    var content string
    if val := auto_get(ctx, "-"); val != nil { content = __string(ctx, val) }
    if ctx.stdout { fmt.Fprint(stdout, content) }
    if ctx.stderr { fmt.Fprint(stderr, content) }
    if ctx.reset  { auto_set(ctx, defVoid, "-", _none(_position(ctx))) }
    return
}

type modifier_prompt struct { modifier_ }
func (ctx *modifier_prompt) x(args ...Value) (result any) {
    if len(args) == 0 {
        if h := auto_get(ctx, "-"); h != nil {
            prompt(ctx, "%s\n", __string(ctx, h))
        }
    } else {
        for _, a := range args { prompt(ctx, "%s\n", __string(ctx, a)) }
    }
    return
}

type modifier_preserve struct { modifier_ }
func (ctx *modifier_preserve) v(args ...Value) (result any) {
    return args
}

type modifier_expand struct { modifier_ }
func (ctx *modifier_expand) v(args ...Value) (result any) {
    result = expands(ctx, args...)
    return
}

type modifier_plain struct { modifier_ }
func (ctx *modifier_plain) v(args ...Value) (result any) {
    result = expands(_final(ctx), args...)
    return
}

type modifier_stringify struct { modifier_ }
func (ctx *modifier_stringify) v(args ...Value) (result any) {
    result = expands(_final(ctx), args...)
    return
}

type modifier_reveal struct { modifier_ }
func (ctx *modifier_reveal) v(args ...Value) (result any) {
    result = expands(original{ctx,defExpand1}, args...)
    return
}

type modifier_disclose struct { modifier_ }
func (ctx *modifier_disclose) v(args ...Value) (result any) {
    result = expands(original{ctx,defExpand2}, args...)
    return
}

// select element by index from group result: (select 0)
type modifier_select struct { modifier_ }
func (ctx *modifier_select) x(args ...Value) (_ any) {
    if h := auto_get(ctx, "-"); h == nil {
        debug(ctx, "no pipe value $-", trace{})
    } else if x, y := h.(*group); y {
        var vals []Value
        for _, a := range xmerge(ctx, args...) {
            vals = append(vals, x.at(int(__int(ctx, a))))
        }
        return vals
    }
    return
}

type modifier_env struct { modifier_ }
func (ctx *modifier_env) x(args ...Value) (result any) {
    args = xmerge(ctx, args...)

    if exe := _execution(ctx); exe != nil {
        for _, a := range args {
            if p, y := a.(*pair); y {
                exe._env = append(exe._env, p)
            } else {
                debug(ctx, "env: not a pair value: %s", ts(a), trace{})
            }
        }
    }
    return
}

type modifier_var struct { modifier_ }
func (ctx *modifier_var) x(args ...Value) (_ any) {
    if checkpoints {
        if args != nil {
            debug(ctx, "%v", args, trace{})
        }
        switch p := _project(ctx); p.name {
        case "configure.base":
        }
    }
    return // no-op
}

// examples:
//     [(set name=value)]    set $(name) to 'value'
//     [(set name)]          clear $(name)
//     [(set -)]             clear $-
type modifier_set struct { modifier_ }
func (ctx *modifier_set) x(args ...Value) (_ any) {
    for _, arg := range args {
        var name string
        var value Value
        switch a := arg.(type) {
        case *word: name = a.s
        case *pair: // NOTE: pair.Value is not expanded, need to do it again.
            name, value = __string(ctx, a.key), expand(_final(ctx),a.val)
            if value == nil { value = a.val }
        case flag:
            name, value = __string(ctx, a.Value), _null(a.Position())
            if name == "" { name = "-" }
        default:
            debug(ctx, "%v is unsupported (try: foo=value)", ts(arg), trace{})
        }

        var d, _ = auto_set(ctx, defVoid, name, value)
        if d == nil {
            debug(ctx, "no auto set: %v : %v", name, ts(ctx), trace{})
        }

        if name == "@" {
            var f, s, _ = as{value}.fullname_file(ctx)
            if ctx.verbose {
                var ts = trimPromptString(s)
                prompt(ctx, "%s …… traversed (%d)\n", ts, f._traved)
            }
        }
    }
    return
}

type modifier_defer struct { modifier_ }
func (ctx *modifier_defer) x(a ...Value) (_ any) {
    if x := _execution(ctx); x != nil { x.defers = append(x.defers, a...) }
    return
}

type modifier_setDirtyPats struct { modifier_
    pats []Value
}
func (ctx *modifier_setDirtyPats) x(args ...Value) (result any) {
    var opts, y = do(ctx, propDirtyOpts).(*dirtyOpts)
    if y { ctx.pats = parseOpts(_final(ctx), opts, args...) }
    return
}

// create closure context for the traversal
type modifier_closure struct { modifier_
    target Value `@,target`
    // depFirst bool `<,dep-first` // TODO: -<=value
    // depLast  bool `>,dep-last` // TODO: ->=value
}
func (ctx *modifier_closure) x(exe *execution, args ...Value) (result any) {
    // Closure the caller program, the context will be restored when execution is finished.
    var cc = exe.Context
    exe.Context = closure_with(cc)

    if false && cast[*term](ctx) != exe.Context {
        debug(ctx, "wrong closure_with", trace{})
    }

    var proj = _project(ctx)
    var set = func(name string, val Value) (t Value) {
        var noop bool
        if v, y := val.(*boolean); y {
            if !v.bool { noop = true }
        } else if isTrivial(val) {
            debug(ctx, "trivial target: %T %v", val, val, trace{})
        } else if true {
            t = expand(ctx,val) //, plain
        } else {
            t = val
        }

        if l, y := t.(*list); y && len(l.elems) == 1 { t = l.elems[0] }
        if !noop && isTrivial(t) { t = auto_get(ctx, name)  }

        if t != nil {
            auto_set(ctx, defVoid, name, t) // aka (set @=&@)
        } else if !noop {
            debug(ctx, "%v: %s is nil", proj, name, trace{})
        }
        return
    }

    var target Value
    if ctx.target != nil {
        if target = expand(ctx,ctx.target); target == ctx.target {
            if t := auto_get(cc, "@"); t != nil {
                target = t
            }
        }
    }
    if ctx.verbose { var t = target
        debug(ctx, "%v: @: %v ⇒ %v %v", proj, ctx.target, typeof(t), t)
    }
    if target != nil {
        var ( t = as{set("@", target)} ; f *file ; s string ; y bool ; n int )
        if f, s, y = t.fullname_file(ctx); !y {
            s = __string(ctx, t)
        } else {
            n = f._traved
        }

        if n > 1 {
            if ctx.verbose {
                var ts = trimPromptString(s)
                prompt(ctx, "%s …… traversed (%d, %v)\n", ts, n)
                if false { debug(ctx, "%v, %v, (%d)", f, s, n) }
            }
            return
        }

        // FIXME: if isInnerauto_get(ctx, t.Value) {
        //     debug(ctx, "loop: %v", t, trace{})
        //     return
        // }
    }

    if proj == nil {
        debug(ctx, "%T: nil project in the context", ctx, trace{})
    } else if scope := proj.scope; scope == nil {
        debug(ctx, "empty closure context", trace{})
    } else if def := scope.finddef("/"); def == nil {
        debug(ctx, "&/ is undefined", trace{})
    } else if dir := __string(ctx, def.value); dir == "" {
        debug(ctx, "&/ is empty", trace{})
    } else if !filepath.IsAbs(dir) {
        debug(ctx, "&/ is relative", trace{})
    } else /* if err := enter(ctx, dir); err == nil */ {
        exe.changedWD = dir
    }
    return
}

type modifier_cd struct{ modifier_
    path bool `path`
    printEnter bool `print-enter`
    printLeave bool `print-leave`
}
func (ctx *modifier_cd) x(args ...Value) (result any) {
    if (ctx.printEnter || ctx.printLeave) && len(args) == 0 { return }
    if len(args) == 1 {
        var dir = __string(ctx, args[0])
        if dir == "" {
            // TODO: do something special
            return
        }

        var proj = _project(ctx)
        if !filepath.IsAbs(dir) { dir = filepath.Join(proj.absPath, dir) }
        if ctx.path && dir != "." && dir != ".." && dir != pathSep {// mkdir -p
            if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                debug(ctx, "make path '%s' failed: %v", dir, err)
            }
        }
        if exe := _execution(ctx); exe != nil { exe.changedWD = dir }
    } else {
        debug(ctx, "wrong number of cd args: %v", args, trace{})
    }
    return
}

type modifier_mkdir struct { modifier_
    mode os.FileMode `mode`
}
func (ctx *modifier_mkdir) x(args ...Value) (result any) {
    if ctx.mode == 0 {
        ctx.mode = os.FileMode(0755)
    } else {
        ctx.mode |= os.FileMode(0111)
    }
    if len(args) == 0 {
        if v := auto_get(ctx, "@"); !isTrivial(v) { args = append(args, v) }
    }
    for _, a := range xmerge(_final(ctx), args...) {
        var s string
        if x, y := a.(*file); y {
            s = x.fullname()
        } else {
            s = __string(ctx, a)
        }
        if strings.Contains(s, " /") || strings.Contains(s, " ./") || strings.Contains(s, " ../") {
            debug(ctx, "multiple paths (%v): '%v'", typeof(a), s, trace{})
        } else if strings.Contains(s, " ") {
            debug(ctx, "path containing spaces (%v): '%v'", typeof(a), s)
        }
        if e := os.MkdirAll(s, ctx.mode); e != nil {
            debug(ctx, "path: %v(%v) ⇒ %s: %v", typeof(a), a, s, e, trace{})
        }
    }
    return
}

type modifier_sudo struct { modifier_ }
func (ctx *modifier_sudo) x(args ...Value) (result any) {
    debug(ctx, "TODO: sudo modifier is not implemented yet", trace{})
    return
}

func parseDependList(ctx Context, dependList *list) (depends *list) {
    depends = new(list)
    for _, depend := range dependList.elems {
        switch d := depend.(type) {
        case *list:
            if dl := parseDependList(ctx, d); dl != nil {
                depends.elems = append(depends.elems, dl.elems...)
            }
        case *exec_result:
            if d.Status != 0 {
                debug(ctx, "bad status %v", d.Status, trace{})
            } else {
                depends.append(d)
            }
        case *rule, *strlit, *file:
            depends.append(d)
        default:
            debug(ctx, "unsupported entry depend `%v' (%v)", depend, _program(ctx).depends, trace{})
        }
    }
    return
}

type langInfoT struct {
    rxs []*regexp.Regexp
    sys []*regexp.Regexp
}

var langInfos = map[string]*langInfoT{
    "asm": &langInfoT{
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*"(.+)".*$`),
        },
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*<(.+)>.*$`),
        },
    },
    "c": &langInfoT{
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*"(.+)".*$`),
        },
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*#\s*include\s*<(.+)>.*$`),
        },
    },
    "i": &langInfoT{
        []*regexp.Regexp{
            regexp.MustCompile(`^\s*include\s*"(.+)".*$`),
        },
        []*regexp.Regexp{
        },
    },
}

func init () {
    if info, ok := langInfos["c"]; ok {
        langInfos["c++"] = info
        langInfos["clang"] = info
        langInfos["objc"] = info
        langInfos["objc++"] = info
    }
    if info, ok := langInfos["i"]; ok {
        langInfos["include"] = info
        langInfos["TableGen"] = info
        langInfos["td"] = info
    }
}

var grepCacheFilebase = make(map[*filebase]*grepCacheFiles)
type grepCacheFiles struct {
    file *file
    list []*file
}
type greptouch struct {
    files []Value
    target as
    targetInfo os.FileInfo
    targetDir string // see splitTargetFileName
    targetFullName string // see splitTargetFileName
}
type grepctx struct {
    *modifier_grep
    greptouch
    report bool // discard or report missing greps
    rxs []*greprex
    done map[string]int
    savedGrepFileName string
    savedGrepFile *file
    save *bufio.Writer
}
type greprex struct{ bool ; *regexp.Regexp }
func (g *greptouch) work(ctx Context, gc *grepctx) (err error) {
    if g.targetInfo == nil {
        debug(ctx, "'%v' not exists", g.target, trace{})
    }
    var tt time.Time = g.targetInfo.ModTime()
    for _, val := range g.files {
        var file *file
        if file, _ = to_file(val); file == nil {
            debug(ctx, "'%v' is not file (%T)", val, val, trace{})
        }
        if file.info == nil && !file.isSysFile() {
            if file.info, _ = os.Stat(__string(ctx, file)); file.info == nil { continue }
            if gc.debug>0 { warn(ctx, "'%v' info is nil (%s)", file, file.fullname()) }
        }
        if file.info == nil {/* ... */} else
        if t := file.info.ModTime(); t.After(tt) {
            if gc.debug>0 { warn(ctx, "touch %v → %v (%v)", g.target, file, t) }
            if tt != t { tt = t }
        }
    }
    if tt.After(g.targetInfo.ModTime()) {
        if err = os.Chtimes(g.targetFullName, tt, tt); err != nil {
            debug(ctx, "%v", err, trace{})
        }
    }
    return
}
func (g *grepctx) isTargetFile(ctx Context, file *file) (res bool) {
    if file == nil {
        // ...
    } else if g.target.Value == file {
        res = true
    } else if s, _ := g.target.fullname_string(ctx); s == g.targetFullName {
        res = true
    } else if f, y := to_file(g.target.Value); y && ident(ctx,f) == ident(ctx,file) {
        res = true
    }
    return
}

var grepcache = make(map[string][]Value)
var grepcacheM sync.Mutex // avoid fatal error: concurrent map writes

func loadGrepCache(ctx Context) {
    s := joinTmpPath(ctx, "", "cache")
    f, err := os.Open(s)
    if err != nil { return } else { defer f.Close() }
    var ( list []Value ; k string )
    scanner := bufio.NewScanner(f)
	// Allocate a 64KB initial buffer, but allow it to grow up to 10MB per line!
    const maxCapacity = 10 * 1024 * 1024 
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, maxCapacity)
    scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
        s = scanner.Text()
        if strings.HasPrefix(s, ":") { //
            if k != "" && len(list) > 0 {
                grepcache[k] = list
            }
            if len(list) > 0 { list = list[:0] }
            k = s[1:]
        } else {
            a := strings.Split(s, "|")
            if len(a) == 3 {
                file := _stat(ctx, a[0], stat_sub{a[1]}, stat_dir{a[2]})
                if file != nil {
                    list = append(list, file)
                }
            }
        }
    }
}

func saveGrepCache(ctx Context) {
    s := joinTmpPath(ctx, "", "cache")
    f, err := os.OpenFile(s, os.O_RDWR|os.O_CREATE, 0666)
    if err != nil { return } else { defer f.Close() }
    var w = bufio.NewWriter(f)    ; defer w.Flush()
    grepcacheM.Lock(); defer grepcacheM.Unlock()
    for k, l := range grepcache {
        if len(l) == 0 { continue }
        fmt.Fprintf(w, ":%s\n", k)
        for _, v := range l {
            var file, ok = to_file(v)
            if !ok { continue }
            fmt.Fprintf(w, "%s|%s|%s\n", ident(ctx,file), file.sub, file.dir)
        }
    }
}

func searchGreppedName(ctx Context, gp Position, gc *grepctx, sys bool, name string) (res *file) {
    var isAbs, isRel bool
    if isAbs = filepath.IsAbs(name); isAbs {
        res = _stat(ctx, name, stat_nonexist{true})
    } else if isRel = isRelPath(name); isRel { // relative to targetDir
        res = _stat(ctx, name, stat_dir{gc.targetDir}, stat_nonexist{true})
    } else if res = findfile(ctx, name); res != nil && res.exists() {
        return // found existed file
    }

    // System files are not treated as missing nor collected
    // for further updating, just discard them immediately.
    if !sys && res != nil && res.filemap != nil && len(res.filemap.paths) == 1 {
        // system files defined by `files ((foo.xxx) ⇒ -)`
        if f, ok := res.filemap.paths[0].(flag); ok {
            sys = isNone(f.Value) || isNull(f.Value)
        }
    }
    if !sys && gc.debug>0 {
        debug(ctx, "%v: %v → %v (exists=%v, sys=%v, from %v)\n",
            _entry(ctx), gc.target, name, res.exists(), sys, _project(ctx),
			callstack{num:gc.debug})
    }
    if sys || res.exists() { return }

    // relative to target directory
    var alt = _stat(ctx, name, stat_dir{gc.targetDir})
    if alt != nil { res = alt; return }

    // Check for bare non-system sub-paths:
    //   foo/bar/name.xxx
    // We search base name 'name.xxx' again:
    var s = filepath.Dir(name) // e.g: foo/bar

    // Search 'name.xxx' and check dir for
    // 'foo/bar' suffix. We use it if found.
    alt = findfile(ctx, filepath.Base(name))
    if alt != nil && strings.HasSuffix(alt.dir, pathSep+s) {
        dir := strings.TrimSuffix(alt.dir, pathSep+s)
        ok1 := alt.change(dir, s, ident(ctx,alt)) // <dir>, foo/bar, name.xxx
        ok2 := alt.change(dir, "", name) // <dir>, "", foo/bar/name.xxx
        res  = alt
        if enable_assertions {
            assert(ok1, "unchanged: %s %s %s", dir, s, ident(ctx,alt))
            assert(ok2, "unchanged: %s %s", dir, ident(ctx,alt))
        }
    } else if res == nil {
        for _, inc := range gc.incs {
            if res = _stat(ctx, name, stat_dir{__string(ctx, inc)}); res != nil {
                if false { debug(ctx, "%v in %v", res, inc) }
                return
            }
        }
        if res == nil { res = _stat(ctx, name, stat_nonexist{true}) }
        debug(ctx, _f("'%s' not found in %v", name, _project(ctx)),
			_f("grepped '%s' has no target dir in %v", name, _project(ctx)),
			_f("from project %v (for %v)", _project(ctx), name))
    }
    return
}

func searchGrepped(ctx Context, gp Position, gc *grepctx, sys bool, name string) (file *file, err error) {
    if file = searchGreppedName(ctx, gp, gc, sys, name); file == nil {
        // The 'name' is not matching the files database.
        if gc.discard { return }
        // FIXME: missing-file error
    } else if gc.isTargetFile(ctx, file) {
        return
    } else if !file.exists() && gc.discard {
        return
    } else if gc.files = append(gc.files, file); false && gc.touch {
        var tt = gc.targetInfo.ModTime()
        if file.info == nil && !file.isSysFile() {
            if file.info, err = os.Stat(__string(ctx, file)); err != nil {
                debug(ctx, "%v", err, trace{})
            }
            if false || gc.debug>0 {
                debug(ctx, "'%v' info is nil (%s)", file, file.fullname(), callstack{num:gc.debug})
            }
        }
        if file.info == nil {/* ... */} else
        if tv := file.info.ModTime(); tv.After(tt) {
            if true || gc.debug>0 {
                debug(ctx, "touch %v → %v (%v)", gc.target, file, tv, callstack{num:gc.debug})
            }
            tv = launchTime //time.Now() // ...
            if err, tt = os.Chtimes(gc.targetFullName, tv, tv), tv; err != nil {
                debug(ctx, "chtimes failed: %v", err, trace{})
            }
        }
    }

    // Report missing files, but system files are not treated as missing.
    if !gc.report {
        // ...
    } else if file == nil {
        info(ctx, "%s: `%s` not found", _project(ctx).name, name)
    } else if !file.exists() {
        info(ctx, "%s: `%s` file not existed", _project(ctx).name, name)
    }
    return
}

func tempfile(ctx Context, prefix, hashee0 string, hasheeN... any) (file *file, err error) {
    var nameHash = sha256.New() // HashByte -> [sha256.Size]byte
    if _, err = fmt.Fprint(nameHash, prefix, hashee0); err != nil {
        debug(ctx, "hashing failed: %v", err, trace{})
    } else if _, err = fmt.Fprint(nameHash, hasheeN...); err != nil {
        debug(ctx, "hashing failed: %v", err, trace{})
    } else if nameSum := nameHash.Sum(nil); len(nameSum) != nameHash.Size() {
        debug(ctx, "hash sum invalid: %v", len(nameSum), trace{})
    } else if project := _project(ctx); project == nil {
        debug(ctx, "current project is nil: %v", ctx, trace{})
    } else {
        // Make names like .deps/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
        // .deps/??/??/????????????????????????????????????????????????????????????
        // .grep/??/??/????????????????????????????????????????????????????????????
        // .cache/??/??/????????????????????????????????????????????????????????????
        file = project.tempfile(ctx, filepath.Join(prefix, // e.g. ".deps", ".grep"
            fmt.Sprintf("%x", nameSum[ :1]),
            fmt.Sprintf("%x", nameSum[1:2]),
            fmt.Sprintf("%x", nameSum[2: ]),
        ))
    }
    return
}

func removeTempDirs(ctx Context, cleanDirs ...string) {
    var uni = _universe(ctx)
    if len(cleanDirs) == 0 {
        var clean =  uni.cleanTmpDirs
        if  clean || uni.cleanDotCache { cleanDirs = append(cleanDirs, ".cache") }
        if  clean || uni.cleanDotDeps  { cleanDirs = append(cleanDirs, ".deps") }
        if  clean || uni.cleanDotGrep  { cleanDirs = append(cleanDirs, ".grep") }
    }
    for _, dir := range cleanDirs {
        if file, err := tempfile(ctx, dir, ""); err != nil {
            debug(ctx, "%v", err, trace{})
        } else if s := file.fullname(); s == "" {
            debug(ctx, `"%v" has no fullname`, file, trace{})
        } else if s = filepath.Dir(filepath.Dir(filepath.Dir(s))); s == "" {
            debug(ctx, `"%v" is invalid temp dir`, file.fullname(), trace{})
        } else if err = os.RemoveAll(s); err != nil {
            debug(ctx, "%v", err, trace{})
        } else if false {
            debug(ctx, "%s: removed %v", _project(ctx), s)
        } else {
            prompt(ctx, "%s: removed %v\n", _project(ctx), s)
        }
    }
}

func getSavedDepsFileName(ctx Context, targetFullName string, strs []string) (filename string, err error) {
    var ( file *file; hashees []any )
    for _, s := range strs { hashees = append(hashees, s) }
    if file, err = tempfile(ctx, ".deps", targetFullName, hashees...); err != nil {
        debug(ctx, "get .deps temp file failed: %v", err, trace{})
    } else {
        filename, _ = as{file}.fullname_string(ctx)
    }
    return
}

func getSavedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
    var ( file *file )
    if file, err = tempfile(ctx, ".grep", targetFullName); err != nil {
        debug(ctx, "get .grep temp file failed: %v", err, trace{})
    } else {
        filename, _ = as{file}.fullname_string(ctx)
    }
    return
}

func loadSavedGrepFile(ctx Context, gc *grepctx) (okay bool, err error) {
    if gc.savedGrepFileName, err = getSavedGrepFileName(ctx, gc.targetFullName); err != nil {
        debug(ctx, "get saved grep filename failed: %v", err, trace{})
    } else if gc.savedGrepFile = _stat(ctx, gc.savedGrepFileName); gc.savedGrepFile == nil {
        return // No saved grepfile yet!
    }

    var f, ok = to_file(gc.target)
    if !ok {
        f = _stat(ctx, gc.targetFullName)
        if f != nil { gc.target.Value = f }
    }
    if f != nil && f.info != nil {
        // Check previously saved grep file into.
        if f.info.ModTime().After(gc.savedGrepFile.info.ModTime()) {
            return
        }
    }

    var savedGrepOSFile *os.File
    if savedGrepOSFile, err = os.Open(gc.savedGrepFileName); err != nil {
        debug(ctx, "open saved grep filename failed: %v", err, trace{})
    }
    defer savedGrepOSFile.Close()

    var gp Position
    //gp.Filename = gc.savedGrepFileName
    gp.Filename = gc.targetFullName

    scanner := bufio.NewScanner(savedGrepOSFile)
	// Allocate a 64KB initial buffer, but allow it to grow up to 10MB per line!
    const maxCapacity = 10 * 1024 * 1024 
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, maxCapacity)
    scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
        var s = scanner.Text() //gp.Line += 1
        var ( sys int; name string )
        if n, e := fmt.Sscanf(s, "%d %d %d %s", &sys, &gp.Line, &gp.Column, &name); e == nil && n == 4 {
            var f *file
            if f, err = searchGrepped(ctx, gp, gc, sys == 1, name); err != nil {
                debug(ctx, "search grepped filename failed: %v", err, trace{})
            } else if f != nil {
                f.position = gp
                if gc.isTargetFile(ctx, f) { continue }
            } else if sys != 1 && !gc.discard {
                debug(ctx,
					_f("%s is nil file", name),
					_f("grepped %s is nil", name),
					_f("from project %v", _project(ctx)))
            }
        }
    }
    if gc.savedGrepFile.info, err = savedGrepOSFile.Stat(); err != nil {
        debug(ctx, "stat saved grep filename error: %v", err, trace{})
    } else { okay = true }
    return
}

func grepTargetFile(ctx Context, gc *grepctx) (err error) {
    var ( f *os.File )
    if f, err = os.Open(gc.targetFullName); err != nil {
        debug(ctx, "%v", err, trace{})
    } else { defer func() { err = f.Close() } () }

    var gp Position
    gp.Filename = gc.targetFullName

    scanner := bufio.NewScanner(f)
	// Allocate a 64KB initial buffer, but allow it to grow up to 10MB per line!
    const maxCapacity = 10 * 1024 * 1024 
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, maxCapacity)
    scanner.Split(bufio.ScanLines)
ForScan:
    for scanner.Scan() {
        var s = scanner.Text(); gp.Line += 1
        for _, x := range gc.rxs {
            if sm := x.FindStringSubmatch(s); len(sm) > 1 && sm[1] != "" {
                var ( f *file ; name = sm[1]; sys = x.bool ) //strings.IndexFunc(s, isNotSpace)
                if gp.Column = strings.Index(s, name); gc.save != nil {
                    var d = 0 ; if sys { d = 1 } // system files
                    fmt.Fprintf(gc.save, "%d %d %d %s\n", d, gp.Line, gp.Column, name)
                }
                if f, err = searchGrepped(ctx, gp, gc, sys, name); err != nil {
                    debug(ctx, "search grepped '%s' failed: %v", name, err, trace{})
                } else if f != nil {
                    if f.position = gp; gc.isTargetFile(ctx, f) { continue }
                } else if !sys && !gc.discard {
                    debug(ctx,
						_f("%s is nil file", name),
						_f("grepped %s is nil", name),
						_f("from project %v", _project(ctx)))
                }
                continue ForScan // found one
            }
        }
    }
    return
}

func grep(ctx Context, gc *grepctx) (err error) { // TODO: using ctx.grepping() to replace grepctx
    var targetName string
    switch v := gc.target.Value.(type) {
    case *file:
        targetName = ident(ctx,v)
        gc.targetInfo = v.info
        gc.targetFullName = v.fullname()
        gc.targetDir = filepath.Dir(gc.targetFullName)
        if v.isSysFile() { return }
    default:
        gc.targetDir = _project(ctx).absPath
        targetName = __string(ctx, v)
        if filepath.IsAbs(targetName) {
            gc.targetFullName = targetName
        } else {
            gc.targetFullName = filepath.Join(gc.targetDir, targetName)
        }
        if file := _stat(ctx, gc.targetFullName); file == nil {
            debug(ctx, "grep: '%s' not found (%v)", gc.targetFullName, gc.target, trace{})
        } else {
            gc.targetInfo = file.info
        }
    }
    if err != nil {
        debug(ctx, "grep target %s: %v", targetName, err, trace{})
    }

    if gc.targetInfo == nil { return }
    if gc.done == nil { gc.done = make(map[string]int) }
    if !filepath.IsAbs(gc.targetFullName) {
        debug(ctx, "grep: '%s' is not abs", gc.targetFullName, trace{})
    } else {
        gc.done[gc.targetFullName] += 1
    }
    if n, done := gc.done[gc.targetFullName]; done && n > 1 {
        if gc.debug>0 {
            debug(ctx, "%v (done %v)", gc.targetFullName, n, trace{})
        }
        return
    }

    //var infos = strings.Contains(gc.targetFullName, "...")
    const infos = false

    if false { defer un(tt(l_traverse, _execution(ctx), gc.target)) }

    defer func(restore []Value) {
        var t = _execution(ctx)
        var touch = gc.greptouch // copy greptouch value
        if len(touch.files) > 0 {
            grepcacheM.Lock()
            grepcache[gc.targetFullName] = touch.files
            grepcacheM.Unlock()
        } else if false {
            var gp Position
            gp.Filename, gp.Line = gc.targetFullName, 1
            debug(ctx, "grebbed zero files: %v", gc.targetFullName)
        }
        gc.files = restore
        if gc.debug>0 {
            debug(ctx, "grepped: %s → %v (grepped=%v) (saved=%s)\n",
                gc.target, touch.files, len(t.grepped), gc.savedGrepFile,
				callstack{num:gc.debug})
        }
        for _, gc.target.Value = range touch.files {
            if t.grepped = append(t.grepped, gc.target); !gc.recursive {
                continue
            } else if err = grep(ctx, gc); err != nil {
                debug(ctx, "grep files (deferred): %v", err, trace{})
            }
        }
        if err == nil && gc.touch {
            if err = touch.work(ctx, gc); err != nil {
                debug(ctx, "grep touch failed: %v", err, trace{})
            }
        }
    } (gc.files)

    gc.files = nil

    var (
        cached bool
        savedGrepFile *os.File
        savedGrepFileLoaded bool
    )
    {
        grepcacheM.Lock()
        gc.files, cached = grepcache[gc.targetFullName]
        grepcacheM.Unlock()
    }
    if cached && len(gc.files) > 0 {
        if gc.debug>0 {
            debug(ctx, "grepcache: %v → %v", gc.targetFullName, gc.files, trace{})
        }
        return
    } else if infos {
        debug(ctx, "grepcache: %s files=%d", gc.targetFullName, len(gc.files))
    }

    if savedGrepFileLoaded, err = loadSavedGrepFile(ctx, gc); err != nil {
        debug(ctx, "load saved grepfile failed: %v", err, trace{})
    } else if savedGrepFileLoaded && len(gc.files) > 0 {
        if infos {
            debug(ctx, "loadSavedGrepFile: %v files=%d grepped=%d",
				gc.targetFullName, len(gc.files), len(_execution(ctx).grepped))
        }
        return
    }
    if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
            debug(ctx, "make grep dir failed: %v", err, trace{})
        }
    }

    var uni = _universe(ctx)
    if uni.saveGrepSource {
        var (
            perm = os.FileMode(0600)
            data = []byte(gc.targetFullName)
            name = gc.savedGrepFileName + ".src"
        )
        if err = ioutil.WriteFile(name, data, perm); err != nil {
            debug(ctx, "grep write file: %v", err, trace{})
        } else if false {
            debug(ctx, "saved grep %s", name)
        }
    }
    if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil {
        debug(ctx, "grep create %s: %v", gc.savedGrepFileName, err, trace{})
    }

    gc.save = bufio.NewWriter(savedGrepFile)
    defer func() {
        gc.save.Flush()
        savedGrepFile.Close()
    } ()

    if err = grepTargetFile(ctx, gc); err != nil && !gc.discard {
        debug(ctx, "grep target file: %v", err, trace{})
    } else {
        err = nil // discard any errors
    }
    return
}

var stopgrep = 0

// grep - grep files from target, example usage:
//
//      (grep -file -x='\s*#\s*include\s*<(.*)>')
//
// https://github.com/google/re2/wiki/Syntax
type modifier_grep struct { modifier_
    discard bool `c,cast;dc,discard;dm,discard-missing;im,ignore-missing`
    fileinc bool `f,file;f,files` // work with the 'incs' field TODO: = true
    langs []string `l,lang;lan,language`
    sys []string `s,sys;ss,system`        // matching system includes
    reg []string `re,reg;regx,regex;x,rx` // matching user includes
    incs []Value `i,inc;i,include` // include search paths, also 'fileinc' field
    touch bool `t,touch;t,touch-outdate;t,touch-outdated`
    recursive bool `a,all;r,recur;rr,recursive`
    noTraverse bool `n,notraverse;nt,no-traverse;go,grep-only`
}
func (ctx *modifier_grep) x(args ...Value) (result any) {
    var uni = _universe(ctx)
    if false && uni.noDepsGrep || uni.noGrep { return }

    var gc = grepctx{ modifier_grep:ctx }
    // gc.fileinc = true // grep files by default
    gc.incs = xmerge(ctx, gc.incs...)//, plain
    for _, s := range gc.sys { gc.rxs = append(gc.rxs, &greprex{true , regexp.MustCompile(s)}) }
    for _, s := range gc.reg { gc.rxs = append(gc.rxs, &greprex{false, regexp.MustCompile(s)}) }
    for _, s := range gc.langs {
        if info, ok := langInfos[s]; ok && info != nil {
            for _, re := range info.rxs { gc.rxs = append(gc.rxs, &greprex{false, re}) }
            for _, re := range info.sys { gc.rxs = append(gc.rxs, &greprex{true , re}) }
        } else {
            debug(ctx, "lang '%s' is unknown", s, trace{})
        }
    }
    if len(gc.rxs) == 0 {
        debug(ctx, "no grep expressions: %v %v %v %v", gc.sys, gc.reg, gc.langs, args, trace{})
    }

    var (
        target = auto_get(ctx, "@")
        targets = args
        grepped = _execution(ctx).grepped
    )
    if len(targets) == 0 {
        if target == nil || isNull(target) || isNone(target) {
            debug(ctx, "no grep target", trace{})
        } else {
            targets = append(targets, target)
        }
    }

    if gc.debug > 0 {
        debug(ctx, "grep files: %v %v %v\n", target, gc.rxs, args, callstack{num:gc.debug})
    }
    if gc.verbose {
        defer func(ts time.Time) {
            var s string
            if len(targets) == 1 { s = targets[0].String() } else {
                for _, v := range targets {
                    if s != "" { s += ", " }
                    if len(s) > 32 { s += "..."; break } else {
                        s += v.String()
                    }
                }
            }
            debug(ctx, "Grep %v …… (%d files in %v)\n", s, len(grepped), time.Now().Sub(ts))
        } (time.Now())
    }

    var pc = _execution(ctx)
    var tar = target
    defer func(v bool) { pc.grepping = v } (pc.grepping)
    pc.grepping = true

    for _, target := range targets {
        if isNull(target) {
            debug(ctx, "found nil grep target for %v", tar, trace{})
        }
        if isNone(target) {
            debug(ctx, "grep target '%v' is none for %v", target, tar, trace{})
        }

        gc.target.Value, pc.grepped = target, nil
        if err := grep(ctx, &gc); err != nil {
            debug(ctx, "grep files from %v failed: %v", target, err, trace{})
        } else if gc.noTraverse {
            // does nothing
        } else if len(pc.grepped) > 0 {
            for _, val := range pc.grepped {
                traverse(ctx, val)
            }
        }
        grepped = append(grepped, pc.grepped...)
    }
    pc.grepped = grepped

    if !gc.noTraverse {
        auto_set(ctx.Context, defVoid, "~", _none(_position(ctx)))
        pc.grepped = nil
    } else {
        result = ease(ctx, pc.grepped)
    }
    return
}

type dep_context struct { diagnostic }
func (ctx *dep_context) inner() Context { return &ctx.diagnostic }
func (ctx *dep_context) cast(t reflect.Type) Context {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}

func parseDeps(ctx Context, targetVal Value, targetStr string, savedDepsFile *file, savedDepsFileName, deps string) (files []Value) {
    var (
        proj = _project(ctx)
        targetFullName, _ = as{targetVal}.fullname_string(ctx)
        filesMux sync.Mutex
        firstWord string
        err error
    )
    var findDepFile = func(name string) (file *file) {
        if filepath.IsAbs(name) {
            file = _stat(ctx, name, stat_nonexist{true})
        } else if file = proj.file(ctx, name); file != nil && file.exists() {
            // good!
        } else {
            // fail!
        }
        return
    }
    var ignored = func(fullname string) (res bool) {
        if fullname == targetFullName { return true }
        return
    }
    var addFile = func(file *file) {
        filesMux.Lock()
        files = append(files, file)
        filesMux.Unlock()
    }
    var (
        missing = make(map[string]Position)
        missMux sync.Mutex
    )

    var depFile = func(ctx Context, depPos Position, word string) {
        var dc = dep_context{diagnostic{ Context: ctx }}

        ctx = &dc

        if i := strings.Index(word, " "); i > 0 {
            debug(ctx, "ignore dep with spaces: %v", word)
        } else if file := findDepFile(word); file == nil {
            prompt(ctx, "%v: unknown dep\n", file)
            if savedDepsFile != nil {
                warn(ctx, "unknown dep '%v' for '%v'", word, firstWord)
                warn(ctx, "from here: %s", word)
                if filepath.IsAbs(firstWord) {
                    var wp Position
                    wp.Filename, wp.Line = firstWord, 1
                    warn(ctx, "in here: %v", word)
                }
                debug(ctx, "for project %v", proj)
            } else {
                debug(ctx, "unknown dep '%v' for '%v'", word, firstWord)
                debug(ctx, "from here: %s", word)
                if filepath.IsAbs(firstWord) {
                    var wp Position
                    wp.Filename, wp.Line = firstWord, 1
                    debug(ctx, "in here: %v", word)
                }
                debug(ctx, "for project %v", proj)
            }
        } else if ignored(file.fullname()) {
            //continue // dep is the target itself
        } else {
            traverse(ctx, file)
            addFile(file)
        }

        var n int
        if savedDepsFile == nil {
            if n = flush(dc.Context); n > 0 { // aka. dc.points = nil
                var s = trimPromptString(targetVal.String())
                prompt(ctx, "%v: %d errors counted\n", word, n)
                debug(ctx, `%v: %d errors for "%s", dep "%s"`, proj, n, s, word)
                debug(ctx, `%v: %v`, ctx, trace{})
            }
        } else {
            if n = diagCount(dc.Context, diagError); n > 0 {
                // reset to reduce diags as we wish to continue with the errors
                dc.points, dc.erros = nil, 0
                debug(ctx, "%v: %d errors counted\n", word, n)
            }
        }
        if n > 0 {
            missMux.Lock()
            missing[word] = depPos
            missMux.Unlock()
        }
        return
    }

    var (
        wordRecs = make(map[string]int)
        firstDep string
        depPos Position
    )
    depPos.Filename = savedDepsFileName
    for l, line := range strings.Split(deps, "\n") {
        var words = line
        if i := strings.Index(words, ":"); i > 0 { words = strings.TrimSpace(words[i+1:]) }
        if words = strings.TrimSpace(strings.TrimRight(words, "\\\r\t ")); words == "" {
            continue // empty line
        }
        for _, word := range strings.Fields(words) {
            depPos.Line, depPos.Column = l + 1, strings.Index(line, word) + 1
            if /*l == 1 && w == 0 &&*/firstWord == "" { firstWord = word }
            if wordRecs[word] += 1; wordRecs[word] == 1 {
                if firstDep != "" {
                    // keep going...
                } else if firstDep = word; savedDepsFile == nil {
                    // no need to compare
                } else if firstDepFile := _stat(ctx, firstDep); firstDepFile == nil {
                    return nil // requests to update savedDepsFile
                } else if firstDepFile.info.ModTime().After(savedDepsFile.info.ModTime()) {
                    return nil // requests to update savedDepsFile
                }
				depFile(ctx, depPos, word)
            }
        }
    }
    if len(missing) > 0 {
        prompt(ctx, "%v: %d deps missing, removing deps file\n", savedDepsFileName, len(missing))
        if savedDepsFile == nil || savedDepsFileName == "" {
            // deps files not saved yet
        } else if err = os.Remove(savedDepsFileName); err != nil {
            for s, _ := range missing { debug(ctx, `missing "%v"`, s) }
            debug(ctx, `%v: "%v" %d deps missing in "%v"`, proj, targetVal, len(missing), savedDepsFileName)
            debug(ctx, "%s", ts(ctx), trace{})
        } else {
            for s, _ := range missing { warn(ctx, `missing "%v"`, s) }
            debug(ctx, `%v: "%v" missing %d deps (%v in total)`, proj, targetVal, len(missing), len(files))
            files = nil // To update savedDepsFileName
        }
    }
    return
}

func loadSavedDepsAndCheckOutdated(ctx Context, args []string) (savedDepsFileName string, files []Value) {
    var (
        savedDepsBytes []byte
        err error
    )
    if targetVal, targetStr := auto_target_valstr(ctx); targetVal == nil {
        debug(ctx, "target is nil", trace{})
    } else if targetStr == "" {
        debug(ctx, "target '%v' is empty", targetVal, trace{})
    } else if savedDepsFileName, err = getSavedDepsFileName(ctx, targetStr, args); err != nil {
        debug(ctx, "get saved deps filename failed: %v", err, trace{})
    } else if savedDepsFileName == "" {
        debug(ctx, "empty saved deps filename", savedDepsFileName, trace{})
    } else if savedDepsFile := _stat(ctx, savedDepsFileName); savedDepsFile == nil {
        // no saved deps file
    } else if savedDepsBytes, err = ioutil.ReadFile(savedDepsFileName); err != nil {
        debug(ctx, "can'ctx open saved deps file: %v", savedDepsFileName, err, trace{})
    } else if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, string(savedDepsBytes)); len(files) > 0 {
        if false { debug(ctx, "loaded deps %s (%d files)", savedDepsFileName, len(files)) }
        var savedDepsFileModTime = savedDepsFile.info.ModTime()
        for _, val := range files { if file, ok := to_file(val); !ok {
            // ignore
        } else if file.info.ModTime().After(savedDepsFileModTime) {
            files = nil // need to reload if outdated
            return
        }}
    }
    return
}

func traverseMissingDep(ctx Context, dep string) (res bool) {
    if proj := _project(ctx); proj == nil {
        prompt(ctx, "%s: traverse dep failed, project %v\n", dep, proj)
        debug(ctx, "%s: no current project for dep", dep)
        debug(ctx, trace{})
    } else if f := proj.file(ctx, dep); f == nil {
        prompt(ctx, "%s: dep is unknown file; project %v\n", dep, proj)
        debug(ctx, "%v: %s is unknown file", proj, dep)
        debug(ctx, trace{})
    } else {
        traverse(ctx, f)
    }
    return true
}

func traverseMissingDeps(ctx Context, lastTry string, errBytes []byte) (res bool, tried string) {
    const promptErrors bool = false
    const promptBeforeTraverse bool = promptErrors && true
    for _, m := range rxFileNotFound.FindAllSubmatch(errBytes, -1) {
        if promptBeforeTraverse { prompt(ctx, "%s\n", m[0]) }
        if dep := string(m[4]); dep == lastTry {
            return false, ""
        } else if res = traverseMissingDep(ctx, dep); !res {
            prompt(ctx, "%s: dep missing, project %v\n", m[4], _project(ctx))
            prompt(ctx, "%s\n", m[0]) // prompt the entire error line
            debug(ctx, "%v", ctx, trace{})
        } else if tried == "" { tried = dep }
    }
    return
}

type modifier_deps struct { modifier_
    useClang bool `cl,clang`
    useGcc bool `gcc`
    addMissing bool `am,add-missing,mg,missing-goal`
    lang string `lang,language`
    flags []Value `flags,opts`
    cc string `cc,compiler`
}
func (ctx *modifier_deps) x(args ...Value) (result any) {
    var uni = _universe(ctx)
    if uni.noDepsGrep || uni.noDeps { return }

    // NOTE: parse opts for (deps) before expanding the args, because we share args
    //       with the compilers!
    var err error
    var targetVal Value
    var targetStr string
    if targetVal, targetStr = auto_target_valstr(ctx); targetVal == nil {
        debug(ctx, "target is nil", trace{})
    } else if targetStr == "" {
        debug(ctx, "target '%v' is empty", targetVal, trace{})
    }

    var files []Value
    if ctx.verbose {
        defer func(ts time.Time) {
            var s string
            if val := auto_get(ctx, "@"); val != nil { s = val.String() }
            prompt(ctx, "Deps %v …… (%d files in %v)\n", s, len(files), time.Now().Sub(ts))
        } (time.Now())
    }

CorrectCC:
    switch ctx.cc {
    case "cl"   : ctx.cc = "clang"; goto CorrectCC
    case "gc"   : ctx.cc = "gcc"  ; goto CorrectCC
    case "clang": ctx.useClang = true
    case "gcc"  : ctx.useGcc   = true
    case "":
        if ctx.useGcc   { ctx.cc = "gcc" }
        if ctx.useClang { ctx.cc = "clang" }
    default:
        if base := filepath.Base(ctx.cc); base == "" {
            debug(ctx, "unsupported cc: %v", ctx.cc, trace{})
        } else if strings.HasPrefix(base, "clang") { ctx.useClang = true
        } else if strings.HasPrefix(base, "gcc")   { ctx.useGcc   = true }
    }

    var _MM, _MG bool
    var ca []string
    var flags = xmerge(_final(ctx), ctx.flags...)
    for _, f := range flags {
        switch s := strings.TrimSpace(__string(ctx, f)); s {
        case "-MM": ca, _MM = append(ca, s), true // only user headers
        case "-MD": break // discard, use -M or -MM instead
        case "-MP": break // discard, not creating phony target
        case "-MV": break // discard, not using NMake/Jom format
        case "-MG": break // discard, add later for missing headers
        case "-M" : break // discard, add later for both user and system headers
        case "-c" : break // discard, compile flag
        case ""   : break // discard, empty string
        default: ca = append(ca, s)
        }
    }
    if !_MM { ca = append(ca, "-M")  } // both user and system headers
    if !_MG && ctx.addMissing { ca = append(ca, "-MG") } // add missing headers
    for _, a := range args {
        var s, y = as{a}.fullname_string(ctx)
        if y { s = strings.TrimSpace(s) }
        switch s {
        case "-M", "-MM", "-MG", "-MD", "-MV", "-MP", "-Os", "-O1", "-O2", "-O3",
            "-c", "-shared", "-static", "-fPIC", "-fvisibility-inlines-hidden",
            "-fcxx-modules", "-fmodules", "-fmodules-ts", "":
            break // discard unused args
        default: ca = append(ca, s)
        }
    }

    var proj = _project(ctx)

    savedDepsFileName, files := loadSavedDepsAndCheckOutdated(ctx, ca)

    if len(files) == 0 {
        var (
            cc = exec.Command(ctx.cc, ca...)
            stdout bytes.Buffer
            stderr bytes.Buffer
            retried string
        )
    retryCC:
        cc.Stdout, cc.Stderr = &stdout, &stderr
        if err = cc.Run(); err != nil {
            var okay = false
            if okay, retried = traverseMissingDeps(ctx, retried, stderr.Bytes()); okay {
                cc = exec.Command(ctx.cc, ca...)
                stdout.Reset()
                stderr.Reset()
                goto retryCC
            }
            prompt(ctx, "%v: failed command '%s':\n", proj, ctx.cc)
            prompt(ctx, "%s \\\n  %s\n----------\n", cc.Path, strings.Join(ca, " \\\n  "))
            prompt(ctx, "%s\n----------\n%s----------\n", &stdout, &stderr)
            debug(ctx, "%s: %s deps failed: %v", proj, filepath.Base(ctx.cc), err)
            debug(ctx, "%s: %v", proj, ctx, trace{})
        }
        if stderr.Reset(); savedDepsFileName == "" {
            stdout.Reset()
            debug(ctx, "empty saved deps file name: %v", savedDepsFileName, trace{})
        }

        var savedDepsFile *file = nil//_stat(ctx, savedDepsFileName)
        if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, stdout.String()); len(files) == 0 {
            debug(ctx, "parse deps file failed") // not saving if failed
        } else if err = os.MkdirAll(filepath.Dir(savedDepsFileName), os.FileMode(0755)); err != nil {
            debug(ctx, "make path '%s' failed: %v", filepath.Dir(savedDepsFileName), err, trace{})
        } else if err = ioutil.WriteFile(savedDepsFileName, stdout.Bytes(), os.FileMode(0666)); err != nil {
            debug(ctx, "save deps file failed: %v", err, trace{})
        }
        stdout.Reset() // release buffers (optional)
    }

    if t := _execution(ctx); t != nil && len(files) > 0 {
        t.grepped = append(t.grepped, files...)
    }
    return
}

type modifier_touch struct { modifier_
    path bool `p,path`
    mode os.FileMode `m,mode`
}
func (ctx *modifier_touch) x(args ...Value) (result any) {
    if len(args) == 0 { if val := auto_get(ctx, "@"); val != nil { args = append(args, val) }}

    var files []*file
    for _, arg := range args {
        if err := touch(ctx, arg, uint32(ctx.mode), ctx.path); err != nil {
            debug(ctx, "touch '%v' failed: %v", arg, err, trace{})
        } else {
            files = append(files, stampFile(must_files_stamp{ctx}, arg)...)
        }
    }

    var p = _program(ctx)
    if false && ctx.verbose { reportFileUpdates(ctx, files) }
    if len(p.getModifiers(ctx, "stamp")) > 0 {
        debug(ctx, "no need to use a (stamp) after (touch)")
    }
    return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
type modifier_check struct { modifier_
    trim bool `trim,trim-string`
    answer bool `answer`
    boolean bool `bool,boolean,res,result`
    silent bool `slient`
    exists bool `exist,exists`
    regular bool `reg,regular`
    isdir bool `isdir,is-dir`
    good bool `good`
    file Value `file`
    dir Value `dir`
}

func (ctx *modifier_check) x(args ...Value) (_ any) {
    var pos = _position(ctx)
    var makeResult func(bool) Value // returns results only if non-nil
    if ctx.answer {
        makeResult = func(v bool) Value { return _answer(pos, v) }
    } else if ctx.boolean ||
        (ctx.file != nil && (ctx.exists || ctx.regular || ctx.isdir)) ||
        (ctx.dir  != nil && (ctx.exists || ctx.regular || ctx.isdir)) {
        makeResult = func(v bool) Value { return _boolean(pos, v) }
    }

    var res bool
    var values []Value
    var checkfile = func (val Value, dir bool) {
        if val == nil {
            debug(pc(ctx, val), "nil file value to check", trace{})
        } else if x, y := val.(*boolean); y {
            if x.bool { val = auto_get(ctx, "@") } else { val = nil }
        }

        var s string
        var f *file
        if f, res = to_file(val); res {
            // best case
        } else if s = __string(ctx, val); filepath.IsAbs(s) {
            if f = _stat(ctx, s); f != nil { res = true }
        } else if f = findfile(ctx, s); f != nil { res = true }

        if f != nil {
            if !dir || ctx.regular {
                res = f.exists()
            } else if dir || ctx.isdir {
                res = f.info != nil && f.info.Mode().IsDir()
            } else if ctx.exists {
                res = f.exists()
            }
        }

        if makeResult != nil {
            values = append(values, makeResult(res))
        } else if !res {
            debug(pc(ctx, val), "'%v' is not file", val, trace{})
        }
    }

    if ctx.file != nil { checkfile(ctx.file, false) }
    if ctx.dir  != nil { checkfile(ctx.dir, true) }

    var program = _program(ctx)
    var value = auto_get(ctx, "-")

argsloop:
    for _, arg := range args {
        var p, y = arg.(*pair)
        if !y {
            if res = __true(ctx, arg); makeResult != nil {
                values = append(values, makeResult(res))
            } else {
                debug(ctx, "value '%v' is false", arg, trace{})
            }
            continue
        }

        var key, str string
        switch key = __string(ctx, p.key); key {
        case "status":
            var exeres, _ = value.(*exec_result)
            if exeres == nil {
                debug(ctx, "not exec result: %v ", tv(value), trace{})
            }

            var num = __int(ctx, p.val)
            if ctx.verbose {
                prompt(ctx, "checking status ")
                if num != 0 { prompt(ctx, "== %d ", num) }
                prompt(ctx, "…")
            }

            var good = exeres.Status == int(num)
            if ctx.verbose {
                var s string
                if good { s = "yes" } else { s = "no" }
                prompt(ctx, "… %s (%d)\n", s, exeres.Status)
            }

            if ctx.debug > 0 {
                var tar = auto_get(ctx, "@")
                var val = auto_get(ctx, "-")
                debug(ctx,
					_f("%v: %v", _entry(ctx), tar),
					_f("hyphen=%v", val),
					_f("status=%v", exeres.Status))
            }

            if makeResult != nil {
                values = append(values, makeResult(good))
            } else if !good {
                debug(ctx, "bad status (%v) (expects %v)", exeres.Status, p.val, trace{})
                break argsloop
            }
        case "stdout", "stderr":
            var exeres, _ = value.(*exec_result)
            if exeres == nil {
                debug(ctx, "value '%v' (%T) is not exec result", value, value, trace{})
            } else { /*exeres.wg.Wait()*/ }

            if ctx.verbose {
                prompt(ctx, "checking %s (status=%d) … ", key, exeres.Status)
            }

            if 0 < ctx.debug {
                var tar = auto_get(ctx, "@")
                var val = auto_get(ctx, "-")
                debug(ctx,
					_f("%v: %v", _entry(ctx), tar),
					_f("hyphen=%v", val),
					_f("status=%v", exeres.Status),
					callstack{num:ctx.debug})
            }

            var v *bytes.Buffer
            switch key {
            case "stdout": v = exeres.Stdout.Buf
            case "stderr": v = exeres.Stderr.Buf
            default: unreachable()
            }

            if v == nil {
                debug(ctx, "bad %s (expects %v)", key, p.val, trace{})
                break argsloop
            }

            str = __string(ctx, p.val)
            if ctx.trim { str = strings.TrimSpace(str) }

            if res := v.String() == str; makeResult != nil {
                values = append(values, makeResult(res))
            } else if !res {
                debug(ctx, "bad %s (%v) (expects %v)", key, v, p.val, trace{})
                break argsloop
            }
        case "file", "dir": // file=xxx and dir=xxx, same as -file=xxx and -dir=xxx
            var ( f *file; res bool )
            if f, res = to_file(p.val); res {
                // ok
            } else if str = __string(ctx, p.val); filepath.IsAbs(str) {
                if f = _stat(ctx, str); f != nil {
                    // ok
                }
            } else if f = findfile(ctx, str); f != nil {
                // ok
            }
            switch key {
            case "file": res = f.info != nil && !f.info.Mode().IsDir()//.IsRegular()
            case "dir":  res = f.info != nil &&  f.info.Mode().IsDir()
            default: unreachable()
            }
            if makeResult != nil {
                values = append(values, makeResult(res))
            } else if !res {
                debug(ctx, "`%v` is not %s", p.val, key, trace{})
                break argsloop
            }
        case "var":
            var g, ok = p.val.(*group)
            if !ok {
                debug(ctx, "`%v` is not a group value", p.val, trace{})
                break argsloop
            }
            for _, elem := range g.elems {
                switch p := elem.(type) {
                case *pair:
                    var a, b string
                    var k = __string(ctx, p.key)
                    var def = program.project.finddef(k)
                    if def != nil {
                        a = __string(ctx, p.val)
                        b = __string(ctx, def.value)
                        if res := a != b; makeResult != nil {
                            values = append(values, makeResult(res))
                        } else if !res {
                            debug(ctx, "`%v` != `%v`", p.key, p.val, trace{})
                            break argsloop
                        }
                    } else if makeResult != nil {
                        values = append(values, makeResult(false))
                    } else {
                        debug(ctx, "`%v` is not defined", k, trace{})
                        break argsloop
                    }
                default:
                    debug(ctx, "`%v` unsupported checks", elem, trace{})
                    break argsloop
                }
            }
        default:
            debug(ctx, "unknown check for %v → %v", p.key, p.val, trace{})
            break argsloop
        }
    }
    return values
}

type copyopts struct {
    program *program
    path, update bool
    mode os.FileMode
    head Value
    foot Value
    files, copied int
    bytes int64
}

func copyRegular(ctx Context, src, dst string, opts *copyopts) (err error) {
    var def1 = auto_find(ctx, "1")
    var def2 = auto_find(ctx, "2")
    defer func(v1, v2 Value) { def1.value, def2.value = v1, v2 } (def1.value, def2.value)

    var pos = _position(ctx)
    def1.value = _strlit(pos, dst)
    def2.value = _strlit(pos, src)

    var head, foot string
    if opts.head != nil { head = __string(ctx, opts.head) }
    if opts.foot != nil { foot = __string(ctx, opts.foot) }

    // Compare mod time for update mode
    if opts.files += 1; opts.update {
        if st2, e := os.Stat(dst); e == nil && st2 != nil {
            var st1 os.FileInfo
            if st1, err = os.Stat(src); err != nil { debug(ctx, "%v", err); return }
            if st1 != nil && (st1.Size()+int64(len(head))+int64(len(foot))) == st2.Size() {
                if st2.ModTime().After(st1.ModTime()) { return }
            }
            if false { prompt(ctx, "%s: %s (%v,%v)\n", pos, dst, st1.Size(), st2.Size()) }
        }
    }

    var srcFile, dstFile *os.File
    if srcFile, err = os.Open(src); err != nil { debug(ctx, "%v", err); return } else {
        defer srcFile.Close()
    }

    // sys default file mode is 0666
    if opts.path { // Make path (mkdir -p)
        if p := filepath.Dir(dst); p != "." && p != "/" {
            err = os.MkdirAll(p, os.FileMode(0755))
            if err != nil { debug(ctx, "%v", err); return }
        }
    }

    if opts.mode == 0 { opts.mode = os.FileMode(0640) }

    dstFile, err = os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, opts.mode)
    if err != nil { debug(ctx, "%v", err); return } else { defer dstFile.Close() }

    srcBuf := bufio.NewReader(srcFile)
    dstBuf := bufio.NewWriter(dstFile)
    if head != "" {
        var n int
        if n, err = dstBuf.WriteString(head); err != nil { debug(ctx, "%v", err); return }
        opts.bytes += int64(n)
    }

    var n int64
    if n, err = io.Copy(dstBuf, srcBuf); err != nil { debug(ctx, "%v", err); } else {
        if opts.bytes += n; foot != "" {
            var n int
            if n, err = dstBuf.WriteString(foot); err != nil { debug(ctx, "%v", err); return }
            opts.bytes += int64(n)
        }
        if err == nil {
			if err = dstBuf.Flush(); err != nil {
                debug(ctx, "flush failed during copy: %v", err)
                return 
            }
            opts.copied += 1
        }
    }
    return
}

func copySymlink(ctx Context, src, dst string, opts *copyopts) (err error) {
    err = errors.New("copy symlink unimplemented")
    return
}

func copyDir(ctx Context, src, dst string, opts *copyopts) (err error) {
    if dst != "." && dst != "/" { // Make path (mkdir -p)
        err = os.MkdirAll(dst, os.FileMode(0755))
        if err != nil { return }
    }

    var fis []os.FileInfo
    if fis, err = ioutil.ReadDir(src); err != nil {
        return
    }
    for _, fi := range fis {
        ss := filepath.Join(src, fi.Name())
        sd := filepath.Join(dst, fi.Name())
        err = copyFile(ctx, fi, ss, sd, opts)
        if err != nil { break }
    }
    return
}

func copyFile(ctx Context, srcFi os.FileInfo, src, dst string, opts *copyopts) (err error) {
    if m := srcFi.Mode(); m&os.ModeSymlink != 0 {
        if opts.mode == 0 { opts.mode = srcFi.Mode() }
        err = copySymlink(ctx, src, dst, opts)
    } else if srcFi.IsDir() {
        err = copyDir(ctx, src, dst, opts)
    } else if m.IsRegular() {
        if opts.mode == 0 { opts.mode = srcFi.Mode() }
        err = copyRegular(ctx, src, dst, opts)
    } else {
        err = fmt.Errorf("copying non-regular files/dirs (%s)", src)
    }
    return
}

// (copy-file -p)
// (copy-file -p,filename)
// (copy-file -p,filename,source)
type modifier_copyfile struct { modifier_
    path bool "p,path"
    recursive bool "r,recursive"
    override bool "o,override"
    update bool "u,update"
    quick bool "q,quick"
    mode os.FileMode "m,mode"
    head Value "h,head"
    foot Value "f,foot"
}
func (ctx *modifier_copyfile) x(args ...Value) (result any) {
    var target Value
    var source Value
    if len(args) > 0 {
        target = args[0]
    } else {
        target = auto_get(ctx, "@")
    }
    if len(args) > 1 {
        source = args[1]
    } else {
        source = auto_get(ctx, "<")
    }

    // Get target filename
    var (
        project = _project(ctx)
        filename, srcname string
        filetime, srctime time.Time
    )
    switch tv := target.(type) {
    case *file:
        if filename = tv.fullname(); tv.info != nil {
            filetime = tv.info.ModTime()
        }
    default:
        filename = __string(ctx, target)
        if file := project.file(ctx, filename); file != nil {
            target, filename = file, file.fullname()
            if file.info != nil {
                filetime = file.info.ModTime()
            }
        }
    }
    switch tv := source.(type) {
    case *file:
        if srcname = tv.fullname(); tv.info != nil {
            srctime = tv.info.ModTime()
        }
    default:
        srcname = __string(ctx, source)
        if file := project.file(ctx, srcname); file != nil {
            source, srcname = file, file.fullname()
            if file.info != nil { srctime = file.info.ModTime() }
        }
    }

    if !filetime.IsZero() && filetime.After(srctime) {
        if ctx.update {
            if ctx.verbose { prompt(ctx, "update %v …", target) }
        } else if ctx.override {
            if ctx.verbose { prompt(ctx, "override %v …", target) }
        } else {
            if ctx.verbose { prompt(ctx, "copy %v …… already existed!\n", target) }
            if !ctx.silent { debug(ctx, "file already existed (%s)", target, trace{}) }
            return
        }
    } else if ctx.verbose {
        if ctx.update {
            prompt(ctx, "Checking %v …", target)
        } else {
            prompt(ctx, "Copy %v …", target)
        }
    }

    if ctx.quick {
        var file = _stat(ctx,filename, stat_nonexist{true})
        if file == nil || file.info != nil {
            if ctx.verbose { prompt(ctx, "… Good\n") }
            return
        }
    }

    var program = _program(ctx)
    var copts = &copyopts{
        program, ctx.path||ctx.recursive,
        ctx.update, ctx.mode, ctx.head, ctx.foot,
        0, 0, 0,
    }
    var file *file
    if file = _stat(ctx,srcname, stat_nonexist{true}); file == nil || file.info == nil {
        debug(ctx, "'%s' source file not found", srcname, trace{})
    } else if !file.info.IsDir() {
        if ctx.mode == 0 { ctx.mode = file.info.Mode() }
        if err := copyFile(ctx, file.info, srcname, filename, copts); err != nil {
            debug(ctx, "%v", err, trace{})
        }
    } else if ctx.recursive {
        if err := copyDir(ctx, srcname, filename, copts); err != nil {
            debug(ctx, "%v", err, trace{})
        }
    } else {
        debug(ctx, "`%v` is a directory (use -r to solve it)", source, trace{})
    }

    if ctx.verbose {
        if copts.copied == 0 {
            prompt(ctx, "… Good (%d files)\n", copts.files)
        } else if copts.copied == 1 {
            prompt(ctx, "… Copied %d bytes\n", copts.bytes)
        } else {
            prompt(ctx, "… Copied %d bytes (%d/%d)\n", copts.bytes, copts.copied, copts.files)
        }
    }
    return
}

type modifier_writefile struct { modifier_ }
func (ctx *modifier_writefile) x(args ...Value) (result any) {
    args = xmerge(ctx, args...) //, plain

    var (
        target = auto_get(ctx, "@")
        filename, str string
        f *os.File
    )
    if target == nil {
        debug(ctx, "target is undefined", trace{})
    }

    defer func() {
        if filename != "" { os.Remove(filename); f = nil }
        if f == nil {
            debug(ctx, "file %s not generated", target, trace{})
        }
    } ()

    filename, _ = as{target}.fullname_string(ctx)

    if h := auto_get(ctx, "-"); h == nil {
        debug(ctx, "buffer value is nil", trace{})
    } else {
        str = __string(ctx, h)
    }

    var err error
    if f, err = os.Create(filename); err != nil {
        debug(ctx, "%v", err, trace{})
    } else if _, err = f.WriteString(str); err != nil {
        f.Close()
        debug(ctx, "%v", err, trace{})
    } else {
        result = _stat(ctx, filename)
        f.Close()
    }
    return
}

type modifier_readfile struct { modifier_
    head Value "h,head"
    foot Value "f,foot"
}
func (ctx *modifier_readfile) x(args ...Value) (result any) {
    var (
        filename string
        file *file
        target as
    )
    if n := len(args); n > 1 {
        debug(ctx, "too many files: %v", args, trace{})
    } else if n == 1 {
        target.Value = args[0]
    } else {
        target.Value = auto_get(ctx, "@")
    }

    if isTrivial(target) {
        debug(ctx, "target for reading is invalid (%T) (%v)", target.Value, args, trace{})
    } else if file, filename, _ = target.fullname_file(ctx); file == nil {
        if val := auto_get(ctx, ">"); val != nil {
            panic(traveTargetNotDefinedFile)
        } else if true {
            debug(ctx, "not a file: %v (%T)", target.Value, target.Value)
            debug(ctx, trace{})
        }
        return
    } else if filename == "" {
        debug(ctx, "%v: empty fullname", target, trace{})
    }

    var ( bytes []byte ; err error )
    if bytes, err = ioutil.ReadFile(filename); err == nil {
		var b strings.Builder
        
        // Pre-grow the buffer to exactly the size we need to prevent re-allocations
        headStr, footStr := "", ""
        if ctx.head != nil { headStr = __string(ctx, ctx.head) }
        if ctx.foot != nil { footStr = __string(ctx, ctx.foot) }

        b.Grow(len(headStr) + len(bytes) + len(footStr))
        b.WriteString(headStr)
        b.Write(bytes) // Writes the byte slice directly, no string cast needed!
        b.WriteString(footStr)
		
        auto_set(ctx.Context, defVoid, "-", _strlit(_position(ctx), b.String()))
        auto_set(ctx.Context, defVoid, "-file", file)
    } else {
        debug(ctx, "%v", err, trace{})
    }
    if 0 < ctx.debug && err != nil {
        debug(ctx, "%v: %v ; stems=%v\n", target, err, _stems(ctx))
    }
    return
}

func crc64CheckFileModeContent(ctx Context, filename string, content []byte, perm os.FileMode) (same bool, err error) {
    var f *os.File
    if f, err = os.Open(filename); err == nil && f != nil {
        defer f.Close()

        var s os.FileInfo
        if s, err = f.Stat(); err != nil { return false, err }

        // Fast Path: If sizes differ, they cannot be the same. Skip hashing!
        if s.Size() != int64(len(content)) {
            return false, nil
        }

        if perm != 0 && s.Mode().Perm() != perm {
            if err = f.Chmod(perm); err != nil { return }
        }

        w1 := crc64.New(crc64Table)
        w2 := crc64.New(crc64Table)
        if _, err = io.Copy(w1, f); err != nil { return }
        if _, err = w2.Write(content); err != nil { return }
        if w1.Sum64() == w2.Sum64() { same = true }
    }
    return
}

func crc64CompareFileChecksum(ctx Context, filename1, filename2 string) (same bool, err error) {
    var s []byte
    if s, err = ioutil.ReadFile(filename1); err != nil {
        debug(ctx, "%v", err, trace{})
        return
    }
    return crc64CheckFileModeContent(ctx, filename2, s, 0)
}

type modifier_updatefile struct { modifier_
    verbFilename bool `verbfile,verb-filename`
    path   bool `p,path,makedir,make-dir,makepath,make-path`
    zero   bool `zero,empty,allow-zero,allow-empty`
    keep   bool `keep,keep-file`
    append bool `app,append,append-content`
    mode os.FileMode "mode"
}
func (ctx *modifier_updatefile) x(args ...Value) (result any) {
    assert(ctx.mode != 0, "zero file mode")

    var target as
    var content string
    var filename string
    if len(args) > 0 { target.Value = args[0] }

    if isTrivial(target.Value) { target.Value = auto_get(ctx, "@") }
    if isTrivial(target.Value) {
        debug(ctx, "update-file: no file target", trace{})
    } else if t := target.fullname(ctx); t.Value == nil {
        debug(ctx, "update-file: not a file: %v", ts(target.Value), trace{})
    } else if filename = __string(ctx, t); filename == "" {
        debug(ctx, "update-file: empty fullname: %v", ts(target.Value), trace{})
    }

    if checkpoints {
        defer func() {
            ctx.x_check(target.Value, filename, content, args, result)
        } ()
    }

    if ctx.path { // Make path (mkdir -p)
        if p := filepath.Dir(filename); p != "." && p != "/" {
            if fi, _ := os.Stat(p); fi != nil && !fi.IsDir() {
                if e := os.Remove(p); e != nil {
                    debug(ctx, "%v (%v)", e, ts(target.Value), trace{})
                }
            }
            if e := os.MkdirAll(p, os.FileMode(0755)); e != nil {
                if proj := _project(ctx); proj != nil {
                    info(ctx, "%v: %v %v", filename, proj, unmap_files(ctx, proj, _pathStr(ctx, filename), nil))
                    info(ctx, "%v: %v %v", filename, proj, proj.file(ctx, filename))
                    debug(ctx, "%v: %v (%v)", filename, e, tv(target), trace{})
                }
                return
            }
        }
    }

    // Check existed file content checksum
    var exeres *exec_result
    if val := auto_get(ctx, "-"); val == nil {
        // no buffer value
    } else if content = __string(ctx, val); false && strings.Contains(content, `"\"`) {
        prompt(ctx, "%v: %T\n", filename, val)
        panic(_failure(ctx, "%s", filename))
    } else {
        exeres, _ = val.(*exec_result)
    }

    if content != "" {
        // good to go
    } else if ctx.zero {
        if ctx.verbose || ctx.debug > 0 {
            debug(ctx, "empty content for '%v'", target, callstack{num:ctx.debug})
        }
    } else {
        if ctx.keep {
            // keep file
        } else if file := _stat(ctx, filename); file != nil && file.info != nil && file.info.Size() == 0 {
            file.info = nil
            if err := os.Remove(filename); err != nil {
                debug(ctx, "remove file failed: %v", err, trace{})
            }
        }
        if exeres != nil {
            if exeres.Stdout.log != nil {
                var pos Position
                pos.Filename = exeres.Stdout.log.filename
                pos.Line = exeres.Stdout.log.lines + 1
                debug(ctx, "empty stdout")
            }
            if exeres.Stderr.log != nil && exeres.Stdout.log != exeres.Stderr.log {
                var pos Position
                pos.Filename = exeres.Stderr.log.filename
                pos.Line = exeres.Stderr.log.lines + 1
                debug(ctx, "empty stderr")
            }
        }

        if v := auto_get(ctx, "-"); v == nil {
            prompt(ctx, "%s:1: empty content\n", filename)
        } else {
            prompt(ctx, "%s:1: empty content: %v\n", filename, v)
        }
        debug(ctx, "empty content for '%v'", target, trace{})
    }

    var (
        wrote int
        same bool
        err error
    )
    if ctx.verbose {
        defer func(st time.Time) {
            var f string
            if ctx.verbFilename {
                f = trimPromptString(filename)
            } else {
                f = trimPromptString(target.String())
            }

            var s string
            if err != nil { s = err.Error() } else if same {
                if true { return } else { s = "unchanged" }
            } else if ctx.debug > 0 {
                s = fmt.Sprintf("changed (%d bytes, %s)", wrote, filename)
            } else {
                s = fmt.Sprintf("changed (%d bytes)", wrote)
            }
            prompt(ctx, "update %v …… %s (in %v)\n", f, s, time.Now().Sub(st))
        } (time.Now())
    }

    if same, err = crc64CheckFileModeContent(ctx, filename, []byte(content), ctx.mode); err != nil {
        if _, ok := err.(*os.PathError); ok {
            err = nil // discard path error (e.g. no such file or directory)
        } else {
            debug(ctx, "crc64 checksum failed: %v", err, trace{})
        }
    } else if same {
        //removeCallerUpdated(ctx, target) // remove timestamp updated
        result = _stat(ctx, filename)
        return
    }

    // Create or update the file with new content

    var f *os.File
    var m = os.O_RDWR | os.O_CREATE
    if ctx.append { m |= os.O_APPEND } else { m |= os.O_TRUNC }
    if f, err = os.OpenFile(filename, m, ctx.mode); err != nil {
        debug(ctx, "open file failed: %v", err, trace{})
    } else if f != nil {
        defer func() {
            if err = f.Close(); err != nil {
                os.Remove(filename)
                debug(ctx, "close file '%s' failed: %v", filename, err, trace{})
            }

            if t := _stat(ctx, filename); t == nil {
                prompt(ctx, "%s: invalid file\n", filename)
                debug(ctx, "%v: invalid file '%s'", _project(ctx), filename, trace{})
            } else {
                var fs = t.stamp(must_files_stamp{ctx})
                if false && ctx.verbose { reportFileUpdates(ctx, fs) }
                result = t // resulting the updated file
            }
        } ()
        if wrote, err = f.WriteString(content); err != nil {
            debug(ctx, "write content failed: %v", err, trace{})
        }
    } else {
        debug(ctx, "%v not updated", target, trace{})
    }
    return
}

type modifier_wait struct { modifier_
    stdout   bool "o,stdout"
    stderr   bool "e,stderr"
    status   bool "s,status"
    trim     bool "t,trim" // trim heading and tailing spaces of the result
    execRes  bool "x,exec"
    noTarget bool `nt,no-target`
    asType string "a,as"
}
func (ctx *modifier_wait) x(args ...Value) (result any) {
    var (
        waitForExecResult = ctx.stdout || ctx.stderr || ctx.status || ctx.execRes
        stampCurrentTarget = !ctx.noTarget
        target Value = auto_get(ctx, "@")
        execRes *exec_result
        err error
    )
    if ctx.verbose {
        defer func (st time.Time) {
            var s string; if err != nil { s = "fail" } else { s = "done" }
            prompt(ctx, "Wait %v …… %s, result=%v\n", target, s, execRes)
            if ctx.debug>0 { debug(ctx, "%v", execRes) }
        } (time.Now())
    }

    // Wait for prerequisites and/or execution
    _, _, execRes = wait(ctx, waitopts{ctx.verbose, waitForExecResult, stampCurrentTarget})
    if execRes == nil { return }

    var (
        pos = _position(ctx)
        a []Value
        s string
        v Value
    )
    if ctx.stdout {
        // TODO: warn(ctx, "deprecated (wait -stdout), use (shell -stdout) instead; %v", execRes).debug()
        if b := execRes.Stdout.Buf; b != nil { s = b.String() }
        if ctx.trim { s = strings.TrimSpace(s) }
        switch ctx.asType {
        case "answer": v = _answer (pos,(s == "yes"))
        case "bool":   v = _boolean(pos,(s == "true"))
        default:       v = _strlit(pos,s)
        }
        a = append(a, v)
    }
    if ctx.stderr {
        // TODO: warn(ctx, "deprecated (wait -stderr), use (shell -stderr) instead; %v", execRes).debug()
        if b := execRes.Stderr.Buf; b != nil { s = b.String() }
        if ctx.trim { s = strings.TrimSpace(s) }
        switch ctx.asType {
        case "answer": v = _answer (pos,(s == "yes"))
        case "bool":   v = _boolean(pos,(s == "true"))
        default:       v = _strlit(pos,s)
        }
        a = append(a, v)
    }
    if ctx.status {
        // TODO: warn(ctx, "deprecated (wait -status), use (shell -status) instead; %v", execRes).debug()
        a = append(a, _decimal(pos,int64(execRes.Status)))
    }

    if len(a) > 0 { result = ease(ctx, a) }
    return
}

func reportFileUpdates(ctx Context, fs []*file) {
    var start = _execution(ctx).start
    for _, f := range fs {
        var mod = f.info.ModTime()
        var d = time.Now().Sub(start)
        if mod.After(start) {
            prompt(ctx, "Updated %v (%v)\n", f.name, d)
        } else {
            prompt(ctx, "File %v not changed (%v, ModTime=%v)\n", f, d, mod)
            warn(ctx, "incorrect timestamp: %v (JobTime=%v, ModTime=%v)", f, start, mod)
            warn(ctx, "the target path name is: %v", f.fullname())
            warn(ctx, "try 'touch' the target %v if the path name and command are correct", f)
            info(ctx, "you may ignore the warnings if all correct")
        }
    }
}

type modifier_stamp struct { modifier_
    prompt bool "prompt"
    next   bool "nxt,next"  // traveNext if failed to stamp
    error  bool "err,error" // traveErro if failed to stamp
}
func (ctx *modifier_stamp) x(args ...Value) (result any) {
    var target = auto_target_value(ctx)

    if isNull(target) {
        prompt(ctx, "%v\n", _project(ctx))
        debug(ctx, "stamp(%v) failed", target, trace{})
    }

    var v = stampFile(must_files_stamp{ctx}, target)
    if v != nil { return /* Done! */ }

    prompt(ctx, "%v: %v\n", target, _project(ctx))
    if ctx.next {
        panic(traverse_state{_position(ctx),traverse_next})
    } else if ctx.error {
        debug(ctx, "stamp(%v) error", trace{})
    } else {
        if f, y := target.(*file); y {
            debug(ctx, "failed stamp(%v): %v %v", target, f.fullname(), f.info, trace{})
        } else {
            debug(ctx, "failed stamp(%v) (%T)", target, target, trace{})
        }
    }
    return
}

type modifier_assert struct { modifier_
    msg string `msg,message`
}
func (ctx *modifier_assert) v(args ...Value) (_ any) { ctx.z(args...) ; return }
func (ctx *modifier_assert) x(args ...Value) (_ any) { ctx.z(args...) ; return }
func (ctx *modifier_assert) z(args ...Value) (_ any) {
    var u = _universe(ctx)
    for _, a := range args {
        if a == nil {
            debug(ctx, "assert: nil", trace{})
        }

        if _, y := a.(*punct); y { continue }

        v := expand(_final(ctx),a)
        b := v != nil && __true(ctx, v)
        f := u.hooks.assert

        if (f != nil && f(ctx, v, b)) || b {
            continue
        } else if ctx.msg == "" {
            var s string
            if v != nil { s = __string(ctx, v) }
            debug(pc(ctx,a), "assert: %v → %v → '%s'", a, v, s, trace{})
        } else {
            debug(pc(ctx,a), "assert: %v → %v: %s", a, v, ctx.msg, trace{})
        }
    }
    return
}

type modifier_cond struct { modifier_ }
func (ctx *modifier_cond) x(args ...Value) (result any) {
    // TODO: make it lisp-like (cond), e.g.:
    //     (cond
    //       ((condition) ...)
    //       (true{} ...))
    for _, a := range args {
        if a == nil { debug(ctx, "nil arg") }
        if a == nil || !__true(ctx.Context, a) {
            panic(traverse_state{_position(ctx),traverse_done})
        }
    }
    return _boolean(_position(ctx), true)
}

type modifier_case struct { modifier_ }
func (ctx *modifier_case) x(args ...Value) (result any) {
    for _, a := range args {
        if __true(ctx.Context, a) {
            panic(traverse_state{_position(ctx),traverse_case})
        }
    }

    if ctx.verbose { prompt(ctx, "%v", auto_get(ctx, "@")) }
    panic(traverse_state{_position(ctx),traverse_next})
    return
}

type modifier_predictDirty struct { modifier_ }
func (ctx *modifier_predictDirty) x(args ...Value) (result any) {
    if res := _execution(ctx).dirty(ctx, args...); res {
        return makePrediction(_position(ctx), res, "")
    } else {
        panic(traverse_state{_position(ctx),traverse_done})
    }
}

type modifier_fork struct { modifier_
    wd string `workdir,work-dir`
}
func (ctx *modifier_fork) _x(args ...Value) (result Value) {
    var (
        attr syscall.ProcAttr
        argv []string
        prog = _program(ctx)
    )
    for _, a := range args { argv = append(argv, __string(ctx, a)) }

    if ctx.wd != "" {
        attr.Dir = ctx.wd
    } else if attr.Dir = prog.workdir(ctx); attr.Dir == "" {
        debug(ctx, "empty workdir", trace{})
    }

    attr.Env, _ = _execution(ctx).env(ctx)
    attr.Files = []uintptr{ // FIXME: see Cmd.Start() for files pipes
        os.Stdin .Fd(),
        os.Stdout.Fd(),
        os.Stderr.Fd(),
    }

    if exe, err := os.Executable(); err != nil {
        debug(ctx, "fork: %v: %v", os.Args[0], err, trace{})
    } else if pid, err := syscall.ForkExec(exe, argv, &attr); err != nil {
        debug(ctx, "fork: %v: %v", exe, err, trace{})
    } else if pid == 0 {
        debug(ctx, "fork: pid is zero", trace{})
    } else {
        // TODO: status code, etc.
    }
    return
}
func (ctx *modifier_fork) x(args ...Value) (result any) {
    var (
        prog = _program(ctx)
        argv []string
        wd string
    )
    for _, a := range args { argv = append(argv, __string(ctx, a)) }

    if ctx.wd != "" {
        wd = ctx.wd
    } else if wd = prog.workdir(ctx); wd == "" {
        debug(ctx, "empty workdir", trace{})
    }

    var exe, err = os.Executable()
    if err != nil {
        debug(ctx, "fork: %v: %v", os.Args[0], err, trace{})
    }

    var cmd = exec.Command(exe, argv...)
    cmd.Stdout, cmd.Stderr = stdout, stderr
    cmd.Env, _ = _execution(ctx).env(ctx)

    if err = cmd.Run(); err != nil {
        debug(ctx, "fork: %v: %v", exe, err, trace{})
    } else {
        // TODO: status code, etc.
    }
    return
}

type modifier_gitmodified struct { modifier_ }
func (ctx *modifier_gitmodified) x(args ...Value) (result any) {
    var out = new(bytes.Buffer)
    var git = exec.Command("git", "status")
    git.Stdout, git.Stderr = out, os.Stderr
    if err := git.Run(); err != nil {
        debug(ctx, "git failed: %v", err, trace{})
    }

    // TODO: check also for `Changes not staged for commit:`

    var rx = regexp.MustCompile(`\n\tmodified:[\ctx ]*(.+?)\n`)
    var sm = rx.FindAllSubmatch(out.Bytes(), -1)
    if len(sm) > 0 {
        var pos = _position(ctx)
        var pred = makePrediction(pos, false, "")
        if result = pred; len(args) == 0 {
            pred.bool, pred.s = true, "modified"
            return
        }
        for _, a := range args {
            var s = __string(ctx, a)
            for _, v := range sm {
                if false { prompt(ctx, "%s: %s\n%v\n", pos, s, v[1]) }
                if s == string(v[1]) {
                    pred.bool, pred.s = true, "modified: "+s
                    return
                }
            }
        }
    }
    return
}

type modifier_gitahead struct { modifier_ }
func (ctx *modifier_gitahead) x(args ...Value) (result any) {
    var out = new(bytes.Buffer)
    var git = exec.Command("git", "status")
    git.Stdout, git.Stderr = out, os.Stderr
    if err := git.Run(); err != nil {
        debug(ctx, "git: %v", err, trace{})
    }

    // TODO: check also for `Changes not staged for commit:`

    var rx = regexp.MustCompile(`\nYour branch is ahead of '(.+?)' by`)
    var sm = rx.FindAllSubmatch(out.Bytes(), 1)
    if len(sm) > 0 {
        result = makePrediction(_position(ctx), true, "Work branch has new commits to push")
    }
    return
}

var (
    onceMutex sync.Mutex
    onceCache0 map[entry]map[Value]int
    onceCache1 map[*program]map[Value]int
    onceSHA256Mutex sync.Mutex
    onceSHA256Cache = make(map[hashbytes]int,64)
)

func onceCacheTest0(ctx Context, target Value) (n int) {
    var rec map[Value]int
    var ent = _entry(ctx)
    if x, y := ent.(*stemmed_rule); y { ent = x.rule }

    onceMutex.Lock(); defer onceMutex.Unlock()
    if onceCache0 == nil { onceCache0 = make(map[entry]map[Value]int, 64) }
    if rec, _ = onceCache0[ent]; rec == nil {
        rec = make(map[Value]int)
        onceCache0[ent] = rec
    }

    rec[target] += 1
    n = rec[target]
    return
}

func onceCacheTest1(ctx Context, target Value) (n int) {
    var (
        prog = _program(ctx)
        rec map[Value]int
    )

    onceMutex.Lock(); defer onceMutex.Unlock()
    if onceCache1 == nil { onceCache1 = make(map[*program]map[Value]int,64) }
    if rec, _ = onceCache1[prog]; rec == nil { rec = make(map[Value]int)
        onceCache1[prog] = rec
    }

    rec[target] += 1
    n = rec[target]
    return
}

func onceCacheTest2(ctx Context, target Value) (n int) {
    var (
        program = _program(ctx)
        h = sha256.New()
        entry = _entry(ctx)
    )
    if stemmed, ok := entry.(*stemmed_rule); ok {
        entry = stemmed.rule
    }

    // NOTE: ensure 'entry', 'program' and 'target' are unique.
    if true {
        fmt.Fprintf(h, "%p", program)
    } else if false {
        // // FIXME: not unique combination
        // fmt.Fprintf(h, "%p", entry)
        fmt.Fprintf(h, "%T%p", entry, entry)
    } else {
        // // FIXME: not unique combination
        // fmt.Fprintf(h, "%p%p", entry, program)
        fmt.Fprintf(h, "%T%p%p", entry, entry, program)
    }

    for _, t := range merge(target) {
        if f, ok := to_file(t); ok {
            fmt.Fprintf(h, "%s", f.fullname())
        } else {
            fmt.Fprintf(h, "%s", __string(ctx, t))
        }
    }

    var sum hashbytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

func onceSHA256Test(ctx Context, sum hashbytes) (n int) {
    onceSHA256Mutex.Lock()
    n = onceSHA256Cache[sum]+1
    onceSHA256Cache[sum] = n
    onceSHA256Mutex.Unlock()
    return
}

func onceSHA256(ctx *modifier_once, target Value, args ...Value) (n int) {
    var (
        program = _program(ctx)
        entry = _entry(ctx)
        h = sha256.New()
    )
    if stemmed, ok := entry.(*stemmed_rule); ok {
        entry = stemmed.rule
    }

    if true {
        // // NOTE: entry and program are unique, since (once) is for runtime, we use their addresses.
        // fmt.Fprintf(h, "%p%p", entry, program)
        fmt.Fprintf(h, "%T%p%p", entry, entry, program)
    } else {
        fmt.Fprintf(h, "%v%v", _position(ctx), program.position)
    }

    var a as
    for _, a.Value = range args {
        s, _ := a.fullname_string(ctx)
        fmt.Fprintf(h, "%s", s)
    }

    var sum hashbytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

type modifier_once struct { modifier_
    checksum bool `cs,checksum,sha,sha256,sum,hash`
    forval Value `for` // TODO: (once -for=$@)
}
func (ctx *modifier_once) x(args ...Value) (result any) {
    // TODO: (once)           --> once for the Rule, aka entry.doneOnce = true
    // TODO: (once -for=$@)   --> once for $@, aka entry.onces[$(expand $@)] = true
    var target Value = auto_get(ctx, "@")

    const onceAlgo = 2 // avaialbe: 0, 1, 2

    if isTrivial(target) {
        debug(ctx, "once: no target $@, %v", args, trace{})
    } else if ctx.checksum {
        onceSHA256(ctx, target, append([]Value{target}, args...)...)
    } else if onceAlgo == 2 {
        onceCacheTest2(ctx, target)
    } else if onceAlgo == 1 {
        onceCacheTest1(ctx, target)
    } else {
        onceCacheTest0(ctx, target)
    }
    return
}
