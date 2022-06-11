//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
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
        "io/fs"
        "io/ioutil"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "strings"
        "sync"
        "time"
)

var launchTime = time.Now()

const (
        TheShellEnvarsDef = "shell→envars" // '→' ' → '
        TheShellStatusDef = "shell→status" // status code of execution
)

type modifier struct {
        valbase
        name Value
        args []Value
}
func (m *modifier) refs(ctx Context, v Value) bool {
        if m.name.refs(ctx, v) { return true }
        for _, a := range m.args {
                if a.refs(ctx, v) { return true }
        }
        return false
}
func (m *modifier) expandible(ctx Context, w expandwhat) (res bool) {
        if res = m.name.expandible(ctx, w); !res {
                for _, a := range m.args {
                        if res = a.expandible(ctx, w); res { break }
                }
        }
        return
}
func (m *modifier) expand(ctx Context, _ expandwhat) (Value, error) { return m, nil }
func (_ *modifier) cmp(ctx Context, v Value) (res cmpres) {
        if _, ok := v.(*modifier); ok { res = cmpEqual }
        return
}
func (m *modifier) traverse(ctx Context) (brks breakers) {
        var proj = ctx.Project()
        var verb = options.verbose || options.verboseBreaks
        ctx = positional(ctx, m.position)
        if brks = ctx.program().modify(ctx, m); !brks.has() {
                if n := ctx.countErrors(); n > 0 {
                        var s = fmt.Sprintf("%s: modifier failed with %d errors", m.name, n)
                        brks.add(ctx, breakFail).message = s
                }
        } else if tb := brks.not(breakCase, breakNext, breakDone); verb && tb.has() {
                prompt(ctx, "%v: %s modify failed for %s\n", ctx.entry(), m.name, proj)
                for _, brk := range tb {
                        switch brk.what {
                        case breakErro: warn(ctx, "%v: %s: %v", proj, m.name, brk.error  ).at(brk.pos)
                        case breakFail: warn(ctx, "%v: %s: %v", proj, m.name, brk.message).at(brk.pos)
                        default:        warn(ctx, "%v: %s: %v", proj, m.name, brk.what   ).at(brk.pos)
                        }
                }
                warnstack(ctx, 3, "").debug(6)
        }
        return
}
func (m *modifier) String() (s string) {
        s = "(" + m.name.String()
        for _, a := range m.args {
                s += " " + a.String()
        }
        s += ")"
        return
}

type modifiergroup struct {
        valbase
        modifiers []*modifier
}
func (g *modifiergroup) refs(ctx Context, v Value) (res bool) {
        for _, m := range g.modifiers {
                if m.refs(ctx, v) { res = true; break }
        }
        return
}
func (g *modifiergroup) expandible(ctx Context, w expandwhat) (res bool) {
        for _, m := range g.modifiers {
                if res = m.expandible(ctx, w); res { break }
        }
        return
}
func (g *modifiergroup) expand(ctx Context, _ expandwhat) (Value, error) { return g, nil }
func (_ *modifiergroup) cmp(ctx Context, v Value) (res cmpres) {
        if _, ok := v.(*modifiergroup); ok { res = cmpEqual }
        return
}
func (g *modifiergroup) traverse(ctx Context) (brks breakers) {
        for _, m := range g.modifiers {
                var (
                        ctx = positional(ctx, m.position)
                        proj = ctx.Project()
                )
                if t := m.traverse(ctx); t.has() {
                        brks = append(brks, t...) // collect breakers
                } else {
                        continue
                }
                if t := brks.not(breakCase, breakDone, breakNext); t.has() {
                        if false { // NOTE: only check breakers in func traverse
                                var _, ent, _ = entryStr(ctx, ctx.entry())
                                prompt(ctx, "%v: traverse %s failed, project %s\n", ent, m.name, proj)
                                for _, brk := range t {
                                        switch brk.what {
                                        case breakErro: erro(ctx, "%v: %s: %v", proj, m.name, brk.error).at(brk.pos)
                                        case breakFail: erro(ctx, "%v: %s: %v", proj, m.name, brk.message).at(brk.pos)
                                        default: erro(ctx, "%v: %s: %v", proj, m.name, brk.what).at(brk.pos)
                                        }
                                }
                                errostack(ctx, 3, "%v: %v: %v", proj, m.name, ctx).debug(6)
                        }
                        break
                } else if t = brks.of(breakCase); t.has() {
                        if false { warn(ctx, "%v: %v", proj, m).debug(16) }
                        continue // case selected
                } else if t = brks.of(breakDone, breakNext); t.has() {
                        break // done or try next rule entry
                }
        }
        return
}

func (g *modifiergroup) String() (s string) {
        s = "["
        for i, m := range g.modifiers {
                if i > 0 { s += " " }
                s += m.String()
        }
        s += "]"
        return
}

type (
        ModifierFunc   func(Context, ...Value) (Value, breakers)
        PredictionFunc func(Context, ...Value) (Value, error)
)

var (
        init_modifiers = map[string]ModifierFunc{
                `print`:        modifierPrint,
                `debug`:        modifierDebug,

                `select`:       modifierSelect,

                `env`:          modifierEnv,  // interpreter environments
                `set`:          modifierSet,

                `by`:           modifierSetDirtyPats,
                `dirty-by`:     modifierSetDirtyPats,
                `dirty-opts`:   modifierSetDirtyPats,

                `closure`:      modifierClosure,
                `for`:          modifierFor,

                `cd`:           modifierCD,
                `mkdir`:        modifierMkdir,
                `path`:         modifierPath,

                `sudo`:         modifierSudo,

                `touch`:        modifierTouch,
                `grep`:         modifierGrep,
                `deps`:         modifierDeps,

                `copy-file`:       modifierCopyFile,
                `write-file`:      modifierWriteFile,
                `read-file`:       modifierReadFile,
                `update-file`:     modifierUpdateFile,
                `configure-input`: modifierConfigureInput,
                `configure-file`:  modifierConfigureFile,
                `configure`:       modifierConfigure,

                `wait`:         modifierWait,
                `stamp`:        modifierStamp,

                `check`:        modifierCheck,
                `assert`:       modifierAssert,
                `case`:         modifierCase,
                `cond`:         modifierCond,

                `once`:         modifierOnce,

                `git-ahead`:    modifierGitAhead,
                `git-modified`: modifierGitModified,
        }

        init_predictors = map[string]PredictionFunc{
                `dirty`:            predictionOutdated,
                `outdated`:         predictionOutdated,
                `no-loop`:          predictionNoLoop,
                `target-1st-visit`: predictionTarget1stVisit,
                `target-max-visit`: predictionTargetMaxVisit,
        }

        modifiers = make(map[string]ModifierFunc)
        predictors = make(map[string]PredictionFunc)
        crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)
func init() {
        // Install recursive modifiers here to avoid Go's loop detection.
        for s, m := range init_modifiers  { modifiers [s] = m }
        for s, m := range init_predictors { predictors[s] = m }
}

func RegisterModifiers(m map[string]ModifierFunc) (err error) {
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

func promptShellResult(ctx Context, value Value, n int) (err error) {
        if g, ok := value.(*Group); ok && g != nil {
                if elem := g.Get(0); elem != nil {
                        var str string
                        if str, err = elem.Strval(ctx); err == nil && str == "shell" {
                                if elem = g.Get(n); elem != nil {
                                        if str, err = elem.Strval(ctx); err != nil {
                                                erro(ctx, "stringify '%v' failed: %v", elem, err).of(elem)
                                                return
                                        } else if strings.HasSuffix(str, "\n") {
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
        stdout bool `o,stdout`
        stderr bool `e,stderr`
        reset  bool `r,reset`
}
func modifierPrint(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                pos = ctx.Position()
                opts = modifierPrintOpts{ stderr: true }
                content string
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).at(pos).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).at(pos).debug(1)
                return
        } else if value, found := ctx.autoGet("-"); !found || isNil(value) {
                // ...
        } else if content, err = value.Strval(ctx) ; err != nil {
                erro(ctx, "stringify buffer value failed: %v", err).at(pos).debug(1)
                return
        }
        if opts.stdout { fmt.Fprint(stdout, content) }
        if opts.stderr { fmt.Fprint(stderr, content) }
        if opts.reset  { ctx.autoSet("-", MakeNone(pos)) }
        return
}

type modifierDebugOpts struct {
        cond Value `if,cond`
        info []Value `i,info`
        warn []Value `w,warn`
        error []Value `e,err;er,error`
        checkOutdated bool `d,dirty;cd,checkdirty;cd,check-dirty;co,check-outdated`
}
func modifierDebug(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierDebugOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        if opts.cond != nil {
                if v, e := opts.cond.True(ctx); e != nil {
                        erro(ctx, "truthfy '%v': %v", opts.cond, err).debug(1)
                        return
                } else if !v {
                       return
                }
        }

        var s string
        for _, v := range opts.info {
                if s, err = v.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", err).of(v).debug(1)
                        return
                }
                info(ctx, "%s", s).of(v).debug(1)
        }
        for _, v := range opts.warn {
                if s, err = v.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", err).of(v).debug(1)
                        return
                }
                warn(ctx, "%s", s).of(v).debug(1)
        }
        for _, v := range opts.error {
                if s, err = v.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", err).of(v).debug(1)
                        return
                }
                erro(ctx, "%s", s).of(v).debug(1)
        }
        var (
                target , _ = ctx.autoGet("@")
                depends, _ = ctx.autoGet("^")
                ordered, _ = ctx.autoGet("|")
                grepped, _ = ctx.autoGet("~")
        )
        if len(opts.info) == 0 && len(opts.warn) == 0 && len(opts.error) == 0 {
                var pos = ctx.Position()
                prompt(ctx, "%v: %v; target=%v stems=%v depends=%v", pos,
                        args, target, ctx.stems(), depends).debug(1)
        }
        if opts.checkOutdated && !isNil(target) {
                var tt = target.stat(ctx).mod()
                if tt.IsZero() {
                        info(ctx, "target not exists: %v", target).debug(1)
                        return
                }
                for _, dep := range merge(depends, ordered, grepped) {
                        var dt = dep.stat(ctx).mod()
                        if false { if s := dep.String(); strings.HasSuffix(s, ".o") {
                                info(ctx, "%v -> %T %v, %v", target, dep, dep, dt.Sub(tt)).debug(false, 1)
                        }}
                        if dt.After(tt) {
                                info(ctx, "%v: outdated by %v (%v)", target, dep, dt.Sub(tt)).debug(1)
                        }
                }
        }
        return
}

// select element by index from group result: (select 0)
func modifierSelect(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                pos = ctx.Position()
                value, _ = ctx.autoGet("-")
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).at(pos).debug(1)
                return
        } else if g, ok := value.(*Group); ok && len(args) > 0 {
                var num int64
                if num, err = args[0].Integer(ctx); err != nil {
                        erro(ctx, "integify '%v' failed: %v", args[0], err).at(pos).debug(1)
                } else {
                        result = g.Get(int(num))
                }
        }
        return
}

