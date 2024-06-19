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
}

type objbase struct {
    valbase
    scope *scope
}
func (_ *objbase) kind() Kind { return KindObject }
func (p *objbase) owner() *project { return p.scope.project }
func (p *objbase) ident(Context) string { panic("inquiring name of an unknown object") }
func (p *objbase) String() string { return fmt.Sprintf("{=obj %p}", p) }
func (p *objbase) string(Context) string { return fmt.Sprintf("{=obj %p}", p) }
func (p *objbase) exists() existence { return existenceMatterless }
func (p *objbase) declscope() *scope { return p.scope }
func (p *objbase) setscope(name string, s *scope) {
    if p.scope != s {
        if p.scope != nil {
            delete(p.scope.elems, name)
        }
        p.scope = s
    }
}

type knownobject struct { // generally named objects
    objbase
    name string // single, or group name if containing '(*)' and corresponding members
}
func (p *knownobject) kind() Kind { return p.objbase.kind()|KindKnownObject }
func (p *knownobject) String() string { return fmt.Sprintf("{=object %s}", p.name) }
func (p *knownobject) string(Context) string { return fmt.Sprintf("{=object %s}", p.name) }
func (p *knownobject) true(Context) bool { return p.name != "" }
func (p *knownobject) ident(Context) string { return p.name }
func (p *knownobject) cmp(ctx Context, v Value) (_ cmpres) {
    if x, y := v.(*knownobject); y {
        if p.scope == x.scope && p.name == x.name {
            return cmpEqual
        }
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}

type origin int

const (
    defUndetermined origin = iota
    defVoid
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
    defProgParam
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
    case defProgParam: s = "param"
    case _defAny:    s = "any"
    default: s = fmt.Sprintf("origin<%d>", o)
    }
    return
}

type auto_defs map[string]*def
func (m auto_defs) len() int { return len(m) }
func (m auto_defs) String() (s string) {
    for _, d := range m {
        if s == "" { s = "{" } else { s += "," }
        s += d.String()
    }
    if s != "" { s += "}" }
    return
}

func _automatic(c Context) *automatic { return cast[*automatic](c) }
func suppress_always(string) bool { return true }
func suppress_never(string) bool { return false }

