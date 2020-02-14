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
                // Save all breakers for (dirty) and interpreters.
                t.breakers = append(t.breakers, e)
                if e.what == breakCase {
                        continue // case selected
                } else if e.what == breakDone && e.scope == breakGroup {
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
                `select`:       modifierSelect,

                //`args`:       modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments
                `set`:          modifierSetVar,

                `closure`:      modifierClosure,

                `cd`:           modifierCD,
                `mkdir`:        modifierMkdir,
                `path`:         modifierPath,

                `sudo`:         modifierSudo,

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
        if c := t.caller; c != nil {
                t.project, t.closure = c.project, c.closure
        }

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
                case 'd': optDump = trueVal(v, false)
                case 'v': optVerbose = trueVal(v, false)
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
                case 'p': optPath = trueVal(v, true)
                case 'e': optPrintEnter = trueVal(v, true)
                case 'l': optPrintLeave = trueVal(v, true)
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
                case 'v': optVerbose = trueVal(v, true)
                case 'm': if v != nil {
                        var num int64
                        if num, err = v.Integer(); err != nil { return } else {
                                optMode = os.FileMode(num & 0777)
                        }
                }}
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
        }
}

type grepCacheFiles struct {
        file *File
        list []*File
}
var grepCacheFilebase = make(map[*filebase]*grepCacheFiles)

// parseGrepOption - parses grep options
// 
//   (regexp=(sys='...' '...') $^)
//   (regexp='...' sys='...' $^)
//   (regexp='...' s='~' $^)
func _parseGrepOption(pos Position, t *traversal, optGrep Value) (result []Value, err error) {
        var (
                optReportMissing bool = true
                optDiscardMissing bool
                optRecursive bool
                optLang string
                rxs []*greprex
                vals []Value
                top []Value
                store Value
                group *Group
        )
        if g, ok := optGrep.(*Group); !ok {
                err = errorf(pos, "`%T` non-group grep option", optGrep)
                return
        } else {
                group = g
        }

        if vals, err = parseFlags(group.Elems, []string{
                "e,report",
                "i,ignore",
                "r,recursive",
                "l,lang", // TODO: -lang=c|c++|go|java|...
                "c,clang", // TODO: same as -lang=c
                "x,regexp",
                "x,exp",
        }, func(ru rune, v Value) {
                switch ru {
                case 'e':
                        if v == nil {
                                optReportMissing = true
                        } else {
                                optReportMissing, _ = v.True()
                        }
                case 'i':
                        if v == nil {
                                optDiscardMissing = true
                        } else {
                                optDiscardMissing, _ = v.True()
                        }
                case 'r':
                        if v == nil {
                                optRecursive = true
                        } else {
                                optRecursive, _ = v.True()
                        }
                case 'x':
                        // regexp=(top=(foo,bar,baz) sys='^\s*#\s*include\s*<(.*)>' '^\s*#\s*include\s*"(.*)"')
                        var s string
                        switch v := v.(type) {
                        case *Group:
                                for _, v := range v.Elems {
                                        if p, ok := v.(*Pair); ok {
                                                if s, err = p.Key.Strval(); err != nil { return }
                                                switch s {
                                                case "sys":
                                                        if s, err = p.Value.Strval(); err != nil { return }
                                                        rxs = append(rxs, &greprex{s, true, nil})
                                                case "top":
                                                        if g, ok := p.Value.(*Group); ok {
                                                                top = merge(g.Elems...)
                                                        } else {
                                                                top = merge(v)
                                                        }
                                                default:
                                                        err = errorf(pos, "`%s` unknown regexp (%v)", s, p.Value)
                                                        return
                                                }
                                        } else {
                                                if s, err = v.Strval(); err != nil { return }
                                                rxs = append(rxs, &greprex{s, false, nil})
                                        }
                                }
                        default:
                                if s, err = v.Strval(); err != nil { return }
                                rxs = append(rxs, &greprex{s, false, nil})
                        }
                case 'l':
                        if v != nil {
                                optLang, _ = v.Strval()
                        } else {
                                optLang = ""
                        }
                        if info, ok := langInfos[optLang]; ok {
                                for _, s := range info.sys {
                                        rxs = append(rxs, &greprex{s, true, nil})
                                }
                                for _, s := range info.rxs {
                                        rxs = append(rxs, &greprex{s, false, nil})
                                }
                        }
                case 's':
                        store = v
                }
        }); err != nil { return }

        var tops []string
        for _, v := range top {
                var s string
                if s, err = v.Strval(); err != nil { return }
                tops = append(tops, s)
        }

        var unique = make(map[*filebase]int) // to remove duplications 
        var grep func(vals []Value)
        grep = func(vals []Value) {
                for _, val := range vals {
                        switch v := val.(type) {
                        case *None: continue
                        case *File:
                                var files []Value
                                if info, ok := grepCacheFilebase[v.filebase]; ok {
                                        for _, file := range info.list {
                                                if _, ok = unique[file.filebase]; !ok {
                                                        files = append(files, file)
                                                }
                                                //unique[file.filebase] += 1
                                        }
                                } else if exists(v) {
                                        var list []Value
                                        list, err = t.project.grepFiles(pos, val, tops, rxs, optReportMissing, optDiscardMissing)
                                        if err != nil { return }
                                        info = &grepCacheFiles{ file:v }
                                        grepCacheFilebase[v.filebase] = info
                                        for _, v := range list {
                                                var file = v.(*File)
                                                info.list = append(info.list, file)
                                                if _, ok := unique[file.filebase]; !ok {
                                                        files = append(files, file)
                                                }
                                                //unique[file.filebase] += 1
                                        }
                                }
                                unique[v.filebase] += 1
                                result = append(result, v)
                                if files != nil {
                                        if optRecursive { grep(files) } else {
                                                result = append(result, files...)
                                        }
                                }
                        default:
                                err = errorf(pos, "'%v' cant grep this type", t)
                                return
                        }
                }
        }

        grep(vals) ; unique = nil
        if err != nil { return }

        var s string
        if store != nil {
                if s, err = store.Strval(); err != nil { return }
        }
        if s == "" { s = "~" } // append to $~ by default
        if def := t.program.scope.FindDef(s); def != nil {
                def.append(result...)
                result = nil
        } else {
                err = errorf(pos, "`%s` no such Def", s)
        }
        return
}