func modifierEnv(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                program = ctx.program()
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "expand args failed: %v", err).debug(1)
                return
        }

        var def, alt = program.scope.define(ctx, DefVoid, TheShellEnvarsDef, nil)
        if def == nil && alt != nil { def, _ = alt.(*Def) }
        if def == nil {
                erro(ctx, "failed setting %v", TheShellEnvarsDef).debug(1)
                return
        }

        var envars = new(List)
        if !isTrivial(def.value) {
                envars.Elems = merge(def.value)
        }
        for _, a := range args {
                if _, ok := a.(*Pair); ok {
                        envars.Append(a)
                } else {
                        erro(ctx, "%v: not a pair value: %v (%T)", TheShellEnvarsDef, a, a).debug(1)
                        return
                }
        }

        def.value = envars
        return
}

// examples:
//     [(set name=value)]    set $(name) to 'value'
//     [(set name)]          clear $(name)
//     [(set -)]             clear $-
type modifierSetOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
}
func modifierSet(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                none = MakeNone(ctx.Position())
                opts modifierSetOpts
                defs []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        var program = ctx.program()
ForArgs:
        for _, arg := range args {
                var (
                        value Value = none
                        name string
                        def *Def
                )
                switch a := arg.(type) {
                case *Bareword: name = a.string
                case *Pair: // NOTE: Pair.Value is not expanded yet! We need to expand it again.
                        if name, err = a.Key.Strval(ctx); err != nil {
                                erro(ctx, "strval '%v' failed: %v", a.Key, err).debug(1)
                                return
                        } else if value, err = a.Value.expand(ctx, expandPlainValue); err != nil {
                                erro(ctx, "expand '%v' failed: %v", a.Value, err).debug(1)
                                return
                        } else if isNil(value) { value = a.Value }
                        if entry := ctx.entry(); false && name == "@" && entry != nil && entry.String() == "archive" {
                                info(ctx, "%v -> %v", a.Value, value)
                                info(ctx, "%s", ctx).debug(10)
                        }
                case *Flag:
                        if name, err = a.name.Strval(ctx); err != nil {
                                erro(ctx, "strval '%v' failed: %v", a.name, err).debug(1)
                                return
                        } else if value = none; name == "" { name = "-" }
                default:
                        erro(ctx, "%T `%s` is unsupported (try: foo=value)", arg, arg).debug(1)
                        return
                }
                if def = program.scope.FindDef(name); def == nil {
                        erro(ctx, "no such def '%s' (%v, %v)", name, arg, args).debug(16)
                        break ForArgs
                } else if err = def.val(ctx, value); err != nil {
                        erro(ctx, "set def '%s' failed: %v", name, err).debug(1)
                        return
                } else {
                        defs = append(defs, def)
                }
        }
        if len(defs) > 0 { result = MakeListOrScalar(ctx.Position(), defs) }
        return
}

type modifierSetDirtyPatsOpts struct {
        verbose bool `v,verbose`
        pats []Value
}
func modifierSetDirtyPats(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts = ctx.dirtyOpts()
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "%v", err).debug(1)
        } else if opts.pats, err = parseOpts(ctx, opts, args...); err != nil {
                erro(ctx, "%v", err).debug(1)
        }
        if false { warn(ctx, "%T %v %v", ctx.inner(), ctx, opts.pats) }
        return
}

// create closure context for the traversal
type modifierClosureOpts struct {
        dump    bool `d,dump`
        verbose bool `v,verbose`
}
func modifierClosure(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierClosureOpts
                pos = ctx.Position()
                err error
        )
        // Closure the caller program, the context will be restored when execution is finished.
        if t := ctx.traversal(); t != nil && false {
                t.Context = closureWith(t.Context, pos)
        } else if pc := ctx.programCtx(); pc != nil {
                pc.Context = closureWith(pc.Context, pos)
        } else {
                erro(ctx, "needs closure context: %v", ctx).debug(1)
                return
        }

        assert(ctx.closure() != nil, "context not closured: %v", ctx)

        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse closure opts failed: %v", err).debug(1)
                return
        }

        if opts.verbose { info(ctx, "%v: %v", ctx.Project(), ctx).debug(1) }
        if opts.dump { infostack(ctx, -1, "%v: %v", ctx.Project(), ctx).debug(1) }

        var dir string // closure work directory
        if proj := ctx.Project(); proj == nil {
                errostack(ctx, 6, "%T: nil project in the context", ctx).debug(64)
        } else if scope := proj.scope; scope == nil {
                erro(ctx, "empty closure context").debug(1)
        } else if def := scope.FindDef("/"); def == nil {
                erro(ctx, "&/ is undefined").at(scope.position).debug(1)
        } else if dir, err = def.value.Strval(ctx); err != nil {
                erro(ctx, "%v", err).of(def.value).debug(1)
        } else if dir == "" {
                erro(ctx, "&/ is empty").at(scope.position).debug(1)
        } else if !filepath.IsAbs(dir) {
                erro(ctx, "&/ is relative").at(scope.position).debug(1)
        } else if err = enter(ctx, dir); err == nil {
                var program = ctx.program()
                program.project.changedWD = dir
                program.changedWD = dir
        }
        return
}

type modifierForOpts struct {
        // ...
}
func modifierFor(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierForOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse cd opts failed: %v", err).debug(1)
                return
        }
        return
}

type modifierCDOpts struct {
        makePath bool `p,path`
        printEnter bool `e,print-enter`
        printLeave bool `l,print-leave`
}
func modifierCD(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierCDOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse cd opts failed: %v", err).debug(1)
                return
        }

        if opts.printEnter { printEnteringDirectory(ctx) }
        if opts.printLeave { printLeavingDirectory(ctx) }
        if (opts.printEnter || opts.printLeave) && len(args) == 0 { return }
        if len(args) == 1 {
                var dir string
                if dir, err = args[0].Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", args[0], err).debug(1)
                        return
                } else if dir == "" {
                        // TODO: do something special
                        return
                }
                var program = ctx.program()
                if !filepath.IsAbs(dir) {
                        dir = filepath.Join(program.project.absPath, dir)
                }
                if opts.makePath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                                erro(ctx, "make path '%s' failed: %v", dir, err)
                                return
                        }
                }
                if err = enter(ctx, dir); err == nil {
                        program.project.changedWD = dir
                        program.changedWD = dir
                }
        } else {
                erro(ctx, "wrong number of cd args: %v", args).debug(1)
        }
        return
}

type modifierMkdirOpts struct {
        mode os.FileMode `m,mode`
        verbose bool `v,verbose`
}
func modifierMkdir(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts = modifierMkdirOpts{ mode: os.FileMode(0755) }
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge mkdir args failed: %v", err).debug(1)
                return
        }
        if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse mkdir opts failed: %v", err).debug(1)
                return
        }
        if len(args) == 0 {
                var target, _ = ctx.autoGet("@")
                var s string
                if s, err = target.Strval(ctx); err != nil {
                        erro(ctx, "stringify target '%v' failed: %v", target, err).debug(1)
                } else if err = os.MkdirAll(filepath.Dir(s), opts.mode); err != nil {
                        erro(ctx, "make path '%s' failed: %v", s, err).debug(1)
                }
                return
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(ctx); err != nil {
                        erro(ctx, "stringify '%v' failed: %v", a, err).debug(1)
                        break
                }
                if err = os.MkdirAll(s, opts.mode); err != nil {
                        erro(ctx, "make path '%s' failed: %v", s, err).debug(1)
                        break
                }
        }
        return
}

type modifierPathOpts struct {
        // TODO: options required
}
// (path $(dir $@))
// (path /example/path)
func modifierPath(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierPathOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge path args failed: %v", err)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse path opts failed: %v", err)
                return
        }

        if len(args) == 0 {
                var target, _ = ctx.autoGet("@")
                var s string
                if s, err = target.Strval(ctx); err != nil {
                        erro(ctx, "stringify target value '%v' failed: %v", target, err).debug(1)
                } else if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                                erro(ctx, "make path '%s' failed: %v", err).debug(1)
                        }
                }
                return
        }

        for _, arg := range args {
                var s string
                if s, err = arg.Strval(ctx); err != nil {
                        erro(ctx, "stringify arg '%v' failed: %v", arg, err).of(arg).debug(1)
                        break
                }
                if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                        erro(ctx, "make path '%s' failed: %v", s, err).debug(1)
                        break
                }
        }
        return
}

func modifierSudo(ctx Context, args... Value) (result Value, brks breakers) {
        erro(ctx, "TODO: sudo modifier is not implemented yet").at(ctx.Position()).debug(1)
        return
}

