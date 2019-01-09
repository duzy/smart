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

        enable_grep_bench = true
)

type breaker struct {
        good bool // it's good to continue
        message string
        //values []Value
}

func (p *breaker) Error() string { return p.message }

func break_bad(s string, a... interface{}) *breaker {
        return &breaker{ false, fmt.Sprintf(s, a...) }
}

func break_good(s string, a... interface{}) *breaker {
        return &breaker{ true, fmt.Sprintf(s, a...) }
}

type modifierFunc func(pos token.Position, prog *Program, args... Value) (Value, error)

var (
        modifiers = map[string]modifierFunc{
                `select`:       modifierSelect,

                //`args`:         modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments

                `unclose`:      modifierUnclose,

                `cd`:           modifierCD,
                `sudo`:         modifierSudo,

                `compare`:           modifierCompare,
                `grep`:              modifierGrep,
                `grep-files`:        modifierGrepFiles,
                `grep-compare`:      modifierGrepCompare,
                `grep-dependencies`: modifierGrepDependencies,

                `check`:        modifierCheck,
                
                `write-file`:   modifierWriteFile,
                `update-file`:  modifierUpdateFile,
                `configure-file`: modifierConfigureFile,

                `configure`: modifierConfigure,
                `extract-configuration`: modifierExtractConfiguration,
        }

        crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)

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
                                                fmt.Printf("%s", str)
                                        } else if str != "" {
                                                fmt.Printf("%s\n", str)
                                        }
                                }
                        }
                }
        }
        return
}

