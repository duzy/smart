//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
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

                `args`:         modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments

                `unclose`:      modifierUnclose,

                `cd`:           modifierCD,
                `sudo`:         modifierSudo,

                `compare`:         modifierCompare,
                `grep-compare`:    modifierGrepCompare,
                `grep-dependents`: modifierGrepDependents,

                `check`:        modifierCheck,
                //`check-dir`:    modifierCheckDir,
                //`check-file`:   modifierCheckFile,
                //`dir-p`:        modifierCheckDir,
                //`file-p`:       modifierCheckFile,
                
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

func modifierStatusEquals(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var v = getGroupElem(value, 1, nil)
        if v != nil && len(args) == 1 {
                var a, s string
                if a, err = args[0].Strval(); err != nil {
                        return
                } else if s, err = v.Strval(); err != nil {
                        return
                } else if s == a {
                        result = v; return
                }
        }
        err = break_bad("bad status (%v)", v)
        return
}

func modifierStdoutEquals(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var v = getGroupElem(value, 2, nil)
        if v != nil && len(args) == 1 {
                var a, s string
                if a, err = args[0].Strval(); err != nil {
                        return
                } else if s, err = v.Strval(); err != nil {
                        return
                } else if s == a {
                        result = v; return
                }
        }
        err = break_bad("bad stdout (%v)", v)
        return
}

func modifierStderrEquals(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var v = getGroupElem(value, 3, nil)
        if v != nil && len(args) == 1 {
                var a, s string
                if a, err = args[0].Strval(); err != nil {
                        return
                } else if s, err = v.Strval(); err != nil {
                        return
                } else if s == a {
                        result = v; return
                }
        }
        err = break_bad("bad stderr (%v)", v)
        return
}

