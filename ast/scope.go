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

// Lookup returns the object with the given name if it is
// found in scope s, otherwise it returns nil. Outer scopes
// are ignored.
//
func (s *Scope) Lookup(name string) *Symbol {
	return s.Symbols[name]
}

// Insert attempts to insert a named object obj into the scope s.
// If the scope already contains an object alt with the same name,
// Insert leaves the scope unchanged and returns alt. Otherwise
// it inserts obj and returns nil.
//
func (s *Scope) Insert(obj *Symbol) (alt *Symbol) {
	if alt = s.Symbols[obj.Name]; alt == nil {
		s.Symbols[obj.Name] = obj
	}
	return
}

// Debugging support
func (s *Scope) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "scope %p {", s)
	if s != nil && len(s.Symbols) > 0 {
		fmt.Fprintln(&buf)
		for _, obj := range s.Symbols {
			fmt.Fprintf(&buf, "\t%s %s\n", obj.Kind, obj.Name)
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
// The Data fields contains object-specific data:
//
//	Kind    Data type         Data value
//	Pkg	*types.Package    package scope
//	Con     int               iota for the respective declaration
//	Con     != nil            constant value
//	Typ     *Scope            (used as method scope during type checking - transient)
//
type Symbol struct {
	Kind SymKind
	Name string      // declared name
	Decl interface{} // corresponding Field, XxxSpec, FuncDecl, LabeledStmt, AssignStmt, Scope; or nil
	Data interface{} // object-specific data; or nil
	Type interface{} // placeholder for type information; may be nil
}

// NewSym creates a new object of a given kind and name.
func NewSym(kind SymKind, name string) *Symbol {
	return &Symbol{Kind: kind, Name: name}
}

// Pos computes the source position of the declaration of an object name.
// The result may be an invalid position if it cannot be computed
// (obj.Decl may be nil or not correct).
func (obj *Symbol) Pos() token.Pos {
        /*
	name := obj.Name
	switch d := obj.Decl.(type) {
	case *Field:
		for _, n := range d.Names {
			if n.Name == name {
				return n.Pos()
			}
		}
	case *ImportSpec:
		if d.Name != nil && d.Name.Name == name {
			return d.Name.Pos()
		}
		return d.Path.Pos()
	case *ValueSpec:
		for _, n := range d.Names {
			if n.Name == name {
				return n.Pos()
			}
		}
	case *TypeSpec:
		if d.Name.Name == name {
			return d.Name.Pos()
		}
	case *FuncDecl:
		if d.Name.Name == name {
			return d.Name.Pos()
		}
	case *LabeledStmt:
		if d.Label.Name == name {
			return d.Label.Pos()
		}
	case *AssignStmt:
		for _, x := range d.Lhs {
			if ident, isIdent := x.(*Ident); isIdent && ident.Name == name {
				return ident.Pos()
			}
		}
	case *Scope:
		// predeclared object - nothing to do for now
	} */
	return token.NoPos
}

// SymKind describes what an object represents.
type SymKind int

// The list of possible Symbol kinds.
const (
	Bad SymKind = iota // for error handling
	Pkg                // package
	Con                // constant
	Typ                // type
	Def                // definition or method
)

var symKindStrings = [...]string{
	Bad: "bad",
	Pkg: "package",
	Con: "const",
	Typ: "type",
	Def: "define",
}

func (kind SymKind) String() string { return symKindStrings[kind] }
