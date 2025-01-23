//
//  Copyright (C) 2012-2022, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "bytes"
    binenc "encoding/binary"
    "crypto/sha256"
    "errors"
    "fmt"
    "hash/fnv" // "hash/maphash"
    "io/fs"
    "math"
    neturl "net/url"
    "os"
    "path/filepath"
    "reflect"
    "regexp"
    "runtime"
    "runtime/debug" // debug.PrintStack()
    "strconv"
    "strings"
    "time"
    "unicode/utf8"
)

type hashbytes [sha256.Size]byte

const escapedChars = "\"\r\n"

const (
    recursiveTraversalClosurePre = false
    recursiveTraversalClosurePost = false
    recursiveTraversalClosure = true
)

const (
    enable_assertions   = true
    enable_grep_bench   = true
    traverseDetectLoops = true // turn on/off traverse loop detection
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

type Kind uint64

const (
    KindUnclassified Kind = 0
    KindUndef Kind = 1<<iota
    KindDelegate
    KindClosure
    KindArgumented
    KindReturner
    KindNone
    KindNull
    KindEscaped
    KindBoolean
    KindFlag
    KindAny
    KindArray
    KindGroup
    KindList
    KindPlainLine
    KindPlain

    KindInteger
    KindBinary
    KindOctal
    KindDecimal
    KindHexadecimal

    KindFloat
    KindRaw
    KindStrLit
    KindStrVal // aka intermediate strlit
    KindWord
    KindBarefile
    KindStrcomp
    KindDateTime
    KindDate
    KindTime

    KindPair
    KindPath
    KindPunct
    KindQuoted
    KindUrl
    KindCompound
    KindCond
    KindGlobpat
    KindRecipe

    KindObject
    KindKnownObject
    KindUnexpanded
    KindUndetermined
    KindExpanded
    KindSkipped
    KindSelf
    KindProject
    KindBuiltin
    KindAuto
    KindDef
    KindRule
    KindStemmedRule

    KindModifier
    KindModification

    KindUse

    KindNumber = KindBoolean|KindInteger|KindFloat
    // TODO: KindObject = ...
)

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

func sfmt(f string, i ...any) string { return fmt.Sprintf(f, i...) }

func original_bits(o origin) (_ property) {
    switch o {
    case defConfig:
    case defConfDir:
    case defConfRef:
    case defDecl:
    case defExecute: //  !=
    case defExpand0: //   =
        return propExDef|propExDef0
    case defExpand1: //  :=
        return propExDef|propExDef1|propExPairVal
    case defExpand2: // ::=
        return propExDef|propExDef2|propExPairVal
    case defExpand3: // ;:= (TODO)
        return propExDef|propExDef3|propExPairVal
    }
    return
}

type get_origin struct{}

// Original initiation of def values.
type original struct{ Context ; o origin }
func (c original) cast(t reflect.Type) Context { return icast(c, t) }
func (c original) inner() Context { return c.Context }
func (c original) ts(t string) string {
	return fmt.Sprintf("{=%s %v %v}", t, c.o, ts(c.Context))
}
func (c original) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case get_origin:  return c.o
    case ex_closure:  return c.o == defExpand2
    case ex_delegate: return c.o == defExpand1
    case property: if t&original_bits(c.o) != 0 { return true }
    }
    return c.Context.do(ctx, op)
}

// Optimize value for final strings
type final struct{ Context }
func (c final) cast(t reflect.Type) Context { return icast(c, t) }
func (c final) inner() Context { return c.Context }
func (c final) ts(t string) string { return fmt.Sprintf("{=%s %v}", t, ts(c.Context)) }
func (c final) do(ctx Context, op any) any {
    switch t := op.(type) {
    case ex_delegate, ex_closure: return true
    case final: return c
    case property:
        if t&(propExAuto|propExDefValue|propExPairVal) != 0 { return true }
    }
    return c.Context.do(ctx, op)
}

func _final(ctx Context) Context {
    if x, y := ctx.(final); y {
        return x
    } else {
        return final{ctx}
    }
}

type ex_path_str struct{}
type expandPathStr struct{ Context }
func (c expandPathStr) cast(t reflect.Type) Context { return icast(c, t) }
func (c expandPathStr) inner() Context { return c.Context }
func (c expandPathStr) do(ctx Context, op any) any {
    switch op.(type) {
    case ex_path_str: return true
    }
    return c.Context.do(ctx, op)
}

type ex_condless struct{}
type condless struct{ Context }
func (c condless) cast(t reflect.Type) Context { return icast(c, t) }
func (c condless) inner() Context { return c.Context }
func (c condless) do(ctx Context, op any) any {
    switch op.(type) {
    case ex_condless: return true
    }
    return c.Context.do(ctx, op)
}

type reversal struct{ Context }
func (c reversal) cast(t reflect.Type) Context { return icast(c, t) }
func (c reversal) inner() Context { return c.Context }
func (c reversal) do(ctx Context, op any) any {
    switch t := op.(type) {
    case property:
        if t&propReversal != 0 { return true }
    }
    return c.Context.do(ctx, op)
}

type partialBit uint
type partial struct{ Context ; bit partialBit }
func (c partial) cast(t reflect.Type) Context { return icast(c, t) }
func (c partial) inner() Context { return c.Context }
func (c partial) ts(t string) string {
    return fmt.Sprintf("{=%s %b %v}", t, c.bit, ts(c.Context))
}
func (c partial) do(ctx Context, op any) any {
    if x, y := op.(is_good_with) ; y {
        for _, v := range x.a {
            switch t := v.(type) {
            case *closure : return c.do(ctx, op)
            case *delegate: return c.do(ctx, op)
            case *auto:
                if true || x.p&(propExAuto) != 0 {
                    if c.bit&placeholderPart != 0 && t.isPlaceholder() { return true }
                    if c.bit&digitalPart != 0 && t.isDigit() { return true }
                }
            }
        }
    }
    return c.Context.do(ctx, op)
}

type negate struct{ Context ; property }
func (c negate) cast(t reflect.Type) Context { return icast(c, t) }
func (c negate) inner() Context { return c.Context }
func (c negate) do(ctx Context, op any) (_ any) {
    if x, y := op.(property); y && x&c.property != 0 { return }
    return c.Context.do(ctx, op)
}

// A comment node represents a single #-style comment.
type comment struct{
    pos Position // position of "#" starting the comment
    string // comment text (excluding '\n')
}
func (c *comment) String() string { return "{"+c.string+"}" }

// A commentgroup represents a sequence of comments
// with no other tokens and no empty lines between.
type commentgroup struct{ comments []*comment }
func (g *commentgroup) Position() Position { return g.comments[0].pos }

type statinfo struct{
    file *file
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

var traveTargetNotDefinedFile = fmt.Errorf("target not defined as file")

// TODO: use chained-context instead of 'term' for closure-scopes
type term struct{ Context ; *scope }
func (c *term) cast(t reflect.Type) Context { return icast(c,t) }
func (c *term) inner() Context { return c.Context }
func (c *term) ts(string) (s string) {
    if c.scope != nil {
        if x, y := c.Context.(*abs_ctx); false && y {
            if s := bases(x.abs, 2, true); s == c.comment {
                return "{=term "+s+" "+ts(x.Context)+"}"
            }
        }
        return "{=term "+c.comment+" "+ts(c.Context)+"}"
    } else {
        return ts(c.Context)
    }
}
func (c *term) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case get_scope: return c.scope
    case get_closure_scopes:
        var res []*scope
        if c.scope != nil {
            res = append(res, c.scope)
        }
        if cc := c.Context; cc != nil {
            if x, y := do(cc, t).([]*scope); y && x != nil {
                res = append(res, x...)
            }
        }
        return res
    }
    return c.Context.do(ctx, op)
}

func closure_scopes(ctx Context) (s []*scope) {
  s, _ = do(ctx, get_closure_scopes{}).([]*scope)
  return
}

func closure_projects(ctx Context) (res []*project) {
    var m = map[*project]struct{}{}
    for _, s := range closure_scopes(ctx) {
        // NOTE: the globe scope has nil project
        if s.project != nil {
            if _, y := m[s.project]; !y {
                for _, p := range s.project.family() { m[p] = struct{}{} }
                res = append(res, s.project)
            }
        }
    }
    return
}

func closure_finddef(ctx Context, name string) (res *def) {
    for _, s := range closure_scopes(ctx) {
        if s.project == nil {
            if _, obj := s.find(name); obj == nil {
                continue
            } else if res, _ = obj.(*def); res != nil {
                return
            }
        } else {
            var pos = _position(ctx)
            if !pos.IsValid() { pos = s.project.position }
            if s != s.project.scope {
                if _, obj := s.find(name); obj != nil {
                    if res, _ = obj.(*def); res != nil {
                        return
                    }
                }
            }
            if obj := s.project.resolve(ctx, name); obj == nil {
                if res = auto_find(ctx, name); res != nil { return }
            } else if res, _ = obj.(*def); res != nil {
                return
            }
        }
    }
    return
}

func closure_get(ctx Context, name string) (res *def) {
    panic("TODO: def "+name)
}

func closure_set(ctx Context, name string, val Value) (prev Value, okay bool) {
    for _, s := range closure_scopes(ctx) {
        if def := s.finddef(name); def != nil {
            prev = def.value
            def.val(ctx, val)
            okay = true
            break
        }
    }
    return
}

func closure_files(ctx Context, name string, one bool) (res []*file) {
    var a = unmap_files(ctx, name)
    if f := select_file(ctx, a); f != nil {
        if res = append(res, f); one { /* break */ }
    }
    return
}

func closure_resolve(ctx Context, name string) (obj object) {
    for _, s := range closure_scopes(ctx) {
        var proj = s.project
        if proj == nil || s != proj.scope {
            if _, obj = s.find(name); isNull(obj) {
                // fallthrough
            } else if a, y := obj.(*auto); y { // assert(a.name == name)
                return auto_find(ctx, a.ident(ctx))
            } else {
                return
            }
        }
        if proj != nil { obj = proj.resolve(ctx, name) }
        if !isNull(obj) { return }
    }
    return
}

func closure_entries(ctx Context, name any) (entries []entry) {
    for _, s := range closure_scopes(ctx) {
        var proj = s.project
        if proj != nil {
            entries = proj._entries(ctx, name, false)
        }
        if entries != nil { break }
    }
    return
}

func closure_entry(ctx Context, name any, _ ...bool) (_ entry) {
    var entries = closure_entries(ctx, name)
    if n := len(entries) ; 0 < n {
        if 1 < n { erro(ctx, "%d entries: %s", n, name).trace() }
        return entries[0]
    }
    return
}

func closure_with(ctx Context, a ...any) Context {
    for _, i := range a {
        switch t := i.(type) {
        case *project: ctx = &term{ctx, t.scope}
        case   *scope: ctx = &term{ctx, t}
        case []*scope: for _, s := range t {  ctx = &term{ctx, s}  }
        case interface{ declscope() *scope }: ctx = &term{ctx, t.declscope()}
        }
    }
    return ctx
}

func refdef(ctx Context, val Value, origin origin) (res bool) {
    for _, def := range val.defs(ctx) {
        if def.o == origin { return true }
        if true && def.value != nil && refdef(ctx, def.value, origin) { return true }
    }
    return
}

func entryIndicator(ctx Context, entry Value) (str, ent, tar string) {
    if !isNull(entry) { ent = entry.string(ctx) }
    if val := auto_get(ctx, "@"); val == nil || isTrivial(val) {
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
    } else if project := _project(ctx); project != nil {
        dir = project.tmpPath
    } else if scope := _scope(ctx); scope != nil && scope.project != nil {
        dir = scope.project.tmpPath
    }
    var h = fmt.Sprintf("%x", k[:2]) // HEX of the first two bytes
    return filepath.Join(dir, ".hash", h[0:1], h[1:2], h[2:3], h[3:])
}

func (p *execution) getRecipesHash(ctx Context, target Value, values ...Value) (k, v hashbytes, err error) {
    var key = sha256.New()
    var val = sha256.New()

    fmt.Fprintf(key, "%s", target.string(ctx))

    if p.prog != nil && p.prog.project != nil {
        fmt.Fprintf(key, "%s", p.prog.project.absPath)
    }

    for _, value := range values {
        fmt.Fprintf(val, "%v", value)
    }

    copy(k[:], key.Sum(nil))
    copy(v[:], val.Sum(nil))
    return
}

