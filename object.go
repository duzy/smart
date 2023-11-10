//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "reflect"
    // "runtime"
    "os/exec"
    "strings"
    "strconv"
    "sync"
    "sync/atomic"
    "bytes"
    "unsafe"
    "time"
    "fmt"
)

type Object interface {
    Value

    declScope() *Scope
    OwnerProject() *Project

    // Get object's named property.
    Get(ctx Context, name string) (Value, error)

    // rescope the object.
    rescope(ctx Context, scope *Scope)
}

type objbase struct { // generally unnamed objects
    valbase
    scope *Scope
    owner *Project
}
func (_ *objbase) kind() Kind { return KindObject }
func (p *objbase) declScope() *Scope { return p.scope }
func (p *objbase) OwnerProject() *Project { return p.owner }
func (p *objbase) String() string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) string(ctx Context) string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) name(ctx Context) string { panic("inquiring name of an unknown object") }
func (p *objbase) Get(_ Context, name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p *objbase) rescope(_ Context, scope *Scope) { panic("rescoping unknown object") }
func (p *objbase) exists() existence { return existenceMatterless }

type knownobject struct { // generally named objects
    objbase
    name_ string // single, or group name if containing '(*)' and corresponding members
    //members [][]string
}
func (p *knownobject) kind() Kind { return p.objbase.kind()|KindKnownObject }
func (p *knownobject) String() string { return fmt.Sprintf("{object %s}", p.name_) }
func (p *knownobject) string(_ Context) string { return fmt.Sprintf("{object %s}", p.name_) }
func (p *knownobject) true(_ Context) bool { return p.name_ != "" }
func (p *knownobject) name(_ Context) string { return p.name_ }
func (p *knownobject) rescope(_ Context, scope *Scope) {
    if p.scope != scope {
        if p.scope != nil {
            delete(p.scope.elems, p.name_)
        }
        if p.scope = scope; p.scope != nil {
            p.scope.elems[p.name_] = p
        }
    }
}
func (p *knownobject) expand(_ Context, _ facet) Value { return p }
func (p *knownobject) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*knownobject); ok {
        assert(ok, "value is not knownobject")
        if p.owner == a.owner && p.scope == a.scope && p.name_ == a.name_ {
            res = cmpEqual
        }
    } else if l, ok := v.(*list); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (_ *knownobject) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *knownobject) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *knownobject) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type unresolved struct { // named callable/executable objects
    Value
    project *Project
}
func (_ unresolved) kind() Kind { return KindObject|KindUnresolved }
func (p unresolved) name(ctx Context) (name string) {
    if p.Value == nil {
        erro(at(ctx,p.Position()), "unresolved object name is nil")
    } else if ctx == nil {
        name = p.Value.String()
    } else {
        name = p.Value.string(ctx)
    }
    return
}
// func (p unresolved) Position() Position { return p.Value.Position() }
// func (p unresolved) String() string { return p.Value.String() }
// func (p unresolved) string(ctx Context) (s string) { return /* p.Value.string(ctx) */ }
func (p unresolved) float(_ Context) (float64, error) { return 0.0, nil }
func (p unresolved) int(_ Context) (int64, error) { return 0, nil }
func (p unresolved) true(_ Context) bool { return false }
func (p unresolved) OwnerProject() *Project { return p.project }
func (p unresolved) declScope() *Scope { return p.project.scope }
func (p unresolved) Get(_ Context, name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p unresolved) patterned(_ Context) bool { return false }
func (p unresolved) invoke(_ Context, _ facet, _, _ []Value) Value { return p }
func (p unresolved) execute(ctx Context, a ...Value) (result []Value, err error) { return []Value{p}, nil }
func (p unresolved) rescope(ctx Context, scope *Scope) {
    if true {
        panic(failure{"cant rescope a unresolved object",ia(p.Value.Position())})
    } else if p.project != scope.project {
        var name = p.Value.string(ctx)
        if p.project.scope != nil { delete(p.project.scope.elems, name) }
        if p.project = scope.project; p.project.scope != nil {
            p.project.scope.elems[name] = p
        }
    }
}
func (p unresolved) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(unresolved); y {
        res = p.Value.cmp(ctx, a.Value)
    } else {
        res = p.Value.cmp(ctx, v)
    }
    return
}
func (p unresolved) defs(_ Context, _ ...string) (res []*def) { return }
func (p unresolved) refs(ctx Context, v Value) (res bool) {
    if o, y := v.(unresolved); y {
        if p.Value == o.Value || p.Value.refs(ctx, o.Value) { return true }
        if p.Value.cmp(ctx, o.Value) == cmpEqual { return true }
        if (p.project == o.project || p.project.hasBase(o.project)) &&
            p.Value.string(ctx) == o.Value.string(ctx) { return true }
    } else if d, y := v.(*def); false && y {
        if o := p.project.resolve(ctx, p.Value.string(ctx)); o == nil {
            if o == d || o.refs(ctx, d) { return true }
        }
    }
    return
}
func (p unresolved) expandable(_ Context, _ facet) bool { return false }
func (p unresolved) expand(ctx Context, w facet) (res Value) {
    var v Value
    var db bool
    if false { if w&expandDebug != 0 { defer func() {
        w.noted(ctx, p, p.Value, ctx.ic().a)
        if false { infostack(of(ctx,p), 3) }

        noted(ctx, "%v: %v %v ⇒ %v %v", p, typeof(p.Value), p.Value, typeof(v), v)
        noted(ctx, "%v: %v %v (same=%v)", p, typeof(res), res, (res == p)).debug(24)
    }(); db = true ; w |= expandDebug }}

    if v = p.Value.expand(ctx, w); w&expandEvoke == 0 {
        if v != nil && v != p.Value {
            return unresolved{v, p.project}
        } else {
            return p
        }
    }

    if ic := ctx.ic(); ic != nil && ic.a != nil { // Always expand invocation args.
        a, _, _ := (w|expandAuto|expandArgs).expand(ctx, ic.a...)
        if db { noted(ctx, "%v: %v ⇒ %v", p, ic.a, a).debug(1) }
        ic.a = a
    }

    if u, y := v.(unexpanded); y {
        if u.Value != v {
            return unresolved{u.Value,p.project}
        } else {
            return p
        }
    }

    // TODO: only if w&expandResolve != 0 ...

    if name := v.string(ctx); name == "" {
        warnstack(ctx, 3, "empty unresolved: %T %v ⇒ %T %v", p.Value, p.Value, v, v).debug(3)
        return p
    } else if o := p.project.resolve(ctx, name); o == nil {
        return p
    } else if o.refs(ctx, p) {
        if true { warnstack(ctx, 3, "recursive: %v ⇒ %v %v", p, typeof(o), o).debug(3) }
        if v != p.Value { return unresolved{v, p.project} }
        return p
    } else {
        if p.String() == ".test.x" { noted(ctx, "unresolved: %v: %v", p, o).debug(1) }
        res = o.expand(ctx, w|expandAuto|expandDigits)
        if db { noted(ctx, "%v: %v ⇒ %v ⇒ %v ; %v", p, name, o, res, ctx.ic().a).debug(1) }
        return
    }
}
func (p unresolved) traverse(ctx Context) { }
func (_ unresolved) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ unresolved) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ unresolved) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type projectname struct { *Project ; scope *Scope }
func (_ *projectname) kind() Kind { return KindObject|KindKnownObject|KindProjectName }
func (_ *projectname) int(_ Context) (int64, error) { return 0, nil }
func (_ *projectname) float(_ Context) (float64, error) { return .0, nil }
func (_ *projectname) updated(_ Context) bool { return false }
func (_ *projectname) updatedDeps(_ Context, _ ...Value) []Value { return nil }
func (_ *projectname) stamp(ctx Context) (files []*File, err error) { return }
func (_ *projectname) delete(_ Context) (files []*File, err error) { return }
func (_ *projectname) defs(_ Context, _ ...string) (res []*def) { return }
func (_ *projectname) refs(_ Context, _ Value) (res bool) { return }
func (_ *projectname) patterned(_ Context) bool { return false }
func (_ *projectname) expandable(_ Context, _ facet) bool { return false }
func (p *projectname) stencil(ctx Context, stems []string) (Value, []string) { return p, stems }
func (p *projectname) match(ctx Context, i interface{}) (bool, interface{}, []string) { return stringMatch(ctx, p, i) }
func (p *projectname) Position() Position { return p.position }
func (p *projectname) String() string { return p.Project.name }
func (p *projectname) string(_ Context) string { return p.Project.name }
func (p *projectname) name(_ Context) string { return p.Project.name }
func (p *projectname) true(_ Context) bool { return p.Project != nil }
func (p *projectname) declScope() *Scope { return p.scope }
func (p *projectname) OwnerProject() *Project { return p.scope.project }
func (p *projectname) Get(ctx Context, name string) (Value, error) { return p.resolve(ctx, name), nil }
func (p *projectname) expand(_ Context, _ facet) (res Value) { return expanded{p} }
// func (p *projectname) invoke(_ Context, _ facet, _, _ []Value) Value { return p }
func (p *projectname) traverse(ctx Context) {
    if t := p.Project.defaultEntry; t != nil {
        switch t.Target().(type) {
        case flag: return
        }

        t.traverse(ctx)
    }
}
func (p *projectname) stat(ctx Context) (si *statinfo) {
    if t := p.Project.defaultEntry; t != nil { si = t.stat(ctx) }
    return
}
func (p *projectname) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*projectname); y {
        assert(y, "value is not projectname")
        if p.Project == a.Project { res = cmpEqual }
    } else if l, y := v.(*list); y && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (p *projectname) rescope(_ Context, scope *Scope) {
    if p.scope != scope {
        if p.scope != nil {
            delete(p.scope.elems, p.Project.name)
        }
        if p.scope = scope; p.scope != nil {
            p.scope.elems[p.Project.name] = p
        }
    }
}
func (_ *projectname) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *projectname) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *projectname) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type self struct { projectname }
func (p *self) kind() Kind { return p.projectname.kind()|KindSelf }
func (_ *self) elemstr(_ Context, o Object, k elembits) (s string) { return "$(.self)" }
func (_ *self) String() string { return ".self" }
func (p *self) name(_ Context) string { return p.String() }
func (p *self) expand(_ Context, _ facet) Value { return expanded{p} }

