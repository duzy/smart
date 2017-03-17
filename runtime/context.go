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
)

type Context struct {
        globe    *types.Globe
        scope    *types.Scope
        registry *Registry
        modules  []*types.Module
}

func (ctx *Context) Globe() *types.Globe {
        return ctx.globe
}

func (ctx *Context) Scope() *types.Scope {
        return ctx.scope
}

func (ctx *Context) Registry() *Registry {
        return ctx.registry
}

func (ctx *Context) LookupAt(name string, pos token.Pos) (sym types.Symbol) {
        if sym = ctx.scope.Lookup(name); sym == nil {
                _, sym = ctx.scope.LookupParent(name, pos)
        }
        return
}

func (ctx *Context) CallSym(sym types.Symbol, args... interface{}) types.Value {
        if sym == nil {
                return values.None
        }

        var av = values.MakeAll(args)
        if sym.Callable() {
                return sym.Call(av...)
        }

        if na := len(av); na > 0 {
                // TODO: create calling scope (lexical $1, $2, etc)
        }
        
        return sym.Value()
}

func (ctx *Context) Call(name string, args... interface{}) types.Value {
        return ctx.CallSym(ctx.LookupAt(name, token.NoPos), args...)
}

func (ctx *Context) CurrentModule() *types.Module {
        if n := len(ctx.modules); n > 0 {
                return ctx.modules[n-1]
        }
        return nil
}

func (ctx *Context) NewModule(pos token.Pos, keyword token.Token, path, name string) *types.Module {
        m := types.NewModule(keyword, path, name)
        // modName := types.NewModuleName(pos, m, name, nil)
        // ctx.scope.Insert(modName)
        ctx.scope = m.Scope()
        ctx.modules = append(ctx.modules, m)
        return m
}

func (ctx *Context) ExitCurrentScope() {
        if scope := ctx.scope.Parent(); !types.IsUniverse(scope) {
                ctx.scope = scope
        }
}

func (ctx *Context) defineAuto(name string, value interface{}) (auto *types.Auto) {
        auto = types.NewAuto(ctx.CurrentModule(), name, values.Make(value))
        ctx.scope.Insert(auto)
        return
}

func (ctx *Context) GetDefaultEntry() (entry *RuleEntry) {
        return ctx.registry.GetDefaultEntry()
}

func (ctx *Context) GetEntry(name string) (entry *RuleEntry) {
        return ctx.registry.Lookup(name)
}

func (ctx *Context) RunEntry(entry *RuleEntry) (err error) {
        ctx.scope = types.NewScope(ctx.scope, token.NoPos, token.NoPos, entry.Name())
        defer func() { ctx.scope = ctx.scope.Parent() } ()

        ctx.defineAuto("@", entry.Name())
        //fmt.Printf("%v\n", i.lookupAt("@", token.NoPos))
        
        return entry.Execute()
}

func (ctx *Context) RunEntryByName(name string) (err error) {
        if entry := ctx.GetEntry(name); entry == nil {
                err = ErrorNoEntry
        } else {
                err = ctx.RunEntry(entry)
        }
        return
}

func MakeContext(name string) Context {
        globe := types.NewGlobe(name)
        return Context{
                globe:    globe,
                scope:    globe.Scope(),
                registry: NewRegistry(),
        }
}
