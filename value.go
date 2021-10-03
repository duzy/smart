//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "extbit.io/smart/token"
    "crypto/sha256"
    "path/filepath"
    "runtime/debug" // debug.PrintStack()
    "net/url"
    "reflect"
    "strconv"
    "strings"
    "bytes"
    "io/fs"
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

type (
    cmpres int
    existence int
    expandwhat int // TODO: -> expandfacet
    HashBytes [sha256.Size]byte
)
const (
    cmpUnknown cmpres = 0
    cmpSmaller     = -1 // meaningless so far
    cmpGreater     = 1  // meaningless so far
    cmpEqual       = 2
)
const (
    existenceMatterless existence = 1<<iota
    existenceConfirmed
    existenceNegated
)
const (
    expandDelegate expandwhat = 1<<iota // $(...)  ->  ......
    expandClosure   // &(...)            ->  $(...)
    expandSelection // foo->bar          -> ...
    expandArgs      // $(foo $(x),$(y))  -> $(foo ...,...)
    expandArgedArgs // foo($(args))      -> foo(...)
    expandDef       // foo=...           -> ...
    expandFullName  // foobar.c          -> /path/to/foobar.c    TODO
    expandPathStr   // "/path/to"/foo    -> /path/to/foo
    expandPairVal   // foo=$(bar)        -> foo=...
    expandPlainValue = expandClosure | expandDelegate | expandSelection | expandDef | expandPathStr | expandArgedArgs
)

func (v cmpres) String() (s string) {
    switch v {
    case cmpUnknown: s = "unknown"
    case cmpSmaller: s = "smaller"
    case cmpGreater: s = "greater"
    case cmpEqual:   s = "equal"
    }
    return
}

func (v existence) String() (s string) {
    switch v {
    case existenceMatterless: s = "matterless"
    case existenceConfirmed:  s = "confirmed"
    case existenceNegated:    s = "negated"
    }
    return
}

// A Comment node represents a single #-style comment.
type Comment struct {
	Pos Position // position of "#" starting the comment
	Text  string // comment text (excluding '\n')
}

// A CommentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

func (g *CommentGroup) Position() Position { return g.List[0].Pos }

type statinfo struct {
    file *File
    next *statinfo
}
func (si *statinfo) mod() (res time.Time) {
    for p := si; p != nil; p = p.next {
        if p.file != nil && p.file.info != nil {
            if t := p.file.info.ModTime(); t.After(res) { res = t }
        }
    }
    return
}
func (si *statinfo) exists() (res existence) {
    res = existenceMatterless
ForStatInfos:
    for p := si; p != nil; p = p.next {
        if  p.file != nil { // matterless is nil file
            if p.file.exists() {
                res = existenceConfirmed
            } else {
                res = existenceNegated
                break ForStatInfos
            }
        }
    }
    return
}

type (
    breakind int
    breaksco int
)

func (k breakind) String() (s string) {
        switch k {
        case breakUnkn: s = "break.unkn"
        case breakDone: s = "break.done"
        case breakNext: s = "break.next"
        case breakCase: s = "break.case"
        case breakFail: s = "break.fail"
        case breakErro: s = "break.error"
        }
        return
}

const (
        breakUnkn breakind = iota
        breakDone // (cond ...) and (case ...)
        breakNext // (cond ...) and (case ...)
        breakCase // (case ...)
        breakFail // (assert ...)
        breakErro // break with an error
)

const (
        breakGroup breaksco = iota
        breakTrave
)

type breaker struct {
        pos Position
        what breakind
        scope breaksco
        message string
        misstar *updatedtarget
        updated []*updatedtarget
        value Value
        error error
}

func (p *breaker) _error() (s string) {
        switch p.what {
        case breakUnkn: s = "unknown"
        case breakDone: s = "done" // ineligible (cond) is ignored
        case breakNext: s = "next"
        case breakCase: s = "case"
        case breakFail: s = "failure" // "break with assertion failure"
        case breakErro: s = fmt.Sprintf("break error: %v", p.error) // "break with an error"
        }
        if p.pos.IsValid() {
                if p.message != "" { s += ": " + p.message }
                if false { s = fmt.Sprintf("%s: %s", p.pos, s) }
        }
        return
}

func (p *breaker) prerequisites() (res []*updatedtarget) {
        for _, u := range p.updated {
                res = append(res, u.prerequisites...)
        }
        return
}

type breakers []*breaker

func (brks *breakers) has(what ...breakind) (res bool) {
    if len(what) == 0 { res = len(*brks) > 0 } else {
    ForBreakers:
        for _, brk := range *brks {
            for _, w := range what {
                if brk.what == w {
                    res = true
                    break ForBreakers
                }
            }
        }
    }
    return
}

func (brks *breakers) append(brk *breaker) {
    *brks = append(*brks, brk)
}

func (brks *breakers) not(what ...breakind) (res breakers) {
ForBreakers:
    for _, brk := range *brks {
        for _, w := range what {
            if brk.what == w { continue ForBreakers }
        }
        res = append(res, brk)
    }
    return
}

func (brks *breakers) of(what ...breakind) (res breakers) {
ForBreakers:
    for _, brk := range *brks {
        for _, w := range what {
            if brk.what == w {
                res = append(res, brk)
                continue ForBreakers
            }
        }
    }
    return
}

func (brks *breakers) add(pos Position, what breakind) *breaker {
    var brk = &breaker{ pos:pos, what:what }
    *brks = append(*brks, brk)
    return brk
}

func (brks *breakers) addf(pos Position, what breakind, s string, a... interface{}) *breaker {
    var brk = brks.add(pos, what)
    brk.message = fmt.Sprintf(s, a...)
    brk.scope = breakGroup
    return brk
}

type prioritizedContext struct {
    Context
    more []Context
}
func (pc *prioritizedContext) String() string {
    var s string
    for _, c := range pc.more { s += fmt.Sprintf(",%s", c) }
    return fmt.Sprintf("prioritized{%s%s}", pc.Context, s)
}
func (pc *prioritizedContext) autoGet(name string) (res Value) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if res = c.autoGet(name); !(isNil(res) /*|| isNone(res) || isUndef(res)*/) {
            break
        }
    }
    return
}
func (pc *prioritizedContext) autoSet(name string, val Value) (res Value, okay bool) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if res, okay = c.autoSet(name, val); okay { break }
    }
    return
}
func (pc *prioritizedContext) closureGet(name string) (res Value) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if res = c.closureGet(name); !(isNil(res) /*|| isNone(res) || isUndef(res)*/) {
            break
        }
    }
    return
}
func (pc *prioritizedContext) closureSet(name string, val Value) (res Value, okay bool) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if res, okay = c.closureSet(name, val); okay { break }
    }
    return
}
func (pc *prioritizedContext) closureResolveObject(pos Position, name string) (obj Object) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if obj = c.closureResolveObject(pos, name); !(isNil(obj) /*|| isNone(res) || isUndef(res)*/) {
            break
        }
    }
    return
}
func (pc *prioritizedContext) closureResolveEntry(pos Position, name string) (entry *RuleEntry) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if entry = c.closureResolveEntry(pos, name); entry != nil {
            break
        }
    }
    return
}

func prioritize(ctxs ...Context) (ctx Context) {
    if n := len(ctxs); n > 0 {
        ctx = &prioritizedContext{ctxs[0], ctxs[1:]}
    } else {
        panic("prioritize zero contexts")
    }
    return
}

type closureContext struct {
    Context
    scope *Scope
}
func (cc *closureContext) inner() Context { return cc.Context }
//func (cc *closureContext) Position() Position { return cc.scope.position }
func (cc *closureContext) Project() *Project { return cc.scope.project }
func (cc *closureContext) String() string {
    return fmt.Sprintf("closure{%s,%s}", cc.scope.comment, cc.Context)
}
func (cc *closureContext) closureGet(name string) (res Value) {
    var (
        pos = cc.Position()
        err error
    )
    if cc.scope.project != nil {
        if !pos.IsValid() { pos = cc.scope.position }
        if !pos.IsValid() { pos = cc.scope.project.position }
        if cc.scope != cc.scope.project.scope {
            if _, res = cc.scope.Find(name); !isNil(res) { return }
        }
        if res, err = cc.scope.project.resolveObject(cc, name); err != nil {
            diag.errorAt(pos, "resolve '%s' failed: %v", name, err).debug(1)
            return
        } else if isNil(res) {
            res = cc.autoGet(name)
        }
    }
    return
}
func (cc *closureContext) closureSet(name string, val Value) (prev Value, okay bool) {
    if def := cc.scope.FindDef(name); def != nil {
        prev = def.value
        def.val(cc, val)
        okay = true
    }
    return
}
func (cc *closureContext) closureResolveObject(pos Position, name string) (obj Object) {
    var err error
    if false && name == "objects" { defer func() {
        _, o1 := cc.scope.Find(name)
        o2, _ := cc.scope.project.resolveObject(cc, name)
        o3, _ := cc.scope.project.resolveObject(cc.inner(), name)
        var o4 Object
        if c := cc.inner().inner(); c != nil { o4, _ = cc.scope.project.resolveObject(c, name) }
        if o1 != nil { diag.warnAt(o1.Position(), "o1 = %T %v, project = %v, scope = %v", o1, o1, cc.scope.project, cc.scope) }
        if o2 != nil { diag.warnAt(o2.Position(), "o2 = %T %v, project = %v, scope = %v", o2, o2, cc.scope.project, cc.scope.project.scope) }
        if o2 != nil {
            val, _ := o2.expand(cc, expandPlainValue)
            diag.infoAt(o2.Position(), "o2 = %v", val)
        }
        diag.warnAt(cc.scope.position, "%v: bases = %v, name = %s", cc.scope.project, cc.scope.project.bases, name)
        diag.warnAt(pos, "%s: %T, %T, %T, %T", cc.scope.project, o1, o2, o3, o4)
        diag.warnAt(pos, "%s: obj = %T %v (%s)", cc.scope.project, obj, obj, cc.scope)
        diag.warnAt(pos, "%s: %s, %s", cc.scope.project, name, cc).debug(24)
    } () }
    if cc.scope != cc.scope.project.scope {
        if _, obj = cc.scope.Find(name); !isNil(obj) { return }
    }
    if obj, err = cc.scope.project.resolveObject(cc, name); err != nil {
        diag.errorAt(pos, "resolve object '%s' in '%s' failed: %v", name, cc.scope.project, err).debug(1)
    } else if isNil(obj) {
        obj = cc.Context.closureResolveObject(pos, name)
    }
    return
}
func (cc *closureContext) closureResolveEntry(pos Position, name string) (entry *RuleEntry) {
    var err error
    if project := cc.scope.project; project == nil {
        // none
    } else if entry, err = project.resolveEntry(cc, name, false); err != nil {
        diag.errorAt(pos, "resolve entry '%s' in '%s' failed: %v", name, project, err).debug(1)
        return
    } else if entry == nil {
        entry = cc.Context.closureResolveEntry(pos, name)
    }
    return
}

func closureWith(ctx Context, scope *Scope) Context { return &closureContext{ ctx, scope } }

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

func newUpdatedTarget(target Value, prerequisites ...*updatedtarget) *updatedtarget {
    if def, ok := target.(*Def); ok { target = def.value }
    return &updatedtarget{target, prerequisites}
}

func refdef(ctx Context, val Value, origin Origin) (res bool) {
    for _, def := range val.defs(ctx) {
        if def.origin == origin || origin == defany {
            return true
        }
    }
    return
}

// traversal is a single thread traverse context, for traversing in a new goroutine,
// a spawned traversal must be used and then merge.
type traversal struct {
    Context

    program *Program

    start time.Time // start time

    params []string // $0, $1, $2, ...
    execRec map[Value]int

    group sync.WaitGroup
    calleeErrs []error
    calleeErrsM sync.Mutex

    entry *RuleEntry // caller entry (target)
    args, arguments []Value // target and argumented prerequisite args

    target0, targetN, targetX string // names for setting first, last and all targets
    targets []Value // all targets def
    grepped []Value
    grepping bool

    updated []*updatedtarget // prerequisites newer than the target (from comparer) ($?)
    stems   []string // set by StemmedEntry

    traceLevel int

    interpreted []interpreter
    isConfigureExecution bool

    print bool // printing work directories (Entering/Leaving)
    debug bool

    batchMux sync.Mutex
    traceCallStackMux sync.Mutex
}
func (t *traversal) inner() Context { return t.Context }
func (t *traversal) String() string {
    var proj *Project
    if t.program != nil { proj = t.program.project }
    if proj == nil { proj = t.Context.Project() }
    return fmt.Sprintf("traversal{%s,%s}", proj, t.Context)
}

func (t *traversal) level(n int) { t.traceLevel += n }
func (t *traversal) trace(a ...interface{}) { printIndentDots(t.traceLevel, a...) }
func (t *traversal) tracef(s string, a ...interface{}) { printIndentDots(t.traceLevel, fmt.Sprintf(s, a...)) }

func (t *traversal) guarding() func() { t.batchMux.Lock(); return t.batchMux.Unlock }
func (t *traversal) batch(job func()) { defer t.guarding()(); job() }
func (t *traversal) caller() (c *traversal, ctx Context) {
    var more []Context
    ctx = t.Context
SwitchCtx:
    switch tc := ctx.(type) {
    case *traversal         : c   = tc
    case *callContext       : ctx = tc.Context; c, _ = tc.Context.(*traversal)
    case *entryContext      : ctx = tc.Context; c, _ = tc.Context.(*traversal)
    case *closureContext    : ctx = tc.Context; goto SwitchCtx
    case *prioritizedContext: ctx = tc.Context; more = append(more, tc.more...); goto SwitchCtx
    default:
        if len(more) > 0 {
            ctx = more[1]
            more = more[1:]
            goto SwitchCtx
        }
        diag.errorAt(t.Position(), "%s", tc)
        diag.errorAt(t.Position(), "%s", t).debug(1)
    }
    return
}

func (t *traversal) traceCallStack(pos Position, n int, s string, a ...interface{}) (point *diagnostic) {
    t.traceCallStackMux.Lock(); defer t.traceCallStackMux.Unlock()
    point = diag.errorAt(pos, s, a...)
    point = diag.errorAt(t.program.position, "from here for %v", t.entry)
    for c, last := t, t.program.position; c != nil && c.program != nil && n != 0; c, _ = c.caller() {
        if pos := c.program.position; !pos.SameLine(&last) {
            point = diag.errorAt(pos, "and here for %v", c.entry)
            last = pos
            n -= 1
        }
    }
    return
}

func (t *traversal) Project() *Project {
    if cc, ok := t.Context.(*closureContext); ok && cc != nil {
        return cc.Project()
    } else if t.program != nil {
        return t.program.project
    }
    return t.Context.Project()
}

func (t *traversal) Position() Position {
    if t.program != nil { return t.program.position }
    return t.Context.Position()
}

func (t *traversal) spawn(ctx Context, args ...Value) (child *traversal) {
    var (
        cc = &callContext{ ctx, make(callDefMap) }
        pos = ctx.Position()
        err error
    )
    child = &traversal{
        Context: cc,
        execRec: make(map[Value]int),
        program: t.program,
        start: time.Now(),
    }
    for _, s := range []string{ "@", "<", ">", "^", "|", "~", "?", "-", "*" } {
        if val := t.autoGet(s); !isNil(val) { child.autoSet(s, val) }
    }
    if child.params, err = cc.setArgs(t.program.params, args); err != nil {
        diag.errorAt(pos, "%v", err).debug(1)
    }
    return
}

func (t *traversal) add(ctx Context, target Value) {
    if isNil(target) || isNone(target) { return } else {
        for _, t := range t.targets {
            if t == target || t.cmp(ctx, target) == cmpEqual { return }
        }
        var n = len(t.targets)
        t.targets = append(t.targets, target)
        if t.targetX != "" { t.autoSet(t.targetX, MakeList(t.targets[0].Position(), t.targets...)) }
        if t.target0 != "" && n == 0 { t.autoSet(t.target0, t.targets[0]) }
        if t.targetN != "" { t.autoSet(t.targetN, t.targets[n]) }
    }
}

func (t *traversal) getCurrentTargetValue(ctx Context) (res Value) {
    if target := t.autoGet("@"); isNil(target) {
        if false { diag.errorOf(target, "target '%v' is nil", target) }
    } else if vals, _, err := expandall2(ctx, expandPlainValue, target); err != nil {
        diag.errorOf(target, "expand target '%v' failed: %v", target, err)
    } else if len(vals) == 1 { res = vals[0] } else {
        diag.errorOf(target, "target '%v' expaned to many: %v", target, res)
    }
    return
}

func (t *traversal) exists(ctx Context, v Value) bool {
    // FIXME: returns true if existenceMatterless ??
    return v != nil && v.stat(t).exists() == existenceConfirmed
}

func (t *traversal) depth() (res int) {
    for c, _ := t.caller(); c != nil; c, _ = c.caller() { res += 1 }
    return
}

func (t *traversal) calleeStart() {
    t.group.Add(1)
}

func (t *traversal) calleeDone(err error) {
    if err != nil {
        t.calleeErrsM.Lock()
        t.calleeErrs = append(t.calleeErrs, err)
        t.calleeErrsM.Unlock()
    }
    t.group.Done()
}

// DEPRECATED
func (t *traversal) any(ctx Context, i interface{}) {
    if optionEnableBenchmarks && false {  defer bench(mark(fmt.Sprintf("traversal.any(%s=%v)", typeof(i), i))) }

    var err error
    var pos = ctx.Position() //t.def.target.position
    if v := reflect.ValueOf(i); v.Kind() == reflect.Slice {
        for n := 0; err == nil && n < v.Len(); n++ {
            if optionEnableBenchmarks && false {
                i := v.Index(n).Interface()
                a, b := mark(fmt.Sprintf("%v: %s %v", n, typeof(i), i))
                t.any(ctx, i)
                bench(a, b)
            } else {
                t.any(ctx, v.Index(n).Interface())
            }
        }
    } else if i == nil {
        diag.errorAt(pos, "updating nil prerequisite")
    } else if value, ok := i.(Value); !ok {
        diag.errorAt(pos, "'%v' is invalid", value)
    } else if isNil(value) { // this could happen
        diag.errorAt(pos, "updating nil prerequisite")
    } else {
        value.traverse(t)
    }
    return
}

// DEPRECATED
func (t *traversal) filestub(ctx Context, p *Project, file *File, stub *filestub) (okay bool, brks  breakers) {
    if optionEnableBenchspots { defer bench(spot("traversal.filestub")) }

    /// Searching entries from the most derived project.
    var ( entry *RuleEntry; err error )
    if entry, err = p.resolveEntry(ctx, stub.name, t.grepping); err != nil {
        diag.errorOf(stub.filemap.pattern, "resolve entry failed: %v", err).debug(1)
        return
    } else if entry != nil {
        entry.traverse(t)
        return
    }

    /// Searching patterns from the most derived project.
    for _, entry := range p.resolvePatterns(ctx, stub) {
        brks = entry.file(t, file);
        okay = !brks.has()
        break
    }
    return
}

func (t *traversal) closuredProjects() (projects []*Project) {
    if t.Project() != nil { projects = append(projects, t.Project()) }
    if t.Project() != t.program.project { projects = append(projects, t.program.project) }
    for c := t; c != nil; c, _ = c.caller() {
        var proj = c.Project()
        for _, p := range projects {
            if proj == p { proj = nil; break }
        }
        if proj != nil { projects = append(projects, proj) }
    }
    return
}

func (t *traversal) forClosuredProjects(f func(*Project) (bool, error)) (okay bool, err error) {
    var projects = t.closuredProjects()
    for _, proj := range projects {
        if okay, err = f(proj); okay || err != nil { break }
    }
    return
}

func (t *traversal) file(file *File) (okay bool, brks breakers) {
    if options.traceTraversal   { t.tracef("traversal.file: %s", file) }
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("traversal.file(%v)", file))) }
    if optionEnableBenchspots { defer bench(spot("traversal.file")) }

    var (
        ctx = t.Context
        projects = t.closuredProjects()
        currentTargetValue = t.getCurrentTargetValue(ctx)
        concreteEntries = make(map[*RuleEntry]bool)
        concreteList []*RuleEntry
        err error
    )
    if isNil(currentTargetValue) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    }
    defer func() {
        // Note that the file maybe not traversed yet at this point. But we
        // still have to check mod-time.
        var (
            a = currentTargetValue.stat(t).mod()
            b = file.stat(t).mod()
        )
        if !a.IsZero() && b.After(a) { // a.IsZero() indicates the target not exists
            t.appendUpdated(ctx, newUpdatedTarget(file))
        }

        // Add to the $^ or $| list
        t.add(ctx, file)
    } ()

    for _, project := range projects {
        var entry *RuleEntry
        if entry, err = project.resolveEntry(ctx, file.name, t.grepping); err != nil {
            t.batch(func() {
                diag.errorAt(file.position, "resolve entry '%v' failed: %v", file.name, err).debug(1)
                t.traceCallStack(file.position, -1, "%v:", file.name)
            })
            return
        } else if entry == nil {
            continue
        } else if _, ok := concreteEntries[entry]; !ok {
            concreteList = append(concreteList, entry)
            concreteEntries[entry] = true
        }
    }
    for _, entry := range concreteList {
        if entry != nil && currentTargetValue != entry {
            if brks = entry.traverse(t); brks.has() {
                okay = brks.has(breakCase, breakDone)
                return
            }
        }
    }

    var (
        stemmedEntries = make(map[*RuleEntry]bool)
        stemmedList []*stemmed
    )
    for _, project := range projects {
        for _, ent := range project.resolvePatterns(ctx, file.name) {
            if _, ok := stemmedEntries[ent.RuleEntry]; !ok {
                stemmedList = append(stemmedList, ent)
                stemmedEntries[ent.RuleEntry] = true
            }
        }
    }
    for _, entry := range stemmedList {
        var proj = entry.OwnerProject()
        if brks = entry.file(t, file); !brks.has() {
            if okay = file.exists(); okay { break }
        } else if tb := brks.of(breakFail, breakErro); tb.has() {
            t.batch(func() {
                for _, brk := range tb {
                    switch brk.what {
                    case breakFail: diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v failed: %v", file, brk.message)
                    case breakErro: diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v with error: %v", file, brk.error)
                    default: diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v (%v)", file, brk.what)
                    }
                }
                if brks.has() {
                    diag.errorAt(entry.position, "broken traversal for stemmed file entry %v in %v (%d more)", entry, proj, len(brks)).debug(1)
                } else {
                    diag.errorAt(entry.position, "broken traversal for stemmed file entry %v in %v", entry, proj).debug(1)
                }
            })
            brks = brks.not(breakFail, breakErro)
            return
        } else if tb := brks.of(breakCase, breakDone); tb.has() {
            if brks = brks.not(breakCase, breakDone); !brks.has() { okay = true }
        } else if nxts := brks.of(breakNext); nxts.has() {
            if brks = brks.not(breakNext); !brks.has() { continue }
        }
        if brks.has() {
            t.batch(func() {
                for _, brk := range brks {
                    diag.errorAt(brk.pos, "broken traversal for stemmed file entry %v (%v)", file, brk.what)
                }
                diag.errorAt(entry.position, "broken traversal for stemmed file entry %v", entry).debug(1)
                t.traceCallStack(entry.position, -1, "broken traversal for stemmed file %v:", file)
            })
            return
        } else if okay { break }
    }

    for _, project := range projects {
        if okay { break } else if file != nil {
            if okay = file.info != nil; okay { break }
            okay = file.searchInMatchedPaths(ctx, project)
        }
    }

    if err != nil {
        t.traceCallStack(file.position, -1, "%v: file(%v): error: %v", t.Project(), file, err).debug(1)
    } else if !okay && len(t.stems) == 0 {
        t.batch(func() {
            for _, concrete := range concreteList {
                diag.errorAt(concrete.position, "concrete: %v", concrete)
            }
            for _, stemmed := range stemmedList {
                diag.errorAt(stemmed.position, "stemmed: %v", stemmed)
            }
            diag.errorAt(file.position, "missing file %v (%d concrete, %d stemmed)", file, len(concreteList), len(stemmedList))
            diag.errorAt(file.position, "missing file %v, call stack:", file).debug(16)
            t.traceCallStack(file.position, -1, "missing file %v required by %v (in %v)", file, currentTargetValue, t.Project())
        })
        brks.add(file.position, breakErro).error = fileNotFoundError{ t.Project(), file }
    } else if !okay && len(t.stems) > 0 {
        if false { brks.add(file.position, breakNext).scope = breakTrave }
    }
    return
}

