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

                /* TODO: (dock sh)
                `dock`: &dialectDock{
                }, */

                `xml`: &dialectXml{
                        whitespace: false,
                },

                `json`: &dialectJson{
                },

                ``: &dialectDefault{
                },
        }

        modifiers = map[string]modifier{
                //`pre-check`:    modifierPreCheck,

                /*
                `shell-status`: modifierShellStatus,
                `shell-stdout`: modifierShellStdout,
                `shell-stderr`: modifierShellStderr,
                `shell-stdin`:  modifierShellStdin, */

                `status`:       modifierShellStatus,
                `stdout`:       modifierShellStdout,
                `stderr`:       modifierShellStderr,
                `stdin`:        modifierShellStdin,

                `status-equals`: modifierStatusEquals,
                `stdout-equals`: modifierStdoutEquals,
                `stderr-equals`: modifierStderrEquals,

                `select`:       modifierSelect,

                `args`:         modifierSetArgs,

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
        if g, ok := value.(*types.GroupValue); ok {
                if elem := g.Get(n); elem != nil {
                        v = elem
                }
        }
        return v
}

func promptShellResult(value types.Value, n int) {
        if g, ok := value.(*types.GroupValue); ok {
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
        if g, ok := value.(*types.GroupValue); ok && len(args) > 0 {
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

func modifierCompare(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                scope         = prog.scope //context.Scope()
                targetDef, _  = scope.Lookup("@").(*types.Def)
                dependDef, _  = scope.Lookup("...").(*types.Def)
                dependsVal, _ = dependDef.Call()
                depends, _    = dependsVal.(*types.ListValue)
                target, _     = targetDef.Call()
                targetFile, _ = target.(*types.FileValue)
                missing       = values.List()
                files         = values.List()
                nonfiles      = values.List()
                shellFalses  int
        )
        if depends != nil || depends.Len() > 0 {
                for _, depend := range depends.Slice(0) {
                        //fmt.Printf("modifierCompare: %T %v (from %T %v)\n", depend, depend, target, target)
                retryDepend:
                        switch d := depend.(type) {
                        case *types.ListValue:
                                if depend = d.Take(0); depend != nil {
                                        goto retryDepend
                                }
                        case *types.GroupValue:
                                switch d.Get(0).(*types.BarewordValue) {
                                case targetRegularKind, targetDirectoryKind:
                                        if f, _ := d.Get(1).(*types.FileValue); f != nil {
                                                files.Append(f)
                                        } else {
                                                Fail("%v: unknown depend %v (%T, %T)", target, depend, depend, d.Get(1))
                                        }
                                case targetShellKind:
                                        if n := d.Get(1).Integer(); n != 0 {
                                                shellFalses += 1
                                        }
                                }
                        case *types.BarefileValue:
                                //files.Append(d)
                                Fail("%v: unknown depend %v (%T)", target, d, d)
                        case *types.RuleEntry:
                                switch d.Class() {
                                case types.FileRuleEntry, types.PatternFileRuleEntry:
                                        //files.Append(d)
                                        Fail("%v: unknown depend %v (%T)", target, depend, depend)
                                case types.GeneralRuleEntry, types.PatternRuleEntry:
                                        nonfiles.Append(d)
                                default:
                                        Fail("%v: unknown depend rule %v (%T)", target, depend, depend)
                                }
                        case *types.StringValue:
                                if prog.project.IsFile(d.String()) {
                                        //files.Append(d)
                                        Fail("%v: unknown depend %v (%T)", target, depend, depend)
                                } else {
                                        nonfiles.Append(d)
                                }
                        case *types.FileValue:
                                files.Append(d)
                        default:
                                Fail("%v: unknown depend %v (%T)", target, depend, depend)
                        }
                }

                if x := missing.Len(); x > 0 {
                        err = &breaker{ fmt.Sprintf("missing %v, required by %s", 
                                missing, target), false }
                        goto DoneWhen
                }

                if files.Len() > 0 {
                        prog.auto("<", files.Get(0))
                        prog.auto("^", files)
                }
        }

        if shellFalses > 0 {
                err = &breaker{ fmt.Sprintf("got %v failures", shellFalses), false }
                goto DoneWhen // target shall be updated
        }

        if targetFile == nil {
                err = &breaker{ fmt.Sprintf("expects file target %v", target), false }
                goto DoneWhen
        }

        if fi := targetFile.Info; fi != nil {
                for _, depend := range files.Slice(0) {
                        //fmt.Printf("modifierCompare: %v -> %v (%v)\n", target, depend, prog.context.outdated)
                        //fmt.Printf("modifierCompare: %v: %v (%T)\n", target, depend, depend)
                        if dependFile, okay := depend.(*types.FileValue); okay {
                                if t, ok := prog.context.outdated[dependFile.String()]; ok && t.After(fi.ModTime()) {
                                        goto DoneWhen // target is outdated
                                }
                                if dependFile.Info == nil {
                                        err = &breaker{ fmt.Sprintf("no file or directory '%v'", dependFile),
                                                false }
                                        goto DoneWhen
                                }
                                if t := dependFile.Info.ModTime(); t.After(fi.ModTime()) {
                                        prog.context.outdated[target.String()] = t
                                        goto DoneWhen // target is outdated
                                }
                        } else {
                                fmt.Printf("modifierCompare: todo: %v -> %v (%T)\n", target, depend, depend)
                        }
                }
                err = &breaker{ fmt.Sprintf("%s already up to date", target), true }
        } else {
                for _, depend := range files.Slice(0) {
                        //fmt.Printf("modifierCompare: (nil) %v -> %v (%v)\n", target, depend, prog.context.outdated)
                        //fmt.Printf("modifierCompare: (nil) %v -> %v (%T)\n", target, depend, depend)
                        if dependFile, okay := depend.(*types.FileValue); okay {
                                if dependFile.Info == nil {
                                        err = &breaker{ fmt.Sprintf("no file or directory '%v'", dependFile),
                                                false }
                                        goto DoneWhen
                                }
                        } else {
                                fmt.Printf("modifierCompare: todo: %v -> %v (%T)\n", target, depend, depend)
                        }
                }
                goto DoneWhen // target shall be updated
        }

DoneWhen:
        return
}

func modifierCheckDir(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var target, _ = prog.scope.Lookup("@").(types.Caller).Call()
        if fi, _ := os.Stat(target.String()); fi != nil && fi.Mode().IsDir() {
                result = values.Group(targetDirectoryKind, target)
        } else {
                err = &breaker{ fmt.Sprintf("directory %s not exists", target),
                        false }
        }
        return
}

func modifierCheckFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                target, _ = targetDef.Call()
                filename = target.String()
        )
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsRegular() {
                result = values.Group(targetRegularKind, target)
        } else {
                err = &breaker{ fmt.Sprintf("file %s not exists", target), 
                        false }
        }
        return
}

func modifierWriteFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                target, _ = targetDef.Call()
                filename = target.String()
        )
        if f, err := os.Create(filename); err == nil {
                defer f.Close()
                var content string
                switch v := value.(type) {
                case *types.GroupValue:
                        switch t, _ := v.Get(0).(*types.BarewordValue); t {
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
                        result = values.Group(targetRegularKind, target)
                } else {
                        os.Remove(filename)
                }
        } else {
                err = &breaker{ fmt.Sprintf("file %s not generated", target), 
                        false }
        }
        return
}

func modifierUpdateFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                target, _ = targetDef.Call()
                filename = target.String()
                content string
        )

        switch v := value.(type) {
        case *types.GroupValue:
                switch t, _ := v.Get(0).(*types.BarewordValue); t {
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
                        result = values.Group(targetRegularKind, target)
                        return
                }
        }

        // Create or update the file with new content
        f, err = os.Create(filename)
        if err == nil && f != nil {
                defer f.Close()
                if _, err = f.WriteString(content); err == nil {
                        result = values.Group(targetRegularKind, target)
                } else {
                        os.Remove(filename)
                }
        } else {
                err = &breaker{ fmt.Sprintf("file %s not updated", target), 
                        false }
        }
        return
}
