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

/*
type modification struct {
        target Value
        result Value
}

func (m *modification) String() string {
        return m.target.String()
}
*/

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
func (m *modifier) traverse(t *traversal) (brks breakers) {
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("modifier.traverse(%s)", m))) }
        if optionTraceTraversal   { defer un(tt(t_traverse, t, m)) }
        if brks = t.program.modify(t, m); !brks.has() {
                if n := diag.numErrors(); n > 0 {
                        brks.add(m.position, breakFail).message = fmt.Sprintf("%s: %d errors", m.name, n)
                }
        } else if tb := brks.not(breakCase, breakDone); tb.has() {
                t.batch(func() {
                        for _, brk := range tb {
                                switch brk.what {
                                case breakFail: diag.errorAt(brk.pos, "broken traversal for modifier %v failed: %v", m.name, brk.message)
                                case breakErro: diag.errorAt(brk.pos, "broken traversal for modifier %v with error: %v", m.name, brk.error)
                                default: diag.errorAt(brk.pos, "broken traversal for modifier %v (%v)", m.name, brk.what)
                                }
                        }
                        diag.errorAt(m.position, "broken traversal for modifier %v in %v", m.name, t.project).debug(1)
                })
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
func (g *modifiergroup) traverse(t *traversal) (brks breakers) {
        if optionTraceTraversal { defer un(tt(t_traverse, t, g)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("modifiergroup.traverse(%s)", g))) }
        for _, m := range g.modifiers {
                if brks = m.traverse(t); !brks.has() { continue }
                if tb := brks.of(breakNext, breakCase, breakDone); tb.has() {
                        break
                } else if tb = brks.of(breakFail, breakErro); tb.has() {
                        t.batch(func() {
                                for _, brk := range brks {
                                        switch brk.what {
                                        case breakFail: diag.errorAt(brk.pos, "broken traversal for modifier %s with failure: %v", m.name, brk.message).debug(1)
                                        case breakErro: diag.errorAt(brk.pos, "broken traversal for modifier %s with error: %v", m.name, brk.error).debug(1)
                                        }
                                }
                                diag.errorAt(m.position, "broken traversal for modifier %s", m.name).debug(1)
                        })
                        break
                } else {
                        t.batch(func() {
                                for _, brk := range brks {
                                        diag.errorAt(brk.pos, "%s: broken %v", m.name, brk.what)
                                }
                                diag.errorAt(m.position, "%s: broken unexpectedly", m.name).debug(1)
                        })
                        break
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
        ModifierFunc   func(*traversal, ...Value) (Value, breakers)
        PredictionFunc func(*traversal, ...Value) (Value, error)
)

var (
        init_modifiers = map[string]ModifierFunc{
                `print`:        modifierPrint,
                `debug`:        modifierDebug,

                `select`:       modifierSelect,

                `env`:          modifierEnv,  // interpreter environments
                `set`:          modifierSet,

                `closure`:      modifierClosure,

                `cd`:           modifierCD,
                `mkdir`:        modifierMkdir,
                `path`:         modifierPath,

                `sudo`:         modifierSudo,

                `touch`:        modifierTouch,
                `grep`:         modifierGrep,
                `deps`:         modifierDeps,

                `copy-file`:      modifierCopyFile,
                `write-file`:     modifierWriteFile,
                `read-file`:      modifierReadFile,
                `update-file`:    modifierUpdateFile,
                `configure-file`: modifierConfigureFile,
                `configure`:      modifierConfigure,

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
                `dirty`:            predictionDirty,
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
                                                diag.errorOf(elem, "stringify '%v' failed: %v", elem, err)
                                                return
                                        } else if strings.HasSuffix(str, "\n") {
                                                diag.prompt("%s", str)
                                        } else if str != "" {
                                                diag.prompt("%s\n", str)
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
func modifierPrint(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts = modifierPrintOpts{ stderr: true }
                content string
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        } else if value, okay := t.Get("-"); !okay || isNil(value) {
                // ...
        } else if content, err = value.Strval(t) ; err != nil {
                diag.errorAt(pos, "stringify buffer value failed: %v", err).debug(1)
                return
        }
        if opts.stdout { fmt.Fprint(stdout, content) }
        if opts.stderr { fmt.Fprint(stderr, content) }
        if opts.reset  { t.Set("-", MakeNone(pos)) }
        return
}

type modifierDebugOpts struct {
        info []Value `i,info`
        warn []Value `w,warn`
        error []Value `e,err;er,error`
        checkDirty bool `d,dirty;cd,checkdirty;cd,check-dirty`
}
func modifierDebug(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierDebugOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var s string
        for _, info := range opts.info {
                if s, err = info.Strval(t); err != nil {
                        diag.errorOf(info, "strval '%v' failed: %v", err).debug(1)
                        return
                }
                diag.infoOf(info, "%s", s).debug(1)
        }
        for _, warn := range opts.warn {
                if s, err = warn.Strval(t); err != nil {
                        diag.errorOf(warn, "strval '%v' failed: %v", err).debug(1)
                        return
                }
                diag.warnOf(warn, "%s", s).debug(1)
        }
        for _, error := range opts.error {
                if s, err = error.Strval(t); err != nil {
                        diag.errorOf(error, "strval '%v' failed: %v", err).debug(1)
                        return
                }
                diag.errorOf(error, "%s", s).debug(1)
        }
        var (
                target , _ = t.Get("@")
                depends, _ = t.Get("^")
                ordered, _ = t.Get("|")
                grepped, _ = t.Get("~")
        )
        if len(opts.info) == 0 && len(opts.warn) == 0 && len(opts.error) == 0 {
                diag.warnAt(pos, "debug: %v %v", target, depends).debug(1)
        }
        if opts.checkDirty && !isNil(target) {
                var tt = target.stat(t).mod()
                if tt.IsZero() {
                        diag.infoAt(pos, "target not exists: %v", target).debug(1)
                        return
                }
                for _, dep := range merge(depends, ordered, grepped) {
                        var dt = dep.stat(t).mod()
                        if false { if s := dep.String(); strings.HasSuffix(s, ".o") {
                                diag.infoAt(pos, "%v -> %T %v, %v", target, dep, dep, dt.Sub(tt)).debug(false, 1)
                        }}
                        if dt.After(tt) {
                                diag.infoAt(pos, "%v: outdated by %v (%v)", target, dep, dt.Sub(tt)).debug(1)
                        }
                }
        }
        return
}

// select element by index from group result: (select 0)
func modifierSelect(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                value, _ = t.Get("-")
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if g, ok := value.(*Group); ok && len(args) > 0 {
                var num int64
                if num, err = args[0].Integer(t); err != nil {
                        diag.errorAt(pos, "integify '%v' failed: %v", args[0], err).debug(1)
                } else {
                        result = g.Get(int(num))
                }
        }
        return
}

func modifierEnv(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "expand args failed: %v", err).debug(1)
                return
        }

        var envars = new(List)
        for _, a := range args {
                if _, ok := a.(*Pair); ok { envars.Append(a) } else {
                        err = errors.New(fmt.Sprintf("invalid env '%v' (%s)", a, typeof(a)))
                        return
                }
        }
        if _, okay := t.Set(TheShellEnvarsDef, envars); !okay {
                diag.errorAt(pos, "set '%s' failed: %v", TheShellEnvarsDef, envars).debug(1)
        } else {
                result = envars
        }
        return
}

type modifierSetOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
}

// examples:
//     [(set name=value)]    set $(name) to 'value'
//     [(set name)]          clear $(name)
//     [(set -)]             clear $-
func modifierSet(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierSetOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }
        var defs []Value
        var none = MakeNone(pos)
ForArgs:
        for _, arg := range args {
                var ( value Value = none; def *Def; name string )
                switch a := arg.(type) {
                case *Bareword: name = a.string
                case *Pair:
                        if name, err = a.Key.Strval(t); err == nil {
                                // Note that Pair.Value is not expanded yet!
                                // We need to expand the value explicitly.
                                if value, err = a.Value.expand(t, expandPlainValue); err != nil {
                                        diag.errorAt(pos, "expand value '%v' failed: %v", a.Value, err).debug(1)
                                        return
                                } else if isNil(value) { value = a.Value }
                        }
                case *Flag:
                        if name, err = a.name.Strval(t); err != nil {
                                diag.errorAt(pos, "strval '%v' failed: %v", a.name, err).debug(1)
                                return
                        } else if value = none; name == "" { name = "-" }
                default:
                        diag.errorAt(pos, "%T `%s` is unsupported (try: foo=value)", arg, arg).debug(1)
                        return
                }
                if def = t.program.scope.FindDef(name); def == nil {
                        diag.errorAt(pos, "no such def '%s' (%v, %v)", name, arg, args).debug(16)
                        break ForArgs
                } else if err = def.val(t, value); err != nil {
                        diag.errorAt(pos, "set def '%s' failed: %v", name, err).debug(1)
                        return
                } else {
                        defs = append(defs, def)
                }
        }
        if len(defs) > 0 { result = MakeListOrScalar(pos, defs) }
        return
}

type modifierClosureOpts struct {
        dump bool `d,dump`
        verbose bool `v,verbose`
}

// create closure context for the traversal
func modifierClosure(t *traversal, args... Value) (result Value, brks breakers) {
        var pos = t.Position()
        // Set caller context before parsing arguments (pop the top one).
        // The context will be restored when execution is finished.
        if c := t.caller(); c != nil { t.project, t.closure = c.project, c.closure }

        if false {
                if len(cloctx) > 0 { cloctx = cloctx[1:] }
        } else if len(cloctx) > 1 && cloctx[0] == t.program.scope {
                setclosure(append(cloctx[1:], cloctx[0]))
        } else if len(cloctx) == 0 || cloctx[0] != t.closure {
                setclosure(cloctx.unshift(t.closure))
        }

        var (
                opts modifierClosureOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse closure opts failed: %v", err).debug(1)
                return
        }

        if opts.verbose {
                diag.infoAt(pos, "%v", cloctx).debug(1)
        }
        if opts.dump {
                t.traceCallStack(pos, -1, "call trace:")
        }

        var dir string // closure work directory
        if len(cloctx) == 0 {
                diag.errorAt(pos, "empty closure context").debug(1)
        } else if def := cloctx[0].FindDef("/"); def == nil {
                diag.errorAt(cloctx[0].position, "&/ is undefined").debug(1)
        } else if dir, err = def.value.Strval(t); err != nil {
                diag.errorOf(def.value, "%v", err).debug(1)
        } else if dir == "" {
                diag.errorAt(cloctx[0].position, "&/ is empty").debug(1)
        } else if !filepath.IsAbs(dir) {
                diag.errorAt(cloctx[0].position, "&/ is relative").debug(1)
        } else if err = enter(t, dir); err == nil {
                t.program.project.changedWD = dir
                t.program.changedWD = dir
        }
        return
}

