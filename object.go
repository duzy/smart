//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "os/exec"
    "reflect"
    "regexp"
    "strings"
    "strconv"
    "sync"
    "bytes"
    "unsafe"
    "time"
    "fmt"
)

type object interface {
    Value
    owner() *project
}

type objbase struct{ valbase ; scope *scope }
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

type knownobject struct{ // generally named objects
    objbase
    name string // single, or group name if containing '(*)' and corresponding members
}
func (p *knownobject) kind() Kind { return p.objbase.kind()|KindKnownObject }
func (p *knownobject) String() string { return fmt.Sprintf("{=object %s}", p.name) }
func (p *knownobject) string(Context) string { return fmt.Sprintf("{=object %s}", p.name) }
func (p *knownobject) true(Context) bool { return p.name != "" }
func (p *knownobject) ident(Context) string { return p.name }

type origin uint

const (
    defUndetermined origin = 0
    defVoid origin = 1<<(iota-1)
    defConfig  // configure
    defDecl    // declaration names
    defStatic  // auto-expand within a code block at parse time (aka def/end, for/end)
    defExpand0 //   =  normal value
    defExpand1 //  :=  expand delegates (simple expand)
    defExpand2 // ::=  expand all (delegates, closures, paths)
    defExpand3 // ;:=  TODO: expand as plain
    defExecute //  !=  value to be executed
    defAssign0 // ?=
    defAssign1 // +=
    defAssign2 // =+
    defAssign3 // -=
    defAssign4 // -+=
    defAssign5 // -=+
    defParam   // program parameter
)

var origin_names = []string{
    "void", "config", "conf_dir", "conf_ref", "decl", "static",
    "expand_0", "expand_1", "expand_2", "expand_3", "execute",
    "assign_0", "assign_1", "assign_2", "assign_3", "assign_4", "assign_5",
    "param", "test_val", "test_str",
}

func (o origin) String() (s string) {
    for i := 0; i < len(origin_names); i += 1 {
        if o&(1<<i) != 0 {
            if s != "" { s += "|" }
            s += origin_names[i]
        }
    }
    return
}

type def_map map[string]*def
func (m def_map) len() int { return len(m) }
func (m def_map) String() (s string) {
    seen := make(map[string]struct{}) // NOTE: digits alias: 1 2 3...
    for _, d := range m {
        if _, y := seen[d.name]; y { continue }
        if s == "" { s = "{" } else { s += "," }
        seen[d.name] = struct{}{}
        s += d.String()
    }
    if s != "" { s += "}" }
    return
}

func _automatic(c Context) *automatic { return cast[*automatic](c) }

