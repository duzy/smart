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
        enable_statcache = true
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
        // Type returns the underlying type of the value.
        Type() Type

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

        cmp(v Value) cmpres

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

type comparer struct {
        program *Program
        target Value
        updated []*updatedtarget // found updated dependencies
        nocomp bool // just checking existence
        nomiss bool // all file dependencies must exist
        level int // compare/trace level
}

type dependcomparable interface {
        // Compare target with the prerequisite.
        dependcompare(c *comparer) error
}

//type comparable interface {
//        compare(c *comparer, d dependcomparable) error
//}

func comptrace(c *comparer, v Value) *comparer {
        t := c.target
        a := fmt.Sprintf("%s{%v}", t.Type(), t)
        b := fmt.Sprintf("%s{%v}", v.Type(), v)
        c.trace(a, ":", b, "(")
        c.level += 1
        return c
}

func compun(c *comparer) {
        c.level -= 1
        c.trace(")")
}

func newcompariation(prog *Program, target Value) (c *comparer, err error) {
        if target, err = target.expand(expandDelegate); err != nil { return }
        if target == nil || target.Type() == NoneType {
                err = break_bad(prog.position, "comparing no target")
        } else if /*tar, ok := target.(comparable); ok ||*/ true {
                var l int
                if len(prog.callers) > 0 {
                        l = prog.callers[0].level
                }
                c = &comparer{ prog, target, /*tar,*/ nil, false, true, l }
        } else {
                err = fmt.Errorf("%s '%s' is incomparable target", target.Type(), target)
        }
        return
}

func (c *comparer) trace(a ...interface{}) {
        printIndentDots(c.level, a...)
}

func (c *comparer) Compare(pos Position, value interface{}) (err error) {
        if optionTraceCompare {
                s := fmt.Sprintf("%s{%s}", c.target.Type(), c.target)
                c.trace("compare:", s, ":", value, "(")
                c.level += 1
                defer func() {
                        if err != nil {
                                if br, ok := err.(*breaker); ok {
                                        switch br.what {
                                        case breakBad: c.trace("bad:", err)
                                        case breakGood: //c.trace("good:", err)
                                        case breakUpdates: //c.trace("updated:", br.updated)
                                        }
                                } else {
                                        c.trace("error:", err)
                                }
                        }
                        compun(c)
                } ()
        }
        if f, ok := c.target.(*File); ok && f.info == nil {
                err = break_updates(pos, newUpdatedTarget(c.target, nil))
                return
        }
        if v := reflect.ValueOf(value); v.Kind() == reflect.Slice {
                for i := 0; i < v.Len(); i++ {
                        var dep = v.Index(i).Interface()
                        if err = c.compareDepend(dep); err == nil {
                                continue
                        } else if optionTraceCompare {
                                c.trace("error:", err)
                                break
                        }
                }
        } else {
                err = c.compareDepend(value)
        }
        if err != nil {
                // hmm...
        } else if c.updated == nil {
                err = break_good(pos, "no need to update")
        } else {
                target := newUpdatedTarget(c.target, c.updated)
                err = break_updates(pos, target)
        }
        return
}

func (c *comparer) compareDepend(value interface{}) (err error) {
        if dep, ok := value.(dependcomparable); ok {
                err = dep.dependcompare(c)
        } else {
                err = fmt.Errorf("'%v' is not dependcomparable", value)
        }
        return
}

