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
        "runtime/debug"
        "strings"
        "sync"
        "time"
)

var launchTime = time.Now()

const (
        TheShellEnvarsDef = "shell→envars" // '→' ' → '
        TheShellStatusDef = "shell→status" // status code of execution
)

type breakind int
type breaksco int

func (k breakind) String() (s string) {
        switch k {
        case breakDone: s = "break.done"
        case breakNext: s = "break.next"
        case breakCase: s = "break.case"
        case breakFail: s = "break.fail"
        case breakErro: s = "break.error"
        }
        return
}

const (
        breakUnkn breakind = iota
        breakDone // (cond ...) and (case ...)
        breakNext // (cond ...) and (case ...)
        breakCase // (case ...)
        breakFail // (assert ...)
        breakErro // break with an error
)

const (
        breakGroup breaksco = iota
        breakTrave
)

type modification struct {
        target Value
        result Value
}

func (m *modification) String() string {
        return m.target.String()
}

type breaker struct {
        pos Position
        what breakind
        scope breaksco
        message string
        misstar *updatedtarget
        updated []*updatedtarget
        value Value
        error error
}

func (p *breaker) _error() (s string) {
        switch p.what {
        case breakUnkn: s = "unknown"
        case breakDone: s = "done" // ineligible (cond) is ignored
        case breakNext: s = "next"
        case breakCase: s = "case"
        case breakFail: s = "failure" // "break with assertion failure"
        case breakErro: s = fmt.Sprintf("break error: %v", p.error) // "break with an error"
        }
        if p.pos.IsValid() {
                if p.message != "" { s += ": " + p.message }
                if false { s = fmt.Sprintf("%s: %s", p.pos, s) }
        }
        return
}

func (p *breaker) prerequisites() (res []*updatedtarget) {
        for _, u := range p.updated {
                res = append(res, u.prerequisites...)
        }
        return
}

