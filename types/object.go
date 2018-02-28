//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package types

import (
        //"path/filepath"
        "os/exec"
        "strings"
        "bytes"
        "fmt"
        "os"
        "github.com/duzy/smart/token"
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
func (obj *object) String() string { return fmt.Sprintf("object{%v}", obj.name) }
func (obj *object) Strval() (string, error) { return obj.String(), nil }

func (obj *object) Get(name string) (Value, error) {
        return nil, fmt.Errorf("No such property `%s' (Object).", name)
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
        return fmt.Sprintf("ProjectName{%s}", n.name)
}
func (n *ProjectName) Strval() (string, error)  {
        return fmt.Sprintf("project %s", n.name), nil
}

func (n *ProjectName) Get(name string) (Value, error) {
        if scope, sym := n.project.scope.Find(name); scope != nil && sym != nil {
                if false {
                        if o, _ := sym.(Object); o != nil && o.Project() != n.project {
                                //fmt.Printf("diverged: %v (%v != %v)\n", name, o.Project().Name(), n.project.Name())
                                //fmt.Printf("%v\n", n.project.scope)
                                //fmt.Printf("%v\n", n.project.scope.chain)
                                //fmt.Printf("%v\n", scope)
                                return nil, fmt.Errorf("Symbol diverged `%s'", name)
                        } else if value, _ := sym.(Value); value != nil {
                                return value, nil
                        } else {
                                return nil, fmt.Errorf("Symbol `%s' is not value (%T)", name, sym)
                        } 
                } else {
                        value, _ := sym.(Value); return value, nil
                }
        }
        return nil, fmt.Errorf("Undefined `%s' in project `%s'.", name, n.project.Name())
}

func (p *ProjectName) prepare(pc *Preparer) (err error) {
        var defent = p.project.DefaultEntry()
        if trace_prepare {
                fmt.Printf("prepare:ProjectName: project %v (default %v) (%v)\n", p.name, defent, pc.entry)
        }
        if defent != nil && defent.Name() != ":" {
                err = defent.prepare(pc)
        }
        return
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
func (n *ScopeName) String() string  { return fmt.Sprintf("ScopeName{%s}", n.name) }
func (n *ScopeName) Strval() (string, error) { return fmt.Sprintf("scope %s", n.name), nil }

func (n *ScopeName) Get(name string) (Value, error) {
        if sym := n.scope.Resolve(name); sym != nil {
                value, _ := sym.(Value)
                return value, nil
        }
        return nil, fmt.Errorf("Undefined `%s' in scope `%s'.", name, n.Name())
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

func (d *Def) disclose(scope *Scope) (res Value, err error) {
        var v Value
        if v, err = d.Value.disclose(scope); err != nil { return }
        if v != nil { res = &Def{ d.object, d.origin, v }}
        return
}

func (d *Def) referencing(o Object) bool {
        if d == o { return true }
        return d.Value.referencing(o)
}

func (d *Def) String() (s string) {
        s = d.name
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
        return fmt.Sprintf("Def{%s}", s)
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
        } else if v.referencing(d) {
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
        return d.Assign(strval(strings.TrimSpace(stdout.String())))
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

func (d *Def) Get(name string) (Value, error) {
        switch name {
        case "name": return strval(d.name), nil
        case "value": return d.Value, nil
        }
        //fmt.Printf("%v %v\n", d.name, d.parent)
        return nil, fmt.Errorf("No such property `%s' (Def)", name)
}

func (d *Def) compare(c *Comparer) error {
        if trace_compare {
                fmt.Printf("compare:Def: %v (%v %T)\n", d.Value, c.target, c.target)
        }
        return c.compare(d.Value)
}

func (d *Def) compareFileDepend(c *Comparer, file *File) (err error) {
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

func (d *Def) comparePathDepend(c *Comparer, path *Path) (err error) {
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

func (p *Builtin) String() string { return fmt.Sprintf("Builtin{%v}", p.name) }
func (p *Builtin) Strval() (string, error) { return fmt.Sprintf("builtin %v", p.name), nil }
func (p *Builtin) Call(pos token.Position, a... Value) (Value, error) {
        return p.f(pos, p.parent, a...)
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
        GlobRuleEntry
        RegexpRuleEntry
        StemmedRuleEntry
        StemmedFileEntry  // entry.file determined by PatternStem
        ExplicitFileEntry // entry.file determined by parser (may also set entry.path)
        ExplicitPathEntry // entry.path determined by parser
        UseRuleEntry
)

var namesForRuleEntryClass = []string{
        GeneralRuleEntry:  "GeneralRuleEntry",
        GlobRuleEntry:     "GlobRuleEntry",
        RegexpRuleEntry:   "RegexpRuleEntry",
        StemmedRuleEntry:  "StemmedRuleEntry",
        StemmedFileEntry:  "StemmedFileEntry",
        ExplicitFileEntry: "ExplicitFileEntry",
        ExplicitPathEntry: "ExplicitPathEntry",
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
        object
        class RuleEntryClass
        file *File  // For ExplicitFileEntry, StemmedFileEntry
        path *Path  // For ExplicitPathEntry
        stem string // For StemmedRuleEntry, StemmedFileEntry
        caller *Preparer
        programs []*Program
        //creator *PatternEntry
        Position token.Position
}

func (entry *RuleEntry) String() string { return fmt.Sprintf("RuleEntry{%v}", entry.name) }
func (entry *RuleEntry) Strval() (s string, e error) { return entry.name, nil }
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
        return entry.class == ExplicitFileEntry || entry.class == StemmedFileEntry;
}

func (entry *RuleEntry) SetExplicitFile(file *File) (prev *File) {
        prev = entry.file
        if file.Dir == "" {
                file.Dir = entry.project.AbsPath()
        }
        entry.class, entry.file = ExplicitFileEntry, file
        return
}

func (entry *RuleEntry) SetExplicitPath(path *Path) (prev *Path) {
        prev = entry.path
        entry.path = path
        if path.File == nil {
                entry.class = ExplicitPathEntry
        } else {
                entry.class, entry.file = ExplicitFileEntry, path.File
                if path.File.Dir == "" {
                        path.File.Dir = entry.project.AbsPath()
                }
        }
        return
}

// RuleEntry.Execute executes the rule program only if the target
// is outdated.
func (entry *RuleEntry) Execute(pos token.Position, a... Value) (result []Value, err error) {
        if entry.class == GlobRuleEntry || entry.class == StemmedFileEntry {
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
        case "class": return strval(entry.class.String()), nil
        case "name": return strval(entry.name), nil
        }
        return nil, fmt.Errorf("no such entry property (%s)", name)
}

func (entry *RuleEntry) setcaller(pc *Preparer) (prev *Preparer) {
        prev = entry.caller
        entry.caller = pc
        return
}

func (entry *RuleEntry) compare(c *Comparer) (err error) {
        if trace_compare {
                fmt.Printf("compare:RuleEntry: %v (%v) (%v %T)\n", entry.name, entry.class, c.target, c.target)
        }
        switch entry.class {
        case ExplicitFileEntry, StemmedFileEntry:
                err = c.target.compareFileDepend(c, entry.file)
        case ExplicitPathEntry:
                err = c.target.comparePathDepend(c, entry.path)
        default:
                err = breakf(false, "incomparable entry (%v)", entry.name)
        }
        return
}

func (entry *RuleEntry) compareFileDepend(c *Comparer, file *File) (err error) {
        if trace_compare {
                fmt.Printf("compare:RuleEntry:File: %v (%v) (depends: %v) (%v %T)\n", entry.name, entry.class, file, c.target, c.target)
        }
        switch entry.class {
        case ExplicitFileEntry, StemmedFileEntry:
                if entry.file != nil {
                        err = entry.file.compareFileDepend(c, file)
                } else {
                        err = breakf(false, "nil file entry (%v)", entry.name)
                }
        case ExplicitPathEntry:
                if entry.path != nil {
                        err = entry.path.compareFileDepend(c, file)
                } else {
                        err = breakf(false, "nil file entry (%v)", entry.name)
                }
        default:
                err = breakf(false, "incomparable entry (%v)", entry.name)
        }
        return
}

func (entry *RuleEntry) comparePathDepend(c *Comparer, path *Path) (err error) {
        if trace_compare {
                fmt.Printf("compare:RuleEntry:Path: %v (%v) (depends: %v) (%v %T)\n", entry.name, entry.class, path, c.target, c.target)
        }
        switch entry.class {
        case ExplicitFileEntry, StemmedFileEntry:
                if entry.file != nil {
                        err = entry.file.comparePathDepend(c, path)
                } else {
                        err = breakf(false, "nil file entry (%v)", entry.name)
                }
        case ExplicitPathEntry:
                if entry.path != nil {
                        err = entry.path.comparePathDepend(c, path)
                } else {
                        err = breakf(false, "nil file entry (%v)", entry.name)
                }
        default:
                if trace_compare {
                        fmt.Printf("compare:RuleEntry:Path: %v (%v) (incomparable) (%v %T)\n", entry.name, entry.class, c.target, c.target)
                }
                if false {
                        err = breakf(false, "incomparable entry (%v)", entry.name)
                }
        }
        return
}

func (entry *RuleEntry) prepare(pc *Preparer) (err error) {
        if trace_prepare {
                switch entry.class {
                case GeneralRuleEntry:
                        fmt.Printf("prepare:RuleEntry: %v (%v) (%v) (%v -> %v)\n", entry.name, entry.Depends(), entry.class, pc.entry.project.name, pc.entry)
                case ExplicitFileEntry:
                        fmt.Printf("prepare:RuleEntry: %v (%v) (%v) (%v) (%v -> %v)\n", entry.name, entry.Depends(), entry.class, entry.file, pc.entry.project.name, pc.entry)
                case StemmedFileEntry:
                        fmt.Printf("prepare:RuleEntry: %v (%v) (%v, stem=%v) (%v) (%v -> %v)\n", entry.name, entry.Depends(), entry.class, pc.stem, entry.file, pc.entry.project.name, pc.entry)
                case StemmedRuleEntry:
                        fmt.Printf("prepare:RuleEntry: %v (%v) (%v, stem=%v) (%v -> %v)\n", entry.name, entry.Depends(), entry.class, pc.stem, pc.entry.project.name, pc.entry)
                default:
                        fmt.Printf("prepare:RuleEntry: %v (%v) (%v) (%v -> %v)\n", entry.name, entry.Depends(), entry.class, pc.entry.project.name, pc.entry)
                }
        }

        // Set prepare context 
        defer entry.setcaller(entry.setcaller(pc))

        if trace_prepare {
                for i, prog := range entry.programs {
                        fmt.Printf("prepare:RuleEntry: %v (program[%v]:%v) (%v -> %v)\n", entry.name, i, prog.depends, pc.entry.project.name, pc.entry)
                }
        }

        ForPrograms: for i, prog := range entry.programs {
                if trace_prepare {
                        fmt.Printf("prepare:RuleEntry: %v (program[%v]:%v) (%s) (%v -> %v)\n", entry.name, i, prog.depends, entry.class, pc.entry.project.name, pc.entry)
                }
                if prog == pc.program {
                        err = fmt.Errorf("depended on itself")
                        fmt.Fprintf(os.Stdout, "%s: %v\n", prog.position, err)
                        break ForPrograms
                }
                if err = pc.execute(entry, prog); err == nil {
                        break ForPrograms
                } else if _, ok := err.(unknownTargetError); ok {
                        break ForPrograms // Don't try other programs if it's unknown.
                } else if entry.class == StemmedFileEntry {
                        break ForPrograms // Don't try other programs if it's pattern.
                }
        }
        return
}

func (pc *Preparer) execute(entry *RuleEntry, prog *Program) (err error) {
        if trace_prepare {
                switch entry.class {
                case GeneralRuleEntry:
                        fmt.Printf("prepare:Execute: %v (%v) (%v) (%v -> %v)\n", entry.name, prog.depends, entry.class, pc.entry.project.name, pc.entry)
                case ExplicitFileEntry:
                        fmt.Printf("prepare:Execute: %v (%v) (%v) (file: %v) (%v -> %v)\n", entry.name, prog.depends, entry.class, entry.file, pc.entry.project.name, pc.entry)
                case StemmedFileEntry:
                        fmt.Printf("prepare:Execute: %v (%v) (%v, stem=%v) (file: %v) (%v -> %v)\n", entry.name, prog.depends, entry.class, pc.stem, entry.file, pc.entry.project.name, pc.entry)
                case StemmedRuleEntry:
                        fmt.Printf("prepare:Execute: %v (%v) (%v, stem=%v) (%v -> %v)\n", entry.name, prog.depends, entry.class, pc.stem, pc.entry.project.name, pc.entry)
                default:
                        fmt.Printf("prepare:Execute: %v (%v) (%v) (%v -> %v)\n", entry.name, prog.depends, entry.class, pc.entry.project.name, pc.entry)
                }
                for i, depent := range prog.depends {
                        fmt.Printf("prepare:Execute: %v (depend[%d]: %v %v)\n", entry.name, i, depent, entry.stem)
                }
        }

        var (
                caller = pc
                res Value
        )

        // Fixes program context if the starting entry and depended entry are
        // in different projects. This ensure disclosures work.
        ForCallers: for c := pc; c != nil; c = c.program.caller {
                if c.program.project != prog.project {
                        if caller = c; trace_prepare {
                                fmt.Printf("prepare:Execute: %v (%s -> %s) 🗸 \n", entry.name, prog.project.name, caller.program.project.name)
                        }
                        break ForCallers
                } else if trace_prepare {
                        fmt.Printf("prepare:Execute: %v (%s -> %s)\n", entry.name, prog.project.name, c.program.project.name)
                }
        }
        
        defer prog.setCallerContext(prog.setCallerContext(caller, caller.program.project.scope))

        // Execute the updating program.
        if res, err = prog.Execute(entry, pc.arguments); err == nil {
                switch dd, _ := prog.scope.Lookup("@").(*Def).Call(entry.Position); entry.class {
                case ExplicitFileEntry, StemmedFileEntry:
                        if trace_prepare {
                                fmt.Printf("prepare:Execute: %v (%v) (append %s (%T)) (%v) (%v)\n",
                                        entry.name, entry.class, dd, dd, entry.file, pc.entry)
                        }
                        if file, _ := dd.(*File); file != nil {
                                // TODO: assert(file == entry.file)
                                pc.targets.Append(file)
                        } else {
                                var s string
                                if s, err = dd.Strval(); err != nil {
                                        return
                                }
                                pc.targets.Append(caller.program.project.SearchFile(s))
                        }
                case ExplicitPathEntry:
                        if trace_prepare {
                                fmt.Printf("prepare:Execute: %v (%v) (append %s (%T)) (%v) (%v)\n",
                                        entry.name, entry.class, dd, dd, entry.path, pc.entry)
                        }
                        if entry.path == nil {
                                pc.targets.Append(entry)
                        } else if entry.path.File == nil {
                                pc.targets.Append(entry.path)
                        } else {
                                pc.targets.Append(entry.path.File)
                        }
                default:
                        if res != nil && res.Type() != NoneType {
                                pc.targets.Append(res); return
                        } else {
                                pc.targets.Append(entry)
                        }
                }
                if res != nil && res.Type() != NoneType {
                        for _, elem := range Join(res) {
                                switch elem.(type) {
                                case *File: pc.targets.Append(elem)
                                }
                        }
                }
        } else {
                fmt.Fprintf(os.Stdout, "%s: %v\n", prog.position, err)
                if trace_prepare {
                        fmt.Printf("prepare:Execute: %v (%v) (error) (%v)\n", entry.name, prog.depends, pc.entry)
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
                        kind, nil, nil, "", nil, nil, //nil,
                        token.Position{},
                }
                scope.replace(name, entry)
        } else if name == "use" {
                if sn, ok := alt.(*ScopeName); ok && sn != nil {
                        entry, alt = sn.Scope().InsertEntry(project, UseRuleEntry, ":")
                }
        }
        return
}