type grepctx struct {
        debug, verbose bool
        discard, report bool // discard or report missing greps
        rxs []*greprex
        targetDir string // see splitTargetFileName
        targetFileName string // see splitTargetFileName
        savedGrepFileName string
        save *bufio.Writer
}
type greprex struct{ string ; bool ; *regexp.Regexp }
func (g *greprex) String() string { return g.string }

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

func (p *Project) grepFiles(pos Position, target Value, tops []string, rxs []*greprex, report, discard bool) (result []Value, err error) {
        var targetDir, targetName, targetFileName string
        switch t := target.(type) {
        case *File:
                targetName = t.name
                targetFileName = t.fullname()
                targetDir = filepath.Dir(targetFileName)
        default:
                targetDir = p.absPath
                if targetName, err = t.Strval(); err != nil { return }
                if filepath.IsAbs(targetName) {
                        targetFileName = targetName
                } else {
                        targetFileName = filepath.Join(targetDir, targetName)
                }
        }
        if !filepath.IsAbs(targetFileName) {
                unreachable(targetFileName, " is not abs")
        }

        var cached bool
        if result, cached = grepcache[targetFileName]; cached {
                return
        } else {
                defer func() { grepcache[targetFileName] = result } ()
        }

        var isTargetFile = func(file *File) (res bool) {
                if target == file {
                        res = true
                } else if s, _ := file.Strval(); s == targetFileName {
                        res = true
                } else if t, ok := target.(*File); ok && t.name == file.name {
                        res = true
                }
                return
        }

        var searchName = func(sys bool, linum, colnum int, name string) (file *File) {
                var isAbs, isRel bool
                if isAbs = filepath.IsAbs(name); isAbs {
                        file = stat(pos, name, "", "", nil)
                } else if isRel = isRelPath(name); isRel { // relative to target dir
                        file = stat(pos, name, "", targetDir, nil)
                        if !exists(file) {
                                var f = p.matchFile(name)
                                if f != nil {
                                        file = f
                                }
                        }
                } else if file = p.matchFile(name); file == nil {
                        // file not found
                }

                if !sys && file != nil && file.match != nil && len(file.match.Paths) == 1 {
                        // mark system files defined by `files ((foo.xxx) => -)`
                        if f, ok := file.match.Paths[0].(*Flag); ok {
                                if _, ok = f.name.(*None); ok {
                                        sys = true
                                }
                        }
                }

                // System files are not treated as missing nor collected
                // for further updating, just discard them immediately.
                if sys { return }

                if !isAbs && !isRel && (file == nil || !exists(file)) {
                        // relative to target directory
                        var alt = stat(pos, name, "", targetDir)
                        if alt != nil { file = alt }

                        // Check for bare non-system sub-paths:
                        //   foo/bar/name.xxx
                        // We search base name 'name.xxx' again:
                        if alt == nil {// got sub-name like 'foo/bar'
                                var s = filepath.Dir(name)

                                /* FIXME: try all names
                                s, i := file.name, strings.LastIndex(file.name, PathSep)
                                for ; s != "" && i > 0; {
                                        name = filepath.Join(s[i+1:], name)
                                        s = s[:i] // slice out the prefix 
                                        // matchFile(name)
                                        i = strings.LastIndex(s, PathSep)
                                }
                                */

                                // Search 'name.xxx' and check dir for
                                // 'foo/bar' suffix. We use it if found.
                                alt = p.matchFile(filepath.Base(name))
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
                        }
                }
                
                if file == nil {
                        // The 'name' is not matching the files database.
                        if discard { return }
                        // FIXME: missing-file error
                } else if isTargetFile(file) {
                        return
                } else if !exists(file) && discard {
                        return
                } else {
                        result = append(result, file)
                }

                // Report missing files, but system files are not treated
                // as missing.
                if report {
                        if file == nil {
                                fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` not found\n", targetFileName, linum, colnum, p.name, name)
                        } else if !exists(file) {
                                fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` file not existed\n", targetFileName, linum, colnum, p.name, name)
                        }
                }
                return
        }

        var targetOSFile *os.File
        var savedGrepOSFile *os.File
        var savedGrepFileName string

        var nameHash = sha256.New() //[sha256.Size]byte
        fmt.Fprintf(nameHash, "%s", targetFileName)

        // Make names like .grep/00/da/bef0cc203d80fa25e0e2d3760518ee1b16bd641f99b9059468cfbbe8f096
        var nameSum = nameHash.Sum(nil)
        var savedGrepFile = p.matchTempFile(pos, filepath.Join(".grep",
                fmt.Sprintf("%x", nameSum[ :1]),
                fmt.Sprintf("%x", nameSum[1:2]),
                fmt.Sprintf("%x", nameSum[2: ]),
        ))
        savedGrepFileName, _ = savedGrepFile.Strval()

        if file1 := stat(pos, savedGrepFileName, "", ""); file1 != nil {
                if file2 := stat(pos, targetFileName, "", ""); file2 != nil {
                        if file2.info.ModTime().After(file1.info.ModTime()) {
                                goto GrepTargetFile
                        }
                }
                var e error
                if savedGrepOSFile, e = os.Open(savedGrepFileName); e == nil {
                        defer savedGrepOSFile.Close()
                        scanner := bufio.NewScanner(savedGrepOSFile)
                        scanner.Split(bufio.ScanLines)
                        for scanner.Scan() {
                                var s = scanner.Text()
                                var sys, linum, colnum int
                                var name string
                                if n, e := fmt.Sscanf(s, "%d %d %d %s", &sys, &linum, &colnum, &name); e == nil && n == 4 {
                                        file := searchName(sys == 1, linum, colnum, name)
                                        if file != nil && isTargetFile(file) {
                                                continue //unreachable("same as target: ", file)
                                        }
                                }
                        }
                        file1.info, _ = savedGrepOSFile.Stat()
                        return
                }
        }

GrepTargetFile:
        if targetOSFile, err = os.Open(targetFileName); err != nil {
                if discard { err = nil }
                return
        } else {
                defer func() { err = targetOSFile.Close() } ()
        }
        if dir := filepath.Dir(savedGrepFileName); dir != "." && dir != ".." {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                        return
                }
        }
        if savedGrepOSFile, err = os.Create(savedGrepFileName); err != nil {
                return
        } else {
                defer savedGrepOSFile.Close()
        }

        if optionSaveGrepSourceName {
                var perm = os.FileMode(0600)
                var data = []byte(targetFileName)
                var name = savedGrepFileName + ".src"
                if err = ioutil.WriteFile(name, data, perm); err != nil {
                        return
                }
        }

        var save = bufio.NewWriter(savedGrepOSFile)
        defer save.Flush()

        for _, x := range rxs {
                x.Regexp, err = regexp.Compile(x.string)
                if err != nil { return }
        }

        var linum int
        scanner := bufio.NewScanner(targetOSFile)
        scanner.Split(bufio.ScanLines)
