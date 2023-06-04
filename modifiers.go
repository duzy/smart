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
    "regexp"
    "strings"
    "sync"
    "syscall"
    "time"
)

var launchTime = time.Now()

const (
    TheShellStatusDef = "shell→status" // status code of execution
)

type generalOpts struct {
    stackNum int  `sn,stack,stacknum,stack-num,stack-number`
    fullname bool `f,fn,fu,ful,full,fullname,full-name`
    debug    int  `d,db,dbg,debug` // NOTE: compatible with 'bool'
    warn     bool `w,warn,warning`
    verbose  bool `v,verb,verbose`
    timing   bool `t,time,timing`
}

type modification struct {
    valbase
    name Value
    args []Value
}
func (m *modification) refs(ctx Context, v Value) bool {
    if m.name.refs(ctx, v) { return true }
    for _, a := range m.args {
        if a.refs(ctx, v) { return true }
    }
    return false
}
func (m *modification) expandable(ctx Context, w facet) (res bool) {
    if res = m.name.expandable(ctx, w); !res {
        for _, a := range m.args {
            if res = a.expandable(ctx, w); res { break }
        }
    }
    return
}
func (m *modification) expand(ctx Context, _ facet) (Value) { return m }
func (_ *modification) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*modification); ok { res = cmpEqual }
    return
}
func (m *modification) traverse(ctx Context) {
    ctx = at(ctx, m.position)

    var (
        pc   = ctx.pc()
        prog = ctx.program()
        proj = ctx.Project()
        name = m.name.Strval(ctx)
    )
    defer func(t0 time.Time) {
        var n time.Duration = 1
        if name == "shell" || name == "sh" { n = 10 }

        var t2 = time.Now()
        if d := t2.Sub(t0); d > n*time.Second {
            var pos = ctx.Position()
            prompt(ctx, "%v: slow: %v ⇒ %v\n", pos, m, d).debug(1)
        }
    } (time.Now())

    if name == "" {
        erro(of(ctx,m.name), "modifier name '%v' is empty", m.name).debug(1)
        return
    }

    // Special modifier processing (implicit interpretation) before (configure)
    if len(pc.interpreted) == 0 && len(prog.recipes) > 0 && name == "configure" {
        // Evaluate for configure modifier
        if i, ok := dialects["eval"]; ok && i != nil {
            if err := pc.interpret(ctx, i, m.args); err != nil {
                var _, ent, _ = entryIndicator(ctx, ctx.entry())
                prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                erro(ctx, "interpret failed: %v", err)
                errostack(ctx, 3, "%v", ctx).debug(1)
                return
            }
        }
    }

    if i, _ := dialects[name]; i != nil {
        if err := pc.interpret(ctx, i, m.args); err != nil {
            var _, ent, _ = entryIndicator(ctx, ctx.entry())
            prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
            erro(ctx, "%s: %v", name, err)
            errostack(ctx, 3, "(%T):", ctx).debug(1)
            fail(m.Position(), "%s failed for project %s", name, proj)
        }
        return
    }

    var mod = modifier{at(ctx, m.position)}
    if value := mod.apply(name, m.args...); pc.traves.has(traveFail) {
        if true && value != nil { warn(ctx, "%v %v", value, pc.traves).debug(1) }

        if t := pc.traves.not(traveCase, traveNext, traveDone); false && t.has() {
            if options.verbose || options.verboseBreaks {
                var _, ent, _ = entryIndicator(ctx, ctx.entry())
                prompt(ctx, "%v: %s failed for %s\n", ent, name, proj)
                for _, s := range t { warn(at(ctx,s.pos), "%v: %s: %v", proj, name, s) }
                warnstack(ctx, 5, "").debug(16)
            }
        }
    } else if h := autoGet(ctx,"-"); value != nil && value != h {
        ctx.autoSet("-", value)
    }

    if n := ctx.flushDiags(true); n > 0 {
        brk := pc.traves.add(ctx, traveFail, nil)
        brk.error = fmt.Errorf("%s: %d errors counted", m.name, n)
    }
}
func (m *modification) String() (s string) {
    s = "(" + m.name.String()
    for _, a := range m.args {
        s += " " + a.String()
    }
    s += ")"
    return
}
func (p *modification) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    erro(ctx, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}

type modifications struct {
    valbase
    list []*modification
}
func (g *modifications) refs(ctx Context, v Value) (res bool) {
    for _, m := range g.list {
        if m.refs(ctx, v) { res = true; break }
    }
    return
}
func (g *modifications) expandable(ctx Context, w facet) (res bool) {
    for _, m := range g.list {
        if res = m.expandable(ctx, w); res { break }
    }
    return
}
func (g *modifications) expand(ctx Context, _ facet) (Value) { return g }
func (_ *modifications) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*modifications); ok { res = cmpEqual }
    return
}
func (g *modifications) traverse(ctx Context) {
    var pc = ctx.pc()
    if pc != nil { pc.Wait() }

    for _, m := range g.list {
        var ctx = at(ctx, m.position)
        if m.traverse(ctx); pc == nil { continue }
        if t := pc.traves.of(traveFail); t.has() { return }
        if t := pc.traves.not(traveCase, traveDone, traveNext, traveRule, traveFile); t.has() {
            if true || (options.verbose || options.verboseBreaks) {
                var _, ent, _ = entryIndicator(ctx, ctx.entry())
                warn(ctx, "%v: %s failed\n", ent, m.name)
                for _, s := range t { warn(at(ctx,s.pos), "%v: %v", m.name, s) }
                warnstack(ctx, 5, "").debug(16)
            }
            break
        }
        if t := pc.traves.of(traveCase); t.has() { continue }
        if t := pc.traves.of(traveDone, traveNext); t.has() { return }
    }
}

func (g *modifications) String() (s string) {
    s = "["
    for i, m := range g.list {
        if i > 0 { s += " " }
        s += m.String()
    }
    s += "]"
    return
}

func (p *modifications) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    erro(ctx, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}

type (
    modifier struct { Context }
    modifierFunc func(modifier, ...Value) (Value)
)

func (m modifier) apply(name string, args ...Value) (res Value) {
    if f, y := modifiers[name]; y {
        for i, a := range args {
            if g, y := a.(*Group); y {
                var ctx = at(m.Context, a.Position())
                var s = g.Elems[0].Strval(ctx)
                var v = modifier{ctx}.apply(s, g.Elems[1:]...)
                if true && s != "dirty" { warn(ctx, "%v -> %v", g, v).debug(1) }
                if v == nil { v = MakeNil(a.Position()) }
                args[i] = v
            }
        }
        return f(m, args...)
    }

    var ctx = m.Context
    var _, ent, _ = entryIndicator(ctx, ctx.entry())
    prompt(ctx, "%v: %s failed for %s\n", ent, name, ctx.Project())
    erro(ctx, "unknown modifier: %s (args=%v)", name, args)
    errostack(ctx, 5, "%v", ctx).debug(10)
    return
}

var (
    init_modifiers = map[string]modifierFunc{
        `print`:        modifier.print,
        `debug`:        modifier.debug,

        `select`:       modifier._select,

        `env`:          modifier.env, // interpreter environments
        `set`:          modifier.set,

        `closure`:      modifier._closure,
        `for`:          modifier._for,

        `cd`:           modifier.cd,
        `mkdir`:        modifier.mkdir,
        `path`:         modifier.path,

        `sudo`:         modifier.sudo,

        `touch`:        modifier.touch,
        `grep`:         modifier.grep,
        `deps`:         modifier.deps,

        `copy-file`:       modifier.copyfile,
        `write-file`:      modifier.writefile,
        `read-file`:       modifier.readfile,
        `update-file`:     modifier.updatefile,
        `configure-input`: modifier.configureinput,
        `configure-file`:  modifier.configurefile,
        `configure`:       modifier.configure,

        `wait`:         modifier._wait,
        `stamp`:        modifier.stamp,

        `check`:        modifier.check,
        `assert`:       modifier.assert,
        `case`:         modifier._case,
        `cond`:         modifier.cond,
        `if`:           modifier.cond,
        `where`:        modifier.cond,

        `once`:         modifier.once,

        `fork`:         modifier.fork,

        `git-ahead`:    modifier.gitahead,
        `git-modified`: modifier.gitmodified,

        `by`:           modifier.setDirtyPats,
        `dirty-by`:     modifier.setDirtyPats,
        `dirty-opts`:   modifier.setDirtyPats,

        `dirty`:            modifier.predictDirty,
        // `outdated`:         modifier.predictDirty,
        // `no-loop`:          modifier.predictNoLoop,
        // `target-1st-visit`: modifier.predictTarget1stVisit,
        // `target-max-visit`: modifier.predictTargetMaxVisit,
    }

    modifiers  = make(map[string]modifierFunc)
    crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)
func init() {
    // Install recursive modifiers here to avoid Go's loop detection.
    for s, m := range init_modifiers { modifiers [s] = m }
}

func RegisterModifiers(m map[string]modifierFunc) (err error) {
    for s, f := range m {
        if _, existed := modifiers[s]; existed {
            err = fmt.Errorf("modifier '%s' already existed", s)
            break
        } else {
            modifiers[s] = f
        }
    }
    return
}

func getGroupElem(value Value, n int, v Value) Value {
    if g, ok := value.(*Group); ok {
        if elem := g.Get(n); elem != nil {
            v = elem
        }
    }
    return v
}

