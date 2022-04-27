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
    "runtime"
	"unicode/utf8"
    "net/url"
    "reflect"
    "strconv"
    "strings"
	"errors"
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
    positionalValueCtx = true
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
const expandArgumentedTraverse = true
const (
    expandDelegate expandwhat = 1<<iota // $(...)  ->  ......
    expandClosure   // &(...)            ->  $(...)
    expandSelection // foo->bar          -> ...
    expandArgs      // $(foo $(x),$(y))  -> $(foo ...,...)
    expandArgedArgs // foo($(args))      -> foo(...)
    expandDef       // foo=...           -> ...
    expandFullName  // foobar.c          -> /path/to/foobar.c
    expandPatterned // %.proto           -> example.proto (if ctx.stems() == [example])
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

func (brks *breakers) add(ctx Context, what breakind) *breaker {
    var pos = ctx.Position()
    var brk = &breaker{ pos:pos, what:what }
    if *brks = append(*brks, brk); false {
        var t = getTargetValue(ctx) // t.String() == "bn/armv8-mont.S" &&
        if what == breakNext && strings.HasSuffix(pos.Filename, "crypto/build.smart") {
            warnstack(ctx, 3, "%v %v", t, ctx).debug(12)
        }
    }
    return brk
}

func (brks *breakers) addf(ctx Context, what breakind, s string, a... interface{}) *breaker {
    var brk = brks.add(ctx, what)
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
func (pc *prioritizedContext) autoGet(name string) (res Value, found bool) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if res, found = c.autoGet(name); found/*!isNil(res)*/ {
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
// func (pc *prioritizedContext) closureGet(name string) (res Value) {
//     for _, c := range append([]Context{ pc.Context }, pc.more...) {
//         if res = c.closureGet(name); !(isNil(res) /*|| isNone(res) || isUndef(res)*/) {
//             break
//         }
//     }
//     return
// }
// func (pc *prioritizedContext) closureSet(name string, val Value) (res Value, okay bool) {
//     for _, c := range append([]Context{ pc.Context }, pc.more...) {
//         if res, okay = c.closureSet(name, val); okay { break }
//     }
//     return
// }
// func (pc *prioritizedContext) closureResolveObject(pos Position, name string) (obj Object) {
//     for _, c := range append([]Context{ pc.Context }, pc.more...) {
//         if obj = c.closureResolveObject(pos, name); !(isNil(obj) /*|| isNone(res) || isUndef(res)*/) {
//             break
//         }
//     }
//     return
// }
// func (pc *prioritizedContext) closureResolveEntry(pos Position, name string) (entry Entry) {
//     for _, c := range append([]Context{ pc.Context }, pc.more...) {
//         if entry = c.closureResolveEntry(pos, name); entry != nil {
//             break
//         }
//     }
//     return
// }

func prioritize(ctxs ...Context) (ctx Context) {
    if n := len(ctxs); n > 1 {
        ctx = &prioritizedContext{ctxs[0], ctxs[1:]}
    } else if n == 1 {
        ctx = ctxs[0]
    } else {
        panic("prioritize zero contexts")
    }
    return
}

const (
    recursiveTraversalClosurePre = false
    recursiveTraversalClosurePost = false
    recursiveTraversalClosure = true
)
type closureContext struct {
    Context
    scopes []*Scope
}
func (cc *closureContext) inner() Context { return cc.Context }
func (cc *closureContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("closure{%s}", cc.Context)
    } else {
        return cc.Context.String()
    }
}
func (cc *closureContext) Scope() (scope *Scope) {
    if len(cc.scopes) > 0 {
        scope = cc.scopes[0]
    } else {
        scope = cc.Context.Scope()
    }
    return
}
func (cc *closureContext) forScopes(work func(*Scope) bool) (scopes []*Scope) {
    if scopes = cc.closureScopes(); work != nil {
        for _, scope := range scopes {
            if scope != nil && work(scope) { break }
        }
    }
    return
}
func (cc *closureContext) Project() (proj *Project) {
    for _, scope := range cc.closureScopes() {
        if proj = scope.project; proj != nil { break }
    }
    if proj == nil { proj = cc.Context.Project() }
    return
}
func (cc *closureContext) closure() *closureContext { return cc }
func (cc *closureContext) closureScopes() (scopes []*Scope) {
    if scopes = cc.Context.closureScopes(); false {
        scopes = append(scopes, cc.scopes...)
    } else if false {
    ForScopes:
        for _, s1 := range cc.scopes {
            for _, s2 := range scopes {
                // discard if s2 has s1 as outer scope (as a sub scope)
                if s2 == s1 || s2.hasOuter(s1) { continue ForScopes }
            }
            scopes = append(scopes, s1)
        }
    } else {
        scopes = append(scopes, cc.scopes...)
    }
    return
}

type spawnClosureContext struct { closureContext }
func (cc *spawnClosureContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("spawn-%s", cc.closureContext.String())
    } else {
        return cc.Context.String()
    }
}
func (cc *closureContext) spawn() Context {
    var ctx = cc.Context
    if t, ok := ctx.(*traverseContext); ok { ctx = t.spawn() }
    return &spawnClosureContext{closureContext{ ctx, cc.scopes }}
}

func closureProjects(ctx Context) (projects []*Project) {
    if false { return []*Project{ ctx.Project() } }

ForScopes:
    for _, scope := range ctx.closureScopes() {
        if scope.project == nil { continue }
        for _, project := range projects {
            if project == scope.project || project.hasBase(scope.project) {
                continue ForScopes
            }
        }
        projects = append(projects, scope.project)
        if false { break }
    }
    return
}

func closureGet(ctx Context, name string) (res Value) {
    var err error
    for _, scope := range ctx.closureScopes() {
        if scope.project == nil {
            if _, res = scope.Find(name); !isNil(res) { break }
        } else {
            var pos = ctx.Position()
            if !pos.IsValid() { pos = scope.position }
            if !pos.IsValid() { pos = scope.project.position }
            if scope != scope.project.scope {
                if _, res = scope.Find(name); !isNil(res) { break  }
            }
            if res, err = scope.project.resolveObject(ctx, name); err != nil {
                erro(ctx, "resolve '%s' failed: %v", name, err).debug(1)
                break
            } else if isNil(res) {
                res, _ = ctx.autoGet(name)
            }
        }
        if !isNil(res) { break }
    }
    return
}

func closureSet(ctx Context, name string, val Value) (prev Value, okay bool) {
    for _, scope := range ctx.closureScopes() {
        if def := scope.FindDef(name); def != nil {
            prev = def.value
            def.val(ctx, val)
            okay = true
            break
        }
    }
    return
}

func closureResolveObject(ctx Context, pos Position, name string) (obj Object) {
    var (
        infos = false && strings.HasPrefix(name, "@")
        scope *Scope
        err error
    )
    if infos { defer func() {
        var val Value
        if obj != nil { val, _ = obj.expand(ctx, expandPlainValue) }
        warn(ctx, "%v: name = %s", scope.project, name).at(scope.position)
        warn(ctx, "%v: %v", scope.project, scope).at(scope.position)
        warn(ctx, "%v: %v", scope.project, ctx).at(scope.position)
        warn(ctx, "%s: %T, %v", scope.project, obj, obj).at(pos)
        warn(ctx, "%s: %T, %v", scope.project, val, val).at(pos).debug(24)
    } () }
    for _, scope = range ctx.closureScopes() {
        var ctx Context = positional(ctx, scope.position)
        if infos { warn(ctx, "%s", scope).debug(1) }
        if scope.project == nil || scope != scope.project.scope {
            if _, obj = scope.Find(name); isNil(obj) {
                // fallthrough
            } else if def, ok := obj.(*Def); ok && (def.origin == DefAuto || def.origin == DefArg) {
                if o, ok := ctx.closureResolveAuto(name); ok && !isNil(o) { obj = o }
                if infos {
                    var proj = def.OwnerProject()
                    val, _ := obj.expand(ctx.closure(), expandPlainValue)
                    va2, _ := ctx.closure().autoGet(name)
                    ob1, _ := ctx.closureResolveAuto(name)
                    warn(ctx, "%v: %v %v => %T %v", proj, def.origin, def.name, def.value, def.value)
                    warn(ctx, "%v: obj = %T %v", proj, obj, obj)
                    warn(ctx, "%v: ob1 = %T %v", proj, ob1, ob1)
                    warn(ctx, "%v: val = %T %v", proj, val, val)
                    warn(ctx, "%v: va2 = %T %v", proj, va2, va2)
                    warn(ctx, "%v: %v", proj, scope)
                    warn(ctx, "%v: %v", proj, ctx).debug(1)
                }
                break
            } else {
                if false && infos { warn(ctx, "%v: %v", obj.OwnerProject(), obj).debug(1) }
                break // got the obj
            }
        }
        if scope.project != nil {
            if obj, err = scope.project.resolveObject(ctx, name); err != nil {
                erro(ctx, "resolve object '%s' in '%s' failed: %v", name, scope.project, err)
                erro(ctx, "failed from here '%s' in '%s'", name, scope).at(pos).debug(1)
                break
            }
        }
        if isNil(obj) && false { obj = closureResolveObject(ctx.inner(), pos, name) }
        if!isNil(obj) { if infos { warn(ctx, "%v", obj).debug(1) }; break }
    }
    return
}

func closureResolveEntry(ctx Context, pos Position, name string) (entries *ResolveEntries) {
    var (
        scope *Scope
        err error
    )
    for _, scope = range ctx.closureScopes() {
        if project := scope.project; project == nil {
            // none
        } else if entries, err = project.resolveEntries(ctx, name, false, /*true*/false); err != nil {
            erro(ctx, "resolve entry '%s' in '%s' failed: %v", name, project, err).at(pos).debug(1)
            break
        } else if entries == nil && false {
            entries = closureResolveEntry(ctx.inner(), pos, name)
        }
        if entries != nil { break }
    }
    return
}

func closureWith(ctx Context, pos Position, scopes ...*Scope) (res Context) {
    if c, ok := ctx.(*closureContext); false && ok {
        res = closureWith(c.Context, pos, scopes...)
    } else {
        res = &closureContext{ ctx, scopes }
    }
    return
}

func refdef(ctx Context, val Value, origin Origin) (res bool) {
    for _, def := range val.defs(ctx) {
        if def.origin == origin || origin == defany {
            return true
        }
    }
    return
}

type spawnTraverseContext struct { traverseContext }
func (sc *spawnTraverseContext) String() string { return fmt.Sprintf("spawn-%s", sc.traverseContext.String()) }

// traverseContext is a single thread traverse context, for traversing in a new goroutine,
// a spawned traversal must be used and then merge.
type traverseContext struct {
    Context

    start time.Time // start time

    execRec map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    targets []Value // all targets def
    grepped []Value
    grepping bool

    traceLevel int

    interpreted []interpreter

    print bool // printing work directories (Entering/Leaving)
}
func (t *traverseContext) inner() Context { return t.Context }
func (t *traverseContext) String() string {
    if fullContextStringer {
        return fmt.Sprintf("traversal{%s}", t.Context)
    } else {
        return t.Context.String()
    }
}

func (t *traverseContext) level(n int) { t.traceLevel += n }
func (t *traverseContext) trace(a ...interface{}) { printIndentDots(t.traceLevel, a...) }
func (t *traverseContext) tracef(s string, a ...interface{}) { printIndentDots(t.traceLevel, fmt.Sprintf(s, a...)) }

func (t *traverseContext) caller() *traverseContext { return t.Context.traversal() }
func (t *traverseContext) traversal() *traverseContext { return t }
func (t *traverseContext) traversed(target Value) (targets []Value) {
    if !isTrivial(target) {
        t.targets = append(t.targets, target)
        targets = t.targets
    }
    return
}

func entryStr(ctx Context, entry Entry) (str, ent, tar string) {
    if s, e := entry.Strval(ctx); e == nil { ent = s } else {
        erro(ctx, "strval '%v' failed: %v", entry, e).debug(1)
        return
    }
    if target, found := ctx.autoGet("@"); !found || isTrivial(target) {
        str = ent // ...
    } else if tar, _ = target.Strval(ctx); ent != tar {
        str = fmt.Sprintf("%s(%s)", ent, tar)
    } else {
        str = ent
    }
    return
}

func entryStr1(ctx Context, entry Entry) (s string) {
    s, _, _ = entryStr(ctx, entry)
    return
}

func infostack(ctx Context, n int, s string, a ...interface{}) *diagPoint { return callstack(ctx, n, diagInfo , s, a...) }
func errostack(ctx Context, n int, s string, a ...interface{}) *diagPoint { return callstack(ctx, n, diagError, s, a...) }
func warnstack(ctx Context, n int, s string, a ...interface{}) *diagPoint { return callstack(ctx, n, diagWarn , s, a...) }
func callstack(ctx Context, n int, dt diagType, s string, a ...interface{}) (point *diagPoint) {
    var (
        proj = ctx.Project()
        entry = ctx.entry()
    )
    if s != "" { point = diag(ctx, dt, s, a...) }
    if entry == nil {
        point = ctx.diag(dt, "in project %v:", proj)
        if false && proj != nil { point.at(proj.position) }
        for last, i := ctx.Position(), ctx.inner(); i != nil; i = i.inner() {
            if pos := i.Position(); !pos.Equals(&last) {
                point = ctx.diag(dt, "%v: from here", proj).at(pos)
                last = pos
            }
        }
        return
    } else if proj == nil {
        proj = entry.OwnerProject()
        point = ctx.diag(dt, "%v: %v -- %v", proj, entry, ctx)
    }

    var str, _, _ = entryStr(ctx, entry)
    if pc := ctx.programCtx(); pc == nil {
        point = ctx.diag(dt, "%v: %v", proj, str)
    } else {
        point = ctx.diag(dt, "%v: %v -- %v", proj, str, pc).at(pc.prog.position)
        for last := pc.prog.position; pc != nil; pc = pc.Context.programCtx() {
            var pos = pc.prog.position
            if !pos.SameLine(&last) {
                var str, _, _ = entryStr(pc, pc.entry())
                if pc != nil { str += " ..." }
                point = ctx.diag(dt, "%v: %v", proj, str).at(pos)
                last = pos
                n -= 1
            }
            if n == 0 {
                if false && pc != nil { point = ctx.diag(dt, "%v: ...", proj).at(pos) }
                break
            }
        }
    }
    return
}

func (t *traverseContext) arguments() []Value { return nil }
func (t *traverseContext) argumented() *argumentedContext { return nil }
func (t *traverseContext) argumentedSet([]Value) []Value { return nil }
func (t *traverseContext) spawn() Context {
    return &spawnTraverseContext{traverseContext{
        Context: t.Context,
        print:   t.print,
        execRec: make(map[Value]int),
        start:   time.Now(),
    }}
}

func exists(ctx Context, v Value) bool {
    // FIXME: returns true if existenceMatterless ??
    return v != nil && v.stat(ctx).exists() == existenceConfirmed
}

func (t *traverseContext) depth() (res int) {
    for c := t.caller(); c != nil; c = c.caller() { res += 1 }
    return
}

func (t *traverseContext) calleeError(err error) {
    if err != nil {
        t.calleeErrsM.Lock()
        t.calleeErrs = append(t.calleeErrs, err)
        t.calleeErrsM.Unlock()
    }
}

// DEPRECATED
func traverseAny(ctx Context, i interface{}) {
    var err error
    var pos = ctx.Position() //t.def.target.position
    if v := reflect.ValueOf(i); v.Kind() == reflect.Slice {
        for n := 0; err == nil && n < v.Len(); n++ {
            traverseAny(ctx, v.Index(n).Interface())
        }
    } else if i == nil {
        erro(ctx, "updating nil prerequisite").at(pos)
    } else if value, ok := i.(Value); !ok {
        erro(ctx, "'%v' is invalid", value).at(pos)
    } else if isNil(value) { // this could happen
        erro(ctx, "updating nil prerequisite").at(pos)
    } else {
        value.traverse(ctx)
    }
    return
}

func traverseFile(ctx Context, file *File) (okay bool, brks breakers) {
    var (
        proj = ctx.Project()
        program = ctx.program()
        targetVal = getTargetValue(ctx)
    )
    if isTrivial(targetVal) {
        erro(ctx, "target is <nil>").at(program.position).debug(1)
        return
    }

    ctx = positional(ctx, file.position)

    var (
        t = ctx.traversal()
        projects = closureProjects(ctx)
        concreteEntries = make(map[Entry]bool)
        concreteList []Entry
        stemmedList []*stemmed
        err error
    )
    defer func() {
        // Note that the file maybe not traversed yet at this point. But we
        // still have to check mod-time.
        var (
            a = targetVal.stat(ctx).mod()
            b = file.stat(ctx).mod()
        )

        // a.IsZero() indicates the target not exists
        if !a.IsZero() && b.After(a) { targetVal.updated(true) }

        // Add to the $^ or $| list
        ctx.traversed(file)
    } ()
    // Try import filemap projects first! See also `files (-import ...)`
    if file.filemap != nil { if proj := file.filemap.project; proj != nil {
        for _, p := range projects { if p == proj { goto afterFilemapProject } }
        projects = append(projects, proj);               afterFilemapProject:
    }}

checkFileEntries:
    for _, project := range projects {
        var entries *ResolveEntries
        if entries, err = project.resolveEntries(ctx, file.name, t.grepping, false); err != nil {
            prompt(ctx, "%v: traverse failed, project %s\n", file.fullname(), proj)
            erro(ctx, "resolve entry '%v' failed: %v", file.name, err).at(file.position)
            errostack(positional(ctx, file.position), -1, "%v:", file.name).debug(1)
            return
        } else if entries == nil {
            continue
        } else {
            for _, entry := range entries.all {
                if _, ok := concreteEntries[entry]; !ok {
                    concreteList = append(concreteList, entry)
                    concreteEntries[entry] = true
                }
            }
        }
    }
    for _, entry := range concreteList {
        if entry != nil && targetVal != entry {
            if brks = entry.traverse(ctx); !brks.has() {
                if okay = file.exists(); okay { break }
            } else if tb := brks.not(breakCase, breakDone, breakNext); tb.has() {
                brks = brks.not(breakFail, breakErro)
                prompt(ctx, "%s: traverse file entry failed, project %v\n", file, proj)
                for _, brk := range tb {
                    switch brk.what {
                    case breakFail: erro(ctx, "broken traversal for stemmed file entry %v failed: %v", file, brk.message).at(brk.pos)
                    case breakErro: erro(ctx, "broken traversal for stemmed file entry %v with error: %v", file, brk.error).at(brk.pos)
                    default: erro(ctx, "broken traversal for stemmed file entry %v (%v)", file, brk.what).at(brk.pos)
                    }
                }
                errostack(ctx, 3, "%v: %v", entry, ctx).debug(6)
                return
            } else {
                if tb = brks.of(breakNext); tb.has() {
                    if brks = brks.not(breakNext); !brks.has() { continue }
                }
                if tb = brks.of(breakCase, breakDone); tb.has() {
                    if brks = brks.not(breakCase, breakDone); !brks.has() { okay = true }
                }
            }
            if okay { break }
        }
    }

    for _, project := range projects {
        var pats = project.resolvePatterns(ctx, file.name)
        stemmedList = append(stemmedList, pats...)
    }
    for _, entry := range stemmedList {
        if false && file.name == "bn/armv8-mont.S" {
            warn(ctx, "%v: %v; %v", entry, entry.Stems, ctx.stems()).at(entry.Position())
            warnstack(ctx, 3, "%v; %v; %v", file, entry, brks).debug(16)
        }
        if brks = entry.file(ctx, file); !brks.has() {
            if okay = file.exists(); okay { break }
        } else if tb := brks.not(breakCase, breakDone, breakNext); tb.has() {
            brks = brks.not(breakFail, breakErro)
            prompt(ctx, "%s: traverse file entry failed, project %v\n", file, entry.OwnerProject())
            for _, brk := range tb {
                switch brk.what {
                case breakFail: erro(ctx, "broken traversal for stemmed file entry %v failed: %v", file, brk.message).at(brk.pos)
                case breakErro: erro(ctx, "broken traversal for stemmed file entry %v with error: %v", file, brk.error).at(brk.pos)
                default: erro(ctx, "broken traversal for stemmed file entry %v (%v)", file, brk.what).at(brk.pos)
                }
            }
            errostack(ctx, 3, "%v: %v", entry, ctx).debug(6)
            return
        } else if tb = brks.of(breakNext);    tb.has() {
            if false && file.name == "bn/armv8-mont.S" {
                warn(ctx, "%v: %v; %v", entry, entry.Stems, ctx.stems()).at(entry.Position())
                for _, brk := range brks {
                    warn(ctx, "%v: %v", entry, brk.what)
                }
                warnstack(ctx, 3, "%v %v", file, entry).debug(16)
            }
            if brks = brks.not(breakNext); !brks.has() { continue }
        } else if tb = brks.of(breakCase, breakDone);    tb.has() {
            if brks = brks.not(breakCase, breakDone); !brks.has() { okay = true }
        }
        if brks.has() {
            prompt(ctx, "%v: traverse failed, project %s\n", file.fullname(), entry.OwnerProject())
            for _, brk := range brks {
                erro(ctx, "%v: broken for stemmed entry %v (%v)", file.fullname(), entry, brk.what).at(brk.pos)
            }
            erro(ctx, "%v: broken for stemmed entry %v", file.fullname(), entry).at(entry.position)
            errostack(positional(ctx, entry.position), 3, "%v: %v", file.fullname(), ctx).debug(6)
            return
        } else if okay {
            break
        }
    }

    for _, project := range projects {
        if okay { break } else if file != nil {
            if okay = file.info != nil; okay { break }
            okay = file.searchInMatchedPaths(ctx, project)
        }
    }

    if err != nil {
        prompt(ctx, "%v: traverse file failed; projects %v, %v\n", file.fullname(), proj, projects)
        errostack(ctx, 5, "%v, %v: error: %v", proj, file, err).debug(8)
    } else if !okay && (len(ctx.stems()) == 0 || ctx.mustExists()) {
        if filemap := file.filemap; filemap != nil && filemap.project != nil {
            var s = filepath.Join(filepath.Base(filepath.Join(file.dir, file.sub)), file.name)
            if f := filemap.project.FindFile(ctx, s); f != nil && f.fullname() == file.fullname() {
                if false { warn(ctx, "%v: %v -> %v (%s)", proj, file.name, f, s).debug(1) }
                file, projects = f, []*Project{ filemap.project }
                goto checkFileEntries
            }
        }
        prompt(ctx, "%v: traverse file failed; projects %v, %v\n", file.fullname(), proj, projects)
        for i, concrete := range concreteList { erro(ctx, "concrete: %d. %v (%d programs)", i, concrete, len(concrete.Programs())).at(concrete.Position()) }
        for i, stemmed  := range stemmedList  { erro(ctx, "stemmed: %d. %v", i, stemmed).at(stemmed.position) }
        for i, brk := range brks { erro(ctx, "%v, %v: %d. %v", proj, file, i, brk.what) }
        if args := t.arguments(); len(args) > 0 { erro(ctx, "%v, %v: arguments %v", proj, file, args) }
        erro(ctx, "%v: no rules for %v, required by %v", proj, file, targetVal) //proj.concrete
        errostack(ctx, 15, "(%T): (exists=%v)", ctx, file.exists()).debug(64)
        brks.add(ctx, breakErro).error = fileNotFoundError{ proj, file }
    } else if !okay && len(ctx.stems()) > 0 {
        if false { brks.add(ctx, breakNext).scope = breakTrave }
    }
    return
}

func traverseString(ctx Context, targetVal Value, target string) (okay bool, brks breakers) {
    var currentTargetValue = getTargetValue(ctx)
    if isNil(currentTargetValue) {
        prompt(ctx, "%s: zero closure projects\n", target)
        errostack(ctx, 3, "target is <nil>").debug(6)
        return
    }

    var projects = closureProjects(ctx)
    if len(projects) == 0 {
        prompt(ctx, "%s: zero closure projects\n", target)
        erro(ctx, "no projects to traverse '%v' (%s)", targetVal, target)
        erro(ctx, "%v: closure %v", target, len(ctx.closureScopes()))
        errostack(ctx, 3, "%v: %v", target, ctx).debug(8)
        return
    }

    var (
        pos Position = ctx.Position()
        concreteEntries = make(map[Entry]bool)
        concreteList []Entry
        stemmedList []*stemmed
        file *File // if target is file
        err error
    )
    defer func() {
        // Note that the file maybe not traversed yet at this point. But we
        // still have to check mod-time.
        if file != nil {
            var (
                a = currentTargetValue.stat(ctx).mod()
                b = file.stat(ctx).mod()
            )
            // a.IsZero() indicates the target not exists
            if !a.IsZero() && b.After(a) { currentTargetValue.updated(true) }
            if !file.position.IsValid() { file.position = pos }
            ctx.traversed(file)// Add to the $^ or $| list
        } else if true {
            if targetVal == nil { targetVal = MakeString(pos, target) }
            ctx.traversed(targetVal)
        }

        if false && brks.has(breakNext) && strings.Contains(target, "fcrange") {
            warn(ctx, "%v %v", target, projects).debug(6)
        }
    } ()

    var t = ctx.traversal()
    for _, project := range projects {
        var entries *ResolveEntries
        if entries, err = project.resolveEntries(ctx, target, t.grepping, false); err != nil {
            prompt(ctx, "%s: traverse entry failed, project %v\n", target, project)
            erro(ctx, "%s: resolve entry failed: %v", target, err).at(pos)
            errostack(ctx, 3, "%s: resolve entry failed: %v", target, err).debug(10)
            return
        } else if entries == nil {
            continue
        } else {
            for _, entry := range entries.all {
                if _, ok := concreteEntries[entry]; !ok {
                    concreteList = append(concreteList, entry)
                    concreteEntries[entry] = true
                }
            }
        }
    }

    for _, entry := range concreteList {
        var project = entry.OwnerProject()
        if entry != nil && currentTargetValue != entry {
            if w, ok := currentTargetValue.(*Bareword); ok && w.string == target {
                continue // target resolve to itself, does nothing
            } else if brks = entry.traverse(ctx); !brks.has() {
                file, _ = entry.Target().(*File)
                okay = true
                return
            }
            if tb := brks.not(breakCase, breakDone, breakNext); tb.has() {
                brks = brks.not(breakFail, breakErro);
                var str, ent, _ = entryStr(ctx, entry)
                prompt(ctx, "%s: traverse entry failed, project %v\n", ent, project)
                for _, brk := range tb {
                    switch brk.what {
                    case breakFail: erro(ctx, "%v: traverse %v failed: %v", project, str, brk.message).at(brk.pos)
                    case breakErro: erro(ctx, "%v: traverse %v error: %v", project, str, brk.error).at(brk.pos)
                    default: erro(ctx, "%v: broken traversal (%v)", project, brk.what).at(brk.pos)
                    }
                }
                errostack(ctx, 3, "%v: %v: %v", str, project, ctx).at(pos).debug(6)
                return
            } else {
                if tb = brks.of(breakNext); tb.has() {
                    if brks = brks.not(breakNext); !brks.has() { continue }
                }
                if tb = brks.of(breakCase, breakDone); tb.has() {
                    if brks = brks.not(breakCase, breakDone); !brks.has() { okay = true }
                }
            }
            if okay { break }
        }
    }

    for _, project := range projects {
        var obj Object
        if obj, err = project.resolveObject(ctx, target); err != nil {
            prompt(ctx, "%v: traverse failed, project %s\n", target, project)
            erro(ctx, "%s: resolve object '%s' failed: %v", target, err).at(pos)
            errostack(ctx, 3, "%v: %v: %v", target, project, ctx).debug(6)
            return
        } else if isTrivial(obj) {
            // does nothing here and keep trying FindFile
        } else if brks = obj.traverse(ctx); brks.has() {
            prompt(ctx, "%v: traverse failed, project %s\n", target, project)
            erro(ctx, "%s: broken traversal '%v' (%T) (project=%v)", target, obj, obj, project).at(pos)
            errostack(ctx, 3, "%v: %v: %v", target, project, ctx).debug(6)
            return
        } else if _, ok := obj.(*ProjectName); ok {
            // solved, no need to check against patterns
            return
        }
    }

    for _, project := range projects {
        stemmedList = append(stemmedList, project.resolvePatterns(ctx, target)...)
    }
    for _, entry := range stemmedList {
        var project = entry.OwnerProject()
        if false && ctx.configuration() && strings.Contains(target, ".c.include") {
             warn(ctx, "%v %v %v %v", project, ctx.Project(), target, entry).debug(1)
        }
        if brks = entry.string(ctx, targetVal, target); !brks.has() {
            okay = true; break // continue
        } else if tb := brks.not(breakCase, breakDone, breakNext); tb.has() {
            brks = brks.not(breakFail, breakErro)
            ctx = positional(ctx, entry.Position())
            prompt(ctx, "%v: traverse failed, project %s\n", target, project)
            for _, brk := range tb {
                switch brk.what {
                case breakFail: erro(ctx, "%v: %v", target, brk.message).debug(1)
                case breakErro: erro(ctx, "%v: %v", target, brk.error).debug(1)
                }
            }
            erro(ctx, "broken traversal for stemmed entry '%v' in %v", entry, project).at(pos).debug(1)
            return
        } else if tb = brks.of(breakNext           );    tb.has() {
            if brks = brks.not(breakNext           ); !brks.has() { continue }
        } else if tb = brks.of(breakCase, breakDone);    tb.has() {
            if brks = brks.not(breakCase, breakDone); !brks.has() { okay = true }
        }
        if fn := file.fullname(); brks.has() {
            ctx = positional(ctx, entry.Position())
            prompt(ctx, "%v: traverse failed, project %s\n", fn, project)
            for _, brk := range brks {
                erro(ctx, "%v: broken for stemmed entry %v (%v)", fn, entry, brk.what).at(brk.pos)
            }
            erro(ctx, "%v: broken for stemmed entry %v", fn, entry)
            errostack(ctx, 3, "%v: %v", fn, ctx).debug(6)
            return
        } else if okay {
            break
        }
    }

    for _, project := range projects {
        if false && target == "fcrange.c" {
            file = project.FindFile(ctx, target)
            warn(ctx, "%v: %v %v", project, target, file.info).debug(16)
        }
        if file = project.FindFile(ctx, target); file != nil {
            file.position = pos // Change the position for tracing
            if okay, brks = traverseFile(ctx, file); brks.has() {
                return
            } else if okay {
                break
            } else if file.info != nil {
                okay = true // it's good
            } else if file != nil {
                okay = file.searchInMatchedPaths(ctx, project)
            }
            if !okay && file.name != target {
                var alt = file //project.FindFile(file.name)
                if alt != nil { okay = alt.sub == "-" || alt.exists() }
            }
            if okay { return } else if false { break } // Done!
        }
    }

    if proj := ctx.Project(); err != nil {
        prompt(ctx, "%s: traverse failed, project %s\n", target, proj)
        errostack(ctx, 6, "%v: %v", proj, ctx).debug(16)
    } else if !okay && !ctx.configuration() && (len(ctx.stems()) == 0 || ctx.mustExists()) {
        if file != nil {
            brks.add(ctx, breakErro).error = fileNotFoundError{proj, file}
            ctx = positional(ctx, file.position)
        } else {
            brks.add(ctx, breakErro).error = targetNotFoundError{proj, target}
            if targetVal != nil { ctx = positional(ctx, targetVal.Position()) }
        }
        if file != nil {
            prompt(ctx, "%v: traverse file failed; projects %v, %v\n", file.fullname(), proj, projects)
        } else {
            prompt(ctx, "%v: traverse target failed; projects %v, %v\n", target, proj, projects)
        }
        for _, brk := range brks {
            switch brk.what {
            case breakErro: erro(ctx, "%v", brk.error)
            case breakFail: erro(ctx, "%v", brk.message)
            default       : erro(ctx, "%v: %v, %v", brk.what, proj, target)
            }
        }
        for i, c := range ctx.closureScopes() { erro(ctx, "%v: closure: %v. %v", proj, i, c) }
        for i, concrete := range concreteList { erro(ctx, "concrete: %d. %v (%d programs)", i, concrete, len(concrete.Programs())).at(concrete.Position()) }
        for i, stemmed  := range stemmedList  { erro(ctx, "stemmed: %d. %v", i, stemmed).at(stemmed.position) }
        errostack(ctx, 12, "%v", ctx).debug(16)
    } else if !okay && len(ctx.stems()) > 0 {
        if true { brks.add(ctx, breakNext).scope = breakTrave }
        if false && strings.HasSuffix(target, "armv8-mont.pl") {
            warnstack(ctx, 3, "%v; %v %v; %v, %v", currentTargetValue, targetVal, target, file, projects).debug(6)
        }
    }
    return
}

func traversePattern(ctx Context, pat Value) (brks breakers) {
    var (
        pos = pat.Position()
        stems = ctx.stems()
        rest []string
        okay bool
        val Value
    )
    if val, rest = pat.stencil(ctx, stems); /*isTrivial*/isNil(val) {
        erro(ctx, "empty stencil: %v %v", pat, ctx.stems()).at(pos).debug(1)
        return
    } else if false && val.patterned(ctx) {
        erro(ctx, "stencil failed: %v -> %T %v; stems=%v", pat, val, val, ctx.stems()).at(pos).debug(1)
        return
    } else if len(rest) > 0 {
        erro(ctx, "partial stencil: %v: %T %v, %v, %v", pat, val, val, rest, ctx.stems()).at(pos).debug(1)
        panic(fmt.Sprintf("%T %v", val, val))
    } else if file, ok := val.(*File); ok {
        okay, brks = traverseFile(ctx, file)
    } else if proj := ctx.Project(); proj == nil {
        erro(ctx, "no project in context: %T", ctx).at(pos).debug(1)
        return
    } else if s, err := val.Strval(ctx); err != nil { // TODO: refine this branch and others
        erro(ctx, "strval %T %v failed: %v", val, val, err).at(pos).debug(1)
        return
    } else if file = proj.FindFile(ctx, s); file != nil {
        file.position = pos
        if okay, brks = traverseFile(ctx, file); !okay {
            if true /*&& strings.Contains(s, "armv8-mont")*/ && brks.has() {
                warn(ctx, "%v; %v; %v", file, brks, ctx.stems()).debug(1)
            }
        }
    } else {
        if false && strings.Contains(s, "arm64cpuid") {
            warnstack(ctx, 5, "%v -> %T %v -> %v", pat, val, val, s).at(pos).debug(24)
        }
        if false && strings.Contains(s, "armv8-mont.S") {
            warn(ctx, "%v; %v; %v; %v", s, pat, stems, ctx).at(pat.Position()).debug(1)
        }
        okay, brks = traverseString(ctx, val, s)
        if true && strings.Contains(s, "armv8-mont.S") && brks.has() {
            warn(ctx, "%v; %v; %v; %v; %v", s, pat, stems, brks, ctx).at(pat.Position()).debug(1)
        }
    }

    if !okay && len(stems) > 0 {
        brk := brks.add(ctx, breakNext)
        brk.scope = breakTrave
        brk.value = pat
    }
    return
}

func getHashDir(ctx Context, k []byte) string {
    var dir string
    if program := ctx.program(); program != nil {
        dir = program.project.tmpPath
    } else if project := ctx.Project(); project != nil {
        dir = project.tmpPath
    } else if scope := ctx.Scope(); scope != nil && scope.project != nil {
        dir = scope.project.tmpPath
    }
    var h = fmt.Sprintf("%x", k[:2]) // HEX of the first two bytes
    return filepath.Join(dir, ".hash", h[0:1], h[1:2], h[2:3], h[3:])
}

func getCmdHash(ctx Context, values ...Value) (k, v HashBytes, err error) {
    var program = ctx.program()
    var targetVal, targetStr = getTargetValueString(ctx)
    if isTrivial(targetVal) {
        erro(ctx, "trivial $@").at(program.position).debug(1)
        return
    } else if targetStr == "" {
        erro(ctx, "trivial $@").at(program.position).debug(1)
        return
    }

    var (
        key = sha256.New()
        val = sha256.New()
    )
    fmt.Fprintf(key, "%s", program.project.absPath)
    fmt.Fprintf(key, "%v", targetStr)

    for _, value := range values {
        if false {
            // FIXME: Strval() varies when &(var) is used
            if targetStr, err = value.Strval(ctx); err != nil { return }
            fmt.Fprintf(val, "%v", targetStr)
        } else {
            fmt.Fprintf(val, "%v", value)
        }
    }
    copy(k[:], key.Sum(nil))
    copy(v[:], val.Sum(nil))
    return
}

func updateRecipesHash(ctx Context) (k, v HashBytes, err error) {
    var program = ctx.program()
    if k, v, err = getCmdHash(ctx, program.recipes...); err != nil {
        erro(ctx, "hashing recipes failed: %v", err).at(program.position).debug(1)
        return
    }

    var dir = getHashDir(ctx, k[:])
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

func isRecipesOutdated(ctx Context) (outdated bool, err error) {
    var k, v HashBytes
    if program := ctx.program(); program == nil {
        erro(ctx, "no program in context %v", ctx).debug(1)
        return
    } else if k, v, err = getCmdHash(ctx, program.recipes...); err != nil {
        erro(ctx, "compute recipes hash failed: %v", err).at(program.position).debug(1)
        return
    }

    var dir = getHashDir(ctx, k[:])
    var name = filepath.Join(dir, fmt.Sprintf("%x", k))
    if f, e := os.Open(name); e == nil {
        defer f.Close()

        var h []byte
        if n, e := fmt.Fscanf(f, "%x", &h); e != nil {
            err = e
        } else if n == 1 {
            outdated = !bytes.Equal(v[:], h)
        }
    }
    return
}

func wait(ctx Context, opts ...bool) (target Value, files []*File, execRes *ExecResult, err error) {
    // Waiting for prerequisites
    var (
        pos Position = ctx.Position()
        calleeErrs []error
    )
    if t := ctx.traversal(); t != nil {
        //t.group.Wait()
        t.calleeErrsM.Lock()
        calleeErrs = t.calleeErrs; t.calleeErrs = nil
        t.calleeErrsM.Unlock()
    }

    if target = getTargetValue(ctx); isTrivial(target) {
        erro(ctx, "target is <none>").at(pos)
        errostack(ctx, 8, "target is <none>").debug(8)
        return
    } else if n := len(calleeErrs); n > 0 /*&& t.stems == nil*/ {
        var (
            numRealErrs = 0
            targetPos = pos//target.Position()
            targetValuePos = target.Position()
        )
        for _, err := range calleeErrs {
            erro(ctx, "%v: %v", target, err).debug(1)
            numRealErrs += 1
        }
        if numRealErrs == 0 { return } // simply return if no real errors
        if !pos.Equals(&targetPos) {
            var s string; if n > 1 { s = "s" }
            erro(ctx, "%d error%s while waiting prerequisites for '%v'", n, s, target).at(targetPos).debug(1)
        }

        var v = target
        if l, ok := v.(*List); ok && l.Len() == 1 { v = l.Elems[0] }
        if targetValuePos.IsValid() && !targetValuePos.Equals(&targetPos) {
            if f, ok := v.(*File); ok && f.filemap != nil {
                erro(ctx, "waiting for '%v'", target).at(targetValuePos)
                erro(ctx, "via pattern '%v' (of %v)", v, f.filemap.project).of(f.filemap.patts[0]).debug(1)
            } else {
                erro(ctx, "waiting for '%v'", target).at(targetValuePos).debug(1)
            }
        }
        if def, ok := v.(*Def); ok && target != v && target != def.value { // trace source Def in diagnostics
            erro(ctx, "waiting for def '%v': %v", def.name, def.value).of(def.value).debug(1)
        }
        return
    }

    var (
        optReportFileUpdates  = len(opts) > 0 && opts[0]
        optWaitForExecResult  = len(opts) > 1 && opts[1]
        optStampCurrentTarget = len(opts) > 2 && opts[2]
    )
    if optWaitForExecResult {
        // Waiting for command (shell/python/etc.) exec result
        if bv, has := ctx.autoGet("-"); has && !isNil(bv) && !isNone(bv) {
            var ok bool
            if execRes, ok = bv.(*ExecResult); ok {
                //execRes.wg.Wait()
            }
        }
    }
    if !optStampCurrentTarget {
        // done!
    } else if files, err = target.stamp(ctx); err != nil {
        if p := target.Position(); p.IsValid() { erro(ctx, "%v", err).at(p) }
        erro(ctx, "%v", err).at(pos).debug(1)
        return
    } else if optReportFileUpdates {
        reportFileUpdates(ctx, ctx.traversal().start, files)
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
    stencil(Context, []string) (val Value, rest []string)

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

    stat(Context) (*statinfo)

    // Stamp the value if it's a file (aka. update FileInfo).
    stamp(Context) ([]*File, error)

    updated(...bool) bool
    updatedDeps(...Value) []Value

    // Delete the file (if it is).
    delete(Context) ([]*File, error)

    traverse(Context) breakers
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
func (_ *valbase) stencil(_ Context, stems []string) (val Value, rest []string) { return }
func (_ *valbase) stat(ctx Context) (si *statinfo) { return }
func (_ *valbase) stamp(ctx Context) (file []*File, err error) { return }
func (_ *valbase) updated(_ ...bool) bool { return false }
func (_ *valbase) updatedDeps(_ ...Value) []Value { return nil }
func (_ *valbase) delete(ctx Context) (file []*File, err error) { return }
func (_ *valbase) traverse(ctx Context) (brks breakers) { return }
func (_ *valbase) _match(ctx Context, p Value, i interface{}) (full bool, s string, stems []string) {
    var ( v string; e error )
    if v, e = p.Strval(ctx); e == nil {
        var is string
        switch t := i.(type) {
        case string: is = t
        case Value:
            if is, e = t.Strval(ctx); e != nil {
                erro(ctx, "strval '%v' error: %v", t, e).of(t).debug(1)
                return
            }
        default:
            erro(ctx, "%T: matching unsupported value: %T %v", p, i, i).of(p).debug(1)
            return
        }
        if strings.HasPrefix(is, v) { s, full = v, (len(v) == len(is)) }
    } else {
        erro(ctx, "strval '%v' error: %v", p, e).of(p).debug(1)
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
func (p *Argumented) updated(v ...bool) bool { return p.value.updated(v...) }
func (p *Argumented) updatedDeps(v ...Value) []Value { return p.value.updatedDeps(v...) }
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
        erro(ctx, "expand '%v' failed: %v", p.value, err).of(p.value).debug(1)
        return
    } else if isNil(val) { val = p.value }
    if w&expandArgedArgs != 0 {
        if args, num, err = expandall1(ctx, w, p.args...); err != nil {
            erro(ctx, "expand args '%v' failed: %v", p.args, err).of(p.value).debug(1)
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
func (p *Argumented) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p.value.stencil(ctx, stems)
    return
}

func (p *Argumented) delete(ctx Context) ([]*File, error) { return p.value.delete(ctx) }
func (p *Argumented) stamp(ctx Context) ([]*File, error) { return p.value.stamp(ctx) }
func (p *Argumented) stat(ctx Context) (si *statinfo) {
    // FIXME: p.value might be not the real target (depending on the arguments)
    return p.value.stat(ctx)
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

func (p *Argumented) traverse(ctx Context) (brks breakers) {
    //!< IMPORTANT! - Don't merge-expand arguments here!
    //!< Arguments should be passed to program.execute as it is.
    var args []Value
    if expandArgumentedTraverse {
        var proj = ctx.Project()
        // NOTE: expand here to avoid args being expanded in the wrong context
        const w = expandPlainValue|expandPairVal|expandPatterned//|expandFullName
        for _, a := range p.args {
            if val, err := a.expand(ctx, w); err != nil {
                erro(ctx, `expand "%v" failed: %v`, a, err).debug(1)
            } else if !isNil(val) && val != a { a = val }
            // TODO: deal with pattern args using expandPatterned instead of stenciling:
            if true && a.patterned(ctx) { if stems := ctx.stems(); len(stems) > 0 {
                if val, rest := a.stencil(ctx, stems); len(rest) > 0 {
                    erro(ctx, "partial stencil: %v, %T %v, %v, %v", a, val, val, rest, stems).of(a).debug(1)
                    panic(fmt.Sprintf("%T %v", val, val))
                } else if file, okay := val.(*File); okay {
                    a = file
                } else if str, err := val.Strval(ctx); err != nil {
                    erro(ctx, "strval %v failed: %v", val, err).of(a).debug(1)
                    panic(fmt.Sprintf("%T %v", val, val))
                } else if file := proj.FindFile(ctx, str); file != nil {
                    a = file
                } else {
                    a = val //MakeString(a.Position(), str)
                }
            }}
            args = append(args, a)
        }
        /*if s := p.value.String(); s == "archive" || s == "program" {
            warn(ctx, "%v: %v -> %v", proj, p.args, args).debug(1)
        }*/
    } else {
        args = p.args
    }
    return p.value.traverse(&argumentedContext{ ctx, args })
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

// aka. isNil(v) || isNone(v) || isUndef(v) || isEmpty(v)
func isTrivial(v Value) (t bool) {
    switch a := v.(type) {
    case *None, *Nil, *unresolvedobject: t = true
    case *String: t = a.string == ""
    case *List: t = len(a.Elems) == 0 ||
        (len(a.Elems) == 1 && isTrivial(a.Elems[0]))
    default: t = isNil(v)
    }
    return
}
func isEmptyList(v Value) (t bool) {
    if l, ok := v.(*List); ok && len(l.Elems) == 0 { t = true }
    return
}
func isEmpty(v Value) (t bool) {
    switch a := v.(type) {
    case *String: t = a.string == ""
    case *List: t = len(a.Elems) == 0 ||
        (len(a.Elems) == 1 && isEmpty(a.Elems[0]))
    }
    return
}
func isUndef(v Value) (t bool) { _, t = v.(*unresolvedobject); return }
func isNone(v Value) (t bool) {
    switch a := v.(type) {
    case *None: t = true
    case *List: t = len(a.Elems) == 0 ||
        (len(a.Elems) == 1 && (isNone(a.Elems[0]) || isNil(a.Elems[0])))
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
func (p *Any) stencil(ctx Context, stems []string) (val Value, rest []string) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        val, rest = v.stencil(ctx, stems)
    }
    return
}
func (p *Any) delete(ctx Context) (files []*File, err error) {
    if a, ok := p.value.(Value); ok { files, err = a.delete(ctx) }
    return
}
func (p *Any) updated(v ...bool) (res bool) {
    if p.value == nil {
        // does nothing
    } else if val, ok := p.value.(Value); ok {
        res = val.updated(v...)
    }
    return
}
func (p *Any) updatedDeps(v ...Value) (res []Value) {
    if p.value == nil {
        // does nothing
    } else if val, ok := p.value.(Value); ok {
        res = val.updatedDeps(v...)
    }
    return
}
func (p *Any) stamp(ctx Context) (files []*File, err error) {
    if a, ok := p.value.(Value); ok { files, err = a.stamp(ctx) }
    return
}
func (p *Any) stat(ctx Context) (si *statinfo) {
    if v, ok := p.value.(Value); ok && v != nil { si = v.stat(ctx) }
    return
}
func (p *Any) expand(ctx Context, w expandwhat) (res Value, err error) {
    if val, ok := p.value.(Value); ok && !isNil(val) {
        if res, err = val.expand(ctx, w); err != nil {
            erro(ctx, "expand '%v' failed: %v", val, err).of(p).debug(1)
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
func (p *Any) traverse(ctx Context) (brks breakers) {
    if v, ok := p.value.(Value); ok { brks = v.traverse(ctx) }
    return
}

type negative struct { valbase; x Value }
func (p *negative) refs(ctx Context, o Value) bool { return p.x.refs(ctx, o) }
func (p *negative) defs(ctx Context, s ...string) []*Def { return p.x.defs(ctx, s...) }
func (p *negative) expandible(ctx Context, w expandwhat) bool { return p.x.expandible(ctx, w) }
func (p *negative) expand(ctx Context, w expandwhat) (res Value, err error) {
    var val Value
    if val, err = p.x.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", p.x, err).of(p.x).debug(1)
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
        erro(ctx, "truthify '%v' failed: %v", p.x, err).at(p.position).debug(1)
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
func (p *negative) traverse(ctx Context) (brks breakers) {
    if p.x != nil { brks = p.x.traverse(ctx) }
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
func (p *boolean) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
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
func (p *answer) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
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
func (p *option) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
    return
}

type prediction struct {
    boolean
    reason string
}

func MakePrediction(pos Position, val bool, reason string) *prediction {
    return &prediction{boolean{valbase{pos}, val}, reason}
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
func (p *Bin) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
    return
}

type Oct struct { integer }
func (p *Oct) String() string { return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8)) }
func (p *Oct) Strval(ctx Context) (string, error) { return strconv.FormatInt(int64(p.int64),8), nil }
func (p *Oct) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Oct) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }

type Int struct { integer }
func (p *Int) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) Strval(ctx Context) (string, error) { return strconv.FormatInt(int64(p.int64),10), nil }
func (p *Int) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Int) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }

type Hex struct { integer }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *Hex) Strval(ctx Context) (string, error) { return strconv.FormatInt(int64(p.int64),16), nil }
func (p *Hex) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Hex) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }

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
func (p *Float) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
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
func (p *DateTime) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
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
func (p *Date) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
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
func (p *Time) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
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
func (p *URL) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
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
func (p *Raw) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}

type String struct { valbase; string }
func (p *String) String() string { return p.elemStr(nil, nil, 0) }
func (p *String) Strval(ctx Context) (s string, _ error) {
    if false {
        s = strings.Replace(p.string, "\\\"", "\"", -1)
    } else {
        s = p.string
    }
    return
}
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
func (p *String) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *String) traverse(ctx Context) (brks breakers) {
    _, brks = traverseString(ctx, p, p.string)
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

// Punctuations: | ; ,
type Punctuation struct { valbase; tok token.Token }
func (p *Punctuation) String() string { return p.tok.String() }
func (p *Punctuation) Strval(ctx Context) (string, error) { return p.tok.String(), nil }
func (p *Punctuation) True(ctx Context) (bool, error) { return false, nil }
func (p *Punctuation) Integer(ctx Context) (int64, error) { return 0, nil }
func (p *Punctuation) Float(ctx Context) (float64, error) { return 0, nil }
func (p *Punctuation) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Punctuation); ok {
        if p.tok == a.tok {
            res = cmpEqual
        } else if p.tok > a.tok {
            res = cmpSmaller
        } else if p.tok < a.tok {
            res = cmpGreater
        }
    }
    return
}
func (p *Punctuation) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return }
func (p *Punctuation) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Punctuation) traverse(ctx Context) (brks breakers) { return }

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
func (p *Bareword) expand(ctx Context, w expandwhat) (res Value, err error) {
    if false && w&expandFullName != 0 {
        if str, err := p.Strval(ctx); err != nil {
            erro(ctx, "strval '%v' failed: %v", p, err).of(p).debug(1)
            return nil, err
        } else if file := ctx.Project().FindFile(ctx, str); file != nil {
            res, err = file.expand(ctx, w)
        }
    } else {
        res = p // optional, return nil is fine
    }
    return
}
func (p *Bareword) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    full, s, stems = p._match(ctx, p, i)
    return
}
func (p *Bareword) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *Bareword) traverse(ctx Context) (brks breakers) {
    _, brks = traverseString(ctx, p, p.string)
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
func (p *Qualiword) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *Qualiword) traverse(ctx Context) (brks breakers) {
    _, brks = traverseString(ctx, p, p.String())
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
            erro(ctx, "truthify '%v' failed: %v", elem, err).of(elem).debug(1)
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
    if false && w&expandFullName != 0 {
        if str, err := p.Strval(ctx); err != nil {
            erro(ctx, "strval '%v' failed: %v", p, err).of(p).debug(1)
            return nil, err
        } else if file := ctx.Project().FindFile(ctx, str); file != nil {
            return file.expand(ctx, w)
        }
    }

    var ( elems []Value; num int )
    if elems, num, err = expandall1(ctx, w, p.Elems...); err != nil {
        erro(ctx, "expand '%v' failed: %v", p, err).of(p).debug(1)
    } else if num > 0 {
        res = &Barecomp{p.valbase, elements{elems}}
    }
    return
}
func (p *Barecomp) traverse(ctx Context) (brks breakers) {
    ctx = positional(ctx, p.Position())
    if target, err := p.Strval(ctx); err == nil {
        if true {
            // fallthrough...
        } else if proj := ctx.Project(); proj == nil {
            // fallthrough...
        } else if file := proj.FindFile(ctx, target); file != nil {
            file.position = p.position
            brks = file.traverse(ctx)
            return
        }

        _, brks = traverseString(ctx, p, target)
    } else {
        erro(ctx, "strval '%v' error: %v", p, err).debug(1)
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
    if false {
        full, s, stems = p._match(ctx, p, i)
        if false && strings.HasPrefix(p.String(), "llvm.tools.") && strings.HasPrefix(fmt.Sprintf("%s", i), "llvm.tools.") {
            for n, elem := range p.Elems {
                var a, b, c = elem.match(ctx, i)
                warn(ctx, "%d %v: %v %v %v; %v %v %v", n, elem, a, b, c, full, s, stems).debug(1)
            }
        }
    } else {
        var ( is string; n int; elem Value )
        switch t := i.(type) {
        case string: is = t
        case Value:
            var e error
            if is, e = t.Strval(ctx); e != nil {
                erro(ctx, "strval '%v' error: %v", t, e).of(t).debug(1)
                return
            }
        default:
            erro(ctx, "%T: matching unsupported value: %T %v", p, i, i).of(p).debug(1)
            return
        }
        if is == "" { return }
        for n, elem = range p.Elems {
            var _, t, ss = elem.match(ctx, is)
            if t == "" { break } else {
                stems = append(stems, ss...)
                is = is[len(t):]
                s += t
            }
        }
        if is == "" && n == len(p.Elems)-1 { full = true }
        if false && strings.HasPrefix(p.String(), "llvm.tools.") && strings.HasPrefix(fmt.Sprintf("%s", i), "llvm.tools.") {
            warn(ctx, "%v: %v %v %v; %s", p, full, s, stems, is).debug(1)
        }
    }
    return
}
func (p *Barecomp) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var (
        elems []Value
        changed int
    )
    rest = stems
    for _, elem := range p.Elems {
        var t Value
        if t, rest = elem.stencil(ctx, rest); t != elem {
            changed += 1
        }
        elems = append(elems, t)
    }
    if changed > 0 {
        val = MakeBarecomp(p.position, elems...)
    } else {
        val = p
    }
    return
}
func (p *Barecomp) Combine(ctx Context, x Value) {
    if o, ok := x.(*Barecomp); ok {
        for _, elem := range o.Elems { p.Combine(ctx, elem) }
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
    var name Value

    if w&expandFullName != 0 {
        var file = p.File
        if file == nil {
            if str, err := p.Name.Strval(ctx); err != nil {
                erro(ctx, "strval '%v' failed: %v", p.Name, err).of(p.Name).debug(1)
                return nil, err
            } else {
                file = ctx.Project().FindFile(ctx, str)
            }
        }
        if file == nil {
            // fallthrough
        } else if name, err = file.expand(ctx, w); err != nil {
            erro(ctx, "expand '%v' failed: %v", file, err).of(file).debug(1)
            return
        } else if !isNil(name) && name != file {
            return name, nil
        }
    }

    if name, err = p.Name.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", p.Name, err).of(p.Name).debug(1)
    } else if !isNil(name) && name != p.Name {
        res = &Barefile{p.valbase, name, p.File}
    }
    return
}
func (p *Barefile) traverse(ctx Context) (brks breakers) {
    ctx = positional(ctx, p.position)
    if p.File == nil { // it happens if p.Name refers argument
        var ( target string; err error )
        if target, err = p.Strval(ctx); err != nil {
            erro(ctx, "strval '%v' failed: %v", p, err).of(p).debug(1)
            return
        }

        for _, project := range closureProjects(ctx) {
            if p.File = project.FindFile(ctx, target); p.File != nil { break }
        }
        if p.File == nil {
            erro(ctx, "barefile '%s' not found", target).at(p.position).debug(1)
            return
        }
    }
    if p.File != nil { brks = p.File.traverse(ctx) } else {
        erro(ctx, "barefile '%s' is nil", p).at(p.position).debug(1)
    }
    return
}
func (p *Barefile) delete(ctx Context) (files []*File, err error) {
    if p.File != nil { files, err = p.File.delete(ctx) }
    return
}
func (p *Barefile) stamp(ctx Context) (files []*File, err error) {
    if p.File != nil { files, err = p.File.stamp(ctx) }
    return
}
func (p *Barefile) stat(ctx Context) (si *statinfo) {
    if p.File != nil { si = p.File.stat(ctx) }
    return
}
func (p *Barefile) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Barefile); ok { res = p.Name.cmp(ctx, a.Name) }
    return
}
func (p *Barefile) patterned(ctx Context) bool { return p.Name.patterned(ctx) }
func (p *Barefile) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    if false && p.File != nil {
        full, s, stems = p.File.match(ctx, i)
    } else {
        full, s, stems = p.Name.match(ctx, i)
    }
    return
}
func (p *Barefile) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var name Value
    if p.File != nil {
        val, rest = p.File.stencil(ctx, stems)
    } else if name, rest = p.Name.stencil(ctx, stems); name != p.Name {
        val = &Barefile{p.valbase, name, p.File}
    } else {
        val = p
    }
    return
}


