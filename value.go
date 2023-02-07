//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
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
    enable_assertions  = true
    enable_grep_bench  = true
    positionalValueCtx = true
    traveseDetectLoops = true // turn on/off traverse loop detection
    traveseLoopBreakState = traveUnkn // eg traveNext or traveDone
    traverseArgumentedExpand = true
)

type (
    cmpres    int
    existence int
    facet    uint
    HashBytes [sha256.Size]byte
)
const (
    cmpUnknown cmpres = 0
    cmpLPrefix     = -2 // L is prefix of R, should also be 'smaller'
    cmpSmaller     = -1 // meaningless so far
    cmpGreater     = 1  // meaningless so far
    cmpRPrefix     = 3  // R is prefix of L, should also be 'greater'
    cmpEqual       = 2
)
const (
    existenceMatterless existence = 1<<iota
    existenceConfirmed
    existenceNegated
)
const (
    expandZero facet = 0
    expandNone facet = 1<<(iota-1) // NOTE: iota starts at 2 here
    expandClosure   // &(...)            -> $(...)
    expandDelegate  // $(...)            -> ......
    expandPlaceholders // $_
    expandDigits       // $1 $2 $3 ...
    expandUnPlaceholders
    expandUnDigits
    expandAuto      // TODO: $0 $1 $3 $@ $<    -> ...       TODO: auto -> placeholder
    expandArgList   // expanding delegate arg list
    expandArgedArgs // foo($(args))      -> foo(...)
    expandDef       // foo=...           -> ...
    expandSelection // foo->bar          -> ...
    expandFullName  // foobar.c          -> /path/to/foobar.c
    expandPatterned // %.proto           -> example.proto (if ctx.stems() == [example])
    expandPathStr   // "/path/to"/foo    -> /path/to/foo
    expandPairVal   // foo=$(bar)        -> foo=...
    expandPlain     // unexpanded        -> ...
    expandUnresName // TODO: ...
    expandClose
    expandDebug

    ident = expandAuto | expandClosure | expandDelegate | expandSelection |
        expandDef | expandPathStr | expandArgedArgs |
        //expandDigits | expandPlaceholders |
        0

    plain = ident | expandPlain
)

func (v cmpres) String() (s string) {
    switch v {
    case cmpUnknown: s = "unknown"
    case cmpLPrefix: s = "lprefix"
    case cmpSmaller: s = "smaller"
    case cmpGreater: s = "greater"
    case cmpRPrefix: s = "rprefix"
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

var (
    traveTargetNotDefinedFile = fmt.Errorf("target not defined as file")
)

const ( // larger value higher priority
    traveUnkn travekind = iota
    traveObj  // found object
    traveRule // found object
    traveFile // exists file
    traveNext // (cond ...) and (case ...)
    traveCase // (case ...) selected
    traveDone // (cond ...) and (case ...)
    traveFail // (assert ...) and errors
)

type travekind int

func (k travekind) String() (s string) {
    switch k {
    case traveUnkn: s = "trave.unkn"
    case traveObj : s = "trave.obj"
    case traveRule: s = "trave.rule"
    case traveFile: s = "trave.file"
    case traveNext: s = "trave.next"
    case traveDone: s = "trave.done"
    case traveCase: s = "trave.case"
    case traveFail: s = "trave.fail"
    }
    return
}

type travestate struct {
    pos Position
    prog *Program
    what travekind
    target, depend, dependPat Value
    error error
}

func (p *travestate) String() (s string) { //_error
    switch p.what {
    case traveUnkn: s = "unknown"
    case traveDone: s = "done" // ineligible (cond) is ignored
    case traveNext: s = "next"
    case traveCase: s = "case"
    case traveFile: s = "file"
    case traveFail: s = "fail"
    case traveRule: s = "rule"
    case traveObj : s = "obj"
    }
    if !isNil(p.target)    { s = fmt.Sprintf("%s@%s", s, p.target) } //⇒
    if !isNil(p.dependPat) { s = fmt.Sprintf("%s:%s", s, p.dependPat) }
    if !isNil(p.depend)    { s = fmt.Sprintf("%s>%s", s, p.depend) } //⇒
    if false && p.pos.IsValid() { s = fmt.Sprintf("%s: %s", p.pos, s) }
    if p.error != nil {
        if pe, ok := p.error.(*os.PathError); ok {
            s += ":" + pe.Err.Error()
        } else if e := p.error.Error(); e != "" {
            s += ":" + e
        }
    }
    return
}

type travestates []*travestate

func (p travestates) String() (s string) {
    const x = 5
    s = "["
    for i, t := range p {
        if i > 0 { s += " " }
        s += t.String()
        if i == x && len(p) > x {
            s += fmt.Sprintf(" …%d…", len(p)-x)
            break
        }
    }
    s += "]"
    return
}

func (traves *travestates) has(what ...travekind) (res bool) {
    if len(what) == 0 { res = len(*traves) > 0 } else {
    ForTravestates:
        for _, s := range *traves {
            for _, w := range what {
                if s.what == w {
                    res = true
                    break ForTravestates
                }
            }
        }
    }
    return
}

func (traves *travestates) append(s *travestate) *travestates {
    *traves = append(*traves, s)
    return traves
}

func (traves *travestates) remove(ss ...*travestate) *travestates {
    var res travestates
ForTravestates:
    for _, s := range ss {
        for _, t := range *traves {
            if s == t { continue ForTravestates }
        }
        res = append(res, s)
    }
    *traves = res
    return traves
}

func (traves *travestates) not(what ...travekind) (res travestates) {
ForTravestates:
    for _, s := range *traves {
        for _, w := range what {
            if s.what == w { continue ForTravestates }
        }
        res = append(res, s)
    }
    return
}

func (traves *travestates) of(what ...travekind) (res travestates) {
ForTravestates:
    for _, s := range *traves {
        for _, w := range what {
            if s.what == w {
                res = append(res, s)
                continue ForTravestates
            }
        }
    }
    return
}

func (traves *travestates) unique(ctx Context, what ...travekind) (res travestates) {
ForTravestates:
    for _, s := range *traves {
        for _, w := range what {
            if s.what == w {
                for _, s2 := range res {
                    // FIXME: seems not working
                    if s.what == s2.what && (s == s2 || (true && eq(ctx, s.target, s2.target) &&
                        ((s.depend != nil && s2.depend != nil && eq(ctx, s.depend, s2.depend))))) {
                        continue ForTravestates
                    }
                } 
                res = append(res, s)
                continue ForTravestates
            }
        }
    }
    return
}

func (traves *travestates) add(ctx Context, what travekind, target Value) *travestate {
    if isTrivial(target) { target = autoGet(ctx, "@") }
    for _, s := range *traves {
        if s.what == what && s.target == target {
            return s
        }
    }

    var pos = ctx.Position()
    var s = &travestate{ pos:pos, what:what, target:target }
    if *traves = append(*traves, s); false {
    }
    return s
}

func (traves *travestates) addf(ctx Context, what travekind, s string, a... interface{}) *travestate {
    t := traves.add(ctx, what, nil)
    t.error = fmt.Errorf(s, a...)
    return t
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
func (pc *prioritizedContext) autoGet(name string) (res *def) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if res = c.autoGet(name); res != nil { break }
    }
    return
}
func (pc *prioritizedContext) autoSet(name string, val Value) (def *def, res Value) {
    for _, c := range append([]Context{ pc.Context }, pc.more...) {
        if def, res = c.autoSet(name, val); def != nil { break }
    }
    return
}

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
    if scopes = cc.Context.closureScopes(); true {
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
    }
    return
}

func closureProjects(ctx Context) (projects []*Project) {
ForScopes:
    for _, scope := range ctx.closureScopes() {
        if proj := scope.project; proj != nil {
            for _, project := range projects {
                if project == proj || project.hasBase(proj) {
                    continue ForScopes
                }
            }
            projects = append(projects, proj)
        }
    }
    return
}

