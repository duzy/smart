//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "crypto/sha256"
        "path/filepath"
        "hash/crc64"
        "strings"
        "regexp"
        "errors"
        "bufio"
        "bytes"
        "time"
        "fmt"
        "os"
        "io"
        "io/ioutil"
)

const (
        TheShellEnvarsDef = "shell→envars" // '→' ' → '
        TheShellStatusDef = "shell→status" // status code of execution
)

type breakind int

func (k breakind) String() (s string) {
        switch k {
        case breakBad:          s = "break.bad"
        case breakGood:         s = "break.good"
        case breakDone:         s = "break.done"
        case breakNext:         s = "break.next"
        case breakCase:         s = "break.case"
        case breakFail:         s = "break.fail"
        }
        return
}

const (
        breakBad breakind = iota
        breakGood // good to continue
        breakDone // (cond ...) and (case ...)
        breakNext // (cond ...) and (case ...)
        breakCase // (case ...)
        breakFail // (assert ...)
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
        message string
        misstar *updatedtarget
        updated []*updatedtarget
}

func (p *breaker) Error() (s string) {
        switch p.what {
        case breakBad: s = "break baddly"
        case breakGood: s = "break normally"
        case breakDone: s = "break done" // ineligible (cond) is ignored
        case breakNext: s = "break for next case"
        case breakCase: s = "break with cases done"
        case breakFail: s = "assert" // "break with failure"
        }
        if p.message != "" {
                if s == "" {
                        s = p.message
                } else {
                        s += ": " + p.message
                }
        }
        return
}

func (p *breaker) prerequisites() (res []*updatedtarget) {
        for _, u := range p.updated {
                res = append(res, u.prerequisites...)
        }
        return
}

func break_bad(pos Position, s string, a... interface{}) *breaker {
        return &breaker{ pos, breakBad, fmt.Sprintf(s, a...), nil, nil }
}

func break_good(pos Position, s string, a... interface{}) *breaker {
        return &breaker{ pos, breakGood, fmt.Sprintf(s, a...), nil, nil }
}

func break_with(pos Position, w breakind, s string, a... interface{}) *breaker {
        return &breaker{ pos, w, fmt.Sprintf(s, a...), nil, nil }
}

