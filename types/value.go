//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import (
        "github.com/duzy/smart/token"
        //"strconv"
        //"time"
)

type Value interface {
        Type() Type
        String() string
}

type Literal struct {
        typ Type
        pos token.Pos
}

func NewLiteral(k BasicKind, pos token.Pos) Literal {
        return Literal{typ: Types[k], pos: pos}
}

type CompoundValue struct {
        pos, end token.Pos
        elems []Value
}

func (v *CompoundValue) String() (s string) {
        for _, e := range v.elems {
                s += e.String()
        }
        return
}

func NewCompound(pos, end token.Pos, values... Value) *CompoundValue {
        return &CompoundValue{pos: pos, end: end, elems: values}
}

type ListValue struct {
        pos, end token.Pos
        elems []Value
}

func (v *ListValue) String() (s string) {
        s = "("
        for i, e := range v.elems {
                if 0 < i {
                        s += " "
                }
                s += e.String()
        }
        s += ")"
        return
}

func NewList(pos, end token.Pos, values... Value) *ListValue {
        return &ListValue{pos: pos, end: end, elems: values}
}

type MapValue struct {
        pos, end token.Pos
        elems map[string]Value
}

func (v *Literal) Type() Type       { return v.typ }
func (v *CompoundValue) Type() Type { return nil }
func (v *ListValue) Type() Type     { return nil }
func (v *MapValue) Type() Type      { return nil }
