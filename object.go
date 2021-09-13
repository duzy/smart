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
        "bytes"
        "sync"
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
        Get(ctx Context, name string) (Value, error)

        // redecl the object.
        redecl(ctx Context, scope *Scope)
}

type objbase struct { // generally unnamed objects
        valbase
        scope *Scope
        owner *Project
}
func (p *objbase) DeclScope() *Scope { return p.scope }
func (p *objbase) OwnerProject() *Project { return p.owner }
func (p *objbase) String() string { return fmt.Sprintf("{unknown %p}", p) }
func (p *objbase) Strval(ctx Context) (string, error) { return fmt.Sprintf("{unknown %p}", p), nil }
func (p *objbase) Name() string { panic("inquiring name of an unknown object") }
func (p *objbase) Get(_ Context, name string) (Value, error) { return nil, fmt.Errorf("no such property `%s`", name) }
func (p *objbase) redecl(_ Context, scope *Scope) { panic("redeclaring unknown object") }
func (p *objbase) exists() existence { return existenceMatterless }
func (p *objbase) cmp(_ Context, v Value) (res cmpres) {
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
func (p *knownobject) String() string { return fmt.Sprintf("{object %s}", p.name) }
func (p *knownobject) Strval(_ Context) (string, error) { return fmt.Sprintf("{object %s}", p.name), nil }
func (p *knownobject) True(_ Context) (bool, error) { return true, nil }
func (p *knownobject) Name() string { return p.name }
func (p *knownobject) redecl(_ Context, scope *Scope) {
        if p.scope != scope {
                if p.scope != nil {
                        delete(p.scope.elems, p.name)
                }
                if p.scope = scope; p.scope != nil {
                        p.scope.elems[p.name] = p
                }
        }
}
func (p *knownobject) cmp(_ Context, v Value) (res cmpres) {
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
func (p *unresolvedobject) Name() string {
        if p.name == nil { panic("unresolved object name is nil") }
        return p.name.String()
}
func (p *unresolvedobject) String() string { return p.name.String() }
func (p *unresolvedobject) Strval(_ Context) (string, error) {
        // The string value of a unresolved object is "", so that a
        // unresolved &(var) is stringed to ""
        return /*p.name.Strval()*/"", nil
}
func (p *unresolvedobject) True(_ Context) (bool, error) { return false, nil }
func (p *unresolvedobject) Call(ctx Context, a... Value) (result Value) { result = p; return }
func (p *unresolvedobject) Execute(ctx Context, a... Value) (result []Value, err error) { return []Value{p}, nil }
func (p *unresolvedobject) redecl(ctx Context, scope *Scope) {
        if p.scope != scope {
                var name, err = p.name.Strval(ctx)
                if err != nil { panic(fmt.Sprintf("unresolved name error: %v", p.name, err)) }
                if p.scope != nil { delete(p.scope.elems, name) }
                if p.scope = scope; p.scope != nil {
                        p.scope.elems[name] = p
                }
        }
}
func (p *unresolvedobject) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*unresolvedobject); ok {
                assert(ok, "value is not unresolvedobject")
                if p.owner == a.owner && p.scope == a.scope {
                        res = p.name.cmp(ctx, a.name)
                }
        }
        return
}
func (p *unresolvedobject) traverse(t *traversal) (brks breakers) { return }

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
func (p *ProjectName) NamedProject() *Project { return p.project }
func (p *ProjectName) String() string { return p.name }
func (p *ProjectName) Strval(_ Context) (string, error) { return p.name, nil }
func (p *ProjectName) True(_ Context) (bool, error) { return p.project != nil, nil }
func (p *ProjectName) Get(ctx Context, name string) (value Value, err error) {
        if p.project != nil { value, err = p.project.resolveObject(ctx, name) }
        return
}

