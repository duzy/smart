//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package types

import (
        "fmt"
        "os"
)

var (
	universe *Scope

        UniversalNone = &None{}
)

// Predeclared types.
var (
        CoreTypes = []*Core {
                DefKind:         {DefKind, IsDef, "Def"},
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
                BinKind:      {BinKind, IsBin, "bin"},
                OctKind:      {OctKind, IsOct, "oct"},
                IntKind:      {IntKind, IsInt, "int"},
                HexKind:      {HexKind, IsHex, "hex"},
                FloatKind:    {FloatKind, IsFloat, "float"},
                DateTimeKind: {DateTimeKind, IsDateTime, "datetime"},
                DateKind:     {DateKind, IsDate, "date"},
                TimeKind:     {TimeKind, IsTime, "time"},
                UriKind:      {UriKind, IsUri, "uri"},
                StringKind:   {StringKind, IsString, "string"},
                BarewordKind: {BarewordKind, IsBareword, "bareword"},
                BarefileKind: {BarefileKind, IsBarefile, "barefile"},
                GlobKind:     {GlobKind, IsGlob, "glob"}, // see filepath.Glob()
                PathSegKind:  {PathSegKind, IsPathSeg, "pathseg"},
                PathKind:     {PathKind, IsPath, "path"},
                FileKind:     {FileKind, IsFile, "file"},
                FlagKind:     {FlagKind, IsFlag, "flag"},
                NoneKind:     {NoneKind, IsNone, "none"},
        }
        
        CompositeTypes = []*Composite {
                CompoundKind: {CompoundKind, IsCompound, "compound"},
                BarecompKind: {BarecompKind, IsBarecomp, "barecomp"},
                ListKind:     {ListKind, IsList, "list"},
                GroupKind:    {GroupKind, IsGroup, "group"},
                MapKind:      {MapKind, IsMap, "map"},
                PairKind:     {PairKind, IsPair, "pair"},
                DelegateKind: {DelegateKind, IsDelegate, "delegate"},
                ClosureKind:  {ClosureKind, IsClosure, "closure"},
        }

        // Shortcuts of core types
        DefType         = CoreTypes[DefKind]
        DefinerType     = CoreTypes[DefinerKind]
        PlainType       = CoreTypes[PlainKind]
        JSONType        = CoreTypes[JSONKind]
        XMLType         = CoreTypes[XMLKind]
        YAMLType        = CoreTypes[YAMLKind]
        ExecResultType  = CoreTypes[ExecResultKind]
        BuiltinType     = CoreTypes[BuiltinKind]
        RuleEntryType   = CoreTypes[RuleEntryKind]
        ScopeNameType   = CoreTypes[ScopeNameKind]
        ProjectNameType = CoreTypes[ProjectNameKind]

        // Shortcuts of basic types.
        InvalidType  = BasicTypes[InvalidKind]
        AnyType      = BasicTypes[AnyKind]
        BinType      = BasicTypes[BinKind]
        OctType      = BasicTypes[OctKind]
        IntType      = BasicTypes[IntKind]
        HexType      = BasicTypes[HexKind]
        FloatType    = BasicTypes[FloatKind]
        DateTimeType = BasicTypes[DateTimeKind]
        DateType     = BasicTypes[DateKind]
        TimeType     = BasicTypes[TimeKind]
        UriType      = BasicTypes[UriKind]
        StringType   = BasicTypes[StringKind]
        BarewordType = BasicTypes[BarewordKind]
        BarefileType = BasicTypes[BarefileKind]
        GlobType     = BasicTypes[GlobKind]
        PathSegType  = BasicTypes[PathSegKind]
        PathType     = BasicTypes[PathKind]
        FileType     = BasicTypes[FileKind]
        FlagType     = BasicTypes[FlagKind]
        NoneType     = BasicTypes[NoneKind]

        // Shortcuts for composite types.
        CompoundType = CompositeTypes[CompoundKind]
        BarecompType = CompositeTypes[BarecompKind]
        ListType     = CompositeTypes[ListKind]
        GroupType    = CompositeTypes[GroupKind]
        MapType      = CompositeTypes[MapKind]
        PairType     = CompositeTypes[PairKind]
        PatternType  = CompositeTypes[PatternKind]
        DelegateType = CompositeTypes[DelegateKind]
        ClosureType  = CompositeTypes[ClosureKind]
)

func defUniverseBuiltins() {
        for name, f := range builtins {
                if _, alt := universe.InsertBuiltin(name, f); alt != nil {
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
        _, _ = universe.InsertDef(nil, "SMART.BIN", bin)
        _, _ = universe.InsertDef(nil, "SMART.ARGS", args)
        _, _ = universe.InsertDef(nil, "SMART", bin)
        
        defUniverseBuiltins()
}

// IsUniverse checks if the scope is universe.
func IsUniverse(scope *Scope) bool {
        return scope == universe
}

// A Globe represents a global execution context in the Universe. 
type Globe struct {
        scope  *Scope
	unsafe *Project
        main   *Project
}

// Scope returns the globe scope.
func (g *Globe) Scope() *Scope { return g.scope }

// Main returns the main project.
func (g *Globe) Main() *Project { return g.main }

// SetMain changes the main project.
/* func (g *Globe) SetMain(m *Project) {
        g.main = m 
} */

func (g *Globe) SetScopeOuter(scope *Scope) {
        scope.outer = g.scope
}

// NewProject returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *Globe) NewProject(outer *Scope, absPath, relPath, spec, name string) (m *Project) {
        if outer == nil {
                outer = g.scope
        }
        
	m = &Project{
                absPath: absPath,
                relPath: relPath, 
                spec: spec,
                name: name,
        }
	
        m.scope = NewScope(outer, m, fmt.Sprintf("project %q", name))

        if name != "@" && g.main == nil {
                for outer != nil && outer != g.scope {
                        if p := outer.Project(); p != nil && p.Name() == "@" {
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
        }
}