type modifierCDOpts struct {
        makePath bool `p,path`
        printEnter bool `e,print-enter`
        printLeave bool `l,print-leave`
}
func modifierCD(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierCDOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        }
        if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse cd opts failed: %v", err).debug(1)
                return
        }

        if opts.printEnter { printEnteringDirectory() }
        if opts.printLeave { printLeavingDirectory() }
        if (opts.printEnter || opts.printLeave) && len(args) == 0 { return }
        if len(args) == 1 {
                var dir string
                if dir, err = args[0].Strval(t); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", args[0], err).debug(1)
                        return
                } else if dir == "" {
                        // TODO: do something special
                        return
                }
                if !filepath.IsAbs(dir) {
                        dir = filepath.Join(t.program.project.absPath, dir)
                }
                if opts.makePath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                                diag.errorAt(pos, "make path '%s' failed: %v", dir, err)
                                return
                        }
                }
                if err = enter(t, dir); err == nil {
                        t.program.project.changedWD = dir
                        t.program.changedWD = dir
                }
        } else {
                diag.errorAt(pos, "wrong number of cd args: %v", args).debug(1)
        }
        return
}

type modifierMkdirOpts struct {
        mode os.FileMode `m,mode`
        verbose bool `v,verbose`
}
func modifierMkdir(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts = modifierMkdirOpts{ mode: os.FileMode(0755) }
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge mkdir args failed: %v", err).debug(1)
                return
        }
        if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse mkdir opts failed: %v", err).debug(1)
                return
        }
        if len(args) == 0 {
                var target, _ = t.Get("@")
                var s string
                if s, err = target.Strval(t); err != nil {
                        diag.errorAt(pos, "stringify target '%v' failed: %v", target, err).debug(1)
                } else if err = os.MkdirAll(filepath.Dir(s), opts.mode); err != nil {
                        diag.errorAt(pos, "make path '%s' failed: %v", s, err).debug(1)
                }
                return
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(t); err != nil {
                        diag.errorAt(pos, "stringify '%v' failed: %v", a, err).debug(1)
                        break
                }
                if err = os.MkdirAll(s, opts.mode); err != nil {
                        diag.errorAt(pos, "make path '%s' failed: %v", s, err).debug(1)
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
func modifierPath(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierPathOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge path args failed: %v", err)
                return
        } else if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse path opts failed: %v", err)
                return
        }

        if len(args) == 0 {
                var target, _ = t.Get("@")
                var s string
                if s, err = target.Strval(t); err != nil {
                        diag.errorAt(pos, "stringify target value '%v' failed: %v", target, err).debug(1)
                } else if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                                diag.errorAt(pos, "make path '%s' failed: %v", err).debug(1)
                        }
                }
                return
        }

        for _, arg := range args {
                var s string
                if s, err = arg.Strval(t); err != nil {
                        diag.errorOf(arg, "stringify arg '%v' failed: %v", arg, err).debug(1)
                        break
                }
                if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                        diag.errorAt(pos, "make path '%s' failed: %v", s, err).debug(1)
                        break
                }
        }
        return
}

func modifierSudo(t *traversal, args... Value) (result Value, brks breakers) {
        diag.errorAt(t.Position(), "TODO: sudo modifier is not implemented yet").debug(1)
        return
}

func parseDependList(t *traversal, dependList *List) (depends *List, brks breakers) {
        var pos = t.Position()
        depends = new(List)
        for _, depend := range dependList.Elems {
                switch d := depend.(type) {
                case *List:
                        if dl, err := parseDependList(t, d); err != nil {
                                diag.errorAt(pos, "%v", err).debug(1)
                                return
                        } else {
                                depends.Elems = append(depends.Elems, dl.Elems...)
                        }
                case *ExecResult:
                        if d.Status != 0 {
                                brks.add(pos, breakFail).message = fmt.Sprintf("bad status %v", d.Status)
                                return // target shall be updated
                        } else {
                                depends.Append(d)
                        }
                case *RuleEntry:
                        switch d.class {
                        case GeneralRuleEntry, PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                                depends.Append(d)
                        default:
                                diag.errorAt(pos, "unsupported entry depend `%v' (%v)", d, d.class).debug(1)
                        }
                case *String:
                        depends.Append(d)
                case *File:
                        depends.Append(d)
                default:
                        diag.errorAt(pos, "unsupported entry depend `%v' (%v)", depend, t.program.depends).debug(1)
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
                diag.errorAt(g.target.Position(), "'%v' not exists", g.target).debug(1)
                return
        }
        var pos = ctx.Position()
        var tt time.Time = g.targetInfo.ModTime()
        for _, val := range g.files {
                var file, ok = val.(*File)
                if !ok { 
                        diag.errorAt(pos, "'%v' is not file (%T)\n", file, file).debug(1)
                        return
                }
                if file.info == nil && !file.isSysFile() {
                        var s string
                        if s, err = file.Strval(ctx); err != nil { diag.errorAt(pos, "%v", err); return }
                        if file.info, _ = os.Stat(s); file.info == nil { continue }
                        if gc.debug { diag.warnAt(pos, "'%v' info is nil (%s)\n", file, file.fullname()) }
                }
                if file.info == nil {/* ... */} else
                if t := file.info.ModTime(); t.After(tt) {
                        if gc.debug { diag.warnAt(pos, "touch %v → %v (%v)\n", g.target, file, t) }
                        if tt != t { tt = t }
                }
        }
        if tt.After(g.targetInfo.ModTime()) {
                if err = os.Chtimes(g.targetFullName, tt, tt); err != nil {
                        diag.errorAt(pos, "%v", err).debug(1)
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
        } else if t, ok := g.target.(*File); ok && t.name == file.name {
                res = true
        }
        return
}

var grepcache = make(map[string][]Value)

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

func (t *traversal) searchGreppedName(ctx Context, gp Position, gc *grepctx, sys bool, name string) (file *File) {
        var pos = ctx.Position()
        var isAbs, isRel bool
        if isAbs = filepath.IsAbs(name); isAbs {
                file = stat(ctx, name, "", "", nil)
        } else if isRel = isRelPath(name); isRel { // relative to targetDir
                file = stat(ctx, name, "", gc.targetDir, nil)
        } else if file = t.project.FindFile(ctx, name); file != nil && file.exists() {
                return // found existed file
        }

        // System files are not treated as missing nor collected
        // for further updating, just discard them immediately.
        if !sys && file != nil && file.filemap != nil && len(file.filemap.Paths) == 1 {
                // system files defined by `files ((foo.xxx) ⇒ -)`
                if f, ok := file.filemap.Paths[0].(*Flag); ok {
                        sys = isNone(f.name) || isNil(f.name)
                }
        }
        if!sys && gc.debug {
                diag.errorAt(pos, "%v: %v → %v (exists=%v, sys=%v, from %v)\n",
                        t.entry.target, gc.target, name, file.exists(), sys, t.project).debug(1)
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
        alt = t.project.FindFile(ctx, filepath.Base(name))
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
                                diag.errorOf(inc, "strval '%v' failed: %v", inc, e).debug(1)
                        } else if file = stat(ctx, name, "", s); file != nil {
                                if false { diag.infoAt(pos, "%v in %v", file, inc).debug(1) }
                                return
                        }
                }
                if file == nil { file = stat(ctx, name, "", "", nil) }
                diag.warnAt(gp, "'%s' not found in %v", name, t.project)
                diag.warnAt(pos, "grepped '%s' has no target dir in %v", name, t.project)
                diag.warnAt(t.project.position, "from project %v (for %v)", t.project, name).debug(8)
        }
        return
}

func (t *traversal) searchGrepped(ctx Context, gp Position, gc *grepctx, sys bool, name string) (file *File, err error) {
        var pos = ctx.Position()
        if file = t.searchGreppedName(ctx, gp, gc, sys, name); file == nil {
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
                                diag.errorAt(pos, "strval '%v' failed: %v", file, err).debug(1)
                                return
                        }
                        if file.info, err = os.Stat(s); err != nil {
                                diag.errorAt(pos, "%v", err).debug(1)
                                return
                        }
                        if false || gc.debug {
                                diag.warnAt(pos, "'%v' info is nil (%s)", file, file.fullname()).debug(1)
                        }
                }
                if file.info == nil {/* ... */} else
                if t := file.info.ModTime(); t.After(tt) {
                        if true || gc.debug {
                                diag.warnAt(pos, "touch %v → %v (%v)", gc.target, file, t).debug(1)
                        }
                        t = launchTime //time.Now() // ...
                        if err, tt = os.Chtimes(gc.targetFullName, t, t), t; err != nil {
                                diag.errorAt(pos, "chtimes failed: %v", err).debug(1)
                                return
                        }
                }
        }

        // Report missing files, but system files are not treated as missing.
        if !gc.report {
                // ...
        } else if file == nil {
                diag.infoAt(gp, "%s: `%s` not found", t.project.name, name)
        } else if !file.exists() {
                diag.infoAt(gp, "%s: `%s` file not existed", t.project.name, name)
        }
        return
}

func (t *traversal) tempFile(ctx Context, prefix, hashee0 string, hasheeN... interface{}) (file *File, err error) {
        var pos = ctx.Position()
        var nameHash = sha256.New() // HashByte -> [sha256.Size]byte
        if _, err = fmt.Fprint(nameHash, prefix, hashee0); err != nil {
                diag.errorAt(pos, "hashing failed: %v", err).debug(1)
        } else if _, err = fmt.Fprint(nameHash, hasheeN...); err != nil {
                diag.errorAt(pos, "hashing failed: %v", err).debug(1)
        } else if nameSum := nameHash.Sum(nil); len(nameSum) != sha256.Size {
                diag.errorAt(pos, "hash sum invalid: %v", len(nameSum)).debug(1)
        } else {
                // Make names like .deps/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
                file = t.project.matchTempFile(ctx, filepath.Join(prefix, // e.g. ".deps", ".grep"
                        fmt.Sprintf("%x", nameSum[ :1]),
                        fmt.Sprintf("%x", nameSum[1:2]),
                        fmt.Sprintf("%x", nameSum[2: ]),
                ))
        }
        return
}

func (t *traversal) savedDepsFileName(ctx Context, targetFullName string) (filename string, err error) {
        var ( pos = ctx.Position(); file *File )
        if file, err = t.tempFile(ctx, ".deps", targetFullName); err != nil {
                diag.errorAt(pos, "get .deps temp file failed: %v", err).debug(1)
        } else if filename, err = fullnameOrStrval(ctx, file); err != nil {
                diag.errorAt(pos, "get .deps temp filename failed: %v", err).debug(1)
        }
        return
}

