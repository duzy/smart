//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "bytes"
    "crypto/sha256"
    "errors"
    "fmt"
    "hash/maphash"
    "io/fs"
    "math"
    "net/url"
    "os"
    "path/filepath"
    "reflect"
    "regexp"
    "runtime"
    "runtime/debug" // debug.PrintStack()
    "strconv"
    "strings"
    "sync"
    "time"
    "unicode/utf8"
)

type (
    hashBytes [sha256.Size]byte
)

const (
    recursiveTraversalClosurePre = false
    recursiveTraversalClosurePost = false
    recursiveTraversalClosure = true
)

const (
    enable_assertions   = true
    enable_grep_bench   = true
    positionalValueCtx  = true
    traverseDetectLoops = true // turn on/off traverse loop detection
    traverseLoopBreakState   = traveUnkn // eg traveNext or traveDone
    traverseArgumentedExpand = true
)

const (
    cmpUnknown cmpres =  0
    cmpLprefix        = -2 // L is prefix of R, should also be 'smaller'
    cmpSmaller        = -1 // meaningless so far
    cmpGreater        =  1 // meaningless so far
    cmpRprefix        =  2 // R is prefix of L, should also be 'greater'
    cmpEqual          =  3
)

const (
    existenceMatterless existence = 1<<iota
    existenceConfirmed
    existenceNegated
)

const (
    nonePart partialBit = 1<<iota
    digitalPart
    placeholderPart
)

const (
    cacheZero int = 0
    cacheStore = 1<<(iota-1) // NOTE: iota is 2 here
    cacheKey
    cacheMatchPatts
    cacheNoConflict
    cacheBare
    cachePath
    cacheFile
    cacheGlob
    cacheRegex
    cacheUnwind
)

const (
    KindUnclassified Kind = 0
    KindUndef = 1<<iota
    KindDelegate
    KindClosure
    KindArgumented
    KindReturner
    KindNone
    KindNull
    KindAny
    KindEscaped
    KindBoolean
    KindList
    KindGroup
    KindFlag

    KindInteger
    KindBinary
    KindOctal
    KindDecimal
    KindHexadecimal

    KindFloat
    KindRaw
    KindStrLit
    KindStrVal // aka intermediate strlit
    KindBareword
    KindBarefile
    KindCompound
    KindDateTime
    KindDate
    KindTime

    KindPair
    KindPath
    KindUrl
    KindBarecomp
    KindCondval
    KindGlobpat

    KindObject
    KindKnownObject
    KindUnexpanded
    KindUndetermined
    KindExpanded
    KindSelected
    KindSkipped
    KindSelf
    Kindproject
    KindBuiltin
    KindAuto
    KindDef
    KindRule
    KindStemmedRule

    KindModifier
    KindModification

    KindUse
    KindUseList

    KindNumber = KindBoolean|KindInteger|KindFloat
    KindComp = KindPath|KindUrl|KindBarecomp
    // TODO: KindObject = ...
)

type Kind uint64

type cmpres int
func (n cmpres) String() (s string) {
    switch n {
    case cmpUnknown: s = "unknown"
    case cmpLprefix: s = "lprefix"
    case cmpRprefix: s = "rprefix"
    case cmpSmaller: s = "smaller"
    case cmpGreater: s = "greater"
    case cmpEqual:   s = "equal"
    }
    return
}

type existence int
func (n existence) String() (s string) {
    switch n {
    case existenceMatterless: s = "matterless"
    case existenceConfirmed:  s = "confirmed"
    case existenceNegated:    s = "negated"
    }
    return
}

func bitdo(ctx Context, a []interface{}, prop, bits property) interface{} {
    if prop&bits == 0 {
        if ctx == nil { return nil }
        return ctx.do(prop, a...)
    } else {
        return true
    }
}

func originalBits(o origin) (bits property) {
    switch o {
    case _defAny:
    case  defConfig:
    case  defConfDir:
    case  defConfRef:
    case  defDecl:
    case  defExecute: //  !=
    case  defExpand0: //   =
        bits = propExDef|propExDef0
    case  defExpand1: //  :=
        bits = propExDef|propExDef1|propExDisjunction|propExPairVal|propExDelegate
    case  defExpand2: // ::=
        bits = propExDef|propExDef2|propExDisjunction|propExPairVal|propExDelegate|propExClosure
    case  defExpand3: // ;:= (TODO)
        bits = propExDef|propExDef3|propExDisjunction|propExPairVal
    }
    return
}

// Original initiation of def values.
type original struct { Context ; o origin }
func (c original) cast(t reflect.Type) Context { return implcast(c, t) }
func (c original) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, originalBits(c.o))
}

// Optimize value to be most evaluated
type evaluation struct { Context; o origin }
func (c evaluation) cast(t reflect.Type) Context { return implcast(c, t) }
func (c evaluation) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, originalBits(c.o)|propExEvaluation)
}

// Optimize value for final strings
type final struct { Context }
func (c final) cast(t reflect.Type) Context { return implcast(c, t) }
func (c final) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propExClosure|propExDelegate|propExAuto|
        propExPlaceholder|propExDefValue|propExDisjunction|propExPairVal|propExFinal)
}

type expandFullFile struct { Context }
func (c expandFullFile) cast(t reflect.Type) Context { return implcast(c, t) }
func (c expandFullFile) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propExFullFile)
}

type expandPathStr struct { Context }
func (c expandPathStr) cast(t reflect.Type) Context { return implcast(c, t) }
func (c expandPathStr) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propExPathStr)
}

type condless struct { Context }
func (c condless) cast(t reflect.Type) Context { return implcast(c, t) }
func (c condless) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propExCondless)
}

type reversal struct { Context }
func (c reversal) cast(t reflect.Type) Context { return implcast(c, t) }
func (c reversal) do(prop property, a ...interface{}) interface{} {
    return bitdo(c.Context, a, prop, propReversal)
}

type partialBit uint
type partial struct { Context ; bit partialBit }
func (c partial) cast(t reflect.Type) Context { return implcast(c, t) }
func (c partial) do(prop property, a ...interface{}) interface{} {
    for _, v := range a {
        switch t := v.(type) {
        case *auto:
            if true || prop&(propExAuto) != 0 {
                if c.bit&placeholderPart != 0 && t.isPlaceholder() { return true }
                if c.bit&digitalPart != 0 && t.isDigit() { return true }
            }
        case *barecomp:
            for _, e := range t.elems {
                if t := c.do(prop, e); t != nil && t.(bool) { return true }
            }
        case *closure : return c.do(prop, t.x)
        case *delegate: return c.do(prop, t.x)
        }
    }
    if c.Context == nil { return nil }
    return c.Context.do(prop, a...)
}

type negate struct { Context; bits property }
func (c negate) cast(t reflect.Type) Context { return implcast(c, t) }
func (c negate) do(prop property, a ...interface{}) interface{} {
    if prop&c.bits != 0 { return false }
    if c.Context == nil { return nil }
    return c.Context.do(prop, a...)
}

// A Comment node represents a single #-style comment.
type comment struct {
    pos Position // position of "#" starting the comment
    string // comment text (excluding '\n')
}
func (c *comment) String() string { return "{"+c.string+"}" }

// A commentGroup represents a sequence of comments
// with no other tokens and no empty lines between.
type commentGroup struct {
    list []*comment // len(List) > 0
}
func (g *commentGroup) Position() Position { return g.list[0].pos }

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
    for p := si; p != nil; p = p.next {
        if p.file != nil { // matterless is nil file
            if p.file.exists() {
                res = existenceConfirmed
            } else {
                res = existenceNegated
                break
            }
        }
    }
    return
}

var (
    traveTargetNotDefinedFile = fmt.Errorf("target not defined as file")
)

func sfmt(f string, i ...interface{}) string { return fmt.Sprintf(f, i...) }

const ( // larger value higher priority
    traveUnkn travekind = iota
    traveObj  // found object
    traveRule // found rule
    traveFile // exists file
    traveNext // (cond ...), (case ...), unfit patterns
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
    prog *program
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
    if !isNull(p.target)    { s = fmt.Sprintf("%s@%s", s, p.target) } //⇒
    if !isNull(p.dependPat) { s = fmt.Sprintf("%s:%s", s, p.dependPat) }
    if !isNull(p.depend)    { s = fmt.Sprintf("%s>%s", s, p.depend) } //⇒
    if false && p.pos.IsValid() { s = fmt.Sprintf("%s: %s", p.pos, s) }
    if p.error != nil { if e, y := p.error.(*os.PathError); y {
        s += ":" + e.Err.Error()
    } else if e := p.error.Error(); e != "" {
        s += ":" + e
    }}
    return
}

type travestates []*travestate
func (traves travestates) String() (s string) {
    const x = 5
    s = "["
    for i, t := range traves {
        if i > 0 { s += " " }
        s += t.String()
        if i == x && len(traves) > x {
            s += fmt.Sprintf(" …%d…", len(traves)-x)
            break
        }
    }
    s += "]"
    return
}
func (traves *travestates) slice(i int) (res travestates) { return (*traves)[i:] }
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
outter:
    for _, s := range ss {
        for _, t := range *traves {
            if s == t { continue outter }
        }
        res = append(res, s)
    }
    *traves = res
    return traves
}
func (traves *travestates) not(what ...travekind) (res travestates) {
outter:
    for _, s := range *traves {
        for _, w := range what {
            if s.what == w { continue outter }
        }
        res = append(res, s)
    }
    return
}
func (traves *travestates) of(what ...travekind) (res travestates) {
outter:
    for _, s := range *traves {
        for _, w := range what {
            if s.what == w {
                res = append(res, s)
                continue outter
            }
        }
    }
    return
}
func (traves *travestates) my(ctx Context, target Value, what ...travekind) (res travestates) {
outter:
    for _, s := range *traves {
        if !eq(ctx, target, s.target) { continue }
        if what == nil {
            res = append(res, s)
            continue
        }
        for _, w := range what {
            if s.what == w {
                res = append(res, s)
                continue outter
            }
        }
    }
    return
}
func (traves *travestates) unique(ctx Context, what ...travekind) (res travestates) {
outter:
    for _, s := range *traves {
        for _, w := range what {
            if s.what == w {
                for _, s2 := range res {
                    // FIXME: seems not working
                    if s.what == s2.what && (s == s2 || (true && eq(ctx, s.target, s2.target) &&
                        ((s.depend != nil && s2.depend != nil && eq(ctx, s.depend, s2.depend))))) {
                        continue outter
                    }
                } 
                res = append(res, s)
                continue outter
            }
        }
    }
    return
}
func (traves *travestates) add(ctx Context, what travekind, target Value) *travestate {
    if isTrivial(target) { target = autoVal(ctx, "@") }
    for _, s := range *traves {
        if s.what == what && s.target == target {
            return s
        }
    }

    var pos = ctx.Position()
    var s = &travestate{ pos:pos, what:what, target:target, prog: _program(ctx) }
    if *traves = append(*traves, s); false { }
    return s
}
func (traves *travestates) addf(ctx Context, what travekind, s string, a... interface{}) *travestate {
    t := traves.add(ctx, what, nil)
    t.error = fmt.Errorf(s, a...)
    return t
}

type terminal struct { Context ; scopes []*Scope }
func (cc *terminal) String() string {
    if fullContextStringer {
        return fmt.Sprintf("closure{%s}", cc.Context)
    } else {
        return cc.Context.String()
    }
}
func (cc *terminal) Scope() (scope *Scope) {
    if len(cc.scopes) > 0 {
        scope = cc.scopes[0]
    } else {
        scope = cc.Context.Scope()
    }
    return
}
func (cc *terminal) project() *project { return cc.Scope().project }
func (cc *terminal) cast(t reflect.Type) Context { return implcast(cc,t) }
func (cc *terminal) closure() []*Scope { return append(cc.scopes, cc.Context.closure()...) }

func closureprojects(ctx Context) (projects []*project) {
outter:
    for _, scope := range ctx.closure() {
        if proj := scope.project; proj != nil {
            for _, project := range projects {
                if project == proj || project.hasBase(proj) {
                    continue outter
                }
            }
            projects = append(projects, proj)
        }
    }
    return
}

func closureGet(ctx Context, name string) (res *def) {
    for _, scope := range ctx.closure() {
        if scope.project == nil {
            if _, obj := scope.find(name); obj == nil {
                continue
            } else if res, _ = obj.(*def); res != nil {
                return
            }
        } else {
            var pos = ctx.Position()
            if !pos.IsValid() { pos = scope.position }
            if !pos.IsValid() { pos = scope.project.position }
            if scope != scope.project.scope {
                if _, obj := scope.find(name); obj != nil {
                    if res, _ = obj.(*def); res != nil {
                        return
                    }
                }
            }
            if obj := scope.project.resolve(ctx, name); obj == nil {
                if res = autoDef(ctx, name); res != nil { return }
            } else if res, _ = obj.(*def); res != nil {
                return
            }
        }
    }
    return
}

func closureSet(ctx Context, name string, val Value) (prev Value, okay bool) {
    for _, scope := range ctx.closure() {
        if def := scope.FindDef(name); def != nil {
            prev = def.value
            def.val(ctx, val)
            okay = true
            break
        }
    }
    return
}

func closureFiles(ctx Context, name string, one bool) (res []*File) {
    var a = files(ctx, name)
    for _, proj := range closureprojects(ctx) {
        if f := proj.selectFile(ctx, a); f != nil {
            if res = append(res, f); one { break }
        }
    }
    return
}

func closureResolve(ctx Context, name string) (obj Object) {
    var (
        infos = false && strings.HasPrefix(name, "@")
        scope *Scope
    )
    if infos { defer func() {
        var val Value
        if obj != nil { val = obj.expand(ctx) }//, plain
        warn(at(ctx,scope.position), "%v: name = %s", scope.project, name)
        warn(at(ctx,scope.position), "%v: %v", scope.project, scope)
        warn(at(ctx,scope.position), "%v: %v", scope.project, ctx)
        warn(ctx, "%s: %T, %v", scope.project, obj, obj)
        warn(ctx, "%s: %T, %v", scope.project, val, val).debug(24)
    } () }
    for _, scope = range ctx.closure() {
        var ctx Context = at(ctx, scope.position)
        if infos { warn(ctx, "%s", scope).debug(1) }
        if scope.project == nil || scope != scope.project.scope {
            if _, obj = scope.find(name); isNull(obj) {
                // fallthrough
            } else if a, y := obj.(*auto); y { // assert(a.name == name)
                if d := autoDef(ctx, a.ident(ctx)); d != nil { obj = d }
                if infos {
                    var proj = a.owner()
                    var cc = cast[*terminal](ctx)
                    val := obj.expand(cc)//, plain
                    va2 := autoVal(cc, name)
                    ob1 := autoDef(ctx, name)
                    warn(ctx, "%v: %v", proj, a.name)
                    warn(ctx, "%v: obj = %T %v", proj, obj, obj)
                    warn(ctx, "%v: ob1 = %T %v", proj, ob1, ob1)
                    warn(ctx, "%v: val = %T %v", proj, val, val)
                    warn(ctx, "%v: va2 = %T %v", proj, va2, va2)
                    warn(ctx, "%v: %v", proj, scope)
                    warn(ctx, "%v: %v", proj, ctx).debug(1)
                }
                break
            } else {
                break // got the obj
            }
        }
        if scope.project != nil {
            obj = scope.project.resolve(ctx, name)
        }
        if isNull(obj) && false { obj = closureResolve(inner(ctx), name) }
        if!isNull(obj) { if infos { warn(ctx, "%v", obj).debug(1) }; break }
    }
    return
}

func closureEntry(ctx Context, name string) (entries entryArray) {
    for _, scope := range ctx.closure() {
        if project := scope.project; project != nil {
            entries = project.resolveEntries(ctx, name, false)
            if false && entries == nil {
                entries = closureEntry(inner(ctx), name)
            }
        }
        if entries != nil { break }
    }
    return
}

func _closureWith(ctx Context, ii ...interface{}) (res Context) {
    var scopes []*Scope
    for _, i := range ii {
        switch s := i.(type) {
        case   *Scope: scopes = append(scopes, s)
        case *project: scopes = append(scopes, s.scope)
        case   Object: scopes = append(scopes, s.declScope())
        }
    }
    return closureWith(ctx, scopes...)
}

func closureWith(ctx Context, scopes ...*Scope) (res Context) {
    if c, y := ctx.(*terminal); false && y {
        res = closureWith(c.Context, scopes...)
    } else {
        res = &terminal{ ctx, scopes }
    }
    return
}

func refdef(ctx Context, val Value, origin origin) (res bool) {
    for _, def := range val.defs(ctx) {
        if def.origin == origin || origin == _defAny { return true }
        if true && def.value != nil && refdef(ctx, def.value, origin) { return true }
    }
    return
}

func entryIndicator(ctx Context, entry Value) (str, ent, tar string) {
    if !isNull(entry) { ent = entry.string(ctx) }
    if val := autoVal(ctx, "@"); val == nil || isTrivial(val) {
        str = ent // ...
    } else if tar = val.string(ctx); ent != tar {
        str = fmt.Sprintf("%s(%s)", ent, tar)
    } else {
        str = ent
    }
    return
}

func exists(ctx Context, v Value) bool {
    // FIXME: returns true if existenceMatterless ??
    return v != nil && v.stat(ctx).exists() == existenceConfirmed
}

func getHashDir(ctx Context, k []byte) string {
    var dir string
    if program := _program(ctx); program != nil {
        dir = program.project.tmpPath
    } else if project := ctx.project(); project != nil {
        dir = project.tmpPath
    } else if scope := ctx.Scope(); scope != nil && scope.project != nil {
        dir = scope.project.tmpPath
    }
    var h = fmt.Sprintf("%x", k[:2]) // HEX of the first two bytes
    return filepath.Join(dir, ".hash", h[0:1], h[1:2], h[2:3], h[3:])
}

func getRecipesHash(ctx Context, target Value, values ...Value) (k, v hashBytes, err error) {
    var (
        // target = getTargetValue(ctx)
        program = _program(ctx)
        key = sha256.New()
        val = sha256.New()
    )
    fmt.Fprintf(key, "%s", program.project.absPath)
    fmt.Fprintf(key, "%s", program.position)
    fmt.Fprintf(key, "%s", target.string(ctx)) // targetStr

    for _, value := range values {
        if false {
            // FIXME: String() varies when &(var) is used
            fmt.Fprintf(val, "%v", value.string(ctx))
        } else {
            fmt.Fprintf(val, "%v", value)
        }
    }
    copy(k[:], key.Sum(nil))
    copy(v[:], val.Sum(nil))
    return
}