func breakers(err error) (res []*breaker, rest []error) {
        switch t := err.(type) {
        case *scanner.Error:
                pos := Position(t.Pos)
                brks, errs := breakers(t.Err)
                res = append(res, brks...)
                rest = append(rest, wrap(pos, errs...))
        case scanner.Errors:
                for _, e := range t {
                        pos := Position(e.Pos)
                        brks, errs := breakers(e.Err)
                        res = append(res, brks...)
                        rest = append(rest, wrap(pos, errs...))
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
func (m *modifier) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, m)) }
        return pc.program.modify(pc, m)
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
func (g *modifiergroup) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, g)) }
ForModifiers:
        for _, m := range g.modifiers {
                if err = m.traverse(pc); err == nil { continue }
                var brks, errs = breakers(err)
                for _, b := range brks {
                        switch pc.breaker = b; b.what {
                        case breakDone:
                                // Stop traversing this group and
                                // return the breaker to the caller.
                                if errs == nil {
                                        err = nil
                                } else {
                                        err = wrap(g.position, errs...)
                                }
                                break ForModifiers
                        }
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

type ModifierFunc func(pos Position, pc *traversal, args... Value) (Value, error)

var (
        modifiers = map[string]ModifierFunc{
                `select`:       modifierSelect,

                //`args`:       modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments
                `set`:          modifierSetVar,

                `closure`:      modifierClosure,
                `unclose`:      modifierUnclose, // deprecated by (closure)
                `un`:           modifierUnclose, // shortcut

                `cd`:           modifierCD,
                `sudo`:         modifierSudo,

                //`compare`:      modifierCompare,
                `grep`:         modifierGrep,
                `grep-files`:   modifierGrepFiles,
                //`grep-compare`:      modifierGrepCompare,
                //`grep-dependencies`: modifierGrepDependencies,

                `path`:         modifierPath,

                `copy-file`:      modifierCopyFile,
                `write-file`:     modifierWriteFile,
                `update-file`:    modifierUpdateFile,
                `configure-file`: modifierConfigureFile,

                `configure`:             modifierConfigure,
                //`extract-configuration`: modifierExtractConfiguration,

                //`parallel`:     modifierParallel,

                `check`:        modifierCheck,
                `assert`:       modifierAssert,
                `case`:         modifierCase,
                `cond`:         modifierCond,
        }

        crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)

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
func modifierSelect(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var value Value
        if value, err = pc.program.scope.Lookup("-").(*Def).Call(pos); err != nil { return }
        if g, ok := value.(*Group); ok && len(args) > 0 {
                var num int64
                if num, err = args[0].Integer(); err == nil {
                        result = g.Get(int(num))
                }
        } else {
                result = &None{trivial{pos}}
        }
        return
}

func modifierSetArgs(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var envars = new(List)
        if _, err = pc.program.auto(TheShellEnvarsDef, envars); err != nil { return }
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
func modifierSetVar(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
ForArgs:
        for _, arg := range args {
                var name string
                var value Value = &None{trivial{pos}}
                switch a := arg.(type) {
                case *Pair:
                        if name, err = a.Key.Strval(); err == nil {
                                value = a.Value
                        } else { break ForArgs }
                case *Flag:
                        if name, err = a.name.Strval(); err == nil {
                                value = &None{trivial{pos}}
                                if name == "" { name = "-" }
                        } else { break ForArgs }
                case *Bareword:
                        name = a.string
                default:
                        err = scanner.Errorf(token.Position(pos), "%T `%s` is unsupported (try: foo=value)", arg, arg)
                        break ForArgs
                }
                if def := pc.program.scope.FindDef(name); def == nil {
                        err = scanner.Errorf(token.Position(pos), "`%s` no such def", name)
                        break ForArgs
                } else {
                        def.set(DefDefault, value)
                }
        }
        // TODO: result = <all changed defs>
        return
}

// create closure context of caller
func modifierClosure(pos Position, pc *traversal, args... Value) (result Value, err error) {
        // Set caller context before parsing arguments (pop the top one).
        // The context will be restored when execution is finished.
        if len(cloctx) > 0 { cloctx = cloctx[1:] }

        var dir string // closure work directory
        var optPrintEnter bool
        var optPrintLeave bool
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "i,print-enter",
                "o,print-leave",
        }, func(ru rune, v Value) {
                switch ru {
                case 'i': optPrintEnter = trueVal(v, false)
                case 'o': optPrintLeave = trueVal(v, false)
                }
        }); err != nil { return }

        if optPrintEnter { printEnteringDirectory() }
        if optPrintLeave { printLeavingDirectory() }
        if len(cloctx) == 0 {
                err = scanner.Errorf(token.Position(pos), "empty closure context")
        } else if def := cloctx[0].FindDef("/"); def == nil {
                err = scanner.Errorf(token.Position(pos), "&/ is undefined (%s)", cloctx[0].comment)
        } else if dir, err = def.value.Strval(); err != nil {
                // oops
        } else if dir == "" {
                err = scanner.Errorf(token.Position(pos), "&/ is empty (%s)", cloctx[0].comment)
        } else if !filepath.IsAbs(dir) {
                err = scanner.Errorf(token.Position(pos), "&/ is relative (%s)", cloctx[0].comment)
        } else if err = enter(pc.program, dir); err == nil {
                pc.program.project.changedWD = dir
                pc.program.changedWD = dir
        }
        return
}

func modifierUnclose(pos Position, pc *traversal, args... Value) (result Value, err error) {
        fmt.Fprintf(stderr, "%v: (unclose) is deprecated by (closure)\n", pos)
        if len(cloctx) > 0 {
                cloctx = cloctx[1:]
        }
        return
}

func findBacktrackDir() (dir string) {
        if len(execstack) > 1 {
                // Find a backtrack.
                top := execstack[0]
                for _, p := range execstack[1:] {
                        if p.project != top.project {
                                dir = p.project.AbsPath()
                                break
                        }
                }
        }
        return
}

func modifierCD(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var optPath bool
        var optPrintEnter bool
        var optPrintLeave bool
        //var target, _ = pc.program.scope.Lookup("@").(*Def).Call(pos)
        //if _, ok := target.(*Flag); ok { optPrint = false }
        if args, err = parseFlags(args, []string{
                "p,path",
                "e,print-enter",
                "l,print-leave",
                //"-,",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': optPath = trueVal(v, false)
                case 'e': optPrintEnter = trueVal(v, false)
                case 'l': optPrintLeave = trueVal(v, false)
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
                        err = scanner.Errorf(token.Position(pos), "no trackback (tracks=%v)", len(execstack))
                        return
                }
                if !filepath.IsAbs(dir) {
                        dir = filepath.Join(pc.program.project.absPath, dir)
                }
                if optPath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
                }
                if err = enter(pc.program, dir); err == nil {
                        pc.program.project.changedWD = dir
                        pc.program.changedWD = dir
                }
        } else {
                err = scanner.Errorf(token.Position(pos), "cd: wrong number of args (%v)", args)
        }
        return
}

func modifierSudo(pos Position, pc *traversal, args... Value) (result Value, err error) {
        panic("todo: sudo modifier is not implemented yet")
        return
}

