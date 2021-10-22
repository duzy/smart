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
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("modifier.traverse(%s)", m))) }
        if options.traceTraversal   { defer un(tt(t_traverse, ctx, m)) }
        ctx = positional(ctx, m.position)
        if brks = ctx.Program().modify(ctx, m); !brks.has() {
                if n := ctx.countErrors(); n > 0 {
                        brks.add(m.position, breakFail).message = fmt.Sprintf("%s: %d errors", m.name, n)
                }
        } else if tb := brks.not(breakCase, breakNext, breakDone); tb.has() {
                for _, brk := range tb {
                        switch brk.what {
                        case breakFail: ctx.error("broken traversal for modifier %v failed: %v", m.name, brk.message).at(brk.pos)
                        case breakErro: ctx.error("broken traversal for modifier %v with error: %v", m.name, brk.error).at(brk.pos)
                        default: ctx.error("broken traversal for modifier %v (%v)", m.name, brk.what).at(brk.pos)
                        }
                }
                ctx.error("broken traversal for modifier %v in %v", m.name, ctx.Project()).debug(1)
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
        if options.traceTraversal { defer un(tt(t_traverse, ctx, g)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("modifiergroup.traverse(%s)", g))) }
        for _, m := range g.modifiers {
                if brks = m.traverse(ctx); !brks.has() { continue }
                if tb := brks.of(breakNext, breakCase, breakDone); tb.has() {
                        break
                } else if tb = brks.of(breakFail, breakErro); tb.has() {
                        for _, brk := range brks {
                                switch brk.what {
                                case breakFail: ctx.error("broken traversal for modifier %s with failure: %v", m.name, brk.message).at(brk.pos).debug(1)
                                case breakErro: ctx.error("broken traversal for modifier %s with error: %v", m.name, brk.error).at(brk.pos).debug(1)
                                }
                        }
                        ctx.error("broken traversal for modifier %s", m.name).at(m.position).debug(1)
                        break
                } else {
                        for _, brk := range brks {
                                ctx.error("%s: broken %v", m.name, brk.what).at(brk.pos)
                        }
                        ctx.error("%s: broken unexpectedly", m.name).at(m.position).debug(1)
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
                                                ctx.error("stringify '%v' failed: %v", elem, err).of(elem)
                                                return
                                        } else if strings.HasSuffix(str, "\n") {
                                                ctx.prompt("%s", str)
                                        } else if str != "" {
                                                ctx.prompt("%s\n", str)
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
                ctx.error("merge args failed: %v", err).at(pos).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).at(pos).debug(1)
                return
        } else if value, found := ctx.autoGet("-"); !found || isNil(value) {
                // ...
        } else if content, err = value.Strval(ctx) ; err != nil {
                ctx.error("stringify buffer value failed: %v", err).at(pos).debug(1)
                return
        }
        if opts.stdout { fmt.Fprint(stdout, content) }
        if opts.stderr { fmt.Fprint(stderr, content) }
        if opts.reset  { ctx.autoSet("-", MakeNone(pos)) }
        return
}

type modifierDebugOpts struct {
        info []Value `i,info`
        warn []Value `w,warn`
        error []Value `e,err;er,error`
        checkDirty bool `d,dirty;cd,checkdirty;cd,check-dirty`
}
func modifierDebug(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                pos = ctx.Position()
                opts modifierDebugOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).at(pos).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse opts failed: %v", err).at(pos).debug(1)
                return
        }

        var s string
        for _, info := range opts.info {
                if s, err = info.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).of(info).debug(1)
                        return
                }
                ctx.info("%s", s).of(info).debug(1)
        }
        for _, warn := range opts.warn {
                if s, err = warn.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).of(warn).debug(1)
                        return
                }
                ctx.warn("%s", s).of(warn).debug(1)
        }
        for _, error := range opts.error {
                if s, err = error.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).of(error).debug(1)
                        return
                }
                ctx.error("%s", s).of(error).debug(1)
        }
        var (
                target , _ = ctx.autoGet("@")
                depends, _ = ctx.autoGet("^")
                ordered, _ = ctx.autoGet("|")
                grepped, _ = ctx.autoGet("~")
        )
        if len(opts.info) == 0 && len(opts.warn) == 0 && len(opts.error) == 0 {
                ctx.warn("debug: %v %v", target, depends).at(pos).debug(1)
        }
        if opts.checkDirty && !isNil(target) {
                var tt = target.stat(ctx).mod()
                if tt.IsZero() {
                        ctx.info("target not exists: %v", target).at(pos).debug(1)
                        return
                }
                for _, dep := range merge(depends, ordered, grepped) {
                        var dt = dep.stat(ctx).mod()
                        if false { if s := dep.String(); strings.HasSuffix(s, ".o") {
                                ctx.info("%v -> %T %v, %v", target, dep, dep, dt.Sub(tt)).at(pos).debug(false, 1)
                        }}
                        if dt.After(tt) {
                                ctx.info("%v: outdated by %v (%v)", target, dep, dt.Sub(tt)).at(pos).debug(1)
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
                ctx.error("merge args failed: %v", err).at(pos).debug(1)
                return
        } else if g, ok := value.(*Group); ok && len(args) > 0 {
                var num int64
                if num, err = args[0].Integer(ctx); err != nil {
                        ctx.error("integify '%v' failed: %v", args[0], err).at(pos).debug(1)
                } else {
                        result = g.Get(int(num))
                }
        }
        return
}

func modifierEnv(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                pos = ctx.Position()
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("expand args failed: %v", err).at(pos).debug(1)
                return
        }

        var envars = new(List)
        for _, a := range args {
                if _, ok := a.(*Pair); ok { envars.Append(a) } else {
                        err = errors.New(fmt.Sprintf("invalid env '%v' (%s)", a, typeof(a)))
                        return
                }
        }
        if ctx.autoSet(TheShellEnvarsDef, envars); false {
                ctx.error("set '%s' failed: %v", TheShellEnvarsDef, envars).at(pos).debug(1)
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
func modifierSet(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                none = MakeNone(ctx.Position())
                opts modifierSetOpts
                defs []Value
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var program = ctx.Program()
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
                                ctx.error("strval '%v' failed: %v", a.Key, err).debug(1)
                                return
                        } else if value, err = a.Value.expand(ctx, expandPlainValue); err != nil {
                                ctx.error("expand '%v' failed: %v", a.Value, err).debug(1)
                                return
                        } else if isNil(value) { value = a.Value }
                        var t = ctx.traversal()
                        if false && name == "@" && t.entry.String() == "archive" {
                                ctx.info("%v -> %v", a.Value, value)
                                ctx.info("%s", ctx).debug(10)
                        }
                case *Flag:
                        if name, err = a.name.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", a.name, err).debug(1)
                                return
                        } else if value = none; name == "" { name = "-" }
                default:
                        ctx.error("%T `%s` is unsupported (try: foo=value)", arg, arg).debug(1)
                        return
                }
                if def = program.scope.FindDef(name); def == nil {
                        ctx.error("no such def '%s' (%v, %v)", name, arg, args).debug(16)
                        break ForArgs
                } else if err = def.val(ctx, value); err != nil {
                        ctx.error("set def '%s' failed: %v", name, err).debug(1)
                        return
                } else {
                        defs = append(defs, def)
                }
        }
        if len(defs) > 0 { result = MakeListOrScalar(ctx.Position(), defs) }
        return
}

type modifierClosureOpts struct {
        dump bool `d,dump`
        verbose bool `v,verbose`
}