func updateRecipesHash(ctx Context, target Value) (k, v hashBytes, err error) {
    var program = _program(ctx)
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
    var k, v hashBytes
    if program := _program(ctx); program == nil {
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

type waitOpts struct {
    ReportUpdates bool
    ExecResults bool
    StampCurrentTarget bool
}
func wait(ctx Context, opts waitOpts) (target Value, files []*File, execRes *execResult, err error) {
    var calleeErrs []error
    var pc = cast[*programContext](ctx)
    if pc != nil {
        // wait for all jobs done
        if false { pc.WaitGroup.Wait() } // FIXME: deadlock

        pc.calleeErrsM.Lock()
        calleeErrs = pc.calleeErrs; pc.calleeErrs = nil
        pc.calleeErrsM.Unlock()
    }

    if target = getTargetValue(ctx); target == nil {
        erro(ctx, "target is nil")
        errostack(ctx, 8, "").debug(8)
        return
    } else if isTrivial(target) {
        erro(ctx, "trivial target (%T)", target)
        errostack(ctx, 8, "").debug(8)
        return
    } else if n := len(calleeErrs); n > 0 /*&& t.stems == nil*/ {
        var numRealErrs = 0
        for _, err := range calleeErrs {
            erro(ctx, "%v: %v", target, err).debug(1)
            numRealErrs += 1
        }
        if numRealErrs == 0 { return } // simply return if no real errors

        var ctxPos, targetPos = ctx.Position(), target.Position()
        var v = target
        if l, ok := v.(*list); ok && l.len() == 1 { v = l.elems[0] }
        if targetPos.IsValid() && !targetPos.Same(&ctxPos) {
            if f, y := toFile(v); y && f != nil && f.filemap != nil {
                erro(at(ctx,targetPos), "waiting for '%v'", target)
                erro(at(ctx,f.filemap.pattern), "via pattern '%v' (of %v)", v, f.filemap.project).debug(1)
            } else {
                erro(at(ctx,targetPos), "waiting for '%v'", target).debug(1)
            }
        }
        if def, ok := v.(*def); ok && target != v && target != def.value { // trace source Def in diagnostics
            erro(at(ctx,def.value), "waiting for def '%v': %v", def.name, def.value).debug(1)
        }
        return
    }

    if opts.ExecResults {
        // Waiting for command (shell/python/etc.) exec result
        if val := autoVal(ctx, "-"); val != nil {
            var ok bool
            if execRes, ok = val.(*execResult); ok {
                //execRes.wg.Wait()
            }
        }
    }
    if !opts.StampCurrentTarget {
        // done!
    } else if files, err = target.stamp(ctx); err != nil {
        if p := target.Position(); p.IsValid() { erro(at(ctx,p), "%v", err) }
        erro(ctx, "%v", err).debug(1)
        return
    } else if opts.ReportUpdates {
        reportFileUpdates(ctx, files)
    }
    return
}

type as struct { Value }
func (a as) file(ctx Context, projects ...*project) (f *File) {
    defer func() { if f == nil {
        var s = a.string(ctx)
        if v, t := a.Value, file(ctx, s); t != nil {
            var ( p = ctx.project() ; ctx = at(ctx, v) )
            for i, m := range files(ctx, t, projects...) {
                erro(ctx, "FIXME: %v: %d. %v", p, i, m)
            }
            erro(ctx, "FIXME: %v: %v (%T, %v)", p, v, v, projects)
            erro(ctx, "FIXME: %v: %v (%s)", p, t, t.fullname())
            errostack(ctx, 5).debug(32)
        }
    }} ()

    switch t := a.Value.(type) {
    case  as       : f = t.file(ctx, projects...)
    case  fullfile : f = t.File
    case *barefile : f = t.File
    case *File     : f = t
    case *def      : if !isTrivial(t.value) { return as{t.value   }.file(ctx) }
    case *list     : if len(t.elems) == 1   { return as{t.elems[0]}.file(ctx) }
    case *rule:                               return as{t.target  }.file(ctx)
    case *strlit, *compound:
        // NOTE: escape 'string' and "compound" values from file parsing,
        // NOTE: this optimized the performance.
    case *bareword, *barecomp, *path:
        if len(projects) == 0 { projects = closureprojects(ctx) }
        {
            b := builtin_file{}
            b.evocation = &evocation{Context:ctx}
            if v := b.z(projects, t); 0 < len(v) { f, _ = v[0].(*File) }
        }
    }
    return
}
func (a as) fullnameFile(ctx Context, projects ...*project) (f *File, s string, ok bool) {
    if f = a.file(ctx, projects...); f == nil {
        // no fullname
    } else if s = f.fullname(); filepath.IsAbs(s) {
        ok = true
    } else {
        // s = ""
    }
    return
}
// DEPRECATED
func (a as) fullnameOrFinal(ctx Context, projects ...*project) (s string, y bool) {
    if _, s, y = a.fullnameFile(ctx, projects...); !y { s = a.string(ctx) }
    return
}
func (a as) fullname(ctx Context, projects ...*project) (o fullname, y bool) {
    if v := a.Value; v == nil {
        return
    } else {
        v = scalarize(v)
        if f, _ := v.(*File); f == nil {
            if f = a.file(ctx, projects...); f != nil {
                o.Value = f ; return o, true
            }
        }
    }
    return
}

// joinPath is different from filepath.Join, which trims and discards empty segments
func joinPath(segs ...string) string { return strings.Join(segs, pathSep) }
func joinPathStr(ctx Context, i interface{}) (str string) {
    switch s := i.(type) {
    case      nil:
    case   string: str = s
    case []string: str = joinPath(s...)
    default:
        warn(ctx, "unexpected path str: %v", us(i)).debug(6)
    }
    return
}

func joinRaws(sep string, vals ...*raw) string {
    var strs []string
    for _, v := range vals { strs = append(strs, v.String()) }
    return strings.Join(strs, sep)
}

type valcache_kv struct { _key Value ; _val interface{} }
type valcache struct {
    valcache_kv
    _fix, fast map[string]*valcache
}

func (cache *valcache) String() (s string) {
    var comma bool
    if cache._key == nil && cache._val == nil {
        s = "{"
    } else {
        s = fmt.Sprintf("{%v=%v{%v}", cache._key, typeof(cache._val), cache._val)
        comma = true
    }

    if l := len(cache._fix); l > 0 {
        if comma { s += ", " }
        s += "FIX:{"
        comma = false
        for k, v := range cache._fix {
            if comma { s += ", "}
            s += fmt.Sprintf("%v:%v", k, v)
            comma = true
        }
        s += "}"
        comma = true
    }
    if l := len(cache.fast); l > 0 {
        if comma { s += ", " }
        s += "FAST:{"
        comma = false
        for k, v := range cache.fast {
            if comma { s += ", "}
            s += fmt.Sprintf("%v:%v", k, v)
            comma = true
        }
        s += "}"
    }
    s += "}"
    return
}

func (cache *valcache) slot(ctx Context, val Value, bits int) (res *valcache) {
    if false { if _, y := val.(*compound); !y { info(ctx, "cache: %T %v %08b", val, val, bits) }}
    if cache == nil { return }

    res = val.cache(ctx, cache, bits&^cacheKey)

    if bits&cacheKey != 0 && bits&cacheStore != 0 && res != nil {
        if res._key == nil { res._key = val } else
        if res._key.cmp(ctx, val) != cmpEqual { a, b, v := res._key, val, val.expand(ctx)//, final
            errostack(ctx, 5, "conflict cache: %v , %v ; %v", us(a), us(b), v).debug(32)
            return
        }
    }
    return
}

func (cache *valcache) strx(ctx Context, str string, bits int) (res *valcache) {
    if cache == nil { return }

    for _, s := range strings.Split(str, pathSep) {
        if cache = cache.str(ctx, s, bits); cache == nil { return }
    }
    return cache
}

func (cache *valcache) str(ctx Context, s string, bits int) (res *valcache) {
    if cache._fix != nil && bits&cacheMatchPatts != 0 {
        defer func() {
            if res == nil && bits&cacheMatchPatts != 0 {
                res = cache.matchPatts(ctx, s)
            }
        }()
    }

    if y := cache.fast == nil; y {
        if bits&cacheStore != 0 {
            res = &valcache{}
            cache.fast = make(map[string]*valcache)
            cache.fast[s] = res
        }
    } else if res, y = cache.fast[s]; !y && bits&cacheStore != 0 {
        res = &valcache{}
        cache.fast[s] = res
    }
    return
}

func (cache *valcache) fix(ctx Context, s string, bits int) (res *valcache) {
    if y := cache._fix == nil; y {
        if bits&cacheStore != 0 { res = &valcache{}
            cache._fix = make(map[string]*valcache)
            cache._fix[s] = res
        }
    } else if res, y = cache._fix[s]; !y && bits&cacheStore != 0 {
        res = &valcache{}
        cache._fix[s] = res
    }
    return
}

func (cache *valcache) matchPatts(ctx Context, s string) (res *valcache) {
    var step = func(k, s string, c *valcache) (r *valcache, tail string) {
        if k == "%" {
            var empty *valcache
            for k, v := range c._fix {
                if k == "" { empty = v } else
                if strings.HasSuffix(s, k) { return c, "" }
            }
            if empty != nil { return empty, "" }
        } else if k == "?" { // match one char
            if s != "" { return c, s[1:] }
        } else if k == "*" || k == "**" { // match many chars
            var empty *valcache
            for k, v := range c._fix {
                if k == "" { empty = v } else
                if i := strings.Index(s, k); i >= 0 { return v, s[i+len(k):] }
            }
            if empty != nil { return empty, "" }
        } else if k == "[]" { // match range of chars
            errostack(ctx, 3, "%v, %s", *c, s).debug(32)
        } else if strings.HasPrefix(s, k) {
            return c, s[len(k):]
        }
        return
    }

FixAgain:
    for k, c := range cache._fix {
        if c, t := step(k, s, c); c != nil {
            if t == "" {
                return c
            } else {
                cache, s = c, t
                goto FixAgain // NOTE: restart loop, not 'continue'
            }
        }
    }
    return
}

func (cache *valcache) collect(ctx Context, pat Value) (res []*valcache) {
    erro(ctx, "%v, %v %v, %v", us(pat), cache._key, cache._fix, cache.fast).debug(1)
    return
}

func srclit(o Object, elem Value) (s string) {
    if p, y := elem.(interface{ srclit(Object) string }); y {
        s = p.srclit(o)
    } else if elem != nil {
        s = elem.String()
    }
    return
}

// Value represents a value of a type.
type Value interface {
    positioner // The position where the value appears (or NoPos).

    kind() Kind

    // Literal representations of the value.
    String() string

    // Final returns the string form of the value.
    string(Context) string

    ident(Context) string // aka name

    // Returns true if the value can be evaluated as 'true', 'yes', etc.
    true(Context) bool

    // Integer returns the integer form of the value.
    int(Context) (int64, error)

    // Float returns the float form of the value.
    float(Context) (float64, error)

    // Equality compare.
    cmp(Context, Value) cmpres

    // whether this value can be used as a pattern
    patterned(Context) bool

    // Match a Value or string, returned 's' is the matched string (or heading part).
    match(Context, interface{}) (full bool, s interface{}, stems []string)

    // Stencil this value with stems.
    stencil(Context, []string) (val Value, rest []string)

    // Recursively detecting whether this value references
    // the object (to avoid loop-delegation).
    refs(Context, Value) bool

    // Returns all defs (of names if specified) used in this value.
    defs(Context, ...string) []*def

    // Test if this value is expandable for some bits.
    expandable(Context) bool
    expand(Context) Value // result is nil or identical to this value if no expansions

    cache(ctx Context, cache *valcache, bits int) (res *valcache)
    collect(ctx Context, cache *valcache, bits int) (res []*valcache)

    stat(Context) (*statinfo)

    // Stamp the value if it's a file (aka. update FileInfo).
    stamp(Context) ([]*File, error)

    // Delete the file (if it is).
    delete(Context) ([]*File, error)

    updated(Context) bool
    updatedDeps(Context, ...Value) []Value

    traverse(Context)
}

func typeof_0(arg interface{}) (s string) {
    switch a := arg.(type) {
    case *list:
        if n := len(a.elems); n == 1 {
            switch v := a.elems[0].(type) {
            case *delegate: // FIXME: recursively undelegate types
                if d, y := v.x.(*def); y && d != nil {
                    s = fmt.Sprintf("$%v", typeof(d.value))
                } else {
                    s = fmt.Sprintf("$%v", typeof(v.x))
                }
            default:
                s = fmt.Sprintf("%T", v) //s = v.Type().String()
            }
        } else if n > 1 {
            s = "list"
        } else if false {
            s = "none"
        }
    default:
        // FIXME: this should be an exception (panic).
        s = fmt.Sprintf("%T", a) //s = a.Type().String()
    }
    if s != "" {
        s = strings.TrimPrefix(s, "*")
        s = strings.TrimPrefix(s, "smart.")
        if false { s = strings.TrimPrefix(s, "ast.") }
        // s = strings.ReplaceAll(strings.TrimPrefix(s, "*"), "smart.", "")
    }
    return
}

func typeof(arg interface{}) (s string) {
    defer func() { if s == "" { panic(fmt.Sprintf("typeof(%T) is empty", arg)) } } ()

    if arg == nil { return "nil" }
    if a, y := arg.(*list); y {
        if a.len() == 1 {
            if t, y := a.elems[0].(*delegate); y {
                if d, y := t.x.(*def); y && d != nil {
                    return typeof(d.value)
                } else {
                    return typeof(t.x)
                }
            }
            return typeof(a.elems[0])
        } else if a.len() > 1 {
            return "list"
        } else if false {
            return "none"
        }
    }

    switch t := reflect.TypeOf(arg); t.Kind() {
    case reflect.Array: return "[]"+t.Elem().Name()
    case reflect.Slice: return "[]"+t.Elem().Name()
    case reflect.Ptr:   return      t.Elem().Name()
    default:            return      t.Name()
    }
}

func is(v Value, i interface{}) bool {
    switch t := i.(type) {
    case Kind:         return v.kind() & t != 0
    case reflect.Type: return reflect.TypeOf(v) == t
    }
    return reflect.TypeOf(v) == reflect.TypeOf(i)
}

func cmp(ctx Context, l, r Value) (res cmpres) {
    if  res = l.cmp(ctx, r); res == cmpUnknown {
        res = r.cmp(ctx, l)
        if res != cmpUnknown && res != cmpEqual {
            res = cmpres(-res)
        }
    }
    return
}

func eq(x Context, a, b Value) bool { return a == b || a.cmp(x, b) == cmpEqual }
func equal(ctx Context, a, b Value, s ...bool) bool {
    if eq(ctx, a, b) { return true }

    var y bool
    for _, y = range s { if y { break } }
    return y && a.string(ctx) == b.string(ctx)
}

func diff(ctx Context, a, b []Value) bool {
    if len(a) == len(b) {
        for i, v := range a { if !equal(ctx, v, b[i]) { return true } }
        return false
    } else {
        return true
    }
}

func indeterminate(ctx Context, val Value) bool {
    var f, y = ctx.(final)
    if !y { f.Context = ctx }
    return val.expandable(f)
}

func isFinalValue(ctx Context, val Value) bool {
    return !indeterminate(ctx, val)
}

// aka. isNull(v) || isNone(v) || isUndef(v) || isEmpty(v)
func isTrivial(v Value) (t bool) {
    switch a := v.(type) {
    case *none, *null, *undef: t = true
    case *list: t = len(a.elems) == 0 || (len(a.elems) == 1 && isTrivial(a.elems[0]))
    case *strlit: t = a.s == ""
    case fullname: t = isTrivial(a.Value)
    case as: t = isTrivial(a.Value)
    default: t = isNull(v)
    }
    return
}
func isEmptyList(v Value) (t bool) {
    if l, y := v.(*list); y && len(l.elems) == 0 { t = true }
    return
}
func isEmpty(v Value) (t bool) {
    switch a := v.(type) {
    case *none, *null, *undef: t = true
    case *strlit: t = a.s == ""
    case *list:
        var n = len(a.elems)
        t = n == 0 || n == 1 && isEmpty(a.elems[0])
    }
    return
}
func isUndef(v Value) (t bool) {
    switch a := v.(type) {
    case *def: t = a.value != nil && isUndef(a.value)
    case *undef: t = true
    }
    return
}
func isNone(v Value) (t bool) {
    switch a := v.(type) {
    case *none: t = true
    case *list: t = len(a.elems) == 0 ||
        (len(a.elems) == 1 && (isNone(a.elems[0]) || isNull(a.elems[0])))
    }
    return
}
func isNull(v Value) (t bool) { // aka is(v, &null{})
    if v == nil {
        t = true
    } else if _, t = v.(*null); t {
        // true
    } else if vv := reflect.ValueOf(v); vv.Kind() == reflect.Ptr && vv.IsNil() {
        t = true
    }
    return
}

type i_prefix interface{ prefix(Context, Value) Value }
type i_suffix interface{ suffix(Context, Value) Value }

func _bifix(ctx Context, x, y Value) Value {
    switch t := y.(type) {
    case *barecomp: return t.prefix(ctx, x)
    case     *pair: return t.prefix(ctx, x)
    }
    return makeBarecomp(x).suffix(ctx, y)
}

func _suffix(ctx Context, x, y Value) Value {
    switch _x := x.(type) { case i_suffix: return _x.suffix(ctx, y) }
    return makeBarecomp(x).suffix(ctx, y)
}

func compose(ctx Context, x, y Value) (res Value) {
    switch _x := x.(type) { case i_suffix: return _x.suffix(ctx, y) }
    switch _y := y.(type) { case i_prefix: return _y.prefix(ctx, x) }

    // if _, t := y.(*path); t {
    // 	switch x.(type) {
    // 	case  flag: // okay: -Ifoo/bar, -Lfoo/bar
    // 	case *path: // okay: combine two paths
    // 	case *barecomp, *strlit, *strval, *compound, *delegate, *closure, *punctuation:
    // 	default: return
    // 	}
    // }

    erro(ctx, "compose: %v %v", us(x), us(y)).debug(3)

    return makeBarecomp(x, y)//.suffix(ctx, y)
}

func hasPrefix(str string, prefixs ...string) (res bool) {
    for _, s := range prefixs { if res = strings.HasPrefix(str, s); res { break }}
    return
}

func hasSuffix(str string, suffixs ...string) (res bool) {
    for _, s := range suffixs { if res = strings.HasSuffix(str, s); res { break }}
    return
}

type valvec []Value
func (vals valvec) has(val Value) (res bool) {
    for _, v := range vals { if res = v == val; res { break } }
    return
}
func (vals valvec) has2(ctx Context, val Value) (res bool) {
    for _, v := range vals { if res = equal(ctx, v, val); res { break } }
    return
}
func (vals *valvec) add(val Value) (res *valvec) {
    if val != nil && !vals.has(val) { *vals = append(*vals, val) }
    return vals
}
func (vals *valvec) add2(ctx Context, val Value) (res *valvec) {
    if val != nil && !vals.has2(ctx, val) { *vals = append(*vals, val) }
    return vals
}

type valbase struct { position Position }
func (_ *valbase) String() (_ string) { return }
func (_ *valbase) cmp(Context, Value) (_ cmpres) { return }
func (_ *valbase) defs(Context, ...string) (_ []*def) { return }
func (_ *valbase) delete(Context) (_ []*File, _ error) { return }
func (_ *valbase) expand(Context) (_ Value) { return }
func (_ *valbase) expandable(Context) (_ bool) { return }
func (_ *valbase) float(Context) (_ float64, _ error) { return }
func (_ *valbase) ident(Context) (_ string) { return }
func (_ *valbase) int(Context) (_ int64, _ error) { return }
func (_ *valbase) kind() Kind { return KindUnclassified }
func (_ *valbase) match(Context, interface{}) (_ bool, _ interface{}, _ []string) { return }
func (_ *valbase) patterned(Context) (_ bool) { return }
func (_ *valbase) refs(Context, Value) (_ bool) { return }
func (_ *valbase) stamp(Context) (_ []*File, _ error) { return }
func (_ *valbase) stat(Context) (_ *statinfo) { return }
func (_ *valbase) stencil(Context, []string) (_ Value, _ []string) { return }
func (_ *valbase) string(Context) (_ string) { return }
func (_ *valbase) traverse(Context) { }
func (_ *valbase) true(Context) (_ bool) { return }
func (_ *valbase) updated(Context) (_ bool) { return }
func (_ *valbase) updatedDeps(Context, ...Value) (_ []Value) { return }
func (p *valbase) Position() Position { return p.position }

type returner struct { valbase ; vals []Value }
func (p *returner) kind() Kind { return KindReturner }
func (p *returner) expand(ctx Context) (res Value) {
    if vals := expand(ctx, p.vals...); diff(ctx, vals, p.vals) {
        res = &returner{p.valbase, vals}
    } else {
        res = p
    }
    return
}
func (p *returner) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b): %v", bits, p.vals).debug(32)
    return
}
func (p *returner) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b): %v", bits, p.vals).debug(32)
    return
}

type undef struct { Value }
func (p undef) kind() Kind { return KindUndef }
func (p undef) String() string { return "{=undef "+p.Value.String()+"}" }
func (p undef) string(Context) (_ string) { return }
func (p undef) int(Context) (_ int64, _ error) { return }
func (p undef) float(Context) (_ float64, _ error) { return }
func (p undef) true(Context) (_ bool) { return }
func (p undef) prefix(_ Context, v Value) Value { return v }
func (p undef) suffix(_ Context, v Value) Value { return v }
func (p undef) expand(_ Context) Value { return p }
func (p undef) cmp(ctx Context, v Value) (_ cmpres) {
    if _, y := v.(*undef); y {
        return cmpEqual
    } else if l, y := v.(*list); y && l.len() == 1 {
        return p.cmp(ctx, l.elems[0])
    } else {
        return
    }
}

func _null(ctx Context) *null { return &null{valbase{ctx.Position()}} }

type null struct { valbase }
func (_ *null) kind() Kind { return KindNull }
func (_ *null) String() string { return "{}" } // {=null}
func (p *null) prefix(_ Context, val Value) Value { return val }
func (p *null) suffix(_ Context, val Value) Value { return val }
func (p *null) expand(Context) Value { return p }
func (p *null) traverse(ctx Context) {
    erro(at(ctx,p), "null traversal").debug(3)
}
func (p *null) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if _, y := v.(*null); y {
        res = cmpEqual
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (_ *null) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if false { errostack(ctx, 5, "cache unsupported: %v", cache).debug(32) }
    return cache // NOTE: for empty flags "-"
}
func (_ *null) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if false { errostack(ctx, 5, "collect unsupported: %v", cache).debug(32) }
    return // NOTE: for empty flags "-"
}

type none struct { valbase ; x Value }
func (_ *none) kind() Kind { return KindNone }
func (p *none) String() (s string) {
    s = "{=none"
    if p.x != nil {
        s += " "
        s += p.x.String()
    }
    s += "}"
    return
}
func (p *none) string(Context) (s string) { return }
func (p *none) true(ctx Context) (res bool) { return }
func (p *none) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if _, ok := v.(*none); ok {
        res = cmpEqual
    } else if s, ok := v.(*strlit); ok && s.s == "" {
        res = cmpEqual
    } else if l, ok := v.(*list); ok && len(l.elems) == 0 {
        res = cmpEqual
    } else if ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *none) prefix(_ Context, val Value) Value { return val }
func (p *none) suffix(_ Context, val Value) Value { return val }
func (p *none) expand(Context) Value { return p }
func (p *none) traverse(ctx Context) {
    erro(at(ctx,p), "none traversal").debug(3)
}
func (_ *none) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    return cache.str(ctx, "", bits)
}
func (_ *none) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if false { errostack(ctx, 5, "collect unsupported: %v", cache).debug(32) }
    return
}

type argumented struct { Value ; args []Value }
func (_ *argumented) kind() Kind { return KindArgumented }
func (p *argumented) String() (s string) { return p.srclit(nil) }
func (p *argumented) srclit(o Object) (s string) {
    for i, a := range p.args {
        if i > 0 { s += "," }
        s += srclit(o, a)
    }
    s = fmt.Sprintf("%s(%s)", srclit(o, p.Value), s)
    return
}
func (p *argumented) string(ctx Context) (s string) {
    s = p.Value.string(ctx) + "("
    for i, a := range p.args {
        if i > 0 { s += "," }
        s += a.string(ctx)
    }
    s += ")"
    return
}
func (p *argumented) refs(ctx Context, v Value) bool {
    if p.Value.refs(ctx, v) { return true }
    for _, a := range p.args { if a.refs(ctx, v) { return true } }
    return false
}
func (p *argumented) defs(ctx Context, s ...string) (res []*def) {
    res = p.Value.defs(ctx, s...)
    for _, a := range p.args { res = append(res, a.defs(ctx, s...)...) }
    return
}
func (p *argumented) expandable(ctx Context) (res bool) {
    if res = p.Value.expandable(ctx); !res {
        for _, a := range p.args {
            if res = a.expandable(ctx); res { break }
        }
    }
    return
}
func (p *argumented) prefix(ctx Context, val Value) Value {
    return &argumented{compose(ctx, val, p.Value), p.args}
}
func (p *argumented) suffix(ctx Context, val Value) Value {
    return &argumented{compose(ctx, p.Value, val), p.args}
}
func (p *argumented) expand(ctx Context) (res Value) {
    var val, args = p.Value.expand(ctx), expand(ctx, p.args...)
    if !equal(ctx, val, p.Value) || diff(ctx, args, p.args) {
        res = &argumented{val, args}
    } else {
        res = p
    }
    return
}
func (p *argumented) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, y := v.(*argumented); y {
        if res = p.Value.cmp(ctx, a.Value); res == cmpEqual {
            // TODO:FIXME: check p.args against a.args?
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *argumented) traverse(ctx Context) {
    //!< IMPORTANT! - Don't merge-expand arguments here!
    //!< Arguments should be passed to program.execute as it is.
    var args []Value
    if traverseArgumentedExpand {
        var proj = ctx.project()
        // NOTE: expand here to avoid args being expanded in the wrong context
        for _, a := range p.args {
            a = a.expand(ctx)//, plain
            // TODO: deal with pattern args using expandPatterned instead of stenciling:
            if true && a.patterned(ctx) { if stems := _stems(ctx); len(stems) > 0 {
                if val, rest := a.stencil(ctx, stems); len(rest) > 0 {
                    erro(at(ctx,a), "partial stencil: %v, %T %v, %v, %v", a, val, val, rest, stems).debug(1)
                    panic(fmt.Sprintf("%T %v", val, val))
                } else if file, okay := toFile(val); okay {
                    a = file
                } else if file := proj.file(ctx, val.string(ctx)); file != nil {
                    a = file
                } else {
                    a = val //makeStrlit(a.Position(), str)
                }
            }}
            args = append(args, a)
        }
    } else {
        args = p.args
    }

    p.Value.traverse(&argumentedContext{ ctx, args })
}
func (_ *argumented) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *argumented) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

// any is used to box an arbitrary value
type any struct { value interface{} }
func (_ *any) kind() Kind { return KindAny }
func (p *any) cmp(ctx Context, v Value) (res cmpres) {
    switch a := v.(type) {
    case *any:
        if p.value == a.value {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            if v2, ok := a.value.(Value); ok {
                res = v1.cmp(ctx, v2)
            }
        } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
            return p.cmp(ctx, l.elems[0])
        }
    case Value:
        if p.value == a {
            res = cmpEqual
        } else if v1, ok := p.value.(Value); ok {
            res = v1.cmp(ctx, a)
        } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
            return p.cmp(ctx, l.elems[0])
        }
    }
    return
}
func (p *any) patterned(ctx Context) (res bool) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        res = v.patterned(ctx)
    }
    return
}
func (p *any) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        full, s, stems = v.match(ctx, i)
    }
    return
}
func (p *any) stencil(ctx Context, stems []string) (val Value, rest []string) {
    if p.value == nil {
        // does nothing
    } else if v, ok := p.value.(Value); ok {
        val, rest = v.stencil(ctx, stems)
    }
    return
}
func (p *any) delete(ctx Context) (files []*File, err error) {
    if a, ok := p.value.(Value); ok { files, err = a.delete(ctx) }
    return
}
func (p *any) updated(ctx Context) (res bool) {
    if p.value == nil {
        // does nothing
    } else if val, ok := p.value.(Value); ok {
        res = val.updated(ctx)
    }
    return
}
func (p *any) updatedDeps(ctx Context, v ...Value) (res []Value) {
    if p.value == nil {
        // does nothing
    } else if val, ok := p.value.(Value); ok {
        res = val.updatedDeps(ctx, v...)
    }
    return
}
func (p *any) stamp(ctx Context) (files []*File, err error) {
    if a, ok := p.value.(Value); ok { files, err = a.stamp(ctx) }
    return
}
func (p *any) stat(ctx Context) (si *statinfo) {
    if v, ok := p.value.(Value); ok && v != nil { si = v.stat(ctx) }
    return
}
func (p *any) expand(ctx Context) Value {
    if v, y := p.value.(Value); y {
        return v.expand(ctx)
    } else {
        return p
    }
}
func (p *any) refs(ctx Context, o Value) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.refs(ctx, o) }
    return
}
func (p *any) defs(ctx Context, s ...string) (res []*def) {
    if v, ok := p.value.(Value); ok { res = v.defs(ctx, s...) }
    return
}
func (p *any) expandable(ctx Context) (res bool) {
    if v, ok := p.value.(Value); ok { res = v.expandable(ctx) }
    return
}
func (p *any) Position() (res Position) {
    if v, ok := p.value.(positioner); ok { res = v.Position() }
    return
}
func (p *any) true(ctx Context) (t bool) {
    switch v := p.value.(type) {
    case Value:     t = v.true(ctx)
    case float32:   t = math.Abs(float64(v))-0 >= FloatEpsilon
    case float64:   t = math.Abs(v)-0 >= FloatEpsilon
    case int64:     t = v != 0
    case int:       t = v != 0
    case bool:      t = v
    }
    return
}
func (p *any) float(ctx Context) (res float64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.float(ctx)
    case float32:     res = float64(v)
    case float64:     res = v
    case int:         res = float64(v)
    case int64:       res = float64(v)
    case bool: if v { res = FloatEpsilon }
    }
    return
}
func (p *any) int(ctx Context) (res int64, err error) {
    switch v := p.value.(type) {
    case Value:       res, err = v.int(ctx)
    case float32:     res = int64(v)
    case float64:     res = int64(v)
    case int:         res = int64(v)
    case int64:       res = v
    case bool: if v { res = 1 }
    }
    return
}
func (p *any) string(ctx Context) (s string) {
    switch v := p.value.(type) {
    case Value:       s = v.string(ctx)
    case float32:     s = strconv.FormatFloat(float64(v),'g', -1, 32)
    case float64:     s = strconv.FormatFloat(float64(v),'g', -1, 64)
    case int:         s = strconv.FormatInt(int64(v),10)
    case int64:       s = strconv.FormatInt(int64(v),10)
    case bool: if v { s = "true" } else { s = "false" }
    default: s = fmt.Sprintf("%s", p.value)
    }
    return
}
func (p *any) ident(ctx Context) (s string) {
    if v, y := p.value.(Value); y { s = v.ident(ctx) }
    return
}
func (p *any) String() string { return fmt.Sprintf("<%v>", p.value) }
func (p *any) traverse(ctx Context) { if v, ok := p.value.(Value); ok { v.traverse(ctx) } }
func (p *any) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if v, y := p.value.(Value); y { return v.cache(ctx, cache, bits) }
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *any) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type negative struct { Value }
func (p negative) String() (s string) { return p.srclit(nil) }
func (p negative) srclit(o Object) string { return `!`+srclit(o, p.Value) }
func (p negative) expand(ctx Context) Value { return negative{p.Value.expand(ctx)} }
func (p negative) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if x, y := v.(negative); y {
        res = p.Value.cmp(ctx, x.Value)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p negative) string(ctx Context) string {
    if true {
        return "!"+p.Value.string(ctx)
    } else if p.true(ctx) {
        return "true"
    } else {
        return "false"
    }
}
func (p negative) true(ctx Context) (res bool) {
    if p.Value != nil { res = !p.Value.true(ctx) }
    return
}
func (p negative) float(ctx Context) (res float64, _ error) {
    if !p.Value.true(ctx) { res = FloatEpsilon }
    return
}
func (p negative) int(ctx Context) (res int64, _ error) {
    if !p.Value.true(ctx) { res = 1 }
    return
}
func (p negative) traverse(ctx Context) { if p.Value != nil { p.Value.traverse(ctx) } }
func (_ negative) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ negative) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

func stringMatch(ctx Context, p Value, i interface{}) (full bool, s string, stems []string) {
    var v = p.string(ctx)
    switch t := i.(type) {
    case Value:
        if w := t.string(ctx); strings.HasPrefix(w, v) { full, s = (len(v) == len(w)), v }
    case string:
        if strings.HasPrefix(t, v) { full, s = (len(v) == len(t)), v }
    case []string:
        if n := len(t); n > 0 { if t[0] == v { full, s = (n == 1), v } }
    default:
        errostack(at(ctx,p), 3, "%v: matching unsupported value: %v", us(p), us(i)).debug(16)
    }
    return
}

type escaped struct { valbase; s string }
func (_ *escaped) kind() Kind { return KindEscaped }
func (p *escaped) String() string { return "\\" + p.s }
func (p *escaped) string(Context) (s string) {
    switch p.s {
    case "n": s = "\n"
    case "r": s = "\r"
    default : s = p.s
    }
    return
}
func (p *escaped) true(Context) bool { return p.s != "" }
func (p *escaped) float(Context) (_ float64, _ error) { return }
func (p *escaped) int(Context) (_ int64, _ error) { return }
func (p *escaped) expand(Context) Value { return p }
func (p *escaped) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if o, y := v.(*escaped); y {
        if p.s == o.s { res = cmpEqual }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *escaped) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    return stringMatch(ctx, p, i)
}
func (p *escaped) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
    return
}
func (_ *escaped) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *escaped) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}


type boolean struct { valbase; bool }
func (_ *boolean) kind() Kind { return KindBoolean }
func (p *boolean) String() string { return "{="+p.string_()+"}" }
func (p *boolean) string(Context) string { return p.string_() }
func (p *boolean) string_() string { if p.bool { return "true" } else { return "false" } }
func (p *boolean) true(Context) bool { return p.bool }
func (p *boolean) float(Context) (v float64, _ error) { if p.bool { v = 1. }; return }
func (p *boolean) int(Context) (v int64, _ error) { if p.bool { v = 1  }; return }
func (p *boolean) expand(Context) Value { return p }
func (p *boolean) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
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
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *boolean) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    return stringMatch(ctx, p, i)
}
func (p *boolean) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
    return
}
func (_ *boolean) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *boolean) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type prediction struct { boolean ; s string }

type answer struct { boolean }
func (p *answer) String() (s string) { return "{="+p.string_()+"}" }
func (p *answer) string(Context) string { return p.string_() }
func (p *answer) string_() (s string) { if p.bool { return "yes" } else { return "no" } }
func (p *answer) expand(Context) Value { return p }

type option struct { boolean }
func (p *option) String() (s string) { return "{="+p.string_()+"}" }
func (p *option) string(ctx Context) string { return p.string_() }
func (p *option) string_() (s string) { if p.bool { return "on" } else { return "off" } }
func (p *option) expand(Context) Value { return p }

func _optionalize(val Value) (name Value, okay bool) {
    if g, y := val.(*globpat); y && len(g.elems) == 2 {
        if m, y := g.elems[1].(*globmeta); y && m.token == QUE {
            name, okay = g.elems[0], true
        }
    }
    return
}
func optionalize(ctx Context, val Value) (res condval, okay bool) {
    if v, y := _optionalize(val); y {
        res, okay = condval{v}, true
    } else if t, y := val.(*barecomp); y {
        if v, y := _optionalize(t.elems[len(t.elems)-1]); y {
            x := makeBarecomp(t.elems[:len(t.elems)-1]...)
            x.elems = append(x.elems, v)
            res, okay = condval{x}, true
        }
    }
    return
}

