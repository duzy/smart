//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "github.com/duzy/smart/token"
        "fmt"
)

// Object is a value defined in a scope.
//
// TODO: defines ObjInfo to classify objects.
// 
type Object interface {
        Value

        Pos() *token.Position // position or nil

        Parent() *Scope
        Project() *Project
        Name() string
        
	// order reflects a package-level object's source order: if object
	// a is before object b in the source, then a.order() < b.order().
	// order returns a value > 0 for package-level objects; it returns
	// 0 for all other objects (including objects in file scopes).
	order() uint32

	// setParent sets the parent scope of the object.
	setParent(*Scope)

	// scopePos returns the start position of the scope of this Object
	scopePos() token.Pos

	// setScopePos sets the start position of the scope for this Object.
	setScopePos(pos token.Pos) // FIXME: it's not applied
}

// An object implements the common parts of an Object.
type object struct {
        value
        parent *Scope
        project *Project
        name string
        typ Type
        ord uint32
        scopos token.Pos
}

func (obj *object) Parent() *Scope        { return obj.parent }
func (obj *object) Project() *Project     { return obj.project }
func (obj *object) Name() string          { return obj.name }
func (obj *object) Pos() *token.Position  { return nil /*obj.scopos*/ }

func (obj *object) Type() Type            { return obj.typ }
func (obj *object) String() string        { return obj.Lit() }
func (obj *object) Lit() string           { return fmt.Sprintf("object %v", obj.name) }

func (obj *object) order() uint32         { return obj.ord }
func (obj *object) scopePos() token.Pos   { return obj.scopos }

func (obj *object) setParent(parent *Scope)   { obj.parent = parent }
func (obj *object) setOrder(order uint32)     { /*assert(order > 0);*/ obj.ord = order }
func (obj *object) setScopePos(pos token.Pos) { obj.scopos = pos }

func (scope *Scope) NewDummy(project *Project, name string) Object {
	return &object{
                parent:  scope,
                project: project,
                name:    name,
                typ:     InvalidType,
                ord:     0,
                scopos:  token.NoPos,
        }
}

func (scope *Scope) InsertDummy(project *Project, name string) (obj, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                obj = scope.NewDummy(project, name)
                scope.replace(name, obj)
        }
        return
}

func IsDummy(s interface{}) bool {
        _, ok := s.(*object)
        return ok
}

type ProjectName struct {
        object
        project *Project
}

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (n *ProjectName) Project() *Project { return n.project }
func (n *ProjectName) String() string  {
        return fmt.Sprintf("project %s %p", n.name, n.project)
}

func (scope *Scope) NewProjectName(container *Project, name string, project *Project) *ProjectName {
	return &ProjectName{
                object{
                        parent:  scope,
                        project: container,
                        name:    name,
                        typ:     ProjectNameType,
                        ord:     0,
                        scopos:  token.NoPos,
                },
                project,
        }
}

func (scope *Scope) InsertProjectName(container *Project, name string, project *Project) (pn *ProjectName, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                pn = scope.NewProjectName(container, name, project)
                scope.replace(name, pn)
        }
        return
}

type ScopeName struct {
        object
        scope *Scope
}

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (n *ScopeName) Scope() *Scope { return n.scope }
func (n *ScopeName) String() string  {
        return fmt.Sprintf("scope %s %p", n.name, n.project)
}

func (scope *Scope) NewScopeName(project *Project, name string, s *Scope) *ScopeName {
	return &ScopeName{
                object{
                        parent:  scope,
                        project: project,
                        name:    name,
                        typ:     InvalidType, //ScopeNameType,
                        ord:     0,
                        scopos:  token.NoPos,
                },
                s,
        }
}

func (scope *Scope) InsertScopeName(project *Project, name string, s *Scope) (pn *ScopeName, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                pn = scope.NewScopeName(project, name, s)
                scope.replace(name, pn)
        }
        return
}

// A Def represents a definition.
type Def struct {
        object
        value Value
}