func modifierSelect(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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

func modifierSetArgs(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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

func modifierUnclose(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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

func modifierCD(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (optPath bool; n = len(args))
        //var target, _ = prog.scope.Lookup("@").(*Def).Call(pos)
        //if _, ok := target.(*Flag); ok { optPrint = false }
        if n > 0 {
                var v []Value
                for _, arg := range args {
                        switch a := arg.(type) {
                        default: v = append(v, arg)
                        case *Flag:
                                var opt bool
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
                args, n = v, len(v) // Reset args
        }
        if n == 1 {
                var dir string
                if dir, err = args[0].Strval(); err != nil {
                        return
                } else if dir == "" {
                        err = fmt.Errorf("no trackback (tracks=%v)", len(execstack))
                        return
                }
                if optPath && dir != "." && dir != ".." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil { return }
                }
                err = enter(prog, dir)
        } else {
                err = fmt.Errorf("wrong number of args (%v)", n)
        }
        return
}

func modifierSudo(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        panic("todo: sudo modifier is not implemented yet")
        return
}

func parseDependList(pos token.Position, prog *Program, dependList *List) (depends *List, err error) {
        depends = new(List)
        for _, depend := range dependList.Elems {
                if trace_compare {
                        fmt.Printf("compare:Depend: %v (%T)\n", depend, depend)
                }
                switch d := depend.(type) {
                case *List:
                        if dl, e := parseDependList(pos, prog, d); e != nil {
                                err = e; return
                        } else {
                                depends.Elems = append(depends.Elems, dl.Elems...)
                        }
                case *ExecResult:
                        if d.Status != 0 {
                                err = break_bad("got shell failure")
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
                                err = fmt.Errorf("unsupported entry depend `%v' (%v)", d, d.Class())
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
                        err = fmt.Errorf("unsupported entry depend `%v' (%v)", depend, prog.depends)
                }
        }
        return
}

func getCompareDepends(pos token.Position, prog *Program) (depends *List, err error) {
        def := prog.scope.Lookup("^").(*Def)
        dependVal, _ := def.Call(pos)
        if dependVal, err = dependVal.expand(expandDelegate); err != nil { return }
        if dependList, _ := dependVal.(*List); dependList != nil && dependList.Len() > 0 {
                depends, err = parseDependList(pos, prog, dependList)
        }
        return
}

func compareTargetDepend(pos token.Position, prog *Program, target, depend Value, tt time.Time) (outdated bool, err error) {
        if dependFile, okay := depend.(*File); okay && dependFile != nil {
                var str string
                if str, err = dependFile.Strval(); err != nil { return }
                if t, ok := prog.globe.timestamps[str]; ok && t.After(tt) {
                        outdated = true; return // target is outdated
                } else if dependFile.info == nil {
                        dependFile.info, _ = os.Stat(str)
                }
                if dependFile.info == nil {
                        err = break_bad("no file or directory '%v'", dependFile)
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
                        //fmt.Printf("compare: %v\n", ent)
                }
        } else {
                fmt.Printf("compare: todo: %v -> %v (%T)\n", target, depend, depend)
        }
        return
}

func modifierCompare(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var optPath, optNoUpdate, optDisMiss bool
        var opts = []string{
                "p,path",
                "i,ignore",
                "n,no-update",
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
                                        err = fmt.Errorf("`%v` unknown argument", a)
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
                                case 'p': optPath = true
                                case 'i': optDisMiss = true
                                case 'n': optNoUpdate = true
                                }
                        }
                }
                args = va // reset args
        }

        var target = prog.scope.Lookup("@").(*Def)
        if n := len(args); n == 1 {
                target.set(DefDefault, args[0])
        } else if n > 1 {
                err = break_bad("two many targets (%v)", args)
                return
        }

        if optPath && target.Value != nil {
                var s string
                if s, err = target.Value.Strval(); err != nil { return }
                if s = filepath.Dir(s); s != "." && s != ".." && s != PathSep {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil { return }
                }
        }

        var c *comparer
        if c, err = newcompariation(prog.globe, target); err == nil {
                c.nomiss = optDisMiss
                c.noexec = optNoUpdate
                if err = c.Compare(prog.scope.Lookup("^")); err == nil {
                        result = MakeListOrScalar(c.result)
                }
        }
        return
}

// grep-compare - grep dependencies and compare, example usage:
//
//      (grep-compare '\s*#\s*include\s*<(.*)>')
//      
func modifierGrepCompare(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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
        if optTarget != nil { target.set(DefDefault, optTarget) }
        if optPath && target.Value != nil {
                var s string
                if s, err = target.Value.Strval(); err != nil { return }
                if s = filepath.Dir(s); s != "." && s != ".." && s != PathSep {
                        if err = os.MkdirAll(s, os.FileMode(0755)); err != nil { return }
                }
        }

        if v, e := modifierGrepFiles(pos, prog, args...); e != nil {
                err = e
        } else if /*v != nil*/false {
                var def = prog.scope.Lookup("^").(*Def)
                defer def.set(def.origin, def.Value)
                if err = def.set(DefDefault, v); err == nil {
                        result, err = modifierCompare(pos, prog)
                }
        } else if v != nil {
                var c *comparer
                if c, err = newcompariation(prog.globe, target); err == nil {
                        c.nomiss = optDisMiss
                        c.noexec = optNoUpdate
                        if err = c.Compare(v); err == nil {
                                result = MakeListOrScalar(c.result)
                        }
                }
        }
        return
}

// grep-dependencies - grep dependencies, example usage:
//
//      (grep-dependencies '\s*#\s*include\s*<(.*)>')
//      
func modifierGrepDependencies(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if v, e := modifierGrepFiles(pos, prog, args...); e != nil {
                err = e
        } else if v != nil {
                err = prog.scope.Lookup("^").(*Def).append(v)
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

// grep-files - grep files from target, example usage:
//
//      (grep-files '\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrepFiles(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var project = mostDerived() // prog.project
        var target, _ = prog.scope.Lookup("@").(*Def).Call(pos)
        var targetName, targetFileName string
        switch t := target.(type) {
        case *File:
                targetName = t.name
                targetFileName = t.FullName()
        default:
                targetName, err = t.Strval()
                targetFileName = targetName
        }
        if err != nil { return }

        var ( list []Value ; cached bool )
        if list, cached = grepcache[targetFileName]; cached {
                result = MakeListOrScalar(list)
                return
        } else {
                defer func() { grepcache[targetFileName] = list } ()
        }

        var optReportMissing = true
        var optDiscardMissing = false

        type rxty struct{ string ; bool ; *regexp.Regexp }
        var rxs []*rxty
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
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
                                rxs = append(rxs, &rxty{s, true, nil})
                        default:
                                err = scanner.Errorf(pos, "`%v` unsupported argument (%s)", a, s)
                                return
                        }
                default:
                        if s, err = a.Strval(); err != nil { return }
                        rxs = append(rxs, &rxty{s, false, nil})
                }
        }
        if len(rxs) == 0 {
                err = scanner.Errorf(pos, "no grep expressions")
                return
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

        var targetDir = filepath.Dir(targetFileName)
        var searchName = func(sys bool, linum, colnum int, name string) (file *File) {
                if file = project.searchInDir(targetDir, name, sys); file != nil {
                        if file.info == nil { unreachable() }
                        if !isSameAsTarget(file) {
                                list = append(list, file)
                        }
                        return
                } else if sys {
                        return // system files are not missing
                } else if file == nil {
                        // file not found
                } else if file.match != nil && len(file.match.Paths) == 1 {
                        // system files defined by `files ((foo.xxx) => -)`
                        if f, ok := file.match.Paths[0].(*Flag); ok && f.Name.Type() == NoneType {
                                return
                        }
                } else if !optDiscardMissing && !isSameAsTarget(file) {
                        // FIXME: file is nil if it's not found by searchInDir
                        list = append(list, file) // add missing files
                }

                if optReportMissing {
                        fmt.Fprintf(os.Stderr, "%s:%d:%d: `%s` not found (project %s)\n", targetFileName, linum, colnum, name, project.name)
                }

                /* if filepath.IsAbs(name) || isRelPath(name) {
                        // If it's absolute or relative.
                } else if file.match == nil {
                        // If it's not found in files database.
                } */
                return
        }

        var targetOSFile *os.File
        var savedGrepOSFile *os.File
        var savedGrepFileName string
        if s := targetName+".d"; filepath.IsAbs(s) {
                dir := filepath.Dir(s)
                s = filepath.Join("_", filepath.Base(s))
                savedGrepFileName = joinTmpPath(dir, s)
        } else {
                if strings.HasPrefix(s, "..") { s = filepath.Join("_", s) }
                // TODO: deal with foo/bar/name.xxx
                savedGrepFileName = joinTmpPath(project.absPath, s)
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
                        result = MakeListOrScalar(list)
                        file1.info, _ = savedGrepOSFile.Stat()
                        return
                }
        }

GrepTargetFile:
        if targetOSFile, err = os.Open(targetFileName); err != nil {
                if optDiscardMissing { err = nil }
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
                for _, v := range list {
                        if v == nil {
                                unreachable()
                        }
                        if f, ok := v.(*File); ok && f == nil {
                                unreachable()
                        }
                }
        }
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
func modifierGrep(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        panic("TODO: grep values from target file")
        return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
// (check var=(NAME,VALUE))
func modifierCheck(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var optGood bool // breaking with good results
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
                if opt, err = t.is('g', "good"); err != nil { return } else if opt { optGood = opt }
        default:
                err = fmt.Errorf("unknown check '%v' (%T)", arg, arg)
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
                                err = &breaker{ optGood, fmt.Sprintf("not an exec result (%T)", value) }
                                return
                        }

                        var num int64
                        if num, err = t.Value.Integer(); err != nil { return }
                        if res := exeres.Status == int(num); makeResult != nil {
                                values = append(values, makeResult(res))
                        } else if !res {
                                err = &breaker{ optGood, fmt.Sprintf("bad status (%v) (expects %v)", exeres.Status, t.Value) }
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var exeres, _ = value.(*ExecResult)
                        if exeres == nil {
                                err = &breaker{ optGood, fmt.Sprintf("not an exec result (%T)", value) }
                                return
                        }

                        var v *bytes.Buffer
                        switch key {
                        case "stdout": v = exeres.Stdout.Buf
                        case "stderr": v = exeres.Stderr.Buf
                        default: unreachable()
                        }

                        if v == nil {
                                err = &breaker{ optGood, fmt.Sprintf("bad %s (expects %v)", key, t.Value) }
                                break ForPairs
                        }
                        if str, err = t.Value.Strval(); err != nil { 
                                return
                        } else if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(res))
                        } else if !res {
                                err = &breaker{ optGood, fmt.Sprintf("bad %s (%v) (expects %v)", key, v, t.Value) }
                                break ForPairs
                        }
                case "file", "dir":
                        var file *File
                        var project = mostDerived() // prog.project
                        if str, err = t.Value.Strval(); err != nil { return }
                        if file := project.search(str); file == nil || !file.exists() {
                                err = &breaker{ optGood, fmt.Sprintf("`%v` no such file or directory", t.Value) }
                                break ForPairs
                        }
                        switch key {
                        case "file":
                                if res := file.info.Mode().IsRegular(); makeResult != nil {
                                        values = append(values, makeResult(res))
                                } else if !res {
                                        err = &breaker{ optGood, fmt.Sprintf("`%v` is not a regular file", t.Value) }
                                        break ForPairs
                                }
                        case "dir":
                                if res := file.info.Mode().IsDir(); makeResult != nil {
                                        values = append(values, makeResult(res))
                                } else if !res {
                                        err = &breaker{ optGood, fmt.Sprintf("`%v` is not a directory", t.Value) }
                                        break ForPairs
                                }
                        default: unreachable()
                        }
                case "var":
                        g, ok := t.Value.(*Group)
                        if !ok {
                                err = &breaker{ optGood, fmt.Sprintf("`%v` is not a group value", t.Value) }
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
                                                        err = &breaker{ optGood, fmt.Sprintf("`%v` != `%v`", p.Key, p.Value) }
                                                        break ForPairs
                                                }
                                        } else if makeResult != nil {
                                                values = append(values, makeResult(false))
                                        } else {
                                                err = &breaker{ optGood, fmt.Sprintf("`%v` is not defined", k) }
                                                break ForPairs
                                        }
                                default:
                                        err = &breaker{ optGood, fmt.Sprintf("`%v` unsupported checks", elem) }
                                        break ForPairs
                                }
                        }
                default:
                        err = fmt.Errorf("unknown check '%v'", t.Key)
                        break ForPairs
                }
        }
        if err == nil && values != nil {
                result = MakeListOrScalar(values)
        }
        return
}