func boolVal(v Value) (res, y bool) {
    switch t := v.(type) {
    case     *answer: res, y = t.bool, true
    case    *boolean: res, y = t.bool, true
    case     *option: res, y = t.bool, true
    case *prediction: res, y = t.bool, true
    }
    return
}

func makePrediction(pos Position, val bool, s string) *prediction {
    return &prediction{boolean{valbase{pos}, val}, s}
}

type integer struct { valbase; int64 }
func (_ *integer) kind() Kind { return KindInteger }
func (p *integer) true(ctx Context) bool { return p.int64 != 0 }
func (p *integer) int(ctx Context) (i int64, _ error) { return p.int64, nil }
func (p *integer) float(ctx Context) (f float64, _ error) { return float64(p.int64), nil }
func (p *integer) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    var o *integer
    switch t := v.(type) {
    case      *binary: o = &t.integer
    case     *decimal: o = &t.integer
    case *hexadecimal: o = &t.integer
    case       *octal: o = &t.integer
    }
    if o != nil {
        if p == o || p.int64 == o.int64 {
            res = cmpEqual
        } else if p.int64 < o.int64 {
            res = cmpSmaller
        } else if p.int64 > o.int64 {
            res = cmpGreater
        }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (_ *integer) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *integer) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type binary struct { integer }
func (p *binary) kind() Kind { return p.integer.kind()|KindBinary }
func (p *binary) String() string { return fmt.Sprintf("0b%s", strconv.FormatInt(int64(p.int64),2)) }
func (p *binary) string(ctx Context) string { return strconv.FormatInt(int64(p.int64),2) }
func (p *binary) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *binary) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *binary) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *binary) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *binary) expand(Context) Value { return p }

type octal struct { integer }
func (p *octal) kind() Kind { return p.integer.kind()|KindOctal }
func (p *octal) String() string { return fmt.Sprintf("0%s", strconv.FormatInt(int64(p.int64),8)) }
func (p *octal) string(ctx Context) string { return strconv.FormatInt(int64(p.int64),8) }
func (p *octal) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *octal) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *octal) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *octal) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *octal) expand(Context) Value { return p }

type decimal struct { integer }
func (p *decimal) kind() Kind { return p.integer.kind()|KindDecimal }
func (p *decimal) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *decimal) string(ctx Context) string { return strconv.FormatInt(int64(p.int64),10) }
func (p *decimal) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *decimal) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *decimal) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *decimal) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *decimal) expand(Context) Value { return p }

type hexadecimal struct { integer }
func (p *hexadecimal) kind() Kind { return p.integer.kind()|KindHexadecimal }
func (p *hexadecimal) String() string { return fmt.Sprintf("0x%s", strconv.FormatInt(int64(p.int64),16)) }
func (p *hexadecimal) string(ctx Context) string { return strconv.FormatInt(int64(p.int64),16) }
func (p *hexadecimal) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *hexadecimal) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *hexadecimal) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *hexadecimal) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *hexadecimal) expand(Context) Value { return p }

type float struct {} // TODO

const FloatEpsilon = 1e-15 /* 1e-16 */
type Float struct { valbase; float64 } // IEEE-754 64-bit binary floating-point
func (p *Float) kind() Kind { return KindFloat }
func (p *Float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) string(ctx Context) string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *Float) true(ctx Context) bool { return math.Abs(p.float64)-0 > FloatEpsilon }
func (p *Float) int(ctx Context) (i int64, _ error) { return int64(p.float64), nil }
func (p *Float) float(ctx Context) (f float64, _ error) { return p.float64, nil }
func (p *Float) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *Float) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Float) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *Float) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *Float) expand(Context) Value { return p }
func (p *Float) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if _, ok := v.(*Float); ok {
        if f, e := v.float(ctx); e != nil {
            if false { warn(ctx, "%v: %v", v, e).debug(1) }
        } else if p.float64 == f {
            res = cmpEqual
        } else if p.float64 < f {
            res = cmpSmaller
        } else if p.float64 > f {
            res = cmpGreater
        }
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (_ *Float) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *Float) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type datetime struct { valbase ; t time.Time }
func (_ *datetime) kind() Kind { return KindDateTime }
func (p *datetime) String() string { return time.Time(p.t).Format("2006-01-02T15:04:05.999999999Z07:00") }
func (p *datetime) string(ctx Context) string { return p.String() } // time.RFC3339Nano
func (p *datetime) true(ctx Context) bool { return !p.t.IsZero() }
func (p *datetime) int(ctx Context) (i int64, _ error) { return p.t.Unix(), nil }
func (p *datetime) float(ctx Context) (f float64, _ error) { return float64(p.t.Unix()), nil }
func (p *datetime) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *datetime) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *datetime) expand(Context) Value { return p }
func (p *datetime) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    var vt time.Time
    switch a := v.(type) {
    case *datetime: vt = a.t
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
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (_ *datetime) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *datetime) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

func ParseDateTime(pos Position, s string) *datetime {
    // time.RFC3339Nano
    if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
        return &datetime{valbase{pos},t}
    } else {
        panic(e)
    }
}

type Date struct { datetime }
func (p *Date) kind() Kind { return p.datetime.kind()|KindDate }
func (p *Date) String() string { return time.Time(p.t).Format("2006-01-02") }
func (p *Date) string(ctx Context) string { return p.String() }
func (p *Date) int(ctx Context) (i int64, _ error) { return p.t.Unix(), nil }
func (p *Date) float(ctx Context) (f float64, _ error) { return float64(p.t.Unix()), nil }
func (p *Date) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *Date) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Date) expand(Context) Value { return p }

type Time struct { datetime }
func (p *Time) kind() Kind { return p.datetime.kind()|KindTime }
func (p *Time) String() string { return time.Time(p.t).Format("15:04:05.999999999Z07:00") }
func (p *Time) string(ctx Context) string { return p.String() }
func (p *Time) int(ctx Context) (i int64, _ error) { return p.t.Unix(), nil }
func (p *Time) float(ctx Context) (f float64, _ error) { return float64(p.t.Unix()), nil }
func (p *Time) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) { return stringMatch(ctx, p, i) }
func (p *Time) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Time) expand(Context) Value { return p }

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//                └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//                     └<userinfo>─(@)┘      └(:)─<port>┘
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
func (_ *URL) kind() Kind { return KindUrl }
func (p *URL) srclit(o Object) (s string) {
    if s = srclit(o, p.Scheme); s == "" { return }
    if s += ":"; p.Host == nil {
        // ...
    } else if _, ok := p.Host.(*none); ok {
        var host string
        if host = srclit(o, p.Host); host == "" { return }
        s += "//"
        if p.Username == nil {
            // ...
        } else if isNone(p.Username) {
            var user string
            if user = srclit(o, p.Username); user != "" {
                s += user + "@"
            }
        }
        s += host
        if p.Port == nil {
            // ...
        } else if _, ok := p.Port.(*none); ok {
            var port string
            if port = srclit(o, p.Port); port != "" {
                s += ":" + port
            }
        }
    }
    if p.Path == nil {
        // ...
    } else if _, ok := p.Path.(*none); ok {
        var path string
        if path = srclit(o, p.Path); path != "" {
            //if !strings.HasPrefix(path, pathSep) { s += pathSep }
            s += path
        }
    }
    if p.Query == nil {
        // ...
    } else if _, ok := p.Query.(*none); ok {
        var query string
        if query = srclit(o, p.Query); query != "" {
            s += "?" + query
        }
    }
    if p.Fragment == nil {
        // ...
    } else if _, ok := p.Fragment.(*none); ok {
        var fragment string
        if fragment = srclit(o, p.Fragment); fragment != "" {
            s += "#" + fragment
        }
    }
    return
}
func (p *URL) String() string { return p.srclit(nil) }
func (p *URL) string(ctx Context) (s string) {
    if s = p.Scheme.string(ctx) + ":"; p.Host != nil && !isNone(p.Host) {
        s += "//"
        if p.Username != nil && !isNone(p.Username) {
            s += p.Username.string(ctx)
            if p.Password != nil {
                s += ":" + p.Password.string(ctx)
            }
            s += "@"
        }
        s += p.Host.string(ctx)
        if p.Port != nil && !isNone(p.Port) {
            s += ":" + p.Port.string(ctx)
        }
    }
    if p.Path != nil && !isNone(p.Path) {
        //if !strings.HasPrefix(path, pathSep) { s += pathSep }
        s += p.Path.string(ctx)
    }
    if p.Query != nil && !isNone(p.Query) {
        s += "?" + p.Query.string(ctx)
    }
    if p.Fragment != nil && !isNone(p.Fragment) {
        s += "#" + p.Fragment.string(ctx)
    }
    return
}
func (p *URL) true(ctx Context) (t bool) {
    if p.Scheme != nil { if t = p.Scheme.true(ctx); t { return }}
    if p.Host   != nil { if t = p.Host  .true(ctx); t { return }}
    if p.Path   != nil { if t = p.Path  .true(ctx); t { return }}
    return //p.String() != "", nil
}
func (p *URL) int(ctx Context) (i int64, _ error) {
    if s := p.string(ctx); s != "" { i = int64(len(s)) }
    return
}
func (p *URL) float(ctx Context) (f float64, e error) { i, e := p.int(ctx); return float64(i), e }
func (p *URL) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
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
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *URL) expand(ctx Context) (res Value) {
    var o = &URL{ valbase: p.valbase }
    if nil != p.Scheme   { o.Scheme   = p.Scheme.expand(ctx) }
    if nil != p.Username { o.Username = p.Username.expand(ctx) }
    if nil != p.Password { o.Password = p.Password.expand(ctx) }
    if nil != p.Host     { o.Host     = p.Host.expand(ctx) }
    if nil != p.Port     { o.Port     = p.Port.expand(ctx) }
    if nil != p.Path     { o.Path     = p.Path.expand(ctx) }
    if nil != p.Query    { o.Query    = p.Query.expand(ctx) }
    if nil != p.Fragment { o.Fragment = p.Fragment.expand(ctx) }
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
func (p *URL) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *URL) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *URL) Validate() (res *url.URL) {
    panic(fmt.Sprintf("validate %s", p))
    return
}
func (_ *URL) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *URL) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type raw struct { valbase; s string }
func (_ *raw) kind() Kind { return KindRaw }
func (p *raw) String() string { return p.s }
func (p *raw) string(ctx Context) string { return p.s }
func (p *raw) true(ctx Context) bool { return p.s != "" }
func (p *raw) int(ctx Context) (i int64, err error) { return strconv.ParseInt(p.s, 10, 64) }
func (p *raw) float(ctx Context) (f float64, err error) { return strconv.ParseFloat(p.s, 64) }
func (p *raw) expand(Context) Value { return p }
func (p *raw) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, ok := v.(*raw); ok {
        if p.s == a.s { res = cmpEqual }
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *raw) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *raw) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (_ *raw) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *raw) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type strlit struct { valbase; s string }
func (_ *strlit) kind() Kind { return KindStrLit }
func (p *strlit) String() string { return `'`+p.s+`'` }
func (p *strlit) string(ctx Context) string { return p.s }
func (p *strlit) true(ctx Context) bool { return p.s != "" }
func (p *strlit) int(ctx Context) (i int64, err error) { return strconv.ParseInt(p.s,10,64) }
func (p *strlit) float(ctx Context) (f float64, err error) { return strconv.ParseFloat(p.s, 64) }
func (p *strlit) expand(ctx Context) Value {
    if _exPathStr(ctx) {
        return _pathstr(ctx, p.s)
    } else {
        return p
    }
}
func (p *strlit) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    switch t := v.(type) {
    case *strlit:
        if p.s == t.s {
            res = cmpEqual
        } else if p.s < t.s {
            res = cmpSmaller
        } else /*if p.s > t.s*/ {
            res = cmpGreater
        }
    case *list:
        if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *strlit) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *strlit) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *strlit) traverse(ctx Context) { ctx.traverse(at(ctx, p.position), p) }
func (p *strlit) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if false { ctx = at(ctx, p.position) }
    cache = cache.str(ctx, "''", bits)
    return cache.strx(ctx, p.s, bits)
}
func (p *strlit) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if cache.fast != nil { if c := cache.str(ctx, "''", cacheZero); c != nil {
        if c = c.strx(ctx, p.s, cacheZero); c != nil { res = append(res, c) }
    }}
    return
}

type strval struct { valbase; v []Value }
func (_ *strval) kind() Kind { return KindStrVal }
func (p *strval) String() (s string) {
    if expandable(original{nil,defExpand2}, p.v...) {
        for _, v := range p.v {
            if s != "" { s += " " }
            s += v.String()
        }
        s = `$(string `+s+`)`
    } else {
        for _, v := range p.v {
            if s != "" { s += " " } // TODO: seperator?
            s += v.String()
        }
        s = `'`+s+`'`
    }
    return
}
func (p *strval) string(ctx Context) (s string) {
    for _, v := range p.v {
        if t := v.string(ctx); t != "" {
            if s != "" { s += " " } // TODO: seperator?
            s += t
        }
    }
    return
}
func (p *strval) expand(ctx Context) Value {
    if _exPathStr(ctx) {
        return _pathstr(ctx, p.string(ctx))
    } else if v := expand(ctx, p.v...); diff(ctx, v, p.v) {
        return &strval{p.valbase,v}
    } else {
        return p
    }
}
func (p *strval) es(ctx Context, f func(string)) {
    var v = p.expand(ctx)
    if isFinalValue(ctx, v) { f(v.string(ctx)) }
}
func (p *strval) true(ctx Context) (res bool) {
    p.es(ctx, func(s string) { res = s != "" })
    return
}
func (p *strval) int(ctx Context) (i int64, err error) {
    p.es(ctx, func(s string) { i, err = strconv.ParseInt(s, 10, 64) })
    return
}
func (p *strval) float(ctx Context) (f float64, err error) {
    p.es(ctx, func(s string) { f, err = strconv.ParseFloat(s, 64) })
    return
}
func (p *strval) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    switch t := v.(type) {
    case *group:
    case *list: if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    case *strval:
        var n = len(t.v)
        for i, v := range p.v {
            if n <= i { break }
            if res = v.cmp(ctx, t.v[i]); res != cmpEqual{
                break
            }
        }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *strval) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *strval) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *strval) traverse(ctx Context) { ctx.traverse(at(ctx, p.position), p) }
func (p *strval) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    cache = cache.str(ctx, "''", bits)
    return cache.strx(ctx, p.string(ctx), bits)
}
func (p *strval) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if cache.fast != nil { if c := cache.str(ctx, "''", cacheZero); c != nil {
        if c = c.strx(ctx, p.string(ctx), cacheZero); c != nil { res = append(res, c) }
    }}
    return
}

func isTrueString(s string) (t bool) {
    switch strings.ToLower(s) {
    case "false", "no" , "off", "force_off", "0", "": t = false
    case "true" , "yes", "on" , "force_on" , "1"    : t = true
    default: t = true
    }
    return
}

type punctuation struct { valbase; tok token }
func (p *punctuation) String() string { return p.tok.String() }
func (p *punctuation) string(ctx Context) string { return p.tok.String() }
func (p *punctuation) true(ctx Context) bool { return false }
func (p *punctuation) int(ctx Context) (i int64, _ error) { return 0, nil }
func (p *punctuation) float(ctx Context) (f float64, _ error) { return 0, nil }
func (p *punctuation) expand(Context) Value { return p }
func (p *punctuation) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, y := v.(*punctuation); y {
        if p.tok == a.tok {
            res = cmpEqual
        } else if p.tok > a.tok {
            res = cmpSmaller
        } else if p.tok < a.tok {
            res = cmpGreater
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *punctuation) match(ctx Context, i interface{}) (full bool, res interface{}, stems []string) {
    var s string
    switch t := i.(type) {
    case   Value : s = t.string(ctx)
    case   string: s = t
    case []string: if len(t) == 1 { s = t[0] } else { return }
    default:
        erro(at(ctx,p), "%T: matching unsupported value: %T %v", p, i, i).debug(1)
        return
    }
    if t := p.string(ctx); strings.HasPrefix(s, t) {
        full, res = len(s) == len(t), s[:len(t)]
        if false && s == ".h" { warn(ctx, "%v %v; %v %v", p, s, full, res).debug(6) }
    }
    return
}
func (p *punctuation) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *punctuation) traverse(ctx Context) { }
func (p *punctuation) filecache(ctx Context, c *filecache) (res *filecache, done bool) {
    if res, done = c.hit(ctx, p.tok); res == nil {
        if cacheMapping(ctx) {
            erro(ctx, "no filecache for %v : %v", us(p), c).debug(16)
        }
    }
    return
}
func (_ *punctuation) cache(ctx Context, cache *valcache, bits int) (res *valcache) { return cache }
func (_ *punctuation) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type bare string
type bareword struct { valbase; s string }
func (_ *bareword) kind() Kind { return KindBareword }
func (p *bareword) String() string { return p.s }
func (p *bareword) string(ctx Context) string { return p.s }
func (p *bareword) true(ctx Context) bool { return p.s != "" }
func (p *bareword) int(ctx Context) (i int64, err error) { return strconv.ParseInt(p.s, 10, 64) }
func (p *bareword) float(ctx Context) (f float64, err error) { return strconv.ParseFloat(p.s, 64) }
func (p *bareword) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, y := v.(*bareword); y {
        if p.s == a.s {
            res = cmpEqual
        } else if p.s < a.s {
            if strings.HasPrefix(a.s, p.s) {
                res = cmpLprefix
            } else {
                res = cmpSmaller
            }
        } else {
            if strings.HasPrefix(p.s, a.s) {
                res = cmpRprefix
            } else {
                res = cmpGreater
            }
        }
    } else if c, y := v.(*barecomp); y {
        if s := c.string(ctx); p.s == s {
            res = cmpEqual
        } else if p.s < s {
            if strings.HasPrefix(s, p.s) {
                res = cmpLprefix
            } else {
                res = cmpSmaller
            }
        } else {
            if strings.HasPrefix(p.s, s) {
                res = cmpRprefix
            } else {
                res = cmpGreater
            }
        }
    } else if l, y := v.(*path); y {
        if len(l.elems) == 1 { return p.cmp(ctx, l.elems[0]) }
    } else if l, y := v.(*list); y {
        if len(l.elems) == 1 { return p.cmp(ctx, l.elems[0]) }
    } else if false {
        // NOTE: find the only valid element (if others are none)
        for _, elem := range l.elems {
            if isNone(elem) { continue }
            if v == l { v = elem }
        }
        if v != l { return p.cmp(ctx, v) }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *bareword) filecache(ctx Context, fc *filecache) (res *filecache, done bool) {
    if res, done = fc.hit(ctx, p.s); res == nil {
        if cacheMapping(ctx) {
            erro(at(ctx,p), "no filecache for %v : %v", us(p), fc).debug(16)
        }
    }
    return
}
func (p *bareword) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    return cache.str(at(ctx, p.position), p.s, bits)
}
func (p *bareword) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if cache.fast != nil { if c := cache.str(ctx, p.s, bits); c != nil {
        res = append(res, c) ; return
    }}
    if bits&cacheMatchPatts != 0 { if c := cache.matchPatts(ctx, p.s); c != nil {
        res = append(res, c) ; return
    }}
    return
}
func (p *bareword) prefix(ctx Context, v Value) Value { return _suffix(ctx, v, p) }
func (p *bareword) suffix(ctx Context, v Value) Value { return _bifix(ctx, p, v) }
func (p *bareword) expand(ctx Context) (res Value) {
    if res = p; false /* && w&expandFullFile != 0 */ {
        if f := file(ctx, p.string(ctx)); f != nil {
            res = f.expand(ctx)
        }
    }
    return
}
func (p *bareword) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    return stringMatch(ctx, p, i)
}
func (p *bareword) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *bareword) traverse(ctx Context) { ctx.traverse(at(ctx, p.position), p) }

type qualiword struct { valbase; words []string } // TODO: foo.bar.zar, foo.&(bar).zar ???
func (p *qualiword) String() string { return strings.Join(p.words,".") }
func (p *qualiword) string(ctx Context) string { return p.String() }
func (p *qualiword) true(ctx Context) bool { return len(p.words)!=0 }
func (p *qualiword) int(ctx Context) (i int64, _ error) { return int64(len(p.words)), nil }
func (p *qualiword) float(ctx Context) (f float64, _ error) { return 0, nil }
func (p *qualiword) expand(Context) Value { return p }
func (p *qualiword) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, y := v.(*qualiword); y {
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
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *qualiword) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *qualiword) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *qualiword) traverse(ctx Context) { ctx.traverse(at(ctx, p.position), p) }
func (p *qualiword) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *qualiword) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type elements struct { elems []Value }
func (p *elements) Position() Position { return p.elems[0].Position() }
func (p *elements) append(v ...Value)  { p.elems = append(p.elems, v...) }
func (p *elements) at(n int) (v Value) { if 0 <= n && n < len(p.elems) { v = p.elems[n] }; return }
func (p *elements) ident(Context) (_ string) { return } // aka name
func (p *elements) list() *list { return &list{*p} }
func (p *elements) path() *path { return &path{*p} }
func (p *elements) barecomp() *barecomp { return &barecomp{*p} }
func (p *elements) compound() *compound { return &compound{*p} }
func (p *elements) len() int { return len(p.elems) }
func (p *elements) slice(n int, m ...int) (a []Value) {
    if x := p.len(); n < 0 {
        if (-n) < x { a = p.elems[:x-n] } // TODO: apply 'm' for ???
    } else if n < x {
        a = p.elems[n:] // TODO: apply 'm' for p.slice(1, -1)
    }
    return
}
func (p *elements) take(n int) (v Value) {
    if x := len(p.elems); n>=0 && n<x {
        v = p.elems[n]
        p.elems = append(p.elems[0:n], p.elems[n+1:]...)
    }
    return
}
func (p *elements) true(ctx Context) (t bool) { // (or elems...)
    for _, elem := range p.elems {
        if elem != nil {
            if t = elem.true(ctx); t { break }
        }
    }
    return
}
func (p *elements) refs(ctx Context, v Value) bool {
    for _, elem := range p.elems {
        if elem != nil && (elem == v || elem.refs(ctx, v)) {
            return true
        }
    }
    return false
}
func (p *elements) defs(ctx Context, s ...string) (res []*def) {
    for _, elem := range p.elems { res = append(res, elem.defs(ctx, s...)...) }
    return
}
func (p *elements) expandable(ctx Context) (res bool) {
    for i, elem := range p.elems {
        if elem == nil {
            erro(ctx, "nil element #%d: %v", i, p).debug(32)
        } else if res = elem.expandable(ctx); res { break }
    }
    return
}
func (_ *elements) delete(Context) (_ []*File, _ error) { return }
func (_ *elements) patterned(Context) (_ bool) { return }
func (_ *elements) stamp(Context) (_ []*File, _ error) { return }
func (_ *elements) stat(Context) (_ *statinfo) { return }
func (_ *elements) traverse(Context) { }
func (_ *elements) updated(Context) (_ bool) { return }
func (_ *elements) updatedDeps(Context, ...Value) (_ []Value) { return }

func compareElems(ctx Context, elemsL, elemsR []Value) (res cmpres) {
    if len(elemsL) != len(elemsR) {
        elemsL = merge(elemsL...)
        elemsR = merge(elemsR...)
    }
    if len(elemsL) == len(elemsR) {
        for i,  elemL := range elemsL {
            var elemR = elemsR[i]
            var l,  r = elemL == nil, elemR == nil
            if l && r { continue }
            if l || r { return }
            if res = elemL.cmp(ctx, elemR); res != cmpEqual { return }
        }
        if res == 0 { res = cmpEqual }
    } else {
        var notEmptyL, notEmptyR bool
        for _, elem := range elemsL {
            if notEmptyL = !(isNone(elem) || isNull(elem)); notEmptyL { break }
        }
        for _, elem := range elemsR {
            if notEmptyR = !(isNone(elem) || isNull(elem)); notEmptyR { break }
        }
        if !notEmptyL && !notEmptyR { return cmpEqual }

        // TODO: list.cmp: cmpSmaller, cmpGreater
    }
    return
}

func cond(v Value) (y bool) {
    switch t := v.(type) {
    case     condval: return true
    case disjunction: return cond(t.Value)
    case        flag: //return cond(t.Value)
    case *barecomp:
        for _, e := range t.elems {
            if cond(e) { return true }
        }
    }
    return
}

func condish(ctx Context, v Value) Value {
    if x, y := v.(condval); _exCondless(ctx) {
        if y {
            if checkpoints {
                if cond(x.Value) {
                    noted(at(ctx,v), "nested condval: %v : %v", v, us(v))
                    erro(ctx, "%v", us(ctx)).debug(10)
                }
            }
            return x.Value // TODO: condless(x.Value)
        } else {
            if checkpoints {
                if cond(v) {
                    noted(at(ctx,v), "nested condval: %v : %v", v, us(v))
                    erro(ctx, "%v", us(ctx)).debug(10)
                }
            }
            return v
        }
    } else if y {
        if checkpoints {
            if cond(x.Value) {
                noted(at(ctx,v), "nested condval: %v : %v", v, us(v))
                erro(ctx, "%v", us(ctx)).debug(10)
            }
        }
        return x
    } else {
        if checkpoints {
            if cond(v) {
                noted(at(ctx,v), "nesting condval: %v : %v", v, us(v))
                erro(ctx, "%v", us(ctx)).debug(10)
            }
        }
        return condval{v} // condish
    }
}

