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
        if _, obj := ctx.lookupAt(pos, ident, false); obj != nil {
                switch t := obj.(type) {
                case types.Caller:
                        if v, _ := t.Call(args...); v != nil {
                                return v
                        }
                }
        }
        return values.None
}

type delegate struct {
        x *Context
        o types.Object
        a []types.Value
        p *token.Position
}

func (p *delegate) Type() types.Type     { return p.o.Type() }
func (p *delegate) Pos() *token.Position { return p.p }
func (p *delegate) Lit() string          { return p.call().Lit() }
func (p *delegate) String() string       { return p.call().String() }
func (p *delegate) Integer() int64       { return p.call().Integer() }
func (p *delegate) Float() float64       { return p.call().Float() }
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
                                                m = t.Imported()
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
                                entry = t.Imported().GetDefaultEntry()
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