func modifierSelect(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        //if args, err = ExpandAll(Merge(args...)...); err != nil {
        if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        }
        var envars = new(List)
        _ = prog.auto(TheShellEnvarsDef, envars)
        for _, a := range args {
                if _, ok := a.(*Pair); ok {
                        //fmt.Printf("env: %v\n", a)
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

func modifierCD(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var (optPrint, optPath bool; n = len(args))
        var target, _ = prog.scope.Lookup("@").(*Def).Call(pos)
        if _, ok := target.(*Flag); ok { optPrint = false }
        if n > 0 {
                var v []Value
                for _, arg := range args {
                        switch a := arg.(type) {
                        default: v = append(v, arg)
                        case *Flag:
                                var opt bool
                                if opt, err = a.is('p', "path"); err != nil { return } else if opt { optPath = opt }
                                if opt, err = a.is('s', "silent"); err != nil { return } else if opt { optPrint = !opt }
                                if opt, err = a.is(0, ""); err != nil { return } else if opt {
                                        var dir string
                                        if len(execstack) > 1 {
                                                // Find a backtrack.
                                                top := execstack[0]
                                                for _, p := range execstack[1:] {
                                                        if p.project != top.project {
                                                                dir = p.project.AbsPath()
                                                                if trace_prepare {
                                                                        fmt.Printf("prepare:CD: %s (%s) (%s)\n", dir, p.project.name, prog.project.name)
                                                                }
                                                                break
                                                        }
                                                }
                                        }
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

        if n = len(args); n == 1 {
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
                if optPath && dir != "." && dir != PathSep {// mkdir -p
                        if err = os.MkdirAll(dir, os.FileMode(0755)); err != nil {
                                return
                        }
                }
                if err = prog.cd(dir, optPrint, false); err == nil {
                        //for _, cd := range prog.cdinfos[1:] { cd.print = false }
                }
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
        //fmt.Printf("compare: %v -> %v (%v)\n", target, depend, prog.context.outdated)
        //fmt.Printf("compare: %v: %v (%T)\n", target, depend, depend)
        if dependFile, okay := depend.(*File); okay && dependFile != nil {
                var str string
                if str, err = dependFile.Strval(); err != nil { return }
                if t, ok := prog.globe.Timestamps[str]; ok && t.After(tt) {
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
                        prog.globe.Timestamps[str] = t
                        outdated = true; return // target is outdated
                } else {
                        var (
                                recipes []Value
                                strings []string
                        )
                        if recipes, err = DiscloseAll(prog.recipes...); err != nil {
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

func modifierCompare_0(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var (
                tardef = prog.scope.Lookup("@").(*Def)
                target Value
                depends = new(List)
                nargs   = len(args)
        )
        if nargs == 0 {
                if target, err = tardef.Call(pos); err != nil { return }
        } else if nargs == 1 {
                tardef.Assign(target)
                if target, err = tardef.Call(pos); err != nil { return }
        } else if nargs > 1 {
                s := fmt.Sprintf("accepts only one optional argument (%v)", args)
                return nil, break_bad(s)
        }

        if trace_compare {
                fmt.Printf("compare:Target: %v\n", target)
        }

        if target, err = target.expand(expandDelegate); err != nil { return }
        if target == nil || target.Type() == NoneType {
                err = break_bad("no target"); return
        }

        // deal with list.
        switch t := target.(type) {
        case *List:
                if n := t.Len(); n == 1 {
                        target = t.Elems[0]
                } else {
                        s := fmt.Sprintf("compare: multiple targets (%v)", target)
                        err = break_bad(s); return
                }
        }

        if depends, err = getCompareDepends(pos, prog); err != nil {
                return
        } else if depends != nil && depends.Len() > 0 {
                prog.auto("<", depends.Get(0))
                prog.auto("^", depends)
        }

        // Comparing target with depends.

        var (
                outdated = false
                tt time.Time
        )
        if target != nil && target.Type() != NoneType {
                var (
                        targetFile *File
                        name string
                )
                if name, err = target.Strval(); err != nil { return }
                switch t := target.(type) {
                case *File: targetFile = t
                case *Barefile: 
                        targetFile = &File{ Name:name }
                case *RuleEntry:
                        switch class := t.class; class {
                        //case ExplicitFileEntry, StemmedFileEntry:
                        //        targetFile = t.file //&File{ Name:name }
                        default:/*if p := t.Project(); p != nil {
                                if p.IsFile(s) {
                                        targetFile = &File{ Name:name }
                                } else {
                                        fmt.Fprintf(os.Stdout, "%v: %v->%v is not file\n", t.Position, p.Name(), s)
                                        err = break_bad("unknown %v->%v (%v)", p.Name(), s, class)
                                        return
                                }
                        } else*/ {
                                //err = break_bad("unknown entry (%v '%v')", class, name)
                                //return
                        }}
                }
                if targetFile == nil {
                        err = break_bad("unknown target (%T '%v')", target, target)
                        return
                } else if nargs == 1 {
                        // Replace the value of "$@"
                        def := prog.scope.Lookup("@").(*Def)
                        def.Assign(targetFile)
                }

                // In case passing a unstated target file.
                if targetFile.Info == nil && targetFile != nil {
                        var str string
                        if str, err = targetFile.Strval(); err != nil { return }
                        targetFile.Info, _ = os.Stat(str)
                }
                if fi := targetFile.Info; fi != nil {
                        tt = fi.ModTime()
                } else {
                        outdated = true; return
                }
        }
        if depends != nil {
                for _, depend := range depends.Elems {
                        switch depend.(type) {
                        case *File:
                                outdated, err = compareTargetDepend(pos, prog, target, depend, tt)
                                //fmt.Printf("compare-target-depend: %v (%v)\n", outdated, err)
                                if err != nil || outdated {
                                        return
                                }
                        }
                }
        }
        err = break_good("%s already up to date", target)
        return
}

func modifierCompare(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var target = prog.scope.Lookup("@").(*Def)
        if nargs := len(args); nargs == 1 {
                target.Assign(args[0])
        } else if nargs > 1 {
                s := fmt.Sprintf("accepts only one optional argument (%v)", args)
                return nil, break_bad(s)
        }

        if c, e := NewComparer(prog.globe, target); e != nil {
                err = e
        } else if err = c.Compare(prog.scope.Lookup("^")); err == nil {
                result = MakeListOrScalar(c.result)
        }
        return
}

// grep-compare - grep dependencies and compare, example usag:
//
//      (grep-compare '\s*#\s*include\s*<(.*)>')
//      
func modifierGrepCompare(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        if v, e := modifierGrepDependents(pos, prog, args...); e != nil {
                err = e; return
        } else if v != nil {
                def := prog.scope.Lookup("^").(*Def)
                old := def.Value
                def.Assign(v)
                result, err = modifierCompare(pos, prog, args...)
                def.Assign(old)
        }
        return
}

// https://github.com/google/re2/wiki/Syntax
func modifierGrepDependents(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var (
                target, _ = prog.scope.Lookup("@").(Caller).Call(pos)
                targetName string
                rxs []*regexp.Regexp
                optDiscardMissing = false
                f *os.File
        )
        if targetName, err = target.Strval(); err != nil { return }
        if len(args) == 0 {
                err = errors.New("no arguments provided"); return
        } else if args, err = mergeresult(ExpandAll(args...)); err != nil {
                return
        }

        for _, arg := range args {
                switch a := arg.(type) {
                case *Flag:
                        var opt bool
                        if opt, err = a.is('d', "discard-missing"); err != nil { return } else if opt { optDiscardMissing = opt }
                case *Pair:
                        //fmt.Printf("todo: grep-dependents: %v\n", a)
                default:
                        var (
                                str string
                                x *regexp.Regexp
                        )
                        if str, err = a.Strval(); err != nil {
                                return
                        } else if x, err = regexp.Compile(str); err != nil {
                                return
                        } else {
                                rxs = append(rxs, x)
                        }
                }
        }
        
        if f, err = os.Open(targetName); err != nil {
                if false {
                        s, _ := os.Getwd()
                        fmt.Fprintf(os.Stderr, "grep-files: %v (%v)\n", err, s)
                }
                if optDiscardMissing {
                        err = nil
                }
                return
        } else {
                defer func() { err = f.Close() }()
        }

        project := prog.project
        /*if p := context.FindProject(); p != nil {
                project = p
        }*/

        // if vargs[0] == '-c' { ... }
        dependList := new(List)
        scanner := bufio.NewScanner(f)
        scanner.Split(bufio.ScanLines)
        for scanner.Scan() {
                s := scanner.Text()
                for _, x := range rxs { //if x.MatchString(s) {
                        if sm := x.FindStringSubmatch(s); len(sm) == 2 && sm[1] != "" {
                                v := project.SearchFile(sm[1])
                                if v.Info == nil && optDiscardMissing {
                                        continue
                                }
                                //fmt.Printf("todo: %v %v\n", v, v.Info.Name())
                                dependList.Append(v)
                                break
                        }
                }
        }
        result = dependList
        return
}

// (check status=1 stdout="foobar" stderr="")
// (check file=filename.txt)
// (check dir=directory)
func modifierCheck(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var value Value
        if value, err = prog.scope.Lookup("-").(*Def).Call(pos); err != nil { return }

        var exeres, _ = value.(*ExecResult)
        if exeres == nil {
                err = break_bad("bad value (%T)", value)
                return
        }

        var (
                optSilent bool // don't break on failures
                makeResult func(bool) Value // returns results only if non-nil
                values []Value
                pairs []*Pair
        )
        for _, arg := range args { switch t := arg.(type) {
        case *Pair: pairs = append(pairs, t)
        case *Flag:
                var opt bool
                if opt, err = t.is('a', "answer"); err != nil { return } else if opt { makeResult = MakeAnswer }
                if opt, err = t.is('r', "result"); err != nil { return } else if opt { makeResult = MakeBoolean }
                if opt, err = t.is('s', "silent"); err != nil { return } else if opt { optSilent = opt }
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
                        var num int64
                        if num, err = t.Value.Integer(); err != nil { return }
                        if res := exeres.Status == int(num); makeResult != nil {
                                values = append(values, makeResult(res))
                        } else if !res {
                                s := "bad status (%v) (expects %v)"
                                err = break_bad(s, exeres.Status, t.Value)
                                break ForPairs
                        }
                case "stdout", "stderr":
                        var v *bytes.Buffer
                        switch key {
                        case "stdout": v = exeres.Stdout.Buf
                        case "stderr": v = exeres.Stderr.Buf
                        default: unreachable()
                        }
                        if v == nil {
                                s := "bad %s (expects %v)"
                                err = break_bad(s, key, t.Value)
                                break ForPairs
                        }
                        if str, err = t.Value.Strval(); err != nil { 
                                return
                        } else if res := v.String() == str; makeResult != nil {
                                values = append(values, makeResult(res))
                        } else if !res {
                                s := "bad %s (%v) (expects %v)"
                                err = break_bad(s, key, v, t.Value)
                                break ForPairs
                        }
                case "file", "dir":
                        if str, err = t.Value.Strval(); err != nil {
                                return
                        } else if fi, er := os.Stat(str); er != nil || fi == nil {
                                s := "`%v` no such file or directory"
                                err = break_bad(s, t.Value)
                                break ForPairs
                        } else {
                                switch key {
                                case "file":
                                        if res := fi.Mode().IsRegular(); makeResult != nil {
                                                values = append(values, makeResult(res))
                                        } else if !res {
                                                s := "`%v` is not a regular file"
                                                err = break_bad(s, t.Value)
                                                break ForPairs
                                        }
                                case "dir":
                                        if res := fi.Mode().IsDir(); makeResult != nil {
                                                values = append(values, makeResult(res))
                                        } else if !res {
                                                s := "`%v` is not a directory"
                                                err = break_bad(s, t.Value)
                                                break ForPairs
                                        }
                                default: unreachable()
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

func modifierCheckDir(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var (
                target Value
                filename string
        )
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        } else if filename, err = target.Strval(); err != nil {
                return
        }
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsDir() {
                var segments []Value
                for _, seg := range filepath.SplitList(filename) {
                        segments = append(segments, &String{seg})
                }
                result = &Path{Elements{segments}, nil}
        } else {
                err = break_bad("file %s not exists", target)
        }
        return
}

func modifierCheckFile(pos token.Position, prog *Program, args... Value) (result Value, err error) {
        var (
                target Value
                filename string
        )
        if target, err = prog.scope.Lookup("@").(*Def).Call(pos); err != nil {
                return
        } else if filename, err = target.Strval(); err != nil {
                return
        }
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsRegular() {
                result = &File{ Name: filename }
        } else {
                err = break_bad("file %s not exists", target)
        }
        return
}

func modifierWriteFile(pos token.Position, prog *Program, args... Value) (result Value, err error) {
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