func (p *execution) updateRecipesHash(ctx Context, target Value) (k, v hashbytes, err error) {
    if k, v, err = p.getRecipesHash(ctx, target, p.recipes...); err != nil {
        erro(ctx, "hashing recipes failed: %v", err).trace()
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

func (p *execution) isRecipesChanged(ctx Context, target Value) (outdated bool, err error) {
    var k, v hashbytes
    if k, v, err = p.getRecipesHash(ctx, target, p.recipes...); err != nil {
        erro(ctx, "compute recipes hash failed: %v", err).trace()
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

type waitopts struct{
    ReportUpdates      bool
    ExecResults        bool
    StampCurrentTarget bool
}
func wait(ctx Context, opts waitopts) (target Value, fs []*file, execRes *exec_result) {
    var calleeErrs []error
    if p := _execution(ctx) ; p != nil {
        if false { p.WaitGroup.Wait() } // FIXME: deadlock

        p.calleeErrsM.Lock()
        calleeErrs = p.calleeErrs; p.calleeErrs = nil
        p.calleeErrsM.Unlock()
    }

    if target = auto_target_value(ctx); target == nil {
        errostack(ctx, 8, "target is nil").trace()
    }

    if isTrivial(target) {
        erro(ctx, "trivial target : %s", ts(target))
        errostack(ctx, 8).trace()
    }

    if n := len(calleeErrs); n > 0 /*&& t.stems == nil*/ {
        var numRealErrs = 0
        for _, err := range calleeErrs {
            erro(ctx, "%v: %v", target, err).trace()
            numRealErrs += 1
        }
        if numRealErrs == 0 { return } // simply return if no real errors

        var ctxPos, targetPos = _position(ctx), target.Position()
        var v = target
        if l, ok := v.(*list); ok && l.len() == 1 { v = l.elems[0] }
        if targetPos.IsValid() && !targetPos.same(&ctxPos) {
            if f, y := to_file(v); y && f != nil && f.filemap != nil {
                erro(ctx, "waiting for '%v'", target)
                erro(ctx, "via pattern '%v' (of %v)", v, f.filemap.project).trace()
            } else {
                erro(ctx, "waiting for '%v'", target).trace()
            }
        }
        if def, ok := v.(*def); ok && target != v && target != def.value { // trace source Def in diagnostics
            erro(ctx, "waiting for def '%v': %v", def.name, def.value).trace()
        }
        return
    }

    if opts.ExecResults {
        // Waiting for command (shell/python/etc.) exec result
        if val := auto_get(ctx, "-"); val != nil {
            var ok bool
            if execRes, ok = val.(*exec_result); ok {
                //execRes.wg.Wait()
            }
        }
    }

    if opts.StampCurrentTarget {
        if fs = target.stamp(ctx); fs != nil {
            if false { reportFileUpdates(ctx, fs) }
        }
    }
    return
}

type as struct{ Value }
func (a as) ts(string) string { return "{=as "+ts(a.Value)+"}" }
func (a as) file(ctx Context, projs ...*project) (res *file) {
    var v = scalarize(a.Value.expand(ctx))
    if checkpoints && truly(ctx, is_test_mode{}) {
        defer a.file_check(ctx, projs, a.Value, &res)
    }
    if x, y := v.(fullname); y { v = x.Value }
    switch t := v.(type) {
    case as: return t.file(ctx, projs...)
    case  fullfile: return t.file
    case *barefile: return t.file
    case *file: return t
    case *rule: return as{t.target}.file(ctx)
    case *def : if t.value != nil { return as{t.value}.file(ctx) }
    case *list: if a := t.elems; len(a) == 1 { return as{a[0]}.file(ctx) }
    case *word, *compound, *path:
        if projs == nil {
            if p := _project(ctx); p != nil { projs = append(projs, p) }
        }
        for _, p := range projs {
            if f := p.file(ctx, t); f != nil { return f }
        }
    default: // *strlit, *strval, *strcomp:
        // NOTE: not parsing 'string' and "strcomp" values to file to optimize.
        // erro(pc(ctx,a), "cannot convert to file: %v", tv(a.Value)).trace()
    }
    return
}
func (a as) file_fullname(ctx Context, projs ...*project) (f *file, s string, y bool) {
    if f = a.file(ctx, projs...); f != nil {
        if s = f.fullname(); filepath.IsAbs(s) {
            y = true
        }
    }
    return
}
func (a as) fullname_string(ctx Context, projs ...*project) (s string, y bool) {
    if _, s, y = a.file_fullname(ctx, projs...); !y { s = a.string(ctx) }
    return
}
func (a as) fullname(ctx Context, projs ...*project) (res fullname) {
    if a.Value != nil {
        if f := a.file(ctx, projs...); f != nil {
            res.Value = f
        } else {
            errostack(pc(ctx,a), 3, "nil file : %v : %v → %v", a.Value, ts(a.Value), a.Value.expand(ctx)).trace()
        }
        if checkpoints && truly(ctx, is_test_mode{}) {
            a.fullname_check(ctx, projs, a.Value, res)
        }
    }
    return
}

// joinpath is different from filepath.Join, which trims and discards empty segments
func joinpath(segs ...string) string { return strings.Join(segs, pathSep) }
func _joinpath(ctx Context, i any) (_ string) {
    switch s := i.(type) {
    case      nil:
    case   string: return s
    case []string: return joinpath(s...)
    case interface{ string(Context) string }: return s.string(ctx)
    default:
        note(ctx, "unexpected path str: %s", ts(i)).debug(6)
    }
    return
}

func joinraws(sep string, vals ...*raw) string {
    var strs []string
    for _, v := range vals { strs = append(strs, v.String()) }
    return strings.Join(strs, sep)
}

type posctx struct{ Context ; pos any }
func (p *posctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *posctx) inner() Context { return p.Context }
func (p *posctx) ts(string) string {
    switch t := p.pos.(type) {
    case Position:
        var s = bases(t.Filename, 3, true)
        return fmt.Sprintf("{=pc %s %s}", s, ts(p.Context))
    default:
        return fmt.Sprintf("{=pc %s %s}", t, ts(p.Context))
    }
}
func (p *posctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case get_position:
        if p.pos != nil {
            var pos Position
            switch t := p.pos.(type) {
            case positioner: pos = t.Position()
            case Position: pos = t
            }
            if pos.valid() { return pos }
        }
    }
    return p.Context.do(ctx, op)
}

func pc(ctx Context, a any, n ...int) Context {
    var p any
    if a != nil {
        switch t := a.(type) {
        case []byte: a = string(t)
        case *file: a = t.fullname()
        }
        switch t := a.(type) {
        case  *scanner   : p = t.pos(n...)
        case  *parser    : p = t.Position()
        case   Position  : if t.valid() { p = t }
        case   positioner: if t != nil  { p = t.Position() }
        case []positioner:
            for _, v := range t {
                if x := v.Position(); x.valid() {
                    p = x
                    break
                }
            }
        case []Value:
            for _, v := range t {
                if x := v.Position(); x.valid() {
                    p = x
                    break
                }
            }
        case string:
            var pos Position
            pos.Filename = t
            if 0 == len(n) { pos.Line = 1 }
            if 0 <  len(n) { pos.Line = n[0] }
            if 1 <  len(n) { pos.Column = n[1] }
            p = pos
        }
    }
    if p != nil { ctx = &posctx{ctx,p} }
    return ctx
}

type stringify_empty struct{}
type stringify_ctx struct{
    Context
    empty bool
}
func (sc *stringify_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case ex_delegate, ex_closure: return true
    case stringify_empty: sc.empty = true; return
    }
    return sc.Context.do(ctx, op)
}

type positioner interface{ Position() Position }
type identer    interface{ ident(Context) string }
type stringer   interface{ string(Context) string }

type Value interface{
    positioner // The position where the value appears (or NoPos).
    identer
    stringer

    // doer

    cond() bool
    kind() Kind

    // Literal representations of the value.
    String() string

    hash(Context) uint64

    // Returns true if the value can be evaluated as 'true', 'yes', etc.
    true(Context) bool

    // Integer returns the integer form of the value.
    int(Context) int64

    // float returns the float form of the value.
    float(Context) float64

    // Equality compare.
    cmp(Context, Value) cmpres

    // whether this value can be used as a pattern
    patterned(Context) bool

    // Match a Value or string, returned 's' is the matched string (or heading part).
    match(Context, any) (full bool, s any, stems []string)

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

    stat(Context) *statinfo

    // Stamp the value if it's a file (aka. update FileInfo).
    stamp(Context) []*file

    // Delete the file (if it is).
    delete(Context) []*file

    traverse(Context)

    updated(Context) bool
    updatedDeps(Context, ...Value) []Value
}

func typeof(arg any) (s string) {
    defer func() { if s == "" { panic(fmt.Sprintf("empty typeof: %T", arg)) } } ()

    if arg == nil { return "nil" }

    if true {
        /* not specially handling list */
    } else if a, y := arg.(*list); y {
        if a.len() == 1 {
            if true {
                /* not specially handling delegate */
            } else if t, y := a.elems[0].(*delegate); y {
                if d, y := t.x.(*def); y && d != nil {
                    return typeof(d.value)
                } else {
                    return typeof(t.x)
                }
            }
            return typeof(a.elems[0])
        } else if false {
            return "none"
        }
    }

    switch t := reflect.TypeOf(arg) ; t.Kind() {
    case reflect.Array: return "[]"+t.Elem().Name()
    case reflect.Slice: return "[]"+t.Elem().Name()
    case reflect.Ptr  : return      t.Elem().Name()
    default:            return      t.Name()
    }
}

func is(v Value, a any) bool {
    switch t := a.(type) {
    case Kind:         return v.kind()&t != 0
    case reflect.Type: return reflect.TypeOf(v) == t
    }
    return reflect.TypeOf(v) == reflect.TypeOf(a)
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

func eq(x Context, a, b Value) bool {
    return a == b || a.cmp(x, b) == cmpEqual
}

func equal(x Context, a, b Value, s ...bool) (_ bool) {
    if a == b || a.cmp(x, b) == cmpEqual { return true } else { return }

    var y bool
    for _, y = range s { if y { break } }
    return y && a.string(x) == b.string(x)
}

func diff(ctx Context, a, b []Value) bool {
    if len(a) == len(b) {
        for i, v := range a { if !equal(ctx, v, b[i]) { return true } }
        return false
    } else {
        return true
    }
}

func indeterminate(ctx Context, vals ...Value) (_ bool) {
    var x, y = do(ctx, final{}).(final)
    if !y { x.Context = ctx }
    for _, v := range vals {
        if v.expandable(x) {
            return true
        }
    }
    return
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
    case *strval: t = len(a.v) == 0
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
    case *compound: return t.prefix(ctx, x)
    case     *pair: return t.prefix(ctx, x)
    }
    return _compound(x).suffix(ctx, y)
}

func _suffix(ctx Context, x, y Value) Value {
    switch _x := x.(type) { case i_suffix: return _x.suffix(ctx, y) }
    return _compound(x).suffix(ctx, y)
}

func compose(ctx Context, x, y Value) Value {
    switch _x := x.(type) { case i_suffix: return _x.suffix(ctx, y) }
    switch _y := y.(type) { case i_prefix: return _y.prefix(ctx, x) }

    erro(ctx, "compose: %v ⇔ %v : %v ⇔ %v", x, y, ts(x), ts(y)).trace()

    return _compound(x, y)
}

func has_prefix(str string, prefixs ...string) (res bool) {
    for _, s := range prefixs { if res = strings.HasPrefix(str, s); res { break }}
    return
}

func has_suffix(str string, suffixs ...string) (res bool) {
    for _, s := range suffixs { if res = strings.HasSuffix(str, s); res { break }}
    return
}

func isAny(str string, ss ...string) bool {
    for _, s := range ss { if str == s { return true } }
    return false
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

type valbase struct{ position Position }
func (p *valbase) Position() Position { return p.position }
func (_ *valbase) kind() Kind { return KindUnclassified }
func (_ *valbase) cond() bool { return false }
func (_ *valbase) cmp(Context, Value) (_ cmpres) { return }
func (_ *valbase) do(Context, any) (_ any) { return }
func (_ *valbase) defs(Context, ...string) (_ []*def) { return }
func (_ *valbase) delete(Context) (_ []*file) { return }
func (_ *valbase) expand(Context) (_ Value) { return }
func (_ *valbase) expandable(Context) (_ bool) { return }
func (_ *valbase) float(Context) (_ float64) { return }
func (_ *valbase) ident(Context) (_ string) { return }
func (_ *valbase) int(Context) (_ int64) { return }
func (_ *valbase) match(Context, any) (_ bool, _ any, _ []string) { return }
func (_ *valbase) patterned(Context) (_ bool) { return }
func (_ *valbase) refs(Context, Value) (_ bool) { return }
func (_ *valbase) stamp(Context) (_ []*file) { return }
func (_ *valbase) stat(Context) (_ *statinfo) { return }
func (_ *valbase) stencil(Context, []string) (_ Value, _ []string) { return }
func (_ *valbase) string(Context) (_ string) { return }
func (_ *valbase) String() (_ string) { return }
func (_ *valbase) true(Context) (_ bool) { return }
func (_ *valbase) updated(Context) (_ bool) { return }
func (_ *valbase) updatedDeps(Context, ...Value) (_ []Value) { return }
func (_ *valbase) traverse(Context) { }

type returner struct{ valbase ; vals []Value }
func (p *returner) kind() Kind { return KindReturner }
func (p *returner) hash(ctx Context) uint64 {
    var a []any
    for _, v := range p.vals { a = append(a, v) }
    return fnv1(ctx, p, a...)
}
func (p *returner) expand(ctx Context) (res Value) {
    if vals := expand(ctx, p.vals...) ; diff(ctx, vals, p.vals) {
        res = &returner{p.valbase, vals}
    } else {
        res = p
    }
    return
}

func fnv1(ctx Context, t any, a ...any) (_ uint64) {
    var h = fnv.New64()
    var o = binenc.LittleEndian
    if t != nil {
        var a []byte
        switch x := t.(type) {
        case interface{ kind() Kind }:
            a = make([]byte, 8)
            o.PutUint64(a, uint64(x.kind()))
        case Kind:
            a = make([]byte, 8)
            o.PutUint64(a, uint64(x))
        default:
            a = []byte(typeof(x))
        }
        h.Write(a)
    }

    for _, v := range a {
        var bs []byte
        switch t := v.(type) {
        case Kind:
            bs = o.AppendUint64(bs, uint64(t))
        case token:
            bs = o.AppendUint64(bs, uint64(t))
        case int64:
            bs = o.AppendUint64(bs, uint64(t))
        case uint64:
            bs = o.AppendUint64(bs, uint64(t))
        case string:
            bs = []byte(typeof(v))
        case []string:
            for _, s := range t { h.Write([]byte(s)) }
            return h.Sum64()
        case Value:
            if t == nil {
                bs = []byte{}
            } else if t.kind()&(KindBoolean|KindInteger|KindFloat|KindDateTime) != 0 {
                bs = o.AppendUint64(bs, uint64(t.int(ctx)))
            } else {
                bs = []byte(t.String()) // BUG: t.string(ctx)
            }
        default:
            erro(ctx, "fnv1: unsupported type : %s", ts(v)).trace()
        }
        h.Write(bs)
    }
    return h.Sum64()
}

type undef struct{ Value }
func (p undef) kind() Kind { return KindUndef }
func (p undef) hash(ctx Context) uint64 { return fnv1(ctx, p) }
func (p undef) String() string { return "{=undef "+p.Value.String()+"}" }
func (p undef) string(Context) (_ string) { return }
func (p undef) ts(t string) string { return fmt.Sprintf("{=%s %s}", t, ts(p.Value)) }
func (p undef) int(Context) (_ int64) { return }
func (p undef) float(Context) (_ float64) { return }
func (p undef) true(Context) (_ bool) { return }
func (p undef) prefix(_ Context, v Value) Value { return v }
func (p undef) suffix(_ Context, v Value) Value { return v }
func (p undef) expand(_ Context) Value { return p }
func (p undef) cmp(ctx Context, v Value) (_ cmpres) {
    if x, y := v.(*undef); y {
        return p.Value.cmp(ctx, x.Value)
    } else if l, y := v.(*list); y && l.len() == 1 {
        return p.cmp(ctx, l.elems[0])
    } else {
        return
    }
}

type null struct{ valbase }
func (_ *null) kind() Kind { return KindNull }
func (_ *null) String() string { return "{}" } // {=null}
func (p *null) ts(t string) string { return "{="+t+"}" }
func (p *null) hash(ctx Context) uint64 { return fnv1(ctx, p) }
func (p *null) prefix(_ Context, val Value) Value { return val }
func (p *null) suffix(_ Context, val Value) Value { return val }
func (p *null) expand(Context) Value { return p }
func (p *null) traverse(ctx Context) { erro(pc(ctx,p), "null traversal").trace() }
func (p *null) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if _, y := v.(*null); y {
        return cmpEqual
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}

type none struct{ valbase ; x Value }
func (_ *none) kind() Kind { return KindNone }
func (p *none) hash(ctx Context) uint64 { return fnv1(ctx, p) }
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
func (p *none) ts(t string) string { return fmt.Sprintf("{=%s %s}", t, ts(p.x)) }
func (p *none) true(ctx Context) (res bool) { return }
func (p *none) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if _, y := v.(*none); y {
        return cmpEqual
    } else if s, y := v.(*strlit); false && y && s.s == "" {
        return cmpEqual
    } else if l, y := v.(*list); y {
        if n := len(l.elems); n == 0 {
            return cmpEqual
        } else if n == 1 {
            return p.cmp(ctx, l.elems[0])
        }
    }
    return
}
func (p *none) prefix(_ Context, v Value) Value { return v }
func (p *none) suffix(_ Context, v Value) Value { return v }
func (p *none) expand(Context) Value { return p }
func (p *none) traverse(ctx Context) { erro(ctx, "none traversal").trace() }

type argumented_ctx struct{ Context ; val Value; args []Value }
func (ac *argumented_ctx) cast(t reflect.Type) Context { return icast(ac,t) }
func (ac *argumented_ctx) inner() Context { return ac.Context }
func (ac *argumented_ctx) ts(t string) (s string) {
    for i, a := range ac.args {
        if i > 0 { s += "," }
        s += a.String()
    }
    return "{="+t+" "+ac.val.String()+"("+s+") "+ts(ac.Context)+"}"
}
func (ac *argumented_ctx) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case get_args: return ac.args
    case init_args:
        t.args(ctx, ac.args)
        if checkpoints && truly(ctx, is_test_mode{}) {
            ac.init_args_check(ctx, ac.args)
        }
        return
    }
    return ac.Context.do(ctx, op)
}

type argumented struct{ Value ; args []Value }
func (_ *argumented) kind() Kind { return KindArgumented }
func (p *argumented) ts(t string) (s string) {
    return fmt.Sprintf("{=%s %s %s}", t, ts(p.Value), p.args)
}
func (p *argumented) hash(ctx Context) uint64 {
    var t = []any{ p.Value }
    for _, a := range p.args { t = append(t, a) }
    return fnv1(ctx, nil, t...)
}
func (p *argumented) String() (s string) {
    for i, a := range p.args {
        if i > 0 { s += "," }
        s += a.String()
    }
    s = fmt.Sprintf("%s(%s)", p.Value.String(), s)
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
    if checkpoints && truly(ctx, is_test_mode{}) {
        p.expand_check(ctx, res, val, args)
    }
    return
}
func (p *argumented) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(*argumented); y {
        if p.Value.cmp(ctx, x.Value) == cmpEqual {
            if len(p.args) == len(x.args) {
                for i, v := range p.args {
                    if v.cmp(ctx, x.args[i]) == cmpEqual {
                        return cmpEqual
                    }
                }
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *argumented) ctx(ctx Context) *argumented_ctx {
    if checkpoints && truly(ctx, is_test_mode{}) {
        if _project(ctx).name == "configure.base" {
            defer func(s string) {
                if s != p.String() {
                    errostack(ctx, 8, "%s != %s", p, s).debug(6)
                }
            } (p.String())
        }
    }

    //!< IMPORTANT! - Don't merge-expand arguments here!
    //!< Arguments should be passed to program.execute as it is.
    var args []Value
    var proj = _project(ctx)

    // NOTE: expand here to avoid args being expanded in the wrong context
    for _, a := range p.args {
        // TODO: deal with pattern args using expandPatterned instead of stenciling:
        if a = a.expand(_final(ctx)); a.patterned(ctx) {
            if stems := _stems(ctx); len(stems) > 0 {
                if v, rest := a.stencil(ctx, stems); len(rest) > 0 {
                    erro(ctx, "partial stencil: %v, %v, %v, %v", a, v, rest, stems).trace()
                } else if f := (as{a}).file(ctx, proj); f != nil {
                    a = f
                } else {
                    a = v
                }
            }
        }
        args = append(args, a)
    }

    if checkpoints && truly(ctx, is_test_mode{}) {
        p.traverse_check(ctx, p.String(), args)
    }

    return &argumented_ctx{ctx, p.Value, args}
}
func (p *argumented) traverse(ctx Context) {
    p.Value.traverse(p.ctx(ctx))
}
func (p *argumented) hit(ctx Context, c *valcache) (_ *valcache, _ bool) {
    return c.hit(ctx, p.Value)
}

type negative struct{ Value }
func (p negative) String() (s string) { return `!`+p.Value.String() }
func (p negative) ts(t string) (s string) { return "{="+t+" "+ts(p.Value)+"}" }
func (p negative) expand(ctx Context) Value { return negative{p.Value.expand(ctx)} }
func (p negative) hash(ctx Context) uint64 { return fnv1(ctx, p, p.Value) }
func (p negative) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(negative); y {
        return p.Value.cmp(ctx, x.Value)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
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
func (p negative) float(ctx Context) (res float64) {
    if !p.Value.true(ctx) { res = epsilon }
    return
}
func (p negative) int(ctx Context) (res int64) {
    if !p.Value.true(ctx) { res = 1 }
    return
}
func (p negative) traverse(ctx Context) { if p.Value != nil { p.Value.traverse(ctx) } }

func stringMatch(ctx Context, p Value, i any) (full bool, s string, stems []string) {
    var v = p.string(ctx)
    switch t := i.(type) {
    case Value:
        if w := t.string(ctx); strings.HasPrefix(w, v) { full, s = (len(v) == len(w)), v }
    case string:
        if strings.HasPrefix(t, v) { full, s = (len(v) == len(t)), v }
    case []string:
        if n := len(t); n > 0 { if t[0] == v { full, s = (n == 1), v } }
    default:
        errostack(ctx, 3, "%v: matching unsupported value: %v", ts(p), ts(i)).trace()
    }
    return
}

type escaped struct{ valbase; s string }
func (_ *escaped) kind() Kind { return KindEscaped }
func (p *escaped) String() string { return "\\" + p.s }
func (p *escaped) string(Context) (s string) {
    switch p.s {
    case "n": s = "\n"
    case "r": s = "\r"
    default:  s = p.s
    }
    return
}
func (p *escaped) true(Context) bool { return p.s != "" }
func (p *escaped) float(Context) (_ float64) { return }
func (p *escaped) int(Context) (_ int64) { return }
func (p *escaped) expand(Context) Value { return p }
func (p *escaped) hash(ctx Context) uint64 { return fnv1(ctx, nil, p.kind(), p.s) }
func (p *escaped) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if o, y := v.(*escaped); y {
        if p.s == o.s { res = cmpEqual }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *escaped) match(ctx Context, i any) (full bool, s any, stems []string) {
    return stringMatch(ctx, p, i)
}
func (p *escaped) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
    return
}
func (p *escaped) prefix(_ Context, val Value) Value {
    v := &compound{elements{}}
    if x, y := val.(*compound); y {
        v.elems = append(x.elems, p)
    } else {
        v.elems = append([]Value{val}, p)
    }
    return v
}
func (p *escaped) suffix(_ Context, val Value) Value {
    v := &compound{elements{}}
    if x, y := val.(*compound); y {
        v.elems = append([]Value{p}, x.elems...)
    } else {
        v.elems = []Value{p, val}
    }
    return v
}

type boolean struct{ valbase; bool }
func (_ *boolean) kind() Kind { return KindBoolean }
func (p *boolean) String() string { return "{="+p.string_()+"}" }
func (p *boolean) string(Context) string { return p.string_() }
func (p *boolean) string_() string { if p.bool { return "true" } else { return "false" } }
func (p *boolean) ts(t string) string { return "{="+t+" "+p.string_()+"}" }
func (p *boolean) true(Context) bool { return p.bool }
func (p *boolean) float(Context) (v float64) { if p.bool { v = 1. }; return }
func (p *boolean) int(Context) (v int64) { if p.bool { v = 1 }; return }
func (p *boolean) expand(Context) Value { return p }
func (p *boolean) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *boolean) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if a, y := v.(*option); y {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, y := v.(*answer); y {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if a, y := v.(*boolean); y {
        if p.bool == a.bool {
            res = cmpEqual
        } else if !p.bool && a.bool {
            res = cmpSmaller
        } else if p.bool && !a.bool {
            res = cmpGreater
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *boolean) match(ctx Context, i any) (full bool, s any, stems []string) {
    return stringMatch(ctx, p, i)
}
func (p *boolean) stencil(ctx Context, stems []string) (val Value, rest []string) {
    val, rest = p, stems
    return
}

type prediction struct{ boolean ; s string }

type answer struct{ boolean }
func (p *answer) String() string { return "{="+p.string_()+"}" }
func (p *answer) string(Context) string { return p.string_() }
func (p *answer) string_() string { if p.bool { return "yes" } else { return "no" } }
func (p *answer) ts(t string) string { return "{="+t+" "+p.string_()+"}" }
func (p *answer) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *answer) expand(Context) Value { return p }

type option struct{ boolean }
func (p *option) String() string { return "{="+p.string_()+"}" }
func (p *option) string(ctx Context) string { return p.string_() }
func (p *option) string_() string { if p.bool { return "on" } else { return "off" } }
func (p *option) ts(t string) string { return "{="+t+" "+p.string_()+"}" }
func (p *option) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *option) expand(Context) Value { return p }

func _optionalize(val Value) (name Value, okay bool) {
    if x, y := val.(*globpat); y {
        var i = x.len() - 1
        if m, y := x.elems[i].(*globmeta); y && m.token == QUE {
            name, okay = _compound(x.elems[:i]...), true
        }
    }
    return
}
func optionalize(ctx Context, val Value) (res cond, okay bool) {
    if v, y := _optionalize(val); y {
        res, okay = cond{v}, true
    } else if t, y := val.(*compound); y {
        if v, y := _optionalize(t.elems[t.len()-1]); y {
            x := _compound(t.elems[:t.len()-1]...)
            x.elems = append(x.elems, v)
            res, okay = cond{x}, true
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

type integer struct{ valbase; int64 }
func (_ *integer) kind() Kind { return KindInteger }
func (p *integer) true(ctx Context) bool { return p.int64 != 0 }
func (p *integer) int(ctx Context) (i int64) { return p.int64 }
func (p *integer) float(ctx Context) (f float64) { return float64(p.int64) }
func (p *integer) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *integer) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
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
    return
}

type binary struct{ integer }
func (p *binary) kind() Kind { return p.integer.kind()|KindBinary }
func (p *binary) String() string { return "0b"+strconv.FormatInt(int64(p.int64),2) }
func (p *binary) string(Context) string { return "0b"+strconv.FormatInt(int64(p.int64),2) }
func (p *binary) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *binary) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *binary) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *binary) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *binary) expand(Context) Value { return p }
func (p *binary) hit(ctx Context, c *valcache) (_ *valcache, _ bool) {
    return c.hit(ctx, p.string(ctx))
}

type octal struct{ integer }
func (p *octal) kind() Kind { return p.integer.kind()|KindOctal }
func (p *octal) String() string { return "0"+strconv.FormatInt(int64(p.int64),8) }
func (p *octal) string(Context) string { return "0"+strconv.FormatInt(int64(p.int64),8) }
func (p *octal) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *octal) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *octal) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *octal) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *octal) expand(Context) Value { return p }
func (p *octal) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    return c.hit(ctx, p.string(ctx))
}

