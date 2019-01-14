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

type unknownobject struct { // generally unnamed objects
        scope *Scope
        owner *Project
}
func (p *unknownobject) refs(_ Value) bool { return false }
func (p *unknownobject) closured() bool { return false }
func (p *unknownobject) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *unknownobject) Type() Type { return UnknownObjectType }
func (p *unknownobject) True() bool { return false }
func (p *unknownobject) Name() string { panic("inquiring name of an unknown object") }
func (p *unknownobject) DeclScope() *Scope { return p.scope }
func (p *unknownobject) OwnerProject() *Project { return p.owner }
func (p *unknownobject) Strval() (string, error) { return fmt.Sprintf("{unknown %p}", p), nil }
func (p *unknownobject) String() string { return fmt.Sprintf("{unknown %p}", p) }
func (p *unknownobject) Integer() (int64, error) { return 0, nil }
func (p *unknownobject) Float() (float64, error) { return 0, nil }
func (p *unknownobject) Get(name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p *unknownobject) redecl(scope *Scope) { panic("redeclaring unknown object") }
func (p *unknownobject) cmp(v Value) (res cmpres) {
        if v.Type() == UnknownObjectType {
                a, ok := v.(*unknownobject)
                assert(ok, "value is not unknownobject")
                if p.owner == a.owner && p.scope == a.scope {
                        res = cmpEqual
                }
        }
        return
}

type knownobject struct { // generally named objects
        unknownobject
        name string
}
func (p *knownobject) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *knownobject) Type() Type { return KnownObjectType }
func (p *knownobject) True() bool { return true }
func (p *knownobject) Name() string { return p.name }
func (p *knownobject) Strval() (string, error) { return fmt.Sprintf("{object %s}", p.name), nil }
func (p *knownobject) String() string { return fmt.Sprintf("{object %s}", p.name) }
func (p *knownobject) redecl(scope *Scope) {
        if p.scope != scope {
                if p.scope != nil {
                        delete(p.scope.elems, p.name)
                }
                if p.scope = scope; p.scope != nil {
                        p.scope.elems[p.name] = p
                }
        }
}
func (p *knownobject) cmp(v Value) (res cmpres) {
        if v.Type() == KnownObjectType {
                a, ok := v.(*knownobject)
                assert(ok, "value is not knownobject")
                if p.owner == a.owner && p.scope == a.scope && p.name == a.name {
                        res = cmpEqual
                }
        }
        return
}

type unresolvedobject struct { // named callable/executable objects
        unknownobject
        name Value // name could be closured
}
func (p *unresolvedobject) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *unresolvedobject) Type() Type { return UnresolvedObjectType }
func (p *unresolvedobject) True() bool { return false }
func (p *unresolvedobject) Name() string {
        if p.name == nil {
                panic("unresolved object name is nil")
        } else if s, err := p.name.Strval(); err != nil {
                panic(fmt.Sprintf("unresolved object name: %v: %v", p.name, err))
        } else {
                return s
        }
}
func (p *unresolvedobject) String() string { return p.name.String() }
func (p *unresolvedobject) Strval() (string, error) {
        // The string value of a unresolved object is "", so that a
        // unresolved &(var) is stringed to ""
        return /*p.name.Strval()*/"", nil
}
func (p *unresolvedobject) Call(pos token.Position, a... Value) (result Value, err error) { result = p; return }
func (p *unresolvedobject) Execute(pos token.Position, a... Value) (result []Value, err error) { result = []Value{p}; return }
func (p *unresolvedobject) redecl(scope *Scope) {
        if p.scope != scope {
                name, err := p.name.Strval()
                if err != nil { panic(fmt.Sprintf("unresolved name error: %v", p.name, err)) }
                if p.scope != nil { delete(p.scope.elems, name) }
                if p.scope = scope; p.scope != nil {
                        p.scope.elems[name] = p
                }
        }
}
func (p *unresolvedobject) cmp(v Value) (res cmpres) {
        if v.Type() == UnresolvedObjectType {
                a, ok := v.(*unresolvedobject)
                assert(ok, "value is not unresolvedobject")
                if p.owner == a.owner && p.scope == a.scope {
                        res = p.name.cmp(a.name)
                }
        }
        return
}