type scopename struct { *Scope ; name_ string }
func (_ *scopename) kind() Kind { return KindObject|KindScopeName }
func (_ *scopename) int(_ Context) (int64, error) { return 0, nil }
func (_ *scopename) float(_ Context) (float64, error) { return .0, nil }
func (_ *scopename) updated(_ Context) bool { return false }
func (_ *scopename) updatedDeps(_ Context, _ ...Value) []Value { return nil }
func (_ *scopename) stamp(ctx Context) (files []*File, err error) { return }
func (_ *scopename) delete(_ Context) (files []*File, err error) { return }
func (_ *scopename) defs(_ Context, _ ...string) (res []*def) { return }
func (_ *scopename) refs(_ Context, _ Value) (res bool) { return }
func (_ *scopename) patterned(_ Context) bool { return false }
func (_ *scopename) expandable(_ Context, _ facet) bool { return false }
func (_ *scopename) stat(ctx Context) (si *statinfo) { return }
func (_ *scopename) traverse(ctx Context) { }
func (p *scopename) match(ctx Context, i interface{}) (bool, interface{}, []string) { return stringMatch(ctx, p, i) }
func (p *scopename) stencil(ctx Context, stems []string) (Value, []string) { return p, stems }
func (p *scopename) Position() Position { return p.position }
func (p *scopename) String() string  { return fmt.Sprintf("{scope %s}", p.name_) }
func (p *scopename) string(_ Context) string { return p.name_ }
func (p *scopename) name(_ Context) string { return p.name_ }
func (p *scopename) true(_ Context) bool { return p.Scope != nil }
func (p *scopename) OwnerProject() *Project { return p.Scope.project }
func (p *scopename) declScope() *Scope { return p.Scope.outer }
// func (p *scopename) invoke(_ Context, _ facet, _, _ []Value) Value { return p }
func (p *scopename) expand(_ Context, _ facet) (res Value) { return p }
func (p *scopename) Get(ctx Context, name string) (value Value, err error) {
    if s := p.resolve(name); s != nil { if value, _ = s.(Value); value == nil {
        err = fmt.Errorf("`%s' in scope is invalid (%T)", name, s)
    }} else {
        err = fmt.Errorf("undefined `%s' in scope `%s'", name, p.name_)
    }
    return
}
func (p *scopename) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*scopename); y {
        if p.Scope == a.Scope { res = cmpEqual }
    } else if l, y := v.(*list); y && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (p *scopename) rescope(_ Context, scope *Scope) {
    if p.Scope != scope {
        if p.Scope != nil {
            delete(p.Scope.elems, p.name_)
        }
        if p.Scope = scope; p.Scope != nil {
            p.Scope.elems[p.name_] = p
        }
    }
}
func (_ *scopename) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *scopename) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *scopename) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type Origin int

const (
    DefVoid Origin = iota
    DefConfig   // configure
    DefConfDir  // configuration defs
    DefConfRef  // referred by config
    DefDecl     // declaration names
    DefDefault  // =    normal value
    DefExecute  // !=  value to be executed
    DefExpand1  // :=   expand delegates (simple expand)
    DefExpand2  // ::=  expand all (delegates, closures, paths)
    DefExpand3  // ;:=  TODO: expand as plain
    // DefExecuted, executed result, add !:= for immediately executed defs
    _DefAny // referred any def
)

func (o Origin) String() (s string) {
    switch o {
    case DefVoid:    s = "Void"
    case DefConfDir: s = "ConfDir"
    case DefConfRef: s = "ConfRef"
    case DefConfig:  s = "Config"
    case DefDecl:    s = "Decl"
    case DefDefault: s = "Default"
    case DefExecute: s = "Execute"
    case DefExpand1: s = "Expand1"
    case DefExpand2: s = "Expand2"
    case DefExpand3: s = "Expand3"
    case _DefAny:    s = "any"
    default: s = fmt.Sprintf("Origin<%d>", o)
    }
    return
}

type autoDefMap map[string]*def
func (am autoDefMap) String() (str string) {
    var strs []string
    for _, d := range am {
        // strs = append(strs, fmt.Sprintf("%s:%v", d.name, d.value))
        strs = append(strs, d.String())
    }
    return fmt.Sprintf("%v", strs)
}
func (am autoDefMap) clone() (res autoDefMap) {
    res = make(autoDefMap)
    for s, d := range am {
        t := new(def)
        // d.mutex.Lock()
        t.knownobject = d.knownobject
        t.value = d.value
        // d.mutex.Unlock()
        res[s] = t
    }
    return
}