type decimal struct{ integer }
func (p *decimal) kind() Kind { return p.integer.kind()|KindDecimal }
func (p *decimal) String() string { return strconv.FormatInt(int64(p.int64),10) }
func (p *decimal) string(Context) string { return strconv.FormatInt(int64(p.int64),10) }
func (p *decimal) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *decimal) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *decimal) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *decimal) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *decimal) expand(Context) Value { return p }
func (p *decimal) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    return c.hit(ctx, p.string(ctx))
}

type hexadecimal struct{ integer }
func (p *hexadecimal) kind() Kind { return p.integer.kind()|KindHexadecimal }
func (p *hexadecimal) String() string { return "0x"+strconv.FormatInt(int64(p.int64),16) }
func (p *hexadecimal) string(Context) string { return "0x"+strconv.FormatInt(int64(p.int64),16) }
func (p *hexadecimal) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *hexadecimal) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *hexadecimal) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *hexadecimal) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *hexadecimal) expand(Context) Value { return p }
func (p *hexadecimal) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    return c.hit(ctx, p.string(ctx))
}

const epsilon = 1e-15 /* 1e-16 */

type float struct{ valbase; float64 } // IEEE-754 64-bit binary floating-point
func (p *float) kind() Kind { return KindFloat }
func (p *float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *float) string(Context) string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }
func (p *float) true(Context) bool { return math.Abs(p.float64)-0 > epsilon }
func (p *float) int(Context) (i int64) { return int64(p.float64) }
func (p *float) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *float) float(ctx Context) (f float64) { return p.float64 }
func (p *float) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *float) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *float) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p *float) suffix(ctx Context, val Value) Value { return _bifix(ctx, p, val) }
func (p *float) expand(Context) Value { return p }
func (p *float) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if _, y := v.(*float); y {
        if f := v.float(ctx); p.float64 == f {
            return cmpEqual
        } else if p.float64 < f {
            return cmpSmaller
        } else if p.float64 > f {
            return cmpGreater
        }
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}

type datetime struct{ valbase ; t time.Time }
func (_ *datetime) kind() Kind { return KindDateTime }
func (p *datetime) String() string { return time.Time(p.t).Format("2006-01-02T15:04:05.999999999Z07:00") }
func (p *datetime) string(ctx Context) string { return p.String() } // time.RFC3339Nano
func (p *datetime) true(ctx Context) bool { return !p.t.IsZero() }
func (p *datetime) int(ctx Context) (i int64) { return p.t.Unix() }
func (p *datetime) float(ctx Context) (f float64) { return float64(p.t.Unix()) }
func (p *datetime) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *datetime) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *datetime) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *datetime) expand(Context) Value { return p }
func (p *datetime) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    var vt time.Time
    switch a := v.(type) {
    case *datetime: vt = a.t
    case *Date:     vt = a.t
    case *Time:     vt = a.t
    default: return
    }
    if p.t.Equal(vt) {
        return cmpEqual
    } else if p.t.Before(vt) {
        return cmpSmaller
    } else /*if p.t.After(vt)*/ {
        return cmpGreater
    }
}

func ParseDateTime(pos Position, s string) *datetime {
    // time.RFC3339Nano
    if t, e := time.Parse("2006-01-02T15:04:05.999999999Z07:00", s); e == nil {
        return &datetime{valbase{pos},t}
    } else {
        panic(e)
    }
}

type Date struct{ datetime }
func (p *Date) kind() Kind { return p.datetime.kind()|KindDate }
func (p *Date) String() string { return time.Time(p.t).Format("2006-01-02") }
func (p *Date) string(ctx Context) string { return p.String() }
func (p *Date) int(ctx Context) (i int64) { return p.t.Unix() }
func (p *Date) float(ctx Context) (f float64) { return float64(p.t.Unix()) }
func (p *Date) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *Date) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Date) expand(Context) Value { return p }

type Time struct{ datetime }
func (p *Time) kind() Kind { return p.datetime.kind()|KindTime }
func (p *Time) String() string { return time.Time(p.t).Format("15:04:05.999999999Z07:00") }
func (p *Time) string(ctx Context) string { return p.String() }
func (p *Time) int(ctx Context) (i int64) { return p.t.Unix() }
func (p *Time) float(ctx Context) (f float64) { return float64(p.t.Unix()) }
func (p *Time) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *Time) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *Time) expand(Context) Value { return p }

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//                └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//                     └<userinfo>─(@)┘      └(:)─<port>┘
type url struct{
    valbase
    Scheme Value
    Username Value
    Password Value
    Host Value
    Port Value
    Path Value
    Query []Value
    Fragment Value
}
func (_ *url) kind() Kind { return KindUrl }
func (p *url) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *url) ts(string) (s string) {
    s = "{=url"
    s += " " + ts(p.Scheme)
    s += " " + ts(p.Username)
    s += " " + ts(p.Password)
    s += " " + ts(p.Host)
    s += " " + ts(p.Port)
    s += " " + ts(p.Path)
    s += " " + ts(p.Query)
    s += " " + ts(p.Fragment)
    s += "}"
    return
}
func (p *url) String() (s string) {
    s = p.Scheme.String() + ":"
    if p.Host != nil {
        s += "//"
        if p.Username != nil {
            s += p.Username.String() + "@"
        }
        s += p.Host.String()
        if p.Port != nil {
            s += ":" + p.Port.String()
        }
    }
    if p.Path != nil {
        s += p.Path.String()
    }
    if p.Query != nil {
        s += "?"
        for i, q := range p.Query {
            if 0 < i { s += "&" }
            s += q.String()
        }
    }
    if p.Fragment != nil {
        s += "#" + p.Fragment.String()
    }
    return
}
func (p *url) string(ctx Context) (s string) {
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
    if p.Query != nil {
        s += "?"
        for i, q := range p.Query {
            if 0 < i { s += "&" }
            s += q.string(ctx)
        }
    }
    if p.Fragment != nil && !isNone(p.Fragment) {
        s += "#" + p.Fragment.string(ctx)
    }
    return
}
func (p *url) true(ctx Context) (t bool) {
    if p.Scheme != nil { if t = p.Scheme.true(ctx); t { return }}
    if p.Host   != nil { if t = p.Host  .true(ctx); t { return }}
    if p.Path   != nil { if t = p.Path  .true(ctx); t { return }}
    return //p.String() != "", nil
}
func (p *url) int(ctx Context) (i int64) {
    if s := p.string(ctx); s != "" { i = int64(len(s)) }
    return
}
func (p *url) float(ctx Context) (f float64) { return float64(p.int(ctx)) }
func (p *url) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(*url); y {
        if p.Scheme == nil || x.Scheme == nil { return }
        if p.Scheme.cmp(ctx, x.Scheme) != cmpEqual { return }
        if p.Username != nil {
            if x.Username == nil { return }
            if p.Username.cmp(ctx, x.Username) != cmpEqual { return }
        }
        if p.Password != nil {
            if x.Password == nil { return }
            if p.Password.cmp(ctx, x.Password) != cmpEqual { return }
        }
        if p.Host != nil {
            if x.Host == nil { return }
            if p.Host.cmp(ctx, x.Host) != cmpEqual { return }
        }
        if p.Port != nil {
            if x.Port == nil { return }
            if p.Port.cmp(ctx, x.Port) != cmpEqual { return }
        }
        if p.Path != nil {
            if x.Path == nil { return }
            if p.Path.cmp(ctx, x.Path) != cmpEqual { return }
        }
        if p.Query != nil {
            if len(x.Query) != len(p.Query) { return }
            for i, q := range p.Query {
                if q.cmp(ctx, x.Query[i]) != cmpEqual { return }
            }
        }
        if p.Fragment != nil {
            if x.Fragment == nil { return }
            if p.Fragment.cmp(ctx, x.Fragment) != cmpEqual { return }
        }
        res = cmpEqual
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}
func (p *url) expand(ctx Context) (res Value) {
    var o = &url{ valbase: p.valbase }
    if nil != p.Scheme   { o.Scheme   = p.Scheme.expand(ctx) }
    if nil != p.Username { o.Username = p.Username.expand(ctx) }
    if nil != p.Password { o.Password = p.Password.expand(ctx) }
    if nil != p.Host     { o.Host     = p.Host.expand(ctx) }
    if nil != p.Port     { o.Port     = p.Port.expand(ctx) }
    if nil != p.Path     { o.Path     = p.Path.expand(ctx) }
    if nil != p.Query    { o.Query    = expand(ctx, p.Query...) }
    if nil != p.Fragment { o.Fragment = p.Fragment.expand(ctx) }
    if  o.Scheme != p.Scheme ||
        o.Username != p.Username ||
        o.Password != p.Password ||
        o.Host != p.Host ||
        o.Port != p.Port ||
        o.Path != p.Path ||
        // o.Query != p.Query ||
        o.Fragment != p.Fragment {
        res = o
    } else {
        res = p
    }
    return
}
func (p *url) match(ctx Context, i any) (full bool, s any, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *url) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *url) Validate() (res *neturl.URL) {
    panic(fmt.Sprintf("validate %s", p))
    return
}

type raw struct{ valbase; s string }
func (_ *raw) kind() Kind { return KindRaw }
func (p *raw) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *raw) String() string { return /* "{=raw "+p.s+"}" */p.s }
func (p *raw) string(Context) string { return p.s }
func (p *raw) true(Context) bool { return p.s != "" }
func (p *raw) change(f func(string) string) Value { return &raw{p.valbase, f(p.s)} }
func (p *raw) expand(Context) Value { return p }
func (p *raw) int(ctx Context) (_ int64) {
    if i, e := strconv.ParseInt(p.s, 10, 64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return i
    }
}
func (p *raw) float(ctx Context) (_ float64) {
    if f, e := strconv.ParseFloat(p.s, 64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return f
    }
}
func (p *raw) trim(pre string) {
    p.s = strings.TrimSpace(strings.TrimPrefix(p.s, pre))
    return
}
func (p *raw) match(ctx Context, i any) (bool, any, []string) {
    return stringMatch(ctx, p, i)
}
func (p *raw) stencil(ctx Context, stems []string) (Value, []string) {
    return p, stems
}
func (p *raw) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    cmp := func(l, r string) cmpres {
        if l == r {
            return cmpEqual
        } else if l < r {
            if strings.HasPrefix(r, l) {
                return cmpLprefix
            } else {
                return cmpSmaller
            }
        } else {
            if strings.HasPrefix(l, r) {
                return cmpRprefix
            } else {
                return cmpGreater
            }
        }
    }
    switch x := v.(type) {
    case *raw: return cmp(p.s, x.s)
    case *word: return cmp(p.s, x.s)
    case *list: if x.len() == 1 { return p.cmp(ctx, x.elems[0]) }
    case *path: if x.len() == 1 { return p.cmp(ctx, x.elems[0]) }
    }
    return
}

type strlit struct{ valbase; s string }
func (_ *strlit) kind() Kind { return KindStrLit }
func (p *strlit) String() string { return `'`+p.s+`'` }
func (p *strlit) string(ctx Context) string { return p.s }
func (p *strlit) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *strlit) true(ctx Context) bool { return p.s != "" }
func (p *strlit) change(f func(string) string) Value { return &strlit{p.valbase, f(p.s)} }
func (p *strlit) expand(ctx Context) Value {
    if truly(ctx, ex_path_str{}) {
        return _pathstr(ctx, p.s)
    } else {
        return p
    }
}
func (p *strlit) int(ctx Context) (_ int64) {
    if i, e := strconv.ParseInt(p.s,10,64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return i
    }
}
func (p *strlit) float(ctx Context) (_ float64) {
    if f, e := strconv.ParseFloat(p.s, 64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return f
    }
}
func (p *strlit) ts(t string) string {
    return "{="+t+" "+strings.Replace(p.s, "\n", "\\n", -1)+"}"
}
func (p *strlit) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    switch t := v.(type) {
    case *strlit:
        if p.s == t.s {
            return cmpEqual
        } else if p.s < t.s {
            return cmpSmaller
        } else /*if p.s > t.s*/ {
            return cmpGreater
        }
    case *list:
        if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    }
    return
}
func (p *strlit) match(ctx Context, i any) (full bool, s any, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *strlit) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *strlit) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *strlit) hit(ctx Context, c *valcache) (res *valcache, full bool) {
    if c, full = c.hit(ctx, STRING); c == nil { return }
    return c.hit(ctx, p.s)
}

type strval struct{ valbase; v []Value }
func (_ *strval) kind() Kind { return KindStrVal }
func (p *strval) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *strval) ts(t string) string {
    var s string
    for i, v := range p.v {
        if 0 < i { s += " " }
        s += ts(v)
    }
    return fmt.Sprintf("{=%s %s}", t, s)
}
func (p *strval) String() (s string) {
    if expandable(original{nil,defExpand2}, p.v...) {
        for _, v := range p.v {
            if s != "" { s += " " }
            s += v.String()
        }
        s = `{=str `+s+`}`
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
    if truly(ctx, ex_path_str{}) {
        return _pathstr(ctx, p.string(ctx))
    } else if v := expand(ctx, p.v...); diff(ctx, v, p.v) {
        return &strval{p.valbase,v}
    } else {
        return p
    }
}
func (p *strval) es(ctx Context, f func(string)) {
    var v = p.expand(ctx)
    if !indeterminate(ctx, v) { f(v.string(ctx)) }
}
func (p *strval) true(ctx Context) (res bool) {
    p.es(ctx, func(s string) { res = s != "" })
    return
}
func (p *strval) int(ctx Context) (i int64) {
    var e error
    p.es(ctx, func(s string) { i, e = strconv.ParseInt(s, 10, 64) })
    if e != nil {
        erro(ctx, "%v", e).trace()
    }
    return
}
func (p *strval) float(ctx Context) (f float64) {
    var e error
    p.es(ctx, func(s string) { f, e = strconv.ParseFloat(s, 64) })
    if e != nil {
        erro(ctx, "%v", e).trace()
    }
    return
}
func (p *strval) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    switch t := v.(type) {
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
    return
}
func (p *strval) match(ctx Context, i any) (full bool, s any, stems []string) { return stringMatch(ctx, p, i) }
func (p *strval) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *strval) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *strval) _hit(ctx Context, c *valcache) (res *valcache, full bool) {
    if c, full = c.hit(ctx, STRING); c == nil { return }
    return c.hit(ctx, p.string(ctx))
}

func isTrueString(s string) (t bool) {
    switch strings.ToLower(s) {
    case "false", "no" , "off", "force_off", "0", "": t = false
    case "true" , "yes", "on" , "force_on" , "1"    : t = true
    default: t = true
    }
    return
}

type quoted struct{ list }
func (_ *quoted) kind() Kind { return KindQuoted }
func (q *quoted) hash(ctx Context) uint64 { return fnv1(ctx, q, q.any()) }
func (q *quoted) String() (s string) { return "{=quote "+q.list.String()+"}" }
func (q *quoted) string(ctx Context) (s string) {
    return strconv.Quote(q.list.string(ctx))
}
func (q *quoted) expand(ctx Context) Value {
    var a = expand(ctx, q.elems...)
    if diff(ctx, a, q.elems) {
        return &quoted{list{elements{a}}}
    } else {
        return q
    }
}
func (q *quoted) cmp(ctx Context, v Value) (res cmpres) {
    switch t := v.(type) {
    case *quoted:
        return compareElems(ctx, q.elems, t.elems)
    }
    return
}

// punct stands for the punctuation
type punct struct{ valbase; token }
func (_ *punct) kind() Kind { return KindPunct }
func (p *punct) String() string { return p.token.String() }
func (p *punct) string(Context) string { return p.token.String() }
func (p *punct) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *punct) true(Context) (_ bool) { return }
func (p *punct) int(Context) (_ int64) { return }
func (p *punct) float(Context) (_ float64) { return }
func (p *punct) expand(Context) Value { return p }
func (p *punct) ts(t string) (s string) {
    switch p.token {
    case PROOT: s = "root"
    case PTAIL: s = "tail"
    default:    s = p.token.String()
    }
    return "{="+t+" "+s+"}"
}
func (p *punct) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    switch x := v.(type) {
    case *punct:
        if p.token == x.token {
            res = cmpEqual
        } else if p.token > x.token {
            res = cmpSmaller
        } else if p.token < x.token {
            res = cmpGreater
        }
    case *list:
        if x.len() == 1 {
            return p.cmp(ctx, x.elems[0])
        }
    }
    return
}
func (p *punct) match(ctx Context, i any) (full bool, res any, stems []string) {
    var s string
    switch t := i.(type) {
    case   Value : s = t.string(ctx)
    case   string: s = t
    case []string: if len(t) == 1 { s = t[0] } else { return }
    default:
        erro(ctx, "%T: matching unsupported value: %T %v", p, i, i).trace()
    }
    if t := p.string(ctx); strings.HasPrefix(s, t) {
        full, res = len(s) == len(t), s[:len(t)]
        if false && s == ".h" { warn(ctx, "%v %v; %v %v", p, s, full, res).debug(6) }
    }
    return
}
func (p *punct) stencil(ctx Context, stems []string) (val Value, rest []string) { return p, stems }
func (p *punct) traverse(ctx Context) { }
func (p *punct) hit(ctx Context, c *valcache) (_ *valcache, _ bool) { return c.hit(ctx, p.token) }
func (p *punct) prefix(ctx Context, val Value) Value {
    if x, y := val.(*path); y {
        var i = x.len() - 1
        var v = &path{elements{x.elems}}
        if t, y := x.elems[i].(*punct); y && t.token == PTAIL {
            v.elems[i] = t
        } else {
            v.elems[0] = compose(ctx, p, v.elems[0])
        }
        return v
    }

    v := &compound{elements{}}
    if x, y := val.(*compound); y {
        v.elems = append(x.elems, p)
    } else {
        v.elems = append([]Value{val}, p)
    }
    return v
}
func (p *punct) suffix(ctx Context, val Value) Value {
    if x, y := val.(*path); y {
        var v = &path{elements{x.elems}}
        if t, y := x.elems[0].(*punct); y && t.token == PROOT {
            v.elems[0] = t
        } else {
            v.elems[0] = compose(ctx, p, v.elems[0])
        }
        return v
    }

    v := &compound{elements{}}
    if x, y := val.(*compound); y {
        v.elems = append([]Value{p}, x.elems...)
    } else {
        v.elems = []Value{p, val}
    }
    return v
}

type bare string
type word struct{ valbase; s string }
func (_ *word) kind() Kind { return KindWord }
func (p *word) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *word) true(Context) bool { return p.s != "" }
func (p *word) string(Context) string { return p.s }
func (p *word) String() string {
    if strings.Contains(p.s, " ") {
        return "{=word "+p.s+"}"
    } else {
        return p.s
    }
}
func (p *word) int(ctx Context) (_ int64) {
    if i, e := strconv.ParseInt(p.s, 10, 64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return i
    }
}
func (p *word) float(ctx Context) (_ float64) {
    if f, e := strconv.ParseFloat(p.s, 64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return f
    }
}
func (p *word) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    cmp := func(l, r string) cmpres {
        if l == r {
            return cmpEqual
        } else if l < r {
            if strings.HasPrefix(r, l) {
                return cmpLprefix
            } else {
                return cmpSmaller
            }
        } else {
            if strings.HasPrefix(l, r) {
                return cmpRprefix
            } else {
                return cmpGreater
            }
        }
    }
    switch x := v.(type) {
    case *raw: return cmp(p.s, x.s)
    case *word: return cmp(p.s, x.s)
    case *list: if x.len() == 1 { return p.cmp(ctx, x.elems[0]) }
    case *path: if x.len() == 1 { return p.cmp(ctx, x.elems[0]) }
    }
    return
}
func (p *word) prefix(ctx Context, v Value) Value { return _suffix(ctx, v, p) }
func (p *word) suffix(ctx Context, v Value) Value { return _bifix(ctx, p, v) }
func (p *word) stencil(ctx Context, stems []string) (_ Value, _ []string) { return p, stems }
func (p *word) match(ctx Context, i any) (_ bool, _ any, _ []string) { return stringMatch(ctx, p, i) }
func (p *word) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) { return c.hit(ctx, p.s) }
func (p *word) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *word) expand(ctx Context) Value { return p }