// create closure context for the traversal
func modifierClosure(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierClosureOpts
                pos = ctx.Position()
                err error
        )
        // Closure the caller program, the context will be restored when execution is finished.
        if t := ctx.traversal(); t != nil {
                t.Context = closureWith(t.Context, pos/*, t.program.scope*/)
        } else {
                ctx.error("needs traversal context: %v", ctx).debug(1)
                return
        }

        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse closure opts failed: %v", err).debug(1)
                return
        }

        if opts.verbose {
                ctx.info("%v: %v", ctx.Project(), ctx).debug(1)
        }
        if opts.dump {
                callstack(ctx, -1, "call trace:")
        }

        var dir string // closure work directory
        if proj := ctx.Project(); proj == nil {
                ctx.error("nil project (%s)", ctx).debug(1)
        } else if scope := proj.scope; scope == nil {
                ctx.error("empty closure context").debug(1)
        } else if def := scope.FindDef("/"); def == nil {
                ctx.error("&/ is undefined").at(scope.position).debug(1)
        } else if dir, err = def.value.Strval(ctx); err != nil {
                ctx.error("%v", err).of(def.value).debug(1)
        } else if dir == "" {
                ctx.error("&/ is empty").at(scope.position).debug(1)
        } else if !filepath.IsAbs(dir) {
                ctx.error("&/ is relative").at(scope.position).debug(1)
        } else if err = enter(ctx, dir); err == nil {
                var program = ctx.Program()
                program.project.changedWD = dir
                program.changedWD = dir
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
                ctx.error("merge args failed: %v", err).debug(1)
                return
        }
        if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse cd opts failed: %v", err).debug(1)
                return
        }

        if opts.printEnter { printEnteringDirectory(ctx) }
        if opts.printLeave { printLeavingDirectory(ctx) }
        if (opts.printEnter || opts.printLeave) && len(args) == 0 { return }
        if len(args) == 1 {
                var dir string
                if dir, err = args[0].Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", args[0], err).debug(1)
                        return
                } else if dir == "" {
                        // TODO: do something special
                        return
                }
                var program = ctx.Program()
                if !filepath.IsAbs(dir) {
                        dir = filepath.Join(program.project.absPath, dir)
                }
                if opts.makePath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                                ctx.error("make path '%s' failed: %v", dir, err)
                                return
                        }
                }
                if err = enter(ctx, dir); err == nil {
                        program.project.changedWD = dir
                        program.changedWD = dir
                }
        } else {
                ctx.error("wrong number of cd args: %v", args).debug(1)
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
                ctx.error("merge mkdir args failed: %v", err).debug(1)
                return
        }
        if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse mkdir opts failed: %v", err).debug(1)
                return
        }
        if len(args) == 0 {
                var target, _ = ctx.autoGet("@")
                var s string
                if s, err = target.Strval(ctx); err != nil {
                        ctx.error("stringify target '%v' failed: %v", target, err).debug(1)
                } else if err = os.MkdirAll(filepath.Dir(s), opts.mode); err != nil {
                        ctx.error("make path '%s' failed: %v", s, err).debug(1)
                }
                return
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(ctx); err != nil {
                        ctx.error("stringify '%v' failed: %v", a, err).debug(1)
                        break
                }
                if err = os.MkdirAll(s, opts.mode); err != nil {
                        ctx.error("make path '%s' failed: %v", s, err).debug(1)
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
                ctx.error("merge path args failed: %v", err)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse path opts failed: %v", err)
                return
        }

        if len(args) == 0 {
                var target, _ = ctx.autoGet("@")
                var s string
                if s, err = target.Strval(ctx); err != nil {
                        ctx.error("stringify target value '%v' failed: %v", target, err).debug(1)
                } else if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                                ctx.error("make path '%s' failed: %v", err).debug(1)
                        }
                }
                return
        }

        for _, arg := range args {
                var s string
                if s, err = arg.Strval(ctx); err != nil {
                        ctx.error("stringify arg '%v' failed: %v", arg, err).of(arg).debug(1)
                        break
                }
                if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                        ctx.error("make path '%s' failed: %v", s, err).debug(1)
                        break
                }
        }
        return
}

func modifierSudo(ctx Context, args... Value) (result Value, brks breakers) {
        ctx.error("TODO: sudo modifier is not implemented yet").at(ctx.Position()).debug(1)
        return
}

func parseDependList(ctx Context, dependList *List) (depends *List, brks breakers) {
        var pos = ctx.Position()
        depends = new(List)
        for _, depend := range dependList.Elems {
                switch d := depend.(type) {
                case *List:
                        if dl, err := parseDependList(ctx, d); err != nil {
                                ctx.error("%v", err).debug(1)
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
                                ctx.error("unsupported entry depend `%v' (%v)", d, d.class).debug(1)
                        }
                case *String:
                        depends.Append(d)
                case *File:
                        depends.Append(d)
                default:
                        var program = ctx.Program()
                        ctx.error("unsupported entry depend `%v' (%v)", depend, program.depends).debug(1)
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
                ctx.error("'%v' not exists", g.target).at(g.target.Position()).debug(1)
                return
        }
        var tt time.Time = g.targetInfo.ModTime()
        for _, val := range g.files {
                var file, ok = val.(*File)
                if !ok { 
                        ctx.error("'%v' is not file (%T)", file, file).debug(1)
                        return
                }
                if file.info == nil && !file.isSysFile() {
                        var s string
                        if s, err = file.Strval(ctx); err != nil { ctx.error("%v", err); return }
                        if file.info, _ = os.Stat(s); file.info == nil { continue }
                        if gc.debug { ctx.warn("'%v' info is nil (%s)", file, file.fullname()) }
                }
                if file.info == nil {/* ... */} else
                if t := file.info.ModTime(); t.After(tt) {
                        if gc.debug { ctx.warn("touch %v → %v (%v)", g.target, file, t) }
                        if tt != t { tt = t }
                }
        }
        if tt.After(g.targetInfo.ModTime()) {
                if err = os.Chtimes(g.targetFullName, tt, tt); err != nil {
                        ctx.error("%v", err).debug(1)
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
        if !sys && file != nil && file.filemap != nil && len(file.filemap.Paths) == 1 {
                // system files defined by `files ((foo.xxx) ⇒ -)`
                if f, ok := file.filemap.Paths[0].(*Flag); ok {
                        sys = isNone(f.name) || isNil(f.name)
                }
        }
        if!sys && gc.debug {
                var t = ctx.traversal()
                ctx.error("%v: %v → %v (exists=%v, sys=%v, from %v)\n", t.entry.Target(), gc.target, name, file.exists(), sys, ctx.Project()).debug(1)
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
                                ctx.error("strval '%v' failed: %v", inc, e).of(inc).debug(1)
                        } else if file = stat(ctx, name, "", s); file != nil {
                                if false { ctx.info("%v in %v", file, inc).debug(1) }
                                return
                        }
                }
                if file == nil { file = stat(ctx, name, "", "", nil) }
                ctx.warn("'%s' not found in %v", name, ctx.Project()).at(gp)
                ctx.warn("grepped '%s' has no target dir in %v", name, ctx.Project())
                ctx.warn("from project %v (for %v)", ctx.Project(), name).at(ctx.Project().position).debug(8)
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
                                ctx.error("strval '%v' failed: %v", file, err).debug(1)
                                return
                        }
                        if file.info, err = os.Stat(s); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                        if false || gc.debug {
                                ctx.warn("'%v' info is nil (%s)", file, file.fullname()).debug(1)
                        }
                }
                if file.info == nil {/* ... */} else
                if tv := file.info.ModTime(); tv.After(tt) {
                        if true || gc.debug {
                                ctx.warn("touch %v → %v (%v)", gc.target, file, tv).debug(1)
                        }
                        tv = launchTime //time.Now() // ...
                        if err, tt = os.Chtimes(gc.targetFullName, tv, tv), tv; err != nil {
                                ctx.error("chtimes failed: %v", err).debug(1)
                                return
                        }
                }
        }

        // Report missing files, but system files are not treated as missing.
        if !gc.report {
                // ...
        } else if file == nil {
                ctx.info("%s: `%s` not found", ctx.Project().name, name).at(gp)
        } else if !file.exists() {
                ctx.info("%s: `%s` file not existed", ctx.Project().name, name).at(gp)
        }
        return
}

func tempFile(ctx Context, prefix, hashee0 string, hasheeN... interface{}) (file *File, err error) {
        var nameHash = sha256.New() // HashByte -> [sha256.Size]byte
        if _, err = fmt.Fprint(nameHash, prefix, hashee0); err != nil {
                ctx.error("hashing failed: %v", err).debug(1)
        } else if _, err = fmt.Fprint(nameHash, hasheeN...); err != nil {
                ctx.error("hashing failed: %v", err).debug(1)
        } else if nameSum := nameHash.Sum(nil); len(nameSum) != sha256.Size {
                ctx.error("hash sum invalid: %v", len(nameSum)).debug(1)
        } else {
                // Make names like .deps/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
                file = ctx.Project().matchTempFile(ctx, filepath.Join(prefix, // e.g. ".deps", ".grep"
                        fmt.Sprintf("%x", nameSum[ :1]),
                        fmt.Sprintf("%x", nameSum[1:2]),
                        fmt.Sprintf("%x", nameSum[2: ]),
                ))
        }
        return
}