func unresolved(p *Project, v Value) *unresolvedobject {
        return &unresolvedobject{unknownobject{ scope: p.scope, owner: p }, v}
}

type ProjectName struct {
        knownobject
        project *Project
}

func (p *ProjectName) expand(_ expandwhat) (Value, error) { return p, nil }

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (p *ProjectName) Type() Type { return ProjectNameType }
func (p *ProjectName) True() bool { return p.project != nil }
func (p *ProjectName) NamedProject() *Project { return p.project }
func (p *ProjectName) Strval() (string, error) { return p.name, nil }
func (p *ProjectName) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{ProjectName '%s' !(%+v)}", s, e)
        }
}

func (p *ProjectName) Get(name string) (value Value, err error) {
        if p.project != nil {
                value, err = p.project.resolveObject(name)
        }
        return
}

// Call a ProjectName returns the project name.
func (p *ProjectName) Call(pos token.Position, a... Value) (value Value, err error) {
        if p.project != nil {
                value = &String{ p.project.name }
        }
        return
}

func (p *ProjectName) prepare(pc *preparer) (err error) {
        if trace_prepare { defer prepun(preptrace(pc, p)) }

        var defent = p.project.DefaultEntry()
        if defent != nil && defent.class != UseRuleEntry {
                err = defent.prepare(pc)
        }
        return
}

func (p *ProjectName) cmp(v Value) (res cmpres) {
        if v.Type() == ProjectNameType {
                a, ok := v.(*ProjectName)
                assert(ok, "value is not ProjectName")
                if p.name == a.name && p.project == a.project {
                        res = cmpEqual
                }
        }
        return
}

type ScopeName struct {
        knownobject
        scope *Scope
}

func (p *ScopeName) expand(_ expandwhat) (Value, error) { return p, nil }

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (n *ScopeName) Type() Type { return ScopeNameType }
func (n *ScopeName) True() bool { return n.scope != nil }
func (n *ScopeName) NamedScope() *Scope { return n.scope }
func (n *ScopeName) String() string  { return fmt.Sprintf("{scope %s}", n.name) }
func (n *ScopeName) Strval() (string, error) { return fmt.Sprintf("scope %s", n.name), nil }

func (n *ScopeName) Get(name string) (Value, error) {
        if sym := n.scope.Resolve(name); sym != nil {
                value, _ := sym.(Value)
                return value, nil
        }
        return nil, fmt.Errorf("Undefined `%s' in scope `%s'.", name, n.Name())
}

func (p *ScopeName) cmp(v Value) (res cmpres) {
        if v.Type() == ScopeNameType {
                a, ok := v.(*ScopeName)
                assert(ok, "value is not ScopeName")
                if p.name == a.name && p.scope == a.scope {
                        res = cmpEqual
                }
        }
        return
}

type DefOrigin int
const (
        // =
        DefDefault DefOrigin = iota // normal value
        
        // :=
        DefSimple // expand delegates

        // ::=
        DefExpand // expand all (delegates, closures, paths)

        // !=
        DefExecute // executed result
)

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type Def struct {
        knownobject
        origin DefOrigin
        Value Value
}

func (d *Def) refs(v Value) bool { return d == v || (d.Value != nil && d.Value.refs(v)) }
func (d *Def) closured() bool { return d.Value.closured() }
func (d *Def) expand(w expandwhat) (res Value, err error) {
        if res = d; d.Value != nil {
                var value Value
                if value, err = d.Value.expand(w); err == nil {
                        if value != d.Value {
                                res = &Def{ d.knownobject, d.origin, value }
                        }
                }
        }
        return
}

func (d *Def) cmp(v Value) (res cmpres) {
        if v.Type() == DefType {
                if d.Value != nil {
                        a, ok := v.(*Def)
                        assert(ok, "value is not Def")
                        if a.Value != nil {
                                res = d.Value.cmp(a.Value)
                        }
                }
        }
        return
}

func (d *Def) Type() Type { return DefType }
func (d *Def) True() bool { return d.Value.True() }
func (d *Def) String() (s string) {
        switch s = d.name; d.origin {
        case DefDefault: s += "="
        case DefSimple:  s += ":="
        case DefExpand:  s += "::="
        case DefExecute: s += "!="
        default: s += " = "
        }
        if d.Value != nil {
                s += elementString(d, d.Value)
        } else {
                s += "<nil>"
        }
        return
}
func (d *Def) Strval() (s string, e error) {
        if d.Value != nil {
                s, e = d.Value.Strval()
        }
        return
}

