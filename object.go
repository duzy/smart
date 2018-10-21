//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "os/exec"
        "strings"
        "bytes"
        "fmt"
        "os"
)

// Object is a value defined in a scope.
//
// TODO: defines ObjInfo to classify objects.
// 
type Object interface {
        Value

        Name() string

        DeclScope() *Scope
        OwnerProject() *Project

        // Get object's named property.
        Get(name string) (Value, error)
        
	// redecl the object.
	redecl(*Scope)
}

// An object implements the common parts of an Object.
type object struct {
        value
        scope *Scope
        owner *Project
        name string
        typ Type
}

func (obj *object) Type() Type { return obj.typ }
func (obj *object) Name() string { return obj.name }

func (obj *object) DeclScope() *Scope { return obj.scope }
func (obj *object) OwnerProject() *Project { return obj.owner }

func (obj *object) Strval() (string, error) { return fmt.Sprintf("{object %+v,%+v}", obj.typ, obj.name), nil }
func (obj *object) String() string {
        if s, e := obj.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{object '%s' !(%+v)}", s, e)
        }
}

func (obj *object) Get(name string) (Value, error) {
        return nil, fmt.Errorf("no such property `%s' (Object)", name)
}

func (obj *object) redecl(scope *Scope) {
        if obj.scope != scope {
                if obj.scope != nil {
                        delete(obj.scope.elems, obj.name)
                }
                if obj.scope = scope; obj.scope != nil {
                        obj.scope.elems[obj.name] = obj
                }
        }
}

type ProjectName struct {
        object
        project *Project
}

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (p *ProjectName) Type() Type { return ProjectNameType }
func (p *ProjectName) NamedProject() *Project { return p.project }
func (p *ProjectName) Strval() (string, error) { return fmt.Sprintf("project %s", p.name), nil }
func (p *ProjectName) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{ProjectName '%s' !(%+v)}", s, e)
        }
}

func (p *ProjectName) Get(name string) (value Value, err error) {
        /*if scope, sym := p.project.scope.Find(name); scope != nil && sym != nil {
                value, _ = sym.(Value); return
        }
        err = fmt.Errorf("`%s` is undefined (in %v)", name, p.project.scope.comment)*/
        value, err = p.project.resolveObject(name)
        return
}

func (p *ProjectName) prepare(pc *preparer) (err error) {
        var defent = p.project.DefaultEntry()
        if trace_prepare {
                fmt.Printf("prepare:ProjectName: project %v (default %v) (%v)\n", p.name, defent, pc.entry)
        }
        if defent != nil && defent.class != UseRuleEntry {
                err = defent.prepare(pc)
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
func (n *ScopeName) NamedScope() *Scope { return n.scope }
func (n *ScopeName) String() string  {
        if s, e := n.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{ScopeName '%s' !(%+v)}", s, e)
        }
}
func (n *ScopeName) Strval() (string, error) { return fmt.Sprintf("scope %s", n.name), nil }

func (n *ScopeName) Get(name string) (Value, error) {
        if sym := n.scope.Resolve(name); sym != nil {
                value, _ := sym.(Value)
                return value, nil
        }
        return nil, fmt.Errorf("Undefined `%s' in scope `%s'.", name, n.Name())
}

// Represents a unknown object, may be referred by some closures
type UnknownObject struct { object }
func (p *UnknownObject) String() string { return p.name } // the source representation
func (p *UnknownObject) Strval() (string, error) { return fmt.Sprintf("{UnknownObject %s}", p.name), nil }
func (p *UnknownObject) Call(pos token.Position, a... Value) (result Value, err error) {
        result = p; return
}
func (p *UnknownObject) Execute(pos token.Position, a... Value) (result []Value, err error) {
        result = []Value{p}; return
}

func MakeUnknownObject(s string) *UnknownObject {
        return &UnknownObject{object{name:s, typ:UnknownObjectType}}
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

func (d *Def) disclose() (res Value, err error) {
        var v Value
        if v, err = d.Value.disclose(); err != nil { return }
        if v != nil { res = &Def{ d.object, d.origin, v }}
        return
}
func (d *Def) reveal() (res Value, err error) {
        var v Value
        if v, err = d.Value.reveal(); err != nil { return }
        if v != nil { res = &Def{ d.object, d.origin, v }}
        return
}

func (d *Def) refs(o Object) bool {
        if d == o { return true }
        return d.Value.refs(o)
}

func (d *Def) closured() bool { return d.Value.closured() }

func (d *Def) String() (s string) {
        if s, e := d.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Def '%s' !(%+v)}", s, e)
        }
}
func (d *Def) Strval() (s string, e error) {
        s = d.name + "="
        if d.Value == nil {
                s += "<nil>"
        } else {
                var v string
                if v, e = d.Value.Strval(); e == nil {
                        s += v
                } else {
                        return
                }
        }
        return
}
func (d *Def) Origin() DefOrigin { return d.origin }
func (d *Def) SetOrigin(k DefOrigin) { d.origin = k }

