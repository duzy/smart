//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
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
        okay bool // it's good to continue
}

func (p *breaker) Error() string {
        return p.message
}

type modifier func(prog *Program, context *types.Scope, value types.Value, args... types.Value) (types.Value, error)

var (
        interpreters = map[string]interpreter{
                `plain`: &dialectPlain{
                },

                `shell`: &dialectShell{
                        interpreter: defaultShellInterpreter, // "sh"
                        xopt: "-c",
                },

                `python`: &dialectShell{
                        interpreter: "python",
                        xopt: "-c",
                },
                
                `perl`: &dialectShell{
                        interpreter: "perl",
                        xopt: "-e",
                },

                `dock`: &dialectDock{
                },

                `xml`: &dialectXml{
                        whitespace: false,
                },

                `json`: &dialectJson{
                },

                ``: &dialectDefault{
                },
        }

        modifiers = map[string]modifier{
                `status`:       modifierShellStatus,
                //`stdout`:       modifierShellStdout,
                //`stderr`:       modifierShellStderr,
                //`stdin`:        modifierShellStdin,

                `status-equals`: modifierStatusEquals,
                `stdout-equals`: modifierStdoutEquals,
                `stderr-equals`: modifierStderrEquals,

                `select`:       modifierSelect,

                `args`:         modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments

                `cd`:           modifierCD,

                `compare`:      modifierCompare,
                `grep-compare`: modifierGrepCompare,
                `grep-dependents`: modifierGrepDependents,
                
                `check-dir`:    modifierCheckDir,
                `check-file`:   modifierCheckFile,
                `dir-p`:        modifierCheckDir,
                `file-p`:       modifierCheckFile,
                
                `write-file`:   modifierWriteFile,
                `update-file`:  modifierUpdateFile,
        }

        crc64Table = crc64.MakeTable(crc64.ECMA /*crc64.ISO*/)
)

func (ctx *Context) IsDialect(s string) (ok bool) {
        _, ok = interpreters[s]
        return
}

func (ctx *Context) IsModifier(s string) (ok bool) {
        _, ok = modifiers[s]
        return
}

func getGroupElem(value types.Value, n int, v types.Value) types.Value {
        if g, ok := value.(*types.Group); ok {
                if elem := g.Get(n); elem != nil {
                        v = elem
                }
        }
        return v
}

func promptShellResult(value types.Value, n int) {
        if g, ok := value.(*types.Group); ok {
                if elem := g.Get(0); elem != nil && elem.Strval() == "shell" {
                        if elem = g.Get(n); elem != nil {
                                if s := elem.Strval(); strings.HasSuffix(s, "\n") {
                                        fmt.Printf("%s", s)
                                } else if s != "" {
                                        fmt.Printf("%s\n", s)
                                }
                        }
                }
        }
}

func modifierShellStatus(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-status", "on")
        if len(args) > 0 && args[0].Strval() == "off" {
                def.Assign(args[0])
        }
        promptShellResult(value, 1)
        return
}

func modifierShellStdout(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-stdout", "on")
        if len(args) > 0 && args[0].Strval() == "off" {
                def.Assign(args[0])
        }
        promptShellResult(value, 2)
        return
}

func modifierShellStderr(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-stderr", "on")
        if len(args) > 0 && args[0].Strval() == "off" {
                def.Assign(args[0])
        }
        promptShellResult(value, 3)
        return
}

func modifierShellStdin(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-stdin", "on")
        if len(args) > 0 && args[0].Strval() == "off" {
                def.Assign(args[0])
        }
        //promptShellResult(value, ?)
        return
}

func modifierStatusEquals(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        v, a := getGroupElem(value, 1, nil), ""
        if v != nil && len(args) == 1 {
                if a = args[0].Strval(); a == v.Strval() {
                        result = v; return
                }
        }
        err = &breaker{ fmt.Sprintf("bad status (%v, expects %v)", v, a), false }
        return
}

func modifierStdoutEquals(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        v, a := getGroupElem(value, 2, nil), ""
        if v != nil && len(args) == 1 {
                if a = args[0].Strval(); a == v.Strval() {
                        result = v; return
                }
        }
        err = &breaker{ fmt.Sprintf("bad stdout (%v, expects %v)", v, a), false }
        return
}

func modifierStderrEquals(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        v, a := getGroupElem(value, 3, nil), ""
        if v != nil && len(args) == 1 {
                if a = args[0].Strval(); a == v.Strval() {
                        result = v; return
                }
        }
        err = &breaker{ fmt.Sprintf("bad stderr (%v, expects %v)", v, a), false }
        return
}

func modifierSelect(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        if g, ok := value.(*types.Group); ok && len(args) > 0 {
                result = g.Get(int(args[0].Integer()))
        } else {
                result = values.None
        }
        return
}

