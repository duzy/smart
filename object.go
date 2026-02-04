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

type object interface{ Value ; owner() *project }
type objbase struct{ valbase ; scope *scope }
func (_ *objbase) kind() Kind { return KindObject }
func (p *objbase) owner() *project { return p.scope.project }
func (p *objbase) String() string { return fmt.Sprintf("{=obj %p}", p) }
func (p *objbase) exists() existence { return existenceMatterless }
func (p *objbase) declscope() *scope { return p.scope }
func (p *objbase) setscope(name string, s *scope) {
    if p.scope != s {
        if p.scope != nil { delete(p.scope.elems, name) }
        p.scope = s
    }
}

type knownobject struct{ // generally named objects
    objbase
    name string // single, or group name if containing '(*)' and corresponding members
}
func (p *knownobject) kind() Kind { return p.objbase.kind()|KindKnownObject }
func (p *knownobject) String() string { return fmt.Sprintf("{=object %s}", p.name) }
func (p *knownobject) ident(Context) string { return p.name }

type origin uint

const (
    defInvalid origin = 0
    defVoid origin = 1<<(iota-1)
    defConfig  // configure
    defDecl    // declaration names
    defStatic  // auto-expand within a code block at parse time (aka def/end, for/end)
    defParam   // program parameter
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
)

var origin_names = []string{
    "void", "config", "decl", "static", "param",
    "expand_0", "expand_1", "expand_2", "expand_3", "execute",
    "assign_0", "assign_1", "assign_2", "assign_3", "assign_4", "assign_5",
    "test_val", "test_str",
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
            debug(ctx, "set $- to def (%v): %v", x.o, x, trace{})
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
            if a.name = __string(ctx, p.key); a.name == "" {
                erro(pc(ctx,a), "empty name: %v", p.key, trace{})
                return
            }

            if ac.params != nil {
                if _, y = ac.params[a.name]; !y {
                    var keys = reflect.ValueOf(ac.params).MapKeys()
                    errostack(pc(ctx,a), 16, "unknown arg#%d: %v ; known: %v", i, p, keys, trace{})
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
            a.name, a.value = paramName(ctx, argn), scalarize(val)
            if a.name == "" { a.name = a.id }
        }

        if a.id != a.name { args[a.id] = a }
        args[a.name] = a
        argn += 1

        if d, _ := ac.set(ctx, defParam, a.name, a.value); d == nil {
            erro(ac, "arg '%s' not set", a.name, trace{})
            return
        }

        if d, y := ac.defs[a.name]; !y || d == nil {
            erro(ac, "arg '%s' not set", a.name, trace{})
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
        debug(ctx, "%v %v", t.d, t.v)
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
func (a *auto) String() string { return a.name }
func (a *auto) ts(ctx Context, t string) string {
    return "{" + lp(ctx, a.position, t) + " " + a.name + "}"
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
        debug(ctx, "set auto failed: %v: %v %v", a.name, value, app, trace{})
    }
}
func (a *auto) isDigit() bool { return IsDigits(a.name) }
func (a *auto) isPlaceholder() bool { return a.name == "_" }
func (a *auto) stat(ctx Context) (si *statinfo) {
    if val := auto_get(ctx, a.name); val != nil { si = statFile(ctx, val) }
    return
}

type def_evoke struct{ *evocation }
func (c def_evoke) inner() Context { return c.evocation }
func (c def_evoke) cast(t reflect.Type) Context { return icast(c, t) }
func (c def_evoke) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case param_name: return // avoids program execution
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
func (d *def) origin(ctx Context, o origin) (res origin) {
    if d.o == o { return o }

    if checkpoints {
        if d.o != defInvalid && (o == defVoid || o == defInvalid) {
            erro(pc(ctx,d), "%v: %v → %v", d.name, d.o, o, trace{})
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
        errostack(pc(pc(ctx,value),d.value), 1, "duplicated %v %v → %v %v", d.o, d, value, app, trace{})
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
    if cmd = __string(ctx, value); cmd == "" {
        debug(pc(ctx,value), "%v: empty command (value=%v)", d.name, value)
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
        debug(pc(ctx,value), "%v: execute command failed: %v", d.name, e, trace{})
        return
    }

    var pos = value.Position()
    if !pos.IsValid() { pos = _position(ctx) }
    res = _raw(pos, strings.TrimSpace(stdout.String()))
    return
}
func (d *def) stat(ctx Context) (si *statinfo) {
    var value Value
    {
        // d.Lock()
        value = d.value
        // d.Unlock()
    }
    if value != nil { si = statFile(ctx, value) }
    return
}

type undetermined struct{
    token token
    identifier Value
    value Value
}
func (_ *undetermined) kind() Kind { return KindObject|KindUndetermined }
func (p *undetermined) Position() Position { return p.identifier.Position() }
func (p *undetermined) exists() existence { return existenceMatterless }
func (p *undetermined) String() (s string) {
    s  = p.identifier.String()
    s += p.token.String()
    s += p.value.String()
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
func (p *builtin) is_x() bool { return reflect.PointerTo(p.t).Implements(builtin_x_t) }
func (p *builtin) String() string { return p.name }
func (p *builtin) benchmark(ctx *evocation, t time.Time, v reflect.Value) {
	var n = time.Now()
	if d := n.Sub(t); 2*time.Second < d {
		var a = xmerge(_final(ctx), ctx.a...)
		var m = time.Since(n)//; %v %v
		debug(pc(ctx,p), "slow %v: %v, %v (%d → %d args)", p, d, m, len(ctx.a), len(a), callstack{frames:-1})
	} else if f := v.Elem().FieldByName("timing"); f.IsValid() {
		if f.Type().Kind() == reflect.Bool && f.Bool() {
			debug(pc(ctx,p), "%v: %v", p, d, callstack{frames:-1})
		}
	}
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
            if false { debug(ctx, "%v %v", t.x, p.rule.target) }
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
    arged []Value
    program []*program
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
func (p *rule) String() string {
    if p.target == nil { return "<nil entry>" }
    return p.target.String()
}
func (p *rule) ts(ctx Context, t string) string {
    return "{=" + t + " " + ts(p.target,ctx) + "}"
}
func (p *rule) execute(ctx Context, a ...Value) (res []Value) {
    if patterned(ctx, p.target) {
        erro(ctx, "execute pattern entry: %v", p.target, trace{})
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

// FIXME: p.target maybe not the real target

func _stemmed(ctx Context) *stemmed_ctx { return cast[*stemmed_ctx](ctx) }
func _stems(ctx Context) (res []Value) {
    res, _ = do(ctx, get_stems{}).([]Value)
    return
}

type get_stems struct{}

type stemmed_ctx struct{ Context ; *stemmed_rule }
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
    stems []Value
}
func (p *stemmed_rule) kind() Kind { return p.rule.kind()|KindStemmedRule }
func (p *stemmed_rule) destiny() Value { return p.target/* versus p.rule.target */ }
func (p *stemmed_rule) String() (s string) {
    for i, stem := range p.stems { if i > 0 { s += "," }; s += stem.String() }
    return fmt.Sprintf("%s:%s", p.target, s) // "<%s:%s>"
}
