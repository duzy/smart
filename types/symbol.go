//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "github.com/duzy/smart/token"
)

// Symbol is a value defined in a scope.
//
// TODO: defines SymInfo to classify symbols.
// 
type Symbol interface {
        Parent() *Scope
        Module() *Module
        Name() string
        Type() Type // the type of the symbol, differs from the value type
        Value() Value

        Call(args... Value) (Value, error)

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
	setScopePos(pos token.Pos) // FIXME: it's not applied
}

// An symbol implements the common parts of an Symbol.
type symbol struct {
        parent *Scope
        module *Module
        name string
        typ Type
        ord uint32
        pos token.Pos
        scopos token.Pos
}

func (sym *symbol) Parent() *Scope        { return sym.parent }
func (sym *symbol) Module() *Module       { return sym.module }
func (sym *symbol) Name() string          { return sym.name }
func (sym *symbol) Type() Type            { return sym.typ }
func (sym *symbol) String() string        { return "" }
func (sym *symbol) Value() Value          { return nil }
func (sym *symbol) Call(a... Value) (Value, error) { return sym.Value(), nil }
func (sym *symbol) order() uint32         { return sym.ord }
func (sym *symbol) scopePos() token.Pos   { return sym.scopos }

func (sym *symbol) setParent(parent *Scope)   { sym.parent = parent }
func (sym *symbol) setOrder(order uint32)     { /*assert(order > 0);*/ sym.ord = order }
func (sym *symbol) setScopePos(pos token.Pos) { sym.scopos = pos }

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
	return &ModuleName{symbol{nil, mod, name, ModuleNameType, 0, pos, token.NoPos}, imported, false}
}

// A Const represents a declared constant.
/* type Const struct {
        symbol
} */

// A Def represents a definition.
type Def struct {
        symbol
        value Value
}

func (d *Def) String() string  { return d.name+" = "+d.value.String() }
func (d *Def) Set(v Value)     { d.value = v }
func (d *Def) Value() Value    { return d.value }
func (d *Def) Call(a... Value) (Value, error) {
        return d.value, nil 
}

func NewDef(pos token.Pos, mod *Module, name string, value Value) *Def {
	return &Def{symbol{nil, mod, name, DefineType, 0, pos, token.NoPos}, value}
}

func NewAuto(mod *Module, name string, value Value) *Def {
        return NewDef(token.NoPos, mod, name, value)
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        symbol
        f BuiltinFunc
}

func (p *Builtin) Call(a... Value) (Value, error) { return p.f(a...) }

func NewBuiltin(name string, f BuiltinFunc) *Builtin {
        return &Builtin{symbol{
                parent: nil, 
                module: nil, 
                name: name, 
                typ: BuiltinType,
                ord: 0,
                pos: token.NoPos,
                scopos: token.NoPos,
        }, f}
}

// RuleEntry represents a declared rule entry.
type RuleEntry struct {
        symbol
        program Program
}

// RuleEntry.Program returns the rule program.
func (entry *RuleEntry) Program() Program { return entry.program }

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
//
// TODO: merge Execute and Call, make RuleEntry behaves like a Def
// 
func (entry *RuleEntry) Call(a... Value) (result Value, err error) {
        if entry.program != nil {
                result, err = entry.program.Execute(entry.name, a, false)
        }
        return
}