func (t *traversal) savedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
        var ( pos = ctx.Position(); file *File )
        if file, err = t.tempFile(ctx, ".grep", targetFullName); err != nil {
                diag.errorAt(pos, "get .grep temp file failed: %v", err).debug(1)
        } else if filename, err = fullnameOrStrval(ctx, file); err != nil {
                diag.errorAt(pos, "get .grep temp filename failed: %v", err).debug(1)
        }
        return
}

func (t *traversal) loadSavedGrepFile(ctx Context, gc *grepctx) (okay bool, err error) {
        var pos = ctx.Position()
        if gc.savedGrepFileName, err = t.savedGrepFileName(ctx, gc.targetFullName); err != nil {
                diag.errorAt(pos, "get saved grep filename failed: %v", err).debug(1)
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
                diag.errorAt(pos, "open saved grep filename failed: %v", err).debug(1)
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
                        if file, err = t.searchGrepped(ctx, gp, gc, sys == 1, name); err != nil {
                                diag.errorAt(pos, "search grepped filename failed: %v", err).debug(1)
                                break
                        } else if file != nil {
                                file.position = gp
                                if gc.isTargetFile(ctx, file) { continue }
                        } else if sys != 1 && !gc.discard {
                                diag.warnAt(gp, "%s is nil file", name)
                                diag.warnAt(pos, "grepped %s is nil", name)
                                diag.warnAt(t.project.position, "from project %v", t.project).debug(6)
                        }
                }
        }
        if gc.savedGrepFile.info, err = savedGrepOSFile.Stat(); err != nil {
                diag.errorAt(pos, "stat saved grep filename error: %v", err).debug(1)
        } else { okay = true }
        return
}

func (t *traversal) grepTargetFile(ctx Context, gc *grepctx) (err error) {
        var ( pos = ctx.Position(); file *os.File )
        if file, err = os.Open(gc.targetFullName); err != nil {
                diag.errorAt(pos, "%v", err).debug(1)
                return
        } else { defer func() { err = file.Close() } () }

        for _, x := range gc.rxs {
                if x.Regexp != nil {
                        continue
                } else if x.Regexp, err = regexp.Compile(x.string); err != nil {
                        diag.errorAt(pos, "%v", err).debug(1)
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
                                if file, err = t.searchGrepped(ctx, gp, gc, sys, name); err != nil {
                                        diag.errorAt(pos, "search grepped '%s' failed: %v", name, err).debug(1)
                                        return
                                } else if file != nil {
                                        if file.position = gp; gc.isTargetFile(ctx, file) { continue }
                                } else if !sys && !gc.discard {
                                        diag.warnAt(gp, "%s is nil file", name)
                                        diag.warnAt(pos, "grepped %s is nil", name)
                                        diag.warnAt(t.project.position, "from project %v", t.project).debug(6)
                                }
                                continue ForScan // found one
                        }
                }
        }
        return
}

func (t *traversal) grep(ctx Context, gc *grepctx) (err error) {
        var ( pos = ctx.Position(); targetName string )
        switch v := gc.target.(type) {
        case *File:
                targetName = v.name
                gc.targetInfo = v.info
                gc.targetFullName = v.fullname()
                gc.targetDir = filepath.Dir(gc.targetFullName)
                if v.isSysFile() { return }
        default:
                gc.targetDir = t.project.absPath
                if targetName, err = v.Strval(ctx); err != nil {
                        diag.errorAt(pos, "strval grep target '%v' failed: %s", v, err).debug(1)
                        return
                }
                if filepath.IsAbs(targetName) {
                        gc.targetFullName = targetName
                } else {
                        gc.targetFullName = filepath.Join(gc.targetDir, targetName)
                }
                if file := stat(ctx, gc.targetFullName, "", ""); file == nil {
                        diag.errorAt(pos, "grep: '%s' not found", gc.targetFullName).debug(1)
                        return
                } else {
                        gc.targetInfo = file.info
                }
        }
        if err != nil {
                diag.errorAt(pos, "grep target %s: %v", targetName, err).debug(1)
                return
        }

        if gc.targetInfo == nil { return }
        if gc.done == nil { gc.done = make(map[string]int) }
        if !filepath.IsAbs(gc.targetFullName) {
                diag.errorAt(pos, "grep: '%s' is not abs", gc.targetFullName).debug(1)
                return
        } else {
                gc.done[gc.targetFullName] += 1
        }
        if n, done := gc.done[gc.targetFullName]; done && n > 1 {
                if gc.debug { diag.errorAt(pos, "%v (done %v)", gc.targetFullName, n).debug(1) }
                return
        }

        //var infos = strings.Contains(gc.targetFullName, "...")
        const infos = false

        if false { defer un(tt(t_traverse, t, gc.target)) }

        defer func(restore []Value) {
                var touch = gc.greptouch // copy greptouch value
                if len(touch.files) > 0 {
                        grepcache[gc.targetFullName] = touch.files
                } else if false {
                        var gp Position
                        gp.Filename, gp.Line = gc.targetFullName, 1
                        diag.warnAt(gp, "grebbed zero files")
                        diag.warnAt(pos, "grebbed zero files: %v", gc.targetFullName).debug(6)
                }
                gc.files = restore
                if gc.debug { diag.errorAt(pos, "grepped: %s → %v (grepped=%v) (saved=%s)\n",
                        gc.target, touch.files, len(t.grepped), gc.savedGrepFile).debug(1) }
                for _, gc.target = range touch.files {
                        if t.grepped = append(t.grepped, gc.target); !gc.recursive {
                                continue
                        } else if err = t.grep(ctx, gc); err != nil {
                                diag.errorAt(pos, "grep files (deferred): %v", err).debug(1)
                                break
                        }
                }
                if err == nil && gc.touch {
                        if err = touch.work(ctx, gc); err != nil {
                                diag.errorAt(pos, "grep touch failed: %v", err).debug(1)
                        }
                }
        } (gc.files)

        gc.files = nil

        var (
                cached bool
                savedGrepFile *os.File
                savedGrepFileLoaded bool
        )
        if gc.files, cached = grepcache[gc.targetFullName]; cached && len(gc.files) > 0 {
                if gc.debug { diag.errorAt(pos, "grepcache: %v → %v", gc.targetFullName, gc.files).debug(1) }
                return
        } else if infos {
                diag.infoAt(pos, "grepcache: %s files=%d", gc.targetFullName, len(gc.files)).debug(1)
        }

        if savedGrepFileLoaded, err = t.loadSavedGrepFile(ctx, gc); err != nil {
                diag.errorAt(pos, "load saved grepfile failed: %v", err).debug(1)
                return
        } else if savedGrepFileLoaded && len(gc.files) > 0 {
                if infos { diag.infoAt(pos, "loadSavedGrepFile: %v files=%d grepped=%d",
                        gc.targetFullName, len(gc.files), len(t.grepped)).debug(1) }
                return
        }
        if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                        diag.errorAt(pos, "make grep dir failed: %v", err).debug(1)
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
                        diag.errorAt(pos, "grep write file: %v", err).debug(1)
                        return
                } else if false {
                        diag.infoAt(pos, "saved grep %s", name).debug(1)
                }
        }
        if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil {
                diag.errorAt(pos, "grep create %s: %v", gc.savedGrepFileName, err).debug(1)
                return
        }

        gc.save = bufio.NewWriter(savedGrepFile)
        defer func() {
                gc.save.Flush()
                savedGrepFile.Close()
        } ()

        if err = t.grepTargetFile(ctx, gc); err != nil && !gc.discard {
                diag.errorAt(pos, "grep target file: %v", err).debug(1)
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
func modifierGrep(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                gc grepctx
                err error
        )
        gc.fileinc = true // grep files by default
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge grep args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &gc.modifierGrepOpts, args...); err != nil {
                diag.errorAt(pos, "parse grep args failed: %v", err).debug(1)
                return
        } else if gc.incs, err = mergeresult2(expandall2(t, expandPlainValue, gc.incs...)); err != nil {
                diag.errorAt(pos, "expand grep incs failed: %v", err).debug(1)
                return
        }
        for _, s := range gc.sys { gc.rxs = append(gc.rxs, &greprex{s, true , nil}) }
        for _, s := range gc.reg { gc.rxs = append(gc.rxs, &greprex{s, false, nil}) }
        for _, s := range gc.langs {
                if info, ok := langInfos[s]; ok && info != nil {
                        for _, re := range info.rxs { gc.rxs = append(gc.rxs, &greprex{re, false, nil}) }
                        for _, re := range info.sys { gc.rxs = append(gc.rxs, &greprex{re, true , nil}) }
                } else {
                        diag.errorAt(pos, "lang '%s' is unknown", s).debug(1)
                        return
                }
        }
        if len(gc.rxs) == 0 {
                diag.errorAt(pos, "no grep expressions: %v %v %v %v", gc.sys, gc.reg, gc.langs, args).debug(1)
                return
        }

        var (
                target, _ = t.Get("@")
                targets = args
                grepped = t.grepped
        )
        if len(targets) == 0 { if isNil(target) || isNone(target) {
                diag.errorAt(pos, "no grep target").debug(1)
                return
        } else {
                targets = append(targets, target)
        }}

        if gc.debug {
                diag.warnAt(pos, "grep files: %v %v %v\n", target, gc.rxs, args).debug(1)
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
                        diag.prompt("Grep %v …… (%d files in %v)\n", s, len(grepped), time.Now().Sub(ts)).debug(gc.debug, 6)
                } (time.Now())
        }

        var tar = target
        defer func(v bool) { t.grepping = v } (t.grepping); t.grepping = true
ForTarget:
        for _, target := range targets {
                if isNil(target) {
                        diag.errorAt(pos, "found nil grep target for %v", tar).debug(1)
                        return
                }
                if isNone(target) {
                        diag.errorAt(pos, "grep target '%v' is none for %v", target, tar).debug(32)
                        return
                }

                gc.target, t.grepped = target, nil
                if err = t.grep(t, &gc); err != nil {
                        diag.errorAt(pos, "grep files from %v failed: %v", target, err).debug(1)
                        return
                } else if gc.noTraverse {
                        // does nothing
                } else if len(t.grepped) > 0 {
                        for _, val := range t.grepped {
                                if brks = val.traverse(t); !brks.has() { continue }
                                t.batch(func() {
                                        for _, brk := range brks {
                                                switch brk.what {
                                                case breakFail: diag.errorAt(brk.pos, "broken traversal for grepped %v failed: %v", val, brk.message)
                                                case breakErro: diag.errorAt(brk.pos, "broken traversal for grepped %v with error: %v", val, brk.error)
                                                default: diag.errorAt(brk.pos, "broken traversal for grepped %v: %v (%v)", val, brk.message, brk.what)
                                                }
                                        }
                                        diag.errorAt(pos, "broken traversal for grepped %v from %v", val, target)
                                        diag.errorAt(t.project.position, "from project %v (for %v)", t.project, val).debug(1)
                                })
                                break ForTarget
                        }
                }
                grepped = append(grepped, t.grepped...)
        }
        t.grepped = grepped

        if err != nil {
                diag.errorAt(pos, "grep files failed: %v", err).debug(1)
        } else if !gc.noTraverse {
                t.Set("~", MakeNone(pos))
                t.grepped = nil
        } else {
                result = MakeListOrScalar(pos, t.grepped)
        }
        return
}

