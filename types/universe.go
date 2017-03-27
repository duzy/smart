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
                DefineKind:     {DefineKind, IsDefine, "Define"},
                ModuleNameKind: {ModuleNameKind, IsBuiltin, "ModuleName"},
                BuiltinKind:    {BuiltinKind, IsRuleEntry, "Builtin"},
                RuleEntryKind:  {RuleEntryKind, IsModuleName, "RuleEntry"},
        }
        
        BasicTypes = []*Basic {
                InvalidKind:  {InvalidKind, 0, "invalid"},
                IntKind:      {IntKind, IsInteger, "int"},
                FloatKind:    {FloatKind, IsFloat, "float"},
                DateTimeKind: {DateTimeKind, IsDateTime, "datetime"},
                DateKind:     {DateKind, IsDate, "date"},
                TimeKind:     {TimeKind, IsTime, "time"},
                UriKind:      {UriKind, IsUri, "uri"},
                StringKind:   {StringKind, IsString, "string"},
                BarewordKind: {BarewordKind, IsBareword, "bareword"},
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
        DefineType     = CoreTypes[DefineKind]
        BuiltinType    = CoreTypes[BuiltinKind]
        RuleEntryType  = CoreTypes[RuleEntryKind]
        ModuleNameType = CoreTypes[ModuleNameKind]

        // Shortcuts of basic types.
        Invalid  = BasicTypes[InvalidKind]
        Int      = BasicTypes[IntKind]
        Float    = BasicTypes[FloatKind]
        DateTime = BasicTypes[DateTimeKind]
        Date     = BasicTypes[DateKind]
        Time     = BasicTypes[TimeKind]
        Uri      = BasicTypes[UriKind]
        String   = BasicTypes[StringKind]
        Bareword = BasicTypes[BarewordKind]
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
	unsafe *Module
        main   *Module
}

// Scope returns the globe scope.
func (g *Globe) Scope() *Scope { return g.scope }

// Main returns the main module.
func (g *Globe) Main() *Module { return g.main }

// SetMain changes the main module.
/* func (g *Globe) SetMain(m *Module) {
        g.main = m 
} */

// NewModule returns a new Module for the given module path and name;
// the name must not be the blank identifier.
// The module is not complete and contains no explicit imports.
func (g *Globe) NewModule(kw token.Token, path, name string) (m *Module) {
	scope := NewScope(g.scope, token.NoPos, token.NoPos, fmt.Sprintf("module %q", path))
	m = &Module{
                keyword: kw, 
                path: path, 
                name: name, 
                scope: scope,
                //entries: make(map[string]*RuleEntry),
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