type autoContext struct {
    Context
    sync.RWMutex
    defs autoDefMap
}
func (ac *autoContext) inner() Context { return ac.Context }
func (ac *autoContext) aquireLock() func() { ac.Lock() ; return func() { ac.Unlock() }}
func (ac *autoContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("auto{%s}", ac.Context)
    } else {
        return ac.Context.String()
    }
}
func (ac *autoContext) ac() *autoContext { return ac }
func (ac *autoContext) amend(ctx Context, name string, val Value) (out *def, res Value) {
    if d := ac.get(ctx, name); d == nil { return ac.set(ctx, name, val) } else
    if res = d.value; d.value != val { out, d.value = d, val }
    return
}
func (ac *autoContext) get(ctx Context, name string) (res *def) {
    ac.RLock()
    res, _ = ac.defs[name]
    ac.RUnlock()

    if res != nil { return }

    var skipDigits = true
    switch ac.Context.(type) {
    case *builtin_foreach, defExpandContext:
        skipDigits = false
    case *builtin_grep:
        skipDigits = true
    }
    if skipDigits && _isDigits(name) {
        i, e := strconv.Atoi(name)
        if e == nil && 0 <= i && i <= maxDigitAutoNum {
            return // Fixes calling-args-pollution: avoid pollution of digit autos ($0, $1, ..., $9)
        } else {
            warn(ctx, "digit auto too big: %v (max %v)", name, maxDigitAutoNum).debug(1)
            return
        }
    }

    if ac.Context != nil { if t := ac.Context.ac(); t != nil {
        if ac != t { res = t.get(ctx, name) } else {
            errostack(ctx, 3, "deadloop: %v", name).debug(32)
        }
    }}
    return
}
func (ac *autoContext) set(ctx Context, name string, val Value) (out *def, res Value) {
    if name == "-" { if d, y := val.(*def); y && d.origin != DefConfig {
        warnstack(ctx, 3, "set $- to def (%v): %v", d.origin, d).debug(16)
    }}

    var ok bool
    ac.RLock()
    out, ok = ac.defs[name]
    ac.RUnlock()

    if ok && out != nil {
        res = out.value
    } else {
        var scope = ac.Scope()
        out = &def{knownobject:knownobject{objbase{scope:scope, owner:scope.project}, name}}
        ac.Lock()
        ac.defs[name] = out
        ac.Unlock()
    }

    // out.Lock()
    if out.value = val; val == nil {
        out.position = ac.Position()
    } else {
        out.position = val.Position()
    }
    // out.Unlock()
    return
}
func (ac *autoContext) args(ctx Context, params []*auto, args []Value) (names []string) {
    var (
        argnum int // setup named/number parameters ($1, $2, etc.)
        compact []Value // compacted args: combine duplicated pairs
        named = make(map[string]struct{}, len(params))
    )
    for _, param := range params { named[param.name_] = struct{}{} }

outer:
    for _, a := range args {
        if p, y := a.(*pair); y { for _, ca := range compact {
            if c, y := ca.(*pair); y && eq(ac, p.Key, c.Key) { var vals = merge(p.Value)
                if l, y := c.Value.(*list); y { l.Elems = append(l.Elems, vals...) } else {
                    c.Value = makeList(append(merge(c.Value), vals...)...)
                }
                continue outer
            }
        }}
        compact = append(compact, a)
    }

    for _, a := range compact {
        //<!IMPORTANT: Don't translate flag, flag values are valid regular arguments.
        //             Pair values are special.
        if a = scalarize(a); false && (isNull(a) || isNone(a)) {
            erro(of(ac,a), "%T '%v' is invalid scalar", a, a).debug(1)
            return
        }

        var name string
        if p, y := a.(*pair); y { s := p.Key.string(ctx)
            if _, y = named[s]; y { name, a = s, p.Value }
        }

        var id = strconv.Itoa(argnum+1)
        if name != "" {
            // Got the name!
        } else if argnum < len(params) {
            name = params[argnum].name(ctx)
        } else {
            name = id
        }

        if d, _ := ac.set(ctx, name, a); d == nil {
            erro(of(ac,a), "arg '%s' not set ($%s)", name, id).debug(1)
            return
        } else if d, y := ac.defs[name]; !y || d == nil {
            erro(of(ac,a), "arg '%s' not set ($%s)", name, id).debug(1)
            return
        } else if id != "" && id != name {
            ac.Lock()
            ac.defs[id] = d // NOTE: set an alias or replace it
            ac.Unlock()
        }
        names = append(names, name)
        argnum += 1
    }

    // // Explicitly 'clear' other digit autos
    // for i := argnum; i <= maxDigitAutoNum; i += 1 {
    //     ac.set(ctx, strconv.Itoa(argnum+1), nil)
    // }

    named = nil
    return
}

func autoVal(ctx Context, name string) (res Value) {
    if d := autoDef(ctx, name); d != nil { res = d.value }
    return
}

func autoDef(ctx Context, name string) (d *def) {
    if a := ctx.ac(); a != nil { d = a.get(ctx, name) }
    return
}

func autoGet(ctx Context, name string) (d *def, res Value) {
    if d = autoDef(ctx, name); d != nil { res = d.value }
    return
}

func autoSet(ctx Context, name string, val Value) (out *def, res Value) {
    if a := ctx.ac(); a != nil { out, res = a.set(ctx, name, val) }
    return
}