func (t *traversal) parseDeps(ctx Context, savedDepsFileName, deps string) (files []Value, brks breakers) {
        var (
                pos = ctx.Position()
                target, _ = t.Get("@")
                targetFullName string
                err error
        )
        if targetFullName, err = fullnameOrStrval(ctx, target); err != nil {
                diag.errorAt(pos, "fullname '%v' failed: %v", target, err).debug(1)
                return
        }
        var ( firstWord string; dp Position ); dp.Filename = savedDepsFileName
        var findDepFile = func(name string) (file *File) {
                if filepath.IsAbs(name) {
                        file = stat(ctx, name, "", "", nil)
                } else if file = t.project.FindFile(ctx, name); file != nil && file.exists() {
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
        const parallel = true
        var wg sync.WaitGroup
        var filesMux sync.Mutex
        var depFile = func(t *traversal, word string) /*(file *File, nxt int)*/ {
                if parallel {
                        defer recoverPanics(ctx)
                        defer wg.Done() // minus 1
                }
                if i := strings.Index(word, " "); i > 0 {
                        diag.warnAt(pos, "ignore dep with spaces: %v", word).debug(1)
                        //nxt = 1 //continue
                } else if file := findDepFile(word); file == nil {
                        t.batch(func() {
                                diag.errorAt(pos, "unknown dep '%v' for '%v'", word, firstWord)
                                diag.errorAt(dp, "from here: %s", word)
                                if filepath.IsAbs(firstWord) {
                                        var wp Position; wp.Filename, wp.Line = firstWord, 1
                                        diag.errorAt(wp, "in here: %v", word)
                                }
                                diag.errorAt(t.project.position, "for project %v", t.project).debug(6)
                        })
                } else if ignored(file.fullname()) {
                        //nxt = 1 //continue // dep is the target itself
                } else if brks = file.traverse(t); brks.has() {
                        t.batch(func() {
                                for _, brk := range brks {
                                        switch brk.what {
                                        case breakFail: diag.errorAt(brk.pos, "broken traversal for dep '%v' failed: %v", file, brk.message)
                                        case breakErro: diag.errorAt(brk.pos, "broken traversal for dep '%v' with error: %v", file, brk.error)
                                        default: diag.errorAt(brk.pos, "broken traversal for dep '%v': %v (%v)", file, brk.message, brk.what)
                                        }
                                }
                                diag.errorAt(dp, "missing dep '%v' for %v", file, target)
                                diag.errorAt(pos, "broken traversal for dep '%v' from %v", file, target)
                                diag.errorAt(t.project.position, "from project %v (for %v)", t.project, file).debug(6)
                        })
                        //nxt = 2 //break ForLines
                } else {
                        filesMux.Lock()
                        files = append(files, file)
                        filesMux.Unlock()
                }
                return
        }
        var wordRecs = make(map[string]int)
        for l, line := range strings.Split(deps, "\n") {
                var words = line
                if i := strings.Index(words, ":"); i > 0 { words = strings.TrimSpace(words[i+1:]) }
                if words = strings.TrimSpace(strings.TrimRight(words, "\\\r\t ")); words == "" {
                        continue // empty line
                }
                for _, word := range strings.Fields(words) {
                        dp.Line, dp.Column = l + 1, strings.Index(line, word) + 1
                        if /*l == 1 && w == 0 &&*/firstWord == "" { firstWord = word }
                        if wordRecs[word] += 1; wordRecs[word] == 1 {
                                if parallel {
                                        wg.Add(1); go depFile(t.spawn(ctx), word)
                                } else {
                                        depFile(t, word)
                                }
                        }
                }
        }
        wg.Wait()
        return
}

func (t *traversal) loadSavedDepsAndCheckOutdated(ctx Context) (savedDepsFileName string, files []Value, brks breakers) {
        var (
                pos = ctx.Position()
                currentTargetValue = t.getCurrentTargetValue(ctx)
                currentTarget string
                savedDeps []byte
                err error
        )
        if isNil(currentTargetValue) {
                diag.errorAt(pos, "target is nil").debug(1)
        } else if currentTarget, err = fullnameOrStrval(ctx, currentTargetValue); err != nil {
                diag.errorOf(currentTargetValue, "strval '%v' failed: %v", currentTargetValue, err).debug(1)
        } else if currentTarget == "" {
                diag.errorAt(pos, "target '%v' is empty", currentTargetValue).debug(1)
        } else if savedDepsFileName, err = t.savedDepsFileName(ctx, currentTarget); err != nil {
                diag.errorAt(pos, "get saved deps filename failed: %v", err).debug(1)
        } else if savedDepsFileName == "" {
                diag.errorAt(pos, "empty saved deps filename", savedDepsFileName).debug(1)
        } else if savedDepsFile := stat(ctx, savedDepsFileName, "", ""); savedDepsFile == nil {
                // no saved deps file
        } else if savedDeps, err = ioutil.ReadFile(savedDepsFileName); err != nil {
                diag.errorAt(pos, "can't open saved deps file: %v", savedDepsFileName, err).debug(1)
        } else if files, brks = t.parseDeps(ctx, savedDepsFileName, string(savedDeps)); len(files) > 0 {
                if false { diag.infoAt(pos, "loaded deps %s (%d files)", savedDepsFileName, len(files)).debug(true, 1) }
                var savedDepsFileModTime = savedDepsFile.info.ModTime()
                for _, val := range files { if file, ok := val.(*File); !ok {
                        // ignore
                } else if file.info.ModTime().After(savedDepsFileModTime) {
                        files = nil // needs reload if outdated
                        return
                }}
        }
        return
}

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
func modifierDeps(t *traversal, args... Value) (result Value, brks breakers) {
        // NOTE: parse opts for (deps) before expanding the args, because we share args
        //       with the compilers!
        var (
                pos = t.Position()
                opts modifierDepsOpts
                err error
        )
        if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse deps args failed: %v", err).debug(1)
                return
        } else if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge deps args failed: %v", err).debug(1)
                return
        }

        var files []Value
        if opts.verbose {
                defer func(ts time.Time) {
                        var s string
                        if target, _ := t.Get("@"); !isNil(target) { s = target.String() }
                        diag.prompt("Deps %v …… (%d files in %v)\n", s, len(files), time.Now().Sub(ts)).debug(opts.debug, 6)
                } (time.Now())
        }

CorrectCC:
        switch opts.cc {
        case "cl"   : opts.cc = "clang"; goto CorrectCC
        case "gc"   : opts.cc = "gcc"  ; goto CorrectCC
        case "clang": opts.useClang = true
        case "gcc"  : opts.useGcc = true
        case "":
                if opts.useGcc   { opts.cc = "gcc" }
                if opts.useClang { opts.cc = "clang" }
        default:
                if base := filepath.Base(opts.cc); base == "" {
                        diag.errorAt(pos, "unsupported cc: %v", opts.cc).debug(1)
                        return
                } else if strings.HasPrefix(base, "clang") { opts.useClang = true
                } else if strings.HasPrefix(base, "gcc")   { opts.useGcc = true }
        }

        var flags []Value
        if flags, err = mergeresult2(expandall2(t, expandPlainValue, opts.flags...)); err != nil {
                diag.errorAt(pos, "merge flags failed: %v", err).debug(1)
                return
        }

        var (
                _MM, _MG bool
                ca []string
        )
        for _, f := range flags {
                var s string
                if s, err = f.Strval(t); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", err).debug(1)
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
                if s, err = fullnameOrStrval(t, a); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", err).debug(1)
                        return
                } else { s = strings.TrimSpace(s) }
                switch s {
                case "", "-M", "-MM", "-MG", "-MD", "-MV", "-MP", "-Os", "-O1", "-O2", "-O3",
                     "-c", "-shared", "-static", "-fPIC", "-fcxx-modules", "-fvisibility-inlines-hidden":
                        break // discard unused args
                default: ca = append(ca, s)
                }
        }

        var savedDepsFileName string
        if savedDepsFileName, files, brks = t.loadSavedDepsAndCheckOutdated(t); brks.has() {
                t.batch(func() {
                        for _, brk := range brks {
                                diag.errorAt(brk.pos, "borken loading saved deps: %v", brk.what).debug(1)
                        }
                        diag.errorAt(pos, "broken loading saved deps").debug(1)
                })
                return
        } else if len(files) == 0 {
                var (
                        cc = exec.Command(opts.cc, ca...)
                        stdout bytes.Buffer
                        stderr bytes.Buffer
                )
                cc.Stdout, cc.Stderr = &stdout, &stderr
                if err = cc.Run(); err != nil {
                        t.batch(func() {
                                if true { diag.prompt("%s \\\n  %s\n%s\n----------\n%s.\n",
                                        cc.Path, strings.Join(ca, " \\\n  "), &stdout, &stderr) }
                                diag.errorAt(pos, "deps with %s: %v", filepath.Base(opts.cc), err)
                                diag.errorAt(t.project.position, "for project %v", t.project).debug(6)
                                t.traceCallStack(pos, -1, "deps with %s: %v", filepath.Base(opts.cc), err)
                        })
                        return
                }
                stderr.Reset() // release buffers (optional)

                if savedDepsFileName == "" {
                        diag.errorAt(pos, "empty saved deps file name: %v", savedDepsFileName).debug(1)
                } else if err = os.MkdirAll(filepath.Dir(savedDepsFileName), os.FileMode(0755)); err != nil {
                        diag.errorAt(pos, "make path '%s' failed: %v", filepath.Dir(savedDepsFileName), err).debug(1)
                } else if err = ioutil.WriteFile(savedDepsFileName, stdout.Bytes(), os.FileMode(0666)); err != nil {
                        diag.errorAt(pos, "save deps file failed: %v", err).debug(1)
                } else if false {
                        diag.infoAt(pos, "saved deps %s", savedDepsFileName).debug(true, 1)
                }

                files, brks = t.parseDeps(t, savedDepsFileName, stdout.String())
                stdout.Reset() // release buffers (optional)
        }
        if len(files) > 0 { t.grepped = append(t.grepped, files...) }
        return
}