type find_auto struct{ s string }
type set_auto  struct{ o origin; s string; v Value }
type res_auto  struct{ d *def; v Value }
type automatic struct{
    Context
    sync.RWMutex
    defs def_map
    params map[string]*auto
}
func (ac *automatic) cast(t reflect.Type) Context { return icast(ac, t) }
func (ac *automatic) inner() Context { return ac.Context }
func (ac *automatic) ts(t string) string {
    s := ac.defs.String()
    if s != "" { s += " " }
    return "{="+t+" "+s+ts(ac.Context)+"}"
}
func (ac *automatic) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case init_args:
        if t.automatic == nil {
            panic("automatic.init_args")
        }
    case find_auto:
        if d, _ := ac.defs[t.s]; d != nil {
            return d
        }
    case set_auto:
        d, v := ac.set(ctx, t.o, t.s, t.v)
        return res_auto{ d, v }
    }
    return ac.Context.do(ctx, op)
}
func (ac *automatic) amend(ctx Context, name string, val Value) (out *def, res Value) {
    if d, _ := ac.do(ctx, find_auto{name}).(*def); d == nil {
        return ac.set(ctx, defVoid, name, val)
    } else if res = d.value; d.value != val {
        out, d.value = d, val
    }
    return
}
func (ac *automatic) has(s string) (y bool) { _, y = ac.defs[s]; return }
func (ac *automatic) set(ctx Context, o origin, name string, val Value) (out *def, old Value) {
    if name == "-" && val != nil {
        if x, y := val.(*def); y && x.o != defConfig {
            errostack(ctx, 3, "set $- to def (%v): %v", x.o, x).debug(16)
        }
    }

    if out, _ = ac.defs[name]; out != nil {
        old = out.value
        // out.Lock()
        out.value = val
        // out.Unlock()
        return
    }

    out = &def{o:o, value:val}
    out.position = _position(ctx)
    out.scope = _scope(ctx)
    out.name = name

    ac.Lock()
    ac.defs[name] = out
    ac.Unlock()
    return
}
func (ac *automatic) args(ctx Context, vals []Value) {
    type arg struct{ id, name string ; value Value }

    if vals == nil { return }

    var argn int // setup named/number parameters ($1, $2, etc.)
    var args = make(map[string]*arg, len(vals)) // compact args: combine duplicated pairs

    for i, val := range vals {
        a := &arg{ id: strconv.Itoa(argn+1) }

        if p, y := val.(*pair); y {
            if a.name = p.key.string(ctx); a.name == "" {
                erro(pc(ctx,a), "empty name: %v", p.key).trace()
                return
            }

            if ac.params != nil {
                if _, y = ac.params[a.name]; !y {
                    var keys = reflect.ValueOf(ac.params).MapKeys()
                    errostack(pc(ctx,a), 16, "unknown arg#%d: %v ; known: %v", i, p, keys).trace()
                    return
                }
            }

            if t, y := args[a.name]; y {
                if x, y := t.value.(*list); y {
                    x.elems = append(x.elems, merge(p.val)...)
                } else {
                    a.value = _list(t.value)
                }
                continue
            }

            a.value = p.val
        } else {
            a.name, a.value = _param_name(ctx, argn), scalarize(val)
            if a.name == "" { a.name = a.id }
        }

        if a.id != a.name { args[a.id] = a }
        args[a.name] = a
        argn += 1

        if d, _ := ac.set(ctx, defParam, a.name, a.value); d == nil {
            erro(ac, "arg '%s' not set", a.name).trace()
            return
        }

        if d, y := ac.defs[a.name]; !y || d == nil {
            erro(ac, "arg '%s' not set", a.name).trace()
            return
        } else if a.id != "" && a.id != a.name {
            ac.Lock()
            ac.defs[a.id] = d // NOTE: alias or replacement
            ac.Unlock()
        }
    }
    return
}

func auto_find(ctx Context, name string) (d *def) {
    d, _ = do(ctx, find_auto{name}).(*def)
    return
}

func auto_get(ctx Context, name string) (_ Value) {
    if d := auto_find(ctx, name); d != nil { return d.value }
    return
}