func promptShellResult(ctx Context, value Value, n int) {
    if g, ok := value.(*Group); ok && g != nil {
        if elem := g.Get(0); elem != nil {
            if str := elem.Strval(ctx); str == "shell" {
                if elem = g.Get(n); elem != nil {
                    if str = elem.Strval(ctx); strings.HasSuffix(str, "\n") {
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

type modifierPrintOpts struct {
    generalOpts
    stdout bool `o,stdout`
    stderr bool `e,stderr`
    reset  bool `r,reset`
}
func (ctx modifier) print(args... Value) (result Value) {
    var (
        pos = ctx.Position()
        opts = modifierPrintOpts{ stderr: true }
        content string
    )
    args = parseOpts(ctx, &opts, plain, args...)
    if val := autoGet(ctx, "-"); val != nil { content = val.Strval(ctx) }
    if opts.stdout { fmt.Fprint(stdout, content) }
    if opts.stderr { fmt.Fprint(stderr, content) }
    if opts.reset  { ctx.autoSet("-", MakeNone(pos)) }
    return
}

func (ctx modifier) debug(aa ...Value) (result Value) {
    var opts, args = mop[struct {
        generalOpts
        cond   Value `if,cond,where,when`
        info []Value `i,info`
        warn []Value `w,warn`
        erro []Value `e,er,err,erro,error`
        checkOutdated bool `dirty,cd,checkdirty,check-dirty,co,check-outdated`
        traverse int `tr,trave,traverse`
        s int `s,stack,sn,stack-number`
        n int `c,count,n,num,cn,call-number`
    }](&ctx, plain, aa...)

    if opts.cond != nil && !opts.cond.True(ctx) { return }
    if n := opts.traverse; n > 0 { ctx.pc().debug_traverse += n }

    for _, v := range opts.info { info(of(ctx,v), "%s", v.Strval(ctx)).debug(1) }
    for _, v := range opts.warn { warn(of(ctx,v), "%s", v.Strval(ctx)).debug(1) }
    for _, v := range opts.erro { erro(of(ctx,v), "%s", v.Strval(ctx)).debug(1) }

    var (
        target  = autoGet(ctx, "@")
        depends = autoGet(ctx, "^")
    )
    if opts.checkOutdated && !isNil(target) {
        var (
            ordered = autoGet(ctx, "|")
            grepped = autoGet(ctx, "~")
            tt = target.stat(ctx).mod()
        )
        if tt.IsZero() {
            info(ctx, "target not exists: %v", target).debug(1)
            return
        }
        for _, dep := range merge(depends, ordered, grepped) {
            if dt := dep.stat(ctx).mod(); dt.After(tt) {
                info(ctx, "%v: outdated by %v (%v)", target, dep, dt.Sub(tt)).debug(1)
            }
        }
    }
    if len(opts.info) == 0 && len(opts.warn) == 0 && len(opts.erro) == 0 {
        var ( p = ctx.Position() ; s = ctx.stems() ; m *diagPoint )
        if len(args) == 0 {
            m = prompt(ctx, "%v: target=%v stems=%v depends=%v\n", p, target, s, depends)
        } else if opts.verbose {
            m = prompt(ctx, "%v: target=%v stems=%v depends=%v ; %v\n", p, target, s, depends, args)
        } else if len(args) == 1 {
            m = prompt(ctx, "%v: %v (%T)\n", p, args[0], args[0])
        } else {
            m = prompt(ctx, "%v: %v\n", p, args)
        }
        if n := opts.n * 2; opts.s > 0 {
            infostack(ctx, opts.s, "").debug(n)
        } else {
            m.debug(n)
        }
    }
    return
}

// select element by index from group result: (select 0)
func (ctx modifier) _select(args... Value) (result Value) {
    args = mergex(ctx, plain, args...)
    if h := autoGet(ctx, "-"); h == nil {
        erro(ctx, "no pipe value $-").debug(1)
    } else if g, ok := h.(*Group); ok && len(args) > 0 {
        if i, e := args[0].Integer(ctx); e != nil {
            erro(ctx, "%v: %v", args[0], e).debug(1)
        } else {
            result = g.Get(int(i))
        }
    }
    return
}

func (ctx modifier) env(args... Value) (result Value) {
    args = mergex(ctx, plain, args...)

    var program = ctx.program()
    for _, a := range args {
        if p, ok := a.(*Pair); ok {
            program._env = append(program._env, p)
        } else {
            erro(ctx, "env: not a pair value: %v (%T)", a, a).debug(1)
        }
    }
    return
}

// examples:
//     [(set name=value)]    set $(name) to 'value'
//     [(set name)]          clear $(name)
//     [(set -)]             clear $-
func (ctx modifier) set(aa ...Value) (result Value) {
    var opts, args = _opts[struct{
        generalOpts
    }](ctx, plain, aa...)

    var defs []Value
    var program = ctx.program()
    var pc = ctx.pc()
ForArgs:
    for _, arg := range args {
        var (
            name string
            value Value
            def *def
        )
        switch a := arg.(type) {
        case *bareword: name = a.string
        case *Pair: // NOTE: Pair.Value is not expanded, need to do it again.
            name, value = a.Key.Strval(ctx), a.Value.expand(ctx, plain)
            if isNil(value) { value = a.Value }
        case *Flag:
            name, value = a.name.Strval(ctx), MakeNone(a.Position())
            if name == "" { name = "-" }
        default:
            erro(ctx, "%T `%s` is unsupported (try: foo=value)", arg, arg).debug(1)
            return
        }

        if def = program.scope.FindDef(name); def == nil {
            erro(ctx, "no such def '%s' (%v, %v)", name, arg, args).debug(16)
            break ForArgs
        } else {
            var auto = def.origin == DefAuto || def.origin == DefArg
            if def.val(ctx, value); !auto && isNil(def.value) && !isNil(value) {
                errostack(ctx, 3, "set def wrong: %T %v (auto: %v)", value, value, autoGet(ctx, def.name)).debug(6)
            }

            defs = append(defs, def)

            if auto && name == "@" {
                var f, s, y = as{value}.fullname(ctx)
                if opts.verbose {
                    var d = ctx.gap(false)
                    var ts = trimPromptString(s)
                    prompt(ctx, "%s …… traversed (%d, %v)\n", ts, f.traversed, d)
                    if false { warnstack(ctx, 64, "%v, %v, (%v)", f, s, d).debug(64) }
                }
                if y && f.traversed > 1 {
                    pc.traves.add(ctx, traveDone, nil)
                }
            }
        }
    }
    if len(defs) > 0 { result = MakeListOrScalar(ctx.Position(), defs) }
    return
}

func (ctx modifier) setDirtyPats(args... Value) (result Value) {
    var opts = ctx.dirtyOpts()
    opts.pats = parseOpts(ctx, opts, plain, args...)
    return
}

// create closure context for the traversal
func (ctx modifier) _closure(aa ...Value) (result Value) {
    // Closure the caller program, the context will be restored when execution is finished.
    var closureCtx Context
    var pc = ctx.pc()
    if pc != nil {
        pc.Context = closureWith(pc.Context)
        closureCtx = pc.Context
    } else {
        erro(ctx, "needs closure context: %v", ctx).debug(1)
        return
    }

    assert(ctx.closure() != nil, "context not closured: %v", ctx)

    var set = func(name string, val Value) (t Value) {
        var noop bool
        if v, y := val.(*boolean); y {
            if !v.bool { noop = true }
        } else if isTrivial(val) {
            errostack(ctx, 3, "trivial target: %T %v", val, val).debug(1)
        } else if true {
            t = val.expand(ctx, plain)
        } else {
            t = val
        }

        if l, y := t.(*List); y && len(l.Elems) == 1 { t = l.Elems[0] }
        if !noop && isTrivial(t) { t = autoGet(closureCtx, name)  }

        if t != nil {
            ctx.autoSet(name, t) // aka (set @=&@)
        } else if !noop {
            errostack(ctx, 3, "%v: %s is nil", closureCtx.Project(), name).debug(1)
        }

        return
    }

    var opts, _ = _opts[struct{
        generalOpts
        dump   bool `d,dump`
        target Value `@,target`
        // depFirst bool `<,dep-first` // TODO: -<=value
        // depLast  bool `>,dep-last` // TODO: ->=value
    }](ctx, plain, aa...)

    if opts.verbose { info(ctx, "%v: %v", ctx.Project(), ctx).debug(1) }
    if opts.dump { infostack(ctx, -1, "%v: %v", ctx.Project(), ctx).debug(1) }
    if closureCtx != ctx { if opts.target != nil {
        var t = as{set("@", opts.target)}

        var ( f *File ; s string ; y bool ; n int )
        if f, s, y = t.fullname(ctx); !y {
            s = t.Strval(ctx)
        } else {
            n = f.traversed
        }

        if n > 1 {
            if opts.verbose {
                var d = ctx.gap(false)
                var ts = trimPromptString(s)
                prompt(ctx, "%s …… traversed (%d, %v)\n", ts, n, d)
                if false { warnstack(ctx, 64, "%v, %v, (%d, %v)", f, s, n, d).debug(64) }
            }

            pc.traves.add(ctx, traveDone, nil)
            return
        }

        if isInnerAuto(ctx, t.Value) {
            errostack(ctx, 16, "loop: %v", t).debug(10)
            return
        }
    } }

    if proj := ctx.Project(); proj == nil {
        errostack(ctx, 6, "%T: nil project in the context", ctx).debug(64)
    } else if scope := proj.scope; scope == nil {
        erro(ctx, "empty closure context").debug(1)
    } else if def := scope.FindDef("/"); def == nil {
        erro(at(ctx,scope.position), "&/ is undefined").debug(1)
    } else if dir := def.value.Strval(ctx); dir == "" {
        erro(at(ctx,scope.position), "&/ is empty").debug(1)
    } else if !filepath.IsAbs(dir) {
        erro(at(ctx,scope.position), "&/ is relative").debug(1)
    } else if err := enter(ctx, dir); err == nil {
        var program = ctx.program()
        program.project.changedWD = dir
        program.changedWD = dir
    }
    return
}

func (ctx modifier) _for(aa... Value) (result Value) {
    var _, _ = _opts[struct{
        generalOpts
    }](&ctx, plain, aa...)

    // TODO: ...

    return
}

func (ctx modifier) cd(aa... Value) (result Value) {
    var opts, args = _opts[struct{
        generalOpts
        makePath bool `p,path`
        printEnter bool `e,print-enter`
        printLeave bool `l,print-leave`
    }](&ctx, plain, aa...)

    if opts.printEnter { printEnteringDirectory(ctx) }
    if opts.printLeave { printLeavingDirectory(ctx) }
    if (opts.printEnter || opts.printLeave) && len(args) == 0 { return }
    if len(args) == 1 {
        var dir = args[0].Strval(ctx)
        if dir == "" {
            // TODO: do something special
            return
        }

        var program = ctx.program()
        if !filepath.IsAbs(dir) {
            dir = filepath.Join(program.project.absPath, dir)
        }
        if opts.makePath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
            if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                erro(ctx, "make path '%s' failed: %v", dir, err)
                return
            }
        }
        if err := enter(ctx, dir); err == nil {
            program.project.changedWD = dir
            program.changedWD = dir
        }
    } else {
        erro(ctx, "wrong number of cd args: %v", args).debug(1)
    }
    return
}

func (ctx modifier) mkdir(aa ...Value) (result Value) {
    var opts, args = _opts[struct{
        generalOpts
        mode os.FileMode `m,mode`
    }](&ctx, plain, aa...)

    if opts.mode == 0 { opts.mode = os.FileMode(0755) }
    if len(args) == 0 {
        var d = autoGet(ctx, "@")
        var s = d.Strval(ctx)
        if err := os.MkdirAll(filepath.Dir(s), opts.mode); err != nil {
            erro(ctx, "make path '%s' failed: %v", s, err).debug(1)
        }
        return
    }

    for _, a := range args {
        var s = a.Strval(ctx)
        if err := os.MkdirAll(s, opts.mode); err != nil {
            erro(ctx, "make path '%s' failed: %v", s, err).debug(1)
            break
        }
    }
    return
}

// (path $(dir $@))
// (path /example/path)
func (ctx modifier) path(aa ...Value) (result Value) {
    var _, args = _opts[struct{
        generalOpts
    }](&ctx, plain, aa...)

    if len(args) == 0 {
        var d = autoGet(ctx, "@")
        var s = d.Strval(ctx)
        if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
            if err := os.MkdirAll(s, os.FileMode(0755)); err != nil {
                erro(ctx, "make path '%s' failed: %v", err).debug(1)
            }
        }
        return
    }

    for _, arg := range args {
        var s = arg.Strval(ctx)
        if err := os.MkdirAll(s, os.FileMode(0755)); err != nil {
            erro(ctx, "make path '%s' failed: %v", s, err).debug(1)
            break
        }
    }
    return
}

func (ctx modifier) sudo(args... Value) (result Value) {
    erro(at(ctx,ctx.Position()), "TODO: sudo modifier is not implemented yet").debug(1)
    return
}

func parseDependList(ctx Context, dependList *List) (depends *List) {
    var pc = ctx.pc()
    depends = new(List)
    for _, depend := range dependList.Elems {
        switch d := depend.(type) {
        case *List:
            if dl := parseDependList(ctx, d); dl != nil {
                depends.Elems = append(depends.Elems, dl.Elems...)
            }
        case *execResult:
            if d.Status != 0 {
                brk := pc.traves.add(ctx, traveFail, nil)
                brk.error = fmt.Errorf("bad status %v", d.Status)
                return // target shall be updated
            } else {
                depends.Append(d)
            }
        case *Rule:
            switch d.class {
            case GeneralRule, PatternRule, PathPattRule:
                depends.Append(d)
            default:
                erro(ctx, "unsupported entry depend `%v' (%v)", d, d.class).debug(1)
            }
        case *String:
            depends.Append(d)
        case *File:
            depends.Append(d)
        default:
            var program = ctx.program()
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
    modifierGrepOpts
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
            if file.info, _ = os.Stat(file.Strval(ctx)); file.info == nil { continue }
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
    } else if s, _ := g.target.fullnameOrStrval(ctx); s == g.targetFullName {
        res = true
    } else if f, y := toFile(g.target.Value); y && f.name == file.name {
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
                file := stat(ctx, a[0], a[1], a[2])
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
            fmt.Fprintf(w, "%s|%s|%s\n", file.name, file.sub, file.dir)
        }
    }
}

func searchGreppedName(ctx Context, gp Position, gc *grepctx, sys bool, name string) (res *File) {
    var isAbs, isRel bool
    if isAbs = filepath.IsAbs(name); isAbs {
        res = stat(ctx, name, "", "", nil)
    } else if isRel = isRelPath(name); isRel { // relative to targetDir
        res = stat(ctx, name, "", gc.targetDir, nil)
    } else if res = file(ctx, name); res != nil && res.exists() {
        return // found existed file
    }

    // System files are not treated as missing nor collected
    // for further updating, just discard them immediately.
    if !sys && res != nil && res.filemap != nil && len(res.filemap.locs) == 1 {
        // system files defined by `files ((foo.xxx) ⇒ -)`
        if f, ok := res.filemap.locs[0].(*Flag); ok {
            sys = isNone(f.name) || isNil(f.name)
        }
    }
    if !sys && gc.debug>0 {
        erro(ctx, "%v: %v → %v (exists=%v, sys=%v, from %v)\n",
            ctx.entry(), gc.target, name, res.exists(), sys, ctx.Project()).
            debug(gc.debug)
    }
    if sys || res.exists() { return }

    // relative to target directory
    var alt = stat(ctx, name, "", gc.targetDir)
    if alt != nil { res = alt; return }

    // Check for bare non-system sub-paths:
    //   foo/bar/name.xxx
    // We search base name 'name.xxx' again:
    var s = filepath.Dir(name) // e.g: foo/bar

    // Search 'name.xxx' and check dir for
    // 'foo/bar' suffix. We use it if found.
    alt = file(ctx, filepath.Base(name))
    if alt != nil && strings.HasSuffix(alt.dir, PathSep+s) {
        dir := strings.TrimSuffix(alt.dir, PathSep+s)
        ok1 := alt.change(dir, s, alt.name) // <dir>, foo/bar, name.xxx
        ok2 := alt.change(dir, "", name) // <dir>, "", foo/bar/name.xxx
        res  = alt
        if enable_assertions {
            assert(ok1, "unchanged: %s %s %s", dir, s, alt.name)
            assert(ok2, "unchanged: %s %s", dir, alt.name)
        }
    } else if res == nil {
        for _, inc := range gc.incs {
            if res = stat(ctx, name, "", inc.Strval(ctx)); res != nil {
                if false { info(ctx, "%v in %v", res, inc).debug(1) }
                return
            }
        }
        if res == nil { res = stat(ctx, name, "", "", nil) }
        warn(at(ctx,gp), "'%s' not found in %v", name, ctx.Project())
        warn(ctx, "grepped '%s' has no target dir in %v", name, ctx.Project())
        warn(at(ctx,ctx.Project().position), "from project %v (for %v)", ctx.Project(), name).debug(8)
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
            if file.info, err = os.Stat(file.Strval(ctx)); err != nil {
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
        info(at(ctx,gp), "%s: `%s` not found", ctx.Project().name, name)
    } else if !file.exists() {
        info(at(ctx,gp), "%s: `%s` file not existed", ctx.Project().name, name)
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
    } else if project := ctx.Project(); project == nil {
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
    if len(cleanDirs) == 0 {
        var clean =  options.cleanTmpDirs
        if  clean || options.cleanDotCache { cleanDirs = append(cleanDirs, ".cache") }
        if  clean || options.cleanDotDeps  { cleanDirs = append(cleanDirs, ".deps") }
        if  clean || options.cleanDotGrep  { cleanDirs = append(cleanDirs, ".grep") }
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
            info(ctx, "%s: removed %v", ctx.Project(), s).debug(1)
        } else {
            prompt(ctx, "%s: removed %v\n", ctx.Project(), s)
        }
    }
}

func getSavedDepsFileName(ctx Context, targetFullName string, strs []string) (filename string, err error) {
    var ( file *File; hashees []interface{} )
    for _, s := range strs { hashees = append(hashees, s) }
    if file, err = tempFile(ctx, ".deps", targetFullName, hashees...); err != nil {
        erro(ctx, "get .deps temp file failed: %v", err).debug(1)
    } else {
        filename, _ = as{file}.fullnameOrStrval(ctx)
    }
    return
}

func getSavedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
    var ( file *File )
    if file, err = tempFile(ctx, ".grep", targetFullName); err != nil {
        erro(ctx, "get .grep temp file failed: %v", err).debug(1)
    } else {
        filename, _ = as{file}.fullnameOrStrval(ctx)
    }
    return
}

func loadSavedGrepFile(ctx Context, gc *grepctx) (okay bool, err error) {
    if gc.savedGrepFileName, err = getSavedGrepFileName(ctx, gc.targetFullName); err != nil {
        erro(ctx, "get saved grep filename failed: %v", err).debug(1)
        return
    } else if gc.savedGrepFile = stat(ctx, gc.savedGrepFileName, "", ""); gc.savedGrepFile == nil {
        return // No saved grepfile yet!
    }

    var file, ok = toFile(gc.target)
    if !ok {
        file = stat(ctx, gc.targetFullName, "", "")
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
                warn(at(ctx,ctx.Project().position), "from project %v", ctx.Project()).debug(6)
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
                    warn(at(ctx,ctx.Project().position), "from project %v", ctx.Project()).debug(6)
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
        targetName = v.name
        gc.targetInfo = v.info
        gc.targetFullName = v.fullname()
        gc.targetDir = filepath.Dir(gc.targetFullName)
        if v.isSysFile() { return }
    default:
        gc.targetDir = ctx.Project().absPath
        targetName = v.Strval(ctx)
        if filepath.IsAbs(targetName) {
            gc.targetFullName = targetName
        } else {
            gc.targetFullName = filepath.Join(gc.targetDir, targetName)
        }
        if file := stat(ctx, gc.targetFullName, "", ""); file == nil {
            erro(of(ctx,gc.target), "grep: '%s' not found (%v)", gc.targetFullName, gc.target).debug(16)
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

    if false { defer un(tt(t_traverse, ctx.pc(), gc.target)) }

    defer func(restore []Value) {
        var t = ctx.pc()
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
            gc.targetFullName, len(gc.files), len(ctx.pc().grepped)).debug(1) }
        return
    }
    if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
            erro(ctx, "make grep dir failed: %v", err).debug(1)
            return
        }
    }
    if options.saveGrepSource {
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
type modifierGrepOpts struct {
    generalOpts
    discard bool `c,cast;dc,discard;dm,discard-missing;im,ignore-missing`
    fileinc bool `f,file;f,files` // work with the 'incs' field
    langs []string `l,lang;lan,language`
    sys []string `s,sys;ss,system`        // matching system includes
    reg []string `re,reg;regx,regex;x,rx` // matching user includes
    incs []Value `i,inc;i,include` // include search paths, also 'fileinc' field
    touch bool `t,touch;t,touch-outdate;t,touch-outdated`
    recursive bool `a,all;r,recur;rr,recursive`
    noTraverse bool `n,notraverse;nt,no-traverse;go,grep-only`
}
func (ctx modifier) grep(args... Value) (result Value) {
    if false && options.noDepsGrep || options.noGrep {
        return
    }

    var gc grepctx
    gc.fileinc = true // grep files by default
    args = parseOpts(ctx, &gc.modifierGrepOpts, plain, args...)
    gc.incs = mergex(ctx, plain, gc.incs...)
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
        target = autoGet(ctx, "@")
        targets = args
        grepped = ctx.pc().grepped
    )
    if len(targets) == 0 { if target == nil || isNil(target) || isNone(target) {
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

    var pc = ctx.pc()
    var tar = target
    defer func(v bool) { pc.grepping = v } (pc.grepping)
    pc.grepping = true

ForTarget:
    for _, target := range targets {
        if isNil(target) {
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

    var pos = ctx.Position()
    if !gc.noTraverse {
        ctx.autoSet("~", MakeNone(pos))
        pc.grepped = nil
    } else {
        result = MakeListOrScalar(pos, pc.grepped)
    }
    return
}

type depContext struct { diagContext }
func (ctx *depContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("dep{%s}", ctx.diagContext)
    } else {
        return ctx.diagContext.String()
    }
}
func (ctx *depContext) appendCallerUpdated() bool { return false }

func parseDeps(ctx Context, targetVal Value, targetStr string, savedDepsFile *File, savedDepsFileName, deps string) (files []Value) {
    const parallel = true
    var (
        proj = ctx.Project()
        targetFullName, _ = as{targetVal}.fullnameOrStrval(ctx)
        filesMux sync.Mutex
        firstWord string
        err error
    )
    var findDepFile = func(name string) (file *File) {
        if filepath.IsAbs(name) {
            file = stat(ctx, name, "", "", nil)
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

    var pc = ctx.pc()
    var depFile = func(ctx Context, depPos Position, word string) {
        var dc = depContext{diagContext{ Context: ctx }}; ctx = &dc
        if parallel { defer func() {
            checkFailure(ctx, true/* don't call flushDiags */)
            if len(dc.points) > 0 { dc.inner().diagnostic().nest(dc.points) }
            jobs.Done() // minus 1
        }() }

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
                var s = filepath.Base(file.name)
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
            if n = dc.flushDiags(true); n > 0 { // aka. dc.points = nil
                var s = trimPromptString(targetVal.String())
                prompt(ctx, "%v: %d errors counted\n", word, n).debug(1)
                erro(ctx, `%v: %d errors for "%s", dep "%s"`, proj, n, s, word)
                errostack(ctx, 5, `%v: %v`, ctx).debug(6)
            }
        } else {
            if n = dc.countErrors(); n > 0 {
                // reset to reduce diags as we wish to continue with the errors
                dc.points, dc.errs = nil, 0
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
                } else if firstDepFile := stat(ctx, firstDep, "", ""); firstDepFile == nil {
                    return nil // requests to update savedDepsFile
                } else if firstDepFile.info.ModTime().After(savedDepsFile.info.ModTime()) {
                    return nil // requests to update savedDepsFile
                }
                if parallel {
                    jobs.Add(1); go depFile(ctx.spawn(ctx), depPos, word)
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
            fail(ctx.Position(), "removed %s", savedDepsFileName)
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
    if targetVal, targetStr := getTargetValueString(ctx); isNil(targetVal) {
        erro(ctx, "target is nil").debug(1)
    } else if targetStr == "" {
        erro(ctx, "target '%v' is empty", targetVal).debug(1)
    } else if savedDepsFileName, err = getSavedDepsFileName(ctx, targetStr, args); err != nil {
        erro(ctx, "get saved deps filename failed: %v", err).debug(1)
    } else if savedDepsFileName == "" {
        erro(ctx, "empty saved deps filename", savedDepsFileName).debug(1)
    } else if savedDepsFile := stat(ctx, savedDepsFileName, "", ""); savedDepsFile == nil {
        // no saved deps file
    } else if savedDepsBytes, err = ioutil.ReadFile(savedDepsFileName); err != nil {
        erro(ctx, "can'ctx open saved deps file: %v", savedDepsFileName, err).debug(1)
    } else if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, string(savedDepsBytes)); len(files) > 0 {
        if false { info(ctx, "loaded deps %s (%d files)", savedDepsFileName, len(files)).debug(true, 1) }
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
        proj = ctx.Project()
        pc = ctx.pc()
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
            fail(ctx.Position(), "dep '%s' is not file", dep)
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
    var pc = ctx.pc()
    for _, rx := range knownerrors {
        var all [][][]byte = rx.FindAllSubmatch(errBytes, -1)
        if all != nil { for _, m := range all {
            if rx == rxFatalErrorFileNotFound {
                if promptBeforeTraverse { prompt(ctx, "%s\n", m[0]).debug(6) }
                if dep := string(m[4]); dep == lastTry {
                    return false, ""
                } else if res = traverseMissingDep(ctx, dep); !res || pc.traves.has() {
                    var (
                        s, l, c = string(m[1]), string(m[2]), string(m[3])
                        pos = convPosition(s, l, c)
                    )
                    prompt(ctx, "%s: dep missing, project %v\n", m[4], ctx.Project())
                    prompt(ctx, "%s\n", m[0]) // prompt the entire error line
                    erro(at(ctx,pos), "%v", ctx).debug(1)
                    return
                } else if tried == "" { tried = dep }
            } else if promptErrors {
                prompt(ctx, "%s\n", m[0])/*.debug(1)*/
            }
        }}
    }
    return
}

type modifierDepsContext struct { Context }
func (mdc *modifierDepsContext) mustExists() bool { return true }
func (mdc *modifierDepsContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("deps{%s}", mdc.Context)
    } else {
        return mdc.Context.String()
    }
}

type modifierDepsOpts struct {
    generalOpts
    useClang bool `cl,clang`
    useGcc bool `g,gcc`
    addMissing bool `am,add-missing;mg,missing-goal;MG,MissingGoal`
    lang string `l,lan,lang,language`
    flags []Value `f,flags,o,opts`
    cc string `c,cc,compiler`
}
func (ctx modifier) deps(args... Value) (result Value) {
    if options.noDepsGrep || options.noDeps { return }

    // NOTE: parse opts for (deps) before expanding the args, because we share args
    //       with the compilers!
    var (
        targetVal Value
        targetStr string
        opts modifierDepsOpts
        err error
    )
    if targetVal, targetStr = getTargetValueString(ctx); isNil(targetVal) {
        erro(ctx, "target is nil").debug(1)
        return
    } else if targetStr == "" {
        erro(ctx, "target '%v' is empty", targetVal).debug(1)
        return
    } else {
        args = parseOpts(ctx, &opts, plain, args...)
    }

    var files []Value
    if opts.verbose {
        defer func(ts time.Time) {
            var s string
            if val := autoGet(ctx, "@"); val != nil { s = val.String() }
            prompt(ctx, "Deps %v …… (%d files in %v)\n", s, len(files), time.Now().Sub(ts)).debug(opts.debug, 6)
        } (time.Now())
    }

CorrectCC:
    switch opts.cc {
    case "cl"   : opts.cc = "clang"; goto CorrectCC
    case "gc"   : opts.cc = "gcc"  ; goto CorrectCC
    case "clang": opts.useClang = true
    case "gcc"  : opts.useGcc   = true
    case "":
        if opts.useGcc   { opts.cc = "gcc" }
        if opts.useClang { opts.cc = "clang" }
    default:
        if base := filepath.Base(opts.cc); base == "" {
            erro(ctx, "unsupported cc: %v", opts.cc).debug(1)
            return
        } else if strings.HasPrefix(base, "clang") { opts.useClang = true
        } else if strings.HasPrefix(base, "gcc")   { opts.useGcc   = true }
    }

    var (
        flags = mergex(ctx, plain, opts.flags...)
        _MM, _MG bool
        ca []string
    )
    for _, f := range flags {
        switch s := strings.TrimSpace(f.Strval(ctx)); s {
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
    if !_MG && opts.addMissing { ca = append(ca, "-MG") } // add missing headers
    for _, a := range args {
        var s, y = as{a}.fullnameOrStrval(ctx) ; if y { s = strings.TrimSpace(s) }
        if strings.Contains(s, "-v -fPIC -fvisibility-inlines-hidden") {
            var v = a.expand(ctx, plain)
            warn(ctx, "%T %v", a, a)
            warn(ctx, "%T %v", v, v).debug(1)
        }
        switch s {
        case "", "-M", "-MM", "-MG", "-MD", "-MV", "-MP", "-Os", "-O1", "-O2", "-O3",
            "-c", "-shared", "-static", "-fPIC", "-fvisibility-inlines-hidden",
            "-fcxx-modules", "-fmodules", "-fmodules-ts":
            break // discard unused args
        default: ca = append(ca, s)
        }
    }

    var (
        proj = ctx.Project()
        pc = ctx.pc()
        savedDepsFileName string
    )

    ctx.Context = &modifierDepsContext{ ctx.Context }
    if savedDepsFileName, files = loadSavedDepsAndCheckOutdated(ctx, ca); pc.traves.has() {
        for _, brk := range pc.traves { erro(at(ctx,brk.pos), "%v", brk) }
        errostack(ctx, 5, "%v: %v", proj, ctx).debug(16)
        return
    } else if len(files) == 0 {
        var (
            cc = exec.Command(opts.cc, ca...)
            stdout bytes.Buffer
            stderr bytes.Buffer
            retried string
        )
    retryCC:
        cc.Stdout, cc.Stderr = &stdout, &stderr
        if err = cc.Run(); err != nil {
            var okay = false
            if okay, retried = traverseMissingDeps(ctx, retried, stderr.Bytes()); okay && !pc.traves.has() {
                cc = exec.Command(opts.cc, ca...)
                stdout.Reset()
                stderr.Reset()
                goto retryCC
            }
            prompt(ctx, "%v: failed command '%s':\n", proj, opts.cc)
            prompt(ctx, "%s \\\n  %s\n----------\n", cc.Path, strings.Join(ca, " \\\n  "))
            prompt(ctx, "%s\n----------\n%s----------\n", &stdout, &stderr)
            erro(ctx, "%s: %s deps failed: %v", proj, filepath.Base(opts.cc), err)
            errostack(ctx, 5, "%s: %v", proj, ctx).debug(8)
            return
        }
        if stderr.Reset(); savedDepsFileName == "" {
            erro(ctx, "empty saved deps file name: %v", savedDepsFileName).debug(1)
            stdout.Reset(); return
        }

        var savedDepsFile *File = nil//stat(ctx, savedDepsFileName, "", "")
        if files = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, stdout.String()); len(files) == 0 {
            warn(ctx, "parse deps file failed").debug(1) // not saving if failed
        } else if err = os.MkdirAll(filepath.Dir(savedDepsFileName), os.FileMode(0755)); err != nil {
            erro(ctx, "make path '%s' failed: %v", filepath.Dir(savedDepsFileName), err).debug(1)
        } else if err = ioutil.WriteFile(savedDepsFileName, stdout.Bytes(), os.FileMode(0666)); err != nil {
            erro(ctx, "save deps file failed: %v", err).debug(1)
        } else if false {
            info(ctx, "saved deps %s", savedDepsFileName).debug(true, 1)
        }
        stdout.Reset() // release buffers (optional)
    }
    if t := ctx.pc(); t != nil && len(files) > 0 {
        t.grepped = append(t.grepped, files...)
    }
    return
}

type modifierTouchOpts struct {
    verbose bool `v,verbose`
    debug bool `d,debug`
    path bool `p,path`
    mode os.FileMode `m,mode`
}
func (ctx modifier) touch(args... Value) (result Value) {
    var opts modifierTouchOpts // = modifierTouchOpts{ mode: os.FileMode(0755) }
    if args = parseOpts(ctx, &opts, plain, args...); len(args) == 0 {
        if val := autoGet(ctx, "@"); val != nil { args = append(args, val) }
    }

    var files []*File
    for _, arg := range args {
        var vf []*File
        if err := touch(ctx, arg, uint32(opts.mode), opts.path); err != nil {
            erro(ctx, "touch '%v' failed: %v", arg, err).debug(1)
            break
        } else if vf, err = arg.stamp(ctx); err != nil {
            erro(ctx, "touch '%v' failed: %v", arg, err).debug(1)
            break
        } else { files = append(files, vf...) }
    }

    var program = ctx.program()
    if opts.verbose { reportFileUpdates(ctx, files) }
    if len(program.getModifiers(ctx, "stamp")) > 0 {
        warn(ctx, "no need to use a (stamp) after (touch)").debug(1)
    }
    return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
func (ctx modifier) check(aa ...Value) (result Value) {
    var (
        pos = ctx.Position()
        pc  = ctx.pc()
        optBreak travekind // breaking with good results
        makeResult func(Position,bool) Value // returns results only if non-nil
        values []Value
        res bool
    )

    var opts, args = _opts[struct{
        generalOpts
        trim bool `trim,trim-string`
        answer bool `a,answer`
        boolean bool `b,boolean;r,result`
        silent bool `s,slient`
        exists bool `e,ex,exists`
        regular bool `reg,regular`
        isdir bool `isdir`
        good bool `g,good`
        file Value `f,file`
        dir Value `di,dir`
    }](ctx.Context, plain, aa...)

    if opts.good   { optBreak   = traveDone }
    if opts.answer { makeResult = func(p Position,v bool) Value { return MakeAnswer(p, v) } }
    if makeResult == nil && ( opts.boolean ||
        (opts.file != nil && (opts.exists || opts.regular || opts.isdir)) ||
        (opts.dir  != nil && (opts.exists || opts.regular || opts.isdir)) ||
        (opts.silent)) { makeResult = func(p Position,v bool) Value { return MakeBoolean(p, v) } }

    var checkFile = func (val Value, dir bool) {
        var ( s string; f *File )
        if v, y := val.(*boolean); y {
            if v.bool { val = autoGet(ctx, "@") } else { val = nil }
        }

        if val == nil {
            erro(ctx, "nil file value to check").debug(1)
            return
        } else if f, res = toFile(val); res {
            // best case
        } else if s = val.Strval(ctx); filepath.IsAbs(s) {
            if f = stat(at(ctx, val.Position()), s, "", ""); f != nil { res = true }
        } else if f = file(ctx, s); f != nil { res = true }

        if f != nil {
            if !dir || opts.regular { res = f.exists()
                if opts.verbose { warnstack(of(ctx,val), 3, "check regular file '%v': %v", val, res).debug(1) }
            } else if dir || opts.isdir { res = f.info != nil && f.info.Mode().IsDir()
                if opts.verbose { warnstack(of(ctx,val), 3, "check dir '%v': %v", val, res).debug(1) }
            } else if opts.exists { res = f.exists()
                if opts.verbose { warnstack(of(ctx,val), 3, "check file exists '%v': %v", val, res).debug(1) }
            } else if opts.verbose {
                warnstack(of(ctx,val), 3, "check file '%v': %v", val, res).debug(1)
            }
        } else if opts.verbose {
            warnstack(of(ctx,val), 3, "check file '%v': %v", val, res).debug(1)
        }

        if makeResult != nil {
            values = append(values, makeResult(pos, res))
        } else if !res {
            pc.traves.addf(ctx, optBreak, "'%v' is not file", val)
            return
        }
    }

    if opts.file != nil { checkFile(opts.file, false) }
    if opts.dir  != nil { checkFile(opts.dir, true) }

    var program = ctx.program()
    var value = autoGet(ctx, "-")
ForPairs:
    for _, arg := range args {
        var p, y = arg.(*Pair)
        if !y {
            if res = arg.True(ctx); makeResult != nil {
                values = append(values, makeResult(pos, res))
            } else {
                pc.traves.addf(ctx, optBreak, "value '%v' is false", arg)
                if opts.verbose { warn(ctx, "value '%v' is false", arg).debug(1) }
            }
            continue
        }

        var key, str string
        switch key = p.Key.Strval(ctx); key {
        case "status":
            var exeres, _ = value.(*execResult)
            if exeres == nil {
                pc.traves.addf(ctx, optBreak, "value '%v' is not exec result", value)
                erro(of(ctx,value), "value '%v' (%T) is not exec result", value, value).debug(6)
                return
            } else { /*exeres.wg.Wait()*/ }

            var num, e = p.Value.Integer(ctx)
            if e != nil {
                erro(ctx, "%v: %v", p.Value, e).debug(1)
                return
            }
            if opts.verbose {
                prompt(ctx, "checking status ")
                if num != 0 { prompt(ctx, "== %d ", num) }
                prompt(ctx, "…")
            }

            var good = exeres.Status == int(num)
            if opts.verbose {
                var s string
                if good { s = "Yes" } else { s = "No" }
                prompt(ctx, "… %s (%d)\n", s, exeres.Status)
            }
            if opts.debug>0 {
                var tar = autoGet(ctx, "@")
                var val = autoGet(ctx, "-")
                warn(at(ctx,program.position), "%v: %v", ctx.entry(), tar)
                warn(ctx, "status=%v", exeres.Status)
                warn(ctx, "hyphen=%v", val)
                warn(ctx, "context: %v", ctx).debug(opts.debug)
            }

            if makeResult != nil {
                values = append(values, makeResult(pos, good))
            } else if !good {
                pc.traves.addf(ctx, optBreak, "bad status (%v) (expects %v)", exeres.Status, p.Value)
                break ForPairs
            }
        case "stdout", "stderr":
            var exeres, _ = value.(*execResult)
            if exeres == nil {
                pc.traves.addf(ctx, optBreak, "not an exec result (%T)", value)
                erro(of(ctx,value), "value '%v' (%T) is not exec result", value, value).debug(6)
                return
            } else { /*exeres.wg.Wait()*/ }

            if opts.verbose {
                prompt(ctx, "checking %s (status=%d) … ", key, exeres.Status)
            }
            if opts.debug>0 {
                var tar = autoGet(ctx, "@")
                var val = autoGet(ctx, "-")
                warn(at(ctx,program.position), "%v: %v", ctx.entry(), tar)
                warn(ctx, "status=%v", exeres.Status)
                warn(ctx, "hyphen=%v", val)
                warn(ctx, "context: %v", ctx).debug(opts.debug)
            }

            var v *bytes.Buffer
            switch key {
            case "stdout": v = exeres.Stdout.Buf
            case "stderr": v = exeres.Stderr.Buf
            default: unreachable()
            }

            if v == nil {
                pc.traves.addf(ctx, optBreak, "bad %s (expects %v)", key, p.Value)
                break ForPairs
            }

            str = p.Value.Strval(ctx)
            if opts.trim { str = strings.TrimSpace(str) }

            if res := v.String() == str; makeResult != nil {
                values = append(values, makeResult(pos, res))
            } else if !res {
                pc.traves.addf(ctx, optBreak, "bad %s (%v) (expects %v)", key, v, p.Value)
                break ForPairs
            }
        case "file", "dir": // file=xxx and dir=xxx, same as -file=xxx and -dir=xxx
            var ( f *File; res bool )
            if f, res = toFile(p.Value); res {
                // ok
            } else if str = p.Value.Strval(ctx); filepath.IsAbs(str) {
                if f = stat(at(ctx, p.Value.Position()), str, "", ""); f != nil {
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
                values = append(values, makeResult(pos, res))
            } else if !res {
                pc.traves.addf(ctx, optBreak, "`%v` is not %s", p.Value, key)
                break ForPairs
            }
        case "var":
            var g, ok = p.Value.(*Group)
            if !ok {
                pc.traves.addf(ctx, optBreak, "`%v` is not a group value", p.Value)
                break ForPairs
            }
            for _, elem := range g.Elems {
                switch p := elem.(type) {
                case *Pair:
                    var a, b string
                    var k = p.Key.Strval(ctx)
                    var def = program.project.scope.FindDef(k)
                    if def != nil {
                        a = p.Value.Strval(ctx)
                        b = def.value.Strval(ctx)
                        if res := a != b; makeResult != nil {
                            values = append(values, makeResult(pos, res))
                        } else if !res {
                            pc.traves.addf(ctx, optBreak, "`%v` != `%v`", p.Key, p.Value)
                            break ForPairs
                        }
                    } else if makeResult != nil {
                        values = append(values, makeResult(pos, false))
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
            erro(ctx, "unknown check for %v -> %v", p.Key, p.Value).debug(1)
            break ForPairs
        }
    }

    if len(values) > 0 { result = MakeListOrScalar(pos, values) }
    return
}

type copyopts struct {
    program *Program
    path, update bool
    mode os.FileMode
    head Value
    foot Value
    files, copied int
    bytes int64
}

func copyRegular(ctx Context, src, dst string, opts *copyopts) (err error) {
    var def1, def2 *def
    if true {
        def1 = opts.program.scope.Lookup("1").(*def)
        def2 = opts.program.scope.Lookup("2").(*def)
    } else if g := ctx.Globe(); g != nil {
        def1 = g.Lookup("1").(*def)
        def2 = g.Lookup("2").(*def)
    }
    defer func(v1, v2 Value) { def1.value, def2.value = v1, v2
        // if err == nil {
        //         var file = stat(ctx, dst, "", "")
        //         ctx.Globe().stamp(dst, file.info.ModTime())
        // }
    } (def1.value, def2.value)

    var pos = ctx.Position()
    def1.value = MakeString(pos, dst)
    def2.value = MakeString(pos, src)

    var head, foot string
    if opts.head != nil { head = opts.head.Strval(ctx) }
    if opts.foot != nil { foot = opts.foot.Strval(ctx) }

    // Compare mod time for update mode
    if opts.files += 1; opts.update {
        if st2, e := os.Stat(dst); e == nil && st2 != nil {
            var st1 os.FileInfo
            if st1, err = os.Stat(src); err != nil { erro(ctx, "%v", err); return }
            if st1 != nil && (st1.Size()+int64(len(head))+int64(len(foot))) == st2.Size() {
                if st2.ModTime().After(st1.ModTime()) { return }
            }
            if false { fmt.Fprintf(stderr, "%s: %s (%v,%v)\n", pos, dst, st1.Size(), st2.Size()) }
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
type modifierCopyFileOpts struct {
    path bool "p,path"
    recursive bool "r,recursive"
    verbose bool "v,verbose"
    silent bool "s,silent"
    override bool "o,override"
    update bool "u,update"
    quick bool "q,quick"
    mode os.FileMode "m,mode"
    head Value "h,head"
    foot Value "f,foot"
}
func (ctx modifier) copyfile(args... Value) (result Value) {
    var opts modifierCopyFileOpts
    args = parseOpts(ctx, &opts, plain, args...)

    var target Value
    var source Value
    if len(args) > 0 {
        target = args[0]
    } else {
        target = autoGet(ctx, "@")
    }
    if len(args) > 1 {
        source = args[1]
    } else {
        source = autoGet(ctx, "<")
    }

    // Get target filename
    var (
        project = ctx.Project()
        filename, srcname string
        filetime, srctime time.Time
    )
    switch tv := target.(type) {
    case *File:
        if filename = tv.fullname(); tv.info != nil {
            filetime = tv.info.ModTime()
        }
    default:
        filename = target.Strval(ctx)
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
        srcname = source.Strval(ctx)
        if file := project.file(ctx, srcname); file != nil {
            source, srcname = file, file.fullname()
            if file.info != nil { srctime = file.info.ModTime() }
        }
    }

    if !filetime.IsZero() && filetime.After(srctime) {
        if opts.update {
            if opts.verbose { prompt(ctx, "update %v …", target) }
        } else if opts.override {
            if opts.verbose { prompt(ctx, "override %v …", target) }
        } else {
            if opts.verbose { prompt(ctx, "copy %v …… already existed!\n", target) }
            if !opts.silent { erro(ctx, "file already existed (%s)", target).debug(1) }
            return
        }
    } else if opts.verbose {
        if opts.update {
            prompt(ctx, "Checking %v …", target)
        } else {
            prompt(ctx, "Copy %v …", target)
        }
    }

    if opts.quick {
        var file = stat(ctx,filename,"","",nil)
        if file == nil || file.info != nil {
            if opts.verbose { prompt(ctx, "… Good\n") }
            return
        }
    }

    var program = ctx.program()
    var copts = &copyopts{
        program, opts.path||opts.recursive,
        opts.update, opts.mode, opts.head, opts.foot,
        0, 0, 0,
    }
    var file *File
    if file = stat(ctx,srcname,"","",nil); file == nil || file.info == nil {
        erro(ctx, "'%s' source file not found", srcname).debug(1)
    } else if !file.info.IsDir() {
        if opts.mode == 0 { opts.mode = file.info.Mode() }
        if err := copyFile(ctx, file.info, srcname, filename, copts); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    } else if opts.recursive {
        if err := copyDir(ctx, srcname, filename, copts); err != nil {
            erro(ctx, "%v", err).debug(1)
        }
    } else {
        erro(ctx, "`%v` is a directory (use -r to solve it)", source).debug(1)
    }

    if opts.verbose {
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

func (ctx modifier) writefile(args... Value) (result Value) {
    args = mergex(ctx, plain, args...)

    var (
        target = autoGet(ctx, "@")
        filename, str string
        f *os.File
    )
    if isNil(target) {
        erro(ctx, "target is undefined").debug(1)
        return
    }

    defer func() {
        if filename != "" { os.Remove(filename); f = nil }
        if f == nil {
            var pc = ctx.pc()
            brk := pc.traves.add(ctx, traveFail, target)
            brk.error = fmt.Errorf("file %s not generated", target)
        }
    } ()

    filename, _ = as{target}.fullnameOrStrval(ctx)

    if h := autoGet(ctx, "-"); h == nil {
        erro(ctx, "buffer value is nil").debug(1)
        return
    } else {
        str = h.Strval(ctx)
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
        result = stat(ctx, filename, "", "")
        f.Close()
    }
    return
}

type modifierReadFileOpts struct {
    generalOpts
    head Value "h,head"
    foot Value "f,foot"
}
func (ctx modifier) readfile(aa... Value) (result Value) {
    var (
        opts modifierReadFileOpts
        args []Value
        filename string
        file *File
        target as
    )
    args = parseOpts(ctx, &opts, plain, aa...)

    if n := len(args); n > 1 {
        erro(ctx, "too many files: %v", args).debug(1)
        return
    } else if n == 1 {
        target.Value = args[0]
    } else {
        target.Value = autoGet(ctx, "@")
    }

    var pc = ctx.pc()
    if target.trivial() {
        errostack(ctx, 3, "target for reading is invalid (%T) (%v -> %v)", target.Value, aa, args).debug(10)
        return
    } else if file, filename, _ = target.fullname(ctx); file == nil {
        if val := autoGet(ctx, ">"); val != nil {
            s := pc.traves.add(ctx, traveFail, target.Value)
            s.error = traveTargetNotDefinedFile
            s.depend = val
        } else if true {
            erro(ctx, "not a file: %v (%T)", target.Value, target.Value)
            errostack(ctx, 8).debug(64)
        }
        return
    } else if filename == "" {
        errostack(of(ctx,target), 3, "%v: empty fullname", target).debug(32)
        return
    }

    var ( bytes []byte ; err error )
    if bytes, err = ioutil.ReadFile(filename); err == nil {
        var s string
        if opts.head != nil { s = opts.head.Strval(ctx) }
        if len(bytes) > 0   { s += string(bytes) }
        if opts.foot != nil { s = opts.foot.Strval(ctx) }
        ctx.autoSet("-", MakeString(ctx.Position(), s))
        ctx.autoSet("-file", file)
    } else {
        brk := pc.traves.add(ctx, traveFail, target)
        brk.error = err
    }
    if opts.debug>0 && err != nil {
        warn(ctx, "%v: %v ; stems=%v\n", target, err, ctx.stems())
        warnstack(ctx, 5).debug(opts.debug)
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

type modifierUpdateFileOpts struct {
    generalOpts
    path   bool `p,path,md,makedir,make-dir,mp,makepath,make-path`
    zero   bool `z,zero;e,empty;az,allow-zero;ae,allow-empty`
    keep   bool `k,keep;keep-file`
    append bool `a,app,append,append-content`
    mode os.FileMode "m,mode"
}
func (ctx modifier) updatefile(args... Value) (result Value) {
    var (
        opts = modifierUpdateFileOpts{ mode: os.FileMode(0640) }
        filename string
        target as
    )
    args = parseOpts(ctx, &opts, plain, args...)
    if len(args) > 1 { opts.mode = permVal(ctx, args[1], 0600) }
    if len(args) > 0 { target.Value = args[0] }
    if target.trivial() { target.Value = autoGet(ctx, "@") }
    if t := target.Value; target.trivial() {
        errostack(ctx, 5, "no file target to update").debug(16)
    } else if opts.fullname {
        var ( f *File ; y bool )
        if f, filename, y = target.fullnameOpt(ctx); !y || !filepath.IsAbs(filename) {
            var s string
            if t := closureFiles(ctx, filename, true); len(t) > 0 { f, s = t[0], t[0].fullname() }
            if s != "" && filepath.IsAbs(s) { filename = s } else if f != nil {
                errostack(ctx, 5, "%v: incorrect fullname (%T, %s)\n", t, t, f.fullname()).debug(16)
            } else if true {
                var m = ctx.universe().unmap(ctx, filename)
                errostack(ctx, 5, "%v: not a file (%T, %s, %v, %v)\n", t, t, filename, m, files(ctx, filename)).debug(16)
            } else if true {
                errostack(ctx, 5, "%v: not a file (%T, %s)\n", t, t, filename).debug(16)
            } else {
                warnstack(ctx, 5, "%v: not a file (%T, %s)\n", t, t, filename).debug(16)
            }
        }
    } else {
        switch p := t.(type) {
        case *File: filename = p.fullname()
        case *Path: filename = p.Strval(ctx)
        default:    filename = target.Strval(ctx)
            if file := file(ctx, filename); file != nil {
                target.Value, filename = file, file.fullname()
            }
        }
    }

    if opts.debug > 0 {
        warnstack(ctx, 5, "update-file: %v (fullname=%v, project=%v)",
            target, filename, ctx.Project()).debug(opts.debug)
    }
    if opts.path { // Make path (mkdir -p)
        if p := filepath.Dir(filename); p != "." && p != "/" {
            if fi, _ := os.Stat(p); fi != nil && !fi.IsDir() {
                if e := os.Remove(p); e != nil {
                    errostack(ctx, 5, "%v (%T %v)", e, target, target).debug(16)
                }
            }
            if e := os.MkdirAll(p, os.FileMode(0755)); e != nil {
                if proj := ctx.Project(); proj != nil {
                    info(ctx, "%v: %v %v", filename, proj, ctx.universe().unmap(ctx, filename))
                    info(ctx, "%v: %v %v", filename, proj, proj.file(ctx, filename))
                    errostack(ctx, 5, "%v: %v (%T %v)", filename, e, target, target).debug(16)
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
    if val := autoGet(ctx, "-"); val == nil {
        // no buffer value
    } else if content = val.Strval(ctx); false && strings.Contains(content, `"\"`) {
        prompt(ctx, "%v: %T\n", filename, val).debug(1)
        fail(ctx.Position(), "%s", filename)
    } else {
        exeres, _ = val.(*execResult)
    }

    if content != "" {
        // good to go
    } else if opts.zero {
        if opts.verbose || opts.debug > 0 {
            warnstack(ctx, 3, "empty content for '%v'", target).debug(opts.debug)
        }
    } else {
        if opts.keep {
            // keep file
        } else if file := stat(at(ctx, target.Position()), filename, "", ""); file != nil && file.info != nil && file.info.Size() == 0 {
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

        if v := autoGet(ctx, "-"); v == nil {
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
    if opts.verbose {
        defer func(st time.Time) {
            var s string
            if err != nil { s = err.Error() } else if same {
                if true { return } else { s = "unchanged" }
            } else if opts.debug > 0 {
                s = fmt.Sprintf("changed (%d bytes, %s)", wrote, filename)
            } else {
                s = fmt.Sprintf("changed (%d bytes)", wrote)
            }
            //printEnteringDirectory(ctx)
            prompt(ctx, "update %v …… %s (in %v)\n", trimPromptString(target.String()), s, time.Now().Sub(st)).
                debug(opts.debug)
        } (time.Now())
    }

    if same, err = crc64CheckFileModeContent(ctx, filename, []byte(content), opts.mode); err != nil {
        if _, ok := err.(*os.PathError); ok {
            err = nil // discard path error (e.g. no such file or directory)
        } else {
            erro(ctx, "crc64 checksum failed: %v", err).debug(1)
            return
        }
    } else if same {
        //removeCallerUpdated(ctx, target) // remove timestamp updated
        result = stat(ctx, filename, "", "")
        return
    }

    printEnteringDirectory(ctx)

    // Create or update the file with new content

    var (
        f *os.File
        pc = ctx.pc()
        m = os.O_RDWR | os.O_CREATE
    )
    if opts.append { m |= os.O_APPEND } else { m |= os.O_TRUNC }
    if f, err = os.OpenFile(filename, m, opts.mode); err != nil {
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

            if file := stat(ctx, filename, "", ""); file == nil {
                prompt(ctx, "%s: invalid file\n", filename)
                errostack(ctx, 6, "%v: invalid file '%s'", ctx.Project(), filename).debug(1)
                fail(ctx.Position(), "invalid file %s", filename)
            } else {
                var files []*File
                if files, err = file.stamp(ctx); err != nil {
                    erro(ctx, "%v", err).debug(1)
                    return
                } else if false && opts.verbose {
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

type modifierWaitOpts struct {
    generalOpts
    stdout   bool "o,stdout"
    stderr   bool "e,stderr"
    status   bool "s,status"
    trim     bool "t,trim" // trim heading and tailing spaces of the result
    execRes  bool "x,exec"
    noTarget bool `nt,no-target`
    asType string "a,as"
}
func (ctx modifier) _wait(args... Value) (result Value) {
    var (
        opts modifierWaitOpts
        execRes *execResult
    )
    args = parseOpts(ctx, &opts, plain, args...)

    var (
        waitForexecResult = opts.stdout || opts.stderr || opts.status || opts.execRes
        stampCurrentTarget = !opts.noTarget
        target Value = autoGet(ctx, "@")
        err error
    )
    if opts.verbose {
        defer func (st time.Time) {
            var s string; if err != nil { s = "fail" } else { s = "done" }
            prompt(ctx, "Wait %v …… %s, result=%v\n", target, s, execRes).debug(opts.debug, 1)
            if opts.debug>0 { info(ctx, "%v", execRes).debug(opts.debug) }
        } (time.Now())
    }

    // Wait for prerequisites and/or execution
    if _, _, execRes, err = wait(ctx, waitOpts{
        opts.verbose, waitForexecResult, stampCurrentTarget,
    }); execRes == nil { return }

    var (
        pos = ctx.Position()
        a []Value
        s string
        v Value
    )
    if opts.stdout {
        // TODO: warn(ctx, "deprecated (wait -stdout), use (shell -stdout) instead; %v", execRes).debug(1)
        if b := execRes.Stdout.Buf; b != nil { s = b.String() }
        if opts.trim { s = strings.TrimSpace(s) }
        switch opts.asType {
        case "answer": v = MakeAnswer (pos,(s == "yes"))
        case "bool":   v = MakeBoolean(pos,(s == "true"))
        default:       v = MakeString (pos,s)
        }
        a = append(a, v)
    }
    if opts.stderr {
        // TODO: warn(ctx, "deprecated (wait -stderr), use (shell -stderr) instead; %v", execRes).debug(1)
        if b := execRes.Stderr.Buf; b != nil { s = b.String() }
        if opts.trim { s = strings.TrimSpace(s) }
        switch opts.asType {
        case "answer": v = MakeAnswer (pos,(s == "yes"))
        case "bool":   v = MakeBoolean(pos,(s == "true"))
        default:       v = MakeString (pos,s)
        }
        a = append(a, v)
    }
    if opts.status {
        // TODO: warn(ctx, "deprecated (wait -status), use (shell -status) instead; %v", execRes).debug(1)
        a = append(a, MakeInt(pos,int64(execRes.Status)))
    }

    if len(a) > 0 { result = MakeListOrScalar(pos, a) }
    return
}

func reportFileUpdates(ctx Context, files []*File) {
    var start = ctx.pc().start
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

type modifierStampOpts struct {
    generalOpts
    prompt bool "m,prompt"
    next   bool "n,nxt,next"  // traveNext if failed to stamp
    error  bool "e,err,error" // traveErro if failed to stamp
}
func (ctx modifier) stamp(args... Value) (result Value) {
    var opts modifierStampOpts
    args = parseOpts(ctx, &opts, plain, args...)

    var target = getTargetValue(ctx)
    if isNil(target) {
        prompt(ctx, "%v\n", ctx.Project())
        erro(ctx, "stamp(%v) failed", target)
        errostack(ctx, 6, "%v", ctx).debug(12)
        return
    }

    var _, err = target.stamp(ctx)
    if err == nil { return /* Done! */ }

    var pc = ctx.pc()
    var p = prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err)
    if n := opts.debug; n>0 { p.debug(n) }
    if opts.next {
        if opts.verbose { warn(ctx, "%v", err).debug(1) }
        s := pc.traves.add(ctx, traveNext, target)
        s.depend = autoGet(ctx, ">")
        err = nil // discard the error
    } else if opts.error {
        s := pc.traves.add(ctx, traveFail, target)
        s.depend = autoGet(ctx, ">")
        s.error = err
        if false {
            prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err).debug(1)
            erro(ctx, "stamp(%v) error")
            errostack(ctx, 10, "%v", ctx).debug(1)
        } else {
            prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err).debug(1)
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

func (ctx modifier) assert(aa... Value) (result Value) {
    var fails int
    var target = autoGet(ctx, "@")
    var pc = ctx.pc()
    var opts, args = mop[struct {
        generalOpts
        msg string `m,msg,message`
    }](&ctx, expandZero, aa...)

    for _, a := range args {
        if a == nil {
            errostack(ctx, 3, "assert failed: nil arg").debug(16)
            return
        }

        if _, y := a.(*punctuation); y { continue }

        if v := a.expand(ctx, plain); v.True(ctx) {
            continue
        } else if s := opts.msg; s == "" {
            erro(of(ctx, a), "assert failed: %s: %v → %v", typeof(a), a, v)
        } else {
            erro(of(ctx, a), "assert failed: %s %v: %v: %s", typeof(a), a, v, s)
        }

        pc.traves.add(ctx, traveFail, target).
            error = fmt.Errorf("assert failed: %v", a)

        fails += 1
    }
    if fails > 0 { errostack(ctx, 8, "%v: %v", target, args).debug(6) }
    if ctx.flushDiags(true) > 0 { fail(ctx.Position(), "assertion: %v", aa) }
    return
}

func (ctx modifier) cond(args... Value) (result Value) {
    // TODO: make it lisp-like (cond), e.g.:
    //     (cond
    //       ((condition) ...)
    //       (true{} ...))
    var pc = ctx.pc()
    for _, a := range args {
        if a == nil { warn(ctx, "nil arg").debug(1) }
        if a == nil || !a.True(ctx.Context) {
            pc.traves.add(ctx, traveDone, nil)
            return
        }
    }
    return MakeBoolean(ctx.Position(), true)
}

func (ctx modifier) _case(args... Value) (result Value) {
    var opts, _ = mop[struct {
        generalOpts
    }](&ctx, plain)

    var w travekind = traveNext
    for _, a := range args {
        if a.True(ctx.Context) { w = traveCase ; break }
    }

    var pc = ctx.pc()
    var s = pc.traves.add(ctx, w, nil) // trave 'case' or 'next'
    // s.error = fmt.Errorf("%s", msg)
    s.prog = ctx.program()

    if opts.verbose { prompt(ctx, "%v: %v", autoGet(ctx, "@"), w) }
    if opts.debug > 0 { warn(ctx, "%v", w) }
    return
}

func (ctx modifier) predictDirty(args... Value) (result Value) {
    if res := ctx.dirty(ctx, args...); res {
        result = MakePrediction(ctx.Position(), res, /*reason*/"")
    } else {
        var pc = ctx.pc()
        pc.traves.add(ctx, traveDone, nil)
    }
    return
}

func (ctx modifier) predictNoLoop(args... Value) (result Value) {
    var loop bool
    var target = autoGet(ctx, "@")
    for caller := ctx.pc().caller(); caller != nil; caller = caller.caller() {
        var t = autoGet(caller, "@")
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
    result = MakePrediction(ctx.Position(), !loop, s)
    return
}

func (ctx modifier) predictTarget1stVisit(aa ...Value) (result Value) {
    var opts, _ = _opts[struct{
        generalOpts
        silent bool "s,silent"
    }](&ctx, plain, aa...)

    var target = autoGet(ctx, "@")
    if isNil(target) {
        erro(ctx, "target is <nil>").debug(1)
        return
    }

    var num int
    for caller := ctx.pc().caller(); caller != nil; caller = caller.caller() {
        if false {
            var t = autoGet(caller, "@")
            var same = t != nil && target == t
            if !same && false { same = eq(ctx, target, t) }
            if same { num += 1 }
        } else if n := caller.execRec[target]; n > 0 {
            num += n
        }
    }

    var s string
    ;      if opts.silent {
    } else if num == 0  { //s = "zero"
    } else { s = fmt.Sprintf("%v visits", num+1)
    }

    result = MakePrediction(ctx.Position(), num==0, s)
    return
}

func (ctx modifier) predictTargetMaxVisit(aa ...Value) (result Value) {
    var opts, args = _opts[struct{
        generalOpts
        closure bool "c,closure"
        silent bool "s,silent"
    }](&ctx, plain, aa...)

    var nth int64
    for _, a := range args {
        if i, e := a.Integer(ctx); e != nil {
            erro(ctx, "%v: %v", a, i).debug(1)
        } else if nth = i; nth <= 0 {
            erro(ctx, "needs positive number (%v, %s)", a, typeof(a)).debug(1)
            return
        }
    }

    var (
        num int64
        head bool = true
        target = autoGet(ctx, "@")
    )
    if isNil(target) {
        erro(ctx, "target is <nil>").debug(1)
        return
    }
    for caller := ctx.pc().caller(); caller != nil; caller = caller.caller() {
        var ct = autoGet(caller, "@")
        if n := caller.execRec[target]; n > 0 { num += int64(n) }
        if opts.debug > 0 && num > 0 {
            if head { head = false
                prompt(ctx, "  %s: nth(%d)\n", ctx.Position(), nth)
            }
            var pos = caller.program().position
            prompt(ctx, "    %s: %v\n", pos, ct)
        }
    }

    var s string;
    ;      if opts.silent {
    } else if num == 0  { //s = "nth: zero"
    } else if num < nth { //s = "nth"
    } else { s = fmt.Sprintf("%d visits", num+1) }

    result = MakePrediction(ctx.Position(), num<nth, s)
    return
}

type modifierForkOpts struct {
    generalOpts
    workDir string `w,wd,workdir,work-dir`
}
func _modifierFork(ctx Context, args... Value) (result Value, traves travestates) {
    var (
        opts modifierForkOpts
        attr syscall.ProcAttr
        argv []string
        prog = ctx.program()
    )
    args = parseOpts(ctx, &opts, plain, args...)
    for _, a := range args { argv = append(argv, a.Strval(ctx)) }

    if opts.workDir != "" {
        attr.Dir = opts.workDir
    } else if attr.Dir = prog.workDir(ctx); attr.Dir == "" {
        erro(ctx, "empty workdir").debug(1)
        return
    }
    attr.Env, _ = prog.env(ctx)
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
func (ctx modifier) fork(args... Value) (result Value) {
    var (
        prog = ctx.program()
        opts modifierForkOpts
        argv []string
        wd string
    )
    args = parseOpts(ctx, &opts, plain, args...)
    for _, a := range args { argv = append(argv, a.Strval(ctx)) }

    if opts.workDir != "" {
        wd = opts.workDir
    } else if wd = prog.workDir(ctx); wd == "" {
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
    cmd.Env, _ = prog.env(ctx)

    if err = cmd.Run(); err != nil {
        erro(ctx, "fork: %v: %v", exe, err).debug(1)
    } else {
        // TODO: status code, etc.
    }
    return
}

type modifierGitModifiedOpts struct {
    generalOpts
}
func (ctx modifier) gitmodified(args... Value) (result Value) {
    var opts modifierGitModifiedOpts
    args = parseOpts(ctx, &opts, plain, args...)

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
        var pred = MakePrediction(pos, false, "")
        if result = pred; len(args) == 0 {
            pred.bool, pred.reason = true, "modified"
            return
        }
        for _, a := range args {
            var s = a.Strval(ctx)
            for _, v := range sm {
                if false { prompt(ctx, "%s: %s\n%v\n", pos, s, v[1]) }
                if s == string(v[1]) {
                    pred.bool, pred.reason = true, "modified: "+s
                    return
                }
            }
        }
    }
    return
}

type modifierGitAheadOpts struct {
    generalOpts
}
func (ctx modifier) gitahead(args... Value) (result Value) {
    var opts modifierGitAheadOpts
    args = parseOpts(ctx, &opts, plain, args...)

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
        result = MakePrediction(ctx.Position(), true, "Work branch has new commits to push")
    }
    return
}

var (
    onceMutex sync.Mutex
    onceCache0 map[Entry]map[Value]int
    onceCache1 map[*Program]map[Value]int
    onceSHA256Mutex sync.Mutex
    onceSHA256Cache = make(map[HashBytes]int,64)
)

func onceCacheTest0(ctx Context, target Value) (n int) {
    var (
        entry = ctx.entry()
        rec map[Value]int
    )
    if stemmed, ok := entry.(*stemmed); ok {
        entry = stemmed.PatternEntry
    }

    onceMutex.Lock(); defer onceMutex.Unlock()
    if onceCache0 == nil {
        onceCache0 = make(map[Entry]map[Value]int,64)
    }
    if rec, _ = onceCache0[entry]; rec == nil {
        rec = make(map[Value]int)
        onceCache0[entry] = rec
    }

    rec[target] += 1
    n = rec[target]
    return
}

func onceCacheTest1(ctx Context, target Value) (n int) {
    var (
        program = ctx.program()
        rec map[Value]int
    )

    onceMutex.Lock(); defer onceMutex.Unlock()
    if onceCache1 == nil {
        onceCache1 = make(map[*Program]map[Value]int,64)
    }
    if rec, _ = onceCache1[program]; rec == nil {
        rec = make(map[Value]int)
        onceCache1[program] = rec
    }

    rec[target] += 1
    n = rec[target]
    return
}

func onceCacheTest2(ctx Context, target Value) (n int) {
    var (
        program = ctx.program()
        h = sha256.New()
        entry = ctx.entry()
    )
    if stemmed, ok := entry.(*stemmed); ok {
        entry = stemmed.PatternEntry
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
            fmt.Fprintf(h, "%s", t.Strval(ctx))
        }
    }

    var sum HashBytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

func onceSHA256Test(ctx Context, sum HashBytes) (n int) {
    onceSHA256Mutex.Lock()
    n = onceSHA256Cache[sum]+1
    onceSHA256Cache[sum] = n
    onceSHA256Mutex.Unlock()
    return
}

func onceSHA256(ctx Context, target Value, opts *modifierOnceOpts, args... Value) (n int) {
    var (
        program = ctx.program()
        entry = ctx.entry()
        h = sha256.New()
    )
    if stemmed, ok := entry.(*stemmed); ok {
        entry = stemmed.PatternEntry
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
        s, _ := a.fullnameOrStrval(ctx)
        fmt.Fprintf(h, "%s", s)
    }

    var sum HashBytes
    copy(sum[:], h.Sum(nil))
    return onceSHA256Test(ctx, sum)
}

type modifierOnceOpts struct {
    generalOpts
    checksum bool `c,cs,checksum,s,sha,sha256,sum,h,hash`
    forval Value `for` // TODO: (once -for=$@)
}
func (ctx modifier) once(aa ...Value) (result Value) {
    // TODO: (once)           --> once for the Rule, aka entry.doneOnce = true
    // TODO: (once -for=$@)   --> once for $@, aka entry.onces[$(expand $@)] = true
    var opts, args = _opts[modifierOnceOpts](&ctx, plain, aa...)
    var (
        target Value = autoGet(ctx, "@")
        n int
    )

    const onceAlgo = 2 // avaialbe: 0, 1, 2

    if isTrivial(target) {
        errostack(ctx, 5, "once: no target $@, %v", args).debug(16)
        return
    } else if opts.checksum {
        n = onceSHA256(ctx, target, &opts, append([]Value{target}, args...)...)
    } else if onceAlgo == 2 {
        n = onceCacheTest2(ctx, target)
    } else if onceAlgo == 1 {
        n = onceCacheTest1(ctx, target)
    } else {
        n = onceCacheTest0(ctx, target)
    }

    var pc = ctx.pc()
    if n > 1 {
        s := pc.traves.add(ctx, traveDone, target)
        s.error = fmt.Errorf(`executed %d times`, n)
    }

    if opts.debug > 0 {
        warn(ctx, "%T %v %p %v", target, target, target, n)
        warnstack(at(ctx, target.Position()), -1, "%p %v %v", target, target, n).debug(16)
    }

    // TODO: new once algorithm:
    if false {
        type traverseRec struct {
            targets map[Value]int // prerequisites
        }

        var entry = ctx.entry()
        var traverseMap = make(map[Entry]*traverseRec)

        if rec, _ := traverseMap[entry]; false {
            if rec == nil {
                rec = &traverseRec{ make(map[Value]int) }
                traverseMap[entry] = rec
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