func (t *traversal) string(targetVal Value, target string) (okay bool, brks breakers) {
    if options.traceTraversal { t.tracef("traversal.target: %s", target) }
    if optionEnableBenchmarks { defer bench(mark(fmt.Sprintf("traversal.target(%v)", target))) }
    if optionEnableBenchspots { defer bench(spot("traversal.target")) }

    var (
        ctx = t.Context
        pos Position = ctx.Position()
        projects = t.closuredProjects()
        currentTargetValue = t.getCurrentTargetValue(ctx)
        concreteEntries = make(map[*RuleEntry]bool)
        concreteList []*RuleEntry
        file *File // if target is file
        err error
    )
    if isNil(currentTargetValue) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    }
    defer func() {
        // Note that the file maybe not traversed yet at this point. But we
        // still have to check mod-time.
        if file != nil {
            var (
                a = currentTargetValue.stat(t).mod()
                b = file.stat(t).mod()
            )
            if !a.IsZero() && b.After(a) { // a.IsZero() indicates the target not exists
                t.appendUpdated(ctx, newUpdatedTarget(file))
            }
            if !file.position.IsValid() {
                file.position = pos
            }

            // Add to the $^ or $| list
            t.add(ctx, file)
        } else if true {
            if targetVal == nil { targetVal = MakeString(pos, target) }
            t.add(ctx, targetVal)
        }
    } ()

    for _, project := range projects {
        var entry *RuleEntry
        if entry, err = project.resolveEntry(ctx, target, t.grepping); err != nil {
            diag.errorAt(pos, "resolve entry '%v' failed: %v", target, err).debug(1)
            t.traceCallStack(pos, -1, "resolve entry '%v' failed: %v", target, err)
            return
        } else if entry == nil {
            continue
        } else if _, ok := concreteEntries[entry]; !ok {
            concreteList = append(concreteList, entry)
            concreteEntries[entry] = true
        }
    }

    for _, entry := range concreteList {
        var project = entry.OwnerProject()
        if entry != nil && currentTargetValue != entry {
            if w, ok := currentTargetValue.(*Bareword); ok && w.string == target {
                continue // target resolve to itself, does nothing
            } else if brks = entry.traverse(t); !brks.has() {
                file, _ = entry.target.(*File)
                okay = true
                return
            } else if tb := brks.of(breakFail, breakErro); len(tb) > 0 {
                brks = brks.not(breakFail, breakErro);
                t.batch(func () {
                    for _, brk := range tb {
                        switch brk.what {
                        case breakFail: diag.errorAt(entry.position, "traverse %v failed: %v", entry, brk.message).debug(1)
                        case breakErro: diag.errorAt(entry.position, "traverse %v error: %v", entry, brk.error).debug(1)
                        }
                    }
                    diag.errorAt(pos, "broken traversal for stemmed entry '%v' in %v", entry, entry.OwnerProject()).debug(1)
                })
                return
            } else if tb = brks.of(breakNext); len(tb) > 0 {
                if brks = brks.not(breakNext); brks.has() {
                    diag.errorAt(entry.position, "next with broken traversal for %v (entry=%v, next=%v)",
                        target, entry, tb[0].value).debug(1)
                }
                continue
            } else if tb = brks.of(breakCase, breakDone); len(tb) > 0 {
                brks, okay = brks.not(breakCase, breakDone), true // reset breakers
                break
            }
            if brks.has() {
                t.batch(func() {
                    for _, brk := range brks {
                        switch brk.what {
                        case breakFail: diag.errorAt(pos, "broken traversal for concrete entry '%v' failed: %v", entry, brk.message)
                        case breakErro: diag.errorAt(pos, "broken traversal for concrete entry '%v' with error: %v", entry, brk.error)
                        default: diag.errorAt(pos, "broken traversal for concrete entry '%v' in %v (%v)", entry, project, brk.what)
                        }
                    }
                    diag.errorAt(pos, "broken traversal for concrete entry '%v' in %v", entry, project).debug(1)
                })
                return
            }
        }
    }

    for _, project := range projects {
        var obj Object
        if obj, err = project.resolveObject(ctx, target); err != nil {
            diag.errorAt(pos, "resolve object '%s' failed: %v", target, err).debug(1)
            return
        } else if isNil(obj) || isUndef(obj) || isNone(obj) {
            // does nothing here and keep trying FindFile
        } else if brks = obj.traverse(t); brks.has() {
            t.batch(func() {
                diag.errorAt(pos, "broken traversal '%v' (%T) (project=%v)", obj, obj, project).debug(1)
            })
            return
        } else if _, ok := obj.(*ProjectName); ok {
            // solved, no need to check against patterns
            return
        }
    }

    var (
        stemmedEntries = make(map[*RuleEntry]bool)
        stemmedList []*stemmed
    )
    for _, project := range projects {
        for _, ent := range project.resolvePatterns(ctx, target) {
            if _, ok := stemmedEntries[ent.RuleEntry]; !ok {
                stemmedList = append(stemmedList, ent)
                stemmedEntries[ent.RuleEntry] = true
            }
        }
    }
    for _, entry := range stemmedList {
        if brks = entry.string(t, targetVal, target); !brks.has() {
            okay = true; break // continue
        } else if tb := brks.of(breakFail, breakErro); len(tb) > 0 {
            brks = brks.not(breakFail, breakErro);
            t.batch(func () {
                for _, brk := range tb {
                    switch brk.what {
                    case breakFail: diag.errorAt(entry.position, "traverse %v failed: %v", file, brk.message).debug(1)
                    case breakErro: diag.errorAt(entry.position, "traverse %v error: %v", file, brk.error).debug(1)
                    }
                }
                diag.errorAt(pos, "broken traversal for stemmed entry '%v' in %v", entry, entry.OwnerProject()).debug(1)
            })
            return
        } else if tb = brks.of(breakNext); len(tb) > 0 {
            if brks = brks.not(breakNext); brks.has() {
                diag.errorAt(entry.position, "next with broken traversal for %v (pattern=%v, next=%v)",
                    target, entry.Pattern, tb[0].value).debug(1)
            }
            continue
        } else if tb = brks.of(breakCase, breakDone); len(tb) > 0 {
            brks, okay = brks.not(breakCase, breakDone), true // reset breakers
            break
        } else {
            t.batch(func () {
                diag.errorAt(entry.position, "unknown breakers for target %v (%v)", target, brks[0].what).debug(1)
                t.traceCallStack(entry.position, -1, "unknown breakers for target %v (%v)", target, brks[0].what)
            })
            return
        }
    }

    for _, project := range projects {
        if file = project.FindFile(ctx, target); file != nil {
            file.position = pos // Change the position for tracing
            if false {
                var names = make(map[string]bool)
                for stub := file.filestub; true; stub = stub.other {
                    names[stub.name] = true // mark to avoid trying many times
                    if okay, brks = t.filestub(ctx, project, file, stub); brks.has() { return }
                    if okay { file.filestub = stub; return }
                    if stub.other == file.filestub { break }
                }

                // Try other names
                var name string
                for s, i := file.name, strings.LastIndex(file.name, PathSep); s != "" && i >= 0; i = strings.LastIndex(s, PathSep) {
                    if i == 0 { name = file.fullname() } else { name = filepath.Join(s[i+1:], name) }
                    s = s[:i] // strip off the prefix

                    if _, tried := names[name]; tried { continue }
                    names[name] = true // mark to avoid duplication

                    var sub = filepath.Join(file.sub, s)
                    var stub = &filestub{ file.dir, sub, name, file.filemap, file.filestub.other }
                    file.filestub.other = stub

                    if okay, brks = t.filestub(ctx, project, file, stub); brks.has() { return }
                    if okay { file.filestub = stub; break }
                }
            } else if okay, brks = t.file(file); brks.has() {
                return
            }

            // Check file existance
            if okay { break } else if file.info != nil {
                okay = true // it's good
            } else if file != nil {
                okay = file.searchInMatchedPaths(ctx, project)
            }
            if !okay && file.name != target {
                var alt = file //project.FindFile(file.name)
                if alt != nil { okay = alt.sub == "-" || alt.exists() }
            }
            if okay { return } // Done!
        }
    }

    if err != nil {
        t.traceCallStack(pos, -1, "%v: target(%v), file=%v: error: %v", t.Project(), target, file, err).debug(1)
    } else if !okay && !t.isConfigureExecution && len(t.stems) == 0 {
        if file != nil {
            diag.errorAt(file.position, "missing file %v", file)
            diag.errorAt(file.position, "concrete: %v", concreteList)
            diag.errorAt(file.position, "stemmed: %v", stemmedList)
            diag.errorAt(file.position, "internal stack:").debug(64)
            t.traceCallStack(file.position, -1, "traverse missing target file '%v' for %v", file, t.Project()).debug(1)
            brks.add(file.position, breakErro).error = fileNotFoundError{t.Project(), file}
        } else {
            diag.errorAt(pos, "missing target %v", target)
            diag.errorAt(pos, "concrete: %v", concreteList)
            diag.errorAt(pos, "stemmed: %v", stemmedList)
            diag.errorAt(pos, "internal stack:").debug(64)
            t.traceCallStack(pos, -1, "traverse missing target '%v' for %v", target, t.Project()).debug(1)
            brks.add(pos, breakErro).error = targetNotFoundError{t.Project(), target}
        }
    } else if !okay && len(t.stems) > 0 {
        brks.add(pos, breakNext).scope = breakTrave
    }
    return
}

func (t *traversal) pattern(pat Value) (brks breakers) {
    var (
        pos = pat.Position()
        ctx = positional(t, pos)
        rest []string
        okay bool
        s string
    )
    ctx = positional(ctx, pos)
    if s, rest = pat.stencil(ctx, t.stems); s == "" {
        diag.errorAt(pos, "empty stencil: %v %v", pat, t.stems).debug(1)
        return
    } else if len(rest) > 0 {
        diag.errorAt(pos, "partial stencil: %v, %v, %v, %v", pat, s, rest, t.stems).debug(1)
        panic(s)
    } else if file := t.Project().FindFile(ctx, s); file != nil {
        file.position = pos
        okay, brks = t.file(file)
    } else {
        okay, brks = t.string(pat, s)
    }

    if !okay && len(t.stems) > 0 {
        b := brks.add(pos, breakNext)
        b.scope = breakTrave
        b.value = pat
    }
    return
}

func (t *traversal) appendUpdated(ctx Context, updated *updatedtarget) {
    var currentTargetValue Value = t.getCurrentTargetValue(ctx)
    if isNil(currentTargetValue) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    } else {
        if currentTargetValue == updated.target { return }
        if currentTargetValue.cmp(ctx, updated.target) == cmpEqual { return }
    }
    for _, u := range t.updated { // check if already added
        if u.target == updated.target { return }
        if u.target.cmp(ctx, updated.target) == cmpEqual { return }
    }
    t.updated = append(t.updated, updated)
    for c, _ := t.caller(); c != nil; c, _ = c.caller() { // clear update loop
        if ct := c.autoGet("@"); !isNil(ct) && ct == updated.target { return }
    }
    if c, _ := t.caller(); c != nil {
        c.appendUpdated(ctx, newUpdatedTarget(currentTargetValue, updated))
    }
}

func (t *traversal) removeUpdated(ctx Context, target Value) (removed []*updatedtarget) {
    var currentTargetValue = t.getCurrentTargetValue(ctx)
    if isNil(currentTargetValue) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    }
    for i, u := range t.updated {
        if u.target == target || u.target.cmp(ctx, target) == cmpEqual {
            removed = append(removed, u)
            t.updated = append(t.updated[:i], t.updated[i+1:]...)
            if c, _ := t.caller(); c != nil && len(t.updated) == 0 {
                c.removeUpdated(ctx, currentTargetValue)
            }
        }
    }
    return
}

func (t *traversal) removeCallerUpdated(ctx Context, target Value) {
    if c, _ := t.caller(); c != nil {
        for _, u := range c.removeUpdated(ctx, target) {
            for _, uu := range u.prerequisites {
                t.removeUpdated(ctx, uu.target)
            }
        }
    }
}

func (t *traversal) hashDir(k []byte) string {
    dir := t.program.project.tmpPath
    h := fmt.Sprintf("%x", k[:2]) // HEX of the first two bytes
    return filepath.Join(dir, ".hash", h[0:1], h[1:2], h[2:3], h[3:])
}

func (t *traversal) cmdHash(ctx Context, values ...Value) (k, v HashBytes, err error) {
    var currentTargetValue = t.getCurrentTargetValue(ctx)
    if isNil(currentTargetValue) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    }

    var (
        key = sha256.New()
        val = sha256.New()
        str string
    )
    if str, err = fullnameOrStrval(ctx, currentTargetValue); err != nil { return }
    fmt.Fprintf(key, "%s", t.program.project.absPath)
    fmt.Fprintf(key, "%v", str)

    for _, value := range values {
        if false {
            // FIXME: Strval() varies when &(var) is used
            if str, err = value.Strval(ctx); err != nil { return }
            fmt.Fprintf(val, "%v", str)
        } else {
            fmt.Fprintf(val, "%v", value)
        }
    }
    copy(k[:], key.Sum(nil))
    copy(v[:], val.Sum(nil))
    return
}

func (t *traversal) updateRecipesHash(ctx Context) (k, v HashBytes, err error) {
    if k, v, err = t.cmdHash(ctx, t.program.recipes...); err != nil {
        diag.errorAt(t.program.position, "hashing recipes failed: %v", err).debug(1)
        return
    }

    var dir = t.hashDir(k[:])
    var name = filepath.Join(dir, fmt.Sprintf("%x", k))
    if f, e := os.Open(name); e == nil {
        defer f.Close()

        var h []byte
        if n, e := fmt.Fscanf(f, "%x", &h); e != nil {
            err = e; return
        } else if n == 1 && bytes.Equal(v[:], h) {
            return
        }
    }

    if err = os.MkdirAll(dir, 0700); err != nil {
        return
    } else if f, e := os.Create(name); e == nil {
        defer f.Close()
        _, err = fmt.Fprintf(f, "%x", v)
    } else {
        err = e
    }
    return
}

func (t *traversal) isRecipesDirty(ctx Context) (dirty bool, err error) {
    var k, v HashBytes
    if k, v, err = t.cmdHash(ctx, t.program.recipes...); err != nil {
        diag.errorAt(t.program.position, "compute recipes hash failed: %v", err).debug(1)
        return
    }

    var dir = t.hashDir(k[:])
    var name = filepath.Join(dir, fmt.Sprintf("%x", k))
    if f, e := os.Open(name); e == nil {
        defer f.Close()

        var h []byte
        if n, e := fmt.Fscanf(f, "%x", &h); e != nil {
            err = e
        } else if n == 1 {
            dirty = !bytes.Equal(v[:], h)
        }
    }
    return
}

func (t *traversal) wait(ctx Context, opts ...bool) (target Value, files []*File, execRes *ExecResult, err error) {
    if optionEnableBenchmarks && false { defer bench(mark("traversal.wait")) }

    var pos Position = ctx.Position()
    // Waiting for prerequisites
    var calleeErrs []error
    t.group.Wait()
    t.calleeErrsM.Lock()
    calleeErrs = t.calleeErrs; t.calleeErrs = nil
    t.calleeErrsM.Unlock()

    if target = t.getCurrentTargetValue(ctx); isNil(target) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    } else if isNone(target) {
        t.batch(func() {
            diag.errorAt(pos, "target is <none>").debug(8)
            t.traceCallStack(pos, 8, "target is <none>")
        })
        return
    } else if n := len(calleeErrs); n > 0 /*&& t.stems == nil*/ {
        t.batch(func() {
            var (
                numRealErrs = 0
                targetPos = pos //target.Position()
                targetValuePos = target.Position()
            )
            for _, err := range calleeErrs {
                diag.errorAt(pos, "%v: %v", target, err).debug(1)
                numRealErrs += 1
            }
            if numRealErrs == 0 { return } // simply return if no real errors
            if !pos.Equals(&targetPos) {
                var s string; if n > 1 { s = "s" }
                diag.errorAt(targetPos, "%d error%s while waiting prerequisites for '%v'", n, s, target).debug(1)
            }

            var v = target
            if l, ok := v.(*List); ok && l.Len() == 1 { v = l.Elems[0] }
            if targetValuePos.IsValid() && !targetValuePos.Equals(&targetPos) {
                if f, ok := v.(*File); ok && f.filemap != nil {
                    diag.errorAt(targetValuePos, "waiting for '%v'", target)
                    diag.errorOf(f.filemap.pattern, "via pattern '%v' (of %v)", v, f.filemap.project).debug(1)
                } else {
                    diag.errorAt(targetValuePos, "waiting for '%v'", target).debug(1)
                }
            }
            if def, ok := v.(*Def); ok && target != v && target != def.value { // trace source Def in diagnostics
                diag.errorOf(def.value, "waiting for def '%v': %v", def.name, def.value).debug(1)
            }
            /*if c := t.closcop; c != nil {
                diag.errorAt(c.position, "waiting closured from %v", c.comment).debug(1)
            }
            if t.isConfigureExecution {
                diag.errorOf(t., "%v: %v = %v", s, t, result)
            }*/
        })
        return
    }

    var (
        optReportFileUpdates  = len(opts) > 0 && opts[0]
        optWaitForExecResult  = len(opts) > 1 && opts[1]
        optStampCurrentTarget = len(opts) > 2 && opts[2]
    )
    if optWaitForExecResult {
        // Waiting for command (shell/python/etc.) exec result
        if bv := t.autoGet("-"); !isNil(bv) && !isNone(bv) {
            var ok bool
            if execRes, ok = bv.(*ExecResult); ok {
                execRes.wg.Wait()
            }
            if target.String() == "tensorflow/core/framework/registration/options.h" {
                diag.infoAt(pos, "%v %v", target, ok).debug(1)
            }
        }
    }
    if !optStampCurrentTarget {
        // done!
    } else if files, err = target.stamp(t); err != nil {
        if p := target.Position(); p.IsValid() { diag.errorAt(p, "%v", err) }
        diag.errorAt(pos, "%v", err).debug(1)
        return
    } else if optReportFileUpdates {
        reportFileUpdates(ctx, t.start, files)
    }
    return
}

// Value represents a value of a type.
type Value interface {
    Positioner // The position where the value appears (or NoPos).

    // Literal representations of the value.
    String() string

    // Strval returns the string form of the value.
    Strval(Context) (string, error)

    // Integer returns the integer form of the value.
    Integer(Context) (int64, error)

    // Float returns the float form of the value.
    Float(Context) (float64, error)

    // Returns true if the value can be evaluated as 'true', 'yes', etc.
    True(Context) (bool, error)

    // Equality compare.
    cmp(Context, Value) cmpres

    // whether this value can be used as a pattern
    patterned(Context) bool

    // Match a Value or string, returned 's' is the matched string (or heading part).
    match(Context, interface{}) (full bool, s string, stems []string)

    // Stencil this value with stems.
    stencil(Context, []string) (s string, rest []string)

    // Recursively detecting whether this value references
    // the object (to avoid loop-delegation).
    refs(Context, Value) bool

    // Returns all defs (of names if specified) used in this value.
    defs(Context, ...string) []*Def

    // Test if this value is expandible for some bits.
    expandible(Context, expandwhat) bool

    // &(...)        -> $(...)
    // $(...)        -> ......
    // $(...)=$(...) -> ...=$(...), ...=...
    // foo->bar      -> ...
    expand(Context, expandwhat) (Value, error) // result is nil or identical to this value if no expansions

    traverse(*traversal) breakers

    stat(*traversal) (*statinfo)

    // Stamp the value if it's a file (aka. update FileInfo).
    stamp(*traversal) ([]*File, error)
}

type elemkind int
const (
    elemNoQuote elemkind = 1<<iota
    elemNoBrace
    elemExpand
)

type elemStrer interface {
    elemStr(ctx Context, o Object, k elemkind) string
}

func elementString(ctx Context, o Object, elem Value, k elemkind) (s string) {
    if p, ok := elem.(elemStrer); ok {
        s = p.elemStr(ctx, o, k)
    } else if elem != nil {
        s = elem.String()
    }
    return
}

type valbase struct { position Position }
func (t *valbase) Position() (res Position) { return t.position }
func (_ *valbase) String() (s string) { return }
func (_ *valbase) Strval(_ Context) (s string, err error) { return }
func (_ *valbase) Integer(_ Context) (i int64, err error) { return }
func (_ *valbase) Float(_ Context) (f float64, err error) { return }
func (_ *valbase) True(_ Context) (res bool, err error) { return }
func (_ *valbase) refs(_ Context, _ Value) (res bool) { return }
func (_ *valbase) defs(_ Context, _ ...string) (res []*Def) { return }
func (_ *valbase) expandible(_ Context, _ expandwhat) bool { return false }
func (_ *valbase) expand(_ Context, _ expandwhat) (v Value, err error) { return }
func (_ *valbase) cmp(_ Context, _ Value) (res cmpres) { return }
func (_ *valbase) patterned(_ Context) bool { return false }
func (_ *valbase) match(_ Context, i interface{}) (full bool, s string, stems []string) { return }
func (_ *valbase) stencil(_ Context, stems []string) (s string, rest []string) { return }
func (_ *valbase) stat(t *traversal) (si *statinfo) { return }
func (_ *valbase) stamp(t *traversal) (file []*File, err error) { return }
func (_ *valbase) traverse(t *traversal) (brks breakers) { return }
func (_ *valbase) _match(ctx Context, p Value, i interface{}) (full bool, s string, stems []string) {
    var ( v string; e error )
    if v, e = p.Strval(ctx); e == nil {
        var is string
        switch t := i.(type) {
        case string: is = t
        case Value:
            if is, e = t.Strval(ctx); e != nil {
                diag.errorOf(t, "strval '%v' error: %v", t, e).debug(1)
                return
            }
        }
        if strings.HasPrefix(is, v) {
            s, full = v, (len(v) == len(is))
        }
    } else {
        diag.errorOf(p, "strval '%v' error: %v", p, e).debug(1)
    }
    return
}
func (_ *valbase) _stencil(ctx Context, p Value, stems []string) (s string, rest []string) {
    var ( v string; e error )
    if v, e = p.Strval(ctx); e == nil {
        s, rest = v, stems
    } else {
        diag.errorOf(p, "strval '%v' error: %v", p, e).debug(1)
    }
    return
}