func barefileize(ctx Context, targets ...Value) []Value {
    var project = ctx.Project()
    for i, target := range targets {
        if target.patterned(ctx) { continue }
        switch t := target.(type) {
        case *Bareword:
            if file := project.FindFile(ctx, t.string); file != nil {
                targets[i] = &Barefile{ Name:target, File:file }
                file.position = target.Position()
            }
        case *Barecomp:
            if t.expandible(ctx, expandClosure) || refdef(ctx, t, DefArg) {
                break
            } else if s, err := t.Strval(ctx); err != nil {
                erro(ctx, "strval '%v' failed: %v", t, err).of(target)
            } else if file := project.FindFile(ctx, s); file != nil {
                targets[i] = &Barefile{ Name:target, File:file }
                file.position = target.Position()
            }
        case *Argumented:
            t.value = barefileize(ctx, t.value)[0]
            t.args  = barefileize(ctx, t.args...)
        }
    }
    return targets
}

var ErrBadPattern = errors.New("syntax error in pattern")

// modified copy of filepath.hasMeta
func globHasMeta(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}

// modified copy of filepath.getEsc
func globGetEsc(chunk string) (r rune, nchunk string, err error) {
	if len(chunk) == 0 || chunk[0] == '-' || chunk[0] == ']' {
		err = ErrBadPattern
		return
	}
	if chunk[0] == '\\' && runtime.GOOS != "windows" {
		chunk = chunk[1:]
		if len(chunk) == 0 {
			err = ErrBadPattern
			return
		}
	}
	r, n := utf8.DecodeRuneInString(chunk)
	if r == utf8.RuneError && n == 1 {
		err = ErrBadPattern
	}
	nchunk = chunk[n:]
	if len(nchunk) == 0 {
		err = ErrBadPattern
	}
	return
}

