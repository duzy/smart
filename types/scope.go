//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/ast"
        "strings"
        "bytes"
        "sort"
        "fmt"
        "io"
)

// A Scope maintains a set of objects and links to its containing
// (parent) and contained (children) scopes. Objects may be inserted
// and looked up by name. The zero value for Scope is a ready-to-use
// empty scope.
type Scope struct {
        outer *Scope
        chain []*Scope
        children []*Scope
        elems map[string]Object
        pos, end token.Pos
        comment string
}

func NewScope(outer *Scope, pos, end token.Pos, comment string) *Scope {
        scope := &Scope{ outer, nil, nil, nil, pos, end, comment }
 	// don't add children to Universe scope!
	if outer != nil && outer != universe {
		outer.children = append(outer.children, scope)
	}
        return scope
}

func (s *Scope) Comment() string { return s.comment }

// Outer returns the scope's containing (outer) scope.
func (s *Scope) Outer() *Scope { return s.outer }

// Len() returns the number of scope elements.
func (s *Scope) Len() int { return len(s.elems) }

// Names returns the scope's element names in sorted order.
func (s *Scope) Names() []string {
	names := make([]string, len(s.elems))
	i := 0
	for name := range s.elems {
		names[i] = name
		i++
	}
	sort.Strings(names)
	return names
}

// NumChildren() returns the number of scopes nested in s.
func (s *Scope) NumChildren() int { return len(s.children) }

// Child returns the i'th child scope for 0 <= i < NumChildren().
func (s *Scope) Child(i int) *Scope { return s.children[i] }

// Lookup returns the object in scope s with the given name if such an
// object exists; otherwise the result is nil.
func (s *Scope) Lookup(name string) Object {
	return s.elems[name]
}

// FindChainUp follows the outer chain of scopes starting with s until
// it finds a scope where Lookup(name) returns a non-nil object, and then
// returns that scope and object. If a valid position pos is provided,
// only objects that were declared at or before pos are considered.
// If no such scope and object exists, the result is (nil, nil).
//
// Note that obj.Outer() may be different from the returned scope if the
// object was inserted into the scope and already had a outer at that
// time (see Insert, below). This can only happen for dot-imported objects
// whose scope is the scope of the package that exported them.
func (s *Scope) FindChainUp(name string, pos token.Pos) (*Scope, Object) {
        for _, p := range s.chain {
                if p, obj := p.FindAt(pos, name); obj != nil {
                        return p, obj
                }
        }
	for p := s; p != nil; p = p.outer {
		if obj := p.elems[name]; obj != nil && (!pos.IsValid() || obj.scopePos() <= pos) {
			return p, obj
		}
	}
	return nil, nil
}

func (s *Scope) FindAt(pos token.Pos, name string) (*Scope, Object) {
        if obj := s.Lookup(name); obj == nil {
                return s.FindChainUp(name, pos)
        } else {
                return s, obj
        }
}

func (s *Scope) Find(name string) (obj Object) {
        _, obj = s.FindAt(token.NoPos, name)
        return 
}

// Insert attempts to insert an object obj into scope s.
// If s already contains an alternative object alt with
// the same name, Insert leaves s unchanged and returns alt.
// Otherwise it inserts obj, sets the object's outer scope
// if not already set, and returns nil.
func (s *Scope) Insert(obj Object) Object {
	name := obj.Name()
	if alt := s.elems[name]; alt != nil {
		return alt
	}
        s.replace(name, obj)
	return nil
}

func (s *Scope) replace(name string, obj Object) {
	if s.elems == nil {
		s.elems = make(map[string]Object)
	}
	s.elems[name] = obj
	if obj.Parent() == nil {
		obj.setParent(s)
	}
}

// Pos and End describe the scope's source code extent [pos, end).
// The results are guaranteed to be valid only if the type-checked
// AST has complete position information. The extent is undefined
// for Universe and package scopes.
func (s *Scope) Pos() token.Pos { return s.pos }
func (s *Scope) End() token.Pos { return s.end }

// Contains returns true if pos is within the scope's extent.
// The result is guaranteed to be valid only if the type-checked
// AST has complete position information.
func (s *Scope) Contains(pos token.Pos) bool {
	return s.pos <= pos && pos < s.end
}

// Innermost returns the innermost (child) scope containing
// pos. If pos is not within any scope, the result is nil.
// The result is also nil for the Universe scope.
// The result is guaranteed to be valid only if the type-checked
// AST has complete position information.
func (s *Scope) Innermost(pos token.Pos) *Scope {
	// Package scopes do not have extents since they may be
	// discontiguous, so iterate over the package's files.
	if s.outer == universe {
		for _, s := range s.children {
			if inner := s.Innermost(pos); inner != nil {
				return inner
			}
		}
	}

	if s.Contains(pos) {
		for _, s := range s.children {
			if s.Contains(pos) {
				return s.Innermost(pos)
			}
		}
		return s
	}
	return nil
}

// WriteTo writes a string representation of the scope to w,
// with the scope elements sorted by name.
// The level of indentation is controlled by n >= 0, with
// n == 0 for no indentation.
// If recurse is set, it also writes nested (children) scopes.
func (s *Scope) WriteTo(w io.Writer, n int, recurse bool) {
	const ind = ".  "
	indn := strings.Repeat(ind, n)

	fmt.Fprintf(w, "%s%s scope %p {", indn, s.comment, s)
	if len(s.elems) == 0 {
		fmt.Fprintf(w, "}")
		return
	}

	fmt.Fprintln(w)
	indn1 := indn + ind
	for _, name := range s.Names() {
		fmt.Fprintf(w, "%s%s\n", indn1, s.elems[name])
	}

	if recurse {
		for _, s := range s.children {
			fmt.Fprintln(w)
			s.WriteTo(w, n+1, recurse)
		}
	}

	fmt.Fprintf(w, "%s}", indn)
}

// String returns a string representation of the scope, for debugging.
func (s *Scope) String() string {
	var buf bytes.Buffer
	s.WriteTo(&buf, 0, false)
	return buf.String()
}

func (s *Scope) OuterScope() ast.Scope { return s.Outer() }

func (s *Scope) Resolve(name string) (sym ast.Symbol) {
        sym = s.Find(name)
        return
}

var none = new(NoneValue)
func (s *Scope) Symbol(name string) (sym, alt ast.Symbol) {
        sym, alt = s.InsertNewDef(nil/* FIXME */, name, none)
        return
}

func (s *Scope) Entry(name string) (sym, alt ast.Symbol) {
        sym, alt = s.InsertNewRuleEntry(nil/* FIXME */, GeneralRuleEntry, name)
        return
}
