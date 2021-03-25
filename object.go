//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        "runtime/debug"
        "os/exec"
        "strings"
        "strconv"
        "bytes"
        "time"
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

  tryTraverse(t *traversal) (okay bool, err error)
}

type trivialobject struct { // generally unnamed objects
        trivial
        scope *Scope
        owner *Project
}
func (p *trivialobject) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *trivialobject) Name() string { panic("inquiring name of an unknown object") }
func (p *trivialobject) DeclScope() *Scope { return p.scope }
func (p *trivialobject) OwnerProject() *Project { return p.owner }
func (p *trivialobject) Strval() (string, error) { return fmt.Sprintf("{unknown %p}", p), nil }
func (p *trivialobject) String() string { return fmt.Sprintf("{unknown %p}", p) }
func (p *trivialobject) Get(name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p *trivialobject) redecl(scope *Scope) { panic("redeclaring unknown object") }
func (p *trivialobject) exists() existence { return existenceMatterless }
func (p *trivialobject) tryTraverse(t *traversal) (okay bool, err error) { return false, nil }
func (p *trivialobject) cmp(v Value) (res cmpres) {
        if a, ok := v.(*trivialobject); ok {
                assert(ok, "value is not trivialobject")
                if p.owner == a.owner && p.scope == a.scope {
                        res = cmpEqual
                }
        }
        return
}

type knownobject struct { // generally named objects
        trivialobject
        name string
}
func (p *knownobject) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *knownobject) True() (bool, error) { return true, nil }
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
        if a, ok := v.(*knownobject); ok {
                assert(ok, "value is not knownobject")
                if p.owner == a.owner && p.scope == a.scope && p.name == a.name {
                        res = cmpEqual
                }
        }
        return
}

type unresolvedobject struct { // named callable/executable objects
        trivialobject
        name Value // name could be closured
}
func (p *unresolvedobject) traverse(t *traversal) (err error) { return }
func (p *unresolvedobject) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *unresolvedobject) True() (bool, error) { return false, nil }
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
        if a, ok := v.(*unresolvedobject); ok {
                assert(ok, "value is not unresolvedobject")
                if p.owner == a.owner && p.scope == a.scope {
                        res = p.name.cmp(a.name)
                }
        }
        return
}

func unresolved(p *Project, v Value) *unresolvedobject {
        return &unresolvedobject{trivialobject{ scope: p.scope, owner: p }, v}
}

type ProjectName struct {
        knownobject
        project *Project
}

func (p *ProjectName) expand(_ expandwhat) (Value, error) { return p, nil }

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (p *ProjectName) True() (bool, error) { return p.project != nil, nil }
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
        if p.project != nil { value, err = p.project.resolveObject(name) }
        return
}

// Call a ProjectName returns the project name.
func (p *ProjectName) Call(pos Position, a... Value) (value Value, err error) {
        if p.project != nil {
                value = &String{trivial{pos},p.project.name}
        }
        return
}
func (p *ProjectName) traverse(t *traversal) (err error) {
        _, err = p.tryTraverse(t)
        return
}
func (p *ProjectName) tryTraverse(t *traversal) (okay bool, err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        var entry = p.project.DefaultEntry()
        if entry != nil && entry.class != UseRuleEntry {
                okay, err = true, entry.traverse(t)
        }
        return
}
func (p *ProjectName) mod(t *traversal) (res time.Time, err error) {
        if p.project != nil {
                var defent = p.project.DefaultEntry()
                if defent != nil && defent.class != UseRuleEntry {
                        res, err = defent.mod(t)
                }
        }
        return
}
func (p *ProjectName) cmp(v Value) (res cmpres) {
        if a, ok := v.(*ProjectName); ok {
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
func (n *ScopeName) True() (bool, error) { return n.scope != nil, nil }
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
        if a, ok := v.(*ScopeName); ok {
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

  DefAuto // automatic: $1, $2, $3, etc.
  DefArg  // ((arg))
  DefDecl //
  DefConfDir
  DefConfRef // referred by config
  DefConfig

  defany // referred any def
)

func (o DefOrigin) String() (s string) {
        switch o {
        case DefDefault: s = "Default"
        case DefSimple:  s = "Simple"
        case DefExpand:  s = "Expand"
        case DefExecute: s = "Execute"
        case DefAuto:    s = "Auto"
        case DefArg:     s = "Arg"
        case DefDecl:    s = "Decl"
        case DefConfDir: s = "ConfDir"
        default: s = fmt.Sprintf("Origin<%d>", o)
        }
        return
}

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type Def struct {
        knownobject
        origin DefOrigin
        value Value
}

func (d *Def) refs(v Value) bool { return d == v || (d.value != nil && d.value.refs(v)) }
func (d *Def) closured() bool { return d.value.closured() }
func (d *Def) expand(w expandwhat) (res Value, err error) {
        if res = d; d.value != nil {
                var value Value
                if value, err = d.value.expand(w); err != nil {
                        res = nil ; return
                } else if value != d.value {
                        if w&expandCaller != 0 {
                                res = value
                        } else {
                                res = &Def{ d.knownobject, d.origin, value }
                        }
                } else if w&expandCaller != 0 {
                        res = d.value
                }
        } else if w&expandCaller != 0 {
                res = nil
        }
        return
}
func (d *Def) exists() existence { return d.value.exists() }
func (d *Def) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Def); ok && d.value != nil {
                assert(ok, "value is not Def")
                if a.value != nil {
                        res = d.value.cmp(a.value)
                }
        }
        return
}