// modified copy of filepath.scanChunk
func globScanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true // TODO: support both '*' and '**'
	}
	inrange := false
	var i int
Scan:
	for i = 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			if runtime.GOOS != "windows" {
				// error check handled in matchChunk: bad pattern.
				if i+1 < len(pattern) {
					i++
				}
			}
		case '[':
			inrange = true
		case ']':
			inrange = false
		case '*':
			if !inrange {
				break Scan
			}
		}
	}
	return star, pattern[0:i], pattern[i:]
}

// modified copy of filepath.matchChunk
// stems: all values matched ? and [...]
func globMatchChunk(chunk, s string) (stems []string, rest string, ok bool, err error) {
	// failed records whether the match has failed.
	// After the match fails, the loop continues on processing chunk,
	// checking that the pattern is well-formed but no longer reading s.
	failed := false
	for len(chunk) > 0 {
		if !failed && len(s) == 0 {
			failed = true
		}
		switch chunk[0] {
		case '[':
			// character class
			var r rune
			if !failed {
				var n int
				r, n = utf8.DecodeRuneInString(s)
				s = s[n:]
			}
			chunk = chunk[1:]
			// possibly negated
			negated := false
			if len(chunk) > 0 && chunk[0] == '^' {
				negated = true
				chunk = chunk[1:]
			}
			// parse all ranges
			match := false
			nrange := 0
			for {
				if len(chunk) > 0 && chunk[0] == ']' && nrange > 0 {
					chunk = chunk[1:]
					break
				}
				var lo, hi rune
				if lo, chunk, err = globGetEsc(chunk); err != nil {
					return stems, "", false, err
				}
				hi = lo
				if chunk[0] == '-' {
					if hi, chunk, err = globGetEsc(chunk[1:]); err != nil {
						return stems, "", false, err
					}
				}
				if lo <= r && r <= hi {
					match = true
				}
				nrange++
			}
			if match == negated {
				failed = true
			} else {
                stems = append(stems, string(r))
            }

		case '?':
			if !failed {
				if s[0] == filepath.Separator {
					failed = true
				}
				_, n := utf8.DecodeRuneInString(s)
                stems = append(stems, s[:n])
				s = s[n:]
			}
			chunk = chunk[1:]

		case '\\':
			if runtime.GOOS != "windows" {
				chunk = chunk[1:]
				if len(chunk) == 0 {
					return stems, "", false, ErrBadPattern
				}
			}
			fallthrough

		default:
			if !failed {
				if chunk[0] != s[0] {
					failed = true
				}
				s = s[1:]
			}
			chunk = chunk[1:]
		}
	}
	if failed {
		return stems, "", false, nil
	}
	return stems, s, true, nil
}