func (d *Def) Assign(v Value) (res Value, err error) {
        if v == nil {
                v = UniversalNone
        } else if v.refs(d) {
                err = fmt.Errorf("Recursive variable `%s' references itself.", d.name)
                return
        }
        
        switch d.origin {
        case TrivialDef, DefaultDef:
                d.Value = v // Keeps delegates and closures.
        case ImmediateDef:
                // Eval expends delegates in the value.
                if d.Value, err = Reveal(v); err != nil { return }
        }
        res = d.Value
        return
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

func (d *Def) AssignExec(a... Value) (res Value, err error) {
        var (
                stdout bytes.Buffer
                stderr bytes.Buffer
        )
        for _, v := range a {
                var s string
                if s, err = v.Strval(); err == nil {
                        sh := exec.Command("sh", "-c", s)
                        sh.Stdout, sh.Stderr = &stdout, &stderr
                        if err = sh.Run(); err != nil {
                                res, err = d.Assign(UniversalNone)
                                return
                        }
                } else {
                        res, err = d.Assign(UniversalNone)
                        return
                }
        }
        return d.Assign(MakeString(strings.TrimSpace(stdout.String())))
}

func (d *Def) Call(pos token.Position, a... Value) (res Value, err error) {
        // TODO: parameterization, e.g. $1, $2, $3, $4, $5
        if d.origin != ImmediateDef {
                res = d.Value
        } else if res, err = Reveal(d.Value); err != nil {
                //fmt.Printf("%v: %v\n", d.position, err)
        }
        return
}

func (d *Def) DiscloseValue() (res Value, err error) {
        if d.Value != nil {
                if res, err = d.Value.disclose(); err != nil { return }
                if res == nil { res = d.Value }
        }
        return
}

func (d *Def) Get(name string) (Value, error) {
        switch name {
        case "name": return MakeString(d.name), nil
        case "value": return d.Value, nil
        }
        //fmt.Printf("%v %v\n", d.name, d.parent)
        return nil, fmt.Errorf("No such property `%s' (Def)", name)
}

func (d *Def) compare(c *comparer) error {
        if trace_compare {
                fmt.Printf("compare:Def: %v (%v %T)\n", d.Value, c.target, c.target)
        }
        return c.compare(d.Value)
}

func (d *Def) compareFileDepend(c *comparer, file *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:Def:File: %v (depends: %v) (%v %T)\n", d.Value, file, c.target, c.target)
        }
        if comp, _ := d.Value.(comparable); comp != nil {
                err = comp.compareFileDepend(c, file)
        } else {
                err = breakf(false, "incomparable target (%v)", d.Value)
        }
        return
}

func (d *Def) comparePathDepend(c *comparer, path *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:Def:Path: %v (depends: %v) (%v %T)\n", d.Value, path, c.target, c.target)
        }
        if comp, _ := d.Value.(comparable); comp != nil {
                err = comp.comparePathDepend(c, path)
        } else {
                err = breakf(false, "incomparable target (%v)", d.Value)
        }
        return
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        object
        f BuiltinFunc
}