func (d *Def) True() (res bool, err error) {
        if d.value != nil {
                res, err = d.value.True()
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
        default:         s += " = "
        }
        if d.value != nil {
                s += elementString(d, d.value, 0)
        } else {
                s += "<nil>"
        }
        return
}
func (d *Def) Strval() (s string, e error) {
        if d.value != nil { s, e = d.value.Strval() }
        return
}

func (d *Def) setval(value Value) (err error) { return d.set(d.origin, value) }
func (d *Def) set(origin DefOrigin, value Value) (err error) {
        if origin != DefSimple && !isNil(value) && value.refs(d) {
                var pos = d.position
                if !pos.IsValid() && d.value != nil { pos = d.value.Position() }
                err = errorf(pos, "value refers `%s`: %v (%T)", d.name, value, value)
                if optionVerbose { fmt.Fprintf(stderr, "%v\n", err) }
                if optionPrintStack { debug.PrintStack() }
                return
        } else if origin != DefExecute && isNil(value) {
                value = &None{trivial{d.position}}
        }

        switch d.origin = origin; origin {
        case DefSimple: // Eval expands delegates in the value.
                if d.value, err = value.expand(expandDelegate); err != nil { return }
        case DefExpand:
                if d.value, err = value.expand(expandAll); err != nil { return }
        case DefExecute:
                var ( stdout, stderr bytes.Buffer; s string )
                if isNone(value) || isNil(value) { d.value = nil } else
                if s, err = value.Strval(); err == nil {
                        sh := exec.Command("sh", "-c", s)
                        sh.Stdout, sh.Stderr = &stdout, &stderr

                        var pos = value.Position()
                        if !pos.IsValid() { pos = d.Position() }
                        if err = sh.Run(); err != nil { value = &None{trivial{pos}} } else {
                                value = &String{trivial{pos},strings.TrimSpace(stdout.String())}
                        }
                        stdout.Reset()
                        stderr.Reset()
                        d.value = value
                } else {
                        var pos = value.Position()
                        if !pos.IsValid() { pos = d.Position() }
                        d.value = &None{trivial{pos}}
                }
        default: // DefDefault, DefArg: keeps delegates and closures.
                d.value = value
        }
        return
}

func (d *Def) append(va... Value) (err error) {
        for _, value := range va {
                if value != nil && value.refs(d) {
                        err = fmt.Errorf("append recursive variable `%s` (from %v)", d.name, d.OwnerProject())
                        if true || optionVerbose {
                                fmt.Fprintf(stderr, "error: %v\n", err)
                                debug.PrintStack()
                        }
                        return
                }
        }

        var list *List
        if num := len(va); num == 0 {
                // Does nothing...
        } else if isNone(d.value) || isNil(d.value) {
                list = &List{elements{ merge(va...) }}
        } else if list, _ = d.value.(*List); list != nil {
                list.Append(merge(va...)...)
        } else {
                elems := []Value{ d.value }
                elems = append(elems, merge(va...)...)
                list = &List{elements{ elems }}
        }
        if list != nil {
                if d.origin == DefExecute {
                        err = d.set(DefDefault, list)
                } else {
                        err = d.setval(list)
                }
        }
        return
}