// modified copy of filepath.Match
func globMatch(pattern, name string) (matched bool, stems []string, err error) {
    var _pattern, _name = pattern, name
    var dbg = false && (
        (_pattern == "lib*.a" && _name == "libunwind.a") ||
        (_pattern == "libun????.a" && _name == "libunwind.a") ||
        (_pattern == "lib[a-z][^0-9]????.a" && _name == "libunwind.a") ||
        (_pattern == "lib?++.a" && _name == "libc++.a"))
    if dbg {
        if dbg {
            fmt.Fprintf(stderr, "(%v, %v):\n", _pattern, _name)
        }
        defer func() {
            if dbg {
                fmt.Fprintf(stderr, "    matched=%v, stems=%v; (%v, %v)\n", matched, stems, pattern, name)
            }
        } ()
    }
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string
        var _p = pattern
		star, chunk, pattern = globScanChunk(pattern)
        if dbg {
            fmt.Fprintf(stderr, "    scan: (%v) -> star=%v, chunk=%v, pattern=%v\n", _p, star, chunk, pattern)
        }
		if star && chunk == "" {
			// Trailing * matches rest of string unless it has a /.
			if matched = !strings.Contains(name, PathSep); matched {
                if false { stems = append(stems, name) }
            }
            return
		}
		// Look for match at current position.
		ss, t, ok, err := globMatchChunk(chunk, name)
        if dbg {
            fmt.Fprintf(stderr, "    match: (%v, %v) -> ss=%v, t=%v, ok=%v\n", chunk, name, ss, t, ok)
        }
        if err == nil && len(ss) > 0 { stems = append(stems, ss...) }
        // if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (len(t) == 0 || len(pattern) > 0) {
            name = t
			continue
		}
		if err != nil {
			return false, stems, err
		}
        if star {
			// Look for match skipping i+1 bytes. Cannot skip /.
			for i := 0; i < len(name) && name[i] != filepath.Separator; i++ {
				ss, t, ok, err := globMatchChunk(chunk, name[i+1:])
                if dbg {
                    fmt.Fprintf(stderr, "    match: name=%v, (%v, %v), %s -> ss=%v, t=%v, ok=%v\n", name, chunk, name[i+1:], name[:i+1], ss, t, ok)
                }
                if ok {
                    // if we're the last chunk, make sure we exhausted the name
					if len(pattern) == 0 && len(t) > 0 {
						continue
					}
                    stems = append(stems, name[:i+1])
					name = t
					continue Pattern
				}
				if err != nil {
					return false, stems, err
				}
			}
		}
		return false, stems, nil
	}
	return len(name) == 0, stems, nil
}