type condval struct { Value } // conditional component: barecomp, pair; aka optional
func (p condval) kind() Kind { return p.Value.kind()|KindCondval }
func (p condval) String() string { return p.Value.String()+"?" }
func (p condval) string(ctx Context) (s string) {
    p.exstr(ctx, func(v Value) { s = v.string(ctx) })
    return
}
func (p condval) true(ctx Context) (t bool) {
    p.exstr(ctx, func(v Value) { t = v.true(ctx) })
    return
}
func (p condval) int(ctx Context) (i int64, e error) {
    p.exstr(ctx, func(v Value) { i, e = v.int(ctx) })
    return
}
func (p condval) float(ctx Context) (f float64, e error) {
    p.exstr(ctx, func(v Value) { f, e = v.float(ctx) })
    return
}
func (p condval) exstr(ctx Context, f func(Value)) {
    if t := p.expand(final{ctx}); t != nil && !equal(ctx, p, t) { f(t) }
}
func (p condval) expand(ctx Context) (res Value) {
    if checkpoints { defer trace(ctx) }

    var v = p.Value.expand(condless{ctx})
    if checkpoints { defer func() { p.expand_check(ctx, v, res) } () }

    if v == nil { // only _exFinal(ctx)
        return makeNull(p.Position())
    } else if x, y := v.(disjunction); y {
        var vals []Value
        for _, v := range merge(x.Value) {
            if indeterminate(ctx, v) { v = condish(ctx, disjunction{v}) }
            vals = append(vals, v)
        }
        return ease(ctx, vals)
    } else if x, y := v.(*list); y {
        var vals []Value
        for _, v := range merge(x.elems...) {
            if indeterminate(ctx, v) { v = condish(ctx, v) }
            vals = append(vals, v)
        }
        return ease(ctx, vals)
    } else if indeterminate(ctx, v) {
        return condish(ctx, v)
    } else {
        return v
    }
}
func (p condval) expand_check(ctx Context, v, res Value) {
    if cond(v) {
        noted(at(ctx,p), "%v → %v → %v", p.Value, v, res)
        noted(at(ctx,p), "%v\t: %v", p.Value, us(p.Value))
        noted(at(ctx,p), "%v\t: %v", v,       us(v))
        noted(at(ctx,p), "%v\t: %v", res,     us(res))
        erro(ctx, "%v", us(ctx)).debug(10)
    }
    if _exFinal(ctx) {
        // ...
    } else {
        if v == nil {
            noted(at(ctx,p), "%v → %v", p.Value, res)
            noted(at(ctx,p), "%v : %v", p.Value, us(p.Value))
            erro(ctx, "%v", us(ctx)).debug(10)
            return
        }
        if !equal(ctx, v, p.Value) && p.Value.String() == v.String() {
            noted(at(ctx,p), "%v → %v → %v", p.Value, v, res)
            noted(at(ctx,p), "%v\t: %v", p.Value, us(p.Value))
            noted(at(ctx,p), "%v\t: %v", v,       us(v))
            noted(at(ctx,p), "%v\t: %v", res,     us(res))
            erro(ctx, "%v", us(ctx)).debug(5)
        }
    }
}
func (p condval) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }
    if x, y := v.(condval); y {
        return p.cmp(ctx, x.Value)
    } else if x, y := v.(*list); y && x.len() == 1 {
        return p.cmp(ctx, x.elems[0])
    } else {
        return p.Value.cmp(ctx, v)
    }
}
func (p condval) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
    }
}

type conjunction struct { *list ; sep Value }
func (p conjunction) String() string {
    var s string
    if p.sep != nil { s = p.sep.String() }
    return "{{"+p.list.String()+"}"+s+"}"
}
func (p conjunction) string(ctx Context) (s string) {
    var sep string
    if p.sep != nil { sep = p.sep.string(ctx) }

    var ss []string
    var elems = expand(final{ctx}, p.elems...)
    for _,  elem := range elems {
        if isFinalValue(ctx, elem) {
            ss = append(ss, elem.string(ctx))
        }
    }
    return strings.Join(ss, sep)
}
func (p conjunction) expand(ctx Context) (res Value) {
    var s Value
    var l = p.list.expand(ctx).(*list)
    if p.sep != nil { s = p.sep.expand(ctx) }
    if !equal(ctx, l, p.list) {
        return conjunction{l, s}
    } else if s != nil && !equal(ctx, s, p.sep) {
        return conjunction{l, s}
    }
    return p
}

type disjunction struct { Value }
func (p disjunction) String() string { return "{"+p.Value.String()+"}" }
func (p disjunction) expand(ctx Context) (res Value) {
    const DIS = false // var DIS = _exDisjunction(ctx)

    var v = p.Value.expand(ctx)

    if x, y := v.(*list); DIS && y {
        var elems []Value
        for _, v := range x.elems {
            if indeterminate(ctx, v) { v = disjunction{v} }
            elems = append(elems, v)
        }
        return ease(ctx, elems)
    } else if !DIS && y {
    xlist:
        if n := x.len(); 0 == n {
            return x // FIXME: {=}, aka null?
        } else if 1 == n {
            var t = x.elems[0]
            if l, y := t.(*list); y { x = l; goto xlist }
            return t
        } else {
            return disjunction{x}
        }
    } else if indeterminate(ctx, v) {
        return disjunction{v} // doesn't matter equal(ctx, v, p.Value)
    } else {
        return v
    }
}
func (p disjunction) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if x, y := v.(disjunction); y {
        if p.Value == x.Value {
            if checkpoints {
                if cr := p.Value.cmp(ctx, x.Value); cr != cmpEqual {
                    erro(ctx, "%v, %v ⇔ %v", cr, us(p.Value), us(x.Value)).debug(3)
                }
            }
            res = cmpEqual
        } else {
            res = p.Value.cmp(ctx, x.Value)
        }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}

type barecomp struct { elements }
func (_ *barecomp) kind() Kind { return KindBarecomp }
func (p *barecomp) String() string { return p.srclit(nil) }
func (p *barecomp) srclit(o Object) (s string) {
    for _, elem := range p.elems { s += srclit(o, elem) }
    return
}
func (p *barecomp) string(ctx Context) (s string) {
    if v := p.expand(final{ctx}); cond(v) && indeterminate(ctx, v) {
        return
    } else if equal(ctx, v, p) {
        for _, elem := range p.elems { s += elem.string(ctx) }
        return
    } else {
        return v.string(ctx)
    }
}
func (p *barecomp) true(ctx Context) bool { return p.elements.true(ctx) }
func (p *barecomp) float(ctx Context) (_ float64, _ error) { return }
func (p *barecomp) int(ctx Context) (res int64, _ error) {
    if n := len(p.elems); n > 0 {
        if i, y := p.elems[0].(*decimal); y {
            switch n {
            case 1: res = i.int64
            case 2:
                if w, y := p.elems[1].(*bareword); y {
                    if  (w.s == "st" && i.int64%1 == 0) ||
                        (w.s == "nd" && i.int64%2 == 0) ||
                        (w.s == "rd" && i.int64%3 == 0) ||
                        (w.s == "th") { res = i.int64 }
                }
            }
        }
    }
    return
}
func (p *barecomp) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *barecomp) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *barecomp) expandable(ctx Context) bool { return p.elements.expandable(ctx) }
func (p *barecomp) expand_check(ctx Context, res Value) {
    if res == nil {
        noted(at(ctx,p), "%v, %v", p, res)
        noted(at(ctx,p), "%v", us(p))
        noted(at(ctx,p), "%v", us(res))
        erro(ctx, "%v", us(ctx)).debug(10)
    } else if p.expandable(ctx) && equal(ctx, p, res) {
        if s := p.String(); strings.Contains(s, "$_") {
            if r := res.String(); res == p || r == s || strings.Contains(r, "$_") {
                if d := autoDef(ctx, "_"); d != nil {
                    noted(at(ctx,d), "%v", us(d))
                    noted(at(ctx,p), "%v → %v", us(p), us(res))
                    erro(ctx, "%v", us(ctx)).debug(30)
                }
            }
        }
    }
}
func (p *barecomp) expand(ctx Context) (res Value) {
    if checkpoints { defer func() { p.expand_check(ctx, res) } () }

    type record struct { v []Value ; c int }

    var f func(_, _ []Value) []record

    f = func(a, elems []Value) (rs []record) {
        var c int
        for i, val := range elems {
            if val == nil {
                erro(at(ctx,p), "%v: nil component #%d", us(p), i).debug(1)
                continue
            }

            var v = val.expand(condless{ctx})
            if v == nil { continue }
            if x, y := v.(disjunction); y {
                switch x.Value.(type) {
                case *barecomp, *bareword, condval: // these are singleton-values
                    a = append(a, x.Value)
                    continue
                }

                if isFinalValue(ctx, x.Value) {
                    var tail = elems[i+1:]
                    for _, v := range merge(x.Value) {
                        rs = append(rs, f(append(a, v), tail)...)
                    }
                    return
                }
            }

            if cond(val) { c += 1 }

            a = append(a, v)
        }
        if 0 < len(a) {
            if false {
                rs = append(rs, record{a,c}) // BUG: yields constant records
            } else {
                rs = append(rs, record{copyvals(a), c})
            }
        }
        return
    }

    var rs = f(nil, p.elems)
    if n := len(rs); 1 < n || (1 == n && diff(ctx, rs[0].v, p.elems)) {
        var _res []Value
        for _, r := range rs {
            var v Value = &barecomp{elements{r.v}}
            if 0 < r.c { v = condish(ctx, v) }
            _res = append(_res, v)
        }
        return ease(ctx, _res)
    } else {
        return p
    }
}
func (p *barecomp) traverse(ctx Context) { ctx.traverse(at(ctx,p), p) }
func (p *barecomp) filecache(ctx Context, _c *filecache) (res *filecache, done bool) {
    var c = _c
    for _, elem := range p.elems {
        if c, done = c.hit(ctx, elem); c == nil {
            if cacheMapping(ctx) {
                erro(ctx, "no filecache for %v : %v : %v", p, us(elem), c).debug(3)
            }
            return
        } else {
            if res = c ; done { return }
        }
    }
    return
}
func (p *barecomp) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    return cache.str(at(ctx,p), p.string(ctx), bits)
}
func (p *barecomp) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    var s string = p.string(ctx)
    if cache.fast != nil { if c := cache.str(ctx, s, bits); c != nil {
        res = append(res, c) ; return
    }}
    if bits&cacheMatchPatts != 0 { if c := cache.matchPatts(ctx, s); c != nil {
        res = append(res, c) ; return
    }}
    return
}
func (p *barecomp) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }

    var cmp = func(elemsL, elemsR []Value) {
        if res = compareElems(ctx, elemsL, elemsR); res != cmpEqual {
            var i int
            for ; i < len(elemsL) && i < len(elemsR); i += 1 {
                var cr = elemsL[i].cmp(ctx, elemsR[i])
                if cr == cmpEqual { continue } else
                if cr == cmpLprefix || cr == cmpRprefix {
                    if false && p.string(ctx) == v.string(ctx) { res = cmpEqual ; return }
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

    if a, y := v.(*barecomp); y {
        if p == a {
            return cmpEqual
        } else {
            cmp(baremerge(p.elems...), baremerge(a.elems...))
            return
        }
    } else if w, y := v.(*bareword); y {
        if s := p.string(ctx); s == w.s {
            return cmpEqual
        } else if s < w.s {
            return cmpSmaller
        } else {
            return cmpGreater
        }
    } else if fR, y := v.(flag); y {
        var elems []Value
        for _, elem := range p.elems {
            if !isNull(elem) && !isNone(elem) {
                elems = append(elems, elem)
            }
        }
        if len(elems) == 2 { if fL, y := elems[0].(flag); y {
            if isNull(fL.Value) || isNone(fL.Value) {
                res = elems[1].cmp(ctx, fR.Value)
            } else if m, r, t := fL.Value.match(ctx, fR.Value); m {
                if isNull(elems[1]) || isNone(elems[1]) { res = cmpEqual }
            } else if r != nil { // matched prefix
                var s = joinPathStr(ctx, r)
                var sL = s + elems[1].string(ctx)
                var sR = fR.Value.string(ctx)
                if sL == sR {
                    return cmpEqual
                } else if s < sR {
                    return cmpSmaller
                } else {
                    return cmpGreater
                }
            } else if t != nil {
                unreachable(p, v)
            }
        }}
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    } else if l != nil && len(p.elems)==2 && len(l.elems)>1 {
        if pl, y := p.elems[1].(*list); y && len(pl.elems)>1 {
            var a = p.elems[1] // Example: p.elems=[a- a b c], l.elems=[z-a b c]
            // FIXME: avoid 'p.elems[1] = pl.elems[0]', the container values are readonly for cmp
            if p.elems[1] = pl.elems[0]; p.cmp(ctx, l.elems[0]) == cmpEqual {
                res = compareElems(ctx, pl.elems[1:], l.elems[1:])
            }
            p.elems[1] = a
        }
    }
    return
}
func (p *barecomp) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        noted(at(ctx,p), "%v, %v ; %v == %v", res, p==v, p, v)
        noted(at(ctx,p), "%v", us(p))
        noted(at(ctx,p), "%v", us(v))
        erro(ctx, "%v", us(ctx)).debug(5)
    }
}
func (p *barecomp) patterned(ctx Context) (res bool) {
    for _, elem := range p.elems {
        if res = elem.patterned(ctx); res { return }
    }
    return
}
func (p *barecomp) match(ctx Context, i interface{}) (full bool, res interface{}, stems []string) {
    var (
        n int
        s string
        elem Value
    )
    switch t := i.(type) {
    case   Value : s = t.string(ctx)
    case   string: s = t
    case []string: if len(t) == 1 { s = t[0] } else { return }
    default:
        erro(at(ctx,p), "%T: matching unsupported value: %T %v", p, i, i).debug(1)
        return
    }
    if s == "" { return }

    var rs string
    for n, elem = range p.elems {
        var _, r, ss = elem.match(ctx, s)
        var t = joinPathStr(ctx, r)
        if t == "" { break } else {
            stems = append(stems, ss...)
            s = s[len(t):]
            rs += t
        }
    }
    if s == "" && n == len(p.elems)-1 { full = true }
    if full || rs != "" { res = rs }
    return
}
func (p *barecomp) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var (
        elems []Value
        changed int
    )
    rest = stems
    for _, elem := range p.elems {
        var t Value
        if t, rest = elem.stencil(ctx, rest); t != elem {
            changed += 1
        }
        elems = append(elems, t)
    }
    if changed > 0 {
        val = makeBarecomp(elems...)
    } else {
        val = p
    }
    return
}
func (p *barecomp) prefix(ctx Context, val Value) Value {
    p = &barecomp{elements{p.elems}}
    if o, t := val.(*barecomp); t {
        p.elems = append(o.elems, p.elems...)
    } else {
        p.elems = append([]Value{val}, p.elems...)
    }
    return p
}
func (p *barecomp) suffix(ctx Context, val Value) Value {
    p = &barecomp{elements{p.elems}}
    if o, t := val.(*barecomp); t {
        p.elems = append(p.elems, o.elems...)
    } else {
        p.elems = append(p.elems, val)
    }
    return p
}

// barefile reduces file lookups, it works like an alias of a File.
type barefile struct {
    Value
    File *File
}
func (_ *barefile) kind() Kind { return KindBarefile }
func (p *barefile) srclit(o Object) string { return srclit(o, p.Value) }
func (p *barefile) String() string { return p.srclit(nil) }
func (p *barefile) string(ctx Context) string {
    if p.File != nil {
        return p.File.string(ctx)
    } else {
        return p.Value.string(ctx)
    }
}
func (p *barefile) true(ctx Context) (t bool) {
    if p.File != nil { t = p.File.true(ctx) }
    return
}
func (p *barefile) int(ctx Context) (res int64, _ error) {
    if p.File.exists() { res = p.File.info.Size() }
    return
}
func (p *barefile) float(ctx Context) (f float64, _ error) {
    i, e := p.int(ctx); return float64(i), e
}
func (p *barefile) refs(ctx Context, v Value) bool { return p.Value.refs(ctx, v) }
func (p *barefile) defs(ctx Context, s ...string) []*def { return p.Value.defs(ctx, s...) }
func (p *barefile) expandable(ctx Context) bool { return p.Value.expandable(ctx) }
func (p *barefile) expand(ctx Context) (res Value) {
    if _exFullFile(ctx) {
        var f = p.File
        if f == nil { f = file(ctx, p.Value.string(ctx)) }
        if f != nil { if v := f.expand(ctx); v != f { return v }}
    }

    if name := p.Value.expand(ctx); name != p.Value {
        res = &barefile{name, p.File}
    } else {
        res = p
    }
    return
}
func (p *barefile) traverse(ctx Context) {
    if p.File != nil { p.File.traverse(ctx) } else
    if p.Value != nil { p.Value.traverse(ctx) }
}
func (p *barefile) updated(ctx Context) (res bool) {
    if p.File != nil { res = p.File.updated(ctx) }
    return
}
func (p *barefile) updatedDeps(ctx Context, v ...Value) (res []Value) {
    if p.File != nil { res = p.File.updatedDeps(ctx, v...) }
    return
}
func (p *barefile) delete(ctx Context) (files []*File, err error) {
    if p.File != nil { files, err = p.File.delete(ctx) }
    return
}
func (p *barefile) stamp(ctx Context) (files []*File, err error) {
    if p.File != nil { files, err = p.File.stamp(ctx) }
    return
}
func (p *barefile) stat(ctx Context) (si *statinfo) {
    if p.File != nil { si = p.File.stat(ctx) }
    return
}
func (p *barefile) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, y := v.(*barefile); y {
        res = p.Value.cmp(ctx, a.Value)
    } else if a, y := toFile(v); y && p.File != nil {
        res = p.File.cmp(ctx, a)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *barefile) patterned(ctx Context) bool { return p.Value.patterned(ctx) }
func (p *barefile) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    if false && p.File != nil {
        full, s, stems = p.File.match(ctx, i)
    } else {
        full, s, stems = p.Value.match(ctx, i)
    }
    return
}
func (p *barefile) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var name Value
    if p.File != nil {
        val, rest = p.File.stencil(ctx, stems)
    } else if name, rest = p.Value.stencil(ctx, stems); name != p.Value {
        val = &barefile{name, p.File}
    } else {
        val = p
    }
    return
}

func barefilize(ctx Context, targets ...Value) []Value {
    var project = ctx.project()
    for i, target := range targets {
        if target.patterned(ctx) { continue }
        switch t := target.(type) {
        case *bareword:
            if file := project.file(ctx, t.s); file != nil {
                targets[i] = &barefile{ target, file }
                file.position = target.Position()
            }
        case *barecomp, *path:
            if t.patterned(ctx) || t.expandable(ctx) /* || refdef(ctx, t, DefArg) */ {//, expandDef2
                break
            } else if file := project.file(ctx, t.string(ctx)); file != nil {
                targets[i] = &barefile{ target, file }
                file.position = target.Position()
            }
        case *argumented:
            t.Value = barefilize(ctx, t.Value)[0]
            t.args  = barefilize(ctx, t.args...)
        }
    }
    return targets
}
func exp_barefilize(ctx Context, targets ...Value) (res []Value) {
    var ( project = ctx.project() ; maps []matchedFileMap )
    for _, target := range targets {
        if !target.patterned(ctx) {
            maps = append(maps, files(ctx, target, project)...)
        }
    }
    for _, file := range project.selectFiles(ctx, maps) {
        res = append(res, file)
    }
    return
}

const windowsOS = runtime.GOOS == "windows"
var ErrBadPattern = errors.New("syntax error in pattern")

// modified copy of filepath.hasMeta
func globHasMeta(path string) bool {
    magicChars := `*?[`
    if !windowsOS {
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
    if chunk[0] == '\\' && !windowsOS {
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
func globScanChunk(pattern string) (stars int, chunk, rest string) {
    for len(pattern) > 0 && pattern[0] == '*' {
        pattern = pattern[1:]
        stars += 1 // TODO: support both '*' and '**'
    }

    inrange := false

    var i int
Scan:
    for i = 0; i < len(pattern); i++ {
        switch pattern[i] {
        case '\\':
            if !windowsOS {
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
    return stars, pattern[0:i], pattern[i:]
}

// modified copy of filepath.matchChunk
// stems: all values matched ? and [...]
func globMatchChunk(chunk, s string) (stems []string, rest string, ok bool, err error) {
    // After the match fails (failed==true), the loop continues on processing chunk,
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
                if s[0] == pathSepByte {
                    failed = true
                }
                _, n := utf8.DecodeRuneInString(s)
                stems = append(stems, s[:n])
                s = s[n:]
            }
            chunk = chunk[1:]

        case '\\':
            if !windowsOS {
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

// modified copy of filepath.Match, use ** to match across path separators
func globMatch(ctx Context, pattern, name string) (matched bool, stems []string, err error) {
    var _pattern, _name = pattern, name
    var dbg = false && (
        (_pattern == "lib*.a" && _name == "libunwind.a") ||
        (_pattern == "libun????.a" && _name == "libunwind.a") ||
        (_pattern == "lib[a-z][^0-9]????.a" && _name == "libunwind.a") ||
        (_pattern == "lib?++.a" && _name == "libc++.a") ||
        false)

    if dbg {
        prompt(ctx, "(%v, %v):\n", _pattern, _name)
        defer func() {
            prompt(ctx, "    matched=%v, stems=%v, remains(pattern=%v, name=%v)\n", matched, stems, pattern, name)
        } ()
    }

Pattern:
    for len(pattern) > 0 {
        var stars int
        var chunk string // stars or chunk: ? [...] a..z
        stars, chunk, pattern = globScanChunk(pattern)
        if dbg {
            prompt(ctx, "    scan: stars=%v, chunk=%v, pattern=%v\n", stars, chunk, pattern)
        }
        if stars > 0 && chunk == "" { // no stars or glob chunks (metas)
            // Trailing * matches rest of string unless it has a /.
            if matched = stars > 1 || !strings.Contains(name, pathSep); matched {
                if true || name != "" { stems = append(stems, name) }
            }
            return
        }

        // Look for match at current position.
        ss, t, ok, err := globMatchChunk(chunk, name)
        if dbg {
            prompt(ctx, "    match: (%v, %v) -> ss=%v, t=%v, ok=%v\n", chunk, name, ss, t, ok)
        }
        if len(ss) > 0 { stems = append(stems, ss...) }

        // if we're the last chunk, make sure we've exhausted the name
        // otherwise we'll give a false result even if we could still match
        // using the stars
        if ok && (len(t) == 0 || len(pattern) > 0) {
            name = t
            continue
        }

        if err != nil { return false, stems, err }
        if stars > 0 {
            // Look for match skipping i+1 bytes. Cannot skip /.
            for i := 0; i < len(name) && (name[i] != pathSepByte || stars > 1); i++ {
                ss, t, ok, err := globMatchChunk(chunk, name[i+1:])
                if dbg {
                    prompt(ctx, "    match: name=%v, (%v, %v), %s -> ss=%v, t=%v, ok=%v\n", name, chunk, name[i+1:], name[:i+1], ss, t, ok)
                }
                if ok {
                    // if we're the last chunk, make sure we exhausted the name
                    if len(pattern) == 0 && len(t) > 0 {
                        continue
                    }
                    stems = append(stems, name[:i+1])
                    if len(ss) > 0 { stems = append(stems, ss...) }
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

type globmeta struct { valbase ; token }
func (p *globmeta) String() string { return p.token.String() }
func (p *globmeta) string(ctx Context) string { return p.token.String() }
func (p *globmeta) expand(Context) Value { return p }
func (p *globmeta) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, ok := v.(*globmeta); ok {
        if p.token == a.token { res = cmpEqual }
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *globmeta) filecache(ctx Context, fc *filecache) (res *filecache, done bool) {
    if res, done = fc.hit(ctx, p.token); res == nil {
        if cacheMapping(ctx) {
            erro(ctx, "no filecache for %v : %v", p.token, fc).debug(3)
        }
    }
    return
}
func (p *globmeta) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    return cache.fix(ctx, p.token.String(), bits)
}
func (_ *globmeta) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

// `[a-b]`, `[abc]`, ...
// `a-b`, `abc`, `a$(var)c`, `a$(spaces)c`...
type globrange struct { valbase ; Chars Value }
func (p *globrange) srclit(o Object) string {
    return fmt.Sprintf("[%s]", srclit(o, p.Chars))
}
func (p *globrange) String() (s string) { return p.srclit(nil) }
func (p *globrange) string(ctx Context) (s string) {
    return fmt.Sprintf("[%s]", p.Chars.string(ctx))
}
func (p *globrange) refs(ctx Context, v Value) bool { return p.Chars.refs(ctx, v) }
func (p *globrange) defs(ctx Context, s ...string) []*def { return p.Chars.defs(ctx, s...) }
func (p *globrange) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, ok := v.(*globrange); ok {
        res = p.Chars.cmp(ctx, a.Chars)
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *globrange) expandable(ctx Context) bool { return p.Chars.expandable(ctx) }
func (p *globrange) expand(ctx Context) (res Value) {
    if val := p.Chars.expand(ctx); val != p.Chars {
        res = &globrange{p.valbase, val}
    } else {
        res = p
    }
    return
}
func (p *globrange) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if cache = cache.fix(ctx, "[]", bits); cache != nil {
        for _, c := range p.Chars.string(ctx) {
            warn(ctx, "range: %v: %s", p.Chars, c)
        }
        warnstack(ctx, 3, "%v: %v", p, p.Chars).debug(1)
    }
    return cache
}
func (_ *globrange) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "collect unsupported: %v", cache).debug(32)
    return
}

type path struct { elements }
func (_ *path) kind() Kind { return KindPath }
func (p *path) srclit(o Object) (s string) {
    for i, elem := range p.elems {
        var v = srclit(o, elem)
        if i > 0 {
            s += pathSep + v
        } else if v != "" {
            s += v
        } else if len(p.elems) == 1 {
            s += pathSep
        }
    }
    return
}
func (p *path) String() (s string) {
    var n int
    for i, elem := range p.elems {
        var v = elem.String()
        if 0 < i {
            s += pathSep + v
            n += 1
        } else if v != "" {
            s += v
        } else if len(p.elems) == 1 {
            s += pathSep
            n += 1
        }
    }
    if n == 0 { s = "{=path "+s+"}" }
    return
}
func (p *path) string(ctx Context) (s string) {
    for i, seg := range p.elems {
        if seg == nil {
            erro(ctx, "`%s` nil path segment", p).debug(1)
            return
        }

        var v string
        if isUndef(seg) {
            erro(at(ctx,seg.Position()), "undef path segment (%T)", seg)
            erro(at(ctx,ctx.Position()), "… from this context: %s", ctx).debug(16)
            return
        } else if v = seg.string(ctx); i > 0 {
            s += pathSep + v
        } else if v != "" {
            s += v
        } else if len(p.elems) == 1 {
            s += pathSep
        }
    }
    return
}
func (p *path) true(ctx Context) (t bool) {
    // FIXME: return p.exists() ??
    for _, elem := range p.elems {
        if t = elem.true(ctx); t { break }
    }
    return
}
func (p *path) isAbs() (_ bool) {
    if x, y := p.elems[0].(*pathpun); y && x.token == PROOT {
        return true
    }
    return
}
func (p *path) float(ctx Context) (_ float64, _ error) { return }
func (p *path) int(ctx Context) (_ int64, _ error) { return }
func (p *path) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *path) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *path) expandable(ctx Context) bool { return p.elements.expandable(ctx) }
func (p *path) expand(ctx Context) Value {
    if elems := expandPathElems(ctx, p.elems...); diff(ctx, elems, p.elems) {
        return &path{elements{elems}}
    } else {
        return p
    }
}
func (p *path) delete(ctx Context) (files []*File, err error) {
    if si := p.stat(ctx); si == nil || si.file == nil {
        erro(ctx, "no path name for `%s`", p)
    } else if files, err = si.file.delete(ctx); err != nil {
        erro(ctx, "stamp: %v (%v)", err, si.file)
    }
    return
}
func (p *path) stamp(ctx Context) (files []*File, err error) {
    if si := p.stat(ctx); si == nil || si.file == nil {
        erro(ctx, "no path name for `%s`", p)
    } else if files, err = si.file.stamp(ctx); err != nil {
        erro(ctx, "stamp: %v (%v)", err, si.file)
    }
    return
}
func (p *path) stat(ctx Context) (si *statinfo) {
    var s string
    if p.patterned(ctx) {
        if val, rest := p.stencil(ctx, _stems(ctx)); len(rest) > 0 {
            erro(ctx, "partial match: %v", rest)
            return
        } else {
            s = val.string(ctx)
        }
    } else {
        s = p.string(ctx)
    }

    if filepath.IsAbs(s) {
        if file := stat(ctx, s, stat_nonexist{true}); file != nil {
            return &statinfo{ file:file }
        }
    }

    if file := file(ctx, s); file != nil {
        si = file.stat(ctx)
    }
    return
}
func (p *path) traverse(ctx Context) { ctx.traverse(at(ctx,p), p) }
func (p *path) patterned(ctx Context) (result bool) {
    for _, seg := range p.elems {
        if result = seg.patterned(ctx); result { break }
    }
    return
}
func (p *path) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }
    if x, y := v.(*path); y {
        return compareElems(ctx, p.elems, x.elems)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    } else if o, y := v.(fullname); y && o.Value != nil {
        if res = o.cmp(ctx, p); res != cmpUnknown && res != cmpEqual {
            res = cmpres(-res)
        }
        return
    } else if f, y := v.(*File); y {
        if s := p.string(ctx); f.ident(ctx) == s { return cmpEqual }
        return
    } else if len(p.elems) == 1 {
        return p.elems[0].cmp(ctx, v)
    }
    return
}
func (p *path) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        noted(at(ctx,p), "%v, %v, %v", p, p==v, res)
        noted(at(ctx,p), "%v", us(p))
        noted(at(ctx,v), "%v", us(v))
        erro(ctx, "%v", us(ctx)).debug(5)
    }
}
func (p *path) filecache(ctx Context, _c *filecache) (res *filecache, done bool) {
    var c = _c
    for x := (pathcache{ctx,p,0}); x.i < p.len(); x.i += 1 {
        var elem = p.elems[x.i]
        if indeterminate(ctx, elem) {
            erro(at(ctx,elem), "filecache %v : indeterminate element : %v", p, us(elem)).debug(3)
            return
        } else if c, done = c.hit(&x, elem) ; c == nil {
            if cacheMapping(ctx) {
                erro(at(ctx,elem), "no filecache for %v : %v", us(elem), _c).debug(3)
            }
            return
        } else {
            if res = c ; done { return }
        }
    }
    return
}
func (p *path) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    for _, elem := range expandPathElems(ctx, p.elems...) {
        cache = cache.slot(ctx, elem, bits)
    }
    return cache
}
func (p *path) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    var elems = expandPathElems(ctx, p.elems...)

    bits |= cachePath

    var cs = []*valcache{ cache }
    for n, elem := range elems {
        var str string
        var s []string
        var t []*valcache
        for _, c := range cs {
            if a := elem.collect(ctx, c, bits); len(a) != 0 {
                t = append(t, a...)
            } else if x, y := c._fix["**"]; y {
                if s == nil && str == "" {
                    for _, v := range elems[n:] { s = append(s, v.string(ctx)) }
                    str = strings.Join(s, pathSep)
                }
                for k, a := range x._fix { // see valcache.matchPatts
                    if k == "" { // TODO: empty
                        warn(at(ctx, p), "%v: %d %s, %v; %v, %v", p, n, typeof(elem), elem, s, a).debug(16)
                        continue
                    }

                    if i := strings.Index(str, k); i < 0 { continue } else
                    if str = str[i+len(k):]; str != "" {
                        x = a // TODO: call to valcache.matchPatts ???
                    } else {
                        if false { warn(at(ctx, p), "%v[%d], %s; %v, %v, %v", p, n,
                            typeof(elem), elem, s, a._key).debug(16) }

                        res = append(res, a)
                        break
                    }
                }
            }
        }
        cs = t
    }
    if cs != nil { res = append(res, cs...) }

    // TODO: check cache._fix["**"]
    return
}

