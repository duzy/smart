//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package runtime

import (
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "strings"
        "time"
        "fmt"
        "os"
)

type Context struct {
        globe      *types.Globe
        outdated   map[string]time.Time
        workdir    string
}
func (ctx *Context) Getwd() string { return ctx.workdir }
func (ctx *Context) Globe() *types.Globe { return ctx.globe }

type delegate struct {
        x *Context
        o types.Object
        a []types.Value
        p *token.Position
}

func (p *delegate) Type() types.Type      { return p.o.Type() }
func (p *delegate) Pos() *token.Position  { return p.p }
func (p *delegate) Lit() string           { return p.call().Lit() }
func (p *delegate) String() string        { return p.call().String() }
func (p *delegate) Integer() int64        { return p.call().Integer() }
func (p *delegate) Float() float64        { return p.call().Float() }
func (p *delegate) call() (v types.Value) {
        if types.IsDummy(p.o) {
                scope := p.o.Parent()
                if _, s := scope.LookupAt(token.NoPos, p.o.Name()); s != nil {
                        p.o = s
                }
        }
        if c, ok := p.o.(types.Caller); ok {
                v, _ = c.Call(p.a...)
        }
        if v == nil {
                v = values.None
        }
        return v 
}

func (ctx *Context) Fold(pos token.Pos, obj types.Object, args... types.Value) types.Value {
        return &delegate{
                x: ctx,
                o: obj,
                a: args,
                //p: pos,
        }
}

func (ctx *Context) defineBuiltin(name string, f builtin) {
        scope := ctx.globe.Scope()
        _, alt := scope.InsertNewBuiltin(name, func(args... types.Value) (types.Value, error) {
                return f(ctx, args...)
        })
        if alt != nil {
                panic(fmt.Sprintf("builtin '%s' already defined", name))
        }
}

func (ctx *Context) defineBuiltins() {
        for name, f := range builtins {
                ctx.defineBuiltin(name, f)
        }
}

func (ctx *Context) Run(targets... string) (err error) {
        var (
                value types.Value
                updated int
                mm = ctx.Globe().Main()
        )

        //fmt.Printf("run: %v\n", targets)
        
        if len(targets) == 0 {
                ctx.outdated = make(map[string]time.Time)
                if entry := mm.GetDefaultEntry(); entry != nil {
                        if value, err = entry.Call(); err == nil {
                                updated += 1
                        }
                }
        } else {
                for _, target := range targets {
                        var m = mm
                        if names := strings.Split(target, "."); len(names)>1 {
                                for _, s := range names[0:len(names)-1] {
                                        var obj = m.Scope().Lookup(s)
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

                        var entry *types.RuleEntry
                        switch t := m.Scope().Lookup(target).(type) {
                        case *types.ProjectName:
                                entry = t.Project().GetDefaultEntry()
                        case *types.RuleEntry:
                                entry = t
                        case nil:
                                fmt.Printf("'%s' is not defined in %v", target, m.Scope())
                                return
                        default:
                                fmt.Printf("object '%s' is not callable (%T)", target, t)
                                return
                        }

                        if entry != nil {
                                ctx.outdated = make(map[string]time.Time)
                                if value, err = entry.Call(); err == nil {
                                        updated += 1
                                } else {
                                        //fmt.Printf("%v\n", err)
                                        break
                                }
                        }
                }
        }
        if value == nil {}
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
        context.defineBuiltins()
        return context
}
