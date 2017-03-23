//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "github.com/duzy/smart/token"
)

type Symbol interface {
        Parent() *Scope
        Module() *Module
        Name() string
        Type() Type
        Value() Value

        Callable() bool
        Call(/*ctx Context,*/ args... Value) Value

        String() string

	// order reflects a package-level symbol's source order: if symbol
	// a is before symbol b in the source, then a.order() < b.order().
	// order returns a value > 0 for package-level symbols; it returns
	// 0 for all other symbols (including symbols in file scopes).
	order() uint32

	// setParent sets the parent scope of the object.
	setParent(*Scope)

	// scopePos returns the start position of the scope of this Symbol
	scopePos() token.Pos

	// setScopePos sets the start position of the scope for this Symbol.
	setScopePos(pos token.Pos)
}

// An symbol implements the common parts of an Symbol.
type symbol struct {
        parent *Scope
        module *Module
        name string
        typ Type
        ord uint32
        pos token.Pos
        scopePos_ token.Pos
}

func (sym *symbol) Parent() *Scope        { return sym.parent }
func (sym *symbol) Module() *Module       { return sym.module }
func (sym *symbol) Name() string          { return sym.name }
func (sym *symbol) Type() Type            { return sym.typ }
func (sym *symbol) String() string        { panic("abstract") }
func (sym *symbol) Value() Value          { panic("abstract") }
func (sym *symbol) Callable() bool        { return false }
func (sym *symbol) Call(/*c Context,*/ a... Value) Value { panic("abstract") }
func (sym *symbol) order() uint32         { return sym.ord }
func (sym *symbol) scopePos() token.Pos   { return sym.scopePos_ }

func (sym *symbol) setParent(parent *Scope)   { sym.parent = parent }
func (sym *symbol) setOrder(order uint32)     { /*assert(order > 0);*/ sym.ord= order }
func (sym *symbol) setScopePos(pos token.Pos) { sym.scopePos_ = pos }

type ModuleName struct {
        symbol
        imported *Module
        used bool // set if the module was used
}

// Imported returns the module that was imported.
// It is distinct from Module(), which is the module
// containing the import statement.
func (n *ModuleName) Imported() *Module { return n.imported }

func NewModuleName(pos token.Pos, mod *Module, name string, imported *Module) *ModuleName {
	return &ModuleName{symbol{nil, mod, name, Invalid, 0, pos, token.NoPos}, imported, false}
}

// A Const represents a declared constant.
type Const struct {
        symbol
}

// A Def represents a definition.
type Def struct {
        symbol
        value Value
}

func (d *Def) String() string { return d.name+" = "+d.value.String() }
func (d *Def) Value() Value { return d.value }

func NewDef(pos token.Pos, mod *Module, name string, value Value) *Def {
        var typ = value.Type()
	return &Def{symbol{nil, mod, name, typ, 0, pos, token.NoPos}, value}
}

// A Auto represents a automatic definition.
type Auto struct {
        Def
}

func NewAuto(mod *Module, name string, value Value) *Auto {
        var (
                typ = value.Type()
                pos = token.NoPos
                end = token.NoPos
        )
	return &Auto{Def{symbol{nil, mod, name, typ, 0, pos, end}, value}}
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        symbol
        f BuiltinFunc
}

//func (p *Builtin) Value() Value { return p.Call() }
func (p *Builtin) Callable() bool { return true }
func (p *Builtin) Call(/*ctx Context,*/ a... Value) Value { return p.f(/*ctx,*/ a...) }

func NewBuiltin(name string, f BuiltinFunc) *Builtin {
        return &Builtin{symbol{
                parent: nil, 
                module: nil, 
                name: name, 
                typ: None,
                ord: 0,
                pos: token.NoPos,
                scopePos_: token.NoPos,
        }, f}
}
