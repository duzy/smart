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
        //"strings"
        //"fmt"
        "os"
)

type builtin func(ctx *Context, args... types.Value) types.Value

var (
        builtins = map[string]builtin {
                `lit`:          builtinLit,
                
                `shell-status`: builtinShellStatus,
                `shell-stdout`: builtinShellStdout,
                `shell-stderr`: builtinShellStderr,

                `select`:       builtinSelect,
                
                `check-dir`:    builtinCheckDir,
                `check-file`:   builtinCheckFile,
                `write-file`:   builtinWriteFile,
                `update-file`:  builtinUpdateFile,
        }

        // Phony targets (always outdate the target)
        targetPhonyKind     = values.Bareword("phony")    // (phony example)

        // Filesystem targets
        targetRegularKind   = values.Bareword("regular")  // (regular example.cpp)
        targetDirectoryKind = values.Bareword("directoy")  // (directory sources)
        targetMissingKind   = values.Bareword("missing")  // (missing example.o)

        // Interpreter targets
        targetPlainKind     = values.Bareword("plain")    // (plain 'plain text')
        targetJsonKind      = values.Bareword("json")     // (json (array a b c 1 2 3 null))
        targetXmlKind       = values.Bareword("xml")      // (xml ((book (title book one)) (book (title book two)) (book (title book three))))
        targetShellKind     = values.Bareword("shell")    // (shell 0 'output' 'error')
)

func builtinLit(ctx *Context, args... types.Value) types.Value {
        var s string
        for _, a := range args {
                s += a.Lit()
        }
        return values.String(s)
}

func getGroupElem(output types.Value, n int, v types.Value) types.Value {
        if g, ok := output.(*values.GroupValue); ok {
                if elem := g.Get(n); elem != nil {
                        v = elem
                }
        }
        return v
}

func builtinShellStatus(ctx *Context, args... types.Value) types.Value {
        return getGroupElem(ctx.Call("-"), 0, values.None)
}

func builtinShellStdout(ctx *Context, args... types.Value) types.Value {
        return getGroupElem(ctx.Call("-"), 1, values.None)
}

func builtinShellStderr(ctx *Context, args... types.Value) types.Value {
        return getGroupElem(ctx.Call("-"), 2, values.None)
}

func builtinSelect(ctx *Context, args... types.Value) types.Value {
        var output = ctx.Call("-")
        if g, ok := output.(*values.GroupValue); ok {
                if len(args) > 0 {
                        return g.Get(int(args[0].Integer()))
                }
        }
        return values.None
}

func builtinCheckDir(ctx *Context, args... types.Value) types.Value {
        var target = ctx.Call("@")
        if fi, _ := os.Stat(target.String()); fi != nil && fi.Mode().IsDir() {
                return target
        }
        return values.None
}

func builtinCheckFile(ctx *Context, args... types.Value) types.Value {
        var (
                scope  = ctx.Scope()
                outputDef, _ = scope.Lookup("-").(*types.Def)
                targetDef, _ = scope.Lookup("@").(*types.Def)
                target = targetDef.Value()
                filename = target.String()
        )
        if fi, _ := os.Stat(filename); fi != nil && fi.Mode().IsRegular() {
                outputDef.Reset(values.Group(targetRegularKind, target))
        } else {
                outputDef.Reset(values.Group(targetMissingKind, target))
        }
        return outputDef.Value()
}

func builtinWriteFile(ctx *Context, args... types.Value) types.Value {
        var (
                scope  = ctx.Scope()
                outputDef, _ = scope.Lookup("-").(*types.Def)
                targetDef, _ = scope.Lookup("@").(*types.Def)
                target = targetDef.Value()
                filename = target.String()
        )
        if f, err := os.Create(filename); err == nil {
                defer f.Close()
                var content string
                switch v := outputDef.Value().(type) {
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
                        outputDef.Reset(values.Group(targetRegularKind, target))
                } else {
                        os.Remove(filename)
                }
        } else {
                outputDef.Reset(values.Group(targetMissingKind, target))
        }
        return nil
}

func builtinUpdateFile(ctx *Context, args... types.Value) types.Value {
        var (
                scope  = ctx.Scope()
                outputDef, _ = scope.Lookup("-").(*types.Def)
                targetDef, _ = scope.Lookup("@").(*types.Def)
                target = targetDef.Value()
                filename = target.String()
        )
        if f, err := os.Create(filename); err == nil {
                defer f.Close()
                var content string
                switch v := outputDef.Value().(type) {
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
                        outputDef.Reset(values.Group(targetRegularKind, target))
                } else {
                        os.Remove(filename)
                }
        } else {
                outputDef.Reset(values.Group(targetMissingKind, target))
        }
        return nil
}