func modifierSetArgs(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        if args, err = types.JoinEval(context, args...); err != nil {
                return
        }
        var envars = values.List()
        _ = prog.auto("shell-envars", envars)
        for _, a := range args {
                if _, ok := a.(*types.Pair); ok {
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

func modifierCD(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        // does nothing
        return
}

func parseDependList(prog *Program, context *types.Scope, dependList *types.List) (depends *types.List, err error) {
        depends = values.List()
        for _, depend := range dependList.Elems {
                //fmt.Printf("compare: depend: %T %v\n", depend, depend)
                switch d := depend.(type) {
                case *types.List:
                        if dl, e := parseDependList(prog, context, d); e != nil {
                                err = e; return
                        } else {
                                depends.Elems = append(depends.Elems, dl.Elems...)
                        }
                case *types.ExecResult:
                        if d.Status != 0 {
                                err = &breaker{ "got shell failure", false }
                                return // target shall be updated
                        } else {
                                depends.Append(d)
                        }
                case *types.RuleEntry:
                        switch d.Class() {
                        case types.FileRuleEntry:
                                depends.Append(values.File(d, d.Strval()))
                        case types.GeneralRuleEntry, types.PatternRuleEntry:
                                depends.Append(d)
                        default:
                                Fail("compare: unsupported entry depend `%v' (%v)", d.Strval(), d.Class())
                        }
                case *types.String:
                        if prog.project.IsFile(d.Strval()) {
                                Fail("compare: discarded file depend %v (%T)", depend, depend)
                        } else {
                                depends.Append(d)
                        }
                case *types.File:
                        depends.Append(d)
                default:
                        Fail("compare: unsupported depend `%T' (%v)", depend, depend)
                }
        }
        return
}

func getCompareDepends(prog *Program, context *types.Scope, targetVal types.Value) (depends *types.List, err error) {
        def := prog.scope.Lookup("^").(*types.Def)
        dependVal, _ := def.Call()
        dependVal = types.Reveal(dependVal)
        if dependList, _ := dependVal.(*types.List); dependList != nil && dependList.Len() > 0 {
                if depends, err = parseDependList(prog, context, dependList); err != nil {
                        return
                }
        }
        return
}

func compareTargetDepend(prog *Program, context *types.Scope, target, depend types.Value, tt time.Time) (outdated bool, err error) {
        //fmt.Printf("compare: %v -> %v (%v)\n", target, depend, prog.context.outdated)
        //fmt.Printf("compare: %v: %v (%T)\n", target, depend, depend)
        if dependFile, okay := depend.(*types.File); okay && dependFile != nil {
                if t, ok := prog.context.outdated[dependFile.Strval()]; ok && t.After(tt) {
                        outdated = true; return // target is outdated
                } else if dependFile.Info == nil {
                        dependFile.Info, _ = os.Stat(dependFile.Strval())
                }
                if dependFile.Info == nil {
                        err = &breaker{ fmt.Sprintf("no file or directory '%v'", dependFile),
                                false }
                        return
                }
                if t := dependFile.Info.ModTime(); t.After(tt) {
                        prog.context.outdated[target.Strval()] = t
                        outdated = true; return // target is outdated
                } else {
                        var recipes []types.Value
                        if recipes, err = prog.discloseRecipes(context); err != nil {
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

func modifierCompare(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetVal types.Value
                depends = values.List()
                nargs   = len(args)
        )
        if def := prog.scope.Lookup("@").(*types.Def); nargs == 0 {
                targetVal, _ = def.Call()
        } else if nargs == 1 {
                //switch targetVal = args[0]; t := targetVal.(type) {
                //case *types.List: targetVal = t.Elems[0]
                //}
                targetVal = args[0]
                def.Assign(targetVal)
        } else if nargs > 1 {
                s := fmt.Sprintf("compare: accepts only one optional argument (%v)", args)
                return nil, &breaker{ s, false }
        }

        //fmt.Printf("compare: %v (%T)\n", targetVal, targetVal)

        if targetVal = types.Reveal(targetVal); targetVal == nil || targetVal.Type() == types.NoneType {
                return nil, &breaker{ "compare: no target", false }
        }

        // deal with list.
        switch t := targetVal.(type) {
        case *types.List:
                if n := t.Len(); n == 1 {
                        targetVal = t.Elems[0]
                } else {
                        return nil, &breaker{ "wrong number of targets", false }
                }
        }

        if depends, err = getCompareDepends(prog, context, targetVal); err != nil {
                return
        } else if depends == nil || depends.Len() == 0 {
                // Nothing to compare!
                return
        } else {
                prog.auto("<", depends.Get(0))
                prog.auto("^", depends)
        }

        //fmt.Printf("compare: %v (%T), depends: %v\n", targetVal, targetVal, depends)
        
        // Comparing target with depends.

        var (
                outdated = false
                tt time.Time
        )
        if targetVal != nil && targetVal.Type() != types.NoneType {
                var targetFile *types.File
                switch t := targetVal.(type) {
                case *types.File: targetFile = t
                case *types.Barefile: targetFile = values.File(t, t.Strval())
                case *types.RuleEntry:
                        if t.Class() == types.FileRuleEntry {
                                targetFile = values.File(t, t.Strval())
                        }
                }
                if targetFile == nil {
                        err = &breaker{ fmt.Sprintf("compare: expects `*types.File' target instead of `%T' (%v)", targetVal, targetVal), false }
                        return
                } else if nargs == 1 {
                        // Replace the value of "$@"
                        def := prog.scope.Lookup("@").(*types.Def)
                        def.Assign(targetFile)
                }

                // In case passing a unstated target file.
                if targetFile.Info == nil && targetFile != nil {
                        targetFile.Info, _ = os.Stat(targetFile.Strval())
                }
                if fi := targetFile.Info; fi != nil {
                        tt = fi.ModTime()
                }
        }
        for _, depend := range depends.Elems {
                switch depend.(type) {
                case *types.File:
                        outdated, err = compareTargetDepend(prog, context, targetVal, depend, tt)
                        //fmt.Printf("compare-target-depend: %v (%v)\n", outdated, err)
                        if err != nil || outdated {
                                return
                        }
                }
        }
        err = &breaker{ fmt.Sprintf("%s already up to date", targetVal), true }
        return
}

func modifierGrepCompare(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        if v, e := modifierGrepDependents(prog, context, value, args...); e != nil {
                err = e; return
        } else if v != nil {
                def := prog.scope.Lookup("^").(*types.Def)
                old := def.Value
                def.Assign(v)
                result, err = modifierCompare(prog, context, value)
                def.Assign(old)
        }
        return
}

func modifierGrepDependents(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetVal, _ = prog.scope.Lookup("@").(types.Caller).Call()
                targetName = targetVal.Strval()
                rxs []*regexp.Regexp                
                f *os.File
                optDiscardMissing = false
        )

        if len(args) == 0 {
                return nil, errors.New("No arguments provided.")
        } else if args, err = types.JoinEval(context, args...); err != nil {
                return
        }

        for _, arg := range args {
                switch a := arg.(type) {
                case *types.Flag:
                        switch a.Strval() {
                        case "-discard-missing": optDiscardMissing = true
                        }
                case *types.Pair:
                        //fmt.Printf("todo: grep-dependents: %v\n", a)
                default:
                        // https://github.com/google/re2/wiki/Syntax
                        if x, e := regexp.Compile(a.Strval()); e != nil {
                                err = e; return
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
        dependList := values.List()
        scanner := bufio.NewScanner(f)
        scanner.Split(bufio.ScanLines)
        for scanner.Scan() {
                s := scanner.Text()
                for _, x := range rxs { //if x.MatchString(s) {
                        if sm := x.FindStringSubmatch(s); len(sm) == 2 && sm[1] != "" {
                                v := values.File(values.String(sm[1]), sm[1])
                                v = project.SearchFile(context, v)
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

func modifierCheckDir(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                targetVal types.Value
                filename string
        )
        if targetVal, err = targetDef.Call(); err != nil {
                return
        } else {
                filename = targetVal.Strval()
        }
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsDir() {
                var segments []types.Value
                for _, seg := range filepath.SplitList(filename) {
                        segments = append(segments, values.String(seg))
                }
                result = values.Path(segments...)
        } else {
                err = &breaker{ fmt.Sprintf("file %s not exists", targetVal), false }
        }
        return
}

func modifierCheckFile(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                targetVal types.Value
                filename string
        )
        if targetVal, err = targetDef.Call(); err != nil {
                return
        } else {
                filename = targetVal.Strval()
        }
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsRegular() {
                result = values.File(targetVal, filename)
        } else {
                err = &breaker{ fmt.Sprintf("file %s not exists", targetVal), false }
        }
        return
}

func modifierWriteFile(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                targetVal types.Value
                filename string
        )
        if targetVal, err = targetDef.Call(); err != nil {
                return
        } else {
                filename = targetVal.Strval()
        }
        if f, err := os.Create(filename); err == nil {
                defer f.Close()
                if _, err = f.WriteString(value.Strval()); err == nil {
                        result = values.File(targetVal, filename)
                } else {
                        os.Remove(filename)
                }
        } else {
                err = &breaker{ fmt.Sprintf("file %s not generated", targetDef.Value), 
                        false }
        }
        return
}

func modifierUpdateFile(prog *Program, context *types.Scope, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                nargs = len(args)
                perm = os.FileMode(0640) // sys default 0666
                targetVal types.Value
                filename string
        )
        if targetVal, err = targetDef.Call(); err != nil {
                return
        }
        if nargs == 0 {
                filename = targetVal.Strval()
        } else {
                filename = args[0].Strval()
                if nargs > 1 {
                        perm = os.FileMode(args[1].Integer() & 0777)
                }
        }

        // Check existed file content checksum
        var content = value.Strval()
        f, err := os.Open(filename)
        if err == nil && f != nil {
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
                        result = values.File(targetVal, filename)
                        return
                }
        }

        var slient = len(args) > 0 && args[0].Strval() == "slient"
        if !slient {
                fmt.Printf("update file `%v' ..", filename)
        }

        // Create or update the file with new content
        
        f, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
        if err == nil && f != nil {
                defer f.Close()
                if _, err = f.WriteString(content); err == nil {
                        result = values.File(targetVal, filename)
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
                err = &breaker{ fmt.Sprintf("file %s not updated", targetDef.Value), 
                        false }
        }
        return
}