func modifierWriteFile(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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
                err = break_bad("file %s not generated", target)
        }
        return
}

func modifierUpdateFile(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var target Value
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        }

        var (
                nargs = len(args)
                perm = os.FileMode(0640) // sys default 0666
                filename, content string
                optPath, optSilent bool
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
                                if opt, err = a.is('s', "silent"); err != nil { return } else if opt { optSilent = opt }
                        }
                }
                args, nargs = v, len(v) // Reset args
        }

        // Get target filename
        if nargs == 0 {
                if filename, err = target.Strval(); err != nil { return }
        } else {
                if filename, err = args[0].Strval(); err != nil { return }
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
                if st, _ := f.Stat(); st.Mode().Perm() != perm {
                        if err = f.Chmod(perm); err != nil {
                                return
                        }
                }
                w1 := crc64.New(crc64Table)
                w2 := crc64.New(crc64Table)
                if _, err = io.Copy(w1, f); err != nil {
                        return
                }
                if _, err = io.WriteString(w2, content); err != nil {
                        return
                }
                if w1.Sum64() == w2.Sum64() {
                        result = stat(filename, "", "")
                        return
                }
        }

        if !optSilent {
                printEnteringDirectory()
                fmt.Printf("update file '%v' …", filename)
        }

        // Create or update the file with new content
        
        f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
        if err == nil && f != nil {
                defer f.Close()
                if _, err = f.WriteString(content); err == nil {
                        result = stat(filename, "", "")
                        if !optSilent { fmt.Printf("… (ok)\n") }
                } else {
                        os.Remove(filename)
                        if !optSilent { fmt.Printf("… (%s)\n", err) }
                }
        } else {
                if !optSilent { fmt.Printf("… (%s)\n", err) }
                err = break_bad("file %s not updated", target)
        }
        return
}
