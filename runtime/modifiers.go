//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        //"github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        //"path/filepath"
        "hash/crc64"
        "strings"
        //"errors"
        "fmt"
        "os"
        "io"
)

type breaker struct {
        message string
        okay bool
}

func (p *breaker) Error() string {
        return p.message
}

type modifier func(prog *Program, value types.Value, args... types.Value) (types.Value, error)

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

                `docksh`: &dialectDocksh{
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
                `stdout`:       modifierShellStdout,
                `stderr`:       modifierShellStderr,
                `stdin`:        modifierShellStdin,

                `status-equals`: modifierStatusEquals,
                `stdout-equals`: modifierStdoutEquals,
                `stderr-equals`: modifierStderrEquals,

                `select`:       modifierSelect,

                `args`:         modifierSetArgs, // interpreter args
                `env`:          modifierSetEnv,  // interpreter environments

                `cd`:           modifierCD,

                `compare`:      modifierCompare,
                
                `check-dir`:    modifierCheckDir,
                `check-file`:   modifierCheckFile,
                `dir-p`:        modifierCheckDir,
                `file-p`:       modifierCheckFile,
                
                `write-file`:   modifierWriteFile,
                `update-file`:  modifierUpdateFile,
        }

        // Phony targets (always outdate the target)
        targetPhonyKind     = values.Bareword("phony")    // (phony example)

        // Filesystem targets
        targetRegularKind   = values.Bareword("regular")  // (regular example.cpp)
        targetDirectoryKind = values.Bareword("directoy") // (directory sources)

        // Interpreter targets
        targetPlainKind     = values.Bareword("plain")    // (plain 'plain text')
        targetJsonKind      = values.Bareword("json")     // (json (array a b c 1 2 3 null))
        targetXmlKind       = values.Bareword("xml")      // (xml ((book (title book one)) (book (title book two)) (book (title book three))))
        targetShellKind     = values.Bareword("shell")    // (shell 0 'output' 'error')

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
                if elem := g.Get(0); elem != nil && elem.String() == "shell" {
                        if elem = g.Get(n); elem != nil {
                                if s := elem.String(); strings.HasSuffix(s, "\n") {
                                        fmt.Printf("%s", s)
                                } else if s != "" {
                                        fmt.Printf("%s\n", s)
                                }
                        }
                }
        }
}

func modifierShellStatus(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-status", "on")
        if len(args) > 0 && args[0].String() == "off" {
                def.Set(args[0])
        }
        promptShellResult(value, 1)
        return
}

func modifierShellStdout(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-stdout", "on")
        if len(args) > 0 && args[0].String() == "off" {
                def.Set(args[0])
        }
        promptShellResult(value, 2)
        return
}

func modifierShellStderr(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-stderr", "on")
        if len(args) > 0 && args[0].String() == "off" {
                def.Set(args[0])
        }
        promptShellResult(value, 3)
        return
}

func modifierShellStdin(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        def := prog.auto("shell-stdin", "on")
        if len(args) > 0 && args[0].String() == "off" {
                def.Set(args[0])
        }
        //promptShellResult(value, ?)
        return
}

func modifierStatusEquals(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        v, a := getGroupElem(value, 1, nil), ""
        if v != nil && len(args) == 1 {
                if a = args[0].String(); a == v.String() {
                        result = v; return
                }
        }
        err = &breaker{ fmt.Sprintf("bad status (%v, expects %v)", v, a), false }
        return
}

func modifierStdoutEquals(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        v, a := getGroupElem(value, 2, nil), ""
        if v != nil && len(args) == 1 {
                if a = args[0].String(); a == v.String() {
                        result = v; return
                }
        }
        err = &breaker{ fmt.Sprintf("bad stdout (%v, expects %v)", v, a), false }
        return
}

func modifierStderrEquals(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        v, a := getGroupElem(value, 3, nil), ""
        if v != nil && len(args) == 1 {
                if a = args[0].String(); a == v.String() {
                        result = v; return
                }
        }
        err = &breaker{ fmt.Sprintf("bad stderr (%v, expects %v)", v, a), false }
        return
}

func modifierSelect(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        if g, ok := value.(*types.Group); ok && len(args) > 0 {
                result = g.Get(int(args[0].Integer()))
        } else {
                result = values.None
        }
        return
}