type modifierTouchOpts struct {
        verbose bool `v,verbose`
        debug bool `d,debug`
        path bool `p,path`
        mode os.FileMode `m,mode`
}
func modifierTouch(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierTouchOpts // = modifierTouchOpts{ mode: os.FileMode(0755) }
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge touch args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse touch opts failed: %v", err).debug(1)
                return
        } else if len(args) == 0 {
                if target, _ := t.Get("@"); !isNil(target) {
                        args = append(args, target)
                }
        }

        var files []*File
        for _, arg := range args {
                var vf []*File
                if err = touch(t, arg, uint32(opts.mode), opts.path); err != nil {
                        diag.errorAt(pos, "touch '%v' failed: %v", arg, err).debug(1)
                        break
                } else if vf, err = arg.stamp(t); err != nil {
                        diag.errorAt(pos, "touch '%v' failed: %v", arg, err).debug(1)
                        break
                } else { files = append(files, vf...) }
        }
        if opts.verbose { reportFileUpdates(t, t.start, files) }
        if len(t.program.getModifiers(t, "stamp")) > 0 {
                diag.warnAt(pos, "no need to use a (stamp) after (touch)").debug(1)
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
func modifierCheck(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierCheckOpts
                optBreak breakind // breaking with good results
                makeResult func(Position,bool) Value // returns results only if non-nil
                value, _ = t.Get("-")
                values []Value
                pairs []*Pair
                err error
                res bool
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge check args failed: %v", err).debug(1)
                return
        }
        if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse check args failed: %v", err).debug(1)
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
                        if res, err = arg.True(t); err != nil {
                                diag.errorAt(pos, "unknown check '%v' (%T)", arg, arg).debug(1)
                        } else if makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else {
                                brks.addf(pos, optBreak, "value '%v' is false", arg)
                                if opts.verbose {
                                        diag.warnAt(pos, "value '%v' is false", arg).debug(1)
                                }
                        }
                }
        }
        if !(isNil(opts.file) || isNone(opts.file)) {
                var ( s string; f *File )
                if f, res = opts.file.(*File); res {
                        if res = f.exists(); !res && opts.verbose {
                                diag.warnOf(opts.file, "file '%v' does not exists", opts.file).debug(1)
                        }
                } else if s, err = opts.file.Strval(t); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", opts.file, err).debug(1)
                        return
                } else if filepath.IsAbs(s) {
                        if f = stat(contextAt(opts.file.Position(), t), s, "", ""); f != nil {
                                res = f.exists()
                        }
                } else if f = t.project.FindFile(t, s); f != nil {
                        res = f.exists()
                }
                if res { res = !f.info.Mode().IsDir() } // .IsRegular()
                if opts.verbose {
                        diag.warnOf(opts.file, "'%v' is file: %v", opts.file, res).debug(1)
                }
                if makeResult != nil {
                        values = append(values, makeResult(pos, res))
                } else if !res {
                        brks.addf(pos, optBreak, "value '%v' is not file", opts.file)
                        return
                }
        }
        if !(isNil(opts.dir) || isNone(opts.dir)) {
                var ( s string; f *File )
                if f, res = opts.dir.(*File); res {
                        if res = f.exists(); !res && opts.verbose {
                                diag.warnOf(opts.dir, "file '%v' does not exists", opts.dir).debug(1)
                        }
                } else if s, err = opts.dir.Strval(t); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", opts.dir, err).debug(1)
                        return
                } else if filepath.IsAbs(s) {
                        if f = stat(contextAt(opts.dir.Position(), t), s, "", ""); f != nil {
                                res = f.exists()
                        }
                } else if f = t.project.FindFile(t, s); f != nil {
                        res = f.exists()
                }
                if res { res = f.info.Mode().IsDir() }
                if opts.verbose {
                        diag.warnOf(opts.dir, "'%v' is file: %v", opts.dir, res).debug(1)
                }
                if makeResult != nil {
                        values = append(values, makeResult(pos, res))
                } else if !res {
                        brks.addf(pos, optBreak, "value '%v' is not dir", opts.dir)
                        return
                }
        }

ForPairs:
        for _, p := range pairs {
                var key, str string
                if key, err = p.Key.Strval(t); err != nil {
                        diag.errorOf(p.Key, "strval '%v' failed: %v", p.Key, err).debug(1)
                        return
                }
                switch key {
                case "status":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                brks.addf(pos, optBreak, "value '%v' is not exec result", value)
                                diag.errorOf(value, "value '%v' (%T) is not exec result", value, value).debug(6)
                                return
                        } else { exeres.wg.Wait() }

                        var num int64
                        if num, err = p.Value.Integer(t); err != nil {
                                diag.errorAt(p.Value.Position(), "%v", err).debug(1)
                                return
                        }
                        if opts.verbose {
                                diag.prompt("Checking status ")
                                if num != 0 { diag.prompt("== %d ", num) }
                                diag.prompt("…")
                        }

                        var good = exeres.Status == int(num)
                        if opts.verbose {
                                var s string 
                                if good { s = "Yes" } else { s = "No" }
                                diag.prompt("… %s (%d)\n", s, exeres.Status)
                        }

                        if false { diag.infoAt(pos, "status=%v, num=%v", exeres.Status, num).debug(true,1) }

                        if makeResult != nil {
                                values = append(values, makeResult(pos, good))
                        } else if !good {
                                brks.addf(pos, optBreak, "bad status (%v) (expects %v)", exeres.Status, p.Value)
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                brks.addf(pos, optBreak, "not an exec result (%T)", value)
                                diag.errorOf(value, "value '%v' (%T) is not exec result", value, value).debug(6)
                                return
                        } else { exeres.wg.Wait() }

                        if opts.verbose {
                                diag.prompt("(status=%d)", exeres.Status)
                        }

                        var v *bytes.Buffer
                        switch key {
                        case "stdout": v = exeres.Stdout.Buf
                        case "stderr": v = exeres.Stderr.Buf
                        default: unreachable()
                        }

                        if v == nil {
                                brks.addf(pos, optBreak, "bad %s (expects %v)", key, p.Value)
                                break ForPairs
                        } else if str, err = p.Value.Strval(t); err != nil {
                                diag.errorOf(p.Value, "strval '%v' failed: %v", p.Value, err).debug(1)
                                return
                        } else if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                brks.addf(pos, optBreak, "bad %s (%v) (expects %v)", key, v, p.Value)
                                break ForPairs
                        }
                case "file", "dir": // file=xxx and dir=xxx, same as -file=xxx and -dir=xxx
                        var ( file *File; res bool )
                        if file, res = p.Value.(*File); res {
                                // ok
                        } else if str, err = p.Value.Strval(t); err != nil {
                                diag.errorAt(pos, "strval '%v' failed: %v", p.Value, err).debug(1)
                                return
                        } else if filepath.IsAbs(str) {
                                if file = stat(contextAt(p.Value.Position(), t), str, "", ""); file != nil {
                                        // ok
                                }
                        } else if file = t.project.FindFile(t, str); file != nil {
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
                                brks.addf(pos, optBreak, "`%v` is not %s", p.Value, key)
                                break ForPairs
                        }
                case "var":
                        var g, ok = p.Value.(*Group)
                        if !ok {
                                brks.addf(pos, optBreak, "`%v` is not a group value", p.Value)
                                break ForPairs
                        }
                        for _, elem := range g.Elems {
                                switch p := elem.(type) {
                                case *Pair:
                                        var k, a, b string
                                        if k, err = p.Key.Strval(t); err != nil { break ForPairs }
                                        var def = t.program.project.scope.FindDef(k)
                                        if def != nil {
                                                if a, err = p.Value.Strval(t); err != nil { break ForPairs }
                                                if b, err = def.value.Strval(t); err != nil { break ForPairs }
                                                if res := a != b; makeResult != nil {
                                                        values = append(values, makeResult(pos, res))
                                                } else if !res {
                                                        brks.addf(pos, optBreak, "`%v` != `%v`", p.Key, p.Value)
                                                        break ForPairs
                                                }
                                        } else if makeResult != nil {
                                                values = append(values, makeResult(pos, false))
                                        } else {
                                                brks.addf(pos, optBreak, "`%v` is not defined", k)
                                                break ForPairs
                                        }
                                default:
                                        brks.addf(pos, optBreak, "`%v` unsupported checks", elem)
                                        break ForPairs
                                }
                        }
                default:
                        diag.errorAt(pos, "unknown check for %v -> %v", p.Key, p.Value).debug(1)
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
        var pos = ctx.Position()
        var def1, def2 *Def
        if true {
                def1 = opts.program.scope.Lookup("1").(*Def)
                def2 = opts.program.scope.Lookup("2").(*Def)
        } else if gs := ctx.GlobeScope(); gs != nil {
                def1 = gs.Lookup("1").(*Def)
                def2 = gs.Lookup("2").(*Def)
        }
        defer func(v1, v2 Value) { def1.value, def2.value = v1, v2
                if err == nil {
                        var file = stat(ctx, dst, "", "")
                        ctx.Stamp(dst, file.info.ModTime())
                }
        } (def1.value, def2.value)
        def1.value = MakeString(pos, dst)
        def2.value = MakeString(pos, src)

        var head, foot string
        if opts.head != nil {
                if head, err = opts.head.Strval(ctx); err != nil { diag.errorAt(pos, "%v", err); return }
                if false { fmt.Fprintf(stderr, "%s: %v => %s\n", opts.head.Position(), opts.head, head) }
        }
        if opts.foot != nil {
                if foot, err = opts.foot.Strval(ctx); err != nil { diag.errorAt(pos, "%v", err); return }
                if false { fmt.Fprintf(stderr, "%s: %v => %s\n", opts.foot.Position(), opts.foot, foot) }
        }

        // Compare mod time for update mode
        if opts.files += 1; opts.update {
                if st2, e := os.Stat(dst); e == nil && st2 != nil {
                        var st1 os.FileInfo
                        if st1, err = os.Stat(src); err != nil { diag.errorAt(pos, "%v", err); return }
                        if st1 != nil && (st1.Size()+int64(len(head))+int64(len(foot))) == st2.Size() {
                                if st2.ModTime().After(st1.ModTime()) { return }
                        }
                        if false { fmt.Fprintf(stderr, "%s: %s (%v,%v)\n", pos, dst, st1.Size(), st2.Size()) }
                }
        }

        var srcFile, dstFile *os.File
        if srcFile, err = os.Open(src); err != nil { diag.errorAt(pos, "%v", err); return } else {
                defer srcFile.Close()
        }

        // sys default file mode is 0666
        if opts.path { // Make path (mkdir -p)
                if p := filepath.Dir(dst); p != "." && p != "/" {
                        err = os.MkdirAll(p, os.FileMode(0755))
                        if err != nil { diag.errorAt(pos, "%v", err); return }
                }
        }

        if opts.mode == 0 { opts.mode = os.FileMode(0640) }

        dstFile, err = os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, opts.mode)
        if err != nil { diag.errorAt(pos, "%v", err); return } else { defer dstFile.Close() }

        srcBuf := bufio.NewReader(srcFile)
        dstBuf := bufio.NewWriter(dstFile)
        if head != "" {
                var n int
                if n, err = dstBuf.WriteString(head); err != nil { diag.errorAt(pos, "%v", err); return }
                opts.bytes += int64(n)
        }

        var n int64
        if n, err = io.Copy(dstBuf, srcBuf); err != nil { diag.errorAt(pos, "%v", err); } else {
                if opts.bytes += n; foot != "" {
                        var n int
                        if n, err = dstBuf.WriteString(foot); err != nil { diag.errorAt(pos, "%v", err); return }
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
func modifierCopyFile(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierCopyFileOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err)
                return
        }

        var target Value
        var source Value
        if len(args) > 0 {
                target = args[0]
        } else {
                target, _ = t.Get("@")
        }
        if len(args) > 1 {
                source = args[1]
        } else {
                source, _ = t.Get("<")
        }

        // Get target filename
        var (
                project = t.project
                filename, srcname string
                filetime, srctime time.Time
        )
        switch tv := target.(type) {
        case *File:
                if filename = tv.fullname(); tv.info != nil {
                        filetime = tv.info.ModTime()
                }
        default:
                if filename, err = target.Strval(t); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", target, err)
                        return
                } else if file := project.FindFile(t, filename); file != nil {
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
                if srcname, err = source.Strval(t); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", source, err)
                        return
                } else if file := project.FindFile(t, srcname); file != nil {
                        source, srcname = file, file.fullname()
                        if file.info != nil { srctime = file.info.ModTime() }
                }
        }

        if filepath.Base(srcname) != filepath.Base(filename) {
                fmt.Fprintf(stderr, "%s:warning: %v, %v, %v\n", pos, target, filename, srcname)

                a, _ := t.Get("@")
                b, _ := t.Get("<")
                c, _ := t.Get("^")
                fmt.Fprintf(stderr, "%s:warning: %v\n", a.Position(), a)
                fmt.Fprintf(stderr, "%s:warning: %v\n", b.Position(), b)
                fmt.Fprintf(stderr, "%s:warning: %v\n", c.Position(), c)
        }

        if !filetime.IsZero() && filetime.After(srctime) {
          if opts.update {
            if opts.verbose { diag.prompt("update %v …", target) }
          } else if opts.override {
            if opts.verbose { diag.prompt("override %v …", target) }
          } else {
            if opts.verbose { diag.prompt("copy %v …… already existed!\n", target) }
            if !opts.silent { diag.errorAt(pos, "file already existed (%s)", target).debug(1) }
            return
          }
        } else if opts.verbose {
                if opts.update {
                        diag.prompt("Checking %v …", target)
                } else {
                        diag.prompt("Copy %v …", target)
                }
        }

        if opts.quick {
                var file = stat(t,filename,"","",nil)
                if file == nil || file.info != nil {
                        if opts.verbose { diag.prompt("… Good\n") }
                        return
                }
        }

        var copts = &copyopts{
                t.program, opts.path||opts.recursive,
                opts.update, opts.mode, opts.head, opts.foot,
                0, 0, 0,
        }
        var file *File
        if file = stat(t,srcname,"","",nil); file == nil || file.info == nil {
                diag.errorAt(pos, "'%s' source file not found", srcname).debug(1)
        } else if !file.info.IsDir() {
                if opts.mode == 0 { opts.mode = file.info.Mode() }
                if err = copyFile(t, file.info, srcname, filename, copts); err != nil {
                        diag.errorAt(pos, "%v", err).debug(1)
                }
        } else if opts.recursive {
                if err = copyDir(t, srcname, filename, copts); err != nil {
                        diag.errorAt(pos, "%v", err).debug(1)
                }
        } else {
                diag.errorAt(pos, "`%v` is a directory (use -r to solve it)", source).debug(1)
        }

        if opts.verbose {
                if err != nil {
                        diag.prompt("… error\n")
                } else if copts.copied == 0 {
                        diag.prompt("… Good (%d files)\n", copts.files)
                } else if copts.copied == 1 {
                        diag.prompt("… Copied %d bytes\n", copts.bytes)
                } else {
                        diag.prompt("… Copied %d bytes (%d/%d)\n", copts.bytes, copts.copied, copts.files)
                }
        }
        return
}