type modifier struct {
        valbase
        name Value
        args []Value
}
func (m *modifier) refs(v Value) bool {
        if m.name.refs(v) { return true }
        for _, a := range m.args {
                if a.refs(v) { return true }
        }
        return false
}
func (m *modifier) expandible(w expandwhat) (res bool) {
        if res = m.name.expandible(w); !res {
                for _, a := range m.args {
                        if res = a.expandible(w); res { break }
                }
        }
        return
}
func (m *modifier) expand(_ expandwhat) (Value, error) { return m, nil }
func (_ *modifier) cmp(v Value) (res cmpres) { 
        if _, ok := v.(*modifier); ok { res = cmpEqual }
        return
}
func (m *modifier) traverse(t *traversal) {
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("modifier.traverse(%s)", m))) }
        if optionTraceTraversal   { defer un(tt(t_traverse, t, m)) }
        if err := t.program.modify(t, m); err != nil {
                if pe, ok := err.(*fs.PathError); ok {
                        diag.errorAt(m.position, "%s failed: %v not found", m.name, trimPromptString(pe.Path)).
                                debug(optionDebugErrors,1)
                } else {
                        diag.errorAt(m.position, "%s failed: %v", m.name, err).
                                debug(optionDebugErrors,1)
                }
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
func (g *modifiergroup) refs(v Value) bool {
        for _, m := range g.modifiers {
                if m.refs(v) { return true }
        }
        return false
}
func (g *modifiergroup) expandible(w expandwhat) (res bool) {
        for _, m := range g.modifiers {
                if res = m.expandible(w); res { break }
        }
        return
}
func (g *modifiergroup) expand(_ expandwhat) (Value, error) { return g, nil }
func (_ *modifiergroup) cmp(v Value) (res cmpres) { 
        if _, ok := v.(*modifiergroup); ok { res = cmpEqual }
        return
}
func (g *modifiergroup) traverse(t *traversal) {
        if optionTraceTraversal { defer un(tt(t_traverse, t, g)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("modifiergroup.traverse(%s)", g))) }
        for _, m := range g.modifiers {
                if m.traverse(t); t.hasBreakers() {
                        /*var breakers = t.breakers; t.breakers = nil
                        for _, brk := range breakers { // otherwise, we check break reasons
                                t.breakers = append(t.breakers, brk) // for (dirty) and interpreters.
                                if brk.what == breakDone && brk.scope == breakGroup {
                                        done = true
                                }
                        }*/
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

type ModifierFunc func(pos Position, t *traversal, args... Value) (Value, error)

var (
        init_modifiers = map[string]ModifierFunc{
                `print`:        modifierPrint,
                `select`:       modifierSelect,

                //`args`:       modifierSetArgs, // interpreter args
                `env`:          modifierEnv,  // interpreter environments
                `set`:          modifierSet,

                `closure`:      modifierClosure,

                `cd`:           modifierCD,
                `mkdir`:        modifierMkdir,
                `path`:         modifierPath,

                `sudo`:         modifierSudo,

                `touch`:        modifierTouch,
                `grep`:         modifierGrep,
                `grep-files`:   modifierGrepFiles,

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

                `dirty`:        modifierDirty,
                `no-loop`:      modifierNoLoop,
                `once`:         modifierOnce,
                `target-1st-visit`: modifierTarget1stVisit,
                `target-max-visit`: modifierTargetMaxVisit,

                `git-ahead`:    modifierGitAhead,
                `git-modified`: modifierGitModified,
        }

        modifiers = make(map[string]ModifierFunc)
        crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)

func init() {
        // Install recursive modifiers here to avoid Go's loop detection.
        for s, m := range init_modifiers { modifiers[s] = m }
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

func promptShellResult(value Value, n int) (err error) {
        if g, ok := value.(*Group); ok && g != nil {
                if elem := g.Get(0); elem != nil {
                        var str string
                        if str, err = elem.Strval(); err == nil && str == "shell" {
                                if elem = g.Get(n); elem != nil {
                                        if str, err = elem.Strval(); err != nil {
                                                diag.errorOf(elem, "stringify '%v' failed: %v", elem, err)
                                                return
                                        } else if strings.HasSuffix(str, "\n") {
                                                fmt.Fprintf(stderr, "%s", str)
                                        } else if str != "" {
                                                fmt.Fprintf(stderr, "%s\n", str)
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
        reset bool `r,reset`
}

func modifierPrint(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                opts = modifierPrintOpts{ stderr: true }
                content string
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }
        if content, err = t.def.buffer.value.Strval() ; err != nil { diag.errorAt(pos, "stringify buffer value failed: %v", err); return }
        if opts.stdout { fmt.Fprint(stdout, content) }
        if opts.stderr { fmt.Fprint(stderr, content) }
        if opts.reset { t.def.buffer.value = MakeNone(pos) }
        return
}

// select element by index from group result: (select 0)
func modifierSelect(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var value Value = t.def.buffer.value
        if g, ok := value.(*Group); ok && len(args) > 0 {
                var num int64
                if num, err = args[0].Integer(); err == nil {
                        result = g.Get(int(num))
                }
        }
        return
}

func modifierSetArgs(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        // TODO: preserve args for interpreter
        return
}

func modifierEnv(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var def *Def
        var envars = new(List)
        if def, err = t.program.auto(TheShellEnvarsDef, envars); err != nil { return }
        for _, a := range args {
                if _, ok := a.(*Pair); ok { def.append(a) } else {
                        err = errors.New(fmt.Sprintf("invalid env '%v' (%s)", a, typeof(a)))
                        return
                }
        }
        result = envars
        return
}

// examples:
//     [(set name=value)]    set $(name) to 'value'
//     [(set name)]          clear $(name)
//     [(set -)]             clear $-
func modifierSet(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err).
                        debug(optionDebugErrors,1)
                return
        }
        var defs []Value
        var none = MakeNone(pos)
ForArgs:
        for _, arg := range args {
                var name string
                var value Value = none
                switch a := arg.(type) {
                case *Bareword: name = a.string
                case *Pair:
                        if name, err = a.Key.Strval(); err == nil {
                                // Note that Pair.Value is not expanded yet!
                                // We need to expand the value explicitly.
                                if value, err = a.Value.expand(expandAll); err != nil {
                                        diag.errorAt(pos, "expand value '%v' failed: %v", a.Value, err).
                                                debug(optionDebugErrors,1)
                                        return
                                } else if isNil(value) { value = a.Value }
                        }
                case *Flag:
                        if name, err = a.name.Strval(); err != nil {
                                diag.errorAt(pos, "strval '%v' failed: %v", a.name, err).
                                        debug(optionDebugErrors, 1)
                                return
                        } else if value = none; name == "" { name = "-" }
                default:
                        diag.errorAt(pos, "%T `%s` is unsupported (try: foo=value)", arg, arg).
                                debug(optionDebugErrors, 1)
                        return
                }
                if def := t.program.scope.FindDef(name); def == nil {
                        diag.errorAt(pos, "`%s` no such def", name).
                                debug(optionDebugErrors, 1)
                        break ForArgs
                } else if err = def.set(DefDefault, value); err != nil {
                        diag.errorAt(pos, "`%s` no such def", name).
                                debug(optionDebugErrors, 1)
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
func modifierClosure(pos Position, t *traversal, args... Value) (result Value, err error) {
        // Set caller context before parsing arguments (pop the top one).
        // The context will be restored when execution is finished.
        if c := t.caller; c != nil { t.project, t.closure = c.project, c.closure }

        if false {
                if len(cloctx) > 0 { cloctx = cloctx[1:] }
        } else if len(cloctx) > 1 && cloctx[0] == t.program.scope {
                setclosure(append(cloctx[1:], cloctx[0]))
        } else if len(cloctx) == 0 || cloctx[0] != t.closure {
                setclosure(cloctx.unshift(t.closure))
        }

        var opts modifierClosureOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err); return
        } else if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse closure opts failed: %v", err); return
        }

        if opts.dump {
                fmt.Fprintf(stderr, "%s: closure:\n", pos)
                for _, cc := range cloctx {
                        fmt.Fprintf(stderr, "    %s: %s\n", cc.position, cc.comment)
                }
        } else if opts.verbose {
                fmt.Fprintf(stderr, "%s: %v\n", pos, cloctx)
        }

        var dir string // closure work directory
        if len(cloctx) == 0 {
                diag.errorAt(pos, "empty closure context")
        } else if def := cloctx[0].FindDef("/"); def == nil {
                diag.errorAt(cloctx[0].position, "&/ is undefined")
        } else if dir, err = def.value.Strval(); err != nil {
                diag.errorOf(def.value, "%v", err)
        } else if dir == "" {
                diag.errorAt(cloctx[0].position, "&/ is empty")
        } else if !filepath.IsAbs(dir) {
                diag.errorAt(cloctx[0].position, "&/ is relative")
        } else if err = enter(t.program, dir); err == nil {
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
func modifierCD(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierCDOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge args failed: %v", err)
                return
        }
        if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse cd opts failed: %v", err)
                return
        }

        if opts.printEnter { printEnteringDirectory() }
        if opts.printLeave { printLeavingDirectory() }
        if (opts.printEnter || opts.printLeave) && len(args) == 0 { return }
        if len(args) == 1 {
                var dir string
                if dir, err = args[0].Strval(); err != nil {
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
                if err = enter(t.program, dir); err == nil {
                        t.program.project.changedWD = dir
                        t.program.changedWD = dir
                }
        } else {
                diag.errorAt(pos, "wrong number of cd args: %v", args)
        }
        return
}

type modifierMkdirOpts struct {
        mode os.FileMode `m,mode`
        verbose bool `v,verbose`
}
func modifierMkdir(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts = modifierMkdirOpts{ mode: os.FileMode(0755) }
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge mkdir args failed: %v", err)
                return
        }
        if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse mkdir opts failed: %v", err)
                return
        }
        if len(args) == 0 {
                var s string
                if s, err = t.def.target.value.Strval(); err != nil {
                        diag.errorAt(pos, "stringify target '%v' failed: %v", t.def.target.value, err)
                } else if err = os.MkdirAll(filepath.Dir(s), opts.mode); err != nil {
                        diag.errorAt(pos, "make path '%s' failed: %v", s, err)
                }
                return
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil {
                        diag.errorAt(pos, "stringify '%v' failed: %v", a, err)
                        break
                }
                if err = os.MkdirAll(s, opts.mode); err != nil {
                        diag.errorAt(pos, "make path '%s' failed: %v", s, err)
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
func modifierPath(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge path args failed: %v", err)
                return
        }

        var opts modifierPathOpts
        if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse path opts failed: %v", err)
                return
        }

        if len(args) == 0 {
                var s string
                if s, err = t.def.target.value.Strval(); err != nil {
                        diag.errorAt(pos, "stringify target value '%v' failed: %v", t.def.target.value, err)
                } else if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                                diag.errorAt(pos, "make path '%s' failed: %v", err)
                        }
                }
                return
        }

        for _, arg := range args {
                var s string
                if s, err = arg.Strval(); err != nil {
                        diag.errorOf(arg, "stringify arg '%v' failed: %v", arg, err)
                        break
                }
                if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                        diag.errorAt(pos, "make path '%s' failed: %v", s, err)
                        break
                }
        }
        return
}

func modifierSudo(pos Position, t *traversal, args... Value) (result Value, err error) {
        diag.errorAt(pos, "TODO: sudo modifier is not implemented yet")
        return
}

func parseDependList(pos Position, t *traversal, dependList *List) (depends *List, err error) {
        depends = new(List)
        for _, depend := range dependList.Elems {
                switch d := depend.(type) {
                case *List:
                        if dl, e := parseDependList(pos, t, d); e != nil {
                                err = e; return
                        } else {
                                depends.Elems = append(depends.Elems, dl.Elems...)
                        }
                case *ExecResult:
                        if d.Status != 0 {
                                t._break(pos, breakFail).message = fmt.Sprintf("bad status %v", d.Status)
                                return // target shall be updated
                        } else {
                                depends.Append(d)
                        }
                case *RuleEntry:
                        switch d.Class() {
                        case GeneralRuleEntry, PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                                depends.Append(d)
                        default:
                                diag.errorAt(pos, "unsupported entry depend `%v' (%v)", d, d.Class())
                        }
                case *String:
                        depends.Append(d)
                case *File:
                        depends.Append(d)
                default:
                        diag.errorAt(pos, "unsupported entry depend `%v' (%v)", depend, t.program.depends)
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
                        `^\s*#\s*include\s*"(.*)"`,
                },
                []string{
                        `^\s*#\s*include\s*<(.*)>`,
                },
        },
        "c": &langInfoT{
                []string{
                        `^\s*#\s*include\s*"(.*)"`,
                },
                []string{
                        `^\s*#\s*include\s*<(.*)>`,
                },
        },
        "i": &langInfoT{
                []string{
                        `^\s*include\s*"(.*)"`,
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
        modifierGrepFilesOpts
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
func (g *greptouch) work(pos Position, gc *grepctx) (err error) {
        if g.targetInfo == nil {
                diag.errorAt(g.target.Position(), "'%v' not exists", g.target)
                if false { debug.PrintStack() }
                return
        }
        var tt time.Time = g.targetInfo.ModTime()
        for _, val := range g.files {
                var file, ok = val.(*File)
                if !ok { 
                        fmt.Fprintf(stderr, "%s: '%v' is not file (%T)\n", pos, file, file)
                        return
                }
                if file.info == nil && !file.isSysFile() {
                        var s string
                        if s, err = file.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if file.info, _ = os.Stat(s); file.info == nil { continue }
                        if gc.debug { fmt.Fprintf(stderr, "%s: '%v' info is nil (%s)\n", pos, file, file.fullname()) }
                }
                if file.info == nil {/* ... */} else
                if t := file.info.ModTime(); t.After(tt) {
                        if gc.debug { fmt.Fprintf(stderr, "%s: touch %v → %v (%v)\n", pos, g.target, file, t) }
                        if tt != t { tt = t }
                }
        }
        if tt.After(g.targetInfo.ModTime()) {
                err = os.Chtimes(g.targetFullName, tt, tt)
        }
        return
}
func (g *grepctx) isTargetFile(file *File) (res bool) {
        if g.target == file {
                res = true
        } else if s, _ := fullnameOrStrval(file); s == g.targetFullName {
                res = true
        } else if t, ok := g.target.(*File); ok && t.name == file.name {
                res = true
        }
        return
}

var grepcache = make(map[string][]Value)

func loadGrepCache() {
        s := joinTmpPath("", "cache")
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
                                file := stat(Position{}, a[0], a[1], a[2])
                                if file != nil {
                                        list = append(list, file)
                                }
                        }
                }
        }
}

func saveGrepCache() {
        s := joinTmpPath("", "cache")
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

func (t *traversal) searchGreppedName0(pos Position, gc *grepctx, sys bool, linum, colnum int, name string) (file *File) {
        var isAbs, isRel bool
        if isAbs = filepath.IsAbs(name); isAbs {
                file = stat(pos, name, "", "", nil)
        } else if isRel = isRelPath(name); isRel { // relative to target dir
                file = stat(pos, name, "", gc.targetDir, nil)
                if !file.exists() {
                        var f = t.project.FindFile(name)
                        if f != nil { file = f }
                }
        } else if file = t.project.FindFile(name); file == nil {
                return // file not found
                /*
        } else if !sys && file.filemap != nil && len(file.filemap.Paths) == 1 {
                // mark system files defined by `files ((foo.xxx) => -)`
                if f, ok := file.filemap.Paths[0].(*Flag); ok {
                        sys = isNone(f.name) || isNil(f.name)
                }*/
        } else if !sys && file.isSysFile() { sys = true }

        //fmt.Fprintf(stderr, "%v: %v %v %v %v\n", pos, t.entry.target, name, sys, t.project)

        // System files are not treated as missing nor collected
        // for further updating, just discard them immediately.
        if sys || isAbs || isRel || file.exists() { return }

        // relative to target directory
        var alt = stat(pos, name, "", gc.targetDir)
        if alt != nil { file = alt; return }

        // Check for bare non-system sub-paths:
        //   foo/bar/name.xxx
        // We search base name 'name.xxx' again:
        var s = filepath.Dir(name) // e.g: foo/bar

        // Search 'name.xxx' and check dir for
        // 'foo/bar' suffix. We use it if found.
        alt = t.project.FindFile(filepath.Base(name))
        if alt != nil && strings.HasSuffix(alt.dir, PathSep+s) {
                dir := strings.TrimSuffix(alt.dir, PathSep+s)
                ok1 := alt.change(dir, s, alt.name) // <dir>, foo/bar, name.xxx
                ok2 := alt.change(dir, "", name) // <dir>, "", foo/bar/name.xxx
                file = alt
                if enable_assertions {
                        assert(ok1, "unchanged: %s %s %s", dir, s, alt.name)
                        assert(ok2, "unchanged: %s %s", dir, alt.name)
                }
        }
        return
}

func (t *traversal) searchGreppedName(pos Position, gc *grepctx, sys bool, linum, colnum int, name string) (file *File) {
        var isAbs, isRel bool
        if file = t.project.FindFile(name); file != nil && file.exists() {
                return // found existed file
        } else if isAbs = filepath.IsAbs(name); isAbs {
                file = stat(pos, name, "", "", nil)
        } else if isRel = isRelPath(name); isRel { // relative to targetDir
                file = stat(pos, name, "", gc.targetDir, nil)
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
                        t.entry.target, gc.target, name, file.exists(), sys, t.project)
        }
        if sys || file.exists() { return }

        // relative to target directory
        var alt = stat(pos, name, "", gc.targetDir)
        if alt != nil { file = alt; return }

        // Check for bare non-system sub-paths:
        //   foo/bar/name.xxx
        // We search base name 'name.xxx' again:
        var s = filepath.Dir(name) // e.g: foo/bar

        // Search 'name.xxx' and check dir for
        // 'foo/bar' suffix. We use it if found.
        alt = t.project.FindFile(filepath.Base(name))
        if alt != nil && strings.HasSuffix(alt.dir, PathSep+s) {
                dir := strings.TrimSuffix(alt.dir, PathSep+s)
                ok1 := alt.change(dir, s, alt.name) // <dir>, foo/bar, name.xxx
                ok2 := alt.change(dir, "", name) // <dir>, "", foo/bar/name.xxx
                file = alt
                if enable_assertions {
                        assert(ok1, "unchanged: %s %s %s", dir, s, alt.name)
                        assert(ok2, "unchanged: %s %s", dir, alt.name)
                }
        }
        return
}

func (t *traversal) searchGrepped(pos Position, gc *grepctx, sys bool, linum, colnum int, name string) (file *File, err error) {
        file = t.searchGreppedName(pos, gc, sys, linum, colnum, name)
        if file == nil {
                // The 'name' is not matching the files database.
                if gc.discard { return }
                // FIXME: missing-file error
        } else if gc.isTargetFile(file) {
                return
        } else if !file.exists() && gc.discard {
                return
        } else if gc.files = append(gc.files, file); false && gc.touch {
                var tt = gc.targetInfo.ModTime()
                if file.info == nil && !file.isSysFile() {
                        var s string
                        if s, err = file.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        if file.info, err = os.Stat(s); err != nil { diag.errorAt(pos, "%v", err); return }
                        if false || gc.debug { fmt.Fprintf(stderr, "%s: '%v' info is nil (%s)\n", pos, file, file.fullname()) }
                }
                if file.info == nil {/* ... */} else
                if t := file.info.ModTime(); t.After(tt) {
                        if true || gc.debug { fmt.Fprintf(stderr, "%s: touch %v → %v (%v)\n", pos, gc.target, file, t) } //gc.targetFullName
                        t = launchTime //time.Now() // ...
                        err, tt = os.Chtimes(gc.targetFullName, t, t), t
                        if err != nil { diag.errorAt(pos, "chtimes failed: %v", err); return }
                }
        }

        // Report missing files, but system files are not treated as missing.
        if gc.report {
                if file == nil {
                        fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` not found\n", gc.targetFullName, linum, colnum, t.project.name, name)
                } else if !file.exists() {
                        fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` file not existed\n", gc.targetFullName, linum, colnum, t.project.name, name)
                }
        }
        return
}

func (t *traversal) savedGrepFileName(pos Position, targetFullName string) (filename string, err error) {
        var nameHash = sha256.New() //[sha256.Size]byte
        fmt.Fprintf(nameHash, "%s", targetFullName)

        // Make names like .grep/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
        var nameSum = nameHash.Sum(nil)
        var savedGrepFile = t.project.matchTempFile(pos, filepath.Join(".grep",
                fmt.Sprintf("%x", nameSum[ :1]),
                fmt.Sprintf("%x", nameSum[1:2]),
                fmt.Sprintf("%x", nameSum[2: ]),
        ))
        filename, err = fullnameOrStrval(savedGrepFile)
        return
}

func (t *traversal) loadSavedGrepFile(pos Position, gc *grepctx) (okay bool, err error) {
        gc.savedGrepFileName, err = t.savedGrepFileName(pos, gc.targetFullName)
        if err != nil { diag.errorAt(pos, "get saved grep filename failed: %v", err); return }

        gc.savedGrepFile = stat(pos, gc.savedGrepFileName, "", "")
        if gc.savedGrepFile == nil { return } // No saved grepfile yet!

        var file, ok = gc.target.(*File)
        if !ok {
                file = stat(pos, gc.targetFullName, "", "")
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
                diag.errorAt(pos, "open saved grep filename failed: %v", err); return
        }

        var gp Position
        //gp.Filename = gc.savedGrepFileName
        gp.Filename = gc.targetFullName

        defer savedGrepOSFile.Close()
        scanner := bufio.NewScanner(savedGrepOSFile)
        scanner.Split(bufio.ScanLines)
        for scanner.Scan() {
                var s = scanner.Text() //gp.Line += 1
                var ( sys, linum, colnum int; name string )
                if n, e := fmt.Sscanf(s, "%d %d %d %s", &sys, &linum, &colnum, &name); e == nil && n == 4 {
                        gp.Line = linum
                        var file *File
                        if file, err = t.searchGrepped(pos, gc, sys == 1, linum, colnum, name); err != nil { break }
                        if file != nil { file.position = gp }
                        if file != nil && gc.isTargetFile(file) { continue }
                }
        }
        gc.savedGrepFile.info, err = savedGrepOSFile.Stat()
        if err != nil { diag.errorAt(pos, "stat saved grep filename error: %v", err) } else { okay = true }
        return
}

func (t *traversal) grepTargetFile(pos Position, gc *grepctx) (err error) {
        var file *os.File
        if file, err = os.Open(gc.targetFullName); err != nil { return }
        defer func() { err = file.Close() } ()

        for _, x := range gc.rxs {
                if x.Regexp == nil {
                        x.Regexp, err = regexp.Compile(x.string)
                        if err != nil { return }
                }
        }

        var gp Position
        gp.Filename = gc.targetFullName

        scanner := bufio.NewScanner(file)
        scanner.Split(bufio.ScanLines)
        ForScan: for scanner.Scan() {
                var s = scanner.Text(); gp.Line += 1
                for _, x := range gc.rxs {
                        if sm := x.FindStringSubmatch(s); len(sm) > 1 && sm[1] != "" {
                                var name = sm[1]
                                var colnum = strings.Index(s, name) //strings.IndexFunc(s, isNotSpace)
                                if gc.save != nil {
                                        var d = 0 ; if x.bool { d = 1 } // system files
                                        fmt.Fprintf(gc.save, "%d %d %d %s\n", d, gp.Line, colnum, name)
                                }
                                var file *File
                                if file, err = t.searchGrepped(pos, gc, x.bool/*system files*/, gp.Line, colnum, name); err != nil { return }
                                if file != nil { file.position = gp }
                                if file == nil || gc.isTargetFile(file) { continue }
                                continue ForScan // found one
                        }
                }
        }
        return
}

func (t *traversal) grep(pos Position, gc *grepctx) (err error) {
        var targetName string
        switch v := gc.target.(type) {
        case *File:
                targetName = v.name
                gc.targetInfo = v.info
                gc.targetFullName = v.fullname()
                gc.targetDir = filepath.Dir(gc.targetFullName)
                if v.isSysFile() { return }
        default:
                gc.targetDir = t.project.absPath
                if targetName, err = v.Strval(); err != nil {
                        diag.errorAt(pos, "strval grep target '%v' failed: %s", v, err).
                                debug(optionDebugErrors, 1)
                        return
                }
                if filepath.IsAbs(targetName) {
                        gc.targetFullName = targetName
                } else {
                        gc.targetFullName = filepath.Join(gc.targetDir, targetName)
                }
                if file := stat(pos, gc.targetFullName, "", ""); file == nil {
                        diag.errorAt(pos, "grep: '%s' not found", gc.targetFullName).
                                debug(optionDebugErrors, 1)
                        return
                } else {
                        gc.targetInfo = file.info
                }
        }
        if err != nil {
                diag.errorAt(pos, "grep target %s: %v", targetName, err).
                        debug(optionDebugErrors, 1)
                return
        }

        if gc.targetInfo == nil { return }
        if gc.done == nil { gc.done = make(map[string]int) }
        if !filepath.IsAbs(gc.targetFullName) {
                diag.errorAt(pos, "grep: '%s' is not abs", gc.targetFullName).
                        debug(optionDebugErrors, 1)
                return
        } else {
                gc.done[gc.targetFullName] += 1
        }
        if n, done := gc.done[gc.targetFullName]; done && n > 1 {
                if gc.debug { diag.errorAt(pos, "%v (done %v)", gc.targetFullName, n).debug(true,1) }
                return
        }

        if false { defer un(tt(t_traverse, t, gc.target)) }
        if files, cached := grepcache[gc.targetFullName]; cached {
                if gc.debug { diag.errorAt(pos, "grepcache: %v → %v", gc.targetFullName, files).debug(true,1) }
                if t.grepped = append(t.grepped, files...); gc.recursive {
                        for _, gc.target = range files {
                                if err = t.grep(pos, gc); err != nil {
                                        diag.errorAt(pos, "grep files: %v", err).
                                                debug(optionDebugErrors, 1)
                                        break
                                }
                        }
                }
                return
        }
        defer func(restore []Value) {
                var touch = gc.greptouch
                gc.files = restore
                grepcache[gc.targetFullName] = touch.files
                if gc.debug { diag.errorAt(pos, "grepped: %s → %v (grepped=%v) (saved=%s)\n",
                        gc.target, touch.files, len(t.grepped), gc.savedGrepFile).debug(true,1) }
                if gc.recursive && len(touch.files) > 0 {
                        for _, gc.target = range touch.files {
                                t.grepped = append(t.grepped, gc.target)
                                if err = t.grep(pos, gc); err != nil {
                                        diag.errorAt(pos, "grep files (deferred): %v", err).
                                                debug(optionDebugErrors, 1)
                                        break
                                }
                        }
                }
                if err == nil && gc.touch {
                        if err = touch.work(pos, gc); err != nil {
                                diag.errorAt(pos, "grep touch failed: %v", err).
                                        debug(optionDebugErrors, 1)
                        }
                }
        } (gc.files)

        gc.files = nil

        var savedGrepFile *os.File
        var savedGrepFileLoaded bool
        savedGrepFileLoaded, err = t.loadSavedGrepFile(pos, gc)
        if false && gc.debug {
                fmt.Fprintf(stderr, "%s: saved: %v → %v (%v)\n", pos, gc.target, gc.files, err)
        }
        if err != nil {
                diag.errorAt(pos, "grep loadSavedGrepFile: %v", err)
                return
        }
        if savedGrepFileLoaded { return } else
        if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                        diag.errorAt(pos, "grep mkdir: %v", err)
                        return
                }
        }
        if optionSaveGrepSourceName {
                var perm = os.FileMode(0600)
                var data = []byte(gc.targetFullName)
                var name = gc.savedGrepFileName + ".src"
                if err = ioutil.WriteFile(name, data, perm); err != nil {
                        diag.errorAt(pos, "grep write file: %v", err)
                        return
                }
        }
        if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil {
                diag.errorAt(pos, "grep create %s: %v", gc.savedGrepFileName, err)
                return
        }
        gc.save = bufio.NewWriter(savedGrepFile)
        defer func() {
                gc.save.Flush()
                savedGrepFile.Close()
        } ()

        err = t.grepTargetFile(pos, gc)
        if false && gc.debug {
                fmt.Fprintf(stderr, "%s: grepped: %v → %v (%v)\n", pos, gc.target, gc.files, t.grepped)
        }
        if err != nil && !gc.discard {
                diag.errorAt(pos, "grep target file: %v", err)
        } else {
                err = nil
        }
        return
}

var stopgrep = 0

type modifierGrepFilesOpts struct {
        debug bool `d,debug`
        verbose bool `v,verbose`
        discard bool `c,cast;dc,discard;dm,discard-missing;im,ignore-missing`
        sys []string `s,sys;ss,system`
        reg []string `re,reg;regx,regex;x,rx`
        langs []string `l,lang;lan,language`
        touch bool `t,touch;t,touch-outdate;t,touch-outdated`
        recursive bool `a,all;r,recur;rr,recursive`
        noTraverse bool `n,notraverse;nt,no-traverse;go,grep-only`
}
// grep-files - grep files from target, example usage:
//
//      (grep-files -x='\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrepFiles(pos Position, t *traversal, args... Value) (result Value, err error) {
        var gc grepctx
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge grep-files args failed: %v", err).
                        debug(optionDebugErrors, 1)
                return
        }
        if args, err = parseOpts(pos, &gc.modifierGrepFilesOpts, args...); err != nil {
                diag.errorAt(pos, "parse grep-files args failed: %v", err).
                        debug(optionDebugErrors, 1)
                return
        }
        for _, s := range gc.sys { gc.rxs = append(gc.rxs, &greprex{s, true , nil}) }
        for _, s := range gc.reg { gc.rxs = append(gc.rxs, &greprex{s, false, nil}) }
        for _, s := range gc.langs {
                if info, ok := langInfos[s]; !ok || info == nil {
                        diag.errorAt(pos, "lang '%s' is unknown", s).
                                debug(optionDebugErrors, 1)
                        return
                } else {
                        for _, re := range info.rxs { gc.rxs = append(gc.rxs, &greprex{re, false, nil}) }
                        for _, re := range info.sys { gc.rxs = append(gc.rxs, &greprex{re, true , nil}) }
                }
        }
        if len(gc.rxs) == 0 {
                diag.errorAt(pos, "no grep expressions: %v %v %v %v", gc.sys, gc.reg, gc.langs, args).
                        debug(optionDebugErrors, 1)
                return
        }

        var (
                targets = args
                grepped = t.grepped
        )
        if len(targets) == 0 { if tv := t.def.target.value; isNil(tv) || isNone(tv) {
                diag.errorAt(pos, "no grep target").
                        debug(optionDebugErrors, 1)
                return
        } else {
                targets = append(targets, tv)
        }}

        if gc.debug {
                diag.warnAt(pos, "grep files: %v %v %v\n", t.def.target.value, gc.rxs, args).
                        debug(optionDebugErrors, 1)
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
                        diag.prompt("smart: Grep %v …… (%d files in %v)\n",
                                s, len(grepped), time.Now().Sub(ts))
                } (time.Now())
        }