func getSavedDepsFileName(ctx Context, targetFullName string, strs []string) (filename string, err error) {
        var ( file *File; hashees []interface{} )
        for _, s := range strs { hashees = append(hashees, s) }
        if file, err = tempFile(ctx, ".deps", targetFullName, hashees...); err != nil {
                ctx.error("get .deps temp file failed: %v", err).debug(1)
        } else if filename, err = fullnameOrStrval(ctx, file); err != nil {
                ctx.error("get .deps temp filename failed: %v", err).debug(1)
        }
        return
}

func getSavedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
        var ( file *File )
        if file, err = tempFile(ctx, ".grep", targetFullName); err != nil {
                ctx.error("get .grep temp file failed: %v", err).debug(1)
        } else if filename, err = fullnameOrStrval(ctx, file); err != nil {
                ctx.error("get .grep temp filename failed: %v", err).debug(1)
        }
        return
}

func loadSavedGrepFile(ctx Context, gc *grepctx) (okay bool, err error) {
        if gc.savedGrepFileName, err = getSavedGrepFileName(ctx, gc.targetFullName); err != nil {
                ctx.error("get saved grep filename failed: %v", err).debug(1)
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
                ctx.error("open saved grep filename failed: %v", err).debug(1)
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
                                ctx.error("search grepped filename failed: %v", err).debug(1)
                                break
                        } else if file != nil {
                                file.position = gp
                                if gc.isTargetFile(ctx, file) { continue }
                        } else if sys != 1 && !gc.discard {
                                ctx.warn("%s is nil file", name).at(gp)
                                ctx.warn("grepped %s is nil", name)
                                ctx.warn("from project %v", ctx.Project()).at(ctx.Project().position).debug(6)
                        }
                }
        }
        if gc.savedGrepFile.info, err = savedGrepOSFile.Stat(); err != nil {
                ctx.error("stat saved grep filename error: %v", err).debug(1)
        } else { okay = true }
        return
}

func grepTargetFile(ctx Context, gc *grepctx) (err error) {
        var ( file *os.File )
        if file, err = os.Open(gc.targetFullName); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else { defer func() { err = file.Close() } () }

        for _, x := range gc.rxs {
                if x.Regexp != nil {
                        continue
                } else if x.Regexp, err = regexp.Compile(x.string); err != nil {
                        ctx.error("%v", err).debug(1)
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
                                        ctx.error("search grepped '%s' failed: %v", name, err).debug(1)
                                        return
                                } else if file != nil {
                                        if file.position = gp; gc.isTargetFile(ctx, file) { continue }
                                } else if !sys && !gc.discard {
                                        ctx.warn("%s is nil file", name).at(gp)
                                        ctx.warn("grepped %s is nil", name)
                                        ctx.warn("from project %v", ctx.Project()).at(ctx.Project().position).debug(6)
                                }
                                continue ForScan // found one
                        }
                }
        }
        return
}

func grep(ctx Context, gc *grepctx) (err error) {
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
                        ctx.error("strval grep target '%v' failed: %s", v, err).debug(1)
                        return
                }
                if filepath.IsAbs(targetName) {
                        gc.targetFullName = targetName
                } else {
                        gc.targetFullName = filepath.Join(gc.targetDir, targetName)
                }
                if file := stat(ctx, gc.targetFullName, "", ""); file == nil {
                        ctx.error("grep: '%s' not found", gc.targetFullName).debug(1)
                        return
                } else {
                        gc.targetInfo = file.info
                }
        }
        if err != nil {
                ctx.error("grep target %s: %v", targetName, err).debug(1)
                return
        }

        if gc.targetInfo == nil { return }
        if gc.done == nil { gc.done = make(map[string]int) }
        if !filepath.IsAbs(gc.targetFullName) {
                ctx.error("grep: '%s' is not abs", gc.targetFullName).debug(1)
                return
        } else {
                gc.done[gc.targetFullName] += 1
        }
        if n, done := gc.done[gc.targetFullName]; done && n > 1 {
                if gc.debug { ctx.error("%v (done %v)", gc.targetFullName, n).debug(1) }
                return
        }

        //var infos = strings.Contains(gc.targetFullName, "...")
        const infos = false

        if false { defer un(tt(t_traverse, ctx.traversal(), gc.target)) }

        defer func(restore []Value) {
                var t = ctx.traversal()
                var touch = gc.greptouch // copy greptouch value
                if len(touch.files) > 0 {
                        grepcache[gc.targetFullName] = touch.files
                } else if false {
                        var gp Position
                        gp.Filename, gp.Line = gc.targetFullName, 1
                        ctx.warn("grebbed zero files").at(gp)
                        ctx.warn("grebbed zero files: %v", gc.targetFullName).debug(6)
                }
                gc.files = restore
                if gc.debug { ctx.error("grepped: %s → %v (grepped=%v) (saved=%s)\n",
                        gc.target, touch.files, len(t.grepped), gc.savedGrepFile).debug(1) }
                for _, gc.target = range touch.files {
                        if t.grepped = append(t.grepped, gc.target); !gc.recursive {
                                continue
                        } else if err = grep(ctx, gc); err != nil {
                                ctx.error("grep files (deferred): %v", err).debug(1)
                                break
                        }
                }
                if err == nil && gc.touch {
                        if err = touch.work(ctx, gc); err != nil {
                                ctx.error("grep touch failed: %v", err).debug(1)
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
                if gc.debug { ctx.error("grepcache: %v → %v", gc.targetFullName, gc.files).debug(1) }
                return
        } else if infos {
                ctx.info("grepcache: %s files=%d", gc.targetFullName, len(gc.files)).debug(1)
        }

        if savedGrepFileLoaded, err = loadSavedGrepFile(ctx, gc); err != nil {
                ctx.error("load saved grepfile failed: %v", err).debug(1)
                return
        } else if savedGrepFileLoaded && len(gc.files) > 0 {
                if infos { ctx.info("loadSavedGrepFile: %v files=%d grepped=%d",
                        gc.targetFullName, len(gc.files), len(ctx.traversal().grepped)).debug(1) }
                return
        }
        if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                        ctx.error("make grep dir failed: %v", err).debug(1)
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
                        ctx.error("grep write file: %v", err).debug(1)
                        return
                } else if false {
                        ctx.info("saved grep %s", name).debug(1)
                }
        }
        if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil {
                ctx.error("grep create %s: %v", gc.savedGrepFileName, err).debug(1)
                return
        }

        gc.save = bufio.NewWriter(savedGrepFile)
        defer func() {
                gc.save.Flush()
                savedGrepFile.Close()
        } ()

        if err = grepTargetFile(ctx, gc); err != nil && !gc.discard {
                ctx.error("grep target file: %v", err).debug(1)
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
        touch bool `ctx,touch;ctx,touch-outdate;ctx,touch-outdated`
        recursive bool `a,all;r,recur;rr,recursive`
        noTraverse bool `n,notraverse;nt,no-traverse;go,grep-only`
}
func modifierGrep(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                gc grepctx
                err error
        )
        gc.fileinc = true // grep files by default
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge grep args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &gc.modifierGrepOpts, args...); err != nil {
                ctx.error("parse grep args failed: %v", err).debug(1)
                return
        } else if gc.incs, err = expandmerge2(ctx, expandPlainValue, gc.incs...); err != nil {
                ctx.error("expand grep incs failed: %v", err).debug(1)
                return
        }
        for _, s := range gc.sys { gc.rxs = append(gc.rxs, &greprex{s, true , nil}) }
        for _, s := range gc.reg { gc.rxs = append(gc.rxs, &greprex{s, false, nil}) }
        for _, s := range gc.langs {
                if info, ok := langInfos[s]; ok && info != nil {
                        for _, re := range info.rxs { gc.rxs = append(gc.rxs, &greprex{re, false, nil}) }
                        for _, re := range info.sys { gc.rxs = append(gc.rxs, &greprex{re, true , nil}) }
                } else {
                        ctx.error("lang '%s' is unknown", s).debug(1)
                        return
                }
        }
        if len(gc.rxs) == 0 {
                ctx.error("no grep expressions: %v %v %v %v", gc.sys, gc.reg, gc.langs, args).debug(1)
                return
        }

        var (
                target, _ = ctx.autoGet("@")
                targets = args
                grepped = ctx.traversal().grepped
        )
        if len(targets) == 0 { if isNil(target) || isNone(target) {
                ctx.error("no grep target").debug(1)
                return
        } else {
                targets = append(targets, target)
        }}

        if gc.debug {
                ctx.warn("grep files: %v %v %v\n", target, gc.rxs, args).debug(1)
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
                        ctx.prompt("Grep %v …… (%d files in %v)\n", s, len(grepped), time.Now().Sub(ts)).debug(gc.debug, 6)
                } (time.Now())
        }

        var t = ctx.traversal()
        var tar = target
        defer func(v bool) { t.grepping = v } (t.grepping)
        t.grepping = true

