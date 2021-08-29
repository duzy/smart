//
//  Copyright (C) 2012-2018, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/token"
        //"runtime/debug"
        "os/exec"
        "strings"
        "strconv"
        "sync"
        "bytes"
        "time"
        "fmt"
        //"os"
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

type objbase struct { // generally unnamed objects
        valbase
        scope *Scope
        owner *Project
}
func (p *objbase) Name() string { panic("inquiring name of an unknown object") }
func (p *objbase) DeclScope() *Scope { return p.scope }
func (p *objbase) OwnerProject() *Project { return p.owner }
func (p *objbase) Strval() (string, error) { return fmt.Sprintf("{unknown %p}", p), nil }
func (p *objbase) String() string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) Get(name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p *objbase) redecl(scope *Scope) { panic("redeclaring unknown object") }
func (p *objbase) exists() existence { return existenceMatterless }
func (p *objbase) cmp(v Value) (res cmpres) {
        if a, ok := v.(*objbase); ok {
                assert(ok, "value is not objbase")
                if p.owner == a.owner && p.scope == a.scope {
                        res = cmpEqual
                }
        }
        return
}

type knownobject struct { // generally named objects
        objbase
        name string // single, or group name if containing '(*)' and corresponding members
        //members [][]string
}
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
        objbase
        name Value // name could be closured
}
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
func (p *unresolvedobject) Call(pos Position, a... Value) (result Value) { result = p; return }
func (p *unresolvedobject) Execute(pos Position, a... Value) (result []Value, err error) { return []Value{p}, nil }
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
func (p *unresolvedobject) traverse(t *traversal) { }
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
        return &unresolvedobject{objbase{ scope: p.scope, owner: p }, v}
}

