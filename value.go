//
//  Copyright (C) 2012-2025, Duzy Chan <code@extbit.io>, all rights reserverd.
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
	"sort"
    "time"
    "unsafe"
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
    KindUndef Kind = 1<<(iota-1)
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

    KindConjunction
    KindDisjunction
    KindPair
    KindPath
    KindPunct
    KindQuote
    KindUrl
    KindCompound
    KindGlobPat
    KindRecipe

    KindObject
    KindKnownObject
    KindUndetermined
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

    KindValues
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

func sortstrs(s []string) []string { sort.Strings(s) ; return s }
func sfmt(f string, i ...any) string { return fmt.Sprintf(f, i...) }
func ssfmt(ss []string, a ...any) (res []string) {
    for _, s := range ss { res = append(res, fmt.Sprintf(s, a...)) }
    return
}

type origin_def struct{ name string }
type get_origin struct{}
type ex_def_0   struct{}
type ex_def_1   struct{}
type ex_def_2   struct{}
type ex_def_3   struct{}
type ex_closure struct{}

// Original initiation of def values.
type original struct{ Context ; o origin }
func (c original) inner() Context { return c.Context }
func (c original) cast(t reflect.Type) Context { return icast(c, t) }
func (c original) ts(t string) string { return "{="+t+" "+c.o.String()+" "+ts(c.Context)+"}" }
func (c original) do(ctx Context, op any) any {
    switch t := op.(type) {
    case get_origin  : return c.o
    case     origin  : return c.o&t != 0
    // case ex_????  : return c.o&(defExpand1|defExpand2|defExpand3) != 0
    case ex_closure  : return c.o&(           defExpand2|defExpand3) != 0
    case ex_def_0    : return c.o&(defExpand0) != 0
    case ex_def_1    : return c.o&(defExpand1) != 0
    case ex_def_2    : return c.o&(defExpand2) != 0
    case ex_def_3    : return c.o&(defExpand3) != 0
    }
    return c.Context.do(ctx, op)
}