ForTarget:
        for _, target := range targets {
                if isNil(target) {
                        ctx.error("found nil grep target for %v", tar).debug(1)
                        return
                }
                if isNone(target) {
                        ctx.error("grep target '%v' is none for %v", target, tar).debug(32)
                        return
                }

                gc.target, t.grepped = target, nil
                if err = grep(ctx, &gc); err != nil {
                        ctx.error("grep files from %v failed: %v", target, err).debug(1)
                        return
                } else if gc.noTraverse {
                        // does nothing
                } else if len(t.grepped) > 0 {
                        for _, val := range t.grepped {
                                if brks = val.traverse(ctx); !brks.has() { continue }
                                for _, brk := range brks {
                                        switch brk.what {
                                        case breakFail: ctx.error("broken traversal for grepped %v failed: %v", val, brk.message).at(brk.pos)
                                        case breakErro: ctx.error("broken traversal for grepped %v with error: %v", val, brk.error).at(brk.pos)
                                        default: ctx.error("broken traversal for grepped %v: %v (%v)", val, brk.message, brk.what).at(brk.pos)
                                        }
                                }
                                ctx.error("broken traversal for grepped %v from %v", val, target)
                                ctx.error("from project %v (for %v)", ctx.Project(), val).at(ctx.Project().position).debug(1)
                                break ForTarget
                        }
                }
                grepped = append(grepped, t.grepped...)
        }
        t.grepped = grepped

        var pos = ctx.Position()
        if err != nil {
                ctx.error("grep files failed: %v", err).debug(1)
        } else if !gc.noTraverse {
                ctx.autoSet("~", MakeNone(pos))
                t.grepped = nil
        } else {
                result = MakeListOrScalar(pos, t.grepped)
        }
        return
}

func parseDeps(ctx Context, savedDepsFileName, deps string) (files []Value, brks breakers) {
        var (
                target, _ = ctx.autoGet("@")
                targetFullName string
                filesMux sync.Mutex
                firstWord string
                dp Position
                err error
        )
        if targetFullName, err = fullnameOrStrval(ctx, target); err != nil {
                ctx.error("fullname '%v' failed: %v", target, err).debug(1)
                return
        } else { dp.Filename = savedDepsFileName }

        findDepFile := func(name string) (file *File) {
                if filepath.IsAbs(name) {
                        file = stat(ctx, name, "", "", nil)
                } else if file = ctx.Project().FindFile(ctx, name); file != nil && file.exists() {
                        // good!
                } else {
                        // fail!
                }
                return
        }
        ignored := func(fullname string) (res bool) {
                if fullname == targetFullName { return true }
                return
        }
        addFile := func(file *File) {
                filesMux.Lock(); defer filesMux.Unlock()
                files = append(files, file)
        }

        const parallel = true
        var wg sync.WaitGroup
        var depFile = func(ctx Context, word string) {
                if parallel {
                        defer checkPanicsErrors(ctx)
                        defer wg.Done() // minus 1
                }
                if i := strings.Index(word, " "); i > 0 {
                        ctx.warn("ignore dep with spaces: %v", word).debug(1)
                        //nxt = 1 //continue
                } else if file := findDepFile(word); file == nil {
                        ctx.error("unknown dep '%v' for '%v'", word, firstWord)
                        ctx.error("from here: %s", word).at(dp)
                        if filepath.IsAbs(firstWord) {
                                var wp Position; wp.Filename, wp.Line = firstWord, 1
                                ctx.error("in here: %v", word).at(wp)
                        }
                        ctx.error("for project %v", ctx.Project()).at(ctx.Project().position).debug(6)
                } else if ignored(file.fullname()) {
                        //nxt = 1 //continue // dep is the target itself
                } else if brks = file.traverse(ctx); brks.has() {
                        for _, brk := range brks {
                                switch brk.what {
                                case breakFail: ctx.error("broken traversal for dep '%v' failed: %v", file, brk.message).at(brk.pos)
                                case breakErro: ctx.error("broken traversal for dep '%v' with error: %v", file, brk.error).at(brk.pos)
                                default: ctx.error("broken traversal for dep '%v': %v (%v)", file, brk.message, brk.what).at(brk.pos)
                                }
                        }
                        ctx.error("missing dep '%v' for %v", file, target).at(dp)
                        ctx.error("broken traversal for dep '%v' from %v", file, target)
                        ctx.error("from project %v (for %v)", ctx.Project(), file).at(ctx.Project().position).debug(6)
                        //nxt = 2 //break ForLines
                } else {
                        addFile(file)
                }
                if n := ctx.countErrors(); n > 0 {
                        ctx.error(`%d errors for dep file "%s"`, n, word).debug(1)
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
                                        wg.Add(1); go depFile(ctx.spawn(), word)
                                } else {
                                        depFile(ctx, word)
                                }
                        }
                }
        }
        wg.Wait()
        return
}