func (d *Def) set(origin DefOrigin, value Value) (err error) {
        if value != nil && value.refs(d) {
                err = fmt.Errorf("self recursive variable `%s`", d.name)
                return
        }

        d.origin = origin

        if origin != DefExecute && value == nil {
                value = universalnone
        }

        switch origin {
        case DefDefault: // Keeps delegates and closures.
                d.Value = value
        case DefSimple: // Eval expands delegates in the value.
                if d.Value, err = value.expand(expandDelegate); err != nil { return }
        case DefExpand:
                if d.Value, err = value.expand(expandAll); err != nil { return }
        case DefExecute:
                var ( stdout, stderr bytes.Buffer; s string )
                if value == nil || value.Type() == NoneType {
                        d.Value = nil // undef
                } else if s, err = value.Strval(); err == nil {
                        sh := exec.Command("sh", "-c", s)
                        sh.Stdout, sh.Stderr = &stdout, &stderr
                        if err = sh.Run(); err != nil { value = universalnone } else {
                                value = &String{strings.TrimSpace(stdout.String())}
                        }
                        d.Value = value
                } else {
                        d.Value = universalnone
                }
        default:
                unreachable()
        }
        return
}

func (d *Def) append(va... Value) (err error) {
        var list *List
        if num := len(va); num == 0 {
                // Does nothing...
        } else if d.Value != nil && d.Value.Type() != NoneType {
                if list, _ = d.Value.(*List); list != nil {
                        list.Append(merge(va...)...)
                } else {
                        elems := []Value{ d.Value }
                        elems = append(elems, merge(va...)...)
                        list = &List{elements{ elems }}
                }
        } else {
                list = &List{elements{ merge(va...) }}
        }
        if list != nil {
                var origin = d.origin
                if origin == DefExecute { origin = DefDefault }
                err = d.set(origin, list)
        }
        return
}

func (d *Def) Call(pos token.Position, a... Value) (res Value, err error) {
        switch d.origin {
        case DefSimple, DefExpand, DefExecute:
                res = d.Value
        case DefDefault:
                // TODO: parameterization, e.g. $1, $2, $3, $4, $5
                res, err = d.Value.expand(expandDelegate)
        default:
                unreachable()
        }
        return
}

func (d *Def) DiscloseValue() (res Value, err error) {
        if d.Value != nil {
                if res, err = d.Value.expand(expandClosure); err != nil { return }
                if res == nil { res = d.Value }
        }
        return
}

func (d *Def) Get(name string) (Value, error) {
        switch name {
        case "name": return &String{d.name}, nil
        case "value": return d.Value, nil
        }
        //fmt.Printf("%v %v\n", d.name, d.parent)
        return nil, fmt.Errorf("no such property `%s' (Def)", name)
}

func (d *Def) dependcompare(c *comparer) error {
        if trace_compare { defer compun(comptrace(c, d))}
        if enable_assertions { assert(c.target != d, "self comparation") }
        return c.compareDepend(d.Value)
}

func (d *Def) prepare(pc *preparer) (err error) {
        if d.Value != nil {
                if p, ok := d.Value.(prerequisite); ok {
                        err = p.prepare(pc)
                } else {
                        err = fmt.Errorf("%s: %s '%s' is not prerequisite", d.name, d.Value.Type(), d.Value)
                }
        }
        return
}

type undetermined struct {
        tok token.Token
        identifier Value
        value Value
}

func (p *undetermined) refs(v Value) bool {
        return p.identifier.refs(v) || p.value.refs(v)
}

func (p *undetermined) closured() bool {
        return p.identifier.closured() || p.value.closured()
}

func (p *undetermined) expand(w expandwhat) (res Value, err error) {
        var i, v Value
        res = p // set the original value
        if i, err = p.identifier.expand(w); err == nil {
                if v, err = p.value.expand(w); err == nil {
                        if i != p.identifier || v != p.value {
                                res = &undetermined{ p.tok, i, v }
                        }
                }
        }
        return
}

