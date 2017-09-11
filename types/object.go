//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "os/exec"
        "strings"
        "errors"
        "bytes"
        "fmt"
)

// Object is a value defined in a scope.
//
// TODO: defines ObjInfo to classify objects.
// 
type Object interface {
        Value

        Parent() *Scope
        Project() *Project
        Name() string

        // Get object's named property.
        Get(name string) (Value, error)
        
	// order reflects a package-level object's source order: if object
	// a is before object b in the source, then a.order() < b.order().
	// order returns a value > 0 for package-level objects; it returns
	// 0 for all other objects (including objects in file scopes).
	order() uint32

	// setParent sets the parent scope of the object.
	setParent(*Scope)
}

// An object implements the common parts of an Object.
type object struct {
        value
        parent *Scope
        project *Project
        name string
        typ Type
        ord uint32
}

func (obj *object) Parent() *Scope        { return obj.parent }
func (obj *object) Project() *Project     { return obj.project }
func (obj *object) Name() string          { return obj.name }

func (obj *object) Type() Type            { return obj.typ }
func (obj *object) Strval() string        { return obj.String() }
func (obj *object) String() string        { return fmt.Sprintf("object %v", obj.name) }

func (obj *object) Get(name string) (Value, error) {
        return nil, errors.New(fmt.Sprintf("No such property `%s' (Object).", name))
}

func (obj *object) order() uint32         { return obj.ord }

func (obj *object) setParent(parent *Scope)   { obj.parent = parent }
func (obj *object) setOrder(order uint32)     { /*assert(order > 0);*/ obj.ord = order }

type ProjectName struct {
        object
        project *Project
}

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (n *ProjectName) Type() Type { return ProjectNameType }
func (n *ProjectName) Project() *Project { return n.project }
func (n *ProjectName) String() string  {
        return fmt.Sprintf("project %s %p", n.name, n.project)
}
func (n *ProjectName) Strval() string  {
        return fmt.Sprintf("project %s %p", n.name, n.project)
}

func (n *ProjectName) Get(name string) (Value, error) {
        if scope, sym := n.project.scope.Find(name); scope != nil && sym != nil {
                if false {
                        if o, _ := sym.(Object); o != nil && o.Project() != n.project {
                                //fmt.Printf("diverged: %v (%v != %v)\n", name, o.Project().Name(), n.project.Name())
                                //fmt.Printf("%v\n", n.project.scope)
                                //fmt.Printf("%v\n", n.project.scope.chain)
                                //fmt.Printf("%v\n", scope)
                                return nil, errors.New(fmt.Sprintf("Symbol diverged `%s'", name))
                        } else if value, _ := sym.(Value); value != nil {
                                return value, nil
                        } else {
                                return nil, errors.New(fmt.Sprintf("Symbol `%s' is not value (%T)", name, sym))
                        } 
                } else {
                        value, _ := sym.(Value); return value, nil
                }
        }
        return nil, errors.New(fmt.Sprintf("Undefined `%s' in project `%s'.", name, n.project.Name()))
}

func (scope *Scope) InsertProjectName(container *Project, name string, project *Project) (pn *ProjectName, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                pn = &ProjectName{
                        object{
                                parent:  scope,
                                project: container,
                                name:    name,
                                typ:     ProjectNameType,
                                ord:     0,
                        },
                        project,
                }
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
func (n *ScopeName) Type() Type { return ScopeNameType }
func (n *ScopeName) Scope() *Scope { return n.scope }
func (n *ScopeName) String() string  {
        return fmt.Sprintf("scope %s %p", n.name, n.scope)
}
func (n *ScopeName) Strval() string  {
        return fmt.Sprintf("scope %s %p", n.name, n.scope)
}

func (n *ScopeName) Get(name string) (Value, error) {
        if sym := n.scope.Resolve(name); sym != nil {
                value, _ := sym.(Value)
                return value, nil
        }
        return nil, errors.New(fmt.Sprintf("Undefined `%s' in scope `%s'.", name, n.Name()))
}

func (scope *Scope) InsertScopeName(project *Project, name string, s *Scope) (sn *ScopeName, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                sn = &ScopeName{
                        object{
                                parent:  scope,
                                project: project,
                                name:    name,
                                typ:     ScopeNameType,
                                ord:     0,
                        },
                        s,
                }
                scope.replace(name, sn)
        }
        return
}

