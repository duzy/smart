//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "crypto/sha256"
        "path/filepath"
        "runtime/debug"
        "hash/crc64"
        "io/ioutil"
        "strings"
        "regexp"
        "errors"
        "bufio"
        "bytes"
        "sync"
        "time"
        "fmt"
        "os"
        "io"
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
        case breakDone:         s = "break.done"
        case breakNext:         s = "break.next"
        case breakCase:         s = "break.case"
        case breakFail:         s = "break.fail"
        }
        return
}

const (
        breakUnkn breakind = iota
        breakDone // (cond ...) and (case ...)
        breakNext // (cond ...) and (case ...)
        breakCase // (case ...)
        breakFail // (assert ...)
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
}

func (p *breaker) Error() (s string) {
        switch p.what {
        case breakUnkn: s = "unknown"
        case breakDone: s = "done" // ineligible (cond) is ignored
        case breakNext: s = "next"
        case breakCase: s = "case"
        case breakFail: s = "failure" // "break with failure"
        }
        if p.pos.IsValid() {
                if p.message != "" { s += ": " + p.message }
                s = fmt.Sprintf("%s: %s", p.pos, s)
        }
        return
}

func (p *breaker) prerequisites() (res []*updatedtarget) {
        for _, u := range p.updated {
                res = append(res, u.prerequisites...)
        }
        return
}

func break_with(pos Position, w breakind, s string, a... interface{}) *breaker {
        return &breaker{ pos, w, breakGroup, fmt.Sprintf(s, a...), nil, nil }
}

func extractBreakers(err error) (res []*breaker, rest []error) {
        if err == nil { return }
        if optionEnableBenchmarks {
                s := fmt.Sprintf("extractBreakers(%s)", typeof(err))
                defer bench(mark(s))
        }
        switch t := err.(type) {
        case nil: break
        case *scanner.Error:
                var pos = Position(t.Pos)
                for _, e := range t.Errs {
                        brks, errs := extractBreakers(e)
                        if res = append(res, brks...); len(errs) > 0 {
                                rest = append(rest, wrap(pos, errs...))
                        }
                }
        case *breaker:
                res = append(res, t)
        default:
                rest = append(rest, err)
        }
        return
}