func (p *Builtin) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Builtin '%s' !(%+v)}", s, e)
        }
}
func (p *Builtin) MakeString() (string, error) { return fmt.Sprintf("builtin %v", p.name), nil }
func (p *Builtin) Call(pos token.Position, a... Value) (Value, error) { return p.f(pos, a...) }

type RuleEntryClass int

const (
        GeneralRuleEntry RuleEntryClass = iota
        GlobRuleEntry
        RegexpRuleEntry
        UseRuleEntry
)

var namesForRuleEntryClass = []string{
        GeneralRuleEntry:  "GeneralRuleEntry",
        GlobRuleEntry:     "GlobRuleEntry",
        RegexpRuleEntry:   "RegexpRuleEntry",
        UseRuleEntry:      "UseRuleEntry",
}

func (c RuleEntryClass) String() string {
        var i = int(c)
        if 0 <= i && i < len(namesForRuleEntryClass) {
                return namesForRuleEntryClass[i]
        }
        return fmt.Sprintf("RuleEntryClass(%d)", i)
}

// RuleEntry represents a declared rule entry.
type RuleEntry struct {
        class RuleEntryClass
        target Value
        programs []*Program
        Position token.Position
}

func (entry *RuleEntry) Type() Type { return RuleEntryType }
func (entry *RuleEntry) Float() (float64, error) { return 0, nil }
func (entry *RuleEntry) Integer() (int64, error) { return 0, nil }
func (entry *RuleEntry) OwnerProject() *Project { return entry.programs[0].project }
func (entry *RuleEntry) DeclScope() *Scope { return entry.OwnerProject().scope }
func (entry *RuleEntry) Name() string {
        if entry == nil {
                panic("entry is nil")
        } else if entry.target == nil {
                fmt.Fprintf(os.Stderr, "%v: nil target", entry.Position)
                panic("entry target is nil")
        }
        s, err := entry.target.Strval()
        if err != nil { panic(err) } // FIXME: error
        return s
}
func (entry *RuleEntry) Strval() (string, error) { return entry.target.Strval() }
func (entry *RuleEntry) String() string {
        if s, e := entry.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{RuleEntry '%s' !(%+v)}", s, e)
        }
}
func (entry *RuleEntry) Class() RuleEntryClass { return entry.class }
func (entry *RuleEntry) SetClass(class RuleEntryClass) { entry.class = class }
func (entry *RuleEntry) Programs() []*Program { return entry.programs }
func (entry *RuleEntry) Depends() (depends []Value) {
        for _, prog := range entry.programs {
                depends = append(depends, prog.depends...)
        }
        return
}

func (entry *RuleEntry) IsFile() bool {
        if entry.target.Type() == FileType { return true }
        if p, ok := entry.target.(*Path); ok && p != nil && p.File != nil {
                return true
        }
        return false
}

func (entry *RuleEntry) SetExplicitFile(file *File) {
        if file.Dir == "" {
                file.Dir = entry.OwnerProject().AbsPath()
        }
        if path, ok := entry.target.(*Path); ok && path != nil {
                path.File = file
        }
        return
}

func (entry *RuleEntry) SetExplicitPath(path *Path) {
        if path.File != nil && path.File.Dir == "" {
                path.File.Dir = entry.OwnerProject().AbsPath()
        }
        //if path, ok := entry.target.(*Path); ok && path != nil {
        //        path
        //}
        return
}


