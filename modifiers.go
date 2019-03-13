//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
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
)

const (
        TheShellEnvarsDef = "shell->envars"
        TheShellStatusDef = "shell->status" // status code of execution
)

type breakind int

const (
        breakBad breakind = iota
        breakGood // good to continue
        breakUpdates // needs to update
)

type breaker struct {
        pos Position
        what breakind
        message string
        updated []*updatedtarget
}

func (p *breaker) Error() string {
        if p.what == breakUpdates {
                return fmt.Sprintf("updated %v", p.updated)
        }
        return p.message
}

func (p *breaker) prerequisites() (res []*updatedtarget) {
        for _, u := range p.updated {
                res = append(res, u.prerequisites...)
        }
        return
}

func break_bad(pos Position, s string, a... interface{}) *breaker {
        return &breaker{ pos, breakBad, fmt.Sprintf(s, a...), nil }
}

func break_good(pos Position, s string, a... interface{}) *breaker {
        return &breaker{ pos, breakGood, fmt.Sprintf(s, a...), nil }
}

func break_updates(pos Position, v ...*updatedtarget) *breaker {
        return &breaker{ pos, breakUpdates, "", v }
}

func break_with(pos Position, w breakind, s string, a... interface{}) *breaker {
        return &breaker{ pos, w, fmt.Sprintf(s, a...), nil }
}

type ModifierFunc func(pos Position, prog *Program, args... Value) (Value, error)

