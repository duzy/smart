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
        "fmt"
)

type Context struct {
        globe    *types.Globe
        scope    *types.Scope
        modules  []*types.Module
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

func (ctx *Context) CurrentModule() *types.Module {
        if n := len(ctx.modules); n > 0 {
                return ctx.modules[n-1]
        }
        return nil
}

func (ctx *Context) DeclareModule(pos token.Pos, kw token.Token, path, name string) *types.Module {
        m := ctx.globe.NewModule(kw, path, name)
        n := types.NewModuleName(pos, ctx.CurrentModule(), m.Name(), m)
        ctx.Scope().Insert(n) //; assert(n.Scope() == ctx.Scope())
        return m
}

func (ctx *Context) EnterModule(m *types.Module) *types.Scope {
        ctx.modules = append(ctx.modules, m)
        return ctx.SetScope(m.Scope())
}

func (ctx *Context) ExitModule(prev *types.Scope) {
        ctx.modules = ctx.modules[0:len(ctx.modules)-1]
        ctx.SetScope(prev)
}

type delegate struct {
        x *Context
        i []string
        a []types.Value
        p token.Pos
}

func (p *delegate) Type() types.Type  { return nil }
func (p *delegate) Lit() string       { return p.call().Lit() }
func (p *delegate) String() string    { return p.call().String() }
func (p *delegate) Integer() int64    { return p.call().Integer() }
func (p *delegate) Float() float64    { return p.call().Float() }
func (p *delegate) call() types.Value {
        var (
                sym types.Symbol
                xname = len(p.i)
                name  = p.i[xname-1]
                scope = p.x.Scope()
        )
        if xname == 2 {
                for _, s := range p.i[0:xname-1] {
                        _, sym = scope.LookupAt(s, p.p)
                        //fmt.Printf("scope: %p: %v (%v)\n", scope, scope.Names(), sym)
                        if t, ok := sym.(*types.ModuleName); ok && t != nil {
                                if m := t.Imported(); m != nil {
                                        scope = m.Scope()
                                        //fmt.Printf("scope: %p: %v: %v\n", scope, m.Name(), scope.Names())
                                } else {
                                        return values.None
                                }
                        } else {
                                //err = ErrorModuleNotFound
                                return values.None
                        }
                }
        } else if xname > 2 {
                //err = ErrorIllName
                return values.None
        }
        if _, sym = scope.LookupAt(name, p.p); sym != nil {
                if v, _ := sym.Call(p.a...); v != nil {
                        return v
                }
        }
        return values.None
}

func (ctx *Context) Fold(pos token.Pos, ident []string, args... types.Value) types.Value {
        return &delegate{
                x: ctx,
                i: ident,
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

func NewContext(name string) *Context {
        var (
                globe = types.NewGlobe(name)
                context = &Context{
                        globe:    globe,
                        scope:    globe.Scope(),
                }
        )
        if false {
                fmt.Printf("context: %p\n", context)
        }
        context.defineBuiltins()
        return context
}