type auto struct { knownobject }
func (a *auto) kind() Kind { return a.knownobject.kind()|KindAuto }
func (a *auto) String() (s string) { return a.name_ }
func (a *auto) string(ctx Context) (res string) {
    if d := a.def(ctx); d != nil && d.value != nil { res = d.value.string(ctx) }
    return
}
func (a *auto) Get(ctx Context, name string) (res Value, _ error) {
    if name == "value" { res = autoVal(ctx, a.name_) }
    return
}
func (a *auto) refs(ctx Context, v Value) (res bool) {
    if u, y := v.(unexpanded); y { v = u.Value }
    if o, y := v.(*auto); y && (o == a || o.name_ == a.name_) { return true }
    if rc := ctx.rc(); rc != nil { if o, y := rc.v.(*auto); y && (a == o || a.name_ == o.name_) {
        if false { noted(ctx, "%v: %v %v , %v", a, typeof(v), v, rc.v.refs(ctx, a)).debug(32) }
        return true
    }}
    if d := a.def(ctx); d != nil && d.value != nil {
        res = d.value.refs(&refContext{ctx, a}, v)
    }
    return
}
func (a *auto) defs(ctx Context, s ...string) (res []*def) {
    if d := a.def(ctx); d != nil { res = append(res, d) }
    return
}
func (a *auto) def(ctx Context) *def { return autoDef(ctx, a.name_) }
func (a *auto) set(ctx Context, value Value, app ...Value) {
    if value == nil && app != nil { if d := autoDef(ctx, a.name_); d != nil {
        d.value = ease(ctx, append(merge(d.value), app...))
        d.position = a.position
        return
    }}

    d, _ := autoSet(ctx, a.name_, ease(ctx, append(merge(value), app...)))
    if d != nil { d.position = a.position } else if true {
        warnstack(of(ctx,a), 3, "set auto failed: %v: %v %v", a.name_, value, app).debug(16)
    }
}
func (a *auto) expandable(ctx Context, w facet) (res bool) {
    if w&expandAuto != 0 && w&expandAutoKept == 0 { res = true } else
    if w&expandUnexpandedForth != 0 { if d := autoDef(ctx, a.name_); d != nil {
        res = d.expandable(ctx, w)
    }}
    return
}
func (a *auto) expand(ctx Context, w facet) (res Value) {
    if false { if w&expandDebug != 0 || a.name_ == "flag" { if d := autoDef(ctx, a.name_); true/* d != nil && d.value != res */ { defer func() {
        if true { w.noted(ctx, a, d) }
        noted(ctx, "%v ⇒ %v %v (same=%v,%p)", a, typeof(res), res, (res==a), res).debug(10)
    }()}}}
    g := w&expandAuto != 0 && w&expandAutoKept == 0 || w&expandDefDefArgs != 0 || w&expandUnexpandedForth != 0
    if g && !ctx.ref(ctx, a) { if d := autoDef(ctx, a.name_); d != nil {
        var recured bool
        if ic := ctx.ic().Context.ic(); ic != nil && ic.in(a) { if false { w.noted(ctx, a) }
            noted(ctx, "recursive: %v ⇒ %v ; %v", a, d, autoDef(ctx.ac().Context, a.name_)).debug(5)
            recured = true
        }
        if d.value == nil {
            return unexpanded{a}
        } else if recured || d.value.refs(ctx, a) {
            if false { noted(ctx, "nested: %v ⇒ %v (%v)", a, d, recured).debug(16) }
            return d.expand(&refContext{ctx,a}, w)
        } else {
            return d.expand(ctx, w)
        }
    }}
    return unexpanded{a}
}
func (a *auto) invoke(ctx Context, w facet, o, v []Value) (res Value) {
    if d := autoDef(ctx, a.name_); d != nil { res = d.invoke(ctx, w, o, v) }
    return
}
func (a *auto) cmp(ctx Context, v Value) (res cmpres) {
    if o, y := v.(*auto); y && (a == o || a.name_ == o.name_) { res = cmpEqual } else
    if val := autoVal(ctx, a.name_); val != nil { res = val.cmp(ctx, v) }
    return
}
func (a *auto) stat(ctx Context) (si *statinfo) {
    if val := autoVal(ctx, a.name_); val != nil { si = val.stat(ctx) }
    return
}
func (a *auto) traverse(ctx Context) {
    if val := autoVal(ctx, a.name_); val != nil { val.traverse(ctx) }
    return
}
func (_ *auto) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *auto) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *auto) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type defExpandContext struct { Context }

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type def struct {
    knownobject // mutex sync.Mutex
    // facet //
    origin Origin
    value Value
}
func (d *def) kind() Kind { return d.knownobject.kind()|KindDef }
func (d *def) Position() (pos Position) {
    if d != nil { if d.value == nil {
        pos = d.valbase.position
    } else {
        pos = d.value.Position()
    }} else if true { panic("nil def") }
    return
}
func (d *def) streq() (s string) {
    switch d.origin {
    case DefDefault: s =   "="
    case DefExpand1: s =  ":="
    case DefExpand2: s = "::="
    case DefExecute: s =  "!="
    default:     s =   "⇒"
    }
    return
}
func (d *def) String() (s string) {
    var value Value
    {
        // d.mutex.Lock()
        value = d.value
        // d.mutex.Unlock()
    }
    if s = d.name_ + d.streq(); value != nil {
        s += elemstr(nil, d, value, 0)
    } else {
        s += "<nil>"
    }
    return
}
func (d *def) string(ctx Context) (res string) {
    var val Value
    {
        // d.mutex.Lock()
        val = d.value
        // d.mutex.Unlock()
    }
    if val != nil { res = val.string(ctx) }
    return
}
func (d *def) true(ctx Context) (res bool) {
    var val Value
    {
        // d.mutex.Lock()
        val = d.value
        // d.mutex.Unlock()
    }
    if val != nil { res = val.true(ctx) }
    return
}
func (d *def) refs(ctx Context, v Value) (res bool) {
    if o, y := v.(*def); y { if d == o { return true } }

    var val Value
    {
        // d.mutex.Lock()
        val = d.value
        // d.mutex.Unlock()
    }
    if val != nil { res = val.refs(ctx, v) }
    return
}
func (d *def) defs(ctx Context, s ...string) (res []*def) {
    if len(s) == 0 {
        res = append(res, d)
    } else {
        for _, a := range s {
            if d.name_ == a {
                res = append(res, d)
                break
            }
        }
    }

    var val Value
    {
        // d.mutex.Lock()
        val = d.value
        // d.mutex.Unlock()
    }
    if val != nil { res = append(res, val.defs(ctx, s...)...) }
    return
}
func (d *def) expandable(ctx Context, w facet) (res bool) {
    var val Value
    {
        // d.mutex.Lock()
        val = d.value
        // d.mutex.Unlock()
    }
    if val != nil { res = val.expandable(ctx, w) }
    return
}
func (d *def) expand(ctx Context, w facet) (res Value) { var db bool
    if false { if w&expandDebug != 0 { defer func(w0 facet) {
        if d.value != nil { ctx = of(ctx, d.value) } else { ctx = of(ctx, d) }
        if true { w0.noted(ctx, d) }
        var a []Value ; if i := ctx.ic(); i != nil { a = i.a }
        noted(ctx, "%v: %v %v", d, d.origin, a)
        noted(ctx, "%v: %v %v", d.name_, typeof(d.value), d.value)
        noted(ctx, "%v: %v %v (same=%v)", d.name_, typeof(res), res, (res==d.value)).debug(16)
    }(w); db = true }}

    if w&expandEvoke == 0 {
        if false { warnstack(ctx, 3, "def.expand: invalid (%030b)", w).debug(16) }
        return d
    } else if false {
        if ctx.ref(ctx, d) { return unexpanded{d} }

        var recured bool
        if ic := ctx.ic().Context.ic(); ic != nil && ic.in(d) {
            noted(ctx, "recursive: %v", d).debug(5)
            recured = true
        }
        if recured || (d.value != nil && d.value.refs(ctx, d)) {
            if false { noted(ctx, "nested: %v (%v)", d, recured).debug(16) }
            return d.value.expand(&refContext{ctx,d}, w)
        }
    }

    var ic = ctx.ic()
    if ic == nil {
        errostack(ctx, 3, "def.expand: nil delegate context (%030b)", w).debug(16)
        return
    }

    if w&expandDefOriginOff == 0 { switch d.origin {
    case DefDefault: if ic.a != nil { w |= expandDefDefArgs }
    case DefExpand1: w |= expandDelegate
    case DefExpand2: w |= expandDelegate|expandClosure
    }}

    var u, n int
    if ic.a, u, n = (w|expandArgs).expand(ctx, ic.a...); u > 0 || n > 0 {}
    if d.value != nil { if res = d.xauto(ctx, w, ic.a...); d.origin == DefExecute {
        res = d.xexec(ctx, res, ic.a...)
    }}

    if db { noted(ctx, "%v %v -> %v ; %v %v %v", d.origin, d, res, ic.a, u, n).debug(1) }

    if ic.x = true; res == nil { return d }
    return scalarize(res)
}
func (d *def) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*def); y {
        if d == a || (isNull(d.value) && isNull(a.value)) {
            res = cmpEqual
            return
        }

        var val1, val2 Value
        {
            // d.mutex.Lock()
            val1 = d.value
            // d.mutex.Unlock()
        }
        {
            // a.mutex.Lock()
            val2 = a.value
            // a.mutex.Unlock()
        }
        if isNull(val1) {
            if isNull(val2) { res = cmpEqual }
        } else if !isNull(val2) {
            res = val1.cmp(ctx, val2)
        }
    } else if l, y := v.(*list); y && len(l.Elems) == 1 {
        res = d.cmp(ctx, l.Elems[0])
    }
    return
}
func (d *def) elemstr(_ Context, o Object, k elembits) (s string) {
    if o != nil {
        if p := d.OwnerProject(); p != o.OwnerProject() {
            return fmt.Sprintf("$(%s->%s)", p.name, d.name_)
        }
    }
    s = fmt.Sprintf(`$(%s)`, d.name_)
    return
}
func (d *def) val(ctx Context, value Value, ii ...interface{}) {
    d.set(ctx, d.origin, value, ii...)
}
func (d *def) set(ctx Context, origin Origin, value Value, ii ...interface{}) {
    if value != nil && d.value == value {
        if d.origin != origin { d.origin = origin }
        return
    }

    if false && !d.position.IsValid() {
        erro(ctx, "%s: invalid def position", d.name_).debug(16)
    }

    var w = plain
    var app []Value
    for _, i := range ii {
        switch v := i.(type) {
        case Value: app = append(app, v)
        case facet: w |= v
        }
    }

    var vals []Value
    if value != nil { if l, y := value.(*list); y {
        vals = l.Elems
    } else {
        vals = []Value{ value }
    }}
    if app != nil { vals = append(vals, app...) }

    var pos = d.position
    if !pos.IsValid() && len(vals) > 0 {
        pos = vals[0].Position()
    }

    for i, val := range vals { if o, y := val.(*def); y {
        if o.value == nil {
            vals[i] = makeNull(pos)
        } else {
            vals[i] = o.value
        }
    }}

    switch origin {
    case DefExpand1: vals, _, _ = (w&^expandClosure).expand(ctx, vals...) //  :=
    case DefExpand2: vals, _, _ = (w| expandClosure).expand(ctx, vals...) // ::=
    default:
        var u = cast[*universe](ctx)
        for _, val := range vals {
            var ctx = at(ctx, val.Position())
            if val != nil && val.refs(ctx, d) {
                if u.verbose { prompt(ctx, "set %s (%v): %v\n", origin, d.name_, val) }
                if u.debug   { noted(ctx, "from here").debug(1) }
                errostack(ctx, 3, "value refers to assigning Def '%s': %v (%T)", d.name_, val, val).debug(10)
                return
            }
        }
    }

    if value == nil && len(app) > 0 {
        // d.mutex.Lock()
        if d.value != nil { if l, y := d.value.(*list); y {
            vals = append(l.Elems, vals...)
        } else if !isNull(d.value) {
            vals = append([]Value{d.value}, vals...)
        }}
        // d.mutex.Unlock()
    }

    if n := len(vals); n == 1 {
        value = vals[0]
    } else if n > 1 {
        value = makeList(vals...)
    } else if origin != DefExecute {
        value = makeNull(pos)
    }

    // d.mutex.Lock()
    if !d.position.IsValid() { d.position = pos }
    d.origin, d.value = origin, value
    // d.mutex.Unlock()
    return
}
func (d *def) append(ctx Context, va ...Value) {
    if len(va) > 0 { d.set(ctx, d.origin, nil, vi(va...)...) }
}
func (d *def) invoke(ctx Context, w facet, o, a []Value) (res Value) {
    return invoke(ctx, d, w, o, a)
}
func (d *def) xauto(ctx Context, w facet, a ...Value) (res Value) {
    // d.mutex.Lock()
    // var bits = d.bits
    // d.bits |= defUnavail
    res = xauto(defExpandContext{ctx}, d.value, w, a...)
    // d.bits = bits
    // d.mutex.Unlock()
    return
}
func (d *def) xexec(ctx Context, value Value, a ...Value) (res Value) {
    if isTrivial(value) { return }

    var cmd string
    if cmd = value.string(ctx); cmd == "" {
        warn(ctx, "%v: empty command (value=%v)", d.name_, value).debug(1)
        return
    }

    // TODO: options for running command in the specified container
    var (
        stdout, stderr bytes.Buffer
        sh = exec.Command("sh", "-c", cmd)
    )
    sh.Stdout, sh.Stderr = &stdout, &stderr
    if err := sh.Run(); err != nil {
        erro(ctx, "%v: execute command failed: %v", d.name_, err)
        erro(ctx, "%v: execute command: %s", d.name_, cmd).debug(2)
        stdout.Reset()
        stderr.Reset()
        return
    }

    var pos = value.Position()
    if !pos.IsValid() { pos = ctx.Position() }
    res = makeStrlit(pos, strings.TrimSpace(stdout.String()))
    stdout.Reset()
    stderr.Reset()
    return
}
func (d *def) Get(ctx Context, name string) (res Value, err error) {
    switch name {
    case "name" : res = makeStrlit(d.position, d.name_)
    case "value":
        // d.mutex.Lock()
        res = d.value
        // d.mutex.Unlock()
    default:
        err = fmt.Errorf("no such property `%s' (Def)", name)
        erro(at(ctx,d.position), "%v", err).debug(1)
    }
    return
}
func (d *def) traverse(ctx Context) {
    var value Value
    {
        // d.mutex.Lock()
        value = d.value
        // d.mutex.Unlock()
    }
    if value != nil { value.traverse(ctx) }
}
func (d *def) stat(ctx Context) (si *statinfo) {
    var value Value
    {
        // d.mutex.Lock()
        value = d.value
        // d.mutex.Unlock()
    }
    if value != nil { si = value.stat(ctx) }
    return
}
func (_ *def) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *def) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *def) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

