//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package smart

import (
        "time"
        "fmt"
        "os"
)

var (
	universe *Scope

        modifierbar = &ModifierBar{}
        universalnone = &None{}
        universalyes = &answer{ true }
        universalno = &answer{ false }
        universaltrue = &boolean{ true }
        universalfalse = &boolean{ false }
)

// Predeclared types.
var (
        CoreTypes = []*Core {
                UnknownObjectKind:    {UnknownObjectKind, IsUnknownObject, "UnknownObject"},
                KnownObjectKind:      {KnownObjectKind, IsKnownObject, "KnownObject"},
                UnresolvedObjectKind: {UnresolvedObjectKind, IsUnresolvedObject, "UnresolvedObject"},
                DefKind:         {DefKind, IsDef, "Def"},
                UndeterminedKind:{UndeterminedKind, IsUndetermined, "Undetermined"},
                UsingKind:       {UsingKind, IsUsing, "using"},
                UsingListKind:   {UsingListKind, IsUsingList, "usinglist"},
                DefinerKind:     {DefinerKind, IsDefiner, "Definer"},
                PlainKind:       {PlainKind, IsPlain, "Plain"},
                JSONKind:        {JSONKind, IsJSON, "JSON"},
                XMLKind:         {XMLKind, IsXML, "XML"},
                YAMLKind:        {YAMLKind, IsYAML, "YAML"},
                ExecResultKind:  {ExecResultKind, IsExecResult, "ExecResult"},
                ScopeNameKind:   {ScopeNameKind, IsScopeName, "ScopeName"},
                ProjectNameKind: {ProjectNameKind, IsProjectName, "ProjectName"},
                BuiltinKind:     {BuiltinKind, IsBuiltin, "Builtin"},
                RuleEntryKind:   {RuleEntryKind, IsRuleEntry, "RuleEntry"},
        }
        
        BasicTypes = []*Basic {
                InvalidKind:  {InvalidKind, 0, "invalid"},
                AnyKind:      {AnyKind, IsAny, "any"},
                AnswerKind:   {AnswerKind, IsAnswer, "answer"},
                BooleanKind:  {BooleanKind, IsBoolean, "boolean"},
                BinKind:      {BinKind, IsBin, "bin"},
                OctKind:      {OctKind, IsOct, "oct"},
                IntKind:      {IntKind, IsInt, "int"},
                HexKind:      {HexKind, IsHex, "hex"},
                FloatKind:    {FloatKind, IsFloat, "float"},
                DateTimeKind: {DateTimeKind, IsDateTime, "datetime"},
                DateKind:     {DateKind, IsDate, "date"},
                TimeKind:     {TimeKind, IsTime, "time"},
                URLKind:      {URLKind, IsURL, "url"},
                RawKind:      {RawKind, IsRaw, "raw"},
                StringKind:   {StringKind, IsString, "string"},
                BarewordKind: {BarewordKind, IsBareword, "bareword"},
                BarefileKind: {BarefileKind, IsBarefile, "barefile"},
                GlobKind:     {GlobKind, IsGlob, "glob"}, // see filepath.Glob()
                PathSegKind:  {PathSegKind, IsPathSeg, "pathseg"},
                PathKind:     {PathKind, IsPath, "path"},
                FileKind:     {FileKind, IsFile, "file"},
                FlagKind:     {FlagKind, IsFlag, "flag"},
                NegativeKind: {NegativeKind, IsNegative, "negative"},
                NoneKind:     {NoneKind, IsNone, "none"},
        }
        
        CompositeTypes = []*Composite {
                CompoundKind:   {CompoundKind, IsCompound, "compound"},
                BarecompKind:   {BarecompKind, IsBarecomp, "barecomp"},
                ArgumentedKind: {ArgumentedKind, IsArgumented, "argumented"},
                ListKind:       {ListKind, IsList, "list"},
                GroupKind:      {GroupKind, IsGroup, "group"},
                MapKind:        {MapKind, IsMap, "map"},
                PairKind:       {PairKind, IsPair, "pair"},
                PercPatternKind:   {PercPatternKind, IsPercPattern, "perc_pattern"},
                GlobPatternKind:   {GlobPatternKind, IsGlobPattern, "glob_pattern"},
                RegexpPatternKind: {RegexpPatternKind, IsRegexpPattern, "regexp_pattern"},
                DelegateKind:   {DelegateKind, IsDelegate, "delegate"},
                ClosureKind:    {ClosureKind, IsClosure, "closure"},
                SelectionKind:  {SelectionKind, IsSelection, "selection"},
        }

        // Shortcuts of core types
        UnknownObjectType    = CoreTypes[UnknownObjectKind]
        KnownObjectType      = CoreTypes[KnownObjectKind]
        UnresolvedObjectType = CoreTypes[UnresolvedObjectKind]
        UndeterminedType = CoreTypes[UndeterminedKind]
        DefType          = CoreTypes[DefKind]
        UsingType        = CoreTypes[UsingKind]
        UsingListType    = CoreTypes[UsingListKind]
        DefinerType      = CoreTypes[DefinerKind]
        PlainType        = CoreTypes[PlainKind]
        JSONType         = CoreTypes[JSONKind]
        XMLType          = CoreTypes[XMLKind]
        YAMLType         = CoreTypes[YAMLKind]
        ExecResultType   = CoreTypes[ExecResultKind]
        BuiltinType      = CoreTypes[BuiltinKind]
        RuleEntryType    = CoreTypes[RuleEntryKind]
        ScopeNameType    = CoreTypes[ScopeNameKind]
        ProjectNameType  = CoreTypes[ProjectNameKind]

        // Shortcuts of basic types.
        InvalidType  = BasicTypes[InvalidKind]
        AnyType      = BasicTypes[AnyKind]
        AnswerType   = BasicTypes[AnswerKind]
        BooleanType  = BasicTypes[BooleanKind]
        BinType      = BasicTypes[BinKind]
        OctType      = BasicTypes[OctKind]
        IntType      = BasicTypes[IntKind]
        HexType      = BasicTypes[HexKind]
        FloatType    = BasicTypes[FloatKind]
        DateTimeType = BasicTypes[DateTimeKind]
        DateType     = BasicTypes[DateKind]
        TimeType     = BasicTypes[TimeKind]
        URLType      = BasicTypes[URLKind]
        RawType      = BasicTypes[RawKind]
        StringType   = BasicTypes[StringKind]
        BarewordType = BasicTypes[BarewordKind]
        BarefileType = BasicTypes[BarefileKind]
        GlobType     = BasicTypes[GlobKind]
        PathSegType  = BasicTypes[PathSegKind]
        PathType     = BasicTypes[PathKind]
        FileType     = BasicTypes[FileKind]
        FlagType     = BasicTypes[FlagKind]
        NegativeType = BasicTypes[NegativeKind]
        NoneType     = BasicTypes[NoneKind]

        // Shortcuts for composite types.
        CompoundType   = CompositeTypes[CompoundKind]
        BarecompType   = CompositeTypes[BarecompKind]
        ArgumentedType = CompositeTypes[ArgumentedKind]
        ListType       = CompositeTypes[ListKind]
        GroupType      = CompositeTypes[GroupKind]
        MapType        = CompositeTypes[MapKind]
        PairType       = CompositeTypes[PairKind]
        PercPatternType   = CompositeTypes[PercPatternKind]
        GlobPatternType   = CompositeTypes[GlobPatternKind]
        RegexpPatternType = CompositeTypes[RegexpPatternKind]
        DelegateType   = CompositeTypes[DelegateKind]
        ClosureType    = CompositeTypes[ClosureKind]
        SelectionType  = CompositeTypes[SelectionKind]
)