ForTarget:
        for _, target := range targets {
                if isNil(target) {
                        diag.errorAt(pos, "found nil grep target for %v", t.def.target.value).
                                debug(optionDebugErrors, 1)
                        return
                }
                if isNone(target) {
                        diag.errorAt(pos, "grep target '%v' is none for %v", target,
                                t.def.target.value).debug(optionDebugErrors, 100)
                        return
                }

                gc.target = target
                t.grepped = nil
                if err = t.grep(pos, &gc); err != nil {
                        diag.errorAt(pos, "grep files from %v failed: %v", target, err).
                                debug(optionDebugErrors, 1)
                        return
                } else if gc.noTraverse {
                        // does nothing
                } else if len(t.grepped) > 0 {
                        for _, val := range t.grepped {
                                if val.traverse(t); t.hasBreakers() {
                                        for _, brk := range t.breakers {
                                                switch brk.what {
                                                case breakErro:
                                                        diag.errorAt(brk.pos, "broken traversal for grepped %v", val)
                                                        diag.errorAt(brk.pos, "broken traversal with error: %v", brk.error).
                                                                debug(optionDebugErrors, 1)
                                                default:
                                                        diag.errorAt(brk.pos, "broken traversal for grepped %v: (%v) %v", val, brk.what, brk.message).
                                                                debug(optionDebugErrors, 1)
                                                }
                                        }
                                        break ForTarget
                                }
                        }
                }
                grepped = append(grepped, t.grepped...)
        }
        t.grepped = grepped

        if err != nil {
                diag.errorAt(pos, "grep files failed: %v", err).
                        debug(optionDebugErrors, 1)
        } else if !gc.noTraverse {
                t.def.grepped.value = MakeNone(pos)
                t.grepped = nil
        } else {
                result = MakeListOrScalar(pos, t.grepped)
        }
        return
}