// Optimize value for final strings
type final struct{ Context }
func (c final) inner() Context { return c.Context }
func (c final) cast(t reflect.Type) Context { return icast(c, t) }
func (c final) ts(t string) string { return fmt.Sprintf("{=%s %v}", t, ts(c.Context)) }
func (c final) do(ctx Context, op any) any {
    switch op.(type) {
    case ex_closure: return true
    case final: return c
    }
    return c.Context.do(ctx, op)
}
func _final(ctx Context) Context {
    c, y := do(ctx, final{}).(final)
    if !y { c.Context = ctx }
    return c
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
func (c partial) ts(t string) string { return fmt.Sprintf("{=%s %b %v}", t, c.bit, ts(c.Context)) }
func (c partial) do(ctx Context, op any) any {
    if x, y := op.(is_good_with) ; y {
        for _, v := range x.a {
            switch t := v.(type) {
            case *closure : return c.do(ctx, op)
            case *delegate: return c.do(ctx, op)
            case *auto:
                if c.bit&placeholderPart != 0 && t.isPlaceholder() { return true }
                if c.bit&digitalPart != 0 && t.isDigit() { return true }
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
            if s := bases(x.abs, "testdata", 2, true); s == c.comment {
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
        if d := s.finddef(name); d != nil {
            prev = d.value
            d.set(ctx, val)
            okay = true
            break
        }
    }
    return
}

func closure_files(ctx Context, name string, one bool) (res []*file) {
    var a = unmap_files(ctx, _project(ctx), name, nil)
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

func defs(ctx Context, val any, s ...string) (res []*def) {
    switch p := val.(type) {
    case []Value:
        for _, v := range p { res = append(res, defs(ctx, v, s...)...) }
    case *loc:
        res = append(res, defs(ctx, p.Value, s...)...)
    case *barefile:
        res = append(res, defs(ctx, p.Value, s...)...)
    case *globrange:
        res = append(res, defs(ctx, p.Value, s...)...)
    case *percpat:
        res = append(res, defs(ctx, p.Prefix, s...)...)
        res = append(res, defs(ctx, p.Suffix, s...)...)
    case *globpat:
        res = append(res, defs(ctx, p.elems, s...)...)
    case *strcomp:
        res = append(res, defs(ctx, p.elems, s...)...)
    case *path:
        res = append(res, defs(ctx, p.elems, s...)...)
    case *group:
        res = append(res, defs(ctx, p.elems, s...)...)
    case *list:
        res = append(res, defs(ctx, p.elems, s...)...)
    case *compound:
        res = append(res, defs(ctx, p.elems, s...)...)
    case *argumented:
        res = append(res, defs(ctx, p.Value, s...)...)
        res = append(res, defs(ctx, p.args, s...)...)
    case *pair:
        res = append(res, defs(ctx, p.key, s...)...)
        res = append(res, defs(ctx, p.val, s...)...)
    case *arrow:
        res = append(res, defs(ctx, p.o, s...)...)
        res = append(res, defs(ctx, p.s, s...)...)
    case *closure:
        res = append(res, defs(ctx, p.x, s...)...)
        res = append(res, defs(ctx, p.o, s...)...)
        res = append(res, defs(ctx, p.a, s...)...)
    case *delegate:
        res = append(res, defs(ctx, p.x, s...)...)
        res = append(res, defs(ctx, p.o, s...)...)
        res = append(res, defs(ctx, p.a, s...)...)
    case *undetermined:
        res = append(res, defs(ctx, p.identifier, s...)...)
        res = append(res, defs(ctx, p.value, s...)...)
    case *use:
        res = append(res, defs(ctx, p.params, s...)...)
    case *uselist:
        res = append(res, defs(ctx, p.list, s...)...)
    case *auto:
        if d := p.def(ctx); d != nil { res = append(res, defs(ctx, d, s...)...) }
    case *url:
        if v := p.Scheme;   v != nil { res = append(res, defs(ctx, v, s...)...) }
        if v := p.Username; v != nil { res = append(res, defs(ctx, v, s...)...) }
        if v := p.Password; v != nil { res = append(res, defs(ctx, v, s...)...) }
        if v := p.Host;     v != nil { res = append(res, defs(ctx, v, s...)...) }
        if v := p.Port;     v != nil { res = append(res, defs(ctx, v, s...)...) }
        if v := p.Path;     v != nil { res = append(res, defs(ctx, v, s...)...) }
        if v := p.Query;    v != nil { res = append(res, defs(ctx, v, s...)...) }
        if v := p.Fragment; v != nil { res = append(res, defs(ctx, v, s...)...) }
    case *rule:
        res = append(res, defs(ctx, p.target, s...)...)
        for _, p := range p.program {
            res = append(res, defs(ctx, p.depends, s...)...)
            res = append(res, defs(ctx, p.recipes, s...)...)
        }
    case *def:
        if len(s) == 0 {
            res = append(res, p)
        } else {
            for _, s := range s {
                if s == p.name {
                    res = append(res, p)
                    break
                }
            }
        }
        res = append(res, defs(ctx, p.value, s...)...)
    default:
        erro(pc(ctx,p), "defs: %v : %v", p, ts(p)).trace()
    }
    return
}

func refdef(ctx Context, val Value, origin origin) (res bool) {
    for _, def := range defs(ctx, val) {
        if def.o == origin { return true }
        if true && def.value != nil && refdef(ctx, def.value, origin) { return true }
    }
    return
}

func entryIndicator(ctx Context, entry Value) (str, ent, tar string) {
    if !isNull(entry) { ent = __string(ctx, entry) }
    if val := auto_get(ctx, "@"); val == nil || isTrivial(val) {
        str = ent // ...
    } else if tar = __string(ctx, val); ent != tar {
        str = fmt.Sprintf("%s(%s)", ent, tar)
    } else {
        str = ent
    }
    return
}

func exists(ctx Context, v Value) bool {
    // FIXME: returns true if existenceMatterless ??
    return v != nil && statFile(ctx, v).exists() == existenceConfirmed
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

    fmt.Fprintf(key, "%s", __string(ctx, target))

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
        if fs = stampFile(ctx, target); fs != nil {
            if false { reportFileUpdates(ctx, fs) }
        }
    }
    return
}

type as struct{ Value }
func (a as) ts(ctx Context, _ string) string { return "{=as "+ts(a.Value,ctx)+"}" }
func (a as) file(ctx Context, projs ...*project) (res *file) {
    var v = scalarize(expand(ctx,a.Value))
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
    if _, s, y = a.file_fullname(ctx, projs...); !y { s = __string(ctx, a) }
    return
}
func (a as) fullname(ctx Context, projs ...*project) (res fullname) {
    if a.Value != nil {
        if f := a.file(ctx, projs...); f != nil {
            res.Value = f
        } else {
            errostack(pc(ctx,a), 3, "nil file : %v : %v → %v", a.Value, ts(a.Value), expand(ctx,a.Value)).trace()
        }
    }
    return
}

func splitpath(ctx Context, a any) (ss []string) {
	switch t := a.(type) {
	case []any   : for _, t := range t { ss = append(ss, splitpath(ctx, t)...) }
	case []string: for _, t := range t { ss = append(ss, strings.Split(t, pathSep)...) }
	case   string: ss = strings.Split(t, pathSep)
	case    Value: ss = strings.Split(__string(ctx, t), pathSep)
	case      nil: break
	default: erro(ctx, "%v | %v", a, ts(a)).trace()
	}
	return
}

// joinpath is different from filepath.Join, which trims and discards empty segments
func joinpath(segs ...string) string { return strings.Join(segs, pathSep) }
func joinp(ctx Context, a any) string { s, _ := _joinpath(ctx, a); return s }
func _joinpath(ctx Context, a any) (s string, ss []string) {
	ss = splitpath(ctx, a)
	s = strings.Join(ss, pathSep)
	return
}

func joinraws(sep string, vals ...*raw) string {
    var strs []string
    for _, v := range vals { strs = append(strs, v.String()) }
    return strings.Join(strs, sep)
}

func posstr(s string) string {
    s = bases(s, 3, true)
    for _, x := range []string{"/testdata/","/smart/"} {
        if i := strings.Index(s, x); i > 0 { s = "…/"+s[i+len(x):]; break }
    }
    return s
}

type posctx struct{ Context ; pos any }
func (p *posctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *posctx) inner() Context { return p.Context }
func (p *posctx) ts(string) string {
    switch t := p.pos.(type) {
    case Position:
        var s = posstr(t.Filename)
        if t.Column == 0 { return fmt.Sprintf("{%s:%d %s}", s, t.Line, ts(p.Context)) }
        return fmt.Sprintf("{%s:%d:%d %s}", s, t.Line, t.Column, ts(p.Context))
    default:
        return fmt.Sprintf("{%v %s}", t, ts(p.Context))
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
        case  *file: a = t.fullname()
        }
        switch t := a.(type) {
        case *loc: return &posctx{pc(ctx,t.Value),t.pos}
        case  *scanner   : p = t.pos(n...)
        case  *parser    : p = t.Position()
        case   Position  :  if t.valid() { p = t }
        case   positioner:  if t != nil  { p = t.Position() }
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
            if 0 == len(n) { pos.Line   = 1 }
            if 0 <  len(n) { pos.Line   = n[0] }
            if 1 <  len(n) { pos.Column = n[1] }
            p = pos
        }
    }
    if p != nil { ctx = &posctx{ctx,p} }
    return ctx
}

type source struct{}
type srcfun struct{}
type srcpos struct{}
type srcctx struct{ posctx ; f *runtime.Func ; a any }
func (p *srcctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *srcctx) inner() Context { return &p.posctx }
func (p *srcctx) ts(t string) string { return "{=src " + p.f_name() + " " + p.posctx.ts(t) + "}" }
func (p *srcctx) f_name() string {
    return strings.TrimPrefix(strings.TrimPrefix(p.f.Name(),"extbit.io/"),"smart.")
}
func (p *srcctx) do(ctx Context, op any) any {
    switch t := op.(type) {
    case get_position: return nil
    case srcpos: return p.pos
    case srcfun: return p.f
    case source: return srctag(p.posctx.do(ctx, get_position{}).(Position))
    case origin_def:
        if x, y := p.a.(*def); y {
            if t.name == "" || t.name == x.name { return x }
        }
    }
    return p.posctx.do(ctx, op)
}

var loader_pos Position
var loader_src string

func src(ctx Context, a any) Context {
    if pc, file, line, ok := runtime.Caller(1); ok {
        p := Position{}
        p.Filename, p.Line = file, line
        if a == nil && loader_pos.Line == 0 && loader_pos.Filename == "" {
            loader_pos, loader_src = p, srctag(p)
        }
        return &srcctx{posctx{ctx,p},runtime.FuncForPC(pc),a}
    }
    return ctx
}

func srctag(p Position) string {
    if s := filepath.Base(p.Filename); p.Column == 0 {
        return fmt.Sprintf("%s:%d", s, p.Line)
    } else {
        return fmt.Sprintf("%s:%d:%d", s, p.Line, p.Column)
    }
}

type positioner interface{ Position() Position }
type identer    interface{ ident(Context) string }

type set_opt_ident  struct{}
type closure_ident  struct{}
type delegate_ident struct{}
type ident_ctx struct{ Context ; nil int }
func (c *ident_ctx) inner() Context { return c.Context }
func (c *ident_ctx) do(ctx Context, op any) (_ any) {
    switch t := op.(type) {
    case delegate_ident: c.nil = 1; return
    case  closure_ident: c.nil = 2; return
    case  set_opt_ident: c.nil = 3; return
    case opt_ident: if c.nil > 0 { return true }
    case ident_ctx: switch t.Context { case nil, c, c.Context: return c }
    }
    return c.Context.do(ctx, op)
}
func identity(ctx Context) (*ident_ctx, Context) {
    x, _ := do(ctx, ident_ctx{}).(*ident_ctx)
    return x, ctx
}
func identity_ctx(ctx Context) (ic *ident_ctx, rc Context) {
    if ic, rc = identity(ctx); ic == nil {
        ic = &ident_ctx{ctx, 0}
        rc = ic
    }
    return
}
func ident(ctx Context, x Value) (s string) {
    switch t := x.(type) {
    case *loc: return ident(ctx,t.Value)
    case *closure: return ident_opt(ctx, "&", x, closure_ident{})
    case *delegate: return ident_opt(ctx, "$", x, delegate_ident{})
    case *file: return t.filestub.name
    case *punct: return t.token.String()
    case *project: return t.name
    case *uselist: return t.name
    case *word: return t.s
    case *raw: return t.s
    case *strlit: return `'`+t.s+`'`
    case *valbase, *null, *none, nil: return
    case *globmeta: return t.token.String()
    case *answer, *boolean, *binary, *octal, *decimal, *hexadecimal, *float, *datetime, *Date, *Time, *globrange:
        return t.String()
    case *arrow: return ident(ctx,t.o) + t.t.String() + ident(ctx,t.s)
    case *rule: return ident(ctx,t.target)
    case *compound:
        for _, t := range t.elems {
            if x, ok := t.(*list); ok && x.len() > 0 {
                s += "⌜"+ident(ctx, t)+"⌟"
            } else {
                s += ident(ctx, t)
            }
        }
        return
    case *globpat:
        for _, t := range t.elems { s += ident(ctx, t) }
        return
    case *strcomp:
        for _, t := range t.elems { s += ident(ctx, t) }
        return `"` + s + `"`
    case *list:
        for i, t := range t.elems { if 0 < i { s += " " }; s += ident(ctx, t) }
        return
    case *group:
        for i, t := range t.elems { if 0 < i { s += " " }; s += ident(ctx, t) }
        return "(" + s + ")"
    case *pair:
        if k := t.key; k != nil { s  = ident(ctx, k) }; s += "="
        if v := t.val; v != nil { s += ident(ctx, v) }
        return
    case *undetermined: return __string(ctx, t.identifier)
    case *disjunction: return "{" + ident(ctx, t.val) + "}"
    case self: return ".self"
    case flag: return "-" + ident(ctx, t.Value)
    case negative: return "!" + ident(ctx, t.Value)
    case identer: return t.ident(ctx)
    case *url:
        if s = ident(ctx, t.Scheme) + ":"; t.Host != nil {
            if s += "//"; t.Username != nil { s += ident(ctx, t.Username) + "@" }
            if s += ident(ctx, t.Host); t.Port != nil { s += ":" + ident(ctx, t.Port) }
        }
        if t.Path != nil { s += ident(ctx, t.Path) }
        if t.Query != nil {
            s += "?" + ident(ctx, t.Query[0])
            for _, q := range t.Query[1:] { s += "&" + ident(ctx, q) }
        }
        if t.Fragment != nil { s += "#" + ident(ctx, t.Fragment) }
        return
    default:
        erro(pc(ctx,x), "todo: ident for %s", ts(x)).trace()
        return
    }
}

func ident_opt(ctx Context, pre string, x Value, op any) (_ string) {
    if truly(ctx, ex_closure{}) {
        var ic *ident_ctx
        if ic, ctx = identity(ctx); ic.nil == 0 {
            if v := expand(ctx,x); ic.nil == 0 {
                do(ctx, set_opt_ident{})
                return ident(ctx, v)
            }
        }
    }
    if op != nil { do(ctx, op) }
    return x.(interface{ id(Context,string) string }).id(ctx, pre)
}

func fnv1(ctx Context, t any, a ...any) (_ uint64) {
    var h = fnv.New64()
    var o = binenc.LittleEndian
    if t != nil {
        var b []byte
        switch x := t.(type) {
        case interface{ kind() Kind }:
            b = make([]byte, 8)
            o.PutUint64(b, uint64(x.kind()))
        case Kind:
            b = make([]byte, 8)
            o.PutUint64(b, uint64(x))
        default:
            b = []byte(typeof(x))
        }
        h.Write(b)
    }
    for _, v := range a {
        var b []byte
        switch t := v.(type) {
        case Kind:
            b = o.AppendUint64(b, uint64(t))
        case token:
            b = o.AppendUint64(b, uint64(t))
        case int64:
            b = o.AppendUint64(b, uint64(t))
        case uint64:
            b = o.AppendUint64(b, uint64(t))
        case string:
            b = []byte(t)
        case []string:
            for _, s := range t { b = append(b, []byte(s)...) }
        case Value:
            if t != nil {
                if t.kind()&(KindBoolean|KindInteger|KindFloat|KindDateTime) != 0 {
                    b = o.AppendUint64(b, uint64(__int(ctx, t)))
                } else {
                    b = []byte(t.String()) // BUG: t.string(ctx)
                }
            }
        case []Value:
            for _, t := range t {
                if t.kind()&(KindBoolean|KindInteger|KindFloat|KindDateTime) != 0 {
                    b = o.AppendUint64(b, uint64(__int(ctx, t)))
                } else {
                    b = []byte(t.String()) // BUG: t.string(ctx)
                }
            }
        default:
            erro(ctx, "fnv1: unsupported type : %s", ts(v)).trace()
        }
        h.Write(b)
    }
    return h.Sum64()
}

func hash(ctx Context, x Value) (u uint64) {
    switch p := x.(type) {
    case *loc: return hash(ctx, p.Value)
    case *returner: return fnv1(ctx, p, p.vals)
    case *argumented: return fnv1(ctx, p, p.Value, p.args)
    case *list: return fnv1(ctx, p, p.any()...)
    case *quote: return fnv1(ctx, p, p.any()...)
    case *strcomp: return fnv1(ctx, p, p.any()...)
    case *compound: return fnv1(ctx, p, p.any()...)
    case *group: return fnv1(ctx, p, p.any()...)
    case *path: return fnv1(ctx, p, p.any()...)
    case *pair: return fnv1(ctx, p, p.key, p.val)
    case *escaped: return fnv1(ctx, p, p.s)
    case *disjunction: return fnv1(ctx, p.kind(), p.val)
    case *barefile: return fnv1(ctx, p.kind(), p.Value)
    case *globrange: return fnv1(ctx, p.kind(), p.Value)
    case *globmeta: return fnv1(ctx, p.kind(), p.token.String())
    case *project: return fnv1(ctx, p, p.name)
    case *auto: return fnv1(ctx, p, p.name)
    case *builtin: return fnv1(ctx, p, p.name)
    case *def: return fnv1(ctx, p, p.name, p.value)
    case *rule: return fnv1(ctx, p, p.target)
    case *file: return fnv1(ctx, p, p.fullname())
    case *modification: return fnv1(ctx, p, p.list)
    case *modifier: return fnv1(ctx, p, p.any()...)
    case *plain: return fnv1(ctx, p, p.any()...)
    case *plainline: return fnv1(ctx, p, p.any()...)
    case *globpat: return fnv1(ctx, p, p.any()...)
    case *percpat: return fnv1(ctx, p, p.Prefix, p.Suffix)
    case *regexpat: return fnv1(ctx, p, p.Regexp.String())
    case *exec_result: return fnv1(ctx, p, p.values)
    case *arrow: return fnv1(ctx, p, p.t, p.o, p.s)
    case *closure: return fnv1(ctx, p, p.l, p.x, p.o, p.a)
    case *delegate: return fnv1(ctx, p, p.l, p.x, p.o, p.a)
    case *use: return fnv1(ctx, p, p.project)
    case *uselist: return fnv1(ctx, p, p.list)
    case *undetermined: return fnv1(ctx, p, p.token, p.identifier, p.value)
    case flag: return fnv1(ctx, p, p.Value.kind(), p.Value)
    case fullname: return fnv1(ctx, p, p.Value)
    case negative: return fnv1(ctx, p, p.Value)
    case conjunction: return fnv1(ctx, p.kind(), p.sep, p.list)
    }
    return fnv1(ctx, x.kind(), x)
}

type opt_ident struct{}
type opt_ident_ctx struct{ Context }
func (c *opt_ident_ctx) inner() Context { return c.Context }
func (c *opt_ident_ctx) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case opt_ident_ctx: return c
	case opt_ident: return true
    }
    return c.Context.do(ctx, op)
}
func optional_ident(ctx Context) Context {
    if do(ctx, opt_ident_ctx{}) == nil { ctx = &opt_ident_ctx{ctx} }
    return ctx
}
func optional(v Value) (_ bool) {
    switch t := v.(type) {
    case *loc: return optional(t.Value)
    case *arrow: return optional(t.o) || optional(t.s)
    case *list: for _, e := range t.elems { if optional(e) { return true } }
    case *compound: for _, e := range t.elems { if optional(e) { return true } }
	case *globpat:
		if i := t.len()-1; 0 < i {
			if x, y := t.elems[i].(*globmeta); y {
				return x.token == QUE
			}
		}
    }
    return
}

type Value interface{
    positioner // The position where the value appears (or NoPos).
    kind() Kind
    String() string
}

func typeof(arg any) (s string) {
    defer func() { if s == "" { panic(fmt.Sprintf("empty typeof: %T", arg)) } } ()

    if arg == nil { return "nil" }

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

func unbox(a any) any {
    switch t := a.(type) {
    case self: return t.project
    case *loc: return unbox(t.Value)
    case *list: if len(t.elems) == 1 { return unbox(t.elems[0]) }
	// case *pair: return &pair{unbox(t.key).(Value), unbox(t.val).(Value)}
    case flag: return flag{unbox(t.Value).(Value)}
    }
    return a
}

func unbox2(a any, b int) (any, int) {
    switch t := a.(type) {
    case self: return t.project, 0
    case *loc: return unbox2(t.Value, b)
    case *list: if t.len() == 1 { return unbox2(t.elems[0], b) }
	// case *pair: return &pair{unbox(t.key).(Value), unbox(t.val).(Value)}, 0
    case *boolean: return t.bool, 0
    case *word: return t.s, 0
    case *raw: return t.s, 0
    case *binary: return t.int64, 2
    case *octal: return t.int64, 8
    case *decimal: return t.int64, 10
    case *hexadecimal: return t.int64, 16
    case *float: return t.float64, 0
    case *datetime: return t.t, 0
    case *Date: return t.t, 1
    case *Time: return t.t, 2
    case flag:
        a2, b2 := unbox2(t.Value, b)
        switch t := a2.(type) {
        case int64: return -t, b2
        case float64: return -t, 0
        }
    }
    return a, b
}

const cmp_compound_flag = true

func cmp(ctx Context, l, r any) (res cmpres) {
    if checkpoints { defer check_cmp(ctx, l, r, &res) }
    var lv, lb = unbox2(l, 0)
    var rv, rb = unbox2(r, 0)
    switch x := lv.(type) {
	case []Value:
		switch y := rv.(type) {
		case []Value:
			var ( i int ; _x Value )
			for i, _x = range x {
				if i < len(y) {
					if t := cmp(ctx, _x, y[i]); t != cmpEqual {
						return t
					}
				} else {
					return cmpGreater
				}
			}
			if i == len(x)-1 || (i == 0 && len(x) == 0 && len(x) == len(y)) {
				return cmpEqual
			}
		}
    case string:
        switch y := rv.(type) {
        case string: return cmp_string(x, y)
        case *punct: return cmp_string(x, y.String())
        case int64  : if i, e := strconv.ParseInt(x, lb, 64); e == nil { return cmp_int(i, y) }
        case float64: if i, e := strconv.ParseFloat(x, 64); e == nil { return cmp_float(i, y) }
        case time.Time:
            switch rb {
            case 0: if t, e := parseDateTime(x); e == nil { return cmp_time(t, y) }
            case 1: if t, e := parseDate(x); e == nil { return cmp_time(t, y) }
            case 2: if t, e := parseTime(x); e == nil { return cmp_time(t, y) }
            }
        }
    case int64:
        switch y := rv.(type) {
        case string: if i, e := strconv.ParseInt(y, rb, 64); e == nil { return cmp_int(x, i) }
        case int64  : return cmp_int(x, y)
        case float64: return cmp_int(x, int64(y))
        }
    case float64:
        switch y := rv.(type) {
        case string: if f, e := strconv.ParseFloat(y, 64); e == nil { return cmp_float(x, f) }
        case int64  : return cmp_float(x, float64(y))
        case float64: return cmp_float(x, y)
        }
    case time.Time:
        switch y := rv.(type) {
        case time.Time: return cmp_time(x, y)
        case string:
            switch lb {
            case 0: if t, e := parseDateTime(y); e == nil { return cmp_time(x, t) }
            case 1: if t, e := parseDate(y); e == nil { return cmp_time(x, t) }
            case 2: if t, e := parseTime(y); e == nil { return cmp_time(x, t) }
            }
        }
    case *punct:
        switch y := rv.(type) {
        case string: return cmp_string(x.String(), y)
        case *punct:
            if x.token == y.token {
                return cmpEqual
            } else if s1, s2 := x.String(), y.String(); s1 < s2 {
                return cmpSmaller
            } else {
                return cmpGreater
            }
        }
    case *compound:
		switch y := rv.(type) {
		case *compound: return cmp(ctx, x.elems, y.elems)
		case flag:
			if cmp_compound_flag && x.len() == 2 {
				if x0, ok := unbox(x.elems[0]).(flag); ok {
					switch t0 := x0.Value.(type) {
					case *valbase, *null: 
						return cmp(ctx, x.elems[1], y.Value)
					case *word:
						switch t1 := unbox(x.elems[1]).(type) {
						case *word:
							switch t := y.Value.(type) {
							case *word:
								switch sx := t0.s + t1.s; {
								case sx == t.s: return cmpEqual
								case sx < t.s: return cmpSmaller
								case sx > t.s: return cmpGreater
								}
							default:
								erro(pc(ctx,l), "%v %v", ts(l), ts(r)).trace()
							}
						default:
							erro(pc(ctx,l), "%v %v", ts(l), ts(r)).trace()
						}
					default:
						erro(pc(ctx,l), "%v %v", ts(l), ts(r)).trace()
					}
				}
			}
		}
    case flag:
		switch y := rv.(type) {
		case flag: return cmp(ctx, x.Value, y.Value)
		case *compound:
			if cmp_compound_flag && y.len() == 2 {
				if y0, ok := unbox(y.elems[0]).(flag); ok {
					switch t0 := y0.Value.(type) {
					case *valbase, *null: 
						return cmp(ctx, x.Value, y.elems[1])
					case *word:
						switch t1 := unbox(y.elems[1]).(type) {
						case *word:
							switch t := x.Value.(type) {
							case *word:
								switch sy := t0.s + t1.s; {
								case t.s == sy: return cmpEqual
								case t.s < sy: return cmpSmaller
								case t.s > sy: return cmpGreater
								}
							default:
								erro(pc(ctx,l), "%v %v", ts(l), ts(r)).trace()
							}
						default:
							erro(pc(ctx,l), "%v %v", ts(l), ts(r)).trace()
						}
					default:
						erro(pc(ctx,l), "%v %v", ts(l), ts(r)).trace()
					}
				}
			}
		}
	case *def:
        switch y := rv.(type) {
		case *def: if x.name == y.name { return cmp(ctx, x.value, y.value) }
		}
    case *list:
        switch y := rv.(type) {
        case *list: return cmp(ctx, x.elems, y.elems)
        }
	case *path:
        switch y := rv.(type) {
		case *path: return cmp(ctx, x.elems, y.elems)
		}
	case *globpat:
        switch y := rv.(type) {
		case *globpat: return cmp(ctx, x.elems, y.elems)
		}
	case *globmeta:
        switch y := rv.(type) {
		case *globmeta:
			switch {
			case x.token == y.token: return cmpEqual
			case x.token < y.token: return cmpSmaller
			case x.token > y.token: return cmpGreater
			}
		}
	case *closure:
		switch y := rv.(type) {
		case *closure:
			if cmp(ctx, x.x, y.x) == cmpEqual {
				if cmp(ctx, x.o, y.o) == cmpEqual {
					return cmp(ctx, x.a, y.a)
				}
			}
		}
	case *delegate:
		switch y := rv.(type) {
		case *delegate:
			if cmp(ctx, x.x, y.x) == cmpEqual {
				if cmp(ctx, x.o, y.o) == cmpEqual {
					return cmp(ctx, x.a, y.a)
				}
			}
		}
	case *defcaps:
		switch y := rv.(type) {
		case *defcaps:
			if res = cmp(ctx, x.Value, y.Value); res == cmpEqual {}
		}
    case *loc:
		switch y := rv.(type) {
		case *loc: return cmp(ctx, x.Value, y.Value)
		default: return cmp(ctx, x.Value, y)
		}
    default:
		switch y := rv.(type) {
		case *loc: return cmp(ctx, x, y.Value)
		}
		if false {
			note(pc(pc(ctx,r),l), "%s ⇔ %s | %v ⇔ %v", typeof(lv), typeof(rv), ts(lv), ts(rv)).debug()
		} else if false {
			ctx = pc(pc(ctx,r),l)
			erro(ctx, "%v ⇔ %v | %v ⇔ %v", ts(unbox(l)), ts(unbox(r)), ts(l), ts(r))
			note(ctx, "into: cmp(%s ⇔ %s) | %v ⇔ %v", typeof(lv), typeof(rv), ts(lv), ts(rv)).trace()
		}
    }
    return cmpUnknown
}
func cmp_string(l, r string) cmpres {
	switch {
	case l == r: return cmpEqual
	case l < r: if strings.HasPrefix(r, l) { return cmpLprefix } else { return cmpSmaller }
	case l > r: if strings.HasPrefix(l, r) { return cmpRprefix } else { return cmpGreater }
	}
    return cmpUnknown
}
func cmp_int(l, r int64) cmpres {
    switch {
    case l == r: return cmpEqual
    case l < r: return cmpSmaller
    case l > r: return cmpGreater
    }
    return cmpUnknown
}
func cmp_float(l, r float64) cmpres {
    switch {
    case l == r: return cmpEqual
    case l < r: return cmpSmaller
    case l > r: return cmpGreater
    }
    return cmpUnknown
}
func cmp_time(l, r time.Time) cmpres {
    switch {
    case l.Equal(r) : return cmpEqual
    case l.Before(r): return cmpSmaller
    case l.After(r) : return cmpGreater
    }
    return cmpUnknown
}

func eq(x Context, a, b any) bool { return cmp(x, a, b) == cmpEqual }
func equal(x Context, a, b any, yes ...bool) (res bool) {
	if checkpoints {
		defer func() {
			if !res && sfmt("%v", a) == sfmt("%v", b) {
				erro(pc(x,a), "%v: equal(%v, %v)", res, a, b).trace()
			}
			if res && sfmt("%v", a) != sfmt("%v", b) {
				erro(pc(x,a), "%v: equal(%v, %v)", res, a, b).trace()
			}
		} ()
	}
	if __t(yes...) {
		return __string(x, a) == __string(x, b)
	} else {
		return cmp(x, a, b) == cmpEqual
	}
}

func diff(ctx Context, a, b []Value) bool {
    if len(a) == len(b) {
        for i, v := range a { if !equal(ctx, v, b[i]) { return true } }
        return false
    } else {
        return true
    }
}

func isTrivial(v any) (_ bool) {
    switch a := v.(type) {
    case *valbase, *none, *null, *undef: return true
    case *loc: return isTrivial(a.Value)
    case fullname: return isTrivial(a.Value)
    case as: return isTrivial(a.Value)
	case *list:
		for _, v := range a.elems { if !isTrivial(v) { return } }
		return true
    default: return isNull(a)
    }
    return
}
func isEmpty(v any) (t bool) {
	switch a := v.(type) {
	case *valbase, *none, *null, *undef, nil: return true
	case *loc: return isEmpty(a.Value)
	case *strlit: return a.s == ""
	case *strval: return len(a.v) == 0
	case *strcomp: return isEmpty(a.elems)
	case *compound: return isEmpty(a.elems)
	case *list: return isEmpty(a.elems)
    case *def: return isEmpty(a.value)
	case []Value:
		for _, v := range a { if !isEmpty(v) { return } }
		return true
	}
	return
}
func isUndef(v any) (_ bool) {
    switch a := v.(type) {
	case *loc: return isUndef(a.Value)
    case *def: return a.value != nil && isUndef(a.value)
    case *undef: return true
    }
    return
}
func isNone(v any) (_ bool) {
	switch a := v.(type) {
	case *none: return true
	case *loc: return isNone(a.Value)
	case *list: return len(a.elems) == 0 ||
		(len(a.elems) == 1 && (isNone(a.elems[0]) || isNull(a.elems[0])))
	}
	return
}
func isNull(v any) (_ bool) {
	switch t := v.(type) {
	case *null, nil: return true
	case *loc: return isNull(t.Value)
	case *list: return len(t.elems) == 1 && isNull(t.elems[0])
	}
	return
}

func prefix(ctx Context, x, y Value) (res Value) { // x+y ⇔ prefix+y
    defer check_prefix(ctx, "p", x, y, &res)

	y = unbox(y).(Value)

	switch tx := x.(type) {
	case *loc:
		return &loc{prefix(ctx, tx.Value, y), tx.pos}
	case flag:
		switch tx.Value.(type) {
		case *valbase, *null, *none, nil: return flag{y}
		}
		switch ty := y.(type) {
		case *pair: return &pair{flag{prefix(ctx, tx.Value, ty.key)}, ty.val}
		default: return flag{prefix(ctx, tx.Value, y)}
		}
	case *pair:
		switch tx.val.(type) {
		case *valbase, *null, *none, nil: return &pair{tx.key, y}
		}
		return &pair{tx.key, prefix(ctx, tx.val, y)}
	case *path:
		return &path{elements{append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], y))}}
	case *compound:
		switch ty := y.(type) {
		case *path:
			return &path{elements{append([]Value{prefix(ctx, tx, ty.elems[0])}, ty.elems[1:]...)}}
		case *compound:
			return &compound{elements{append(tx.elems, ty.elems...)}}
		}
		if i := len(tx.elems)-1; 0 <= i {
			switch t := tx.elems[i].(type) {
			case flag:
				if v := prefix(ctx, t, y); 0 == i { return v } else {
					return &compound{elements{append(dup(tx.elems[:i]), v)}}
				}
			}
		}
		return &compound{elements{append(dup(tx.elems), y)}}
	}

	switch ty := y.(type) {
	case *pair:
		switch ty.key.(type) {
		case *valbase, *null, *none, nil: return &pair{x, ty.val}
		default: return &pair{prefix(ctx, x, ty.key), ty.val}
		}
	case *compound:
		switch ty.elems[0].(type) {
		case *pair: erro(ctx, "%v", ty.elems).trace()
		default: return &compound{elements{append([]Value{x}, ty.elems...)}}
		}
	}

    return &compound{elements{[]Value{x, y}}}
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
func (_ *valbase) String() (_ string) { return }

