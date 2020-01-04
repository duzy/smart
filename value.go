//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
        "extbit.io/smart/scanner"
        "extbit.io/smart/token"
        "path/filepath"
        "runtime/debug"
        "net/url"
        "reflect"
        "strconv"
        "strings"
        "sync"
        "time"
        "math"
        "fmt"
        "os"
)

const (
        enable_assertions = true
        enable_grep_bench = true
)

type expandwhat int

const (
        expandDelegate expandwhat = 1<<iota // $(...)  ->  ......
        expandClosure // &(...)   ->  $(...)
        expandCaller // foo=...   ->  ...
        expandPath // $(...)/foo  ->  /path/to/foo
        expandAll = expandDelegate | expandClosure | expandCaller | expandPath
)

type cmpres int

const (
        cmpUnknown cmpres = 0
        cmpLess        = -1 // meaningless so far
        cmpGreater     = 1  // meaningless so far
        cmpEqual       = 2
)

// Value represents a value of a type.
type Value interface {
        Positioner // The position where the value appears (or NoPos).

        // Returns true if the value can be evaluated as 'true', 'yes', etc.
        True() bool

        // Lit returns the literal representations of the value.
        String() string

        // Strval returns the string form of the value.
        Strval() (string, error)

        // Integer returns the integer form of the value.
        Integer() (int64, error)

        // Float returns the float form of the value.
        Float() (float64, error)

        // Equality compare.
        cmp(v Value) cmpres

        // Returns true if value modification is after the other.
        after(v Value) (bool, error)

        // Recursively detecting whether this value references
        // the object (to avoid loop-delegation).
        refs(v Value) bool

        closured() bool

        // &(...) -> $(...)
        // $(...) -> ......
        expand(what expandwhat) (Value, error)
}

type closurecontext []*Scope

var cloctx closurecontext

func setclosure(cc closurecontext) (saved closurecontext) {
        saved = cloctx; cloctx = cc; return
}

func scoping(a ...*Project) (saved closurecontext) {
        saved = cloctx
        for _, i := range a {
                cloctx = append(cloctx, i.Scope())
        }
        return
}

func (cc closurecontext) unshift(scopers ...*Scope) closurecontext {
        return append(scopers, cc...)
}

func (cc closurecontext) append(scopers ...*Scope) closurecontext {
        return append(cc, scopers...)
}

func (cc closurecontext) String() (s string) {
        s = "closure{"
        for i, scope := range cc {
                if i > 0 { s += ", " }
                s += scope.comment
        }
        s += "}"
        return
}

type updatedtarget struct {
        target Value
        prerequisites []*updatedtarget
}

func (p *updatedtarget) String() string {
        if len(p.prerequisites) > 0 {
                return fmt.Sprintf("%v→%v", p.target, p.prerequisites)
        }
        return p.target.String()
}

func newUpdatedTarget(target Value, prerequisites []*updatedtarget) *updatedtarget {
        if def, ok := target.(*Def); ok { target = def.Value }
        for _, p := range prerequisites {
                if p.target != nil {
                        if def, ok := p.target.(*Def); ok {
                                p.target = def.Value
                        }
                }
        }
        return &updatedtarget{target, prerequisites}
}

/*
func comptrace(c *comparer, v Value) *comparer {
        t := c.target
        a := fmt.Sprintf("%T{%v}", t, t)
        b := fmt.Sprintf("%T{%v}", v, v)
        c.trace(a, ":", b, "(")
        c.level += 1
        return c
}

func compun(c *comparer) {
        c.level -= 1
        c.trace(")")
}

func (c *comparer) compareStatDepend(d Value, ds string, di os.FileInfo) (err error) {
        var tt, dt time.Time
        if f, ok := d.(*File); ok && f.info != nil {
                d, ds, dt = f, f.FullName(), f.info.ModTime()
        } else if ds == "" {
                err = break_bad(c.program.position, "'%v' unknown depend", d)
                return
        } else if t := context.globe.timestamp(ds); !t.IsZero() {
                dt = t
        } else if di != nil {
                dt = di.ModTime()
        } else {
                //for _, project := range c.program.pc.related {
                if project := mostDerived(); project != nil {
                        if t := project.searchFile(ds); t != nil {
                                d, ds, dt = t, t.FullName(), t.info.ModTime()
                                if f != nil { *f = *t } // replace the file 
                                //break
                        }
                }
        }

        var ts string
        if f, ok := c.target.(*File); ok && f.info != nil {
                ts, tt = f.FullName(), f.info.ModTime()
        } else if ts, err = c.target.Strval(); err != nil {
                return
        } else if ts == "" {
                err = break_bad(c.program.position, "'%v' unknown target", c.target)
                return
        } else if t := context.globe.timestamp(ts); !t.IsZero() {
                tt = t
        } else {
                //for _, project := range c.program.pc.related {
                if project := mostDerived(); project != nil {
                        if t := project.searchFile(ts); t != nil {
                                ts, tt = t.FullName(), t.info.ModTime()
                                if f != nil { *f = *t } // replace the file
                                //break
                        }
                }
        }

        if optionTraceCompare {
                if false {
                        c.trace("compare:", tt, ";", c.target, "("+ts+")")
                        c.trace("compare:", dt, ";", d, "("+ds+")")
                } else if false {
                        c.trace("compare:", tt, dt, ";", c.target, d, ";", ts, ds)
                } else {
                        c.trace("compare:", c.target, d, '\t', '\t', tt, ":", dt, "; ", dt.After(tt))
                }
        } else if optionTraceCompareOutdated && dt.After(tt) {
                c.trace("outdate:", c.target, d, '\t', '\t', tt, ":", dt)
        }

        if tt.IsZero() {
                if optionTraceCompare { c.trace("compare: misstar:", c.target) }
                if false {
                        br := break_bad(c.program.position, "%T '%v' is missing", c.target, c.target)
                        br.misstar = newUpdatedTarget(c.target, nil)
                        err = br
                } else {
                        // Treat all dependency as updated when the target is not existed.
                        c.updated = append(c.updated, newUpdatedTarget(d, nil))
                        if optionTraceCompare || optionTraceCompareOutdated {
                                c.trace("compare: missing", c.target)
                        }
                        if !dt.IsZero() {
                                context.globe.stamp(ds, dt)
                        }
                }
        } else if dt.IsZero() || dt.After(tt) {
                c.updated = append(c.updated, newUpdatedTarget(d, nil))
                if optionTraceCompare { c.trace("compare: updated", d) }

                // Update timestamps to depended file, so that
                // further updates can happen.
                if !dt.IsZero() {
                        // FIXME: set file ModTime instead of using the
                        //        timestamps, it may cause some targets
                        //        updated multiple times if target is
                        //        compared with different deps.
                        context.globe.stamp(ts, dt)
                        context.globe.stamp(ds, dt)
                }
        } else if true {
                // Just save the timestamps to optimize further stats.
                if !tt.IsZero() { context.globe.stamp(ts, tt) }
                if !dt.IsZero() { context.globe.stamp(ds, dt) }
        }
        return
}
*/

type traversecontext struct {
        group *sync.WaitGroup
        calleeErrors []error
        entry *RuleEntry // caller entry (target)
        //visitInsteadUpdate bool // target don't really need to update
        args, arguments []Value // target and argumented prerequisite args
        targets []Value // prerequisite targets ($^ $<)
        modified []*modification // modified prerequisite targets
        updated []*updatedtarget // prerequisites newer than the target (from comparer) ($?)
        derived *Project // the most derived project
        //related []*Project // the related projects in the context
        stems []string // set by StemmedEntry
        traceLevel int
}

// traversal prepares prerequisites of targets.
type traversal struct {
        program *Program
        traversecontext
        print bool // printing work directories (Entering/Leaving)
        targetDef  *Def // $@
        dependsDef *Def // $^
        depend0Def *Def // $<
        orderedDef *Def // $|
        greppedDef *Def // $~
        updatedDef *Def // $?
        modifyBuf  *Def // $-
        stemDef    *Def // $*
        params   []*Def

        preModifiers, postModifiers []*modifier // deprecated
        interpreted []interpreter

        debug bool
}

type prerequisite interface {
        traverse(pc *traversal) error
}

// Usage pattern: defer un(tt(pc, "..."))
func tt(pc *traversal, i Value) tracer {
        // Note that pc.args and pc.arguments are different, they're
        // target execution args and argumented-prerequisite args.
        var a string
        if t := pc.entry.target; len(pc.args) > 0 {
                a = fmt.Sprintf("%T{%s}%s", t, t, pc.args)
        } else {
                a = fmt.Sprintf("%T{%v}", t, t)
        }
        var b = fmt.Sprintf("%T{%v}", i, i)
        pc.trace(a, ":", b, "(")
        pc.level(+1)
        return pc
}

func (pc *traversal) level(n int) { pc.traceLevel += n }
func (pc *traversal) trace(a ...interface{}) {
        printIndentDots(pc.traceLevel, a...)
}

func (pc *traversal) tracef(s string, a ...interface{}) {
        printIndentDots(pc.traceLevel, fmt.Sprintf(s, a...))
}

func (pc *traversal) addNotExistedTarget1(target Value) {
        if pc.debug && false {
                fmt.Fprintf(stderr, "add: %T %v\n", target, target)
                if c, ok := target.(*Compound); ok && len(c.Elems) > 0 {
                        h := c.Elems[0]
                        fmt.Fprintf(stderr, "---: %T %v\n", h, h)
                        debug.PrintStack()
                }
        }
        if target == nil {
                // ...
        } else if _, ok := target.(*None); ok {
                for _, t := range pc.targets {
                        if t == target { return }
                        if t.cmp(target) == cmpEqual { return }
                }
                pc.targets = append(pc.targets, target)
        }
}

func (pc *traversal) addNotExistedTargets(targets ...Value) {
        var valid []Value
        for _, elem := range targets {
                if elem == nil {
                        // ...
                } else if _, ok := elem.(*None); ok {
                        valid = append(valid, elem)
                }
        }
        for _, elem := range merge(valid...) {
                pc.addNotExistedTarget1(elem)
        }
}

func (pc *traversal) traverseAll(value interface{}, nested bool) (err error) {
        if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
                for i := 0; i < v.Len(); i++ {
                        if err = pc.traverse(v.Index(i).Interface()); err == nil {
                                // Good!
                        } else {
                                break
                        }
                }
        } else {
                err = pc.traverse(value)
        }
        return
}

func (pc *traversal) traverse(value interface{}) (err error) {
        var pos = token.Position(pc.entry.position)
        if value == nil {
                err = scanner.Errorf(pos, "updating nil prerequisite")
        } else if p, ok := value.(prerequisite); !ok {
                err = scanner.Errorf(pos, "'%v' is not prerequisite", value)
        } else if p == nil { // this could happen
                err = scanner.Errorf(pos, "updating nil prerequisite")
        } else if err = p.traverse(pc); err != nil {
                err = pc.checkUpdates(err)
        }
        return
}

func (pc *traversal) checkUpdates(src error) (err error) {
        if src != nil {
                var br, ok = src.(*breaker)
                if ok && br.what == breakUpdates {
                        pc.updated = append(pc.updated, br.updated...)
                        for _, updated := range br.updated {
                                pc.updatedDef.append(updated.target)
                        }
                        if len(pc.updated) > 0 {
                                // switch into update mode
                                //pc.mode = updateMode
                        } else {
                                err = pc.checkTargetMode()
                        }
                } else {
                        err = src
                }
        }
        return
}

func (pc *traversal) checkBreakerErrs(errs scanner.Errors, err error) (scanner.Errors, error, bool) {
        var pos = token.Position(pc.program.position)
        if br, done := err.(*breaker); done {
                if n := len(errs); n == 0 {
                        return nil, err, done
                } else if n == 1 {
                        if false {
                                fmt.Fprintf(stderr, "%s: break with error (reason=%d):\n", pos, br.what)
                        }
                } else {
                        if false {
                                fmt.Fprintf(stderr, "%s: break with %d errors (reason=%d):\n", pos, n, br.what)
                        }
                }
                /*for _, e := range errs {
                        fmt.Fprintf(stderr, "%s\n", e.Error())
                }*/
                return nil, err, done
        } else {
                switch e := scanner.WrapErrors(pos, err).(type) {
                case *scanner.Error: errs = append(errs, e)
                case scanner.Errors: errs = append(errs, e...)
                }
                switch err.(type) {
                case fileNotFoundError: // will retry
                case targetNotFoundError: // will retry
                default: done = true
                }
                return errs, err, done
        }
}

func (pc *traversal) updateFile(file *File) (err error) {
        var ( errs scanner.Errors ; done bool )
        if project := mostDerived(); project != nil {
                if _, err = project.updateFile(pc, file); err == nil { return }
                if errs, err, done = pc.checkBreakerErrs(errs, err); done {  }
                if errs != nil && err != nil { err = errs }
        }
        return
}

func (pc *traversal) tarverseTargetErrs(target string) (errs scanner.Errors) {
        if project := mostDerived(); project != nil {
                var done bool
                var err = project.tarverseTarget(pc, target)
                if err == nil { /* Good! */ return }
                if errs, err, done = pc.checkBreakerErrs(errs, err); done {
                        /*break*/
                }
        }
        return
}

func (pc *traversal) tarverseTarget(target string) (err error) {
        if errs := pc.tarverseTargetErrs(target); len(errs) > 0 {
                err = errs
        }
        return
}

func (pc *traversal) tarverseTargetValue(value Value) (err error) {
        var s string
        if s, err = value.Strval(); err == nil {
                 err = pc.tarverseTarget(s)
        }
        return
}

