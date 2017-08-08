//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        "github.com/duzy/smart/token"
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

        Pos() *token.Position // position or nil

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
func (obj *object) Strval() string        { return obj.String() }
func (obj *object) String() string        { return fmt.Sprintf("object %v", obj.name) }

func (obj *object) Get(name string) (Value, error) {
        return nil, errors.New(fmt.Sprintf("No such property `%s' (Object).", name))
}

func (obj *object) order() uint32         { return obj.ord }
func (obj *object) scopePos() token.Pos   { return obj.scopos }

func (obj *object) setParent(parent *Scope)   { obj.parent = parent }
func (obj *object) setOrder(order uint32)     { /*assert(order > 0);*/ obj.ord = order }
func (obj *object) setScopePos(pos token.Pos) { obj.scopos = pos }

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
        if sym := n.project.Scope().Resolve(name); sym != nil {
                value, _ := sym.(Value)
                return value, nil
        }
        return nil, errors.New(fmt.Sprintf("undefined %s in project %s", name, n.project.Name()))
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
        return nil, errors.New(fmt.Sprintf("undefined %s in scope %s", name, n.Name()))
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
        s := d.name + "="
        if d.Value == nil {
                s += "<nil>"
        } else {
                s += d.Value.String()
        }
        return s
}
func (d *Def) Strval() string {
        fmt.Printf("Def.Strval: %p %p\n", d, d.Value)
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

// TODO: move it into 'runtime' package
type definer struct {
        name string
        op token.Token
}
func (p *definer) disclose(_ *Scope) (Value, error) { return nil, nil }
func (p *definer) referencing(_ Object) bool { return false }
func (p *definer) Type() Type     { return DefinerType }
func (p *definer) Name() string   { return p.name }
func (p *definer) String() string { return "definer " + p.name }
func (p *definer) Strval() string { return "definer " + p.name }
func (p *definer) Integer() int64 { return 0 }
func (p *definer) Float() float64 { return 0 }
func (p *definer) Define(scope *Scope, project *Project) (def *Def, err error) {
        fmt.Printf("define: %v in %v\n", p.name, scope)
        var src *Def
        if o := scope.Lookup(p.name); o == nil {
                err = errors.New(fmt.Sprintf("%s undefined in source scope", p.name))
                return
        } else if src, _ = o.(*Def); src == nil {
                err = errors.New(fmt.Sprintf("%s in source scope is not Def", p.name))
                return
        }
        
        var isAltDef = false
        if obj, alt := project.Scope().InsertDef(project, p.name, UniversalNone); alt != nil {
                def, isAltDef = alt.(*Def), true
        } else {
                def = obj
        }
        switch p.op {
        default:               _, err = def.Assign(src.Value)
        case token.EXC_ASSIGN: _, err = def.AssignExec(src.Value)
        case token.ADD_ASSIGN:
                var value = src.Value
                // Inorder to avoid recursive referencing, assignment have to
                // check if the heading elements are the same or referencing to
                // each other. This restrition only to rule-program definers and
                // required when importing (and using) multiple projects and they're
                // appending to them same symbol name.
                if isAltDef && src.Value.Type() == ListType && def.Value.Type() == ListType {
                        srcList, dstList := src.Value.(*List), def.Value.(*List)
                        if len(srcList.Elems) > 0 && len(dstList.Elems) > 0 {
                                srcDel0, _ := srcList.Elems[0].(*delegate)
                                dstDel0, _ := dstList.Elems[0].(*delegate)
                                if srcDel0 == dstDel0 || srcDel0.o == dstDel0.o {
                                        fmt.Printf("append--: %p %p, %v %v, %v %v\n", srcDel0.o, dstDel0.o, 
                                                srcDel0.o.referencing(dstDel0.o),
                                                dstDel0.o.referencing(srcDel0.o),
                                                srcDel0.referencing(dstDel0.o),
                                                dstDel0.referencing(srcDel0.o))
                                } else if srcDel0.o.Name() == dstDel0.o.Name() {
                                        srcDef := srcDel0.o.(*Def)
                                        dstDef := dstDel0.o.(*Def)
                                        /*if srcDef == nil || dstDef == nil {
                                                // FIXME: unreachable...
                                        } else if srcDel0.o.referencing(dstDel0.o) {
                                                var d *delegate
                                                switch t := srcDef.Value.(type) {
                                                case *List: d, _ = t.Elems[0].(*delegate)
                                                case *delegate: d = t
                                                }
                                                if d != nil && d.o == dstDef {
                                                        //fmt.Printf("append: %p %p\n", d.o, dstDef)
                                                        value = &List{ Elements{ srcList.Elems[1:] } }
                                                }
                                        } else if dstDel0.o.referencing(srcDel0.o) {
                                                var d *delegate
                                                switch t := dstDef.Value.(type) {
                                                case *List: d, _ = t.Elems[0].(*delegate)
                                                case *delegate: d = t
                                                }
                                                if d != nil && d.o == srcDef {
                                                        // The def value already contains src delegate.
                                                        return
                                                }
                                        }*/
                                        if srcDef != nil && dstDef != nil {
                                                var del *delegate
                                                
                                                // Case: srcDel0.o.referencing(dstDel0.o)
                                                switch t := srcDef.Value.(type) {
                                                case *List: del, _ = t.Elems[0].(*delegate)
                                                case *delegate: del = t
                                                }
                                                if del != nil && del.o == dstDef {
                                                        //fmt.Printf("append: %p %p\n", d.o, dstDef)
                                                        value = &List{ Elements{ srcList.Elems[1:] } }
                                                        goto AppendValue
                                                }

                                                // Case: dstDel0.o.referencing(srcDel0.o)
                                                switch t := dstDef.Value.(type) {
                                                case *List: del, _ = t.Elems[0].(*delegate)
                                                case *delegate: del = t
                                                }
                                                if del != nil && del.o == srcDef {
                                                        // The def value already contains src delegate.
                                                        return
                                                }
                                        }
                                }
                        }
                }
                AppendValue: _, err = def.Append(value)
        }
        fmt.Printf("defined: %v: %v = %v\n", project.Name(), p.Name(), src.Value)
        return
}

func MakeDefiner(op token.Token, name string) Value {
        return &definer{ name:name, op:op }
}

func (scope *Scope) NewDef(project *Project, name string, value Value) *Def {
	return &Def{
                object{
                        parent:  scope,
                        project: project,
                        name:    name,
                        typ:     DefType,
                        ord:     0,
                        scopos:  token.NoPos,
                },
                TrivialDef, value,
        }
}

func (scope *Scope) InsertDef(project *Project, name string, value Value) (def *Def, alt Object) {
        if alt = scope.elems[name]; alt == nil {
                def = scope.NewDef(project, name, value)
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
        program Program
        stem string // only applied for PatternRuleEntry
}

func (entry *RuleEntry) String() string { return fmt.Sprintf("entry %v", entry.name) }
func (entry *RuleEntry) Strval() string { return entry.name }
func (entry *RuleEntry) Stem() string { return entry.stem }

func (entry *RuleEntry) Class() RuleEntryClass { return entry.class }
func (entry *RuleEntry) SetClass(class RuleEntryClass) { entry.class = class }

// RuleEntry.Program returns the rule program.
func (entry *RuleEntry) Program() Program { return entry.program }

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
func (entry *RuleEntry) Call(a... Value) (result Value, err error) {
        if entry.program != nil {
                context := entry.Project().Scope()
                result, err = entry.program.Execute(context, entry, a, false)
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

func (p *PatternEntry) MakeConcreteEntry(stem string) (*RuleEntry, error) {
        return p.Pattern.MakeConcreteEntry(p.RuleEntry, stem)
}

/*type ArgumentedEntry struct {
        *RuleEntry
        Args []Value
}*/

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
        } else if name == "use" {
                if sn, ok := alt.(*ScopeName); ok && sn != nil {
                        entry, alt = sn.Scope().InsertEntry(project, UseRuleEntry, ":")
                }
        }
        return
}