func loadSavedDepsAndCheckOutdated(ctx Context, args []string) (savedDepsFileName string, files []Value, brks breakers) {
        var (
                savedDeps []byte
                err error
        )
        if targetVal, targetStr := getTargetValueString(ctx); isNil(targetVal) {
                ctx.error("target is nil").debug(1)
        } else if targetStr == "" {
                ctx.error("target '%v' is empty", targetVal).debug(1)
        } else if savedDepsFileName, err = getSavedDepsFileName(ctx, targetStr, args); err != nil {
                ctx.error("get saved deps filename failed: %v", err).debug(1)
        } else if savedDepsFileName == "" {
                ctx.error("empty saved deps filename", savedDepsFileName).debug(1)
        } else if savedDepsFile := stat(ctx, savedDepsFileName, "", ""); savedDepsFile == nil {
                // no saved deps file
        } else if savedDeps, err = ioutil.ReadFile(savedDepsFileName); err != nil {
                ctx.error("can'ctx open saved deps file: %v", savedDepsFileName, err).debug(1)
        } else if files, brks = parseDeps(ctx, savedDepsFileName, string(savedDeps)); len(files) > 0 {
                if false { ctx.info("loaded deps %s (%d files)", savedDepsFileName, len(files)).debug(true, 1) }
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
func modifierDeps(ctx Context, args... Value) (result Value, brks breakers) {
        // NOTE: parse opts for (deps) before expanding the args, because we share args
        //       with the compilers!
        var (
                opts modifierDepsOpts
                err error
        )
        /*ctx.info("%v", ctx)
        //ctx.info("%v", ctx.inner())
        //ctx.info("%v", ctx.inner().inner())
        for _, a := range args {
                v1, _ := a.expand(ctx, expandPlainValue)
                v2, _ := a.expand(ctx.inner().inner(), expandPlainValue)
                ctx.info("%T %v -> %v", a, a, v1)
                ctx.info("%T %v -> %v", a, a, v2)
        }
        ctx.info("%T", ctx.Context).debug(1)*/
        if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse deps args failed: %v", err).debug(1)
                return
        } else if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge deps args failed: %v", err).debug(1)
                return
        }

        var files []Value
        if opts.verbose {
                defer func(ts time.Time) {
                        var s string
                        if target, _ := ctx.autoGet("@"); !isNil(target) { s = target.String() }
                        ctx.prompt("Deps %v …… (%d files in %v)\n", s, len(files), time.Now().Sub(ts)).debug(opts.debug, 6)
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
                        ctx.error("unsupported cc: %v", opts.cc).debug(1)
                        return
                } else if strings.HasPrefix(base, "clang") { opts.useClang = true
                } else if strings.HasPrefix(base, "gcc")   { opts.useGcc   = true }
        }

        var flags []Value
        if flags, err = expandmerge2(ctx, expandPlainValue, opts.flags...); err != nil {
                ctx.error("merge flags failed: %v", err).debug(1)
                return
        }

        var (
                _MM, _MG bool
                ca []string
        )
        for _, f := range flags {
                var s string
                if s, err = f.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", err).debug(1)
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
                        ctx.error("strval '%v' failed: %v", err).debug(1)
                        return
                } else { s = strings.TrimSpace(s) }
                switch s {
                case "", "-M", "-MM", "-MG", "-MD", "-MV", "-MP", "-Os", "-O1", "-O2", "-O3",
                     "-c", "-shared", "-static", "-fPIC", "-fcxx-modules",
                     "-fvisibility-inlines-hidden": break // discard unused args
                default: ca = append(ca, s)
                }
        }

        var t = ctx.traversal()
        var savedDepsFileName string
        if savedDepsFileName, files, brks = loadSavedDepsAndCheckOutdated(ctx, ca); brks.has() {
                for _, brk := range brks {
                        ctx.error("borken loading saved deps: %v", brk.what).at(brk.pos).debug(1)
                }
                ctx.error("broken loading saved deps").debug(1)
                return
        } else if len(files) == 0 {
                var (
                        cc = exec.Command(opts.cc, ca...)
                        stdout bytes.Buffer
                        stderr bytes.Buffer
                )
                cc.Stdout, cc.Stderr = &stdout, &stderr
                if err = cc.Run(); err != nil {
                        if true { ctx.prompt("%s \\\n  %s\n%s\n----------\n%s.\n",
                                cc.Path, strings.Join(ca, " \\\n  "), &stdout, &stderr) }
                        ctx.error("deps with %s failed: %v", filepath.Base(opts.cc), err)
                        ctx.error("for project %v", ctx.Project()).at(ctx.Project().position).debug(6)
                        callstack(ctx, -1, "deps with %s faled: %v", filepath.Base(opts.cc), err)
                        return
                }
                stderr.Reset() // release buffers (optional)

                if savedDepsFileName == "" {
                        ctx.error("empty saved deps file name: %v", savedDepsFileName).debug(1)
                } else if err = os.MkdirAll(filepath.Dir(savedDepsFileName), os.FileMode(0755)); err != nil {
                        ctx.error("make path '%s' failed: %v", filepath.Dir(savedDepsFileName), err).debug(1)
                } else if err = ioutil.WriteFile(savedDepsFileName, stdout.Bytes(), os.FileMode(0666)); err != nil {
                        ctx.error("save deps file failed: %v", err).debug(1)
                } else if false {
                        ctx.info("saved deps %s", savedDepsFileName).debug(true, 1)
                }

                files, brks = parseDeps(ctx, savedDepsFileName, stdout.String())
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
func modifierTouch(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierTouchOpts // = modifierTouchOpts{ mode: os.FileMode(0755) }
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge touch args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse touch opts failed: %v", err).debug(1)
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
                        ctx.error("touch '%v' failed: %v", arg, err).debug(1)
                        break
                } else if vf, err = arg.stamp(ctx); err != nil {
                        ctx.error("touch '%v' failed: %v", arg, err).debug(1)
                        break
                } else { files = append(files, vf...) }
        }

        var t = ctx.traversal()
        if opts.verbose { reportFileUpdates(ctx, t.start, files) }
        if len(t.program.getModifiers(ctx, "stamp")) > 0 {
                ctx.warn("no need to use a (stamp) after (touch)").debug(1)
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
                ctx.error("merge check args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse check args failed: %v", err).debug(1)
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
                                ctx.error("unknown check '%v' (%T)", arg, arg).debug(1)
                        } else if makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else {
                                brks.addf(pos, optBreak, "value '%v' is false", arg)
                                if opts.verbose {
                                        ctx.warn("value '%v' is false", arg).debug(1)
                                }
                        }
                }
        }
        if !(isNil(opts.file) || isNone(opts.file)) {
                var ( s string; f *File )
                if f, res = opts.file.(*File); res {
                        if res = f.exists(); !res && opts.verbose {
                                ctx.warn("file '%v' does not exists", opts.file).of(opts.file).debug(1)
                        }
                } else if s, err = opts.file.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", opts.file, err).debug(1)
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
                        ctx.warn("'%v' is file: %v", opts.file, res).of(opts.file).debug(1)
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
                                ctx.warn("file '%v' does not exists", opts.dir).of(opts.dir).debug(1)
                        }
                } else if s, err = opts.dir.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", opts.dir, err).debug(1)
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
                        ctx.warn("'%v' is file: %v", opts.dir, res).of(opts.dir).debug(1)
                }
                if makeResult != nil {
                        values = append(values, makeResult(pos, res))
                } else if !res {
                        brks.addf(pos, optBreak, "value '%v' is not dir", opts.dir)
                        return
                }
        }

        var t = ctx.traversal()
ForPairs:
        for _, p := range pairs {
                var key, str string
                if key, err = p.Key.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", p.Key, err).of(p.Key).debug(1)
                        return
                }
                switch key {
                case "status":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                brks.addf(pos, optBreak, "value '%v' is not exec result", value)
                                ctx.error("value '%v' (%T) is not exec result", value, value).of(value).debug(6)
                                return
                        } else { /*exeres.wg.Wait()*/ }

                        var num int64
                        if num, err = p.Value.Integer(ctx); err != nil {
                                ctx.error("%v", err).of(p.Value).debug(1)
                                return
                        }
                        if opts.verbose {
                                ctx.prompt("checking status ")
                                if num != 0 { ctx.prompt("== %d ", num) }
                                ctx.prompt("…")
                        }

                        var good = exeres.Status == int(num)
                        if opts.verbose {
                                var s string 
                                if good { s = "Yes" } else { s = "No" }
                                ctx.prompt("… %s (%d)\n", s, exeres.Status)
                        }
                        if opts.debug {
                                var tar, _ = ctx.autoGet("@")
                                var val, _ = ctx.autoGet("-")
                                ctx.warn("%v: %v", t.entry, tar).at(t.program.position)
                                ctx.warn("status=%v", exeres.Status)
                                ctx.warn("hyphen=%v", val)
                                ctx.warn("context: %v", ctx).debug(1)
                        }

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
                                ctx.error("value '%v' (%T) is not exec result", value, value).of(value).debug(6)
                                return
                        } else { /*exeres.wg.Wait()*/ }

                        if opts.verbose {
                                ctx.prompt("checking %s (status=%d) … ", key, exeres.Status)
                        }
                        if opts.debug {
                                var tar, _ = ctx.autoGet("@")
                                var val, _ = ctx.autoGet("-")
                                ctx.warn("%v: %v", t.entry, tar).at(t.program.position)
                                ctx.warn("status=%v", exeres.Status)
                                ctx.warn("hyphen=%v", val)
                                ctx.warn("context: %v", ctx).debug(1)
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
                        } else if str, err = p.Value.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", p.Value, err).of(p.Value).debug(1)
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
                        } else if str, err = p.Value.Strval(ctx); err != nil {
                                ctx.error("strval '%v' failed: %v", p.Value, err).debug(1)
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
                                        if k, err = p.Key.Strval(ctx); err != nil { break ForPairs }
                                        var def = t.program.project.scope.FindDef(k)
                                        if def != nil {
                                                if a, err = p.Value.Strval(ctx); err != nil { break ForPairs }
                                                if b, err = def.value.Strval(ctx); err != nil { break ForPairs }
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
                        ctx.error("unknown check for %v -> %v", p.Key, p.Value).debug(1)
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
                if err == nil {
                        var file = stat(ctx, dst, "", "")
                        ctx.Globe().stamp(dst, file.info.ModTime())
                }
        } (def1.value, def2.value)

        var pos = ctx.Position()
        def1.value = MakeString(pos, dst)
        def2.value = MakeString(pos, src)

        var head, foot string
        if opts.head != nil {
                if head, err = opts.head.Strval(ctx); err != nil { ctx.error("%v", err); return }
                if false { fmt.Fprintf(stderr, "%s: %v => %s\n", opts.head.Position(), opts.head, head) }
        }
        if opts.foot != nil {
                if foot, err = opts.foot.Strval(ctx); err != nil { ctx.error("%v", err); return }
                if false { fmt.Fprintf(stderr, "%s: %v => %s\n", opts.foot.Position(), opts.foot, foot) }
        }

        // Compare mod time for update mode
        if opts.files += 1; opts.update {
                if st2, e := os.Stat(dst); e == nil && st2 != nil {
                        var st1 os.FileInfo
                        if st1, err = os.Stat(src); err != nil { ctx.error("%v", err); return }
                        if st1 != nil && (st1.Size()+int64(len(head))+int64(len(foot))) == st2.Size() {
                                if st2.ModTime().After(st1.ModTime()) { return }
                        }
                        if false { fmt.Fprintf(stderr, "%s: %s (%v,%v)\n", pos, dst, st1.Size(), st2.Size()) }
                }
        }

        var srcFile, dstFile *os.File
        if srcFile, err = os.Open(src); err != nil { ctx.error("%v", err); return } else {
                defer srcFile.Close()
        }

        // sys default file mode is 0666
        if opts.path { // Make path (mkdir -p)
                if p := filepath.Dir(dst); p != "." && p != "/" {
                        err = os.MkdirAll(p, os.FileMode(0755))
                        if err != nil { ctx.error("%v", err); return }
                }
        }

        if opts.mode == 0 { opts.mode = os.FileMode(0640) }

        dstFile, err = os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, opts.mode)
        if err != nil { ctx.error("%v", err); return } else { defer dstFile.Close() }

        srcBuf := bufio.NewReader(srcFile)
        dstBuf := bufio.NewWriter(dstFile)
        if head != "" {
                var n int
                if n, err = dstBuf.WriteString(head); err != nil { ctx.error("%v", err); return }
                opts.bytes += int64(n)
        }

        var n int64
        if n, err = io.Copy(dstBuf, srcBuf); err != nil { ctx.error("%v", err); } else {
                if opts.bytes += n; foot != "" {
                        var n int
                        if n, err = dstBuf.WriteString(foot); err != nil { ctx.error("%v", err); return }
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
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
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
                        ctx.error("strval '%v' failed: %v", target, err).debug(1)
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
                        ctx.error("strval '%v' failed: %v", source, err).debug(1)
                        return
                } else if file := project.FindFile(ctx, srcname); file != nil {
                        source, srcname = file, file.fullname()
                        if file.info != nil { srctime = file.info.ModTime() }
                }
        }

        if filepath.Base(srcname) != filepath.Base(filename) {
                a, _ := ctx.autoGet("@")
                b, _ := ctx.autoGet("<")
                c, _ := ctx.autoGet("^")
                ctx.warn("%v", a).at(a.Position())
                ctx.warn("%v", b).at(b.Position())
                ctx.warn("%v", c).at(c.Position())
                ctx.warn("%v, %v, %v", target, filename, srcname).debug(1)
        }

        if !filetime.IsZero() && filetime.After(srctime) {
          if opts.update {
            if opts.verbose { ctx.prompt("update %v …", target) }
          } else if opts.override {
            if opts.verbose { ctx.prompt("override %v …", target) }
          } else {
            if opts.verbose { ctx.prompt("copy %v …… already existed!\n", target) }
            if !opts.silent { ctx.error("file already existed (%s)", target).debug(1) }
            return
          }
        } else if opts.verbose {
                if opts.update {
                        ctx.prompt("Checking %v …", target)
                } else {
                        ctx.prompt("Copy %v …", target)
                }
        }

        if opts.quick {
                var file = stat(ctx,filename,"","",nil)
                if file == nil || file.info != nil {
                        if opts.verbose { ctx.prompt("… Good\n") }
                        return
                }
        }

        var t = ctx.traversal()
        var copts = &copyopts{
                t.program, opts.path||opts.recursive,
                opts.update, opts.mode, opts.head, opts.foot,
                0, 0, 0,
        }
        var file *File
        if file = stat(ctx,srcname,"","",nil); file == nil || file.info == nil {
                ctx.error("'%s' source file not found", srcname).debug(1)
        } else if !file.info.IsDir() {
                if opts.mode == 0 { opts.mode = file.info.Mode() }
                if err = copyFile(ctx, file.info, srcname, filename, copts); err != nil {
                        ctx.error("%v", err).debug(1)
                }
        } else if opts.recursive {
                if err = copyDir(ctx, srcname, filename, copts); err != nil {
                        ctx.error("%v", err).debug(1)
                }
        } else {
                ctx.error("`%v` is a directory (use -r to solve it)", source).debug(1)
        }

        if opts.verbose {
                if err != nil {
                        ctx.prompt("… error\n")
                } else if copts.copied == 0 {
                        ctx.prompt("… Good (%d files)\n", copts.files)
                } else if copts.copied == 1 {
                        ctx.prompt("… Copied %d bytes\n", copts.bytes)
                } else {
                        ctx.prompt("… Copied %d bytes (%d/%d)\n", copts.bytes, copts.copied, copts.files)
                }
        }
        return
}

