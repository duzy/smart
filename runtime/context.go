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
        registry *Registry
        modules  []*types.Module
}

func (ctx *Context) Registry() *Registry {
        return ctx.registry
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

func (ctx *Context) CallSym(sym types.Symbol, args... types.Value) types.Value {
        if sym == nil {
                return values.None
        }

        if sym.Callable() {
                return sym.Call(/*ctx,*/ args...)
        }

        if na := len(args); na > 0 {
                // TODO: create calling scope (lexical $1, $2, etc)
        }
        
        return sym.Value()
}

func (ctx *Context) Call(name string, args... types.Value) types.Value {
        _, sym := ctx.scope.LookupAt(name, token.NoPos)
        return ctx.CallSym(sym, args...)
}

/*
func (ctx *Context) Reset(name string, value types.Value) (def *types.Def) {
        if def, _ = ctx.scope.Lookup(name).(*types.Def); def != nil {
                def.Reset(value)
        }
        return
}

func (ctx *Context) Set(name string, value types.Value) (def *types.Def) {
        if def, _ = ctx.scope.Lookup(name).(*types.Def); def == nil {
                def = types.
        } else {
                def.Reset(value)
        }
        return
} */

func (ctx *Context) GetDefaultEntry() (entry *RuleEntry) {
        return ctx.registry.GetDefaultEntry()
}

func (ctx *Context) GetEntry(name string) (entry *RuleEntry) {
        return ctx.registry.Lookup(name)
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
        _, sym = scope.LookupAt(name, p.p)
        if sym != nil {
                if sym.Callable() {
                        if v := sym.Call(/*p.x,*/ p.a...); v != nil {
                                return v
                        }
                } else {
                        return sym.Value()
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
        ctx.globe.Scope()/*.Parent()*/.Insert(types.NewBuiltin(name, func(/*x types.Context,*/ args... types.Value) types.Value {
                return f(/*x.(*Context)*/ctx, args...)
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
                        registry: NewRegistry(),
                }
        )
        if false {
                fmt.Printf("context: %p\n", context)
        }
        context.defineBuiltins()
        return context
}
