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
        "bytes"
        // "unsafe"
        "time"
        "fmt"
)

type Object interface {
        Value

        DeclScope() *Scope
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
func (_ *objbase) Kind() Kind { return KindObject }
func (p *objbase) DeclScope() *Scope { return p.scope }
func (p *objbase) OwnerProject() *Project { return p.owner }
func (p *objbase) String() string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) Strval(ctx Context) string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) Name(ctx Context) string { panic("inquiring name of an unknown object") }
func (p *objbase) Get(_ Context, name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p *objbase) rescope(_ Context, scope *Scope) { panic("rescoping unknown object") }
func (p *objbase) exists() existence { return existenceMatterless }

type knownobject struct { // generally named objects
        objbase
        name string // single, or group name if containing '(*)' and corresponding members
        //members [][]string
}
func (_ *knownobject) Kind() Kind { return KindObject|KindKnownObject }
func (p *knownobject) String() string { return fmt.Sprintf("{object %s}", p.name) }
func (p *knownobject) Strval(_ Context) string { return fmt.Sprintf("{object %s}", p.name) }
func (p *knownobject) True(_ Context) bool { return true }
func (p *knownobject) Name(ctx Context) string { return p.name }
func (p *knownobject) rescope(_ Context, scope *Scope) {
        if p.scope != scope {
                if p.scope != nil {
                        delete(p.scope.elems, p.name)
                }
                if p.scope = scope; p.scope != nil {
                        p.scope.elems[p.name] = p
                }
        }
}
func (p *knownobject) expand(_ Context, _ facet) Value { return p }
func (p *knownobject) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*knownobject); ok {
                assert(ok, "value is not knownobject")
                if p.owner == a.owner && p.scope == a.scope && p.name == a.name {
                        res = cmpEqual
                }
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
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
func (_ unresolved) Kind() Kind { return KindObject|KindUnresolved }
func (p unresolved) Name(ctx Context) (name string) {
        if p.Value == nil {
                erro(at(ctx,p.Position()), "unresolved object name is nil")
        } else if ctx == nil {
                name = p.Value.String()
        } else {
                name = p.Value.Strval(ctx)
        }
        return
}
func (p unresolved) Position() Position { return p.Value.Position() }
func (p unresolved) String() string { return p.Value.String() }
func (p unresolved) Strval(_ Context) (s string) { return "" }
func (p unresolved) Float(_ Context) (float64, error) { return 0.0, nil }
func (p unresolved) Integer(_ Context) (int64, error) { return 0, nil }
func (p unresolved) True(_ Context) bool { return false }
func (p unresolved) refs(_ Context, _ Value) (res bool) { return }
func (p unresolved) defs(_ Context, _ ...string) (res []*def) { return }
func (p unresolved) patterned(_ Context) bool { return false }
func (p unresolved) invoke(_ Context, _ facet, _, _ []Value) Value { return p }
func (p unresolved) Get(_ Context, name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p unresolved) Execute(ctx Context, a ...Value) (result []Value, err error) { return []Value{p}, nil }
func (p unresolved) OwnerProject() *Project { return p.project }
func (p unresolved) DeclScope() *Scope { return p.project.scope }
func (p unresolved) rescope(ctx Context, scope *Scope) {
        if true {
                fail(p.Value.Position(), "cant rescope a unresolved object")
        } else if p.project != scope.project {
                var name = p.Value.Strval(ctx)
                if p.project.scope != nil { delete(p.project.scope.elems, name) }
                if p.project = scope.project; p.project.scope != nil {
                        p.project.scope.elems[name] = p
                }
        }
}
func (p unresolved) cmp(ctx Context, v Value) (res cmpres) {
        if a, y := v.(unresolved); y {
                res = p.Value.cmp(ctx, a.Value)
        } else if l, y := v.(*List); y && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        } else if u, y := v.(unexpanded); y && u.Value != nil {
                res = p.cmp(ctx, u.Value)
        }
        return
}
func (p unresolved) expandable(_ Context, _ facet) bool { return false }
func (p unresolved) expand(ctx Context, w facet) (res Value) {
        var db bool
        if true { if w&expandDebug != 0 { defer func() {
                var t = p.Value.expand(ctx, w&^expandInvoke)
                w.noted(ctx, p, p.Value, ctx.ic().a)
                if false { infostack(of(ctx,p), 3) }
                noted(ctx, "%v: %v %v ⇒ %v %v", p, typeof(p.Value), p.Value, typeof(t), t)
                noted(ctx, "%v: %v %v (same=%v)", p, typeof(res), res, (res == p)).debug(24)
        }(); db = true ; w |= expandDebug }}

        v := p.Value.expand(ctx, w)

        if w&expandInvoke == 0 { if v != nil && v != p.Value {
                return unresolved{v, p.project}
        } else {
                return p
        }}

        ic := ctx.ic()

        // Always expand invocation args
        if ic != nil && ic.a != nil {
                a, _, _ := (w|expandAuto|expandArgs).expand(ctx, ic.a...)
                if db { noted(ctx, "%v: %v ⇒ %v", p, ic.a, a).debug(1) }
                ic.a = a
        }

        if u, y := v.(unexpanded); y { if u.Value != v {
                return unresolved{u.Value,p.project}
        } else {
                return p
        }}

        // TODO: only if w&expandResolve != 0 ...

        name := v.Strval(ctx)
        if name == "" { return p }

        o := p.project.resolveObject(ctx, name)
        if o == nil { return p } else { res = o.expand(ctx, w|expandAuto|expandDigits) }
        if db { noted(ctx, "%v: %v ⇒ %v ⇒ %v ; %v", p, name, o, res, ic.a).debug(1) }
        return
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

type self struct { projectname }
func (_ *self) Kind() Kind { return KindObject|KindSelf|KindProjectName }
func (_ *self) String() string { return ".self" }
func (p *self) Name(_ Context) string { return p.String() }
func (p *self) expand(_ Context, _ facet) Value { return expanded{/* &p.projectname */p} }

type projectname struct { *Project ; scope *Scope }
func (_ *projectname) Kind() Kind { return KindObject|KindProjectName }
func (_ *projectname) Integer(_ Context) (int64, error) { return 0, nil }
func (_ *projectname) Float(_ Context) (float64, error) { return .0, nil }
func (_ *projectname) updated(_ Context) bool { return false }
func (_ *projectname) updatedDeps(_ Context, _ ...Value) []Value { return nil }
func (_ *projectname) stamp(ctx Context) (files []*File, err error) { return }
func (_ *projectname) delete(_ Context) (files []*File, err error) { return }
func (_ *projectname) defs(_ Context, _ ...string) (res []*def) { return }
func (_ *projectname) refs(_ Context, _ Value) (res bool) { return }
func (_ *projectname) patterned(_ Context) bool { return false }
func (_ *projectname) expandable(_ Context, _ facet) bool { return false }
func (p *projectname) stencil(ctx Context, stems []string) (Value, []string) { return p, stems }
func (p *projectname) match(ctx Context, i interface{}) (bool, interface{}, []string) { return matchStrval(ctx, p, i) }
func (p *projectname) Position() Position { return p.position }
func (p *projectname) String() string { return p.name }
func (p *projectname) Strval(_ Context) string { return p.name }
func (p *projectname) Name(_ Context) string { return p.name }
func (p *projectname) True(_ Context) bool { return p.Project != nil }
func (p *projectname) DeclScope() *Scope { return p.scope }
func (p *projectname) OwnerProject() *Project { return p.scope.project }
func (p *projectname) Get(ctx Context, name string) (Value, error) { return p.resolveObject(ctx, name), nil }
func (p *projectname) expand(_ Context, _ facet) (res Value) { return expanded{p} }
// func (p *projectname) invoke(_ Context, _ facet, _, _ []Value) Value { return p }
func (p *projectname) traverse(ctx Context) {
        if t := p.Project.defaultEntry; t != nil {
                switch t.Target().(type) {
                case *Flag: return
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
        } else if l, y := v.(*List); y && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        }
        return
}
func (p *projectname) rescope(_ Context, scope *Scope) {
        if p.scope != scope {
                if p.scope != nil {
                        delete(p.scope.elems, p.name)
                }
                if p.scope = scope; p.scope != nil {
                        p.scope.elems[p.name] = p
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

type scopename struct { *Scope ; name string }
func (_ *scopename) Kind() Kind { return KindObject|KindScopeName }
func (_ *scopename) Integer(_ Context) (int64, error) { return 0, nil }
func (_ *scopename) Float(_ Context) (float64, error) { return .0, nil }
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
func (p *scopename) match(ctx Context, i interface{}) (bool, interface{}, []string) { return matchStrval(ctx, p, i) }
func (p *scopename) stencil(ctx Context, stems []string) (Value, []string) { return p, stems }
func (p *scopename) Position() Position { return p.position }
func (p *scopename) String() string  { return fmt.Sprintf("{scope %s}", p.name) }
func (p *scopename) Strval(_ Context) string { return p.name }
func (p *scopename) Name(_ Context) string { return p.name }
func (p *scopename) True(_ Context) bool { return p.Scope != nil }
func (p *scopename) OwnerProject() *Project { return p.Scope.project }
func (p *scopename) DeclScope() *Scope { return p.Scope.outer }
// func (p *scopename) invoke(_ Context, _ facet, _, _ []Value) Value { return p }
func (p *scopename) expand(_ Context, _ facet) (res Value) { return p }
func (p *scopename) Get(ctx Context, name string) (value Value, err error) {
        if s := p.Resolve(name); s != nil { if value, _ = s.(Value); value == nil {
                err = fmt.Errorf("`%s' in scope is invalid (%T)", name, s)
        }} else {
                err = fmt.Errorf("undefined `%s' in scope `%s'", name, p.name)
        }
        return
}
func (p *scopename) cmp(ctx Context, v Value) (res cmpres) {
        if a, y := v.(*scopename); y {
                if p.Scope == a.Scope { res = cmpEqual }
        } else if l, y := v.(*List); y && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        }
        return
}
func (p *scopename) rescope(_ Context, scope *Scope) {
        if p.Scope != scope {
                if p.Scope != nil {
                        delete(p.Scope.elems, p.name)
                }
                if p.Scope = scope; p.Scope != nil {
                        p.Scope.elems[p.name] = p
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
func (ac *autoContext) closureResolveAuto(name string) (obj Object, found bool) {
        if cc, ok := ac.Context.(*closureContext); ok {
                obj, found = cc.closureResolveAuto(name)
        } else if /*ac.Scope() == scope*/true {
                ac.Lock()
                obj, found = ac.defs[name]
                ac.Unlock()
        }
        if false && obj == nil { if o := ac.Scope().FindDef(name); o != nil {
                obj, found = o, true
        }}
        return
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
        if res == nil && ac.Context != nil {
                if false { for ic := ac.Context.ac(); ic != nil; ic = ic.ac() { if ic == ac {
                        errostack(ctx, 3, "deadloop: %v", name).debug(32)
                        return
                }}}
                if t := ac.Context.ac(); t != nil {
                        if ac == t {
                                errostack(ctx, 3, "deadloop: %v", name).debug(32)
                                return
                        }
                        res = t.get(ctx, name)
                }
        }
        if false && res == nil { warn(ctx, "undefined: %v in %v", name, ctx).debug(32) }
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
func (ac *autoContext) args(ctx Context, params []*def, args []Value) (names []string, err error) {
        var (
                argnum int // setup named/number parameters ($1, $2, etc.)
                compactArgs []Value // compacted args: combine duplicated pairs
                namedParam = func(name string) (res bool) {
                        for _, param := range params {
                                if res = param.name == name; res { break }
                        }
                        return
                }
        )
        if false { for i, a := range args { if w := expandAuto|expandDigits|expandDelegate; a.expandable(ctx, w) {
                t, d := a.expand(ctx, w|expandDebug), autoDef(ctx, "1")
                noted(of(ctx,a), "%v. %v %v -> %v %v, %v", i, typeof(a), a, typeof(t), t, d).debug(1)
        }}}

        outer: for _, a := range args {
                if p, y := a.(*Pair); y { for _, ca := range compactArgs {
                        if c, y := ca.(*Pair); y && eq(ac, p.Key, c.Key) {
                                var vals = merge(p.Value)
                                if l, y := c.Value.(*List); y {
                                        l.Elems = append(l.Elems, vals...)
                                } else {
                                        c.Value = MakeList(c.Position(), append(merge(c.Value), vals...)...)
                                }
                                continue outer
                        }
                }}
                compactArgs = append(compactArgs, a)
        }
        for _, a := range compactArgs {
                var (
                        id = strconv.Itoa(argnum+1)
                        name string
                )
                //<!IMPORTANT: Don't translate Flag, Flag values are valid regular arguments.
                //             Pair values are special.
                if a = scalarize(a); false && (isNull(a) || isNone(a)) {
                        erro(of(ac,a), "%T '%v' is invalid scalar", a, a).debug(1)
                        return
                } else if p, y := a.(*Pair); y { if s := p.Key.Strval(ac); namedParam(s) {
                        name, a = s, p.Value
                } else if false {
                        for _, param := range params { warn(ac, "%v: %v", s, param) }
                        warn(ac, "%s: %T %v", s, a, a).debug(1)
                }}

                if name != "" {
                        // Got the name!
                } else if argnum < len(params) {
                        name = params[argnum].name
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

        if false { for i := argnum+1; i < 9; i += 1 {
                var id = strconv.Itoa(i)
                ac.set(ctx, id, nil)
        }}
        return
}
func autoVal(ctx Context, name string) (res Value) {
        if d := autoDef(ctx, name); d != nil { res = d.value }
        return
}

func autoDef(ctx Context, name string) (d *def) {
        if a := ctx.ac(); a != nil { d = a.get(ctx, name) } else if false {
                if o, y := ctx.closureResolveAuto(name); y { d, y = o.(*def) }
        }
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
func (_ *auto) Kind() Kind { return KindObject|KindAuto }
func (a *auto) String() (s string) { return a.name }
func (a *auto) Strval(ctx Context) (res string) {
        if d := a.def(ctx); d != nil && d.value != nil { res = d.value.Strval(ctx) }
        return
}
func (a *auto) Get(ctx Context, name string) (res Value, _ error) {
        if name == "value" { res = autoVal(ctx, a.name) }
        return
}
func (a *auto) refs(ctx Context, v Value) (res bool) {
        if o, y := v.(*auto); y && (o == a || o.name == a.name) { return true }
        if val := autoVal(ctx, a.name); val != nil { res = val.refs(ctx, v) }
        return
}
func (a *auto) defs(ctx Context, s ...string) (res []*def) {
        if val := autoVal(ctx, a.name); val != nil { res = val.defs(ctx, s...) }
        return
}
func (a *auto) def(ctx Context) *def { return autoDef(ctx, a.name) }
func (a *auto) set(ctx Context, value Value, app ...Value) {
        if value == nil && app != nil { if d := autoDef(ctx, a.name); d != nil {
                d.value = ease(ctx, append(merge(d.value), app...))
                d.position = a.position
                return
        }}

        d, _ := autoSet(ctx, a.name, ease(ctx, append(merge(value), app...)))
        if d != nil { d.position = a.position } else if true {
                warnstack(of(ctx,a), 3, "set auto failed: %v: %v %v", a.name, value, app).debug(16)
        }
}
func (a *auto) expandable(ctx Context, w facet) (res bool) {
        if w&expandAuto != 0 && w&expandAutoKept == 0 { res = true } else
        if w&expandUnexpandedForth != 0 { if d := autoDef(ctx, a.name); d != nil {
                res = d.expandable(ctx, w)
        }}
        return
}
func (a *auto) expand(ctx Context, w facet) (res Value) {
        if true { if w&expandDebug != 0 { if d := autoDef(ctx, a.name); true/* d != nil && d.value != res */ { defer func() {
                if true { w.noted(ctx, a, d) }
                noted(ctx, "%v ⇒ %v %v (same=%v)", a, typeof(res), res, (res==a)).debug(10)
        }()}}}
        g := w&expandAuto != 0 && w&expandAutoKept == 0 || w&expandDefDefArgs != 0 || w&expandUnexpandedForth != 0
        if g && !ctx.un(ctx, a) { if d := autoDef(ctx, a.name); d != nil {
                var recured bool
                if ic := ctx.ic().Context.ic(); ic != nil && ic.in(a) { if false { w.noted(ctx, a) }
                        noted(ctx, "recursive: %v ⇒ %v ; %v", a, d, autoDef(ctx.ac().Context, a.name)).debug(5)
                        recured = true
                }
                if d.value == nil {
                        return unexpanded{a}
                } else if recured || d.value.refs(ctx, a) {
                        if false { noted(ctx, "nested: %v ⇒ %v (%v)", a, d, recured).debug(16) }
                        return d.expand(&unexpandContext{ctx,a}, w)
                } else {
                        return d.expand(ctx, w)
                }
        }}
        return unexpanded{a}
}
func (a *auto) invoke(ctx Context, w facet, o, v []Value) (res Value) {
        if d := autoDef(ctx, a.name); d != nil { res = d.invoke(ctx, w, o, v) }
        return
}
func (a *auto) cmp(ctx Context, v Value) (res cmpres) {
        if val := autoVal(ctx, a.name); val != nil { res = val.cmp(ctx, v) }
        return
}
func (a *auto) stat(ctx Context) (si *statinfo) {
        if val := autoVal(ctx, a.name); val != nil { si = val.stat(ctx) }
        return
}
func (a *auto) traverse(ctx Context) {
        if val := autoVal(ctx, a.name); val != nil { val.traverse(ctx) }
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

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type def struct {
        knownobject // mutex sync.Mutex
        origin Origin
        value Value
}
func (_ *def) Kind() Kind { return KindObject|KindDef }
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
        default:         s =   "⇒"
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
        if s = d.name + d.streq(); value != nil {
                s += elemstr(nil, d, value, 0)
        } else {
                s += "<nil>"
        }
        return
}
func (d *def) Strval(ctx Context) (res string) {
        var val Value
        {
                // d.mutex.Lock()
                val = d.value
                // d.mutex.Unlock()
        }
        if val != nil { res = val.Strval(ctx) }
        return
}
func (d *def) True(ctx Context) (res bool) {
        var val Value
        {
                // d.mutex.Lock()
                val = d.value
                // d.mutex.Unlock()
        }
        if val != nil { res = val.True(ctx) }
        return
}
func (d *def) refs(ctx Context, v Value) (res bool) {
        if o, y := v.(*def); y { if d == o { return true }}

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
                        if d.name == a {
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
func (d *def) expand(ctx Context, w facet) (res Value) {
        var db bool
        if false { if w&expandDebug != 0 { defer func(w0 facet) {
                if d.value != nil { ctx = of(ctx, d.value) } else { ctx = of(ctx, d) }
                if true { w0.noted(ctx, d) }
                var a []Value ; if i := ctx.ic(); i != nil { a = i.a }
                noted(ctx, "%v: %v %v", d, d.origin, a)
                noted(ctx, "%v: %v %v", d.name, typeof(d.value), d.value)
                noted(ctx, "%v: %v %v (same=%v)", d.name, typeof(res), res, (res==d.value)).debug(16)
        }(w); db = true }}

        if w&expandInvoke == 0 {
                if false { warnstack(ctx, 3, "def.expand: invalid (%030b)", w).debug(16) }
                return d
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

        if false {
                var recured bool
                if ic := ctx.ic().Context.ic(); ic != nil && ic.in(d) { if false { w.noted(ctx, d) }
                        noted(ctx, "recursive: %v", d).debug(5)
                        recured = true
                }
                if recured || (d.value != nil && d.value.refs(ctx, d)) {
                        if false { noted(ctx, "nested: %v (%v)", d, recured).debug(16) }
                        ctx = &unexpandContext{ctx,d}
                }
        }

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
        } else if l, y := v.(*List); y && len(l.Elems) == 1 {
                res = d.cmp(ctx, l.Elems[0])
        }
        return
}
func (d *def) elemstr(_ Context, o Object, k elembits) (s string) {
        if o != nil {
                if p := d.OwnerProject(); p != o.OwnerProject() {
                        return fmt.Sprintf("$(%s->%s)", p.name, d.name)
                }
        }
        s = fmt.Sprintf(`$(%s)`, d.name)
        return
}
func (d *def) val(ctx Context, value Value) { d.set(ctx, d.origin, value) }
func (d *def) set(ctx Context, origin Origin, value Value, app... Value) {
        var pos = d.position
        if value == nil {
                // NOTE: will append values iif len(app) > 0
        } else if d.value == value {
                if d.origin != origin { d.origin = origin }
                return
        }

        if !pos.IsValid() {
                if value != nil { pos =  value.Position() } else
                if len(app) > 0 { pos = app[0].Position() }
        }

        var vals []Value
        if value != nil { vals = merge(value) }
        if   app != nil { vals = append(vals, app...) }

        for i, val := range vals { if o, y := val.(*def); y {
                if false {
                        // Appending Def value is not recommended, but if it does, we make
                        // a warning here to give a chance for further optimization.
                        warn(ctx, "use def as value: %v", o)
                        warn(ctx, "%v: %v", d.origin, d)
                        warnstack(ctx, 5).debug(16)
                }
                vals[i] = o.value // replace defs
        }}

        switch w := plain|expandAuto/* &^expandOptimal */; origin {
        case DefExpand1: vals, _, _ = (w&^expandClosure).expand(ctx, vals...) //  :=
        case DefExpand2: vals, _, _ = (w| expandClosure).expand(ctx, vals...) // ::=
        default: for _, val := range vals { if val != nil && val.refs(ctx, d) { ctx = at(ctx, pos)
                if options.verbose { prompt(ctx, "set %s (%v): %v\n", origin, d.name, val) }
                if options.debug { warn(ctx, "from here").debug(1) }
                errostack(ctx, 3, "value refers to assigning Def '%s': %v (%T)", d.name, val, val).debug(10)
                return
        }}}

        if value == nil { if len(app) > 0 {
                // d.mutex.Lock()
                if !isTrivial(d.value) { vals = append(merge(d.value), vals...) }
                // d.mutex.Unlock()
        } else if origin != DefExecute {
                // d.mutex.Lock()
                d.origin, d.value = origin, MakeNone(d.position)
                // d.mutex.Unlock()
                return
        }}

        // d.mutex.Lock()
        d.origin, d.value = origin, ease(ctx, vals)
        // d.mutex.Unlock()
        return
}
func (d *def) append(ctx Context, va... Value) {
        if len(va) > 0 { d.set(ctx, d.origin, nil, va...) }
}
func (d *def) invoke(ctx Context, w facet, o, a []Value) (res Value) {
        return invoke(ctx, d, w, o, a)
}
func (d *def) xauto(ctx Context, w facet, a ...Value) (res Value) {
        if false { if ctx.Project().name == "testforeach1" && (d.name == ".test.x") {
                w.noted(ctx, d)
                noted(ctx, "%v - %v", d, a).debug(10)
                defer func() { noted(ctx, "%v: %v: %v", d.name, typeof(res), res).debug(10) } ()
        }}
        if len(a)>0 {
                ac := autoContext{ Context:ctx, defs:make(autoDefMap) }
                ac.args(ctx, nil, a)
                w |= expandAuto|expandDigits
                ctx = &ac
        }

        // d.mutex.Lock()
        // var bits = d.bits
        // d.bits |= defUnavail
        res = d.value.expand(ctx, w)
        // d.bits = bits
        // d.mutex.Unlock()
        return
}
func (d *def) xexec(ctx Context, value Value, a... Value) (res Value) {
        if isTrivial(value) { return }

        var cmd string
        if cmd = value.Strval(ctx); cmd == "" {
                warn(ctx, "%v: empty command (value=%v)", d.name, value).debug(1)
                return
        }

        // TODO: options for running command in the specified container
        var (
                stdout, stderr bytes.Buffer
                sh = exec.Command("sh", "-c", cmd)
        )
        sh.Stdout, sh.Stderr = &stdout, &stderr
        if err := sh.Run(); err != nil {
                erro(ctx, "%v: execute command failed: %v", d.name, err)
                erro(ctx, "%v: execute command: %s", d.name, cmd).debug(2)
                stdout.Reset()
                stderr.Reset()
                return
        }

        var pos = value.Position()
        if !pos.IsValid() { pos = ctx.Position() }
        res = MakeString(pos, strings.TrimSpace(stdout.String()))
        stdout.Reset()
        stderr.Reset()
        return
}
func (d *def) Get(ctx Context, name string) (res Value, err error) {
        switch name {
        case "name" : res = MakeString(d.position, d.name)
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
func (_ *undetermined) Kind() Kind { return KindObject|KindUndetermined }
func (p *undetermined) Position() Position { return p.identifier.Position() }
func (p *undetermined) String() (s string) {
        s = p.identifier.String()
        s += p.tok.String()
        s += p.value.String()
        return
}
func (p *undetermined) Strval(ctx Context) string { return p.value.Strval(ctx) }
func (p *undetermined) Name(ctx Context) string { return p.identifier.Strval(ctx) }
func (p *undetermined) True(ctx Context) bool { return false }
func (p *undetermined) Float(ctx Context) (f float64, _ error) { return 0, nil }
func (p *undetermined) Integer(ctx Context) (i int64, _ error) { return 0, nil }
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
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
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

// A builtin represents a built-in function. builtins don't have a valid type.
type builtin struct { knownobject ; t reflect.Type }
func (_ *builtin) Kind() Kind { return KindObject|KindBuiltin }
func (p *builtin) String() string { return p.name }
func (p *builtin) True(_ Context) bool { return p.t != nil }
func (p *builtin) isCommand() bool { return reflect.PointerTo(p.t).Implements(builtin_c_t) }
func (p *builtin) invoke(ctx Context, w facet, o, a []Value) Value { return invoke(ctx, p, w, o, a) }
func (p *builtin) expand(ctx Context, w facet) (res Value) {
        if w&expandInvoke == 0 {
                if false { warnstack(ctx, 3, "builtin.expand: invalid (%030b)", w).debug(16) }
                return p
        }

        var ic = ctx.ic()
        if ic == nil {
                errostack(ctx, 3, "builtin.expand: nil delegate context (%030b)", w).debug(16)
                return p
        }

        bv := reflect.New(p.t)
        bi := bv.Interface()

        var y bool
        var g builtin_c
        var f builtin_x
        if f, y = bi.(builtin_x); !y { if g, y = bi.(builtin_c); !y {
                errostack(ctx, 3, "no method: (*%s).exp(...) (%030b)", typeof(bi), w).debug(16)
                return
        }} else if c := bv.Elem().FieldByName("Context"); !c.IsValid() {
                errostack(ctx, 3, "no field: %s.Context (%030b)", typeof(bi), w).debug(16)
                return // c.Type().String() == "smart.Context"
        } else { c.Set(reflect.ValueOf(ctx)) }

        if ic.o == nil {} else if t := _parseOpts(ctx, bv, plain, ic.o); t != nil {
                errostack(ctx, 3, "%v: unsupported opts: %v (%v, %030b)", p.name, t, typeof(bv), w).debug(16)
                return
        } else if false { if t, y := bi.(*builtin_wildcard); y {
                noted(ctx, "%v: %v %v", p, ic.o, t.dir).debug(1)
        }}

        var u int
        if x, y := bi.(builtin_a); y {
                if x.a(ic, w|expandArgs) { return p }
        } else if ic.a, u, _ = (w|expandArgs).expand(ctx, ic.a...); u>0 {
                return p
        }

        var t0 = time.Now()
        if f != nil { if t := f.x(ic, w); t == f { return p } else {
                res, ic.x = ease(ctx, t), true
        }} else if g != nil { if t := g.c(ic, w); true {
                if ic.x = true; t != nil {
                        warnstack(ctx, 3, "discarded command result: %v", t).debug(5)
                }
        }}
        if d := time.Now().Sub(t0); d > 1*time.Second {
                noted(ctx, "%v: slow: %v", p, d).debug(3)
        } else if t := bv.Elem().FieldByName("timing"); !t.IsValid() {
                if false { noted(ctx, "%v: %v", p, d).debug(1) }
        } else if t.Type().Kind() == reflect.Bool && t.Bool() {
                noted(ctx, "%v: %v", p, d).debug(1)
        }

        if false && res == nil { if t, y := bi.(*builtin_wildcard); y {
                var s string
                for i, a := range ic.a { if i>0 { s += ", " } ; s += a.String() }
                w.noted(ctx, p)
                noted(ctx, "%v: dir=%s", p, t.dir)
                noted(ctx, "%v: (%s) -> %v (%s)", p, s, res, typeof(res)).debug(8)
        }}

        // FIXME: panic: reflect.Value.Set using unaddressable value
        if false { bv.Set(reflect.Value{}) } else if false {
                // ptr := unsafe.Pointer(bv.Pointer())
                // runtime.Free(ptr)
        }
        return
}
func (p *builtin) cmp(ctx Context, v Value) (res cmpres) {
        if a, y := v.(*builtin); y { assert(y, "value is not builtin")
                if /*p.f == a.f &&*/ p.name == a.name { res = cmpEqual }
        } else if l, y := v.(*List); y && len(l.Elems) == 1 {
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

type RuleClass int

const (
        GeneralRule RuleClass = iota
        PatternRule
        PathPattRule
)

var namesForRuleClass = []string{
        GeneralRule:  "GeneralRule",
        PatternRule:  "PatternRule",
        PathPattRule: "PathPattRule",
}

func (c RuleClass) String() string {
        var i = int(c)
        if 0 <= i && i < len(namesForRuleClass) {
                return namesForRuleClass[i]
        }
        return fmt.Sprintf("RuleClass(%d)", i)
}

type ruleContext struct {
        Context
        rule *Rule
}
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
                        if false {
                                cc = append([]*ruleContext{ c }, cc...)
                        } else {
                                cc = append(cc, c)
                        }
                }
                for _, c := range cc {
                        if t := c.rule.Strval(c.Context); s != "" {
                                s = fmt.Sprintf("%s{%s}", t, s)
                        } else {
                                s = t
                        }
                }
                return s
        } else if s, t := ec.rule.Strval(ec.Context), ec.Context.String(); t != "" {
                return fmt.Sprintf("%s{%s}", s, t)
        } else {
                return fmt.Sprintf("%s", s)
        }
}
// func (ec *ruleContext) stems() (stems []string) {
//         if sc, ok := ec.Context.(*stemmedContext); ok {
//                 stems = sc.stem.Stems // only if the inner is stemmed
//         }
//         return
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
type Executer interface { Execute(Context, ...Value) ([]Value, travestates) }
type Entry interface {
        Object
        Executer

        Class() RuleClass
        Target() Value // pattern or concrete target
        Programs() []*Program
        setPrograms([]*Program)

        hasRecipes() bool

        option(Context) (bool, []Value)
        execute(Context, ...Value) ([]Value, travestates)
}

type ResolveEntries struct {
        Entry
        all []Entry
}
func (p *ResolveEntries) String() string { return fmt.Sprintf("%s(%d)", p.Entry, len(p.all)) }
func (p *ResolveEntries) add(entry Entry) {
        if p.Entry == nil { p.Entry = entry }
        p.all = append(p.all, entry)
}
func (p *ResolveEntries) join(o *ResolveEntries) {
        if p.Entry == nil { p.Entry = o.Entry }
        p.all = append(p.all, o.all...)
}
func (p *ResolveEntries) Programs() (programs []*Program) {
        for _, entry := range p.all {
               programs = append(programs, entry.Programs()...)
        }
        return
}

// Rule represents a declared rule entry.
type Rule struct {
        class RuleClass
        target Value
        arged []Value // for restriction/filter
        programs []*Program
        position Position
}
func (_ *Rule) Kind() Kind { return KindObject|KindRule }
func (entry *Rule) Class() RuleClass { return entry.class }
func (entry *Rule) Target() Value { return entry.target }
func (entry *Rule) Programs() []*Program { return entry.programs }
func (entry *Rule) DeclScope() *Scope { return entry.OwnerProject().scope }
func (entry *Rule) OwnerProject() *Project { return entry.programs[0].project }
func (entry *Rule) setPrograms(programs []*Program) { entry.programs = programs }
func (entry *Rule) setPosition(position Position) { entry.position = position }
func (entry *Rule) Position() (pos Position) {
        if pos = entry.position; !pos.IsValid() {
                if pos = entry.target.Position(); !pos.IsValid() {
                        for _, prog := range entry.programs {
                                if pos = prog.position; pos.IsValid() { break }
                        }
                }
        }
        return
}
func (entry *Rule) Name(ctx Context) (name string) {
        if entry == nil {
                erro(ctx, "nil entry")
        } else if isNull(entry.target) {
                erro(at(ctx,entry.position), "entry target is nil")
        } else {
                name = entry.target.Strval(ctx)
        }
        return
}
func (entry *Rule) True(ctx Context) bool { return entry.target.True(ctx) }
func (entry *Rule) Float(_ Context) (f float64, _ error) { return 0, nil }
func (entry *Rule) Integer(_ Context) (i int64, _ error) { return 0, nil }
func (entry *Rule) Strval(ctx Context) string { return entry.target.Strval(ctx) }
func (entry *Rule) String() string {
        if entry.target == nil { return "<nil entry>" }
        return entry.target.String()
}
func (entry *Rule) updated(ctx Context) bool {
        var res = entry.target.updated(ctx)
        if res { ctx.dirtyMark(entry.target) }
        return res
}
func (entry *Rule) updatedDeps(ctx Context, v ...Value) []Value {
        var res = entry.target.updatedDeps(ctx, v...)
        return res
}
// Rule.Execute executes the rule program only if the target is outdated.
func (entry *Rule) Execute(ctx Context, a ...Value) (result []Value, traves travestates) {
        switch entry.class {
        case PatternRule, PathPattRule:
                erro(ctx, "executing pattern entry '%v'", entry.target).debug(1)
        default:
                result, traves = entry.execute(ctx, a...)
        }
        return
}
func (entry *Rule) execute(cc Context, a... Value) (result []Value, traves travestates) {
        if cc = (&ruleContext{ cc, entry }); len(a) > 0 { cc = &argumentedContext{ cc, a } }
ForPrograms:
        for _, program := range entry.programs {
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
func (entry *Rule) Get(_ Context, name string) (Value, error) {
        switch name {
        case "class": return MakeString(entry.position, entry.class.String()), nil
        case "name" : return entry.target, nil //return MakeString(entry.position, entry.Name()), nil
        //case "prerequisites": ...
        }
        return nil, fmt.Errorf("no such entry property (%s)", name)
}
func (entry *Rule) rescope(ctx Context, scope *Scope) { panic("Rule.rescope not supported") }
func (entry *Rule) recipes() (recipes []Value) {
        for _, prog := range entry.programs {
                for _, recipe := range prog.recipes {
                        recipes = append(recipes, recipe)
                }
        }
        return
}
func (entry *Rule) hasRecipes() (res bool) {
        for _, prog := range entry.programs {
                if res = len(prog.recipes) > 0; res { break }
        }
        return
}
func (entry *Rule) refs(ctx Context, v Value) bool {
        if entry.target.refs(ctx, v) { return true }

        return false

        for _, prog := range entry.programs {
                for _, depend := range prog.depends {
                        if depend.refs(ctx, v) { return true }
                }
                for _, recipe := range prog.recipes {
                        if recipe.refs(ctx, v) { return true }
                }
        }
        return false
}
func (entry *Rule) defs(ctx Context, s ...string) (res []*def) {
        res = entry.target.defs(ctx, s...)
        for _, prog := range entry.programs {
                for _, depend := range prog.depends {
                        res = append(res, depend.defs(ctx, s...)...)
                }
                for _, recipe := range prog.recipes {
                        res = append(res, recipe.defs(ctx, s...)...)
                }
        }
        return
}
func (entry *Rule) expandable(ctx Context, w facet) (res bool) {
        if res = entry.target.expandable(ctx, w); res { return }
        if false {
                for _, prog := range entry.programs {
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
func (entry *Rule) expand(ctx Context, w facet) (res Value) {
        if entry == nil {
                // happens from some &{xxx} exprs
                erro(ctx, "expand nil entry (w=%016b)", w).debug(1)
                return
        }

        var target Value
        if target = entry.target.expand(ctx, w); target != entry.target {
                // TODO: test if programs are needed to be disclosed??
                res = &Rule{
                        entry.class, target,
                        entry.arged,
                        entry.programs,
                        entry.position,
                }
        } else {
                res = entry
        }
        return
}
func (entry *Rule) delete(  ctx Context) (files []*File, err error) { return entry.target.delete(ctx) }
func (entry *Rule) stamp(   ctx Context) (files []*File, err error) { return entry.target.stamp(ctx) }
func (entry *Rule) traverse(ctx Context) {
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
        for i, prog := range entry.programs {
                var ctx = at(ctx, prog.position)
                var v, t = prog.execute(ctx)

                if true && sc != nil && sc.stem.target.Strval(ctx) == "Unwind-EHABI.o" {
                        info(ctx, "Rule.traverse: %v: %d: %v %v", target, i, t, v)
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
func (entry *Rule) stat(ctx Context) (si *statinfo) { return entry.target.stat(ctx) }
func (entry *Rule) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*Rule); ok {
                assert(ok, "value is not Rule")
                if /*entry.class == a.class &&*/ entry.target.cmp(ctx, a.target) == cmpEqual {
                        if entry.OwnerProject() == a.OwnerProject() {
                                res = cmpEqual
                        }
                }
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
                res = entry.cmp(ctx, l.Elems[0])
        }
        return
}
func (entry *Rule) patterned(ctx Context) bool { return entry.target.patterned(ctx) }
func (entry *Rule) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = entry.target.match(ctx, i)
    return
}
func (entry *Rule) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = entry.target.stencil(ctx, stems)
    return
}

func (entry *Rule) option(ctx Context) (res bool, infos []Value) {
        ForPrograms: for _, program := range entry.programs {
                if !program.configure { continue }
                for _, depend := range program.depends {
                        g, ok := depend.(*modifications)
                        if!ok { continue }
                        for _, m := range g.list {
                                if m.Elems[0].Strval(ctx) != "configure" { continue }
                                for _, arg := range m.Elems[1:] {
                                        a, ok := arg.(*argumented)
                                        if!ok { continue }
                                        f, ok := a.value.(*Flag)
                                        if!ok { continue }
                                        if f.name.Strval(ctx) != "option" { continue }
                                        for _, v := range a.args {
                                                if p, ok := v.(*Pair); ok {
                                                        if p.Key.Strval(ctx) != "info" { continue }
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

func (_ *Rule) hit(ctx Context, cache hitch, bits int) (res *filemapCache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *Rule) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *Rule) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
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
        *Rule
        target Value
        stems []string
}
func (_ *stemmed) Kind() Kind { return KindObject|KindStemmedRule }
func (p *stemmed) String() (s string) {
        for i, stem := range p.stems { if i > 0 { s += "," }; s += stem }
        return fmt.Sprintf("%s:%s", p.target, s) // "<%s:%s>"
}
func (p *stemmed) Target() Value { return p.target }
func (p *stemmed) expand(ctx Context, w facet) (res Value) {
        if v := p.Rule.expand(ctx, w); v != p.Rule {
                res = &stemmed{v.(*Rule), p.target, p.stems}
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
                res = p.Rule.cmp(ctx, a.Rule)
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        }
        return
}
func (p *stemmed) traverse(ctx Context) {
        var real = p.Rule // TODO: avoid copying the Rule, use p directly
        real.target = p.target
        real.traverse(&stemmedContext{ ctx, p })
}