func _isDigits(s string) bool {
    return strings.IndexFunc(s, func(c rune) bool { return !IsDigit(c) }) < 0
}

type undetermined struct {
    tok Token
    identifier Value
    value Value
}
func (_ *undetermined) kind() Kind { return KindObject|KindUndetermined }
func (p *undetermined) Position() Position { return p.identifier.Position() }
func (p *undetermined) String() (s string) {
    s = p.identifier.String()
    s += p.tok.String()
    s += p.value.String()
    return
}
func (p *undetermined) string(ctx Context) string { return p.value.string(ctx) }
func (p *undetermined) name(ctx Context) string { return p.identifier.string(ctx) }
func (p *undetermined) true(ctx Context) bool { return false }
func (p *undetermined) float(ctx Context) (f float64, _ error) { return 0, nil }
func (p *undetermined) int(ctx Context) (i int64, _ error) { return 0, nil }
func (p *undetermined) updated(_ Context) bool { return false }
func (p *undetermined) updatedDeps(_ Context, _ ...Value) []Value { return nil }
func (p *undetermined) refs(ctx Context, v Value) bool {
    return p.identifier.refs(ctx, v) || p.value.refs(ctx, v)
}
func (p *undetermined) defs(ctx Context, s ...string) (res []*def) {
    return append(p.identifier.defs(ctx, s...), p.value.defs(ctx, s...)...)
}
func (p *undetermined) expandable(ctx Context, w facet) bool {
    return p.identifier.expandable(ctx, w) || p.value.expandable(ctx, w)
}
func (p *undetermined) expand(ctx Context, w facet) (res Value) {
    var (
        i = p.identifier.expand(ctx, w)
        v = p.value.expand(ctx, w)
    )
    if i != p.identifier || v != p.value {
        res = &undetermined{ p.tok, i, v }
    } else {
        res = p
    }
    return
}
func (p *undetermined) traverse(ctx Context) { }
func (p *undetermined) exists() existence { return existenceMatterless }
func (p *undetermined) stat(ctx Context) (si *statinfo) { return }
func (p *undetermined) stamp(ctx Context) (files []*File, err error) { return }
func (p *undetermined) delete(ctx Context) (files []*File, err error) { return }
func (p *undetermined) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*undetermined); ok {
        assert(ok, "value is not undetermined")
        if p.identifier.cmp(ctx, a.identifier) == cmpEqual {
            if p.value.cmp(ctx, a.value) == cmpEqual {
                res = cmpEqual
            }
        }
    } else if l, ok := v.(*list); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (p *undetermined) patterned(ctx Context) bool { return false }