type returner struct {
    valbase
    Values []Value
}

type Argumented struct {
    value Value
    args []Value
}
func (p *Argumented) Position() Position { return p.value.Position() }
func (p *Argumented) refs(ctx Context, v Value) bool {
    if p.value.refs(ctx, v) { return true }
    for _, a := range p.args {
        if a.refs(ctx, v) { return true }
    }
    return false
}
func (p *Argumented) defs(ctx Context, s ...string) (res []*Def) {
    res = p.value.defs(ctx, s...)
    for _, a := range p.args {
        res = append(res, a.defs(ctx, s...)...)
    }
    return
}
func (p *Argumented) expandible(ctx Context, w expandwhat) (res bool) {
    if res = p.value.expandible(ctx, w); !res && w&expandArgedArgs != 0 {
        for _, a := range p.args {
            if res = a.expandible(ctx, w); res { break }
        }
    }
    return
}
func (p *Argumented) expand(ctx Context, w expandwhat) (res Value, err error) {
    var (
        val Value
        args []Value
        num int
    )
    if val, err = p.value.expand(ctx, w); err != nil {
        diag.errorOf(p.value, "expand '%v' failed: %v", p.value, err).debug(1)
        return
    } else if isNil(val) { val = p.value }
    if w&expandArgedArgs != 0 {
        if args, num, err = expandall1(ctx, w, p.args...); err != nil {
            diag.errorOf(p.value, "expand args '%v' failed: %v", p.args, err).debug(1)
            return
        }
    }
    if val != p.value || num > 0 { res = &Argumented{ val, args }}
    return
}
func (p *Argumented) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Argumented); ok {
        if res = p.value.cmp(ctx, a.value); res == cmpEqual {
            // FIXME: check p.args against a.args?
        }
    }
    return
}
func (p *Argumented) patterned(ctx Context) bool { return p.value.patterned(ctx) }
func (p *Argumented) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p.value.match(ctx, i)
    return
}
func (p *Argumented) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p.value.stencil(ctx, stems)
    return
}

func (p *Argumented) stamp(t *traversal) ([]*File, error) { return p.value.stamp(t) }
func (p *Argumented) stat(t *traversal) (si *statinfo) {
    // FIXME: p.value might be not the real target (depending on the arguments)
    return p.value.stat(t)
}
func (p *Argumented) True(ctx Context) (res bool, err error) {
    if p.value != nil { res, err = p.value.True(ctx) }
    return
}
func (p *Argumented) Integer(ctx Context) (i int64, err error) {
    var s string
    if s, err = p.Strval(ctx); err == nil {
        i, err = strconv.ParseInt(s, 10, 64)
    }
    return
}
func (p *Argumented) Float(ctx Context) (f float64, err error) {
    var s string
    if s, err = p.Strval(ctx); err == nil {
        f, err = strconv.ParseFloat(s, 64)
    }
    return
}
func (p *Argumented) elemStr(ctx Context, o Object, k elemkind) (s string) {
    for i, a := range p.args {
        if i > 0 { s += "," }
        s += elementString(ctx, o, a, k)
    }
    s = fmt.Sprintf("%s(%s)", elementString(ctx, o, p.value, k), s)
    return
}
func (p *Argumented) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *Argumented) Strval(ctx Context) (s string, err error) {
    if s, err = p.value.Strval(ctx); err != nil {
        return
    }
    s += "("
    for i, a := range p.args {
        if i > 0 { s += "," }
        var v string
        if v, err = a.Strval(ctx); err == nil { s += v } else {
            break
        }
    }
    s += ")"
    return
}
func (p *Argumented) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    //!< IMPORTANT! - Don't merge-expand arguments here!
    //!< Arguments should be passed to program.execute as it is.
    defer func(a []Value) { t.arguments = a } (t.arguments)
    t.arguments = p.args
    return p.value.traverse(t)
}

type None struct { valbase }
func (_ *None) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*None); ok { res = cmpEqual }
    return
}

type Nil struct { valbase }
func (p *Nil) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*Nil); ok { res = cmpEqual }
    return
}

func isUndef(v Value) (t bool) { _, t = v.(*unresolvedobject); return }
func isNone(v Value) (t bool) {
    switch a := v.(type) {
    case *None: t = true
    case *List: t = len(a.Elems) == 0 || isNone(a.Elems[0])
    }
    return
}
func isNil(v Value) (t bool) {
    if v == nil {
        t = true
    } else if _, t = v.(*Nil); t {
        // true
    } else if vv := reflect.ValueOf(v); vv.Kind() == reflect.Ptr && vv.IsNil() {
        t = true
    }
    return
}

// Any is used to box an arbitrary value
type Any struct { value interface{} }
func (p *Any) cmp(ctx Context, v Value) (res cmpres) {
    switch a := v.(type) {
    case *Any:
        if p.value == a.value {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            if v2, ok := a.value.(Value); ok {
                res = v1.cmp(ctx, v2)
            }
        }
    case Value:
        if p.value == a {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            res = v1.cmp(ctx, a)
        }
    }
    return
}
func (p *Any) patterned(ctx Context) (res bool) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
       res = v.patterned(ctx)
    }
    return
}
func (p *Any) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        full, s, stems = v.match(ctx, i)
    }
    return
}
func (p *Any) stencil(ctx Context, stems []string) (s string, rest []string) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        s, rest = v.stencil(ctx, stems)
    }
    return
}
func (p *Any) stamp(t *traversal) (files []*File, err error) {
    if a, ok := p.value.(Value); ok { files, err = a.stamp(t) }
    return
}
func (p *Any) stat(t *traversal) (si *statinfo) {
    if v, ok := p.value.(Value); ok && v != nil { si = v.stat(t) }
    return
}
func (p *Any) expand(ctx Context, w expandwhat) (res Value, err error) {
    if val, ok := p.value.(Value); ok && !isNil(val) {
        if res, err = val.expand(ctx, w); err != nil {
            diag.errorOf(p, "expand '%v' failed: %v", val, err).debug(1)
        }
    }
    return
}
func (p *Any) refs(ctx Context, o Value) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.refs(ctx, o) }
    return
}
func (p *Any) defs(ctx Context, s ...string) (res []*Def) {
    if v, ok := p.value.(Value); ok { res = v.defs(ctx, s...) }
    return
}
func (p *Any) expandible(ctx Context, w expandwhat) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.expandible(ctx, w) }
    return
}
func (p *Any) Position() (res Position) {
    if v, ok := p.value.(Positioner); ok { res = v.Position() }
    return
}
func (p *Any) True(ctx Context) (t bool, err error) {
    switch v := p.value.(type) {
    case Value:     t, err = v.True(ctx)
    case float32:   t      = math.Abs(float64(v))-0 >= FloatEpsilon
    case float64:   t      = math.Abs(v)-0 >= FloatEpsilon
    case int64:     t      = v != 0
    case int:       t      = v != 0
    case bool:      t      = v
    }
    return
}
func (p *Any) Float(ctx Context) (res float64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.Float(ctx)
    case float32:     res      = float64(v)
    case float64:     res      = v
    case int:         res      = float64(v)
    case int64:       res      = float64(v)
    case bool: if v { res      = FloatEpsilon }
    }
    return
}
func (p *Any) Integer(ctx Context) (res int64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.Integer(ctx)
    case float32:     res      = int64(v)
    case float64:     res      = int64(v)
    case int:         res      = int64(v)
    case int64:       res      = v
    case bool: if v { res      = 1 }
    }
    return
}
func (p *Any) Strval(ctx Context) (s string, err error) {
    switch v := p.value.(type) {
    case Value:       s, err = v.Strval(ctx)
    case float32:     s      = strconv.FormatFloat(float64(v),'g', -1, 32)
    case float64:     s      = strconv.FormatFloat(float64(v),'g', -1, 64)
    case int:         s      = strconv.FormatInt(int64(v),10)
    case int64:       s      = strconv.FormatInt(int64(v),10)
    case bool: if v { s      = "true" } else { s = "false" }
    default: s = fmt.Sprintf("%s", p.value)
    }
    return
}
func (p *Any) String() string { return fmt.Sprintf("<%v>", p.value) }
func (p *Any) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if v, ok := p.value.(Value); ok { brks = v.traverse(t) }
    return
}

type negative struct { valbase; x Value }
func (p *negative) refs(ctx Context, o Value) bool { return p.x.refs(ctx, o) }
func (p *negative) defs(ctx Context, s ...string) []*Def { return p.x.defs(ctx, s...) }
func (p *negative) expandible(ctx Context, w expandwhat) bool { return p.x.expandible(ctx, w) }
func (p *negative) expand(ctx Context, w expandwhat) (res Value, err error) {
    var val Value
    if val, err = p.x.expand(ctx, w); err != nil {
        diag.errorOf(p.x, "expand '%v' failed: %v", p.x, err).debug(1)
    } else if !isNil(val) && val != p.x {
        res = &negative{p.valbase, val}
    }
    return
}
func (p *negative) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*negative); ok { res = p.x.cmp(ctx, a.x) }
    return
}
func (p *negative) True(ctx Context) (res bool, err error) {
    if p.x != nil { if res, err = p.x.True(ctx); err == nil { res = !res }}
    return
}
func (p *negative) elemStr(ctx Context, o Object, k elemkind) string {
    return `!`+elementString(ctx, o, p.x, k)
}
func (p *negative) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *negative) Strval(ctx Context) (s string, err error) {
    var t bool
    if t, err = p.x.True(ctx); err != nil {
        diag.errorAt(p.position, "truthify '%v' failed: %v", p.x, err).debug(1)
    } else {
        s = fmt.Sprintf("%v", !t)
    }
    return
}
func (p *negative) Float(ctx Context) (res float64, err error) {
    var t bool
    if t, err = p.x.True(ctx); err == nil && !t {
        res = FloatEpsilon
    }
    return
}
func (p *negative) Integer(ctx Context) (res int64, err error) {
    var t bool
    if t, err = p.x.True(ctx); err == nil && !t {
        res = 1
    }
    return
}
func (p *negative) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if p.x != nil { brks = p.x.traverse(t) }
    return
}

func Negative(val Value) *negative { return &negative{valbase{val.Position()},val} }

type boolean struct { valbase; bool }
func (p *boolean) String() (s string) {
    if p.bool { s = "true" } else { s = "false" }
    return
}
func (p *boolean) Strval(_ Context) (string, error) { return p.String(), nil }
func (p *boolean) True(_ Context) (bool, error) { return p.bool, nil }
func (p *boolean) Float(_ Context) (v float64, err error) {
    if p.bool { v = 1. }
    return
}
func (p *boolean) Integer(_ Context) (v int64, err error) {
    if p.bool { v = 1 }
    return
}
func (p *boolean) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*option); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, ok := v.(*answer); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, ok := v.(*boolean); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    }
    return
}
func (p *boolean) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *boolean) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

type answer struct { valbase; bool }
func (p *answer) String() (s string) {
    if p.bool { s = "yes" } else { s = "no" }
    return
}
func (p *answer) Strval(ctx Context) (string, error) { return p.String(), nil }
func (p *answer) True(ctx Context) (bool, error) { return p.bool, nil }
func (p *answer) Float(ctx Context) (v float64, err error) {
    if p.bool { v = 1. }
    return
}
func (p *answer) Integer(ctx Context) (v int64, err error) {
    if p.bool { v = 1 }
    return
}
func (p *answer) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*option); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, ok := v.(*answer); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, ok := v.(*boolean); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    }
    return
}
func (p *answer) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *answer) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

type option struct { valbase; bool }
func (p *option) String() (s string) {
    if p.bool { s = "on" } else { s = "off" }
    return
}
func (p *option) Strval(ctx Context) (string, error) { return p.String(), nil }
func (p *option) True(ctx Context) (bool, error) { return p.bool, nil }
func (p *option) Float(ctx Context) (v float64, err error) {
    if p.bool { v = 1. }
    return
}
func (p *option) Integer(ctx Context) (v int64, err error) {
    if p.bool { v = 1 }
    return
}
func (p *option) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*option); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, ok := v.(*answer); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, ok := v.(*boolean); ok {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    }
    return
}
func (p *option) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *option) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

type prediction struct {
    boolean
    reason string
}

type integer struct {
    valbase
    int64
}
func (p *integer) True(ctx Context) (bool, error) { return p.int64 != 0, nil }
func (p *integer) Integer(ctx Context) (int64, error) { return p.int64, nil }
func (p *integer) Float(ctx Context) (float64, error) { return float64(p.int64), nil }
func (p *integer) cmp(ctx Context, v Value) (res cmpres) {
    var i, e = v.Integer(ctx)
    assert(e == nil, "%T: %v", v, e)
    if p.int64 == i {
        res = cmpEqual
    } else if p.int64 < i {
        res = cmpSmaller
    } else if p.int64 > i {
        res = cmpGreater
    }
    return
}

type Bin struct { integer }
func (p *Bin) String() string { return fmt.Sprintf("0b%s", strconv.FormatInt(int64(p.int64),2)) }
func (p *Bin) Strval(ctx Context) (string, error) { return strconv.FormatInt(int64(p.int64),2), nil }
func (p *Bin) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Bin) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

type Oct struct { integer }
func (p *Oct) String() string { return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8)) }
func (p *Oct) Strval(ctx Context) (string, error) { return strconv.FormatInt(int64(p.int64),8), nil }
func (p *Oct) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Oct) stencil(ctx Context, stems []string) (s string, rest []string) { return p._stencil(ctx, p, stems) }

type Int struct { integer }
func (p *Int) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) Strval(ctx Context) (string, error) { return strconv.FormatInt(int64(p.int64),10), nil }
func (p *Int) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Int) stencil(ctx Context, stems []string) (s string, rest []string) { return p._stencil(ctx, p, stems) }

type Hex struct { integer }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *Hex) Strval(ctx Context) (string, error) { return strconv.FormatInt(int64(p.int64),16), nil }
func (p *Hex) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Hex) stencil(ctx Context, stems []string) (s string, rest []string) { return p._stencil(ctx, p, stems) }

const FloatEpsilon = 1e-15 /* 1e-16 */
type Float struct { valbase; float64 } // IEEE-754 64-bit binary floating-point
func (p *Float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) Strval(ctx Context) (string, error) { return strconv.FormatFloat(float64(p.float64),'g', -1, 64), nil }
func (p *Float) True(ctx Context) (bool, error) { return math.Abs(p.float64)-0 > FloatEpsilon, nil }
func (p *Float) Integer(ctx Context) (int64, error) { return int64(p.float64), nil }
func (p *Float) Float(ctx Context) (float64, error) { return p.float64, nil }
func (p *Float) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*Float); ok {
        f, e := v.Float(ctx)
        assert(e == nil, "%T: %v", v, e)
        if p.float64 == f {
            res = cmpEqual
        } else if p.float64 < f {
            res = cmpSmaller
        } else if p.float64 > f {
            res = cmpGreater
        }
    }
    return
}
func (p *Float) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Float) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

type DateTime struct {
    valbase
    t time.Time
}
func (p *DateTime) String() string { return time.Time(p.t).Format("2006-01-02T15:04:05.999999999Z07:00") }
func (p *DateTime) Strval(ctx Context) (string, error) { return p.String(), nil } // time.RFC3339Nano
func (p *DateTime) True(ctx Context) (bool, error) { return !p.t.IsZero(), nil }
func (p *DateTime) Integer(ctx Context) (int64, error) { return p.t.Unix(), nil }
func (p *DateTime) Float(ctx Context) (float64, error) { i, e := p.Integer(ctx); return float64(i), e }
func (p *DateTime) cmp(ctx Context, v Value) (res cmpres) {
    var vt time.Time
    switch a := v.(type) {
    case *DateTime: vt = a.t
    case *Date:     vt = a.t
    case *Time:     vt = a.t
    default: return
    }
    if p.t.Equal(vt) {
        res = cmpEqual
    } else if p.t.Before(vt) {
        res = cmpSmaller
    } else /*if p.t.After(vt)*/ {
        res = cmpGreater
    }
    return
}
func (p *DateTime) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *DateTime) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

func ParseDateTime(pos Position, s string) *DateTime {
    // time.RFC3339Nano
    if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
        return &DateTime{valbase{pos},t}
    } else {
        panic(e)
    }
}

type Date struct { DateTime }
func (p *Date) String() string { return time.Time(p.t).Format("2006-01-02") }
func (p *Date) Strval(ctx Context) (string, error) { return p.String(), nil }
func (p *Date) Integer(ctx Context) (int64, error) { return p.t.Unix(), nil }
func (p *Date) Float(ctx Context) (float64, error) { i, e := p.Integer(ctx); return float64(i), e }
func (p *Date) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Date) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

type Time struct { DateTime }
func (p *Time) String() string { return time.Time(p.t).Format("15:04:05.999999999Z07:00") }
func (p *Time) Strval(ctx Context) (string, error) { return p.String(), nil }
func (p *Time) Integer(ctx Context) (int64, error) { return p.t.Unix(), nil }
func (p *Time) Float(ctx Context) (float64, error) { i, e := p.Integer(ctx); return float64(i), e }
func (p *Time) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Time) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//        └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//             └<userinfo>─(@)┘      └(:)─<port>┘
type URL struct {
    valbase
    Scheme Value
    Username Value
    Password Value
    Host Value
    Port Value
    Path Value
    Query Value
    Fragment Value
}
func (p *URL) String() string { return p.elemStr(nil, nil, 0) }
func (p *URL) True(ctx Context) (t bool, e error) {
    if p.Scheme != nil { if t, e = p.Scheme.True(ctx); t { return }}
    if p.Host   != nil { if t, e = p.Host  .True(ctx); t { return }}
    if p.Path   != nil { if t, e = p.Path  .True(ctx); t { return }}
    return //p.String() != "", nil
}
func (p *URL) Strval(ctx Context) (s string, err error) {
    if s, err = p.Scheme.Strval(ctx); err != nil { return }
    if s += ":"; p.Host != nil && !isNone(p.Host) {
        var host string
        if host, err = p.Host.Strval(ctx); err != nil { return }
        s += "//"
        if p.Username != nil && !isNone(p.Username) {
            var user string
            if user, err = p.Username.Strval(ctx); err != nil { return }
            s += user
            if p.Password != nil {
                var pass string
                s += ":"
                if pass, err = p.Password.Strval(ctx); err != nil { return }
                s += pass
            }
            s += "@"
        }
        s += host
        if p.Port != nil && !isNone(p.Port) {
            var port string
            if port, err = p.Port.Strval(ctx); err != nil { return }
            s += ":" + port
        }
    }
    if p.Path != nil && !isNone(p.Path) {
        var path string
        if path, err = p.Path.Strval(ctx); err != nil { return }
        //if !strings.HasPrefix(path, PathSep) { s += PathSep }
        s += path
    }
    if p.Query != nil && !isNone(p.Query) {
        var query string
        if query, err = p.Query.Strval(ctx); err != nil { return }
        s += "?" + query
    }
    if p.Fragment != nil && !isNone(p.Fragment) {
        var fragment string
        if fragment, err = p.Fragment.Strval(ctx); err != nil { return }
        s += "#" + fragment
    }
    return
}
func (p *URL) Integer(ctx Context) (i int64, err error) {
    var s string
    if s, err = p.Strval(ctx); err == nil {
        i = int64(len(s))
    }
    return
}
func (p *URL) Float(ctx Context) (float64, error) { i, e := p.Integer(ctx); return float64(i), e }
func (p *URL) elemStr(ctx Context, o Object, k elemkind) (s string) {
    if s = elementString(ctx, o, p.Scheme, k); s == "" { return }
    if s += ":"; p.Host == nil {
        // ...
    } else if _, ok := p.Host.(*None); ok {
        var host string
        if host = elementString(ctx, o, p.Host, k); host == "" { return }
        s += "//"
        if p.Username == nil {
            // ...
        } else if isNone(p.Username) {
            var user string
            if user = elementString(ctx, o, p.Username, k); user != "" {
                s += user + "@"
            }
        }
        s += host
        if p.Port == nil {
            // ...
        } else if _, ok := p.Port.(*None); ok {
            var port string
            if port = elementString(ctx, o, p.Port, k); port != "" {
                s += ":" + port
            }
        }
    }
    if p.Path == nil {
        // ...
    } else if _, ok := p.Path.(*None); ok {
        var path string
        if path = elementString(ctx, o, p.Path, k); path != "" {
            //if !strings.HasPrefix(path, PathSep) { s += PathSep }
            s += path
        }
    }
    if p.Query == nil {
        // ...
    } else if _, ok := p.Query.(*None); ok {
        var query string
        if query = elementString(ctx, o, p.Query, k); query != "" {
            s += "?" + query
        }
    }
    if p.Fragment == nil {
        // ...
    } else if _, ok := p.Fragment.(*None); ok {
        var fragment string
        if fragment = elementString(ctx, o, p.Fragment, k); fragment != "" {
            s += "#" + fragment
        }
    }
    return
}
func (p *URL) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*URL); ok {
        if p.Scheme == nil || a.Scheme == nil { return }
        if p.Scheme.cmp(ctx, a.Scheme) != cmpEqual { return }
        if p.Username != nil {
            if a.Username == nil { return }
            if p.Username.cmp(ctx, a.Username) != cmpEqual { return }
        }
        if p.Password != nil {
            if a.Password == nil { return }
            if p.Password.cmp(ctx, a.Password) != cmpEqual { return }
        }
        if p.Host != nil {
            if a.Host == nil { return }
            if p.Host.cmp(ctx, a.Host) != cmpEqual { return }
        }
        if p.Port != nil {
            if a.Port == nil { return }
            if p.Port.cmp(ctx, a.Port) != cmpEqual { return }
        }
        if p.Path != nil {
            if a.Path == nil { return }
            if p.Path.cmp(ctx, a.Path) != cmpEqual { return }
        }
        if p.Query != nil {
            if a.Query == nil { return }
            if p.Query.cmp(ctx, a.Query) != cmpEqual { return }
        }
        if p.Fragment != nil {
            if a.Fragment == nil { return }
            if p.Fragment.cmp(ctx, a.Fragment) != cmpEqual { return }
        }
        res = cmpEqual
    }
    return
}
func (p *URL) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *URL) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}
func (p *URL) Validate() (res *url.URL) {
    panic(fmt.Sprintf("validate %s", p))
    return
}

