//
//  Copyright (C) 2012-2026, Duzy Chan <code@extbit.io>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//

package smart

import (
    "sync"
    "os/exec"
	"bytes"
	encbin "encoding/binary"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash/fnv" // "hash/maphash"
	"io/fs"
	"math"
	neturl "net/url"
	"path/filepath"
    "plugin"
	"reflect"
	"regexp"
	"runtime"
	rt_debug "runtime/debug"
	"strconv"
	"strings"
	"sort"
	"time"
	"unsafe"
	"unicode/utf8"
	"os"
	"io"
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
	cmpLprefix cmpres = -2 // L is smaller then R, and L is the prefix of R
	cmpSmaller cmpres = -1 // L is smaller then R
	cmpEqual   cmpres =  0 // L is equal to R
	cmpGreater cmpres =  1 // L is greater than R
	cmpRprefix cmpres =  2 // L is greater than R, and R is the prefix of L
)

const (
    existenceMatterless existence = 1<<iota
    existenceConfirmed
    existenceNegated
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
func (n cmpres) String() string {
    switch n {
    case cmpLprefix: return "lprefix"
    case cmpSmaller: return "smaller"
    case cmpEqual:   return "equal"
    case cmpGreater: return "greater"
    case cmpRprefix: return "rprefix"
    }
    return "unknown"
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
func sf(f string, i ...any) string { return fmt.Sprintf(f, i...) }
func ssf(ss []string, a ...any) (res []string) {
    for _, s := range ss {
		if strings.Contains(s, "%") { s = fmt.Sprintf(s, a...) }
		res = append(res, s)
	}
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

type negate struct{ Context ; property }
func (c negate) cast(t reflect.Type) Context { return icast(c, t) }
func (c negate) inner() Context { return c.Context }
func (c negate) do(ctx Context, op any) (_ any) {
    if x, y := op.(property); y && x&c.property != 0 { return }
    return c.Context.do(ctx, op)
}

// A comment node represents a single #-style comment.
type comment struct{
    pos Pos // position of "#" starting the comment
    string // comment text (excluding '\n')
}
func (c *comment) String() string { return "{"+c.string+"}" }
func (c *comment) Pos() Pos { return c.pos }

// A commentgroup represents a sequence of comments
// with no other tokens and no empty lines between.
type commentgroup struct{ comments []*comment }
func (g *commentgroup) Pos() Pos { return g.comments[0].pos }

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
			if s := bases(2, x.abs, "testdata", true); s == c.comment {
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
		if c.scope != nil { res = append(res, c.scope) }
		if cc := c.Context; cc != nil {
			if x, y := do(cc, t).([]*scope); y && x != nil { res = append(res, x...) }
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
		// NOTE: the globe scope has nil project; closure_scopes already
		//       puts the derived polymorphic context at Index 0.
		if s.project != nil {
			if _, ok := m[s.project]; !ok {
				for _, p := range s.project.family() { m[p] = struct{}{} }
				res = append(res, s.project)
			}
		}
	}
	return
}

func closure_resolve(ctx Context, name string) object {
	for _, s := range closure_scopes(ctx) {
		var p = s.project
		if p == nil || s != p.scope {
			if _, o := s.find(name); !isNull(o) {
				if a, ok := o.(*auto); ok { if d := auto_find(ctx, a.name); d != nil { return d } }
				return o
			}
		}
		if p != nil { if o := p.resolve(ctx, name); !isNull(o) { return o } }
	}
	return nil
}

func closure_entries(ctx Context, name any) []entry {
	for _, p := range closure_projects(ctx) {
		if t := p._entries(ctx, name, false); t != nil { return t }
	}
	return nil
}

func closure_entry(ctx Context, name any, _ ...bool) entry {
	for _, p := range closure_projects(ctx) {
		if t := p._entries(ctx, name, false); t != nil { return t[0] }
	}
	return nil
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
        debug(ctx, "hashing recipes failed: %v", err, trace{})
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
        debug(ctx, "compute recipes hash failed: %v", err, trace{})
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
        debug(ctx, "target is nil", trace{})
    }

    if isTrivial(target) {
        debug(ctx, "trivial target : %s", ts(target), trace{})
    }

    if n := len(calleeErrs); n > 0 /*&& t.stems == nil*/ {
        var numRealErrs = 0
        for _, err := range calleeErrs {
            debug(ctx, "%v: %v", target, err, trace{})
            numRealErrs += 1
        }
        if numRealErrs == 0 { return } // simply return if no real errors

        var ctxPos, targetPos = _pos(ctx), target.Pos()
        var v = target
        if l, ok := v.(*list); ok && l.len() == 1 { v = l.elems[0] }
        if targetPos.IsValid() && targetPos != ctxPos {
            if f, y := to_file(v); y && f != nil && f.filemap != nil {
                debug(ctx,
					_f("waiting for '%v'", target),
					_f("via pattern '%v' (of %v)", v, f.filemap.project),
					trace{})
            } else {
                debug(ctx,
					_f("waiting for '%v'", target),
					trace{})
            }
        }
        if def, ok := v.(*def); ok && target != v && target != def.value { // trace source Def in diagnostics
            debug(ctx, "waiting for def '%v': %v", def.name, def.value, trace{})
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

func as_file(ctx Context, val Value, projs ...*project) (res *file) {
    var v = scalarize(expand(ctx, val))
    if x, y := v.(fullname); y { v = x.Value }
    switch t := v.(type) {
    case  fullfile: return t.file
    case *barefile: return t.file
    case *file: return t
    case *rule: return as_file(ctx, t.target, projs...)
    case *def : if t.value != nil { return as_file(ctx, t.value, projs...) }
    case *list: if a := t.elems; len(a) == 1 { return as_file(ctx, a[0], projs...) }
    case *word, *qualword, *compound, *path:
        if projs == nil {
            if p := _project(ctx); p != nil { projs = append(projs, p) }
        }
        for _, p := range projs {
            if f := p.file(ctx, t); f != nil { return f }
        }
    default: // *strlit, *strval, *strcomp:
        // NOTE: not parsing 'string' and "strcomp" values to file to optimize.
        // debug(pc(ctx,a), "cannot convert to file: %v", ts(a.Value), trace{})
    }
    return
}
func as_fullname_file(ctx Context, val Value, projs ...*project) (f *file, s string, y bool) {
    if f = as_file(ctx, val, projs...); f != nil {
        s = f.fullname()
		y = filepath.IsAbs(s)
    }
    return
}
func as_fullname_string(ctx Context, val Value, projs ...*project) (s string, y bool) {
    if _, s, y = as_fullname_file(ctx, val, projs...); !y { s = __string(ctx, val) }
    return
}
func as_fullname(ctx Context, val Value, projs ...*project) (res fullname) {
	if f := as_file(ctx, val, projs...); f != nil {
		res.Value = f
	} else {
		debug(pc(ctx,val), "nil file : %v : %v → %v", val, ts(val), expand(ctx,val),
			callstack{num:10}, trace{})
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
	default: debug(ctx, "err splitpath: %v %s", a, ts(a), callstack{num:10}, trace{})
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
    s = bases(3, s, true)
    for _, x := range []string{"/testdata/","/smart/"} {
        if i := strings.Index(s, x); i > 0 { s = "…/"+s[i+len(x):]; break }
    }
    return s
}

type posctx struct{ Context ; pos any }
func (p *posctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *posctx) inner() Context { return p.Context }
func (p *posctx) ts(string) string {
	var pos Position
	
	// Dynamically resolve Pos -> Position for tracing
	switch t := p.pos.(type) {
	case Position:
		pos = t
	case Pos:
		if t.IsValid() {
			if u := _universe(p.Context); u != nil && u.fset != nil {
				pos = u.fset.Position(t)
			}
		}
	case positioner:
		if x := t.Pos(); x.IsValid() {
			if u := _universe(p.Context); u != nil && u.fset != nil {
				pos = u.fset.Position(x)
			}
		}
	}

	if pos.valid() {
		var s = posstr(pos.Filename)
		if pos.Column == 0 { return fmt.Sprintf("{%s:%d %s}", s, pos.Line, ts(p.Context)) }
		return fmt.Sprintf("{%s:%d:%d %s}", s, pos.Line, pos.Column, ts(p.Context))
	}
	return fmt.Sprintf("{%v %s}", p.pos, ts(p.Context))
}

func (p *posctx) do(ctx Context, op any) (_ any) {
	switch op.(type) {
	case get_position:
		if p.pos != nil {
			switch t := p.pos.(type) {
			case Pos:        if t.IsValid() { return t }
			case Position:   if t.valid()   { return t }
			case positioner: if x := t.Pos(); x.IsValid() { return x }
			}
		}
	}
	if p.Context == nil { return nil } // CRITICAL FIX: Prevent nil pointer dereference
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
		case *loc: return &posctx{pc(ctx,t.Value), t.pos}
		case *xloc: return &posctx{pc(ctx,t.Value), t.pos} // CRITICAL FIX: Extract fat Position
		case  *scanner   : p = t.pos(n...)
		case  *parser    : p = t.pos
		case   Pos       : if t.IsValid() { p = t }
		case   Position  : if t.valid()   { p = t }
		case   positioner: if x := t.Pos(); x.IsValid() { p = x }
		case []positioner:
			for _, v := range t {
				if x := v.Pos(); x.IsValid() { // FIX: Use IsValid() for compact Pos
					p = x
					break
				}
			}
		case []Value:
			for _, v := range t {
				if x := v.Pos(); x.IsValid() { // FIX: Use IsValid() for compact Pos
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

type chksrc string
type source struct{}
type srcfun struct{}
type srcpos struct{}
type srcctx struct{ posctx ; f *runtime.Func ; a any }
func (p *srcctx) cast(t reflect.Type) Context { return icast(p,t) }
func (p *srcctx) inner() Context { return &p.posctx }
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

func src(ctx Context, a any) Context {
	if pc, file, line, ok := runtime.Caller(1); ok {
		var p = Position{} ; p.Filename, p.Line = file, line
		// if a == nil && loader_pos.Line == 0 && loader_pos.Filename == "" {
		// 	loader_pos, loader_src = p, srctag(p)
		// }
		return &srcctx{posctx{ctx,p}, runtime.FuncForPC(pc), a}
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

type positioner interface{ Pos() Pos }
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
func ident(ctx Context, x Value) string {
	switch t := x.(type) {
	case *loc:
		return ident(ctx, t.Value)
	case *xloc:
		return ident(ctx, t.Value)
	case *closure:
		return ident_opt(ctx, "&", x, closure_ident{})
	case *delegate:
		return ident_opt(ctx, "$", x, delegate_ident{})
	case *file:
		return t.filestub.name
	case *project:
		return t.name
	case *uselist:
		return t.name
	case *def:
		return t.name
	case *defcaps:
		return ident(ctx, t.Value) // The main captured value (e.g., $0)
	case *word:
		return t.s
	case *raw:
		return t.s
	case *strlit:
		return `'` + t.s + `'`
	case *punct:
		return t.token.String()
	case *globmeta:
		return t.token.String()
	case *valbase, *null, *none, nil:
		return ""
	case *answer, *boolean, *binary, *octal, *decimal, *hexadecimal, *float, *datetime, *Date, *Time, *globrange:
		return t.String()
	case *arrow:
		return ident(ctx, t.o) + t.t.String() + ident(ctx, t.s)
	case *rule:
		return ident(ctx, t.target)
	case *compound:
		var b strings.Builder
		for _, elem := range t.elems {
			if lst, ok := elem.(*list); ok && lst.len() > 0 {
				b.WriteString("⌜")
				b.WriteString(ident(ctx, elem))
				b.WriteString("⌟")
			} else {
				b.WriteString(ident(ctx, elem))
			}
		}
		return b.String()
	case *qualword:
		var b strings.Builder
		for i, elem := range t.elems { if i > 0 { b.WriteByte('.') }
			if lst, ok := elem.(*list); ok && lst.len() > 0 {
				b.WriteString("⌜")
				b.WriteString(ident(ctx, elem))
				b.WriteString("⌟")
			} else {
				b.WriteString(ident(ctx, elem))
			}
		}
		return b.String()
	case *globpat:
		var b strings.Builder
		for _, elem := range t.elems {
			b.WriteString(ident(ctx, elem))
		}
		return b.String()
	case *strcomp:
		var b strings.Builder
		b.WriteByte('"')
		for _, elem := range t.elems {
			b.WriteString(ident(ctx, elem))
		}
		b.WriteByte('"')
		return b.String()
	case *list:
		var b strings.Builder
		for i, elem := range t.elems {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(ident(ctx, elem))
		}
		return b.String()
	case *group:
		var b strings.Builder
		b.WriteByte('(')
		for i, elem := range t.elems {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(ident(ctx, elem))
		}
		b.WriteByte(')')
		return b.String()
	case *pair:
		var b strings.Builder
		if t.key != nil {
			b.WriteString(ident(ctx, t.key))
		}
		b.WriteByte('=')
		if t.val != nil {
			b.WriteString(ident(ctx, t.val))
		}
		return b.String()
	case *undetermined:
		return __string(ctx, t.identifier)
	case *disjunction:
		return "{" + ident(ctx, t.val) + "}"
	case self:
		return ".self"
	case flag:
		return "-" + ident(ctx, t.Value)
	case negative:
		return "!" + ident(ctx, t.Value)
	case identer:
		return t.ident(ctx)
	case *url:
		var b strings.Builder
		b.WriteString(ident(ctx, t.Scheme))
		b.WriteByte(':')
		if t.Host != nil {
			b.WriteString("//")
			if t.Username != nil {
				b.WriteString(ident(ctx, t.Username))
				b.WriteByte('@')
			}
			b.WriteString(ident(ctx, t.Host))
			if t.Port != nil {
				b.WriteByte(':')
				b.WriteString(ident(ctx, t.Port))
			}
		}
		if t.Path != nil {
			b.WriteString(ident(ctx, t.Path))
		}
		if len(t.Query) > 0 {
			b.WriteByte('?')
			b.WriteString(ident(ctx, t.Query[0]))
			for _, q := range t.Query[1:] {
				b.WriteByte('&')
				b.WriteString(ident(ctx, q))
			}
		}
		if t.Fragment != nil {
			b.WriteByte('#')
			b.WriteString(ident(ctx, t.Fragment))
		}
		return b.String()
	default:
		debug(pc(ctx, x), "no ident for %s", x, callstack{num:24}, trace{})
		return ""
	}
}
func ident_any(ctx Context, x any) string {
	switch t := x.(type) {
	case  Value: return ident(ctx, t)
	case string: return t
	}
	return ""
}

func ident_opt(ctx Context, pre string, x Value, op any) (_ string) {
    if truly(ctx, ex_closure{}) {
        var ic *ident_ctx
        // CRITICAL FIX: Use identity_ctx to safely initialize the context if it is nil!
        if ic, ctx = identity_ctx(ctx); ic.nil == 0 {
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
	var o = encbin.LittleEndian
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
					b = []byte(__string(ctx, t))
				}
			}
		case []Value:
			for _, t := range t {
				if t.kind()&(KindBoolean|KindInteger|KindFloat|KindDateTime) != 0 {
					b = o.AppendUint64(b, uint64(__int(ctx, t)))
				} else {
					b = []byte(__string(ctx, t))
				}
			}
		default:
			debug(ctx, "fnv1: unsupported type : %s", ts(v), trace{})
		}
		h.Write(b)
	}
	return h.Sum64()
}

func hash(ctx Context, x Value) (u uint64) {
	switch p := x.(type) {
	case *loc: return hash(ctx, p.Value)
	case *xloc: return hash(ctx, p.Value)
	case *returner: return fnv1(ctx, p, p.vals)
	case *argumented: return fnv1(ctx, p, p.Value, p.args)
	case *list: return fnv1(ctx, p, p.any()...)
	case *quote: return fnv1(ctx, p, p.any()...)
	case *strcomp: return fnv1(ctx, p, p.any()...)
	case *compound: return fnv1(ctx, p, p.any()...)
	case *qualword:
		var a []any
		for _, v := range unpack(p) { a = append(a, v) }
		return fnv1(ctx, p, a...)
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
    case *xloc: return optional(t.Value)
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

func unloc(a Value) Value {
	switch a := a.(type) {
	case *loc: return unloc(a.Value)
	case *xloc: return unloc(a.Value)
	default: return a
	}
}

var implicitDot = &punct{valbase{}, DOT}

func unpack(v Value) []Value {
	switch t := v.(type) {
	case *loc: return unpack(t.Value)
	case *xloc: return unpack(t.Value)
	case *compound: return t.elems
	case *globpat: return t.elems
	case *qualword:
		if len(t.elems) == 0 { return nil }
		// Pre-allocate the exact slice capacity to avoid append() resizing!
		res := make([]Value, 0, len(t.elems)*2-1)
		for i, e := range t.elems {
			if i > 0 {
				res = append(res, implicitDot) // Zero heap allocation!
			}
			// CRITICAL FIX: Drop empty placeholders so .deps unpacks 
			// structurally into [punct(.), word(deps)] instead of [valbase, punct(.), word(deps)]
			if !isEmpty(e) {
				res = append(res, e)
			}
		}
		return res
	}
	return []Value{v}
}

func unbox(a any) any {
    switch t := a.(type) {
    case self: return t.project
	case fullname: return unbox(t.Value)
    case *loc: return unbox(t.Value)
    case *xloc: return unbox(t.Value)
    case *list: if len(t.elems) == 1 { return unbox(t.elems[0]) }
    case flag: return flag{unbox(t.Value).(Value)}
		// case *pair: return &pair{unbox(t.key).(Value), unbox(t.val).(Value)}
    }
    return a
}

// underlay convert 'a' into native value underneath (aka bool, int, etc.).
func underlay(a any, b int) (any, int) {
	if a == nil { return "", 0 } // Safely handle nil interfaces
    switch t := a.(type) {
	case *valbase, *null, *none, *undef: return "", 0 // Unbox empty AST nodes to ""
    case *loc: return underlay(t.Value, b)
    case *xloc: return underlay(t.Value, b)
	case fullname: return underlay(t.Value, b)
    case self: return t.project, 0
	case *globmeta: return t.token, 0
	case *punct: if t.token == PROOT || t.token == PTAIL { return "", 0 } else { return t.token, 0 }
	case *answer: return t.bool, 0
	case *option: return t.bool, 0
	case *prediction: return t.bool, 0
	case *boolean: return t.bool, 0
    case *word: return t.s, 0
    case *raw: return t.s, 0
	case *strlit: return t.s, 0          // ADDED: instantly maps to primitive string
	case *file: return t.filestub.name, 0 // ADDED: prevents deep file stringification
	case *project: return t.name, 0      // ADDED
    case *binary: return t.int64, 2
    case *octal: return t.int64, 8
    case *decimal: return t.int64, 10
    case *hexadecimal: return t.int64, 16
    case *float: return t.float64, 0
    case *datetime: return t.t, 0
    case *Date: return t.t, 1
    case *Time: return t.t, 2
	case []string:
		var res []any
		for _, s := range t { res = append(res, s) }
		return res, 0
	case []Value:
		var res []any
		for _, v := range t { res = append(res, v) }
		return res, 0
    }
    return a, b
}

func _underlay(a any) any { a, _ = underlay(a, 0); return a }

// globRank returns the specificity rank of a wildcard token/string.
// ? < * < *? ≈ **
func globRank(v any) int {
	switch t := v.(type) {
	case token:
		switch t {
		case QUE : return 1 // ?
		case SAST: return 2 // *
		case ASTQ: return 3 // *?
		case DAST: return 3 // **
		}
	case string:
		switch t {
		case  "?": return 1
		case  "*": return 2
		case "*?": return 3
		case "**": return 3
		}
	}
	return 0
}

// cmpValues is for slices.SortFunc(vals, cmp_values)
func cmpValues(ctx Context, a, b Value) int {
	if t := cmp(ctx, a, b); t == cmpEqual { return 0 } else { return int(t) }
}

func cmp_rank(rx, ry int) cmpres {
	if 0 < rx && 0 < ry {
		if rx < ry { return cmpSmaller }
		if rx > ry { return cmpGreater }
		return cmpEqual
	} else if 0 < rx {
		return cmpGreater
	} else if 0 < ry {
		return cmpSmaller
	}
	return cmpEqual // Indicates no rank disparity, proceed to normal cmp
}

func cmp(ctx Context, l, r any, syntactic ...bool) (res cmpres) {
	if checkpoints { defer check_cmp(ctx, l, r, syntactic)(&res) }

	// 1. Unbox/Underlay: convert wrappers (*word, *boolean) to primitives (string, bool)
	var lv, lb = underlay(l, 0)
	var rv, rb = underlay(r, 0)

	// 2. Structural & Recursive Comparison
	switch x := lv.(type) {
	case []any:
		return cmp_slice(ctx, x, rv)

	case *qualword: // CRITICAL FIX: Unpack dots for structural alignment!
		switch y := rv.(type) {
		case *qualword:
			// SUPER FAST PATH: Both are qualwords. 
			// The implicit dots perfectly align, so we just compare their elements!
			// (Assuming you have a way to iterate without allocating a new slice, 
			//  or you can safely pass their internal element slices)
			return cmp_slice(ctx, x.any(), y.any()) 
		case slicer:
			// Fallback: The right side is a globpat or compound. 
			// We MUST inject dots to align them structurally.
			return cmp_slice(ctx, _underlay(unpack(x)).([]any), _underlay(y.slice()).([]any))
		}

		// CRITICAL FALLBACK: The right side is a scalar (token, string, word).
		// Unpack the qualword into a slice and let the top-level cmp handle it.
		return cmp(ctx, _underlay(unpack(x)), r)

	case slicer: // *list, *compound, *path, *globpat, etc.
		return cmp(ctx, _underlay(x.slice()), r)

	case token:
		switch y := rv.(type) {
		case token:
			// Specificity: ? < * < *? ≈ **
			return cmp_rank(globRank(x), globRank(y))
		case string:
			if rx, ry := globRank(x), globRank(y); 0 < rx || 0 < ry {
				if res = cmp_rank(rx, ry); res != cmpEqual { return res }
			}
			return cmp_string(x.String(), y)
		case flag:
			if x == MINUS && isEmpty(y.Value) { return cmpEqual }
		case slicer:
			// Treat token as [token] vs slicer
			return cmp(ctx, []any{x}, y)
		}

	case string:
		switch y := rv.(type) {
		case string:
			// Specificity check for string patterns (e.g. "*")
			if rx, ry := globRank(x), globRank(y); 0 < rx || 0 < ry {
				if res = cmp_rank(rx, ry); res != cmpEqual { return res }
			}
			return cmp_string(x, y)
		case token:
			if rx, ry := globRank(x), globRank(y); 0 < rx || 0 < ry {
				if res = cmp_rank(rx, ry); res != cmpEqual { return res }
			}
			return cmp_string(x, y.String())
		case int64:   if i, e := strconv.ParseInt(x, lb, 64); e == nil { return cmp_int(i, y) }
		case float64: if f, e := strconv.ParseFloat(x, 64); e == nil { return cmp_float(f, y) }
		case slicer:
			// Treat string as [string] vs slicer
			return cmp(ctx, []any{x}, y)
		}

	case int64:
		switch y := rv.(type) {
		case int64:   return cmp_int(x, y)
		case float64: return cmp_float(float64(x), y)
		case string:  if i, e := strconv.ParseInt(y, rb, 64); e == nil { return cmp_int(x, i) }
		case slicer:  return cmp(ctx, []any{x}, y)
		}

	case float64:
		switch y := rv.(type) {
		case float64: return cmp_float(x, y)
		case int64:   return cmp_float(x, float64(y))
		case string:  if f, e := strconv.ParseFloat(y, 64); e == nil { return cmp_float(x, f) }
		case slicer:  return cmp(ctx, []any{x}, y)
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
		case slicer: return cmp(ctx, []any{x}, y)
		}

	case flag:
		switch y := rv.(type) {
		case flag: return cmp(ctx, x.Value, y.Value)
		case token:
			if y == MINUS && isEmpty(x.Value) { return cmpEqual }
		case slicer: // Reverse check: -Ifoo vs [- I foo]
			// Delegate to cmp_slice with swapped args, invert result
			if res := cmp_slice(ctx, _underlay(y.slice()).([]any), x); res != cmpEqual {
				return -res // invert cmpSmaller <-> cmpGreater
			}
			return cmpEqual
		}

	case *arrow:
		if y, ok := rv.(*arrow); ok {
			if x.t != y.t {
				if x.t < y.t { return cmpSmaller } else { return cmpGreater }
			}
			if t := cmp(ctx, x.o, y.o); t != cmpEqual { return t }
			return cmp(ctx, x.s, y.s)
		}

	case *closure:
		if y, ok := rv.(*closure); ok {
			if t := cmp(ctx, x.x, y.x); t != cmpEqual { return t }
			if t := cmp(ctx, x.o, y.o); t != cmpEqual { return t }
			return cmp(ctx, x.a, y.a)
		}

	case *delegate:
		if y, ok := rv.(*delegate); ok {
			if t := cmp(ctx, x.x, y.x); t != cmpEqual { return t }
			if t := cmp(ctx, x.o, y.o); t != cmpEqual { return t }
			return cmp(ctx, x.a, y.a)
		}

	case *def:
		if y, ok := rv.(*def); ok && x.name == y.name {
			return cmp(ctx, x.value, y.value)
		}

	case *file:
		if y, ok := rv.(*file); ok {
			if x.filestub == y.filestub { return cmpEqual }
			return cmp_string(x.filestub.name, y.filestub.name)
		}
	}

	// 3. Universal Fallback: String Comparison
	// STRICTLY decoupled from __string(ctx, ...) to prevent deep AST expansion hell.
	// We rely purely on the structural .String() representation to establish a fast sorting order!
	var sx, sy string
	if x, ok := lv.(fmt.Stringer); ok { sx = x.String() } else { sx = fmt.Sprintf("%v", lv) }
	if y, ok := rv.(fmt.Stringer); ok { sy = y.String() } else { sy = fmt.Sprintf("%v", rv) }	
	return cmp_string(sx, sy)
}

// cmp_slice handles comparison for []any, including logic for compound vs flag.
func cmp_slice(ctx Context, x []any, rv any) cmpres {
	switch y := rv.(type) {
	case []any:
		var i int
		for ; i < len(x) && i < len(y); i++ {
			if res := cmp(ctx, x[i], y[i]); res != cmpEqual {
				// FIX: If either element is a wildcard, return the comparison result immediately.
				if globRank(_underlay(x[i])) > 0 || globRank(_underlay(y[i])) > 0 {
					return res
				}

				// STRICTLY decouple from __string to prevent AST expansion hell!
				// Attempt fragmentation ONLY if both items can be safely converted to static strings.
				sx, okx := staticStr(x[i])
				sy, oky := staticStr(y[i])

				// If both are safe, trivial strings, try fragmentation
				if okx && oky {
					if sx == sy { continue } 
					
					if len(sx) < len(sy) && strings.HasPrefix(sy, sx) {
						restX := x[i+1:]
						restY := append([]any{sy[len(sx):]}, y[i+1:]...)
						return cmp(ctx, restX, restY)
					}
					if len(sy) < len(sx) && strings.HasPrefix(sx, sy) {
						restX := append([]any{sx[len(sy):]}, x[i+1:]...)
						restY := y[i+1:]
						return cmp(ctx, restX, restY)
					}
				}
				return res
			}
		}
		if i == len(x) && i < len(y) { return cmpSmaller }
		if i < len(x) && i == len(y) { return cmpGreater }
		return cmpEqual

	case *qualword: // CRITICAL FIX: Unpack dots for right-hand alignment!
		return cmp_slice(ctx, x, _underlay(unpack(y)).([]any))

	case slicer:
		return cmp_slice(ctx, x, _underlay(y.slice()).([]any))

	case flag:
		// Logic to match split flags: e.g. [- I foo] vs -Ifoo
		var a, b cmpres
		for i := 0; i < len(x); i++ {
			if isEmpty(x[i]) {
				continue
			} else if a == 0 && b == 0 {
				switch a = cmp(ctx, x[i], MINUS); a {
				case cmpEqual:   // Matched first part "-"
				case cmpLprefix: // Partial match
				default: return cmp(ctx, x, []any{rv}) // Fallback
				}
			} else if a == cmpEqual && b == 0 {
				// Matched "-", now match value "Ifoo" or "I" then "foo"
				if b = cmp(ctx, x[i], y.Value); b != cmpEqual { return b }
			} else {
				// We have more elements but already matched the flag -> x is Greater
				return cmpGreater 
			}
		}
		if a == cmpEqual && b == cmpEqual { return cmpEqual }
		// Fallthrough to standard comparison if split logic didn't conclusively return
		return cmp(ctx, x, []any{rv})

	default:
		// Treat scalar as single-element list (e.g. [foo] vs foo)
		return cmp(ctx, x, []any{rv})
	}
}

func cmp_string(l, r string) cmpres {
	switch {
	case l < r: if strings.HasPrefix(r, l) { return cmpLprefix } else { return cmpSmaller }
	case l > r: if strings.HasPrefix(l, r) { return cmpRprefix } else { return cmpGreater }
	}
    return cmpEqual
}
func cmp_int(l, r int64) cmpres {
    switch {
    case l < r: return cmpSmaller
    case l > r: return cmpGreater
    }
	return cmpEqual
}
func cmp_float(l, r float64) cmpres {
    switch {
    case l < r: return cmpSmaller
    case l > r: return cmpGreater
    }
	return cmpEqual
}
func cmp_time(l, r time.Time) cmpres {
    switch {
    case l.Before(r): return cmpSmaller
    case l.After(r) : return cmpGreater
    }
	return cmpEqual // l.Equal(r)
}

func eq(x Context, a, b any) bool { return cmp(x, a, b) == cmpEqual }
func equal(x Context, a, b any, yes ...bool) (res bool) {
	if false && checkpoints {
		if s, t := sf("%v",a), sf("%v",b); (!res && s == t) || res && s != t {
			defer debug(pc(x,a), "%v: equal(%v, %v)", res, s, t, callstack{num:6})
		}
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
	switch t := v.(type) {
	case *valbase, *none, *null, *undef, nil: return true
	case *def: return isTrivial(t.value)
	case *loc: return isTrivial(t.Value)
	case *xloc: return isTrivial(t.Value)
	case *list: return isTrivial(t.elems)
	case *compound: return isTrivial(t.elems)
	case *qualword: return isTrivial(t.elems)
	case fullname: return isTrivial(t.Value)
	case *word: return t.s == ""
	case *raw: return t.s == ""
	case *pair: return isTrivial(t.key) && isTrivial(t.val) // CRITICAL FIX
	case []Value:
		for _, v := range t { if !isTrivial(v) { return } }
		return true
	}
	return
}
func isEmpty(v any) (_ bool) {
	switch t := v.(type) {
	case *valbase, *none, *null, *undef, nil: return true
	case *valcache: return len(t.a) == 0 && len(t.o) == 0 && len(t.v) == 0
	case *def: return isEmpty(t.value)
	case *loc: return isEmpty(t.Value)
	case *xloc: return isEmpty(t.Value)
	case *list: return isEmpty(t.elems)
	case *compound: return isEmpty(t.elems)
	case *qualword: return isEmpty(t.elems)
	case *strcomp: return isEmpty(t.elems)
	case *strval: return len(t.v) == 0
	case *strlit: return t.s == ""
	case *word: return t.s == ""
	case *raw: return t.s == ""
	case *pair: return isEmpty(t.key) && isEmpty(t.val) // CRITICAL FIX
	case []Value:
		for _, v := range t { if !isEmpty(v) { return } }
		return true
	}
	return
}
func isUndef(v any) (_ bool) {
    switch t := v.(type) {
    case *undef: return true
	case *loc: return isUndef(t.Value)
	case *xloc: return isUndef(t.Value)
    case *def: return t.value != nil && isUndef(t.value)
    }
    return
}
func isNone(v any) (_ bool) {
	switch t := v.(type) {
	case *none: return true
	case *loc: return isNone(t.Value)
	case *xloc: return isNone(t.Value)
	case *list: return t.len() == 0 || (t.len() == 1 && isNone(t.elems[0]))
	case *pair: return isNone(t.key) && isNone(t.val) // CRITICAL FIX
	case *qualword: 
		for _, e := range t.elems { if !isNone(e) { return false } }
		return true
	}
	return
}
func isNull(v any) (_ bool) {
	switch t := v.(type) {
	case *null, nil: return true
	case *loc: return isNull(t.Value)
	case *xloc: return isNull(t.Value)
	case *list: return t.len() == 1 && isNull(t.elems[0])
	case *pair: return isNull(t.key) && isNull(t.val) // CRITICAL FIX
	case *qualword: 
		for _, e := range t.elems { if !isNull(e) { return false } }
		return true
	}
	return
}

func prefix(ctx Context, x, y Value) (res Value) { // x+y ⇔ prefix+y
	defer check_prefix(ctx, "p", x, y, &res)

	y = unbox(y).(Value)

	switch tx := x.(type) {
	case *loc:
		return &loc{prefix(ctx, tx.Value, y), tx.pos}
	case *xloc:
		return &xloc{prefix(ctx, tx.Value, y), tx.pos}
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
		default: return &pair{tx.key, prefix(ctx, tx.val, y)}
		}
	case *path:
		return &path{elements{append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], y))}}
	case *globpat:
		switch ty := y.(type) {
		case *compound: return &globpat{elements{append(tx.elems, ty.elems...)}}
		case *globpat: return &globpat{elements{append(tx.elems, ty.elems...)}}
		default: return &globpat{elements{append(tx.elems, y)}}
		}
	case *compound:
		switch ty := y.(type) {
		case *path: return &path{elements{append([]Value{prefix(ctx, tx, ty.elems[0])}, ty.elems[1:]...)}}
		case *compound: return &compound{elements{append(tx.elems, ty.elems...)}}
		case *globpat: return &globpat{elements{append(tx.elems, ty.elems...)}}
		case *qualword: // CRITICAL FIX: Compound + qualword fusion!
			if len(ty.elems) == 0 { return tx }
			return &qualword{elements{append([]Value{prefix(ctx, tx, ty.elems[0])}, ty.elems[1:]...)}}
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
	case *qualword: // CRITICAL FIX: Left-hand qualword fusion!
		if len(tx.elems) == 0 { return y }
		switch ty := y.(type) {
		case *qualword: // qualword + qualword
			if len(ty.elems) == 0 { return tx }
			return &qualword{elements{append(append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], ty.elems[0])), ty.elems[1:]...)}}
		default: // qualword + scalar
			return &qualword{elements{append(dup(tx.elems[:len(tx.elems)-1]), prefix(ctx, tx.elems[len(tx.elems)-1], y))}}
		}
	}

	switch ty := y.(type) {
	case *pair:
		switch ty.key.(type) {
		case *valbase, *null, *none, nil: return &pair{x, ty.val}
		default: return &pair{prefix(ctx, x, ty.key), ty.val}
		}
	case *globpat:
		switch x.(type) {
		case *valbase, *null, *none, nil: return &globpat{elements{dup(ty.elems)}}
		default: return &globpat{elements{append([]Value{x}, ty.elems...)}}
		}
	case *qualword: // CRITICAL FIX: Right-hand qualword fusion!
		if len(ty.elems) == 0 { return x }
		return &qualword{elements{append([]Value{prefix(ctx, x, ty.elems[0])}, ty.elems[1:]...)}}
	case *compound:
		switch ty.elems[0].(type) {
		case *pair: debug(ctx, "%v", ty.elems, trace{})
		case *valbase, *null, *none, nil: return &compound{elements{dup(ty.elems)}}
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

type valbase struct{ pos Pos }
func (_ *valbase) kind() Kind { return KindUnclassified }
func (_ *valbase) String() (_ string) { return }
func (p *valbase) Pos() Pos { return p.pos }

type loc struct{ Value ; pos Pos }
func (p *loc) Pos() Pos { return p.pos }

// xloc is a synthetic wrapper for values generated from external runtime 
// files (like grep) that do not exist in the compiler's FileSet.
type xloc struct { Value ; pos Position }
func (p *xloc) Pos() Pos { return 0 } 
func (p *xloc) Position() Position { return p.pos }

func _loc(v Value, p Pos) Value {
	if t := v.Pos(); t == p {
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

type null struct{ valbase }
func (_ *null) kind() Kind { return KindNull }
func (_ *null) String() string { return "{}" }

type none struct{ valbase }
func (_ *none) kind() Kind { return KindNone }
func (p *none) String() (_ string) { return }

type argumented_ctx struct{ Context ; val Value; args []Value }
func (ac *argumented_ctx) cast(t reflect.Type) Context { return icast(ac,t) }
func (ac *argumented_ctx) inner() Context { return ac.Context }
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
                var stemVals []Value
                for _, s := range stems {
                    stemVals = append(stemVals, s)
                }
                if v, rest := stencil(ctx, a, stemVals); len(rest) > 0 {
                    debug(pc(ctx,p), "partial stencil: %v, %v, %v, %v", a, v, rest, stems, trace{})
                } else if f := as_file(ctx, a, proj); f != nil {
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

type escaped struct{ valbase; s string }
func (_ *escaped) kind() Kind { return KindEscaped }
func (p *escaped) String() string { return "\\" + p.s }

type boolean struct{ valbase; bool }
func (_ *boolean) kind() Kind { return KindBoolean }
func (p *boolean) String() string { if p.bool { return "{=true}" } else { return "{=false}" } }

type answer struct{ boolean }
func (p *answer) String() string { if p.bool { return "{=yes}" } else { return "{=no}" } }

type option struct{ boolean }
func (p *option) String() string { if p.bool { return "{=on}" } else { return "{=off}" } }

type prediction struct{ boolean ; s string }

func toBool(v Value, t bool) Value {
	if l, ok := v.(*loc); ok {
		return &loc{toBool(l.Value, t), l.pos}
	}
	return _boolean(v.Pos(), t)
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

func makePrediction(pos Pos, val bool, s string) *prediction {
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
func (p *url) Pos() Pos { return p.Scheme.Pos() }
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

// Upgraded qualword: holds a dot-separated sequence of Values
type qualword struct{ elements }
func (_ *qualword) kind() Kind { return KindWord } // Behaves like a word/compound
func (p *qualword) String() string {
	var b strings.Builder
	for i, elem := range p.elems {
		if i > 0 { b.WriteByte('.') }
		if elem == nil {
			b.WriteString("{}")
		} else if e, ok := elem.(*list); ok {
			switch len(e.elems) {
			case 0: continue
			case 1: b.WriteString(elem.String())
			default:
				b.WriteString("⌜")
				b.WriteString(elem.String())
				b.WriteString("⌟")
			}
		} else {
			b.WriteString(elem.String())
		}
	}
	return b.String()
}

type slicer interface{ slice(...int) []Value }
type elements struct{ elems []Value }
func (p *elements) len() int { return len(p.elems) }
func (p *elements) list() *list { return &list{*p} }
func (p *elements) path() *path { return &path{*p} }
func (p *elements) compound() *compound { return &compound{*p} }
func (p *elements) globpat() *globpat { return &globpat{*p} }
func (p *elements) append(v ...Value) { p.elems = append(p.elems, v...) }
func (p *elements) Pos() (_ Pos) {
	for _, e := range p.elems {
		if e != nil { if t := e.Pos(); t.IsValid() { return t } }
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
func (p *elements) slice(i ...int) (_ []Value) {
	switch len(i) {
	case 0: return p.elems[:]
	case 1: return p.elems[:i[0]]
	case 2: return p.elems[i[0]:i[1]]
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
func (p *elements) true(ctx Context) bool { // (or elems...)
	for _, elem := range p.elems { if __true(ctx, elem) { return true } }
	return false
}

type conjunction struct{ *list ; sep Value }
func (p conjunction) kind() Kind { return KindConjunction }
func (p conjunction) String() (s string) {
    if p.sep != nil { s = p.sep.String() }
    return "{{"+p.list.String()+"}"+s+"}"
}

func redis(v Value) Value { v, _ = _redis(v); return v }
func _redis(v Value) (res Value, dis bool) {
	if v == nil { return nil, false }
	switch t := v.(type) {
	case *closure, *delegate:
		return &disjunction{valbase{v.Pos()},v}, true
	case *loc:
		res, dis = _redis(t.Value)
		res = &loc{res, t.pos}
		return
	case *xloc:
		res, dis = _redis(t.Value)
		res = &xloc{res, t.pos}
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
	case *arrow:
		var o, d1 = _redis(t.o)
		var s, d2 = _redis(t.s)
		res, dis = &arrow{t.valbase, t.t, o, s}, d1 || d2
		return
	case *percpat:
		var p, d1 = _redis(t.Prefix)
		var s, d2 = _redis(t.Suffix)
		res, dis = &percpat{t.valbase, p, s}, d1 || d2
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
	case *globpat:
		res = &globpat{elements{_redis_elems(t.elems, &dis)}}
		return
	case *strcomp:
		res = &strcomp{elements{_redis_elems(t.elems, &dis)}}
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

func _disjunction(v Value) (res Value) {
    switch v.(type) {
    case *integer, *binary, *octal, *decimal, *hexadecimal, *float, *disjunction,
        *punct, *word, *raw, *strlit, *datetime, *file, *Date, *Time, *project, self:
        return v
    }
    return &disjunction{valbase{v.Pos()},v}
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
		if false && sf("%v",op) == "&(.test.xx)" {
			debug(pc(ctx,op), "%v %d", op, c.i)
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
		case *xloc:
			for _, v := range com(ctx, nil, []Value{t.Value}) {
				res = append(res, com(ctx, append(a, &xloc{v,t.pos}), tail)...)
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

func com_qualword(ctx *comctx, a, elems []Value) (res []Value) {
	for i, elem := range elems {
		switch tail := elems[i+1:]; t := elem.(type) {
		case *disjunction:
			for _, e := range merge(expand(ctx, t.val)) {
				if isTrivial(e) { continue }
				for _, v := range com_qualword(ctx, append(dup(a), e), tail) {
					res = append(res, redis(v))
				}
			}
			return
		case *loc:
			for _, v := range com_qualword(ctx, nil, []Value{t.Value}) {
				res = append(res, com_qualword(ctx, append(dup(a), &loc{v, t.pos}), tail)...)
			}
			return
		case *xloc:
			for _, v := range com_qualword(ctx, nil, []Value{t.Value}) {
				res = append(res, com_qualword(ctx, append(dup(a), &xloc{v, t.pos}), tail)...)
			}
			return
		case *compound:
			// CRITICAL FIX: Evaluate the compound to trigger variants or annihilation!
			for _, v := range com(ctx, nil, t.elems) {
				res = append(res, com_qualword(ctx, append(dup(a), v), tail)...)
			}
			return
		default:
			a = append(a, expand(ctx, elem))
		}
	}
	switch len(a) {
	case 0 : return
	case 1 : return a
	default: return []Value{&qualword{elements{a}}}
	}
}

type compound struct{ elements }
func (_ *compound) kind() Kind { return KindCompound }
func (p *compound) String() (s string) {
	var b strings.Builder
	for _, elem := range p.elems {
		if elem == nil {
			b.WriteString("{}")
		} else if e, ok := elem.(*list); ok {
			switch len(e.elems) {
			case 0: continue
			case 1: b.WriteString(elem.String())
			default:
				b.WriteString("⌜")
				b.WriteString(elem.String())
				b.WriteString("⌟")
			}
		} else {
			b.WriteString(elem.String())
		}
	}
	return b.String()
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
            if f := project.file(ctx, t.s); f != nil {
                targets[i] = &barefile{ target, f }
                f.pos = target.Pos()
            }
        case *compound, *path:
            if patterned(ctx,t) /* || t.expandable(ctx) */ /* || refdef(ctx, t, DefArg) */ {//, expandDef2
                break
            } else if f := project.file(ctx, __string(ctx, t)); f != nil {
                targets[i] = &barefile{ target, f }
                f.pos = target.Pos()
            }
        case *argumented:
            t.Value = barefilize(ctx, t.Value)[0]
            t.args  = barefilize(ctx, t.args...)
        }
    }
    return targets
}
func exp_barefilize(ctx Context, targets ...Value) (res []Value) {
    var maps []matched_filemap
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

func _pathStr(ctx Context, str string) *path {
    return makePath(splitPathStr(ctx, str)...)
}

func _punct(ctx Context, tok token) *punct {
    return &punct{valbase{_pos(ctx)}, tok}
}

func to_file(v Value) (x *file, y bool) {
    if x, y = v.(*file); !y {
        switch t := v.(type) {
		case     *loc: x, y = to_file(t.Value)
		case    *xloc: x, y = to_file(t.Value)
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

func try_fullfile(ctx Context, f *file) Value {
    // if !truly(ctx, is_compound{}) && truly(ctx, is_exec{})
    if truly(ctx, wants_fullfile{}) {
        return fullfile{f}
    }
    return f
}

type fullname struct{ Value }
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
        debug(ctx, "file `%s` has no fullname", p, trace{})
    }

    var e error

    if p.info, e = os.Stat(fn); e != nil {
        if truly(ctx, must_file_stamp{p}) {
            ctx = pc(pc(ctx, strings.TrimSuffix(fn, ".x")), fn+".log")
            if _, y := e.(*fs.PathError); y {
                debug(ctx, "no such file: %s", p.name, trace{})
            } else {
                debug(ctx, "%v", e, trace{})
            }
        }
        return
    } else if p.info == nil {
        if truly(ctx, must_file_stamp{p}) {
            ctx = pc(pc(ctx, strings.TrimSuffix(fn, ".x")), fn+".log")
            debug(ctx, "no such file: %s", p.name, trace{})
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
            debug(ctx, "%v: %v", trimPromptString(x.Path), x.Err, trace{})
        }
        return
    } else {
        debug(ctx, "file.stat: %v", e, trace{})
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
func (p flag) Pos() (pos Pos) {
    if p.Value != nil { pos = p.Value.Pos() /* - 1 */ }
    return
}
func (p flag) String() (s string) {
    if s = "-"; p.Value != nil { s += p.Value.String() }
    return
}
func (p flag) opt(ctx Context, name string) (res string, match bool) {
	if val := p.Value; isTrivial(val) {
		if false { debug(ctx, "flag name is trivial", trace{}) }
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
            panic(err) // debug(ctx, "%v", err, trace{})
            return
        }

        var esc string
        switch s[i] {
        case '"':  esc = `\"`
        case '\r': esc = `\r`
        case '\n': esc = `\n`
        }
        if _, err = buf.WriteString(esc); err != nil {
            panic(err) // debug(ctx, "%v", err, trace{})
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
func (p *list) Pos() (pos Pos) {
    if 0 < len(p.elems) { pos = p.elems[0].Pos() }
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
func (p *group) Pos() Pos { return p.valbase.Pos() }
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
                    debug(ctx, "unsupported name type: %T %v", name, name, trace{})
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
func (p *pair) Pos() (pos Pos) {
    if p.key != nil { return p.key.Pos() }
    if p.val != nil { return p.val.Pos() }
    return
}
func (p *pair) String() (s string) {
    if k := p.key; k != nil { s  = k.String() }; s += "="
    if v := p.val; v != nil { s += v.String() }
    return
}

type skipped struct{ Value }
func (p skipped) kind() Kind { return p.Value.kind()|KindSkipped }

type _not_evoker struct{}

const (
    not_evoker int = 1<<iota
)

type expand_closure_delegate struct{ Context ; state int }
func (c *expand_closure_delegate) do(ctx Context, op any) (_ any) {
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
func (p *delegate) resolve(ctx Context) (res []Value) {
	for _, x := range merge(expand(ctx, p.x)) {
		if x != nil && x.kind()&KindBuiltin == 0 {
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

type closure struct{ delegate } // polymorphic
func (p *closure) kind() Kind { return p.valbase.kind()|KindClosure }
func (p *closure) String() (s string) { return p.src("&") }
func (p *closure) resolve(ctx Context) (res []Value) {
	for _, x := range merge(expand(ctx, p.x)) {
		if x != nil && x.kind()&KindBuiltin == 0 {
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
func (p *arrow) String() string { return p.o.String()+p.t.String()+p.s.String() }

// percpat represents percent pattern expressions (e.g. '%.o')
type percpat struct{
    valbase
    Prefix Value
    Suffix Value
}
func (p *percpat) String() (s string) {
    if !isNull(p.Prefix) { s += p.Prefix.String() }
    s += `%`
    if !isNull(p.Suffix) { s += p.Suffix.String() }
    return
}

func indexRootTailPunct(v ...Value) int {
	for i, v := range v {
		p, isPunct := unloc(v).(*punct)
		if isPunct && (p.token == PROOT || p.token == PTAIL) {
			return i
		}
	}
	return -1
}

func isMultiWildcard(v Value) bool {
	v = unloc(v) // Safely unwrap location nodes

	// Case 1: Direct meta token (e.g., sliced elements)
	if m, ok := v.(*globmeta); ok && (m.token == DAST || m.token == ASTQ) {
		return true
	}
	
	// Case 2: Wrapped in a glob pattern
	if gp, ok := v.(*globpat); ok && len(gp.elems) > 0 {
		if m, ok := unloc(gp.elems[0]).(*globmeta); ok && (m.token == DAST || m.token == ASTQ) {
			return true
		}
	}
	
	return false
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

type globbrace struct{ globpat } // globpat embraced in '{' and '}'
func (p *globbrace) String() string { return "{=glob "+p.globpat.String()+"}" }

// glob wildcards: ? * *? **
type globmeta struct{ valbase ; token }
func (p *globmeta) String() string { return p.token.String() }

// `[a-b]`, `[abc]`, `[a$(var)c]`, `[a$(spaces)c]`
type globrange struct{ Value }
func (p *globrange) String() string { return "["+p.Value.String()+"]" }

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

func values(args ...any) (elems []Value) {
	for _, a := range args {
		if x, y := a.(Value); y {
			elems = append(elems, x)
		} else if v := reflect.ValueOf(a); v.Kind() == reflect.Slice {
			for n := 0; n < v.Len(); n++ {
				elems = append(elems, values(v.Index(n).Interface())...)
			}
		} else {
			// debug(ctx, "'%v' is not value type (%T)", a, a, trace{})
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
                elems = append(elems, _null(d.pos))
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
        case *xloc: for _, v := range dmerge(disjunction, x.Value) { elems = append(elems, &xloc{v,x.pos}) }
        case *list: elems = append(elems, dmerge(disjunction, x.elems...)...)
        default:
            if disjunction { if x, y := x.(*compound); y {
                var saved = len(elems)
                for i, e := range x.elems {
                    if t := dmerge(disjunction, e); len(t) > 1 {
                        for _, e := range t {
                            t := append(append(x.elems[:i],e), dmerge(disjunction, x.elems[i+1:]...)...)
                            if false { fmt.Printf("%v: %v | %v\n", e.Pos(), a, t) }
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
            debug(ctx, "no path name for `%s`", val, trace{})
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
            debug(ctx, "no path name for `%s`", val, trace{})
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
                debug(ctx, "partial match: %v", rest)
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

func __lsa(s ...string) (a []any) { for _, s := range s { a = append(a, strings.ToLower(s)) }; return }
func __sa(s ...string) (a []any) { for _, s := range s { a = append(a, s) }; return }
func __t(a ...bool) (_ bool) { for _, a := range a { if a { return true } }; return }
func _if[T any](cond bool, t, f T) T { if cond { return t } else { return f } }
func _if_cmp[T any](c Context, r cmpres, t, f T) T { if cmp(c, t, f) == r { return t } else { return f } }
func _unless[T any](cond bool, t, f T) T { if !cond { return t } else { return f } }
func _truly[T any](c Context, a any, t T, f T) T { if truly(c,a) { return t } else { return f } }
func _falsely[T any](c Context, a any, t T, f T) T { if !truly(c,a) { return t } else { return f } }

type __stringing struct{}
type __string_ctx struct{ Context }
func (c *__string_ctx) inner() Context { return c.Context }
func (c *__string_ctx) do(ctx Context, op any) any {
	switch t := op.(type) {
	case __stringing: return true
	case __string_ctx: switch t.Context { case c, c.Context, nil: return c }
	case ex_closure, ex_def_1, ex_def_2, ex_def_3: return true
	}
	return c.Context.do(ctx, op)
}
func __string(ctx Context, v any) (res string) {
	var val Value
	if checkpoints { defer check_string(ctx, v)(&val, &res) }
	if !truly(ctx, __stringing{}) { ctx = &__string_ctx{ctx} }
	switch t := v.(type) {
	case string: return t
	case rune: return string(t)
	case token: return t.String()
	case *binary, *octal, *decimal, *hexadecimal, *float, *datetime, *Date, *Time, *globmeta, *punct:
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
	case *filestub: return t.name
	case *project: return t.name
	case self: return t.name
	case fullfile: return t.fullname()
	case negative: return "!"+__string(ctx, t.Value)
	case flag: return "-"+__string(ctx, t.Value)
	case *disjunction: return __string(ctx, t.val)
    case *undetermined: return __string(ctx, t.value)
	case *xloc: return __string(ctx, t.Value)
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
	case *qualword:
		for i, e := range t.elems { if i > 0 { res += "." }
			res += __string(ctx, e)
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
		// Check if v differs from val without triggering __string recursion
		if val = expand(ctx, t.(Value)); val != v {
			// Fix: avoid infinite recursion via equal -> cmp -> __string -> expand -> equal
			// If expand returns a different object (v != val), we use it. 
			// We DO NOT check cmp(v, val) because cmp might fall back to __string(val), causing a loop.
			if reflect.TypeOf(val) != reflect.TypeOf(v) {
				res = __string(ctx, val)
			} else if cmp(ctx, val, v) != cmpEqual {
				res = __string(ctx, val)
			}
		}
	case fullname:
		if v := t.Value; v != nil {
			if x, y := v.(*file); y {
				if x == nil {
					debug(ctx, "nil file", trace{})
				} else if x.filestub == nil {
					debug(ctx, "nil file stub", trace{})
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
	case *recipe:
		for _, elem := range t.elems { res += __string(ctx, elem) }
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
	case *compound: return t.elements.true(ctx)
	case *qualword: return t.elements.true(ctx)
	case *globpat: return t.elements.true(ctx)
	case *strcomp: return t.elements.true(ctx)
	case *strlit: return t.s != ""
	case *raw: return t.s != ""
	case *escaped: return t.s != ""
	case *builtin: return t.t != nil
	case self: return t.name != ""
	case *project: return t.name != ""
	case *file: return t.filestub.name != ""
	case *xloc: return __true(ctx, t.Value)
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
	case *word:
		switch t.s {
		case "false", "False", "FALSE", "no", "No", "NO": return false
		case "true", "True", "TRUE", "yes", "Yes", "YES": return true
		}
		return t.s != ""
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					debug(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t), trace{})
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
	case *xloc: return __int(ctx, t.Value)
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
			debug(ctx, "%v", e, trace{})
		}
	case *word:
		if i, e := strconv.ParseInt(t.s, 10, 64); e == nil {
			return i
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *strlit:
		if i, e := strconv.ParseInt(t.s, 10, 64); e == nil {
			return i
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *strcomp:
		if i, e := strconv.ParseInt(__string(ctx, t), 10, 64); e == nil {
			return i
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *strval:
		if i, e := strconv.ParseInt(__string(ctx, t), 10, 64); e == nil {
			return i
		} else {
			debug(ctx, "%v", e, trace{})
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
	case *qualword:
		if t.len() > 0 { 
			// Instantly returns the Major version (e.g., 1 from 1.10.1)
			return __int(ctx, t.elems[0]) 
		}
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					debug(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t), trace{})
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
	case *exec_result: return float64(t.Status)
	case *binary: return float64(t.int64)
	case *octal: return float64(t.int64)
	case *decimal: return float64(t.int64)
	case *datetime: return float64(t.t.Unix())
	case *Date: return float64(t.t.Unix())
	case *Time: return float64(t.t.Unix())
	case *pair: return __float(ctx, t.val)
	case *auto: return __float(ctx, t.def(ctx))
	case *def: return __float(ctx, t.value)
	case *loc: return __float(ctx, t.Value)
	case *xloc: return __float(ctx, t.Value)
	case *float: return t.float64
	case *list: if t.len() > 0 { return __float(ctx, t.elems[0]) }
	case *plain: if t.len() > 0 { return __float(ctx, t.elems[0]) }
	case *plainline: if t.len() > 0 { return __float(ctx, t.elems[0]) }
	case negative: if t.Value != nil && !__true(ctx, t.Value) { return 1. }
	case flag: if t.Value != nil { return -__float(ctx, t.Value) }
	case *qualword:
		if n := t.len(); n == 1 {
			return __float(ctx, t.elems[0])
		} else if n > 1 {
			// Combine ONLY the Major and Minor segments (e.g., "1" + "." + "10")
			s := __string(ctx, t.elems[0]) + "." + __string(ctx, t.elems[1])
			if f, e := strconv.ParseFloat(s, 64); e == nil {
				return f
			} else {
				debug(ctx, "%v", e, trace{})
			}
		}
	case *raw:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *word:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *strlit:
		if f, e := strconv.ParseFloat(t.s, 64); e == nil {
			return f
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *strcomp:
		if f, e := strconv.ParseFloat(__string(ctx, t), 64); e == nil {
			return f
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *strval:
		if f, e := strconv.ParseFloat(__string(ctx, t), 64); e == nil {
			return f
		} else {
			debug(ctx, "%v", e, trace{})
		}
	case *arrow, *closure, *delegate:
		if v = expand(ctx, t); !equal(ctx, v, t) {
			if checkpoints {
				if v.String() == t.String() {
					debug(pc(ctx,t), "%v %v (%v)", t, v, equal(ctx, v, t), trace{})
				}
			}
			return __float(ctx, v)
		}
	}
	return
}

// Bubble: Sent by an unbound *auto to request deferral
type act_defer_macro struct{}

// Bubble: Sent by a *builtin to configure argument evaluation
type act_macro_scope struct {
	force_collapse bool
	skip_expansion map[int]bool // arg_index -> skip. -1 means skip ALL args.
}

// The intercepting context wrapping the delegate/closure expansion
type collapse_ctx struct {
	Context
	good_to_collapse bool
	force_collapse   bool
	skip_expansion   map[int]bool
	current_arg      int 
}

func (c *collapse_ctx) inner() Context { return c.Context }
func (c *collapse_ctx) cast(t reflect.Type) Context { return icast(c, t) }
func (c *collapse_ctx) do(ctx Context, op any) any {
	switch t := op.(type) {
	case act_defer_macro:
		if !c.force_collapse {
			c.good_to_collapse = false
		}
		return true

	case act_macro_scope:
		if t.force_collapse { c.force_collapse = true }
		if t.skip_expansion != nil { c.skip_expansion = t.skip_expansion }
		return true
	}
	return c.Context.do(ctx, op)
}

func expand(ctx Context, v Value) (res Value) {
    switch t := v.(type) {
	case *auto:
		val := t.def(ctx)
		if val == nil || isTrivial(val) {
			// Unbound! Bubble the deferral state.
			if truly(ctx, keep_autos{}) {
				do(ctx, act_defer_macro{})
			}
			return t // Return AST node because it's unbound
		}
		return expand(ctx, val) // CRITICAL FIX: Expand bound values!

	case *builtin:
		// Polymorphic Configuration Bubbling
		var scope act_macro_scope
		switch t.name {
		case "auto":
			scope.force_collapse = true
			scope.skip_expansion = map[int]bool{-1: true} // Skip ALL arguments. Let __auto handle them!
		case "foreach", "grep":
			scope.force_collapse = true
			scope.skip_expansion = map[int]bool{1: true} // Skip expanding the locally-bound body argument
		case "addprefix", "addsuffix", "join", "conjunct":
			scope.force_collapse = true // Eagerly evaluate so unbound autos get wrapped by redis()!
		}

		do(ctx, scope) 
		return t

	case *delegate:
		var dx = &delegate_x{closure_delegate_x{ctx, 0}}
		var c = collapse_ctx{ Context: dx, good_to_collapse: true }

		var rx = t.resolve(&c)
		for _, xv := range rx { expand(&c, xv) } // Polymorphic Lookahead

		var o []Value
		for _, opt := range t.o {
			c.current_arg = -1
			o = append(o, expand(&c, opt))
		}

		var a []Value
		for i, arg := range t.a {
			c.current_arg = i
			if c.skip_expansion != nil && (c.skip_expansion[i] || c.skip_expansion[-1]) {
				a = append(a, arg)
			} else {
				a = append(a, expand(&c, arg))
			}
		}

		var vals []Value

		if c.force_collapse || (dx.closure_delegate == 0 && c.good_to_collapse) {
			// Fast path: fully resolved, try to execute!
			for _, xv := range rx {
				cc := expand_closure_delegate{ctx, 0}
				if rv := evoke(&cc, xv, o, a); cc.state == not_evoker {
					// Execution failed.
					// CRITICAL FIX: Only rebuild if the resolved target actually exists!
					if !isNull(xv) && truly(ctx, keep_autos{}) {
						dv := &delegate{t.valbase, t.l, xv, o, a}
						vals = append(vals, dv)
						do(ctx, dv)
					} else {
						vals = append(vals, _null(t.pos)) // Target is dead, return null.
					}
				} else if rv == nil {
					// Target executed successfully but produced NO output.
					vals = append(vals, _null(t.pos))
				} else {
					vals = append(vals, _loc(rv, t.pos))
				}
			}
		} else {
			// Deferred path (due to unbound arguments)
			if !c.good_to_collapse { do(ctx, act_defer_macro{}) }

			// CRITICAL FIX: Protect deferred path from ghost delegates
			for _, xv := range rx {
				if isNull(xv) {
					vals = append(vals, _null(t.pos))
				} else {
					dv := &delegate{t.valbase, t.l, xv, o, a}
					vals = append(vals, dv)
					do(ctx, dv)
				}
			}
		}
		
		if res = ease(ctx, vals); checkpoints { check(ctx, res, v, rx...) }
		return

	case *closure:
		var cx = &closure_x{closure_delegate_x{ctx, 0}}
		var c = collapse_ctx{ Context: cx, good_to_collapse: true }

		var rx = t.resolve(&c)
		for _, xv := range rx { expand(&c, xv) }

		var o []Value
		for _, opt := range t.o {
			c.current_arg = -1
			o = append(o, expand(&c, opt))
		}

		var a []Value
		for i, arg := range t.a {
			c.current_arg = i
			if c.skip_expansion != nil && (c.skip_expansion[i] || c.skip_expansion[-1]) {
				a = append(a, arg)
			} else {
				a = append(a, expand(&c, arg))
			}
		}

		var vals []Value

		if c.force_collapse || (cx.closure_delegate == 0 && c.good_to_collapse && truly(ctx, ex_closure{})) {
			for _, xv := range rx {
				var cc = expand_closure_delegate{ctx, 0}
				if rv := evoke(&cc, xv, o, a); cc.state == not_evoker || rv == nil {
					// CRITICAL FIX: Only rebuild if the resolved target actually exists!
					if !isNull(xv) && truly(ctx, keep_autos{}) {
						cv := &closure{delegate{t.valbase, t.l, xv, o, a}}
						vals = append(vals, cv)
						do(ctx, cv)
					} else {
						vals = append(vals, _null(t.pos)) // Target is dead, return null.
					}
				} else {
					vals = append(vals, _loc(rv, t.pos))
				}
			}
		} else {
			if !c.good_to_collapse { do(ctx, act_defer_macro{}) }

			// CRITICAL FIX: Protect deferred path from ghost closures
			for _, xv := range rx {
				if isNull(xv) {
					vals = append(vals, _null(t.pos))
				} else {
					cv := &closure{delegate{t.valbase, t.l, xv, o, a}}
					vals = append(vals, cv)
					do(ctx, cv)
				}
			}
		}
		
		if res = ease(ctx, vals); checkpoints { check(ctx, res, v, rx...) }
		return

    case *arrow:
        var vals []Value
        var p0 = t.Pos()
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
                        debug(pc(ctx,t), "%v %v %v", o, t.t, ss, trace{})
                    }
                case truly(ctx, delegate_t{}):
                    switch t.t {
                    case SELECT_PROG1, SELECT_PROG2:
                        if v := project_entry(ctx, o); v != nil { o = v }
                    case SELECT_PROP:
                        if v := project_resolve(ctx, str); v != nil { o = v }
                    default:
                        debug(pc(ctx,t), "%v %v %v", o, t.t, ss, trace{})
                    }
                default:
                    debug(pc(ctx,t), "%v %v", o, ss, trace{})
                }
            }
			for _, s := range ss {
				var p = s.Pos()
				
				// Safely extract the raw property name and its optional modifier flag
				var prop_str, is_opt = _sel_prop(ctx, s)
				
				switch res := sel(ctx, o, prop_str).(type) {
				case nil:
					// CRITICAL FIX: If the property is missing but marked as optional, 
					// preserve the AST node and safely bubble the deferral!
					if is_opt || optional(s) || optional(o) {
						if truly(ctx, keep_autos{}) {
							do(ctx, act_defer_macro{}) // Signal parent to defer!
						}
						
						// Unwrap 'o' ONLY for AST reconstruction to prevent leaking `*def` 
						// assignment metadata into the frozen arrow syntax.
						var actual_o = o
						for {
							if l, ok := actual_o.(*loc); ok { actual_o = l.Value; continue }
							if d, ok := actual_o.(*def); ok {
								if d.value != nil { actual_o = d.value } else { actual_o = _null(d.Pos()) }
								continue
							}
							break
						}
						
						// Return the perfectly preserved arrow (e.g. {=self foo} -> xxxx?)
						vals = append(vals, &arrow{t.valbase, t.t, actual_o, s})
					} else {
						vals = append(vals, _loc(_null(p), p0))
					}
				default:
					vals = append(vals, _loc(_loc(res, p), p0)) // NOTE: not def.value
				}
			}
        }
        if res = ease(ctx, vals); checkpoints && false { check(ctx, res, v) }
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
        if res = ease(pc(ctx,v), vals); checkpoints && false { check(ctx, res, v) }
        return
	case *qualword:
        // Expand elements, but we do NOT use com() because dots are strict boundaries,
        // not whitespace-separated compound elements.
        var vals = com_qualword(&comctx{ctx, 0}, nil, t.elems)
        if res = ease(ctx, vals); checkpoints && false { check(ctx, res, v) }
        return
    case *pair:
        var vals []Value
        var ks = merge(expand(ctx, t.key))
        var vs = merge(expand(ctx, t.val))
		for _, k := range ks { for _, v := range vs { vals = append(vals, &pair{k, v}) }}
        if res = ease(ctx, vals); checkpoints && false { check(ctx, res, v) }
        return
    case flag:
        if t.Value == nil {
            return t
        } else {
            var vals []Value
            for _, v := range merge(expand(ctx, t.Value)) { vals = append(vals, flag{v}) }
            if res = ease(ctx, vals); checkpoints && false { check(ctx, res, v) }
            return
        }
    case conjunction:
        c := conjunction{&list{elements{expands(ctx, t.list.elems...)}}, nil}
        if c.sep != nil { c.sep = expand(ctx, t.sep) }
        return c
    case *disjunction:
        var vals []Value
        for _, v := range merge(expand(ctx, t.val)) { vals = append(vals, redis(v)) }
        if res = ease(ctx, vals); checkpoints && false { check(ctx, res, v) }
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
	case matched_rule: return matched_rule{expand(ctx, t.rule).(*rule), expand(ctx, t.value)}
    case fullfile: if truly(ctx,is_compound{}) { return t.file  } else { return t }
    case fullname:
		if truly(ctx,is_compound{}) { return t.Value } else {
			v := expand(ctx,t.Value)
			// Lazily resolve the file object at execution time!
			if !isTrivial(v) {
				if f := as_file(ctx, v); f != nil {
					return fullname{f}
				}
			}
			return fullname{v}
		}
    case negative: return negative{expand(ctx, t.Value)}
    case *loc: return &loc{expand(ctx, t.Value), t.pos}
    case *xloc: return &xloc{expand(ctx, t.Value), t.pos}
    case *list: return &list{elements{expands(ctx, t.elems...)}}
    case *path: return &path{elements{expandp(ctx, t.elems...)}}
    case *quote: return &quote{list{elements{expands(ctx, t.elems...)}}}
    case *group: return &group{t.valbase,elements{expands(ctx, t.elems...)}}
    case *recipe: return &recipe{strcomp{elements{expands(ctx, t.elems...)}}}
    case *percpat: return &percpat{t.valbase, expand(ctx,t.Prefix), expand(ctx,t.Suffix)}
    case *globpat: return &globpat{elements{expands(ctx, t.elems...)}}
    case *globrange: return &globrange{expand(ctx, t.Value)}
    case *argumented: return &argumented{expand(ctx, t.Value), expands(ctx, t.args...)}
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
    case *valbase, *answer, *boolean, *binary, *def, *none, *null, *punct, *word, *globmeta, *octal, *decimal, *hexadecimal, *escaped, *raw, *regexpat, *defcaps, *project, *file, self, undef, nil:
        if false && v == nil { debug(pc(ctx,v), "%v", v) } //, *modification
        return v
    default:
        if checkpoints { debug(pc(ctx,v), "%v", ts(v,ctx), callstack{stop:"smart.runcase"}, trace{}) }
        return v
    }
}

func expands(ctx Context, v ...Value) (res []Value) {
    for _, v := range v { res = append(res, expand(ctx, v)) }
    return
}

func expandp(ctx Context, v ...Value) []Value {
	return pathElems(expands(ctx, v...)...)
}

func pathElems(v ...Value) (res []Value) {
	for _, v := range v {
		switch t := v.(type) {
		case *loc:
			for _, v := range pathElems(t.Value) { res = append(res, &loc{v, t.pos}) }
		case *xloc:
			for _, v := range pathElems(t.Value) { res = append(res, &xloc{v, t.pos}) }
		case *path:
			res = append(res, pathElems(t.elems...)...)
		default:
			res = append(res, t)
		}
	}
	return
}

func can_sel(ctx Context, v Value) (res bool) {
    switch t := v.(type) {
    case *loc: return can_sel(ctx, t.Value)
    case *xloc: return can_sel(ctx, t.Value)
    case *def, *project, self, *use, *uselist: return true
    case *list: for _, v := range t.elems { if can_sel(ctx, v) { return true } }
    default: if false { debug(pc(ctx,v), "%v", ts(v)) }
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
	case *xloc: return _sel_prop(ctx, t.Value)
	default: return __string(ctx, t), false
	}
	return
}
func sel_prop(c Context, v Value) (s string) { s, _ = _sel_prop(c, v); return }
func sel(ctx Context, v any, s string) (res Value) {
	var g *globpat
    switch t := v.(type) {
    case *loc: return sel(ctx, t.Value, s)
    case *xloc: return sel(ctx, t.Value, s)
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
        debug(pc(ctx,v), "cannot sel: %v %v", ts(t), s, trace{})
    }
	if g != nil {}
    return
}

func evoke(ctx Context, x Value, o, a []Value) (res Value) {
	switch t := x.(type) {
	case *project, self: return x
	case *loc: return evoke(ctx, t.Value, o, a)
	case *xloc: return evoke(ctx, t.Value, o, a)
	case matched_rule: return evoke(ctx, t.rule, o, a)

	case *stemmed_rule:
		r := *t.rule
		r.target = t.target
		return evoke(&stemmed_ctx{ctx, t}, &r, o, a)

    case *auto:
		if d := auto_find(ctx, t.name); d != nil { return evoke(ctx, d, o, a) }
		return _null(x.Pos())

	case *def:
		if truly(ctx, evoke_detect_loop{x}) {
			if truly(ctx, evoke_loop_panic{}) { panic(trace_evoke_loop_err{ctx, x}) }
			res = _null(x.Pos())
		} else {
			ctx := def_evoke{&evocation{automatic{Context:ctx, defs:make(def_map)}, x, o, a}}
			if ctx.a != nil { ctx.args(ctx, expands(ctx.Context, ctx.a...)) }
			if res = expand(ctx, t.value); t.o == defExecute && !isEmpty(res) { res = t.xexe(ctx, res) }
		}
		return

    case *rule:
		if truly(ctx, evoke_detect_loop{x}) {
			if truly(ctx, evoke_loop_panic{}) { panic(trace_evoke_loop_err{ctx, x}) }
			res = _null(x.Pos())
		} else {
			res = ease(ctx, t.execute(ctx, expands(ctx, a...)...))
		}
		return

    case *builtin:
		if truly(ctx, evoke_detect_loop{x}) {
			if truly(ctx, evoke_loop_panic{}) { panic(trace_evoke_loop_err{ctx, x}) }
			return _null(x.Pos())
		}

		ctx := &evocation{automatic{Context:ctx, defs:make(def_map)}, x, nil, a}
		_v := reflect.New(t.t)

		defer t.benchmark(ctx, time.Now(), _v)

		if f := _v.Elem().FieldByName("builtinbase"); !f.IsValid() {
			debug(ctx, "no such field: %s.builtinbase", _v.Elem().Type(), trace{})
		} else if f.CanAddr() {
			b := (*builtinbase)(unsafe.Pointer(f.Addr().Pointer()))
			b.evocation = ctx
		} else if f = _v.Elem().FieldByName("evocation"); !f.IsValid() {
			debug(ctx, "no such field: %s.evocation", _v.Elem().Type(), trace{})
		} else if f.CanSet() {
			f.Set(reflect.ValueOf(ctx))
		} else if f.CanAddr() && f.Addr().CanSet() {
			f.Addr().SetPointer(unsafe.Pointer(ctx))
		} else {
			debug(ctx, "cannot set field: %s.evocation", _v.Elem().Type(), trace{})
		}

		if o != nil { ctx.o = _opts(ctx, _v, o) }

		if x, y := _v.Interface().(builtin_x); y {
			res = ease(ctx, x.x())
		} else {
			debug(pc(ctx,x), "no method: %v", t.t.Name(), trace{})
		}

	case *closure, *delegate, *word, *raw, *strlit, *compound, *qualword, *globpat, *arrow, flag:
		do(ctx, _not_evoker{})

    default:
		if x != nil && !isTrivial(x) {
			debug(pc(ctx,x), "%s", ts(x), callstack{num:10}, trace{})
		}
    }
	return
}

// === Pattern Matching Engine Contract ===
//
// 1. Data Taxonomy & Containers
// The engine evaluates Values categorized into three fundamental types:
// - Scalars: Non-composite, non-pattern fundamental values (e.g., word, raw, file, decimal, globmeta).
// - Composites: Structural containers holding other values (e.g., compound, path, list).
//   * path: Special composite whose elements are separated by path-separators.
//   * flag: Special composite representing a virtual dash "-" and a wrapped Value.
// - Patterns: Abstract matching rules (e.g., globpat, percpat, regexpat).
//
// 2. The Universal `matched` vs. `full` Principle
// Lower-level match functions NEVER return `full`. They only return `matched` (often labeled `ok`).
// - matched == true: The pattern successfully evaluated against the text. It DOES NOT mean 
//   the value was fully consumed.
// - rem != nil: The exact unconsumed portion of the value left behind by the pattern.
// - full == true: A derived state. A match is only "full" if `matched == true AND rem == nil 
//   AND idx == len(vals)`. Only the top-level `match()` function asserts `full`.
//
// 3. Gap Absorption (The Wildcard Rule)
// When a literal pattern is adjacent to a gap wildcard (*, **, ?), partial matches are successes.
// - If a literal sub-pattern matches successfully but leaves a remainder (matched == true AND 
//   rem != nil), the engine MUST NOT reject it.
// - The literal pattern anchors to the boundary, and the wildcard's explicit job is to 
//   "absorb" the `rem` along with any intermediate segments.
//
// 4. Bidirectionality (`trail` Parameter)
// The engine executes matches inward from the boundaries to isolate wildcards.
// - Forward Match (trail = false): Evaluates Left-to-Right. The pattern anchors to the START 
//   of the value. `rem` represents the unconsumed text on the RIGHT.
// - Trailing Match (trail = true): Evaluates Right-to-Left. The pattern anchors to the END 
//   of the value. `rem` represents the unconsumed text on the LEFT.
//
// 5. Dispatcher & Function Signatures
// - Top-Level Dispatcher:
//   match(ctx Context, pat, val Value) (full bool, res, rem Value, stems []Value)
//
// - Low-Level Evaluators (shedding unnecessary returns based on scope):
//   matchScalarScalar (...) (matched bool, res, rem Value)
//   matchCompComp     (...) (matched bool, res, rem Value, idx int)
//   matchGlobComp     (...) (matched bool, res, rem Value, stems []Value, idx int, wildToken token)
//   matchGlobPath     (...) (matched bool, res, rem Value, stems []Value, idx int, wildToken token)
//   matchPathPath     (...) (matched bool, res, rem Value, stems []Value, idx int)
//
// 6. Core Execution Algorithm
// IF pat is pure literal (no wildcards):
//   - Iterate through elements (L-to-R or R-to-L based on `trail`).
//   - IF match fails -> RETURN matched=false.
//   - RETURN matched=true, res=consumed, rem=leftovers, idx=next_position.
// IF pat contains GAP_WILDCARD (** or * or ?):
//   - Split pat into PREFIX, GAP, and SUFFIX.
//   - matchedPrefix, resPrefix, remPrefix <- MATCH(PREFIX, val, trail=false) // Anchor Left
//   - matchedSuffix, resSuffix, remSuffix <- MATCH(SUFFIX, val, trail=true)  // Anchor Right
//   - IF both matched:
//       GAP absorbs remPrefix, remSuffix, and all intermediate values.
//       RETURN matched=true, res=combined_res, rem=nil.
//
// Ends.

type is_swapped struct{}
type swapped_ctx struct{ Context }
func (c swapped_ctx) inner() Context { return c.Context }
func (c swapped_ctx) cast(t reflect.Type) Context { return icast(c, t) }
func (c swapped_ctx) do(ctx Context, op any) any {
	switch op.(type) {
	case is_swapped: return true
	}
	return c.Context.do(ctx, op)
}

// globMatchCharSet checks if char matches the set pattern [range].
func globMatchCharSet(pattern string, char rune) bool {
	if len(pattern) < 3 { return false } // []]
	inner := pattern[1 : len(pattern)-1]
	negate := false
	if len(inner) > 0 && inner[0] == '^' {
		negate = true
		inner = inner[1:]
	}

	match := false
	for i := 0; i < len(inner); i++ {
		if i+2 < len(inner) && inner[i+1] == '-' {
			start, end := rune(inner[i]), rune(inner[i+2])
			if char >= start && char <= end {
				match = true
				break
			}
			i += 2
		} else {
			if rune(inner[i]) == char {
				match = true
				break
			}
		}
	}
	return match != negate
}

// staticStr returns the string representation of a value ONLY if it can be 
// determined without evaluating definitions, closures, or rules.
func staticStr(v any) (string, bool) {
	if v == nil { return "", true }
	switch t := v.(type) {
	case *valbase, *null, *none, *undef: return "", true // Map empty AST nodes to ""
	case *loc: return staticStr(t.Value)
	case *xloc: return staticStr(t.Value)
	case string: return t, true
	case *raw: return t.s, true
	case *word: return t.s, true
	case *strlit: return t.s, true
	case *project: return t.name, true
	case *globmeta: return t.token.String(), true
	case *punct: return t.token.String(), true // (PROOT, PTAIL).String() is empty
	case token: return t.String(), true
	case int: return strconv.Itoa(t), true
	case int64: return strconv.FormatInt(t, 10), true
	case float64: return strconv.FormatFloat(t, 'g', -1, 64), true
	case *decimal: return strconv.FormatInt(t.int64, 10), true
	case *float: return strconv.FormatFloat(t.float64, 'g', -1, 64), true
	case *boolean: if t.bool { return "true", true } else { return "false", true }
	case *file: if t.filestub != nil { return t.filestub.name, true } else { return "", false }
	case flag:
		if t.Value == nil || isEmpty(t.Value) { return "-", true }
		if s, ok := staticStr(t.Value); ok { return "-" + s, true }
		return "", false
	case *compound: //panic(sf("disabled staticStr(compound(%v))",t))
		var sb strings.Builder
		for _, e := range t.elems {
			if s, ok := staticStr(e); ok { sb.WriteString(s) } else { return "", false }
		}
		return sb.String(), true
	case *path: //panic(sf("disabled staticStr(path(%v))",t))
		var sb strings.Builder
		for i, e := range t.elems {
			if i > 0 { sb.WriteString(pathSep) }
			if s, ok := staticStr(e); ok { sb.WriteString(s) } else { return "", false }
		}
		return sb.String(), true
	}
	return "", false
}

// Replacement of __string for fast match. TODO: more optimization/interning
func quickStr(ctx Context, v Value) string {
	if s, ok := staticStr(v); ok { return s }
	return __string(ctx, v) // Only fallback if staticStr refuses
}

func getScalarLength(ctx Context, v Value) int {
	if t, ok := v.(flag); ok {
		return 1 + getScalarLength(ctx, t.Value)
	} else {
		return len(quickStr(ctx, v)) // TODO: optimization	
	}
}

// getScalarSubstr extracts a substring from a scalar Value.
// If end is -1, it returns the substring from start to the end of the string.
// It returns false if the value is not a supported scalar type.
func getScalarSubstr(ctx Context, v Value, start, end int) string {
	if end != -1 && start >= end { return "" }

	if t, ok := v.(flag); ok {
		// Map [start, end) to the underlying value's coordinates.
		var vStart, vEnd int
		var includeDash bool

		if start <= 0 {
			includeDash = true
			vStart = 0
		} else {
			vStart = start - 1
		}

		if end == -1 {
			vEnd = -1
		} else {
			if end <= 1 {
				if includeDash && end == 1 { return "-" }
				return ""
			}
			vEnd = end - 1
		}

		valStr := getScalarSubstr(ctx, t.Value, vStart, vEnd)
		if includeDash { return "-" + valStr }
		return valStr
	}

	var s = quickStr(ctx, v)
	if start < 0 { start = 0 }
	if end < 0 || end > len(s) { end = len(s) }
	
	if start > end { return "" }
	if start >= len(s) { return "" }

	return s[start:end]
}

func isPureWildcard(val Value) bool {
	switch e := unloc(val).(type) {
	case *globmeta:
		// Any standalone wildcard node (*, **, or an exploded PERC) is pure by definition!
		return true
	case *percpat:
		// A *percpat is only pure if it lacks both a prefix and a suffix
		suf := e.Suffix
		if pp, ok := unloc(e.Suffix).(*percpat); ok && isEmpty(pp.Prefix) { suf = pp.Suffix }
		return isEmpty(e.Prefix) && isEmpty(suf)
	}
	return false
}

// getWildcardSuffix extracts the hidden suffix from a wildcard node if it exists.
func getWildcardSuffix(v Value) Value {
	if pp, ok := unloc(v).(*percpat); ok {
		suf := pp.Suffix
		// Handle the %% edge case where the inner percpat holds the true suffix
		if inner, isPP := unloc(pp.Suffix).(*percpat); isPP && isEmpty(inner.Prefix) { 
			suf = inner.Suffix 
		}
		if suf != nil && !isEmpty(suf) { return suf }
	}
	return nil
}

// getWildcardPrefix extracts the hidden prefix from a wildcard node if it exists.
func getWildcardPrefix(v Value) Value {
	if pp, ok := unloc(v).(*percpat); ok {
		if pp.Prefix != nil && !isEmpty(pp.Prefix) { return pp.Prefix }
	}
	return nil
}

// Helper to unify *globmeta and *percpat logic
func getGlobToken(v Value) token {
	switch e := unloc(v).(type) {
	case *globmeta: return e.token
	case *percpat: return PERC
	}
	return ILLEGAL
}

type stemseg struct {
	e, v Value
}

// gapseg handles the conditional boundary logic for wildcard gaps
type gapseg struct {
	bound bool
	e     Value
	rem   []Value
}

type optraw struct {
	b bool
	p Pos
	s string
}

func concat(args ...any) (parts []Value) {
	for _, arg := range args {
		if arg != nil {
			switch t := arg.(type) {
			case []Value: parts = append(parts, t...)
			case Value: parts = append(parts, t)
			case optraw:
				if t.b {
					var v Value
					if t.s == "" { v = &valbase{t.p} } else { v = _raw(t.p, t.s) }
					parts = append(parts, v)
				}
			case stemseg:
				if t.v == nil { t.v = &valbase{t.e.Pos()} }
				parts = append(parts, t.v)
			case gapseg:
				switch len(t.rem) {
				case 0: if t.bound { parts = append(parts, &valbase{t.e.Pos()}) }
				case 1: parts = append(parts, t.rem[0])
				default: parts = append(parts, &compound{elements{t.rem}})
				}
			}
		}
	}
	return
}

func packParts(trail bool, parts []Value, args ...any) []Value {
	var vals []Value
	for _, arg := range args {
		if arg != nil {
			switch t := arg.(type) {
			case []Value: vals = append(vals, t...)
			// case *compound: vals = append(vals, t.elems...)
			case *path: vals = append(vals, t.elems...)
			case Value: vals = append(vals, t)
			}
		}
	}
	if 0 < len(vals) {
		if trail {
			parts = append(vals, parts...)
		} else {
			parts = append(parts, vals...)
		}
	}
	return parts
}

func packComp(parts []Value) Value {
	switch len(parts) { case 0: return nil; case 1: return parts[0] }
	return &compound{elements{parts}}
}

func packPath(parts []Value) Value {
	switch len(parts) { case 0: return nil; case 1: return parts[0] }
	return &path{elements{parts}}
}

func forwardCompComp(ctx Context, elems, vals []Value) (matched bool, res, rem []Value, eIdx, sIdx int) {
	currVal := vals[0] // Guaranteed by caller contract to have at least 1 element

	for eIdx < len(elems) {
		m, r1, r2 := matchScalarScalar(ctx, elems[eIdx], currVal, false)
		if !m { break }

		if r2 == nil || isEmpty(r2) {
			res = concat(res, currVal)
			sIdx++
			if sIdx < len(vals) { 
				currVal = vals[sIdx] 
			} else { 
				currVal = nil 
			}
		} else {
			res = concat(res, r1)
			currVal = r2 // Shrink the current segment
		}

		eIdx++

		if currVal == nil { break } // Ran out of value segments
	}
	
	matched = eIdx == len(elems)

	// FIX: Safely reconstruct the unconsumed remainder without out-of-bounds truncation
	if currVal != nil {
		rem = concat(currVal, vals[sIdx+1:])
	} else if sIdx < len(vals) {
		rem = vals[sIdx:]
	}

	return
}

func backwardCompComp(ctx Context, elems, vals []Value) (matched bool, res, rem []Value, eIdx, sIdx int) {
	eIdx = len(elems) - 1
	sIdx = len(vals) - 1
	currVal := vals[sIdx] // Guaranteed by caller contract to have at least 1 element

	for 0 <= eIdx {
		m, r1, r2 := matchScalarScalar(ctx, elems[eIdx], currVal, true)
		if !m { break }

		if r2 == nil || isEmpty(r2) {
			res = concat(currVal, res)
			sIdx--
			if sIdx >= 0 { 
				currVal = vals[sIdx] 
			} else { 
				currVal = nil 
			}
		} else {
			res = concat(r1, res)
			currVal = r2 // Shrink the current segment
		}

		eIdx--

		if currVal == nil { break } // Ran out of value segments
	}
	
	matched = eIdx < 0

	// FIX: Symmetrical fix to preserve backward unconsumed elements
	if currVal != nil {
		rem = concat(vals[:sIdx], currVal)
	} else if sIdx >= 0 {
		rem = vals[:sIdx+1]
	}

	return
}

func forwardGlobComp(ctx Context, elems, vals []Value) (matched bool, res, rem, stems []Value, iE, iV int, wildToken token) {
	forward := func(str string, size int) (bool, []Value, []Value, []Value, int, int, token) {
		pos := vals[iV].Pos()
		val := _raw(pos, str[:size])

		m, r, rm, s, ie, iv, wt := forwardGlobComp(ctx, elems[iE+1:],
			concat(optraw{size < len(str), pos, str[size:]}, vals[iV+1:]))

		if size == len(str) { iV += 1 }
		return m, concat(res, val, r), rm, concat(stems, val, s), iE + 1 + ie, iV + iv, wt
	}

	for iE < len(elems) { 
		switch e := unloc(elems[iE]).(type) {
		case *globrange: 
			if iV >= len(vals) { return false, res, nil, stems, iE, iV, ILLEGAL } 
			str, t := getScalarSubstr(ctx, vals[iV], 0, -1), getScalarSubstr(ctx, e.Value, 0, -1)
			r, size := utf8.DecodeRuneInString(str)
			if r == utf8.RuneError && size <= 1 || !globMatchCharSet(t, r) {
				return false, res, vals[iV:], stems, iE, iV, ILLEGAL
			}
			return forward(str, size)

		case *globmeta:
			switch tok := getGlobToken(e); tok {
			case QUE: 
				if iV >= len(vals) { return false, res, nil, stems, iE, iV, ILLEGAL } 
				str := getScalarSubstr(ctx, vals[iV], 0, -1) 
				r, size := utf8.DecodeRuneInString(str)
				if r == utf8.RuneError && size <= 1 {
					return false, res, vals[iV:], stems, iE, iV, ILLEGAL
				}
				return forward(str, size)

			case SAST, DAST, ASTQ: 
				suffix, gap := elems[iE+1:], vals[iV:]

				if tok == ASTQ {
					for k := 0; k <= len(gap); k++ {
						if k < len(gap) {
							str := getScalarSubstr(ctx, gap[k], 0, -1)
							pos := gap[k].Pos()
							
							for i := 0; i < len(str); {
								if m, r, rm, s, _, _, wt := forwardGlobComp(ctx, suffix, concat(_raw(pos, str[i:]), gap[k+1:])); m {
									stemParts := concat(gap[:k], _raw(pos, str[:i]))
									return true, concat(res, stemParts, r), rm, concat(stems, gapseg{true, elems[iE], stemParts}, s), len(elems), len(vals), wt
								}

								_, size := utf8.DecodeRuneInString(str[i:])
								i += size
							}
						} else {
							if m, r, rm, s, _, _, wt := forwardGlobComp(ctx, suffix, nil); m {
								return true, concat(res, gap, r), rm, concat(stems, gapseg{true, elems[iE], gap}, s), len(elems), len(vals), wt
							}
						}
					}
				} else {
					for k := len(gap); k != -1; k += -1 {
						if m, r, rm, s, _, _, wt := backwardGlobComp(ctx, suffix, gap[k:]); m { 
							stemParts := concat(gap[:k], rm)
							return true, concat(res, stemParts, r), nil, concat(stems, gapseg{true, elems[iE], stemParts}, s), len(elems), len(vals), wt
						}
					}
				}

				if tok == SAST {
					return false, concat(res, gap), nil, concat(stems, gapseg{true, elems[iE], gap}), iE, len(vals), ILLEGAL
				} else {
					return false, res, gap, stems, iE, iV, tok
				}

			default:
				return false, res, vals[iV:], stems, iE, iV, ILLEGAL
			}

		case *percpat:
			// 1. Dynamically explode the *percpat into [Prefix, DAST, Suffix]
			var exploded []Value
			if e.Prefix != nil && !isEmpty(e.Prefix) { exploded = append(exploded, e.Prefix) }
			exploded = append(exploded, _globmeta(e.Pos(), DAST)) // '%' acts as '**'
			if e.Suffix != nil && !isEmpty(e.Suffix) { exploded = append(exploded, e.Suffix) }

			newElems := concat(exploded, elems[iE+1:])
			m, r, rm, s, ie_rec, iv_ret, wt := forwardGlobComp(ctx, newElems, vals[iV:])

			if m {
				mapped_iE := iE
				if ie_rec >= len(exploded) {
					mapped_iE = iE + 1 + (ie_rec - len(exploded))
				}
				if rm == nil || isEmpty(rm) {
					if iv_ret > 0 { r = vals[iV : iV+iv_ret] }
				} else if iv_ret == 0 {
					matchedStr := ""
					for _, v := range r { matchedStr += getScalarSubstr(ctx, v, 0, -1) }
					r = []Value{_raw(vals[iV].Pos(), matchedStr)}
				}

				// CRITICAL FIX: Flatten the stem produced by the DAST explosion!
				// The stem for a '%' must be a single string, not a nested compound of atoms!
				if len(s) > 0 {
					stemStr := ""
					for _, v := range unpack(s[0]) {
						stemStr += getScalarSubstr(ctx, v, 0, -1)
					}
					s[0] = _raw(vals[iV].Pos(), stemStr)
				}

				return true, concat(res, r), rm, concat(stems, s), mapped_iE, iV + iv_ret, wt
			}

			// 2. Fallback: Intra-string match
			if iV < len(vals) {
				valStr := getScalarSubstr(ctx, vals[iV], 0, -1)
				pfx := ""
				if e.Prefix != nil && !isEmpty(e.Prefix) { pfx = getScalarSubstr(ctx, e.Prefix, 0, -1) }
				
				sfx := ""
				if e.Suffix != nil && !isEmpty(e.Suffix) {
					if pp, ok := unloc(e.Suffix).(*percpat); ok && isEmpty(pp.Prefix) {
						sfx = getScalarSubstr(ctx, pp.Suffix, 0, -1) // Handle %%
					} else {
						sfx = getScalarSubstr(ctx, e.Suffix, 0, -1)
					}
				}

				if len(pfx) + len(sfx) <= len(valStr) && strings.HasPrefix(valStr, pfx) {
					// Use LastIndex because Make's '%' is greedy!
					idx := strings.LastIndex(valStr[len(pfx):], sfx)
					if idx >= 0 {
						idx += len(pfx)
						stemStr := valStr[len(pfx):idx]
						matchedLen := idx + len(sfx)
						
						resAtom := _raw(vals[iV].Pos(), valStr[:matchedLen])
						stemAtom := _raw(vals[iV].Pos(), stemStr)
						
						var nextVals []Value
						hasRem := matchedLen < len(valStr)
						if hasRem {
							nextVals = append(nextVals, _raw(vals[iV].Pos(), valStr[matchedLen:]))
						}
						nextVals = concat(nextVals, vals[iV+1:])

						m_in, r_in, rm_in, s_in, ie_rec_in, iv_rec_in, wt_in := forwardGlobComp(ctx, elems[iE+1:], nextVals)
						if m_in {
							// Mathematically map the returned index consumption
							ret_iV := iV + 1
							if hasRem {
								if iv_rec_in > 0 { ret_iV += (iv_rec_in - 1) }
							} else {
								ret_iV += iv_rec_in
							}
							return true, concat(res, resAtom, r_in), rm_in, concat(stems, stemAtom, s_in), iE + 1 + ie_rec_in, ret_iV, wt_in
						}
					}
				}
			}

			return false, concat(res, r), rm, concat(stems, s), iE, iV + iv_ret, wt			

		default:
			if iV >= len(vals) { return false, res, nil, stems, iE, iV, ILLEGAL }

			m, r, rm := matchScalarScalar(ctx, e, vals[iV], false)
			if !m {
				if rm != nil && isEmpty(rm) { rm = nil }
				return false, concat(res, r), concat(rm, vals[iV+1:]), stems, iE, iV, ILLEGAL
			}

			res = concat(res, r)

			if rm == nil || isEmpty(rm) {
				iE++
				iV++
			} else {
				m, rr, rrm, s, ie, iv, wt := forwardGlobComp(ctx, elems[iE+1:], concat(rm, vals[iV+1:]))
				return m, concat(res, rr), rrm, concat(stems, s), iE + 1 + ie, iV + iv, wt
			}
		}
	}

	if iV < len(vals) { rem = vals[iV:] }
	return true, res, rem, stems, iE, iV, ILLEGAL
}

func backwardGlobComp(ctx Context, elems, vals []Value) (matched bool, res, rem, stems []Value, iE, iV int, wildToken token) {
	iE = len(elems) - 1
	iV = len(vals) - 1

	backward := func(str string, size int) (bool, []Value, []Value, []Value, int, int, token) {
		pos := vals[iV].Pos()
		val := _raw(pos, str[len(str)-size:]) 

		m, r, rm, s, ie, iv, wt := backwardGlobComp(ctx, elems[:iE],
			concat(vals[:iV], optraw{size < len(str), pos, str[:len(str)-size]}))

		return m, concat(r, val, res), rm, concat(s, val, stems), ie, iv, wt
	}

	for 0 <= iE {
		switch e := unloc(elems[iE]).(type) {
		case *globrange: 
			if iV < 0 { return false, res, nil, stems, iE, iV + 1, ILLEGAL } 
			str, t := getScalarSubstr(ctx, vals[iV], 0, -1), getScalarSubstr(ctx, e.Value, 0, -1)
			r, size := utf8.DecodeLastRuneInString(str) 
			if r == utf8.RuneError && size <= 1 || !globMatchCharSet(t, r) {
				return false, res, vals[:iV+1], stems, iE, iV + 1, ILLEGAL
			}
			return backward(str, size)

		case *globmeta:
			switch tok := getGlobToken(e); tok {
			case QUE: 
				if iV < 0 { return false, res, nil, stems, iE, iV + 1, ILLEGAL } 
				str := getScalarSubstr(ctx, vals[iV], 0, -1)
				r, size := utf8.DecodeLastRuneInString(str) 
				if r == utf8.RuneError && size <= 1 {
					return false, res, vals[:iV+1], stems, iE, iV + 1, ILLEGAL
				}
				return backward(str, size)

			case SAST, DAST, ASTQ: 
				prefix, gap := elems[:iE], vals[:iV+1]

				if tok == ASTQ {
					for k := len(gap); k >= 0; k-- {
						if k > 0 {
							str := getScalarSubstr(ctx, gap[k-1], 0, -1)
							pos := gap[k-1].Pos()
							
							for i := len(str); i > 0; {
								if m, r, rm, s, _, _, wt := backwardGlobComp(ctx, prefix, concat(gap[:k-1], _raw(pos, str[:i]))); m {
									stemParts := concat(optraw{i < len(str), pos, str[i:]}, gap[k:])
									return true, concat(r, stemParts, res), rm, concat(s, gapseg{true, elems[iE], stemParts}, stems), -1, 0, wt
								}
								
								_, size := utf8.DecodeLastRuneInString(str[:i])
								i -= size
							}
						} else {
							if m, r, rm, s, _, _, wt := backwardGlobComp(ctx, prefix, nil); m {
								return true, concat(r, gap, res), rm, concat(s, gapseg{true, elems[iE], gap}, stems), -1, 0, wt
							}
						}
					}
				} else {
					for k := 0; k != len(gap); k += 1 {
						if m, r, rm, s, _, _, wt := forwardGlobComp(ctx, prefix, gap[:k]); m { 
							stemParts := concat(rm, gap[k:])
							return true, concat(r, stemParts, res), nil, concat(s, gapseg{true, elems[iE], stemParts}, stems), -1, 0, wt
						}
					}
				}

				if tok == SAST {
					return false, concat(gap, res), nil, concat(gapseg{true, elems[iE], gap}, stems), iE, 0, ILLEGAL
				} else {
					return false, res, gap, stems, iE, iV + 1, tok
				}

			default:
				return false, res, vals[:iV+1], stems, iE, iV + 1, ILLEGAL
			}

		case *percpat:
			// 1. Dynamically explode the *percpat into [Prefix, DAST, Suffix]
			var exploded []Value
			if e.Prefix != nil && !isEmpty(e.Prefix) { exploded = append(exploded, e.Prefix) }
			exploded = append(exploded, _globmeta(e.Pos(), DAST))
			if e.Suffix != nil && !isEmpty(e.Suffix) { exploded = append(exploded, e.Suffix) }

			newElems := concat(elems[:iE], exploded)
			m, r, rm, s, ie_rec, iv_ret, wt := backwardGlobComp(ctx, newElems, vals[:iV+1])
			
			if m {
				mapped_iE := iE
				if ie_rec < len(elems[:iE]) {
					mapped_iE = ie_rec
				}

				if rm == nil || isEmpty(rm) {
					if iv_ret < iV+1 { r = vals[iv_ret : iV+1] }
				} else if iv_ret == iV+1 {
					matchedStr := ""
					for _, v := range r { matchedStr += getScalarSubstr(ctx, v, 0, -1) }
					r = []Value{_raw(vals[iV].Pos(), matchedStr)}
				}

				// CRITICAL FIX: Flatten the stem produced by the DAST explosion!
				// In backward matching, the current stem is appended at the END of the slice.
				if len(s) > 0 {
					idx := len(s) - 1
					stemStr := ""
					for _, v := range unpack(s[idx]) {
						stemStr += getScalarSubstr(ctx, v, 0, -1)
					}
					s[idx] = _raw(vals[iV].Pos(), stemStr)
				}

				return true, concat(r, res), rm, concat(s, stems), mapped_iE, iv_ret, wt
			}

			// 2. Fallback: Intra-string match (Backward traversal)
			if iV >= 0 {
				valStr := getScalarSubstr(ctx, vals[iV], 0, -1)
				pfx := ""
				if e.Prefix != nil && !isEmpty(e.Prefix) { pfx = getScalarSubstr(ctx, e.Prefix, 0, -1) }
				
				sfx := ""
				if e.Suffix != nil && !isEmpty(e.Suffix) {
					if pp, ok := unloc(e.Suffix).(*percpat); ok && isEmpty(pp.Prefix) {
						sfx = getScalarSubstr(ctx, pp.Suffix, 0, -1)
					} else {
						sfx = getScalarSubstr(ctx, e.Suffix, 0, -1)
					}
				}

				if len(pfx) + len(sfx) <= len(valStr) && strings.HasSuffix(valStr, sfx) {
					// Use Index to anchor the greedy match as far left as possible
					idx := strings.Index(valStr[:len(valStr)-len(sfx)], pfx)
					if idx >= 0 {
						stemStr := valStr[idx+len(pfx) : len(valStr)-len(sfx)]
						
						resAtom := _raw(vals[iV].Pos(), valStr[idx:])
						stemAtom := _raw(vals[iV].Pos(), stemStr)
						
						var nextVals []Value
						hasRem := idx > 0
						if hasRem {
							nextVals = append(nextVals, _raw(vals[iV].Pos(), valStr[:idx]))
						}
						nextVals = concat(vals[:iV], nextVals)

						m_in, r_in, rm_in, s_in, ie_rec_in, iv_rec_in, wt_in := backwardGlobComp(ctx, elems[:iE], nextVals)
						if m_in {
							return true, concat(r_in, resAtom, res), rm_in, concat(s_in, stemAtom, stems), ie_rec_in, iv_rec_in, wt_in
						}
					}
				}
			}

			return false, concat(r, res), rm, concat(s, stems), iE, iv_ret, wt	
			
		default:
			if iV < 0 { return false, res, nil, stems, iE, iV + 1, ILLEGAL }

			m, r, rm := matchScalarScalar(ctx, e, vals[iV], true)
			if !m {
				if rm != nil && isEmpty(rm) { rm = nil }
				return false, concat(r, res), concat(vals[:iV], rm), stems, iE, iV + 1, ILLEGAL
			}

			res = concat(r, res)

			if rm == nil || isEmpty(rm) {
				iE--
				iV--
			} else {
				m, rr, rrm, s, ie, iv, wt := backwardGlobComp(ctx, elems[:iE], concat(vals[:iV], rm))
				return m, concat(rr, res), rrm, concat(s, stems), ie, iv, wt 
			}
		}
	}

	if iV >= 0 { rem = vals[:iV+1] }
	return true, res, rem, stems, iE, iV + 1, ILLEGAL
}

func forwardGlobPath(ctx Context, elems, segments []Value) (matched bool, res, rem, stems []Value, iE, iS int, wildToken token) {
	matched, res, rem, stems, iE, _, wildToken = forwardGlobComp(ctx, elems, unpack(segments[0]))

	if matched && wildToken == ILLEGAL && len(segments) > 1 {
		tok := getGlobToken(elems[len(elems)-1]) // FIX: Unify retroactive trigger
		if tok == DAST || tok == ASTQ || tok == PERC {
			wildToken, iE = tok, len(elems)-1
			if len(stems) > 0 {
				rem = concat(unpack(stems[len(stems)-1]), rem)
				stems = stems[:len(stems)-1]
			}
		}
	}

	// Always calculate the full path remainder, even if matched is false
	fullRem := concat(
		packComp(rem),
		optraw{len(rem) == 0 && len(segments) > 1, segments[0].Pos(), ""},
		segments[1:],
	)

	if wildToken == DAST || wildToken == ASTQ || wildToken == PERC {
		// CRITICAL FIX: Ensure the element is a PURE wildcard before taking the shortcut!
		if isPureWildcard(elems[iE]) && len(elems[iE:]) == 1 {
			return true,
				segments, 
				nil,      
				concat(stems, stemseg{elems[iE], packPath(concat(gapseg{iE > 0, elems[iE], rem}, segments[1:]))}),
				len(elems), len(segments), ILLEGAL 
		}

		step, k := 1, 0
		if wildToken == DAST || wildToken == PERC { step, k = -1, len(segments)-1 }

		for ; k >= 0 && k < len(segments); k += step {
			if k == 0 && len(rem) > 0 {
				if mSuf, rSufAtoms, remSuf, sSuf, _, _, _ := forwardGlobComp(ctx, elems[iE:], rem); mSuf {
					return true,
						concat(packComp(concat(res, rSufAtoms))),
						concat(packComp(remSuf), optraw{len(remSuf) == 0 && len(segments) > 1, segments[0].Pos(), ""}, segments[1:]),
						concat(stems, sSuf), len(elems), 1, ILLEGAL
				}
			}

			if mSuf, rSufAtoms, remSuf, sSuf, _, _, _ := forwardGlobComp(ctx, elems[iE:], unpack(segments[k])); mSuf {
				return true,
					concat(segments[:k], packComp(rSufAtoms)),
					concat(
						packComp(remSuf),
						optraw{len(remSuf) == 0 && k+1 < len(segments), segments[k].Pos(), ""},
						segments[k+1:],
					),
					concat(stems, stemseg{elems[iE], packPath(concat(
						gapseg{iE > 0, elems[iE], rem},
						segments[1:k],
						gapseg{true, elems[iE], unpack(sSuf[0])},
					))}, sSuf[1:]), len(elems), k + 1, ILLEGAL
			}
		}

		return false,
			segments, 
			nil,      
			concat(stems, stemseg{elems[iE], packPath(concat(gapseg{iE > 0, elems[iE], rem}, segments[1:]))}),
			iE + 1, len(segments), wildToken 
	}

	if !matched { return false, concat(packComp(res)), fullRem, stems, iE, 1, ILLEGAL }
	return true, concat(packComp(res)), fullRem, stems, len(elems), 1, ILLEGAL
}

func backwardGlobPath(ctx Context, elems, segments []Value) (matched bool, res, rem, stems []Value, iE, iS int, wildToken token) {
	matched, res, rem, stems, iE, _, wildToken = backwardGlobComp(ctx, elems, unpack(segments[len(segments)-1]))

	if matched && wildToken == ILLEGAL && len(segments) > 1 {
		tok := getGlobToken(elems[0]) // FIX: Unify retroactive trigger
		if tok == DAST || tok == ASTQ || tok == PERC {
			wildToken, iE = tok, 0
			if len(stems) > 0 {
				rem = concat(rem, unpack(stems[0]))
				stems = stems[1:]
			}
		}
	}

	// Always calculate the full path remainder, even if matched is false
	fullRem := concat(
		segments[:len(segments)-1],
		packComp(rem),
		optraw{len(rem) == 0 && len(segments) > 1, segments[len(segments)-1].Pos(), ""},
	)

	if wildToken == DAST || wildToken == ASTQ || wildToken == PERC {
		// CRITICAL FIX: Ensure the element is a PURE wildcard before taking the shortcut!
		if isPureWildcard(elems[iE]) && len(elems[:iE+1]) == 1 {
			return true,
				segments, 
				nil,      
				concat(stemseg{elems[iE], packPath(concat(segments[:len(segments)-1], gapseg{iE+1 < len(elems), elems[iE], rem}))}, stems),
				-1, len(segments), ILLEGAL 
		}

		step, k := -1, len(segments)-1
		if wildToken == ASTQ { step, k = 1, 0 }

		for ; k >= 0 && k < len(segments); k += step {
			if k == len(segments)-1 && len(rem) > 0 {
				if mPre, rPreAtoms, remPre, sPre, _, _, _ := backwardGlobComp(ctx, elems[:iE+1], rem); mPre {
					return true,
						concat(packComp(concat(rPreAtoms, res))),
						concat(
							segments[:len(segments)-1],
							packComp(remPre),
							optraw{len(remPre) == 0 && len(segments) > 1, segments[len(segments)-1].Pos(), ""},
						),
						concat(sPre, stems), 0, 1, ILLEGAL
				}
			}

			if mPre, rPreAtoms, remPre, sPre, _, _, _ := backwardGlobComp(ctx, elems[:iE+1], unpack(segments[k])); mPre {
				return true,
					concat(packComp(rPreAtoms), segments[k+1:]),
					concat(
						segments[:k],
						packComp(remPre),
						optraw{len(remPre) == 0 && k > 0, segments[k].Pos(), ""},
					),
					concat(
						sPre[:len(sPre)-1],
						stemseg{elems[iE], packPath(concat(
							gapseg{true, elems[iE], unpack(sPre[len(sPre)-1])}, 
							segments[k+1:len(segments)-1],
							gapseg{iE+1 < len(elems), elems[iE], rem},
						))},
						stems,
					), -1, len(segments) - k, ILLEGAL
			}
		}

		return false,
			segments, 
			nil,      
			concat(stemseg{elems[iE], packPath(concat(segments[:len(segments)-1], gapseg{iE+1 < len(elems), elems[iE], rem}))}, stems),
			iE - 1, len(segments), wildToken 
	}

	if !matched { return false, concat(packComp(res)), fullRem, stems, iE, 1, ILLEGAL }
	return true, concat(packComp(res)), fullRem, stems, -1, 1, ILLEGAL
}

func forwardPathPath(ctx Context, elems, segments []Value) (matched bool, res, rem, stems []Value, iE, iS int) {
	for iE < len(elems) { 
		patAtoms := unpack(elems[iE])

		if len(patAtoms) == 1 {
			if p, ok := unloc(patAtoms[0]).(*punct); ok && p.token == PTAIL {
				if iS < len(segments) {
					var bypass = true
					if targetAtoms := unpack(segments[iS]); len(targetAtoms) == 1 {
						if t, ok := unloc(targetAtoms[0]).(*punct); ok && t.token == PTAIL { bypass = false }
					}
					if bypass {
						res = concat(res, packComp(patAtoms))
						iE++; continue
					}
				}
			}
		}

		if iS >= len(segments) { return false, res, nil, stems, iE, iS }

		m, resAtoms, remAtoms, s, ie, _, wt := forwardGlobComp(ctx, patAtoms, unpack(segments[iS]))

		if m && wt == ILLEGAL && iS+1 < len(segments) {
			tok := getGlobToken(patAtoms[len(patAtoms)-1])
			if tok == DAST || tok == ASTQ || tok == PERC {
				wt, ie = tok, len(patAtoms)-1
				if len(s) > 0 {
					remAtoms = concat(unpack(s[len(s)-1]), remAtoms)
					s = s[:len(s)-1]
				}
			}
		}

		partial := len(remAtoms) > 0

		if wt == DAST || wt == ASTQ || wt == PERC {
			sufAtoms := patAtoms[ie+1:]

			// BEAUTIFUL ABSTRACTION: Cleanly extract and prepend the suffix!
			if suf := getWildcardSuffix(patAtoms[ie]); suf != nil {
				sufAtoms = concat([]Value{suf}, sufAtoms)
			}

			pathElems := elems[iE+1:]
			gap := segments[iS+1:] 

			preBound := ie > 0
			postBound := len(sufAtoms) > 0

			isLast := len(sufAtoms) == 0 && (len(pathElems) == 0 || (len(pathElems) == 1 && func() bool {
				if p, ok := unloc(unpack(pathElems[0])[0]).(*punct); ok && p.token == PTAIL { return true }
				return false
			}()))

			if wt == ASTQ && !isLast {
				for k := 0; k <= len(gap); k++ {
					var targetAtoms []Value
					var targetPos Pos
					if k == 0 {
						if !partial { continue }
						targetAtoms = remAtoms
						targetPos = segments[iS].Pos()
					} else {
						targetAtoms = unpack(gap[k-1])
						targetPos = gap[k-1].Pos()
					}

					str := getScalarSubstr(ctx, packComp(targetAtoms), 0, -1)
					for i := 0; i <= len(str); {
						var mSuf bool
						var rSuf, remSuf, sSuf []Value
						
						if len(sufAtoms) > 0 {
							var testVals []Value
							if i < len(str) { testVals = unpack(_raw(targetPos, str[i:])) }
							mSuf, rSuf, remSuf, sSuf, _, _, _ = forwardGlobComp(ctx, sufAtoms, testVals)
						} else {
							mSuf = true
							if i < len(str) { remSuf = unpack(_raw(targetPos, str[i:])) }
						}

						if mSuf {
							var gapAtoms []Value
							if i > 0 { gapAtoms = []Value{_raw(targetPos, str[:i])} }

							var nextSegs []Value
							if k == 0 {
								nextSegs = concat(packComp(remSuf), optraw{len(remSuf) == 0 && len(gap) > 0, targetPos, ""}, gap)
							} else {
								nextSegs = concat(packComp(remSuf), optraw{len(remSuf) == 0 && k < len(gap), targetPos, ""}, gap[k:])
							}

							if mPath, rPath, remPath, sPath, iEPath, iSPath := forwardPathPath(ctx, pathElems, nextSegs); mPath {
								shift := iSPath
								if shift == 0 { shift = 1 } 

								if k == 0 {
									return true,
										concat(res, packComp(concat(resAtoms, gapAtoms, rSuf)), rPath),
										remPath,
										concat(stems, s, stemseg{patAtoms[ie], packPath(concat(gapseg{postBound, patAtoms[ie], gapAtoms}))}, sSuf, sPath),
										iE + 1 + iEPath, iS + shift
								} else {
									return true,
										concat(res, segments[iS], gap[:k-1], packComp(concat(gapAtoms, rSuf)), rPath),
										remPath,
										concat(stems, s, stemseg{patAtoms[ie], packPath(concat(
											gapseg{preBound, patAtoms[ie], remAtoms},
											gap[:k-1],
											gapseg{postBound, patAtoms[ie], gapAtoms},
										))}, sSuf, sPath),
										iE + 1 + iEPath, iS + k + shift
								}
							}
						}
						if i == len(str) { break }
						_, size := utf8.DecodeRuneInString(str[i:])
						i += size
					}
				}
			} else {
				for k := len(gap); k >= 0; k-- {
					if k == 0 {
						if !partial { continue }
						var mSuf bool
						var remSuf, sSuf []Value
						
						if len(sufAtoms) > 0 {
							mSuf, _, remSuf, sSuf, _, _, _ = forwardGlobComp(ctx, sufAtoms, remAtoms)
							if !mSuf || len(remSuf) > 0 {
								if mB, _, remB, sB, _, _, _ := backwardGlobComp(ctx, sufAtoms, remAtoms); mB {
									mSuf, remSuf, sSuf = mB, remB, sB
								}
							}
							if !mSuf { continue }
						} else {
							remSuf = remAtoms
						}

						if mPath, rPath, remPath, sPath, iEPath, iSPath := forwardPathPath(ctx, pathElems, gap); mPath {
							return true,
								concat(res, segments[iS], rPath),
								remPath,
								concat(stems, s, stemseg{patAtoms[ie], packPath(concat(gapseg{postBound, patAtoms[ie], remSuf}))}, sSuf, sPath),
								iE + 1 + iEPath, iS + 1 + iSPath
						}
					} else {
						var mSuf bool
						var remSuf, sSuf []Value
						
						if len(sufAtoms) > 0 {
							mSuf, _, remSuf, sSuf, _, _, _ = forwardGlobComp(ctx, sufAtoms, unpack(gap[k-1]))
							if !mSuf || len(remSuf) > 0 {
								if mB, _, remB, sB, _, _, _ := backwardGlobComp(ctx, sufAtoms, unpack(gap[k-1])); mB {
									mSuf, remSuf, sSuf = mB, remB, sB
								}
							}
							if !mSuf { continue }
						} else {
							remSuf = unpack(gap[k-1])
						}

						if mPath, rPath, remPath, sPath, iEPath, iSPath := forwardPathPath(ctx, pathElems, gap[k:]); mPath {
							return true,
								concat(res, segments[iS], gap[:k], rPath),
								remPath,
								concat(stems, s, stemseg{patAtoms[ie], packPath(concat(
									gapseg{preBound, patAtoms[ie], remAtoms},
									gap[:k-1],
									gapseg{postBound, patAtoms[ie], remSuf},
								))}, sSuf, sPath),
								iE + 1 + iEPath, iS + 1 + k + iSPath
						}
					}
				}
			}

			return false,
				concat(res, segments[iS:]),
				nil,
				concat(stems, s, stemseg{patAtoms[ie], packPath(concat(
					gapseg{preBound, patAtoms[ie], remAtoms},
					gap,
				))}),
				iE + 1, len(segments)
		}
		
		res, stems = concat(res, packComp(resAtoms)), concat(stems, s)
		rem = concat(
			packComp(remAtoms),
			optraw{len(remAtoms) == 0 && iS+1 < len(segments), segments[iS].Pos(), ""},
			segments[iS+1:],
		)

		if !m { return false, res, rem, stems, iE, iS + 1 }

		if partial {
			if iE < len(elems)-1 { return false, res, rem, stems, iE, iS + 1 }
			return true, res, rem, stems, iE + 1, iS + 1
		}

		iE++
		iS++
	}

	if iS < len(segments) && iS > 0 {
		consumedSlash := false
		if len(elems) > 0 {
			if lastPat := unpack(elems[len(elems)-1]); len(lastPat) == 1 {
				if p, ok := unloc(lastPat[0]).(*punct); ok && p.token == PTAIL { consumedSlash = true }
			}
		}
		if !consumedSlash {
			rem = concat(&valbase{segments[iS].Pos()}, segments[iS:])
		} else {
			rem = segments[iS:]
		}
	} else {
		rem = segments[iS:]
	}
	return true, res, rem, stems, iE, iS
}

func backwardPathPath(ctx Context, elems, segments []Value) (matched bool, res, rem, stems []Value, iE, iS int) {
	iE, iS = len(elems) - 1, len(segments) - 1

	for iE >= 0 { 
		patAtoms := unpack(elems[iE])

		if len(patAtoms) == 1 {
			if p, ok := unloc(patAtoms[0]).(*punct); ok && p.token == PROOT {
				if 0 <= iS {
					var bypass = true
					if targetAtoms := unpack(segments[iS]); len(targetAtoms) == 1 {
						if t, ok := unloc(targetAtoms[0]).(*punct); ok && t.token == PROOT { bypass = false }
					}
					if bypass {
						res = concat(packComp(patAtoms), res) 
						iE--; continue
					}
				}
			}
		}

		if iS < 0 { return false, res, nil, stems, iE + 1, iS + 1 }

		m, resAtoms, remAtoms, s, ie, _, wt := forwardGlobComp(ctx, patAtoms, unpack(segments[iS]))
		if !m || len(remAtoms) > 0 {
			if mB, resB, remB, sB, ieB, _, wtB := backwardGlobComp(ctx, patAtoms, unpack(segments[iS])); mB {
				m, resAtoms, remAtoms, s, ie, wt = mB, resB, remB, sB, ieB, wtB
			}
		}

		if m && wt == ILLEGAL && iS > 0 {
			tok := getGlobToken(patAtoms[0])
			if tok == DAST || tok == ASTQ || tok == PERC {
				wt, ie = tok, 0
				if len(s) > 0 {
					remAtoms = concat(remAtoms, unpack(s[0]))
					s = s[1:]
				}
			}
		}

		partial := len(remAtoms) > 0

		if wt == DAST || wt == ASTQ || wt == PERC {
			preAtoms := patAtoms[:ie]

			// BEAUTIFUL ABSTRACTION: Cleanly extract and append the prefix!
			if pfx := getWildcardPrefix(patAtoms[ie]); pfx != nil {
				preAtoms = concat(preAtoms, []Value{pfx})
			}

			pathElems := elems[:iE]
			gap := segments[:iS] 

			preBound := len(preAtoms) > 0
			postBound := ie+1 < len(patAtoms)

			isFirst := len(preAtoms) == 0 && (len(pathElems) == 0 || (len(pathElems) == 1 && func() bool {
				if p, ok := unloc(unpack(pathElems[0])[0]).(*punct); ok && p.token == PROOT { return true }
				return false
			}()))

			if wt == ASTQ && !isFirst {
				for k := len(gap); k >= 0; k-- {
					var targetAtoms []Value
					var targetPos Pos
					if k == len(gap) {
						if !partial { continue }
						targetAtoms = remAtoms
						targetPos = segments[iS].Pos()
					} else {
						targetAtoms = unpack(gap[k])
						targetPos = gap[k].Pos()
					}

					str := getScalarSubstr(ctx, packComp(targetAtoms), 0, -1)
					for i := len(str); i >= 0; {
						var mPre bool
						var rPre, remPre, sPre []Value
						
						if len(preAtoms) > 0 {
							var testVals []Value
							if i > 0 { testVals = unpack(_raw(targetPos, str[:i])) }
							mPre, rPre, remPre, sPre, _, _, _ = forwardGlobComp(ctx, preAtoms, testVals)
							if !mPre || len(remPre) > 0 {
								if mB, rB, remB, sB, _, _, _ := backwardGlobComp(ctx, preAtoms, testVals); mB {
									mPre, rPre, remPre, sPre = mB, rB, remB, sB
								}
							}
						} else {
							mPre = true
							if i > 0 { remPre = unpack(_raw(targetPos, str[:i])) }
						}

						if mPre {
							var gapAtoms []Value
							if i < len(str) { gapAtoms = []Value{_raw(targetPos, str[i:])} }

							var nextSegs []Value
							if k == len(gap) {
								nextSegs = concat(gap, packComp(remPre), optraw{len(remPre) == 0 && len(gap) > 0, targetPos, ""})
							} else {
								nextSegs = concat(gap[:k], packComp(remPre), optraw{len(remPre) == 0 && k > 0, targetPos, ""})
							}

							if mPath, rPath, remPath, sPath, iEPath, iSPath := backwardPathPath(ctx, pathElems, nextSegs); mPath {
								shift := iSPath
								if shift == len(nextSegs) { shift-- } 

								if k == len(gap) {
									return true,
										concat(rPath, packComp(concat(rPre, gapAtoms, resAtoms)), res),
										remPath,
										concat(sPath, sPre, stemseg{patAtoms[ie], packPath(concat(gapseg{preBound, patAtoms[ie], gapAtoms}))}, s, stems),
										iEPath, shift
								} else {
									return true,
										concat(rPath, packComp(concat(rPre, gapAtoms)), gap[k+1:], segments[iS], res),
										remPath,
										concat(sPath, sPre, stemseg{patAtoms[ie], packPath(concat(
											gapseg{preBound, patAtoms[ie], gapAtoms},
											gap[k+1:],
											gapseg{postBound, patAtoms[ie], remAtoms},
										))}, s, stems),
										iEPath, shift
								}
							}
						}
						if i == 0 { break }
						_, size := utf8.DecodeLastRuneInString(str[:i])
						i -= size
					}
				}
			} else {
				for k := 0; k <= len(gap); k++ {
					if k == len(gap) {
						if !partial { continue }
						var mPre bool
						var remPre, sPre []Value

						if len(preAtoms) > 0 {
							mPre, _, remPre, sPre, _, _, _ = forwardGlobComp(ctx, preAtoms, remAtoms)
							if !mPre || len(remPre) > 0 {
								if mB, _, remB, sB, _, _, _ := backwardGlobComp(ctx, preAtoms, remAtoms); mB {
									mPre, remPre, sPre = mB, remB, sB
								}
							}
							if !mPre { continue }
						} else {
							mPre = true
							remPre = remAtoms
						}

						if mPath, rPath, remPath, sPath, iEPath, iSPath := backwardPathPath(ctx, pathElems, gap); mPath {
							return true,
								concat(rPath, segments[iS], res),
								remPath,
								concat(sPath, sPre, stemseg{patAtoms[ie], packPath(concat(gapseg{preBound, patAtoms[ie], remPre}))}, s, stems),
								iEPath, iSPath
						}
					} else {
						var mPre bool
						var remPre, sPre []Value
						
						if len(preAtoms) > 0 {
							mPre, _, remPre, sPre, _, _, _ = forwardGlobComp(ctx, preAtoms, unpack(gap[k]))
							if !mPre || len(remPre) > 0 {
								if mB, _, remB, sB, _, _, _ := backwardGlobComp(ctx, preAtoms, unpack(gap[k])); mB {
									mPre, remPre, sPre = mB, remB, sB
								}
							}
							if !mPre { continue }
						} else {
							mPre = true
							remPre = unpack(gap[k])
						}

						if mPath, rPath, remPath, sPath, iEPath, iSPath := backwardPathPath(ctx, pathElems, gap[:k]); mPath {
							return true,
								concat(rPath, gap[k:], segments[iS], res),
								remPath,
								concat(sPath, sPre, stemseg{patAtoms[ie], packPath(concat(
									gapseg{preBound, patAtoms[ie], remPre},
									gap[k+1:],
									gapseg{postBound, patAtoms[ie], remAtoms},
								))}, s, stems),
								iEPath, iSPath
						}
					}
				}
			}

			return false,
				concat(segments[:iS+1], res),
				nil,
				concat(stemseg{patAtoms[ie], packPath(concat(
					gap,
					gapseg{postBound, patAtoms[ie], remAtoms},
				))}, s, stems),
				iE + 1, 0 
		}
		
		res, stems = concat(packComp(resAtoms), res), concat(s, stems)
		rem = concat(
			segments[:iS],
			packComp(remAtoms),
			optraw{len(remAtoms) == 0 && iS > 0, segments[iS].Pos(), ""},
		)

		if !m { return false, res, rem, stems, iE + 1, iS }

		if partial {
			if iE > 0 { return false, res, rem, stems, iE + 1, iS }
			return true, res, rem, stems, iE, iS
		}

		iE--
		iS--
	}

	if iS >= 0 && iS < len(segments)-1 {
		consumedSlash := false
		if len(elems) > 0 {
			if firstPat := unpack(elems[0]); len(firstPat) == 1 {
				if p, ok := unloc(firstPat[0]).(*punct); ok && p.token == PROOT { consumedSlash = true }
			}
		}
		if !consumedSlash {
			rem = concat(segments[:iS+1], &valbase{segments[iS+1].Pos()})
		} else {
			rem = segments[:iS+1]
		}
	} else {
		rem = segments[:iS+1]
	}
	return true, res, rem, stems, iE + 1, iS + 1 
}

// matchScalarScalar matches a scalar value against a scalar value.
// Returns matched=true if the pattern successfully matched (fully or partially).
// res is the portion of the value that matched.
// rem is the leftover unconsumed portion of the value.
func matchScalarScalar(ctx Context, pat, val Value, trail bool) (matched bool, res, rem Value) {
	pStr := getScalarSubstr(ctx, pat, 0, -1)
	vStr := getScalarSubstr(ctx, val, 0, -1)
	
	// 1. Exact Match (Cleanly handles identical strings and structural boundaries)
	if pStr == vStr {
		return true, val, nil
	}
	
	// 2. 0-Byte Prefix Consumption
	if pStr == "" {
		if isEmpty(pat) { return true, pat, val }
		return false, nil, val
	}

	// 3. Atomic Token Protection
	// If 'val' is an atomic structural token (like *punct or *globmeta),
	// it CANNOT be partially sliced. Since pStr != vStr and pStr != "", it fails.
	switch unloc(val).(type) {
	case *punct, *globmeta:
		return false, nil, val
	}
	
	// 4. Partial String Match
	if trail {
		if strings.HasSuffix(vStr, pStr) {
			res = _raw(val.Pos(), pStr)
			rem = _raw(val.Pos(), vStr[:len(vStr)-len(pStr)])
			return true, res, rem
		}
	} else {
		if strings.HasPrefix(vStr, pStr) {
			res = _raw(val.Pos(), pStr)
			rem = _raw(val.Pos(), vStr[len(pStr):])
			return true, res, rem
		}
	}
	return false, nil, val
}

// matchCompComp matches a segment (elems) against another segment (vals) (non-path).
// It iterates strictly (no wildcards), consuming vals segment by segment.
// Returns matched=true if ALL pattern elements were successfully matched and consumed.
// Returns rem as the remainder of the last partially consumed value segment.
func matchCompComp(ctx Context, elems, vals []Value, trail bool) (matched bool, res, rem []Value, iE, iV int) {
	if checkpoints { defer check_matchCompComp(ctx, elems, vals, trail)(&matched, &res, &rem, &iE, &iV) }
	if trail { return backwardCompComp(ctx, elems, vals) } else { return forwardCompComp(ctx, elems, vals) }
}

// matchGlobComp matches glob elements (elems) against value atoms (vals).
func matchGlobComp(ctx Context, elems, vals []Value, trail bool) (matched bool, res, rem, stems []Value, iE, iV int, wildToken token) {
	if checkpoints { defer check_matchGlobComp(ctx, elems, vals, trail)(&matched, &res, &rem, &stems, &iE, &iV, &wildToken) }
	if trail { return backwardGlobComp(ctx, elems, vals) } else { return forwardGlobComp(ctx, elems, vals) }
}

// matchGlobPath matches glob elements (elems) against path segments (segments).
func matchGlobPath(ctx Context, elems, segs []Value, trail bool) (matched bool, res, rem, stems []Value, iE, iS int, wildToken token) {
	if checkpoints { defer check_matchGlobPath(ctx, elems, segs, trail)(&matched, &res, &rem, &stems, &iE, &iS, &wildToken) }
	if trail { return backwardGlobPath(ctx, elems, segs) } else { return forwardGlobPath(ctx, elems, segs) }
}

// matchPathPath match path segments (elems) against path segments.
func matchPathPath(ctx Context, elems, segments []Value, trail bool) (matched bool, res, rem, stems []Value, iE, iS int) {
	if checkpoints { defer check_matchPathPath(ctx, elems, segments, trail)(&matched, &res, &rem, &stems, &iE, &iS) }
	if trail { return backwardPathPath(ctx, elems, segments) } else { return forwardPathPath(ctx, elems, segments) }
}

// match matches pattern `pat` against value `val`.
// Returns matched=true if the pattern was completely satisfied, even if the target value has a remainder.
// The unconsumed portion of the value is returned as `rem`.
func match(ctx Context, pat, val Value) (matched bool, res, rem Value, stems []Value) {
	switch p := pat.(type) {
	case *loc: return match(ctx, p.Value, val)
	case *xloc: return match(ctx, p.Value, val)
	case *globbrace: return match(ctx, &p.globpat, val)
	case *globmeta, *globrange, *percpat: return match(ctx, _globpat(pat), val)
	case *delegate, *closure: return false, nil, nil, nil
	case *file: 
		switch segs := splitPathStr(ctx, p.name); len(segs) {
		case 0: rem = val ; goto Normalize
		case 1: return match(ctx, segs[0], val)
		default: return match(ctx, packPath(segs), val)
		}
	case flag:
		if v, ok := val.(flag); ok { return match(ctx, p.Value, v.Value) }
	}
	switch v := val.(type) {
	case *loc: return match(ctx, pat, v.Value)
	case *xloc: return match(ctx, pat, v.Value)
	case *globbrace: return match(ctx, pat, &v.globpat)
	case *globmeta, *globrange, *percpat: return match(ctx, pat, _globpat(val))
	case *delegate, *closure: return false, nil, nil, nil
	case *file: 
		switch segs := splitPathStr(ctx, v.name); len(segs) {
		case 0: rem = val ; goto Normalize
		case 1: return match(ctx, pat, segs[0])
		default: return match(ctx, pat, packPath(segs))
		}
	}

	if checkpoints { defer check_match(ctx, pat, val)(&matched, &res, &rem, &stems) }

	{ trail := truly(ctx, propReversal); switch p := pat.(type) {
	case *globpat:
		switch v := val.(type) {
		case *path:
			var r, rm []Value
			matched, r, rm, stems, _, _, _ = matchGlobPath(ctx, p.elems, v.elems, trail)
			res, rem = packPath(r), packPath(rm)
		case *globpat:
			var r, rm []Value
			matched, r, rm, stems, _, _, _ = matchGlobComp(ctx, p.elems, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		case *compound:
			var r, rm []Value
			matched, r, rm, stems, _, _, _ = matchGlobComp(ctx, p.elems, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		case *qualword: // CRITICAL FIX: Unpack qualwords!
			var r, rm []Value
			matched, r, rm, stems, _, _, _ = matchGlobComp(ctx, p.elems, unpack(v), trail)
			res, rem = packComp(r), packComp(rm)
		default:
			var r, rm []Value
			matched, r, rm, stems, _, _, _ = matchGlobComp(ctx, p.elems, []Value{v}, trail)
			res, rem = packComp(r), packComp(rm)
		}

	case *compound:
		switch v := val.(type) {
		case *path:
			if trail {
				matched, res, rem, stems = match(ctx, pat, v.elems[len(v.elems)-1])
				rem = packPath(concat(v.elems[:len(v.elems)-1], optraw{(rem == nil || isEmpty(rem)) && len(v.elems) > 1, v.elems[len(v.elems)-1].Pos(), ""}, rem))
			} else {
				matched, res, rem, stems = match(ctx, pat, v.elems[0])
				rem = packPath(concat(rem, optraw{(rem == nil || isEmpty(rem)) && len(v.elems) > 1, v.elems[0].Pos(), ""}, v.elems[1:]))
			}
		case *compound:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, p.elems, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		case *qualword: // CRITICAL FIX: Unpack qualwords!
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, p.elems, unpack(v), trail)
			res, rem = packComp(r), packComp(rm)
		case *globpat:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, p.elems, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		default:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, p.elems, []Value{v}, trail)
			res, rem = packComp(r), packComp(rm)
		}

	case *qualword:
		pElems := unpack(p) // Unpack the pattern into structural elements
		switch v := val.(type) {
		case *path:
			if trail {
				matched, res, rem, stems = match(ctx, pat, v.elems[len(v.elems)-1])
				rem = packPath(concat(v.elems[:len(v.elems)-1], optraw{(rem == nil || isEmpty(rem)) && len(v.elems) > 1, v.elems[len(v.elems)-1].Pos(), ""}, rem))
			} else {
				matched, res, rem, stems = match(ctx, pat, v.elems[0])
				rem = packPath(concat(rem, optraw{(rem == nil || isEmpty(rem)) && len(v.elems) > 1, v.elems[0].Pos(), ""}, v.elems[1:]))
			}
		case *compound:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, pElems, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		case *qualword:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, pElems, unpack(v), trail)
			res, rem = packComp(r), packComp(rm)
		case *globpat:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, pElems, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		default:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, pElems, []Value{v}, trail)
			res, rem = packComp(r), packComp(rm)
		}

	case *path:
		switch v := val.(type) {
		case *path:
			var r, rm []Value
			matched, r, rm, stems, _, _ = matchPathPath(ctx, p.elems, v.elems, trail)
			res, rem = packPath(r), packPath(rm)
		default:
			matched, res, rem, stems = match(ctx, p.elems[0], v)
			matched = matched && len(p.elems) <= 1
		}

	case *regexpat:
		if p.Regexp == nil { debug(ctx, "err match: <nil-regexp> %s", ts(val), callstack{num: 10}, trace{}) }
		if sm := p.Regexp.FindStringSubmatch(__string(ctx, val)); sm != nil {
			res = _raw(val.Pos(), sm[0])
			for _, s := range sm[1:] { stems = append(stems, _raw(val.Pos(), s)) }
			matched = true
		} else {
			rem = val
		}
		goto Normalize

	case *list:
		if t, ok := val.(*list); ok {
			for _, _p := range p.elems {
				for _, _v := range t.elems {
					if matched, res, rem, stems = match(ctx, _p, _v); matched { goto Normalize }
				}
			}
		} else {
			for _, _p := range p.elems {
				if matched, res, rem, stems = match(ctx, _p, val); matched { goto Normalize }
			}
		}
		goto Normalize

	default: 
		switch v := val.(type) {
		case *path:
			if trail {
				matched, res, rem, stems = match(ctx, pat, v.elems[len(v.elems)-1])
				rem = packPath(concat(v.elems[:len(v.elems)-1], optraw{(rem == nil || isEmpty(rem)) && len(v.elems) > 1, v.elems[len(v.elems)-1].Pos(), ""}, rem))
			} else {
				matched, res, rem, stems = match(ctx, pat, v.elems[0])
				rem = packPath(concat(rem, optraw{(rem == nil || isEmpty(rem)) && len(v.elems) > 1, v.elems[0].Pos(), ""}, v.elems[1:]))
			}
		case *compound:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, []Value{pat}, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		case *globpat:
			var r, rm []Value
			matched, r, rm, _, _ = matchCompComp(ctx, []Value{pat}, v.elems, trail)
			res, rem = packComp(r), packComp(rm)
		default:
			matched, res, rem = matchScalarScalar(ctx, pat, val, trail)
		}
	}}

Normalize:
	if !matched && !truly(ctx, is_swapped{}) && patterned(ctx, val) {
		if matched, _, _, _ = match(swapped_ctx{ctx}, val, pat); matched { res, rem, stems = val, nil, nil }
	}

	// Universal unwrapper: safely destroys degenerate wrappers and empty AST ghosts
	res, rem = correctMatchRes(res), correctMatchRes(rem)

	// Universal fallback: if the matcher completely failed and dropped the remainder,
	// securely restore the unconsumed original value
	if !matched && res == nil && rem == nil { rem = val }
	return
}

func correctMatchRes(v Value) Value {
	for v != nil {
		if isEmpty(v) { return nil }
		switch t := v.(type) {
		case *compound: if len(t.elems) == 1 { v = t.elems[0]; continue }
		case *path:     if len(t.elems) == 1 { v = t.elems[0]; continue }
		}
		break
	}
	return v
}

func stencil(ctx Context, pat Value, stems []Value) (res Value, rest []Value) {
	switch p := pat.(type) {
	case *loc: 
		return stencil(ctx, p.Value, stems)
	case *xloc: 
		return stencil(ctx, p.Value, stems)
	case *rule: 
		return stencil(ctx, p.target, stems)
	case flag:
		var name Value
		name, rest = stencil(ctx, p.Value, stems)
		return flag{name}, rest
	case *compound:
		v := new(compound)
		rest = stems
		for _, elem := range p.elems {
			var t Value
			t, rest = stencil(ctx, elem, rest)
			v.elems = append(v.elems, t)
		}
		return v, rest
	case *qualword:
		v := new(qualword)
		rest = stems
		for _, elem := range p.elems {
			var t Value
			t, rest = stencil(ctx, elem, rest)
			v.elems = append(v.elems, t)
		}
		return v, rest
	case *path:
		v := new(path)
		rest = stems
		// REFINED: Iterate natively. Do not call xmerge here! Stenciling 
		// is a structural mapping, not a runtime evaluation.
		for _, elem := range p.elems {
			var t Value
			if t, rest = stencil(ctx, elem, rest); isTrivial(t) { t = elem }
			v.elems = append(v.elems, t)
		}
		return v, rest
	case *list:
		v := new(list)
		rest = stems
		for _, elem := range p.elems {
			var t Value
			t, rest = stencil(ctx, elem, rest)
			v.elems = append(v.elems, t)
		}
		return v, rest
	case *pair:
		v := new(pair)
		v.key, rest = stencil(ctx, p.key, stems)
		v.val, rest = stencil(ctx, p.val, rest)
		return v, rest
	case *percpat:
		var vals []Value
		
		// 1. Prefix
		if !isTrivial(p.Prefix) {
			if patterned(ctx, p.Prefix) {
				debug(ctx, "patterned prefix: %T %v", p.Prefix, p.Prefix, trace{})
			} else {
				vals = append(vals, p.Prefix)
			}
		}

		// 2. Stem Substitution (The '%')
		rest = stems
		if len(stems) > 0 {
			if s := stems[0]; !isEmpty(s) { vals = append(vals, s) }
			rest = stems[1:]
		}

		// 3. Suffix
		if !isTrivial(p.Suffix) {
			// REFINED: Cleaned up the nested *percpat checks and gotos
			var suffix = p.Suffix
			if pp, ok := p.Suffix.(*percpat); ok && isTrivial(pp.Prefix) && !isTrivial(pp.Suffix) {
				suffix = pp.Suffix
			}
			
			if suf, r := stencil(ctx, suffix, rest); !isNull(suf) && suf != suffix {
				vals = append(vals, suf)
				rest = r
			} else {
				vals = append(vals, suffix)
			}
		}

		// 4. Flatten the Result
		if len(vals) == 1 {
			return vals[0], rest
		}
		return &compound{elements{vals}}, rest

	case *barefile:
		var name Value
		if p.file != nil {
			res, rest = stencil(ctx, p.file, stems)
		} else if name, rest = stencil(ctx, p.Value, stems); name != p.Value {
			res = &barefile{name, p.file}
		} else {
			res = p
		}
		return res, rest
		
	default:
		return pat, stems
	}
}

func patterned(ctx Context, v any) (res bool) {
    switch t := v.(type) {
    case []Value: for _, t := range t { if patterned(ctx, t) { return true } }
    case *regexpat, *percpat, *globpat, *globrange, *globmeta: return true
    case *loc: return patterned(ctx, t.Value)
    case *xloc: return patterned(ctx, t.Value)
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
    case *xloc: return stamp(ctx, t.Value)
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
        debug(pc(ctx,a), "%v", ts(a), trace{})
        return
    }
}

func stat(ctx Context, a any) (res []*statinfo) {
    switch t := a.(type) {
    case *loc: return stat(ctx, t.Value)
    case *xloc: return stat(ctx, t.Value)
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
        debug(pc(ctx,a), "%v", ts(a), trace{})
        return
    }
}

func traverse(ctx Context, val Value) {
    switch p := val.(type) {
    case *loc: traverse(ctx, p.Value)
    case *xloc: traverse(ctx, p.Value)
    case *list: for _, e := range p.elems { traverse(ctx, e) }
    case *argumented: traverse(p.ctx(ctx), p.Value)
    case *auto: if v := auto_get(ctx, p.name); v != nil { traverse(ctx, v) }
    case *def: if v := p.value; v != nil { traverse(ctx, v) }
    case negative: if v := p.Value; v != nil { traverse(ctx, v) }
	case matched_rule: traverse(ctx, p.rule)
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
            debug(ctx, "$@ is not defined", trace{})
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
            debug(ctx, "%v: %v: %v", proj, p, target)
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
            debug(ctx, "selected trivial value '%v' (%v, %v) ", p, ts(p.o), ts(p.s))
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
            debug(ctx, "empty name: %v", p.elems[0], trace{})
        } else if truly(ctx, interpret{name, p.elems[1:]}) {
            modify(ctx, &p.group, true)
        }
    case *modification:
        if e := _execution(ctx); e != nil { e.Wait() }
        for _, m := range p.list { traverse(ctx, m) }
    case *compound, *word, *strlit, *strval, *strcomp, *qualword, *path, *percpat, *globpat, *regexpat, flag:
        do(ctx, act_traverse{p})
    default:
        debug(pc(ctx,p), "unsupported traversal: %v", ts(val), trace{})
    }
}

func unique(ctx Context, values ...Value) []Value {
	if len(values) == 0 {
		return nil
	}

	// 1. Pre-allocate slice capacity to completely eliminate memory copying during append
	elems := make([]Value, 0, len(values))
	
	// 2. Use []Value to safely chain hash collisions
	seen := make(map[uint64][]Value, len(values))

	for _, v := range values {
		n := hash(ctx, v)

		// Check if we've seen this hash
		if existing, found := seen[n]; found {
			// Slow path: Hash collision or duplicate. Verify structural equality!
			isDuplicate := false
			for _, ev := range existing {
				if equal(ctx, v, ev) {
					isDuplicate = true
					break
				}
			}
			if isDuplicate {
				continue
			}
		}

		// Fast path: First time seeing this value (or a legitimate hash collision)
		seen[n] = append(seen[n], v)
		elems = append(elems, v)
	}

	return elems
}

func reverse_unique(ctx Context, values ...Value) []Value {
	if len(values) == 0 {
		return nil
	}

	elems := make([]Value, 0, len(values))
	seen := make(map[uint64][]Value, len(values))

	for j := len(values) - 1; 0 <= j; j-- {
		v := values[j]
		n := hash(ctx, v)

		if existing, found := seen[n]; found {
			isDuplicate := false
			for _, ev := range existing {
				if equal(ctx, v, ev) {
					isDuplicate = true
					break
				}
			}
			if isDuplicate {
				continue
			}
		}

		seen[n] = append(seen[n], v)
		elems = append(elems, v)
	}

	return elems
}

func splitPathStr(ctx Context, str string) (segments []Value) {
    var pos = _pos(ctx)
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
				default  : v = _raw(pos, s)
            }
        } else if s == "" {
            if i+1 == len(a) {
                v = makePunct(pos, PTAIL)
            } else if false {
                v = makePunct(pos, PCON)
            } else {
                debug(ctx, "%s: %v[%d]: empty path seg", str, a, i)
                continue
            }
        } else {
            v = _raw(pos, s)
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
    case    bare : elems = append(elems, _word(   _pos(ctx),  string(t)))
    case    bool : elems = append(elems, _boolean(_pos(ctx),         t ))
    case    int  : elems = append(elems, _decimal(_pos(ctx),   int64(t)))
    case    int16: elems = append(elems, _decimal(_pos(ctx),   int64(t)))
    case    int32: elems = append(elems, _decimal(_pos(ctx),   int64(t)))
    case    int64: elems = append(elems, _decimal(_pos(ctx),         t ))
    case   uint  : elems = append(elems, _decimal(_pos(ctx),   int64(t)))
    case   uint16: elems = append(elems, _decimal(_pos(ctx),   int64(t)))
    case   uint32: elems = append(elems, _decimal(_pos(ctx),   int64(t)))
    case   uint64: elems = append(elems, _decimal(_pos(ctx),   int64(t)))
    case  float32: elems = append(elems, _float(  _pos(ctx), float64(t)))
    case  float64: elems = append(elems, _float(  _pos(ctx),         t ))
    case   string: elems = append(elems, _strlit( _pos(ctx),         t ))
    case []string: for _, s := range t { elems = append(elems, _strlit(_pos(ctx),        s )) }
    case   []bare: for _, s := range t { elems = append(elems, _word(  _pos(ctx), string(s))) }
	case  []*file: for _, f := range t { elems = append(elems, f) }
    default: debug(ctx, "unsupported value: %v", ts(t), trace{})
    }

    switch len(elems) {
    case 0: return _null(_pos(ctx))
    case 1: return elems[0]
    default: return &list{elements{elems}}
    }
}

func scalarize(v Value) (res Value) { // NOTE: unexpanded is not scalar
    switch t := v.(type) {
    case *list:
        var n = len(t.elems)
        if n == 0 { return _null(t.Pos()) }
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

// loc-prefix
func lp(ctx Context, ap any, t string, a ...any) (s string) {
	var p Position
	var rawPos Pos // Keep track of the raw integer for fallback

	// 1. Dynamically resolve the incoming position type!
	switch x := ap.(type) {
	case Position: p = x
	case Pos:
		if rawPos = x; x.IsValid() && ctx != nil {
			p, _ = do(ctx, get_fatpos{x}).(Position)
		}
	case positioner:
		if t := x.Pos(); t.IsValid() { if rawPos = t; ctx != nil {
			p, _ = do(ctx, get_fatpos{t}).(Position)
		}}
	}

	// 2. Format the output
	if ctx != nil {
		if f, f2 := p.Filename, _position(ctx).Filename; f != "" && f != f2 { s = f + ":" }
	}

	// 3. Smart Positional Printing (The Fix)
	if p.Line > 0 {
		s += fmt.Sprintf("%d:%d", p.Line, p.Column)
	} else if rawPos.IsValid() {
		// Fallback: If we have a valid Pos integer but no Context to resolve it, print the offset!
		s += fmt.Sprintf("@%d", rawPos)
	} else {
		// Only print 0:0 if the node ACTUALLY has no position data whatsoever
		s += "0:0"
	}

	if t != "" { s += ":" + t }

	// CRITICAL FIX: Push the local position into the context for variadic children!
	var local = ctx
	if ctx != nil && ap != nil { local = pc(ctx, ap) }

	// Safely range over the additional arguments using the new local context
	for _, val := range a { if val != nil { s += " " + ts(val, local) } }
	
	return
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

	// CRITICAL FIX: Establish a local context to strip redundant filenames from children!
	var cc = c
	if c != nil {
		if pos, ok := i.(positioner); ok && pos != nil {
			cc = pc(c, pos.Pos())
		}
	}

	var t string
	switch i.(type) {
	case *globpat: t = "glob"
	case *globmeta: t = "meta"
	case *globrange: t = "range"
	case *regexpat: t = "regex"
	default:
		if t = typeof(i); strings.HasPrefix(t, "[]") {
			v := reflect.ValueOf(i)
			s  = "["
			for i := 0; i < v.Len(); i += 1 {
				if 0 < i { s += " " }
				// Slices don't have positions, so we just pass cc directly
				s += ts(v.Index(i).Interface(), cc) 
			}
			s += "]"
			return
		}
	}

	switch x := i.(type) {
	case tst: return ts(x.i, cc)
	case    filemap: return "{="+t+" "+x.String()+"}"
	case    Context: return "{="+t+" "+ts(inner(x), cc)+"}"
	case        opt: return "{="+t+" "+ts(x.Value,cc)+"}"
	case    skipped: return "{="+t+" "+ts(x.Value,cc)+"}"
	case   negative: return "{="+t+" "+ts(x.Value,cc)+"}"
	case   fullname: return "{="+t+" "+ts(x.Value,cc)+"}"
	case       flag: return "{="+t+" "+ts(x.Value,cc)+"}"
	case      *rule: return "{="+t+" "+ts(x.target,cc)+"}"
	case   *percpat: return "{="+t+" "+ts(x.Prefix,cc)+" "+ts(x.Suffix,cc)+"}"
	case      *pair: return "{="+t+" "+ts(x.key,cc)+" "+ts(x.val,cc)+"}"
	case      *file: return "{="+t+" "+x.filestub.name+"}"
	case   fullfile: return "{="+t+" "+x.filestub.name+"}"
	case   *valbase: return "{"+lp(c, x.pos, "")+"}"
	case       *loc: return "{"+lp(c, x.pos, "", x.Value)+"}" // lp handles cc internally
	case      *xloc: return "{"+lp(c, x.pos, "", x.Value)+"}" // lp handles cc internally
	case      *auto: return "{"+lp(c, x.pos, t)+" "+x.name+"}"
	case       *def: return "{"+lp(c, x.pos, t)+" "+x.name+"}"
	case   *project: return "{"+lp(c, x.pos, t)+" "+x.name+"}"
	case       self: return "{"+lp(c, x.pos, t)+" "+x.name+"}"
	case  *regexpat: return "{"+lp(c, x.pos, t)+" "+x.Regexp.String()+"}"
	case  *globmeta: return "{"+lp(c, x.pos, t)+" "+x.token.String()+"}"
	case *globrange: return "{"+lp(c, x.Pos(), t)+" "+x.Value.String()+"}"
	case    *answer: return "{"+lp(c, x.pos, _if(x.bool, "yes", "no"))+"}"
	case   *boolean: return "{"+lp(c, x.pos, _if(x.bool, "true", "false"))+"}"
	case    *option: return "{"+lp(c, x.pos, _if(x.bool, "on", "off"))+"}"
	case *none, *null: return "{"+lp(c, x.(Value).Pos(), t)+"}"
	case *arrow: return "{="+t+" "+ts(x.o,cc)+x.t.String()+ts(x.s,cc)+"}"
	case *argumented: return "{="+t+" "+ts(x.Value,cc)+ts(x.args,cc)+"}"
	case *argumented_ctx:
		for i, a := range x.args { if i > 0 { s += "," }; s += a.String() }
		return "{="+t+" "+x.val.String()+"("+s+") "+ts(x.Context,cc)+"}"
	case *defcaps:
		var s string
		for _, cap := range x.caps { s += " {"+cap.name+":"+ts(cap.value,cc)+"}" }
		return "{="+t+" "+ts(x.Value,cc)+s+"}"
	case *evocation:
		var s = x.defs.String() ; if s != "" { s += " " }
		return "{="+t+" "+x.x.String()+" | "+s+ts(x.Context,cc)+"}"
	case *delegate:
		var s = "{" + lp(c, x.pos, t, x.x)
		if cc = pc(c, x.pos); x.o != nil { s += " " + ts(x.o, cc) }
		for _, a := range x.a { s += " " + ts(a, cc) }
		s += "}"
		return s
	case *closure:
		var s = "{" + lp(c, x.pos, t, x.x) 
		if cc = pc(c, x.pos); x.o != nil { s += " " + ts(x.o, cc) }
		for _, a := range x.a { s += " " + ts(a, cc) }
		s += "}"
		return s
	case *url:
		return "{"+lp(c,x.Pos(),t, x.Scheme,x.Username,x.Password,x.Host,x.Port,x.Path,x.Query,x.Fragment)+"}"
	case *strval:
		for _, v := range x.v { s += " "+ts(v,cc) }
		return "{"+lp(c, x.pos, t)+s+"}"
	case *punct:
		switch x.token {
		case PROOT: s = "PROOT"
		case PTAIL: s = "PTAIL"
		default: s = x.token.String()
		}
		return "{"+lp(c, x.pos, t)+" "+s+"}"
	case *disjunction:
		return "{"+lp(c, x.pos, t, x.val)+"}"
	case conjunction:
		// CRITICAL FIX: Use the global ts() since elements.ts no longer exists!
		return fmt.Sprintf("{=%s %s%s}", t, ts(x.list, cc), ts(x.sep, cc))
		
	case slicer:
		// Safely extract the compact Pos
		var pos Pos
		if p, ok := x.(positioner); ok { pos = p.Pos() }

		var prefix = "="
		if c != nil && pos.IsValid() {
			if fat, ok := do(c, get_fatpos{pos}).(Position); ok && fat.Filename != "" {
				// Change prefix to the absolute filename if it differs from the current trace context!
				if f1 := _position(c).Filename; f1 != fat.Filename {
					prefix = fat.Filename + ":"
				}
			}
		}

		if p, ok := x.(*plain); ok && p.name != "" {
			s += "(" + p.name + ")"
		}

		var cc = pc(c, pos)
		for _, a := range x.slice() {
			s += " " + ts(a, cc)
		}
		
		return "{" + prefix + t + s + "}"

	case Value:
		if t := x.String(); t != "" { s += " " + strings.ReplaceAll(t, "\n", `\n`) }
		return "{"+lp(c, x.Pos(), t)+s+"}"
	default:
		if t := fmt.Sprintf("%v", x); t != "" { s += " " + strings.ReplaceAll(t, "\n", `\n`) }
		return "{="+t+s+"}"
	}
}

func (p *valcache) km() (ss map[string]struct{}) {
	ss = make(map[string]struct{})
	for _, v := range reflect.ValueOf(p.v).MapKeys() {
		ss[fmt.Sprintf("%s", v.Interface())] = struct{}{}
	}
	return
}

type tst struct{ i any }
func (p tst) String() string { return ts(p.i) }
func (p tst) Pos() (_ Pos) {
    if x, y := p.i.(positioner); y && x != nil {
        return x.Pos()
    }
    return
}

func _argumented(val Value, a ...Value) *argumented { return &argumented{val, a} }
func _arrow(pos Pos, tok token, lhs, rhs Value) *arrow { return &arrow{valbase{pos}, tok, lhs, rhs} }
func _null(pos Pos) *null { return &null{valbase{pos}} }
func _none(pos Pos) *none { return &none{valbase{pos}} }
func _answer(pos Pos, v bool) *answer { return &answer{boolean{valbase{pos},v}} }
func _boolean(pos Pos, v bool) *boolean { return &boolean{valbase{pos},v} }
func _option(pos Pos, v bool) *option { return &option{boolean{valbase{pos},v}} }
func _binary(pos Pos, i int64) *binary { return &binary{integer{valbase{pos},i}} }
func _octal(pos Pos, i int64) *octal { return &octal{integer{valbase{pos},i}} }
func _decimal(pos Pos, i int64) *decimal { return &decimal{integer{valbase{pos},i}} }
func _hexadecimal(pos Pos, i int64) *hexadecimal { return &hexadecimal{integer{valbase{pos},i}} }
func _float(pos Pos, f float64) *float  { return &float{valbase{pos},f} }
func _strlit(pos Pos, s string) *strlit { return &strlit{valbase{pos},s} }
func _strcomp(elems ...Value) *strcomp { return &strcomp{elements{elems}} }
func _compound(elems ...Value) *compound { return &compound{elements{elems}} }
func _list(elems ...Value) *list { return &list{elements{elems}} }
func _group(pos Pos, elems ...Value) (v *group) { return &group{valbase{pos},elements{elems}} }
func _globmeta(pos Pos, tok token) *globmeta { return &globmeta{valbase{pos},tok} }
func _globrange(val Value) *globrange { return &globrange{val} }
func _globpat(elems ...Value) *globpat { return &globpat{elements{elems}} }
func _word(pos Pos, w string) *word { return &word{valbase{pos},w} }
func _raw(pos Pos, s string) Value {
	if s == "" { return &valbase{pos} } else { return &raw{valbase{pos},s} }
}

func makeDate(pos Pos, s time.Time) *Date  { return &Date{datetime{valbase{pos},s}} }
func makeTime(pos Pos, t time.Time) *Time  { return &Time{datetime{valbase{pos},t}} }
func makeUrl(pos Pos, s *neturl.URL) *url {
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

func list_t[T Value](ii ...T) *list {
    var elems []Value
    for _, i := range ii { elems = append(elems, i) }
    return &list{elements{elems}}
}

func makePair(k, v Value) (p *pair) { return &pair{k, v} }
func makePath(segments ...Value) *path { return &path{elements{segments}} }
func makePunct(pos Pos, t token) *punct { return &punct{valbase{pos},t} }
func makePercpat(pos Pos, prefix, suffix Value) *percpat {
    if prefix == nil { prefix = &valbase{pos} }
    if suffix == nil { suffix = &valbase{pos} }
    return &percpat{valbase{pos},prefix,suffix}
}
func makeDelegate(pos Pos, tok token, obj Value, opts []Value, args ...Value) Value {
    return &delegate{valbase{pos}, tok, obj, opts, args}
}
func makeClosure(pos Pos, tok token, obj Value, opts []Value, args ...Value) Value {
    return &closure{delegate{valbase{pos}, tok, obj, opts, args}}
}

func Make(pos Pos, in any) (out Value) {
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
func MakeAll(pos Pos, in... any) (out []Value) {
    for _, v := range in {
        // TODO: position for each element
        out = append(out, Make(pos,v))
    }
    return
}

func ParseBinary(pos Pos, s string) *binary {
    if strings.HasPrefix(s, "0b") || strings.HasPrefix(s, "0B") {
        s = s[2:]
    }
    if i, e := strconv.ParseInt(s, 2, 64); e == nil {
        return _binary(pos,i)
    } else {
        panic(e)
    }
}

func ParseOctal(pos Pos, s string) *octal {
    if strings.HasPrefix(s, "0") {
        s = s[1:]
    }
    if i, e := strconv.ParseInt(s, 8, 64); e == nil {
        return _octal(pos,i)
    } else {
        panic(e)
    }
}

func ParseDecimal(pos Pos, s string) *decimal {
    if i, e := strconv.ParseInt(s, 10, 64); e == nil {
        return _decimal(pos,i)
    } else {
        panic(e)
    }
}

func ParseHexadecimal(pos Pos, s string) *hexadecimal {
    if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
        s = s[2:]
    }
    if i, e := strconv.ParseInt(s, 16, 64); e == nil {
        return _hexadecimal(pos,i)
    } else {
        panic(e)
    }
}

func parseFloat(pos Pos, s string) *float {
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

func ParseDateTime(pos Pos, s string) Value {
    if t, e := parseDateTime(s); e == nil {
        return &datetime{valbase{pos},t}
    } else {
        panic(e)
    }
}

func ParseDate(pos Pos, s string) *Date {
    if t, e := parseDate(s); e == nil {
        return makeDate(pos,t)
    } else {
        panic(e)
    }
}

func ParseTime(pos Pos, s string) *Time {
    if t, e := parseTime(s); e == nil {
        return makeTime(pos,t)
    } else {
        panic(e)
    }
}

func ParseURL(pos Pos, s string) *url {
    if u, e := neturl.Parse(s); e == nil {
        return makeUrl(pos,u)
    } else {
        panic(e)
    }
}

// NOTE: evokeTraceots is for debugging call trace, if this finally goes into a formal
//       feature, it should need a sync-lock protection.
var evokeTraceots string

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
func (p *evocation) do(ctx Context, op any) (_ any) {
    switch t := op.(type) { // case keep_autos: return false
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
            debug(pc(ctx,p.x), "%s : %s", p.x, ts(p.x), trace{})
        }

    case get_position:
		// Try extracting a fat Position first via our escape hatch (*xloc)
        if p.x != nil {
            if ep, ok := p.x.(interface{ Position() Position }); ok {
                if pos := ep.Position(); pos.valid() { return pos }
            }
        }
        
        // Fallback to compact Pos
        var pos Pos
        if p.x != nil { pos = p.x.Pos() }
        if !pos.IsValid() && p.a != nil && p.a[0] != nil { pos = p.a[0].Pos() }
        if !pos.IsValid() && p.o != nil && p.o[0] != nil { pos = p.o[0].Pos() }
        if  pos.IsValid()  {  return pos  }
    }
    return p.automatic.do(ctx, op)
}

type opts struct{ vals []Value }
type opt  struct{ Value }

func call(ctx Context, name string, o []Value, a ...Value) (res Value) {
    if v := _universe(ctx).lookup(name); v != nil { res = evoke(ctx, v, o, a) }
    return
}

var         name_prefix = regexp.MustCompile(`^((android|darwin|linux|bsd|ios|windows|mingw|[^~]+)~)(.+)$`)
var illegal_name_prefix = regexp.MustCompile(`^use\.(android|darwin|linux|bsd|ios|windows|mingw|[^~]+)~`)

// A scope maintains a set of objects;
// TODO: remote scope struct, use scopeContext instead
type scope struct {
	mutex sync.Mutex
	elems map[string]object
	project *project
	outer *scope
	comment string
}

func newscope(outer *scope, owner *project, c string) (s *scope) {
	return &scope{outer:outer, project:owner, comment:c, elems:make(map[string]object)}
}

func (s *scope) has_outer(outer *scope) bool {
	return s.outer != nil && (s.outer == outer || s.outer.has_outer(outer))
}

func (s *scope) copyElems() (result map[string]object) {
	s.mutex.Lock(); defer s.mutex.Unlock()
	result = make(map[string]object, len(s.elems))
	for k, o := range s.elems { result[k] = o }
	return
}

func (s *scope) estr() (res string) {
	for _, o := range s.elems {
		if res != "" { res += " " }
		res += fmt.Sprintf("%v", o)
	}
	return
}

// Names returns the scope's element names in sorted order.
func (s *scope) names() []string {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	var i = 0
	var names = make([]string, len(s.elems))
	for name := range s.elems {
		names[i] = name
		i++
	}
	sort.Strings(names)
	return names
}

// Lookup returns the object in scope s with the given name if such an
// object exists; otherwise the result is nil.
func (s *scope) Lookup(name string) object {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	return s.lookup(name)
}
func (s *scope) lookup(name string) (obj object) {
	if s.elems != nil { obj, _ = s.elems[name] }
	return
}

// findouter follows the outer chain of scopes starting with s until
// it finds a scope where Lookup(name) returns a non-nil object, and then
// returns that scope and object. If no such scope and object exists, the
// result is (nil, nil).
//
// Note that obj.Outer() may be different from the returned scope if the
// object was inserted into the scope and already had a outer at that
// time (see Insert, below). This can only happen for dot-imported objects
// whose scope is the scope of the package that exported them.
func (s *scope) find(name string) (res *scope, obj object) {
	if obj = s.lookup(name) ; obj != nil {
		return s,obj
	} else if  s.outer != nil  {
		return s.outer.find(name)
	}
	return
}

func (s *scope) finddef(name string) (d *def) {
	if _, o := s.find(name) ; o != nil { d, _ = o.(*def) }
	return
}

func (s *scope) resolve(name string) (obj object) {
	if false { s.mutex.Lock() ; defer s.mutex.Unlock() }
	_, obj = s.find(name)
	return
}

// Insert attempts to insert an object obj into scope s.
// If s already contains an alternative object alt with
// the same name, Insert leaves s unchanged and returns alt.
// Otherwise it inserts obj, sets the object's outer scope
// if not already set, and returns nil.
func (s *scope) insert(ctx Context, obj object) object {
	s.mutex.Lock(); defer s.mutex.Unlock()

    var ic *ident_ctx
    if ic, ctx = identity(ctx); ic.nil > 0 {
		debug(pc(ctx,obj), "no ident: %v", obj)
	}

	if name := ident(ctx, obj); name == "" {
		debug(pc(ctx,obj), "no ident: %v", obj)
		return nil
	} else if alt := s.elems[name]; alt != nil {
		return alt
	} else {
		s.replace(ctx, name, obj)
		return nil
	}
}

func (s *scope) replace(ctx Context, name string, obj object) {
	switch o := obj.(type) {
	case interface { setscope(string, *scope) }:
		o.setscope(name, s)
	}
	s.elems[name] = obj
}

// WriteTo writes a string representation of the scope to w,
// with the scope elements sorted by name.
// The level of indentation is controlled by n >= 0, with
// n == 0 for no indentation.
func (s *scope) WriteTo(w io.Writer, n int) {
	s.mutex.Lock() ; defer s.mutex.Unlock()

	const ind = ".  "

	var indn  = strings.Repeat(ind, n)
	var indn1 = indn + ind

	fmt.Fprintf(w, "%s%s scope %p {", indn, s.comment, s)

	if len(s.elems) == 0 {
		fmt.Fprintf(w, "}")
		return
	}

	fmt.Fprintln(w)

	for _, name := range s.names() {
		fmt.Fprintf(w, "%s%s\n", indn1, s.elems[name])
	}

	fmt.Fprintf(w, "%s}", indn)
}

// String returns a string representation of the scope, for debugging.
func (s *scope) String() string { return fmt.Sprintf("{=scope %s}", s.string()) }
func (s *scope) string() string {
	var buf bytes.Buffer
	if s.outer != nil {
		if false {
			fmt.Fprintf(&buf, "%s → %s", s.outer.string(), s.comment)
		} else {
			fmt.Fprintf(&buf, "%s ← %s", s.comment, s.outer.string())
		}
	} else {
		fmt.Fprintf(&buf, "%s", s.comment)
	}
	return buf.String()
}

func (s *scope) projectname(ctx Context, name string, project *project) (p *project, a object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	if a = s.elems[name] ; a == nil {
		p = project
		s.replace(ctx, name, p)
	}
	return
}

func (s *scope) builtin(ctx Context, name string, f reflect.Type) (res *builtin, a object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()
	if a = s.elems[name] ; a == nil {
		res = &builtin{knownobject{objbase{scope:s}, name}, f}
		s.replace(ctx, name, res)
	}
	return
}

func (s *scope) _auto(ctx Context, name string) (a *auto, o object) {
	s.mutex.Lock() ; defer s.mutex.Unlock()

	var y bool

	if o, y = s.elems[name]; y && o == nil {
		delete(s.elems, name)
		y = false
	}

	if !y {
		a = &auto{knownobject{objbase{valbase{_pos(ctx)},s}, name}}
		s.replace(ctx, name, a)
	}
	return
}

func (s *scope) auto(ctx Context, name string) (a *auto) {
	var y bool
	var o object
	if a, o = s._auto(ctx, name); o != nil {
		if a, y = o.(*auto); !y {
			debug(ctx, "name already taken (%s)", typeof(o))
		}
	}
	return
}

func (s *scope) alias(ctx Context, o object, alias ...string) {
	for _, a := range alias { s.elems[a] = o }
}

func (s *scope) _def(ctx Context, o origin, id any, vals ...Value) (d *def, isNew bool) {
	var name string = ident_any(ctx, id)
	if name == "" {
		debug(ctx, "empty name: %s: `%v`", typeof(id), id, callstack{num:16}, trace{})
	}
	if checkpoints { if illegal_name_prefix.MatchString(name) {
		debug(ctx, "illegal name: %v", name, callstack{num:16}, trace{})
	}}

	s.mutex.Lock()

	if a, y := s.elems[name]; !y || a == nil {
		d = new(def)
		d.name, d.pos, d.scope = name, _pos(ctx), s
		s.elems[name] = d
		isNew = true
	} else {
		d, _ = a.(*def)
	}

	s.mutex.Unlock()
	
	if o != defInvalid && o != d.o {
		if d.o == defInvalid { d.o = o } else {
			debug(pc(ctx, d), "%v: conflicts origin: %v <> %v", id, d.o, o, trace{})
		}
	}
	
	if vals != nil { d.set(ctx, ease(ctx, vals)) }
	return
}

func (s *scope) def(ctx Context, o origin, ident any, vals ...Value) (d *def) {
	d, _ = s._def(ctx, o, ident, vals...)
	return
}

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
    out.pos, out.scope, out.name = _pos(ctx), _scope(ctx), name

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
                debug(pc(ctx,a), "empty name: %v", p.key, trace{})
                return
            }

            if ac.params != nil {
                if _, y = ac.params[a.name]; !y {
                    var keys = reflect.ValueOf(ac.params).MapKeys()
                    debug(pc(ctx,a), "unknown arg#%d: %v ; known: %v", i, p, keys, trace{})
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
            debug(ac, "arg '%s' not set", a.name, trace{})
            return
        }

        if d, y := ac.defs[a.name]; !y || d == nil {
            debug(ac, "arg '%s' not set", a.name, trace{})
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
func (a *auto) def(ctx Context) *def { return auto_find(ctx, a.name) }
func (a *auto) set(ctx Context, o origin, value Value, app ...Value) {
    if value == nil && app != nil {
        if d := auto_find(ctx, a.name); d != nil {
            d.value = ease(ctx, append(merge(d.value), app...))
            if false { d.pos = a.pos }
            return
        }
    }

    d, _ := auto_set(ctx, o, a.name, ease(ctx, append(merge(value), app...)))
    if d != nil {
        if false { d.pos = a.pos }
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
func (d *def) Pos() (pos Pos) {
	if pos = d.pos; !pos.IsValid() && d.value != nil { pos = d.value.Pos() }
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
            debug(pc(ctx,d), "%v: %v → %v", d.name, d.o, o, trace{})
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
		debug(pc(pc(ctx,value),d.value),
			_f("duplicated %v %v → %v %v", d.o, d, value, app),
			callstack{num:16}, trace{})
	}
    if value == d.value && len(app) == 0 { return }

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
    } else if d.pos.IsValid() {
        d.value = _null(d.pos)
    } else {
        d.value = _null(_pos(ctx))
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

    var pos = value.Pos()
    if !pos.IsValid() { pos = _pos(ctx) }
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
func (p *undetermined) Pos() Pos { return p.identifier.Pos() }
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
func (p *rule_ctx) cast(t reflect.Type) Context { return icast(p, t) }
func (p *rule_ctx) inner() Context { return p.Context }
func (p *rule_ctx) Pos() Pos { return p.rule.Pos() }
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
func (p *rule) Pos() (_ Pos) {
    if pos := p.target.Pos(); pos.IsValid() {
        return pos
    }
    for _, prog := range p.program {
        if prog.pos.IsValid() {
            return prog.pos
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
func (p *rule) execute(ctx Context, a ...Value) (res []Value) {
    if patterned(ctx, p.target) {
        debug(ctx, "execute pattern entry: %v", p.target, trace{})
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

const configuration_sm = "configuration.sm"
const pathSepByte = filepath.Separator
const pathSep = string(pathSepByte)

type _filemap struct { project *project ; patts, paths []Value }
func (p *_filemap) String() string { return fmt.Sprintf("%s", p.patts) }

func filemap_str(t *[]filemap) (s string) {
	for i, t := range *t { if 0 < i { s += " " }; s += t.String() }
	return "["+s+"]"
}

type filemap struct { *_filemap ; pattern Value }
func (p *filemap) ts(t string) string { return fmt.Sprintf("{=%s %v}", t, ts(p.pattern)) }
func (p *filemap) String() (s string) {
    if p.pattern == nil {
        s = p._filemap.String()
    } else {
        s = p.pattern.String()
    }
    return
}

func (p *filemap) patterns(ctx Context) (pats []Value) {
    var patts = []Value{ p.pattern }
    if patts[0] == nil { patts = p.patts }

    for _, pattern := range patts {
        // NOTE it may preserve closure patterns after this expand
        pat := expand(ctx,pattern)
        pats = append(pats, merge(pat)...)
    }
    return
}

// match split filename into list and match each part with the pattern correspondingly.
func (p *filemap) match(ctx Context, val Value) (_ bool, _ Value, _ string) {
    // TODO: escape file matching for 'String' and "strcomp" values
    for _, pat := range p.patterns(ctx) {
        if matched, name := p._match(ctx, pat, val); matched {
            return matched, pat, name
        }
    }
    return
}

func (p *filemap) _match(ctx Context, pat, val Value) (matched bool, name string) {
    // TODO: escape file matching for 'String' and "strcomp" values
    var res any
    matched, res, _, _ = match(ctx, pat, val)

    if false && !matched && !(isNone(pat) || isNull(pat)) {
        var str string // NOOP
        if n := strings.Index(str, pathSep); n < 0 { return }

        // NOTE: Dealing with these files:
        //     files (
        //         (foo.c) ⇒ $(srcdir)/sub/dir
        //         (sub/dir/foo.c) ⇒ $(srcdir)
        //     )
        for _, p := range p.paths { // FIXME: performance, operate on p.(*path) instead
            if _, ok := p.(*path); !ok { continue } // NOTE: only work with paths to improve performance
            var ps = __string(ctx, p)
            for i := strings.LastIndex(ps, pathSep); -1 <= i; {
                var ( prefix = ps[i+1:]; l = len(prefix) ) // NOTE: -1 <= i < len(ps)
                if has := strings.HasPrefix(str, prefix) && str[l] == '/'; has {
                    if matched, _, _, _ = match(ctx, pat, &raw{valbase{pat.Pos()}, str[len(prefix)+1:]}); matched { break }
                }
                if 0 < i { i = strings.LastIndex(ps[:i], pathSep) } else { break }
            }
        }
    }

    if res == nil {
        // okay
    } else if s, y := res.(string); y {
        name = s
    } else if a, y := res.([]string); y {
        name = joinpath(a...)
    } else {
        debug(ctx, "unexpected result: %v", ts(res), trace{})
    }
    return // NOTE: also `globMatchFile(ctx, pat, str, true)`
}

func (p *filemap) stat(ctx Context, name string) (res *file) {
    var patts = p.patts
    if len(patts) == 0 {
        debug(ctx, "no map patterns: %v", p, trace{})
    }

    if len(p.paths) == 0 {
        for _, pat := range p.patts {
            if f, y := pat.(*file); y && ident(ctx,f) == name { return f }
        }
        for i, pat := range p.patts {
            if f, y := pat.(*file); y {
                info(ctx, "pattern %d. %v %s (exists=%v)", i, f, f.fullname(), f.exists())
            } else {
                info(ctx, "pattern %d. %v", i, ts(pat))
            }
        }
        debug(ctx, "%s → %v", name, p.patts, trace{})
    }

    for _, path := range p.paths {
        if isNull(path) {
            debug(ctx, _f("nil path: name=%s",  name), _f("nil path: %v", p), trace{})
        } else if isNone(path) {
            debug(ctx, _f("nil path: name=%s",  name), _f("nil path: %v", p))
            continue
        }

        var dir, sub string

        if sub = __string(ctx, path); sub == "" {
            if false {
                debug(ctx, "empty filemap path: %v, patterns=%v", path, patts, trace{})
            }
            return
        }

        if s := filepath.Clean(sub); sub != s { sub = s }

        if filepath.IsAbs(sub) {      // 'sub' is abs
            if filepath.IsAbs(name) { // 'name' is abs too
                if s := sub+pathSep; strings.HasPrefix(name, s) { // 'name' should have 'sub' prefix
                    name = strings.TrimPrefix(name, s)
                } else {
                    continue
                }
            }
        } else if !filepath.IsAbs(name) {
            // dir = filepath.Clean(base)
        }

        if res = _stat(ctx, name, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true}); res != nil { break }

        var pre string // Not used!
        if filepath.IsAbs(sub) {
            if pre == "" { // Fullmatch!
                // For example of:
                //   xxx.c  <->  (*.c => /path/to/source)
                // Become:
                //   /path/to/source  ""  xxx.c
                res = _stat(ctx, name, stat_dir{sub}, stat_nonexist{true})
            } else if strings.HasSuffix(sub, pathSep+pre) {
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => /path/to/source/foo/bar)
                // Become:
                //   /path/to/source  foo/bar  xxx.c
                s := strings.TrimSuffix(sub, pathSep+pre)
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{pre}, stat_dir{s}, stat_nonexist{true})
            } else if false { // This is wrong, only base name matched!!
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => /path/to/source)
                // Become:
                //   /path/to/source  foo/bar  xxx.c
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{pre}, stat_dir{sub}, stat_nonexist{true})
            }
        } else {
            if pre == "" { // Fullmatch!
                // For example of:
                //   xxx.c  <->  (*.c => source)
                // Become:
                //   <p.absPath>  source  xxx.c
                res = _stat(ctx, name, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true})
            } else if sub == pre {
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => foo/bar)
                // Become:
                //   <dir>  foo/bar  xxx.c
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{sub}, stat_dir{dir}, stat_nonexist{true})
            } else if strings.HasSuffix(sub, pathSep+pre) {
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => source/foo/bar)
                // Become:
                //   <dir>  source/foo/bar  xxx.c
                s := strings.TrimSuffix(sub, pathSep+pre)
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{pre}, stat_dir{s}, stat_nonexist{true})
            } else if false { // This is wrong, only base name matched!!
                // For example of:
                //   foo/bar/xxx.c  <->  (*.c => source)
                // Become:
                //   <dir>  source/foo/bar  xxx.c
                s := filepath.Join(sub, pre)
                n := strings.TrimPrefix(name, pre+pathSep)
                res = _stat(ctx, n, stat_sub{s}, stat_dir{dir}, stat_nonexist{true})
            }
        }
    }
    return
}

type project_ctx struct{ Context ; p *project }
func (c project_ctx) cast(t reflect.Type) Context { return icast(c,t) }
func (c project_ctx) inner() Context { return c.Context }
func (c project_ctx) do(ctx Context, op any) any {
    switch op.(type) {
    case get_project: return c.p
    }
    return c.Context.do(ctx, op)
}

type shadowCache struct {
	val  string
	trie *valcache
}
type project_ext struct{ *plugin.Plugin }
type project struct {
    *scope

    pos Pos

    bases []*project

    use *uselist

    configure *project // .configure project
    configuration *file // configuration.sm if saved or loaded

    absPath string
    tmpPath string
    name    string
    rel     string // path segment relative to the workBaseDir
    spec    string // relative to search-paths as a specification

    filemap valcache
    entries valcache

	shadowsMu sync.RWMutex
	shadows map[Value]*shadowCache

    patterns []*rule // order is important
    configs  []*def // configure entries
    main entry

    ext project_ext
    opt project_opts
}
func (_ *project) kind() Kind { return KindObject|KindKnownObject|KindProject }
func (p *project) Pos() Pos { return p.pos }
func (p *project) String() string { return "{=project "+p.name+"}" }
func (p *project) owner() *project { return p.scope.project }
func (p *project) stencil(_ Context, stems []string) (Value, []string) { return p, stems }

// shadow builds an isolated valcache for the dynamic payload,
// shadowing stale results by tracking the evaluated string value.
func (p *project) shadow(ctx Context, payload any) *valcache {
	var cacheKey Value
	switch t := payload.(type) {
	case filemap: cacheKey = t.pattern
	case *rule: cacheKey = t.target
	}
	if cacheKey == nil { return nil }

	// 1. Fully evaluate the closure to get the CURRENT files
	compiledVal := unloc(expand(_final(ctx), cacheKey))
	if compiledVal == nil { return nil }

	// 2. Stringify the result to detect reassignments (Stale Cache Detection)
	compiledStr := __string(ctx, compiledVal)

	// 3. O(1) Memoization Hit (Thread-Safe Read)
	p.shadowsMu.RLock()
	if sc, ok := p.shadows[cacheKey]; ok && sc.val == compiledStr {
		p.shadowsMu.RUnlock()
		return sc.trie
	}
	p.shadowsMu.RUnlock()

	// 4. Cache Miss / Stale Cache: The variable changed! Build a NEW isolated shadow trie.
	shadow := &valcache{}

	inject := func(elem Value) {
		var boundPayload any = payload
		switch t := payload.(type) {
		case filemap: boundPayload = filemap{_filemap: t._filemap, pattern: elem}
		case *rule:
			r := *t
			r.target = elem
			boundPayload = &r
		}

		if nodes := hit(cache_t{ctx}, shadow, elem); len(nodes) > 0 {
			found := false
			for _, a := range nodes[0].a {
				switch existing := a.(type) {
				case filemap:
					if bp, ok := boundPayload.(filemap); ok {
						if existing._filemap == bp._filemap && __string(ctx, existing.pattern) == __string(ctx, bp.pattern) { found = true }
					}
				case *rule:
					if bp, ok := boundPayload.(*rule); ok {
						if __string(ctx, existing.target) == __string(ctx, bp.target) { found = true }
					}
				}
			}
			if !found { nodes[0].a = append(nodes[0].a, boundPayload) }
		}
	}

	// 5. Inject the NEW current files into the isolated shadow trie
	if list, ok := compiledVal.(*list); ok {
		for _, elem := range list.elems { inject(elem) }
	} else {
		inject(compiledVal)
	}

	// 6. Save the new shadow trie (Thread-Safe Write)
	p.shadowsMu.Lock()
	if p.shadows == nil { p.shadows = make(map[Value]*shadowCache) }
	p.shadows[cacheKey] = &shadowCache{val: compiledStr, trie: shadow}
	p.shadowsMu.Unlock()
	
	return shadow
}

type self struct { *project }
func (p self) kind() Kind { return p.project.kind()|KindSelf }
func (p self) String() string { return "{=self "+p.name+"}" }

func findfile(ctx Context, s string, ps ...*project) (_ *file) {
    if len(ps) == 0 { ps = append(ps, _project(ctx)) }
    for _, p := range ps { if f := p.file(ctx, s); f != nil { return f } }
    return
}

func select_file_1(ctx Context, m matched_filemap) (res *file) {
    defer func() { if res != nil { res.filemap = &m.filemap } } ()

    if m.paths == nil {
        if res, _ = m.pattern.(*file); res != nil {
            return
        } else {
            var d = _project(ctx).absPath
            return _stat(ctx, m.value, stat_dir{d}, stat_nonexist{true})
        }
    }

    var fs []*file

    for _, v := range m.paths {
        if t := expand(_final(ctx),v); t != nil {
            if d := __string(ctx, t); d != "" {
                if f := _stat(ctx, m.value, stat_dir{d}, stat_nonexist{true}); f != nil {
                    fs = append(fs, f)
                } else {
                    debug(ctx, "%s ⇒ %v → %v → ''", m.value, v, t, trace{})
                }
            } else if false {
                debug(ctx, "%s ⇒ %v → %v → ''", m.value, v, t, trace{})
            }
        } else {
            debug(ctx, "%s ⇒ %v", m.value, v, trace{})
        }
    }

    for _, f := range fs { if f.exists() { return f } }
    if 0 < len(fs) { res = fs[0] }
    return
}

func select_files(ctx Context, m []matched_filemap) (res []*file) {
    for _, m := range m {
        if f := select_file_1(ctx, m); f != nil {
            res = append(res, f)
        }
    }
    return
}

func select_file(ctx Context, m []matched_filemap) (res *file) {
    if a := select_files(ctx, m); 0 < len(a) {
        if res = a[0] ; !res.exists() {
            for _, f := range a { if f.exists() { return f } }
        }
    }
    return
}

func (p *project) file(ctx Context, a any) *file {
    return select_file(ctx, unmap_files(ctx, p, a, nil))
}

func (p *project) tempdir(ctx Context) (d *def, s string) {
	for _, t := range []string{"outtmp", ".tmp", "CTD"} {
		if d = p.resolveDef(ctx, t); d != nil { break }
	}

	if d == nil {
		debug(ctx, "%v: tmp is not defined", p, trace{})
		return
	}

	// =========================================================
	// CRITICAL FIX: Polymorphic Macro Expansion
	// =========================================================
	// 1. Create a closure context locked to the active project
	// 2. Stringify the dynamically evaluated result
	s = filepath.Clean(__string(closure_with(ctx, p), d.value))

	if false && checkpoints { tempdir_check(ctx, p, d, s) }
	return
}

func (p *project) tempfile(ctx Context, name string) (f *file) {
    var t, d = p.tempdir(ctx)
    switch d {
    case "", "/":
        debug(ctx,
			_f("%v: %s: tempdir is illegal: %v → '%s', %s", p.name, name, t, __string(ctx, t), d),
			_f("%v", p.resolveDef(ctx, "outtmp")),
			_f("%v", p.resolveDef(ctx, "target.tmp")),
			_f("%v", p.resolveDef(ctx, "target.out")),
			_f("%v", p.resolveDef(ctx, "target.triple")),
			_f("%v", p.resolveDef(ctx, "rel.remnant")),
			_f("%v", p.resolveDef(ctx, "rel.chop")),
			_f("%v", p.resolveDef(ctx, "variant.tag")),
			trace{})
    }

    if f = _stat(ctx, name, stat_dir{d}, stat_nonexist{true}); f == nil {
        debug(ctx, "%v: not a file: %v : %v", p, name, d, trace{})
    }

    if false && checkpoints { tempfile_check(ctx, p, name, d, f) }
    return
}

func (p *project) configuration_sm(ctx Context) (f *file) {
    if f = p.tempfile(ctx, configuration_sm); f == nil {
        debug(ctx, "%v: no file %s", p, configuration_sm, trace{})
    }
    return
}

func project_entry(c Context, s any, a ...bool) entry { return _project(c).entry(c, s, a...) }
func project_resolve(c Context, s string) object { return _project(c).resolve(c, s) }

func (p *project) resolveDef(ctx Context, name string) (res *def) {
    if o := p.resolve(ctx, name); o != nil { res, _ = o.(*def) }
    return
}

func (p *project) resolve(ctx Context, name string) (obj object) {
    if _, obj = p.find(name); obj != nil { return }

    if p.ext.Plugin != nil {
        if sym, e := p.ext.Lookup(name); e == nil && sym != nil {
            debug(ctx, "TODO: convert ext symbol: %v: %s", name, typeof(sym), trace{})
        }
    }

    for _, base := range p.bases {
        if base.has_base(p) {
            debug(ctx, "recursive derivation: %v ⇔ %v", ts(p), ts(base), trace{})
        }
        if obj = base.resolve(ctx, name); obj != nil { return }
    }

    if p.configure != nil && p.configure != p {
        return p.configure.resolve(ctx, name)
    }
    return
}

func (p *project) _entries(ctx Context, name any, _b ...bool) (entries []entry) {
    entries = unmap_entries(ctx, p, name, nil)

    if false && p.configure != nil && is_configurecontext(ctx) {
        entries = append(entries, p.configure._entries(ctx, name, true)...)
    }

    var resolveBases = __t(_b...)
    if resolveBases || entries == nil {
        for _, base := range p.bases {
            if t := base._entries(ctx, name, resolveBases); t != nil {
                entries = append(entries, t...)
                break
            }
        }
    }

    if false && entries == nil { // NOTE: this would be SLOW
        for _, u := range p.use.list {
            t := u.project._entries(ctx, name, resolveBases)
            if t != nil { entries = append(entries, t...); break }
        }
    }
    return
}

func (p *project) entry(c Context, name any, a ...bool) (_ entry) {
    var entries = p._entries(c, name, a...)
    if n := len(entries); 0 < n {
        if 1 < n { debug(c, "%v : %d entries", name, n, trace{}) }
        return entries[0]
    }
    return
}

func (p *project) resolvePatterns(ctx Context, v Value, s string) (res []*stemmed_rule) {
    var t1, t2 time.Time

    defer func(t0 time.Time) {
        var t = time.Now()
        if d := t.Sub(t0); d > 1*time.Second {
            var ( d1 = t1.Sub(t0) ; d2 = t2.Sub(t1) ; d3 = t.Sub(t2) ; n int )
            var a = auto_get(ctx, "@")
            for sc := _stemmed(ctx); sc != nil; n += 1 {
                if c := inner(sc); c != nil { sc = _stemmed(c) } else { break }
            }

            var pos = _position(ctx)
            prompt(ctx, "%v: slow: %v: %v, %v %v %v", pos, p, d, d1, d2, d3)
            prompt(ctx, "%v: slow: %v: %v: %v %v, %d nests", pos, p, a, v, p.patterns, n)

            for _, pat := range p.patterns {
                var pt = pat.target
                var pa = pat.arged
                var full, r, _, stems = match(ctx, pt, &raw{valbase{pt.Pos()}, s})
                var m = joinp(ctx, r)
                prompt(ctx, "%v: slow: %v%v: %v: %v %v %v, %v ; %v", pos, pt, pa, s, full, r, stems, m)
            }
            debug(ctx, "slow")
        }
    } (time.Now())

    if res, t1, t2 = p.resolvePatterns123(ctx, v, s); false && len(res) > 0 {
        for _, t := range res {
            if f, _ := to_file(t.target); f != nil {
                f.pos = t.Pos()
            } else if f = p.file(ctx, s); f != nil {
                f.pos = t.Pos()
                t.target = f
            }
        }
    }
    return
}

func (p *project) resolvePatterns123(ctx Context, v Value, s string) (res []*stemmed_rule, t1, t2 time.Time) {
    if true  { res = append(res, p.resolvePatterns1(ctx, v, s)...) } ; t1 = time.Now()
    if true  { res = append(res, p.resolvePatterns2(ctx, v, s)...) } ; t2 = time.Now()
    if false { res = append(res, p.resolvePatterns3(ctx, v, s)...)/* heavy work, VERY SLOW! */ }
    return
}

func (p *project) resolvePatterns1(ctx Context, val Value, s string) (res []*stemmed_rule) {
    defer func(t0 time.Time) {
        if d := time.Now().Sub(t0); d > 1*time.Second {
            debug(ctx, "%v: slow: %v %v", _position(ctx), val, d)
        }
    } (time.Now())

ForPatterns:
    for _, pat := range p.patterns {
        if full, r, _, stems := match(ctx, pat.target, &raw{valbase{pat.target.Pos()}, s}); full {
            var m = joinp(ctx, r)

            if true {
                for sc := _stemmed(ctx); sc != nil; { // pattern loop detection
                    if s := __string(ctx, sc.target); s == m { continue ForPatterns }
                    if c := inner(sc); c != nil { sc = _stemmed(c) } else { break }
                }
            }

            if pa := pat.arged; len(pa) > 0 {
                var y bool
                var t1 = time.Now()
                var av = xmerge(ctx, pa...)
                var t2 = time.Now()
                for _, a := range av { if y, _, _, _ = match(ctx, a, &raw{valbase{a.Pos()}, s}); y { break } }

                var t3 = time.Now()
                if d := t3.Sub(t1); d > 1*time.Second {
                    var ( d2 = t2.Sub(t1) ; d3 = t3.Sub(t2) )
                    var ( p = _position(ctx) ; pt = pat.target )
                    debug(ctx, "%v: slow: %v, %v→%d; %v⇒%v+%v", p, pt, pa, len(av), d, d2, d3)
                }

                if !y { continue ForPatterns }
            }

            res = append(res, &stemmed_rule{pat, val, stems})
        }
    }
    return
}

func (p *project) resolvePatterns2(ctx Context, val Value, s string) (res []*stemmed_rule) {
    for _, base := range p.bases {
        var a, _, _ = base.resolvePatterns123(ctx, val, s)
        res = append(res, a...)
    }
    if p.configure != nil && is_configurecontext(ctx) {
        var a, _, _ = p.configure.resolvePatterns123(ctx, val, s)
        res = append(res, a...)
    }
    return
}

func (p *project) resolvePatterns3(ctx Context, val Value, s string) (res []*stemmed_rule) {
    for _, use := range p.use.list {
        var a, _, _ = use.project.resolvePatterns123(ctx, val, s)
        res = append(res, a...)
    }
    return
}

func (p *project) family() (res []*project) {
    res = append(res, p)
    for _, base := range p.bases {
        res = append(res, base.family()...)
    }
    return
}

func (p *project) _isa(s string) (_ bool) {
    for _, base := range p.bases {
        if base.name == s || base._isa(s) { return true }
    }
    return
}

func (p *project) isa(proj *project) (_ bool) {
    for _, base := range p.bases {
        if base == proj || base.isa(proj) { return true }
    }
    return
}

func (p *project) has_base(proj *project) (_ bool) {
    for _, base := range p.bases {
        if base == proj || base.has_base(proj) { return true }
    }
    return
}

func (p *project) has_loaded(ctx Context, proj *project, traveUseLoop bool) (rp *project, res, isb bool) {
    if u := _universe(ctx) ; u.checkLoadGraph || !u.fastMode {
        rp, res, isb, _ = p.has_loaded_recur(ctx, p, proj, 1, traveUseLoop)
    }
    return
}

func (p *project) has_loaded_recur(ctx Context, top, proj *project, depth int, traveUseLoop bool) (rp *project, res, isb bool, err error) {
    if depth > 1 && top == p && true {
        err = fmt.Errorf("loop '%v' (depth=%d)", p.loop_load_path(), depth)
        debug(ctx, "%v: %v", p, err, trace{})
    } else if depth > 128 {
        err = fmt.Errorf("exceeds maximum base depth (%d) (start=%v, target=%v)", depth, top, proj)
        debug(ctx, "%v: %v", p, err)
        debug(ctx, "start: %v", top)
        debug(ctx, "target: %v", proj, trace{})
    }
    for _, base := range p.bases {
        if isb = base == proj; isb { return }
        if rp, res, isb, err = base.has_loaded_recur(ctx, top, proj, depth+1, traveUseLoop); err != nil {
            return
        } else if res || isb { rp = base ; return }
    }
    for _, use := range /*p.loads*/p.use.list {
        var imp = use.project
        if imp == top && !traveUseLoop {
            s := top.loop_load_path()
            err = fmt.Errorf("loop `%v`", s)
            debug(ctx, "start: %v", top)
            debug(ctx, "stop: %v", proj)
            debug(ctx, "%v: %v", p, err, trace{})
        }
        if res = imp == proj; res { rp = imp; return }
        if rp, res, res, err = imp.has_loaded_recur(ctx, top, proj, depth+1, traveUseLoop); err != nil {
            return
        } else if res { rp = imp; return }
    }
    rp = p
    return
}

func (p *project) loop_base_path(ctx Context, _p *project, s string) (_ string) {
    if s == "" { s = p.name }
    for _, base := range p.bases {
        if t := s + " → " + base.name; base == _p {
            return t
        } else if t = base.loop_base_path(ctx, _p, t); t != "" {
            return t
        }
    }
    return
}

func (p *project) loop_load_path() (s string) { return p.loop_load_recur(p) }
func (p *project) loop_load_recur(top *project) (s string) {
    for _, use := range /*p.loads*/p.use.list {
        var imp = use.project
        if imp == top {
            if p != top { s = "⇢" }
            s += fmt.Sprintf("(%s)⇢(%s)", p.spec, imp.spec)
            break
        }
        if t := imp.loop_load_recur(top); t != "" {
            if p != top { s = "⇢" }
            s += fmt.Sprintf("(%s)%s", p.spec, t)
            break
        }
    }
    return
}

func (p *project) isUsing(usee *project) (res bool) {
    for _, use := range p.use.list {
        if res = use.project == usee; res { break  }
        if res = use.project.isUsing(usee); res { break }
    }
    return
}

func (p *project) isUsingDirectly(proj *project) (res bool) {
    for _, u := range p.use.list {
        if res = u.project == proj; res { break }
    }
    return
}

func (p *project) usees(bases, basesRecur, useeRecur, pre bool) (res []*project) {
    if p.opt.traveUseLoop { return }
    if bases {
        for _, base := range p.bases {
            res = append(res, base.usees(basesRecur, basesRecur, useeRecur, pre)...)
        }
    }
    for _, u := range p.use.list {
        if pre { res = append(res, u.project) }
        if useeRecur {
            for _, u := range u.project.usees(bases && basesRecur, basesRecur, true, pre) {
                if !p.isUsingDirectly(u) { res = append(res, u) }
            }
        }
        if !pre { res = append(res, u.project) }
    }
    return
}

// Note: this is okay not using an atomic value, because
// chdirMutex can serve to protect the whole timeframe.
var chdirMutex = new(sync.Mutex)

func lockCD(dir string, dura time.Duration) error {
    // Protect the work directory, `chdirMutex` ensures that
    // there's only one timer being counting to avoid work
    // directory being changed before the deadline.
    chdirMutex.Lock()
    go func() {
        if dura > 0 { time.Sleep(dura) }
        chdirMutex.Unlock()
    } ()
    return os.Chdir(dir)
}

type dirtyOpts struct {
    general_opts
    verboseUpdated  bool "verbose-updated"  // vu,
    verboseOutdated bool "verbose-outdated" // vo,
    checksum bool "checksum" // cs,crc,
    silent   bool "silent" // s,
    pats []Value
}

type default_value struct{ v Value }
type exe_res       struct{ v Value }
type get_program   struct{}

type is_ordered_prereq struct{}
type is_prerequisite   struct{}

func _execution(c Context) *execution { return cast[*execution](c) }

type execution_lang struct{}
type missing_file struct{ file string }

type interpret struct { name string ; args []Value }
type execution struct {
    automatic
    sync.Mutex
    sync.WaitGroup

    by dirtyOpts

    proj *project
    prog *program
    recipes []Value
    language string

    defval   Value
    defers []Value
    values []Value

    prerequisite Value
    _ordered bool

    _env []*pair
    changedWD string

    dirt string // reason of outdated
    start time.Time // start time

    recs map[Value]int

    calleeErrs []error
    calleeErrsM sync.Mutex

    missing []string

    grepping bool
    grepped []Value
    ordered []Value
    targets []Value // all targets def

    countFiles int
    traceLevel int

    interpreted []interpreter
}
func (p *execution) inner() Context { return &p.automatic }
func (p *execution) caller() *execution { return _execution(p.Context) }
func (p *execution) cast(t reflect.Type) Context {
    if t == reflect.TypeOf(p) { return p }
    if t == reflect.TypeOf((*argumented_ctx)(nil)) { return nil }
    if t == reflect.TypeOf((*stemmed_ctx)(nil)) { return nil }
    return p.automatic.cast(t)
}
func (p *execution) do(ctx Context, op any) (res any) {
    switch t := op.(type) {
    case ex_closure: return true
    case get_position:
        if p.prerequisite != nil { return p.prerequisite.Pos() }
        if len(p.recipes) > 0 { return p.recipes[0].Pos() }

    case get_program : return    p.prog
    case get_project : if nil != p.proj { return p.proj }
    case get_scope   : if nil != p.proj { return p.proj.scope }

    case is_ordered_prereq : return p._ordered && p.prerequisite != nil
    case is_prerequisite, evoke_loop_null: return p.prerequisite != nil

    case default_value:  p.defval = t.v; return
    case exe_res:        p.values = append(p.values, t.v); return
    case mark_dirty:     p.dirty_mark(t.a...)//; return
    case act_traverse:   p.traverse(ctx, t.v); return
    case act_traversed:  return p.traversed(ctx,t.v)
    case act_dirt:       return p.dirty(ctx,t.a...)

    case param_name:
        if p.prog != nil {
            if t.i < len(p.prog.params) {
                res = p.prog.params[t.i].name
            }
        }
        return

    case execution_lang:
        return p.language

    case interpret:
        return p.interp(ctx, t.name, t.args)

    case missing_file:
        p.missing = append(p.missing, t.file)

    case property:
        if t&propDirtyOpts != 0 { return &p.by }
    }
    return p.automatic.do(ctx, op)
}

func (p *execution) ts(t string) (s string) {
    s = "{=" + t
    if v := p.prerequisite; v != nil {
        s += " " + v.String()
    }
    s += " " + ts(p.Context) + "}"
	return
}

func (p *execution) aquire() func() { p.Lock() ; return p.Unlock }

func (p *execution) depth() (res int) {
    for c := p.caller(); c != nil; c = c.caller() { res += 1 }
    return
}
func (p *execution) calleeError(err error) {
    if err != nil {
        p.calleeErrsM.Lock()
        p.calleeErrs = append(p.calleeErrs, err)
        p.calleeErrsM.Unlock()
    }
}

// traverse_context is a single thread traverse context, for traversing in a new goroutine,
// a spawned traversal must be used and then merge.
func (p *execution) level(n int) { p.traceLevel += n }
func (p *execution) trace(a ...any) { printIndentDots(p.traceLevel, a...) }
func (p *execution) tracef(s string, a ...any) { printIndentDots(p.traceLevel, fmt.Sprintf(s, a...)) }

func (p *execution) traversed(ctx Context, target Value) []Value {
    if !isTrivial(target) {
        if _, y := to_file(target); y {
            p.addFilesCount(1)
        }
        if truly(ctx, is_ordered_prereq{}) {
            p.ordered = append(p.ordered, target)
            auto_set(ctx, defVoid, "|", _list(p.ordered...))
        } else {
            p.targets = append(p.targets, target)
            auto_set(ctx, defVoid, "^", _list(p.targets...))
            auto_set(ctx, defVoid, "<", p.targets[0])
            auto_set(ctx, defVoid, ">", p.targets[len(p.targets)-1])
        }
    }
    return p.targets
}

func (p *execution) addFilesCount(n int) {
    p.countFiles += n
    if c := p.caller(); c != nil { c.addFilesCount(1) }
}

func (p *execution) env(ctx Context) (env []string, osi int) {
    env = os.Environ()
    osi = len(env)
    for _, p := range p._env {
        var k, v = __string(ctx, p.key), __string(ctx, p.val)
        env = append(env, fmt.Sprintf("%s=%s", k, v)) // strconv.Quote(v)
    }
    return
}

func (p *execution) dirty_mark(vals ...Value) {
    const (
        enableDirtyMark = true
        perUpdatedDep = true
    )
    if !enableDirtyMark {
        // does nothing
    } else if targets := merge(auto_get(p, "@")); len(targets) == 0 {
        // should not happen, but safely ignoring..
    } else if len(vals) == 0 {
        vals = append(vals, targets...)
    } else if true {
        vals = merge(vals...)

        var mat, dup bool
        var opts = &p.by
        for _, t := range targets {
        ForVals:
            for _, val := range vals {
                if eq(p, val, t) {
                    dup = true; continue ForVals
                }
                for _, pat := range opts.pats {
                    if mat, _, _, _ = match(p, pat, val); mat {
                        if perUpdatedDep { updatedDeps(p, t, val) }
                        break ForVals
                    }
                }
            }
            if !perUpdatedDep && mat { updatedDeps(p, t, vals...) }
            if !dup { // vals = append(vals, merge(targets)...)
                vals = append(updatedDeps(p, t), vals...)
                vals = append(merge(t), vals...)
            }
        }
    }
    if false && enableDirtyMark { p.dirty_mark(vals...) }
}

func (p *execution) interp(ctx Context, name string, args []Value) (res bool) {
    if len(p.interpreted) == 0 && 0 < len(p.recipes) && name == "configure" {
        if x, y := dialects["eval"]; y && x != nil {
            p.interpret(ctx, x, args)
        }
    } else if x, y := dialects[name]; y && x != nil {
        p.interpret(ctx, x, args)
        return
    }
    return true
}

func (p *execution) interpret(ctx Context, i interpreter, args []Value) (res Value) {
    target, _, _ := wait(ctx, waitopts{
        ExecResults: false,
        ReportUpdates: false,
        StampCurrentTarget: false,
    })

    if false && truly(ctx, is_test_univ{}) {
        if x, y := target.(*file); y && strings.HasSuffix(x.name, ".log") {
            defer func() {
                var cc = pc(pc(ctx,auto_get(ctx,">")),x.fullname())
                debug(cc, "%v %v %v", p.language, args, res)
            } ()
        }
    }
    if _, y := target.(*file); y && !truly(ctx, is_configure{}) && !p.dirty(ctx) {
        // p.traves.add(ctx, traveDone, nil) // NOTE: modifier.predictDirty
        return
    }

    res = i.evaluate(ctx, args...)

    if checkpoints {
        p.evaluate_check(ctx, i, args, res)
    }

    if res != nil {
        if d, prev := auto_set(ctx, defVoid, "-", res); d == nil {
            _, ent, _ := entryIndicator(ctx, _entry(ctx))
            prompt(ctx, "%v: %s\n", ent, intername(i))
            debug(ctx, "set buffer value failed: %v → %v", prev, res, trace{})
        }
    }

    p.interpreted = append(p.interpreted, i)

    if _, _, e := p.updateRecipesHash(ctx, target); e != nil {
        _, ent, _ := entryIndicator(ctx, _entry(ctx))
        prompt(ctx, "%v: %s\n", ent, intername(i))
        debug(ctx, "update recipes hash failed: %v", e, trace{})
    }
    return
}

func isDirty(ctx Context, target Value, a ...Value) (dirty bool) {
    var opts, y = do(ctx, propDirtyOpts).(*dirtyOpts)
    if !y {
        debug(ctx, "nil dirtyopts : %v", ts(ctx), trace{})
        return
    }
    if len(updatedDeps(ctx, target)) > 0 { return true }
    if v := auto_get(ctx, "^"); v != nil { a = append(a, v) }
    for _, dep := range xmerge(ctx, a...) {
        var mat bool = len(opts.pats) == 0
        if !mat {
            for _, pat := range opts.pats {
                if mat, _, _, _ = match(ctx, pat, dep); mat { break }
            }
        }
        if mat && (updated(ctx, dep) || statFile(ctx, dep).mod().After(statFile(ctx, target).mod())) {
            return true
        }
    }
    return
}

func isDirtyAfter(ctx Context, target Value, t time.Time) (res bool) {
    for _, dep := range updatedDeps(ctx, target) {
        if ds := statFile(ctx, dep); ds != nil {
            res = ds.mod().After(t) || isDirtyAfter(ctx, dep, t)
            if res { break }
        }
    }
    return
}

func (p *execution) dirty(ctx Context, aa ...Value) (outdated bool) {
    var target Value

    target, _, _ = wait(p, waitopts{
        ReportUpdates: false,
        ExecResults: false,
        StampCurrentTarget: false,
    })

    var y bool
    var reason string
    var targetFile *file
    var targetFull string
    var opts, args = _opts_[dirtyOpts](ctx, aa...)

    if targetFile, targetFull, y = as_fullname_file(ctx, target); !y {
        targetFull = __string(ctx, target)
    } else if n := targetFile._traved; n > 1 {
        if false { debug(ctx, 5, "%v, %v, %d", targetFile, targetFull, n) }
        return
    }

    var verb = opts.debug>0 || opts.verbose
    var ts = trimPromptString(targetFull)

    if s := statFile(ctx, target); s == nil || s.exists() != existenceConfirmed {
        outdated, reason = true, "not exists" //fmt.Sprintf("not exists: %s %v", typeof(target), target)
    } else if isDirty(ctx, target, args...) && isDirtyAfter(ctx, target, s.mod()) {
        outdated, reason = true, "prerequisites updated"
    }

    if outdated {
        assert(reason != "", "needs outdated reason")
    } else if y, e := p.isRecipesChanged(ctx, target); e != nil {
        debug(ctx, "recipes changed: %v", e, trace{})
        return
    } else if y {
        outdated, reason = true, "recipes changed"
    } else if !opts.checksum {
        // does nothing
    } else {
        debug(ctx, "FIXME: check prerequisites against the saved checksums", trace{})
        return
    }

    verb = verb ||
        (opts.verboseOutdated && outdated) ||
        (opts.verboseUpdated && !outdated)

    if verb {
        var d = time.Now().Sub(p.start)
        if false && d > 10*time.Second {
            debug(ctx, "%v: %v", target, d)
        }

        var m string
        if outdated { m = "outdated" } else { m = "updated" }

        var s = d.String()
        if targetFile != nil {
            if targetFile._travin > 1 {
                s += fmt.Sprintf(", travin %d", targetFile._travin)
            }
            if targetFile._traved > 1 {
                s += fmt.Sprintf(", traved %d", targetFile._traved)
            }
            targetFile._dirty += 1
        }

        var n = p.countFiles + len(p.grepped)
        if db := opts.debug>0; db && !opts.verbose {
            debug(ctx, "%s (%T) (%s) …… %s (%d files in %s, debug=%d)", ts, target, targetFull, m, n, s, opts.debug)
        } else {
            prompt(ctx, "%s …… %s (%d files in %s)\n", ts, m, n, s)
			debug(ctx, "%d %d", db, 6)
        }
    }

    if outdated && p.dirt != "" { reason = p.dirt + "; " + reason }
    if !opts.silent && reason != "" { p.dirt = reason }
    return
}

func probPrereqValue(ctx Context, projects []*project, val Value) (prereqValue, prereqPattern Value, prereqFinal string, prereqFile *file) {
	var mapPrereqFile = func(name Value) {
		var ms = unmap_files(ctx, _project(ctx), name, nil)
		if ms != nil {
			defer func() {
				if prereqFile != nil { return }
				for _, m := range ms { warn(ctx, "%v, skipped %v", name, m) }
				debug(ctx, "skipped %d, projects %v", len(ms), projects)
			}()
		}

        if prereqFile = select_file(ctx, ms); prereqFile != nil {
            prereqValue = prereqFile
			prereqFinal = prereqFile.name
		} else if prereqValue != nil {
			if f, y := to_file(prereqValue); y {
				prereqFile = f
				prereqFinal = f.name
			} else {
				prereqFinal = __string(ctx, prereqValue)
				if _, y := prereqValue.(*path); y {
					if f := _stat(ctx, __string(ctx, prereqValue)); f != nil { prereqFile, prereqValue = f, f }
				}
			}
		}
    }

	prereqValue = val
    if _, y := prereqValue.(object); y { return }

    if !patterned(ctx,prereqValue) {
        switch prereqValue.(type) {
        case flag, *strlit, *strcomp: // skip checking files for performance
		default: mapPrereqFile(prereqValue)
        }
		return
    }

    var stems = _stems(ctx)
    if len(stems) == 0 {
        if false { debug(ctx, "%v: no stems, %v", prereqValue, ctx, trace{}) }
        return
    }

	var stemVals []Value
	for _, s := range stems {
		stemVals = append(stemVals, s)
	}

    var rest []Value
    prereqPattern = prereqValue
    prereqValue, rest = stencil(ctx, prereqPattern, stemVals)
    if isTrivial(prereqValue) {
        debug(ctx, "%v: empty stencil with %v", prereqPattern, stems, trace{})
    } else if len(rest) > 0 {
        debug(ctx, "%v: partial stencil with %v, rest=%v", prereqPattern, stems, rest, trace{})
    }

    mapPrereqFile(prereqValue)
    return
}

func (p *execution) traved(ctx Context, targetValue, prereqValue, prereqPattern Value, prereqFile *file) {
    var av = targetValue
    var bv = prereqValue
    if prereqFile == nil {
        do(ctx, act_traversed{prereqValue}) // set $< $> $^ or $|
    } else if targetValue != prereqFile {
        do(ctx, act_traversed{prereqFile}) // set $< $> $^ or $|
        bv = prereqFile
    }

    if !isTrivial(av) && !isTrivial(bv) {
        var a = statFile(ctx, av).mod()
        var b = statFile(ctx, bv).mod()
        if (!a.IsZero() && b.After(a)) || updated(ctx, bv) || updatedDeps(ctx, bv) != nil {
            updatedDeps(ctx, av, bv)
        }
    }
}

func with(ctx Context, target Value) (res bool) {
    if t := auto_find(ctx, "@"); t == nil || t.value == nil {
        return
    } else if res = equal(ctx, target, t.value); res {
        return
    } else {
        return with(inner(ctx), target)
    }
}

func (p *execution) traverse(ctx Context, prereqValue Value) {
    var (
        targetValue Value

        prereqPattern Value
        prereqFinal string
        prereqFile *file

        concreteList []entry
        stemmedList []*stemmed_rule
    )

    if targetValue = auto_target_value(ctx); targetValue == nil {
        debug(ctx, "%s: target is nil\n", prereqFinal, trace{})
    } else if isTrivial(targetValue) {
        debug(ctx, "%s: target is trivial (%T)\n", prereqFinal, targetValue, trace{})
    }

    var projs = []*project{ p.proj }

    if len(projs) == 0 {
        note(ctx, "%v", closure_projects(ctx))
        debug(ctx, "%v: %v → %v: no projects", p.proj, targetValue, prereqValue, trace{})
    }

    prereqValue, prereqPattern, prereqFinal, prereqFile = probPrereqValue(ctx, projs, prereqValue)

    if f := prereqFile; f != nil {
        if f._travin += 1; f._travin > 1 { return }
    }

    // Recursion detection -- simply return to break it if looped.
    if traverseDetectLoops {
        if eq(ctx, targetValue, prereqValue) {
            prompt(ctx, "%v: %v: self dependency, consider using [(once)] to avoid\n", targetValue, prereqValue)
            warn(ctx, "recursion: %T %v", prereqValue, prereqValue)
            warn(ctx, "recursion: %T %v", targetValue, targetValue)
            debug(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projs, trace{})
        }
        for c := p; c != nil; c = c.caller() {
            if val := auto_get(c, "@"); val != nil && eq(c, val, prereqValue) {
                var f = as_file(ctx, targetValue, projs...)
                if true && f == nil {
                    prompt(ctx, "%v: %v: recursion detected, consider using [(once)] to avoid\n", targetValue, prereqValue)
                    warn(ctx, "recursion: %T %v", prereqValue, prereqValue)//.of(prereqValue)
                    warn(ctx, "recursion: %T %v", targetValue, targetValue)//.of(targetValue)
                    debug(ctx, "recursion: %v : %v ; in %v", targetValue, prereqFile, projs)
                }
                return
            }
        }
    }

    if prereqFile != nil {
        if n := prereqFile._traved; n > 0 {
            return
        }
    }

    t1 := time.Now()

    for _, proj := range projs {
        var entries = proj._entries(ctx, prereqValue, false)
        if len(entries) == 0 { continue }
        concreteList = append(concreteList, entries...)
        for _, entry := range entries {
            if !isNull(entry) && targetValue == entry { continue }
            if w, k := targetValue.(*word); k && w.s == prereqFinal {
                continue // target resolve to itself, does nothing
            }
            traverse(ctx, entry)
        }
    }

    if d := time.Now().Sub(t1); 60*time.Second < d {
        for _, concrete := range concreteList {
            prompt(ctx, "%v: slow: %v %v\n", concrete.Pos(), concrete, targetValue)
        }
        debug(ctx, "%v: slow: %v: %v %v (%d concretes)\n", _position(ctx), targetValue, prereqValue, d, len(concreteList))
    }

    if prereqFile != nil && prereqFile.exists() {
        p.traved(ctx, targetValue, prereqValue, prereqPattern, prereqFile)
        return
    }

    t2 := time.Now()

    for _, proj := range projs {
        for _, p := range proj.patterns { assert(patterned(ctx,p.target), "not pattern") }

        var patterns = proj.resolvePatterns(ctx, prereqValue, prereqFinal)
        if len(patterns) == 0 { continue }

        stemmedList = append(stemmedList, patterns...)

        for _, entry := range patterns { traverse(ctx, entry) }
    }

    if d := time.Now().Sub(t2); 60*time.Second < d {
        for _, stemmed  := range stemmedList { prompt(ctx, "%v: slow: %v\n", stemmed.Pos(), stemmed) }
        debug(ctx, "%v: slow: %v: %v %v (%d stemmed)\n", _position(ctx), targetValue, prereqValue, d, len(stemmedList))
    }

    p.traved(ctx, targetValue, prereqValue, prereqPattern, prereqFile)
    return // no operation
}

func (p *execution) prerequisites(va []Value, ordered bool) {
    defer p.Wait()
    p._ordered = ordered
    for _, p.prerequisite = range va { traverse(p, p.prerequisite) }
    p.prerequisite = nil
    return
}

func _program(ctx Context) (p *program) {
    p, _ = do(ctx, get_program{}).(*program)
    return
}

type program struct {
    pos Pos
    project *project
    params   []*auto
    depends  []Value // normal
    ordered  []Value // order-only
    recipes  []Value
    language  string
}

func (prog *program) getModifiers(ctx Context, name string) (ms []*modifier) {
    for _, d := range prog.depends {
        if g, y := d.(*modification); y {
            for _, m := range g.list {
                if __string(ctx, m.elems[0]) == name { ms = append(ms, m) }
            }
        }
    }
    return
}

const maxCallRecursion  = 32 //64

func (prog *program) workdir(ctx Context) (workdir string) {
    var proj = prog.project
    if p := _execution(ctx); p == nil {
        workdir = proj.absPath
    } else if p.changedWD == "" {
        var o object
        if o = proj.resolve(ctx, "CWD"); isTrivial(o) {
            if o = proj.resolve(ctx, "/"); isTrivial(o) {
                debug(ctx, "both $(CWD) and $/ are trivial", trace{})
            }
        }
        if v := expand(_final(ctx),o); v == nil {
            debug(ctx, "trivial %v", ts(o), trace{})
        } else {
            workdir = __string(ctx, v)
        }
    } else if filepath.IsAbs(p.changedWD) {
        workdir = p.changedWD
    } else {
        workdir = filepath.Join(proj.absPath, p.changedWD)
    }
    return
}

func (prog *program) target(ctx *execution) (target Value) {
    if target = _entry(ctx).destiny(); target == nil {
        debug(ctx, "%v: nil entry target", target, trace{})
    }

    switch a := target.(type) {
    case *strlit, *strcomp: // NOTE: skip strings to optimize speed from searching
    case *file: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    case fullfile: if a._traved > 1 { return } // alreadyUpdated = a.info != nil && a.updated
    default:
        if _, y := a.(flag); y {
            if s := prog.project.name; s == "configure" || s == "configure.base" {
                break
            }
        }
        if f := prog.project.file(ctx, a); f != nil {
            if f._traved > 1 { return } else { target = f }
        }
    }

    if ctx.recs == nil { ctx.recs = make(map[Value]int) }
    ctx.recs[target] += 1
    return
}

func (prog *program) check_exe(ctx *execution) {
    var cc = ctx.Context
    var a = []Value{ auto_get(cc, "@") }
    var depth, loop int = 0, -1

callerloop:
    for c := ctx.caller(); c != nil; c = c.caller() {
        if _program(c) == prog {
            if depth += 1; depth == maxCallRecursion { break callerloop }
            var t = auto_get(c, "@")
            for i, v := range a {
                if eq(ctx, t, v) { loop = i; break callerloop }
            }
            if loop < 0 { a = append(a, t) }
        }
    }

    if 0 <= loop {
        var o = cast[*term](cc)
        var v = auto_get(o, "@")
        var t = auto_get(cc, "@")
        if o != nil {
            if v != nil && eq(cc, v, t) {
                if true { debug(ctx, "skip closure loop: %v %v", o, t) }
                // FIXES: skip execution as it's closure, for example:
                //
                //   %.h($(headers)): $(srcinc)/%.h update-file
                //
                // where the 'update-file' is like:
                //
                //   update-file: [((in)) (closure) (set @=&@)] $(in) \
                //       [(read-file $>) (update-file -p)]
                //
                // see also Rule.traverse for the same skip.
                return
            }
        }

        prompt(ctx, "%v: %v: %v, %v\n", a[0], v, cc, o)
        for i, t := range a { erro(ctx, "loop: %v: %v", i, t) }
        debug(ctx, "loop, (depth=%d, %v, %v)\n", depth, a[loop], a, trace{})
    }

    if depth < maxCallRecursion {
        // continues
    } else if c := ctx.caller(); c != nil {
        ctx.traceLevel = c.traceLevel

        var tt = auto_get(c, "@")
        var s, _ = as_fullname_string(ctx, tt)
        prompt(ctx, "%v: max recursion call (%d)\n", s, depth)
        debug(ctx, "max recursion call (%d)\n", depth)

        const collapse = false

        for ; c != nil; c = c.caller() {
            var n int

            if collapse {
                for next := c.caller(); next != nil; next = next.caller() {
                    if d := auto_get(next, "@"); d == nil { continue } else
                    if t := d; t != nil && eq(ctx, t, tt) { n += 1;  continue }
                    if _program(next) == _program(c) { n += 1; c = next } else { break }
                }
            }

            if prog, t := _program(c), auto_get(c, "@"); prog == nil {
                erro(ctx, "%v (@=%v)", _entry(ctx), tt)
                break
            } else if n > 0 {
                erro(ctx, "%v (repeated %d times)", t, n)
            } else if !collapse {
                erro(ctx, "%v : %v", t, auto_get(c, ">"))
            } else if depth -= 1; maxCallRecursion - depth > 5 {
                erro(ctx, "%v ... (%d)", t, maxCallRecursion - depth)
                break
            } else {
                erro(ctx, "%v : %v", t, auto_get(c, ">"))
            }

            flush(ctx) // dump immediately
        }

        debug(ctx, depth, "#>", _entry(ctx), trace{})
    }
}

func (prog *program) result_or_default_interpret(ctx *execution) (res Value) {
    if res = auto_get(ctx, "-"); res != nil {
        return
    }
    if len(ctx.interpreted) == 0 {
        if x, y := dialects[""]; y && x != nil {
            return ctx.interpret(ctx, x, nil)
        }
        debug(ctx, "no default dialect", trace{})
    }
    return
}

func (prog *program) execute(_ctx Context) (res Value) {
    var exe = &execution{
        automatic:automatic{Context:_ctx, defs:make(def_map)},
        recs:make(map[Value]int), start:time.Now(), prog:prog,
        proj:prog.project, recipes:prog.recipes, language:prog.language,
    }

    if checkpoints { defer prog.execute_check(exe, &res) }

    prog.check_exe(exe)

    defer func() {
        for _, a := range exe.defers {
            if x, y := a.(*group); y {
                modify(exe, x, true)
            } else {
                debug(exe, "defer: not a modifier: %s", ts(a), trace{})
            }
        }

        if res == nil { res = auto_get(exe, "-") }
        if res == nil { res = exe.defval }
        if res != nil { do(exe.Context, default_value{res}) }

        exe.defval = nil
    } ()

    for _, param := range prog.params {
        if  exe.params == nil  {
            exe.params = make(map[string]*auto, len(prog.params))
        }
        exe.params[param.name] = param
    }

    if checkpoints { prog.execute_check_0(exe) }

    // NOTE: set "@" before setting auto args
    // Select the right target value before setting parameters,
    // because the target could be overrided by parameters.
    exe.set(exe, defVoid, "@", prog.target(exe))

    if t := _stems(exe); t != nil {
        exe.set(exe, defVoid, "*", ease(exe, t))
    }

    exe.do(exe, init_args{&exe.automatic})
    exe.prerequisites(prog.depends, false)
    exe.prerequisites(prog.ordered, true)

    if checkpoints { prog.execute_check_1(exe) }

    if len(prog.recipes) == 0  { return }
    return prog.result_or_default_interpret(exe)
}


func get_filename(n int) string {
    var num int
    var filename string
    var lines = strings.Split(string(rt_debug.Stack()), "\n")
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