func matchPathSeg(ctx Context, seg Value, src string) (bool, string, []string) {
    var full, i, ss = seg.match(ctx, src)
    var s = joinPathStr(ctx, i)
    // if !full {
    //     if x, y := seg.(*pathpun); y && PTAIL == x.token && numSrc < lenSrcs {
    //         // ...
    //     }
    // }
    return full, s, ss
}

func (p *path) match2(ctx Context, srcs ...string) (full bool, res []string, stems []string) {
    if checkpoints {
        var s, t = joinPath(srcs...), p.string(ctx)
        if strings.HasPrefix(s, t) { defer func() { if res == nil {
            noted(ctx, "%v →", p)
            noted(ctx, "%v →", s)
            noted(ctx, "→ %v %v %v", full, res, stems)
            erro(ctx, "%v", us(ctx)).debug(5)
        }}()}
    }

    if len(srcs) == 0 {
        if false { erro(at(ctx,p), "empty: %v", srcs) }
        return
    }

    var segs = expandPathElems(ctx, p.elems...)
    if expandable(final{ctx}, segs...) {
        if false { erro(ctx, "expandable path: %v, %v", p, segs).debug(1) }
        return
    }

    var (
        lenSegs = len(segs)
        lenSrcs = len(srcs)
        nxtSeg, nxtSrc int
        undone func() bool
        app func([]string, ...string) []string
        pun func(*pathpun) bool
        reverse bool
        step int
    )
    if reverse, _ = ctx.do(propReversal).(bool); reverse {
        app = func(a []string, s ...string) []string { return append(s, a...) }
        pun = func(x *pathpun) bool { return PROOT == x.token && 0 <= nxtSrc }
        step, undone = -1, func() bool { return 0 <= nxtSeg && 0 <= nxtSrc }
        nxtSeg, nxtSrc = lenSegs-1, lenSrcs-1
    } else {
        app = func(a []string, s ...string) []string { return append(a, s...) }
        pun = func(x *pathpun) bool { return PTAIL == x.token && nxtSrc < lenSrcs }
        step, undone = 1, func() bool { return nxtSeg < lenSegs && nxtSrc < lenSrcs }
    }

    if true { defer func() { if full && len(stems) == 0 && len(res) > 0 && p.patterned(ctx) {
        if lenSegs == 1 /* && lenSrcs == 1 */ && len(res) == 1 && segs[0].patterned(ctx) {
            stems = res
        } else {
            ctx = at(ctx, p)
            warn(ctx, "incorrect full match: %v: srcs=%s, res=%v, stems=%v", p, srcs, res, stems)
            warnstack(ctx, 3).debug(6)
        }
    }}()}

    for undone() {
        var seg = segs[nxtSeg]; nxtSeg += step // move to the next seg
        if s := correctPathPunForMatch(seg); s == nil {
            erro(at(ctx,seg), "invalid path segment: %v", tv(seg)).debug(1)
            return
        } else {
            seg = s
        }

        var multi, pre, suf = multia(ctx, seg) // %% or **
        var src = srcs[nxtSrc]; nxtSrc += step // move to the next src

        if multi {
            var stem []string
            var st, prefix, suffix string
            if !isTrivial(pre) { prefix = pre.string(ctx) }
            if !isTrivial(suf) { suffix = suf.string(ctx) }

            if prefix == "" {
                st = src[:]
            } else if strings.HasPrefix(src, prefix) {
                st = strings.TrimPrefix(src, prefix)
            } else {
                return
            }

            var nful bool
            var tail []string // stem
            if suffix != "" {
                for {
                    res = app(res, src)

                    if strings.HasSuffix(st, suffix) {
                        st = strings.TrimSuffix(st, suffix)
                        stem = app(stem, st)
                        if nxtSeg == lenSegs {
                            full = nxtSrc == lenSrcs
                        } else {
                            full = nxtSeg == lenSegs-1
                        }
                        break
                    } else if prefix == "" && st == "" {
                        stem = app(stem, src)
                    } else {
                        stem = app(stem, st)
                    }

                    if nxtSrc < lenSrcs {
                        src = srcs[nxtSrc] ; nxtSrc += step
                        st = src[:]
                    } else {
                        full = nxtSeg == lenSegs-1
                        nful = !full
                        break
                    }
                }
            } else if nxtSeg < lenSegs {
                if prefix == "" || st != "" { res = app(res, src) }
                if st == "" { st = src }   ; stem = app(stem, st)

                prefix = ""

                var con bool
                var nxt = segs[nxtSeg]
                if multi, pre, suf = multia(ctx, nxt); multi { // x%%y or x**y
                    if !isTrivial(pre) { prefix = pre.string(ctx) }
                    if !isTrivial(suf) { suffix = suf.string(ctx) }
                    con = prefix == "" // x**/**y
                }

                // Finding the best match stopped by nxt:
                for ; nxtSrc < lenSrcs ; nxtSrc += step { if src = srcs[nxtSrc]; multi {
                    if prefix == "" { res = app(res, src)
                        if suffix != "" && strings.HasSuffix(src, suffix) {
                            if st = strings.TrimSuffix(src, suffix); con { // x**/**y
                                stems = app(stems, joinPath(stem...))
                                stem = []string{ st }
                            } else {
                                stem = app(stem, st)
                            }
                            full = nxtSrc == lenSrcs && nxtSeg+1 == lenSegs
                            nxtSrc += step
                            break
                        } else {
                            stem = app(stem, src)
                        }
                    } else if strings.HasPrefix(src, prefix) {
                        if len(stem) > 0 { stems = app(stems, joinPath(stem...)) }
                        stem = []string{ strings.TrimPrefix(src, prefix) }
                        nxtSrc += step
                        break
                    } else {
                        stem = app(stem, src)
                    }
                } else if y, s, ss := matchPathSeg(ctx, nxt, src); y || s == src {
                    if res = app(res, s); len(ss) == 0 {
                        if false { stem = app(stem, src) }
                        if false { noted(ctx, "%v %v , %v %v", p, stem, nxt, src) }
                    } else {
                        tail = ss
                    }

                    nxtSeg += step
                    full = nxtSeg == lenSegs && nxtSrc == lenSrcs
                    nxtSrc += step
                    break
                } else {
                    res = app(res, src)
                    stem = app(stem, src)
                }}
            } else {
                res = app(res, src)
                stem = app(stem, st)
                if nxtSrc < lenSrcs {
                    var t = srcs[nxtSrc:]
                    res = app(res, t...)
                    stem = app(stem, t...)
                    nxtSrc, full = lenSrcs, true
                }
            }

            if !full && !nful { full = nxtSrc == lenSrcs }

            if len(stem) > 0 { stems = app(stems, joinPath(stem...)) }
            if len(tail) > 0 { stems = app(stems, tail...) }
        } else if y, s, ss := matchPathSeg(ctx, seg, src); y {
            res = app(res, s)
            stems = app(stems, ss...)
            full = nxtSeg == lenSegs && nxtSrc == lenSrcs
        } else if x, y := seg.(*pathpun); y && pun(x) {
            res = app(res, "")
            return
        } else {
            if checkpoints {
                if seg.string(ctx) == src {
                    erro(ctx, "%v : %s == %s, (%d,%d) ; %v ; %s %s", p, us(seg), us(src), nxtSeg, nxtSrc, srcs, s, ss).debug(3)
                }
                if false && y {
                    noted(ctx, "%v: %d. %v , %d. %v ; %v %v", p, nxtSeg, us(seg), nxtSrc, us(src), s, ss).debug(1)
                }
            }
            return
        }
    }
    return
}
func (p *path) match1(ctx Context, str string) (full bool, result []string, stems []string) {
    if srcs := strings.Split(str, pathSep); 0 < len(srcs) {
        return p.match2(ctx, srcs...)
    }
    return
}
func (p *path) match(ctx Context, i interface{}) (full bool, res interface{}, stems []string) {
    var result []string

    defer func() {
        if n := len(result); n == 1 {
            res = result[0]
        } else if n > 1 {
            res = result
        }
    } ()

    switch t := i.(type) {
    case []string:
        if n := len(t); n == 1 {
            full, result, stems = p.match1(ctx, t[0])
            return
        } else if n > 1 {
            full, result, stems = p.match2(ctx, t...)
            return
        } else {
            return
        }
    case string:
        if t != "" {
            full, result, stems = p.match1(ctx, t)
            return
        } else {
            return
        }
    case *filestub:
        if full, result, stems = p.match1(ctx, t.name); full || len(result) > 0 {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            full, result, stems = p.match1(ctx, filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
            return
        } else {
            return
        }
    case *File:
        if full, result, stems = p.match1(ctx, t.ident(ctx)); full || len(result) > 0 {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            full, result, stems = p.match1(ctx, t.fullname()) // NOTE: matching the fullname form
            return
        } else {
            return
        }
    case Value:
        if s := t.string(ctx); s != "" {
            full, result, stems = p.match1(ctx, s)
            return
        } else {
            return
        }
    default:
        erro(at(ctx,p), "path.match unsupport %v", tv(i)).debug(16)
        return
    }
}

func (p *path) stencil(ctx Context, stems []string) (result Value, rest []string) {
    var changed int
    var elems []Value
    for _, seg := range xmerge(ctx, p.elems...) {
        var val Value
        if val, stems = seg.stencil(ctx, stems); !isTrivial(val) {
            if val != seg { changed += 1 }
            elems = append(elems, val)
        } else {
            elems = append(elems, seg)
        }
    }
    if rest = stems; changed > 0 {
        result = makePath(elems...)
    } else {
        result = p
    }
    return
}

func (p *path) suffix(ctx Context, val Value) (res Value) {
    if isTrivial(val) {
        erro(at(ctx,p), "path combines invalid value: %v", val).debug(1)
        return
    }
    if _, y := p.elems[0].(*pathpun); y /* && t.token == PLUS */ {
        noted(at(ctx,p), "%v %v", p, val).debug(10)
    }

    var ti = p.len()-1
    var tv = p.elems[ti]
    if tv == nil {
        erro(at(ctx,p), "path has nil tail").debug(1)
        return
    }

    var comp *barecomp

    if x, y := tv.(*pathpun); y {
        switch x.token {
        case 0, PCON: comp = makeBarecomp()
        default: comp = makeBarecomp(tv)
        }
    } else if comp, y = tv.(*barecomp); !y {
        if isTrivial(tv) {
            comp = makeBarecomp()
        } else {
            comp = makeBarecomp(tv)
        }
    }

    p = &path{elements{append(p.elems[:ti], comp)}}

    if x, y := val.(*path); y {
        if t, y := x.elems[0].(*pathpun); y {
            switch t.token { case 0, PCON: goto apptail }
        }
        p.elems[ti] = comp.suffix(ctx, x.elems[0])
    apptail:
        p.elems = append(p.elems, x.elems[1:]...)
    } else {
        p.elems[ti] = comp.suffix(ctx, val)
    }

    return p
}

func _pathstr(ctx Context, str string) *path {
    return makePath(splitPathStr(ctx, str)...)
}

func _pathpun(ctx Context, tok token) *pathpun {
    return &pathpun{valbase{ctx.Position()}, tok}
}

type pathpun struct { valbase; token } // TODO: use token instead of rune
func (p *pathpun) String() (s string) { return p.token.String() }
func (p *pathpun) string(ctx Context) (s string) { return p.token.String() }
func (p *pathpun) expand(Context) Value { return p }
func (p *pathpun) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }
    switch t := v.(type) {
    case *pathpun: if p.token == t.token { return cmpEqual }
    case *path: if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    case *list: if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    }
    return
}
func (p *pathpun) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
    }
}
func (p *pathpun) match(ctx Context, i interface{}) (full bool, result interface{}, stems []string) {
    var s string
    switch t := i.(type) {
    case string: s = t
    case Value:
        if indeterminate(ctx, t) {
            erro(ctx, "%v: pathpun.match unexpanded: %v", p, us(i)).debug(5)
            return
        }

        s = t.string(ctx)

    default:
        erro(ctx, "%v: pathpun.match unsupported: %v", p, us(i)).debug(5)
        return
    }

    if s == p.token.String() { full, result = true, s }
    return
}
func (p *pathpun) stencil(ctx Context, stems []string) (result Value, rest []string) {
    return p, stems
}

func (p *pathpun) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if p.token == PCON { res = cache } else {
        return cache.str(/* at(ctx, p.position) */ctx, p.String(), bits)
    }
    return
}
func (p *pathpun) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if p.token != PCON { if c := cache.str(ctx, p.String(), cacheZero); c != nil {
        res = append(res, c)
    }}
    return
}

func toFile(v Value) (f *File, y bool) {
    if f, y = v.(*File); !y {
        switch t := v.(type) {
        case fullfile: f, y = t.File, true
        case fullname: f, y = toFile(t.Value)
        case as: f, y = toFile(t.Value)
        }
    }
    return
}

func splitFileName(ctx Context, val Value) (dir, name string) {
    if f, _ := toFile(val); f != nil {
        dir, name = filepath.Join(f.dir, f.sub), f.ident(ctx)
    } else {
        name = val.string(ctx)
        dir = filepath.Dir(name)
    }
    return
}

type fullname struct { Value }
func (o fullname) expand(ctx Context) Value { return fullname{o.Value.expand(ctx)} }
func (o fullname) string(ctx Context) (res string) {
    if x, y := o.Value.(*File); y && x != nil { return x.fullname() }
    return o.Value.string(ctx)
}
func (o fullname) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { o.cmp_check(ctx, v, res) }() }
    switch t := v.(type) {
    case *list:
        if t.len() == 1 { return o.cmp(ctx, t.elems[0]) }
    case *File:
        if x, y := o.Value.(*File); y {
            if res = x.cmp(ctx, t); res == cmpUnknown {
                a, b := x.fullname(), t.fullname()
                if a == b {
                    return cmpEqual
                } else if a < b {
                    return cmpSmaller
                } else if a > b {
                    return cmpGreater
                }
            }
            return
        } else {
            return o.Value.cmp(ctx, t)
        }
    case *path:
        if x, y := o.Value.(*File); y {
            if t.isAbs() && isFinalValue(ctx, t) {
                a, b := x.fullname(), t.string(ctx)
                if a == b {
                    return cmpEqual
                } else if a < b {
                    return cmpSmaller
                } else if a > b {
                    return cmpGreater
                }
            }
            return
        } else {
            return o.Value.cmp(ctx, t)
        }
    case fullname:
        return o.Value.cmp(ctx, t.Value)
    default:
        return o.Value.cmp(ctx, v)
    }
    return
}
func (o fullname) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && o.String() == v.String() {
        erro(ctx, "%v != %v, %v", us(o), us(v), res).debug(5)
    }
}

type fullfile struct { *File }
func (u fullfile) string(ctx Context) string { return u.fullname() }
func (u fullfile) expand(ctx Context) Value { return u }

type File struct {
    valbase
    *filebase
    *filestub
}
func (p *File) String() string { return "{=file "+p.filestub.name+"}" }
func (p *File) string(ctx Context) (s string) { return p.filestub.name }
func (p *File) hash(h *maphash.Hash) { h.WriteString(p.fullname()) }
func (p *File) ident(Context) string { return p.filestub.name }
func (p *File) true(ctx Context) (t bool) {
    if p.filestub.name != "" { t = true } // p.exists() == existenceConfirmed
    return
}
func (p *File) BaseName() (s string) {
    if p.info != nil { s = p.info.Name() } else {
        s = filepath.Base(p.filestub.name)
    }
    return
}
func (p *File) fullname() (s string) {
    return filepath.Join(p.dir, p.sub, p.filestub.name)
}
func (p *File) searchInMatchedPaths(ctx Context, proj *project) (res bool) {
    if p.filemap != nil {
        // FIXME: File should keep both 'match' and 'pre', or just remove searchInMatchedPaths
        var f = p.filemap.stat(ctx, p.filestub.name)
        if f != nil && f.info != nil { p.info, res = f.info, true }
    }
    return
}
func (p *File) absolute_delete(ctx Context) (files []*File, err error) {
    if positionalValueCtx { ctx = at(ctx, p.position) }

    var fullname string
    if fullname = p.fullname(); fullname == "" {
        erro(at(ctx,p), "file `%s` has no fullname", p).debug(1)
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
        erro(at(ctx,p), "file `%s` has no fullname", p).debug(1)
    } else if p.info, err = os.Stat(fullname); err != nil {
        if false { erro(ctx, "%v", err).debug(1) }
    } else if p.info == nil {
        if false { warn(ctx, "%v: no such file", p).debug(1) }
    } else if files = append(files, p); !isConfigure(ctx) {
        p._updated = true
        ctx.dirtyMark(p)
    }
    return
}
func (p *File) expandable(ctx Context) bool {
    return _exFullFile(ctx) && !filepath.IsAbs(p.filestub.name)
}
func (p *File) expand(ctx Context) Value {
    if _exFullFile(ctx) {
        return fullfile{p}
    } else {
        return p
    }
}
func (p *File) exists() (res bool) {
    if p != nil && p.filebase != nil {
        res = p.filebase.exists()
    }
    return
}
func (p *File) updated(ctx Context) bool { return p._updated }
func (p *File) updatedDeps(_ Context, v ...Value) []Value {
    if len(v) > 0 { p._updatedDeps = append(p._updatedDeps, v...) }
    return p._updatedDeps
}
func (p *File) stat(ctx Context) (si *statinfo) {
    if err := error(nil); p.info != nil {
        // good already
    } else if p.info, err = os.Stat(p.fullname()); err == nil {
        // good
    } else if pe, ok := err.(*fs.PathError); ok {
        if false {
            erro(at(ctx,p.position), "File.stat %v: %v", trimPromptString(pe.Path), pe.Err).debug(1)
        }
        return
    } else {
        erro(at(ctx,p.position), "File.stat: %v", err).debug(1)
        return
    }
    return &statinfo{ file: p }
}
func (p *File) isSysFile() (res bool) {
    if p.filemap != nil && len(p.filemap.paths) == 1 {
        // system files defined by:
        //     files (
        //       (foo.xxx) ⇒ -
        //     )
        if f, ok := p.filemap.paths[0].(flag); ok {
            res = isNone(f.Value) || isNull(f.Value)
        }
    }
    return
}
func (p *File) traverse(ctx Context) {
    ctx = at(ctx, p.position)
    if !p.isSysFile() && p._traved == 0 {
        ctx.traverse(ctx, p)
    } else if pc := cast[*programContext](ctx); pc != nil {
        pc.deferTrave(ctx, getTargetValue(ctx), p, nil, p)
    }
}

func (p *File) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    switch t := v.(type) {
    case *list: if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    case *barefile: if t.File != nil { return p.cmp(ctx, t.File) }
    case *barecomp, *bareword, *path:
        if s := v.string(ctx); s == p.filestub.name { res = cmpEqual }
    default:
        if x, y := toFile(v); !y {
            res = cmpUnknown
        } else if p.filebase == x.filebase {
            res = cmpEqual
        } else if checkpoints && p.fullname() == x.fullname() {
            erro(ctx, "same files: %v != %v", us(p), us(v)).debug(5)
        }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *File) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    return cache.strx(at(ctx, p.position), p.filestub.name, bits|cacheFile)
}
func (p *File) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if c := cache.strx(ctx, p.filestub.name, bits); c != nil { res = append(res, c) }
    return
}

func (p *File) patterned(ctx Context, ) bool { return false }
func (p *File) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *File) match1(ctx Context, v string) (full bool, s string, stems []string) {
    if name := p.filestub.name; name == v {
        s, full = name, true
    } else if name = filepath.Join(p.sub, p.filestub.name); name == v {
        s, full = name, true
    }
    return
}
func (p *File) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    switch t := i.(type) {
    case string: return p.match1(ctx, t)
    case Value:
        if !isTrivial(t) {
            return p.match1(ctx, t.string(ctx))
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

type filecontent struct { *File ; content []byte }

type flag struct { Value }
func (p flag) kind() Kind { return KindFlag }
func (p flag) String() (s string) { return p.srclit(nil) }
func (p flag) string(ctx Context) (s string) {
    if s = "-"; p.Value != nil && !isNone(p.Value) && !isNull(p.Value) {
        s += p.Value.string(ctx)
    }
    return
}
func (p flag) srclit(o Object) (s string) {
    if s = "-"; p.Value != nil && !isNone(p.Value) && !isNull(p.Value) {
        s += p.Value.String()
    }
    return
}
func (p flag) int(ctx Context) (i int64, e error) {
    if i, e = p.Value.int(ctx); e == nil { i = -i }
    return
}
func (p flag) float(ctx Context) (f float64, e error) {
    if f, e = p.Value.float(ctx); e == nil { f = -f }
    return
}
func (p flag) Position() (pos Position) {
    pos = p.Value.Position()
    pos.Column -= 1
    return
}
func (p flag) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p flag) suffix(ctx Context, val Value) Value { return flag{compose(ctx, p.Value, val)} }
func (p flag) match(ctx Context, i interface{}) (full bool, res interface{}, stems []string) {
    switch t := i.(type) {
    case flag:
        full, res, stems = p.Value.match(ctx, t.Value)
        if s, y := res.(string); y { res = "-" + s }
    case Value:
        if v := t.string(ctx); strings.HasPrefix(v, "-") {
            full, res, stems = p.Value.match(ctx, v[1:])
            if s, y := res.(string); y { res = "-" + s }
        }
    case string:
        if strings.HasPrefix(t, "-") {
            full, res, stems = p.Value.match(ctx, t[1:])
            if s, y := res.(string); y { res = "-" + s }
        }
    case *none, *null:
    default:
        erro(at(ctx,p), "%v → %v", p, us(i)).debug(16)
    }
    return
}
func (p flag) expand(ctx Context) Value {
    var v = p.Value.expand(ctx)
    if equal(ctx, v, p.Value) { return p }

    var vs []Value
    switch t := v.(type) { case disjunction: vs = merge(t.Value) }
    if vs == nil { return flag{v} }

    var vals []Value
    for _, v := range vs { vals = append(vals, flag{v}) }
    return ease(ctx, vals)
}
func (p flag) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var name Value
    name, rest = p.Value.stencil(ctx, stems)
    if name != nil && name != p.Value {
        val = flag{name}
    } else {
        rest = stems
    }
    return
}
func (p flag) opt(ctx Context, name string) (res string, match bool) {
    if isTrivial(p.Value) {
        if false { erro(at(ctx,p), "flag name is trivial").debug(16) }
    } else if f, y := p.Value.(flag); y {
        res, match = f.opt(ctx, name)
    } else if s := p.Value.string(ctx); s == name {
        res, match = name, true
    }
    return
}
func (p flag) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if v == nil {
        // ...
    } else if a, y := v.(flag); y {
        res = p.Value.cmp(ctx, a.Value)
    } else if c, y := v.(*barecomp); y {
        var elems []Value // right hand side barecomp elements
        for _, elem := range c.elems {
            if !isTrivial(elem) { elems = append(elems, elem) }
        }
        if len(elems) == 2 { if fR, y := elems[0].(flag); y {
            if isTrivial(fR.Value) {
                res = p.Value.cmp(ctx, elems[1])
            } else if m, r, t := fR.Value.match(ctx, p.Value); m {
                if isTrivial(elems[1]) { res = cmpEqual }
            } else if r != nil { // matched prefix
                var s, _ = r.(string)
                var sL = p.Value.string(ctx)
                var sR = s + elems[1].string(ctx)
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
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    } else if i, y := v.(*decimal); y && i.int64 < 0 {
        if a, e := p.Value.int(ctx); e == nil {
            if b := -i.int64; a == b { res = cmpEqual } else
            if a < b { res = cmpGreater } else { res = cmpSmaller }
        }
    } else if f, y := v.(*Float); y && f.float64 < 0 {
        if a, e := p.Value.float(ctx); e == nil {
            if b := -f.float64; a == b { res = cmpEqual } else
            if a < b { res = cmpGreater } else { res = cmpSmaller }
        }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p flag) traverse(ctx Context) { ctx.traverse(at(ctx,p.Value), p) }
func (p flag) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    return cache.str(ctx, "-", bits).slot(ctx, p.Value, bits)
}
func (p flag) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if c := cache.str(ctx, "-", cacheZero); c != nil {
        if c = c.slot(ctx, p.Value, cacheZero); c != nil { res = append(res, c) }
    }
    return
}

const escapedChars = "\"\r\n"

type compound struct { elements } // "compound string"
func (_ *compound) kind() Kind { return KindCompound }
func (p *compound) String() (s string) {
    s = p.srclit(nil)

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
            // erro(at(ctx,p), "%v", err).debug(1)
            panic(err)
            return
        }

        var esc string
        switch s[i] {
        case '"':  esc = `\"`
        case '\r': esc = `\r`
        case '\n': esc = `\n`
        }
        if _, err = buf.WriteString(esc); err != nil {
            // erro(at(ctx,p), "%v", err).debug(1)
            panic(err)
            return
        }

        s = s[i+1:]
        i = strings.IndexAny(s, escapedChars)
    }
    if _, err = buf.WriteString(s); err != nil {
        panic(err)
    }
    return
}
func (p *compound) srclit(o Object) (s string) {
    for _, elem := range p.elems { s += srclit(o, elem) }
    return
}
func (p *compound) string(ctx Context) (s string) {
    if v := p.expand(final{ctx}); equal(ctx, v, p) {
        for _, elem := range p.elems { s += elem.string(ctx) }
    } else {
        s = v.string(ctx)
    }
    return
}
func (p *compound) float(ctx Context) (float64, error) { return strconv.ParseFloat(p.string(ctx), 64) }
func (p *compound) int(ctx Context) (int64, error) { return strconv.ParseInt(p.string(ctx), 10, 64) }
func (p *compound) true(ctx Context) bool { return p.elements.true(ctx) }
func (p *compound) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *compound) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *compound) expandable(ctx Context) bool { return p.elements.expandable(ctx) }
func (p *compound) expand(ctx Context) (res Value) {
    if _exPathStr(ctx) {
        res = _pathstr(ctx, p.string(ctx))
    } else if elems := expand(ctx, p.elems...); diff(ctx, elems, p.elems) {
        res = &compound{elements{elems}}
    } else {
        res = p
    }
    return
}
func (p *compound) match(ctx Context, i interface{}) (bool, interface{}, []string) {
    return stringMatch(ctx, p, i)
}
func (p *compound) stencil(ctx Context, stems []string) (_ Value, _ []string) {
    return p, stems
}
func (p *compound) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, y := v.(*compound); y {
        if p.len() == a.len() {
            for i, elem := range p.elems {
                var r = elem.cmp(ctx, a.elems[i])
                if r != cmpEqual { return r }
            }
            return cmpEqual
        }
    } else if l, y := v.(*list); y {
        if n := l.len(); n == 1 {
            return p.cmp(ctx, l.elems[0])
        } else if n == 0 && 0 == p.len() {
            return cmpEqual
        }
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *compound) traverse(ctx Context) { ctx.traverse(ctx, p) }
func (p *compound) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if cache = cache.str(ctx, "\"\"", bits); true {
        cache = cache.strx(ctx, p.string(ctx), bits)
    } else {
        for _, elem := range expandPathElems(ctx, p.elems...) {
            cache = cache.slot(ctx, elem, bits)
        }
    }
    return cache
}
func (p *compound) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if c := cache.str(ctx, "\"\"", cacheZero); c != nil {
        if c = c.strx(ctx, p.string(ctx), cacheZero); c != nil { res = append(res, c) }
    }
    return
}