func (p *undetermined) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return }
func (p *undetermined) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (_ *undetermined) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *undetermined) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *undetermined) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

const max_expand = 30

type skip struct {}

// A builtin represents a built-in function. builtins don't have a valid type.
type builtin struct { knownobject ; t reflect.Type }
func (p *builtin) kind() Kind { return p.knownobject.kind()|KindBuiltin }
func (p *builtin) String() string { return p.name_ }
func (p *builtin) true(_ Context) bool { return p.t != nil }
func (p *builtin) isCommand() bool { return reflect.PointerTo(p.t).Implements(builtin_c_t) }
func (p *builtin) invoke(ctx Context, w facet, o, a []Value) Value { return invoke(ctx, p, w, o, a) }
func (p *builtin) refs(ctx Context, v Value) (res bool) {
    if o, y := v.(*builtin); y { res = o == p /* || p.name_ == o.name_ */ }
    return
}
func (p *builtin) expand(ctx Context, w facet) (res Value) {
    if w&expandEvoke == 0 {
        if false { warnstack(ctx, 3, "builtin.expand: invalid (%030b)", w).debug(16) }
        return p
    }

	defer dtrace(ctx, "builtin.expand")

    var ic = ctx.ic()
    if ic == nil {
        errostack(ctx, 3, "builtin.expand: nil delegate context (%030b)", w).debug(16)
        return p
    }

    if false { if w&expandDebug != 0 || (cast[*universe](ctx).db("delegate.expand") && p.name_ == "if") { defer func() {
        noted(ctx, "%v: %v ⇒ %v %v", p, ic.a, typeof(res), res).debug(24)
    }()}}

    // Check builtin maximum expand-depth per invocation.
    if t := atomic.AddInt32(&ic.int32, 1); t > int32(max_expand) {
        if len(ic.a) > 0 && ic.a[0].String() == "unique" {
            noted(ctx, "%v: %v", p, ic.a).debug(1)
        }
        errostack(of(ctx, p), 3, "max expand: %v %v (depth=%v,facet=%030b)", p, ic.a, t, w).debug(t)
        panic(failure{"max expand: %v %v (depth=%d)",ia(p, ic.a, t)})
    }
    defer atomic.AddInt32(&ic.int32, -1)

    // Check self-dependency in arguments.
    for i, a := range ic.a { if a == p /* || a.refs(ctx, p) */ {
        errostack(of(ctx,a), 5, "self-dependency: %v ⇒ %v [%d]", p, ic.a, i).debug(10)
        panic(failure{"self-dependency: %v ⇒ %d %v",ia(p, a, i)})
    }}

    bv := reflect.New(p.t)
    bi := bv.Interface()

    defer func(t0 time.Time) {
        if d := time.Now().Sub(t0); d > 1*time.Second {
            noted(ctx, "%v: slow: %v", p, d).debug(3)
        } else if t := bv.Elem().FieldByName("timing"); !t.IsValid() {
            if false { noted(ctx, "%v: %v", p, d).debug(1) }
        } else if t.Type().Kind() == reflect.Bool && t.Bool() {
            noted(ctx, "%v: %v", p, d).debug(1)
        }
    }(time.Now())

    var y bool
    var g builtin_c
    var f builtin_x
    if f, y = bi.(builtin_x); !y {
        if g, y = bi.(builtin_c); !y {
            errostack(ctx, 3, "no method: (*%s).[cx](...) (%030b)", typeof(bi), w).debug(16)
            return
        }
    }

    if c := bv.Elem().FieldByName("Context"); !c.IsValid() {
        errostack(ctx, 3, "no field: %s.Context (%030b)", typeof(bi), w).debug(16)
        return // c.Type().String() == "smart.Context"
    } else { c.Set(reflect.ValueOf(ctx)) }

    if ic.o == nil {} else if t := _parseOpts(ctx, bv, plain, ic.o); t != nil {
        errostack(ctx, 3, "%v: unsupported opts: %v (%v, %030b)", p.name_, t, typeof(bv), w).debug(16)
        return
    }

    if false { if _, y := bi.(*builtin_or); y && len(ic.a) == 2 && ic.a[0].String() == "$3" && ic.a[1].String() == "$2" {
        defer func(a []Value) { noted(ctx, "%v: %v %v %v", p, ic.o, a, ic.a).debug(2) }(ic.a)
    }}

    var forth bool = w&expandUnexpandedForth != 0
    if !forth { if t, y := bi.(builtin_m); y && t.m()&builtinForth != 0 {
        // TODO: forth = true ???
    }}
    if f := bv.Elem().FieldByName("forth"); f.IsValid() && f.Kind() == reflect.Bool {
        if forth {
            if f.CanSet() { f.SetBool(true) } else {
                *(*bool)(unsafe.Pointer(f.UnsafeAddr())) = true
            }
        } else {
            if forth = f.Bool(); false && forth {
                w |= expandUnexpandedForth
            }
        }
    }

    if x, y := bi.(builtin_a); y {
        if x.a(ic, w|expandArgs) && !forth { return p }
    } else { var u int
        if ic.a, u, _ = (w|expandArgs).expand(ctx, ic.a...); u>0 && !forth { return p }
    }

    // FIXES unexpected expansion for autos on def-assigns.
    if w&expandDefAssign != 0 && p.name_ != "auto" {
        const ad = expandAuto|expandDigits
        var f func(v Value) (res bool)
        f = func(v Value) (res bool) {
            var args []Value
            switch t := v.(type) {
            case *list: for _, v := range t.Elems { if f(v) { return true } }
            case *delegate: if f(t.x) { return true } else { args = t.a }
            case *closure: if f(t.x) { return true } else { args = t.a }
            case unexpanded: return f(t.Value)
            case digital, *auto: return true
            }
            if false { for _, a := range args { if f(a) { return true } }}
            return
        }
        for _, a := range ic.a { if w&ad != 0 && a.expandable(ctx, ad) {
            if false && cast[*universe](ctx).db("builtin") { noted(ctx, "builtin: %T %v", a, a).debug(1) }
            if f(a) { return p }
        }}
    }

    if f != nil {
        if t := f.x(ic, w); t == f {
            noted(ctx, "%v: use skip{} instead: %T %v", p, typeof(t), t).debug(1)
            return p
        } else if _, y := t.(skip); y {
            return p
        } else {
            ic.x = true
            return ease(ctx, t)
        }
    } else if g != nil {
        if t := g.c(ic, w); t != nil {
            warnstack(ctx, 3, "discarded command result: %v", t).debug(5)
        }
        ic.x = true
    }
    return
}
func (p *builtin) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*builtin); y { assert(y, "value is not builtin")
        if /*p.f == a.f &&*/ p.name_ == a.name_ { res = cmpEqual }
    } else if l, y := v.(*list); y && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (_ *builtin) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *builtin) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *builtin) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type ruleClass int

const (
    GeneralRule ruleClass = iota
    PatternRule
    PathPattRule
)

var namesForRuleClass = []string{
    GeneralRule:  "GeneralRule",
    PatternRule:  "PatternRule",
    PathPattRule: "PathPattRule",
}