func modifierSetArgs(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        // TODO: preserve args for interpreter
        return
}

func modifierSetEnv(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        // TODO: preserve env for interpreter
        return
}

func modifierCD(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        // does nothing
        return
}

func modifierCompare(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                nargs = len(args)
                targetName = "@" // default target name
                dependName = "^" // default depend name
        )
        if nargs > 0 { targetName = args[0].String() }
        if nargs > 1 { } // TODO: warnings

        var (
                targetDef, _  = prog.scope.Lookup(targetName).(*types.Def)
                targetVal, _  = targetDef.Call()
                targetFile, _ = targetVal.(*types.File)

                dependDef, _  = prog.scope.Lookup(dependName).(*types.Def)
                dependVal, _  = dependDef.Call()
                dependList, _ = dependVal.(*types.List)

                missing       = values.List()
                files         = values.List()
                nonfiles      = values.List()
                shellFalses  int
        )
        if targetFile == nil {
                err = &breaker{ fmt.Sprintf("compare expects File (%T %v)", targetVal, targetVal), false }
                goto DoneCompare
        }
        if dependList != nil && dependList.Len() > 0 {
                for _, depend := range dependList.Slice(0) {
                        //fmt.Printf("modifierCompare: %T %v (from %T %v)\n", depend, depend, target, target)
                        DependSwitch: switch d := depend.(type) {
                        case *types.List:
                                if depend = d.Take(0); depend != nil {
                                        goto DependSwitch
                                }
                        case *types.Group:
                                switch d.Get(0).(*types.Bareword) {
                                case targetRegularKind, targetDirectoryKind:
                                        switch d1 := d.Get(1).(type) {
                                        case *types.File: files.Append(d1)
                                        case *types.RuleEntry:
                                                // Retry switching
                                                depend = d1; goto DependSwitch
                                        default: Fail("compare: %v: unsupported group depend %v (%T)", targetVal, d, d1)
                                        }
                                case targetShellKind:
                                        if d1 := d.Get(1); d1.Integer() != 0 {
                                                shellFalses += 1
                                        }
                                }
                        case *types.Barefile:
                                Fail("compare: %v: unsupported depend %v (%T)", targetVal, d, d)
                        case *types.RuleEntry:
                                switch d.Class() {
                                case types.FileRuleEntry, types.PatternFileRuleEntry:
                                        Fail("compare: %v: unhandled file depend %v (%v)", targetVal, d, d.Class())
                                case types.GeneralRuleEntry, types.PatternRuleEntry:
                                        nonfiles.Append(d)
                                default:
                                        Fail("compare: %v: unsupported entry depend %v (%v)", targetVal, d.Class())
                                }
                        case *types.String:
                                if prog.project.IsFile(d.String()) {
                                        Fail("compare: %v: unhandled file depend %v (%T)", targetVal, depend, depend)
                                } else {
                                        nonfiles.Append(d)
                                }
                        case *types.File:
                                files.Append(d)
                        default:
                                Fail("compare: %v: unsupported depend %v (%T)", targetVal, depend, depend)
                        }
                }

                if x := missing.Len(); x > 0 {
                        err = &breaker{ fmt.Sprintf("missing %v, required by %s", 
                                missing, targetVal), false }
                        goto DoneCompare
                }

                if files.Len() > 0 {
                        prog.auto("<", files.Get(0))
                        prog.auto("^", files)
                }
        }

        if shellFalses > 0 {
                err = &breaker{ fmt.Sprintf("got %v failures", shellFalses), false }
                goto DoneCompare // target shall be updated
        }

        if fi := targetFile.Info; fi != nil {
                for _, depend := range files.Slice(0) {
                        //fmt.Printf("modifierCompare: %v -> %v (%v)\n", targetVal, depend, prog.context.outdated)
                        //fmt.Printf("modifierCompare: %v: %v (%T)\n", targetVal, depend, depend)
                        if dependFile, okay := depend.(*types.File); okay {
                                if t, ok := prog.context.outdated[dependFile.String()]; ok && t.After(fi.ModTime()) {
                                        goto DoneCompare // target is outdated
                                }
                                if dependFile.Info == nil {
                                        err = &breaker{ fmt.Sprintf("no file or directory '%v'", dependFile),
                                                false }
                                        goto DoneCompare
                                }
                                if t := dependFile.Info.ModTime(); t.After(fi.ModTime()) {
                                        prog.context.outdated[targetVal.String()] = t
                                        goto DoneCompare // target is outdated
                                }
                        } else {
                                fmt.Printf("modifierCompare: todo: %v -> %v (%T)\n", targetVal, depend, depend)
                        }
                }
                err = &breaker{ fmt.Sprintf("%s already up to date", targetVal), true }
        } else {
                for _, depend := range files.Slice(0) {
                        //fmt.Printf("modifierCompare: (nil) %v -> %v (%v)\n", targetVal, depend, prog.context.outdated)
                        //fmt.Printf("modifierCompare: (nil) %v -> %v (%T)\n", targetVal, depend, depend)
                        if dependFile, okay := depend.(*types.File); okay {
                                if dependFile.Info == nil {
                                        dependFile.Info, _ = os.Stat(dependFile.String())
                                }
                                if dependFile.Info == nil {
                                        err = &breaker{ fmt.Sprintf("no file or directory '%v'", dependFile),
                                                false }
                                        goto DoneCompare
                                }
                        } else {
                                fmt.Printf("modifierCompare: todo: %v -> %v (%T)\n", targetVal, depend, depend)
                        }
                }
                goto DoneCompare // target shall be updated
        }

        DoneCompare: return
}

