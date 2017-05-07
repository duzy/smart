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
        scope      *types.Scope
        projects    []*types.Project
        outdated   map[string]time.Time
        workdir    string
}

func (ctx *Context) Getwd() string {
        return ctx.workdir
}

func (ctx *Context) Globe() *types.Globe {
        return ctx.globe
}

func (ctx *Context) Scope() *types.Scope {
        return ctx.scope
}

func (ctx *Context) SetScope(scope *types.Scope) (prev *types.Scope) {
        prev = ctx.scope
        ctx.scope = scope
        return
}

func (ctx *Context) CurrentProject() *types.Project {
        if n := len(ctx.projects); n > 0 {
                return ctx.projects[n-1]
        }
        return nil
}

func (ctx *Context) EnterProject(m *types.Project, imported bool) *types.Scope {
        if imported {
                if cm := ctx.CurrentProject(); cm != nil {
                        cm.AddImport(m)
                }
        }
        ctx.projects = append(ctx.projects, m)
        return ctx.SetScope(m.Scope())
}

func (ctx *Context) ExitProject(prev *types.Scope) {
        ctx.projects = ctx.projects[0:len(ctx.projects)-1]
        ctx.SetScope(prev)
}

func (ctx *Context) lookupAt(pos token.Pos, ident []string, findProject bool) (*types.Scope, types.Object) {
        var (
                sym types.Object
                xname = len(ident)
                name  = ident[xname-1]
                scope = ctx.Scope()
        )
        if xname == 2 {
                for _, s := range ident[0:xname-1] {
                        _, sym = scope.LookupAt(pos, s)
                        //fmt.Printf("scope: %p: %v (%v)\n", scope, scope.Names(), sym)
                        if t, ok := sym.(*types.ProjectName); ok && t != nil {
                                if m := t.Imported(); m != nil {
                                        scope = m.Scope()
                                } else {
                                        return nil, nil
                                }
                        } else {
                                return nil, nil
                        }
                }
        } else if xname > 2 {
                // FIXME: supports multi-scopes lookup (e.g. foo.bar.name)?
                return nil, nil
        }

lookupName:
        _, sym = scope.LookupAt(pos, name)
        if !findProject && sym != nil {
                // Skip any ProjectName objects, and lookup upword.
                if _, ok := sym.(*types.ProjectName); ok {
                        scope = scope.Parent()
                        goto lookupName
                }
        }
        
//lookupDone:
        return scope, sym
}

func (ctx *Context) callAt(pos token.Pos, ident []string, args... types.Value) types.Value {
        if _, sym := ctx.lookupAt(pos, ident, false); sym != nil {
                if v, _ := sym.Call(args...); v != nil {
                        return v
                }
        }
        return values.None
}

type delegate struct {
        x *Context
        s types.Object
        a []types.Value
        p token.Pos
}

func (p *delegate) Type() types.Type  { return nil }
func (p *delegate) Pos() token.Pos    { return p.p }
func (p *delegate) Lit() string       { return p.call().Lit() }
func (p *delegate) String() string    { return p.call().String() }
func (p *delegate) Integer() int64    { return p.call().Integer() }
func (p *delegate) Float() float64    { return p.call().Float() }
func (p *delegate) call() (v types.Value) {
        if types.IsDummy(p.s) {
                scope := p.s.Parent()
                if _, s := scope.LookupAt(token.NoPos, p.s.Name()); s == nil {
                        v = values.None
                } else {
                        p.s = s
                        v, _ = s.Call(p.a...)
                }
        } else {
                v, _ = p.s.Call(p.a...)
        }
        return v 
}

func (ctx *Context) Fold(pos token.Pos, sym types.Object, args... types.Value) types.Value {
        return &delegate{
                x: ctx,
                s: sym,
                a: args,
                p: pos,
        }
}

func (ctx *Context) defineBuiltin(name string, f builtin) {
        ctx.globe.Scope().Insert(types.NewBuiltin(name, func(args... types.Value) (types.Value, error) {
                return f(ctx, args...)
        }))
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
        
        defer ctx.ExitProject(ctx.EnterProject(mm, false))
        
        if len(targets) == 0 {
                //ctx.outdated = make(map[string][]string)
                ctx.outdated = make(map[string]time.Time)
                if entry := mm.GetDefaultEntry(); entry != nil {
                        if value, err = entry.Call(); err == nil {
                                updated += 1
                        }
                }
        } else {
                var m = mm
                for _, target := range targets {
                        if names := strings.Split(target, "."); len(names)>1 {
                                for _, s := range names[0:len(names)-1] {
                                        m = m.FindImport(s)
                                        if m == nil {
                                                fmt.Printf("project '%s' not imported", s)
                                                return
                                        }
                                }
                                target = names[len(names)-1]
                        }
                        if entry := m.Lookup(target); entry != nil {
                                //ctx.outdated = make(map[string][]string)
                                ctx.outdated = make(map[string]time.Time)
                                if value, err = entry.Call(); err == nil {
                                        updated += 1
                                } else {
                                        break
                                }
                        } else {
                                fmt.Printf("target %s not found\n", target)
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
                        scope:    globe.Scope(),
                        workdir:  workdir,
                }
        )
        if false {
                fmt.Printf("context: %p\n", context)
        }
        context.defineBuiltins()
        return context
}