func (c ruleClass) String() string {
    var i = int(c)
    if 0 <= i && i < len(namesForRuleClass) {
        return namesForRuleClass[i]
    }
    return fmt.Sprintf("ruleClass(%d)", i)
}

type ruleContext struct { Context ; rule *rule }
func (ec *ruleContext) Position() Position { return ec.rule.position }
func (ec *ruleContext) ruleContext() *ruleContext { return ec }
func (ec *ruleContext) entry() Entry { return ec.rule }
func (ec *ruleContext) inner() Context { return ec.Context }
func (ec *ruleContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("entry{%s,%s}", ec.rule, ec.Context)
    } else if true {
        var ( cc []*ruleContext; s string )
        for c := ec; c != nil && len(cc) < 5; c = c.Context.ruleContext() {
            if t := c.rule.string(c.Context); s != "" {
                s = fmt.Sprintf("%s{%s}", t, s)
            } else {
                s = t
            }
        }
        return s
    } else if s, t := ec.rule.string(ec.Context), ec.Context.String(); t != "" {
        return fmt.Sprintf("%s{%s}", s, t)
    } else {
        return fmt.Sprintf("%s", s)
    }
}
// func (ec *ruleContext) stems() (stems []string) {
//     if sc, ok := ec.Context.(*stemmedContext); ok {
//         stems = sc.stem.Stems // only if the inner is stemmed
//     }
//     return
// }

func isInnerAuto(ctx Context, target Value) (res bool) {
    if ac, n := ctx.ac(), 0; ac != nil {
        for ac = ac.inner().ac(); ac != nil; ac = ac.inner().ac() {
            if n > 1 { return true }
            if t := autoVal(ac, "@"); t != nil && eq(ctx, t, target) { n += 1 }
        }
    }
    return
}

type invoker interface { invoke(Context, facet, []Value, []Value) Value }
type executer interface { execute(Context, ...Value) ([]Value, travestates) }
type Entry interface {
    Object
    executer

    Class() ruleClass
    Target() Value // pattern or concrete target

    programs() []*program
    setPrograms([]*program)

    hasRecipes() bool

    option(Context) (bool, []Value)
    execute(Context, ...Value) ([]Value, travestates)
}

type resolvedEntries struct { Entry ; all []Entry }
func (p *resolvedEntries) String() string { return fmt.Sprintf("%s(%d)", p.Entry, len(p.all)) }
func (p *resolvedEntries) add(entry Entry) {
    if p.Entry == nil { p.Entry = entry }
    p.all = append(p.all, entry)
}
func (p *resolvedEntries) join(o *resolvedEntries) {
    if p.Entry == nil { p.Entry = o.Entry }
    p.all = append(p.all, o.all...)
}
func (p *resolvedEntries) programs() (programs []*program) {
    for _, entry := range p.all {
           programs = append(programs, entry.programs()...)
    }
    return
}

