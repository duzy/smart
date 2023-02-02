//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
        "sync"
        "bytes"
        "time"
        "fmt"
)

// Object is a value defined in a scope.
//
// TODO: defines ObjInfo to classify objects.
// 
type Object interface {
        Value

        DeclScope() *Scope
        OwnerProject() *Project

        Name(ctx Context) string

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
func (_ *objbase) kind() kind { return valOther }
func (p *objbase) DeclScope() *Scope { return p.scope }
func (p *objbase) OwnerProject() *Project { return p.owner }
func (p *objbase) String() string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) Strval(ctx Context) string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) Name(ctx Context) string { panic("inquiring name of an unknown object") }
func (p *objbase) Get(_ Context, name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p *objbase) rescope(_ Context, scope *Scope) { panic("rescoping unknown object") }
func (p *objbase) exists() existence { return existenceMatterless }
// func (p *objbase) cmp(ctx Context, v Value) (res cmpres) {
//         if a, ok := v.(*objbase); ok {
//                 assert(ok, "value is not objbase")
//                 if p.owner == a.owner && p.scope == a.scope {
//                         res = cmpEqual
//                 }
//         } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
//                 res = p.cmp(ctx, l.Elems[0])
//         }
//         return
// }

type knownobject struct { // generally named objects
        objbase
        name string // single, or group name if containing '(*)' and corresponding members
        //members [][]string
}
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

type unresolvedobject struct { // named callable/executable objects
        objbase
        name Value // name could be closured
}
func (p *unresolvedobject) Name(ctx Context) (name string) {
        if isNil(p.name) {
                erro(ctx, "unresolved object name is nil").at(p.position)
        } else if ctx == nil {
                name = p.name.String()
        } else {
                name = p.name.Strval(ctx)
        }
        return
}
func (p *unresolvedobject) String() string { return p.name.String() }
func (p *unresolvedobject) Strval(_ Context) string {
        // The string value of a unresolved object is "", so that a
        // unresolved &(var) is stringed to ""
        return /*p.name.Strval()*/""
}
func (p *unresolvedobject) True(_ Context) bool { return false }
func (p *unresolvedobject) Call(ctx Context, a ...Value) (result Value) { result = p; return }
func (p *unresolvedobject) Execute(ctx Context, a ...Value) (result []Value, err error) { return []Value{p}, nil }
func (p *unresolvedobject) rescope(ctx Context, scope *Scope) {
        if p.scope != scope {
                var name = p.name.Strval(ctx)
                if p.scope != nil { delete(p.scope.elems, name) }
                if p.scope = scope; p.scope != nil { p.scope.elems[name] = p }
        }
}
func (p *unresolvedobject) expand(_ Context, _ facet) Value { return p }
func (p *unresolvedobject) cmp(ctx Context, v Value) (res cmpres) {
        if a, y := v.(*unresolvedobject); y {
                res = p.name.cmp(ctx, a.name)
        } else if l, y := v.(*List); y && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        } else if u, y := v.(unexpanded); y && u.Value != nil {
                res = p.cmp(ctx, u.Value)
        }
        return
}
func (p *unresolvedobject) traverse(ctx Context) (traves travestates) { return }

func unresolved(p *Project, v Value) *unresolvedobject {
        var pos = v.Position()
        if !pos.IsValid() { pos = p.position }
        return &unresolvedobject{objbase{valbase:valbase{pos}, scope:p.scope, owner:p}, v}
}

type ProjectName struct {
        knownobject
        project *Project
}

// Imported returns the project that was imported.
// It is distinct from Project(), which is the project
// containing the import statement.
func (p *ProjectName) NamedProject() *Project { return p.project }
func (p *ProjectName) Position() (pos Position) {
        if pos = p.position; !pos.IsValid() { pos = p.project.position }
        return
}
func (p *ProjectName) String() string { return p.name }
func (p *ProjectName) Strval(_ Context) string { return p.name }
func (p *ProjectName) True(_ Context) bool { return p.project != nil }
func (p *ProjectName) Get(ctx Context, name string) (value Value, err error) {
        if p.project != nil { value = p.project.resolveObject(ctx, name) }
        return
}

// Call a ProjectName returns the project name.
func (p *ProjectName) Call(ctx Context, a... Value) (value Value) {
        var pos = p.position
        if !pos.IsValid() { pos = ctx.Position() }
        if p.project == nil {
                erro(ctx, "nil project '%s'", p.name).at(pos).debug(1)
        } else {
                value = MakeString(pos, p.project.name)
        }
        return
}