func auto_set(ctx Context, o origin, name string, val Value) (_ *def, _ Value) {
    t, _ := do(ctx, set_auto{o, name, val}).(res_auto)
    if t.d != nil && name == "TYPE" && _project(ctx).name == "configure.base" {
        note(ctx, "%v %v", t.d, t.v)
        note(ctx, "%v", ts(ctx)).debug()
    }
    return t.d, t.v
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

type auto struct{ knownobject }
func (a *auto) kind() Kind { return a.knownobject.kind()|KindAuto }
func (a *auto) hash(ctx Context) uint64 { return fnv1(ctx, a, a.name) }
func (a *auto) String() string { return a.name }
func (a *auto) string(ctx Context) (res string) {
    if d := a.def(ctx); d != nil && d.value != nil { res = d.value.string(ctx) }
    return
}
func (a *auto) ts(ctx Context, t string) string {
    return "{" + lp(ctx, a.position, t) + " " + a.name + "}"
}
func (a *auto) sel(ctx Context, name string) (res any) {
    if name == "value" { res = auto_get(ctx, a.name) }
    return
}
func (a *auto) defs(ctx Context, s ...string) (res []*def) {
    if d := a.def(ctx); d != nil { res = append(res, d) }
    return
}
func (a *auto) def(ctx Context) *def { return auto_find(ctx, a.name) }
func (a *auto) set(ctx Context, o origin, value Value, app ...Value) {
    if value == nil && app != nil {
        if d := auto_find(ctx, a.name); d != nil {
            d.value = ease(ctx, append(merge(d.value), app...))
            if false { d.position = a.position }
            return
        }
    }

    d, _ := auto_set(ctx, o, a.name, ease(ctx, append(merge(value), app...)))
    if d != nil {
        if false { d.position = a.position }
    } else {
        errostack(ctx, 3, "set auto failed: %v: %v %v", a.name, value, app).trace()
    }
}
func (a *auto) isDigit() bool { return IsDigits(a.name) }
func (a *auto) isPlaceholder() bool { return a.name == "_" }
func (a *auto) expand(ctx Context) (res Value) {
    if d := auto_find(ctx, a.name); d != nil { res = d }
    return
}
func (a *auto) evoke(ctx *evocation) (res Value) {
    if d := auto_find(ctx, a.name); d != nil { res = d.evoke(ctx) }
    return
}
func (a *auto) invoke(ctx Context, o, v []Value) (res Value) {
    if d := auto_find(ctx, a.name); d != nil { res = evoke(ctx, d, o, v) }
    return
}
func (a *auto) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer check_cmp(ctx, a, v, &res) }
    switch x := v.(type) {
    case *auto:
        if a == x || a.name == x.name { res = cmpEqual }
    case *def:
        if false {
            if d := auto_find(ctx, a.name); d != nil { res = d.cmp(ctx, x) }
        } else if x.name == a.name /* && x.o == defVoid */ {
            return cmpEqual
        }
    case *list:
        if x.len() == 1 { res = a.cmp(ctx, x.elems[0]) }
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

type def_evoke struct{ *evocation }
func (c def_evoke) inner() Context { return c.evocation }
func (c def_evoke) cast(t reflect.Type) Context { return icast(c, t) }
func (c def_evoke) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case param_name: return
    case find_auto:
        if IsDigits(t.s) {
            if x, y := c.defs[t.s]; y {
                return x
            } else {
                return
            }
        }
    }
    return c.evocation.do(ctx, op)
}