type qualword struct{ valbase; words []string } // TODO: foo.bar.zar, foo.&(bar).zar ???
func (p *qualword) String() string { return strings.Join(p.words,".") }
func (p *qualword) string(ctx Context) string { return p.String() }
func (p *qualword) hash(ctx Context) uint64 { return fnv1(ctx, nil, p) }
func (p *qualword) true(ctx Context) bool { return len(p.words)!=0 }
func (p *qualword) int(ctx Context) (_ int64) { return int64(len(p.words)) }
func (p *qualword) float(ctx Context) (_ float64) { return }
func (p *qualword) expand(Context) Value { return p }
func (p *qualword) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if a, y := v.(*qualword); y {
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
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}
func (p *qualword) match(ctx Context, i any) (full bool, s any, stems []string) {
    full, s, stems = stringMatch(ctx, p, i)
    return
}
func (p *qualword) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}
func (p *qualword) traverse(ctx Context) { do(ctx, act_traverse{p}) }

type elements struct{ elems []Value }
func (p *elements) Position() Position { return p.elems[0].Position() }
func (p *elements) ident(Context) (_ string) { return } // aka name
func (p *elements) list() *list { return &list{*p} }
func (p *elements) path() *path { return &path{*p} }
func (p *elements) compound() *compound { return &compound{*p} }
func (p *elements) strcomp() *strcomp { return &strcomp{*p} }
func (p *elements) len() int { return len(p.elems) }
func (p *elements) append(v ...Value) { p.elems = append(p.elems, v...) }
func (p *elements) at(n int) (v Value) {
    if 0 <= n && n < len(p.elems) { v = p.elems[n] }
    return
}
func (p *elements) any() (a []any) {
    for _, v := range p.elems { a = append(a, v) }
    return
}
func (p *elements) cond() (_ bool) {
    for _, elem := range p.elems {
        if elem.cond() { return true }
    }
    return
}
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
            erro(ctx, "nil element #%d: %v", i, p).trace()
        } else if res = elem.expandable(ctx); res { break }
    }
    return
}
func (p *elements) ts(t string) (s string) {
    s = "{="+t
    for _, a := range p.elems { s += " "+ts(a) }
    s += "}"
    return
}
func (_ *elements) delete(Context) (_ []*file) { return }
func (_ *elements) patterned(Context) (_ bool) { return }
func (_ *elements) stamp(Context) (_ []*file) { return }
func (_ *elements) stat(Context) (_ *statinfo) { return }
func (_ *elements) updated(Context) (_ bool) { return }
func (_ *elements) updatedDeps(Context, ...Value) (_ []Value) { return }
func (_ *elements) traverse(Context) {}

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

func _cond(v Value) (y bool) {
    switch t := v.(type) {
    case cond: return true
    case flag: return _cond(t.Value)
    case disjunction: return _cond(t.Value)
    case *compound:
        for _, e := range t.elems {
            if _cond(e) { return true }
        }
    }
    return
}

func condish(ctx Context, v Value) Value {
    var x, y = v.(cond)
    if checkpoints && truly(ctx, is_test_mode{}) {
        if _cond(x.Value) {
            errostack(pc(ctx,v), 3, "nested cond, (y=%v): %v : %v", y, v, ts(v)).trace()
        }
    }
    if truly(ctx, ex_condless{}) {
        if y {
            return x.Value // TODO: condless(x.Value)
        } else {
            return v
        }
    } else if y {
        return x
    } else {
        return cond{v} // condish
    }
}

type cond struct{ Value } // conditional component: compound, pair; aka optional
func (p cond) cond() bool { return true }
func (p cond) kind() Kind { return p.Value.kind()|KindCond }
func (p cond) hash(ctx Context) uint64 { return fnv1(ctx, nil, p.kind(), p.Value) }
func (p cond) String() string { return p.Value.String()+"?" }
func (p cond) string(ctx Context) (s string) {
    p.final(ctx, func(v Value) { s = v.string(ctx) })
    return
}
func (p cond) true(ctx Context) (t bool) {
    p.final(ctx, func(v Value) { t = v.true(ctx) })
    return
}
func (p cond) int(ctx Context) (i int64) {
    p.final(ctx, func(v Value) { i = v.int(ctx) })
    return
}
func (p cond) float(ctx Context) (f float64) {
    p.final(ctx, func(v Value) { f = v.float(ctx) })
    return
}
func (p cond) final(ctx Context, f func(Value)) {
    if t := p.expand(_final(ctx)); t != nil && !equal(ctx, p, t) { f(t) }
}
func (p cond) expand(ctx Context) (res Value) {
    var v = p.Value.expand(condless{ctx})

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.expand_check(ctx, v, &res)
    }

    if v == nil {
        return _null(p.Position())
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
func (p cond) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    switch x := v.(type) {
    case cond: return p.Value.cmp(ctx, x.Value)
    case *list: if x.len() == 1 { return p.cmp(ctx, x.elems[0]) }
    }
    return
}

type conjunction struct{ *list ; sep Value }
func (p conjunction) hash(ctx Context) uint64 {
    return fnv1(nil, p.kind(), p.sep, p.list)
}
func (p conjunction) String() string {
    var s string
    if p.sep != nil { s = p.sep.String() }
    return "{{"+p.list.String()+"}"+s+"}"
}
func (p conjunction) string(ctx Context) (s string) {
    var sep string
    if p.sep != nil { sep = p.sep.string(ctx) }

    var ss []string
    var elems = expand(_final(ctx), p.elems...)
    for _,  elem := range elems {
        if !indeterminate(ctx, elem) {
            ss = append(ss, elem.string(ctx))
        }
    }
    return strings.Join(ss, sep)
}
func (p conjunction) ts(t string) string {
    return fmt.Sprintf("{=%s {%v}%v}", t, ts(p.list), ts(p.sep))
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

type disjunction struct{ Value }
func (p disjunction) hash(ctx Context) uint64 { return fnv1(ctx, nil, p.kind(), p.Value) }
func (p disjunction) ts(t string) string { return "{="+t+" "+ts(p.Value)+"}" }
func (p disjunction) String() string { return "{"+p.Value.String()+"}" }
func (p disjunction) expand(ctx Context) (res Value) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.expand_check(ctx, &res) }

    var v = p.Value.expand(ctx)

    if x, y := v.(*list); y {
    xlist:
        switch x.len() {
        case 0: return x // FIXME: _null(x.Position())
        case 1:
            v = x.elems[0]
            if x, y = v.(*list); y { goto xlist }
            return v
        default:
            return disjunction{x}
        }
    } else if indeterminate(ctx, v) {
        return disjunction{v} // doesn't matter equal(ctx, v, p.Value)
    } else {
        return v
    }
}
func (p disjunction) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(disjunction); y {
        return p.Value.cmp(ctx, x.Value)
    } else {
        return
    }
}

type wants_fullfile  struct{}
type is_compound     struct{}
type    compound_ctx struct{ Context }
func (c compound_ctx) cast(t reflect.Type) Context { return icast(c, t) }
func (c compound_ctx) inner() Context { return c.Context }
func (c compound_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case ex_condless: return true
    case is_compound: return true
    case wants_fullfile: return false
    }
    return c.Context.do(ctx, op)
}

type compound struct{ elements }
func (_ *compound) kind() Kind { return KindCompound }
func (p *compound) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()...) }
func (p *compound) String() (s string) {
    for _, elem := range p.elems {
        if x, y := elem.(*list); y && x.len() > 1 && false {
            s += "{"+elem.String()+"}"
        } else {
            s += elem.String()
        }
    }
    return
}
func (p *compound) string(ctx Context) (s string) {
    var sc = stringify_ctx{ctx, false}
    var v = p.expand(&sc)
    if sc.empty {
        return
    } else if _cond(v) && indeterminate(ctx, v) {
        return
    } else if equal(ctx, p, v) {
        if o, y := v.(*compound); y {
            for _, elem := range o.elems {
                if _, y := elem.(fullfile); y {
                    errostack(pc(ctx,elem), 8, "unexpected fullfile: %v ; %v ; %v", ts(elem), ts(v), ts(p)).trace()
                } else {
                    s += elem.string(ctx)
                }
            }
        } else {
            errostack(pc(ctx,p), 8, "unexpected compound: %v ; %v", ts(v), ts(p)).trace()
        }
        return
    } else {
        if checkpoints && truly(ctx, is_test_mode{}) {
            if eq(ctx, p, v) {
                errostack(pc(ctx,p), 3, "%s == %s", ts(p), ts(v)).trace()
            }
        }
        return v.string(ctx)
    }
}
func (p *compound) true(ctx Context) bool { return p.elements.true(ctx) }
func (p *compound) float(ctx Context) (_ float64) { return }
func (p *compound) int(ctx Context) (res int64) {
    if n := len(p.elems); n > 0 {
        if i, y := p.elems[0].(*decimal); y {
            switch n {
            case 1: res = i.int64
            case 2:
                if w, y := p.elems[1].(*word); y {
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
func (p *compound) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *compound) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *compound) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *compound) expandable(ctx Context) bool { return p.elements.expandable(ctx) }
func (p *compound) expand(ctx Context) (res Value) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.expand_check(ctx, &res) }

    var f func(_, _ []Value) []Value

    f = func(a, elems []Value) (rs []Value) {
        var c = a != nil && indeterminate(ctx, a...)
        for i, val := range elems {
            if val == nil {
                erro(ctx, "%v: nil component #%d; %v : %v", p, val, ts(val), i).trace()
            }

            if !c && _cond(val) { c = true }

            var v = val.expand(compound_ctx{ctx})
            if v == nil { continue }

        check_v_type:
            switch x := v.(type) {
            case disjunction:
                if indeterminate(ctx, x.Value) {
                    do(ctx, stringify_empty{})
                    if x.Value.String() == "&(.test.h)a &(.test.h)b &(.test.h)c" {
                        note(pc(ctx,x), "%v: %v %v; %v", p, a, x.Value, auto_get(ctx,"1")).debug(16)
                    }
                } else {
                    var tail = elems[i+1:]
                    for _, v := range merge(x.Value) {
                        rs = append(rs, f(append(a, v), tail)...)
                    }
                    return
                }
            case *list:
                if n, es := x.len(), x.elems; n == 0 {
                    continue
                } else {
                    if a == nil {
                        rs = append(rs, es[:n-1]...)
                    } else {
                        var v Value = &compound{elements{append(a, es[0])}}
                        if c { v = condish(ctx, v) }
                        if rs = append(rs, v); n == 1 { return }
                        if 2 < n { rs = append(rs, es[1:n-1]...) }
                        a = nil
                    }

                    v = es[n-1]
                    goto check_v_type
                }
            }

            a = append(a, v)
        }
        switch len(a) {
        case 0: return
        case 1: //return append(rs, a[0])
        }

        // NOTE: copyvals avoids yielding constant records
        var v Value = &compound{elements{copyvals(a)}}
        if c { v = condish(ctx, v) }
        return append(rs, v)
    }
    return ease(ctx, f(nil, p.elems))
}
func (p *compound) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    if false && indeterminate(ctx, p) {
        erro(ctx, "indeterminate : %v", tv(p)).trace()
    }

    if x, y := do(ctx, hit_bare{c, p}).(valcache_bool); y {
        return x.valcache, x.bool
    }

    errostack(pc(ctx,p), 3, "miss hit : %v : %v", ts(p), ts(ctx)).trace()
    return
}
func (p *compound) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }

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

    if a, y := v.(*compound); y {
        if p == a {
            return cmpEqual
        } else {
            cmp(baremerge(p.elems...), baremerge(a.elems...))
            return
        }
    } else if w, y := v.(*word); y {
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
        if len(elems) == 2 {
            if fL, y := elems[0].(flag); y {
                if isNull(fL.Value) || isNone(fL.Value) {
                    res = elems[1].cmp(ctx, fR.Value)
                } else if m, r, t := fL.Value.match(ctx, fR.Value); m {
                    if isNull(elems[1]) || isNone(elems[1]) { res = cmpEqual }
                } else if r != nil { // matched prefix
                    var s = _joinpath(ctx, r)
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
            }
        }
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
func (p *compound) patterned(ctx Context) (res bool) {
    for _, elem := range p.elems {
        if res = elem.patterned(ctx); res { return }
    }
    return
}
func (p *compound) match(ctx Context, i any) (full bool, res any, stems []string) {
    var n int
    var s string
    var elem Value
    switch t := i.(type) {
    case   Value : s = t.string(ctx)
    case   string: s = t
    case []string: if len(t) == 1 { s = t[0] } else { return }
    default:
        errostack(pc(ctx,p), 3, "matching unsupported value: %v", tv(i)).trace()
    }
    if s == "" { return }

    var rs string
    for n, elem = range p.elems {
        var _, r, ss = elem.match(ctx, s)
        var t = _joinpath(ctx, r)
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
func (p *compound) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var elems []Value
    var changed int
    rest = stems
    for _, elem := range p.elems {
        var t Value
        if t, rest = elem.stencil(ctx, rest); t != elem {
            changed += 1
        }
        elems = append(elems, t)
    }
    if changed > 0 {
        val = _compound(elems...)
    } else {
        val = p
    }
    return
}
func (p *compound) prefix(ctx Context, val Value) (_ Value) {
    switch x := val.(type) {
    case *globpat:
        t := &globpat{p.elements}
        t.elems = append(x.elems, p.elems...)
        return t
    case *compound:
        t := &compound{p.elements}
        t.elems = append(x.elems, p.elems...)
        return t
    default:
        t := &compound{p.elements}
        t.elems = append([]Value{val}, p.elems...)
        return t
    }
}
func (p *compound) suffix(ctx Context, val Value) (_ Value) {
    switch x := val.(type) {
    case *globpat:
        t := &globpat{p.elements}
        t.elems = append(p.elems, x.elems...)
        return t
    case *compound:
        t := &compound{p.elements}
        t.elems = append(p.elems, x.elems...)
        return t
    default:
        t := &compound{p.elements}
        t.elems = append(p.elems, val)
        return t
    }
}
func (p *compound) app(vals ...Value) {
    for _, v := range vals {
        if x, y := v.(*compound); y {
            p.app(x.elems...)
        } else {
            p.elems = append(p.elems, v)
        }
    }
}

type prefix_value struct{ Value; prefix Value }
func (p prefix_value) expand(ctx Context) (_ Value) {
    return prefix_value{p.Value.expand(ctx), p.prefix.expand(ctx)}
}
func (p prefix_value) string(ctx Context) (_ string) {
    if s := p.Value.string(ctx); s != "" {
        return p.prefix.string(ctx) + s
    }
    return
}

type value_suffix struct{ Value; suffix Value }
func (p value_suffix) expand(ctx Context) (_ Value) {
    return value_suffix{p.suffix.expand(ctx), p.Value.expand(ctx)}
}
func (p value_suffix) string(ctx Context) (_ string) {
    if s := p.Value.string(ctx); s != "" {
        return s + p.suffix.string(ctx)
    }
    return
}

type recipe struct{ strcomp }
func (_ *recipe) kind() Kind { return KindRecipe }
func (p *recipe) String() string { return p.src() }
func (p *recipe) expand(ctx Context) (_ Value) {
    if x, y := p.strcomp.expand(ctx).(*strcomp); y {
        return &recipe{strcomp{x.elements}}
    } else {
        errostack(pc(ctx,p), 3, "recipe.expand: %v", tv(x)).trace()
    }
    return
}
func (p *recipe) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(*recipe); y {
        return p.strcomp.cmp(ctx, &x.strcomp)
    } else {
        return p.strcomp.cmp(ctx, v)
    }
}

// barefile reduces file lookups, it works like an alias of a File.
type barefile struct{
    Value
    file *file
}
func (_ *barefile) kind() Kind { return KindBarefile }
func (p *barefile) hash(ctx Context) uint64 { return fnv1(ctx, nil, p.kind(), p.Value) }
func (p *barefile) string(ctx Context) string {
    if p.file != nil {
        return p.file.string(ctx)
    } else {
        return p.Value.string(ctx)
    }
}
func (p *barefile) true(ctx Context) (t bool) {
    if p.file != nil { t = p.file.true(ctx) }
    return
}
func (p *barefile) int(ctx Context) (res int64) {
    if p.file.exists() { res = p.file.info.Size() }
    return
}
func (p *barefile) float(ctx Context) (f float64) {
    return float64(p.int(ctx))
}
func (p *barefile) refs(ctx Context, v Value) bool { return p.Value.refs(ctx, v) }
func (p *barefile) defs(ctx Context, s ...string) []*def { return p.Value.defs(ctx, s...) }
func (p *barefile) expandable(ctx Context) bool { return p.Value.expandable(ctx) }
func (p *barefile) expand(ctx Context) (res Value) {
    if truly(ctx, wants_fullfile{}) {
        var f = p.file
        if f == nil { f = findfile(ctx, p.Value.string(ctx)) }
        if f != nil { if v := f.expand(ctx); v != f { return v }}
    }

    if name := p.Value.expand(ctx); name != p.Value {
        res = &barefile{name, p.file}
    } else {
        res = p
    }
    return
}
func (p *barefile) traverse(ctx Context) {
    if p.file != nil { p.file.traverse(ctx) } else
    if p.Value != nil { p.Value.traverse(ctx) }
}
func (p *barefile) updated(ctx Context) (res bool) {
    if p.file != nil { res = p.file.updated(ctx) }
    return
}
func (p *barefile) updatedDeps(ctx Context, v ...Value) (res []Value) {
    if p.file != nil { res = p.file.updatedDeps(ctx, v...) }
    return
}
func (p *barefile) delete(ctx Context) (_ []*file) {
    if p.file != nil { return p.file.delete(ctx) }
    return
}
func (p *barefile) stamp(ctx Context) (_ []*file) {
    if p.file != nil { return p.file.stamp(ctx) }
    return
}
func (p *barefile) stat(ctx Context) (_ *statinfo) {
    if p.file != nil { return p.file.stat(ctx) }
    return
}
func (p *barefile) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(*barefile); y {
        return p.Value.cmp(ctx, x.Value)
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}
func (p *barefile) patterned(ctx Context) bool { return p.Value.patterned(ctx) }
func (p *barefile) match(ctx Context, i any) (full bool, s any, stems []string) {
    if false && p.file != nil {
        full, s, stems = p.file.match(ctx, i)
    } else {
        full, s, stems = p.Value.match(ctx, i)
    }
    return
}
func (p *barefile) stencil(ctx Context, stems []string) (val Value, rest []string) {
    var name Value
    if p.file != nil {
        val, rest = p.file.stencil(ctx, stems)
    } else if name, rest = p.Value.stencil(ctx, stems); name != p.Value {
        val = &barefile{name, p.file}
    } else {
        val = p
    }
    return
}

func barefilize(ctx Context, targets ...Value) []Value {
    var project = _project(ctx)
    for i, target := range targets {
        if target.patterned(ctx) { continue }
        switch t := target.(type) {
        case *word:
            if file := project.file(ctx, t.s); file != nil {
                targets[i] = &barefile{ target, file }
                file.position = target.Position()
            }
        case *compound, *path:
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
    var maps []filemap_name
    for _, target := range targets {
        if !target.patterned(ctx) {
            maps = append(maps, unmap_files(ctx, target)...)
        }
    }
    for _, f := range select_files(ctx, maps) { res = append(res, f) }
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

type globmeta struct{ valbase ; token }
func (p *globmeta) hash(ctx Context) uint64 { return fnv1(ctx, nil, p.kind(), p.token) }
func (p *globmeta) String() string { return p.token.String() }
func (p *globmeta) string(ctx Context) string { return p.token.String() }
func (p *globmeta) expand(Context) Value { return p }
func (p *globmeta) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(*globmeta); y {
        if p.token == x.token { return cmpEqual }
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}
func (p *globmeta) hit(ctx Context, fc *valcache) (_ *valcache, _ bool) {
    return fc.hit(ctx, p.token)
}

// `[a-b]`, `[abc]`, ...
// `a-b`, `abc`, `a$(var)c`, `a$(spaces)c`...
type globrange struct{ Value }
func (p *globrange) hash(ctx Context) uint64 { return fnv1(ctx, nil, p.kind(), p.Value) }
func (p *globrange) String() string { return "["+p.Value.String()+"]" }
func (p *globrange) string(ctx Context) string { return "["+p.Value.string(ctx)+"]" }
func (p *globrange) refs(ctx Context, v Value) bool { return p.Value.refs(ctx, v) }
func (p *globrange) defs(ctx Context, s ...string) []*def { return p.Value.defs(ctx, s...) }
func (p *globrange) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(*globrange); y {
        return p.Value.cmp(ctx, x.Value)
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}
func (p *globrange) expandable(ctx Context) bool { return p.Value.expandable(ctx) }
func (p *globrange) expand(ctx Context) Value {
    if val := p.Value.expand(ctx); val != p.Value {
        return &globrange{val}
    } else {
        return p
    }
}

type path struct{ elements }
func (_ *path) kind() Kind { return KindPath }
func (p *path) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()...) }
func (p *path) String() (s string) {
    var n int
    for i, elem := range p.elems {
        if t := elem.String(); 0 < i {
            s += pathSep + t
            n += 1
        } else if t != "" {
            s += t
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
            erro(ctx, "`%s` nil path segment", p).trace()
        }

        var v string
        if isUndef(seg) {
            erro(ctx, "undef path segment (%T)", seg)
            erro(ctx, "… from this context: %s", ctx).trace()
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
    if x, y := p.elems[0].(*punct); y && x.token == PROOT {
        return true
    }
    return
}
func (p *path) float(ctx Context) (_ float64) { return }
func (p *path) int(ctx Context) (_ int64) { return }
func (p *path) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *path) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *path) expandable(ctx Context) bool { return p.elements.expandable(ctx) }
func (p *path) expand(ctx Context) Value {
    if elems := expand_path_elems(ctx, p.elems...); diff(ctx, elems, p.elems) {
        return &path{elements{elems}}
    } else {
        return p
    }
}
func (p *path) delete(ctx Context) (_ []*file) {
    var si = p.stat(ctx)
    if si == nil || si.file == nil {
        erro(ctx, "no path name for `%s`", p).trace()
    }
    return si.file.delete(ctx)
}
func (p *path) stamp(ctx Context) (files []*file) {
    var si = p.stat(ctx)
    if si == nil || si.file == nil {
        erro(ctx, "no path name for `%s`", p).trace()
    }
    return si.file.stamp(ctx)
}
func (p *path) stat(ctx Context) (si *statinfo) {
    var s string
    if p.patterned(ctx) {
        if val, rest := p.stencil(ctx, _stems(ctx)); len(rest) > 0 {
            erro(ctx, "partial match: %v", rest)
        } else {
            s = val.string(ctx)
        }
    } else {
        s = p.string(ctx)
    }

    if filepath.IsAbs(s) {
        if f := stat(ctx, s, stat_nonexist{true}); f != nil {
            return &statinfo{ file:f }
        }
    }

    if f := findfile(ctx, s); f != nil { si = f.stat(ctx) }
    return
}
func (p *path) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *path) patterned(ctx Context) (result bool) {
    for _, seg := range p.elems {
        if result = seg.patterned(ctx); result { break }
    }
    return
}
func (p *path) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if x, y := v.(*path); y {
        return compareElems(ctx, p.elems, x.elems)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    // } else if o, y := v.(fullname); y && o.Value != nil {
    //     if res = o.cmp(ctx, p); res != cmpUnknown && res != cmpEqual {
    //         res = cmpres(-res)
    //     }
    //     return
    // } else if f, y := v.(*file); y {
    //     if s := p.string(ctx); f.ident(ctx) == s { return cmpEqual }
    //     return
    } else if len(p.elems) == 1 {
        return p.elems[0].cmp(ctx, v)
    }
    return
}
func (p *path) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    if false && indeterminate(ctx, p) {
        erro(ctx, "%v : indeterminate : %v", p, ts(p)).trace()
    }

    if x, y := do(ctx, hit_path{c, p}).(valcache_bool); y {
        return x.valcache, x.bool
    }

    erro(ctx, "miss hit: %v : %v", ts(p), ts(ctx)).trace()
    return
}

