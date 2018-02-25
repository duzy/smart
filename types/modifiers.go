//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "github.com/duzy/smart/token"
        "path/filepath"
        "hash/crc64"
        "strings"
        "regexp"
        "errors"
        //"bytes"
        "bufio"
        "time"
        "fmt"
        "os"
        "io"
)

type breaker struct {
        message string
        good bool // it's good to continue
}

func (p *breaker) Error() string {
        return p.message
}

func breakf(good bool, s string, a... interface{}) *breaker {
        return &breaker{ fmt.Sprintf(s, a...), good }
}

type modifierFunc func(pos token.Position, prog *Program, value Value, args... Value) (Value, error)

const (
        TheShellEnvarsDef = "shell->envars"
        TheShellStatusDef = "shell->status" // status code of execution
        TheCurrWorkDirDef = "CWD"
)

var (
        modifiers = map[string]modifierFunc{
                `select`:       modifierSelect,

                `args`:         modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments

                `cd`:           modifierCD,

                `compare`:         modifierCompare,
                `grep-compare`:    modifierGrepCompare,
                `grep-dependents`: modifierGrepDependents,

                `check`:        modifierCheck,
                `check-dir`:    modifierCheckDir,
                `check-file`:   modifierCheckFile,
                `dir-p`:        modifierCheckDir,
                `file-p`:       modifierCheckFile,
                
                `write-file`:   modifierWriteFile,
                `update-file`:  modifierUpdateFile,
        }

        crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)