func modifierWriteFile(t *traversal, args... Value) (result Value, brks breakers) {
        var ( pos = t.Position(); err error )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        }

        var (
                target, _ = t.Get("@")
                filename, str string
                f *os.File
        )
        defer func() {
                if err != nil && filename != "" { os.Remove(filename); f = nil }
                if f == nil { brks.add(pos, breakFail).message = fmt.Sprintf("file %s not generated", target) }
        } ()
        if isNil(target) {
                diag.errorAt(pos, "target is undefined").debug(1)
                return
        } else if filename, err = fullnameOrStrval(t, target); err != nil {
                diag.errorAt(pos, "fullname failed: %v", err).debug(1)
                return
        } else if buffer, _ := t.Get("-"); isNil(buffer) {
                diag.errorAt(pos, "buffer value is nil").debug(1)
                return
        } else if str, err = buffer.Strval(t); err != nil {
                diag.errorAt(pos, "strval buffer failed: %v", err).debug(1)
                return
        } else if f, err = os.Create(filename); err != nil {
                diag.errorAt(pos, "%v", err).debug(1)
                return
        } else if _, err = f.WriteString(str); err != nil {
                f.Close()
                diag.errorAt(pos, "%v", err).debug(1)
                return
        } else {
                result = stat(t, filename, "", "")
                f.Close()
        }
        return
}

type modifierReadFileOpts struct {
        debug bool "d,debug"
        verbose bool "v,verbose"
        head Value "h,head"
        foot Value "f,foot"
}
func modifierReadFile(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierReadFileOpts
                filename string
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var target Value
        if n := len(args); n > 1 {
                diag.errorAt(pos, "too many files: %v", args).debug(1)
                return
        } else if n == 1 {
                target = args[0]
        } else {
                target, _ = t.Get("@")
        }

        if isNil(target) {
                diag.errorAt(pos, "target is <nil>").debug(8)
                return
        } else if isNone(target) {
                diag.errorAt(pos, "target is <none>").debug(8)
                return
        } else if filename, err = fullnameOrStrval(t, target); err != nil {
                diag.errorOf(target, "strval '%v' error: %v", target, err).debug(1)
                return
        } else if filename == "" {
                diag.errorOf(target, "target filename is empty").debug(1)
                return
        }

        if opts.debug {
                diag.infoAt(pos, "read-file: %v", filename)
        }

        var bytes []byte
        if bytes, err = ioutil.ReadFile(filename); err == nil {
                var s, v string
                if opts.head != nil {
                        if v, err = opts.head.Strval(t); err == nil { s = v } else {
                                diag.errorAt(pos, "%v", err).debug(1)
                                return
                        }
                }
                s += string(bytes)
                if opts.foot != nil {
                        if v, err = opts.foot.Strval(t); err == nil { s += v } else {
                                diag.errorAt(pos, "%v", err).debug(1)
                                return
                        }
                }
                t.Set("-", MakeString(pos, s))
        } else {
                brks.add(pos, breakErro).error = err
        }
        return
}

func crc64CheckFileModeContent(filename string, content []byte, perm os.FileMode) (same bool, err error) {
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
            diag.prompt("crc64CheckFileModeContent: %v %v\n%s\n%s\n", a, b, s, content)
          }
        }
        return
}

func crc64CompareFileChecksum(filename1, filename2 string) (same bool, err error) {
        var s []byte
        if s, err = ioutil.ReadFile(filename1); err != nil { return }
        return crc64CheckFileModeContent(filename2, s, 0)
}

