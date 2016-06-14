//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "path/filepath"
        "strings"
        "regexp"
        "bytes"
        "fmt"
        "io/ioutil"
        "os/exec"
        "os"
)

type builtin func(ctx *Context, loc location, args Items) Items

var (
        rxName = regexp.MustCompile(`@(?P<N>.*?)@`)
        
        builtins = map[string]builtin {
                "info":         builtinInfo,
                
                "assert":       builtinAssert,
                "assert-not":   builtinAssertNot,
                "ignore":       builtinIgnore,
                
                "dir":          builtinDir,

                "trim-space":   builtinTrimSpace,
                "trim-prefix":  builtinTrimPrefix,
                "trim-suffix":  builtinTrimSuffix,
                
                "upper":        builtinUpper,
                "lower":        builtinLower,
                "title":        builtinTitle,

                "when":         builtinWhen,
                "unless":       builtinUnless,
                "let":          builtinLet,
                "set":          builtinSet,

                "expr":         builtinExpr,

                "=":            builtinSetEqual,
                //"!=":         builtinSetNot,
                "?=":           builtinSetQuestioned,
                "+=":           builtinSetAppend,

                "shell":        builtinShell,
                "pkg-config":   builtinPkgConfig,
                // pkg-include-dirs
                // pkg-lib-dirs
                // pkg-libs
                // pkg-cflags
                // pkg-ldflags
                "configure":    builtinConfigure,
        }

        builtinInfoFunc = func(ctx *Context, args Items) {
                var as []string
                for _, a := range args {
                        as = append(as, a.Expand(ctx))
                }
                fmt.Printf("%v\n", strings.Join(as, ","))
        }
)

func SetBuiltinInfoFunc(f func(ctx *Context, args Items)) func(ctx *Context, args Items) {
        previous := builtinInfoFunc
        builtinInfoFunc = f
        return previous
}

func builtinInfo(ctx *Context, loc location, args Items) (is Items) {
        if builtinInfoFunc != nil {
                builtinInfoFunc(ctx, args)
        }
        return
}

func builtinAssert(ctx *Context, loc location, args Items) (is Items) {
        if strings.TrimSpace(args.Expand(ctx)) != "true" {
                errorf("assersion failed")
        }
        return
}

func builtinAssertNot(ctx *Context, loc location, args Items) (is Items) {
        if strings.TrimSpace(args.Expand(ctx)) != "false" {
                errorf("assersion failed")
        }
        return
}

func builtinIgnore(ctx *Context, loc location, args Items) (is Items) {
        return
}

func builtinDir(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(filepath.Dir(a.Expand(ctx))))
        }
        return
}

func builtinTrimSpace(ctx *Context, loc location, args Items) (is Items) {
        if 1 < len(args) {
                for _, a := range args[1:] {
                        is.AppendString(strings.TrimSpace(a.Expand(ctx)))
                }
        }
        return
}

func builtinTrimPrefix(ctx *Context, loc location, args Items) (is Items) {
        if 1 < len(args) {
                prefix := strings.TrimSpace(args[0].Expand(ctx))
                for _, a := range args[1:] {
                        is.AppendString(strings.TrimPrefix(a.Expand(ctx), prefix))
                }
        }
        return
}

func builtinTrimSuffix(ctx *Context, loc location, args Items) (is Items) {
        if 1 < len(args) {
                suffix := strings.TrimSpace(args[0].Expand(ctx))
                for _, a := range args[1:] {
                        is.AppendString(strings.TrimSuffix(a.Expand(ctx), suffix))
                }
        }
        return
}

func builtinUpper(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToUpper(a.Expand(ctx))))
        }
        return
}

func builtinLower(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToLower(a.Expand(ctx))))
        }
        return
}

func builtinTitle(ctx *Context, loc location, args Items) (is Items) {
        for _, a := range args {
                is = append(is, stringitem(strings.ToTitle(a.Expand(ctx))))
        }
        return
}

func builtinSet(ctx *Context, loc location, args Items) (is Items) {
        return builtinSetEqual(ctx, loc, args)
}

func builtinSetEqual(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                name := strings.TrimSpace(args[0].Expand(ctx))
                hasPrefix, prefix, parts := ctx.expandNameString(name)
                ctx.setWithDetails(hasPrefix, prefix, parts, args[1:]...)
        }
        return
}