// A def represents a definition, it's a caller but mustn't be a Valuer.
type def struct{
    knownobject
    value Value
    o origin
}
func (d *def) kind() Kind { return d.knownobject.kind()|KindDef }
func (d *def) hash(ctx Context) uint64 { return fnv1(ctx, d, d.value) }
func (d *def) Position() (pos Position) {
    if d != nil {
        pos = d.position
        if !pos.valid() && d.value != nil {
            pos = d.value.Position()
        }
    }
    return
}
func (d *def) streq() (s string) {
    switch d.o {
    case defExpand0: s =   "="
    case defExpand1: s =  ":="
    case defExpand2: s = "::="
    case defExpand3: s = ";:="
    case defExecute: s =  "!="
    default:         s =   "⇒"
    }
    return
}
func (d *def) ts(ctx Context, t string) string {
    return "{" + lp(ctx, d.position, t) + " " + d.name + "}"
}
func (d *def) String() (s string) {
    var value Value
    {
        // d.Lock()
        value = d.value
        // d.Unlock()
    }

    if s = d.name + d.streq(); value != nil {
        s += value.String()
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
func (d *def) expand(Context) Value { return d }
func (d *def) evoke(ctx *evocation) (res Value) {
    // d.Lock()
    o, v := d.o, d.value
    // d.Unlock()

    if ctx.a != nil { ctx.a = expand(ctx.Context, ctx.a...) }
    if v == nil {
        return
    }

    dev := def_evoke{ctx}
    ctx.args(dev, ctx.a)

    if v = v.expand(dev); v == nil {
        return
    }
    if o == defExecute {
        v = d.xexe(dev, v)
    }
    if false { v = scalarize(v) }
    return v
}
func (d *def) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer check_cmp(ctx, d, v, &res) }
    switch x := v.(type) {
    case *def:
        if d == x { return cmpEqual }
        if d.name == d.name  {
            var v1, v2 Value
            {
                // d.Lock()
                v1 = d.value
                // d.Unlock()
            }
            {
                // a.Lock()
                v2 = x.value
                // a.Unlock()
            }
            if v1 == nil {
                if v2 == nil { return cmpEqual }
            } else if v2 != nil {
                return v1.cmp(ctx, v2)
            }
        }
    case *auto:
        if false {
            if d2 := auto_find(ctx, x.name); d2 != nil { res = d.cmp(ctx, d2) }
        } else if d.name == x.name /* && d.o == defVoid */ {
            return cmpEqual
        }
    case *list:
        if x.len() == 1 { res = d.cmp(ctx, x.elems[0]) }
    }
    return
}
func (d *def) origin(ctx Context, o origin) (res origin) {
    if d.o == o { return o }

    if checkpoints {
        if d.o != defUndetermined && (o == defVoid || o == defUndetermined) {
            erro(pc(ctx,d), "%v: %v → %v", d.name, d.o, o).trace()
        }
    }

    res, d.o = d.o, o
    return
}
func (d *def) val(ctx Context, vals []Value) {
    var val Value
    if n := len(vals); n == 1 {
        val = vals[0]
    } else if 1 < n {
        val = _list(vals...)
    }
    d.set(ctx, val)
}
func (d *def) set(ctx Context, value Value, app ...Value) {
    if checkpoints && d.o == defConfig && d.value != nil {
        errostack(pc(pc(ctx,value),d.value), 1, "duplicated %v %v → %v %v", d.o, d, value, app).trace()
    }
    if value == d.value && len(app) == 0 {
        return
    }

    var vals []Value
    if value != nil { vals = merge(value) }

    var a bool
    if a = len(app) > 0; a { vals = append(vals, app...) }
    if a && d.value != nil {
        // d.Lock()
        var v = d.value
        // d.Unlock()
        vals = append(merge(v), vals...)
    }

    // d.Lock()
    if n := len(vals); 1 < n {
        if true || d.o == defExpand0 {
            d.value = _list(vals...)
        } else {
            l, t := new(list), Value(nil)
            for _, v := range vals {
                if isNull(v) {
                    if t == nil { t = v }
                } else {
                    l.elems = append(l.elems, v)
                }
            }
            if l.len() == 0 && t != nil {
                d.value = t
            } else {
                d.value = l
            }
        }
    } else if 1 == n {
        d.value = vals[0]
    } else if d.o == defExecute {
        d.value = nil
    } else if d.position.valid() {
        d.value = _null(d.position)
    } else {
        d.value = _null(_position(ctx))
    }
    // d.Unlock()
    return
}
func (d *def) append(ctx Context, a ...Value) { if len(a) > 0 { d.set(ctx, nil, a...) } }
func (d *def) xexe(ctx Context, value Value, a ...Value) (res Value) {
    if isTrivial(value) { return }

    var cmd string
    if cmd = value.string(ctx); cmd == "" {
        notestack(pc(ctx,value), 3, "%v: empty command (value=%v)", d.name, value).debug()
        return
    }

    // TODO: options for running command in the specified container
    var stdout, stderr bytes.Buffer
    var sh = exec.Command("sh", "-c", cmd)
    sh.Stdout, sh.Stderr = &stdout, &stderr
    defer func() {
        stdout.Reset()
        stderr.Reset()
    } ()

    if e := sh.Run(); e != nil {
        erro(pc(ctx,value), "%v: execute command failed: %v", d.name, e)
        errostack(pc(ctx,value), 3, "%v: execute command: %s", d.name, cmd).trace()
        return
    }

    var pos = value.Position()
    if !pos.IsValid() { pos = _position(ctx) }
    res = _raw(pos, strings.TrimSpace(stdout.String()))
    return
}
func (d *def) sel(ctx Context, name string) (res any) {
    if x, y := d.value.(seler); y { res = x.sel(ctx, name) }
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

type undetermined struct{
    token token
    identifier Value
    value Value
}
func (_ *undetermined) kind() Kind { return KindObject|KindUndetermined }
func (p *undetermined) hash(ctx Context) uint64 { return fnv1(ctx, p, p.token, p.identifier, p.value) }
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
func (p *undetermined) float(Context) (_ float64) { return }
func (p *undetermined) int(Context) (_ int64) { return }
func (p *undetermined) updated(_ Context) bool { return false }
func (p *undetermined) updatedDeps(_ Context, _ ...Value) []Value { return nil }
func (p *undetermined) defs(ctx Context, s ...string) (res []*def) {
    return append(p.identifier.defs(ctx, s...), p.value.defs(ctx, s...)...)
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
func (p *undetermined) stamp(ctx Context) (files []*file) { return }
func (p *undetermined) delete(ctx Context) (files []*file) { return }
func (p *undetermined) patterned(ctx Context) bool { return false }
func (p *undetermined) match(ctx Context, i any) (_ bool, _ any, _ []string) { return }
func (p *undetermined) stencil(ctx Context, ss []string) (Value, []string) { return p, ss }
func (p *undetermined) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer check_cmp(ctx, p, v, &res) }
    switch x := v.(type) {
    case *undetermined:
        if p.identifier.cmp(ctx, x.identifier) == cmpEqual {
            if p.value.cmp(ctx, x.value) == cmpEqual { return cmpEqual }
        }
    case *list:
        if len(x.elems) == 1 { return p.cmp(ctx, x.elems[0]) }
    }
    return
}