func closureGet(ctx Context, name string) (res *def) {
    for _, scope := range ctx.closureScopes() {
        if scope.project == nil {
            if _, obj := scope.Find(name); obj == nil {
                continue
            } else if res, _ = obj.(*def); res != nil {
                return
            }
        } else {
            var pos = ctx.Position()
            if !pos.IsValid() { pos = scope.position }
            if !pos.IsValid() { pos = scope.project.position }
            if scope != scope.project.scope {
                if _, obj := scope.Find(name); obj != nil {
                    if res, _ = obj.(*def); res != nil {
                        return
                    }
                }
            }
            if obj := scope.project.resolveObject(ctx, name); obj == nil {
                if res = ctx.autoGet(name); res != nil { return }
            } else if res, _ = obj.(*def); res != nil {
                return
            }
        }
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

func closureResolveObject(ctx Context, name string) (obj Object) {
    var (
        infos = false && strings.HasPrefix(name, "@")
        scope *Scope
    )
    if infos { defer func() {
        var val Value
        if obj != nil { val = obj.expand(ctx, plain) }
        warn(at(ctx,scope.position), "%v: name = %s", scope.project, name)
        warn(at(ctx,scope.position), "%v: %v", scope.project, scope)
        warn(at(ctx,scope.position), "%v: %v", scope.project, ctx)
        warn(ctx, "%s: %T, %v", scope.project, obj, obj)
        warn(ctx, "%s: %T, %v", scope.project, val, val).debug(24)
    } () }
    for _, scope = range ctx.closureScopes() {
        var ctx Context = at(ctx, scope.position)
        if infos { warn(ctx, "%s", scope).debug(1) }
        if scope.project == nil || scope != scope.project.scope {
            if _, obj = scope.Find(name); isNil(obj) {
                // fallthrough
            } else if def, ok := obj.(*def); ok && (def.origin == DefAuto || def.origin == DefArg) {
                if o, ok := ctx.closureResolveAuto(name); ok && !isNil(o) { obj = o }
                if infos {
                    var proj = def.OwnerProject()
                    val := obj.expand(ctx.closure(), plain)
                    va2 := autoGet(ctx.closure(), name)
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
            obj = scope.project.resolveObject(ctx, name)
        }
        if isNil(obj) && false { obj = closureResolveObject(ctx.inner(), name) }
        if!isNil(obj) { if infos { warn(ctx, "%v", obj).debug(1) }; break }
    }
    return
}

func closureResolveEntry(ctx Context, name string) (entries *ResolveEntries) {
    for _, scope := range ctx.closureScopes() {
        if project := scope.project; project != nil {
            entries = project.resolveEntries(ctx, name, false, /*true*/false)
            if entries == nil && false {
                entries = closureResolveEntry(ctx.inner(), name)
            }
        }
        if entries != nil { break }
    }
    return
}

func closureWith(ctx Context, scopes ...*Scope) (res Context) {
    if c, ok := ctx.(*closureContext); false && ok {
        res = closureWith(c.Context, scopes...)
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

// traverseContext is a single thread traverse context, for traversing in a new goroutine,
// a spawned traversal must be used and then merge.
func (pc *programContext) level(n int) { pc.traceLevel += n }
func (pc *programContext) trace(a ...interface{}) { printIndentDots(pc.traceLevel, a...) }
func (pc *programContext) tracef(s string, a ...interface{}) { printIndentDots(pc.traceLevel, fmt.Sprintf(s, a...)) }
func (pc *programContext) traversed(target Value) (targets []Value) {
    if !isTrivial(target) {
        pc.targets = append(pc.targets, target)
    }
    targets = pc.targets
    return
}

func entryStr(ctx Context, entry Entry) (str, ent, tar string) {
    if !isNil(entry) { ent = entry.Strval(ctx) }
    if val := autoGet(ctx, "@"); val == nil || isTrivial(val) {
        str = ent // ...
    } else if tar = val.Strval(ctx); ent != tar {
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
        proj  = ctx.Project()
        entry = ctx.entry()
    )
    if s != "" && s != "<#" && s != "#>" {
        point = diag(ctx, dt, s, a...)
    } else if entry == nil {
        if point = ctx.diag(dt, "in project %v:", proj); proj != nil {
            point.position = proj.position
        }
        for last, i := ctx.Position(), ctx.inner(); i != nil; i = i.inner() {
            if pos := i.Position(); !pos.Same(&last) {
                point = ctx.diag(dt, "%v: from here", proj)
                point.position = pos
                last = pos
            }
        }
        return
    } else if proj == nil {
        proj = entry.OwnerProject()
    }

    var str, _, _ = entryStr(ctx, entry)
    if s == "<#" && len(a) > 0 { for _, t := range a {
        if v, ok := t.(Value); ok {
            if str == "" {
                point = ctx.diag(dt, "%v: %v (%T)", proj, v, v)
            } else {
                point = ctx.diag(dt, "%v: %v: %v (%T)", proj, str, v, v)
            }
            point.position = v.Position()
        }
    }}

    if point == nil {
        point = ctx.diag(dt, "%v", proj)
    } else if true {
        // ...
    } else if str == "" {
        point = ctx.diag(dt, "%v: %v", proj, ctx)
    } else {
        point = ctx.diag(dt, "%v: %v: %v", proj, str, ctx)
    }

    if pc := ctx.positionContext(); pc != nil {
        point.position = pc.position

        if s == "#>" && len(a) > 0 { for _, t := range a {
            if v, ok := t.(Value); ok {
                if str == "" {
                    point = ctx.diag(dt, "%v: %v (%T)", proj, v, v)
                } else {
                    point = ctx.diag(dt, "%v: %v: %v (%T)", proj, str, v, v)
                }
                point.position = v.Position()
            }
        }}

        for last := &pc.position; pc != nil && n > 0; pc = pc.Context.positionContext() {
            var pos = &pc.position
            if (last == pos || last./* SameLine */Same(pos)) { continue }

            var suf string
            if n == 1 && pc.caller() != nil { suf = " ..." }
            if entry := pc.entry(); entry == nil {
                point = ctx.diag(dt, "%v%s", proj, suf)
            } else if str, _, _ := entryStr(pc, entry); str != "" {
                point = ctx.diag(dt, "%v: (project=%v)%s", str, proj, suf)
            } else {
                point = ctx.diag(dt, "%v: %v%s", proj, entry, suf)
            }
            point.position = *pos

            if false && pc != nil { point = ctx.diag(dt, "%v: ...", proj) }

            last, n = pos, n-1
        }
    }
    return
}

func (pc *programContext) arguments() []Value { return nil }
func (pc *programContext) argumented() *argumentedContext { return nil }
func (pc *programContext) argumentedSet([]Value) []Value { return nil }
// func (pc *programContext) spawn(ctx Context) Context {
//     return &traverseContext{
//         Context: pc.Context.spawn(ctx),
//         print:   pc.print,
//         execRec: make(map[Value]int),
//         start:   time.Now(),
//     }
// }

func (pc *programContext) depth() (res int) {
    for c := pc.caller(); c != nil; c = c.caller() { res += 1 }
    return
}
func (pc *programContext) calleeError(err error) {
    if err != nil {
        pc.calleeErrsM.Lock()
        pc.calleeErrs = append(pc.calleeErrs, err)
        pc.calleeErrsM.Unlock()
    }
}

func exists(ctx Context, v Value) bool {
    // FIXME: returns true if existenceMatterless ??
    return v != nil && v.stat(ctx).exists() == existenceConfirmed
}

// traverse - traverse the prerrequiste for the current target $@
func traverse(ctx Context, prereqValue Value, prereq string, projects... *Project) (traves travestates) {
    var targetValue = getTargetValue(ctx)
    if targetValue == nil {
        prompt(of(ctx,prereqValue), "%s: target is nil\n", prereq)
        errostack(ctx, 3, "").debug(6)
        return
    } else if isTrivial(targetValue) {
        prompt(of(ctx,prereqValue), "%s: target is trivial (%T)\n", prereq, targetValue)
        errostack(ctx, 3, "").debug(6)
        return
    }

    const objectsFirst = true

    var (
        prereqPattern Value
        prereqFile *File
        prereqObj Object
    )

    // NOTE: Don't delete, keep it for future debugging.
    if false {
    // if true && (strings.HasSuffix(prereq, ".o") || strings.HasSuffix(prereq, ".a")) {
    // if true && strings.Contains(prereqValue.String(), "&(objects)($(objDeps))") {
    // if true && strings.Contains(targetValue.String(), "llvm-tools-driver") {
        prompt(ctx, "%v : %v\n", targetValue, prereqValue)
        warn(ctx, "@: %T %v", targetValue, targetValue)
        warn(ctx, ">: %T %v", prereqValue, prereqValue)
        warn(ctx, "^: %v", ctx.program().depends)
        warn(ctx, "&: %v", projects)
        warnstack(ctx, 3, "").debug(10)
        defer func() {
            warn(ctx, "%v : %v, %v, %v (%T)", targetValue,
                prereqValue, prereqFile, prereqObj, prereqObj).debug(10)
        } ()
    }

    if len(projects) == 0 { projects = ctx.projects(ctx) }
    if len(projects) == 0 {
        prompt(ctx, "%s: zero closure projects\n", prereq)
        erro(ctx, "no projects to traverse '%v' (%s)", prereqValue, prereq)
        erro(ctx, "%v: closure %v", prereq, len(ctx.closureScopes()))
        errostack(ctx, 3, "").debug(8)
        return
    } else if prereqValue == nil {
        if prereq != "" {
            prereqValue = MakeString(ctx.Position(), prereq)
        } else {
            errostack(ctx, 3, "prerequisite is none").debug(8)
            return
        }
    } else if prereqValue.patterned(ctx) {
        var stems = ctx.stems()
        if len(stems) == 0 {
            errostack(ctx, 3, "%v: no stems", prereqValue).debug(8)
            return
        } else if true {
            // does nothing
        } else if prereq != "" {
            errostack(ctx, 3, "%v: unwanted pattern name: %s",
                prereqValue, prereq).debug(8)
            return
        }

        var rest []string
        prereqPattern = prereqValue
        prereqValue, rest = prereqPattern.stencil(ctx, stems)
        if isTrivial(prereqValue) {
            errostack(ctx, 3, "%v: empty stencil with %v",
                prereqPattern, stems).debug(8)
            return
        } else if len(rest) > 0 {
            errostack(ctx, 3, "%v: partial stencil with %v, rest=%v",
                prereqPattern, stems, rest).debug(8)
            return
        } else if prereq != "" {
            // does nothing
        } else if prereq = prereqValue.Strval(ctx); prereq == "" {
            errostack(ctx, 3, "%v: empty prerequisite, stems=%v",
                prereqValue, stems).debug(8)
            return
        }
    } else {
        if prereq == "" { if prereq = prereqValue.Strval(ctx); prereq == "" {
            errostack(ctx, 3, "%v: %v: empty prerequisite, stems=%v",
                targetValue, prereqValue, ctx.stems()).debug(8)
            return
        }}

        var ok bool
        if prereqFile, ok = toFile(prereqValue); !ok {
            if prereqObj, ok = prereqValue.(Object); ok {
                info(ctx, "%T %v %s", prereqObj, prereqObj, prereq).debug(1)
            }
        }
    }

    if prereqFile == nil { if _, ok := prereqValue.(*String); ok {
        // escape file parsing for optimizaition
    } else if _, ok := prereqValue.(*Compound); ok {
        // escape file parsing for optimizaition
    } else { for _, project := range projects {
        if prereqFile = project.FindFile(ctx, prereq); prereqFile != nil {
            prereqValue = prereqFile
            break
        }
    }}}
    if prereqFile == nil { if _, ok := prereqValue.(*Path); ok /*&& filepath.IsAbs(prereq)*/ {
       if prereqFile = stat(ctx, prereq, "", ""); prereqFile != nil {
          prereqValue = prereqFile
       }
    }}

    // Recursion detection -- simply return to break it if this happens.
    if traveseDetectLoops { if eq(ctx, targetValue, prereqValue) {
        prompt(ctx, "%v: %v: self dependency, consider using [(once)] to avoid\n",
            targetValue, prereqValue)
        warn(of(ctx,prereqValue), "recursion: %T %v", prereqValue, prereqValue)
        warn(of(ctx,targetValue), "recursion: %T %v", targetValue, targetValue)
        warn(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projects)
        if false {
            warnstack(ctx, 16, "").debug(32)
        } else {
            errostack(ctx, 16, "").debug(32)
        }
        return
    }}
    if traveseDetectLoops { for c := ctx.programContext(); c != nil; c = c.caller() {
        if val := autoGet(c, "@"); val != nil && eq(c, val, prereqValue) {
            if traveseLoopBreakState != traveUnkn {
                var s = traves.add(ctx, traveseLoopBreakState, targetValue)
                if s.dependPat = prereqPattern; prereqFile == nil {
                    s.depend = prereqValue
                } else {
                    s.depend = prereqFile
                }
            }
            if f := asFile(ctx, targetValue, projects...); f != nil {
                // silent
            } else if true {
                prompt(ctx, "%v: %v: recursion detected, consider using [(once)] to avoid\n",
                    targetValue, prereqValue)
                warn(ctx, "recursion: %T %v", prereqValue, prereqValue)//.of(prereqValue)
                warn(ctx, "recursion: %T %v", targetValue, targetValue)//.of(targetValue)
                warn(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projects)
                warnstack(ctx, 3, "").debug(16)
            }
            return
        }
    }}

    // NOTE: Don't delete, keep this segment to save time for future debugging.
    // NOTE: Open this segment will draw very clear traverse path of a prerequisite.
    if false && strings.HasSuffix(prereq, ".o") {
        var s string
        if f := prereqFile; f != nil { s = "(" + f.dir + "," + f.sub + ")" }
        prompt(ctx, "%v : %v ; pattern=%v , file=%v%s\n", //, ctx.stemmed()
            targetValue, prereqValue, prereqPattern, prereqFile, s)
        warn(ctx, "@: %T %v", targetValue, targetValue)
        warn(ctx, ">: %T %v", prereqValue, prereqValue)
        warn(ctx, ">: %v in %v", prereqFile, projects)
        warnstack(ctx, 3, "").debug(10)
    }

    const db0 = false
    var dbg = true && (
        // strings.Contains(prereq, "table-gen"               ) ||
        strings.Contains(prereq, "llvm-tools-objcopy"      ) ||
            strings.Contains(prereq, "llvm-tools-bitcode-strip") ||
            strings.Contains(targetValue.Strval(ctx), "llvm-tools-objcopy"      ) ||
            strings.Contains(targetValue.Strval(ctx), "llvm-tools-bitcode-strip") ||
            // strings.Contains(targetValue.Strval(ctx), "llvm-driver-ar"         ) ||
            // strings.Contains(targetValue.Strval(ctx), "strstream.cpp.sm"       ) ||
            false)

    dbg = dbg || (false &&
        strings.Contains(prereq, "touch") &&
        strings.Contains(targetValue.Strval(ctx), "stamp") &&
        true)

    const promptTraveEntries = false
    // const promptTraveEntries = true
    // var promptTraveEntries = strings.Contains(prereq, "polly/Config/config.h")

    var (
        verb = options.verbose || options.verboseBreaks
        t = ctx.programContext()
        concreteList []Entry
        stemmedList []*stemmed
        traversed int
        okay bool
        err error
    )
    defer func() {
        var (
            av = targetValue
            bv = prereqValue
        )

        // Note that the file maybe not traversed yet at this point.
        // But we still have to check mod-time.
        if prereqFile == nil {
            ctx.traversed(prereqValue) // set $< $> $^ or $|
        } else if /*targetValue != prereqValue && */targetValue != prereqFile {
            ctx.traversed(prereqFile) // set $< $> $^ or $|
            bv = prereqFile
        } else if t := traves.of(traveFile); t.has() { for _, s := range t {
            if d := s.depend; d != bv && !isTrivial(d) { bv = d }
        }}

        if !isNil(av) && !isNil(bv) {
            var (
                a = av.stat(ctx).mod()
                b = bv.stat(ctx).mod()
            )
            if (!a.IsZero() && b.After(a)) || bv.updated(ctx) || bv.updatedDeps(ctx) != nil {
                if false {
                    av.updated(ctx, true)
                } else {
                    av.updatedDeps(ctx, bv)
                }
            }
        }

        if prereqFile != nil {
            if okay && traversed > 0 && prereqFile.exists() && traves.has(traveNext) {
                traves = traves.not(traveNext)
            }
            if true && !traves.has(traveFile) {
                trave := traves.add(ctx, traveFile, targetValue)
                trave.dependPat = prereqPattern
                trave.depend = prereqFile
            }
        }
    } ()

    var searchObjects = func() {
        var obj Object

    ForProjectsObjects:
        for _, project := range projects {
            switch obj = project.resolveObject(ctx, prereq); obj.(type) {
            case *ProjectName: if objectsFirst {
                switch prereqValue.(type) {
                case *String, *Compound: return // let it find rule entries
                }
                if prereqFile != nil { return }  // let it find rule entries
            }
            case *Builtin, *def, *ScopeName, *unresolvedobject:
                continue ForProjectsObjects
            default: if isTrivial(obj) { continue ForProjectsObjects }
            }

            prereqObj = obj

            var t = obj.traverse(ctx)
            if promptTraveEntries { prompt(ctx, "%v: %T: %T %v %v ; %v\n",
                targetValue, targetValue, prereqValue, prereqValue, obj, t).debug(1) }

            okay = true
            traversed += 1
            traves = append(traves, t...)
            s := traves.add(ctx, traveObj, targetValue)
            s.depend = obj

            if t.has(traveFail) {
                prompt(ctx, "%v → %T %v\n", prereq, obj, obj)
                for _, s := range t { warn(at(ctx,s.pos), "%v: (%T) %v", obj, obj, s) }
                warnstack(ctx, 5, "").debug(8)
                return
            }
        } // ForProjectsObjects
    }
    if objectsFirst {
        if searchObjects(); okay { return }
    }

    // %.h <-> 'llvm/PassSupport.h' <-> [file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h done@llvm/PassSupport.h file@llvm/PassSupport.h>/Volumes/workspace/external/llvm-project/llvm/include/llvm/PassSupport.h done@llvm/PassSupport.h]
    var nonFilePrereqTravedFile = func (s *travestate) (res bool, resFile *File) {
        if prereqFile == nil { // the prereqValue is not a *File (eg a *String)
            // and the trave target is a *File with the name matched
            if f, ok := toFile(s.target); ok && f.name == prereq {
                res, resFile = true, f
            }
        }
        return
    }

    type traveResT int
    const (
        traveResContinue traveResT = iota
        traveResBreak
        traveResReturn
    )
    var trave = func (project *Project, entry Entry, pattern bool) (result traveResT) {
        traversed += 1
        traves = nil

        var t = entry.traverse(ctx)
        {
            s := traves.add(ctx, traveRule, targetValue);
            s.depend = entry
        }

        if !t.has() { return traveResContinue }
        if promptTraveEntries || false && t.has(traveFail, traveNext) {
            var g = targetValue.Strval(ctx)
            if s, ok := entry.(*stemmed); ok {
                prompt(ctx, "%v:(%T): %T %v ⇒ %v\n", g, targetValue, prereqValue, prereqValue, s.PatternEntry)
            } else {
                prompt(ctx, "%v:(%T): %T %v ⇒ %v\n", g, targetValue, prereqValue, prereqValue, entry)
            }
            prompt(ctx, "%v:(%T): %v\n", g, targetValue, t)
            infostack(ctx, 5, "<#", prereqValue).debug(1)
            defer func() { prompt(ctx, "%v:(%T): %v\n", g, targetValue, t).debug(32) } ()
        }

        // NOTE: collect travestates from t according to each trave type

        if tt := t.of(traveFail); tt.has() {
            if !pattern || verb {
                var stems = ctx.stems()
                prompt(ctx, "%s: traverse entry failed (project=%v, target=%v, stems=%v)\n", entry, project, targetValue, stems)
                warn(of(ctx,entry), "%v (%T) (by %v, in %v)", entry, entry.Target(), targetValue, entry.OwnerProject())
                for _, s := range traves { warn(at(ctx,s.pos), "%v: %v: %v", targetValue, entry, s) }
                if n, m := 5, 16; len(stems) == 0 {
                    errostack(ctx, n, "#>", prereqValue).debug(m)
                } else {
                    warnstack(ctx, n, "#>", prereqValue).debug(m)
                }
            }
            for _, s := range tt {
                if g := s.target; g != nil && eq(ctx, prereqValue, g) && true {
                    warn(of(ctx,entry), "%v (%T) (by %v, in %v)", entry, entry.Target(), targetValue, entry.OwnerProject()).debug(1)
                    return traveResContinue
                }
            }
            return traveResReturn
        }

        if tt := t.of(traveCase, traveDone); tt.has() {
            for _, s := range tt {
                if /*s.what == traveDone*/true { if ok, f := nonFilePrereqTravedFile(s); ok {
                    okay, prereqFile = true, f
                    return traveResReturn
                }}
                // if g, d := s.target, s.depend; g != nil && d != nil &&
                if g := s.target; g != nil &&
                    // eq(ctx, targetValue, g) &&
                    // eq(ctx, prereqValue, d) &&
                    // eq(ctx, targetValue, d) &&
                    eq(ctx, prereqValue, g) && true {
                    // traves = traves.not(traveCase, traveDone)
                    okay = true
                    return traveResReturn
                }
            }
        }

        // NOTE: foo foo.o [next@foo.o>foo.c]
        // NOTE: foo.pdf foo.tex [next@foo.tex:%%.org>foo.org]
        if tt := t.of(traveNext); tt.has() {
            const dbgNext = false
            var stemmedThisTarget bool
            if s := ctx.stemmed(); s != nil && s.target != nil {
                stemmedThisTarget = eq(ctx, s.target, targetValue)
            }
            for _, s := range tt {
                if g := s.target; g == nil {
                    prompt(ctx, "%v: %v\n", targetValue.Strval(ctx), t)
                    erro(ctx, "%v: %v %v %v\n", targetValue.Strval(ctx),
                        prereqPattern, prereqValue, pattern)
                    errostack(ctx, 5, "").debug(1)
                } else if true && eq(ctx, targetValue, g) {
                    if dbgNext || false {
                        prompt(ctx, "%v: %v\n", targetValue.Strval(ctx), t)
                        prompt(ctx, "%v: %v %v (%v,%v)\n", targetValue.Strval(ctx),
                            prereqPattern, prereqValue, pattern, stemmedThisTarget)
                        info(ctx, "%T %v ; %v", prereqValue, prereqValue, ctx.stemmed())
                        infostack(of(ctx,prereqValue), 5, "").debug(1)
                    }
                    if /*pattern && */stemmedThisTarget {
                        traves = append(traves, s) // collect the state
                    }
                    return traveResContinue // try the next pattern
                } else if true && eq(ctx, prereqValue, g) {
                    if dbgNext {
                        prompt(ctx, "%v: %v\n", targetValue.Strval(ctx), t)
                        prompt(ctx, "%v: %v %v (%v,%v)\n", targetValue.Strval(ctx),
                            prereqPattern, prereqValue, pattern, stemmedThisTarget)
                        info(ctx, "%T %v %v", prereqValue, prereqValue, ctx.stemmed())
                        infostack(of(ctx,prereqValue), 5, "").debug(1)
                    }
                    if pattern && stemmedThisTarget {
                        // IMPORTANT NOTE: traveNext state should be remained
                        //   This traveNext state muse be returned, so that the
                        //   (*Program).traverse func can break it's loop properly.
                        traves = append(traves, s) // collect the state
                    }
                    return traveResContinue // try the next pattern
                }
            }
        }

        // NOTE: *smart.File memory.o <-> [file@memory.o>memory.cpp file@memory.o>...]
        if tt := t.of(traveFile); tt.has() {
            for _, s := range tt {
                if ok, f := nonFilePrereqTravedFile(s); ok {
                    okay, prereqFile = true, f
                    return traveResReturn // file processed
                } else if g := s.target; g != nil && eq(ctx, prereqValue, g) &&
                    true {

                    // NOTE:2: turn of this so that files get updated if changed
                    // NOTE: only 'okay' if files exists, it can continue trying other
                    //       rules if not.
                    if false && prereqFile != nil && prereqFile.exists() {
                        if f, ok = toFile(s.depend); ok && f != nil {
                            okay = f.exists()
                        } else {
                            okay = true
                        }
                    }

                    var op = traveResReturn // assuming file processed
                    if !okay || !entry.hasRecipes() {
                        op = traveResContinue
                    }
                    return op
                } else if _, ok := prereqValue.(*String); false && ok {
                    if prereqFile == nil { prompt(ctx, "nonfile: %T %v: %T %v : %T %v ; %v\n",
                        targetValue, targetValue, prereqValue, prereqValue,
                        s.target, s.target, s).debug(1) }
                }
            }
        }

        return
    }

    ForProjectsConcretes: for _, project := range projects {
        var entries = project.resolveEntries(ctx, prereq, t.grepping, false)
        if entries != nil && len(entries.all) > 0 {
            concreteList = append(concreteList, entries.all...)
        } else {
            continue ForProjectsConcretes
        }
        ForEntries: for _, entry := range entries.all {
            if !isNil(entry) && targetValue == entry { continue ForEntries }
            if w, k := targetValue.(*Bareword); k && w.string == prereq {
                continue ForEntries // target resolve to itself, does nothing
            }
            switch trave(project, entry, false) {
            case traveResBreak : break ForEntries
            case traveResReturn: break ForEntries //return
            }
        }
    }

    if !okay || traversed == 0 { ForProjectsPatterns: for _, project := range projects {
        var patterns = project.resolvePatterns(ctx, prereqValue, prereq)
        if len(patterns) > 0 {
            stemmedList = append(stemmedList, patterns...)
        } else {
            continue ForProjectsPatterns
        }
        ForPatterns: for _, entry := range patterns {
            switch trave(project, entry, true) {
            case traveResBreak : break ForPatterns
            case traveResReturn: return
            }
        }
    }}

    if okay && traversed > 0 {
        return
    } else if prereqFile != nil && prereq == prereqFile.name {
        if okay = prereqFile.exists(); okay {
            trave := traves.add(ctx, traveFile, targetValue)
            trave.dependPat = prereqPattern
            trave.depend = prereqFile
            return
        }
        for _, project := range projects {
            if okay = prereqFile.searchInMatchedPaths(ctx, project); okay {
                assert(prereqFile.exists(), "file must exists at this point")
                trave := traves.add(ctx, traveFile, targetValue)
                trave.dependPat = prereqPattern
                trave.depend = prereqFile
                return
            }
        }
    } else if !okay && traversed == 0 { ForProjectsFiles: for _, project := range projects {
        if prereqFile = project.FindFile(ctx, prereq); prereqFile != nil {
            if prereqFile.position = ctx.Position(); prereqFile.isSysFile() {
                continue ForProjectsFiles
            }
            if okay = prereqFile.exists(); okay {
                trave := traves.add(ctx, traveFile, targetValue)
                trave.dependPat = prereqPattern
                trave.depend = prereqFile
                return
            }
            if okay = prereqFile.searchInMatchedPaths(ctx, project); okay {
                trave := traves.add(ctx, traveFile, targetValue)
                trave.dependPat = prereqPattern
                trave.depend = prereqFile
                return
            }
        }
    }} // ForProjectsFiles

    if prereqFile != nil && prereqFile.exists() && !traves.has(traveFail) {
        okay = true
        return
    } else if objectsFirst || okay || traversed>0 {
        // does nothing
    } else if searchObjects(); okay {
        return
    }

    // Stat directly for files not in included in "files (...)".
    if !okay && prereqFile != nil && prereqFile.exists() && !traves.has() {
        return // done
    } else if true && !okay && prereqFile == nil && prereqObj == nil && !traves.has() {
        if f := stat(ctx, prereq, "", ""); f != nil && f.exists() {
            if false { warn(ctx, "%v (%T) is a file (%s)",
                prereqValue, prereqValue, f.fullname()).debug(6) }
            trave := traves.add(ctx, traveFile, targetValue)
            trave.dependPat = prereqPattern
            trave.depend = f
            prereqFile = f
            return
        }
    }

    var proj = ctx.Project()
    if err != nil {
        prompt(ctx, "%s: traverse failed, project %s\n", prereq, proj)
        errostack(ctx, 6, "%v: %v", proj, ctx).debug(16)
        return
    }

    if okay {
        return // done
    } else if traves.has(traveDone) {
        return // done
    } else if ctx.configuration() {
        return // done
    }

    if t := traves.of(traveRule); t.has() && t[0].depend != nil {
        if s, ok := t[0].depend.(*stemmed); ok {
            if false {
                for i, concrete := range concreteList {
                    warn(at(ctx,concrete.Position()), "concrete: %d. %v (%d programs)", i, concrete, len(concrete.Programs()))
                }
                for i, stemmed  := range stemmedList {
                    warn(at(ctx,stemmed.position), "stemmed: %d. %v", i, stemmed)
                }
                warn(ctx, "%v %s", s.Name(ctx), prereq).debug(1)
            }
            if true { // NOTE: add traveNext or not seems the same (not better)
                s := traves.add(ctx, traveNext, targetValue)
                s.dependPat = prereqPattern
                s.depend = prereqValue
            }
            return
        }
        if t[0].depend.(Entry).Name(ctx) == prereq {
            return // done
        }
    }
    if t := traves.of(traveObj); t.has() && t[0].depend != nil {
        if t[0].depend.(Object).Name(ctx) == prereq {
            return // done
        }
    }

    if prereqPattern != nil {
        if false {
            for i, concrete := range concreteList {
                warn(at(ctx,concrete.Position()), "concrete: %d. %v (%d programs)", i, concrete, len(concrete.Programs()))
            }
            for i, stemmed  := range stemmedList {
                warn(at(ctx,stemmed.position), "stemmed: %d. %v", i, stemmed)
            }
            warn(ctx, "%v %s", prereqPattern, prereq).debug(1)
        }
        s := traves.add(ctx, traveNext, targetValue)
        s.dependPat = prereqPattern
        s.depend = prereqValue
        return
    }

    if false && (traversed>0 && !traves.has()) {
        return // done
    }

    if len(ctx.stems()) == 0 || ctx.mustExists() {
        if s := traves.add(ctx, traveFail, targetValue); prereqFile != nil {
            s.error = fileNotFoundError{proj, prereqFile}
            ctx = at(ctx, prereqFile.position)
        } else {
            s.error = targetNotFoundError{proj, prereq}
            if prereqValue != nil { ctx = at(ctx, prereqValue.Position()) }
        }

        if prereqFile != nil && prereqValue != prereqFile {
            prompt(ctx, "%v:(%T): %T %v; okay=%v traversed=%d file=%v projects=%v\n",
                targetValue, targetValue, prereqValue, prereqValue,
                okay, traversed, prereqFile, projects).debug(1)
        } else if prereqFile != nil {
            prompt(ctx, "%v:(%T): %T %v; okay=%v traversed=%d path=%s projects=%v\n",
                targetValue, targetValue, prereqValue, prereqValue,
                okay, traversed, prereqFile.fullname(), projects).debug(1)
        } else if prereqObj != nil {
            prompt(ctx, "%v:(%T): %T %v; okay=%v traversed=%d obj=%v projects=%v\n",
                targetValue, targetValue, prereqValue, prereqValue,
                okay, traversed, prereqObj, projects).debug(1)
        } else {
            prompt(ctx, "%v:(%T): %T %v; okay=%v traversed=%d projects=%v\n",
                targetValue, targetValue, prereqValue, prereqValue,
                okay, traversed, projects).debug(1)
        }

        for i, s := range traves { erro(at(ctx,s.pos), "%v: %v: %d. %v", targetValue, prereqValue, i, s) }
        for i, c := range ctx.closureScopes() { erro(ctx, "%v: closure: %v. %v", proj, i, c) }
        for i, concrete := range concreteList { erro(at(ctx,concrete.Position()), "concrete: %d. %v (%d programs)", i, concrete, len(concrete.Programs())) }
        for i, stemmed  := range stemmedList  { erro(at(ctx,stemmed.position), "stemmed: %d. %v", i, stemmed) }
        if obj := prereqObj; obj != nil { erro(at(ctx,obj.Position()), "object: %T %v", obj, obj) }
        errostack(ctx, 12, "").debug(512)
        return
    } else {
        return
    }
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

func getRecipesHash(ctx Context, target Value, values ...Value) (k, v HashBytes, err error) {
    var (
        // target = getTargetValue(ctx)
        program = ctx.program()
        key = sha256.New()
        val = sha256.New()
    )
    fmt.Fprintf(key, "%s", program.project.absPath)
    fmt.Fprintf(key, "%s", program.position)
    fmt.Fprintf(key, "%s", target.Strval(ctx)) // targetStr

    for _, value := range values {
        if false {
            // FIXME: Strval() varies when &(var) is used
            fmt.Fprintf(val, "%v", value.Strval(ctx))
        } else {
            fmt.Fprintf(val, "%v", value)
        }
    }
    copy(k[:], key.Sum(nil))
    copy(v[:], val.Sum(nil))
    return
}

func updateRecipesHash(ctx Context, target Value) (k, v HashBytes, err error) {
    var program = ctx.program()
    if k, v, err = getRecipesHash(ctx, target, program.recipes...); err != nil {
        erro(at(ctx,program.position), "hashing recipes failed: %v", err).debug(1)
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

func isRecipesChanged(ctx Context, target Value) (outdated bool, err error) {
    var k, v HashBytes
    if program := ctx.program(); program == nil {
        erro(ctx, "no program in context %v", ctx).debug(1)
        return
    } else if k, v, err = getRecipesHash(ctx, target, program.recipes...); err != nil {
        erro(at(ctx,program.position), "compute recipes hash failed: %v", err).debug(1)
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
    var ( // waiting for prerequisites
        pos Position = ctx.Position()
        calleeErrs []error
    )

    if false { ctx.wait() } // aka programContext.WaitGroup.Wait(), FIXME: deadlock

    if t := ctx.programContext(); t != nil {
        //t.group.Wait()
        t.calleeErrsM.Lock()
        calleeErrs = t.calleeErrs; t.calleeErrs = nil
        t.calleeErrsM.Unlock()
    }

    if target = getTargetValue(ctx); target == nil {
        erro(at(ctx,pos), "target is nil")
        errostack(ctx, 8, "").debug(8)
        return
    } else if isTrivial(target) {
        erro(at(ctx,pos), "trivial target (%T)", target)
        errostack(ctx, 8, "").debug(8)
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
        if !pos.Same(&targetPos) {
            var s string; if n > 1 { s = "s" }
            erro(at(ctx,targetPos), "%d error%s while waiting prerequisites for '%v'", n, s, target).debug(1)
        }

        var v = target
        if l, ok := v.(*List); ok && l.Len() == 1 { v = l.Elems[0] }
        if targetValuePos.IsValid() && !targetValuePos.Same(&targetPos) {
            if f, y := toFile(v); y && f != nil && f.filemap != nil {
                erro(at(ctx,targetValuePos), "waiting for '%v'", target)
                erro(of(ctx,f.filemap.patts[0]), "via pattern '%v' (of %v)", v, f.filemap.project).debug(1)
            } else {
                erro(at(ctx,targetValuePos), "waiting for '%v'", target).debug(1)
            }
        }
        if def, ok := v.(*def); ok && target != v && target != def.value { // trace source Def in diagnostics
            erro(of(ctx,def.value), "waiting for def '%v': %v", def.name, def.value).debug(1)
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
        if val := autoGet(ctx, "-"); val != nil {
            var ok bool
            if execRes, ok = val.(*ExecResult); ok {
                //execRes.wg.Wait()
            }
        }
    }
    if !optStampCurrentTarget {
        // done!
    } else if files, err = target.stamp(ctx); err != nil {
        if p := target.Position(); p.IsValid() { erro(at(ctx,p), "%v", err) }
        erro(at(ctx,pos), "%v", err).debug(1)
        return
    } else if optReportFileUpdates {
        reportFileUpdates(ctx, ctx.programContext().start, files)
    }
    return
}

type kind int

const (
    valOther kind = iota
    valInteger
    valFloat
)

// Value represents a value of a type.
type Value interface {
    Positioner // The position where the value appears (or NoPos).

    kind() kind

    // Literal representations of the value.
    String() string

    // Strval returns the string form of the value.
    Strval(Context) string

    // Returns true if the value can be evaluated as 'true', 'yes', etc.
    True(Context) bool

    // Integer returns the integer form of the value.
    Integer(Context) (int64, error)

    // Float returns the float form of the value.
    Float(Context) (float64, error)

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
    defs(Context, ...string) []*def

    // Test if this value is expandible for some bits.
    expandible(Context, facet) bool

    // &(...)        -> $(...)
    // $(...)        -> ......
    // $(...)=$(...) -> ...=$(...), ...=...
    // foo->bar      -> ...
    expand(Context, facet) Value // result is nil or identical to this value if no expansions

    stat(Context) (*statinfo)

    // Stamp the value if it's a file (aka. update FileInfo).
    stamp(Context) ([]*File, error)

    // Delete the file (if it is).
    delete(Context) ([]*File, error)

    updated(Context, ...bool) bool
    updatedDeps(Context, ...Value) []Value

    traverse(Context) travestates
}

type valueList []Value

func (vals valueList) contains(val Value) (res bool) {
    if val != nil {
        for _, v := range vals {
            if res = v == val; res { break }
        }
    }
    return
}

func (vals *valueList) add(val Value) (res *valueList) {
    if val != nil && !vals.contains(val) {
        *vals = append(*vals, val)
    }
    return vals
}

type elemkind int
const (
    elemNoQuote elemkind = 1<<iota
    elemNoBrace
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
func (_ *valbase) Strval(_ Context) (s string) { return }
func (_ *valbase) Integer(_ Context) (i int64, e error) { return }
func (_ *valbase) Float(_ Context) (f float64, e error) { return }
func (_ *valbase) True(_ Context) (res bool) { return }
func (_ *valbase) refs(_ Context, _ Value) (res bool) { return }
func (_ *valbase) defs(_ Context, _ ...string) (res []*def) { return }
func (_ *valbase) expandible(_ Context, _ facet) bool { return false }
func (_ *valbase) cmp(_ Context, _ Value) (res cmpres) { return }
func (_ *valbase) patterned(_ Context) bool { return false }
func (_ *valbase) match(_ Context, i interface{}) (full bool, s string, stems []string) { return }
func (_ *valbase) stencil(_ Context, stems []string) (val Value, rest []string) { return }
func (_ *valbase) stat(ctx Context) (si *statinfo) { return }
func (_ *valbase) stamp(ctx Context) (file []*File, err error) { return }
func (_ *valbase) updated(_ Context, _ ...bool) bool { return false }
func (_ *valbase) updatedDeps(_ Context, _ ...Value) []Value { return nil }
func (_ *valbase) delete(ctx Context) (file []*File, err error) { return }
func (_ *valbase) traverse(ctx Context) (traves travestates) { return }
func (_ *valbase) _match(ctx Context, p Value, i interface{}) (full bool, s string, stems []string) {
    var is string
    var v = p.Strval(ctx)
    switch t := i.(type) {
    case string: is = t
    case Value : is = t.Strval(ctx)
    default:
        erro(of(ctx,p), "%T: matching unsupported value: %T %v", p, i, i).debug(1)
        return
    }
    if strings.HasPrefix(is, v) { s, full = v, (len(v) == len(is)) }
    return
}
func (_ *valbase) kind() kind { return valOther }

type undef struct { Value }
func (p undef) expand(Context, facet) Value { return p }
func (p undef) String() string { return fmt.Sprintf("undef{%s}",p.Value) }
func (p undef) Strval(Context) (s string) { return }
func (p undef) Integer(Context) (i int64, e error) { return }
func (p undef) Float(Context) (f float64, e error) { return }
func (p undef) True(Context) (res bool) { return }

type returner struct {
    valbase
    Values []Value
}
func (p *returner) expand(ctx Context, w facet) (res Value) {
    var vals, u, n = w.expand(ctx, p.Values...)
    if n > 0 { res = &returner{p.valbase, vals} } else { res = p }
    if u > 0 { res = unexpanded{res} }
    return
}

type Argumented struct {
    value Value
    args []Value
}
func (p *Argumented) Position() Position { return p.value.Position() }
func (_ *Argumented) kind() kind { return valOther }
func (p *Argumented) refs(ctx Context, v Value) bool {
    if p.value.refs(ctx, v) { return true }
    for _, a := range p.args {
        if a.refs(ctx, v) { return true }
    }
    return false
}
func (p *Argumented) defs(ctx Context, s ...string) (res []*def) {
    res = p.value.defs(ctx, s...)
    for _, a := range p.args {
        res = append(res, a.defs(ctx, s...)...)
    }
    return
}
func (p *Argumented) updated(ctx Context, v ...bool) bool { return p.value.updated(ctx, v...) }
func (p *Argumented) updatedDeps(ctx Context, v ...Value) []Value { return p.value.updatedDeps(ctx, v...) }
func (p *Argumented) expandible(ctx Context, w facet) (res bool) {
    if res = p.value.expandible(ctx, w); !res && w&expandArgedArgs != 0 {
        for _, a := range p.args {
            if res = a.expandible(ctx, w); res { break }
        }
    }
    return
}
func (p *Argumented) expand(ctx Context, w facet) (res Value) {
    var (
        args []Value = p.args
        val = p.value.expand(ctx, w)
        u, n int
    )
    if w&expandArgedArgs != 0 {
        args, u, n = w.expand(ctx, args...)
    }
    if len(args) == 0 {
        res = val
    } else if val != p.value || n > 0 {
        if un, y := val.(unexpanded); y {
            if un.Value != p.value { val = un.Value }
            u += 1
        }
        res = &Argumented{ val, args }
    } else {
        res = p
    }
    if u > 0 { res = unexpanded{res} }
    return
}
func (p *Argumented) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Argumented); ok {
        if res = p.value.cmp(ctx, a.value); res == cmpEqual {
            // FIXME: check p.args against a.args?
        }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
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
func (p *Argumented) True(ctx Context) (res bool) {
    if p.value != nil { res = p.value.True(ctx) }
    return
}
func (p *Argumented) Integer(ctx Context) (i int64, err error) {
    return strconv.ParseInt(p.value.Strval(ctx), 10, 64)
}
func (p *Argumented) Float(ctx Context) (f float64, err error) {
    return strconv.ParseFloat(p.value.Strval(ctx), 64)
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
func (p *Argumented) Strval(ctx Context) (s string) {
    s = p.value.Strval(ctx) + "("
    for i, a := range p.args {
        if i > 0 { s += "," }
        s += a.Strval(ctx)
    }
    s += ")"
    return
}

func (p *Argumented) traverse(ctx Context) (traves travestates) {
    //!< IMPORTANT! - Don't merge-expand arguments here!
    //!< Arguments should be passed to program.execute as it is.
    var args []Value
    if traverseArgumentedExpand {
        var proj = ctx.Project()
        // NOTE: expand here to avoid args being expanded in the wrong context
        const w = plain|expandPairVal|expandPatterned//|expandFullName
        for _, a := range p.args {
            a = a.expand(ctx, w)
            // TODO: deal with pattern args using expandPatterned instead of stenciling:
            if true && a.patterned(ctx) { if stems := ctx.stems(); len(stems) > 0 {
                if val, rest := a.stencil(ctx, stems); len(rest) > 0 {
                    erro(of(ctx,a), "partial stencil: %v, %T %v, %v, %v", a, val, val, rest, stems).debug(1)
                    panic(fmt.Sprintf("%T %v", val, val))
                } else if file, okay := toFile(val); okay {
                    a = file
                } else if file := proj.FindFile(ctx, val.Strval(ctx)); file != nil {
                    a = file
                } else {
                    a = val //MakeString(a.Position(), str)
                }
            }}
            args = append(args, a)
        }
    } else {
        args = p.args
    }
    return p.value.traverse(&argumentedContext{ ctx, args })
}

type None struct { valbase }
func (p *None) expand(_ Context, _ facet) Value { return p }
func (p *None) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*None); ok {
        res = cmpEqual
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if s, ok := v.(*String); ok && s.string == "" {
        res = cmpEqual
    } else if l, ok := v.(*List); ok && len(l.Elems) == 0 {
        res = cmpEqual
    } else if ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}

type Nil struct { valbase }
func (p *Nil) expand(_ Context, _ facet) Value { return p }
func (p *Nil) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*Nil); ok {
        res = cmpEqual
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
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

func eq(c Context, l, r Value) bool {
    return l == r || l.cmp(c, r) == cmpEqual
}

// Any is used to box an arbitrary value
type Any struct { value interface{} }
func (_ *Any) kind() kind { return valOther }
func (p *Any) cmp(ctx Context, v Value) (res cmpres) {
    switch a := v.(type) {
    case *Any:
        if p.value == a.value {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            if v2, ok := a.value.(Value); ok {
                res = v1.cmp(ctx, v2)
            }
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
            res = p.cmp(ctx, l.Elems[0])
        }
    case Value:
        if p.value == a {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            res = v1.cmp(ctx, a)
        } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
            res = p.cmp(ctx, l.Elems[0])
        }
    case unexpanded:
        if a.Value != nil { res = p.cmp(ctx, a.Value) }
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
func (p *Any) updated(ctx Context, v ...bool) (res bool) {
    if p.value == nil {
        // does nothing
    } else if val, ok := p.value.(Value); ok {
        res = val.updated(ctx, v...)
    }
    return
}
func (p *Any) updatedDeps(ctx Context, v ...Value) (res []Value) {
    if p.value == nil {
        // does nothing
    } else if val, ok := p.value.(Value); ok {
        res = val.updatedDeps(ctx, v...)
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
func (p *Any) expand(ctx Context, w facet) (res Value) {
    if val, ok := p.value.(Value); ok && !isNil(val) {
        res = val.expand(ctx, w)
    } else {
        res = p
    }
    return
}
func (p *Any) refs(ctx Context, o Value) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.refs(ctx, o) }
    return
}
func (p *Any) defs(ctx Context, s ...string) (res []*def) {
    if v, ok := p.value.(Value); ok { res = v.defs(ctx, s...) }
    return
}
func (p *Any) expandible(ctx Context, w facet) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.expandible(ctx, w) }
    return
}
func (p *Any) Position() (res Position) {
    if v, ok := p.value.(Positioner); ok { res = v.Position() }
    return
}
func (p *Any) True(ctx Context) (t bool) {
    switch v := p.value.(type) {
    case Value:     t = v.True(ctx)
    case float32:   t = math.Abs(float64(v))-0 >= FloatEpsilon
    case float64:   t = math.Abs(v)-0 >= FloatEpsilon
    case int64:     t = v != 0
    case int:       t = v != 0
    case bool:      t = v
    }
    return
}
func (p *Any) Float(ctx Context) (res float64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.Float(ctx)
    case float32:     res = float64(v)
    case float64:     res = v
    case int:         res = float64(v)
    case int64:       res = float64(v)
    case bool: if v { res = FloatEpsilon }
    }
    return
}
func (p *Any) Integer(ctx Context) (res int64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.Integer(ctx)
    case float32:     res = int64(v)
    case float64:     res = int64(v)
    case int:         res = int64(v)
    case int64:       res = v
    case bool: if v { res = 1 }
    }
    return
}
func (p *Any) Strval(ctx Context) (s string) {
    switch v := p.value.(type) {
    case Value:       s = v.Strval(ctx)
    case float32:     s = strconv.FormatFloat(float64(v),'g', -1, 32)
    case float64:     s = strconv.FormatFloat(float64(v),'g', -1, 64)
    case int:         s = strconv.FormatInt(int64(v),10)
    case int64:       s = strconv.FormatInt(int64(v),10)
    case bool: if v { s = "true" } else { s = "false" }
    default: s = fmt.Sprintf("%s", p.value)
    }
    return
}
func (p *Any) String() string { return fmt.Sprintf("<%v>", p.value) }
func (p *Any) traverse(ctx Context) (traves travestates) {
    if v, ok := p.value.(Value); ok { traves = v.traverse(ctx) }
    return
}

type negative struct { valbase; x Value }
func (p *negative) refs(ctx Context, o Value) bool { return p.x.refs(ctx, o) }
func (p *negative) defs(ctx Context, s ...string) []*def { return p.x.defs(ctx, s...) }
func (p *negative) expandible(ctx Context, w facet) bool { return p.x.expandible(ctx, w) }
func (p *negative) expand(ctx Context, w facet) (res Value) {
    if val := p.x.expand(ctx, w); val != p.x {
        res = &negative{p.valbase, val}
    } else {
        res = p
    }
    return
}
func (p *negative) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*negative); ok {
        res = p.x.cmp(ctx, a.x)
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (p *negative) True(ctx Context) (res bool) {
    if p.x != nil { res = !p.x.True(ctx) }
    return
}
func (p *negative) elemStr(ctx Context, o Object, k elemkind) string {
    return `!`+elementString(ctx, o, p.x, k)
}
func (p *negative) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *negative) Strval(ctx Context) string { return fmt.Sprintf("%v", !p.x.True(ctx)) }
func (p *negative) Float(ctx Context) (res float64, _ error) {
    if !p.x.True(ctx) { res = FloatEpsilon }
    return
}
func (p *negative) Integer(ctx Context) (res int64, _ error) {
    if !p.x.True(ctx) { res = 1 }
    return
}
func (p *negative) traverse(ctx Context) (traves travestates) {
    if p.x != nil { traves = p.x.traverse(ctx) }
    return
}

func Negative(val Value) *negative { return &negative{valbase{val.Position()},val} }

type boolean struct { valbase; bool }
func (p *boolean) String() (s string) {
    if p.bool { s = "true" } else { s = "false" }
    return s+"{}" // or: fmt.Sprintf("bool{%s}",p.bool)
}
func (p *boolean) Strval(_ Context) string { return p.String() }
func (p *boolean) True(_ Context) bool { return p.bool }
func (p *boolean) Float(_ Context) (v float64, _ error) {
    if p.bool { v = 1. }
    return
}
func (p *boolean) Integer(_ Context) (v int64, _ error) {
    if p.bool { v = 1 }
    return
}
func (p *boolean) expand(_ Context, _ facet) Value { return p }
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
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
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
    return s+"{}"
}
func (p *answer) Strval(ctx Context) string { return p.String() }
func (p *answer) True(ctx Context) bool { return p.bool }
func (p *answer) Float(ctx Context) (v float64, _ error) {
    if p.bool { v = 1. }
    return
}
func (p *answer) Integer(ctx Context) (v int64, _ error) {
    if p.bool { v = 1 }
    return
}
func (p *answer) expand(_ Context, _ facet) Value { return p }
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
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
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
func (p *option) Strval(ctx Context) string { return p.String() }
func (p *option) True(ctx Context) bool { return p.bool }
func (p *option) Float(ctx Context) (v float64, _ error) {
    if p.bool { v = 1. }
    return
}
func (p *option) Integer(ctx Context) (v int64, _ error) {
    if p.bool { v = 1 }
    return
}
func (p *option) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *option) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *option) expand(_ Context, _ facet) Value { return p }
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
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
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
func (p *integer) True(ctx Context) bool { return p.int64 != 0 }
func (p *integer) Integer(ctx Context) (i int64, _ error) { return p.int64, nil }
func (p *integer) Float(ctx Context) (f float64, _ error) { return float64(p.int64), nil }
func (p *integer) kind() kind { return valInteger }
func (p *integer) cmp(ctx Context, v Value) (res cmpres) {
    if i, e := v.Integer(ctx); e != nil {
        warnstack(ctx, 6, "%T %v: %v", v, v, e).debug(20)
    } else if p.int64 == i {
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
func (p *Bin) Strval(ctx Context) string { return strconv.FormatInt(int64(p.int64),2) }
func (p *Bin) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Bin) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Bin) expand(_ Context, _ facet) Value { return p }

type Oct struct { integer }
func (p *Oct) String() string { return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8)) }
func (p *Oct) Strval(ctx Context) string { return strconv.FormatInt(int64(p.int64),8) }
func (p *Oct) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Oct) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Oct) expand(_ Context, _ facet) Value { return p }

type Int struct { integer }
func (p *Int) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) Strval(ctx Context) string { return strconv.FormatInt(int64(p.int64),10) }
func (p *Int) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Int) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Int) expand(_ Context, _ facet) Value { return p }