type modifierGrepOpts struct {
        files bool `f,file;fi,files`
        string bool `s,str;sv,strings;v,value`
}

// grep - grep from target file, flags:
//
//    -files            grep files with stats, set $-
//    -dependencies     grep dependencies values (or files), set $^, $<, etc.
//
// Example usage:
//
//      (grep -files '\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrep(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierGrepOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge grep args failed: %v", err).
                        debug(optionDebugErrors, 1)
                return
        }
        if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse grep args failed: %v", err).
                        debug(optionDebugErrors, 1)
                return
        }
        diag.errorAt(pos, "unimplemented grep %v", args).
                debug(optionDebugErrors, 1)
        return
}

type modifierTouchOpts struct {
        mode os.FileMode `m,mode`
        path bool `p,path`
        verbose bool `v,verbose`
}
func modifierTouch(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierTouchOpts // = modifierTouchOpts{ mode: os.FileMode(0755) }
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge touch args failed: %v", err)
                return
        }
        if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse touch opts failed: %v", err)
                return
        } else if len(args) == 0 {
                args = append(args, t.def.target.value)
        }

        var files []*File
        for _, arg := range args {
                var vf []*File
                if err = touch(arg, uint32(opts.mode), opts.path); err != nil {
                        diag.errorAt(pos, "touch '%v' failed: %v", arg, err).
                                debug(optionDebugErrors, 1)
                        break
                } else if vf, err = arg.stamp(t); err != nil {
                        diag.errorAt(pos, "touch '%v' failed: %v", arg, err).
                                debug(optionDebugErrors, 1)
                        break
                } else { files = append(files, vf...) }
        }
        if opts.verbose { reportFileUpdates(pos, t.start, files) }
        if len(t.program.getModifies("stamp")) > 0 {
                diag.warnAt(pos, "no need to use a (stamp) after (touch)")
        }
        return
}