// globMatchFile - Glob matching each component of the filename against the
// glob value. It checks in two different ways. If the filename and the
// glob pattern has the some number of components (splitted by PathSep),
// all components are compared. If the pattern has only one component,
// the last filename component is compared with the pattern, and the prefix
// components are returned in 'pre'.
func globMatchFile(ctx Context, patVal Value, filename string, tailMatch bool) (matched bool, pre string, stems []string) {
    switch patVal.(type) {
    default: // good to go!
    case *List:
        erro(ctx, "invalid glob matching pattern: %v", patVal).of(patVal).debug(8)
        return
    }

    var patStr, err = patVal.Strval(ctx)
    if err != nil { return }

    var patList = strings.Split(filepath.Clean(patStr), PathSep)
    if len(patList) == 0 { return } // FIXME: match any?

    var st []string
    var srcList = strings.Split(filepath.Clean(filename), PathSep)
    if len(patList) == len(srcList) { // src/*.o  <->  src/foo.o
        for i, pat := range patList { // Matching all components
            if matched, st, _ = globMatch(pat, srcList[i]); !matched { return }
            stems = append(stems, st...)
        }
    } else if !(len(patList) == 1 && len(srcList) > 1) {
        // Done!
    } else if tailMatch && true { // *.o|foo.o  <->  src/foo.o
        // NOTE: partially matching only the last part is logically incorrect!
        //       for example of this wrong match: stdint.h <-> isl/stdint.h
        for i, j := len(patList)-1, len(srcList)-1; -1 < i && -1 < j; i, j = i-1, j-1 {
            if v, st, _ := globMatch(patList[i], srcList[j]); !v {
                pre = filepath.Join(srcList[:j]...)
                return
            } else {
                matched = true
                stems = append(stems, st...)
            }
        }
    } else if tailMatch && false { // *.o|foo.o  <->  src/foo.o
        if matched, st, _ = globMatch(patList[0], srcList[len(srcList)-1]); matched {
            pre = filepath.Join(srcList[:len(srcList)-1]...)
            stems = append(stems, st...)
            return
        }
    } else if matched, st, _ = globMatch(patList[0], filename); matched {
        stems = append(stems, st...)
        return // *.o|foo.o  <->  src/foo.o
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
        erro(ctx, "expand '%v' failed: %v", p.Chars, err).of(p.Chars).debug(1)
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
            erro(ctx, "undef path segment (%T)", seg).at(seg.Position())
            erro(ctx, "… from this context: %s", ctx).at(ctx.Position()).debug(16)
            return
        } else if v, e = seg.Strval(ctx); e != nil {
            erro(ctx, "%v: %v", seg, e).at(seg.Position()).debug(1)
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
            erro(ctx, "truthify path element '%v' failed: %v", elem, err).at(p.position).debug(1)
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
        erro(ctx, "expand path elems failed: %v", err).at(p.position).debug(1)
    } else if num > 0 {
        res = &Path{p.valbase, elements{elems}}
    }
    return
}
func (p *Path) pathname(ctx Context, stems []string) (pathname string, err error) {// the addressed file target
    var (
        val Value
        rest []string // unmatched path segmants
    )
    if len(stems) == 0 {
        if pathname, err = p.Strval(ctx); err != nil {
            erro(ctx, "strval '%v' failed: %v", p, err).at(p.position).debug(1)
        }
    } else if val, rest = p.stencil(ctx, stems); len(rest) > 0 {
        //err = errorf(p.position, "partial match: %v", rest)
    } else if pathname, err = val.Strval(ctx); err != nil {
        erro(ctx, "strval '%v' failed: %v", p, err).at(p.position).debug(1)
    }
    return
}
func (p *Path) delete(ctx Context) (files []*File, err error) {
    var pathname string
    if positionalValueCtx { ctx = positional(ctx, p.position) }
    if pathname, err = p.Strval(ctx); err == nil {
        if pathname == "" {
            erro(ctx, "no pathname for `%s`", p)
        } else if file := stat(ctx,pathname,"","",nil); file != nil {
            if files, err = file.delete(ctx); err != nil {
                erro(ctx, "stamp: %v (%v)", err, file)
            }
        }
    }
    return
}
func (p *Path) stamp(ctx Context) (files []*File, err error) {
    var pathname string
    if positionalValueCtx { ctx = positional(ctx, p.position) }
    if pathname, err = p.Strval(ctx); err == nil {
        if pathname == "" {
            erro(ctx, "no pathname for `%s`", p)
        } else if file := stat(ctx,pathname,"","",nil); file != nil {
            if files, err = file.stamp(ctx); err != nil {
                erro(ctx, "stamp: %v (%v)", err, file)
            }
        }
    }
    return
}
func (p *Path) stat(ctx Context) (si *statinfo) {
    var (
        pathname string // the addressed file target
        err error
    )
    ctx = positional(ctx, p.position)
    if pathname, err = p.pathname(ctx, ctx.stems()); err != nil {
        erro(ctx, "pathname error: %v", err)
    } else if pathname == "" {
        erro(ctx, "pathname is empty: %v", p)
    } else if file := stat(ctx, pathname, "", "", nil); file != nil {
        si = &statinfo{ file: file }
    }
    return
}
func (p *Path) traverse(ctx Context) (brks breakers) {
    ctx = positional(ctx, p.position)

    var (
        pathname string
        err error
    )
    if p.patterned(ctx) && len(ctx.stems()) == 0 {
        erro(ctx, "empty stems to traverse pattern: %v", p).at(p.position).debug(8)
        return
    } else if pathname, err = p.pathname(ctx, ctx.stems()); err == nil && pathname == "" {
        erro(ctx, "path matches no target: %v", p).at(p.position).debug(1)
        return
    } else if err != nil {
        erro(ctx, "compute pathname failed: %v", err).at(p.position).debug(1)
        return
    }

    var okay bool
    // Stat the file by pathname.
    if file := stat(ctx, pathname, "", ""/*, nil*/); file != nil {
        file.traverse(ctx)
    } else if okay, brks = traverseString(ctx, p, pathname); !okay && len(ctx.stems()) > 0 {
        if false { brks.add(ctx, breakNext).scope = breakTrave }
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
        erro(ctx, "expand path elems failed: %v", err).at(pos).debug(1)
        return
    }
    for _, elem := range xelems {
        if p, ok := elem.(*Path); ok {
            var ( ev []Value; n int )
            if ev, n, err = expandPathElems(ctx, w, p.Elems...); err != nil {
                erro(ctx, "expand sub path '%v' failed: %v", elem, err).of(elem).debug(1)
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
                    erro(ctx, "strval '%v' failed: %v", v, err).at(v.position).debug(1)
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
        if false { erro(ctx, "empty: %v", str).at(p.position) }
        return
    }
    if segs, _, err = expandPathElems(ctx, expandPlainValue, p.Elems...); err != nil {
        erro(ctx, "failed to expand path '%v': %v", p, segs).at(p.position).debug(1)
        return
    }

    const warns = false
    var (
        //warns = p.String() == "llvm/%%"
        infos = warns
        lenSegs = len(segs)
        lenSrcs = len(srcs)
        lastSuf Value
        res []string
        n, m int
    )
    if warns {
        defer func() {
            prompt(ctx, "%v: %v %v %v\n", p, result, full, stems)
            warn(ctx, "%v: %s (%d, %d; %d, %d)", p, str, n, lenSegs, m, lenSrcs)
            warnstack(ctx, 3, "%v: %T", p, ctx).debug(8)
        } ()
    }
SegsSrcsLoop:
    for ; n < lenSegs && m < lenSrcs; {
        var si, seg, src = n, segs[n], srcs[m]
        if cs := correctPathSegForMatch(seg); cs != nil { seg = cs } else {
            erro(ctx, "invalid path seg: %v (%T)", seg, seg).of(seg).debug(1)
            break SegsSrcsLoop
        }

        var (
            pp, pre, suf = percperc(seg)
            ss []string
            s    string
            f    bool
        )
        if pp {
            if infos { info(ctx, "%d: path=%v seg=%v (%T) src=%v pp=%v pre=%v suf=%v res=%v stems=%v srcs[%d]=%s lenSegs=%d", si, p, seg, seg, src, pp, pre, suf, res, stems, m, src, lenSegs).of(p).debug(1) }
            if !isTrivial(lastSuf) && !isTrivial(pre) {
                erro(ctx, "the continual %%/%% makes no sense").of(seg)
                break SegsSrcsLoop
            }
            if /*ps, ok := seg.(*PathSeg);*/ src == "" /*&& ok && (ps.rune == 0 || ps.rune == '/')*/ { // for root path seg '/'
                // NOTE: seg could also be % or %% here
                res   = append(res, "") // for '/'
                stems = append(stems, "")
                //info(ctx, "%v %v; %v %v; %v %v", p, seg, res, stems, ps, ok).of(p)
            }
            lastSuf = suf
            n += 1 // move forward to the next seg
            m += 1 // move forward to the next src
        } else if f, s, ss = seg.match(ctx, src); f || s == src {
            if infos { info(ctx, "%d: path=%v seg=%v (%T); str=%v srcs[%d]=%v -> f=%v s=%v ss=%v => res=%v stems=%v", si, p, seg, seg, str, m, src, f, s, ss, res, stems).of(p).debug(1) }
            // NOTE: `s` could be empty string, e.g. when `str` is absolute path
            res   = append(res  , s)
            stems = append(stems, ss...)
            lastSuf = nil
            n += 1 // move forward to the next seg
            m += 1 // move forward to the next src
            if f { continue SegsSrcsLoop } else { break SegsSrcsLoop }
        } else {
            if ps, ok := seg.(*PathSeg); (s == "" && ok && ps.rune == 0) || s != "" {
                res = append(res, s)
            } else if false {
                res, stems = nil, nil
            }
            if infos { info(ctx, "%d: path=%v seg=%v (%T) res=%v stems=%v f=%v s=%s ss=%v src=%s lenSegs=%d str=%s", si, p, seg, seg, res, stems, f, s, ss, src, lenSegs, str).of(p).debug(1) }
            break SegsSrcsLoop
        }

        var prefix string
        if !isTrivial(pre) {
            if prefix, err = pre.Strval(ctx); err != nil {
                erro(ctx, "strval prefix '%v' failed: %v", pre, err).of(pre)
                return
            } else if !strings.HasPrefix(src, prefix) {
                if infos { info(ctx, "%d: seg=%v (%T) pp=%v pre=%v suf=%v res=%v stems=%v src=%s", si, seg, seg, pp, pre, suf, res, stems, src).of(p).debug(1) }
                break SegsSrcsLoop
            }
        }

        // Iterate segs for %%, e.g. bar, baz in foo/%%/bar/baz
        var stem []string
        if prefix != "" { stem = append(stem, strings.TrimPrefix(src, prefix)) }
        if !isTrivial(suf) {
            var suffix string
            if suffix, err = suf.Strval(ctx); err != nil {
                erro(ctx, "strval suffix '%v' failed: %v", suf, err).of(suf)
                break SegsSrcsLoop
            }
            if res = append(res, src); m < lenSrcs {
                for stem = append(stem, src); m < lenSrcs; m += 1 {
                    src = srcs[m]
                    res = append(res, src)
                    if strings.HasSuffix(src, suffix) {
                        stem = append(stem, strings.TrimSuffix(src, suffix))
                        stems = append(stems, strings.Join(stem, PathSep))
                        if infos { info(ctx, "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d", si, p, seg, seg, res, stems, suffix, src, lenSegs).of(p).debug(1) }
                        n += 1 // continue for next seg
                        m += 1 // move forward to the next src
                        continue SegsSrcsLoop
                    } else {
                        stem = append(stem, src)
                    }
                }
            } else {
                stem = append(stem, strings.TrimSuffix(src, suffix))
            }
            if len(stem) > 0 {
                stems = append(stems, strings.Join(stem, PathSep))
                if infos { info(ctx, "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d lenSrcs=%d m=%d", si, p, seg, seg, res, stems, suffix, src, lenSegs, lenSrcs, m).of(p).debug(1) }
            }
        } else if n < lenSegs && !isTrivial(segs[n]) {
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
                    if infos { info(ctx, "%d: path=%v seg=%v (%T) res=%v stems=%v matched=%v s=%s ss=%v src=%s lenSegs=%d", si, p, seg, seg, res, stems, matched, s, ss, src, lenSegs).of(p).debug(1) }
                    n += 1 // continue for next seg
                    m += 1 // move forward to the next src
                    continue SegsSrcsLoop
                } else {
                    res = append(res, src)
                    stem = append(stem, src)
                }
            }
            if len(stem) > 0 {
                stems = append(stems, strings.Join(stem, PathSep))
                if infos { info(ctx, "%d: path=%v seg=%v (%T) res=%v stems=%v s=%s ss=%v src=%s lenSegs=%d", si, p, seg, seg, res, stems, s, ss, src, lenSegs).of(p).debug(1) }
            }
        } else { // the tailing %%
            if /*m < lenSrcs*/true {
                var rest = srcs[m-1:]
                res = append(res, rest...)
                stem = append(stem, rest...)
                stems = append(stems, strings.Join(stem, PathSep))
            }
            if infos { info(ctx, "%d: seg=%v pre=%v suf=%v res=%v stems=%v src=%s", si, seg, pre, suf, res, stems, src).of(p).debug(1) }
            break SegsSrcsLoop
        }
        if infos && n == lenSegs {
            info(ctx, "%d: path=%v seg=%v (%T) str=%v src=%v -> f=%v s=%v ss=%v -> res=%v stems=%v stem=%v m=%d/%d 3.lengSegs=%d", si, p, seg, seg, str, src, f, s, ss, res, stems, stem, m, lenSrcs, lenSegs).of(p).debug(true, 1)
        }
    }
    if lenRes := len(res); lenRes > 0 { // full or partial matched
        //TODO: if n < lenSegs { rest = strings.Join(segs[n:], PathSep) }
        result = strings.Join(res, PathSep) //NOTE: do NOT use `filepath.Join(res...)` here
        full = n == lenSegs && m <= lenSrcs &&
            lenRes == lenSrcs && lenRes >= lenSegs &&
            result == str
        if infos { if false {
            warn(ctx, "Path.match: path=%v str=%v res=%v stems=%v -> full=%v result=%v lens=%d,%d", p, str, res, stems, full, result, lenRes, lenSrcs).of(p).debug(1)
        } else {
            warn(ctx, "Path.match: path=%v res=%v stems=%v lenRes=%d", p, res, stems, lenRes).of(p)
            warn(ctx, "Path.match: str=%v full=%v result=%v lenSrcs=%d", str, full, result, lenSrcs).of(p).debug(4)
        }}
        if correct := (!full && strings.HasPrefix(str, result)) || (full && str == result); false {
            assert(correct, "incorrect result: res=%v result=%v full=%v stems=%v str=%s", res, result, full, stems, str)
        } else if !correct {
            prompt(ctx, "%v: %v: incorrect match: full=%v; segs=%v; srcs=%v; res=%v\n", str, p, full, segs, srcs, res)
            erro(ctx, "incorrect match: path=%v, str=%s, res=%v result=%v", p, str, res, result).of(p)
            errostack(ctx, 8, "%v", ctx).debug(10)
        }
        if full && p.patterned(ctx) && len(stems) == 0 {
            prompt(ctx, "%v: %v: incorrect full match: segs=%v; srcs=%v; res=%v\n", str, p, segs, srcs, res)
            warn(ctx, "incorrect full match: path=%v, str=%s, res=%v result=%v", p, str, res, result).of(p)
            warnstack(ctx, 3, "(%T):", ctx).debug(6)
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
            erro(ctx, "strval '%v' failed: %v", t, err).of(t).debug(1)
        } else if str != "" {
            return p.match1(ctx, str)
        }
    default:
        erro(ctx, "matching unsupport value: %T %v", i, i).at(p.position).debug(8)
    }
    return
}

func (p *Path) stencil(ctx Context, stems []string) (result Value, rest []string) {
    var (
        segs []Value
        err error
    )
    if segs, err = expandmerge2(ctx, expandPlainValue, p.Elems...); err != nil {
        erro(ctx, "expand path '%v' failed: %v", p, err).of(p).debug(1)
        return
    }

    var (
        elems []Value
        changed int
    )
    for _, seg := range segs {
        var val Value
        if val, stems = seg.stencil(ctx, stems); !isTrivial(val) {
            if val != seg { changed += 1 }
            elems = append(elems, val)
        } else {
            elems = append(elems, seg)
        }
    }
    if rest = stems; changed > 0 {
        result = MakePath(p.position, elems...)
    } else {
        result = p
    }
    return
}