func defUniverseBuiltins() {
        for name, f := range builtins {
                if _, alt := universe.Builtin(name, f); alt != nil {
                        panic(fmt.Sprintf("builtin '%s' already defined", name))
                }
        }
}

func init() {
        universe = NewScope(nil, nil, "universe")

        bin, args := &String{ os.Args[0] }, new(List)
        for _, a := range os.Args[1:] {
                args.Elems = append(args.Elems, &String{ a })
        }
        _, _ = universe.Def(nil, "SMART.BIN", bin)
        _, _ = universe.Def(nil, "SMART.ARGS", args)
        _, _ = universe.Def(nil, "SMART", bin)
        
        defUniverseBuiltins()
}

// IsUniverse checks if the scope is universe.
func IsUniverse(scope *Scope) bool {
        return scope == universe
}

// A Globe represents a global execution context. 
type Globe struct {
        scope  *Scope
	unsafe *Project
        main   *Project
        timestamps map[string]time.Time
}

// Scope returns the globe scope.
func (g *Globe) Scope() *Scope { return g.scope }

// Main returns the main project.
func (g *Globe) Main() *Project { return g.main }

func (g *Globe) SetScopeOuter(scope *Scope) {
        scope.outer = g.scope
}

// project returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *Globe) project(outer *Scope, absPath, relPath, tmpPath, spec, name string) (m *Project) {
        if outer == nil {
                outer = g.scope
        }

	m = &Project{
                absPath: absPath,
                relPath: relPath, 
                tmpPath: tmpPath,
                usings: new(usinglist),
                spec: spec,
                name: name,
        }

        m.scope = NewScope(outer, m, fmt.Sprintf("project %q", name))
        m.usings.name = "use"
        m.usings.owner = m
        m.usings.scope = m.scope

        if g.main == nil && spec != "" && name != "@" && name != "~" {
                for outer != nil && outer != g.scope {
                        if p := outer.project; p != nil && p.Name() == "@" {
                                return
                        }
                        outer = outer.outer
                }
                g.main = m
        }
        return
}

// NewGlobe creates a new Globe context.
func NewGlobe(name string) *Globe {
        return &Globe{
                scope: NewScope(universe, nil, fmt.Sprintf("globe %q", name)),
                timestamps: make(map[string]time.Time),
        }
}