func (d *Def) Value() Value    { return d.value }
func (d *Def) String() string  { return d.name+" = "+d.value.String() }
func (d *Def) Set(v Value)     { d.value = v }
func (d *Def) Call(a... Value) (Value, error) {
        // TODO: parameterization, e.g. $1, $2, $3, $4, $5
        return d.value, nil 
}

func (scope *Scope) NewDef(project *Project, name string, value Value) *Def {
	return &Def{
                object{
                        parent:  scope,
                        project: project,
                        name:    name,
                        typ:     DefineType,
                        ord:     0,
                        scopos:  token.NoPos,
                },
                value,
        }
}

func (scope *Scope) InsertDef(project *Project, name string, value Value) (def *Def, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                def = scope.NewDef(project, name, value)
                scope.replace(name, def)
        } else if d,b := alt.(*Def); d != nil && b {
                //d.Set(value)
        }
        return
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        object
        f BuiltinFunc
}

func (p *Builtin) String() string { return fmt.Sprintf("builtin %v", p.name) }
func (p *Builtin) Call(context *Scope, a... Value) (Value, error) {
        return p.f(context, a...) 
}

func (scope *Scope) NewBuiltin(name string, f BuiltinFunc) *Builtin {
        return &Builtin{
                object{
                        parent:  scope,
                        project: nil,
                        name:    name, 
                        typ:     BuiltinType,
                        ord:     0,
                        scopos:  token.NoPos,
                },
                f,
        }
}

func (scope *Scope) InsertBuiltin(name string, f BuiltinFunc) (bui *Builtin, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                bui = scope.NewBuiltin(name, f)
                scope.replace(name, bui)
        }
        return
}

type RuleEntryClass int

const (
        GeneralRuleEntry RuleEntryClass = 1<<iota
        FileRuleEntry
        PatternRuleEntry
        PatternFileRuleEntry
        UseRuleEntry
)

var ruleEntryClassNames = []string{
        GeneralRuleEntry:     "GeneralRuleEntry",
        FileRuleEntry:        "FileRuleEntry",
        PatternRuleEntry:     "PatternRuleEntry",
        PatternFileRuleEntry: "PatternFileRuleEntry",
        UseRuleEntry:         "UseRuleEntry",
}

func (c RuleEntryClass) String() string {
        var i = int(c)
        if 0 < i && i < len(ruleEntryClassNames) {
                return ruleEntryClassNames[i]
        }
        return fmt.Sprintf("RuleEntryClass(%d)", i)
}

// RuleEntry represents a declared rule entry.
type RuleEntry struct {
        object
        class RuleEntryClass
        program Program
        stem string // only applied for PatternRuleEntry
}

func (entry *RuleEntry) String() string { return entry.name }
func (entry *RuleEntry) Stem() string { return entry.stem }

func (entry *RuleEntry) Class() RuleEntryClass { return entry.class }
func (entry *RuleEntry) SetClass(class RuleEntryClass) { entry.class = class }

// RuleEntry.Program returns the rule program.
func (entry *RuleEntry) Program() Program { return entry.program }

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
func (entry *RuleEntry) Call(context *Scope, a... Value) (result Value, err error) {
        if entry.program != nil {
                result, err = entry.program.Execute(context, entry, a, false)
        }
        return
}

type ArgumentedEntry struct {
        *RuleEntry
        Args []Value
}

func (scope *Scope) NewRuleEntry(project *Project, kind RuleEntryClass, name string) (entry *RuleEntry) {
        return &RuleEntry{
                object{
                        parent:  scope,
                        project: project,
                        name:    name,
                        typ:     RuleEntryType,
                        ord:     0,
                        scopos:  token.NoPos,
                },
                kind, nil, "",
        }
}

func (scope *Scope) InsertEntry(project *Project, kind RuleEntryClass, name string) (entry *RuleEntry, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                entry = scope.NewRuleEntry(project, kind, name)
                scope.replace(name, entry)
        }
        return
}