func (p *Path) Combine(ctx Context, val Value) {
    var (
        comp *Barecomp
        ti = len(p.Elems)-1
        tail = p.Elems[ti]
    )
    if isNil(val) || isNone(val) {
        erro(ctx, "path combines invalid value: %v", val).of(p)
        return
    } else if isNil(tail) {
        erro(ctx, "path tail is nil: %v", p).of(p)
        return
    } else if isNone(tail) {
        p.Elems[ti] = val
        return
    } else if seg, ok := tail.(*PathSeg); ok {
        switch comp = MakeBarecomp(tail.Position()); seg.rune {
        case 0, '/': break // discard
        default: comp.Combine(ctx, tail)
        }
        p.Elems[ti] = comp
    } else if comp, ok = tail.(*Barecomp); !ok || comp == nil {
        comp = MakeBarecomp(tail.Position(), tail)
        p.Elems[ti] = comp
    }

    if vp, ok := val.(*Path); ok {
        var head = vp.Elems[0]
        if seg, ok := head.(*PathSeg); ok {
            switch seg.rune {
            case 0, '/': break // discard
            default: comp.Combine(ctx, head)
            }
        } else {
            comp.Combine(ctx, head)
        }
        p.Elems = append(p.Elems, vp.Elems[1:]...)
    } else {
        comp.Combine(ctx, val)
    }
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
        erro(ctx, "unknown segment '%s' ('%v')", string(p.rune), p).at(p.position).debug(1)
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
            erro(ctx, "strval '%v' failed: %v", t, e).at(p.position).debug(1)
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
func (p *PathSeg) stencil(ctx Context, stems []string) (result Value, rest []string) {
    return p, stems
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
    _updated bool // true if this file has been updated by a program
    _updatedDeps []Value // any updated deps
}
func (p *filebase) exists() bool { return p.info != nil }

var statmutex sync.Mutex
var filecache = make(map[string]*filebase) // File.fullname() -> File

func stat(ctx Context, name, sub, dir string, infos ...os.FileInfo) (file *File) {
    var (
        base *filebase ; stub *filestub ; fullname string
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
                erro(ctx, "dir name conflicts: %s <-> %s (sub=%v)", dir, name, sub).debug(16)
                unreachable("path error")
            } else {
                return
            }
        }
    } else if filepath.IsAbs(sub) {
        if fullname = filepath.Join(sub, name); dir == "" {
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
                prompt(ctx, "%s: {%s,%s,%s}\n", fullname, name, sub, dir)
                erro(ctx, "stat dir: dir  = %s", dir)
                erro(ctx, "stat dir: sub  = %s", sub)
                erro(ctx, "stat dir: name = %s", name)
                erro(ctx, "stat dir: full = %s", fullname)
                errostack(ctx, 48, "stat: %v", ctx).debug(16)
                assert(false, "dir is not empty for abs name: %s", name)
            }
            if sub != "" {
                prompt(ctx, "%s: {%s,%s,%s}\n", fullname, name, sub, dir)
                erro(ctx, "stat sub: dir  = %s", dir)
                erro(ctx, "stat sub: sub  = %s", sub)
                erro(ctx, "stat sub: name = %s", name)
                erro(ctx, "stat sub: full = %s", fullname)
                errostack(ctx, 48, "stat: %v", ctx).debug(16)
                assert(false, "sub is not empty for abs name: %s", name)
            }
            if true {
                // skips clean name checks
            } else if s := filepath.Clean(name); fullname != s && strings.Contains(name, "//")/* skips /../ */ {
                prompt(ctx, "%s: {%s,%s,%s}\n", fullname, name, sub, dir)
                erro(ctx, "stat fullname: dir  = %s", dir)
                erro(ctx, "stat fullname: sub  = %s", sub)
                erro(ctx, "stat fullname: name  = %s", name)
                erro(ctx, "stat fullname: clean = %s", s)
                erro(ctx, "stat fullname: full  = %s", fullname)
                errostack(ctx, 48, "stat: %v", ctx).debug(64)
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
    var fileInfo os.FileInfo
    if len(infos) == 1 {
        if fileInfo = infos[0]; fileInfo == nil {
            addNotExisted = true
        }
        if enable_assertions && fileInfo != nil {
            assert(fileInfo.Name() == filepath.Base(fullname), "`%s` file name conflicted", fileInfo.Name())
        }
    } else if len(infos) > 1 {
        unreachable("too many input file infos")
    }

    var okay bool // NOTE: filepath.Join can have the same efffect as filepath.Clean
    var cleanFullname = filepath.Clean(fullname) // clean paths like /path/to/foo/../bar -> /path/to/bar
    if base, okay = filecache[cleanFullname]; okay {
        if base.info == nil {
            if fileInfo == nil { fileInfo, _ = os.Stat(fullname) }
            if fileInfo == nil && !addNotExisted { return nil } // file not exists
            base.info = fileInfo
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
        if fileInfo == nil {
            fileInfo, _ = os.Stat(fullname)
            if fileInfo == nil && !addNotExisted {
                return nil // file not exists
            }
        }

        base = &filebase{ filestub{ dir, sub, name, nil, nil }, fileInfo, false, nil }
        base.stub.other = &base.stub
        stub = &base.stub
        filecache[cleanFullname] = base
    }
    GotFile: file = &File{valbase{ctx.Position()},base,stub}

    if enable_assertions {
        if !addNotExisted { assert(file.exists(), "`%s` file not existed", fullname) }
        assert(file.name == name, "(%s %s %s).name != %s", file.name, file.sub, file.dir, name)
        assert(file.sub == sub, "(%s %s %s).sub != %s", file.name, file.sub, file.dir, sub)
        if file.dir != dir {
            var head = &base.stub
            for stub := head; stub != nil; stub = stub.other {
                info(ctx, "stat: %s %s %s", stub.dir, stub.sub, stub.name).debug(1)
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
                    erro(ctx, "{%s %s %s} file info is nil", file.dir, file.sub, file.name).debug(true,24)
                }
                if file.fullname() != s {
                    erro(ctx, "{dir=%s sub=%s name=%s} fullname incorrect: %s", file.dir, file.sub, file.name, s).debug(true,24)
                }
                if file.info.Name() != filepath.Base(s) { // NOTE: file.name might be "" here
                    erro(ctx, "{dir=%s sub=%s name=%s} name incorrect: %s", file.dir, file.sub, file.name, file.info.Name()).debug(true,24)
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
func (p *File) Strval(ctx Context) (s string, err error) { return p.name, nil }
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
func (p *File) delete(ctx Context) (files []*File, err error) {
    if positionalValueCtx { ctx = positional(ctx, p.position) }

    var fullname string
    if fullname = p.fullname(); fullname == "" {
        erro(ctx, "file `%s` has no fullname", p).of(p).debug(1)
        return
    }

    if p.info == nil {
        // ignore
    } else if err = os.Remove(fullname); err != nil {
        erro(ctx, "%v", err).at(p.position).debug(1)
    } else {
        // TODO: ctx.Globe().delete(fullname)
        files = append(files, p)
        p.info = nil
    }
    return
}
func (p *File) stamp(ctx Context) (files []*File, err error) {
    if positionalValueCtx { ctx = positional(ctx, p.position) }
    if fullname := p.fullname(); fullname == "" {
        erro(ctx, "file `%s` has no fullname", p).of(p).debug(1)
    } else if p.info, err = os.Stat(fullname); err != nil {
        if false { erro(ctx, "%v", err).debug(1) }
    } else if p.info == nil {
        if false { warn(ctx, "%v: no such file", p).debug(1) }
    } else if files = append(files, p); !ctx.configuration() {
        p.updated(true)
        ctx.dirtyMark(/*p*/)
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
func (p *File) updated(v ...bool) bool {
    if t := len(v) > 0; t {
        for _, a := range v { t = t && a }
        p._updated = t
    }
    return p._updated
}
func (p *File) updatedDeps(v ...Value) []Value {
    if len(v) > 0 { p._updatedDeps = append(p._updatedDeps, v...) }
    return p._updatedDeps
}
func (p *File) stat(ctx Context) (si *statinfo) {
    var err error
    if p.info == nil {
        if p.info, err = os.Stat(p.fullname()); err == nil {
            // good
        } else if pe, ok := err.(*fs.PathError); ok {
            if false {
                erro(ctx, "File.stat %v: %v", trimPromptString(pe.Path), pe.Err).at(p.position).debug(1)
            }
            return
        } else {
            erro(ctx, "File.stat failed: %v", err).at(p.position).debug(1)
        }
    }
    if err == nil { si = &statinfo{ file: p } }
    return
}
func (p *File) isSysFile() (res bool) {
    if p.filemap != nil && len(p.filemap.paths) == 1 {
        // system files defined by:
        //     files (
        //       (foo.xxx) ⇒ -
        //     )
        if f, ok := p.filemap.paths[0].(*Flag); ok {
            res = isNone(f.name) || isNil(f.name)
        }
    }
    return
}
func (p *File) traverse(ctx Context) (brks breakers) {
    if p.isSysFile() {
        return
    }

    if strings.HasSuffix(p.name, ".a") {
        warn(ctx, "%v: %v", p.name, ctx).debug(1)
    }
    ctx = positional(ctx, p.position)

    var (
        proj = ctx.Project()
        program = ctx.program()
        targetValue = getTargetValue(ctx)
    )
    if isTrivial(targetValue) {
        erro(ctx, "trivial $@").at(program.position).debug(1)
        return
    }

    // FIXES: checks none-File file target
    switch a := targetValue.(type) {
    case *Barecomp: // convert barecomp path into a real Path
        if p, ok := a.Elems[0].(*Path); ok {
            a.Elems = append(p.Elems[len(p.Elems)-1:], a.Elems[1:]...)
            p.Elems[len(p.Elems)-1] = a
            targetValue = p
            ctx.autoSet("@", p)
        } else {
            var s string
            var err error
            if s, err = a.Strval(ctx); err != nil {
                erro(ctx, "strval '%v' failed: %v", a, err).of(a).debug(1)
                return
            }
            if file := proj.FindFile(ctx, s); file != nil {
                targetValue = file
                ctx.autoSet("@", file)
            }
        }
    }

    if _, brks = traverseFile(ctx, p); p.info == nil {
        brks.add(ctx, breakErro).error = fileNotFoundError{ proj, p }
        prompt(ctx, "%s: file not found, project %s\n", p.fullname(), proj)
        if s1, s2 := p.String(), p.fullname(); s1 == s2 {
            erro(ctx, `%v: missing file "%s"`, proj, s1)
        } else {
            erro(ctx, `%v: missing file "%s" (at "%s")`, proj, s1, s2)
        }
        errostack(ctx, 5, "%v: %v", proj, ctx).debug(8)
    } else if tb := brks.not(breakNext, breakCase, breakDone); len(tb) > 0 {
        prompt(ctx, "%s: file not found, project %s\n", p.fullname(), proj)
        if s1, s2 := p.String(), p.fullname(); s1 == s2 {
            erro(ctx, "%v: missing file %s", proj, s1)
        } else {
            erro(ctx, "%v: missing file %s (at %s)", proj, s1, s2)
        }
        for _, brk := range tb {
            switch brk.what {
            case breakFail: erro(ctx, "broken for file %v: %v", p, brk.message).at(brk.pos)
            case breakErro: erro(ctx, "broken for file %v: %v", p, brk.error).at(brk.pos)
            default: erro(ctx, "broken for file %v (%v)", p, brk.what).at(brk.pos)
            }
        }
        errostack(ctx, 5, "%v: %v", proj, ctx).debug(6)
    } else if false && brks.has() {
        prompt(ctx, "%s: file not found, project %s\n", p.fullname(), proj)
        if s1, s2 := p.String(), p.fullname(); s1 == s2 {
            erro(ctx, "%v: missing file %s", proj, s1)
        } else {
            erro(ctx, "%v: missing file %s (at %s)", proj, s1, s2)
        }
        errostack(ctx, 5, "%v: %v", proj, ctx).debug(6)
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
        if !/*(isNil(t) || isNone(t))*/isTrivial(t) {
            var ( v string; e error )
            if v, e = t.Strval(ctx); e != nil {
                erro(ctx, "strval '%v' failed: %v", t, e).of(t).debug(1)
            } else { return p.match1(ctx, v) }
        }
    default:
        erro(ctx, "matching file '%v' with unknown input: %v", p, i).at(p.position).debug(1)
    }
    return
}
func (p *File) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
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
            erro(ctx, "strval '%v' failed: %v", t, e).of(t).debug(1)
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
        warn(ctx, "-%v <-> %T %v", p.name, i, i).at(p.position).debug(true, 16)
    }
    return
}
func (p *Flag) expandible(ctx Context, w expandwhat) bool { return p.name.expandible(ctx, w) }
func (p *Flag) expand(ctx Context, w expandwhat) (res Value, err error) {
    var name Value
    if name, err = p.name.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", p.name, err).of(p.name).debug(1)
    } else if !isNil(name) && name != p.name {
        res = &Flag{p.valbase, name}
    }
    return
}
func (p *Flag) elemStr(ctx Context, o Object, k elemkind) (s string) {
    return "-" + elementString(ctx, o, p.name, k)
}
func (p *Flag) opt(ctx Context, name string) (res string, match bool) {
    if isTrivial(p.name) {
        if false { erro(ctx, "flag name is trivial").of(p).debug(16) }
    } else if f, ok := p.name.(*Flag); ok {
        res, match = f.opt(ctx, name)
    } else if s, err := p.name.Strval(ctx); err != nil {
        erro(ctx, "strval '%v' failed: %v", p.name, err).of(p.name).debug(16)
    } else if s == name {
        res, match = name, true
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
            erro(ctx, "unknown flag (known: %s)", strings.Join(opts, ", ")).of(p)
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
            erro(ctx, "unknown flag (known: %s)", strings.Join(opts, ", ")).of(p)
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
func (p *Flag) traverse(ctx Context) (brks breakers) {
    if s, err := p.Strval(positional(ctx, p.position)); err != nil {
        erro(ctx, "strval '%v' failed: %v", p, err).of(p).debug(1)
    } else {
        _, brks = traverseString(ctx, p, s)
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
    // NOTE: escaping \" here makes the string complicated
    if false { s = strings.Replace(s, `\"`, `"`, -1) }
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
        erro(ctx, "expand compound elems failed: %v", err).at(p.position).debug(1)
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
            erro(ctx, "%v", err).of(p)
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
            erro(ctx, "%v", err).of(p)
			return
        }
        i = strings.IndexAny(s, escapedChars)
    }
    if _, err = buf.WriteString(s); err != nil {
        erro(ctx, "%v", err).of(p)
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
        erro(ctx, "expand list elems failed: %v", err).at(p.position).debug(1)
    } else if num > 0 {
        if false && len(elems) == 1 { res = elems[0] } else {
            res = &List{p.position, elements{elems}}
        }
    }
    return
}
func (p *List) traverse(ctx Context) (brks breakers) {
    if len(p.Elems) > 0 {
        for _, elem := range p.Elems {
            if brks = elem.traverse(ctx); brks.has() {
                if false {
                    for _, b := range brks {
                        warn(ctx, "%T %v; %v; %v", elem, elem, b.what, p)
                        warn(ctx, "%v; %v", ctx.stems(), ctx).debug(1)
                    }
                }
                break
            }
        }
    }
    return
}
func (p *List) updated(v ...bool) (res bool) {
    for _, elem := range p.Elems {
        if res = elem.updated(v...); res { break }
    }
    return
}
func (p *List) updatedDeps(v ...Value) (res []Value) {
    for _, elem := range p.Elems {
        res = append(res, elem.updatedDeps(v...)...)
    }
    return
}
func (p *List) stat(ctx Context) (si *statinfo) {
    if len(p.Elems) > 0 {
        for _, elem := range p.Elems {
            if ei := elem.stat(ctx); ei == nil {
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
func (p *List) delete(ctx Context) (files []*File, err error) {
    for _, elem := range p.Elems {
        var a []*File
        if a, err = elem.delete(ctx); err != nil { break }
        files = append(files, a...)
    }
    return
}
func (p *List) stamp(ctx Context) (files []*File, err error) {
    for _, elem := range p.Elems {
        var a []*File
        if a, err = elem.stamp(ctx); err != nil { break }
        files = append(files, a...)
    }
    return
}

func (p *List) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*List); ok { res = p.cmpElems(ctx, a.Elems) }
    return
}

func (p *List) patterned(ctx Context) (res bool) {
    if len(p.Elems) == 1 { res = p.Elems[0].patterned(ctx) } else {
        /* FIXME: apply to each element??
        for _, elem := range p.Elems {
          if elem.patterned() { return true }
        }*/
    }
    return
}

func (p *List) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    if len(p.Elems) == 1 { full, s, stems = p.Elems[0].match(ctx, i) } else {
        /* FIXME: apply to each element??
        for _, elem := range p.Elems {
          ...
        }*/
    }
    return
}

func (p *List) stencil(ctx Context, stems []string) (val Value, rest []string) {
    if len(p.Elems) == 1 {
        val, rest = p.Elems[0].stencil(ctx, stems)
        return
    }

    var (
        elems []Value
        changed int
    )
    rest = stems
    for _, elem := range p.Elems {
        var t Value
        if t, rest = elem.stencil(ctx, rest); t != elem { changed += 1 }
        elems = append(elems, t)
    }
    if changed > 0 {
        val = MakeList(p.position, elems...)
    } else {
        val = p
    }
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
    s = "("
    for i, elem := range p.Elems {
        var str string
        if str, err = elem.Strval(ctx); err != nil {
            erro(ctx, "strval group element '%v' failed: %v", elem, err).debug(1)
            return
        }
        if i > 0 { s += " " }
        s += str
    }
    s += ")"
    return
}
func (p *Group) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *Group) defs(ctx Context, s ...string) []*Def { return p.elements.defs(ctx, s...) }
func (p *Group) stat(ctx Context) (si *statinfo) { return }
func (p *Group) stamp(ctx Context) (files []*File, err error) { return }
func (p *Group) delete(ctx Context) (files []*File, err error) { return }
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
        erro(ctx, "expand group elems failed: %v", err).at(p.position).debug(1)
    } else if num > 0 {
        res = &Group{p.valbase, elements{elems}}
    }
    return
}
func (p *Group) traverse(ctx Context) (brks breakers) {
    warn(ctx, "traversing group: %v", p).at(p.position).debug(32)
    return
}
func (p *Group) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Group); ok { res = p.cmpElems(ctx, a.Elems) }
    return
}
func (p *Group) patterned(ctx Context, ) bool { return false }
func (p *Group) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return }
func (p *Group) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }

