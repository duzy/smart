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
        Parent() *Scope
        Project() *Project
        Name() string
        Type() Type // the type of the object, differs from the value type
        Pos() *token.Position // position or nil

        Lit() string
        String() string
        Integer() int64
        Float() float64

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
        parent *Scope
        project *Project
        name string
        typ Type
        ord uint32
        scopos token.Pos
}

func (obj *object) Parent() *Scope        { return obj.parent }
func (obj *object) Project() *Project     { return obj.project }
func (obj *object) Pos() *token.Position  { return nil /*obj.pos*/ }
func (obj *object) Name() string          { return obj.name }
func (obj *object) Type() Type            { return obj.typ }
func (obj *object) Lit() string           { return "" }
func (obj *object) String() string        { return fmt.Sprintf("object %v", obj.name) }
func (obj *object) Integer() int64        { return 0 }
func (obj *object) Float() float64        { return 0 }
//func (obj *object) Call(a... Value) (Value, error) { return nil, nil }
func (obj *object) order() uint32         { return obj.ord }
func (obj *object) scopePos() token.Pos   { return obj.scopos }

func (obj *object) setParent(parent *Scope)   { obj.parent = parent }
func (obj *object) setOrder(order uint32)     { /*assert(order > 0);*/ obj.ord = order }
func (obj *object) setScopePos(pos token.Pos) { obj.scopos = pos }

func NewDummy(mod *Project, scope *Scope, name string) Object {
	return &object{scope, mod, name, Invalid, 0, token.NoPos}
}

func IsDummy(s interface{}) bool {
        _, ok := s.(*object)
        return ok
}

func IsDummyObject(s Object) bool {
        _, ok := s.(*object)
        return ok
}

func IsDummyValue(s Value) bool {
        _, ok := s.(*object)
        return ok
}

type ProjectName struct {
        object
        imported *Project
        used bool // set if the project was used
}

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (n *ProjectName) Imported() *Project { return n.imported }
func (n *ProjectName) String() string  {
        //return fmt.Sprintf("%s[%p]", n.name, n.imported)
        return fmt.Sprintf("project %s", n.name)
}

func NewProjectName(mod *Project, name string, imported *Project) *ProjectName {
	return &ProjectName{object{nil, mod, name, ProjectNameType, 0, token.NoPos}, imported, false}
}

// A Const represents a declared constant.
/* type Const struct {
        object
} */

// A Def represents a definition.
type Def struct {
        object
        value Value
}

func (d *Def) Value() Value    { return d.value }
func (d *Def) String() string  { return d.name+" = "+d.value.String() }
func (d *Def) Set(v Value)     { d.value = v }
func (d *Def) Call(a... Value) (Value, error) { return d.value, nil }

func NewDef(mod *Project, name string, value Value) *Def {
	return &Def{object{nil, mod, name, DefineType, 0, token.NoPos}, value}
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        object
        f BuiltinFunc
}

func (p *Builtin) Call(a... Value) (Value, error) { return p.f(a...) }
func (p *Builtin) String() string { return fmt.Sprintf("builtin %v", p.name) }

func NewBuiltin(name string, f BuiltinFunc) *Builtin {
        return &Builtin{object{
                parent: nil, 
                project: nil, 
                name: name, 
                typ: BuiltinType,
                ord: 0,
                scopos: token.NoPos,
        }, f}
}

type RuleEntryClass int

const (
        GeneralRuleEntry RuleEntryClass = 1<<iota
        FileRuleEntry
        PatternRuleEntry
        PatternFileRuleEntry
)

var ruleEntryClassNames = []string{
        GeneralRuleEntry:     "GeneralRuleEntry",
        FileRuleEntry:        "FileRuleEntry",
        PatternRuleEntry:     "PatternRuleEntry",
        PatternFileRuleEntry: "PatternFileRuleEntry",
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
        kind RuleEntryClass
        program Program
        stem string // only applied for PatternRuleEntry
}

func (entry *RuleEntry) String() string { return entry.name }
func (entry *RuleEntry) Stem() string { return entry.stem }

func (entry *RuleEntry) Kind() RuleEntryClass { return entry.kind }

// RuleEntry.Program returns the rule program.
func (entry *RuleEntry) Program() Program { return entry.program }

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
//
// TODO: merge Execute and Call, make RuleEntry behaves like a Def
// 
func (entry *RuleEntry) Call(a... Value) (result Value, err error) {
        if entry.program != nil {
                result, err = entry.program.Execute(entry, a, false)
        }
        return
}

func NewRuleEntry(kind RuleEntryClass, name string) (entry *RuleEntry) {
        return &RuleEntry{
                object{
                        nil, nil, name, RuleEntryType, 
                        0, token.NoPos,
                },
                kind, nil, 
                "",
        }
}
