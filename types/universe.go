//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

// This file sets up the global scope and the root project/module.

package types

import (
        "github.com/duzy/smart/token"
        "fmt"
)

var (
	universe *Scope
	//unsafe   *Module

        UniversalNone = &None{}
)

// Predeclared types.
var (
        CoreTypes = []*Core {
                DefKind:         {DefKind, IsDef, "Def"},
                ProjectNameKind: {ProjectNameKind, IsBuiltin, "ProjectName"},
                BuiltinKind:     {BuiltinKind, IsRuleEntry, "Builtin"},
                RuleEntryKind:   {RuleEntryKind, IsProjectName, "RuleEntry"},
        }
        
        BasicTypes = []*Basic {
                InvalidKind:  {InvalidKind, 0, "invalid"},
                AnyKind:      {AnyKind, IsAny, "any"},
                IntKind:      {IntKind, IsInteger, "int"},
                FloatKind:    {FloatKind, IsFloat, "float"},
                DateTimeKind: {DateTimeKind, IsDateTime, "datetime"},
                DateKind:     {DateKind, IsDate, "date"},
                TimeKind:     {TimeKind, IsTime, "time"},
                UriKind:      {UriKind, IsUri, "uri"},
                StringKind:   {StringKind, IsString, "string"},
                BarewordKind: {BarewordKind, IsBareword, "bareword"},
                BarefileKind: {BarefileKind, IsBarefile, "barefile"},
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
        BuiltinType     = CoreTypes[BuiltinKind]
        RuleEntryType   = CoreTypes[RuleEntryKind]
        ProjectNameType = CoreTypes[ProjectNameKind]

        // Shortcuts of basic types.
        InvalidType  = BasicTypes[InvalidKind]
        AnyType      = BasicTypes[AnyKind]
        IntType      = BasicTypes[IntKind]
        FloatType    = BasicTypes[FloatKind]
        DateTimeType = BasicTypes[DateTimeKind]
        DateType     = BasicTypes[DateKind]
        TimeType     = BasicTypes[TimeKind]
        UriType      = BasicTypes[UriKind]
        StringType   = BasicTypes[StringKind]
        BarewordType = BasicTypes[BarewordKind]
        BarefileType = BasicTypes[BarefileKind]
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
        universe = NewScope(nil, token.NoPos, token.NoPos, "universe")
        //unsafe = NewModule(token.ILLEGAL, "unsafe", "unsafe")
        //unsafe.complete = true

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

// NewProject returns a new Project for the given project path and name;
// the name must not be the blank identifier.
// The project is not complete and contains no explicit imports.
func (g *Globe) NewProject(outer *Scope, absPath, relPath, spec, name string) (m *Project) {
        if outer == nil {
                outer = g.scope
        }
        
	scope := NewScope(outer, token.NoPos, token.NoPos, fmt.Sprintf("project %q", name/*specPath*/))
	m = &Project{
                absPath: absPath,
                relPath: relPath, 
                spec: spec,
                name: name,
                scope: scope,
        }
        if name != "@" && g.main == nil {
                g.main = m
        }
        return
}

// NewGlobe creates a new Globe context.
func NewGlobe(name string) *Globe {
        scope := NewScope(universe, token.NoPos, token.NoPos, fmt.Sprintf("globe %q", name))
        return &Globe{
                scope: scope,
                main: nil,
        }
}