// loc-prefix
func lp(ctx Context, p Position, t string, a ...any) (s string) {
    if ctx != nil {
        var pos = _position(ctx)
        if fn := pos.Filename; fn != "" {
            if f := p.Filename; fn != f && f != "" { s = f + ":" }
        }
    }
    if s += fmt.Sprintf("%d:%d", p.Line, p.Column); t != "" { s += ":" + t }
    for _, a := range a { if a != nil { s += " " + ts(a, ctx) } }
    return
}

type loc struct{ Value ; pos Position }
func (p *loc) Position() Position { return p.pos }
func (p *loc) ts(c Context, t string) string { return "{"+lp(c,p.pos,"",p.Value)+"}" }

func _loc(v Value, p Position) Value {
	if t := v.Position(); t.sameLoc(&p) {
		return v
	} else {
		return &loc{v, p}
	}
}

type returner struct{ valbase ; vals []Value }
func (p *returner) kind() Kind { return KindReturner }

type undef struct{ Value }
func (_ undef) kind() Kind { return KindUndef }
func (p undef) String() string { return "{=undef "+p.Value.String()+"}" }
func (p undef) ts(ctx Context, t string) string { return fmt.Sprintf("{=%s %s}", t, ts(p.Value, ctx)) }

type null struct{ valbase }
func (_ *null) kind() Kind { return KindNull }
func (_ *null) String() string { return "{}" }
func (p *null) ts(ctx Context, t string) string { return "{"+lp(ctx, p.position, t)+"}" }

type none struct{ valbase }
func (_ *none) kind() Kind { return KindNone }
func (p *none) String() (_ string) { return }

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
        return
    }
    return ac.Context.do(ctx, op)
}

type argumented struct{ Value ; args []Value }
func (_ *argumented) kind() Kind { return KindArgumented }
func (p *argumented) ts(ctx Context, t string) (s string) {
    return "{="+t+" "+ts(p.Value)+ts(p.args)+"}"
}
func (p *argumented) String() (s string) {
    for i, a := range p.args {
        if i > 0 { s += "," }
        s += a.String()
    }
    s = fmt.Sprintf("%s(%s)", p.Value.String(), s)
    return
}
func (p *argumented) ctx(ctx Context) *argumented_ctx {
    //!< IMPORTANT! - Don't merge-expand arguments here!
    //!< Arguments should be passed to program.execute as it is.
    var args []Value
    var proj = _project(ctx)

    // NOTE: expand here to avoid args being expanded in the wrong context
    for _, a := range p.args {
        // TODO: deal with pattern args using expandPatterned instead of stenciling:
        if a = expand(_final(ctx),a); patterned(ctx,a) {
            if stems := _stems(ctx); len(stems) > 0 {
                if v, rest := stencil(ctx, a, stems); len(rest) > 0 {
                    erro(pc(ctx,p), "partial stencil: %v, %v, %v, %v", a, v, rest, stems).trace()
                } else if f := (as{a}).file(ctx, proj); f != nil {
                    a = f
                } else {
                    a = v
                }
            }
        }
        args = append(args, a)
    }

    return &argumented_ctx{ctx, p.Value, args}
}

type negative struct{ Value }
func (p negative) String() string { return `!`+p.Value.String() }
func (p negative) ts(ctx Context, t string) string { return "{="+t+" "+ts(p.Value,ctx)+"}" }

type escaped struct{ valbase; s string }
func (_ *escaped) kind() Kind { return KindEscaped }
func (p *escaped) String() string { return "\\" + p.s }

type boolean struct{ valbase; bool }
func (_ *boolean) kind() Kind { return KindBoolean }
func (p *boolean) String() string { if p.bool { return "{=true}" } else { return "{=false}" } }
func (p *boolean) ts(ctx Context, _ string) (s string) {
	if p.bool { s = "true" } else { s = "false" }
    return fmt.Sprintf("{%s}", lp(ctx, p.position, s))
}

type answer struct{ boolean }
func (p *answer) String() string { if p.bool { return "{=yes}" } else { return "{=no}" } }
func (p *answer) ts(ctx Context, _ string) (s string) {
	if p.bool { s = "yes" } else { s = "no" }
    return fmt.Sprintf("{%s}", lp(ctx, p.position, s))
}

type option struct{ boolean }
func (p *option) String() string { if p.bool { return "{=on}" } else { return "{=off}" } }
func (p *option) ts(ctx Context, _ string) (s string) {
	if p.bool { s = "on" } else { s = "off" }
    return fmt.Sprintf("{%s}", lp(ctx, p.position, s))
}

type prediction struct{ boolean ; s string }

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

type binary struct{ integer }
func (p *binary) kind() Kind { return p.integer.kind()|KindBinary }
func (p *binary) String() string { return "0b"+strconv.FormatInt(int64(p.int64),2) }

type octal struct{ integer }
func (p *octal) kind() Kind { return p.integer.kind()|KindOctal }
func (p *octal) String() string { return "0"+strconv.FormatInt(int64(p.int64),8) }

type decimal struct{ integer }
func (p *decimal) kind() Kind { return p.integer.kind()|KindDecimal }
func (p *decimal) String() string { return strconv.FormatInt(int64(p.int64),10) }

type hexadecimal struct{ integer }
func (p *hexadecimal) kind() Kind { return p.integer.kind()|KindHexadecimal }
func (p *hexadecimal) String() string { return "0x"+strconv.FormatInt(int64(p.int64),16) }

const epsilon = 1e-15 /* 1e-16 */

type float struct{ valbase; float64 } // IEEE-754 64-bit binary floating-point
func (p *float) kind() Kind { return KindFloat }
func (p *float) String() string { return strconv.FormatFloat(float64(p.float64),'g', -1, 64) }

type datetime struct{ valbase ; t time.Time }
func (_ *datetime) kind() Kind { return KindDateTime }
func (p *datetime) String() string { return time.Time(p.t).Format("2006-01-02T15:04:05.999999999Z07:00") }

type Date struct{ datetime }
func (p *Date) kind() Kind { return KindDate }
func (p *Date) String() string { return time.Time(p.t).Format("2006-01-02") }

type Time struct{ datetime }
func (p *Time) kind() Kind { return KindTime }
func (p *Time) String() string { return time.Time(p.t).Format("15:04:05.999999999Z07:00") }