func matchPathSeg(ctx Context, seg Value, src string) (bool, string, []string) {
    var full, i, ss = seg.match(ctx, src)
    var s = _joinpath(ctx, i)
    // if !full {
    //     if x, y := seg.(*punct); y && PTAIL == x.token && numSrc < lenSrcs {
    //         // ...
    //     }
    // }
    return full, s, ss
}

func (p *path) match2(ctx Context, srcs ...string) (full bool, res, stems []string) {
    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.match2_check(ctx, srcs, &full, &res, &stems)
    }

    if len(srcs) == 0 {
        if false { erro(ctx, "empty: %v", srcs) }
        return
    }

    var segs = expand_path_elems(ctx, p.elems...)
    if expandable(_final(ctx), segs...) {
        if false { erro(ctx, "expandable path: %v, %v", p, segs).trace() }
        return
    }

    var (
        lenSegs = len(segs)
        lenSrcs = len(srcs)
        nxtSeg, nxtSrc int
        undone func() bool
        app func([]string, ...string) []string
        tail func(token) bool
        reverse bool
        step int
    )
    if reverse = truly(ctx, propReversal); reverse {
        app = func(a []string, s ...string) []string { return append(s, a...) }
        tail = func(t token) bool { return PROOT == t && 0 <= nxtSrc }
        undone = func() bool { return 0 <= nxtSeg && 0 <= nxtSrc }
        nxtSeg, nxtSrc, step = lenSegs-1, lenSrcs-1, -1
    } else {
        app = func(a []string, s ...string) []string { return append(a, s...) }
        tail = func(t token) bool { return PTAIL == t && nxtSrc <= lenSrcs }
        undone = func() bool { return nxtSeg < lenSegs && nxtSrc < lenSrcs }
        step = 1
    }

    for undone() {
        var seg = segs[nxtSeg]; nxtSeg += step // move to the next seg
        if s := correctPunctForMatch(seg); s == nil {
            erro(ctx, "invalid path segment: %v", tv(seg)).trace()
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
                                stems = app(stems, joinpath(stem...))
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
                        if len(stem) > 0 { stems = app(stems, joinpath(stem...)) }
                        stem = []string{ strings.TrimPrefix(src, prefix) }
                        nxtSrc += step
                        break
                    } else {
                        stem = app(stem, src)
                    }
                } else if y, s, ss := matchPathSeg(ctx, nxt, src); y || s == src {
                    if res = app(res, s); len(ss) == 0 {
                        if false { stem = app(stem, src) }
                        if false { note(ctx, "%v %v , %v %v", p, stem, nxt, src) }
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

            if len(stem) > 0 { stems = app(stems, joinpath(stem...)) }
            if len(tail) > 0 { stems = app(stems, tail...) }
        } else if y, s, ss := matchPathSeg(ctx, seg, src); y {
            res = app(res, s)
            stems = app(stems, ss...)
            full = nxtSeg == lenSegs && nxtSrc == lenSrcs
        } else if x, y := seg.(*punct); y && tail(x.token) {
            res = app(res, "")
            return
        } else {
            if checkpoints && truly(ctx, is_test_mode{}) {
                if seg.string(ctx) == src {
                    erro(ctx, "%v : %s == %s, (%d,%d) ; %v ; %s %s", p, ts(seg), ts(src), nxtSeg, nxtSrc, srcs, s, ss).trace()
                }
                if !reverse && y && x.token == PTAIL && nxtSeg == lenSegs && nxtSrc <= lenSrcs { // checks for PTAIL, aka /
                    erro(ctx, "seg: %v %v %v/%v; src: %v %v %v/%v", p, ts(seg), nxtSeg, lenSegs, srcs, ts(src), nxtSrc, lenSrcs).trace()
                }
                if false && y {
                    note(ctx, "%v: %d. %v , %d. %v ; %v %v", p, nxtSeg, ts(seg), nxtSrc, ts(src), s, ss).debug()
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
func (p *path) match(ctx Context, i any) (full bool, res any, stems []string) {
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
    case *file:
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
        erro(ctx, "path.match unsupport %v", tv(i)).trace()
    }
    return
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

func (p *path) _suffix(ctx Context, val Value) (res Value) {
    if isTrivial(val) {
        erro(ctx, "path combines invalid value: %v", val).trace()
    }
    if _, y := p.elems[0].(*punct); y /* && t.token == PLUS */ {
        note(ctx, "%v %v", p, val).debug(10)
    }

    var ti = p.len()-1
    var tv = p.elems[ti]
    if tv == nil {
        erro(ctx, "path has nil tail").trace()
    }

    var comp *compound

    switch x := tv.(type) {
    case *punct:
        switch x.token {
        case PCON,0: comp = _compound()
        default: comp = _compound(tv)
        }
    case *compound:
        comp = x
    default:
        if isTrivial(tv) {
            comp = _compound()
        } else {
            comp = _compound(tv)
        }
    }

    p = &path{elements{append(p.elems[:ti], comp)}}

    if x, y := val.(*path); y {
        if t, y := x.elems[0].(*punct); y {
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

func (p *path) _prefix(ctx Context, val Value) (_ Value) {
    v := &path{p.elements}
    switch x := val.(type) {
    case *path:
        i := x.len() - 1
        v.elems[0] = compose(ctx, x.elems[i], v.elems[0])
        v.elems = append(x.elems[:i], v.elems...)
        return v
    default:
        v.elems[0] = compose(ctx, val, v.elems[0])
        return v
    }
}

func (p *path) suffix(ctx Context, val Value) (_ Value) {
    i := p.len() - 1
    v := &path{p.elements}
    switch x := val.(type) {
    case *path:
        v.elems[i] = compose(ctx, v.elems[i], x.elems[0])
        v.elems = append(v.elems, x.elems[1:]...)
        return v
    default:
        if checkpoints && truly(ctx, is_test_mode{}) {
            switch x := x.(type) {
            case *globpat:
                for i, e := range x.elems {
                    if _, y := e.(*path); y {
                        e = v.elems[i]
                        t := compose(ctx, e, val)
                        erro(ctx, "%d %s:%v,%s:%v → %s:%v", i, typeof(e), e, typeof(val), val, typeof(t), t).trace()
                    }
                }
            }
        }
        v.elems[i] = compose(ctx, v.elems[i], val)
        return v
    }
}

func _pathstr(ctx Context, str string) *path {
    return makePath(splitPathStr(ctx, str)...)
}

func _punct(ctx Context, tok token) *punct {
    return &punct{valbase{_position(ctx)}, tok}
}

func to_file(v Value) (x *file, y bool) {
    if x, y = v.(*file); !y {
        switch t := v.(type) {
        case       as: x, y = to_file(t.Value)
        case fullname: x, y = to_file(t.Value)
        case fullfile: x, y = t.file, true
        case *list: if t.len() == 1 { x, y = to_file(t.elems[0]) }
        }
    }
    return
}

func splitFileName(ctx Context, val Value) (dir, name string) {
    if f, _ := to_file(val); f != nil {
        dir, name = filepath.Join(f.dir, f.sub), f.ident(ctx)
    } else {
        name = val.string(ctx)
        dir = filepath.Dir(name)
    }
    return
}


type fullfile_ctx struct{ Context }
func (c fullfile_ctx) cast(t reflect.Type) Context { return icast(c, t) }
func (c fullfile_ctx) inner() Context { return c.Context }
func (c fullfile_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case wants_fullfile: return true
    }
    return c.Context.do(ctx, op)
}

type fullfile struct{ *file }
func (o fullfile) string(Context) string { return o.fullname() }
func (o fullfile) ts(string) string { return "{=fullfile "+o.filestub.name+"}" }
func (o fullfile) expand(ctx Context) Value {
    if truly(ctx, is_compound{}) { return o.file }
    return o
}

func try_fullfile(ctx Context, f *file) Value {
    // if !truly(ctx, is_compound{}) && truly(ctx, is_exec{})
    if truly(ctx, wants_fullfile{}) {
        return fullfile{f}
    }
    return f
}

type fullname struct{ Value }
func (o fullname) hash(ctx Context) uint64 { return fnv1(ctx, o, o.Value) }
func (o fullname) ts(string) string { return "{=fullname "+ts(o.Value)+"}" }
func (o fullname) expand(ctx Context) Value {
    if truly(ctx, is_compound{}) { return o.Value }
    return fullname{o.Value.expand(ctx)}
}
func (o fullname) String() string {
    if v := o.Value; v == nil {
        panic("nil file value")
    } else if x, y := v.(*file); y {
        if x == nil {
            panic("nil file")
        } else if x.filestub == nil {
            panic("nil file stub")
        }
    }
    return o.Value.String()
}
func (o fullname) string(ctx Context) (_ string) {
    if v := o.Value; v != nil {
        if x, y := v.(*file); y {
            if x == nil {
                erro(ctx, "nil file").trace()
            } else if x.filestub == nil {
                erro(ctx, "nil file stub").trace()
            }
            return x.fullname()
        }
        return v.string(ctx)
    }
    return
}
func (o fullname) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer o.cmp_check(ctx, v, &res) }
    switch t := v.(type) {
    case *list:
        if t.len() == 1 { return o.cmp(ctx, t.elems[0]) }
    case *file:
        if x, y := o.Value.(*file); y {
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
        if x, y := o.Value.(*file); y {
            if t.isAbs() && !indeterminate(ctx, t) {
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

type must_file_stamp struct{ *file }

type must_files_stamp struct{ Context }
func (c must_files_stamp) cast(t reflect.Type) Context { return icast(c,t) }
func (c must_files_stamp) inner() Context { return c.Context }
func (c must_files_stamp) do(ctx Context, op any) any {
    switch op.(type) {
    case must_file_stamp: return true
    }
    return c.Context.do(ctx, op)
}

type file struct{
    valbase
    *filebase
    *filestub
}
func (p *file) hash(ctx Context) uint64 { return fnv1(ctx, p, p.fullname()) }
func (p *file) ts(string) string { return "{=file "+p.filestub.name+"}" }
func (p *file) String() string { return "{=file "+p.filestub.name+"}" }
func (p *file) string(Context) string { return p.filestub.name }
func (p *file) ident(Context) string { return p.filestub.name }
func (p *file) true(Context) (_ bool) {
    if p.filestub.name != "" { return true }
    return
}
func (p *file) fullname() string {
    return filepath.Join(p.dir, p.sub, p.filestub.name)
}
func (p *file) basename() (s string) {
    if p.info != nil { return p.info.Name() }
    return filepath.Base(p.filestub.name)
}
func (p *file) hit(ctx Context, c *valcache) (_ *valcache, _ bool) {
    if ss := strings.Split(p.name, pathSep) ; true {
        return c.hit(ctx, ss)
    } else if len(ss) == 1 {
        return c.hit(ctx, p.name)
    } else {
        return unmap_path(ctx, c, p.name, ss)
    }
}
func (p *file) searchInMatchedPaths(ctx Context, proj *project) (res bool) {
    if p.filemap != nil {
        // FIXME: file should keep both 'match' and 'pre', or just remove searchInMatchedPaths
        var f = p.filemap.stat(ctx, p.filestub.name)
        if f != nil && f.info != nil { p.info, res = f.info, true }
    }
    return
}
func (p *file) stamp(ctx Context) (res []*file) {
    var fn = p.fullname()
    if fn == "" {
        erro(ctx, "file `%s` has no fullname", p).trace()
    }

    var e error

    if p.info, e = os.Stat(fn); e != nil {
        if truly(ctx, must_file_stamp{p}) {
            ctx = pc(pc(ctx, strings.TrimSuffix(fn, ".x")), fn+".log")
            if _, y := e.(*fs.PathError); y {
                errostack(ctx, 8, "no such file: %s", p.name).trace()
            } else {
                errostack(ctx, 8, "%v", e).trace()
            }
        }
        return
    } else if p.info == nil {
        if truly(ctx, must_file_stamp{p}) {
            ctx = pc(pc(ctx, strings.TrimSuffix(fn, ".x")), fn+".log")
            errostack(ctx, 8, "no such file: %s", p.name).trace()
        }
        return
    }

    res = append(res, p)

    if !is_configurecontext(ctx) {
        p._updated = true
        do(ctx, mark_dirty{[]Value{p}})
    }
    return
}
func (p *file) expandable(ctx Context) bool {
    return false//BUG: truly(ctx, fullfile{}) && !filepath.IsAbs(p.filestub.name)
}
func (p *file) expand(ctx Context) Value {
    if truly(ctx, wants_fullfile{}) {
        return fullfile{p}
    } else {
        return p
    }
}
func (p *file) updated(ctx Context) bool { return p._updated }
func (p *file) updatedDeps(_ Context, v ...Value) []Value {
    if len(v) > 0 { p._updatedDeps = append(p._updatedDeps, v...) }
    return p._updatedDeps
}
func (p *file) stat(ctx Context) (si *statinfo) {
    var e error
    if p.info != nil {
        // good already
    } else if p.info, e = os.Stat(p.fullname()); e == nil {
        // good
    } else if x, y := e.(*fs.PathError); y {
        if false {
            erro(ctx, "%v: %v", trimPromptString(x.Path), x.Err).trace()
        }
        return
    } else {
        erro(ctx, "file.stat: %v", e).trace()
    }
    return &statinfo{ file: p }
}
func (p *file) isSysFile() (res bool) {
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
func (p *file) traverse(ctx Context) {
    if p._traved == 0 && !p.isSysFile() {
        do(ctx, act_traverse{p})
    } else if x := _execution(ctx); x != nil {
        x.traved(ctx, auto_target_value(ctx), p, nil, p)
    }
}

func (p *file) cmp(ctx Context, v Value) (res cmpres) {
    switch t := v.(type) {
    case *list: if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    case *barefile: if t.file != nil { return p.cmp(ctx, t.file) }
    case *compound, *word, *path:
        if s := v.string(ctx); s == p.filestub.name { res = cmpEqual }
    default:
        if x, y := to_file(v); !y {
            res = cmpUnknown
        } else if p.filebase == x.filebase {
            res = cmpEqual
        } else if checkpoints && p.fullname() == x.fullname() {
            erro(ctx, "same files: %v != %v", ts(p), ts(v)).trace()
        }
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
        }
    }
    return
}

func (p *file) patterned(Context) bool { return false }
func (p *file) stencil(Context, []string) (Value, []string) { return p, nil }
func (p *file) match1(_ Context, v string) (full bool, s string, stems []string) {
    if name := p.filestub.name; name == v {
        s, full = name, true
    } else if name = filepath.Join(p.sub, p.filestub.name); name == v {
        s, full = name, true
    }
    return
}
func (p *file) match(ctx Context, i any) (full bool, s any, stems []string) {
    switch t := i.(type) {
    case string: return p.match1(ctx, t)
    case Value: if !isTrivial(t) { return p.match1(ctx, t.string(ctx)) }
    default:
        erro(ctx, "matching file '%v' with unknown input: %v", p, i).trace()
    }
    return
}

func (p *file) change(dir, sub, name string) (okay bool) {
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

func (p *file) suffix(ctx Context, val Value) (_ Value) {
    var stub = *p.filestub
    switch v := val.(type) {
    case *punct, *word:
        if false && indeterminate(ctx, v) {
            erro(ctx, "indeterminate file suffix: %v", ts(v)).trace()
        }
        stub.name += v.string(ctx)
    default:
        erro(ctx, "wrong file suffix: %v %s", stub.name, ts(v)).trace()
    }
    return &file{p.valbase,p.filebase,&stub}
}

type filecontent struct{ *file ; content []byte }

type flag struct{ Value }
func (p flag) kind() Kind { return KindFlag }
func (p flag) hash(ctx Context) uint64 { return fnv1(ctx, p, p.Value) }
func (p flag) ts(string) string { return "{=flag "+ts(p.Value)+"}" }
func (p flag) String() (s string) {
    if s = "-"; p.Value != nil && !isNone(p.Value) && !isNull(p.Value) {
        s += p.Value.String()
    }
    return
}
func (p flag) string(ctx Context) (s string) {
    if s = "-"; p.Value != nil && !isNone(p.Value) && !isNull(p.Value) {
        s += p.Value.string(ctx)
    }
    return
}
func (p flag) int(ctx Context) (_ int64) { return -p.Value.int(ctx) }
func (p flag) float(ctx Context) (_ float64) { return -p.Value.float(ctx) }
func (p flag) Position() (pos Position) {
    pos = p.Value.Position()
    pos.Column -= 1
    return
}
func (p flag) prefix(ctx Context, val Value) Value { return _suffix(ctx, val, p) }
func (p flag) suffix(ctx Context, val Value) Value { return flag{compose(ctx, p.Value, val)} }
func (p flag) match(ctx Context, i any) (full bool, res any, stems []string) {
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
        erro(ctx, "%v → %v", p, ts(i)).trace()
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
        if false { erro(ctx, "flag name is trivial").trace() }
    } else if f, y := p.Value.(flag); y {
        res, match = f.opt(ctx, name)
    } else if s := p.Value.string(ctx); s == name {
        res, match = name, true
    }
    return
}
func (p flag) cmp(ctx Context, v Value) (res cmpres) {
    if v == nil {
        // ...
    } else if a, y := v.(flag); y {
        res = p.Value.cmp(ctx, a.Value)
    } else if c, y := v.(*compound); y {
        var elems []Value // right hand side compound elements
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
        var a = p.Value.int(ctx)
        if b := -i.int64; a == b { res = cmpEqual } else
        if a < b { res = cmpGreater } else { res = cmpSmaller }
    } else if f, y := v.(*float); y && f.float64 < 0 {
        var a = p.Value.float(ctx)
        if b := -f.float64; a == b { res = cmpEqual } else
        if a < b { res = cmpGreater } else { res = cmpSmaller }
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
        }
    }
    return
}
func (p flag) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p flag) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.hit_check(ctx, c, &res, &fullmatch)
    }

    if false && indeterminate(ctx, p.Value) {
        erro(ctx, "%v : indeterminate : %v", p, ts(p.Value)).trace()
    }

    if c, fullmatch = c.hit(ctx, MINUS); c == nil { return }

    if p.Value.kind()&(KindNull|KindNone) != 0 { return c, fullmatch }

    return c.hit(&flag_hit{ctx,p}, p.Value)
}

type strcomp struct{ elements } // "string compound"
func (_ *strcomp) kind() Kind { return KindStrcomp }
func (p *strcomp) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()...) }
func (p *strcomp) String() (s string) { return `"` + p.src() + `"` }
func (p *strcomp) src() (s string) {
    for _, elem := range p.elems { s += elem.String() }

    var err error
    var buf bytes.Buffer
    // buf.WriteString(`"`)
    defer func() {
        // buf.WriteString(`"`)
        s = buf.String()
    } ()

    // Escape string chars
    for i := strings.IndexAny(s, escapedChars); i != -1; {
        if _, err = buf.WriteString(s[:i]); err != nil {
            // erro(ctx, "%v", err).trace()
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
            // erro(ctx, "%v", err).trace()
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
func (p *strcomp) string(ctx Context) (s string) {
    if v := p.expand(_final(ctx)); equal(ctx, v, p) {
        for _, elem := range p.elems { s += elem.string(ctx) }
    } else {
        s = v.string(ctx)
    }
    if truly(ctx, is_defname{}) {
        s = `"`+s+`"`
    }
    return
}
func (p *strcomp) float(ctx Context) (_ float64) {
    if f, e := strconv.ParseFloat(p.string(ctx), 64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return f
    }
}
func (p *strcomp) int(ctx Context) (_ int64) {
    if i, e := strconv.ParseInt(p.string(ctx), 10, 64); e != nil {
        erro(ctx, "%v", e).trace()
        return
    } else {
        return i
    }
}
func (p *strcomp) true(ctx Context) bool { return p.elements.true(ctx) }
func (p *strcomp) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *strcomp) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (p *strcomp) expandable(ctx Context) bool { return p.elements.expandable(ctx) }
func (p *strcomp) expand(ctx Context) (res Value) {
    if truly(ctx, ex_path_str{}) {
        res = _pathstr(ctx, p.string(ctx))
    } else if elems := expand(ctx, p.elems...); diff(ctx, elems, p.elems) {
        res = &strcomp{elements{elems}}
    } else {
        res = p
    }
    return
}
func (p *strcomp) match(ctx Context, i any) (bool, any, []string) {
    return stringMatch(ctx, p, i)
}
func (p *strcomp) stencil(ctx Context, stems []string) (Value, []string) {
    return p, stems
}
func (p *strcomp) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.cmp_check(ctx, v, &res)
    }
    switch x := v.(type) {
    case *strcomp:
        if p.len() == x.len() {
            for i, elem := range p.elems {
                if t := elem.cmp(ctx, x.elems[i]); t != cmpEqual {
                    return t
                }
            }
            return cmpEqual
        } else if s, t := p.String(), x.String(); s == t {
            return cmpEqual
        } else if s < t {
            return cmpSmaller
        } else {
            return cmpGreater
        }
    case *list:
        if n := x.len(); n == 1 {
            return p.cmp(ctx, x.elems[0])
        } else if n == 0 && 0 == p.len() {
            return cmpEqual
        }
    }
    return
}
func (p *strcomp) traverse(ctx Context) { do(ctx, act_traverse{p}) }

type list struct{ elements }
func (_ *list) kind() Kind { return KindList }
func (p *list) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()) }
func (p *list) Position() (pos Position) {
    if 0 < len(p.elems) { pos = p.elems[0].Position() }
    return
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
        if e != nil {
            if t := e.string(ctx); t != "" {
                if s != "" && !strings.HasSuffix(s, "\n") { s += " " }
                s += t
            }
        }
    }
    return
}
func (p *list) float(ctx Context) (f float64) { return float64(p.int(ctx)) }
func (p *list) int(ctx Context) (i int64) {
    if n := len(p.elems); n == 1 {
        // If there's only one element, treat it as a scalar.
        return p.elems[0].int(ctx)
    } else {
        return int64(n)
    }
}
func (p *list) suffix(ctx Context, val Value) (res Value) {
    var n = p.len()-1
    if n < 0 { return val }

    var a = append(p.elems[:n], compose(ctx, p.elems[n], val))
    return &list{elements{a}}
}
func (p *list) expand(ctx Context) (res Value) {
    var a = expand(ctx, p.elems...)
    var d = diff(ctx, a, p.elems)
    if d {
        res = &list{elements{a}}
    } else {
        res = p
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        p.expand_check(ctx, d, a, res)
    }
    return
}
func (p *list) traverse(ctx Context) {
    for _, elem := range p.elems {
        elem.traverse(ctx)
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
func (p *list) delete(ctx Context) (files []*file) {
    for _, elem := range p.elems {
        files = append(files, elem.delete(ctx)...)
    }
    return
}
func (p *list) stamp(ctx Context) (files []*file) {
    for _, elem := range p.elems {
        files = append(files, elem.stamp(ctx)...)
    }
    return
}
func (p *list) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.cmp_check(ctx, v, &res)
    }

    if l, y := v.(*list); y {
        return compareElems(ctx, merge(p.elems...), merge(l.elems...))
    } else if 1 == p.len() {
        return p.elems[0].cmp(ctx, v)
    }

    if c, y := v.(*compound); y && 2 == c.len() && 1 < p.len() {
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

func (p *list) match(ctx Context, i any) (full bool, s any, stems []string) {
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

type group struct{ valbase ; elements }
func (_ *group) kind() Kind { return KindGroup }
func (_ *group) ident(Context) (s string) { return }
func (p *group) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()) }
func (p *group) cond() (_ bool) { return p.elements.cond() }
func (p *group) Position() Position { return p.valbase.Position() }
func (p *group) String() string {
    var strs []string
    for _, elem := range p.elems {
        if elem == nil {
            strs = append(strs, "<nil>")
        } else {
            strs = append(strs, elem.String())
        }
    }
    return "(" + strings.Join(strs, " ") + ")"
}
func (p *group) string(ctx Context) (s string) {
    s = "("
    for i, elem := range p.elems {
        if i > 0 { s += " " }
        s += elem.string(ctx)
    }
    s += ")"
    return
}
func (p *group) true(ctx Context) (t bool) {
    if t = len(p.elems) > 0; t {
        for _, elem := range p.elems {
            if t = elem.true(ctx); !t { break }
        }
    }
    return
}
func (p *group) refs(ctx Context, v Value) bool { return p.elements.refs(ctx, v) }
func (p *group) defs(ctx Context, s ...string) []*def { return p.elements.defs(ctx, s...) }
func (_ *group) delete(Context) (_ []*file) { return }
func (_ *group) patterned(Context) (_ bool) { return }
func (_ *group) stamp(Context) (_ []*file) { return }
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
    errostack(ctx, 3, "traversing group: %v", p).trace()
}
func (p *group) cmp(ctx Context, v Value) (res cmpres) {
    if a, y := v.(*group); y {
        if l1, l2 := len(p.elems), len(a.elems); l1 == 0 && l2 == 0 {
           return cmpEqual
        }
        res = compareElems(ctx, p.elems, a.elems)
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
        }
    }
    return
}
func (p *group) match(ctx Context, i any) (full bool, s any, stems []string) {
    // TODO: for _, elem := range { elem.match(ctx, i) }
    return
}
func (p *group) stencil(ctx Context, stems []string) (val Value, rest []string) {
    return p, stems
}