type list struct { elements }
func (_ *list) kind() Kind { return KindList }
func (p *list) Position() (pos Position) {
    if 0 < len(p.elems) { pos = p.elems[0].Position() }
    return
}
func (p *list) srclit(o Object) (s string) {
    var strs []string
    for _, elem := range p.elems {
        if s := srclit(o, elem); s != "" { strs = append(strs, s) }
    }
    return strings.Join(strs, " ")
}
func (p *list) String() (s string) {
    var strs []string
    for _, elem := range p.elems {
        if s := elem.String(); s != "" { strs = append(strs, s) }
    }
    return strings.Join(strs, " ")
}
func (p *list) string(ctx Context) (s string) {
    for _, e := range p.elems {
        if e == nil {
            // TODO: special process for nil elements in a list??
        } else if false && cond(e) && indeterminate(ctx, e) {
            continue
        } else if t := e.string(ctx); t != "" {
            if s != "" { s += " " }
            s += t
        }
    }
    return
}
func (p *list) float(ctx Context) (f float64, _ error) {
    i, e := p.int(ctx); return float64(i), e
}
func (p *list) int(ctx Context) (i int64, err error) {
    if n := len(p.elems); n == 1 {
        // If there's only one element, treat it as a scalar.
        return p.elems[0].int(ctx)
    } else {
        return int64(n), nil
    }
}
func (p *list) suffix(ctx Context, val Value) (res Value) {
    var n = p.len()-1
    if n < 0 { return val }

    var a = append(p.elems[:n], compose(ctx, p.elems[n], val))
    return &list{elements{a}}
}
func (p *list) expand(ctx Context) (res Value) {
    defer trace(ctx)

    var a = expand(ctx, p.elems...)
    var d = diff(ctx, a, p.elems)
    if d {
        res = &list{elements{a}}
    } else {
        res = p
    }
    if checkpoints {
        if s1, s2 := us(p.elems), us(a); (d && s1 == s2) || (!d && s1 != s2) {
            for i, v := range a {
                if p.len() <= i {
                    erro(ctx, "%d. {=nil} → %v", i, us(v)).debug(1)
                    continue
                }

                var t = p.elems[i]
                var x = equal(ctx, t, v)
                var y = equal(ctx, v, t)
                if x != y {
                    erro(ctx, "%d. %v → %v → equal→%v,%v", i, us(t), us(v), x, y).debug(1)
                }
            }

            erro(ctx, "wrong: diff=%v : %v → %v", d, s1, s2).debug(1)
        }
    }
    return
}
func (p *list) traverse(ctx Context) {
    var pc = cast[*programContext](ctx)
    for _, elem := range p.elems {
        elem.traverse(ctx)

        if pc.traves.has(traveCase, traveNext, traveDone, traveFail) {
            break
        }
    }
    return
}
func (p *list) updated(ctx Context) (res bool) {
    for _, elem := range p.elems {
        if res = elem.updated(ctx); res { break }
    }
    return
}
func (p *list) updatedDeps(ctx Context, v ...Value) (res []Value) {
    for _, elem := range p.elems {
        res = append(res, elem.updatedDeps(ctx, v...)...)
    }
    return
}
func (p *list) stat(ctx Context) (si *statinfo) {
    if len(p.elems) > 0 {
        for _, elem := range p.elems {
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
func (p *list) delete(ctx Context) (files []*File, err error) {
    for _, elem := range p.elems {
        var a []*File
        if a, err = elem.delete(ctx); err != nil { break }
        files = append(files, a...)
    }
    return
}
func (p *list) stamp(ctx Context) (files []*File, err error) {
    for _, elem := range p.elems {
        var a []*File
        if a, err = elem.stamp(ctx); err != nil { break }
        files = append(files, a...)
    }
    return
}
func (p *list) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) }()}

    if l, y := v.(*list); y {
        return compareElems(ctx, merge(p.elems...), merge(l.elems...))
    } else if 1 == p.len() {
        return p.elems[0].cmp(ctx, v)
    }

    if c, y := v.(*barecomp); y && 2 == c.len() && 1 < p.len() {
        if cl, y := c.elems[1].(*list); y && 1 < cl.len() {
            var a = c.elems[1] // example: p.elems=[z-a b c], c.elems=[a- a b c]
            // FIXME: avoid 'c.elems[1] = cl.elems[0]', the container values are readonly for cmp
            if c.elems[1] = cl.elems[0]; p.elems[0].cmp(ctx, c) == cmpEqual {
                res = compareElems(ctx, cl.elems[1:], p.elems[1:])
            }
            c.elems[1] = a
            return
        }
    }

    return
}
func (p *list) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
    }
    return
}
func (p *list) patterned(ctx Context) (res bool) {
    if len(p.elems) == 1 {
        res = p.elems[0].patterned(ctx)
    } else {
        /* FIXME: check pattern for each element??
        for _, elem := range p.elems {
          if elem.patterned() { return true }
        }*/
    }
    return
}

func (p *list) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    if len(p.elems) == 1 {
        full, s, stems = p.elems[0].match(ctx, i)
    } else {
        /* FIXME: match for to each element??
        for _, elem := range p.elems {
          ...
        }*/
    }
    return
}

func (p *list) stencil(ctx Context, stems []string) (val Value, rest []string) {
    if len(p.elems) == 1 {
        val, rest = p.elems[0].stencil(ctx, stems)
        return
    }

    var (
        elems []Value
        changed int
    )
    rest = stems
    for _, elem := range p.elems {
        var t Value
        if t, rest = elem.stencil(ctx, rest); t != elem { changed += 1 }
        elems = append(elems, t)
    }
    if changed > 0 {
        val = &list{elements{elems}}
    } else {
        val = p
    }
    return
}

func (p *list) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if n := len(p.elems); n == 1 {
        res = p.elems[0].cache(ctx, cache, bits)
    } else {
        errostack(ctx, 5, "cache list of many unsupported (bits=%08b): %v", bits, p).debug(32)
    }
    return
}
func (p *list) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    for _, elem := range p.elems {
        if c := elem.collect(ctx, cache, bits); c != nil { res = append(res, c...) }
    }
    return
}

type group struct { valbase ; elements }
func (_ *group) kind() Kind { return KindGroup }
func (_ *group) ident(Context) (s string) { return }
func (p *group) Position() Position { return p.valbase.Position() }
func (p *group) String() string { return p.srclit(nil) }
func (p *group) string(ctx Context) (s string) {
    s = "("
    for i, elem := range p.elems {
        if i > 0 { s += " " }
        s += elem.string(ctx)
    }
    s += ")"
    return
}
func (p *group) srclit(o Object) string {
    var strs []string
    for _, elem := range p.elems {
        strs = append(strs, srclit(o, elem))
    }
    return fmt.Sprintf("(%s)", strings.Join(strs, " "))
}
func (p *group) true(ctx Context) (t bool) {
    if t = len(p.elems) > 0; t {
        for _, elem := range p.elems {
            if t = elem.true(ctx); !t { break }
        }
    }
    return
}
//func (p *group) float(ctx Context) (f float64, _ error) { return p.valbase.float(ctx) }
//func (p *group) int(ctx Context) (i int64, e error) { return p.valbase.int(ctx) }
func (p *group) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *group) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (_ *group) delete(Context) (_ []*File, _ error) { return }
func (_ *group) patterned(Context) (_ bool) { return }
func (_ *group) stamp(Context) (_ []*File, _ error) { return }
func (_ *group) stat(Context) (_ *statinfo) { return }
func (_ *group) updated(Context) (_ bool) { return }
func (_ *group) updatedDeps(Context, ...Value) (_ []Value) { return }
func (p *group) expandable(ctx Context) bool { return p.elements.expandable(ctx) }
func (p *group) expand(ctx Context) (res Value) {
    if elems := expand(ctx, p.elems...); diff(ctx, elems, p.elems) {
        res = &group{p.valbase, elements{elems}}
    } else {
        res = p
    }
    return
}
func (p *group) traverse(ctx Context) {
    errostack(at(ctx,p.position), 3, "traversing group: %v", p).debug(32)
}
func (p *group) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, y := v.(*group); y {
        if l1, l2 := len(p.elems), len(a.elems); l1 == 0 && l2 == 0 {
           return cmpEqual
        }
        res = compareElems(ctx, p.elems, a.elems)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *group) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    // TODO: for _, elem := range { elem.match(ctx, i) }
    return
}
func (p *group) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (_ *group) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *group) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported: %v", cache).debug(32)
    return
}

func parseGroupValue(ctx Context, g *group) (result Value) {
    if len(g.elems) == 0 { return g } else {
        var word *bareword
        switch kind := g.elems[0].(type) {
        case *bareword: word = kind
        case *group: if len(kind.elems) > 0 {
            var ( name = kind.elems[0]; y bool )
            if word, y = name.(*bareword); !y {
                erro(at(ctx,name), "unsupported name type: %T %v", name, name).debug(1)
            }
        }}
        if word != nil {
            switch word.s {
            case "plain", "json", "yaml", "xml":
                result = makeList(g.elems[1:]...)
            }
        }
        if isNull(result) { result = g }
    }
    return
}

type pair struct { key, val Value }
func (p *pair) kind() Kind { return KindPair }
func (p *pair) Position() Position { return p.key.Position() }
func (p *pair) String() string { return p.srclit(nil) }
func (p *pair) string(ctx Context) string {
    return p.key.string(ctx) + "=" + p.val.string(ctx)
}
func (p *pair) srclit(o Object) string {
    return srclit(o, p.key)+`=`+srclit(o, p.val)
}
func (p *pair) SetValue(v Value) {
    p.val = v
}
func (p *pair) SetKey(k Value) {
    if _, y := k.(*pair); y { /* k = o.key */panic("pair.setkey") }
    p.key = k
}
func (p *pair) true(ctx Context) (t bool) {
    if t = p.key.true(ctx); !t && !isNull(p.val) {
        t = p.val.true(ctx)
    }
    return
}
func (p *pair) int(ctx Context) (i int64, e error) { return p.val.int(ctx) }
func (p *pair) float(ctx Context) (f float64, e error) { return p.val.float(ctx) }
func (p *pair) refs(ctx Context, v Value) bool { return p.key.refs(ctx, v) || p.val.refs(ctx, v) }
func (p *pair) defs(ctx Context, s ...string) []*def {
    return append(p.key.defs(ctx, s...), p.val.defs(ctx, s...)...)
}
func (p *pair) ident(Context) (_ string) { return }
func (p *pair) stamp(Context) (_ []*File, _ error) { return }
func (p *pair) stat(Context) (_ *statinfo) { return }
func (p *pair) match(Context, interface{}) (_ bool, _ interface{}, _ []string) { return }
func (p *pair) patterned(Context) (_ bool) { return }
func (p *pair) delete(Context) (_ []*File, _ error) { return }
func (p *pair) updated(Context) (_ bool) { return }
func (p *pair) updatedDeps(Context, ...Value) (_ []Value) { return }
func (p *pair) expandable(ctx Context) bool {
    return p.key.expandable(ctx) || (_exPairVal(ctx) && p.val.expandable(ctx))
}
func (p *pair) expand(ctx Context) (res Value) {
    var k, v = p.key.expand(ctx), p.val
    if _exPairVal(ctx) { v = v.expand(ctx) }
    if equal(ctx, k, p.key) && equal(ctx, v, p.val) {
        return p
    }

    var ks, vs []Value
    switch t := k.(type) { case disjunction: ks = merge(t.Value) }
    switch t := v.(type) { case disjunction: vs = merge(t.Value) }

    if ks == nil && vs == nil {
        return &pair{k, v}
    }
    if ks == nil { ks = []Value{k} }
    if vs == nil { vs = []Value{v} }

    var vals []Value
    for _, k := range ks {
        for _, v := range vs {
            vals = append(vals, &pair{k, v})
        }
    }
    return ease(ctx, vals)
}
func (p *pair) prefix(ctx Context, val Value) (res Value) {
    var pk = p.key
    switch p = (&pair{nil, p.val}); pk.(type) {
    case *null, *none: p.key = val
    default: p.key = compose(ctx, val, p.key)
    }
    if indeterminate(ctx, p) {
        return condish(ctx, p)
    } else {
        return p
    }
}
func (p *pair) suffix(ctx Context, val Value) (res Value) {
    var pv = p.val
    switch p = (&pair{p.key, nil}); pv.(type) {
    case *null, *none: p.val = val
    default: p.val = compose(ctx, p.val, val)
    }
    if indeterminate(ctx, p) {
        return condish(ctx, p)
    } else {
        return p
    }
}
func (p *pair) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var k, v Value
    k, rest = p.key.stencil(ctx, stems)
    v, rest = p.val.stencil(ctx, rest)

    var (
        knull = isNull(k)
        vnull = isNull(v)
    )
    if (!knull && k != p.key) || (!vnull && v != p.val) {
        if knull { k = p.key   }
        if vnull { v = p.val }
        val = &pair{k, v}
    }
    return
}
func (p *pair) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if x, y := v.(*pair); y {
        if p == x {
            if checkpoints {
                if cr := p.key.cmp(ctx, x.key); cr != cmpEqual {
                    erro(ctx, "%v, %v ⇔ %v", cr, us(p.key), us(x.key)).debug(3)
                }
                if cr := p.val.cmp(ctx, x.val); cr != cmpEqual {
                    erro(ctx, "%v, %v ⇔ %v", cr, us(p.val), us(x.val)).debug(3)
                }
            }
            res = cmpEqual
        } else {
            res = p.key.cmp(ctx, x.key)
            if res == cmpEqual {
                res = p.val.cmp(ctx, x.val)
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *pair) traverse(ctx Context) {
    erro(ctx, "traversing pair '%v' is undefined", p)
    errostack(ctx, -1, "pair is not traversible: %v", p).debug(16)
}
func (_ *pair) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *pair) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}

type skipped struct { Value }
func (s skipped) kind() Kind { return s.Value.kind()|KindSkipped }

type selected struct { Value }
func (s selected) kind() (k Kind) { return s.Value.kind()|KindSelected }
func (s selected) _cmp(ctx Context, v Value) (res cmpres) {
    if x, y := v.(selected); y {
        res = s.Value.cmp(ctx, x.Value)
    } else if x, y := v.(condval); y {
        res = s.Value.cmp(ctx, x.Value)
    } else if l, y := v.(*list); y && l.len() == 1 {
        res = s.Value.cmp(ctx, l.elems[0])
    } else if false {
        res = s.Value.cmp(ctx, v)
    }
    return
}

type expanded struct { Value }
func (s expanded) kind() Kind { return s.Value.kind()|KindExpanded }
func (s expanded) _cmp(ctx Context, v Value) (res cmpres) {
    if x, y := v.(expanded); y {
        res = s.Value.cmp(ctx, x.Value)
    } else if x, y := v.(condval); y {
        res = s.Value.cmp(ctx, x.Value)
    } else if x, y := v.(*list); y && x.len() == 1 {
        res = s.Value.cmp(ctx, x.elems[0])
    } else if false {
        res = s.Value.cmp(ctx, v)
    }
    return
}

type untraversed struct { Value }
func (u untraversed) traverse(ctx Context) {}
func (u untraversed) expand(ctx Context) Value {
    return untraversed{u.Value.expand(ctx)}
}

func unresolved(ctx Context, x Value) bool {
    switch x.(type) {
    case *auto, *builtin, *def:
        return false
    default: return true
    }
}

func exable(ctx Context, a, b Value) bool {
    if b == nil || equal(ctx, a, b) {
        switch a.(type) { case *auto, *project: return true }
        return indeterminate(ctx, a)
    }
    return false
}
func ex(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool) (res Value) {
    defer trace(ctx)

    var ( t, x Value ; a []Value ; entry, e bool )

    if checkpoints { defer func() { ex_check(ctx, p, _x, _a, _o, _l, _cl, res, x, a) } () }
    if false && _exDef1(ctx) { if s := p.String(); true &&
        "$(foo)" == s { defer func() {
            // _exDef1(ctx) _exFinal(ctx) _exEvaluation(ctx) _exDelegate(ctx, _x) _exClosure(ctx, _x)
            noted(ctx, "%v", us(p))
            noted(ctx, "x=%v → %v, e=%v", us(_x), us(x), e)//, x.expandable(ctx)
            noted(ctx, "o=%v, a=%v→%v", us(_o), us(_a), us(a))
            noted(ctx, "res=%v", us(res))
            noted(ctx, "%v", us(ctx)).debug(32)
        }()}}

    x = _x.expand(ctx)

    if _, y := x.(disjunction); y {
        erro(ctx, "TODO: %v", us(x)).debug(1)
        return
    }

    if l, y := x.(*list); y {
        var vb = valbase{p.Position()}
        var vals []Value
        for _, v := range l.elems {
            if t := (delegate{vb, _l, v, _o, _a}); _cl {
                v = ex(ctx, &closure{t}, v, _a, _o, _l, true)
            } else {
                v = ex(ctx, &t, v, _a, _o, _l, false)
            }
            vals = append(vals, v)
        }
        if n := len(vals); 0 == n {
            return makeNull(_x.Position())
        } else if 1 == n {
            return vals[0]
        }
        return disjunction{&list{elements{vals}}}
    }

    switch _l { case LBRACE, STRING, COMPOUND: entry = true } // &{xxx}  &'xxx'  &"xxx", else/ILLEGAL &(xxx)

    // NOTE: `x` must be expandable in final context for x.string() as resolving name.
    if exable(ctx, x, _x) {
        e = false
    } else if _cl {
        if _exClosure(ctx, x) {
            var v Value
            var s string
            switch t := x.(type) {
            case Object: s = t.ident(ctx)
            default:     s = t.string(ctx)
            }
            if s == "" {
                return
            } else if entry {
                v = closureEntry(ctx, s)
            } else {
                v = closureResolve(ctx, s)
            }
            if v != nil { x, e = v, true }
        }
    } else if _exDelegate(ctx, x) {
        if unresolved(ctx, x) {
            var v Value
            if s := x.string(ctx); s == "" {
                if false { erro(ctx, "empty unresolved name: %v", us(x)).debug(3) }
            } else if entry {
                v = resolveEntries(ctx, s)
            } else {
                v = resolve(ctx, s)
            }
            if v != nil { x, e = v, true }
        } else {
            e = true
        }
    }

    var un = func(a []Value) Value {
        if !equal(ctx, _x, x) || diff(ctx, a, _a) {
            if d := (delegate{valbase{p.Position()}, _l, x, _o, a}); _cl {
                return &closure{d}
            } else {
                return &d
            }
        }
        return p
    }

    if !e {
        return un(expand(ctx, _a...))
    } else if t, a = evoke(ctx, x, _o, _a); equal(ctx, x, t) {
        return un(a)
    } else if x, y := t.(expanded); y {
        return x.Value
    } else {
        return t
    }
}
func ex_check(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool, res, x Value, a []Value) {
    if res == nil {
        if !_cl && x == nil {
            erro(ctx, "%v: %v → %v", us(p), us(_x), us(x)).debug(5)
            return
        }

        var v = _x
        if a, y := v.(*auto); y { if d := autoDef(ctx, a.name); d != nil {
            noted(ctx, "%v", us(_x))
            noted(ctx, "%v", us(x))
            noted(ctx, "%v", us(p))
            erro(ctx, "%v", us(ctx)).debug(24)
        }}
        if _cl {
            // TODO: closure checkpoints ...
        } else if _x == nil {
            noted(ctx, "%v: nil", us(p))
            erro(ctx, "%v", us(ctx)).debug(24)
        } else {
            if d, y := _x.(*def); y { if d == nil {
                erro(ctx, "%v", us(ctx)).debug(24)
            } else if d.value != nil {
                noted(ctx, "%v", us(_x))
                noted(ctx, "%v", us(x))
                noted(ctx, "%v", us(p))
                erro(ctx, "%v", us(ctx)).debug(24)
            }}
        }
    } else if false && p != res && equal(ctx, p, res) {
        noted(ctx, "%v: %p != %p", us(res), res, p)
        erro(ctx, "%v", us(ctx)).debug(3)
    }
}