func (pc *traversal) execute(entry *RuleEntry, prog *Program) (err error) {
        // Push the context to the program, so that patterns will work.
        defer func(a []*traversecontext) { prog.callers = a } (prog.callers)
        prog.callers = append([]*traversecontext{&pc.traversecontext}, prog.callers...)

        // Execute the updating program.
        var res Value
        if res, err = prog.Execute(entry, pc.arguments); err != nil {
                //if optionTraceTraversal { pc.tracef("%s: %s", entry, err) }
                if br, ok := err.(*breaker); ok {
                        switch br.what {
                        case breakBad:
                                fmt.Fprintf(stderr, "%s: %v\n", prog.position, err)
                        }
                }
        }

        target, _ := prog.scope.Lookup("@").(*Def).Call(entry.position)
        pc.addNotExistedTargets(target, res)
        return
}

type elemkind int
const (
        elemNoQuote elemkind = 1<<iota
        elemNoBrace
        elemExpand
)

type elemstrer interface {
        elemstr(o Object, k elemkind) string
}

func elementString(o Object, elem Value, k elemkind) (s string) {
        if p, ok := elem.(elemstrer); ok {
                s = p.elemstr(o, k)
        } else if elem != nil {
                s = elem.String()
        }
        return
}

type trivial struct { position Position }
func (_ *trivial) refs(_ Value) (res bool) { return }
func (_ *trivial) closured() (res bool) { return }
func (_ *trivial) expand(_ expandwhat) (v Value, err error) { return }
func (_ *trivial) cmp(_ Value) (res cmpres) { return }
func (_ *trivial) after(_ Value) (after bool, err error) { return }
func (p *trivial) Position() (res Position) { return p.position }
func (_ *trivial) True() (res bool) { return }
func (_ *trivial) Integer() (i int64, err error) { return }
func (_ *trivial) Float() (f float64, err error) { return }
func (_ *trivial) String() (s string) { return }
func (_ *trivial) Strval() (s string, err error) { return }
func (_ *trivial) traverse(pc *traversal) (err error) { return }

type Argumented struct {
        value Value
        args []Value
}
func (p *Argumented) refs(v Value) bool {
        if p.value.refs(v) { return true }
        for _, a := range p.args {
                if a.refs(v) { return true }
        }
        return false
}
func (p *Argumented) closured() bool {
        if p.value.closured() { return true }
        for _, a := range p.args {
                if a.closured() { return true }
        }
        return false
}
func (p *Argumented) expand(w expandwhat) (res Value, err error) {
        var (v Value; args []Value)
        if v, err = p.value.expand(w); err == nil {
                if v != p.value {
                        var num int
                        args, num, err = expandall(w, p.args...)
                        if err == nil && (num > 0 || v != p.value) {
                                res = &Argumented{ v, args }
                        }
                }
        }
        if err == nil && res == nil {
                res = p
        }
        return
}
func (p *Argumented) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Argumented); ok {
                assert(ok, "value is not Argumented")
                if res = p.value.cmp(a.value); res == cmpEqual {
                        // FIXME: check p.args, a.args too
                }
        }
        return
}
func (p *Argumented) after(v Value) (bool, error) { return p.value.after(v) }
func (p *Argumented) Position() Position { return p.value.Position() }
func (p *Argumented) True() (res bool) {
        if p.value != nil {
                res = p.value.True()
        }
        return
}
func (p *Argumented) Integer() (i int64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                i, err = strconv.ParseInt(s, 10, 64)
        }
        return
}
func (p *Argumented) Float() (f float64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                f, err = strconv.ParseFloat(s, 64)
        }
        return
}
func (p *Argumented) elemstr(o Object, k elemkind) (s string) {
        for i, a := range p.args {
                if i > 0 { s += "," }
                s += elementString(o, a, k)
        }
        s = fmt.Sprintf("%s(%s)", elementString(o, p.value, k), s)
        return
}
func (p *Argumented) String() (s string) { return p.elemstr(nil, 0) }
func (p *Argumented) Strval() (s string, err error) {
        if s, err = p.value.Strval(); err != nil {
                return
        }
        s += "("
        for i, a := range p.args {
                if i > 0 {
                        s += ","
                }
                var v string
                if v, err = a.Strval(); err == nil {
                        s += v
                } else {
                        break
                }
        }
        s += ")"
        return
}
func (p *Argumented) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        //<!IMPORTANT! - Don't merge-expand arguments here!
        // Arguments should be passed to Execute as it's
        // represented.
        pc.arguments = p.args
        err = pc.traverse(p.value)
        return
}
func (p *Argumented) checkPatternDepends(pc *traversal, project *Project, se *StemmedEntry, prog *Program) (ok, res1 bool, err error) {
        switch v := p.value.(type) {
        case Pattern:
                ok = true
                res1, err = checkPatternDepend(pc, project, se, prog, v)
        case *Argumented:
                ok, res1, err = v.checkPatternDepends(pc, project, se, prog)
        }
        return
}

type None struct { trivial }
func (p *None) expand(_ expandwhat) (Value, error) { return p, nil }
func (_ *None) cmp(v Value) (res cmpres) { 
        if _, ok := v.(*None); ok { res = cmpEqual }
        return
}

type Nil struct { None }
func (p *Nil) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Nil) cmp(v Value) (res cmpres) {
        if _, ok := v.(*Nil); ok { res = cmpEqual }
        return
}

func isNone(v Value) (t bool) { _, t = v.(*None); return }
func isNil(v Value) (t bool) { _, t = v.(*Nil); return }

// Any is used to box an arbitrary value
type Any struct { value interface{} }
func (p *Any) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Any); ok {
                assert(ok, "value is not Any")
                if v1, ok := p.value.(Value); ok {
                        if v2, ok := a.value.(Value); ok {
                                res = v1.cmp(v2)
                        }
                } else if p.value == a.value {
                        res = cmpEqual
                }
        }
        return
}
func (p *Any) after(v Value) (after bool, err error) {
        if a, ok := p.value.(Value); ok {
                after, err = a.after(v)
        }
        return
}
func (p *Any) expand(w expandwhat) (res Value, err error) {
        if v, ok := p.value.(Value); ok {
                res, err = v.expand(w)
        } else {
                res = p
        }
        return
}
func (p *Any) refs(o Value) (res bool) {
        if v, ok := p.value.(Value); ok {
                res = v.refs(o)
        }
        return
}
func (p *Any) closured() (res bool) {
        if v, ok := p.value.(Value); ok {
                res = v.closured()
        }
        return
}
func (p *Any) Position() (res Position) {
        if v, ok := p.value.(Positioner); ok {
                res = v.Position()
        }
        return
}
func (p *Any) True() (t bool) {
        switch v := p.value.(type) {
        case Value: t = v.True()
        case float32: t = math.Abs(float64(v))-0 >= FloatEpsilon
        case float64: t = math.Abs(v)-0 >= FloatEpsilon
        case int: t = v != 0
        case int64: t = v != 0
        case bool: t = v
        }
        return
}
func (p *Any) Float() (res float64, err error) {
        switch v := p.value.(type) {
        case Value: res, err = v.Float()
        case float32: res = float64(v)
        case float64: res = v
        case int: res = float64(v)
        case int64: res = float64(v)
        case bool: if v { res = FloatEpsilon }
        }
        return
}
func (p *Any) Integer() (res int64, err error) {
        switch v := p.value.(type) {
        case Value: res, err = v.Integer()
        case float32: res = int64(v)
        case float64: res = int64(v)
        case int: res = int64(v)
        case int64: res = v
        case bool: if v { res = 1 }
        }
        return
}
func (p *Any) Strval() (s string, err error) {
        s = fmt.Sprintf("%v", p.value)
        return
}
func (p *Any) String() string { return fmt.Sprintf("<%v>", p.value) }
func (p *Any) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        if p, ok := p.value.(prerequisite); ok {
                err = p.traverse(pc)
        }
        return 
}

// Boxing any value
/*func Box(v interface{}) (any *Any) {
        if any, _ = v.(*Any); any == nil {
                any = &Any{v}
        }
        return
}
func Unbox(any *Any) interface{} { return any.value }*/

type negative struct { trivial; x Value }
func (p *negative) refs(o Value) bool { return p.x.refs(o) }
func (p *negative) closured() bool { return p.x.closured() }
func (p *negative) expand(w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.x.expand(w); err != nil { return }
        if v == p.x { res = p } else { res = &negative{p.trivial,v} }
        return
}
func (p *negative) cmp(v Value) (res cmpres) {
        if a, ok := v.(*negative); ok {
                assert(ok, "value is not negative")
                res = p.x.cmp(a.x)
        }
        return
}
func (p *negative) True() (res bool) {
        if p.x == nil {
                res = true
        } else {
                res = !p.x.True()
        }
        return
}
func (p *negative) elemstr(o Object, k elemkind) string { return `!`+elementString(o, p.x, k) }
func (p *negative) String() (s string) { return p.elemstr(nil, 0) }
func (p *negative) Strval() (string, error) { return fmt.Sprintf("%v", !p.x.True()), nil }
func (p *negative) Float() (res float64, err error) {
        if !p.x.True() { res = FloatEpsilon }
        return
}
func (p *negative) Integer() (res int64, err error) {
        if !p.x.True() { res = 1 }
        return
}
func (p *negative) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        if p, ok := p.x.(prerequisite); ok {
                err = p.traverse(pc)
        }
        return
}

func Negative(val Value) *negative { return &negative{trivial{val.Position()},val} }

type boolean struct {
        trivial
        bool
}
func (p *boolean) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *boolean) True() bool { return p.bool }
func (p *boolean) String() (s string) {
        if p.bool { s = "true" } else { s = "false" }
        return
}
func (p *boolean) Strval() (string, error) { return p.String(), nil }
func (p *boolean) Float() (v float64, err error) {
        if p.bool { v = 1. }
        return
}
func (p *boolean) Integer() (v int64, err error) {
        if p.bool { v = 1 }
        return
}
func (p *boolean) cmp(v Value) (res cmpres) {
        if a, ok := v.(*boolean); ok {
                assert(ok, "value is not boolean")
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpLess
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*answer); ok {
                assert(ok, "value is not answer")
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpLess
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        }
        return
}

type answer struct {
        trivial
        bool
}
func (p *answer) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *answer) True() bool { return p.bool }
func (p *answer) String() (s string) {
        if p.bool { s = "yes" } else { s = "no" }
        return
}
func (p *answer) Strval() (string, error) { return p.String(), nil }
func (p *answer) Float() (v float64, err error) {
        if p.bool { v = 1. }
        return
}
func (p *answer) Integer() (v int64, err error) {
        if p.bool { v = 1 }
        return
}
func (p *answer) cmp(v Value) (res cmpres) {
        if a, ok := v.(*answer); ok {
                assert(ok, "value is not answer")
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpLess
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if a, ok := v.(*boolean); ok {
                assert(ok, "value is not boolean")
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpLess
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        }
        return
}

type integer struct {
        trivial
        int64
}
func (p *integer) True() bool { return p.int64 != 0 }
func (p *integer) Integer() (int64, error) { return p.int64, nil }
func (p *integer) Float() (float64, error) { return float64(p.int64), nil }
func (p *integer) cmp(v Value) (res cmpres) {
        i, e := v.Integer()
        assert(e == nil, "%T: %v", v, e)
        if p.int64 == i {
                res = cmpEqual
        } else if p.int64 < i {
                res = cmpLess
        } else if p.int64 > i {
                res = cmpGreater
        }
        return
}

type Bin struct { integer }
func (p *Bin) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Bin) String() string { return fmt.Sprintf("0b%s", strconv.FormatInt(int64(p.int64),2)) }
func (p *Bin) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),2), nil }

type Oct struct { integer }
func (p *Oct) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Oct) String() string { return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8)) }
func (p *Oct) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),8), nil }

type Int struct { integer }
func (p *Int) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Int) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),10), nil }

type Hex struct { integer }
func (p *Hex) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *Hex) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),16), nil }

const FloatEpsilon = 1e-15 /* 1e-16 */
type Float struct {
        trivial
        float64
} // IEEE-754 64-bit binary floating-point
func (p *Float) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Float) True() bool { return math.Abs(p.float64)-0 > FloatEpsilon }
func (p *Float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) Strval() (string, error) { return strconv.FormatFloat(float64(p.float64),'g', -1, 64), nil }
func (p *Float) Integer() (int64, error) { return int64(p.float64), nil }
func (p *Float) Float() (float64, error) { return p.float64, nil }
func (p *Float) cmp(v Value) (res cmpres) {
        if _, ok := v.(*Float); ok {
                f, e := v.Float()
                assert(e == nil, "%T: %v", v, e)
                if p.float64 == f {
                        res = cmpEqual
                } else if p.float64 < f {
                        res = cmpLess
                } else if p.float64 > f {
                        res = cmpGreater
                }
        }
        return
}

type DateTime struct {
        trivial
        t time.Time
}
func (p *DateTime) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *DateTime) True() bool { return !p.t.IsZero() }
func (p *DateTime) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{DateTime '%s' !(%+v)}", s, e)
        }
}
func (p *DateTime) Strval() (string, error) { return time.Time(p.t).Format("2006-01-02T15:04:05.999999999Z07:00"), nil } // time.RFC3339Nano
func (p *DateTime) Integer() (int64, error) { return p.t.Unix(), nil }
func (p *DateTime) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *DateTime) after(v Value) (after bool, err error) {
        switch t := v.(type) {
        case *DateTime: after = p.t.After(t.t)
        case *Date: after = p.t.After(t.t)
        case *Time: after = p.t.After(t.t)
        } 
        return
}
func (p *DateTime) cmp(v Value) (res cmpres) {
        var vt time.Time
        switch a := v.(type) {
        case *DateTime:
                vt = a.t
        case *Date:
                vt = a.t
        case *Time:
                vt = a.t
        default:
                return
        }
        if p.t.Equal(vt) {
                res = cmpEqual
        } else if p.t.Before(vt) {
                res = cmpLess
        } else /*if p.t.After(vt)*/ {
                res = cmpGreater
        }
        return
}

func ParseDateTime(pos Position, s string) *DateTime {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                return &DateTime{trivial{pos},t}
        } else {
                panic(e)
        }
}