ForScan:
        for scanner.Scan() {
                linum += 1
                var s = scanner.Text()
                for _, x := range rxs {
                        if sm := x.FindStringSubmatch(s); len(sm) > 1 && sm[1] != "" {
                                var name = sm[1]
                                var colnum = strings.Index(s, name) //strings.IndexFunc(s, isNotSpace)
                                var sys = x.bool // System files defined by `sys=xxx` arguments
         
                                var d = 0 ; if sys { d = 1 }
                                fmt.Fprintf(save, "%d %d %d %s\n", d, linum, colnum, name)

                                file := searchName(sys, linum, colnum, name)
                                if file != nil {
                                        if isTargetFile(file) {
                                                continue //unreachable("same as target: %s", file)
                                        }
                                        continue ForScan
                                }
                        }
                }
        }
        if enable_assertions {
                for _, v := range result {
                        if v == nil { unreachable() }
                        if f, ok := v.(*File); ok && f == nil {
                                unreachable()
                        }
                }
        }
        return
}

func (t *traversal) splitTargetFileName() (targetDir, targetFileName string, err error) {
        var target = t.def.target.value
        var targetName string
        switch v := target.(type) {
        case *File:
                targetName = v.name
                targetFileName = v.fullname()
                targetDir = filepath.Dir(targetFileName)
        default:
                targetDir = t.project.absPath
                if targetName, err = v.Strval(); err != nil { return }
                if filepath.IsAbs(targetName) {
                        targetFileName = targetName
                } else {
                        targetFileName = filepath.Join(targetDir, targetName)
                }
        }
        return
}