func builtinSetNot(ctx *Context, loc location, args Items) (is Items) {
        panic("todo: $(!= name, ...)")
        return
}

func builtinSetQuestioned(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                name := strings.TrimSpace(args[0].Expand(ctx))
                hasPrefix, prefix, parts := ctx.expandNameString(name)
                if d := ctx.getDefineWithDetails(hasPrefix, prefix, parts); d == nil || d.value.IsEmpty(ctx) {
                        ctx.setWithDetails(hasPrefix, prefix, parts, args[1:]...)
                }
        }
        return
}

func builtinSetAppend(ctx *Context, loc location, args Items) (is Items) {
        if num := len(args); 1 < num {
                name := strings.TrimSpace(args[0].Expand(ctx))
                hasPrefix, prefix, parts := ctx.expandNameString(name)
                if d := ctx.getDefineWithDetails(hasPrefix, prefix, parts); d == nil {
                        ctx.setWithDetails(hasPrefix, prefix, parts, args[1:]...)
                } else {
                        d.value = append(d.value, args...)
                }
        }
        return
}

func builtinWhen(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

func builtinUnless(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

func builtinLet(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

// builtinExpr evaluates a math expression.
func builtinExpr(ctx *Context, loc location, args Items) (is Items) {
        errorf("todo: %v", args)
        return
}

// $(shell pwd)
func builtinShell(ctx *Context, loc location, args Items) (result Items) {
        var stdout, stderr bytes.Buffer
        
        commands := args.Expand(ctx)
        sh := exec.Command("sh", "-c", commands)
        sh.Stdout, sh.Stderr = &stdout, &stderr
        if e := sh.Run(); e != nil {
                fmt.Printf("smart: shell: %v (%v)\n", e, commands)
                return
        }

        result.AppendString(strings.TrimSuffix(stdout.String(), "\n"))
        return
}

// $(pkg-config --cflags --libs, foo, bar, foobar)
func builtinPkgConfig(ctx *Context, loc location, args Items) (result Items) {
        var stdout, stderr bytes.Buffer
        
        a := Split(args.Expand(ctx))
        pc := exec.Command("pkg-config", a...)
        pc.Stdout, pc.Stderr = &stdout, &stderr
        if e := pc.Run(); e != nil {
                fmt.Printf("smart: pkg-config: %v\n", e)
                return
        }

        result.AppendString(strings.TrimSuffix(stdout.String(), "\n"))
        return
}

// $(configure <OUTPUT>, <INTPUT>, ...)
func builtinConfigure(ctx *Context, loc location, args Items) (result Items) {
        if len(args) < 2 {
                result.AppendString("false")
                errorf("configure: insufficient arguments: %v", args.Expand(ctx))
                return
        }

        var (
                outputName = strings.TrimSpace(args[0].Expand(ctx))
                inputName = strings.TrimSpace(args[1].Expand(ctx))
        )

        input, e := os.Open(inputName)
        if e != nil {
                result.AppendString("false")
                errorf("configure: %v", e)
                return
        }

        if e = os.MkdirAll(filepath.Dir(outputName), os.FileMode(0755)); e != nil {
                result.AppendString("false")
                errorf("configure: %v", e)
                return
        }
        
        output, e := os.Create(outputName)
        if e != nil {
                result.AppendString("false")
                errorf("configure: %v", e)
                return
        }

        s, e := ioutil.ReadAll(input)
        if e != nil {
                result.AppendString("false")
                errorf("configure: %v", e)
                return
        }
        
        s = rxName.ReplaceAllFunc(s, func(x []byte) []byte {
                m := rxName.FindSubmatchIndex(x)
                b := rxName.Expand(nil, []byte("$N"), x, m)
                if ctx.m != nil {
                        name := string(b)
                        if d, ok := ctx.m.defines[name]; ok && d != nil {
                                return []byte(d.value.Expand(ctx))
                        }
                }
                return []byte("")
        })

        if _, e = output.Write(s); e != nil {
                result.AppendString("false")
                errorf("configure: %v", e)
                return
        }

        result.AppendString("true")
        return
}