type Date struct { DateTime }
func (p *Date) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Date '%s' !(%+v)}", s, e)
        }
}
func (p *Date) Strval() (string, error) { return time.Time(p.t).Format("2006-01-02"), nil }
func (p *Date) Integer() (int64, error) { return p.t.Unix(), nil }
func (p *Date) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

type Time struct { DateTime }
func (p *Time) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Time '%s' !(%+v)}", s, e)
        }
}
func (p *Time) Strval() (string, error) { return time.Time(p.t).Format("15:04:05.999999999Z07:00"), nil }
func (p *Time) Integer() (int64, error) { return p.t.Unix(), nil }
func (p *Time) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//                └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//                     └<userinfo>─(@)┘      └(:)─<port>┘
type URL struct {
        trivial
        Scheme Value
        Username Value
        Password Value
        Host Value
        Port Value
        Path Value
        Query Value
        Fragment Value
}
func (p *URL) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *URL) True() bool { return p.String() != "" }
func (p *URL) elemstr(o Object, k elemkind) (s string) {
        if s = elementString(o, p.Scheme, k); s == "" { return }
        if s += ":"; p.Host == nil {
                // ...
        } else if _, ok := p.Host.(*None); ok {
                var host string
                if host = elementString(o, p.Host, k); host == "" { return }
                s += "//"
                if p.Username == nil {
                        // ...
                } else if isNone(p.Username) {
                        var user string
                        if user = elementString(o, p.Username, k); user != "" {
                                s += user + "@"
                        }
                }
                s += host
                if p.Port == nil {
                        // ...
                } else if _, ok := p.Port.(*None); ok {
                        var port string
                        if port = elementString(o, p.Port, k); port != "" {
                                s += ":" + port
                        }
                }
        }
        if p.Path == nil {
                // ...
        } else if _, ok := p.Path.(*None); ok {
                var path string
                if path = elementString(o, p.Path, k); path != "" {
                        //if !strings.HasPrefix(path, PathSep) { s += PathSep }
                        s += path
                }
        }
        if p.Query == nil {
                // ...
        } else if _, ok := p.Query.(*None); ok {
                var query string
                if query = elementString(o, p.Query, k); query != "" {
                        s += "?" + query
                }
        }
        if p.Fragment == nil {
                // ...
        } else if _, ok := p.Fragment.(*None); ok {
                var fragment string
                if fragment = elementString(o, p.Fragment, k); fragment != "" {
                        s += "#" + fragment
                }
        }
        return
}
func (p *URL) String() string { return p.elemstr(nil, 0) }
func (p *URL) Strval() (s string, err error) {
        if s, err = p.Scheme.Strval(); err != nil { return }
        if s += ":"; p.Host != nil && !isNone(p.Host) {
                var host string
                if host, err = p.Host.Strval(); err != nil { return }
                s += "//"
                if p.Username != nil && !isNone(p.Username) {
                        var user string
                        if user, err = p.Username.Strval(); err != nil { return }
                        s += user
                        if p.Password != nil {
                                var pass string
                                s += ":"
                                if pass, err = p.Password.Strval(); err != nil { return }
                                s += pass
                        }
                        s += "@"
                }
                s += host
                if p.Port != nil && !isNone(p.Port) {
                        var port string
                        if port, err = p.Port.Strval(); err != nil { return }
                        s += ":" + port
                }
        }
        if p.Path != nil && !isNone(p.Path) {
                var path string
                if path, err = p.Path.Strval(); err != nil { return }
                //if !strings.HasPrefix(path, PathSep) { s += PathSep }
                s += path
        }
        if p.Query != nil && !isNone(p.Query) {
                var query string
                if query, err = p.Query.Strval(); err != nil { return }
                s += "?" + query
        }
        if p.Fragment != nil && !isNone(p.Fragment) {
                var fragment string
                if fragment, err = p.Fragment.Strval(); err != nil { return }
                s += "#" + fragment
        }
        return
}
func (p *URL) Integer() (i int64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                i = int64(len(s))
        }
        return
}
func (p *URL) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *URL) cmp(v Value) (res cmpres) {
        if a, ok := v.(*URL); ok {
                assert(ok, "value is not URL")
                if p.Scheme == nil || a.Scheme == nil { return }
                if p.Scheme.cmp(a.Scheme) != cmpEqual { return }
                if p.Username != nil {
                        if a.Username == nil { return }
                        if p.Username.cmp(a.Username) != cmpEqual { return }
                }
                if p.Password != nil {
                        if a.Password == nil { return }
                        if p.Password.cmp(a.Password) != cmpEqual { return }
                }
                if p.Host != nil {
                        if a.Host == nil { return }
                        if p.Host.cmp(a.Host) != cmpEqual { return }
                }
                if p.Port != nil {
                        if a.Port == nil { return }
                        if p.Port.cmp(a.Port) != cmpEqual { return }
                }
                if p.Path != nil {
                        if a.Path == nil { return }
                        if p.Path.cmp(a.Path) != cmpEqual { return }
                }
                if p.Query != nil {
                        if a.Query == nil { return }
                        if p.Query.cmp(a.Query) != cmpEqual { return }
                }
                if p.Fragment != nil {
                        if a.Fragment == nil { return }
                        if p.Fragment.cmp(a.Fragment) != cmpEqual { return }
                }
                res = cmpEqual
        }
        return
}
func (p *URL) Validate() (res *url.URL) {
        panic(fmt.Sprintf("validate %s", p))
        return
}

type Raw struct {
        trivial
        string
}
func (p *Raw) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Raw) True() bool { return p.string != "" }
func (p *Raw) String() string { return p.string }
func (p *Raw) Strval() (string, error) { return p.string, nil }
func (p *Raw) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Raw) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Raw) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Raw); ok {
                assert(ok, "value is not Raw")
                if p.string == a.string {
                        res = cmpEqual
                }
        }
        return
}

type String struct {
        trivial
        string
}
func (p *String) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *String) True() bool { return p.string != "" }
func (p *String) elemstr(o Object, k elemkind) (s string) {
        if k&elemNoQuote == 0 { s = `'`+p.string+`'` } else { s = p.string }
        return
}
func (p *String) String() string { return p.elemstr(nil, 0) }
func (p *String) Strval() (string, error) { return strings.Replace(p.string, "\\\"", "\"", -1), nil }
func (p *String) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *String) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *String) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        if false { err = pc.tarverseTarget(p.string) }
        return
}
func (p *String) cmp(v Value) (res cmpres) {
        if a, ok := v.(*String); ok {
                assert(ok, "value is not String")
                if p.string == a.string {
                        res = cmpEqual
                } else if p.string < a.string {
                        res = cmpLess
                } else /*if p.string > a.string*/ {
                        res = cmpGreater
                }
        }
        return
}

type Bareword struct {
        trivial
        string
}
func (p *Bareword) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Bareword) True() (t bool) {
        switch p.string {
        case "", "false", "no", "off", "0": t = false
        case "true", "yes", "on", "1": t = true
        default: t = true
        }
        return
}
func (p *Bareword) String() string { return p.string }
func (p *Bareword) Strval() (string, error) { return p.string, nil }
func (p *Bareword) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Bareword) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Bareword) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        err = pc.tarverseTarget(p.string) // TODO: rename to traverseTarget
        return
}
func (p *Bareword) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Bareword); ok {
                assert(ok, "value is not Bareword")
                if p.string == a.string {
                        res = cmpEqual
                } else if p.string > a.string {
                        res = cmpLess
                } else if p.string < a.string {
                        res = cmpGreater
                }
        }
        return
}

type elements struct { Elems []Value }
func (p *elements) Len() int                    { return len(p.Elems) }
func (p *elements) Append(v... Value)           { p.Elems = append(p.Elems, v...) }
func (p *elements) Get(n int) (v Value)         { if n>=0 && n<len(p.Elems) { v = p.Elems[n] }; return }
func (p *elements) Slice(n int) (a []Value) {
        if n>=0 && n<len(p.Elems) {
                a = p.Elems[n:]
        }
        return 
}
func (p *elements) Take(n int) (v Value) {
        if x := len(p.Elems); n>=0 && n<x {
                v = p.Elems[n]
                p.Elems = append(p.Elems[0:n], p.Elems[n+1:]...)
        }
        return 
}
func (p *elements) ToBarecomp() *Barecomp { return &Barecomp{trivial{},*p} }
func (p *elements) ToCompound() *Compound { return &Compound{trivial{},*p} }
func (p *elements) ToList() *List { return &List{*p} }
func (p *elements) True() (t bool) { // (or elems...)
        for _, elem := range p.Elems {
                if elem != nil {
                        if t = elem.True(); t {
                                break
                        }
                }
        }
        return
}
func (p *elements) refs(v Value) bool {
        for _, elem := range p.Elems {
                if elem != nil && (elem == v || elem.refs(v)) {
                        return true
                }
        }
        return false 
}
func (p *elements) closured() bool {
        for _, elem := range p.Elems {
                if elem.closured() { return true }
        }
        return false 
}
func (p *elements) cmpElems(elems []Value) (res cmpres) {
        if len(p.Elems) == len(elems) {
                for i, elem := range p.Elems {
                        if elem == nil { continue }
                        if other := elems[i]; other == nil {
                                continue
                        } else if r := elem.cmp(other); r != cmpEqual {
                                return cmpUnknown
                        }
                }
                res = cmpEqual
        }
        return
}

type Barecomp struct { trivial ; elements }
func (p *Barecomp) refs(v Value) bool { return p.elements.refs(v) }
func (p *Barecomp) closured() bool { return p.elements.closured() }
func (p *Barecomp) Strval() (s string, e error) {
        for _, elem := range p.Elems {
                var v string
                if elem == nil {
                        continue
                } else if v, e = elem.Strval(); e == nil {
                        s += v
                } else {
                        break
                }
        }
        return
}
func (p *Barecomp) elemstr(o Object, k elemkind) (s string) {
        for _, elem := range p.Elems {
                s += elementString(o, elem, k)
        }
        return
}
func (p *Barecomp) True() (t bool) {
        if s, e := p.Strval(); e == nil {
                switch s {
                case "", "false", "no", "off", "0": t = false
                case "true", "yes", "on", "1": t = true
                default: t = true
                }
        }
        return
}
func (p *Barecomp) String() (s string) { return p.elemstr(nil, 0) }
func (p *Barecomp) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Barecomp{p.trivial,elements{elems}}
                } else {
                        res = p
                }
        }
        return
}
func (p *Barecomp) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        err = pc.tarverseTargetValue(p)
        return
}
func (p *Barecomp) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Barecomp); ok {
                assert(ok, "value is not Barecomp")
                res = p.cmpElems(a.Elems)
        }
        return
}

type Barefile struct {
        trivial
        Name Value
        File *File
}
func (p *Barefile) refs(v Value) bool { return p.Name.refs(v) }
func (p *Barefile) closured() bool { return p.Name.closured() }
func (p *Barefile) expand(w expandwhat) (res Value, err error) {
        var name Value
        if name, err = p.Name.expand(w); err == nil {
                if name != p.Name {
                        res = &Barefile{p.trivial,name,p.File}
                } else {
                        res = p
                }
        }
        return
}
func (p *Barefile) True() bool { return p.File != nil }
func (p *Barefile) elemstr(o Object, k elemkind) (s string) { return elementString(o, p.Name, k) }
func (p *Barefile) String() string { return p.elemstr(nil, 0) }
func (p *Barefile) Strval() (string, error) { return p.Name.Strval() }
func (p *Barefile) Integer() (res int64, err error) {
        //var str string
        //if str, err = p.Name.Strval(); err != nil { return }
        if p.File != nil && p.File.exists() {
                res = p.File.info.Size()
        }
        return
}
func (p *Barefile) Float() (float64, error) {
        i, e := p.Integer()
        return float64(i), e
}
func (p *Barefile) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        /*switch pc.mode {
        case compareMode: err = p.traverseCompare(pc)
        case updateMode: err = p.traverseUpdate(pc)
        }*/
        if p.File != nil {
                /*// Fixes the case that '$@.o' is parsed and become '.o'.
                var s string
                if s, err = p.Name.Strval(); err != nil {
                        return
                } else if s != p.File.name {
                        p.File.name = s
                }*/
                err = p.File.traverse(pc)
        } else {
                err = pc.tarverseTargetValue(p)
        }
        return
}
func (p *Barefile) traverseCompare(pc *traversal) (err error) {
        if p.File != nil {
                var after bool
                after, err = pc.targetDef.Value.after(p.File)
                if !after {
                        pc.updated = append(pc.updated, newUpdatedTarget(p, nil))
                }
        } else {
                pc.updated = append(pc.updated, newUpdatedTarget(p, nil))
        }
        return
}
func (p *Barefile) traverseUpdate(pc *traversal) (err error) {
        if p.File != nil {
                var s string
                if s, err = p.Name.Strval(); err != nil {
                        return
                } else if s != p.File.name {
                        // Fixes the case that '$@.o' is parsed and become '.o'.
                        p.File.name = s
                }
                err = p.File.traverse(pc)
        } else {
                err = pc.tarverseTargetValue(p)
        }
        return
}
func (p *Barefile) after(v Value) (after bool, err error) {
        if p.File != nil {
                after, err = p.File.after(v)
        }
        return
}
func (p *Barefile) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Barefile); ok {
                assert(ok, "value is not Barefile")
                // FIXME: check p.File.filebase == a.File.filebase first
                res = p.Name.cmp(a.Name)
        }
        return
}

type GlobMeta struct {
        trivial
        token.Token
}
func (p *GlobMeta) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *GlobMeta) String() string { return p.Token.String() }
func (p *GlobMeta) Strval() (string, error) { return p.Token.String(), nil }
func (p *GlobMeta) cmp(v Value) (res cmpres) {
        if a, ok := v.(*GlobMeta); ok {
                assert(ok, "value is not GlobMeta")
                if p.Token == a.Token {
                        res = cmpEqual
                }
        }
        return
}