type Raw struct { valbase; string }
func (p *Raw) String() string { return p.string }
func (p *Raw) Strval(ctx Context) (string, error) { return p.string, nil }
func (p *Raw) True(ctx Context) (bool, error) { return p.string != "", nil }
func (p *Raw) Integer(ctx Context) (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Raw) Float(ctx Context) (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Raw) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Raw); ok && p.string == a.string {
        res = cmpEqual
    }
    return
}
func (p *Raw) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Raw) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}

type String struct { valbase; string }
func (p *String) String() string { return p.elemStr(nil, nil, 0) }
func (p *String) Strval(ctx Context) (string, error) { return strings.Replace(p.string, "\\\"", "\"", -1), nil }
func (p *String) True(ctx Context) (bool, error) { return p.string != "", nil }
func (p *String) Integer(ctx Context) (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *String) Float(ctx Context) (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *String) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*String); ok {
        if p.string == a.string {
            res = cmpEqual
        } else if p.string < a.string {
            res = cmpSmaller
        } else /*if p.string > a.string*/ {
            res = cmpGreater
        }
    }
    return
}
func (p *String) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *String) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}
func (p *String) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    _, brks = t.string(p, p.string)
    return
}
func (p *String) elemStr(_ Context, o Object, k elemkind) (s string) {
    if k&elemNoQuote == 0 { s = `'`+p.string+`'` } else { s = p.string }
    return
}

func isTrueString(s string) (t bool) {
    switch strings.ToLower(s) {
    case "false", "no" , "off", "force_off", "0", "": t = false
    case "true" , "yes", "on" , "force_on" , "1": t = true
    default: t = true
    }
    return
}

type Bareword struct { valbase; string }
func (p *Bareword) String() string { return p.string }
func (p *Bareword) Strval(ctx Context) (string, error) { return p.string, nil }
func (p *Bareword) True(ctx Context) (bool, error) { return isTrueString(p.string), nil }
func (p *Bareword) Integer(ctx Context) (int64, error) { return strconv.ParseInt(p.string, 10, 64) }
func (p *Bareword) Float(ctx Context) (float64, error) { return strconv.ParseFloat(p.string, 64) }
func (p *Bareword) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Bareword); ok {
        if p.string == a.string {
            res = cmpEqual
        } else if p.string > a.string {
            res = cmpSmaller
        } else if p.string < a.string {
            res = cmpGreater
        }
    }
    return
}
func (p *Bareword) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Bareword) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}
func (p *Bareword) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    _, brks = t.string(p, p.string)
    return
}

type Qualiword struct { valbase; words []string } // foo.bar.zar, foo.&(bar).zar
func (p *Qualiword) String() string { return strings.Join(p.words,".") }
func (p *Qualiword) Strval(ctx Context) (string, error) { return p.String(), nil }
func (p *Qualiword) True(ctx Context) (bool, error) { return len(p.words)!=0, nil }
func (p *Qualiword) Integer(ctx Context) (int64, error) { return int64(len(p.words)), nil }
func (p *Qualiword) Float(ctx Context) (float64, error) { return float64(len(p.words)), nil }
func (p *Qualiword) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Qualiword); ok {
        var n int
        var al, pl = len(a.words), len(p.words)
        for i, w := range p.words {
            if al <= i {
                break
            } else if w == a.words[n] {
                if n += 1; n == al && al == pl {
                    res = cmpEqual
                } else {
                    continue
                }
            } else if w > a.words[n] {
                res = cmpSmaller // cmpGreater??
            } else {
                res = cmpGreater // cmpSmaller??
            }
            break
        }
    }
    return
}
func (p *Qualiword) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Qualiword) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}
func (p *Qualiword) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    _, brks = t.string(p, p.String())
    return
}

type elements struct { Elems []Value }
func (p *elements) Len() int                { return len(p.Elems) }
func (p *elements) Append(v... Value)       { p.Elems = append(p.Elems, v...) }
func (p *elements) Get(n int) (v Value)     { if n>=0 && n<len(p.Elems) { v = p.Elems[n] }; return }
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
func (p *elements) ToBarecomp(pos Position) *Barecomp { return &Barecomp{valbase{pos},*p} }
func (p *elements) ToCompound(pos Position) *Compound { return &Compound{valbase{pos},*p} }
func (p *elements) ToList(pos Position) *List { return &List{pos, *p} }
func (p *elements) True(ctx Context) (t bool, err error) { // (or elems...)
    for _, elem := range p.Elems {
        if isNil(elem) {
            continue
        } else if t, err = elem.True(ctx); err != nil {
            diag.errorOf(elem, "truthify '%v' failed: %v", elem, err).debug(1)
            break
        } else if t { break }
    }
    return
}
func (p *elements) refs(ctx Context, v Value) bool {
    for _, elem := range p.Elems {
        if elem != nil && (elem == v || elem.refs(ctx, v)) {
            return true
        }
    }
    return false
}
func (p *elements) defs(ctx Context, s ...string) (res []*Def) {
    for _, elem := range p.Elems { res = append(res, elem.defs(ctx, s...)...) }
    return
}
func (p *elements) expandible(ctx Context, w expandwhat) (res bool) {
    for _, elem := range p.Elems {
        if res = elem.expandible(ctx, w); res { break }
    }
    return
}
func (p *elements) cmpElems(ctx Context, elems []Value) (res cmpres) {
    if len(p.Elems) == len(elems) {
        for i, elem := range p.Elems {
            if elem == nil { continue } else
            if other := elems[i]; other == nil { continue } else
            if elem.cmp(ctx, other) != cmpEqual { return cmpUnknown }
        }
        res = cmpEqual
    }
    return
}

type Barecomp struct { valbase ; elements }
func (p *Barecomp) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *Barecomp) Strval(ctx Context) (s string, e error) {
    for _, elem := range p.Elems {
        var v string
        if elem == nil { continue } else
        if v, e = elem.Strval(ctx); e == nil { s += v } else { break }
    }
    return
}
func (p *Barecomp) True(ctx Context) (bool, error) { return p.elements.True(ctx) }
func (p *Barecomp) Integer(ctx Context) (res int64, err error) {
    if len(p.Elems) == 2 {
        if i, ok := p.Elems[0].(*Int); ok {
            var n = i.int64
            if w, ok := p.Elems[1].(*Bareword); ok {
                if (w.string == "st" && n%1 == 0) ||
                    (w.string == "nd" && n%2 == 0) ||
                    (w.string == "rd" && n%3 == 0) ||
                    (w.string == "th") { res = n }
            }
        }
    }
    return
}
func (p *Barecomp) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *Barecomp) defs(ctx Context, s ...string) []*Def { return p.elements.defs(ctx, s...) }
func (p *Barecomp) elemStr(ctx Context, o Object, k elemkind) (s string) {
    for _, elem := range p.Elems {
        s += elementString(ctx, o, elem, k)
    }
    return
}
func (p *Barecomp) expandible(ctx Context, w expandwhat) bool { return p.elements.expandible(ctx, w) }
func (p *Barecomp) expand(ctx Context, w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(ctx, w, p.Elems...); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", p, err).debug(1)
    } else if num > 0 {
        res = &Barecomp{p.valbase, elements{elems}}
    }
    return
}
func (p *Barecomp) traverse(t *traversal) (brks breakers) {
    var ctx = positional(t, p.Position())
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if target, err := p.Strval(ctx); err == nil {
        if false { diag.warnAt(p.position, "%v (%s)", p, target).debug(1) }
        _, brks = t.string(p, target)
    } else {
        diag.errorOf(p, "strval '%v' error: %v", p, err).debug(1)
    }
    return
}
func (p *Barecomp) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Barecomp); ok { res = p.cmpElems(ctx, a.Elems) }
    return
}
func (p *Barecomp) patterned(ctx Context) (res bool) {
    for _, elem := range p.Elems {
        if res = elem.patterned(ctx); res { break }
    }
    return
}
func (p *Barecomp) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Barecomp) stencil(ctx Context, stems []string) (s string, rest []string) {
    s, rest = p._stencil(ctx, p, stems)
    return
}
func (p *Barecomp) Combine(x Value) {
    if o, ok := x.(*Barecomp); ok {
        for _, elem := range o.Elems {
            p.Combine(elem)
        }
    } else {
        p.Elems = append(p.Elems, x)
    }
}

// Barefile works like an alias of a File, the Strval() is identical to File.
type Barefile struct {
    valbase
    Name Value
    File *File
}
func (p *Barefile) True(ctx Context) (t bool, err error) {
    if p.File != nil { t, err = p.File.True(ctx) }
    return
}
func (p *Barefile) String() string { return p.elemStr(nil, nil, 0) }
func (p *Barefile) Strval(ctx Context) (string, error) {
    if p.File != nil {
        return p.File.Strval(ctx)
    } else {
        return p.Name.Strval(ctx)
    }
}
func (p *Barefile) Integer(ctx Context) (res int64, err error) {
    if p.File.exists() { res = p.File.info.Size() }
    return
}
func (p *Barefile) Float(ctx Context) (float64, error) {
    i, e := p.Integer(ctx)
    return float64(i), e
}
func (p *Barefile) refs(ctx Context, v Value) bool { return p.Name.refs(ctx, v) }
func (p *Barefile) defs(ctx Context, s ...string) []*Def { return p.Name.defs(ctx, s...) }
func (p *Barefile) elemStr(ctx Context, o Object, k elemkind) (s string) { return elementString(ctx, o, p.Name, k) }
func (p *Barefile) expandible(ctx Context, w expandwhat) bool { return p.Name.expandible(ctx, w) }
func (p *Barefile) expand(ctx Context, w expandwhat) (res Value, err error) {
    if w&expandFullName != 0 {
        var file Value
        if p.File == nil {
            return
        } else if file, err = p.File.expand(ctx, w); err != nil {
            diag.errorOf(p.File, "expand '%v' failed: %v", p.File, err).debug(1)
            return
        } else if !isNil(file) && file != p.File {
            return file, nil
        }
    }

    var name Value
    if name, err = p.Name.expand(ctx, w); err != nil {
        diag.errorOf(p.Name, "expand '%v' failed: %v", p.Name, err).debug(1)
    } else if !isNil(name) && name != p.Name {
        res = &Barefile{p.valbase, name, p.File}
    }
    return
}
func (p *Barefile) traverse(t *traversal) (brks breakers) {
    var ctx = positional(t, p.position)
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if p.File == nil { // it happens if p.Name refers argument
        var ( target string; err error )
        if target, err = p.Strval(ctx); err != nil {
            diag.errorOf(p, "strval '%v' failed: %v", p, err).debug(1)
            return
        }

        var okay bool
        okay, err = t.forClosuredProjects(func(project *Project) (bool, error) {
            p.File = project.FindFile(ctx, target)
            return p.File != nil, nil
        })
        if !okay || p.File == nil {
            diag.errorAt(p.position, "barefile '%s' not found", target).debug(1)
            return
        }
    }
    if p.File != nil { brks = p.File.traverse(t) } else {
        diag.errorAt(p.position, "barefile '%s' is nil", p).debug(1)
    }
    return
}
func (p *Barefile) stamp(t *traversal) (files []*File, err error) {
    if p.File != nil { files, err = p.File.stamp(t) }
    return
}
func (p *Barefile) stat(t *traversal) (si *statinfo) {
    if p.File != nil { si = p.File.stat(t) }
    return
}
func (p *Barefile) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Barefile); ok { res = p.Name.cmp(ctx, a.Name) }
    return
}
func (p *Barefile) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    if p.File != nil {
        full, s, stems = p.File.match(ctx, i)
    } else {
        full, s, stems = p.Name.match(ctx, i)
    }
    return
}
func (p *Barefile) stencil(ctx Context, stems []string) (s string, rest []string) {
    if p.File != nil {
        s, rest = p.File.stencil(ctx, stems)
    } else {
        s, rest = p.Name.stencil(ctx, stems)
    }
    return
}

type GlobMeta struct { valbase ; token.Token }
func (p *GlobMeta) String() string { return p.Token.String() }
func (p *GlobMeta) Strval(ctx Context) (string, error) { return p.Token.String(), nil }
func (p *GlobMeta) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*GlobMeta); ok && p.Token == a.Token {
        res = cmpEqual
    }
    return
}

// `[a-b]`, `[abc]`, ...
// `a-b`, `abc`, `a$(var)c`, `a$(spaces)c`...
type GlobRange struct { valbase ; Chars Value }
func (p *GlobRange) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *GlobRange) Strval(ctx Context) (s string, err error) {
    var chars string
    if chars, err = p.Chars.Strval(ctx); err == nil {
        s = fmt.Sprintf("[%s]", chars)
    }
    return
}
func (p *GlobRange) refs(ctx Context, v Value) bool { return p.Chars.refs(ctx, v) }
func (p *GlobRange) defs(ctx Context, s ...string) []*Def { return p.Chars.defs(ctx, s...) }
func (p *GlobRange) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*GlobRange); ok { res = p.Chars.cmp(ctx, a.Chars) }
    return
}
func (p *GlobRange) expandible(ctx Context, w expandwhat) bool { return p.Chars.expandible(ctx, w) }
func (p *GlobRange) expand(ctx Context, w expandwhat) (res Value, err error) {
    var val Value
    if val, err = p.Chars.expand(ctx, w); err != nil {
        diag.errorOf(p.Chars, "expand '%v' failed: %v", p.Chars, err).debug(1)
    } else if !isNil(val) && val != p.Chars {
        res = &GlobRange{p.valbase, val}
    }
    return
}
func (p *GlobRange) elemStr(ctx Context, o Object, k elemkind) (s string) {
    return fmt.Sprintf("[%s]", elementString(ctx, o, p.Chars, k))
}

// Path is addressing a file (dynamically), the real located file varies
// base on 'elements' and the context.
type Path struct { valbase ; elements }
func (p *Path) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *Path) Strval(ctx Context) (s string, e error) {
    for i, seg := range p.Elems {
        if seg == nil {
            e = fmt.Errorf("`%s` nil path segment", p)
            return
        }

        var v string
        if isUndef(seg) {
            diag.errorAt(seg.Position(), "undef path segment (%T)", seg)
            diag.errorAt(ctx.Position(), "… from this context: %s", ctx).debug(16)
            return
        } else if v, e = seg.Strval(ctx); e != nil {
            diag.errorAt(seg.Position(), "%v: %v", seg, e).debug(1)
            return
        } else if i > 0 {
            s += PathSep + v
        } else if v != "" {
            s += v
        } else if len(p.Elems) == 1 {
            s += PathSep
        }
    }
    return
}
func (p *Path) True(ctx Context) (t bool, err error) {
    // FIXME: return p.exists() ??
    for _, elem := range p.Elems {
        if t, err = elem.True(ctx); err != nil {
            diag.errorAt(p.position, "truthify path element '%v' failed: %v", elem, err).debug(1)
        } else if t { break }
    }
    return
}
func (p *Path) refs(ctx Context, v Value) (res bool) { return p.elements.refs(ctx, v) }
func (p *Path) defs(ctx Context, s ...string) (res []*Def) { return p.elements.defs(ctx, s...) }
func (p *Path) expandible(ctx Context, w expandwhat) bool { return p.elements.expandible(ctx, w) }
func (p *Path) expand(ctx Context, w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandPathElems(ctx, w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand path elems failed: %v", err).debug(1)
    } else if num > 0 {
        res = &Path{p.valbase, elements{elems}}
    }
    return
}
func (p *Path) pathname(ctx Context, stems []string) (pathname string, err error) {// the addressed file target
    var rest []string // unmatched path segmants
    if len(stems) == 0 {
        if pathname, err = p.Strval(ctx); err != nil {
            diag.errorAt(p.position, "strval '%v' failed: %v", p, err).debug(1)
        }
    } else if pathname, rest = p.stencil(ctx, stems); len(rest) > 0 {
        //err = errorf(p.position, "partial match: %v", rest)
    }
    return
}
func (p *Path) stamp(t *traversal) (files []*File, err error) {
    var ctx = positional(t, p.position)
    var pathname string
    if pathname, err = p.Strval(ctx); err == nil {
        if pathname == "" {
            diag.errorOf(p, "no pathname for `%s`", p)
        } else if file := stat(ctx,pathname,"","",nil); file != nil {
            if files, err = file.stamp(t); err != nil {
                diag.errorOf(p, "stamp: %v (%v)", err, file)
            }
        }
    }
    return
}
func (p *Path) stat(t *traversal) (si *statinfo) {
    var (
        ctx = positional(t, p.position)
        pathname string // the addressed file target
        err error
    )
    if pathname, err = p.pathname(ctx, t.stems); err != nil {
        diag.errorAt(p.position, "pathname error: %v", err)
    } else if pathname == "" {
        diag.errorAt(p.position, "pathname is empty: %v", p)
    } else if file := stat(ctx, pathname, "", "", nil); file != nil {
        si = &statinfo{ file: file }
    }
    return
}
func (p *Path) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }

    var (
        ctx = positional(t, p.position)
        pathname string
        err error
    )
    if p.patterned(ctx) && len(t.stems) == 0 {
        diag.errorAt(p.position, "empty stems to traverse pattern: %v", p).debug(8)
        return
    } else if pathname, err = p.pathname(ctx, t.stems); err == nil && pathname == "" {
        diag.errorAt(p.position, "path matches no target: %v", p).debug(1)
        return
    } else if err != nil {
        diag.errorAt(p.position, "compute pathname failed: %v", err).debug(1)
        return
    }

    var okay bool
    // Stat the file by pathname.
    if file := stat(ctx, pathname, "", ""/*, nil*/); file != nil {
        file.traverse(t)
    } else if okay, brks = t.string(p, pathname); !okay && len(t.stems) > 0 {
        if false { brks.add(p.position, breakNext).scope = breakTrave }
    }
    return
}
func (p *Path) patterned(ctx Context) (result bool) {
    for _, seg := range p.Elems {
        if result = seg.patterned(ctx); result { break }
    }
    return
}
func (p *Path) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Path); ok { res = p.cmpElems(ctx, a.Elems) }
    return
}
func (p *Path) elemStr(ctx Context, o Object, k elemkind) (s string) {
    for i, elem := range p.Elems {
        var v = elementString(ctx, o, elem, k)
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

func expandPathElems(ctx Context, w expandwhat, elems ...Value) (res []Value, num int, err error) {
    var pos Position = ctx.Position()
    var xelems []Value
    if xelems, num, err = expandall1(ctx, w, elems...); err != nil {
        diag.errorAt(pos, "expand path elems failed: %v", err).debug(1)
        return
    }
    for _, elem := range xelems {
        if p, ok := elem.(*Path); ok {
            var ( ev []Value; n int )
            if ev, n, err = expandPathElems(ctx, w, p.Elems...); err != nil {
                diag.errorOf(elem, "expand sub path '%v' failed: %v", elem, err).debug(1)
                return
            }
            res = append(res, ev...)
            num += n
        } else {
            res = append(res, elem)
        }
    }
    if w&expandPathStr != 0 {
        var vals []Value
        for _, elem := range res {
            var s string
            switch v := elem.(type) {
            case *String:
                if v.string != "" {
                    vals = append(vals, splitPathStr(v.position, v.string)...)
                }
                num += 1
            case *Compound:
                if s, err = v.Strval(ctx); err != nil {
                    diag.errorAt(v.position, "strval '%v' failed: %v", v, err).debug(1)
                    return
                } else if s != "" {
                    vals = append(vals, splitPathStr(v.position, s)...)
                }
                num += 1
            default:
                vals = append(vals, elem)
            }
        }
        res = vals
    }
    return
}

func (p *Path) match1(ctx Context, str string) (full bool, result string, stems []string) {
    var (
        srcs []string
        segs []Value
        err error
    )
    if srcs = strings.Split(str, PathSep); len(srcs) == 0 {
        //diag.errorAt(p.position, "empty: %v", str)
        return
    }
    if segs, _, err = expandPathElems(ctx, expandPlainValue, p.Elems...); err != nil {
        diag.errorAt(p.position, "failed to expand path '%v': %v", p, segs).debug(1)
        return
    }

    const info = false
    //var info = strings.Contains(p.String(), "...") && strings.Contains(str, "...")
    //var ps, _ = p.Strval(ctx)
    //var info = strings.Contains(ps, "...") || strings.Contains(str, "...")
    //var info = strings.Contains(ps, "%") && strings.Contains(str, "%")
    //var info = strings.Contains(str, "/Volumes/workspace/external/google/tensorflow/tensorflow/core/util")
    //var info = strings.Contains(p.String(), "/Volumes/workspace/external/google/tensorflow/%%util")
    //var info = false ||
    //    strings.HasPrefix(p.String(), "/Volumes/workspace/.out/x86_64-apple-darwin-extbit/bootstrap/lib/python") ||
    //    strings.HasPrefix(str, "/Volumes/workspace/.out/x86_64-apple-darwin-extbit/bootstrap/lib/python")
    //var info = strings.HasSuffix(p.String(), "/%%.py") && strings.HasSuffix(str, "__future__.py")
    var (
        lenSegs = len(segs)
        lenSrcs = len(srcs)
        lastSuf Value
        res []string
    )
SegsLoop:
    for n, m := 0, 0; n < lenSegs && m < lenSrcs; {
        var si, seg, src = n, segs[n], srcs[m]
        if cs := correctPathSegForMatch(seg); cs != nil { seg = cs } else {
            diag.errorOf(seg, "invalid path seg: %v (%T)", seg, seg).debug(1)
            break SegsLoop
        }
        var (
            pp, pre, suf = percperc(seg)
            ss []string
            s    string
            f    bool
        )
        if pp {
            if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) src=%v pp=%v pre=%v suf=%v res=%v stems=%v srcs[%d]=%s lenSegs=%d",
                si, p, seg, seg, src, pp, pre, suf, res, stems, m, src, lenSegs).debug(true,1) }
            if !isNil(lastSuf) && !isNil(pre) && !isNone(pre) {
                diag.errorOf(seg, "the continual %%/%% makes no sense")
                break SegsLoop
            }
            if /*ps, ok := seg.(*PathSeg);*/ src == "" /*&& ok && (ps.rune == 0 || ps.rune == '/')*/ { // for root path seg '/'
                // NOTE: seg could also be % or %% here
                res = append(res, "") // for '/'
                stems = append(stems, "")
                //diag.infoOf(p, "%v %v; %v %v; %v %v", p, seg, res, stems, ps, ok)
            }
            lastSuf = suf
            n += 1 // move forward to the next seg
            m += 1 // move forward to the next src
        } else if f, s, ss = seg.match(ctx, src); f || s == src {
            if info { diag.infoOf(p, "%d: path=%v seg=%v (%T); str=%v srcs[%d]=%v -> f=%v s=%v ss=%v => res=%v stems=%v",
                si, p, seg, seg, str, m, src, f, s, ss, res, stems).debug(true,1) }
            // NOTE: `s` could be empty string, e.g. when `str` is absolute path
            res   = append(res  , s)
            stems = append(stems, ss...)
            lastSuf = nil
            n += 1 // move forward to the next seg
            m += 1 // move forward to the next src
            if f { continue SegsLoop } else { break SegsLoop }
        } else {
            if ps, ok := seg.(*PathSeg); (s == "" && ok && ps.rune == 0) || s != "" {
                res = append(res, s)
            } else {
                res, stems = nil, nil
            }
            if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v f=%v s=%s ss=%v str=%s src=%s lenSegs=%d",
                si, p, seg, seg, res, stems, f, s, ss, str, src, lenSegs).debug(true,1) }
            break SegsLoop
        }

        var prefix string
        if !(isNil(pre) || isNone(pre)) {
            if prefix, err = pre.Strval(ctx); err != nil {
                diag.errorOf(pre, "strval prefix '%v' failed: %v", pre, err)
                return
            } else if !strings.HasPrefix(src, prefix) {
                if info { diag.infoOf(p, "%d: seg=%v (%T) pp=%v pre=%v suf=%v res=%v stems=%v src=%s",
                    si, seg, seg, pp, pre, suf, res, stems, src).debug(true,1) }
                break SegsLoop
            }
        }

        // Iterate segs for %%, e.g. bar, baz in foo/%%/bar/baz
        var stem []string
        if prefix != "" { stem = append(stem, strings.TrimPrefix(src, prefix)) }
        if !(isNil(suf) || isNone(suf)) {
            var suffix string
            if suffix, err = suf.Strval(ctx); err != nil {
                diag.errorOf(suf, "strval suffix '%v' failed: %v", suf, err)
                break SegsLoop
            }
            res = append(res, src)
            if m < lenSrcs {
                stem = append(stem, src)
                for ; m < lenSrcs; m += 1 {
                    src = srcs[m]
                    res = append(res, src)
                    if strings.HasSuffix(src, suffix) {
                        stem = append(stem, strings.TrimSuffix(src, suffix))
                        stems = append(stems, strings.Join(stem, PathSep))
                        if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d",
                            si, p, seg, seg, res, stems, suffix, src, lenSegs).debug(true,1) }
                        n += 1 // continue for next seg
                        continue SegsLoop
                    } else {
                        stem = append(stem, src)
                    }
                }
            } else {
                stem = append(stem, strings.TrimSuffix(src, suffix))
            }
            if len(stem) > 0 {
                stems = append(stems, strings.Join(stem, PathSep))
                if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d lenSrcs=%d m=%d",
                    si, p, seg, seg, res, stems, suffix, src, lenSegs, lenSrcs, m).debug(true,1) }
            }
        } else if n < lenSegs && !(isNil(segs[n]) || isNone(segs[n])) {
            for seg = segs[n]; m < lenSrcs; m += 1 {
                src = srcs[m]
                if matched, s, ss := seg.match(ctx, src); matched || s == src {
                    res = append(res, s)
                    if s == "" && len(ss) == 0 {
                        stem = append(stem, s)
                    } else if false {
                        stem = append(stem, ss...)
                    }
                    if si == 0 && len(stems) == 1 && stems[0] == "" { // heading %% matched root '/'
                        stem = append(stems, stem...) // for the root '/'
                        stems[0] = strings.Join(stem, PathSep)
                    } else {
                        stems = append(stems, strings.Join(stem, PathSep))
                    }
                    if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v matched=%v s=%s ss=%v src=%s lenSegs=%d",
                        si, p, seg, seg, res, stems, matched, s, ss, src, lenSegs).debug(true,1) }
                    n += 1 // continue for next seg
                    continue SegsLoop
                } else {
                    res = append(res, src)
                    stem = append(stem, src)
                }
            }
            if len(stem) > 0 {
                stems = append(stems, strings.Join(stem, PathSep))
                if info { diag.infoOf(p, "%d: path=%v seg=%v (%T) res=%v stems=%v s=%s ss=%v src=%s lenSegs=%d",
                    si, p, seg, seg, res, stems, s, ss, src, lenSegs).debug(true,1) }
            }
        } else { // the tailing %%
            if /*m < lenSrcs*/true {
                var rest = srcs[m-1:]
                res = append(res, rest...)
                stem = append(stem, rest...)
                stems = append(stems, strings.Join(stem, PathSep))
            }
            if info { diag.infoOf(p, "%d: seg=%v pre=%v suf=%v res=%v stems=%v src=%s",
                si, seg, pre, suf, res, stems, src).debug(true,1) }
            break SegsLoop
        }
        if info && n == lenSegs {
            diag.infoOf(p, "%d: path=%v seg=%v (%T) str=%v src=%v -> f=%v s=%v ss=%v -> res=%v stems=%v stem=%v m=%d/%d 3.lengSegs=%d",
                si, p, seg, seg, str, src, f, s, ss, res, stems, stem, m, lenSrcs, lenSegs).debug(true, 1)
        }
    }
    if lenRes := len(res); lenRes > 0 { // full or partial matched
        result = strings.Join(res, PathSep) // NOTE: don NOT use `filepath.Join(res...)` here
        full = lenRes == lenSrcs && result == str
        if info { if false {
            diag.warnOf(p, "Path.match: path=%v str=%v res=%v stems=%v -> full=%v result=%v lens=%d,%d",
                p, str, res, stems, full, result, lenRes, lenSrcs).debug(1)
        } else {
            diag.warnOf(p, "Path.match: path=%v res=%v stems=%v lenRes=%d", p, res, stems, lenRes)
            diag.warnOf(p, "Path.match: str=%v full=%v result=%v lenSrcs=%d", str, full, result, lenSrcs).debug(1)
        }}
        if correct := (!full && strings.HasPrefix(str, result)) || (full && str == result); false {
            assert(correct, "incorrect result: res=%v result=%v full=%v stems=%v str=%s",
                res, result, full, stems, str)
        } else if !correct {
            diag.errorAt(p.position, "incorrect result: str=%s res=%v stems=%v full=%v result=%v",
                str, res, stems, full, result).debug(1)
        }
        if p.patterned(ctx) && full && len(stems) == 0 {
            diag.errorAt(p.position, "incorrect result: path=%v, str=%s, res=%v result=%v",
                p, str, res, result).debug(1)
        }
    }
    return
}
func (p *Path) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    ctx = positional(ctx, p.position)
    switch t := i.(type) {
    case  string  : return p.match1(ctx, t)
    case *filestub:
        if full, result, stems = p.match1(ctx, t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(ctx, filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
        }
    case *File:
        if false {
            for stub := t.filestub; true; stub = stub.other {
                if full, result, stems = p.match1(ctx, stub.name); full || result != "" {
                    return
                } else if stub.other == t.filestub { break }
            }
            var s = t.name
            if t.sub != "" {
                s = filepath.Join(t.sub, t.name)
                if full, result, stems = p.match1(ctx, s); full || result != "" {
                    return
                }
            }
            if t.dir != "" {
                s = filepath.Join(t.dir, s)
                if full, result, stems = p.match1(ctx, s); full || result != "" {
                    return
                }
            }
        } else if full, result, stems = p.match1(ctx, t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(ctx, t.fullname()) // NOTE: matching the fullname form
        }
    case Value :
        if str, err := t.Strval(ctx); err != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, err).debug(1)
        } else if str != "" {
            return p.match1(ctx, str)
        }
    default:
        diag.errorAt(p.position, "matching unsupport value: %T %v", i, i).debug(8)
    }
    return
}