func (d *Def) Call(pos Position, a... Value) (res Value, err error) {
        if isNil(d.value) { return }
        switch d.origin {
        case DefDefault:
                var ( defs []*Def; vals []Value )
                defer func() { for i, d := range defs { d.value = vals[i] }} ()
                for i := 0; i < len(a) && i < maxNumVarVal; i += 1 {
                        var def = context.globe.scope.Lookup(strconv.Itoa(i)).(*Def)
                        vals = append(vals, def.value)
                        defs = append(defs, def)
                        def.value = a[i]
                }
                res, err = d.value.expand(expandClosure|expandDelegate)
        default: // DefArg, DefSimple, DefExpand, DefExecute:
                res = d.value
        }
        if res == nil {
                // ...
        } else if list, ok := res.(*List); ok {
                if n := len(list.Elems); n == 0 {
                        res = &None{trivial{d.position}}
                } else if n == 1 {
                        res = list.Elems[0] 
                }
        }
        return
}

func (d *Def) DiscloseValue() (res Value, err error) {
        if d.value != nil {
                if res, err = d.value.expand(expandClosure); err != nil { return }
                if res == nil { res = d.value }
        }
        return
}

func (d *Def) Get(name string) (Value, error) {
        switch name {
        case "name": return &String{trivial{d.Position()},d.name}, nil
        case "value": return d.value, nil
        }
        return nil, fmt.Errorf("no such property `%s' (Def)", name)
}
func (d *Def) traverse(t *traversal) (err error) {
        if d.value != nil { err = d.value.traverse(t) }
        return
}
func (d *Def) mod(t *traversal) (res time.Time, err error) {
        if d.value != nil { res, err = d.value.mod(t) }
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
func (p *undetermined) refdef(origin DefOrigin) bool { return false }
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
func (p *undetermined) traverse(t *traversal) (err error) { return }
func (p *undetermined) Position() Position { return p.identifier.Position() }
func (p *undetermined) stamp(t *traversal) (files []*File, err error) { return }
func (p *undetermined) exists() existence { return existenceMatterless }
func (p *undetermined) True() (bool, error) { return false, nil }
func (p *undetermined) String() (s string) {
        s = p.identifier.String()
        s += p.tok.String()
        s += p.value.String()
        return
}
func (p *undetermined) Strval() (string, error) { return p.value.Strval() }
func (p *undetermined) Float() (float64, error) { return 0, nil }
func (p *undetermined) Integer() (int64, error) { return 0, nil }
func (p *undetermined) mod(t *traversal) (res time.Time, err error) { return }
func (p *undetermined) cmp(v Value) (res cmpres) {
        if a, ok := v.(*undetermined); ok {
                assert(ok, "value is not undetermined")
                if p.identifier.cmp(a.identifier) == cmpEqual {
                        if p.value.cmp(a.value) == cmpEqual {
                                res = cmpEqual
                        }
                }
        }
        return
}

type builtinFlag uint32
const (
        builtinFunction builtinFlag = 1<<iota
        builtinCommand
)

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        knownobject
        flag builtinFlag
        f BuiltinFunc
}
func (p *Builtin) True() (bool, error) { return p.f != nil, nil }
func (p *Builtin) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Builtin) String() string { return fmt.Sprintf("%s", p.name) }
func (p *Builtin) Call(pos Position, a... Value) (Value, error) { return p.f(pos, a...) }
func (p *Builtin) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Builtin); ok {
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
        position Position
}

