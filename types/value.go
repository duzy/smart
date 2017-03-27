//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import "github.com/duzy/smart/token"

// Value represents a value of a type.
type Value interface {
        // Presented position in file, or token.NoPos if not literals.
        Pos() token.Pos
        
        // Type returns the underlying type of the value.
        Type() Type

        // Lit returns the literal representations of the value.
        Lit() string

        // String returns the string form of the value.
        String() string

        // Integer returns the integer form of the value.
        Integer() int64

        // Float returns the float form of the value.
        Float() float64
}