type modifierCheckOpts struct {
        answer bool `a,answer`
        boolean bool `b,boolean;r,result`
        verbose bool `v,verbose`
        silent bool `s,slient`
        good bool `g,good`
}
// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
func modifierCheck(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                diag.errorAt(pos, "merge check args failed: %v", err)
                return
        }

        var (
                opts modifierCheckOpts
                optBreak breakind // breaking with good results
                makeResult func(Position,bool) Value // returns results only if non-nil
                value Value = t.def.buffer.value
                values []Value
                pairs []*Pair
        )
        if args, err = parseOpts(pos, &opts, args...); err != nil {
                diag.errorAt(pos, "parse check args failed: %v", err)
                return
        }
        if opts.good    { optBreak   = breakDone }
        if opts.answer  { makeResult = MakeAnswer }
        if opts.boolean { makeResult = MakeBoolean }
        if opts.silent && makeResult == nil { makeResult = MakeBoolean }
        for _, arg := range args {
                switch t := arg.(type) {
                case *Pair: pairs = append(pairs, t)
                default:
                        diag.errorAt(pos, "unknown check '%v' (%T)", arg, arg)
                        return
                }
        }

ForPairs:
        for _, p := range pairs {
                var key, str string
                if key, err = p.Key.Strval(); err != nil {
                        diag.errorOf(p.Key, "stringify '%v' failed: %v", p.Key, err)
                        return
                }
                switch key {
                case "status":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                diag.errorOf(value, "value '%v' (%T) is not exec result", value, value)
                                t._breakf(pos, optBreak, "value '%v' is not exec result", value)
                                return
                        } else { exeres.wg.Wait() }

                        var num int64
                        if num, err = p.Value.Integer(); err != nil {
                                diag.errorAt(p.Value.Position(), "%v", err)
                                return
                        }
                        if opts.verbose {
                                fmt.Fprintf(stderr, "smart: Checking status ")
                                if num != 0 { fmt.Fprintf(stderr, "== %d ", num) }
                                fmt.Fprintf(stderr, "…")
                        }

                        var good = exeres.Status == int(num)
                        if opts.verbose {
                                var s string 
                                if good { s = "Yes" } else { s = "No" }
                                fmt.Fprintf(stderr, "… %s (%d)\n", s, exeres.Status)
                        }

                        if false { fmt.Fprintf(stderr, "%s: %v, %v\n", pos, exeres.Status, num) }

                        if makeResult != nil {
                                values = append(values, makeResult(pos, good))
                        } else if !good {
                                t._breakf(pos, optBreak, "bad status (%v) (expects %v)", exeres.Status, p.Value)
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                t._breakf(pos, optBreak, "not an exec result (%T)", value)
                                return
                        } else { exeres.wg.Wait() }
                        if opts.verbose { fmt.Printf("(status=%d)", exeres.Status) }

                        var v *bytes.Buffer
                        switch key {
                        case "stdout": v = exeres.Stdout.Buf
                        case "stderr": v = exeres.Stderr.Buf
                        default: unreachable()
                        }

                        if v == nil {
                                t._breakf(pos, optBreak, "bad %s (expects %v)", key, p.Value)
                                break ForPairs
                        }
                        if str, err = p.Value.Strval(); err != nil { 
                                return
                        } else if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                t._breakf(pos, optBreak, "bad %s (%v) (expects %v)", key, v, p.Value)
                                break ForPairs
                        }
                case "file", "dir":
                        var file *File
                        var project = t.project
                        if str, err = p.Value.Strval(); err != nil { return }
                        if file := project.FindFile(str); !file.exists() {
                                t._breakf(pos, optBreak, "`%v` no such file or directory", p.Value)
                                break ForPairs
                        }
                        switch key {
                        case "file":
                                if res := file.info.Mode().IsRegular(); makeResult != nil {
                                        values = append(values, makeResult(pos, res))
                                } else if !res {
                                        t._breakf(pos, optBreak, "`%v` is not a regular file", p.Value)
                                        break ForPairs
                                }
                        case "dir":
                                if res := file.info.Mode().IsDir(); makeResult != nil {
                                        values = append(values, makeResult(pos, res))
                                } else if !res {
                                        t._breakf(pos, optBreak, "`%v` is not a directory", p.Value)
                                        break ForPairs
                                }
                        default: unreachable()
                        }
                case "var":
                        var g, ok = p.Value.(*Group)
                        if !ok {
                                t._breakf(pos, optBreak, "`%v` is not a group value", p.Value)
                                break ForPairs
                        }
                        for _, elem := range g.Elems {
                                switch p := elem.(type) {
                                case *Pair:
                                        var k, a, b string
                                        if k, err = p.Key.Strval(); err != nil { break ForPairs }
                                        var def = t.program.project.scope.FindDef(k)
                                        if def != nil {
                                                if a, err = p.Value.Strval(); err != nil { break ForPairs }
                                                if b, err = def.value.Strval(); err != nil { break ForPairs }
                                                if res := a != b; makeResult != nil {
                                                        values = append(values, makeResult(pos, res))
                                                } else if !res {
                                                        t._breakf(pos, optBreak, "`%v` != `%v`", p.Key, p.Value)
                                                        break ForPairs
                                                }
                                        } else if makeResult != nil {
                                                values = append(values, makeResult(pos, false))
                                        } else {
                                                t._breakf(pos, optBreak, "`%v` is not defined", k)
                                                break ForPairs
                                        }
                                default:
                                        t._breakf(pos, optBreak, "`%v` unsupported checks", elem)
                                        break ForPairs
                                }
                        }
                default:
                        diag.errorAt(pos, "unknown check '%v'", p.Key)
                        break ForPairs
                }
        }
        if err == nil && values != nil {
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

func copyRegular(pos Position, src, dst string, opts *copyopts) (err error) {
        var def1, def2 *Def
        if true {
                def1 = opts.program.scope.Lookup("1").(*Def)
                def2 = opts.program.scope.Lookup("2").(*Def)
        } else {
                def1 = context.globe.scope.Lookup("1").(*Def)
                def2 = context.globe.scope.Lookup("2").(*Def)
        }
        defer func(v1, v2 Value) { def1.value, def2.value = v1, v2
                if err == nil {
                        var file = stat(pos, dst, "", "")
                        context.globe.stamp(dst, file.info.ModTime())
                }
        } (def1.value, def2.value)
        def1.value = &String{valbase{pos},dst}
        def2.value = &String{valbase{pos},src}

        var head, foot string
        if opts.head != nil {
                if head, err = opts.head.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                if false { fmt.Fprintf(stderr, "%s: %v => %s\n", opts.head.Position(), opts.head, head) }
        }
        if opts.foot != nil {
                if foot, err = opts.foot.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
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

func copySymlink(pos Position, src, dst string, opts *copyopts) (err error) {
        err = errors.New("copy symlink unimplemented")
        return
}

func copyDir(pos Position, src, dst string, opts *copyopts) (err error) {
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
                err = copyFile(pos, fi, ss, sd, opts)
                if err != nil { break }
        }
        return
}

func copyFile(pos Position, srcFi os.FileInfo, src, dst string, opts *copyopts) (err error) {
        if m := srcFi.Mode(); m&os.ModeSymlink != 0 {
                if opts.mode == 0 { opts.mode = srcFi.Mode() }
                err = copySymlink(pos, src, dst, opts)
        } else if srcFi.IsDir() {
                err = copyDir(pos, src, dst, opts)
        } else if m.IsRegular() {
                if opts.mode == 0 { opts.mode = srcFi.Mode() }
                err = copyRegular(pos, src, dst, opts)
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
func modifierCopyFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierCopyFileOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var target Value
        var source Value
        if len(args) > 0 {
                target = args[0]
        } else {
                target = t.def.target.value
        }
        if len(args) > 1 {
                source = args[1]
        } else {
                source = t.def.depend0.value
        }

        // Get target filename
        var (
                project = t.project
                filename, srcname string
                filetime, srctime time.Time
        )
        switch t := target.(type) {
        case *File:
                if filename = t.fullname(); t.info != nil {
                        filetime = t.info.ModTime()
                }
        default:
                if filename, err = target.Strval(); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", target, err)
                        return
                } else if file := project.FindFile(filename); file != nil {
                        target, filename = file, file.fullname()
                        if file.info != nil {
                                filetime = file.info.ModTime()
                        }
                }
        }
        switch t := source.(type) {
        case *File:
                if srcname = t.fullname(); t.info != nil {
                        srctime = t.info.ModTime()
                }
        default:
                if srcname, err = source.Strval(); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", source, err)
                        return
                } else if file := project.FindFile(srcname); file != nil {
                        source, srcname = file, file.fullname()
                        if file.info != nil { srctime = file.info.ModTime() }
                }
        }

        if filepath.Base(srcname) != filepath.Base(filename) {
                fmt.Fprintf(stderr, "%s:warning: %v, %v, %v\n", pos, target, filename, srcname)

                a := t.def.target.value
                b := t.def.depend0.value
                c := t.def.depends.value
                fmt.Fprintf(stderr, "%s:warning: %v\n", a.Position(), a)
                fmt.Fprintf(stderr, "%s:warning: %v\n", b.Position(), b)
                fmt.Fprintf(stderr, "%s:warning: %v\n", c.Position(), c)
        }

        if !filetime.IsZero() && filetime.After(srctime) {
          if opts.update {
            if opts.verbose { fmt.Fprintf(stderr, "smart: Update %v …", target) }
          } else if opts.override {
            if opts.verbose { fmt.Fprintf(stderr, "smart: Override %v …", target) }
          } else {
            if opts.verbose { fmt.Fprintf(stderr, "smart: Copy %v …… already existed!\n", target) }
            if !opts.silent { diag.errorAt(pos, "file already existed (%s)", target) }
            return
          }
        } else if opts.verbose {
                if opts.update {
                        fmt.Fprintf(stderr, "smart: Checking %v …", target)
                } else {
                        fmt.Fprintf(stderr, "smart: Copy %v …", target)
                }
        }

        if opts.quick {
                var file = stat(pos,filename,"","",nil)
                if file == nil || file.info != nil {
                        if opts.verbose { fmt.Fprintf(stderr, "… Good\n") }
                        return
                }
        }

        var copts = &copyopts{
                t.program, opts.path||opts.recursive,
                opts.update, opts.mode, opts.head, opts.foot,
                0, 0, 0,
        }
        var file *File
        if file = stat(pos,srcname,"","",nil); file == nil || file.info == nil {
                diag.errorAt(pos, "'%s' source file not found", srcname)
        } else if !file.info.IsDir() {
                if opts.mode == 0 { opts.mode = file.info.Mode() }
                if err = copyFile(pos, file.info, srcname, filename, copts); err != nil { diag.errorAt(pos, "%v", err) }
        } else if opts.recursive {
                if err = copyDir(pos, srcname, filename, copts); err != nil { diag.errorAt(pos, "%v", err) }
        } else {
                diag.errorAt(pos, "`%v` is a directory (use -r to solve it)", source)
        }

        if opts.verbose {
                if err != nil {
                        fmt.Fprintf(stderr, "… error\n")
                } else if copts.copied == 0 {
                        fmt.Fprintf(stderr, "… Good (%d files)\n", copts.files)
                } else if copts.copied == 1 {
                        fmt.Fprintf(stderr, "… Copied %d bytes\n", copts.bytes)
                } else {
                        fmt.Fprintf(stderr, "… Copied %d bytes (%d/%d)\n", copts.bytes, copts.copied, copts.files)
                }
        }
        return
}

func modifierWriteFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                filename, str string
                f *os.File
        )
        if filename, err = fullnameOrStrval(t.def.target.value); err != nil {
                return
        } else if f, err = os.Create(filename); err == nil {
                defer f.Close()
                if str, err = t.def.buffer.value.Strval(); err != nil {
                        return
                } else if _, err = f.WriteString(str); err == nil {
                        result = stat(pos, filename, "", "")
                } else {
                        os.Remove(filename)
                }
        } else {
                t._break(pos, breakFail).message = fmt.Sprintf("file %s not generated", t.def.target.value)
        }
        return
}