// `[a-b]`, `[abc]`, ...
// `a-b`, `abc`, `a$(var)c`, `a$(spaces)c`...
type GlobRange struct { trivial ; Chars Value }
func (p *GlobRange) refs(v Value) bool { return p.Chars.refs(v) }
func (p *GlobRange) closured() bool { return p.Chars.closured() }
func (p *GlobRange) expand(w expandwhat) (Value, error) {
        if v, err := p.Chars.expand(w); err != nil {
                return nil, err
        } else if v != p.Chars {
                return &GlobRange{p.trivial,v}, nil
        } else {
                return p, nil
        }
}
func (p *GlobRange) elemstr(o Object, k elemkind) (s string) {
        return fmt.Sprintf("[%s]", elementString(o, p.Chars, k))
}
func (p *GlobRange) String() (s string) { return p.elemstr(nil, 0) }
func (p *GlobRange) Strval() (s string, err error) {
        var chars string
        if chars, err = p.Chars.Strval(); err == nil {
                s = fmt.Sprintf("[%s]", chars)
        }
        return
}
func (p *GlobRange) cmp(v Value) (res cmpres) {
        if a, ok := v.(*GlobRange); ok {
                assert(ok, "value is not GlobRange")
                res = p.Chars.cmp(a.Chars)
        }
        return
}

type Path struct {
        trivial
        elements
        //File *File // if this path is pointed to a file, ie. the last element matched a FileMap
}
func (p *Path) elemstr(o Object, k elemkind) (s string) {
        for i, elem := range p.Elems {
                var v = elementString(o, elem, k)
                if i > 0 {
                        s += PathSep + v
                } else if v != "" {
                        s += v
                } else if len(p.Elems) == 1 {
                        s += PathSep
                }
        }
        return
}
func (p *Path) String() (s string) { return p.elemstr(nil, 0) }
func (p *Path) Strval() (s string, e error) {
        for i, seg := range p.Elems {
                if seg == nil {
                        e = fmt.Errorf("`%s` nil path segment", p)
                        return
                }

                var v string
                if v, e = seg.Strval(); e != nil { return }
                if i > 0 {
                        s += PathSep + v
                } else if v != "" {
                        s += v
                } else if len(p.Elems) == 1 {
                        s += PathSep
                }
        }
        return
}
func (p *Path) True() (t bool) {
        for _, elem := range p.Elems {
                t = elem.True(); break
        }
        return
}
func (p *Path) refs(v Value) (res bool) { return p.elements.refs(v) }
func (p *Path) closured() (res bool) { return p.elements.closured() }
func (p *Path) expand(w expandwhat) (res Value, err error) {
        var (elems []Value; num int)
        if elems, num, err = expandall(w, p.Elems...); err != nil { return }
        if w&expandPath != 0 {
                var vals []Value
                for _, elem := range elems {
                        switch v := elem.(type) {
                        case *String:
                                segs := MakePathStr(elem.Position(),v.string).Elems
                                vals = append(vals, segs...)
                        default:
                                vals = append(vals, elem)
                        }
                }
                elems = vals
        }
        if num > 0 {
                res = &Path{p.trivial,elements{elems}/*, p.File*/}
        } else {
                res = p
        }
        return
}
func (p *Path) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }

        var s string // path/file target
        var rest []string
        if s, rest, err = p.stencil(pc.stems); err != nil { return }
        if false && len(rest) > 0 { panic("FIXME: unhandled stems") }

        if /*p.File == nil*/true {
                if pc.program.project.isFileName(filepath.Base(s)) || pc.program.project.isFileName(s) {
                        //if p.File = pc.program.project.searchFile(s); p.File != nil {
                        if file := pc.program.project.searchFile(s); file != nil {
                                pc.addNotExistedTarget1(file)
                                return
                        }
                }
        }

        /*if p.File != nil {
                err = p.File.traverse(pc)
                if err != nil { return }
        }*/

        var errs scanner.Errors
        var checked = make(map[string]bool)
        for _, err := range pc.tarverseTargetErrs(s) {
                e, isTargetNotFound := err.Err.(targetNotFoundError)
                if !isTargetNotFound {
                        errs = append(errs, err)
                        continue
                } else if b1, b2 := checked[e.target]; b1 && b2 {
                        continue
                } else {
                        checked[e.target] = true
                }
                //if p.File = stat(e.target, "", ""); p.File == nil {
                if file := stat(e.target, "", ""); file == nil {
                        pc.addNotExistedTarget1(&String{trivial{p.Position()},e.target}) // Append unknown path anyway.
                        err.Err = pathNotFoundError{e.project, p}
                        errs = append(errs, err)
                } else if /*p.File*/file.info != nil {
                        if /*p.File*/file.info.IsDir() {
                                //pc.addNotExistedTarget1(p)
                        } else {
                                //pc.addNotExistedTarget1(p/*.File*/)
                        }
                        pc.addNotExistedTarget1(file)
                } else {
                        // Search this path target as a file.
                        /*p.File*/file = e.project.searchFile(e.target)
                        if /*p.File*/file != nil {
                                pc.addNotExistedTarget1(/*p.File*/file)
                        } else {
                                errs = append(errs, err)
                        }
                }
        }

        checked = nil

        if len(errs) > 0 { err = errs } else { err = nil }
        return
}
func (p *Path) isPattern() (result bool) {
        for _, seg := range p.Elems {
                _, result = seg.(Pattern)
                if result { return }
        }
        return
}
func (p *Path) after(v Value) (after bool, err error) {
        // TODO: ...
        return
}
func (p *Path) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Path); ok {
                assert(ok, "value is not Path")
                res = p.cmpElems(a.Elems)
        }
        return
}
func (p *Path) match(i interface{}) (result string, stems []string, err error) {
        var retained []string
        result, retained, stems, err = p.partialMatch(i)
        if len(retained) > 0 {
                // clear results if not fully matched
                result, stems = "", nil
        }
        return
}
func (p *Path) partialMatch(i interface{}) (result string, retained, stems []string, err error) {
        switch t := i.(type) {
        case *File:
                result, retained, stems, err = p.match1(t.name)
                if err != nil {
                        return
                } else if /*result == "" &&*/ stems == nil {
                        s := filepath.Join(t.dir, t.sub, t.name)
                        result, retained, stems, err = p.match1(s)
                }
        case *filestub:
                result, retained, stems, err = p.match1(t.name)
                if err != nil {
                        return
                } else if /*result == "" &&*/ stems == nil {
                        s := filepath.Join(t.dir, t.sub, t.name)
                        result, retained, stems, err = p.match1(s)
                }
        default:
                result, retained, stems, err = p.match1(i)
        }
        return
}
func (p *Path) match1(i interface{}) (result string, retained, stems []string, err error) {
        var (
                srcs []string
                segs []Value
                idx = 0
        )
        switch t := i.(type) {
        case []string: srcs = t
        case string: srcs = strings.Split(t, PathSep)
        /*case *File: srcs = strings.Split(t.FullName(), PathSep)
        case *filestub:
                s := filepath.Join(t.dir, t.sub, t.name)
                srcs = strings.Split(s, PathSep)*/
        case Value:
                var s string
                if s, err = t.Strval(); err != nil { return }
                srcs = strings.Split(s, PathSep)
        default:
                unreachable("path.match: %T %v", i, i)
        }

        if segs, err = ExpandAll(p.Elems...); err != nil {
                return
        }

ForPathSegs:
        for n, seg := range segs {
                if len(srcs) <= idx { break ForPathSegs }

                var ( s string ; r []string )
                switch t := seg.(type) {
                case *Path:// Note that Path is also a Pattern!
                        var ss []string
                        if s, r, ss, err = t.match1(srcs[idx:]); err != nil {
                                break ForPathSegs
                        } else if s == "" {
                                return
                        } else if len(r) > 0 {
                                // not fully matched
                        }
                        stems = append(stems, ss...)
                        idx += len(strings.Split(s, PathSep))
                case Pattern:// Note that Path is also a Pattern!
                        var ss []string

                        // Special case of "%%" to match many segs at once.
                        if pp, ok := seg.(*PercPattern); ok {
                                if ps, ok := pp.Suffix.(*PercPattern); ok {
                                        if isNone(ps.Prefix) && isNone(ps.Suffix) {
                                                // for all /%%/ segs
                                                if n+1 < len(segs) && idx+1 < len(srcs) {
                                                        // Find 'next' matched seg, aka. %%/next/xxx
                                                        var next = segs[n+1] // e.g. '.smart' like in '%%/.smart/modules'
                                                        switch t := next.(type) {
                                                        case Pattern:
                                                                for x, src := range srcs[idx+1:] {
                                                                        var res string
                                                                        res, ss, err = t.match(src)
                                                                        if err != nil { return }
                                                                        if res != "" && len(ss) > 0 {
                                                                                end := idx + 1 + x
                                                                                stem := strings.Join(srcs[idx:end], PathSep)
                                                                                stems = append(stems, stem)
                                                                                idx = end
                                                                                continue ForPathSegs //break //
                                                                        }
                                                                }
                                                        default:
                                                                for x, src := range srcs[idx+1:] {
                                                                        var res string
                                                                        if res, err = t.Strval(); err != nil {
                                                                                return
                                                                        }
                                                                        if res == src {
                                                                                end := idx + 1 + x
                                                                                stem := strings.Join(srcs[idx:end], PathSep)
                                                                                stems = append(stems, stem)
                                                                                idx = end
                                                                                continue ForPathSegs //break //
                                                                        }
                                                                }
                                                        }
                                                        //continue ForPathSegs
                                                } else {
                                                        stem := strings.Join(srcs[idx:], PathSep)
                                                        stems = append(stems, stem)
                                                        idx = len(srcs)
                                                        break ForPathSegs
                                                }
                                        }
                                }
                        }
                        
                        if s, ss, err = t.match(srcs[idx]); err != nil {
                                break ForPathSegs
                        } else if s == "" || ss == nil {
                                return
                        }
                        stems = append(stems, ss...)
                        idx += 1
                default:
                        if s, err = seg.Strval(); err != nil {
                                break ForPathSegs
                        }
                        for _, s := range strings.Split(s, PathSep) {
                                if srcs[idx] != s { return }
                                idx += 1
                        }
                }
        }

        if 0 < idx && idx <= len(srcs) {
                // Don't use filepath.Join as in the case that ""
                // have to be root "/".
                result = strings.Join(srcs[:idx], PathSep)
                retained = srcs[idx:]
        }
        return
}
func (p *Path) stencil(stems []string) (result string, rest []string, err error) {
        var (
                strs []string
                segs []Value
        )
        if segs, err = ExpandAll(p.Elems...); err != nil {
                return
        }

ForPathSegs:
        for _, seg := range segs {
                var s string
                switch t := seg.(type) {
                case Pattern: // including *Path
                        if len(stems) > 0 {
                                s, stems, err = t.stencil(stems)
                                if err != nil { return }
                                strs = append(strs, s)
                                continue ForPathSegs
                        }
                }
                if s, err = seg.Strval(); err != nil {
                        break ForPathSegs
                } else {
                        strs = append(strs, s)
                }
        }
        if err == nil {
                result = strings.Join(strs, PathSep)
                rest = stems // the rest stems
        }
        return
}

type PathSeg struct {
        trivial
        rune
}
func (p *PathSeg) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *PathSeg) String() (s string) { 
        var e error
        if s, e = p.Strval(); e != nil { s = "?" }
        return
}
func (p *PathSeg) Strval() (s string, e error) {
        switch p.rune {
        case '/': s = "" // the first '/', aka. root -- PathSep is added when joining
        case '~': s = "~"
        case '.': s = "."
        case '^': s = ".."
        case 0: s = "" // empty segment after the last '/', e.g. /foo/bar/ 
        default: e = fmt.Errorf("unknown pathseg (%s)", p.rune)
        }
        return
}
func (p *PathSeg) cmp(v Value) (res cmpres) {
        if a, ok := v.(*PathSeg); ok && p.rune == a.rune {
                res = cmpEqual
        }
        return
}

type filestub struct {
        // TODO: project *Project // the project in which the file was found
        dir string       // full directory where the file was or should be found
        sub string       // matched sub path (see Project.search), may be Dir (absoletep path)
        name string      // constant represented name (e.g. relative filename)
        match *FileMap   // matched pattern (see 'files' directive)
        other *filestub  // pointed to another stub (in a different project) of the same file
}

type filebase struct {
        stub filestub    // cycled-list of file stubs of different projects
        info os.FileInfo // file info if exists
        updated bool // true if this file has been updated by a program
}

var filecache = make(map[string]*filebase) // File.FullName() -> File
var statmutex = new(sync.Mutex)

func (p *filestub) subname() (s string) {
        if isAbsOrRel(p.sub) {
                s = p.name
        } else {
                s = filepath.Join(p.sub, p.name)
        }
        return
}
func (p *filebase) exists() bool { return p.info != nil }