func (entry *RuleEntry) Position() Position { return entry.position/*entry.target.Position()*/ }
func (entry *RuleEntry) stamp(t *traversal) (files []*File, err error) { return entry.target.stamp(t) }
func (entry *RuleEntry) exists() existence { return entry.target.exists() }
func (entry *RuleEntry) True() (bool, error) { return entry.target.True() }
func (entry *RuleEntry) Float() (float64, error) { return 0, nil }
func (entry *RuleEntry) Integer() (int64, error) { return 0, nil }
func (entry *RuleEntry) OwnerProject() *Project { return entry.programs[0].project }
func (entry *RuleEntry) DeclScope() *Scope { return entry.OwnerProject().scope }
func (entry *RuleEntry) Name() string {
        if entry == nil {
                panic("entry is nil")
        } else if entry.target == nil {
                fmt.Fprintf(stderr, "%v: nil target\n", entry.position)
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
        if p, ok := entry.target.(*File); ok && p != nil { return true }
        if p, ok := entry.target.(*Path); ok && p != nil /*&& p.File != nil*/ {
                return true
        }
        return false
}
func (entry *RuleEntry) SetExplicitFile(file *File) {
        if file.dir == "" {
                file.dir = entry.OwnerProject().absPath
        }
        if path, ok := entry.target.(*Path); ok && path != nil {
                //path.File = file
        }
        return
}
// func (entry *RuleEntry) SetExplicitPath(path *Path) {
//         /*if path.File != nil && path.File.dir == "" {
//                 path.File.dir = entry.OwnerProject().absPath
//         }*/
//         //if path, ok := entry.target.(*Path); ok && path != nil {
//         //        path
//         //}
//         return
// }
// RuleEntry.Execute executes the rule program only if the target is outdated.
func (entry *RuleEntry) Execute(pos Position, a... Value) (result []Value, err error) {
        switch entry.class {
        case PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                err = errorf(pos, "executing pattern entry '%v'", entry.target)
                return
        }

        var caller *traversal
        ForPrograms: for _, program := range entry.programs {
                var ( val Value ; e error )
                if val, e = program.execute(caller, entry, a); e == nil {
                        result = append(result, val)
                        continue ForPrograms
                }

                var brks, errs = extractBreakers(e)
                if len(errs) > 0 {
                        err = wrap(program.position, errs...)
                        break
                }
                
                for _, brk := range brks {
                        switch brk.what {
                        case breakNext: continue ForPrograms
                        case breakCase, breakDone: break ForPrograms
                        default: err = wrap(program.position, brk, err)
                        }
                }
        }
        if err != nil { err = wrap(pos, wrap(entry.position, err)) }
        return
}
func (entry *RuleEntry) Get(name string) (Value, error) {
        switch name {
        case "class": return &String{trivial{entry.position},entry.class.String()}, nil
        case "name": return &String{trivial{entry.position},entry.Name()}, nil
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
                /*for _, m := range prog.pipline {
                        for _, a := range m.args {
                                if a.refs(v) { return true }
                        }
                }*/
                for _, depend := range prog.depends {
                        if depend.refs(v) { return true }
                }
                for _, recipe := range prog.recipes {
                        if recipe.refs(v) { return true }
                }
        }
        return false
}
func (entry *RuleEntry) refdef(origin DefOrigin) bool {
        return entry.target.refdef(origin)
}
func (entry *RuleEntry) closured() bool {
        if entry.target.closured() { return true }

        // TODO: do more tests for this to see if we need to fallthrough
        return false // only check closured agaist target

        for _, prog := range entry.programs {
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
        if entry == nil {
                // happens from some &{xxx} exprs
                err = fmt.Errorf("entry is nil")
                return
        }

        var target Value
        if target, err = entry.target.expand(w); err != nil { return }
        if target != entry.target {
                // TODO: test if programs are needed to be disclosed??
                res = &RuleEntry{
                        entry.class,
                        target,
                        entry.programs,
                        entry.position,
                }
        } else {
                res = entry
        }
        return
}
func (entry *RuleEntry) tryTraverse(t *traversal) (okay bool, err error) {
        if err = entry.traverse(t); err == nil { okay = true }
        return
}
func (entry *RuleEntry) traverse(t *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(t, entry.target)) }
        if optionEnableBenchmarks && false { defer bench(mark("RuleEntry.traverse")) }
        if optionEnableBenchspots { defer bench(spot("RuleEntry.traverse")) }
        ForPrograms: for _, prog := range entry.programs {
                var _, e = prog.execute(t, entry, t.arguments)
                if e == nil { continue }

                var brks, errs = extractBreakers(e)
                if len(errs) > 0 {
                        err = wrap(prog.position, errs...)
                        break
                }

                for _, brk := range brks {
                        switch brk.what {
                        case breakNext: continue ForPrograms
                        case breakCase, breakDone: break ForPrograms
                        default: err = wrap(prog.position, brk, err)
                        }
                }
        }
        if err != nil { err = wrap(entry.position, err) }
        return
}
func (entry *RuleEntry) mod(t *traversal) (time.Time, error) {
        // FIXME: entry.target maybe not the real target
        return entry.target.mod(t)
}
func (entry *RuleEntry) cmp(v Value) (res cmpres) {
        if a, ok := v.(*RuleEntry); ok {
                assert(ok, "value is not RuleEntry")
                if /*entry.class == a.class &&*/ entry.target.cmp(a.target) == cmpEqual {
                        if entry.OwnerProject() == a.OwnerProject() {
                                res = cmpEqual
                        }
                }
        }
        return
}

