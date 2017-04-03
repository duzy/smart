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
        "os"
)

type Context struct {
        globe    *types.Globe
        scope    *types.Scope
        modules  []*types.Module
        exts     map[string][]string
        workdir  string
}

func (ctx *Context) SetExts(m map[string][]string) {
        ctx.exts = m
}

func (ctx *Context) CheckExt(s string) (a []string, v bool) {
        if len(s) > 0 {
                if s[0] == '.' {
                        s = s[1:]
                }
                a, v = ctx.exts[s]
        }
        return
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

func (ctx *Context) lookupAt(pos token.Pos, ident []string, findModule bool) (*types.Scope, types.Symbol) {
        var (
                sym types.Symbol
                xname = len(ident)
                name  = ident[xname-1]
                scope = ctx.Scope()
        )
        if xname == 2 {
                for _, s := range ident[0:xname-1] {
                        _, sym = scope.LookupAt(pos, s)
                        //fmt.Printf("scope: %p: %v (%v)\n", scope, scope.Names(), sym)
                        if t, ok := sym.(*types.ModuleName); ok && t != nil {
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
        if !findModule && sym != nil {
                // Skip any ModuleName symbols, and lookup upword.
                if _, ok := sym.(*types.ModuleName); ok {
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
        s types.Symbol
        a []types.Value
        p token.Pos
}

func (p *delegate) call() types.Value { v, _ := p.s.Call(p.a...); return v }
func (p *delegate) Lit() string       { return p.call().Lit() }
func (p *delegate) String() string    { return p.call().String() }
func (p *delegate) Integer() int64    { return p.call().Integer() }
func (p *delegate) Float() float64    { return p.call().Float() }
func (p *delegate) Pos() token.Pos    { return p.p }
func (p *delegate) Type() types.Type  { return nil }

func (ctx *Context) Fold(pos token.Pos, sym types.Symbol, args... types.Value) types.Value {
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
                m = ctx.Globe().Main()
        )
        
        defer ctx.ExitModule(ctx.EnterModule(m))
        
        if len(targets) == 0 {
                if entry := m.GetDefaultEntry(); entry != nil {
                        if value, err = entry.Call(); err == nil {
                                updated += 1
                        }
                }
        } else {
                for _, target := range targets {
                        if entry := m.Lookup(target); entry != nil {
                                if value, err = entry.Call(); err == nil {
                                        updated += 1
                                } else {
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