func stat(name, sub, dir string, infos ...os.FileInfo) (file *File) {
        var ( base *filebase ; stub *filestub ; fullname string )

        statmutex.Lock()
        defer statmutex.Unlock()

        // Trims / suffix
        if dir != "" { dir = filepath.Clean(dir) }
        if sub != "" { sub = filepath.Clean(sub) }
        name = filepath.Clean(name)

        if filepath.IsAbs(name) {
                if fullname = name; dir == "" {
                        dir, sub = filepath.Dir(fullname), ""
                        name = filepath.Base(fullname)
                        if enable_assertions {
                                assert(sub == "", "invalid file{%s %s %s}", dir, sub, name)
                        }
                } else if strings.HasPrefix(fullname, dir+PathSep) {
                        //tail := strings.TrimPrefix(fullname, dir)
                        //tail  = strings.TrimPrefix(tail, PathSep)
                        tail := fullname[len(dir)+1:]
                        sub  = filepath.Dir(tail)
                        name = filepath.Base(tail)
                        if enable_assertions {
                                assert(filepath.Join(dir, sub, name) == fullname, "(%s %s %s) components conflicted: %s", fullname)
                        }
                } else if false {
                        dir, sub = filepath.Dir(fullname), ""
                        name = filepath.Base(fullname)
                } else if false {
                        dir, sub = filepath.Dir(fullname), ""
                } else if true {
                        dir = "" //dir = filepath.Dir(fullname)
                        sub = ""
                } else {
                        unreachable("conflicted dir prefix: ", dir, " ", sub, " ", name)
                }
        } else if filepath.IsAbs(sub) {
                fullname = filepath.Join(sub, name)
                if dir == "" {
                        dir = sub // trims / suffix
                        sub = "" // .
                } else if sub == dir {
                        sub = "" // .
                } else if strings.HasPrefix(sub, dir) {
                        sub = strings.TrimPrefix(sub, dir)
                        sub = strings.TrimPrefix(sub, PathSep)
                        sub = filepath.Clean(sub)
                } else if false {
                        dir = sub
                        sub = ""
                } else {
                        unreachable("conflicted sub/dir: ", sub, " ", dir) //return
                }
        } else if filepath.IsAbs(dir) {
                fullname = filepath.Join(dir, sub, name)
        } else {
                fullname = filepath.Join(context.workdir, dir, sub, name)
        }

        fullname = filepath.Clean(fullname)

        if enable_assertions {
                assert(filepath.IsAbs(fullname), "`%s` is not abs {%s %s %s}", fullname, name, sub, dir)
                if filepath.IsAbs(name) {
                        assert(dir == "", "`%s` invalid file{%s %s %s}", fullname, dir, sub, name)
                        assert(sub == "", "`%s` invalid file{%s %s %s}", fullname, dir, sub, name)
                }
                assert(!filepath.IsAbs(sub), "`%s` sub is abs", sub)
                
                if filepath.IsAbs(name) {
                        s := name
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                } else if filepath.IsAbs(sub) {
                        s := filepath.Join(sub, name)
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                } else if filepath.IsAbs(dir) {
                        s := filepath.Join(dir, sub, name)
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                } else {
                        s := filepath.Join(context.workdir, dir, sub, name)
                        assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
                }
        }

        var addNotExisted bool
        var info os.FileInfo
        if len(infos) == 1 {
                if info = infos[0]; info == nil {
                        addNotExisted = true
                }
                if enable_assertions && info != nil {
                        assert(info.Name() == filepath.Base(fullname), "`%s` file name conflicted", info.Name())
                }
        } else if len(infos) > 1 {
                unreachable("too many file infos")
        }

        var okay bool
        if base, okay = filecache[fullname]; okay {
                if base.info == nil {
                        if info == nil { info, _ = os.Stat(fullname) }
                        if info == nil && !addNotExisted {
                                return nil // file not exists
                        }
                        base.info = info
                }

                var head = &base.stub
                /*
                if enable_assertions {
                        for stub = head; stub != nil ; stub = stub.other {
                                s := filepath.Join(stub.dir, stub.sub, stub.name)
                                assert(fullname == s, "(%s %s %s) fullname conflicted", stub.dir, stub.sub, stub.name)
                                if stub.other == head { break }
                        }
                }
                */
                for stub = head; stub != nil; stub = stub.other {
                        if stub.dir == dir && stub.sub == sub && stub.name == name {
                                goto GotFile
                        }
                        if stub.other == head { break }
                }

                stub = &filestub{ dir, sub, name, nil, head.other }
                head.other = stub
        } else {
                if info == nil {
                        info, _ = os.Stat(fullname)
                        if info == nil && !addNotExisted {
                                return nil // file not exists
                        }
                }

                base = &filebase{ filestub{ dir, sub, name, nil, nil }, info, false }
                base.stub.other = &base.stub
                stub = &base.stub
                filecache[fullname] = base
        }
GotFile:
        file = &File{trivial{},base,stub} // FIXME: needs position information
        if enable_assertions {
                if !addNotExisted {
                        assert(file.exists(), "`%s` file not existed", fullname)
                }
                assert(file.name == name, "(%s %s %s).name != %s", file.name, file.sub, file.dir, name)
                assert(file.sub == sub, "(%s %s %s).sub != %s", file.name, file.sub, file.dir, sub)
                if file.dir != dir {
                        var head = &base.stub
                        for stub := head; stub != nil; stub = stub.other {
                                fmt.Fprintf(stderr, "stat: %s %s %s\n", stub.dir, stub.sub, stub.name)
                                if stub.other == head { break }
                        }
                }
                assert(file.dir == dir, "(%s %s %s).dir != %s", file.name, file.sub, file.dir, dir)
                //assert(file.dir != "", "(%s %s %s) empty dir", file.name, file.sub, file.dir)
                if file.exists() {
                        assert(file.info != nil, "(%s %s %s) info is nil", file.name, file.sub, file.dir)
                        assert(file.info.Name() == filepath.Base(file.name), "(%s %s %s) name conflicted", file.name, file.sub, file.dir)
                        s := filepath.Join(file.dir, file.sub, file.name)
                        assert(file.FullName() == s, "(%s %s %s) fullname conflicted (%s)", file.dir, file.sub, file.name, s)
                }
        }
        return
}

type File struct {
        trivial
        *filebase
        *filestub
}
func (p *File) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *File) True() bool { return p.name != "" }
func (p *File) String() string { return p.name }
func (p *File) Strval() (s string, err error) { s = p.FullName(); return }
func (p *File) BaseName() (s string) {
        if p.info != nil {
                s = p.info.Name()
        } else {
                s = filepath.Base(p.name)
        }
        return
}
func (p *File) FullName() (s string) {
        return filepath.Join(p.dir, p.sub, p.name)
}
func (p *File) searchInMatchedPaths(proj *Project) (res bool) {
        if p.match != nil {
                var pre string
                // FIXME: File should keep both 'match' and 'pre',
                // or just remove searchInMatchedPaths
                f := p.match.stat(proj.absPath, pre, p.name)
                res = f != nil && f.exists()
        }
        return
}
func (p *File) traverse(pc *traversal) (err error) {
        if optionTraceTraversal {
                defer un(tt(pc, p))
                if p.exists() { pc.tracef("exists: %T{%s}", p, p) }
        }

        var after bool
        if pc.entry.target == p {
                pc.tracef("error: target depends on itself")
                unreachable(p, "target depends on itself")
        } else if !p.exists() {
                if optionTraceTraversal { pc.tracef("!exists: update %v", p) }
                if err = pc.updateFile(p); err == nil {
                        pc.updated = append(pc.updated, newUpdatedTarget(p, nil))
                }
        } else if after, err = p.after(pc.entry.target); after && err == nil {
                if optionTraceTraversal { pc.tracef("after: %v", p) }
                pc.updated = append(pc.updated, newUpdatedTarget(p, nil))
        }
        return
}

// check pattern depends to find out if all depends are updatable
// or updated/exists.
func checkPatternDepends(pc *traversal, project *Project, se *StemmedEntry, prog *Program) (res bool, err error) {
        if len(prog.depends) == 0 {
                // Pattern is always good as no depends to check.
                return true, nil
        }

        // Set arguments in case that depends may refer to a parameter.
        if prog.params == nil || pc.arguments == nil {
                // no need to set arguments
        } else if e, clearParams := prog.setParams(pc.arguments); e != nil {
                err = e; return
        } else {
                defer clearParams()
        }

        var checkedPatterns = 0
        for _, dep := range prog.depends {
                switch d := dep.(type) {
                case Pattern:
                        res, err = checkPatternDepend(pc, project, se, prog, d)
                        checkedPatterns += 1
                        if err != nil { return }
                        if !res { break }
                case *Argumented:
                        var ok, res1 bool
                        ok, res1, err = d.checkPatternDepends(pc, project, se, prog)
                        if err != nil { return }
                        if ok && !res1 { break }
                        res = res1
                default:
                        /*
                        var name, str string
                        var rest []string // rest stems
                        name, rest, err = se.stencil(se.Stems)
                        if err != nil { return }
                        if len(rest) > 0 { panic("FIXME: unhandled stems") }
                        if str, err = dep.Strval(); err != nil { return }
                        if res = str == name; !res { break }
                        */
                }
        }
        if !res && checkedPatterns == 0 {
                // If there's no pattern depends, we're good to use the
                // pattern to update target.
                res = true
        }
        return
}

func checkPatternDepend(pc *traversal, project *Project, se *StemmedEntry, prog *Program, pat Pattern) (res bool, err error) {
        var name string
        var rest []string // rest stems
        if name, rest, err = pat.stencil(se.Stems); err != nil { return }
        if false && len(rest) > 0 { panic("FIXME: unhandled stems") }

        // Check entires (and patterns) no matter if the file exists or
        // or not.
        var entry *RuleEntry
        if entry, err = project.resolveEntry(name); err != nil {
                return
        } else if entry != nil {
                return true, nil
        }

        var ses []*StemmedEntry
        if ses, err = project.resolvePatterns(name); err != nil {
                return
        } else /*if len(ses) > 0*/ {
        ForPatterns:
                for _, se := range ses {
                        for _, prog := range se.programs {
                                var ok bool
                                ok, err = checkPatternDepends(pc, project, se, prog)
                                if !ok { continue ForPatterns }
                        }
                        return true, nil
                }
        }

        // Matches a FileMap (IsKnown(), may exists or not)
        var file = project.searchFile(name)
        if file != nil && file.exists() {
                return true, nil
        }

        if filepath.IsAbs(name) {
                file = stat(name, "", "")
                if file != nil && file.exists() {
                        return true, nil
                }
        }

        // TODO: check filepath.Join(project.absPath, name)
        return
}

func (p *File) after(v Value) (after bool, err error) {
        if p.info != nil {
                switch t := v.(type) {
                case *File:
                        if t.info != nil {
                                after = p.info.ModTime().After(t.info.ModTime())
                        }
                }
        }
        return
}

func (p *File) cmp(v Value) (res cmpres) {
        if v == nil {
                // ...
        } else if a, ok := v.(*File); ok {
                assert(ok, "value is not File")
                if a == nil {
                        //assert(a != nil, "nil file")
                } else if p.filebase == a.filebase {
                        res = cmpEqual
                } else if p.FullName() == a.FullName() {
                        s := fmt.Sprintf("\na: %s %s %s (%s)", p.dir, p.sub, p.name, p.FullName())
                        s += fmt.Sprintf("\nb: %s %s %s (%s)", a.dir, a.sub, a.name, a.FullName())
                        unreachable("same files differed: ", p.name, " != ", a.name, s)
                } else if false /*p.dir != a.dir && p.sub == a.sub && p.name == a.name*/ {
                        s := fmt.Sprintf("\n      a: %s: %s %s", p.name, p.dir, p.sub)
                        s += fmt.Sprintf("\n      b: %s: %s %s", a.name, a.dir, a.sub)
                        fmt.Fprintf(stderr, "warning: files may differ: %s != %s :%s\n", p.name, a.name, s)
                }
        }
        return
}

func (p *File) change(dir, sub, name string) (okay bool) {
        var fullname = filepath.Join(dir, sub, name)
        if p.FullName() == fullname {
                var head = &p.filebase.stub
                for stub := p.filestub; stub != nil; stub = stub.other {
                        if stub.dir == dir && stub.sub == sub && stub.name == name {
                                p.filestub, okay = stub, true
                                return
                        }
                        if stub.other == head { break }
                }
                
                p.filestub = &filestub{ dir, sub, name, nil, head.other }
                head.other, okay = p.filestub, true
                
                if enable_assertions {
                        assert(p.FullName() == fullname, "Changed invalid File")
                }
        }
        return
}

type FileContent struct {
        file *File
        content []byte
}

type Flag struct { trivial ; name Value }
func (p *Flag) refs(v Value) bool { return p.name.refs(v) }
func (p *Flag) closured() bool { return p.name.closured() }
func (p *Flag) expand(w expandwhat) (res Value, err error) {
        var name Value
        if name, err = p.name.expand(w); err == nil {
                if name != p.name {
                        res = &Flag{p.trivial,name}
                } else {
                        res = p
                }
        }
        return
}
func (p *Flag) True() bool { return p.name.True() }
func (p *Flag) elemstr(o Object, k elemkind) (s string) {
        return "-" + elementString(o, p.name, k)
}
func (p *Flag) String() (s string) { return p.elemstr(nil, 0) }
func (p *Flag) Strval() (s string, e error) {
        if p.name == nil {
                s = "-"
        } else if  _, ok := p.name.(*None); ok {
                s = "-"
        } else if s, e = p.name.Strval(); e == nil {
                s = "-" + s
        }
        return
}
func (p *Flag) _opts(opts ...string) (runes []rune, names []string, err error) {
        switch t := p.name.(type) {
        case *Flag:
                runes, names, err = t._opts(opts...)
        case *String:
                for _, opt := range opts {
                        if t.string == opt {
                                if len(opt) > 0 {
                                        names = append(names, opt)
                                }
                        }
                }
        case *Bareword:
                for _, opt := range opts {
                        if i := strings.IndexRune(opt, ','); i == 0 {
                                if t.string == opt[1:] {
                                        names = append(names, opt)
                                }
                        } else if i > 0 {
                                if t.string == opt[i+1:] {
                                        runes = append(runes, rune(opt[0]))
                                        names = append(names, opt[1:])
                                } else if strings.ContainsAny(t.string, opt[0:i]) {
                                        runes = append(runes, rune(opt[0]))
                                        names = append(names, opt[i+1:])
                                }
                        }
                }
        }
        if enable_assertions {
                assert(len(runes) == len(names), "unmatched opts lengths")
        }
        return
}
func (p *Flag) cmp(v Value) (res cmpres) {
        if v == nil {
                // ...
        } else if a, ok := v.(*Flag); ok {
                res = p.name.cmp(a.name)
        }
        return
}
      
