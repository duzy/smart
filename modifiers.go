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
        "io"
        "io/fs"
        "io/ioutil"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "strings"
        "syscall"
        "sync"
        "time"
)

var launchTime = time.Now()

const (
        TheShellEnvarsDef = "shell→envars" // '→' ' → '
        TheShellStatusDef = "shell→status" // status code of execution
)

type generalOpts struct {
        debug   int  `d,db,debug` // NOTE: compatible with 'bool'
        verbose bool `v,verb,verbose`
        timing  bool `t,time,timing`
}
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
func (m *modifier) expand(ctx Context, _ expandwhat) (Value) { return m }
func (_ *modifier) cmp(ctx Context, v Value) (res cmpres) {
        if _, ok := v.(*modifier); ok { res = cmpEqual }
        return
}
func (m *modifier) traverse(ctx Context) (traves travestates) {
        ctx = positional(ctx, m.position)
        traves = ctx.program().modify(ctx, m)
        if n := ctx.checkErrors(true); n > 0 { // if n := ctx.countErrors(); n > 0 {
                brk := traves.add(ctx, traveFail, nil)
                brk.error = fmt.Errorf("%s: %d errors counted", m.name, n)
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
func (g *modifiergroup) expand(ctx Context, _ expandwhat) (Value) { return g }
func (_ *modifiergroup) cmp(ctx Context, v Value) (res cmpres) {
        if _, ok := v.(*modifiergroup); ok { res = cmpEqual }
        return
}
func (g *modifiergroup) traverse(ctx Context) (traves travestates) {
        for _, m := range g.modifiers {
                var ctx = positional(ctx, m.position)
                if t := m.traverse(ctx); t.has() {
                        traves = append(traves, t...) // collect travestates
                } else {
                        continue
                }
                if t := traves.of(traveFail); t.has() {
                        return
                } else if t = traves.not(traveCase, traveDone, traveNext); t.has() {
                        if true || (options.verbose || options.verboseBreaks) {
                                var _, ent, _ = entryStr(ctx, ctx.entry())
                                warn(ctx, "%v: %s failed\n", ent, m.name)
                                for _, s := range t { warn(ctx, "%v: %v", m.name, s).at(s.pos) }
                                warnstack(ctx, 5, "").debug(16)
                        }
                        break
                } else if t = traves.of(traveCase); t.has() {
                        continue // case selected
                } else if t = traves.of(traveDone, traveNext); t.has() {
                        return // done or try next rule entry
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
        ModifierFunc   func(Context, ...Value) (Value, travestates)
        PredictionFunc func(Context, ...Value) (Value)
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
                `if`:           modifierCond,
                `where`:        modifierCond,

                `once`:         modifierOnce,

                `fork`:         modifierFork,

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
        stdout bool `o,stdout`
        stderr bool `e,stderr`
        reset  bool `r,reset`
}
func modifierPrint(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                pos = ctx.Position()
                opts = modifierPrintOpts{ stderr: true }
                content string
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
        if value, found := ctx.autoGet("-"); !found || isNil(value) {
                // ...
        } else {
                content = value.Strval(ctx)
        }
        if opts.stdout { fmt.Fprint(stdout, content) }
        if opts.stderr { fmt.Fprint(stderr, content) }
        if opts.reset  { ctx.autoSet("-", MakeNone(pos)) }
        return
}

type modifierDebugOpts struct {
        cond    Value `if,cond,where,when`
        info  []Value `i,info`
        warn  []Value `w,warn`
        error []Value `e,err;er,error`
        verbose bool `v,verbose`
        checkOutdated bool `d,dirty;cd,checkdirty;cd,check-dirty;co,check-outdated`
        s int `s,stack,sn,stack-number`
        n int `c,count,n,num,cn,call-number`
}
func modifierDebug(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierDebugOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
        if opts.cond != nil && !opts.cond.True(ctx) { return }

        for _, v := range opts.info  { info(ctx, "%s", v.Strval(ctx)).of(v).debug(1) }
        for _, v := range opts.warn  { warn(ctx, "%s", v.Strval(ctx)).of(v).debug(1) }
        for _, v := range opts.error { erro(ctx, "%s", v.Strval(ctx)).of(v).debug(1) }

        var (
                target , _ = ctx.autoGet("@")
                depends, _ = ctx.autoGet("^")
        )
        if opts.checkOutdated && !isNil(target) {
                var (
                        ordered, _ = ctx.autoGet("|")
                        grepped, _ = ctx.autoGet("~")
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
        if len(opts.info) == 0 && len(opts.warn) == 0 && len(opts.error) == 0 {
                var m *diagPoint
                if len(args) == 0 {
                        m = prompt(ctx, "%v: target=%v stems=%v depends=%v\n",
                                ctx.Position(), target, ctx.stems(), depends)
                } else if opts.verbose {
                        m = prompt(ctx, "%v: %v ; target=%v stems=%v depends=%v\n",
                                ctx.Position(), args, target, ctx.stems(), depends)
                } else if len(args) == 1 {
                        m = prompt(ctx, "%v: %v\n", ctx.Position(), args[0])
                } else {
                        m = prompt(ctx, "%v: %v\n", ctx.Position(), args)
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
func modifierSelect(ctx Context, args... Value) (result Value, traves travestates) {
        args = mergeExpand(ctx, expandPlainValue, args...)
        if value, _ := ctx.autoGet("-"); isNil(value) {
                erro(ctx, "no pipe value $-").debug(1)
        } else if g, ok := value.(*Group); ok && len(args) > 0 {
                if i, e := args[0].Integer(ctx); e != nil {
                        erro(ctx, "%v: %v", args[0], e).debug(1)
                } else {
                        result = g.Get(int(i))
                }
        }
        return
}

func modifierEnv(ctx Context, args... Value) (result Value, traves travestates) {
        args = mergeExpand(ctx, expandPlainValue, args...)

        var program = ctx.program()
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
        debug   bool `d,debug`
        verbose bool `v,verbose`
}
func modifierSet(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                program = ctx.program()
                none = MakeNone(ctx.Position())
                opts modifierSetOpts
                defs []Value
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

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
                        name = a.Key.Strval(ctx)
                        if value = a.Value.expand(ctx, expandPlainValue); isNil(value) {
                                value = a.Value
                        }
                        if entry := ctx.entry(); false && name == "@" && entry != nil && entry.String() == "archive" {
                                info(ctx, "%v -> %v", a.Value, value)
                                info(ctx, "%s", ctx).debug(10)
                        }
                case *Flag:
                        name = a.name.Strval(ctx)
                        if value = none; name == "" { name = "-" }
                default:
                        erro(ctx, "%T `%s` is unsupported (try: foo=value)", arg, arg).debug(1)
                        return
                }
                if def = program.scope.FindDef(name); def == nil {
                        erro(ctx, "no such def '%s' (%v, %v)", name, arg, args).debug(16)
                        break ForArgs
                } else {
                        def.val(ctx, value)
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
func modifierSetDirtyPats(ctx Context, args... Value) (result Value, traves travestates) {
        var opts = ctx.dirtyOpts()
        opts.pats = parseOpts(ctx, opts, mergeExpand(ctx, expandPlainValue, args...)...)
        return
}

// create closure context for the traversal
type modifierClosureOpts struct {
        dump    bool `d,dump`
        verbose bool `v,verbose`
}
func modifierClosure(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                opts modifierClosureOpts
                pos = ctx.Position()
        )
        // Closure the caller program, the context will be restored when execution is finished.
        if t := ctx.traversal(); t != nil && false {
                t.Context = closureWith(t.Context, pos)
        } else if pc := ctx.programContext(); pc != nil {
                pc.Context = closureWith(pc.Context, pos)
        } else {
                erro(ctx, "needs closure context: %v", ctx).debug(1)
                return
        }

        assert(ctx.closure() != nil, "context not closured: %v", ctx)

        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
        if opts.verbose { info(ctx, "%v: %v", ctx.Project(), ctx).debug(1) }
        if opts.dump { infostack(ctx, -1, "%v: %v", ctx.Project(), ctx).debug(1) }

        var dir string // closure work directory
        if proj := ctx.Project(); proj == nil {
                errostack(ctx, 6, "%T: nil project in the context", ctx).debug(64)
        } else if scope := proj.scope; scope == nil {
                erro(ctx, "empty closure context").debug(1)
        } else if def := scope.FindDef("/"); def == nil {
                erro(ctx, "&/ is undefined").at(scope.position).debug(1)
        } else if dir = def.value.Strval(ctx); dir == "" {
                erro(ctx, "&/ is empty").at(scope.position).debug(1)
        } else if !filepath.IsAbs(dir) {
                erro(ctx, "&/ is relative").at(scope.position).debug(1)
        } else if err := enter(ctx, dir); err == nil {
                var program = ctx.program()
                program.project.changedWD = dir
                program.changedWD = dir
        }
        return
}

type modifierForOpts struct {
        // ...
}
func modifierFor(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierForOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
        // TODO: ...
        return
}

type modifierCDOpts struct {
        makePath bool `p,path`
        printEnter bool `e,print-enter`
        printLeave bool `l,print-leave`
}
func modifierCD(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierCDOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

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

type modifierMkdirOpts struct {
        mode os.FileMode `m,mode`
        verbose bool `v,verbose`
}
func modifierMkdir(ctx Context, args... Value) (result Value, traves travestates) {
        var opts = modifierMkdirOpts{ mode: os.FileMode(0755) }
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        if len(args) == 0 {
                var target, _ = ctx.autoGet("@")
                var s = target.Strval(ctx)
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

type modifierPathOpts struct {
        // TODO: options required
}
// (path $(dir $@))
// (path /example/path)
func modifierPath(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierPathOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        if len(args) == 0 {
                var target, _ = ctx.autoGet("@")
                var s = target.Strval(ctx)
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

func modifierSudo(ctx Context, args... Value) (result Value, traves travestates) {
        erro(ctx, "TODO: sudo modifier is not implemented yet").at(ctx.Position()).debug(1)
        return
}

func parseDependList(ctx Context, dependList *List) (depends *List, traves travestates) {
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
                                brk := traves.add(ctx, traveFail, nil)
                                brk.error = fmt.Errorf("bad status %v", d.Status)
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
                        if file.info, _ = os.Stat(file.Strval(ctx)); file.info == nil { continue }
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
        } else if fullnameOrStrval(ctx, file) == g.targetFullName {
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
                        if file = stat(ctx, name, "", inc.Strval(ctx)); file != nil {
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
                        if file.info, err = os.Stat(file.Strval(ctx)); err != nil {
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
        } else {
                filename = fullnameOrStrval(ctx, file)
        }
        return
}

func getSavedGrepFileName(ctx Context, targetFullName string) (filename string, err error) {
        var ( file *File )
        if file, err = tempFile(ctx, ".grep", targetFullName); err != nil {
                erro(ctx, "get .grep temp file failed: %v", err).debug(1)
        } else {
                filename = fullnameOrStrval(ctx, file)
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
                targetName = v.Strval(ctx)
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
func modifierGrep(ctx Context, args... Value) (result Value, traves travestates) {
        if false && options.noDepsGrep || options.noGrep {
                return
        }

        var gc grepctx
        gc.fileinc = true // grep files by default
        args = parseOpts(ctx, &gc.modifierGrepOpts, mergeExpand(ctx, expandPlainValue, args...)...)
        gc.incs = mergeExpand(ctx, expandPlainValue, gc.incs...)
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
                if err := grep(ctx, &gc); err != nil {
                        erro(ctx, "grep files from %v failed: %v", target, err).debug(1)
                        return
                } else if gc.noTraverse {
                        // does nothing
                } else if len(t.grepped) > 0 {
                        for _, val := range t.grepped {
                                if traves = val.traverse(ctx); !traves.has() { continue }
                                for _, brk := range traves {
                                        erro(ctx, "%v: %v", val, brk).at(brk.pos)
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
        if !gc.noTraverse {
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

func parseDeps(ctx Context, targetVal Value, targetStr string, savedDepsFile *File, savedDepsFileName, deps string) (files []Value, traves travestates) {
        const parallel = true
        var (
                proj = ctx.Project()
                targetFullName = fullnameOrStrval(ctx, targetVal)
                filesMux sync.Mutex
                firstWord string
                err error
        )

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
                missMux, travesMux sync.Mutex
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
                } else if t := file.traverse(ctx); !t.has() {
                        addFile(file)
                } else if t = t.not(traveCase, traveDone, traveNext); t.has() {
                        prompt(ctx, "%v: missing dep\n", file)
                        if savedDepsFile != nil {
                                var s = filepath.Base(file.name)
                                warn(ctx, `%v: missing "%v"`, targetVal, s).at(depPos)
                                warnstack(ctx, 3, "%v: (%T):", proj, ctx).debug(4)
                        } else {
                                travesMux.Lock()
                                traves = append(traves, t...)
                                travesMux.Unlock()
                                erro(ctx, `%v: missing "%v"`, targetVal, file).at(depPos)
                                for _, brk := range t {
                                        erro(ctx, `%v: broken for "%s": %v`, proj, targetVal, brk).at(brk.pos)
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
                        files, traves = nil, nil // To update savedDepsFileName
                }
        }
        return
}

func loadSavedDepsAndCheckOutdated(ctx Context, args []string) (savedDepsFileName string, files []Value, traves travestates) {
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
        } else if files, traves = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, string(savedDepsBytes)); len(files) > 0 {
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

func traverseMissingDep(ctx Context, dep string) (res bool, traves travestates) {
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
                        traves = traverse(ctx, nil, dep)
                        okay = !traves.has(traveFail)
                } else {
                        prompt(ctx, "%s: dep is unknown file; project %v\n", dep, proj)
                        erro(ctx, "%v: %s is unknown file", proj, dep)
                        errostack(ctx, 5, "(%T):", ctx).debug(24)
                        fail(ctx.Position(), "dep '%s' is not file", dep)
                }
                fullname = dep
        } else {
                traves = file.traverse(ctx)
                okay = !traves.has(traveFail) && file.exists()
                fullname = file.fullname()
        }
        if traves.has(traveCase, traveNext, traveDone) {
                traves = traves.not(traveCase, traveNext, traveDone)
                // TODO: for _, brk := range t { ... }
        }
        if traves.has() {
                prompt(ctx, "%s: traverse dep failed (okay=%v), project %v\n", fullname, okay, proj)
                for _, brk := range traves { erro(ctx, "%v: missing %v: %v", proj, dep, brk.what   ).at(brk.pos) }
                errostack(ctx, 5, "%v: %v", proj, ctx).debug(10)
        } else {
                res = okay
        }
        return
}

func traverseMissingDeps(ctx Context, lastTry string, errBytes []byte) (res bool, tried string, traves travestates) {
        const promptErrors bool = false
        const promptBeforeTraverse bool = promptErrors && true
        for _, rx := range knownerrors {
                var all [][][]byte = rx.FindAllSubmatch(errBytes, -1)
                if all != nil { for _, m := range all {
                        if rx == rxFatalErrorFileNotFound {
                                if promptBeforeTraverse { prompt(ctx, "%s\n", m[0]).debug(6) }
                                if dep := string(m[4]); dep == lastTry {
                                        return false, "", nil
                                } else if res, traves = traverseMissingDep(ctx, dep); !res || traves.has() {
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
func modifierDeps(ctx Context, args... Value) (result Value, traves travestates) {
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
                return
        } else if targetStr == "" {
                erro(ctx, "target '%v' is empty", targetVal).debug(1)
                return
        } else if args = parseOpts(ctx, &opts, args...); len(args) > 0 {
                args = mergeExpand(ctx, expandPlainValue, args...)
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
                flags = mergeExpand(ctx, expandPlainValue, opts.flags...)
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
                var s = strings.TrimSpace(fullnameOrStrval(ctx, a))
                if strings.Contains(s, "-v -fPIC -fvisibility-inlines-hidden") {
                        var v = a.expand(ctx, expandPlainValue)
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
        if savedDepsFileName, files, traves = loadSavedDepsAndCheckOutdated(ctx, ca); traves.has() {
                for _, brk := range traves { erro(ctx, "%v", brk).at(brk.pos) }
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
                        if okay, retried, traves = traverseMissingDeps(ctx, retried, stderr.Bytes()); okay && !traves.has() {
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
                if files, traves = parseDeps(ctx, targetVal, targetStr, savedDepsFile, savedDepsFileName, stdout.String()); len(files) == 0 {
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
func modifierTouch(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierTouchOpts // = modifierTouchOpts{ mode: os.FileMode(0755) }
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
        if len(args) == 0 {
                if target, found := ctx.autoGet("@"); found && !isTrivial(target) {
                        args = append(args, target)
                }
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
        if opts.verbose { reportFileUpdates(ctx, ctx.traversal().start, files) }
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
func modifierCheck(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                pos = ctx.Position()
                opts modifierCheckOpts
                optBreak travekind // breaking with good results
                makeResult func(Position,bool) Value // returns results only if non-nil
                value, _ = ctx.autoGet("-")
                values []Value
                pairs []*Pair
                res bool
        )

        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        if opts.good    { optBreak   = traveDone }
        if opts.answer  { makeResult = MakeAnswer }
        if opts.boolean { makeResult = MakeBoolean }
        if opts.silent && makeResult == nil { makeResult = MakeBoolean }

        for _, arg := range args {
                switch a := arg.(type) {
                case *Pair: pairs = append(pairs, a)
                default:
                        if res = arg.True(ctx); makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else {
                                traves.addf(ctx, optBreak, "value '%v' is false", arg)
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
                } else if s = opts.file.Strval(ctx); filepath.IsAbs(s) {
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
                        traves.addf(ctx, optBreak, "value '%v' is not file", opts.file)
                        return
                }
        }
        if !(isNil(opts.dir) || isNone(opts.dir)) {
                var ( s string; f *File )
                if f, res = opts.dir.(*File); res {
                        if res = f.exists(); !res && opts.verbose {
                                warn(ctx, "file '%v' does not exists", opts.dir).of(opts.dir).debug(1)
                        }
                } else if s = opts.dir.Strval(ctx); filepath.IsAbs(s) {
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
                        traves.addf(ctx, optBreak, "value '%v' is not dir", opts.dir)
                        return
                }
        }

        var program = ctx.program()
ForPairs:
        for _, p := range pairs {
                var key, str string
                switch key = p.Key.Strval(ctx); key {
                case "status":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                traves.addf(ctx, optBreak, "value '%v' is not exec result", value)
                                erro(ctx, "value '%v' (%T) is not exec result", value, value).of(value).debug(6)
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
                                traves.addf(ctx, optBreak, "bad status (%v) (expects %v)", exeres.Status, p.Value)
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                traves.addf(ctx, optBreak, "not an exec result (%T)", value)
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
                                traves.addf(ctx, optBreak, "bad %s (expects %v)", key, p.Value)
                                break ForPairs
                        }

                        str = p.Value.Strval(ctx)

                        if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                traves.addf(ctx, optBreak, "bad %s (%v) (expects %v)", key, v, p.Value)
                                break ForPairs
                        }
                case "file", "dir": // file=xxx and dir=xxx, same as -file=xxx and -dir=xxx
                        var ( file *File; res bool )
                        if file, res = p.Value.(*File); res {
                                // ok
                        } else if str = p.Value.Strval(ctx); filepath.IsAbs(str) {
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
                                traves.addf(ctx, optBreak, "`%v` is not %s", p.Value, key)
                                break ForPairs
                        }
                case "var":
                        var g, ok = p.Value.(*Group)
                        if !ok {
                                traves.addf(ctx, optBreak, "`%v` is not a group value", p.Value)
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
                                                        traves.addf(ctx, optBreak, "`%v` != `%v`", p.Key, p.Value)
                                                        break ForPairs
                                                }
                                        } else if makeResult != nil {
                                                values = append(values, makeResult(pos, false))
                                        } else {
                                                traves.addf(ctx, optBreak, "`%v` is not defined", k)
                                                break ForPairs
                                        }
                                default:
                                        traves.addf(ctx, optBreak, "`%v` unsupported checks", elem)
                                        break ForPairs
                                }
                        }
                default:
                        erro(ctx, "unknown check for %v -> %v", p.Key, p.Value).debug(1)
                        break ForPairs
                }
        }
        if len(values) > 0 {
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
func modifierCopyFile(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierCopyFileOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

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
                filename = target.Strval(ctx)
                if file := project.FindFile(ctx, filename); file != nil {
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
                if file := project.FindFile(ctx, srcname); file != nil {
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

func modifierWriteFile(ctx Context, args... Value) (result Value, traves travestates) {
        args = mergeExpand(ctx, expandPlainValue, args...)

        var (
                target, _ = ctx.autoGet("@")
                filename, str string
                f *os.File
        )
        defer func() {
                if filename != "" { os.Remove(filename); f = nil }
                if f == nil {
                        brk := traves.add(ctx, traveFail, target)
                        brk.error = fmt.Errorf("file %s not generated", target)
                }
        } ()
        if isNil(target) {
                erro(ctx, "target is undefined").debug(1)
                return
        }

        filename = fullnameOrStrval(ctx, target)

        if buffer, _ := ctx.autoGet("-"); isNil(buffer) {
                erro(ctx, "buffer value is nil").debug(1)
                return
        } else {
                str = buffer.Strval(ctx)
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
        debug    bool "d,debug"
        verbose  bool "v,verbose"
        fullname bool "f,full,fullname"
        head Value "h,head"
        foot Value "f,foot"
}
func modifierReadFile(ctx Context, aa... Value) (result Value, traves travestates) {
        var (
                opts modifierReadFileOpts
                args []Value
        )
        if args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, aa...)...); opts.fullname {
                args = mergeExpand(ctx, expandFullName, args...)
        }

        var (
                file *File
                filename string
                target Value
        )
        if n := len(args); n > 1 {
                erro(ctx, "too many files: %v", args).debug(1)
                return
        } else if n == 1 {
                target = args[0]
        } else {
                target, _ = ctx.autoGet("@")
        }

        if isTrivial(target) {
                errostack(ctx, 3, "target for reading is invalid (%T) (%v -> %v)",
                        target, aa, args).debug(10)
                return
        } else if file, filename, _ = fullname(ctx, target); file == nil {
                if depend, found := ctx.autoGet(">"); found && !isTrivial(depend) {
                        s := traves.add(ctx, traveFail, target)
                        s.error = traveTargetNotDefinedFile
                        s.depend = depend
                } else if true {
                        prompt(ctx, "%v: not defined as file\n", target.Strval(ctx))
                        erro(ctx, "(%T) %v", target, target)
                        errostack(ctx, 8, "").debug(64)
                }
                return
        } else if filename == "" {
                errostack(ctx, 3, "target filename is empty").of(target).debug(32)
                return
        }

        var err error
        var bytes []byte
        if bytes, err = ioutil.ReadFile(filename); err == nil {
                var s string
                if opts.head != nil { s = opts.head.Strval(ctx) }
                s += string(bytes)
                if opts.foot != nil { s = opts.foot.Strval(ctx) }
                ctx.autoSet("-", MakeString(ctx.Position(), s))
        } else {
                brk := traves.add(ctx, traveFail, target)
                brk.error = err
        }
        if opts.debug && err != nil {
                warn(ctx, "%v: %v ; stems=%v\n", target, err, ctx.stems())
                warnstack(ctx, 5, "").debug(36)
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
func modifierUpdateFile(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                opts = modifierUpdateFileOpts{ mode: os.FileMode(0640) }
                filename string
                target Value
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
        if opts.full { args = mergeExpand(ctx, expandFullName, args...) }

        if len(args) > 0 {
                target = args[0]
        } else {
                target, _ = ctx.autoGet("@")
        }
        if len(args) > 1 {
                opts.mode = permVal(ctx, args[1], 0600)
        }

        // Get target filename
        if opts.full {
                if filename = fullnameOrStrval(ctx, target); !filepath.IsAbs(filename) {
                        var ( file *File ; s string )
                        var projs = closureProjects(ctx)
                        for _, proj := range projs {
                                if file = proj.FindFile(ctx, filename); file != nil {
                                        s = file.fullname()
                                        break
                                }
                        }
                        if filepath.IsAbs(s) {
                                filename = s // good!
                        } else if file != nil {
                                prompt(ctx, "%v: %T %v; projects=%v\n",
                                        file.fullname(), target, target, projs)
                                errostack(ctx, 5, "fullname is incorrect").debug(16)
                        } else {
                                prompt(ctx, "%v: %T; file=%v projects=%v\n",
                                        target, target, file, projs)
                                warnstack(ctx, 5, "").debug(16)
                        }
                }
        } else {
                switch p := target.(type) {
                case *File: filename = p.fullname()
                case *Path: filename = p.Strval(ctx)
                default:    filename = target.Strval(ctx)
                        if file := ctx.Project().FindFile(ctx, filename); file != nil {
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
                                        erro(ctx, "%v", e).debug(1)
                                }
                        }
                        if err := os.MkdirAll(p, os.FileMode(0755)); err != nil {
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
        } else if content = value.Strval(ctx); false && strings.Contains(content, `"\"`) {
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
                        if err := os.Remove(filename); err != nil {
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
                brk := traves.add(ctx, traveFail, target)
                brk.error = fmt.Errorf("update %v failed", target)
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
                brk := traves.add(ctx, traveFail, target)
                brk.error = fmt.Errorf("%v not updated", target)
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
func modifierWait(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                opts modifierWaitOpts
                execRes *ExecResult
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        var (
                waitForExecResult = opts.stdout || opts.stderr || opts.status || opts.execRes
                stampCurrentTarget = !opts.noTarget
                target, _ = ctx.autoGet("@")
                err error
        )
        if opts.verbose {
                defer func (st time.Time) {
                        var s string; if err != nil { s = "fail" } else { s = "done" }
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
        next bool "n,nxt,next" // traveNext if failed to stamp
        error bool "e,err;e,error" // traveErro if failed to stamp
        prompt bool "m,prompt"
        verbose bool "v,verbose"
        debug int "d,debug"
}
func modifierStamp(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                opts modifierStampOpts
                pos = ctx.Position()
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        var target = getTargetValue(ctx)
        if isNil(target) {
                prompt(ctx, "%v\n", ctx.Project())
                erro(ctx, "stamp(%v) failed", target)
                errostack(ctx, 6, "%v", ctx).debug(12)
                return
        }

        var _, err = target.stamp(ctx)
        if err == nil { return /* Done! */ }

        prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err)
        if opts.next {
                if opts.verbose { warn(ctx, "%v", err).debug(1) }
                s := traves.add(ctx, traveNext, target)
                s.depend, _ = ctx.autoGet(">")
                err = nil // discard the error
        } else if opts.error {
                s := traves.add(ctx, traveFail, target)
                s.depend, _ = ctx.autoGet(">")
                s.error = err
                if false {
                        prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err)
                        erro(ctx, "stamp(%v) error")
                        errostack(ctx, -1, "%v", ctx).debug(1)
                } else {
                        prompt(ctx, "%v: %v: %v\n", target, ctx.Project(), err)
                        warn(ctx, "stamp(%v) error")
                        warnstack(ctx, -1, "%v", ctx).debug(1)
                }
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
        message  string "m,message;m,msg"
        verbose  bool "v,verbose"
        verbose0 bool
}
func predict(ctx Context, args... Value) (result bool, message string, err error) {
        var (
                target, _ = ctx.autoGet("@")
                targetStr string
                num int64
        )
        if isTrivial(target) {
                errostack(ctx, 5, "target is trivial, %v", ctx).debug(10)
                return
        }
        for caller := ctx.traversal().caller(); caller != nil; caller = caller.caller() {
                if tarVal, _ := caller.autoGet("@"); isNil(tarVal) {
                        // top level execution, aka via RuleEntry.Execute(...)
                } else if true {
                        var same = target == tarVal
                        if !same && false {
                                same = (target.cmp(ctx, tarVal) == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n := caller.execRec[target]; n > 0 {
                        num += int64(n)
                }
        }

        var (
                opts predictOpts
                reasons = make(map[string]int)
        )
        defer func() { if opts.verbose {
                var status string
                for reason, n := range reasons {
                        if status != "" { status += ", " }
                        if n == 1 { status += reason } else {
                                status += fmt.Sprintf("%s (%d)", reason, n)
                        }
                }
                if status == "" {
                        var s string
                        if result { s = "Yes" } else { s = "No" }
                        status = fmt.Sprintf("%v (%d)", s, num)
                } else if false {
                        status += fmt.Sprintf(" (result=%v)", result)
                }
                prompt(ctx, "… %s\n", status)
        }} ()

ForArgs:
        for _, arg := range args {
                switch tv := arg.(type) {
                case *String, *Compound:
                        message = tv.Strval(ctx)
                        continue ForArgs
                }

                var va = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, arg)...)
                if len(va) == 0 {
                        continue ForArgs
                } else if len(va) == 1 && isTrivial(va[0]) {
                        continue ForArgs
                }

                if opts.verbose && !opts.verbose0 {
                        targetStr = fullnameOrStrval(ctx, target)
                        prompt(ctx, "checking %v …", filepath.Base(targetStr))
                        opts.verbose0 = true
                }

                if !opts.and && result { break }
                if !opts.and || (opts.and && result) { for i, a := range va {
                        var tru bool
                        if g, ok := a.(*Group); !ok {
                                // preserve the value of 'a'
                        } else if len(g.Elems) == 0 {
                                erro(ctx, "predictor is empty group").at(g.position).debug(1)
                                return
                        } else if pret, ok := predictors[g.Elems[0].Strval(ctx)]; !ok {
                                erro(ctx, "predictor '%s' undefined (%T %v)", g.Elems[0], a, a).at(g.position).debug(1)
                                return
                        } else {
                                a = pret(positional(ctx, g.Elems[0].Position()), g.Elems[1:]...)
                        }

                        if a == nil {
                                warn(ctx, "predictor %v is <nil>", arg).debug(1)
                                continue // skip
                        } else if p, ok := a.(*prediction); ok {
                                if p.reason != "" { reasons[p.reason] += 1 }
                                tru = p.bool
                        } else if tru = a.True(ctx); tru {
                                reasons[fmt.Sprintf("#%v", i+1)] += 1
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
func modifierAssert(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                res bool
                msg string
                err error
        )
        if res, msg, err = predict(ctx, args...); err != nil {
                erro(ctx, "prediction %v failed: %v", args, err).debug(6)
        } else if !res {
                var target, _ = ctx.autoGet("@")
                if msg == "" {
                        for _, a := range args {
                                erro(ctx, "assertion failed: %v", a).of(a)
                        }
                        errostack(ctx, 8, "(%T):", ctx).debug(6)
                } else {
                        var vals = mergeExpand(ctx, expandPlainValue, args...)
                        erro(ctx, "assertion failed: %v (target = %s)", msg, target)
                        erro(ctx, "assertion args: %v", args)
                        erro(ctx, "assertion args: %v (mergeExpandd)", vals)
                        erro(ctx, "assertion context: %v", ctx).debug(6)
                }
                s := traves.add(ctx, traveFail, target)
                s.error = fmt.Errorf("assertion failure: %v", args)
        }
        return
}

func modifierCond(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                res bool
                msg string
                err error
        )
        if res, msg, err = predict(ctx, args...); err != nil {
                erro(ctx, "predict: %v", err).debug(1)
        } else if !res {
                s := traves.add(ctx, traveDone, nil)
                if msg != "" { s.error = fmt.Errorf("%s", msg) }
                s.prog = ctx.program()
        }
        // var target, _ = ctx.autoGet("@")
        // if strings.Contains(target.Strval(ctx), "Unwind") {
        //         var v, _ = ctx.autoGet("^")
        //         var d = target.updatedDeps(ctx)
        //         prompt(ctx, "cond:%v: %v: %v; %v, %v; %v\n",
        //                 args, target, v, res, traves, d)
        // }
        return
}

type modifierCaseOpts struct {
        debug   bool `d,debug`
        verbose bool `v,verbose`
}
func modifierCase(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierCaseOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        if res, msg, err := predict(ctx, args...); err != nil {
                erro(ctx, "case: (res=%v) %v", res, err).debug(1)
        } else {
                var w travekind
                if res { w = traveCase } else { w = traveNext }

                var s = traves.add(ctx, w, nil) // trave 'case' or 'next'
                if msg != "" { s.error = fmt.Errorf("%s", msg) }
                s.prog = ctx.program()

                if opts.verbose {
                        var a, _ = ctx.autoGet("@")
                        prompt(ctx, "%v: %v (msg=%s)", a, w, msg)
                }
                if opts.debug {
                        warn(ctx, "%v", w)
                }
        }
        return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
        var opts = ctx.dirtyOpts()
        if len(target.updatedDeps(ctx)) > 0 { return true }
        if v, found := ctx.autoGet("^"); found && !isTrivial(v) { a = append(a, v) }
        for _, dep := range mergeExpand(ctx, expandPlainValue, a...) {
                var mat bool = len(opts.pats) == 0
                if !mat { for _, pat := range opts.pats { if mat, _, _ = pat.match(ctx, dep); mat { break }}}
                if mat && (dep.updated(ctx) || dep.stat(ctx).mod().After(target.stat(ctx).mod())) {
                        return true
                }
        }
        return
}

type predictionOutdatedOpts struct {
        generalOpts
        checksum bool "c,cs,checksum,crc"
        verboseUpdated  bool "vu,verbose-updated"
        verboseOutdated bool "vo,verbose-outdated"
        silent   bool "s,silent"
}
func predictionOutdated(ctx Context, args... Value) (result Value) {
        var (
                program = ctx.program()
                opts predictionOutdatedOpts
                target Value
                targetFullname string
                reason string
                outdated bool
                err error
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        if target, _, _, err = wait(ctx); err != nil {
                erro(ctx, "waiting traversal failed: %v", err).debug(1)
                return
        }

        if outdated = len(target.updatedDeps(ctx)) > 0; outdated {
                reason = "updated prerequisites"
        }

        targetFullname = fullnameOrStrval(ctx, target)
        if outdated = !exists(ctx, target); outdated {
                reason = "target not exists"
        } else if outdated = isDirty(ctx, target, args...); outdated {
                reason = "dirty"
        } else if outdated, err = isRecipesChanged(ctx, target); err != nil {
                erro(ctx, "isRecipesChanged: %v", err).debug(1)
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
                        } else if file2 = fullnameOrStrval(ctx, depend); file2 != "" {
                                // see: same, err = crc64CompareFileChecksum(ctx, targetFullname, file2)
                                // TODO.1: load saved checksum for depend, set outdated if no such
                                // TODO.2: calculate checksum for depend and compare with the loaded
                                // TODO.3: set outdated if the two checksums differred
                        }
                }
        }

        if opts.debug>0 || opts.verbose || (opts.verboseOutdated && outdated) || (opts.verboseUpdated && !outdated) {
                var ( t = ctx.traversal(); m, s string )
                if outdated { m = "outdated" } else { m = "updated" }
                if s = time.Now().Sub(t.start).String(); reason != "" {
                        s += "; " + strings.TrimSpace(strings.TrimPrefix(reason, "outdated:"))
                }
                var (
                        ts = trimPromptString(targetFullname)
                        n = len(t.targets) + len(t.grepped)
                )
                if db := opts.debug>0; db && !opts.verbose {
                        warn(ctx, "%s (%T) (%s) …… %s (%d files in %s, debug=%d)",
                                ts, target, targetFullname, m, n, s, opts.debug).debug(opts.debug * 2)
                } else {
                        prompt(ctx, "%s …… %s (%d files in %s)\n",
                                ts, m, n, s).debug(db, 6)
                }
        }

        if opts.silent { reason = "" }
        if result = MakePrediction(ctx.Position(), outdated, reason); outdated {
                if program.dirt != "" { reason = program.dirt + "; " + reason }
                program.dirt = reason
        }
        return
}

func predictionNoLoop(ctx Context, args... Value) (result Value) {
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
func predictionTarget1stVisit(ctx Context, args... Value) (result Value) {
        var opts predictionTarget1stVisitOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

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
func predictionTargetMaxVisit(ctx Context, args... Value) (result Value) {
        var opts predictionTargetMaxVisitOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        var nth int64
        for _, a := range args {
                if i, e := a.Integer(ctx); e != nil {
                        erro(ctx, "%v: %v", a, i).debug(1)
                } else if nth = i; nth <= 0 {
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
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
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
func modifierFork(ctx Context, args... Value) (result Value, traves travestates) {
        var (
                prog = ctx.program()
                opts modifierForkOpts
                argv []string
                wd string
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)
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
        debug bool "d,debug"
        verbose bool "v,verbose"
}
func modifierGitModified(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierGitModifiedOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

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
        debug bool "d,debug"
        verbose bool "v,verbose"
}
func modifierGitAhead(ctx Context, args... Value) (result Value, traves travestates) {
        var opts modifierGitAheadOpts
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

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
                // FIXME: not unique combination
                fmt.Fprintf(h, "%p", entry)
        } else {
                // FIXME: not unique combination
                fmt.Fprintf(h, "%p%p", entry, program)
        }

        for _, t := range merge(target) {
                if f, ok := t.(*File); ok {
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
        onceSHA256Cache[sum] += 1
        onceSHA256Mutex.Unlock()
        return onceSHA256Cache[sum]
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
                // NOTE: entry and program are unique, since (once) is for runtime, we use their addresses.
                fmt.Fprintf(h, "%p%p", entry, program)
        } else {
                fmt.Fprintf(h, "%v%v", ctx.Position(), program.position)
        }

        for _, a := range args {
                fmt.Fprintf(h, "%s", fullnameOrStrval(ctx, a))
        }

        var sum HashBytes
        copy(sum[:], h.Sum(nil))
        return onceSHA256Test(ctx, sum)
}

type modifierOnceOpts struct {
        debug    bool `d,debug`
        verbose  bool `v,verbose`
        checksum bool `c,cs,checksum,s,sha,sha256,sum,h,hash`
        forval Value `for` // TODO: (once -for=$@)
}
func modifierOnce(ctx Context, args... Value) (result Value, traves travestates) {
        // TODO: (once)           --> once for the RuleEntry, aka entry.doneOnce = true
        // TODO: (once -for=$@)   --> once for $@, aka entry.onces[$(expand $@)] = true
        var (
                n int
                opts modifierOnceOpts
                target, found = ctx.autoGet("@")
        )
        args = parseOpts(ctx, &opts, mergeExpand(ctx, expandPlainValue, args...)...)

        const onceAlgo = 2 // avaialbe: 0, 1, 2
        if !found || isTrivial(target) {
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

        if n > 1 {
                s := traves.add(ctx, traveDone, target)
                s.error = fmt.Errorf(`executed %d times`, n)
                s.prog = ctx.program()
        }

        // if strings.Contains(target.Strval(ctx), "Unwind") {
        //         var v, _ = ctx.autoGet("^")
        //         prompt(ctx, "once: %v: %v, %v, %v\n", target, traves, v, n)
        // }

        if opts.debug {
                warn(ctx, "%T %v %p %v", target, target, target, n)
                warnstack(positional(ctx, target.Position()), -1, "%p %v %v", target, target, n).debug(16)
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
                                s := traves.add(ctx, traveDone, target)
                                s.error = fmt.Errorf(`executed %d times`, n)
                                s.prog = ctx.program()
                        }
                }
        }
        return
}