func modifierWriteFile(ctx Context, args... Value) (result Value, brks breakers) {
        var err error
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        }

        var (
                target, _ = ctx.autoGet("@")
                filename, str string
                f *os.File
        )
        defer func() {
                if err != nil && filename != "" { os.Remove(filename); f = nil }
                if f == nil { brks.add(ctx.Position(), breakFail).message = fmt.Sprintf("file %s not generated", target) }
        } ()
        if isNil(target) {
                ctx.error("target is undefined").debug(1)
                return
        } else if filename, err = fullnameOrStrval(ctx, target); err != nil {
                ctx.error("fullname failed: %v", err).debug(1)
                return
        } else if buffer, _ := ctx.autoGet("-"); isNil(buffer) {
                ctx.error("buffer value is nil").debug(1)
                return
        } else if str, err = buffer.Strval(ctx); err != nil {
                ctx.error("strval buffer failed: %v", err).debug(1)
                return
        } else if f, err = os.Create(filename); err != nil {
                ctx.error("%v", err).debug(1)
                return
        } else if _, err = f.WriteString(str); err != nil {
                f.Close()
                ctx.error("%v", err).debug(1)
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
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var target Value
        if n := len(args); n > 1 {
                ctx.error("too many files: %v", args).debug(1)
                return
        } else if n == 1 {
                target = args[0]
        } else {
                target, _ = ctx.autoGet("@")
        }

        if isNil(target) {
                ctx.error("target is <nil>").debug(8)
                return
        } else if isNone(target) {
                ctx.error("target is <none>").debug(8)
                return
        } else if filename, err = fullnameOrStrval(ctx, target); err != nil {
                ctx.error("strval '%v' error: %v", target, err).of(target).debug(1)
                return
        } else if filename == "" {
                ctx.error("target filename is empty").of(target).debug(1)
                return
        }

        if opts.debug {
                ctx.info("read-file: %v", filename)
        }

        var bytes []byte
        if bytes, err = ioutil.ReadFile(filename); err == nil {
                var s, v string
                if opts.head != nil {
                        if v, err = opts.head.Strval(ctx); err == nil { s = v } else {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                }
                s += string(bytes)
                if opts.foot != nil {
                        if v, err = opts.foot.Strval(ctx); err == nil { s += v } else {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                }
                ctx.autoSet("-", MakeString(ctx.Position(), s))
        } else {
                brks.add(ctx.Position(), breakErro).error = err
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
            ctx.prompt("crc64CheckFileModeContent: %v %v\n%s\n%s\n", a, b, s, content)
          }
        }
        return
}

func crc64CompareFileChecksum(ctx Context, filename1, filename2 string) (same bool, err error) {
        var s []byte
        if s, err = ioutil.ReadFile(filename1); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }
        return crc64CheckFileModeContent(ctx, filename2, s, 0)
}