type delegateContext struct { Context ; entry bool }
type delegate struct {
    valbase
    l   token
    x   Value
    o []Value
    a []Value
    // TODO: patsubst Value, aka lhs%=rhs% like in $(var:lhs%=rhs%)
}
func (p *delegate) kind() Kind { return p.valbase.kind()|KindDelegate }
func (p *delegate) String() string { return p.srclit(nil) }
func (p *delegate) string(ctx Context) (s string) {
    if t, y := p.x.(*project); y { return t.name }
    p.exstr(ctx, func(v Value) { s = v.string(ctx) })
    if checkpoints {
        if p.String() == "$/" {
            if s == "" {
                erro(ctx, "%v", us(p))
                erro(ctx, "%v", p)
                erro(ctx, "%v", us(ctx)).debug(3)
            } else if !filepath.IsAbs(s) {
                erro(ctx, "%v", p)
                erro(ctx, "%v", us(p))
                erro(ctx, "→ %v", s)
                erro(ctx, "%v", us(ctx)).debug(3)
            }
        }
        if p.String() == "$." {
            if strings.HasPrefix(s, "./") {
                erro(ctx, "%v", p)
                erro(ctx, "%v", us(p))
                erro(ctx, "→ %v", s)
                erro(ctx, "%v", us(ctx)).debug(3)
            }
        }
    }
    return
}
func (p *delegate) true(ctx Context) (t bool) {
    p.exstr(ctx, func(v Value) { t = v.true(ctx) })
    return
}
func (p *delegate) int(ctx Context) (i int64, e error) {
    p.exstr(ctx, func(v Value) { i, e = v.int(ctx) })
    return
}
func (p *delegate) float(ctx Context) (f float64, e error) {
    p.exstr(ctx, func(v Value) { f, e = v.float(ctx) })
    return
}
func (p *delegate) aone(ctx Context, v Value) (res bool) {
    var q, y = v.(*delegate)
    if y && equal(ctx, p.x, q.x) && len(p.o) == len(q.o) && len(p.a) == len(q.a) {
        for i, o := range p.o { if !equal(ctx, o, q.o[i]) { return true } }
        for i, a := range p.a { if !equal(ctx, a, q.a[i]) { return true } }
    }
    return
}
func (p *delegate) exstr(ctx Context, f func(Value)) {
    var v = p.expand(final{ctx})
    if v == nil || equal(ctx, p, v) || (v.expandable(ctx) && !p.aone(ctx, v)) {
        return
    }

    var nr = !v.refs(ctx, p)
    if checkpoints { p.exstr_check(ctx, v, nr) }
    if nr { f(v) }
}
func (p *delegate) exstr_check(ctx Context, v Value, nr bool) {
    if false && p.String() == "$/" {
        noted(at(ctx,p), "%v → %v", us(p), us(v))
        erro(ctx, "%v", us(ctx)).debug(10)
    }
    if nr {
        var u = v.expandable(ctx)
        if v == p || (false && v.refs(ctx, p.x)) {
            noted(at(ctx,p), "%v → %v (%v)", us(p), us(v), (v==p))
            erro(ctx, "%v", us(ctx)).debug(16)
            return
        }
        if u && v == p {
            noted(at(ctx,p), "%v → %v", us(p), us(v))
            erro(ctx, "%v", us(ctx)).debug(16)
            return
        }
        if p.String() == v.String() {
            if u {
                noted(at(ctx,p), "%v → %v , %v", us(p), us(v), p.cmp(ctx, v))
                erro(ctx, "%v", us(ctx)).debug(16)
            } else {
                noted(at(ctx,p), "%v → %v , %v", us(p), us(v), v.cmp(ctx, p))
                erro(ctx, "%v", us(ctx)).debug(16)
            }
            return
        }
    }
}
func (p *delegate) isValidtoken() (res bool) {
    switch p.l {
    case LPAREN, LBRACE, STRING, COMPOUND, ILLEGAL:
        res = true
    default: // for $. $/ $1 ... &. &/ &1 ... etc.
        res = p.l.isClosureDelegate()
    }
    return
}
func (p *delegate) refs(ctx Context, v Value) (res bool) {
    if p == v || p.x == v || p.x.refs(ctx, v) { return true }
    for _, a := range p.a { if a.refs(ctx, v) { return true } }
    return
}
func (p *delegate) defs(ctx Context, s ...string) (res []*def) {
    if p.x == nil {
        erro(at(ctx,p), "delegation of nil (s=%v)", p, s).debug(1)
        return
    } else if d, y := p.x.(*def); y {
        if y = len(s) == 0; !y {
            for _, a := range s { if y = d.ident(ctx) == a; y { break } }
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
func (p *delegate) traverse(ctx Context) {
    ctx = at(ctx, p.position)
    p.exstr(ctx, func(v Value) { v.traverse(ctx) })
}
func (p *delegate) ident(ctx Context) (name string) {
    const sel = true
    switch x := p.x.(type) {
    case interface{ ident(Context) string }: name = x.ident(ctx)
    case *selection: if sel { name = x.string(ctx) }
    }
    return
}
func (p *delegate) srclit(o Object) string { return p.src(o, "$") }
func (p *delegate) src(o Object, l string) (s string) {
    if s = p.srcrep(o); !(p.l.isClosureDelegate()) { s = l + s }
    return
}
func (p *delegate) srcrep(o Object) (s string) { // source representation
    switch x := p.x.(type) {
    case       *def: s = x.name
    case *selection: s = x.String()
    default: if x != nil { s = x.String() }
    }

    if p.o != nil { // options
        s += "("
        for i, v := range p.o {
            if 0 < i { s += " " }
            s += srclit(o, v)
        }
        s += ")"
    }

    for i, a := range p.a {
        if 0 == i { s += " " } else { s += "," }
        s += srclit(o, a)
    }

    switch p.l {
    case COMPOUND: return `"`+s+`"`
    case   STRING: return `'`+s+`'`
    case   LPAREN: return "("+s+")"
    case   LBRACE: return "{"+s+"}"
    case  ILLEGAL: return "["+s+"]"
    default:
        if p.l.isClosureDelegate() {
            return p.l.String() // $@, $<, ...
        } else {
            return fmt.Sprintf("[%s]!(%v)", s, p.l)
        }
    }
}
func (p *delegate) expandable(ctx Context) (res bool) {
    if false {
        res = true
    } else {
        if res = _exDelegate(ctx, p.x) || p.x.expandable(ctx); !res {
            for _, a := range p.a { if a.expandable(ctx) { return true }}
        }
    }
    return
}
func (p *delegate) expand(ctx Context) (res Value) {
    return ex(at(ctx,p.position), p, p.x, p.a, p.o, p.l, false)
}
func (p *delegate) prefix(ctx Context, v Value) Value { return _suffix(ctx, v, p) }
func (p *delegate) suffix(ctx Context, v Value) Value { return makeBarecomp(p).suffix(ctx, v) }
func (p *delegate) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    if v := p.expand(ctx); v != nil {
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
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }

    if d, y := v.(*delegate); y { // NOTE: delegate not expanded!
        if p == d { return cmpEqual }

        var t cmpres
        if p.x == d.x {
            t = cmpEqual
        } else {
            t = p.x.cmp(ctx, d.x)
        }

        if checkpoints {
            if t != cmpEqual && p.x.String() == d.x.String() {
                erro(at(ctx,p.x), "%v: %v %v", us(p), us(p.x), us(d.x)).debug(3)
            }
            if len(p.a) == len(d.a) { for i, v := range p.a {
                if v.cmp(ctx, d.a[i]) != cmpEqual && v.String() == d.a[i].String() {
                    erro(at(ctx,v), "%v: %v %v", us(p), us(v), us(d.a[i])).debug(3)
                }
            }} else if false {
                erro(at(ctx,p.x), "%v: %v %v", us(p), us(p.x), us(d.x)).debug(3)
            }
        }

        if t == cmpEqual && len(p.a) == len(d.a) {
            for i, v := range p.a {
                if r := v.cmp(ctx, d.a[i]); r != cmpEqual { return r }
            }
        }

        res = t
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    } else if true {
        return // Done here!
    } else if d, y := p.x.(*def); y && len(p.a) == 0 && d.value != nil {
        res = d.value.cmp(ctx, v)
    }
    return
}
func (p *delegate) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
    }
}
func (p *delegate) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    p.exstr(ctx, func(v Value) { res = cache.slot(ctx, v, bits) })
    return
}
func (p *delegate) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    p.exstr(ctx, func(v Value) { res = v.collect(ctx, cache, bits) })
    return
}

type closure struct { delegate }
func (p *closure) kind() Kind { return p.valbase.kind()|KindClosure }
func (p *closure) String() (s string) { return p.srclit(nil) }
func (p *closure) srclit(o Object) string { return p.src(o, "&") }
func (p *closure) string(ctx Context) (s string) {
    p.exstr(ctx, func(v Value) { s = v.string(ctx) })
    return
}
func (p *closure) true(ctx Context) (t bool) {
    p.exstr(ctx, func(v Value) { t = v.true(ctx) })
    return
}
func (p *closure) prefix(ctx Context, v Value) Value { return _suffix(ctx, v, p) }
func (p *closure) suffix(ctx Context, v Value) Value { return makeBarecomp(p).suffix(ctx, v) }
func (p *closure) exstr(ctx Context, f func(Value)) {
    var v = p.expand(final{ctx})
    if v == nil || equal(ctx, p, v) || (v.expandable(ctx) && !p.aone(ctx, v)) {
        return
    } else if !v.refs(ctx, p) {
        f(v)
    }
}
func (p *closure) expand(ctx Context) (res Value) {
    return ex(at(ctx, p.position), p, p.x, p.a, p.o, p.l, true)
}
func (p *closure) expandable(ctx Context) (res bool) {
    if false {
        res = true
    } else {
        if res = _exClosure(ctx, p.x) || p.x.expandable(ctx); !res {
            for _, a := range p.a { if a.expandable(ctx) { return true }}
        }
    }
    return
}
func (p *closure) match(ctx Context, i interface{}) (full bool, s interface{}, stems []string) {
    if v := p.expand(ctx); v != p {
        return v.match(ctx, i)
    } else if false {
        errostack(at(ctx,p), 3, "unexpand closure: %v", v).debug(16)
    }
    return
}
func (p *closure) refs(ctx Context, v Value) (res bool) {
    if p.x == nil {
        erro(at(ctx,p), "closure of nil: %v (%v)", us(p), us(v)).debug(10)
        return
    }
    if p == v || p.x == v || p.x.refs(ctx, v) { return true }
    for _, a := range p.a { if a.refs(ctx, v) { return true } }
    return
}
func (p *closure) traverse(ctx Context) {
    ctx = at(ctx, p.position)
    p.exstr(ctx, func(v Value) { v.traverse(ctx) })
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
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }

    if a, y := v.(*closure); y {
        if p == a { return cmpEqual } else
        if res = p.x.cmp(ctx, a.x); res == cmpEqual && len(p.a) == len(a.a) {
            for i, t := range p.a {
                if res = t.cmp(ctx, a.a[i]); res != cmpEqual { return }
            }
            return
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *closure) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
    }
}
func (p *closure) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    p.exstr(ctx, func(v Value) {
        if v == nil || v == p || v.expandable(ctx) {
            errostack(ctx, 10, "cache unsupported (bits=%08b): %v", bits, v).debug(32)
        } else {
            res = cache.slot(ctx, v, bits)
        }
    })
    return
}
func (p *closure) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    p.exstr(ctx, func(v Value) {
        if v == nil || v == p || v.expandable(ctx) {
            errostack(ctx, 10, "cache unsupported (bits=%08b): %v", bits, v).debug(32)
        } else {
            res = v.collect(ctx, cache, bits)
        }
    })
    return
}

type selection struct {
    valbase
    t token
    o Value // Object or selection
    s Value
}
func (p *selection) srclit(o Object) (s string) {
    return srclit(o, p.o) + p.t.String() + srclit(o, p.s)
}
func (p *selection) String() (s string) { return p.srclit(nil) }
func (p *selection) string(ctx Context) (s string) {
    p.ex(ctx, func(v Value) { s = v.string(ctx) })
    return
}
func (p *selection) true(ctx Context) (t bool) {
    p.ex(ctx, func(v Value) { t = v.true(ctx) })
    return
}
func (p *selection) int(ctx Context) (i int64, e error) {
    p.ex(ctx, func(v Value) {
        if s := v.string(ctx); s != "" {
            i, e = strconv.ParseInt(s, 10, 64)
        }
    })
    return
}
func (p *selection) float(ctx Context) (f float64, e error) {
    p.ex(ctx, func(v Value) {
        if s := v.string(ctx); s != "" {
            f, e = strconv.ParseFloat(s, 64)
        }
    })
    return
}
func (p *selection) refs(ctx Context, v Value) bool {
    return p.o.refs(ctx, v) || p.s.refs(ctx, v)
}
func (p *selection) defs(ctx Context, s ...string) []*def {
    return append(p.o.defs(ctx, s...), p.s.defs(ctx, s...)...)
}
func (p *selection) expandable(ctx Context) (res bool) {
    return p.o.expandable(ctx) || p.s.expandable(ctx)
}
func (p *selection) expand(ctx Context) (res Value) {
    var o = p.o.expand(ctx)
    if checkpoints {
        if x, y := o.(condval); y {
            erro(ctx, "wrong: %v", us(x)).debug(5)
        }
    }
    if p.t.isSelectProg() {
        if x, y := o.(*project); y && x != nil {
            res = selected{ x.resolveEntries(ctx, p.s, false) }
        }
        return
    }
    if x, y := o.(*project); false && y && x != nil {
        ctx = closureWith(ctx, x.scope)
    }

    var s = p.s.expand(ctx)
    if checkpoints {
        if x, y := s.(condval); y {
            erro(ctx, "wrong: %v", us(x)).debug(5)
        }
    }
    if i, y := o.(interface{ get(Context,string) Value }); y && isFinalValue(ctx, s) {
        if v := i.get(ctx, s.string(ctx)); v != nil { return selected{ v } }
    }

    if !equal(ctx, o, p.o) || !equal(ctx, s, p.s) {
        if cond(p.o) && !cond(o) { o = condish(ctx, o) }
        if cond(p.s) && !cond(s) { s = condish(ctx, s) }
        return &selection{p.valbase, p.t, o, s}
    } else {
        return p
    }
}
func (p *selection) ex(ctx Context, f func(Value)) {
    if v := p.expand(ctx); v != nil && !equal(ctx, v, p) { f(v) }
}
func (p *selection) traverse(ctx Context) {
    ctx = at(ctx, p.position)

    if val := p.expand(ctx); isTrivial(val) {
        warn(ctx, "selected trivial value '%v' (%v, %v) ", p, us(p.o), us(p.s)).debug(10)
    } else {
        val.updated(ctx) // NOTE: ensure that updated flag is correct (see rule.updated)
        val.traverse(ctx)
    }
}
func (p *selection) updated(ctx Context) (res bool) { // NOTE: this seems not affecting the result
    if val := p.expand(ctx); isTrivial(val) {
        noted(ctx, "selected value '%v' is trivial", p).debug(1)
    } else {
        res = val.updated(ctx)
    }
    return res
}
func (p *selection) updatedDeps(ctx Context, v ...Value) (res []Value) { // NOTE: this seems not affecting the result
    if val := p.expand(ctx); isTrivial(val) {
        noted(ctx, "selected value '%v' is trivial", p).debug(1)
    } else {
        res = val.updatedDeps(ctx, v...)
    }
    return res
}
func (p *selection) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if x, y := v.(*selection); y {
        if p.t == x.t {
            if  res = p.o.cmp(ctx, x.o); res == cmpEqual {
                res = p.s.cmp(ctx, x.s)
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            if x, y := v.(*selection); y {
                noted(ctx, "%v %v, %v", us(p.o), us(x.o), p.o.cmp(ctx, x.o))
                noted(ctx, "%v %v, %v", us(p.s), us(x.s), p.s.cmp(ctx, x.s))
            }
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
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
func (_ *selection) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *selection) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported").debug(32)
    return
}

// percpat represents percent pattern expressions (e.g. '%.o')
type percpat struct {
    valbase // TODO: supporting multiple %: foo%bar%xxx
    Prefix Value
    Suffix Value
}
func (p *percpat) String() (s string) { return p.srclit(nil) }
func (p *percpat) srclit(o Object) (s string) {
    if !isNull(p.Prefix) { s += srclit(o, p.Prefix) }
    s += `%`
    if !isNull(p.Suffix) { s += srclit(o, p.Suffix) }
    return
}
func (p *percpat) string(ctx Context) (s string) {
    if p.Prefix != nil { s += p.Prefix.string(ctx) }
    s += "%"
    if p.Suffix != nil { s += p.Suffix.string(ctx) }
    return
}
func (p *percpat) refs(ctx Context, v Value) bool {
    return p.Prefix.refs(ctx, v) || p.Suffix.refs(ctx, v)
}
func (p *percpat) defs(ctx Context, s ...string) []*def {
    return append(p.Prefix.defs(ctx, s...), p.Suffix.defs(ctx, s...)...)
}
func (p *percpat) expandable(ctx Context) bool {
    return p.Prefix.expandable(ctx) || p.Suffix.expandable(ctx)
}
func (p *percpat) expand(ctx Context) (res Value) {
    var (
        prefix Value
        suffix Value
    )
    if p.Prefix != nil { prefix = p.Prefix.expand(ctx) }
    if p.Suffix != nil { suffix = p.Suffix.expand(ctx) }
    if prefix != p.Prefix || suffix != p.Suffix {
        res = &percpat{ p.valbase, prefix, suffix }
    } else {
        res = p
    }
    return
}
func (p *percpat) patterned(ctx Context) bool { return true }
func (p *percpat) match1(ctx Context, rep string) (full bool, result string, stems []string) {
    var prefix string
    if !isTrivial(p.Prefix) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if prefix = p.Prefix.string(ctx); strings.HasPrefix(rep, prefix) {
            result = prefix
        } else {
            return
        }
    }

    var a, b = len(prefix), len(rep)
    if isTrivial(p.Suffix) {
        if a < b { stems, result, full = append(stems, rep[a:]), rep, true }
    } else if pp, ok := p.Suffix.(*percpat); a < b && ok {
        // FIXME: separate paths???  *) use % to match single path sep; *) use %% to match full path
        // fooxxbaryybaz -> foo%bar%baz => (foo xx bar yy baz) [xx yy]
        // fooxxx -> foo%% => (foo xxx) [xxx]
        // fooxxxbar -> foo%%bar => (foo xxx bar) [xxx]
        for ok {
            if isTrivial(pp.Prefix) {
                // does nothing
            } else if s := pp.Prefix.string(ctx); s != "" {
                if n := strings.Index(rep[a:], s); n < 0 {
                    break
                } else {
                    var v = rep[a:a+n]
                    stems = append(stems, v)
                    result += v + s
                    a += n + len(s)
                }
            }
            var pp2 *percpat
            if isTrivial(pp.Suffix) {
                var s = rep[a:] // let %% matches everything else
                full, stems = true, append(stems, s)
                result += s
                break
            } else if pp2, ok = pp.Suffix.(*percpat); ok {
                pp = pp2
                continue
            } else if s := pp.Suffix.string(ctx); s != "" && strings.HasSuffix(rep[a:], s) {
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
            warn(at(ctx,p.Suffix), "mixing % pattern might have performance impact: %v", p).debug(1)
        }
        for n := b-1; a < n; n -= 1 {
            if f, r, ss := p.Suffix.match(ctx, rep[n:]); f && r != nil {
                var s, _ = r.(string)
                stems = append(append(stems, rep[a:n]), ss...)
                result += s // rep[a:]
                full = f
                break
            }
        }
    } else if a <= b {
        if s := p.Suffix.string(ctx); strings.HasSuffix(rep[a:], s) {
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
func (p *percpat) match(ctx Context, i interface{}) (full bool, result interface{}, stems []string) {
    switch t := i.(type) {
    case string: return p.match1(ctx, t)
    case *filestub:
        if full, result, stems = p.match1(ctx, t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(ctx, filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
        }
    case *File:
        if full, result, stems = p.match1(ctx, t.ident(ctx)); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(ctx, t.fullname()) // NOTE: matching the fullname form
        }
    case Value:
        return p.match1(ctx, t.string(ctx))
    default:
        unreachable(fmt.Sprintf("perc.match: %T %v", i, i))
    }
    return
}
func (p *percpat) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var vals []Value
    if isTrivial(p.Prefix) {
        // does nothing
    } else if p.Prefix.patterned(ctx) {
        erro(at(ctx,p.Prefix), "patterned prefix: %T %v", p.Prefix, p.Prefix).debug(1)
        return
    } else {
        vals = append(vals, p.Prefix)
    }

    if len(stems) > 0 {
        if s := stems[0]; s != "" { vals = append(vals, makeBareword(p.position, s)) }
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
    } else if pp, ok := p.Suffix.(*percpat); ok && isTrivial(pp.Prefix) {
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
    } else if suf, res := suffix.stencil(ctx, rest); !isNull(suf) && suf != suffix {
        // NOTE: patterns like '...%xxx%...' use multiple stems.
        vals, rest = append(vals, suf), res
    } else {
        vals, rest = append(vals, suffix), res
    }

DoneVals:
    if n := len(vals); n == 1 {
        val = vals[0]
    } else if n > 1 {
        val = makeBarecomp(vals...)
    } else {
        val = p
    }
    return
}
func (p *percpat) traverse(ctx Context) { ctx.traverse(ctx, p) }
func (p *percpat) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if a, ok := v.(*percpat); ok {
        if p.Prefix.cmp(ctx, a.Prefix) == cmpEqual {
            if p.Suffix.cmp(ctx, a.Suffix) == cmpEqual {
                res = cmpEqual
            }
        }
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
        }
    }
    return
}
func (p *percpat) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    if false { ctx = at(ctx, p.position) }

    var fix string
    switch t := p.Prefix.(type) {
    case *barecomp: fix = t.string(ctx)
    case *bareword: fix = t.s
    case *null,nil: fix = ""
    default:
        errostack(at(ctx, p.Prefix), 3, "unsupported prefix: %T %v", t, t).debug(16)
        return
    }

    if cache = cache.fix(ctx, fix, bits); cache == nil { return }
    if cache = cache.fix(ctx, "%", bits); cache == nil { return }

    switch t := p.Suffix.(type) {
    case *barecomp: fix = t.string(ctx)
    case *bareword: fix = t.s
    case *null,nil: fix = ""
    case *percpat: return t.cache(ctx, cache, bits)
    default:
        errostack(at(ctx, p.Suffix), 3, "unsupported suffix: %T %v", t, t).debug(16)
        return
    }
    return cache.fix(ctx, fix, bits)
}
func (p *percpat) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    var pre, suf string
    switch t := p.Prefix.(type) {
    case *barecomp: pre = t.string(ctx)
    case *bareword: pre = t.s
    case *none,nil: pre = ""
    default:
        errostack(at(ctx, p.Prefix), 3, "unsupported prefix: %T %v", t, t).debug(16)
        return
    }

    switch t := p.Suffix.(type) {
    case *barecomp: suf = t.string(ctx)
    case *bareword: suf = t.s
    case *none,nil: suf = ""
    case *percpat:
        // TODO: use map, somehow
        for k, c := range cache.fast {
            if !strings.HasPrefix(k, pre) { continue } else { k = k[len(pre):] }
            if full, _, _ := t.match(ctx, k); full { res = append(res, c) }
        }
        return
    default:
        errostack(at(ctx, p.Suffix), 3, "unsupported suffix: %T %v", t, t).debug(16)
        return
    }

    // TODO: use map, somehow
    for k, c := range cache.fast {
        if strings.HasPrefix(k, pre) && strings.HasSuffix(k, suf) {
            res = append(res, c)
        }
    }
    return
}

// Check for patterns like foo%%bar foo**bar
func multia(ctx Context, p Value) (result bool, prefix, suffix Value) {
    if p1, y := p.(*percpat); y {
        if p2, y := p1.Suffix.(*percpat); y {
            prefix = p1.Prefix
            suffix = p2.Suffix
            result = true
        }
    } else if g, y := p.(*globpat); y && len(g.elems) > 0 {
        var glob, n = false, -1
        for i, comp := range g.elems { if m, y := comp.(*globmeta); y {
            if m.token == DAST && n == -1 { t := g.elems[:i]
                if n = i; n > 0 { if glob {
                    suffix = makeGlobPat(ctx, t...)
                } else {
                    prefix = makeBarecomp(t...)
                }}
                break
            } else {
                glob = true
            }
        }}
        if result = n > -1; result && n < len(g.elems) {
            t, glob := g.elems[n+1:], false
            for _, comp := range t {
                if _, y := comp.(*globmeta); y { glob = true ; break }
            }
            if glob {
                suffix = makeGlobPat(ctx, t...)
            } else if len(t) > 1 {
                suffix = makeBarecomp(t...)
            } else if len(t) > 0 {
                suffix = t[0]
            }
        }
        if false && n != -1 { noted(at(ctx,p), "%v %v %v ; %v %v %v",
            p, g.elems[:n], g.elems[n+1:], result, prefix, suffix).debug(10) }
    }
    return
}

func correctPathPunForMatch(seg Value) Value {
    if x, y := seg.(*barecomp); y {
        for _, elem := range x.elems {
            if _, y := elem.(*path); y { seg = nil; break }
        }
    }
    return seg
}

type compositePattern struct { Value ; constraints []Value }
func (p compositePattern) String() (s string) {
    s += "[" + p.Value.String() + ", "
    for i, v := range p.constraints {
        if i > 0 { s += " " } ; s += v.String()
    }
    s += "]"
    return
}
// func (p compositePattern) string(ctx Context) (s string) {
//     s += "["
//     for i, v := range p.vals { if i > 0 { s += " " } ; s += v.string(ctx) }
//     s += "]"
//     return
// }
func (p compositePattern) match(ctx Context, i interface{}) (full bool, result interface{}, stems []string) {
    if full, result, stems = p.Value.match(ctx, i); full {
        for _, con := range p.constraints {
            if a, b, c := con.match(ctx, i); !a { return a, b, c }
        }
    }
    return
}
// func (p compositePattern) expand(ctx Context) (res Value) { return p }
// func (p compositePattern) refs(ctx Context, v Value) (res bool) {
//     for _, val := range p.vals { if res = val.refs(ctx, v); res { break } }
//     return
// }
// func (p compositePattern) defs(ctx Context, s ...string) (res []*def) {
//     for _, val := range p.vals { res = append(res, val.defs(ctx, s...)...) }
//     return
// }
// func (p compositePattern) patterned(ctx Context) (res bool) {
//     for _, val := range p.vals { if res = val.patterned(ctx); res { break } }
//     return
// }
// func (p compositePattern) stencil(ctx Context, stems []string) (val Value, rest []string) {
//     errostack(ctx, 5, "stencil unsupported").debug(32)
//     return
// }
// func (p compositePattern) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
//     errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
//     return
// }
// func (p compositePattern) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
//     errostack(ctx, 5, "cache unsupported").debug(32)
//     return
// }

// globpat represents glob pattern expressions (e.g. '*.o', '[a-z].o', 'a?a.o')
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
type globpat struct { elements }
func (_ *globpat) kind() Kind { return KindGlobpat }
func (p *globpat) String() string { return p.srclit(nil) }
func (p *globpat) srclit(o Object) (s string) {
    var explicit bool
    for _, comp := range p.elems {
        s += srclit(o, comp)
        if x, y := comp.(*globmeta); y && x.token == QUE {
            explicit = true
        }
    }
    if explicit { s = "{=glob "+s+"}" }
    return
}
func (p *globpat) string(ctx Context) (s string) {
    for _, comp := range p.elems { s += comp.string(ctx) }
    return
}
func (p *globpat) true(ctx Context) bool { return p.elements.true(ctx) }
func (p *globpat) float(ctx Context) (_ float64, _ error) { return }
func (p *globpat) int(ctx Context) (_ int64, _ error) { return }
func (p *globpat) refs(ctx Context, v Value) (res bool) {
    for _, comp := range p.elems {
        if res = comp.refs(ctx, v); res { break }
    }
    return
}
func (p *globpat) defs(ctx Context, s ...string) (res []*def) {
    for _, comp := range p.elems {
        res = append(res, comp.defs(ctx, s...)...)
    }
    return
}
func (p *globpat) expandable(ctx Context) (res bool) {
    for _, comp := range p.elems {
        if res = comp.expandable(ctx); res { break }
    }
    return
}
func (p *globpat) expand(ctx Context) (res Value) {
    if elems := expand(ctx, p.elems...); diff(ctx, elems, p.elems) {
        res = &globpat{elements{elems}}
    } else {
        res = p
    }
    return
}
func (p *globpat) patterned(ctx Context) bool { return true }
func (p *globpat) match(ctx Context, i interface{}) (full bool, result interface{}, stems []string) {
    var s string
    switch t := i.(type) {
    case *filestub: s = t.name
    case *File:     s = t.ident(ctx)
    case Value:     s = t.string(ctx)
    case string:    s = t
    case []string:
        if n := len(t); n == 1 {
            s = t[0]
        } else if true && n > 1 {
            s = filepath.Join(t...) // TODO: optimization: avoid joining
        } else {
            return
        }
    default:
        errostack(at(ctx,p), 3, "%v : unsupported glob match: %T %v", p, i, i).debug(16)
        return
    }

    var err error
    var pattern = p.string(ctx)
    if full, stems, err = globMatch(ctx, pattern, s); full { result = s }
    if false && p.String() == "*.def.in" { info(ctx, "%v %v ; %v %v %v", p, s, full, stems, err).debug(1) }
    if err != nil { errostack(at(ctx,p), 3, "%v: glob match: %v", p, err).debug(16) }
    return
}
func (p *globpat) stencil(ctx Context, stems []string) (val Value, rest []string) {
    erro(ctx, "Unimplemented globpat stencil %v (stems=%v)", p, stems)
    return
}
func (p *globpat) traverse(ctx Context) { ctx.traverse(ctx, p) }
func (p *globpat) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }

    if a, y := v.(*globpat); y {
        if len(p.elems) == len(a.elems) {
            for i, c := range p.elems {
                if c.cmp(ctx, a.elems[i]) != cmpEqual {
                    return
                }
            }
            return cmpEqual
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *globpat) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
    }
}
func (p *globpat) filecache(ctx Context, _c *filecache) (res *filecache, done bool) {
    var c = _c
    if c, done = c.hit(ctx, reflect.TypeOf(p).Elem()); c != nil {
        if false {
            for _, elem := range p.elems {
                if indeterminate(ctx, elem) {
                    erro(at(ctx,elem), "filecache %v : indeterminate element : %v", p, us(elem)).debug(3)
                    return
                }
                if c, done = c.hit(ctx, elem); c == nil {
                    if cacheMapping(ctx) {
                        erro(at(ctx,elem), "no filecache for %v : %v", elem, c).debug(3)
                    }
                    return
                } else {
                    if res = c ; done { return }
                }
            }
            return
        }
        if c, done = c.hit(ctx, p.String()); c == nil {
            if cacheMapping(ctx) {
                erro(at(ctx,p), "no filecache for %v : %v", p, c).debug(3)
            }
            return
        } else {
            res = c
        }
    }
    return
}
func (p *globpat) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    var fix string
    for _, comp := range p.elems {
        switch t := comp.(type) {
        case *barecomp, *bareword: fix = t.string(ctx)
            if cache = cache.fix(ctx, fix, bits); cache == nil { return }

        case *globmeta, *globrange:
            if cache = t.cache(ctx, cache, bits); cache == nil { return }

        default:
            errostack(at(ctx, comp), 3, "glob: unsupported component: %T %v", t, t).debug(16)
            return
        }
    }
    return cache
}
func (p *globpat) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    if p.elems == nil { return }

    var m sync.Mutex
    var g sync.WaitGroup
    var collect func(*valcache, int)
    var iterate = func(_ string, c *valcache, depth int) {
        const a = false
        if c._key == nil {
            if a { collect(c, depth+1) }
        } else if y, _, _ := p.match(ctx, c._key); y {
            m.Lock()
            res = append(res, c)
            m.Unlock()
        }
        if !a { collect(c, depth+1) }
        g.Done()
    }

    collect = func(cache *valcache, depth int) {
        for k, c := range cache.fast { g.Add(1) ; go iterate(k, c, depth) }
        for k, c := range cache._fix { g.Add(1) ; go iterate(k, c, depth) }
    }

    collect(cache, 0) ; g.Wait()
    return
}