func parseDependList(pos Position, pc *traversal, dependList *List) (depends *List, err error) {
        depends = new(List)
        for _, depend := range dependList.Elems {
                switch d := depend.(type) {
                case *List:
                        if dl, e := parseDependList(pos, pc, d); e != nil {
                                err = e; return
                        } else {
                                depends.Elems = append(depends.Elems, dl.Elems...)
                        }
                case *ExecResult:
                        if d.Status != 0 {
                                err = break_bad(pos, "got shell failure")
                                return // target shall be updated
                        } else {
                                depends.Append(d)
                        }
                case *RuleEntry:
                        switch d.Class() {
                        case GeneralRuleEntry, PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                                depends.Append(d)
                        default:
                                err = scanner.Errorf(token.Position(pos), "unsupported entry depend `%v' (%v)", d, d.Class())
                        }
                case *String:
                        /*if pc.program.project.IsFile(d.Strval()) {
                                Fail("compare: discarded file depend %v (%T)", depend, depend)
                        } else*/ {
                                depends.Append(d)
                        }
                case *File:
                        depends.Append(d)
                default:
                        err = scanner.Errorf(token.Position(pos), "unsupported entry depend `%v' (%v)", depend, pc.program.depends)
                }
        }
        return
}

/*
func compareTargetDepend(pos Position, pc *traversal, target, depend Value, tt time.Time) (outdated bool, err error) {
        if dependFile, okay := depend.(*File); okay && dependFile != nil {
                var str string
                if str, err = dependFile.Strval(); err != nil { return }
                if t := context.globe.timestamp(str); t.After(tt) {
                        outdated = true; return // target is outdated
                } else if dependFile.info == nil {
                        dependFile.info, _ = os.Stat(str)
                }
                if dependFile.info == nil {
                        err = break_bad(pos, "no file or directory '%v'", dependFile)
                        return
                }
                if t := dependFile.info.ModTime(); t.After(tt) {
                        if str, err = target.Strval(); err != nil { return }
                        context.globe.stamp(str, t)
                        outdated = true; return // target is outdated
                } else {
                        var (
                                recipes []Value
                                strings []string
                        )
                        if recipes, err = Disclose(pc.program.recipes...); err != nil {
                                return
                        }
                        for _, recipe := range recipes {
                                strings = append(strings, recipe.String())
                        }
                        if same, e := pc.program.project.CheckCmdHash(target, strings); e == nil {
                                outdated = !same
                        }
                }
                if !outdated {
                        //ent, _ := pc.program.project.Entry(depend.Strval())
                        //fmt.Fprintf(stderr, "compare: %v\n", ent)
                }
        } else {
                fmt.Fprintf(stderr, "compare: todo: %v -> %v (%T)\n", target, depend, depend)
        }
        return
}
*/

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
func parseGrepOption(pos Position, pc *traversal, optGrep Value) (result []Value, err error) {
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
                err = scanner.Errorf(token.Position(pos), "`%T` non-group grep option", optGrep)
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
                                                        err = scanner.Errorf(token.Position(pos), "`%s` unknown regexp (%v)", s, p.Value)
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
                        switch t := val.(type) {
                        case *None: continue
                        case *File:
                                var files []Value
                                if info, ok := grepCacheFilebase[t.filebase]; ok {
                                        for _, file := range info.list {
                                                if _, ok = unique[file.filebase]; !ok {
                                                        files = append(files, file)
                                                }
                                                //unique[file.filebase] += 1
                                        }
                                } else if exists(t) {
                                        var list []Value
                                        list, err = pc.derived.grepFiles(val, tops, rxs, optReportMissing, optDiscardMissing)
                                        if err != nil { return }
                                        info = &grepCacheFiles{ file:t }
                                        grepCacheFilebase[t.filebase] = info
                                        for _, v := range list {
                                                var file = v.(*File)
                                                info.list = append(info.list, file)
                                                if _, ok := unique[file.filebase]; !ok {
                                                        files = append(files, file)
                                                }
                                                //unique[file.filebase] += 1
                                        }
                                }
                                unique[t.filebase] += 1
                                result = append(result, t)
                                if files != nil {
                                        if optRecursive { grep(files) } else {
                                                result = append(result, files...)
                                        }
                                }
                        default:
                                err = scanner.Errorf(token.Position(pos), "'%v' cant grep this type", t)
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
        if def := pc.program.scope.FindDef(s); def != nil {
                def.append(result...)
                result = nil
        } else {
                err = scanner.Errorf(token.Position(pos), "`%s` no such Def", s)
        }
        return
}

//var uniqueCompareGood = make(map[string]*breaker)
// sysgrcmp := '^\s*#\s*include\s*<(.*)>'
// grepcmps := '^\s*#\s*include\s*"(.*)"'
// (compare -pg=(regexp=(top=(llvm,llvm-c,clang,clang-c) sys=$(sysgrcmp) $(grepcmps)) -re $<))
// (compare -pg=(lang=c++) -re $<)
/*func modifierCompare(pos Position, pc *traversal, args... Value) (result Value, err error) {
        var (
                optDiscardMissing, optVerbose bool
                optPath, optNoUpdate, optMulti bool
                optGrep Value
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "p,path",
                "e,report",
                "i,ignore",
                "n,no-time",
                "m,multi", // multiple compare/execution
                "g,grep", // -grep=(regexp=(sys='...' '...') $^)
                //"t,touch", // touch target file when updated
                //"t,stamp", // touch target file when updated
                //"r,recursive",
                "v,verbose",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': optPath = trueVal(v, true)
                case 'm': optMulti = trueVal(v, true)
                case 'v': optVerbose = trueVal(v, true)
                case 'n': optNoUpdate = trueVal(v, true)
                case 'i': optDiscardMissing = trueVal(v, true)
                //case 't': optTouch = trueVal(v, true)
                case 'g': optGrep = v
                }
        }); err != nil { return }

        // No need to do further comparation if any prerequisites
        // already modified. We have to update the target if so.
        if len(pc.modified) > 0 {
                // Just return to continue with the rest modifiers.
                return
        }
        
        var target Value
        var targetStr string
        var targetDef = pc.targetDef // $@
        if n := len(args); n == 1 {
                // Change $@ to args[0]
                targetDef.set(DefDefault, args[0])
        } else if n > 1 {
                err = break_bad(pos, "two many targets (%v)", args)
                return
        }

        // Get target value from $@.
        if target, err = targetDef.Call(pos); err != nil {
                err = break_bad(pos, "comparing %v: %v", targetDef, err)
                return
        } else if target == nil {
                err = break_bad(pos, "comparing nil target")
                return
        } else if _, ok := target.(*None); ok {
                err = break_bad(pos, "comparing 'none' target")
                return
        } else if file, ok := target.(*File); ok && file.info != nil && file.updated {
                // already updated
                return
        }

        // Target string value (name).
        if targetStr, err = target.Strval(); err != nil { return }
        if optPath {
                var s string = filepath.Dir(targetStr)
                if s != "." && s != ".." && s != PathSep {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                                return
                        }
                }
        }

        // Block until all prerequisites are good.
        if err = pc.program.waitForPrerequisites(); err != nil {
                return
        }

        var comparedGood bool
        if !optMulti {
                good, found := uniqueCompareGood[targetStr]
                if found && good != nil {
                        comparedGood = true
                        //fmt.Printf("compare: %v (%v)\n", targetStr, good)

                        //// Return if the target has already compared
                        //err =  good
                        //return
                }
        }

        // Grep files first if enabled to ensure that the $~ is updated.
        var grepped Value
        if optGrep != nil {
                var res []Value
                if res, err = parseGrepOption(pos, prog, optGrep); err != nil { return }
                if res != nil { grepped = MakeListOrScalar(res) }
        }

        var depends []Value
        depends = append(depends, pc.dependsDef.Value) // $^
        depends = append(depends, pc.orderedDef.Value) // $|
        depends = append(depends, pc.greppedDef.Value) // $~
        if grepped != nil { depends = append(depends, grepped) }

        if true { // 'false' is okay here
                depends, err = mergeresult(ExpandAll(depends...))
                if err != nil { return }
        }

        // Change into compare mode to avoid interpretion if there're
        // no targets are updated
        //pc.mode = compareMode

        var c *comparer
        if c, err = newcompariation(prog, target); err != nil {
                return
        }

        if (optVerbose || optionVerboseChecks) && !comparedGood {
                if true {
                        fmt.Fprintf(stderr, "smart: Checking %s …", target)
                } else {
                        tar, _ := filepath.Rel(pc.program.project.absPath, targetStr)
                        fmt.Fprintf(stderr, "smart: Checking %s …", tar)
                }
        }

        c.nomiss, c.nocomp = optDiscardMissing, optNoUpdate
        if err = c.Compare(pos, depends); err == nil {
                result = &boolean{pos,true}
        } else {
                result = &boolean{pos,false}
                if br, ok := err.(*breaker); ok {
                        switch br.what {
                        case breakGood:
                                uniqueCompareGood[targetStr] = br
                        }
                } else {
                        fmt.Fprintf(stderr, "%v: compare: %v\n", pos, err)
                        fmt.Fprintf(stderr, "%v: %v\n", targetStr, depends)
                }
        }

        if (optVerbose || optionVerboseChecks) && !comparedGood {
                if err == nil {
                        fmt.Fprintf(stderr, "… (done)")
                } else if br, ok := err.(*breaker); ok {
                        switch br.what {
                        case breakBad: fmt.Fprintf(stderr, "… Bad (%s)", br.message)
                        case breakGood: fmt.Fprintf(stderr, "… Good")
                        case breakUpdates:
                                if a := br.prerequisites(); len(a) == 0 {
                                        fmt.Fprintf(stderr, "… Outdated")
                                } else {
                                        fmt.Fprintf(stderr, "… %v", a)
                                }
                        }
                }
                fmt.Fprintf(stderr, "\n")
        }

        if len(c.updated) > 0 && enable_assertions {
                assert(err != nil, "expects update breaker")
                if false { fmt.Fprintf(stderr, "compare: %v\n", err) }
                // e, ok := err.(*breaker)
                // assert(ok, "expects update breaker")
                // assert(e.what == breakUpdates, "expects update breaker")
                // assert(e.updated != nil, "nil updated target")
                //assert(e.updated.target == c.target, "updated target differs")
                //assert(len(e.updated.prerequisites) == len(c.updated), "updated target differs")
                //for i, preq := range e.updated.prerequisites {
                //       assert(preq == c.updated[i], "updated target differs")
                //}
        }
        return
}*/

// grep-compare - grep files from target and compare, example usage:
//
//      (grep-compare '\s*#\s*include\s*<(.*)>')
//      
/*func modifierGrepCompare(pos Position, pc *traversal, args... Value) (result Value, err error) {
        var (
                optPath, optNoUpdate, optDisMiss bool
                optTarget Value
        )
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "p,path",
                "i,ignore",
                "n,no-update",
                "t,target",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': optPath = trueVal(v, true)
                case 'i': optDisMiss = trueVal(v, true)
                case 'n': optNoUpdate = trueVal(v, true)
                case 't': optTarget = v
                }
        }); err != nil { return }

        var target = pc.program.scope.Lookup("@").(*Def)
        if optTarget != nil {
                defer func(v Value) { target.set(DefDefault, v) } (target.Value)
                target.set(DefDefault, optTarget)
        }
        if optPath && target.Value != nil {
                var s string
                if s, err = target.Value.Strval(); err != nil { return }
                if s = filepath.Dir(s); s != "." && s != ".." && s != PathSep {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil { return }
                }
        }

        //if v, e := grepFiles(target, rxs, true, optDisMiss); e != nil {
        if v, e := modifierGrepFiles(pos, prog, args...); e != nil {
                err = e
        } else if v != nil {
                var c *comparer
                if c, err = newcompariation(prog, target); err == nil {
                        c.nomiss = optDisMiss
                        c.nocomp = optNoUpdate
                        if err = c.Compare(pos, v); err == nil {
                                result = &boolean{pos,true} //MakeListOrScalar(c.result)
                        } else {
                                result = &boolean{pos,false}
                        }
                }
        }
        return
}

// grep-dependencies - grep dependencies from target, example usage:
//
//      (grep-dependencies '\s*#\s*include\s*<(.*)>')
//      
func modifierGrepDependencies(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if v, e := modifierGrepFiles(pos, prog, args...); e != nil {
                err = e
        } else if v != nil {
                if false {
                        err = pc.program.scope.Lookup("^").(*Def).append(v)
                } else {
                        err = pc.program.scope.Lookup("|").(*Def).append(v)
                }
        }
        return
}
*/

type greprex struct{ string ; bool ; *regexp.Regexp }
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
                                file := stat(a[0], a[1], a[2])
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

func (p *Project) grepFiles(target Value, tops []string, rxs []*greprex, report, discard bool) (result []Value, err error) {
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

        var isSameAsTarget = func(file *File) (res bool) {
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
                        file = stat(name, "", "", nil)
                } else if isRel = isRelPath(name); isRel { // relative to target dir
                        file = stat(name, "", targetDir, nil)
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
                        var alt = stat(name, "", targetDir)
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
                } else if isSameAsTarget(file) {
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
        var savedGrepFile = p.matchTempFile(filepath.Join(".grep",
                fmt.Sprintf("%x", nameSum[ :1]),
                fmt.Sprintf("%x", nameSum[1:2]),
                fmt.Sprintf("%x", nameSum[2: ]),
        ))
        savedGrepFileName, _ = savedGrepFile.Strval()

        if file1 := stat(savedGrepFileName, "", ""); file1 != nil {
                if file2 := stat(targetFileName, "", ""); file2 != nil {
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
                                        if file != nil && isSameAsTarget(file) {
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
                                        if isSameAsTarget(file) {
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

// grep-files - grep files from target, example usage:
//
//      (grep-files '\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrepFiles(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var top []Value
        var rxs []*greprex
        var optReportMissing = true // TODO: option "e,report" 
        var optDiscardMissing = false // TODO: option "i,ignore"
        if args, err = parseFlags(args, []string{
                "d,discard-missing",
                "s,system",
                "s,sys",
                "t,top",
        }, func(ru rune, v Value) {
                var s string
                switch ru {
                case 's':
                        // FIXME: if v.(*Group) ...
                        if v == nil { return }
                        if s, err = v.Strval(); err != nil { return }
                        rxs = append(rxs, &greprex{s, true, nil})
                case 't':
                        if v == nil { return }
                        if g, ok := v.(*Group); ok {
                                top = merge(g.Elems...)
                        } else {
                                top = merge(v)
                        }
                default:
                        err = scanner.Errorf(token.Position(pos), "`%v` unsupported argument (%s)", v, s)
                        return
                }
        }); err != nil { return }
        
        for _, arg := range args {
                var s string
                if s, err = arg.Strval(); err != nil { return }
                rxs = append(rxs, &greprex{s, false, nil})
        }
        if len(rxs) == 0 {
                err = scanner.Errorf(token.Position(pos), "no grep expressions")
                return
        }

        var tops []string
        for _, v := range top {
                var s string
                if s, err = v.Strval(); err != nil { return }
                tops = append(tops, s)
        }

        var list []Value
        var target, _ = pc.program.scope.Lookup("@").(*Def).Call(pos)
        list, err = pc.derived.grepFiles(target, tops, rxs, optReportMissing, optDiscardMissing)
        result = MakeListOrScalar(pos, list)
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
func modifierGrep(pos Position, pc *traversal, args... Value) (result Value, err error) {
        err = scanner.Errorf(token.Position(pos), "unimplemented grep %v", args)
        return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
func modifierCheck(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var value Value
        if value, err = pc.program.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var optBreak breakind // breaking with good results
        var optSilent bool // don't break on failures
        var makeResult func(Position,bool) Value // returns results only if non-nil
        var values []Value
        var pairs []*Pair
        if args, err = parseFlags(args, []string{
                "a,answer",
                "r,result",
                "s,silent",
                "g,good",
        }, func(ru rune, v Value) {
                switch ru {
                case 'a': makeResult = MakeAnswer
                case 'r': makeResult = MakeBoolean
                case 'g': optBreak = breakGood
                case 's': optSilent = trueVal(v, false)
                }
        }); err != nil { return }

        for _, arg := range args {
                switch t := arg.(type) {
                case *Pair: pairs = append(pairs, t)
                default:
                        err = scanner.Errorf(token.Position(pos), "unknown check '%v' (%T)", arg, arg)
                return
                }
        }

        if optSilent && makeResult == nil {
                makeResult = MakeBoolean
        }

ForPairs:
        for _, t := range pairs {
                var key, str string
                if key, err = t.Key.Strval(); err != nil { return }
                switch key {
                case "status":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                err = break_with(pos, optBreak, "not an exec result (%T)", value)
                                return
                        }

                        var num int64
                        if num, err = t.Value.Integer(); err != nil { return }
                        if res := exeres.Status == int(num); makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                err = break_with(pos, optBreak, "bad status (%v) (expects %v)", exeres.Status, t.Value)
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                err = break_with(pos, optBreak, "not an exec result (%T)", value)
                                return
                        }

                        var v *bytes.Buffer
                        switch key {
                        case "stdout": v = exeres.Stdout.Buf
                        case "stderr": v = exeres.Stderr.Buf
                        default: unreachable()
                        }

                        if v == nil {
                                err = break_with(pos, optBreak, "bad %s (expects %v)", key, t.Value)
                                break ForPairs
                        }
                        if str, err = t.Value.Strval(); err != nil { 
                                return
                        } else if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(pos, res))
                        } else if !res {
                                err = break_with(pos, optBreak, "bad %s (%v) (expects %v)", key, v, t.Value)
                                break ForPairs
                        }
                case "file", "dir":
                        var file *File
                        var project = pc.derived //mostDerived() // pc.program.project
                        if str, err = t.Value.Strval(); err != nil { return }
                        if file := project.searchFile(str); !exists(file) {
                                err = break_with(pos, optBreak, "`%v` no such file or directory", t.Value)
                                break ForPairs
                        }
                        switch key {
                        case "file":
                                if res := file.info.Mode().IsRegular(); makeResult != nil {
                                        values = append(values, makeResult(pos, res))
                                } else if !res {
                                        err = break_with(pos, optBreak, "`%v` is not a regular file", t.Value)
                                        break ForPairs
                                }
                        case "dir":
                                if res := file.info.Mode().IsDir(); makeResult != nil {
                                        values = append(values, makeResult(pos, res))
                                } else if !res {
                                        err = break_with(pos, optBreak, "`%v` is not a directory", t.Value)
                                        break ForPairs
                                }
                        default: unreachable()
                        }
                case "var":
                        g, ok := t.Value.(*Group)
                        if !ok {
                                err = break_with(pos, optBreak, "`%v` is not a group value", t.Value)
                                break ForPairs
                        }
                        for _, elem := range g.Elems {
                                switch p := elem.(type) {
                                case *Pair:
                                        var k, a, b string
                                        if k, err = p.Key.Strval(); err != nil { break ForPairs }
                                        var def = pc.program.project.scope.FindDef(k)
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
                        err = scanner.Errorf(token.Position(pos), "unknown check '%v'", t.Key)
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

func copyRegular(src, dst string, opts *copyopts) (err error) {
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
                
                var file = stat(dst, "", "")
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

func copySymlink(src, dst string, opts *copyopts) (err error) {
        panic("unimplemented copySymlink")
}

func copyDir(src, dst string, opts *copyopts) (err error) {
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
                err = copyFile(fi, ss, sd, opts)
                if err != nil { break }
        }
        return
}

func copyFile(srcFi os.FileInfo, src, dst string, opts *copyopts) (err error) {
        if m := srcFi.Mode(); m&os.ModeSymlink != 0 {
                if opts.mode == 0 { opts.mode = srcFi.Mode() }
                err = copySymlink(src, dst, opts)
        } else if srcFi.IsDir() {
                err = copyDir(src, dst, opts)
        } else if m.IsRegular() {
                if opts.mode == 0 { opts.mode = srcFi.Mode() }
                err = copyRegular(src, dst, opts)
        } else {
                err = fmt.Errorf("copying non-regular files/dirs (%s)", src)
        }
        return
}

// (path $(dir $@))
// (path /example/path)
func modifierPath(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                // ...
        }, func(ru rune, v Value) {
                // ...
        }); err != nil { return }
        if len(args) == 0 {
                var target Value
                if target, err = pc.program.scope.Lookup("@").(*Def).Call(pos); err != nil {
                        return
                }
                var s string
                if s, err = target.Strval(); err != nil { return }
                if s = filepath.Dir(s); s != "" && s != "." && s != "/" {
                        err = os.MkdirAll(s, os.FileMode(0755))
                }
                return
        }
        for _, arg := range args {
                var s string
                if s, err = arg.Strval(); err != nil { return }
                fmt.Printf("path: %v\n", s)
                if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                        return
                }
        }
        return
}

// (copy-file -vp)
// (copy-file -p,filename)
// (copy-file -p,filename,source)
func modifierCopyFile(pos Position, pc *traversal, args... Value) (result Value, err error) {
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
        } else if target, err = pc.program.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        }
        if len(args) > 1 {
                source = args[1]
        } else if source, err = pc.program.scope.Lookup("<").(*Def).Call(pos); err != nil {
                return
        }

        // Get target filename
        var (
                project = pc.derived //mostDerived() // pc.program.project
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
                t, _ := pc.program.scope.Lookup("@").(*Def)
                a, _ := pc.program.scope.Lookup("^").(*Def)
                fmt.Printf("warning: %v, %v (%v) (%v)\n", target, source, t, a)
        }

        if !filetime.IsZero() && filetime.After(srctime) {
                if optVerbose {
                        fmt.Fprintf(stderr, "smart: Copying %v …… existed.\n", target)
                }
                return
        } else if optVerbose {
                fmt.Fprintf(stderr, "smart: Copying %v …", target)
        }
        
        var copyOpts = &copyopts{ optPath, optMode, optHead, optFoot }
        var fi os.FileInfo
        if fi, err = os.Stat(srcname); err != nil {
                err = wrap(pos, err)
        } else if !fi.IsDir() {
                if optMode == 0 { optMode = fi.Mode() }
                if err = copyFile(fi, srcname, filename, copyOpts); err != nil {
                        err = wrap(pos, err)
                }
        } else if optRecursive {
                if err = copyDir(srcname, filename, copyOpts); err != nil {
                        err = wrap(pos, err)
                }
        } else {
                err = fmt.Errorf("`%v` is a directory (use -r to solve it)", source)
                err = wrap(pos, err)
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

func modifierWriteFile(pos Position, pc *traversal, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                filename, str string
                target Value
                f *os.File
        )
        if target, err = pc.program.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        } else if filename, err = target.Strval(); err != nil {
                return
        }
        if f, err = os.Create(filename); err == nil {
                defer f.Close()

                var value Value
                if value, err = pc.program.scope.Lookup("-").(*Def).Call(pos); err != nil {
                        return
                } else if str, err = value.Strval(); err != nil {
                        return
                } else if _, err = f.WriteString(str); err == nil {
                        result = stat(filename, "", "")
                } else {
                        os.Remove(filename)
                }
        } else {
                err = break_bad(pos, "file %s not generated", target)
        }
        return
}

func modifierUpdateFile(pos Position, pc *traversal, args... Value) (result Value, err error) {
        //if m := pc.mode; m == compareMode {
        //        return /* not working in compare mode */
        //}
        //if m := pc.mode; m != updateMode {
        //        //return /* only configure in update mode */
        //}

        var (
                optPath bool
                optVerbose bool
                optMode = os.FileMode(0640) // sys default 0666
                filename, content string
        )

        // Process flags
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        } else if args, err = parseFlags(args, []string{
                "p,path",
                "v,verbose",
                "m,mode",
        }, func(ru rune, v Value) {
                switch ru {
                case 'p': optPath = trueVal(v, true)
                case 'v': optVerbose = trueVal(v, true)
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
        if target, err = pc.program.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        }
        
        if len(args) > 0 { target = args[0] }
        if len(args) > 1 {
                var num int64
                if num, err = args[1].Integer(); err != nil { return }
                optMode = os.FileMode(num & 0777)
        }

        // Get target filename
        var project = pc.derived //mostDerived() // pc.program.project
        switch t := target.(type) {
        case *File:
                if filename, err = t.Strval(); err != nil {
                        return
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
                }
        }

        // Make path (mkdir -p)
        if optPath {
                if p := filepath.Dir(filename); p != "." && p != "/" {
                        if err = os.MkdirAll(p, os.FileMode(0755)); err != nil {
                                return
                        }
                }
        }

        // Check existed file content checksum
        var value Value
        if value, err = pc.program.scope.Lookup("-").(*Def).Call(pos); err != nil { return }
        if content, err = value.Strval(); err != nil { return }

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
                        result = stat(filename, "", "")
                        return
                }
                if optVerbose { fmt.Fprintf(stderr, "… Outdated\n") }
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
                        var file = stat(filename, "", "")
                        file.stamp(pc)
                        result = file // resulting the updated file
                        if optVerbose { fmt.Fprintf(stderr, "… (ok)\n") }
                } else {
                        os.Remove(filename)
                        if optVerbose { fmt.Fprintf(stderr, "… (%s)\n", err) }
                }
        } else {
                if optVerbose { fmt.Fprintf(stderr, "… (%s)\n", err) }
                err = break_bad(pos, "file %s not updated", target)
        }
        return
}

/*func modifierParallel(pos Position, pc *traversal, args... Value) (result Value, err error) {
        // TODO: specify parallel options, e.g.:
        //   (parallel -n=0) # turn off
        //   (parallel -n=5) # five workers
        return
}*/

// (assert condition,'error message...')
func modifierAssert(pos Position, pc *traversal, args... Value) (result Value, err error) {
        var t bool
        if len(args) == 0 {
                err = &breaker{
                        pos:pos, what:breakFail,
                        message: "zero-args assertion",
                }
        } else if t, err = args[0].True(); err == nil && !t {
                var br = &breaker{
                        pos:pos, what:breakFail,
                        message: "assertion failed",
                }
                if len(args) > 1 {
                        br.message, _ = args[1].Strval()
                }
                err = br
        }
        return
}

func modifierCase(pos Position, pc *traversal, args... Value) (result Value, err error) {
        for _, arg := range args {
                for _, a := range merge(arg) {
                        var t bool
                        if t, err = a.True(); err != nil {
                                break
                        } else if !t {
                                err = &breaker{ pos:pos, what:breakNext }
                                return
                        }
                }
        }
        err = &breaker{ pos:pos, what:breakCase }
        return
}

func modifierCond(pos Position, pc *traversal, args... Value) (result Value, err error) {
        var target string
        if target, err = pc.targetDef.value.Strval(); err != nil {
                err = wrap(pos, err)
                return
        }

        target = filepath.Base(target)

        var reasons []string
        var optVerbose, optAnd, verbose0, done bool
        defer func() {
                if optVerbose {
                        var status = "Good"
                        if reasons != nil {
                                status = "Bad (" + strings.Join(reasons, ",") + ")"
                        }
                        fmt.Fprintf(stderr, "… %s\n", status)
                }
        } ()

        for _, arg := range args {
                var optDirty bool
                var va = merge(arg)
                if va, err = parseFlags(va, []string{
                        "a,and",
                        "d,dirty",
                        "v,verbose",
                }, func(ru rune, v Value) {
                        switch ru {
                        case 'a': optAnd = trueVal(v, false)
                        case 'd': optDirty = trueVal(v, false)
                        case 'v': optVerbose = trueVal(v, optVerbose)
                                if optVerbose && !verbose0 {
                                        fmt.Fprintf(stderr, "smart: Checking %v …", target)
                                        verbose0 = true
                                }
                        }
                }); err != nil { return }
                if !optAnd || (optAnd && done) {
                        if optDirty {
                                var dirty = pc.breaker != nil || !(exists(pc.targetDef.value) && len(pc.updated) == 0)
                                if !dirty {
                                        dirty, err = pc.isRecipesDirty()
                                        if err != nil { return }
                                }
                                if dirty { reasons = append(reasons, "-dirty") }
                                if optionTraceTraversal {
                                        pc.tracef("dirty: %v (updated=%v, exists=%v, target=%s)", dirty, len(pc.updated), exists(pc.targetDef.value), pc.targetDef.value)
                                        if len(pc.updated) > 0 { pc.tracef("dirty: updated=%v", pc.updated) }
                                }
                                if optAnd {
                                        done = done && !dirty
                                        optAnd = false // reset -and flag
                                } else if !dirty {
                                        done = true
                                }
                        }
                        for i, a := range va {
                                var t = true
                                if t, err = a.True(); err != nil { return }
                                if t { reasons = append(reasons, fmt.Sprintf("#%v", i+1)) }
                                if optAnd {
                                        done = done && !t
                                        optAnd = false // reset -and flag
                                } else if !t {
                                        done = true
                                        break
                                }
                        }
                }
        }
        if done && err == nil { err = &breaker{ pos:pos, what:breakDone }}
        return
}