func (c *comparer) compareStatDepend(d Value, ds string, di os.FileInfo) (err error) {
        var tt, dt time.Time

        if ds == "" {
                err = break_bad(c.program.position, "'%v' unknown depend", d)
                return
        } else if t, ok := c.program.globe.timestamps[ds]; ok && !t.IsZero() {
                dt = t
        } else if di != nil {
                dt = di.ModTime()
        } else if f, ok := d.(*File); ok && f.info != nil {
                d, ds, dt = f, f.FullName(), f.info.ModTime()
        } else {
                for _, project := range c.program.pc.related {
                        if t := project.searchFile(ds); t != nil {
                                d, ds, dt = t, t.FullName(), t.info.ModTime()
                                if f != nil { *f = *t } // replace the file 
                                break
                        }
                }
        }

        var ts string
        if ts, err = c.target.Strval(); err != nil {
                return
        } else if ts == "" {
                err = break_bad(c.program.position, "'%v' unknown target", c.target)
                return
        } else if t, ok := c.program.globe.timestamps[ts]; ok && !t.IsZero() {
                tt = t
        } else if f, ok := c.target.(*File); ok && f.info != nil {
                ts, tt = f.FullName(), f.info.ModTime()
        } else {
                for _, project := range c.program.pc.related {
                        if t := project.searchFile(ts); t != nil {
                                ts, tt = t.FullName(), t.info.ModTime()
                                if f != nil { *f = *t } // replace the file
                                break
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
                        c.trace("compare:", c.target, d, '\t', '\t', tt, ":", dt)
                }
        }

        if tt.IsZero() {
                err = break_bad(c.program.position, "%s '%v' is missing", c.target.Type(), c.target)
        } else if dt.IsZero() || dt.After(tt) {
                c.updated = append(c.updated, newUpdatedTarget(d, nil))

                // Update timestamps to depended file, so that
                // further updates can happen.
                if !dt.IsZero() {
                        // FIXME: set file ModTime instead of using the
                        //        timestamps, it may cause some targets
                        //        updated multiple times if target is
                        //        compared with different deps.
                        c.program.globe.timestamps[ts] = dt
                        c.program.globe.timestamps[ds] = dt
                }
        } else if true {
                // Just save the timestamps to optimize further stats.
                if !tt.IsZero() { c.program.globe.timestamps[ts] = tt }
                if !dt.IsZero() { c.program.globe.timestamps[ds] = dt }
        }
        return
}

// State machine:
//
//    default ---→ compare ---→ update --→ interpret
//             |      |
//             |      +--> <done>
//             |
//             +-> interpret
//
type traversemode int
const (
        defaultMode traversemode = iota
        compareMode // compared but no updated targets
        updateMode // work to update targets
)

func (m traversemode) String() (s string) {
        s = m.name()
        return
}

func (m traversemode) name() (s string) {
        switch m {
        case defaultMode: s = "default"
        case compareMode: s = "compare"
        case updateMode: s = "update"
        default: s = "unknown"
        }
        return
}

type preparecontext struct {
        mode traversemode
        group *sync.WaitGroup
        calleeErrors []error
        entry *RuleEntry // caller entry (target)
        //visitInsteadUpdate bool // target don't really need to update
        args, arguments []Value // target and argumented prerequisite args
        targets []Value // prerequisite targets ($^ $<)
        updated []*updatedtarget // prerequisites newer than the target (from comparer) ($?)
        derived *Project // the most derived project
        related []*Project // the related projects in the context
        stem string // set by StemmedEntry
        level int // prepare/trace level
}

// preparer prepares prerequisites of targets.
type preparer struct {
        program *Program
        preparecontext
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

        preModifiers, postModifiers []*modifier
        interpreted []interpreter
}

//type preparable interface {
type prerequisite interface {
        prepare(pc *preparer) error
}

func preptrace(pc *preparer, i Value) *preparer {
        // Note that pc.args and pc.arguments are different, they're
        // target execution args and argumented-prerequisite args.
        var a string
        if t := pc.entry.target; len(pc.args) > 0 {
                a = fmt.Sprintf("%s{%s%s}", t.Type(), t, pc.args)
        } else {
                a = fmt.Sprintf("%s{%v}", t.Type(), t)
        }
        b := fmt.Sprintf("%s{%v}", i.Type(), i)
        pc.trace(a, ":", b, "(")
        pc.level += 1
        return pc
}

func prepun(pc *preparer) {
        pc.level -= 1
        pc.trace(")")
}

func (pc *preparer) trace(a ...interface{}) {
        printIndentDots(pc.level, a...)
}

func (pc *preparer) tracef(s string, a ...interface{}) {
        printIndentDots(pc.level, fmt.Sprintf(s, a...))
}

func (pc *preparer) addNotExistedTarget1(target Value) {
        if target != nil && target.Type() != NoneType {
                switch t := target.(type) {
                //case *File:
                case *Path:
                        if t.File != nil {
                                target = t.File
                        }
                default:
                        /*
                        var s string
                        if s, err = target.Strval(); err != nil {
                                return
                        } else if s == "" {
                                panic(fmt.Sprintf("Empty target! (%s `%v`)", target.Type(), target))
                        } else if file := prog.project.searchFile(s); file != nil {
                                target = file
                        }
                        */
                }
                for _, t := range pc.targets {
                        if t == target { return }
                        if t.cmp(target) == cmpEqual { return }
                }
                pc.targets = append(pc.targets, target)
        }
}

func (pc *preparer) addNotExistedTargets(targets ...Value) {
        var valid []Value
        for _, elem := range targets {
                if elem != nil && elem.Type() != NoneType {
                        valid = append(valid, elem)
                }
        }
        for _, elem := range merge(valid...) {
                pc.addNotExistedTarget1(elem)
        }
}

func (pc *preparer) traverseAll(value interface{}) (err error) {
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

func (pc *preparer) traverse(value interface{}) (err error) {
        var pos = token.Position(pc.entry.Position)
        if value == nil {
                err = scanner.Errorf(pos, "updating nil prerequisite")
        } else if p, ok := value.(prerequisite); ok {
                if p != nil {
                        err = pc.checkUpdates(p.prepare(pc))
                } else { // this could happen
                        err = scanner.Errorf(pos, "updating nil prerequisite")
                }
        } else {
                err = scanner.Errorf(pos, "'%v' is not prerequisite", value)
        }
        return
}

func (pc *preparer) updateErrs(errs scanner.Errors, err error) (scanner.Errors, error, bool) {
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
                switch e := scanner.WrapError(pos, err).(type) {
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

func (pc *preparer) updateFile(file *File) (err error) {
        var ( errs scanner.Errors ; done bool )
        for _, project := range pc.related {
                if _, err = project.updateFile(pc, file); err == nil { break }
                if errs, err, done = pc.updateErrs(errs, err); done { break }
        }
        if errs != nil && err != nil { err = errs }
        return
}

func (pc *preparer) updateTargetErrs(target string) (errs scanner.Errors) {
        for _, project := range pc.related {
                var done bool
                var err = project.updateTarget(pc, target)
                if err == nil { /* Good! */ break }
                if errs, err, done = pc.updateErrs(errs, err); done { break }
        }
        return
}

func (pc *preparer) updateTarget(target string) (err error) {
        if errs := pc.updateTargetErrs(target); len(errs) > 0 {
                err = errs
        }
        return
}

func (pc *preparer) updateTargetValue(value Value) (err error) {
        var s string
        if s, err = value.Strval(); err == nil {
                 err = pc.updateTarget(s)
        }
        return
}

func (pc *preparer) execute(entry *RuleEntry, prog *Program) (err error) {
        // Push the context to the program, so that patterns will work.
        defer func(a []*preparecontext) { prog.callers = a } (prog.callers)
        prog.callers = append([]*preparecontext{&pc.preparecontext}, prog.callers...)

        // Execute the updating program.
        var res Value
        if res, err = prog.Execute(entry, pc.arguments); err != nil {
                //if optionTracePrepare { pc.tracef("%s: %s", entry, err) }
                if br, ok := err.(*breaker); ok {
                        switch br.what {
                        case breakBad:
                                fmt.Fprintf(stderr, "%s: %v\n", prog.position, err)
                        }
                }
        }

        target, _ := prog.scope.Lookup("@").(*Def).Call(entry.Position)
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

type Argumented struct {
        Val Value
        Args []Value
}
func (p *Argumented) refs(v Value) bool {
        if p.Val.refs(v) { return true }
        for _, a := range p.Args {
                if a.refs(v) { return true }
        }
        return false
}
func (p *Argumented) closured() bool {
        if p.Val.closured() { return true }
        for _, a := range p.Args {
                if a.closured() { return true }
        }
        return false
}
func (p *Argumented) expand(w expandwhat) (res Value, err error) {
        var (v Value; args []Value)
        if v, err = p.Val.expand(w); err == nil {
                if v != p.Val {
                        var num int
                        args, num, err = expandall(w, p.Args...)
                        if err == nil && (num > 0 || v != p.Val) {
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
        if v.Type() == ArgumentedType {
                a, ok := v.(*Argumented)
                assert(ok, "value is not Argumented")
                if res = p.Val.cmp(a.Val); res == cmpEqual {
                        // FIXME: check p.Args, a.Args too
                }
        }
        return
}
func (p *Argumented) Type() Type { return ArgumentedType }
func (p *Argumented) True() (res bool) {
        if p.Val != nil {
                res = p.Val.True()
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
        for i, a := range p.Args {
                if i > 0 { s += "," }
                s += elementString(o, a, k)
        }
        s = fmt.Sprintf("%s(%s)", elementString(o, p.Val, k), s)
        return
}
func (p *Argumented) String() (s string) { return p.elemstr(nil, 0) }
func (p *Argumented) Strval() (s string, err error) {
        if s, err = p.Val.Strval(); err != nil {
                return
        }
        s += "("
        for i, a := range p.Args {
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

func (p *Argumented) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        if false {
                pc.arguments, err = mergeresult(ExpandAll(p.Args...))
        } else {
                //<!IMPORTANT! - Don't merge-expand arguments here!
                // Arguments should be passed to Execute as it's
                // represented.
                pc.arguments = p.Args
        }
        if err == nil { err = pc.traverse(p.Val) }
        return
}

type None struct {}
func (_ *None) refs(_ Value) bool { return false }
func (_ *None) closured() bool { return false }
func (p *None) expand(_ expandwhat) (Value, error) { return p, nil }
func (_ *None) cmp(v Value) (res cmpres) { 
        if v.Type() == NoneType { res = cmpEqual }
        return
}
func (_ *None) Type() Type { return NoneType }
func (_ *None) True() bool { return false }
func (_ *None) Integer() (int64, error) { return 0, nil }
func (_ *None) Float() (float64, error) { return 0, nil }
func (p *None) String() (s string) { return }
func (p *None) Strval() (s string, err error) { return }
func (p *None) prepare(pc *preparer) (err error) { return }
func (p *None) dependcompare(c *comparer) (err error) {
        if enable_assertions { assert(c.target != p, "self comparation") }
        return
}

type Nil struct { None }
func (p *Nil) cmp(v Value) (res cmpres) {
        if v.Type() == p.Type() {
                if _, ok := v.(*Nil); ok {
                        res = cmpEqual
                }
        }
        return
}

type ModifierBar struct { None }
func (p *ModifierBar) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *ModifierBar) Strval() (string, error) { return "|", nil }
func (p *ModifierBar) String() string { return "|" }
func (p *ModifierBar) cmp(v Value) (res cmpres) {
        if v.Type() == p.Type() {
                if _, ok := v.(*ModifierBar); ok {
                        res = cmpEqual
                }
        }
        return
}

// Any is used to box an arbitrary value
type Any struct { value interface{} }
func (p *Any) Type() Type { return AnyType }
func (p *Any) cmp(v Value) (res cmpres) {
        if v.Type() == AnyType {
                a, ok := v.(*Any)
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
func (p *Any) dependcompare(c *comparer) (err error) {
        if enable_assertions { assert(c.target != p, "self comparation") }
        if v, ok := p.value.(dependcomparable); ok {
                err = v.dependcompare(c)
        }
        return
}
func (p *Any) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        if p, ok := p.value.(prerequisite); ok {
                err = p.prepare(pc)
        }
        return 
}

// Boxing any value
func Box(v interface{}) (any *Any) {
        if any, _ = v.(*Any); any == nil {
                any = &Any{v}
        }
        return
}
func Unbox(any *Any) interface{} { return any.value }

type negative struct { x Value }
func (p *negative) refs(o Value) bool { return p.x.refs(o) }
func (p *negative) closured() bool { return p.x.closured() }
func (p *negative) expand(w expandwhat) (res Value, err error) {
        var v Value
        if v, err = p.x.expand(w); err != nil { return }
        if v == p.x { res = p } else { res = &negative{v} }
        return
}
func (p *negative) cmp(v Value) (res cmpres) {
        if v.Type() == NegativeType {
                a, ok := v.(*negative)
                assert(ok, "value is not negative")
                res = p.x.cmp(a.x)
        }
        return
}
func (p *negative) Type() Type { return NegativeType }
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
func (p *negative) dependcompare(c *comparer) (err error) {
        if optionTraceCompare { defer compun(comptrace(c, p)) }
        if enable_assertions { assert(c.target != p, "self comparation") }
        if p, ok := p.x.(dependcomparable); ok {
                err = p.dependcompare(c)
        }
        return
}
func (p *negative) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        if p, ok := p.x.(prerequisite); ok {
                err = p.prepare(pc)
        }
        return
}

func Negative(val Value) *negative { return &negative{val} }

type boolean struct { bool }
func (p *boolean) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *boolean) refs(_ Value) bool { return false }
func (p *boolean) closured() bool { return false }
func (p *boolean) Type() Type { return BooleanType }
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
func (p *boolean) prepare(pc *preparer) error { return nil }
func (p *boolean) cmp(v Value) (res cmpres) {
        if t := v.Type(); t == BooleanType {
                a, ok := v.(*boolean)
                assert(ok, "value is not boolean")
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpLess
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if t == AnswerType {
                a, ok := v.(*answer)
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

type answer struct { bool }
func (p *answer) refs(_ Value) bool { return false }
func (p *answer) closured() bool { return false }
func (p *answer) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *answer) Type() Type { return AnswerType }
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
func (p *answer) prepare(pc *preparer) error { return nil }
func (p *answer) cmp(v Value) (res cmpres) {
        if t := v.Type(); t == AnswerType {
                a, ok := v.(*answer)
                assert(ok, "value is not answer")
                if p.bool == a.bool {
                        res = cmpEqual
                } else if !p.bool && a.bool {
                        res = cmpLess
                } else if p.bool && !a.bool {
                        res = cmpGreater
                }
        } else if t == BooleanType {
                a, ok := v.(*boolean)
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

type integer struct { int64 }
func (p *integer) refs(_ Value) bool { return false }
func (p *integer) closured() bool { return false }
func (p *integer) Type() Type { return InvalidType }
func (p *integer) True() bool { return p.int64 != 0 }
func (p *integer) Integer() (int64, error) { return p.int64, nil }
func (p *integer) Float() (float64, error) { return float64(p.int64), nil }
func (p *integer) cmp(v Value) (res cmpres) {
        if t := v.Type(); t == BinType || t == OctType || t == IntType || t == HexType {
                i, e := v.Integer()
                assert(e == nil, "%s: %v", v.Type(), e)
                if p.int64 == i {
                        res = cmpEqual
                } else if p.int64 < i {
                        res = cmpLess
                } else if p.int64 > i {
                        res = cmpGreater
                }
        }
        return
}

type Bin struct { integer }
func (p *Bin) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Bin) Type() Type { return BinType }
func (p *Bin) String() string { return fmt.Sprintf("0b%s", strconv.FormatInt(int64(p.int64),2)) }
func (p *Bin) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),2), nil }

type Oct struct { integer }
func (p *Oct) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Oct) Type() Type { return OctType }
func (p *Oct) String() string {
        return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8))
}
func (p *Oct) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),8), nil }

type Int struct { integer }
func (p *Int) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Int) Type() Type { return IntType }
func (p *Int) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),10), nil }

