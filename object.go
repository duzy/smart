//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "reflect"
    "os/exec"
    "strings"
    "strconv"
    "sync"
    "bytes"
    "unsafe"
    "time"
    "fmt"
)

type Object interface {
    Value

    owner() *project
    declScope() *Scope
}

type objbase struct {
    valbase
    scope_ *Scope
    owner_ *project
}
func (_ *objbase) kind() Kind { return KindObject }
func (p *objbase) owner() *project { return p.owner_ }
func (p *objbase) declScope() *Scope { return p.scope_ }
func (p *objbase) String() string { return fmt.Sprintf("{obj %p}", p) }
func (p *objbase) string(ctx Context) string { return fmt.Sprintf("{obj %p}", p) }
func (p *objbase) ident(ctx Context) string { panic("inquiring name of an unknown object") }
func (p *objbase) exists() existence { return existenceMatterless }
func (p *objbase) setscope(scope *Scope) { p.scope_ = scope }

type knownobject struct { // generally named objects
    objbase
    name string // single, or group name if containing '(*)' and corresponding members
    //members [][]string
}
func (p *knownobject) kind() Kind { return p.objbase.kind()|KindKnownObject }
func (p *knownobject) String() string { return fmt.Sprintf("{object %s}", p.name) }
func (p *knownobject) string(_ Context) string { return fmt.Sprintf("{object %s}", p.name) }
func (p *knownobject) true(_ Context) bool { return p.name != "" }
func (p *knownobject) ident(_ Context) string { return p.name }
func (p *knownobject) expand(_ Context) Value { return p }
func (p *knownobject) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*knownobject); y {
        if p.owner_ == a.owner_ && p.scope_ == a.scope_ && p.name == a.name {
            res = cmpEqual
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
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

type origin int

const (
    defVoid origin = iota
    defConfig   // configure
    defConfDir  // configuration defs
    defConfRef  // referred by config
    defCodeBlockAuto
    defDecl     // declaration names
    defExpand0  //   =  normal value
    defExpand1  //  :=  expand delegates (simple expand)
    defExpand2  // ::=  expand all (delegates, closures, paths)
    defExpand3  // ;:=  TODO: expand as plain
    defExecute  //  !=  value to be executed
    _defAny // referred any def
)

func (o origin) String() (s string) {
    switch o {
    case defVoid:    s = "void"
    case defConfDir: s = "confdir"
    case defConfRef: s = "confref"
    case defConfig:  s = "config"
    case defDecl:    s = "decl"
    case defExpand0: s = "expand_0"
    case defExpand1: s = "expand_1"
    case defExpand2: s = "expand_2"
    case defExpand3: s = "expand_3"
    case defExecute: s = "execute"
    case _defAny:    s = "any"
    default: s = fmt.Sprintf("origin<%d>", o)
    }
    return
}

type autodefs map[string]*def
func (m autodefs) len() int { return len(m) }
func (m autodefs) String() (s string) {
    for _, d := range m {
        if s == "" { s = "{" } else { s += "," }
        s += d.String()
    }
    if s != "" { s += "}" }
    return
}
func (m autodefs) clone() (res autodefs) {
    res = make(autodefs)
    for s, d := range m {
        var t = new(def)
        // d.Lock()
        t.knownobject, t.value = d.knownobject, d.value
        // d.Unlock()
        res[s] = t
    }
    return
}

func _automatic(c Context) *automatic { return cast[*automatic](c) }

type automatic struct {
    Context
    sync.RWMutex
    defs autodefs
    suppress func(string) bool
}
func (ac *automatic) cast(t reflect.Type) Context { return implcast(ac, t) }
func (ac *automatic) do(prop property, a ...interface{}) interface{} {
    return bitdo(ac.Context, a, prop, propExAuto)
}
func (ac *automatic) String() string {
    if fullContextStringer {
        return fmt.Sprintf("auto{%s}", ac.Context)
    } else {
        return ac.Context.String()
    }
}
func (ac *automatic) amend(ctx Context, name string, val Value) (out *def, res Value) {
    if d := ac.search(ctx, name); d == nil {
        return ac.set(ctx, name, val)
    } else if res = d.value; d.value != val {
        out, d.value = d, val
    }
    return
}
func (ac *automatic) def(ctx Context, name string) (res *def, y bool) {
    ac.Lock() ; defer ac.Unlock()
    res, y = ac.defs[name]
    return
}
func (ac *automatic) search(ctx Context, name string) (res *def) {
    if res, _ = ac.def(ctx, name) ; res != nil { return }
    if ac.suppress != nil && ac.suppress(name) { return }
    var t = _automatic(ac.Context)
    if t == ac {
        erro(ctx, "%v: loop auto context", name).debug(32)
        return
    } else if t != nil {
        return t.search(ctx, name)
    }
    return
}
func (ac *automatic) set(ctx Context, name string, val Value) (out *def, old Value) {
    if name == "-" { if d, y := val.(*def); y && d.origin != defConfig {
        notestack(ctx, 3, "set $- to def (%v): %v", d.origin, d).debug(16)
    }}

    out, _ = ac.def(ctx, name)

    if out != nil {
        old = out.value
    } else {
        s := ac.Scope()
        out = &def{knownobject:knownobject{objbase{scope_:s, owner_:s.project}, name}}
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
func (ac *automatic) args(ctx Context, vals []Value) {
    type _at struct {
        id, name string
        value Value
    }

    var argnum int // setup named/number parameters ($1, $2, etc.)
    var args = make(map[string]*_at, len(vals)) // compact args: combine duplicated pairs
    var params = _parameters(ctx)

    for _, val := range vals {
        var a = &_at{ id: strconv.Itoa(argnum+1) }
        if p, y := val.(*pair); y {
            if a.name = p.key.string(ctx); a.name == "" {
                erro(ctx, "empty name: %v", p.key).debug(5)
                return
            } else if params == nil {
                // noop
            } else if _, y = params[a.name]; !y {
                erro(ctx, "unknown arg: %v %v", us(p.key), us(p.val))
                erro(ctx, "%v", us(ctx)).debug(5)
                return
            }

            if a, y := args[a.name]; y {
                if l, y := a.value.(*list); y {
                    l.elems = append(l.elems, merge(p.val)...)
                } else {
                    a.value = makeList(a.value)
                }
                continue
            }

            a.value = p.val
        } else {
            a.name, a.value = _paramName(ctx, argnum), scalarize(val)
            if a.name == "" { a.name = a.id }
        }

        if a.id != a.name { args[a.id] = a }
        args[a.name] = a
        argnum += 1

        if d, _ := ac.set(ctx, a.name, a.value); d == nil {
            erro(at(ac,a.value), "arg '%s' not set", a.name).debug(1)
            return
        } else if d, y := ac.defs[a.name]; !y || d == nil {
            erro(at(ac,a.value), "arg '%s' not set", a.name).debug(1)
            return
        } else if a.id != "" && a.id != a.name {
            const LOCK = true
            if LOCK { ac.Lock() }
            ac.defs[a.id] = d // NOTE: set an alias or replace it
            if LOCK { ac.Unlock() }
        }
    }
    return
}

func autoDef(ctx Context, name string) (d *def) {
    if a := _automatic(ctx); a != nil { d = a.search(ctx, name) }
    return
}

func autoVal(ctx Context, name string) (res Value) {
    if d := autoDef(ctx, name); d != nil { res = d.value }
    return
}

func autoGet(ctx Context, name string) (d *def, res Value) {
    if d = autoDef(ctx, name); d != nil { res = d.value }
    return
}

func autoSet(ctx Context, name string, val Value) (out *def, res Value) {
    if ac := _automatic(ctx); ac != nil { out, res = ac.set(ctx, name, val) }
    return
}

func isInnerAuto(ctx Context, target Value) (res bool) {
    if ac, n := _automatic(ctx), 0; ac != nil {
        for ac = _automatic(inner(ac)); ac != nil; ac = _automatic(inner(ac)) {
            if n > 1 { return true }
            if t := autoVal(ac, "@"); t != nil && eq(ctx, t, target) { n += 1 }
        }
    }
    return
}

type auto struct { knownobject }
func (a *auto) kind() Kind { return a.knownobject.kind()|KindAuto }
func (a *auto) String() (s string) { return a.name }
func (a *auto) string(ctx Context) (res string) {
    if d := a.def(ctx); d != nil && d.value != nil { res = d.value.string(ctx) }
    return
}
func (a *auto) get(ctx Context, name string) (res Value) {
    if name == "value" { res = autoVal(ctx, a.name) }
    return
}
func (a *auto) refs(ctx Context, v Value) (res bool) {
    if o, y := v.(*auto); y && (o == a || o.name == a.name) { return true }
    if d := a.def(ctx); d != nil && d.value != nil { res = d.value.refs(ctx, v) }
    return
}
func (a *auto) defs(ctx Context, s ...string) (res []*def) {
    if d := a.def(ctx); d != nil { res = append(res, d) }
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
    if d != nil {
        d.position = a.position
    } else if true {
        warnstack(at(ctx,a), 3, "set auto failed: %v: %v %v", a.name, value, app).debug(16)
    }
}
func (a *auto) isDigit() bool { return _isDigits(a.name) }
func (a *auto) isPlaceholder() bool { return a.name == "_" }
func (a *auto) expandable(ctx Context) (res bool) {
    if _exAuto(ctx) {
        var d = autoDef(ctx, a.name)
        return d != nil && d.value != nil
    }
    return
}
func (a *auto) expand(ctx Context) (res Value) {
    if _exAuto(ctx) {
        var d = autoDef(ctx, a.name)
        if d != nil && d.value != nil { return d }
    }
    return a
}
func (a *auto) invoke(ctx Context, o, v []Value) (res Value) {
    if d := autoDef(ctx, a.name); d != nil { res = d.invoke(ctx, o, v) }
    return
}
func (a *auto) cmp(ctx Context, v Value) (res cmpres) {
    if o, y := v.(*auto); y && (a == o || a.name == o.name) {
        res = cmpEqual
    } else if val := autoVal(ctx, a.name); val != nil {
        res = val.cmp(ctx, v)
    }
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
    knownobject // sync.Mutex
    origin origin
    value Value
}
func (d *def) kind() Kind { return d.knownobject.kind()|KindDef }
func (d *def) Position() (pos Position) {
    if  pos = d.position; !pos._valid() && d.value != nil {
        pos = d.value.Position()
    }
    return
}
func (d *def) streq() (s string) {
    switch d.origin {
    case defExpand0: s =   "="
    case defExpand1: s =  ":="
    case defExpand2: s = "::="
    case defExpand3: s = ";:="
    case defExecute: s =  "!="
    default:         s =   "⇒"
    }
    return
}
func (d *def) String() (s string) {
    var value Value
    {
        // d.Lock()
        value = d.value
        // d.Unlock()
    }

    if s = d.name + d.streq(); value != nil {
        s += srclit(d, value)
    } else {
        s += "<nil>"
    }
    return
}
func (d *def) string(ctx Context) (res string) {
    var val Value
    {
        // d.Lock()
        val = d.value
        // d.Unlock()
    }

    if val != nil { res = val.string(ctx) }
    return
}
func (d *def) true(ctx Context) (res bool) {
    var val Value
    {
        // d.Lock()
        val = d.value
        // d.Unlock()
    }

    if val != nil { res = val.true(ctx) }
    return
}
func (d *def) refs(ctx Context, v Value) (res bool) {
    if o, y := v.(*def); y { if d == o { return true } }

    var val Value
    {
        // d.Lock()
        val = d.value
        // d.Unlock()
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
        // d.Lock()
        val = d.value
        // d.Unlock()
    }

    if val != nil { res = append(res, val.defs(ctx, s...)...) }
    return
}
func (d *def) expandable(ctx Context) (res bool) {
    if _, y := ctx.(*evocation); y {
        var v Value
        {
            // d.Lock()
            v = d.value
            // d.Unlock()
        }
        if v != nil { return v.expandable(ctx) }
    }
    return
}
func (d *def) expand(ctx Context) Value {
    if x, y := ctx.(*evocation); y {
        return d.evoke(x)
    } else {
        return d
    }
}
func (d *def) evoke(ctx *evocation) (res Value) {
    // d.Lock()
    var o, v = d.origin, d.value
    // d.Unlock()

    ctx.a = expand(ctx, ctx.a...) // to save the changed args

    if !_exDefValue(ctx) && expandable(ctx, ctx.a...) {
        return d
    } else if v == nil {
        return
    }

    var x = automatic{ Context:ctx, defs:make(autodefs),
        suppress:func(s string) bool { return s == "_" || _isDigits(s) } }

    x.args(ctx, ctx.a)

    res = v.expand(&x)

    if res != nil && o == defExecute { res = d.xexec(ctx, res) }
    if res != nil { return scalarize(res) }
    return
}
func (d *def) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*def); y {
        if d == a || (d.value == nil && a.value == nil ) {
            res = cmpEqual
            return
        }

        var val1, val2 Value
        {
            // d.Lock()
            val1 = d.value
            // d.Unlock()
        }
        {
            // a.Lock()
            val2 = a.value
            // a.Unlock()
        }

        if val1 == nil {
            if val2 == nil { res = cmpEqual }
        } else if val2 != nil {
            res = val1.cmp(ctx, val2)
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = d.cmp(ctx, l.elems[0])
    }
    return
}
func (d *def) srclit(_ Context, o Object) (s string) {
    if o != nil {
        if p := d.owner(); p != o.owner() {
            return fmt.Sprintf("$(%s→%s)", p.name, d.name)
        }
    }
    s = fmt.Sprintf(`$(%s)`, d.name)
    return
}
func (d *def) val(ctx Context, value Value, vals ...Value) { d.set(ctx, d.origin, value, vals...) }
func (d *def) set(ctx Context, origin origin, value Value, app ...Value) {
    defer trace(ctx)

    if checkpoints { defer func() {
        d.set_check(ctx, origin, value, app...)
    }()}

    if value == d.value && len(app) == 0 {
        if d.origin != origin { d.origin = origin }
        return
    }

    var vals []Value
    if !isTrivial(value) { vals = append(vals, merge(value)...) }
    if len(app) > 0      { vals = append(vals, merge(app...)...) }
    if len(vals) > 0 && origin != defExpand0 {
        vals = expand(original{ctx,origin}, vals...)
    }

    if value == nil && len(app) > 0 {
        // d.Lock()
        var v = d.value
        // d.Unlock()
        if !isTrivial(v) { vals = append(merge(v), vals...) }
    }

    if n := len(vals); 1 == n {
        value = vals[0]
    } else if 1 < n {
        value = makeList(vals...)
    } else if origin == defExecute {
        value = nil
    } else {
        value = makeNull(d.position)
    }

    // d.Lock()
    d.origin, d.value = origin, value
    // d.Unlock()
    return
}
func (d *def) set_check(ctx Context, origin origin, value Value, app ...Value) {
    if _exDef(ctx) && (d.value == nil || isTrivial(d.value)) && (value != nil || len(app) > 0) {
        erro(ctx, "%v ; %v %v", d, value, app).debug(32)
    }
    if !d.position._valid() && d.name != ".goals" {
        erro(ctx, "%v ; %v %v", d, value, app).debug(32)
    }
    if (value != nil || app != nil) && d.value == nil {
        erro(ctx, "%v ; %v %v", d, value, app).debug(32)
    }
}
func (d *def) append(ctx Context, a ...Value) { if len(a) > 0 { d.set(ctx, d.origin, nil, a...) } }
func (d *def) invoke(ctx Context, o, a []Value) (res Value) { return invoke(ctx, d, o, a) }
func (d *def) xexec(ctx Context, value Value, a ...Value) (res Value) {
    if isTrivial(value) { return }

    var cmd string
    if cmd = value.string(ctx); cmd == "" {
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
    res = makeStrlit(pos, strings.TrimSpace(stdout.String()))
    stdout.Reset()
    stderr.Reset()
    return
}
func (d *def) get(ctx Context, name string) (res Value) {
    switch name {
    case "name" : res = makeStrlit(d.position, d.name)
    case "value":
        // d.Lock()
        res = d.value
        // d.Unlock()
    default:
        erro(at(ctx,d.position), "def: no such property `%s'", name).debug(1)
    }
    return
}
func (d *def) traverse(ctx Context) {
    var value Value
    {
        // d.Lock()
        value = d.value
        // d.Unlock()
    }
    if value != nil { value.traverse(ctx) }
}
func (d *def) stat(ctx Context) (si *statinfo) {
    var value Value
    {
        // d.Lock()
        value = d.value
        // d.Unlock()
    }
    if value != nil { si = value.stat(ctx) }
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
    tok token
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
func (p *undetermined) ident(ctx Context) string { return p.identifier.string(ctx) }
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
func (p *undetermined) expandable(ctx Context) bool {
    return p.identifier.expandable(ctx) || p.value.expandable(ctx)
}
func (p *undetermined) expand(ctx Context) (res Value) {
    var (
        i = p.identifier.expand(ctx)
        v = p.value.expand(ctx)
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
    if a, y := v.(*undetermined); y {
        assert(y, "value is not undetermined")
        if p.identifier.cmp(ctx, a.identifier) == cmpEqual {
            if p.value.cmp(ctx, a.value) == cmpEqual {
                res = cmpEqual
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *undetermined) patterned(ctx Context) bool { return false }
func (p *undetermined) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return }
func (p *undetermined) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (_ *undetermined) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *undetermined) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

const max_expand = 32

func builtinForceField(ctx Context, bv reflect.Value, bi interface{}, force bool) bool {
    if f := bv.Elem().FieldByName("force"); f.IsValid() && f.Kind() == reflect.Bool {
        if force {
            if f.CanSet() {
                f.SetBool(true)
            } else {
                *(*bool)(unsafe.Pointer(f.UnsafeAddr())) = true
            }
        } else {
            force = f.Bool()
        }
    }
    return force
}

type skip struct {}

// A builtin represents a built-in function. builtins don't have a valid type.
type builtin struct { knownobject; t reflect.Type }
func (p *builtin) kind() Kind { return p.knownobject.kind()|KindBuiltin }
func (p *builtin) String() string { return p.name }
func (p *builtin) true(_ Context) bool { return p.t != nil }
func (p *builtin) isCommand() bool { return reflect.PointerTo(p.t).Implements(builtin_c_t) }
func (p *builtin) invoke(ctx Context, o, a []Value) Value { return invoke(ctx, p, o, a) }
func (p *builtin) refs(ctx Context, v Value) (res bool) {
    if o, y := v.(*builtin); y { res = o == p /* || p.name == o.name */ }
    return
}
func (p *builtin) benchmark_expand(ctx Context, t0 time.Time, v reflect.Value) {
    if d := time.Now().Sub(t0); d > 1*time.Second {
        noted(ctx, "%v: slow: %v", p, d).debug(3)
    } else if f := v.Elem().FieldByName("timing"); !f.IsValid() {
        if false { noted(ctx, "%v: %v", p, d).debug(1) }
    } else if f.Type().Kind() == reflect.Bool && f.Bool() {
        noted(ctx, "%v: %v", p, d).debug(1)
    }
}
func (p *builtin) expand(ctx Context) (res Value) {
    if evo, y := ctx.(*evocation); y { return p.evoke(evo) }
    return p
}
func (p *builtin) evoke(ctx *evocation) (res Value) {
	defer trace(ctx)

    _v := reflect.New(p.t)
    _i := _v.Interface()

    if f := _universe(ctx).benchmark_builtin_expand; f != nil { defer f(p, ctx, time.Now(), _v) }

    if f := _v.Elem().FieldByName("builtin_"); !f.IsValid() {
        erro(ctx, "no such field: %s.builtin_", _v.Elem().Type()).debug(16)
        return
    } else if f.CanAddr() {
        b := (*builtin_)(unsafe.Pointer(f.Addr().Pointer()))
        b.evocation = ctx
    } else if f := _v.Elem().FieldByName("evocation"); !f.IsValid() {
        erro(ctx, "no such field: %s.evocation", _v.Elem().Type()).debug(16)
        return // f.Type().String() == "*smart.evocation"
    } else if f.CanSet() {
        // FIXME: can't set value for struct fields of type `*evocation`
        f.Set(reflect.ValueOf(ctx))
    } else if f.CanAddr() && f.Addr().CanSet() {
        // FIXME: still can't set pointer for values of type `**evocation`
        f.Addr().SetPointer(unsafe.Pointer(ctx))
    } else {
        unreachable("cannot set builtin_.evocation")
    }

    if ctx.o != nil { if o := _opts(ctx, _v, ctx.o); o != nil {
        errostack(ctx, 3, "%v: unsupported opts: %v", p, o).debug(16)
        return
    }}

    var force = /* _exFinal(ctx) || */ builtinForceField(ctx, _v, _i, false)

    if x, y := _i.(builtin_a); y {
        if skip := x.a(); skip && !force { return p }
    } else {
        if ctx.a = expand(ctx, ctx.a...); !force {
            if expandable(final{ctx}, ctx.a...) { return p }
        }
    }

    switch x := _i.(type) {
    case builtin_c:
        if t := x.c(); t != nil {
            erro(ctx, "discarded command result: %v", t).debug(10)
        }
        return
    case builtin_x:
        if t := x.x(); t == nil {
            return
        } else if _, y := t.(skip); y {
            return p
        } else {
            return ease(ctx, t)
        }
    default:
        // p.t.Name()    → builtin_auto
        // p.t.PkgPath() → extbit.io/smart
        erro(ctx, "no method: %v (%s)", p.t.Name(), us(_v)).debug(16)
        return
    }
}
func (p *builtin) cmp(ctx Context, v Value) (res cmpres) {
    if b, y := v.(*builtin); y {
        if p.t == b.t /* || p.name == a.name */ { res = cmpEqual }
        if checkpoints {
            if res != cmpEqual {
                if p.t == b.t { erro(ctx, "%v", us(v)).debug(1) }
                if p.name == b.name { erro(ctx, "%v", us(v)).debug(1) }
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
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
    pathPatRule
)

var namesForRuleClass = []string{
    GeneralRule:  "GeneralRule",
    PatternRule:  "PatternRule",
    pathPatRule: "pathPatRule",
}

func (c ruleClass) String() string {
    var i = int(c)
    if 0 <= i && i < len(namesForRuleClass) {
        return namesForRuleClass[i]
    }
    return fmt.Sprintf("ruleClass(%d)", i)
}

func _ruleContext(ctx Context) *ruleContext { return cast[*ruleContext](ctx) }
func _entry(ctx Context) (res entry) {
    if p := _ruleContext(ctx); p != nil { res = p.rule }
    return
}

type ruleContext struct { Context ; rule *rule }
func (ec *ruleContext) cast(t reflect.Type) Context { return implcast(ec,t) }
func (ec *ruleContext) Position() Position { return ec.rule.position }
func (ec *ruleContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("entry{%s,%s}", ec.rule, ec.Context)
    } else if true {
        var ( cc []*ruleContext; s string )
        for c := ec; c != nil && len(cc) < 5; c = cast[*ruleContext](c.Context) {
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

type invoker interface { invoke(Context, []Value, []Value) Value }
type executer interface { execute(Context, ...Value) ([]Value, travestates) }
type entry interface {
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

type entryArray []entry
func (p entryArray) Position() Position { return p[0].Position() }
func (p entryArray) String() string { return p[0].String() }
func (p entryArray) cache(ctx Context, cache *valcache, bits int) *valcache { return p[0].cache(ctx, cache, bits) }
func (p entryArray) cmp(ctx Context, v Value) cmpres { return p[0].cmp(ctx, v) }
func (p entryArray) collect(ctx Context, cache *valcache, bits int) []*valcache { return p[0].collect(ctx, cache, bits) }
func (p entryArray) declScope() *Scope { return p[0].declScope() }
func (p entryArray) delete(ctx Context) ([]*File, error) { return p[0].delete(ctx) }
func (p entryArray) execute(ctx Context, a ...Value) ([]Value, travestates) { return p[0].execute(ctx, a...) }
func (p entryArray) expand(ctx Context) Value { return p[0].expand(ctx) }
func (p entryArray) expandable(ctx Context) bool { return p[0].expandable(ctx) }
func (p entryArray) float(ctx Context) (float64, error) { return p[0].float(ctx) }
func (p entryArray) hasRecipes() bool { return p[0].hasRecipes() }
func (p entryArray) ident(ctx Context) string { return p[0].ident(ctx) }
func (p entryArray) int(ctx Context) (int64, error) { return p[0].int(ctx) }
func (p entryArray) kind() Kind { return p[0].kind() }
func (p entryArray) match(ctx Context, i interface{}) (bool, interface{}, []string) { return p[0].match(ctx, i) }
func (p entryArray) option(ctx Context) (bool, []Value) { return p[0].option(ctx) }
func (p entryArray) owner() *project { return p[0].owner() }
func (p entryArray) patterned(ctx Context) bool { return p[0].patterned(ctx) }
func (p entryArray) stamp(ctx Context) ([]*File, error) { return p[0].stamp(ctx) }
func (p entryArray) stat(ctx Context) *statinfo { return p[0].stat(ctx) }
func (p entryArray) stencil(ctx Context, s []string) (Value, []string) { return p[0].stencil(ctx, s) }
func (p entryArray) string(ctx Context) string { return p[0].string(ctx) }
func (p entryArray) traverse(ctx Context) { p[0].traverse(ctx) }
func (p entryArray) true(ctx Context) bool { return p[0].true(ctx) }
func (p entryArray) updated(ctx Context) bool { return p[0].updated(ctx) }
func (p entryArray) updatedDeps(ctx Context, v ...Value) []Value { return p[0].updatedDeps(ctx, v...) }
func (p entryArray) setPrograms(a []*program) { p[0].setPrograms(a) }
func (p entryArray) programs() (res []*program) {
    for _, e := range p { res = append(res, e.programs()...) }
    return
}
func (p entryArray) refs(ctx Context, v Value) (res bool) {
    for _, e := range p { if e.refs(ctx, v) { return true } }
    return
}
func (p entryArray) defs(ctx Context, s ...string) (res []*def) {
    for _, e := range p { res = append(res, e.defs(ctx, s...)...) }
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
func (p *rule) Target() Value { return p.target }
func (p *rule) Class() ruleClass { return p.class }
func (p *rule) programs() []*program { return p.program_ }
func (p *rule) declScope() *Scope { return p.owner().scope }
func (p *rule) owner() *project { return p.program_[0].project }
func (p *rule) setPrograms(programs []*program) { p.program_ = programs }
func (p *rule) setPosition(position Position) { p.position = position }
func (p *rule) setTarget(v Value) { p.target = v }
func (p *rule) Position() (pos Position) {
    if pos = p.position; !pos.IsValid() {
        if pos = p.target.Position(); !pos.IsValid() {
            for _, prog := range p.program_ {
                if pos = prog.position; pos.IsValid() { break }
            }
        }
    }
    return
}
func (p *rule) ident(ctx Context) (name string) {
    if p == nil {
        erro(ctx, "nil entry")
    } else if p.target == nil {
        erro(at(ctx,p.position), "entry target is nil")
    } else {
        name = p.target.string(ctx)
    }
    return
}
func (p *rule) true(ctx Context) bool { return p.target.true(ctx) }
func (p *rule) float(_ Context) (f float64, _ error) { return 0, nil }
func (p *rule) int(_ Context) (i int64, _ error) { return 0, nil }
func (p *rule) string(ctx Context) string { return p.target.string(ctx) }
func (p *rule) String() string {
    if p.target == nil { return "<nil entry>" }
    return p.target.String()
}
func (p *rule) updated(ctx Context) (res bool) {
    if res = p.target.updated(ctx); res {
        ctx.dirtyMark(p.target)
    }
    return
}
func (p *rule) updatedDeps(ctx Context, v ...Value) []Value {
    return p.target.updatedDeps(ctx, v...)
}
// rule.execute executes the rule program only if the target is outdated.
func (p *rule) execute(ctx Context, a ...Value) (result []Value, traves travestates) {
    switch p.class {
    case PatternRule, pathPatRule:
        erro(ctx, "executing pattern entry '%v'", p.target).debug(1)
        return
    }

    if ctx = (&ruleContext{ctx, p}); len(a)>0 { ctx = &argumentedContext{ctx, a} }

outer:
    for _, program := range p.program_ {
        var pos = program.position
        if !pos.IsValid() { pos = p.Position() }

        var res, t = program.execute(at(ctx, pos))
        result = append(result, merge(res)...)
        traves = append(traves, t...)
        if t.has(traveFail) { break outer }
        for _, s := range t.of(traveCase, traveDone) {
            if s.prog == program { break outer }
        }
    }
    return
}
func (p *rule) get(_ Context, name string) Value {
    switch name {
    case "class": return makeStrlit(p.position, p.class.String())
    case "name" : return p.target //return makeStrlit(p.position, p.name())
    //case "prerequisites": ...
    }
    return nil
}
func (p *rule) recipes() (recipes []Value) {
    for _, prog := range p.program_ {
        for _, recipe := range prog.recipes {
            recipes = append(recipes, recipe)
        }
    }
    return
}
func (p *rule) hasRecipes() (res bool) {
    for _, prog := range p.program_ {
        if res = len(prog.recipes) > 0; res { break }
    }
    return
}
func (p *rule) refs(ctx Context, v Value) bool {
    if p.target.refs(ctx, v) { return true }

    return false

    for _, prog := range p.program_ {
        for _, depend := range prog.depends {
            if depend.refs(ctx, v) { return true }
        }
        for _, recipe := range prog.recipes {
            if recipe.refs(ctx, v) { return true }
        }
    }
    return false
}
func (p *rule) defs(ctx Context, s ...string) (res []*def) {
    res = p.target.defs(ctx, s...)
    for _, prog := range p.program_ {
        for _, depend := range prog.depends {
            res = append(res, depend.defs(ctx, s...)...)
        }
        for _, recipe := range prog.recipes {
            res = append(res, recipe.defs(ctx, s...)...)
        }
    }
    return
}
func (p *rule) expandable(ctx Context) (res bool) {
    if res = p.target.expandable(ctx); res { return }
    if false {
        for _, prog := range p.program_ {
            for _, depend := range prog.depends {
                if res = depend.expandable(ctx); res { return }
            }
            for _, recipe := range prog.recipes {
                if res = recipe.expandable(ctx); res { return }
            }
        }
    }
    return
}
func (p *rule) expand(ctx Context) (res Value) {
    if evo, y := ctx.(*evocation); y {
        vals, t := p.execute(ctx, evo.a...) // evo.x = true
        if vals != nil { res = ease(ctx, vals) }
        if t.has(traveFail) {
            for _, s := range t { erro(at(ctx,s.pos), "%v", s) }
            errostack(ctx, 3).debug(3)
        }
        return
    }

    var target Value
    if target = p.target.expand(ctx); target != p.target {
        // TODO: test if programs are needed to be disclosed??
        res = &rule{
            p.class, target,
            p.arged,
            p.program_,
            p.position,
        }
    } else {
        res = p
    }
    return
}
func (p *rule) delete(  ctx Context) (files []*File, err error) { return p.target.delete(ctx) }
func (p *rule) stamp(   ctx Context) (files []*File, err error) { return p.target.stamp(ctx) }
func (p *rule) traverse(ctx Context) {
    var pc = cast[*programContext](ctx)
    var sc, _ = ctx.(*stemmedContext)
    var target = autoVal(ctx, "@")

    if target == nil {
        erro(ctx, "$@ is not defined").debug(1)
        return
    } else if _entry(ctx) == p {
        var proj = ctx.project()

        if c := cast[*terminal](ctx); c != nil {
            if t := autoVal(c, "@"); t != nil && eq(ctx, t, target) {
                if true { warn(ctx, "%v: %v: %v\n", proj, p, t) }
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

        prompt(ctx, "%v: %v: %v\n", proj, p, target)
        warnstack(ctx, 8, "%v: %v: %v", proj, p, target).debug(16)
    } else {
        ctx = &ruleContext{ctx, p}
    }

ForPrograms:
    for i, prog := range p.program_ {
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
            s.depend = p
            s.prog = prog
        }

        for _, s := range t.of(traveCase, traveDone) {
            if s.prog == prog { break ForPrograms }
        }
    }

    if pc != nil && sc == nil {
        // if sc != nil { depend = sc.stem } else { depend = p }
        pc.traves.add(ctx, traveRule, target).depend = p
    }
    return
}
// FIXME: p.target maybe not the real target
func (p *rule) stat(ctx Context) (si *statinfo) { return p.target.stat(ctx) }
func (p *rule) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*rule); y {
        assert(y, "value is not rule")
        if /*p.class == a.class &&*/ p.target.cmp(ctx, a.target) == cmpEqual {
            if p.owner() == a.owner() {
                res = cmpEqual
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *rule) patterned(ctx Context) bool { return p.target.patterned(ctx) }
func (p *rule) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = p.target.match(ctx, i)
    return
}
func (p *rule) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p.target.stencil(ctx, stems)
    return
}

func (p *rule) option(ctx Context) (res bool, infos []Value) {
    ForPrograms: for _, program := range p.program_ {
        if !program.configure { continue }
        for _, depend := range program.depends {
            g, ok := depend.(*modification)
            if!ok { continue }
            for _, m := range g.list {
                if m.elems[0].string(ctx) != "configure" { continue }
                for _, arg := range m.elems[1:] {
                    a, ok := arg.(*argumented)
                    if!ok { continue }
                    f, ok := a.Value.(flag)
                    if!ok { continue }
                    if f.Value.string(ctx) != "option" { continue }
                    for _, v := range a.args {
                        if p, ok := v.(*pair); ok {
                            if p.key.string(ctx) != "info" { continue }
                            v = p.val
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

func (_ *rule) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *rule) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

func _stemmedContext(ctx Context) *stemmedContext { return cast[*stemmedContext](ctx) }
func _stems(ctx Context) (res []string) {
    if p := _stemmedContext(ctx); p != nil { res = p.stem.stems }
    return
}

type stemmedContext struct {
    Context
    stem *stemmed
}
func (sc *stemmedContext) cast(t reflect.Type) Context { return implcast(sc,t) }
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
func (p *stemmed) expand(ctx Context) (res Value) {
    if v := p.rule.expand(ctx); v != p.rule {
        res = &stemmed{v.(*rule), p.target, p.stems}
    } else {
        res = p
    }
    return
}
func (p *stemmed) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*stemmed); y {
        assert(y, "value is not stemmed")
        if len(p.stems) != len(p.stems) { return }
        for i, stem := range p.stems {
            if stem != a.stems[i] { return }
        }
        res = p.rule.cmp(ctx, a.rule)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
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
