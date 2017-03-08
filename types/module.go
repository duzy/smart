//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "fmt"
)

type Module struct {
	path     string
	name     string
        scope    *Scope
        //node     ast.Module
        imports  map[string]*Module
        uses     map[string]*Use
	complete bool
}

// NewModule returns a new Module for the given module path and name;
// the name must not be the blank identifier.
// The module is not complete and contains no explicit imports.
func NewModule(path, name string) *Module {
	if name == "_" {
		panic("invalid module name _")
	}
	scope := NewScope(Universe, token.NoPos, token.NoPos, fmt.Sprintf("module %q", path))
	return &Module{path: path, name: name, scope: scope}
}
