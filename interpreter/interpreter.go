//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package interpreter

import (
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
        "github.com/duzy/smart/values"
        "github.com/duzy/smart/runtime"
        //"errors"
        //"fmt"
)

type Interpreter struct {
        fset     *token.FileSet
        globe    *types.Globe
        scope    *types.Scope
        registry *runtime.Registry
        modules  []*types.Module
        loading  []*loadingInfo
}

type loadingInfo struct {
        dir, file string
}

// Create and initialize a new interpreter.
func New() *Interpreter {
        globe := types.NewGlobe("interpreter")
        return &Interpreter{
                fset:     token.NewFileSet(), 
                globe:    globe,
                scope:    globe.Scope(),
                registry: runtime.NewRegistry(),
        }
}

func (i *Interpreter) newModule(pos token.Pos, kw token.Token, path, name string) *types.Module {
        m := types.NewModule(kw, path, name)
        modName := types.NewModuleName(pos, m, name, nil)
        i.scope.Insert(modName)

        i.scope = m.Scope()
        i.modules = append(i.modules, m)
        return m
}

func (i *Interpreter) currentModule() *types.Module {
        if n := len(i.modules); n > 0 {
                return i.modules[n-1]
        }
        return nil
}

func (i *Interpreter) upperScope() {
        if scope := i.scope.Parent(); !types.IsUniverse(scope) {
                i.scope = scope
        }
}

func (i *Interpreter) lookupAt(name string, pos token.Pos) (sym types.Symbol) {
        if sym = i.scope.Lookup(name); sym == nil {
                _, sym = i.scope.LookupParent(name, pos)
        }
        return
}

func (i *Interpreter) call(sym types.Symbol, args... interface{}) types.Value {
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

func (i *Interpreter) Call(name string, args... interface{}) types.Value {
        return i.call(i.lookupAt(name, token.NoPos), args...)
}

func (i *Interpreter) Run(targets... string) (err error) {
        var updated = 0
        if len(targets) == 0 {
                if entry := i.registry.GetDefaultEntry(); entry != nil {
                        if err = entry.Execute(); err == nil {
                                updated += 1
                        }
                }
        } else {
                for _, target := range targets {
                        entry := i.registry.Lookup(target)
                        if err = entry.Execute(); err == nil {
                                updated += 1
                        } else {
                                break
                        }
                }
        }
        //fmt.Printf("updated %v targets\n", updated)
        //return errors.New("TODO: run entry rules of projects")
        return
}