func parseGroupValue(ctx Context, g *group) (result Value) {
    if len(g.elems) == 0 {
        return g
    } else {
        var w *word
        switch kind := g.elems[0].(type) {
        case *word: w = kind
        case *group:
            if len(kind.elems) > 0 {
                var ( name = kind.elems[0]; y bool )
                if w, y = name.(*word); !y {
                    erro(ctx, "unsupported name type: %T %v", name, name).trace()
                }
            }
        }
        if w != nil {
            switch w.s {
            case "plain", "json", "yaml", "xml":
                result = _list(g.elems[1:]...)
            }
        }
        if isNull(result) { result = g }
    }
    return
}

type pair struct{ key, val Value }
func (p *pair) kind() Kind { return KindPair }
func (p *pair) hash(ctx Context) uint64 { return fnv1(ctx, p, p.key, p.val) }
func (p *pair) Position() Position { return p.key.Position() }
func (p *pair) cond() (_ bool) {
    if p.key.cond() { return true }
    if p.val.cond() { return true }
    return
}
func (p *pair) String() string {
    return p.key.String() + `=` + p.val.String()
}
func (p *pair) string(ctx Context) string {
    return p.key.string(ctx) + "=" + p.val.string(ctx)
}
func (p *pair) ts(t string) string {
    return fmt.Sprintf("{=%s %v=%v}", t, ts(p.key), ts(p.val))
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
func (p *pair) int(ctx Context) (_ int64) { return p.val.int(ctx) }
func (p *pair) float(ctx Context) (_ float64) { return p.val.float(ctx) }
func (p *pair) refs(ctx Context, v Value) bool { return p.key.refs(ctx, v) || p.val.refs(ctx, v) }
func (p *pair) defs(ctx Context, s ...string) []*def {
    return append(p.key.defs(ctx, s...), p.val.defs(ctx, s...)...)
}
func (p *pair) ident(Context) (_ string) { return }
func (p *pair) stamp(Context) (_ []*file) { return }
func (p *pair) stat(Context) (_ *statinfo) { return }
func (p *pair) match(Context, any) (_ bool, _ any, _ []string) { return }
func (p *pair) patterned(Context) (_ bool) { return }
func (p *pair) delete(Context) (_ []*file) { return }
func (p *pair) updated(Context) (_ bool) { return }
func (p *pair) updatedDeps(Context, ...Value) (_ []Value) { return }
func (p *pair) expandable(ctx Context) bool {
    return p.key.expandable(ctx) || (truly(ctx, propExPairVal) && p.val.expandable(ctx))
}
func (p *pair) expand(ctx Context) (res Value) {
    var k, v = p.key.expand(ctx), p.val
    if truly(ctx, propExPairVal) { v = v.expand(ctx) }
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
    if x, y := v.(cond); y {
        return p.cmp(ctx, x.Value)
    }
    if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    if x, y := v.(*pair); y {
        if p == x {
            if checkpoints && truly(ctx, is_test_mode{}) {
                if p.key == nil {
                    erro(ctx, "nil key : %v", p.val).trace()
                } else if r := p.key.cmp(ctx, x.key); r != cmpEqual {
                    erro(ctx, "%v, %v ⇔ %v", r, tv(p.key), tv(x.key)).trace()
                }
                if p.val == nil {
                    // erro(ctx, "%v : nil val", p.key).trace()
                } else if r := p.val.cmp(ctx, x.val); r != cmpEqual {
                    erro(ctx, "%v, %v ⇔ %v", r, tv(p.val), tv(x.val)).trace()
                }
            }
            res = cmpEqual
        } else {
            if  res = p.key.cmp(ctx, x.key); res == cmpEqual {
                res = p.val.cmp(ctx, x.val)
            }
        }
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, tv(p), tv(v)).trace()
        }
    }
    return
}
func (p *pair) traverse(ctx Context) {
    erro(ctx, "traversing pair '%v' is undefined", p)
    errostack(ctx, -1, "pair is not traversible: %v", p).trace()
}

type skipped struct{ Value }
func (s skipped) kind() Kind { return s.Value.kind()|KindSkipped }

type expanded struct{ Value }
func (s expanded) kind() Kind { return s.Value.kind()|KindExpanded }
func (s expanded) hash(ctx Context) uint64 { return fnv1(ctx, s, s.Value) }
func (s expanded) _cmp(ctx Context, v Value) (res cmpres) {
    if x, y := v.(expanded); y {
        res = s.Value.cmp(ctx, x.Value)
    } else if x, y := v.(cond); y {
        res = s.Value.cmp(ctx, x.Value)
    } else if x, y := v.(*list); y && x.len() == 1 {
        res = s.Value.cmp(ctx, x.elems[0])
    } else if false {
        res = s.Value.cmp(ctx, v)
    }
    return
}

type untraversed struct{ Value }
func (u untraversed) hash(c Context) uint64 { return fnv1(c, u, u.Value) }
func (u untraversed) expand(c Context) Value { return untraversed{u.Value.expand(c)} }
func (u untraversed) traverse(Context) {}

type check_ex struct{}

func ex(ctx Context, p, _x Value, _a, _o []Value, _l token, _cl bool) (res Value) {
    var x, t, v Value
    var a, o []Value

    unex := func() Value {
        if !equal(ctx, _x, x) || diff(ctx, a, _a) {
            d := delegate{valbase{p.Position()}, _l, x, _o, a}
            if _cl {
                return &closure{d}
            } else {
                return &d
            }
        }
        return p
    }

    _un := func() Value {
        a = expand(ctx, _a...)
        return unex()
    }

    _ev := func() Value {
        if t, a = evoke(ctx, x, _o, _a); equal(ctx, x, t) {
            return unex()
        } else if x, y := t.(expanded); y {
            return x.Value
        } else {
            return t
        }
    }

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer ex_check(ctx, p, _x, _a, _o, _l, _cl, &res, &t, &x, &a, &o)
    }

    x = _x.expand(ctx)

    switch _x.(type) {
    case *auto:
        if _cl {
            break
        } else if _, y := x.(*def); !y && !truly(ctx, ex_auto{}) {
            return _un()
        } else {
            return _ev()
        }
    }

    switch t := x.(type) {
    case disjunction:
        errostack(pc(ctx,p), 3, "TODO: %s → %s", ts(_x), ts(x)).trace()
    case *project, self:
        a = expand(ctx, _a...)
        return x
    case *arrow:
        return _ev()
    case *list:
        var va []Value
        var vb = valbase{p.Position()}
        for _, v := range t.elems {
            var t Value
            var d = delegate{vb, _l, v, _o, _a}
            if _cl { t = &closure{d} } else { t = &d }
            va = append(va, ex(ctx, t, v, _a, _o, _l, _cl))
        }
        switch len(va) {
        case 0 : return _null(_x.Position())
        case 1 : return va[0]
        default: return disjunction{&list{elements{va}}}
        }
    default:
        if indeterminate(ctx, x) {
            return _un()
        }
    }

    const (
        DELEGATE = 1
        CLOSURE  = 2
    )

    var prop int
    if !_cl && truly(ctx, ex_delegate{}) {
        switch x.(type) {
        case *auto:
            if equal(ctx, x, _x) {
                return _un()
            }
            return _ev()
        case *builtin, *def:
            return _ev()
        }
        prop = DELEGATE
    } else if truly(ctx, ex_closure{}) {
        prop = CLOSURE
    }
    if prop == 0 || indeterminate(ctx, x) {
        return _un()
    }

    switch _l {
    case LBRACE, STRING, STRCOMP: // &{xxx}  &'xxx'  &"xxx",  else/illegal: &(xxx)
        switch prop {
        case DELEGATE: v = project_entry(ctx, x)
        case CLOSURE:  v = closure_entry(ctx, x)
        }

    default:
        var s string
        switch t := x.(type) {
        case   object: s = t.ident(ctx)
        case stringer: s = t.string(ctx)
        default:
            erro(ctx, "unexable: %s", tv(x)).trace()
        }
        if s != "" {
            switch prop {
            case DELEGATE: v = project_resolve(ctx, s)
            case CLOSURE:  v = closure_resolve(ctx, s)
            }
        }
    }

    if v != nil {
        x = v
        return _ev()
    }

    if _, y := _x.(cond); y {
        if false { return p }
        return _null(p.Position())
    }

    return _un()
}