func (p *undetermined) Type() Type { return UndeterminedType }
func (p *undetermined) True() bool { return p.value.True() }

func (p *undetermined) String() (s string) {
        s = p.identifier.String()
        s += p.tok.String()
        s += p.value.String()
        return
}

func (p *undetermined) Strval() (s string, err error) {
        s, err = p.value.Strval()
        return
}

func (p *undetermined) Float() (float64, error) { return 0, nil }
func (p *undetermined) Integer() (int64, error) { return 0, nil }

func (p *undetermined) cmp(v Value) (res cmpres) {
        if v.Type() == UndeterminedType {
                a, ok := v.(*undetermined)
                assert(ok, "value is not undetermined")
                if p.identifier.cmp(a.identifier) == cmpEqual {
                        if p.value.cmp(a.value) == cmpEqual {
                                res = cmpEqual
                        }
                }
        }
        return
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        knownobject
        f BuiltinFunc
}
func (p *Builtin) Type() Type { return BuiltinType }
func (p *Builtin) True() bool { return p.f != nil }
func (p *Builtin) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Builtin) String() string { return fmt.Sprintf("%s", p.name) }
func (p *Builtin) Call(pos token.Position, a... Value) (Value, error) { return p.f(pos, a...) }
func (p *Builtin) cmp(v Value) (res cmpres) {
        if v.Type() == BuiltinType {
                a, ok := v.(*Builtin)
                assert(ok, "value is not Builtin")
                if /*p.f == a.f &&*/ p.name == a.name {
                        res = cmpEqual
                }
        }
        return
}

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
func (entry *RuleEntry) True() bool { return entry.target.True() }
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
        if file.dir == "" {
                file.dir = entry.OwnerProject().absPath
        }
        if path, ok := entry.target.(*Path); ok && path != nil {
                path.File = file
        }
        return
}

func (entry *RuleEntry) SetExplicitPath(path *Path) {
        if path.File != nil && path.File.dir == "" {
                path.File.dir = entry.OwnerProject().absPath
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
                var v Value
                if v, err = program.Execute(entry, a); err != nil {
                        break
                } else {
                        result = append(result, v)
                }
        }
        return
}

func (entry *RuleEntry) Get(name string) (Value, error) {
        switch name {
        case "class": return &String{entry.class.String()}, nil
        case "name": return &String{entry.Name()}, nil
        // case "prerequisites": ...
        }
        return nil, fmt.Errorf("no such entry property (%s)", name)
}

func (entry *RuleEntry) redecl(scope *Scope) {
        panic("RuleEntry.redecl not supported")
}

