//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import "github.com/duzy/smart/token"

// Value represents a value of a type.
type Value interface {
        // Pos returns the position of the value occurs position in file or nil.
        //Pos() *token.Position
        
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

type Definer interface {
        Value
        Define(p *Project) (Value, error)
}

type Caller interface {
        Value
        Call(args... Value) (Value, error)
}

type Poser interface {
        Value
        Pos() *token.Position
}

type positional struct {
        Value
        pos *token.Position
}

func (p *positional) Pos() *token.Position { return p.pos }

// Positional wraps a value with a valid position
func Positional(v Value, pos *token.Position) Poser {
        if p, ok := v.(*positional); ok {
                p.pos = pos
                return p
        }
        return &positional{ v, pos }
}

type Holder struct {
        I interface{}
}

