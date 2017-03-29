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
        "path/filepath"
        "hash/crc64"
        //"strings"
        //"errors"
        "fmt"
        "os"
        "io"
)

type breaker struct {
        message string
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
                
                `shell-status`: modifierShellStatus,
                `shell-stdout`: modifierShellStdout,
                `shell-stderr`: modifierShellStderr,

                `select`:       modifierSelect,

                `when-outdated`: modifierWhenOutdated,
                
                `check-dir`:    modifierCheckDir,
                `check-file`:   modifierCheckFile,
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

func getGroupElem(output types.Value, n int, v types.Value) types.Value {
        if g, ok := output.(*values.GroupValue); ok {
                if elem := g.Get(n); elem != nil {
                        v = elem
                }
        }
        return v
}

func modifierShellStatus(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        result = getGroupElem(value, 0, values.None)
        return
}

func modifierShellStdout(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        result = getGroupElem(value, 1, values.None)
        return
}

func modifierShellStderr(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        result = getGroupElem(value, 2, values.None)
        return
}

func modifierSelect(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        if g, ok := value.(*values.GroupValue); ok && len(args) > 0 {
                result = g.Get(int(args[0].Integer()))
        } else {
                result = values.None
        }
        return
}

func modifierWhenOutdated(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                scope        = prog.context.Scope()
                targetDef, _ = scope.Lookup("@").(*types.Def)
                dependDef, _ = scope.Lookup("...").(*types.Def)
                depends, _   = dependDef.Value().(*values.ListValue)
                target       = targetDef.Value().String()
                missing      = values.List()
                files        = values.List()
                shellFalses  int
        )
        if depends != nil || depends.Len() > 0 {
                for _, depend := range depends.Slice(0) {
                retryDepend:
                        //fmt.Printf("depend: %T %v (from %s)\n", depend, depend, target)
                        switch d := depend.(type) {
                        case *values.ListValue:
                                if depend = d.Take(0); depend != nil {
                                        goto retryDepend
                                }
                        case *values.GroupValue:
                                switch k, _ := d.Get(0).(*values.BarewordValue); { 
                                case k == targetRegularKind, k == targetDirectoryKind:
                                        files.Append(d.Get(1))
                                case k == targetShellKind:
                                        if n := d.Get(1).Integer(); n != 0 {
                                                shellFalses += int(n)
                                        }
                                }
                        case *values.StringValue:
                                if ext := filepath.Ext(d.String()); ext != "" {
                                        if _, ok := prog.context.CheckExt(ext); ok {
                                                files.Append(d)
                                        } else {
                                                FailAt(depend.Pos(), "unsupported file %v", d)
                                        }
                                } else {
                                        fmt.Printf("depend: %v\n", d)
                                }
                        default:
                                //fmt.Printf("modifierWhenOutdated: todo: %T %v (from %s)\n", depend, depend, target)
                                FailAt(depend.Pos(), "unsupported depend %v (%T)", depend, depend)
                        }
                }

                if x := missing.Len(); x > 0 {
                        err = &breaker{ fmt.Sprintf("missing %v, required by %s", 
                                missing, target) }
                        goto DoneWhen
                }
                
                if files.Len() > 0 {
                        prog.auto("^", files)
                        prog.auto("<", files.Get(0))
                        goto CheckTargetOutdated
                }
        }

        if shellFalses > 0 {
                goto DoneWhen // target shall be updated
        }

CheckTargetOutdated:
        if fi, _ := os.Stat(target); fi != nil {
                for _, depend := range files.Slice(0) {
                        fi2, e := os.Stat(depend.String())
                        if fi2 == nil {
                                err = e
                                goto DoneWhen
                        }
                        
                        if fi2.ModTime().After(fi.ModTime()) {
                                goto DoneWhen // target is outdated
                        }
                }
                err = &breaker{ fmt.Sprintf("%s already up to date", target) }
        }

DoneWhen:
        return
}

func modifierCheckDir(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var target, _ = prog.scope.Lookup("@").Call()
        if fi, _ := os.Stat(target.String()); fi != nil && fi.Mode().IsDir() {
                result = values.Group(targetDirectoryKind, target)
        } else {
                err = &breaker{ fmt.Sprintf("directory %s not exists", target) }
        }
        return
}

func modifierCheckFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                target = targetDef.Value()
                filename = target.String()
        )
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsRegular() {
                result = values.Group(targetRegularKind, target)
        } else {
                err = &breaker{ fmt.Sprintf("file %s not exists", target) }
        }
        return
}

func modifierWriteFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                target = targetDef.Value()
                filename = target.String()
        )
        if f, err := os.Create(filename); err == nil {
                defer f.Close()
                var content string
                switch v := value.(type) {
                case *values.GroupValue:
                        switch t, _ := v.Get(0).(*values.BarewordValue); t {
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
                err = &breaker{ fmt.Sprintf("file %s not generated", target) }
        }
        return
}

func modifierUpdateFile(prog *Program, value types.Value, args... types.Value) (result types.Value, err error) {
        var (
                targetDef, _ = prog.scope.Lookup("@").(*types.Def)
                target = targetDef.Value()
                filename = target.String()
                content string
        )

        switch v := value.(type) {
        case *values.GroupValue:
                switch t, _ := v.Get(0).(*values.BarewordValue); t {
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
                err = &breaker{ fmt.Sprintf("file %s not updated", target) }
        }
        return
}