func (entry *RuleEntry) option() (res bool, infos []Value) {
        ForProgram: for _, program := range entry.programs {
                if !program.configure { continue }
                for _, depend := range program.depends {
                        g, ok := depend.(*modifiergroup)
                        if!ok { continue }
                        for _, m := range g.modifiers {
                                s, e := m.name.Strval()
                                if e != nil || s != "configure" { continue }
                                for _, arg := range m.args {
                                        a, ok := arg.(*Argumented)
                                        if!ok { continue }
                                        f, ok := a.value.(*Flag)
                                        if!ok { continue }
                                        s, e := f.name.Strval()
                                        if e != nil { continue }
                                        if s != "option" { continue }
                                        for _, v := range a.args {
                                                if p, ok := v.(*Pair); ok {
                                                        s, _ := p.Key.Strval()
                                                        if s != "info" { continue }
                                                        v = p.Value
                                                }
                                                infos = append(infos, v)
                                        }
                                        res = true
                                        break ForProgram
                                }
                        }
                }
        }
        return
}

type PatternEntry struct {
        Pattern Pattern
        *RuleEntry
}
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
        if a, ok := v.(*PatternEntry); ok {
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
}
func (p *StemmedEntry) expand(w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.PatternEntry.expand(w); err != nil {
                return
        } else if v != p.PatternEntry {
                res = &StemmedEntry{v.(*PatternEntry),p.Stems}
        }
        return
}
func (p *StemmedEntry) cmp(v Value) (res cmpres) {
        if a, ok := v.(*StemmedEntry); ok {
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
func (p *StemmedEntry) traverse(t *traversal) (err error) {
        return errorf(p.position, "cant traverse stemmed entry directly")
}
func (p *StemmedEntry) _target(t *traversal, target string) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("StemmedEntry.traverse(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("StemmedEntry.traverse")) }

        defer func(a Value) { p.target = a } (p.target)
        defer func(stems []string) { t.stems = stems } (t.stems)
        t.stems = p.Stems // set stems for the traversal

        if _, ok := p.Pattern.(*Path); ok {
                p.target = MakePathStr(p.position, target)
        } else if file := t.project.matchFile(target); file != nil {
                p.target = file
        } else {
                p.target = &String{trivial{p.position}, target}
        }

        if false { fmt.Fprintf(stderr, "%s:stemmed: %T %v -> %v, stems=%v\n", p.position, p.Pattern, p.Pattern, target, t.stems) }

        if err = p.RuleEntry.traverse(t); err != nil { err = wrap(p.position, err) }
        return
}
func (p *StemmedEntry) file(t *traversal, file *File) (err error) {
        if optionTraceTraversal { defer un(tt(t, p)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("StemmedEntry.file(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("StemmedEntry.file")) }

        defer func(a Value) { p.target = a } (p.target)
        defer func(stems []string) { t.stems = stems } (t.stems)
        t.stems = p.Stems // set stems for the traversal
        p.target = file

        if file.info == nil && file.match == nil { // !isAbsOrRel()
                if f := t.project.matchFile(file.name); f != nil { *file = *f }
                if file.info == nil { file.info, _ = os.Stat(file.name) }
        }

        if err = p.RuleEntry.traverse(t); err != nil { err = wrap(p.position, err) }
        return
}