// RuleEntry.Execute executes the rule program only if the target is outdated.
func (entry *RuleEntry) Execute(pos token.Position, a... Value) (result []Value, err error) {
        if entry.class == GlobRuleEntry /*|| entry.class == StemmedFileEntry*/ {
                return nil, fmt.Errorf("%s: executing pattern entry '%s'.", pos, entry.Name())
        }
        for _, program := range entry.programs {
                if v, e := program.Execute(entry, a); e != nil {
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
        case "class": return MakeString(entry.class.String()), nil
        case "name": return MakeString(entry.Name()), nil
        // case "prerequisites": ...
        }
        return nil, fmt.Errorf("no such entry property (%s)", name)
}

func (entry *RuleEntry) redecl(scope *Scope) {
        panic("RuleEntry.redecl not supported")
}

func (entry *RuleEntry) refs(o Object) bool {
        if entry.target.refs(o) { return true }
        
        // TODO: do more tests for this to see if we need to fallthrough
        return false // only check closured agaist target

        for _, prog := range entry.programs {
                for _, m := range prog.pipline {
                        for _, a := range m.args {
                                if a.refs(o) { return true }
                        }
                }
                for _, depend := range prog.depends {
                        if depend.refs(o) { return true }
                }
                for _, recipe := range prog.recipes {
                        if recipe.refs(o) { return true }
                }
        }
        return false
}

func (entry *RuleEntry) closured() bool {
        if entry.target.closured() { return true }
        
        // TODO: do more tests for this to see if we need to fallthrough
        return false // only check closured agaist target

        for _, prog := range entry.programs {
                for _, m := range prog.pipline {
                        for _, a := range m.args {
                                if a.closured() { return true }
                        }
                }
                for _, depend := range prog.depends {
                        if depend.closured() { return true }
                }
                for _, recipe := range prog.recipes {
                        if recipe.closured() { return true }
                }
        }
        return false
}
func (entry *RuleEntry) disclose() (res Value, err error) {
        var target Value
        if target, err = entry.target.disclose(); err != nil { return }
        if target != nil {
                // TODO: test if programs are needed to be disclosed??
                res = &RuleEntry{
                        entry.class, target, entry.programs, entry.Position,
                }
        }
        return
}
func (entry *RuleEntry) reveal() (res Value, err error) {
        var target Value
        if target, err = entry.target.reveal(); err != nil { return }
        if target != nil {
                // TODO: test if programs are needed to be revealed??
                res = &RuleEntry{
                        entry.class, target, entry.programs, entry.Position,
                }
        }
        return
}

func (entry *RuleEntry) compare(c *comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:RuleEntry: %v (%v) (%v %T)\n", entry.target, entry.class, c.target, c.target)
        }
        switch target := entry.target.(type) {
        case *File: err = c.target.compareFileDepend(c, target)
        case *Path: err = c.target.comparePathDepend(c, target)
        default: err = breakf(false, "incomparable entry (%v)", target)
        }
        return
}

func (entry *RuleEntry) compareFileDepend(c *comparer, file *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:RuleEntry:File: %v (%v) (depends: %v) (%v %T)\n", entry.target, entry.class, file, c.target, c.target)
        }
        switch target := entry.target.(type) {
        case *File: err = target.compareFileDepend(c, file)
        case *Path: err = target.compareFileDepend(c, file)
        default: err = breakf(false, "incomparable entry (%v)", target)
        }
        return
}

func (entry *RuleEntry) comparePathDepend(c *comparer, path *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:RuleEntry:Path: %v (%v) (depends: %v) (%v %T)\n", entry.target, entry.class, path, c.target, c.target)
        }
        switch target := entry.target.(type) {
        case *File: err = target.comparePathDepend(c, path)
        case *Path: err = target.comparePathDepend(c, path)
        default: err = breakf(false, "incomparable entry (%v)", target)
        }
        return
}

func (entry *RuleEntry) prepare(pc *preparer) (err error) {
        if trace_prepare {
                fmt.Printf("prepare:RuleEntry: %v (%v) (%v) (%v -> %v)\n", entry.target, entry.Depends(), entry.class, pc.entry.OwnerProject().name, pc.entry)
        }

        if trace_prepare {
                for i, prog := range entry.programs {
                        fmt.Printf("prepare:RuleEntry: %v (program[%v]:%v) (%v -> %v)\n", entry.target, i, prog.depends, pc.entry.OwnerProject().name, pc.entry)
                }
        }

        ForPrograms: for i, prog := range entry.programs {
                if trace_prepare {
                        fmt.Printf("prepare:RuleEntry: %v (program[%v]:%v) (%s) (%v -> %v)\n", entry.target, i, prog.depends, entry.class, pc.entry.OwnerProject().name, pc.entry)
                }
                if prog == pc.program {
                        err = fmt.Errorf("depended on itself")
                        fmt.Fprintf(os.Stdout, "%s: %v\n", prog.position, err)
                        break ForPrograms
                }

                if err = pc.execute(entry, prog); err == nil {
                        break ForPrograms
                } else if _, ok := err.(targetNotFoundError); ok {
                        break ForPrograms // Don't try other programs if it's unknown.
                }
        }
        return
}