func (p *Path) stencil(ctx Context, stems []string) (result string, rest []string) {
    var (
        strs []string
        segs []Value
        err error
    )
    if segs, err = mergeresult2(expandall2(ctx, expandPlainValue, p.Elems...)); err != nil {
        diag.errorOf(p, "expand path '%v' failed: %v", p, err).debug(1)
        return
    }

ForPathSegs:
    for _, seg := range segs {
        var s string
        if s, stems = seg.stencil(ctx, stems); s != "" {
            strs = append(strs, s)
            continue ForPathSegs
        } else if s, err = seg.Strval(ctx); err != nil {
            diag.errorOf(seg, "strval seg '%v' failed: %v", seg, err).debug(1)
            break ForPathSegs
        } else {
            strs = append(strs, s)
        }
    }
    result = strings.Join(strs, PathSep)
    rest = stems // the rest stems
    return
}

func (p *Path) Combine(val Value) {
    var (
        ti = len(p.Elems)-1
        tail = p.Elems[ti]
        comp *Barecomp
    )
    if isNil(tail) {
        diag.errorOf(p, "path tail is nil: %v", p)
        return
    } else if seg, ok := tail.(*PathSeg); ok {
        switch comp = MakeBarecomp(tail.Position()); seg.rune {
        case 0, '/': break // discard
        default: comp.Combine(tail)
        }
        p.Elems[ti] = comp
    } else if comp, ok = tail.(*Barecomp); !ok || comp == nil {
        comp = MakeBarecomp(tail.Position(), tail)
        p.Elems[ti] = comp
    } else {
        diag.errorAt(p.position, "FIXME: join %v (%T), %v (%T)", tail, tail, val, val).debug(1)
        return
    }

    if vp, ok := val.(*Path); ok {
        var head = vp.Elems[0]
        if seg, ok := head.(*PathSeg); ok {
            switch seg.rune {
            case 0, '/': break // discard
            default: comp.Combine(head)
            }
        } else { comp.Combine(head) }
        p.Elems = append(p.Elems, vp.Elems[1:]...)
    } else { comp.Combine(val) }
}

type PathSeg struct { valbase; rune }
func (p *PathSeg) String() (s string) {
    switch p.rune {
    case '/': s = "" // the first '/', aka. root -- PathSep is added when joining
    case '~': s = "~"
    case '.': s = "."
    case '^': s = ".."
    case 0  : s = "" // empty segment after the last '/', e.g. /foo/bar/
    }
    return
}
func (p *PathSeg) Strval(ctx Context) (s string, e error) {
    if p.rune == 0 { // zero segment,
        s = "" // the 'empty' after the last '/'
    } else if p.rune == '/' { // root segment,
        s = "" // the 'empty' before the first '/'
    } else if s = p.String(); s == "" {
        e = fmt.Errorf("unknown pathseg '%s'", string(p.rune))
        diag.errorAt(p.position, "unknown segment '%s' ('%v')", string(p.rune), p).debug(1)
    }
    return
}
func (p *PathSeg) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*PathSeg); ok && p.rune == a.rune { res = cmpEqual }
    return
}
func (p *PathSeg) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    var s string
    switch t := i.(type) {
    case string: s = t
    case Value:
        var e error
        if s, e = t.Strval(ctx); e != nil {
            diag.errorAt(p.position, "strval '%v' failed: %v", t, e).debug(1)
            return
        }
    }
    switch p.rune {
    case '/': if s == ""   { result, full = s, true }
    case '~': if s == "~"  { result, full = s, true }
    case '.': if s == "."  { result, full = s, true }
    case '^': if s == ".." { result, full = s, true }
    case 0  : if s == ""   { result, full = s, true }
    }
    return
}
func (p *PathSeg) stencil(ctx Context, stems []string) (result string, rest []string) {
    var e error
    if result, e = p.Strval(ctx); e != nil {
        diag.errorAt(p.position, "strval '%v' failed: %v", p, e).debug(1)
    }
    return
}

type filestub struct {
    dir  string      // full directory where the file was or should be found
    sub  string      // matched sub path (see Project.search), may be Dir (absoletep path)
    name string      // constant represented name (e.g. relative filename)
    filemap *FileMap // matched pattern (see 'files' directive)
    other *filestub  // pointed to another stub (in a different project) of the same file
}
func (p *filestub) subname() (s string) {
    if isAbsOrRel(p.sub) {
        s = p.name
    } else {
        s = filepath.Join(p.sub, p.name)
    }
    return
}

type filebase struct {
    stub filestub    // cycled-list of file stubs of different projects
    info os.FileInfo // file info if exists
    //updated bool // true if this file has been updated by a program
}
func (p *filebase) exists() (res bool) { return p.info != nil }

var statmutex sync.Mutex
var filecache = make(map[string]*filebase) // File.fullname() -> File

func stat(ctx Context, name, sub, dir string, infos ...os.FileInfo) (file *File) {
    var (
        base *filebase ; stub *filestub ; fullname string
        pos Position = ctx.Position()
    )

    statmutex.Lock(); defer statmutex.Unlock()

    // Trims / suffix
    if dir != "" { dir = filepath.Clean(dir) }
    if sub != "" { sub = filepath.Clean(sub) }
    if false {
        t := strings.HasPrefix(name, "./")
        if name!= "" { name = filepath.Clean(name) }
        if t { name = "./" + name }
    }

    if filepath.IsAbs(name) {
        if fullname = name; dir == "" {
            //dir, sub = filepath.Dir(fullname), ""
            //name = filepath.Base(fullname)
        } else if strings.HasPrefix(fullname, dir+PathSep) {
            tail := fullname[len(dir)+1:]
            //sub  = filepath.Dir(tail)
            //name = filepath.Base(tail)
            if sub == "" { name = tail } else
            if strings.HasPrefix(fullname, sub+PathSep) {
                name = tail[len(sub)+1:]
            }
        } else if dir != "" {
            if true { dir = "" } else if false {
                diag.errorAt(pos, "dir name conflicts: %s <-> %s (sub=%v)", dir, name, sub).debug(16)
                unreachable("path error")
            } else {
                return
            }
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
        dir = filepath.Join(ctx.WorkDir(), dir)
        fullname = filepath.Join(dir, sub, name)
    }

    if false { fullname = filepath.Clean(fullname) }
    if enable_assertions {
        //assert(filepath.IsAbs(sub) == false, "`%s` sub is abs", sub)
        assert(filepath.IsAbs(fullname), "`%s` is not abs {%s %s %s}", fullname, name, sub, dir)
        if filepath.IsAbs(name) && sub != "-" {
            if dir != "" {
                diag.errorAt(pos, "dirty dir: dir  = %s", dir)
                diag.errorAt(pos, "dirty dir: sub  = %s", sub)
                diag.errorAt(pos, "dirty dir: name = %s", name)
                diag.errorAt(pos, "dirty dir: full = %s", fullname).debug(16)
                assert(false, "dir is not empty for abs name: %s", name)
            }
            if sub != "" {
                diag.errorAt(pos, "dirty sub: dir  = %s", dir)
                diag.errorAt(pos, "dirty sub: sub  = %s", sub)
                diag.errorAt(pos, "dirty sub: name = %s", name)
                diag.errorAt(pos, "dirty sub: full = %s", fullname).debug(16)
                assert(false, "sub is not empty for abs name: %s", name)
            }
            if s := filepath.Clean(name); fullname != s {
                diag.errorAt(pos, "dirty fullname: dir  = %s", dir)
                diag.errorAt(pos, "dirty fullname: sub  = %s", sub)
                diag.errorAt(pos, "dirty fullname: name = %s", name)
                diag.errorAt(pos, "dirty fullname: clean = %s", s)
                diag.errorAt(pos, "dirty fullname: full  = %s", fullname).debug(16)
                assert(false, "fullname is not clean: %s", name)
            }
        } else if filepath.IsAbs(sub) {
            s := filepath.Join(sub, name)
            assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
            assert(dir == "", "`%s` invalid file (dir=%s, sub=%s, name=%s)", fullname, dir, sub, name)
        } else if filepath.IsAbs(dir) {
            s := filepath.Join(dir, sub, name)
            assert(fullname == s, "`%s` conflicted fullname (%s)", fullname, s)
            assert(filepath.IsAbs(sub) == false, "`%s` sub is abs", sub)
            assert(filepath.IsAbs(name) == false, "`%s` name is abs", name)
        } else {
            s := filepath.Join(ctx.WorkDir(), dir, sub, name)
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
        unreachable("too many input file infos")
    }

    var okay bool // NOTE: filepath.Join can have the same efffect as filepath.Clean
    var cleanFullname = filepath.Clean(fullname) // clean paths like /path/to/foo/../bar -> /path/to/bar
    if base, okay = filecache[cleanFullname]; okay {
        if base.info == nil {
            if info == nil { info, _ = os.Stat(fullname) }
            if info == nil && !addNotExisted { return nil } // file not exists
            base.info = info
        }

        var head = &base.stub
        if enable_assertions {
            for stub = head; stub != nil ; stub = stub.other {
                s1, s2 := filepath.Join(stub.dir, stub.sub, stub.name), filepath.Join(fullname)
                assert(s1 == s2, "fullname '%s' conflicted:\n" +
                    "panic: (%s, %s, %s) %s\n" +
                    "panic: (%s, %s, %s) %s\n",
                    fullname,
                    stub.dir, stub.sub, stub.name, s1,
                    dir, sub, name, s2)
                if stub.other == head { break }
            }
        }
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

        base = &filebase{ filestub{ dir, sub, name, nil, nil }, info/*, false*/ }
        base.stub.other = &base.stub
        stub = &base.stub
        filecache[cleanFullname] = base
    }
GotFile:
    file = &File{valbase{pos},base,stub} // FIXME: needs position information
    if enable_assertions {
        if !addNotExisted { assert(file.exists(), "`%s` file not existed", fullname) }
        assert(file.name == name, "(%s %s %s).name != %s", file.name, file.sub, file.dir, name)
        assert(file.sub == sub, "(%s %s %s).sub != %s", file.name, file.sub, file.dir, sub)
        if file.dir != dir {
            var head = &base.stub
            for stub := head; stub != nil; stub = stub.other {
                diag.infoAt(pos, "stat: %s %s %s", stub.dir, stub.sub, stub.name).debug(true,1)
                if stub.other == head { break }
            }
        }
        assert(file.dir == dir, "{%s %s %s} dir incorrect: %s", file.dir, file.sub, file.name, dir)
        //assert(file.dir != "", "(%s %s %s) empty dir", file.name, file.sub, file.dir)
        if file.exists() {
            if s := filepath.Join(file.dir, file.sub, file.name); false {
                assert(file.info != nil, "{%s %s %s} file info is nil", file.dir, file.sub, file.name)
                assert(file.info.Name() == filepath.Base(file.name), "{dir=%s sub=%s name=%s} name incorrect: %s",
                    file.dir, file.sub, file.name, file.info.Name())
                assert(file.fullname() == s, "{dir=%s sub=%s name=%s} fullname incorrect: %s",
                    file.dir, file.sub, file.name, s)
            } else {
                if file.info == nil {
                    diag.errorAt(pos, "{%s %s %s} file info is nil", file.dir, file.sub, file.name).debug(true,24)
                }
                if file.fullname() != s {
                    diag.errorAt(pos, "{dir=%s sub=%s name=%s} fullname incorrect: %s", file.dir, file.sub, file.name, s).debug(true,24)
                }
                if file.info.Name() != filepath.Base(s) { // NOTE: file.name might be "" here
                    diag.errorAt(pos, "{dir=%s sub=%s name=%s} name incorrect: %s",
                        file.dir, file.sub, file.name, file.info.Name()).debug(true,24)
                }
            }
        }
    }
    return
}

type File struct {
    valbase
    *filebase
    *filestub
}
func (p *File) String() string { return p.name }
func (p *File) Strval(ctx Context) (s string, err error) {
    if false { diag.warnOf(p, "use file.fullname() instead").debug(true, 8) }
    s = p.name; return }
func (p *File) True(ctx Context) (t bool, err error) {
    if p.name != "" {
        t = true // p.exists() == existenceConfirmed
    }
    return
}
func (p *File) BaseName() (s string) {
    if p.info != nil { s = p.info.Name() } else {
        s = filepath.Base(p.name)
    }
    return
}
func (p *File) fullname() (s string) {
    return filepath.Join(p.dir, p.sub, p.name)
}
func (p *File) searchInMatchedPaths(ctx Context, proj *Project) (res bool) {
    if p.filemap != nil {
        var pre string
        // FIXME: File should keep both 'match' and 'pre',
        // or just remove searchInMatchedPaths
        f := p.filemap.stat(ctx, proj.absPath, pre, p.name)
        if f != nil && f.info != nil { p.info, res = f.info, true }
    }
    return
}
func (p *File) stamp(t *traversal) (files []*File, err error) {
    var fullname string
    if fullname = p.fullname(); fullname == "" {
        diag.errorOf(p, "file `%s` has no fullname", p).debug(1)
        return
    }

    var ctx = positional(t, p.position)
    var currentTargetValue = t.getCurrentTargetValue(ctx)
    if isNil(currentTargetValue) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    }

    //var oldModTime time.Time
    //if p.info != nil { oldModTime = p.info.ModTime() }
    if p.info, err = os.Stat(fullname); err != nil {
        if false { diag.errorAt(p.position, "%v", err).debug(1) }
    } else if p.info != nil {
        var newModTime = p.info.ModTime()
        ctx.Globe().stamp(fullname, newModTime)
        // p.updated = newModTime.After(oldModTime)
        files = append(files, p)

        var target = currentTargetValue
        var cmp = target.cmp(ctx, p)
        if c, _ := t.caller(); cmp == cmpEqual && c != nil {
            // Add to caller context
            c.appendUpdated(ctx, newUpdatedTarget(p))
            target = c.autoGet("@")
        } else {
            t.appendUpdated(ctx, newUpdatedTarget(p))
        }
    }
    return
}
func (p *File) expandible(ctx Context, w expandwhat) (res bool) {
    return w&expandFullName != 0 && !filepath.IsAbs(p.name)
}
func (p *File) expand(ctx Context, w expandwhat) (res Value, err error) {
    if w&expandFullName != 0 && !filepath.IsAbs(p.name) {
        var fullname = p.fullname()
        if false && !filepath.IsAbs(fullname) { return }

        var stub, fullStub *filestub
        for stub = p.filestub; stub != nil; stub = stub.other {
            if stub.name == fullname /*&& stub.dir == "" && stub.sub == ""*/ {
                fullStub = stub; break
            } else if stub.other == p.filestub { break }
        }
        if fullStub == nil {
            fullStub = &filestub{ name:fullname, other:stub.other }
            stub.other = fullStub
        }
        res = &File{p.valbase, p.filebase, fullStub}
    }
    return
}
func (p *File) exists() (res bool) {
    if p != nil && p.filebase != nil {
        res = p.filebase.exists()
    }
    return
}
func (p *File) stat(t *traversal) (si *statinfo) {
    var err error
    if p.info == nil {
        if p.info, err = os.Stat(p.fullname()); err == nil {
            // good
        } else if pe, ok := err.(*fs.PathError); ok {
            if false {
                diag.errorAt(p.position, "File.stat %v: %v", trimPromptString(pe.Path), pe.Err).debug(1)
            }
            return
        } else {
            diag.errorAt(p.position, "File.stat failed: %v", err).debug(1)
        }
    }
    if err == nil { si = &statinfo{ file: p } }
    return
}
func (p *File) isSysFile() (res bool) {
    if p.filemap != nil && len(p.filemap.Paths) == 1 {
        // system files defined by:
        //     files (
        //       (foo.xxx) ⇒ -
        //     )
        if f, ok := p.filemap.Paths[0].(*Flag); ok {
            res = isNone(f.name) || isNil(f.name)
        }
    }
    return
}
func (p *File) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if options.traceExec { defer un(trace(t_exec, fmt.Sprintf("File %v", p))) }
    if p.isSysFile() { if options.traceTraversal { t.tracef("SysFile: true") }
        return
    }

    var ctx = positional(t, p.position)
    var currentTargetValue = t.getCurrentTargetValue(ctx)
    if isNil(currentTargetValue) {
        diag.errorAt(t.program.position, "target is <nil>").debug(1)
        return
    }

    // FIXES: checks none-File file target
    switch a := currentTargetValue.(type) {
    case *Barecomp: // convert barecomp path into a real Path
        if p, ok := a.Elems[0].(*Path); ok {
            a.Elems = append(p.Elems[len(p.Elems)-1:], a.Elems[1:]...)
            p.Elems[len(p.Elems)-1] = a
            currentTargetValue = p
            t.autoSet("@", p)
            if options.traceTraversal { t.tracef("FIX: barecomp path: %v", p) }
        } else {
            var s string
            var err error
            if s, err = a.Strval(ctx); err != nil {
                diag.errorOf(a, "strval '%v' failed: %v", a, err).debug(1)
                return
            }
            if file := t.Project().FindFile(ctx, s); file != nil {
                currentTargetValue = file
                t.autoSet("@", file)
                if options.traceTraversal { t.tracef("FIX: barecomp file: %v", p) }
            }
        }
    }

    if _, brks = t.file(p); p.info == nil {
        brks.add(p.position, breakErro).error = fileNotFoundError{ t.Project(), p }
        t.batch(func () {
            diag.errorAt(p.position, "break: missing file %v (at %s)", p, p.fullname())
            diag.errorAt(t.Project().position, "for project %v (for '%v')", t.Project(), p).debug(6)
        })
    } else if tb := brks.not(breakNext, breakCase, breakDone); len(tb) > 0 {
        t.batch(func () {
            for _, brk := range tb {
                switch brk.what {
                case breakFail: diag.errorAt(brk.pos, "broken traversal for file %v: %v", p, brk.message)
                case breakErro: diag.errorAt(brk.pos, "broken traversal for file %v with error: %v", p, brk.error)
                default: diag.errorAt(brk.pos, "broken traversal for file %v (%v)", p, brk.what)
                }
            }
            diag.errorAt(p.position, "broken traversal for file '%v' (at %s)", p, p.fullname())
            diag.errorAt(t.Project().position, "broken traversal for file '%v' from %v", p, t.Project()).debug(1)
        })
    } else if false && brks.has() {
        diag.errorAt(p.position, "broken traversal for file '%v' (at %s)", p, p.fullname()).debug(1)
    }
    return
}