type DefOrigin int
const (
        // Never assigned a value
        TrivialDef DefOrigin = iota

        // =, !=
        DefaultDef
        
        // :=, ::=
        ImmediateDef

        // Immediate Def without closure.
        // DisclosureDef
)

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type Def struct {
        object
        origin DefOrigin
        Value Value
}

func (d *Def) disclose(scope *Scope) (Value, error) {
        if v, e := d.Value.disclose(scope); e != nil {
                return nil, e
        } else if v != nil {
                return &Def{ d.object, d.origin, v }, nil
        }
        return nil, nil
}

func (d *Def) referencing(o Object) bool {
        return d.Value.referencing(o)
}

func (d *Def) String() string {
        s := "define " + d.name
        if d.origin == ImmediateDef {
                s += " := "
        } else {
                s += " = "
        }
        if d.Value == nil {
                s += "<nil>"
        } else {
                s += d.Value.String()
        }
        return s
}
func (d *Def) Strval() string {
        s := d.name + "="
        if d.Value == nil {
                s += "<nil>"
        } else {
                s += d.Value.Strval()
        }
        return s
}
func (d *Def) Origin() DefOrigin { return d.origin }
func (d *Def) SetOrigin(k DefOrigin) { d.origin = k }

func (d *Def) Assign(v Value) (Value, error) {
        if v == nil {
                v = UniversalNone
        } else if v.referencing(d) {
                err := errors.New(fmt.Sprintf("Recursive variable `%s' references itself.", d.name))
                return nil, err
        }
        
        switch d.origin {
        case TrivialDef, DefaultDef:
                d.Value = v // Keeps delegates and closures.
        case ImmediateDef:
                // Eval expends delegates in the value.
                d.Value = Eval(v)
        }
        return d.Value, nil
}

func (d *Def) Append(va... Value) (Value, error) {
        var (
                nva = len(va)
                nv Value // new value
        )
        if nva == 0 {
                // Does nothing...
        } else if d.Value != nil && d.Value.Type() != NoneType {
                nv = d.Value
                if l, ok := nv.(*List); ok && l != nil {
                        l.Append(Join(va...)...)
                } else if nva > 0 {
                        elems := []Value{ nv }
                        elems = append(elems, Join(va...)...)
                        nv = &List{ Elements{ elems } }
                }
        } else if nva > 0 {
                nv = &List{ Elements{ Join(va...) } }
        }
        if nv != nil {
                return d.Assign(nv)
        }
        return d.Value, nil
}

func (d *Def) AssignExec(a... Value) (Value, error) {
        var (
                stdout bytes.Buffer
                stderr bytes.Buffer
        )
        for _, v := range a {
                sh := exec.Command("sh", "-c", v.Strval())
                sh.Stdout, sh.Stderr = &stdout, &stderr
                if err := sh.Run(); err != nil {
                        v, _ = d.Assign(UniversalNone)
                        return v, err
                }
        }
        return d.Assign(&String{strings.TrimSpace(stdout.String())})
}

func (d *Def) Call(a... Value) (Value, error) {
        // TODO: parameterization, e.g. $1, $2, $3, $4, $5
        //return Eval(d.Value), nil
        return d.Value, nil
}

func (d *Def) Get(name string) (Value, error) {
        switch name {
        case "name": return &String{d.name}, nil
        case "value": return d.Value, nil
        }
        //fmt.Printf("%v %v\n", d.name, d.parent)
        return nil, errors.New(fmt.Sprintf("No such property `%s' (Def)", name))
}