type modifierUpdateFileOpts struct {
        debug bool "d,debug"
        verbose bool "v,verbose"
        path bool "p,path"
        zero bool `z,zero;e,empty;az,allow-zero;ae,allow-empty`
        mode os.FileMode "m,mode"
}
func modifierUpdateFile(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts = modifierUpdateFileOpts{ mode: os.FileMode(0640) }
                filename string
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var target Value
        if len(args) > 0 { target = args[0] } else { target, _ = t.Get("@") }
        if len(args) > 1 { if opts.mode, err = permVal(t, args[1], 0600); err != nil {
                diag.errorOf(args[1], "perm value '%v' failed: %v", args[1], err).debug(1)
                return
        }}

        // Get target filename
        switch p := target.(type) {
        case *File: filename = p.fullname()
        case *Path:
                if filename, err = p.Strval(t); err != nil {
                        diag.errorAt(pos, "strval path '%v' failed: %v", p, err).debug(1)
                        return
                }
        default:
                if filename, err = target.Strval(t); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", p, err).debug(1)
                        return
                } else if file := t.project.FindFile(t, filename); file != nil {
                        target, filename = file, file.fullname()
                }
        }

        if opts.debug {
                diag.infoAt(pos, "update-file: %v (%v) (%v, %v)", target, filename, t.project, cloctx).debug(1)
        }

        if opts.path { // Make path (mkdir -p)
                if p := filepath.Dir(filename); p != "." && p != "/" {
                        if err = os.MkdirAll(p, os.FileMode(0755)); err != nil {
                                diag.errorAt(pos, "%v", err).debug(1)
                                return
                        }
                }
        }

        // Check existed file content checksum
        var content string
        if value, _ := t.Get("-"); isNil(value) {
                // no buffer value
        } else if content, err = value.Strval(t); err != nil {
                diag.errorAt(pos, "%v", err).debug(1)
                return
        }

        if content == "" {
                if !opts.zero {
                        if file := stat(contextAt(target.Position(), t), filename, "", ""); file != nil && file.info != nil && file.info.Size() == 0 {
                                file.info = nil
                                if err = os.Remove(filename); err != nil {
                                        diag.errorAt(pos, "remove file failed: %v", err).debug(1)
                                }
                        }
                        if s := target.String(); filepath.IsAbs(s) {
                                diag.errorAt(pos, "empty content for '%s'", s).debug(1)
                        } else {
                                diag.errorAt(pos, "empty content for '%s' (at %s)", s, filename).debug(1)
                        }
                        return
                } else if opts.verbose || opts.debug {
                        diag.warnAt(pos, "empty content for '%v'", target).debug(1)
                }
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
                        printEnteringDirectory()
                        diag.prompt("update %v …… %s (in %v)\n", trimPromptString(target.String()), s, time.Now().Sub(st)).debug(opts.debug, 6)
                } (time.Now())
        }
        if same, err = crc64CheckFileModeContent(filename, []byte(content), opts.mode); err != nil {
                if _, ok := err.(*os.PathError); ok {
                        err = nil // discard path error (e.g. no such file or directory)
                } else {
                        diag.errorAt(pos, "crc64 checksum failed: %v", err).debug(1)
                        return
                }
        } else if same {
                t.removeCallerUpdated(t, target) // remove timestamp updated
                result = stat(t, filename, "", "")
                return
        }

        // Create or update the file with new content

        var f *os.File
        if f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts.mode); err != nil {
                brks.add(pos, breakFail).message = fmt.Sprintf("update %v failed", target)
                diag.errorAt(pos, "open file failed: %v", err).debug(1)
        } else if f != nil {
                defer func() {
                        if err = f.Close(); err != nil {
                                os.Remove(filename)
                                diag.errorAt(pos, "close file '%s' failed: %v", filename, err).debug(1)
                                return
                        }
                        var file = stat(t, filename, "", "")
                        if  file == nil {
                                diag.errorAt(pos, "invalid file '%s'", filename).debug(1)
                        } else {
                                var files []*File
                                if files, err = file.stamp(t); err != nil {
                                        diag.errorAt(pos, "%v", err).debug(1)
                                        return
                                } else if false && opts.verbose {
                                        reportFileUpdates(t, t.start, files)
                                }
                                result = file // resulting the updated file
                        }
                } ()
                if wrote, err = f.WriteString(content); err != nil {
                        diag.errorAt(pos, "write content failed: %v", err).debug(1)
                }
        } else {
                brks.add(pos, breakFail).message = fmt.Sprintf("%v not updated", target)
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
func modifierWait(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierWaitOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var (
                execRes *ExecResult
                waitForExecResult = opts.stdout || opts.stderr || opts.status || opts.execRes
                stampCurrentTarget = !opts.noTarget
                target, _ = t.Get("@")
        )
        if opts.verbose {
                defer func (st time.Time) {
                        var s string; if err != nil { s = "fail" } else { s = "done" }
                        diag.prompt("Wait %v …… %s, result=%v, updated=%v\n",
                                target, s, execRes, t.updated).debug(opts.debug, 1)
                        if opts.debug { diag.infoAt(pos, "%v", execRes).debug(6) }
                } (time.Now())
        }

        // Wait for prerequisites and/or execution
        if _, _, execRes, err = t.wait(t, opts.verbose, waitForExecResult, stampCurrentTarget); execRes != nil {
                var (
                        a []Value
                        s string
                        v Value
                )
                if opts.stdout {
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
                        a = append(a, MakeInt(pos,int64(execRes.Status)))
                }
                if len(a) > 0 { result = MakeListOrScalar(pos, a) }
        }
        return
}

func reportFileUpdates(ctx Context, start time.Time, files []*File) {
        var pos = ctx.Position()
        for _, file := range files {
                var (
                        mod = file.info.ModTime()
                        d = time.Now().Sub(start)
                )
                if mod.After(start) {
                        if false {
                                diag.prompt("Updated %v (%v, ModTime=%v)\n", file, d, mod)
                        } else {
                                diag.prompt("Updated %v (%v)\n", file, d)
                        }
                } else {
                        diag.prompt("File %v not changed (%v, ModTime=%v)\n", file, d, mod)
                        diag.warnAt(pos, "incorrect timestamp: %v (JobTime=%v, ModTime=%v)", file, start, mod)
                        diag.warnAt(pos, "the target path name is: %v", file.fullname())
                        diag.warnAt(pos, "try 'touch' the target %v if the path name and command are correct", file)
                        diag.infoAt(pos, "you may ignore the warnings if all correct")
                }
        }
}

type modifierStampOpts struct {
        next bool "n,next" // breakNext if failed to stamp
        error bool "e,err;e,error" // breakErro if failed to stamp
        prompt bool "m,prompt"
        verbose bool "v,verbose"
        debug int "d,debug"
}
func modifierStamp(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierStampOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err)
                return
        }

        // Wait for ExecResult (see also modifier (wait -exec))
        const waitForExecResult = true
        const stampCurrentTarget = true

        // Wait for prerequisites
        var target Value
        if target, _, _, err = t.wait(t, opts.prompt, waitForExecResult, stampCurrentTarget); err == nil {
                return
        } else if opts.next {
                if opts.verbose { diag.warnAt(pos, "%v", err).debug(1) }
                brks.add(pos, breakNext).scope = breakTrave
                err = nil // discard the error
        } else if opts.error {
                if opts.debug > 0 {
                        t.traceCallStack(pos, -1, "%v", err).debug(opts.debug)
                } else {
                        diag.errorAt(pos, "%v", err).debug(1)
                }
                brks.add(pos, breakErro).error = err
        } else if t.stems != nil {
                if opts.debug > 0 {
                        diag.warnAt(pos, "%v", err).debug(opts.debug)
                        t.traceCallStack(pos, -1, "%v", err).debug(opts.debug)
                } else {
                        diag.warnAt(pos, "%v", err).debug(1)
                }
                brks.add(pos, breakNext).scope = breakTrave
                err = nil // discard the error
        } else if pos.IsValid() {
                t.traceCallStack(pos, -1, "failed: %v", err).debug(1)
        } else if targetPos := target.Position(); targetPos.IsValid() {
                t.traceCallStack(targetPos, -1, "failed: %v", err).debug(1)
        } else {
                // TODO: dump more diagnostics information here
        }

        if err != nil {
                if pe, ok := err.(*fs.PathError); ok {
                        diag.errorAt(pos, "stamp %s: %v", trimPromptString(pe.Path), pe.Err)
                        err = pe.Err
                }
        }
        return
}

type predictOpts struct {
        and bool "a,and"
        group bool "g,group"
        traverse bool "t,traverse;t,trave;t,target"
        message string "m,message;m,msg"
        verbose bool "v,verbose"
        verbose0 bool
}
func predict(t *traversal, args... Value) (result bool, breakScope breaksco, message string, err error) {
        var (
                pos = t.Position()
                targetVal, _ = t.Get("@")
                target string
                num int64
        )
        if isNil(targetVal) {
                diag.errorAt(pos, "target is <nil>").debug(1)
                return
        } else if target, err = fullnameOrStrval(t, targetVal); err != nil {
                diag.errorAt(pos, "stringify predict target failed: %v", err).debug(1)
                return
        }
        for caller := t.caller(); caller != nil; caller= caller.caller() {
                if tarVal, _ := caller.Get("@"); isNil(tarVal) {
                        // top level execution, aka via RuleEntry.Execute(...)
                } else if true {
                        var same = targetVal == tarVal
                        if !same && false {
                                same = (targetVal.cmp(t, tarVal) == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n := caller.execRec[targetVal]; n > 0 {
                        num += int64(n)
                }
        }

        target = filepath.Base(target)

        var reasons []string
        var opts predictOpts
        defer func() {
                if opts.verbose {
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
                        diag.prompt("… %s\n", status)
                }
        } ()

ForArgs:
        for _, arg := range args {
                var va []Value // va = merge(arg)
                if va, err = mergeresult2(expandall2(t, expandPlainValue, arg)); err != nil {
                        diag.errorAt(pos, "merge arg '%v' failed: %v", arg, err).debug(1)
                        return
                } else if len(va) == 1 {
                        switch tv := va[0].(type) {
                        case *String: message = tv.string; continue ForArgs
                        case *Compound: if message, err = tv.Strval(t); err != nil {
                                diag.errorOf(tv, "strval '%v' failed: %v", tv, err).debug(1)
                                return
                        } else { continue ForArgs }}
                }
                if va, err = parseOpts(t, &opts, va...) ; err != nil {
                        diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                        return
                }
                if opts.group    { breakScope = breakGroup }
                if opts.traverse { breakScope = breakTrave }
                if opts.verbose && !opts.verbose0 {
                        diag.prompt("checking %v …", target)
                        opts.verbose0 = true
                }
                //if opts.and && !result { continue }
                if !opts.and || (opts.and && result) { for i, a := range va {
                        var ( name string; res Value; tr bool )
                        if g, ok := a.(*Group); !ok || len(g.Elems) == 0 {
                                a = nil // not prediction group
                        } else if name, err = g.Elems[0].Strval(t); err != nil {
                                diag.errorOf(g.Elems[0], "predict: %v", err).debug(1)
                                return
                        } else if predict, ok := predictors[name]; !ok {
                                a = nil // no such named predictor
                        } else if res, err = predict(t, g.Elems[1:]...); err != nil {
                                diag.errorOf(a, "predict: %v", err).debug(1)
                                return
                        } else { a = res } // replace `a`

                        if a == nil {
                                continue // skip
                        } else if p, ok := a.(*prediction); ok {
                                if p.reason != "" { reasons = append(reasons, p.reason) }
                                tr = p.bool
                        } else if tr, err = a.True(t); err != nil {
                                diag.errorOf(a, "truthify '%v' failed: %v", a, err).debug(1)
                                return
                        } else if tr {
                                reasons = append(reasons, fmt.Sprintf("#%v", i+1))
                        }

                        if opts.and {
                                result = result && tr
                                opts.and = false // reset -and flag
                        } else if tr {
                                result = true
                                break
                        }
                }}
        }
        return
}

// (assert condition,'error message...')
func modifierAssert(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                res bool
                sco breaksco
                msg string
                err error
        )
        if res, sco, msg, err = predict(t, args...); err != nil {
                diag.errorAt(pos, "prediction %v failed: %v", args, err).debug(1)
        } else if !res {
                if msg == "" {
                        diag.errorAt(pos, "assertion failed: %v", args).debug(1)
                } else {
                        diag.errorAt(pos, "assertion failed: %v", msg).debug(1)
                }
                brk := brks.add(pos, breakFail)
                brk.message = "assertion failure"
                brk.scope = sco
        }
        return
}

func modifierCond(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                res bool
                sco breaksco
                msg string
                err error
        )
        if res, sco, msg, err = predict(t, args...); err != nil {
                diag.errorAt(pos, "predict: %v", err).debug(1)
        } else if !res {
                brk := brks.add(pos, breakDone)
                brk.message = msg
                brk.scope = sco
        }
        return
}

func modifierCase(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                res bool
                sco breaksco
                msg string
                err error
        )
        if res, sco, msg, err = predict(t, args...); err == nil {
                brk := brks.add(pos, breakNext) // next case
                brk.message = msg
                brk.scope = sco
                if res { brk.what = breakCase } // select case
        }
        return
}