type modifier struct {
        trivial
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
func (m *modifier) closured() (res bool) {
        if res = m.name.closured(); !res {
                for _, a := range m.args {
                        if res = a.closured(); res { break }
                }
        }
        return
}
func (m *modifier) expand(_ expandwhat) (Value, error) { return m, nil }
func (_ *modifier) cmp(v Value) (res cmpres) { 
        if _, ok := v.(*modifier); ok { res = cmpEqual }
        return
}
func (m *modifier) traverse(t *traversal) (err error) {
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("modifier.traverse(%s)", m))) }
        if optionTraceTraversal { defer un(tt(t, m)) }
        return t.program.modify(t, m)
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
        trivial
        modifiers []*modifier
}
func (g *modifiergroup) refs(v Value) bool {
        for _, m := range g.modifiers {
                if m.refs(v) { return true }
        }
        return false
}
func (g *modifiergroup) closured() bool {
        for _, m := range g.modifiers {
                if m.closured() { return true }
        }
        return false
}
func (g *modifiergroup) exists() (res existence) {
        res = existenceMatterless
ForElems:
        for _, elem := range g.modifiers {
                switch elem.exists() {
                case existenceMatterless:
                case existenceConfirmed:
                        res = existenceConfirmed
                case existenceNegated:
                        res = existenceNegated
                        break ForElems
                }
        }
        return
}
func (g *modifiergroup) expand(_ expandwhat) (Value, error) { return g, nil }
func (_ *modifiergroup) cmp(v Value) (res cmpres) { 
        if _, ok := v.(*modifiergroup); ok { res = cmpEqual }
        return
}
func (g *modifiergroup) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, g)) }
        if optionEnableBenchmarks {
                s := fmt.Sprintf("modifiergroup.traverse(%s)", g)
                defer bench(mark(s))
        }
        for _, m := range g.modifiers {
                var done bool
                if done, err = g.one(t, m); done || err != nil { break }
        }
        return
}
func (g *modifiergroup) one(t *traversal, m *modifier) (bool, error) {
        if optionEnableBenchmarks && false {
                s := fmt.Sprintf("modifiergroup.one(%s)", m)
                defer bench(mark(s))
        }
        return g.handle(t, m.traverse(t))
}
func (g *modifiergroup) handle(t *traversal, e error) (done bool, err error) {
        if optionEnableBenchmarks && false {
                s := fmt.Sprintf("modifiergroup.handle")
                defer bench(mark(s))
        }

        if e == nil { return }

        var brks, errs = extractBreakers(e)
        if len(errs) > 0 {
                err = wrap(g.position, errs...)
                return
        }

        for _, e := range brks {
                t.breakers = append(t.breakers, e) // for (dirty) and interpreters.
                if e.what == breakCase { continue /* case selected */ }
                if e.what == breakDone && e.scope == breakGroup {
                        done = true
                } else {
                        // return the breakers
                        err = wrap(g.position, e, err)
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
                `env`:          modifierSetEnv,  // interpreter environments
                `set`:          modifierSetVar,

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

                `check`:        modifierCheck,
                `assert`:       modifierAssert,
                `case`:         modifierCase,
                `cond`:         modifierCond,

                `dirty`:        modifierDirty,
                `no-loop`:      modifierNoLoop,
                `once`:         modifierOnce,
                `target-1st-visit`: modifierTarget1stVisit,
                `target-max-visit`: modifierTargetMaxVisit,
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
                        err = fmt.Errorf("Modifier '%s' already existed", s)
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

func modifierPrint(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optStdout bool
                optStderr bool = true
                content string
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "o,stdout",
                "e,stderr",
        }, func(ru rune, v Value) {
                switch ru {
                case 'o': if optStdout, err = trueVal(v, true); err != nil { return }
                case 'e': if optStderr, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        if content, err = t.def.buffer.value.Strval(); err != nil { return }
        if optStdout { fmt.Fprint(stdout, content) }
        if optStderr { fmt.Fprint(stderr, content) }
        t.def.buffer.value = &None{trivial{pos}}
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

func modifierSetEnv(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var envars = new(List)
        if _, err = t.program.auto(TheShellEnvarsDef, envars); err != nil { return }
        for _, a := range args {
                if _, ok := a.(*Pair); ok {
                        envars.Append(a)
                } else {
                        err = errors.New(fmt.Sprintf("Invalid env `%v' (%T)", a, a))
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
func modifierSetVar(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        var defs []Value
        var none = &None{trivial{pos}}
ForArgs:
        for _, arg := range args {
                var name string
                var value Value = none
                switch a := arg.(type) {
                case *Bareword: name = a.string
                case *Pair:
                        if name, err = a.Key.Strval(); err == nil {
                                value = a.Value
                        } else { break ForArgs }
                case *Flag:
                        if name, err = a.name.Strval(); err == nil {
                                if value = none; name == "" { name = "-" }
                        } else { break ForArgs }
                default:
                        err = errorf(pos, "%T `%s` is unsupported (try: foo=value)", arg, arg)
                        break ForArgs
                }
                if def := t.program.scope.FindDef(name); def == nil {
                        err = errorf(pos, "`%s` no such def", name)
                        break ForArgs
                } else {
                        def.set(DefDefault, value)
                        defs = append(defs, def)
                }
        }
        if len(defs) > 0 { result = MakeListOrScalar(pos, defs) }
        return
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

        var (
                optDump bool
                optVerbose bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "d,dump",
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': if optDump    , err = trueVal(v, false); err != nil { return }
                case 'v': if optVerbose , err = trueVal(v, false); err != nil { return }
                }
        }); err != nil { return }

        if optDump {
                fmt.Fprintf(stderr, "%s: closure:\n", pos)
                for _, cc := range cloctx {
                        fmt.Fprintf(stderr, "    %s: %s\n", cc.position, cc.comment)
                }
        } else if optVerbose {
                fmt.Fprintf(stderr, "%s: %v\n", pos, cloctx)
        }

        var dir string // closure work directory
        if len(cloctx) == 0 {
                err = errorf(pos, "empty closure context")
        } else if def := cloctx[0].FindDef("/"); def == nil {
                err = wrap(pos, errorf(cloctx[0].position, "&/ is undefined"))
        } else if dir, err = def.value.Strval(); err != nil {
                err = wrap(pos, err)
        } else if dir == "" {
                err = wrap(pos, errorf(cloctx[0].position, "&/ is empty"))
        } else if !filepath.IsAbs(dir) {
                err = wrap(pos, errorf(cloctx[0].position, "&/ is relative"))
        } else if err = enter(t.program, dir); err == nil {
                t.program.project.changedWD = dir
                t.program.changedWD = dir
        }
        return
}

func modifierCD(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optPath bool
                optPrintEnter bool
                optPrintLeave bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "p,path",
                "e,print-enter",
                "l,print-leave",
                //"-",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': if optPath      , err = trueVal(v, true); err != nil { return }
                case 'e': if optPrintEnter, err = trueVal(v, true); err != nil { return }
                case 'l': if optPrintLeave, err = trueVal(v, true); err != nil { return }
                /*case '-':
                        var dir = findBacktrackDir()
                        // Back to main project if no backtracks.
                        if dir == "" && context.globe.main != nil {
                                dir = context.globe.main.AbsPath()
                        }
                        v = append(v, &String{dir})*/
                }
        }); err != nil { return }

        if optPrintEnter || optPrintLeave {
                if optPrintEnter { printEnteringDirectory() }
                if optPrintLeave { printLeavingDirectory() }
                if len(args) == 0 { return }
        }
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
                if optPath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
                }
                if err = enter(t.program, dir); err == nil {
                        t.program.project.changedWD = dir
                        t.program.changedWD = dir
                }
        } else {
                err = errorf(pos, "cd: wrong number of args (%v)", args)
        }
        return
}

func modifierMkdir(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optMode = os.FileMode(0755)
                optVerbose bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "m,mode",
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                case 'm': if optMode   , err = permVal(v, 0600); err != nil { return }
                }
        }); err != nil { return }
        if len(args) == 0 {
                var s string
                if s, err = t.def.target.value.Strval(); err != nil { err = wrap(pos, err) } else
                if err = os.MkdirAll(filepath.Dir(s), optMode); err != nil { err = wrap(pos, err) }
                return
        }
        for _, a := range args {
                var s string
                if s, err = a.Strval(); err != nil { err = wrap(pos, err); return }
                if err = os.MkdirAll(s, optMode); err != nil { err = wrap(pos, err); return }
        }
        return
}
// (path $(dir $@))
// (path /example/path)
func modifierPath(pos Position, t *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                // ...
        }, func(ru rune, v Value) {
                // ...
        }); err != nil { return }
        if len(args) == 0 {
                var s string
                if s, err = t.def.target.value.Strval(); err != nil { return }
                if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
                        err = os.MkdirAll(s, os.FileMode(0755))
                }
                return
        }
        for _, arg := range args {
                var s string
                if s, err = arg.Strval(); err != nil { return }
                if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                        return
                }
        }
        return
}