const max_expand = 32

func builtinFinalField(ctx Context, bv reflect.Value, bi any, force bool) bool {
    if f := bv.Elem().FieldByName("final"); f.IsValid() && f.Kind() == reflect.Bool {
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

// A builtin represents a built-in function. builtins don't have a valid type.
type builtin struct{ knownobject ; t reflect.Type }
func (p *builtin) kind() Kind { return p.knownobject.kind()|KindBuiltin }
func (p *builtin) hash(ctx Context) uint64 { return fnv1(ctx, p, p.name) }
func (p *builtin) is_x() bool { return reflect.PointerTo(p.t).Implements(builtin_x_t) }
func (p *builtin) String() string { return p.name }
func (p *builtin) true(Context) bool { return p.t != nil }
func (p *builtin) expand(Context) Value { return p }
func (p *builtin) benchmark(ctx *evocation, t time.Time, v reflect.Value) {
    var n = time.Now()
    if d := n.Sub(t); d > 2*time.Second {
        var a = xmerge(_final(ctx), ctx.a...)
        var m = time.Since(n)//; %v %v
        notestack(pc(ctx,p), 16, "slow %v: %v, %v (%d → %d args)", p, d, m, len(ctx.a), len(a)).debug(256)
    } else if f := v.Elem().FieldByName("timing"); f.IsValid() {
        if f.Type().Kind() == reflect.Bool && f.Bool() {
            notestack(pc(ctx,p), 16, "%v: %v", p, d).debug(256)
        }
    }
}
func (p *builtin) evoke(ctx *evocation) (res Value) {
    _v := reflect.New(p.t)
    _i := _v.Interface()

    defer p.benchmark(ctx, time.Now(), _v)

    if f := _v.Elem().FieldByName("builtinbase"); !f.IsValid() {
        errostack(pc(ctx,_i), 8, "no such field: %s.builtinbase", _v.Elem().Type()).trace()
    } else if f.CanAddr() {
        b := (*builtinbase)(unsafe.Pointer(f.Addr().Pointer()))
        b.evocation = ctx
    } else if f = _v.Elem().FieldByName("evocation"); !f.IsValid() {
        errostack(pc(ctx,_i), 8, "no such field: %s.evocation", _v.Elem().Type()).trace()
    } else if f.CanSet() {
        f.Set(reflect.ValueOf(ctx))
    } else if f.CanAddr() && f.Addr().CanSet() {
        f.Addr().SetPointer(unsafe.Pointer(ctx))
    } else {
        errostack(pc(ctx,_i), 8, "cannot set field: %s.evocation", _v.Elem().Type()).trace()
    }

    if ctx.o != nil { ctx.o = _opts(ctx, _v, ctx.o) }

    if x, y := _i.(builtin_x); y {
        if v := x.x(); v != nil {
            return ease(ctx, v)
        }
    } else {
        errostack(pc(ctx,p), 3, "no method: %v", p.t.Name()).trace()
    }
    return
}
func (p *builtin) invoke(c Context, o, a []Value) Value { return evoke(c, p, o, a) }
func (p *builtin) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer check_cmp(ctx, p, v, &res) }
    switch t := v.(type) {
    case *builtin:
        if p.t == t.t /* || p.name == a.name */ { res = cmpEqual }
        if res != cmpEqual {
            if p.t == t.t {
                erro(ctx, "%v", ts(v)).trace()
            }
            if p.name == t.name {
                erro(ctx, "%v", ts(v)).trace()
            }
        }
    case *list:
        if t.len() == 1 { res = p.cmp(ctx, t.elems[0]) }
    }
    return
}