type modifierReadFileOpts struct {
        debug bool "d,debug"
        verbose bool "v,verbose"
        head Value "h,head"
        foot Value "f,foot"
}
func modifierReadFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( opts modifierReadFileOpts; filename string )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var target Value
        if n := len(args); n > 1 {
                diag.errorAt(pos, "too many files: %v", args)
                return
        } else if n == 1 {
                target = args[0]
        } else {
                target = t.def.target.value
        }

        if isNil(target) {
                diag.errorAt(pos, "target is <nil>")
                return
        } else if isNone(target) {
                diag.errorAt(pos, "target is <none>")
                return
        } else if filename, err = fullnameOrStrval(target); err != nil {
                diag.errorOf(target, "strval '%v' error: %v", target, err)
                return
        } else if filename == "" {
                diag.errorOf(target, "target filename is empty")
                return
        }

        if opts.debug {
                diag.infoAt(pos, "read-file: %v", filename)
        }

        var bytes []byte
        if bytes, err = ioutil.ReadFile(filename); err == nil {
                var s, v string
                if opts.head != nil {
                        if v, err = opts.head.Strval(); err == nil { s = v } else {
                                diag.errorAt(pos, "%v", err); return
                        }
                }
                s += string(bytes)
                if opts.foot != nil {
                        if v, err = opts.foot.Strval(); err == nil { s += v } else {
                                diag.errorAt(pos, "%v", err); return
                        }
                }
                t.def.buffer.value = &String{valbase{pos},s}
        } else {
                t._break(pos, breakErro).error = err
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
            fmt.Fprintf(stderr, "crc64CheckFileModeContent: %v %v\n%s\n%s\n", a, b, s, content)
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
        path bool "p,path"
        debug bool "d,debug"
        verbose bool "v,verbose"
        mode os.FileMode "m,mode"
}
func modifierUpdateFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( opts = modifierUpdateFileOpts{ mode: os.FileMode(0640) }; filename string )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var target Value
        if len(args) > 0 { target = args[0] } else { target = t.def.target.value }
        if len(args) > 1 { if opts.mode, err = permVal(args[1], 0600); err != nil { return }}

        // Get target filename
        switch p := target.(type) {
        case *File: filename = p.fullname()
        case *Path:
                if filename, err = p.Strval(); err != nil {
                        diag.errorAt(pos, "strval path '%v' failed: %v", p, err)
                        return
                }
        default:
                if filename, err = target.Strval(); err != nil {
                        diag.errorAt(pos, "strval '%v' failed: %v", p, err)
                        return
                } else if file := t.project.FindFile(filename); file != nil {
                        target, filename = file, file.fullname()
                }
        }

        if opts.debug {
                diag.infoAt(pos, "update-file: %v (%v) (%v, %v)",
                        target, filename, t.project, cloctx).debug(optionDebugErrors, 1)
        }

        if opts.path { // Make path (mkdir -p)
                if p := filepath.Dir(filename); p != "." && p != "/" {
                        err = os.MkdirAll(p, os.FileMode(0755))
                        if err != nil { return }
                }
        }

        // Check existed file content checksum
        var content string
        if content, err = t.def.buffer.value.Strval(); err != nil { return }
        if content == "" && (opts.verbose || opts.debug) { fmt.Fprintf(stderr, "%s: empty content\n", pos) }
        if opts.verbose { fmt.Fprintf(stderr, "smart: Checking %v …", trimPromptString(target.String())) }
        if same, e := crc64CheckFileModeContent(filename, []byte(content), opts.mode); e != nil {
                if false { // discard error (e.g.: no such file or directory)
                        fmt.Fprintf(stderr, "… (error: %s)\n", e)
                        diag.errorAt(pos, "%v", e)
                        return
                }
        } else if same {
                if opts.verbose { fmt.Fprintf(stderr, "… Good\n") }
                t.removeCallerUpdated(target) // remove timestamp updated
                result = stat(pos, filename, "", "")
                return
        } else if opts.verbose {
                fmt.Fprintf(stderr, "… Outdated (%s)\n", filename)
        }

        if opts.verbose {
                printEnteringDirectory()
                if false {
                        fmt.Fprintf(stderr, "smart: Update %v …", filename)
                } else {
                        s := target.String()
                        if len(s) > maxPromptStr { s = "…"+s[len(s)-maxPromptStr:] }
                        fmt.Fprintf(stderr, "smart: Update %v …", s)
                }
        }

        // Create or update the file with new content

        var f *os.File
        f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts.mode)
        if err == nil && f != nil {
                defer func() {
                        if err = f.Close(); err != nil {
                                os.Remove(filename)
                                return
                        }
                        var file = stat(pos, filename, "", "")
                        if  file == nil {
                                diag.errorAt(pos, "invalid file '%s'", filename)
                        } else {
                                var files []*File
                                if files, err = file.stamp(t); err != nil {
                                        diag.errorAt(pos, "%v", err).debug(optionDebugErrors,1)
                                        return
                                } else if opts.verbose {
                                        reportFileUpdates(pos, t.start, files)
                                }
                                result = file // resulting the updated file
                        }
                } ()
                if _, err = f.WriteString(content); err == nil {
                        if opts.verbose { fmt.Fprintf(stderr, "… (ok)\n") }
                } else {
                        if opts.verbose { fmt.Fprintf(stderr, "… (%s)\n", err) }
                }
        } else {
                if opts.verbose { fmt.Fprintf(stderr, "… (%s)\n", err) }
                t._break(pos, breakFail).message = fmt.Sprintf("file %s not updated", t.def.target.value)
        }
        return
}