func (p *File) cmp(ctx Context, v Value) (res cmpres) {
    if isNil(v) || isNone(v) {
        // ...
    } else if a, ok := v.(*File); ok {
        if a == nil {
            //assert(a != nil, "nil file")
        } else if p.filebase == a.filebase {
            res = cmpEqual
        } else if p.fullname() == a.fullname() {
            s := fmt.Sprintf("\na: %s %s %s (%s)", p.dir, p.sub, p.name, p.fullname())
            s += fmt.Sprintf("\nb: %s %s %s (%s)", a.dir, a.sub, a.name, a.fullname())
            unreachable("same files differed: ", p.name, " != ", a.name, s)
        } else if false /*p.dir != a.dir && p.sub == a.sub && p.name == a.name*/ {
            s := fmt.Sprintf("\n      a: %s: %s %s", p.name, p.dir, p.sub)
            s += fmt.Sprintf("\n      b: %s: %s %s", a.name, a.dir, a.sub)
            fmt.Fprintf(stderr, "%s: warning: files may differ: %s != %s :%s\n", p.position, p.name, a.name, s)
        }
    }
    return
}

func (p *File) patterned(ctx Context, ) bool { return false }
func (p *File) match1(ctx Context, v string) (full bool, s string, stems []string) {
    if name := p.name; name == v {
        s, full = name, true
    } else if name = filepath.Join(p.sub, p.name); name == v {
        s, full = name, true
    }
    return
}
func (p *File) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    switch t := i.(type) {
    case string: return p.match1(ctx, t)
    case Value:
        if !(isNil(t) || isNone(t)) {
            var ( v string; e error )
            if v, e = t.Strval(ctx); e != nil {
                diag.errorOf(t, "strval '%v' failed: %v", t, e).debug(1)
            } else { return p.match1(ctx, v) }
        }
    default:
        diag.errorAt(p.position, "matching file '%v' with unknown input: %v", p, i).debug(1)
    }
    return
}
func (p *File) stencil(ctx Context, stems []string) (s string, rest []string) {
    s = p.name
    return
}

func (p *File) change(dir, sub, name string) (okay bool) {
    if fullname := filepath.Join(dir, sub, name); p.fullname() == fullname {
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
            assert(p.fullname() == fullname, "changed invalid File")
        }
    }
    return
}

type FileContent struct {
    file *File
    content []byte
}

type Flag struct { valbase ; name Value }
func (p *Flag) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *Flag) Strval(ctx Context) (s string, e error) {
    if p.name == nil {
        s = "-"
    } else if isNone(p.name) {
        s = "-"
    } else if s, e = p.name.Strval(ctx); e == nil {
        s = "-" + s
    }
    return
}
func (p *Flag) True(ctx Context) (t bool, err error) { return p.name.True(ctx) }
func (p *Flag) refs(ctx Context, v Value) bool { return p.name.refs(ctx, v) }
func (p *Flag) defs(ctx Context, s ...string) []*Def { return p.name.defs(ctx, s...) }
func (p *Flag) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    switch t := i.(type) {
    case *None, *Nil, *unresolvedobject:
    case *Flag:
        full, s, stems = p.name.match(ctx, t.name)
        s = "-" + s
    case Value:
        if v, e := t.Strval(ctx); e != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, e).debug(1)
        } else if strings.HasPrefix(v, "-") {
            full, s, stems = p.name.match(ctx, v[1:])
            s = "-" + s
        }
    case string:
        if strings.HasPrefix(t, "-") {
            full, s, stems = p.name.match(ctx, t[1:])
            s = "-" + s
        }
    default:
        diag.warnAt(p.position, "-%v <-> %T %v", p.name, i, i).debug(true, 16)
    }
    return
}
func (p *Flag) expandible(ctx Context, w expandwhat) bool { return p.name.expandible(ctx, w) }
func (p *Flag) expand(ctx Context, w expandwhat) (res Value, err error) {
    var name Value
    if name, err = p.name.expand(ctx, w); err != nil {
        diag.errorOf(p.name, "expand '%v' failed: %v", p.name, err).debug(1)
    } else if !isNil(name) && name != p.name {
        res = &Flag{p.valbase, name}
    }
    return
}
func (p *Flag) elemStr(ctx Context, o Object, k elemkind) (s string) {
    return "-" + elementString(ctx, o, p.name, k)
}
func (p *Flag) opt(ctx Context, short, long string) (res string, match bool) {
        if isNil(p.name) {
                diag.errorOf(p, "flag name is nil")
        } else if f, ok := p.name.(*Flag); ok {
                res, match = f.opt(ctx, short, long)
        } else if s, err := p.name.Strval(ctx); err != nil {
                diag.errorOf(p.name, "strval '%v' failed: %v", p.name, err)
        } else if s == short {
                res, match = short, true
        } else if s == long {
                res, match = long, true
        }
        return
}
// DEPRECATED
func (p *Flag) opts(ctx Context, try bool, opts ...string) (runes []rune, names []string, err error) {
    switch t := p.name.(type) {
    case *Flag:
        runes, names, err = t.opts(ctx, try, opts...)
    case *String:
        for _, opt := range opts {
            if t.string == opt { names = append(names, opt) }
        }
        if !try && len(names) == 0 {
            diag.errorOf(p, "unknown flag (known: %s)", strings.Join(opts, ", "))
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
                    names = append(names, opt[i+1:])
                } else if t.string ==  opt[0:i]/*strings.ContainsAny(t.string, opt[0:i])*/ {
                    runes = append(runes, rune(opt[0]))
                    names = append(names, opt[i+1:])
                }
            }
        }
        if !try && (len(runes) == 0 || len(names) == 0) {
            diag.errorOf(p, "unknown flag (known: %s)", strings.Join(opts, ", "))
        }
    }
    if enable_assertions {
        assert(len(runes) == len(names), "unmatched opts lengths")
    }
    return
}
func (p *Flag) cmp(ctx Context, v Value) (res cmpres) {
    if v == nil {
        // ...
    } else if a, ok := v.(*Flag); ok {
        res = p.name.cmp(ctx, a.name)
    }
    return
}
func (p *Flag) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if s, err := p.Strval(positional(t, p.position)); err != nil {
        diag.errorOf(p, "strval '%v' failed: %v", p, err).debug(1)
    } else {
        _, brks = t.string(p, s)
    }
    return
}

const escapedChars = "\"\r\n"

type Compound struct { valbase ; elements } // "compound string"
func (p *Compound) String() string { return p.elemStr(nil, nil, 0) }
func (p *Compound) Strval(ctx Context) (s string, err error) {
    for _, e := range p.Elems {
        var v string
        if v, err = e.Strval(ctx); err == nil {
            s += v
        } else {
            break
        }
    }
    return
}
func (p *Compound) Float(ctx Context) (f float64, err error) {
    var s string
    if s, err = p.Strval(ctx); err == nil {
        f, err = strconv.ParseFloat(s, 64)
    }
    return
}
func (p *Compound) Integer(ctx Context) (i int64, err error) {
    var s string
    if s, err = p.Strval(ctx); err == nil {
        i, err = strconv.ParseInt(s, 10, 64)
    }
    return
}
func (p *Compound) True(ctx Context) (bool, error) { return p.elements.True(ctx) }
func (p *Compound) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *Compound) defs(ctx Context, s ...string) []*Def { return p.elements.defs(ctx, s...) }
func (p *Compound) expandible(ctx Context, w expandwhat) bool { return p.elements.expandible(ctx, w) }
func (p *Compound) expand(ctx Context, w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(ctx, w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand compound elems failed: %v", err).debug(1)
    } else if num > 0 {
        res = &Compound{p.valbase, elements{elems}}
    }
    return
}
func (p *Compound) elemStr(ctx Context, o Object, k elemkind) (s string) {
    var tk = k|elemNoQuote
    for _, elem := range p.Elems { s += elementString(ctx, o, elem, tk) }
    if k&elemNoQuote != 0 { return }
    var err error
    var buf bytes.Buffer
    buf.WriteString(`"`)
    defer func() {
        buf.WriteString(`"`)
        s = buf.String()
    } ()
    for i := strings.IndexAny(s, escapedChars); i != -1; {
        if _, err = buf.WriteString(s[:i]); err != nil {
            diag.errorOf(p, "%v", err)
            return
        }
        var esc string
        switch s[i] {
        case '"':  esc = `\"`
        case '\r': esc = `\r`
        case '\n': esc = `\n`
        }
        s = s[i+1:]
        if _, err = buf.WriteString(esc); err != nil {
            diag.errorOf(p, "%v", err)
			return
        }
        i = strings.IndexAny(s, escapedChars)
    }
    if _, err = buf.WriteString(s); err != nil {
        diag.errorOf(p, "%v", err)
    }
    return
}
func (p *Compound) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Compound); ok {
        s1, e := p.Strval(ctx)
        if e != nil { return }
        s2, e := a.Strval(ctx)
        if e != nil { return }
        if s1 == s2 { res = cmpEqual }
    }
    return
}

type List struct {
        position Position
        elements
}
func (p *List) Position() (pos Position) { return p.position }
func (p *List) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *List) Strval(ctx Context) (s string, err error) {
    var x = 0
    for _, e := range p.Elems {
        var v string
        if v, err = e.Strval(ctx); err == nil {
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
func (p *List) Float(ctx Context) (float64, error) {
    i, e := p.Integer(ctx)
    return float64(i), e
}
func (p *List) Integer(ctx Context) (int64, error) {
    if n := len(p.Elems); n == 1 {
        // If there's only one element, treat it as a scalar.
        return p.Elems[0].Integer(ctx)
    } else {
        return int64(n), nil
    }
}
func (p *List) elemStr(ctx Context, o Object, k elemkind) (s string) {
    var strs []string
    for _, elem := range p.Elems {
        strs = append(strs, elementString(ctx, o, elem, k))
    }
    return strings.Join(strs, " ")
}
func (p *List) expand(ctx Context, w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(ctx, w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand list elems failed: %v", err).debug(1)
    } else if num > 0 {
        res = &List{p.position, elements{elems}}
    }
    return
}
func (p *List) traverse(t *traversal) (brks breakers) {
    if len(p.Elems) == 0 { return }
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    for _, v := range p.Elems {
        if brks = v.traverse(t); brks.has() { break }
    }
    return
}
func (p *List) stat(t *traversal) (si *statinfo) {
    if len(p.Elems) > 0 {
        for _, elem := range p.Elems {
            if ei := elem.stat(t); ei == nil {
                // FIXME: insert new statinfo or just discard it ??
            } else if si == nil {
                si = ei
            } else {
                si.next = ei
            }
        }
    }
    return
}
func (p *List) stamp(t *traversal) (files []*File, err error) {
    for _, elem := range p.Elems {
        var a []*File
        if a, err = elem.stamp(t); err != nil { break }
        files = append(files, a...)
    }
    return
}

func (p *List) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*List); ok { res = p.cmpElems(ctx, a.Elems) }
    return
}

func (p *List) patterned(ctx Context) bool {
    // TODO: apply to each element:
    /*for _, elem := range p.Elems {
        if elem.patterned() { return true }
    }*/
    return false
}

func (p *List) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    // TODO: match each element
    return
}

func (p *List) stencil(ctx Context, stems []string) (s string, rest []string) {
    // TODO: stencil each element
    return
}

type Group struct { valbase ; elements }
func (p *Group) String() string { return p.elemStr(nil, nil, 0) }
func (p *Group) Position() Position { return p.valbase.Position() }
func (p *Group) Float(ctx Context) (float64, error) { return p.valbase.Float(ctx) }
func (p *Group) Integer(ctx Context) (int64, error) { return p.valbase.Integer(ctx) }
func (p *Group) True(ctx Context) (t bool, err error) {
    t = len(p.Elems) > 0
    return
}
func (p *Group) Strval(ctx Context) (s string, err error) {
    if s, err = p.Strval(ctx); err == nil {
        s = "(" + s + ")"
    }
    return
}
func (p *Group) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *Group) defs(ctx Context, s ...string) []*Def { return p.elements.defs(ctx, s...) }
func (p *Group) stat(t *traversal) (si *statinfo) { return }
func (p *Group) stamp(t *traversal) (files []*File, err error) { return }
func (p *Group) elemStr(ctx Context, o Object, k elemkind) string {
    var strs []string
    for _, elem := range p.Elems {
        strs = append(strs, elementString(ctx, o, elem, k))
    }
    return fmt.Sprintf("(%s)", strings.Join(strs, " "))
}
func (p *Group) expandible(ctx Context, w expandwhat) bool { return p.elements.expandible(ctx, w) }
func (p *Group) expand(ctx Context, w expandwhat) (res Value, err error) {
    var ( elems []Value; num int )
    if elems, num, err = expandall1(ctx, w, p.Elems...); err != nil {
        diag.errorAt(p.position, "expand group elems failed: %v", err).debug(1)
    } else if num > 0 {
        res = &Group{p.valbase, elements{elems}}
    }
    return
}
func (p *Group) traverse(t *traversal) (brks breakers) {
    diag.warnAt(p.position, "traversing group: %v", p).debug(32)
    return
}
func (p *Group) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Group); ok { res = p.cmpElems(ctx, a.Elems) }
    return
}
func (p *Group) patterned(ctx Context, ) bool { return false }
func (p *Group) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return }
func (p *Group) stencil(ctx Context, stems []string) (s string, rest []string) { return }

func parseGroupValue(g *Group) (result Value) {
    if len(g.Elems) == 0 { return g } else {
        var word *Bareword
        switch kind := g.Elems[0].(type) {
        case *Bareword: word = kind
        case *Group: if len(kind.Elems) > 0 {
            var ( name = kind.Elems[0]; ok bool )
            if word, ok = name.(*Bareword); !ok {
                diag.errorOf(name, "unsupported name type: %T %v", name, name).debug(1)
            }
        }}
        if word != nil {
            switch word.string {
            case "plain", "json", "yaml", "xml":
                result = MakeList(g.Elems[1].Position(), g.Elems[1:]...)
            }
        }
        if isNil(result) { result = g }
    }
    return
}

type Pair struct { // key=value
    valbase
    Key Value
    Value Value
}
func (p *Pair) SetValue(v Value) { p.Value = v }
func (p *Pair) SetKey(k Value) {
    switch o := k.(type) {
    case *Pair: k = o.Key
    }
    p.Key = k
}
func (p *Pair) String() string { return p.elemStr(nil, nil, 0) }
func (p *Pair) Strval(ctx Context) (s string, err error) {
    var k, v string
    if k, err = p.Key.Strval(ctx); err == nil {
        if v, err = p.Value.Strval(ctx); err == nil {
            s = k + "=" + v
        }
    }
    return
}
func (p *Pair) True(ctx Context) (t bool, err error) {
    if t, err = p.Key.True(ctx); err != nil {
        diag.errorOf(p.Key, "truthify '%v' failed: %v", p.Key, err).debug(1)
    } else if t || isNil(p.Value) {
        // done
    } else if t, err = p.Value.True(ctx); err != nil {
        diag.errorOf(p.Key, "truthify '%v' failed: %v", p.Value, err).debug(1)
    }
    return
}
func (p *Pair) Integer(ctx Context) (int64, error) { return p.Value.Integer(ctx) }
func (p *Pair) Float(ctx Context) (float64, error) { return p.Value.Float(ctx) }
func (p *Pair) refs(ctx Context, v Value) bool { return p.Key.refs(ctx, v) || p.Value.refs(ctx, v) }
func (p *Pair) defs(ctx Context, s ...string) []*Def {
    return append(p.Key.defs(ctx, s...), p.Value.defs(ctx, s...)...)
}
func (p *Pair) traverse(t *traversal) (brks breakers) {
    diag.errorAt(p.position, "traversing pair '%v' is undefined", p).debug(16)
    t.traceCallStack(p.position, -1, "pair is not traversible: %v", p)
    return
}
func (p *Pair) expandible(ctx Context, w expandwhat) bool {
    if p.Key.expandible(ctx, w) { return true }
    return w&expandPairVal != 0 && p.Value.expandible(ctx, w)
}
func (p *Pair) expand(ctx Context, w expandwhat) (res Value, err error) {
    var k, v Value
    if k, err = p.Key.expand(ctx, w); err != nil {
        diag.errorOf(p.Key, "expand '%v' failed: %v", p.Key, err).debug(1)
        return
    } else if isNil(k) { k = p.Key }

    // Note: donot expand the p.Value! It's used as template
    // in arguments (see copy-file for example).
    if w&expandPairVal != 0 {
        if v, err = p.Value.expand(ctx, w); err != nil {
            diag.errorOf(p.Value, "expand '%v' failed: %v").debug(1)
        } else if (!isNil(k) && k != p.Key) || (!isNil(v) && v != p.Value) {
            if isNil(v) { v = p.Value }
            res = &Pair{p.valbase, k, v}
        }
    } else if !isNil(k) && k != p.Key {
        res = &Pair{p.valbase, k, p.Value}
    }
    return
}
func (p *Pair) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Pair); ok {
        if p.Key.cmp(ctx, a.Key) == cmpEqual {
            if p.Value.cmp(ctx, a.Value) == cmpEqual {
                res = cmpEqual
            }
        }
    }
    return
}
func (p *Pair) elemStr(ctx Context, o Object, k elemkind) string {
    return elementString(ctx, o, p.Key, k)+`=`+elementString(ctx, o, p.Value, k)
}


// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct {
    valbase
    l token.Token
    x Value
    a []Value
}
func (p *delegate) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *delegate) Strval(ctx Context) (s string, err error) {
    var v Value
    if v, err = p.value(ctx); err != nil {
        diag.errorOf(p, "delegate '%v' value failed: %v", p, err).debug(1)
    } else if isNil(v) {
        diag.errorOf(p, "delegate value is nil: %v", p).debug(1)
    } else if s, err = v.Strval(ctx); err != nil {
        diag.errorOf(v, "strval '%v' failed: %v", v, err).debug(1)
    }
    return
}
func (p *delegate) True(ctx Context) (t bool, err error) {
    var v Value
    if v, err = p.expand(ctx, expandPlainValue); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).debug(1)
    } else if isNil(v) {
        diag.errorAt(p.position, "expand '%v' to nil", p).debug(1)
    } else if t, err = v.True(ctx); err != nil {
        diag.errorAt(p.position, "truthify '%v' failed: %v", p, err).debug(1)
    } else if false {
        diag.infoAt(p.position, "%v -> %T %v -> %v", p, v, v, t).debug(8)
    }
    return
}
func (p *delegate) Integer(ctx Context) (i int64, err error) {
    var v Value
    if v, err = p.value(ctx); err != nil {
        diag.errorOf(p, "delegate '%v' value failed: %v", p, err).debug(1)
    } else if isNil(v) {
        diag.errorOf(p, "delegate value is nil: %v", p).debug(1)
    } else if i, err = v.Integer(ctx); err != nil {
        diag.errorOf(v, "integify '%v' failed: %v", v, err).debug(1)
    }
    return
}
func (p *delegate) Float(ctx Context) (f float64, err error) {
    var v Value
    if v, err = p.value(ctx); err != nil {
        diag.errorOf(p, "delegate '%v' value failed: %v", p, err).debug(1)
    } else if isNil(v) {
        diag.errorOf(p, "nil delegate value: %v", p).debug(1)
    } else if f, err = v.Float(ctx); err != nil {
        diag.errorOf(v, "floatify '%v' failed: %v", v, err).debug(1)
    }
    return
}
func (p *delegate) isValidToken() (res bool) {
    switch p.l {
    case token.LCOLON, token.LPAREN, token.LBRACE, token.STRING, token.COMPOUND, token.ILLEGAL:
        res = true
    default:
        // for $. $/ $1 ... &. &/ &1 ... etc.
        res = p.l.IsClosure() || p.l.IsDelegate()
    }
    return
}
func (p *delegate) refs(ctx Context, v Value) (res bool) {
    if isNil(p.x) {
        diag.errorOf(p, "delegation of nil (v=%v)", v).debug(1)
        return
    } else if p.x == v || p.x.refs(ctx, v) {
        return true
    }
    for _, a := range p.a {
        if a.refs(ctx, v) { return true }
    }
    return
}
func (p *delegate) defs(ctx Context, s ...string) (res []*Def) {
    if isNil(p.x) {
        diag.errorOf(p, "delegation of nil (s=%v)", p, s).debug(1)
        return
    } else if d, ok := p.x.(*Def); ok {
        if ok = len(s) == 0; !ok {
            for _, a := range s { if ok = d.name == a; ok { break } }
        }
        if ok { res = append(res, d) }
    } else {
        res = p.x.defs(ctx, s...)
    }
    for _, a := range p.a {
        res = append(res, a.defs(ctx, s...)...)
    }
    return
}
func (p *delegate) traverse(t *traversal) (brks breakers) {
    var ctx = positional(t, p.position)
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if val, err := p.expand(ctx, expandPlainValue); err != nil { if true {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).debug(16)
        if true { t.traceCallStack(p.position, -1, "expand '%v' failed", p) }
    }} else if isNil(val) { if true {
        diag.warnAt(p.position, "delegate '%v' expands to nil", p).debug(16)
        if true { t.traceCallStack(p.position, -1, "delegate '%v' expands to <nil>", p) }
    }} else if isNone(val) { if false {
        diag.warnAt(p.position, "delegate '%v' expands to none", p).debug(16)
        if true { t.traceCallStack(p.position, -1, "delegate '%v' expands to <none>", p) }
    }} else {
        brks = val.traverse(t)
    }
    return
}
func (p *delegate) string(ctx Context, o Object, k elemkind) (s string) { // source representation
    for i, a := range p.a {
        if i == 0 { s = " " } else { s += "," }
        s += elementString(ctx, o, a, k)
    }

    var name string
    switch x := p.x.(type) {
    case *selection: name = x.String()
    case Object    : name = x.Name()
    }

    switch p.l {
    case token.LCOLON: s = fmt.Sprintf(":%s%s:", name, s)
    case token.LPAREN: s = fmt.Sprintf("(%s%s)", name, s)
    case token.LBRACE:
        if k&elemNoBrace == 0 {
            s = fmt.Sprintf("{%s%s}", name, s)
        } else {
            s = fmt.Sprintf("(%s%s)", name, s)
        }
    case token.STRING, token.COMPOUND:
        s = fmt.Sprintf("%s%s", name, s)
    case token.ILLEGAL: // $@, &@, $<, &<, etc.
        if len(name) == 1 && len(s) == 0 {
            s = fmt.Sprintf("%s", name)
        } else {
            s = fmt.Sprintf("[%s%s]", name, s)
        }
    default:
        if p.l.IsClosure() || p.l.IsDelegate() {
            s = p.l.String()
        } else {
            s = fmt.Sprintf("[%s%s]!(%v)", name, s, p.l)
        }
    }
    return
}
func (p *delegate) elemStr(ctx Context, o Object, k elemkind) (s string) {
    if ctx == nil || k&elemExpand == 0 {
        if s = p.string(ctx, o, k); !(p.l.IsClosure() || p.l.IsDelegate()) { s = "$" + s }
    } else if v, e := p.expand(ctx, expandDelegate); e != nil {
        diag.errorAt(p.position, "expand failed: %v", e).debug(1)
    } else {
        if isNil(v) { v = p }
        s = elementString(ctx, o, v, k)
    }
    return
}
func (p *delegate) value(ctx Context) (v Value, err error) {
    if v, err = p.expand(ctx, expandDelegate); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).debug(1)
    } else if v == p { // d, ok := v.(*delegate); ok && d == p
        diag.errorOf(p, "self delegation: %v", p).debug(1)
    }
    return
}
func (p *delegate) args(ctx Context, w expandwhat) (args []Value, num int, err error) {
    if w&expandArgs != 0 {
        if args, num, err = expandall1(ctx, w, p.a...); err != nil {
            diag.errorAt(p.position, "expand args %v failed: %v", p.a, err).debug(1)
            return
        } else if len(args) == 0 && num == 0 && len(p.a) > 0 { args = p.a }
    } else if len(p.a) > 0 { args = p.a }
    return
}
func (p *delegate) expandible(ctx Context, w expandwhat) (res bool) {
    if res = w&expandDelegate != 0; !res {
        if res = p.x.expandible(ctx, w); !res && w&expandArgs != 0 {
            for _, a := range p.a {
                if res = a.expandible(ctx, w); res { break }
            }
        }
    }
    return
}
func (p *delegate) expand(ctx Context, w expandwhat) (res Value, err error) {
    if isNil(p.x) {
        diag.errorAt(p.position, "expand nil delegation: %v (w=%016b)", p, w).debug(32)
        return
    }

    var v Value
    if w&expandDelegate == 0 {
        var x Value
        if x, err = p.x.expand(ctx, w); err != nil {
            diag.errorOf(p.x, "expand '%v' failed: %v", p.x, err).debug(1)
            return
        } else if isNil(x) { x = p.x }

        var ( args []Value; num int )
        if args, num, err = p.args(ctx, w); err != nil {
            diag.errorAt(p.position, "expand args failed: %v", err).debug(1)
            return
        }

        if (!isNil(x) && x != p.x) || num > 0 {
            if num == 0 { args = p.a }
            res = &delegate{p.valbase, p.l, x, args}
        }
    } else if res, err = p.reveal(ctx, w); err != nil {
        diag.errorOf(p, "reveal '%v' failed: %v", p, err).debug(1)
    } else if isNil(res) {
        res = MakeNone(p.position)
    } else if v, err = res.expand(ctx, w); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", res, err).debug(1)
    } else if !isNil(v) && v != res {
        res = v
    }
    return
}
func (p *delegate) reveal(ctx Context, w expandwhat) (res Value, err error) {
    var x Object
    /*if pn, ok := p.x.(*ProjectName); ok {
        ...
    } else*/ if t, ok := p.x.(Object); ok && t != nil {
        x = t
    } else if t, ok := p.x.(*usinglist); ok {
        x = t
    } else if t, ok := p.x.(*selection); ok && t != nil && !isNil(t.o) {
        if isNil(t.o) {
            diag.errorAt(p.position, "%v: selection object is nil (w=%016b)", p, w).debug(16)
            return
        } else if n, ok := t.o.(*ProjectName); ok && n != nil && n.project != nil {
            ctx = closureWith(ctx, n.project.scope)
        }
        if v, e := t.value(ctx); e != nil {
            diag.errorAt(p.position, "select value '%v' failed: %v", p.x, e).debug(1)
            err = e; return
        } else if isNil(v) {
            diag.errorAt(p.position, "%v: selected value is nil (%T %v) (w=%016b)", p, t.o, t.o, w).debug(16)
            return
        } else if t, ok := v.(Object); ok {
            x = t
        }
    } else {
        diag.errorOf(p, "delegate unsupported value '%v' (%T)", p.x, p.x).debug(1)
        return
    }

    var args []Value
    if args, _, err = p.args(ctx, w); err != nil {
        diag.errorAt(p.position, "apply args failed: %v", err).debug(1)
        return
    }

    ctx = positional(ctx, p.position)
    switch t := x.(type) {
    case Caller:
        if res = t.Call(ctx, args...); isNil(res) {
            if d, ok := x.(*Def); ok && !isNil(d.value) {
                diag.errorAt(p.position, "calling def '%v' (%v) returns <nil> (def.value=%v %T, %T)",
                    d.name, d.origin, d.value, d.value, ctx).debug(16)
            }
        }
    case Executer:
        if vals, brks := t.Execute(ctx, args...); len(brks) > 0 {
            for _, brk := range brks {
                var s string
                if brk.message != "" { s = brk.message }
                if brk.error != nil { s += fmt.Sprintf(" (error: %s)", brk.error) }
                diag.errorAt(brk.pos, "broken '%v': (%s) %s", x, brk.what, s).debug(1)
            }
        } else if len(vals) > 0 {
            res = MakeList(ctx.Position(), vals...)
        }
    default:
        var pos = t.Position()
        if !pos.IsValid() { pos = p.position }
        diag.errorAt(pos, "%s: unknown delegation: %T %v -> %T %v", x.Name(), p.x, p.x, x, x).debug(32)
    }
    return
}
func (p *delegate) stat(t *traversal) (si *statinfo) {
    diag.errorAt(p.position, "cant stat delegate %v, must expand it first", p).debug(16)
    return
}
func (p *delegate) stamp(t *traversal) (file []*File, err error) {
    diag.errorAt(p.position, "cant stamp delegate %v, must expand it first", p).debug(16)
    return
}
func (p *delegate) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*delegate); ok {
        // NOTE: don't compare the expanded value!!
        if res = p.x.cmp(ctx, a.x); res == cmpEqual && len(p.a) == len(a.a) {
            for i, t := range p.a {
                if res = t.cmp(ctx, a.a[i]); res != cmpEqual { return }
            }
        }
    } else if d, ok := p.x.(*Def); ok && len(p.a) == 0 {
        res = d.value.cmp(ctx, v)
    }
    return
}

type closure struct { delegate }
func (p *closure) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *closure) Strval(ctx Context) (s string, err error) {
    if !p.isValidToken() {
        err = fmt.Errorf("invalid closure token: %v", p.l)
        diag.errorAt(p.Position(), err.Error()).debug(1)
        return
    }

    var val Value
    if val, err = p.expand(ctx, expandDelegate|expandClosure); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", p, err).debug(1)
    } else if isNil(val) {
        if false { diag.warnOf(p, "expand '%v' to nil", p).debug(1) }
    } else if s, err = val.Strval(ctx); err != nil {
        diag.errorOf(p, "strval '%v' failed: %v", p, err).debug(1)
    }
    return
}
func (p *closure) True(ctx Context) (t bool, err error) {
    var v Value
    if v, err = p.expand(ctx, expandPlainValue); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).debug(1)
    } else if isNil(v) {
        // does nothing
    } else if t, err = v.True(ctx); err != nil {
        diag.errorAt(p.position, "truthify '%v' failed: %v", p, err).debug(1)
    } else if false {
        diag.infoAt(p.position, "%v -> %T %v -> %v", p, v, v, t).debug(8)
    }
    return
}
func (p *closure) elemStr(ctx Context, o Object, k elemkind) (s string) {
    if ctx == nil || k&elemExpand == 0 {
        if s = p.string(ctx, o, k); !(p.l.IsClosure() || p.l.IsDelegate()) { s = "&" + s }
    } else if v, e := p.expand(ctx, expandDelegate/*|expandClosure*/); e != nil {
        diag.errorAt(p.position, "expand failed: %v", e).debug(1)
    } else {
        if isNil(v) { v = p }
        s = elementString(ctx, o, v, k)
    }
    return
}
func (p *closure) expandible(ctx Context, w expandwhat) (res bool) {
    if res = w&expandClosure != 0; !res {
        if res = p.x.expandible(ctx, w); !res && w&expandArgs != 0 {
            for _, a := range p.a {
                if res = a.expandible(ctx, w); res { break }
            }
        }
    }
    return
}
func (p *closure) expand(ctx Context, w expandwhat) (res Value, err error) {
    if isNil(p.x) {
        diag.errorAt(p.position, "expand nil closure: %v (%d)", p, w).debug(1)
        return
    }

    var val Value
    if w&expandClosure == 0 {
        if val, err = p.x.expand(ctx, w); err != nil {
            diag.errorOf(p.x, "expand '%v' failed: %v", p.x, err).debug(1)
            return
        } else if isNil(val) { val = p.x }

        var ( args []Value; num int )
        if args, num, err = p.args(ctx, w); err != nil {
            diag.errorOf(p, "expand args failed: %v", err).debug(1)
            return
        } else if (!isNil(val) && val != p.x) || num > 0 {
            if num == 0 { args = p.a }
            res = &closure{delegate{p.valbase, p.l, val, args}}
        }
    } else if res, err = p.disclose(ctx, w); err != nil {
        diag.errorOf(p, "disclose '%v' failed: %v", p, err).debug(1)
    } else if isNil(res) {
        diag.errorOf(p, "disclose '%v' to nil (%s '%v')", p, typeof(p.x), p.x).debug(16)
    } else if w&^expandClosure == 0 {
        // done, no more expand
    } else if val, err = res.expand(ctx, w); err != nil {
        diag.errorOf(p, "expand '%v' failed: %v", res, err).debug(1)
    } else if !isNil(val) && val != res {
        if p.String() == "&(objects)" {
            var v, _ = res.expand(ctx.inner(), w)
            diag.infoAt(p.position, "%v -> %T %v -> %T %v -> %T %v", p, res, res, val, val, v, v)
            diag.infoAt(p.position, "%v %v", p, ctx.inner())
            diag.infoAt(p.position, "%v %v", p, ctx).debug(16)
        }
        res = val
    }
    return
}
func (p *closure) disclose(ctx Context, w expandwhat) (res Value, err error) {
    var x Object
    if true { defer func() { if name, proj := p.x.(Object).Name(), ctx.Project(); name == "objects" {
        var val1, _ = res.expand(ctx, w)
        var s, _ = val1.Strval(ctx); if !strings.Contains(s, "/unknown/") { return }
        var o1 = ctx.closureResolveObject(p.position, name)
        diag.warnAt(p.Position(), "%v: p = %v, %016b", proj, p, w) //.debug(1)
        diag.warnAt(p.x.Position(), "%v: p.x = %T %v", proj, p.x, p.x) //.debug(1)
        diag.warnAt(res.Position(), "%v: res = %T %v", proj, res, res) //.debug(1)
        diag.warnAt(val1.Position(), "%v: val1 = %T %v", proj, val1, val1) //.debug(1)
        diag.warnAt(val1.Position(), "%v: val1 = %T %v", proj, val1, s) //.debug(1)
        if o1 != nil { diag.warnAt(o1.Position(), "%v: o1  = %T %v", proj, o1, o1)/*.debug(1)*/ }
        if d, ok := res.(*delegate); ok {
            diag.warnAt(d.x.Position(), "%v: d.x = %T %v", proj, d.x, d.x)/*.debug(1)*/ }
        diag.warnAt(ctx.Position(), "%v: %s", proj, ctx).debug(240)
    } } () }

    if t, ok := p.x.(Object); ok && t != nil {
        x = t
    } else if t, ok := p.x.(*selection); ok && t != nil && !isNil(t.o) {
        if n, ok := t.o.(*ProjectName); ok && n != nil && n.project != nil {
            ctx = closureWith(ctx, n.project.scope)
        }
        if v, e := t.value(ctx); e != nil {
            diag.errorOf(p.x, "select value '%v' failed: %v", p.x, e).debug(1)
            err = e; return
        } else if t, ok := v.(Object); ok {
            x = t
        }
    }

    var name string
    if isNil(x) {
        diag.errorOf(p.x, "closure non-object: %v (%T)", p.x, p.x).debug(1)
        return
    } else if name = x.Name(); name == "" {
        diag.errorOf(p.x, "empty closure name: %T %v -> %T %v", p.x, p.x, x, x).debug(1)
        return
    }

ClosureTok:
    switch p.l {
    case token.LBRACE, token.STRING, token.COMPOUND:
        if entry := ctx.closureResolveEntry(p.position, name); entry == nil {
            // continue
        } else if p.l == token.LBRACE {
            x = entry
            break ClosureTok
        } else { // token.STRING, token.COMPOUND
            // &'xxx' and &"xxx" are fetching entry in the closure context
            res = entry
            return
        }
    default: //case token.LPAREN, token.ILLEGAL:
        if obj := ctx.closureResolveObject(p.position, name); obj == nil {
            // continue
        } else {
            x = obj
            break ClosureTok
        }
    }

    var ( args []Value; num int )
    if args, num, err = p.args(ctx, w); err != nil {
        diag.errorAt(p.position, "get args failed: %v", err).debug(1)
    } else if isNil(x) {
        diag.errorAt(p.position, "closure object is nil: %v", p.x).debug(1)
    } else if p.x != x || num > 0 {
        if t1, t2 := reflect.TypeOf(p.x), reflect.TypeOf(x); t1 != t2 {
            if unres := reflect.TypeOf((*unresolvedobject)(nil)); t1 != unres {
                var o1 = ctx.closureResolveObject(p.position, name)
                var o2 = ctx.inner().closureResolveObject(p.position, name)
                var o3 = ctx.inner().inner().closureResolveObject(p.position, name)
                diag.errorAt(p.position, "%T %v", o1, o1)
                diag.errorAt(p.position, "%T %v", o2, o2)
                diag.errorAt(p.position, "%T %v", o3, o3)
                diag.errorAt(p.position, "closure object type differs: %v (!= %v)", t2, t1)
                diag.errorAt(p.x.Position(), "%v '%s' is found here", t1, name)
                diag.errorAt(ctx.Position(), "%v", ctx).debug(16)
                return
            }
        }
        res = &delegate{p.valbase, p.l, x, args}
    } else {
        res = &p.delegate
    }
    return
}
func (p *closure) traverse(t *traversal) (brks breakers) {
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if val, err := p.expand(positional(t, p.position), expandClosure); err != nil {
        diag.errorAt(p.position, "expand '%v' failed: %v", p, err).debug(1)
    } else if isNil(val) {
        diag.warnAt(p.position, "closure '%v' expands to nil", p).debug(1)
    } else if isNone(val) {
        diag.warnAt(p.position, "closure '%v' expands to none", p).debug(1)
    } else {
        brks = val.traverse(t)
    }
    return
}
func (p *closure) stat(t *traversal) (si *statinfo) {
    diag.errorAt(p.position, "cant stat closure %v, must expand it first", p).debug(16)
    return
}
func (p *closure) stamp(t *traversal) (file []*File, err error) {
    diag.errorAt(p.position, "cant stamp closure %v, must expand it first", p).debug(16)
    return
}
func (p *closure) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*closure); ok {
        // NOTE: don't compare the expanded value!!
        if res = p.x.cmp(ctx, a.x); res == cmpEqual && len(p.a) == len(a.a) {
            for i, t := range p.a {
                if res = t.cmp(ctx, a.a[i]); res != cmpEqual { return }
            }
        }
    }
    return
}

type selection struct {
    valbase
    t token.Token
    o Value // Object or selection
    s Value
}
func (p *selection) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *selection) Strval(ctx Context) (s string, err error) {
    if n, ok := p.o.(*ProjectName); ok && n != nil {
        ctx = closureWith(ctx, n.project.scope)
    }

    var v Value
    if v, err = p.value(ctx); err != nil {
        diag.errorAt(p.position, "%v", err)
    } else if v != nil {
        if s, err = v.Strval(ctx); err != nil {
            diag.errorAt(p.position, "%v", err).debug(1)
        }
    } else if false {
        diag.errorAt(p.position, "selection.strval: `%s` is nil", p.String())
    }
    return
}
func (p *selection) True(ctx Context) (t bool, err error) {
    var v Value
    if v, err = p.value(ctx); err == nil {
        t, err = v.True(ctx)
    }
    return
}
func (p *selection) Integer(ctx Context) (int64, error) {
    if s, err := p.Strval(ctx); err == nil {
        return strconv.ParseInt(s, 10, 64)
    } else {
        return 0, err
    }
}
func (p *selection) Float(ctx Context) (float64, error) {
    if s, err := p.Strval(ctx); err == nil {
        return strconv.ParseFloat(s, 64)
    } else {
        return 0, err
    }
}
func (p *selection) refs(ctx Context, v Value) bool { return p.o.refs(ctx, v) || p.s.refs(ctx, v) }
func (p *selection) defs(ctx Context, s ...string) []*Def {
    return append(p.o.defs(ctx, s...), p.s.defs(ctx, s...)...)
}
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
func (p *selection) object(ctx Context) (o Object, err error) {
    if s, ok := p.o.(*selection); ok {
        var v Value
        if v, err = s.value(ctx); err != nil {
            // sth's wrong!
        } else if o, _ = v.(Object); o == nil {
            diag.errorAt(p.position, "selection.object: `%s` is nil", s.String())
        }
    } else if o, ok = p.o.(Object); !ok {
        diag.errorAt(p.position, "selection.object: '%v' is not object (but %s)", p.o, typeof(p.o))
    }
    return
}
func (p *selection) value(ctx Context) (v Value, err error) {
    var o Object
    if isNil(p.s) {
        diag.errorAt(p.position, "selection prop is nil: %s", p.String()).debug(1)
    } else if o, err = p.object(ctx); err != nil {
        diag.errorAt(p.position, "get selection object failed: %v", err).debug(1)
    } else if s := ""; o != nil {
        /*if n, ok := o.(*ProjectName); ok && n != nil && n.project != nil {
            defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
        }*/
        if s, err = p.s.Strval(ctx); err == nil {
            if pn, ok := o.(*ProjectName); ok && (p.t == token.SELECT_PROG1 || p.t == token.SELECT_PROG2) {
                var entry *RuleEntry
                if entry, err = pn.project.resolveEntry(ctx, s, false); err != nil {
                    return
                } else if entry == nil {
                    diag.errorAt(p.position, "selection.value: no entry `%s` (%+v)", s, p.String())
                } else {
                    v = entry
                }
            } else if v, err = o.Get(ctx, s); err != nil {
                diag.errorAt(p.position, "%v", err)
            }
        }
    } else /*if o == nil*/ {
        diag.errorAt(p.position, "selection.value: nil object `%s`", p.String())
    }
    return
}
func (p *selection) expandible(ctx Context, w expandwhat) (res bool) {
    if res = w&expandSelection != 0; !res {
        res = p.o.expandible(ctx, w) || p.s.expandible(ctx, w)
    }
    return
}
func (p *selection) expand(ctx Context, w expandwhat) (res Value, err error) {
    if w&expandSelection != 0 {
        if res, err = p.value(ctx); err != nil {
            diag.errorAt(p.position, "selection '%v' failed: %v", p, err).debug(1)
        }
    } else if isNil(p.o) {
        return // nil object
    } else if isNil(p.s) {
        return // nil prop
    }

    var o, s Value
    if o, err = p.o.expand(ctx, w); err != nil {
        diag.errorOf(p.o, "expand '%v' failed: %v", p.o, err).debug(1)
        return
    } else if isNil(o) { o = p.o }
    if s, err = p.s.expand(ctx, w); err != nil {
        diag.errorOf(p.s, "expand '%v' failed: %v", p.s, err).debug(1)
        return
    } else if isNil(s) { s = p.s }

    if o != p.o || s != p.s { res = &selection{p.valbase,p.t,o,s}}
    return
}
func (p *selection) traverse(t *traversal) (brks breakers) {
    var ctx = positional(t, p.position)
    if options.traceTraversal { defer un(tt(t_traverse, t, p)) }
    if val, err := p.value(ctx); err != nil {
        diag.errorAt(p.position, "select value '%v' failed: %v", p, err).debug(1)
    } else if isNil(val) {
        diag.warnAt(p.position, "selected value '%v' is nil", p).debug(1)
    } else if isNone(val) {
        diag.warnAt(p.position, "selected value '%v' is none", p).debug(1)
    } else {
        brks = val.traverse(t)
    }
    return
}
func (p *selection) stat(t *traversal) (si *statinfo) {
    diag.errorAt(p.position, "cant stat selection %v, must expand it first", p).debug(1)
    return
}
func (p *selection) stamp(t *traversal) (file []*File, err error) {
    diag.errorAt(p.position, "cant stamp selection %v, must expand it first", p).debug(1)
    return
}
func (p *selection) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*selection); ok && p.t == a.t {
        if res = p.o.cmp(ctx, a.o); res == cmpEqual {
            if res = p.s.cmp(ctx, a.s); res == cmpEqual {
                // if p.t == a.t { res = cmpEqual }
            }
        }
    }
    return
}
func (p *selection) elemStr(ctx Context, o Object, k elemkind) (s string) {
    if _, ok := p.o.(*usinglist); ok { s = "usee" } else {
        s = elementString(ctx, o, p.o, k)
    }
    s += p.t.String() + elementString(ctx, o, p.s, k)
    return
}