func modifierCheckDir(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var targetVal, _ = prog.scope.Lookup("@").(types.Caller).Call()
        if fi, _ := os.Stat(targetVal.String()); fi != nil && fi.Mode().IsDir() {
                result = values.Group(targetDirectoryKind, targetVal)
        } else {
                err = &breaker{ fmt.Sprintf("directory %s not exists", targetVal),
                        false }
        }
        return
}

func modifierCheckFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                targetVal, _ = targetDef.Call()
                filename = targetVal.String()
        )
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsRegular() {
                result = values.Group(targetRegularKind, targetVal)
        } else {
                err = &breaker{ fmt.Sprintf("file %s not exists", targetVal), 
                        false }
        }
        return
}

func modifierWriteFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                targetVal, _ = targetDef.Call()
                filename = targetVal.String()
        )
        if f, err := os.Create(filename); err == nil {
                defer f.Close()
                var content string
                switch v := value.(type) {
                case *types.Group:
                        switch t, _ := v.Get(0).(*types.Bareword); t {
                        case targetPlainKind:
                                content = v.Get(1).String()
                        case targetJsonKind:
                                // TODO: convert to json value
                                content = v.Get(1).String()
                        case targetXmlKind:
                                // TODO: convert to xml value
                                content = v.Get(1).String()
                        default:
                                // TODO: convert value
                                content = v.Get(1).Lit()
                        }
                default:
                        content = v.String()
                }
                if _, err = f.WriteString(content); err == nil {
                        result = values.Group(targetRegularKind, targetVal)
                } else {
                        os.Remove(filename)
                }
        } else {
                err = &breaker{ fmt.Sprintf("file %s not generated", targetVal), 
                        false }
        }
        return
}

func modifierUpdateFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                targetVal, _ = targetDef.Call()
                filename = targetVal.String()
                content string
        )

        switch v := value.(type) {
        case *types.Group:
                switch t, _ := v.Get(0).(*types.Bareword); t {
                case targetPlainKind:
                        content = v.Get(1).String()
                case targetJsonKind:
                        // TODO: convert to json value
                        content = v.Get(1).String()
                case targetXmlKind:
                        // TODO: convert to xml value
                        content = v.Get(1).String()
                default:
                        // TODO: convert value
                        content = v.Get(1).Lit()
                }
        default:
                content = v.String()
        }

        // Check existed file content checksum
        f, err := os.Open(filename)
        if err == nil && f != nil {
                defer f.Close()
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
                        result = values.Group(targetRegularKind, targetVal)
                        return
                }
        }

        var slient = len(args) > 0 && args[0].String() == "slient"
        if !slient {
                fmt.Printf("update %v ..", filename)
        }

        // Create or update the file with new content
        f, err = os.Create(filename)
        if err == nil && f != nil {
                defer f.Close()
                if _, err = f.WriteString(content); err == nil {
                        result = values.Group(targetRegularKind, targetVal)
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
                err = &breaker{ fmt.Sprintf("file %s not updated", targetVal), 
                        false }
        }
        return
}