func IsModifier(s string) (ok bool) {
        _, ok = modifiers[s]
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

func modifierStatusEquals(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
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
        err = breakf(false, "bad status (%v)", v)
        return
}

func modifierStdoutEquals(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        v := getGroupElem(value, 2, nil)
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
        err = breakf(false, "bad stdout (%v)", v)
        return
}

func modifierStderrEquals(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        v := getGroupElem(value, 3, nil)
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
        err = breakf(false, "bad stderr (%v)", v)
        return
}

func modifierSelect(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        if g, ok := value.(*Group); ok && len(args) > 0 {
                var num int64
                if num, err = args[0].Integer(); err == nil {
                        result = g.Get(int(num))
                }
        } else {
                result = UniversalNone
        }
        return
}

func modifierSetArgs(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        if args, err = JoinEval(prog.scope, args...); err != nil {
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

func modifierCD(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        if n := len(args); n == 1 {
                var dir string
                if dir, err = args[0].Strval(); err != nil {
                        return
                }
                if dir == "-" {
                        project := prog.project
                        if prog.caller != nil {
                                project = prog.caller.project // prog.caller.program.project
                        }
                        dir = project.AbsPath()
                        if trace_prepare {
                                fmt.Printf("prepare:CD: %s (%s) (%s)\n", dir, project.name, prog.project.name)
                        }
                } else if trace_prepare {
                        fmt.Printf("prepare:CD: %s (%s)\n", dir, prog.project.name)
                }
                if dir != "" {
                        if err = os.Chdir(dir); err == nil {
                                prog.auto(TheCurrWorkDirDef, &String{dir})
                        }
                }
        } else {
                err = fmt.Errorf("cd: wrong number of args (%v)", n)
        }
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
                                err = breakf(false, "got shell failure")
                                return // target shall be updated
                        } else {
                                depends.Append(d)
                        }
                case *RuleEntry:
                        switch d.Class() {
                        case ExplicitFileEntry:
                                var name string
                                if name, err = d.Strval(); err == nil {
                                        depends.Append(&File{ Name:name })
                                }
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
        if dependVal, err = Reveal(dependVal); err != nil { return }
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
                        err = breakf(false, "no file or directory '%v'", dependFile)
                        return
                }
                if t := dependFile.Info.ModTime(); t.After(tt) {
                        if str, err = target.Strval(); err != nil { return }
                        prog.globe.Timestamps[str] = t
                        outdated = true; return // target is outdated
                } else {
                        var recipes []Value
                        if recipes, err = prog.disclose(prog.recipes); err != nil {
                                return
                        }
                        if same, e := prog.project.CheckCmdHash(target, recipes); e == nil {
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

func modifierCompare_0(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var (
                targetVal Value
                depends = new(List)
                nargs   = len(args)
        )
        if def := prog.scope.Lookup("@").(*Def); nargs == 0 {
                targetVal, _ = def.Call(pos)
        } else if nargs == 1 {
                //switch targetVal = args[0]; t := targetVal.(type) {
                //case *List: targetVal = t.Elems[0]
                //}
                targetVal = args[0]
                def.Assign(targetVal)
        } else if nargs > 1 {
                s := fmt.Sprintf("accepts only one optional argument (%v)", args)
                return nil, breakf(false, s)
        }

        if trace_compare {
                //fmt.Printf("compare: %T %v (%v)\n", targetVal, targetVal, Reveal(Disclose(prog.disctx, targetVal)))
                fmt.Printf("compare:Target: %v\n", targetVal)
        }

        if targetVal, err = Reveal(targetVal); err != nil { return }
        if targetVal == nil || targetVal.Type() == NoneType {
                err = breakf(false, "no target"); return
        }

        // deal with list.
        switch t := targetVal.(type) {
        case *List:
                if n := t.Len(); n == 1 {
                        targetVal = t.Elems[0]
                } else {
                        s := fmt.Sprintf("compare: multiple targets (%v)", targetVal)
                        err = breakf(false, s); return
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
        if targetVal != nil && targetVal.Type() != NoneType {
                var (
                        targetFile *File
                        name string
                )
                if name, err = targetVal.Strval(); err != nil { return }
                switch t := targetVal.(type) {
                case *File: targetFile = t
                case *Barefile: 
                        targetFile = &File{ Name:name }
                case *RuleEntry:
                        switch class := t.class; class {
                        case ExplicitFileEntry, StemmedFileEntry:
                                targetFile = t.file //&File{ Name:name }
                        default:/*if p := t.Project(); p != nil {
                                if p.IsFile(s) {
                                        targetFile = &File{ Name:name }
                                } else {
                                        fmt.Fprintf(os.Stdout, "%v: %v->%v is not file\n", t.Position, p.Name(), s)
                                        err = breakf(false, "unknown %v->%v (%v)", p.Name(), s, class)
                                        return
                                }
                        } else*/ {
                                err = breakf(false, "unknown entry (%v '%v')", class, name)
                                return
                        }}
                }
                if targetFile == nil {
                        err = breakf(false, "unknown target (%T '%v')", targetVal, targetVal)
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
                                outdated, err = compareTargetDepend(pos, prog, targetVal, depend, tt)
                                //fmt.Printf("compare-target-depend: %v (%v)\n", outdated, err)
                                if err != nil || outdated {
                                        return
                                }
                        }
                }
        }
        err = breakf(true, "%s already up to date", targetVal)
        return
}

func modifierCompare(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var target = prog.scope.Lookup("@").(*Def)
        if nargs := len(args); nargs == 1 {
                target.Assign(args[0])
        } else if nargs > 1 {
                s := fmt.Sprintf("accepts only one optional argument (%v)", args)
                return nil, breakf(false, s)
        }

        if c, e := NewComparer(prog.globe, target); e != nil {
                err = e
        } else if err = c.Compare(prog.scope.Lookup("^")); err == nil {
                result = MakeListOrScalar(c.result)
        }
        return
}

func modifierGrepCompare(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        if v, e := modifierGrepDependents(pos, prog, value, args...); e != nil {
                err = e; return
        } else if v != nil {
                def := prog.scope.Lookup("^").(*Def)
                old := def.Value
                def.Assign(v)
                result, err = modifierCompare(pos, prog, value)
                def.Assign(old)
        }
        return
}

func modifierGrepDependents(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var (
                targetVal, _ = prog.scope.Lookup("@").(Caller).Call(pos)
                targetName string
                rxs []*regexp.Regexp                
                f *os.File
                optDiscardMissing = false
        )
        if targetName, err = targetVal.Strval(); err != nil {
                return
        }

        if len(args) == 0 {
                return nil, errors.New("No arguments provided.")
        } else if args, err = JoinEval(prog.scope, args...); err != nil {
                return
        }

        for _, arg := range args {
                switch a := arg.(type) {
                case *Flag:
                        var name string
                        if name, err = a.Name.Strval(); err != nil {
                                return
                        }
                        switch name {
                        case "discard-missing": optDiscardMissing = true
                        }
                case *Pair:
                        //fmt.Printf("todo: grep-dependents: %v\n", a)
                default:
                        var (
                                str string
                                x *regexp.Regexp
                        )
                        if str, err = a.Strval(); err != nil {
                                return
                        }
                        // https://github.com/google/re2/wiki/Syntax
                        if x, err = regexp.Compile(str); err != nil {
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
                defer func() {
                        err = f.Close()
                }()
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
func modifierCheck(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var exeres, _ = value.(*ExecResult)
        if exeres == nil {
                s := fmt.Sprintf("bad value (%T)", value)
                err = breakf(false, s)
                return
        }
        ForArgs: for _, arg := range args {
                switch t := arg.(type) {
                case *Pair:
                        var (
                                key, str string
                                num int64
                        )
                        if key, err = t.Key.Strval(); err != nil { return }
                        switch key {
                        case "status":
                                if num, err = t.Value.Integer(); err != nil { return }
                                if exeres.Status != int(num) {
                                        s := fmt.Sprintf("bad status (%v) (expects %v)", exeres.Status, t.Value)
                                        err = breakf(false, s)
                                        break ForArgs
                                }
                        case "stdout":
                                if v := exeres.Stdout.Buf; v != nil {
                                        if str, err = t.Value.Strval(); err != nil { 
                                                return
                                        } else if v.String() != str {
                                                s := fmt.Sprintf("bad stdout (%v) (expects %v)", v, t.Value)
                                                err = breakf(false, s)
                                                break ForArgs
                                        }
                                } else {
                                        s := fmt.Sprintf("bad stdout (expects %v)", t.Value)
                                        err = breakf(false, s)
                                        break ForArgs
                                }
                        case "stderr":
                                if v := exeres.Stderr.Buf; v != nil {
                                        if str, err = t.Value.Strval(); err != nil {
                                                return
                                        } else if v.String() != str {
                                                s := fmt.Sprintf("bad stderr (%v) (expects %v)", v, t.Value)
                                                err = breakf(false, s)
                                                break ForArgs
                                        }
                                } else {
                                        s := fmt.Sprintf("bad stderr (expects %v)", t.Value)
                                        err = breakf(false, s)
                                        break ForArgs
                                }
                        default:
                                err = fmt.Errorf("unknown check '%v'", t.Key)
                                break ForArgs
                        }
                default:
                        err = fmt.Errorf("unknown check '%v' (%T)", arg, arg)
                        break ForArgs
                }
        }
        return
}

func modifierCheckDir(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*Def)
                targetVal Value
                filename string
        )
        if targetVal, err = targetDef.Call(pos); err != nil {
                return
        } else if filename, err = targetVal.Strval(); err != nil {
                return
        }
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsDir() {
                var segments []Value
                for _, seg := range filepath.SplitList(filename) {
                        segments = append(segments, &String{seg})
                }
                result = &Path{Elements{segments}, nil}
        } else {
                err = breakf(false, "file %s not exists", targetVal)
        }
        return
}

func modifierCheckFile(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*Def)
                targetVal Value
                filename string
        )
        if targetVal, err = targetDef.Call(pos); err != nil {
                return
        } else if filename, err = targetVal.Strval(); err != nil {
                return
        }
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsRegular() {
                result = &File{ Name: filename }
        } else {
                err = breakf(false, "file %s not exists", targetVal)
        }
        return
}

func modifierWriteFile(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*Def)
                targetVal Value
                filename, str string
                f *os.File
        )
        if targetVal, err = targetDef.Call(pos); err != nil {
                return
        } else if filename, err = targetVal.Strval(); err != nil {
                return
        }
        if f, err = os.Create(filename); err == nil {
                defer f.Close()
                if str, err = value.Strval(); err != nil {
                        return
                } else if _, err = f.WriteString(str); err == nil {
                        result = &File{ Name: filename }
                } else {
                        os.Remove(filename)
                }
        } else {
                err = breakf(false, "file %s not generated", targetDef.Value)
        }
        return
}

func modifierUpdateFile(pos token.Position, prog *Program, value Value, args... Value) (result Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*Def)
                nargs = len(args)
                perm = os.FileMode(0640) // sys default 0666
                targetVal Value
                filename, content, s string
                num int64
                f *os.File
        )
        if targetVal, err = targetDef.Call(pos); err != nil {
                return
        }
        if nargs == 0 {
                if filename, err = targetVal.Strval(); err != nil { return }
        } else {
                if filename, err = args[0].Strval(); err != nil { return }
                if nargs > 1 {
                        if num, err = args[1].Integer(); err != nil { return }
                        perm = os.FileMode(num & 0777)
                }
        }

        // Check existed file content checksum
        if content, err = value.Strval(); err != nil { return }
        if f, err = os.Open(filename); err != nil { return }
        if f != nil {
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

        if s, err = args[0].Strval(); err != nil { return }
        var slient = len(args) > 0 && s == "slient"
        if !slient {
                //s, _ := os.Getwd()
                //fmt.Printf("update file `%v' (%s) ..", filename, s)
                fmt.Printf("update file '%v' ..", filename)
        }

        // Create or update the file with new content
        
        f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
        if err == nil && f != nil {
                defer f.Close()
                if _, err = f.WriteString(content); err == nil {
                        result = &File{ Name: filename }
                        if !slient {
                                fmt.Printf(". (ok)\n")
                        }
                } else {
                        os.Remove(filename)
                        if !slient {
                                fmt.Printf(". (%s)\n", err)
                        }
                }
        } else {
                if !slient {
                        fmt.Printf(". (%s)\n", err)
                }
                err = breakf(false, "file %s not updated", targetDef.Value)
        }
        return
}