func (scope *Scope) InsertDef(project *Project, name string, value Value) (def *Def, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                def = &Def{
                        object{
                                parent:  scope,
                                project: project,
                                name:    name,
                                typ:     DefType,
                                ord:     0,
                        },
                        TrivialDef, value,
                }
                scope.replace(name, def)
        } else if name == "use" {
                if sn, ok := alt.(*ScopeName); ok && sn != nil {
                        def, alt = sn.Scope().InsertDef(project, "=", value)
                }
        }
        return
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        object
        f BuiltinFunc
}

func (p *Builtin) Strval() string { return fmt.Sprintf("builtin %v", p.name) }
func (p *Builtin) Call(a... Value) (Value, error) {
        return p.f(p.parent, a...)
}

func (scope *Scope) InsertBuiltin(name string, f BuiltinFunc) (bui *Builtin, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                bui = &Builtin{
                        object{
                                parent:  scope,
                                project: nil,
                                name:    name, 
                                typ:     BuiltinType,
                                ord:     0,
                        },
                        f,
                }
                scope.replace(name, bui)
        }
        return
}

type RuleEntryClass int

const (
        GeneralRuleEntry RuleEntryClass = iota
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
        programs []Program
        stem string // only applied for PatternRuleEntry
        Creator *PatternEntry
}

func (entry *RuleEntry) String() string { return fmt.Sprintf("entry %v", entry.name) }
func (entry *RuleEntry) Strval() string { return entry.name }
func (entry *RuleEntry) Stem() string { return entry.stem }

func (entry *RuleEntry) Class() RuleEntryClass { return entry.class }
func (entry *RuleEntry) SetClass(class RuleEntryClass) { entry.class = class }

func (entry *RuleEntry) IsPattern() bool {
        return entry.class == PatternRuleEntry || entry.class == PatternFileRuleEntry;
}

func (entry *RuleEntry) IsFile() bool {
        return entry.class == FileRuleEntry || entry.class == PatternFileRuleEntry;
}

/*func (entry *RuleEntry) HasRecipes() bool {
        for _, prog := range entry.programs {
                if len(prog.recipes) > 0 {
                        return true
                }
        }
        return false
}*/

// RuleEntry.Program returns the rule program.
func (entry *RuleEntry) Programs() []Program { return entry.programs }

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
func (entry *RuleEntry) Execute(context *Scope, a... Value) (result []Value, err error) {
        if entry.IsPattern() {
                return nil, errors.New(fmt.Sprintf("Calling pattern entry `%s'.", entry.Name()))
        }
        if context == nil {
                context = entry.Project().Scope()
        }
        for _, program := range entry.programs {
                if v, e := program.Execute(context, entry, a); e != nil {
                        //fmt.Printf("failed: %v: %v\n", entry.Name(), e)
                        err = e; return
                } else {
                        result = append(result, v)
                }
        }
        return
}

func (entry *RuleEntry) Get(name string) (Value, error) {
        switch name {
        case "name": return &String{entry.Name()}, nil
        case "class": return &String{entry.class.String()}, nil
        //case "stem": return &String{entry.stem}, nil
        }
        return nil, errors.New(fmt.Sprintf("no such entry property (%s)", name))
}

type PatternEntry struct {
        *RuleEntry
        Pattern Pattern
}

func (p *PatternEntry) MakeConcreteEntry(stem string) (entry *RuleEntry, err error) {
        if entry, err = p.Pattern.MakeConcreteEntry(p.RuleEntry, stem); err == nil && entry != nil {
                entry.Creator = p
        }
        return
}

func (scope *Scope) InsertEntry(project *Project, kind RuleEntryClass, name string) (entry *RuleEntry, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                entry = &RuleEntry{
                        object{
                                parent:  scope,
                                project: project,
                                name:    name,
                                typ:     RuleEntryType,
                                ord:     0,
                        },
                        kind, nil, "", nil,
                }
                scope.replace(name, entry)
        } else if name == "use" {
                if sn, ok := alt.(*ScopeName); ok && sn != nil {
                        entry, alt = sn.Scope().InsertEntry(project, UseRuleEntry, ":")
                }
        }
        return
}
