//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "os/exec"
        "strings"
        "strconv"
        "bytes"
        "fmt"
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
func (p *unresolvedobject) Call(pos Position, a... Value) (result Value, err error) { result = p; return }
func (p *unresolvedobject) Execute(pos Position, a... Value) (result []Value, err error) { result = []Value{p}; return }
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
func (p *ProjectName) Call(pos Position, a... Value) (value Value, err error) {
        if p.project != nil {
                value = &String{ p.project.name }
        }
        return
}

func (p *ProjectName) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }

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
                if value, err = d.Value.expand(w); err != nil {
                        res = nil ; return
                } else if value != d.Value {
                        if w&expandCaller != 0 {
                                res = value
                        } else {
                                res = &Def{ d.knownobject, d.origin, value }
                        }
                } else if w&expandCaller != 0 {
                        res = d.Value
                }
        } else if w&expandCaller != 0 {
                res = nil
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
func (d *Def) True() (res bool) {
        if d.Value != nil {
                res = d.Value.True()
        }
        return
}
func (d *Def) elemstr(o Object, k elemkind) (s string) {
        if o != nil {
                if p := d.OwnerProject(); p != o.OwnerProject() {
                        return fmt.Sprintf("$(%s->%s)", p.name, d.name)
                }
        }
        s = fmt.Sprintf(`$(%s)`, d.name)
        return
}
func (d *Def) String() (s string) {
        switch s = d.name; d.origin {
        case DefDefault: s += "="
        case DefSimple:  s += ":="
        case DefExpand:  s += "::="
        case DefExecute: s += "!="
        default: s += " = "
        }
        if d.Value != nil {
                s += elementString(d, d.Value, 0)
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
        if origin != DefSimple && value != nil && value.refs(d) {
                err = fmt.Errorf("self recursive variable `%s`", d.name)
                return
        } else if origin != DefExecute && value == nil {
                value = universalnone
        }

        switch d.origin = origin; origin {
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

func (d *Def) Call(pos Position, a... Value) (res Value, err error) {
        switch d.origin {
        case DefSimple, DefExpand, DefExecute:
                res = d.Value
        case DefDefault:
                // TODO: parameterization, e.g. $1, $2, $3, $4, $5 (see foreach)
                for i := 0; i < len(a) && i < maxNumVarVal; i += 1 {
                        var def = context.globe.scope.Lookup(strconv.Itoa(i)).(*Def)
                        defer func(v Value) { def.Value = v } (def.Value)
                        def.Value = a[i]
                }
                res, err = d.Value.expand(expandClosure|expandDelegate)
        default:
                unreachable()
        }
        if res != nil && res.Type() == ListType {
                var list = res.(*List)
                if n := len(list.Elems); n == 0 {
                        res = universalnone
                } else if n == 1 {
                        res = list.Elems[0] 
                }
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
        if optionTraceCompare { defer compun(comptrace(c, d))}
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
func (p *Builtin) Call(pos Position, a... Value) (Value, error) { return p.f(pos, a...) }
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
        PercRuleEntry
        GlobRuleEntry
        RegexpRuleEntry
        PathPattRuleEntry
        UseRuleEntry
)

var namesForRuleEntryClass = []string{
        GeneralRuleEntry:  "GeneralRuleEntry",
        PercRuleEntry:     "PercRuleEntry",
        GlobRuleEntry:     "GlobRuleEntry",
        RegexpRuleEntry:   "RegexpRuleEntry",
        PathPattRuleEntry: "PathPattRuleEntry",
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
        Position Position
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
                fmt.Fprintf(stderr, "%v: nil target\n", entry.Position)
                panic("entry target is nil")
        }
        s, err := entry.target.Strval()
        if err != nil { panic(err) } // FIXME: error
        return s
}
func (entry *RuleEntry) Strval() (string, error) { return entry.target.Strval() }
func (entry *RuleEntry) String() string { return entry.target.String() }
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
func (entry *RuleEntry) Execute(pos Position, a... Value) (result []Value, err error) {
        switch entry.class {
        case PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                return nil, fmt.Errorf("%s: executing pattern entry '%s'.", pos, entry.Name())
        }

        // The position where (assert) failed.
        var failed token.Position

ForPrograms:
        for _, program := range entry.programs {
                var ( val Value ; e error )
                if val, e = program.Execute(entry, a); e == nil {
                        result = append(result, val)
                        continue ForPrograms
                } else if br, ok := e.(*breaker); ok {
                        switch br.what {
                        case breakFail: // (assert) failure
                                failed = token.Position(br.pos)//program.position
                                err = scanner.WrapErrors(failed, e, err)
                                continue ForPrograms // break ForPrograms
                        case breakNext: // continue with the next (case)
                                continue ForPrograms
                        case breakCase, breakDone:
                                // finish (case) or (cond) peacefully
                                break ForPrograms
                        }
                }
                var pp = program.position
                err = scanner.WrapErrors(token.Position(pp), e, err)
        }
        if err != nil {
                err = scanner.WrapErrors(token.Position(pos), err)
        } else if failed.IsValid() {
                //err = scanner.Errorf(failed, "assertion failed")
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

func (entry *RuleEntry) recipes() (recipes []Value) {
        for _, prog := range entry.programs {
                for _, recipe := range prog.recipes {
                        recipes = append(recipes, recipe)
                }
        }
        return
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
        if optionTraceCompare { defer compun(comptrace(c, entry)) }
        if enable_assertions { assert(c.target != entry, "self comparation") }
        return c.compareDepend(entry.target)
}

func (entry *RuleEntry) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, entry.target)) }
        var failed bool
ForPrograms:
        for _, prog := range entry.programs {
                if err = pc.execute(entry, prog); err == nil {
                        // continue with the next program
                } else if _, ok := err.(targetNotFoundError); ok {
                        break ForPrograms // Don't try other programs if it's undefined.
                } else if br, ok := err.(*breaker); ok {
                        switch br.what {
                        case breakFail:
                                fmt.Fprintf(stderr, "%s: entry.prepare: %s\n", br.pos, br.message)
                                failed, err = true, nil
                        case breakNext: continue ForPrograms
                        case breakCase, breakDone:
                                break ForPrograms
                        }
                }
        }
        if err == nil && failed {
                err = scanner.Errorf(token.Position(entry.Position), "assertion failed")
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
        Pattern Pattern
        *RuleEntry
}

func (p *PatternEntry) Type() Type { return PatternEntryType }
func (p *PatternEntry) expand(w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.RuleEntry.expand(w); err != nil {
                return
        } else if v != p.RuleEntry {
                res = &PatternEntry{p.Pattern, v.(*RuleEntry)}
        }
        return
}
func (p *PatternEntry) cmp(v Value) (res cmpres) {
        if v.Type() == PatternEntryType {
                a, ok := v.(*PatternEntry)
                assert(ok, "value is not PatternEntry")
                // FIXME: p.Pattern.cmp(p.Pattern)
                if p.RuleEntry.cmp(a.RuleEntry) == cmpEqual {
                        res = cmpEqual
                }
        }
        return
}

type StemmedEntry struct {
        *PatternEntry
        Stems []string // stem string
        target string // source target matching the pattern
        stub *filestub // source file matching the pattern
}

func (p *StemmedEntry) Type() Type { return StemmedEntryType }
func (p *StemmedEntry) expand(w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.PatternEntry.expand(w); err != nil {
                return
        } else if v != p.PatternEntry {
                res = &StemmedEntry{v.(*PatternEntry),p.Stems,p.target,p.stub}
        }
        return
}
func (p *StemmedEntry) cmp(v Value) (res cmpres) {
        if v.Type() == PatternEntryType {
                a, ok := v.(*StemmedEntry)
                assert(ok, "value is not StemmedEntry")
                if len(p.Stems) != len(p.Stems) { return }
                for i, stem := range p.Stems {
                        if stem != a.Stems[i] { return }
                }
                res = p.PatternEntry.cmp(a.PatternEntry)
        }
        return
}

func (p *StemmedEntry) String() (s string) {
        return fmt.Sprintf("<%s,%s>", p.PatternEntry, p.Stems)
}

func (p *StemmedEntry) concrete(pc *preparer, stems []string) (entry *RuleEntry, err error) {
        entry = new(RuleEntry)
        *entry = *p.RuleEntry // copy RuleEntry bits

        var name string
        var rest []string
        name, rest, err = p.Pattern.stencil(stems)
        if err != nil { return }
        if len(rest) > 0 { panic("FIXME: unhandled stems") }

        if p.stub != nil {
                file := stat(name, p.stub.sub, p.stub.dir, nil)
                entry.target = file
                if enable_assertions {
                        //assert(name == p.stub.name, "'%s' stemmed name is wrong (!= %s)", name, p.stub.name)
                        //assert(file.filebase == p.file.filebase, "'%v' stemmed file is wrong (!= %v)", file, p.file)
                }
        } else {
                if enable_assertions && p.target != "" {
                        assert(name == p.target, "'%s' stemmed name is wrong (!= %s)", name, p.target)
                }

                var proj = pc.derived
                if proj == nil {
                        proj = pc.related[0]
                }

                if file := proj.matchFile(name); file != nil {
                        if enable_assertions {
                                assert(proj.isFileName(name), "`%s` is not file", name)
                        }
                        entry.target = file
                } else {
                        if enable_assertions {
                                assert(!proj.isFileName(name), "`%s` is file", name)
                        }
                        entry.target = &String{ name }
                }
        }
        return
}

func (p *StemmedEntry) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }

        var names = []string{ p.target }
        if p.stub != nil {
                names = append(names, p.stub.name)
        }

        // Find all useful stems.
        var stemsList = [][]string{ p.Stems }
ForNames:
        for _, source := range names {
                var ( s string ; ss []string )
                if source == "" { continue }
                s, ss, err = p.Pattern.match(source)
                if s != "" && ss != nil {
                ForStemsList:
                        for _, s := range stemsList {
                                if len(s) != len(ss) {
                                        continue ForStemsList
                                }
                                for i, t := range s {
                                        if ss[i] != t {
                                                continue ForStemsList
                                        }
                                }
                                continue ForNames
                        }
                        stemsList = append(stemsList, ss)
                }
        }

        // Recover pc.stem when done.
        defer func(ss []string) { pc.stems = ss } (pc.stems)

        // Try preparing target with all stems.
ForStems:
        for _, stems := range stemsList {
                var entry *RuleEntry
                if entry, err = p.concrete(pc, stems); err != nil {
                        return
                }

                pc.stems = stems // set current stem string
                if err = entry.prepare(pc); err == nil {
                        break ForStems // Good!
                } else if ute, ok := err.(targetNotFoundError); ok {
                        if optionTracePrepare { pc.trace("stemmed: unknown target:", ute.target) }
                } else if ufe, ok := err.(fileNotFoundError); ok {
                        if optionTracePrepare { pc.trace("stemmed: unknown file:", ufe.file) }
                }
        }
        return
}
