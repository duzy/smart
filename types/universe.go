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
)

// Predeclared types.
var (
        CoreTypes = []*Core {
                DefineKind:      {DefineKind, IsDefine, "Define"},
                ProjectNameKind: {ProjectNameKind, IsBuiltin, "ProjectName"},
                BuiltinKind:     {BuiltinKind, IsRuleEntry, "Builtin"},
                RuleEntryKind:   {RuleEntryKind, IsProjectName, "RuleEntry"},
        }
        
        BasicTypes = []*Basic {
                InvalidKind:  {InvalidKind, 0, "invalid"},
                IdentKind:    {IdentKind, IsIdent, "ident"},
                IntKind:      {IntKind, IsInteger, "int"},
                FloatKind:    {FloatKind, IsFloat, "float"},
                DateTimeKind: {DateTimeKind, IsDateTime, "datetime"},
                DateKind:     {DateKind, IsDate, "date"},
                TimeKind:     {TimeKind, IsTime, "time"},
                UriKind:      {UriKind, IsUri, "uri"},
                StringKind:   {StringKind, IsString, "string"},
                BarewordKind: {BarewordKind, IsBareword, "bareword"},
                BarefileKind: {BarefileKind, IsBarefile, "barefile"},
                PathKind:     {PathKind, IsBarefile, "path"},
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
        }

        // Shortcuts of core types
        DefineType      = CoreTypes[DefineKind]
        BuiltinType     = CoreTypes[BuiltinKind]
        RuleEntryType   = CoreTypes[RuleEntryKind]
        ProjectNameType = CoreTypes[ProjectNameKind]

        // Shortcuts of basic types.
        Invalid  = BasicTypes[InvalidKind]
        Ident    = BasicTypes[IdentKind]
        Int      = BasicTypes[IntKind]
        Float    = BasicTypes[FloatKind]
        DateTime = BasicTypes[DateTimeKind]
        Date     = BasicTypes[DateKind]
        Time     = BasicTypes[TimeKind]
        Uri      = BasicTypes[UriKind]
        String   = BasicTypes[StringKind]
        Bareword = BasicTypes[BarewordKind]
        Barefile = BasicTypes[BarefileKind]
        Path     = BasicTypes[PathKind]
        Flag     = BasicTypes[FlagKind]
        None     = BasicTypes[NoneKind]

        // Shortcuts for composite types.
        Compound = CompositeTypes[CompoundKind]
        Barecomp = CompositeTypes[BarecompKind]
        List     = CompositeTypes[ListKind]
        Group    = CompositeTypes[GroupKind]
        Map      = CompositeTypes[MapKind]
        Pair     = CompositeTypes[PairKind]
)

func defUniverseBuiltins() {
        for name, f := range builtins {
                universe.Insert(NewBuiltin(name, f))
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
func (g *Globe) NewProject(absPath, specPath, name string) (m *Project) {
	scope := NewScope(g.scope, token.NoPos, token.NoPos, fmt.Sprintf("project %q", name/*specPath*/))
	m = &Project{
                absPath: absPath,
                specPath: specPath, 
                name: name, 
                scope: scope,
        }
        if g.main == nil {
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