type Hex struct { integer }
func (p *Hex) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Hex) Type() Type { return HexType }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *Hex) Strval() (string, error) { return strconv.FormatInt(int64(p.int64),16), nil }

const FloatEpsilon = 1e-15 /* 1e-16 */
type Float struct { float64 } // IEEE-754 64-bit binary floating-point
func (p *Float) refs(_ Value) bool { return false }
func (p *Float) closured() bool { return false }
func (p *Float) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Float) Type() Type { return FloatType }
func (p *Float) True() bool { return math.Abs(p.float64)-0 > FloatEpsilon }
func (p *Float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) Strval() (string, error) { return strconv.FormatFloat(float64(p.float64),'g', -1, 64), nil }
func (p *Float) Integer() (int64, error) { return int64(p.float64), nil }
func (p *Float) Float() (float64, error) { return p.float64, nil }
func (p *Float) cmp(v Value) (res cmpres) {
        if v.Type() == FloatType {
                f, e := v.Float()
                assert(e == nil, "%s: %v", v.Type(), e)
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

type DateTime struct { Value time.Time }
func (_ *DateTime) refs(_ Value) bool { return false }
func (_ *DateTime) closured() bool { return false }
func (p *DateTime) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *DateTime) Type() Type { return DateTimeType }
func (p *DateTime) True() bool { return !p.Value.IsZero() }
func (p *DateTime) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{DateTime '%s' !(%+v)}", s, e)
        }
}
func (p *DateTime) Strval() (string, error) { return time.Time(p.Value).Format("2006-01-02T15:04:05.999999999Z07:00"), nil } // time.RFC3339Nano
func (p *DateTime) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *DateTime) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *DateTime) cmp(v Value) (res cmpres) {
        var vt time.Time
        if t := v.Type(); t == DateTimeType {
                a, ok := v.(*DateTime)
                assert(ok, "value is not DateTime")
                vt = a.Value
        } else if t == DateType {
                a, ok := v.(*Date)
                assert(ok, "value is not Date")
                vt = a.Value
        } else if t == TimeType {
                a, ok := v.(*Time)
                assert(ok, "value is not Time")
                vt = a.Value
        } else {
                return
        }
        if p.Value.Equal(vt) {
                res = cmpEqual
        } else if p.Value.Before(vt) {
                res = cmpLess
        } else /*if p.Value.After(vt)*/ {
                res = cmpGreater
        }
        return
}

func ParseDateTime(s string) *DateTime {
        // time.RFC3339Nano
        if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
                return &DateTime{t}
        } else {
                panic(e)
        }
}

type Date struct { DateTime }
func (p *Date) Type() Type { return DateType }
func (p *Date) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Date '%s' !(%+v)}", s, e)
        }
}
func (p *Date) Strval() (string, error) { return time.Time(p.Value).Format("2006-01-02"), nil }
func (p *Date) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *Date) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