func parseDependList(ctx Context, dependList *List) (depends *List, brks breakers) {
        depends = new(List)
        for _, depend := range dependList.Elems {
                switch d := depend.(type) {
                case *List:
                        if dl, err := parseDependList(ctx, d); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        } else {
                                depends.Elems = append(depends.Elems, dl.Elems...)
                        }
                case *ExecResult:
                        if d.Status != 0 {
                                brks.add(ctx, breakFail).message = fmt.Sprintf("bad status %v", d.Status)
                                return // target shall be updated
                        } else {
                                depends.Append(d)
                        }
                case *RuleEntry:
                        switch d.class {
                        case GeneralRuleEntry, PatternRuleEntry, PathPattRuleEntry:
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
        target Value
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
                erro(ctx, "'%v' not exists", g.target).at(g.target.Position()).debug(1)
                return
        }
        var tt time.Time = g.targetInfo.ModTime()
        for _, val := range g.files {
                var file, ok = val.(*File)
                if !ok { 
                        erro(ctx, "'%v' is not file (%T)", file, file).debug(1)
                        return
                }
                if file.info == nil && !file.isSysFile() {
                        var s string
                        if s, err = file.Strval(ctx); err != nil { erro(ctx, "%v", err); return }
                        if file.info, _ = os.Stat(s); file.info == nil { continue }
                        if gc.debug { warn(ctx, "'%v' info is nil (%s)", file, file.fullname()) }
                }
                if file.info == nil {/* ... */} else
                if t := file.info.ModTime(); t.After(tt) {
                        if gc.debug { warn(ctx, "touch %v → %v (%v)", g.target, file, t) }
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
        } else if g.target == file {
                res = true
        } else if s, _ := fullnameOrStrval(ctx, file); s == g.targetFullName {
                res = true
        } else if ctx, ok := g.target.(*File); ok && ctx.name == file.name {
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
                        file, ok := v.(*File)
                        if !ok { continue }
                        fmt.Fprintf(w, "%s|%s|%s\n", file.name, file.sub, file.dir)
                }
        }
}

func searchGreppedName(ctx Context, gp Position, gc *grepctx, sys bool, name string) (file *File) {
        var isAbs, isRel bool
        if isAbs = filepath.IsAbs(name); isAbs {
                file = stat(ctx, name, "", "", nil)
        } else if isRel = isRelPath(name); isRel { // relative to targetDir
                file = stat(ctx, name, "", gc.targetDir, nil)
        } else if file = ctx.Project().FindFile(ctx, name); file != nil && file.exists() {
                return // found existed file
        }

        // System files are not treated as missing nor collected
        // for further updating, just discard them immediately.
        if !sys && file != nil && file.filemap != nil && len(file.filemap.paths) == 1 {
                // system files defined by `files ((foo.xxx) ⇒ -)`
                if f, ok := file.filemap.paths[0].(*Flag); ok {
                        sys = isNone(f.name) || isNil(f.name)
                }
        }
        if!sys && gc.debug {
                erro(ctx, "%v: %v → %v (exists=%v, sys=%v, from %v)\n", ctx.entry(), gc.target, name, file.exists(), sys, ctx.Project()).debug(1)
        }
        if sys || file.exists() { return }

        // relative to target directory
        var alt = stat(ctx, name, "", gc.targetDir)
        if alt != nil { file = alt; return }

        // Check for bare non-system sub-paths:
        //   foo/bar/name.xxx
        // We search base name 'name.xxx' again:
        var s = filepath.Dir(name) // e.g: foo/bar

        // Search 'name.xxx' and check dir for
        // 'foo/bar' suffix. We use it if found.
        alt = ctx.Project().FindFile(ctx, filepath.Base(name))
        if alt != nil && strings.HasSuffix(alt.dir, PathSep+s) {
                dir := strings.TrimSuffix(alt.dir, PathSep+s)
                ok1 := alt.change(dir, s, alt.name) // <dir>, foo/bar, name.xxx
                ok2 := alt.change(dir, "", name) // <dir>, "", foo/bar/name.xxx
                file = alt
                if enable_assertions {
                        assert(ok1, "unchanged: %s %s %s", dir, s, alt.name)
                        assert(ok2, "unchanged: %s %s", dir, alt.name)
                }
        } else if file == nil {
                for _, inc := range gc.incs {
                        if s, e := inc.Strval(ctx); e != nil {
                                erro(ctx, "strval '%v' failed: %v", inc, e).of(inc).debug(1)
                        } else if file = stat(ctx, name, "", s); file != nil {
                                if false { info(ctx, "%v in %v", file, inc).debug(1) }
                                return
                        }
                }
                if file == nil { file = stat(ctx, name, "", "", nil) }
                warn(ctx, "'%s' not found in %v", name, ctx.Project()).at(gp)
                warn(ctx, "grepped '%s' has no target dir in %v", name, ctx.Project())
                warn(ctx, "from project %v (for %v)", ctx.Project(), name).at(ctx.Project().position).debug(8)
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
                        var s string
                        if s, err = file.Strval(ctx); err != nil {
                                erro(ctx, "strval '%v' failed: %v", file, err).debug(1)
                                return
                        }
                        if file.info, err = os.Stat(s); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                        if false || gc.debug {
                                warn(ctx, "'%v' info is nil (%s)", file, file.fullname()).debug(1)
                        }
                }
                if file.info == nil {/* ... */} else
                if tv := file.info.ModTime(); tv.After(tt) {
                        if true || gc.debug {
                                warn(ctx, "touch %v → %v (%v)", gc.target, file, tv).debug(1)
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
                info(ctx, "%s: `%s` not found", ctx.Project().name, name).at(gp)
        } else if !file.exists() {
                info(ctx, "%s: `%s` file not existed", ctx.Project().name, name).at(gp)
        }
        return
}

func tempFile(ctx Context, prefix, hashee0 string, hasheeN... interface{}) (file *File, err error) {
        var nameHash = sha256.New() // HashByte -> [sha256.Size]byte
        if _, err = fmt.Fprint(nameHash, prefix, hashee0); err != nil {
                erro(ctx, "hashing failed: %v", err).debug(1)
        } else if _, err = fmt.Fprint(nameHash, hasheeN...); err != nil {
                erro(ctx, "hashing failed: %v", err).debug(1)
        } else if nameSum := nameHash.Sum(nil); len(nameSum) != sha256.Size {
                erro(ctx, "hash sum invalid: %v", len(nameSum)).debug(1)
        } else if project := ctx.Project(); project == nil {
                erro(ctx, "current project is nil: %v", ctx).debug(1)
        } else {
                // Make names like .deps/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
                // .deps/??/??/????????????????????????????????????????????????????????????
                // .grep/??/??/????????????????????????????????????????????????????????????
                // .cache/??/??/????????????????????????????????????????????????????????????
                file = project.matchTempFile(ctx, filepath.Join(prefix, // e.g. ".deps", ".grep"
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
        } else if filename, err = fullnameOrStrval(ctx, file); err != nil {
                erro(ctx, "get .deps temp filename failed: %v", err).debug(1)
        }
        return
}

func getSavedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
        var ( file *File )
        if file, err = tempFile(ctx, ".grep", targetFullName); err != nil {
                erro(ctx, "get .grep temp file failed: %v", err).debug(1)
        } else if filename, err = fullnameOrStrval(ctx, file); err != nil {
                erro(ctx, "get .grep temp filename failed: %v", err).debug(1)
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

        var file, ok = gc.target.(*File)
        if !ok {
                file = stat(ctx, gc.targetFullName, "", "")
                if file != nil { gc.target = file }
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
                                warn(ctx, "%s is nil file", name).at(gp)
                                warn(ctx, "grepped %s is nil", name)
                                warn(ctx, "from project %v", ctx.Project()).at(ctx.Project().position).debug(6)
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
                                        warn(ctx, "%s is nil file", name).at(gp)
                                        warn(ctx, "grepped %s is nil", name)
                                        warn(ctx, "from project %v", ctx.Project()).at(ctx.Project().position).debug(6)
                                }
                                continue ForScan // found one
                        }
                }
        }
        return
}

func grep(ctx Context, gc *grepctx) (err error) { // TODO: using ctx.grepping() to replace grepctx
        var targetName string
        switch v := gc.target.(type) {
        case *File:
                targetName = v.name
                gc.targetInfo = v.info
                gc.targetFullName = v.fullname()
                gc.targetDir = filepath.Dir(gc.targetFullName)
                if v.isSysFile() { return }
        default:
                gc.targetDir = ctx.Project().absPath
                if targetName, err = v.Strval(ctx); err != nil {
                        erro(ctx, "strval grep target '%v' failed: %s", v, err).debug(1)
                        return
                }
                if filepath.IsAbs(targetName) {
                        gc.targetFullName = targetName
                } else {
                        gc.targetFullName = filepath.Join(gc.targetDir, targetName)
                }
                if file := stat(ctx, gc.targetFullName, "", ""); file == nil {
                        erro(ctx, "grep: '%s' not found (%v)", gc.targetFullName, gc.target).of(gc.target).debug(16)
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
                if gc.debug { erro(ctx, "%v (done %v)", gc.targetFullName, n).debug(1) }
                return
        }

        //var infos = strings.Contains(gc.targetFullName, "...")
        const infos = false

        if false { defer un(tt(t_traverse, ctx.traversal(), gc.target)) }

        defer func(restore []Value) {
                var t = ctx.traversal()
                var touch = gc.greptouch // copy greptouch value
                if len(touch.files) > 0 {
                        grepcacheM.Lock()
                        grepcache[gc.targetFullName] = touch.files
                        grepcacheM.Unlock()
                } else if false {
                        var gp Position
                        gp.Filename, gp.Line = gc.targetFullName, 1
                        warn(ctx, "grebbed zero files").at(gp)
                        warn(ctx, "grebbed zero files: %v", gc.targetFullName).debug(6)
                }
                gc.files = restore
                if gc.debug { erro(ctx, "grepped: %s → %v (grepped=%v) (saved=%s)\n", gc.target, touch.files, len(t.grepped), gc.savedGrepFile).debug(1) }
                for _, gc.target = range touch.files {
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
                if gc.debug { erro(ctx, "grepcache: %v → %v", gc.targetFullName, gc.files).debug(1) }
                return
        } else if infos {
                info(ctx, "grepcache: %s files=%d", gc.targetFullName, len(gc.files)).debug(1)
        }

        if savedGrepFileLoaded, err = loadSavedGrepFile(ctx, gc); err != nil {
                erro(ctx, "load saved grepfile failed: %v", err).debug(1)
                return
        } else if savedGrepFileLoaded && len(gc.files) > 0 {
                if infos { info(ctx, "loadSavedGrepFile: %v files=%d grepped=%d",
                        gc.targetFullName, len(gc.files), len(ctx.traversal().grepped)).debug(1) }
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
        debug bool `d,debug`
        verbose bool `v,verbose`
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
func modifierGrep(ctx Context, args... Value) (result Value, brks breakers) {
        if false && options.noDepsGrep || options.noGrep {
                return
        }

        var (
                gc grepctx
                err error
        )
        gc.fileinc = true // grep files by default
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge grep args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &gc.modifierGrepOpts, args...); err != nil {
                erro(ctx, "parse grep args failed: %v", err).debug(1)
                return
        } else if gc.incs, err = expandmerge2(ctx, expandPlainValue, gc.incs...); err != nil {
                erro(ctx, "expand grep incs failed: %v", err).debug(1)
                return
        }
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
                target, _ = ctx.autoGet("@")
                targets = args
                grepped = ctx.traversal().grepped
        )
        if len(targets) == 0 { if isNil(target) || isNone(target) {
                erro(ctx, "no grep target").debug(1)
                return
        } else {
                targets = append(targets, target)
        }}

        if gc.debug {
                warn(ctx, "grep files: %v %v %v\n", target, gc.rxs, args).debug(1)
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

        var t = ctx.traversal()
        var tar = target
        defer func(v bool) { t.grepping = v } (t.grepping)
        t.grepping = true

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

                gc.target, t.grepped = target, nil
                if err = grep(ctx, &gc); err != nil {
                        erro(ctx, "grep files from %v failed: %v", target, err).debug(1)
                        return
                } else if gc.noTraverse {
                        // does nothing
                } else if len(t.grepped) > 0 {
                        for _, val := range t.grepped {
                                if brks = val.traverse(ctx); !brks.has() { continue }
                                for _, brk := range brks {
                                        switch brk.what {
                                        case breakFail: erro(ctx, "broken traversal for grepped %v failed: %v", val, brk.message).at(brk.pos)
                                        case breakErro: erro(ctx, "broken traversal for grepped %v with error: %v", val, brk.error).at(brk.pos)
                                        default: erro(ctx, "broken traversal for grepped %v: %v (%v)", val, brk.message, brk.what).at(brk.pos)
                                        }
                                }
                                erro(ctx, "broken traversal for grepped %v from %v", val, target)
                                errostack(ctx, 5, "%v", ctx).debug(16)
                                break ForTarget
                        }
                }
                grepped = append(grepped, t.grepped...)
        }
        t.grepped = grepped

        var pos = ctx.Position()
        if err != nil {
                erro(ctx, "grep files failed: %v", err).debug(1)
        } else if !gc.noTraverse {
                ctx.autoSet("~", MakeNone(pos))
                t.grepped = nil
        } else {
                result = MakeListOrScalar(pos, t.grepped)
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

func parseDeps(ctx Context, targetVal Value, targetStr string, savedDepsFile *File, savedDepsFileName, deps string) (files []Value, brks breakers) {
        const parallel = true
        var (
                proj = ctx.Project()
                targetFullName string
                filesMux sync.Mutex
                firstWord string
                err error
        )
        if targetFullName, err = fullnameOrStrval(ctx, targetVal); err != nil {
                erro(ctx, `fullname "%v" failed: %v`, targetVal, err).of(targetVal).debug(1)
                return
        }

        var findDepFile = func(name string) (file *File) {
                if filepath.IsAbs(name) {
                        file = stat(ctx, name, "", "", nil)
                } else if file = proj.FindFile(ctx, name); file != nil && file.exists() {
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
                missMux, brksMux sync.Mutex
                jobs sync.WaitGroup
        )
        var depFile = func(ctx Context, depPos Position, word string) {
                var dc = depContext{diagContext{ Context: ctx }}; ctx = &dc
                if parallel { defer func() {
                        checkPanicsErrors(ctx, true/* don't call checkErrors */)
                        if len(dc.points) > 0 { dc.inner().diagnostic().nest(dc.points) }
                        jobs.Done() // minus 1
                }() }
                if i := strings.Index(word, " "); i > 0 {
                        warn(ctx, "ignore dep with spaces: %v", word).debug(1)
                } else if file := findDepFile(word); file == nil {
                        prompt(ctx, "%v: unknown dep\n", file)
                        if savedDepsFile != nil {
                                warn(ctx, "unknown dep '%v' for '%v'", word, firstWord)
                                warn(ctx, "from here: %s", word).at(depPos)
                                if filepath.IsAbs(firstWord) {
                                        var wp Position
                                        wp.Filename, wp.Line = firstWord, 1
                                        warn(ctx, "in here: %v", word).at(wp)
                                }
                                warn(ctx, "for project %v", proj).at(proj.position)//.debug(6)
                        } else {
                                erro(ctx, "unknown dep '%v' for '%v'", word, firstWord)
                                erro(ctx, "from here: %s", word).at(depPos)
                                if filepath.IsAbs(firstWord) {
                                        var wp Position
                                        wp.Filename, wp.Line = firstWord, 1
                                        erro(ctx, "in here: %v", word).at(wp)
                                }
                                erro(ctx, "for project %v", proj).at(proj.position)//.debug(6)
                        }
                } else if ignored(file.fullname()) {
                        //continue // dep is the target itself
                } else if tb := file.traverse(ctx); !tb.has() {
                        addFile(file)
                } else if tb = tb.not(breakCase, breakDone, breakNext); tb.has() {
                        prompt(ctx, "%v: missing dep\n", file)
                        if savedDepsFile != nil {
                                var s = filepath.Base(file.name)
                                warn(ctx, `%v: missing "%v"`, targetVal, s).at(depPos)
                                warnstack(ctx, 3, "%v: (%T):", proj, ctx).debug(4)
                        } else {
                                brksMux.Lock()
                                brks = append(brks, tb...)
                                brksMux.Unlock()
                                erro(ctx, `%v: missing "%v"`, targetVal, file).at(depPos)
                                for _, brk := range tb {
                                        switch brk.what {
                                        case breakFail: erro(ctx, `%v: broken for "%s": %v`, proj, targetVal, brk.message).at(brk.pos)
                                        case breakErro: erro(ctx, `%v: broken for "%s", error: %v`, proj, targetVal, brk.error).at(brk.pos)
                                        default:        erro(ctx, `%v: broken for "%s": %v (%v)`, proj, targetVal, brk.message, brk.what).at(brk.pos)
                                        }
                                }
                                errostack(ctx, 5, "%v: (%T):", proj, ctx).debug(16)
                        }
                } else {
                        addFile(file)
                }
                var n int
                if savedDepsFile == nil {
                        if n = dc.checkErrors(true); n > 0 { // aka. dc.points = nil
                                var s = trimPromptString(targetVal.String())
                                prompt(ctx, "%v: %d errors counted\n", word, n)
                                erro(ctx, `%v: %d errors for "%s", dep "%s"`, proj, n, s, word)
                                errostack(ctx, 5, `%v: %v`, ctx).debug(6)
                        }
                } else {
                        if n = dc.countErrors(); n > 0 {
                                // reset to reduce diags as we wish to continue with the errors
                                dc.points, dc.errs = nil, 0
                                var s = trimPromptString(targetVal.String())
                                prompt(ctx, "%v: %d errors counted\n", word, n)
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
                                        return nil, nil // requests to update savedDepsFile
                                } else if firstDepFile.info.ModTime().After(savedDepsFile.info.ModTime()) {
                                        return nil, nil // requests to update savedDepsFile
                                }
                                if parallel {
                                        if false { info(ctx, "spawn %v", ctx) }
                                        jobs.Add(1); go depFile(ctx.spawn(), depPos, word)
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
                        for s, p := range missing { erro(ctx, `missing "%v"`, s).at(p) }
                        erro(ctx, `%v: "%v" %d deps missing in "%v"`, proj, targetVal, len(missing), savedDepsFileName)
                        errostack(ctx, 3, "%v", ctx).debug(10)
                        fail(ctx.Position(), "removed %s", savedDepsFileName)
                } else {
                        for s, p := range missing { warn(ctx, `missing "%v"`, s).at(p) }
                        warn(ctx, `%v: "%v" missing %d deps (%v in total)`, proj, targetVal, len(missing), len(files))
                        warnstack(ctx, 3, "%T:", ctx).debug(6)
                        files, brks = nil, nil // To update savedDepsFileName
                }
        }
        return
}

func loadSavedDepsAndCheckOutdated(ctx Context, args []string) (savedDepsFileName string, files []Value, brks breakers) {
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
        } else if files, brks = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, string(savedDepsBytes)); len(files) > 0 {
                if false { info(ctx, "loaded deps %s (%d files)", savedDepsFileName, len(files)).debug(true, 1) }
                var savedDepsFileModTime = savedDepsFile.info.ModTime()
                for _, val := range files { if file, ok := val.(*File); !ok {
                        // ignore
                } else if file.info.ModTime().After(savedDepsFileModTime) {
                        files = nil // need to reload if outdated
                        return
                }}
        }
        return
}

func traverseMissingDep(ctx Context, dep string) (res bool, brks breakers) {
        var (
                okay bool
                fullname string
                proj = ctx.Project()
        )
        if proj == nil {
                prompt(ctx, "%s: traverse dep failed, project %v\n", dep, proj)
                erro(ctx, "%s: no current project for dep", dep)
                errostack(ctx, 5, "%s: %v", dep, ctx).debug(10)
                return
        } else if file := proj.FindFile(ctx, dep); file == nil {
                if false {
                        // FIXME: traverse won't work with 'nil' target value
                        okay, brks = traverse(ctx, nil, dep)
                } else {
                        prompt(ctx, "%s: dep is unknown file; project %v\n", dep, proj)
                        erro(ctx, "%v: %s is unknown file", proj, dep)
                        errostack(ctx, 5, "(%T):", ctx).debug(24)
                        fail(ctx.Position(), "dep '%s' is not file", dep)
                }
                fullname = dep
        } else {
                brks = file.traverse(ctx)
                okay = !brks.has(breakErro, breakFail) && file.exists()
                fullname = file.fullname()
        }
        if brks.has(breakCase, breakNext, breakDone) {
                brks = brks.not(breakCase, breakNext, breakDone)
                // TODO: for _, brk := range tb { ... }
        }
        if brks.has() {
                prompt(ctx, "%s: traverse dep failed (okay=%v), project %v\n", fullname, okay, proj)
                for _, brk := range brks {
                        switch brk.what {
                        case breakErro: erro(ctx, "%v: missing %v: %v", proj, dep, brk.error  ).at(brk.pos)
                        case breakFail: erro(ctx, "%v: missing %v: %v", proj, dep, brk.message).at(brk.pos)
                        default       : erro(ctx, "%v: missing %v: %v", proj, dep, brk.what   ).at(brk.pos)
                        }
                }
                errostack(ctx, 5, "%v: %v", proj, ctx).debug(10)
        } else {
                res = okay
        }
        return
}

func traverseMissingDeps(ctx Context, lastTry string, errBytes []byte) (res bool, tried string, brks breakers) {
        const promptErrors bool = false
        const promptBeforeTraverse bool = promptErrors && true
        for _, rx := range knownerrors {
                var all [][][]byte = rx.FindAllSubmatch(errBytes, -1)
                if all != nil { for _, m := range all {
                        if rx == rxFatalErrorFileNotFound {
                                if promptBeforeTraverse { prompt(ctx, "%s\n", m[0]).debug(6) }
                                if dep := string(m[4]); dep == lastTry {
                                        return false, "", nil
                                } else if res, brks = traverseMissingDep(ctx, dep); !res || brks.has() {
                                        var (
                                                s, l, c = string(m[1]), string(m[2]), string(m[3])
                                                pos = convPosition(s, l, c)
                                        )
                                        prompt(ctx, "%s: dep missing, project %v\n", m[4], ctx.Project())
                                        prompt(ctx, "%s\n", m[0]) // prompt the entire error line
                                        erro(ctx, "%v", ctx).at(pos).debug(1)
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
func (mdc *modifierDepsContext) String() string {
        if fullContextStringer {
                return fmt.Sprintf("deps{%s}", mdc.Context)
        } else {
                return mdc.Context.String()
        }
}
func (mdc *modifierDepsContext) spawn() Context { return &modifierDepsContext{mdc.Context.spawn()} }
//func (mdc *modifierDepsContext) appendCallerUpdated() bool { return false }
func (mdc *modifierDepsContext) mustExists() bool { return true }

type modifierDepsOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
        useClang bool `cl,clang`
        useGcc bool `g,gcc`
        addMissing bool `am,add-missing;mg,missing-goal;MG,MissingGoal`
        lang string `l,lang;lan,language`
        flags []Value `f,flags;o,opts`
        cc string `c,cc;c,compiler`
}
func modifierDeps(ctx Context, args... Value) (result Value, brks breakers) {
        if options.noDepsGrep || options.noDeps {
                return
        }

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
        } else if targetStr == "" {
                erro(ctx, "target '%v' is empty", targetVal).debug(1)
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse deps args failed: %v", err).debug(1)
                return
        } else if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge deps args failed: %v", err).debug(1)
                return
        }

        var files []Value
        if opts.verbose {
                defer func(ts time.Time) {
                        var s string
                        if target, _ := ctx.autoGet("@"); !isNil(target) { s = target.String() }
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
                _MM, _MG bool
                ca []string
                flags []Value
        )
        if flags, err = expandmerge2(ctx, expandPlainValue, opts.flags...); err != nil {
                erro(ctx, "merge flags failed: %v", err).debug(1)
                return
        }

        for _, f := range flags {
                var s string
                if s, err = f.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", err).debug(1)
                        return
                } else { s = strings.TrimSpace(s) }
                switch s {
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
                var s string
                if s, err = fullnameOrStrval(ctx, a); err != nil {
                        erro(ctx, "strval '%v' failed: %v", err).debug(1)
                        return
                } else { s = strings.TrimSpace(s) }
                if strings.Contains(s, "-v -fPIC -fvisibility-inlines-hidden") {
                        v, _ := a.expand(ctx, expandPlainValue)
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
                savedDepsFileName string
        )
        ctx = &modifierDepsContext{ ctx }
        if savedDepsFileName, files, brks = loadSavedDepsAndCheckOutdated(ctx, ca); brks.has() {
                for _, brk := range brks {
                        switch brk.what {
                        case breakErro: erro(ctx, "borken loading saved deps: %v", brk.error).at(brk.pos)
                        case breakFail: erro(ctx, "borken loading saved deps: %v", brk.message).at(brk.pos)
                        default: erro(ctx, "borken loading saved deps: %v", brk.what).at(brk.pos)
                        }
                }
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
                        if okay, retried, brks = traverseMissingDeps(ctx, retried, stderr.Bytes()); okay && !brks.has() {
                                if false {
                                        var target, _ = ctx.autoGet("@")
                                        warn(ctx, "%v: retry deps '%s'", target).debug(1)
                                }
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
                if files, brks = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, stdout.String()); len(files) == 0 {
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
        if t := ctx.traversal(); t != nil && len(files) > 0 {
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
func modifierTouch(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierTouchOpts // = modifierTouchOpts{ mode: os.FileMode(0755) }
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge touch args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse touch opts failed: %v", err).debug(1)
                return
        } else if len(args) == 0 {
                if target, found := ctx.autoGet("@"); found && !isNil(target) {
                        args = append(args, target)
                }
        }

        var files []*File
        for _, arg := range args {
                var vf []*File
                if err = touch(ctx, arg, uint32(opts.mode), opts.path); err != nil {
                        erro(ctx, "touch '%v' failed: %v", arg, err).debug(1)
                        break
                } else if vf, err = arg.stamp(ctx); err != nil {
                        erro(ctx, "touch '%v' failed: %v", arg, err).debug(1)
                        break
                } else { files = append(files, vf...) }
        }

        var t = ctx.traversal()
        var program = ctx.program()
        if opts.verbose { reportFileUpdates(ctx, t.start, files) }
        if len(program.getModifiers(ctx, "stamp")) > 0 {
                warn(ctx, "no need to use a (stamp) after (touch)").debug(1)
        }
        return
}

type modifierCheckOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
        answer bool `a,answer`
        boolean bool `b,boolean;r,result`
        silent bool `s,slient`
        good bool `g,good`
        file Value `f,file`
        dir Value `d,dir`
}
// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
func modifierCheck(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                pos = ctx.Position()
                opts modifierCheckOpts
                optBreak breakind // breaking with good results
                makeResult func(Position,bool) Value // returns results only if non-nil
                value, _ = ctx.autoGet("-")
                values []Value
                pairs []*Pair
                err error
                res bool
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge check args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse check args failed: %v", err).debug(1)
                return
        }
        if opts.good    { optBreak   = breakDone }
        if opts.answer  { makeResult = MakeAnswer }
        if opts.boolean { makeResult = MakeBoolean }
        if opts.silent && makeResult == nil { makeResult = MakeBoolean }
        for _, arg := range args {
                switch a := arg.(type) {
                case *Pair: pairs = append(pairs, a)
                default:
                        if res, err = arg.True(ctx); err != nil {
                                erro(ctx, "unknown check '%v' (%T)", arg, arg).debug(1)
                        } else if makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else {
                                brks.addf(ctx, optBreak, "value '%v' is false", arg)
                                if opts.verbose {
                                        warn(ctx, "value '%v' is false", arg).debug(1)
                                }
                        }
                }
        }
        if !(isNil(opts.file) || isNone(opts.file)) {
                var ( s string; f *File )
                if f, res = opts.file.(*File); res {
                        if res = f.exists(); !res && opts.verbose {
                                warn(ctx, "file '%v' does not exists", opts.file).of(opts.file).debug(1)
                        }
                } else if s, err = opts.file.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", opts.file, err).debug(1)
                        return
                } else if filepath.IsAbs(s) {
                        if f = stat(positional(ctx, opts.file.Position()), s, "", ""); f != nil {
                                res = f.exists()
                        }
                } else if f = ctx.Project().FindFile(ctx, s); f != nil {
                        res = f.exists()
                }
                if res { res = !f.info.Mode().IsDir() } // .IsRegular()
                if opts.verbose {
                        warn(ctx, "'%v' is file: %v", opts.file, res).of(opts.file).debug(1)
                }
                if makeResult != nil {
                        values = append(values, makeResult(pos, res))
                } else if !res {
                        brks.addf(ctx, optBreak, "value '%v' is not file", opts.file)
                        return
                }
        }
        if !(isNil(opts.dir) || isNone(opts.dir)) {
                var ( s string; f *File )
                if f, res = opts.dir.(*File); res {
                        if res = f.exists(); !res && opts.verbose {
                                warn(ctx, "file '%v' does not exists", opts.dir).of(opts.dir).debug(1)
                        }
                } else if s, err = opts.dir.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", opts.dir, err).debug(1)
                        return
                } else if filepath.IsAbs(s) {
                        if f = stat(positional(ctx, opts.dir.Position()), s, "", ""); f != nil {
                                res = f.exists()
                        }
                } else if f = ctx.Project().FindFile(ctx, s); f != nil {
                        res = f.exists()
                }
                if res { res = f.info.Mode().IsDir() }
                if opts.verbose {
                        warn(ctx, "'%v' is file: %v", opts.dir, res).of(opts.dir).debug(1)
                }
                if makeResult != nil {
                        values = append(values, makeResult(pos, res))
                } else if !res {
                        brks.addf(ctx, optBreak, "value '%v' is not dir", opts.dir)
                        return
                }
        }

        var program = ctx.program()
ForPairs:
        for _, p := range pairs {
                var key, str string
                if key, err = p.Key.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", p.Key, err).of(p.Key).debug(1)
                        return
                }
                switch key {
                case "status":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                brks.addf(ctx, optBreak, "value '%v' is not exec result", value)
                                erro(ctx, "value '%v' (%T) is not exec result", value, value).of(value).debug(6)
                                return
                        } else { /*exeres.wg.Wait()*/ }

                        var num int64
                        if num, err = p.Value.Integer(ctx); err != nil {
                                erro(ctx, "%v", err).of(p.Value).debug(1)
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
                        if opts.debug {
                                var tar, _ = ctx.autoGet("@")
                                var val, _ = ctx.autoGet("-")
                                warn(ctx, "%v: %v", ctx.entry(), tar).at(program.position)
                                warn(ctx, "status=%v", exeres.Status)
                                warn(ctx, "hyphen=%v", val)
                                warn(ctx, "context: %v", ctx).debug(1)
                        }

                        if makeResult != nil {
                                values = append(values, makeResult(pos, good))
                        } else if !good {
                                brks.addf(ctx, optBreak, "bad status (%v) (expects %v)", exeres.Status, p.Value)
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                brks.addf(ctx, optBreak, "not an exec result (%T)", value)
                                erro(ctx, "value '%v' (%T) is not exec result", value, value).of(value).debug(6)
                                return
                        } else { /*exeres.wg.Wait()*/ }

                        if opts.verbose {
                                prompt(ctx, "checking %s (status=%d) … ", key, exeres.Status)
                        }
                        if opts.debug {
                                var tar, _ = ctx.autoGet("@")
                                var val, _ = ctx.autoGet("-")
                                warn(ctx, "%v: %v", ctx.entry(), tar).at(program.position)
                                warn(ctx, "status=%v", exeres.Status)
                                warn(ctx, "hyphen=%v", val)
                                warn(ctx, "context: %v", ctx).debug(1)
                        }

                        var v *bytes.Buffer
                        switch key {
                        case "stdout": v = exeres.Stdout.Buf
                        case "stderr": v = exeres.Stderr.Buf
                        default: unreachable()
                        }

                        if v == nil {
                                brks.addf(ctx, optBreak, "bad %s (expects %v)", key, p.Value)
                                break ForPairs
                        } else if str, err = p.Value.Strval(ctx); err != nil {
                                erro(ctx, "strval '%v' failed: %v", p.Value, err).of(p.Value).debug(1)
                                return
                        } else if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                brks.addf(ctx, optBreak, "bad %s (%v) (expects %v)", key, v, p.Value)
                                break ForPairs
                        }
                case "file", "dir": // file=xxx and dir=xxx, same as -file=xxx and -dir=xxx
                        var ( file *File; res bool )
                        if file, res = p.Value.(*File); res {
                                // ok
                        } else if str, err = p.Value.Strval(ctx); err != nil {
                                erro(ctx, "strval '%v' failed: %v", p.Value, err).debug(1)
                                return
                        } else if filepath.IsAbs(str) {
                                if file = stat(positional(ctx, p.Value.Position()), str, "", ""); file != nil {
                                        // ok
                                }
                        } else if file = ctx.Project().FindFile(ctx, str); file != nil {
                                // ok
                        }
                        switch key {
                        case "file": res = file.info != nil && !file.info.Mode().IsDir()//.IsRegular()
                        case "dir":  res = file.info != nil &&  file.info.Mode().IsDir()
                        default: unreachable()
                        }
                        if makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                brks.addf(ctx, optBreak, "`%v` is not %s", p.Value, key)
                                break ForPairs
                        }
                case "var":
                        var g, ok = p.Value.(*Group)
                        if !ok {
                                brks.addf(ctx, optBreak, "`%v` is not a group value", p.Value)
                                break ForPairs
                        }
                        for _, elem := range g.Elems {
                                switch p := elem.(type) {
                                case *Pair:
                                        var k, a, b string
                                        if k, err = p.Key.Strval(ctx); err != nil { break ForPairs }
                                        var def = program.project.scope.FindDef(k)
                                        if def != nil {
                                                if a, err = p.Value.Strval(ctx); err != nil { break ForPairs }
                                                if b, err = def.value.Strval(ctx); err != nil { break ForPairs }
                                                if res := a != b; makeResult != nil {
                                                        values = append(values, makeResult(pos, res))
                                                } else if !res {
                                                        brks.addf(ctx, optBreak, "`%v` != `%v`", p.Key, p.Value)
                                                        break ForPairs
                                                }
                                        } else if makeResult != nil {
                                                values = append(values, makeResult(pos, false))
                                        } else {
                                                brks.addf(ctx, optBreak, "`%v` is not defined", k)
                                                break ForPairs
                                        }
                                default:
                                        brks.addf(ctx, optBreak, "`%v` unsupported checks", elem)
                                        break ForPairs
                                }
                        }
                default:
                        erro(ctx, "unknown check for %v -> %v", p.Key, p.Value).debug(1)
                        break ForPairs
                }
        }
        if err == nil && len(values) > 0 {
                result = MakeListOrScalar(pos, values)
        }
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
        var def1, def2 *Def
        if true {
                def1 = opts.program.scope.Lookup("1").(*Def)
                def2 = opts.program.scope.Lookup("2").(*Def)
        } else if gs := ctx.Globe().scope; gs != nil {
                def1 = gs.Lookup("1").(*Def)
                def2 = gs.Lookup("2").(*Def)
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
        if opts.head != nil {
                if head, err = opts.head.Strval(ctx); err != nil { erro(ctx, "%v", err); return }
                if false { fmt.Fprintf(stderr, "%s: %v => %s\n", opts.head.Position(), opts.head, head) }
        }
        if opts.foot != nil {
                if foot, err = opts.foot.Strval(ctx); err != nil { erro(ctx, "%v", err); return }
                if false { fmt.Fprintf(stderr, "%s: %v => %s\n", opts.foot.Position(), opts.foot, foot) }
        }

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
func modifierCopyFile(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierCopyFileOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        var target Value
        var source Value
        if len(args) > 0 {
                target = args[0]
        } else {
                target, _ = ctx.autoGet("@")
        }
        if len(args) > 1 {
                source = args[1]
        } else {
                source, _ = ctx.autoGet("<")
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
                if filename, err = target.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", target, err).debug(1)
                        return
                } else if file := project.FindFile(ctx, filename); file != nil {
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
                if srcname, err = source.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", source, err).debug(1)
                        return
                } else if file := project.FindFile(ctx, srcname); file != nil {
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
                if err = copyFile(ctx, file.info, srcname, filename, copts); err != nil {
                        erro(ctx, "%v", err).debug(1)
                }
        } else if opts.recursive {
                if err = copyDir(ctx, srcname, filename, copts); err != nil {
                        erro(ctx, "%v", err).debug(1)
                }
        } else {
                erro(ctx, "`%v` is a directory (use -r to solve it)", source).debug(1)
        }

        if opts.verbose {
                if err != nil {
                        prompt(ctx, "… error\n")
                } else if copts.copied == 0 {
                        prompt(ctx, "… Good (%d files)\n", copts.files)
                } else if copts.copied == 1 {
                        prompt(ctx, "… Copied %d bytes\n", copts.bytes)
                } else {
                        prompt(ctx, "… Copied %d bytes (%d/%d)\n", copts.bytes, copts.copied, copts.files)
                }
        }
        return
}

func modifierWriteFile(ctx Context, args... Value) (result Value, brks breakers) {
        var err error
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        }

        var (
                target, _ = ctx.autoGet("@")
                filename, str string
                f *os.File
        )
        defer func() {
                if err != nil && filename != "" { os.Remove(filename); f = nil }
                if f == nil { brks.add(ctx, breakFail).message = fmt.Sprintf("file %s not generated", target) }
        } ()
        if isNil(target) {
                erro(ctx, "target is undefined").debug(1)
                return
        } else if filename, err = fullnameOrStrval(ctx, target); err != nil {
                erro(ctx, "fullname failed: %v", err).debug(1)
                return
        } else if buffer, _ := ctx.autoGet("-"); isNil(buffer) {
                erro(ctx, "buffer value is nil").debug(1)
                return
        } else if str, err = buffer.Strval(ctx); err != nil {
                erro(ctx, "strval buffer failed: %v", err).debug(1)
                return
        } else if f, err = os.Create(filename); err != nil {
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
        debug bool "d,debug"
        verbose bool "v,verbose"
        fullname bool "f,full,fullname"
        head Value "h,head"
        foot Value "f,foot"
}
func modifierReadFile(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierReadFileOpts
                filename string
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        } else if !opts.fullname {
                // does nothing
        } else if args, err = expandmerge2(ctx, expandFullName, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if false {
                for _, a := range args {
                        warn(ctx, "%v: %T %v", ctx.Project(), a, a).debug(1)
                }
        }

        var target Value
        if n := len(args); n > 1 {
                erro(ctx, "too many files: %v", args).debug(1)
                return
        } else if n == 1 {
                target = args[0]
        } else {
                target, _ = ctx.autoGet("@")
        }

        if isTrivial(target) {
                errostack(ctx, 3, "target for reading is invalid (%T)", target).debug(8)
                return
        } else if filename, err = fullnameOrStrval(ctx, target); err != nil {
                errostack(ctx, 3, "strval '%v' error: %v", target, err).of(target).debug(1)
                return
        } else if filename == "" {
                errostack(ctx, 3, "target filename is empty").of(target).debug(1)
                return
        }

        var bytes []byte
        if bytes, err = ioutil.ReadFile(filename); err == nil {
                var s, v string
                if opts.head != nil {
                        if v, err = opts.head.Strval(ctx); err == nil { s = v } else {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                }
                s += string(bytes)
                if opts.foot != nil {
                        if v, err = opts.foot.Strval(ctx); err == nil { s += v } else {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                }
                ctx.autoSet("-", MakeString(ctx.Position(), s))
        } else if stems := ctx.stems(); false && len(stems) > 0 {
                brks.add(ctx, breakNext).scope = breakTrave
                if opts.debug {
                        warn(ctx, "%v", err).debug(1)
                        warnstack(ctx, -1, "%v", err).debug(1)
                }
        } else {
                brks.add(ctx, breakErro).error = err
                if opts.debug {
                        erro(ctx, "read-file: %T %s (stems=%v)", target, filename, stems)
                        errostack(ctx, 5, "%v", err).debug(36)
                }
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
        debug bool `d,debug`
        verbose bool `v,verbose`
        full bool `f,fn,full,fullname`
        path bool `p,path,md,makedir,make-dir,mp,makepath,make-path`
        zero bool `z,zero;e,empty;az,allow-zero;ae,allow-empty`
        append bool `a,app,append,append-content`
        mode os.FileMode "m,mode"
}
func modifierUpdateFile(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts = modifierUpdateFileOpts{ mode: os.FileMode(0640) }
                filename string
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        } else if !opts.full {
                // does nothing
        } else if args, err = expandmerge2(ctx, expandFullName, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        }

        var target Value
        if len(args) > 0 {
                target = args[0]
        } else {
                target, _ = ctx.autoGet("@")
        }
        if len(args) > 1 { if opts.mode, err = permVal(ctx, args[1], 0600); err != nil {
                erro(ctx, "perm value '%v' failed: %v", args[1], err).of(args[1]).debug(1)
                return
        }}

        // Get target filename
        if opts.full {
                if filename, err = fullnameOrStrval(ctx, target); err != nil {
                        erro(ctx, "fullnameOrStrval: %v", err).debug(1)
                        return
                }
        } else {
                switch p := target.(type) {
                case *File: filename = p.fullname()
                case *Path:
                        if filename, err = p.Strval(ctx); err != nil {
                                erro(ctx, "strval path '%v' failed: %v", p, err).debug(1)
                                return
                        }
                default:
                        if filename, err = target.Strval(ctx); err != nil {
                                erro(ctx, "strval '%v' failed: %v", p, err).debug(1)
                                return
                        } else if file := ctx.Project().FindFile(ctx, filename); file != nil {
                                target, filename = file, file.fullname()
                        }
                }
        }

        if opts.debug {
                warnstack(ctx, 5, "update-file: %v (fullname=%v, project=%v)",
                        target, filename, ctx.Project()).debug(12)
        }
        if opts.path { // Make path (mkdir -p)
                if p := filepath.Dir(filename); p != "." && p != "/" {
                        if fi, _ := os.Stat(p); fi != nil && !fi.IsDir() {
                                if e := os.Remove(p); e != nil {
                                        erro(ctx, "%v", err).debug(1)
                                }
                        }
                        if err = os.MkdirAll(p, os.FileMode(0755)); err != nil {
                                erro(ctx, "%v", err).debug(1)
                                return
                        }
                }
        }

        // Check existed file content checksum
        var (
                content string
                exeres *ExecResult
        )
        if value, found := ctx.autoGet("-"); !found || isNil(value) {
                // no buffer value
        } else if content, err = value.Strval(ctx); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        } else if false && strings.Contains(content, `"\"`) {
                prompt(ctx, "%v: %T\n", filename, value).debug(1)
                fail(ctx.Position(), "%s", filename)
        } else if er, ok := value.(*ExecResult); ok {
                exeres = er
        }

        if content != "" {
                // good to go
        } else if opts.zero {
                if file := stat(positional(ctx, target.Position()), filename, "", ""); file != nil && file.info != nil && file.info.Size() == 0 {
                        file.info = nil
                        if err = os.Remove(filename); err != nil {
                                erro(ctx, "remove file failed: %v", err).debug(1)
                        }
                }
                if exeres != nil {
                        if exeres.Stdout.log != nil {
                                var pos Position
                                pos.Filename = exeres.Stdout.log.filename
                                pos.Line = exeres.Stdout.log.lines + 1
                                erro(ctx, "empty stdout").at(pos)
                        }
                        if exeres.Stderr.log != nil && exeres.Stdout.log != exeres.Stderr.log {
                                var pos Position
                                pos.Filename = exeres.Stderr.log.filename
                                pos.Line = exeres.Stderr.log.lines + 1
                                erro(ctx, "empty stderr").at(pos)
                        }
                }
                if s := target.String(); filepath.IsAbs(s) {
                        erro(ctx, "empty content for '%s'", s).debug(1)
                } else {
                        erro(ctx, "empty content for '%s' (at %s)", s, filename).debug(1)
                }
                return
        } else if opts.verbose || opts.debug {
                warnstack(ctx, 3, "empty content for '%v'", target).debug(6)
        }

        var ( wrote int; same bool )
        if opts.verbose {
                defer func(st time.Time) {
                        var s string
                        if err != nil { s = err.Error() } else if same {
                                if true { return } else { s = "unchanged" }
                        } else if opts.debug {
                                s = fmt.Sprintf("changed (%d bytes, %s)", wrote, filename)
                        } else {
                                s = fmt.Sprintf("changed (%d bytes)", wrote)
                        }
                        //printEnteringDirectory(ctx)
                        prompt(ctx, "update %v …… %s (in %v)\n", trimPromptString(target.String()), s, time.Now().Sub(st)).
                                debug(opts.debug, 6)
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
                m = os.O_RDWR | os.O_CREATE
        )
        if opts.append {
                m |= os.O_APPEND
        } else {
                m |= os.O_TRUNC
        }
        if f, err = os.OpenFile(filename, m, opts.mode); err != nil {
                brks.add(ctx, breakFail).message = fmt.Sprintf("update %v failed", target)
                erro(ctx, "open file failed: %v", err).debug(1)
        } else if f != nil {
                defer func() {
                        if err = f.Close(); err != nil {
                                os.Remove(filename)
                                erro(ctx, "close file '%s' failed: %v", filename, err).debug(1)
                                return
                        }
                        var file = stat(ctx, filename, "", "")
                        if  file == nil {
                                prompt(ctx, "%s: invalid file\n", filename)
                                errostack(ctx, 6, "%v: invalid file '%s'", ctx.Project(), filename).debug(1)
                                fail(ctx.Position(), "invalid file %s", filename)
                        } else {
                                var files []*File
                                if files, err = file.stamp(ctx); err != nil {
                                        erro(ctx, "%v", err).debug(1)
                                        return
                                } else if false && opts.verbose {
                                        var t = ctx.traversal()
                                        reportFileUpdates(ctx, t.start, files)
                                }
                                result = file // resulting the updated file
                        }
                } ()
                if wrote, err = f.WriteString(content); err != nil {
                        erro(ctx, "write content failed: %v", err).debug(1)
                }
        } else {
                brks.add(ctx, breakFail).message = fmt.Sprintf("%v not updated", target)
        }
        return
}

type modifierWaitOpts struct {
        debug bool `d,debug`
        verbose bool "v,verbose"
        stdout bool "o,stdout"
        stderr bool "e,stderr"
        status bool "s,status"
        trim bool "t,trim" // trim heading and tailing spaces of the result
        noTarget bool `nt,no-target`
        execRes bool "x,exec"
        asType string "a,as"
}
func modifierWait(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierWaitOpts
                execRes *ExecResult
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        var (
                waitForExecResult = opts.stdout || opts.stderr || opts.status || opts.execRes
                stampCurrentTarget = !opts.noTarget
                target, _ = ctx.autoGet("@")
        )
        if opts.verbose {
                defer func (st time.Time) {
                        var s string; if err != nil { s = "fail" } else { s = "done" }
                        //prompt(ctx, "Wait %v …… %s, result=%v, updated=%v\n", target, s, execRes, t.updated).debug(opts.debug, 1)
                        prompt(ctx, "Wait %v …… %s, result=%v\n", target, s, execRes).debug(opts.debug, 1)
                        if opts.debug { info(ctx, "%v", execRes).debug(6) }
                } (time.Now())
        }

        // Wait for prerequisites and/or execution
        if _, _, execRes, err = wait(ctx, opts.verbose, waitForExecResult, stampCurrentTarget); execRes != nil {
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
        }
        return
}

func reportFileUpdates(ctx Context, start time.Time, files []*File) {
        for _, file := range files {
                var (
                        mod = file.info.ModTime()
                        d = time.Now().Sub(start)
                )
                if mod.After(start) {
                        if false {
                                prompt(ctx, "Updated %v (%v, ModTime=%v)\n", file, d, mod)
                        } else {
                                prompt(ctx, "Updated %v (%v)\n", file, d)
                        }
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
        next bool "n,nxt,next" // breakNext if failed to stamp
        error bool "e,err;e,error" // breakErro if failed to stamp
        prompt bool "m,prompt"
        verbose bool "v,verbose"
        debug int "d,debug"
}
func modifierStamp(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierStampOpts
                pos = ctx.Position()
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err)
                return
        }

        var target = getTargetValue(ctx)
        if isNil(target) {
                prompt(ctx, "%v\n", ctx.Project())
                erro(ctx, "stamp(%v) failed", target)
                errostack(ctx, 6, "%v", ctx).debug(12)
                return
        } else if /*files*/_, err = target.stamp(ctx); err == nil {
                return
        }

        prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err)
        if opts.next {
                if opts.verbose { warn(ctx, "%v", err).debug(1) }
                brks.add(ctx, breakNext).scope = breakTrave
                err = nil // discard the error
        } else if opts.error {
                brks.add(ctx, breakErro).error = err
                if false {
                        prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err)
                        erro(ctx, "stamp(%v) error")
                        errostack(ctx, -1, "%v", ctx).debug(1)
                } else {
                        prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err)
                        warn(ctx, "stamp(%v) error")
                        warnstack(ctx, -1, "%v", ctx).debug(1)
                }
        } else if stems := ctx.stems(); false && len(stems) == 0 {
                brks.add(ctx, breakNext).scope = breakTrave
                if opts.debug > 0 && err != nil {
                        warn(ctx, "%v", err).debug(1)
                        warnstack(ctx, -1, "%v", err).debug(1)
                } else if false && err != nil {
                        warn(ctx, "%v", err).debug(1)
                }
                err = nil // discard the error
        } else if pos.IsValid() {
                erro(ctx, "failed stamp(%v)", target)
                errostack(ctx, -1, "failed: %v", ctx).debug(10)
        } else if pos = target.Position(); pos.IsValid() {
                ctx = positional(ctx, pos)
                erro(ctx, "failed stamp(%v)", target)
                errostack(ctx, -1, "failed: %v", ctx).debug(10)
        }

        if err != nil { if pe, ok := err.(*fs.PathError); ok {
                erro(ctx, "stamp %s: %v", trimPromptString(pe.Path), pe.Err)
                err = pe.Err
        }}
        return
}

type predictOpts struct {
        and      bool "a,and"
        group    bool "g,group"
        traverse bool "t,traverse;ctx,trave;ctx,target"
        message  string "m,message;m,msg"
        verbose  bool "v,verbose"
        verbose0 bool
}
func predict(ctx Context, args... Value) (result bool, breakScope breaksco, message string, err error) {
        var (
                targetVal, _ = ctx.autoGet("@")
                targetStr string
                num int64
        )
        if isTrivial(targetVal) {
                errostack(ctx, 5, "target is trivial, %v", ctx).debug(10)
                return
        }
        for caller := ctx.traversal().caller(); caller != nil; caller = caller.caller() {
                if tarVal, _ := caller.autoGet("@"); isNil(tarVal) {
                        // top level execution, aka via RuleEntry.Execute(...)
                } else if true {
                        var same = targetVal == tarVal
                        if !same && false {
                                same = (targetVal.cmp(ctx, tarVal) == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n := caller.execRec[targetVal]; n > 0 {
                        num += int64(n)
                }
        }

        var (
                opts predictOpts
                reasons []string
        )
        defer func() { if opts.verbose {
                var status string
                if reasons != nil {
                        s := strings.Join(reasons, ",")
                        if s != "" { status = s }
                }
                if status == "" {
                        var s string
                        if result { s = "Yes" } else { s = "No" }
                        status = fmt.Sprintf("%v (%d)", s, num)
                } else if false {
                        status += fmt.Sprintf(" (result=%v)", result)
                }
                prompt(ctx, "… %s\n", status)
        } } ()

ForArgs:
        for _, arg := range args {
                switch tv := arg.(type) {
                case *String: message = tv.string; continue ForArgs
                case *Compound: if message, err = tv.Strval(ctx); err != nil {
                        erro(ctx, "strval '%v' failed: %v", tv, err).of(tv).debug(1)
                        return
                } else { continue ForArgs }}

                var va []Value
                if va, err = expandmerge2(ctx, expandPlainValue, arg); err != nil {
                        erro(ctx, "merge arg '%v' failed: %v", arg, err).debug(1)
                        return
                } else if va, err = parseOpts(ctx, &opts, va...); err != nil {
                        erro(ctx, "parse opts failed: %v", err).debug(1)
                        return
                } else if len(va) == 0 {
                        continue ForArgs
                } else if len(va) == 1 && isTrivial(va[0]) {
                        continue ForArgs
                }

                if opts.group    { breakScope = breakGroup }
                if opts.traverse { breakScope = breakTrave }
                if opts.verbose && !opts.verbose0 {
                        if targetStr, err = fullnameOrStrval(ctx, targetVal); err != nil {
                                erro(ctx, "fullname-strval '%v' failed: %v", targetVal, err).debug(1)
                                return
                        }
                        prompt(ctx, "checking %v …", filepath.Base(targetStr))
                        opts.verbose0 = true
                }

                if !opts.and && result { break }
                if !opts.and || (opts.and && result) { for i, a := range va {
                        var (
                                name string
                                val Value
                                tru bool
                        )
                        if g, ok := a.(*Group); !ok {
                                // preserve the value of 'a'
                        } else if len(g.Elems) == 0 {
                                erro(ctx, "predictor is empty group").at(g.position).debug(1)
                                return
                        } else if name, err = g.Elems[0].Strval(ctx); err != nil {
                                erro(ctx, "strval predictor failed: %v", err).of(g.Elems[0]).debug(1)
                                return
                        } else if pret, ok := predictors[name]; !ok {
                                erro(ctx, "predictor '%s' undefined (%T %v)", name, a, a).at(g.position).debug(1)
                                return
                        } else if val, err = pret(positional(ctx, g.Elems[0].Position()), g.Elems[1:]...); err != nil {
                                erro(ctx, "prediction '%v' failed: %v", g.Elems[0], err).of(a).debug(1)
                                return
                        } else {
                                a = val // reset the value of 'a'
                        }

                        if a == nil {
                                warn(ctx, "predictor %v is <nil>", arg).debug(1)
                                continue // skip
                        } else if p, ok := a.(*prediction); ok {
                                if p.reason != "" { reasons = append(reasons, p.reason) }
                                tru = p.bool
                        } else if tru, err = a.True(ctx); err != nil {
                                erro(ctx, "truthify '%v' failed: %v", a, err).of(a).debug(1)
                                return
                        } else if tru {
                                reasons = append(reasons, fmt.Sprintf("#%v", i+1))
                        }

                        if opts.and { // logical 'and' mode
                                result = result && tru
                                opts.and = false // reset -and flag
                        } else if tru { // logical 'or' mode
                                result = true
                                break
                        }
                }}
        }
        return
}

// (assert condition,'error message...')
func modifierAssert(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                res bool
                sco breaksco
                msg string
                err error
        )
        if res, sco, msg, err = predict(ctx, args...); err != nil {
                erro(ctx, "prediction %v failed: %v", args, err).debug(6)
        } else if !res {
                if msg == "" {
                        for _, a := range args {
                                erro(ctx, "assertion failed: %v", a).of(a)
                        }
                        errostack(ctx, 8, "(%T):", ctx).debug(6)
                } else {
                        var target, _ = ctx.autoGet("@")
                        var vals, _ = expandmerge2(ctx, expandPlainValue, args...)
                        erro(ctx, "assertion failed: %v (target = %s)", msg, target)
                        erro(ctx, "assertion args: %v", args)
                        erro(ctx, "assertion args: %v (expandmerged)", vals)
                        erro(ctx, "assertion context: %v", ctx).debug(6)
                }
                brk := brks.add(ctx, breakFail)
                brk.message = "assertion failure"
                brk.scope = sco
        }
        return
}

func modifierCond(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                res bool
                sco breaksco
                msg string
                err error
        )
        if res, sco, msg, err = predict(ctx, args...); err != nil {
                erro(ctx, "predict: %v", err).debug(1)
        } else if !res {
                brk := brks.add(ctx, breakDone)
                brk.message = msg
                brk.scope = sco
        }
        return
}

func modifierCase(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                res bool
                sco breaksco
                msg string
                err error
        )
        if res, sco, msg, err = predict(ctx, args...); err == nil {
                brk := brks.add(ctx, breakNext) // next case
                brk.message = msg
                brk.scope = sco
                if res { brk.what = breakCase } // select case
        }
        return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
        var (
                opts = ctx.dirtyOpts()
                deps []Value
                err error
        )
        if len(target.updatedDeps(ctx)) > 0 { return true }
        if v, found := ctx.autoGet("^"); found && !isTrivial(v) { a = append(a, v) }
        if deps, err = expandmerge2(ctx, expandPlainValue, a...); err != nil {
                erro(ctx, "%v", err).debug(1)
                return
        }
        for _, dep := range deps {
                var mat bool = len(opts.pats) == 0
                if !mat { for _, pat := range opts.pats { if mat, _, _ = pat.match(ctx, dep); mat { break }}}
                if mat && (dep.updated(ctx) || dep.stat(ctx).mod().After(target.stat(ctx).mod())) {
                        return true
                }
        }
        return
}

type predictionOutdatedOpts struct {
        checksum bool "c,checksum;c,crc"
        debug bool "d,debug"
        verbose bool "v,verbose"
        verboseUpdated bool "vu,verbose-updated"
        verboseOutdated bool "vo,verbose-outdated"
        silent bool "s,silent"
}
func predictionOutdated(ctx Context, args... Value) (result Value, err error) {
        var (
                opts predictionOutdatedOpts
                target Value
                targetFullname string
                reason string
                outdated bool
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        } else if target, _, _, err = wait(ctx); err != nil {
                erro(ctx, "waiting traversal failed: %v", err).debug(1)
                return
        }

        if false { warn(ctx, "%T %v %v", target, target, target.updatedDeps(ctx)).debug(1) }
        if outdated = len(target.updatedDeps(ctx)) > 0; outdated {
                reason = "updated prerequisites"
        } else if targetFullname, err = fullnameOrStrval(ctx, target); err != nil {
                erro(ctx, "strval '%v' failed: %v", target, err).debug(1)
                return
        } else if outdated = !exists(ctx, target); outdated {
                reason = "target not exists"
        } else if outdated = isDirty(ctx, target, args...); outdated {
                reason = "dirty"
        } else if outdated, err = isRecipesOutdated(ctx); err != nil {
                erro(ctx, "isRecipesOutdated: %v", err).debug(1)
                return
        } else if outdated {
                reason = "recipes changed"
        } else if !opts.checksum {
                // does nothing
        } else if true {
                erro(ctx, "FIXME: check prerequisites against the saved checksums").debug(1)
                return
        } else if depends, _ := ctx.autoGet("^"); !isTrivial(depends) {
                for _, depend := range merge(depends) {
                        var file2 string
                        if isTrivial(depend) {
                                // does nothing
                        } else if file2, err = fullnameOrStrval(ctx, depend); err != nil {
                                erro(ctx, "strval '%v' failed: %v", depend, err).debug(1)
                                return
                        } else if file2 != "" {
                                // see: same, err = crc64CompareFileChecksum(ctx, targetFullname, file2)
                                // TODO.1: load saved checksum for depend, set outdated if no such
                                // TODO.2: calculate checksum for depend and compare with the loaded
                                // TODO.3: set outdated if the two checksums differred
                        }
                }
        }

        if opts.verbose || (opts.verboseOutdated && outdated) || (opts.verboseUpdated && !outdated) {
                var ( t = ctx.traversal(); m, s string )
                if outdated { m = "outdated" } else { m = "updated" }
                if s = time.Now().Sub(t.start).String(); reason != "" {
                        s += "; " + strings.TrimSpace(strings.TrimPrefix(reason, "outdated:"))
                }
                var (
                        ts = trimPromptString(targetFullname)
                        n = len(t.targets) + len(t.grepped)
                )
                prompt(ctx, "%s …… %s (%d files in %s; updated=%v)\n", ts, m, n, s, outdated).debug(opts.debug, 6)
        }

        if opts.silent { reason = "" }
        result = MakePrediction(ctx.Position(), outdated, reason)
        return
}

func predictionNoLoop(ctx Context, args... Value) (result Value, err error) {
        var loop bool
        var target, _ = ctx.autoGet("@")
        for caller := ctx.traversal().caller(); caller != nil; caller = caller.caller() {
                var ct, found = caller.autoGet("@")
                var same = found && target == ct
                if !same && false {
                        same = (target.cmp(ctx, ct) == cmpEqual)
                }
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

type predictionTarget1stVisitOpts struct {
        silent bool "s,silent"
}
func predictionTarget1stVisit(ctx Context, args... Value) (result Value, err error) {
        var ( opts predictionTarget1stVisitOpts )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        var target, _ = ctx.autoGet("@")
        if isNil(target) {
                erro(ctx, "target is <nil>").debug(1)
                return
        }

        var num int
        for caller := ctx.traversal().caller(); caller != nil; caller = caller.caller() {
                if false {
                        var ct, found = caller.autoGet("@")
                        var same = found && target == ct
                        if !same && false {
                                same = (target.cmp(ctx, ct) == cmpEqual)
                        }
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

type predictionTargetMaxVisitOpts struct {
        closure bool "c,closure"
        debug bool "d,debug;d,debug-trace;d,dump"
        silent bool "s,silent"
}
func predictionTargetMaxVisit(ctx Context, args... Value) (result Value, err error) {
        var ( opts predictionTargetMaxVisitOpts )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        var nth int64
        for _, a := range args {
                if nth, err = a.Integer(ctx); err != nil {
                        erro(ctx, "%v", err).debug(1)
                        return
                } else if nth <= 0 {
                        erro(ctx, "needs positive number (%v, %s)", a, typeof(a)).debug(1)
                        return
                }
        }

        var ( num int64; head bool = true )
        var target, _ = ctx.autoGet("@")
        if isNil(target) {
                erro(ctx, "target is <nil>").debug(1)
                return
        }
        for caller := ctx.traversal().caller(); caller != nil; caller = caller.caller() {
                var ct, _ = caller.autoGet("@")
                if n := caller.execRec[target]; n > 0 {
                        num += int64(n)
                }
                if opts.debug && num > 0 {
                        if head { head = false
                                prompt(ctx, "  %s: nth(%d)\n", ctx.Position(), nth)
                        }
                        var pos = caller.program().position
                        prompt(ctx, "    %s: %v\n", pos, ct)
                }
        }

        var s string;
        if opts.silent {
        } else if num == 0  { //s = "nth: zero"
        } else if num < nth { //s = "nth"
        } else { s = fmt.Sprintf("%d visits", num+1) }

        result = MakePrediction(ctx.Position(), num<nth, s)
        return
}

type modifierGitModifiedOpts struct {
        debug bool "d,debug"
        verbose bool "v,verbose"
}
func modifierGitModified(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierGitModifiedOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil {
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
                        var s string
                        if s, err = a.Strval(ctx); err != nil {
                                erro(ctx, "strval '%v' failed: %v", err).debug(1)
                                return
                        }
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
        debug bool "d,debug"
        verbose bool "v,verbose"
}
func modifierGitAhead(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierGitAheadOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                erro(ctx, "parse opts failed: %v", err)
                return
        }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil {
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
        onceCache = make(map[Value]int,64)
        onceSHA256Mutex sync.Mutex
        onceSHA256Cache = make(map[HashBytes]int,64)
)

func onceCacheTest(ctx Context, tv Value) (n int) {
        onceMutex.Lock()
        onceCache[tv] += 1
        onceMutex.Unlock()
        return onceCache[tv]
}

func onceSHA256Test(ctx Context, sum HashBytes) (n int) {
        onceSHA256Mutex.Lock()
        onceSHA256Cache[sum] += 1
        onceSHA256Mutex.Unlock()
        return onceSHA256Cache[sum]
}

func onceSHA256(ctx Context, opts *modifierOnceOpts, args... Value) (result Value, brks breakers) {
        var (
                program = ctx.program()
                h = sha256.New()
                s string
        )
        if true {
                // NOTE: entry and program are unique, since (once) is for runtime, we use their addresses.
                fmt.Fprintf(h, "%p%p", ctx.entry(), program)
        } else {
                fmt.Fprintf(h, "%v%v", ctx.Position(), program.position)
        }

        var err error
        for _, a := range args {
                if s, err = fullnameOrStrval(ctx, a); err != nil {
                        erro(ctx, "strval '%v' failed: %v", a, err).debug(1)
                        return
                } else {
                        if false { info(ctx, "%v", s).debug(true, 1) }
                        fmt.Fprintf(h, "%s", s)
                }
        }

        var sum HashBytes
        copy(sum[:], h.Sum(nil))

        var num = onceSHA256Test(ctx, sum)
        if opts.debug {
                info(ctx, "%T: (once: num=%d)\n", ctx, num)
        } else if opts.verbose {
                prompt(ctx, "once: (num=%d)\n", num)
        }
        if num > 1 { brks.add(ctx, breakDone).message = fmt.Sprintf("once (num=%d)", num) }
        return
}

type modifierOnceOpts struct {
        debug    bool `d,debug`
        verbose  bool `v,verbose`
        checksum bool `c,cs,checksum,s,sha,sha256,sum,h,hash`
        forval Value `for` // TODO: (once -for=$@)
}
func modifierOnce(ctx Context, args... Value) (result Value, brks breakers) {
        // TODO: (once)           --> once for the RuleEntry, aka entry.doneOnce = true
        // TODO: (once -for=$@)   --> once for $@, aka entry.onces[$(expand $@)] = true
        var (
                opts modifierOnceOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                erro(ctx, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                erro(ctx, "parse opts failed: %v", err).debug(1)
                return
        }

        if target, found := ctx.autoGet("@"); !found || isTrivial(target) {
                errostack(ctx, 5, "once: no target $@, %v", args).debug(16)
                return
        } else if opts.checksum {
                result, brks = onceSHA256(ctx, &opts, append([]Value{target}, args...)...)
        } else {
                var n = onceCacheTest(ctx, target)
                if  n > 1 { brks.add(ctx, breakDone).message = fmt.Sprintf(`executed %d times`, n) }
                if opts.debug {
                        warn(ctx, "%T %v %p %v", target, target, target, n)
                        warnstack(positional(ctx, target.Position()), -1, "%p %v %v", target, target, n).debug(16)
                }
        }
        return
}