type ex_delegate struct{}
type delegate struct{
    valbase
    l   token
    x   Value
    o []Value
    a []Value
    // TODO: patsubst Value, aka lhs%=rhs% like in $(var:lhs%=rhs%)
}
func (p *delegate) kind() Kind { return p.valbase.kind()|KindDelegate }
func (p *delegate) hash(ctx Context) uint64 {
    var a = []any{ p.l, p.x }
    for _, o := range p.o { a = append(a, o) }
    for _, v := range p.a { a = append(a, v) }
    return fnv1(ctx, p, a...)
}
func (p *delegate) String() string { return p.src("$") }
func (p *delegate) string(ctx Context) (s string) {
    if x, y := p.x.(*project); y { return x.name }
    if v := p.final(ctx); v != nil { s = v.string(ctx) }
    if checkpoints && truly(ctx, is_test_mode{}) {
        if p.String() == "$/" {
            if s == "" {
                errostack(pc(ctx,p), 3, "%v", ts(p))
                errostack(pc(ctx,p), 3, "%v", p)
                errostack(pc(ctx,p), 3, "%v", ts(ctx)).trace()
            } else if !filepath.IsAbs(s) {
                errostack(pc(ctx,p), 3, "%v", p)
                errostack(pc(ctx,p), 3, "%v", ts(p))
                errostack(pc(ctx,p), 3, "→ %v", s)
                errostack(pc(ctx,p), 3, "%v", ts(ctx)).trace()
            }
        }
        if p.String() == "$." {
            if strings.HasPrefix(s, "./") {
                errostack(pc(ctx,p), 3, "%v", p)
                errostack(pc(ctx,p), 3, "%v", ts(p))
                errostack(pc(ctx,p), 3, "→ %v", s)
                errostack(pc(ctx,p), 3, "%v", ts(ctx)).trace()
            }
        }
    }
    return
}
func (p *delegate) ts(t string) (s string) {
    s = "{=" + t + " " + ts(p.x)
    if p.o != nil { s += " " + ts(p.o) }
    for _, a := range p.a { s += " " + ts(a) }
    s += "}"
    return
}
func (p *delegate) true(ctx Context) (t bool) {
    if v := p.final(ctx); v != nil { t = v.true(ctx) }
    return
}
func (p *delegate) int(ctx Context) (i int64) {
    if v := p.final(ctx); v != nil { i = v.int(ctx) }
    return
}
func (p *delegate) float(ctx Context) (f float64) {
    if v := p.final(ctx); v != nil { f = v.float(ctx) }
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
func (p *delegate) final(ctx Context) (_ Value) {
    var v = p.expand(_final(ctx))
    if v == nil || equal(ctx, p, v) || (v.expandable(ctx) && !p.aone(ctx, v)) {
        return
    }
    if v.refs(ctx, p) {
        return
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        p.final_val_check(ctx, v, !v.refs(ctx, p))
    }
    return v
}
func (p *delegate) isValidtoken() (res bool) {
    switch p.l {
    case LPAREN, LBRACE, STRING, STRCOMP, ILLEGAL:
        res = true
    default: // for $. $/ $1 ... &. &/ &1 ... etc.
        res = p.l.is_closure_delegate()
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
        errostack(pc(ctx,p), 3, "delegation of nil (s=%v)", p, s).trace()
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
    if v := p.final(ctx); v != nil { v.traverse(ctx) }
}
func (p *delegate) ident(ctx Context) (name string) {
    const sel = true
    switch x := p.x.(type) {
    case interface{ ident(Context) string }: name = x.ident(ctx)
    case *arrow: if sel { name = x.string(ctx) }
    }
    return
}
func (p *delegate) src(l string) (s string) {
    if s = p._src(); !(p.l.is_closure_delegate()) { s = l + s }
    return
}
func (p *delegate) _src() (s string) { // source representation
    switch x := p.x.(type) {
    case       *def:       s += x.name
    case *arrow:       s += x.String()
    default: if x != nil { s += x.String() }
    }

    if p.o != nil { // options
        s += "("
        for i, v := range p.o {
            if 0 < i { s += " " }
            s += v.String()
        }
        s += ")"
    }

    for i, a := range p.a {
        if 0 == i { s += " " } else { s += "," }
        s += a.String()
    }

    switch p.l {
    case STRCOMP: return `"`+s+`"`
    case  STRING: return `'`+s+`'`
    case  LPAREN: return "("+s+")"
    case  LBRACE: return "{"+s+"}"
    case ILLEGAL: return "["+s+"]"
    default:
        if p.l.is_closure_delegate() {
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
        if res = truly(ctx, ex_delegate{}) || p.x.expandable(ctx); !res {
            for _, a := range p.a { if a.expandable(ctx) { return true }}
        }
    }
    return
}
func (p *delegate) expand(ctx Context) (res Value) {
    return ex(ctx, p, p.x, p.a, p.o, p.l, false)
}
func (p *delegate) prefix(ctx Context, v Value) Value { return _suffix(ctx, v, p) }
func (p *delegate) suffix(ctx Context, v Value) Value { return _compound(p).suffix(ctx, v) }
func (p *delegate) match(ctx Context, i any) (full bool, s any, stems []string) {
    if v := p.expand(ctx); v != nil {
        if v != p { return v.match(ctx, i) }
    } else {
        errostack(pc(ctx,p), 3, "%v: nil match", p).trace()
    }
    return
}
func (p *delegate) stat(ctx Context) (_ *statinfo) {
    errostack(pc(ctx,p), 3, "cant stat delegate %v, must expand it first", p).trace()
    return
}
func (p *delegate) stamp(ctx Context) (_ []*file) {
    errostack(pc(ctx,p), 3, "cant stamp delegate %v, must expand it first", p).trace()
    return
}
func (p *delegate) delete(ctx Context) (_ []*file) {
    errostack(pc(ctx,p), 3, "cant delete delegate %v, must expand it first", p).trace()
    return
}
func (p *delegate) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.cmp_check(ctx, v, &res)
    }

    switch t := v.(type) {
    case *list:
        if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }

    case *delegate:
        if p == t { return cmpEqual }

        var cr cmpres
        if p.x == t.x {
            cr = cmpEqual
        } else {
            cr = p.x.cmp(ctx, t.x)
        }

        if checkpoints && truly(ctx, is_test_mode{}) {
            if cr != cmpEqual && p.x.String() == t.x.String() {
                errostack(pc(ctx,p), 3, "%v: %v %v", ts(p), ts(p.x), ts(t.x)).trace()
            }
            if len(p.a) == len(t.a) {
                for i, v := range p.a {
                    if v.cmp(ctx, t.a[i]) != cmpEqual && v.String() == t.a[i].String() {
                        errostack(pc(ctx,p), 3, "%v: %v %v", ts(p), ts(v), ts(t.a[i])).trace()
                    }
                }
            }
        }

        if cr == cmpEqual && len(p.a) == len(t.a) {
            for i, v := range p.a {
                if r := v.cmp(ctx, t.a[i]); r != cmpEqual { return r }
            }
        }

        return cr
    }
    return
}

type ex_closure struct{}
type closure struct{ delegate }
func (p *closure) kind() Kind { return p.valbase.kind()|KindClosure }
func (p *closure) hash(ctx Context) uint64 {
    var a = []any{ p.l, p.x }
    for _, o := range p.o { a = append(a, o) }
    for _, v := range p.a { a = append(a, v) }
    return fnv1(ctx, p, a...)
}
func (p *closure) String() (s string) { return p.src("&") }
func (p *closure) string(ctx Context) (s string) {
    if v := p.final(ctx); v != nil { s = v.string(ctx) }
    return
}
func (p *closure) ts(t string) (s string) {
    s = "{=" + t + " " + ts(p.x)
    if p.o != nil { s += " " + ts(p.o) }
    for _, a := range p.a { s += " " + ts(a) }
    s += "}"
    return
}
func (p *closure) true(ctx Context) (t bool) {
    if v := p.final(ctx); v != nil { t = v.true(ctx) }
    return
}
func (p *closure) prefix(ctx Context, v Value) Value { return _suffix(ctx, v, p) }
func (p *closure) suffix(ctx Context, v Value) Value { return _compound(p).suffix(ctx, v) }
func (p *closure) final(ctx Context) (_ Value) {
    var v = p.expand(_final(ctx))
    if v == nil || equal(ctx, p, v) || (v.expandable(ctx) && !p.aone(ctx, v)) {
        return
    }
    if v.refs(ctx, p) {
        return
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        p.final_val_check(ctx, v, !v.refs(ctx, p))
    }
    return v
}
func (p *closure) expand(ctx Context) (res Value) {
    return ex(ctx, p, p.x, p.a, p.o, p.l, true)
}
func (p *closure) expandable(ctx Context) (res bool) {
    if false {
        res = true
    } else {
        if res = truly(ctx, ex_closure{}) ||  p.x.expandable(ctx); !res {
            for _, a := range p.a { if a.expandable(ctx) { return true }}
        }
    }
    return
}
func (p *closure) match(ctx Context, i any) (full bool, s any, stems []string) {
    if v := p.expand(ctx); v != p {
        return v.match(ctx, i)
    } else if false {
        errostack(ctx, 3, "unexpand closure: %v", v).trace()
    }
    return
}
func (p *closure) refs(ctx Context, v Value) (res bool) {
    if p.x == nil {
        errostack(pc(ctx,p), 3, "closure of nil: %v (%v)", ts(p), ts(v)).trace()
    }
    if p == v || p.x == v || p.x.refs(ctx, v) { return true }
    for _, a := range p.a { if a.refs(ctx, v) { return true } }
    return
}
func (p *closure) traverse(ctx Context) {
    if v := p.final(ctx); v != nil { v.traverse(ctx) }
}
func (p *closure) stat(ctx Context) (_ *statinfo) {
    errostack(pc(ctx,p), 3, "cant stat closure %v, must expand it first", p).trace()
    return
}
func (p *closure) stamp(ctx Context) (_ []*file) {
    errostack(pc(ctx,p), 3, "cant stamp closure %v, must expand it first", p).trace()
    return
}
func (p *closure) delete(ctx Context) (_ []*file) {
    errostack(pc(ctx,p), 3, "cant stamp closure %v, must expand it first", p).trace()
    return
}
func (p *closure) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    switch t := v.(type) {
    case *list: if t.len() == 1 { return p.cmp(ctx, t.elems[0]) }
    case *closure:
        if p == t { return cmpEqual }
        if res = p.x.cmp(ctx, t.x); res == cmpEqual {
            if len(p.a) == len(t.a) {
                for i, a := range p.a {
                    if res = a.cmp(ctx, t.a[i]); res != cmpEqual { return }
                }
            }
            return
        }
    }
    return
}

type arrow struct{
    valbase
    t token
    o Value // object or arrow
    s Value
}
func (p *arrow) hash(ctx Context) uint64 { return fnv1(ctx, p, p.t, p.o, p.s) }
func (p *arrow) ts(t string) string { return "{="+t+" "+ts(p.o)+p.t.String()+ts(p.s)+"}" }
func (p *arrow) String() string { return p.o.String() + p.t.String() + p.s.String() }
func (p *arrow) string(ctx Context) (s string) {
    p.ex(ctx, func(v Value) { s = v.string(ctx) })
    return
}
func (p *arrow) true(ctx Context) (t bool) {
    p.ex(ctx, func(v Value) { t = v.true(ctx) })
    return
}
func (p *arrow) int(ctx Context) (i int64) {
    var e error
    p.ex(ctx, func(v Value) {
        if s := v.string(ctx); s != "" {
            i, e = strconv.ParseInt(s, 10, 64)
        }
    })
    if e != nil {
        errostack(pc(ctx,p), 3, "%v", e).trace()
    }
    return
}
func (p *arrow) float(ctx Context) (f float64) {
    var e error
    p.ex(ctx, func(v Value) {
        if s := v.string(ctx); s != "" {
            f, e = strconv.ParseFloat(s, 64)
        }
    })
    if e != nil {
        errostack(pc(ctx,p), 3, "%v", e).trace()
    }
    return
}
func (p *arrow) refs(ctx Context, v Value) bool {
    return p.o.refs(ctx, v) || p.s.refs(ctx, v)
}
func (p *arrow) defs(ctx Context, s ...string) []*def {
    return append(p.o.defs(ctx, s...), p.s.defs(ctx, s...)...)
}
func (p *arrow) expandable(ctx Context) (res bool) {
    return p.o.expandable(ctx) || p.s.expandable(ctx)
}
func (p *arrow) expand(ctx Context) (res Value) {
    o := p.o.expand(ctx)
    s := p.s.expand(ctx)

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.expand_check(ctx, &o, &s, &res)
    }

    if !equal(ctx, o, p.o) || !equal(ctx, s, p.s) {
        return &arrow{p.valbase, p.t, o, s}
    } else {
        return p
    }
}
func (p *arrow) evoke(ctx *evocation) (res Value) {
    var o, s Value

    if checkpoints && truly(ctx, is_test_mode{}) {
        defer p.evoke_check(ctx, &o, &s, &res)
    }

    if x, y := p.o.(evoker); y {
        o = x.evoke(ctx)
    } else {
        o = p.o.expand(ctx)
    }

    switch x := o.(type) {
    case *word:
        if t := project_resolve(ctx, x.s); t != nil { o = t }
    }

    if false && p.t.is_select_prog() {
        if x, y := o.(*project); y && x != nil {
            res = x.entry(ctx, p.s, false)
        }
        return
    }

    s = p.s.expand(ctx)

    if x, y := o.(interface{ sel(Context, string) any }); y {
        if indeterminate(ctx, s) {
            if true { note(pc(ctx,s), "%v %v", o, s).debug() }
        } else if x, y := x.sel(ctx, s.string(ctx)).(evoker); y {
            return x.evoke(ctx)
        } else {
            return ease(ctx, x)
        }
    }
    return _null(p.Position())
}
func (p *arrow) ex(ctx Context, f func(Value)) {
    if v := p.expand(ctx); v != nil && !equal(ctx, v, p) { f(v) }
}
func (p *arrow) traverse(ctx Context) {
    if val := p.expand(ctx); isTrivial(val) {
        warn(ctx, "selected trivial value '%v' (%v, %v) ", p, ts(p.o), ts(p.s)).debug(10)
    } else {
        val.updated(ctx) // NOTE: ensure that updated flag is correct (see rule.updated)
        val.traverse(ctx)
    }
}
func (p *arrow) updated(ctx Context) (res bool) { // NOTE: this seems not affecting the result
    if val := p.expand(ctx); isTrivial(val) {
        note(ctx, "selected value '%v' is trivial", p).debug()
    } else {
        res = val.updated(ctx)
    }
    return res
}
func (p *arrow) updatedDeps(ctx Context, v ...Value) (res []Value) { // NOTE: this seems not affecting the result
    if val := p.expand(ctx); isTrivial(val) {
        note(ctx, "selected value '%v' is trivial", p).debug()
    } else {
        res = val.updatedDeps(ctx, v...)
    }
    return res
}
func (p *arrow) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    switch x := v.(type) {
    case *arrow:
        if p.t == x.t && p.o.cmp(ctx, x.o) == cmpEqual {
            return p.s.cmp(ctx, x.s)
        }
    case *list:
        if len(x.elems) == 1 {
            return p.cmp(ctx, x.elems[0])
        }
    }
    return
}
func (p *arrow) stat(ctx Context) (_ *statinfo) {
    errostack(pc(ctx,p), 3, "cant stat arrow %v, must expand it first", p).trace()
    return
}
func (p *arrow) stamp(ctx Context) (_ []*file) {
    errostack(pc(ctx,p), 3, "cant stamp arrow %v, must expand it first", p).trace()
    return
}
func (p *arrow) delete(ctx Context) (_ []*file) {
    errostack(pc(ctx,p), 3, "cant stamp arrow %v, must expand it first", p).trace()
    return
}

// percpat represents percent pattern expressions (e.g. '%.o')
type percpat struct{
    valbase // TODO: supporting multiple %: foo%bar%xxx
    Prefix Value
    Suffix Value
}
func (p *percpat) hash(ctx Context) uint64 { return fnv1(ctx, p, p.Prefix, p.Suffix) }
func (p *percpat) String() (s string) {
    if !isNull(p.Prefix) { s += p.Prefix.String() }
    s += `%`
    if !isNull(p.Suffix) { s += p.Suffix.String() }
    return
}
func (p *percpat) string(ctx Context) (s string) {
    if p.Prefix != nil { s += p.Prefix.string(ctx) }
    s += "%"
    if p.Suffix != nil { s += p.Suffix.string(ctx) }
    return
}
func (p *percpat) ts(t string) string {
    return fmt.Sprintf("{=%s %s %s}", t, ts(p.Prefix), ts(p.Suffix))
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
            warn(ctx, "mixing % pattern might have performance impact: %v", p).debug()
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
func (p *percpat) match(ctx Context, i any) (full bool, result any, stems []string) {
    switch t := i.(type) {
    case string: return p.match1(ctx, t)
    case *filestub:
        if full, result, stems = p.match1(ctx, t.name); full || result != "" {
            return // NOTE: done if path fully or partially matched
        } else if t.dir != "" || t.sub != "" {
            return p.match1(ctx, filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
        }
    case *file:
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
        erro(ctx, "patterned prefix: %T %v", p.Prefix, p.Prefix).trace()
    } else {
        vals = append(vals, p.Prefix)
    }

    if len(stems) > 0 {
        if s := stems[0]; s != "" { vals = append(vals, _word(p.position, s)) }
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
        val = _compound(vals...)
    } else {
        val = p
    }
    return
}
func (p *percpat) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *percpat) cmp(ctx Context, v Value) (res cmpres) {
    if a, ok := v.(*percpat); ok {
        if p.Prefix.cmp(ctx, a.Prefix) == cmpEqual {
            if p.Suffix.cmp(ctx, a.Suffix) == cmpEqual {
                res = cmpEqual
            }
        }
    } else if l, ok := v.(*list); ok && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    if checkpoints && truly(ctx, is_test_mode{}) {
        if res != cmpEqual && p.String() == v.String() {
            erro(ctx, "%v, %v ⇔ %v", res, ts(p), ts(v)).trace()
        }
    }
    return
}
func (p *percpat) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    if checkpoints && truly(ctx, is_test_mode{}) {
        if s := ts(p.Suffix); strings.Contains(s, "{=argumented ") {
            erro(ctx, "wrong percpat: %v : suffix=%s", p, s).trace()
        }
    }

    if false && indeterminate(ctx, p) {
        erro(ctx, "valcache %v : indeterminate element : %v", p, ts(p)).trace()
    }

    var s = p.string(ctx)
    if x, y := do(&percpat_hit{ctx,p}, hit_perc{c,s}).(valcache_bool); y {
        return x.valcache, x.bool
    }

    erro(ctx, "unhit: %v : %v", ts(p), ts(ctx)).trace()
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
                    suffix = _globpat(t...)
                } else {
                    prefix = _compound(t...)
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
                suffix = _globpat(t...)
            } else if len(t) > 1 {
                suffix = _compound(t...)
            } else if len(t) > 0 {
                suffix = t[0]
            }
        }
    }
    return
}

func correctPunctForMatch(seg Value) Value {
    if x, y := seg.(*compound); y {
        for _, elem := range x.elems {
            if _, y := elem.(*path); y { seg = nil; break }
        }
    }
    return seg
}