type modifierUpdateFileOpts struct {
        debug bool "d,debug"
        verbose bool "v,verbose"
        path bool "p,path"
        zero bool `z,zero;e,empty;az,allow-zero;ae,allow-empty`
        mode os.FileMode "m,mode"
}
func modifierUpdateFile(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts = modifierUpdateFileOpts{ mode: os.FileMode(0640) }
                filename string
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var target Value
        if len(args) > 0 { target = args[0] } else { target, _ = ctx.autoGet("@") }
        if len(args) > 1 { if opts.mode, err = permVal(ctx, args[1], 0600); err != nil {
                ctx.error("perm value '%v' failed: %v", args[1], err).of(args[1]).debug(1)
                return
        }}

        // Get target filename
        switch p := target.(type) {
        case *File: filename = p.fullname()
        case *Path:
                if filename, err = p.Strval(ctx); err != nil {
                        ctx.error("strval path '%v' failed: %v", p, err).debug(1)
                        return
                }
        default:
                if filename, err = target.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", p, err).debug(1)
                        return
                } else if file := ctx.Project().FindFile(ctx, filename); file != nil {
                        target, filename = file, file.fullname()
                }
        }

        if opts.debug {
                ctx.info("update-file: %v (%v) (%v, %v)", target, filename, ctx.Project(), ctx).debug(1)
        }

        if opts.path { // Make path (mkdir -p)
                if p := filepath.Dir(filename); p != "." && p != "/" {
                        if err = os.MkdirAll(p, os.FileMode(0755)); err != nil {
                                ctx.error("%v", err).debug(1)
                                return
                        }
                }
        }

        // Check existed file content checksum
        var content string
        if value, found := ctx.autoGet("-"); !found || isNil(value) {
                // no buffer value
        } else if content, err = value.Strval(ctx); err != nil {
                ctx.error("%v", err).debug(1)
                return
        }

        if content == "" {
                if !opts.zero {
                        if file := stat(positional(ctx, target.Position()), filename, "", ""); file != nil && file.info != nil && file.info.Size() == 0 {
                                file.info = nil
                                if err = os.Remove(filename); err != nil {
                                        ctx.error("remove file failed: %v", err).debug(1)
                                }
                        }
                        if s := target.String(); filepath.IsAbs(s) {
                                ctx.error("empty content for '%s'", s).debug(1)
                        } else {
                                ctx.error("empty content for '%s' (at %s)", s, filename).debug(1)
                        }
                        return
                } else if opts.verbose || opts.debug {
                        ctx.warn("empty content for '%v'", target).debug(1)
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
                        printEnteringDirectory(ctx)
                        ctx.prompt("update %v …… %s (in %v)\n", trimPromptString(target.String()), s, time.Now().Sub(st)).debug(opts.debug, 6)
                } (time.Now())
        }

        if same, err = crc64CheckFileModeContent(ctx, filename, []byte(content), opts.mode); err != nil {
                if _, ok := err.(*os.PathError); ok {
                        err = nil // discard path error (e.g. no such file or directory)
                } else {
                        ctx.error("crc64 checksum failed: %v", err).debug(1)
                        return
                }
        } else if same {
                removeCallerUpdated(ctx, target) // remove timestamp updated
                result = stat(ctx, filename, "", "")
                return
        }

        // Create or update the file with new content

        var f *os.File
        if f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts.mode); err != nil {
                brks.add(ctx.Position(), breakFail).message = fmt.Sprintf("update %v failed", target)
                ctx.error("open file failed: %v", err).debug(1)
        } else if f != nil {
                defer func() {
                        if err = f.Close(); err != nil {
                                os.Remove(filename)
                                ctx.error("close file '%s' failed: %v", filename, err).debug(1)
                                return
                        }
                        var file = stat(ctx, filename, "", "")
                        if  file == nil {
                                ctx.error("invalid file '%s'", filename).debug(1)
                        } else {
                                var files []*File
                                if files, err = file.stamp(ctx); err != nil {
                                        ctx.error("%v", err).debug(1)
                                        return
                                } else if false && opts.verbose {
                                        var t = ctx.traversal()
                                        reportFileUpdates(ctx, t.start, files)
                                }
                                result = file // resulting the updated file
                        }
                } ()
                if wrote, err = f.WriteString(content); err != nil {
                        ctx.error("write content failed: %v", err).debug(1)
                }
        } else {
                brks.add(ctx.Position(), breakFail).message = fmt.Sprintf("%v not updated", target)
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
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var (
                execRes *ExecResult
                waitForExecResult = opts.stdout || opts.stderr || opts.status || opts.execRes
                stampCurrentTarget = !opts.noTarget
                target, _ = ctx.autoGet("@")
                t = ctx.traversal()
        )
        if opts.verbose {
                defer func (st time.Time) {
                        var s string; if err != nil { s = "fail" } else { s = "done" }
                        ctx.prompt("Wait %v …… %s, result=%v, updated=%v\n", target, s, execRes, t.updated).debug(opts.debug, 1)
                        if opts.debug { ctx.info("%v", execRes).debug(6) }
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
        for _, file := range files {
                var (
                        mod = file.info.ModTime()
                        d = time.Now().Sub(start)
                )
                if mod.After(start) {
                        if false {
                                ctx.prompt("Updated %v (%v, ModTime=%v)\n", file, d, mod)
                        } else {
                                ctx.prompt("Updated %v (%v)\n", file, d)
                        }
                } else {
                        ctx.prompt("File %v not changed (%v, ModTime=%v)\n", file, d, mod)
                        ctx.warn("incorrect timestamp: %v (JobTime=%v, ModTime=%v)", file, start, mod)
                        ctx.warn("the target path name is: %v", file.fullname())
                        ctx.warn("try 'touch' the target %v if the path name and command are correct", file)
                        ctx.info("you may ignore the warnings if all correct")
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
func modifierStamp(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierStampOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err)
                return
        }

        // Wait for ExecResult (see also modifier (wait -exec))
        const waitForExecResult = true
        const stampCurrentTarget = true

        // Wait for prerequisites
        var pos = ctx.Position()
        var t = ctx.traversal()
        var target Value
        if target, _, _, err = wait(ctx, opts.prompt, waitForExecResult, stampCurrentTarget); err == nil {
                return
        } else if opts.next {
                if opts.verbose { ctx.warn("%v", err).debug(1) }
                brks.add(pos, breakNext).scope = breakTrave
                err = nil // discard the error
        } else if opts.error {
                if opts.debug > 0 {
                        callstack(ctx, -1, "%v", err).debug(opts.debug)
                } else {
                        ctx.error("%v", err).debug(1)
                }
                brks.add(pos, breakErro).error = err
        } else if t.stems != nil {
                if opts.debug > 0 {
                        ctx.warn("%v", err).debug(opts.debug)
                        callstack(ctx, -1, "%v", err).debug(opts.debug)
                } else {
                        ctx.warn("%v", err).debug(1)
                }
                brks.add(pos, breakNext).scope = breakTrave
                err = nil // discard the error
        } else if pos.IsValid() {
                callstack(ctx, -1, "failed: %v", err).debug(1)
        } else if targetPos := target.Position(); targetPos.IsValid() {
                callstack(positional(ctx, targetPos), -1, "failed: %v", err).debug(1)
        } else {
                // TODO: dump more diagnostics information here
        }

        if err != nil {
                if pe, ok := err.(*fs.PathError); ok {
                        ctx.error("stamp %s: %v", trimPromptString(pe.Path), pe.Err)
                        err = pe.Err
                }
        }
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
        if false && targetVal.String() == "ISL_GIT_HEAD_ID" {
                defer func() {
                        ctx.info("%v: %v", targetVal, args).debug(6)
                } ()
        }
        if isNil(targetVal) || isNone(targetVal) {
                ctx.error("target is <nil>")
                ctx.error("target is <nil>: %v", ctx).debug(1)
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
                ctx.prompt("… %s\n", status)
        } } ()

ForArgs:
        for _, arg := range args {
                switch tv := arg.(type) {
                case *String: message = tv.string; continue ForArgs
                case *Compound: if message, err = tv.Strval(ctx); err != nil {
                        ctx.error("strval '%v' failed: %v", tv, err).of(tv).debug(1)
                        return
                } else { continue ForArgs }}

                var va []Value
                if va, err = expandmerge2(ctx, expandPlainValue, arg); err != nil {
                        ctx.error("merge arg '%v' failed: %v", arg, err).debug(1)
                        return
                } else if va, err = parseOpts(ctx, &opts, va...); err != nil {
                        ctx.error("parse opts failed: %v", err).debug(1)
                        return
                } else if len(va) == 0 { continue ForArgs }
                if opts.group    { breakScope = breakGroup }
                if opts.traverse { breakScope = breakTrave }
                if opts.verbose && !opts.verbose0 {
                        if targetStr, err = fullnameOrStrval(ctx, targetVal); err != nil {
                                ctx.error("fullname-strval '%v' failed: %v", targetVal, err).debug(1)
                                return
                        }
                        ctx.prompt("checking %v …", filepath.Base(targetStr))
                        opts.verbose0 = true
                }

                if false && targetVal.String() == "ISL_GIT_HEAD_ID" {
                        ctx.info("arg: %v -> %v -> result = %v", arg, va, result)
                        ctx.info("arg: %v", ctx).debug(1)
                }

                if !opts.and && result { break }
                if !opts.and || (opts.and && result) { for i, a := range va {
                        var (
                                name string
                                val Value
                                tru bool
                        )
                        if false && targetVal.String() == "ISL_GIT_HEAD_ID" {
                                ctx.info("arg: %v, %d, %T %v", arg, i, a, a).debug(1)
                        }
                        if g, ok := a.(*Group); !ok {
                                // preserved the value of 'a'
                        } else if len(g.Elems) == 0 {
                                ctx.warn("predictor is empty group").at(g.position).debug(1)
                                a = nil // not prediction group
                        } else if name, err = g.Elems[0].Strval(ctx); err != nil {
                                ctx.error("strval predictor failed: %v", err).of(g.Elems[0]).debug(1)
                                return
                        } else if pret, ok := predictors[name]; !ok {
                                ctx.warn("predictor '%s' undefined (%T %v)", name, a, a).at(g.position).debug(1)
                                a = nil // no such named predictor
                        } else if val, err = pret(positional(ctx, g.Elems[0].Position()), g.Elems[1:]...); err != nil {
                                ctx.error("prediction '%v' failed: %v", g.Elems[0], err).of(a).debug(1)
                                return
                        } else {
                                a = val // reset the value of 'a'
                        }

                        if false && targetVal.String() == "ISL_GIT_HEAD_ID" {
                                ctx.info("arg: %v, result = %v", arg, result)
                                ctx.info("arg: %v, %d, %T %v", arg, i, a, a).debug(1)
                        }

                        if a == nil {
                                ctx.warn("predictor #%d is <nil>", i).debug(1)
                                continue // skip
                        } else if p, ok := a.(*prediction); ok {
                                if p.reason != "" { reasons = append(reasons, p.reason) }
                                tru = p.bool
                        } else if tru, err = a.True(ctx); err != nil {
                                ctx.error("truthify '%v' failed: %v", a, err).of(a).debug(1)
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
                ctx.error("prediction %v failed: %v", args, err).debug(6)
        } else if !res {
                if msg == "" {
                        ctx.error("assertion failed: %v", args).debug(6)
                } else {
                        var target, _ = ctx.autoGet("@")
                        var vals, _ = expandmerge2(ctx, expandPlainValue, args...)
                        ctx.error("assertion failed: %v (target = %s)", msg, target)
                        ctx.error("assertion args: %v", args)
                        ctx.error("assertion args: %v (expandmerged)", vals)
                        ctx.error("assertion context: %v", ctx).debug(6)
                }
                brk := brks.add(ctx.Position(), breakFail)
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
                ctx.error("predict: %v", err).debug(1)
        } else if !res {
                brk := brks.add(ctx.Position(), breakDone)
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
                var pos = ctx.Position()
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
func predictionDirty(ctx Context, args... Value) (result Value, err error) {
        var opts predictionDirtyOpts
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var (
                t = ctx.traversal()
                target Value
                targetFullname string
                reason string
                dirty bool
        )
        // Wait for prerequisites only
        if target, _, _, err = wait(ctx); err != nil {
                ctx.error("waiting traversal failed: %v", err).debug(1)
                return
        } else if targetFullname, err = fullnameOrStrval(ctx, target); err != nil {
                ctx.error("strval '%v' failed: %v", target, err).debug(1)
                return
        } else if dirty = !exists(ctx, target); dirty {
                reason = "target not exists"
        } else if dirty = len(t.updated) > 0; dirty {
                reason = fmt.Sprintf("%v updated", len(t.updated))
        } else if dirty, err = isRecipesDirty(ctx); err != nil {
                ctx.error("isRecipesDirty: %v", err).debug(1)
                return
        } else if dirty {
                reason = "recipes changed"
        } else if !opts.checksum {
                // does nothing
        } else if depend0, _ := ctx.autoGet("<"); !(isNil(depend0) || isNone(depend0)) {
                var ( file2 string; same bool )
                if file2, err = fullnameOrStrval(ctx, depend0); err != nil {
                        ctx.error("strval '%v' failed: %v", depend0, err).debug(1)
                        return
                } else if same, err = crc64CompareFileChecksum(ctx, targetFullname, file2); err != nil {
                        ctx.error("crc64 checksum failed: %v", err).debug(1)
                        return
                } else if dirty = !same; dirty {
                        reason = "content changed"
                }
        }

        if opts.debug {
                var a = typeof(target)
                var e = exists(ctx, target)
                var s, _ = target.Strval(ctx)
                ctx.error("type=%s target=%s (exists=%v, dirty=%v, updated=%v)", a, s, e, dirty, t.updated).debug(1)
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
                        ctx.prompt("stamp %s …… %s (%d files in %s)\n", target, m, n, s).debug(opts.debug, 1)
                } else {
                        ctx.prompt("%s …… %s (%d files in %s)\n", trimPromptString(targetFullname), m, n, s).debug(opts.debug, 1)
                }
        }

        if options.traceTraversal {
                t_traverse.tracef("dirty: %v (updated=%v, exists=%v, target=%v)", dirty, len(t.updated), exists(ctx, target), target)
                if len(t.updated) > 0 { t_traverse.tracef("dirty: updated=%v", t.updated) }
        }

        if opts.silent { reason = "" }
        result = MakePrediction(ctx.Position(), dirty, reason)
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
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var target, _ = ctx.autoGet("@")
        if isNil(target) {
                ctx.error("target is <nil>").debug(1)
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
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var nth int64
        for _, a := range args {
                if nth, err = a.Integer(ctx); err != nil {
                        ctx.error("%v", err).debug(1)
                        return
                } else if nth <= 0 {
                        ctx.error("needs positive number (%v, %s)", a, typeof(a)).debug(1)
                        return
                }
        }

        var ( num int64; head bool = true )
        var target, _ = ctx.autoGet("@")
        if isNil(target) {
                ctx.error("target is <nil>").debug(1)
                return
        }
        for caller := ctx.traversal().caller(); caller != nil; caller = caller.caller() {
                var ct, _ = caller.autoGet("@")
                if n := caller.execRec[target]; n > 0 {
                        num += int64(n)
                }
                if opts.debug && num > 0 {
                        if head { head = false
                                ctx.prompt("  %s: nth(%d)\n", ctx.Position(), nth)
                        }
                        var pos = caller.program.position
                        ctx.prompt("    %s: %v\n", pos, ct)
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
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil {
                ctx.error("git failed: %v", err).debug(1)
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
                                ctx.error("strval '%v' failed: %v", err).debug(1)
                                return
                        }
                        for _, v := range sm {
                                if false { ctx.prompt("%s: %s\n%v\n", pos, s, v[1]) }
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
                ctx.error("merge args failed: %v", err)
                return
        } else if args, err = parseOpts(ctx, &opts, args...) ; err != nil {
                ctx.error("parse opts failed: %v", err)
                return
        }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil {
                ctx.error("git: %v", err).debug(1)
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

func onceSHA256(ctx Context, opts *modifierOnceOpts, args... Value) (result Value, brks breakers) {
        var (
                t = ctx.traversal()
                h = sha256.New()
                s string
        )
        if true {
                // NOTE: entry and program are unique, since (once) is for runtime, we use their addresses.
                fmt.Fprintf(h, "%p%p", t.entry, t.program)
        } else {
                fmt.Fprintf(h, "%v%v", ctx.Position(), t.program.position)
        }

        var target, found = ctx.autoGet("@")
        if !found || isNil(target) {
                ctx.error("target is <nil>").debug(1)
                return
        }

        var err error
        if s, err = fullnameOrStrval(ctx, target); err != nil {
                ctx.error("fullname '%v' failed: %v", target, err).debug(1)
                return
        } else if s != "" {
                fmt.Fprintf(h, "%s", s)
        }
        for _, a := range args {
                if s, err = fullnameOrStrval(ctx, a); err != nil {
                        ctx.error("strval '%v' failed: %v", a, err).debug(1)
                        return
                } else {
                        if false { ctx.info("%v", s).debug(true, 1) }
                        fmt.Fprintf(h, "%s", s)
                }
        }

        var sum HashBytes
        copy(sum[:], h.Sum(nil))

        var num = onceSHA256Test(ctx, sum)
        if opts.debug {
                ctx.info("%v (once: num=%d)\n", target, num)
        } else if opts.verbose {
                ctx.prompt("once: %v (num=%d)\n", target, num)
        }
        if num > 1 { brks.add(ctx.Position(), breakDone).message = fmt.Sprintf("once (num=%d)", num) }
        return
}

type modifierOnceOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
        checksum bool `c,checksum;s,sha256`
}
func modifierOnce(ctx Context, args... Value) (result Value, brks breakers) {
        var (
                opts modifierOnceOpts
                err error
        )
        if args, err = expandmerge2(ctx, expandPlainValue, args...); err != nil {
                ctx.error("merge args failed: %v", err).debug(1)
                return
        } else if args, err = parseOpts(ctx, &opts, args...); err != nil {
                ctx.error("parse opts failed: %v", err).debug(1)
                return
        }


        if opts.checksum {
                result, brks = onceSHA256(ctx, &opts, args...)
        } else if target, found := ctx.autoGet("@"); !found || isNil(target) {
                ctx.error("target is <nil>").debug(1)
                return
        } else if !isNil(target) && !isNone(target) {
                var n = onceTest(ctx, target)
                if  n > 1 { brks.add(ctx.Position(), breakDone).message = fmt.Sprintf(`executed %d times`, n) }
                if opts.debug {
                        ctx.warn("%T %v %p %v", target, target, target, n).debug(16)
                        callstack(positional(ctx, target.Position()), -1, "%p %v %v", target, target, n)
                }
        }
        return
}
