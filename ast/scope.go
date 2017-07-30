//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package ast

// A Scope maintains the set of named language entities declared
// in the scope and a link to the immediately surrounding (outer)
// scope.
//
type Scope interface {
	OuterScope() Scope
        Resolve(name string) Symbol
        Symbol(name string) (sym, alt Symbol)
        Entry(name string) (sym, alt Symbol)
        //String() string // Debugging support
}

type Symbol interface {
        Name() string
}