// Call a ProjectName returns the project name.
func (p *ProjectName) Call(ctx Context, a... Value) (value Value) {
        if p.project != nil {
                value = MakeString(ctx.Position(), p.project.name)
        }
        return
}
func (p *ProjectName) traverse(t *traversal) (brks breakers) {
        if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
        if entry := p.project.DefaultEntry(); entry == nil {
                // does nothing
        } else if entry.class != UseRuleEntry {
                brks = entry.traverse(t)
        }
        return
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
func (p *ProjectName) cmp(ctx Context, v Value) (res cmpres) {
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
func (n *ScopeName) NamedScope() *Scope { return n.scope }
func (n *ScopeName) String() string  { return fmt.Sprintf("{scope %s}", n.name) }
func (n *ScopeName) Strval(ctx Context) (string, error) { return fmt.Sprintf("scope %s", n.name), nil }
func (n *ScopeName) True(ctx Context) (bool, error) { return n.scope != nil, nil }
func (n *ScopeName) Get(_ Context, name string) (Value, error) {
        if sym := n.scope.Resolve(name); sym != nil {
                value, _ := sym.(Value)
                return value, nil
        }
        return nil, fmt.Errorf("Undefined `%s' in scope `%s'.", name, n.Name())
}
func (p *ScopeName) cmp(ctx Context, v Value) (res cmpres) {
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
        case DefVoid:    s = "Void"
        case DefDefault: s = "Default"
        case DefExpand1: s = "Expand1"
        case DefExpand2: s = "Expand2"
        case DefExecute: s = "Execute"
        case DefArg:     s = "Arg"
        case DefAuto:    s = "Auto"
        case DefConfDir: s = "ConfDir"
        case DefConfRef: s = "ConfRef"
        case DefConfig:  s = "Config"
        case DefDecl:    s = "Decl"
        case defany:     s = "any"
        default: s = fmt.Sprintf("Origin<%d>", o)
        }
        return
}

type scopedContext struct { // TODO: replace cloctx with this
        Context
        scope *Scope
}

type callContextDefs map[string]struct{
        position Position
        value Value
}
type callContext struct {
        Context
        defs callContextDefs
}

func (cc *callContext) Get(name string) (res Value, okay bool) {
        if def, ok := cc.defs[name]; ok {
                res, okay = def.value, true
        } else if false && cc.Context != nil {
               res, okay = cc.Context.Get(name)
        }
        if false && name == "@" {
                for s, d := range cc.defs {
                        diag.infoAt(d.position, "%s %v", s, d.value)
                }
                diag.infoAt(cc.Position(), "%v %v %v", name, res, okay).debug(16)
        }
        return
}

func (cc *callContext) Set(name string, val Value) (res Value, okay bool) {
        var def, ok = cc.defs[name]
        if ok { res = def.value }
        def.position = cc.Context.Position()
        def.value = val
        cc.defs[name] = def
        okay = true
        return
}

func (cc *callContext) setArgs(params []*Def, args []Value) (names []string, err error) {
        var (
                argnum int // setup named/number parameters ($1, $2, etc.)
                name string
        )
        for _, a := range args {
                //<!IMPORTANT: Don't translate Flag, Flag values are valid
                //         regular arguments. Pair values are special.
                if l, ok := a.(*List); ok && l.Len() == 1 { a = l.Elems[0] }
                if p, ok := a.(*Pair); ok {
                        if name, err = p.Key.Strval(cc); err != nil {
                                diag.errorOf(p.Key, "strval '%v' failed: %v", p.Key, err).debug(1)
                                return
                        } else {
                                a = p.Value
                        }
                } else if argnum < len(params) {
                        name = params[argnum].name
                } else {
                        name = strconv.Itoa(argnum+1)
                }
                argnum += 1
                if _, okay := cc.Set(name, a); !okay {
                        diag.errorOf(a, "arg '%s': %v", name, err).debug(1)
                        return
                } else {
                        names = append(names, name)
                }
        }
        return
}

// A Def represents a definition, it's a Caller but mustn't be a Valuer.
type Def struct {
        knownobject
        mutex sync.Mutex
        origin Origin
        value Value
}
func (d *Def) String() (s string) {
        switch s = d.name; d.origin {
        case DefDefault: s +=   "="
        case DefExpand1: s +=  ":="
        case DefExpand2: s += "::="
        case DefExecute: s +=  "!="
        default:         s +=   "⇒"
        }
        var value Value
        d.mutex.Lock()
        value = d.value
        d.mutex.Unlock()
        if value != nil {
                s += elementString(nil, d, value, 0)
        } else {
                s += "<nil>"
        }
        return
}
func (d *Def) Strval(ctx Context) (res string, err error) {
        if d.origin == DefArg || d.origin == DefAuto {
                if val, _ := ctx.Get(d.name); !isNil(val) {
                        res, err = val.Strval(ctx)
                }
        } else {
                var value Value
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
                if !isNil(value) { res, err = value.Strval(ctx) }
        }
        return
}
func (d *Def) True(ctx Context) (res bool, err error) {
        if d.origin == DefArg || d.origin == DefAuto {
                if val, okay := ctx.Get(d.name); okay && !isNil(val) {
                        res, err = val.True(ctx)
                }
        } else {
                var value Value
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
                if !isNil(value) { res, err = value.True(ctx) }
        }
        return
}
func (d *Def) refs(ctx Context, v Value) (res bool) {
        if d.origin == DefArg || d.origin == DefAuto {
                if val, _ := ctx.Get(d.name); !isNil(val) {
                        res = val.refs(ctx, v)
                }
        } else if res = d == v; !res {
                var value Value
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
                if !(isNil(value) || isNone(value)) {
                        res = value.refs(ctx, v)
                }
        }
        return
}
func (d *Def) defs(ctx Context, s ...string) (res []*Def) {
        if d.origin == DefArg || d.origin == DefAuto {
                if val, _ := ctx.Get(d.name); !isNil(val) {
                        res = val.defs(ctx, s...)
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
        d.mutex.Lock()
        value = d.value
        d.mutex.Unlock()
        if true && !isNil(value) {
                res = append(res, value.defs(ctx, s...)...)
        }
        return
}
func (d *Def) expandible(ctx Context, w expandwhat) (res bool) {
        if d.origin == DefArg || d.origin == DefAuto {
                // res = true // expand to DefAutoVal
                if val, _ := ctx.Get(d.name); !(isNil(val) || isNone(val)) {
                        res = val.expandible(ctx, w)
                }
        } else {
                var value Value
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
                res = value.expandible(ctx, w)
        }
        return
}
func (d *Def) expand(ctx Context, w expandwhat) (res Value, err error) {
        var (
                origin = d.origin
                value0, value1, value2 Value
        )
        if origin == DefArg || origin == DefAuto {
                value0, _ = ctx.Get(d.name)
        } else {
                d.mutex.Lock()
                value0 = d.value
                d.mutex.Unlock()
        }

        if isNil(value0) || isNone(value0) {
                return // does nothing
        } else if value1, err = value0.expand(ctx, w); err != nil {
                diag.errorOf(value0, "expand '%v' failed: %v", value0, err).debug(1)
                return
        } else if w&expandDef == 0 {
                if !isNil(value1) && value1 != value0 {
                        res = &Def{ knownobject: d.knownobject, origin: origin, value: value1 }
                }
        } else if isNil(value1) {
                res = value0
        } else if value2, err = value1.expand(ctx, w); err != nil {
                diag.errorOf(value0, "expand '%v' failed: %v", value1, err).debug(1)
        } else if isNil(value2) {
                res = value1
        } else {
                res = value2
        }
        return
}
func (d *Def) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*Def); ok && !isNil(d.value) {
                var val1, val2 Value
                if d.origin == DefArg || d.origin == DefAuto {
                        val1, _ = ctx.Get(d.name)
                } else {
                        d.mutex.Lock()
                        val1 = d.value
                        d.mutex.Unlock()
                }
                if a.origin == DefArg || a.origin == DefAuto {
                        val2, _ = ctx.Get(a.name)
                } else {
                        a.mutex.Lock()
                        val2 = a.value
                        a.mutex.Unlock()
                }
                if isNil(val1) {
                        if isNil(val2) { res = cmpEqual }
                } else if !isNil(val2) {
                        res = val1.cmp(ctx, val2)
                }
        }
        return
}
func (d *Def) elemstr(_ Context, o Object, k elemkind) (s string) {
        if o != nil {
                if p := d.OwnerProject(); p != o.OwnerProject() {
                        return fmt.Sprintf("$(%s->%s)", p.name, d.name)
                }
        }
        s = fmt.Sprintf(`$(%s)`, d.name)
        return
}
func (d *Def) isEmpty(ctx Context) bool {
        var val Value
        if d.origin == DefArg || d.origin == DefAuto {
                val, _ = ctx.Get(d.name)
        } else {
                d.mutex.Lock()
                val = d.value
                d.mutex.Unlock()
        }
        return isNone(val) || isNil(val)
}
func (d *Def) val(ctx Context, value Value) (err error) { return d.set(ctx, d.origin, value) }
func (d *Def) set(ctx Context, origin Origin, value Value) (err error) {
        if d.origin == DefArg || d.origin == DefAuto {
                var pos = ctx.Position()
                if !pos.IsValid() { pos = d.position }
                if _, ok := ctx.Set(d.name, value); !ok {
                        diag.warnAt(pos, "setting auto '%s' failed (value=%v)", d.name, value).debug(6)
                }
                return
        } else if origin != DefExpand1 && !isNil(value) && value.refs(ctx, d) {
                var pos = d.position
                var val Value
                d.mutex.Lock()
                val = d.value
                d.mutex.Unlock()
                if !pos.IsValid() && val != nil { pos = val.Position() }
                diag.errorAt(pos, "value refers to assigning Def '%s': %v (%T)", d.name, value, value).debug(1)
                if options.verbose { diag.prompt("set %s (%v): %v\n", origin, d.name, value) }
                if options.debug { diag.infoAt(pos, "from here").debug(1) }
                return
        } else if origin != DefExecute && isNil(value) {
                value = MakeNone(d.position)
        }

        var elems []Value
        switch d.origin = origin; d.origin {
        case DefExpand1: // expands delegates
                if elems, _, err = expandall2(ctx, expandDelegate, value); err != nil {
                        diag.errorOf(value, "%v: expand value '%v' failed: %v", d.origin, value, err)
                        return
                } else {
                        var val = MakeListOrScalar(value.Position(), elems)
                        d.mutex.Lock()
                        d.value = val
                        d.mutex.Unlock()
                }
        case DefExpand2: // expands delegates and closures
                if elems, _, err = expandall2(ctx, expandPlainValue/*|expandArgs*/, value); err != nil {
                        diag.errorOf(value, "%v: expand value '%v' failed: %v", d.origin, value, err)
                        return
                } else {
                        var val = MakeListOrScalar(value.Position(), elems)
                        d.mutex.Lock()
                        d.value = val
                        d.mutex.Unlock()
                }
                /*
        case DefExecute:
                if err = d.execute(); err != nil {
                        diag.errorOf(value, "%v: stringify value '%v' failed: %v", d.origin, value, err).
                                debug(1)
                        return
                }*/
        default: // DefVoid, DefDefault, DefArg, etc.
                d.mutex.Lock()
                d.value = value
                d.mutex.Unlock()
        }
        return
}

func (d *Def) append(ctx Context, va... Value) (err error) {
        var (
                pos = ctx.Position()
                list *List
                value Value
        )
        if !pos.IsValid() { pos = d.position }

        for _, value := range va {
                if !isNil(value) && value.refs(ctx, d) {
                        err = fmt.Errorf("%v: append recursive variable '%s'", d.owner, d.name)
                        diag.infoAt(pos, "%v", err).debug(6)
                        return
                }
        }

        if d.origin == DefArg || d.origin == DefAuto {
                value, _ = ctx.Get(d.name)
        } else {
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
        }

        if num := len(va); num == 0 {
                return // Does nothing...
        } else if isNil(value) || isNone(value) {
                list = MakeList(pos, va...)
        } else if list, _ = value.(*List); list != nil {
                list.Append(va...)
        } else {
                list = MakeList(pos, append(merge(value), va...)...)
        }

        if true { assert(list != nil, "nil values evaluated") }
        return d.val(ctx, list)
}

func (d *Def) callVal(ctx Context, a... Value) (res Value) {
        var (
                pos Position = ctx.Position()
                cc = callContext{ ctx, make(callContextDefs) }
                w = expandClosure|expandDelegate
                value Value
                err error
        )
        if d.origin == DefArg || d.origin == DefAuto {
                value, _ = ctx.Get(d.name)
        }
        if isNil(value) {
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
        }
        if isNil(value) || !value.expandible(ctx, w) {
                res = value
                return
        }

        cc.setArgs(nil, a)

        if res, err = value.expand(&cc, w); err != nil {
                diag.errorAt(pos, "expand def value failed: %v", err).debug(1)
        } else if isNil(res) && !isNil(value) {
                if value.expandible(&cc, w) {
                        diag.warnAt(pos, "expand '%v' incomplete (value=%v (%T))", d.name, value, value).debug(1)
                } else { res = value }
        }
        return
}

func (d *Def) execute(ctx Context, a... Value) (res Value) {
        var (
                pos = ctx.Position()
                origin Origin
                value Value
                cmd string
                err error
        )
        d.mutex.Lock()
        origin, value = d.origin, d.value
        if !pos.IsValid() { pos = d.position }
        d.mutex.Unlock()
        if origin != DefExecute {
                diag.errorAt(pos, "%v: non-execute def: %v", origin, value).debug(1)
        } else if isNil(value) || isNone(value) {
                // does nothing
        } else if cmd, err = value.Strval(ctx); err != nil {
                diag.errorAt(pos, "%v: strval '%v' failed: %v", origin, value, err).debug(1)
                res = MakeNone(pos)
        } else if cmd == "" {
                diag.warnAt(pos, "%v: empty command (value=%v)", origin, value).debug(1)
                res = MakeNone(pos)
        } else {
                // TODO: possibility to run command in the specified container
                var stdout, stderr bytes.Buffer
                var sh = exec.Command("sh", "-c", cmd)
                sh.Stdout, sh.Stderr = &stdout, &stderr
                if err = sh.Run(); err != nil {
                        diag.errorAt(pos, "%v: execute command failed: %v", origin, err).debug(1)
                        res = MakeNone(pos)
                } else {
                        res = MakeString(pos, strings.TrimSpace(stdout.String()))
                }
                stdout.Reset()
                stderr.Reset()
                origin = DefExecuted
        }
        d.mutex.Lock()
        d.origin, d.value = origin, res
        d.mutex.Unlock()
        return
}

func (d *Def) Call(ctx Context, a... Value) (res Value) {
        switch d.origin {
        case DefAuto, DefArg, DefDefault: res = d.callVal(ctx, a...)
        case DefExecute: res = d.execute(ctx, a...)
        case DefExpand1:
                if isNil(d.value) {
                        // does nothing
                } else if d.value.expandible(ctx, expandClosure) {
                        res = d.callVal(ctx, a...)
                } else {
                        res = d.value
                }
        default: res = d.value // DefExpand2, DefExecuted, etc.
        }
        if isNil(res) {
                // does nothing
        } else if list, ok := res.(*List); !ok {
                // does nothing
        } else if n := len(list.Elems); n == 0 {
                var pos = ctx.Position()
                if !pos.IsValid() { pos = d.position }
                res = MakeNone(pos)
        } else if n == 1 {
                res = list.Elems[0]
        }
        return
}

func (d *Def) DiscloseValue(ctx Context) (res Value, err error) {
        var (
                pos = ctx.Position()
                value Value
        )
        d.mutex.Lock()
        if value = d.value; !pos.IsValid() { pos = d.position }
        d.mutex.Unlock()
        if isNil(value) {
                // does nothing
        } else if res, err = value.expand(ctx, expandClosure); err != nil {
                diag.errorAt(pos, "expand '%v' failed: %v", value, err).debug(1)
        } else if isNil(res) {
                res = value
        }
        return
}

func (d *Def) Get(ctx Context, name string) (res Value, err error) {
        switch name {
        case "name" : res = MakeString(d.position, d.name)
        case "value":
                if d.origin == DefArg || d.origin == DefAuto {
                        res, _ = ctx.Get(d.name)
                } else {
                        d.mutex.Lock()
                        res = d.value
                        d.mutex.Unlock()
                }
        default:
                err = fmt.Errorf("no such property `%s' (Def)", name)
                diag.errorAt(d.position, "%v", err).debug(1)
        }
        return
}
func (d *Def) traverse(t *traversal) (brks breakers) {
        var value Value
        if d.origin == DefArg || d.origin == DefAuto {
                value, _ = t.Get(d.name)
        } else {
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
        }
        if value != nil { brks = value.traverse(t) }
        return
}
func (d *Def) stat(t *traversal) (si *statinfo) {
        var value Value
        if d.origin == DefArg || d.origin == DefAuto {
                value, _ = t.Get(d.name)
        } else {
                d.mutex.Lock()
                value = d.value
                d.mutex.Unlock()
        }
        if value != nil { si = value.stat(t) }
        return
}

type undetermined struct {
        tok token.Token
        identifier Value
        value Value
}
func (p *undetermined) Position() Position { return p.identifier.Position() }
func (p *undetermined) String() (s string) {
        s = p.identifier.String()
        s += p.tok.String()
        s += p.value.String()
        return
}
func (p *undetermined) Strval(ctx Context) (string, error) { return p.value.Strval(ctx) }
func (p *undetermined) True(ctx Context) (bool, error) { return false, nil }
func (p *undetermined) Float(ctx Context) (float64, error) { return 0, nil }
func (p *undetermined) Integer(ctx Context) (int64, error) { return 0, nil }
func (p *undetermined) refs(ctx Context, v Value) bool {
        return p.identifier.refs(ctx, v) || p.value.refs(ctx, v)
}
func (p *undetermined) defs(ctx Context, s ...string) (res []*Def) {
        return append(p.identifier.defs(ctx, s...), p.value.defs(ctx, s...)...)
}
func (p *undetermined) expandible(ctx Context, w expandwhat) bool {
        return p.identifier.expandible(ctx, w) || p.value.expandible(ctx, w)
}
func (p *undetermined) expand(ctx Context, w expandwhat) (res Value, err error) {
        var i, v Value
        if i, err = p.identifier.expand(ctx, w); err != nil {
                diag.errorOf(p.identifier, "expand '%v' failed: %v", p.identifier, err).debug(1)
        } else if v, err = p.value.expand(ctx, w); err != nil {
                diag.errorOf(p.value, "expand '%v' failed: %v", p.value, err).debug(1)
        } else if (!isNil(i) && i != p.identifier) || (!isNil(v) && v != p.value) {
                if isNil(i) { i = p.identifier }
                if isNil(v) { v = p.value }
                res = &undetermined{ p.tok, i, v }
        }
        return
}
func (p *undetermined) traverse(t *traversal) (brks breakers) { return }
func (p *undetermined) exists() existence { return existenceMatterless }
func (p *undetermined) stat(t *traversal) (si *statinfo) { return }
func (p *undetermined) stamp(t *traversal) (files []*File, err error) { return }
func (p *undetermined) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*undetermined); ok {
                assert(ok, "value is not undetermined")
                if p.identifier.cmp(ctx, a.identifier) == cmpEqual {
                        if p.value.cmp(ctx, a.value) == cmpEqual {
                                res = cmpEqual
                        }
                }
        }
        return
}
func (p *undetermined) patterned(ctx Context) bool { return false }
func (p *undetermined) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return }
func (p *undetermined) stencil(ctx Context, stems []string) (s string, rest []string) { return }

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
func (p *Builtin) String() string { return fmt.Sprintf("%s", p.name) }
func (p *Builtin) True(_ Context) (bool, error) { return p.f != nil, nil }
func (p *Builtin) Call(ctx Context, a... Value) Value { return p.f(ctx, a...) }
func (p *Builtin) cmp(_ Context, v Value) (res cmpres) {
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
func (entry *RuleEntry) DeclScope() *Scope { return entry.OwnerProject().scope }
func (entry *RuleEntry) OwnerProject() *Project { return entry.programs[0].project }
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
func (entry *RuleEntry) Name() string {
        if entry == nil {
                panic("entry is nil")
        } else if entry.target == nil {
                fmt.Fprintf(stderr, "%v: nil target\n", entry.position)
                panic("entry target is nil")
        }
        return entry.target.String()
}
func (entry *RuleEntry) True(ctx Context) (bool, error) { return entry.target.True(ctx) }
func (entry *RuleEntry) Float(_ Context) (float64, error) { return 0, nil }
func (entry *RuleEntry) Integer(_ Context) (int64, error) { return 0, nil }
func (entry *RuleEntry) String() string { return entry.target.String() }
func (entry *RuleEntry) Strval(ctx Context) (string, error) { return entry.target.Strval(ctx) }
// func (entry *RuleEntry) Class() RuleEntryClass { return entry.class }
// func (entry *RuleEntry) SetClass(class RuleEntryClass) { entry.class = class }
// func (entry *RuleEntry) Programs() []*Program { return entry.programs }
// func (entry *RuleEntry) Depends() (depends []Value) {
//         for _, prog := range entry.programs {
//                 depends = append(depends, prog.depends...)
//         }
//         return
// }
// func (entry *RuleEntry) IsFile() bool {
//         if p, ok := entry.target.(*File); ok && p != nil { return true }
//         if p, ok := entry.target.(*Path); ok && p != nil /*&& p.File != nil*/ {
//                 return true
//         }
//         return false
// }
// func (entry *RuleEntry) SetExplicitFile(file *File) {
//         if file.dir == "" { file.dir = entry.OwnerProject().absPath }
//         if path, ok := entry.target.(*Path); ok && path != nil {
//                 //path.File = file
//         }
//         return
// }
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
func (entry *RuleEntry) Execute(ctx Context, a... Value) (result []Value, brks breakers) {
        switch entry.class {
        case PercRuleEntry, GlobRuleEntry, RegexpRuleEntry, PathPattRuleEntry:
                diag.errorAt(ctx.Position(), "executing pattern entry '%v'", entry.target).debug(1)
                return
        }
        var t = &traversal{
                callContext: callContext{ ctx, make(callContextDefs) },
                project: entry.OwnerProject(),
                execRec: make(map[Value]int),
                start: time.Now(),
        }
        return entry.execute(t, a...)
}
func (entry *RuleEntry) execute(t *traversal, a... Value) (result []Value, brks breakers) {
        for _, program := range entry.programs {
                var res Value
                res, brks = program.execute(t, entry, a); result = append(result, res)
                if tb := brks.of(breakFail, breakErro); tb.has() {
                        brks = brks.not(breakFail, breakErro)
                        for _, brk := range brks {
                                switch brk.what {
                                case breakFail: diag.errorAt(program.position, "execution failed: %v", brk.message).debug(1)
                                case breakErro: diag.errorAt(program.position, "execution error: %v", brk.error).debug(1)
                                default: diag.errorAt(program.position, "breaker: %v", brk.what).debug(1)
                                }
                        }
                } else if tb = brks.of(breakCase, breakDone); tb.has() {
                        brks = brks.not(breakCase, breakDone)
                        for _, brk := range brks {
                                diag.errorAt(program.position, "breaker: %v", brk.what).debug(1)
                        }
                        break
                } else if tb = brks.of(breakNext); tb.has() {
                        brks = brks.not(breakNext)
                        for _, brk := range brks {
                                diag.errorAt(program.position, "breaker: %v", brk.what).debug(1)
                        }
                        continue
                } else if brks.has() {
                        for _, brk := range brks {
                                diag.errorAt(program.position, "unknown breaker: %v", brk.what).debug(1)
                        }
                        break
                }
        }
        return
}
func (entry *RuleEntry) Get(_ Context, name string) (Value, error) {
        switch name {
        case "class": return MakeString(entry.position, entry.class.String()), nil
        case "name": return entry.target, nil //return MakeString(entry.position, entry.Name()), nil
        //case "prerequisites": ...
        }
        return nil, fmt.Errorf("no such entry property (%s)", name)
}
func (entry *RuleEntry) redecl(ctx Context, scope *Scope) {
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
func (entry *RuleEntry) refs(ctx Context, v Value) bool {
        if entry.target.refs(ctx, v) { return true }
        
        // TODO: do more tests for this to see if we need to fallthrough
        return false // only check closured agaist target

        for _, prog := range entry.programs {
                /*for _, m := range prog.pipline {
                        for _, a := range m.args {
                                if a.refs(v) { return true }
                        }
                }*/
                for _, depend := range prog.depends {
                        if depend.refs(ctx, v) { return true }
                }
                for _, recipe := range prog.recipes {
                        if recipe.refs(ctx, v) { return true }
                }
        }
        return false
}
func (entry *RuleEntry) defs(ctx Context, s ...string) (res []*Def) {
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
func (entry *RuleEntry) expandible(ctx Context, w expandwhat) (res bool) {
        if res = entry.target.expandible(ctx, w); res { return }

        // // TODO: do more tests for this to see if we need to fallthrough
        // return // only check closured agaist target

        for _, prog := range entry.programs {
                for _, depend := range prog.depends {
                        if res = depend.expandible(ctx, w); res { return }
                }
                for _, recipe := range prog.recipes {
                        if res = recipe.expandible(ctx, w); res { return }
                }
        }
        return
}
func (entry *RuleEntry) expand(ctx Context, w expandwhat) (res Value, err error) {
        if entry == nil {
                // happens from some &{xxx} exprs
                err = fmt.Errorf("expand nil entry")
                return
        }

        var target Value
        if target, err = entry.target.expand(ctx, w); err != nil {
                diag.errorAt(entry.position, "expand '%v' failed: %v", entry.target, err).debug(1)
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
func (entry *RuleEntry) stamp(t *traversal) (files []*File, err error) {
        return entry.target.stamp(t)
}
func (entry *RuleEntry) traverse(t *traversal) (brks breakers) {
        if optionTraceTraversal   { defer un(tt(t_traverse, t, entry.target)) }
        if optionEnableBenchmarks && false { defer bench(mark("RuleEntry.traverse")) }
        if optionEnableBenchspots { defer bench(spot("RuleEntry.traverse")) }
        var target, _ = t.Get("@")
ForPrograms:
        for _, prog := range entry.programs {
                if brks = brks.not(breakNext); len(brks) > 0 {
                        diag.warnAt(prog.position, "broken traversal %v: %v (stems = %v)",
                                entry, brks[0].what, t.stems).debug(6)
                        return
                } else if _, brks = prog.execute(t, entry, t.arguments); false && brks.has() {
                        diag.warnAt(prog.position, "entry: %v %d, %v, %v, %v",
                                entry, len(entry.programs), t.stems, target, brks[0].what).
                                debug(breakDone > 0, 6)
                }

                // Update traversal breakers
                var prevBrks = brks; brks = nil
                for _, brk := range prevBrks {
                        // NOTE: see traversal.file and traversal.target for further processing
                        switch brk.what {
                        case breakCase, breakDone:
                                // FIXME: t.breakers = append(t.breakers, brk)
                                break ForPrograms // case selected or execution fully done
                        case breakFail, breakErro:
                                brks.append(brk)
                                break ForPrograms
                        case breakNext:
                                brks.append(brk)
                                continue ForPrograms
                        default:
                                diag.warnAt(prog.position, "broken traversal %v: %v", entry, brk.what).debug(6)
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
func (entry *RuleEntry) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*RuleEntry); ok {
                assert(ok, "value is not RuleEntry")
                if /*entry.class == a.class &&*/ entry.target.cmp(ctx, a.target) == cmpEqual {
                        if entry.OwnerProject() == a.OwnerProject() {
                                res = cmpEqual
                        }
                }
        }
        return
}
func (entry *RuleEntry) patterned(ctx Context) bool { return entry.target.patterned(ctx) }
func (entry *RuleEntry) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = entry.target.match(ctx, i)
    return
}
func (entry *RuleEntry) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = entry.target.stencil(ctx, stems)
    return
}

func (entry *RuleEntry) option(ctx Context) (res bool, infos []Value) {
        ForProgram: for _, program := range entry.programs {
                if !program.configure { continue }
                for _, depend := range program.depends {
                        g, ok := depend.(*modifiergroup)
                        if!ok { continue }
                        for _, m := range g.modifiers {
                                s, e := m.name.Strval(ctx)
                                if e != nil || s != "configure" { continue }
                                for _, arg := range m.args {
                                        a, ok := arg.(*Argumented)
                                        if!ok { continue }
                                        f, ok := a.value.(*Flag)
                                        if!ok { continue }
                                        s, e := f.name.Strval(ctx)
                                        if e != nil { continue }
                                        if s != "option" { continue }
                                        for _, v := range a.args {
                                                if p, ok := v.(*Pair); ok {
                                                        s, _ := p.Key.Strval(ctx)
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
func (p *PatternEntry) expandible(ctx Context, w expandwhat) (res bool) {
        if res = p.Pattern.expandible(ctx, w); !res {
                res = p.RuleEntry.expandible(ctx, w)
        }
        return
}
func (p *PatternEntry) expand(ctx Context, w expandwhat) (res Value, err error) {
        var pat, ent Value
        if pat, err = p.Pattern.expand(ctx, w); err != nil {
                diag.errorOf(p.Pattern, "expand '%v' failed: %v", p.Pattern, err).debug(1)
        } else if ent, err = p.RuleEntry.expand(ctx, w); err != nil {
                diag.errorOf(p.RuleEntry, "expand '%v' failed: %v", p.RuleEntry, err).debug(1)
        } else if (!isNil(pat) && pat != p.Pattern) && (!isNil(ent) && ent != p.RuleEntry) {
                res = &PatternEntry{p.Pattern, ent.(*RuleEntry)}
        }
        return
}
func (p *PatternEntry) cmp(ctx Context, v Value) (res cmpres) {
        if a, ok := v.(*PatternEntry); ok {
                assert(ok, "value is not PatternEntry")
                // FIXME: p.Pattern.cmp(p.Pattern)
                if p.RuleEntry.cmp(ctx, a.RuleEntry) == cmpEqual {
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
func (p *stemmed) expand(ctx Context, w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.PatternEntry.expand(ctx, w); err != nil {
                diag.errorOf(p.PatternEntry, "expand '%v' failed: %v", p.PatternEntry, err).debug(1)
                return
        } else if !isNil(v) && v != p.PatternEntry {
                res = &stemmed{v.(*PatternEntry), p.Stems}
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
        }
        return
}
func (p *stemmed) traverse(t *traversal) (brks breakers) {
        diag.errorAt(p.position, "cant traverse stemmed entry directly: %v", p).debug(1)
        brks.add(p.position, breakErro).error = fmt.Errorf("traversing stemmed entry: %v", p)
        return
}
func (p *stemmed) string(t *traversal, targetVal Value, target string) (res breakers) {
        if optionTraceTraversal   { defer un(tt(t_traverse, t, p)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("stemmed.traverse(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("stemmed.traverse")) }

        defer func(a Value, s []string) { p.target, t.stems = a, s } (p.target, t.stems)
        t.stems = p.Stems // set stems for the traversal

        if file := t.project.FindFile(t, target); file != nil {
                file.position = p.position
                p.target = file
        } else {
                p.target = targetVal
        }

        return p.RuleEntry.traverse(t)
}
func (p *stemmed) file(t *traversal, file *File) (res breakers) {
        if optionTraceTraversal { defer un(tt(t_traverse, t, p)) }
        if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("stemmed.file(%v)", p))) }
        if optionEnableBenchspots { defer bench(spot("stemmed.file")) }

        defer func(a Value, s []string) { p.target, t.stems = a, s } (p.target, t.stems)
        t.stems = p.Stems // set stems for the traversal

        if file.info == nil && file.filemap == nil { // !isAbsOrRel()
                if f := t.project.FindFile(t, file.name); f != nil { *file = *f }
                //if file.info == nil { file.info, _ = os.Stat(file.name) }
        }
        file.position = p.position
        p.target = file
        return p.RuleEntry.traverse(t)
}
