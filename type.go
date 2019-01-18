//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

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

type Kind uint64

// kinds for predeclared types
const (
        InvalidKind Kind = iota

        AnyKind

        // basic types
        AnswerKind
        BooleanKind
        BinKind
        OctKind
        IntKind
        HexKind
        FloatKind
        DateTimeKind
        DateKind
        TimeKind
        URLKind
        RawKind
        StringKind
        BarewordKind
        BarefileKind
        GlobKind
        PathSegKind
        PathKind
        FileKind
        FlagKind

        NegativeKind

        // composite types
        CompoundKind
        BarecompKind
        ArgumentedKind
        ListKind
        GroupKind
        MapKind
        PairKind
        PercPatternKind
        GlobPatternKind
        RegexpPatternKind
        DelegateKind
        ClosureKind
        SelectionKind

        // object types
        UnknownObjectKind
        KnownObjectKind
        UnresolvedObjectKind

        BuiltinKind
        DefKind
        UndeterminedKind
        RuleEntryKind
        PatternEntryKind
        StemmedEntryKind
        ProjectNameKind
        ScopeNameKind

        // special types
        UsingKind
        UsingListKind
        
        DefinerKind
        PlainKind
        JSONKind
        XMLKind
        YAMLKind
        ExecResultKind

        // type for expressions compute to nothing/empty
        NoneKind
)