/*
type partialMatcher interface {
    partialMatch(i interface{}) (result string, rest, stems []string, err error)
}

// TODO: endingMatcher is not implemented (e.g. $(trim-suffix .%, a.xxx b.xxx))
type endingMatcher interface {
    endingMatch(i interface{}) (result string, rest, stems []string, err error)
}
*/

// Pattern
/*type Pattern interface {
    Value
    match(i interface{}) (s string, stems []string, err error)
    stencil(stems []string) (s string, rest []string, err error)
}*/

// PercPattern represents percent pattern expressions (e.g. '%.o')
type PercPattern struct {
    valbase // TODO: supporting multiple %: foo%bar%xxx
    Prefix Value
    Suffix Value
}
func (p *PercPattern) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *PercPattern) elemStr(_ Context, o Object, k elemkind) (s string) {
    s  = elementString(nil, o, p.Prefix, 0) + `%`
    s += elementString(nil, o, p.Suffix, 0)
    return
}
func (p *PercPattern) Strval(ctx Context) (s string, err error) {
    if p.Prefix != nil {
        var v string
        if v, err = p.Prefix.Strval(ctx); err == nil {
            s = v
        } else {
            return
        }
    }
    s += "%"
    if p.Suffix != nil {
        var v string
        if v, err = p.Suffix.Strval(ctx); err == nil {
            s += v
        } else {
            return
        }
    }
    return
}
func (p *PercPattern) refs(ctx Context, v Value) bool { return p.Prefix.refs(ctx, v) || p.Suffix.refs(ctx, v) }
func (p *PercPattern) defs(ctx Context, s ...string) []*Def { return append(p.Prefix.defs(ctx, s...), p.Suffix.defs(ctx, s...)...) }
func (p *PercPattern) expandible(ctx Context, w expandwhat) bool { return p.Prefix.expandible(ctx, w) || p.Suffix.expandible(ctx, w) }
func (p *PercPattern) patterned(ctx Context) bool { return true }
func (p *PercPattern) match1(ctx Context, rep string) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("PercPattern.match")) }

    var ( prefix string; err error )
    if !(isNil(p.Prefix) || isNone(p.Prefix)) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if prefix, err = p.Prefix.Strval(ctx); err != nil {
            diag.errorOf(p.Prefix, "prefix strval '%v' failed: %v", p.Prefix, err)
            return
        } else if strings.HasPrefix(rep, prefix) {
            result = prefix
        } else {
            return
        }
    }

    var a, b = len(prefix), len(rep)
    if isNil(p.Suffix) || isNone(p.Suffix) {
        if a < b { stems, result, full = append(stems, rep[a:]), rep, true }
    } else if pp, ok := p.Suffix.(*PercPattern); a < b && ok {
        // fooxxbaryybaz -> foo%bar%baz => (foo xx bar yy baz) [xx yy]
        // fooxxx -> foo%% => (foo xxx) [xxx]
        // fooxxxbar -> foo%%bar => (foo xxx bar) [xxx]
        for ok {
            if isNil(pp.Prefix) || isNone(pp.Prefix) {
                // does nothing
            } else if s, e := pp.Prefix.Strval(ctx); e != nil {
                diag.errorOf(pp.Prefix, "strval '%v' failed: %v", pp.Prefix, e)
                return
            } else if s != "" {
                if n := strings.Index(rep[a:], s); n < 0 {
                    break
                } else {
                    var v = rep[a:a+n]
                    stems = append(stems, v)
                    result += v + s
                    a += n + len(s)
                }
            }
            var pp2 *PercPattern
            if isNil(pp.Suffix) || isNone(pp.Suffix) {
                var s = rep[a:] // let %% matches everything else
                full, stems = true, append(stems, s)
                result += s
                break
            } else if pp2, ok = pp.Suffix.(*PercPattern); ok {
                pp = pp2
                continue
            } else if s, e := pp.Suffix.Strval(ctx); e != nil {
                diag.errorOf(pp.Prefix, "strval '%v' failed: %v", pp.Suffix, e)
                return
            } else if s != "" && strings.HasSuffix(rep[a:], s) {
                if b -= len(s); a < b {
                    stems = append(stems, rep[a:b])
                    result += rep[a:]
                    full = true
                }
                break
            }
        }
    } else if a < b && p.Suffix.patterned(ctx) {
        if false {
            diag.warnOf(p.Suffix, "mixing % pattern might have performance impact: %v", p).debug(1)
        }
        for n := b-1; a < n; n -= 1 {
            if f, s, ss := p.Suffix.match(ctx, rep[n:]); f && s != "" {
                stems = append(append(stems, rep[a:n]), ss...)
                result += s // rep[a:]
                full = f
                break
            }
        }
   } else if a <= b {
        var s string
        if s, err = p.Suffix.Strval(ctx); err != nil {
            diag.errorOf(p.Suffix, "strval '%v' failed: %v", p.Suffix, err)
        } else if strings.HasSuffix(rep[a:], s) {
            if b -= len(s); a < b {
                stems = append(stems, rep[a:b])
                result = rep
                full = true
            }
        }
    } else {
        // does nothing
    }
    return
}
func (p *PercPattern) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("PercPattern.match")) }

    switch t := i.(type) {
    case string: return p.match1(ctx, t)
    case *filestub:
        if full, result, stems = p.match1(ctx, t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(ctx, filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
        }
    case *File:
        if full, result, stems = p.match1(ctx, t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(ctx, t.fullname()) // NOTE: matching the fullname form
        }
    case Value:
        if rep, err := t.Strval(ctx); err != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, err)
        } else { return p.match1(ctx, rep) }
    default:
        unreachable(fmt.Sprintf("perc.match: %T %v", i, i))
    }
    return
}
func (p *PercPattern) stencil(ctx Context, stems []string) (s string, rest []string) {
    if optionEnableBenchmarks && false { defer bench(mark(fmt.Sprintf("PercPattern.stencil(%v)", p))) }
    if optionEnableBenchspots { defer bench(spot("PercPattern.stencil")) }

    var err error
    if !(isNil(p.Prefix) || isNone(p.Prefix)) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if s, err = p.Prefix.Strval(ctx); err != nil {
            diag.errorOf(p.Suffix, "strval prefix '%v' failed: %v", p.Prefix, err).debug(1)
            return
        }
    }

    if len(stems) > 0 {
        s += stems[0]
        rest = stems[1:]
    } else if s != "" {
        // return
    }

    var v string
    if isNil(p.Suffix) || isNone(p.Suffix) {
        // done
    } else if p.Suffix.patterned(ctx) {
        // FIXME: patterns like '%%...' should use only one stem,
        // FIXME: patterns like '%xxx%...' use multiple stems.
        if v, rest = p.Suffix.stencil(ctx, rest); v != "" { s += v }
    } else if v, err = p.Suffix.Strval(ctx); err != nil {
        diag.errorOf(p.Suffix, "strval suffix '%v' failed: %v", p.Suffix, err).debug(1)
        return
    } else if v != "" {
        s += v
    }
    return
}
func (p *PercPattern) traverse(t *traversal) (brks breakers) { return t.pattern(p) }
func (p *PercPattern) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*PercPattern); ok {
        if p.Prefix.cmp(ctx, a.Prefix) == cmpEqual {
            if p.Suffix.cmp(ctx, a.Suffix) == cmpEqual {
                res = cmpEqual
            }
        }
    }
    return
}

// Check for patterns like foo%%bar
func percperc(p Value) (t bool, prefix, suffix Value) {
    if p1, ok := p.(*PercPattern); ok {
        if p2, ok := p1.Suffix.(*PercPattern); ok {
            // assert(isNone(p2.Prefix))
            prefix = p1.Prefix
            suffix = p2.Suffix
            t = true
        }
    }
    return
}

func correctPathSegForMatch(seg Value) Value {
    if bc, ok := seg.(*Barecomp); ok {
        for _, elem := range bc.Elems {
            if _, t := elem.(*Path); t { seg = nil; break }
        }
    }
    return seg
}

// GlobPattern represents glob pattern expressions (e.g. '*.o', '[a-z].o', 'a?a.o')
// 
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'     matches any sequence of non-Separator characters
//		'?'     matches any single non-Separator character
//		'[' [ '^' ] { character-range } ']'
//		        character class (must be non-empty)
//		c       matches character c (c != '*', '?', '\\', '[')
//		'\\' c      matches character c
//
//	character-range:
//		c       matches character c (c != '\\', '-', ']')
//		'\\' c      matches character c
//		lo '-' hi   matches character c for lo <= c <= hi
type GlobPattern struct {
    valbase
    Components []Value
}
func (p *GlobPattern) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *GlobPattern) elemStr(ctx Context, o Object, k elemkind) (s string) {
    for _, comp := range p.Components {
        s += elementString(ctx, o, comp, k)
    }
    return
}
func (p *GlobPattern) Strval(ctx Context) (s string, err error) {
    for _, comp := range p.Components {
        var v string
        if v, err = comp.Strval(ctx); err != nil {
            return
        }
        s += v
    }
    return
}
func (p *GlobPattern) refs(ctx Context, v Value) (res bool) {
    for _, comp := range p.Components {
        if res = comp.refs(ctx, v); res { break }
    }
    return
}
func (p *GlobPattern) defs(ctx Context, s ...string) (res []*Def) {
    for _, comp := range p.Components {
        res = append(res, comp.defs(ctx, s...)...)
    }
    return
}
func (p *GlobPattern) expandible(ctx Context, w expandwhat) (res bool) {
    for _, comp := range p.Components {
        if res = comp.expandible(ctx, w); res { break }
    }
    return
}
func (p *GlobPattern) expand(ctx Context, w expandwhat) (res Value, err error) {
    var ( components []Value; num int )
    if components, num, err = expandall1(ctx, w, p.Components...); err != nil {
        diag.errorOf(p, "expand glob components failed: %v (w=%016b)", err, w).debug(1)
    } else if num > 0 {
        res = &GlobPattern{p.valbase, components}
    }
    return
}
func (p *GlobPattern) patterned(ctx Context) bool { return true }
func (p *GlobPattern) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("GlobPattern.match")) }
    var ( pat, s string; e error )
    switch t := i.(type) {
    case string: s = t
    case *File: s = t.name
    case *filestub: s = t.name
    case Value:
        if s, e = t.Strval(ctx); e != nil {
            diag.errorOf(t, "strval '%v' failed: %v", t, e)
            return
        }
    default:
        unreachable("glob.match: %T %v", i, i)
    }
    if pat, e = p.Strval(ctx); e != nil {
        diag.errorAt(p.position, "strval '%v' failed: %v", p, e)
    } else if full, e = filepath.Match(pat, s); e != nil {
        diag.errorAt(p.position, "glob match '%s' failed: %v", pat, e)
    } else if full {
        result = s
        // FIXME: calculate stems from matching
    }
    return
}
func (p *GlobPattern) stencil(ctx Context, stems []string) (s string, rest []string) {
    unreachable(fmt.Sprintf("Unimplemented GlobPattern stencil %v (stems=%v)", p, stems))
    return
}
func (p *GlobPattern) traverse(t *traversal) (brks breakers) { return t.pattern(p) }
func (p *GlobPattern) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*GlobPattern); ok {
        if len(p.Components) == len(a.Components) {
            for i, c := range p.Components {
                if c.cmp(ctx, a.Components[i]) != cmpEqual {
                    return
                }
            }
            res = cmpEqual
        }
    }
    return
}

// TODO: implement regexp pattern
type RegexpPattern struct { valbase }
func (p *RegexpPattern) String() string { return "{RegexpPattern}" }
func (p *RegexpPattern) Strval(ctx Context) (s string, err error) { return "", nil }
func (p *RegexpPattern) patterned(ctx Context) bool { return true }
func (p *RegexpPattern) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    if optionEnableBenchspots { defer bench(spot("RegexpPattern.match")) }
    unreachable("regexp.match: %T %v", i, i)
    return
}
func (p *RegexpPattern) stencil(ctx Context, stems []string) (s string, rest []string) {
    unreachable("regexp.stencil: %v", stems)
    return
}
func (p *RegexpPattern) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*RegexpPattern); ok {
        if a != nil { /* FIXME: ... */ }
    }
    return
}
func (p *RegexpPattern) traverse(t *traversal) (brks breakers) { return t.pattern(p) }

func NewRegexpPattern(pos Position) Value {
    return &RegexpPattern{valbase{pos}} // TODO: RegexpPattern implementation
}

type Valuer interface {
    Value() Value
}

type Caller interface {
    Call(ctx Context, args... Value) (result Value)
}

type Executer interface {
    Execute(ctx Context, args... Value) (result []Value, brks breakers)
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
func Reveal(ctx Context, values ...Value) (res []Value, err error) {
    for _, v := range values {
        var t Value
        //if v, err = Reveal(v); err != nil { break }
        if t, err = v.expand(ctx, expandDelegate); err != nil {
            diag.errorOf(v, "expand '%v' failed: %v", v, err).debug(1)
            break
        } else if isNil(t) { t = v }
        res = append(res, t)
    }
    return
}

// Disclose expands closures to normal value recursively.
func Disclose(ctx Context, values ...Value) (res []Value, err error) {
    for _, v := range values {
        var t Value
        if t, err = v.expand(ctx, expandClosure); err != nil {
            diag.errorOf(v, "expand '%v' failed: %v", v, err).debug(1)
            break
        } else if isNil(t) { t = v }
        res = append(res, t)
    }
    return
}

func values(args ...interface{}) (elems []Value) {
    for _, a := range args {
        if v, ok := a.(Value); ok {
            elems = append(elems, v)
        } else if v := reflect.ValueOf(a); v.Kind() == reflect.Slice {
            for n := 0; n < v.Len(); n++ {
                elems = append(elems, values(v.Index(n).Interface())...)
            }
        } else {
            diag.error("'%v' is not value type (%T)", a, a).debug(6)
        }
    }
    return
}

// Merge lists recursively into a single list. Previously called Join.
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

// example: mergeresult(expandall(...))
func mergeresult(res []Value, err error) ([]Value, error) {
    if err == nil { res = merge(res...) }
    return res, err
}

// example: mergeresult2(expandall2(...))
func mergeresult2(res []Value, _ int, err error) ([]Value, error) {
    if err == nil { res = merge(res...) }
    return res, err
}

func trueVal(ctx Context, v Value, i bool) (res bool, err error) {
    if res = i; v != nil { res, err = v.True(ctx) }
    return
}

func int64Val(ctx Context, v Value, i int64) (res int64, err error) {
    if res = i; v != nil { res, err = v.Integer(ctx) }
    return
}

func intVal(ctx Context, v Value, i int) (res int, err error) {
    if res = i; v != nil {
        var i int64
        if i, err = v.Integer(ctx); err == nil {
            res = int(i)
        }
    }
    return
}

func uintVal(ctx Context, v Value, i uint32) (res uint32, err error) {
    if res = i; v != nil {
        var i int64
        if i, err = v.Integer(ctx); err == nil {
            res = uint32(i)
        }
    }
    return
}

func permVal(ctx Context, v Value, i uint32) (res os.FileMode, err error) {
    if i, err = uintVal(ctx, v, i); err == nil {
        res = os.FileMode(i) & os.ModePerm
    }
    return
}

var expandDepth int64 = 0
func expandall1(ctx Context, w expandwhat, values ...Value) (elems []Value, num int, err error) {
    defer func(i int64) { expandDepth = i } (expandDepth)
    if expandDepth += 1; expandDepth > 128 {
        err = fmt.Errorf("exceeds maximum expand depth")
        return
    }
    for _, elem := range values {
        var val Value
        if isNil(elem) {
            // TODO: report nil expand ??
        } else if val, err = elem.expand(ctx, w); err != nil {
            diag.errorOf(elem, "expand '%v' failed: %v", elem, err).debug(1)
            break
        }
        if isNil(val) || val == elem {
            elems = append(elems, elem)
        } else {
            elems = append(elems, val)
            num += 1
        }
    }
    return
}

func expandall2(ctx Context, w expandwhat, values ...Value) (res []Value, num int, err error) {
    if res, num, err = expandall1(ctx, w, values...); err == nil && w != 0 {
        for i, v := range res {
            if v.expandible(ctx, w) {
                t, _ := v.expand(ctx, w)
                diag.warnOf(values[i], "expand incomplete: %T %v -> %T %v (equal=%v) -> %v (w=%016b)",
                    values[i], values[i], v, v, (values[i]==v), t, w).debug(true,16)
            }
        }
    }
    return
}

func ExpandAll(ctx Context, values ...Value) (res []Value, err error) {
    res, _, err = expandall2(ctx, expandPlainValue, values...)
    return
}

func splitPathStr(pos Position, str string) (segments []Value) {
    for _, s := range strings.Split(str, PathSep) {
        // TODO: calculate position of each segment
        segments = append(segments, MakeBareword(pos, s))
    }
    return
}

func Refs(ctx Context, a Value, v Value) bool { return a.refs(ctx, v) }

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

func MakeNil(pos Position) Value { return &Nil{valbase{pos}} }
func MakeNone(pos Position) Value { return &None{valbase{pos}} }
func MakeSelection(pos Position, tok token.Token, lhs, rhs Value) Value { return &selection{valbase{pos}, tok, lhs, rhs} }
func MakeAnswer(pos Position, v bool) (res Value) {
    if v {
        res = &answer{valbase{pos},true}
    } else {
        res = &answer{valbase{pos},false}
    }
    return
}
func MakeBoolean(pos Position, v bool) (res Value) {
    if v {
        res = &boolean{valbase{pos},true}
    } else {
        res = &boolean{valbase{pos},false}
    }
    return
}
func MakeBin(pos Position, i int64) *Bin { return &Bin{integer{valbase{pos},i}} }
func MakeOct(pos Position, i int64) *Oct { return &Oct{integer{valbase{pos},i}} }
func MakeInt(pos Position, i int64) *Int { return &Int{integer{valbase{pos},i}} }
func MakeHex(pos Position, i int64) *Hex { return &Hex{integer{valbase{pos},i}} }
func MakeFloat(pos Position, f float64) *Float { return &Float{valbase{pos},f} }
func MakeDate(pos Position, s time.Time) *Date { return &Date{DateTime{valbase{pos},s}} }
func MakeTime(pos Position, t time.Time) *Time { return &Time{DateTime{valbase{pos},t}} }
func MakeRaw(pos Position, s string) *Raw { return &Raw{valbase{pos},s} }
func MakeString(pos Position, s string) *String { return &String{valbase{pos},s} }
func MakeFlag(pos Position, s string) *Flag { return &Flag{valbase{pos}, &Bareword{valbase{pos},s}} }
func MakeFlagValue(pos Position, v Value) *Flag { return &Flag{valbase{pos}, v} }
func MakeURL(pos Position, s *url.URL) *URL {
    var host, port string
    v := strings.Split(s.Host, ":")
    if len(v) == 1 { host = v[0] }
    if len(v) == 2 { host, port = v[0], v[1] }
    var password Value
    if t, ok := s.User.Password(); ok {password = MakeString(pos, t)}
    return &URL{ // FIXME: calculate component positions
        valbase: valbase{pos},
        Scheme: MakeString(pos, s.Scheme),
        Username: MakeString(pos, s.User.Username()),
        Password: password,
        Host: MakeString(pos, host),
        Port: MakeString(pos, port),
        Path: MakeString(pos, s.Path),
        Query: MakeString(pos, s.RawQuery),
        Fragment: MakeString(pos, s.Fragment),
    }
}
func MakeBareword(pos Position, word string) *Bareword { return &Bareword{valbase{pos},word} }
func MakeBarecomp(pos Position, elems... Value) *Barecomp { return &Barecomp{valbase{pos},elements{elems}} }
func MakeCompound(pos Position, elems... Value) *Compound { return &Compound{valbase{pos},elements{elems}} }
func MakeArgumented(val Value, args... Value) *Argumented { return &Argumented{val, args} }
func MakeList(pos Position, elems... Value) *List {
    if !pos.IsValid() && len(elems) > 0 {
        pos = elems[0].Position()
    }
    return &List{pos,elements{elems}}
}
func MakeGroup(pos Position, elems... Value) (v *Group) { return &Group{valbase{pos},elements{elems}} }
func MakeGlobMeta(pos Position, tok token.Token) *GlobMeta { return &GlobMeta{valbase{pos},tok} }
func MakeGlobRange(pos Position, v Value) *GlobRange { return &GlobRange{valbase{pos},v} }
func MakePath(pos Position, segments... Value) (v *Path) { return &Path{valbase{pos},elements{segments}/*, nil*/} }
func MakePathSeg(pos Position, ch rune) *PathSeg { return &PathSeg{valbase{pos},ch} }
func MakePathStr(pos Position, str string) *Path { return MakePath(pos, splitPathStr(pos, str)...) }
func MakePair(pos Position, k, v Value) (p *Pair) {
    p = &Pair{valbase{pos},nil,nil}
    p.SetKey(k)
    p.SetValue(v)
    return
}
func MakePercPattern(pos Position, prefix, suffix Value) *PercPattern {
    if prefix == nil { prefix = MakeNone(pos) }
    if suffix == nil { suffix = MakeNone(pos) }
    return &PercPattern{
        valbase: valbase{pos},
        Prefix: prefix,
        Suffix: suffix,
    }
}
func MakeGlobPattern(pos Position, components... Value) Value {
    return &GlobPattern{valbase:valbase{pos},Components:components}
}
func MakeDelegate(pos Position, tok token.Token, obj Value, args... Value) Value {
    return &delegate{valbase{pos}, tok, obj, args}
}
func MakeClosure(pos Position, tok token.Token, obj Value, args... Value) Value {
    if isNil(obj) {
        if false && pos.IsValid() { diag.errorAt(pos, "closure nil object").debug(1) }
        panic(failure{pos,"making closure on <nil> object"})
    }
    return &closure{delegate{valbase{pos}, tok, obj, args}}
}
func MakeListOrScalar(pos Position, elems []Value) (res Value) {
    if x := len(elems); x > 1 {
        res = MakeList(elems[0].Position(), elems...)
    } else if x == 1 {
        res = elems[0]
    } else {
        res = MakeNone(pos)
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
    case string:    out = MakeString(pos, v)
    case time.Time: out = &DateTime{valbase{pos},v} // FIXME: NewDate, NewTime
    case Value:     out = v
    default:    out = &Any{in} // TODO: position for any
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

func get_filename(n int) string {
    var num int
    var filename string
    var lines = strings.Split(string(debug.Stack()), "\n")
    for _, line := range lines {
        if !strings.HasPrefix(line, "\t") { continue }
        if i := strings.Index(line, ":"); num == n && i > 0 {
            filename = line[1:i]
            break
        }
        num += 1
    }
    return filename
}