func parseGroupValue(ctx Context, g *Group) (result Value) {
    if len(g.Elems) == 0 { return g } else {
        var word *Bareword
        switch kind := g.Elems[0].(type) {
        case *Bareword: word = kind
        case *Group: if len(kind.Elems) > 0 {
            var ( name = kind.Elems[0]; ok bool )
            if word, ok = name.(*Bareword); !ok {
                erro(ctx, "unsupported name type: %T %v", name, name).of(name).debug(1)
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
        erro(ctx, "truthify '%v' failed: %v", p.Key, err).of(p.Key).debug(1)
    } else if t || isNil(p.Value) {
        // done
    } else if t, err = p.Value.True(ctx); err != nil {
        erro(ctx, "truthify '%v' failed: %v", p.Value, err).of(p.Key).debug(1)
    }
    return
}
func (p *Pair) Integer(ctx Context) (int64, error) { return p.Value.Integer(ctx) }
func (p *Pair) Float(ctx Context) (float64, error) { return p.Value.Float(ctx) }
func (p *Pair) refs(ctx Context, v Value) bool { return p.Key.refs(ctx, v) || p.Value.refs(ctx, v) }
func (p *Pair) defs(ctx Context, s ...string) []*Def {
    return append(p.Key.defs(ctx, s...), p.Value.defs(ctx, s...)...)
}
func (p *Pair) traverse(ctx Context) (brks breakers) {
    erro(ctx, "traversing pair '%v' is undefined", p).at(p.position)
    errostack(positional(ctx, p.position), -1, "pair is not traversible: %v", p).debug(16)
    return
}
func (p *Pair) expandible(ctx Context, w expandwhat) bool {
    if p.Key.expandible(ctx, w) { return true }
    return w&expandPairVal != 0 && p.Value.expandible(ctx, w)
}
func (p *Pair) expand(ctx Context, w expandwhat) (res Value, err error) {
    var k, v Value
    if k, err = p.Key.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", p.Key, err).of(p.Key).debug(1)
        return
    } else if isNil(k) { k = p.Key }

    // Note: donot expand the p.Value! It's used as template
    // in arguments (see copy-file for example).
    if w&expandPairVal != 0 {
        if v, err = p.Value.expand(ctx, w); err != nil {
            erro(ctx, "expand '%v' failed: %v").of(p.Value).debug(1)
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


const (
    fixUnresolvedObjectClosure = false
    fixUnresolvedObjectDelegate = true
)

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
        erro(ctx, "delegate '%v' value failed: %v", p, err).of(p).debug(1)
    } else if isNil(v) {
        erro(ctx, "delegate value is nil: %v", p).of(p).debug(1)
    } else if s, err = v.Strval(ctx); err != nil {
        erro(ctx, "strval '%v' failed: %v", v, err).of(v).debug(1)
    }
    return
}
func (p *delegate) True(ctx Context) (t bool, err error) {
    var v Value
    if v, err = p.expand(ctx, expandPlainValue); err != nil {
        erro(ctx, "expand '%v' failed: %v", p, err).at(p.position).debug(1)
    } else if isNil(v) {
        erro(ctx, "expand '%v' to nil", p).at(p.position).debug(1)
    } else if t, err = v.True(ctx); err != nil {
        erro(ctx, "truthify '%v' failed: %v", p, err).at(p.position).debug(1)
    } else if false {
        info(ctx, "%v -> %T %v -> %v", p, v, v, t).at(p.position).debug(8)
    }
    return
}
func (p *delegate) Integer(ctx Context) (i int64, err error) {
    var v Value
    if v, err = p.value(ctx); err != nil {
        erro(ctx, "delegate '%v' value failed: %v", p, err).of(p).debug(1)
    } else if isNil(v) {
        erro(ctx, "delegate value is nil: %v", p).of(p).debug(1)
    } else if i, err = v.Integer(ctx); err != nil {
        erro(ctx, "integify '%v' failed: %v", v, err).of(v).debug(1)
    }
    return
}
func (p *delegate) Float(ctx Context) (f float64, err error) {
    var v Value
    if v, err = p.value(ctx); err != nil {
        erro(ctx, "delegate '%v' value failed: %v", p, err).of(p).debug(1)
    } else if isNil(v) {
        erro(ctx, "nil delegate value: %v", p).of(p).debug(1)
    } else if f, err = v.Float(ctx); err != nil {
        erro(ctx, "floatify '%v' failed: %v", v, err).of(v).debug(1)
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
        erro(ctx, "delegation of nil (v=%v)", v).of(p).debug(1)
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
        erro(ctx, "delegation of nil (s=%v)", p, s).of(p).debug(1)
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
func (p *delegate) traverse(ctx Context) (brks breakers) {
    ctx = positional(ctx, p.position)
    if val, err := p.expand(ctx, expandPlainValue); err != nil {
        erro(ctx, "expand '%v' failed: %v", p, err).at(p.position)
        errostack(ctx, -1, "expand '%v' failed", p).debug(16)
    } else if isNil(val) {
        warn(ctx, "delegate '%v' expands to nil", p).at(p.position)
        warnstack(ctx, -1, "delegate '%v' expands to <nil>", p).debug(16)
    } else if isNone(val) {
        if false {
            warn(ctx, "delegate '%v' expands to none", p).at(p.position)
            warnstack(ctx, -1, "delegate '%v' expands to <none>", p).debug(16)
        }
    } else if brks = val.traverse(ctx); len(brks) > 0 {
        if brks = brks.not(breakCase, breakNext, breakDone); true && len(brks) > 0 {
            for _, brk := range brks { warn(ctx, "%v: %v -> %T %v", brk.what, p, val, val).debug(8) }
        }
    } else if false && strings.HasSuffix(val.String(), ".a") {
        var v, _ = ctx.autoGet("^")
        warn(ctx, "%v: %v", p, val)
        warn(ctx, "%v: %v", p, v).debug(1)
    }
    return
}
func (p *delegate) name(ctx Context, sel bool) (name string) {
    switch x := p.x.(type) {
    case Object: name = x.Name(ctx)
    case *selection: if sel {
        var err error
        if name, err = x.Strval(ctx); err != nil {
            erro(ctx, "strval '%v' failed: %v", x).debug(1)
        }
    }
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
    case Object    : name = x.Name(ctx)
    case *selection:
        var err error
        if name, err = x.Strval(ctx); err != nil {
            erro(ctx, "strval '%v' failed: %v", x).debug(1)
        }
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
        erro(ctx, "expand failed: %v", e).at(p.position).debug(1)
    } else {
        if isNil(v) { v = p }
        s = elementString(ctx, o, v, k)
    }
    return
}
func (p *delegate) value(ctx Context) (v Value, err error) {
    if v, err = p.expand(ctx, expandDelegate); err != nil {
        erro(ctx, "expand '%v' failed: %v", p, err).at(p.position).debug(1)
    } else if v == p { // d, ok := v.(*delegate); ok && d == p
        erro(ctx, "self delegation: %v", p).of(p).debug(1)
    }
    return
}
func (p *delegate) args(ctx Context, w expandwhat) (args []Value, num int, err error) {
    if w&expandArgs != 0 {
        if args, num, err = expandall1(ctx, w, p.a...); err != nil {
            erro(ctx, "expand args %v failed: %v", p.a, err).at(p.position).debug(1)
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
    if isNil(p.x) || isNone(p.x) {
        erro(ctx, "expand nil delegation: %v (w=%016b)", p, w).at(p.position).debug(64)
        return
    }
    if false && ctx.Project().Name() == "external.python._freeze_module" { if o, ok := p.x.(Object); ok && strings.HasPrefix(o.Name(ctx), "<") {
        // warn(ctx, "%T %v", p.x, p.x).debug(128); ctx.checkErrors(true)
        defer func() {
            //info(ctx, "%v : x = %T %v -> res = %T %v", p, p.x, p.x, res, res)
            //info(ctx, "%v : %v", p, ctx).debug(6)
            if s, _ := res.Strval(ctx); strings.HasSuffix(s, "config.c") {
                v, _ := res.expand(ctx, expandFullName)
                warnstack(ctx, 3, "%v: %T %v, %T %v (%v)", p, res, res, v, v, (w&expandFullName)).debug(32)
            }
        } ()
    }}

    var v Value
    if w&expandDelegate == 0 {
        var x Value
        if u, ok := p.x.(*unresolvedobject); ok && fixUnresolvedObjectDelegate {
            var name string
            if name, err = u.name.Strval(ctx); err != nil {
                erro(ctx, "expand '%v' failed: %v", u.name, err).of(u.name).debug(1)
                return
            } else if name == "" {
                erro(ctx, "empty unresolved name: %v", u.name).of(u.name).debug(1)
                return
            }
            if false {
                for _, proj := range closureProjects(ctx) {
                    if x, err = proj.resolveObject(ctx, name); err != nil {
                        erro(ctx, "resolve '%v' failed: %v", name, err).of(u.name).debug(1)
                        return
                    } else {
                        break
                    }
                }
            } else if _, sym := ctx.Scope().Find(name); sym != nil && sym.(Value) != p {
                x = sym
            }
        } else if x, err = p.x.expand(ctx, w); err != nil {
            erro(ctx, "expand '%v' failed: %v", p.x, err).of(p.x).debug(1)
            return
        } else if isNil(x) { x = p.x }

        var ( args []Value; num int )
        if args, num, err = p.args(ctx, w); err != nil {
            erro(ctx, "expand args failed: %v", err).at(p.position).debug(1)
            return
        }

        if (!isNil(x) && x != p.x) || num > 0 {
            if num == 0 { args = p.a }
            res = &delegate{p.valbase, p.l, x, args}
            return
        }
        return
    } else if ctx, res, err = p.reveal(ctx, w); err != nil {
        erro(ctx, "reveal '%v' failed: %v", p, err).of(p).debug(1)
        return
    } else if isNil(res) {
        res = MakeNone(p.position)
        return
    } else if v, err = res.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", res, err).of(p).debug(1)
        return
    } else if !isNil(v) && v != res {
        res = v
        return
    }
    return
}
func (p *delegate) reveal(ctx Context, w expandwhat) (retctx Context, res Value, err error) {
    ctx = positional(ctx, p.position)

    if false { if s := p.x.String(); s == "main→sources" {
        warn(ctx, "%T %v; %v", p.x, p.x, ctx).debug(1)
    }}

    var x Object
    if u, ok := p.x.(*unresolvedobject); ok && fixUnresolvedObjectDelegate {
        var name string
        if name, err = u.name.Strval(ctx); err != nil {
            erro(ctx, "expand '%v' failed: %v", u.name, err).of(u.name).debug(1)
            return
        } else if name == "" {
            erro(ctx, "empty unresolved name: %v", u.name).of(u.name).debug(1)
            return
        }
        if false {
            for _, proj := range closureProjects(ctx) {
                if x, err = proj.resolveObject(ctx, name); err != nil {
                    erro(ctx, "resolve '%v' failed: %v", name, err).of(u.name).debug(1)
                    return
                } else {
                    break
                }
            }
        } else if _, sym := ctx.Scope().Find(name); sym != nil && sym.(Value) != p {
            x = sym
        }
        if isNil(x) {
            if false { erro(ctx, "undefined '%v'", name).of(u.name).debug(1) }
            return
        }
    } else if t, ok := p.x.(Object); ok && t != nil {
        x = t
    } else if t, ok := p.x.(*usinglist); ok {
        x = t
    } else if t, ok := p.x.(*selection); !ok || t == nil {
        erro(ctx, "delegate unsupported object: %T %v", p.x, p.x)
        errostack(ctx, 3, "%v", ctx).debug(16)
        return
    } else if isNil(t.o) {
        erro(ctx, "%v: selection object is nil (w=%016b)", p, w)
        errostack(ctx, 3, "%v", ctx).debug(16)
        return
    } else {
        if n, ok := t.o.(*ProjectName); ok && n != nil && n.project != nil {
            if false {if s := p.x.String(); s == "main→sources" {
                warn(ctx, "%v: %T %v %v", p.x, ctx, ctx, ctx.closureScopes()).debug(1)
            }}
            // FIXME: closure with n.project not working
            ctx = closureWith(ctx, n.position, n.project.scope)
        }

        var v Value
        if v, err = t.value(ctx); err != nil {
            erro(ctx, "selected value '%v' failed: %v", p.x, err)
            errostack(ctx, 3, "%v", ctx).debug(16)
            return
        } else if isNil(v) {
            erro(ctx, "%v: selected value is nil (%T %v) (w=%016b)", p, t.o, t.o, w)
            errostack(ctx, 3, "%v", ctx).debug(16)
            return
        } else if t, ok := v.(Object); !ok {
            erro(ctx, "%v: selected value is not object (%T %v) (w=%016b)", t, v, v, w)
            errostack(ctx, 3, "%v", ctx).debug(16)
            return
        } else {
            x = t
        }
        if false { if s := p.x.String(); s == "main→sources" {
            warn(ctx, "%v: %T %v", p.x, t.o, t.o)
            warn(ctx, "%v: %T %v", p.x, x, x)
            warn(ctx, "%v: %T %v %v", p.x, ctx, ctx, ctx.closureScopes()).debug(1)
        }}
    }

    var args []Value
    if args, _, err = p.args(ctx, w); err != nil {
        erro(ctx, "compute args failed: %v", err).debug(1)
        return
    }

    switch t := x.(type) {
    case Caller:
        if res = t.Call(ctx, args...); isNil(res) {
            if d, ok := x.(*Def); ok && !isNil(d.value) {
                erro(ctx, "calling def '%v' (%v) returns <nil> (def.value=%v %T, %T)", d.name, d.origin, d.value, d.value, ctx).debug(16)
            }
        }
        if false {if s := p.x.String(); s == "main→sources" {
            xv, _ := res.expand(ctx, w)
            warn(ctx, "%v: %v", p.x, t)
            warn(ctx, "%v: %v", p.x, res)
            warn(ctx, "%v: %v", p.x, xv).debug(1)
        }}
    case Executer:
        if vals, brks := t.Execute(ctx, args...); len(brks) > 0 {
            for _, brk := range brks {
                var s string
                if brk.message != "" { s = brk.message }
                if brk.error != nil { s += fmt.Sprintf(" (error: %s)", brk.error) }
                erro(ctx, "broken '%v': (%s) %s", x, brk.what, s).at(brk.pos).debug(1)
            }
        } else if len(vals) > 0 {
            res = MakeList(ctx.Position(), vals...)
        }
    default:
        var pos = t.Position()
        if !pos.IsValid() { pos = p.position }
        erro(ctx, "%s: unknown delegation: %T %v -> %T %v", x.Name(ctx), p.x, p.x, x, x).at(pos).debug(32)
    }

    retctx = ctx
    return
}
func (p *delegate) stat(ctx Context) (si *statinfo) {
    erro(ctx, "cant stat delegate %v, must expand it first", p).at(p.position).debug(16)
    return
}
func (p *delegate) stamp(ctx Context) (file []*File, err error) {
    erro(ctx, "cant stamp delegate %v, must expand it first", p).at(p.position).debug(16)
    return
}
func (p *delegate) delete(ctx Context) (file []*File, err error) {
    erro(ctx, "cant delete delegate %v, must expand it first", p).at(p.position).debug(16)
    return
}
func (p *delegate) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*delegate); ok {
        // NOTE: don't compare the expanded value!!
        if p == a {
            return cmpEqual
        } else if res = p.x.cmp(ctx, a.x); res == cmpEqual && len(p.a) == len(a.a) {
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
        erro(ctx, "%v", err.Error()).at(p.Position()).debug(1)
        return
    }

    var val Value
    if val, err = p.expand(ctx, expandDelegate|expandClosure); err != nil {
        erro(ctx, "expand '%v' failed: %v", p, err).of(p).debug(1)
    } else if isNil(val) {
        if false { warn(ctx, "expand '%v' to nil", p).of(p).debug(1) }
    } else if s, err = val.Strval(ctx); err != nil {
        erro(ctx, "strval '%v' failed: %v", p, err).of(p).debug(1)
    }
    return
}
func (p *closure) True(ctx Context) (t bool, err error) {
    var v Value
    if v, err = p.expand(ctx, expandPlainValue); err != nil {
        erro(ctx, "expand '%v' failed: %v", p, err).at(p.position).debug(1)
    } else if isNil(v) {
        // does nothing
    } else if t, err = v.True(ctx); err != nil {
        erro(ctx, "truthify '%v' failed: %v", p, err).at(p.position).debug(1)
    } else if false {
        info(ctx, "%v -> %T %v -> %v", p, v, v, t).at(p.position).debug(8)
    }
    return
}
func (p *closure) elemStr(ctx Context, o Object, k elemkind) (s string) {
    if ctx == nil || k&elemExpand == 0 {
        if s = p.string(ctx, o, k); !(p.l.IsClosure() || p.l.IsDelegate()) { s = "&" + s }
    } else if v, e := p.expand(ctx, expandDelegate/*|expandClosure*/); e != nil {
        erro(ctx, "expand failed: %v", e).at(p.position).debug(1)
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
        erro(ctx, "expand nil closure: %v (%d)", p, w).at(p.position).debug(1)
        return
    }

    var val Value
    if w&expandClosure == 0 {
        // Can't expand Def here as closure still need it
        if u, ok := p.x.(*unresolvedobject); ok && fixUnresolvedObjectClosure {
            var name string
            if name, err = u.name.Strval(ctx); err != nil {
                erro(ctx, "expand '%v' failed: %v", u.name, err).of(u.name).debug(1)
                return
            } else if name == "" {
                erro(ctx, "empty unresolved name: %v", u.name).of(u.name).debug(1)
                return
            }
            var x Object
            for _, proj := range closureProjects(ctx) {
                if x, err = proj.resolveObject(ctx, name); err != nil {
                    erro(ctx, "resolve '%v' failed: %v", name, err).of(u.name).debug(1)
                    return
                } else if !isNil(x) {
                    break
                }
            }
            if isNil(x) {
                val = p.x
            } else if val, err = x.expand(ctx, w&^expandDef); err != nil {
                erro(ctx, "expand '%v' failed: %v", x, err).of(x).debug(1)
                return
            }
        } else if val, err = p.x.expand(ctx, w&^expandDef); err != nil {
            erro(ctx, "expand '%v' failed: %v", p.x, err).of(p.x).debug(1)
            return
        } else if isNil(val) { val = p.x }

        var ( args []Value; num int )
        if args, num, err = p.args(ctx, w); err != nil {
            erro(ctx, "expand args failed: %v", err).of(p).debug(1)
            return
        } else if (!isNil(val) && val != p.x) || num > 0 {
            if num == 0 { args = p.a }
            res = &closure{delegate{p.valbase, p.l, val, args}}
        }
    } else if res, err = p.disclose(ctx, w); err != nil {
        erro(ctx, "disclose '%v' failed: %v", p, err).of(p).debug(1)
    } else if isNil(res) {
        erro(ctx, "disclose '%v' to nil (%s '%v')", p, typeof(p.x), p.x).of(p).debug(16)
    } else if w&^expandClosure == 0 {
        // done, no more expand
    } else if val, err = res.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", res, err).of(p).debug(1)
    } else if !isNil(val) && val != res {
        if /*p.String() == "&(objects)"*/false {
            var v, _ = res.expand(ctx.inner(), w)
            info(ctx, "%v -> %T %v -> %T %v -> %T %v", p, res, res, val, val, v, v).at(p.position)
            info(ctx, "%v %v", p, ctx.inner()).at(p.position)
            info(ctx, "%v %v", p, ctx).at(p.position).debug(16)
        }
        res = val
    }
    return
}
func (p *closure) disclose(ctx Context, w expandwhat) (res Value, err error) {
    var x Object
    if false { defer func() { if name, proj := p.x.(Object).Name(ctx), ctx.Project(); name == "@" {
        var val, _ = res.expand(ctx, w)
        var obj = closureResolveObject(ctx, p.position, name)
        warn(ctx, "%v: p = %v, %016b", proj, p, w).at(p.Position())
        warn(ctx, "%v: p.x = %T %v", proj, p.x, p.x).at(p.x.Position())
        warn(ctx, "%v: res = %T %v", proj, res, res).at(res.Position())
        warn(ctx, "%v: val = %T %v", proj, val, val).at(res.Position())
        warn(ctx, "%v: obj = %T %v", proj, obj, obj)
        if d, ok := res.(*delegate); ok { warn(ctx, "%v: d.x = %T %v", proj, d.x, d.x).at(d.position)/*.debug(1)*/ }
        warn(ctx, "%v: %s", proj, ctx).debug(32)
    } } () }

    if u, ok := p.x.(*unresolvedobject); ok && fixUnresolvedObjectClosure {
        var name string
        if name, err = u.name.Strval(ctx); err != nil {
            erro(ctx, "expand '%v' failed: %v", u.name, err).of(u.name).debug(1)
            return
        } else if name == "" {
            erro(ctx, "empty unresolved name: %v", u.name).of(u.name).debug(1)
            return
        }
        for _, proj := range closureProjects(ctx) {
            if x, err = proj.resolveObject(ctx, name); err != nil {
                erro(ctx, "resolve '%v' failed: %v", name, err).of(u.name).debug(1)
                return
            } else if !isNil(x) {
                break
            }
        }
    }

    if !isNil(x) {
        // fallthrough
    } else if t, ok := p.x.(Object); ok && t != nil {
        x = t
    } else if t, ok := p.x.(*selection); ok && t != nil && !isNil(t.o) {
        if n, ok := t.o.(*ProjectName); ok && n != nil && n.project != nil {
            ctx = closureWith(ctx, n.position, n.project.scope)
        }
        if v, e := t.value(ctx); e != nil {
            erro(ctx, "select value '%v' failed: %v", p.x, e).of(p.x).debug(1)
            err = e; return
        } else if t, ok := v.(Object); ok {
            x = t
        }
    }

    var name string
    if isNil(x) {
        erro(ctx, "closure undefined object: %T %v", p.x, p.x).of(p.x).debug(16)
        return
    } else if name = x.Name(ctx); name == "" {
        erro(ctx, "empty closure name: %T %v -> %T %v", p.x, p.x, x, x).of(p.x).debug(1)
        return
    }

ClosureTok:
    switch p.l {
    case token.LBRACE, token.STRING, token.COMPOUND:
        if entry := closureResolveEntry(ctx, p.position, name); entry == nil {
            // continue
        } else if p.l == token.LBRACE {
            x = entry
            break ClosureTok
        } else { // token.STRING, token.COMPOUND
            res = entry // &'xxx' and &"xxx" are fetching entry in the closure context
            return
        }
    default: //case token.LPAREN, token.ILLEGAL:
        if obj := closureResolveObject(ctx, p.position, name); obj == nil {
            // continue
        } else {
            x = obj
            break ClosureTok
        }
    }

    var ( args []Value; num int )
    if args, num, err = p.args(ctx, w); err != nil {
        erro(ctx, "get args failed: %v", err).at(p.position).debug(1)
    } else if isNil(x) {
        erro(ctx, "closure object is nil: %v", p.x).at(p.position).debug(1)
    } else if p.x != x || num > 0 {
        if t1, t2 := reflect.TypeOf(p.x), reflect.TypeOf(x); t1 != t2 {
            if unres := reflect.TypeOf((*unresolvedobject)(nil)); t1 != unres {
                if false {
                    var o1 = closureResolveObject(ctx, p.position, name)
                    var o2 = closureResolveObject(ctx.inner(), p.position, name)
                    var o3 = closureResolveObject(ctx.inner().inner(), p.position, name)
                    erro(ctx, "%T %v", o1, o1).at(p.position)
                    erro(ctx, "%T %v", o2, o2).at(p.position)
                    erro(ctx, "%T %v", o3, o3).at(p.position)
                }
                erro(ctx, "closure object type differs: %v (!= %v)", t2, t1).at(p.position)
                erro(ctx, "%v '%s' is found here", t1, name).at(p.x.Position())
                erro(ctx, "%v", ctx).at(ctx.Position()).debug(16)
                return
            }
        }
        res = &delegate{p.valbase, p.l, x, args}
    } else {
        res = &p.delegate
    }
    return
}
func (p *closure) traverse(ctx Context) (brks breakers) {
    if val, err := p.expand(positional(ctx, p.position), expandClosure); err != nil {
        erro(ctx, "expand '%v' failed: %v", p, err).at(p.position).debug(1)
    } else if isNil(val) {
        warn(ctx, "closure '%v' expands to nil", p).at(p.position).debug(1)
    } else if isNone(val) {
        warn(ctx, "closure '%v' expands to none", p).at(p.position).debug(1)
    } else if brks = val.traverse(ctx); false && len(brks) > 0 {
        for _, brk := range brks { warn(ctx, "%v; %v -> %T %v", brk.what, p, val, val).debug(8) }
    }
    return
}
func (p *closure) stat(ctx Context) (si *statinfo) {
    erro(ctx, "cant stat closure %v, must expand it first", p).at(p.position).debug(16)
    return
}
func (p *closure) stamp(ctx Context) (file []*File, err error) {
    erro(ctx, "cant stamp closure %v, must expand it first", p).at(p.position).debug(16)
    return
}
func (p *closure) delete(ctx Context) (file []*File, err error) {
    erro(ctx, "cant stamp closure %v, must expand it first", p).at(p.position).debug(16)
    return
}
func (p *closure) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*closure); ok {
        // NOTE: don't compare the expanded value!!
        if p == a {
            return cmpEqual
        } else if res = p.x.cmp(ctx, a.x); res == cmpEqual && len(p.a) == len(a.a) {
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
        ctx = closureWith(ctx, n.position, n.project.scope)
    }

    var v Value
    if v, err = p.value(ctx); err != nil {
        erro(ctx, "%v", err).at(p.position)
    } else if v != nil {
        if s, err = v.Strval(ctx); err != nil {
            erro(ctx, "%v", err).at(p.position).debug(1)
        }
    } else if false {
        erro(ctx, "selection.strval: `%s` is nil", p.String()).at(p.position)
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
/*
func (p *selection) objectName(ctx Context) (s string) {
    switch t := p.o.(type) {
    case Object: s = t.Name(ctx)
    }
    return
}
func (p *selection) propName(ctx Context) (s string) {
    switch t := p.s.(type) {
    case Object: s = t.Name(ctx)
    case *Bareword: s = t.string
    case *String: s = t.string
    }
    return
}
*/
func (p *selection) object(ctx Context) (o Object, err error) {
    if s, ok := p.o.(*selection); ok {
        var v Value
        if v, err = s.value(ctx); err != nil {
            // sth's wrong!
        } else if o, _ = v.(Object); o == nil {
            erro(ctx, "selection.object: `%s` is nil", s.String()).at(p.position)
        }
    } else if o, ok = p.o.(Object); !ok {
        erro(ctx, "selection.object: '%v' is not object (but %s)", p.o, typeof(p.o)).at(p.position)
    }
    return
}
func (p *selection) value(ctx Context) (v Value, err error) {
    var o Object
    if isNil(p.s) {
        erro(ctx, "selection prop is nil: %s", p.String()).at(p.position).debug(1)
    } else if o, err = p.object(ctx); err != nil {
        erro(ctx, "get selection object failed: %v", err).at(p.position).debug(1)
    } else if s := ""; o != nil {
        /*if n, ok := o.(*ProjectName); ok && n != nil && n.project != nil {
            defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
        }*/
        if s, err = p.s.Strval(ctx); err == nil {
            if pn, ok := o.(*ProjectName); ok && (p.t == token.SELECT_PROG1 || p.t == token.SELECT_PROG2) {
                var entries *ResolveEntries
                if entries, err = pn.project.resolveEntries(ctx, s, false, false); err != nil {
                    return
                } else if entries == nil {
                    erro(ctx, "selection.value: no entry `%s` (%+v)", s, p.String()).at(p.position)
                } else {
                    v = entries
                }
            } else if v, err = o.Get(ctx, s); err != nil {
                erro(ctx, "%v", err).at(p.position)
            }
        }
    } else /*if o == nil*/ {
        erro(ctx, "selection.value: nil object `%s`", p.String()).at(p.position)
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
            erro(ctx, "selection '%v' failed: %v", p, err).at(p.position).debug(1)
        }
    } else if isNil(p.o) {
        return // nil object
    } else if isNil(p.s) {
        return // nil prop
    }

    var o, s Value
    if o, err = p.o.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", p.o, err).of(p.o).debug(1)
        return
    } else if isNil(o) { o = p.o }
    if s, err = p.s.expand(ctx, w); err != nil {
        erro(ctx, "expand '%v' failed: %v", p.s, err).of(p.s).debug(1)
        return
    } else if isNil(s) { s = p.s }

    if o != p.o || s != p.s { res = &selection{p.valbase,p.t,o,s}}
    return
}
func (p *selection) traverse(ctx Context) (brks breakers) {
    ctx = positional(ctx, p.position)
    if val, err := p.value(ctx); err != nil {
        erro(ctx, "select value '%v' failed: %v", p, err).debug(1)
    } else if isNil(val) {
        warn(ctx, "selected value '%v' is nil", p).debug(1)
    } else if isNone(val) {
        warn(ctx, "selected value '%v' is none", p).debug(1)
    } else {
        brks = val.traverse(ctx)
    }
    return
}
func (p *selection) stat(ctx Context) (si *statinfo) {
    erro(ctx, "cant stat selection %v, must expand it first", p).at(p.position).debug(1)
    return
}
func (p *selection) stamp(ctx Context) (file []*File, err error) {
    erro(ctx, "cant stamp selection %v, must expand it first", p).at(p.position).debug(1)
    return
}
func (p *selection) delete(ctx Context) (file []*File, err error) {
    erro(ctx, "cant stamp selection %v, must expand it first", p).at(p.position).debug(1)
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
    var (
        prefix string
        err error
    )
    if !isTrivial(p.Prefix) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if prefix, err = p.Prefix.Strval(ctx); err != nil {
            erro(ctx, "prefix strval '%v' failed: %v", p.Prefix, err).of(p.Prefix)
            return
        } else if strings.HasPrefix(rep, prefix) {
            result = prefix
        } else {
            return
        }
    }
    if false && p.String() == "%%" {
        defer func() {
            warn(ctx, "%v(%v,%v): %v, %v, %v, %v", p, p.Prefix, p.Suffix, rep, full, result, stems).debug(6)
        } ()
    }

    var a, b = len(prefix), len(rep)
    if isTrivial(p.Suffix) {
        if a < b { stems, result, full = append(stems, rep[a:]), rep, true }
    } else if pp, ok := p.Suffix.(*PercPattern); a < b && ok {
        // FIXME: separate paths???  *) use % to match single path sep; *) use %% to match full path
        // fooxxbaryybaz -> foo%bar%baz => (foo xx bar yy baz) [xx yy]
        // fooxxx -> foo%% => (foo xxx) [xxx]
        // fooxxxbar -> foo%%bar => (foo xxx bar) [xxx]
        for ok {
            if isTrivial(pp.Prefix) {
                // does nothing
            } else if s, e := pp.Prefix.Strval(ctx); e != nil {
                erro(ctx, "strval '%v' failed: %v", pp.Prefix, e).of(pp.Prefix)
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
            if isTrivial(pp.Suffix) {
                var s = rep[a:] // let %% matches everything else
                full, stems = true, append(stems, s)
                result += s
                break
            } else if pp2, ok = pp.Suffix.(*PercPattern); ok {
                pp = pp2
                continue
            } else if s, e := pp.Suffix.Strval(ctx); e != nil {
                erro(ctx, "strval '%v' failed: %v", pp.Suffix, e).of(pp.Prefix)
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
            warn(ctx, "mixing % pattern might have performance impact: %v", p).of(p.Suffix).debug(1)
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
            erro(ctx, "strval '%v' failed: %v", p.Suffix, err).of(p.Suffix)
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
            erro(ctx, "strval '%v' failed: %v", t, err).of(t)
        } else { return p.match1(ctx, rep) }
    default:
        unreachable(fmt.Sprintf("perc.match: %T %v", i, i))
    }
    return
}
func (p *PercPattern) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var (
        s string
        err error
    )
    if !isTrivial(p.Prefix) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if s, err = p.Prefix.Strval(ctx); err != nil {
            erro(ctx, "strval prefix '%v' failed: %v", p.Prefix, err).of(p.Suffix).debug(1)
            return
        }
    }
    if false && p.String() == "%%" {
        defer func() {
            warn(ctx, "%v(%v,%v): %v, %T %v, %v", p, p.Prefix, p.Suffix, stems, val, val, rest).debug(6)
        } ()
    }

    if len(stems) > 0 {
        s += stems[0]
        rest = stems[1:]
    } else if s == "" {
        // return
    }

    var suffix Value
    if isTrivial(p.Suffix) {
        // done
    } else if p.Suffix.patterned(ctx) {
        // FIXME: patterns like '%xxx%...' use multiple stems.
        if suf, res := p.Suffix.stencil(ctx, rest); suf != p.Suffix {
            // NOTE: patterns like '%%...' uses only one stem,
            suffix, rest = suf, res
        }
    } else {
        suffix = p.Suffix
    }

    if isTrivial(suffix) {
        if s != "" {
            val = MakeBareword(p.position, s)
        } else {
            val = p
        }
    } else if s != "" {
        val = MakeBareword(p.position, s)
        val = MakeBarecomp(p.position, val, suffix)
    } else {
        val = suffix
    }
    return
}
func (p *PercPattern) traverse(ctx Context) (brks breakers) { return traversePattern(ctx, p) }
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
        erro(ctx, "expand glob components failed: %v (w=%016b)", err, w).of(p).debug(1)
    } else if num > 0 {
        res = &GlobPattern{p.valbase, components}
    }
    return
}
func (p *GlobPattern) patterned(ctx Context) bool { return true }
func (p *GlobPattern) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    var ( s string; e error )
    switch t := i.(type) {
    case string:    s = t
    case *File:     s = t.name
    case *filestub: s = t.name
    case Value:
        if s, e = t.Strval(ctx); e != nil {
            erro(ctx, "strval '%v' failed: %v", t, e).of(t)
            return
        }
    default: unreachable("glob.match: %T %v", i, i)
    }
    if matched, pre, t := globMatchFile(ctx, p, s, true); matched {
        result, stems, full = s, t, true // FIXME: calculate stems from matching
        if pre != "" { /*full = false*/ }
    }
    return
}
func (p *GlobPattern) stencil(ctx Context, stems []string) (val Value, rest []string) {
    unreachable(fmt.Sprintf("Unimplemented GlobPattern stencil %v (stems=%v)", p, stems))
    return
}
func (p *GlobPattern) traverse(ctx Context) (brks breakers) { return traversePattern(ctx, p) }
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
    unreachable("regexp.match: %T %v", i, i)
    return
}
func (p *RegexpPattern) stencil(ctx Context, stems []string) (val Value, rest []string) {
    unreachable("regexp.stencil: %v", stems) // TODO: regexp stencil
    return
}
func (p *RegexpPattern) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*RegexpPattern); ok {
        if a != nil { /* FIXME: ... */ }
    }
    return
}
func (p *RegexpPattern) traverse(ctx Context) (brks breakers) { return traversePattern(ctx, p) }

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
            erro(ctx, "expand '%v' failed: %v", v, err).of(v).debug(1)
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
            erro(ctx, "expand '%v' failed: %v", v, err).of(v).debug(1)
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
            //erro(ctx, "'%v' is not value type (%T)", a, a).debug(6)
        }
    }
    return
}