type PatternEntry struct {
        *RuleEntry
        Pattern Pattern
}

func (p *PatternEntry) MakeConcreteEntry(stem string) (entry *RuleEntry, err error) {
        if entry, err = p.Pattern.MakeConcreteEntry(p.RuleEntry, stem); err == nil && entry != nil {
                // entry.creator = p
        }
        return
}

type StemmedEntry struct {
        Patent *PatternEntry
        Stem string
        target string // source target matched the pattern
        file *File // source file matched the pattern
}

//func (ps *StemmedEntry) Strval() (string, error) {return ps.target, nil }
func (ps *StemmedEntry) String() (s string) {
        return fmt.Sprintf("{StemmedEntry %s,%s,%s,%s}", ps.Patent, ps.Stem, ps.target, ps.file)
}

func (ps *StemmedEntry) MakeConcreteEntry() (*RuleEntry, error) {
        return ps.Patent.MakeConcreteEntry(ps.Stem)
}

func (ps *StemmedEntry) prepare(pc *preparer) (err error) {
        if trace_prepare {
                if ps.file != nil {
                        fmt.Printf("prepare:StemmedEntry: %v (%v) (file: %v) (%v -> %v)\n", ps, ps.Patent.class, ps.file, pc.entry.OwnerProject().name, pc.entry)
                } else if ps.target != "" {
                        fmt.Printf("prepare:StemmedEntry: %v (%v) (target: %v) (%v -> %v)\n", ps, ps.Patent.class, ps.target, pc.entry.OwnerProject().name, pc.entry)
                } else {
                        fmt.Printf("prepare:StemmedEntry: %v (%v) (%v -> %v)\n", ps, ps.Patent.class, pc.entry.OwnerProject().name, pc.entry)
                }
        }
        
        var (
                stems = []string{ ps.Stem }
                sources = []string{ ps.target }
                entry *RuleEntry
        )
        if ps.file != nil {
                sources = append(sources, ps.file.Name)
        }

        // Find all useful stems.
        ForSources: for _, source := range sources {
                var ( stem string; ok bool )
                if source == "" { continue }
                if ok, stem, err = ps.Patent.Pattern.Match(source); ok && stem != "" {
                        for _, s := range stems { if s == stem { continue ForSources } }
                        stems = append(stems, stem)
                }
        }

        // Try preparing target with all stems.
        ForStems: for i, stem := range stems {
                if entry, err = ps.Patent.MakeConcreteEntry(stem); err != nil {
                        return
                }

                fmt.Printf("prepare: stemmed %v\n", entry)

                if trace_prepare {
                        fmt.Printf("prepare:StemmedEntry: %v (%v) ([%d/%d]: %v %v) (file: %v) (%v -> %v)\n", ps, entry.class, i, len(stems), entry.Depends(), stem, ps.file, pc.entry.OwnerProject().name, pc.entry)
                }

                pc.stem = stem // set for the current stem.
                if err = entry.prepare(pc); err == nil {
                        break ForStems // Good!
                } else if ute, ok := err.(targetNotFoundError); ok {
                        fmt.Printf("prepare:StemmedEntry: FIXME: unknown target %v (%v)\n", ute.target, pc.entry)
                } else if ufe, ok := err.(fileNotFoundError); ok {
                        fmt.Printf("prepare:StemmedEntry: FIXME: unknown file %v (%v)\n", ufe.file, pc.entry)
                }
        }
        return
}
