//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package types

import "strconv"

// A Type represents a type of Smart (borrowed from Go).
// All types implement the Type interface.
type Type interface {
	// Underlying returns the underlying type of a type.
	Underlying() Type

	// String returns a string representation of a type.
	String() string
}

type BasicKind int

const (
        Invalid BasicKind = iota
        
        // predeclared types
        Int
        Float
        DateTime
        Date
        Time
        Uri
        String
        Bareword

        // type for empty defines
        None
)

var basicNames = [...]string{
        Invalid:    "Invalid",
        Int:        "Int",
        Float:      "Float",
        DateTime:   "DateTime",
        Date:       "Date",
        Time:       "Time",
        Uri:        "Uri",
        String:     "String",
        Bareword:   "Bareword",
}

func (t BasicKind) String() (s string) {
	if 0 <= t && t < BasicKind(len(basicNames)) {
		s = basicNames[t]
	}
	if s == "" {
		s = "type(" + strconv.Itoa(int(t)) + ")"
	}
	return
}

// BasicInfo is a set of flags describing properties of a basic type.
type BasicInfo int

// Properties of basic types.
const (
	IsBoolean BasicInfo = 1 << iota
	IsInteger
	IsUnsigned
	IsFloat
	IsString
        IsDate     // Time type with date component
        IsTime     // Time type with time component
        IsUri
        //IsCompound
        //IsList
	IsNone

        IsDateTime  = IsDate | IsTime
	IsOrdered   = IsInteger | IsFloat | IsString
	IsNumeric   = IsInteger | IsFloat
	IsConstType = IsBoolean | IsNumeric | IsString
)

// A Basic represents a basic type.
type Basic struct {
	kind BasicKind
	info BasicInfo
	name string
}

type Compound struct {
        
}

type List struct {
}

type Named struct {
        underlying Type
}

func (t *Basic) Underlying() Type     { return t }
func (t *Compound) Underlying() Type  { return t }
func (t *List) Underlying() Type      { return t }
func (t *Named) Underlying() Type     { return t.underlying }

func (t *Basic) String() string     { return TypeString(t, nil) }
func (t *Compound) String() string  { return TypeString(t, nil) }
func (t *List) String() string      { return TypeString(t, nil) }
func (t *Named) String() string     { return TypeString(t, nil) }