type Hex struct { integer }
func (p *Hex) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *Hex) Strval(ctx Context) string { return strconv.FormatInt(int64(p.int64),16) }
func (p *Hex) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Hex) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Hex) expand(_ Context, _ facet) Value { return p }

const FloatEpsilon = 1e-15 /* 1e-16 */
type Float struct { valbase; float64 } // IEEE-754 64-bit binary floating-point
func (p *Float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) Strval(ctx Context) string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) True(ctx Context) bool { return math.Abs(p.float64)-0 > FloatEpsilon }
func (p *Float) Integer(ctx Context) (i int64, _ error) { return int64(p.float64), nil }
func (p *Float) Float(ctx Context) (f float64, _ error) { return p.float64, nil }
func (p *Float) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Float) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Float) kind() kind { return valFloat }
func (p *Float) expand(_ Context, _ facet) Value { return p }
func (p *Float) cmp(ctx Context, v Value) (res cmpres) {
    if _, ok := v.(*Float); ok {
        if f, e := v.Float(ctx); e != nil {
            warn(ctx, "%v: %v", v, e).debug(1)
        } else if p.float64 == f {
            res = cmpEqual
        } else if p.float64 < f {
            res = cmpSmaller
        } else if p.float64 > f {
            res = cmpGreater
        }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}

type DateTime struct {
    valbase
    t time.Time
}
func (p *DateTime) String() string { return time.Time(p.t).Format("2006-01-02T15:04:05.999999999Z07:00") }
func (p *DateTime) Strval(ctx Context) string { return p.String() } // time.RFC3339Nano
func (p *DateTime) True(ctx Context) bool { return !p.t.IsZero() }
func (p *DateTime) Integer(ctx Context) (i int64, _ error) { return p.t.Unix(), nil }
func (p *DateTime) Float(ctx Context) (f float64, _ error) { return float64(p.t.Unix()), nil }
func (p *DateTime) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *DateTime) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *DateTime) expand(_ Context, _ facet) Value { return p }
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
func (p *Date) Strval(ctx Context) string { return p.String() }
func (p *Date) Integer(ctx Context) (i int64, _ error) { return p.t.Unix(), nil }
func (p *Date) Float(ctx Context) (f float64, _ error) { return float64(p.t.Unix()), nil }
func (p *Date) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Date) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Date) expand(_ Context, _ facet) Value { return p }