type modifierWaitOpts struct {
        verbose bool "v,verbose"
        stdout bool "o,stdout"
        stderr bool "e,stderr"
        status bool "s,status"
        trim bool "t,trim" // trim heading and tailing spaces of the result
        execRes bool "x,exec"
        asType string "a,as"
}
func modifierWait(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierWaitOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }
        if opts.verbose {
                fmt.Fprintf(stderr, "smart: Wait (%v) …\n", t.def.target.value)
        }

        const stampCurrentTarget = true
        var (
                execRes *ExecResult
                waitForExecResult = opts.stdout || opts.stderr || opts.status || opts.execRes
        )

        // Wait for prerequisites and/or execution
        _, _, execRes, err = t.wait(pos, opts.verbose, waitForExecResult, stampCurrentTarget)
        if opts.verbose {
                var s = "Done"
                if err != nil { s = "Fail" }
                fmt.Fprintf(stderr, "smart: %s (%v), updated=%v\n", s, t.def.target.value, t.updated)
        }

        if execRes != nil {
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

func reportFileUpdates(pos Position, start time.Time, files []*File) {
        for _, file := range files {
                var (
                        mod = file.info.ModTime()
                        d = time.Now().Sub(start)
                )
                if mod.After(start) {
                        if false {
                                diag.prompt("smart: Updated %v (%v, ModTime=%v)\n", file, d, mod)
                        } else {
                                diag.prompt("smart: Updated %v (%v)\n", file, d)
                        }
                } else {
                        diag.prompt("smart: File %v not changed (%v, ModTime=%v)\n", file, d, mod)
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
func modifierStamp(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierStampOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        // Wait for ExecResult (see also modifier (wait -exec))
        const waitForExecResult = true
        const stampCurrentTarget = true

        // Wait for prerequisites
        var target Value
        if target, _, _, err = t.wait(pos, opts.prompt, waitForExecResult, stampCurrentTarget); err == nil {
                return
        } else if opts.next {
                if opts.verbose { diag.warnAt(pos, "%v", err).debug(optionDebugErrors, 1) }
                t._break(pos, breakNext).scope = breakTrave
                err = nil // discard the error
        } else if opts.error {
                if opts.debug > 0 {
                        t.traceCallStack(pos, "%v", err).debug(optionDebugErrors, opts.debug)
                } else {
                        diag.errorAt(pos, "%v", err).debug(optionDebugErrors, 1)
                }
                t._break(pos, breakErro).error = err
        } else if t.stems != nil {
                if opts.debug > 0 {
                        diag.warnAt(pos, "%v", err).debug(optionDebugErrors, opts.debug)
                        t.traceCallStack(pos, "%v", err).debug(optionDebugErrors, opts.debug)
                } else {
                        diag.warnAt(pos, "%v", err).debug(optionDebugErrors, 1)
                }
                t._break(pos, breakNext).scope = breakTrave
                err = nil // discard the error
        } else if pos.IsValid() {
                t.traceCallStack(pos, "failed: %v", err).debug(optionDebugErrors, 1)
        } else if targetPos := target.Position(); targetPos.IsValid() {
                t.traceCallStack(targetPos, "failed: %v", err).debug(optionDebugErrors, 1)
        } else {
                // TODO: dump more diagnostics information here
        }

        if err != nil {
                if pe, ok := err.(*fs.PathError); ok {
                        //err = fmt.Errorf("stamp %s: %s", trimPromptString(pe.Path), pe.Err.Error())
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
func predict(pos Position, t *traversal, args... Value) (result bool, breakScope breaksco, message string, err error) {
        var target string
        if target, err = fullnameOrStrval(t.def.target.value); err != nil {
                diag.errorAt(pos, "stringify predict target failed: %v", err)
                return
        }

        var num int64
        for caller := t.caller; caller != nil; caller= caller.caller {
                if true {
                        var same = t.def.target.value == caller.def.target.value
                        if !same && false {
                                cmp := t.def.target.value.cmp(caller.def.target.value)
                                same = (cmp == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n, ok := caller.visited[t.def.target.value]; ok && n > 0 {
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
                        fmt.Fprintf(stderr, "… %s\n", status)
                }
        } ()

        for _, arg := range args {
                var va []Value // va = merge(arg)
                if va, err = mergeresult(ExpandAll(arg))  ; err != nil {
                        diag.errorAt(pos, "merge arg '%v' failed: %v", arg, err)
                        return
                }
                if va, err = parseOpts(pos, &opts, va...) ; err != nil {
                        diag.errorAt(pos, "parse opts failed: %v", err)
                        return
                }
                if opts.group    { breakScope = breakGroup }
                if opts.traverse { breakScope = breakTrave }
                if opts.verbose && !opts.verbose0 {
                        fmt.Fprintf(stderr, "smart: Checking %v …", target)
                        opts.verbose0 = true
                }
                //if opts.and && !result { continue }
                if !opts.and || (opts.and && result) { for i, a := range va {
                        if g, ok := a.(*Group); ok && len(g.Elems) > 0 {
                                var name string
                                if name, err = g.Elems[0].Strval(); err != nil {
                                        diag.errorOf(a, "predict: %v", err)
                                        return
                                }
                                if m, ok := modifiers[name]; ok {
                                        var res Value
                                        res, err = m(a.Position(), t, g.Elems[1:]...)
                                        if err != nil { diag.errorOf(a, "predict: %v", err); return }
                                        a = res // replace
                                }
                        }

                        var t bool
                        if a == nil {
                                continue // skip
                        } else if p, ok := a.(*prediction); ok {
                                if p.reason != "" { reasons = append(reasons, p.reason) }
                                t = p.bool
                        } else if t, err = a.True(); err != nil {
                                return
                        } else if t {
                                reasons = append(reasons, fmt.Sprintf("#%v", i+1))
                        }

                        if opts.and {
                                result = result && t
                                opts.and = false // reset -and flag
                        } else if t {
                                result = true
                                break
                        }
                }}
        }
        return
}

// (assert condition,'error message...')
func modifierAssert(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( res bool; sco breaksco; msg string )
        if res, sco, msg, err = predict(pos, t, args...); err != nil {
                diag.errorAt(pos, "prediction %v failed: %v", args, err)
        } else if !res {
                if msg == "" {
                        diag.errorAt(pos, "assertion failed: %v", args)
                } else {
                        diag.errorAt(pos, "assertion %v failed: %v", args, msg)
                }
                brk := t._break(pos, breakFail)
                brk.message = "assertion failure"
                brk.scope = sco
        }
        return
}

func modifierCond(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( res bool; sco breaksco; msg string )
        if res, sco, msg, err = predict(pos, t, args...); err != nil {
                diag.errorAt(pos, "predict: %v", err)
        } else if !res {
                brk := t._break(pos, breakDone)
                brk.message = msg
                brk.scope = sco
        }
        return
}

func modifierCase(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( res bool; sco breaksco; msg string )
        if res, sco, msg, err = predict(pos, t, args...); err == nil {
                brk := t._break(pos, breakNext) // next case
                brk.message = msg
                brk.scope = sco
                if res { brk.what = breakCase } // select case
        }
        return
}

type modifierDirtyOpts struct {
        checksum bool "c,checksum"
        debug bool "d,debug"
        verbose bool "v,verbose"
        silent bool "s,silent"
}
func modifierDirty(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierDirtyOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var (
                target Value
                reason string
                dirty bool
        )
        // Wait for prerequisites only
        if target, _, _, err = t.wait(pos); err != nil {
                diag.errorAt(pos, "wainting traversal failed: %v", err)
                return
        } else if dirty = t.hasBreakers(); dirty {
                reason = fmt.Sprintf("dirty (%v breakers)", len(t.breakers))
        } else if dirty = !t.exists(target); dirty {
                reason = "dirty: target not exists"
        } else if dirty = len(t.updated) > 0; dirty {
                reason = fmt.Sprintf("dirty (%v updated)", len(t.updated))
        } else if dirty, err = t.isRecipesDirty(); err != nil {
                diag.errorAt(pos, "isRecipesDirty: %v", err)
                return
        } else if dirty {
                reason = "dirty: recipes changed"
        } else if opts.checksum && !(isNil(t.def.depend0.value) || isNone(t.def.depend0.value)) {
                var file1, file2 string
                if file1, err = target.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
                if file2, err = t.def.depend0.value.Strval(); err != nil {
                        diag.errorAt(pos, "%v", err); return
                }
                if same, e := crc64CompareFileChecksum(file1, file2); e != nil {
                        diag.errorAt(pos, "%v", e); return
                } else if same {
                        reason = "Good"
                } else {
                        reason = "dirty: content changed"
                        dirty = true
                }
        } else {
                reason = "Good"
        }

        if opts.debug {
                var a = typeof(target)
                var e = t.exists(target)
                var s, _ = target.Strval()
                diag.errorAt(pos, "type=%s target=%s (exists=%v, dirty=%v, updated=%v)\n",
                        a, s, e, dirty, t.updated).debug(optionDebugErrors, 1)
        }

        if opts.verbose {
                var ( m, s string )
                if len(t.updated) > 0 { //s = fmt.Sprintf(", %v", t.updated)
                        s = " ("
                        for i, v := range t.updated {
                                if i > 0 { s += " " }
                                if len(s) > maxPromptStr {
                                        s += "…"
                                        break
                                } else { s += v.String() }
                        }
                        s += ")"
                } else if dirty {
                        s = " (" + strings.TrimPrefix(reason, "dirty: ") + ")"
                }
                if dirty { m = "Dirty" } else { m = "Good" }
                fmt.Fprintf(stderr, "smart: Stamp %s …… %s%s\n", target, m, s)
        }

        if optionTraceTraversal {
                t_traverse.tracef("dirty: %v (updated=%v, exists=%v, target=%v)", dirty, len(t.updated), t.exists(target), target)
                if len(t.updated) > 0 { t_traverse.tracef("dirty: updated=%v", t.updated) }
        }

        if opts.silent { reason = "" }
        result = &prediction{boolean{valbase{pos},dirty},reason}
        return
}

func modifierNoLoop(pos Position, t *traversal, args... Value) (result Value, err error) {
        var loop bool
        for caller := t.caller; caller != nil; caller= caller.caller {
                var same = t.def.target.value == caller.def.target.value
                if !same && false {
                        cmp := t.def.target.value.cmp(caller.def.target.value)
                        same = (cmp == cmpEqual)
                }
                if same {
                        //fmt.Printf("%s: loop: %v\n", pos, t.def.target.value)
                        loop = true
                        break
                }
        }

        var s string
        if !loop { s = "not " }
        s = fmt.Sprintf("loop %sdetected (%v)", s, t.def.target.value)
        result = &prediction{boolean{valbase{pos},!loop},s}
        return
}

type modifierTarget1stVisitOpts struct {
        silent bool "s,silent"
}
func modifierTarget1stVisit(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierTarget1stVisitOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var num int
        for caller := t.caller; caller != nil; caller= caller.caller {
                if false {
                        var same = t.def.target.value == caller.def.target.value
                        if !same && false {
                                cmp := t.def.target.value.cmp(caller.def.target.value)
                                same = (cmp == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n, ok := caller.visited[t.def.target.value]; ok && n > 0 {
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

type modifierTargetMaxVisitOpts struct {
        closure bool "c,closure"
        debug bool "d,debug;d,debug-trace;d,dump"
        silent bool "s,silent"
}
func modifierTargetMaxVisit(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierTargetMaxVisitOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var nth int64
        for _, a := range args {
                if nth, err = a.Integer(); err != nil {
                        diag.errorAt(pos, "%v", err)
                        return
                } else if nth <= 0 {
                        diag.errorAt(pos, "needs positive number (%v, %s)", a, typeof(a))
                        return
                }
        }

        var num int64
        var head bool = true
        for caller := t.caller; caller != nil; caller= caller.caller {
                if false {
                        if opts.closure && caller.closure == t.closure { continue }
                        var same = t.def.target.value == caller.def.target.value
                        if !same && false {
                                cmp := t.def.target.value.cmp(caller.def.target.value)
                                same = (cmp == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n, ok := caller.visited[t.def.target.value]; ok && n > 0 {
                        num += int64(n)
                }
                if opts.debug && num > 0 {
                        if head { head = false
                                fmt.Fprintf(stderr, "  %s: nth(%d)\n", pos, nth)
                        }
                        var pos = caller.program.position
                        fmt.Fprintf(stderr, "    %s: %v\n", pos, caller.def.target)
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
func modifierGitModified(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierGitModifiedOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil { diag.errorAt(pos, "%v", err); return }
 
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
                        if s, err = a.Strval(); err != nil { diag.errorAt(pos, "%v", err); return }
                        for _, v := range sm {
                                if false { fmt.Fprintf(stderr, "%s: %s\n%v\n", pos, s, v[1]) }
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
func modifierGitAhead(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierGitAheadOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var out = new(bytes.Buffer)
        var git = exec.Command("git", "status")
        git.Stdout, git.Stderr = out, os.Stderr
        if err = git.Run(); err != nil { diag.errorAt(pos, "%v", err); return }
 
        // TODO: check also for `Changes not staged for commit:`

        var rx = regexp.MustCompile(`\nYour branch is ahead of '(.+?)' by`)
        var sm = rx.FindAllSubmatch(out.Bytes(), 1)
        if len(sm) > 0 {
                var val bool = true
                result = &prediction{boolean{valbase{pos},val},"Work branch has new commits to push"}
        }
        return
}

var onceMutex sync.Mutex
var onceCache = make(map[HashBytes]int,64)
type modifierOnceOpts struct {
        debug bool "d,debug"
        verbose bool "v,verbose"
}
func modifierOnce(pos Position, t *traversal, args... Value) (result Value, err error) {
        var opts modifierOnceOpts
        if args, err = mergeresult(ExpandAll(args...)); err != nil { diag.errorAt(pos, "merge args failed: %v", err); return }
        if args, err = parseOpts(pos, &opts, args...) ; err != nil { diag.errorAt(pos, "parse opts failed: %v", err); return }

        var s string
        var h = sha256.New()
        if s, err = t.entry.target.Strval(); err != nil { return } else {
                fmt.Fprintf(h, "%s", s)
        }
        if s, err = fullnameOrStrval(t.def.target.value); err != nil { return } else {
                fmt.Fprintf(h, "%s", s)
        }
        for _, a := range args {
                if s, err = a.Strval(); err != nil { return } else {
                        fmt.Fprintf(h, "%s", s)
                }
        }

        var sum HashBytes
        copy(sum[:], h.Sum(nil))

        onceMutex.Lock()
        onceCache[sum] += 1
        onceMutex.Unlock()

        var num = onceCache[sum]

        if opts.debug {
                fmt.Fprintf(stderr, "%s: %v (once: num=%d)\n", pos, t.def.target.value, num)
        } else if opts.verbose {
                fmt.Fprintf(stderr, "once: %v (num=%d)\n", t.def.target.value, num)
        }

        if num > 1 {
                t._break(pos, breakDone).message = fmt.Sprintf("once (num=%d)", num)
        }
        return
}
