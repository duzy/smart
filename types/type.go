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

        // Kind of the type
        Kind() Kind

        // Type classification info
        Bits() TypeBits
}

type Kind int

// kinds for predeclared types
const (
        InvalidKind Kind = iota

        AnyKind

        // basic types
        IntKind
        FloatKind
        DateTimeKind
        DateKind
        TimeKind
        UriKind
        StringKind
        BarewordKind
        BarefileKind
        PathKind
        FileKind
        FlagKind
        
        // composite types
        CompoundKind
        BarecompKind
        ListKind
        GroupKind
        MapKind
        PairKind
        ClosureKind

        // named types
        NamedKind

        // symbolic types
        DefineKind
        BuiltinKind
        RuleEntryKind
        ProjectNameKind
        ScopeNameKind

        // type for expressions compute to nothing/empty
        NoneKind
)

var (
        typeNames = [...]string{
                InvalidKind:    "Invalid",
                AnyKind:        "Any",
                IntKind:        "Int",
                FloatKind:      "Float",
                DateTimeKind:   "DateTime",
                DateKind:       "Date",
                TimeKind:       "Time",
                UriKind:        "Uri",
                StringKind:     "String",
                BarewordKind:   "Bareword",
                BarefileKind:   "Barefile",
                PathKind:       "Path",
                FileKind:       "File",
                FlagKind:       "Flag",
                CompoundKind:   "Compound",
                ListKind:       "List",
                GroupKind:      "Group",
                MapKind:        "Map",
                PairKind:       "Pair",
                ClosureKind:    "Closure",
                NamedKind:      "Named",
                DefineKind:     "Define",
                BuiltinKind:    "Builtin",
                RuleEntryKind:  "RuleEntry",
                ProjectNameKind: "ProjectName",
                ScopeNameKind:  "ScopeName",
                NoneKind:       "None",
        }
)

func (t Kind) String() (s string) {
	if 0 <= t && t < Kind(len(typeNames)) {
		s = typeNames[t]
	}
	if s == "" {
		s = "type(" + strconv.Itoa(int(t)) + ")"
	}
	return
}

// TypeInfo is a set of flags describing properties of a basic type.
type TypeBits int
const (
        // Properties of basic types.
	IsBoolean TypeBits = 1 << iota
        IsAny
	IsInteger
	IsUnsigned
	IsFloat
	IsString
        IsDate     // Time type with date component
        IsTime     // Time type with time component
        IsUri
        IsBareword
        IsBarefile
        IsPath
        IsFile
        IsFlag
	IsNone

        // Properties of composite types.
        IsCompound
        IsBarecomp
        IsList
        IsGroup
        IsMap
        IsPair // key-value pair
        IsClosure

        // Custom type
        IsNamed

        // Symbolic types
        IsDefine
        IsBuiltin
        IsRuleEntry
        IsProjectName
        IsScopeName

        IsCore      = IsDefine | IsBuiltin | IsRuleEntry | IsProjectName | IsScopeName
        IsSymbolic  = IsCore
        
        IsDateTime  = IsDate | IsTime
	IsNumeric   = IsInteger | IsFloat
        IsKeyName   = IsInteger | IsString | IsBareword
	IsOrdered   = IsNumeric | IsDateTime | IsString | IsUri | IsBareword | IsBarefile | IsPath | IsFlag
	IsBasic     = IsBoolean | IsOrdered | IsNone
        IsComposite = IsCompound | IsBarecomp | IsList | IsGroup | IsMap | IsPair
        IsConstType = IsBasic
)

type Core struct {
	kind Kind
	info TypeBits
	name string
}

func (t *Core) String() string   { return TypeString(t, nil) }
func (t *Core) Underlying() Type { return t }
func (t *Core) Kind() Kind       { return t.kind }
func (t *Core) Bits() TypeBits   { return t.info }

// A Basic represents a basic type.
type Basic struct {
	kind Kind
	info TypeBits
	name string
}

func (t *Basic) String() string   { return TypeString(t, nil) }
func (t *Basic) Underlying() Type { return t }
func (t *Basic) Kind() Kind       { return t.kind }
func (t *Basic) Bits() TypeBits   { return t.info }
func (t *Basic) IsBoolean() bool  { return t.info&IsBoolean != 0 }
func (t *Basic) IsInteger() bool  { return t.info&IsInteger != 0 }
func (t *Basic) IsUnsigned() bool { return t.info&IsUnsigned != 0 }
func (t *Basic) IsFloat() bool    { return t.info&IsFloat != 0 }
func (t *Basic) IsNumeric() bool  { return t.info&IsNumeric != 0 }
func (t *Basic) IsString() bool   { return t.info&IsString != 0 }
func (t *Basic) IsDate() bool     { return t.info&IsDate != 0 }
func (t *Basic) IsTime() bool     { return t.info&IsTime != 0 }
func (t *Basic) IsDateTime() bool { return t.info&IsDateTime != 0 }
func (t *Basic) IsUri() bool      { return t.info&IsUri != 0 }
func (t *Basic) IsBareword() bool { return t.info&IsBareword != 0 }
func (t *Basic) IsBarefile() bool { return t.info&IsBarefile != 0 }
func (t *Basic) IsPath() bool     { return t.info&IsPath != 0 }
func (t *Basic) IsFile() bool     { return t.info&IsFile != 0 }
func (t *Basic) IsFlag() bool     { return t.info&IsFlag != 0 }
func (t *Basic) IsNone() bool     { return t.info&IsNone != 0 }

type Composite struct {
	kind Kind
	info TypeBits
	name string
}

func (t *Composite) String() string   { return TypeString(t, nil) }
func (t *Composite) Underlying() Type { return t }
func (t *Composite) Kind() Kind       { return t.kind }
func (t *Composite) Bits() TypeBits   { return t.info }
func (t *Composite) IsCompound() bool { return t.info&IsCompound != 0 }
func (t *Composite) IsList() bool     { return t.info&IsList != 0 }
func (t *Composite) IsGroup() bool    { return t.info&IsGroup != 0 }
func (t *Composite) IsMap() bool      { return t.info&IsMap != 0 }
func (t *Composite) IsPair() bool     { return t.info&IsPair != 0 }

type Named struct {
        underlying Type
	name string
        value
}

func (t *Named) String() string   { return TypeString(t, nil) }
func (t *Named) Name() string     { return t.name }
func (t *Named) Underlying() Type { return t.underlying }
func (t *Named) Kind() Kind       { return NamedKind }
func (t *Named) Bits() TypeBits   { return IsNamed }