func (entry *RuleEntry) refs(v Value) bool {
        if entry.target.refs(v) { return true }
        
        // TODO: do more tests for this to see if we need to fallthrough
        return false // only check closured agaist target

        for _, prog := range entry.programs {
                for _, m := range prog.pipline {
                        for _, a := range m.args {
                                if a.refs(v) { return true }
                        }
                }
                for _, depend := range prog.depends {
                        if depend.refs(v) { return true }
                }
                for _, recipe := range prog.recipes {
                        if recipe.refs(v) { return true }
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
func (entry *RuleEntry) expand(w expandwhat) (res Value, err error) {
        var target Value
        if target, err = entry.target.expand(w); err != nil { return }
        if target != entry.target {
                // TODO: test if programs are needed to be disclosed??
                res = &RuleEntry{
                        entry.class, target, entry.programs, entry.Position,
                }
        } else {
                res = entry
        }
        return
}

func (entry *RuleEntry) dependcompare(c *comparer) (err error) {
        if trace_compare { defer compun(comptrace(c, entry)) }
        if enable_assertions { assert(c.target != entry, "self comparation") }
        return c.compareDepend(entry.target)
}

func (entry *RuleEntry) prepare(pc *preparer) (err error) {
        if trace_prepare { defer prepun(preptrace(pc, entry.target)) }

ForPrograms:
        for _, prog := range entry.programs {
                if false && prog == pc.program {
                        //err = fmt.Errorf("%v: depended on itself", entry.target)
                        //fmt.Fprintf(os.Stdout, "%s: %v\n", prog.position, err)
                        fmt.Fprintf(os.Stderr, "%s: %v: depended on itself\n", prog.position, entry.target)
                        //continue //break ForPrograms
                }
                if err = pc.execute(entry, prog); err == nil {
                        break ForPrograms
                } else if _, ok := err.(targetNotFoundError); ok {
                        break ForPrograms // Don't try other programs if it's undefined.
                }
        }
        return
}

func (entry *RuleEntry) cmp(v Value) (res cmpres) {
        if v.Type() == RuleEntryType {
                a, ok := v.(*RuleEntry)
                assert(ok, "value is not RuleEntry")
                if /*entry.class == a.class &&*/ entry.target.cmp(a.target) == cmpEqual {
                        if entry.OwnerProject() == a.OwnerProject() {
                                res = cmpEqual
                        }
                }
        }
        return
}

type PatternEntry struct {
        *RuleEntry
        Pattern Pattern
}

/*
func (p *PatternEntry) concrete(stem string) (entry *RuleEntry, err error) {
        if entry, err = p.Pattern.concrete(p.RuleEntry, stem); err == nil && entry != nil {
                // entry.creator = p
        }
        return
}
*/

type StemmedEntry struct {
        Patent *PatternEntry
        Stem string // stem string
        target string // source target matching the pattern
        file *File // source file matching the pattern
}

//func (ps *StemmedEntry) Strval() (string, error) {return ps.target, nil }
func (ps *StemmedEntry) String() (s string) {
        return fmt.Sprintf("StemmedEntry{%s,%s,%s,%s}", ps.Patent, ps.Stem, ps.target, ps.file)
}

func (ps *StemmedEntry) concrete(stem string) (entry *RuleEntry, err error) {
        entry = new(RuleEntry)

        // Copy the rule entry bits
        *entry = *ps.Patent.RuleEntry

        var name string
        if name, err = ps.Patent.Pattern.MakeString(stem); err != nil {
                return
        }

        if ps.file != nil {
                file := stat(name, ps.file.sub, ps.file.dir, nil)
                entry.target = file
                if enable_assertions {
                        assert(name == ps.file.name, "'%s' stemmed name is wrong (!= %s)", name, ps.file.name)
                        assert(file.filebase == ps.file.filebase, "'%v' stemmed file is wrong (!= %v)", file, ps.file)
                }
        } else {
                if enable_assertions {
                        assert(ps.target == "", "no stemmed target name")
                        assert(name == ps.target, "'%s' stemmed name is wrong (!= %s)", name, ps.target)
                }

                var project = mostDerived()
                if project.isFileName(name) {
                        var file = project.search(name)
                        if file == nil { // still stat non-existed file
                                file = stat(name, "", project.absPath, nil)
                        }
                        assert(file != nil, "`%s` nil file", name)
                        entry.target = file
                } else {
                        entry.target = &String{ name }
                }
        }
        return
}

func (ps *StemmedEntry) prepare(pc *preparer) (err error) {
        if trace_prepare { defer prepun(preptrace(pc, ps.Patent)) }

        var stems = []string{ ps.Stem }
        var sources = []string{ ps.target }
        var entry *RuleEntry
        if ps.file != nil {
                sources = append(sources, ps.file.name)
        }

        // Find all useful stems.
ForSources:
        for _, source := range sources {
                var ( stem string; ok bool )
                if source == "" { continue }
                if ok, stem, err = ps.Patent.Pattern.match(source); ok && stem != "" {
                        for _, s := range stems { if s == stem { continue ForSources } }
                        stems = append(stems, stem)
                }
        }

        // Recover pc.stem when done.
        defer func(s string) { pc.stem = s } (pc.stem)

        // Try preparing target with all stems.
ForStems:
        for _, stem := range stems {
                if entry, err = ps.concrete(stem); err != nil {
                        return
                }

                pc.stem = stem // set for the current stem.
                if err = entry.prepare(pc); err == nil {
                        break ForStems // Good!
                } else if ute, ok := err.(targetNotFoundError); ok {
                        fmt.Printf("FIXME: stemmed unknown target %v\n", ute.target)
                } else if ufe, ok := err.(fileNotFoundError); ok {
                        fmt.Printf("FIXME: stemmed unknown file %v\n", ufe.file)
                }
        }
        return
}