type ProjectName struct {
        knownobject
        project *Project
}

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
func (p *ProjectName) Call(pos Position, a... Value) (value Value) {
        if p.project != nil {
                value = MakeString(pos, p.project.name)
        }
        return
}
func (p *ProjectName) traverse(t *traversal) {
        if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
        if entry := p.project.DefaultEntry(); entry == nil {
                // does nothing
        } else if entry.class != UseRuleEntry {
                entry.traverse(t)
        }
}
func (p *ProjectName) stat(t *traversal) (si *statinfo) {
        if p.project != nil {
                if defent := p.project.DefaultEntry(); defent == nil {
                        // does nothing
                } else if defent.class != UseRuleEntry {
                        si = defent.stat(t)
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

type Origin int
const (
        DefVoid Origin = iota // no initialization

        // =
        DefDefault // normal value

        // :=
        DefExpand1 // expand delegates (simple expand)

        // ::=
        DefExpand2 // expand all (delegates, closures, paths)

        // !=
        DefExecute  // value to be executed
        DefExecuted // executed result

        DefAuto // automatic: $1, $2, $3, etc.
        DefArg  // ((arg))
        DefDecl // declaration names
        DefConfDir
        DefConfRef // referred by config
        DefConfig

        defany // referred any def
)

func (o Origin) String() (s string) {
        switch o {
        case DefVoid:    s = "Void"
        case DefDefault: s = "Default"
        case DefExpand1: s = "Expand1"
        case DefExpand2: s = "Expand2"
        case DefExecute: s = "Execute"
        case DefAuto:    s = "Auto"
        case DefArg:     s = "Arg"
        case DefDecl:    s = "Decl"
        case DefConfDir: s = "ConfDir"
        case DefConfRef: s = "ConfRef"
        case DefConfig:  s = "Config"
        case defany:     s = "any"
        default: s = fmt.Sprintf("Origin<%d>", o)
        }
        return
}

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type Def struct {
        knownobject
        origin Origin
        value Value
}
func (d *Def) True() (res bool, err error) {
        if d.value != nil {
                res, err = d.value.True()
        }
        return
}
func (d *Def) String() (s string) {
        switch s = d.name; d.origin {
        case DefDefault: s += "="
        case DefExpand1: s += ":="
        case DefExpand2: s += "::="
        case DefExecute: s += "!="
        default:         s += "⇒"
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
func (d *Def) refs(v Value) bool { return d == v || (d.value != nil && d.value.refs(v)) }
func (d *Def) defs(s string) (res []*Def) {
        if d.name == s { return append(res, d) }
        return d.value.defs(s)
}
func (d *Def) expandible(w expandwhat) (res bool) {
        //if w&expandDef != 0 { return true }
        if !isNil(d.value) { res = d.value.expandible(w) }
        return
}
func (d *Def) expand(w expandwhat) (res Value, err error) {
        var value, value2 Value
        if isNil(d.value) || isNone(d.value) {
                // does nothing
        } else if value, err = d.value.expand(w); err != nil {
                diag.errorOf(d.value, "expand '%v' failed: %v", d.value, err).
                        debug(options.debugErrors, 1)
        } else if !isNil(value) && value != d.value && w&expandDef == 0 {
                res = &Def{ d.knownobject, d.origin, value }
        } else if w&expandDef == 0 {
                // does nothing
        } else if isNil(value) {
                res = d.value
        } else if value2, err = value.expand(w); err != nil {
                diag.errorOf(d.value, "expand '%v' failed: %v", value, err).
                        debug(options.debugErrors, 1)
        } else if !isNil(value2) && value2 != value {
                res = value2
        } else {
                res = value
        }
        return
}
func (d *Def) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Def); ok && d.value != nil {
                assert(ok, "value is not Def")
                if a.value != nil {
                        res = d.value.cmp(a.value)
                }
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
func (d *Def) isEmpty() bool { return isNone(d.value) || isNil(d.value) }
func (d *Def) val(value Value) (err error) { return d.set(d.origin, value) }
func (d *Def) set(origin Origin, value Value) (err error) {
        if origin != DefExpand1 && !isNil(value) && value.refs(d) {
                var pos = d.position
                if !pos.IsValid() && d.value != nil { pos = d.value.Position() }
                diag.errorAt(pos, "value refers to assigning Def '%s': %v (%T)", d.name, value, value)

                if options.verbose { diag.prompt("set %s (%v): %v\n", origin, d.name, value) }
                if options.debug { diag.infoAt(pos, "from here").debug(1) }
                return
        } else if origin != DefExecute && isNil(value) {
                value = MakeNone(d.position)
        }

        var elems []Value
        switch d.origin = origin; d.origin {
        case DefExpand1: // expands delegates
                if elems, _, err = expandall2(expandDelegate, value); err != nil {
                        diag.errorOf(value, "%v: expand value '%v' failed: %v", d.origin, value, err)
                        return
                } else { d.value = MakeListOrScalar(value.Position(), elems) }
        case DefExpand2: // expands delegates and closures
                if elems, _, err = expandall2(expandPlainValue/*|expandArgs*/, value); err != nil {
                        diag.errorOf(value, "%v: expand value '%v' failed: %v", d.origin, value, err)
                        return
                } else { d.value = MakeListOrScalar(value.Position(), elems) }
                /*
        case DefExecute:
                if err = d.execute(); err != nil {
                        diag.errorOf(value, "%v: stringify value '%v' failed: %v", d.origin, value, err).
                                debug(1)
                        return
                }*/
        default: // DefVoid, DefDefault, DefArg, etc.
                d.value = value
        }
        return
}

func (d *Def) append(va... Value) (err error) {
        for _, value := range va {
                if !isNil(value) && value.refs(d) {
                        err = fmt.Errorf("%v: append recursive variable '%s'", d.owner, d.name)
                        diag.infoAt(d.position, "%v", err).
                                debug(options.debugInfos)
                        return
                }
        }
        var list *List
        if num := len(va); num == 0 {
                return // Does nothing...
        } else if isNil(d.value) || isNone(d.value) {
                list = MakeList(d.position)
        } else if list, _ = d.value.(*List); list == nil {
                list = MakeList(d.position, d.value)
        }
        list.Append(merge(va...)...)
        return d.val(list)
}

func (d *Def) execute() (err error) {
        var (
                cmd string
                value = d.value
        )
        if d.origin != DefExecute {
                diag.errorAt(d.position, "%v: non-execute def: %v", d.origin, value).
                        debug(1)
        } else if isNil(value) || isNone(value) {
                d.value = nil
        } else if cmd, err = value.Strval(); err != nil {
                diag.errorAt(d.position, "%v: strval '%v' failed: %v", d.origin, value, err).
                        debug(1)
                d.value = MakeNone(d.position)
        } else if cmd == "" {
                diag.warnAt(d.position, "%v: empty command (value=%v)", d.origin, value).
                        debug(1)
                d.value = MakeNone(d.position)
        } else {
                // TODO: possibility to run command in the specified container
                var stdout, stderr bytes.Buffer
                var sh = exec.Command("sh", "-c", cmd)
                sh.Stdout, sh.Stderr = &stdout, &stderr
                if err = sh.Run(); err != nil {
                        diag.errorAt(d.position, "%v: execute command failed: %v", d.origin, err).
                                debug(1)
                        d.value = MakeNone(d.position)
                } else {
                        d.value = MakeString(d.position, strings.TrimSpace(stdout.String()))
                }
                stdout.Reset()
                stderr.Reset()
                d.origin = DefExecuted
        }
        return
}

func (d *Def) Call(pos Position, a... Value) (res Value) {
        switch d.origin {
        case DefDefault:
                var w = expandClosure|expandDelegate
                if isNil(d.value) || !d.value.expandible(w) {
                        res = d.value
                        break
                }

                var (
                        defs []*Def
                        vals []Value
                        err error
                )
                defer func() { for i, d := range defs { d.value = vals[i] }} ()
                for i := 0; i < len(a) && i < maxNumVarVal; i += 1 {
                        var def = context.globe.scope.Lookup(strconv.Itoa(i)).(*Def)
                        vals = append(vals, def.value)
                        defs = append(defs, def)
                        def.value = a[i]
                }
                if res, err = d.value.expand(w); err != nil {
                        diag.errorAt(pos, "expand def value failed: %v", err).
                                debug(1)
                } else if isNil(res) && !isNil(d.value) {
                        if d.value.expandible(w) {
                                diag.warnAt(pos, "expand '%v' incomplete (value=%v (%T))", d.name, d.value, d.value).
                                        debug(1)
                        } else { res = d.value }
                }
        case DefExecute:
                if err := d.execute(); err != nil {
                        diag.errorAt(pos, "execute '%s' failed: %v", d.name, err).
                                debug(1)
                        return
                } else {
                        res = d.value
                }
        default: // DefArg, DefExpand1, DefExpand2, DefExecuted, etc.
                res = d.value
        }
        if isNil(res) {
                // ...
        } else if list, ok := res.(*List); ok {
                if n := len(list.Elems); n == 0 {
                        res = MakeNone(d.position)
                } else if n == 1 {
                        res = list.Elems[0] 
                }
        }
        return
}

func (d *Def) DiscloseValue() (res Value, err error) {
        if d.value != nil {
                if res, err = d.value.expand(expandClosure); err != nil {
                        diag.errorAt(d.position, "expand '%v' failed: %v", d.value, err).
                                debug(1)
                        return
                } else if isNil(res) { res = d.value }
        }
        return
}

func (d *Def) Get(name string) (res Value, err error) {
        switch name {
        case "name" : res = MakeString(d.Position(), d.name)
        case "value": res = d.value
        default:
                err = fmt.Errorf("no such property `%s' (Def)", name)
                diag.errorAt(d.position, "%v", err).
                        debug(1)
        }
        return
}
func (d *Def) traverse(t *traversal) {
        if d.value != nil { d.value.traverse(t) }
}
func (d *Def) stat(t *traversal) (si *statinfo) {
        if d.value != nil { si = d.value.stat(t) }
        return
}

type undetermined struct {
        tok token.Token
        identifier Value
        value Value
}
func (p *undetermined) Position() Position { return p.identifier.Position() }
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
func (p *undetermined) refs(v Value) bool {
        return p.identifier.refs(v) || p.value.refs(v)
}
func (p *undetermined) defs(s string) (res []*Def) {
        return append(p.identifier.defs(s), p.value.defs(s)...)
}
func (p *undetermined) expandible(w expandwhat) bool {
        return p.identifier.expandible(w) || p.value.expandible(w)
}
func (p *undetermined) expand(w expandwhat) (res Value, err error) {
        var i, v Value
        if i, err = p.identifier.expand(w); err != nil {
                diag.errorOf(p.identifier, "expand '%v' failed: %v", p.identifier, err).
                        debug(1)
        } else if v, err = p.value.expand(w); err != nil {
                diag.errorOf(p.value, "expand '%v' failed: %v", p.value, err).
                        debug(1)
        } else if (!isNil(i) && i != p.identifier) || (!isNil(v) && v != p.value) {
                if isNil(i) { i = p.identifier }
                if isNil(v) { v = p.value }
                res = &undetermined{ p.tok, i, v }
        }
        return
}
func (p *undetermined) traverse(t *traversal) { }
func (p *undetermined) exists() existence { return existenceMatterless }
func (p *undetermined) stat(t *traversal) (si *statinfo) { return }
func (p *undetermined) stamp(t *traversal) (files []*File, err error) { return }
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
func (p *undetermined) patterned() bool { return false }
func (p *undetermined) match(i interface{}) (full bool, s string, stems []string) { return }
func (p *undetermined) stencil(stems []string) (s string, rest []string) { return }

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
func (p *Builtin) String() string { return fmt.Sprintf("%s", p.name) }
func (p *Builtin) Call(pos Position, a... Value) Value { return p.f(pos, a...) }
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

func (entry *RuleEntry) Position() (pos Position) {
        if pos = entry.position; !pos.IsValid() {
                if pos = entry.target.Position(); !pos.IsValid() {
                        pos = entry.programs[0].position
                }
        }
        return
}
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
func (entry *RuleEntry) Execute(pos Position, a... Value) (result []Value, breakers []*breaker) {
        switch entry.class {
        case PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                diag.errorAt(pos, "executing pattern entry '%v'", entry.target).
                        debug(1)
                return
        }

        var t = &traversal{
                start: time.Now(),
                visited: make(map[Value]int),
                group: new(sync.WaitGroup),
        }
        for _, program := range entry.programs {
                result = append(result, program.execute(t, entry, a))
                if brks := t.breakersOf(breakFail, breakErro); len(brks) > 0 {
                        t.breakers = t.breakersNot(breakFail, breakErro)
                        for _, brk := range brks {
                                switch brk.what {
                                case breakFail: diag.errorAt(program.position, "execution failed: %v", brk.message).
                                        debug(1)
                                case breakErro: diag.errorAt(program.position, "execution error: %v", brk.error).
                                        debug(1)
                                default: diag.errorAt(program.position, "breaker: %v", brk.what).
                                        debug(1)
                                }
                        }
                } else if brks = t.breakersOf(breakCase, breakDone); len(brks) > 0 {
                        t.breakers = t.breakersNot(breakCase, breakDone)
                        for _, brk := range t.breakers {
                                diag.errorAt(program.position, "breaker: %v", brk.what).
                                        debug(1)
                        }
                        break
                } else if brks = t.breakersOf(breakNext); len(brks) > 0 {
                        t.breakers = t.breakersNot(breakNext)
                        for _, brk := range t.breakers {
                                diag.errorAt(program.position, "breaker: %v", brk.what).
                                        debug(1)
                        }
                        continue
                } else if len(t.breakers) > 0 {
                        for _, brk := range t.breakers {
                                diag.errorAt(program.position, "unknown breaker: %v", brk.what).
                                        debug(1)
                        }
                        break
                }
        }
        breakers = t.breakers
        return
}
func (entry *RuleEntry) Get(name string) (Value, error) {
        switch name {
        case "class": return MakeString(entry.position, entry.class.String()), nil
        case "name": return MakeString(entry.position, entry.Name()), nil
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
func (entry *RuleEntry) defs(s string) (res []*Def) {
        res = entry.target.defs(s)
        for _, prog := range entry.programs {
                for _, depend := range prog.depends {
                        res = append(res, depend.defs(s)...)
                }
                for _, recipe := range prog.recipes {
                        res = append(res, recipe.defs(s)...)
                }
        }
        return
}
func (entry *RuleEntry) expandible(w expandwhat) (res bool) {
        if res = entry.target.expandible(w); res { return }

        // // TODO: do more tests for this to see if we need to fallthrough
        // return // only check closured agaist target

        for _, prog := range entry.programs {
                for _, depend := range prog.depends {
                        if res = depend.expandible(w); res { return }
                }
                for _, recipe := range prog.recipes {
                        if res = recipe.expandible(w); res { return }
                }
        }
        return
}
func (entry *RuleEntry) expand(w expandwhat) (res Value, err error) {
        if entry == nil {
                // happens from some &{xxx} exprs
                err = fmt.Errorf("expand nil entry")
                return
        }

        var target Value
        if target, err = entry.target.expand(w); err != nil {
                diag.errorAt(entry.position, "expand '%v' failed: %v", entry.target, err).
                        debug(1)
                return
        } else if !isNil(target) && target != entry.target {
                // TODO: test if programs are needed to be disclosed??
                res = &RuleEntry{
                        entry.class, target,
                        entry.programs,
                        entry.position,
                }
        }
        return
}
func (entry *RuleEntry) stamp(t *traversal) (files []*File, err error) { return entry.target.stamp(t) }
func (entry *RuleEntry) traverse(t *traversal) {
        if optionTraceTraversal { defer un(tt(t_traverse, t, entry.target)) }
        if optionEnableBenchmarks && false { defer bench(mark("RuleEntry.traverse")) }
        if optionEnableBenchspots { defer bench(spot("RuleEntry.traverse")) }
ForPrograms:
        for _, prog := range entry.programs {
                if brks := t.breakersNot(breakNext); len(brks) > 0 {
                        diag.warnAt(prog.position, "broken traversal %v: %v (stems = %v)",
                                entry, brks[0].what, t.stems).
                                debug(6)
                        return
                }
                if prog.execute(t, entry, t.arguments); false && t.hasBreakers() {
                        diag.warnAt(prog.position, "entry: %v %d, %v, %v, %v",
                                entry, len(entry.programs), t.stems, t.def.target.value, t.breakers[0].what).
                                debug(breakDone > 0, 6)
                }

                // Update traversal breakers
                var breakers []*breaker = t.breakers
                t.breakers = nil // take and reset breakers
                for _, brk := range breakers {
                        // NOTE: see traversal.file and traversal.target for further processing
                        switch brk.what {
                        case breakCase, breakDone:
                                // FIXME: t.breakers = append(t.breakers, brk)
                                break ForPrograms // case selected or execution fully done
                        case breakFail, breakErro:
                                t.breakers = append(t.breakers, brk)
                                break ForPrograms
                        case breakNext:
                                t.breakers = append(t.breakers, brk)
                                continue ForPrograms
                        default:
                                diag.warnAt(prog.position, "broken traversal %v: %v", entry, brk.what).
                                        debug(6)
                                break ForPrograms
                        }
                }
        }
        return
}
func (entry *RuleEntry) stat(t *traversal) (si *statinfo) {
        // FIXME: entry.target maybe not the real target
        return entry.target.stat(t)
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
func (entry *RuleEntry) patterned() bool { return entry.target.patterned() }
func (entry *RuleEntry) match(i interface{}) (full bool, s string, stems []string) {
    full, s, stems = entry.target.match(i)
    return
}
func (entry *RuleEntry) stencil(stems []string) (s string, rest []string) {
    s, rest = entry.target.stencil(stems)
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

type PatternEntry struct { Pattern Value; *RuleEntry }
func (p *PatternEntry) expandible(w expandwhat) (res bool) {
        if res = p.Pattern.expandible(w); !res {
                res = p.RuleEntry.expandible(w)
        }
        return
}
func (p *PatternEntry) expand(w expandwhat) (res Value, err error) {
        var pat, ent Value
        if pat, err = p.Pattern.expand(w); err != nil {
                diag.errorOf(p.Pattern, "expand '%v' failed: %v", p.Pattern, err).
                        debug(1)
        } else if ent, err = p.RuleEntry.expand(w); err != nil {
                diag.errorOf(p.RuleEntry, "expand '%v' failed: %v", p.RuleEntry, err).
                        debug(1)
        } else if (!isNil(pat) && pat != p.Pattern) && (!isNil(ent) && ent != p.RuleEntry) {
                res = &PatternEntry{p.Pattern, ent.(*RuleEntry)}
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

type stemmed struct { *PatternEntry; Stems []string }
func (p *stemmed) String() (s string) {
        for i, stem := range p.Stems { if i > 0 { s += "," }; s += stem }
        return fmt.Sprintf("<%s,%s>", p.PatternEntry, s)
}
func (p *stemmed) expand(w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.PatternEntry.expand(w); err != nil {
                diag.errorOf(p.PatternEntry, "expand '%v' failed: %v", p.PatternEntry, err).
                        debug(1)
                return
        } else if !isNil(v) && v != p.PatternEntry {
                res = &stemmed{v.(*PatternEntry), p.Stems}
        }
        return
}
func (p *stemmed) cmp(v Value) (res cmpres) {
        if a, ok := v.(*stemmed); ok {
                assert(ok, "value is not stemmed")
                if len(p.Stems) != len(p.Stems) { return }
                for i, stem := range p.Stems {
                        if stem != a.Stems[i] { return }
                }
                res = p.PatternEntry.cmp(a.PatternEntry)
        }
        return
}
func (p *stemmed) traverse(t *traversal) {
        diag.errorAt(p.position, "cant traverse stemmed entry directly: %v", p).
                debug(1)
        t.breakers = append(t.breakers, &breaker{
                pos: p.position, what:breakErro,
                error: fmt.Errorf("traversing stemmed entry: %v", p),
        })
        return
}
func (p *stemmed) string(t *traversal, targetVal Value, target string) {
        if optionTraceTraversal   { defer un(tt(t_traverse, t, p)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("stemmed.traverse(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("stemmed.traverse")) }

        defer func(a Value, s []string) { p.target, t.stems = a, s } (p.target, t.stems)
        t.stems = p.Stems // set stems for the traversal

        if file := t.project.FindFile(target); file != nil {
                file.position = p.position
                p.target = file
        } else {
                p.target = targetVal
        }

        p.RuleEntry.traverse(t)
}
func (p *stemmed) file(t *traversal, file *File) {
        if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("stemmed.file(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("stemmed.file")) }

        defer func(a Value, s []string) { p.target, t.stems = a, s } (p.target, t.stems)
        t.stems = p.Stems // set stems for the traversal

        if file.info == nil && file.filemap == nil { // !isAbsOrRel()
                if f := t.project.FindFile(file.name); f != nil { *file = *f }
                //if file.info == nil { file.info, _ = os.Stat(file.name) }
        }
        file.position = p.position
        p.target = file
        p.RuleEntry.traverse(t)
}
