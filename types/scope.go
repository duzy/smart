//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        "strings"
        "bytes"
        "sort"
        "fmt"
        "io"
)

// A Scope maintains a set of symbols and links to its containing
// (parent) and contained (children) scopes. Symbols may be inserted
// and looked up by name. The zero value for Scope is a ready-to-use
// empty scope.
type Scope struct {
        parent *Scope
        children []*Scope
        elems map[string]Symbol
        pos, end token.Pos
        comment string
}

func NewScope(parent *Scope, pos, end token.Pos, comment string) *Scope {
        scope := &Scope{ parent, nil, nil, pos, end, comment }
 	// don't add children to Universe scope!
	if parent != nil && parent != universe {
		parent.children = append(parent.children, scope)
	}
       return scope
}

// Parent returns the scope's containing (parent) scope.
func (s *Scope) Parent() *Scope { return s.parent }

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

// Lookup returns the symbol in scope s with the given name if such an
// symbol exists; otherwise the result is nil.
func (s *Scope) Lookup(name string) Symbol {
	return s.elems[name]
}

// LookupParent follows the parent chain of scopes starting with s until
// it finds a scope where Lookup(name) returns a non-nil symbol, and then
// returns that scope and symbol. If a valid position pos is provided,
// only symbols that were declared at or before pos are considered.
// If no such scope and symbol exists, the result is (nil, nil).
//
// Note that sym.Parent() may be different from the returned scope if the
// symbol was inserted into the scope and already had a parent at that
// time (see Insert, below). This can only happen for dot-imported symbols
// whose scope is the scope of the package that exported them.
func (s *Scope) LookupParent(name string, pos token.Pos) (*Scope, Symbol) {
	for ; s != nil; s = s.parent {
		if sym := s.elems[name]; sym != nil && (!pos.IsValid() || sym.scopePos() <= pos) {
			return s, sym
		}
	}
	return nil, nil
}

// Insert attempts to insert an symbol sym into scope s.
// If s already contains an alternative symbol alt with
// the same name, Insert leaves s unchanged and returns alt.
// Otherwise it inserts sym, sets the symbol's parent scope
// if not already set, and returns nil.
func (s *Scope) Insert(sym Symbol) Symbol {
	name := sym.Name()
	if alt := s.elems[name]; alt != nil {
		return alt
	}
	if s.elems == nil {
		s.elems = make(map[string]Symbol)
	}
	s.elems[name] = sym
	if sym.Parent() == nil {
		sym.setParent(s)
	}
	return nil
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
	if s.parent == universe {
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
		fmt.Fprintf(w, "}\n")
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