type Time struct { DateTime }
func (p *Time) String() string { return time.Time(p.t).Format("15:04:05.999999999Z07:00") }
func (p *Time) Strval(ctx Context) string { return p.String() }
func (p *Time) Integer(ctx Context) (i int64, _ error) { return p.t.Unix(), nil }
func (p *Time) Float(ctx Context) (f float64, _ error) { return float64(p.t.Unix()), nil }
func (p *Time) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return p._match(ctx, p, i) }
func (p *Time) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Time) expand(_ Context, _ facet) Value { return p }

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
func (p *URL) True(ctx Context) (t bool) {
    if p.Scheme != nil { if t = p.Scheme.True(ctx); t { return }}
    if p.Host   != nil { if t = p.Host  .True(ctx); t { return }}
    if p.Path   != nil { if t = p.Path  .True(ctx); t { return }}
    return //p.String() != "", nil
}
func (p *URL) Strval(ctx Context) (s string) {
    if s = p.Scheme.Strval(ctx) + ":"; p.Host != nil && !isNone(p.Host) {
        s += "//"
        if p.Username != nil && !isNone(p.Username) {
            s += p.Username.Strval(ctx)
            if p.Password != nil {
                s += ":" + p.Password.Strval(ctx)
            }
            s += "@"
        }
        s += p.Host.Strval(ctx)
        if p.Port != nil && !isNone(p.Port) {
            s += ":" + p.Port.Strval(ctx)
        }
    }
    if p.Path != nil && !isNone(p.Path) {
        //if !strings.HasPrefix(path, PathSep) { s += PathSep }
        s += p.Path.Strval(ctx)
    }
    if p.Query != nil && !isNone(p.Query) {
        s += "?" + p.Query.Strval(ctx)
    }
    if p.Fragment != nil && !isNone(p.Fragment) {
        s += "#" + p.Fragment.Strval(ctx)
    }
    return
}
func (p *URL) Integer(ctx Context) (i int64, _ error) {
    if s := p.Strval(ctx); s != "" { i = int64(len(s)) }
    return
}
func (p *URL) Float(ctx Context) (f float64, e error) { i, e := p.Integer(ctx); return float64(i), e }
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
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}
func (p *URL) expand(ctx Context, w facet) (res Value) {
    var o = &URL{ valbase: p.valbase }
    if nil != p.Scheme   { o.Scheme   = p.Scheme.expand(ctx, w) }
    if nil != p.Username { o.Username = p.Username.expand(ctx, w) }
    if nil != p.Password { o.Password = p.Password.expand(ctx, w) }
    if nil != p.Host     { o.Host     = p.Host.expand(ctx, w) }
    if nil != p.Port     { o.Port     = p.Port.expand(ctx, w) }
    if nil != p.Path     { o.Path     = p.Path.expand(ctx, w) }
    if nil != p.Query    { o.Query    = p.Query.expand(ctx, w) }
    if nil != p.Fragment { o.Fragment = p.Fragment.expand(ctx, w) }
    if  o.Scheme != p.Scheme ||
        o.Username != p.Username ||
        o.Password != p.Password ||
        o.Host != p.Host ||
        o.Port != p.Port ||
        o.Path != p.Path ||
        o.Query != p.Query ||
        o.Fragment != p.Fragment {
        res = o
    } else {
        res = p
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
func (p *Raw) Strval(ctx Context) string { return p.string }
func (p *Raw) True(ctx Context) bool { return p.string != "" }
func (p *Raw) Integer(ctx Context) (i int64, err error) {
    return strconv.ParseInt(p.string, 10, 64)
}
func (p *Raw) Float(ctx Context) (f float64, err error) {
    return strconv.ParseFloat(p.string, 64)
}
func (p *Raw) expand(_ Context, _ facet) Value { return p }
func (p *Raw) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Raw); ok {
        if p.string == a.string { res = cmpEqual }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
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
func (p *String) Strval(ctx Context) (s string) {
    if false {
        s = strings.Replace(p.string, "\\\"", "\"", -1)
    } else {
        s = p.string
    }
    return
}
func (p *String) True(ctx Context) (v bool) {
    switch p.string {
    case "no", "false": v = false
    default: v = p.string != ""
    }
    return
}
func (p *String) Integer(ctx Context) (i int64, err error) {
    return strconv.ParseInt(p.string, 10, 64)
}
func (p *String) Float(ctx Context) (f float64, err error) {
    return strconv.ParseFloat(p.string, 64)
}
func (p *String) expand(_ Context, _ facet) Value { return p }
func (p *String) cmp(ctx Context, v Value) (res cmpres) {
    switch t := v.(type) {
    case *Group:
    case *List: if len(t.Elems) == 1 { res = p.cmp(ctx, t.Elems[0]) }
    case unexpanded: if t.Value != nil { res = p.cmp(ctx, t.Value) }
    default: if s := t.Strval(ctx); p.string == s {
            res = cmpEqual
        } else if p.string < s {
            res = cmpSmaller
        } else /*if p.string > s*/ {
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
func (p *String) traverse(ctx Context) (traves travestates) {
    return traverse(at(ctx, p.position), p, p.string)
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
func (p *Punctuation) Strval(ctx Context) string { return p.tok.String() }
func (p *Punctuation) True(ctx Context) bool { return false }
func (p *Punctuation) Integer(ctx Context) (i int64, _ error) { return 0, nil }
func (p *Punctuation) Float(ctx Context) (f float64, _ error) { return 0, nil }
func (p *Punctuation) expand(_ Context, _ facet) Value { return p }
func (p *Punctuation) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Punctuation); ok {
        if p.tok == a.tok {
            res = cmpEqual
        } else if p.tok > a.tok {
            res = cmpSmaller
        } else if p.tok < a.tok {
            res = cmpGreater
        }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}
func (p *Punctuation) match(ctx Context, i interface{}) (full bool, s string, stems []string) { return }
func (p *Punctuation) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Punctuation) traverse(ctx Context) (traves travestates) { return }

type Bareword struct { valbase; string }
func (p *Bareword) String() string { return p.string }
func (p *Bareword) Strval(ctx Context) string { return p.string }
func (p *Bareword) True(ctx Context) bool { return isTrueString(p.string) }
func (p *Bareword) Integer(ctx Context) (i int64, err error) {
    return strconv.ParseInt(p.string, 10, 64)
}
func (p *Bareword) Float(ctx Context) (f float64, err error) {
    return strconv.ParseFloat(p.string, 64)
}
func (p *Bareword) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Bareword); ok {
        if p.string == a.string {
            res = cmpEqual
        } else if p.string < a.string {
            if strings.HasPrefix(a.string, p.string) {
                res = cmpLPrefix
            } else {
                res = cmpSmaller
            }
        } else {
            if strings.HasPrefix(p.string, a.string) {
                res = cmpRPrefix
            } else {
                res = cmpGreater
            }
        }
    } else if c, ok := v.(*Barecomp); ok {
        if s := c.Strval(ctx); p.string == s {
            res = cmpEqual
        } else if p.string < s {
            if strings.HasPrefix(s, p.string) {
                res = cmpLPrefix
            } else {
                res = cmpSmaller
            }
        } else {
            if strings.HasPrefix(p.string, s) {
                res = cmpRPrefix
            } else {
                res = cmpGreater
            }
        }
    } else if l, ok := v.(*Path); ok {
        if len(l.Elems) == 1 { res = p.cmp(ctx, l.Elems[0]) }
    } else if l, ok := v.(*List); ok {
        if len(l.Elems) == 1 { res = p.cmp(ctx, l.Elems[0]) }
    } else if u, ok := v.(unexpanded); ok && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if false {
        // NOTE: find the only valid element (if others are none)
        for _, elem := range l.Elems {
            if isNone(elem) { continue }
            if v == l { v = elem }
        }
        if v != l { res = p.cmp(ctx, v) }
    }
    return
}
func (p *Bareword) expand(ctx Context, w facet) (res Value) {
    if res = p; false && w&expandFullName != 0 {
        if file := ctx.Project().FindFile(ctx, p.Strval(ctx)); file != nil {
            res = file.expand(ctx, w)
        }
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
func (p *Bareword) traverse(ctx Context) (traves travestates) {
    return traverse(at(ctx, p.position), p, p.string)
}

type Qualiword struct { valbase; words []string } // foo.bar.zar, foo.&(bar).zar
func (p *Qualiword) String() string { return strings.Join(p.words,".") }
func (p *Qualiword) Strval(ctx Context) string { return p.String() }
func (p *Qualiword) True(ctx Context) bool { return len(p.words)!=0 }
func (p *Qualiword) Integer(ctx Context) (i int64, _ error) { return int64(len(p.words)), nil }
func (p *Qualiword) Float(ctx Context) (f float64, _ error) { return 0, nil }
func (p *Qualiword) expand(_ Context, _ facet) Value { return p }
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
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
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
func (p *Qualiword) traverse(ctx Context) (traves travestates) {
    return traverse(at(ctx, p.position), p, p.Strval(ctx))
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
func (p *elements) True(ctx Context) (t bool) { // (or elems...)
    for _, elem := range p.Elems {
        if elem != nil { if t = elem.True(ctx); t { break }}
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
func (p *elements) defs(ctx Context, s ...string) (res []*def) {
    for _, elem := range p.Elems { res = append(res, elem.defs(ctx, s...)...) }
    return
}
func (p *elements) expandible(ctx Context, w facet) (res bool) {
    var pos = ctx.Position()
    for _, elem := range p.Elems {
        if elem == nil {
            warnstack(at(ctx,pos), 6, "nil element: %v", p).debug(32)
            break
        } else {
            pos = elem.Position()
        }
        if res = elem.expandible(ctx, w); res { break }
    }
    return
}

func compareElems(ctx Context, elemsL, elemsR []Value) (res cmpres) {
    if len(elemsL) != len(elemsR) {
        elemsL = merge(elemsL...)
        elemsR = merge(elemsR...)
    }
    if a, b := len(elemsL), len(elemsR); a == b {
        for i, elem := range elemsL {
            var other = elemsR[i]
            if a, b := (elem == nil), (other == nil); a && b {
                continue
            } else if a || b {
                return
            }
            if res = elem.cmp(ctx, other); res != cmpEqual {
                if options.debug && elem.String() == other.String() {
                    warn(ctx, "L.%d: %T %v", i, elem, elem)
                    warn(ctx, "R.%d: %T %v", i, other, other)
                    if b1, y := elem.(*Barecomp); y {
                        if b2, y := other.(*Barecomp); y {
                            warn(ctx, "%v", b1.Elems)
                            warn(ctx, "%v", b2.Elems)
                        }
                    }
                    warn(ctx, "cmp: %v", res).debug(1)
                }
                return
            }
        }
        if res == 0 { res = cmpEqual }
    } else {
        var notEmptyL, notEmptyR bool
        for _, elem := range elemsL {
            if notEmptyL = !(isNone(elem) || isNil(elem)); notEmptyL { break }
        }
        for _, elem := range elemsR {
            if notEmptyR = !(isNone(elem) || isNil(elem)); notEmptyR { break }
        }
        if !notEmptyL && !notEmptyR { return cmpEqual }

        // TODO: list.cmp: cmpSmaller, cmpGreater
    }
    return
}

type paircomp struct { *Pair }
func (p paircomp) True(ctx Context) (res bool) {
    if !(p.Key.expandible(ctx, expandDelegate|expandClosure) ||
        p.Value.expandible(ctx, expandDelegate|expandClosure)) {
         res = p.Pair.True(ctx)
    }
    return
}
func (p paircomp) String() (s string) {
    return p.Pair.String()
}
func (p paircomp) Strval(ctx Context) (s string) {
    if !(p.Key.expandible(ctx, expandDelegate|expandClosure) ||
        p.Value.expandible(ctx, expandDelegate|expandClosure)) {
        s = p.Pair.Strval(ctx)
    }
    return
}
func (p paircomp) expand(ctx Context, w facet) (res Value) {
    var a = p.Pair.expand(ctx, w)
    if res = p; a != p.Pair { res = paircomp{a.(*Pair)} }
    return
}

type precomp struct {
    Value
    suffix Value
}
func (p precomp) True(ctx Context) bool { return p.suffix.True(ctx) }
func (p precomp) String() (s string) {
    return p.Value.String() + p.suffix.String()
}
func (p precomp) Strval(ctx Context) (s string) {
    if t := p.suffix.Strval(ctx); t != "" { s = p.Value.Strval(ctx) + t }
    return
}
func (p precomp) expand(ctx Context, w facet) (res Value) {
    var a, b = p.Value.expand(ctx, w), p.suffix.expand(ctx, w)
    if res = p; a != p.Value || b != p.suffix { res = precomp{a, b} }
    return
}
// func (p precomp) cmp(ctx Context, v Value) (res cmpres) {
//     return
// }

type rearcomp struct {
    prefix Value
    Value
}
func (p rearcomp) True(ctx Context) bool { return p.prefix.True(ctx) }
func (p rearcomp) String() (s string) {
    return p.prefix.String() + p.Value.String()
}
func (p rearcomp) Strval(ctx Context) (s string) {
    if t := p.prefix.Strval(ctx); t != "" { s = t + p.Value.Strval(ctx) }
    return
}
func (p rearcomp) expand(ctx Context, w facet) (res Value) {
    var a, b = p.prefix.expand(ctx, w), p.Value.expand(ctx, w)
    if res = p; a != p.Value || b != p.prefix { res = rearcomp{a, b} }
    return
}
// func (p rearcomp) cmp(ctx Context, v Value) (res cmpres) {
//     return
// }

type Barecomp struct { valbase ; elements }
func (_ *Barecomp) kind() kind { return valOther }
func (p *Barecomp) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *Barecomp) Strval(ctx Context) (s string) {
    for _, elem := range p.Elems {
        if isTrivial(elem) { continue } else {
            s += elem.Strval(ctx)
        }
    }
    return
}
func (p *Barecomp) True(ctx Context) bool { return p.elements.True(ctx) }
func (p *Barecomp) Integer(ctx Context) (res int64, _ error) {
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
func (p *Barecomp) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *Barecomp) elemStr(ctx Context, o Object, k elemkind) (s string) {
    for _, elem := range p.Elems {
        s += elementString(ctx, o, elem, k)
    }
    return
}
func (p *Barecomp) expandible(ctx Context, w facet) bool {
    return p.elements.expandible(ctx, w)
}
func (p *Barecomp) expand(ctx Context, w facet) (res Value) {
    var elems, u, n = w.expand(ctx, p.Elems...)
    if n > 0 { res = &Barecomp{p.valbase, elements{elems}} } else { res = p }
    if u > 0 { res = unexpanded{res} }
    return
}
func (p *Barecomp) traverse(ctx Context) (traves travestates) {
    return traverse(at(ctx, p.Position()), p, p.Strval(ctx))
}
func (p *Barecomp) cmp(ctx Context, v Value) (res cmpres) {
    var cmp = func(elemsL, elemsR []Value) {
        if res = compareElems(ctx, elemsL, elemsR); res != cmpEqual {
            var i int
            for ; i < len(elemsL) && i < len(elemsR); i += 1 {
                var cr = elemsL[i].cmp(ctx, elemsR[i])
                if cr == cmpEqual { continue } else
                if cr == cmpLPrefix || cr == cmpRPrefix {
                    if false && p.Strval(ctx) == v.Strval(ctx) { res = cmpEqual ; return }
                    if true  && p.String(   ) == v.String(   ) { res = cmpEqual ; return }
                } else {
                    break
                }
            }
            if i < len(elemsL) && i == len(elemsR) {
                for ; i < len(elemsL); i += 1 {
                    if !isTrivial(elemsL[i]) { return }
                }
                res = cmpEqual ; return
            }
            if i == len(elemsL) && i < len(elemsR) {
                for ; i < len(elemsR); i += 1 {
                    if !isTrivial(elemsR[i]) { return }
                }
                res = cmpEqual ; return
            }
        }
    }

    if a, ok := v.(*Barecomp); ok {
        cmp(mergeBare(p.Elems...), mergeBare(a.Elems...))
    } else if a, ok := v.(precomp); ok {
        cmp(mergeBare(p.Elems...), mergeBare(a.Value, a.suffix))
    } else if a, ok := v.(rearcomp); ok {
        cmp(mergeBare(p.Elems...), mergeBare(a.prefix, a.Value))
    } else if w, ok := v.(*Bareword); ok {
        if s := p.Strval(ctx); s == w.string {
            res = cmpEqual
        } else if s < w.string {
            res = cmpSmaller
        } else {
            res = cmpGreater
        }
    } else if fR, ok := v.(*Flag); ok {
        var elems []Value
        for _, elem := range p.Elems {
            if !isNil(elem) && !isNone(elem) {
                elems = append(elems, elem)
            }
        }
        if len(elems) == 2 { if fL, ok := elems[0].(*Flag); ok {
            if isNil(fL.name) || isNone(fL.name) {
                res = elems[1].cmp(ctx, fR.name)
            } else if m, s, t := fL.name.match(ctx, fR.name); m {
                if isNil(elems[1]) || isNone(elems[1]) { res = cmpEqual }
            } else if s != "" { // matched prefix
                var sL = s + elems[1].Strval(ctx)
                var sR = fR.name.Strval(ctx)

                if sL == sR {
                    res = cmpEqual
                } else if s < sR {
                    res = cmpSmaller
                } else {
                    res = cmpGreater
                }
            } else if t != nil {
                unreachable(p, v)
            }
        }}
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if true && l != nil && len(p.Elems)==2 && len(l.Elems)>1 {
        if pl, ok := p.Elems[1].(*List); ok && len(pl.Elems)>1 {
            var a = p.Elems[1] // Example: p.Elems=[a- a b c], l.Elems=[z-a b c]
            // FIXME: avoid 'p.Elems[1] = pl.Elems[0]', the container values are readonly for cmp
            if p.Elems[1] = pl.Elems[0]; p.cmp(ctx, l.Elems[0]) == cmpEqual {
                res = compareElems(ctx, pl.Elems[1:], l.Elems[1:])
            }
            p.Elems[1] = a
        }
    }
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
        var (
            n int
            is string
            elem Value
        )
        switch t := i.(type) {
        case string: is = t
        case Value : is = t.Strval(ctx)
        default:
            erro(of(ctx,p), "%T: matching unsupported value: %T %v", p, i, i).debug(1)
            return
        }
        if is == "" { return }
        for n, elem = range p.Elems {
            var m, t, ss = elem.match(ctx, is)
            if false && elem.String() == "$(name)" {
                warn(ctx, "%T %v %s ; %v -> %v '%v' %v", elem, elem, elem.Strval(ctx), is, m, t, ss).debug(1)
            }
            if t == "" { break } else {
                stems = append(stems, ss...)
                is = is[len(t):]
                s += t
            }
        }
        if is == "" && n == len(p.Elems)-1 { full = true }
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
type Barefile struct { // TODO: remove this type
    valbase
    Name Value
    File *File
}
func (p *Barefile) True(ctx Context) (t bool) {
    if p.File != nil { t = p.File.True(ctx) }
    return
}
func (p *Barefile) String() string { return p.elemStr(nil, nil, 0) }
func (p *Barefile) Strval(ctx Context) string {
    if p.File != nil {
        return p.File.Strval(ctx)
    } else {
        return p.Name.Strval(ctx)
    }
}
func (p *Barefile) Integer(ctx Context) (res int64, _ error) {
    if p.File.exists() { res = p.File.info.Size() }
    return
}
func (p *Barefile) Float(ctx Context) (f float64, _ error) {
    i, e := p.Integer(ctx); return float64(i), e
}
func (p *Barefile) refs(ctx Context, v Value) bool { return p.Name.refs(ctx, v) }
func (p *Barefile) defs(ctx Context, s ...string) []*def { return p.Name.defs(ctx, s...) }
func (p *Barefile) elemStr(ctx Context, o Object, k elemkind) (s string) { return elementString(ctx, o, p.Name, k) }
func (p *Barefile) expandible(ctx Context, w facet) bool { return p.Name.expandible(ctx, w) }
func (p *Barefile) expand(ctx Context, w facet) (res Value) {
    var name Value

    if w&expandFullName != 0 {
        var file = p.File
        if file == nil { file = ctx.Project().FindFile(ctx, p.Name.Strval(ctx)) }
        if file != nil { if name = file.expand(ctx, w); name != file {
            return name
        }}
    }

    if name = p.Name.expand(ctx, w); name != p.Name {
        res = &Barefile{p.valbase, name, p.File}
    } else {
        res = p
    }
    return
}
func (p *Barefile) traverse(ctx Context) (traves travestates) {
    return traverse(at(ctx, p.position), p, p.Strval(ctx))
}
func (p *Barefile) updated(ctx Context, v ...bool) (res bool) {
    if p.File != nil { res = p.File.updated(ctx, v...) }
    return
}
func (p *Barefile) updatedDeps(ctx Context, v ...Value) (res []Value) {
    if p.File != nil { res = p.File.updatedDeps(ctx, v...) }
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
    if a, ok := v.(*Barefile); ok {
        res = p.Name.cmp(ctx, a.Name)
    } else if a, ok := toFile(v); ok && p.File != nil {
        res = p.File.cmp(ctx, a)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
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

func barefilize(ctx Context, targets ...Value) []Value {
    var project = ctx.Project()
    for i, target := range targets {
        if target.patterned(ctx) { continue }
        switch t := target.(type) {
        case *Bareword:
            if file := project.FindFile(ctx, t.string); file != nil {
                var pos = target.Position()
                targets[i] = &Barefile{ valbase{pos}, target, file }
                file.position = pos
            }
        case *Barecomp, *Path:
            if t.patterned(ctx) || t.expandible(ctx, expandClosure) || refdef(ctx, t, DefArg) {
                break
            } else if file := project.FindFile(ctx, t.Strval(ctx)); file != nil {
                var pos = target.Position()
                targets[i] = &Barefile{ valbase{pos}, target, file }
                file.position = pos
            }
        case *Argumented:
            t.value = barefilize(ctx, t.value)[0]
            t.args  = barefilize(ctx, t.args...)
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
        erro(of(ctx,patVal), "invalid glob matching pattern: %v", patVal).debug(8)
        return
    }

    var patList = strings.Split(filepath.Clean(patVal.Strval(ctx)), PathSep)
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
func (p *GlobMeta) Strval(ctx Context) string { return p.Token.String() }
func (p *GlobMeta) expand(_ Context, _ facet) Value { return p }
func (p *GlobMeta) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*GlobMeta); ok {
        if p.Token == a.Token { res = cmpEqual }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}

// `[a-b]`, `[abc]`, ...
// `a-b`, `abc`, `a$(var)c`, `a$(spaces)c`...
type GlobRange struct { valbase ; Chars Value }
func (p *GlobRange) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *GlobRange) Strval(ctx Context) (s string) {
    return fmt.Sprintf("[%s]", p.Chars.Strval(ctx))
}
func (p *GlobRange) refs(ctx Context, v Value) bool { return p.Chars.refs(ctx, v) }
func (p *GlobRange) defs(ctx Context, s ...string) []*def { return p.Chars.defs(ctx, s...) }
func (p *GlobRange) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*GlobRange); ok {
        res = p.Chars.cmp(ctx, a.Chars)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}
func (p *GlobRange) expandible(ctx Context, w facet) bool { return p.Chars.expandible(ctx, w) }
func (p *GlobRange) expand(ctx Context, w facet) (res Value) {
    if val := p.Chars.expand(ctx, w); val != p.Chars {
        res = &GlobRange{p.valbase, val}
    } else {
        res = p
    }
    return
}
func (p *GlobRange) elemStr(ctx Context, o Object, k elemkind) (s string) {
    return fmt.Sprintf("[%s]", elementString(ctx, o, p.Chars, k))
}

// Path is addressing a file (dynamically), the real located file varies
// base on 'elements' and the context.
type Path struct { valbase ; elements }
func (_ *Path) kind() kind { return valOther }
func (p *Path) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *Path) Strval(ctx Context) (s string) {
    for i, seg := range p.Elems {
        if seg == nil {
            erro(ctx, "`%s` nil path segment", p).debug(1)
            return
        }

        var v string
        if isUndef(seg) {
            erro(at(ctx,seg.Position()), "undef path segment (%T)", seg)
            erro(at(ctx,ctx.Position()), "… from this context: %s", ctx).debug(16)
            return
        } else if v = seg.Strval(ctx); i > 0 {
            s += PathSep + v
        } else if v != "" {
            s += v
        } else if len(p.Elems) == 1 {
            s += PathSep
        }
    }
    return
}
func (p *Path) True(ctx Context) (t bool) {
    // FIXME: return p.exists() ??
    for _, elem := range p.Elems {
        if t = elem.True(ctx); t { break }
    }
    return
}
func (p *Path) refs(ctx Context, v Value) (res bool) { return p.elements.refs(ctx, v) }
func (p *Path) defs(ctx Context, s ...string) (res []*def) { return p.elements.defs(ctx, s...) }
func (p *Path) expandible(ctx Context, w facet) bool { return p.elements.expandible(ctx, w) }
func (p *Path) expand(ctx Context, w facet) (res Value) {
    var elems, u, n = expandPathElems(ctx, w, p.Elems...)
    if n > 0 { res = &Path{p.valbase, elements{elems}} } else { res = p }
    if u > 0 { res = unexpanded{res} }
    return
}
func (p *Path) delete(ctx Context) (files []*File, err error) {
    var pathname string
    if positionalValueCtx { ctx = at(ctx, p.position) }
    if pathname = p.Strval(ctx); pathname == "" {
        erro(ctx, "no pathname for `%s`", p)
    } else if file := stat(ctx,pathname,"","",nil); file != nil {
        if files, err = file.delete(ctx); err != nil {
            erro(ctx, "stamp: %v (%v)", err, file)
        }
    }
    return
}
func (p *Path) stamp(ctx Context) (files []*File, err error) {
    var pathname string
    if positionalValueCtx { ctx = at(ctx, p.position) }
    if pathname = p.Strval(ctx); pathname == "" {
        erro(ctx, "no pathname for `%s`", p)
    } else if file := stat(ctx,pathname,"","",nil); file != nil {
        if files, err = file.stamp(ctx); err != nil {
            erro(ctx, "stamp: %v (%v)", err, file)
        } else {
            return
        }
    }

    //p.updated(ctx, true)
    //ctx.dirtyMark(p)
    return
}
func (p *Path) stat(ctx Context) (si *statinfo) {
    ctx = at(ctx, p.position)

    var file *File
    if p.patterned(ctx) {
        if val, rest := p.stencil(ctx, ctx.stems()); len(rest) > 0 {
            erro(ctx, "partial match: %v", rest)
            return
        } else {
            file = stat(ctx, val.Strval(ctx), "", "", nil)
        }
    } else {
        file = stat(ctx, p.Strval(ctx), "", "", nil)
    }

    if file != nil { si = &statinfo{ file:file }}
    return
}
func (p *Path) traverse(ctx Context) (traves travestates) {
    return traverse(at(ctx, p.position), p, "")
}
func (p *Path) patterned(ctx Context) (result bool) {
    for _, seg := range p.Elems {
        if result = seg.patterned(ctx); result { break }
    }
    return
}
func (p *Path) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Path); ok {
        res = compareElems(ctx, p.Elems, a.Elems)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
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

func expandPathElems(ctx Context, w facet, elems ...Value) (res []Value, u, n int) {
    var xelems []Value
    xelems, u, n = w.expand(ctx, elems...)
    for _, elem := range xelems {
        if p, ok := elem.(*Path); ok {
            var ev, u1, n1 = expandPathElems(ctx, w, p.Elems...)
            res = append(res, ev...)
            u += u1
            n += n1
        } else {
            res = append(res, elem)
        }
    }
    if w&expandPathStr != 0 {
        var vals []Value
        for _, elem := range res {
            switch v := elem.(type) {
            case *String:
                if v.string != "" {
                    vals = append(vals, splitPathStr(v.position, v.string)...)
                }
                n += 1
            case *Compound:
                if s := v.Strval(ctx); s != "" {
                    vals = append(vals, splitPathStr(v.position, s)...)
                }
                n += 1
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
        un int
    )
    if srcs = strings.Split(str, PathSep); len(srcs) == 0 {
        if false { erro(at(ctx,p.position), "empty: %v", str) }
        return
    } else if segs, un, _ = expandPathElems(ctx, plain, p.Elems...); un > 0 {
        errostack(ctx, 3, "can't expand path: %v", p).debug(1)
        return
    }

    const warns = false
    //var warns = p.String() == "llvm/%%"

    var (
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
            erro(of(ctx,seg), "invalid path seg: %v (%T)", seg, seg).debug(1)
            break SegsSrcsLoop
        }

        var (
            pp, pre, suf = percperc(seg)
            ss []string
            s    string
            f    bool
        )
        if pp {
            if infos { info(of(ctx,p), "%d: path=%v seg=%v (%T) src=%v pp=%v pre=%v suf=%v res=%v stems=%v srcs[%d]=%s lenSegs=%d", si, p, seg, seg, src, pp, pre, suf, res, stems, m, src, lenSegs).debug(1) }
            if !isTrivial(lastSuf) && !isTrivial(pre) {
                erro(of(ctx,seg), "the continual %%/%% makes no sense")
                break SegsSrcsLoop
            }
            if /*ps, ok := seg.(*PathSeg);*/ src == "" /*&& ok && (ps.rune == 0 || ps.rune == '/')*/ { // for root path seg '/'
                // NOTE: seg could also be % or %% here
                res   = append(res, "") // for '/'
                stems = append(stems, "")
                //info(of(ctx,p), "%v %v; %v %v; %v %v", p, seg, res, stems, ps, ok)
            }
            lastSuf = suf
            n += 1 // move forward to the next seg
            m += 1 // move forward to the next src
        } else if f, s, ss = seg.match(ctx, src); f || s == src {
            if infos { info(of(ctx,p), "%d: path=%v seg=%v (%T); str=%v srcs[%d]=%v -> f=%v s=%v ss=%v => res=%v stems=%v", si, p, seg, seg, str, m, src, f, s, ss, res, stems).debug(1) }
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
            if infos { info(of(ctx,p), "%d: path=%v seg=%v (%T) res=%v stems=%v f=%v s=%s ss=%v src=%s lenSegs=%d str=%s", si, p, seg, seg, res, stems, f, s, ss, src, lenSegs, str).debug(1) }
            break SegsSrcsLoop
        }

        var prefix string
        if !isTrivial(pre) {
            if prefix = pre.Strval(ctx); !strings.HasPrefix(src, prefix) {
                if infos { info(of(ctx,p), "%d: seg=%v (%T) pp=%v pre=%v suf=%v res=%v stems=%v src=%s", si, seg, seg, pp, pre, suf, res, stems, src).debug(1) }
                break SegsSrcsLoop
            }
        }

        // Iterate segs for %%, e.g. bar, baz in foo/%%/bar/baz
        var stem []string
        if prefix != "" { stem = append(stem, strings.TrimPrefix(src, prefix)) }
        if !isTrivial(suf) {
            var suffix = suf.Strval(ctx)
            if res = append(res, src); m < lenSrcs {
                for stem = append(stem, src); m < lenSrcs; m += 1 {
                    src = srcs[m]
                    res = append(res, src)
                    if strings.HasSuffix(src, suffix) {
                        stem = append(stem, strings.TrimSuffix(src, suffix))
                        stems = append(stems, strings.Join(stem, PathSep))
                        if infos { info(of(ctx,p), "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d", si, p, seg, seg, res, stems, suffix, src, lenSegs).debug(1) }
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
                if infos { info(of(ctx,p), "%d: path=%v seg=%v (%T) res=%v stems=%v suffix=%v src=%s lenSegs=%d lenSrcs=%d m=%d", si, p, seg, seg, res, stems, suffix, src, lenSegs, lenSrcs, m).debug(1) }
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
                    if infos { info(of(ctx,p), "%d: path=%v seg=%v (%T) res=%v stems=%v matched=%v s=%s ss=%v src=%s lenSegs=%d", si, p, seg, seg, res, stems, matched, s, ss, src, lenSegs).debug(1) }
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
                if infos { info(of(ctx,p), "%d: path=%v seg=%v (%T) res=%v stems=%v s=%s ss=%v src=%s lenSegs=%d", si, p, seg, seg, res, stems, s, ss, src, lenSegs).debug(1) }
            }
        } else { // the tailing %%
            if /*m < lenSrcs*/true {
                var rest = srcs[m-1:]
                res = append(res, rest...)
                stem = append(stem, rest...)
                stems = append(stems, strings.Join(stem, PathSep))
            }
            if infos { info(of(ctx,p), "%d: seg=%v pre=%v suf=%v res=%v stems=%v src=%s", si, seg, pre, suf, res, stems, src).debug(1) }
            break SegsSrcsLoop
        }
        if infos && n == lenSegs {
            info(of(ctx,p), "%d: path=%v seg=%v (%T) str=%v src=%v -> f=%v s=%v ss=%v -> res=%v stems=%v stem=%v m=%d/%d 3.lengSegs=%d", si, p, seg, seg, str, src, f, s, ss, res, stems, stem, m, lenSrcs, lenSegs).debug(true, 1)
        }
    }
    if lenRes := len(res); lenRes > 0 { // full or partial matched
        //TODO: if n < lenSegs { rest = strings.Join(segs[n:], PathSep) }
        result = strings.Join(res, PathSep) //NOTE: do NOT use `filepath.Join(res...)` here
        full = /* n == lenSegs && */ m <= lenSrcs &&
            lenSrcs == lenRes && lenSegs <= lenRes &&
            result == str
        if false && !full && result == str && str == "..." {
            warn(ctx, "%d/%d, %d/%d, %d: %v (%s)", n, lenSegs, m, lenSrcs, lenRes, res, str)
            warn(ctx, "%d/%d, %d/%d, %d: %v %v", n, lenSegs, m, lenSrcs, lenRes, srcs, segs).debug(1)
        }
        if infos { if false {
            warn(of(ctx,p), "Path.match: path=%v str=%v res=%v stems=%v -> full=%v result=%v lens=%d,%d", p, str, res, stems, full, result, lenRes, lenSrcs).debug(1)
        } else {
            warn(of(ctx,p), "Path.match: path=%v res=%v stems=%v lenRes=%d", p, res, stems, lenRes)
            warn(of(ctx,p), "Path.match: str=%v full=%v result=%v lenSrcs=%d", str, full, result, lenSrcs).debug(4)
        }}
        if correct := (!full && strings.HasPrefix(str, result)) || (full && str == result); false {
            assert(correct, "incorrect result: res=%v result=%v full=%v stems=%v str=%s", res, result, full, stems, str)
        } else if !correct {
            prompt(ctx, "%v: %v: incorrect match: full=%v; segs=%v; srcs=%v; res=%v\n", str, p, full, segs, srcs, res)
            erro(of(ctx,p), "incorrect match: path=%v, str=%s, res=%v result=%v", p, str, res, result)
            errostack(ctx, 8, "%v", ctx).debug(10)
        }
        if full && p.patterned(ctx) && len(stems) == 0 {
            if lenSegs == 1 && lenSrcs == 1 && len(res) == 1 && segs[0].patterned(ctx) {
                stems = res
            } else {
                prompt(ctx, "%v: %v: incorrect full match: segs=%v; srcs=%v; res=%v\n", str, p, segs, srcs, res)
                warn(of(ctx,p), "incorrect full match: path=%v, str=%s, res=%v result=%v", p, str, res, result)
                warnstack(ctx, 3, "(%T):", ctx).debug(6)
            }
        }
    }
    return
}
func (p *Path) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    ctx = at(ctx, p.position)
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
        if str := t.Strval(ctx); str != "" {
            return p.match1(ctx, str)
        }
    default:
        erro(at(ctx,p.position), "matching unsupport value: %T %v", i, i).debug(8)
    }
    return
}

func (p *Path) stencil(ctx Context, stems []string) (result Value, rest []string) {
    var (
        elems []Value
        changed int
    )
    for _, seg := range mergex(ctx, plain, p.Elems...) {
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
        erro(of(ctx,p), "path combines invalid value: %v", val)
        return
    } else if isNil(tail) {
        erro(of(ctx,p), "path tail is nil: %v", p)
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
func (p *PathSeg) Strval(ctx Context) (s string) {
    if p.rune == 0 { // zero segment,
        s = "" // the 'empty' after the last '/'
    } else if p.rune == '/' { // root segment,
        s = "" // the 'empty' before the first '/'
    } else if s = p.String(); s == "" {
        erro(at(ctx,p.position), "unknown segment '%s' ('%v')", string(p.rune), p).debug(1)
    }
    return
}
func (p *PathSeg) expand(_ Context, _ facet) Value { return p }
func (p *PathSeg) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*PathSeg); ok {
        if p.rune == a.rune { res = cmpEqual }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}
func (p *PathSeg) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    var s string
    switch t := i.(type) {
    case string: s = t
    case Value: s = t.Strval(ctx)
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
        base *filebase
        stub *filestub
        fullname string
    )

    statmutex.Lock(); defer statmutex.Unlock()

    // Trims / suffix
    if dir != "" { dir = filepath.Clean(dir) }
    if sub != "" { sub = filepath.Clean(sub) }
    if false {
        var t = strings.HasPrefix(name, "./")
        if name!= "" { name = filepath.Clean(name) }
        if t         { name = "./" + name }
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

func toFile(v Value) (f *File, y bool) {
    if f, y = v.(*File); !y {
        if t, ok := v.(fullfile); ok { f, y = t.File, ok }
    }
    return
}

func splitFileName(ctx Context, val Value) (dir, name string) {
    var f  *File
    switch t := val.(type) {
    case fullfile: f = t.File
    case    *File: f = f
    }
    if f != nil {
        dir, name = filepath.Join(f.dir, f.sub), f.name
    } else {
        name = val.Strval(ctx)
        dir = filepath.Dir(name)
    }
    return
}

type fullfile struct { *File }
// func (u fullfile) String() string { return u.fullname() }
func (u fullfile) Strval(ctx Context) (s string) { return u.fullname() }
func (u fullfile) expand(ctx Context, w facet) Value {
    if w != expandZero && (w&expandFullName == 0 /* || filepath.IsAbs(u.name) */) {
        if v := u.File.expand(ctx, w); v != u.File {
            if f, y := v.(*File); y && f != u.File { return fullfile{f} }
        }
    }
    return u
}

type File struct {
    valbase
    *filebase
    *filestub
}
func (p *File) String() string { return p.name }
func (p *File) Strval(ctx Context) (s string) { return p.name }
func (p *File) True(ctx Context) (t bool) {
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
func (p *File) absolute_delete(ctx Context) (files []*File, err error) {
    if positionalValueCtx { ctx = at(ctx, p.position) }

    var fullname string
    if fullname = p.fullname(); fullname == "" {
        erro(of(ctx,p), "file `%s` has no fullname", p).debug(1)
        return
    }

    if p.info == nil {
        // ignore
    } else if err = os.Remove(fullname); err != nil {
        erro(at(ctx,p.position), "%v", err).debug(1)
    } else {
        // TODO: ctx.Globe().delete(fullname)
        files = append(files, p)
        p.info = nil
    }
    return
}
func (p *File) stamp(ctx Context) (files []*File, err error) {
    if positionalValueCtx { ctx = at(ctx, p.position) }
    if fullname := p.fullname(); fullname == "" {
        erro(of(ctx,p), "file `%s` has no fullname", p).debug(1)
    } else if p.info, err = os.Stat(fullname); err != nil {
        if false { erro(ctx, "%v", err).debug(1) }
    } else if p.info == nil {
        if false { warn(ctx, "%v: no such file", p).debug(1) }
    } else if files = append(files, p); !ctx.configuration() {
        p.updated(ctx, true)
    }
    return
}
func (p *File) expandible(ctx Context, w facet) (res bool) {
    return w&expandFullName != 0 && !filepath.IsAbs(p.name)
}
func (p *File) expand(ctx Context, w facet) (res Value) {
    if w&expandFullName != 0 && !filepath.IsAbs(p.name) {
        res = fullfile{p}
    } else if /* w&expandFullName != 0 && !filepath.IsAbs(p.name) */false {
        var fullname = p.fullname()
        if false && !filepath.IsAbs(fullname) { return p }

        var stub, fullstub *filestub
        for stub = p.filestub; stub != nil; stub = stub.other {
            if stub.name == fullname /*&& stub.dir == "" && stub.sub == ""*/ {
                fullstub = stub; break
            } else if stub.other == p.filestub {
                break
            }
        }
        if fullstub == nil {
            fullstub = &filestub{ name:fullname, other:stub.other }
            stub.other = fullstub
        }

        res = &File{p.valbase, p.filebase, fullstub}
    } else {
        res = p
    }
    return
}
func (p *File) exists() (res bool) {
    if p != nil && p.filebase != nil {
        res = p.filebase.exists()
    }
    return
}
func (p *File) updated(ctx Context, v ...bool) bool {
    if t := len(v) > 0; t && !p._updated {
        for _, a := range v { t = t && a }
        if p._updated = t; t { ctx.dirtyMark(p) }
    }
    return p._updated
}
func (p *File) updatedDeps(_ Context, v ...Value) []Value {
    if len(v) > 0 { p._updatedDeps = append(p._updatedDeps, v...) }
    return p._updatedDeps
}
func (p *File) stat(ctx Context) (si *statinfo) {
    var err error
    if p.info != nil {
        // good already
    } else if p.info, err = os.Stat(p.fullname()); err == nil {
        // good
    } else if pe, ok := err.(*fs.PathError); ok {
        if false {
            erro(at(ctx,p.position), "File.stat %v: %v", trimPromptString(pe.Path), pe.Err).debug(1)
        }
        return
    } else {
        erro(at(ctx,p.position), "File.stat failed: %v", err).debug(1)
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
func (p *File) traverse(ctx Context) (traves travestates) {
    if p.isSysFile() { return }

    ctx = at(ctx, p.position)

    var projects = ctx.projects(ctx)
    if len(projects) == 0 {
        var targetValue = getTargetValue(ctx)
        prompt(ctx, "%v: zero closure projects\n", p)
        erro(ctx, "no projects to traverse '%v' (%v)", targetValue, p)
        erro(ctx, "%v: closure %v", p, len(ctx.closureScopes()))
        errostack(ctx, 3, "%v: %v", p, ctx).debug(8)
        return
    }

    if p.filemap != nil {
        // Add -import filemap projects! See also `files (-import ...)`
        if proj := p.filemap.project; proj != nil {
            var projs = projects
            for _, p := range projects { if p == proj { goto afterFilemapProject } }
            projects = ctx.projects(ctx, append(projects, proj)...)
            defer func() { ctx.projects(ctx, append([]*Project{nil}, projs...)...) } ()
        afterFilemapProject:
        }
    }

    return traverse(ctx, p, p.name)
}

func (p *File) cmp(ctx Context, v Value) (res cmpres) {
    if isTrivial(v) { return } else
    if a, y := v.(*Barefile); y {
        if a.File != nil { res = p.cmp(ctx, a.File) }
        return
    } else if a, y := toFile(v); y {
        if p.filebase == a.filebase {
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
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}

func (p *File) patterned(ctx Context, ) bool { return false }
func (p *File) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
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
        if !isTrivial(t) {
            return p.match1(ctx, t.Strval(ctx))
        }
    default:
        erro(at(ctx,p.position), "matching file '%v' with unknown input: %v", p, i).debug(1)
    }
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
    }
    return
}

type FileContent struct {
    file *File
    content []byte
}

type Flag struct { valbase ; name Value }
func (p *Flag) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *Flag) Strval(ctx Context) (s string) {
    if p.name == nil {
        s = "-"
    } else if isNone(p.name) {
        s = "-"
    } else {
        s = "-" + p.name.Strval(ctx)
    }
    return
}
func (p *Flag) Integer(ctx Context) (i int64, e error) {
    if i, e = p.name.Integer(ctx); e == nil { i = -i }
    return
}
func (p *Flag) Float(ctx Context) (f float64, e error) {
    if f, e = p.name.Float(ctx); e == nil { f = -f }
    return
}
func (p *Flag) True(ctx Context) (t bool) { return p.name.True(ctx) }
func (p *Flag) refs(ctx Context, v Value) bool { return p.name.refs(ctx, v) }
func (p *Flag) defs(ctx Context, s ...string) []*def { return p.name.defs(ctx, s...) }
func (p *Flag) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    switch t := i.(type) {
    case *None, *Nil, *unresolvedobject:
    case *Flag:
        full, s, stems = p.name.match(ctx, t.name)
        s = "-" + s
    case Value:
        if v := t.Strval(ctx); strings.HasPrefix(v, "-") {
            full, s, stems = p.name.match(ctx, v[1:])
            s = "-" + s
        }
    case string:
        if strings.HasPrefix(t, "-") {
            full, s, stems = p.name.match(ctx, t[1:])
            s = "-" + s
        }
    default:
        warn(at(ctx,p.position), "-%v <-> %T %v", p.name, i, i).debug(true, 16)
    }
    return
}
func (p *Flag) expandible(ctx Context, w facet) bool { return p.name.expandible(ctx, w) }
func (p *Flag) expand(ctx Context, w facet) (res Value) {
    if name := p.name.expand(ctx, w); name != p.name {
        res = &Flag{p.valbase, name}
    } else {
        res = p
    }
    return
}
func (p *Flag) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var name Value
    name, rest = p.name.stencil(ctx, stems)
    if !isNil(name) && name != p.name {
        val = &Flag{p.valbase, name}
    } else {
        rest = stems
    }
    return
}
func (p *Flag) elemStr(ctx Context, o Object, k elemkind) (s string) {
    return "-" + elementString(ctx, o, p.name, k)
}
func (p *Flag) opt(ctx Context, name string) (res string, match bool) {
    if isTrivial(p.name) {
        if false { erro(of(ctx,p), "flag name is trivial").debug(16) }
    } else if f, ok := p.name.(*Flag); ok {
        res, match = f.opt(ctx, name)
    } else if s := p.name.Strval(ctx); s == name {
        res, match = name, true
    }
    return
}
// DEPRECATED and not used anymore
func (p *Flag) opts(ctx Context, try bool, opts ...string) (runes []rune, names []string, err error) {
    switch t := p.name.(type) {
    case *Flag:
        runes, names, err = t.opts(ctx, try, opts...)
    case *String:
        for _, opt := range opts {
            if t.string == opt { names = append(names, opt) }
        }
        if !try && len(names) == 0 {
            erro(of(ctx,p), "unknown flag (known: %s)", strings.Join(opts, ", "))
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
            erro(of(ctx,p), "unknown flag (known: %s)", strings.Join(opts, ", "))
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
    } else if c, ok := v.(*Barecomp); ok {
        var elems []Value // right hand side barecomp elements
        for _, elem := range c.Elems {
            if !isNil(elem) && !isNone(elem) {
                elems = append(elems, elem)
            }
        }
        if len(elems) == 2 { if fR, ok := elems[0].(*Flag); ok {
            if isNil(fR.name) || isNone(fR.name) {
                res = p.name.cmp(ctx, elems[1])
            } else if m, s, t := fR.name.match(ctx, p.name); m {
                if isNil(elems[1]) || isNone(elems[1]) { res = cmpEqual }
            } else if s != "" { // matched prefix
                var sL = p.name.Strval(ctx)
                var sR = s + elems[1].Strval(ctx)
                if sL == sR {
                    res = cmpEqual
                } else if s < sR {
                    res = cmpSmaller
                } else {
                    res = cmpGreater
                }
            } else if t != nil {
                unreachable(p, v)
            }
        }}
    } else if l, y := v.(*List); y && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, y := v.(unexpanded); y && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if i, y := v.(*Int); y && i.int64 < 0 {
        if a, e := p.name.Integer(ctx); e == nil {
            if b := -i.int64; a == b { res = cmpEqual } else
            if a < b { res = cmpGreater } else { res = cmpSmaller }
        }
    } else if f, y := v.(*Float); y && f.float64 < 0 {
        if a, e := p.name.Float(ctx); e == nil {
            if b := -f.float64; a == b { res = cmpEqual } else
            if a < b { res = cmpGreater } else { res = cmpSmaller }
        }
    }
    return
}
func (p *Flag) traverse(ctx Context) (traves travestates) {
    ctx = at(ctx, p.position)
    return traverse(ctx, p, p.Strval(ctx))
}

const escapedChars = "\"\r\n"

type Compound struct { valbase ; elements } // "compound string"
func (p *Compound) String() string { return p.elemStr(nil, nil, 0) }
func (p *Compound) Strval(ctx Context) (s string) {
    for _, e := range p.Elems { s += e.Strval(ctx) }
    // NOTE: escaping \" here makes the string complicated
    if false { s = strings.Replace(s, `\"`, `"`, -1) }
    return
}
func (p *Compound) Float(ctx Context) (f float64, err error) {
    return strconv.ParseFloat(p.Strval(ctx), 64)
}
func (p *Compound) Integer(ctx Context) (i int64, err error) {
    return strconv.ParseInt(p.Strval(ctx), 10, 64)
}
func (p *Compound) True(ctx Context) bool { return p.elements.True(ctx) }
func (p *Compound) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *Compound) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *Compound) expandible(ctx Context, w facet) bool { return p.elements.expandible(ctx, w) }
func (p *Compound) expand(ctx Context, w facet) (res Value) {
    var elems, u, n = w.expand(ctx, p.Elems...)
    if n > 0 { res = &Compound{p.valbase, elements{elems}} } else { res = p }
    if u > 0 { res = unexpanded{res} }
    return
}
func (p *Compound) elemStr(ctx Context, o Object, k elemkind) (s string) {
    for _, elem := range p.Elems {
        s += elementString(ctx, o, elem, k|elemNoQuote)
    }
    if k&elemNoQuote != 0 {
        return
    }

    var (
        buf bytes.Buffer
        err error
    )
    buf.WriteString(`"`)
    defer func() {
        buf.WriteString(`"`)
        s = buf.String()
    } ()

    // Escape string chars
    for i := strings.IndexAny(s, escapedChars); i != -1; {
        if _, err = buf.WriteString(s[:i]); err != nil {
            erro(of(ctx,p), "%v", err)
            return
        }

        var esc string
        switch s[i] {
        case '"':  esc = `\"`
        case '\r': esc = `\r`
        case '\n': esc = `\n`
        }
        if _, err = buf.WriteString(esc); err != nil {
            erro(of(ctx,p), "%v", err)
            return
        }

        s = s[i+1:]
        i = strings.IndexAny(s, escapedChars)
    }
    if _, err = buf.WriteString(s); err != nil {
        erro(of(ctx,p), "%v", err)
    }
    return
}
func (p *Compound) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Compound); ok {
        if p.Strval(ctx) == a.Strval(ctx) { res = cmpEqual }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}

type List struct {
    position Position
    elements
}
func (_ *List) kind() kind { return valOther }
func (p *List) Position() (pos Position) { return p.position }
func (p *List) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *List) Strval(ctx Context) (s string) {
    var x = 0
    for _, e := range p.Elems {
        if e == nil {
            // TODO: special process for nil elements in a list??
        } else if v := e.Strval(ctx); v != "" {
            if 0 < x { s += " " }
            s += v
            x += 1
        }
    }
    return
}
func (p *List) Float(ctx Context) (f float64, _ error) {
    i, e := p.Integer(ctx); return float64(i), e
}
func (p *List) Integer(ctx Context) (i int64, err error) {
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
func (p *List) expand(ctx Context, w facet) (res Value) {
    var elems, u, n = w.expand(ctx, p.Elems...)
    if res = p; n > 0 { res = &List{p.position, elements{elems}}}
    if u > 0 { res = unexpanded{res} }
    return
}
func (p *List) traverse(ctx Context) (traves travestates) {
    if options.parallel {
        const (
            stateCont int = 0
            stateBrek     = 1
        )
        var state int
        var m sync.Mutex
        var pc sync.WaitGroup //= ctx.programContext()
        defer pc.Wait() ; for i, elem := range p.Elems {
            if state == stateBrek { break } else { pc.Add(1) }
            go func (ctx Context, i int, elem Value) {
                defer func() {
                    pc.Done()
                    if n, e := checkFailure(ctx); n>0 || e> 0 {
                        // TODO: panics and errors
                    }
                } ()

                var t = elem.traverse(ctx)
                {
                    m.Lock() //var unlock = ctx.aquireLock()
                    traves = append(traves, t...)
                    m.Unlock() //if unlock != nil { unlock() }
                }

                if _, ok := elem.(*modifiergroup); ok && t.has(traveNext) {
                    warn(ctx, "%T %v", elem, elem).debug(1)
                }
                if _, ok := elem.(*modifier); ok && t.has(traveNext) {
                    warn(ctx, "%T %v", elem, elem).debug(1)
                }
                if t.has(/*traveCase, traveNext, traveDone, */traveFail) {
                    m.Lock() //var unlock = ctx.aquireLock()
                    state = stateBrek
                    m.Unlock() //if unlock != nil { unlock() }
                }
            } (ctx.spawn(ctx), i, elem)
        }
    } else {
        for _, elem := range p.Elems {
            var t = elem.traverse(ctx)
            traves = append(traves, t...)
            if _, ok := elem.(*modifiergroup); ok && t.has(traveNext) {
                warn(ctx, "%T %v", elem, elem).debug(1)
            }
            if _, ok := elem.(*modifier); ok && t.has(traveNext) {
                warn(ctx, "%T %v", elem, elem).debug(1)
            }
            if t.has(/*traveCase, traveNext, traveDone, */traveFail) {
                break
            }
        }
    }
    return
}
func (p *List) updated(ctx Context, v ...bool) (res bool) {
    for _, elem := range p.Elems {
        if res = elem.updated(ctx, v...); res { break }
    }
    return
}
func (p *List) updatedDeps(ctx Context, v ...Value) (res []Value) {
    for _, elem := range p.Elems {
        res = append(res, elem.updatedDeps(ctx, v...)...)
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
    if a, ok := v.(*List); ok {
        var elemsL, elemsR []Value
        if len(p.Elems) == len(a.Elems) {
            elemsL, elemsR = p.Elems, a.Elems
        } else {
            elemsL, elemsR = merge(p.Elems...), merge(a.Elems...)
        }

        res = compareElems(ctx, elemsL, elemsR)

        if false && res != cmpEqual && p.String() == a.String() {
            for i, elem := range elemsL {
                warn(ctx, "L: %v: %d: %T %v", p, i, elem, elem)
            }
            for i, elem := range elemsR {
                warn(ctx, "R: %v: %d: %T %v", p, i, elem, elem)
            }
            warn(ctx, "%v <=> %v", p.Elems, a.Elems).debug(1)
        }
    } else if len(p.Elems) == 1 {
        res = p.Elems[0].cmp(ctx, v)
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if c, ok := v.(*Barecomp); ok && len(c.Elems)==2 && len(p.Elems)>1 {
        if cl, ok := c.Elems[1].(*List); ok && len(cl.Elems)>1 {
            var a = c.Elems[1] // Example: p.Elems=[z-a b c], c.Elems=[a- a b c]
            // FIXME: avoid 'c.Elems[1] = cl.Elems[0]', the container values are readonly for cmp
            if c.Elems[1] = cl.Elems[0]; p.Elems[0].cmp(ctx, c) == cmpEqual {
                res = compareElems(ctx, cl.Elems[1:], p.Elems[1:])
            }
            c.Elems[1] = a
        }
    }
    return
}

func (p *List) patterned(ctx Context) (res bool) {
    if len(p.Elems) == 1 {
        res = p.Elems[0].patterned(ctx)
    } else {
        /* FIXME: check pattern for each element??
        for _, elem := range p.Elems {
          if elem.patterned() { return true }
        }*/
    }
    return
}

func (p *List) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    if len(p.Elems) == 1 {
        full, s, stems = p.Elems[0].match(ctx, i)
    } else {
        /* FIXME: match for to each element??
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
//func (p *Group) Float(ctx Context) (f float64, _ error) { return p.valbase.Float(ctx) }
//func (p *Group) Integer(ctx Context) (i int64, e error) { return p.valbase.Integer(ctx) }
func (p *Group) True(ctx Context) (t bool) {
    if t = len(p.Elems) > 0; t {
        for _, elem := range p.Elems {
            if t = elem.True(ctx); !t { break }
        }
    }
    return
}
func (_ *Group) kind() kind { return valOther }
func (p *Group) Strval(ctx Context) (s string) {
    s = "("
    for i, elem := range p.Elems {
        if i > 0 { s += " " }
        s += elem.Strval(ctx)
    }
    s += ")"
    return
}
func (p *Group) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *Group) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
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
func (p *Group) expandible(ctx Context, w facet) bool { return p.elements.expandible(ctx, w) }
func (p *Group) expand(ctx Context, w facet) (res Value) {
    var elems, u, n = w.expand(ctx, p.Elems...)
    if n > 0 { res = &Group{p.valbase, elements{elems}} } else { res = p }
    if u > 0 { res = unexpanded{res} }
    return
}
func (p *Group) traverse(ctx Context) (traves travestates) {
    warn(at(ctx,p.position), "traversing group: %v", p).debug(32)
    return
}
func (p *Group) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Group); ok {
        if l1, l2 := len(p.Elems), len(a.Elems); l1 == 0 && l2 == 0 {
           return cmpEqual
        }
        res = compareElems(ctx, p.Elems, a.Elems)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}
func (p *Group) patterned(ctx Context, ) bool { return false }
func (p *Group) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    // TODO: for _, elem := range { elem.match(ctx, i) }
    return
}
func (p *Group) stencil(ctx Context, stems []string) (val Value, rest []string) {
    // TODO: for _, elem := range { elem.match(ctx, i) }
    return p, stems
}

func parseGroupValue(ctx Context, g *Group) (result Value) {
    if len(g.Elems) == 0 { return g } else {
        var word *Bareword
        switch kind := g.Elems[0].(type) {
        case *Bareword: word = kind
        case *Group: if len(kind.Elems) > 0 {
            var ( name = kind.Elems[0]; ok bool )
            if word, ok = name.(*Bareword); !ok {
                erro(of(ctx,name), "unsupported name type: %T %v", name, name).debug(1)
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
func (p *Pair) Strval(ctx Context) string {
    return p.Key.Strval(ctx) + "=" + p.Value.Strval(ctx)
}
func (p *Pair) True(ctx Context) (t bool) {
    if t = p.Key.True(ctx); !t && !isNil(p.Value) {
        t = p.Value.True(ctx)
    }
    return
}
func (p *Pair) Integer(ctx Context) (i int64, e error) { return p.Value.Integer(ctx) }
func (p *Pair) Float(ctx Context) (f float64, e error) { return p.Value.Float(ctx) }
func (p *Pair) refs(ctx Context, v Value) bool { return p.Key.refs(ctx, v) || p.Value.refs(ctx, v) }
func (p *Pair) defs(ctx Context, s ...string) []*def {
    return append(p.Key.defs(ctx, s...), p.Value.defs(ctx, s...)...)
}
func (p *Pair) traverse(ctx Context) (traves travestates) {
    erro(at(ctx,p.position), "traversing pair '%v' is undefined", p)
    errostack(at(ctx, p.position), -1, "pair is not traversible: %v", p).debug(16)
    return
}
func (p *Pair) expandible(ctx Context, w facet) bool {
    if p.Key.expandible(ctx, w) { return true }
    return w&expandPairVal != 0 && p.Value.expandible(ctx, w)
}
func (p *Pair) expand(ctx Context, w facet) (res Value) {
    res = p

    // NOTE: The value (p.Value) could be used as template arguments in many
    // NOTE: cases (eg copy-file). So don't implicitly expand p.Value!
    if k := p.Key.expand(ctx, w); w&expandPairVal != 0 {
        if v := p.Value.expand(ctx, w); k != p.Key || v != p.Value {
            res = &Pair{p.valbase, k, v}
        }
    } else if k != p.Key {
        res = &Pair{p.valbase, k, p.Value}
    }
    return
}
func (p *Pair) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var k, v Value
    k, rest = p.Key.stencil(ctx, stems)
    v, rest = p.Value.stencil(ctx, rest)

    var (
        kNil = isNil(k)
        vNil = isNil(v)
    )
    if (!kNil && k != p.Key) || (!vNil && v != p.Value) {
        if kNil { k = p.Key   }
        if vNil { v = p.Value }
        val = &Pair{p.valbase, k, v}
    }
    return
}
func (p *Pair) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*Pair); ok {
        if res = p.Key.cmp(ctx, a.Key); res == cmpEqual {
            if false {
                res = p.Value.expand(ctx, plain).cmp(ctx, a.Value)
            } else {
                res = p.Value.cmp(ctx, a.Value)
            }
        }
    } else if u, o := v.(paircomp); o && u.Value != nil {
        res = p.cmp(ctx, u.Pair)
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    }
    return
}
func (p *Pair) elemStr(ctx Context, o Object, k elemkind) string {
    return elementString(ctx, o, p.Key, k)+`=`+elementString(ctx, o, p.Value, k)
}

func containsUndefinedAutos(ctx Context, noUnderscore bool, vals... Value) (res bool) {
    for _, val := range vals {
        for _, d := range val.defs(ctx) {
            if d.origin == DefArg || d.origin == DefAuto {
                if noUnderscore && d.name == "_" {
                    continue
                }

                var v Value
                if a := ctx.autoGet(d.name); a == nil {
                    return true // not ready to reveal (call/execute)
                } else if v = a.value; isTrivial(v) {
                    continue
                }

                // Check for examples:
                //   .test.y := $(.test$1)
                //   .test.x := $(.test.y .$1)
                //   .test   := $(.test.x)
                //   assert $(equal -d $(string -e $(.test)),'$(.test.$1)')
                for _, dd := range v.defs(ctx, d.name) {
                    if dd == d {
                        return true // not ready to reveal (call/execute)
                    }
                }
            }
        }
    }
    return
}

type unexpanded struct { Value }
func (u unexpanded) True(ctx Context) bool { return false }
func (u unexpanded) traverse(ctx Context) (traves travestates) { return }
func (u unexpanded) match(_ Context, _ interface{}) (b bool, s string, a []string) { return }
func (u unexpanded) expand(ctx Context, w facet) Value {
    return u.Value.expand(ctx, w) // NOTE: for a stack in debug traces
}

type untraversed struct { Value }
// func (u untraversed) True(ctx Context) bool { return false }
func (u untraversed) traverse(ctx Context) (traves travestates) { return }
func (u untraversed) expand(ctx Context, w facet) Value {
    if v := u.Value.expand(ctx, w); v != u.Value { u = untraversed{v} }
    return u
}

// Delegate wraps '$(foo a,b,c)' into Valuer
type delegate struct {
    valbase
    l token.Token
    x Value
    a []Value
    //TODO: patsubst Value, aka lhs%=rhs% like in $(var:lhs%=rhs%)
}
func (p *delegate) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *delegate) Strval(ctx Context) (s string) {
    if v := p.expand(ctx, plain); v == nil {
        if false { erro(of(ctx,p), "delegate value is nil: %v", p).debug(1) }
    } else if u, y := v.(unexpanded); (y /* && u.Value == p */) || v == p {
        if false {
            erro(of(ctx,p), "delegate can't expand: %v -> %v", p, u.Value).debug(1)
        } else if options.debug {
            warn(of(ctx,p), "delegate can't expand: %v (%v)", p, u.Value==p).debug(1)
        }
    } else {
        if v.refs(ctx, p.x) {
            warnstack(ctx, 3, "%v %v ; %T %v", p, p.x, v, v).debug(128)
            ctx.checkErrors(true)
        } else {
            s = v.Strval(ctx)
        }
    }
    return
}
func (p *delegate) True(ctx Context) (t bool) {
    if v := p.expand(ctx, plain); v == nil {
        if false { erro(at(ctx,p.position), "expand '%v' to nil", p).debug(10) }
    } else if u, y := v.(unexpanded); (y /* && u.Value == p */) || v == p {
        if false {
            erro(of(ctx,p), "delegate can't expand: %v -> %v", p, u).debug(1)
        } else if options.debug {
            warn(of(ctx,p), "delegate can't expand: %v (%v)", p, u.Value==p).debug(1)
        }
    } else if t = v.True(ctx); false {
        info(at(ctx,p.position), "%v -> %T %v -> %v", p, v, v, t).debug(8)
    }
    return
}
func (p *delegate) Integer(ctx Context) (i int64, e error) {
    if v := p.expand(ctx, plain); v == nil {
        if false { erro(of(ctx,p), "delegate value is nil: %v", p).debug(1) }
    } else if u, y := v.(unexpanded); (y /* && u.Value == p */) || v == p {
        if false {
            erro(of(ctx,p), "delegate can't expand: %v -> %v", p, u).debug(1)
        } else if options.debug {
            warn(of(ctx,p), "delegate can't expand: %v (%v)", p, u.Value==p).debug(1)
        }
    } else {
        i, e = v.Integer(ctx)
    }
    return
}
func (p *delegate) Float(ctx Context) (f float64, e error) {
    if v := p.expand(ctx, plain); v == nil {
        if false { erro(of(ctx,p), "delegate value is nil: %v", p).debug(1) }
    } else if u, y := v.(unexpanded); (y /* && u.Value == p */) || v == p {
        if false {
            erro(of(ctx,p), "delegate can't expand: %v -> %v", p, u).debug(1)
        } else if options.debug {
            warn(of(ctx,p), "delegate can't expand: %v (%v)", p, u.Value==p).debug(1)
        }
    } else {
        f, e = v.Float(ctx)
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
        erro(of(ctx,p), "delegation of nil (v=%v)", v).debug(1)
        return
    } else if p.x == v || p.x.refs(ctx, v) {
        return true
    }
    for _, a := range p.a {
        if a.refs(ctx, v) { return true }
    }
    return
}
func (p *delegate) defs(ctx Context, s ...string) (res []*def) {
    if isNil(p.x) {
        erro(of(ctx,p), "delegation of nil (s=%v)", p, s).debug(1)
        return
    } else if d, y := p.x.(*def); y {
        if y = len(s) == 0; !y {
            for _, a := range s { if y = d.name == a; y { break } }
        }
        if y { res = append(res, d) }
    } else {
        res = p.x.defs(ctx, s...)
    }
    for _, a := range p.a {
        res = append(res, a.defs(ctx, s...)...)
    }
    return
}
func (p *delegate) traverse(ctx Context) (traves travestates) {
    ctx = at(ctx, p.position)
    if val := p.expand(ctx, plain); val == nil {
        warn(at(ctx,p.position), "delegate '%v' expands to nil", p)
        warnstack(ctx, -1, "").debug(16)
    } else if isTrivial(val) {
        if false {
            warn(at(ctx,p.position), "delegate '%v' expands to none", p)
            warnstack(ctx, -1, "", p).debug(16)
        }
    } else {
        traves = val.traverse(ctx)
    }
    return
}
func (p *delegate) name(ctx Context, sel bool) (name string) {
    switch x := p.x.(type) {
    case Object: name = x.Name(ctx)
    case *selection: if sel { name = x.Strval(ctx) }
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
    case *selection: name = x.String() // x.Strval(ctx)
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
func (p *delegate) _elemStr(ctx Context, o Object, k elemkind, l string) (s string) {
    if ctx == nil {
        if s = p.string(ctx, o, k); !(p.l.IsClosure() || p.l.IsDelegate()) { s = l + s }
    } else if false {
        s = elementString(ctx, o, p, k)
    }
    return
}
func (p *delegate) elemStr(ctx Context, o Object, k elemkind) string {
    return p._elemStr(ctx, o, k, "$")
}
func (p *delegate) call(ctx Context, w facet, d *def, args ...Value) (res Value, final bool) {
    if res = d.call(ctx, w, args...); res == nil {
        if final = true; d.value == nil {
            if false {    } else
            if o := ctx.Scope().FindDef(d.name); o != nil && o != d {
                res = o.call(ctx, w, args...)
            }
            if res == nil { res = unexpanded{p} }
        } else {
            res = d.value
        }
    }
    return
}
func (p *delegate) args(ctx Context, w facet) (args []Value, u, n int) {
    if len(p.a) < 1 { return } else
    if b, y := p.x.(*Builtin); y {
        var ( left = (w|b.s.m) ; right = (w&^b.s.m|b.s.o) )
        if l := len(p.a); b.s.m == expandZero || l <= b.s.n {
            args, u, n = left.expand(ctx, p.a...)
        } else if l == 0 { // NOTE: b.s.n could be -1
            // does nothing...
        } else if b.s.n <= 0 {
            args, u, n = right.expand(ctx, p.a...)
        } else if args, u, n = left.expand(ctx, p.a[:b.s.n]...); b.s.n < l {
            a, x, y := right.expand(ctx, p.a[b.s.n:]...)
            args, u, n = append(args, a...), u+x, n+y
        }
    } else {
        args, u, n = w.expand(ctx, p.a...)
    }
    if len(args) == 0 && u == 0 && n == 0 { args = p.a }
    return
}
func (p *delegate) expandible(ctx Context, w facet) (res bool) {
    if res = w&expandDelegate != 0; !res {
        if def, ok := p.x.(*def); ok && (def.origin == DefAuto && w&expandAuto == 0) {
            res = false // p.x.expandible(ctx, w) // TODO: auto -> placeholder
        } else {
            res = p.x.expandible(ctx, w)
        }
        if !res { for _, a := range p.a {
            if res = a.expandible(ctx, w); res { break }
        }}
    }
    return
}
func (p *delegate) expand(ctx Context, w facet) (res Value) {
    if /* ctx = at(ctx, p.position) */; isNil(p.x) {
        erro(at(ctx,p.position), "delegate of nil: %v (w=%024b)", p, w)
        errostack(at(ctx,p.position), 5, "delegate nil: %v (%p)", p, p).debug(64)
        return
    } else if isNone(p.x) {
        erro(at(ctx,p.position), "invalid delegate: %v (w=%024b)", p, w)
        errostack(at(ctx,p.position), 5, "%v", p).debug(64)
        return
    } else if w == 0 {
        erro(ctx, "%v: zero expand", p).debug(64)
        return
    }

    var final bool
    var offBits = expandPlain
    var close = w&expandPlain != 0
    if  close { w = (w&^offBits) | expandClose }
    if false { if s := p.String();
        // s == "$(foreach q p $(foreach $1,&(.test.foo)$_),x$_)" ||
        // s == "$(call -c &(.test.x),$1$1,$2$2)" ||
        // s == "$(foreach $1,&(.test.foo)$_)" ||
        // s == "$(value -c &(.test.x))" ||
        // s == "$(.test$1)" ||
        // s == "$(.test)" ||
        // s == "${.test.foobar}" ||
        // p.x.String() == ".test.foobar" ||
        (false && s == "") {
        defer func() { if r := res.String();
            // r == "foobar $1-$2 foobar $1$1$1$1-$2$2$2$2 foobar $1$1$1$1-$2$2$2$2" ||
            (false && r == "") {
            var t, y = res.(unexpanded)
            var same = res == p || (y && t.Value == p)
            warn(ctx, "delegate.expand: 1: %v", autoGet(ctx, "1"))
            warn(ctx, "delegate.expand: 2: %v", autoGet(ctx, "2"))
            warn(ctx, "delegate.expand: _: %v", autoGet(ctx, "_"))
            warn(ctx, "delegate.expand: %v", s)
            warn(ctx, "delegate.expand: -> %T %v", res, res)
            if true {
                // ...
            } else if !same {
                var t = res.expand(ctx, w)
                warn(ctx, "delegate.expand: -> %T %v", t, t)
            } else if y {
                warn(ctx, "delegate.expand: unexpanded: %T %v", t.Value, t.Value)
            }
            warn(ctx, "delegate.expand: final=%v; same=%v; close=%v; %020b",
                final, same, close, w).debug(64)
            ctx.checkErrors(true)
        }} ()
    }}

    if res, final = p.reveal(ctx, w); res == nil {
        return MakeNil(p.position)
    } else if close {
        w |= offBits
        w &= ^(expandClose)
        // if u, y := res.(unexpanded); y { res = u.Value }
    } else if final {
        return
    } else {
        w &= ^(expandDigits|expandPlaceholders)
    }

    // avoid self-expand loop
    if u, y := res.(unexpanded); res == p || (y && u.Value == p) {
        return
    } else if a := ctx.auto(); a != nil {
        // chop off the auto context to preserve $1, $2... in res
        return res.expand(a.inner(), w)
    } else {
        return res.expand(ctx, w)
    }
}
func (p *delegate) reveal(ctx Context, w facet) (res Value, final bool) {
    res = p // set default to selfing

    var (
        // NOTE: expand arguments' delegates (including known autos)
        args, u, n = p.args(ctx, w)
        x Object
        db int
    )
    if false { if s := p.String(); (
        // s == "$(call -c &(.test.x),$1$1,$2$2)" ||
        // s == "$(foreach $1 $2,$(value .test.$_) $(value .test~&(.test.s).$_))" ||
        // s == "$(foreach $1,$(value -c .test.$_)$1)" ||
        // s == "$(foreach $1,&(.test.foo)$_)" ||
        // s == "$(foreach $1,&(.test.x.$_))" ||
        // s == "$(foreach $1,&(.test.x.))" ||
        // s == "$(foreach $1 $2,-x$_)" ||
        // s == "$(foreach $1 $2,-y$_)" ||
        // s == "$(value -c &(.test.x))" ||
        // s == "$(value .test~&(.test.s))" ||
        // s == "$(value .test~$(.test.s))" ||
        // s == "$(.test$1)" ||
        // s == "$(.test.x)" ||
        // s == "$(.test)" ||
        // s == "$(.test.z y$1,y$2)" ||
        // s == "$(ext $@)" ||
        // s == "$(debug -s=50 -n=60 $1)" ||
        // s == "$(.test.s $_,AsmParser)" ||
        // (db > 0 && s == "$1") ||
        // (dd && s == "$(-std.$_)") ||
        // s == "${.test.foobar}" ||
        // s == "${.test.foobaz}" ||
        // s == "${.test.foobay}" ||
        (dd && s == "$_") ||
        (false && s == "")) { db += 1
        defer func() { if res != nil { if r := res.String(); true ||
            // r == "foobar $1-$2 foobar $1$1$1$1-$2$2$2$2 foobar $1$1$1$1-$2$2$2$2" ||
            // r == "$(call -c &(.test.x),$1$1,$2$2)" ||
            // r == "$(.test.z y$1,y$2)" ||
            // r == "$(ext $@)" ||
            (false && r == "") {
            if false && (len(p.a)>0 || len(args)>0 || n>0) {
                warn(ctx, "%T %v %v (merge(p.a))", x, x, merge(p.a...))
                for _, a := range merge(p.a...) {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v; %T %v", a, a, t, t) }
                warn(ctx, "%T %v %v (merge(args))", x, x, merge(args...))
                for _, a := range merge(args...) {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v; %T %v", a, a, t, t) }

                warn(ctx, "%T %v %v (p.a)", x, x, p.a)
                for _, a := range p.a {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v; %T %v", a, a, t, t) }
                warn(ctx, "%T %v %v (args)", x, x, args)
                for _, a := range args {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v; %T %v", a, a, t, t) }
            }
            var t, y = res.(unexpanded)
            var same = res == p || (y && t.Value == p)
            warn(ctx, "reveal: 1: %v", autoGet(ctx, "1"))
            warn(ctx, "reveal: 2: %v", autoGet(ctx, "2"))
            warn(ctx, "reveal: _: %v", autoGet(ctx, "_"))
            warn(ctx, "reveal: <: %v", autoGet(ctx, "<"))
            warn(ctx, "reveal: >: %v", autoGet(ctx, ">"))
            warn(ctx, "reveal: %T: %v", p.x, p)
            warn(ctx, "reveal: args=%v, unexpanded=%v, transformed=%v", args, u, n)
            warn(ctx, "reveal: -> %T %v (same=%v)", res, res, same)
            if !same {
                ; dd = true
                var t = res.expand(ctx, w)
                warn(ctx, "reveal: -> %T %v", t, t)
                ; dd = false
            } else if y {
                warn(ctx, "reveal: unexpanded: %T %v", t.Value, t.Value)
            }
            warnstack(ctx, 3, "reveal: final=%v; same=%v; %020b", final, p.x==x, w).debug(100)
            ctx.checkErrors(true)
            db -= 1
        }}} ()
    }}

    if ur, y := p.x.(*unresolvedobject); y {
        // Expand name first, for example of $(.test$1) containing '$1':
        //     .test.y := $(.test$1)
        //     .test.x := $(.test.y .$1)
        //     .test   := $(.test.x)
        //     assert $(equal -d $(string -e $(.test)),'$(.test.$1)')
        var ux unexpanded
        var name = ur.name.expand(ctx, w|expandUnresName)
        if ux, y = ur.name.(unexpanded); !y { ux, y = name.(unexpanded) }
        if y {
            if res = p; ux.Value != ur.name || n>0 {
                res = &delegate{p.valbase, p.l, &unresolvedobject{ur.objbase, ux.Value}, args}
            }
            return res, true // not ready to reveal (call/execute)
        }

        // NOTE: return immediately if there are any autos/args not defined yet
        if _, y := name.(undef); y {
            return res, true
        } else if str := name.Strval(ctx); str == "" {
            erro(of(ctx,name), "empty unresolved name: %v", name).debug(1)
            return
        } else {
            if _, sym := ctx.Scope().Find(str); sym != nil {
                if sym.(Value) != p { x = sym }
            }
            if x == nil {
                if res = p; name != ur.name || n>0 {
                    res = &delegate{p.valbase, p.l, &unresolvedobject{ur.objbase, name}, args}
                }
                return res, true // not ready to reveal (call/execute)
            }
        }
    } else if t, y := p.x.(Object); y && t != nil {
        x = t
    } else if t, y := p.x.(*uselist); y {
        x = t
    } else if t, y := p.x.(*selection); !y || t == nil {
        erro(ctx, "delegate unsupported object: %T %v", p.x, p.x)
        errostack(ctx, 3, "%v", ctx).debug(16)
        return
    } else if isNil(t.o) {
        erro(ctx, "%v: selection object is nil (w=%024b)", p, w)
        errostack(ctx, 3, "%v", ctx).debug(16)
        return
    } else {
        if pn, y := t.o.(*ProjectName); y && pn != nil && pn.project != nil {
            // FIXME: closure with pn.project not working
            ctx = closureWith(ctx, pn.project.scope)
        }

        if v := t.value(ctx); isNil(v) {
            erro(ctx, "%v: selected value is nil (%T %v) (w=%024b)", p, t.o, t.o, w)
            errostack(ctx, 3, "%v", ctx).debug(16)
            return
        } else if t, y := v.(Object); !y {
            erro(ctx, "%v: selected value is not object (%T %v) (w=%024b)", t, v, v, w)
            errostack(ctx, 3, "%v", ctx).debug(16)
            return
        } else {
            x = t
        }
   }

    if false && db > 0 {
        var t1 = autoGet(ctx, "_")
        var t2 = autoGet(ctx, "1")
        var t3 = autoGet(ctx, "2")
        info(ctx, "%v ; %v ; %v", t1, t2, t3)
        info(ctx, "%v %v %024b", x, args, w)
        for i, a := range merge(args...) { info(ctx, "%d. %T %v", i, a, a) }
        info(ctx, "u=%d, n=%d", u, n)
        infostack(ctx, 3, "").debug(64)
    }

    var unexpand = (w&expandDelegate == 0)
    if !unexpand && u > 0 {
        if w&expandClose == 0 {
            unexpand = true
        } else if w&expandPlaceholders == 0 && w&expandUnPlaceholders != 0 {
            for _, a := range args {
                if d := a.defs(ctx, "_"); len(d) > 0 {
                    unexpand = true; break
                }
            }
        } else if w&expandDigits == 0 && w&expandUnDigits != 0 {
            var ds = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
            for _, a := range args {
                if d := a.defs(ctx, ds...); len(d) > 0 {
                    unexpand = true; break
                }
            }
        }
    }

    var bin, _ = x.(*Builtin)
    if bin == nil {
        // def, Caller, Executer ...
    } else if bin.name == "auto" {
        return builtinAuto(ctx, p, w, args...), final
    }

    if unexpand {
        if p.x != x || n > 0 { res = &delegate{p.valbase, p.l, x, args} }
        if w&expandClose != 0 { return res, true }
        return unexpanded{res}, true
    } else if bin != nil {
        if ctx = at(ctx, p.position); bin.name == "call" {
            res, final = builtinCall(ctx, p, w, args...), true
        } else if bin.s.f == nil {
            // ...
        } else if res = bin.s.f(ctx, w, args...); false && db > 0 {
            for i, a := range p.a  { info(ctx, "p.a[%d]: %T %v",  i, a, a) }
            for i, a := range args { info(ctx, "args[%d]: %T %v", i, a, a) }
            infostack(ctx, 5, "%v %v -> %T %v ; %v %v ; %v %v", x, args, res, res,
                autoGet(ctx, "3"), autoGet(ctx, "4"), u, n).debug(64)
        }
        return
    } else if d, y := x.(*def); y {
        return p.call(ctx, w, d, args...)
    }

    switch t := x.(type) {
    case Caller:
        res = t.Call(ctx, args...)
        return
    case Executer:
        if vals, traves := t.Execute(ctx, args...); traves.has(traveFail) {
            for _, s := range traves {
                erro(at(ctx,s.pos), "broken '%v': %v", x, s).debug(1)
            }
        } else if len(vals) > 0 {
            res = MakeList(ctx.Position(), vals...)
        }
        return
    }

    erro(ctx, "unknown delegation: %T %v -> %T %v", p.x, p.x, x, x).debug(32)
    return
}
func (p *delegate) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    if v := p.expand(ctx, plain); v != nil {
        if v != p { full, s, stems = v.match(ctx, i) }
        return
    } else {
        erro(ctx, "%v: expand to nil", p).debug(1)
    }
    return
}
func (p *delegate) stat(ctx Context) (si *statinfo) {
    erro(at(ctx,p.position), "cant stat delegate %v, must expand it first", p).debug(16)
    return
}
func (p *delegate) stamp(ctx Context) (file []*File, err error) {
    erro(at(ctx,p.position), "cant stamp delegate %v, must expand it first", p).debug(16)
    return
}
func (p *delegate) delete(ctx Context) (file []*File, err error) {
    erro(at(ctx,p.position), "cant delete delegate %v, must expand it first", p).debug(16)
    return
}
func (p *delegate) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*delegate); ok { // NOTE: don't expand the delegate!!!
        if p == a { return cmpEqual }
        if t := p.x.cmp(ctx, a.x); t == cmpEqual {
            if len(p.a) == 0 && len(a.a) == 0 {
                res = t
            } else if len(p.a) == len(a.a) { for i, t := range p.a {
                if res = t.cmp(ctx, a.a[i]); res != cmpEqual { return }
            }}
        }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if d, ok := p.x.(*def); ok && len(p.a) == 0 && d.value != nil {
        res = d.value.cmp(ctx, v)
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}

type closure struct { delegate }
func (p *closure) String() (s string) { return p.elemStr(nil, nil, 0) }
func (p *closure) Strval(ctx Context) (s string) {
    if !p.isValidToken() {
        erro(at(ctx,p.Position()), "invalid closure token: %v", p.l).debug(1)
        return
    }

    if val := p.expand(ctx, plain); isNil(val) {
        if false { warn(of(ctx,p), "expand '%v' to nil", p).debug(1) }
    } else if val == p {
        if false { errostack(of(ctx,p), 10, "closure can't expand: %v", p).debug(32) }
    } else if u, y := val.(unexpanded); y && u.Value == p {
        // ""
    } else {
        s = val.Strval(ctx)
    }
    return
}
func (p *closure) True(ctx Context) (t bool) {
    if val := p.expand(ctx, plain); isNil(val) {
        // does nothing
    } else if val == p {
        if false { errostack(of(ctx,p), 10, "closure can't expand: %v", p).debug(16) }
    } else if u, y := val.(unexpanded); y && u.Value == p {
        // false
    } else if t = val.True(ctx); false {
        info(at(ctx,p.position), "%v -> %T %v -> %v", p, val, val, t).debug(8)
    }
    return
}
func (p *closure) elemStr(ctx Context, o Object, k elemkind) string {
    return p._elemStr(ctx, o, k, "&")
}
func (p *closure) expandible(ctx Context, w facet) (res bool) {
    if res = w&expandClosure != 0; !res {
        if res = p.x.expandible(ctx, w); !res {
            for _, a := range p.a {
                if res = a.expandible(ctx, w); res { break }
            }
        }
    }
    return
}
func (p *closure) match(ctx Context, i interface{}) (full bool, s string, stems []string) {
    if v := p.expand(ctx, plain); v != p {
        return v.match(ctx, i)
    } else if false {
        errostack(of(ctx,p), 3, "unexpand closure: %v", v).debug(16)
    }
    return
}
func (p *closure) expand(ctx Context, w facet) (res Value) {
    if ctx, res = at(ctx, p.position), p; isNil(p.x) {
        erro(ctx, "expand nil closure: %v (%d)", p, w).debug(1)
        return
    } else if w == 0 {
        erro(ctx, "%v: zero expand", p).debug(64)
        return
    }

    var final bool
    var offBits = expandPlain
    var close = w&expandPlain != 0
    if close { w = (w&^offBits) | expandClose }
    if false { if s := p.String();
        // s == "&(.test.x)" ||
        // s == "&(objects)" ||
        (false && s == "") {
        defer func() { if res != nil { if r := res.String(); //true ||
            // strings.Contains(r, "libunwind.o") ||
            (false && r == "") {
            var t, y = res.(unexpanded)
            var same = res == p || (y && t.Value == p)
            warn(ctx, "closure.expand: %v", s)
            warn(ctx, "closure.expand: -> %T %v", res, res)
            if !same {
                var t = res.expand(ctx, w)
                warn(ctx, "closure.expand: -> %T %v", t, t)
            } else if y {
                warn(ctx, "closure.expand: unexpanded: %T %v", t.Value, t.Value)
            }
            warn(ctx, "closure.expand: final=%v; same=%v; close=%v; %020b",
                final, same, close, w)
            warnstack(ctx, 5, "").debug(64)
            ctx.checkErrors(true)
        }}} ()
    }}

    if false && w&expandUnresName != 0 {
        return unexpanded{ res }
    }

    if res, final = p.disclose(ctx, w); isNil(res) {
        erro(of(ctx,p), "disclose '%v' to nil (%s '%v')", p, typeof(p.x), p.x).debug(16)
        return
    } else if close {
        w |= offBits
        w &= ^(expandClose)
        // if u, y := res.(unexpanded); y { res = u.Value }
        // if res == p { return MakeNone(p.position) }
    } else if final {
        return
    } else {
        w &= ^(expandDigits|expandPlaceholders)
    }

    // avoid self-expand loop
    if u, y := res.(unexpanded); res == p || (y && u.Value == p) {
        return
    } else {
        return res.expand(ctx, w)
    }
}
func (p *closure) disclose(ctx Context, w facet) (res Value, final bool) {
    var (
        args, u, n = p.args(ctx, w)
        unresolved *unresolvedobject
        str string // the name in string form
        x Object
        y bool
    )
    if false { if s := p.String();
        // s == "&(.test.v2)" ||
        // s == "&(objects)" ||
        s == "&(cflags $<)" ||
        (dd && s == "&(-std.$_)") ||
        (false && s == "") {
        defer func() { if r := res.String(); p != res && ( true ||
            // strings.Contains(r, "libunwind.o") ||
            // r == "&(.test.v2)" ||
            (false && r == "")) {
            if len(p.a)>0 || len(args)>0 || n>0 {
                warn(ctx, "%T %v %v (merge(p.a))", x, x, merge(p.a...))
                for _, a := range merge(p.a...) {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v -> %T %v", a, a, t, t) }

                warn(ctx, "%T %v %v (merge(args))", x, x, merge(args...))
                for _, a := range merge(args...) {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v -> %T %v", a, a, t, t) }

                warn(ctx, "%T %v %v (p.a)", x, x, p.a)
                for _, a := range p.a {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v -> %T %v", a, a, t, t) }

                warn(ctx, "%T %v %v (args)", x, x, args)
                for _, a := range args {
                    var t = a.expand(ctx, plain)
                    warn(ctx, ">  %T %v -> %T %v", a, a, t, t) }
            }
            var t, y = res.(unexpanded)
            var same = res == p || (y && t.Value == p)
            warn(ctx, "disclose: 1: %v", autoGet(ctx, "1"))
            warn(ctx, "disclose: 2: %v", autoGet(ctx, "2"))
            warn(ctx, "disclose: _: %v", autoGet(ctx, "_"))
            warn(ctx, "disclose: args=%v ; unexpanded=%v, transformed=%v", args, u, n)
            warn(ctx, "disclose: %v ; x: %T %v (%v)", p, x, x, (x==unresolved))
            warn(ctx, "disclose: -> %T %v (same=%v)", res, res, same)
            if !same {
                var t = res.expand(ctx, w|expandFullName)
                warn(ctx, "disclose: -> %T %v", t, t)
            } else if y {
                warn(ctx, "disclose: unexpanded: %T %v", t.Value, t.Value)
            }
            warnstack(ctx, 10, "final=%v; same=%v; %020b", final, p.x==x, w).debug(32)
            ctx.checkErrors(true)
        }} ()
    }}

    res = p // set default result to self

    if unresolved, y = p.x.(*unresolvedobject); y {
        // Expand name first, for example of $(.test$1) containing '$1':
        //     .test.y := &(.test$1)
        //     .test.x := $(.test.y .$1)
        //     .test   := $(.test.x)
        //     assert $(equal -d $(string -e $(.test)),'&(.test.$1)')
        var name Value   = unresolved.name.expand(ctx, w|expandUnresName)
        if false { if s := unresolved.name.String();
            s == ".test.x.$_" || s == ".test.x.o.$_" {
            if bc, y := unresolved.name.(*Barecomp); y {
                for i, elem := range bc.Elems {
                    warn(ctx, "%d. %T %v", i, elem, elem)
                }
            }
            if bc, y := name.(*Barecomp); y && name != unresolved.name {
                for i, elem := range bc.Elems {
                    warn(ctx, "%d. %T %v", i, elem, elem)
                }
            }
            warn(ctx, "%v", autoGet(ctx, "_"))
            warnstack(ctx, 3, "%T %v -> %T %v, %024b %024b",
                unresolved.name, unresolved.name, name, name, w,
                w & (expandPlaceholders | expandDigits)).debug(32)
            ctx.checkErrors(true)
        }}

        var ux unexpanded
        if ux, y = name.(unexpanded); y {
            if t := unresolved.name != ux.Value; n > 0 || t {
                if t { unresolved = &unresolvedobject{unresolved.objbase, ux.Value}}
                res = &closure{delegate{p.valbase, p.l, unresolved, args}}
            }
            return unexpanded{res}, true // not ready to disclose (call/execute)
        } else if unresolved.name != name {
            unresolved = &unresolvedobject{unresolved.objbase, name}
        }

        if x, str = unresolved, name.Strval(ctx); str == "" {
            erro(of(ctx,p.x), "empty closure name: %T %v -> %T %v ; %v", p.x, p.x, name, name, unresolved).debug(16)
            return
        }
    } else if t, y := p.x.(Object); y && t != nil {
        if x, str = t, t.Name(ctx); str == "" {
            erro(of(ctx,p.x), "empty closure name: %T %v -> %T %v", p.x, p.x, x, x).debug(16)
            return
        }
    } else if t, y := p.x.(*selection); y && t != nil && !isNil(t.o) {
        if t, y := t.o.(*ProjectName); y && t != nil && t.project != nil {
            ctx = closureWith(ctx, t.project.scope)
        }
        if v := t.value(ctx); isNil(v) {
            erro(of(ctx,p.x), "selected nil value: %T %v", p.x, p.x).debug(1)
            return
        } else if t, y := v.(Object); !y {
            erro(of(ctx,p.x), "selected non object: %T %v", v, v).debug(16)
            return
        } else if x, str = t, t.Name(ctx); str == "" {
            erro(of(ctx,p.x), "empty closure name: %T %v -> %T %v ; %v", p.x, p.x, x, x, unresolved).debug(16)
            return
        }
    } else {
        erro(of(ctx,p.x), "undefined closure object: %T %v", p.x, p.x).debug(16)
        return
    }

    switch p.l {
    case token.STRING, token.COMPOUND:
        // &'xxx' and &"xxx" are fetching entry in the closure context
        res = closureResolveEntry(ctx, str)
        return
    case token.LBRACE:
        if t := closureResolveEntry(ctx, str); t != nil { x = t }
    default: // token.LPAREN, token.ILLEGAL
        if t := closureResolveObject(ctx, str); t != nil { x = t }
    }

    var changed = p.x != x || n > 0
    if w&expandClosure == 0 || (unresolved != nil) {
        if changed { res = &closure{delegate{p.valbase, p.l, x, args}} }
        if w&expandClose != 0 { return res, true }
        return unexpanded{res}, true
    } else if changed {
        if t1, t2 := reflect.TypeOf(p.x), reflect.TypeOf(x); t1 != t2 {
            if unres := reflect.TypeOf((*unresolvedobject)(nil)); t1 != unres {
                erro(at(ctx,p.position), "closure object type differs: %v (!= %v)", t2, t1)
                erro(at(ctx,p.x.Position()), "%v '%s' is found here", t1, str)
                erro(at(ctx,ctx.Position()), "%v", ctx).debug(16)
                return
            }
        }
        res = &delegate{p.valbase, p.l, x, args}
        return
    } else {
        res = &p.delegate
        return
    }
}
func (p *closure) traverse(ctx Context) (traves travestates) {
    ctx = at(ctx, p.position)
    if val := p.expand(ctx, /* expandClosure */plain); isNil(val) {
        warn(ctx, "closure '%v' expands to nil", p).debug(1)
    } else if isNone(val) {
        warn(ctx, "closure '%v' expands to none", p).debug(1)
    } else {
        traves = val.traverse(ctx)
    }
    return
}
func (p *closure) stat(ctx Context) (si *statinfo) {
    erro(at(ctx,p.position), "cant stat closure %v, must expand it first", p).debug(16)
    return
}
func (p *closure) stamp(ctx Context) (file []*File, err error) {
    erro(at(ctx,p.position), "cant stamp closure %v, must expand it first", p).debug(16)
    return
}
func (p *closure) delete(ctx Context) (file []*File, err error) {
    erro(at(ctx,p.position), "cant stamp closure %v, must expand it first", p).debug(16)
    return
}
func (p *closure) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*closure); y {
        if p == a { return cmpEqual } else
        if res = p.x.cmp(ctx, a.x); res == cmpEqual && len(p.a) == len(a.a) {
            for i, t := range p.a {
                if res = t.cmp(ctx, a.a[i]); res != cmpEqual { return }
            }
            return
        }
    } else if l, y := v.(*List); y && len(l.Elems) == 1 {
        return p.cmp(ctx, l.Elems[0])
    } else if u, y := v.(unexpanded); y && u.Value != nil {
        return p.cmp(ctx, u.Value)
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
func (p *selection) Strval(ctx Context) (s string) {
    if n, ok := p.o.(*ProjectName); ok && n != nil {
        ctx = closureWith(ctx, n.project.scope)
    }
    if v := p.value(ctx); !isNil(v) {
        s = v.Strval(ctx)
    }
    return
}
func (p *selection) True(ctx Context) (t bool) {
    if v := p.value(ctx); !isNil(v) { t = v.True(ctx) }
    return
}
func (p *selection) Integer(ctx Context) (i int64, e error) {
    if s := p.Strval(ctx); s != "" {
        i, e = strconv.ParseInt(s, 10, 64)
    }
    return
}
func (p *selection) Float(ctx Context) (f float64, e error) {
    if s := p.Strval(ctx); s != "" {
        f, e = strconv.ParseFloat(s, 64)
    }
    return
}
func (p *selection) refs(ctx Context, v Value) bool { return p.o.refs(ctx, v) || p.s.refs(ctx, v) }
func (p *selection) defs(ctx Context, s ...string) []*def {
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
func (p *selection) object(ctx Context) (o Object) {
    if s, ok := p.o.(*selection); ok {
        if v := s.value(ctx); v == nil {
            // sth's wrong!
        } else if o, _ = v.(Object); o == nil {
            erro(at(ctx,p.position), "selection.object: `%s` is nil", s.String())
        }
    } else if o, ok = p.o.(Object); !ok {
        erro(at(ctx,p.position), "selection.object: '%v' is not object (but %s)", p.o, typeof(p.o))
    }
    return
}
func (p *selection) value(ctx Context) (v Value) {
    var o Object
    if isNil(p.s) {
        erro(at(ctx,p.position), "selection prop is nil: %s", p.String()).debug(1)
    } else if o = p.object(ctx); o != nil {
        /*if n, ok := o.(*ProjectName); ok && n != nil && n.project != nil {
            defer setclosure(setclosure(cloctx.unshift(n.project.scope)))
        }*/
        var (
            s = p.s.Strval(ctx)
            optional = strings.HasSuffix(s, "?")
        )
        if optional { s = strings.TrimSuffix(s, "?") }
        if pn, ok := o.(*ProjectName); ok && (p.t == token.SELECT_PROG1 || p.t == token.SELECT_PROG2) {
            if entries := pn.project.resolveEntries(ctx, s, false, false); entries == nil {
                if optional {
                    v = unresolved(pn.project, MakeBareword(p.s.Position(), s))
                } else {
                    erro(at(ctx,p.position), "selection.value: no entry `%s` (%+v)", s, p.String())
                }
            } else {
                v = entries
            }
        } else {
            var e error
            if v, e = o.Get(ctx, s); e != nil {
                erro(ctx, "%v", e).debug(1)
            } else if optional && isNil(v) {
                v = unresolved(pn.project, MakeBareword(p.s.Position(), s))
            }
        }
    } else /*if o == nil*/ {
        erro(at(ctx,p.position), "selection.value: nil object `%s`", p.String())
    }
    return
}
func (p *selection) expandible(ctx Context, w facet) (res bool) {
    if res = w&expandSelection != 0; !res {
        res = p.o.expandible(ctx, w) || p.s.expandible(ctx, w)
    }
    return
}
func (p *selection) expand(ctx Context, w facet) (res Value) {
    if res = p; w&expandSelection != 0 {
        return p.value(ctx)
    } else if isNil(p.o) {
        return
    } else if isNil(p.s) {
        return
    }

    if o, s := p.o.expand(ctx,w), p.s.expand(ctx,w); o != p.o || s != p.s {
        res = &selection{p.valbase, p.t, o, s}
    }
    return
}
func (p *selection) traverse(ctx Context) (traves travestates) {
    ctx = at(ctx, p.position)
    if val := p.value(ctx); isTrivial(val) {
        warn(ctx, "selected value '%v' is trivial", p).debug(1)
    } else {
        _ = val.updated(ctx) // NOTE: ensure that updated flag is correct (see RuleEntry.updated)
        traves = val.traverse(ctx)
    }
    return
}
func (p *selection) updated(ctx Context, v ...bool) (res bool) { // NOTE: this seems not affecting the result
    if val := p.value(ctx); isTrivial(val) {
        warn(ctx, "selected value '%v' is trivial", p).debug(1)
    } else {
        res = val.updated(ctx, v...)
    }
    return res
}
func (p *selection) updatedDeps(ctx Context, v ...Value) (res []Value) {  // NOTE: this seems not affecting the result
    if val := p.value(ctx); isTrivial(val) {
        warn(ctx, "selected value '%v' is trivial", p).debug(1)
    } else {
        res = val.updatedDeps(ctx, v...)
    }
    return res
}
func (p *selection) stat(ctx Context) (si *statinfo) {
    erro(at(ctx,p.position), "cant stat selection %v, must expand it first", p).debug(1)
    return
}
func (p *selection) stamp(ctx Context) (file []*File, err error) {
    erro(at(ctx,p.position), "cant stamp selection %v, must expand it first", p).debug(1)
    return
}
func (p *selection) delete(ctx Context) (file []*File, err error) {
    erro(at(ctx,p.position), "cant stamp selection %v, must expand it first", p).debug(1)
    return
}
func (p *selection) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*selection); ok {
        if p.t == a.t {
            if res = p.o.cmp(ctx, a.o); res == cmpEqual {
                if res = p.s.cmp(ctx, a.s); res == cmpEqual {
                    // if p.t == a.t { res = cmpEqual }
                }
            }
        }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}
func (p *selection) elemStr(ctx Context, o Object, k elemkind) (s string) {
    if _, ok := p.o.(*uselist); ok { s = "usee" } else {
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
func (p *PercPattern) Strval(ctx Context) (s string) {
    if p.Prefix != nil { s = p.Prefix.Strval(ctx) }
    s += "%"
    if p.Suffix != nil { s += p.Suffix.Strval(ctx) }
    return
}
func (p *PercPattern) refs(ctx Context, v Value) bool { return p.Prefix.refs(ctx, v) || p.Suffix.refs(ctx, v) }
func (p *PercPattern) defs(ctx Context, s ...string) []*def { return append(p.Prefix.defs(ctx, s...), p.Suffix.defs(ctx, s...)...) }
func (p *PercPattern) expandible(ctx Context, w facet) bool { return p.Prefix.expandible(ctx, w) || p.Suffix.expandible(ctx, w) }
func (p *PercPattern) expand(ctx Context, w facet) (res Value) {
    var (
        prefix Value
        suffix Value
    )
    if p.Prefix != nil { prefix = p.Prefix.expand(ctx, w) }
    if p.Suffix != nil { suffix = p.Suffix.expand(ctx, w) }
    if prefix != p.Prefix || suffix != p.Suffix {
        res = &PercPattern{ p.valbase, prefix, suffix }
    } else {
        res = p
    }
    return
}
func (p *PercPattern) patterned(ctx Context) bool { return true }
func (p *PercPattern) match1(ctx Context, rep string) (full bool, result string, stems []string) {
    var prefix string
    if !isTrivial(p.Prefix) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if prefix = p.Prefix.Strval(ctx); strings.HasPrefix(rep, prefix) {
            result = prefix
        } else {
            return
        }
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
            } else if s := pp.Prefix.Strval(ctx); s != "" {
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
            } else if s := pp.Suffix.Strval(ctx); s != "" && strings.HasSuffix(rep[a:], s) {
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
            warn(of(ctx,p.Suffix), "mixing % pattern might have performance impact: %v", p).debug(1)
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
        if s := p.Suffix.Strval(ctx); strings.HasSuffix(rep[a:], s) {
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
        return p.match1(ctx, t.Strval(ctx))
    default:
        unreachable(fmt.Sprintf("perc.match: %T %v", i, i))
    }
    return
}
func (p *PercPattern) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var vals []Value
    if isTrivial(p.Prefix) {
        // does nothing
    } else if p.Prefix.patterned(ctx) {
        erro(of(ctx,p.Prefix), "patterned prefix: %T %v", p.Prefix, p.Prefix).debug(1)
        return
    } else {
        vals = append(vals, p.Prefix)
    }

    if len(stems) > 0 {
        if s := stems[0]; s != "" { vals = append(vals, MakeBareword(p.position, s)) }
        rest = stems[1:]
    } else {
        // return
    }

    var suffix Value
    /*
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
    }*/
    if isTrivial(p.Suffix) {
        goto DoneVals
    } else if pp, ok := p.Suffix.(*PercPattern); ok && isTrivial(pp.Prefix) {
        if isTrivial(pp.Suffix) {
            goto DoneVals
        } else {
            // NOTE: patterns like '...%%...' uses only one stem,
            suffix = pp.Suffix
        }
    } else {
        suffix = p.Suffix
    }

    if isTrivial(suffix) {
        goto DoneVals
    } else if suf, res := suffix.stencil(ctx, rest); !isNil(suf) && suf != suffix {
        // NOTE: patterns like '...%xxx%...' use multiple stems.
        vals, rest = append(vals, suf), res
    } else {
        vals, rest = append(vals, suffix), res
    }

DoneVals:
    if n := len(vals); n == 1 {
        val = vals[0]
    } else if n > 1 {
        val = MakeBarecomp(p.position, vals...)
    } else {
        val = p
    }
    return
}
func (p *PercPattern) traverse(ctx Context) (traves travestates) { return traverse(ctx, p, "") }
func (p *PercPattern) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*PercPattern); ok {
        if p.Prefix.cmp(ctx, a.Prefix) == cmpEqual {
            if p.Suffix.cmp(ctx, a.Suffix) == cmpEqual {
                res = cmpEqual
            }
        }
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
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
func (p *GlobPattern) Strval(ctx Context) (s string) {
    for _, comp := range p.Components { s += comp.Strval(ctx) }
    return
}
func (p *GlobPattern) refs(ctx Context, v Value) (res bool) {
    for _, comp := range p.Components {
        if res = comp.refs(ctx, v); res { break }
    }
    return
}
func (p *GlobPattern) defs(ctx Context, s ...string) (res []*def) {
    for _, comp := range p.Components {
        res = append(res, comp.defs(ctx, s...)...)
    }
    return
}
func (p *GlobPattern) expandible(ctx Context, w facet) (res bool) {
    for _, comp := range p.Components {
        if res = comp.expandible(ctx, w); res { break }
    }
    return
}
func (p *GlobPattern) expand(ctx Context, w facet) (res Value) {
    var components, u, n = w.expand(ctx, p.Components...)
    if n > 0 { res = &GlobPattern{p.valbase, components} } else { res = p }
    if u > 0 { res = unexpanded{res} }
    return
}
func (p *GlobPattern) patterned(ctx Context) bool { return true }
func (p *GlobPattern) match(ctx Context, i interface{}) (full bool, result string, stems []string) {
    var s string
    switch t := i.(type) {
    case string:    s = t
    case *File:     s = t.name
    case *filestub: s = t.name
    case Value:     s = t.Strval(ctx)
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
func (p *GlobPattern) traverse(ctx Context) (traves travestates) { return traverse(ctx, p, "") }
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
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}

// TODO: implement regexp pattern
type RegexpPattern struct { valbase }
func (p *RegexpPattern) String() string { return "{RegexpPattern}" }
func (p *RegexpPattern) Strval(ctx Context) (s string) { return "" }
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
    } else if l, ok := v.(*List); ok && len(l.Elems) == 1 {
        res = p.cmp(ctx, l.Elems[0])
    } else if u, o := v.(unexpanded); o && u.Value != nil {
        res = p.cmp(ctx, u.Value)
    }
    return
}
func (p *RegexpPattern) traverse(ctx Context) (traves travestates) { return traverse(ctx, p, "") }
func (p *RegexpPattern) expand(_ Context, _ facet) Value { return p }

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
    Execute(ctx Context, args... Value) (result []Value, traves travestates)
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

func mergeBare(args... Value) (elems []Value) {
    for _, arg := range args {
        if l, o := arg.(*Barecomp); o && l != nil {
            elems = append(elems, mergeBare(l.Elems...)...)
        } else if l, o := arg.(*List); o && len(l.Elems) == 1 {
            elems = append(elems, mergeBare(l.Elems...)...)
        } else if u, o := arg.(unexpanded); o && u.Value != nil {
            elems = append(elems, mergeBare(u.Value)...)
        } else {
            elems = append(elems, arg)
        }
    }
    return
}

// Merge lists recursively into a single list. Previously called Join.
func merge(args... Value) (elems []Value) {
    for _, arg := range args {
        if arg == nil {
            // continue
        } else if l, o := arg.(*List); o && l != nil {
            elems = append(elems, merge(l.Elems...)...)
        } else if u, o := arg.(unexpanded); o && u.Value != nil {
            elems = append(elems, merge(u.Value)...)
        } else {
            elems = append(elems, arg)
        }
    }
    return
}

func mergex(ctx Context, w facet, values ...Value) (res []Value) {
    res, _, _ = w.expand(ctx, values...)
    return merge(res...)
}

func trueVal(ctx Context, v Value, i bool) (res bool) {
    if res = i; v != nil { res = v.True(ctx) }
    return
}

func int64Val(ctx Context, v Value, i int64) (res int64, err error) {
    if res = i; v != nil {
        if i, err = v.Integer(ctx); err == nil { res = i }
    }
    return
}

func intVal(ctx Context, v Value, i int) (res int, e error) {
    if res = i; v != nil {
        var t int64
        if t, e = v.Integer(ctx); e == nil { res = int(t) }
    }
    return
}

func uintVal(ctx Context, v Value, i uint32) (res uint32, e error) {
    if res = i; v != nil {
        var t int64
        if t, e = v.Integer(ctx); e== nil { res = uint32(t) }
    }
    return
}

func permVal(ctx Context, v Value, i uint32) (res os.FileMode) {
    i, _ = uintVal(ctx, v, i)
    res = os.FileMode(i) & os.ModePerm
    return
}

func (w facet) expand(ctx Context, values ...Value) (elems []Value, u, n int) {
    for _, elem := range values {
        if elem == nil { continue } // TODO: report nil expand ??
        if val := elem.expand(ctx, w); val == nil {
            errostack(ctx, 5, "nil: %T %v", elem, elem).debug(10)
            return
        } else if elems = append(elems, val); val != elem {
            if v, y := val.(unexpanded); y {
                if u += 1; v.Value != elem { n += 1 }
            } else if true {
                if val != elem { n += 1 }
            } else if elem.cmp(ctx, val) != cmpEqual {
                n += 1
            }
        } else if _, y := val.(unexpanded); y {
            u += 1
        }
    }
    return
}

func Expand(ctx Context, values ...Value) (res []Value) {
    res, _, _ = plain.expand(ctx, values...)
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
func MakeBarecomp(pos Position, elems... Value) *Barecomp { return &Barecomp{valbase{pos},elements{merge(elems...)}} }
func MakeCompound(pos Position, elems... Value) *Compound { return &Compound{valbase{pos},elements{merge(elems...)}} }
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