type predictionDirtyOpts struct {
        checksum bool "c,checksum;c,crc"
        debug bool "d,debug"
        verbose bool "v,verbose"
        silent bool "s,silent"
}
func predictionDirty(t *traversal, args... Value) (result Value, err error) {
        var pos = t.Position()
        var opts predictionDirtyOpts
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var (
                target Value
                targetFullname string
                reason string
                dirty bool
        )
        // Wait for prerequisites only
        if target, _, _, err = t.wait(t); err != nil {
                diag.errorAt(pos, "waiting traversal failed: %v", err).debug(1)
                return
        } else if targetFullname, err = fullnameOrStrval(t, target); err != nil {
                diag.errorAt(pos, "strval '%v' failed: %v", target, err).debug(1)
                return
        } else if dirty = !t.exists(t, target); dirty {
                reason = "target not exists"
        } else if dirty = len(t.updated) > 0; dirty {
                reason = fmt.Sprintf("%v updated", len(t.updated))
        } else if dirty, err = t.isRecipesDirty(t); err != nil {
                diag.errorAt(pos, "isRecipesDirty: %v", err).debug(1)
                return
        } else if dirty {
                reason = "recipes changed"
        } else if !opts.checksum {
                // does nothing
        } else if depend0, _ := t.Get("<"); !(isNil(depend0) || isNone(depend0)) {
                var ( file2 string; same bool )
                if file2, err = fullnameOrStrval(t, depend0); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", depend0, err).debug(1)
                        return
                } else if same, err = crc64CompareFileChecksum(targetFullname, file2); err != nil {
                        diag.errorAt(pos, "crc64 checksum failed: %v", err).debug(1)
                        return
                } else if dirty = !same; dirty {
                        reason = "content changed"
                }
        }

        if opts.debug {
                var a = typeof(target)
                var e = t.exists(t, target)
                var s, _ = target.Strval(t)
                diag.errorAt(pos, "type=%s target=%s (exists=%v, dirty=%v, updated=%v)", a, s, e, dirty, t.updated).debug(1)
        }
        if opts.verbose {
                var ( m, s string )
                if dirty { m = "dirty" } else { m = "noop" }
                s = time.Now().Sub(t.start).String()
                if len(t.updated) > 0 { //s = fmt.Sprintf(", %v", t.updated)
                        s += "; "
                        for i, v := range t.updated {
                                if i > 0 { s += " " }
                                if len(s) > maxPromptStr {
                                        s += "…"
                                        break
                                } else { s += trimPromptString(v.String()) }
                        }
                } else if reason != "" {
                        s += "; " + strings.TrimSpace(strings.TrimPrefix(reason, "dirty:"))
                }
                var n = len(t.targets) + len(t.grepped)
                if false {
                        diag.prompt("stamp %s …… %s (%d files in %s)\n", target, m, n, s).debug(opts.debug, 1)
                } else {
                        diag.prompt("%s …… %s (%d files in %s)\n", trimPromptString(targetFullname), m, n, s).debug(opts.debug, 1)
                }
        }

        if optionTraceTraversal {
                t_traverse.tracef("dirty: %v (updated=%v, exists=%v, target=%v)", dirty, len(t.updated), t.exists(t, target), target)
                if len(t.updated) > 0 { t_traverse.tracef("dirty: updated=%v", t.updated) }
        }

        if opts.silent { reason = "" }
        result = &prediction{boolean{valbase{pos},dirty},reason}
        return
}

func predictionNoLoop(t *traversal, args... Value) (result Value, err error) {
        var loop bool
        var target, _ = t.Get("@")
        for caller := t.caller(); caller != nil; caller= caller.caller() {
                var ct, _ = caller.Get("@")
                var same = target == ct
                if !same && false {
                        same = (target.cmp(t, ct) == cmpEqual)
                }
                if same {
                        //fmt.Printf("%s: loop: %v\n", pos, t.def.target.value)
                        loop = true
                        break
                }
        }

        var s string
        if !loop { s = "not " }
        s = fmt.Sprintf("loop %sdetected (%v)", s, target)
        result = &prediction{boolean{valbase{t.Position()},!loop},s}
        return
}

type predictionTarget1stVisitOpts struct {
        silent bool "s,silent"
}
func predictionTarget1stVisit(t *traversal, args... Value) (result Value, err error) {
        var ( pos = t.Position(); opts predictionTarget1stVisitOpts )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var target, _ = t.Get("@")
        if isNil(target) {
                diag.errorAt(pos, "target is <nil>").debug(1)
                return
        }

        var num int
        for caller := t.caller(); caller != nil; caller = caller.caller() {
                if false {
                        var ct, _ = caller.Get("@")
                        var same = target == ct
                        if !same && false {
                                same = (target.cmp(t, ct) == cmpEqual)
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

        result = &prediction{boolean{valbase{pos},num==0},s}
        return
}

type predictionTargetMaxVisitOpts struct {
        closure bool "c,closure"
        debug bool "d,debug;d,debug-trace;d,dump"
        silent bool "s,silent"
}
func predictionTargetMaxVisit(t *traversal, args... Value) (result Value, err error) {
        var ( pos = t.Position(); opts predictionTargetMaxVisitOpts )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var nth int64
        for _, a := range args {
                if nth, err = a.Integer(t); err != nil {
                        diag.errorAt(pos, "%v", err).debug(1)
                        return
                } else if nth <= 0 {
                        diag.errorAt(pos, "needs positive number (%v, %s)", a, typeof(a)).debug(1)
                        return
                }
        }

        var ( num int64; head bool = true )
        var target, _ = t.Get("@")
        if isNil(target) {
                diag.errorAt(pos, "target is <nil>").debug(1)
                return
        }
        for caller := t.caller(); caller != nil; caller = caller.caller() {
                var ct, _ = caller.Get("@")
                if false {
                        if opts.closure && caller.closure == t.closure { continue }
                        var same = target == ct
                        if !same && false {
                                same = (target.cmp(t, ct) == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n := caller.execRec[target]; n > 0 {
                        num += int64(n)
                }
                if opts.debug && num > 0 {
                        if head { head = false
                                diag.prompt("  %s: nth(%d)\n", pos, nth)
                        }
                        var pos = caller.program.position
                        diag.prompt("    %s: %v\n", pos, ct)
                }
        }

        var s string;
        if opts.silent {
        } else if num == 0  { //s = "nth: zero"
        } else if num < nth { //s = "nth"
        } else { s = fmt.Sprintf("%d visits", num+1) }

        result = &prediction{boolean{valbase{pos},num<nth},s}
        return
}

type modifierGitModifiedOpts struct {
        debug bool "d,debug"
        verbose bool "v,verbose"
}
func modifierGitModified(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierGitModifiedOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil {
                diag.errorAt(pos, "git failed: %v", err).debug(1)
                return
        }
 
        // TODO: check also for `Changes not staged for commit:`

        var rx = regexp.MustCompile(`\n\tmodified:[\t ]*(.+?)\n`)
        var sm = rx.FindAllSubmatch(out.Bytes(), -1)
        if len(sm) > 0 {
                var pred = &prediction{boolean{valbase{pos},false},""}
                if result = pred; len(args) == 0 {
                        pred.bool, pred.reason = true, "modified"
                        return
                }
                for _, a := range args {
                        var s string
                        if s, err = a.Strval(t); err != nil {
                                diag.errorAt(pos, "strval '%v' failed: %v", err).debug(1)
                                return
                        }
                        for _, v := range sm {
                                if false { diag.prompt("%s: %s\n%v\n", pos, s, v[1]) }
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
func modifierGitAhead(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierGitAheadOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        } else if args, err = parseOpts(t, &opts, args...) ; err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err)
                return
        }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil {
                diag.errorAt(pos, "git: %v", err).debug(1)
                return
        }
 
        // TODO: check also for `Changes not staged for commit:`

        var rx = regexp.MustCompile(`\nYour branch is ahead of '(.+?)' by`)
        var sm = rx.FindAllSubmatch(out.Bytes(), 1)
        if len(sm) > 0 {
                var val bool = true
                result = &prediction{boolean{valbase{pos},val},"Work branch has new commits to push"}
        }
        return
}

var (
        onceMutex sync.Mutex
        onceCache = make(map[Value]int,64)
        onceSHA256Mutex sync.Mutex
        onceSHA256Cache = make(map[HashBytes]int,64)
)

func onceTest(ctx Context, tv Value) (n int) {
        onceMutex.Lock(); defer onceMutex.Unlock()
        onceCache[tv] += 1
        return onceCache[tv]
}

func onceSHA256Test(ctx Context, sum HashBytes) (n int) {
        onceSHA256Mutex.Lock(); defer onceSHA256Mutex.Unlock()
        onceSHA256Cache[sum] += 1
        return onceSHA256Cache[sum]
}

func onceSHA256(t *traversal, opts *modifierOnceOpts, args... Value) (result Value, brks breakers) {
        var ( pos = t.Position(); h = sha256.New(); s string )
        if true {
                // NOTE: entry and program are unique, since (once) is for runtime, we use their addresses.
                fmt.Fprintf(h, "%p%p", t.entry, t.program)
        } else {
                fmt.Fprintf(h, "%v%v", pos, t.program.position)
        }

        var target, _ = t.Get("@")
        if isNil(target) {
                diag.errorAt(pos, "target is <nil>").debug(1)
                return
        }

        var err error
        if s, err = fullnameOrStrval(t, target); err != nil {
                diag.errorAt(pos, "fullname '%v' failed: %v", target, err).debug(1)
                return
        } else if s != "" {
                fmt.Fprintf(h, "%s", s)
        }
        for _, a := range args {
                if s, err = fullnameOrStrval(t, a); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", a, err).debug(1)
                        return
                } else {
                        if false { diag.infoAt(pos, "%v", s).debug(true, 1) }
                        fmt.Fprintf(h, "%s", s)
                }
        }

        var sum HashBytes
        copy(sum[:], h.Sum(nil))

        var num = onceSHA256Test(t, sum)
        if opts.debug {
                diag.prompt("%s: %v (once: num=%d)\n", pos, target, num)
        } else if opts.verbose {
                diag.prompt("once: %v (num=%d)\n", target, num)
        }
        if num > 1 { brks.add(pos, breakDone).message = fmt.Sprintf("once (num=%d)", num) }
        return
}

type modifierOnceOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
        checksum bool `c,checksum;s,sha256`
}
func modifierOnce(t *traversal, args... Value) (result Value, brks breakers) {
        var (
                pos = t.Position()
                opts modifierOnceOpts
                err error
        )
        if args, err = mergeresult2(expandall2(t, expandPlainValue, args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(t, &opts, args...); err != nil {
                diag.errorAt(pos, "parse opts failed: %v", err).debug(1)
                return
        }


        if opts.checksum {
                result, brks = onceSHA256(t, &opts, args...)
        } else if target, _ := t.Get("@"); isNil(target) {
                diag.errorAt(pos, "target is <nil>").debug(1)
                return
        } else if !isNil(target) && !isNone(target) {
                var n = onceTest(t, target)
                if  n > 1 { brks.add(pos, breakDone).message = fmt.Sprintf(`executed %d times`, n) }
                if opts.debug { t.batch(func() {
                        diag.warnAt(pos, "%T %v %p %v", target, target, target, n).debug(16)
                        t.traceCallStack(pos, -1, "%p %v %v", target, target, n)
                })}
        }
        return
}