type automatic struct {
    Context
    sync.RWMutex
    defs auto_defs
    suppress func(string) bool
}
func (ac *automatic) cast(t reflect.Type) Context { return implcast(ac, t) }
func (ac *automatic) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case act_arguments:
        if x := try[[]Value](ctx, get_arguments{}); len(x) > 0 { ac.args(ctx, x) }
        return
    }
    return do_bits(ctx, ac.Context, op, propExAuto)
}
func (ac *automatic) amend(ctx Context, name string, val Value) (out *def, res Value) {
    if d := ac.search(ctx, name); d == nil {
        return ac.set(ctx, name, val)
    } else if res = d.value; d.value != val {
        out, d.value = d, val
    }
    return
}
func (ac *automatic) has(s string) (y bool) { _, y = ac.defs[s]; return }
func (ac *automatic) def(ctx Context, name string) (res *def, y bool) {
    ac.Lock() ; defer ac.Unlock()
    res, y = ac.defs[name]
    return
}
func (ac *automatic) search(ctx Context, name string) (res *def) {
    if res, _ = ac.def(ctx, name) ; res != nil { return }
    if ac.suppress != nil && ac.suppress(name) { return }
    if t := _automatic(ac.Context) ; t == ac {
        erro(ctx, "%v: loop auto context", name).debug()
        trace(ctx)
        return
    } else if t != nil {
        return t.search(ctx, name)
    }
    return
}
func (ac *automatic) set(ctx Context, name string, val Value) (out *def, old Value) {
    if name == "-" {
        if x, y := val.(*def); y && x.origin != defConfig {
            notestack(ctx, 3, "set $- to def (%v): %v", x.origin, x).debug(16)
        }
    }

    out, _ = ac.def(ctx, name)

    if out != nil {
        old = out.value
    } else {
        var pos Position
        if val == nil {
            pos = _position(ctx)
        } else {
            pos = val.Position()
        }

        out = &def{}
        out.name = name
        out.position = pos
        out.scope = _scope(ctx)

        ac.Lock()
        ac.defs[name] = out
        ac.Unlock()
    }

    if false && checkpoints && name == "-" {
        var x = _execution(ctx)
        if x != nil && &x.automatic == ac {
            note(ctx, "%v %v", val, val.expand(ctx))
            note(ctx, "%v", ts(ctx)).debug()
        }
    }

    // out.Lock()
    out.value = val
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
                erro(ctx, "unknown arg: %v %v", ts(p.key), ts(p.val))
                erro(ctx, "%v", ts(ctx)).debug(5)
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
            erro(at(ac,a.value), "arg '%s' not set", a.name).debug()
            return
        } else {
            d.origin = defProgParam
        }

        if d, y := ac.defs[a.name]; !y || d == nil {
            erro(at(ac,a.value), "arg '%s' not set", a.name).debug()
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

func auto_find(ctx Context, name string) (d *def) {
    if a := _automatic(ctx); a != nil { d = a.search(ctx, name) }
    return
}

func auto_get(ctx Context, name string) (_ Value) {
    if d := auto_find(ctx, name); d != nil { return d.value }
    return
}

func auto_set(ctx Context, name string, val Value) (out *def, res Value) {
    if ac := _automatic(ctx); ac != nil { out, res = ac.set(ctx, name, val) }
    return
}

func hasAutoInner(ctx Context, target Value) (res bool) {
    if ac, n := _automatic(ctx), 0; ac != nil {
        for ac = _automatic(inner(ac)); ac != nil; ac = _automatic(inner(ac)) {
            if n > 1 { return true }
            if t := auto_get(ac, "@"); t != nil && eq(ctx, t, target) { n += 1 }
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
func (a *auto) sel(ctx Context, name string) (res any) {
    if name == "value" { res = auto_get(ctx, a.name) }
    return
}
func (a *auto) refs(ctx Context, v Value) (res bool) {
    if x, y := v.(*auto); y && (x == a || x.name == a.name) { return true }
    if d := a.def(ctx); d != nil && d.value != nil { res = d.value.refs(ctx, v) }
    return
}
func (a *auto) defs(ctx Context, s ...string) (res []*def) {
    if d := a.def(ctx); d != nil { res = append(res, d) }
    return
}
func (a *auto) def(ctx Context) *def { return auto_find(ctx, a.name) }
func (a *auto) set(ctx Context, value Value, app ...Value) {
    if value == nil && app != nil { if d := auto_find(ctx, a.name); d != nil {
        d.value = ease(ctx, append(merge(d.value), app...))
        d.position = a.position
        return
    }}

    d, _ := auto_set(ctx, a.name, ease(ctx, append(merge(value), app...)))
    if d != nil {
        d.position = a.position
    } else if true {
        warnstack(at(ctx,a), 3, "set auto failed: %v: %v %v", a.name, value, app).debug(16)
    }
}
func (a *auto) isDigit() bool { return IsDigits(a.name) }
func (a *auto) isPlaceholder() bool { return a.name == "_" }
func (a *auto) expandable(ctx Context) (res bool) {
    if ex_auto(ctx) {
        var d = auto_find(ctx, a.name)
        return d != nil && d.value != nil
    }
    return
}
func (a *auto) expand(ctx Context) (res Value) {
    if ex_auto(ctx) {
        var d = auto_find(ctx, a.name)
        if d != nil && d.value != nil { return d }
    }
    return a
}
func (a *auto) invoke(ctx Context, o, v []Value) (res Value) {
    if d := auto_find(ctx, a.name); d != nil { res = d.invoke(ctx, o, v) }
    return
}
func (a *auto) cmp(ctx Context, v Value) (res cmpres) {
    if o, y := v.(*auto); y && (a == o || a.name == o.name) {
        res = cmpEqual
    } else if val := auto_get(ctx, a.name); val != nil {
        res = val.cmp(ctx, v)
    }
    return
}
func (a *auto) stat(ctx Context) (si *statinfo) {
    if val := auto_get(ctx, a.name); val != nil { si = val.stat(ctx) }
    return
}
func (a *auto) traverse(ctx Context) {
    if val := auto_get(ctx, a.name); val != nil { val.traverse(ctx) }
    return
}

// A def represents a definition, it's a caller but mustn't be a Valuer.
type def struct {
    knownobject
    origin origin
    value Value
}
func (d *def) kind() Kind { return d.knownobject.kind()|KindDef }
func (d *def) Position() (pos Position) {
    if  pos = d.position ; !pos.valid() && d.value != nil {
        pos = d.value.Position()
    }
    return
}
func (d *def) ts(t string) string {
    return fmt.Sprintf("{=%s %s⇒%v}", t, d.name, ts(d.value))
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

    if !ex_def_value(ctx) && expandable(ctx, ctx.a...) {
        return d
    } else if v == nil {
        return
    }

    var x = automatic{ Context:ctx, defs:make(auto_defs), suppress:
        func(s string) bool { return s == "_" || IsDigits(s) } }

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
    if checkpoints { defer d.set_check(ctx, origin, value, app...) }

    defer trace(ctx)

    if origin == defUndetermined { origin = defVoid }

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

    if checkpoints && truly(ctx, is_test_mode{}) {
        if origin != defExpand0 && value != nil && value.String() == "$(auto ,$(a))" && auto_find(ctx, "a") == nil {
            defer func(v Value) {
                if len(vals) != 1 {
                    erro(ctx, "%v → %v → %v", ts(v), ts(vals), ts(d.value)).debug()
                } else if ts(vals[0]) != "{=delegate {=auto a}}" {
                    erro(ctx, "%v → %v → %v", ts(v), ts(vals[0]), ts(d.value)).debug()
                }
            } (value)
        }
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
    defer trace(ctx)
    if ex_def(ctx) && isNull(d.value) && (value != nil || len(app) > 0) {
        erro(ctx, "%v ; %v %v", d, value, app).debug()
    }
    if origin == defExpand0 && (!isNull(value) || app != nil) && isNull(d.value) {
        erro(ctx, "%v ; %v %v", d, value, app).debug()
    }
    if !d.position.valid() && d.name != ".goals" {
        erro(ctx, "%v ; %v %v", d, value, app).debug()
    }
}
func (d *def) append(ctx Context, a ...Value) { if len(a) > 0 { d.set(ctx, d.origin, nil, a...) } }
func (d *def) invoke(ctx Context, o, a []Value) (res Value) {
	res, _ = evoke(ctx, d, o, a)
    return
}
func (d *def) xexec(ctx Context, value Value, a ...Value) (res Value) {
    if isTrivial(value) { return }

    var cmd string
    if cmd = value.string(ctx); cmd == "" {
        warn(ctx, "%v: empty command (value=%v)", d.name, value).debug()
        return
    }

    // TODO: options for running command in the specified container
    var stdout, stderr bytes.Buffer
    var sh = exec.Command("sh", "-c", cmd)
    sh.Stdout, sh.Stderr = &stdout, &stderr
    if err := sh.Run(); err != nil {
        erro(ctx, "%v: execute command failed: %v", d.name, err)
        erro(ctx, "%v: execute command: %s", d.name, cmd).debug()
        stdout.Reset()
        stderr.Reset()
        return
    }

    var pos = value.Position()
    if !pos.IsValid() { pos = _position(ctx) }
    res = makeStrlit(pos, strings.TrimSpace(stdout.String()))
    stdout.Reset()
    stderr.Reset()
    return
}
func (d *def) sel(ctx Context, name string) (res any) {
    switch name {
    case "name" : return d.name
    case "value":
        // d.Lock() ; defer d.Unlock()
        return d.value
    default:
        erro(at(ctx,d.position), "def: no such operator `%s'", name).debug()
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

type undetermined struct {
    token token
    identifier Value
    value Value
}
func (_ *undetermined) kind() Kind { return KindObject|KindUndetermined }
func (p *undetermined) Position() Position { return p.identifier.Position() }
func (p *undetermined) String() (s string) {
    s  = p.identifier.String()
    s += p.token.String()
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
    var i = p.identifier.expand(ctx)
    var v = p.value.expand(ctx)
    if i != p.identifier || v != p.value {
        res = &undetermined{p.token, i, v}
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
func (p *undetermined) match(ctx Context, i any) (full bool, s any, stems []string) { return }
func (p *undetermined) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
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
func (p *builtin) invoke(ctx Context, o, a []Value) (res Value) {
	res, _ = evoke(ctx, p, o, a)
    return
}
func (p *builtin) refs(ctx Context, v Value) (res bool) {
    if o, y := v.(*builtin); y { res = o == p /* || p.name == o.name */ }
    return
}
func (p *builtin) benchmark_expand(ctx Context, t0 time.Time, v reflect.Value) {
    if d := time.Now().Sub(t0); d > 1*time.Second {
        note(ctx, "%v: slow: %v", p, d).debug(3)
    } else if f := v.Elem().FieldByName("timing"); !f.IsValid() {
        if false { note(ctx, "%v: %v", p, d).debug() }
    } else if f.Type().Kind() == reflect.Bool && f.Bool() {
        note(ctx, "%v: %v", p, d).debug()
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

    var force = /* ex_final(ctx) || */ builtinForceField(ctx, _v, _i, false)

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
        erro(ctx, "no method: %v (%s)", p.t.Name(), ts(_v)).debug(16)
        return
    }
}
func (p *builtin) cmp(ctx Context, v Value) (res cmpres) {
    if b, y := v.(*builtin); y {
        if p.t == b.t /* || p.name == a.name */ { res = cmpEqual }
        if checkpoints {
            if res != cmpEqual {
                if p.t == b.t { erro(ctx, "%v", ts(v)).debug() }
                if p.name == b.name { erro(ctx, "%v", ts(v)).debug() }
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
    return
}

type get_entry struct{}

type rule_context struct { Context ; rule *rule }
func (p *rule_context) Position() Position { return p.rule.Position() }
func (p *rule_context) cast(t reflect.Type) Context { return implcast(p,t) }
func (p *rule_context) ts(t string) string {
    return fmt.Sprintf("{=%s %v %v}", t, p.rule, ts(p.Context))
}
func (p *rule_context) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case get_entry: return p.rule
    }
    return do_bits(ctx, p.Context, op)
}

func _entry(ctx Context) entry { return try[entry](ctx, get_entry{}) }

type entry interface {
    destiny() Value // aka target
    programs(...*program) []*program
    executer
    Object
}

func hasRecipes(e entry) (_ bool) {
    for _, p := range e.programs() {
        if 0 < len(p.recipes) { return true }
    }
    return
}

func executeEntry(ctx Context, e entry, args ...Value) (result []Value, okay bool) {
    result = e.execute(ctx, args...)
    okay = true
    return
}

// rule represents a declared rule entry.
type rule struct {
    target Value
    program []*program
    arged []Value
}
func (_ *rule) kind() Kind { return KindObject|KindRule }
func (p *rule) destiny() Value { return p.target }
func (p *rule) owner() *project { return p.program[0].project }
func (p *rule) Position() (_ Position) {
    if pos := p.target.Position(); pos.IsValid() {
        return pos
    }
    for _, prog := range p.program {
        if prog.position.IsValid() {
            return prog.position
        }
    }
    return
}
func (p *rule) programs(a ...*program) []*program {
    if 0 < len(a) { p.program = a }
    return p.program
}
func (p *rule) ident(ctx Context) (name string) {
    if p == nil {
        erro(ctx, "nil entry")
    } else if p.target == nil {
        erro(at(ctx,p), "entry target is nil")
    } else {
        name = p.target.string(ctx)
    }
    return
}
func (p *rule) true(ctx Context) bool { return p.target.true(ctx) }
func (p *rule) float(_ Context) (_ float64, _ error) { return 0, nil }
func (p *rule) int(_ Context) (_ int64, _ error) { return 0, nil }
func (p *rule) string(ctx Context) string { return p.target.string(ctx) }
func (p *rule) String() string {
    if p.target == nil { return "<nil entry>" }
    return p.target.String()
}
func (p *rule) updated(ctx Context) (res bool) {
    if res = p.target.updated(ctx); res {
        do(ctx, act_dirty_mark{[]Value{ p.target }})
    }
    return
}
func (p *rule) updatedDeps(ctx Context, v ...Value) []Value {
    return p.target.updatedDeps(ctx, v...)
}
func (p *rule) execute(ctx Context, a ...Value) (result []Value) {
    if p.patterned(ctx) {
        erro(ctx, "executing pattern entry '%v'", p.target).debug()
        trace(ctx)
    }

    ctx = &rule_context{&argumented_context{at(ctx, p), a}, p}

    for _, program := range p.program {
        if v := program.execute(at(ctx, p)); v != nil {
            result = append(result, v)
        }
    }
    return
}
func (p *rule) recipes() (recipes []Value) {
    for _, prog := range p.program {
        for _, recipe := range prog.recipes {
            recipes = append(recipes, recipe)
        }
    }
    return
}
func (p *rule) refs(ctx Context, v Value) bool {
    if p.target.refs(ctx, v) { return true }

    return false

    for _, prog := range p.program {
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
    for _, prog := range p.program {
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
        for _, prog := range p.program {
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
func (p *rule) expand(ctx Context) (_ Value) {
    defer trace(ctx)

    if x, y := ctx.(*evocation); y {
        return ease(ctx, p.execute(ctx, x.a...))
    }

    var target = p.target.expand(ctx)
    if equal(ctx, target, p.target) { return p }

    return &rule{ target, p.program, p.arged }
}
func (p *rule) delete(  ctx Context) (files []*File, err error) { return p.target.delete(ctx) }
func (p *rule) stamp(   ctx Context) (files []*File, err error) { return p.target.stamp(ctx) }
func (p *rule) traverse(ctx Context) {
    var pc = _execution(ctx)
    var target = auto_get(ctx, "@")

    defer trace(ctx)

    if target == nil {
        erro(ctx, "$@ is not defined").debug()
        return
    }

    if _entry(ctx) == p {
        var proj = _project(ctx)

        if c := cast[*terminal](ctx); c != nil {
            if t := auto_get(c, "@"); t != nil && eq(ctx, t, target) {
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
        ctx = &rule_context{ctx, p}
    }

ForPrograms:
    for _, prog := range p.program {
        var ctx = at(ctx, prog.position)
        var next bool
        func(){
            defer func() {
                if t := recover(); t != nil {
                    if _, next = t.(traverse_next); next {
                        return
                    }
                    erro(ctx, "%v", t).debug()
                    trace(ctx)
                }
            }()

            var v = prog.execute(ctx)
            if pc != nil && v != nil {
                pc.values = append(pc.values, merge(v)...)
            }
        }()
        if next { continue ForPrograms }

        // if pc != nil && sc != nil {
        //     s := pc.traves.add(ctx, traveRule, target)
        //     s.depend, s.prog = p, prog
        // }
    }

    // if pc != nil && sc == nil {
    //     // if sc != nil { depend = sc.stem } else { depend = p }
    //     pc.traves.add(ctx, traveRule, target).depend = p
    // }
    return
}
// FIXME: p.target maybe not the real target
func (p *rule) stat(ctx Context) (si *statinfo) { return p.target.stat(ctx) }
func (p *rule) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*rule); y {
        if p.target.cmp(ctx, a.target) == cmpEqual {
            if p.owner() == a.owner() { res = cmpEqual }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        res = p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *rule) patterned(ctx Context) bool { return p.target.patterned(ctx) }
func (p *rule) match(ctx Context, i any) (full bool, s any, stems []string) {
    full, s, stems = p.target.match(ctx, i)
    return
}
func (p *rule) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p.target.stencil(ctx, stems)
    return
}

func _stemmed_context(ctx Context) *stemmed_context { return cast[*stemmed_context](ctx) }
func _stems(ctx Context) (res []string) {
    if p := _stemmed_context(ctx); p != nil { res = p.stem.stems }
    return
}

type stemmed_context struct { Context ; stem *stemmed }
func (sc *stemmed_context) cast(t reflect.Type) Context { return implcast(sc,t) }

type stemmed struct {
    *rule
    target Value
    stems []string
}
func (p *stemmed) kind() Kind { return p.rule.kind()|KindStemmedRule }
func (p *stemmed) destiny() Value { return p.target/* versus p.rule.target */ }
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
    t.target = p.target
    t.traverse(&stemmed_context{ ctx, p })
}
