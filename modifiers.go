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

type breaker struct {
        good bool // it's good to continue
        message string
}

func (p *breaker) Error() string { return p.message }

func break_bad(s string, a... interface{}) *breaker {
        return &breaker{ false, fmt.Sprintf(s, a...) }
}

func break_good(s string, a... interface{}) *breaker {
        return &breaker{ true, fmt.Sprintf(s, a...) }
}

type modifierFunc func(pos token.Position, prog *Program, args... Value) (Value, error)

const (
        TheShellEnvarsDef = "shell->envars"
        TheShellStatusDef = "shell->status" // status code of execution
)

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
        if args, err = Disclose(args...); err != nil { return }

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
        if args, err = Disclose(args...); err != nil { return }
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        //if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        if args, err = Disclose(args...); err != nil { return }
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
                                if trace_prepare {
                                        fmt.Printf("prepare:CD: %s (%s)\n", dir, p.project.name)
                                }
                                break
                        }
                }
        }
        return
}

func modifierCD(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = Disclose(args...); err != nil { return }

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
                if trace_prepare {
                        fmt.Printf("prepare: cd %s (%s)\n", dir, prog.project.name)
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
                                        depends.Append(&File{ Name:name })
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
                } else if dependFile.Info == nil {
                        dependFile.Info, _ = os.Stat(str)
                }
                if dependFile.Info == nil {
                        err = break_bad("no file or directory '%v'", dependFile)
                        return
                }
                if t := dependFile.Info.ModTime(); t.After(tt) {
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
        if args, err = Disclose(args...); err != nil { return }

        var ( optPath bool; nargs = len(args) )
        if nargs > 0 {
                var v []Value
                for _, arg := range args {
                        switch a := arg.(type) {
                        default: v = append(v, arg)
                        case *Flag:
                                var opt bool
                                if opt, err = a.is('p', "path"); err != nil { return } else if opt { optPath = opt }
                        }
                }
                args, nargs = v, len(v) // Reset args
        }

        var target = prog.scope.Lookup("@").(*Def)
        if nargs == 1 {
                target.set(DefDefault, args[0])
        } else if nargs > 1 {
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
        if c, err = NewComparer(prog.globe, target); err == nil {
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
        //if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        if v, e := /*modifierGrepFiles*/modifierGrepDependencies(pos, prog, args...); e != nil {
                err = e
        } else if v != nil {
                /*
                var def = prog.scope.Lookup("^").(*Def)
                defer def.set(def.origin, def.Value)
                if err = def.set(DefDefault, v); err == nil {
                        result, err = modifierCompare(pos, prog)
                }
                */
                result, err = modifierCompare(pos, prog)
        }
        return
}

// grep-dependencies - grep dependencies, example usage:
//
//      (grep-dependencies '\s*#\s*include\s*<(.*)>')
//      
func modifierGrepDependencies(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        //if args, err = mergeresult(ExpandAll(args...)); err != nil { return }
        if v, e := modifierGrepFiles(pos, prog, args...); e != nil {
                err = e
        } else if v != nil {
                var def = prog.scope.Lookup("^").(*Def)
                err = def.set(DefDefault, v)
        }
        return
}

// grep-files - grep files from target, example usage:
//
//      (grep-files '\s*#\s*include\s*<(.*)>')
//      
// https://github.com/google/re2/wiki/Syntax
func modifierGrepFiles(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = mergeresult(ExpandAll(args...)); err != nil { return }

        var (
                optDiscardMissing = false

                target, _ = prog.scope.Lookup("@").(Caller).Call(pos)
                sys = make(map[*regexp.Regexp]bool)
                rxs []*regexp.Regexp
        )

        for _, arg := range args {
                var ( x *regexp.Regexp ; s string )
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
                                if x, err = regexp.Compile(s); err != nil { return }
                                rxs = append(rxs, x)
                                sys[x] = true
                        default:
                                err = scanner.Errorf(pos, "`%v` unsupported argument (%s)", a, s)
                                return
                        }
                default:
                        if s, err = a.Strval(); err != nil { return }
                        if x, err = regexp.Compile(s); err != nil { return }
                        rxs = append(rxs, x)
                        sys[x] = false
                }
        }

        if len(rxs) == 0 {
                err = scanner.Errorf(pos, "no grep expressions")
                return
        }

        var ( targetFileName, targetDir string ; targetFile *os.File )
        if targetFileName, err = target.Strval(); err != nil {
                return
        } else if targetFile, err = os.Open(targetFileName); err != nil {
                if optDiscardMissing { err = nil }
                return
        } else {
                defer func() { err = targetFile.Close() } ()
        }

        targetDir = filepath.Dir(targetFileName)

        var ( list []Value ; linum int )
        project := mostDerived() // prog.project
        scanner := bufio.NewScanner(targetFile)
        scanner.Split(bufio.ScanLines)
ForScan:
        for scanner.Scan() {
                linum += 1
                var s = scanner.Text()
                for _, x := range rxs {
                        if sm := x.FindStringSubmatch(s); len(sm) > 1 && sm[1] != "" {
                                var ( name = sm[1] ; file = &File{ Name:name } )
                                var colnum = strings.Index(s, name) //strings.IndexFunc(s, isNotSpace)
                                var yes, ok = sys[x] // System files defined by `sys=xxx` arguments
                                if project.searchInDir(file, targetDir, name, ok && yes) {
                                        if file.Info == nil { unreachable() }
                                        list = append(list, file)
                                        continue ForScan
                                } else if ok && yes {
                                        // Missing system files is okay.
                                        continue ForScan
                                } else if !(filepath.IsAbs(name) || isRelPath(name)) {
                                        // System files defined by `files ((foo.xxx) => -)`
                                        if file.Match != nil && len(file.Match.Paths) == 1 {
                                                if f, ok := file.Match.Paths[0].(*Flag); ok && f.Name.Type() == NoneType {
                                                        continue ForScan
                                                }
                                        }

                                        // If it's not found in files database.
                                        if file.Match == nil || file.Sub == nil {
                                                //continue ForScan
                                        }

                                        if optDiscardMissing {
                                                //continue ForScan
                                        }
                                }

                                if true {
                                        fmt.Fprintf(os.Stderr, "%s:%d:%d: `%s` not found (project %s)\n", targetFileName, linum, colnum, name, project.name)
                                } else {
                                        fmt.Fprintf(os.Stderr, "%s:%d:%d: `%s` not found (project %s) (%v)\n", targetFileName, linum, colnum, name, project.name, file.Match)
                                }
                                continue ForScan // break ForScan
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
        if args, err = Disclose(args...); err != nil { return }

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

        ForPairs: for _, t := range pairs {
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
                        var fi os.FileInfo
                        if str, err = t.Value.Strval(); err != nil { return }
                        if fi, err = os.Stat(str); err != nil || fi == nil {
                                err = &breaker{ optGood, fmt.Sprintf("`%v` no such file or directory", t.Value) }
                                break ForPairs
                        }
                        switch key {
                        case "file":
                                if res := fi.Mode().IsRegular(); makeResult != nil {
                                        values = append(values, makeResult(res))
                                } else if !res {
                                        err = &breaker{ optGood, fmt.Sprintf("`%v` is not a regular file", t.Value) }
                                        break ForPairs
                                }
                        case "dir":
                                if res := fi.Mode().IsDir(); makeResult != nil {
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
        if args, err = Disclose(args...); err != nil { return }

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
                        result = &File{ Name: filename }
                } else {
                        os.Remove(filename)
                }
        } else {
                err = break_bad("file %s not generated", target)
        }
        return
}

func modifierUpdateFile(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if args, err = Disclose(args...); err != nil { return }

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
                        if false {
                                fmt.Printf("%s already up to date\n", filename)
                        }
                        result = &File{ Name: filename }
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
                        result = &File{ Name: filename }
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