// ie. https://en.wikipedia.org/wiki/URL
// ▶▶─<scheme>─(:)┬──────────────────────────────────────┬<path>┬───────────┬┬──────────────┬─▶◀
//                └(//)┬──────────────┬<host>┬──────────┬┘      └(?)─<query>┘└(#)─<fragment>┘
//                     └<userinfo>─(@)┘      └(:)─<port>┘
type url struct{
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
func (p *url) Position() Position { return p.Scheme.Position() }
func (p *url) ts(ctx Context, t string) string {
    return "{"+lp(ctx, p.Position(), t, p.Scheme, p.Username, p.Password, p.Host, p.Port, p.Path, p.Query, p.Fragment)+"}"
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
func (p *url) Validate() (res *neturl.URL) {
    panic(fmt.Sprintf("validate %s", p))
    return
}

type raw struct{ valbase; s string }
func (_ *raw) kind() Kind { return KindRaw }
func (p *raw) String() string { return /* "{=raw "+p.s+"}" */p.s }
func (p *raw) trim(pre string) {
    p.s = strings.TrimSpace(strings.TrimPrefix(p.s, pre))
    return
}

type strlit struct{ valbase; s string }
func (_ *strlit) kind() Kind { return KindStrLit }
func (p *strlit) String() string { return `'`+p.s+`'` }

type strval struct{ valbase; v []Value }
func (_ *strval) kind() Kind { return KindStrVal }
func (p *strval) ts(ctx Context, t string) (s string) {
    for _, v := range p.v { s += " "+ts(v, ctx) }
    return "{"+lp(ctx, p.position, t)+s+"}"
}
func (p *strval) String() (s string) {
    for _, v := range p.v {
        if s != "" { s += " " }
        s += v.String()
    }
    s = `{=str `+s+`}`
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

type quote struct{ list }
func (_ *quote) kind() Kind { return KindQuote }
func (q *quote) String() (s string) { return "{=quote "+q.list.String()+"}" }

// punct stands for the punctuation
type punct struct{ valbase; token }
func (_ *punct) kind() Kind { return KindPunct }
func (p *punct) String() string { return p.token.String() }
func (p *punct) ts(ctx Context, t string) (s string) {
    switch p.token {
    case PROOT: s = "ROOT"
    case PTAIL: s = "TAIL"
    default: s = p.token.String()
    }
	return "{"+lp(ctx, p.position, t)+" "+s+"}"
}

type bare string
type word struct{ valbase; s string }
func (_ *word) kind() Kind { return KindWord }
func (p *word) String() string {
    if strings.Contains(p.s, " ") {
        return "{=word "+p.s+"}"
    } else {
        return p.s
    }
}

type qualword struct{ valbase; words []string } // TODO: foo.bar.zar, foo.&(bar).zar ???
func (p *qualword) String() string { return strings.Join(p.words,".") }

type elements struct{ elems []Value }
func (p *elements) list() *list { return &list{*p} }
func (p *elements) path() *path { return &path{*p} }
func (p *elements) compound() *compound { return &compound{*p} }
func (p *elements) strcomp() *strcomp { return &strcomp{*p} }
func (p *elements) len() int { return len(p.elems) }
func (p *elements) append(v ...Value) { p.elems = append(p.elems, v...) }
func (p *elements) Position() (_ Position) {
    for _, e := range p.elems {
        if e != nil { if t := e.Position(); t.valid() { return t } }
    }
    return
}
func (p *elements) at(n int) (v Value) {
    if 0 <= n && n < len(p.elems) { v = p.elems[n] }
    return
}
func (p *elements) any() (a []any) {
    for _, v := range p.elems { a = append(a, v) }
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
func (p *elements) ts(ctx Context, t string) (s string) {
    for _, a := range p.elems { s += " " + ts(a, ctx) }
    return "{="+t+s+"}"
}
func (p *elements) true(ctx Context) bool { // (or elems...)
    for _, elem := range p.elems { if __true(ctx, elem) { return true } }
    return false
}

type conjunction struct{ *list ; sep Value }
func (p conjunction) kind() Kind { return KindConjunction }
func (p conjunction) ts(ctx Context, t string) string {
	return fmt.Sprintf("{=%s %s%s}", t, p.list.ts(ctx,"list"), ts(p.sep,ctx))
}
func (p conjunction) String() (s string) {
    if p.sep != nil { s = p.sep.String() }
    return "{{"+p.list.String()+"}"+s+"}"
}

func redis(v Value) Value { v, _ = _redis(v); return v }
func _redis(v Value) (res Value, dis bool) {
	switch t := v.(type) {
	case *closure, *delegate:
		return &disjunction{valbase{v.Position()},v}, true
	case *loc:
		res, dis = _redis(t.Value)
		res = &loc{res, t.pos}
		return
	case flag:
		res, dis = _redis(t.Value)
		res = flag{res}
		return
	case *pair:
		var key, d1 = _redis(t.key)
		var val, d2 = _redis(t.val)
		res, dis = &pair{key, val}, d1 || d2
		return
	case *compound:
		res = &compound{elements{_redis_elems(t.elems, &dis)}}
		return
	case *path:
		res = &path{elements{_redis_elems(t.elems, &dis)}}
		return
	case *list:
		res = &list{elements{_redis_elems(t.elems, &dis)}}
		return
	}
    return v, false
}
func _redis_elems(vals []Value, dis *bool) (res []Value) {
	for _, val := range vals {
		e, d := _redis(val); if d { *dis = true }
		res = append(res, e)
	}
	return
}

type  ex_disjunction struct{}
type     disjunction struct{ valbase ; val Value }
func (p *disjunction) kind() Kind { return KindDisjunction/* |p.Value.kind() */ }
func (p *disjunction) String() string { return "{"+p.val.String()+"}" }
func (p *disjunction) ts(c Context, t string) string { return "{"+lp(c, p.position, t, p.val)+"}" }

func _disjunction(v Value) (res Value) {
    switch v.(type) {
    case *integer, *binary, *octal, *decimal, *hexadecimal, *float, *disjunction,
        *punct, *word, *raw, *strlit, *datetime, *file, *Date, *Time, *project, self:
        return v
    }
    return &disjunction{valbase{v.Position()},v}
}

type wants_fullfile   struct{}
type  is_compound     struct{}
type     comctx struct{ Context ; i int }
func (c *comctx) cast(t reflect.Type) Context { return icast(c, t) }
func (c *comctx) inner() Context { return c.Context }
func (c *comctx) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case wants_fullfile: return false
	case is_compound: return true
	case *closure, *delegate: c.i += 1
		if false && sfmt("%v",op) == "&(.test.xx)" {
			note(pc(ctx,op), "%v %d", op, c.i).debug(5)
		}
	}
	return c.Context.do(ctx, op)
}

func com(ctx *comctx, a, elems []Value) (res []Value) {
	for i, elem := range elems {
		switch tail := elems[i+1:]; t := elem.(type) {
		case *disjunction:
			for _, elem := range merge(expand(ctx, t.val)) {
				if isTrivial(elem) { continue }
				for _, v := range com(ctx, append(dup(a), elem), tail) {
					res = append(res, redis(v))
				}
			}
			return
		case flag:
			for _, v := range com(ctx, nil, []Value{t.Value}) {
				res = append(res, com(ctx, append(a, flag{v}), tail)...)
			}
			return
		case *loc:
			for _, v := range com(ctx, nil, []Value{t.Value}) {
				res = append(res, com(ctx, append(a, &loc{v,t.pos}), tail)...)
			}
			return
		case *pair:
			var keys = com(ctx, nil, []Value{t.key})
			var vals = com(ctx, nil, []Value{t.val})
			for _, k := range keys {
				for _, v := range vals {
					res = append(res, com(ctx, append(a, &pair{k,v}), tail)...)
				}
			}
			return
		case *path:
			var front = t.elems[:len(t.elems)-1]
			for _, v := range com(ctx, t.elems[len(t.elems)-1:], tail) {
				res = append(res, &path{elements{append(dup(front), v)}})
			}
			return
		case *compound:
			return com(ctx, a, append(t.elems, tail...))
		default:
			a = append(a, expand(ctx, elem))
		}
	}
	switch len(a) {
	case 0 : return
	case 1 : return a
	default: return []Value{&compound{elements{a}}}
	}
}

type compound struct{ elements }
func (_ *compound) kind() Kind { return KindCompound }
func (p *compound) String() (s string) {
    for _, elem := range p.elems {
        if elem == nil {
            s += "{}"
        } else if e, ok := elem.(*list); ok {
            switch len(e.elems) {
            case 0: continue
            case 1: s += elem.String()
            default: s += "⌜" + elem.String() + "⌟"
            }
        } else {
            s += elem.String()
        }
    }
    return
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

type recipe struct{ strcomp }
func (p *recipe) kind() Kind { return KindRecipe|p.strcomp.kind() }
func (p *recipe) String() string { return p.src() }

// barefile reduces file lookups, it works like an alias of a File.
type barefile struct{ Value ; file *file }
func (_ *barefile) kind() Kind { return KindBarefile }

func barefilize(ctx Context, targets ...Value) []Value {
    var project = _project(ctx)
    for i, target := range targets {
        if patterned(ctx,target) { continue }
        switch t := target.(type) {
        case *word:
            if file := project.file(ctx, t.s); file != nil {
                targets[i] = &barefile{ target, file }
                file.position = target.Position()
            }
        case *compound, *path:
            if patterned(ctx,t) /* || t.expandable(ctx) */ /* || refdef(ctx, t, DefArg) */ {//, expandDef2
                break
            } else if file := project.file(ctx, __string(ctx, t)); file != nil {
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
	var proj = _project(ctx)
    for _, target := range targets {
        if !patterned(ctx,target) {
            maps = append(maps, unmap_files(ctx, proj, target, nil)...)
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

type path struct{ elements }
func (_ *path) kind() Kind { return KindPath }
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
func (p *path) isAbs() bool { x, y := p.elems[0].(*punct); return y && x.token == PROOT }

func path_match2(ctx Context, pats []any, vals ...string) (full bool, res, stems []string) {
	var iPats, iVals = 0, 0
	var lPats, lVals = len(pats), len(vals)
	for ; iPats < lPats && iVals < lVals ; iPats, iVals = iPats+1, iVals+1 {
		var pat = unbox(pats[iPats])
		var mr, pre, suf = multia(ctx, pat) // %%  **  *?
		if val := vals[iVals]; mr != multia_no {
			var prefix = __string(ctx, pre)
			if strings.HasPrefix(val, prefix) {
				val = strings.TrimPrefix(val, prefix)
			} else {
				break
			}

			var (
				suffix = __string(ctx, suf)
				i int ; stem []string
			)
			sw_multia: switch full = false; mr {
			case multia_ss, multia_pp: // ** %%
				if suffix == "" {
					if n := iPats+1; n < lPats {
						switch t := unbox(pats[n]).(type) {
						case flag, *word, string:
							for j := lVals-1; iVals < j; j -= 1 {
								if equal(ctx, t, vals[j]) {
									res = append(res, vals[iVals:j+1]...)
									stem = append(stem, vals[iVals:j]...)
									iPats, iVals, full = n, j, true
									break sw_multia
								}
							}
						case *globpat, *percpat, *regexpat:
							for j := lVals-1; iVals < j; j -= 1 {
								if f, _, s := match(ctx, t, vals[j]); f {
									res = append(res, vals[iVals:j+1]...)
									stem = append(append(stem, val), vals[iVals+1:j]...)
									stems = append(append(stems, joinpath(stem...)), s...)
									iPats, iVals, full, stem = n, j, true, nil
									break sw_multia
								}
							}
						case *punct:
							for j := lVals-1; iVals < j; j -= 1 {
								if t.token == PTAIL {
									if vals[j] == "" {
										res = append(res, vals[iVals:j+1]...)
										stem = append(stem, vals[iVals:j]...)
										iPats, iVals, full = n, j, true
									} else {
										res = append(append(res, vals[iVals:j]...), "")
										stem = append(append(stem, vals[iVals:j]...), "")
										iPats, iVals = iPats, j
									}
									break sw_multia
								}
							}
						}
					}
					res = append(res, vals[iVals:]...)
					stem = append(append(stem, val), vals[iVals+1:]...)
					iVals, full = lVals-1, true
				} else {
					for i = lVals-1; iVals < i; i -= 1 {
						if strings.HasSuffix(vals[i], suffix) {
							res = append(res, vals[iVals:i+1]...)
							stem = append(append(append(append(stem, val), vals[iVals+1:i]...)), strings.TrimSuffix(vals[i], suffix))
							iVals, full = i, true
							break sw_multia
						}
					}
					if i == iVals {
						if strings.HasSuffix(val, suffix) {
							res = append(res, vals[i])
							stem = append(stem, strings.TrimSuffix(val, suffix))
							full = true
						} else {
							res = append(res, vals[i:]...)
							stem = append(append(stem, val), vals[i+1:]...)
						}
						break sw_multia
					}
				}
			case multia_sq: // *? %?
				if suffix == "" {
					if n := iPats+1; n < lPats {
						switch t := unbox(pats[n]).(type) {
						case flag, *word, string:
							for j := iVals+1; j < lVals; j += 1 {
								if equal(ctx, t, vals[j]) {
									res = append(res, vals[iVals:j+1]...)
									stem = append(stem, vals[iVals:j]...)
									iPats, iVals, full = n, j, true
									break sw_multia
								}
							}
						case *globpat, *percpat, *regexpat:
							for j := iVals+1; j < lVals; j += 1 {
								if f, _, s := match(ctx, t, vals[j]); f {
									res = append(res, vals[iVals:j+1]...)
									stem = append(append(stem, val), vals[iVals+1:j]...)
									stems = append(append(stems, joinpath(stem...)), s...)
									iPats, iVals, full, stem = n, j, true, nil
									break sw_multia
								}
							}
						case *punct:
							for j := iVals+1; j < lVals; j += 1 {
								if t.token == PTAIL {
									if vals[j] == "" {
										res = append(append(res, vals[iVals:j]...), "")
										stem = append(stem, vals[iVals:j]...)
										iPats, iVals, full = n, j, true
										break sw_multia
									}
								}
							}
						}
					}
					res = append(res, vals[iVals:]...)
					stem = append(append(stem, val), vals[iVals+1:]...)
					iVals, full = lVals-1, true
				} else {
					for i = iVals+1; i < lVals; i += 1 {
						if strings.HasSuffix(vals[i], suffix) {
							res = append(res, vals[iVals:i+1]...)
							stem = append(append(append(stem, val), vals[iVals+1:i]...), strings.TrimSuffix(vals[i], suffix))
							iVals, full = i, true
							break sw_multia
						}
					}
					if i == iVals {
						if strings.HasSuffix(val, suffix) {
							res = append(res, vals[i])
							stem = append(stem, strings.TrimSuffix(val, suffix))
							full = true
						} else {
							res = append(res, vals[i:]...)
							stem = append(append(stem, val), vals[i+1:]...)
						}
						break sw_multia
					}
				}
			default:
				erro(pc(ctx,pat), "wrong match: %v %v (%d/%d %v)", pat, vals, iVals, lVals, mr).trace()
			}
			if stem != nil { stems = append(stems, joinpath(stem...)) }
		} else if x, y := pat.(*punct); y && x.token == PTAIL {
			if iVals+1 < lVals { full, res = val == "", append(res, "") }
		} else if f, r, s := match(ctx, pat, val); f {
			full, res, stems = true, append(res, r.(string)), append(stems, s...)
		} else {
			break
		}
	}
	if checkpoints {
		if false && sfmt("%v",pats) == "/*?/test*/*?/" {
			note(pc(ctx,pats), "%v %d/%d %d/%d %v %v", full, iPats, lPats, iVals, lVals, res, stems).debug()
		}
		if false && iPats == lPats && iVals == lVals {
			if s1, s2 := sfmt("%v", vals), sfmt("%v", res); (s1 == s2) != full {
				erro(pc(ctx,pats), "%v full=%v", pats, full)
				note(pc(ctx,pats), "%v", vals)
				note(pc(ctx,pats), "%v", res)
				note(pc(ctx,pats), "%v", stems).trace()
			}
		}
	}
	full = full && iPats == lPats && iVals == lVals
	return
}

func path_match3(ctx Context, pats []any, vals ...string) (full bool, res, stems []string) {
	var lPats, lVals = len(pats), len(vals)
	var iPats, iVals = lPats-1, lVals-1
	for ; 0 <= iPats && 0 <= iVals; iPats, iVals = iPats-1, iVals-1 {
		var pat = unbox(pats[iPats])
		var mr, pre, suf = multia(ctx, pat) // %%  **  *?
		if val := vals[iVals]; mr != multia_no {
			var suffix = __string(ctx, suf)
			if strings.HasSuffix(val, suffix) {
				val = strings.TrimSuffix(val, suffix)
			} else {
				break
			}

			var (
				prefix = __string(ctx, pre)
				i int ; stem []string
			)
			sw_multia: switch full = false; mr {
			case multia_ss, multia_pp: // ** %%
				if prefix == "" {
					if n := iPats-1; 0 <= n {
						switch t := unbox(pats[n]).(type) {
						case flag, *word, string:
							for i = 0; i < iVals; i += 1 {
								if equal(ctx, t, vals[i]) {
									res = append(vals[i:iVals+1], res...)
									stem = append(vals[i+1:iVals], append([]string{val}, stem...)...)
									iPats, iVals, full = n, i, true
									break sw_multia
								}
							}
						case *globpat, *percpat, *regexpat:
							for i = 0; i < iVals; i += 1 {
								if f, _, s := match(ctx, t, vals[i]); f {
									res = append(vals[i:iVals+1], res...)
									stem = append(vals[i+1:iVals], append([]string{val}, stem...)...)
									stems = append(s, append([]string{joinpath(stem...)}, stems...)...)
									iPats, iVals, full, stem = n, i, true, nil
									break sw_multia
								}
							}
						case *punct:
							for i = 0; i < iVals; i += 1 {
								if t.token == PROOT {
									if vals[i] == "" {
										res = append(vals[i:iVals+1], res...)
										stem = append(vals[i+1:iVals], append([]string{val}, stem...)...)
										iPats, iVals, full = n, i, true
									} else {
										res = append([]string{""}, append(vals[i:iVals+1], res...)...)
										stem = append([]string{""}, append(vals[i+1:iVals], append([]string{val}, stem...)...)...)
										iVals = i
									}
									break sw_multia
								}
							}
						}
					}
					res = append(vals[:iVals+1], res...)
					stem = append(vals[:iVals], append([]string{val}, stem...)...)
					iVals, full = 0, true
				} else {
					for i = 0; i < iVals; i += 1 {
						if strings.HasPrefix(vals[i], prefix) {
							res = append(vals[i:iVals+1], res...)
							stem = append([]string{strings.TrimPrefix(vals[i], prefix)}, append(vals[i+1:iVals+1], append([]string{val}, stem...)...)...)
							iVals, full = i, true
							break sw_multia
						}
					}
					if i == iVals {
						if strings.HasPrefix(val, prefix) {
							res = append([]string{vals[i]}, res...)
							stem = append([]string{strings.TrimPrefix(val, prefix)}, stem...)
							full = true
						} else {
							res = append(vals[:i], res...)
							stem = append(vals[:i+1], append([]string{val}, stem...)...)
						}
						break sw_multia
					}
				}
			case multia_sq: // *? %?
				if prefix == "" {
					if n := iPats-1; 0 <= n {
						switch t := unbox(pats[n]).(type) {
						case flag, *word, string:
							for i = iVals-1; 0 <= i; i -= 1 {
								if equal(ctx, t, vals[i]) {
									res = append(vals[i:iVals+1], res...)
									stem = append(vals[i+1:iVals], append([]string{val}, stem...)...)
									iPats, iVals, full = n, i, true
									break sw_multia
								}
							}
						case *globpat, *percpat, *regexpat:
							for i = iVals-1; 0 <= i; i -= 1 {
								if f, _, s := match(ctx, t, vals[i]); f {
									res = append(vals[i:iVals+1], res...)
									stem = append(vals[i+1:iVals], append([]string{val}, stem...)...)
									stems = append(s, append([]string{joinpath(stem...)}, stems...)...)
									iPats, iVals, full, stem = n, i, true, nil
									break sw_multia
								}
							}
						case *punct:
							for i = iVals-1; 0 <= i; i -= 1 {
								if t.token == PROOT && vals[i] == "" {
									res = append(vals[i:iVals+1], res...)
									stem = append(vals[i+1:iVals], append([]string{val}, stem...)...)
									iPats, iVals, full = n, i, true
									break sw_multia
								}
							}
						}
					}
					res = append(vals[:iVals+1], res...)
					stem = append(vals[:iVals], append([]string{val}, stem...)...)
					iVals, full = 0, true
				} else {
					for i = iVals-1; 0 <= i; i -= 1 {
						if strings.HasPrefix(vals[i], prefix) {
							res = append(vals[i:iVals+1], res...)
							stem = append([]string{strings.TrimPrefix(vals[i], prefix)}, append(vals[i+1:iVals+1], append([]string{val}, stem...)...)...)
							iVals, full = i, true
							break sw_multia
						}
					}
					if i == iVals {
						if strings.HasPrefix(val, prefix) {
							res = append([]string{vals[i]}, res...)
							stem = append([]string{strings.TrimPrefix(val, prefix)}, stem...)
							full = true
						} else {
							res = append(vals[:i], res...)
							stem = append(vals[:i+1], append([]string{val}, stem...)...)
						}
						break sw_multia
					}
				}
			default:
				erro(pc(ctx,pat), "wrong match: %v %v (%d/%d %v)", pat, vals, iVals, lVals, mr).trace()
			}
			if stem != nil { stems = append([]string{joinpath(stem...)}, stems...) }
		} else if x, y := pat.(*punct); y && x.token == PROOT {
			if 0 <= iVals { full, res = val == "", append([]string{""}, res...) }
		} else if f, r, s := match(ctx, pat, val); f {
			full, res, stems = true, append([]string{r.(string)}, res...), append(s, stems...)
		} else {
			break
		}
	}
	if checkpoints {
		if false && sfmt("%v",pats) == "**/testdata/**" {
			note(pc(ctx,pats), "%v %d/%d %d/%d %v %v", full, iPats, lPats, iVals, lVals, res, stems).debug()
		}
	}
	full = full && iPats == -1 && iVals == -1
	return
}

func _pathStr(ctx Context, str string) *path {
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
        dir, name = filepath.Join(f.dir, f.sub), ident(ctx, f)
    } else {
        name = __string(ctx, val)
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
func (o fullfile) ts(t string) string { return "{="+t+" "+o.filestub.name+"}" }

func try_fullfile(ctx Context, f *file) Value {
    // if !truly(ctx, is_compound{}) && truly(ctx, is_exec{})
    if truly(ctx, wants_fullfile{}) {
        return fullfile{f}
    }
    return f
}

type fullname struct{ Value }
func (o fullname) ts(ctx Context, t string) string { return "{="+t+" "+ts(o.Value, ctx)+"}" }
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
func (p *file) ts(string) string { return "{=file "+p.filestub.name+"}" }
func (p *file) String() string { return "{=file "+p.filestub.name+"}" }
func (p *file) fullname() string { return filepath.Join(p.dir, p.sub, p.filestub.name) }
func (p *file) basename() (s string) {
    if p.info != nil { return p.info.Name() }
    return filepath.Base(p.filestub.name)
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

type flag struct{ Value }
func (p flag) kind() Kind { return KindFlag }
func (p flag) Position() (pos Position) {
    if p.Value != nil {
        pos = p.Value.Position()
        pos.Column -= 1
    }
    return
}
func (p flag) ts(ctx Context, t string) string { return "{="+t+" "+ts(p.Value,ctx)+"}" }
func (p flag) String() (s string) {
    if s = "-"; p.Value != nil { s += p.Value.String() }
    return
}
func (p flag) opt(ctx Context, name string) (res string, match bool) {
	if val := p.Value; isTrivial(val) {
		if false { erro(ctx, "flag name is trivial").trace() }
	} else if x, y := val.(flag); y {
		res, match = x.opt(ctx, name)
	} else if s := __string(ctx, val); s == name {
		res, match = name, true
	}
	return
}

type strcomp struct{ elements } // "string compound"
func (_ *strcomp) kind() Kind { return KindStrcomp }
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

type list struct{ elements }
func (_ *list) kind() Kind { return KindList }
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

type group struct{ valbase ; elements }
func (p *group) Position() Position { return p.valbase.Position() }
func (p *group) kind() Kind { return KindGroup }
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
func (p *pair) Position() (pos Position) {
    if p.key != nil { return p.key.Position() }
    if p.val != nil {
        pos = p.val.Position()
        pos.Column -= 1
    }
    return
}
func (p *pair) String() (s string) {
    if k := p.key; k != nil { s  = k.String() }; s += "="
    if v := p.val; v != nil { s += v.String() }
    return
}
func (p *pair) ts(ctx Context, t string) string {
    return "{="+t+" "+ts(p.key,ctx)+" "+ts(p.val,ctx)+"}"
}

type skipped struct{ Value }
func (p skipped) kind() Kind { return p.Value.kind()|KindSkipped }
func (p skipped) ts(ctx Context, t string) string { return "{="+t+" "+ts(p.Value, ctx)+"}" }

type _not_evoker struct{}

const (
    not_evoker int = 1<<iota
)

type closure_delegate_evoke struct{ Context ; state int }
func (c *closure_delegate_evoke) do(ctx Context, op any) (_ any) {
    switch op.(type) {
    case _not_evoker: c.state = not_evoker; return
    }
    return c.Context.do(ctx, op)
}
type closure_delegate_x struct{ Context ; closure_delegate int }
func (c *closure_delegate_x) do(ctx Context, op any) any {
    switch op.(type) {
    case *closure, *delegate: c.closure_delegate += 1
    }
    return c.Context.do(ctx, op)
}
type closure_t struct{}
type closure_x struct{ closure_delegate_x }
func (c *closure_x) do(ctx Context, op any) any {
    switch op.(type) {
    case closure_t: return true
    }
    return c.closure_delegate_x.do(ctx, op)
}
type delegate_t struct{}
type delegate_x struct{ closure_delegate_x }
func (c *delegate_x) do(ctx Context, op any) any {
    switch op.(type) {
    case delegate_t: return true
    }
    return c.closure_delegate_x.do(ctx, op)
}

type delegate struct{
    valbase
    l   token
    x   Value
    o []Value
    a []Value
    // TODO: patsubst Value, aka lhs%=rhs% like in $(var:lhs%=rhs%)
}
func (p *delegate) kind() Kind { return p.valbase.kind()|KindDelegate }
func (p *delegate) String() string { return p.src("$") }
func (p *delegate) src(l string) (s string) {
    if s = p._src(); !p.l.is_closure_delegate() { s = l + s }
    return
}
func (p *delegate) _src() (s string) { // source representation
    switch x := p.x.(type) {
    case   *def: s += x.name
    case *arrow: s += x.String()
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

    return p.wraps(s)
}
func (p *delegate) ts(ctx Context, t string) (s string) {
    s = "{" + lp(ctx, p.position, t, p.x)
    if p.o != nil { s += " " + ts(p.o, ctx) }
    for _, a := range p.a { s += " " + ts(a, ctx) }
    s += "}"
    return
}
func (p *delegate) id(ctx Context, s string) string {
    s += ident(ctx, p.x)

    if p.o != nil { // options
        s += "("
        for i, v := range p.o {
            if 0 < i { s += " " }
            s += ident(ctx, v)
        }
        s += ")"
    }

    for i, a := range p.a {
        if 0 == i { s += " " } else { s += "," }
        s += ident(ctx, a)
    }

    return p.wraps(s)
}
func (p *delegate) wraps(s string) string {
    switch p.l {
    case STRCOMP: return `"`+s+`"`
    case  STRING: return `'`+s+`'`
    case  LPAREN: return "("+s+")"
    case  LBRACE: return "{"+s+"}"
    case ILLEGAL: return "["+s+"]"
    case INTEGER: return s // $1, $2, $3, ...
    default: return p.l.String() // $@, $<, ...
    }
}
func (p *delegate) resolve(ctx Context, x Value) (res []Value) {
    for _, x := range merge(x) {
        if x != nil && !evoker(x) && x.kind()&KindBuiltin == 0 {
            switch p.l {
            case LBRACE, STRING, STRCOMP: // ${xxx}  $'xxx'  $"xxx",  else/illegal: $(xxx)
                if t := project_entry(ctx, x); t != nil { x = t }
            default:
                if t := project_resolve(ctx, ident(ctx, x)); t != nil { x = t }
            }
        }
        res = append(res, x)
    }
    return
}

type closure struct{ delegate }
func (p *closure) kind() Kind { return p.valbase.kind()|KindClosure }
func (p *closure) String() (s string) { return p.src("&") }
func (p *closure) resolve(ctx Context, x Value) (res []Value) {
	for _, x := range merge(x) {
		if x != nil && !evoker(x) && x.kind()&KindBuiltin == 0 {
			switch p.l {
			case LBRACE, STRING, STRCOMP: // &{xxx}  &'xxx'  &"xxx",  else/illegal: &(xxx)
				if t := closure_entry(ctx, x); t != nil { x = t }
			default:
				if t := closure_resolve(ctx, ident(ctx, x)); t != nil { x = t }
			}
		}
		res = append(res, x)
	}
    return
}

type arrow struct{
    valbase
    t token
    o Value // object or arrow
    s Value
}
func (p *arrow) ts(ctx Context, t string) string { return "{="+t+" "+ts(p.o,ctx)+p.t.String()+ts(p.s,ctx)+"}" }
func (p *arrow) String() string { return p.o.String()+p.t.String()+p.s.String() }

// percpat represents percent pattern expressions (e.g. '%.o')
type percpat struct{
    valbase // TODO: supporting multiple %: foo%bar%xxx
    Prefix Value
    Suffix Value
}
func (p *percpat) String() (s string) {
    if !isNull(p.Prefix) { s += p.Prefix.String() }
    s += `%`
    if !isNull(p.Suffix) { s += p.Suffix.String() }
    return
}
func (p *percpat) ts(ctx Context, t string) string {
    return fmt.Sprintf("{=%s %s %s}", t, ts(p.Prefix, ctx), ts(p.Suffix, ctx))
}
func (p *percpat) match1(ctx Context, rep string) (full bool, result string, stems []string) {
    var prefix string
    if !isTrivial(p.Prefix) {
        // FIXME: the prefix could be Glob, Regexp, etc.
        if prefix = __string(ctx, p.Prefix); strings.HasPrefix(rep, prefix) {
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
            } else if s := __string(ctx, pp.Prefix); s != "" {
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
            } else if s := __string(ctx, pp.Suffix); s != "" && strings.HasSuffix(rep[a:], s) {
                if b -= len(s); a < b {
                    stems = append(stems, rep[a:b])
                    result += rep[a:]
                    full = true
                }
                break
            }
        }
    } else if a < b && patterned(ctx,p.Suffix) {
        if false {
            warn(ctx, "mixing % pattern might have performance impact: %v", p).debug()
        }
        for n := b-1; a < n; n -= 1 {
            if f, r, ss := match(ctx, p.Suffix, rep[n:]); f && r != nil {
                var s, _ = r.(string)
                stems = append(append(stems, rep[a:n]), ss...)
                result += s // rep[a:]
                full = f
                break
            }
        }
    } else if a <= b {
        if s := __string(ctx, p.Suffix); strings.HasSuffix(rep[a:], s) {
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

type multia_result int
const (
    multia_no multia_result = iota
    multia_pp
    multia_ss
    multia_sq
)

// check for patterns like foo%%bar foo**bar foo*?bar
func multia(ctx Context, p any) (result multia_result, prefix, suffix Value) {
    switch t := p.(type) {
	case *loc: return multia(ctx, t.Value)
    case *percpat:
        switch x := t.Suffix.(type) {
        case *percpat:
            result, prefix, suffix = multia_pp, t.Prefix, x.Suffix
        }
    case *globpat:
        var glob, n = false, -1
        for i, comp := range t.elems { if m, y := comp.(*globmeta); y {
            if n == -1 && (m.token == DAST || m.token == ASTQ) {
                var t = t.elems[:i]
                if n = i; n > 0 { if glob {
                    suffix = _globpat(t...)
                } else if len(t) > 1 {
                    prefix = _compound(t...)
                } else if len(t) > 0 {
                    prefix = t[0]
                }}
                switch m.token {
                case DAST: result = multia_ss
                case ASTQ: result = multia_sq
                }
                break
            } else {
                glob = true
            }
        }}
        if result != multia_no && n < len(t.elems) {
            var t, glob = t.elems[n+1:], false
            for _, comp := range t { if _, y := comp.(*globmeta); y { glob = true ; break } }
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

type compositePattern struct{ Value ; constraints []Value }
func (p compositePattern) String() (s string) {
    s += "[" + p.Value.String() + ", "
    for i, v := range p.constraints {
        if i > 0 { s += " " } ; s += v.String()
    }
    s += "]"
    return
}

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
func (_ *globpat) kind() Kind { return KindGlobPat }
func (p *globpat) String() (s string) { for _, e := range p.elems { s += e.String() }; return }
func (p *globpat) ts(ctx Context, _ string) string { return p.elements.ts(ctx, "glob") }

type globbrace struct{ globpat }
func (p *globbrace) String() string { return "{=glob "+p.globpat.String()+"}" }
func (p *globbrace) ts(c Context, t string) string { return p.globpat.ts(c, t) }

type globmeta struct{ valbase ; token }
func (p *globmeta) String() string { return p.token.String() }
func (p *globmeta) ts(ctx Context, _ string) string { return "{"+lp(ctx,p.position,"meta")+" "+p.token.String()+"}" }

// `[a-b]`, `[abc]`, `[a$(var)c]`, `[a$(spaces)c]`
type globrange struct{ Value }
func (p *globrange) String() string { return "["+p.Value.String()+"]" }
func (p *globrange) ts(ctx Context, _ string) string { return "{"+lp(ctx,p.Position(),"range")+" "+p.Value.String()+"}" }

/*
 * # Regexp Syntax (see go/src/regexp/syntax/doc.go)
 *
 * The regular expression syntax understood by this package when parsing with the Perl flag is as follows.
 * Parts of the syntax can be disabled by passing alternate flags to Parse.
 *
 * Single characters:
 *
 * 	.              any character, possibly including newline (flag s=true)
 * 	[xyz]          character class
 * 	[^xyz]         negated character class
 * 	\d             Perl character class
 * 	\D             negated Perl character class
 * 	[[:alpha:]]    ASCII character class
 * 	[[:^alpha:]]   negated ASCII character class
 * 	\pN            Unicode character class (one-letter name)
 * 	\p{Greek}      Unicode character class
 * 	\PN            negated Unicode character class (one-letter name)
 * 	\P{Greek}      negated Unicode character class
 *
 * Composites:
 *
 * 	xy             x followed by y
 * 	x|y            x or y (prefer x)
 *
 * Repetitions:
 *
 * 	x*             zero or more x, prefer more
 * 	x+             one or more x, prefer more
 * 	x?             zero or one x, prefer one
 * 	x{n,m}         n or n+1 or ... or m x, prefer more
 * 	x{n,}          n or more x, prefer more
 * 	x{n}           exactly n x
 * 	x*?            zero or more x, prefer fewer
 * 	x+?            one or more x, prefer fewer
 * 	x??            zero or one x, prefer zero
 * 	x{n,m}?        n or n+1 or ... or m x, prefer fewer
 * 	x{n,}?         n or more x, prefer fewer
 * 	x{n}?          exactly n x
 *
 * Implementation restriction: The counting forms x{n,m}, x{n,}, and x{n}
 * reject forms that create a minimum or maximum repetition count above 1000.
 * Unlimited repetitions are not subject to this restriction.
 *
 * Grouping:
 *
 * 	(re)           numbered capturing group (submatch)
 * 	(?P<name>re)   named & numbered capturing group (submatch)
 * 	(?:re)         non-capturing group
 * 	(?flags)       set flags within current group; non-capturing
 * 	(?flags:re)    set flags during re; non-capturing
 *
 * 	flag syntax is xyz (set) or -xyz (clear) or xy-z (set xy, clear z). The flags are:
 *
 * 	i              case-insensitive (default false)
 * 	m              multi-line mode: ^ and $ match begin/end line in addition to begin/end text (default false)
 * 	s              let . match \n (default false)
 * 	U              ungreedy: swap meaning of x* and x*?, x+ and x+?, etc (default false)
 *
 * Empty strings:
 *
 * 	^              at beginning of text or line (flag m=true)
 * 	$              at end of text (like \z not \Z) or line (flag m=true)
 * 	\A             at beginning of text
 * 	\b             at ASCII word boundary (\w on one side and \W, \A, or \z on the other)
 * 	\B             not at ASCII word boundary
 * 	\z             at end of text
 *
 * Escape sequences:
 *
 * 	\a             bell (== \007)
 * 	\f             form feed (== \014)
 * 	\t             horizontal tab (== \011)
 * 	\n             newline (== \012)
 * 	\r             carriage return (== \015)
 * 	\v             vertical tab character (== \013)
 * 	\*             literal *, for any punctuation character *
 * 	\123           octal character code (up to three digits)
 * 	\x7F           hex character code (exactly two digits)
 * 	\x{10FFFF}     hex character code
 * 	\Q...\E        literal text ... even if ... has punctuation
 *
 * Character class elements:
 *
 * 	x              single character
 * 	A-Z            character range (inclusive)
 * 	\d             Perl character class
 * 	[:foo:]        ASCII character class foo
 * 	\p{Foo}        Unicode character class Foo
 * 	\pF            Unicode character class F (one-letter name)
 *
 * Named character classes as character class elements:
 *
 * 	[\d]           digits (== \d)
 * 	[^\d]          not digits (== \D)
 * 	[\D]           not digits (== \D)
 * 	[^\D]          not not digits (== \d)
 * 	[[:name:]]     named ASCII class inside character class (== [:name:])
 * 	[^[:name:]]    named ASCII class inside negated character class (== [:^name:])
 * 	[\p{Name}]     named Unicode operator inside character class (== \p{Name})
 * 	[^\p{Name}]    named Unicode operator inside negated character class (== \P{Name})
 *
 * Perl character classes (all ASCII-only):
 *
 * 	\d             digits (== [0-9])
 * 	\D             not digits (== [^0-9])
 * 	\s             whitespace (== [\t\n\f\r ])
 * 	\S             not whitespace (== [^\t\n\f\r ])
 * 	\w             word characters (== [0-9A-Za-z_])
 * 	\W             not word characters (== [^0-9A-Za-z_])
 *
 * ASCII character classes:
 *
 * 	[[:alnum:]]    alphanumeric (== [0-9A-Za-z])
 * 	[[:alpha:]]    alphabetic (== [A-Za-z])
 * 	[[:ascii:]]    ASCII (== [\x00-\x7F])
 * 	[[:blank:]]    blank (== [\t ])
 * 	[[:cntrl:]]    control (== [\x00-\x1F\x7F])
 * 	[[:digit:]]    digits (== [0-9])
 * 	[[:graph:]]    graphical (== [!-~] == [A-Za-z0-9!"#$%&'()*+,\-./:;<=>?@[\\\]^_`{|}~])
 * 	[[:lower:]]    lower case (== [a-z])
 * 	[[:print:]]    printable (== [ -~] == [ [:graph:]])
 * 	[[:punct:]]    punctuation (== [!-/:-@[-`{-~])
 * 	[[:space:]]    whitespace (== [\t\n\v\f\r ])
 * 	[[:upper:]]    upper case (== [A-Z])
 * 	[[:word:]]     word characters (== [0-9A-Za-z_])
 * 	[[:xdigit:]]   hex digit (== [0-9A-Fa-f])
 */
type regexpat struct{ valbase ; *regexp.Regexp }
func (p *regexpat) String() string {
    var s = p.Regexp.String()
    if x, y := strings.CutSuffix(s, "$"); y { s = x + "$$" }
    return "{=regex "+s+"}"
}
func (p *regexpat) ts(ctx Context, t string) string {
    return "{"+lp(ctx,p.position,"regex")+" "+p.Regexp.String()+"}"
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
            // erro(ctx, "'%v' is not value type (%T)", a, a).trace()
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
func dmerge(disjunction bool, args ...Value) (elems []Value) {
    for _, a := range args {
        switch x := a.(type) {
        case *loc: for _, v := range dmerge(disjunction, x.Value) { elems = append(elems, &loc{v,x.pos}) }
        case *list: elems = append(elems, dmerge(disjunction, x.elems...)...)
        default:
            if disjunction { if x, y := x.(*compound); y {
                var saved = len(elems)
                for i, e := range x.elems {
                    if t := dmerge(disjunction, e); len(t) > 1 {
                        for _, e := range t {
                            t := append(append(x.elems[:i],e), dmerge(disjunction, x.elems[i+1:]...)...)
                            if false { fmt.Printf("%v: %v | %v\n", e.Position(), a, t) }
                            elems = append(elems, &compound{elements{t}})
                        }
                        break
                    }
                }
                if len(elems) > saved { continue }
            }}
            elems = append(elems, x)
        }
    }
    return
}

func xmerge(c Context, v ...Value) []Value { return merge(expands(c,v...)...) }
func merge(v ...Value) []Value { return dmerge(false, v...) }

func dup(vals []Value) (res []Value) {
    if n := len(vals); 0 < n {
        if false {
            res = append([]Value{}, vals...)
        } else if false {
            for _, v := range vals { res = append(res, v) }
        } else {
            res = make([]Value, n)
            copy(res, vals)
        }
    }
    return
}

func intVal(ctx Context, v Value, i int) (res int) {
    if res = i; v != nil { res = int(__int(ctx, v)) }
    return
}

func uintVal(ctx Context, v Value, i uint32) (res uint32) {
    if res = i; v != nil { res = uint32(__int(ctx, v)) }
    return
}

func filePerm(ctx Context, v Value, i uint32) (res os.FileMode) {
    res = os.FileMode(uintVal(ctx, v, i))&os.ModePerm
    if res == 0 { res = os.FileMode(0640) }
    return
}

func deleteFile(ctx Context, val Value) (res []*file) {
    switch t := val.(type) {
    case *barefile: if t.file != nil { return deleteFile(ctx, t.file) }
    case *rule: return deleteFile(ctx, t.target)
    case *project: if e := t.main; e != nil { return deleteFile(ctx, e) }
    case *use: if e := t.project.main; e != nil { return deleteFile(ctx, e) }
    case *uselist: for _, e := range t.list { res = append(res, deleteFile(ctx, e)...) }
    case *list:
        for _, e := range t.elems { res = append(res, deleteFile(ctx, e)...) }
    case *path:
        if si := statFile(ctx, val); si == nil || si.file == nil {
            erro(ctx, "no path name for `%s`", val).trace()
        } else {
            return deleteFile(ctx, si.file)
        }
    }
    return
}

func stampFile(ctx Context, val Value) (res []*file) {
    switch t := val.(type) {
    case *barefile: if t.file != nil { return stampFile(ctx, t.file) }
    case *rule: return stampFile(ctx, t.target)
    case *project: if e := t.main; e != nil { return stampFile(ctx, e) }
    case *use: if e := t.project.main; e != nil { return stampFile(ctx, e) }
    case *uselist: for _, e := range t.list { res = append(res, stampFile(ctx, e)...) }
    case *list:
        for _, e := range t.elems { res = append(res, stampFile(ctx, e)...) }
    case *path:
        if si := statFile(ctx, val); si == nil || si.file == nil {
            erro(ctx, "no path name for `%s`", val).trace()
        } else {
            return stampFile(ctx, si.file)
        }
    }
    return
}

func statFile(ctx Context, val Value) (res *statinfo) {
    switch t := val.(type) {
    case *barefile: if t.file != nil { return statFile(ctx, t.file) }
    case *rule: return statFile(ctx, t.target)
    case *project: if e := t.main; e != nil { return statFile(ctx, e) }
    case *use: if e := t.project.main; e != nil { return statFile(ctx, e) }
    case *uselist:
        for _, e := range t.list {
            if si := statFile(ctx, e); si != nil {
                if res == nil { res = si } else { res.next = si }
            }
        }
    case *list:
        for _, e := range t.elems {
            if si := statFile(ctx, e); si != nil {
                if res == nil { res = si } else { res.next = si }
            }
        }
    case *path:
        var s string
        if patterned(ctx, t) {
            if v, rest := stencil(ctx, t, _stems(ctx)); len(rest) > 0 {
                erro(ctx, "partial match: %v", rest)
            } else {
                s = __string(ctx, v)
            }
        } else {
            s = __string(ctx, t)
        }

        if filepath.IsAbs(s) {
            if f := _stat(ctx, s, stat_nonexist{true}); f != nil {
                return &statinfo{ file: f }
            }
        }

        if f := findfile(ctx, s); f != nil { return statFile(ctx, f) }
    }
    return
}

func updated(ctx Context, val Value) (res bool) {
    switch t := val.(type) {
    case *barefile: if t.file != nil { return updated(ctx, t.file) }
    case *rule: if res = updated(ctx, t.target); res { do(ctx, mark_dirty{[]Value{ t.target }}) }
    case *list: for _, e := range t.elems { if res = updated(ctx, e); res { break } }
    case *use: if e := t.project.main; e != nil { return updated(ctx, e) }
    case *uselist: for _, e := range t.list { res = res || updated(ctx, e) }
    }
    return
}

func updatedDeps(ctx Context, val Value, deps ...Value) (res []Value) {
    switch t := val.(type) {
    case *barefile: if t.file != nil { return updatedDeps(ctx, t.file, deps...) }
    case *rule: return updatedDeps(ctx, t.target, deps...)
    case *list: for _, e := range t.elems { res = append(res, updatedDeps(ctx, e, deps...)...) }
    case *use: if e := t.project.main; e != nil { return updatedDeps(ctx, e, deps...) }
    case *uselist: for _, e := range t.list { res = append(res, updatedDeps(ctx, e, deps...)...) }
    }
    return
}

func __t(a ...bool) (_ bool) { for _, a := range a { if a { return true } }; return }
func __string_any(s ...string) (a []any) { for _, s := range s { a = append(a, s) }; return }
type __string_ctx struct{ Context /* ; i int */ }
func (c *__string_ctx) inner() Context { return c.Context }
func (c *__string_ctx) do(ctx Context, op any) any {
	switch t := op.(type) {
	case __string_ctx: switch t.Context { case c, c.Context, nil: return c }
	case ex_closure, ex_def_1, ex_def_2, ex_def_3: return true
	// case *delegate, *closure: c.i += 1
	}
	return c.Context.do(ctx, op)
}
func __string(ctx Context, v any) (res string) {
    if _, ok := do(ctx, __string_ctx{}).(*__string_ctx); !ok { ctx = &__string_ctx{ctx} }
	switch t := v.(type) {
	case string: return t
	case rune: return string(t)
	case *binary, *octal, *decimal, *hexadecimal, *float, *datetime, *Date, *Time, *qualword, *globmeta, *punct:
		return t.(Value).String()
	case *valbase, *null, *none, nil: return
	case *answer : if t.bool { return "yes"  } else { return "no" }
	case *option : if t.bool { return "on"   } else { return "off" }
	case *boolean: if t.bool { return "true" } else { return "false" }
	case *strlit: return t.s
	case *word: return t.s
	case *raw: return t.s
	case *regexpat: return t.Regexp.String()
	case *file: return t.filestub.name
	case *project: return t.name
	case self: return t.name
	case fullfile: return t.fullname()
	case negative: return "!"+__string(ctx, t.Value)
	case flag: return "-"+__string(ctx, t.Value)
	case *disjunction: return __string(ctx, t.val)
    case *undetermined: return __string(ctx, t.value)
	case *loc: return __string(ctx, t.Value)
	case *def: return __string(ctx, t.value)
	case *auto: return __string(ctx, t.def(ctx))
	case *rule: return __string(ctx, t.target)
	case *quote: return strconv.Quote(__string(ctx, &t.list))
	case *globrange: return "[" + __string(ctx, t.Value) + "]"
	case *globbrace: for _, e := range t.elems { res += __string(ctx, e) }
	case *globpat: for _, e := range t.elems { res += __string(ctx, e) }
	case *percpat:
		if t.Prefix != nil { res += __string(ctx, t.Prefix) }; res += "%"
		if t.Suffix != nil { res += __string(ctx, t.Suffix) }
	case *barefile:
		if v = t.Value; t.file != nil { v = t.file }
		return __string(ctx, v)
	case *compound:
		var cc = &comctx{ctx, 0}
		for i, e := range com(cc, nil, t.elems) { // a{x y z}b → axb ayb azb
			if checkpoints { check__string_com(ctx, t, e) }
			if 0 < i && res != "" { res += " " }
			if x, y := e.(*compound); y {
				for _, e := range x.elems { res += __string(ctx, e) }	
			} else {
				res += __string(ctx, e)
			}
		}
	case *path:
		for i, elem := range t.elems {
			if v := __string(ctx, elem); 0 < i {
				res += pathSep + v
			} else if v != "" {
				res += v
			} else if len(t.elems) == 1 {
				res += pathSep
			}
		}
	case *strcomp:
		for _, elem := range t.elems { res += __string(ctx, elem) }
		if truly(ctx, is_defname{}) { return `"` + res + `"` }
	case *strval:
		for _, v := range t.v { if res != "" { res += " " }; res += __string(ctx, v) }
		if truly(ctx, is_defname{}) { return `"` + res + `"` }
	case *list:
		for _, elem := range t.elems {
			if t := __string(ctx, elem); t != "" {
				if res != "" && !strings.HasSuffix(res, "\n") { res += " " }
				res += t
			}
		}
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t.(Value)); !equal(ctx, v.(Value), t.(Value)) { res = __string(ctx, v) }
		if checkpoints { check_string(ctx, t.(Value), v.(Value), res) }
	case fullname:
		if v := t.Value; v != nil {
			if x, y := v.(*file); y {
				if x == nil {
					erro(ctx, "nil file").trace()
				} else if x.filestub == nil {
					erro(ctx, "nil file stub").trace()
				}
				return x.fullname()
			}
			return __string(ctx, v)
		}
	case conjunction: // see also $(join ...)
		var ( sep = __string(ctx, t.sep); ss []string )
		for _, v := range t.elems { if s := __string(ctx, v); s != "" { ss = append(ss, s) } }
		return strings.Join(ss, sep)
	case *pair:
		var ks = merge(expand(ctx, t.key))
		var vs = merge(expand(ctx, t.val))
		for _, k := range ks {
			if !isNull(k) {
				for _, v := range vs {
					if !isNull(v) {
						if res != "" { res += " " }
						res += __string(ctx, k) + "=" + __string(ctx, v)
					}
				}
			}
		}
	case *group:
		res = "("
		for i, elem := range t.elems { if 0 < i { res += " " }; res += __string(ctx, elem) }
		res += ")"
	case *argumented:
		res = __string(ctx, t.Value) + "("
		for i, arg := range t.args { if 0 < i { res += "," }; res += __string(ctx, arg) }
		res += ")"
	case *plainline:
		if len(t.elems) == 1 {
			if d, y := t.elems[0].(*delegate); y {
				if x, y := d.x.(*builtin); y && x.name == "foreach" {
					return __string(plainline_ctx{ctx}, d)
				}
			}
		}

		for _, v := range t.elems { res += __string(ctx, v) }

		if i := len(t.elems); 0 < i {
			var v = t.elems[i-1]
			if _, y := v.(*null); y { return }
			if x, y := v.(*list); y { if i := x.len(); 0 < i { v = x.elems[i-1] } }
			if _, y := v.(*plainline); y { return }
		}

		res += "\n"
	case *plain:
		var cc = plain_ctx{ctx, len(t.elems)==1}
		for i, e := range t.elems {
			if x, y := e.(*plainline); y {
				res += __string(cc, x)
			} else {
				if 0 < i && res != "" { res += " " }
				res += __string(cc, e)
			}
		}
	case *use:
		return fmt.Sprintf("use %s %v", t.project.name, t.params)
	case *uselist:
		for i, u := range t.list {
			if 0 < i && res != "" { res += " " }
			res += u.project.name
		}
		return "[" + res + "]"
	case *modifier:
		return __string(ctx, modify(ctx, &t.group, false))
	case *modification:
		for i, m := range t.list {
			if 0 < i && res != "" { res += " " }
			if m != nil { res += __string(ctx, modify(ctx, &m.group, false)) }
		}
	case *url:
		if res = __string(ctx, t.Scheme) + ":" ; t.Host != nil && !isNone(t.Host) {
			if res += "//" ; t.Username != nil && !isNone(t.Username) {
				res += __string(ctx, t.Username)
				if t.Password != nil { res += ":" + __string(ctx, t.Password) }
				res += "@"
			}
			res += __string(ctx, t.Host)
			if t.Port != nil && !isNone(t.Port) { res += ":" + __string(ctx, t.Port) }
		}
		if t.Path != nil && !isNone(t.Path) {
			//if !strings.HasPrefix(res, pathSep) { res += pathSep }
			res += __string(ctx, t.Path)
		}
		if t.Query != nil {
			res += "?"
			for i, q := range t.Query { if 0 < i { res += "&" }; res += __string(ctx, q) }
		}
		if t.Fragment != nil && !isNone(t.Fragment) { res += "#" + __string(ctx, t.Fragment) }
	case *exec_result:
		if t.Stdout.Buf != nil { return t.Stdout.Buf.String() }
		if t.Stderr.Buf != nil { return t.Stderr.Buf.String() }
		return strconv.Itoa(t.Status)
	case *escaped:
		switch t.s {
		case "n": return "\n"
		case "r": return "\r"
		default:  return t.s
		}
	}
	return
}

func __true(ctx Context, v Value) (res bool) {
	switch t := v.(type) {
	case *answer: return t.bool
	case *option: return t.bool
	case *boolean: return t.bool
	case *binary: return t.int64 != 0
	case *octal: return t.int64 != 0
	case *decimal: return t.int64 != 0
	case *hexadecimal: return t.int64 != 0
	case *float: return math.Abs(t.float64)-0 > epsilon
	case *datetime: return !t.t.IsZero()
	case *Date: return !t.t.IsZero()
	case *Time: return !t.t.IsZero()
	case *qualword: return len(t.words) != 0
	case *compound: return t.elements.true(ctx)
	case *globpat: return t.elements.true(ctx)
	case *strcomp: return t.elements.true(ctx)
	case *strlit: return t.s != ""
	case *word: return t.s != ""
	case *raw: return t.s != ""
	case *escaped: return t.s != ""
	case *builtin: return t.t != nil
	case self: return t.name != ""
	case *project: return t.name != ""
	case *file: return t.filestub.name != ""
	case *loc: return __true(ctx, t.Value)
	case *def: return __true(ctx, t.value)
	case *auto: return __true(ctx, t.def(ctx))
	case *rule: return __true(ctx, t.target)
	case *pair: return __true(ctx, t.key) || __true(ctx, t.val)
	case flag: return __true(ctx, t.Value)
	case negative: return !__true(ctx, t.Value)
	case *barefile: return __true(ctx, t.file)
	case *url: return __true(ctx, t.Scheme) || __true(ctx, t.Host) || __true(ctx, t.Path)
	case *exec_result: return t.Status == 0 && t.Stderr.Buf != nil && t.Stderr.Buf.Len() == 0 /* && t.Stdout.Buf.Len() > 0 */
	case *strval: for _, v := range t.v { if __true(ctx, v) { return true } }
	case *list: for _, v := range t.elems { if __true(ctx, v) { return true } }
	case *path: for _, v := range t.elems { if __true(ctx, v) { return true } }
	case *group: for _, v := range t.elems { if __true(ctx, v) { return true } }
	case *plain: for _, v := range t.elems { if __true(ctx, v) { return true } }
	case *plainline: for _, v := range t.elems { if __true(ctx, v) { return true } }
	case *uselist: return 0 < len(t.list)
	case *use: return t.project != nil
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					erro(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t)).trace()
				}
			}
			return __true(ctx, v)
		}
	}
	return
}

func __int(ctx Context, v Value) (res int64) {
	switch t := v.(type) {
	case *answer: if t.bool { return 1 }
	case *option: if t.bool { return 1 }
	case *boolean: if t.bool { return 1 }
	case *binary: return t.int64
	case *octal: return t.int64
	case *decimal: return t.int64
	case *float: return int64(t.float64)
	case *datetime: return t.t.Unix()
	case *Date: return t.t.Unix()
	case *Time: return t.t.Unix()
	case *pair: return __int(ctx, t.val)
	case *auto: return __int(ctx, t.def(ctx))
	case *loc: return __int(ctx, t.Value)
	case *def: return __int(ctx, t.value)
	case *exec_result: return int64(t.Status)
	case *uselist: return int64(len(t.list))
	case *barefile: if t.file != nil && t.file.exists() { return t.file.info.Size() }
	case *list: if t.len() > 0 { return __int(ctx, t.elems[0]) }
	case *plain: if t.len() > 0 { return __int(ctx, t.elems[0]) }
	case *plainline: if t.len() > 0 { return __int(ctx, t.elems[0]) }
	case negative: if t.Value != nil && !__true(ctx, t.Value) { return 1 }
	case flag: if t.Value != nil { return -__int(ctx, t.Value) }
	case *raw:
		if i, e := strconv.ParseInt(t.s, 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *word:
		if i, e := strconv.ParseInt(t.s, 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *strlit:
		if i, e := strconv.ParseInt(t.s, 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *strcomp:
		if i, e := strconv.ParseInt(__string(ctx, t), 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *strval:
		if i, e := strconv.ParseInt(__string(ctx, t), 10, 64); e == nil {
			return i
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *compound:
		if n := len(t.elems); n > 0 {
			if i, y := t.elems[0].(*decimal); y {
				switch n {
				case 1: res = i.int64
				case 2:
					if w, y := t.elems[1].(*word); y {
						if  (w.s == "st" && i.int64%1 == 0) ||
							(w.s == "nd" && i.int64%2 == 0) ||
							(w.s == "rd" && i.int64%3 == 0) ||
							(w.s == "th") { res = i.int64 }
					}
				}
			}
		}
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					erro(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t)).trace()
				}
			}
			return __int(ctx, v)
		}
	}
	return
}

func __float(ctx Context, v Value) (_ float64) {
	switch t := v.(type) {
	case *answer: if t.bool { return 1. }
	case *option: if t.bool { return 1. }
	case *boolean: if t.bool { return 1. }
	case *binary: return float64(t.int64)
	case *octal: return float64(t.int64)
	case *decimal: return float64(t.int64)
	case *datetime: return float64(t.t.Unix())
	case *Date: return float64(t.t.Unix())
	case *Time: return float64(t.t.Unix())
	case *exec_result: return float64(t.Status)
	case *pair: return __float(ctx, t.val)
	case *auto: return __float(ctx, t.def(ctx))
	case *def: return __float(ctx, t.value)
	case *loc: return __float(ctx, t.Value)
	case *float: return t.float64
	case *list: if t.len() > 0 { return __float(ctx, t.elems[0]) }
	case *plain: if t.len() > 0 { return __float(ctx, t.elems[0]) }
	case *plainline: if t.len() > 0 { return __float(ctx, t.elems[0]) }
	case negative: if t.Value != nil && !__true(ctx, t.Value) { return 1. }
	case flag: if t.Value != nil { return -__float(ctx, t.Value) }
	case *raw:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *word:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *strlit:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *strcomp:
		if f, e := strconv.ParseFloat(__string(ctx, t), 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *strval:
		if f, e := strconv.ParseFloat(__string(ctx, t), 64); e == nil {
			return f
		} else {
			erro(ctx, "%v", e).trace()
		}
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					erro(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t)).trace()
				}
			}
			return __float(ctx, v)
		}
	}
	return
}

func expand(ctx Context, v Value) (res Value) {
    switch t := v.(type) {
    case *delegate:
        var c = delegate_x{closure_delegate_x{ctx, 0}}
        if x := t.resolve(ctx, expand(&c, t.x)); c.closure_delegate == 0 {
			var vals []Value
            for _, x := range x {
                cc := closure_delegate_evoke{ctx, 0}
                if v := evoke(&cc, x, t.o, t.a); cc.state == not_evoker || v == nil {
                    vals = append(vals, _null(t.position))
                } else {
                    vals = append(vals, _loc(v, t.position))
                }
            }
			if res = ease(ctx, vals); checkpoints { check(ctx, res, v, x...) }
        } else {
			var vals []Value
            var o, a = expands(ctx, t.o...), expands(ctx, t.a...)
            for _, x := range x {
                v := &delegate{t.valbase, t.l, x, o, a}
                vals = append(vals, v)
                do(ctx, v)
            }
			if res = ease(ctx, vals); checkpoints { check(ctx, res, v, x...) }
        }
        return
    case *closure:
        var c = closure_x{closure_delegate_x{ctx, 0}}
		if x := t.resolve(ctx, expand(&c, t.x)); c.closure_delegate == 0 && truly(ctx, ex_closure{}) {
			var vals []Value
			for _, x := range x {
				var cc = closure_delegate_evoke{ctx, 0}
				if v := evoke(&cc, x, t.o, t.a); cc.state == not_evoker || v == nil {
					vals = append(vals, _null(t.position))
				} else {
					vals = append(vals, _loc(v, t.position))
				}
			}
			if res = ease(ctx, vals); checkpoints { check(ctx, res, v, x...) }
		} else {
			var vals []Value
			var o, a = expands(ctx, t.o...), expands(ctx, t.a...)
			for _, x := range x {
				v := &closure{delegate{t.valbase, t.l, x, o, a}}
				vals = append(vals, v)
				do(ctx, v)
			}
			if res = ease(ctx, vals); checkpoints { check(ctx, res, v, x...) }
		}
        return
    case *arrow:
        var vals []Value
        var p0 = t.Position()
        var os = merge(expand(ctx, t.o))
        var ss = merge(expand(ctx, t.s))
        for _, o := range os {
			if !can_sel(ctx, o) {
				var str, ok = _sel_prop(ctx, o)
				if t.t == SELECT_PROP && !ok { str = ident(ctx, o) }
                switch {
                case truly(ctx, closure_t{}):
                    switch t.t {
                    case SELECT_PROG1, SELECT_PROG2:
                        if v := closure_entry(ctx, o); v != nil { o = v }
                    case SELECT_PROP:
                        if v := closure_resolve(ctx, str); v != nil { o = v }
                    default:
                        erro(pc(ctx,t), "%v %v %v", o, t.t, ss).trace()
                    }
                case truly(ctx, delegate_t{}):
                    switch t.t {
                    case SELECT_PROG1, SELECT_PROG2:
                        if v := project_entry(ctx, o); v != nil { o = v }
                    case SELECT_PROP:
                        if v := project_resolve(ctx, str); v != nil { o = v }
                    default:
                        erro(pc(ctx,t), "%v %v %v", o, t.t, ss).trace()
                    }
                default:
                    erro(pc(ctx,t), "%v %v", o, ss).trace()
                }
            }
            for _, s := range ss {
                var p = s.Position()
                switch t := sel(ctx, o, sel_prop(ctx, s)).(type) {
                case nil: vals = append(vals, _loc(_null(p), p0))
                default:  vals = append(vals, _loc(_loc(t,p), p0)) // NOTE: not def.value
                }
            }
        }
        if res = ease(ctx, vals); checkpoints { /* check(ctx, res, v) */ }
        return
    case *strlit:
        if truly(ctx, ex_path_str{}) {
            return _pathStr(ctx, t.s)
        } else {
            return v
        }
    case *strval:
        if truly(ctx, ex_path_str{}) {
            return &strval{t.valbase, _pathStr(ctx, __string(ctx, t)).elems}
        } else {
            return &strval{t.valbase, expands(ctx, t.v...)}
        }
    case *strcomp:
        if truly(ctx, ex_path_str{}) {
            return &strcomp{elements{_pathStr(ctx, __string(ctx, t)).elems}}
        } else {
            return &strcomp{elements{expands(ctx, t.elems...)}}
        }
    case *compound:
        var vals = com(&comctx{ctx,0}, nil, t.elems)
        if res = ease(pc(ctx,v), vals); checkpoints { check(ctx, res, v) }
        return
    case *pair:
        var vals []Value
        var ks = merge(expand(ctx, t.key))
        var vs = merge(expand(ctx, t.val))
		for _, k := range ks { for _, v := range vs { vals = append(vals, &pair{k, v}) }}
        if res = ease(ctx, vals); checkpoints { /* check(ctx, res, v) */ }
        return
    case flag:
        if t.Value == nil {
            return t
        } else {
            var vals []Value
            for _, v := range merge(expand(ctx, t.Value)) { vals = append(vals, flag{v}) }
            if res = ease(ctx, vals); checkpoints { /* check(ctx, res, v) */ }
            return
        }
    case conjunction:
        c := conjunction{&list{elements{expands(ctx, t.list.elems...)}}, nil}
        if c.sep != nil { c.sep = expand(ctx, t.sep) }
        return c
    case *disjunction:
        var vals []Value
        for _, v := range merge(expand(ctx, t.val)) { vals = append(vals, redis(v)) }
        if res = ease(ctx, vals); checkpoints { /* check(ctx, res, v) */ }
        return
    case *barefile:
        var f = t.file
        if truly(ctx, wants_fullfile{}) {
            if f == nil { f = findfile(ctx, __string(ctx, t.Value)) }
            if f != nil { return fullfile{f} }
        }
        return &barefile{expand(ctx, t.Value), f}
	case *rule:
		if v := expand(ctx, t.target); !equal(ctx, v, t.target) { t = &rule{v, t.arged, t.program} }
		return t
	case *stemmed_rule:
		if v := expand(ctx, t.rule).(*rule); v != t.rule { t = &stemmed_rule{v, t.target, t.stems} }
		return t
    case fullfile: if truly(ctx,is_compound{}) { return t.file  } else { return t }
    case fullname: if truly(ctx,is_compound{}) { return t.Value } else { return fullname{expand(ctx,t.Value)} }
    case *loc: return &loc{expand(ctx, t.Value), t.pos}
    case *list: return &list{elements{expands(ctx, t.elems...)}}
    case *path: return &path{elements{expandp(ctx, t.elems...)}}
    case *quote: return &quote{list{elements{expands(ctx, t.elems...)}}}
    case *group: return &group{t.valbase,elements{expands(ctx, t.elems...)}}
    case *recipe: return &recipe{strcomp{elements{expands(ctx, t.elems...)}}}
    case *percpat: return &percpat{t.valbase, expand(ctx,t.Prefix), expand(ctx,t.Suffix)}
    case *globpat: return &globpat{elements{expands(ctx, t.elems...)}}
    case *globrange: return &globrange{expand(ctx, t.Value)}
    case *argumented: return &argumented{expand(ctx, t.Value), expands(ctx, t.args...)}
    case negative: return negative{expand(ctx, t.Value)}
    case *url: return &url{expand(ctx, t.Scheme), expand(ctx, t.Username), expand(ctx, t.Password), expand(ctx, t.Host), expand(ctx, t.Port), expand(ctx, t.Path), expands(ctx, t.Query...), expand(ctx, t.Fragment)}
    case *plain: return &plain{elements{expands(ctx, t.elems...)}, t.name}
    case *plainline: return &plainline{elements{expands(ctx, t.elems...)}}
    case *returner: return &returner{t.valbase, expands(ctx, t.vals...)}
    case *use: return &use{t.valbase, t.project, expands(ctx, t.params...), t.opts}
    case *uselist:
        var us []*use
        for _, u := range t.list {
            u = &use{u.valbase, u.project, expands(ctx, u.params...), u.opts}
            us = append(us, u)
        }
        return &uselist{t.owner_, t.name, t.scope, us}
	case *modifier: return modify(ctx, &t.group, false)
	case *modification:
		var va []Value
		for _, m := range t.list {
			if v := expand(ctx, m); v != nil { va = append(va, v) }
		}
		return ease(ctx, va)
    case *undetermined:
		return &undetermined{t.token, expand(ctx, t.identifier), expand(ctx, t.value)}
    case *valbase, *answer, *boolean, *binary, *auto, *builtin, *def, *none, *null, *punct, *qualword, *word, *globmeta, *octal, *decimal, *hexadecimal, *escaped, *raw, *regexpat, *defcaps, *project, self, undef, nil:
        if false && v == nil { note(pc(ctx,v), "%v", v).debug() } //, *modification
        return v
    default:
        if checkpoints { erro(pc(ctx,v), "%v", ts(v)).trace() }
        return v
    }
}

func expands(ctx Context, v ...Value) (res []Value) {
    for _, v := range v { res = append(res, expand(ctx, v)) }
    return
}

func expandp(ctx Context, v ...Value) []Value {
	return path_elems(expands(ctx, v...)...)
}

func path_elems(v ...Value) (res []Value) {
	for _, v := range v {
		switch t := v.(type) {
		case *loc:
			for _, v := range path_elems(t.Value) { res = append(res, &loc{v, t.pos}) }
		case *path:
			res = append(res, path_elems(t.elems...)...)
		default:
			res = append(res, t)
		}
	}
	return
}

func can_sel(ctx Context, v Value) (res bool) {
    switch t := v.(type) {
    case *loc: return can_sel(ctx, t.Value)
    case *def, *project, self, *use, *uselist: return true
    case *list: for _, v := range t.elems { if can_sel(ctx, v) { return true } }
    default: if false { notestack(pc(ctx,v), 8, "%v", ts(v)).debug(10) }
    }
    return
}
func _sel_prop(ctx Context, v Value) (res string, okay bool) {
	switch t := v.(type) {
	case *globpat:
		if l := t.len()-1; 0 < l {
			if m, y := t.elems[l].(*globmeta); y && m.token == QUE {
				for _, e := range t.elems[:l] { res += __string(ctx, e) }
				return res, true
			}
		}
		return __string(ctx, t), false
	case *globbrace: return _sel_prop(ctx, &t.globpat)
	case *loc: return _sel_prop(ctx, t.Value)
	default: return __string(ctx, t), false
	}
	return
}
func sel_prop(c Context, v Value) (s string) { s, _ = _sel_prop(c, v); return }
func sel(ctx Context, v any, s string) (res Value) {
	var g *globpat
    switch t := v.(type) {
    case *loc: return sel(ctx, t.Value, s)
    case *def: return sel(ctx, t.value, s)
    case *list: return sel(ctx, t.elems, s)
    case *project: return t.resolve(ctx, s)
    case self: return t.resolve(ctx, s)
    case *word: return
	case *globbrace: g = &t.globpat
	case *globpat: g = t
    case []Value:
        var vals []Value
        for _, t := range t { vals = append(vals, sel(ctx, t, s)) }
        return ease(ctx, vals)
    case *uselist:
        var vals []Value
        if m, x := name_prefix.FindStringSubmatch(s), "use."; m != nil {
            s = m[1]+x+m[3] // prefix, name = m[1], m[3]
        } else {
            s = x+s
        }
        for _, u := range t.list {
            if !u.opts.noVars {
                if o := u.project.Lookup(s); o != nil {
                    vals = append(vals, o)
                }
            }
        }
        return ease(ctx, vals)
    default:
        errostack(pc(ctx,v), 8, "cannot sel: %v %v", ts(t), s).trace()
    }
	if g != nil {}
    return
}

func evoker(x Value) bool {
    switch t := x.(type) {
    case *loc: return evoker(t.Value)
    case *auto, *builtin, *def, *rule, *project, self: return true
    case interface{ evoke(*evocation) Value }: return true
    default: return false
    }
}
func evoke(ctx Context, x Value, o, a []Value) (res Value) {
    switch t := x.(type) {
	case *project, self: return x
    case *loc: return evoke(ctx, t.Value, o, a)
    case *rule: return ease(ctx, t.execute(ctx, expands(ctx, a...)...))
    case *auto:
		if d := auto_find(ctx, t.name); d != nil { return evoke(ctx, d, o, a) }
		return _null(x.Position())
	case *def:
		if truly(ctx, evoke_detect_loop{x}) {
			if truly(ctx, evoke_loop_panic{}) { panic(trace_evoke_loop_err{ctx, x}) }
			res = _null(x.Position())
		} else {
			ctx := def_evoke{&evocation{automatic{Context:ctx, defs:make(def_map)}, x, o, a}}
			if ctx.a != nil { ctx.args(ctx, expands(ctx.Context, ctx.a...)) }
			if res = expand(ctx, t.value); t.o == defExecute && !isEmpty(res) { res = t.xexe(ctx, res) }
		}
		return
    case *builtin:
        if truly(ctx, evoke_detect_loop{x}) {
            if truly(ctx, evoke_loop_panic{}) { panic(trace_evoke_loop_err{ctx, x}) }
            return _null(x.Position())
        }

		ctx := &evocation{automatic{Context:ctx, defs:make(def_map)}, x, nil, a}
		_v := reflect.New(t.t)
		_i := _v.Interface()

		defer t.benchmark(ctx, time.Now(), _v)

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

		if o != nil { ctx.o = _opts(ctx, _v, o) }

		if x, y := _i.(builtin_x); y {
			res = ease(ctx, x.x())
		} else {
			errostack(pc(ctx,x), 3, "no method: %v", t.t.Name()).trace()
		}
    default:
        do(ctx, _not_evoker{})
    }
	return
}

func match(ctx Context, pat, val any) (full bool, res any, stems []string) {
	var vals []string

	switch t := unbox(val).(type) {
	case *filestub:
		if full, res, stems = match(ctx, pat, t.name); full || res != nil {
			if checkpoints { check_match(pc(ctx,pat), pat, val, full, res, stems) }
		} else if t.dir != "" || t.sub != "" {
			full, res, stems = match(ctx, pat, strings.Split(filepath.Join(t.dir, t.sub, t.name), pathSep))
			if checkpoints { check_match(pc(ctx,pat), pat, val, full, res, stems) }
		}
		return
	case *file: return match(ctx, pat, &t.filestub)
	case    Value: vals = strings.Split(__string(ctx, t), pathSep)
	case   string: vals = strings.Split(t, pathSep)
	case []string: vals = t
	case nil: return
	}
	
	switch p := unbox(pat).(type) {
	case []any:
		var result []string
		if truly(ctx, propReversal) {
			full, result, stems = path_match3(ctx, p, vals...)
			if checkpoints { check_match(ctx, p, vals, full, result, stems) }
		} else {
			full, result, stems = path_match2(ctx, p, vals...)
			if checkpoints { check_match(ctx, p, vals, full, result, stems) }
		}
		switch len(result) {
		case 0 : res = nil
		case 1 : res = result[0]
		default: res = result
		}
		return
    case string:
		switch strs := strings.Split(p, pathSep); len(strs) {
		case 0: return
		case 1:
			switch len(vals) {
			case 0:
			case 1:
				if t := vals[0]; strings.HasPrefix(t, p) {
					if full, res = len(t) == len(p), t[:len(p)]; checkpoints {
						check_match(ctx, pat, val, full, res, stems)
					}
				}
			default:
				if p == vals[0] { res = []string{vals[0]} }
			}
			return
		default:
			return match(ctx, __string_any(strs...), vals)
		}
	case *raw: return match(ctx, p.s, val)
	case *word: return match(ctx, p.s, val)
	case *project: return match(ctx, p.name, val)
	case *strlit: return match(ctx, p.s, val)
	case *strval: return match(ctx, __string(ctx, p), val)
	case *strcomp: return match(ctx, __string(ctx, p), val)
	case *qualword: return match(ctx, __string(ctx, p), val)
	case *plainline: return match(ctx, __string(ctx, p), val)
	case *plain: return match(ctx, __string(ctx, p), val)
	case *url: return match(ctx, __string(ctx, p), val)
	case *rule: return match(ctx, p.target, val)
	case *punct: return match(ctx, p.token.String(), val)
	case *barefile: return match(ctx, p.Value, val)
	case *path: return match(ctx, p.any(), val)
	case *file: return match(ctx, &p.filestub, val)
	case *filestub:
		if full, res, stems = match(ctx, p.name, val); full || res != nil {
			if checkpoints { check_match(pc(ctx,pat), pat, val, full, res, stems) }
		} else if p.dir != "" || p.sub != "" {
			full, res, stems = match(ctx, strings.Split(filepath.Join(p.dir, p.sub, p.name), pathSep), val)
			if checkpoints { check_match(pc(ctx,pat), pat, val, full, res, stems) }
		}
		return
	case *compound:
		var n int
		var s string
		var elem Value
		switch t := val.(type) {
		case   Value : s = __string(ctx, t)
		case   string: s = t
		case []string: if len(t) == 1 { s = t[0] } else { return }
		default:
			erro(pc(ctx,p), "unsupported match: %v", ts(val)).trace()
		}

		if s == "" { return }

		var rs string
		for n, elem = range p.elems {
			if _, r, ss := match(ctx, elem, s); r == nil {
				break
			} else if t := joinp(ctx, r); t == "" {
				break
			} else {
				stems = append(stems, ss...)
				s = s[len(t):]
				rs += t
			}
		}
        if s == "" && n == len(p.elems)-1 { full = true }
        if full || rs != "" { res = rs }
        if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
		return
    case flag:
        switch t := val.(type) {
        case *valbase, *none, *null:
        case flag:
            full, res, stems = match(ctx, p.Value, t.Value)
            if s, y := res.(string); y { res = "-" + s }
            if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
        case Value:
            if v := __string(ctx, t); strings.HasPrefix(v, "-") {
                full, res, stems = match(ctx, p.Value, v[1:])
                if s, y := res.(string); y { res = "-" + s }
                if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
            }
        case string:
            if strings.HasPrefix(t, "-") {
                full, res, stems = match(ctx, p.Value, t[1:])
                if s, y := res.(string); y { res = "-" + s }
                if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
            }
        default:
            erro(ctx, "%v → %v", p, ts(val)).trace()
        }
		return
    case *percpat:
        switch t := val.(type) {
        case *filestub:
            if full, res, stems = p.match1(ctx, t.name); full || res != "" {
                if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
            } else if t.dir != "" || t.sub != "" {
                full, res, stems = p.match1(ctx, filepath.Join(t.dir, t.sub, t.name))  // NOTE: matching the fullname form
                if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
            }
        case *file:
            if full, res, stems = p.match1(ctx, ident(ctx, t)); full || res != "" {
                if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
            } else if t.dir != "" || t.sub != "" {
                full, res, stems = p.match1(ctx, t.fullname()) // NOTE: matching the fullname form
                if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
            }
        case string:
            full, res, stems = p.match1(ctx, t)
            if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
        case Value:
            full, res, stems = p.match1(ctx, __string(ctx, t))
            if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
        default:
            unreachable(fmt.Sprintf("match: %s", ts(val)))
        }
		return
    case *globbrace: return match(ctx, &p.globpat, val)
    case *globpat:
        var s string
        switch t := val.(type) {
        case      bare: s = string(t)
        case *filestub: s = t.name
        case     *file: s = ident(ctx, t)
        case     Value: s = __string(ctx, t)
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
            erro(ctx, "%v : unsupported match type: %v %v", p, val, ts(val)).trace()
        }

        var err error
        var pattern = __string(ctx, p)
        if full, stems, err = globMatch(ctx, pattern, s); full { res = s }
        if err != nil { erro(ctx, "%v : glob error: %v", p, err).trace() }
        if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
		return
    case *regexpat:
        if p.Regexp != nil {
            var str string
            switch t := val.(type) {
            case *filestub: str = t.name
            case     *file: str = ident(ctx, t)
            case     Value: str = __string(ctx, t)
            case    string: str = t
            case  []string: if len(t) == 1 { str = t[0] } else { return }
            default:
                erro(ctx, "%v :matching unsupported value: %s", p, ts(val)).trace()
            }

            if sm := p.Regexp.FindStringSubmatch(str); sm != nil && sm[0] == str {
                full, res, stems = true, sm[0], sm[1:]
                if checkpoints { check_match(pc(ctx,p), pat, val, full, res, stems) }
            }
        }
		return
    case *delegate, *closure:
		if t1, ok := p.(Value); !ok {
			erro(ctx, "incompatible value: %s", ts(p)).trace()
		} else if t2 := expand(ctx, t1); t2 != nil {
			if !equal(ctx, t2, t1) { return match(ctx, t2, val) }
		} else {
			erro(pc(ctx,p), "%v: nil match", t1).trace()
		}
		return
    case compositePattern:
		if full, res, stems = match(ctx, p.Value, val); full {
			for _, con := range p.constraints {
				if a, b, c := match(ctx, con, val); !a { return a, b, c }
			}
		}
		return full, res, stems
	case filemap:
		var s string
		full, res, s = p.match(ctx, val)
		stems = []string{ s }
		return
	case nil: return
    }

	erro(ctx, "TODO: match(%v, %v) | %v | %v, %v", pat, val, vals, ts(pat), ts(val)).trace()
	return
}

func stencil(ctx Context, pat any, stems []string) (res Value, rest []string) {
    switch p := pat.(type) {
    case *loc: return stencil(ctx, p.Value, stems)
    case *rule: return stencil(ctx, p.target, stems)
    case flag:
        var name Value
        name, rest = stencil(ctx, p.Value, stems)
        res = flag{name}
        return
    case *compound:
        v := new(compound)
        res, rest = v, stems
        for _, elem := range p.elems {
            var t Value
            t, rest = stencil(ctx, elem, rest)
            v.elems = append(v.elems, t)
        }
        return
    case *path:
        v := new(path)
        res, rest = v, stems
        for _, elem := range xmerge(ctx, p.elems...) {
            var t Value
            if t, rest = stencil(ctx, elem, rest); isTrivial(t) { t = elem }
            v.elems = append(v.elems, t)
        }
        return
    case *list:
        v := new(list)
        res, rest = v, stems
        for _, elem := range p.elems {
            var t Value
            t, rest = stencil(ctx, elem, rest)
            v.elems = append(v.elems, t)
        }
        return
    case *pair:
        v := new(pair)
        v.key, rest = stencil(ctx, p.key, stems)
        v.val, rest = stencil(ctx, p.val, rest)
        return
    case *percpat:
        var vals []Value
        if isTrivial(p.Prefix) {
            // does nothing
        } else if patterned(ctx,p.Prefix) {
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
        // if isTrivial(p.Suffix) {
        //     // done
        // } else if p.Suffix.patterned(ctx) {
        //     // FIXME: patterns like '%xxx%...' use multiple stems.
        //     if suf, res := p.Suffix.stencil(ctx, rest); suf != p.Suffix {
        //         // NOTE: patterns like '%%...' uses only one stem,
        //         suffix, rest = suf, res
        //     }
        // } else {
        //     suffix = p.Suffix
        // }
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
        } else if suf, res := stencil(ctx, suffix, rest); !isNull(suf) && suf != suffix {
            // NOTE: patterns like '...%xxx%...' use multiple stems.
            vals, rest = append(vals, suf), res
        } else {
            vals, rest = append(vals, suffix), res
        }

    DoneVals:
        if len(vals) == 1 {
            res = vals[0]
        } else {
            res = &compound{elements{vals}}
        }
        return
    case *barefile:
        var name Value
        if p.file != nil {
            res, rest = stencil(ctx, p.file, stems)
        } else if name, rest = stencil(ctx, p.Value, stems); name != p.Value {
            res = &barefile{name, p.file}
        } else {
            res = p
        }
        return
    case Value:
        return p, stems
    }
    erro(pc(ctx,pat), "stencil: %v %v", ts(pat), stems).trace()
    return
}

func patterned(ctx Context, v any) (res bool) {
    switch t := v.(type) {
    case []Value: for _, t := range t { if patterned(ctx, t) { return true } }
    case *regexpat, *percpat, *globpat, *globrange: return true
    case *loc: return patterned(ctx, t.Value)
    case flag: return patterned(ctx, t.Value)
    case *barefile: return patterned(ctx, t.Value)
    case *strval: return patterned(ctx, t.v)
    case *strcomp: return patterned(ctx, t.elems)
    case *compound: return patterned(ctx, t.elems)
    case *list: return patterned(ctx, t.elems)
    case *group: return patterned(ctx, t.elems)
    case *pair: return patterned(ctx, t.key) || patterned(ctx, t.val)
    case *arrow: return patterned(ctx, t.o) || patterned(ctx, t.s)
    case *closure: return patterned(ctx, t.x) || patterned(ctx, t.o) || patterned(ctx, t.a)
    case *delegate: return patterned(ctx, t.x) || patterned(ctx, t.o) || patterned(ctx, t.a)
    case *path: return patterned(ctx, t.elems)
    case *rule: return patterned(ctx, t.target)
    case *recipe: return patterned(ctx, t.elems)
    case *disjunction: return patterned(ctx, t.val)
    case *argumented: return patterned(ctx, t.Value) || patterned(ctx, t.args)
    case *quote: return patterned(ctx, t.elems)
    case conjunction: return patterned(ctx, t.list.elems) || patterned(ctx, t.sep)
    case fullname: return patterned(ctx, t.Value)
    case negative: return patterned(ctx, t.Value)
    case *plain: return patterned(ctx, t.elems)
    case *plainline: return patterned(ctx, t.elems)
    case *returner: return patterned(ctx, t.vals)
    case *undetermined: return patterned(ctx, t.identifier) || patterned(ctx, t.value)
    case *url: return patterned(ctx, t.Scheme) || patterned(ctx, t.Username) || patterned(ctx, t.Password) ||
        patterned(ctx, t.Host) || patterned(ctx, t.Port) || patterned(ctx, t.Path) ||
        patterned(ctx, t.Query) || patterned(ctx, t.Fragment)
    }
    return false
}

func stamp(ctx Context, a any) (res []*file) {
    switch t := a.(type) {
    case *loc: return stamp(ctx, t.Value)
    case flag: return stamp(ctx, t.Value)
    case *barefile: return stamp(ctx, t.Value)
    case *strval: return stamp(ctx, t.v)
    case *strcomp: return stamp(ctx, t.elems)
    case *compound: return stamp(ctx, t.elems)
    case *list: return stamp(ctx, t.elems)
    case *group: return stamp(ctx, t.elems)
    case *path: return stamp(ctx, t.elems)
    case *rule: return stamp(ctx, t.target)
    case *recipe: return stamp(ctx, t.elems)
    case *globpat: return stamp(ctx, t.elems)
    case *globrange: return stamp(ctx, t.Value)
    case *argumented: return stamp(ctx, t.Value)
    case *disjunction: return stamp(ctx, t.val)
    case *quote: return stamp(ctx, t.elems)
    case fullname: return stamp(ctx, t.Value)
    case negative: return stamp(ctx, t.Value)
    case *plain: return stamp(ctx, t.elems)
    case *plainline: return stamp(ctx, t.elems)
    case *returner: return stamp(ctx, t.vals)
    default:
        erro(pc(ctx,a), "%v", ts(a)).trace()
        return
    }
}

func stat(ctx Context, a any) (res []*statinfo) {
    switch t := a.(type) {
    case *loc: return stat(ctx, t.Value)
    case flag: return stat(ctx, t.Value)
    case *barefile: return stat(ctx, t.Value)
    case *strval: return stat(ctx, t.v)
    case *strcomp: return stat(ctx, t.elems)
    case *compound: return stat(ctx, t.elems)
    case *list: return stat(ctx, t.elems)
    case *group: return stat(ctx, t.elems)
    case *path: return stat(ctx, t.elems)
    case *rule: return stat(ctx, t.target)
    case *recipe: return stat(ctx, t.elems)
    case *argumented: return stat(ctx, t.Value)
    case *disjunction: return stat(ctx, t.val)
    case *quote: return stat(ctx, t.elems)
    case fullname: return stat(ctx, t.Value)
    case negative: return stat(ctx, t.Value)
    default:
        erro(pc(ctx,a), "%v", ts(a)).trace()
        return
    }
}

func traverse(ctx Context, val Value) {
    switch p := val.(type) {
    case *loc: traverse(ctx, p.Value)
    case *list: for _, e := range p.elems { traverse(ctx, e) }
    case *argumented: traverse(p.ctx(ctx), p.Value)
    case *auto: if v := auto_get(ctx, p.name); v != nil { traverse(ctx, v) }
    case *def: if v := p.value; v != nil { traverse(ctx, v) }
    case negative: if v := p.Value; v != nil { traverse(ctx, v) }
    case *project:
        if t := p.main; t != nil {
            switch t.destiny().(type) { case flag: return }
            traverse(ctx, t)
        }
    case *stemmed_rule:
        // NOTE: Make a clone of the underlying rule for traversing the real target;
        //       the underlying rule target is readonly, it must not be changed, for
        //       next traversal be done correctly.
        var t = *p.rule // TODO: consider not copying the rule, use pointer instead
        t.target = p.target
        traverse(&stemmed_ctx{ctx, p}, &t)
    case *rule:
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
    case *arrow:
        if val := expand(ctx,p); !isTrivial(val) {
            updated(ctx, val) // NOTE: ensure that updated flag is correct (see rule.updated)
            traverse(ctx, val)
        } else {
            warn(ctx, "selected trivial value '%v' (%v, %v) ", p, ts(p.o), ts(p.s)).debug(10)
        }
    case *barefile:
        if p.file != nil { traverse(ctx, p.file) } else
        if p.Value != nil { traverse(ctx, p.Value) }
    case *file:
        if p._traved == 0 && !p.isSysFile() {
            do(ctx, act_traverse{p})
        } else if x := _execution(ctx); x != nil {
            x.traved(ctx, auto_target_value(ctx), p, nil, p)
        }
    case *modifier:
        if name := __string(ctx, p.elems[0]); name == "" {
            erro(ctx, "empty name: %v", p.elems[0]).trace()
        } else if truly(ctx, interpret{name, p.elems[1:]}) {
            modify(ctx, &p.group, true)
        }
    case *modification:
        if e := _execution(ctx); e != nil { e.Wait() }
        for _, m := range p.list { traverse(ctx, m) }
    case *compound, *word, *strlit, *strval, *strcomp, *qualword, *path, *percpat, *globpat, *regexpat, flag:
        do(ctx, act_traverse{p})
    default:
        erro(pc(ctx,p), "unsupported traversal: %v", ts(val)).trace()
    }
}

func unique(ctx Context, values ...Value) (elems []Value) {
    seen := make(map[uint64]struct{}, len(values))
    for _, v := range values {
        var n = hash(ctx, v)
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
        var n = hash(ctx, v)
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

func ease(ctx Context, a any) (res Value) {
    if a == nil { return }

    var elems []Value

    switch t := a.(type) {
    case    Value: elems = append(elems, merge(t   )...)
    case  []Value: elems = append(elems, merge(t...)...)
    case    bare : elems = append(elems, _word(   _position(ctx),  string(t)))
    case    bool : elems = append(elems, _boolean(_position(ctx),         t ))
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

    switch len(elems) {
    case 0: return _null(_position(ctx))
    case 1: return elems[0]
    default: return &list{elements{elems}}
    }
}

func scalarize(v Value) (res Value) { // NOTE: unexpanded is not scalar
    switch t := v.(type) {
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
    if i == nil { return "{=nil}" }

    var t = typeof(i)

    if strings.HasPrefix(t, "[]") {
        v := reflect.ValueOf(i)
        s  = "["
        for i := 0; i < v.Len(); i += 1 {
            if 0 < i { s += " " }
            s += tv(v.Index(i).Interface())
        }
        s += "]"
        return
    }

    switch x := i.(type) {
    case *null, *none:
    case *compound, *list:
        var t = x.(interface{ slice(int,...int) []Value })
        for _, v := range t.slice(0) { if s != "" { s += " " }; s += tv(v) }
        if s != "" { s = ":"+s }
    case interface{ String() string }: s = ":"+x.String()
    default: s = fmt.Sprintf(":%v", i)
    }
    return "{"+t+s+"}"
}
func ts(i any, o ...any) (s string) {
    if i == nil { return "{}" }

    var c Context
    for _, o := range o {
        switch t := o.(type) {
        case Context: c = t
        case nil: c = nil
        default: panic(o)
        }
    }

    var t = typeof(i)
    if strings.HasPrefix(t, "[]") {
        v := reflect.ValueOf(i)
        s  = "["
        for i := 0; i < v.Len(); i += 1 {
            if 0 < i { s += " " }
            s += ts(v.Index(i).Interface(),o...)
        }
        s += "]"
        return
    }

    switch x := i.(type) {
    case interface{ ts(Context,string) string }: return x.ts(c,t)
    case interface{ ts(string) string }: return x.ts(t)
	case  filemap: return "{=filemap "+x.String()+"}"
    case  Context: return "{="+t+" "+ts(inner(x),o...)+"}"
    case *valbase: return "{"+lp(c, x.position, "")+"}"
    case Value:
        if t := x.String(); t != "" { s += " " + strings.ReplaceAll(t, "\n", `\n`) }
        return "{"+lp(c, x.Position(), t)+s+"}"
    default:
        if t := fmt.Sprintf("%v", x); t != "" { s += " " + strings.ReplaceAll(t, "\n", `\n`) }
        return "{="+t+s+"}"
    }
}

type tst struct{ i any }
func (p tst) ts(ctx Context, _ string) string { return ts(p.i, ctx) }
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
func _none(pos Position) *none { return &none{valbase{pos}/*,nil*/} }
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
    if prefix == nil { prefix = &valbase{pos} }
    if suffix == nil { suffix = &valbase{pos} }
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

// time.RFC3339Nano
func parseDateTime(s string) (time.Time, error) { return time.Parse("2006-01-02T15:04:05.999999999Z07:00", s) }
func parseDate(s string) (time.Time, error) { return time.Parse("2006-01-02", s) }
func parseTime(s string) (time.Time, error) { return time.Parse("15:04:05.999999999Z07:00", s) }

func ParseDateTime(pos Position, s string) Value {
    if t, e := parseDateTime(s); e == nil {
        return &datetime{valbase{pos},t}
    } else {
        panic(e)
    }
}

func ParseDate(pos Position, s string) *Date {
    if t, e := parseDate(s); e == nil {
        return makeDate(pos,t)
    } else {
        panic(e)
    }
}

func ParseTime(pos Position, s string) *Time {
    if t, e := parseTime(s); e == nil {
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

type evoke_x           struct{ name string }
type evoke_builtin     struct{ name string }
type evoke_def         struct{ name string }
type evoke_detect_loop struct{ Value }
type evoke_count       struct{}

type evocation struct{
    automatic
    x   Value
    o []Value
    a []Value
}
func (p *evocation) inner() Context { return &p.automatic }
func (p *evocation) cast(t reflect.Type) Context {
    if reflect.TypeOf((*automatic)(nil)) == t { return &p.automatic }
    if reflect.TypeOf(p) == t { return p }
    return p.automatic.cast(t)
}
func (p *evocation) ts(t string) string {
    var s = p.defs.String()
    if s != "" { s += " " }
    return "{="+t+" "+p.x.String()+" | "+s+ts(p.Context)+"}"
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
        if p.x != nil && (t.name == "" || t.name == ident(ctx, p.x)) {
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

type opts struct{ vals []Value }
type opt  struct{ Value }
func (p opt) ts(ctx Context, t string) string { return "{="+t+" "+ts(p.Value, ctx)+"}" }

func call(ctx Context, name string, o []Value, a ...Value) (res Value) {
    if v := _universe(ctx).lookup(name); v != nil { res = evoke(ctx, v, o, a) }
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