type get_rule struct{}
type is_rule struct{ x *regexp.Regexp }

type rule_ctx struct{ Context ; rule *rule ; args []Value }
func (p *rule_ctx) Position() Position { return p.rule.Position() }
func (p *rule_ctx) cast(t reflect.Type) Context { return icast(p, t) }
func (p *rule_ctx) inner() Context { return p.Context }
func (p *rule_ctx) ts(t string) (s string) {
    s = "{="+t+" "+p.rule.String()
    if p.args != nil {
        s += "("
        for i, a := range p.args {
            if 0 < i { s += "," }
            s += a.String()
        }
        s += ")"
    }
    s += " "+ts(p.Context)+"}"
    return
}
func (p *rule_ctx) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case  get_rule: return p.rule
    case  get_args: if p.args != nil { return p.args }
    case init_args: if p.args != nil { t.args(ctx, p.args); return }
    case is_rule:
        if v := t.x.MatchString(p.rule.target.String()); v {
            if false { note(ctx, "%v %v", t.x, p.rule.target).debug() }
            return true
        }
    }
    return p.Context.do(ctx, op)
}

func _entry(ctx Context) entry { return try[entry](ctx, get_rule{}) }

type entry interface {
    destiny() Value // aka target
    programs(...*program) []*program
    executer
    object
}

func hasRecipes(e entry) (_ bool) {
    for _, p := range e.programs() {
        if 0 < len(p.recipes) { return true }
    }
    return
}

func execute_entry(ctx Context, e entry, args ...Value) ([]Value, bool) {
    return e.execute(ctx, args...), true
}

const (
    traverse_noop uint = iota
    traverse_case
    traverse_done
    traverse_next
)
type traverse_state struct{ p any ; uint }
func (t traverse_state) String() (_ string) {
    switch t.uint {
    case traverse_noop: return "noop"
    case traverse_case: return "case"
    case traverse_done: return "done"
    case traverse_next: return "next"
    }
    return fmt.Sprintf("%v", t.uint)
}

