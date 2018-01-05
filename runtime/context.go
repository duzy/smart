//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/values"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/token"
        "strings"
        "fmt"
        "os"
)

type Context struct {
        globe    *types.Globe
        workdir  string
}
func (ctx *Context) Getwd() string { return ctx.workdir }
func (ctx *Context) Globe() *types.Globe { return ctx.globe }

/*func (ctx *Context) defineBuiltin(name string, f builtin) {
        scope := ctx.globe.Scope()
        _, alt := scope.InsertBuiltin(name, func(scope *types.Scope, args... types.Value) (types.Value, error) {
                return f(ctx, scope, args...)
        })
        if alt != nil {
                panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
}

func (ctx *Context) defineBuiltins() {
        for name, f := range builtins {
                ctx.defineBuiltin(name, f)
        }
}*/

func (context *Context) NewProgram(position token.Position, project *types.Project, params []string, scope *types.Scope, depends []types.Value, recipes... types.Value) *types.Program {
        return types.NewProgram(context.globe, position, project, params, scope, depends, recipes...)
}

func (ctx *Context) Run(targets... string) (err error) {
        var (
                result []types.Value
                updated int
                mm = ctx.Globe().Main()
        )

        if mm == nil {
                types.Fail("no targets to update")
        }

        if len(targets) == 0 {
                if entry := mm.DefaultEntry(); entry != nil {
                        if result, err = entry.Execute(); err == nil {
                                updated += 1
                        }
                }
        } else {
                for _, target := range targets {
                        var m = mm
                        if names := strings.Split(target, "->"); len(names)>1 {
                                for _, s := range names[0:len(names)-1] {
                                        var _, obj = m.Scope().Find(s)
                                        switch t := obj.(type) {
                                        case *types.ProjectName:
                                                m = t.Project()
                                        case nil:
                                                fmt.Printf("'%s' is not defined in %v", s, m.Scope())
                                                return
                                        default:
                                                fmt.Printf("object '%s' is not project (%T)", s, t)
                                                return
                                        }
                                        if m == nil {
                                                fmt.Printf("project '%s' not imported %v", s)
                                                return
                                        }
                                }
                                target = names[len(names)-1]
                        }

                        var (
                                entry *types.RuleEntry
                                args = strings.Split(target, ":")
                        )
                        target, args = args[0], args[1:]
                        switch t := m.Scope().Resolve(target).(type) {
                        case *types.ProjectName: entry = t.Project().DefaultEntry()
                        case *types.RuleEntry:   entry = t
                        case nil:
                                fmt.Printf("'%s' is not defined in %v", target, m.Scope())
                                return
                        default:
                                fmt.Printf("object '%s' is not callable (%T)", target, t)
                                return
                        }

                        if entry != nil {
                                var v []types.Value
                                for _, a := range args {
                                        v = append(v, values.String(a))
                                }
                                
                                // The the base project scope as execution context. For
                                // example of 'base.test', the entry 'test' can resolve
                                // '&(FOO)', '&(BAR)', etc.
                                if result, err = entry.Execute(v...); err == nil {
                                        updated += 1
                                } else {
                                        //fmt.Printf("%v\n", err)
                                        break
                                }
                        }
                }
        }
        if err == nil && result != nil && len(result) > 0 {
                for _, v := range result {
                        fmt.Printf(v.Strval())
                }
        }
        return
}

func NewContext(name string) *Context {
        var (
                workdir, _ = os.Getwd()
                globe = types.NewGlobe(name)
                context = &Context{
                        globe:    globe,
                        workdir:  workdir,
                }
        )
        if false {
                fmt.Printf("context: %p\n", context)
        }
        //context.defineBuiltins()
        return context
}