type regexpat struct { valbase ; *regexp.Regexp }
func (p *regexpat) String() string { return "{=regex "+p.Regexp.String()+"}" }
func (p *regexpat) string(ctx Context) (s string) { return p.Regexp.String() }
func (p *regexpat) patterned(ctx Context) bool { return true }
func (p *regexpat) match(ctx Context, i interface{}) (full bool, result interface{}, stems []string) {
    if p.Regexp != nil {
        var str string
        switch t := i.(type) {
        case *filestub: str = t.name
        case     *File: str = t.ident(ctx)
        case     Value: str = t.string(ctx)
        case    string: str = t
        case  []string: if len(t) == 1 { str = t[0] } else { return }
        default:
            errostack(at(ctx,p), 3, "%T %v :matching unsupported value: %T %v", p, p, i, i).debug(16)
            return
        }

        if sms := p.Regexp.FindStringSubmatch(str); sms != nil && sms[0] == str {
            full, result, stems = true, sms[0], sms[1:]
        }
    }
    return
}
func (p *regexpat) stencil(ctx Context, stems []string) (val Value, rest []string) {
    if p.Regexp != nil {
        erro(ctx, "regexp stencil unsupported: %v %v", p, stems)
    } else {
        rest = stems
    }
    return
}
func (p *regexpat) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints { defer trace(ctx) }
    if checkpoints { defer func() { p.cmp_check(ctx, v, res) } () }

    if a, ok := v.(*regexpat); ok {
        if a != nil {
            if s1, s2 := p.String(), a.String(); s1 == s2 {
                res = cmpEqual
            } else if s1 < s2 {
                res = cmpSmaller
            } else /*if s1 > s2*/ {
                res = cmpGreater
            }
        }
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *regexpat) cmp_check(ctx Context, v Value, res cmpres) {
    if res != cmpEqual && p.String() == v.String() {
        erro(ctx, "%v, %v ⇔ %v", res, us(p), us(v)).debug(5)
    }
}
func (p *regexpat) traverse(ctx Context) { ctx.traverse(ctx, p) }
func (p *regexpat) expand(Context) Value { return p }
func (p *regexpat) filecache(ctx Context, _c *filecache) (res *filecache, done bool) {
    var c = _c
    if c, done = c.hit(ctx, reflect.TypeOf(p).Elem()); c != nil {
        if c, done = res.hit(ctx, p.Regexp.String()); c == nil {
            if cacheMapping(ctx) {
                erro(at(ctx,p), "no filecache for %v : %v", p.Regexp, c).debug(3)
            }
            return
        } else {
            res = c
        }
    }
    return
}
func (_ *regexpat) cache(ctx Context, cache *valcache, bits int) (res *valcache) {
    errostack(ctx, 5, "cache unsupported (bits=%08b)", bits).debug(32)
    return
}
func (_ *regexpat) collect(ctx Context, cache *valcache, bits int) (res []*valcache) {
    errostack(ctx, 5, "cache unsupported").debug(32)
    return
}

type positioner interface {
    Position() Position
}

type Namer interface {
    Name() string
}

type Scoper interface {
    Scope() *Scope
}

type namescoper struct {
    name string
    scope *Scope
}
func (ns *namescoper) Scope() *Scope { return ns.scope }

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

func baremerge(args ...Value) (elems []Value) {
    for _, arg := range args {
        if l, o := arg.(*barecomp); o && l != nil {
            elems = append(elems, baremerge(l.elems...)...)
        } else if l, o := arg.(*list); o && len(l.elems) == 1 {
            elems = append(elems, baremerge(l.elems...)...)
        } else if d, o := arg.(*def); o {
            if d.value == nil {
                elems = append(elems, makeNull(d.position))
            } else {
                elems = append(elems, d.value)
            }
        } else {
            elems = append(elems, arg)
        }
    }
    return
}

// Merge lists recursively into a single list. Previously called Join.
// FIXME: merge all unexpanded.value may cause deadloop.
func merge(args ...Value) (elems []Value) {
    for _, a := range args {
        if a == nil {
            // continue ...
        } else if l := ulist(a); l != nil {
            elems = append(elems, merge(l.elems...)...)
        } else {
            elems = append(elems, a)
        }
    }
    return
}

func xmerge(ctx Context, values ...Value) (res []Value) {
    return merge(expand(ctx, values...)...)
}

func copyvals(vals []Value) (res []Value) {
    if n := len(vals); 0 < n {
        if false {
            res = append([]Value{}, vals...)
        } else {
            res = make([]Value, n)
            copy(res, vals)
        }
    }
    return
}

func detachBarecompList(values ...Value) (res []Value) {
    for _, v := range values {
        if x, y := v.(*barecomp); y {
            var l int
        xelems:
            for i, e := range x.elems {
                if t, y := e.(*list); y {
                    l += 1
                    if n := t.len(); 0 == n {
                        a := append(x.elems[:i], x.elems[i+1:]...)
                        res = append(res, &barecomp{elements{a}})
                    } else if 1 == n {
                        a := append(append(x.elems[:i], t.elems[0]), x.elems[i+1:]...)
                        res = append(res, &barecomp{elements{a}})
                    } else if 2 == n {
                        a := append(x.elems[:i], t.elems[0])
                        b := append(t.elems[1:], x.elems[i+1:]...)
                        res = append(res, &barecomp{elements{a}}, &barecomp{elements{b}})
                    } else {
                        a := append(x.elems[:i], t.elems[0])
                        b := detachBarecompList(t.elems[1:t.len()-1]...)
                        c := append(t.elems[t.len()-1:], x.elems[i+1:]...)
                        res = append(append(res, &barecomp{elements{a}}), b...)
                        x = &barecomp{elements{c}}
                        goto xelems
                    }
                }
            }
            if l == 0 { res = append(res, v) }
        } else {
            res = append(res, v)
        }
    }
    return
}

func trueVal(ctx Context, v Value, i bool) (res bool) {
    if res = i; v != nil { res = v.true(ctx) }
    return
}

func intVal(ctx Context, v Value, i int) (res int) {
    if res = i; v != nil {
        if t, e := v.int(ctx); e == nil { res = int(t) }
    }
    return
}

func int64Val(ctx Context, v Value, i int64) (res int64) {
    if res = i; v != nil {
        if i, e := v.int(ctx); e == nil { res = i }
    }
    return
}

func uintVal(ctx Context, v Value, i uint32) (res uint32) {
    if res = i; v != nil {
        if t, e := v.int(ctx); e == nil {
            res = uint32(t)
        }
    }
    return
}

func filePerm(ctx Context, v Value, i uint32) (res os.FileMode) {
    res = os.FileMode(uintVal(ctx, v, i)) & os.ModePerm
    if res == 0 { res = os.FileMode(0640) }
    return
}

func expandable(ctx Context, values ...Value) bool {
    for _, v := range values { if v.expandable(ctx) { return true } }
    return false
}

func expand(ctx Context, values ...Value) (elems []Value) {
    if checkpoints { defer trace(ctx) }
    for _, elem := range values {
        if elem != nil {
            var v = elem.expand(ctx)
            if v != nil {
                elems = append(elems, v)
                if checkpoints {
                    a := elem.cmp(ctx, v) // equal(ctx, elem, v)
                    b := v.cmp(ctx, elem) // equal(ctx, v, elem)
                    if a != b {
                        erro(ctx, "%v → %v : equal→%v,%v", us(elem), us(v), a, b).debug(1)
                    }
                }
            }
        }
    }
    return
}

func expandPathElems(ctx Context, elems ...Value) (res []Value) {
    for _, elem := range expand(ctx, elems...) {
        if x, y := elem.(*path); y {
            res = append(res, expandPathElems(ctx, x.elems...)...)
        } else {
            res = append(res, elem)
        }
    }
    return
}

func unique(ctx Context, values ...Value) (elems []Value) {
    var m = len(values)
outer:
    for i, a := range values {
        for n := i+1; n < m; n += 1 {
            if equal(ctx, a, values[n]) { continue outer }
        }
        elems = append(elems, a)
    }
    return
}
func reverse_unique(ctx Context, values ...Value) (elems []Value) {
    var m = len(values) - 1
outer:
    for i, a := range values {
        for n := m; i < n; n -= 1 {
            if equal(ctx, a, values[n]) { continue outer }
        }
        elems = append(elems, a)
    }
    return
}

func splitPathStr(ctx Context, str string) (segments []Value) {
    var pos = ctx.Position()
    var a = strings.Split(str, pathSep)
    for i, s := range a {
        // TODO: calculate position for each segment
        var v Value
        if i == 0 {
            switch s {
            case ""  : v = makePathPun(pos, PROOT)
            case "~" : v = makePathPun(pos, TILDE)
            case "." : v = makePathPun(pos, DOT)
            case "..": v = makePathPun(pos, DOTDOT)
            default  : v = makeBareword(pos, s)
            }
        } else if s == "" {
            if i+1 == len(a) {
                v = makePathPun(pos, PTAIL)
            } else if false {
                v = makePathPun(pos, PCON)
            } else {
                warn(at(ctx, pos), "%s: %v[%d]: empty path seg", str, a, i).debug(1)
                continue
            }
        } else {
            v = makeBareword(pos, s)
        }
        segments = append(segments, v)
    }
    return
}

func refs(ctx Context, a Value, v Value) bool { return a == v || a.refs(ctx, v) }
func ease(ctx Context, iv interface{}) (res Value) {
    defer trace(ctx)

    var elems []Value
    switch t := iv.(type) {
    case nil: return
    case    Value: elems = append(elems, merge(t)...)
    case  []Value: elems = append(elems, merge(t...)...)
    case     bare: elems = append(elems, makeBareword(ctx.Position(), string(t)))
    case     bool: elems = append(elems, makeBoolean(ctx.Position(), t))
    case    int  : elems = append(elems, makeDecimal(ctx.Position(), int64(t)))
    case    int16: elems = append(elems, makeDecimal(ctx.Position(), int64(t)))
    case    int32: elems = append(elems, makeDecimal(ctx.Position(), int64(t)))
    case    int64: elems = append(elems, makeDecimal(ctx.Position(),       t ))
    case   uint  : elems = append(elems, makeDecimal(ctx.Position(), int64(t)))
    case   uint16: elems = append(elems, makeDecimal(ctx.Position(), int64(t)))
    case   uint32: elems = append(elems, makeDecimal(ctx.Position(), int64(t)))
    case   uint64: elems = append(elems, makeDecimal(ctx.Position(), int64(t)))
    case  float32: elems = append(elems, makeFloat(ctx.Position(), float64(t)))
    case  float64: elems = append(elems, makeFloat(ctx.Position(),         t))
    case   string: elems = append(elems, makeStrlit(ctx.Position(), t))
    case   []bare: for _, s := range t { elems = append(elems, makeBareword(ctx.Position(), string(s))) }
    case []string: for _, s := range t { elems = append(elems, makeStrlit(ctx.Position(), s)) }
    default: erro(ctx, "unsupported result: %v", tv(t)).debug(3) ; return
    }
    if n := len(elems); 1 == n {
        return elems[0]
    } else if 1 < n {
        return makeList(elems...)
    } else {
        return makeNull(ctx.Position())
    }
}

func ia(a ...interface{}) []interface{} { return a }
func va(ctx Context, i interface{}) (v Value) {
    switch t := i.(type) {
    case   Value: v = t
    case []Value: v = makeList(t...)
    case  int:    v = makeDecimal(ctx.Position(), int64(t))
    case  int16:  v = makeDecimal(ctx.Position(), int64(t))
    case  int32:  v = makeDecimal(ctx.Position(), int64(t))
    case  int64:  v = makeDecimal(ctx.Position(), int64(t))
    case uint:    v = makeDecimal(ctx.Position(), int64(t))
    case uint16:  v = makeDecimal(ctx.Position(), int64(t))
    case uint32:  v = makeDecimal(ctx.Position(), int64(t))
    case uint64:  v = makeDecimal(ctx.Position(), int64(t))
    case string:
        if t == "" {
            v = makeNone(ctx.Position())
        } else {
            v = makeBareword(ctx.Position(), t)
        }
    case []string: {
        var l = makeList()
        for _, s := range t {
            if s == "" {
                v = makeNone(ctx.Position())
            } else {
                v = makeBareword(ctx.Position(), s)
            }
            l.elems = append(l.elems, v)
        }
        v = l
    }
    case []interface{}:
        var l = makeList()
        for _, i := range t { l.elems = append(l.elems, va(ctx, i)) }
        v = l
    case nil:
        v = makeNull(ctx.Position())
    default:
        erro(ctx, "%v", us(i)).debug(2)
    }
    return
}
func vi(a ...Value) (ii []interface{}) {
    for _, v := range a { ii = append(ii, v) }
    return
}

func scalarize(v Value) (res Value) { // NOTE: unexpanded is not scalar
    switch t := v.(type) {
    case *none: t.x = scalarize(t.x)
    case *list:
        var n = len(t.elems)
        if n == 0 { return makeNull(t.Position()) }
        if n == 1 { return scalarize(t.elems[0]) }
    }
    return v
}

func uval(v Value) Value {
    switch t := v.(type) {
    case *list: if len(t.elems) == 1 { v = uval(t.elems[0]) }
    }
    return v
}

func ulist(v Value) (l *list) {
    switch t := v.(type) {
    case *list: l = t
    }
    return
}

func tv(i interface{}) (_ string) { return fmt.Sprintf("%s{%v}", typeof(i), i) }
func us(i interface{}) (s string) {
    if i == nil { return "nil" }

    var ts = typeof(i) //strings.Replace(fmt.Sprintf("%T", i), "smart.", "", -1)

    switch t := i.(type) {
    default:                  return fmt.Sprintf("%s{%v}",      ts, i)
    case *universe:           return fmt.Sprintf("%s{%v}",      ts, us(&t.diagnostic))
    case *loader:             return fmt.Sprintf("%s{%v}",      ts, us(&t.terminal))
    case *parser:             return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_:           return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_addprefix:  return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_addsuffix:  return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_auto:       return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_assert:     return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_call:       return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_foreach:    return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_finalize:   return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_file:       return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_if:         return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_string:     return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_trimprefix: return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_value:      return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *builtin_wildcard:   return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *configureContext:   return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *diagnostic:         return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *evocation:          return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *automatic:          return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case *terminal:           return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case condless:            return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case final:               return fmt.Sprintf("%s{%v}",      ts, us(t.Context))
    case partial:             return fmt.Sprintf("%s{%b %v}",   ts, t.bit, us(t.Context))
    case evaluation:          return fmt.Sprintf("%s{%v %v}",   ts, t.o,   us(t.Context))
    case original:            return fmt.Sprintf("%s{%v %v}",   ts, t.o,   us(t.Context))
    case argumented:          return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case as:                  return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case fullname:            return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case flag:                return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case opt:                 return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case skipped:             return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case selected:            return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case expanded:            return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case condval:             return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case conjunction:         return fmt.Sprintf("%s{{%v}%v}",  ts, us(t.list), us(t.sep))
    case disjunction:         return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case undef:               return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case untraversed:         return fmt.Sprintf("%s{%v}",      ts, us(t.Value))
    case ust:                 return                                us(t.i)
    case *positional:         return                                us(t.Context)
    case *pair:               return fmt.Sprintf("%s{%v=%v}",   ts, us(t.key), us(t.val))
    case *def:                return fmt.Sprintf("%s{%s⇒%v}",   ts, t.name,      us(t.value)) // ⇒
    case *auto:               return fmt.Sprintf("%s{%s}",      ts, t.name)
    case *pathpun:
        if t.token == PROOT { return "pathpun{root}" }
        if t.token == PTAIL { return "pathpun{tail}" }
        return fmt.Sprintf("%s{%s}", ts, t.token)
    case *strval:
        s = us(t.v)[1:]
        return fmt.Sprintf("%s{%s}", ts, s[:len(s)-1])
    case *barecomp:
        s = ts+"{"
        for i, a := range t.elems {
            if 0 < i { s += " " }
            s += us(a)
        }
        s += "}"
        return
    case *globpat:
        s = ts+"{"
        for i, a := range t.elems {
            if 0 < i { s += " " }
            s += us(a)
        }
        s += "}"
        return
    case *path:
        s = ts+"{"
        for i, a := range t.elems {
            if 0 < i { s += " " }
            s += us(a)
        }
        s += "}"
        return
    case *closure:
        s  = ts+"{"
        s += us(t.x)
        if t.o != nil { s += us(t.o) }
        for i, a := range t.a {
            if false && 0 < i { s += "," }
            s += " " + us(a)
        }
        s += "}"
        return
    case *delegate:
        s  = ts+"{"
        s += us(t.x)
        if t.o != nil { s += us(t.o) }
        for i, a := range t.a {
            if false && 0 < i { s += "," }
            s += " " + us(a)
        }
        s += "}"
        return
    case Value:
        s = ts+"{"
        s += t.String()
        s += "}"
        return
    case []Value:
        s = "["
        for i, v := range t {
            if i > 0 { s += " " }
            s += us(v)
        }
        s += "]"
        return
    }
}

type ust struct { i interface{} }
func (p ust) String() string { return us(p.i) }
func (p ust) Position() (pos Position) {
    if x, y := p.i.(Value); y { pos = x.Position() }
    return
}

func makeArgumented(val Value, a ...Value) *argumented { return &argumented{val, a} }
func makeAnswer(pos Position, v bool) *answer          { return &answer{boolean{valbase{pos},v}} }
func makeOption(pos Position, v bool) *option          { return &option{boolean{valbase{pos},v}} }

func makeNull(pos Position) *null { return &null{valbase{pos}} }
func makeNone(pos Position) *none { return &none{valbase{pos}, nil} }
func makeSelection(pos Position, tok token, lhs, rhs Value) *selection { return &selection{valbase{pos}, tok, lhs, rhs} }
func makeBoolean(pos Position, v bool) *boolean { return &boolean{valbase{pos},v} }
func makeBinary(pos Position, i int64) *binary { return &binary{integer{valbase{pos},i}} }
func makeOctal(pos Position, i int64) *octal { return &octal{integer{valbase{pos},i}} }
func makeDecimal(pos Position, i int64) *decimal { return &decimal{integer{valbase{pos},i}} }
func makeHexadecimal(pos Position, i int64) *hexadecimal { return &hexadecimal{integer{valbase{pos},i}} }
func makeFloat(pos Position, f float64) *Float  { return &Float{valbase{pos},f} }
func makeDate(pos Position, s time.Time) *Date  { return &Date{datetime{valbase{pos},s}} }
func makeTime(pos Position, t time.Time) *Time  { return &Time{datetime{valbase{pos},t}} }
func makeRaw(pos Position, s string) *raw       { return &raw{valbase{pos},s} }
func makeStrlit(pos Position, s string) *strlit { return &strlit{valbase{pos},s} }
func makeUrl(pos Position, s *url.URL) *URL {
    var host, port string
    v := strings.Split(s.Host, ":")
    if len(v) == 1 { host = v[0] }
    if len(v) == 2 { host, port = v[0], v[1] }
    var password Value
    if t, ok := s.User.Password(); ok {password = makeStrlit(pos, t)}
    return &URL{ // FIXME: calculate component positions
        valbase: valbase{pos},
        Scheme: makeStrlit(pos, s.Scheme),
        Username: makeStrlit(pos, s.User.Username()),
        Password: password,
        Host: makeStrlit(pos, host),
        Port: makeStrlit(pos, port),
        Path: makeStrlit(pos, s.Path),
        Query: makeStrlit(pos, s.RawQuery),
        Fragment: makeStrlit(pos, s.Fragment),
    }
}
func makeBareword(pos Position, word string) *bareword { return &bareword{valbase{pos},word} }
func makeBarecomp(elems ...Value) *barecomp { return &barecomp{elements{elems}} }
func makeCompound(elems ...Value) *compound { return &compound{elements{elems}} }
func makeList(elems ...Value) *list { return &list{elements{elems}} }
func _makeList[T Value](ii ...T) *list {
    var l = &list{elements{}}
    for _, i := range ii { l.elems = append(l.elems, i) }
    return l
}
func makeGroup(pos Position, elems ...Value) (v *group) { return &group{valbase{pos},elements{elems}} }
func makeGlobMeta(pos Position, tok token) *globmeta { return &globmeta{valbase{pos},tok} }
func makeGlobRange(pos Position, v Value) *globrange { return &globrange{valbase{pos},v} }
func makeGlobPat(ctx Context, elems ...Value) *globpat { return &globpat{elements{elems}} }

func makePair(k, v Value) (p *pair) { return &pair{k, v} }
func makePath(segments ...Value) *path { return &path{elements{segments}} }
func makePathPun(pos Position, t token) *pathpun { return &pathpun{valbase{pos},t} }
func makePercpat(pos Position, prefix, suffix Value) *percpat {
    if prefix == nil { prefix = &null{valbase{pos}} }
    if suffix == nil { suffix = &null{valbase{pos}} }
    return &percpat{valbase{pos},prefix,suffix}
}
func makeDelegate(pos Position, tok token, obj Value, opts []Value, args ...Value) Value {
    return &delegate{valbase{pos}, tok, obj, opts, args}
}
func makeClosure(pos Position, tok token, obj Value, opts []Value, args ...Value) Value {
    return &closure{delegate{valbase{pos}, tok, obj, opts, args}}
}

func Make(pos Position, in interface{}) (out Value) {
    switch v := in.(type) {
    case int:       out = makeDecimal(pos,int64(v))
    case int32:     out = makeDecimal(pos,int64(v))
    case int64:     out = makeDecimal(pos,v)
    case float32:   out = makeFloat(pos,float64(v))
    case float64:   out = makeFloat(pos,v)
    case string:    out = makeStrlit(pos, v)
    case time.Time: out = &datetime{valbase{pos},v} // FIXME: NewDate, NewTime
    case Value:     out = v
    default:    out = &any{in} // TODO: position for any
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

func ParseBinary(pos Position, s string) *binary {
    if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
        s = s[2:]
    }
    if i, e := strconv.ParseInt(s, 2, 64); e == nil {
        return makeBinary(pos,i)
    } else {
        panic(e)
    }
}

func ParseOctal(pos Position, s string) *octal {
    if strings.HasPrefix(s, "0") {
        s = s[1:]
    }
    if i, e := strconv.ParseInt(s, 8, 64); e == nil {
        return makeOctal(pos,i)
    } else {
        panic(e)
    }
}

func ParseDecimal(pos Position, s string) *decimal {
    if i, e := strconv.ParseInt(s, 10, 64); e == nil {
        return makeDecimal(pos,i)
    } else {
        panic(e)
    }
}

func ParseHexadecimal(pos Position, s string) *hexadecimal {
    if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
        s = s[2:]
    }
    if i, e := strconv.ParseInt(s, 16, 64); e == nil {
        return makeHexadecimal(pos,i)
    } else {
        panic(e)
    }
}

func ParseFloat(pos Position, s string) *Float {
    if f, e := strconv.ParseFloat(strings.Replace(s, "_", "", -1), 64); e == nil {
        return makeFloat(pos,f)
    } else {
        panic(e)
    }
}

func ParseDate(pos Position, s string) *Date {
    if t, e := time.Parse("2006-01-02", s); e == nil {
        return makeDate(pos,t)
    } else {
        panic(e)
    }
}

func ParseTime(pos Position, s string) *Time {
    if t, e := time.Parse("15:04:05.999999999Z07:00", s); e == nil {
        return makeTime(pos,t)
    } else {
        panic(e)
    }
}

func ParseURL(pos Position, s string) *URL {
    if u, e := url.Parse(s); e == nil {
        return makeUrl(pos,u)
    } else {
        panic(e)
    }
}

const (
    max_evoke = 999
    fixEvokedFullnames = false
)

// NOTE: evokeTraceDots is for debugging call trace, if this finally goes into a formal
//       feature, it should need a sync-lock protection.
var evokeTraceDots string

func _evocation(c Context) *evocation { return cast[*evocation](c) }

type evocation struct {
    Context
    a []Value
    o []Value
}
func (p *evocation) cast(t reflect.Type) Context { return implcast(p, t) }
func (p *evocation) do(prop property, a ...interface{}) interface{} {
    return p.Context.do(prop, a...)
}

func evoke(ctx Context, x Value, o, a []Value) (_ Value, _ []Value) {
    // NOTE: the evo.a represents the arguments, which is a COPY of the original slice;
    // NOTE: making a COPY of the arguments FIXES the bug of delegate-altered-args.
    switch ev := (evocation{ctx, copyvals(a), o}); i := x.(type) {
    case *closure, *delegate, nil:
        erro(ctx, "illicit x=%v, o=%v, a=%v", us(x), us(o), us(a)).debug(16)
        return
    case interface{ evoke(*evocation) Value }:
        return i.evoke(&ev), ev.a
    default:
        return x.expand(&ev), ev.a
    }
}

func invoke(ctx Context, v Value, o, a []Value) (res Value) {
    res, _ = evoke(ctx, v, o, a)
    return
}

type opt  struct { Value }
type opts struct { vals []Value }

func ao(ctx Context, ii ...interface{}) (a, o []Value) {
    for _, i := range ii {
        switch t := i.(type) {
        case    opt : o = append(o, t.Value)
        case    opts: o = append(o, t.vals...)
        case []Value: a = append(a, t...)
        case   Value: a = append(a, t)
        default:      a = append(a, va(ctx, i))
        }
    }
    return
}

func inv(ctx Context, v Value, ii ...interface{}) (res Value) {
    var a, o = ao(ctx, ii...)
    return invoke(ctx, v, o, a)
}

func call(ctx Context, name string, o []Value, a ...Value) (res Value) {
    if v := _universe(ctx).scope.lookup(name); v != nil {
        if t, _ := evoke(ctx, v, o, a); !equal(ctx, v, t) { res = t }
    }
    return
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