// rule represents a declared rule entry.
type rule struct{
    target Value
    program []*program
    arged []Value
}
func (_ *rule) kind() Kind { return KindObject|KindRule }
func (p *rule) hash(c Context) uint64 { return fnv1(c, p, p.target) }
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
        erro(ctx, "nil entry").trace()
    } else if p.target == nil {
        erro(ctx, "entry target is nil").trace()
    } else {
        name = p.target.string(ctx)
    }
    return
}
func (p *rule) true(ctx Context) bool { return p.target.true(ctx) }
func (p *rule) float(Context) (_ float64) { return }
func (p *rule) int(Context) (_ int64) { return }
func (p *rule) string(ctx Context) string { return p.target.string(ctx) }
func (p *rule) String() string {
    if p.target == nil { return "<nil entry>" }
    return p.target.String()
}
func (p *rule) ts(ctx Context, t string) string {
    return "{=" + t + " " + ts(p.target,ctx) + "}"
}
func (p *rule) updated(ctx Context) (res bool) {
    if res = p.target.updated(ctx); res {
        do(ctx, mark_dirty{[]Value{ p.target }})
    }
    return
}
func (p *rule) updatedDeps(ctx Context, v ...Value) []Value {
    return p.target.updatedDeps(ctx, v...)
}
func (p *rule) execute(ctx Context, a ...Value) (res []Value) {
    if p.patterned(ctx) {
        erro(ctx, "execute pattern entry: %v", p.target).trace()
    }

    ctx = &rule_ctx{ctx, p, a}

    for _, prog := range p.program {
        if v := prog.execute(ctx); v != nil {
            res = append(res, v)
        }
    }
    return
}
func (p *rule) recipes() (res []Value) {
    for _, prog := range p.program {
        for _, recipe := range prog.recipes {
            res = append(res, recipe)
        }
    }
    return
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
func (p *rule) expand(ctx Context) (_ Value) {
    var target = p.target.expand(ctx)
    if equal(ctx, target, p.target) { return p }
    return &rule{ target, p.program, p.arged }
}
func (p *rule) evoke(ctx *evocation) Value {
    ctx.a = expand(ctx, ctx.a...) // to save the changed args
    return ease(ctx, p.execute(ctx, ctx.a...))
}
func (p *rule) delete(  ctx Context) []*file { return p.target.delete(ctx) }
func (p *rule) stamp(   ctx Context) []*file { return p.target.stamp(ctx) }
func (p *rule) traverse(ctx Context) {
    var target = auto_get(ctx, "@")

    if target == nil {
        erro(ctx, "$@ is not defined").trace()
    }

    if _entry(ctx) == p {
        var proj = _project(ctx)

        if c := cast[*term](ctx); c != nil {
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
        ctx = &rule_ctx{ctx, p, nil}
    }

progloop:
    for _, prog := range p.program {
        switch func () (u uint) {
            defer func() {
                if e := recover(); e != nil {
                    switch t := e.(type) {
                    case traverse_state: u = t.uint
                    case test_fail: t.i += 1; panic(t)
                    default: panic(e)
                    }
                }
            } ()
            if v := prog.execute(ctx); v != nil { do(ctx, exe_res{v}) }
            return
        } () {
        case traverse_done: break progloop
        case traverse_next: continue progloop
        }
    }
    return
}
func (p *rule) hit(ctx Context, c *valcache) (*valcache, bool) {
    return c.hit(ctx, p.target)
}
// FIXME: p.target maybe not the real target
func (p *rule) stat(ctx Context) *statinfo { return p.target.stat(ctx) }
func (p *rule) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer check_cmp(ctx, p, v, &res) }
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

func _stemmed(ctx Context) *stemmed_ctx { return cast[*stemmed_ctx](ctx) }
func _stems(ctx Context) (res []string) {
    res, _ = do(ctx, get_stems{}).([]string)
    return
}

type get_stems struct{}

type stemmed_ctx struct{ Context ; *stemmed_rule }
func (p *stemmed_ctx) hash(ctx Context) uint64 { return fnv1(ctx, p, p.target) }
func (p *stemmed_ctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *stemmed_ctx) inner() Context { return p.Context }
func (p *stemmed_ctx) ts(_t string) string {
    s, t := p.target.String(), p.rule.target.String()
    return "{="+_t+" "+s+" "+t+" "+ts(p.Context)+"}"
}
func (p *stemmed_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case get_stems: return p.stems
    }
    return p.Context.do(ctx, op)
}

type stemmed_rule struct{
    *rule
    target Value
    stems []string
}
func (p *stemmed_rule) kind() Kind { return p.rule.kind()|KindStemmedRule }
func (p *stemmed_rule) destiny() Value { return p.target/* versus p.rule.target */ }
func (p *stemmed_rule) String() (s string) {
    for i, stem := range p.stems { if i > 0 { s += "," }; s += stem }
    return fmt.Sprintf("%s:%s", p.target, s) // "<%s:%s>"
}
func (p *stemmed_rule) expand(ctx Context) (res Value) {
    if v := p.rule.expand(ctx); v != p.rule {
        res = &stemmed_rule{v.(*rule), p.target, p.stems}
    } else {
        res = p
    }
    return
}
func (p *stemmed_rule) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer check_cmp(ctx, p, v, &res) }
    if x, y := v.(*stemmed_rule); y {
        if len(p.stems) != len(p.stems) { return }
        for i, stem := range p.stems {
            if stem != x.stems[i] { return }
        }
        return p.rule.cmp(ctx, x.rule)
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}
func (p *stemmed_rule) traverse(ctx Context) {
    // NOTE: Make a clone of the underlying rule for traversing the real target;
    //       the underlying rule target is readonly, it must not be changed, for
    //       next traversal be done correctly.
    var t = *p.rule // TODO: consider not copying the rule, use pointer instead
    t.target = p.target
    t.traverse(&stemmed_ctx{ctx, p})
}