type Compound struct { trivial ; elements } // "compound string"
func (p *Compound) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Compound{p.trivial,elements{elems}}
                } else {
                        res = p
                }
        }
        return
}
func (p *Compound) elemstr(o Object, k elemkind) (s string) {
        var tk = k|elemNoQuote
        for _, elem := range p.Elems {
                s += elementString(o, elem, tk)
        }
        if k&elemNoQuote == 0 { s = `"`+s+`"` }
        return
}
func (p *Compound) String() string { return p.elemstr(nil, 0) }
func (p *Compound) Strval() (s string, err error) {
        for _, e := range p.Elems {
                var v string
                if v, err = e.Strval(); err == nil {
                        s += v
                } else {
                        break
                }
        }
        return
}
func (p *Compound) Float() (f float64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                f, err = strconv.ParseFloat(s, 64)
        }
        return
}
func (p *Compound) Integer() (i int64, err error) {
        var s string
        if s, err = p.Strval(); err == nil {
                i, err = strconv.ParseInt(s, 10, 64)
        }
        return
}
func (p *Compound) True() (res bool) {
        if s, err := p.Strval(); err == nil {
                res = s != ""
        }
        return
}
func (p *Compound) refs(v Value) bool { return p.elements.refs(v) }
func (p *Compound) closured() bool { return p.elements.closured() }
func (p *Compound) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Compound); ok {
                assert(ok, "value is not Compound")
                s1, e := p.Strval()
                if e != nil { return }
                s2, e := a.Strval()
                if e != nil { return }
                if s1 == s2 { res = cmpEqual }
        }
        return
}

type List struct { elements }
func (p *List) elemstr(o Object, k elemkind) (s string) {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elementString(o, elem, k))
        }
        return strings.Join(strs, " ")
}
func (p *List) Position() (pos Position) {
        if len(p.Elems) > 0 {
                pos = p.Elems[0].Position()
        }
        return
}
func (p *List) Float() (float64, error) {
        i, e := p.Integer(); return float64(i), e
}
func (p *List) Integer() (int64, error) {
        if n := len(p.Elems); n == 1 {
                // If there's only one element, treat it as a scalar.
                return p.Elems[0].Integer()
        } else {
                return int64(n), nil
        }
}
func (p *List) String() (s string) { return p.elemstr(nil, 0) }
func (p *List) Strval() (s string, err error) {
        var x = 0
        for _, e := range p.Elems {
                var v string
                if v, err = e.Strval(); err == nil {
                        if v != "" {
                                if 0 < x { s += " " }
                                s += v
                                x += 1
                        }
                } else {
                        break
                }
        }
        return
}

func (p *List) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &List{ elements{ elems } }
                } else {
                        res = p
                }
        }
        return
}

func (p *List) traverse(pc *traversal) (err error) {
        if len(p.Elems) == 0 { return } 
        if optionTraceTraversal { defer un(tt(pc, p)) }
        var modified, updates, good *breaker
        for _, v := range p.Elems {
                var pre, ok = v.(prerequisite)
                if !ok {
                        err = fmt.Errorf("%T `%s` is not prerequisite", v, v)
                        break
                }
                if err = pre.traverse(pc); err == nil {
                        continue // The element target is good!
                } else if br, ok := err.(*breaker); ok {
                        switch br.what {
                        case breakModified:
                                if modified == nil { modified = br } else {
                                        modified.modified = append(modified.modified, br.modified...)
                                }
                        case breakUpdates:
                                if updates == nil { updates = br } else {
                                        updates.updated = append(updates.updated, br.updated...)
                                }
                                err = nil
                        case breakGood:
                                err, good = nil, br
                        case breakDone, breakNext, breakCase:
                                err = nil
                        default:
                                fmt.Fprintf(stderr, "%s: list: %v: %v (%v)\n", pc.program.position, pc.entry, v, br.what)
                        }
                } else {
                        break
                }
        }
        if err != nil {
                //fmt.Printf("%s: %v: prepare list error\n", pc.program.position, pc.entry)
        } else if modified != nil && err != modified {
                err = modified
        } else if updates != nil && err != updates {
                err = updates
        } else if err == nil && good != nil {
                err = good
        }
        return
}

func (p *List) after(v Value) (after bool, err error) {
        for _, elem := range p.Elems {
                if after, err = elem.after(v); err != nil && after { break }
        }
        return
}

func (p *List) cmp(v Value) (res cmpres) {
        if a, ok := v.(*List); ok {
                assert(ok, "value is not List")
                res = p.cmpElems(a.Elems)
        }
        return
}

type Group struct { trivial ; List }
func (p *Group) elemstr(o Object, k elemkind) string {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elementString(o, elem, k))
        }
        return fmt.Sprintf("(%s)", strings.Join(strs, " "))
}
func (p *Group) after(v Value) (bool, error) { return p.trivial.after(v) }
func (p *Group) Position() Position { return p.trivial.Position() }
func (p *Group) Float() (float64, error) { return p.trivial.Float() }
func (p *Group) Integer() (int64, error) { return p.trivial.Integer() }
func (p *Group) String() string { return p.elemstr(nil, 0) }
func (p *Group) Strval() (s string, err error) {
        if s, err = p.List.Strval(); err == nil {
                s = "(" + s + ")"
        }
        return
}
func (p *Group) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Group{p.trivial,List{elements{elems}}}
                } else {
                        res = p
                }
        }
        return
}
func (p *Group) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Group); ok {
                assert(ok, "value is not Group")
                res = p.cmpElems(a.Elems)
        }
        return
}

type Pair struct { // key=value
        Key Value
        Value Value
}
func (p *Pair) refs(v Value) bool { return p.Key.refs(v) || p.Value.refs(v) }
func (p *Pair) closured() bool { return p.Key.closured() || p.Value.closured() }
func (p *Pair) expand(x expandwhat) (res Value, err error) {
        var k, v Value
        res = p // set the original value
        if k, err = p.Key.expand(x); err == nil {
                if v, err = p.Value.expand(x); err == nil {
                        if k != p.Key || v != p.Value {
                                if k == nil { k = p.Key }
                                if v == nil { v = p.Value }
                                res = &Pair{ k, v }
                        }
                }
        }
        return
}
func (p *Pair) Position() Position { return p.Key.Position() }
func (p *Pair) True() bool { return p.Value.True() || p.Key.True() }
func (p *Pair) elemstr(o Object, k elemkind) string {
        return elementString(o, p.Key, k)+`=`+elementString(o, p.Value, k)
}
func (p *Pair) String() string { return p.elemstr(nil, 0) }
func (p *Pair) Strval() (s string, err error) {
        var k, v string
        if k, err = p.Key.Strval(); err == nil {
                if v, err = p.Value.Strval(); err == nil {
                        s = k + "=" + v
                }
        }
        return
}
func (p *Pair) Integer() (int64, error) { return p.Value.Integer() }
func (p *Pair) Float() (float64, error) { return p.Value.Float() }
func (p *Pair) SetValue(v Value) { p.Value = v }
func (p *Pair) SetKey(k Value) {
        switch o := k.(type) {
        case *Pair: k = o.Key
        }
        p.Key = k
}
func (p *Pair) after(v Value) (after bool, err error) { return }
func (p *Pair) cmp(v Value) (res cmpres) {
        if a, ok := v.(*Pair); ok {
                assert(ok, "value is not Pair")
                if p.Key.cmp(a.Key) == cmpEqual {
                        if p.Value.cmp(a.Value) == cmpEqual {
                                res = cmpEqual
                        }
                }
        }
        return
}

type closuredelegate struct {
        l token.Token
        o Object
        a []Value
}
func (p *closuredelegate) string(o Object, k elemkind) (s string) { // source representation
        for i, a := range p.a {
                if i == 0 { s = " " } else { s += "," }
                s += elementString(o, a, k)
        }
        switch name := p.o.Name(); p.l {
        case token.LCOLON:
                if p.o == context.globe.os.self {
                        s = ":os:"
                } else {
                        s = fmt.Sprintf(":%s%s:", name, s)
                }
        case token.LPAREN: s = fmt.Sprintf("(%s%s)", name, s)
        case token.LBRACE:
                if k&elemNoBrace == 0 {
                        s = fmt.Sprintf("{%s%s}", name, s)
                } else {
                        s = fmt.Sprintf("(%s%s)", name, s)
                }
        case token.STRING, token.COMPOUND:
                s = fmt.Sprintf("%s%s", name, s)
        case token.ILLEGAL:
                if len(name) == 1 && len(s) == 0 {
                        s = fmt.Sprintf("%s", name)
                } else {
                        s = fmt.Sprintf("[%s%s]", name, s)
                }
        default:
                s = fmt.Sprintf("[%s%s]!(%v)", name, s, p.l)
        }
        return
}

// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct { trivial ; closuredelegate }
func (p *delegate) True() (t bool) {
        if v, err := p.expand(expandAll); err == nil {
                t = v.True()
        }
        return
}
func (p *delegate) elemstr(o Object, k elemkind) (s string) {
        if k&elemExpand == 0 {
                s = "$"+p.string(o, k)
        } else if v, e := p.expand(expandDelegate); e == nil {
                s = elementString(o, v, k)
        }
        return
}
func (p *delegate) String() (s string) { return p.elemstr(nil, 0) }
func (p *delegate) Strval() (string, error) { if v, e := p.expand(expandDelegate); e == nil { return v.Strval() } else { return "", e }}
func (p *delegate) Integer() (int64, error) { if v, e := p.expand(expandDelegate); e == nil { return v.Integer() } else { return 0, e }}
func (p *delegate) Float() (float64, error) { if v, e := p.expand(expandDelegate); e == nil { return v.Float() } else { return 0, e }}
func (p *delegate) expand(w expandwhat) (res Value, err error) {
        switch {
        case w&expandClosure != 0:
                if res, err = p.disclose(); err != nil {
                        return
                }
                if res != nil && w&expandDelegate != 0 {
                        res, err = res.expand(expandDelegate)
                }
        case w&expandDelegate != 0:
                if res, err = p.reveal(); err != nil {
                        return
                }
                if res != nil && w&expandClosure != 0 {
                        res, err = res.expand(expandClosure)
                }
        }
        if err == nil && res == nil { res = p }
        return
}
func (p *delegate) reveal() (res Value, err error) {
        var args []Value
        if args, _, err = expandall(expandClosure, p.a...); err != nil {
                return
        }

        switch o := p.o.(type) {
        default: err = fmt.Errorf("%T '%v' is unknown delegation", o, o)
        case Caller:
                if res, err = o.Call(p.Position(), args...); err != nil {
                        if p.o.Name() != "error" {
                                err = scanner.WrapErrors(token.Position(p.Position()), err)
                        } else {
                                return
                        }
                }
        case Executer:
                if args, err = o.Execute(p.Position(), args...); err != nil {
                        if p.o.Name() != "error" {
                                err = scanner.WrapErrors(token.Position(p.Position()), err)
                        } else {
                                return
                        }
                } else {
                        res = &List{elements{args}}
                }
        }

        if err != nil {
                //fmt.Fprintf(stderr, "%v: %v\n", p.p, err)
        } else if res == nil {
                res = &None{}
        }
        return
}
func (p *delegate) disclose() (res Value, err error) {
        var ( o = p.o; v Value; changed bool )
        if v, err = o.expand(expandClosure); err != nil { return }
        if v != nil {
                if o, _ = v.(Object); o != nil {
                        changed = true
                } else {
                        err = fmt.Errorf("invalid delegate %v", v)
                        return
                }
        }

        var args []Value
        for _, a := range p.a {
                if v, err = a.expand(expandClosure); err != nil { return }
                if v != nil { a, changed = v, true }
                args = append(args, a)
        }
        if err == nil {
                if changed {
                        res = &delegate{p.trivial,closuredelegate{p.l,o,args}}
                } else {
                        res = p
                }
        }
        return
}
func (p *delegate) refs(v Value) bool {
        if p.o == v || p.o.refs(v) {
                return true
        }
        for _, a := range p.a {
                if a.refs(v) {
                        return true
                }
        }
        return false
}
func (p *delegate) closured() bool {
        if p.o.closured() { return true }
        for _, a := range p.a {
                if a.closured() { return true }
        }
        return false
}
func (p *delegate) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }

        var val Value
        if val, err = p.expand(expandDelegate); err != nil { return }
        err = pc.traverse(val) //err = pc.traverseAll(merge(val))
        return
}
func (p *delegate) after(v Value) (after bool, err error) {
        // TODO: ...
        return
}
func (p *delegate) cmp(v Value) (res cmpres) {
        if a, ok := v.(*delegate); ok {
                assert(ok, "value is not delegate")
                // FIXME: compare the expanded value instead??
                if p.o.cmp(a.o) == cmpEqual && len(p.a) == len(a.a) {
                        for i, t := range p.a {
                                if t.cmp(a.a[i]) != cmpEqual {
                                        return
                                }
                        }
                        res = cmpEqual
                }
        }
        return
}