// Merge lists recursively into a single list. Previously called Join.
func merge(args... Value) (elems []Value) {
    for _, arg := range args {
        if l, o := arg.(*List); o && l != nil {
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
    return mergeresult(res, err)
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

func expandall1(ctx Context, w expandwhat, values ...Value) (elems []Value, num int, err error) {
    for _, elem := range values {
        var val Value
        if isNil(elem) {
            // TODO: report nil expand ??
        } else if val, err = elem.expand(ctx, w); err != nil {
            erro(ctx, "expand '%v' failed: %v", elem, err).of(elem).debug(1)
            break
        } else if isNil(val) || val == elem {
            elems = append(elems, elem)
        } else if true && val.expandible(ctx, w) {
            if f, ok := val.(*File); ok {
                prompt(ctx, "%v: %s\n", f.position, f.name)
                warnstack(ctx, 6, "incomplete expand: %T %v -> %T %v -> {%v,%v,%v} (w=%016b)",
                    elem, elem, val, val, f.name, f.sub, f.dir, w).of(elem).debug(16)
            } else {
                var t, _ = val.expand(ctx, w)
                warnstack(ctx, 6, "incomplete expand: %T %v -> %T %v (equal=%v) -> %v (w=%016b)",
                    elem, elem, val, val, (elem==val), t, w).of(elem).debug(16)
            }
        } else {
            elems = append(elems, val)
            num += 1
        }
    }
    return
}

func expandall2(ctx Context, w expandwhat, values ...Value) (res []Value, num int, err error) {
    if res, num, err = expandall1(ctx, w, values...); err == nil && w != 0 {
        /*for i, v := range res {
            if v.expandible(ctx, w) {
                t, _ := v.expand(ctx, w)
                warn(ctx, "expand incomplete: %T %v -> %T %v (equal=%v) -> %v (w=%016b)", values[i], values[i], v, v, (values[i]==v), t, w).of(values[i]).debug(16)
            }
        }*/
    }
    return
}

func expandmerge1(ctx Context, w expandwhat, values ...Value) (res []Value, err error) {
    var num int
    if res, num, err = expandall1(ctx, w, values...); err == nil && num > 0 {
         res, err = mergeresult(res, err)
    }
    return
}

func expandmerge2(ctx Context, w expandwhat, values ...Value) ([]Value, error) {
    return mergeresult2(expandall2(ctx, w, values...))
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
    if l, ok := v.(*List); l != nil && ok && l.Len() == 1 {
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
    case "\"": s = "\""
    case "'":  s = "'"
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
func MakeFloat(pos Position, f float64) *Float  { return &Float{valbase{pos},f} }
func MakeDate(pos Position, s time.Time) *Date  { return &Date{DateTime{valbase{pos},s}} }
func MakeTime(pos Position, t time.Time) *Time  { return &Time{DateTime{valbase{pos},t}} }
func MakeRaw(pos Position, s string) *Raw       { return &Raw{valbase{pos},s} }
func MakeString(pos Position, s string) *String { return &String{valbase{pos},s} }
func MakeFlag(pos Position, s string) *Flag     { return &Flag{valbase{pos}, &Bareword{valbase{pos},s}} }
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
    if isNil(obj) { panic(failure{pos,"making closure on <nil> object"}) }
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
