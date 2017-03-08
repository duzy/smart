//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package interpreter

import (
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/types"
)

type Interpreter struct {
        fset *token.FileSet
        loading []*loadingInfo
        scope *types.Scope
        modules map[string]*types.Module
}

type loadingInfo struct {
        dir, file string
}

// Create and initialize a new interpreter.
func New() *Interpreter {
        return &Interpreter{
                fset: token.NewFileSet(),
                scope: types.NewScope(types.Universe, token.NoPos, token.NoPos, "top scope"),
                modules: make(map[string]*types.Module),
        }
}

func (i *Interpreter) lookupAt(name string, pos token.Pos) (sym types.Symbol) {
        if sym = i.scope.Lookup(name); sym == nil {
                _, sym = i.scope.LookupParent(name, pos)
        }
        return
}

func (i *Interpreter) call(sym types.Symbol, args... interface{}) types.Value {
        if sym != nil {
                // TODO: expend a definition
        }
        return nil // FIXME: return value of empty string
}

func (i *Interpreter) Call(name string, args... interface{}) types.Value {
        return i.call(i.lookupAt(name, token.NoPos), args...)
}