type closure struct { trivial ; closuredelegate }
func (p *closure) True() (t bool) {
        if s, err := p.Strval(); err == nil {
                t = s != ""
        }        
        return
}
func (p *closure) elemstr(o Object, k elemkind) (s string) {
        if k&elemExpand == 0 {
                s = "&"+p.string(o, k)
        } else if v, e := p.expand(expandDelegate); e == nil {
                s = elementString(o, v, k)
        }
        return
}
func (p *closure) String() (s string) { return p.elemstr(nil, 0) }
func (p *closure) Strval() (s string, err error) {
        var v Value

        // &(...) -> $(...)
        if v, err = p.expand(expandClosure); err != nil {
                return
        } else if v == nil {
                //err = fmt.Errorf("{closure %+v &<nil>}", p.o)
                return
        }

        // $(...) -> .....
        if v, err = v.expand(expandDelegate); err != nil {
                return
        } else if v != nil {
                s, err = v.Strval()
        } else {
                //err = fmt.Errorf("{closure %+v $<nil>}", p.o)
        }
        return
}
func (p *closure) expand(w expandwhat) (res Value, err error) {
        switch {
        case w&expandClosure != 0:
                if res, err = p.disclose(); err != nil {
                        return
                }
                if res != nil && w&expandDelegate != 0 {
                        res, err = res.expand(expandDelegate)
                }
        case w&expandDelegate != 0:
                if res, err = p.reveal(); err != nil {
                        return
                }
                if res != nil && w&expandClosure != 0 {
                        res, err = res.expand(expandClosure)
                }
        }
        if err == nil && res == nil { res = p }
        return
}
func (p *closure) reveal() (res Value, err error) {
        if p.o == nil { return }

        var ( t Value; o Object )
        if t, err = p.o.expand(expandDelegate); err != nil { return }
        if t != nil {
                if o, _ = t.(Object); o == nil {
                        err = fmt.Errorf("closure.reveal: %T '%s' is not object", t, t)
                        return
                }
        }
        
        var ( a []Value; num int )
        for _, v := range p.a {
                if t, err = v.expand(expandDelegate); err != nil { return }
                if t == nil { t = v } else { num = num + 1 }
                a = append(a, t)
        }

        if o != nil || num > 1 {
                res = &closure{p.trivial,closuredelegate{p.l,o,a}}
        }
        return
}
func (p *closure) disclose() (res Value, err error) {
        if p.o == nil { return nil, nil }

        var ( o Object; v Value; changed bool )
        SeeL: switch name := p.o.Name(); p.l {
        case token.LPAREN, token.ILLEGAL:
                for _, scope := range cloctx {
                        if scope.project == nil {
                                if _, o = scope.Find(name); o != nil {
                                        changed = true; break SeeL
                                }
                                continue
                        }
                        if scope != scope.project.scope {
                                // inquire non-project scope first
                                if _, o = scope.Find(name); o != nil {
                                        changed = true; break SeeL
                                }
                        }
                        if o, err = scope.project.resolveObject(name); err != nil {
                                return
                        } else if o != nil {
                                changed = true; break SeeL
                        }
                }
        case token.LBRACE, token.STRING, token.COMPOUND:
                for _, scope := range cloctx {
                        if o, err = scope.project.resolveEntry(name); err != nil {
                                return
                        } else if o != nil {
                                if p.l == token.LBRACE {
                                        changed = true; break SeeL
                                } else {
                                        // &'xxx' and &"xxx" are not disclosed into
                                        // the resolved objects instead of converting
                                        // into delegates.
                                        res = o; return
                                }
                        }
                }
        default:
                err = fmt.Errorf("unknown closure `&%+v%+v`", p.l, name)
                return
        }

        if o == nil { o = p.o } // assert changed == false

        // Disclose the object, which may contain closures.
        if v, err = o.expand(expandClosure); err != nil {
                return
        } else if v != nil {
                var ok bool
                if o, ok = v.(Object); ok && o != nil {
                        changed = true
                } else {
                        err = fmt.Errorf("invalid closure %+v", v)
                        return
                }
        }

        var args []Value
        for _, a := range p.a {
                if v, err = a.expand(expandClosure); err != nil { return }
                if v != nil { a, changed = v, true }
                args = append(args, a)
        }

        if changed && err == nil {
                res = &delegate{p.trivial,closuredelegate{p.l,o,args}}
        }
        return
}
func (p *closure) refs(v Value) bool {
        if p.o == v {
                return true
        }
        for _, a := range p.a {
                if a.refs(v) {
                        return true
                }
        }
        return false
}
func (p *closure) closured() bool { return true }
func (p *closure) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }

        if v, e := p.expand(expandClosure); e != nil {
                err = e
        } else if v == nil {
                err = fmt.Errorf("undefined closure target `%v`", p.o.Name())
                fmt.Fprintf(stderr, "%s: closure.prepare: %v\n", p.Position(), err)
        } else {
                err = pc.traverse(v)
        }
        return
}
func (p *closure) after(v Value) (after bool, err error) {
        // TODO: ...
        return
}
func (p *closure) cmp(v Value) (res cmpres) {
        if a, ok := v.(*closure); ok {
                assert(ok, "value is not closure")
                // FIXME: compare the expanded value instead??
                if p.o.cmp(a.o) == cmpEqual && len(p.a) == len(a.a) {
                        for i, t := range p.a {
                                if t.cmp(a.a[i]) != cmpEqual {
                                        return
                                }
                        }
                        res = cmpEqual
                }
        }
        return
}

type selection struct {
        trivial
        t token.Token
        o Value // Object or selection
        s Value
}
func (p *selection) True() (t bool) {
        if s, err := p.Strval(); err == nil {
                t = s != ""
        }
        return
}
func (p *selection) elemstr(o Object, k elemkind) (s string) {
        s = elementString(o, p.o, k) + p.t.String()
        s += elementString(o, p.s, k)
        return
}
func (p *selection) String() string { return p.elemstr(nil, 0) }
func (p *selection) objectName() (s string) {
        switch t := p.o.(type) {
        case Object: s = t.Name()
        }
        return
}
func (p *selection) propName() (s string) {
        switch t := p.s.(type) {
        case Object: s = t.Name()
        case *Bareword: s = t.string
        case *String: s = t.string
        }
        return
}
func (p *selection) object() (o Object, err error) {
        if s, ok := p.o.(*selection); ok {
                var v Value
                if v, err = s.value(); err != nil {
                        // sth's wrong!
                } else if o, _ = v.(Object); o == nil {
                        err = fmt.Errorf("selection.object: `%s` is nil", s.String())
                }
        } else if o, ok = p.o.(Object); !ok {
                err = fmt.Errorf("selection.object: %T '%v' is not object", p.o, p.o)
        }
        return
}
func (p *selection) value() (v Value, err error) {
        var o Object
        if p.s == nil {
                err = fmt.Errorf("selection.value: nil prop `%s`", p.String())
        } else if o, err = p.object(); err != nil {
                // sth's wrong!
        } else if s := ""; o != nil {
                if s, err = p.s.Strval(); err == nil {
                        if pn, ok := o.(*ProjectName); ok && (p.t == token.SELECT_PROG1 || p.t == token.SELECT_PROG2) {
                                var entry *RuleEntry
                                if entry, err = pn.project.resolveEntry(s); err != nil {
                                        return
                                } else if entry == nil {
                                        err = fmt.Errorf("selection.value: no entry `%s` (%+v)", s, p.String())
                                } else {
                                        v = entry
                                }
                        } else if v, err = o.Get(s); err != nil {
                                //fmt.Fprintf(stderr, "selection: %v: %v\n", p, err)
                        }
                }
        } else /*if o == nil*/ {
                err = fmt.Errorf("selection.value: nil object `%s`", p.String())
        }
        return
}
func (p *selection) Strval() (s string, err error) {
        var v Value
        if v, err = p.value(); err != nil {
                // sth's wrong
        } else if v != nil {
                s, err = v.Strval()
        } else if false {
                err = fmt.Errorf("selection.strval: `%s` is nil", p.String())
        }
        return
}
func (p *selection) Integer() (int64, error) {
        if s, err := p.Strval(); err == nil {
                return strconv.ParseInt(s, 10, 64)
        } else {
                return 0, err
        }
}
func (p *selection) Float() (float64, error) {
        if s, err := p.Strval(); err == nil {
                return strconv.ParseFloat(s, 64)
        } else {
                return 0, err
        }
}
func (p *selection) refs(v Value) bool { return p.o.refs(v) || p.s.refs(v) }
func (p *selection) closured() bool { return p.o.closured() || p.s.closured() }
func (p *selection) expand(w expandwhat) (res Value, err error) {
        var o, s Value
        if p.o != nil {
                if o, err = p.o.expand(w); err != nil {
                        return
                } else if o == nil { o = p.o }
        }
        if p.s != nil {
                if s, err = p.s.expand(w); err != nil {
                        return
                } else if s == nil { s = p.s }
        }
        if o != p.o || s != p.s {
                res = &selection{p.trivial,p.t,o,s}
        } else {
                res = p
        }
        return
}
func (p *selection) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }

        var v Value
        if v, err = p.value(); err != nil {
                // sth's wrong
        } else if v == nil {
                err = fmt.Errorf("`%v` is nil", p)
        } else {
                err = pc.traverse(v)
        }
        return
}
func (p *selection) after(v Value) (after bool, err error) {
        // TODO: ...
        return
}
func (p *selection) cmp(v Value) (res cmpres) {
        if a, ok := v.(*selection); ok {
                assert(ok, "value is not selection")
                if p.o.cmp(a.o) == cmpEqual && p.s.cmp(a.s) == cmpEqual {
                        if p.t == a.t { res = cmpEqual }
                }
        }
        return
}

type partialMatcher interface {
        partialMatch(i interface{}) (result string, rest, stems []string, err error)
}

// TODO: endingMatcher is not implemented (e.g. $(trim-suffix .%, a.xxx b.xxx))
type endingMatcher interface {
        endingMatch(i interface{}) (result string, rest, stems []string, err error)
}

// Pattern
type Pattern interface {
        Value
        match(i interface{}) (s string, stems []string, err error)
        stencil(stems []string) (s string, rest []string, err error)
}

/*
func (p *pattern) concrete(patent *RuleEntry, target, stem string) (entry *RuleEntry, err error) {
        entry = new(RuleEntry)
        *entry = *patent // Copy the entry object bits

        var project = mostDerived()
        if project.isFileName(target) {
                var file = project.searchFile(target)
                if file == nil { // stat non-existed file
                        file = stat(target, "", project.absPath, nil)
                }
                assert(file != nil, "`%s` nil file", target)
                entry.target = file
        } else {
                entry.target = &String{ target }
        }
        return
}
*/

// PercPattern represents percent pattern expressions (e.g. '%.o')
type PercPattern struct {
        trivial // TODO: supporting multiple %: foo%bar%xxx
        Prefix Value
        Suffix Value
}
func (p *PercPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *PercPattern) elemstr(o Object, k elemkind) (s string) {
        s = elementString(o, p.Prefix, k) + `%`
        s += elementString(o, p.Suffix, k)
        return
}
func (p *PercPattern) String() string { return p.elemstr(nil, 0) }
func (p *PercPattern) Strval() (s string, err error) {
        if p.Prefix != nil {
                var v string
                if v, err = p.Prefix.Strval(); err == nil {
                        s = v
                } else {
                        return
                }
        }
        s += "%"
        if p.Suffix != nil {
                var v string
                if v, err = p.Suffix.Strval(); err == nil {
                        s += v
                } else {
                        return
                }
        }
        return
}
func (p *PercPattern) match(i interface{}) (result string, stems []string, err error) {
        var s string
        switch t := i.(type) {
        case string: s = t
        case *File: s = t.name
        case *filestub: s = t.name
        case Value:
                s, err = t.Strval()
                if err != nil { return }
        default:
                unreachable(fmt.Sprintf("perc.match: %T %v", i, i))
        }

        var prefix string
        if p.Prefix == nil {
                // ...
        } else if !isNone(p.Prefix) {
                prefix, err = p.Prefix.Strval()
                if err != nil { return }
                if !strings.HasPrefix(s, prefix) {
                        return
                }
        }

        switch t := p.Suffix.(type) {
        case *None:
                if a, b := len(prefix), len(s); a < b {
                        stems = []string{ s[a:] }
                        result = s
                }                
        case Pattern:
                if a, b := len(prefix), len(s); a < b {
                        var res string
                        res, stems, err = t.match(s[a:])
                        if res != "" && len(stems) > 0 {
                                result = s
                        }
                }
        default:
                var suffix string
                suffix, err = t.Strval()
                if err != nil { break }
                if !strings.HasSuffix(s, suffix) { break }
                if a, b := len(prefix), len(s)-len(suffix); a < b {
                        stems = []string{ s[a:b] }
                        result = s
                }
        }
        return
}
func (p *PercPattern) stencil(stems []string) (s string, rest []string, err error) {
        // FIXME: the prefix is possible to be Glob, Regexp, etc.
        if !isNone(p.Prefix) {
                s, err = p.Prefix.Strval()
                if err != nil { return }
        }

        var v string
        if isNone(p.Suffix) {
                s += stems[0] + v
                rest = stems[1:]
        } else if pp, ok := p.Suffix.(*PercPattern); ok {
                // Special cases like '%%...' use only one stem,
                // other cases like '%xxx%...' use multiple stems.
                if !isNone(pp.Prefix) {
                        s += stems[0]
                        stems = stems[1:]
                }
                var ss string
                ss, rest, err = pp.stencil(stems)
                if err == nil { s += ss }
        } else if pp, ok := p.Suffix.(Pattern); ok {
                var ss string
                ss, rest, err = pp.stencil(stems)
                if err == nil { s += ss }
        } else if v, err = p.Suffix.Strval(); err == nil {
                s += stems[0] + v
                rest = stems[1:]
        }
        return
}
/*
func (p *PercPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        var target string
        if target, err = p.stencil(stem); err == nil {
                entry, err = p.pattern.concrete(patent, target, stem)
        }
        return
}
*/
func (p *PercPattern) refs(v Value) bool { return p.Prefix.refs(v) || p.Suffix.refs(v) }
func (p *PercPattern) closured() bool { return p.Prefix.closured() || p.Suffix.closured() }
func (p *PercPattern) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        if pc.stems == nil {
                err = fmt.Errorf("empty stem (%s)", p)
                return
        }

        var target string
        var rest []string
        if target, rest, err = p.stencil(pc.stems); err != nil { return }
        if false && len(rest) > 0 { panic("FIXME: unhandled stems") }
        if err = pc.tarverseTarget(target); err != nil {
                err = patternPrepareError{err}
        }
        return
}
func (p *PercPattern) cmp(v Value) (res cmpres) {
        if a, ok := v.(*PercPattern); ok {
                assert(ok, "value is not PercPattern")
                if p.Prefix.cmp(a.Prefix) == cmpEqual {
                        if p.Suffix.cmp(a.Suffix) == cmpEqual {
                                res = cmpEqual
                        }
                }
        }
        return
}