// rule represents a declared rule entry.
type rule struct {
    class ruleClass
    target Value
    arged []Value // for restriction/filter
    program_ []*program
    position Position
}
func (_ *rule) kind() Kind { return KindObject|KindRule }
func (entry *rule) Target() Value { return entry.target }
func (entry *rule) Class() ruleClass { return entry.class }
func (entry *rule) programs() []*program { return entry.program_ }
func (entry *rule) declScope() *Scope { return entry.OwnerProject().scope }
func (entry *rule) OwnerProject() *Project { return entry.program_[0].project }
func (entry *rule) setPrograms(programs []*program) { entry.program_ = programs }
func (entry *rule) setPosition(position Position) { entry.position = position }
func (entry *rule) setTarget(v Value) { entry.target = v }
func (entry *rule) Position() (pos Position) {
    if pos = entry.position; !pos.IsValid() {
        if pos = entry.target.Position(); !pos.IsValid() {
            for _, prog := range entry.program_ {
                if pos = prog.position; pos.IsValid() { break }
            }
        }
    }
    return
}
func (entry *rule) name(ctx Context) (name string) {
    if entry == nil {
        erro(ctx, "nil entry")
    } else if entry.target == nil {
        erro(at(ctx,entry.position), "entry target is nil")
    } else {
        name = entry.target.string(ctx)
    }
    return
}
func (entry *rule) true(ctx Context) bool { return entry.target.true(ctx) }
func (entry *rule) float(_ Context) (f float64, _ error) { return 0, nil }
func (entry *rule) int(_ Context) (i int64, _ error) { return 0, nil }
func (entry *rule) string(ctx Context) string { return entry.target.string(ctx) }
func (entry *rule) String() string {
    if entry.target == nil { return "<nil entry>" }
    return entry.target.String()
}
func (entry *rule) updated(ctx Context) (res bool) {
    if res = entry.target.updated(ctx); res {
        ctx.dirtyMark(entry.target)
    }
    return
}
func (entry *rule) updatedDeps(ctx Context, v ...Value) []Value {
    return entry.target.updatedDeps(ctx, v...)
}
// rule.Execute executes the rule program only if the target is outdated.
func (entry *rule) execute(ctx Context, a ...Value) (result []Value, traves travestates) {
    switch entry.class {
    case PatternRule, PathPattRule:
        erro(ctx, "executing pattern entry '%v'", entry.target).debug(1)
    default:
        result, traves = entry.exec(ctx, a...)
    }
    return
}
func (entry *rule) exec(cc Context, a ...Value) (result []Value, traves travestates) {
    if cc = (&ruleContext{ cc, entry }); len(a)>0 { cc = &argumentedContext{ cc, a } }
ForPrograms:
    for _, program := range entry.program_ {
        var pos = program.position
        if !pos.IsValid() { pos = entry.Position() }

        var res, t = program.execute(at(cc, pos))
        result = append(result, merge(res)...)
        traves = append(traves, t...)
        if t.has(traveFail) { break ForPrograms }
        for _, s := range t.of(traveCase, traveDone) {
            if s.prog == program { break ForPrograms }
        }
    }
    return
}
func (entry *rule) Get(_ Context, name string) (Value, error) {
    switch name {
    case "class": return makeStrlit(entry.position, entry.class.String()), nil
    case "name" : return entry.target, nil //return makeStrlit(entry.position, entry.name()), nil
    //case "prerequisites": ...
    }
    return nil, fmt.Errorf("no such entry property (%s)", name)
}
func (entry *rule) rescope(ctx Context, scope *Scope) { panic("rule.rescope not supported") }
func (entry *rule) recipes() (recipes []Value) {
    for _, prog := range entry.program_ {
        for _, recipe := range prog.recipes {
            recipes = append(recipes, recipe)
        }
    }
    return
}
func (entry *rule) hasRecipes() (res bool) {
    for _, prog := range entry.program_ {
        if res = len(prog.recipes) > 0; res { break }
    }
    return
}
func (entry *rule) refs(ctx Context, v Value) bool {
    if entry.target.refs(ctx, v) { return true }

    return false

    for _, prog := range entry.program_ {
        for _, depend := range prog.depends {
            if depend.refs(ctx, v) { return true }
        }
        for _, recipe := range prog.recipes {
            if recipe.refs(ctx, v) { return true }
        }
    }
    return false
}
func (entry *rule) defs(ctx Context, s ...string) (res []*def) {
    res = entry.target.defs(ctx, s...)
    for _, prog := range entry.program_ {
        for _, depend := range prog.depends {
            res = append(res, depend.defs(ctx, s...)...)
        }
        for _, recipe := range prog.recipes {
            res = append(res, recipe.defs(ctx, s...)...)
        }
    }
    return
}
func (entry *rule) expandable(ctx Context, w facet) (res bool) {
    if res = entry.target.expandable(ctx, w); res { return }
    if false {
        for _, prog := range entry.program_ {
            for _, depend := range prog.depends {
                if res = depend.expandable(ctx, w); res { return }
            }
            for _, recipe := range prog.recipes {
                if res = recipe.expandable(ctx, w); res { return }
            }
        }
    }
    return
}
func (entry *rule) expand(ctx Context, w facet) (res Value) {
    if entry == nil { // This happens from some &{xxx} exprs
        errostack(ctx, 3, "expand nil entry (w=%030b)", w).debug(16)
        return
    }

    if w&expandEvoke != 0 {
        if ic := ctx.ic(); ic == nil {
            erro(ctx, "not an invocation (w=%030b)", w).debug(1)
        } else if reses, t := entry.execute(ctx, ic.a...); reses != nil {
            if t.has(traveFail) { for _, s := range t { erro(at(ctx,s.pos), "%v", s) }
                errostack(ctx, 3).debug(3) }
            res, ic.x = ease(ctx, reses), true
        } else { ic.x = true }
        return
    }

    var target Value
    if target = entry.target.expand(ctx, w); target != entry.target {
        // TODO: test if programs are needed to be disclosed??
        res = &rule{
            entry.class, target,
            entry.arged,
            entry.program_,
            entry.position,
        }
    } else {
        res = entry
    }
    return
}
func (entry *rule) delete(  ctx Context) (files []*File, err error) { return entry.target.delete(ctx) }
func (entry *rule) stamp(   ctx Context) (files []*File, err error) { return entry.target.stamp(ctx) }
func (entry *rule) traverse(ctx Context) {
    var pc = ctx.pc()
    var sc, _ = ctx.(*stemmedContext)
    var target = autoVal(ctx, "@")

    if target == nil {
        erro(ctx, "$@ is not defined").debug(1)
        return
    } else if ctx.entry() == entry {
        var proj = ctx.Project()

        if c := ctx.closure(); c != nil {
            if t := autoVal(c, "@"); t != nil && eq(ctx, t, target) {
                if true { warn(ctx, "%v: %v: %v\n", proj, entry, t) }
                // FIXES: skip traversal as it's closure, for example:
                //
                //   %.h($(headers)): $(srcinc)/%.h update-file
                //
                // where the 'update-file' is like:
                //
                //   update-file: [((in)) (closure) (set @=&@)] $(in) \
                //       [(read-file $>) (update-file -p)]
                //
                // see also program.execute for the same skip.
                return
            }
        }

        prompt(ctx, "%v: %v: %v\n", proj, entry, target)
        warnstack(ctx, 8, "%v: %v: %v", proj, entry, target).debug(16)
    } else {
        ctx = &ruleContext{ ctx, entry }
    }

ForPrograms:
    for i, prog := range entry.program_ {
        var ctx = at(ctx, prog.position)
        var v, t = prog.execute(ctx)

        if true && sc != nil && sc.stem.target.string(ctx) == "Unwind-EHABI.o" {
            info(ctx, "rule.traverse: %v: %d: %v %v", target, i, t, v)
        }

        if a := t.of(traveFail); a.has() {
            erro(ctx, "%v: %v", target, a).debug(1)
            return
        } else if t.has(traveNext) {
            continue ForPrograms
        }

        if pc != nil && v != nil {
            pc.values = append(pc.values, merge(v)...)
        }
        if pc != nil && sc != nil {
            var s = pc.traves.add(ctx, traveRule, target)
            s.depend = entry
            s.prog = prog
        }

        for _, s := range t.of(traveCase, traveDone) {
            if s.prog == prog { break ForPrograms }
        }
    }

    if pc != nil && sc == nil {
        // if sc != nil { depend = sc.stem } else { depend = entry }
        pc.traves.add(ctx, traveRule, target).depend = entry
    }
    return
}
// FIXME: entry.target maybe not the real target
func (entry *rule) stat(ctx Context) (si *statinfo) { return entry.target.stat(ctx) }
func (entry *rule) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*rule); ok {
        assert(ok, "value is not rule")
        if /*entry.class == a.class &&*/ entry.target.cmp(ctx, a.target) == cmpEqual {
            if entry.OwnerProject() == a.OwnerProject() {
                res = cmpEqual
            }
        }
    } else if l, ok := v.(*list); ok && len(l.Elems) == 1 {
        res = entry.cmp(ctx, l.Elems[0])
    }
    return
}
func (entry *rule) patterned(ctx Context) bool { return entry.target.patterned(ctx) }
func (entry *rule) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = entry.target.match(ctx, i)
    return
}
func (entry *rule) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = entry.target.stencil(ctx, stems)
    return
}

func (entry *rule) option(ctx Context) (res bool, infos []Value) {
    ForPrograms: for _, program := range entry.program_ {
        if !program.configure { continue }
        for _, depend := range program.depends {
            g, ok := depend.(*modification)
            if!ok { continue }
            for _, m := range g.list {
                if m.Elems[0].string(ctx) != "configure" { continue }
                for _, arg := range m.Elems[1:] {
                    a, ok := arg.(*argumented)
                    if!ok { continue }
                    f, ok := a.Value.(flag)
                    if!ok { continue }
                    if f.Value.string(ctx) != "option" { continue }
                    for _, v := range a.args {
                        if p, ok := v.(*pair); ok {
                            if p.Key.string(ctx) != "info" { continue }
                            v = p.Value
                        }
                        infos = append(infos, v)
                    }
                    res = true
                    break ForPrograms
                }
            }
        }
    }
    return
}

func (_ *rule) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *rule) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *rule) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type stemmedContext struct {
    Context
    stem *stemmed
}
func (sc *stemmedContext) inner() Context { return sc.Context }
func (sc *stemmedContext) sc() *stemmedContext { return sc }
func (sc *stemmedContext) stems() []string { return sc.stem.stems }
func (sc *stemmedContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("stemmed{%s}", sc.Context)
    } else {
        return sc.Context.String()
    }
}

type stemmed struct {
    *rule
    target Value
    stems []string
}
func (p *stemmed) kind() Kind { return p.rule.kind()|KindStemmedRule }
func (p *stemmed) Target() Value { return p.target }
func (p *stemmed) String() (s string) {
    for i, stem := range p.stems { if i > 0 { s += "," }; s += stem }
    return fmt.Sprintf("%s:%s", p.target, s) // "<%s:%s>"
}
func (p *stemmed) expand(ctx Context, w facet) (res Value) {
    if v := p.rule.expand(ctx, w); v != p.rule {
        res = &stemmed{v.(*rule), p.target, p.stems}
    } else {
        res = p
    }
    return
}
func (p *stemmed) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*stemmed); ok {
        assert(ok, "value is not stemmed")
        if len(p.stems) != len(p.stems) { return }
        for i, stem := range p.stems {
            if stem != a.stems[i] { return }
        }
        res = p.rule.cmp(ctx, a.rule)
    } else if l, ok := v.(*list); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (p *stemmed) traverse(ctx Context) {
    // NOTE: Make a clone of the underlying rule for traversing the real target;
    //       the underlying rule target is readonly, it must not be changed, for
    //       next traversal be done correctly.
    var t = *p.rule // TODO: consider not copying the rule, use pointer instead
    t.setTarget(p.target)
    t.traverse(&stemmedContext{ ctx, p })
}
