//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package ast

import (
	"bytes"
	"fmt"
        "github.com/duzy/smart/token"
)

// A Scope maintains the set of named language entities declared
// in the scope and a link to the immediately surrounding (outer)
// scope.
//
type Scope struct {
	Outer   *Scope
	Symbols map[string]*Symbol
}

// NewScope creates a new scope nested in the outer scope.
func NewScope(outer *Scope) *Scope {
	const n = 4 // initial scope capacity
	return &Scope{outer, make(map[string]*Symbol, n)}
}

// Lookup returns the symbol with the given name if it is
// found in scope s, otherwise it returns nil. Outer scopes
// are ignored.
//
func (s *Scope) Lookup(name string) *Symbol {
	return s.Symbols[name]
}

// Insert attempts to insert a named symbol sym into the scope s.
// If the scope already contains an symbol alt with the same name,
// Insert leaves the scope unchanged and returns alt. Otherwise
// it inserts sym and returns nil.
//
func (s *Scope) Insert(sym *Symbol) (alt *Symbol) {
	if alt = s.Symbols[sym.Name]; alt == nil {
		s.Symbols[sym.Name] = sym
	}
	return
}

// Debugging support
func (s *Scope) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "scope %p {", s)
	if s != nil && len(s.Symbols) > 0 {
		fmt.Fprintln(&buf)
		for _, sym := range s.Symbols {
			fmt.Fprintf(&buf, "\t%s %s\n", sym.Kind, sym.Name)
		}
	}
	fmt.Fprintf(&buf, "}\n")
	return buf.String()
}

// ----------------------------------------------------------------------------
// Symbols

// An Symbol describes a named language entity such as a package,
// constant, definition, or label.
//
// The Data fields contains symbol-specific data:
//
//	Kind    Data type         Data value
//	Pro	*types.Project    project scope
//	Mod	*types.Module     module scope
//	Rul	*types.RuleEntry  rule scope
//	Def	*types.Define     definition value
//	Con     != nil            constant value
//
type Symbol struct {
	Kind SymKind
	Name string      // declared name
	Decl interface{} // corresponding declaration; or nil
	Data interface{} // symbol-specific data; or nil
	Type interface{} // placeholder for type information; may be nil
}

// NewSym creates a new symbol of a given kind and name.
func NewSym(kind SymKind, name string) *Symbol {
	return &Symbol{Kind: kind, Name: name}
}

// Pos computes the source position of the declaration of an symbol name.
// The result may be an invalid position if it cannot be computed
// (sym.Decl may be nil or not correct).
func (sym *Symbol) Pos() token.Pos {
	//name := sym.Name
	switch d := sym.Decl.(type) {
	case *DefineClause:
                return d.TokPos

	case *Scope:
		// predeclared symbol - nothing to do for now
	}
	return token.NoPos
}

// SymKind describes what an symbol represents.
type SymKind int

// The list of possible Symbol kinds.
const (
	Bad SymKind = iota // for error handling
	Pro                // project
	Mod                // module
	Def                // definition
	Rul                // rule
	Con                // constant
)

var symKindStrings = [...]string{
	Bad: "bad",
	Pro: "project",
	Mod: "module",
	Def: "define",
	Rul: "rule",
	Con: "const",
}

func (kind SymKind) String() string { return symKindStrings[kind] }
