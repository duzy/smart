//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
    // "hash/maphash"
    "io"
    "io/fs"
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
type generalOpts struct {
    debug    int  `d,db,dbg,debug` // NOTE: compatible with 'bool'
    stack    int  `stack,stacknum,stack-num,stack-number`
    fail     bool `fail` // fail on errors
    force    bool `force,forth` // force expand
    fullname bool `full,fullname,full-name,fullfile,full-file`
    silent   bool `silent` // force silent, contrast 'verbose'
    timing   bool `time,timing`
    trace    bool `trace`
    verbose  bool `verb,verbose` // prompts more information
    warn     bool `warn,warning` // prompts more warnings
}

type modifier_ struct { Context ; generalOpts }
type modifier_v interface{ v(...Value) interface{} }
type modifier_x interface{ x(...Value) interface{} }
type modifier_y interface{ x(*programContext, ...Value) interface{} }

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
        `stringify`:       reflect.TypeOf((*modifier_stringify)(nil)).Elem(),
        `reveal`:       reflect.TypeOf((*modifier_reveal)(nil)).Elem(),
        `disclose`:     reflect.TypeOf((*modifier_disclose)(nil)).Elem(),

        `select`:       reflect.TypeOf((*modifier_select)(nil)).Elem(),

        `env`:          reflect.TypeOf((*modifier_env)(nil)).Elem(), // interpreter environments
        `var`:          reflect.TypeOf((*modifier_var)(nil)).Elem(),
        `set`:          reflect.TypeOf((*modifier_set)(nil)).Elem(),
        `defer`:        reflect.TypeOf((*modifier_defer)(nil)).Elem(),

        `closure`:      reflect.TypeOf((*modifier_closure)(nil)).Elem(),
        `for`:          reflect.TypeOf((*modifier_for)(nil)).Elem(),

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
        `configure`:       reflect.TypeOf((*modifier_configure)(nil)).Elem(),

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
        // `dirty-by`:     reflect.TypeOf((*modifier_setDirtyPats)(nil)).Elem(),
        // `dirty-opts`:   reflect.TypeOf((*modifier_setDirtyPats)(nil)).Elem(),
        // `outdated`:         reflect.TypeOf((*modifier_predictDirty)(nil)).Elem(),
        // `no-loop`:          reflect.TypeOf((*modifier_predictNoLoop)(nil)).Elem(),
        // `target-1st-visit`: reflect.TypeOf((*modifier_predictTarget1stVisit)(nil)).Elem(),
        // `target-max-visit`: reflect.TypeOf((*modifier_predictTargetMaxVisit)(nil)).Elem(),
    }

    crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)

func modify(x Context, g *group, hyphen bool) (res Value) {
    var uni = _universe(x)
    var pc = cast[*programContext](x)
    var ctx = at(x, g.position)
    var name = g.elems[0].string(ctx)
    var args = g.elems[1:]

	defer trace(ctx)

    if t, y := modifiers[name]; !y {
        var _, ent, _ = entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s failed for %s\n", ent, name, ctx.project())
        erro(ctx, "unknown modifier: %s (args=%v)", name, args)
        errostack(ctx, 5, "%v", ctx).debug(10)
        return
    } else {
        mv := reflect.New(t)
        mi := mv.Interface()

        var fv modifier_v
        var fx modifier_x
        var fy modifier_y
        if !hyphen {
            if fv, _ = mi.(modifier_v); fv == nil {
                errostack(ctx, 3, "%v: no method: (*%s).v(...)", name, typeof(mi)).debug(16)
                return
            }
        } else if fx, _ = mi.(modifier_x); fx == nil {
            if fy, _ = mi.(modifier_y); fy == nil {
                errostack(ctx, 3, "%v: no method: (*%s).x(...)", name, typeof(mi)).debug(16)
                return
            } else if pc == nil {
                errostack(ctx, 3, "%v: nil pc for: (*%s).x(...)", name, typeof(mi)).debug(16)
                return
            }
        }

        if c := mv.Elem().FieldByName("Context"); c.IsValid() {
            c.Set(reflect.ValueOf(ctx)) // c.Type().String() == "smart.Context"
        } else {
            errostack(ctx, 3, "%v: no field: %s.Context", name, typeof(mi)).debug(16)
            return
        }

        args = _opts(ctx, mv, args)
        if false && name != "defer" { for i, a := range args { if g, y := a.(*group); y {
            args[i] = modify(ctx, g, hyphen)
        }}}

        if fv != nil { res = ease(ctx, fv.v(args...)) }
        if fx != nil { res = ease(ctx, fx.x(args...)) } else
        if fy != nil { res = ease(ctx, fy.x(pc, args...)) }
    }

    if pc != nil && pc.traves.has(traveFail) {
        if true && res != nil { warn(ctx, "%v %v", res, pc.traves).debug(1) }
        if t := pc.traves.not(traveCase, traveNext, traveDone); false && t.has() {
            if uni.verbose || uni.verboseBreaks {
                proj := ctx.project()
                var _, ent, _ = entryIndicator(ctx, _entry(ctx))
                prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                for _, s := range t { warn(at(ctx,s.pos), "%v: %s: %v", proj, name, s) }
                warnstack(ctx, 5, "").debug(16)
            }
        }
    } else if !hyphen {
        // $- remains
    } else if res == nil {
        res = makeNull(g.position) // $- remains too
    } else if name == "defer" || name == "set" || name == "var" {
        errostack(ctx, 3, "invalid result: (set ...) ⇒ %T %v", res, res).debug(1)
    } else if a := _automatic(ctx); a != nil {
        a.amend(ctx, "-", res)
    }

    if pc != nil { if n := flush(ctx); n > 0 {
        pc.traves.add(ctx, traveFail, nil).
            error = fmt.Errorf("%s: %d errors counted", name, n)
    }}
    return
}

type modifier struct { group }
func (m *modifier) kind() Kind { return m.group.kind()|KindModifier }
func (m *modifier) cmp(ctx Context, v Value) (res cmpres) {
    if o, y := v.(*modifier); y { res = m.group.cmp(ctx, &o.group) }
    return
}
func (m *modifier) expandable(ctx Context) (res bool) { return m.group.expandable(ctx) }
func (m *modifier) expand(ctx Context) Value { return modify(ctx, &m.group, false) }
func (m *modifier) string(ctx Context) string { return m.expand(final{ctx}).string(ctx) }
func (m *modifier) traverse(ctx Context) { ctx = at(ctx, m.position)
    var name = m.elems[0].string(ctx)
    if name == "" {
        erro(at(ctx,m.elems[0]), "empty name: %v", m.elems[0]).debug(1)
        return
    }

    if false { defer func(t0 time.Time) { var n time.Duration = 1*time.Second
        switch name { case "shell", "sh": n = 60*time.Second }
        if d := time.Now().Sub(t0); d > n { note(ctx, "slow: %v ⇒ %v\n", m, d).debug(1) }
    }(time.Now())}

    var pc, prog = cast[*programContext](ctx), _program(ctx)
    if len(pc.interpreted) == 0 && len(prog.recipes) > 0 && name == "configure" {
        if i, y := dialects["eval"]; y && i != nil { pc.interpret(ctx, i, m.elems[1:]) }
    } else if i, y := dialects[name]; y && i != nil {
        pc.interpret(ctx, i, m.elems[1:])
        return
    }

    if v := modify(ctx, &m.group, true); v != nil {
        // TODO: deal with modify result `v`
    }
}
func (_ *modifier) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *modifier) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported").debug(32)
    return
}

type modification struct {
    valbase
    list []*modifier
}
func (_ *modification) kind() Kind { return KindModification }
func (g *modification) cmp(ctx Context, v Value) (res cmpres) {
    if o, y := v.(*modification); y && len(g.list) == len(o.list) {
        for i, m := range g.list {
            if t := m.cmp(ctx, o.list[i]); t != cmpEqual { return t }
        }
        res = cmpEqual
    }
    return
}
func (g *modification) refs(ctx Context, v Value) (res bool) {
    for _, m := range g.list {
        if res = m.refs(ctx, v); res { return }
    }
    return
}
func (g *modification) expandable(ctx Context) (res bool) {
    for _, m := range g.list {
        if res = m.expandable(ctx); res { return }
    }
    return
}
func (g *modification) expand(ctx Context) (res Value) {
    var vals []Value
    for _, m := range g.list {
        if v := m.expand(ctx); v != nil {
            vals = append(vals, v)
        }
    }
    return ease(ctx, vals)
}
func (g *modification) traverse(ctx Context) {
    var pc = cast[*programContext](ctx)
    if pc != nil { pc.Wait() }

    var u = _universe(ctx)
    for _, m := range g.list {
        var ctx = at(ctx, m.position)
        if m.traverse(ctx); pc == nil { continue }
        if t := pc.traves.of(traveFail); t.has() { return }
        if t := pc.traves.not(traveCase, traveDone, traveNext, traveRule, traveFile); t.has() {
            if true || (u.verbose || u.verboseBreaks) {
                var _, ent, _ = entryIndicator(ctx, _entry(ctx))
                warn(ctx, "%v: %s failed\n", ent, m.elems[0])
                for _, s := range t { warn(at(ctx,s.pos), "%v: %v", m.elems[0], s) }
                warnstack(ctx, 5, "").debug(16)
            }
            break
        }
        if t := pc.traves.of(traveCase); t.has() { continue }
        if t := pc.traves.of(traveDone, traveNext); t.has() { return }
    }
}
func (g *modification) String() (s string) {
    s = "{"
    for i, m := range g.list {
        if i > 0 { s += " " }
        s += m.String()
    }
    s += "}"
    return
}
func (g *modification) string(ctx Context) (s string) {
    for i, m := range g.list {
        if i > 0 { s += " " }
        s += m.string(ctx)
    }
    return
}
func (_ *modification) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *modification) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported").debug(32)
    return
}

func getGroupElem(value Value, n int, v Value) Value {
    if g, ok := value.(*group); ok {
        if elem := g.at(n); elem != nil {
            v = elem
        }
    }
    return v
}