type Time struct { DateTime }
func (p *Time) Type() Type { return TimeType }
func (p *Time) String() string {
        if s, e := p.Strval(); e == nil {
                return s
        } else {
                return fmt.Sprintf("{Time '%s' !(%+v)}", s, e)
        }
}
func (p *Time) Strval() (string, error) { return time.Time(p.Value).Format("15:04:05.999999999Z07:00"), nil }
func (p *Time) Integer() (int64, error) { return p.Value.Unix(), nil }
func (p *Time) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//                └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//                     └<userinfo>─(@)┘      └(:)─<port>┘
type URL struct {
        Scheme Value
        Username Value
        Password Value
        Host Value
        Port Value
        Path Value
        Query Value
        Fragment Value
}
func (_ *URL) refs(_ Value) bool { return false }
func (_ *URL) closured() bool { return false }
func (p *URL) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *URL) Type() Type { return URLType }
func (p *URL) True() bool { return p.String() != "" }
func (p *URL) elemstr(o Object, k elemkind) (s string) {
        if s = elementString(o, p.Scheme, k); s == "" { return }
        if s += ":"; p.Host != nil && p.Host.Type() != NoneType {
                var host string
                if host = elementString(o, p.Host, k); host == "" { return }
                s += "//"
                if p.Username != nil && p.Username.Type() != NoneType {
                        var user string
                        if user = elementString(o, p.Username, k); user != "" {
                                s += user + "@"
                        }
                }
                s += host
                if p.Port != nil && p.Port.Type() != NoneType {
                        var port string
                        if port = elementString(o, p.Port, k); port != "" {
                                s += ":" + port
                        }
                }
        }
        if p.Path != nil && p.Path.Type() != NoneType {
                var path string
                if path = elementString(o, p.Path, k); path != "" {
                        //if !strings.HasPrefix(path, PathSep) { s += PathSep }
                        s += path
                }
        }
        if p.Query != nil && p.Query.Type() != NoneType {
                var query string
                if query = elementString(o, p.Query, k); query != "" {
                        s += "?" + query
                }
        }
        if p.Fragment != nil && p.Fragment.Type() != NoneType {
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
        if s += ":"; p.Host != nil && p.Host.Type() != NoneType {
                var host string
                if host, err = p.Host.Strval(); err != nil { return }
                s += "//"
                if p.Username != nil && p.Username.Type() != NoneType {
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
                if p.Port != nil && p.Port.Type() != NoneType {
                        var port string
                        if port, err = p.Port.Strval(); err != nil { return }
                        s += ":" + port
                }
        }
        if p.Path != nil && p.Path.Type() != NoneType {
                var path string
                if path, err = p.Path.Strval(); err != nil { return }
                //if !strings.HasPrefix(path, PathSep) { s += PathSep }
                s += path
        }
        if p.Query != nil && p.Query.Type() != NoneType {
                var query string
                if query, err = p.Query.Strval(); err != nil { return }
                s += "?" + query
        }
        if p.Fragment != nil && p.Fragment.Type() != NoneType {
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
        if v.Type() == URLType {
                a, ok := v.(*URL)
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

func (p *URL) Validate() (res *url.URL){
        panic(fmt.Sprintf("validate %s", p))
        return
}

type Raw struct { string }
func (_ *Raw) refs(_ Value) bool { return false }
func (_ *Raw) closured() bool { return false }
func (p *Raw) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Raw) Type() Type { return RawType }
func (p *Raw) True() bool { return p.string != "" }
func (p *Raw) String() string { return p.string }
func (p *Raw) Strval() (string, error) { return p.string, nil }
func (p *Raw) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Raw) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Raw) prepare(pc *preparer) error { return fmt.Errorf("preparing raw string") }
func (p *Raw) cmp(v Value) (res cmpres) {
        if v.Type() == RawType {
                a, ok := v.(*Raw)
                assert(ok, "value is not Raw")
                if p.string == a.string {
                        res = cmpEqual
                }
        }
        return
}

type String struct { string }
func (_ *String) refs(_ Value) bool { return false }
func (_ *String) closured() bool { return false }
func (p *String) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *String) Type() Type { return StringType }
func (p *String) True() bool { return p.string != "" }
func (p *String) elemstr(o Object, k elemkind) (s string) {
        if k&elemNoQuote == 0 { s = `'`+p.string+`'` } else { s = p.string }
        return
}
func (p *String) String() string { return p.elemstr(nil, 0) }
func (p *String) Strval() (string, error) { return p.string, nil }
func (p *String) Integer() (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *String) Float() (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *String) dependcompare(c *comparer) (err error) { return c.compareStatDepend(p, p.string, nil) }
func (p *String) prepare(pc *preparer) error {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
         return pc.updateTarget(p.string)
}
func (p *String) cmp(v Value) (res cmpres) {
        if v.Type() == StringType {
                a, ok := v.(*String)
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

type Bareword struct { string }
func (_ *Bareword) refs(_ Value) bool { return false }
func (_ *Bareword) closured() bool { return false }
func (p *Bareword) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *Bareword) Type() Type { return BarewordType }
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
func (p *Bareword) dependcompare(c *comparer) (err error) { return c.compareStatDepend(p, p.string, nil) }
func (p *Bareword) prepare(pc *preparer) error {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        return pc.updateTarget(p.string)
}
func (p *Bareword) cmp(v Value) (res cmpres) {
        if v.Type() == BarewordType {
                a, ok := v.(*Bareword)
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
func (p *elements) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *elements) Integer() (int64, error) {
        if n := len(p.Elems); n == 1 {
                // If there's only one element, treat it as a scalar.
                return p.Elems[0].Integer()
        } else {
                return int64(n), nil
        }
}
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
func (p *elements) ToBarecomp() *Barecomp { return &Barecomp{*p} }
func (p *elements) ToCompound() *Compound { return &Compound{*p} }
func (p *elements) ToList() *List { return &List{*p} }
func (p *elements) True() (t bool) {
        for _, elem := range p.Elems {
                if t = elem.True(); t { break }
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
                        other := elems[i]
                        if r := elem.cmp(other); r != cmpEqual {
                                return cmpUnknown
                        }
                }
                res = cmpEqual
        }
        return
}

type Barecomp struct { elements }
func (p *Barecomp) Type() Type { return BarecompType }
func (p *Barecomp) Strval() (s string, e error) {
        for _, elem := range p.Elems {
                var v string
                if v, e = elem.Strval(); e == nil {
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
func (p *Barecomp) String() (s string) { return p.elemstr(nil, 0) }

func (p *Barecomp) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Barecomp{ elements{ elems } }
                } else {
                        res = p
                }
        }
        return
}

func (p *Barecomp) dependcompare(c *comparer) (err error) {
        if ds, err := p.Strval(); err == nil {
                err =  c.compareStatDepend(p, ds, nil)
        }
        return
}

func (p *Barecomp) prepare(pc *preparer) error {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        return pc.updateTargetValue(p)
}

func (p *Barecomp) cmp(v Value) (res cmpres) {
        if v.Type() == BarecompType {
                a, ok := v.(*Barecomp)
                assert(ok, "value is not Barecomp")
                res = p.cmpElems(a.Elems)
        }
        return
}

type Barefile struct {
        Name Value
        File *File
}
func (p *Barefile) refs(v Value) bool { return p.Name.refs(v) }
func (p *Barefile) closured() bool { return p.Name.closured() }
func (p *Barefile) expand(w expandwhat) (res Value, err error) {
        var name Value
        if name, err = p.Name.expand(w); err == nil {
                if name != p.Name {
                        res = &Barefile{ name, p.File }
                } else {
                        res = p
                }
        }
        return
}
func (p *Barefile) Type() Type { return BarefileType }
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

func (p *Barefile) dependcompare(c *comparer) (err error) {
        if optionTraceCompare { defer compun(comptrace(c, p)) }
        if enable_assertions { assert(c.target != p, "self comparation") }
        if p.File != nil {
                err = p.File.dependcompare(c)
        } else if c.nomiss {
                err = break_bad(c.program.position, "no such file '%v'", p)
        }
        return
}

func (p *Barefile) prepare(pc *preparer) error {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        if p.File != nil {
                if s, e := p.Name.Strval(); e != nil {
                        return e
                } else if s != p.File.name {
                        // Fixes the case that '$@.o' is parsed and become '.o'.
                        p.File.name = s
                }
                return p.File.prepare(pc)
        } else {
                return pc.updateTargetValue(p)
        }
}

func (p *Barefile) cmp(v Value) (res cmpres) {
        if v.Type() == BarefileType {
                a, ok := v.(*Barefile)
                assert(ok, "value is not Barefile")
                // FIXME: check p.File.filebase == a.File.filebase first
                res = p.Name.cmp(a.Name)
        }
        return
}

type GlobMeta struct { token.Token }
func (p *GlobMeta) refs(o Value) bool { return false }
func (p *GlobMeta) closured() bool { return false }
func (p *GlobMeta) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *GlobMeta) Type() Type { return GlobType }
func (p *GlobMeta) True() bool { return false }
func (p *GlobMeta) String() string { return p.Token.String() }
func (p *GlobMeta) Strval() (string, error) { return p.Token.String(), nil }
func (p *GlobMeta) Integer() (int64, error) { return 0, nil }
func (p *GlobMeta) Float() (float64, error) { return 0, nil }
func (p *GlobMeta) cmp(v Value) (res cmpres) {
        if v.Type() == GlobType {
                a, ok := v.(*GlobMeta)
                assert(ok, "value is not GlobMeta")
                if p.Token == a.Token {
                        res = cmpEqual
                }
        }
        return
}

// `[a-b]`, `[abc]`, ...
// `a-b`, `abc`, `a$(var)c`, `a$(spaces)c`...
type GlobRange struct { Chars Value }
func (p *GlobRange) refs(v Value) bool { return p.Chars.refs(v) }
func (p *GlobRange) closured() bool { return p.Chars.closured() }
func (p *GlobRange) expand(w expandwhat) (Value, error) {
        if v, err := p.Chars.expand(w); err != nil {
                return nil, err
        } else if v != p.Chars {
                return &GlobRange{v}, nil
        } else {
                return p, nil
        }
}
func (p *GlobRange) Type() Type { return GlobType }
func (p *GlobRange) True() bool { return false }
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
func (p *GlobRange) Integer() (int64, error) { return 0, nil }
func (p *GlobRange) Float() (float64, error) { return 0, nil }
func (p *GlobRange) cmp(v Value) (res cmpres) {
        if v.Type() == GlobType {
                a, ok := v.(*GlobRange)
                assert(ok, "value is not GlobRange")
                res = p.Chars.cmp(a.Chars)
        }
        return
}

type Path struct {
        elements
        File *File // if this path is pointed to a file, ie. the last element matched a FileMap
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
func (p *Path) Integer() (int64, error) { return 0, nil }
func (p *Path) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *Path) Type() Type { return PathType }
func (p *Path) True() (t bool) {
        if t = p.File != nil; !t {
                for _, elem := range p.Elems {
                        t = elem.True(); break
                }
        }
        return
}
func (p *Path) expand(w expandwhat) (res Value, err error) {
        var (elems []Value; num int)
        if elems, num, err = expandall(w, p.Elems...); err != nil { return }
        if w&expandPath != 0 {
                var vals []Value
                for _, elem := range elems {
                        switch v := elem.(type) {
                        case *String:
                                segs := MakePathStr(v.string).Elems
                                vals = append(vals, segs...)
                        default:
                                vals = append(vals, elem)
                        }
                }
                elems = vals
        }
        if num > 0 {
                res = &Path{elements{elems}, p.File}
        } else {
                res = p
        }
        return
}

func (p *Path) dependcompare(c *comparer) (err error) {
        if optionTraceCompare { defer compun(comptrace(c, p)) }
        if enable_assertions { assert(c.target != p, "self comparation") }
        if ds, err := p.Strval(); err == nil {
                var di os.FileInfo
                if p.File != nil { di = p.File.info }
                err =  c.compareStatDepend(p, ds, di)
        }
        return
}

func (p *Path) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }

        var s string // path/file target
        if s, err = p.Strval(); err != nil {
                return
        }

        if p.File == nil {
                if pc.program.project.isFileName(filepath.Base(s)) || pc.program.project.isFileName(s) {
                        if p.File = pc.program.project.searchFile(s); p.File != nil {
                                pc.addNotExistedTarget1(p)
                                return
                        }
                }
        }

        if p.File != nil {
                err = p.File.prepare(pc)
                if err != nil { return }
        }

        var errs scanner.Errors
        var checked = make(map[string]bool)
        for _, err := range pc.updateTargetErrs(s) {
                e, isTargetNotFound := err.Err.(targetNotFoundError)
                if !isTargetNotFound {
                        errs = append(errs, err)
                        continue
                } else if b1, b2 := checked[e.target]; b1 && b2 {
                        continue
                } else {
                        checked[e.target] = true
                }
                if p.File = stat(e.target, "", ""); p.File == nil {
                        pc.addNotExistedTarget1(p) // Append unknown path anyway.
                        err.Err = pathNotFoundError{e.project, p}
                        errs = append(errs, err)
                        if optionTracePrepare {
                                //pc.tracef("execstack: %s", execstack)
                                //pc.tracef("%s: %s", e.project.name, err)
                                pc.tracef("%s", err)
                        }
                } else if p.File.info != nil {
                        if p.File.info.IsDir() {
                                pc.addNotExistedTarget1(p)
                        } else {
                                pc.addNotExistedTarget1(p/*.File*/)
                        }
                } else {
                        // Search this path target as a file.
                        p.File = e.project.searchFile(e.target)
                        if p.File != nil {
                                pc.addNotExistedTarget1(p.File)
                        } else {
                                errs = append(errs, err)
                        }
                }
        }
        checked = nil

        if len(errs) > 0 { err = errs } else { err = nil }
        return
}