func (t *traversal) isTargetFile(file *File, targetFileName string) (res bool) {
        var target = t.def.target.value
        if target == file {
                res = true
        } else if s, _ := file.Strval(); s == targetFileName {
                res = true
        } else if t, ok := target.(*File); ok && t.name == file.name {
                res = true
        }
        return
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
        if!sys && gc.debug { fmt.Fprintf(stderr, "%v: %v: %v (exists=%v, sys=%v, from %v)\n", pos, t.entry.target, name, exists(file), sys, t.project) }
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

func (t *traversal) searchGrepped(pos Position, gc *grepctx, sys bool, linum, colnum int, name string) (file *File) {
        file = t.searchGreppedName(pos, gc, sys, linum, colnum, name)
        if file == nil {
                // The 'name' is not matching the files database.
                if gc.discard { return }
                // FIXME: missing-file error
        } else if t.isTargetFile(file, gc.targetFileName) {
                return
        } else if !exists(file) && gc.discard {
                return
        } else {
                t.grepped = append(t.grepped, file)
        }

        // Report missing files, but system files are not treated
        // as missing.
        if gc.report {
                if file == nil {
                        fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` not found\n", gc.targetFileName, linum, colnum, t.project.name, name)
                } else if !exists(file) {
                        fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` file not existed\n", gc.targetFileName, linum, colnum, t.project.name, name)
                }
        }
        return
}

func (t *traversal) savedGrepFileName(pos Position, targetFileName string) (filename string, err error) {
        var nameHash = sha256.New() //[sha256.Size]byte
        fmt.Fprintf(nameHash, "%s", targetFileName)

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
        gc.savedGrepFileName, err = t.savedGrepFileName(pos, gc.targetFileName)
        if err != nil { return }

        // Check into previously saved grep file.
        if file1 := stat(pos, gc.savedGrepFileName, "", ""); file1 != nil {
                if file2 := stat(pos, gc.targetFileName, "", ""); file2 != nil {
                        if file2.info.ModTime().After(file1.info.ModTime()) {
                                return
                        }
                }
                var e error
                var savedGrepOSFile *os.File
                if savedGrepOSFile, e = os.Open(gc.savedGrepFileName); e == nil {
                        defer savedGrepOSFile.Close()
                        scanner := bufio.NewScanner(savedGrepOSFile)
                        scanner.Split(bufio.ScanLines)
                        for scanner.Scan() {
                                var s = scanner.Text()
                                var sys, linum, colnum int
                                var name string
                                if n, e := fmt.Sscanf(s, "%d %d %d %s", &sys, &linum, &colnum, &name); e == nil && n == 4 {
                                        file := t.searchGrepped(pos, gc, sys == 1, linum, colnum, name)
                                        if file != nil && t.isTargetFile(file, gc.targetFileName) {
                                                continue
                                        }
                                }
                        }
                        file1.info, _ = savedGrepOSFile.Stat()
                        okay = true
                }
        }
        return
}

func (t *traversal) grepTargetFile(pos Position, gc *grepctx) (err error) {
        var targetOSFile *os.File
        if targetOSFile, err = os.Open(gc.targetFileName); err != nil { return }
        defer func() { err = targetOSFile.Close() } ()

        for _, x := range gc.rxs {
                if x.Regexp == nil {
                        x.Regexp, err = regexp.Compile(x.string)
                        if err != nil { return }
                }
        }

        var linum int
        scanner := bufio.NewScanner(targetOSFile)
        scanner.Split(bufio.ScanLines)
        ForScan: for scanner.Scan() {
                linum += 1
                var s = scanner.Text()
                for _, x := range gc.rxs {
                        if sm := x.FindStringSubmatch(s); len(sm) > 1 && sm[1] != "" {
                                var name = sm[1]
                                var colnum = strings.Index(s, name) //strings.IndexFunc(s, isNotSpace)
                                if gc.save != nil {
                                        var d = 0 ; if x.bool { d = 1 } // system files
                                        fmt.Fprintf(gc.save, "%d %d %d %s\n", d, linum, colnum, name)
                                }
                                var file = t.searchGrepped(pos, gc, x.bool/*system files*/, linum, colnum, name)
                                if file == nil || t.isTargetFile(file, gc.targetFileName) { continue }
                                continue ForScan // found one
                        }
                }
        }
        return
}

func (t *traversal) grepFiles(pos Position, gc *grepctx) (err error) {
        gc.targetDir, gc.targetFileName, err = t.splitTargetFileName()
        if err != nil { err = wrap(pos, err); return } else
        if !filepath.IsAbs(gc.targetFileName) {
                err = errorf(pos, "grep: '%s' is not abs", gc.targetFileName)
                return
        }
        if files, cached := grepcache[gc.targetFileName]; cached {
                fmt.Fprintf(stderr, "grep: %v\n", files)
                t.grepped = append(t.grepped, files...)
                return
        }
        defer func() { grepcache[gc.targetFileName] = t.grepped } ()

        var savedGrepFile *os.File
        var savedGrepFileLoaded bool
        savedGrepFileLoaded, err = t.loadSavedGrepFile(pos, gc)
        if err != nil || savedGrepFileLoaded { return } else
        if dir := filepath.Dir(gc.savedGrepFileName); dir != "." && dir != ".." {
                if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
        }
        if optionSaveGrepSourceName {
                var perm = os.FileMode(0600)
                var data = []byte(gc.targetFileName)
                var name = gc.savedGrepFileName + ".src"
                if err = ioutil.WriteFile(name, data, perm); err != nil { return }
        }
        if savedGrepFile, err = os.Create(gc.savedGrepFileName); err != nil { return }
        gc.save = bufio.NewWriter(savedGrepFile)
        defer func() { gc.save.Flush(); savedGrepFile.Close() } ()

        err = t.grepTargetFile(pos, gc)
        if err != nil {if gc.discard { err = nil } else { err = wrap(pos, err) }}
        return
}

// grep-files - grep files from target, example usage:
//
//      (grep-files '\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrepFiles(pos Position, t *traversal, args... Value) (result Value, err error) {
        var gc grepctx
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "c,discard-missing",
                "c,discard",
                "s,system",
                "s,sys",
                "x,regex",
                "l,lang",
                "v,verbose",
                "d,debug",
        }, func(ru rune, v Value) {
                var s string
                switch ru {
                case 'c': gc.discard = trueVal(v,true)
                case 'd': gc.debug = trueVal(v,true)
                case 'v': gc.verbose = trueVal(v,true)
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

        if err = t.grepFiles(pos, &gc); err == nil {
                result = MakeListOrScalar(pos, t.grepped)
        } else {
                err = wrap(pos, err)
        }
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
                case 'v': optVerbose = trueVal(v, true)
                case 's': optSilent = trueVal(v, true)
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
                optMode os.FileMode
                optHead Value
                optFoot Value
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "p,path", // prepare paths for files
                "r,recursive",
                "v,verbose",
                "m,mode",
                "h,head", // insert header content
                "f,foot", // insert footer content
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': optPath = trueVal(v, true)
                case 'r': optRecursive = trueVal(v, true)
                case 'v': optVerbose = trueVal(v, true)
                case 'h': if v != nil { optHead = v }
                case 'f': if v != nil { optFoot = v }
                case 'm':
                        if v != nil {
                                var num int64
                                if num, err = v.Integer(); err != nil {
                                        return
                                } else {
                                        optMode = os.FileMode(num & 0777)
                                }
                        }
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
                if optVerbose { fmt.Fprintf(stderr, "smart: Copying %v …… existed.\n", target) }
                return
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
                filename string
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "d,debug",
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': optDebug = trueVal(v, true)
                case 'v': optVerbose = trueVal(v, true)
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

        var s []byte
        if s, err = ioutil.ReadFile(filename); err == nil {
                t.def.buffer.value = &String{trivial{pos},string(s)}
        } else {
                err = &breaker{pos:pos, what:breakFail, message:err.Error()}
        }
        return
}

func modifierUpdateFile(pos Position, t *traversal, args... Value) (result Value, err error) {
        var (
                optPath bool
                optDebug bool
                optVerbose bool
                optMode = os.FileMode(0640) // sys default 0666
                filename, content string
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "d,debug",
                "p,path",
                "v,verbose",
                "m,mode",
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': optDebug = trueVal(v, true)
                case 'p': optPath = trueVal(v, true)
                case 'v': optVerbose = trueVal(v, true)
                case 'm': if v != nil {
                        var num int64
                        if num, err = v.Integer(); err != nil {
                                return
                        } else if num != 0 {
                                optMode = os.FileMode(num & 0777)
                        }
                }}
        }); err != nil { return }

        var target Value
        if len(args) > 0 { target = args[0] } else { target = t.def.target.value }
        if len(args) > 1 {
                var num int64
                if num, err = args[1].Integer(); err != nil { return }
                optMode = os.FileMode(num & 0777)
        }

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

        var f *os.File
        if f, err = os.Open(filename); err == nil && f != nil {
                defer f.Close()
                if optVerbose { fmt.Fprintf(stderr, "smart: Checking %v …", target) }
                if st, _ := f.Stat(); st.Mode().Perm() != optMode {
                        if err = f.Chmod(optMode); err != nil {
                                fmt.Fprintf(stderr, "… (error: %s)\n", err)
                                return
                        }
                }
                w1 := crc64.New(crc64Table)
                w2 := crc64.New(crc64Table)
                if _, err = io.Copy(w1, f); err != nil {
                        fmt.Fprintf(stderr, "… (error: %s)\n", err)
                        return
                }
                if _, err = io.WriteString(w2, content); err != nil {
                        fmt.Fprintf(stderr, "… (error: %s)\n", err)
                        return
                }
                if w1.Sum64() == w2.Sum64() {
                        if optVerbose { fmt.Fprintf(stderr, "… Good\n") }
                        result = stat(pos, filename, "", "")
                        return
                }
                if optVerbose { fmt.Fprintf(stderr, "… Outdated (%s)\n", filename) }
        }

        if optVerbose {
                printEnteringDirectory()
                if false {
                        fmt.Fprintf(stderr, "smart: Update %v …", filename)
                } else {
                        fmt.Fprintf(stderr, "smart: Update %v …", target)
                }
        }

        // Create or update the file with new content
        
        f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, optMode)
        if err == nil && f != nil {
                defer f.Close()
                if _, err = f.WriteString(content); err == nil {
                        var file = stat(pos, filename, "", "")
                        file.stamp(t)
                        result = file // resulting the updated file
                        if optVerbose { fmt.Fprintf(stderr, "… (ok)\n") }
                } else {
                        os.Remove(filename)
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
        var optVerbose bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'v': optVerbose = trueVal(v, true)
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
                        case 'a': optAnd = trueVal(v, false)
                        case 'g': breakScope = breakGroup
                        case 't': breakScope = breakTrave
                        case 'm': message, err = v.Strval()
                        case 'v': optVerbose = trueVal(v, optVerbose)
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
                                        if err != nil {
                                                err = wrap(a.Position(), err)
                                                return
                                        }
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
        var optDebug bool
        var optSilent bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "d,debug",
                "s,silent",
        }, func(ru rune, v Value) {
                switch ru {
                case 'd': optDebug = trueVal(v, true)
                case 's': optSilent = trueVal(v, true)
                }
        }); err != nil { return }

        err = t.wait(pos) // Wait for prerequisites
        if err != nil { err = wrap(pos, err); return }

        var reason string
        var dirty bool
        if dirty = len(t.breakers) > 0; dirty {
                reason = fmt.Sprintf("dirty: %v breakers", len(t.breakers))
        } else if dirty = !exists(t.def.target.value); dirty {
                reason = "dirty: target not exists"
        } else if dirty = len(t.updated) > 0; dirty {
                reason = fmt.Sprintf("dirty: %v updated", len(t.updated))
        } else if dirty, err = t.isRecipesDirty(); err != nil {
                err = wrap(pos, err); return
        } else if dirty {
                reason = "dirty: recipes changed"
        } else {
                reason = "Good"
        }

        if optDebug {
                var a = typeof(t.def.target.value)
                var s, _ = t.def.target.value.Strval()
                fmt.Fprintf(stderr, "%s: %s %s (dirty=%v) (updated=%v)\n", pos, a, s, dirty, t.updated)
        }

        if optionTraceTraversal {
                var v = t.def.target.value
                t.tracef("dirty: %v (updated=%v, exists=%v, target=%s)", dirty, len(t.updated), exists(v), t)
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
                case 's': optSilent = trueVal(v, true)
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
                case 'c': optClosure = trueVal(v, true)
                case 'd': optDump = trueVal(v, true)
                case 's': optSilent = trueVal(v, true)
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
                optVerbose bool
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return } else
        if args, err = parseFlags(args, []string{
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'v': optVerbose = trueVal(v, true)
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
        if n := onceCache[sum]; n > 1 {
                msg := fmt.Sprintf("once (n=%d)", n)
                err = &breaker{ pos:pos, what:breakDone, message:msg }
        }
        return
}