func promptShellResult(ctx Context, value Value, n int) {
    if g, ok := value.(*group); ok && g != nil {
        if elem := g.at(0); elem != nil {
            if str := elem.string(ctx); str == "shell" {
                if elem = g.at(n); elem != nil {
                    if str = elem.string(ctx); strings.HasSuffix(str, "\n") {
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
    info []Value `i,info`
    warn []Value `w,warn`
    erro []Value `e,er,err,erro,error`
    checkOutdated bool `dirty,cd,checkdirty,check-dirty,co,check-outdated`
    trave int `tr,trave,traverse`
    s int `s,stack,sn,stack-number`
    n int `c,count,n,num,cn,call-number`
}
func (ctx *modifier_debug) x(args ...Value) (result interface{}) {
    if ctx.cond != nil && !ctx.cond.true(ctx) { return }
    if ctx.s == 0 && ctx.stack > 0 { ctx.s = ctx.stack }
    if ctx.n == 0 && ctx.debug > 0 { ctx.n = ctx.debug }
    for _, v := range ctx.info { info(at(ctx,v), "%s", v.string(ctx)).debug(1) }
    for _, v := range ctx.warn { warn(at(ctx,v), "%s", v.string(ctx)).debug(1) }
    for _, v := range ctx.erro { erro(at(ctx,v), "%s", v.string(ctx)).debug(1) }

    var (
        target  = autoVal(ctx, "@")
        depends = autoVal(ctx, "^")
    )
    if ctx.checkOutdated && target != nil {
        var (
            ordered = autoVal(ctx, "|")
            grepped = autoVal(ctx, "~")
            tt = target.stat(ctx).mod()
        )
        if tt.IsZero() {
            note(ctx, "target not exists: %v", target).debug(1)
            return
        }
        for _, dep := range merge(depends, ordered, grepped) {
            if dt := dep.stat(ctx).mod(); dt.After(tt) {
                note(ctx, "%v: outdated by %v (%v)", target, dep, dt.Sub(tt)).debug(1)
            }
        }
    }
    if len(ctx.info) == 0 && len(ctx.warn) == 0 && len(ctx.erro) == 0 {
        var ( p = ctx.Position() ; s = _stems(ctx) ; m *diagPoint )
        if len(args) == 0 {
            m = prompt(ctx, "%v: target=%v stems=%v depends=%v\n", p, target, s, depends)
        } else if ctx.verbose {
            m = prompt(ctx, "%v: target=%v stems=%v depends=%v ; %v\n", p, target, s, depends, args)
        } else if len(args) == 1 {
            m = prompt(ctx, "%v: %v (%T)\n", p, args[0], args[0])
        } else {
            m = prompt(ctx, "%v: %v\n", p, args)
        }
        if ctx.s > 0 { m = infostack(ctx, ctx.s) }
        if ctx.n > 0 { m.debug(ctx.n) }
    }
    return
}

type modifier_print struct { modifier_
    stdout bool `o,stdout`
    stderr bool `e,stderr` // TODO: = true
    reset  bool `r,reset`
}
func (ctx *modifier_print) x(args ...Value) (result interface{}) {
    var content string
    if val := autoVal(ctx, "-"); val != nil { content = val.string(ctx) }
    if ctx.stdout { fmt.Fprint(stdout, content) }
    if ctx.stderr { fmt.Fprint(stderr, content) }
    if ctx.reset  { autoSet(ctx, "-", makeNone(ctx.Position())) }
    return
}

type modifier_prompt struct { modifier_ }
func (ctx *modifier_prompt) x(args ...Value) (result interface{}) {
    for _, a := range args { prompt(ctx, "%s\n", a.string(ctx)) }
    if len(args) == 0 { if h := autoVal(ctx, "-"); h != nil {
        prompt(ctx, "%s\n", h.string(ctx))
    }}
    return
}

type modifier_preserve struct { modifier_ }
func (ctx *modifier_preserve) v(args ...Value) (result interface{}) {
    return args
}

type modifier_expand struct { modifier_ }
func (ctx *modifier_expand) v(args ...Value) (result interface{}) {
    result = expand(ctx, args...)
    return
}

type modifier_plain struct { modifier_ }
func (ctx *modifier_plain) v(args ...Value) (result interface{}) {
    result = expand(ctx, args...) // plain
    return
}

type modifier_stringify struct { modifier_ }
func (ctx *modifier_stringify) v(args ...Value) (result interface{}) {
    result = expand(ctx, args...) // final
    return
}

type modifier_reveal struct { modifier_ }
func (ctx *modifier_reveal) v(args ...Value) (result interface{}) {
    result = expand(ctx, args...) // expandDef1
    return
}

type modifier_disclose struct { modifier_ }
func (ctx *modifier_disclose) v(args ...Value) (result interface{}) {
    result = expand(ctx, args...) // expandDef2
    return
}

// select element by index from group result: (select 0)
type modifier_select struct { modifier_ }
func (ctx *modifier_select) x(args ...Value) (result interface{}) {
    args = xmerge(ctx, args...)
    if h := autoVal(ctx, "-"); h == nil {
        erro(ctx, "no pipe value $-").debug(1)
    } else if g, ok := h.(*group); ok && len(args) > 0 {
        if i, e := args[0].int(ctx); e != nil {
            erro(ctx, "%v: %v", args[0], e).debug(1)
        } else {
            result = g.at(int(i))
        }
    }
    return
}

type modifier_env struct { modifier_ }
func (ctx *modifier_env) x(args ...Value) (result interface{}) {
    args = xmerge(ctx, args...)

    if pc := cast[*programContext](ctx); pc != nil {
        for _, a := range args {
            if p, y := a.(*pair); y {
                pc._env = append(pc._env, p)
            } else {
                erro(ctx, "env: not a pair value: %v (%T)", a, a).debug(1)
            }
        }
    }
    return
}

type modifier_var struct { modifier_ }
func (ctx *modifier_var) x(args ...Value) (_ interface{}) {
    // noop; TODO: do something better
    return
}

// examples:
//     [(set name=value)]    set $(name) to 'value'
//     [(set name)]          clear $(name)
//     [(set -)]             clear $-
type modifier_set struct { modifier_ }
func (ctx *modifier_set) x(args ...Value) (_ interface{}) {
    defer trace(ctx)

    var pc = cast[*programContext](ctx)

ForArgs:
    for _, arg := range args {
        var name string
        var value Value
        switch a := arg.(type) {
        case *bareword: name = a.s
        case *pair: // NOTE: pair.Value is not expanded, need to do it again.
            name, value = a.key.string(ctx), a.val.expand(final{ctx})
            if value == nil { value = a.val }
        case flag:
            name, value = a.Value.string(ctx), makeNone(a.Position())
            if name == "" { name = "-" }
        default:
            erro(ctx, "%v is unsupported (try: foo=value)", us(arg)).debug(1)
            return
        }

        var d *def
        var isauto, y bool
        var o = pc.prog.scope.resolve(name)
        if o == nil {
            erro(ctx, "no such def '%s' (%v, %v)", name, arg, args).debug(1)
            break ForArgs
        } else if _, isauto = o.(*auto); isauto {
            d, _ = autoSet(ctx, name, nil)
        }

        if d == nil { if d, y = o.(*def); !y {
            erro(ctx, "not a def: %v", name, us(o)).debug(1)
            break ForArgs
        }}

        if d.val(ctx, value); !isauto && d.value != nil && value != nil {
            erro(ctx, "set def wrong: %v (auto: %v)", us(value), autoVal(ctx, d.ident(ctx))).debug(1)
        }

        if isauto && name == "@" {
            var f, s, y = as{value}.fullnameFile(ctx)
            if ctx.verbose {
                var ts = trimPromptString(s)
                prompt(ctx, "%s …… traversed (%d)\n", ts, f._traved)
                if false { warnstack(ctx, 64).debug(64) }
            }
            if y && f._traved > 1 {
                pc.traves.add(ctx, traveDone, nil)
            }
        }
    }
    return
}

type modifier_defer struct { modifier_ }
func (ctx *modifier_defer) x(args ...Value) (_ interface{}) {
    if pc := cast[*programContext](ctx); pc != nil { pc.defers = append(pc.defers, args...) }
    return
}

type modifier_setDirtyPats struct { modifier_
    pats []Value
}
func (ctx *modifier_setDirtyPats) x(args ...Value) (result interface{}) {
    ctx.pats = parseOpts(ctx, ctx.dirtyOpts(), args...) //, plain
    return
}

// create closure context for the traversal
type modifier_closure struct { modifier_
    target Value `@,target`
    // depFirst bool `<,dep-first` // TODO: -<=value
    // depLast  bool `>,dep-last` // TODO: ->=value
}
func (ctx *modifier_closure) x(pc *programContext, args ...Value) (result interface{}) {
    // Closure the caller program, the context will be restored when execution is finished.
    var cc = pc.Context
    pc.Context = closureWith(cc)
    assert(cast[*terminal](ctx) == pc.Context, "closure context: %v", ctx)

    var proj = ctx.project()
    var set = func(name string, val Value) (t Value) {
        var noop bool
        if v, y := val.(*boolean); y {
            if !v.bool { noop = true }
        } else if isTrivial(val) {
            errostack(ctx, 3, "trivial target: %T %v", val, val).debug(1)
        } else if true {
            t = val.expand(ctx) //, plain
        } else {
            t = val
        }

        if l, y := t.(*list); y && len(l.elems) == 1 { t = l.elems[0] }
        if !noop && isTrivial(t) { t = autoVal(ctx, name)  }

        if t != nil {
            autoSet(ctx, name, t) // aka (set @=&@)
        } else if !noop {
            errostack(ctx, 3, "%v: %s is nil", proj, name).debug(16)
        }
        return
    }

    var target Value
    if ctx.target != nil {
        if target = ctx.target.expand(ctx); target == ctx.target {
            if t := autoVal(cc, "@"); t != nil {
                target = t
            }
        }
    }
    if ctx.verbose { var t = target
        note(ctx, "%v: @: %v ⇒ %v %v", proj, ctx.target, typeof(t), t).debug(3)
    }
    if target != nil {
        var ( t = as{set("@", target)} ; f *File ; s string ; y bool ; n int )
        if f, s, y = t.fullnameFile(ctx); !y {
            s = t.string(ctx)
        } else {
            n = f._traved
        }

        if n > 1 {
            if ctx.verbose {
                var ts = trimPromptString(s)
                prompt(ctx, "%s …… traversed (%d, %v)\n", ts, n)
                if false { warnstack(ctx, 64, "%v, %v, (%d)", f, s, n).debug(64) }
            }

            pc.traves.add(ctx, traveDone, nil)
            return
        }

        // FIXME: if isInnerautoVal(ctx, t.Value) {
        //     errostack(ctx, 16, "loop: %v", t).debug(10)
        //     return
        // }
    }

    if proj == nil {
        errostack(ctx, 6, "%T: nil project in the context", ctx).debug(64)
    } else if scope := proj.scope; scope == nil {
        erro(ctx, "empty closure context").debug(1)
    } else if def := scope.FindDef("/"); def == nil {
        erro(at(ctx,scope.position), "&/ is undefined").debug(1)
    } else if dir := def.value.string(ctx); dir == "" {
        erro(at(ctx,scope.position), "&/ is empty").debug(1)
    } else if !filepath.IsAbs(dir) {
        erro(at(ctx,scope.position), "&/ is relative").debug(1)
    } else /* if err := enter(ctx, dir); err == nil */ {
        pc.changedWD = dir
    }
    return
}

type modifier_for struct { modifier_ }
func (ctx *modifier_for) x(args ...Value) (result interface{}) {
    // TODO: ...
    return
}

type modifier_cd struct{ modifier_
    path bool `path`
    printEnter bool `print-enter`
    printLeave bool `print-leave`
}
func (ctx *modifier_cd) x(args ...Value) (result interface{}) {
    // if false && ctx.printEnter { promptEnteringDirectory(ctx) }
    // if false && ctx.printLeave { promptLeavingDirectory(ctx) }
    if (ctx.printEnter || ctx.printLeave) && len(args) == 0 { return }
    if len(args) == 1 {
        var dir = args[0].string(ctx)
        if dir == "" {
            // TODO: do something special
            return
        }

        var proj = ctx.project()
        if !filepath.IsAbs(dir) { dir = filepath.Join(proj.absPath, dir) }
        if ctx.path && dir != "." && dir != ".." && dir != pathSep {// mkdir -p
            if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                erro(ctx, "make path '%s' failed: %v", dir, err)
                return
            }
        }
        // if err := enter(ctx, dir); err == nil {
        if pc := cast[*programContext](ctx); pc != nil { pc.changedWD = dir }
        // }
    } else {
        erro(ctx, "wrong number of cd args: %v", args).debug(1)
    }
    return
}

type modifier_mkdir struct { modifier_
    mode os.FileMode `mode`
}
func (ctx *modifier_mkdir) x(args ...Value) (result interface{}) {
    if ctx.mode == 0 {
        ctx.mode = os.FileMode(0755)
    } else {
        ctx.mode |= os.FileMode(0111)
    }
    if len(args) == 0 { if v := autoVal(ctx, "@"); !isTrivial(v) {
        args = append(args, v)
    }}
    for _, a := range xmerge(ctx, args...) { //, final
        var s string
        if f, y := a.(*File); y {
            s = f.fullname()
        } else {
            s = a.string(ctx)
        }
        if strings.Contains(s, " /") || strings.Contains(s, " ./") || strings.Contains(s, " ../") {
            erro(at(ctx, a), "multiple paths (%v): '%v'", typeof(a), s).debug(1)
            break
        } else if strings.Contains(s, " ") {
            warn(at(ctx, a), "path containing spaces (%v): '%v'", typeof(a), s).debug(1)
        }
        if err := os.MkdirAll(s, ctx.mode); err != nil {
            erro(ctx, "path: %v(%v) ⇒ %s: %v", typeof(a), a, s, err).debug(1)
            break
        }
    }
    return
}

type modifier_sudo struct { modifier_ }
func (ctx *modifier_sudo) x(args ...Value) (result interface{}) {
    erro(at(ctx,ctx.Position()), "TODO: sudo modifier is not implemented yet").debug(1)
    return
}

func parseDependList(ctx Context, dependList *list) (depends *list) {
    var pc = cast[*programContext](ctx)
    depends = new(list)
    for _, depend := range dependList.elems {
        switch d := depend.(type) {
        case *list:
            if dl := parseDependList(ctx, d); dl != nil {
                depends.elems = append(depends.elems, dl.elems...)
            }
        case *execResult:
            if d.Status != 0 {
                brk := pc.traves.add(ctx, traveFail, nil)
                brk.error = fmt.Errorf("bad status %v", d.Status)
                return // target shall be updated
            } else {
                depends.append(d)
            }
        case *rule:
            switch d.class {
            case GeneralRule, PatternRule, pathPatRule:
                depends.append(d)
            default:
                erro(ctx, "unsupported entry depend `%v' (%v)", d, d.class).debug(1)
            }
        case *strlit:
            depends.append(d)
        case *File:
            depends.append(d)
        default:
            var program = _program(ctx)
            erro(ctx, "unsupported entry depend `%v' (%v)", depend, program.depends).debug(1)
        }
    }
    return
}

type langInfoT struct {
    rxs []string
    sys []string
}

var langInfos = map[string]*langInfoT{
    "asm": &langInfoT{
        []string{
            `^\s*#\s*include\s*"(.+)".*$`,
        },
        []string{
            `^\s*#\s*include\s*<(.+)>.*$`,
        },
    },
    "c": &langInfoT{
        []string{
            `^\s*#\s*include\s*"(.+)".*$`,
        },
        []string{
            `^\s*#\s*include\s*<(.+)>.*$`,
        },
    },
    "i": &langInfoT{
        []string{
            `^\s*include\s*"(.+)".*$`,
        },
        []string{
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
    file *File
    list []*File
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
    savedGrepFile *File
    save *bufio.Writer
}
type greprex struct{ string ; bool ; *regexp.Regexp }
func (g *greprex) String() string { return g.string }
func (g *greptouch) work(ctx Context, gc *grepctx) (err error) {
    if g.targetInfo == nil {
        erro(at(ctx,g.target.Position()), "'%v' not exists", g.target).debug(1)
        return
    }
    var tt time.Time = g.targetInfo.ModTime()
    for _, val := range g.files {
        var file *File
        if file, _ = toFile(val); file == nil {
            erro(ctx, "'%v' is not file (%T)", val, val).debug(1)
            return
        }
        if file.info == nil && !file.isSysFile() {
            if file.info, _ = os.Stat(file.string(ctx)); file.info == nil { continue }
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
            erro(ctx, "%v", err).debug(1)
        }
    }
    return
}
func (g *grepctx) isTargetFile(ctx Context, file *File) (res bool) {
    if file == nil {
        // ...
    } else if g.target.Value == file {
        res = true
    } else if s, _ := g.target.fullnameOrFinal(ctx); s == g.targetFullName {
        res = true
    } else if f, y := toFile(g.target.Value); y && f.ident(ctx) == file.ident(ctx) {
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
                file := stat(ctx, a[0], stat_sub{a[1]}, stat_dir{a[2]})
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
            var file, ok = toFile(v)
            if !ok { continue }
            fmt.Fprintf(w, "%s|%s|%s\n", file.ident(ctx), file.sub, file.dir)
        }
    }
}

func searchGreppedName(ctx Context, gp Position, gc *grepctx, sys bool, name string) (res *File) {
    var isAbs, isRel bool
    if isAbs = filepath.IsAbs(name); isAbs {
        res = stat(ctx, name, stat_nonexist{true})
    } else if isRel = isRelPath(name); isRel { // relative to targetDir
        res = stat(ctx, name, stat_dir{gc.targetDir}, stat_nonexist{true})
    } else if res = file(ctx, name); res != nil && res.exists() {
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
        erro(ctx, "%v: %v → %v (exists=%v, sys=%v, from %v)\n",
            _entry(ctx), gc.target, name, res.exists(), sys, ctx.project()).
            debug(gc.debug)
    }
    if sys || res.exists() { return }

    // relative to target directory
    var alt = stat(ctx, name, stat_dir{gc.targetDir})
    if alt != nil { res = alt; return }

    // Check for bare non-system sub-paths:
    //   foo/bar/name.xxx
    // We search base name 'name.xxx' again:
    var s = filepath.Dir(name) // e.g: foo/bar

    // Search 'name.xxx' and check dir for
    // 'foo/bar' suffix. We use it if found.
    alt = file(ctx, filepath.Base(name))
    if alt != nil && strings.HasSuffix(alt.dir, pathSep+s) {
        dir := strings.TrimSuffix(alt.dir, pathSep+s)
        ok1 := alt.change(dir, s, alt.ident(ctx)) // <dir>, foo/bar, name.xxx
        ok2 := alt.change(dir, "", name) // <dir>, "", foo/bar/name.xxx
        res  = alt
        if enable_assertions {
            assert(ok1, "unchanged: %s %s %s", dir, s, alt.ident(ctx))
            assert(ok2, "unchanged: %s %s", dir, alt.ident(ctx))
        }
    } else if res == nil {
        for _, inc := range gc.incs {
            if res = stat(ctx, name, stat_dir{inc.string(ctx)}); res != nil {
                if false { info(ctx, "%v in %v", res, inc).debug(1) }
                return
            }
        }
        if res == nil { res = stat(ctx, name, stat_nonexist{true}) }
        warn(at(ctx,gp), "'%s' not found in %v", name, ctx.project())
        warn(ctx, "grepped '%s' has no target dir in %v", name, ctx.project())
        warn(at(ctx,ctx.project().position), "from project %v (for %v)", ctx.project(), name).debug(8)
    }
    return
}

func searchGrepped(ctx Context, gp Position, gc *grepctx, sys bool, name string) (file *File, err error) {
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
            if file.info, err = os.Stat(file.string(ctx)); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
            }
            if false || gc.debug>0 {
                warn(ctx, "'%v' info is nil (%s)",
                    file, file.fullname()).debug(gc.debug)
            }
        }
        if file.info == nil {/* ... */} else
        if tv := file.info.ModTime(); tv.After(tt) {
            if true || gc.debug>0 {
                warn(ctx, "touch %v → %v (%v)",
                    gc.target, file, tv).debug(gc.debug)
            }
            tv = launchTime //time.Now() // ...
            if err, tt = os.Chtimes(gc.targetFullName, tv, tv), tv; err != nil {
                erro(ctx, "chtimes failed: %v", err).debug(1)
                return
            }
        }
    }

    // Report missing files, but system files are not treated as missing.
    if !gc.report {
        // ...
    } else if file == nil {
        info(at(ctx,gp), "%s: `%s` not found", ctx.project().name, name)
    } else if !file.exists() {
        info(at(ctx,gp), "%s: `%s` file not existed", ctx.project().name, name)
    }
    return
}

func tempFile(ctx Context, prefix, hashee0 string, hasheeN... interface{}) (file *File, err error) {
    var nameHash = sha256.New() // HashByte -> [sha256.Size]byte
    if _, err = fmt.Fprint(nameHash, prefix, hashee0); err != nil {
        erro(ctx, "hashing failed: %v", err).debug(1)
    } else if _, err = fmt.Fprint(nameHash, hasheeN...); err != nil {
        erro(ctx, "hashing failed: %v", err).debug(1)
    } else if nameSum := nameHash.Sum(nil); len(nameSum) != nameHash.Size() {
        erro(ctx, "hash sum invalid: %v", len(nameSum)).debug(1)
    } else if project := ctx.project(); project == nil {
        erro(ctx, "current project is nil: %v", ctx).debug(1)
    } else {
        // Make names like .deps/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
        // .deps/??/??/????????????????????????????????????????????????????????????
        // .grep/??/??/????????????????????????????????????????????????????????????
        // .cache/??/??/????????????????????????????????????????????????????????????
        file = project.tempFile(ctx, filepath.Join(prefix, // e.g. ".deps", ".grep"
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
        if file, err := tempFile(ctx, dir, ""); err != nil {
            erro(ctx, "%v", err).debug(1)
            return
        } else if s := file.fullname(); s == "" {
            erro(ctx, `"%v" has no fullname`, file).debug(1)
            return
        } else if s = filepath.Dir(filepath.Dir(filepath.Dir(s))); s == "" {
            erro(ctx, `"%v" is invalid temp dir`, file.fullname()).debug(1)
            return
        } else if err = os.RemoveAll(s); err != nil {
            erro(ctx, "%v", err).debug(1)
            return
        } else if false {
            info(ctx, "%s: removed %v", ctx.project(), s).debug(1)
        } else {
            prompt(ctx, "%s: removed %v\n", ctx.project(), s)
        }
    }
}

func getSavedDepsFileName(ctx Context, targetFullName string, strs []string) (filename string, err error) {
    var ( file *File; hashees []interface{} )
    for _, s := range strs { hashees = append(hashees, s) }
    if file, err = tempFile(ctx, ".deps", targetFullName, hashees...); err != nil {
        erro(ctx, "get .deps temp file failed: %v", err).debug(1)
    } else {
        filename, _ = as{file}.fullnameOrFinal(ctx)
    }
    return
}

func getSavedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
    var ( file *File )
    if file, err = tempFile(ctx, ".grep", targetFullName); err != nil {
        erro(ctx, "get .grep temp file failed: %v", err).debug(1)
    } else {
        filename, _ = as{file}.fullnameOrFinal(ctx)
    }
    return
}

func loadSavedGrepFile(ctx Context, gc *grepctx) (okay bool, err error) {
    if gc.savedGrepFileName, err = getSavedGrepFileName(ctx, gc.targetFullName); err != nil {
        erro(ctx, "get saved grep filename failed: %v", err).debug(1)
        return
    } else if gc.savedGrepFile = stat(ctx, gc.savedGrepFileName); gc.savedGrepFile == nil {
        return // No saved grepfile yet!
    }

    var file, ok = toFile(gc.target)
    if !ok {
        file = stat(ctx, gc.targetFullName)
        if file != nil { gc.target.Value = file }
    }
    if file != nil && file.info != nil {
        // Check previously saved grep file into.
        if file.info.ModTime().After(gc.savedGrepFile.info.ModTime()) {
            return
        }
    }

    var savedGrepOSFile *os.File
    if savedGrepOSFile, err = os.Open(gc.savedGrepFileName); err != nil {
        erro(ctx, "open saved grep filename failed: %v", err).debug(1)
        return
    }
    defer savedGrepOSFile.Close()

    var gp Position
    //gp.Filename = gc.savedGrepFileName
    gp.Filename = gc.targetFullName

    scanner := bufio.NewScanner(savedGrepOSFile)
    scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
        var s = scanner.Text() //gp.Line += 1
        var ( sys int; name string )
        if n, e := fmt.Sscanf(s, "%d %d %d %s", &sys, &gp.Line, &gp.Column, &name); e == nil && n == 4 {
            var file *File
            if file, err = searchGrepped(ctx, gp, gc, sys == 1, name); err != nil {
                erro(ctx, "search grepped filename failed: %v", err).debug(1)
                break
            } else if file != nil {
                file.position = gp
                if gc.isTargetFile(ctx, file) { continue }
            } else if sys != 1 && !gc.discard {
                warn(at(ctx,gp), "%s is nil file", name)
                warn(ctx, "grepped %s is nil", name)
                warn(at(ctx,ctx.project().position), "from project %v", ctx.project()).debug(6)
            }
        }
    }
    if gc.savedGrepFile.info, err = savedGrepOSFile.Stat(); err != nil {
        erro(ctx, "stat saved grep filename error: %v", err).debug(1)
    } else { okay = true }
    return
}

func grepTargetFile(ctx Context, gc *grepctx) (err error) {
    var ( file *os.File )
    if file, err = os.Open(gc.targetFullName); err != nil {
        erro(ctx, "%v", err).debug(1)
        return
    } else { defer func() { err = file.Close() } () }

    for _, x := range gc.rxs {
        if x.Regexp != nil {
            continue
        } else if x.Regexp, err = regexp.Compile(x.string); err != nil {
            erro(ctx, "%v", err).debug(1)
            return
        }
    }

    var gp Position
    gp.Filename = gc.targetFullName


    scanner := bufio.NewScanner(file)
    scanner.Split(bufio.ScanLines)
ForScan:
    for scanner.Scan() {
        var s = scanner.Text(); gp.Line += 1
        for _, x := range gc.rxs {
            if sm := x.FindStringSubmatch(s); len(sm) > 1 && sm[1] != "" {
                var ( file *File ; name = sm[1]; sys = x.bool ) //strings.IndexFunc(s, isNotSpace)
                if gp.Column = strings.Index(s, name); gc.save != nil {
                    var d = 0 ; if sys { d = 1 } // system files
                    fmt.Fprintf(gc.save, "%d %d %d %s\n", d, gp.Line, gp.Column, name)
                }
                if file, err = searchGrepped(ctx, gp, gc, sys, name); err != nil {
                    erro(ctx, "search grepped '%s' failed: %v", name, err).debug(1)
                    return
                } else if file != nil {
                    if file.position = gp; gc.isTargetFile(ctx, file) { continue }
                } else if !sys && !gc.discard {
                    warn(at(ctx,gp), "%s is nil file", name)
                    warn(ctx, "grepped %s is nil", name)
                    warn(at(ctx,ctx.project().position), "from project %v", ctx.project()).debug(6)
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
    case *File:
        targetName = v.ident(ctx)
        gc.targetInfo = v.info
        gc.targetFullName = v.fullname()
        gc.targetDir = filepath.Dir(gc.targetFullName)
        if v.isSysFile() { return }
    default:
        gc.targetDir = ctx.project().absPath
        targetName = v.string(ctx)
        if filepath.IsAbs(targetName) {
            gc.targetFullName = targetName
        } else {
            gc.targetFullName = filepath.Join(gc.targetDir, targetName)
        }
        if file := stat(ctx, gc.targetFullName); file == nil {
            erro(at(ctx,gc.target), "grep: '%s' not found (%v)", gc.targetFullName, gc.target).debug(16)
            return
        } else {
            gc.targetInfo = file.info
        }
    }
    if err != nil {
        erro(ctx, "grep target %s: %v", targetName, err).debug(1)
        return
    }

    if gc.targetInfo == nil { return }
    if gc.done == nil { gc.done = make(map[string]int) }
    if !filepath.IsAbs(gc.targetFullName) {
        erro(ctx, "grep: '%s' is not abs", gc.targetFullName).debug(1)
        return
    } else {
        gc.done[gc.targetFullName] += 1
    }
    if n, done := gc.done[gc.targetFullName]; done && n > 1 {
        if gc.debug>0 { erro(ctx, "%v (done %v)", gc.targetFullName, n).debug(gc.debug) }
        return
    }

    //var infos = strings.Contains(gc.targetFullName, "...")
    const infos = false

    if false { defer un(tt(l_traverse, cast[*programContext](ctx), gc.target)) }

    defer func(restore []Value) {
        var t = cast[*programContext](ctx)
        var touch = gc.greptouch // copy greptouch value
        if len(touch.files) > 0 {
            grepcacheM.Lock()
            grepcache[gc.targetFullName] = touch.files
            grepcacheM.Unlock()
        } else if false {
            var gp Position
            gp.Filename, gp.Line = gc.targetFullName, 1
            warn(at(ctx,gp), "grebbed zero files")
            warn(ctx, "grebbed zero files: %v", gc.targetFullName).debug(6)
        }
        gc.files = restore
        if gc.debug>0 { erro(ctx, "grepped: %s → %v (grepped=%v) (saved=%s)\n",
            gc.target, touch.files, len(t.grepped), gc.savedGrepFile).debug(gc.debug) }
        for _, gc.target.Value = range touch.files {
            if t.grepped = append(t.grepped, gc.target); !gc.recursive {
                continue
            } else if err = grep(ctx, gc); err != nil {
                erro(ctx, "grep files (deferred): %v", err).debug(1)
                break
            }
        }
        if err == nil && gc.touch {
            if err = touch.work(ctx, gc); err != nil {
                erro(ctx, "grep touch failed: %v", err).debug(1)
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
        if gc.debug>0 { erro(ctx, "grepcache: %v → %v",
            gc.targetFullName, gc.files).debug(gc.debug) }
        return
    } else if infos {
        info(ctx, "grepcache: %s files=%d", gc.targetFullName, len(gc.files)).debug(1)
    }

    if savedGrepFileLoaded, err = loadSavedGrepFile(ctx, gc); err != nil {
        erro(ctx, "load saved grepfile failed: %v", err).debug(1)
        return
    } else if savedGrepFileLoaded && len(gc.files) > 0 {
        if infos { info(ctx, "loadSavedGrepFile: %v files=%d grepped=%d",
            gc.targetFullName, len(gc.files), len(cast[*programContext](ctx).grepped)).debug(1) }
        return
    }
    if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
            erro(ctx, "make grep dir failed: %v", err).debug(1)
            return
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
            erro(ctx, "grep write file: %v", err).debug(1)
            return
        } else if false {
            info(ctx, "saved grep %s", name).debug(1)
        }
    }
    if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil {
        erro(ctx, "grep create %s: %v", gc.savedGrepFileName, err).debug(1)
        return
    }

    gc.save = bufio.NewWriter(savedGrepFile)
    defer func() {
        gc.save.Flush()
        savedGrepFile.Close()
    } ()

    if err = grepTargetFile(ctx, gc); err != nil && !gc.discard {
        erro(ctx, "grep target file: %v", err).debug(1)
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
func (ctx *modifier_grep) x(args ...Value) (result interface{}) {
    var uni = _universe(ctx)
    if false && uni.noDepsGrep || uni.noGrep { return }

    var gc = grepctx{ modifier_grep:ctx }
    // gc.fileinc = true // grep files by default
    gc.incs = xmerge(ctx, gc.incs...)//, plain
    for _, s := range gc.sys { gc.rxs = append(gc.rxs, &greprex{s, true , nil}) }
    for _, s := range gc.reg { gc.rxs = append(gc.rxs, &greprex{s, false, nil}) }
    for _, s := range gc.langs {
        if info, ok := langInfos[s]; ok && info != nil {
            for _, re := range info.rxs { gc.rxs = append(gc.rxs, &greprex{re, false, nil}) }
            for _, re := range info.sys { gc.rxs = append(gc.rxs, &greprex{re, true , nil}) }
        } else {
            erro(ctx, "lang '%s' is unknown", s).debug(1)
            return
        }
    }
    if len(gc.rxs) == 0 {
        erro(ctx, "no grep expressions: %v %v %v %v", gc.sys, gc.reg, gc.langs, args).debug(1)
        return
    }

    var (
        target = autoVal(ctx, "@")
        targets = args
        grepped = cast[*programContext](ctx).grepped
    )
    if len(targets) == 0 { if target == nil || isNull(target) || isNone(target) {
        erro(ctx, "no grep target").debug(1)
        return
    } else {
        targets = append(targets, target)
    }}

    if gc.debug > 0 {
        warn(ctx, "grep files: %v %v %v\n", target, gc.rxs, args).debug(gc.debug)
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
            prompt(ctx, "Grep %v …… (%d files in %v)\n", s, len(grepped), time.Now().Sub(ts)).debug(gc.debug, 6)
        } (time.Now())
    }

    var pc = cast[*programContext](ctx)
    var tar = target
    defer func(v bool) { pc.grepping = v } (pc.grepping)
    pc.grepping = true

ForTarget:
    for _, target := range targets {
        if isNull(target) {
            erro(ctx, "found nil grep target for %v", tar).debug(1)
            return
        }
        if isNone(target) {
            erro(ctx, "grep target '%v' is none for %v", target, tar).debug(32)
            return
        }

        gc.target.Value, pc.grepped = target, nil
        if err := grep(ctx, &gc); err != nil {
            erro(ctx, "grep files from %v failed: %v", target, err).debug(1)
            return
        } else if gc.noTraverse {
            // does nothing
        } else if len(pc.grepped) > 0 {
            for _, val := range pc.grepped {
                if val.traverse(ctx); !pc.traves.has() { continue }
                for _, brk := range pc.traves { erro(at(ctx,brk.pos), "%v: %v", val, brk).debug(1) }
                erro(ctx, "broken traversal for grepped %v from %v", val, target)
                errostack(ctx, 5, "%v", ctx).debug(16)
                break ForTarget
            }
        }
        grepped = append(grepped, pc.grepped...)
    }
    pc.grepped = grepped

    if !gc.noTraverse {
        autoSet(ctx.Context, "~", makeNone(ctx.Position()))
        pc.grepped = nil
    } else {
        result = ease(ctx, pc.grepped)
    }
    return
}

type depContext struct { diagnostic }
func (ctx *depContext) inner() Context { return &ctx.diagnostic }
func (ctx *depContext) cast(t reflect.Type) (c Context) {
    if reflect.TypeOf(ctx) == t { return ctx }
    return ctx.diagnostic.cast(t)
}
func (ctx *depContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("dep{%v}", ctx.diagnostic)
    } else {
        return ctx.diagnostic.String()
    }
}

func parseDeps(ctx Context, targetVal Value, targetStr string, savedDepsFile *File, savedDepsFileName, deps string) (files []Value) {
    const parallel = true
    var (
        proj = ctx.project()
        targetFullName, _ = as{targetVal}.fullnameOrFinal(ctx)
        filesMux sync.Mutex
        firstWord string
        err error
    )
    var findDepFile = func(name string) (file *File) {
        if filepath.IsAbs(name) {
            file = stat(ctx, name, stat_nonexist{true})
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
    var addFile = func(file *File) {
        filesMux.Lock()
        files = append(files, file)
        filesMux.Unlock()
    }
    var (
        missing = make(map[string]Position)
        missMux sync.Mutex
        jobs sync.WaitGroup
    )

    var pc = cast[*programContext](ctx)
    var depFile = func(ctx Context, depPos Position, word string) {
        var dc = depContext{diagnostic{ Context: ctx }}

        ctx = &dc

        if parallel { defer func() {
            if false { assured(ctx, true/* don't call diagFlush */) }
            if len(dc.points) > 0 { _diagnostic(inner(ctx)).nest(dc.points) }
            jobs.Done() // minus 1
        }()}

        if i := strings.Index(word, " "); i > 0 {
            warn(ctx, "ignore dep with spaces: %v", word).debug(1)
        } else if file := findDepFile(word); file == nil {
            prompt(ctx, "%v: unknown dep\n", file)
            if savedDepsFile != nil {
                warn(ctx, "unknown dep '%v' for '%v'", word, firstWord)
                warn(at(ctx,depPos), "from here: %s", word)
                if filepath.IsAbs(firstWord) {
                    var wp Position
                    wp.Filename, wp.Line = firstWord, 1
                    warn(at(ctx,wp), "in here: %v", word)
                }
                warn(at(ctx,proj.position), "for project %v", proj)//.debug(6)
            } else {
                erro(ctx, "unknown dep '%v' for '%v'", word, firstWord)
                erro(at(ctx,depPos), "from here: %s", word)
                if filepath.IsAbs(firstWord) {
                    var wp Position
                    wp.Filename, wp.Line = firstWord, 1
                    erro(at(ctx,wp), "in here: %v", word)
                }
                erro(at(ctx,proj.position), "for project %v", proj)//.debug(6)
            }
        } else if ignored(file.fullname()) {
            //continue // dep is the target itself
        } else if file.traverse(ctx); !pc.traves.has() {
            addFile(file)
        } else if t := pc.traves.not(traveCase, traveDone, traveNext); t.has() {
            prompt(ctx, "%v: missing dep\n", file)
            if savedDepsFile != nil {
                var s = filepath.Base(file.ident(ctx))
                warn(at(ctx,depPos), `%v: missing "%v"`, targetVal, s)
                warnstack(ctx, 3, "%v: (%T):", proj, ctx).debug(4)
            } else {
                erro(at(ctx,depPos), `%v: missing "%v"`, targetVal, file)
                for _, brk := range t {
                    erro(at(ctx,brk.pos), `%v: broken for "%s": %v`, proj, targetVal, brk)
                }
                errostack(ctx, 5, "%v: (%T):", proj, ctx).debug(16)
            }
        } else {
            addFile(file)
        }

        var n int
        if savedDepsFile == nil {
            if n = flush(dc.Context); n > 0 { // aka. dc.points = nil
                var s = trimPromptString(targetVal.String())
                prompt(ctx, "%v: %d errors counted\n", word, n).debug(1)
                erro(ctx, `%v: %d errors for "%s", dep "%s"`, proj, n, s, word)
                errostack(ctx, 5, `%v: %v`, ctx).debug(6)
            }
        } else {
            if n = _diagnostic(dc.Context).countError(); n > 0 {
                // reset to reduce diags as we wish to continue with the errors
                dc.points, dc.erros = nil, 0
                var s = trimPromptString(targetVal.String())
                prompt(ctx, "%v: %d errors counted\n", word, n).debug(1)
                if false {
                    warn(ctx, `%v: %d errors for "%s", dep "%s"`, proj, n, s, word)
                    warnstack(ctx, 3, `%v: %v`, ctx).debug(6)
                }
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
                } else if firstDepFile := stat(ctx, firstDep); firstDepFile == nil {
                    return nil // requests to update savedDepsFile
                } else if firstDepFile.info.ModTime().After(savedDepsFile.info.ModTime()) {
                    return nil // requests to update savedDepsFile
                }
                if parallel {
                    // jobs.Add(1); go depFile(ctx.spawn(ctx), depPos, word)
                } else {
                    depFile(ctx, depPos, word)
                }
            }
        }
    }
    if jobs.Wait(); len(missing) > 0 {
        prompt(ctx, "%v: %d deps missing, removing deps file\n", savedDepsFileName, len(missing))
        if savedDepsFile == nil || savedDepsFileName == "" {
            // deps files not saved yet
        } else if err = os.Remove(savedDepsFileName); err != nil {
            for s, p := range missing { erro(at(ctx,p), `missing "%v"`, s) }
            erro(ctx, `%v: "%v" %d deps missing in "%v"`, proj, targetVal, len(missing), savedDepsFileName)
            errostack(ctx, 3, "%v", ctx).debug(10)
            panic(_failure(ctx, "removed %s", savedDepsFileName))
        } else {
            for s, p := range missing { warn(at(ctx,p), `missing "%v"`, s) }
            warn(ctx, `%v: "%v" missing %d deps (%v in total)`, proj, targetVal, len(missing), len(files))
            warnstack(ctx, 3, "%T:", ctx).debug(6)
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
    if targetVal, targetStr := getTargetValueString(ctx); isNull(targetVal) {
        erro(ctx, "target is nil").debug(1)
    } else if targetStr == "" {
        erro(ctx, "target '%v' is empty", targetVal).debug(1)
    } else if savedDepsFileName, err = getSavedDepsFileName(ctx, targetStr, args); err != nil {
        erro(ctx, "get saved deps filename failed: %v", err).debug(1)
    } else if savedDepsFileName == "" {
        erro(ctx, "empty saved deps filename", savedDepsFileName).debug(1)
    } else if savedDepsFile := stat(ctx, savedDepsFileName); savedDepsFile == nil {
        // no saved deps file
    } else if savedDepsBytes, err = ioutil.ReadFile(savedDepsFileName); err != nil {
        erro(ctx, "can'ctx open saved deps file: %v", savedDepsFileName, err).debug(1)
    } else if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, string(savedDepsBytes)); len(files) > 0 {
        if false { info(ctx, "loaded deps %s (%d files)", savedDepsFileName, len(files)).debug(1) }
        var savedDepsFileModTime = savedDepsFile.info.ModTime()
        for _, val := range files { if file, ok := toFile(val); !ok {
            // ignore
        } else if file.info.ModTime().After(savedDepsFileModTime) {
            files = nil // need to reload if outdated
            return
        }}
    }
    return
}

func traverseMissingDep(ctx Context, dep string) (res bool) {
    var (
        okay bool
        fullname string
        proj = ctx.project()
        pc = cast[*programContext](ctx)
    )
    if proj == nil {
        prompt(ctx, "%s: traverse dep failed, project %v\n", dep, proj)
        erro(ctx, "%s: no current project for dep", dep)
        errostack(ctx, 5, "%s: %v", dep, ctx).debug(10)
        return
    } else if file := proj.file(ctx, dep); file == nil {
        if false {
            // FIXME: traverse won't work with 'nil' target value
            var t = ctx.traverse(ctx, nil/*, dep*/)
            okay = !t.has(traveFail)
        } else {
            prompt(ctx, "%s: dep is unknown file; project %v\n", dep, proj)
            erro(ctx, "%v: %s is unknown file", proj, dep)
            errostack(ctx, 5, "(%T):", ctx).debug(24)
            panic(_failure(ctx, "dep '%s' is not file", dep))
        }
        fullname = dep
    } else {
        file.traverse(ctx)
        okay = !pc.traves.has(traveFail) && file.exists()
        fullname = file.fullname()
    }
    if pc.traves.has(traveCase, traveNext, traveDone) {
        pc.traves = pc.traves.not(traveCase, traveNext, traveDone)
        // TODO: for _, brk := range t { ... }
    }
    if pc.traves.has() {
        prompt(ctx, "%s: traverse dep failed (okay=%v), project %v\n", fullname, okay, proj)
        for _, brk := range pc.traves { erro(at(ctx,brk.pos), "%v: missing %v: %v", proj, dep, brk.what   ) }
        errostack(ctx, 5, "%v: %v", proj, ctx).debug(10)
    } else {
        res = okay
    }
    return
}

func traverseMissingDeps(ctx Context, lastTry string, errBytes []byte) (res bool, tried string) {
    const promptErrors bool = false
    const promptBeforeTraverse bool = promptErrors && true
    var pc = cast[*programContext](ctx)
    for _, m := range rxFatalErrorFileNotFound.FindAllSubmatch(errBytes, -1) {
        if promptBeforeTraverse { prompt(ctx, "%s\n", m[0]).debug(6) }
        if dep := string(m[4]); dep == lastTry {
            return false, ""
        } else if res = traverseMissingDep(ctx, dep); !res || pc.traves.has() {
            var (
                s, l, c = string(m[1]), string(m[2]), string(m[3])
                pos = convPosition(s, l, c)
            )
            prompt(ctx, "%s: dep missing, project %v\n", m[4], ctx.project())
            prompt(ctx, "%s\n", m[0]) // prompt the entire error line
            erro(at(ctx,pos), "%v", ctx).debug(1)
            return
        } else if tried == "" { tried = dep }
    }
    return
}

type modifier_deps struct { modifier_
    useClang bool `cl,clang`
    useGcc bool `gcc`
    addMissing bool `am,add-missing,mg,missing-goal`
    lang string `l,lang,language`
    flags []Value `f,flags,o,opts`
    cc string `cc,compiler`
}
func (ctx *modifier_deps) x(args ...Value) (result interface{}) {
    var uni = _universe(ctx)
    if uni.noDepsGrep || uni.noDeps { return }

    // NOTE: parse opts for (deps) before expanding the args, because we share args
    //       with the compilers!
    var (
        targetVal Value
        targetStr string
        err error
    )
    if targetVal, targetStr = getTargetValueString(ctx); isNull(targetVal) {
        erro(ctx, "target is nil").debug(1)
        return
    } else if targetStr == "" {
        erro(ctx, "target '%v' is empty", targetVal).debug(1)
        return
    }

    var files []Value
    if ctx.verbose {
        defer func(ts time.Time) {
            var s string
            if val := autoVal(ctx, "@"); val != nil { s = val.String() }
            prompt(ctx, "Deps %v …… (%d files in %v)\n", s, len(files), time.Now().Sub(ts)).debug(ctx.debug, 6)
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
            erro(ctx, "unsupported cc: %v", ctx.cc).debug(1)
            return
        } else if strings.HasPrefix(base, "clang") { ctx.useClang = true
        } else if strings.HasPrefix(base, "gcc")   { ctx.useGcc   = true }
    }

    var (
        flags = xmerge(ctx, ctx.flags...) //, plain
        _MM, _MG bool
        ca []string
    )
    for _, f := range flags {
        switch s := strings.TrimSpace(f.string(ctx)); s {
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
        var s, y = as{a}.fullnameOrFinal(ctx)
        if y { s = strings.TrimSpace(s) }
        switch s {
        case "", "-M", "-MM", "-MG", "-MD", "-MV", "-MP", "-Os", "-O1", "-O2", "-O3",
            "-c", "-shared", "-static", "-fPIC", "-fvisibility-inlines-hidden",
            "-fcxx-modules", "-fmodules", "-fmodules-ts":
            break // discard unused args
        default: ca = append(ca, s)
        }
    }

    var (
        proj = ctx.project()
        pc = cast[*programContext](ctx)
        savedDepsFileName string
    )

    if savedDepsFileName, files = loadSavedDepsAndCheckOutdated(ctx, ca); pc.traves.has() {
        for _, brk := range pc.traves { erro(at(ctx,brk.pos), "%v", brk) }
        errostack(ctx, 5, "%v: %v", proj, ctx).debug(16)
        return
    } else if len(files) == 0 {
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
            if okay, retried = traverseMissingDeps(ctx, retried, stderr.Bytes()); okay && !pc.traves.has() {
                cc = exec.Command(ctx.cc, ca...)
                stdout.Reset()
                stderr.Reset()
                goto retryCC
            }
            prompt(ctx, "%v: failed command '%s':\n", proj, ctx.cc)
            prompt(ctx, "%s \\\n  %s\n----------\n", cc.Path, strings.Join(ca, " \\\n  "))
            prompt(ctx, "%s\n----------\n%s----------\n", &stdout, &stderr)
            erro(ctx, "%s: %s deps failed: %v", proj, filepath.Base(ctx.cc), err)
            errostack(ctx, 5, "%s: %v", proj, ctx).debug(8)
            return
        }
        if stderr.Reset(); savedDepsFileName == "" {
            erro(ctx, "empty saved deps file name: %v", savedDepsFileName).debug(1)
            stdout.Reset(); return
        }

        var savedDepsFile *File = nil//stat(ctx, savedDepsFileName)
        if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, stdout.String()); len(files) == 0 {
            warn(ctx, "parse deps file failed").debug(1) // not saving if failed
        } else if err = os.MkdirAll(filepath.Dir(savedDepsFileName), os.FileMode(0755)); err != nil {
            erro(ctx, "make path '%s' failed: %v", filepath.Dir(savedDepsFileName), err).debug(1)
        } else if err = ioutil.WriteFile(savedDepsFileName, stdout.Bytes(), os.FileMode(0666)); err != nil {
            erro(ctx, "save deps file failed: %v", err).debug(1)
        } else if false {
            info(ctx, "saved deps %s", savedDepsFileName).debug(1)
        }
        stdout.Reset() // release buffers (optional)
    }
    if t := cast[*programContext](ctx); t != nil && len(files) > 0 {
        t.grepped = append(t.grepped, files...)
    }
    return
}

type modifier_touch struct { modifier_
    path bool `p,path`
    mode os.FileMode `m,mode`
}
func (ctx *modifier_touch) x(args ...Value) (result interface{}) {
    if len(args) == 0 { if val := autoVal(ctx, "@"); val != nil { args = append(args, val) }}

    var files []*File
    for _, arg := range args {
        var vf []*File
        if err := touch(ctx, arg, uint32(ctx.mode), ctx.path); err != nil {
            erro(ctx, "touch '%v' failed: %v", arg, err).debug(1)
            break
        } else if vf, err = arg.stamp(ctx); err != nil {
            erro(ctx, "touch '%v' failed: %v", arg, err).debug(1)
            break
        } else { files = append(files, vf...) }
    }

    var program = _program(ctx)
    if ctx.verbose { reportFileUpdates(ctx, files) }
    if len(program.getModifiers(ctx, "stamp")) > 0 {
        warn(ctx, "no need to use a (stamp) after (touch)").debug(1)
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

func (ctx *modifier_check) x(args ...Value) (result interface{}) {
    var (
        pos = ctx.Position()
        pc  = cast[*programContext](ctx)
        optBreak travekind // breaking with good results
        makeResult func(bool) Value // returns results only if non-nil
        values []Value
        res bool
    )

    if ctx.good   { optBreak   = traveDone }
    if ctx.answer { makeResult = func(v bool) Value { return makeAnswer(pos, v) } }
    if makeResult == nil && ( ctx.boolean ||
        (ctx.file != nil && (ctx.exists || ctx.regular || ctx.isdir)) ||
        (ctx.dir  != nil && (ctx.exists || ctx.regular || ctx.isdir)) ||
        (ctx.silent)) { makeResult = func(v bool) Value { return makeBoolean(pos, v) } }

    var checkFile = func (val Value, dir bool) {
        var ( s string; f *File )
        if v, y := val.(*boolean); y {
            if v.bool { val = autoVal(ctx, "@") } else { val = nil }
        }

        if val == nil {
            erro(ctx, "nil file value to check").debug(1)
            return
        } else if f, res = toFile(val); res {
            // best case
        } else if s = val.string(ctx); filepath.IsAbs(s) {
            if f = stat(at(ctx, val.Position()), s); f != nil { res = true }
        } else if f = file(ctx, s); f != nil { res = true }

        if f != nil {
            if !dir || ctx.regular { res = f.exists()
                if ctx.verbose { warnstack(at(ctx,val), 3, "check regular file '%v': %v", val, res).debug(1) }
            } else if dir || ctx.isdir { res = f.info != nil && f.info.Mode().IsDir()
                if ctx.verbose { warnstack(at(ctx,val), 3, "check dir '%v': %v", val, res).debug(1) }
            } else if ctx.exists { res = f.exists()
                if ctx.verbose { warnstack(at(ctx,val), 3, "check file exists '%v': %v", val, res).debug(1) }
            } else if ctx.verbose {
                warnstack(at(ctx,val), 3, "check file '%v': %v", val, res).debug(1)
            }
        } else if ctx.verbose {
            warnstack(at(ctx,val), 3, "check file '%v': %v", val, res).debug(1)
        }

        if makeResult != nil {
            values = append(values, makeResult(res))
        } else if !res {
            pc.traves.addf(ctx, optBreak, "'%v' is not file", val)
            return
        }
    }

    if ctx.file != nil { checkFile(ctx.file, false) }
    if ctx.dir  != nil { checkFile(ctx.dir, true) }

    var program = _program(ctx)
    var value = autoVal(ctx, "-")
ForPairs:
    for _, arg := range args {
        var p, y = arg.(*pair)
        if !y {
            if res = arg.true(ctx); makeResult != nil {
                values = append(values, makeResult(res))
            } else {
                pc.traves.addf(ctx, optBreak, "value '%v' is false", arg)
                if ctx.verbose { warn(ctx, "value '%v' is false", arg).debug(1) }
            }
            continue
        }

        var key, str string
        switch key = p.key.string(ctx); key {
        case "status":
            var exeres, _ = value.(*execResult)
            if exeres == nil {
                pc.traves.addf(ctx, optBreak, "value '%v' is not exec result", value)
                erro(at(ctx,value), "value '%v' (%T) is not exec result", value, value).debug(6)
                return
            } else { /*exeres.wg.Wait()*/ }

            var num, e = p.val.int(ctx)
            if e != nil {
                erro(ctx, "%v: %v", p.val, e).debug(1)
                return
            }
            if ctx.verbose {
                prompt(ctx, "checking status ")
                if num != 0 { prompt(ctx, "== %d ", num) }
                prompt(ctx, "…")
            }

            var good = exeres.Status == int(num)
            if ctx.verbose {
                var s string
                if good { s = "Yes" } else { s = "No" }
                prompt(ctx, "… %s (%d)\n", s, exeres.Status)
            }
            if ctx.debug>0 {
                var tar = autoVal(ctx, "@")
                var val = autoVal(ctx, "-")
                warn(at(ctx,program.position), "%v: %v", _entry(ctx), tar)
                warn(ctx, "status=%v", exeres.Status)
                warn(ctx, "hyphen=%v", val)
                warn(ctx, "context: %v", ctx).debug(ctx.debug)
            }

            if makeResult != nil {
                values = append(values, makeResult(good))
            } else if !good {
                pc.traves.addf(ctx, optBreak, "bad status (%v) (expects %v)", exeres.Status, p.val)
                break ForPairs
            }
        case "stdout", "stderr":
            var exeres, _ = value.(*execResult)
            if exeres == nil {
                pc.traves.addf(ctx, optBreak, "not an exec result (%T)", value)
                erro(at(ctx,value), "value '%v' (%T) is not exec result", value, value).debug(6)
                return
            } else { /*exeres.wg.Wait()*/ }

            if ctx.verbose {
                prompt(ctx, "checking %s (status=%d) … ", key, exeres.Status)
            }
            if ctx.debug>0 {
                var tar = autoVal(ctx, "@")
                var val = autoVal(ctx, "-")
                warn(at(ctx,program.position), "%v: %v", _entry(ctx), tar)
                warn(ctx, "status=%v", exeres.Status)
                warn(ctx, "hyphen=%v", val)
                warn(ctx, "context: %v", ctx).debug(ctx.debug)
            }

            var v *bytes.Buffer
            switch key {
            case "stdout": v = exeres.Stdout.Buf
            case "stderr": v = exeres.Stderr.Buf
            default: unreachable()
            }

            if v == nil {
                pc.traves.addf(ctx, optBreak, "bad %s (expects %v)", key, p.val)
                break ForPairs
            }

            str = p.val.string(ctx)
            if ctx.trim { str = strings.TrimSpace(str) }

            if res := v.String() == str; makeResult != nil {
                values = append(values, makeResult(res))
            } else if !res {
                pc.traves.addf(ctx, optBreak, "bad %s (%v) (expects %v)", key, v, p.val)
                break ForPairs
            }
        case "file", "dir": // file=xxx and dir=xxx, same as -file=xxx and -dir=xxx
            var ( f *File; res bool )
            if f, res = toFile(p.val); res {
                // ok
            } else if str = p.val.string(ctx); filepath.IsAbs(str) {
                if f = stat(at(ctx, p.val.Position()), str); f != nil {
                    // ok
                }
            } else if f = file(ctx, str); f != nil {
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
                pc.traves.addf(ctx, optBreak, "`%v` is not %s", p.val, key)
                break ForPairs
            }
        case "var":
            var g, ok = p.val.(*group)
            if !ok {
                pc.traves.addf(ctx, optBreak, "`%v` is not a group value", p.val)
                break ForPairs
            }
            for _, elem := range g.elems {
                switch p := elem.(type) {
                case *pair:
                    var a, b string
                    var k = p.key.string(ctx)
                    var def = program.project.scope.FindDef(k)
                    if def != nil {
                        a = p.val.string(ctx)
                        b = def.value.string(ctx)
                        if res := a != b; makeResult != nil {
                            values = append(values, makeResult(res))
                        } else if !res {
                            pc.traves.addf(ctx, optBreak, "`%v` != `%v`", p.key, p.val)
                            break ForPairs
                        }
                    } else if makeResult != nil {
                        values = append(values, makeResult(false))
                    } else {
                        pc.traves.addf(ctx, optBreak, "`%v` is not defined", k)
                        break ForPairs
                    }
                default:
                    pc.traves.addf(ctx, optBreak, "`%v` unsupported checks", elem)
                    break ForPairs
                }
            }
        default:
            erro(ctx, "unknown check for %v -> %v", p.key, p.val).debug(1)
            break ForPairs
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
    var def1 = autoDef(ctx, "1")
    var def2 = autoDef(ctx, "2")
    defer func(v1, v2 Value) { def1.value, def2.value = v1, v2 } (def1.value, def2.value)

    var pos = ctx.Position()
    def1.value = makeStrlit(pos, dst)
    def2.value = makeStrlit(pos, src)

    var head, foot string
    if opts.head != nil { head = opts.head.string(ctx) }
    if opts.foot != nil { foot = opts.foot.string(ctx) }

    // Compare mod time for update mode
    if opts.files += 1; opts.update {
        if st2, e := os.Stat(dst); e == nil && st2 != nil {
            var st1 os.FileInfo
            if st1, err = os.Stat(src); err != nil { erro(ctx, "%v", err); return }
            if st1 != nil && (st1.Size()+int64(len(head))+int64(len(foot))) == st2.Size() {
                if st2.ModTime().After(st1.ModTime()) { return }
            }
            if false { prompt(ctx, "%s: %s (%v,%v)\n", pos, dst, st1.Size(), st2.Size()) }
        }
    }

    var srcFile, dstFile *os.File
    if srcFile, err = os.Open(src); err != nil { erro(ctx, "%v", err); return } else {
        defer srcFile.Close()
    }

    // sys default file mode is 0666
    if opts.path { // Make path (mkdir -p)
        if p := filepath.Dir(dst); p != "." && p != "/" {
            err = os.MkdirAll(p, os.FileMode(0755))
            if err != nil { erro(ctx, "%v", err); return }
        }
    }

    if opts.mode == 0 { opts.mode = os.FileMode(0640) }

    dstFile, err = os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, opts.mode)
    if err != nil { erro(ctx, "%v", err); return } else { defer dstFile.Close() }

    srcBuf := bufio.NewReader(srcFile)
    dstBuf := bufio.NewWriter(dstFile)
    if head != "" {
        var n int
        if n, err = dstBuf.WriteString(head); err != nil { erro(ctx, "%v", err); return }
        opts.bytes += int64(n)
    }

    var n int64
    if n, err = io.Copy(dstBuf, srcBuf); err != nil { erro(ctx, "%v", err); } else {
        if opts.bytes += n; foot != "" {
            var n int
            if n, err = dstBuf.WriteString(foot); err != nil { erro(ctx, "%v", err); return }
            opts.bytes += int64(n)
        }
        if err == nil {
            dstBuf.Flush() // flush content
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
func (ctx *modifier_copyfile) x(args ...Value) (result interface{}) {
    var target Value
    var source Value
    if len(args) > 0 {
        target = args[0]
    } else {
        target = autoVal(ctx, "@")
    }
    if len(args) > 1 {
        source = args[1]
    } else {
        source = autoVal(ctx, "<")
    }

    // Get target filename
    var (
        project = ctx.project()
        filename, srcname string
        filetime, srctime time.Time
    )
    switch tv := target.(type) {
    case *File:
        if filename = tv.fullname(); tv.info != nil {
            filetime = tv.info.ModTime()
        }
    default:
        filename = target.string(ctx)
        if file := project.file(ctx, filename); file != nil {
            target, filename = file, file.fullname()
            if file.info != nil {
                filetime = file.info.ModTime()
            }
        }
    }
    switch tv := source.(type) {
    case *File:
        if srcname = tv.fullname(); tv.info != nil {
            srctime = tv.info.ModTime()
        }
    default:
        srcname = source.string(ctx)
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
            if !ctx.silent { erro(ctx, "file already existed (%s)", target).debug(1) }
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
        var file = stat(ctx,filename, stat_nonexist{true})
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
    var file *File
    if file = stat(ctx,srcname, stat_nonexist{true}); file == nil || file.info == nil {
        erro(ctx, "'%s' source file not found", srcname).debug(1)
    } else if !file.info.IsDir() {
        if ctx.mode == 0 { ctx.mode = file.info.Mode() }
        if err := copyFile(ctx, file.info, srcname, filename, copts); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    } else if ctx.recursive {
        if err := copyDir(ctx, srcname, filename, copts); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    } else {
        erro(ctx, "`%v` is a directory (use -r to solve it)", source).debug(1)
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
func (ctx *modifier_writefile) x(args ...Value) (result interface{}) {
    args = xmerge(ctx, args...) //, plain

    var (
        target = autoVal(ctx, "@")
        filename, str string
        f *os.File
    )
    if target == nil {
        erro(ctx, "target is undefined").debug(1)
        return
    }

    defer func() {
        if filename != "" { os.Remove(filename); f = nil }
        if f == nil {
            var pc = cast[*programContext](ctx)
            brk := pc.traves.add(ctx, traveFail, target)
            brk.error = fmt.Errorf("file %s not generated", target)
        }
    } ()

    filename, _ = as{target}.fullnameOrFinal(ctx)

    if h := autoVal(ctx, "-"); h == nil {
        erro(ctx, "buffer value is nil").debug(1)
        return
    } else {
        str = h.string(ctx)
    }

    var err error
    if f, err = os.Create(filename); err != nil {
        erro(ctx, "%v", err).debug(1)
        return
    } else if _, err = f.WriteString(str); err != nil {
        f.Close()
        erro(ctx, "%v", err).debug(1)
        return
    } else {
        result = stat(ctx, filename)
        f.Close()
    }
    return
}

type modifier_readfile struct { modifier_
    head Value "h,head"
    foot Value "f,foot"
}
func (ctx *modifier_readfile) x(args ...Value) (result interface{}) {
    var (
        filename string
        file *File
        target as
    )
    if n := len(args); n > 1 {
        erro(ctx, "too many files: %v", args).debug(1)
        return
    } else if n == 1 {
        target.Value = args[0]
    } else {
        target.Value = autoVal(ctx, "@")
    }

    var pc = cast[*programContext](ctx)
    if isTrivial(target) {
        errostack(ctx, 3, "target for reading is invalid (%T) (%v)", target.Value, args).debug(10)
        return
    } else if file, filename, _ = target.fullnameFile(ctx); file == nil {
        if val := autoVal(ctx, ">"); val != nil {
            s := pc.traves.add(ctx, traveFail, target.Value)
            s.error = traveTargetNotDefinedFile
            s.depend = val
        } else if true {
            erro(ctx, "not a file: %v (%T)", target.Value, target.Value)
            errostack(ctx, 8).debug(64)
        }
        return
    } else if filename == "" {
        errostack(at(ctx,target), 3, "%v: empty fullname", target).debug(32)
        return
    }

    var ( bytes []byte ; err error )
    if bytes, err = ioutil.ReadFile(filename); err == nil {
        var s string
        if ctx.head != nil { s = ctx.head.string(ctx) }
        if len(bytes) > 0   { s += string(bytes) }
        if ctx.foot != nil { s = ctx.foot.string(ctx) }
        autoSet(ctx.Context, "-", makeStrlit(ctx.Position(), s))
        autoSet(ctx.Context, "-file", file)
    } else {
        brk := pc.traves.add(ctx, traveFail, target)
        brk.error = err
    }
    if ctx.debug>0 && err != nil {
        warn(ctx, "%v: %v ; stems=%v\n", target, err, _stems(ctx))
        warnstack(ctx, 5).debug(ctx.debug)
    }
    return
}

func crc64CheckFileModeContent(ctx Context, filename string, content []byte, perm os.FileMode) (same bool, err error) {
    var f *os.File
    if f, err = os.Open(filename); err == nil && f != nil {
        defer f.Close()

        if perm != 0 {
            if s, _ := f.Stat(); s.Mode().Perm() != perm {
                if err = f.Chmod(perm); err != nil { return }
            }
        }

        w1 := crc64.New(crc64Table)
        w2 := crc64.New(crc64Table)
        if _, err = io.Copy(w1, f); err != nil { return }
        if _, err = w2.Write(content); err != nil { return }
        var a, b = w1.Sum64(), w2.Sum64()
        if a == b { same = true }

        if false {
            var s []byte
            if s, err = ioutil.ReadFile(filename); err != nil { return }
            prompt(ctx, "crc64CheckFileModeContent: %v %v\n%s\n%s\n", a, b, s, content)
        }
    }
    return
}

func crc64CompareFileChecksum(ctx Context, filename1, filename2 string) (same bool, err error) {
    var s []byte
    if s, err = ioutil.ReadFile(filename1); err != nil {
        erro(ctx, "%v", err).debug(1)
        return
    }
    return crc64CheckFileModeContent(ctx, filename2, s, 0)
}

type modifier_updatefile struct { modifier_
    verbFilename bool `vf,verbfile,verb-filename`
    path   bool `p,path,md,makedir,make-dir,mp,makepath,make-path`
    zero   bool `z,zero;e,empty;az,allow-zero;ae,allow-empty`
    keep   bool `k,keep,keep-file`
    append bool `a,app,append,append-content`
    mode os.FileMode "m,mode"
}
func (ctx *modifier_updatefile) x(args ...Value) (result interface{}) {
    assert(ctx.mode != 0, "zero file mode")

    defer trace(ctx)

    var target as
    var filename string
    if len(args) > 0 { target.Value = args[0] }
    if len(args) > 1 { if false { ctx.mode = filePerm(ctx, args[1], 0600) }}

    if isTrivial(target) { target.Value = autoVal(ctx, "@") }
    if isTrivial(target) {
        erro(ctx, "no file target to update").debug(1)
        return
    } else if f, y := target.Value.(*File); y {
        filename = f.fullname()
    } else if o, y := target.fullname(ctx); !y {
        erro(ctx, "update-file: not a file (%T: %v)\n", target.Value, target).debug(1)
        return
    } else {
        filename = o.string(ctx)
    }

    if filename == "" {
        erro(ctx, "update-file: empty fullname (%T %v)\n", target.Value, target).debug(1)
        return
    }

    if ctx.debug > 0 {
        warnstack(ctx, 5, "update-file: %v (fullname=%v, project=%v)",
            target, filename, ctx.project()).debug(ctx.debug)
    }

    if ctx.path { // Make path (mkdir -p)
        if p := filepath.Dir(filename); p != "." && p != "/" {
            if fi, _ := os.Stat(p); fi != nil && !fi.IsDir() {
                if e := os.Remove(p); e != nil {
                    erro(ctx, "%v (%T %v)", e, target, target).debug(1)
                    return
                }
            }
            if e := os.MkdirAll(p, os.FileMode(0755)); e != nil {
                if proj := ctx.project(); proj != nil {
                    info(ctx, "%v: %v %v", filename, proj, unmapfiles(ctx, filename))
                    info(ctx, "%v: %v %v", filename, proj, proj.file(ctx, filename))
                    erro(ctx, "%v: %v (%T %v)", filename, e, target, target).debug(1)
                }
                return
            }
        }
    }

    // Check existed file content checksum
    var (
        content string
        exeres *execResult
    )
    if val := autoVal(ctx, "-"); val == nil {
        // no buffer value
    } else if content = val.string(ctx); false && strings.Contains(content, `"\"`) {
        prompt(ctx, "%v: %T\n", filename, val).debug(1)
        panic(_failure(ctx, "%s", filename))
    } else {
        exeres, _ = val.(*execResult)
    }

    if content != "" {
        // good to go
    } else if ctx.zero {
        if ctx.verbose || ctx.debug > 0 {
            warnstack(ctx, 3, "empty content for '%v'", target).debug(ctx.debug)
        }
    } else {
        if ctx.keep {
            // keep file
        } else if file := stat(at(ctx, target.Position()), filename); file != nil && file.info != nil && file.info.Size() == 0 {
            file.info = nil
            if err := os.Remove(filename); err != nil {
                erro(ctx, "remove file failed: %v", err).debug(1)
            }
        }
        if exeres != nil {
            if exeres.Stdout.log != nil {
                var pos Position
                pos.Filename = exeres.Stdout.log.filename
                pos.Line = exeres.Stdout.log.lines + 1
                erro(at(ctx,pos), "empty stdout")
            }
            if exeres.Stderr.log != nil && exeres.Stdout.log != exeres.Stderr.log {
                var pos Position
                pos.Filename = exeres.Stderr.log.filename
                pos.Line = exeres.Stderr.log.lines + 1
                erro(at(ctx,pos), "empty stderr")
            }
        }

        if v := autoVal(ctx, "-"); v == nil {
            prompt(ctx, "%s:1: empty content\n", filename).debug(1)
        } else {
            prompt(ctx, "%s:1: empty content: %v\n", filename, v).debug(1)
        }
        erro(ctx, "empty content for '%v'", target)
        errostack(ctx, 6).debug(64)
        return
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
            prompt(ctx, "update %v …… %s (in %v)\n", f, s, time.Now().Sub(st)).debug(ctx.debug)
        } (time.Now())
    }

    if same, err = crc64CheckFileModeContent(ctx, filename, []byte(content), ctx.mode); err != nil {
        if _, ok := err.(*os.PathError); ok {
            err = nil // discard path error (e.g. no such file or directory)
        } else {
            erro(ctx, "crc64 checksum failed: %v", err).debug(1)
            return
        }
    } else if same {
        //removeCallerUpdated(ctx, target) // remove timestamp updated
        result = stat(ctx, filename)
        return
    }

    // if false { promptEnteringDirectory(ctx) }

    // Create or update the file with new content

    var (
        f *os.File
        pc = cast[*programContext](ctx)
        m = os.O_RDWR | os.O_CREATE
    )
    if ctx.append { m |= os.O_APPEND } else { m |= os.O_TRUNC }
    if f, err = os.OpenFile(filename, m, ctx.mode); err != nil {
        brk := pc.traves.add(ctx, traveFail, target)
        brk.error = fmt.Errorf("update %v failed", target)
        erro(ctx, "open file failed: %v", err).debug(1)
    } else if f != nil {
        defer func() {
            if err = f.Close(); err != nil {
                os.Remove(filename)
                erro(ctx, "close file '%s' failed: %v", filename, err).debug(1)
                return
            }

            if file := stat(ctx, filename); file == nil {
                prompt(ctx, "%s: invalid file\n", filename)
                errostack(ctx, 6, "%v: invalid file '%s'", ctx.project(), filename).debug(1)
                panic(_failure(ctx, "invalid file %s", filename))
            } else {
                var files []*File
                if files, err = file.stamp(ctx); err != nil {
                    erro(ctx, "%v", err).debug(1)
                    return
                } else if false && ctx.verbose {
                    reportFileUpdates(ctx, files)
                }
                result = file // resulting the updated file
            }
        } ()
        if wrote, err = f.WriteString(content); err != nil {
            erro(ctx, "write content failed: %v", err).debug(1)
        }
    } else {
        brk := pc.traves.add(ctx, traveFail, target)
        brk.error = fmt.Errorf("%v not updated", target)
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
func (ctx *modifier_wait) x(args ...Value) (result interface{}) {
    var (
        waitForexecResult = ctx.stdout || ctx.stderr || ctx.status || ctx.execRes
        stampCurrentTarget = !ctx.noTarget
        target Value = autoVal(ctx, "@")
        execRes *execResult
        err error
    )
    if ctx.verbose {
        defer func (st time.Time) {
            var s string; if err != nil { s = "fail" } else { s = "done" }
            prompt(ctx, "Wait %v …… %s, result=%v\n", target, s, execRes).debug(ctx.debug, 1)
            if ctx.debug>0 { info(ctx, "%v", execRes).debug(ctx.debug) }
        } (time.Now())
    }

    // Wait for prerequisites and/or execution
    if _, _, execRes, err = wait(ctx, waitOpts{
        ctx.verbose, waitForexecResult, stampCurrentTarget,
    }); execRes == nil { return }

    var (
        pos = ctx.Position()
        a []Value
        s string
        v Value
    )
    if ctx.stdout {
        // TODO: warn(ctx, "deprecated (wait -stdout), use (shell -stdout) instead; %v", execRes).debug(1)
        if b := execRes.Stdout.Buf; b != nil { s = b.String() }
        if ctx.trim { s = strings.TrimSpace(s) }
        switch ctx.asType {
        case "answer": v = makeAnswer (pos,(s == "yes"))
        case "bool":   v = makeBoolean(pos,(s == "true"))
        default:       v = makeStrlit (pos,s)
        }
        a = append(a, v)
    }
    if ctx.stderr {
        // TODO: warn(ctx, "deprecated (wait -stderr), use (shell -stderr) instead; %v", execRes).debug(1)
        if b := execRes.Stderr.Buf; b != nil { s = b.String() }
        if ctx.trim { s = strings.TrimSpace(s) }
        switch ctx.asType {
        case "answer": v = makeAnswer (pos,(s == "yes"))
        case "bool":   v = makeBoolean(pos,(s == "true"))
        default:       v = makeStrlit (pos,s)
        }
        a = append(a, v)
    }
    if ctx.status {
        // TODO: warn(ctx, "deprecated (wait -status), use (shell -status) instead; %v", execRes).debug(1)
        a = append(a, makeDecimal(pos,int64(execRes.Status)))
    }

    if len(a) > 0 { result = ease(ctx, a) }
    return
}

func reportFileUpdates(ctx Context, files []*File) {
    var start = cast[*programContext](ctx).start
    for _, file := range files {
        var (
            mod = file.info.ModTime()
            d = time.Now().Sub(start)
        )
        if mod.After(start) {
            prompt(ctx, "Updated %v (%v)\n", file, d)
        } else {
            prompt(ctx, "File %v not changed (%v, ModTime=%v)\n", file, d, mod)
            warn(ctx, "incorrect timestamp: %v (JobTime=%v, ModTime=%v)", file, start, mod)
            warn(ctx, "the target path name is: %v", file.fullname())
            warn(ctx, "try 'touch' the target %v if the path name and command are correct", file)
            info(ctx, "you may ignore the warnings if all correct")
        }
    }
}

type modifier_stamp struct { modifier_
    prompt bool "m,prompt"
    next   bool "n,nxt,next"  // traveNext if failed to stamp
    error  bool "e,err,error" // traveErro if failed to stamp
}
func (ctx *modifier_stamp) x(args ...Value) (result interface{}) {
    var target = getTargetValue(ctx)
    if isNull(target) {
        prompt(ctx, "%v\n", ctx.project())
        erro(ctx, "stamp(%v) failed", target)
        errostack(ctx, 6, "%v", ctx).debug(12)
        return
    }

    var _, err = target.stamp(ctx)
    if err == nil { return /* Done! */ }

    var pc = cast[*programContext](ctx)
    var p = prompt(ctx, "%v: %v: %v\n", target, ctx.project(), err)
    if n := ctx.debug; n>0 { p.debug(n) }
    if ctx.next {
        if ctx.verbose { warn(ctx, "%v", err).debug(1) }
        s := pc.traves.add(ctx, traveNext, target)
        s.depend = autoVal(ctx, ">")
        err = nil // discard the error
    } else if ctx.error {
        s := pc.traves.add(ctx, traveFail, target)
        s.depend = autoVal(ctx, ">")
        s.error = err
        if false {
            prompt(ctx, "%v: %v: %v\n", target, ctx.project(), err).debug(1)
            erro(ctx, "stamp(%v) error")
            errostack(ctx, 10, "%v", ctx).debug(1)
        } else {
            prompt(ctx, "%v: %v: %v\n", target, ctx.project(), err).debug(1)
            warn(ctx, "stamp(%v) error")
            warnstack(ctx, 10, "%v", ctx).debug(1)
        }
    } else {
        if f, y := target.(*File); y {
            erro(ctx, "failed stamp(%v): %v %v", target, f.fullname(), f.info)
        } else {
            erro(ctx, "failed stamp(%v) (%T)", target, target)
        }
        errostack(ctx, 10, "failed: %v", ctx).debug(10)
    }

    if err != nil { if pe, ok := err.(*fs.PathError); ok {
        erro(ctx, "stamp %s: %v", trimPromptString(pe.Path), pe.Err)
        err = pe.Err
    }}
    return
}

type modifier_assert struct { modifier_
    msg string `msg,message`
}
func (ctx *modifier_assert) v(args ...Value) (_ interface{}) { ctx.z(args...) ; return }
func (ctx *modifier_assert) x(args ...Value) (_ interface{}) { ctx.z(args...) ; return }
func (ctx *modifier_assert) z(args ...Value) (_ interface{}) {
    defer trace(ctx)

    var uni = _universe(ctx)
    var target = autoVal(ctx, "@")
    var pc = cast[*programContext](ctx)
    for _, a := range args {
        if a == nil { erro(ctx, "assert: nil").debug(1); return }
        if _, y := a.(*punctuation); y { continue }

        v := a.expand(ctx) //, final|w
        b := v.true(ctx)

        if (uni.hooks.assert != nil && uni.hooks.assert(ctx, v, b)) || b {
            continue
        } else if s := ctx.msg; s == "" {
            erro(at(ctx,a), "assert: %v → %v → '%s'", us(a), us(v), v.string(ctx)).debug(1)
        } else {
            erro(at(ctx,a), "assert: %v → %v: %s", us(a), us(v), s).debug(1)
        }

        pc.traves.add(ctx, traveFail, target).error = fmt.Errorf("assert(%v)", a)
    }
    return
}

type modifier_cond struct { modifier_ }
func (ctx *modifier_cond) x(args ...Value) (result interface{}) {
    // TODO: make it lisp-like (cond), e.g.:
    //     (cond
    //       ((condition) ...)
    //       (true{} ...))
    var pc = cast[*programContext](ctx)
    for _, a := range args {
        if a == nil { warn(ctx, "nil arg").debug(1) }
        if a == nil || !a.true(ctx.Context) {
            pc.traves.add(ctx, traveDone, nil)
            return
        }
    }
    return makeBoolean(ctx.Position(), true)
}

type modifier_case struct { modifier_ }
func (ctx *modifier_case) x(args ...Value) (result interface{}) {
    var w travekind = traveNext
    for _, a := range args { if a.true(ctx.Context) { w = traveCase ; break } }

    var pc = cast[*programContext](ctx)
    var s = pc.traves.add(ctx, w, nil) // trave 'case' or 'next'
    s.prog = _program(ctx) // s.error = fmt.Errorf("%s", msg)

    if ctx.verbose { prompt(ctx, "%v: %v", autoVal(ctx, "@"), w) }
    if ctx.debug > 0 { warn(ctx, "%v", w) }
    return
}

type modifier_predictDirty struct { modifier_ }
func (ctx *modifier_predictDirty) x(args ...Value) (result interface{}) {
    if res := ctx.dirty(ctx, args...); res {
        result = makePrediction(ctx.Position(), res, /*reason*/"")
    } else {
        var pc = cast[*programContext](ctx)
        pc.traves.add(ctx, traveDone, nil)
    }
    return
}

type modifier_predictNoLoop struct { modifier_ }
func (ctx *modifier_predictNoLoop) x(args ...Value) (result interface{}) {
    var loop bool
    var target = autoVal(ctx, "@")
    for caller := cast[*programContext](ctx).caller(); caller != nil; caller = caller.caller() {
        var t = autoVal(caller, "@")
        var same = t != nil && target == t
        if!same && false { same = eq(ctx, target, t) }
        if same {
            //fmt.Printf("%s: loop: %v\n", pos, ctx.def.target.value)
            loop = true
            break
        }
    }

    var s string
    if !loop { s = "not " }
    s = fmt.Sprintf("loop %sdetected (%v)", s, target)
    result = makePrediction(ctx.Position(), !loop, s)
    return
}

type modifier_predictTarget1stVisit struct { modifier_
    silent bool "s,silent"
}
func (ctx *modifier_predictTarget1stVisit) x(args ...Value) (result interface{}) {
    var target = autoVal(ctx, "@")
    if isNull(target) {
        erro(ctx, "target is <nil>").debug(1)
        return
    }

    var num int
    for caller := cast[*programContext](ctx).caller(); caller != nil; caller = caller.caller() {
        if false {
            var t = autoVal(caller, "@")
            var same = t != nil && target == t
            if !same && false { same = eq(ctx, target, t) }
            if same { num += 1 }
        } else if n := caller.execRec[target]; n > 0 {
            num += n
        }
    }

    var s string
    ;      if ctx.silent {
    } else if num == 0  { //s = "zero"
    } else { s = fmt.Sprintf("%v visits", num+1)
    }

    result = makePrediction(ctx.Position(), num==0, s)
    return
}

type modifier_predictTargetMaxVisit struct { modifier_
    clo bool "c,closure"
}
func (ctx *modifier_predictTargetMaxVisit) x(args ...Value) (result interface{}) {
    var nth int64
    for _, a := range args {
        if i, e := a.int(ctx); e != nil {
            erro(ctx, "%v: %v", a, i).debug(1)
        } else if nth = i; nth <= 0 {
            erro(ctx, "needs positive number (%v, %s)", a, typeof(a)).debug(1)
            return
        }
    }

    var (
        num int64
        head bool = true
        target = autoVal(ctx, "@")
    )
    if isNull(target) {
        erro(ctx, "target is <nil>").debug(1)
        return
    }
    for caller := cast[*programContext](ctx).caller(); caller != nil; caller = caller.caller() {
        var ct = autoVal(caller, "@")
        if n := caller.execRec[target]; n > 0 { num += int64(n) }
        if ctx.debug > 0 && num > 0 {
            if head { head = false
                prompt(ctx, "  %s: nth(%d)\n", ctx.Position(), nth)
            }
            var pos = _program(caller).position
            prompt(ctx, "    %s: %v\n", pos, ct)
        }
    }

    var s string;
    ;      if ctx.silent {
    } else if num == 0  { //s = "nth: zero"
    } else if num < nth { //s = "nth"
    } else { s = fmt.Sprintf("%d visits", num+1) }

    result = makePrediction(ctx.Position(), num<nth, s)
    return
}

type modifier_fork struct { modifier_
    wd string `w,wd,workdir,work-dir`
}
func (ctx *modifier_fork) _x(args ...Value) (result Value, traves travestates) {
    var (
        attr syscall.ProcAttr
        argv []string
        prog = _program(ctx)
    )
    for _, a := range args { argv = append(argv, a.string(ctx)) }

    if ctx.wd != "" {
        attr.Dir = ctx.wd
    } else if attr.Dir = prog.workdir(ctx); attr.Dir == "" {
        erro(ctx, "empty workdir").debug(1)
        return
    }

    attr.Env, _ = cast[*programContext](ctx).env(ctx)
    attr.Files = []uintptr{ // FIXME: see Cmd.Start() for files pipes
        os.Stdin .Fd(),
        os.Stdout.Fd(),
        os.Stderr.Fd(),
    }

    if exe, err := os.Executable(); err != nil {
        erro(ctx, "fork: %v: %v", os.Args[0], err).debug(1)
    } else if pid, err := syscall.ForkExec(exe, argv, &attr); err != nil {
        erro(ctx, "fork: %v: %v", exe, err).debug(1)
    } else if pid == 0 {
        erro(ctx, "fork: pid is zero").debug(1)
    } else {
        // TODO: status code, etc.
    }
    return
}
func (ctx *modifier_fork) x(args ...Value) (result interface{}) {
    var (
        prog = _program(ctx)
        argv []string
        wd string
    )
    for _, a := range args { argv = append(argv, a.string(ctx)) }

    if ctx.wd != "" {
        wd = ctx.wd
    } else if wd = prog.workdir(ctx); wd == "" {
        erro(ctx, "empty workdir").debug(1)
        return
    }

    var exe, err = os.Executable()
    if err != nil {
        erro(ctx, "fork: %v: %v", os.Args[0], err).debug(1)
        return
    }

    var cmd = exec.Command(exe, argv...)
    cmd.Stdout, cmd.Stderr = stdout, stderr
    cmd.Env, _ = cast[*programContext](ctx).env(ctx)

    if err = cmd.Run(); err != nil {
        erro(ctx, "fork: %v: %v", exe, err).debug(1)
    } else {
        // TODO: status code, etc.
    }
    return
}

type modifier_gitmodified struct { modifier_ }
func (ctx *modifier_gitmodified) x(args ...Value) (result interface{}) {
    var out = new(bytes.Buffer)
    var git = exec.Command("git", "status")
    git.Stdout, git.Stderr = out, os.Stderr
    if err := git.Run(); err != nil {
        erro(ctx, "git failed: %v", err).debug(1)
        return
    }

    // TODO: check also for `Changes not staged for commit:`

    var rx = regexp.MustCompile(`\n\tmodified:[\ctx ]*(.+?)\n`)
    var sm = rx.FindAllSubmatch(out.Bytes(), -1)
    if len(sm) > 0 {
        var pos = ctx.Position()
        var pred = makePrediction(pos, false, "")
        if result = pred; len(args) == 0 {
            pred.bool, pred.s = true, "modified"
            return
        }
        for _, a := range args {
            var s = a.string(ctx)
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
func (ctx *modifier_gitahead) x(args ...Value) (result interface{}) {
    var out = new(bytes.Buffer)
    var git = exec.Command("git", "status")
    git.Stdout, git.Stderr = out, os.Stderr
    if err := git.Run(); err != nil {
        erro(ctx, "git: %v", err).debug(1)
        return
    }

    // TODO: check also for `Changes not staged for commit:`

    var rx = regexp.MustCompile(`\nYour branch is ahead of '(.+?)' by`)
    var sm = rx.FindAllSubmatch(out.Bytes(), 1)
    if len(sm) > 0 {
        result = makePrediction(ctx.Position(), true, "Work branch has new commits to push")
    }
    return
}

var (
    onceMutex sync.Mutex
    onceCache0 map[entry]map[Value]int
    onceCache1 map[*program]map[Value]int
    onceSHA256Mutex sync.Mutex
    onceSHA256Cache = make(map[hashBytes]int,64)
)

func onceCacheTest0(ctx Context, target Value) (n int) {
    var rec map[Value]int
    var ent = _entry(ctx)
    if x, y := ent.(*stemmed); y { ent = x.rule }

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
    if stemmed, ok := entry.(*stemmed); ok {
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
        if f, ok := toFile(t); ok {
            fmt.Fprintf(h, "%s", f.fullname())
        } else {
            fmt.Fprintf(h, "%s", t.string(ctx))
        }
    }

    var sum hashBytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

func onceSHA256Test(ctx Context, sum hashBytes) (n int) {
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
    if stemmed, ok := entry.(*stemmed); ok {
        entry = stemmed.rule
    }

    if true {
        // // NOTE: entry and program are unique, since (once) is for runtime, we use their addresses.
        // fmt.Fprintf(h, "%p%p", entry, program)
        fmt.Fprintf(h, "%T%p%p", entry, entry, program)
    } else {
        fmt.Fprintf(h, "%v%v", ctx.Position(), program.position)
    }

    var a as
    for _, a.Value = range args {
        s, _ := a.fullnameOrFinal(ctx)
        fmt.Fprintf(h, "%s", s)
    }

    var sum hashBytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

type modifier_once struct { modifier_
    checksum bool `c,cs,checksum,s,sha,sha256,sum,h,hash`
    forval Value `for` // TODO: (once -for=$@)
}
func (ctx *modifier_once) x(args ...Value) (result interface{}) {
    // TODO: (once)           --> once for the Rule, aka entry.doneOnce = true
    // TODO: (once -for=$@)   --> once for $@, aka entry.onces[$(expand $@)] = true
    var (
        target Value = autoVal(ctx, "@")
        n int
    )

    const onceAlgo = 2 // avaialbe: 0, 1, 2

    if isTrivial(target) {
        errostack(ctx, 5, "once: no target $@, %v", args).debug(16)
        return
    } else if ctx.checksum {
        n = onceSHA256(ctx, target, append([]Value{target}, args...)...)
    } else if onceAlgo == 2 {
        n = onceCacheTest2(ctx, target)
    } else if onceAlgo == 1 {
        n = onceCacheTest1(ctx, target)
    } else {
        n = onceCacheTest0(ctx, target)
    }

    var pc = cast[*programContext](ctx)
    if n > 1 {
        s := pc.traves.add(ctx, traveDone, target)
        s.error = fmt.Errorf(`executed %d times`, n)
    }

    if ctx.debug > 0 {
        warn(ctx, "%T %v %p %v", target, target, target, n)
        warnstack(at(ctx, target.Position()), -1, "%p %v %v", target, target, n).debug(16)
    }

    // TODO: new once algorithm:
    if false {
        type traverseRec struct {
            targets map[Value]int // prerequisites
        }

        var ent = _entry(ctx)
        var traverseMap = make(map[entry]*traverseRec)

        if rec, _ := traverseMap[ent]; false {
            if rec == nil {
                rec = &traverseRec{ make(map[Value]int) }
                traverseMap[ent] = rec
            }
            // TODO: once: if rec.prerequisites[]
            if rec.targets[target] += 1; rec.targets[target] > 1 {
                n := rec.targets[target]
                s := pc.traves.add(ctx, traveDone, target)
                s.error = fmt.Errorf(`executed %d times`, n)
            }
        }
    }
    return
}