func (p *Path) cmp(v Value) (res cmpres) {
        if v.Type() == PathType {
                a, ok := v.(*Path)
                assert(ok, "value is not Path")
                res = p.cmpElems(a.Elems)
        }
        return
}

type PathSeg struct { rune }
func (p *PathSeg) refs(_ Value) bool { return false }
func (p *PathSeg) closured() bool { return false }
func (p *PathSeg) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *PathSeg) Type() Type { return PathSegType }
func (p *PathSeg) True() bool { return true }
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
func (p *PathSeg) Float() (v float64, err error) { return }
func (p *PathSeg) Integer() (v int64, err error) { return }
func (p *PathSeg) prepare(pc *preparer) error { return nil }
func (p *PathSeg) cmp(v Value) (res cmpres) {
        if p.Type() == PathSegType {
                a, ok := v.(*PathSeg)
                assert(ok, "value is not PathSeg")
                if p.rune == a.rune {
                        res = cmpEqual
                }
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
                base = &filebase{ filestub{ dir, sub, name, nil, nil }, info }
                base.stub.other = &base.stub
                stub = &base.stub
                filecache[fullname] = base
        }
GotFile:
        file = &File{ base, stub }
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

type File struct { *filebase ; *filestub }
func (p *File) refs(_ Value) bool { return false }
func (p *File) closured() bool { return false }
func (p *File) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *File) Type() Type { return FileType }
func (p *File) True() bool { return p.name != "" }
func (p *File) String() string { return p.name }
func (p *File) Strval() (s string, err error) { s = p.FullName(); return }
func (_ *File) Integer() (int64, error) { return 0, nil }
func (_ *File) Float() (float64, error) { return 0, nil }

func (p *File) Dir() string { return p.dir }
func (p *File) BaseName() (s string) {
        if p.info != nil {
                s = p.info.Name()
        } else {
                s = filepath.Base(p.name)
        }
        return
}
func (p *File) FullName() (s string) {
        /*if filepath.IsAbs(p.name) {
                s = p.name
                if enable_assertions {
                        assert(strings.HasPrefix(p.name, filepath.Join(p.dir, p.sub)), "invalid file{%s %s %s}", p.dir, p.sub, p.name)
                }
        } else {
                s = filepath.Join(p.dir, p.sub, p.name)
        }
        return*/
        return filepath.Join(p.dir, p.sub, p.name)
}

func (p *File) searchInMatchedPaths(proj *Project) (res bool) {
        if p.match != nil {
                f := p.match.stat(proj.absPath, p.name)
                res = f != nil && f.exists()
        }
        return
}

func (p *File) dependcompare(c *comparer) (err error) {
        if optionTraceCompare { defer compun(comptrace(c, p)) }
        if enable_assertions { assert(c.target != p, "self comparation") }
        if ds, err := p.Strval(); err == nil {
                err =  c.compareStatDepend(p, ds, p.info)
        }
        return
}