type compositePattern struct{ Value ; constraints []Value }
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
func (p compositePattern) match(ctx Context, i any) (full bool, result any, stems []string) {
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
//     errostack(ctx, 5, "stencil unsupported").trace()
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
type globpat struct{ elements }
func (_ *globpat) kind() Kind { return KindGlobpat }
func (p *globpat) hash(ctx Context) uint64 { return fnv1(ctx, p, p.any()...) }
func (p *globpat) String() (s string) {
    var explicit bool
    for _, comp := range p.elems {
        s += comp.String()
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
func (p *globpat) float(ctx Context) (_ float64) { return }
func (p *globpat) int(ctx Context) (_ int64) { return }
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
func (p *globpat) match(ctx Context, i any) (full bool, result any, stems []string) {
    var s string
    switch t := i.(type) {
    case      bare: s = string(t)
    case *filestub: s = t.name
    case     *file: s = t.ident(ctx)
    case     Value: s = t.string(ctx)
    case    string: s = t
    case  []string:
        if n := len(t); n == 1 {
            s = t[0]
        } else if true && n > 1 {
            s = filepath.Join(t...) // TODO: optimization: avoid joining
        } else {
            return
        }
    default:
        errostack(ctx, 3, "%v : unsupported match type: %v", p, ts(i)).trace()
    }

    var err error
    var pattern = p.string(ctx)
    if full, stems, err = globMatch(ctx, pattern, s); full { result = s }
    if err != nil {
        errostack(ctx, 3, "%v : glob error: %v", p, err).trace()
    }
    return
}
func (p *globpat) stencil(ctx Context, stems []string) (val Value, rest []string) {
    erro(ctx, "unimplemented globpat stencil %v (stems=%v)", p, stems)
    return
}
func (p *globpat) suffix(ctx Context, val Value) (res Value) {
    if checkpoints && truly(ctx, is_test_mode{}) {
        if _, y := val.(*path); y {
            defer func() {
                erro(ctx, "%v", ts(res)).trace()
            } ()
        }
    }
    switch x := val.(type) {
    case *path:
        v := &path{x.elements}
        v.elems[0] = compose(ctx, p, v.elems[0])
        return v
    default:
        v := &globpat{p.elements}
        v.elems = append(v.elems, val)
        return v
    }
}
func (p *globpat) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *globpat) cmp(ctx Context, v Value) (res cmpres) {
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if a, y := v.(*globpat); y {
        if len(p.elems) == len(a.elems) {
            for i, c := range p.elems {
                if c.cmp(ctx, a.elems[i]) != cmpEqual {
                    return
                }
            }
            return cmpEqual
        }
    } else if x, y := v.(*list); y && len(x.elems) == 1 {
        return p.cmp(ctx, x.elems[0])
    }
    return
}
func (p *globpat) hit(ctx Context, c *valcache) (res *valcache, doneFull bool) {
    if false && indeterminate(ctx, p) {
        erro(ctx, "%v : indeterminate : %v", p, ts(p)).trace()
    }

    t := do(&globpat_hit{ctx, p}, hit_glob{c, p.string(ctx)})

    if x, y := t.(valcache_bool); y {
        return x.valcache, x.bool
    }

    erro(ctx, "miss hit: %v : %v", ts(p), ts(ctx)).trace()
    return
}

type regexpat struct{ valbase ; *regexp.Regexp }
func (p *regexpat) hash(ctx Context) uint64 { return fnv1(ctx, p, p.Regexp.String()) }
func (p *regexpat) string(ctx Context) (s string) { return p.Regexp.String() }
func (p *regexpat) String() string {
    var s = p.Regexp.String()
    if x, y := strings.CutSuffix(s, "$"); y { s = x + "$$" }
    return "{=regex "+s+"}"
}
func (p *regexpat) ts(string) string { return p.String() }
func (p *regexpat) patterned(ctx Context) bool { return true }
func (p *regexpat) match(ctx Context, i any) (full bool, result any, stems []string) {
    if p.Regexp != nil {
        var str string
        switch t := i.(type) {
        case *filestub: str = t.name
        case     *file: str = t.ident(ctx)
        case     Value: str = t.string(ctx)
        case    string: str = t
        case  []string: if len(t) == 1 { str = t[0] } else { return }
        default:
            errostack(ctx, 3, "%T %v :matching unsupported value: %T %v", p, p, i, i).trace()
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
    if checkpoints && truly(ctx, is_test_mode{}) { defer p.cmp_check(ctx, v, &res) }
    if a, y := v.(*regexpat); y {
        if a != nil {
            if s1, s2 := p.String(), a.String(); s1 == s2 {
                res = cmpEqual
            } else if s1 < s2 {
                res = cmpSmaller
            } else /*if s1 > s2*/ {
                res = cmpGreater
            }
        }
    } else if l, y := v.(*list); y && len(l.elems) == 1 {
        return p.cmp(ctx, l.elems[0])
    }
    return
}
func (p *regexpat) traverse(ctx Context) { do(ctx, act_traverse{p}) }
func (p *regexpat) expand(Context) Value { return p }
func (p *regexpat) hit(ctx Context, c *valcache) (res *valcache, fullmatch bool) {
    if false && indeterminate(ctx, p) {
        erro(ctx, "valcache %v : indeterminate element : %v", p, ts(p)).trace()
    }

    t := do(&regexpat_hit{ctx, p}, hit_regex{c, p.string(ctx)})

    if x, y := t.(valcache_bool); y {
        return x.valcache, x.bool
    }

    erro(ctx, "unhit: %v : %v", ts(p), ts(ctx)).trace()
    return
}

func values(args ...any) (elems []Value) {
    for _, a := range args {
        if x, y := a.(Value); y {
            elems = append(elems, x)
        } else if v := reflect.ValueOf(a); v.Kind() == reflect.Slice {
            for n := 0; n < v.Len(); n++ {
                elems = append(elems, values(v.Index(n).Interface())...)
            }
        } else {
            //erro(ctx, "'%v' is not value type (%T)", a, a).trace()
        }
    }
    return
}

func baremerge(args ...Value) (elems []Value) {
    for _, arg := range args {
        if l, o := arg.(*compound); o && l != nil {
            elems = append(elems, baremerge(l.elems...)...)
        } else if l, o := arg.(*list); o && len(l.elems) == 1 {
            elems = append(elems, baremerge(l.elems...)...)
        } else if d, o := arg.(*def); o {
            if d.value == nil {
                elems = append(elems, _null(d.position))
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
        switch x := a.(type) {
        case *list:
            elems = append(elems, merge(x.elems...)...)
        case Value:
            elems = append(elems, x)
        }
    }
    return
}

func xmerge(ctx Context, values ...Value) []Value {
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

func decoupleCompoundList(values ...Value) (res []Value) {
    for _, v := range values {
        if x, y := v.(*compound); y {
            var l int
        xelemsloop:
            for i, e := range x.elems {
                if t, y := e.(*list); y {
                    l += 1
                    if n := t.len(); 0 == n {
                        a := append(x.elems[:i], x.elems[i+1:]...)
                        res = append(res, &compound{elements{a}})
                    } else if 1 == n {
                        a := append(append(x.elems[:i], t.elems[0]), x.elems[i+1:]...)
                        res = append(res, &compound{elements{a}})
                    } else if 2 == n {
                        a := append(x.elems[:i], t.elems[0])
                        b := append(t.elems[1:], x.elems[i+1:]...)
                        res = append(res, &compound{elements{a}}, &compound{elements{b}})
                    } else {
                        a := append(x.elems[:i], t.elems[0])
                        b := decoupleCompoundList(t.elems[1:t.len()-1]...)
                        c := append(t.elems[t.len()-1:], x.elems[i+1:]...)
                        res = append(append(res, &compound{elements{a}}), b...)
                        x = &compound{elements{c}}
                        goto xelemsloop
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
        return int(v.int(ctx))
    }
    return
}

func int64Val(ctx Context, v Value, i int64) (res int64) {
    if res = i; v != nil {
        return v.int(ctx)
    }
    return
}

func uintVal(ctx Context, v Value, i uint32) (res uint32) {
    if res = i; v != nil {
        return uint32(v.int(ctx))
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
    for _, elem := range values {
        if elem != nil {
            if val := elem.expand(ctx) ; val != nil {
                if checkpoints && truly(ctx, is_test_mode{}) {
                    expand_check_elem(ctx, elem, val)
                }
                elems = append(elems, val)
            }
        }
    }
    return
}

func expand_path_elems(ctx Context, elems ...Value) (res []Value) {
    for _, elem := range expand(ctx, elems...) {
        if x, y := elem.(*path); y {
            res = append(res, expand_path_elems(ctx, x.elems...)...)
        } else {
            res = append(res, elem)
        }
    }
    return
}

func unique(ctx Context, values ...Value) (elems []Value) {
    seen := make(map[uint64]struct{}, len(values))
    for _, v := range values {
        var n = v.hash(ctx)
        if _, y := seen[n]; !y {
            elems = append(elems, v)
            seen[n] = struct{}{}
        }
    }
    return
}
func reverse_unique(ctx Context, values ...Value) (elems []Value) {
    seen := make(map[uint64]struct{}, len(values))
    for j := len(values)-1; 0 <= j; j -= 1 {
        var v = values[j]
        var n = v.hash(ctx)
        if _, y := seen[n]; !y {
            elems = append(elems, v)
            seen[n] = struct{}{}
        }
    }
    return
}

func splitPathStr(ctx Context, str string) (segments []Value) {
    var pos = _position(ctx)
    var a = strings.Split(str, pathSep)
    for i, s := range a {
        // TODO: calculate position for each segment
        var v Value
        if i == 0 {
            switch s {
            case ""  : v = makePunct(pos, PROOT)
            case "~" : v = makePunct(pos, TILDE)
            case "." : v = makePunct(pos, DOT)
            case "..": v = makePunct(pos, DOTDOT)
            default  : v = _word(pos, s)
            }
        } else if s == "" {
            if i+1 == len(a) {
                v = makePunct(pos, PTAIL)
            } else if false {
                v = makePunct(pos, PCON)
            } else {
                warn(ctx, "%s: %v[%d]: empty path seg", str, a, i).debug()
                continue
            }
        } else {
            v = _word(pos, s)
        }
        segments = append(segments, v)
    }
    return
}

func refs(ctx Context, a Value, v Value) bool { return a == v || a.refs(ctx, v) }
func ease(ctx Context, a any) (res Value) {
    if a == nil { return }

    var elems []Value

    switch t := a.(type) {
    case    Value: elems = append(elems, merge(t   )...)
    case  []Value: elems = append(elems, merge(t...)...)
    case     bare: elems = append(elems, _word(   _position(ctx),  string(t)))
    case     bool: elems = append(elems, _boolean(_position(ctx),         t ))
    case    int  : elems = append(elems, _decimal(_position(ctx),   int64(t)))
    case    int16: elems = append(elems, _decimal(_position(ctx),   int64(t)))
    case    int32: elems = append(elems, _decimal(_position(ctx),   int64(t)))
    case    int64: elems = append(elems, _decimal(_position(ctx),         t ))
    case   uint  : elems = append(elems, _decimal(_position(ctx),   int64(t)))
    case   uint16: elems = append(elems, _decimal(_position(ctx),   int64(t)))
    case   uint32: elems = append(elems, _decimal(_position(ctx),   int64(t)))
    case   uint64: elems = append(elems, _decimal(_position(ctx),   int64(t)))
    case  float32: elems = append(elems, _float(  _position(ctx), float64(t)))
    case  float64: elems = append(elems, _float(  _position(ctx),         t ))
    case   string: elems = append(elems, _strlit( _position(ctx),         t ))
    case []string: for _, s := range t { elems = append(elems, _strlit(_position(ctx),        s )) }
    case   []bare: for _, s := range t { elems = append(elems, _word(  _position(ctx), string(s))) }
    default: erro(ctx, "unsupported value: %v", ts(t)).trace()
    }

    if n := len(elems); 1 == n {
        return  elems[0]
    } else if 1 < n {
        return _list(elems...)
    } else {
        return _null(_position(ctx))
    }
}

func scalarize(v Value) (res Value) { // NOTE: unexpanded is not scalar
    switch t := v.(type) {
    case *none: t.x = scalarize(t.x)
    case *list:
        var n = len(t.elems)
        if n == 0 { return _null(t.Position()) }
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

func tv(i any) (s string) {
    if i == nil {
        return "{=nil}"
    } else if _, y := i.(*null); y {
        return "{=null}"
    } else if x, y := i.(*none); y {
        var s string
        if x.x != nil {
            if s = x.x.String(); s != "" { s = ": " + s }
        }
        return "{none"+s+"}"
    } else {
        if x, y := i.(interface{ String() string }); y {
            s = x.String()
        } else {
            s = fmt.Sprintf("%v", i)
        }
        return "{" + typeof(i) + ":" + s + "}"
    }
}
func ts(i any) (s string) {
    if i == nil { return "{}" }

    var t = typeof(i) //strings.Replace(fmt.Sprintf("%T", i), "smart.", "", -1)

    if strings.HasPrefix(t, "[]") {
        v := reflect.ValueOf(i)
        s  = "{="+t
        for i := 0; i < v.Len(); i += 1 {
            s += " "+ts(v.Index(i).Interface())
        }
        s += "}"
        return
    }

    switch x := i.(type) {
    case interface{ ts(string) string }: return x.ts(t)
    case Context:     return "{="+t+" "+ts(inner(x))+"}"
    case opt:         return "{="+t+" "+ts(x.Value)+"}"
    case skipped:     return "{="+t+" "+ts(x.Value)+"}"
    case expanded:    return "{="+t+" "+ts(x.Value)+"}"
    case cond:        return "{="+t+" "+ts(x.Value)+"}"
    case untraversed: return "{="+t+" "+ts(x.Value)+"}"
    default:
        s = "{="+t
        if t := fmt.Sprintf("%v", x); t != "" {
            s += " " + strings.ReplaceAll(t, "\n", `\n`)
        }
        s += "}"
        return
    }
}

type tst struct{ i any }
func (p tst) ts(string) string { return ts(p.i) }
func (p tst) String() string { return ts(p.i) }
func (p tst) Position() (_ Position) {
    if x, y := p.i.(positioner); y && x != nil {
        return x.Position()
    }
    return
}

func _argumented(val Value, a ...Value) *argumented { return &argumented{val, a} }
func _arrow(pos Position, tok token, lhs, rhs Value) *arrow { return &arrow{valbase{pos}, tok, lhs, rhs} }
func _answer(pos Position, v bool) *answer { return &answer{boolean{valbase{pos},v}} }
func _option(pos Position, v bool) *option { return &option{boolean{valbase{pos},v}} }

func _null(pos Position) *null { return &null{valbase{pos}} }
func _none(pos Position) *none { return &none{valbase{pos}, nil} }
func _boolean(pos Position, v bool) *boolean { return &boolean{valbase{pos},v} }
func _binary(pos Position, i int64) *binary { return &binary{integer{valbase{pos},i}} }
func _octal(pos Position, i int64) *octal { return &octal{integer{valbase{pos},i}} }
func _decimal(pos Position, i int64) *decimal { return &decimal{integer{valbase{pos},i}} }
func _hexadecimal(pos Position, i int64) *hexadecimal { return &hexadecimal{integer{valbase{pos},i}} }
func _float(pos Position, f float64) *float  { return &float{valbase{pos},f} }
func _raw(pos Position, s string) *raw       { return &raw{valbase{pos},s} }
func _strlit(pos Position, s string) *strlit { return &strlit{valbase{pos},s} }
func makeDate(pos Position, s time.Time) *Date  { return &Date{datetime{valbase{pos},s}} }
func makeTime(pos Position, t time.Time) *Time  { return &Time{datetime{valbase{pos},t}} }
func makeUrl(pos Position, s *neturl.URL) *url {
    var host, port string
    var v = strings.Split(s.Host, ":")
    if len(v) == 1 { host = v[0] }
    if len(v) == 2 { host, port = v[0], v[1] }

    var password Value
    if t, y := s.User.Password(); y {password = _strlit(pos, t)}
    if s.RawQuery != "" { panic("url query: "+s.RawQuery) }
    return &url{ // FIXME: calculate component positions
        valbase: valbase{pos},
        Scheme:   _strlit(pos, s.Scheme),
        Username: _strlit(pos, s.User.Username()),
        Password: password,
        Host:     _strlit(pos, host),
        Port:     _strlit(pos, port),
        Path:     _strlit(pos, s.Path),
        // Query:    _strlit(pos, s.RawQuery),
        Fragment: _strlit(pos, s.Fragment),
    }
}
func _word(pos Position, w string) *word { return &word{valbase{pos},w} }
func _compound(elems ...Value) *compound { return &compound{elements{elems}} }
func _strcomp(elems ...Value) *strcomp { return &strcomp{elements{elems}} }
func _list(elems ...Value) *list { return &list{elements{elems}} }
func list_t[T Value](ii ...T) *list {
    var elems []Value
    for _, i := range ii { elems = append(elems, i) }
    return &list{elements{elems}}
}
func _group(pos Position, elems ...Value) (v *group) { return &group{valbase{pos},elements{elems}} }
func _globmeta(pos Position, tok token) *globmeta { return &globmeta{valbase{pos},tok} }
func _globrange(val Value) *globrange { return &globrange{val} }
func _globpat(elems ...Value) *globpat { return &globpat{elements{elems}} }

func makePair(k, v Value) (p *pair) { return &pair{k, v} }
func makePath(segments ...Value) *path { return &path{elements{segments}} }
func makePunct(pos Position, t token) *punct { return &punct{valbase{pos},t} }
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

func Make(pos Position, in any) (out Value) {
    switch v := in.(type) {
    case int:       out = _decimal(pos,int64(v))
    case int32:     out = _decimal(pos,int64(v))
    case int64:     out = _decimal(pos,v)
    case float32:   out = _float(pos,float64(v))
    case float64:   out = _float(pos,v)
    case string:    out = _strlit(pos, v)
    case time.Time: out = &datetime{valbase{pos},v} // FIXME: NewDate, NewTime
    case Value:     out = v
    }
    return
}
func MakeAll(pos Position, in... any) (out []Value) {
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
        return _binary(pos,i)
    } else {
        panic(e)
    }
}

func ParseOctal(pos Position, s string) *octal {
    if strings.HasPrefix(s, "0") {
        s = s[1:]
    }
    if i, e := strconv.ParseInt(s, 8, 64); e == nil {
        return _octal(pos,i)
    } else {
        panic(e)
    }
}

func ParseDecimal(pos Position, s string) *decimal {
    if i, e := strconv.ParseInt(s, 10, 64); e == nil {
        return _decimal(pos,i)
    } else {
        panic(e)
    }
}

func ParseHexadecimal(pos Position, s string) *hexadecimal {
    if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
        s = s[2:]
    }
    if i, e := strconv.ParseInt(s, 16, 64); e == nil {
        return _hexadecimal(pos,i)
    } else {
        panic(e)
    }
}

func parseFloat(pos Position, s string) *float {
    if f, e := strconv.ParseFloat(strings.Replace(s, "_", "", -1), 64); e == nil {
        return _float(pos,f)
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

func ParseURL(pos Position, s string) *url {
    if u, e := neturl.Parse(s); e == nil {
        return makeUrl(pos,u)
    } else {
        panic(e)
    }
}

const (
    max_evoke = 999
)

// NOTE: evokeTraceDots is for debugging call trace, if this finally goes into a formal
//       feature, it should need a sync-lock protection.
var evokeTraceDots string

type prerequisite_evoke_loop struct{ Context ; Value }

type evoke_builtin struct{ name string }
type evoke_def     struct{ name string }
type evoke_x       struct{ name string }
type evoke_detect_loop struct{ Value }
type evoke_count       struct{}

type evocation struct{
    automatic
    x   Value
    a []Value
    o []Value
}
func (p *evocation) inner() Context { return &p.automatic }
func (p *evocation) cast(t reflect.Type) Context {
    if reflect.TypeOf((*automatic)(nil)) == t { return &p.automatic }
    if reflect.TypeOf(p) == t { return p }
    return p.automatic.cast(t)
}
func (p *evocation) ts(t string) string {
    if true {
        var s = p.defs.String()
        if s != "" { s += " " }
        return "{="+t+" "+p.x.String()+" "+s+ts(p.Context)+"}"
    } else if false {
        return "{="+t+" "+p.x.String()+" "+ts(&p.automatic)+"}"
    } else {
        return "{="+t+" "+p.x.String()+" "+ts(p.Context)+"}"
    }
}
func (p *evocation) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case evoke_builtin:
        if x, y := p.x.(*builtin); y && (t.name == "" || t.name == x.name) {
            return x
        }
    case evoke_def:
        if x, y := p.x.(*def); y && (t.name == "" || t.name == x.name) {
            return x
        }
    case evoke_x:
        if p.x != nil && (t.name == "" || t.name == p.x.ident(ctx)) {
            return p.x
        }

    case evoke_count:
        u, _ := p.Context.do(ctx, t).(uint)
        return 1 + u

    case evoke_detect_loop:
        if t.Value == p.x {
            switch p.x.(type) {
            case *auto, *def: return true
            }
        }
        if u, y := do(ctx, evoke_count{}).(uint); y && u > 255 {
            errostack(pc(ctx,p.x), 16, "%s : %s", p.x, ts(p.x)).trace()
        }

    case get_position:
        var pos Position
        if p.x != nil { pos = p.x.Position() }
        if !pos.valid() && p.a != nil && p.a[0] != nil { pos = p.a[0].Position() }
        if !pos.valid() && p.o != nil && p.o[0] != nil { pos = p.o[0].Position() }
        if  pos.valid()  {  return pos  }
    }
    return p.automatic.do(ctx, op)
}

type evoker interface{ evoke(*evocation) Value }
func evoke(ctx Context, x Value, o, a []Value) (res Value, _ []Value) {
    if truly(ctx, evoke_detect_loop{x}) {
        switch {
        case truly(ctx, evoke_loop_null{}): return _null(x.Position()), a
        case truly(ctx, evoke_loop_panic{}): panic(trace_evoke_loop_err{ctx, x})
        }
        if false { errostack(pc(ctx,x), 32, "evoke loop: %v", x).trace() }
        if false { note(pc(ctx,x), "evoke loop: %v", x).debug(64) }
        return _null(_position(ctx)), a
    }
    if t, y := x.(evoker); y {
        // NOTE: the evo.a represents the arguments, which is a COPY of the original slice;
        // NOTE: making a COPY of the argument slice FIXES the bug of delegate-altered-args.
        e := evocation{automatic{Context:ctx, defs:make(defmap)}, x, copyvals(a), copyvals(o)}
        return t.evoke(&e), e.a
    } else {
        return x.expand(ctx), a
    }
}

type opt  struct{ Value }
type opts struct{ vals []Value }

func call(ctx Context, name string, o []Value, a ...Value) (res Value) {
    if v := _universe(ctx).lookup(name); v != nil {
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

// FIXME: duplicate is still buggy
func buggy__duplicate(v []Value) []Value {
    return _duplicate(reflect.ValueOf(v)).Interface().([]Value)
}

func _duplicate(v reflect.Value) reflect.Value {
    switch v.Kind() {
    case reflect.Slice, reflect.Array:
        // Allocate a new slice/array with same length and capacity
        dst := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
        // Recursively duplicate elements
        for i := 0; i < v.Len(); i++ {
            if elem := v.Index(i) ; !elem.IsNil() {
                dst.Index(i).Set(_duplicate(elem))
            }
        }
        return dst
    case reflect.Int, reflect.String, reflect.Bool:
        // Simple value types can be copied directly
        return reflect.ValueOf(v.Interface())
    case reflect.Ptr/* , reflect.Interface */:
        // Duplicate pointers by creating a new pointer and copying the underlying value
        elem := reflect.New(v.Type().Elem())
        elem.Elem().Set(_duplicate(v.Elem()))
        return elem
    case reflect.Interface:
        elem := reflect.New(v.Type()).Elem()
        elem.Set(_duplicate(v.Elem()))
        return elem
    case reflect.Struct:
        // Duplicate structs by creating a new struct and copying fields
        newStruct := reflect.New(v.Type()).Elem()
        for i := 0; i < v.NumField(); i++ {
            field := v.Field(i)
            newField := newStruct.Field(i)
            newField.Set(_duplicate(field)) // Recursively duplicate nested values
        }
        return newStruct
    default:
        i := v.Interface()
        panic(fmt.Sprintf("Unsupported value type for duplication: %T %v : %s", i, v.Kind(), ts(i)))
    }
}
