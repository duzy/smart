//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        //"fmt"
)

type Module struct {
        keyword  token.Token
	path     string
	name     string
        scope    *Scope
        imports  []*Module
        uses     []*Use
	complete bool
}

func (m *Module) Keyword() token.Token { return m.keyword }
func (m *Module) Path() string { return m.path }
func (m *Module) Name() string { return m.name }
func (m *Module) Scope() *Scope { return m.scope }
func (m *Module) Imports() []*Module { return m.imports }
func (m *Module) Uses() []*Use { return m.uses }
func (m *Module) Complete() bool { return m.complete }

/* func (m *Module) AddImport(om *Module) {
        // ...
} */