func (p *ProjectName) traverse(ctx Context) (traves travestates) {
        if entry := p.project.DefaultEntry(); entry != nil {
                traves = entry.traverse(ctx)
        }
        return
}
func (p *ProjectName) stat(ctx Context) (si *statinfo) {
        if p.project != nil {
                if defent := p.project.DefaultEntry(); defent == nil {
                        // does nothing
                } else if defent.Class() != UseRuleEntry {
                        si = defent.stat(ctx)
                }
        }
        return
}
func (p *ProjectName) expand(_ Context, _ facet) Value { return p }
func (p *ProjectName) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*ProjectName); ok {
                assert(ok, "value is not ProjectName")
                if p.name == a.name && p.project == a.project {
                        res = cmpEqual
                }
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
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
func (n *ScopeName) NamedScope() *Scope { return n.scope }
func (n *ScopeName) String() string  { return fmt.Sprintf("{scope %s}", n.name) }
func (n *ScopeName) Strval(ctx Context) string { return fmt.Sprintf("scope %s", n.name) }
func (n *ScopeName) True(ctx Context) bool { return n.scope != nil }
func (n *ScopeName) Get(ctx Context, name string) (Value, error) {
        if sym := n.scope.Resolve(name); sym != nil {
                value, _ := sym.(Value)
                return value, nil
        }
        return nil, fmt.Errorf("Undefined `%s' in scope `%s'.", name, n.Name(ctx))
}
func (p *ScopeName) expand(_ Context, _ facet) Value { return p }
func (p *ScopeName) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*ScopeName); ok {
                assert(ok, "value is not ScopeName")
                if p.name == a.name && p.scope == a.scope {
                        res = cmpEqual
                }
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
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
        DefExecuted // executed result, TODO: remove DefExecuted state, add !:= for immediately executed defs

        // context bound defs
        DefArg     // ((arg))
        DefAuto    // automatic: $1, $2, $3, etc.

        // configuration defs
        DefConfDir
        DefConfRef // referred by config
        DefConfig

        DefDecl    // declaration names

        defany // referred any def
)