var (
        typeNames = [...]string{
                InvalidKind:    "Invalid",
                AnyKind:        "Any",
                AnswerKind:     "Answer",
                BooleanKind:    "Boolean",
                BinKind:        "Bin",
                OctKind:        "Oct",
                IntKind:        "Int",
                HexKind:        "Hex",
                FloatKind:      "Float",
                DateTimeKind:   "DateTime",
                DateKind:       "Date",
                TimeKind:       "Time",
                URLKind:        "URL",
                RawKind:        "Raw",
                StringKind:     "String",
                BarewordKind:   "Bareword",
                BarefileKind:   "Barefile",
                GlobKind:       "Glob",
                PathSegKind:    "PathSeg",
                PathKind:       "Path",
                FileKind:       "File",
                FlagKind:       "Flag",
                NegativeKind:   "Negative",
                CompoundKind:   "Compound",
                BarecompKind:   "Barecomp",
                ArgumentedKind: "Argumented",
                ListKind:       "List",
                GroupKind:      "Group",
                MapKind:        "Map",
                PairKind:       "Pair",
                PercPatternKind:    "PercPattern",
                GlobPatternKind:    "GlobPattern",
                RegexpPatternKind:  "RegexpPattern",
                DelegateKind:   "Delegate",
                ClosureKind:    "Closure",
                SelectionKind:  "Selection",
                UnknownObjectKind:    "UnknownObject",
                KnownObjectKind:      "KnownObject",
                UnresolvedObjectKind: "UnresolvedObject",
                BuiltinKind:    "Builtin",
                DefKind:        "Def",
                UndeterminedKind: "Undetermined",
                RuleEntryKind:  "RuleEntry",
                PatternEntryKind:  "PatternEntry",
                StemmedEntryKind:  "StemmedEntry",
                ProjectNameKind: "ProjectName",
                ScopeNameKind:  "ScopeName",
                UsingKind:      "Using",
                UsingListKind:  "UsingList",
                DefinerKind:    "Definer",
                PlainKind:      "Plain",
                JSONKind:       "JSON",
                XMLKind:        "XML",
                YAMLKind:       "YAML",
                ExecResultKind: "ExecResult",
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
type TypeBits uint64
const (
        IsAny TypeBits = 1 << iota

	IsNone

	IsAnswer
	IsBoolean
	IsBin
	IsOct
	IsInt
	IsHex
	IsUnsigned
	IsFloat
	IsRaw
	IsString
        IsDate     // Time type with date component
        IsTime     // Time type with time component
        IsURL
        IsBareword
        IsBarefile
        IsGlob
        IsPathSeg
        IsPath

        IsFile
        IsFlag
        
        IsNegative

        // Properties of composite types.
        IsCompound
        IsBarecomp
        IsArgumented
        IsList
        IsGroup
        IsMap
        IsPair // name-value pair
        IsPercPattern // percent pattern
        IsGlobPattern // glob pattern
        IsRegexpPattern // regexp pattern
        IsDelegate // $(foo ...)
        IsClosure  // &(foo ...)
        IsSelection  // foo->bar  foo=>bar

        // Object types
        IsUnknownObject
        IsKnownObject
        IsUnresolvedObject
        IsBuiltin
        IsDef
        IsUndetermined
        IsRuleEntry
        IsPatternEntry
        IsStemmedEntry
        IsScopeName
        IsProjectName

        IsUsing
        IsUsingList
        IsDefiner
        IsPlain
        IsJSON
        IsXML
        IsYAML
        IsExecResult

        IsPattern   = IsPercPattern | IsGlobPattern | IsRegexpPattern

        IsDateTime  = IsDate | IsTime
	IsNumeric   = IsAnswer | IsBoolean | IsBin | IsOct | IsInt | IsHex | IsFloat
	IsOrdered   = IsNumeric | IsDateTime | IsString | IsCompound | IsURL | IsBareword | IsBarecomp | IsBarefile | IsPath | IsPathSeg | IsFlag
        IsKeyName   = IsNumeric | IsOrdered | IsAnswer | IsBoolean
	IsBasic     = IsAnswer | IsBoolean | IsOrdered | IsNone
        IsComposite = IsCompound | IsBarecomp | IsArgumented | IsList | IsGroup | IsMap | IsPair | IsPattern
        IsConstType = IsBasic

        IsInternal  = IsUnknownObject | IsKnownObject | IsUnresolvedObject | IsClosure | IsDelegate
        IsCore      = IsInternal | IsBuiltin | IsDef | IsRuleEntry | IsPatternEntry | IsStemmedEntry | IsProjectName | IsScopeName | IsDefiner
        IsObject    = IsInternal | IsBuiltin | IsDef | IsRuleEntry | IsPatternEntry | IsStemmedEntry | IsProjectName | IsScopeName

        // Custom type
        IsNamed     = IsObject | IsPair | IsProjectName | IsScopeName
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
func (t *Basic) IsBin() bool      { return t.info&IsBin != 0 }
func (t *Basic) IsOct() bool      { return t.info&IsOct != 0 }
func (t *Basic) IsInt() bool      { return t.info&IsInt != 0 }
func (t *Basic) IsHex() bool      { return t.info&IsHex != 0 }
func (t *Basic) IsUnsigned() bool { return t.info&IsUnsigned != 0 }
func (t *Basic) IsFloat() bool    { return t.info&IsFloat != 0 }
func (t *Basic) IsNumeric() bool  { return t.info&IsNumeric != 0 }
func (t *Basic) IsString() bool   { return t.info&IsString != 0 }
func (t *Basic) IsDate() bool     { return t.info&IsDate != 0 }
func (t *Basic) IsTime() bool     { return t.info&IsTime != 0 }
func (t *Basic) IsDateTime() bool { return t.info&IsDateTime != 0 }
func (t *Basic) IsURL() bool      { return t.info&IsURL != 0 }
func (t *Basic) IsBareword() bool { return t.info&IsBareword != 0 }
func (t *Basic) IsBarefile() bool { return t.info&IsBarefile != 0 }
func (t *Basic) IsGlob() bool     { return t.info&IsGlob != 0 }
func (t *Basic) IsPathSeg() bool  { return t.info&IsPathSeg != 0 }
func (t *Basic) IsPath() bool     { return t.info&IsPath != 0 }
func (t *Basic) IsFile() bool     { return t.info&IsFile != 0 }
func (t *Basic) IsFlag() bool     { return t.info&IsFlag != 0 }
func (t *Basic) IsNegative() bool { return t.info&IsNegative != 0 }
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