func (p *File) prepare(pc *preparer) (err error) {
        if optionTracePrepare {
                defer prepun(preptrace(pc, p))
                if p.exists() { pc.tracef("exists: %s{%s}", p.Type(), p) }
        }

        if pc.entry.target == p {
                pc.tracef("error: target depends on itself")
                unreachable(p, "target depends on itself")
        } else {
                var s string
                switch t := pc.entry.target.(type) {
                case *Bareword: s = t.string
                case *String: s = t.string
                case *File: s = t.name
                }
                if s == p.name {
                        pc.tracef("error:%d: `%s` file depends on itself", pc.level, s)
                        unreachable(s, "file depends on itself")
                }
        }

        return pc.updateFile(p)
}

func checkPatternFileDepend(pc *preparer, project *Project, ps *StemmedEntry, prog *Program, g *PercPattern) (res bool, err error) {
        var name string
        if name, err = g.MakeString(ps.Stem); err != nil { return }
        if file := project.searchFile(name); file != nil { // Matches a FileMap (IsKnown(), may exists or not)
                if file.exists() {
                        res = true; return
                }
        }

        var entry *RuleEntry
        if entry, err = project.resolveEntry(name); err != nil {
                return
        } else if entry != nil {
                res = true; return
        }

        // TODO: project.resolvePatterns(name)
        return
}

func (p *File) cmp(v Value) (res cmpres) {
        if v.Type() == FileType {
                a, ok := v.(*File)
                assert(ok, "value is not File")
                if p.filebase == a.filebase {
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

type Flag struct { Name Value }
func (p *Flag) refs(v Value) bool { return p.Name.refs(v) }
func (p *Flag) closured() bool { return p.Name.closured() }
func (p *Flag) expand(w expandwhat) (res Value, err error) {
        var name Value
        if name, err = p.Name.expand(w); err == nil {
                if name != p.Name {
                        res = &Flag{ name }
                } else {
                        res = p
                }
        }
        return
}
func (p *Flag) Type() Type { return FlagType }
func (p *Flag) True() bool { return p.Name.True() }
func (p *Flag) elemstr(o Object, k elemkind) (s string) {
        return "-" + elementString(o, p.Name, k)
}
func (p *Flag) String() (s string) { return p.elemstr(nil, 0) }
func (p *Flag) Strval() (s string, e error) {
        if p.Name == nil || p.Name.Type() == NoneType {
                s = "-"
        } else if s, e = p.Name.Strval(); e == nil {
                s = "-" + s
        }
        return
}
func (p *Flag) Integer() (int64, error) { return 0, nil }
func (p *Flag) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *Flag) is(r rune, s string) (result bool, err error) {
        switch t := p.Name.(type) {
        case *Flag: result, err = t.is(r, s)
        case *String: result = t.string == s
        case *Bareword: if result = t.string == s; !result && r != 0 {
                result = strings.ContainsRune(t.string, r)
        }}
        return
}
func (p *Flag) opts(opts ...string) (runes []rune, names []string, err error) {
        switch t := p.Name.(type) {
        case *Flag:
                runes, names, err = t.opts(opts...)
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
        if v.Type() == FlagType {
                a, ok := v.(*Flag)
                assert(ok, "value is not Flag")
                res = p.Name.cmp(a.Name)
        }
        return
}
      
type Compound struct { elements } // "compound string"
func (p *Compound) expand(w expandwhat) (res Value, err error) {
        var ( elems []Value; num int )
        if elems, num, err = expandall(w, p.Elems...); err == nil {
                if num > 0 {
                        res = &Compound{ elements{ elems } }
                } else {
                        res = p
                }
        }
        return
}
func (p *Compound) Type() Type { return CompoundType }
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
func (p *Compound) Integer() (int64, error) { return int64(len(p.Elems)), nil }
func (p *Compound) Float() (float64, error) { i, e := p.Integer(); return float64(i), e }
func (p *Compound) cmp(v Value) (res cmpres) {
        if v.Type() == CompoundType {
                a, ok := v.(*Compound)
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
func (p *List) Type() Type { return ListType }
func (p *List) elemstr(o Object, k elemkind) (s string) {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elementString(o, elem, k))
        }
        return strings.Join(strs, " ")
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

func (p *List) dependcompare(c *comparer) (err error) {
        if optionTraceCompare { defer compun(comptrace(c, p)) }
        if enable_assertions { assert(c.target != p, "self comparation") }
        for _, elem := range p.Elems {
                if err = c.compareDepend(elem); err != nil { break }
        }
        return
}

func (p *List) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        var updates, good *breaker
        for _, v := range p.Elems {
                if p, ok := v.(prerequisite); ok {
                        if err = p.prepare(pc); err == nil { continue }
                        if br, ok := err.(*breaker); ok {
                                if br.what == breakUpdates {
                                        if updates == nil { updates = br } else {
                                                updates.updated = append(updates.updated, br.updated...)
                                        }
                                        err = nil
                                } else if br.what == breakGood {
                                        err, good = nil, br
                                }
                        }
                } else {
                        err = fmt.Errorf("%s `%s` is not prerequisite", v.Type(), v)
                }
                if err != nil { break }
        }
        if updates != nil && err != updates {
                err = updates
        } else if err == nil && good != nil {
                err = good
        }
        return
}

func (p *List) cmp(v Value) (res cmpres) {
        if v.Type() == ListType {
                a, ok := v.(*List)
                assert(ok, "value is not List")
                res = p.cmpElems(a.Elems)
        }
        return
}

type Group struct { List }
func (p *Group) Type() Type { return GroupType }
func (p *Group) elemstr(o Object, k elemkind) string {
        var strs []string
        for _, elem := range p.Elems {
                strs = append(strs, elementString(o, elem, k))
        }
        return fmt.Sprintf("(%s)", strings.Join(strs, " "))
}
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
                        res = &Group{ List{ elements{ elems } } }
                } else {
                        res = p
                }
        }
        return
}

func (p *Group) cmp(v Value) (res cmpres) {
        if v.Type() == GroupType {
                a, ok := v.(*Group)
                assert(ok, "value is not Group")
                res = p.cmpElems(a.Elems)
        }
        return
}