var (
        modifiers = map[string]ModifierFunc{
                `select`:       modifierSelect,

                //`args`:       modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments
                `set`:          modifierSetVar,

                `unclose`:      modifierUnclose,

                `cd`:           modifierCD,
                `sudo`:         modifierSudo,

                `compare`:      modifierCompare,
                `grep`:         modifierGrep,
                `grep-files`:   modifierGrepFiles,
                //`grep-compare`:      modifierGrepCompare,
                //`grep-dependencies`: modifierGrepDependencies,

                `check`:          modifierCheck,
                
                `write-file`:     modifierWriteFile,
                `update-file`:    modifierUpdateFile,
                `configure-file`: modifierConfigureFile,

                `configure`:             modifierConfigure,
                `extract-configuration`: modifierExtractConfiguration,

                `parallel`:     modifierParallel,
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

func modifierSelect(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }
        if g, ok := value.(*Group); ok && len(args) > 0 {
                var num int64
                if num, err = args[0].Integer(); err == nil {
                        result = g.Get(int(num))
                }
        } else {
                result = universalnone
        }
        return
}

func modifierSetArgs(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var envars = new(List)
        if _, err = prog.auto(TheShellEnvarsDef, envars); err != nil { return }
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

func modifierSetVar(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        for _, arg := range args {
                p, ok := arg.(*Pair)
                if !ok {
                        err = scanner.Errorf(token.Position(pos), "%s `%s` is unsupported (try: foo=value)", arg.Type(), arg)
                        break
                }
                var name string
                if name, err = p.Key.Strval(); err != nil { return }
                if def := prog.scope.FindDef(name); def == nil {
                        err = scanner.Errorf(token.Position(pos), "`%s` no such def", name)
                        break
                } else {
                        def.set(DefDefault, p.Value)
                }
        }
        //result = <all defs changed>
        return
}

func modifierUnclose(pos Position, prog *Program, args... Value) (result Value, err error) {
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

func modifierCD(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var optPath bool
        var optPrintEnter bool
        var optPrintLeave bool
        //var target, _ = prog.scope.Lookup("@").(*Def).Call(pos)
        //if _, ok := target.(*Flag); ok { optPrint = false }
        if len(args) > 0 {
                var v []Value
                for _, arg := range args {
                        switch a := arg.(type) {
                        default: v = append(v, arg)
                        case *Flag:
                                var opt bool
                                if opt, err = a.is(0, "print-enter"); err != nil { return } else if opt { optPrintEnter = opt }
                                if opt, err = a.is(0, "print-leave"); err != nil { return } else if opt { optPrintLeave = opt }
                                if opt, err = a.is('p', "path"); err != nil { return } else if opt { optPath = opt }
                                //if opt, err = a.is('s', "silent"); err != nil { return } else if opt { optPrint = false }
                                if opt, err = a.is(0, ""); err != nil { return } else if opt {
                                        var dir = findBacktrackDir()
                                        // Back to main project if no backtracks.
                                        if dir == "" && prog.globe.main != nil {
                                                dir = prog.globe.main.AbsPath()
                                        }
                                        v = append(v, &String{dir})
                                }
                        }
                }
                args = v // Reset args
        }
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
                if optPath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
                }
                err = enter(prog, dir)
        } else {
                err = scanner.Errorf(token.Position(pos), "wrong number of args (%v)", args)
        }
        return
}

func modifierSudo(pos Position, prog *Program, args... Value) (result Value, err error) {
        panic("todo: sudo modifier is not implemented yet")
        return
}

func parseDependList(pos Position, prog *Program, dependList *List) (depends *List, err error) {
        depends = new(List)
        for _, depend := range dependList.Elems {
                switch d := depend.(type) {
                case *List:
                        if dl, e := parseDependList(pos, prog, d); e != nil {
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
                        /*case ExplicitFileEntry:
                                var name string
                                if name, err = d.Strval(); err == nil {
                                        depends.Append(...)
                                }*/
                        case GeneralRuleEntry, GlobRuleEntry:
                                depends.Append(d)
                        default:
                                err = scanner.Errorf(token.Position(pos), "unsupported entry depend `%v' (%v)", d, d.Class())
                        }
                case *String:
                        /*if prog.project.IsFile(d.Strval()) {
                                Fail("compare: discarded file depend %v (%T)", depend, depend)
                        } else*/ {
                                depends.Append(d)
                        }
                case *File:
                        depends.Append(d)
                default:
                        err = scanner.Errorf(token.Position(pos), "unsupported entry depend `%v' (%v)", depend, prog.depends)
                }
        }
        return
}

func compareTargetDepend(pos Position, prog *Program, target, depend Value, tt time.Time) (outdated bool, err error) {
        if dependFile, okay := depend.(*File); okay && dependFile != nil {
                var str string
                if str, err = dependFile.Strval(); err != nil { return }
                if t, ok := prog.globe.timestamps[str]; ok && t.After(tt) {
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
                        prog.globe.timestamps[str] = t
                        outdated = true; return // target is outdated
                } else {
                        var (
                                recipes []Value
                                strings []string
                        )
                        if recipes, err = Disclose(prog.recipes...); err != nil {
                                return
                        }
                        for _, recipe := range recipes {
                                strings = append(strings, recipe.String())
                        }
                        if same, e := prog.project.CheckCmdHash(target, strings); e == nil {
                                outdated = !same
                        }
                }
                if !outdated {
                        //ent, _ := prog.project.Entry(depend.Strval())
                        //fmt.Fprintf(stderr, "compare: %v\n", ent)
                }
        } else {
                fmt.Fprintf(stderr, "compare: todo: %v -> %v (%T)\n", target, depend, depend)
        }
        return
}

type langInfoT struct {
        rxs []string
}

var langInfos = map[string]*langInfoT{
        "c": &langInfoT{
                []string{
                        `^\s*#\s*include\s*<(.*)>`,
                        `^\s*#\s*include\s*"(.*)"`,
                },
        },
        "i": &langInfoT{
                []string{
                        `^\s*include\s*"(.*)"`,
                },
        },
}
func init () {
        if info, ok := langInfos["c"]; ok {
                langInfos["c++"] = info
                langInfos["clang"] = info
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
func parseGrepOption(pos Position, prog *Program, optGrep Value) (result []Value, err error) {
        var group *Group
        if g, ok := optGrep.(*Group); ok { group = g } else {
                err = scanner.Errorf(token.Position(pos), "`%T` non-group grep option", optGrep)
                return
        }

        var (
                top []Value
                rxs []*greprex
                vals []Value
                store Value
                opts = []string{
                        "e,report",
                        "i,ignore",
                        "r,recursive",
                        "l,lang", // TODO: -lang=c|c++|go|java|...
                        "c,clang", // TODO: same as -lang=c
                }
                optReportMissing bool = true
                optDiscardMissing bool
                optRecursive bool
                optLang string
        )
ForGroupElems:
        for _, elem := range group.Elems {
                var ( runes []rune ; names []string ; v Value )
                switch a := elem.(type) {
                case *Flag:
                        if runes, names, err = a.opts(opts...); err != nil { return }
                        v = nil // no flag value
                case *Pair:
                        if flag, ok := a.Key.(*Flag); ok && flag != nil {
                                if runes, names, err = flag.opts(opts...); err != nil { return }
                                v = a.Value // got flag value
                                break
                        }

                        var s string
                        if s, err = a.Key.Strval(); err != nil { return }
                        switch s {
                        case "regexp", "exp":
                                switch v := a.Value.(type) {
                                case *Group: for _, v := range v.Elems {
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
                                                                top = merge(a.Value)
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
                        case "lang":
                                optLang, _ = a.Value.Strval()
                                if optLang == "" {/* ... */}
                        case "s":
                                store = a.Value
                        }
                        continue ForGroupElems
                default:
                        vals = append(vals, merge(a)...)
                        continue ForGroupElems
                }
                if enable_assertions {
                        assert(len(runes) == len(names), "Flag.opts(...) error")
                }
                for _, ru := range runes {
                        switch ru {
                        case 'e':
                                if v == nil {
                                        optReportMissing = true
                                } else {
                                        optReportMissing = v.True()
                                }
                        case 'i':
                                if v == nil {
                                        optDiscardMissing = true
                                } else {
                                        optDiscardMissing = v.True()
                                }
                        case 'r':
                                if v == nil {
                                        optRecursive = true
                                } else {
                                        optRecursive = v.True()
                                }
                        }
                }
        }

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
                                } else if t.exists() {
                                        if strings.Index(t.name, "IntrinsicImpl") > 0 {
                                                fmt.Fprintf(stderr, "%s: %s\n", t, t.exists())
                                        }
                                        var list []Value
                                        list, err = prog.pc.derived.grepFiles(val, tops, rxs, optReportMissing, optDiscardMissing)
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
        if def := prog.scope.FindDef(s); def != nil {
                def.append(result...)
                result = nil
        } else {
                err = scanner.Errorf(token.Position(pos), "`%s` no such Def", s)
        }
        return
}

func modifierCompare(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var va []Value
        var optDiscardMissing, optVerbose bool
        var optPath, optNoUpdate bool
        var optGrep Value
        var opts = []string{
                "p,path",
                "e,report",
                "i,ignore",
                "n,no-time",
                "g,grep", // -grep=(regexp=(sys='...' '...') $^)
                "r,recursive",
                "v,verbose",
        }
ForArgs:
        for _, v := range args {
                var ( runes []rune ; names []string )
                switch a := v.(type) {
                case *Flag:
                        if runes, names, err = a.opts(opts...); err != nil { return }
                        v = nil // no flag value
                case *Pair:
                        if flag, ok := a.Key.(*Flag); ok && flag != nil {
                                if runes, names, err = flag.opts(opts...); err != nil { return }
                                v = a.Value // use flag value
                        } else {
                                err = scanner.Errorf(token.Position(pos), "`%v` unknown argument", a)
                                return
                        }
                default:
                        va = append(va, a)
                        continue ForArgs
                }
                if enable_assertions {
                        assert(len(runes) == len(names), "Flag.opts(...) error")
                }
                for _, ru := range runes {
                        switch ru {
                        case 'v': optVerbose = true
                        case 'i': optDiscardMissing = true
                        case 'p': optPath = true
                        case 'n': optNoUpdate = true
                        case 'g': optGrep = v
                        }
                }
        }
        args = va // reset args
        
        prog.pc.group.Add(1)
        defer prog.pc.group.Done()

        var targetDef = prog.pc.targetDef
        if n := len(args); n == 1 {
                // Change $@ to args[0]
                targetDef.set(DefDefault, args[0])
        } else if n > 1 {
                err = break_bad(pos, "two many targets (%v)", args)
                return
        }

        var target Value
        if target, err = targetDef.Call(pos); err != nil {
                err = break_bad(pos, "comparing %v: %v", targetDef, err)
                return
        } else if target == nil || target.Type() == NoneType {
                err = break_bad(pos, "`%s` target type invalid", target.Type())
                return
        }

        var targetStr string
        if targetStr, err = target.Strval(); err != nil { return }
        if optPath {
                var s string = filepath.Dir(targetStr)
                if s != "." && s != ".." && s != PathSep {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil {
                                return
                        }
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
        depends = append(depends, prog.pc.dependsDef.Value) // $^
        depends = append(depends, prog.pc.orderedDef.Value) // $|
        depends = append(depends, prog.pc.greppedDef.Value) // $~
        if grepped != nil { depends = append(depends, grepped) }
        if true { // 'false' is okay here
                depends, err = mergeresult(ExpandAll(depends...))
                if err != nil { return }
        }

        // Change into compare mode to avoid interpretion if there're
        // no targets are updated
        prog.pc.mode = compareMode

        if true {
                var c *comparer
                if c, err = newcompariation(prog, target); err == nil {
                        if optVerbose || optionVerboseChecks {
                                if true {
                                        fmt.Fprintf(stderr, "smart: Checking %s …", target)
                                } else {
                                        tar, _ := filepath.Rel(prog.project.absPath, targetStr)
                                        fmt.Fprintf(stderr, "smart: Checking %s …", tar)
                                }
                        }
                        c.nomiss, c.nocomp = optDiscardMissing, optNoUpdate
                        if err = c.Compare(pos, depends); err == nil {
                                result = universaltrue
                        } else {
                                result = universalfalse
                        }
                        if optVerbose || optionVerboseChecks {
                                if err == nil {
                                        fmt.Fprintf(stderr, "… (done)")
                                } else if br, ok := err.(*breaker); ok {
                                        switch br.what {
                                        case breakBad: fmt.Fprintf(stderr, "… Bad (%s)", br)
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
                        if len(c.updated) > 0 {
                                if enable_assertions {
                                        assert(err != nil, "expects update breaker")
                                        e, ok := err.(*breaker)
                                        assert(ok, "expects update breaker")
                                        assert(e.what == breakUpdates, "expects update breaker")
                                        assert(e.updated != nil, "nil updated target")
                                        //assert(e.updated.target == c.target, "updated target differs")
                                        //assert(len(e.updated.prerequisites) == len(c.updated), "updated target differs")
                                        //for i, preq := range e.updated.prerequisites {
                                        //       assert(preq == c.updated[i], "updated target differs")
                                        //}
                                }
                        }
                }
        } else {
                unreachable("compare in", prog.pc.mode.name(), "mode")
        }
        return
}

// grep-compare - grep files from target and compare, example usage:
//
//      (grep-compare '\s*#\s*include\s*<(.*)>')
//      
func modifierGrepCompare(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var optPath, optNoUpdate, optDisMiss bool
        var optTarget Value
        var opts = []string{
                "p,path",
                "i,ignore",
                "n,no-update",
                "t,target",
        }
        if len(args) > 0 {
                var va []Value
        ForArgs:
                for _, v := range args {
                        var ( runes []rune ; names []string )
                        switch a := v.(type) {
                        case *Flag:
                                if runes, names, err = a.opts(opts...); err != nil { return }
                                v = nil // no flag value
                        case *Pair:
                                if flag, ok := a.Key.(*Flag); ok && flag != nil {
                                        if runes, names, err = flag.opts(opts...); err != nil { return }
                                        v = a.Value // use flag value
                                } else {
                                        va = append(va, a)
                                        continue ForArgs
                                }
                        default:
                                va = append(va, a)
                                continue ForArgs
                        }
                        if enable_assertions {
                                assert(len(runes) == len(names), "Flag.opts(...) error")
                        }
                        for _, ru := range runes {
                                switch ru {
                                case 'p': optPath = true
                                case 'i': optDisMiss = true
                                case 'n': optNoUpdate = true
                                case 't': optTarget = v
                                }
                        }
                }
                args = va // reset args
        }

        var target = prog.scope.Lookup("@").(*Def)
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
                                result = universaltrue //MakeListOrScalar(c.result)
                        } else {
                                result = universalfalse
                        }
                }
        }
        return
}

// grep-dependencies - grep dependencies from target, example usage:
//
//      (grep-dependencies '\s*#\s*include\s*<(.*)>')
//      
func modifierGrepDependencies(pos Position, prog *Program, args... Value) (result Value, err error) {
        if v, e := modifierGrepFiles(pos, prog, args...); e != nil {
                err = e
        } else if v != nil {
                if false {
                        err = prog.scope.Lookup("^").(*Def).append(v)
                } else {
                        err = prog.scope.Lookup("|").(*Def).append(v)
                }
        }
        return
}

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
                targetFileName = t.FullName()
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
                        if !file.exists() {
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
                        if f, ok := file.match.Paths[0].(*Flag); ok && f.Name.Type() == NoneType {
                                sys = true
                        }
                }

                // System files are not treated as missing nor collected
                // for further updating, just discard them immediately.
                if sys { return }

                if !isAbs && !isRel && (file == nil || !file.exists()) {
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
                } else if !file.exists() && discard {
                        return
                } else {
                        result = append(result, file)
                }

                // Report missing files, but system files are not treated
                // as missing.
                if report {
                        if file == nil {
                                fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` not found\n", targetFileName, linum, colnum, p.name, name)
                        } else if !file.exists() {
                                fmt.Fprintf(stderr, "%s:%d:%d: %s: `%s` file not existed\n", targetFileName, linum, colnum, p.name, name)
                        }
                }
                return
        }

        var targetOSFile *os.File
        var savedGrepOSFile *os.File
        var savedGrepFileName string
        if s := targetName+".d"; filepath.IsAbs(s) {
                dir := filepath.Dir(s)
                s = filepath.Join("_", filepath.Base(s))
                savedGrepFileName = joinTmpPath(dir, s)
        } else if t := p.absPath; t != "" && t != "." && len(tops) > 0 {
                ////if strings.HasPrefix(s, "..") { s = filepath.Join("_", s) }
                //s = strings.Replace(s, "..", "_", -1)
                istop := func(s string) (res bool) {
                        for _, t := range tops {
                                if res = s == t; res { break }
                        }
                        return
                }

                // Change "/foo/bar/.smart/.../a/b/c/x":"a/b/c/x" into
                // "/foo/bar/.smart/...":"a/b/c/x"
                v1 := strings.Split(t, PathSep)
                v2 := strings.Split(s, PathSep)
                t2 := istop(v2[0])
                for i := len(v1)-1; i >= 0; i -= 1 {
                        if t2 && istop(v1[i]) {
                                t = filepath.Join(v1[:i]...)
                                break
                        }
                }
                savedGrepFileName = joinTmpPath(t, s)
        } else {
                //s = strings.Replace(s, "..", "_", -1)
                savedGrepFileName = joinTmpPath(t, s)
        }
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
                                                unreachable()
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
                                        if isSameAsTarget(file) { unreachable() }
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
func modifierGrepFiles(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var top []Value
        var rxs []*greprex
        var optReportMissing = true // TODO: option "e,report" 
        var optDiscardMissing = false // TODO: option "i,ignore"
        for _, arg := range args {
                var s string
                switch a := arg.(type) {
                case *Flag:
                        var opt bool
                        if opt, err = a.is('d', "discard-missing"); err != nil { return } else if opt { optDiscardMissing = opt }
                case *Pair:
                        if s, err = a.Key.Strval(); err != nil { return }
                        switch s {
                        case "sys", "system":
                                // FIXME: if a.Value.(*Group) ...
                                if s, err = a.Value.Strval(); err != nil { return }
                                rxs = append(rxs, &greprex{s, true, nil})
                        case "top":
                                if g, ok := a.Value.(*Group); ok {
                                        top = merge(g.Elems...)
                                } else {
                                        top = merge(a.Value)
                                }
                        default:
                                err = scanner.Errorf(token.Position(pos), "`%v` unsupported argument (%s)", a, s)
                                return
                        }
                default:
                        if s, err = a.Strval(); err != nil { return }
                        rxs = append(rxs, &greprex{s, false, nil})
                }
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
        var target, _ = prog.scope.Lookup("@").(*Def).Call(pos)
        list, err = prog.pc.derived.grepFiles(target, tops, rxs, optReportMissing, optDiscardMissing)
        result = MakeListOrScalar(list)
        return
}

// grep - grep  from target file, flags:
//
//    -files            grep files with stats, set $-
//    -dependencies     grep dependencies values (or files), set $^, $<, etc.
//    -compare          grep values and compare target with them
//
// Example usage:
//
//      (grep -files '\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrep(pos Position, prog *Program, args... Value) (result Value, err error) {
        panic("TODO: grep values from target file")
        return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
func modifierCheck(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var optBreak breakind // breaking with good results
        var optSilent bool // don't break on failures
        var makeResult func(bool) Value // returns results only if non-nil
        var values []Value
        var pairs []*Pair
        for _, arg := range args { switch t := arg.(type) {
        case *Pair: pairs = append(pairs, t)
        case *Flag:
                var opt bool
                if opt, err = t.is('a', "answer"); err != nil { return } else if opt { makeResult = MakeAnswer }
                if opt, err = t.is('r', "result"); err != nil { return } else if opt { makeResult = MakeBoolean }
                if opt, err = t.is('s', "silent"); err != nil { return } else if opt { optSilent = opt }
                if opt, err = t.is('g', "good"); err != nil { return } else if opt { optBreak = breakGood }
        default:
                err = scanner.Errorf(token.Position(pos), "unknown check '%v' (%T)", arg, arg)
                return
        }}

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
                                values = append(values, makeResult(res))
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
                                values = append(values, makeResult(res))
                        } else if !res {
                                err = break_with(pos, optBreak, "bad %s (%v) (expects %v)", key, v, t.Value)
                                break ForPairs
                        }
                case "file", "dir":
                        var file *File
                        var project = prog.pc.derived //mostDerived() // prog.project
                        if str, err = t.Value.Strval(); err != nil { return }
                        if file := project.searchFile(str); file == nil || !file.exists() {
                                err = break_with(pos, optBreak, "`%v` no such file or directory", t.Value)
                                break ForPairs
                        }
                        switch key {
                        case "file":
                                if res := file.info.Mode().IsRegular(); makeResult != nil {
                                        values = append(values, makeResult(res))
                                } else if !res {
                                        err = break_with(pos, optBreak, "`%v` is not a regular file", t.Value)
                                        break ForPairs
                                }
                        case "dir":
                                if res := file.info.Mode().IsDir(); makeResult != nil {
                                        values = append(values, makeResult(res))
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
                                        var def = prog.project.scope.FindDef(k)
                                        if def != nil {
                                                if a, err = p.Value.Strval(); err != nil { break ForPairs }
                                                if b, err = def.Value.Strval(); err != nil { break ForPairs }
                                                if res := a != b; makeResult != nil {
                                                        values = append(values, makeResult(res))
                                                } else if !res {
                                                        err = break_with(pos, optBreak, "`%v` != `%v`", p.Key, p.Value)
                                                        break ForPairs
                                                }
                                        } else if makeResult != nil {
                                                values = append(values, makeResult(false))
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
                result = MakeListOrScalar(values)
        }
        return
}

func modifierWriteFile(pos Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                filename, str string
                target Value
                f *os.File
        )
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        } else if filename, err = target.Strval(); err != nil {
                return
        }
        if f, err = os.Create(filename); err == nil {
                defer f.Close()

                var value Value
                if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil {
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

func modifierUpdateFile(pos Position, prog *Program, args... Value) (result Value, err error) {
        //if m := prog.pc.mode; m == compareMode {
        //        return /* not working in compare mode */
        //}
        if m := prog.pc.mode; m != updateMode {
                //return /* only configure in update mode */
        }
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var target Value
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        }

        var optPath, optVerbose bool
        var (
                nargs = len(args)
                perm = os.FileMode(0640) // sys default 0666
                filename, content string
                num int64
                f *os.File
        )

        // Process flags
        if nargs > 0 {
                var v []Value
                for _, arg := range args {
                        switch a := arg.(type) {
                        default: v = append(v, arg)
                        case *Flag:
                                var opt bool
                                if opt, err = a.is('p', "path"); err != nil { return } else if opt { optPath = opt }
                                if opt, err = a.is('v', "verbose"); err != nil { return } else if opt { optVerbose = opt }
                                if opt, err = a.is('s', "silent"); err != nil { return } else if opt { optVerbose = !opt }
                        }
                }
                args, nargs = v, len(v) // Reset args
        }

        // Get target filename
        if nargs == 0 {
                if filename, err = target.Strval(); err != nil { return }
        } else {
                target = args[0]
                if filename, err = target.Strval(); err != nil { return }
                if nargs > 1 {
                        if num, err = args[1].Integer(); err != nil { return }
                        perm = os.FileMode(num & 0777)
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
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }
        if content, err = value.Strval(); err != nil { return }

        if f, err = os.Open(filename); err == nil && f != nil {
                defer f.Close()
                if optVerbose { fmt.Fprintf(stderr, "smart: Checking %v …", target) }
                if st, _ := f.Stat(); st.Mode().Perm() != perm {
                        if err = f.Chmod(perm); err != nil {
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
                fmt.Fprintf(stderr, "smart: Updating '%v' …", filename)
        }

        // Create or update the file with new content
        
        f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
        if err == nil && f != nil {
                defer f.Close()
                if _, err = f.WriteString(content); err == nil {
                        result = stat(filename, "", "")
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

func modifierParallel(pos Position, prog *Program, args... Value) (result Value, err error) {
        return
}