func (o Origin) String() (s string) {
        switch o {
        case defany:     s = "any"
        case DefVoid:    s = "Void"
        case DefDefault: s = "Default"
        case DefExpand1: s = "Expand1"
        case DefExpand2: s = "Expand2"
        case DefExecute: s = "Execute"
        case DefExecuted: s = "Executed"
        case DefArg:     s = "Arg"
        case DefAuto:    s = "Auto"
        case DefConfDir: s = "ConfDir"
        case DefConfRef: s = "ConfRef"
        case DefConfig:  s = "Config"
        case DefDecl:    s = "Decl"
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
// func (ac *autoContext) spawn(ctx Context) Context {
//         ac.mutex.Lock(); defer ac.mutex.Unlock()
//         return &autoContext{ Context: ac.Context.spawn(ctx), defs: ac.defs.clone() }
// }
func (ac *autoContext) aquireLock() (unlock func()) {
    ac.Lock() ; return func() { ac.Unlock() }
}
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
func (ac *autoContext) auto() *autoContext { return ac }
func (ac *autoContext) autoGet(name string) (res *def) {
        var ok bool
        ac.RLock()
        res, ok = ac.defs[name]
        ac.RUnlock()
        if ok && res != nil {
                // done!
        } else if ic := ac.inner(); ic == nil {
                warn(ac, "missing: %v in %v", name, ac).debug(32)
        } else if res = ic.autoGet(name); traverseArgumentedExpand || res != nil {
                // done!
        }
        if false && res == nil { res = ac.Scope().FindDef(name) }
        return
}

func autoGet(ctx Context, name string) (res Value) {
        if def := ctx.autoGet(name); def != nil {
                res = def.value
        }
        return
}

func (ac *autoContext) autoSet(name string, val Value) (out *def, res Value) {
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

func (ac *autoContext) autoArgs(params []*def, args []Value) (names []string, err error) {
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
        ForArgs: for _, a := range args {
                if p, ok := a.(*Pair); ok { for _, ca := range compactArgs {
                        if c, ok := ca.(*Pair); ok && eq(ac, p.Key, c.Key) {
                                var vals = merge(p.Value)
                                if l, ok := c.Value.(*List); ok {
                                        l.Elems = append(l.Elems, vals...)
                                } else {
                                        c.Value = MakeList(c.Position(),
                                                append(merge(c.Value), vals...)...)
                                }
                                continue ForArgs
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
                if a = Scalar(a); false && (isNil(a) || isNone(a)) {
                        erro(ac, "%T '%v' is invalid scalar", a, a).of(a).debug(1)
                        return
                } else if p, ok := a.(*Pair); ok {
                        var s string
                        if s = p.Key.Strval(ac); namedParam(s) {
                                name, a = s, p.Value
                        } else if false {
                                for _, param := range params { warn(ac, "%v: %v", s, param) }
                                warn(ac, "%s: %T %v", s, a, a).debug(1)
                        }
                }
                if name != "" {
                        // Got the name!
                } else if argnum < len(params) {
                        name = params[argnum].name
                } else {
                        name = id
                }

                if def, _ := ac.autoSet(name, a); def == nil {
                        erro(ac, "arg '%s' not set ($%s)", name, id).of(a).debug(1)
                        return
                } else if def, ok := ac.defs[name]; !ok || def == nil {
                        erro(ac, "arg '%s' not set ($%s)", name, id).of(a).debug(1)
                        return
                } else if id != "" && id != name {
                        ac.Lock()
                        ac.defs[id] = def // NOTE: set an alias or replace it
                        ac.Unlock()
                }
                names = append(names, name)
                argnum += 1
        }
        if false { for i := argnum+1; i < 9; i += 1 {
                var id = strconv.Itoa(i)
                ac.autoSet(id, nil)
        }}
        return
}

type defbit uint

const (
        defUnavail defbit = 1<<iota
)

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type def struct {
        knownobject
        // mutex sync.Mutex
        origin Origin
        value Value
        bits defbit
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
        // d.mutex.Lock()
        value = d.value
        // d.mutex.Unlock()
        if s = d.name + d.streq(); value != nil {
                s += elementString(nil, d, value, 0)
        } else {
                s += "<nil>"
        }
        return
}
func (d *def) Strval(ctx Context) (res string) {
        if d.origin == DefArg || d.origin == DefAuto {
                if v := autoGet(ctx, d.name); !isNil(v) {
                        res = v.Strval(ctx)
                }
        } else {
                var value Value
                // d.mutex.Lock()
                value = d.value
                // d.mutex.Unlock()
                if !isNil(value) { res = value.Strval(ctx) }
        }
        return
}
func (d *def) True(ctx Context) (res bool) {
        if d.origin == DefArg || d.origin == DefAuto {
                if v := autoGet(ctx, d.name); !isNil(v) {
                        res = v.True(ctx)
                }
        } else {
                var value Value
                // d.mutex.Lock()
                value = d.value
                // d.mutex.Unlock()
                if !isNil(value) { res = value.True(ctx) }
        }
        return
}
func (d *def) refs(ctx Context, v Value) (res bool) {
        if d.origin == DefArg || d.origin == DefAuto {
                if def := ctx.autoGet(d.name); def != nil {
                        if def == d { return true }
                        if def.value != nil { res = def.value.refs(ctx, v) }
                }
        } else if res = d == v; !res {
                var value Value
                // d.mutex.Lock()
                value = d.value
                // d.mutex.Unlock()
                if !(isNil(value) || isNone(value)) {
                        res = value.refs(ctx, v)
                }
        }
        return
}
func (d *def) defs(ctx Context, s ...string) (res []*def) {
        if true && (d.origin == DefArg || d.origin == DefAuto) && d.value == nil {
                if def := ctx.autoGet(d.name); def != nil && def != d && !isNil(def.value) {
                        res = def.value.defs(ctx, s...)
                }
                return
        } else if len(s) == 0 {
                res = append(res, d)
        } else {
                for _, a := range s {
                        if d.name == a {
                                res = append(res, d)
                                break
                        }
                }
        }

        var value Value
        // d.mutex.Lock()
        value = d.value
        // d.mutex.Unlock()
        if true && !isNil(value) {
                res = append(res, value.defs(ctx, s...)...)
        }
        return
}
func (d *def) expandible(ctx Context, w facet) (res bool) {
        if d.origin == DefArg || d.origin == DefAuto {
                // res = true // expand to DefAutoVal
                if v := autoGet(ctx,d.name); !isTrivial(v) {
                        res = v.expandible(ctx, w)
                }
        } else {
                var value Value
                // d.mutex.Lock()
                value = d.value
                // d.mutex.Unlock()
                if value != nil {
                        res = value.expandible(ctx, w)
                }
        }
        return
}
func (d *def) expand(ctx Context, w facet) (res Value) {
        if d == nil {
                errostack(ctx, 3, "expand nil def (w=%016b)", w).debug(16)
                return
        }

        res = d

        if origin, value0 := d._value(ctx); isNil(value0) {
                return // does nothing
        } else if isNone(value0) {
                if /* w&expandDef != 0 */true { res = value0 }
        } else if value1 := value0.expand(ctx, w); w&expandDef != 0 {
                res = value1.expand(ctx, w)
        } else if value1 != value0 {
                res = &def{ knownobject: d.knownobject, origin: origin, value: value1 }
        }
        return
}
func (d *def) _value(ctx Context) (origin Origin, res Value) {
        if origin = d.origin; origin == DefArg || origin == DefAuto {
                // a.mutex.Lock()
                res = autoGet(ctx, d.name)
                // a.mutex.Unlock()
        } else {
                // d.mutex.Lock()
                res = d.value
                // d.mutex.Unlock()
        }
        return
}
func (d *def) cmp(ctx Context, v Value) (res cmpres) {
        if a, y := v.(*def); y {
                if d == a || (isNil(d.value) && isNil(a.value)) {
                        res = cmpEqual
                        return
                }

                var val1, val2 Value
                if d.origin == DefArg || d.origin == DefAuto {
                        if a.origin == DefArg || a.origin == DefAuto {
                                if d.name == a.name { return cmpEqual }
                        }
                        val1 = autoGet(ctx, d.name)
                } else {
                        // d.mutex.Lock()
                        val1 = d.value
                        // d.mutex.Unlock()
                }
                if a.origin == DefArg || a.origin == DefAuto {
                        val2 = autoGet(ctx, a.name)
                } else {
                        // a.mutex.Lock()
                        val2 = a.value
                        // a.mutex.Unlock()
                }
                if isNil(val1) {
                        if isNil(val2) { res = cmpEqual }
                } else if !isNil(val2) {
                        res = val1.cmp(ctx, val2)
                }
        } else if l, y := v.(*List); y && len(l.Elems) == 1 {
                res = d.cmp(ctx, l.Elems[0])
        }
        return
}
func (d *def) elemstr(_ Context, o Object, k elemkind) (s string) {
        if o != nil {
                if p := d.OwnerProject(); p != o.OwnerProject() {
                        return fmt.Sprintf("$(%s->%s)", p.name, d.name)
                }
        }
        s = fmt.Sprintf(`$(%s)`, d.name)
        return
}
func (d *def) isEmpty(ctx Context) bool {
        var val Value
        if d.origin == DefArg || d.origin == DefAuto {
                val = autoGet(ctx, d.name)
        } else {
                // d.mutex.Lock()
                val = d.value
                // d.mutex.Unlock()
        }
        return isTrivial(val) //isNone(val) || isNil(val)
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
        if len(app) > 0 { vals = append(vals, app...) }
        for _, val := range vals {
                if def, ok := val.(*def); ok {
                        // Appending Def value is not recommended, but if it does, we make
                        // a warning here to give a chance for further optimization.
                        warn(ctx, "%v; (%v)", d, d.origin)
                        warnstack(ctx, 5, "%v: append a Def value: %v", d.name, def).debug(16)
                }
        }

        if d.origin == DefArg || d.origin == DefAuto {
                if origin != d.origin {
                        warnstack(ctx, 3, "%s: %v != %v", d.name, d.origin, origin).debug(6)
                }

                if value == nil && len(app) != 0 {
                        if a := ctx.autoGet(d.name); a != nil {
                                if !isTrivial(a.value) {
                                        vals = append(merge(a.value), vals...)
                                }
                                // a.mutex.Lock()
                                a.value = MakeListOrScalar(pos, vals)
                                // a.mutex.Unlock()
                                return
                        }
                }

                ctx.autoSet(d.name, MakeListOrScalar(pos, vals))
                return
        }

        switch w := plain&^expandPlain; origin {
        case DefExpand1: vals, _, _ = (w&^expandClosure).expand(ctx, vals...) //  :=
        case DefExpand2: vals, _, _ = (w| expandClosure).expand(ctx, vals...) // ::=
        default: for _, val := range vals {
                if val != nil && val.refs(ctx, d) {
                        if options.verbose { prompt(ctx, "set %s (%v): %v\n", origin, d.name, val) }
                        if options.debug { info(ctx, "from here").at(pos).debug(1) }
                        erro(ctx, "value refers to assigning Def '%s': %v (%T)",
                                d.name, val, val).at(pos).debug(1)
                        return
                }
        }}

        if value == nil {
                if len(app) > 0 {
                        // d.mutex.Lock()
                        if !isTrivial(d.value) {
                                vals = append(merge(d.value), vals...)
                        }
                        // d.mutex.Unlock()
                } else if origin != DefExecute {
                        // d.mutex.Lock()
                        d.origin, d.value = origin, MakeNone(d.position)
                        // d.mutex.Unlock()
                        return
                }
        }

        // d.mutex.Lock()
        d.origin, d.value = origin, MakeListOrScalar(pos, vals)
        // d.mutex.Unlock()
        return
}
func (d *def) append(ctx Context, va... Value) {
        if len(va) > 0 { d.set(ctx, d.origin, nil, va...) }
}

func isDigits(s string) bool {
    return strings.IndexFunc(s, func(c rune) bool {
        return !scanner.IsDigit(c) }) < 0
}

func (d *def) call1(ctx Context, w facet, a... Value) (res Value) {
        if d.bits&defUnavail == 0 && d.value != nil {
                if len(a)>0 {
                        ac := autoContext{ Context:ctx, defs:make(autoDefMap) }
                        ac.autoArgs(nil, a)
                        ctx = &ac
                }

                // d.mutex.Lock()
                var bits = d.bits
                d.bits |= defUnavail
                res = d.value.expand(ctx, w|expandDigits)
                d.bits = bits
                // d.mutex.Unlock()
        }
        return
}

func (d *def) call(ctx Context, w facet, a... Value) (res Value) {
        switch d.origin {
        case DefArg, DefAuto:
                if w&expandPlaceholders == 0 && d.name == "_" {
                        // nil ⇒ unexpanded
                } else if w&expandDigits == 0 && isDigits(d.name) {
                        // nil ⇒ unexpanded
                } else if w&expandAuto == 0 {
                        // nil ⇒ unexpanded
                } else if t := ctx.autoGet(d.name); t != nil {
                        res = t.call1(ctx, w, a...)
                } else if isTrivial(d.value) {
                        res = d.value
                } else {
                        res = d.call1(ctx, w, a...)
                }
        case DefDefault, DefVoid:
                res = d.call1(ctx, w, a...)
        case DefExecute:
                res = d.execute(ctx, a...)
        case DefExpand1:
                if isTrivial(d.value) {
                        res = d.value
                } else {
                        res = d.call1(ctx, w, a...)
                }
        default: // DefExpand2, DefExecuted, etc.
                res = d.value
        }

        if res == nil {
                // does nothing
        } else if list, ok := res.(*List); !ok {
                // does nothing
        } else if n := len(list.Elems); n == 0 {
                var pos = list.position
                if !pos.IsValid() { pos = ctx.Position() }
                if !pos.IsValid() { pos = d.position }
                res = MakeNone(pos)
        } else if n == 1 {
                res = list.Elems[0]
        }
        return
}

func (d *def) execute(ctx Context, a... Value) (res Value) {
        var (
                origin Origin
                value Value
                cmd string
        )
        // d.mutex.Lock()
        origin, value = d.origin, d.value
        // d.mutex.Unlock()

        if origin != DefExecute {
                erro(ctx, "%v: non-execute def: %v", origin, value).debug(1)
                return
        } else if isTrivial(value) {
                return // does nothing
        } else if cmd = value.Strval(ctx); cmd == "" {
                warn(ctx, "%v: empty command (value=%v)", origin, value).debug(1)
                return
        }

        // TODO: options for running command in the specified container
        var (
                stdout, stderr bytes.Buffer
                sh = exec.Command("sh", "-c", cmd)
        )
        sh.Stdout, sh.Stderr = &stdout, &stderr
        if err := sh.Run(); err != nil {
                erro(ctx, "%v: execute command failed: %v", origin, err)
                erro(ctx, "%v: execute command: %s", origin, cmd).debug(2)
                stdout.Reset()
                stderr.Reset()
                return
        }

        var pos = value.Position()
        if !pos.IsValid() { pos = ctx.Position() }
        res = MakeString(pos, strings.TrimSpace(stdout.String()))
        stdout.Reset()
        stderr.Reset()

        // d.mutex.Lock()
        d.origin, d.value = DefExecuted, res
        // d.mutex.Unlock()
        return
}

func (d *def) Call(ctx Context, a... Value) (res Value) {
        // NOTE: expandDelegate is still required for DefExpand1 as some autos
        //       ($_, $1, $2, etc) may still not expanded.
        return d.call(ctx, expandAuto|expandClosure|expandDelegate|expandPairVal, a...)
}

func (d *def) Get(ctx Context, name string) (res Value, err error) {
        switch name {
        case "name" : res = MakeString(d.position, d.name)
        case "value":
                if d.origin == DefArg || d.origin == DefAuto {
                        res = autoGet(ctx, d.name)
                } else {
                        // d.mutex.Lock()
                        res = d.value
                        // d.mutex.Unlock()
                }
        default:
                err = fmt.Errorf("no such property `%s' (Def)", name)
                erro(ctx, "%v", err).at(d.position).debug(1)
        }
        return
}
func (d *def) traverse(ctx Context) (traves travestates) {
        var value Value
        if d.origin == DefArg || d.origin == DefAuto {
                value = autoGet(ctx, d.name)
        } else {
                // d.mutex.Lock()
                value = d.value
                // d.mutex.Unlock()
        }
        if value != nil { traves = value.traverse(ctx) }
        return
}
func (d *def) stat(ctx Context) (si *statinfo) {
        var value Value
        if d.origin == DefArg || d.origin == DefAuto {
                value = autoGet(ctx, d.name)
        } else {
                // d.mutex.Lock()
                value = d.value
                // d.mutex.Unlock()
        }
        if value != nil { si = value.stat(ctx) }
        return
}

type undetermined struct {
        tok token.Token
        identifier Value
        value Value
}
func (_ *undetermined) kind() kind { return valOther }
func (p *undetermined) Position() Position { return p.identifier.Position() }
func (p *undetermined) String() (s string) {
        s = p.identifier.String()
        s += p.tok.String()
        s += p.value.String()
        return
}
func (p *undetermined) Strval(ctx Context) string { return p.value.Strval(ctx) }
func (p *undetermined) True(ctx Context) bool { return false }
func (p *undetermined) Float(ctx Context) (f float64, _ error) { return 0, nil }
func (p *undetermined) Integer(ctx Context) (i int64, _ error) { return 0, nil }
func (p *undetermined) updated(_ Context, _ ...bool) bool { return false }
func (p *undetermined) updatedDeps(_ Context, _ ...Value) []Value { return nil }
func (p *undetermined) refs(ctx Context, v Value) bool {
        return p.identifier.refs(ctx, v) || p.value.refs(ctx, v)
}
func (p *undetermined) defs(ctx Context, s ...string) (res []*def) {
        return append(p.identifier.defs(ctx, s...), p.value.defs(ctx, s...)...)
}
func (p *undetermined) expandible(ctx Context, w facet) bool {
        return p.identifier.expandible(ctx, w) || p.value.expandible(ctx, w)
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
func (p *undetermined) traverse(ctx Context) (traves travestates) { return }
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
func (p *undetermined) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return }
func (p *undetermined) stencil(ctx Context, stems []string) (val Value, rest []string) {
        return p, stems
}

// A Builtin represents a built-in function.
// Builtins don't have a valid type.
type Builtin struct {
        knownobject
        s BuiltinFunc
}
func (p *Builtin) String() string { return fmt.Sprintf("%s", p.name) }
func (p *Builtin) True(_ Context) bool { return p.s.f != nil }
func (p *Builtin) Call(ctx Context, a... Value) (res Value) {
        if p.s.f != nil { res = p.s.f(positional(ctx, p.position), plain, a...) }
        return
}
func (p *Builtin) expand(_ Context, _ facet) Value { return p }
func (p *Builtin) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*Builtin); ok {
                assert(ok, "value is not Builtin")
                if /*p.f == a.f &&*/ p.name == a.name {
                        res = cmpEqual
                }
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        }
        return
}

type RuleEntryClass int

const (
        GeneralRuleEntry RuleEntryClass = iota
        PatternRuleEntry
        PathPattRuleEntry
        UseRuleEntry
)

var namesForRuleEntryClass = []string{
        GeneralRuleEntry:  "GeneralRuleEntry",
        PatternRuleEntry:  "PatternRuleEntry",
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

type entryContext struct {
        Context
        ent *RuleEntry
}
func (ec *entryContext) entry() Entry { return ec.ent }
func (ec *entryContext) entryContext() *entryContext { return ec }
func (ec *entryContext) inner() Context { return ec.Context }
func (ec *entryContext) String() string {
        if fullContextStringer {
                return fmt.Sprintf("entry{%s,%s}", ec.ent, ec.Context)
        } else if true {
                var ( cc []*entryContext; s string )
                for c := ec; c != nil && len(cc) < 5; c = c.Context.entryContext() {
                        if false {
                                cc = append([]*entryContext{ c }, cc...)
                        } else {
                                cc = append(cc, c)
                        }
                }
                for _, c := range cc {
                        if t := c.ent.Strval(c.Context); s != "" {
                                s = fmt.Sprintf("%s{%s}", t, s)
                        } else {
                                s = t
                        }
                }
                return s
        } else if s, t := ec.ent.Strval(ec.Context), ec.Context.String(); t != "" {
                return fmt.Sprintf("%s{%s}", s, t)
        } else {
                return fmt.Sprintf("%s", s)
        }
}
func (ec *entryContext) Position() Position { return ec.ent.position }
// func (ec *entryContext) stems() (stems []string) {
//         if sc, ok := ec.Context.(*stemmedContext); ok {
//                 stems = sc.stem.Stems // only if the inner is stemmed
//         }
//         return
// }

type Entry interface {
        Object
        Executer
        Class() RuleEntryClass
        Target() Value // pattern or concrete target
        Programs() []*Program
        setPrograms([]*Program)

        hasRecipes() bool

        //isTrivial(Context) bool // draws prerequisites only, no recipes

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

// RuleEntry represents a declared rule entry.
type RuleEntry struct {
        class RuleEntryClass
        target Value
        argumented []Value // for restriction/filter
        programs []*Program
        position Position
}
func (_ *RuleEntry) kind() kind { return valOther }
func (entry *RuleEntry) Class() RuleEntryClass { return entry.class }
func (entry *RuleEntry) Target() Value { return entry.target }
func (entry *RuleEntry) Programs() []*Program { return entry.programs }
func (entry *RuleEntry) DeclScope() *Scope { return entry.OwnerProject().scope }
func (entry *RuleEntry) OwnerProject() *Project { return entry.programs[0].project }
func (entry *RuleEntry) setPrograms(programs []*Program) { entry.programs = programs }
func (entry *RuleEntry) setPosition(position Position) { entry.position = position }
func (entry *RuleEntry) Position() (pos Position) {
        if pos = entry.position; !pos.IsValid() {
                if pos = entry.target.Position(); !pos.IsValid() {
                        for _, prog := range entry.programs {
                                if pos = prog.position; pos.IsValid() { break }
                        }
                }
        }
        return
}
func (entry *RuleEntry) Name(ctx Context) (name string) {
        if entry == nil {
                erro(ctx, "nil entry")
        } else if isNil(entry.target) {
                erro(ctx, "entry target is nil").at(entry.position)
        } else {
                name = entry.target.Strval(ctx)
        }
        return
}
func (entry *RuleEntry) True(ctx Context) bool { return entry.target.True(ctx) }
func (entry *RuleEntry) Float(_ Context) (f float64, _ error) { return 0, nil }
func (entry *RuleEntry) Integer(_ Context) (i int64, _ error) { return 0, nil }
func (entry *RuleEntry) Strval(ctx Context) string { return entry.target.Strval(ctx) }
func (entry *RuleEntry) String() string {
        if entry.target == nil { return "<nil entry>" }
        return entry.target.String()
}
func (entry *RuleEntry) updated(ctx Context, v ...bool) bool {
        var res = entry.target.updated(ctx, v...)
        if res { ctx.dirtyMark(entry.target) }
        return res
}
func (entry *RuleEntry) updatedDeps(ctx Context, v ...Value) []Value {
        var res = entry.target.updatedDeps(ctx, v...)
        return res
}
// RuleEntry.Execute executes the rule program only if the target is outdated.
func (entry *RuleEntry) Execute(ctx Context, a ...Value) (result []Value, traves travestates) {
        switch entry.class {
        case PatternRuleEntry, PathPattRuleEntry:
                erro(ctx, "executing pattern entry '%v'", entry.target).debug(1)
                return
        }
        var t = traverseContext{
                Context: ctx,
                execRec: make(map[Value]int),
                start: time.Now(),
        }
        return entry.execute(&t, a...)
}
func (entry *RuleEntry) execute(cc Context, a... Value) (result []Value, traves travestates) {
        if cc = (&entryContext{ cc, entry }); len(a) > 0 { cc = &argumentedContext{ cc, a } }
ForPrograms:
        for _, program := range entry.programs {
                var pos = program.position
                if !pos.IsValid() { pos = entry.Position() }

                var res, t = program.execute(positional(cc, pos))
                result = append(result, merge(res)...)
                traves = append(traves, t...)
                if t.has(traveFail) { break ForPrograms }
                for _, s := range t.of(traveCase, traveDone) {
                        if s.prog == program { break ForPrograms }
                }
        }
        return
}
func (entry *RuleEntry) Get(_ Context, name string) (Value, error) {
        switch name {
        case "class": return MakeString(entry.position, entry.class.String()), nil
        case "name" : return entry.target, nil //return MakeString(entry.position, entry.Name()), nil
        //case "prerequisites": ...
        }
        return nil, fmt.Errorf("no such entry property (%s)", name)
}
func (entry *RuleEntry) rescope(ctx Context, scope *Scope) { panic("RuleEntry.rescope not supported") }
func (entry *RuleEntry) recipes() (recipes []Value) {
        for _, prog := range entry.programs {
                for _, recipe := range prog.recipes {
                        recipes = append(recipes, recipe)
                }
        }
        return
}
func (entry *RuleEntry) hasRecipes() (res bool) {
        for _, prog := range entry.programs {
                if res = len(prog.recipes) > 0; res { break }
        }
        return
}
func (entry *RuleEntry) refs(ctx Context, v Value) bool {
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
func (entry *RuleEntry) defs(ctx Context, s ...string) (res []*def) {
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
func (entry *RuleEntry) expandible(ctx Context, w facet) (res bool) {
        if res = entry.target.expandible(ctx, w); res { return }
        if false {
                for _, prog := range entry.programs {
                        for _, depend := range prog.depends {
                                if res = depend.expandible(ctx, w); res { return }
                        }
                        for _, recipe := range prog.recipes {
                                if res = recipe.expandible(ctx, w); res { return }
                        }
                }
        }
        return
}
func (entry *RuleEntry) expand(ctx Context, w facet) (res Value) {
        if entry == nil {
                // happens from some &{xxx} exprs
                erro(ctx, "expand nil entry (w=%016b)", w).debug(1)
                return
        }

        var target Value
        if target = entry.target.expand(ctx, w); target != entry.target {
                // TODO: test if programs are needed to be disclosed??
                res = &RuleEntry{
                        entry.class, target,
                        entry.argumented,
                        entry.programs,
                        entry.position,
                }
        } else {
                res = entry
        }
        return
}
func (entry *RuleEntry) delete(  ctx Context) (files []*File, err error) { return entry.target.delete(ctx) }
func (entry *RuleEntry) stamp(   ctx Context) (files []*File, err error) { return entry.target.stamp(ctx) }
func (entry *RuleEntry) traverse(ctx Context) (traves travestates) {
        if target := autoGet(ctx, "@"); target == nil {
                erro(ctx, "$@ is not defined").debug(1)
                return
        } else if ctx.entry() == entry {
                var proj = ctx.Project()
                if c := ctx.closure(); c != nil {
                        if t := autoGet(c, "@"); t != nil && eq(ctx, t, target) {
                                if false { warn(ctx, "%v: %v: %v\n", proj, entry, t) }
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
                ctx = &entryContext{ ctx, entry }
        }

        var (
                entryPos = entry.Position()
                result []Value
        )
ForPrograms:
        for _, prog := range entry.programs {
                var pos = prog.position
                if !pos.IsValid() { pos = entryPos }

                var res, t = prog.execute(positional(ctx, pos))
                result = append(result, merge(res)...)
                traves = append(traves, t...)
                if t.has(traveFail) { break ForPrograms }
                for _, s := range t.of(traveCase, traveDone) {
                        if s.prog == prog { break ForPrograms }
                }
        }
        return
}
// FIXME: entry.target maybe not the real target
func (entry *RuleEntry) stat(ctx Context) (si *statinfo) { return entry.target.stat(ctx) }
func (entry *RuleEntry) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*RuleEntry); ok {
                assert(ok, "value is not RuleEntry")
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
func (entry *RuleEntry) patterned(ctx Context) bool { return entry.target.patterned(ctx) }
func (entry *RuleEntry) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = entry.target.match(ctx, i)
    return
}
func (entry *RuleEntry) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = entry.target.stencil(ctx, stems)
    return
}

func (entry *RuleEntry) option(ctx Context) (res bool, infos []Value) {
        ForPrograms: for _, program := range entry.programs {
                if !program.configure { continue }
                for _, depend := range program.depends {
                        g, ok := depend.(*modifiergroup)
                        if!ok { continue }
                        for _, m := range g.modifiers {
                                if m.name.Strval(ctx) != "configure" { continue }
                                for _, arg := range m.args {
                                        a, ok := arg.(*Argumented)
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

type PatternEntry struct { RuleEntry }
func (p *PatternEntry) expand(ctx Context, w facet) (res Value) {
        if ent := p.RuleEntry.expand(ctx, w); ent != &p.RuleEntry {
                res = &PatternEntry{ *ent.(*RuleEntry) }
        } else {
                res = p
        }
        return
}
func (p *PatternEntry) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*PatternEntry); ok {
                assert(ok, "value is not PatternEntry")
                // FIXME: p.Pattern.cmp(p.Pattern)
                if p.RuleEntry.cmp(ctx, &a.RuleEntry) == cmpEqual {
                        res = cmpEqual
                }
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        }
        return
}

type stemmedContext struct {
        Context
        stem *stemmed
}
func (sc *stemmedContext) inner() Context { return sc.Context }
func (sc *stemmedContext) String() string {
        if fullContextStringer {
                return fmt.Sprintf("stemmed{%s}", sc.Context)
        } else {
                return sc.Context.String()
        }
}
func (sc *stemmedContext) stemmedContext() *stemmedContext { return sc }
func (sc *stemmedContext) stemmed() *stemmed { return sc.stem }
func (sc *stemmedContext) stems() []string { return sc.stem.Stems }

type stemmed struct {
        *PatternEntry
        target Value
        Stems []string
}
func (p *stemmed) String() (s string) {
        for i, stem := range p.Stems { if i > 0 { s += "," }; s += stem }
        return fmt.Sprintf("%s:%s", p.target, s) // "<%s:%s>"
}
func (p *stemmed) Target() Value { return p.target }
func (p *stemmed) expand(ctx Context, w facet) (res Value) {
        if v := p.PatternEntry.expand(ctx, w); v != p.PatternEntry {
                res = &stemmed{v.(*PatternEntry), p.target, p.Stems}
        } else {
                res = p
        }
        return
}
func (p *stemmed) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*stemmed); ok {
                assert(ok, "value is not stemmed")
                if len(p.Stems) != len(p.Stems) { return }
                for i, stem := range p.Stems {
                        if stem != a.Stems[i] { return }
                }
                res = p.PatternEntry.cmp(ctx, a.PatternEntry)
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
                res = p.cmp(ctx, l.Elems[0])
        }
        return
}
func (p *stemmed) traverse(ctx Context) (traves travestates) {
        var real = p.RuleEntry // TODO: avoid copying the RuleEntry, use p directly
        real.target = p.target
        return real.traverse(&stemmedContext{ ctx, p })
}