//type Map struct {
//        Elems map[string]Value
//}
/* func (p *Map) String() string {
        return "(" + p.List.String() + ")"
}
func (p *Map) Strval() string {
        return "(" + p.List.Strval() + ")"
} */

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
func (p *Pair) Type() Type { return PairType }
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
        if k.Type().Bits()&IsKeyName != 0 {
                p.Key = k
        } else {
                panic(fmt.Errorf("%s '%v' is not key name type", k.Type(), k))
        }
}
func (p *Pair) isFlag(r rune, s string) (result bool, err error) {
        if k, ok := p.Key.(*Flag); ok { result, err = k.is(r, s) }
        return
}
func (p *Pair) cmp(v Value) (res cmpres) {
        if v.Type() == PairType {
                a, ok := v.(*Pair)
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
        p Position
        l token.Token
        o Object
        a []Value
}

func (p *closuredelegate) Position() Position { return p.p }
func (p *closuredelegate) string(o Object, k elemkind) (s string) { // source representation
        for i, a := range p.a {
                if i == 0 { s = " " } else { s += "," }
                s += elementString(o, a, k)
        }
        switch name := p.o.Name(); p.l {
        case token.COLON: s = fmt.Sprintf(":%s%s:", name, s)
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
type delegate struct { closuredelegate }
func (p *delegate) Type() Type { return DelegateType }
func (p *delegate) True() (t bool) {
        if v, err:= p.expand(expandAll); err == nil {
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
        default: err = fmt.Errorf("%s '%v' is unknown delegation", o.Type(), o)
        case Caller:
                if res, err = o.Call(p.p, args...); err != nil {
                        if p.o.Name() != "error" {
                                err = scanner.WrapError(token.Position(p.p), err)
                        } else {
                                return
                        }
                }
        case Executer:
                if args, err = o.Execute(p.p, args...); err != nil {
                        if p.o.Name() != "error" {
                                err = scanner.WrapError(token.Position(p.p), err)
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
                res = universalnone
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
                        res = &delegate{closuredelegate{ p.p, p.l, o, args }}
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

func (p *delegate) dependcompare(c *comparer) (err error) {
        if optionTraceCompare { defer compun(comptrace(c, p)) }
        if enable_assertions { assert(c.target != p, "self comparation") }

        var v Value
        if v, err = p.expand(expandDelegate); err == nil {
                err = c.compareDepend(v)
        }
        return
}

func (p *delegate) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }

        var val Value
        if val, err = p.expand(expandDelegate); err != nil { return }
        /*for _, d := range merge(val) {
                if err = pc.traverse(d); err != nil { break }
        }*/
        err = pc.traverse(val) //err = pc.traverseAll(merge(val))
        return
}

func (p *delegate) cmp(v Value) (res cmpres) {
        if v.Type() == DelegateType {
                a, ok := v.(*delegate)
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

type closure struct { closuredelegate }
func (p *closure) Type() Type { return ClosureType }
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
func (p *closure) Integer() (int64, error) {
        if p.o == nil {
                return 0, nil
        }
        return p.o.Integer()
}
func (p *closure) Float() (float64, error) {
        if p.o == nil {
                return 0, nil
        }
        return p.o.Float()
}
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
                        err = fmt.Errorf("%s '%s' is not object", t.Type(), t)
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
                res = &closure{closuredelegate{ p.p, p.l, o, a }}
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
                res = &delegate{closuredelegate{ p.p, p.l, o, args }}
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

func (p *closure) dependcompare(c *comparer) (err error) {
        if optionTraceCompare { defer compun(comptrace(c, p)) }
        if enable_assertions { assert(c.target != p, "self comparation") }

        var v Value
        if v, err = p.expand(expandClosure); err == nil {
                err = c.compareDepend(v)
        }
        return
}

func (p *closure) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }

        if v, e := p.expand(expandClosure); e != nil {
                err = e
        } else if v == nil {
                err = fmt.Errorf("undefined closure target `%v`", p.o.Name())
                fmt.Fprintf(stderr, "%s: closure.prepare: %v\n", p.p, err)
        } else {
                err = pc.traverse(v)
        }
        return
}

func (p *closure) cmp(v Value) (res cmpres) {
        if v.Type() == ClosureType {
                a, ok := v.(*closure)
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
        t token.Token
        o Value // Object or selection
        s Value
}

func (p *selection) Type() Type { return SelectionType }
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

func (p *selection) object() (o Object, err error) {
        if s, ok := p.o.(*selection); ok {
                var v Value
                if v, err = s.value(); err != nil {
                        // sth's wrong!
                } else if o, _ = v.(Object); o == nil {
                        err = fmt.Errorf("selection.object: `%s` is nil", s.String())
                }
        } else if o, ok = p.o.(Object); !ok {
                err = fmt.Errorf("selection.object: %s '%v' is not object", p.o.Type(), p.o)
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
                        if pn, ok := o.(*ProjectName); ok && p.t == token.SELECT_PROG {
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
                res = &selection{ p.t, o, s }
        } else {
                res = p
        }
        return
}

func (p *selection) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }

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

func (p *selection) cmp(v Value) (res cmpres) {
        if v.Type() == SelectionType {
                a, ok := v.(*selection)
                assert(ok, "value is not selection")
                if p.o.cmp(a.o) == cmpEqual && p.s.cmp(a.s) == cmpEqual {
                        if p.t == a.t { res = cmpEqual }
                }
        }
        return
}

// Pattern
type Pattern interface {
        Value
        //concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error)
        match(i interface{}) (matched bool, stem string, err error)
        MakeString(stem string) (s string, err error)
}

type pattern struct {}
func (p *pattern) True() bool { return false }
func (p *pattern) Integer() (int64, error) { return 0, nil }
func (p *pattern) Float() (float64, error) { return 0, nil }
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
        pattern
        Prefix Value
        Suffix Value
}
func (p *PercPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *PercPattern) Type() Type { return PercPatternType }
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
func (p *PercPattern) match(i interface{}) (matched bool, stem string, err error) {
        var prefix, suffix, s string
        switch t := i.(type) {
        case string: s = t
        case *File: s = t.name
        default: if v, ok := i.(Value); ok {
                if s, err = v.Strval(); err != nil { return }
        }}
        if prefix, err = p.Prefix.Strval(); err == nil && strings.HasPrefix(s, prefix) {
                if suffix, err = p.Suffix.Strval(); err == nil && strings.HasSuffix(s, suffix) {
                        if a, b := len(prefix), len(s)-len(suffix); a < b {
                                matched, stem = true, s[a:b]
                        }
                }
        }
        return
}

func (p *PercPattern) MakeString(stem string) (s string, err error) {
        if s, err = p.Prefix.Strval(); err == nil {
                var v string
                if v, err = p.Suffix.Strval(); err == nil {
                        s += stem + v
                }
        }
        return
}

/*
func (p *PercPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        var target string
        if target, err = p.MakeString(stem); err == nil {
                entry, err = p.pattern.concrete(patent, target, stem)
        }
        return
}
*/

func (p *PercPattern) refs(v Value) bool { return p.Prefix.refs(v) || p.Suffix.refs(v) }
func (p *PercPattern) closured() bool { return p.Prefix.closured() || p.Suffix.closured() }

func (p *PercPattern) dependcompare(c *comparer) (err error) {
        if enable_assertions { assert(c.target != p, "self comparation") }
        
        var stem string
        if len(c.program.callers) == 0 {
                //err = fmt.Errorf("no calltrace (%s)", p)
                return
        } else if stem = c.program.callers[0].stem; stem == "" {
                //err = fmt.Errorf("empty stem (%s)", p)
                return
        }

        var target string
        if target, err = p.MakeString(stem); err != nil { return }

        if err = c.compareDepend(target); err != nil {
                err = patternCompareError{err}
        }
        return
}

func (p *PercPattern) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        if pc.stem == "" {
                err = fmt.Errorf("empty stem (%s)", p)
                return
        }

        var target string
        if target, err = p.MakeString(pc.stem); err != nil { return }
        if err = pc.updateTarget(target); err != nil {
                err = patternPrepareError{err}
        }
        return
}

func (p *PercPattern) cmp(v Value) (res cmpres) {
        if v.Type() == PercPatternType {
                a, ok := v.(*PercPattern)
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
        pattern
        Components []Value
}
func (p *GlobPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *GlobPattern) Type() Type { return GlobPatternType }
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
func (p *GlobPattern) match(i interface{}) (matched bool, stem string, err error) {
        var pat, s string
        switch t := i.(type) {
        case string: s = t
        case *File: s = t.name
        default: if v, ok := i.(Value); ok {
                if s, err = v.Strval(); err != nil { return }
        }}
        if pat, err = p.Strval(); err == nil {
                matched, err = filepath.Match(pat, s)
        }
        // FIXME: calculate stem from matching
        return
}

func (p *GlobPattern) MakeString(stem string) (s string, err error) {
        unreachable("FIXME: make string from stem")
        return
}

/*
func (p *GlobPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        var target string
        if target, err = p.MakeString(stem); err == nil {
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

func (p *GlobPattern) dependcompare(c *comparer) (err error) {
        if enable_assertions { assert(c.target != p, "self comparation") }

        var stem string
        if len(c.program.callers) == 0 {
                //err = fmt.Errorf("no calltrace (%s)", p)
                return
        } else if stem = c.program.callers[0].stem; stem == "" {
                //err = fmt.Errorf("empty stem (%s)", p)
                return
        }

        var target string
        if target, err = p.MakeString(stem); err != nil { return }

        if err = c.compareDepend(target); err != nil {
                err = patternCompareError{err}
        }
        return
}

func (p *GlobPattern) prepare(pc *preparer) (err error) {
        if optionTracePrepare { defer prepun(preptrace(pc, p)) }
        if pc.stem == "" {
                err = fmt.Errorf("empty stem (%s)", p)
                return
        }

        var target string
        if target, err = p.MakeString(pc.stem); err != nil { return }
        if err = pc.updateTarget(target); err != nil {
                err = patternPrepareError{err}
        }
        return
}

func (p *GlobPattern) cmp(v Value) (res cmpres) {
        if v.Type() == GlobPatternType {
                a, ok := v.(*GlobPattern)
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
type RegexpPattern struct { pattern }
func (p *RegexpPattern) refs(_ Value) bool { return false }
func (p *RegexpPattern) closured() bool { return false }
func (p *RegexpPattern) expand(_ expandwhat) (Value, error) { return p, nil }
func (p *RegexpPattern) Type() Type { return RegexpPatternType }
func (p *RegexpPattern) String() string { return "{RegexpPattern}" }
func (p *RegexpPattern) Strval() (s string, err error) { return "", nil }
func (p *RegexpPattern) match(i interface{}) (matched bool, stem string, err error) {
        panic("TODO: regexp matching...")
        return
}
/*
func (p *RegexpPattern) concrete(patent *RuleEntry, stem string) (entry *RuleEntry, err error) {
        panic("TODO: creating new match entry")
        return
}
*/
func (p *RegexpPattern) MakeString(stem string) (s string, err error) {
        panic("TODO: regexp makestring...")
        return
}
func (p *RegexpPattern) cmp(v Value) (res cmpres) {
        if v.Type() == RegexpPatternType {
                a, ok := v.(*RegexpPattern)
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

type positional struct {
        Value
        pos Position
}

// Position() returns the position of the value occurs position in file or nil.
func (p *positional) Position() Position { return p.pos }

// Positional wraps a value with a valid position
func Positional(v Value, pos Position) Positioner {
        if p, ok := v.(*positional); ok {
                p.pos = pos
                return p
        }
        return &positional{ v, pos }
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

func expandall(w expandwhat, values ...Value) (res []Value, num int, err error) {
        var v Value
        for _, elem := range values {
                if elem == nil {
                        res = append(res, universalnil)
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

func Scalar(v Value, t Type) (res Value) {
        if v.Type() == t {
                res = v
        } else if l, o := v.(*List); l != nil && o && l.Len() > 0 {
                res = Scalar(l.Elems[0], t)
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

func MakeAnswer(v bool) Value { if v { return universalyes } else { return universalno } }
func MakeBoolean(v bool) Value { if v { return universaltrue } else { return universalfalse } }
func MakeBin(i int64) *Bin { return &Bin{integer{i}} }
func MakeOct(i int64) *Oct { return &Oct{integer{i}} }
func MakeInt(i int64) *Int { return &Int{integer{i}} }
func MakeHex(i int64) *Hex { return &Hex{integer{i}} }
func MakeFloat(f float64) *Float { return &Float{f} }
func MakeDate(s time.Time) *Date { return &Date{DateTime{s}} }
func MakeTime(t time.Time) *Time { return &Time{DateTime{t}} }
func MakeString(s string) *String { return &String{s} }
func MakeURL(s *url.URL) *URL {
        var host, port string
        v := strings.Split(s.Host, ":")
        if len(v) == 1 { host = v[0] }
        if len(v) == 2 { host, port = v[0], v[1] }
        var password Value
        if t, ok := s.User.Password(); ok {password = &String{t}}
        return &URL{
                Scheme: &String{s.Scheme},
                Username: &String{s.User.Username()},
                Password: password,
                Host: &String{host},
                Port: &String{port},
                Path: &String{s.Path},
                Query: &String{s.RawQuery},
                Fragment: &String{s.Fragment},
        }
}
func MakeBarecomp(elems... Value) *Barecomp { return &Barecomp{elements{elems}} }
func MakeCompound(elems... Value) *Compound { return &Compound{elements{elems}} }
func MakeList(elems... Value) *List { return &List{elements{elems}} }
func MakeGroup(elems... Value) (v *Group) { return &Group{List{elements{elems}}} }
func MakeGlobMeta(tok token.Token) *GlobMeta { return &GlobMeta{tok} }
func MakeGlobRange(v Value) *GlobRange { return &GlobRange{v} }
func MakePath(segments... Value) (v *Path) { return &Path{elements{segments}, nil} }
func MakePathSeg(ch rune) *PathSeg { return &PathSeg{ch} }
func MakePathStr(str string) (v *Path) {
        var segments []Value
        for _, s := range strings.Split(str, PathSep) {
                segments = append(segments, &Bareword{s})
        }
        return MakePath(segments...)
}
func MakePair(k, v Value) (p *Pair) {
        if k.Type().Bits()&IsKeyName != 0 {
                p = &Pair{nil, nil}
                p.SetKey(k)
                p.SetValue(v)
        } else {
                panic(fmt.Errorf("%s '%v' is not key name type", k.Type(), k))
        }
        return
}
func MakePercPattern(prefix, suffix Value) Pattern {
        if prefix == nil { prefix = universalnone }
        if suffix == nil { suffix = universalnone }
        return &PercPattern{
                Prefix: prefix,
                Suffix: suffix,
        }
}
func MakeGlobPattern(components... Value) Pattern {
        return &GlobPattern{Components:components}
}
func MakeDelegate(pos Position, tok token.Token, obj Object, args... Value) Value {
        return &delegate{closuredelegate{ pos, tok, obj, args }}
}
func MakeClosure(pos Position, tok token.Token, obj Object, args... Value) Value {
        if obj == nil { panic("closure of nil") }
        return &closure{closuredelegate{ pos, tok, obj, args }}
}
func MakeListOrScalar(elems []Value) (res Value) {
        if x := len(elems); x > 1 {
                res = &List{elements{elems}}
        } else if x == 1 {
                res = elems[0]
        } else {
                res = universalnone
        }
        return
}

func Make(in interface{}) (out Value) {
        switch v := in.(type) {
        case int:       out = MakeInt(int64(v))
        case int32:     out = MakeInt(int64(v))
        case int64:     out = MakeInt(v)
        case float32:   out = MakeFloat(float64(v))
        case float64:   out = MakeFloat(v)
        case string:    out = &String{v}
        case time.Time: out = &DateTime{v} // FIXME: NewDate, NewTime
        case Value:     out = v
        default:        out = &Any{in}
        }
        return
}

func MakeAll(in... interface{}) (out []Value) {
        for _, v := range in {
                out = append(out, Make(v))
        }
        return
}

func ParseBin(s string) *Bin {
        if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 2, 64); e == nil {
                return MakeBin(i)
        } else {
                panic(e)
        }
}

func ParseOct(s string) *Oct {
        if strings.HasPrefix(s, "0") {
                s = s[1:]
        }
        if i, e := strconv.ParseInt(s, 8, 64); e == nil {
                return MakeOct(i)
        } else {
                panic(e)
        }
}

func ParseInt(s string) *Int {
        if i, e := strconv.ParseInt(s, 10, 64); e == nil {
                return MakeInt(i)
        } else {
                panic(e)
        }
}

func ParseHex(s string) *Hex {
        if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
                s = s[2:]
        }
        if i, e := strconv.ParseInt(s, 16, 64); e == nil {
                return MakeHex(i)
        } else {
                panic(e)
        }
}

func ParseFloat(s string) *Float {
        if f, e := strconv.ParseFloat(strings.Replace(s, "_", "", -1), 64); e == nil {
                return MakeFloat(f)
        } else {
                panic(e)
        }
}

func ParseDate(s string) *Date {
        if t, e := time.Parse("2006-01-02", s); e == nil {
                return MakeDate(t)
        } else {
                panic(e)
        }
}

func ParseTime(s string) *Time {
        if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
                return MakeTime(t)
        } else {
                panic(e)
        }
}

func ParseURL(s string) *URL {
        if u, e := url.Parse(s); e == nil {
                return MakeURL(u)
        } else {
                panic(e)
        }
}