func modifierSudo(pos Position, t *traversal, args... Value) (result Value, err error) {
        panic("todo: sudo modifier is not implemented yet")
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
                                s := fmt.Sprintf("bad status %v", d.Status)
                                err = &breaker{ pos:pos, what:breakFail, message:s }
                                return // target shall be updated
                        } else {
                                depends.Append(d)
                        }
                case *RuleEntry:
                        switch d.Class() {
                        case GeneralRuleEntry, PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                                depends.Append(d)
                        default:
                                err = errorf(pos, "unsupported entry depend `%v' (%v)", d, d.Class())
                        }
                case *String:
                        /*if t.program.project.IsFile(d.Strval()) {
                                Fail("compare: discarded file depend %v (%T)", depend, depend)
                        } else*/ {
                                depends.Append(d)
                        }
                case *File:
                        depends.Append(d)
                default:
                        err = errorf(pos, "unsupported entry depend `%v' (%v)", depend, t.program.depends)
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
        debug, verbose bool
        discard, report bool // discard or report missing greps
        recursive, touch bool
        rxs []*greprex
        done map[string]int
        greptouch
        savedGrepFileName string
        savedGrepFile *File
        save *bufio.Writer
}
type greprex struct{ string ; bool ; *regexp.Regexp }
func (g *greprex) String() string { return g.string }
func (g *greptouch) work(pos Position, gc *grepctx) (err error) {
        if g.targetInfo == nil {
                err = errorf(g.target.Position(), "'%v' not exists", g.target)
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
                        if s, err = file.Strval(); err != nil { err = wrap(pos, err); return }
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
        } else if s, _ := file.Strval(); s == g.targetFullName {
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
                if !exists(file) {
                        var f = t.project.matchFile(name)
                        if f != nil { file = f }
                }
        } else if file = t.project.matchFile(name); file == nil {
                return // file not found
                /*
        } else if !sys && file.match != nil && len(file.match.Paths) == 1 {
                // mark system files defined by `files ((foo.xxx) => -)`
                if f, ok := file.match.Paths[0].(*Flag); ok {
                        sys = isNone(f.name) || isNil(f.name)
                }*/
        } else if !sys && file.isSysFile() { sys = true }

        //fmt.Fprintf(stderr, "%v: %v %v %v %v\n", pos, t.entry.target, name, sys, t.project)

        // System files are not treated as missing nor collected
        // for further updating, just discard them immediately.
        if sys || isAbs || isRel || exists(file) { return }

        // relative to target directory
        var alt = stat(pos, name, "", gc.targetDir)
        if alt != nil { file = alt; return }

        // Check for bare non-system sub-paths:
        //   foo/bar/name.xxx
        // We search base name 'name.xxx' again:
        var s = filepath.Dir(name) // e.g: foo/bar

        // Search 'name.xxx' and check dir for
        // 'foo/bar' suffix. We use it if found.
        alt = t.project.matchFile(filepath.Base(name))
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
        if file = t.project.matchFile(name); file != nil && exists(file) {
                return // found existed file
        } else if isAbs = filepath.IsAbs(name); isAbs {
                file = stat(pos, name, "", "", nil)
        } else if isRel = isRelPath(name); isRel { // relative to targetDir
                file = stat(pos, name, "", gc.targetDir, nil)
        }

        // System files are not treated as missing nor collected
        // for further updating, just discard them immediately.
        if !sys && file != nil && file.match != nil && len(file.match.Paths) == 1 {
                // system files defined by `files ((foo.xxx) ⇒ -)`
                if f, ok := file.match.Paths[0].(*Flag); ok {
                        sys = isNone(f.name) || isNil(f.name)
                }
        }
        if!sys && gc.debug { fmt.Fprintf(stderr, "%v: %v: %v → %v (exists=%v, sys=%v, from %v)\n", pos, t.entry.target, gc.target, name, exists(file), sys, t.project) }
        if sys || exists(file) { return }

        // relative to target directory
        var alt = stat(pos, name, "", gc.targetDir)
        if alt != nil { file = alt; return }

        // Check for bare non-system sub-paths:
        //   foo/bar/name.xxx
        // We search base name 'name.xxx' again:
        var s = filepath.Dir(name) // e.g: foo/bar

        // Search 'name.xxx' and check dir for
        // 'foo/bar' suffix. We use it if found.
        alt = t.project.matchFile(filepath.Base(name))
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
        } else if !exists(file) && gc.discard {
                return
        } else if gc.files = append(gc.files, file); false && gc.touch {
                var tt = gc.targetInfo.ModTime()
                if file.info == nil && !file.isSysFile() {
                        var s string
                        if s, err = file.Strval(); err != nil { err = wrap(pos, err); return }
                        if file.info, err = os.Stat(s); err != nil { err = wrap(pos, err); return }
                        if false || gc.debug { fmt.Fprintf(stderr, "%s: '%v' info is nil (%s)\n", pos, file, file.fullname()) }
                }
                if file.info == nil {/* ... */} else
                if t := file.info.ModTime(); t.After(tt) {
                        if true || gc.debug { fmt.Fprintf(stderr, "%s: touch %v → %v (%v)\n", pos, gc.target, file, t) } //gc.targetFullName
                        t = launchTime //time.Now() // ...
                        err, tt = os.Chtimes(gc.targetFullName, t, t), t
                        if err != nil { err = wrap(pos, err); return }
                }
        }

        // Report missing files, but system files are not treated as missing.
        if gc.report {
                if file == nil {
                        fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` not found\n", gc.targetFullName, linum, colnum, t.project.name, name)
                } else if !exists(file) {
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
        filename, err = savedGrepFile.Strval()
        return
}

func (t *traversal) loadSavedGrepFile(pos Position, gc *grepctx) (okay bool, err error) {
        gc.savedGrepFileName, err = t.savedGrepFileName(pos, gc.targetFullName)
        if err != nil { err = wrap(pos, err); return }

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
                err = wrap(pos, err); return
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
        if err != nil { err = wrap(pos, err) } else { okay = true }
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

func (t *traversal) grepFiles(pos Position, gc *grepctx) (err error) {
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
                if targetName, err = v.Strval(); err != nil { return }
                if filepath.IsAbs(targetName) {
                        gc.targetFullName = targetName
                } else {
                        gc.targetFullName = filepath.Join(gc.targetDir, targetName)
                }
                if file := stat(pos, gc.targetFullName, "", ""); file == nil {
                        err = errorf(pos, "grep: '%s' not found", gc.targetFullName)
                        return
                } else { gc.targetInfo = file.info }
        }
        if err != nil { err = wrap(pos, err); return }
        if gc.targetInfo == nil { return }
        if gc.done == nil { gc.done = make(map[string]int) }
        if !filepath.IsAbs(gc.targetFullName) {
                err = errorf(pos, "grep: '%s' is not abs", gc.targetFullName)
                return
        } else { gc.done[gc.targetFullName] += 1 }
        if n, done := gc.done[gc.targetFullName]; done && n > 1 {
                if gc.debug { fmt.Fprintf(stderr, "%s: %v (done %v)\n", pos, gc.targetFullName, n) }
                return
        }

        if false { defer un(trace(t, targetName)) }

        if files, cached := grepcache[gc.targetFullName]; cached {
                if gc.debug { fmt.Fprintf(stderr, "%s: grepcache: %v → %v\n", pos, gc.targetFullName, files) }
                t.grepped = append(t.grepped, files...)
                if gc.recursive { for _, gc.target = range files {
                        if err = t.grepFiles(pos, gc); err != nil { break }
                }}
                return
        }
        defer func(restore []Value) {
                var touch = gc.greptouch
                gc.files = restore
                grepcache[gc.targetFullName] = touch.files
                if gc.debug { fmt.Fprintf(stderr, "%s: grepped: %s → %v (grepped=%v) (saved=%s)\n", pos, gc.target, touch.files, len(t.grepped), gc.savedGrepFile) }
                if gc.recursive && len(touch.files) > 0 {
                        for _, gc.target = range touch.files {
                                t.grepped = append(t.grepped, gc.target)
                                if err = t.grepFiles(pos, gc); err != nil { break }
                        }
                }
                if err == nil && gc.touch { err = touch.work(pos, gc) }
        } (gc.files)
        gc.files = nil

        var savedGrepFile *os.File
        var savedGrepFileLoaded bool
        savedGrepFileLoaded, err = t.loadSavedGrepFile(pos, gc)
        if false && gc.debug { fmt.Fprintf(stderr, "%s: saved: %v → %v (%v)\n", pos, gc.target, gc.files, err) }
        if err != nil || savedGrepFileLoaded { return } else
        if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
        }
        if optionSaveGrepSourceName {
                var perm = os.FileMode(0600)
                var data = []byte(gc.targetFullName)
                var name = gc.savedGrepFileName + ".src"
                if err = ioutil.WriteFile(name, data, perm); err != nil { return }
        }
        if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil { return }
        gc.save = bufio.NewWriter(savedGrepFile)
        defer func() { gc.save.Flush(); savedGrepFile.Close() } ()

        err = t.grepTargetFile(pos, gc)
        if false && gc.debug { fmt.Fprintf(stderr, "%s: grepped: %v → %v (%v)\n", pos, gc.target, gc.files, t.grepped) }
        if err != nil { if gc.discard { err = nil } else { err = wrap(pos, err) }}
        return
}

var stopgrep = 0

// grep-files - grep files from target, example usage:
//
//      (grep-files -x='\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrepFiles(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( gc grepctx ; optNoTraverse bool )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "c,discard",
                "c,discard-missing",
                "d,debug",
                "l,lang",
                "n,notraverse",
                "r,recursive",
                "s,sys",
                "s,system",
                "t,touch", // touch file if outdated
                "t,touch-outdated", // touch file if outdated
                "v,verbose",
                "x,regex",
        }, func(ru rune, v Value) {
                var s string
                switch ru {
                case 'c': if gc.discard   , err = trueVal(v,true); err != nil { return }
                case 'd': if gc.debug     , err = trueVal(v,true); err != nil { return }
                case 'v': if gc.verbose   , err = trueVal(v,true); err != nil { return }
                case 'r': if gc.recursive , err = trueVal(v,true); err != nil { return }
                case 't': if gc.touch     , err = trueVal(v,true); err != nil { return }
                case 'n': if optNoTraverse, err = trueVal(v,true); err != nil { return }
                case 's', 'x': if v != nil {
                        if s, err = v.Strval(); err != nil { return }
                        gc.rxs = append(gc.rxs, &greprex{s, ru=='s', nil})
                }
                case 'l': if v != nil {
                        if s, err = v.Strval(); err != nil { return } else
                        if info, ok := langInfos[s]; !ok || info == nil {
                                err = errorf(v.Position(), "unknown lang: %s", s)
                                return
                        } else {
                                for _, re := range info.rxs { gc.rxs = append(gc.rxs, &greprex{re, false, nil}) }
                                for _, re := range info.sys { gc.rxs = append(gc.rxs, &greprex{re, true, nil}) }
                        }
                }}
        }); err != nil { return }
        if len(gc.rxs) == 0 {
                err = errorf(pos, "no grep expressions")
                return
        }

        var files []Value
        if gc.verbose {
                if gc.debug { fmt.Fprintf(stderr, "%s: grep-files: %v %v %v\n", pos, t.def.target.value, gc.rxs, args) }
                fmt.Fprintf(stderr, "smart: Grep %v …", t.def.target.value)
                defer func(t time.Time) {
                        d := time.Now().Sub(t)
                        fmt.Fprintf(stderr, "… (%d files, %v)\n", len(files), d)
                } (time.Now())
        }

        if len(args) == 0 { args = append(args, t.def.target.value) }

        var grepped = t.grepped
        ForTarget: for _, target := range args {
                gc.target = target
                t.grepped = nil

                if err = t.grepFiles(pos, &gc); err != nil { err = wrap(pos, err); break }
                if !optNoTraverse && len(t.grepped) > 0 {
                        if false && gc.debug { fmt.Fprintf(stderr, "%v: %v: %v\n", t.project, pos, t.grepped) }
                        for _, val := range t.grepped {
                                if err = val.traverse(t); err != nil { err = wrap(pos, err); break ForTarget }
                        }
                }
                grepped = append(grepped, t.grepped...)
        }
        if false && t.project.name == "c++" { fmt.Fprintf(stderr, "%v: %v: %v %v\n", t.project, pos, args, grepped) }
        t.grepped = grepped

        if err != nil {} else if !optNoTraverse {
                if false && gc.debug { fmt.Fprintf(stderr, "%s: %v\n", pos, t.grepped) }
                t.def.grepped.value = &None{trivial{pos}}
                t.grepped = nil
        } else { result = MakeListOrScalar(pos, t.grepped) }

        //if stopgrep += 1; stopgrep > 99 { err = errorf(pos, "pause") }
        return
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
        err = errorf(pos, "unimplemented grep %v", args)
        return
}

func modifierTouch(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optMode os.FileMode
                optPath bool
                optVerbose bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "m,mode",
                "p,path",
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'm': if optMode   , err = permVal(v, 0600); err != nil { return }
                case 'p': if optPath   , err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }
        if len(args) == 0 { args = append(args, t.def.target.value) }
        for _, arg := range args {
                if err = touch(arg, uint32(optMode), optPath); err != nil {
                        err = wrap(pos, err)
                        break
                }
        }
        return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
func modifierCheck(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optBreak breakind // breaking with good results
                optVerbose bool // verbose more details
                optSilent bool // don't break on failures
                makeResult func(Position,bool) Value // returns results only if non-nil
                value Value = t.def.buffer.value
                values []Value
                pairs []*Pair
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "a,answer",
                "r,result",
                "v,verbose",
                "s,silent",
                "g,good",
        }, func(ru rune, v Value) {
                switch ru {
                case 'a': makeResult = MakeAnswer
                case 'r': makeResult = MakeBoolean
                case 'g': optBreak = breakDone
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                case 's': if optSilent , err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }
        for _, arg := range args {
                switch t := arg.(type) {
                case *Pair: pairs = append(pairs, t)
                default: err = errorf(pos, "unknown check '%v' (%T)", arg, arg)
                return
                }
        }
        if optSilent && makeResult == nil {
                makeResult = MakeBoolean
        }

        ForPairs: for _, p := range pairs {
                var key, str string
                if key, err = p.Key.Strval(); err != nil { return }
                switch key {
                case "status":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                err = break_with(pos, optBreak, "not an exec result (%T)", value)
                                return
                        } else { exeres.wg.Wait() }
                        if optVerbose { fmt.Printf("(status=%d)", exeres.Status) }

                        var num int64
                        if num, err = p.Value.Integer(); err != nil { return }
                        if res := exeres.Status == int(num); makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                err = break_with(pos, optBreak, "bad status (%v) (expects %v)", exeres.Status, p.Value)
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                err = break_with(pos, optBreak, "not an exec result (%T)", value)
                                return
                        } else { exeres.wg.Wait() }
                        if optVerbose { fmt.Printf("(status=%d)", exeres.Status) }

                        var v *bytes.Buffer
                        switch key {
                        case "stdout": v = exeres.Stdout.Buf
                        case "stderr": v = exeres.Stderr.Buf
                        default: unreachable()
                        }

                        if v == nil {
                                err = break_with(pos, optBreak, "bad %s (expects %v)", key, p.Value)
                                break ForPairs
                        }
                        if str, err = p.Value.Strval(); err != nil { 
                                return
                        } else if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                err = break_with(pos, optBreak, "bad %s (%v) (expects %v)", key, v, p.Value)
                                break ForPairs
                        }
                case "file", "dir":
                        var file *File
                        var project = t.project
                        if str, err = p.Value.Strval(); err != nil { return }
                        if file := project.matchFile(str); !exists(file) {
                                err = break_with(pos, optBreak, "`%v` no such file or directory", p.Value)
                                break ForPairs
                        }
                        switch key {
                        case "file":
                                if res := file.info.Mode().IsRegular(); makeResult != nil {
                                        values = append(values, makeResult(pos, res))
                                } else if !res {
                                        err = break_with(pos, optBreak, "`%v` is not a regular file", p.Value)
                                        break ForPairs
                                }
                        case "dir":
                                if res := file.info.Mode().IsDir(); makeResult != nil {
                                        values = append(values, makeResult(pos, res))
                                } else if !res {
                                        err = break_with(pos, optBreak, "`%v` is not a directory", p.Value)
                                        break ForPairs
                                }
                        default: unreachable()
                        }
                case "var":
                        var g, ok = p.Value.(*Group)
                        if !ok {
                                err = break_with(pos, optBreak, "`%v` is not a group value", p.Value)
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
                                                        err = break_with(pos, optBreak, "`%v` != `%v`", p.Key, p.Value)
                                                        break ForPairs
                                                }
                                        } else if makeResult != nil {
                                                values = append(values, makeResult(pos, false))
                                        } else {
                                                err = break_with(pos, optBreak, "`%v` is not defined", k)
                                                break ForPairs
                                        }
                                default:
                                        err = break_with(pos, optBreak, "`%v` unsupported checks", elem)
                                        break ForPairs
                                }
                        }
                default:
                        err = errorf(pos, "unknown check '%v'", p.Key)
                        break ForPairs
                }
        }
        if err == nil && values != nil {
                result = MakeListOrScalar(pos, values)
        }
        return
}

type copyopts struct {
        path bool
        mode os.FileMode
        head Value
        foot Value
}

func copyRegular(pos Position, src, dst string, opts *copyopts) (err error) {
        var srcFile, dstFile *os.File
        if srcFile, err = os.Open(src); err != nil { return }

        // sys default file mode is 0666
        if opts.path { // Make path (mkdir -p)
                if p := filepath.Dir(dst); p != "." && p != "/" {
                        err = os.MkdirAll(p, os.FileMode(0755))
                        if err != nil { return }
                }
        }

        if opts.mode == 0 { opts.mode = os.FileMode(0640) }

        dstFile, err = os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, opts.mode)
        if err != nil { srcFile.Close(); return }

        srcBuf := bufio.NewReader(srcFile)
        dstBuf := bufio.NewWriter(dstFile)
        defer func() {
                dstFile.Close()
                srcFile.Close()
                
                var file = stat(pos, dst, "", "")
                context.globe.stamp(dst, file.info.ModTime())
        } ()

        if opts.head != nil {
                var s string
                if s, err = opts.head.Strval(); err != nil { return }
                if s != "" {
                        dstBuf.WriteString(s)
                }
        }
        if _, err = io.Copy(dstBuf, srcBuf); err == nil {
                if opts.foot != nil {
                        var s string
                        s, err = opts.foot.Strval()
                        if err == nil && s != "" {
                                dstBuf.WriteString(s)
                        }
                }
                dstBuf.Flush() // flush content
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

// (copy-file -vp)
// (copy-file -p,filename)
// (copy-file -p,filename,source)
func modifierCopyFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optPath bool
                optRecursive bool
                optVerbose bool
                optSilent bool
                optOverride bool
                optMode os.FileMode
                optHead Value
                optFoot Value
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "p,path", // prepare paths for files
                "r,recursive",
                "v,verbose",
                "s,silent", // optSilent
                "s,silent-existed", // optSilent
                "o,override",
                "m,mode",
                "h,head", // insert header content
                "f,foot", // insert footer content
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': if optPath     , err = trueVal(v, true); err != nil { return }
                case 'r': if optRecursive, err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose  , err = trueVal(v, true); err != nil { return }
                case 's': if optSilent   , err = trueVal(v, true); err != nil { return }
                case 'o': if optOverride , err = trueVal(v, true); err != nil { return }
                case 'm': if optMode     , err = permVal(v, 0600); err != nil { return }
                case 'h': if !(isNil(v) || isNone(v)) { optHead = v }
                case 'f': if !(isNil(v) || isNone(v)) { optFoot = v }
                }
        }); err != nil { return }
        
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
                if filename, err = t.Strval(); err != nil {
                        return
                } else if t.info != nil {
                        filetime = t.info.ModTime()
                }
        default:
                if filename, err = target.Strval(); err != nil {
                        return
                } else if file := project.matchFile(filename); file != nil {
                        if filename, err = file.Strval(); err != nil {
                                return
                        } else {
                                target = file
                        }
                        if file.info != nil {
                                filetime = file.info.ModTime()
                        }
                }
        }
        switch t := source.(type) {
        case *File:
                if srcname, err = t.Strval(); err != nil {
                        return
                } else if t.info != nil {
                        srctime = t.info.ModTime()
                }
        default:
                if srcname, err = source.Strval(); err != nil {
                        return
                } else if file := project.matchFile(srcname); file != nil {
                        if srcname, err = file.Strval(); err != nil {
                                return
                        } else {
                                source = file
                        }
                        if file.info != nil {
                                srctime = file.info.ModTime()
                        }
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
                if optOverride {
                        if optVerbose { fmt.Fprintf(stderr, "smart: Override %v …", target) }
                } else {
                        if optVerbose { fmt.Fprintf(stderr, "smart: Copying %v …… already existed!\n", target) }
                        if !optSilent { err = errorf(pos, "file already existed (%s)", target) }
                        return
                }
        } else if optVerbose { fmt.Fprintf(stderr, "smart: Copying %v …", target) }
        
        var copyOpts = &copyopts{ optPath, optMode, optHead, optFoot }
        var fi os.FileInfo
        if fi, err = os.Stat(srcname); err != nil {
                err = wrap(pos, err)
        } else if !fi.IsDir() {
                if optMode == 0 { optMode = fi.Mode() }
                if err = copyFile(pos, fi, srcname, filename, copyOpts); err != nil {
                        err = wrap(pos, err)
                }
        } else if optRecursive {
                if err = copyDir(pos, srcname, filename, copyOpts); err != nil {
                        err = wrap(pos, err)
                }
        } else {
                err = errorf(pos, "`%v` is a directory (use -r to solve it)", source)
        }

        if optVerbose {
                if err != nil {
                        fmt.Fprintf(stderr, "… error\n")
                } else {
                        fmt.Fprintf(stderr, "… ok\n")
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
        if filename, err = t.def.target.value.Strval(); err != nil {
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
                s := fmt.Sprintf("file %s not generated", t.def.target.value)
                err = &breaker{ pos:pos, what:breakFail, message:s }
        }
        return
}

func modifierReadFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optDebug bool
                optVerbose bool
                optHead Value
                optFoot Value
                filename string
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "d,debug",
                "v,verbose",
                "h,head", // insert header content
                "f,foot", // insert footer content
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': if optDebug  , err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                case 'h': if !(isNil(v) || isNone(v)) { optHead = v }
                case 'f': if !(isNil(v) || isNone(v)) { optFoot = v }
                }
        }); err != nil { return }

        if n := len(args); n > 1 {
                err = errorf(pos, "too many files: %v", args)
                return
        } else if n == 1 {
                if filename, err = args[0].Strval(); err != nil { return }
        } else if filename, err = t.def.target.value.Strval(); err != nil {
                return
        }

        if optDebug {
                fmt.Fprintf(stderr, "%s:debug: read-file: %v\n", pos, filename)
        }

        var bytes []byte
        if bytes, err = ioutil.ReadFile(filename); err == nil {
                var s, v string
                if optHead != nil {
                        if v, err = optHead.Strval(); err == nil { s = v } else {
                                err = wrap(pos, err); return
                        }
                }
                s += string(bytes)
                if optFoot != nil {
                        if v, err = optFoot.Strval(); err == nil { s += v } else {
                                err = wrap(pos, err); return
                        }
                }
                t.def.buffer.value = &String{trivial{pos},s}
        } else {
                err = &breaker{pos:pos, what:breakFail, message:err.Error()}
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

func modifierUpdateFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optPath bool
                optDebug bool
                optVerbose bool
                optMode = os.FileMode(0640) // sys default 0666
                filename, content string
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "d,debug",
                "p,path",
                "v,verbose",
                "m,mode",
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': if optDebug  , err = trueVal(v, true); err != nil { return }
                case 'p': if optPath   , err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                case 'm': if optMode   , err = permVal(v, 0600); err != nil { return }
                }
        }); err != nil { return }

        var target Value
        if len(args) > 0 { target = args[0] } else { target = t.def.target.value }
        if len(args) > 1 { if optMode, err = permVal(args[1], 0600); err != nil { return }}

        // Get target filename
        switch p := target.(type) {
        case *File, *Path:
                if filename, err = p.Strval(); err != nil { return }
        default:
                if filename, err = target.Strval(); err != nil { return } else
                if file := t.project.matchFile(filename); file != nil {
                        if filename, err = file.Strval(); err != nil { return } else {
                                target = file
                        }
                }
        }

        if optDebug {
                fmt.Fprintf(stderr, "%s:debug: update-file: %v (%v) (%v, %v)\n", pos, target, filename, t.project, cloctx)
        }

        if optPath { // Make path (mkdir -p)
                if p := filepath.Dir(filename); p != "." && p != "/" {
                        err = os.MkdirAll(p, os.FileMode(0755))
                        if err != nil { return }
                }
        }

        // Check existed file content checksum
        if content, err = t.def.buffer.value.Strval(); err != nil { return }
        if content == "" && (optVerbose || optDebug) { fmt.Fprintf(stderr, "%s: empty content\n", pos) }
        if optVerbose { fmt.Fprintf(stderr, "smart: Checking %v …", trimPromptString(target.String())) }
        if same, e := crc64CheckFileModeContent(filename, []byte(content), optMode); e != nil {
                if false { // discard error (e.g.: no such file or directory)
                        fmt.Fprintf(stderr, "… (error: %s)\n", e)
                        err = wrap(pos, e)
                        return
                }
        } else if same {
                if optVerbose { fmt.Fprintf(stderr, "… Good\n") }
                t.removeCallerUpdated(target) // remove timestamp updated
                result = stat(pos, filename, "", "")
                return
        } else if optVerbose {
                fmt.Fprintf(stderr, "… Outdated (%s)\n", filename)
        }

        if optVerbose {
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
        f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, optMode)
        if err == nil && f != nil {
                defer func() {
                        if f.Close(); err != nil {
                                os.Remove(filename)
                                return
                        }
                        var file = stat(pos, filename, "", "")
                        if  file == nil {
                                err = errorf(pos, "invalid file '%s'", filename)
                        } else {
                                file.stamp(t)
                                result = file // resulting the updated file
                        }
                } ()
                if _, err = f.WriteString(content); err == nil {
                        if optVerbose { fmt.Fprintf(stderr, "… (ok)\n") }
                } else {
                        if optVerbose { fmt.Fprintf(stderr, "… (%s)\n", err) }
                }
        } else {
                if optVerbose { fmt.Fprintf(stderr, "… (%s)\n", err) }
                s := fmt.Sprintf("file %s not updated", t.def.target.value)
                err = &breaker{ pos:pos, what:breakFail, message:s }
        }
        return
}

func modifierWait(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optVerbose bool
                optStdout bool
                optStderr bool
                optStatus bool
                optExecRes bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "o,stdout",
                "e,stderr",
                "s,status",
                "x,exec",
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'o': if optStdout , err = trueVal(v, true); err != nil { return }
                case 'e': if optStderr , err = trueVal(v, true); err != nil { return }
                case 's': if optStatus , err = trueVal(v, true); err != nil { return }
                case 'x': if optExecRes, err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        if optVerbose {
                fmt.Fprintf(stderr, "smart: Wait (%v) …\n", t.def.target.value)
        }

        err = t.wait(pos)

        if optVerbose {
                var s = "Done"
                if err != nil { s = "Fail" }
                fmt.Fprintf(stderr, "smart: %s (%v), updated=%v\n", s, t.def.target.value, t.updated)
        }

        if (optStdout || optStderr || optStatus || optExecRes) && !isNil(t.def.buffer.value) {
                if exeres, ok := t.def.buffer.value.(*ExecResult); ok {
                        exeres.wg.Wait()
                        var a []Value
                        if optStdout {
                                var s string
                                if b := exeres.Stdout.Buf; b != nil { s = b.String() }
                                a = append(a, &String{trivial{pos},s})
                        }
                        if optStderr {
                                var s string
                                if b := exeres.Stderr.Buf; b != nil { s = b.String() }
                                a = append(a, &String{trivial{pos},s})
                        }
                        if optStatus {
                                a = append(a, &Int{integer{trivial{pos},int64(exeres.Status)}})
                        }
                        result = MakeListOrScalar(pos, a)
                }
        }
        return
}

func predict(pos Position, t *traversal, args... Value) (result bool, breakScope breaksco, message string, err error) {
        var target string
        if target, err = t.def.target.value.Strval(); err != nil {
                err = wrap(pos, err)
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
        var optVerbose, optAnd, verbose0 bool
        defer func() {
                if optVerbose {
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
                var va = merge(arg)
                if va, err = parseFlags(va, []string{
                        "a,and",
                        "g,group",  // breakGroup
                        "t,trave",  // breakTrave
                        "t,target", // breakTrave
                        "m,message", // message
                        "m,msg",     // message
                        "v,verbose",
                }, func(ru rune, v Value) {
                        switch ru {
                        case 'a': if optAnd, err = trueVal(v, false); err != nil { return }
                        case 'g': breakScope = breakGroup
                        case 't': breakScope = breakTrave
                        case 'm': message, err = v.Strval()
                        case 'v': if optVerbose, err = trueVal(v, optVerbose); err != nil { return }
                                if optVerbose && !verbose0 {
                                        fmt.Fprintf(stderr, "smart: Checking %v …", target)
                                        verbose0 = true
                                }
                        }
                }); err != nil { return }
                //if optAnd && !result { continue }
                if !optAnd || (optAnd && result) { for i, a := range va {
                        if g, ok := a.(*Group); ok && len(g.Elems) > 0 {
                                var name string
                                if name, err = g.Elems[0].Strval(); err != nil {
                                        err = wrap(a.Position(), err)
                                        return
                                }
                                if m, ok := modifiers[name]; ok {
                                        var res Value
                                        res, err = m(a.Position(), t, g.Elems[1:]...)
                                        if err != nil { err = wrap(a.Position(), err); return }
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

                        if optAnd {
                                result = result && t
                                optAnd = false // reset -and flag
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
        if res, sco, msg, err = predict(pos, t, args...); !res && err == nil {
                if msg == "" {
                        msg = fmt.Sprintf("%v", args)
                } else {
                        msg = fmt.Sprintf("%v: %v", args, msg)
                }
                err = &breaker{ pos:pos, what:breakFail, message:msg, scope:sco }
        }
        return
}

func modifierCond(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( res bool; sco breaksco; msg string )
        if res, sco, msg, err = predict(pos, t, args...); !res && err == nil {
                err = &breaker{ pos:pos, what:breakDone, message:msg, scope:sco }
        }
        return
}

func modifierCase(pos Position, t *traversal, args... Value) (result Value, err error) {
        var ( res bool; sco breaksco; msg string )
        if res, sco, msg, err = predict(pos, t, args...); err == nil {
                var what = breakNext // next case
                if res { what = breakCase } // select case
                err = &breaker{ pos:pos, what:what, message:msg, scope:sco }
        }
        return
}

func modifierDirty(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optChecksum bool
                optDebug bool
                optVerbose bool
                optSilent bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "c,checksum",
                "d,debug",
                "v,verbose",
                "s,silent",
        }, func(ru rune, v Value) {
                switch ru {
                case 'c': if optChecksum, err = trueVal(v, true); err != nil { return }
                case 'd': if optDebug   , err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose , err = trueVal(v, true); err != nil { return }
                case 's': if optSilent  , err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        err = t.wait(pos) // Wait for prerequisites
        if err != nil { err = wrap(pos, err); return }

        var reason string
        var dirty bool
        if dirty = len(t.breakers) > 0; dirty {
                reason = fmt.Sprintf("dirty (%v breakers)", len(t.breakers))
        } else if dirty = !exists(t.def.target.value); dirty {
                reason = "dirty: target not exists"
        } else if dirty = len(t.updated) > 0; dirty {
                reason = fmt.Sprintf("dirty (%v updated)", len(t.updated))
        } else if dirty, err = t.isRecipesDirty(); err != nil {
                err = wrap(pos, err); return
        } else if dirty {
                reason = "dirty: recipes changed"
        } else if optChecksum && !(isNil(t.def.depend0.value) || isNone(t.def.depend0.value)) {
                var file1, file2 string
                if file1, err = t.def.target.value.Strval(); err != nil {
                        err = wrap(pos, err); return
                }
                if file2, err = t.def.depend0.value.Strval(); err != nil {
                        err = wrap(pos, err); return
                }
                if same, e := crc64CompareFileChecksum(file1, file2); e != nil {
                        err = wrap(pos, e); return
                } else if same {
                        reason = "Good"
                } else {
                        reason = "dirty: content changed"
                        dirty = true
                }
        } else {
                reason = "Good"
        }

        if optDebug {
                var e = exists(t.def.target.value)
                var a = typeof(t.def.target.value)
                var s, _ = t.def.target.value.Strval()
                fmt.Fprintf(stderr, "%s: %s %s (exists=%v, dirty=%v, updated=%v)\n", pos, a, s, e, dirty, t.updated)
        }

        if optVerbose {
                var s string
                if len(t.updated) > 0 { //s = fmt.Sprintf(", %v", t.updated)
                        s = ", ["
                        for i, v := range t.updated {
                                if i > 0 { s += " " }
                                if len(s) > maxPromptStr {
                                        s += "…"
                                        break
                                } else { s += v.String() }
                        }
                        s += "]"
                } else if dirty { s = ", "+strings.TrimPrefix(reason, "dirty: ") }
                fmt.Fprintf(stderr, "smart: Checking dirty %s (%v%s)\n", t.def.target.value, dirty, s)
        }

        if optionTraceTraversal {
                var v = t.def.target.value
                t.tracef("dirty: %v (updated=%v, exists=%v, target=%v)", dirty, len(t.updated), exists(v), v)
                if len(t.updated) > 0 { t.tracef("dirty: updated=%v", t.updated) }
        }

        if optSilent { reason = "" }
        result = &prediction{boolean{trivial{pos},dirty},reason}
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
        result = &prediction{boolean{trivial{pos},!loop},s}
        return
}

func modifierTarget1stVisit(pos Position, t *traversal, args... Value) (result Value, err error) {
        var optSilent bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "s,silent",
        }, func(ru rune, v Value) {
                switch ru {
                case 's': if optSilent, err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

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
        ;      if optSilent {
        } else if num == 0  { //s = "zero"
        } else { s = fmt.Sprintf("%v visits", num+1)
        }

        result = &prediction{boolean{trivial{pos},num==0},s}
        return
}

func modifierTargetMaxVisit(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optClosure bool
                optDump bool
                optSilent bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "c,closure",
                "d,debug-trace", // debug-trace
                "d,debug", // debug-trace
                "d,dump", // debug-trace
                "s,silent", // for reason
        }, func(ru rune, v Value) {
                switch ru {
                case 'c': if optClosure, err = trueVal(v, true); err != nil { return }
                case 'd': if optDump   , err = trueVal(v, true); err != nil { return }
                case 's': if optSilent , err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        var nth int64
        for _, a := range args {
                if nth, err = a.Integer(); err != nil {
                        err = wrap(pos, err)
                        return
                } else if nth <= 0 {
                        err = errorf(pos, "needs positive number (%v, %s)", a, typeof(a))
                        return
                }
        }

        var num int64
        var head bool = true
        for caller := t.caller; caller != nil; caller= caller.caller {
                if false {
                        if optClosure && caller.closure == t.closure { continue }
                        var same = t.def.target.value == caller.def.target.value
                        if !same && false {
                                cmp := t.def.target.value.cmp(caller.def.target.value)
                                same = (cmp == cmpEqual)
                        }
                        if same { num += 1 }
                } else if n, ok := caller.visited[t.def.target.value]; ok && n > 0 {
                        num += int64(n)
                }
                if optDump && num > 0 {
                        if head { head = false
                                fmt.Fprintf(stderr, "  %s: nth(%d)\n", pos, nth)
                        }
                        var pos = caller.program.position
                        fmt.Fprintf(stderr, "    %s: %v\n", pos, caller.def.target)
                }
        }

        var s string
        ;      if optSilent {
        } else if num == 0  { //s = "nth: zero"
        } else if num < nth { //s = "nth"
        } else { s = fmt.Sprintf("%d visits", num+1) }

        result = &prediction{boolean{trivial{pos},num<nth},s}
        return
}

var onceMutex sync.Mutex
var onceCache = make(map[HashBytes]int,64)
func modifierOnce(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optDebug bool
                optVerbose bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "d,debug",
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': if optDebug   , err = trueVal(v, true); err != nil { return }
                case 'v': if optVerbose , err = trueVal(v, true); err != nil { return }
                }
        }); err != nil { return }

        var s string
        var h = sha256.New()
        if s, err = t.entry.target.Strval(); err != nil { return } else {
                fmt.Fprintf(h, "%s", s)
        }
        if s, err = t.def.target.value.Strval(); err != nil { return } else {
                fmt.Fprintf(h, "%s", s)
        }
        for _, a := range args {
                if s, err = a.Strval(); err != nil { return } else {
                        fmt.Fprintf(h, "%s", s)
                }
        }

        var sum HashBytes
        copy(sum[:], h.Sum(nil))

        onceMutex.Lock(); defer onceMutex.Unlock()
        onceCache[sum] += 1

        var num = onceCache[sum]

        if optDebug {
                fmt.Fprintf(stderr, "%s: %v (once: num=%d)\n", pos, t.def.target.value, num)
        } else if optVerbose {
                fmt.Fprintf(stderr, "once: %v (num=%d)\n", t.def.target.value, num)
        }

        if num > 1 {
                msg := fmt.Sprintf("once (num=%d)", num)
                err = &breaker{ pos:pos, what:breakDone, message:msg }
        }
        return
}