// GlobPattern represents glob pattern expressions (e.g. '*.o', '[a-z].o', 'a?a.o')
// 
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-Separator characters
//		'?'         matches any single non-Separator character
//		'[' [ '^' ] { character-range } ']'
//		            character class (must be non-empty)
//		c           matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c           matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
type GlobPattern struct {
        trivial
        Components []Value
}
func (p *GlobPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *GlobPattern) elemstr(o Object, k elemkind) (s string) {
        for _, comp := range p.Components {
                s += elementString(o, comp, k)
        }
        return
}
func (p *GlobPattern) String() (s string) { return p.elemstr(nil, 0) }
func (p *GlobPattern) Strval() (s string, err error) {
        for _, comp := range p.Components {
                var v string
                if v, err = comp.Strval(); err != nil {
                        return
                }
                s += v
        }
        return
}
func (p *GlobPattern) match(i interface{}) (result string, stems []string, err error) {
        var pat, s string
        switch t := i.(type) {
        case string: s = t
        case *File: s = t.name
        case *filestub: s = t.name
        case Value:
                s, err = t.Strval()
                if err != nil { return }
        default:
                unreachable("glob.match: %T %v", i, i)
        }
        if pat, err = p.Strval(); err == nil {
                var matched bool
                matched, err = filepath.Match(pat, s)
                if matched { result = s }
        }
        // FIXME: calculate stems from matching
        return
}
func (p *GlobPattern) stencil(stems []string) (s string, rest []string, err error) {
        unreachable(fmt.Sprintf("Unimplemented GlobPattern stencil %v (stems=%v)", p, stems))
        return
}
/*
func (p *GlobPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        var target string
        if target, err = p.stencil(stem); err == nil {
                entry, err = p.pattern.concrete(patent, target, stem)
        }
        return
}
*/
func (p *GlobPattern) refs(v Value) (res bool) {
        for _, comp := range p.Components {
                if res = comp.refs(v); res { break }
        }
        return
}
func (p *GlobPattern) closured() (res bool) {
        for _, comp := range p.Components {
                if res = comp.closured(); res { break }
        }
        return
}
func (p *GlobPattern) traverse(pc *traversal) (err error) {
        if optionTraceTraversal { defer un(tt(pc, p)) }
        if pc.stems == nil {
                err = fmt.Errorf("empty stem (%s)", p)
                return
        }

        var rest []string
        var target string
        if target, rest, err = p.stencil(pc.stems); err != nil { return }
        if len(rest) > 0 { panic("FIXME: unhandled stems") }
        if err = pc.tarverseTarget(target); err != nil {
                err = patternPrepareError{err}
        }
        return
}
func (p *GlobPattern) cmp(v Value) (res cmpres) {
        if a, ok := v.(*GlobPattern); ok {
                assert(ok, "value is not GlobPattern")
                if len(p.Components) == len(a.Components) {
                        for i, c := range p.Components {
                                if c.cmp(a.Components[i]) != cmpEqual {
                                        return
                                }
                        }
                        res = cmpEqual
                }
        }
        return
}

// TODO: implement regexp pattern
type RegexpPattern struct { trivial }
func (p *RegexpPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *RegexpPattern) String() string { return "{RegexpPattern}" }
func (p *RegexpPattern) Strval() (s string, err error) { return "", nil }
func (p *RegexpPattern) match(i interface{}) (result string, stems []string, err error) {
        unreachable("regexp.match: %T %v", i, i)
        return
}
func (p *RegexpPattern) stencil(stems []string) (s string, rest []string, err error) {
        unreachable("regexp.stencil: %v", stems)
        return
}
func (p *RegexpPattern) cmp(v Value) (res cmpres) {
        if a, ok := v.(*RegexpPattern); ok {
                assert(ok, "value is not RegexpPattern")
                if a != nil { /* FIXME: ... */ }
        }
        return
}

func NewRegexpPattern() Pattern {
        return &RegexpPattern{}
}

type Valuer interface {
        Value() Value
}

type Caller interface {
        Call(pos Position, args... Value) (Value, error)
}

type Executer interface {
        Execute(pos Position, a... Value) (result []Value, err error)
}

type Positioner interface {
        Position() Position
}

type Namer interface {
        Name() string
}

type Scoper interface {
        Scope() *Scope
}

type NameScoper interface {
        Namer
        Scoper
}

type namescoper struct {
        name string
        scope *Scope
}

func (ns *namescoper) Name() string { return ns.name }
func (ns *namescoper) Scope() *Scope { return ns.scope }
func NameScope(name string, scope *Scope) NameScoper {
        return &namescoper{ name, scope }
}

// Reveal reveals delegated component and Valuer recursively.
func Reveal(values ...Value) (res []Value, err error) {
        for _, v := range values {
                //if v, err = Reveal(v); err != nil { break }
                if v, err = v.expand(expandDelegate); err != nil { break }
                if v != nil { res = append(res, v) }
        }
        return
}

// Disclose expands closures to normal value recursively.
func Disclose(values ...Value) (res []Value, err error) {
        for _, v := range values {
                if v, err = v.expand(expandClosure); err != nil { break }
                if v != nil { res = append(res, v) }
        }
        return
}

func values(args... interface{}) (elems []Value) {
        for _, a := range args {
                if v, ok := a.(Value); ok {
                        elems = append(elems, v)
                } else {
                        unreachable()
                }
        }
        return
}

// Merge combines lists recursively into one list. Previously called Join.
func merge(args... Value) (elems []Value) {
        for _, arg := range args {
                if l, _ := arg.(*List); l != nil {
                        elems = append(elems, merge(l.Elems...)...)
                } else {
                        elems = append(elems, arg)
                }
        }
        return
}

func mergeresult(res []Value, err error) ([]Value, error) {
        if err == nil { res = merge(res...) }
        return res, err
}

func trueVal(v Value, res bool) bool {
        if v != nil { res = v.True() }
        return res
}

func intVal(v Value, res int) int {
        if v != nil {
                n, _ := v.Integer()
                res = int(n)
        }
        return res
}

var expanddepth int64 = 0
func expandall(w expandwhat, values ...Value) (res []Value, num int, err error) {
        defer func(i int64) { expanddepth = i } (expanddepth)
        if expanddepth += 1; expanddepth > 128 {
                err = fmt.Errorf("exceeds maximum expand depth")
                return
        }

        var v Value
        for _, elem := range values {
                if elem == nil {
                        res = append(res, &Nil{})
                } else if v, err = elem.expand(w); err == nil {
                        if v != elem { num += 1 }
                        res = append(res, v)
                } else {
                        break //res = append(res, elem)
                }
        }
        return
}

func ExpandAll(values ...Value) (res []Value, err error) {
        if res, _, err = expandall(expandAll, values...); err == nil {
                // second expand to ensure having real value
                res, _, err = expandall(expandAll, res...)
        }
        return
}

func Refs(a Value, v Value) bool { return a.refs(v) }

func Scalar(v Value) (res Value) {
        if l, o := v.(*List); l != nil && o && l.Len() > 0 {
                res = Scalar(l.Elems[0])
        } else {
                res = v
        }
        return
}

func EscapeChar(s string) string {
        switch s {
        case "a":  s = "\a"
        case "b":  s = "\b"
        case "f":  s = "\f"
        case "n":  s = "\n"
        case "r":  s = "\r"
        case "t":  s = "\t"
        case "v":  s = "\v"
        case "\\": s = "\\"
        case "$":  s = "$"
        case "&":  s = "&"
        default:   s = "\\" + s // give back the '\' character
        }
        return s
}

func MakeAnswer(pos Position, v bool) (res Value) {
        if v {
                res = &boolean{trivial{pos},true}
        } else {
                res = &boolean{trivial{pos},false}
        }
        return
}
func MakeBoolean(pos Position, v bool) (res Value) {
        if v {
                res = &answer{trivial{pos},true}
        } else {
                res = &answer{trivial{pos},false}
        }
        return
}
func MakeBin(pos Position, i int64) *Bin { return &Bin{integer{trivial{pos},i}} }
func MakeOct(pos Position, i int64) *Oct { return &Oct{integer{trivial{pos},i}} }
func MakeInt(pos Position, i int64) *Int { return &Int{integer{trivial{pos},i}} }
func MakeHex(pos Position, i int64) *Hex { return &Hex{integer{trivial{pos},i}} }
func MakeFloat(pos Position, f float64) *Float { return &Float{trivial{pos},f} }
func MakeDate(pos Position, s time.Time) *Date { return &Date{DateTime{trivial{pos},s}} }
func MakeTime(pos Position, t time.Time) *Time { return &Time{DateTime{trivial{pos},t}} }
func MakeString(pos Position, s string) *String { return &String{trivial{pos},s} }
func MakeURL(pos Position, s *url.URL) *URL {
        var host, port string
        v := strings.Split(s.Host, ":")
        if len(v) == 1 { host = v[0] }
        if len(v) == 2 { host, port = v[0], v[1] }
        var password Value
        if t, ok := s.User.Password(); ok {password = &String{trivial{pos},t}}
        return &URL{ // FIXME: calculate component positions
                trivial: trivial{pos},
                Scheme: &String{trivial{pos},s.Scheme},
                Username: &String{trivial{pos},s.User.Username()},
                Password: password,
                Host: &String{trivial{pos},host},
                Port: &String{trivial{pos},port},
                Path: &String{trivial{pos},s.Path},
                Query: &String{trivial{pos},s.RawQuery},
                Fragment: &String{trivial{pos},s.Fragment},
        }
}
func MakeBarecomp(pos Position, elems... Value) *Barecomp { return &Barecomp{trivial{pos},elements{elems}} }
func MakeCompound(pos Position, elems... Value) *Compound { return &Compound{trivial{pos},elements{elems}} }
func MakeList(pos Position, elems... Value) *List { return &List{elements{elems}} }
func MakeGroup(pos Position, elems... Value) (v *Group) { return &Group{trivial{pos},List{elements{elems}}} }
func MakeGlobMeta(pos Position, tok token.Token) *GlobMeta { return &GlobMeta{trivial{pos},tok} }
func MakeGlobRange(pos Position, v Value) *GlobRange { return &GlobRange{trivial{pos},v} }
func MakePath(pos Position, segments... Value) (v *Path) { return &Path{trivial{pos},elements{segments}/*, nil*/} }
func MakePathSeg(pos Position, ch rune) *PathSeg { return &PathSeg{trivial{pos},ch} }
func MakePathStr(pos Position, str string) (v *Path) {
        var segments []Value
        for _, s := range strings.Split(str, PathSep) {
                // TODO: calculate position of each segment
                segments = append(segments, &Bareword{trivial{pos},s})
        }
        return MakePath(pos, segments...)
}
func MakePair(k, v Value) (p *Pair) {
        p = &Pair{nil, nil}
        p.SetKey(k)
        p.SetValue(v)
        return
}
func MakePercPattern(prefix, suffix Value) Pattern {
        if prefix == nil { prefix = &None{} }
        if suffix == nil { suffix = &None{} }
        return &PercPattern{
                Prefix: prefix,
                Suffix: suffix,
        }
}
func MakeGlobPattern(components... Value) Pattern {
        return &GlobPattern{Components:components}
}
func MakeDelegate(pos Position, tok token.Token, obj Object, args... Value) Value {
        return &delegate{trivial{pos},closuredelegate{tok,obj,args}}
}
func MakeClosure(pos Position, tok token.Token, obj Object, args... Value) Value {
        if obj == nil { panic("closure of nil") }
        return &closure{trivial{pos},closuredelegate{tok,obj,args}}
}
func MakeListOrScalar(elems []Value) (res Value) {
        if x := len(elems); x > 1 {
                res = &List{elements{elems}}
        } else if x == 1 {
                res = elems[0]
        } else {
                res = &None{trivial{/*pos*/}}
        }
        return
}

func Make(pos Position, in interface{}) (out Value) {
        switch v := in.(type) {
        case int:       out = MakeInt(pos,int64(v))
        case int32:     out = MakeInt(pos,int64(v))
        case int64:     out = MakeInt(pos,v)
        case float32:   out = MakeFloat(pos,float64(v))
        case float64:   out = MakeFloat(pos,v)
        case string:    out = &String{trivial{pos},v}
        case time.Time: out = &DateTime{trivial{pos},v} // FIXME: NewDate, NewTime
        case Value:     out = v
        default:        out = &Any{in} // TODO: position for any
        }
        return
}

func MakeAll(pos Position, in... interface{}) (out []Value) {
        for _, v := range in {
                // TODO: position for each element
                out = append(out, Make(pos,v))
        }
        return
}

func ParseBin(pos Position, s string) *Bin {
        if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 2, 64); e == nil {
                return MakeBin(pos,i)
        } else {
                panic(e)
        }
}

func ParseOct(pos Position, s string) *Oct {
        if strings.HasPrefix(s, "0") {
                s = s[1:]
        }
        if i, e := strconv.ParseInt(s, 8, 64); e == nil {
                return MakeOct(pos,i)
        } else {
                panic(e)
        }
}

func ParseInt(pos Position, s string) *Int {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                return MakeInt(pos,i)
        } else {
                panic(e)
        }
}

func ParseHex(pos Position, s string) *Hex {
        if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 16, 64); e == nil {
                return MakeHex(pos,i)
        } else {
                panic(e)
        }
}

func ParseFloat(pos Position, s string) *Float {
        if f, e := strconv.ParseFloat(strings.Replace(s, "_", "", -1), 64); e == nil {
                return MakeFloat(pos,f)
        } else {
                panic(e)
        }
}

func ParseDate(pos Position, s string) *Date {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                return MakeDate(pos,t)
        } else {
                panic(e)
        }
}

func ParseTime(pos Position, s string) *Time {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                return MakeTime(pos,t)
        } else {
                panic(e)
        }
}

func ParseURL(pos Position, s string) *URL {
        if u, e := url.Parse(s); e == nil {
                return MakeURL(pos,u)
        } else {
                panic(e)
        }
}
